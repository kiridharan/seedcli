// Package seeder provides the main seeding orchestration
package seeder

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/kiridharan/seedcli/pkg/core"
	"github.com/kiridharan/seedcli/pkg/data"
	"github.com/kiridharan/seedcli/pkg/logger"
	"github.com/kiridharan/seedcli/pkg/plugin"
	"github.com/kiridharan/seedcli/pkg/schema"
)

// Seeder orchestrates the database seeding process
type Seeder struct {
	adapter      core.Adapter
	schemaEngine core.SchemaEngine
	dataEngine   *data.Engine
	hookManager  *plugin.HookManager
	config       *Config
}

// Config holds seeder configuration
type Config struct {
	RowsPerCollection int
	BatchSize         int
	Seed              int64
	DryRun            bool
	SkipErrors        bool
	Truncate          bool
	DisableForeignKeys bool
	OnConflict        core.ConflictStrategy
}

// DefaultConfig returns a default seeder configuration
func DefaultConfig() *Config {
	return &Config{
		RowsPerCollection: 10,
		BatchSize:         100,
		Seed:              0,
		DryRun:            false,
		SkipErrors:        false,
		Truncate:          false,
		DisableForeignKeys: false,
		OnConflict:        core.ConflictError,
	}
}

// NewSeeder creates a new seeder instance
func NewSeeder(adapter core.Adapter, config *Config) *Seeder {
	if config == nil {
		config = DefaultConfig()
	}

	schemaEngine := schema.NewSQLEngine()
	schemaEngine.SetAdapter(adapter)

	dataEngine := data.NewEngine()
	if config.Seed != 0 {
		dataEngine.SetSeed(config.Seed)
	}

	return &Seeder{
		adapter:      adapter,
		schemaEngine: schemaEngine,
		dataEngine:   dataEngine,
		hookManager:  plugin.NewHookManager(),
		config:       config,
	}
}

// AddPlugin adds a plugin to the seeder
func (s *Seeder) AddPlugin(p core.Plugin) {
	s.hookManager.AddPlugin(p)
}

// RegisterGenerator registers a custom generator
func (s *Seeder) RegisterGenerator(name string, gen core.Generator) {
	s.dataEngine.RegisterGenerator(name, gen)
}

// Seed seeds the specified collections
func (s *Seeder) Seed(ctx context.Context, collectionNames []string, opts core.SeedOptions) (*core.SeedResult, error) {
	startTime := time.Now()
	result := &core.SeedResult{
		Collections: []core.CollectionResult{},
		Errors:      []error{},
	}

	// Override config with opts
	if opts.RowsPerCollection > 0 {
		s.config.RowsPerCollection = opts.RowsPerCollection
	}
	if opts.BatchSize > 0 {
		s.config.BatchSize = opts.BatchSize
	}
	if opts.Seed > 0 {
		s.config.Seed = opts.Seed
		s.dataEngine.SetSeed(opts.Seed)
	}
	s.config.DryRun = opts.DryRun
	s.config.SkipErrors = opts.SkipErrors
	s.config.Truncate = opts.Truncate
	s.config.DisableForeignKeys = opts.DisableFK
	s.config.OnConflict = opts.OnConflict

	// Introspect collections
	logger.Info("Introspecting schema...", logger.F("tables", len(collectionNames)))
	collections := make([]*core.Collection, 0, len(collectionNames))
	for _, name := range collectionNames {
		col, err := s.schemaEngine.IntrospectCollection(ctx, name)
		if err != nil {
			return nil, fmt.Errorf("failed to introspect %s: %w", name, err)
		}
		collections = append(collections, col)
	}

	// Get dependency order
	sortedCollections, err := s.schemaEngine.GetDependencyOrder(collections)
	if err != nil {
		logger.Warn("Cyclic dependencies detected, using best-effort order")
		sortedCollections = collections
	}

	// Execute BeforeSeed hooks
	if err := s.hookManager.BeforeSeed(ctx, sortedCollections); err != nil {
		return nil, fmt.Errorf("BeforeSeed hook failed: %w", err)
	}

	// Disable FK checks if requested
	if s.config.DisableForeignKeys {
		if err := s.disableFKChecks(ctx); err != nil {
			logger.Warn("Failed to disable FK checks", logger.F("error", err.Error()))
		}
	}

	// Truncate if requested
	if s.config.Truncate {
		if err := s.truncateCollections(ctx, sortedCollections); err != nil {
			return nil, fmt.Errorf("failed to truncate: %w", err)
		}
	}

	// Seed each collection
	for _, col := range sortedCollections {
		colResult, err := s.seedCollection(ctx, col)
		result.Collections = append(result.Collections, colResult)
		result.TotalRows += colResult.RowsInserted

		if err != nil {
			result.Errors = append(result.Errors, err)
			s.hookManager.OnError(ctx, err)

			if !s.config.SkipErrors {
				break
			}
		}
	}

	// Re-enable FK checks
	if s.config.DisableForeignKeys {
		if err := s.enableFKChecks(ctx); err != nil {
			logger.Warn("Failed to enable FK checks", logger.F("error", err.Error()))
		}
	}

	result.Duration = time.Since(startTime)

	// Execute AfterSeed hooks
	if err := s.hookManager.AfterSeed(ctx, result); err != nil {
		logger.Warn("AfterSeed hook failed", logger.F("error", err.Error()))
	}

	return result, nil
}

