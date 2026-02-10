package validation

import (
	"encoding/json"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/assert"
)

// Helper functions for creating test schemas

func createSchema(schemaType string) *openapi3.Schema {
	schema := &openapi3.Schema{}
	schema.Type = schemaType
	return schema
}

func createSchemaWithFormat(schemaType, format string) *openapi3.Schema {
	schema := &openapi3.Schema{}
	schema.Type = schemaType
	schema.Format = format
	return schema
}

func createSchemaWithMinLength(minLength uint64) *openapi3.Schema {
	schema := &openapi3.Schema{}
	schema.Type = "string"
	schema.MinLength = minLength
	return schema
}

func createObjectSchema(properties map[string]*openapi3.SchemaRef, required []string) *openapi3.Schema {
	schema := &openapi3.Schema{}
	schema.Type = "object"
	schema.Properties = properties
	schema.Required = required
	return schema
}

func createSchemaWithPattern(pattern string) *openapi3.Schema {
	schema := &openapi3.Schema{}
	schema.Type = "string"
	schema.Pattern = pattern
	return schema
}

func createArraySchema(itemsSchema *openapi3.Schema) *openapi3.Schema {
	schema := &openapi3.Schema{}
	schema.Type = "array"
	schema.Items = &openapi3.SchemaRef{Value: itemsSchema}
	return schema
}

func createArraySchemaWithMinItems(itemsSchema *openapi3.Schema, minItems uint64) *openapi3.Schema {
	schema := &openapi3.Schema{}
	schema.Type = "array"
	schema.Items = &openapi3.SchemaRef{Value: itemsSchema}
	schema.MinItems = minItems
	return schema
}

