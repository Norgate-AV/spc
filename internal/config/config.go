package config

import (
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/Norgate-AV/spc/internal/utils"
	"github.com/spf13/viper"
)

// Default configuration values
const (
	DefaultCompilerPath = "C:/Program Files (x86)/Crestron/Simpl/SPlusCC.exe"
	DefaultTarget       = "34"
	DefaultSilent       = false
	DefaultVerbose      = false
)

// Holds the configuration options for spc
type Config struct {
	// Path to the Crestron SIMPL+ compiler
	CompilerPath string

	// Compilation target series (e.g., 2, 23, 234)
	Target string
	// Parsed target series
	Series []string

	// User SIMPL+ folders
	UserFolders []string

	// Output file for compilation log
	OutputFile string

	// Suppress console output from the SIMPL+ compiler
	Silent bool

	// Enable verbose output
	Verbose bool
}

func Load() (*Config, error) {
	cfg := &Config{
		CompilerPath: viper.GetString("compiler_path"),
		Target:       viper.GetString("target"),
		UserFolders:  viper.GetStringSlice("usersplusfolder"),
		OutputFile:   viper.GetString("out"),
		Silent:       viper.GetBool("silent"),
		Verbose:      viper.GetBool("verbose"),
	}

	// Apply defaults if not set
	if cfg.CompilerPath == "" {
		if runtime.GOOS != "windows" {
			cfg.CompilerPath = DefaultCompilerPath
		}
	}

	if cfg.Target == "" {
		cfg.Target = DefaultTarget
	}

	// Validate required fields
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) Validate() error {
	if abs, err := filepath.Abs(c.CompilerPath); err == nil {
		c.CompilerPath = abs
	}

	// Resolve output file path
	if c.OutputFile != "" {
		abs, err := filepath.Abs(c.OutputFile)
		if err != nil {
			return fmt.Errorf("invalid output file path: %v", err)
		}

		c.OutputFile = abs
	}

	// Validate target
	if !isValidTarget(c.Target) {
		return fmt.Errorf("invalid target series: %s", c.Target)
	}

	// Resolve user folders
	for i, folder := range c.UserFolders {
		if folder != "" {
			abs, err := filepath.Abs(folder)
			if err != nil {
				return fmt.Errorf("invalid user folder path: %v", err)
			}

			c.UserFolders[i] = abs
		}
	}

	return nil
}

func isValidTarget(target string) bool {
	series := utils.ParseTarget(target)
	return len(series) > 0
}
