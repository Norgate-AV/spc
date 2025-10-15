package compiler

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Norgate-AV/spc/internal/config"
	"github.com/Norgate-AV/spc/internal/utils"
)

// Commander interface for testing
type Commander interface {
	Run() error
}

// CommandBuilder handles building compiler commands
type CommandBuilder struct {
	execCommand func(name string, args ...string) Commander
}

// NewCommandBuilder creates a new command builder
func NewCommandBuilder() *CommandBuilder {
	return &CommandBuilder{
		execCommand: func(name string, args ...string) Commander {
			return exec.Command(name, args...)
		},
	}
}

// BuildCommandArgs builds the command arguments for the compiler
func (cb *CommandBuilder) BuildCommandArgs(cfg *config.Config, files []string) ([]string, error) {
	series := utils.ParseTarget(cfg.Target)
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

// ExecuteCommand executes the compiler command
func (cb *CommandBuilder) ExecuteCommand(compilerPath string, cmdArgs []string) error {
	c := cb.execCommand(compilerPath, cmdArgs...)
	if cmd, ok := c.(*exec.Cmd); ok {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	err := c.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			code := exitErr.ExitCode()
			if IsSuccess(code) {
				// Crestron compiler success (may have warnings)
				return nil
			}

			// Print descriptive error message
			fmt.Fprintf(os.Stderr, "Compilation failed (exit code %d): %s\n", code, GetErrorMessage(code))
		}

		return err
	}

	return nil
}

// PrintBuildInfo prints verbose build information
func (cb *CommandBuilder) PrintBuildInfo(cfg *config.Config, series []string, args []string, cmdArgs []string) {
	fmt.Printf("Compiler: %s\nTarget: %s\nSeries: %v\nFiles: %v\nOut: %s\nUsersPlusFolders: %v\nCommand: %s %s\n",
		cfg.CompilerPath, cfg.Target, series, args, cfg.OutputFile, cfg.UserFolders, cfg.CompilerPath, strings.Join(cmdArgs, " "))
}