func TestValidateSchema(t *testing.T) {
	validator := NewRequestValidator(&openapi3.T{})

	t.Run("ValidateSchema_SimpleTypes", func(t *testing.T) {
		tests := []struct {
			name          string
			schema        *openapi3.Schema
			data          interface{}
			expectedValid bool
		}{
			{
				name:          "Valid String",
				schema:        createSchema("string"),
				data:          "test",
				expectedValid: true,
			},
			{
				name:          "Invalid String",
				schema:        createSchema("string"),
				data:          123,
				expectedValid: false,
			},
			{
				name:          "Valid Integer",
				schema:        createSchema("integer"),
				data:          123,
				expectedValid: true,
			},
			{
				name:          "Invalid Integer",
				schema:        createSchema("integer"),
				data:          "123",
				expectedValid: false,
			},
			{
				name:          "Valid Number",
				schema:        createSchema("number"),
				data:          123.45,
				expectedValid: true,
			},
			{
				name:          "Invalid Number",
				schema:        createSchema("number"),
				data:          "123.45",
				expectedValid: false,
			},
			{
				name:          "Valid Boolean",
				schema:        createSchema("boolean"),
				data:          true,
				expectedValid: true,
			},
			{
				name:          "Invalid Boolean",
				schema:        createSchema("boolean"),
				data:          "true",
				expectedValid: false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				data, _ := json.Marshal(tt.data)
				errs := validator.validateSchema(&openapi3.SchemaRef{Value: tt.schema}, data)
				assert.Equal(t, tt.expectedValid, len(errs) == 0)
			})
		}
	})

	t.Run("ValidateSchema_StringFormats", func(t *testing.T) {
		tests := []struct {
			name          string
			format        string
			data          string
			expectedValid bool
		}{
			{
				name:          "Valid Email",
				format:        "email",
				data:          "test@example.com",
				expectedValid: true,
			},
			{
				name:          "Invalid Email",
				format:        "email",
				data:          "not-an-email",
				expectedValid: false,
			},
			{
				name:          "Valid UUID",
				format:        "uuid",
				data:          "550e8400-e29b-41d4-a716-446655440000",
				expectedValid: true,
			},
			{
				name:          "Invalid UUID",
				format:        "uuid",
				data:          "not-a-uuid",
				expectedValid: false,
			},
			{
				name:          "Valid Date",
				format:        "date",
				data:          "2023-01-01",
				expectedValid: true,
			},
			{
				name:          "Invalid Date",
				format:        "date",
				data:          "2023/01/01",
				expectedValid: false,
			},
			{
				name:          "Valid DateTime",
				format:        "date-time",
				data:          "2023-01-01T12:00:00Z",
				expectedValid: true,
			},
			{
				name:          "Invalid DateTime",
				format:        "date-time",
				data:          "2023-01-01",
				expectedValid: false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				schema := createSchemaWithFormat("string", tt.format)
				data, _ := json.Marshal(tt.data)
				errs := validator.validateSchema(&openapi3.SchemaRef{Value: schema}, data)
				assert.Equal(t, tt.expectedValid, len(errs) == 0)
			})
		}
	})

	t.Run("ValidateSchema_StringConstraints", func(t *testing.T) {
		tests := []struct {
			name          string
			schema        *openapi3.Schema
			data          string
			expectedValid bool
		}{
			{
				name:          "Valid MinLength",
				schema:        createSchemaWithMinLength(3),
				data:          "test",
				expectedValid: true,
			},
			{
				name:          "Invalid MinLength",
				schema:        createSchemaWithMinLength(3),
				data:          "te",
				expectedValid: false,
			},
			{
				name:          "Valid Pattern",
				schema:        createSchemaWithPattern("^[a-z]+$"),
				data:          "test",
				expectedValid: true,
			},
			{
				name:          "Invalid Pattern",
				schema:        createSchemaWithPattern("^[a-z]+$"),
				data:          "test123",
				expectedValid: false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				data, _ := json.Marshal(tt.data)
				errs := validator.validateSchema(&openapi3.SchemaRef{Value: tt.schema}, data)
				assert.Equal(t, tt.expectedValid, len(errs) == 0)
			})
		}
	})

	t.Run("ValidateSchema_Objects", func(t *testing.T) {
		tests := []struct {
			name          string
			schema        *openapi3.Schema
			data          interface{}
			expectedValid bool
		}{
			{
				name: "Valid Object",
				schema: createObjectSchema(
					map[string]*openapi3.SchemaRef{
						"name": {Value: createSchema("string")},
						"age":  {Value: createSchema("integer")},
					},
					[]string{"name"},
				),
				data: map[string]interface{}{
					"name": "John",
					"age":  30,
				},
				expectedValid: true,
			},
			{
				name: "Missing Required Field",
				schema: createObjectSchema(
					map[string]*openapi3.SchemaRef{
						"name": {Value: createSchema("string")},
						"age":  {Value: createSchema("integer")},
					},
					[]string{"name"},
				),
				data: map[string]interface{}{
					"age": 30,
				},
				expectedValid: false,
			},
			{
				name: "Invalid Field Type",
				schema: createObjectSchema(
					map[string]*openapi3.SchemaRef{
						"name": {Value: createSchema("string")},
						"age":  {Value: createSchema("integer")},
					},
					[]string{"name"},
				),
				data: map[string]interface{}{
					"name": "John",
					"age":  "30",
				},
				expectedValid: false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				data, _ := json.Marshal(tt.data)
				errs := validator.validateSchema(&openapi3.SchemaRef{Value: tt.schema}, data)
				assert.Equal(t, tt.expectedValid, len(errs) == 0)
			})
		}
	})

	t.Run("ValidateSchema_Arrays", func(t *testing.T) {
		tests := []struct {
			name          string
			schema        *openapi3.Schema
			data          interface{}
			expectedValid bool
		}{
			{
				name:          "Valid Array",
				schema:        createArraySchema(createSchema("string")),
				data:          []string{"one", "two", "three"},
				expectedValid: true,
			},
			{
				name:          "Invalid Array Items",
				schema:        createArraySchema(createSchema("string")),
				data:          []interface{}{"one", 2, "three"},
				expectedValid: false,
			},
			{
				name:          "Valid MinItems",
				schema:        createArraySchemaWithMinItems(createSchema("string"), 2),
				data:          []string{"one", "two", "three"},
				expectedValid: true,
			},
			{
				name:          "Invalid MinItems",
				schema:        createArraySchemaWithMinItems(createSchema("string"), 2),
				data:          []string{"one"},
				expectedValid: false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				data, _ := json.Marshal(tt.data)
				errs := validator.validateSchema(&openapi3.SchemaRef{Value: tt.schema}, data)
				assert.Equal(t, tt.expectedValid, len(errs) == 0)
			})
		}
	})
}