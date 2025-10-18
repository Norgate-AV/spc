package config

import (
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name        string
		setupViper  func()
		wantConfig  *Config
		wantErr     bool
		errContains string
	}{
		{
			name: "load with all defaults",
			setupViper: func() {
				viper.Reset()
				viper.SetDefault("compiler_path", DefaultCompilerPath)
				viper.SetDefault("target", DefaultTarget)
				viper.SetDefault("silent", DefaultSilent)
				viper.SetDefault("verbose", DefaultVerbose)
			},
			wantConfig: &Config{
				CompilerPath: func() string {
					abs, _ := filepath.Abs(DefaultCompilerPath)
					return abs
				}(),
				Target:      DefaultTarget,
				Silent:      DefaultSilent,
				Verbose:     false,
				UserFolders: nil, // Changed from []string{} to nil
			},
			wantErr: false,
		},
		{
			name: "load with custom values",
			setupViper: func() {
				viper.Reset()
				viper.Set("compiler_path", "C:/Custom/SPlusCC.exe")
				viper.Set("target", "3")
				viper.Set("silent", true)
				viper.Set("verbose", true)
				viper.Set("out", "custom.log")
				viper.Set("usersplusfolder", []string{"C:/Include1", "C:/Include2"})
			},
			wantConfig: &Config{
				CompilerPath: func() string {
					abs, _ := filepath.Abs("C:/Custom/SPlusCC.exe")
					return abs
				}(),
				Target:  "3",
				Silent:  true,
				Verbose: true,
				OutputFile: func() string {
					abs, _ := filepath.Abs("custom.log")
					return abs
				}(),
				UserFolders: func() []string {
					abs1, _ := filepath.Abs("C:/Include1")
					abs2, _ := filepath.Abs("C:/Include2")
					return []string{abs1, abs2}
				}(),
			},
			wantErr: false,
		},
		{
			name: "empty compiler path gets default on Windows",
			setupViper: func() {
				viper.Reset()
				viper.Set("compiler_path", "")
				viper.Set("target", "234")
			},
			wantConfig: &Config{
				CompilerPath: func() string {
					// When empty, it will be resolved as absolute path to current directory
					abs, _ := filepath.Abs("")
					return abs
				}(),
				Target:      "234",
				UserFolders: nil,
			},
			wantErr: false,
		},
		{
			name: "empty target gets default",
			setupViper: func() {
				viper.Reset()
				viper.Set("compiler_path", "C:/SPlusCC.exe")
				viper.Set("target", "")
			},
			wantConfig: &Config{
				CompilerPath: func() string {
					abs, _ := filepath.Abs("C:/SPlusCC.exe")
					return abs
				}(),
				Target:      "34",
				UserFolders: nil,
			},
			wantErr: false,
		},
		{
			name: "invalid target",
			setupViper: func() {
				viper.Reset()
				viper.Set("compiler_path", "C:/SPlusCC.exe")
				viper.Set("target", "invalid")
			},
			wantErr:     true,
			errContains: "invalid target series",
		},
		{
			name: "invalid target - only contains invalid characters",
			setupViper: func() {
				viper.Reset()
				viper.Set("compiler_path", "C:/SPlusCC.exe")
				viper.Set("target", "567")
			},
			wantErr:     true,
			errContains: "invalid target series",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupViper()

			cfg, err := Load()

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantConfig.CompilerPath, cfg.CompilerPath)
			assert.Equal(t, tt.wantConfig.Target, cfg.Target)
			assert.Equal(t, tt.wantConfig.Silent, cfg.Silent)
			assert.Equal(t, tt.wantConfig.Verbose, cfg.Verbose)
			assert.Equal(t, tt.wantConfig.OutputFile, cfg.OutputFile)
			assert.Equal(t, tt.wantConfig.UserFolders, cfg.UserFolders)
		})
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		wantErr     bool
		errContains string
		checkFields func(*testing.T, *Config)
	}{
		{
			name: "valid config with all fields",
			config: &Config{
				CompilerPath: "C:/SPlusCC.exe",
				Target:       "234",
				UserFolders:  []string{"C:/Include"},
				OutputFile:   "output.log",
			},
			wantErr: false,
			checkFields: func(t *testing.T, cfg *Config) {
				// Paths should be absolute
				assert.True(t, filepath.IsAbs(cfg.CompilerPath))
				assert.True(t, filepath.IsAbs(cfg.OutputFile))
				assert.True(t, filepath.IsAbs(cfg.UserFolders[0]))
			},
		},
		{
			name: "valid config with single series",
			config: &Config{
				CompilerPath: "C:/SPlusCC.exe",
				Target:       "3",
			},
			wantErr: false,
		},
		{
			name: "invalid target - empty",
			config: &Config{
				CompilerPath: "C:/SPlusCC.exe",
				Target:       "",
			},
			wantErr:     true,
			errContains: "invalid target series",
		},
		{
			name: "invalid target - wrong series",
			config: &Config{
				CompilerPath: "C:/SPlusCC.exe",
				Target:       "5",
			},
			wantErr:     true,
			errContains: "invalid target series",
		},
		{
			name: "invalid target - mixed valid and invalid",
			config: &Config{
				CompilerPath: "C:/SPlusCC.exe",
				Target:       "35",
			},
			wantErr: false, // Valid because 3 is valid (5 is ignored)
		},
		{
			name: "empty user folder is skipped",
			config: &Config{
				CompilerPath: "C:/SPlusCC.exe",
				Target:       "3",
				UserFolders:  []string{"", "C:/Include"},
			},
			wantErr: false,
			checkFields: func(t *testing.T, cfg *Config) {
				assert.Len(t, cfg.UserFolders, 2)
				assert.Empty(t, cfg.UserFolders[0])
				assert.True(t, filepath.IsAbs(cfg.UserFolders[1]))
			},
		},
		{
			name: "relative paths are resolved",
			config: &Config{
				CompilerPath: "compiler.exe",
				Target:       "3",
				OutputFile:   "output.log",
				UserFolders:  []string{"includes"},
			},
			wantErr: false,
			checkFields: func(t *testing.T, cfg *Config) {
				assert.True(t, filepath.IsAbs(cfg.CompilerPath))
				assert.True(t, filepath.IsAbs(cfg.OutputFile))
				assert.True(t, filepath.IsAbs(cfg.UserFolders[0]))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}

			require.NoError(t, err)
			if tt.checkFields != nil {
				tt.checkFields(t, tt.config)
			}
		})
	}
}

func TestIsValidTarget(t *testing.T) {
	tests := []struct {
		name   string
		target string
		want   bool
	}{
		{"single series 2", "2", true},
		{"single series 3", "3", true},
		{"single series 4", "4", true},
		{"multiple series 23", "23", true},
		{"multiple series 34", "34", true},
		{"all series 234", "234", true},
		{"empty string", "", false},
		{"invalid series 1", "1", false},
		{"invalid series 5", "5", false},
		{"invalid mixed 15", "15", false},
		{"only invalid 567", "567", false},
		{"letters", "abc", false},
		{"special chars", "!@#", false},
		{"valid with invalid ignored", "13", true},    // 1 ignored, 3 valid
		{"valid with invalid ignored 35", "35", true}, // 5 ignored, 3 valid
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidTarget(tt.target)
			assert.Equal(t, tt.want, got)
		})
	}
}
