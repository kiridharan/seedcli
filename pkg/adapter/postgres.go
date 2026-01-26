// Package adapter provides database adapter implementations
package adapter

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/kiridharan/seedcli/pkg/core"
)

// PostgresAdapter implements core.Adapter for PostgreSQL
type PostgresAdapter struct {
	db     *sql.DB
	dsn    string
	schema string
}

// NewPostgresAdapter creates a new PostgreSQL adapter
func NewPostgresAdapter() *PostgresAdapter {
	return &PostgresAdapter{
		schema: "public",
	}
}

// Connect establishes a connection to the database
func (a *PostgresAdapter) Connect(ctx context.Context, dsn string) error {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return fmt.Errorf("failed to ping database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)

	a.db = db
	a.dsn = dsn

	return nil
}

// Close terminates the database connection
func (a *PostgresAdapter) Close() error {
	if a.db != nil {
		return a.db.Close()
	}
	return nil
}

// Ping verifies the connection is alive
func (a *PostgresAdapter) Ping(ctx context.Context) error {
	if a.db == nil {
		return fmt.Errorf("database not connected")
	}
	return a.db.PingContext(ctx)
}

// Execute runs a query without returning rows
func (a *PostgresAdapter) Execute(ctx context.Context, query string, args ...interface{}) (core.Result, error) {
	if a.db == nil {
		return nil, fmt.Errorf("database not connected")
	}
	return a.db.ExecContext(ctx, query, args...)
}

// Query runs a query and returns rows
func (a *PostgresAdapter) Query(ctx context.Context, query string, args ...interface{}) (core.Rows, error) {
	if a.db == nil {
		return nil, fmt.Errorf("database not connected")
	}
	return a.db.QueryContext(ctx, query, args...)
}

// BeginTx starts a new transaction
func (a *PostgresAdapter) BeginTx(ctx context.Context) (core.Transaction, error) {
	if a.db == nil {
		return nil, fmt.Errorf("database not connected")
	}
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	return &PostgresTransaction{tx: tx}, nil
}

// Dialect returns the database dialect identifier
func (a *PostgresAdapter) Dialect() core.Dialect {
	return core.DialectPostgres
}

// QuoteIdentifier quotes an identifier for PostgreSQL
func (a *PostgresAdapter) QuoteIdentifier(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

// Placeholder returns the placeholder syntax for PostgreSQL
func (a *PostgresAdapter) Placeholder(index int) string {
	return fmt.Sprintf("$%d", index)
}

// SetSchema sets the search path schema
func (a *PostgresAdapter) SetSchema(schema string) {
	a.schema = schema
}

// GetDB returns the underlying sql.DB (for advanced operations)
func (a *PostgresAdapter) GetDB() *sql.DB {
	return a.db
}

// PostgresTransaction implements core.Transaction for PostgreSQL
type PostgresTransaction struct {
	tx *sql.Tx
}

// Execute runs a query without returning rows
func (t *PostgresTransaction) Execute(ctx context.Context, query string, args ...interface{}) (core.Result, error) {
	return t.tx.ExecContext(ctx, query, args...)
}

// Query runs a query and returns rows
func (t *PostgresTransaction) Query(ctx context.Context, query string, args ...interface{}) (core.Rows, error) {
	return t.tx.QueryContext(ctx, query, args...)
}

// Commit commits the transaction
func (t *PostgresTransaction) Commit() error {
	return t.tx.Commit()
}

// Rollback rolls back the transaction
func (t *PostgresTransaction) Rollback() error {
	return t.tx.Rollback()
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

// BuildInsertQuery builds an INSERT query for PostgreSQL
func (a *PostgresAdapter) BuildInsertQuery(table string, columns []string) string {
	quoted := make([]string, len(columns))
	placeholders := make([]string, len(columns))

	for i, col := range columns {
		quoted[i] = a.QuoteIdentifier(col)
		placeholders[i] = a.Placeholder(i + 1)
	}

	return fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s) RETURNING *",
		a.QuoteIdentifier(table),
		strings.Join(quoted, ", "),
		strings.Join(placeholders, ", "),
	)
}

// BuildBulkInsertQuery builds a bulk INSERT query
func (a *PostgresAdapter) BuildBulkInsertQuery(table string, columns []string, rowCount int) string {
	quoted := make([]string, len(columns))
	for i, col := range columns {
		quoted[i] = a.QuoteIdentifier(col)
	}

	var values []string
	placeholder := 1
	for i := 0; i < rowCount; i++ {
		rowPlaceholders := make([]string, len(columns))
		for j := range columns {
			rowPlaceholders[j] = a.Placeholder(placeholder)
			placeholder++
		}
		values = append(values, "("+strings.Join(rowPlaceholders, ", ")+")")
	}

	return fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES %s RETURNING *",
		a.QuoteIdentifier(table),
		strings.Join(quoted, ", "),
		strings.Join(values, ", "),
	)
}

// TruncateTable truncates a table
func (a *PostgresAdapter) TruncateTable(ctx context.Context, table string, cascade bool) error {
	query := fmt.Sprintf("TRUNCATE TABLE %s", a.QuoteIdentifier(table))
	if cascade {
		query += " CASCADE"
	}
	_, err := a.Execute(ctx, query)
	return err
}

// DisableForeignKeyChecks disables FK checks (within transaction)
func (a *PostgresAdapter) DisableForeignKeyChecks(ctx context.Context) error {
	_, err := a.Execute(ctx, "SET session_replication_role = replica")
	return err
}

// EnableForeignKeyChecks enables FK checks
func (a *PostgresAdapter) EnableForeignKeyChecks(ctx context.Context) error {
	_, err := a.Execute(ctx, "SET session_replication_role = DEFAULT")
	return err
}

// GetTableRowCount returns the number of rows in a table
func (a *PostgresAdapter) GetTableRowCount(ctx context.Context, table string) (int64, error) {
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", a.QuoteIdentifier(table))
	rows, err := a.Query(ctx, query)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var count int64
	if rows.Next() {
		if err := rows.Scan(&count); err != nil {
			return 0, err
		}
	}

	return count, nil
}
