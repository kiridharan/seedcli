// Package core provides the foundational interfaces for seedcli's extensible architecture.
// All components (adapters, engines, plugins) implement these interfaces.
package core

import (
	"context"
	"io"
	"time"
)

// =============================================================================
// DATABASE ADAPTER INTERFACE
// =============================================================================

// Adapter defines the contract for database connections.
// Implement this interface to add support for new databases.
type Adapter interface {
	// Connect establishes a connection to the database
	Connect(ctx context.Context, dsn string) error

	// Close terminates the database connection
	Close() error

	// Ping verifies the connection is alive
	Ping(ctx context.Context) error

	// Execute runs a query without returning rows
	Execute(ctx context.Context, query string, args ...interface{}) (Result, error)

	// Query runs a query and returns rows
	Query(ctx context.Context, query string, args ...interface{}) (Rows, error)

	// BeginTx starts a new transaction
	BeginTx(ctx context.Context) (Transaction, error)

	// Dialect returns the database dialect identifier
	Dialect() Dialect

	// QuoteIdentifier quotes an identifier (table/column name) for the dialect
	QuoteIdentifier(name string) string

	// Placeholder returns the placeholder syntax for the dialect (e.g., $1, ?)
	Placeholder(index int) string
}

// Dialect represents a database dialect
type Dialect string

const (
	DialectPostgres Dialect = "postgres"
	DialectSQLite   Dialect = "sqlite"
	DialectMySQL    Dialect = "mysql"
	DialectMongo    Dialect = "mongodb"
)

// Result represents the result of an Execute operation
type Result interface {
	LastInsertId() (int64, error)
	RowsAffected() (int64, error)
}

// Rows represents query result rows
type Rows interface {
	Next() bool
	Scan(dest ...interface{}) error
	Columns() ([]string, error)
	Close() error
	Err() error
}

// Transaction represents a database transaction
type Transaction interface {
	Execute(ctx context.Context, query string, args ...interface{}) (Result, error)
	Query(ctx context.Context, query string, args ...interface{}) (Rows, error)
	Commit() error
	Rollback() error
}

// =============================================================================
// SCHEMA ENGINE INTERFACE
// =============================================================================

// SchemaEngine introspects and manages database schemas.
// SQL and NoSQL databases have different implementations.
type SchemaEngine interface {
	// SetAdapter sets the database adapter to use
	SetAdapter(adapter Adapter)

	// ListCollections returns all tables/collections in the database
	ListCollections(ctx context.Context) ([]string, error)

	// IntrospectCollection analyzes a table/collection structure
	IntrospectCollection(ctx context.Context, name string) (*Collection, error)

	// IntrospectAll analyzes all collections
	IntrospectAll(ctx context.Context) ([]*Collection, error)

	// GetDependencyOrder returns collections in dependency order (topological sort)
	GetDependencyOrder(collections []*Collection) ([]*Collection, error)

	// ValidateSchema checks if the schema is valid for seeding
	ValidateSchema(collections []*Collection) []SchemaError
}

// Collection represents a database table or document collection
type Collection struct {
	Name        string
	Type        CollectionType
	Fields      []*Field
	PrimaryKey  []string
	ForeignKeys []*ForeignKey
	Indexes     []*Index
	Constraints []*Constraint
	Metadata    map[string]interface{}
}

// CollectionType indicates the type of collection
type CollectionType string

const (
	CollectionTypeTable    CollectionType = "table"
	CollectionTypeDocument CollectionType = "document"
	CollectionTypeView     CollectionType = "view"
)

// Field represents a column or document field
type Field struct {
	Name         string
	Type         FieldType
	RawType      string // Original database type
	IsNullable   bool
	IsPrimaryKey bool
	IsAutoIncr   bool
	IsUnique     bool
	Default      interface{}
	MaxLength    int64
	Precision    int
	Scale        int
	EnumValues   []string
	Metadata     map[string]interface{}
}

// FieldType represents abstract field types
type FieldType string

