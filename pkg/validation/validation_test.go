package validation

import (
	"testing"
)

func TestMaxLength(t *testing.T) {
	rule := MaxLength(5)

	if !rule.Test("hello") {
		t.Error("should allow 5 character string")
	}

	if !rule.Test("hi") {
		t.Error("should allow shorter string")
	}

	if rule.Test("hello world") {
		t.Error("should reject string over max length")
	}
}

func TestMinLength(t *testing.T) {
	rule := MinLength(3)

	if !rule.Test("hello") {
		t.Error("should allow string over min length")
	}

	if rule.Test("hi") {
		t.Error("should reject string under min length")
	}
}

func TestNotEmpty(t *testing.T) {
	rule := NotEmpty()

	if !rule.Test("content") {
		t.Error("should allow non-empty string")
	}

	if rule.Test("") {
		t.Error("should reject empty string")
	}

	if rule.Test("   ") {
		t.Error("should reject whitespace-only string")
	}
}

func TestPattern(t *testing.T) {
	rule := Pattern(`^\d{3}-\d{3}-\d{4}$`)

	if !rule.Test("123-456-7890") {
		t.Error("should match phone number pattern")
	}

	if rule.Test("123-456-789") {
		t.Error("should reject invalid pattern")
	}
}

func TestAlphanumeric(t *testing.T) {
	rule := Alphanumeric()

	if !rule.Test("hello123") {
		t.Error("should allow alphanumeric string")
	}

	if rule.Test("hello-123") {
		t.Error("should reject string with dash")
	}

	if rule.Test("hello_123") {
		t.Error("should reject string with underscore")
	}
}

func TestAlphanumericWithDashes(t *testing.T) {
	rule := AlphanumericWithDashes()

	if !rule.Test("hello-world_123") {
		t.Error("should allow alphanumeric with dashes/underscores")
	}

	if rule.Test("hello@world") {
		t.Error("should reject string with special chars")
	}
}

func TestNoSQLInjection(t *testing.T) {
	rule := NoSQLInjection()

	if !rule.Test("normal_input") {
		t.Error("should allow normal input")
	}

	if rule.Test("'; DROP TABLE users;--") {
		t.Error("should reject SQL injection pattern")
	}

	if rule.Test("1' OR '1'='1") {
		t.Error("should reject SQL injection pattern")
	}

	if rule.Test("UNION SELECT") {
		t.Error("should reject UNION SELECT pattern")
	}
}

func TestNoXSS(t *testing.T) {
	rule := NoXSS()

	if !rule.Test("normal text content") {
		t.Error("should allow normal text")
	}

	if rule.Test("<script>alert('xss')</script>") {
		t.Error("should reject script tag")
	}

	if rule.Test("<img onerror='alert(1)'>") {
		t.Error("should reject img with onerror")
	}

	if rule.Test("javascript:alert('xss')") {
		t.Error("should reject javascript: URI")
	}

	if rule.Test("<svg onload='alert(1)'>") {
		t.Error("should reject svg with onload")
	}
}

func TestEmail(t *testing.T) {
	rule := Email()

	if !rule.Test("user@example.com") {
		t.Error("should allow valid email")
	}

	if rule.Test("invalid.email") {
		t.Error("should reject email without @")
	}

	if rule.Test("@example.com") {
		t.Error("should reject email without local part")
	}
}

func TestUUID(t *testing.T) {
	rule := UUID()

	if !rule.Test("550e8400-e29b-41d4-a716-446655440000") {
		t.Error("should allow valid UUID")
	}

	if !rule.Test("550E8400-E29B-41D4-A716-446655440000") {
		t.Error("should allow UUID with uppercase")
	}

	if rule.Test("550e8400-e29b-41d4-a716-44665544000") {
		t.Error("should reject invalid UUID (too short)")
	}

	if rule.Test("550e8400-e29b-41d4-a716-44665544000g") {
		t.Error("should reject UUID with invalid character")
	}
}

func TestWhitelist(t *testing.T) {
	rule := Whitelist("admin", "user", "guest")

	if !rule.Test("admin") {
		t.Error("should allow whitelisted value")
	}

	if rule.Test("superuser") {
		t.Error("should reject non-whitelisted value")
	}
}

func TestNumericRange(t *testing.T) {
	rule := NumericRange(1, 100)

	if !rule.Test(int64(50)) {
		t.Error("should allow value in range")
	}

	if !rule.Test(int(50)) {
		t.Error("should allow int value in range")
	}

	if rule.Test(int64(101)) {
		t.Error("should reject value above range")
	}

	if rule.Test(int64(0)) {
		t.Error("should reject value below range")
	}
}

func TestHasPrefix(t *testing.T) {
	rule := HasPrefix("dh1")

	if !rule.Test("dh1abc123def") {
		t.Error("should allow string with prefix")
	}

	if rule.Test("abc1dh1def") {
		t.Error("should reject string without prefix")
	}
}

