// Package data provides the data generation engine for seedcli
package data

import (
	"context"
	"fmt"
	"math/rand"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	gofakeit "github.com/brianvoe/gofakeit/v6"
	"github.com/kiridharan/seedcli/pkg/core"
)

// Engine implements the core.DataEngine interface
type Engine struct {
	mu              sync.RWMutex
	faker           *gofakeit.Faker
	seed            int64
	generators      map[string]core.Generator
	validators      map[string]core.Validator
	referenceData   map[string][]interface{} // table -> inserted PKs
	usedValues      map[string]map[interface{}]bool
	nullProbability float64
}

// NewEngine creates a new data generation engine
func NewEngine() *Engine {
	seed := time.Now().UnixNano()
	return &Engine{
		faker:           gofakeit.New(seed),
		seed:            seed,
		generators:      make(map[string]core.Generator),
		validators:      make(map[string]core.Validator),
		referenceData:   make(map[string][]interface{}),
		usedValues:      make(map[string]map[interface{}]bool),
		nullProbability: 0.3,
	}
}

// SetSeed sets the random seed for reproducible generation
func (e *Engine) SetSeed(seed int64) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.seed = seed
	e.faker = gofakeit.New(seed)
	rand.Seed(seed)
}

// SetNullProbability sets the probability of generating NULL for nullable fields
func (e *Engine) SetNullProbability(prob float64) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.nullProbability = prob
}

// RegisterGenerator registers a custom generator
func (e *Engine) RegisterGenerator(name string, gen core.Generator) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.generators[name] = gen
}

// RegisterValidator registers a custom validator
func (e *Engine) RegisterValidator(name string, val core.Validator) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.validators[name] = val
}

// SetReferenceData sets foreign key reference data
func (e *Engine) SetReferenceData(tableName string, column string, values []interface{}) {
	e.mu.Lock()
	defer e.mu.Unlock()
	key := tableName
	e.referenceData[key] = values
}

// GetReferenceData gets inserted primary keys for FK references
func (e *Engine) GetReferenceData(tableName string) []interface{} {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.referenceData[tableName]
}

// GenerateRow generates a single row of data for a collection
func (e *Engine) GenerateRow(ctx context.Context, collection *core.Collection) (map[string]interface{}, error) {
	row := make(map[string]interface{})

	for _, field := range collection.Fields {
		// Skip auto-increment fields
		if field.IsAutoIncr {
			continue
		}

		value, err := e.generateValue(ctx, collection, field)
		if err != nil {
			return nil, fmt.Errorf("failed to generate value for %s.%s: %w", collection.Name, field.Name, err)
		}

		row[field.Name] = value
	}

	return row, nil
}

// GenerateRows generates multiple rows for a collection
func (e *Engine) GenerateRows(ctx context.Context, collection *core.Collection, count int) ([]map[string]interface{}, error) {
	rows := make([]map[string]interface{}, 0, count)

	for i := 0; i < count; i++ {
		select {
		case <-ctx.Done():
			return rows, ctx.Err()
		default:
		}

		row, err := e.GenerateRow(ctx, collection)
		if err != nil {
			return nil, err
		}
		rows = append(rows, row)
	}

	return rows, nil
}

// generateValue generates a value for a field
func (e *Engine) generateValue(ctx context.Context, collection *core.Collection, field *core.Field) (interface{}, error) {
	// Check for foreign key
	for _, fk := range collection.ForeignKeys {
		if fk.ColumnName == field.Name {
			return e.generateForeignKeyValue(fk, field)
		}
	}

	// Check for nullable
	if field.IsNullable && !field.IsPrimaryKey {
		e.mu.RLock()
		prob := e.nullProbability
		e.mu.RUnlock()
		if rand.Float64() < prob {
			return nil, nil
		}
	}

	// Try custom generators first (sorted by priority)
	generators := e.getSortedGenerators()
	for _, gen := range generators {
		if gen.Supports(field) {
			opts := core.GeneratorOptions{
				Seed:       e.seed,
				IsUnique:   field.IsUnique || field.IsPrimaryKey,
				UsedValues: e.getUsedValues(collection.Name, field.Name),
			}
			value, err := gen.Generate(ctx, field, opts)
			if err == nil {
				if opts.IsUnique {
					e.markValueUsed(collection.Name, field.Name, value)
				}
				return value, nil
			}
		}
	}

	// Fall back to default generation
	return e.generateByHeuristic(collection, field)
}

