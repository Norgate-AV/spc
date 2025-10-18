// Package cache provides artifact management for SIMPL+ build caching.
//
// The Crestron SIMPL+ compiler (SPlusCC.exe) generates output files in two locations:
//  1. A shared "SPlsWork" directory adjacent to source files (for most artifacts)
//  2. A .ush header file placed adjacent to the source file itself
//
// Multiple SIMPL+ source files in the same directory all output to the same SPlsWork
// folder, making it critical to filter artifacts by source file name when caching.
//
// For a source file named "example.usp", the compiler generates:
//   - Adjacent: example.ush (header file in same directory as source)
//   - SPlsWork: example.dll, example.cs, example.inf
//   - SPlsWork: S2_example.c, S2_example.h, S2_example.elf, etc.
//
// This package handles selective copying/restoration of only the artifacts
// belonging to a specific source file, ignoring shared libraries and other
// source files' artifacts.
package cache

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// CopyArtifacts copies compiled outputs from a base directory to cache
// The outputs paths are relative to baseDir (e.g., "SPlsWork/example.dll", "example.ush")
func CopyArtifacts(baseDir, destDir string, outputs []string) error {
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return fmt.Errorf("failed to create artifact directory: %w", err)
	}

	for _, output := range outputs {
		src := filepath.Join(baseDir, output)
		dst := filepath.Join(destDir, output)

		// Only copy if file doesn't exist or differs (optimization for re-caching)
		if _, err := copyFileIfNeeded(src, dst); err != nil {
			return fmt.Errorf("failed to copy %s: %w", output, err)
		}
	}

	return nil
}

// RestoreArtifacts copies cached outputs back to the base directory
// The outputs paths are relative to destDir (e.g., "SPlsWork/example.dll", "example.ush")
func RestoreArtifacts(cacheDir, destDir string, outputs []string) error {
	for _, output := range outputs {
		src := filepath.Join(cacheDir, output)
		dst := filepath.Join(destDir, output)

		// Create parent directory if needed (e.g., for SPlsWork/...)
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}

		// Only copy if file doesn't exist or differs
		if _, err := copyFileIfNeeded(src, dst); err != nil {
			return fmt.Errorf("failed to restore %s: %w", output, err)
		}
	}

	return nil
}

