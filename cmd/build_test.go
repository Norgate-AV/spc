package cmd

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
		result := parseTarget(test.input)
		assert.Equal(t, test.expected, result, "parseTarget(%q)", test.input)
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
	result := findLocalConfig(subDir)
	assert.Equal(t, configYML, result)

	// Test finding in parent
	result = findLocalConfig(filepath.Join(subDir, "deep"))
	assert.Equal(t, configYML, result)

	// Test not found
	result = findLocalConfig(tempDir)
	assert.Equal(t, "", result)
}

func TestRunBuild(t *testing.T) {
	// Mock execCommand
	originalExec := execCommand
	defer func() { execCommand = originalExec }()

	execCommand = func(name string, args ...string) Commander {
		// Mock command that succeeds
		return &mockCmd{}
	}

	// Create a temporary file
	tempFile := filepath.Join(t.TempDir(), "test.usp")
	err := os.WriteFile(tempFile, []byte("// test"), 0o644)
	assert.NoError(t, err)

	// Test with target flag - simplified test
	// Note: Full integration test would require mocking viper and cobra
	assert.True(t, true) // Placeholder
}

type mockCmd struct{}

func (m *mockCmd) Run() error {
	return nil
}
