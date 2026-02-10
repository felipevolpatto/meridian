package validation

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

// ValidationError represents a validation error with details
type ValidationError struct {
	Field   string `json:"field,omitempty"`
	Message string `json:"message"`
	Code    string `json:"code"`
}

// ValidationErrors is a slice of ValidationError that implements the error interface
type ValidationErrors []*ValidationError

func (e ValidationErrors) Error() string {
	if len(e) == 0 {
		return ""
	}
	var messages []string
	for _, err := range e {
		messages = append(messages, err.Error())
	}
	return strings.Join(messages, "; ")
}

func (e *ValidationError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("%s: %s", e.Field, e.Message)
	}
	return e.Message
}

// RequestValidator handles validation of requests against OpenAPI spec
type RequestValidator struct {
	spec *openapi3.T
}

// NewRequestValidator creates a new request validator
func NewRequestValidator(spec *openapi3.T) *RequestValidator {
	return &RequestValidator{spec: spec}
}

// ValidateRequest validates a request against the OpenAPI spec
func (v *RequestValidator) ValidateRequest(method, path string, headers map[string][]string, query url.Values, body []byte) ValidationErrors {
	var errors ValidationErrors

	// Find the path in the spec
	pathItem := v.findPath(path)
	if pathItem == nil {
		return append(errors, &ValidationError{
			Message: "Path not found in API specification",
			Code:    "path_not_found",
		})
	}

	// Get the operation
	op := v.getOperation(pathItem, method)
	if op == nil {
		return append(errors, &ValidationError{
			Message: fmt.Sprintf("Method %s not allowed for path %s", method, path),
			Code:    "method_not_allowed",
		})
	}

	// Validate path parameters
	pathParams := extractPathParams(path, v.findPathTemplate(path))
	if pathErrs := v.validatePathParams(op, pathItem, pathParams); len(pathErrs) > 0 {
		errors = append(errors, pathErrs...)
	}

	// Validate query parameters
	if queryErrs := v.validateQueryParams(op, query); len(queryErrs) > 0 {
		errors = append(errors, queryErrs...)
	}

	// Validate headers
	if headerErrs := v.validateHeaders(op, headers); len(headerErrs) > 0 {
		errors = append(errors, headerErrs...)
	}

	// Validate request body
	if bodyErrs := v.validateRequestBody(op, body); len(bodyErrs) > 0 {
		errors = append(errors, bodyErrs...)
	}

	return errors
}

// ValidateResponse validates a response against the OpenAPI spec
func (v *RequestValidator) ValidateResponse(method, path string, statusCode int, headers map[string][]string, body []byte) ValidationErrors {
	var errors ValidationErrors

	// Find the path in the spec
	pathItem := v.findPath(path)
	if pathItem == nil {
		return append(errors, &ValidationError{
			Message: "Path not found in API specification",
			Code:    "path_not_found",
		})
	}

	op := v.getOperation(pathItem, method)
	if op == nil {
		return append(errors, &ValidationError{
			Message: fmt.Sprintf("Method %s not allowed for path %s", method, path),
			Code:    "method_not_allowed",
		})
	}

	// Get the response specification
	statusStr := fmt.Sprintf("%d", statusCode)
	responses := op.Responses.Map()
	resp := responses[statusStr]
	if resp == nil {
		// Check for default response
		defaultResp := responses["default"]
		if defaultResp == nil {
			return ValidationErrors{
				{
					Message: fmt.Sprintf("Status code %d not defined in API specification", statusCode),
					Code:    "status_not_found",
				},
			}
		}
		resp = defaultResp
	}

	// Skip validation if response is nil
	if resp == nil || resp.Value == nil {
		return ValidationErrors{
			{
				Message: fmt.Sprintf("Response for status code %d is not defined", statusCode),
				Code:    "status_not_found",
			},
		}
	}

	// Validate response headers
	if headerErrs := v.validateResponseHeaders(resp.Value, headers); len(headerErrs) > 0 {
		errors = append(errors, headerErrs...)
	}

	// Validate response body
	if bodyErrs := v.validateResponseBody(resp.Value, body); len(bodyErrs) > 0 {
		errors = append(errors, bodyErrs...)
	}

	return errors
}