const (
	FieldTypeString    FieldType = "string"
	FieldTypeInt       FieldType = "int"
	FieldTypeFloat     FieldType = "float"
	FieldTypeBool      FieldType = "bool"
	FieldTypeDate      FieldType = "date"
	FieldTypeTime      FieldType = "time"
	FieldTypeDateTime  FieldType = "datetime"
	FieldTypeTimestamp FieldType = "timestamp"
	FieldTypeUUID      FieldType = "uuid"
	FieldTypeJSON      FieldType = "json"
	FieldTypeBinary    FieldType = "binary"
	FieldTypeEnum      FieldType = "enum"
	FieldTypeArray     FieldType = "array"
	FieldTypeObject    FieldType = "object"
	FieldTypeUnknown   FieldType = "unknown"
)

// ForeignKey represents a foreign key relationship
type ForeignKey struct {
	Name             string
	ColumnName       string
	ReferencedTable  string
	ReferencedColumn string
	OnDelete         string
	OnUpdate         string
}

// Index represents a database index
type Index struct {
	Name      string
	Columns   []string
	IsUnique  bool
	IsPrimary bool
	Type      string
}

// Constraint represents a database constraint
type Constraint struct {
	Name       string
	Type       ConstraintType
	Columns    []string
	Expression string
}

// ConstraintType represents types of constraints
type ConstraintType string

const (
	ConstraintTypeCheck       ConstraintType = "check"
	ConstraintTypeUnique      ConstraintType = "unique"
	ConstraintTypePrimaryKey  ConstraintType = "primary_key"
	ConstraintTypeForeignKey  ConstraintType = "foreign_key"
	ConstraintTypeNotNull     ConstraintType = "not_null"
	ConstraintTypeDefault     ConstraintType = "default"
)

// SchemaError represents a schema validation error
type SchemaError struct {
	Collection string
	Field      string
	Message    string
	Severity   ErrorSeverity
}

// ErrorSeverity indicates how severe an error is
type ErrorSeverity string

const (
	SeverityWarning  ErrorSeverity = "warning"
	SeverityError    ErrorSeverity = "error"
	SeverityCritical ErrorSeverity = "critical"
)

// =============================================================================
// DATA ENGINE INTERFACE
// =============================================================================

// DataEngine generates fake data for seeding.
// Composed of multiple generators and validators.
type DataEngine interface {
	// SetSeed sets the random seed for reproducible generation
	SetSeed(seed int64)

	// GenerateRow generates a single row of data for a collection
	GenerateRow(ctx context.Context, collection *Collection) (map[string]interface{}, error)

	// GenerateRows generates multiple rows
	GenerateRows(ctx context.Context, collection *Collection, count int) ([]map[string]interface{}, error)

	// RegisterGenerator registers a custom generator for a field type or name pattern
	RegisterGenerator(name string, gen Generator)

	// RegisterValidator registers a custom validator
	RegisterValidator(name string, val Validator)

	// SetReferenceData sets foreign key reference data
	SetReferenceData(tableName string, column string, values []interface{})

	// GetReferenceData gets inserted primary keys for FK references
	GetReferenceData(tableName string) []interface{}
}

// Generator generates fake values for a specific type or pattern
type Generator interface {
	// Generate produces a fake value
	Generate(ctx context.Context, field *Field, opts GeneratorOptions) (interface{}, error)

	// Supports returns true if this generator supports the given field
	Supports(field *Field) bool

	// Priority returns the generator priority (higher = checked first)
	Priority() int
}

// GeneratorOptions provides context for value generation
type GeneratorOptions struct {
	Seed         int64
	IsUnique     bool
	UsedValues   map[interface{}]bool
	FKValues     []interface{}
	CustomParams map[string]interface{}
}

// Validator validates generated data
type Validator interface {
	// Validate checks if the value is valid for the field
	Validate(field *Field, value interface{}) error

	// Name returns the validator name
	Name() string
}

// =============================================================================
// SEEDER INTERFACE
// =============================================================================

// Seeder orchestrates the seeding process
type Seeder interface {
	// Seed seeds the specified collections
	Seed(ctx context.Context, collections []string, opts SeedOptions) (*SeedResult, error)

	// SeedAll seeds all collections in the database
	SeedAll(ctx context.Context, opts SeedOptions) (*SeedResult, error)

	// Preview generates sample data without inserting
	Preview(ctx context.Context, collections []string, opts SeedOptions) (*PreviewResult, error)
}

// SeedOptions configures the seeding process
type SeedOptions struct {
	RowsPerCollection int
	BatchSize         int
	Seed              int64
	DryRun            bool
	SkipErrors        bool
	Truncate          bool
	DisableFK         bool
	OnConflict        ConflictStrategy
}

