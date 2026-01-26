// Package adapter provides an adapter factory
package adapter

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/kiridharan/seedcli/pkg/core"
)

// Factory creates database adapters
type Factory struct {
	adapters map[core.Dialect]func() core.Adapter
}

// NewFactory creates a new adapter factory with built-in adapters
func NewFactory() *Factory {
	f := &Factory{
		adapters: make(map[core.Dialect]func() core.Adapter),
	}

	// Register built-in adapters
	f.Register(core.DialectPostgres, func() core.Adapter {
		return NewPostgresAdapter()
	})
	f.Register(core.DialectSQLite, func() core.Adapter {
		return NewSQLiteAdapter()
	})

	return f
}

// Register registers an adapter factory function
func (f *Factory) Register(dialect core.Dialect, creator func() core.Adapter) {
	f.adapters[dialect] = creator
}

// Create creates an adapter for the given dialect
func (f *Factory) Create(dialect core.Dialect) (core.Adapter, error) {
	creator, ok := f.adapters[dialect]
	if !ok {
		return nil, fmt.Errorf("unsupported database dialect: %s", dialect)
	}
	return creator(), nil
}

// CreateAndConnect creates an adapter and connects to the database
func (f *Factory) CreateAndConnect(ctx context.Context, dsn string) (core.Adapter, error) {
	dialect, err := DetectDialect(dsn)
	if err != nil {
		return nil, err
	}

	adapter, err := f.Create(dialect)
	if err != nil {
		return nil, err
	}

	if err := adapter.Connect(ctx, dsn); err != nil {
		return nil, err
	}

	return adapter, nil
}

// ListDialects returns all registered dialects
func (f *Factory) ListDialects() []core.Dialect {
	dialects := make([]core.Dialect, 0, len(f.adapters))
	for d := range f.adapters {
		dialects = append(dialects, d)
	}
	return dialects
}

// DetectDialect detects the database dialect from a DSN
func DetectDialect(dsn string) (core.Dialect, error) {
	// Handle sqlite:// URLs
	if strings.HasPrefix(dsn, "sqlite://") || strings.HasPrefix(dsn, "sqlite3://") {
		return core.DialectSQLite, nil
	}

	// Handle file paths (SQLite)
	if strings.HasSuffix(dsn, ".db") || strings.HasSuffix(dsn, ".sqlite") || strings.HasSuffix(dsn, ".sqlite3") {
		return core.DialectSQLite, nil
	}

	// Handle :memory: (SQLite)
	if dsn == ":memory:" || strings.Contains(dsn, ":memory:") {
		return core.DialectSQLite, nil
	}

	// Parse URL
	u, err := url.Parse(dsn)
	if err != nil {
		return "", fmt.Errorf("invalid database URL: %w", err)
	}

	switch u.Scheme {
	case "postgres", "postgresql":
		return core.DialectPostgres, nil
	case "sqlite", "sqlite3":
		return core.DialectSQLite, nil
	case "mysql":
		return core.DialectMySQL, nil
	case "mongodb", "mongodb+srv":
		return core.DialectMongo, nil
	default:
		return "", fmt.Errorf("unsupported database scheme: %s", u.Scheme)
	}
}

// ValidateDSN validates a database connection string
func ValidateDSN(dsn string) error {
	if dsn == "" {
		return fmt.Errorf("database connection string is required")
	}

	_, err := DetectDialect(dsn)
	return err
}

// Global factory instance
var defaultFactory = NewFactory()

// GetFactory returns the default adapter factory
func GetFactory() *Factory {
	return defaultFactory
}

// Create creates an adapter using the default factory
func Create(dialect core.Dialect) (core.Adapter, error) {
	return defaultFactory.Create(dialect)
}

// CreateAndConnect creates and connects an adapter using the default factory
func CreateAndConnect(ctx context.Context, dsn string) (core.Adapter, error) {
	return defaultFactory.CreateAndConnect(ctx, dsn)
}
