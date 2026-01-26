---
title: Configuration Reference
description: Complete reference for seedcli.yaml configuration options.
sidebar:
  order: 2
---

## Full Configuration Example

```yaml
# seedcli.yaml - Complete configuration reference

# =============================================================================
# DATABASE CONNECTION
# =============================================================================
database:
  # Adapter type: postgres, sqlite
  adapter: postgres
  
  # Individual connection fields
  host: localhost
  port: 5432
  name: myapp
  user: postgres
  password: secret
  ssl_mode: disable  # disable, require, verify-ca, verify-full
  
  # OR use connection URL (takes precedence over individual fields)
  # url: "postgresql://user:pass@localhost:5432/myapp?sslmode=disable"
  
  # For SQLite:
  # adapter: sqlite
  # path: ./myapp.db

# =============================================================================
# SEEDING SETTINGS
# =============================================================================
seeding:
  # Default number of rows to generate per table
  default_rows: 10
  
  # Number of rows per batch insert (affects performance)
  batch_size: 100
  
  # Conflict handling strategy: error, skip, update
  on_conflict: error

# =============================================================================
# DATA GENERATION SETTINGS
# =============================================================================
data_generation:
  # Probability (0.0 - 1.0) of generating NULL for nullable columns
  null_probability: 0.3
  
  # Locale for generating localized data (names, addresses, etc.)
  locale: en_US

# =============================================================================
# LOGGING SETTINGS
# =============================================================================
logging:
  # Log level: debug, info, warn, error
  level: info
  
  # Write logs to .logseed/ directory
  to_file: true
  
  # Show logs in console
  to_console: true
  
  # Include timestamps in log messages
  timestamp: true
  
  # Use JSON format for log output
  json: false

# =============================================================================
# PLUGIN SETTINGS
# =============================================================================
plugins:
  # Directory containing plugin files
  directory: ./plugins
  
  # List of enabled plugins
  enabled:
    - my-custom-plugin
  
  # Plugin-specific configuration
  my-custom-plugin:
    option1: value1
    option2: value2

# =============================================================================
# TABLE-SPECIFIC SETTINGS
# =============================================================================
tables:
  # Override settings for specific tables
  users:
    # Custom row count for this table
    rows: 100
    
    # Column-specific configuration
    columns:
      email:
        # Force specific generator
        generator: email
        # Ensure uniqueness
        unique: true
      
      role:
        # Use specific values only
        values:
          - admin
          - moderator
          - user
          - guest
      
      username:
        # Pattern-based generation
        pattern: "user_{a-z}{5}"
        unique: true
      
      age:
        # Numeric range
        min: 18
        max: 65
      
      bio:
        # Set as nullable (override NOT NULL)
        nullable: true
      
      status:
        # Default value
        default: active
  
  products:
    rows: 50
    columns:
      price:
        min: 0.99
        max: 999.99
      
      sku:
        pattern: "SKU-{A-Z}{2}-{0-9}{4}"
        unique: true
      
      category:
        values:
          - electronics
          - clothing
          - home
          - garden
  
  orders:
    rows: 200
    columns:
      status:
        values:
          - pending
          - processing
          - shipped
          - delivered
          - cancelled
      
      total:
        min: 10.00
        max: 5000.00
```

## Configuration Sections

### database

| Field | Type | Description | Default |
|-------|------|-------------|---------|
| `adapter` | string | Database type (`postgres`, `sqlite`) | required |
| `host` | string | Database host | `localhost` |
| `port` | int | Database port | `5432` |
| `name` | string | Database name | required |
| `user` | string | Database user | - |
| `password` | string | Database password | - |
| `ssl_mode` | string | SSL mode (PostgreSQL) | `disable` |
| `url` | string | Full connection URL | - |
| `path` | string | Database file path (SQLite) | - |

### seeding

| Field | Type | Description | Default |
|-------|------|-------------|---------|
| `default_rows` | int | Default rows per table | `10` |
| `batch_size` | int | Rows per batch insert | `100` |
| `on_conflict` | string | Conflict strategy | `error` |

### data_generation

| Field | Type | Description | Default |
|-------|------|-------------|---------|
| `null_probability` | float | NULL probability for nullable columns | `0.3` |
| `locale` | string | Locale for localized data | `en_US` |

### logging

| Field | Type | Description | Default |
|-------|------|-------------|---------|
| `level` | string | Log level | `info` |
| `to_file` | bool | Write to log files | `true` |
| `to_console` | bool | Output to console | `true` |
| `timestamp` | bool | Include timestamps | `true` |
| `json` | bool | Use JSON format | `false` |

### plugins

| Field | Type | Description | Default |
|-------|------|-------------|---------|
| `directory` | string | Plugin directory | `./plugins` |
| `enabled` | []string | Enabled plugin names | `[]` |

### tables.[name].columns.[name]

| Field | Type | Description |
|-------|------|-------------|
| `generator` | string | Generator to use |
| `values` | []any | Allowed values (ENUM-like) |
| `pattern` | string | Pattern for generation |
| `unique` | bool | Ensure uniqueness |
| `nullable` | bool | Allow NULL values |
| `min` | number | Minimum value |
| `max` | number | Maximum value |
| `default` | any | Default value |

## Environment Variables

Use environment variables with `${VAR:-default}` syntax:

```yaml
database:
  host: ${DB_HOST:-localhost}
  port: ${DB_PORT:-5432}
  name: ${DB_NAME}
  user: ${DB_USER}
  password: ${DB_PASSWORD}
```

## Config File Locations

seedcli searches for `seedcli.yaml` in order:

1. Current working directory (`./seedcli.yaml`)
2. User config (`$HOME/.config/seedcli/seedcli.yaml`)
3. System config (`/etc/seedcli/seedcli.yaml`)