// CollectOutputs scans for compiled output files specific to the given source file.
// It checks two locations:
//  1. The source file directory for .ush header files
//  2. The SPlsWork directory for source-specific artifacts
//
// Only collects files for the specified target (e.g., if target="34", skips S2_* files)
// Returns paths relative to the source directory (e.g., "example.ush", "SPlsWork/example.dll")
func CollectOutputs(sourceFile string, target string) ([]string, error) {
	var outputs []string

	// Extract base name without extension (e.g., "example1" from "example1.usp")
	baseName := filepath.Base(sourceFile)
	baseName = baseName[:len(baseName)-len(filepath.Ext(baseName))]

	sourceDir := filepath.Dir(sourceFile)
	splsWorkDir := filepath.Join(sourceDir, "SPlsWork")

	// Check for .ush file adjacent to source
	ushFile := baseName + ".ush"
	ushPath := filepath.Join(sourceDir, ushFile)
	if _, err := os.Stat(ushPath); err == nil {
		outputs = append(outputs, ushFile)
	}

	// Scan SPlsWork directory
	entries, err := os.ReadDir(splsWorkDir)
	if err != nil {
		if os.IsNotExist(err) {
			return outputs, nil // No SPlsWork directory yet
		}
		return nil, fmt.Errorf("failed to read SPlsWork directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()

		// Skip metadata files
		if name == "metadata.json" {
			continue
		}

		// Check if this file belongs to our source file AND target
		// Match patterns: {basename}.* or S2_{basename}.* (depending on target)
		if isOutputFileForTarget(name, baseName, target) {
			// Store with SPlsWork/ prefix for proper path handling
			outputs = append(outputs, filepath.Join("SPlsWork", name))
		}
	}

	return outputs, nil
}

// CollectSharedFiles scans the SPlsWork directory for shared library files
// that are not specific to any source file (DLLs, config files, etc.)
// Returns paths relative to the source directory (e.g., "SPlsWork/Version.ini")
func CollectSharedFiles(sourceDir string) ([]string, error) {
	var sharedFiles []string

	splsWorkDir := filepath.Join(sourceDir, "SPlsWork")

	entries, err := os.ReadDir(splsWorkDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No SPlsWork directory
		}
		return nil, fmt.Errorf("failed to read SPlsWork directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()

		// Check if this is a shared file (not matching any source pattern)
		// Shared files: *.dll, *.dat, *.xml, *.ini (except source-specific ones)
		if isSharedFile(name) {
			sharedFiles = append(sharedFiles, filepath.Join("SPlsWork", name))
		}
	}

	return sharedFiles, nil
}

// isSharedFile checks if a file is a shared library/config file
func isSharedFile(filename string) bool {
	// Common shared file patterns in SPlsWork
	ext := filepath.Ext(filename)
	baseName := filename[:len(filename)-len(ext)]

	// DLL files that don't match source patterns
	if ext == ".dll" {
		// Check if it's NOT a source-specific DLL (which would be in format "sourcename.dll")
		// Shared DLLs have names like "ManagedUtilities.dll", "SplusLibrary.dll"
		// If it contains certain keywords, it's shared
		sharedKeywords := []string{"Managed", "Simpl", "Sharp", "Splus", "Smart", "Utilities", "Newtonsoft", "Json"}
		for _, keyword := range sharedKeywords {
			if containsIgnoreCase(baseName, keyword) {
				return true
			}
		}
	}

	// Config/data files are always shared
	if ext == ".ini" || ext == ".xml" || ext == ".dat" || ext == ".der" {
		return true
	}

	return false
}

// containsIgnoreCase checks if a string contains a substring (case-insensitive)
func containsIgnoreCase(s, substr string) bool {
	s = filepath.Base(s) // normalize
	return len(s) >= len(substr) &&
		(s == substr ||
			len(s) > len(substr) &&
				(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
					findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// isOutputFile checks if a filename belongs to the given source base name
func isOutputFile(filename, baseName string) bool {
	fileBase := filename[:len(filename)-len(filepath.Ext(filename))]

	// Direct match: example1.dll, example1.cs, etc.
	if fileBase == baseName {
		return true
	}

	// Target-prefixed match: S2_example1.c, S2_example1.h, etc.
	if len(fileBase) > 3 && fileBase[0] == 'S' && fileBase[2] == '_' {
		// Extract after "S2_" prefix
		if fileBase[3:] == baseName {
			return true
		}
	}

	return false
}

// isOutputFileForTarget checks if a file belongs to the given source AND target
// For target "34", only matches example1.* (not S2_example1.*)
// For target "234", matches both example1.* and S2_example1.*
func isOutputFileForTarget(filename, baseName, target string) bool {
	fileBase := filename[:len(filename)-len(filepath.Ext(filename))]

	// Direct match: example1.dll, example1.cs, etc.
	// These are for Series 3 and 4
	if fileBase == baseName {
		return true
	}

	// Target-prefixed match: S2_example1.c, S2_example1.h, S3_example1.*, S4_example1.*
	if len(fileBase) > 3 && fileBase[0] == 'S' && fileBase[2] == '_' {
		// Extract the series number
		seriesChar := fileBase[1]

		// Extract the base name after prefix
		if fileBase[3:] == baseName {
			// Check if this series is in the target
			// For example, if target="34", we want Series 3 and 4, not Series 2
			switch seriesChar {
			case '2':
				return contains(target, '2')
			case '3':
				return contains(target, '3')
			case '4':
				return contains(target, '4')
			}
		}
	}

	return false
}

// contains checks if a string contains a specific character
func contains(s string, ch byte) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == ch {
			return true
		}
	}
	return false
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}

	defer srcFile.Close()

	// Create parent directory if needed
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}

	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return err
	}

	// Preserve file permissions
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	return os.Chmod(dst, srcInfo.Mode())
}

// filesAreIdentical checks if two files have the same content
// Uses a fast size check first, then hash comparison if needed
func filesAreIdentical(file1, file2 string) bool {
	// Get file info for both files
	info1, err1 := os.Stat(file1)
	info2, err2 := os.Stat(file2)

	// If either file doesn't exist or we can't stat it, they're not identical
	if err1 != nil || err2 != nil {
		return false
	}

	// Quick check: if sizes differ, files are different
	if info1.Size() != info2.Size() {
		return false
	}

	// If size is 0, both empty files are identical
	if info1.Size() == 0 {
		return true
	}

	// For small files (< 64KB), compare content directly
	if info1.Size() < 65536 {
		content1, err1 := os.ReadFile(file1)
		content2, err2 := os.ReadFile(file2)
		if err1 != nil || err2 != nil {
			return false
		}
		return bytes.Equal(content1, content2)
	}

	// For larger files, use hash comparison
	hash1, err1 := hashFile(file1)
	hash2, err2 := hashFile(file2)
	if err1 != nil || err2 != nil {
		return false
	}

	return bytes.Equal(hash1, hash2)
}

// hashFile computes SHA256 hash of a file
func hashFile(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return nil, err
	}

	return hash.Sum(nil), nil
}

// copyFileIfNeeded copies a file only if destination doesn't exist or differs from source
// Returns true if file was copied, false if copy was skipped
func copyFileIfNeeded(src, dst string) (bool, error) {
	// Check if files are already identical
	if filesAreIdentical(src, dst) {
		return false, nil // Skip copy
	}

	// Files differ or destination doesn't exist, perform copy
	if err := copyFile(src, dst); err != nil {
		return false, err
	}

	return true, nil
}
