package cmd

import (
	"fmt"
	"os"

	"github.com/Norgate-AV/spc/internal/version"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:          "spc",
	Short:        "Crestron SIMPL+ compiler wrapper",
	Long:         `A CLI wrapper for the Crestron SIMPL+ compiler tool.`,
	RunE:         runBuild,
	SilenceUsage: true,
	Args:         cobra.ArbitraryArgs,
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Version = fmt.Sprintf("%s (%s) %s", version.Version, version.Commit, version.BuildTime)
	rootCmd.PersistentFlags().StringP("target", "t", "", "Target series to compile for (e.g., 3, 34, 234)")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Verbose output")
	rootCmd.AddCommand(buildCmd)
}
