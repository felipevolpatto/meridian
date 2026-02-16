package generator

import (
	"fmt"
	"sort"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/jaswdr/faker"
)

// AutoSeedConfig configures automatic seed generation
type AutoSeedConfig struct {
	// Number of items to generate per resource
	ItemsPerResource int
	// Resources to include (empty means all)
	IncludeResources []string
	// Resources to exclude
	ExcludeResources []string
}

// ResourceDependency represents a dependency between resources
type ResourceDependency struct {
	Resource       string
	DependsOn      string
	ForeignKeyField string
	IsRequired     bool
}

// AutoSeeder generates seed data respecting relationships
type AutoSeeder struct {
	spec         *openapi3.T
	config       AutoSeedConfig
	faker        faker.Faker
	dependencies []ResourceDependency
	resources    map[string]*openapi3.SchemaRef
	generated    map[string][]map[string]interface{}
}

// NewAutoSeeder creates a new auto seeder
func NewAutoSeeder(spec *openapi3.T, config AutoSeedConfig) *AutoSeeder {
	if config.ItemsPerResource <= 0 {
		config.ItemsPerResource = 5
	}
	return &AutoSeeder{
		spec:      spec,
		config:    config,
		faker:     faker.New(),
		generated: make(map[string][]map[string]interface{}),
		resources: make(map[string]*openapi3.SchemaRef),
	}
}

// Generate generates seed data for all resources
func (s *AutoSeeder) Generate() (map[string][]interface{}, error) {
	// Extract resources from paths
	s.extractResources()

	// Detect dependencies between resources
	s.detectDependencies()

	// Sort resources by dependency order
	sortedResources := s.topologicalSort()

	// Generate data for each resource in order
	result := make(map[string][]interface{})
	for _, resourceName := range sortedResources {
		if !s.shouldInclude(resourceName) {
			continue
		}

		schema := s.resources[resourceName]
		if schema == nil {
			continue
		}

		items, err := s.generateResourceItems(resourceName, schema)
		if err != nil {
			return nil, fmt.Errorf("failed to generate %s: %w", resourceName, err)
		}

		result[resourceName] = make([]interface{}, len(items))
		for i, item := range items {
			result[resourceName][i] = item
		}
		s.generated[resourceName] = items
	}

	return result, nil
}

// extractResources extracts resource names and schemas from paths
func (s *AutoSeeder) extractResources() {
	for path, pathItem := range s.spec.Paths.Map() {
		resourceName := extractResourceName(path)
		if resourceName == "" {
			continue
		}

		// Try to find schema from POST request body or GET response
		var schema *openapi3.SchemaRef

		if pathItem.Post != nil && pathItem.Post.RequestBody != nil {
			if content := pathItem.Post.RequestBody.Value; content != nil {
				if mt := content.Content.Get("application/json"); mt != nil {
					schema = mt.Schema
				}
			}
		}

		if schema == nil && pathItem.Get != nil {
			for _, resp := range pathItem.Get.Responses.Map() {
				if resp.Value != nil {
					if mt := resp.Value.Content.Get("application/json"); mt != nil {
						// Check if it's an array (list endpoint)
						if mt.Schema.Value != nil && mt.Schema.Value.Type == "array" {
							schema = mt.Schema.Value.Items
						} else {
							schema = mt.Schema
						}
						break
					}
				}
			}
		}

		if schema != nil {
			// Resolve reference if needed
			if schema.Ref != "" {
				refName := getSchemaRefName(schema.Ref)
				if refSchema, ok := s.spec.Components.Schemas[refName]; ok {
					schema = refSchema
				}
			}
			s.resources[resourceName] = schema
		}
	}
}

