package generator

import (
	"regexp"
	"testing"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerator_Generate(t *testing.T) {
	g := New()

	t.Run("string types", func(t *testing.T) {
		tests := []struct {
			name     string
			format   string
			validate func(t *testing.T, value interface{})
		}{
			{
				name:   "email",
				format: "email",
				validate: func(t *testing.T, value interface{}) {
					str, ok := value.(string)
					require.True(t, ok)
					assert.Contains(t, str, "@")
					assert.Contains(t, str, ".")
				},
			},
			{
				name:   "date-time",
				format: "date-time",
				validate: func(t *testing.T, value interface{}) {
					str, ok := value.(string)
					require.True(t, ok)
					_, err := time.Parse(time.RFC3339, str)
					assert.NoError(t, err)
				},
			},
			{
				name:   "date",
				format: "date",
				validate: func(t *testing.T, value interface{}) {
					str, ok := value.(string)
					require.True(t, ok)
					_, err := time.Parse("2006-01-02", str)
					assert.NoError(t, err)
				},
			},
			{
				name:   "uuid",
				format: "uuid",
				validate: func(t *testing.T, value interface{}) {
					str, ok := value.(string)
					require.True(t, ok)
					assert.Regexp(t, regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`), str)
				},
			},
			{
				name:   "uri",
				format: "uri",
				validate: func(t *testing.T, value interface{}) {
					str, ok := value.(string)
					require.True(t, ok)
					assert.Contains(t, str, "://")
				},
			},
			{
				name:   "ipv4",
				format: "ipv4",
				validate: func(t *testing.T, value interface{}) {
					str, ok := value.(string)
					require.True(t, ok)
					assert.Regexp(t, regexp.MustCompile(`^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}$`), str)
				},
			},
			{
				name:   "ipv6",
				format: "ipv6",
				validate: func(t *testing.T, value interface{}) {
					str, ok := value.(string)
					require.True(t, ok)
					assert.Contains(t, str, ":")
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				schema := &openapi3.Schema{
					Type:   "string",
					Format: tt.format,
				}
				value, err := g.Generate(schema, nil)
				require.NoError(t, err)
				tt.validate(t, value)
			})
		}
	})

	t.Run("number types", func(t *testing.T) {
		t.Run("integer with bounds", func(t *testing.T) {
			min := float64(10)
			max := float64(20)
			schema := &openapi3.Schema{
				Type: "integer",
				Min:  &min,
				Max:  &max,
			}
			value, err := g.Generate(schema, nil)
			require.NoError(t, err)
			num, ok := value.(int64)
			require.True(t, ok)
			assert.GreaterOrEqual(t, num, int64(min))
			assert.LessOrEqual(t, num, int64(max))
		})

		t.Run("number with bounds", func(t *testing.T) {
			min := float64(1.5)
			max := float64(2.5)
			schema := &openapi3.Schema{
				Type: "number",
				Min:  &min,
				Max:  &max,
			}
			value, err := g.Generate(schema, nil)
			require.NoError(t, err)
			num, ok := value.(float64)
			require.True(t, ok)
			assert.GreaterOrEqual(t, num, min)
			assert.LessOrEqual(t, num, max)
		})
	})

	t.Run("array type", func(t *testing.T) {
		minItems := uint64(2)
		maxItems := uint64(4)
		schema := &openapi3.Schema{
			Type:     "array",
			MinItems: minItems,
			MaxItems: &maxItems,
			Items: &openapi3.SchemaRef{
				Value: &openapi3.Schema{
					Type: "string",
				},
			},
		}
		value, err := g.Generate(schema, nil)
		require.NoError(t, err)
		arr, ok := value.([]interface{})
		require.True(t, ok)
		assert.GreaterOrEqual(t, len(arr), int(minItems))
		assert.LessOrEqual(t, len(arr), int(maxItems))
		for _, item := range arr {
			_, ok := item.(string)
			assert.True(t, ok)
		}
	})

	t.Run("object type", func(t *testing.T) {
		schema := &openapi3.Schema{
			Type: "object",
			Properties: map[string]*openapi3.SchemaRef{
				"name": {
					Value: &openapi3.Schema{
						Type: "string",
					},
				},
				"age": {
					Value: &openapi3.Schema{
						Type: "integer",
					},
				},
			},
			Required: []string{"name"},
		}
		value, err := g.Generate(schema, nil)
		require.NoError(t, err)
		obj, ok := value.(map[string]interface{})
		require.True(t, ok)
		assert.Contains(t, obj, "name")
		assert.Contains(t, obj, "age")
		_, ok = obj["name"].(string)
		assert.True(t, ok)
		_, ok = obj["age"].(int64)
		assert.True(t, ok)
	})
}

func TestGenerator_CustomFuncs(t *testing.T) {
	g := New()

	// Register a custom function for generating phone numbers
	g.RegisterCustomFunc("phone", func(schema *openapi3.Schema, context *GenerationContext) (interface{}, error) {
		return "+1-555-0123", nil
	})

	schema := &openapi3.Schema{
		Type:   "string",
		Format: "phone",
	}

	value, err := g.Generate(schema, nil)
	require.NoError(t, err)
	assert.Equal(t, "+1-555-0123", value)
}

func TestGenerator_GenerationRules(t *testing.T) {
	g := New()

	// Register a rule for generating consistent user IDs
	rule := GenerationRule{
		Pattern: "users.*.id",
		Generator: func(schema *openapi3.Schema, context *GenerationContext) (interface{}, error) {
			return "USR-" + context.ParentID, nil
		},
		Cache: true,
	}
	err := g.RegisterRule(rule)
	require.NoError(t, err)

	schema := &openapi3.Schema{
		Type: "string",
	}

	context := &GenerationContext{
		Path:     "users.123.id",
		ParentID: "123",
	}

	value, err := g.Generate(schema, context)
	require.NoError(t, err)
	assert.Equal(t, "USR-123", value)
}

func TestGenerator_Relationships(t *testing.T) {
	g := New()

	userSchema := &openapi3.Schema{
		Type: "object",
		Properties: map[string]*openapi3.SchemaRef{
			"id": {
				Value: &openapi3.Schema{
					Type: "string",
				},
			},
			"name": {
				Value: &openapi3.Schema{
					Type: "string",
				},
			},
		},
	}

	t.Run("one_to_one relationship", func(t *testing.T) {
		value, err := g.GenerateRelated(userSchema, "profile", "123", map[string]string{
			"user": "one_to_one",
		})
		require.NoError(t, err)
		obj, ok := value.(map[string]interface{})
		require.True(t, ok)
		assert.Contains(t, obj, "profile_id")
		assert.Equal(t, "123", obj["profile_id"])
	})

	t.Run("one_to_many relationship", func(t *testing.T) {
		value, err := g.GenerateRelated(userSchema, "post", "123", map[string]string{
			"comments": "one_to_many",
		})
		require.NoError(t, err)
		arr, ok := value.([]interface{})
		require.True(t, ok)
		assert.GreaterOrEqual(t, len(arr), 1)
		assert.LessOrEqual(t, len(arr), 5)
		for _, item := range arr {
			obj, ok := item.(map[string]interface{})
			require.True(t, ok)
			assert.Contains(t, obj, "post_id")
			assert.Equal(t, "123", obj["post_id"])
		}
	})
}

func TestGenerator_Cache(t *testing.T) {
	g := New()

	// Create a schema that should generate consistent values
	schema := &openapi3.Schema{
		Type: "object",
		Properties: map[string]*openapi3.SchemaRef{
			"id": {
				Value: &openapi3.Schema{
					Type: "string",
				},
			},
		},
	}

	// Register a rule that caches values
	rule := GenerationRule{
		Pattern: "users.*.id",
		Generator: func(schema *openapi3.Schema, context *GenerationContext) (interface{}, error) {
			return "CACHED-" + context.Path, nil
		},
		Cache: true,
	}
	err := g.RegisterRule(rule)
	require.NoError(t, err)

	// Generate twice with the same path
	context := &GenerationContext{
		Path: "users.test.id",
	}

	value1, err := g.Generate(schema, context)
	require.NoError(t, err)

	value2, err := g.Generate(schema, context)
	require.NoError(t, err)

	// Values should be the same
	assert.Equal(t, value1, value2)
}

func TestGenerator_Dependencies(t *testing.T) {
	g := New()

	// Register a rule that depends on another field
	rule := GenerationRule{
		Pattern: "users.*.fullName",
		Generator: func(schema *openapi3.Schema, context *GenerationContext) (interface{}, error) {
			firstName := context.Cache["users.*.firstName"].(string)
			lastName := context.Cache["users.*.lastName"].(string)
			return firstName + " " + lastName, nil
		},
		Dependencies: []string{"users.*.firstName", "users.*.lastName"},
	}
	err := g.RegisterRule(rule)
	require.NoError(t, err)

	schema := &openapi3.Schema{
		Type: "string",
	}

	context := &GenerationContext{
		Path: "users.123.fullName",
		Cache: map[string]interface{}{
			"users.*.firstName": "John",
			"users.*.lastName":  "Doe",
		},
	}

	value, err := g.Generate(schema, context)
	require.NoError(t, err)
	assert.Equal(t, "John Doe", value)
}

func TestGenerator_Validation(t *testing.T) {
	g := New()

	// Register a rule with validation
	rule := GenerationRule{
		Pattern: "users.*.age",
		Generator: func(schema *openapi3.Schema, context *GenerationContext) (interface{}, error) {
			return 25, nil
		},
		Validator: func(value interface{}) error {
			age := value.(int)
			if age < 0 || age > 120 {
				return assert.AnError
			}
			return nil
		},
	}
	err := g.RegisterRule(rule)
	require.NoError(t, err)

	schema := &openapi3.Schema{
		Type: "integer",
	}

	context := &GenerationContext{
		Path: "users.123.age",
	}

	value, err := g.Generate(schema, context)
	require.NoError(t, err)
	assert.Equal(t, 25, value)
} 