// ConflictStrategy defines how to handle conflicts
type ConflictStrategy string

const (
	ConflictSkip    ConflictStrategy = "skip"
	ConflictUpdate  ConflictStrategy = "update"
	ConflictReplace ConflictStrategy = "replace"
	ConflictError   ConflictStrategy = "error"
)

// SeedResult contains seeding results
type SeedResult struct {
	Collections []CollectionResult
	TotalRows   int64
	Duration    time.Duration
	Errors      []error
}

// CollectionResult contains results for a single collection
type CollectionResult struct {
	Name         string
	RowsInserted int64
	Duration     time.Duration
	Error        error
}

// PreviewResult contains preview data
type PreviewResult struct {
	Collections map[string][]map[string]interface{}
}

// =============================================================================
// PLUGIN INTERFACE
// =============================================================================

// Plugin defines the contract for seedcli plugins
type Plugin interface {
	// Name returns the plugin name
	Name() string

	// Version returns the plugin version
	Version() string

	// Description returns a brief description
	Description() string

	// Init initializes the plugin with config
	Init(config map[string]interface{}) error

	// Type returns what the plugin provides
	Type() PluginType

	// Hooks returns lifecycle hooks
	Hooks() PluginHooks
}

// PluginType indicates what the plugin provides
type PluginType string

const (
	PluginTypeAdapter   PluginType = "adapter"
	PluginTypeGenerator PluginType = "generator"
	PluginTypeValidator PluginType = "validator"
	PluginTypeHook      PluginType = "hook"
)

// PluginHooks provides lifecycle hooks
type PluginHooks struct {
	BeforeSeed     func(ctx context.Context, collections []*Collection) error
	AfterSeed      func(ctx context.Context, result *SeedResult) error
	BeforeInsert   func(ctx context.Context, collection string, rows []map[string]interface{}) error
	AfterInsert    func(ctx context.Context, collection string, count int64) error
	OnError        func(ctx context.Context, err error) error
}

// AdapterPlugin provides a database adapter
type AdapterPlugin interface {
	Plugin
	Adapter() Adapter
}

// GeneratorPlugin provides custom generators
type GeneratorPlugin interface {
	Plugin
	Generators() []Generator
}

// ValidatorPlugin provides custom validators
type ValidatorPlugin interface {
	Plugin
	Validators() []Validator
}

// =============================================================================
// LOGGER INTERFACE
// =============================================================================

// Logger defines the logging interface
type Logger interface {
	// Log methods
	Debug(msg string, fields ...LogField)
	Info(msg string, fields ...LogField)
	Warn(msg string, fields ...LogField)
	Error(msg string, fields ...LogField)
	Fatal(msg string, fields ...LogField)

	// WithFields returns a logger with additional fields
	WithFields(fields ...LogField) Logger

	// WithContext returns a logger with context
	WithContext(ctx interface{}) Logger

	// SetLevel sets the logging level
	SetLevel(level LogLevel)

	// SetOutput sets the output writer
	SetOutput(w io.Writer)
}

// LogLevel represents logging levels
type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
	LogLevelFatal
)

// LogField is a key-value pair for structured logging
// (using Field name to avoid conflict with other Field type)
type LogField struct {
	Key   string
	Value interface{}
}

// Field alias for LogField in logger context
func F(key string, value interface{}) LogField {
	return LogField{Key: key, Value: value}
}

// =============================================================================
// REGISTRY INTERFACE
// =============================================================================

// Registry manages adapters, generators, validators, and plugins
type Registry interface {
	// Adapter management
	RegisterAdapter(name string, adapter Adapter)
	GetAdapter(name string) (Adapter, bool)
	ListAdapters() []string

	// Schema engine management
	RegisterSchemaEngine(name string, engine SchemaEngine)
	GetSchemaEngine(name string) (SchemaEngine, bool)
	ListSchemaEngines() []string

	// Generator management
	RegisterGenerator(name string, gen Generator)
	GetGenerator(name string) (Generator, bool)
	ListGenerators() []string

	// Validator management
	RegisterValidator(name string, val Validator)
	GetValidator(name string) (Validator, bool)
	ListValidators() []string

	// Plugin management
	RegisterPlugin(plugin Plugin) error
	GetPlugin(name string) (Plugin, bool)
	ListPlugins() []string
	LoadPlugins(dir string) error
}
