---
title: Examples
description: Common use cases and examples for seedcli.
sidebar:
  order: 3
---

## Basic Workflows

### Development Database Setup

```bash
# Create and seed a development database
createdb myapp_dev
seedcli init
# Edit seedcli.yaml with your connection details
seedcli seed --all -n 50
```

### Testing Pipeline

```bash
# Reproducible test data in CI/CD
seedcli seed --all -n 100 --seed 42
```

### Quick Prototyping

```bash
# Connect and seed in one command
seedcli seed --all -n 10 --db-url "postgresql://localhost/prototype"
```

---

## Database-Specific Examples

### PostgreSQL

#### Local Development

```yaml
# seedcli.yaml
database:
  adapter: postgres
  host: localhost
  port: 5432
  name: myapp_dev
  user: postgres
  password: postgres
  ssl_mode: disable
```

```bash
seedcli seed --all -n 100
```

#### Docker PostgreSQL

```yaml
database:
  adapter: postgres
  url: "postgresql://postgres:postgres@localhost:5432/myapp?sslmode=disable"
```

#### Remote Database

```yaml
database:
  adapter: postgres
  host: db.example.com
  port: 5432
  name: staging
  user: app_user
  password: ${DB_PASSWORD}
  ssl_mode: require
```

### SQLite

#### New Database

```yaml
database:
  adapter: sqlite
  path: ./myapp.db
```

```bash
seedcli seed --all -n 50
```

#### Existing Database

```yaml
database:
  adapter: sqlite
  path: /path/to/existing.db
```

---

## E-commerce Example

### Schema

```sql
-- PostgreSQL schema
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    username VARCHAR(50) UNIQUE NOT NULL,
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    price DECIMAL(10,2) NOT NULL,
    sku VARCHAR(50) UNIQUE NOT NULL,
    category VARCHAR(50),
    in_stock BOOLEAN DEFAULT true
);

CREATE TABLE orders (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id),
    status VARCHAR(20) DEFAULT 'pending',
    total DECIMAL(10,2),
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE order_items (
    id SERIAL PRIMARY KEY,
    order_id INTEGER REFERENCES orders(id),
    product_id INTEGER REFERENCES products(id),
    quantity INTEGER NOT NULL,
    price DECIMAL(10,2) NOT NULL
);
```

### Configuration

```yaml
# seedcli.yaml
database:
  adapter: postgres
  host: localhost
  name: ecommerce
  user: postgres
  password: secret

seeding:
  default_rows: 10
  batch_size: 100

tables:
  users:
    rows: 100
    columns:
      email:
        unique: true
      username:
        unique: true
  
  products:
    rows: 50
    columns:
      price:
        min: 9.99
        max: 499.99
      sku:
        pattern: "SKU-{A-Z}{2}-{0-9}{4}"
        unique: true
      category:
        values:
          - electronics
          - clothing
          - home
          - garden
          - sports
  
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
        min: 25.00
        max: 2500.00
  
  order_items:
    rows: 500
    columns:
      quantity:
        min: 1
        max: 10
```

### Seeding

```bash
# Preview data first
seedcli preview -t users -t products -n 5

# Seed all tables
seedcli seed --all

# Result:
# ✅ users: 100 rows (0.45s)
# ✅ products: 50 rows (0.32s)
# ✅ orders: 200 rows (0.28s)
# ✅ order_items: 500 rows (0.21s)
```

---

## CI/CD Integration

### GitHub Actions

```yaml
# .github/workflows/test.yml
name: Test

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    
    services:
      postgres:
        image: postgres:15
        env:
          POSTGRES_PASSWORD: postgres
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
        ports:
          - 5432:5432
    
    steps:
      - uses: actions/checkout@v4
      
      - uses: actions/setup-go@v5
        with:
          go-version: '1.21'
      
      - name: Install seedcli
        run: go install github.com/kiridharan/seedcli@latest
      
      - name: Setup database
        run: |
          PGPASSWORD=postgres psql -h localhost -U postgres -c "CREATE DATABASE test_db"
          PGPASSWORD=postgres psql -h localhost -U postgres -d test_db -f schema.sql
      
      - name: Seed database
        run: |
          seedcli seed --all -n 100 --seed 42 \
            --db-url "postgresql://postgres:postgres@localhost:5432/test_db?sslmode=disable"
      
      - name: Run tests
        run: go test ./...
```

### Docker Compose

```yaml
# docker-compose.yml
version: '3.8'

services:
  db:
    image: postgres:15
    environment:
      POSTGRES_DB: myapp
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
    volumes:
      - ./schema.sql:/docker-entrypoint-initdb.d/schema.sql
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres"]
      interval: 5s
      timeout: 5s
      retries: 5

  seed:
    image: golang:1.21
    depends_on:
      db:
        condition: service_healthy
    volumes:
      - ./seedcli.yaml:/app/seedcli.yaml
    working_dir: /app
    command: >
      sh -c "go install github.com/kiridharan/seedcli@latest &&
             seedcli seed --all -n 100"
    environment:
      DB_HOST: db
```

```yaml
# seedcli.yaml for Docker
database:
  adapter: postgres
  host: ${DB_HOST:-localhost}
  port: 5432
  name: myapp
  user: postgres
  password: postgres
```

---

## Performance Tuning

### Large Datasets

```bash
# Seed 100,000 rows with large batches
seedcli seed --all -n 100000 --batch-size 5000
```

### Disable Foreign Keys

For faster seeding (use with caution):

```bash
seedcli seed --all -n 50000 --disable-fk --batch-size 2000
```

### Truncate Before Seeding

```bash
# Clear existing data first
seedcli seed --all -n 1000 --truncate
```

---

## Troubleshooting

### Connection Issues

```bash
# Test connection
seedcli list --db-url "postgresql://localhost/mydb"

# Check logs
cat .logseed/seedcli-*.log | tail -50
```

### Constraint Violations

```bash
# Skip errors and continue
seedcli seed --all --skip-errors

# Or use dry-run to debug
seedcli seed --all --dry-run
```

### Foreign Key Errors

```bash
# Disable FK checks temporarily
seedcli seed --all --disable-fk
```