// SeedAll seeds all collections in the database
func (s *Seeder) SeedAll(ctx context.Context, opts core.SeedOptions) (*core.SeedResult, error) {
	collectionNames, err := s.schemaEngine.ListCollections(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list collections: %w", err)
	}

	return s.Seed(ctx, collectionNames, opts)
}

// Preview generates sample data without inserting
func (s *Seeder) Preview(ctx context.Context, collectionNames []string, opts core.SeedOptions) (*core.PreviewResult, error) {
	result := &core.PreviewResult{
		Collections: make(map[string][]map[string]interface{}),
	}

	rowCount := opts.RowsPerCollection
	if rowCount == 0 {
		rowCount = 3
	}

	for _, name := range collectionNames {
		col, err := s.schemaEngine.IntrospectCollection(ctx, name)
		if err != nil {
			return nil, fmt.Errorf("failed to introspect %s: %w", name, err)
		}

		rows, err := s.dataEngine.GenerateRows(ctx, col, rowCount)
		if err != nil {
			return nil, fmt.Errorf("failed to generate preview for %s: %w", name, err)
		}

		result.Collections[name] = rows
	}

	return result, nil
}

// seedCollection seeds a single collection
func (s *Seeder) seedCollection(ctx context.Context, collection *core.Collection) (core.CollectionResult, error) {
	startTime := time.Now()
	result := core.CollectionResult{
		Name: collection.Name,
	}

	logger.Info("Seeding collection", 
		logger.F("table", collection.Name), 
		logger.F("rows", s.config.RowsPerCollection))

	if s.config.DryRun {
		logger.Info("DRY RUN: Would insert rows", 
			logger.F("table", collection.Name), 
			logger.F("count", s.config.RowsPerCollection))
		result.RowsInserted = int64(s.config.RowsPerCollection)
		result.Duration = time.Since(startTime)
		return result, nil
	}

	// Filter out auto-increment columns
	insertColumns := s.getInsertColumns(collection)

	// Generate and insert in batches
	totalInserted := int64(0)
	remaining := s.config.RowsPerCollection

	for remaining > 0 {
		batchSize := s.config.BatchSize
		if batchSize > remaining {
			batchSize = remaining
		}

		// Generate batch
		rows, err := s.dataEngine.GenerateRows(ctx, collection, batchSize)
		if err != nil {
			result.Error = err
			result.Duration = time.Since(startTime)
			return result, err
		}

		// Execute BeforeInsert hooks
		if err := s.hookManager.BeforeInsert(ctx, collection.Name, rows); err != nil {
			result.Error = err
			result.Duration = time.Since(startTime)
			return result, err
		}

		// Insert batch
		insertedPKs, err := s.insertBatch(ctx, collection, insertColumns, rows)
		if err != nil {
			result.Error = err
			result.Duration = time.Since(startTime)
			return result, err
		}

		// Store PKs for FK references
		if len(insertedPKs) > 0 {
			existingPKs := s.dataEngine.GetReferenceData(collection.Name)
			s.dataEngine.SetReferenceData(collection.Name, "", append(existingPKs, insertedPKs...))
		}

		// Execute AfterInsert hooks
		if err := s.hookManager.AfterInsert(ctx, collection.Name, int64(len(rows))); err != nil {
			logger.Warn("AfterInsert hook failed", logger.F("error", err.Error()))
		}

		totalInserted += int64(len(rows))
		remaining -= batchSize
	}

	result.RowsInserted = totalInserted
	result.Duration = time.Since(startTime)

	logger.Success("Seeded collection", 
		logger.F("table", collection.Name), 
		logger.F("rows", totalInserted),
		logger.F("duration", result.Duration.String()))

	return result, nil
}