// detectDependencies finds foreign key relationships between resources
func (s *AutoSeeder) detectDependencies() {
	s.dependencies = nil

	for resourceName, schema := range s.resources {
		if schema.Value == nil {
			continue
		}

		for propName, propSchema := range schema.Value.Properties {
			dep := s.detectForeignKey(resourceName, propName, propSchema, schema.Value)
			if dep != nil {
				s.dependencies = append(s.dependencies, *dep)
			}
		}
	}
}

// detectForeignKey checks if a property is a foreign key
func (s *AutoSeeder) detectForeignKey(resourceName, propName string, propSchema *openapi3.SchemaRef, parentSchema *openapi3.Schema) *ResourceDependency {
	if propSchema.Value == nil {
		return nil
	}

	// Check for _id suffix pattern (e.g., owner_id, customer_id)
	if strings.HasSuffix(propName, "_id") {
		relatedResource := strings.TrimSuffix(propName, "_id")
		// Convert to plural form
		relatedResourcePlural := pluralize(relatedResource)

		// Check if this resource exists
		if _, exists := s.resources[relatedResourcePlural]; exists {
			isRequired := false
			for _, req := range parentSchema.Required {
				if req == propName {
					isRequired = true
					break
				}
			}

			return &ResourceDependency{
				Resource:        resourceName,
				DependsOn:       relatedResourcePlural,
				ForeignKeyField: propName,
				IsRequired:      isRequired,
			}
		}
	}

	// Check for schema references
	if propSchema.Ref != "" {
		refName := getSchemaRefName(propSchema.Ref)
		relatedResource := pluralize(strings.ToLower(refName))
		if _, exists := s.resources[relatedResource]; exists {
			return &ResourceDependency{
				Resource:        resourceName,
				DependsOn:       relatedResource,
				ForeignKeyField: propName,
				IsRequired:      false,
			}
		}
	}

	return nil
}

// topologicalSort sorts resources by dependency order
func (s *AutoSeeder) topologicalSort() []string {
	// Build adjacency list
	graph := make(map[string][]string)
	inDegree := make(map[string]int)

	for name := range s.resources {
		graph[name] = []string{}
		inDegree[name] = 0
	}

	for _, dep := range s.dependencies {
		graph[dep.DependsOn] = append(graph[dep.DependsOn], dep.Resource)
		inDegree[dep.Resource]++
	}

	// Kahn's algorithm
	var queue []string
	for name, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, name)
		}
	}

	// Sort queue for deterministic output
	sort.Strings(queue)

	var result []string
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		result = append(result, node)

		neighbors := graph[node]
		sort.Strings(neighbors)
		for _, neighbor := range neighbors {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
		sort.Strings(queue)
	}

	// Add any remaining resources (circular dependencies)
	for name := range s.resources {
		found := false
		for _, r := range result {
			if r == name {
				found = true
				break
			}
		}
		if !found {
			result = append(result, name)
		}
	}

	return result
}

// generateResourceItems generates items for a single resource
func (s *AutoSeeder) generateResourceItems(resourceName string, schema *openapi3.SchemaRef) ([]map[string]interface{}, error) {
	var items []map[string]interface{}

	for i := 0; i < s.config.ItemsPerResource; i++ {
		item, err := s.generateItem(resourceName, schema, i)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return items, nil
}

// generateItem generates a single item with foreign key references
func (s *AutoSeeder) generateItem(resourceName string, schema *openapi3.SchemaRef, index int) (map[string]interface{}, error) {
	// Generate base data
	data, err := GenerateDataWithFieldName(schema, "")
	if err != nil {
		return nil, err
	}

	item, ok := data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("expected object, got %T", data)
	}

	// Generate a stable ID
	item["id"] = fmt.Sprintf("%s-%03d", singularize(resourceName), index+1)

	// Fill in foreign key references
	for _, dep := range s.dependencies {
		if dep.Resource != resourceName {
			continue
		}

		parentItems := s.generated[dep.DependsOn]
		if len(parentItems) == 0 {
			if dep.IsRequired {
				return nil, fmt.Errorf("no %s available for required reference", dep.DependsOn)
			}
			continue
		}

		// Assign a parent ID (distribute evenly or randomly)
		parentIndex := index % len(parentItems)
		parentItem := parentItems[parentIndex]
		if parentID, ok := parentItem["id"]; ok {
			item[dep.ForeignKeyField] = parentID
		}
	}

	return item, nil
}