func (v *RequestValidator) findPath(path string) *openapi3.PathItem {
	// First try exact match
	if pathItem := v.spec.Paths.Find(path); pathItem != nil {
		return pathItem
	}

	// Try matching with path parameters
	for specPath, pathItem := range v.spec.Paths.Map() {
		if matchPath(specPath, path) {
			return pathItem
		}
	}

	return nil
}

func (v *RequestValidator) findPathTemplate(path string) string {
	// First try exact match
	if v.spec.Paths.Find(path) != nil {
		return path
	}

	// Try matching with path parameters
	for specPath := range v.spec.Paths.Map() {
		if matchPath(specPath, path) {
			return specPath
		}
	}

	return ""
}

func (v *RequestValidator) getOperation(pathItem *openapi3.PathItem, method string) *openapi3.Operation {
	switch strings.ToUpper(method) {
	case "GET":
		return pathItem.Get
	case "POST":
		return pathItem.Post
	case "PUT":
		return pathItem.Put
	case "DELETE":
		return pathItem.Delete
	case "PATCH":
		return pathItem.Patch
	case "HEAD":
		return pathItem.Head
	case "OPTIONS":
		return pathItem.Options
	case "TRACE":
		return pathItem.Trace
	default:
		return nil
	}
}

func matchPath(template, path string) bool {
	// Split both paths into segments
	templateParts := strings.Split(strings.Trim(template, "/"), "/")
	pathParts := strings.Split(strings.Trim(path, "/"), "/")

	if len(templateParts) != len(pathParts) {
		return false
	}

	for i, templatePart := range templateParts {
		if strings.HasPrefix(templatePart, "{") && strings.HasSuffix(templatePart, "}") {
			// This is a path parameter, it matches any value
			continue
		}
		if templatePart != pathParts[i] {
			return false
		}
	}

	return true
}

func extractPathParams(path, template string) map[string]string {
	params := make(map[string]string)

	// Split both paths into segments
	pathParts := strings.Split(strings.Trim(path, "/"), "/")
	templateParts := strings.Split(strings.Trim(template, "/"), "/")

	if len(pathParts) != len(templateParts) {
		return params
	}

	for i, templatePart := range templateParts {
		if strings.HasPrefix(templatePart, "{") && strings.HasSuffix(templatePart, "}") {
			paramName := templatePart[1 : len(templatePart)-1]
			params[paramName] = pathParts[i]
		}
	}

	return params
}

func (v *RequestValidator) validatePathParams(op *openapi3.Operation, pathItem *openapi3.PathItem, params map[string]string) ValidationErrors {
	var errors ValidationErrors

	// Collect all path parameters from both pathItem and operation
	allParams := make([]*openapi3.ParameterRef, 0)
	
	// Add path-level parameters first
	if pathItem.Parameters != nil {
		allParams = append(allParams, pathItem.Parameters...)
	}
	
	// Add operation-level parameters (these can override path-level)
	if op.Parameters != nil {
		allParams = append(allParams, op.Parameters...)
	}

	for _, param := range allParams {
		if param.Value.In == "path" {
			value := params[param.Value.Name]
			if value == "" && param.Value.Required {
				errors = append(errors, &ValidationError{
					Field:   fmt.Sprintf("path.%s", param.Value.Name),
					Message: "Required path parameter missing",
					Code:    "missing_required",
				})
				continue
			}

			if err := validateParameterValue(param.Value, value); err != nil {
				errors = append(errors, &ValidationError{
					Field:   fmt.Sprintf("path.%s", param.Value.Name),
					Message: err.Error(),
					Code:    "invalid_format",
				})
			}
		}
	}

	return errors
}

func (v *RequestValidator) validateQueryParams(op *openapi3.Operation, query url.Values) ValidationErrors {
	var errors ValidationErrors

	for _, param := range op.Parameters {
		if param.Value.In == "query" {
			values := query[param.Value.Name]
			if len(values) == 0 {
				if param.Value.Required {
					errors = append(errors, &ValidationError{
						Field:   fmt.Sprintf("query.%s", param.Value.Name),
						Message: "Required query parameter missing",
						Code:    "missing_required",
					})
				}
				continue
			}

			for _, value := range values {
				if err := validateParameterValue(param.Value, value); err != nil {
					errors = append(errors, &ValidationError{
						Field:   fmt.Sprintf("query.%s", param.Value.Name),
						Message: err.Error(),
						Code:    "invalid_format",
					})
				}
			}
		}
	}

	return errors
}

