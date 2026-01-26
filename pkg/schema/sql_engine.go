// Package schema provides SQL schema introspection capabilities
package schema

import (
	"context"
	"fmt"
	"strings"

	"github.com/kiridharan/seedcli/pkg/core"
)

// SQLEngine implements SchemaEngine for SQL databases
type SQLEngine struct {
	adapter core.Adapter
}

// NewSQLEngine creates a new SQL schema engine
func NewSQLEngine() *SQLEngine {
	return &SQLEngine{}
}

// SetAdapter sets the database adapter
func (e *SQLEngine) SetAdapter(adapter core.Adapter) {
	e.adapter = adapter
}

// ListCollections returns all tables in the database
func (e *SQLEngine) ListCollections(ctx context.Context) ([]string, error) {
	if e.adapter == nil {
		return nil, fmt.Errorf("adapter not set")
	}

	query := e.getListTablesQuery()
	rows, err := e.adapter.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list tables: %w", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return nil, err
		}
		tables = append(tables, tableName)
	}

	return tables, rows.Err()
}

// IntrospectCollection analyzes a table's structure
func (e *SQLEngine) IntrospectCollection(ctx context.Context, name string) (*core.Collection, error) {
	if e.adapter == nil {
		return nil, fmt.Errorf("adapter not set")
	}

	collection := &core.Collection{
		Name:     name,
		Type:     core.CollectionTypeTable,
		Fields:   []*core.Field{},
		Metadata: make(map[string]interface{}),
	}

	// Get columns
	fields, err := e.getColumns(ctx, name)
	if err != nil {
		return nil, err
	}
	collection.Fields = fields

	// Get primary keys
	pks, err := e.getPrimaryKeys(ctx, name)
	if err != nil {
		return nil, err
	}
	collection.PrimaryKey = pks

	// Mark primary key fields
	for _, field := range collection.Fields {
		for _, pk := range pks {
			if field.Name == pk {
				field.IsPrimaryKey = true
				break
			}
		}
	}

	// Get foreign keys
	fks, err := e.getForeignKeys(ctx, name)
	if err != nil {
		return nil, err
	}
	collection.ForeignKeys = fks

	// Get unique constraints
	uniques, err := e.getUniqueConstraints(ctx, name)
	if err != nil {
		return nil, err
	}
	collection.Constraints = uniques

	// Mark unique fields
	for _, constraint := range uniques {
		if constraint.Type == core.ConstraintTypeUnique && len(constraint.Columns) == 1 {
			for _, field := range collection.Fields {
				if field.Name == constraint.Columns[0] {
					field.IsUnique = true
					break
				}
			}
		}
	}

	// Get indexes
	indexes, err := e.getIndexes(ctx, name)
	if err != nil {
		return nil, err
	}
	collection.Indexes = indexes

	return collection, nil
}

// IntrospectAll analyzes all collections in the database
func (e *SQLEngine) IntrospectAll(ctx context.Context) ([]*core.Collection, error) {
	tables, err := e.ListCollections(ctx)
	if err != nil {
		return nil, err
	}

	collections := make([]*core.Collection, 0, len(tables))
	for _, tableName := range tables {
		collection, err := e.IntrospectCollection(ctx, tableName)
		if err != nil {
			return nil, fmt.Errorf("failed to introspect %s: %w", tableName, err)
		}
		collections = append(collections, collection)
	}

	return collections, nil
}

// GetDependencyOrder returns collections in topological order
func (e *SQLEngine) GetDependencyOrder(collections []*core.Collection) ([]*core.Collection, error) {
	return TopologicalSort(collections)
}

// ValidateSchema checks if the schema is valid for seeding
func (e *SQLEngine) ValidateSchema(collections []*core.Collection) []core.SchemaError {
	var errors []core.SchemaError

	for _, col := range collections {
		// Check for tables with no columns
		if len(col.Fields) == 0 {
			errors = append(errors, core.SchemaError{
				Collection: col.Name,
				Message:    "table has no columns",
				Severity:   core.SeverityError,
			})
		}

		// Check foreign key references
		for _, fk := range col.ForeignKeys {
			found := false
			for _, other := range collections {
				if other.Name == fk.ReferencedTable {
					found = true
					break
				}
			}
			if !found {
				errors = append(errors, core.SchemaError{
					Collection: col.Name,
					Field:      fk.ColumnName,
					Message:    fmt.Sprintf("foreign key references unknown table: %s", fk.ReferencedTable),
					Severity:   core.SeverityWarning,
				})
			}
		}
	}

	return errors
}

