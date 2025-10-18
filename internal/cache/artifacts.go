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

		if err := copyFile(src, dst); err != nil {
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

		if err := copyFile(src, dst); err != nil {
			return fmt.Errorf("failed to restore %s: %w", output, err)
		}
	}

	return nil
}

// CollectOutputs scans for compiled output files specific to the given source file.
// It checks two locations:
//  1. The source file directory for .ush header files
//  2. The SPlsWork directory for other artifacts
//
// Returns paths relative to the source directory (e.g., "example.ush", "SPlsWork/example.dll")
func CollectOutputs(sourceFile string) ([]string, error) {
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

		// Check if this file belongs to our source file
		// Match patterns: {basename}.* or S2_{basename}.*
		if isOutputFile(name, baseName) {
			// Store with SPlsWork/ prefix for proper path handling
			outputs = append(outputs, filepath.Join("SPlsWork", name))
		}
	}

	return outputs, nil
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
