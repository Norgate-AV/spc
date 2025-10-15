package compiler

import (
	"fmt"
	"path/filepath"

	"github.com/Norgate-AV/spc/internal/config"
	"github.com/Norgate-AV/spc/internal/utils"
)

type ShellCommand struct {
	Path string
	Args []string
}

func GetBuildCommand(cfg *config.Config, files []string) (*ShellCommand, error) {
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

	return &ShellCommand{
		Path: cfg.CompilerPath,
		Args: cmdArgs,
	}, nil
}