// getListTablesQuery returns the query to list tables based on dialect
func (e *SQLEngine) getListTablesQuery() string {
	switch e.adapter.Dialect() {
	case core.DialectPostgres:
		return `
			SELECT table_name 
			FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_type = 'BASE TABLE'
			ORDER BY table_name`
	case core.DialectSQLite:
		return `
			SELECT name 
			FROM sqlite_master 
			WHERE type='table' 
			AND name NOT LIKE 'sqlite_%'
			ORDER BY name`
	case core.DialectMySQL:
		return `
			SELECT table_name 
			FROM information_schema.tables 
			WHERE table_schema = DATABASE()
			AND table_type = 'BASE TABLE'
			ORDER BY table_name`
	default:
		return ""
	}
}

// getColumns retrieves column information for a table
func (e *SQLEngine) getColumns(ctx context.Context, tableName string) ([]*core.Field, error) {
	switch e.adapter.Dialect() {
	case core.DialectPostgres:
		return e.getColumnsPostgres(ctx, tableName)
	case core.DialectSQLite:
		return e.getColumnsSQLite(ctx, tableName)
	default:
		return nil, fmt.Errorf("unsupported dialect: %s", e.adapter.Dialect())
	}
}

// getColumnsPostgres retrieves columns for PostgreSQL
func (e *SQLEngine) getColumnsPostgres(ctx context.Context, tableName string) ([]*core.Field, error) {
	query := `
		SELECT 
			column_name,
			data_type,
			is_nullable,
			column_default,
			character_maximum_length,
			numeric_precision,
			numeric_scale,
			udt_name
		FROM information_schema.columns
		WHERE table_name = $1 AND table_schema = 'public'
		ORDER BY ordinal_position`

	rows, err := e.adapter.Query(ctx, query, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var fields []*core.Field
	for rows.Next() {
		var (
			name, dataType, isNullable string
			defaultVal, udtName        *string
			maxLength, precision       *int64
			scale                      *int
		)

		err := rows.Scan(&name, &dataType, &isNullable, &defaultVal, &maxLength, &precision, &scale, &udtName)
		if err != nil {
			return nil, err
		}

		field := &core.Field{
			Name:       name,
			RawType:    dataType,
			Type:       mapPostgresType(dataType),
			IsNullable: isNullable == "YES",
			Metadata:   make(map[string]interface{}),
		}

		if defaultVal != nil {
			field.Default = *defaultVal
			if strings.Contains(*defaultVal, "nextval") {
				field.IsAutoIncr = true
			}
		}

		if maxLength != nil {
			field.MaxLength = *maxLength
		}

		if precision != nil {
			field.Precision = int(*precision)
		}

		if scale != nil {
			field.Scale = *scale
		}

		// Handle ENUM types
		if udtName != nil && dataType == "USER-DEFINED" {
			enumValues, err := e.getEnumValues(ctx, *udtName)
			if err == nil && len(enumValues) > 0 {
				field.Type = core.FieldTypeEnum
				field.EnumValues = enumValues
			}
		}

		fields = append(fields, field)
	}

	return fields, rows.Err()
}

// getColumnsSQLite retrieves columns for SQLite
func (e *SQLEngine) getColumnsSQLite(ctx context.Context, tableName string) ([]*core.Field, error) {
	query := fmt.Sprintf("PRAGMA table_info(%s)", e.adapter.QuoteIdentifier(tableName))

	rows, err := e.adapter.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var fields []*core.Field
	for rows.Next() {
		var (
			cid      int
			name     string
			dataType string
			notNull  int
			dfltVal  *string
			pk       int
		)

		if err := rows.Scan(&cid, &name, &dataType, &notNull, &dfltVal, &pk); err != nil {
			return nil, err
		}

		field := &core.Field{
			Name:         name,
			RawType:      dataType,
			Type:         mapSQLiteType(dataType),
			IsNullable:   notNull == 0,
			IsPrimaryKey: pk > 0,
			Metadata:     make(map[string]interface{}),
		}

		if dfltVal != nil {
			field.Default = *dfltVal
		}

		// SQLite autoincrement detection
		if pk > 0 && strings.ToUpper(dataType) == "INTEGER" {
			field.IsAutoIncr = true
		}

		fields = append(fields, field)
	}

	return fields, rows.Err()
}

// getPrimaryKeys retrieves primary key columns
func (e *SQLEngine) getPrimaryKeys(ctx context.Context, tableName string) ([]string, error) {
	var query string

	switch e.adapter.Dialect() {
	case core.DialectPostgres:
		query = `
			SELECT a.attname
			FROM pg_index i
			JOIN pg_attribute a ON a.attrelid = i.indrelid AND a.attnum = ANY(i.indkey)
			WHERE i.indrelid = $1::regclass
			AND i.indisprimary`
	case core.DialectSQLite:
		// SQLite returns PK info in table_info
		return e.getPrimaryKeysSQLite(ctx, tableName)
	default:
		return nil, fmt.Errorf("unsupported dialect")
	}

	rows, err := e.adapter.Query(ctx, query, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pks []string
	for rows.Next() {
		var pk string
		if err := rows.Scan(&pk); err != nil {
			return nil, err
		}
		pks = append(pks, pk)
	}

	return pks, rows.Err()
}

// getPrimaryKeysSQLite gets primary keys for SQLite
func (e *SQLEngine) getPrimaryKeysSQLite(ctx context.Context, tableName string) ([]string, error) {
	query := fmt.Sprintf("PRAGMA table_info(%s)", e.adapter.QuoteIdentifier(tableName))

	rows, err := e.adapter.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pks []string
	for rows.Next() {
		var cid int
		var name, dataType string
		var notNull, pk int
		var dfltVal *string

		if err := rows.Scan(&cid, &name, &dataType, &notNull, &dfltVal, &pk); err != nil {
			return nil, err
		}

		if pk > 0 {
			pks = append(pks, name)
		}
	}

	return pks, rows.Err()
}

// getForeignKeys retrieves foreign key relationships
func (e *SQLEngine) getForeignKeys(ctx context.Context, tableName string) ([]*core.ForeignKey, error) {
	switch e.adapter.Dialect() {
	case core.DialectPostgres:
		return e.getForeignKeysPostgres(ctx, tableName)
	case core.DialectSQLite:
		return e.getForeignKeysSQLite(ctx, tableName)
	default:
		return nil, fmt.Errorf("unsupported dialect")
	}
}

// getForeignKeysPostgres retrieves foreign keys for PostgreSQL
func (e *SQLEngine) getForeignKeysPostgres(ctx context.Context, tableName string) ([]*core.ForeignKey, error) {
	query := `
		SELECT
			kcu.column_name,
			ccu.table_name AS foreign_table_name,
			ccu.column_name AS foreign_column_name,
			rc.delete_rule,
			rc.update_rule
		FROM information_schema.table_constraints AS tc
		JOIN information_schema.key_column_usage AS kcu
			ON tc.constraint_name = kcu.constraint_name
		JOIN information_schema.constraint_column_usage AS ccu
			ON ccu.constraint_name = tc.constraint_name
		LEFT JOIN information_schema.referential_constraints AS rc
			ON tc.constraint_name = rc.constraint_name
		WHERE tc.constraint_type = 'FOREIGN KEY'
		AND tc.table_name = $1`

	rows, err := e.adapter.Query(ctx, query, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var fks []*core.ForeignKey
	for rows.Next() {
		fk := &core.ForeignKey{}
		var onDelete, onUpdate *string

		if err := rows.Scan(&fk.ColumnName, &fk.ReferencedTable, &fk.ReferencedColumn, &onDelete, &onUpdate); err != nil {
			return nil, err
		}

		if onDelete != nil {
			fk.OnDelete = *onDelete
		}
		if onUpdate != nil {
			fk.OnUpdate = *onUpdate
		}

		fks = append(fks, fk)
	}

	return fks, rows.Err()
}

// getForeignKeysSQLite retrieves foreign keys for SQLite
func (e *SQLEngine) getForeignKeysSQLite(ctx context.Context, tableName string) ([]*core.ForeignKey, error) {
	query := fmt.Sprintf("PRAGMA foreign_key_list(%s)", e.adapter.QuoteIdentifier(tableName))

	rows, err := e.adapter.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var fks []*core.ForeignKey
	for rows.Next() {
		var id, seq int
		var table, from, to, onUpdate, onDelete, match string

		if err := rows.Scan(&id, &seq, &table, &from, &to, &onUpdate, &onDelete, &match); err != nil {
			return nil, err
		}

		fks = append(fks, &core.ForeignKey{
			ColumnName:       from,
			ReferencedTable:  table,
			ReferencedColumn: to,
			OnDelete:         onDelete,
			OnUpdate:         onUpdate,
		})
	}

	return fks, rows.Err()
}

// getUniqueConstraints retrieves unique constraints
func (e *SQLEngine) getUniqueConstraints(ctx context.Context, tableName string) ([]*core.Constraint, error) {
	switch e.adapter.Dialect() {
	case core.DialectPostgres:
		return e.getUniqueConstraintsPostgres(ctx, tableName)
	case core.DialectSQLite:
		return e.getUniqueConstraintsSQLite(ctx, tableName)
	default:
		return nil, nil
	}
}

// getUniqueConstraintsPostgres retrieves unique constraints for PostgreSQL
func (e *SQLEngine) getUniqueConstraintsPostgres(ctx context.Context, tableName string) ([]*core.Constraint, error) {
	query := `
		SELECT
			tc.constraint_name,
			array_agg(kcu.column_name ORDER BY kcu.ordinal_position)
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu
			ON tc.constraint_name = kcu.constraint_name
		WHERE tc.table_name = $1
		AND tc.constraint_type = 'UNIQUE'
		GROUP BY tc.constraint_name`

	rows, err := e.adapter.Query(ctx, query, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var constraints []*core.Constraint
	for rows.Next() {
		var name string
		var columns []string

		if err := rows.Scan(&name, &columns); err != nil {
			return nil, err
		}

		constraints = append(constraints, &core.Constraint{
			Name:    name,
			Type:    core.ConstraintTypeUnique,
			Columns: columns,
		})
	}

	return constraints, rows.Err()
}

// getUniqueConstraintsSQLite retrieves unique constraints for SQLite
func (e *SQLEngine) getUniqueConstraintsSQLite(ctx context.Context, tableName string) ([]*core.Constraint, error) {
	query := fmt.Sprintf("PRAGMA index_list(%s)", e.adapter.QuoteIdentifier(tableName))

	rows, err := e.adapter.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var constraints []*core.Constraint
	for rows.Next() {
		var seq int
		var name, origin string
		var unique, partial int

		if err := rows.Scan(&seq, &name, &unique, &origin, &partial); err != nil {
			return nil, err
		}

		if unique == 1 {
			// Get columns for this index
			cols, err := e.getIndexColumns(ctx, name)
			if err != nil {
				continue
			}

			constraints = append(constraints, &core.Constraint{
				Name:    name,
				Type:    core.ConstraintTypeUnique,
				Columns: cols,
			})
		}
	}

	return constraints, rows.Err()
}

// getIndexes retrieves index information
func (e *SQLEngine) getIndexes(ctx context.Context, tableName string) ([]*core.Index, error) {
	switch e.adapter.Dialect() {
	case core.DialectPostgres:
		return e.getIndexesPostgres(ctx, tableName)
	case core.DialectSQLite:
		return e.getIndexesSQLite(ctx, tableName)
	default:
		return nil, nil
	}
}

// getIndexesPostgres retrieves indexes for PostgreSQL
func (e *SQLEngine) getIndexesPostgres(ctx context.Context, tableName string) ([]*core.Index, error) {
	query := `
		SELECT
			i.relname AS index_name,
			array_agg(a.attname ORDER BY array_position(ix.indkey, a.attnum)) AS columns,
			ix.indisunique,
			ix.indisprimary,
			am.amname AS index_type
		FROM pg_class t
		JOIN pg_index ix ON t.oid = ix.indrelid
		JOIN pg_class i ON i.oid = ix.indexrelid
		JOIN pg_am am ON i.relam = am.oid
		JOIN pg_attribute a ON a.attrelid = t.oid AND a.attnum = ANY(ix.indkey)
		WHERE t.relname = $1
		GROUP BY i.relname, ix.indisunique, ix.indisprimary, am.amname`

	rows, err := e.adapter.Query(ctx, query, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var indexes []*core.Index
	for rows.Next() {
		idx := &core.Index{}
		var columns []string

		if err := rows.Scan(&idx.Name, &columns, &idx.IsUnique, &idx.IsPrimary, &idx.Type); err != nil {
			return nil, err
		}

		idx.Columns = columns
		indexes = append(indexes, idx)
	}

	return indexes, rows.Err()
}

// getIndexesSQLite retrieves indexes for SQLite
func (e *SQLEngine) getIndexesSQLite(ctx context.Context, tableName string) ([]*core.Index, error) {
	query := fmt.Sprintf("PRAGMA index_list(%s)", e.adapter.QuoteIdentifier(tableName))

	rows, err := e.adapter.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var indexes []*core.Index
	for rows.Next() {
		var seq int
		var name, origin string
		var unique, partial int

		if err := rows.Scan(&seq, &name, &unique, &origin, &partial); err != nil {
			return nil, err
		}

		cols, err := e.getIndexColumns(ctx, name)
		if err != nil {
			continue
		}

		indexes = append(indexes, &core.Index{
			Name:     name,
			Columns:  cols,
			IsUnique: unique == 1,
		})
	}

	return indexes, rows.Err()
}

// getIndexColumns gets columns for an index
func (e *SQLEngine) getIndexColumns(ctx context.Context, indexName string) ([]string, error) {
	query := fmt.Sprintf("PRAGMA index_info(%s)", e.adapter.QuoteIdentifier(indexName))

	rows, err := e.adapter.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []string
	for rows.Next() {
		var seqno, cid int
		var name string

		if err := rows.Scan(&seqno, &cid, &name); err != nil {
			return nil, err
		}

		columns = append(columns, name)
	}

	return columns, rows.Err()
}

// getEnumValues retrieves enum values for PostgreSQL
func (e *SQLEngine) getEnumValues(ctx context.Context, typeName string) ([]string, error) {
	query := `
		SELECT e.enumlabel
		FROM pg_type t
		JOIN pg_enum e ON t.oid = e.enumtypid
		WHERE t.typname = $1
		ORDER BY e.enumsortorder`

	rows, err := e.adapter.Query(ctx, query, typeName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var values []string
	for rows.Next() {
		var val string
		if err := rows.Scan(&val); err != nil {
			return nil, err
		}
		values = append(values, val)
	}

	return values, rows.Err()
}

// mapPostgresType maps PostgreSQL types to abstract types
func mapPostgresType(pgType string) core.FieldType {
	pgType = strings.ToLower(pgType)

	switch {
	case pgType == "boolean" || pgType == "bool":
		return core.FieldTypeBool
	case strings.Contains(pgType, "int") || pgType == "serial" || pgType == "bigserial":
		return core.FieldTypeInt
	case strings.Contains(pgType, "numeric") || strings.Contains(pgType, "decimal") ||
		strings.Contains(pgType, "float") || strings.Contains(pgType, "double") || pgType == "real":
		return core.FieldTypeFloat
	case pgType == "date":
		return core.FieldTypeDate
	case pgType == "time" || pgType == "time without time zone":
		return core.FieldTypeTime
	case strings.Contains(pgType, "timestamp"):
		return core.FieldTypeTimestamp
	case pgType == "uuid":
		return core.FieldTypeUUID
	case pgType == "json" || pgType == "jsonb":
		return core.FieldTypeJSON
	case pgType == "bytea":
		return core.FieldTypeBinary
	case strings.Contains(pgType, "array"):
		return core.FieldTypeArray
	case strings.Contains(pgType, "char") || strings.Contains(pgType, "text"):
		return core.FieldTypeString
	default:
		return core.FieldTypeString
	}
}

// mapSQLiteType maps SQLite types to abstract types
func mapSQLiteType(sqliteType string) core.FieldType {
	sqliteType = strings.ToUpper(sqliteType)

	switch {
	case sqliteType == "INTEGER" || sqliteType == "INT":
		return core.FieldTypeInt
	case sqliteType == "REAL" || sqliteType == "FLOAT" || sqliteType == "DOUBLE":
		return core.FieldTypeFloat
	case sqliteType == "BLOB":
		return core.FieldTypeBinary
	case sqliteType == "BOOLEAN" || sqliteType == "BOOL":
		return core.FieldTypeBool
	case strings.Contains(sqliteType, "DATE"):
		return core.FieldTypeDate
	case strings.Contains(sqliteType, "TIME"):
		return core.FieldTypeTimestamp
	default:
		return core.FieldTypeString
	}
}