// generateForeignKeyValue generates a foreign key value
func (e *Engine) generateForeignKeyValue(fk *core.ForeignKey, field *core.Field) (interface{}, error) {
	e.mu.RLock()
	pks := e.referenceData[fk.ReferencedTable]
	e.mu.RUnlock()

	if len(pks) == 0 {
		if field.IsNullable {
			return nil, nil
		}
		return nil, fmt.Errorf("no reference data for table %s", fk.ReferencedTable)
	}

	return pks[rand.Intn(len(pks))], nil
}

// generateByHeuristic generates a value based on field name and type heuristics
func (e *Engine) generateByHeuristic(collection *core.Collection, field *core.Field) (interface{}, error) {
	nameLower := strings.ToLower(field.Name)

	// Handle enum types
	if len(field.EnumValues) > 0 {
		return field.EnumValues[rand.Intn(len(field.EnumValues))], nil
	}

	// Generate based on field type and name
	switch field.Type {
	case core.FieldTypeBool:
		return rand.Intn(2) == 1, nil

	case core.FieldTypeInt:
		return e.generateInt(nameLower, field)

	case core.FieldTypeFloat:
		return e.generateFloat(nameLower, field)

	case core.FieldTypeDate:
		return e.generateDate(nameLower)

	case core.FieldTypeTime:
		return e.generateTime(nameLower)

	case core.FieldTypeDateTime, core.FieldTypeTimestamp:
		return e.generateTimestamp(nameLower)

	case core.FieldTypeUUID:
		return e.faker.UUID(), nil

	case core.FieldTypeJSON:
		return e.generateJSON()

	case core.FieldTypeBinary:
		return e.generateBinary()

	case core.FieldTypeArray:
		return e.generateArray(field)

	case core.FieldTypeString:
		return e.generateString(nameLower, field)

	default:
		return e.generateString(nameLower, field)
	}
}

// generateInt generates an integer value
func (e *Engine) generateInt(name string, field *core.Field) (interface{}, error) {
	switch {
	case strings.Contains(name, "age"):
		return rand.Intn(80) + 18, nil
	case strings.Contains(name, "rating") || strings.Contains(name, "score"):
		return rand.Intn(5) + 1, nil
	case strings.Contains(name, "count") || strings.Contains(name, "quantity"):
		return rand.Intn(100) + 1, nil
	case strings.Contains(name, "year"):
		return rand.Intn(30) + 1990, nil
	case strings.Contains(name, "month"):
		return rand.Intn(12) + 1, nil
	case strings.Contains(name, "day"):
		return rand.Intn(28) + 1, nil
	case strings.Contains(name, "hour"):
		return rand.Intn(24), nil
	case strings.Contains(name, "minute") || strings.Contains(name, "second"):
		return rand.Intn(60), nil
	case strings.Contains(name, "percent"):
		return rand.Intn(101), nil
	default:
		return rand.Intn(10000), nil
	}
}

