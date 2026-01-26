// Package data provides validators for generated data
package data

import (
	"fmt"
	"net/mail"
	"net/url"
	"regexp"
	"strings"

	"github.com/kiridharan/seedcli/pkg/core"
)

// =============================================================================
// BUILT-IN VALIDATORS
// =============================================================================

// NotNullValidator ensures non-null values for required fields
type NotNullValidator struct{}

func NewNotNullValidator() *NotNullValidator {
	return &NotNullValidator{}
}

func (v *NotNullValidator) Name() string {
	return "not_null"
}

func (v *NotNullValidator) Validate(field *core.Field, value interface{}) error {
	if !field.IsNullable && value == nil {
		return fmt.Errorf("field %s cannot be null", field.Name)
	}
	return nil
}

// MaxLengthValidator ensures string values don't exceed max length
type MaxLengthValidator struct{}

func NewMaxLengthValidator() *MaxLengthValidator {
	return &MaxLengthValidator{}
}

func (v *MaxLengthValidator) Name() string {
	return "max_length"
}

func (v *MaxLengthValidator) Validate(field *core.Field, value interface{}) error {
	if field.MaxLength <= 0 {
		return nil
	}

	str, ok := value.(string)
	if !ok {
		return nil
	}

	if int64(len(str)) > field.MaxLength {
		return fmt.Errorf("field %s exceeds max length %d (got %d)", field.Name, field.MaxLength, len(str))
	}

	return nil
}

// TypeValidator ensures values match the expected type
type TypeValidator struct{}

func NewTypeValidator() *TypeValidator {
	return &TypeValidator{}
}

func (v *TypeValidator) Name() string {
	return "type"
}

func (v *TypeValidator) Validate(field *core.Field, value interface{}) error {
	if value == nil {
		return nil
	}

	switch field.Type {
	case core.FieldTypeString:
		if _, ok := value.(string); !ok {
			return fmt.Errorf("field %s expected string, got %T", field.Name, value)
		}
	case core.FieldTypeInt:
		switch value.(type) {
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
			// OK
		default:
			return fmt.Errorf("field %s expected integer, got %T", field.Name, value)
		}
	case core.FieldTypeFloat:
		switch value.(type) {
		case float32, float64, int, int64:
			// OK (allow int for float fields)
		default:
			return fmt.Errorf("field %s expected float, got %T", field.Name, value)
		}
	case core.FieldTypeBool:
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("field %s expected bool, got %T", field.Name, value)
		}
	}

	return nil
}

// EmailValidator validates email format
type EmailValidator struct{}

func NewEmailValidator() *EmailValidator {
	return &EmailValidator{}
}

func (v *EmailValidator) Name() string {
	return "email"
}

func (v *EmailValidator) Validate(field *core.Field, value interface{}) error {
	if !strings.Contains(strings.ToLower(field.Name), "email") {
		return nil
	}

	if value == nil {
		return nil
	}

	str, ok := value.(string)
	if !ok {
		return nil
	}

	_, err := mail.ParseAddress(str)
	if err != nil {
		return fmt.Errorf("field %s has invalid email format: %s", field.Name, str)
	}

	return nil
}

// URLValidator validates URL format
type URLValidator struct{}

func NewURLValidator() *URLValidator {
	return &URLValidator{}
}

func (v *URLValidator) Name() string {
	return "url"
}

func (v *URLValidator) Validate(field *core.Field, value interface{}) error {
	nameLower := strings.ToLower(field.Name)
	if !strings.Contains(nameLower, "url") && !strings.Contains(nameLower, "website") && !strings.Contains(nameLower, "link") {
		return nil
	}

	if value == nil {
		return nil
	}

	str, ok := value.(string)
	if !ok {
		return nil
	}

	_, err := url.ParseRequestURI(str)
	if err != nil {
		return fmt.Errorf("field %s has invalid URL format: %s", field.Name, str)
	}

	return nil
}

// EnumValidator ensures values are within allowed enum values
type EnumValidator struct{}

func NewEnumValidator() *EnumValidator {
	return &EnumValidator{}
}

func (v *EnumValidator) Name() string {
	return "enum"
}

