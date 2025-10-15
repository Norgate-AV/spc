package cmd

import (
	"github.com/Norgate-AV/spc/internal/build"
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
	service := build.NewBuildService()
	return service.Build(cmd, args)

	// Resolve config
	// cfg, err := config.Load()
	// if err != nil {
	// 	return fmt.Errorf("error loading config: %w", err)
	// }

	// fmt.Printf("%+v\n", cfg)
	// fmt.Printf("%+v\n", args)

	// Validate input
	// if len(args) == 0 {
	// 	return cmd.Help()
	// }

	// return nil
}