func (v *RequestValidator) validateHeaders(op *openapi3.Operation, headers map[string][]string) ValidationErrors {
	var errors ValidationErrors

	for _, param := range op.Parameters {
		if param.Value.In == "header" {
			values := headers[param.Value.Name]
			if len(values) == 0 {
				if param.Value.Required {
					errors = append(errors, &ValidationError{
						Field:   fmt.Sprintf("header.%s", param.Value.Name),
						Message: "Required header missing",
						Code:    "missing_required",
					})
				}
				continue
			}

			for _, value := range values {
				if err := validateParameterValue(param.Value, value); err != nil {
					errors = append(errors, &ValidationError{
						Field:   fmt.Sprintf("header.%s", param.Value.Name),
						Message: err.Error(),
						Code:    "invalid_format",
					})
				}
			}
		}
	}

	return errors
}

func (v *RequestValidator) validateRequestBody(op *openapi3.Operation, body []byte) ValidationErrors {
	var errors ValidationErrors

	if op.RequestBody == nil || op.RequestBody.Value == nil {
		if len(body) > 0 {
			errors = append(errors, &ValidationError{
				Message: "Request body not allowed",
				Code:    "body_not_allowed",
			})
		}
		return errors
	}

	if len(body) == 0 {
		if op.RequestBody.Value.Required {
			errors = append(errors, &ValidationError{
				Message: "Request body is required",
				Code:    "missing_body",
			})
		}
		return errors
	}

	// Validate content type and schema
	contentType := "application/json" // Default to JSON for now
	content := op.RequestBody.Value.Content.Get(contentType)
	if content == nil {
		errors = append(errors, &ValidationError{
			Message: fmt.Sprintf("Unsupported content type: %s", contentType),
			Code:    "unsupported_content",
		})
		return errors
	}

	if schemaErrs := v.validateSchema(content.Schema, body); len(schemaErrs) > 0 {
		errors = append(errors, schemaErrs...)
	}

	return errors
}

func (v *RequestValidator) validateResponseHeaders(resp *openapi3.Response, headers map[string][]string) ValidationErrors {
	var errors ValidationErrors

	if resp.Headers == nil {
		return errors
	}

	for name, header := range resp.Headers {
		values := headers[name]
		if len(values) == 0 {
			if header.Value.Required {
				errors = append(errors, &ValidationError{
					Field:   fmt.Sprintf("header.%s", name),
					Message: "Required response header missing",
					Code:    "missing_required",
				})
			}
			continue
		}

		for _, value := range values {
			if err := validateParameterValue(&header.Value.Parameter, value); err != nil {
				errors = append(errors, &ValidationError{
					Field:   fmt.Sprintf("header.%s", name),
					Message: err.Error(),
					Code:    "invalid_format",
				})
			}
		}
	}

	return errors
}

func (v *RequestValidator) validateResponseBody(resp *openapi3.Response, body []byte) ValidationErrors {
	var errors ValidationErrors

	if len(body) == 0 {
		return errors
	}

	if resp.Content == nil {
		return errors
	}

	// Validate content type and schema
	contentType := "application/json" // Default to JSON for now
	content := resp.Content.Get(contentType)
	if content == nil {
		errors = append(errors, &ValidationError{
			Message: fmt.Sprintf("Unsupported content type: %s", contentType),
			Code:    "unsupported_content",
		})
		return errors
	}

	if schemaErrs := v.validateSchema(content.Schema, body); len(schemaErrs) > 0 {
		errors = append(errors, schemaErrs...)
	}

	return errors
}

