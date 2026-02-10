package generator

import (
	"regexp"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

func TestDetectSemanticType(t *testing.T) {
	tests := []struct {
		fieldName string
		expected  SemanticFieldType
	}{
		{"id", SemanticID},
		{"user_id", SemanticID},
		{"userId", SemanticID},
		{"email", SemanticEmail},
		{"email_address", SemanticEmail},
		{"first_name", SemanticFirstName},
		{"firstName", SemanticFirstName},
		{"last_name", SemanticLastName},
		{"lastName", SemanticLastName},
		{"full_name", SemanticFullName},
		{"name", SemanticName},
		{"phone", SemanticPhone},
		{"phone_number", SemanticPhone},
		{"address", SemanticAddress},
		{"street", SemanticStreet},
		{"city", SemanticCity},
		{"state", SemanticState},
		{"country", SemanticCountry},
		{"zip_code", SemanticZipCode},
		{"postal_code", SemanticPostalCode},
		{"url", SemanticURL},
		{"website", SemanticWebsite},
		{"username", SemanticUsername},
		{"password", SemanticPassword},
		{"title", SemanticTitle},
		{"description", SemanticDescription},
		{"company", SemanticCompany},
		{"price", SemanticPrice},
		{"amount", SemanticAmount},
		{"quantity", SemanticQuantity},
		{"age", SemanticAge},
		{"created_at", SemanticCreatedAt},
		{"updated_at", SemanticUpdatedAt},
		{"birthday", SemanticBirthday},
		{"avatar", SemanticAvatar},
		{"image", SemanticImage},
		{"color", SemanticColor},
		{"status", SemanticStatus},
		{"category", SemanticCategory},
		{"slug", SemanticSlug},
		{"sku", SemanticSKU},
		{"latitude", SemanticLatitude},
		{"longitude", SemanticLongitude},
		{"currency", SemanticCurrency},
		{"language", SemanticLanguage},
		{"timezone", SemanticTimezone},
		{"ip_address", SemanticIPAddress},
		{"random_field", SemanticUnknown},
		{"xyz", SemanticUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.fieldName, func(t *testing.T) {
			result := DetectSemanticType(tt.fieldName)
			if result != tt.expected {
				t.Errorf("DetectSemanticType(%q) = %v, want %v", tt.fieldName, result, tt.expected)
			}
		})
	}
}

func TestGenerateBySemanticType(t *testing.T) {
	schema := &openapi3.Schema{Type: "string"}

	tests := []struct {
		semanticType SemanticFieldType
		validate     func(interface{}) bool
	}{
		{SemanticID, func(v interface{}) bool { return len(v.(string)) > 0 }},
		{SemanticEmail, func(v interface{}) bool { return strings.Contains(v.(string), "@") }},
		{SemanticFirstName, func(v interface{}) bool { return len(v.(string)) > 0 }},
		{SemanticLastName, func(v interface{}) bool { return len(v.(string)) > 0 }},
		{SemanticFullName, func(v interface{}) bool { return len(v.(string)) > 0 }},
		{SemanticPhone, func(v interface{}) bool { return len(v.(string)) > 0 }},
		{SemanticCity, func(v interface{}) bool { return len(v.(string)) > 0 }},
		{SemanticCountry, func(v interface{}) bool { return len(v.(string)) > 0 }},
		{SemanticURL, func(v interface{}) bool { return strings.HasPrefix(v.(string), "http") }},
		{SemanticUsername, func(v interface{}) bool { return len(v.(string)) > 0 }},
		{SemanticAge, func(v interface{}) bool {
			age := v.(int)
			return age >= 18 && age <= 80
		}},
		{SemanticPrice, func(v interface{}) bool { return v.(float64) > 0 }},
		{SemanticLatitude, func(v interface{}) bool {
			lat := v.(float64)
			return lat >= -90 && lat <= 90
		}},
		{SemanticLongitude, func(v interface{}) bool {
			lon := v.(float64)
			return lon >= -180 && lon <= 180
		}},
		{SemanticCurrency, func(v interface{}) bool {
			currencies := []string{"USD", "EUR", "GBP", "JPY", "CAD", "AUD", "CHF", "BRL"}
			for _, c := range currencies {
				if v.(string) == c {
					return true
				}
			}
			return false
		}},
		{SemanticColor, func(v interface{}) bool { return strings.HasPrefix(v.(string), "#") }},
		{SemanticIPAddress, func(v interface{}) bool {
			parts := strings.Split(v.(string), ".")
			return len(parts) == 4
		}},
	}

	for _, tt := range tests {
		t.Run(string(rune(tt.semanticType)), func(t *testing.T) {
			result := GenerateBySemanticType(tt.semanticType, schema)
			if result == nil {
				t.Errorf("GenerateBySemanticType returned nil for type %v", tt.semanticType)
				return
			}
			if !tt.validate(result) {
				t.Errorf("GenerateBySemanticType validation failed for type %v, got %v", tt.semanticType, result)
			}
		})
	}
}

func TestGenerateFromPattern(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		validate func(string) bool
	}{
		{
			name:    "simple literal",
			pattern: "hello",
			validate: func(s string) bool {
				return s == "hello"
			},
		},
		{
			name:    "digit class",
			pattern: `\d\d\d`,
			validate: func(s string) bool {
				matched, _ := regexp.MatchString(`^\d{3}$`, s)
				return matched
			},
		},
		{
			name:    "character class range",
			pattern: "[a-z][A-Z][0-9]",
			validate: func(s string) bool {
				matched, _ := regexp.MatchString(`^[a-z][A-Z][0-9]$`, s)
				return matched
			},
		},
		{
			name:    "quantifier exact",
			pattern: `a{3}`,
			validate: func(s string) bool {
				return s == "aaa"
			},
		},
		{
			name:    "quantifier range",
			pattern: `x{2,4}`,
			validate: func(s string) bool {
				return len(s) >= 2 && len(s) <= 4 && strings.Count(s, "x") == len(s)
			},
		},
		{
			name:    "alternation",
			pattern: "(cat|dog)",
			validate: func(s string) bool {
				return s == "cat" || s == "dog"
			},
		},
		{
			name:    "word characters",
			pattern: `\w\w\w\w`,
			validate: func(s string) bool {
				matched, _ := regexp.MatchString(`^\w{4}$`, s)
				return matched
			},
		},
		{
			name:    "optional",
			pattern: "ab?c",
			validate: func(s string) bool {
				return s == "ac" || s == "abc"
			},
		},
		{
			name:    "SKU pattern",
			pattern: "SKU-[A-Z]{3}-[0-9]{4}",
			validate: func(s string) bool {
				matched, _ := regexp.MatchString(`^SKU-[A-Z]{3}-[0-9]{4}$`, s)
				return matched
			},
		},
		{
			name:    "phone pattern",
			pattern: `\+1-\d{3}-\d{3}-\d{4}`,
			validate: func(s string) bool {
				matched, _ := regexp.MatchString(`^\+1-\d{3}-\d{3}-\d{4}$`, s)
				return matched
			},
		},
		{
			name:    "escaped characters",
			pattern: `\.\*\+`,
			validate: func(s string) bool {
				return s == ".*+"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GenerateFromPattern(tt.pattern)
			if err != nil {
				t.Errorf("GenerateFromPattern(%q) error = %v", tt.pattern, err)
				return
			}
			if !tt.validate(result) {
				t.Errorf("GenerateFromPattern(%q) = %q, validation failed", tt.pattern, result)
			}
		})
	}
}

