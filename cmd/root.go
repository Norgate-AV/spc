package cmd

import (
	"fmt"
	"os"

	"github.com/Norgate-AV/spc/internal/version"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:          "spc",
	Short:        "Better SIMPL+ Compiler",
	Long:         `A better way to compile Crestron SIMPL+ files`,
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
	rootCmd.PersistentFlags().BoolP("silent", "s", false, "Suppress console output from the SIMPL+ compiler")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Verbose output")
	rootCmd.PersistentFlags().StringP("out", "o", "", "Output file for compilation logs")
	rootCmd.PersistentFlags().StringSliceP("usersplusfolder", "u", []string{}, "User SIMPL+ folders")
	rootCmd.PersistentFlags().Bool("no-cache", false, "Disable build cache")
	rootCmd.AddCommand(buildCmd)

	viper.SetDefault("compiler_path", "C:/Program Files (x86)/Crestron/Simpl/SPlusCC.exe")
	viper.SetDefault("target", "234")
	viper.SetDefault("silent", false)
	viper.SetDefault("verbose", false)
}