func validateParameterValue(param *openapi3.Parameter, value string) error {
	if param.Schema == nil || param.Schema.Value == nil {
		return nil
	}

	schema := param.Schema.Value

	// Validate based on type
	switch schema.Type {
	case "integer":
		intVal, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return fmt.Errorf("must be an integer")
		}
		// Validate min/max
		if schema.Min != nil && float64(intVal) < *schema.Min {
			return fmt.Errorf("must be >= %v", *schema.Min)
		}
		if schema.Max != nil && float64(intVal) > *schema.Max {
			return fmt.Errorf("must be <= %v", *schema.Max)
		}
	case "number":
		numVal, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("must be a number")
		}
		// Validate min/max
		if schema.Min != nil && numVal < *schema.Min {
			return fmt.Errorf("must be >= %v", *schema.Min)
		}
		if schema.Max != nil && numVal > *schema.Max {
			return fmt.Errorf("must be <= %v", *schema.Max)
		}
	case "boolean":
		if value != "true" && value != "false" {
			return fmt.Errorf("must be a boolean")
		}
	case "string":
		if schema.Format != "" {
			if errs := validateStringFormat(schema.Format, value, ""); len(errs) > 0 {
				return fmt.Errorf(errs[0].Message)
			}
		}
		if schema.Pattern != "" {
			if matched, _ := regexp.MatchString(schema.Pattern, value); !matched {
				return fmt.Errorf("must match pattern: %s", schema.Pattern)
			}
		}
		if schema.MinLength > 0 && len(value) < int(schema.MinLength) {
			return fmt.Errorf("length must be >= %d", schema.MinLength)
		}
		if schema.MaxLength != nil && len(value) > int(*schema.MaxLength) {
			return fmt.Errorf("length must be <= %d", *schema.MaxLength)
		}
	}

	return nil
}

func (v *RequestValidator) validateSchema(schema *openapi3.SchemaRef, data []byte) ValidationErrors {
	var errors ValidationErrors

	if schema == nil || schema.Value == nil {
		return errors
	}

	// Parse JSON data
	var jsonData interface{}
	if err := json.Unmarshal(data, &jsonData); err != nil {
		return append(errors, &ValidationError{
			Message: "Invalid JSON format",
			Code:    "invalid_json",
		})
	}

	// Validate against schema
	if errs := validateValue(schema.Value, jsonData, ""); len(errs) > 0 {
		errors = append(errors, errs...)
	}

	return errors
}

// ValidateSchema validates a value against an OpenAPI schema
func ValidateSchema(schema *openapi3.SchemaRef, data []byte) ValidationErrors {
	var errors ValidationErrors

	if schema == nil || schema.Value == nil {
		return errors
	}

	// Parse JSON data
	var jsonData interface{}
	if err := json.Unmarshal(data, &jsonData); err != nil {
		return append(errors, &ValidationError{
			Message: "Invalid JSON format",
			Code:    "invalid_json",
		})
	}

	// Validate against schema
	if errs := validateValue(schema.Value, jsonData, ""); len(errs) > 0 {
		errors = append(errors, errs...)
	}

	return errors
}

func validateValue(schema *openapi3.Schema, value interface{}, path string) ValidationErrors {
	var errors ValidationErrors

	// Handle nil value
	if value == nil {
		if !schema.Nullable {
			errors = append(errors, &ValidationError{
				Field:   path,
				Message: "value cannot be null",
				Code:    "null_not_allowed",
			})
		}
		return errors
	}

	// Get the schema type
	if schema.Type == "" {
		return errors
	}

	// Validate type
	switch schema.Type {
	case "string":
		str, ok := value.(string)
		if !ok {
			errors = append(errors, &ValidationError{
				Field:   path,
				Message: fmt.Sprintf("expected string, got %T", value),
				Code:    "invalid_type",
			})
			return errors
		}
		if errs := validateString(schema, str, path); len(errs) > 0 {
			errors = append(errors, errs...)
		}

	case "number", "integer":
		num, ok := value.(float64)
		if !ok {
			errors = append(errors, &ValidationError{
				Field:   path,
				Message: fmt.Sprintf("expected number, got %T", value),
				Code:    "invalid_type",
			})
			return errors
		}
		if errs := validateNumber(schema, num, path); len(errs) > 0 {
			errors = append(errors, errs...)
		}

	case "boolean":
		_, ok := value.(bool)
		if !ok {
			errors = append(errors, &ValidationError{
				Field:   path,
				Message: fmt.Sprintf("expected boolean, got %T", value),
				Code:    "invalid_type",
			})
			return errors
		}

	case "array":
		arr, ok := value.([]interface{})
		if !ok {
			errors = append(errors, &ValidationError{
				Field:   path,
				Message: fmt.Sprintf("expected array, got %T", value),
				Code:    "invalid_type",
			})
			return errors
		}
		if errs := validateArray(schema, arr, path); len(errs) > 0 {
			errors = append(errors, errs...)
		}

	case "object":
		obj, ok := value.(map[string]interface{})
		if !ok {
			errors = append(errors, &ValidationError{
				Field:   path,
				Message: fmt.Sprintf("expected object, got %T", value),
				Code:    "invalid_type",
			})
			return errors
		}
		if errs := validateObject(schema, obj, path); len(errs) > 0 {
			errors = append(errors, errs...)
		}
	}

	return errors
}

