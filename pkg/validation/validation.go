// Package validation provides whitelist-based input validation for dh/v1.
//
// This package implements:
// - Whitelist-based validation rules
// - Size limits enforcement
// - Type validation
// - SQL injection prevention
// - XSS prevention
// - Pattern matching
package validation

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

// ValidationError describes a validation failure.
type ValidationError struct {
	Field   string
	Message string
	Value   interface{}
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("validation error on field %q: %s (value: %v)", e.Field, e.Message, e.Value)
}

// Rule defines a validation rule.
type Rule struct {
	Name    string
	Test    func(value interface{}) bool
	Message string
}

// Validator chains validation rules.
type Validator struct {
	rules []Rule
}

// NewValidator creates a new validator.
func NewValidator() *Validator {
	return &Validator{
		rules: make([]Rule, 0),
	}
}

// Add adds a validation rule.
func (v *Validator) Add(rule Rule) *Validator {
	v.rules = append(v.rules, rule)
	return v
}

// ValidateString validates a string value against all rules.
func (v *Validator) ValidateString(value string) error {
	return v.Validate(value)
}

// Validate validates a value against all rules.
func (v *Validator) Validate(value interface{}) error {
	for _, rule := range v.rules {
		if !rule.Test(value) {
			return fmt.Errorf("%s: %s", rule.Name, rule.Message)
		}
	}
	return nil
}

// ValidateField validates a named field.
func (v *Validator) ValidateField(fieldName string, value interface{}) error {
	for _, rule := range v.rules {
		if !rule.Test(value) {
			return ValidationError{
				Field:   fieldName,
				Message: rule.Message,
				Value:   value,
			}
		}
	}
	return nil
}

// Common validation rules

// MaxLength creates a rule that checks string length.
func MaxLength(max int) Rule {
	return Rule{
		Name: "MaxLength",
		Test: func(value interface{}) bool {
			s, ok := value.(string)
			return ok && len(s) <= max
		},
		Message: fmt.Sprintf("must be at most %d characters", max),
	}
}

// MinLength creates a rule that checks minimum string length.
func MinLength(min int) Rule {
	return Rule{
		Name: "MinLength",
		Test: func(value interface{}) bool {
			s, ok := value.(string)
			return ok && len(s) >= min
		},
		Message: fmt.Sprintf("must be at least %d characters", min),
	}
}

// NotEmpty checks that a string is not empty.
func NotEmpty() Rule {
	return Rule{
		Name: "NotEmpty",
		Test: func(value interface{}) bool {
			s, ok := value.(string)
			return ok && strings.TrimSpace(s) != ""
		},
		Message: "must not be empty",
	}
}

// Pattern checks if value matches a regex pattern.
func Pattern(pattern string) Rule {
	re := regexp.MustCompile(pattern)
	return Rule{
		Name: "Pattern",
		Test: func(value interface{}) bool {
			s, ok := value.(string)
			return ok && re.MatchString(s)
		},
		Message: fmt.Sprintf("must match pattern %s", pattern),
	}
}

// Alphanumeric checks if string contains only alphanumeric characters.
func Alphanumeric() Rule {
	return Rule{
		Name: "Alphanumeric",
		Test: func(value interface{}) bool {
			s, ok := value.(string)
			if !ok {
				return false
			}
			for _, r := range s {
				if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
					return false
				}
			}
			return true
		},
		Message: "must contain only alphanumeric characters",
	}
}

// AlphanumericWithDashes checks if string is alphanumeric with dashes/underscores.
func AlphanumericWithDashes() Rule {
	return Rule{
		Name: "AlphanumericWithDashes",
		Test: func(value interface{}) bool {
			s, ok := value.(string)
			if !ok {
				return false
			}
			for _, r := range s {
				if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '-' && r != '_' {
					return false
				}
			}
			return true
		},
		Message: "must contain only alphanumeric characters, dashes, and underscores",
	}
}

// NoSQLInjection checks for common SQL injection patterns.
func NoSQLInjection() Rule {
	return Rule{
		Name: "NoSQLInjection",
		Test: func(value interface{}) bool {
			s, ok := value.(string)
			if !ok {
				return true
			}

			// Check for common SQL keywords used in injection
			injectionPatterns := []string{
				"'",
				"\"",
				"--",
				";",
				"/*",
				"*/",
				"xp_",
				"sp_",
				"exec(",
				"execute(",
				"union",
				"select",
				"insert",
				"update",
				"delete",
				"drop",
				"create",
				"alter",
				"truncate",
			}

			lower := strings.ToLower(s)
			for _, pattern := range injectionPatterns {
				if strings.Contains(lower, pattern) {
					return false
				}
			}
			return true
		},
		Message: "contains potentially dangerous SQL patterns",
	}
}

