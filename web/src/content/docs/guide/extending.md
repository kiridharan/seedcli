---
title: Extending seedcli
description: How to extend seedcli with custom generators, adapters, and plugins.
sidebar:
  order: 4
---

seedcli's interface-based architecture makes it easy to extend with custom functionality.

## Custom Generators

Create generators for domain-specific data.

### Example: Credit Card Generator

```go
package mygenerators

import (
    "context"
    "strings"
    
    "github.com/brianvoe/gofakeit/v6"
    "github.com/kiridharan/seedcli/pkg/core"
)

type CreditCardGenerator struct {
    faker *gofakeit.Faker
}

func NewCreditCardGenerator() *CreditCardGenerator {
    return &CreditCardGenerator{
        faker: gofakeit.New(0),
    }
}

// Generate creates a credit card number
func (g *CreditCardGenerator) Generate(
    ctx context.Context, 
    field *core.Field, 
    opts core.GeneratorOptions,
) (interface{}, error) {
    return g.faker.CreditCardNumber(nil), nil
}

// Supports returns true if this generator handles the field
func (g *CreditCardGenerator) Supports(field *core.Field) bool {
    name := strings.ToLower(field.Name)
    return strings.Contains(name, "credit_card") || 
           strings.Contains(name, "card_number")
}

// Priority determines generator precedence (higher = first)
func (g *CreditCardGenerator) Priority() int {
    return 100  // Higher than default generators
}
```

### Registering Generators

```go
engine := data.NewEngine()
engine.RegisterGenerator("credit_card", NewCreditCardGenerator())
```

## Custom Adapters

Add support for new databases by implementing the Adapter interface.

### Example: MySQL Adapter

```go
package mysql

import (
    "context"
    "database/sql"
    
    _ "github.com/go-sql-driver/mysql"
    "github.com/kiridharan/seedcli/pkg/core"
)

type MySQLAdapter struct {
    db *sql.DB
}

func NewMySQLAdapter() *MySQLAdapter {
    return &MySQLAdapter{}
}

func (a *MySQLAdapter) Connect(ctx context.Context, dsn string) error {
    db, err := sql.Open("mysql", dsn)
    if err != nil {
        return err
    }
    a.db = db
    return a.db.PingContext(ctx)
}

func (a *MySQLAdapter) Close() error {
    if a.db != nil {
        return a.db.Close()
    }
    return nil
}

func (a *MySQLAdapter) Dialect() core.Dialect {
    return core.DialectMySQL
}

func (a *MySQLAdapter) QuoteIdentifier(name string) string {
    return "`" + name + "`"
}

func (a *MySQLAdapter) Placeholder(index int) string {
    return "?"
}

// Implement remaining interface methods...
```

### Registering Adapters

```go
factory := adapter.GetFactory()
factory.Register(core.DialectMySQL, func() core.Adapter {
    return mysql.NewMySQLAdapter()
})
```

## Plugins

Plugins hook into the seeding lifecycle for custom behavior.

### Plugin Hooks

| Hook | When Called |
|------|-------------|
| `BeforeSeed` | Before seeding starts |
| `AfterSeed` | After seeding completes |
| `OnError` | When an error occurs |
| `BeforeCollection` | Before each table is seeded |
| `AfterCollection` | After each table is seeded |

### Example: Notification Plugin

```go
package plugins

import (
    "context"
    "fmt"
    
    "github.com/kiridharan/seedcli/pkg/core"
    "github.com/kiridharan/seedcli/pkg/plugin"
)

type NotificationPlugin struct {
    *plugin.BasePlugin
    webhookURL string
}

func NewNotificationPlugin() *NotificationPlugin {
    p := &NotificationPlugin{
        BasePlugin: plugin.NewBasePlugin(
            "notification",
            "1.0.0",
            "Sends notifications on seeding events",
            core.PluginTypeHook,
        ),
    }
    
    p.SetHooks(core.PluginHooks{
        BeforeSeed: func(ctx context.Context, collections []*core.Collection) error {
            fmt.Printf("🚀 Starting to seed %d tables\n", len(collections))
            return nil
        },
        AfterSeed: func(ctx context.Context, result *core.SeedResult) error {
            fmt.Printf("✅ Seeded %d total rows in %s\n", 
                result.TotalRows, result.Duration)
            // Send webhook notification here
            return nil
        },
        OnError: func(ctx context.Context, err error) error {
            fmt.Printf("❌ Error: %s\n", err.Error())
            // Send alert here
            return nil
        },
    })
    
    return p
}

func (p *NotificationPlugin) Init(config map[string]interface{}) error {
    if url, ok := config["webhook_url"].(string); ok {
        p.webhookURL = url
    }
    return nil
}
```

### Registering Plugins

```go
seeder := seeder.NewSeeder(adapter, config)
seeder.AddPlugin(NewNotificationPlugin())
```

### Plugin Configuration

Configure plugins in `seedcli.yaml`:

```yaml
plugins:
  enabled:
    - notification
  
  notification:
    webhook_url: https://hooks.slack.com/services/xxx
```

## Custom Validators

Validate generated data before insertion.

### Example: Email Domain Validator

```go
package validators

import (
    "fmt"
    "strings"
    
    "github.com/kiridharan/seedcli/pkg/core"
)

type EmailDomainValidator struct {
    allowedDomains []string
}

func NewEmailDomainValidator(domains []string) *EmailDomainValidator {
    return &EmailDomainValidator{
        allowedDomains: domains,
    }
}

func (v *EmailDomainValidator) Validate(
    field *core.Field, 
    value interface{},
) core.ValidationResult {
    email, ok := value.(string)
    if !ok {
        return core.ValidationResult{Valid: true}
    }
    
    for _, domain := range v.allowedDomains {
        if strings.HasSuffix(email, "@"+domain) {
            return core.ValidationResult{Valid: true}
        }
    }
    
    return core.ValidationResult{
        Valid:   false,
        Message: fmt.Sprintf("email must use allowed domain: %v", v.allowedDomains),
    }
}

func (v *EmailDomainValidator) Supports(field *core.Field) bool {
    return strings.Contains(strings.ToLower(field.Name), "email")
}
```

### Registering Validators

```go
engine := data.NewEngine()
engine.RegisterValidator("email_domain", 
    NewEmailDomainValidator([]string{"company.com", "corp.net"}))
```

## Best Practices

1. **Use Interfaces** - Always program against interfaces for flexibility
2. **Handle Errors** - Return meaningful errors from generators
3. **Support Uniqueness** - Check `opts.IsUnique` and use `opts.UsedValues`
4. **Set Priority** - Higher priority generators run first
5. **Test Thoroughly** - Generators should handle edge cases
6. **Document** - Add clear descriptions to plugins