// generateFloat generates a float value
func (e *Engine) generateFloat(name string, field *core.Field) (interface{}, error) {
	switch {
	case strings.Contains(name, "price") || strings.Contains(name, "cost") || strings.Contains(name, "amount"):
		return float64(rand.Intn(100000)) / 100, nil // 0.00 - 999.99
	case strings.Contains(name, "rate") || strings.Contains(name, "percent"):
		return float64(rand.Intn(10000)) / 100, nil // 0.00 - 99.99
	case strings.Contains(name, "lat") || strings.Contains(name, "latitude"):
		return e.faker.Latitude(), nil
	case strings.Contains(name, "lon") || strings.Contains(name, "lng") || strings.Contains(name, "longitude"):
		return e.faker.Longitude(), nil
	default:
		return rand.Float64() * 1000, nil
	}
}

// generateDate generates a date value
func (e *Engine) generateDate(name string) (interface{}, error) {
	daysAgo := rand.Intn(365 * 5)
	date := time.Now().AddDate(0, 0, -daysAgo)
	return date.Format("2006-01-02"), nil
}

// generateTime generates a time value
func (e *Engine) generateTime(name string) (interface{}, error) {
	return fmt.Sprintf("%02d:%02d:%02d", rand.Intn(24), rand.Intn(60), rand.Intn(60)), nil
}

// generateTimestamp generates a timestamp value
func (e *Engine) generateTimestamp(name string) (interface{}, error) {
	daysAgo := rand.Intn(365 * 2)
	ts := time.Now().AddDate(0, 0, -daysAgo)

	switch {
	case strings.Contains(name, "created"):
		// Created dates should be older
		return ts.AddDate(0, 0, -rand.Intn(365)), nil
	case strings.Contains(name, "updated") || strings.Contains(name, "modified"):
		// Updated dates should be more recent
		return ts, nil
	case strings.Contains(name, "deleted"):
		// Most records shouldn't be deleted
		if rand.Float64() < 0.9 {
			return nil, nil
		}
		return ts, nil
	default:
		return ts, nil
	}
}

// generateJSON generates a JSON value
func (e *Engine) generateJSON() (interface{}, error) {
	data := map[string]interface{}{
		"id":      e.faker.UUID(),
		"name":    e.faker.Name(),
		"email":   e.faker.Email(),
		"active":  rand.Intn(2) == 1,
		"score":   rand.Intn(100),
		"created": time.Now().Format(time.RFC3339),
	}
	return data, nil
}

// generateBinary generates a binary value
func (e *Engine) generateBinary() (interface{}, error) {
	size := rand.Intn(100) + 10
	data := make([]byte, size)
	rand.Read(data)
	return data, nil
}

// generateArray generates an array value
func (e *Engine) generateArray(field *core.Field) (interface{}, error) {
	size := rand.Intn(5) + 1
	arr := make([]interface{}, size)
	for i := 0; i < size; i++ {
		arr[i] = e.faker.Word()
	}
	return arr, nil
}

// generateString generates a string value based on heuristics
func (e *Engine) generateString(name string, field *core.Field) (interface{}, error) {
	value := e.generateStringByName(name)

	// Truncate if necessary
	if field.MaxLength > 0 && int64(len(value)) > field.MaxLength {
		value = value[:field.MaxLength]
	}

	// Handle uniqueness
	if field.IsUnique || field.IsPrimaryKey {
		value = e.ensureUnique(name, value, field)
	}

	return value, nil
}

