// Package cmd provides CLI commands for seedcli
package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/kiridharan/seedcli/pkg/adapter"
	"github.com/kiridharan/seedcli/pkg/core"
	"github.com/kiridharan/seedcli/pkg/logger"
	"github.com/kiridharan/seedcli/pkg/schema"
	"github.com/kiridharan/seedcli/pkg/seeder"
)

var (
	previewTables []string
	previewAll    bool
	previewRows   int
	previewDBURL  string
	previewJSON   bool
)

// previewCmd represents the preview command
var previewCmd = &cobra.Command{
	Use:   "preview",
	Short: "Preview generated data without inserting",
	Long: `Generate and display sample fake data without inserting into the database.

This is useful for testing your configuration and seeing what kind of data
will be generated for your tables.

Examples:
  # Preview data for all tables
  seedcli preview --all

  # Preview specific tables
  seedcli preview -t users -t orders

  # Preview with more rows
  seedcli preview -t users -n 5

  # Output as JSON
  seedcli preview -t users --json`,
	RunE: runPreview,
}

func init() {
	previewCmd.Flags().StringSliceVarP(&previewTables, "table", "t", []string{}, "table(s) to preview")
	previewCmd.Flags().BoolVarP(&previewAll, "all", "a", false, "preview all tables")
	previewCmd.Flags().IntVarP(&previewRows, "rows", "n", 3, "number of sample rows")
	previewCmd.Flags().StringVar(&previewDBURL, "db-url", "", "database URL (overrides config)")
	previewCmd.Flags().BoolVar(&previewJSON, "json", false, "output as JSON")
}

func runPreview(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Load config
	cfg, err := loadConfig()
	if err != nil {
		if previewDBURL == "" {
			return err
		}
		cfg = defaultMinimalConfig()
	}

	// Get database URL
	dbURL := cfg.GetDSN()
	if previewDBURL != "" {
		dbURL = previewDBURL
	}

	if dbURL == "" {
		return fmt.Errorf("database URL required")
	}

	// Check table selection
	if !previewAll && len(previewTables) == 0 {
		return fmt.Errorf("must specify --all or --table/-t")
	}

	// Connect
	logger.Info("Connecting to database...")
	dbAdapter, err := adapter.CreateAndConnect(ctx, dbURL)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer dbAdapter.Close()

	// Get tables
	var tables []string
	if previewAll {
		engine := schema.NewSQLEngine()
		engine.SetAdapter(dbAdapter)
		tables, err = engine.ListCollections(ctx)
		if err != nil {
			return fmt.Errorf("failed to list tables: %w", err)
		}
	} else {
		tables = previewTables
	}

	// Create seeder for preview
	s := seeder.NewSeeder(dbAdapter, seeder.DefaultConfig())

	// Generate preview
	opts := core.SeedOptions{
		RowsPerCollection: previewRows,
	}

	result, err := s.Preview(ctx, tables, opts)
	if err != nil {
		return fmt.Errorf("preview failed: %w", err)
	}

	// Output results
	if previewJSON {
		outputPreviewJSON(result)
	} else {
		outputPreviewText(result)
	}

	return nil
}

func outputPreviewJSON(result *core.PreviewResult) {
	data, _ := json.MarshalIndent(result.Collections, "", "  ")
	fmt.Println(string(data))
}

func outputPreviewText(result *core.PreviewResult) {
	for tableName, rows := range result.Collections {
		fmt.Printf("\n═══════════════════════════════════════\n")
		fmt.Printf("📋 %s (%d rows)\n", tableName, len(rows))
		fmt.Printf("═══════════════════════════════════════\n")

		for i, row := range rows {
			fmt.Printf("\n  Row %d:\n", i+1)
			for col, val := range row {
				displayVal := formatValue(val)
				fmt.Printf("    %s: %s\n", col, displayVal)
			}
		}
	}
}

func formatValue(val interface{}) string {
	if val == nil {
		return "NULL"
	}

	switch v := val.(type) {
	case string:
		if len(v) > 50 {
			return fmt.Sprintf("\"%s...\"", v[:47])
		}
		return fmt.Sprintf("\"%s\"", v)
	case []byte:
		return fmt.Sprintf("[binary %d bytes]", len(v))
	case map[string]interface{}:
		data, _ := json.Marshal(v)
		if len(data) > 50 {
			return string(data[:47]) + "..."
		}
		return string(data)
	default:
		return fmt.Sprintf("%v", v)
	}
}
