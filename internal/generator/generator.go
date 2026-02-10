package generator

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/jaswdr/faker"
)

// Generator handles data generation based on OpenAPI schemas
type Generator struct {
	faker       faker.Faker
	customFuncs map[string]CustomFakerFunc
	cache       *generationCache
	mu          sync.RWMutex
}

// CustomFakerFunc is a function that generates custom fake data
type CustomFakerFunc func(schema *openapi3.Schema, context *GenerationContext) (interface{}, error)

// GenerationContext provides context for data generation
type GenerationContext struct {
	// Path to current field being generated (e.g., "user.address.street")
	Path string

	// Parent resource name and ID if generating related data
	ParentResource string
	ParentID      string

	// Resource relationships
	Relationships map[string]map[string]string

	// Cache for consistent data generation
	Cache map[string]interface{}
}

// generationCache handles caching of generated values for consistency
type generationCache struct {
	mu    sync.RWMutex
	data  map[string]map[string]interface{}
	rules map[string]GenerationRule
}

// GenerationRule defines custom rules for data generation
type GenerationRule struct {
	// Field path pattern (supports wildcards, e.g., "users.*.email")
	Pattern string

	// Custom generation function
	Generator CustomFakerFunc

	// Dependencies on other fields
	Dependencies []string

	// Validation function for generated values
	Validator func(interface{}) error

	// Whether to cache the generated value for consistency
	Cache bool
}

// New creates a new data generator
func New() *Generator {
	return &Generator{
		faker:       faker.New(),
		customFuncs: make(map[string]CustomFakerFunc),
		cache: &generationCache{
			data:  make(map[string]map[string]interface{}),
			rules: make(map[string]GenerationRule),
		},
	}
}

// RegisterCustomFunc registers a custom faker function for a specific format
func (g *Generator) RegisterCustomFunc(format string, fn CustomFakerFunc) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.customFuncs[format] = fn
}

// RegisterRule registers a custom generation rule
func (g *Generator) RegisterRule(rule GenerationRule) error {
	if rule.Pattern == "" {
		return fmt.Errorf("rule pattern cannot be empty")
	}

	g.cache.mu.Lock()
	defer g.cache.mu.Unlock()
	g.cache.rules[rule.Pattern] = rule
	return nil
}

// Generate generates data based on an OpenAPI schema
func (g *Generator) Generate(schema *openapi3.Schema, context *GenerationContext) (interface{}, error) {
	if context == nil {
		context = &GenerationContext{
			Cache: make(map[string]interface{}),
		}
	}

	// Check for cached value if caching is enabled
	if cached := g.getCachedValue(context.Path); cached != nil {
		return cached, nil
	}

	// Check for custom generation rules
	if rule := g.findMatchingRule(context.Path); rule != nil {
		value, err := g.applyRule(rule, schema, context)
		if err != nil {
			return nil, fmt.Errorf("failed to apply rule for %s: %w", context.Path, err)
		}
		return value, nil
	}

	// Generate based on schema type
	value, err := g.generateByType(schema, context)
	if err != nil {
		return nil, fmt.Errorf("failed to generate value for %s: %w", context.Path, err)
	}

	// Cache the value if needed
	g.cacheValue(context.Path, value)

	return value, nil
}

// GenerateRelated generates related data based on relationships
func (g *Generator) GenerateRelated(schema *openapi3.Schema, parentResource, parentID string, relationships map[string]string) (interface{}, error) {
	context := &GenerationContext{
		ParentResource: parentResource,
		ParentID:      parentID,
		Cache:         make(map[string]interface{}),
	}

	// Handle different relationship types
	for _, relType := range relationships {
		switch relType {
		case "one_to_one":
			return g.generateOneToOne(schema, context)
		case "one_to_many":
			return g.generateOneToMany(schema, context)
		case "many_to_one":
			return g.generateManyToOne(schema, context)
		case "many_to_many":
			return g.generateManyToMany(schema, context)
		default:
			return nil, fmt.Errorf("unsupported relationship type: %s", relType)
		}
	}

	return g.Generate(schema, context)
}