func validateString(schema *openapi3.Schema, value string, path string) ValidationErrors {
	var errors ValidationErrors

	// Validate length
	if schema.MinLength > 0 && len(value) < int(schema.MinLength) {
		errors = append(errors, &ValidationError{
			Field:   path,
			Message: fmt.Sprintf("string length must be >= %d", schema.MinLength),
			Code:    "min_length",
		})
	}
	if schema.MaxLength != nil && len(value) > int(*schema.MaxLength) {
		errors = append(errors, &ValidationError{
			Field:   path,
			Message: fmt.Sprintf("string length must be <= %d", *schema.MaxLength),
			Code:    "max_length",
		})
	}

	// Validate pattern
	if schema.Pattern != "" {
		if matched, _ := regexp.MatchString(schema.Pattern, value); !matched {
			errors = append(errors, &ValidationError{
				Field:   path,
				Message: fmt.Sprintf("must match pattern: %s", schema.Pattern),
				Code:    "invalid_pattern",
			})
		}
	}

	// Validate format
	if schema.Format != "" {
		if errs := validateStringFormat(schema.Format, value, path); len(errs) > 0 {
			errors = append(errors, errs...)
		}
	}

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
			errors = append(errors, &ValidationError{
				Field:   path,
				Message: fmt.Sprintf("value must be one of: %v", schema.Enum),
				Code:    "invalid_enum",
			})
		}
	}

	return errors
}

func validateStringFormat(format string, value string, path string) ValidationErrors {
	var errors ValidationErrors

	switch format {
	case "email":
		if !strings.Contains(value, "@") {
			errors = append(errors, &ValidationError{
				Field:   path,
				Message: "invalid email format",
				Code:    "invalid_format",
			})
		}
	case "uri":
		if !strings.Contains(value, "://") {
			errors = append(errors, &ValidationError{
				Field:   path,
				Message: "invalid URI format",
				Code:    "invalid_format",
			})
		}
	case "uuid":
		if len(value) != 36 {
			errors = append(errors, &ValidationError{
				Field:   path,
				Message: "invalid UUID format",
				Code:    "invalid_format",
			})
		}
	case "date":
		// Validate date format: YYYY-MM-DD
		datePattern := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
		if !datePattern.MatchString(value) {
			errors = append(errors, &ValidationError{
				Field:   path,
				Message: "invalid date format (expected YYYY-MM-DD)",
				Code:    "invalid_format",
			})
		}
	case "date-time":
		if !strings.Contains(value, "T") || !strings.Contains(value, "Z") {
			errors = append(errors, &ValidationError{
				Field:   path,
				Message: "invalid date-time format (expected RFC3339)",
				Code:    "invalid_format",
			})
		}
	}

	return errors
}

