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
	if schema == nil || schema.Value == nil {
		return nil, fmt.Errorf("schema is nil")
	}

	if schema.Value.Example != nil {
		return schema.Value.Example, nil
	}

	if len(schema.Value.Enum) > 0 {
		return schema.Value.Enum[rand.Intn(len(schema.Value.Enum))], nil
	}

	switch schema.Value.Type {
	case "string":
		return generateString(schema.Value), nil
	case "number", "integer":
		return generateNumber(schema.Value), nil
	case "boolean":
		return faker.New().Bool(), nil
	case "array":
		data, err := generateArray(schema.Value)
		if err != nil {
			return nil, err
		}
		return data, nil
	case "object":
		data, err := generateObject(schema.Value)
		if err != nil {
			return nil, err
		}
		return data, nil
	default:
		return nil, fmt.Errorf("unsupported schema type: %s", schema.Value.Type)
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

	// Handle properties
	for name, propSchema := range schema.Properties {
		data, err := GenerateData(propSchema)
		if err != nil {
			return nil, err
		}
		obj[name] = data
	}

	// Handle allOf (composition)
	for _, allOfSchema := range schema.AllOf {
		allOfObj, err := generateObject(allOfSchema.Value)
		if err != nil {
			return nil, err
		}
		for k, v := range allOfObj {
			obj[k] = v
		}
	}

	// Handle discriminator (if applicable, though usually handled by external logic selecting the correct schema)
	if schema.Discriminator != nil && len(schema.Discriminator.Mapping) > 0 {
		// For generation, pick one of the mapped types
		for k, _ := range schema.Discriminator.Mapping {
			// This is a simplification; in a real scenario, you'd resolve the ref
			// and generate based on the actual schema. For now, just set the discriminator property.
			obj[schema.Discriminator.PropertyName] = k
			break // Just pick the first one for now
		}
	}

	// Handle additional properties
	if schema.AdditionalProperties.Has != nil && *schema.AdditionalProperties.Has {
		// Generate a few additional properties
		for i := 0; i < f.IntBetween(1, 3); i++ {
			key := f.Lorem().Word()
			val, err := GenerateData(schema.AdditionalProperties.Schema)
			if err != nil {
				return nil, err
			}
			obj[key] = val
		}
	}

	return obj, nil
}
