// Package cmd provides CLI commands for seedcli
package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/kiridharan/seedcli/pkg/adapter"
	"github.com/kiridharan/seedcli/pkg/config"
	"github.com/kiridharan/seedcli/pkg/core"
	"github.com/kiridharan/seedcli/pkg/logger"
	"github.com/kiridharan/seedcli/pkg/schema"
	"github.com/kiridharan/seedcli/pkg/seeder"
)

var (
	seedTables     []string
	seedAll        bool
	seedRows       int
	seedBatchSize  int
	seedSeed       int64
	seedDryRun     bool
	seedSkipErrors bool
	seedTruncate   bool
	seedDisableFK  bool
	seedDBURL      string
)

// seedCmd represents the seed command
var seedCmd = &cobra.Command{
	Use:   "seed",
	Short: "Seed database tables with fake data",
	Long: `Seed one or more database tables with intelligently generated fake data.

seedcli analyzes your schema to:
  • Detect column types and generate appropriate data
  • Handle foreign key relationships automatically
  • Respect unique constraints
  • Follow table insertion order based on dependencies

Examples:
  # Seed all tables
  seedcli seed --all

  # Seed specific tables
  seedcli seed -t users -t orders

  # Seed with custom row count
  seedcli seed --all -n 100

  # Dry run (preview without inserting)
  seedcli seed --all --dry-run

  # Reproducible seeding
  seedcli seed --all --seed 12345`,
	RunE: runSeed,
}

func init() {
	seedCmd.Flags().StringSliceVarP(&seedTables, "table", "t", []string{}, "table(s) to seed (can be repeated)")
	seedCmd.Flags().BoolVarP(&seedAll, "all", "a", false, "seed all tables")
	seedCmd.Flags().IntVarP(&seedRows, "rows", "n", 0, "number of rows per table (default from config)")
	seedCmd.Flags().IntVar(&seedBatchSize, "batch-size", 0, "batch size for inserts (default from config)")
	seedCmd.Flags().Int64Var(&seedSeed, "seed", 0, "random seed for reproducible data")
	seedCmd.Flags().BoolVar(&seedDryRun, "dry-run", false, "preview without inserting")
	seedCmd.Flags().BoolVar(&seedSkipErrors, "skip-errors", false, "continue on errors")
	seedCmd.Flags().BoolVar(&seedTruncate, "truncate", false, "truncate tables before seeding")
	seedCmd.Flags().BoolVar(&seedDisableFK, "disable-fk", false, "disable foreign key checks")
	seedCmd.Flags().StringVar(&seedDBURL, "db-url", "", "database URL (overrides config)")
}

func runSeed(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	// Load config
	cfg, err := loadConfig()
	if err != nil {
		// If no config but db-url provided, use minimal config
		if seedDBURL == "" {
			return err
		}
		cfg = defaultMinimalConfig()
	}

	// Override with flags
	dbURL := cfg.GetDSN()
	if seedDBURL != "" {
		dbURL = seedDBURL
	}

	if dbURL == "" {
		return fmt.Errorf("database URL required (use --db-url or configure in seedcli.yaml)")
	}

	// Check table selection
	if !seedAll && len(seedTables) == 0 {
		return fmt.Errorf("must specify --all or --table/-t")
	}

	// Connect to database
	logger.Info("Connecting to database...")
	dbAdapter, err := adapter.CreateAndConnect(ctx, dbURL)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer dbAdapter.Close()

	logger.Success("Connected to database", logger.F("dialect", string(dbAdapter.Dialect())))

	// Create seeder
	seederConfig := &seeder.Config{
		RowsPerCollection: cfg.Seeding.DefaultRows,
		BatchSize:         cfg.Seeding.BatchSize,
		Seed:              seedSeed,
		DryRun:            seedDryRun,
		SkipErrors:        seedSkipErrors,
		Truncate:          seedTruncate,
		DisableForeignKeys: seedDisableFK,
	}

	// Override with flags
	if seedRows > 0 {
		seederConfig.RowsPerCollection = seedRows
	}
	if seedBatchSize > 0 {
		seederConfig.BatchSize = seedBatchSize
	}
	if seedSeed == 0 {
		seederConfig.Seed = time.Now().UnixNano()
	}

	s := seeder.NewSeeder(dbAdapter, seederConfig)

	// Get tables to seed
	var tables []string
	if seedAll {
		// Get all tables from schema engine
		logger.Info("Discovering tables...")
		// We need to use the schema engine here
		allTables, err := listTablesFromDB(ctx, dbAdapter)
		if err != nil {
			return fmt.Errorf("failed to list tables: %w", err)
		}
		tables = allTables
		logger.Info("Found tables", logger.F("count", len(tables)))
	} else {
		tables = seedTables
	}

	if len(tables) == 0 {
		return fmt.Errorf("no tables to seed")
	}

	// Seed
	logger.Info("Starting seeding process...", 
		logger.F("tables", len(tables)), 
		logger.F("rows_per_table", seederConfig.RowsPerCollection),
		logger.F("seed", seederConfig.Seed))

	opts := core.SeedOptions{
		RowsPerCollection: seederConfig.RowsPerCollection,
		BatchSize:         seederConfig.BatchSize,
		Seed:              seederConfig.Seed,
		DryRun:            seederConfig.DryRun,
		SkipErrors:        seederConfig.SkipErrors,
		Truncate:          seederConfig.Truncate,
		DisableFK:         seederConfig.DisableForeignKeys,
	}

	result, err := s.Seed(ctx, tables, opts)
	if err != nil && !seedSkipErrors {
		return fmt.Errorf("seeding failed: %w", err)
	}

	// Print results
	printSeedResults(result)

	if len(result.Errors) > 0 {
		logger.Warn("Completed with errors", logger.F("error_count", len(result.Errors)))
	} else {
		logger.Success("Seeding completed successfully!")
	}

	return nil
}

func printSeedResults(result *core.SeedResult) {
	fmt.Println("\n📊 Seeding Results")
	fmt.Println(strings.Repeat("─", 50))

	for _, col := range result.Collections {
		status := "✅"
		if col.Error != nil {
			status = "❌"
		}
		fmt.Printf("%s %s: %d rows (%.2fs)\n", 
			status, col.Name, col.RowsInserted, col.Duration.Seconds())
	}

	fmt.Println(strings.Repeat("─", 50))
	fmt.Printf("Total: %d rows in %.2fs\n", result.TotalRows, result.Duration.Seconds())

	if len(result.Errors) > 0 {
		fmt.Printf("\n⚠️  %d error(s) occurred:\n", len(result.Errors))
		for _, err := range result.Errors {
			fmt.Printf("   • %s\n", err.Error())
		}
	}
}

// listTablesFromDB lists all tables using the schema engine
func listTablesFromDB(ctx context.Context, adapter core.Adapter) ([]string, error) {
	engine := schema.NewSQLEngine()
	engine.SetAdapter(adapter)
	return engine.ListCollections(ctx)
}

// defaultMinimalConfig returns a minimal config for flag-only usage
func defaultMinimalConfig() *config.Config {
	return &config.Config{
		Seeding: config.SeedingConfig{
			DefaultRows: 10,
			BatchSize:   100,
		},
	}
}
