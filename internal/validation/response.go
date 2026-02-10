package validation

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

// ResponseValidator handles validation of responses against OpenAPI spec
type ResponseValidator struct {
	spec *openapi3.T
}

// NewResponseValidator creates a new response validator
func NewResponseValidator(spec *openapi3.T) *ResponseValidator {
	return &ResponseValidator{spec: spec}
}

// ValidateResponse validates a response against the OpenAPI spec
func (v *ResponseValidator) ValidateResponse(path, method string, statusCode int, headers http.Header, body []byte) ValidationErrors {
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

	// Get the response specification for this status code
	response := v.getResponse(op, statusCode)
	if response == nil {
		return append(errors, &ValidationError{
			Message: fmt.Sprintf("Status code %d not defined in API specification", statusCode),
			Code:    "status_code_not_found",
		})
	}

	// Validate response headers
	if headerErrs := v.validateResponseHeaders(response, headers); len(headerErrs) > 0 {
		errors = append(errors, headerErrs...)
	}

	// Validate response content type and body
	if contentErrs := v.validateResponseContent(response, headers.Get("Content-Type"), body); len(contentErrs) > 0 {
		errors = append(errors, contentErrs...)
	}

	return errors
}

func (v *ResponseValidator) findPath(path string) *openapi3.PathItem {
	// First try exact match
	if pathItem := v.spec.Paths.Find(path); pathItem != nil {
		return pathItem
	}

	// Try to match path templates
	for specPath, pathItem := range v.spec.Paths.Map() {
		if pathMatches(specPath, path) {
			return pathItem
		}
	}

	return nil
}

func (v *ResponseValidator) getOperation(pathItem *openapi3.PathItem, method string) *openapi3.Operation {
	switch strings.ToUpper(method) {
	case "GET":
		return pathItem.Get
	case "POST":
		return pathItem.Post
	case "PUT":
		return pathItem.Put
	case "DELETE":
		return pathItem.Delete
	case "OPTIONS":
		return pathItem.Options
	case "HEAD":
		return pathItem.Head
	case "PATCH":
		return pathItem.Patch
	case "TRACE":
		return pathItem.Trace
	default:
		return nil
	}
}

func (v *ResponseValidator) getResponse(op *openapi3.Operation, statusCode int) *openapi3.Response {
	if op.Responses == nil {
		return nil
	}

	// Try exact status code
	if response := op.Responses.Status(statusCode); response != nil {
		return response.Value
	}

	// Try default response
	if defaultResponse := op.Responses.Default(); defaultResponse != nil {
		return defaultResponse.Value
	}

	return nil
}

func (v *ResponseValidator) validateResponseHeaders(response *openapi3.Response, headers http.Header) ValidationErrors {
	var errors ValidationErrors

	if response.Headers == nil {
		return errors
	}

	// Check required headers are present
	for name, header := range response.Headers {
		if header.Value.Required {
			if _, exists := headers[name]; !exists {
				errors = append(errors, &ValidationError{
					Field:   fmt.Sprintf("header.%s", name),
					Message: "Required header is missing",
					Code:    "missing_required_header",
				})
				continue
			}
		}

		// Validate header value if present
		if values, exists := headers[name]; exists && header.Value.Schema != nil {
			for _, value := range values {
				if err := validateParameterValue(&openapi3.Parameter{
					Schema: header.Value.Schema,
				}, value); err != nil {
					errors = append(errors, &ValidationError{
						Field:   fmt.Sprintf("header.%s", name),
						Message: err.Error(),
						Code:    "invalid_header_value",
					})
				}
			}
		}
	}

	return errors
}

func (v *ResponseValidator) validateResponseContent(response *openapi3.Response, contentType string, body []byte) ValidationErrors {
	var errors ValidationErrors

	if len(body) == 0 {
		return errors
	}

	if response.Content == nil {
		errors = append(errors, &ValidationError{
			Message: "Response body not allowed",
			Code:    "unexpected_body",
		})
		return errors
	}

	// Find matching content type
	var matchedContent *openapi3.MediaType
	var matchedContentType string

	for ct, content := range response.Content {
		if matchContentType(ct, contentType) {
			matchedContent = content
			matchedContentType = ct
			break
		}
	}

	if matchedContent == nil {
		errors = append(errors, &ValidationError{
			Field:   "Content-Type",
			Message: fmt.Sprintf("Unsupported content type: %s", contentType),
			Code:    "unsupported_content_type",
		})
		return errors
	}

	// Validate schema if present
	if matchedContent.Schema != nil {
		var data interface{}
		if strings.HasPrefix(matchedContentType, "application/json") {
			if err := json.Unmarshal(body, &data); err != nil {
				errors = append(errors, &ValidationError{
					Message: "Invalid JSON format",
					Code:    "invalid_json",
				})
				return errors
			}
		}

		if errs := validateValue(matchedContent.Schema.Value, data, ""); len(errs) > 0 {
			errors = append(errors, errs...)
		}
	}

	return errors
}

func matchContentType(spec, actual string) bool {
	spec = strings.ToLower(strings.TrimSpace(spec))
	actual = strings.ToLower(strings.TrimSpace(actual))

	// Extract main type/subtype
	specParts := strings.Split(spec, ";")
	actualParts := strings.Split(actual, ";")

	return specParts[0] == actualParts[0] || specParts[0] == "*/*"
}

// pathMatches checks if an actual path matches a spec path template
func pathMatches(specPath, actualPath string) bool {
	specSegments := strings.Split(strings.Trim(specPath, "/"), "/")
	actualSegments := strings.Split(strings.Trim(actualPath, "/"), "/")

	if len(specSegments) != len(actualSegments) {
		return false
	}

	for i := range specSegments {
		if !strings.HasPrefix(specSegments[i], "{") || !strings.HasSuffix(specSegments[i], "}") {
			if specSegments[i] != actualSegments[i] {
				return false
			}
		}
	}

	return true
} 