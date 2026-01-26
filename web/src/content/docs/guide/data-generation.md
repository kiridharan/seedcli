---
title: Data Generation
description: How seedcli generates intelligent fake data.
sidebar:
  order: 3
---

seedcli uses intelligent heuristics to generate contextually appropriate fake data based on column names and types.

## How It Works

The Data Engine analyzes each column and applies generation strategies in this order:

1. **Custom Generators** - User-registered generators (highest priority)
2. **Column Name Heuristics** - Based on naming patterns (email, phone, etc.)
3. **Type-Based Generation** - Based on column data type
4. **Default Fallback** - Random strings or numbers

## Name-Based Heuristics

seedcli automatically detects common column naming patterns:

| Column Name Contains | Generated Data |
|---------------------|----------------|
| `email` | `john.doe@example.com` |
| `first_name`, `firstname` | `John` |
| `last_name`, `lastname` | `Doe` |
| `name` | `John Doe` |
| `username`, `user` | `johndoe42` |
| `phone` | `+1-555-123-4567` |
| `address` | `123 Main St, Anytown, ST 12345` |
| `city` | `New York` |
| `state` | `California` |
| `country` | `United States` |
| `zip`, `postal` | `12345` |
| `url`, `website` | `https://example.com` |
| `slug` | `my-awesome-post` |
| `title` | `Senior Software Engineer` |
| `company` | `Acme Corporation` |
| `description`, `bio` | `Lorem ipsum dolor sit amet...` |
| `uuid`, `guid` | `550e8400-e29b-41d4-a716-446655440000` |
| `price`, `amount` | `99.99` |
| `age` | `32` |
| `rating`, `score` | `4` (1-5 range) |
| `created_at`, `updated_at` | `2025-03-15 10:30:00` |

## Type-Based Generation

When name heuristics don't apply, seedcli generates based on column type:

| Column Type | Generated Data |
|-------------|----------------|
| `BOOLEAN` | `true` or `false` |
| `INTEGER`, `SERIAL` | Random integer |
| `FLOAT`, `DOUBLE`, `NUMERIC` | Random decimal |
| `VARCHAR`, `TEXT`, `CHAR` | Random sentence |
| `DATE` | Random date within 5 years |
| `TIMESTAMP`, `DATETIME` | Random datetime |
| `TIME` | `14:30:00` |
| `UUID` | Valid UUID v4 |
| `JSON`, `JSONB` | `{"key": "value"}` |
| `BYTEA`, `BLOB` | Random bytes |
| `ARRAY` | Array of appropriate type |

## ENUM Support

For ENUM columns, seedcli automatically selects from allowed values:

```sql
CREATE TABLE users (
  status user_status  -- ENUM('active', 'inactive', 'pending')
);
```

Generated values will be one of: `active`, `inactive`, or `pending`.

## Constraint Handling

### UNIQUE Constraints

seedcli tracks generated values to ensure uniqueness:

```sql
CREATE TABLE users (
  email VARCHAR(255) UNIQUE
);
```

Each generated email will be unique within the seeding session.

### NOT NULL

Non-nullable columns always receive values:

```sql
CREATE TABLE users (
  email VARCHAR(255) NOT NULL  -- Always gets a value
);
```

### Nullable Columns

Nullable columns have a configurable probability of being NULL (default 30%):

```yaml
data_generation:
  null_probability: 0.3  # 30% chance of NULL
```

## Foreign Key Handling

seedcli automatically handles foreign key relationships:

1. Tables are sorted by dependencies (topological sort)
2. Parent tables are seeded first
3. Child tables reference actual inserted IDs

```sql
CREATE TABLE orders (
  user_id INTEGER REFERENCES users(id)  -- Uses real user IDs
);
```

## Custom Configuration

Override generation for specific columns in `seedcli.yaml`:

```yaml
tables:
  users:
    columns:
      # Use specific values
      role:
        values:
          - admin
          - user
          - guest
      
      # Set numeric range
      age:
        min: 18
        max: 65
      
      # Use pattern
      employee_id:
        pattern: "EMP-{A-Z}{2}-{0-9}{4}"
        unique: true
      
      # Force specific generator
      avatar_url:
        generator: url
```

## Pattern Syntax

Use patterns for formatted strings:

| Pattern | Example Output |
|---------|----------------|
| `{A-Z}` | Single uppercase letter |
| `{a-z}` | Single lowercase letter |
| `{0-9}` | Single digit |
| `{A-Z}{3}` | Three uppercase letters |
| `SKU-{A-Z}{2}-{0-9}{4}` | `SKU-AB-1234` |
| `{a-z}{5}@example.com` | `abcde@example.com` |

## Reproducible Data

Use the `--seed` flag for deterministic generation:

```bash
seedcli seed --all --seed 42
```

Running with the same seed produces identical data every time.
