package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:          "spc",
	Short:        "Crestron SIMPL+ compiler wrapper",
	Long:         `A CLI wrapper for the Crestron SIMPL+ compiler tool.`,
	RunE:         runBuild,
	SilenceUsage: true,
	Args:         cobra.ExactArgs(1),
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringP("target", "t", "", "Target series to compile for (e.g., 3, 34, 234)")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Verbose output")
	rootCmd.AddCommand(buildCmd)
}
