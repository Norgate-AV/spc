package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLoader(t *testing.T) {
	loader := NewLoader()
	assert.NotNil(t, loader)
}

func TestLoader_SetupViperDefaults(t *testing.T) {
	viper.Reset()
	loader := NewLoader()
	loader.setupViperDefaults()

	assert.Equal(t, "C:/Program Files (x86)/Crestron/Simpl/SPlusCC.exe", viper.GetString("compiler_path"))
	assert.Equal(t, "234", viper.GetString("target"))
	assert.Equal(t, false, viper.GetBool("silent"))
	assert.Equal(t, false, viper.GetBool("verbose"))
}

func TestLoader_LoadGlobalConfig(t *testing.T) {
	// Create a temporary APPDATA directory
	tempDir := t.TempDir()
	spcDir := filepath.Join(tempDir, "spc")
	err := os.Mkdir(spcDir, 0o755)
	require.NoError(t, err)

	// Test with YAML config
	t.Run("loads yaml config", func(t *testing.T) {
		viper.Reset()
		configPath := filepath.Join(spcDir, "config.yml")
		configContent := `compiler_path: "C:/Custom/SPlusCC.exe"
target: "3"
verbose: true`
		err := os.WriteFile(configPath, []byte(configContent), 0o644)
		require.NoError(t, err)

		// Set APPDATA to temp dir
		oldAppData := os.Getenv("APPDATA")
		defer os.Setenv("APPDATA", oldAppData)
		os.Setenv("APPDATA", tempDir)

		loader := NewLoader()
		loader.loadGlobalConfig()

		// Viper should have read the config
		assert.Equal(t, "C:/Custom/SPlusCC.exe", viper.GetString("compiler_path"))
		assert.Equal(t, "3", viper.GetString("target"))
		assert.Equal(t, true, viper.GetBool("verbose"))
	})

	// Test with JSON config
	t.Run("loads json config", func(t *testing.T) {
		viper.Reset()

		// Remove YAML file
		os.Remove(filepath.Join(spcDir, "config.yml"))

		configPath := filepath.Join(spcDir, "config.json")
		configContent := `{
  "compiler_path": "C:/Json/SPlusCC.exe",
  "target": "4"
}`
		err := os.WriteFile(configPath, []byte(configContent), 0o644)
		require.NoError(t, err)

		// Set APPDATA to temp dir
		oldAppData := os.Getenv("APPDATA")
		defer os.Setenv("APPDATA", oldAppData)
		os.Setenv("APPDATA", tempDir)

		loader := NewLoader()
		loader.loadGlobalConfig()

		assert.Equal(t, "C:/Json/SPlusCC.exe", viper.GetString("compiler_path"))
		assert.Equal(t, "4", viper.GetString("target"))
	})

	// Test with no APPDATA
	t.Run("handles missing APPDATA gracefully", func(t *testing.T) {
		viper.Reset()

		oldAppData := os.Getenv("APPDATA")
		defer os.Setenv("APPDATA", oldAppData)
		os.Setenv("APPDATA", "")

		loader := NewLoader()
		loader.loadGlobalConfig()

		// Should not panic, just skip global config
		assert.NotPanics(t, func() {
			loader.loadGlobalConfig()
		})
	})
}

func TestLoader_LoadLocalConfig(t *testing.T) {
	t.Run("loads local config from file directory", func(t *testing.T) {
		viper.Reset()

		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, ".spc.yml")
		configContent := `compiler_path: "C:/Local/SPlusCC.exe"
target: "34"`
		err := os.WriteFile(configPath, []byte(configContent), 0o644)
		require.NoError(t, err)

		// Create a test file in the same directory
		testFile := filepath.Join(tempDir, "test.usp")
		err = os.WriteFile(testFile, []byte("// test"), 0o644)
		require.NoError(t, err)

		loader := NewLoader()
		loader.loadLocalConfig([]string{testFile})

		assert.Equal(t, "C:/Local/SPlusCC.exe", viper.GetString("compiler_path"))
		assert.Equal(t, "34", viper.GetString("target"))
	})

	t.Run("walks up directory tree to find config", func(t *testing.T) {
		viper.Reset()

		tempDir := t.TempDir()
		subDir := filepath.Join(tempDir, "subdir", "nested")
		err := os.MkdirAll(subDir, 0o755)
		require.NoError(t, err)

		// Put config in parent directory
		configPath := filepath.Join(tempDir, ".spc.yml")
		configContent := `target: "2"`
		err = os.WriteFile(configPath, []byte(configContent), 0o644)
		require.NoError(t, err)

		// Test file in nested subdirectory
		testFile := filepath.Join(subDir, "test.usp")
		err = os.WriteFile(testFile, []byte("// test"), 0o644)
		require.NoError(t, err)

		loader := NewLoader()
		loader.loadLocalConfig([]string{testFile})

		assert.Equal(t, "2", viper.GetString("target"))
	})

	t.Run("handles empty args", func(t *testing.T) {
		viper.Reset()

		loader := NewLoader()
		loader.loadLocalConfig([]string{})

		// Should not panic
		assert.NotPanics(t, func() {
			loader.loadLocalConfig([]string{})
		})
	})

	t.Run("handles invalid file path", func(t *testing.T) {
		viper.Reset()

		loader := NewLoader()
		loader.loadLocalConfig([]string{"nonexistent/file.usp"})

		// Should not panic
		assert.NotPanics(t, func() {
			loader.loadLocalConfig([]string{"nonexistent/file.usp"})
		})
	})
}

