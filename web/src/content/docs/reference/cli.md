---
title: CLI Commands
description: Complete reference for all seedcli commands.
sidebar:
  order: 1
---

## seedcli init

Initialize a new seedcli project.

```bash
seedcli init [flags]
```

### Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--force` | `-f` | Overwrite existing config |

### Examples

```bash
# Initialize new project
seedcli init

# Force overwrite existing config
seedcli init --force
```

### Output

Creates:
- `seedcli.yaml` - Configuration file
- `.logseed/` - Log directory

---

## seedcli seed

Seed database tables with fake data.

```bash
seedcli seed [flags]
```

### Flags

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--table` | `-t` | Table(s) to seed (repeatable) | - |
| `--all` | `-a` | Seed all tables | `false` |
| `--rows` | `-n` | Rows per table | from config |
| `--batch-size` | - | Rows per batch insert | `100` |
| `--seed` | - | Random seed for reproducibility | current time |
| `--dry-run` | - | Preview without inserting | `false` |
| `--skip-errors` | - | Continue on errors | `false` |
| `--truncate` | - | Truncate tables first | `false` |
| `--disable-fk` | - | Disable foreign key checks | `false` |
| `--db-url` | - | Database URL (overrides config) | - |

### Examples

```bash
# Seed all tables
seedcli seed --all

# Seed specific tables
seedcli seed -t users -t orders

# Seed with custom row count
seedcli seed --all -n 100

# Reproducible seeding
seedcli seed --all --seed 42

# Dry run preview
seedcli seed --all --dry-run

# Large dataset with batching
seedcli seed --all -n 10000 --batch-size 1000

# Skip constraint errors
seedcli seed --all --skip-errors
```

---

## seedcli list

List tables in the database.

```bash
seedcli list [flags]
```

### Flags

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--db-url` | - | Database URL | - |
| `--format` | - | Output format (table, json) | `table` |

### Examples

```bash
# List all tables
seedcli list

# JSON output
seedcli list --format json

# With different database
seedcli list --db-url "postgresql://localhost/other_db"
```

### Output

```
📋 Tables in database

┌─────────────┬─────────┬──────────────┬─────────────────────┐
│ Table       │ Columns │ Dependencies │ Estimated Rows      │
├─────────────┼─────────┼──────────────┼─────────────────────┤
│ users       │ 6       │ 0            │ 0                   │
│ products    │ 5       │ 0            │ 0                   │
│ orders      │ 4       │ 2            │ 0                   │
│ order_items │ 3       │ 2            │ 0                   │
└─────────────┴─────────┴──────────────┴─────────────────────┘

Total: 4 tables
```

---

## seedcli preview

Preview generated data without inserting.

```bash
seedcli preview [flags]
```

### Flags

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--table` | `-t` | Table(s) to preview (repeatable) | - |
| `--rows` | `-n` | Number of sample rows | `3` |
| `--format` | - | Output format (table, json) | `table` |
| `--seed` | - | Random seed | current time |
| `--db-url` | - | Database URL | - |

### Examples

```bash
# Preview users table
seedcli preview -t users

# Preview multiple tables
seedcli preview -t users -t products

# More sample rows
seedcli preview -t users -n 10

# JSON output
seedcli preview -t users --format json

# Reproducible preview
seedcli preview -t users --seed 42
```

### Output

```
📋 Preview: users (3 rows)

┌─────┬──────────────┬─────────────────────────────┬────────────┐
│ id  │ username     │ email                       │ created_at │
├─────┼──────────────┼─────────────────────────────┼────────────┤
│ 1   │ johndoe42    │ john.doe@example.com        │ 2025-03-15 │
│ 2   │ janesmith    │ jane.smith@company.org      │ 2025-06-22 │
│ 3   │ mikebrown    │ mike.brown@startup.io       │ 2025-09-01 │
└─────┴──────────────┴─────────────────────────────┴────────────┘
```

---

## seedcli version

Display version information.

```bash
seedcli version
```

### Output

```
seedcli version 2.0.0
  Build date: 2026-01-26
  Go version: go1.21
  OS/Arch:    darwin/arm64

Supported adapters:
  • postgres  - PostgreSQL 12+
  • sqlite    - SQLite 3
```

---

## Global Behavior

### Configuration Loading

All commands automatically load `seedcli.yaml` from:

1. Current working directory
2. `$HOME/.config/seedcli/`
3. `/etc/seedcli/`

### Logging

All operations are logged to `.logseed/seedcli-YYYY-MM-DD.log`.

### Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Success |
| `1` | General error |
| `2` | Configuration error |
| `3` | Database connection error |
| `4` | Seeding error |
