// Package config manages seedcli configuration
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	// ConfigFileName is the default config file name
	ConfigFileName = "seedcli.yaml"
	// LogDirName is the directory for logs
	LogDirName = ".logseed"
)

// Config represents the complete seedcli configuration
type Config struct {
	// Version of the config schema
	Version string `yaml:"version"`

	// Database connection configuration
	Database DatabaseConfig `yaml:"database"`

	// Seeding configuration
	Seeding SeedingConfig `yaml:"seeding"`

	// Data generation configuration
	DataGeneration DataGenerationConfig `yaml:"data_generation"`

	// Logging configuration
	Logging LoggingConfig `yaml:"logging"`

	// Plugin configuration
	Plugins PluginConfig `yaml:"plugins"`

	// Table-specific overrides
	Tables map[string]TableConfig `yaml:"tables,omitempty"`

	// Metadata
	Metadata ConfigMetadata `yaml:"_metadata,omitempty"`
}

// DatabaseConfig holds database connection settings
type DatabaseConfig struct {
	// Adapter type: postgres, sqlite, mysql, mongodb
	Adapter string `yaml:"adapter"`

	// Connection URL (DSN format)
	URL string `yaml:"url,omitempty"`

	// Individual connection parameters (alternative to URL)
	Host     string `yaml:"host,omitempty"`
	Port     int    `yaml:"port,omitempty"`
	User     string `yaml:"user,omitempty"`
	Password string `yaml:"password,omitempty"`
	DBName   string `yaml:"database,omitempty"`
	SSLMode  string `yaml:"ssl_mode,omitempty"`

	// Connection pool settings
	MaxOpenConns    int           `yaml:"max_open_conns,omitempty"`
	MaxIdleConns    int           `yaml:"max_idle_conns,omitempty"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime,omitempty"`

	// Schema (for postgres)
	Schema string `yaml:"schema,omitempty"`
}

// SeedingConfig holds seeding behavior settings
type SeedingConfig struct {
	// Default number of rows per table
	DefaultRows int `yaml:"default_rows"`

	// Batch size for inserts
	BatchSize int `yaml:"batch_size"`

	// Random seed for reproducible data (0 = random)
	Seed int64 `yaml:"seed,omitempty"`

	// Skip errors and continue
	SkipErrors bool `yaml:"skip_errors"`

	// Truncate tables before seeding
	Truncate bool `yaml:"truncate"`

	// Disable foreign key checks during seeding
	DisableForeignKeys bool `yaml:"disable_foreign_keys"`

	// Conflict handling strategy: skip, update, replace, error
	OnConflict string `yaml:"on_conflict"`

	// Tables to exclude from seeding
	Exclude []string `yaml:"exclude,omitempty"`

	// Tables to include (if set, only these are seeded)
	Include []string `yaml:"include,omitempty"`
}

// DataGenerationConfig configures the data generation engine
type DataGenerationConfig struct {
	// Locale for fake data (e.g., en_US, de_DE)
	Locale string `yaml:"locale"`

	// Null probability for nullable fields (0.0 - 1.0)
	NullProbability float64 `yaml:"null_probability"`

	// Custom generators for field patterns
	FieldPatterns map[string]FieldPattern `yaml:"field_patterns,omitempty"`

	// Type mappings for custom types
	TypeMappings map[string]string `yaml:"type_mappings,omitempty"`
}

// FieldPattern defines a custom generator for matching fields
type FieldPattern struct {
	// Pattern to match (supports glob: *email*, user_*)
	Pattern string `yaml:"pattern"`

	// Generator to use
	Generator string `yaml:"generator"`

	// Generator parameters
	Params map[string]interface{} `yaml:"params,omitempty"`
}

// LoggingConfig configures logging behavior
type LoggingConfig struct {
	// Log level: debug, info, warn, error
	Level string `yaml:"level"`

	// Log to file in .logseed directory
	ToFile bool `yaml:"to_file"`

	// Log to console
	ToConsole bool `yaml:"to_console"`

	// JSON format for logs
	JSON bool `yaml:"json"`

	// Include timestamp
	Timestamp bool `yaml:"timestamp"`

	// Maximum log file size in MB
	MaxSize int `yaml:"max_size"`

	// Number of log files to keep
	MaxBackups int `yaml:"max_backups"`
}

// PluginConfig configures plugins
type PluginConfig struct {
	// Directory containing plugins
	Directory string `yaml:"directory,omitempty"`

	// Enabled plugins
	Enabled []string `yaml:"enabled,omitempty"`

	// Plugin-specific configuration
	Settings map[string]map[string]interface{} `yaml:"settings,omitempty"`
}

// TableConfig provides table-specific overrides
type TableConfig struct {
	// Number of rows for this table
	Rows int `yaml:"rows,omitempty"`

	// Column-specific configuration
	Columns map[string]ColumnConfig `yaml:"columns,omitempty"`

	// Skip this table
	Skip bool `yaml:"skip,omitempty"`

	// Custom SQL to run before seeding
	BeforeSQL string `yaml:"before_sql,omitempty"`

	// Custom SQL to run after seeding
	AfterSQL string `yaml:"after_sql,omitempty"`
}