// generateStringByName generates string based on column name patterns
func (e *Engine) generateStringByName(name string) string {
	switch {
	// Personal information
	case strings.Contains(name, "email"):
		return e.faker.Email()
	case strings.Contains(name, "first_name") || name == "firstname":
		return e.faker.FirstName()
	case strings.Contains(name, "last_name") || name == "lastname":
		return e.faker.LastName()
	case strings.Contains(name, "full_name") || name == "name":
		return e.faker.Name()
	case strings.Contains(name, "username") || strings.Contains(name, "user_name"):
		return e.faker.Username()
	case strings.Contains(name, "password"):
		return e.faker.Password(true, true, true, true, false, 16)
	case strings.Contains(name, "phone") || strings.Contains(name, "mobile") || strings.Contains(name, "tel"):
		return e.faker.Phone()

	// Address related
	case strings.Contains(name, "address") && !strings.Contains(name, "email"):
		return e.faker.Address().Address
	case strings.Contains(name, "street"):
		return e.faker.Address().Street
	case strings.Contains(name, "city"):
		return e.faker.City()
	case strings.Contains(name, "state") || strings.Contains(name, "province"):
		return e.faker.State()
	case strings.Contains(name, "country"):
		return e.faker.Country()
	case strings.Contains(name, "zip") || strings.Contains(name, "postal"):
		return e.faker.Zip()

	// Internet related
	case strings.Contains(name, "url") || strings.Contains(name, "website") || strings.Contains(name, "link"):
		return e.faker.URL()
	case strings.Contains(name, "domain"):
		return e.faker.DomainName()
	case strings.Contains(name, "ip"):
		return e.faker.IPv4Address()

	// Business related
	case strings.Contains(name, "company") || strings.Contains(name, "organization"):
		return e.faker.Company()
	case strings.Contains(name, "title") && !strings.Contains(name, "job"):
		return e.faker.Sentence(5)
	case strings.Contains(name, "job") || strings.Contains(name, "position"):
		return e.faker.JobTitle()

	// Content related
	case strings.Contains(name, "description") || strings.Contains(name, "bio"):
		return e.faker.Paragraph(1, 3, 10, " ")
	case strings.Contains(name, "content") || strings.Contains(name, "body") || strings.Contains(name, "text"):
		return e.faker.Paragraph(2, 5, 15, " ")
	case strings.Contains(name, "summary") || strings.Contains(name, "excerpt"):
		return e.faker.Sentence(15)
	case strings.Contains(name, "comment"):
		return e.faker.Sentence(10)

	// Identifiers
	case strings.Contains(name, "uuid") || strings.Contains(name, "guid"):
		return e.faker.UUID()
	case strings.Contains(name, "slug"):
		return strings.ToLower(strings.ReplaceAll(e.faker.BuzzWord()+" "+e.faker.Word(), " ", "-"))
	case strings.Contains(name, "code") || strings.Contains(name, "sku"):
		return strings.ToUpper(e.faker.LetterN(3)) + fmt.Sprintf("%05d", rand.Intn(100000))
	case strings.Contains(name, "token") || strings.Contains(name, "key"):
		return e.faker.UUID()

	// Images
	case strings.Contains(name, "image") || strings.Contains(name, "avatar") || strings.Contains(name, "photo"):
		return fmt.Sprintf("https://picsum.photos/seed/%s/200/200", e.faker.UUID()[:8])

	// Status/Type
	case strings.Contains(name, "status"):
		statuses := []string{"active", "inactive", "pending", "completed", "cancelled"}
		return statuses[rand.Intn(len(statuses))]
	case strings.Contains(name, "type") || strings.Contains(name, "category"):
		return e.faker.Word()

	// Colors
	case strings.Contains(name, "color"):
		return e.faker.Color()

	// Currency
	case strings.Contains(name, "currency"):
		currencies := []string{"USD", "EUR", "GBP", "JPY", "CNY", "INR"}
		return currencies[rand.Intn(len(currencies))]

	// Default
	default:
		return e.faker.Sentence(5)
	}
}

// ensureUnique ensures the value is unique
func (e *Engine) ensureUnique(fieldKey, value string, field *core.Field) string {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.usedValues[fieldKey] == nil {
		e.usedValues[fieldKey] = make(map[interface{}]bool)
	}

	original := value
	attempts := 0
	for e.usedValues[fieldKey][value] && attempts < 100 {
		value = original + fmt.Sprintf("_%d", rand.Intn(10000))
		if field.MaxLength > 0 && int64(len(value)) > field.MaxLength {
			value = value[:field.MaxLength]
		}
		attempts++
	}

	e.usedValues[fieldKey][value] = true
	return value
}