func TestGenerateFromPattern_Errors(t *testing.T) {
	_, err := GenerateFromPattern("")
	if err == nil {
		t.Error("Expected error for empty pattern")
	}
}

func TestGenerateFromOneOf(t *testing.T) {
	schemas := []*openapi3.SchemaRef{
		{Value: &openapi3.Schema{Type: "string", Enum: []interface{}{"a"}}},
		{Value: &openapi3.Schema{Type: "string", Enum: []interface{}{"b"}}},
	}

	for i := 0; i < 10; i++ {
		result, err := GenerateFromOneOf(schemas)
		if err != nil {
			t.Fatalf("GenerateFromOneOf error: %v", err)
		}
		if result != "a" && result != "b" {
			t.Errorf("GenerateFromOneOf got %v, want 'a' or 'b'", result)
		}
	}
}

func TestGenerateFromOneOf_Empty(t *testing.T) {
	_, err := GenerateFromOneOf([]*openapi3.SchemaRef{})
	if err == nil {
		t.Error("Expected error for empty oneOf")
	}
}

func TestGenerateFromAnyOf(t *testing.T) {
	schemas := []*openapi3.SchemaRef{
		{Value: &openapi3.Schema{Type: "string", Enum: []interface{}{"x"}}},
		{Value: &openapi3.Schema{Type: "string", Enum: []interface{}{"y"}}},
	}

	for i := 0; i < 10; i++ {
		result, err := GenerateFromAnyOf(schemas)
		if err != nil {
			t.Fatalf("GenerateFromAnyOf error: %v", err)
		}
		if result != "x" && result != "y" {
			t.Errorf("GenerateFromAnyOf got %v, want 'x' or 'y'", result)
		}
	}
}

func TestGenerateFromAnyOf_Empty(t *testing.T) {
	_, err := GenerateFromAnyOf([]*openapi3.SchemaRef{})
	if err == nil {
		t.Error("Expected error for empty anyOf")
	}
}

