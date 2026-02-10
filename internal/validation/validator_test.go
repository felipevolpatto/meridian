package validation

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/assert"
)

func TestRequestValidator_ValidateRequest(t *testing.T) {
	// Load test OpenAPI spec
	loader := openapi3.NewLoader()
	spec, err := loader.LoadFromFile("../../docs/openapi.yaml")
	if err != nil {
		t.Fatalf("Failed to load OpenAPI spec: %v", err)
	}

	validator := NewRequestValidator(spec)

	tests := []struct {
		name           string
		method         string
		path          string
		headers        map[string][]string
		query         url.Values
		body          []byte
		expectedError bool
		errorCode     string
	}{
		{
			name:   "Valid GET users request",
			method: "GET",
			path:   "/users",
			headers: map[string][]string{
				"Accept-Language": {"en-US"},
				"X-API-Key":      {"test-key"},
			},
			query: url.Values{
				"limit":  {"10"},
				"offset": {"0"},
			},
			expectedError: false,
		},
		{
			name:   "Invalid limit parameter",
			method: "GET",
			path:   "/users",
			headers: map[string][]string{
				"X-API-Key": {"test-key"},
			},
			query: url.Values{
				"limit": {"1000"}, // Exceeds maximum
			},
			expectedError: true,
			errorCode:    "invalid_format",
		},
		{
			name:   "Invalid Accept-Language format",
			method: "GET",
			path:   "/users",
			headers: map[string][]string{
				"Accept-Language": {"invalid"},
				"X-API-Key":      {"test-key"},
			},
			expectedError: true,
			errorCode:    "invalid_format",
		},
		{
			name:   "Valid POST user request",
			method: "POST",
			path:   "/users",
			headers: map[string][]string{
				"Authorization": {"Bearer test-token"},
				"Content-Type": {"application/json"},
			},
			body: []byte(`{
				"name": "John Doe",
				"email": "john@example.com",
				"age": 30,
				"preferences": {
					"newsletter": true,
					"theme": "dark"
				}
			}`),
			expectedError: false,
		},
		{
			name:   "Invalid POST user request - missing required fields",
			method: "POST",
			path:   "/users",
			headers: map[string][]string{
				"Authorization": {"Bearer test-token"},
				"Content-Type": {"application/json"},
			},
			body: []byte(`{
				"age": 30
			}`),
			expectedError: true,
			errorCode:    "required",
		},
		{
			name:   "Invalid POST user request - invalid email format",
			method: "POST",
			path:   "/users",
			headers: map[string][]string{
				"Authorization": {"Bearer test-token"},
				"Content-Type": {"application/json"},
			},
			body: []byte(`{
				"name": "John Doe",
				"email": "invalid-email",
				"age": 30
			}`),
			expectedError: true,
			errorCode:    "invalid_format",
		},
		{
			name:   "Invalid POST user request - invalid age",
			method: "POST",
			path:   "/users",
			headers: map[string][]string{
				"Authorization": {"Bearer test-token"},
				"Content-Type": {"application/json"},
			},
			body: []byte(`{
				"name": "John Doe",
				"email": "john@example.com",
				"age": 15
			}`),
			expectedError: true,
			errorCode:    "min_value",
		},
		{
			name:   "Valid PUT user request",
			method: "PUT",
			path:   "/users/1",
			headers: map[string][]string{
				"Authorization": {"Bearer test-token"},
				"Content-Type": {"application/json"},
			},
			body: []byte(`{
				"name": "John Smith",
				"email": "john.smith@example.com"
			}`),
			expectedError: false,
		},
		{
			name:   "Invalid path parameter",
			method: "GET",
			path:   "/users/invalid",
			headers: map[string][]string{
				"X-API-Key": {"test-key"},
			},
			expectedError: true,
			errorCode:    "invalid_format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := validator.ValidateRequest(tt.method, tt.path, tt.headers, tt.query, tt.body)
			
			if tt.expectedError {
				assert.NotEmpty(t, errors, "Expected validation errors")
				if tt.errorCode != "" {
					found := false
					for _, err := range errors {
						if err.Code == tt.errorCode {
							found = true
							break
						}
					}
					assert.True(t, found, "Expected error code %s not found", tt.errorCode)
				}
			} else {
				assert.Empty(t, errors, "Expected no validation errors, got: %v", errors)
			}
		})
	}
}