// ColumnConfig provides column-specific overrides
type ColumnConfig struct {
	// Generator to use
	Generator string `yaml:"generator,omitempty"`

	// Fixed value
	Value interface{} `yaml:"value,omitempty"`

	// Values to pick from
	Values []interface{} `yaml:"values,omitempty"`

	// Skip this column (use default)
	Skip bool `yaml:"skip,omitempty"`

	// Generator parameters
	Params map[string]interface{} `yaml:"params,omitempty"`
}

// ConfigMetadata stores config file metadata
type ConfigMetadata struct {
	CreatedAt  time.Time `yaml:"created_at"`
	UpdatedAt  time.Time `yaml:"updated_at"`
	CreatedBy  string    `yaml:"created_by,omitempty"`
	SeedCLIVer string    `yaml:"seedcli_version"`
}

// DefaultConfig returns a new config with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		Version: "1.0",
		Database: DatabaseConfig{
			Adapter:         "postgres",
			MaxOpenConns:    25,
			MaxIdleConns:    5,
			ConnMaxLifetime: 5 * time.Minute,
			Schema:          "public",
		},
		Seeding: SeedingConfig{
			DefaultRows: 10,
			BatchSize:   100,
			SkipErrors:  false,
			Truncate:    false,
			OnConflict:  "error",
		},
		DataGeneration: DataGenerationConfig{
			Locale:          "en_US",
			NullProbability: 0.3,
		},
		Logging: LoggingConfig{
			Level:      "info",
			ToFile:     true,
			ToConsole:  true,
			JSON:       false,
			Timestamp:  true,
			MaxSize:    10,
			MaxBackups: 5,
		},
		Plugins: PluginConfig{
			Directory: ".seedcli/plugins",
		},
		Metadata: ConfigMetadata{
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
			SeedCLIVer: "2.0.0",
		},
	}
}

// Load loads configuration from a file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &cfg, nil
}

// LoadFromDir loads configuration from the given directory
func LoadFromDir(dir string) (*Config, error) {
	return Load(filepath.Join(dir, ConfigFileName))
}

// LoadFromCWD loads configuration from the current working directory
func LoadFromCWD() (*Config, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	return LoadFromDir(cwd)
}

// Save saves configuration to a file
func (c *Config) Save(path string) error {
	c.Metadata.UpdatedAt = time.Now()

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// SaveToCWD saves configuration to the current working directory
func (c *Config) SaveToCWD() error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	return c.Save(filepath.Join(cwd, ConfigFileName))
}

// GetDSN builds a connection string from config
func (c *Config) GetDSN() string {
	if c.Database.URL != "" {
		return c.Database.URL
	}

	switch c.Database.Adapter {
	case "postgres", "postgresql":
		sslMode := c.Database.SSLMode
		if sslMode == "" {
			sslMode = "disable"
		}
		return fmt.Sprintf(
			"postgres://%s:%s@%s:%d/%s?sslmode=%s",
			c.Database.User,
			c.Database.Password,
			c.Database.Host,
			c.Database.Port,
			c.Database.DBName,
			sslMode,
		)
	case "sqlite", "sqlite3":
		return fmt.Sprintf("sqlite://%s", c.Database.DBName)
	case "mysql":
		return fmt.Sprintf(
			"%s:%s@tcp(%s:%d)/%s",
			c.Database.User,
			c.Database.Password,
			c.Database.Host,
			c.Database.Port,
			c.Database.DBName,
		)
	default:
		return c.Database.URL
	}
}

// Validate validates the configuration
func (c *Config) Validate() []error {
	var errs []error

	if c.Database.Adapter == "" {
		errs = append(errs, fmt.Errorf("database.adapter is required"))
	}

	if c.Database.URL == "" && c.Database.Host == "" && c.Database.DBName == "" {
		errs = append(errs, fmt.Errorf("database connection details required (url or host/database)"))
	}

	if c.Seeding.DefaultRows < 1 {
		errs = append(errs, fmt.Errorf("seeding.default_rows must be at least 1"))
	}

	if c.Seeding.BatchSize < 1 {
		errs = append(errs, fmt.Errorf("seeding.batch_size must be at least 1"))
	}

	if c.DataGeneration.NullProbability < 0 || c.DataGeneration.NullProbability > 1 {
		errs = append(errs, fmt.Errorf("data_generation.null_probability must be between 0 and 1"))
	}

	return errs
}

// EnsureLogDir ensures the .logseed directory exists
func EnsureLogDir() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	logDir := filepath.Join(cwd, LogDirName)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create log directory: %w", err)
	}

	// Create .gitignore in .logseed
	gitignore := filepath.Join(logDir, ".gitignore")
	if _, err := os.Stat(gitignore); os.IsNotExist(err) {
		os.WriteFile(gitignore, []byte("*\n!.gitignore\n"), 0644)
	}

	return logDir, nil
}

// GetLogPath returns the path to the log file
func GetLogPath() (string, error) {
	logDir, err := EnsureLogDir()
	if err != nil {
		return "", err
	}

	timestamp := time.Now().Format("2006-01-02")
	return filepath.Join(logDir, fmt.Sprintf("seedcli-%s.log", timestamp)), nil
}
