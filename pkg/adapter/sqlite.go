// Package adapter provides database adapter implementations
package adapter

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	"github.com/kiridharan/seedcli/pkg/core"
)

// SQLiteAdapter implements core.Adapter for SQLite
type SQLiteAdapter struct {
	db  *sql.DB
	dsn string
}

// NewSQLiteAdapter creates a new SQLite adapter
func NewSQLiteAdapter() *SQLiteAdapter {
	return &SQLiteAdapter{}
}

// Connect establishes a connection to the database
func (a *SQLiteAdapter) Connect(ctx context.Context, dsn string) error {
	// Handle sqlite:// URL prefix
	dsn = strings.TrimPrefix(dsn, "sqlite://")
	dsn = strings.TrimPrefix(dsn, "sqlite3://")

	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return fmt.Errorf("failed to ping database: %w", err)
	}

	// Enable foreign keys
	if _, err := db.ExecContext(ctx, "PRAGMA foreign_keys = ON"); err != nil {
		db.Close()
		return fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	a.db = db
	a.dsn = dsn

	return nil
}

// Close terminates the database connection
func (a *SQLiteAdapter) Close() error {
	if a.db != nil {
		return a.db.Close()
	}
	return nil
}

// Ping verifies the connection is alive
func (a *SQLiteAdapter) Ping(ctx context.Context) error {
	if a.db == nil {
		return fmt.Errorf("database not connected")
	}
	return a.db.PingContext(ctx)
}

// Execute runs a query without returning rows
func (a *SQLiteAdapter) Execute(ctx context.Context, query string, args ...interface{}) (core.Result, error) {
	if a.db == nil {
		return nil, fmt.Errorf("database not connected")
	}
	return a.db.ExecContext(ctx, query, args...)
}

// Query runs a query and returns rows
func (a *SQLiteAdapter) Query(ctx context.Context, query string, args ...interface{}) (core.Rows, error) {
	if a.db == nil {
		return nil, fmt.Errorf("database not connected")
	}
	return a.db.QueryContext(ctx, query, args...)
}

// BeginTx starts a new transaction
func (a *SQLiteAdapter) BeginTx(ctx context.Context) (core.Transaction, error) {
	if a.db == nil {
		return nil, fmt.Errorf("database not connected")
	}
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	return &SQLiteTransaction{tx: tx}, nil
}

// Dialect returns the database dialect identifier
func (a *SQLiteAdapter) Dialect() core.Dialect {
	return core.DialectSQLite
}

// QuoteIdentifier quotes an identifier for SQLite
func (a *SQLiteAdapter) QuoteIdentifier(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

// Placeholder returns the placeholder syntax for SQLite
func (a *SQLiteAdapter) Placeholder(index int) string {
	return "?"
}

// GetDB returns the underlying sql.DB (for advanced operations)
func (a *SQLiteAdapter) GetDB() *sql.DB {
	return a.db
}

// SQLiteTransaction implements core.Transaction for SQLite
type SQLiteTransaction struct {
	tx *sql.Tx
}

// Execute runs a query without returning rows
func (t *SQLiteTransaction) Execute(ctx context.Context, query string, args ...interface{}) (core.Result, error) {
	return t.tx.ExecContext(ctx, query, args...)
}

// Query runs a query and returns rows
func (t *SQLiteTransaction) Query(ctx context.Context, query string, args ...interface{}) (core.Rows, error) {
	return t.tx.QueryContext(ctx, query, args...)
}

// Commit commits the transaction
func (t *SQLiteTransaction) Commit() error {
	return t.tx.Commit()
}

// Rollback rolls back the transaction
func (t *SQLiteTransaction) Rollback() error {
	return t.tx.Rollback()
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

// BuildInsertQuery builds an INSERT query for SQLite
func (a *SQLiteAdapter) BuildInsertQuery(table string, columns []string) string {
	quoted := make([]string, len(columns))
	placeholders := make([]string, len(columns))

	for i, col := range columns {
		quoted[i] = a.QuoteIdentifier(col)
		placeholders[i] = "?"
	}

	return fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s)",
		a.QuoteIdentifier(table),
		strings.Join(quoted, ", "),
		strings.Join(placeholders, ", "),
	)
}

// BuildBulkInsertQuery builds a bulk INSERT query for SQLite
func (a *SQLiteAdapter) BuildBulkInsertQuery(table string, columns []string, rowCount int) string {
	quoted := make([]string, len(columns))
	for i, col := range columns {
		quoted[i] = a.QuoteIdentifier(col)
	}

	var values []string
	for i := 0; i < rowCount; i++ {
		rowPlaceholders := make([]string, len(columns))
		for j := range columns {
			rowPlaceholders[j] = "?"
		}
		values = append(values, "("+strings.Join(rowPlaceholders, ", ")+")")
	}

	return fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES %s",
		a.QuoteIdentifier(table),
		strings.Join(quoted, ", "),
		strings.Join(values, ", "),
	)
}

// TruncateTable truncates a table (SQLite uses DELETE)
func (a *SQLiteAdapter) TruncateTable(ctx context.Context, table string, cascade bool) error {
	// SQLite doesn't have TRUNCATE, use DELETE
	query := fmt.Sprintf("DELETE FROM %s", a.QuoteIdentifier(table))
	_, err := a.Execute(ctx, query)
	if err != nil {
		return err
	}

	// Reset autoincrement counter
	_, err = a.Execute(ctx, "DELETE FROM sqlite_sequence WHERE name = ?", table)
	return err
}

// DisableForeignKeyChecks disables FK checks
func (a *SQLiteAdapter) DisableForeignKeyChecks(ctx context.Context) error {
	_, err := a.Execute(ctx, "PRAGMA foreign_keys = OFF")
	return err
}

// EnableForeignKeyChecks enables FK checks
func (a *SQLiteAdapter) EnableForeignKeyChecks(ctx context.Context) error {
	_, err := a.Execute(ctx, "PRAGMA foreign_keys = ON")
	return err
}

// GetTableRowCount returns the number of rows in a table
func (a *SQLiteAdapter) GetTableRowCount(ctx context.Context, table string) (int64, error) {
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

// GetLastInsertID gets the last inserted rowid
func (a *SQLiteAdapter) GetLastInsertID(ctx context.Context) (int64, error) {
	rows, err := a.Query(ctx, "SELECT last_insert_rowid()")
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var id int64
	if rows.Next() {
		if err := rows.Scan(&id); err != nil {
			return 0, err
		}
	}

	return id, nil
}

// TableExists checks if a table exists
func (a *SQLiteAdapter) TableExists(ctx context.Context, table string) (bool, error) {
	query := "SELECT name FROM sqlite_master WHERE type='table' AND name=?"
	rows, err := a.Query(ctx, query, table)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	return rows.Next(), nil
}
