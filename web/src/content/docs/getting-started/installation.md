---
title: Installation
description: How to install seedcli on your system.
sidebar:
  order: 1
---

## Prerequisites

- **Go 1.21** or higher
- **PostgreSQL** and/or **SQLite** database

## Installation Methods

### Method 1: Go Install (Recommended)

```bash
go install github.com/kiridharan/seedcli@latest
```

### Method 2: Build from Source

```bash
# Clone the repository
git clone https://github.com/kiridharan/seedcli
cd seedcli

# Build the binary
go build -o seedcli .

# Move to PATH (optional)
sudo mv seedcli /usr/local/bin/
```

### Method 3: Download Binary

Download pre-built binaries from the [releases page](https://github.com/kiridharan/seedcli/releases).

## Verify Installation

```bash
seedcli version
```

**Expected output:**

```
seedcli version 2.0.0
  Build date: 2026-01-26
  Go version: go1.21
  OS/Arch:    darwin/arm64

Supported adapters:
  • postgres  - PostgreSQL 12+
  • sqlite    - SQLite 3
```

## Database Setup

### PostgreSQL

Ensure PostgreSQL is installed and running:

```bash
# Check PostgreSQL status
pg_isready

# Create a test database
createdb seedcli_test
```

### SQLite

No additional setup required! SQLite databases are created automatically.

## Next Steps

Continue to [Quick Start](/getting-started/quickstart) to learn how to use seedcli.