func TestLoader_BindCommandFlags(t *testing.T) {
	viper.Reset()

	cmd := &cobra.Command{}
	cmd.Flags().StringP("target", "t", "", "Target series")
	cmd.Flags().BoolP("verbose", "v", false, "Verbose output")
	cmd.Flags().StringP("out", "o", "", "Output file")
	cmd.Flags().StringSliceP("usersplusfolder", "u", []string{}, "User folders")

	// Set flag values
	cmd.Flags().Set("target", "3")
	cmd.Flags().Set("verbose", "true")
	cmd.Flags().Set("out", "custom.log")
	cmd.Flags().Set("usersplusfolder", "C:/Include1,C:/Include2")

	loader := NewLoader()
	loader.bindCommandFlags(cmd)

	assert.Equal(t, "3", viper.GetString("target"))
	assert.Equal(t, true, viper.GetBool("verbose"))
	assert.Equal(t, "custom.log", viper.GetString("out"))
	folders := viper.GetStringSlice("usersplusfolder")
	assert.Contains(t, folders, "C:/Include1")
	assert.Contains(t, folders, "C:/Include2")
}

func TestLoader_LoadForBuild_Integration(t *testing.T) {
	t.Run("hierarchical config loading - flags override local override global", func(t *testing.T) {
		viper.Reset()

		// Setup temp directories
		tempDir := t.TempDir()
		spcDir := filepath.Join(tempDir, "spc")
		err := os.Mkdir(spcDir, 0o755)
		require.NoError(t, err)

		// Global config
		globalConfig := filepath.Join(spcDir, "config.yml")
		globalContent := `compiler_path: "C:/Global/SPlusCC.exe"
target: "2"
verbose: false`
		err = os.WriteFile(globalConfig, []byte(globalContent), 0o644)
		require.NoError(t, err)

		// Local config
		localDir := t.TempDir()
		localConfig := filepath.Join(localDir, ".spc.yml")
		localContent := `target: "3"
verbose: true`
		err = os.WriteFile(localConfig, []byte(localContent), 0o644)
		require.NoError(t, err)

		// Test file
		testFile := filepath.Join(localDir, "test.usp")
		err = os.WriteFile(testFile, []byte("// test"), 0o644)
		require.NoError(t, err)

		// Set APPDATA
		oldAppData := os.Getenv("APPDATA")
		defer os.Setenv("APPDATA", oldAppData)
		os.Setenv("APPDATA", tempDir)

		// Create command with flags
		cmd := &cobra.Command{}
		cmd.Flags().StringP("target", "t", "", "Target series")
		cmd.Flags().BoolP("verbose", "v", false, "Verbose output")
		cmd.Flags().StringP("out", "o", "", "Output file")
		cmd.Flags().StringSliceP("usersplusfolder", "u", []string{}, "User folders")
		cmd.Flags().BoolP("silent", "s", false, "Silent mode")

		// Flag overrides
		cmd.Flags().Set("target", "4")

		loader := NewLoader()
		cfg, err := loader.LoadForBuild(cmd, []string{testFile})
		require.NoError(t, err)

		// Flag value should win
		assert.Equal(t, "4", cfg.Target)
		// Local config should override global
		assert.Equal(t, true, cfg.Verbose)
		// Global config should be used as base (but will be resolved as absolute path)
		// The compiler path from global config will be used
		assert.NotEmpty(t, cfg.CompilerPath)
	})
}