// NoXSS checks for common XSS patterns.
func NoXSS() Rule {
	return Rule{
		Name: "NoXSS",
		Test: func(value interface{}) bool {
			s, ok := value.(string)
			if !ok {
				return true
			}

			// Check for common XSS patterns
			xssPatterns := []string{
				"<script",
				"</script",
				"<iframe",
				"</iframe",
				"<svg",
				"</svg",
				"<img",
				"onerror=",
				"onload=",
				"onclick=",
				"onmouseover=",
				"<body",
				"</body",
				"javascript:",
				"data:",
				"vbscript:",
			}

			lower := strings.ToLower(s)
			for _, pattern := range xssPatterns {
				if strings.Contains(lower, pattern) {
					return false
				}
			}
			return true
		},
		Message: "contains potentially dangerous HTML/XSS patterns",
	}
}

// Email validates email format.
func Email() Rule {
	// Simple email pattern, not RFC 5322 complete
	pattern := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return Rule{
		Name: "Email",
		Test: func(value interface{}) bool {
			s, ok := value.(string)
			return ok && pattern.MatchString(s)
		},
		Message: "must be a valid email address",
	}
}

// UUID validates UUID format.
func UUID() Rule {
	pattern := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	return Rule{
		Name: "UUID",
		Test: func(value interface{}) bool {
			s, ok := value.(string)
			return ok && pattern.MatchString(strings.ToLower(s))
		},
		Message: "must be a valid UUID",
	}
}

// Whitelist checks if value is in a whitelist.
func Whitelist(allowed ...string) Rule {
	allowMap := make(map[string]bool)
	for _, v := range allowed {
		allowMap[v] = true
	}

	return Rule{
		Name: "Whitelist",
		Test: func(value interface{}) bool {
			s, ok := value.(string)
			if !ok {
				return false
			}
			return allowMap[s]
		},
		Message: fmt.Sprintf("must be one of: %s", strings.Join(allowed, ", ")),
	}
}

// NumericRange validates integer is within range.
func NumericRange(min, max int64) Rule {
	return Rule{
		Name: "NumericRange",
		Test: func(value interface{}) bool {
			switch v := value.(type) {
			case int:
				return int64(v) >= min && int64(v) <= max
			case int64:
				return v >= min && v <= max
			case int32:
				return int64(v) >= min && int64(v) <= max
			default:
				return false
			}
		},
		Message: fmt.Sprintf("must be between %d and %d", min, max),
	}
}

// HasPrefix checks if string starts with prefix.
func HasPrefix(prefix string) Rule {
	return Rule{
		Name: "HasPrefix",
		Test: func(value interface{}) bool {
			s, ok := value.(string)
			return ok && strings.HasPrefix(s, prefix)
		},
		Message: fmt.Sprintf("must start with %q", prefix),
	}
}

// HasSuffix checks if string ends with suffix.
func HasSuffix(suffix string) Rule {
	return Rule{
		Name: "HasSuffix",
		Test: func(value interface{}) bool {
			s, ok := value.(string)
			return ok && strings.HasSuffix(s, suffix)
		},
		Message: fmt.Sprintf("must end with %q", suffix),
	}
}

// NoControl checks that string doesn't contain control characters.
func NoControl() Rule {
	return Rule{
		Name: "NoControl",
		Test: func(value interface{}) bool {
			s, ok := value.(string)
			if !ok {
				return true
			}
			for _, r := range s {
				if unicode.IsControl(r) {
					return false
				}
			}
			return true
		},
		Message: "must not contain control characters",
	}
}

// FieldValidator validates multiple named fields.
type FieldValidator struct {
	validators map[string]*Validator
}

// NewFieldValidator creates a new field validator.
func NewFieldValidator() *FieldValidator {
	return &FieldValidator{
		validators: make(map[string]*Validator),
	}
}

// AddField adds a validator for a field.
func (fv *FieldValidator) AddField(name string, validator *Validator) *FieldValidator {
	fv.validators[name] = validator
	return fv
}

// Validate validates a map of values.
func (fv *FieldValidator) Validate(values map[string]interface{}) []ValidationError {
	var errors []ValidationError

	for fieldName, validator := range fv.validators {
		if value, ok := values[fieldName]; ok {
			if err := validator.ValidateField(fieldName, value); err != nil {
				if ve, ok := err.(ValidationError); ok {
					errors = append(errors, ve)
				} else {
					errors = append(errors, ValidationError{
						Field:   fieldName,
						Message: err.Error(),
						Value:   value,
					})
				}
			}
		}
	}

	return errors
}

// Helper function to create common validators

// IDValidator creates a validator for dh1 identifiers.
func IDValidator() *Validator {
	return NewValidator().
		Add(NotEmpty()).
		Add(Pattern(`^dh1[a-z2-7]{26}$`))
}

// NodeIDValidator creates a validator for node identifiers.
func NodeIDValidator() *Validator {
	return NewValidator().
		Add(NotEmpty()).
		Add(MaxLength(50)).
		Add(AlphanumericWithDashes())
}

// PublicKeyValidator creates a validator for base64 public keys.
func PublicKeyValidator() *Validator {
	return NewValidator().
		Add(NotEmpty()).
		Add(MaxLength(100)).
		Add(Pattern(`^[A-Za-z0-9_-]+$`))
}

// ActionValidator creates a validator for action names.
func ActionValidator() *Validator {
	return NewValidator().
		Add(NotEmpty()).
		Add(MaxLength(100)).
		Add(AlphanumericWithDashes())
}
