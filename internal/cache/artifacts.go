package cache

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// CopyArtifacts copies compiled outputs from source to cache
func CopyArtifacts(sourceDir, destDir string, outputs []string) error {
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return fmt.Errorf("failed to create artifact directory: %w", err)
	}

	for _, output := range outputs {
		src := filepath.Join(sourceDir, output)
		dst := filepath.Join(destDir, output)

		if err := copyFile(src, dst); err != nil {
			return fmt.Errorf("failed to copy %s: %w", output, err)
		}
	}

	return nil
}

// RestoreArtifacts copies cached outputs back to the working directory
func RestoreArtifacts(cacheDir, destDir string, outputs []string) error {
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	for _, output := range outputs {
		src := filepath.Join(cacheDir, output)
		dst := filepath.Join(destDir, output)

		if err := copyFile(src, dst); err != nil {
			return fmt.Errorf("failed to restore %s: %w", output, err)
		}
	}

	return nil
}

// CollectOutputs scans a directory and returns a list of compiled output files
func CollectOutputs(dir string) ([]string, error) {
	var outputs []string

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No outputs yet
		}
		return nil, fmt.Errorf("failed to read output directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Only collect actual output files, skip metadata
		name := entry.Name()
		if name != "metadata.json" {
			outputs = append(outputs, name)
		}
	}

	return outputs, nil
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
