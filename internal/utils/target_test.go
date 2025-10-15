package utils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseTarget(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"2", []string{"series2"}},
		{"3", []string{"series3"}},
		{"4", []string{"series4"}},
		{"23", []string{"series2", "series3"}},
		{"34", []string{"series3", "series4"}},
		{"234", []string{"series2", "series3", "series4"}},
		{"", []string{}},
		{"5", []string{}},
		{"13", []string{"series3"}},
	}

	for _, test := range tests {
		result := ParseTarget(test.input)
		assert.Equal(t, test.expected, result, "ParseTarget(%q)", test.input)
	}
}

func TestFindLocalConfig(t *testing.T) {
	// Create a temporary directory structure
	tempDir := t.TempDir()
	subDir := filepath.Join(tempDir, "subdir")
	err := os.Mkdir(subDir, 0o755)
	assert.NoError(t, err)

	// Create config files
	configYML := filepath.Join(subDir, ".spc.yml")
	err = os.WriteFile(configYML, []byte("target: \"3\""), 0o644)
	assert.NoError(t, err)

	// Test finding in subdir
	result := FindLocalConfig(subDir)
	assert.Equal(t, configYML, result)

	// Test finding in parent
	result = FindLocalConfig(filepath.Join(subDir, "deep"))
	assert.Equal(t, configYML, result)

	// Test not found
	result = FindLocalConfig(tempDir)
	assert.Equal(t, "", result)
}
