---
title: Quick Start
description: Get up and running with seedcli in minutes.
sidebar:
  order: 2
---

This guide walks you through your first steps with seedcli.

## Basic Workflow

1. **Initialize** your project with `seedcli init`
2. **Configure** your database connection in `seedcli.yaml`
3. **List tables** to see what's available
4. **Preview data** to see what will be generated
5. **Seed tables** with fake data

## Step 1: Initialize Your Project

```bash
seedcli init
```

This creates:
- `seedcli.yaml` - Configuration file
- `.logseed/` - Directory for log files

## Step 2: Configure Your Database

Edit `seedcli.yaml`:

### PostgreSQL

```yaml
database:
  adapter: postgres
  host: localhost
  port: 5432
  name: myapp
  user: postgres
  password: your_password
  ssl_mode: disable
```

### SQLite

```yaml
database:
  adapter: sqlite
  path: ./myapp.db
```

### Using Connection URL

```yaml
database:
  adapter: postgres
  url: "postgresql://user:pass@localhost:5432/myapp?sslmode=disable"
```

## Step 3: List Tables

```bash
seedcli list
```

**Example output:**

```
📋 Tables in database

┌─────────────┬─────────┬──────────────┐
│ Table       │ Columns │ Dependencies │
├─────────────┼─────────┼──────────────┤
│ users       │ 6       │ 0            │
│ products    │ 5       │ 0            │
│ orders      │ 4       │ 2            │
│ order_items │ 3       │ 2            │
└─────────────┴─────────┴──────────────┘
```

## Step 4: Preview Generated Data

Before inserting data, preview what will be generated:

```bash
seedcli preview -t users -n 3
```

**Example output:**

```
📋 Preview: users (3 rows)

┌─────┬──────────────┬─────────────────────────────┬────────────┐
│ id  │ username     │ email                       │ created_at │
├─────┼──────────────┼─────────────────────────────┼────────────┤
│ 1   │ johndoe42    │ john.doe@example.com        │ 2025-03-15 │
│ 2   │ janesmit     │ jane.smith@company.org      │ 2025-06-22 │
│ 3   │ mikebrown    │ mike.brown@startup.io       │ 2025-09-01 │
└─────┴──────────────┴─────────────────────────────┴────────────┘
```

## Step 5: Seed Your Database

### Seed a single table

```bash
seedcli seed -t users -n 50
```

### Seed multiple tables

```bash
seedcli seed -t users -t products -t orders -n 100
```

### Seed all tables

```bash
seedcli seed --all -n 100
```

**Example output:**

```
📊 Seeding Results
──────────────────────────────────────────────────────
✅ users: 100 rows (0.45s)
✅ products: 100 rows (0.32s)
✅ orders: 100 rows (0.28s)
✅ order_items: 100 rows (0.21s)

✓ Seeding completed successfully!
```

## Useful Options

### Reproducible Data

Use the `--seed` flag for deterministic data:

```bash
seedcli seed --all -n 100 --seed 42
```

Running this multiple times generates identical data.

### Dry Run

Test without inserting:

```bash
seedcli seed --all --dry-run
```

### Skip Errors

Continue seeding even if some tables fail:

```bash
seedcli seed --all --skip-errors
```

## Next Steps

- Learn about [Configuration](/guide/configuration) options
- Explore [CLI Commands](/reference/cli) in detail
- Understand the [Architecture](/guide/architecture)