func (g *Generator) generateByType(schema *openapi3.Schema, context *GenerationContext) (interface{}, error) {
	// Check for custom format handler
	if schema.Format != "" {
		if fn, ok := g.customFuncs[schema.Format]; ok {
			return fn(schema, context)
		}
	}

	switch schema.Type {
	case "string":
		return g.generateString(schema)
	case "integer", "number":
		return g.generateNumber(schema)
	case "boolean":
		return g.faker.Bool(), nil
	case "array":
		return g.generateArray(schema, context)
	case "object":
		return g.generateObject(schema, context)
	default:
		return nil, fmt.Errorf("unsupported type: %s", schema.Type)
	}
}

func (g *Generator) generateString(schema *openapi3.Schema) (interface{}, error) {
	switch schema.Format {
	case "email":
		return g.faker.Internet().Email(), nil
	case "date-time":
		return g.faker.Time().Time(time.Now()).Format(time.RFC3339), nil
	case "date":
		return g.faker.Time().Time(time.Now()).Format("2006-01-02"), nil
	case "uuid":
		return g.faker.UUID().V4(), nil
	case "uri":
		return g.faker.Internet().URL(), nil
	case "hostname":
		return g.faker.Internet().Domain(), nil
	case "ipv4":
		return g.faker.Internet().Ipv4(), nil
	case "ipv6":
		return g.faker.Internet().Ipv6(), nil
	default:
		if schema.Pattern != "" {
			// TODO: Implement regex-based string generation
			return g.faker.Lorem().Word(), nil
		}
		if schema.Enum != nil {
			// Convert enum values to strings
			enumStrings := make([]string, len(schema.Enum))
			for i, v := range schema.Enum {
				enumStrings[i] = fmt.Sprintf("%v", v)
			}
			return g.faker.RandomStringElement(enumStrings), nil
		}
		return g.faker.Lorem().Word(), nil
	}
}

func (g *Generator) generateNumber(schema *openapi3.Schema) (interface{}, error) {
	if schema.Type == "integer" {
		min := int64(0)
		max := int64(100)
		if schema.Min != nil {
			min = int64(*schema.Min)
		}
		if schema.Max != nil {
			max = int64(*schema.Max)
		}
		return g.faker.Int64Between(min, max), nil
	}

	min := 0.0
	max := 100.0
	if schema.Min != nil {
		min = *schema.Min
	}
	if schema.Max != nil {
		max = *schema.Max
	}
	// Use Int64Between and convert to float64 since faker doesn't have Float64Between
	intMin := int64(min * 100)
	intMax := int64(max * 100)
	randomInt := g.faker.Int64Between(intMin, intMax)
	return float64(randomInt) / 100.0, nil
}

func (g *Generator) generateArray(schema *openapi3.Schema, context *GenerationContext) (interface{}, error) {
	if schema.Items == nil {
		return nil, fmt.Errorf("array items schema is required")
	}

	minItems := 1
	maxItems := 5

	if schema.MinItems > 0 {
		minItems = int(schema.MinItems)
	}
	if schema.MaxItems != nil && *schema.MaxItems > 0 {
		maxItems = int(*schema.MaxItems)
	}

	count := g.faker.IntBetween(minItems, maxItems)
	items := make([]interface{}, count)

	for i := 0; i < count; i++ {
		itemContext := &GenerationContext{
			Path:          fmt.Sprintf("%s[%d]", context.Path, i),
			ParentResource: context.ParentResource,
			ParentID:      context.ParentID,
			Cache:         context.Cache,
		}
		item, err := g.Generate(schema.Items.Value, itemContext)
		if err != nil {
			return nil, fmt.Errorf("failed to generate array item %d: %w", i, err)
		}
		items[i] = item
	}

	return items, nil
}

func (g *Generator) generateObject(schema *openapi3.Schema, context *GenerationContext) (interface{}, error) {
	obj := make(map[string]interface{})

	for name, prop := range schema.Properties {
		if prop.Value == nil {
			continue
		}

		propContext := &GenerationContext{
			Path:          fmt.Sprintf("%s.%s", context.Path, name),
			ParentResource: context.ParentResource,
			ParentID:      context.ParentID,
			Cache:         context.Cache,
		}

		value, err := g.Generate(prop.Value, propContext)
		if err != nil {
			return nil, fmt.Errorf("failed to generate property %s: %w", name, err)
		}

		obj[name] = value
	}

	return obj, nil
}