// shouldInclude checks if a resource should be included
func (s *AutoSeeder) shouldInclude(resourceName string) bool {
	// Check exclusions first
	for _, excluded := range s.config.ExcludeResources {
		if strings.EqualFold(excluded, resourceName) {
			return false
		}
	}

	// If no inclusions specified, include all
	if len(s.config.IncludeResources) == 0 {
		return true
	}

	// Check inclusions
	for _, included := range s.config.IncludeResources {
		if strings.EqualFold(included, resourceName) {
			return true
		}
	}

	return false
}

// GetDependencies returns detected dependencies
func (s *AutoSeeder) GetDependencies() []ResourceDependency {
	return s.dependencies
}

// GetResourceOrder returns the topologically sorted resource order
func (s *AutoSeeder) GetResourceOrder() []string {
	return s.topologicalSort()
}

// extractResourceName extracts resource name from path
func extractResourceName(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 {
		return ""
	}

	// Find the first non-parameter segment
	for _, part := range parts {
		if !strings.HasPrefix(part, "{") {
			return part
		}
	}

	return ""
}

// getSchemaRefName extracts schema name from reference
func getSchemaRefName(ref string) string {
	parts := strings.Split(ref, "/")
	return parts[len(parts)-1]
}

// pluralize converts a singular word to plural
func pluralize(word string) string {
	if word == "" {
		return ""
	}

	// Common irregular plurals
	irregulars := map[string]string{
		"person":   "people",
		"child":    "children",
		"man":      "men",
		"woman":    "women",
		"category": "categories",
		"company":  "companies",
	}

	if plural, ok := irregulars[strings.ToLower(word)]; ok {
		return plural
	}

	lower := strings.ToLower(word)

	// Words ending in s, x, z, ch, sh
	if strings.HasSuffix(lower, "s") || strings.HasSuffix(lower, "x") ||
		strings.HasSuffix(lower, "z") || strings.HasSuffix(lower, "ch") ||
		strings.HasSuffix(lower, "sh") {
		return word + "es"
	}

	// Words ending in y preceded by consonant
	if strings.HasSuffix(lower, "y") && len(word) > 1 {
		prev := lower[len(lower)-2]
		if prev != 'a' && prev != 'e' && prev != 'i' && prev != 'o' && prev != 'u' {
			return word[:len(word)-1] + "ies"
		}
	}

	return word + "s"
}

// singularize converts a plural word to singular
func singularize(word string) string {
	if word == "" {
		return ""
	}

	// Common irregular singulars
	irregulars := map[string]string{
		"people":     "person",
		"children":   "child",
		"men":        "man",
		"women":      "woman",
		"categories": "category",
		"companies":  "company",
	}

	if singular, ok := irregulars[strings.ToLower(word)]; ok {
		return singular
	}

	lower := strings.ToLower(word)

	// Words ending in ies
	if strings.HasSuffix(lower, "ies") {
		return word[:len(word)-3] + "y"
	}

	// Words ending in es
	if strings.HasSuffix(lower, "es") {
		base := word[:len(word)-2]
		baseLower := strings.ToLower(base)
		if strings.HasSuffix(baseLower, "s") || strings.HasSuffix(baseLower, "x") ||
			strings.HasSuffix(baseLower, "z") || strings.HasSuffix(baseLower, "ch") ||
			strings.HasSuffix(baseLower, "sh") {
			return base
		}
	}

	// Words ending in s
	if strings.HasSuffix(lower, "s") && !strings.HasSuffix(lower, "ss") {
		return word[:len(word)-1]
	}

	return word
}
