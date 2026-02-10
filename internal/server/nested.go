package server

import (
	"strings"
)

// NestedResourceInfo contains information about a nested resource path
type NestedResourceInfo struct {
	// IsNested indicates if this is a nested resource path
	IsNested bool

	// ParentResource is the parent resource name (e.g., "users")
	ParentResource string

	// ParentID is the parent resource ID extracted from path params
	ParentID string

	// ParentIDParam is the parameter name for parent ID (e.g., "userId")
	ParentIDParam string

	// ChildResource is the child resource name (e.g., "posts")
	ChildResource string

	// ChildID is the child resource ID if present
	ChildID string

	// ChildIDParam is the parameter name for child ID (e.g., "postId")
	ChildIDParam string

	// ForeignKeyField is the field name in child that references parent (e.g., "user_id")
	ForeignKeyField string
}

// ParseNestedResource parses a path and path params to detect nested resources
// Handles patterns like:
//   - /users/{userId}/posts
//   - /users/{userId}/posts/{postId}
//   - /organizations/{orgId}/teams/{teamId}/members
func ParseNestedResource(path string, pathParams map[string]string) *NestedResourceInfo {
	parts := strings.Split(strings.Trim(path, "/"), "/")

	if len(parts) < 3 {
		return &NestedResourceInfo{IsNested: false}
	}

	// Look for pattern: resource/{param}/resource or resource/{param}/resource/{param}
	info := &NestedResourceInfo{}

	for i := 0; i < len(parts)-1; i++ {
		// Check if current part is a resource and next is a param placeholder or actual param
		if isResourceName(parts[i]) && i+2 < len(parts) && isResourceName(parts[i+2]) {
			info.IsNested = true
			info.ParentResource = parts[i]
			info.ChildResource = parts[i+2]

			// Find the parent ID param name and value
			parentIDParam := findParamForPosition(pathParams, parts, i+1)
			if parentIDParam != "" {
				info.ParentIDParam = parentIDParam
				info.ParentID = pathParams[parentIDParam]
			}

			// Check if there's a child ID
			if i+3 < len(parts) {
				childIDParam := findParamForPosition(pathParams, parts, i+3)
				if childIDParam != "" {
					info.ChildIDParam = childIDParam
					info.ChildID = pathParams[childIDParam]
				}
			}

			// Infer foreign key field name
			info.ForeignKeyField = inferForeignKeyField(info.ParentResource, info.ParentIDParam)

			return info
		}
	}

	return &NestedResourceInfo{IsNested: false}
}

// isResourceName checks if a path part looks like a resource name (not a parameter)
func isResourceName(part string) bool {
	if part == "" {
		return false
	}
	// Parameters in OpenAPI are wrapped in braces
	if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
		return false
	}
	// Actual parameter values could be UUIDs, numbers, etc.
	// Resource names are typically lowercase alphabetic
	for _, c := range part {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_' || c == '-') {
			return false
		}
	}
	return true
}

// findParamForPosition finds the parameter name for a given position in the path
func findParamForPosition(pathParams map[string]string, parts []string, position int) string {
	if position >= len(parts) {
		return ""
	}

	value := parts[position]

	// Look for a param that matches this value
	for paramName, paramValue := range pathParams {
		if paramValue == value {
			return paramName
		}
	}

	return ""
}

// inferForeignKeyField infers the foreign key field name from parent resource
// Examples:
//   - users, userId -> user_id
//   - organizations, orgId -> organization_id
//   - teams, teamId -> team_id
func inferForeignKeyField(parentResource, parentIDParam string) string {
	// Remove trailing 's' for singular form
	singular := parentResource
	if strings.HasSuffix(singular, "ies") {
		singular = singular[:len(singular)-3] + "y"
	} else if strings.HasSuffix(singular, "s") {
		singular = singular[:len(singular)-1]
	}

	return singular + "_id"
}

// BuildNestedResourceKey creates a composite key for storing nested resources
func BuildNestedResourceKey(parentResource, parentID, childResource string) string {
	return parentResource + ":" + parentID + ":" + childResource
}

// ParseNestedResourceKey parses a composite key back to its components
func ParseNestedResourceKey(key string) (parentResource, parentID, childResource string, ok bool) {
	parts := strings.Split(key, ":")
	if len(parts) != 3 {
		return "", "", "", false
	}
	return parts[0], parts[1], parts[2], true
}

// ExtractResourceInfo extracts resource information from a path
// Returns the primary resource name, whether it's nested, and nested info
func ExtractResourceInfo(path string, pathParams map[string]string) (resourceName string, nestedInfo *NestedResourceInfo) {
	nestedInfo = ParseNestedResource(path, pathParams)

	if nestedInfo.IsNested {
		resourceName = nestedInfo.ChildResource
	} else {
		parts := strings.Split(strings.Trim(path, "/"), "/")
		if len(parts) > 0 {
			resourceName = parts[0]
		}
	}

	return resourceName, nestedInfo
}