// getUsedValues gets used values for a field
func (e *Engine) getUsedValues(table, field string) map[interface{}]bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	key := table + "." + field
	if e.usedValues[key] == nil {
		return make(map[interface{}]bool)
	}
	return e.usedValues[key]
}

// markValueUsed marks a value as used for a field
func (e *Engine) markValueUsed(table, field string, value interface{}) {
	e.mu.Lock()
	defer e.mu.Unlock()
	key := table + "." + field
	if e.usedValues[key] == nil {
		e.usedValues[key] = make(map[interface{}]bool)
	}
	e.usedValues[key][value] = true
}

// getSortedGenerators returns generators sorted by priority
func (e *Engine) getSortedGenerators() []core.Generator {
	e.mu.RLock()
	defer e.mu.RUnlock()

	gens := make([]core.Generator, 0, len(e.generators))
	for _, gen := range e.generators {
		gens = append(gens, gen)
	}

	sort.Slice(gens, func(i, j int) bool {
		return gens[i].Priority() > gens[j].Priority()
	})

	return gens
}

// ClearUsedValues clears the used values cache
func (e *Engine) ClearUsedValues() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.usedValues = make(map[string]map[interface{}]bool)
}

// ClearReferenceData clears the reference data cache
func (e *Engine) ClearReferenceData() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.referenceData = make(map[string][]interface{})
}

// Reset clears all caches
func (e *Engine) Reset() {
	e.ClearUsedValues()
	e.ClearReferenceData()
}

// =============================================================================
// BUILT-IN GENERATORS
// =============================================================================

// PatternGenerator generates values matching a regex pattern
type PatternGenerator struct {
	pattern  *regexp.Regexp
	generate func() string
	priority int
}

// NewPatternGenerator creates a pattern-based generator
func NewPatternGenerator(pattern string, generate func() string, priority int) (*PatternGenerator, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	return &PatternGenerator{
		pattern:  re,
		generate: generate,
		priority: priority,
	}, nil
}

func (g *PatternGenerator) Generate(ctx context.Context, field *core.Field, opts core.GeneratorOptions) (interface{}, error) {
	return g.generate(), nil
}

func (g *PatternGenerator) Supports(field *core.Field) bool {
	return g.pattern.MatchString(strings.ToLower(field.Name))
}

func (g *PatternGenerator) Priority() int {
	return g.priority
}

// FixedValueGenerator always returns the same value
type FixedValueGenerator struct {
	fieldName string
	value     interface{}
}

// NewFixedValueGenerator creates a fixed value generator
func NewFixedValueGenerator(fieldName string, value interface{}) *FixedValueGenerator {
	return &FixedValueGenerator{
		fieldName: strings.ToLower(fieldName),
		value:     value,
	}
}

func (g *FixedValueGenerator) Generate(ctx context.Context, field *core.Field, opts core.GeneratorOptions) (interface{}, error) {
	return g.value, nil
}

func (g *FixedValueGenerator) Supports(field *core.Field) bool {
	return strings.ToLower(field.Name) == g.fieldName
}

func (g *FixedValueGenerator) Priority() int {
	return 100
}

// ChoiceGenerator picks from a list of values
type ChoiceGenerator struct {
	fieldName string
	choices   []interface{}
}

// NewChoiceGenerator creates a choice-based generator
func NewChoiceGenerator(fieldName string, choices []interface{}) *ChoiceGenerator {
	return &ChoiceGenerator{
		fieldName: strings.ToLower(fieldName),
		choices:   choices,
	}
}

func (g *ChoiceGenerator) Generate(ctx context.Context, field *core.Field, opts core.GeneratorOptions) (interface{}, error) {
	return g.choices[rand.Intn(len(g.choices))], nil
}

func (g *ChoiceGenerator) Supports(field *core.Field) bool {
	return strings.ToLower(field.Name) == g.fieldName
}

func (g *ChoiceGenerator) Priority() int {
	return 90
}
