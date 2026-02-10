package validation

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

// ValidateSchemaValue validates a value against an OpenAPI schema
func ValidateSchemaValue(schema *openapi3.Schema, value interface{}) []string {
	var errors []string

	// Handle nil value
	if value == nil {
		if schema.Nullable {
			return nil
		}
		return []string{"value cannot be null"}
	}

	// Get the schema type
	if schema.Type == "" {
		return nil
	}

	valueType := reflect.TypeOf(value)

	// Validate type
	switch schema.Type {
	case "string":
		if valueType.Kind() != reflect.String {
			errors = append(errors, fmt.Sprintf("expected string, got %T", value))
			return errors
		}
		errors = append(errors, validateSchemaString(schema, value.(string))...)

	case "number", "integer":
		if !isNumeric(valueType.Kind()) {
			errors = append(errors, fmt.Sprintf("expected number, got %T", value))
			return errors
		}
		errors = append(errors, validateSchemaNumber(schema, value)...)

	case "boolean":
		if valueType.Kind() != reflect.Bool {
			errors = append(errors, fmt.Sprintf("expected boolean, got %T", value))
			return errors
		}

	case "array":
		if valueType.Kind() != reflect.Slice && valueType.Kind() != reflect.Array {
			errors = append(errors, fmt.Sprintf("expected array, got %T", value))
			return errors
		}
		errors = append(errors, validateSchemaArray(schema, value)...)

	case "object":
		if valueType.Kind() != reflect.Map {
			errors = append(errors, fmt.Sprintf("expected object, got %T", value))
			return errors
		}
		errors = append(errors, validateSchemaObject(schema, value)...)
	}

	return errors
}

func validateSchemaString(schema *openapi3.Schema, value string) []string {
	var errors []string

	// Validate enum
	if len(schema.Enum) > 0 {
		valid := false
		for _, enum := range schema.Enum {
			if str, ok := enum.(string); ok && str == value {
				valid = true
				break
			}
		}
		if !valid {
			errors = append(errors, fmt.Sprintf("value must be one of: %v", schema.Enum))
		}
	}

	// Validate length
	if schema.MinLength > 0 && len(value) < int(schema.MinLength) {
		errors = append(errors, fmt.Sprintf("string length must be >= %d", schema.MinLength))
	}
	if schema.MaxLength != nil && len(value) > int(*schema.MaxLength) {
		errors = append(errors, fmt.Sprintf("string length must be <= %d", *schema.MaxLength))
	}

	// Validate pattern
	if schema.Pattern != "" {
		// TODO: Implement pattern validation
	}

	// Validate format
	if schema.Format != "" {
		switch schema.Format {
		case "email":
			if !strings.Contains(value, "@") {
				errors = append(errors, "invalid email format")
			}
		case "uri":
			if !strings.Contains(value, "://") {
				errors = append(errors, "invalid URI format")
			}
		case "uuid":
			if len(value) != 36 {
				errors = append(errors, "invalid UUID format")
			}
		case "date":
			if len(value) != 10 {
				errors = append(errors, "invalid date format (expected YYYY-MM-DD)")
			}
		case "date-time":
			if !strings.Contains(value, "T") || !strings.Contains(value, "Z") {
				errors = append(errors, "invalid date-time format (expected RFC3339)")
			}
		}
	}

	return errors
}

func validateSchemaNumber(schema *openapi3.Schema, value interface{}) []string {
	var errors []string
	var floatValue float64

	// Convert value to float64 for comparison
	switch v := value.(type) {
	case int:
		floatValue = float64(v)
	case int32:
		floatValue = float64(v)
	case int64:
		floatValue = float64(v)
	case float32:
		floatValue = float64(v)
	case float64:
		floatValue = v
	}

	// Validate minimum
	if schema.Min != nil && floatValue < *schema.Min {
		errors = append(errors, fmt.Sprintf("value must be >= %v", *schema.Min))
	}

	// Validate maximum
	if schema.Max != nil && floatValue > *schema.Max {
		errors = append(errors, fmt.Sprintf("value must be <= %v", *schema.Max))
	}

	// Validate multiple of
	if schema.MultipleOf != nil {
		if floatValue != float64(int64(floatValue/(*schema.MultipleOf)))*(*schema.MultipleOf) {
			errors = append(errors, fmt.Sprintf("value must be a multiple of %v", *schema.MultipleOf))
		}
	}

	return errors
}

func validateSchemaArray(schema *openapi3.Schema, value interface{}) []string {
	var errors []string
	arr := reflect.ValueOf(value)

	// Validate length
	if schema.MinItems > 0 && arr.Len() < int(schema.MinItems) {
		errors = append(errors, fmt.Sprintf("array length must be >= %d", schema.MinItems))
	}
	if schema.MaxItems != nil && arr.Len() > int(*schema.MaxItems) {
		errors = append(errors, fmt.Sprintf("array length must be <= %d", *schema.MaxItems))
	}

	// Validate items
	if schema.Items != nil && schema.Items.Value != nil {
		for i := 0; i < arr.Len(); i++ {
			itemErrors := ValidateSchemaValue(schema.Items.Value, arr.Index(i).Interface())
			for _, err := range itemErrors {
				errors = append(errors, fmt.Sprintf("item %d: %s", i, err))
			}
		}
	}

	// Validate uniqueness
	if schema.UniqueItems {
		seen := make(map[interface{}]bool)
		for i := 0; i < arr.Len(); i++ {
			item := arr.Index(i).Interface()
			if seen[item] {
				errors = append(errors, "array items must be unique")
				break
			}
			seen[item] = true
		}
	}

	return errors
}

func validateSchemaObject(schema *openapi3.Schema, value interface{}) []string {
	var errors []string
	obj := reflect.ValueOf(value)

	// Get object as map
	objMap, ok := value.(map[string]interface{})
	if !ok {
		return []string{"value must be a map[string]interface{}"}
	}

	// Validate required properties
	for _, required := range schema.Required {
		if _, ok := objMap[required]; !ok {
			errors = append(errors, fmt.Sprintf("missing required property: %s", required))
		}
	}

	// Validate properties
	for propName, propValue := range objMap {
		// Check if property is defined in schema
		if propSchema, ok := schema.Properties[propName]; ok {
			propErrors := ValidateSchemaValue(propSchema.Value, propValue)
			for _, err := range propErrors {
				errors = append(errors, fmt.Sprintf("property %s: %s", propName, err))
			}
		} else if schema.AdditionalProperties.Has != nil && !*schema.AdditionalProperties.Has {
			errors = append(errors, fmt.Sprintf("additional property %s not allowed", propName))
		} else if schema.AdditionalProperties.Schema != nil {
			propErrors := ValidateSchemaValue(schema.AdditionalProperties.Schema.Value, propValue)
			for _, err := range propErrors {
				errors = append(errors, fmt.Sprintf("property %s: %s", propName, err))
			}
		}
	}

	// Validate min/max properties
	if schema.MinProps > 0 && obj.Len() < int(schema.MinProps) {
		errors = append(errors, fmt.Sprintf("object must have >= %d properties", schema.MinProps))
	}
	if schema.MaxProps != nil && obj.Len() > int(*schema.MaxProps) {
		errors = append(errors, fmt.Sprintf("object must have <= %d properties", *schema.MaxProps))
	}

	return errors
}

func isNumeric(kind reflect.Kind) bool {
	switch kind {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return true
	default:
		return false
	}
} 