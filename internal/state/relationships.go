package state

import (
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

// RelationshipType represents the type of relationship between resources
type RelationshipType string

const (
	// OneToOne represents a one-to-one relationship
	OneToOne RelationshipType = "one_to_one"
	// OneToMany represents a one-to-many relationship
	OneToMany RelationshipType = "one_to_many"
	// ManyToMany represents a many-to-many relationship
	ManyToMany RelationshipType = "many_to_many"
)

// Relationship represents a relationship between two resources
type Relationship struct {
	Type         RelationshipType
	FromResource string
	ToResource   string
	FromField    string
	ToField      string
}

// ExtractRelationships extracts relationships from an OpenAPI schema
func ExtractRelationships(spec *openapi3.T) []Relationship {
	var relationships []Relationship

	// Get all schema references
	for name, schema := range spec.Components.Schemas {
		if schema.Value == nil {
			continue
		}

		// Check if this is an object schema
		if len(schema.Value.Type) > 0 && string(schema.Value.Type[0]) == "object" {
			// Look for relationships in properties
			for propName, propSchema := range schema.Value.Properties {
				if propSchema.Value == nil {
					continue
				}

				// Check if this is an array or object reference
				if len(propSchema.Value.Type) > 0 {
					schemaType := string(propSchema.Value.Type[0])
					if schemaType == "object" {
						// Check for object reference
						if propSchema.Ref != "" {
							refName := getRefName(propSchema.Ref)
							relationships = append(relationships, Relationship{
								Type:         OneToOne,
								FromResource: name,
								ToResource:   refName,
								FromField:    propName,
								ToField:      "id",
							})
						}
					} else if schemaType == "array" {
						// Check for array of references
						if propSchema.Value.Items != nil && propSchema.Value.Items.Ref != "" {
							refName := getRefName(propSchema.Value.Items.Ref)
							relationships = append(relationships, Relationship{
								Type:         OneToMany,
								FromResource: name,
								ToResource:   refName,
								FromField:    propName,
								ToField:      "id",
							})
						}
					}
				}
			}
		}
	}

	return relationships
}

// getRefName extracts the resource name from a schema reference
func getRefName(ref string) string {
	parts := strings.Split(ref, "/")
	return parts[len(parts)-1]
} 