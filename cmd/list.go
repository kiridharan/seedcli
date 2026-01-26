// Package cmd provides CLI commands for seedcli
package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/kiridharan/seedcli/pkg/adapter"
	"github.com/kiridharan/seedcli/pkg/core"
	"github.com/kiridharan/seedcli/pkg/logger"
	"github.com/kiridharan/seedcli/pkg/schema"
)

var (
	listDBURL   string
	listDetails bool
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List tables in the database",
	Long: `List all tables in the connected database along with their structure.

Examples:
  # List all tables
  seedcli list

  # List with details (columns, FKs, etc.)
  seedcli list --details

  # Use specific database URL
  seedcli list --db-url "postgres://..."`,
	RunE: runList,
}

func init() {
	listCmd.Flags().StringVar(&listDBURL, "db-url", "", "database URL (overrides config)")
	listCmd.Flags().BoolVarP(&listDetails, "details", "d", false, "show table details")
}

func runList(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Load config
	cfg, err := loadConfig()
	if err != nil {
		if listDBURL == "" {
			return err
		}
		cfg = defaultMinimalConfig()
	}

	// Get database URL
	dbURL := cfg.GetDSN()
	if listDBURL != "" {
		dbURL = listDBURL
	}

	if dbURL == "" {
		return fmt.Errorf("database URL required")
	}

	// Connect
	logger.Info("Connecting to database...")
	dbAdapter, err := adapter.CreateAndConnect(ctx, dbURL)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer dbAdapter.Close()

	logger.Success("Connected", logger.F("dialect", string(dbAdapter.Dialect())))

	// Create schema engine
	engine := schema.NewSQLEngine()
	engine.SetAdapter(dbAdapter)

	// List tables
	tables, err := engine.ListCollections(ctx)
	if err != nil {
		return fmt.Errorf("failed to list tables: %w", err)
	}

	if len(tables) == 0 {
		fmt.Println("\nNo tables found in the database.")
		return nil
	}

	fmt.Printf("\n📋 Tables (%d)\n", len(tables))
	fmt.Println("─────────────────────────────────────")

	if listDetails {
		// Show detailed view
		for _, tableName := range tables {
			col, err := engine.IntrospectCollection(ctx, tableName)
			if err != nil {
				logger.Warn("Failed to introspect table", logger.F("table", tableName), logger.F("error", err.Error()))
				continue
			}

			printTableDetails(col)
		}
	} else {
		// Simple list
		for _, tableName := range tables {
			fmt.Printf("  • %s\n", tableName)
		}
	}

	fmt.Printf("\nTotal: %d table(s)\n", len(tables))

	return nil
}

func printTableDetails(col *core.Collection) {
	fmt.Printf("\n📦 %s\n", col.Name)
	
	// Columns
	fmt.Println("   Columns:")
	for _, field := range col.Fields {
		flags := ""
		if field.IsPrimaryKey {
			flags += " [PK]"
		}
		if field.IsAutoIncr {
			flags += " [AUTO]"
		}
		if field.IsUnique {
			flags += " [UNIQUE]"
		}
		if !field.IsNullable {
			flags += " [NOT NULL]"
		}
		
		fmt.Printf("     • %s %s%s\n", field.Name, field.RawType, flags)
	}

	// Foreign Keys
	if len(col.ForeignKeys) > 0 {
		fmt.Println("   Foreign Keys:")
		for _, fk := range col.ForeignKeys {
			fmt.Printf("     • %s → %s.%s\n", fk.ColumnName, fk.ReferencedTable, fk.ReferencedColumn)
		}
	}

	// Indexes
	if len(col.Indexes) > 0 {
		fmt.Println("   Indexes:")
		for _, idx := range col.Indexes {
			unique := ""
			if idx.IsUnique {
				unique = " (unique)"
			}
			fmt.Printf("     • %s%s\n", idx.Name, unique)
		}
	}
}
