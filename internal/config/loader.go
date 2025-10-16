package config

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Loader handles configuration loading from various sources
type Loader struct{}

// NewLoader creates a new configuration loader
func NewLoader() *Loader {
	return &Loader{}
}

// LoadForBuild loads configuration specifically for build operations
func (l *Loader) LoadForBuild(cmd *cobra.Command, args []string) (*Config, error) {
	l.setupViperDefaults()
	l.loadGlobalConfig()
	l.loadLocalConfig(args)
	l.bindCommandFlags(cmd)

	return Load()
}

// setupViperDefaults sets up default values for viper
func (l *Loader) setupViperDefaults() {
	viper.SetDefault("compiler_path", DefaultCompilerPath)
	viper.SetDefault("target", DefaultTarget)
	viper.SetDefault("silent", DefaultSilent)
	viper.SetDefault("verbose", DefaultVerbose)
}

// loadGlobalConfig loads global configuration from APPDATA
func (l *Loader) loadGlobalConfig() {
	appdata := os.Getenv("APPDATA")
	if appdata != "" {
		globalDir := filepath.Join(appdata, "spc")

		for _, ext := range []string{"yml", "yaml", "json", "toml"} {
			globalPath := filepath.Join(globalDir, "config."+ext)

			if _, err := os.Stat(globalPath); err == nil {
				viper.SetConfigFile(globalPath)

				if err := viper.ReadInConfig(); err == nil {
					break
				}
			}
		}
	}
}

// loadLocalConfig loads local configuration from project directory
func (l *Loader) loadLocalConfig(args []string) {
	if len(args) > 0 {
		absFirstFile, err := filepath.Abs(args[0])
		if err != nil {
			return // silently ignore, config.Load() will handle validation
		}

		dir := filepath.Dir(absFirstFile)
		localPath := FindLocalConfig(dir)
		if localPath != "" {
			viper.SetConfigFile(localPath)
			_ = viper.ReadInConfig()
		}
	}
}

// bindCommandFlags binds command flags to viper
func (l *Loader) bindCommandFlags(cmd *cobra.Command) {
	_ = viper.BindPFlag("target", cmd.Flags().Lookup("target"))
	_ = viper.BindPFlag("verbose", cmd.Flags().Lookup("verbose"))
	_ = viper.BindPFlag("out", cmd.Flags().Lookup("out"))
	_ = viper.BindPFlag("usersplusfolder", cmd.Flags().Lookup("usersplusfolder"))
}