// getInsertColumns returns columns that should be included in INSERT
func (s *Seeder) getInsertColumns(collection *core.Collection) []string {
	columns := []string{}
	for _, field := range collection.Fields {
		if !field.IsAutoIncr {
			columns = append(columns, field.Name)
		}
	}
	return columns
}

// insertBatch inserts a batch of rows
func (s *Seeder) insertBatch(ctx context.Context, collection *core.Collection, columns []string, rows []map[string]interface{}) ([]interface{}, error) {
	if len(rows) == 0 {
		return nil, nil
	}

	// Build INSERT query
	query := s.buildInsertQuery(collection.Name, columns, len(rows))

	// Build args
	args := make([]interface{}, 0, len(rows)*len(columns))
	for _, row := range rows {
		for _, col := range columns {
			args = append(args, row[col])
		}
	}

	// Execute insert
	result, err := s.adapter.Execute(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("insert failed: %w", err)
	}

	// Get inserted PKs (for FK references)
	var pks []interface{}
	if len(collection.PrimaryKey) > 0 && collection.PrimaryKey[0] != "" {
		// Try to get last insert ID
		if lastID, err := result.LastInsertId(); err == nil && lastID > 0 {
			// For single-row insert
			if len(rows) == 1 {
				pks = append(pks, lastID)
			} else {
				// For batch insert, we need to query the PKs
				// This is a limitation - we'll use the generated values from rows
				for _, row := range rows {
					if pk, ok := row[collection.PrimaryKey[0]]; ok && pk != nil {
						pks = append(pks, pk)
					}
				}
			}
		}
	}

	return pks, nil
}

// buildInsertQuery builds an INSERT query
func (s *Seeder) buildInsertQuery(table string, columns []string, rowCount int) string {
	quotedCols := make([]string, len(columns))
	for i, col := range columns {
		quotedCols[i] = s.adapter.QuoteIdentifier(col)
	}

	var valueSets []string
	placeholder := 1
	for i := 0; i < rowCount; i++ {
		rowPlaceholders := make([]string, len(columns))
		for j := range columns {
			rowPlaceholders[j] = s.adapter.Placeholder(placeholder)
			placeholder++
		}
		valueSets = append(valueSets, "("+strings.Join(rowPlaceholders, ", ")+")")
	}

	return fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES %s",
		s.adapter.QuoteIdentifier(table),
		strings.Join(quotedCols, ", "),
		strings.Join(valueSets, ", "),
	)
}

// truncateCollections truncates all collections
func (s *Seeder) truncateCollections(ctx context.Context, collections []*core.Collection) error {
	// Truncate in reverse order (dependents first)
	for i := len(collections) - 1; i >= 0; i-- {
		col := collections[i]
		logger.Info("Truncating table", logger.F("table", col.Name))

		var query string
		if s.adapter.Dialect() == core.DialectSQLite {
			query = fmt.Sprintf("DELETE FROM %s", s.adapter.QuoteIdentifier(col.Name))
		} else {
			query = fmt.Sprintf("TRUNCATE TABLE %s CASCADE", s.adapter.QuoteIdentifier(col.Name))
		}

		if _, err := s.adapter.Execute(ctx, query); err != nil {
			return fmt.Errorf("failed to truncate %s: %w", col.Name, err)
		}
	}

	return nil
}

// disableFKChecks disables foreign key checks
func (s *Seeder) disableFKChecks(ctx context.Context) error {
	var query string
	switch s.adapter.Dialect() {
	case core.DialectPostgres:
		query = "SET session_replication_role = replica"
	case core.DialectSQLite:
		query = "PRAGMA foreign_keys = OFF"
	default:
		return nil
	}
	_, err := s.adapter.Execute(ctx, query)
	return err
}

// enableFKChecks enables foreign key checks
func (s *Seeder) enableFKChecks(ctx context.Context) error {
	var query string
	switch s.adapter.Dialect() {
	case core.DialectPostgres:
		query = "SET session_replication_role = DEFAULT"
	case core.DialectSQLite:
		query = "PRAGMA foreign_keys = ON"
	default:
		return nil
	}
	_, err := s.adapter.Execute(ctx, query)
	return err
}
