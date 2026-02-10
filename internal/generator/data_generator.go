package generator

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/jaswdr/faker"
)

// GenerateData generates data based on the provided OpenAPI schema.
func GenerateData(schema *openapi3.SchemaRef) (interface{}, error) {
	return GenerateDataWithFieldName(schema, "")
}

// GenerateDataWithFieldName generates data with semantic field detection.
func GenerateDataWithFieldName(schema *openapi3.SchemaRef, fieldName string) (interface{}, error) {
	if schema == nil || schema.Value == nil {
		return nil, fmt.Errorf("schema is nil")
	}

	s := schema.Value

	if s.Example != nil {
		return s.Example, nil
	}

	// Handle oneOf
	if len(s.OneOf) > 0 {
		return GenerateFromOneOf(s.OneOf)
	}

	// Handle anyOf
	if len(s.AnyOf) > 0 {
		return GenerateFromAnyOf(s.AnyOf)
	}

	// Handle allOf
	if len(s.AllOf) > 0 {
		return GenerateFromAllOf(s.AllOf)
	}

	if len(s.Enum) > 0 {
		return s.Enum[rand.Intn(len(s.Enum))], nil
	}

	// Try semantic detection for strings
	if s.Type == "string" && fieldName != "" {
		semanticType := DetectSemanticType(fieldName)
		if semanticType != SemanticUnknown {
			if value := GenerateBySemanticType(semanticType, s); value != nil {
				return value, nil
			}
		}
	}

	switch s.Type {
	case "string":
		return generateString(s), nil
	case "number", "integer":
		return generateNumber(s), nil
	case "boolean":
		return faker.New().Bool(), nil
	case "array":
		data, err := generateArray(s)
		if err != nil {
			return nil, err
		}
		return data, nil
	case "object":
		data, err := generateObject(s)
		if err != nil {
			return nil, err
		}
		return data, nil
	default:
		return nil, fmt.Errorf("unsupported schema type: %s", s.Type)
	}
}

// GenerateExample generates an example based on the provided OpenAPI schema.
func GenerateExample(schema *openapi3.SchemaRef) (interface{}, error) {
	if schema == nil || schema.Value == nil {
		return nil, fmt.Errorf("schema is nil")
	}

	if schema.Value.Example != nil {
		return schema.Value.Example, nil
	}

	// For examples, we can simplify and just generate basic types or default values
	switch schema.Value.Type {
	case "string":
		return "string", nil
	case "number", "integer":
		return 1, nil
	case "boolean":
		return true, nil
	case "array":
		return []interface{}{}, nil
	case "object":
		obj := make(map[string]interface{})
		for name, propSchema := range schema.Value.Properties {
			example, err := GenerateExample(propSchema)
			if err != nil {
				return nil, err
			}
			obj[name] = example
		}
		return obj, nil
	default:
		return nil, fmt.Errorf("unsupported schema type for example generation: %s", schema.Value.Type)
	}
}

func generateString(schema *openapi3.Schema) string {
	f := faker.New()

	// Try pattern-based generation first
	if schema.Pattern != "" {
		if generated, err := GenerateFromPattern(schema.Pattern); err == nil {
			return generated
		}
	}

	switch schema.Format {
	case "email":
		return f.Internet().Email()
	case "uuid":
		return f.UUID().V4()
	case "uri":
		return f.Internet().URL()
	case "date-time":
		return f.Time().Time(time.Now()).Format(time.RFC3339)
	case "date":
		return f.Time().Time(time.Now()).Format("2006-01-02")
	case "time":
		return f.Time().Time(time.Now()).Format("15:04:05")
	default:
		return f.Lorem().Word()
	}
}

func generateNumber(schema *openapi3.Schema) interface{} {
	f := faker.New()
	if schema.Type == "integer" {
		min := int64(0)
		max := int64(100)
		if schema.Min != nil {
			min = int64(*schema.Min)
		}
		if schema.Max != nil {
			max = int64(*schema.Max)
		}
		return f.Int64Between(min, max)
	} else { // number (float)
		min := 0
		max := 100
		if schema.Min != nil {
			min = int(*schema.Min)
		}
		if schema.Max != nil {
			max = int(*schema.Max)
		}
		return f.Float64(2, min, max)
	}
}

func generateArray(schema *openapi3.Schema) ([]interface{}, error) {
	f := faker.New()
	minItems := int(schema.MinItems)
	maxItems := 0
	if schema.MaxItems != nil {
		maxItems = int(*schema.MaxItems)
	}
	if maxItems == 0 { // If MaxItems is not set, default to a reasonable number
		maxItems = minItems + 3
	}
	if minItems > maxItems {
		minItems = maxItems
	}

	count := f.IntBetween(minItems, maxItems)
	arr := make([]interface{}, count)
	for i := 0; i < count; i++ {
		item, err := GenerateData(schema.Items)
		if err != nil {
			return nil, err
		}
		arr[i] = item
	}
	return arr, nil
}

func generateObject(schema *openapi3.Schema) (map[string]interface{}, error) {
	obj := make(map[string]interface{})
	f := faker.New()

	// Handle properties with semantic field detection
	for name, propSchema := range schema.Properties {
		data, err := GenerateDataWithFieldName(propSchema, name)
		if err != nil {
			return nil, err
		}
		obj[name] = data
	}

	// Handle allOf (composition)
	for _, allOfSchema := range schema.AllOf {
		if allOfSchema.Value == nil {
			continue
		}
		allOfObj, err := generateObject(allOfSchema.Value)
		if err != nil {
			return nil, err
		}
		for k, v := range allOfObj {
			obj[k] = v
		}
	}

	// Handle discriminator
	if schema.Discriminator != nil && len(schema.Discriminator.Mapping) > 0 {
		for k := range schema.Discriminator.Mapping {
			obj[schema.Discriminator.PropertyName] = k
			break
		}
	}

	// Handle additional properties
	if schema.AdditionalProperties.Has != nil && *schema.AdditionalProperties.Has {
		for i := 0; i < f.IntBetween(1, 3); i++ {
			key := f.Lorem().Word()
			val, err := GenerateDataWithFieldName(schema.AdditionalProperties.Schema, key)
			if err != nil {
				return nil, err
			}
			obj[key] = val
		}
	}

	return obj, nil
}