func TestResponseValidator_ValidateResponse(t *testing.T) {
	// Load test OpenAPI spec
	loader := openapi3.NewLoader()
	spec, err := loader.LoadFromFile("../../docs/openapi.yaml")
	if err != nil {
		t.Fatalf("Failed to load OpenAPI spec: %v", err)
	}

	validator := NewResponseValidator(spec)

	tests := []struct {
		name           string
		method         string
		path          string
		statusCode     int
		headers        http.Header
		body          []byte
		expectedError bool
		errorCode     string
	}{
		{
			name:       "Valid GET users response",
			method:    "GET",
			path:      "/users",
			statusCode: 200,
			headers: http.Header{
				"Content-Type":           {"application/json"},
				"X-Total-Count":         {"100"},
				"X-Rate-Limit-Remaining": {"50"},
			},
			body: []byte(`[{
				"id": 1,
				"name": "John Doe",
				"email": "john@example.com",
				"age": 30,
				"preferences": {
					"newsletter": true,
					"theme": "dark"
				},
				"createdAt": "2024-01-01T00:00:00Z",
				"updatedAt": "2024-01-01T00:00:00Z"
			}]`),
			expectedError: false,
		},
		{
			name:       "Invalid content type",
			method:    "GET",
			path:      "/users",
			statusCode: 200,
			headers: http.Header{
				"Content-Type": {"text/plain"},
			},
			body:          []byte(`Hello World`),
			expectedError: true,
			errorCode:    "unsupported_content_type",
		},
		{
			name:       "Missing required response header",
			method:    "GET",
			path:      "/users",
			statusCode: 200,
			headers: http.Header{
				"Content-Type": {"application/json"},
			},
			body:          []byte(`[]`),
			expectedError: false,
			errorCode:    "",
		},
		{
			name:       "Invalid response body schema",
			method:    "GET",
			path:      "/users",
			statusCode: 200,
			headers: http.Header{
				"Content-Type":           {"application/json"},
				"X-Total-Count":         {"100"},
				"X-Rate-Limit-Remaining": {"50"},
			},
			body:          []byte(`[{"invalid": "data"}]`),
			expectedError: true,
			errorCode:    "required",
		},
		{
			name:       "Undefined status code",
			method:    "GET",
			path:      "/users",
			statusCode: 418,
			headers: http.Header{
				"Content-Type": {"application/json"},
			},
			body:          []byte(`{"error": "I'm a teapot"}`),
			expectedError: true,
			errorCode:    "status_code_not_found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := validator.ValidateResponse(tt.path, tt.method, tt.statusCode, tt.headers, tt.body)
			
			if tt.expectedError {
				assert.NotEmpty(t, errors, "Expected validation errors")
				if tt.errorCode != "" {
					found := false
					for _, err := range errors {
						if err.Code == tt.errorCode {
							found = true
							break
						}
					}
					assert.True(t, found, "Expected error code %s not found", tt.errorCode)
				}
			} else {
				assert.Empty(t, errors, "Expected no validation errors, got: %v", errors)
			}
		})
	}
}

func TestSecurityValidator_ValidateSecurity(t *testing.T) {
	// Load test OpenAPI spec
	loader := openapi3.NewLoader()
	spec, err := loader.LoadFromFile("../../docs/openapi.yaml")
	if err != nil {
		t.Fatalf("Failed to load OpenAPI spec: %v", err)
	}

	validator := NewSecurityValidator(spec)

	// Get the operation for testing
	path := spec.Paths.Find("/users")
	if path == nil {
		t.Fatal("Path /users not found in spec")
	}
	postOp := path.Post

	tests := []struct {
		name           string
		headers        http.Header
		query         map[string][]string
		expectedError bool
		errorCode     string
	}{
		{
			name: "Valid Bearer token",
			headers: http.Header{
				"Authorization": {"Bearer valid-token"},
			},
			expectedError: false,
		},
		{
			name: "Missing Authorization header",
			headers: http.Header{},
			expectedError: true,
			errorCode:    "missing_authorization",
		},
		{
			name: "Invalid Authorization scheme",
			headers: http.Header{
				"Authorization": {"Basic invalid-token"},
			},
			expectedError: true,
			errorCode:    "invalid_auth_scheme",
		},
		{
			name: "Empty Bearer token",
			headers: http.Header{
				"Authorization": {"Bearer "},
			},
			expectedError: true,
			errorCode:    "missing_bearer_token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := validator.ValidateSecurity(postOp, tt.headers, tt.query)
			
			if tt.expectedError {
				assert.NotEmpty(t, errors, "Expected validation errors")
				if tt.errorCode != "" {
					found := false
					for _, err := range errors {
						if err.Code == tt.errorCode {
							found = true
							break
						}
					}
					assert.True(t, found, "Expected error code %s not found", tt.errorCode)
				}
			} else {
				assert.Empty(t, errors, "Expected no validation errors, got: %v", errors)
			}
		})
	}
}