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
)

var buildCmd = &cobra.Command{
	Use:          "build",
	Short:        "Build SIMPL+ program",
	Long:         `Compile a SIMPL+ program for the specified target series.`,
	RunE:         runBuild,
	SilenceUsage: true,
}

func runBuild(cmd *cobra.Command, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("requires exactly one file argument")
	}

	file := args[0]
	// check file extension
	if !strings.HasSuffix(file, ".usp") && !strings.HasSuffix(file, ".usl") {
		return fmt.Errorf("file must have .usp or .usl extension")
	}

	// resolve to absolute path
	absFile, err := filepath.Abs(file)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path: %w", err)
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
	dir := filepath.Dir(absFile)
	localPath := findLocalConfig(dir)
	if localPath != "" {
		viper.SetConfigFile(localPath)
		viper.ReadInConfig()
	}

	// bind flag
	viper.BindPFlag("target", cmd.Flags().Lookup("target"))
	viper.BindPFlag("verbose", cmd.Flags().Lookup("verbose"))
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

	for _, s := range series {
		cmdArgs = append(cmdArgs, s)
	}

	cmdArgs = append(cmdArgs, "/rebuild")
	cmdArgs = append(cmdArgs, absFile)

	if verbose {
		fmt.Printf("Compiler: %s\nTarget: %s\nSeries: %v\nFile: %s\nCommand: %s %s\n", compiler, target, series, absFile, compiler, strings.Join(cmdArgs, " "))
	}

	c := execCommand(compiler, cmdArgs...)
	if cmd, ok := c.(*exec.Cmd); ok {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	err = c.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 116 {
			// Crestron compiler success code
			return nil
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
