---
title: Architecture
description: Understanding seedcli's extensible architecture.
sidebar:
  order: 2
---

seedcli v2 is designed with **extensibility** as the core principle. Every major component is defined by interfaces, allowing easy replacement, extension, and community contributions.

## System Overview

```
┌─────────────────────────────────────────────────────────────────────┐
│                           CLI Layer                                  │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐   │
│  │  init   │  │  seed   │  │  list   │  │ preview │  │ version │   │
│  └─────────┘  └─────────┘  └─────────┘  └─────────┘  └─────────┘   │
└─────────────────────────────────────────────────────────────────────┘
                                  │
                                  ▼
┌─────────────────────────────────────────────────────────────────────┐
│                          Core Seeder                                 │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │                     Seeder Orchestrator                       │   │
│  │  • Coordinates schema introspection                          │   │
│  │  • Manages data generation                                    │   │
│  │  • Handles batch insertion                                    │   │
│  │  • Executes plugin hooks                                      │   │
│  └──────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────┘
         │                    │                    │
         ▼                    ▼                    ▼
┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐
│  Schema Engine  │  │   Data Engine   │  │    Adapters     │
│  ┌───────────┐  │  │  ┌───────────┐  │  │  ┌───────────┐  │
│  │    SQL    │  │  │  │  Faker    │  │  │  │ PostgreSQL│  │
│  └───────────┘  │  │  │  Core     │  │  │  └───────────┘  │
│  ┌───────────┐  │  │  └───────────┘  │  │  ┌───────────┐  │
│  │   NoSQL   │  │  │  ┌───────────┐  │  │  │  SQLite   │  │
│  │  (future) │  │  │  │Generators │  │  │  └───────────┘  │
│  └───────────┘  │  │  └───────────┘  │  │  ┌───────────┐  │
│                 │  │  ┌───────────┐  │  │  │  MySQL    │  │
│                 │  │  │Validators │  │  │  │ (future)  │  │
│                 │  │  └───────────┘  │  │  └───────────┘  │
└─────────────────┘  └─────────────────┘  └─────────────────┘
         │                    │                    │
         └────────────────────┼────────────────────┘
                              ▼
┌─────────────────────────────────────────────────────────────────────┐
│                         Plugin System                                │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐                  │
│  │   Registry  │  │    Hooks    │  │  Community  │                  │
│  │             │  │  • Before   │  │   Plugins   │                  │
│  │ • Adapters  │  │  • After    │  │             │                  │
│  │ • Engines   │  │  • OnError  │  │             │                  │
│  │ • Generators│  │             │  │             │                  │
│  └─────────────┘  └─────────────┘  └─────────────┘                  │
└─────────────────────────────────────────────────────────────────────┘
```

## Directory Structure

```
seedcli/
├── cmd/                    # CLI commands
│   ├── root.go            # Root command & global flags
│   ├── init.go            # seedcli init
│   ├── seed.go            # seedcli seed
│   ├── list.go            # seedcli list
│   ├── preview.go         # seedcli preview
│   └── version.go         # seedcli version
│
├── pkg/                    # Core packages
│   ├── core/              # Interface definitions
│   │   └── interfaces.go  # All core interfaces
│   │
│   ├── config/            # Configuration management
│   │   └── config.go      # YAML config handling
│   │
│   ├── logger/            # Logging system
│   │   └── logger.go      # Structured logging to .logseed
│   │
│   ├── schema/            # Schema engines
│   │   ├── sql_engine.go  # SQL introspection
│   │   └── topological.go # Dependency sorting
│   │
│   ├── data/              # Data generation
│   │   ├── engine.go      # Faker core
│   │   └── validators.go  # Data validators
│   │
│   ├── adapter/           # Database adapters
│   │   ├── factory.go     # Adapter factory
│   │   ├── postgres.go    # PostgreSQL adapter
│   │   └── sqlite.go      # SQLite adapter
│   │
│   ├── seeder/            # Seeding orchestration
│   │   └── seeder.go      # Main seeder
│   │
│   └── plugin/            # Plugin system
│       ├── registry.go    # Component registry
│       └── hooks.go       # Plugin hooks
│
├── seedcli.yaml           # User configuration
└── .logseed/              # Log files
```

## Core Interfaces

### Adapter Interface

Database adapters implement the `Adapter` interface:

```go
type Adapter interface {
    Connect(ctx context.Context, dsn string) error
    Close() error
    Ping(ctx context.Context) error
    Execute(ctx context.Context, query string, args ...interface{}) (Result, error)
    Query(ctx context.Context, query string, args ...interface{}) (Rows, error)
    BeginTx(ctx context.Context) (Transaction, error)
    Dialect() Dialect
    QuoteIdentifier(name string) string
    Placeholder(index int) string
}
```

### Schema Engine Interface

Schema engines implement `SchemaEngine`:

```go
type SchemaEngine interface {
    SetAdapter(adapter Adapter)
    ListCollections(ctx context.Context) ([]string, error)
    IntrospectCollection(ctx context.Context, name string) (*Collection, error)
    IntrospectAll(ctx context.Context) ([]*Collection, error)
    GetDependencyOrder(collections []*Collection) ([]*Collection, error)
    ValidateSchema(collections []*Collection) []SchemaError
}
```

### Generator Interface

Custom generators implement `Generator`:

```go
type Generator interface {
    Generate(ctx context.Context, field *Field, opts GeneratorOptions) (interface{}, error)
    Supports(field *Field) bool
    Priority() int
}
```

### Plugin Interface

Plugins implement `Plugin`:

```go
type Plugin interface {
    Name() string
    Version() string
    Description() string
    Init(config map[string]interface{}) error
    Type() PluginType
    Hooks() PluginHooks
}
```

## Seeding Flow

1. **Load Configuration** - Parse `seedcli.yaml`
2. **Connect to Database** - Create adapter and establish connection
3. **Introspect Schema** - Discover tables, columns, and relationships
4. **Resolve Dependencies** - Topological sort based on foreign keys
5. **Execute BeforeSeed Hooks** - Run plugin pre-processing
6. **Generate & Insert Data** - For each table in dependency order:
   - Generate fake data using Data Engine
   - Validate data with registered validators
   - Insert in batches via Adapter
7. **Execute AfterSeed Hooks** - Run plugin post-processing
8. **Report Results** - Display summary

## Logging

All operations are logged to `.logseed/seedcli-YYYY-MM-DD.log`:

```
2026-01-26 10:30:15 [info] Starting seeding process tables=5 rows_per_table=10
2026-01-26 10:30:15 [info] Seeding collection table=users rows=10
2026-01-26 10:30:16 [success] Seeded collection table=users rows=10 duration=0.5s
```

## Future Roadmap

- [ ] MongoDB adapter (NoSQL support)
- [ ] MySQL adapter
- [ ] WASM plugin support
- [ ] Web UI for configuration
- [ ] Data masking/anonymization
