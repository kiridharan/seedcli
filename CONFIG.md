# Config File Feature

## `.seedfake` Configuration File

The CLI now automatically creates and reads from a `.seedfake` configuration file in your current directory. This saves your database connection settings so you don't have to type them every time.

## How It Works

### First Time Usage

When you run seedcli with `--db-url` for the first time:

```bash
./seedcli --db-url "postgresql://postgres:postgres@localhost:5432/postgres" --list
```

**Output:**
```
✓ Connected to database (postgres)
INFO: Saved connection to .seedfake
```

This creates a `.seedfake` file containing:
```json
{
  "database_url": "postgresql://postgres:postgres@localhost:5432/postgres",
  "dialect": "postgres",
  "last_used": "2025-12-03T16:17:57+05:30"
}
```

### Subsequent Usage

After the config file exists, you can omit the `--db-url` flag:

```bash
./seedcli --list
```

**Output:**
```
INFO: Using database URL from .seedfake config
✓ Connected to database (postgres)
```

## Usage Examples

```bash
# First time - create config
./seedcli --db-url "postgresql://postgres:postgres@localhost:5432/postgres" --all --rows 10

# After that - use saved config
./seedcli --list
./seedcli --table users --rows 50
./seedcli --all --preview

# Override saved config
./seedcli --db-url "sqlite://test.db" --list
```

## Config File Location

The `.seedfake` file is created in your **current working directory**, so you can have different configs for different projects.

## Switching Databases

To switch to a different database, just provide a new `--db-url`:

```bash
# Switch to SQLite
./seedcli --db-url "sqlite://myapp.db" --list

# This updates .seedfake with the new SQLite connection
```

## Benefits

✅ **Convenience**: No need to type long connection strings  
✅ **Per-Project**: Each directory can have its own config  
✅ **Git-Ignored**: `.seedfake` is in `.gitignore` to avoid committing credentials  
✅ **Transparent**: Shows when using saved config  

## Security Note

The `.seedfake` file contains your database credentials in plain text. It is automatically added to `.gitignore`, but ensure you don't accidentally commit it to version control.
