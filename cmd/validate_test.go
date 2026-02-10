package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateRequest(t *testing.T) {
	// Create test OpenAPI spec
	specYAML := `
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /users:
    post:
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required: [name, email]
              properties:
                name:
                  type: string
                  minLength: 3
                email:
                  type: string
                  format: email
      responses:
        '200':
          description: Success
  /items:
    get:
      parameters:
        - name: limit
          in: query
          required: true
          schema:
            type: integer
            minimum: 1
            maximum: 100
        - name: X-API-Key
          in: header
          required: true
          schema:
            type: string
      responses:
        '200':
          description: Success
`
	specPath := filepath.Join(t.TempDir(), "openapi.yaml")
	err := os.WriteFile(specPath, []byte(specYAML), 0644)
	require.NoError(t, err)

	tests := []struct {
		name        string
		request     RequestData
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid user creation request",
			request: RequestData{
				Method: "POST",
				Path:   "/users",
				Headers: map[string]string{
					"Content-Type": "application/json",
				},
				Body: json.RawMessage(`{
					"name": "John Doe",
					"email": "john@example.com"
				}`),
			},
			expectError: false,
		},
		{
			name: "invalid user creation - missing required field",
			request: RequestData{
				Method: "POST",
				Path:   "/users",
				Headers: map[string]string{
					"Content-Type": "application/json",
				},
				Body: json.RawMessage(`{
					"name": "John Doe"
				}`),
			},
			expectError: true,
			errorMsg:    "missing required property: email",
		},
		{
			name: "invalid user creation - invalid email format",
			request: RequestData{
				Method: "POST",
				Path:   "/users",
				Headers: map[string]string{
					"Content-Type": "application/json",
				},
				Body: json.RawMessage(`{
					"name": "John Doe",
					"email": "invalid-email"
				}`),
			},
			expectError: true,
			errorMsg:    "invalid email format",
		},
		{
			name: "invalid user creation - name too short",
			request: RequestData{
				Method: "POST",
				Path:   "/users",
				Headers: map[string]string{
					"Content-Type": "application/json",
				},
				Body: json.RawMessage(`{
					"name": "Jo",
					"email": "john@example.com"
				}`),
			},
			expectError: true,
			errorMsg:    "string too short, minimum 3",
		},
		{
			name: "valid items request",
			request: RequestData{
				Method: "GET",
				Path:   "/items",
				Headers: map[string]string{
					"X-API-Key": "test-key",
				},
				Query: "limit=50",
			},
			expectError: false,
		},
		{
			name: "invalid items request - missing required header",
			request: RequestData{
				Method: "GET",
				Path:   "/items",
				Query:  "limit=50",
			},
			expectError: true,
			errorMsg:    "missing required header: X-API-Key",
		},
		{
			name: "invalid items request - invalid limit",
			request: RequestData{
				Method: "GET",
				Path:   "/items",
				Headers: map[string]string{
					"X-API-Key": "test-key",
				},
				Query: "limit=150",
			},
			expectError: true,
			errorMsg:    "greater than maximum",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request file
			requestData, err := json.Marshal(tt.request)
			require.NoError(t, err)

			requestPath := filepath.Join(t.TempDir(), "request.json")
			err = os.WriteFile(requestPath, requestData, 0644)
			require.NoError(t, err)

			// Run validation
			err = validateRequest(loadTestSpec(t, specPath), requestPath, false)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateResponse(t *testing.T) {
	// Create test OpenAPI spec
	specYAML := `
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /users/{id}:
    get:
      responses:
        '200':
          description: Success
          headers:
            X-Rate-Limit:
              required: true
              schema:
                type: integer
          content:
            application/json:
              schema:
                type: object
                required: [id, name, email]
                properties:
                  id:
                    type: integer
                  name:
                    type: string
                  email:
                    type: string
                    format: email
                  age:
                    type: integer
                    minimum: 0
                    maximum: 120
        '404':
          description: Not Found
          content:
            application/json:
              schema:
                type: object
                required: [error]
                properties:
                  error:
                    type: string
`
	specPath := filepath.Join(t.TempDir(), "openapi.yaml")
	err := os.WriteFile(specPath, []byte(specYAML), 0644)
	require.NoError(t, err)

	tests := []struct {
		name        string
		response    ResponseData
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid success response",
			response: ResponseData{
				StatusCode: 200,
				Headers: map[string]string{
					"Content-Type":   "application/json",
					"X-Rate-Limit":  "100",
				},
				Body: json.RawMessage(`{
					"id": 1,
					"name": "John Doe",
					"email": "john@example.com",
					"age": 30
				}`),
			},
			expectError: false,
		},
		{
			name: "valid error response",
			response: ResponseData{
				StatusCode: 404,
				Headers: map[string]string{
					"Content-Type": "application/json",
				},
				Body: json.RawMessage(`{
					"error": "User not found"
				}`),
			},
			expectError: false,
		},
		{
			name: "invalid response - missing required header",
			response: ResponseData{
				StatusCode: 200,
				Headers: map[string]string{
					"Content-Type": "application/json",
				},
				Body: json.RawMessage(`{
					"id": 1,
					"name": "John Doe",
					"email": "john@example.com"
				}`),
			},
			expectError: true,
			errorMsg:    "missing required header: X-Rate-Limit",
		},
		{
			name: "invalid response - missing required field",
			response: ResponseData{
				StatusCode: 200,
				Headers: map[string]string{
					"Content-Type":   "application/json",
					"X-Rate-Limit":  "100",
				},
				Body: json.RawMessage(`{
					"id": 1,
					"name": "John Doe"
				}`),
			},
			expectError: true,
			errorMsg:    "missing required property: email",
		},
		{
			name: "invalid response - invalid email format",
			response: ResponseData{
				StatusCode: 200,
				Headers: map[string]string{
					"Content-Type":   "application/json",
					"X-Rate-Limit":  "100",
				},
				Body: json.RawMessage(`{
					"id": 1,
					"name": "John Doe",
					"email": "invalid-email"
				}`),
			},
			expectError: true,
			errorMsg:    "invalid email format",
		},
		{
			name: "invalid response - age out of range",
			response: ResponseData{
				StatusCode: 200,
				Headers: map[string]string{
					"Content-Type":   "application/json",
					"X-Rate-Limit":  "100",
				},
				Body: json.RawMessage(`{
					"id": 1,
					"name": "John Doe",
					"email": "john@example.com",
					"age": 150
				}`),
			},
			expectError: true,
			errorMsg:    "greater than maximum",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create response file
			responseData, err := json.Marshal(tt.response)
			require.NoError(t, err)

			responsePath := filepath.Join(t.TempDir(), "response.json")
			err = os.WriteFile(responsePath, responseData, 0644)
			require.NoError(t, err)

			// Run validation
			err = validateResponse(loadTestSpec(t, specPath), responsePath, false)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTimeFormatParsing(t *testing.T) {
	// The magic reference time in Go
	referenceTime := time.Date(2006, time.January, 2, 15, 4, 5, 0, time.UTC)

	// Time-only formats (contain hour indicators like 15, 3, 04, 05)
	timeFormats := map[string]string{
		"15:04:05":     "Hour:Minute:Second (24h)",
		"3:04:05PM":    "Hour:Minute:Second (12h)",
		"15:04":        "Hour:Minute",
		"15:04:05.000": "Hour:Minute:Second.Milliseconds",
	}

	// Date formats
	dateFormats := map[string]string{
		"2006-01-02":     "Year-Month-Day",
		"Jan 2":          "Month Day",
		"January 2 2006": "Month Day Year",
	}

	for layout, description := range timeFormats {
		formatted := referenceTime.Format(layout)
		t.Logf("%s (%s) = %s", description, layout, formatted)

		// Parse it back
		parsed, err := time.Parse(layout, formatted)
		require.NoError(t, err)

		// For time formats, compare hour and minute
		require.Equal(t, referenceTime.Hour(), parsed.Hour(), "hour mismatch for %s", layout)
		require.Equal(t, referenceTime.Minute(), parsed.Minute(), "minute mismatch for %s", layout)
		if strings.Contains(layout, ":05") {
			require.Equal(t, referenceTime.Second(), parsed.Second(), "second mismatch for %s", layout)
		}
	}

	for layout, description := range dateFormats {
		formatted := referenceTime.Format(layout)
		t.Logf("%s (%s) = %s", description, layout, formatted)

		// Parse it back
		parsed, err := time.Parse(layout, formatted)
		require.NoError(t, err)

		// For date formats, just verify round-trip formatting works
		require.Equal(t, formatted, parsed.Format(layout), "round-trip failed for %s", layout)
	}
}

func loadTestSpec(t *testing.T, path string) *openapi3.T {
	loader := openapi3.NewLoader()
	spec, err := loader.LoadFromFile(path)
	require.NoError(t, err)
	return spec
} 