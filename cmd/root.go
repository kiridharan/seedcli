// Package cmd provides CLI commands for seedcli
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/kiridharan/seedcli/pkg/config"
	"github.com/kiridharan/seedcli/pkg/logger"
)

var (
	cfgFile string
	verbose bool
)

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "seedcli",
	Short: "A powerful database seeding tool",
	Long: `seedcli is a flexible and extensible database seeding tool.

It supports multiple database adapters (PostgreSQL, SQLite) and provides
intelligent fake data generation based on column types and names.

Features:
  • Automatic schema introspection
  • Topological sorting for FK dependencies
  • Extensible generator and validator system
  • Plugin support for community extensions
  • YAML-based configuration

Get started:
  seedcli init           # Create configuration file
  seedcli seed --all     # Seed all tables
  seedcli list           # List available tables`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Initialize logger
		if verbose {
			logger.Get().SetLevel(0) // Debug level
		}
		logger.Debug("Starting seedcli", logger.F("version", "2.0.0"))
	},
}

// Execute runs the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ./seedcli.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose output")

	// Add subcommands
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(seedCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(previewCmd)
	rootCmd.AddCommand(versionCmd)
}

// initConfig reads in config file
func initConfig() {
	// If config file specified, use it
	if cfgFile != "" {
		cfg, err := config.Load(cfgFile)
		if err != nil {
			logger.Warn("Failed to load config", logger.F("file", cfgFile), logger.F("error", err.Error()))
		}
		if cfg != nil {
			if err := logger.Init(cfg); err != nil {
				logger.Warn("Failed to init logger from config", logger.F("error", err.Error()))
			}
		}
		return
	}

	// Try to load from current directory
	cfg, err := config.LoadFromCWD()
	if err != nil {
		logger.Debug("No config file found in current directory")
	}
	if cfg != nil {
		if err := logger.Init(cfg); err != nil {
			logger.Warn("Failed to init logger from config", logger.F("error", err.Error()))
		}
	}
}

// loadConfig loads and validates configuration
func loadConfig() (*config.Config, error) {
	var cfg *config.Config
	var err error

	if cfgFile != "" {
		cfg, err = config.Load(cfgFile)
	} else {
		cfg, err = config.LoadFromCWD()
	}

	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	if cfg == nil {
		return nil, fmt.Errorf("no configuration found - run 'seedcli init' first")
	}

	// Validate
	if errs := cfg.Validate(); len(errs) > 0 {
		for _, e := range errs {
			logger.Error("Config validation error", logger.F("error", e.Error()))
		}
		return nil, fmt.Errorf("configuration validation failed")
	}

	return cfg, nil
}
