package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/Norgate-AV/spc/internal/compiler"
	"github.com/Norgate-AV/spc/internal/config"
)

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build SIMPL+ file(s)",
	Long:  `Build a SIMPL+ file(s) for the specified target series.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runBuild(cmd, args)
	},
	SilenceUsage: true,
}

func runBuild(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("no files specified")
	}

	cfg, err := loadBuildConfig(cmd, args)
	if err != nil {
		return err
	}

	cmdArgs, err := buildCommandArgs(cfg, args)
	if err != nil {
		return err
	}

	series := parseTarget(cfg.Target)

	if cfg.Verbose {
		fmt.Printf("Compiler: %s\nTarget: %s\nSeries: %v\nFiles: %v\nOut: %s\nUsersPlusFolders: %v\nCommand: %s %s\n", cfg.CompilerPath, cfg.Target, series, args, cfg.OutputFile, cfg.UserFolders, cfg.CompilerPath, strings.Join(cmdArgs, " "))
	}

	c := execCommand(cfg.CompilerPath, cmdArgs...)
	if cmd, ok := c.(*exec.Cmd); ok {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	err = c.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			code := exitErr.ExitCode()
			if compiler.IsSuccess(code) {
				// Crestron compiler success (may have warnings)
				return nil
			}

			// Print descriptive error message
			fmt.Fprintf(os.Stderr, "Compilation failed (exit code %d): %s\n", code, codes.GetErrorMessage(code))
		}

		return err
	}

	return nil
}

func buildCommandArgs(cfg *config.Config, files []string) ([]string, error) {
	series := parseTarget(cfg.Target)
	if len(series) == 0 {
		return nil, fmt.Errorf("invalid target series")
	}

	var cmdArgs []string
	cmdArgs = append(cmdArgs, "/target")
	cmdArgs = append(cmdArgs, series...)

	for _, folder := range cfg.UserFolders {
		if folder != "" {
			cmdArgs = append(cmdArgs, "/usersplusfolder", folder)
		}
	}

	cmdArgs = append(cmdArgs, "/rebuild")

	for _, file := range files {
		absFile, err := filepath.Abs(file)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve absolute path for %s: %w", file, err)
		}

		cmdArgs = append(cmdArgs, absFile)
	}

	if cfg.OutputFile != "" {
		cmdArgs = append(cmdArgs, "/out", cfg.OutputFile)
	}

	if cfg.Silent {
		cmdArgs = append(cmdArgs, "/silent")
	}

	return cmdArgs, nil
}

func parseTarget(t string) []string {
	series := make([]string, 0)

	for _, r := range t {
		if s := int(r - '0'); s >= 2 && s <= 4 {
			series = append(series, "series"+strconv.Itoa(s))
		}
	}

	return series
}

func findLocalConfig(dir string) string {
	for {
		for _, ext := range []string{"yml", "yaml", "json", "toml"} {
			path := filepath.Join(dir, ".spc."+ext)

			if _, err := os.Stat(path); err == nil {
				return path
			}
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}

		dir = parent
	}

	return ""
}

func loadBuildConfig(cmd *cobra.Command, args []string) (*config.Config, error) {
	// Set defaults
	viper.SetDefault("compiler_path", "C:/Program Files (x86)/Crestron/Simpl/SPlusCC.exe")
	viper.SetDefault("target", "234")
	viper.SetDefault("silent", false)
	viper.SetDefault("verbose", false)

	// global config
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

	// local config
	if len(args) > 0 {
		absFirstFile, err := filepath.Abs(args[0])
		if err != nil {
			return nil, fmt.Errorf("failed to resolve absolute path for first file: %w", err)
		}

		dir := filepath.Dir(absFirstFile)
		localPath := findLocalConfig(dir)
		if localPath != "" {
			viper.SetConfigFile(localPath)
			_ = viper.ReadInConfig()
		}
	}

	// bind flags
	_ = viper.BindPFlag("target", cmd.Flags().Lookup("target"))
	_ = viper.BindPFlag("verbose", cmd.Flags().Lookup("verbose"))
	_ = viper.BindPFlag("out", cmd.Flags().Lookup("out"))
	_ = viper.BindPFlag("usersplusfolder", cmd.Flags().Lookup("usersplusfolder"))

	return config.Load()
}

var execCommand = func(name string, args ...string) Commander {
	return exec.Command(name, args...)
}

type Commander interface {
	Run() error
}