func TestHasSuffix(t *testing.T) {
	rule := HasSuffix(".example.com")

	if !rule.Test("api.example.com") {
		t.Error("should allow string with suffix")
	}

	if rule.Test("example.com.proxy") {
		t.Error("should reject string without suffix")
	}
}

func TestNoControl(t *testing.T) {
	rule := NoControl()

	if !rule.Test("normal string") {
		t.Error("should allow normal string")
	}

	if rule.Test("string\nwith\nnewlines") {
		t.Error("should reject string with newlines")
	}

	if rule.Test("string\twith\ttabs") {
		t.Error("should reject string with tabs")
	}
}

func TestValidator(t *testing.T) {
	validator := NewValidator().
		Add(NotEmpty()).
		Add(MaxLength(20)).
		Add(Alphanumeric())

	if err := validator.Validate("hello123"); err != nil {
		t.Errorf("should validate correct string: %v", err)
	}

	if err := validator.Validate(""); err == nil {
		t.Error("should reject empty string")
	}

	if err := validator.Validate("this-is-a-very-long-string-with-special-chars"); err == nil {
		t.Error("should reject string that fails multiple rules")
	}
}

func TestValidatorField(t *testing.T) {
	validator := NewValidator().
		Add(NotEmpty()).
		Add(MaxLength(20))

	if err := validator.ValidateField("username", "validuser"); err != nil {
		t.Errorf("should validate field: %v", err)
	}

	if err := validator.ValidateField("username", ""); err == nil {
		t.Error("should reject empty field")
	}

	if err := validator.ValidateField("username", "a"); err == nil {
		// Single character is allowed
	}
}

func TestFieldValidator(t *testing.T) {
	fv := NewFieldValidator().
		AddField("username", NewValidator().
			Add(NotEmpty()).
			Add(MaxLength(20)).
			Add(AlphanumericWithDashes())).
		AddField("email", NewValidator().
			Add(NotEmpty()).
			Add(Email()))

	values := map[string]interface{}{
		"username": "valid-user",
		"email":    "user@example.com",
	}

	errors := fv.Validate(values)
	if len(errors) > 0 {
		t.Errorf("should validate correct fields: %v", errors)
	}

	values["username"] = "invalid@user"
	errors = fv.Validate(values)
	if len(errors) == 0 {
		t.Error("should reject invalid username")
	}

	values["email"] = "not-an-email"
	errors = fv.Validate(values)
	if len(errors) < 2 {
		t.Error("should have at least 2 validation errors")
	}
}

func TestIDValidator(t *testing.T) {
	validator := IDValidator()

	if err := validator.Validate("dh1abcdefghijklmnopqrstuvwxyz"); err != nil {
		t.Errorf("should validate dh1 ID: %v", err)
	}

	if err := validator.Validate("invalid-id"); err == nil {
		t.Error("should reject invalid dh1 ID")
	}
}

func TestNodeIDValidator(t *testing.T) {
	validator := NodeIDValidator()

	if err := validator.Validate("node-1"); err != nil {
		t.Errorf("should validate node ID: %v", err)
	}

	if err := validator.Validate("node_2_test"); err != nil {
		t.Errorf("should validate node ID with underscores: %v", err)
	}

	if err := validator.Validate("node@invalid"); err == nil {
		t.Error("should reject node ID with special characters")
	}
}

func TestPublicKeyValidator(t *testing.T) {
	validator := PublicKeyValidator()

	if err := validator.Validate("abc123def456-_"); err != nil {
		t.Errorf("should validate base64 key: %v", err)
	}

	if err := validator.Validate("abc!@#$"); err == nil {
		t.Error("should reject key with invalid characters")
	}
}

func TestActionValidator(t *testing.T) {
	validator := ActionValidator()

	if err := validator.Validate("create-workload"); err != nil {
		t.Errorf("should validate action: %v", err)
	}

	if err := validator.Validate("scale_replicas"); err != nil {
		t.Errorf("should validate action with underscore: %v", err)
	}

	if err := validator.Validate("action@invalid"); err == nil {
		t.Error("should reject action with special characters")
	}
}

func BenchmarkValidate(b *testing.B) {
	validator := NewValidator().
		Add(NotEmpty()).
		Add(MaxLength(100)).
		Add(AlphanumericWithDashes()).
		Add(NoSQLInjection()).
		Add(NoXSS())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		validator.Validate("valid-input-123")
	}
}

func BenchmarkFieldValidate(b *testing.B) {
	fv := NewFieldValidator().
		AddField("username", NewValidator().
			Add(NotEmpty()).
			Add(MaxLength(20)).
			Add(AlphanumericWithDashes())).
		AddField("email", NewValidator().
			Add(NotEmpty()).
			Add(Email()))

	values := map[string]interface{}{
		"username": "valid-user",
		"email":    "user@example.com",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fv.Validate(values)
	}
}
