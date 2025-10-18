package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Norgate-AV/spc/internal/cache"
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

	// Check if cache is disabled
	noCache, _ := cmd.Flags().GetBool("no-cache")

	// Initialize cache (unless disabled)
	var buildCache *cache.Cache
	if !noCache {
		buildCache, err = cache.New("")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to initialize cache: %v\n", err)
			// Continue without cache
			noCache = true
		} else {
			defer buildCache.Close()
		}
	}

	// Process each source file
	for _, file := range args {
		absFile, err := filepath.Abs(file)
		if err != nil {
			return fmt.Errorf("failed to resolve path for %s: %w", file, err)
		}

		// Check cache (if enabled)
		if !noCache && buildCache != nil {
			entry, err := buildCache.Get(absFile, cfg)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: Cache lookup failed: %v\n", err)
			} else if entry != nil && entry.Success {
				// Cache hit! Restore to source directory
				sourceDir := filepath.Dir(absFile)
				if err := buildCache.Restore(entry, sourceDir); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: Failed to restore from cache: %v\n", err)
				} else {
					if cfg.Verbose {
						fmt.Printf("✓ Using cached build for %s\n", filepath.Base(file))
					}
					continue // Skip compilation
				}
			}
		}

		// Cache miss or disabled - compile
		if cfg.Verbose {
			fmt.Printf("Compiling %s...\n", filepath.Base(file))
		}

		success := true
		if err := compileSingle(cfg, absFile); err != nil {
			success = false
			// Store failed build in cache too (so we don't retry immediately)
			if !noCache && buildCache != nil {
				_ = buildCache.Store(absFile, cfg, false)
			}
			return err
		}

		// Store successful build in cache
		if !noCache && buildCache != nil {
			if err := buildCache.Store(absFile, cfg, success); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: Failed to cache build: %v\n", err)
			}
		}
	}

	return nil
}

// compileSingle compiles a single source file
func compileSingle(cfg *config.Config, sourceFile string) error {
	builder := compiler.NewCommandBuilder()
	cmdArgs, err := builder.BuildCommandArgs(cfg, []string{sourceFile})
	if err != nil {
		return err
	}

	// Print build info if verbose mode is enabled
	if cfg.Verbose {
		series := utils.ParseTarget(cfg.Target)
		builder.PrintBuildInfo(cfg, series, []string{sourceFile}, cmdArgs)
	}

	// Execute the compiler command
	return builder.ExecuteCommand(cfg.CompilerPath, cmdArgs)
}