func TestGenerateFromAllOf(t *testing.T) {
	schemas := []*openapi3.SchemaRef{
		{Value: &openapi3.Schema{
			Type: "object",
			Properties: openapi3.Schemas{
				"name": &openapi3.SchemaRef{Value: &openapi3.Schema{Type: "string", Enum: []interface{}{"test"}}},
			},
		}},
		{Value: &openapi3.Schema{
			Type: "object",
			Properties: openapi3.Schemas{
				"age": &openapi3.SchemaRef{Value: &openapi3.Schema{Type: "integer", Min: ptr(float64(25)), Max: ptr(float64(25))}},
			},
		}},
	}

	result, err := GenerateFromAllOf(schemas)
	if err != nil {
		t.Fatalf("GenerateFromAllOf error: %v", err)
	}

	obj, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map, got %T", result)
	}

	if obj["name"] != "test" {
		t.Errorf("Expected name='test', got %v", obj["name"])
	}
	if obj["age"] != int64(25) {
		t.Errorf("Expected age=25, got %v", obj["age"])
	}
}

func TestGenerateFromAllOf_Empty(t *testing.T) {
	_, err := GenerateFromAllOf([]*openapi3.SchemaRef{})
	if err == nil {
		t.Error("Expected error for empty allOf")
	}
}

func TestGenerateAdvancedData_WithSemanticDetection(t *testing.T) {
	schema := &openapi3.SchemaRef{
		Value: &openapi3.Schema{Type: "string"},
	}

	result, err := GenerateAdvancedData(schema, "email")
	if err != nil {
		t.Fatalf("GenerateAdvancedData error: %v", err)
	}

	str, ok := result.(string)
	if !ok {
		t.Fatalf("Expected string, got %T", result)
	}

	if !strings.Contains(str, "@") {
		t.Errorf("Expected email format, got %q", str)
	}
}

func TestGenerateAdvancedData_WithPattern(t *testing.T) {
	schema := &openapi3.SchemaRef{
		Value: &openapi3.Schema{
			Type:    "string",
			Pattern: "SKU-[A-Z]{3}-[0-9]{4}",
		},
	}

	result, err := GenerateAdvancedData(schema, "sku")
	if err != nil {
		t.Fatalf("GenerateAdvancedData error: %v", err)
	}

	str, ok := result.(string)
	if !ok {
		t.Fatalf("Expected string, got %T", result)
	}

	matched, _ := regexp.MatchString(`^SKU-[A-Z]{3}-[0-9]{4}$`, str)
	if !matched {
		t.Errorf("Expected SKU pattern match, got %q", str)
	}
}

func TestGenerateAdvancedData_WithOneOf(t *testing.T) {
	schema := &openapi3.SchemaRef{
		Value: &openapi3.Schema{
			OneOf: []*openapi3.SchemaRef{
				{Value: &openapi3.Schema{Type: "string", Enum: []interface{}{"option1"}}},
				{Value: &openapi3.Schema{Type: "string", Enum: []interface{}{"option2"}}},
			},
		},
	}

	result, err := GenerateAdvancedData(schema, "")
	if err != nil {
		t.Fatalf("GenerateAdvancedData error: %v", err)
	}

	if result != "option1" && result != "option2" {
		t.Errorf("Expected 'option1' or 'option2', got %v", result)
	}
}

func TestGenerateAdvancedData_Object(t *testing.T) {
	schema := &openapi3.SchemaRef{
		Value: &openapi3.Schema{
			Type: "object",
			Properties: openapi3.Schemas{
				"email":      &openapi3.SchemaRef{Value: &openapi3.Schema{Type: "string"}},
				"first_name": &openapi3.SchemaRef{Value: &openapi3.Schema{Type: "string"}},
				"age":        &openapi3.SchemaRef{Value: &openapi3.Schema{Type: "integer"}},
			},
		},
	}

	result, err := GenerateAdvancedData(schema, "")
	if err != nil {
		t.Fatalf("GenerateAdvancedData error: %v", err)
	}

	obj, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map, got %T", result)
	}

	email, ok := obj["email"].(string)
	if !ok || !strings.Contains(email, "@") {
		t.Errorf("Expected email with @, got %v", obj["email"])
	}

	firstName, ok := obj["first_name"].(string)
	if !ok || len(firstName) == 0 {
		t.Errorf("Expected non-empty first_name, got %v", obj["first_name"])
	}
}

func TestMakeUnique(t *testing.T) {
	arr := []interface{}{"a", "b", "a", "c", "b", "d"}
	result := makeUnique(arr)

	if len(result) != 4 {
		t.Errorf("Expected 4 unique items, got %d", len(result))
	}

	expected := map[string]bool{"a": true, "b": true, "c": true, "d": true}
	for _, v := range result {
		if !expected[v.(string)] {
			t.Errorf("Unexpected value %v", v)
		}
	}
}

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"firstName", "first_name"},
		{"LastName", "last_name"},
		{"emailAddress", "email_address"},
		{"ID", "i_d"},
		{"simple", "simple"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ToSnakeCase(tt.input)
			if result != tt.expected {
				t.Errorf("ToSnakeCase(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func ptr(f float64) *float64 {
	return &f
}
