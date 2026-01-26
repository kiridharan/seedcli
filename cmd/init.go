// Package cmd provides CLI commands for seedcli
package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/kiridharan/seedcli/pkg/config"
	"github.com/kiridharan/seedcli/pkg/logger"
)

var (
	initForce    bool
	initAdapter  string
	initURL      string
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize seedcli configuration",
	Long: `Creates a seedcli.yaml configuration file in the current directory.

This command will guide you through setting up your database connection
and seeding preferences. You can also use flags to skip the interactive prompts.

Examples:
  # Interactive setup
  seedcli init

  # Quick setup with flags
  seedcli init --adapter postgres --url "postgres://user:pass@localhost:5432/mydb"

  # Force overwrite existing config
  seedcli init --force`,
	RunE: runInit,
}

func init() {
	initCmd.Flags().BoolVarP(&initForce, "force", "f", false, "overwrite existing config file")
	initCmd.Flags().StringVarP(&initAdapter, "adapter", "a", "", "database adapter (postgres, sqlite)")
	initCmd.Flags().StringVarP(&initURL, "url", "u", "", "database connection URL")
}

func runInit(cmd *cobra.Command, args []string) error {
	logger.Info("Initializing seedcli configuration...")

	// Check if config already exists
	if _, err := os.Stat(config.ConfigFileName); err == nil && !initForce {
		return fmt.Errorf("config file %s already exists (use --force to overwrite)", config.ConfigFileName)
	}

	// Create default config
	cfg := config.DefaultConfig()

	// Interactive or flag-based setup
	if initAdapter == "" || initURL == "" {
		// Interactive mode
		if err := interactiveSetup(cfg); err != nil {
			return err
		}
	} else {
		// Flag-based setup
		cfg.Database.Adapter = initAdapter
		cfg.Database.URL = initURL
	}

	// Validate config
	if errs := cfg.Validate(); len(errs) > 0 {
		for _, e := range errs {
			logger.Warn("Validation warning", logger.F("issue", e.Error()))
		}
	}

	// Save config
	if err := cfg.SaveToCWD(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	// Create .logseed directory
	if _, err := config.EnsureLogDir(); err != nil {
		logger.Warn("Failed to create log directory", logger.F("error", err.Error()))
	}

	logger.Success("Configuration created successfully!")
	logger.Info("Config file", logger.F("path", config.ConfigFileName))
	logger.Info("Log directory", logger.F("path", config.LogDirName))
	
	fmt.Println("\nNext steps:")
	fmt.Println("  1. Review and edit seedcli.yaml as needed")
	fmt.Println("  2. Run 'seedcli list' to see available tables")
	fmt.Println("  3. Run 'seedcli seed --all' to seed all tables")

	return nil
}

func interactiveSetup(cfg *config.Config) error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("\n🌱 seedcli Configuration Setup\n")

	// Database adapter
	fmt.Println("Available database adapters:")
	fmt.Println("  1. postgres  - PostgreSQL")
	fmt.Println("  2. sqlite    - SQLite")
	fmt.Print("\nSelect adapter [1]: ")
	
	adapterChoice, _ := reader.ReadString('\n')
	adapterChoice = strings.TrimSpace(adapterChoice)
	
	switch adapterChoice {
	case "", "1", "postgres":
		cfg.Database.Adapter = "postgres"
	case "2", "sqlite":
		cfg.Database.Adapter = "sqlite"
	default:
		cfg.Database.Adapter = adapterChoice
	}

	// Database URL
	var defaultURL string
	switch cfg.Database.Adapter {
	case "postgres":
		defaultURL = "postgres://user:password@localhost:5432/database"
	case "sqlite":
		defaultURL = "sqlite://./data.db"
	}

	fmt.Printf("\nDatabase URL [%s]: ", defaultURL)
	dbURL, _ := reader.ReadString('\n')
	dbURL = strings.TrimSpace(dbURL)
	if dbURL == "" {
		dbURL = defaultURL
	}
	cfg.Database.URL = dbURL

	// Default rows
	fmt.Print("\nDefault rows per table [10]: ")
	rowsStr, _ := reader.ReadString('\n')
	rowsStr = strings.TrimSpace(rowsStr)
	if rowsStr != "" {
		if rows, err := strconv.Atoi(rowsStr); err == nil {
			cfg.Seeding.DefaultRows = rows
		}
	}

	// Batch size
	fmt.Print("Batch size for inserts [100]: ")
	batchStr, _ := reader.ReadString('\n')
	batchStr = strings.TrimSpace(batchStr)
	if batchStr != "" {
		if batch, err := strconv.Atoi(batchStr); err == nil {
			cfg.Seeding.BatchSize = batch
		}
	}

	// Logging
	fmt.Print("\nEnable file logging? [Y/n]: ")
	logChoice, _ := reader.ReadString('\n')
	logChoice = strings.TrimSpace(strings.ToLower(logChoice))
	cfg.Logging.ToFile = logChoice != "n" && logChoice != "no"

	return nil
}