func (v *EnumValidator) Validate(field *core.Field, value interface{}) error {
	if len(field.EnumValues) == 0 {
		return nil
	}

	if value == nil {
		return nil
	}

	str, ok := value.(string)
	if !ok {
		return nil
	}

	for _, allowed := range field.EnumValues {
		if str == allowed {
			return nil
		}
	}

	return fmt.Errorf("field %s value '%s' not in allowed enum values: %v", field.Name, str, field.EnumValues)
}

// PhoneValidator validates phone number format
type PhoneValidator struct {
	pattern *regexp.Regexp
}

func NewPhoneValidator() *PhoneValidator {
	// Basic phone pattern - allows various formats
	pattern := regexp.MustCompile(`^[\d\s\-\+\(\)\.]{7,20}$`)
	return &PhoneValidator{pattern: pattern}
}

func (v *PhoneValidator) Name() string {
	return "phone"
}

func (v *PhoneValidator) Validate(field *core.Field, value interface{}) error {
	nameLower := strings.ToLower(field.Name)
	if !strings.Contains(nameLower, "phone") && !strings.Contains(nameLower, "mobile") && !strings.Contains(nameLower, "tel") {
		return nil
	}

	if value == nil {
		return nil
	}

	str, ok := value.(string)
	if !ok {
		return nil
	}

	if !v.pattern.MatchString(str) {
		return fmt.Errorf("field %s has invalid phone format: %s", field.Name, str)
	}

	return nil
}

// UUIDValidator validates UUID format
type UUIDValidator struct {
	pattern *regexp.Regexp
}

func NewUUIDValidator() *UUIDValidator {
	pattern := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	return &UUIDValidator{pattern: pattern}
}

func (v *UUIDValidator) Name() string {
	return "uuid"
}

func (v *UUIDValidator) Validate(field *core.Field, value interface{}) error {
	if field.Type != core.FieldTypeUUID {
		return nil
	}

	if value == nil {
		return nil
	}

	str, ok := value.(string)
	if !ok {
		return nil
	}

	if !v.pattern.MatchString(strings.ToLower(str)) {
		return fmt.Errorf("field %s has invalid UUID format: %s", field.Name, str)
	}

	return nil
}

// RangeValidator validates numeric values are within range
type RangeValidator struct {
	min, max float64
}

func NewRangeValidator(min, max float64) *RangeValidator {
	return &RangeValidator{min: min, max: max}
}

func (v *RangeValidator) Name() string {
	return "range"
}

func (v *RangeValidator) Validate(field *core.Field, value interface{}) error {
	if value == nil {
		return nil
	}

	var num float64
	switch val := value.(type) {
	case int:
		num = float64(val)
	case int64:
		num = float64(val)
	case float64:
		num = val
	case float32:
		num = float64(val)
	default:
		return nil
	}

	if num < v.min || num > v.max {
		return fmt.Errorf("field %s value %v is outside range [%v, %v]", field.Name, num, v.min, v.max)
	}

	return nil
}

// =============================================================================
// VALIDATOR CHAIN
// =============================================================================

// ValidatorChain runs multiple validators
type ValidatorChain struct {
	validators []core.Validator
}

func NewValidatorChain(validators ...core.Validator) *ValidatorChain {
	return &ValidatorChain{validators: validators}
}

func (c *ValidatorChain) Add(v core.Validator) {
	c.validators = append(c.validators, v)
}

func (c *ValidatorChain) Validate(field *core.Field, value interface{}) []error {
	var errs []error
	for _, v := range c.validators {
		if err := v.Validate(field, value); err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}

func (c *ValidatorChain) ValidateRow(collection *core.Collection, row map[string]interface{}) []error {
	var errs []error
	for _, field := range collection.Fields {
		value := row[field.Name]
		fieldErrs := c.Validate(field, value)
		errs = append(errs, fieldErrs...)
	}
	return errs
}

// DefaultValidatorChain returns a chain with all default validators
func DefaultValidatorChain() *ValidatorChain {
	return NewValidatorChain(
		NewNotNullValidator(),
		NewMaxLengthValidator(),
		NewTypeValidator(),
		NewEmailValidator(),
		NewURLValidator(),
		NewEnumValidator(),
		NewPhoneValidator(),
		NewUUIDValidator(),
	)
}