func (g *Generator) generateOneToOne(schema *openapi3.Schema, context *GenerationContext) (interface{}, error) {
	// Generate a single related object with consistent ID reference
	value, err := g.Generate(schema, context)
	if err != nil {
		return nil, err
	}

	// Ensure the generated object has a reference to the parent
	if obj, ok := value.(map[string]interface{}); ok {
		obj[context.ParentResource+"_id"] = context.ParentID
	}

	return value, nil
}

func (g *Generator) generateOneToMany(schema *openapi3.Schema, context *GenerationContext) (interface{}, error) {
	// Generate 1-5 related objects
	count := g.faker.IntBetween(1, 5)
	items := make([]interface{}, count)

	for i := 0; i < count; i++ {
		value, err := g.Generate(schema, context)
		if err != nil {
			return nil, err
		}

		// Add parent reference to each item
		if obj, ok := value.(map[string]interface{}); ok {
			obj[context.ParentResource+"_id"] = context.ParentID
		}

		items[i] = value
	}

	return items, nil
}

func (g *Generator) generateManyToOne(schema *openapi3.Schema, context *GenerationContext) (interface{}, error) {
	// Generate a single object that multiple parents refer to
	value, err := g.Generate(schema, context)
	if err != nil {
		return nil, err
	}

	// Store the generated value in cache for consistency
	if obj, ok := value.(map[string]interface{}); ok {
		cacheKey := fmt.Sprintf("%s_%s", context.ParentResource, obj["id"])
		g.cacheValue(cacheKey, value)
	}

	return value, nil
}

func (g *Generator) generateManyToMany(schema *openapi3.Schema, context *GenerationContext) (interface{}, error) {
	// Generate 1-3 related objects
	count := g.faker.IntBetween(1, 3)
	items := make([]interface{}, count)

	for i := 0; i < count; i++ {
		value, err := g.Generate(schema, context)
		if err != nil {
			return nil, err
		}

		items[i] = value
	}

	return items, nil
}

func (g *Generator) findMatchingRule(path string) *GenerationRule {
	g.cache.mu.RLock()
	defer g.cache.mu.RUnlock()

	// Check exact match first
	if rule, ok := g.cache.rules[path]; ok {
		return &rule
	}

	// Check pattern matches
	parts := strings.Split(path, ".")
	for pattern, rule := range g.cache.rules {
		if matchPattern(pattern, parts) {
			return &rule
		}
	}

	return nil
}

func (g *Generator) applyRule(rule *GenerationRule, schema *openapi3.Schema, context *GenerationContext) (interface{}, error) {
	// Check dependencies
	for _, dep := range rule.Dependencies {
		if _, ok := context.Cache[dep]; !ok {
			return nil, fmt.Errorf("missing dependency: %s", dep)
		}
	}

	// Generate value using custom generator
	value, err := rule.Generator(schema, context)
	if err != nil {
		return nil, err
	}

	// Validate generated value
	if rule.Validator != nil {
		if err := rule.Validator(value); err != nil {
			return nil, fmt.Errorf("validation failed: %w", err)
		}
	}

	// Cache if needed
	if rule.Cache {
		g.cacheValue(context.Path, value)
	}

	return value, nil
}

func (g *Generator) getCachedValue(path string) interface{} {
	g.cache.mu.RLock()
	defer g.cache.mu.RUnlock()

	parts := strings.Split(path, ".")
	if len(parts) < 2 {
		return nil
	}

	resource := parts[0]
	field := parts[1]

	if resourceCache, ok := g.cache.data[resource]; ok {
		return resourceCache[field]
	}

	return nil
}

func (g *Generator) cacheValue(path string, value interface{}) {
	g.cache.mu.Lock()
	defer g.cache.mu.Unlock()

	parts := strings.Split(path, ".")
	if len(parts) < 2 {
		return
	}

	resource := parts[0]
	field := parts[1]

	if _, ok := g.cache.data[resource]; !ok {
		g.cache.data[resource] = make(map[string]interface{})
	}

	g.cache.data[resource][field] = value
}

func matchPattern(pattern string, parts []string) bool {
	patternParts := strings.Split(pattern, ".")
	if len(patternParts) != len(parts) {
		return false
	}

	for i, part := range patternParts {
		if part != "*" && part != parts[i] {
			return false
		}
	}

	return true
} 