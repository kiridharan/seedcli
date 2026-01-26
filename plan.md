PROMPT START

I want you to generate a complete Seed CLI tool called seedcli that automatically inserts fake data into a relational database (Postgres + SQLite).
The tool must be Python, using SQLAlchemy, Click, and Faker, and must solve ALL of the following problems:

✅ 1. Database Support

Must support Postgres and SQLite via SQLAlchemy URLs.

Automatically detect dialect via engine.dialect.name.

✅ 2. Table & Schema Introspection

The CLI must:

List all tables (--list)

Select one or many tables (--table)

Select all tables (--all)

Reflect tables including:

column names

column types

primary keys

autoincrement & default detection

ENUM values

JSON types

Foreign keys (table + column)

Build a dependency graph to determine correct insertion order.

✅ 3. Topological Sorting & FK Integrity

The tool MUST:

Perform topological sorting of tables based on foreign keys.

Insert parent tables before child tables.

Handle cyclic foreign keys:

Insert rows with NULL FKs (if allowed)

OR use Postgres deferred constraints (SET CONSTRAINTS ALL DEFERRED)

OR perform a 2-phase insert (insert, then update FK column)

✅ 4. Fake Data Generation (Smart)

Fake values must be intelligent, based on:

Column type (Integer, Float, Numeric, String, Text, Date, DateTime, UUID, JSON, Enum, Boolean, arrays for Postgres)

Column name heuristics:

email → fake.email()

name → fake.name()

first_name, last_name

phone, address, city, country

uuid → uuid4

url, slug, username

Timestamps (created_at, updated_at)

For ENUM → random choice of allowed values.

For JSON → random small objects / arrays.

For arrays (Postgres) → random list of items.

For BLOB → small random bytes.

Nullable columns:

Should sometimes be NULL.

NOT NULL columns:

Never NULL.

Unique columns:

Must enforce uniqueness via maintaining an in-memory “used_values” registry.

✅ 5. Foreign Key Value Resolution

While inserting child tables, FK columns must be filled using:

Real PKs collected from parent tables that were already inserted.

OR existing rows in DB if table already had data.

For many-to-one FK:

Pick a random PK from referenced table’s PK pool.

✅ 6. Primary Key Handling

If PK is autoincrement → do NOT generate value.

After insert, retrieve real PKs:

Postgres → use RETURNING

SQLite → use cursor.lastrowid per row

Store PKs in inserted_pk_map[table_name].

✅ 7. Performance

Must insert in batches, configurable via --batch-size.

For Postgres:

Optionally support a --copy mode using COPY FROM STDIN.

For SQLite:

Wrap in one big transaction for speed.

✅ 8. Uniqueness & Constraint Handling

Detect UNIQUE constraints & unique indexes.

Guarantee unique values per column.

Respect column length constraints on VARCHAR.

Respect default values when present.

✅ 9. Deterministic Output

Add a global --seed option.

Seed both Faker + Python random.

✅ 10. CLI Features (Click)

The tool should expose these commands / flags:

--db-url           (required)
--list             (list all tables)
--table -t         (multiple values)
--all              (select all tables)
--rows -n          number of rows per table
--preview          show sample rows without inserting
--dry-run          do everything except insert
--seed             set deterministic seed
--batch-size
--skip-errors
--copy             (Postgres only, use COPY)