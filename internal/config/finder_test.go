package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

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