func validateNumber(schema *openapi3.Schema, value float64, path string) ValidationErrors {
	var errors ValidationErrors

	// Validate minimum
	if schema.Min != nil {
		if schema.ExclusiveMin && value <= *schema.Min {
			errors = append(errors, &ValidationError{
				Field:   path,
				Message: fmt.Sprintf("value must be > %v", *schema.Min),
				Code:    "min_value",
			})
		} else if !schema.ExclusiveMin && value < *schema.Min {
			errors = append(errors, &ValidationError{
				Field:   path,
				Message: fmt.Sprintf("value must be >= %v", *schema.Min),
				Code:    "min_value",
			})
		}
	}

	// Validate maximum
	if schema.Max != nil {
		if schema.ExclusiveMax && value >= *schema.Max {
			errors = append(errors, &ValidationError{
				Field:   path,
				Message: fmt.Sprintf("value must be < %v", *schema.Max),
				Code:    "max_value",
			})
		} else if !schema.ExclusiveMax && value > *schema.Max {
			errors = append(errors, &ValidationError{
				Field:   path,
				Message: fmt.Sprintf("value must be <= %v", *schema.Max),
				Code:    "max_value",
			})
		}
	}

	// Validate multiple of
	if schema.MultipleOf != nil {
		if value != float64(int64(value/(*schema.MultipleOf)))*(*schema.MultipleOf) {
			errors = append(errors, &ValidationError{
				Field:   path,
				Message: fmt.Sprintf("value must be a multiple of %v", *schema.MultipleOf),
				Code:    "multiple_of",
			})
		}
	}

	return errors
}

func validateArray(schema *openapi3.Schema, value []interface{}, path string) ValidationErrors {
	var errors ValidationErrors

	// Validate length
	if schema.MinItems > 0 && len(value) < int(schema.MinItems) {
		errors = append(errors, &ValidationError{
			Field:   path,
			Message: fmt.Sprintf("array length must be >= %d", schema.MinItems),
			Code:    "min_items",
		})
	}
	if schema.MaxItems != nil && len(value) > int(*schema.MaxItems) {
		errors = append(errors, &ValidationError{
			Field:   path,
			Message: fmt.Sprintf("array length must be <= %d", *schema.MaxItems),
			Code:    "max_items",
		})
	}

	// Validate items
	if schema.Items != nil && schema.Items.Value != nil {
		for i, item := range value {
			itemPath := fmt.Sprintf("%s[%d]", path, i)
			if errs := validateValue(schema.Items.Value, item, itemPath); len(errs) > 0 {
				errors = append(errors, errs...)
			}
		}
	}

	// Validate uniqueness
	if schema.UniqueItems {
		seen := make(map[interface{}]bool)
		for _, item := range value {
			if seen[item] {
				errors = append(errors, &ValidationError{
					Field:   path,
					Message: "array items must be unique",
					Code:    "unique_items",
				})
				break
			}
			seen[item] = true
		}
	}

	return errors
}

func validateObject(schema *openapi3.Schema, value map[string]interface{}, path string) ValidationErrors {
	var errors ValidationErrors

	// Validate required properties
	for _, required := range schema.Required {
		if _, ok := value[required]; !ok {
			errors = append(errors, &ValidationError{
				Field:   fmt.Sprintf("%s.%s", path, required),
				Message: fmt.Sprintf("missing required property: %s", required),
				Code:    "required",
			})
		}
	}

	// Validate properties
	for propName, propValue := range value {
		propPath := fmt.Sprintf("%s.%s", path, propName)

		// Check if property is defined in schema
		if propSchema, ok := schema.Properties[propName]; ok {
			if errs := validateValue(propSchema.Value, propValue, propPath); len(errs) > 0 {
				errors = append(errors, errs...)
			}
		} else if schema.AdditionalProperties.Has != nil && !*schema.AdditionalProperties.Has {
			errors = append(errors, &ValidationError{
				Field:   propPath,
				Message: fmt.Sprintf("additional property %s not allowed", propName),
				Code:    "additional_properties",
			})
		} else if schema.AdditionalProperties.Schema != nil {
			if errs := validateValue(schema.AdditionalProperties.Schema.Value, propValue, propPath); len(errs) > 0 {
				errors = append(errors, errs...)
			}
		}
	}

	// Validate min/max properties
	if schema.MinProps > 0 && len(value) < int(schema.MinProps) {
		errors = append(errors, &ValidationError{
			Field:   path,
			Message: fmt.Sprintf("object must have >= %d properties", schema.MinProps),
			Code:    "min_properties",
		})
	}
	if schema.MaxProps != nil && len(value) > int(*schema.MaxProps) {
		errors = append(errors, &ValidationError{
			Field:   path,
			Message: fmt.Sprintf("object must have <= %d properties", *schema.MaxProps),
			Code:    "max_properties",
		})
	}

	return errors
} 