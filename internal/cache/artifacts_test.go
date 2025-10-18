package cache

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsOutputFileForTarget_SpacesInFilename(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		baseName string
		target   string
		want     bool
	}{
		// Test space handling for Series 3/4 files
		{
			name:     "cs file with underscores matches space baseName",
			filename: "example_3.cs",
			baseName: "example 3",
			target:   "34",
			want:     true,
		},
		{
			name:     "dll file with underscores matches space baseName",
			filename: "example_3.dll",
			baseName: "example 3",
			target:   "34",
			want:     true,
		},
		{
			name:     "inf file with spaces matches space baseName",
			filename: "example 3.inf",
			baseName: "example 3",
			target:   "34",
			want:     true,
		},
		{
			name:     "ush file with spaces matches space baseName",
			filename: "example 3.ush",
			baseName: "example 3",
			target:   "34",
			want:     true,
		},
		// Test space handling for Series 2 files
		{
			name:     "S2 c file with underscores matches space baseName for target 2",
			filename: "S2_example_3.c",
			baseName: "example 3",
			target:   "2",
			want:     true,
		},
		{
			name:     "S2 h file with underscores matches space baseName for target 234",
			filename: "S2_example_3.h",
			baseName: "example 3",
			target:   "234",
			want:     true,
		},
		{
			name:     "S2 file should not match target 34",
			filename: "S2_example_3.c",
			baseName: "example 3",
			target:   "34",
			want:     false,
		},
		// Test that we don't match wrong files
		{
			name:     "different file should not match",
			filename: "other_file.cs",
			baseName: "example 3",
			target:   "34",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isOutputFileForTarget(tt.filename, tt.baseName, tt.target)
			if got != tt.want {
				t.Errorf("isOutputFileForTarget(%q, %q, %q) = %v, want %v",
					tt.filename, tt.baseName, tt.target, got, tt.want)
			}
		})
	}
}

func TestCollectOutputs_SpacesInFilename(t *testing.T) {
	// Create a temporary directory structure
	tmpDir := t.TempDir()

	// Create a source file with spaces
	sourceFile := filepath.Join(tmpDir, "example 3.usp")
	if err := os.WriteFile(sourceFile, []byte("test"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create .ush file (keeps spaces)
	if err := os.WriteFile(filepath.Join(tmpDir, "example 3.ush"), []byte("test"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create SPlsWork directory
	splsWorkDir := filepath.Join(tmpDir, "SPlsWork")
	if err := os.MkdirAll(splsWorkDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create files with different space handling
	testFiles := map[string]string{
		"example 3.inf":  "test", // Keeps spaces
		"example_3.cs":   "test", // Converts to underscores
		"example_3.dll":  "test", // Converts to underscores
		"S2_example_3.c": "test", // S2 files use underscores
		"S2_example_3.h": "test", // S2 files use underscores
		// Shared files (should not be collected)
		"SplusLibrary.dll": "test",
		"Version.ini":      "test",
	}

	for filename, content := range testFiles {
		path := filepath.Join(splsWorkDir, filename)
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	tests := []struct {
		name            string
		target          string
		expectedFiles   []string
		unexpectedFiles []string
	}{
		{
			name:   "target 34 should collect Series 3/4 files only",
			target: "34",
			expectedFiles: []string{
				"example 3.ush",
				"SPlsWork/example 3.inf",
				"SPlsWork/example_3.cs",
				"SPlsWork/example_3.dll",
			},
			unexpectedFiles: []string{
				"SPlsWork/S2_example_3.c",
				"SPlsWork/S2_example_3.h",
				"SPlsWork/SplusLibrary.dll",
				"SPlsWork/Version.ini",
			},
		},
		{
			name:   "target 234 should collect all files",
			target: "234",
			expectedFiles: []string{
				"example 3.ush",
				"SPlsWork/example 3.inf",
				"SPlsWork/example_3.cs",
				"SPlsWork/example_3.dll",
				"SPlsWork/S2_example_3.c",
				"SPlsWork/S2_example_3.h",
			},
			unexpectedFiles: []string{
				"SPlsWork/SplusLibrary.dll",
				"SPlsWork/Version.ini",
			},
		},
		{
			name:   "target 2 should collect Series 2 files only",
			target: "2",
			expectedFiles: []string{
				"example 3.ush",
				"SPlsWork/S2_example_3.c",
				"SPlsWork/S2_example_3.h",
			},
			unexpectedFiles: []string{
				"SPlsWork/example 3.inf",
				"SPlsWork/example_3.cs",
				"SPlsWork/example_3.dll",
				"SPlsWork/SplusLibrary.dll",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outputs, err := CollectOutputs(sourceFile, tt.target)
			if err != nil {
				t.Fatalf("CollectOutputs() error = %v", err)
			}

			// Check expected files are present
			for _, expected := range tt.expectedFiles {
				found := false
				for _, output := range outputs {
					// Normalize path separators for cross-platform comparison
					normalizedOutput := filepath.ToSlash(output)
					if normalizedOutput == expected {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected file %q not found in outputs: %v", expected, outputs)
				}
			}

			// Check unexpected files are not present
			for _, unexpected := range tt.unexpectedFiles {
				for _, output := range outputs {
					// Normalize path separators for cross-platform comparison
					normalizedOutput := filepath.ToSlash(output)
					if normalizedOutput == unexpected {
						t.Errorf("Unexpected file %q found in outputs", unexpected)
					}
				}
			}
		})
	}
}
