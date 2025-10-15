package cmd

import (
	"fmt"

	"github.com/Norgate-AV/spc/internal/compiler"
	"github.com/Norgate-AV/spc/internal/config"
	"github.com/Norgate-AV/spc/internal/utils"
	"github.com/spf13/cobra"
)

var buildCmd = &cobra.Command{
	Use:          "build",
	Short:        "Build SIMPL+ file(s)",
	Long:         `Build a SIMPL+ file(s) for the specified target series.`,
	RunE:         runBuild,
	SilenceUsage: true,
}

func runBuild(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("no files specified")
	}

	// Load and validate configuration
	configLoader := config.NewLoader()
	cfg, err := configLoader.LoadForBuild(cmd, args)
	if err != nil {
		return err
	}

	// Build compiler command arguments
	builder := compiler.NewCommandBuilder()
	cmdArgs, err := builder.BuildCommandArgs(cfg, args)
	if err != nil {
		return err
	}

	// Print build info if verbose mode is enabled
	if cfg.Verbose {
		series := utils.ParseTarget(cfg.Target)
		builder.PrintBuildInfo(cfg, series, args, cmdArgs)
	}

	// Execute the compiler command
	return builder.ExecuteCommand(cfg.CompilerPath, cmdArgs)
}
