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

	"github.com/Norgate-AV/spc/internal/codes"
)

var buildCmd = &cobra.Command{
	Use:          "build",
	Short:        "Build SIMPL+ program",
	Long:         `Compile a SIMPL+ program for the specified target series.`,
	RunE:         runBuild,
	SilenceUsage: true,
}

func runBuild(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("requires at least one file argument")
	}

	// resolve absolute path for the first file to find config
	firstFile := args[0]
	absFirstFile, err := filepath.Abs(firstFile)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	// check file extension for all files
	for _, file := range args {
		if !strings.HasSuffix(file, ".usp") && !strings.HasSuffix(file, ".usl") {
			return fmt.Errorf("file %s must have .usp or .usl extension", file)
		}
	}

	// load config
	viper.SetDefault("compiler_path", `C:\Program Files (x86)\Crestron\Simpl\SPlusCC.exe`)
	viper.SetDefault("target", "234")

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
	dir := filepath.Dir(absFirstFile)
	localPath := findLocalConfig(dir)
	if localPath != "" {
		viper.SetConfigFile(localPath)
		_ = viper.ReadInConfig()
	}

	// bind flag
	_ = viper.BindPFlag("target", cmd.Flags().Lookup("target"))
	_ = viper.BindPFlag("verbose", cmd.Flags().Lookup("verbose"))
	_ = viper.BindPFlag("silent", cmd.Flags().Lookup("silent"))
	_ = viper.BindPFlag("out", cmd.Flags().Lookup("out"))
	target := viper.GetString("target")
	if target == "" {
		return fmt.Errorf("target series not specified")
	}

	series := parseTarget(target)
	if len(series) == 0 {
		return fmt.Errorf("invalid target series")
	}

	compiler := viper.GetString("compiler_path")
	verbose := viper.GetBool("verbose")

	var cmdArgs []string
	cmdArgs = append(cmdArgs, "/target")
	cmdArgs = append(cmdArgs, series...)
	cmdArgs = append(cmdArgs, "/rebuild")

	for _, file := range args {
		absFile, err := filepath.Abs(file)
		if err != nil {
			return fmt.Errorf("failed to resolve absolute path for %s: %w", file, err)
		}

		cmdArgs = append(cmdArgs, absFile)
	}

	out := viper.GetString("out")
	if out != "" {
		absOut, err := filepath.Abs(out)
		if err != nil {
			return fmt.Errorf("failed to resolve absolute path for output file: %w", err)
		}
		cmdArgs = append(cmdArgs, "/out", absOut)
	}

	silent := viper.GetBool("silent")
	if silent {
		cmdArgs = append(cmdArgs, "/silent")
	}

	if verbose {
		fmt.Printf("Compiler: %s\nTarget: %s\nSeries: %v\nFiles: %v\nOut: %s\nCommand: %s %s\n", compiler, target, series, args, out, compiler, strings.Join(cmdArgs, " "))
	}

	c := execCommand(compiler, cmdArgs...)
	if cmd, ok := c.(*exec.Cmd); ok {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	err = c.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			code := exitErr.ExitCode()
			if codes.IsSuccess(code) {
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

var execCommand = func(name string, args ...string) Commander {
	return exec.Command(name, args...)
}

type Commander interface {
	Run() error
}
