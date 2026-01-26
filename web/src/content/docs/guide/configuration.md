---
title: Configuration
description: Configure seedcli using the seedcli.yaml file.
sidebar:
  order: 1
---

seedcli uses a YAML configuration file (`seedcli.yaml`) for all settings.

## Creating the Config File

Run `seedcli init` to create a default configuration:

```bash
seedcli init
```

## Configuration Reference

### Database Settings

```yaml
database:
  # Adapter type: postgres, sqlite
  adapter: postgres
  
  # Connection via individual fields
  host: localhost
  port: 5432
  name: myapp
  user: postgres
  password: secret
  ssl_mode: disable
  
  # OR connection via URL (takes precedence)
  url: "postgresql://user:pass@localhost:5432/myapp?sslmode=disable"
```

#### PostgreSQL

```yaml
database:
  adapter: postgres
  host: localhost
  port: 5432
  name: myapp
  user: postgres
  password: your_password
  ssl_mode: disable  # disable, require, verify-ca, verify-full
```

#### SQLite

```yaml
database:
  adapter: sqlite
  path: ./myapp.db
```

### Seeding Settings

```yaml
seeding:
  # Default number of rows per table
  default_rows: 10
  
  # Rows per batch insert (performance tuning)
  batch_size: 100
  
  # How to handle constraint conflicts
  # Options: error, skip, update
  on_conflict: error
```

### Data Generation Settings

```yaml
data_generation:
  # Probability of generating NULL for nullable fields (0.0 - 1.0)
  null_probability: 0.3
  
  # Locale for generating localized data
  locale: en_US
```

### Logging Settings

```yaml
logging:
  # Log level: debug, info, warn, error
  level: info
  
  # Write logs to .logseed/ directory
  to_file: true
  
  # Show logs in console
  to_console: true
  
  # Include timestamps
  timestamp: true
  
  # Use JSON format for logs
  json: false
```

### Plugin Settings

```yaml
plugins:
  # Directory containing plugin files
  directory: ./plugins
  
  # List of enabled plugins
  enabled:
    - my-custom-plugin
```

### Table-Specific Settings

Override settings for specific tables:

```yaml
tables:
  users:
    # Override row count for this table
    rows: 50
    
    # Column-specific settings
    columns:
      email:
        # Use specific generator
        generator: email
        
        # Make unique
        unique: true
      
      role:
        # Use specific values
        values:
          - admin
          - user
          - guest
      
      status:
        # Use pattern-based generation
        pattern: "STATUS_{A-Z}{3}"
      
      age:
        # Set min/max range
        min: 18
        max: 65
```

## Complete Example

```yaml
# seedcli.yaml
database:
  adapter: postgres
  host: localhost
  port: 5432
  name: ecommerce
  user: postgres
  password: secret
  ssl_mode: disable

seeding:
  default_rows: 10
  batch_size: 100
  on_conflict: skip

data_generation:
  null_probability: 0.2
  locale: en_US

logging:
  level: info
  to_file: true
  to_console: true
  timestamp: true

tables:
  users:
    rows: 100
    columns:
      email:
        generator: email
        unique: true
      role:
        values: [admin, moderator, user]
      
  products:
    rows: 50
    columns:
      price:
        min: 9.99
        max: 999.99
      sku:
        pattern: "SKU-{A-Z}{2}-{0-9}{4}"
        unique: true
      
  orders:
    rows: 200
    columns:
      status:
        values: [pending, processing, shipped, delivered]
```

## Environment Variables

You can use environment variables in your config:

```yaml
database:
  adapter: postgres
  host: ${DB_HOST:-localhost}
  port: ${DB_PORT:-5432}
  name: ${DB_NAME}
  user: ${DB_USER}
  password: ${DB_PASSWORD}
```

## Config File Location

seedcli looks for `seedcli.yaml` in the following order:

1. Current working directory
2. `$HOME/.config/seedcli/`
3. `/etc/seedcli/`

## Command-Line Overrides

Most config options can be overridden via command-line flags:

```bash
# Override database URL
seedcli seed --all --db-url "postgresql://localhost/other_db"

# Override row count
seedcli seed --all -n 500

# Override batch size
seedcli seed --all --batch-size 1000
```
