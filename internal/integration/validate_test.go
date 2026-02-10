package integration

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateWithLiveServer(t *testing.T) {
	// Skip if meridian binary is not available
	if _, err := exec.LookPath("meridian"); err != nil {
		t.Skip("Skipping test: meridian binary not found in PATH. Run 'go install' first.")
	}
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
        '201':
          description: Created
          headers:
            Location:
              required: true
              schema:
                type: string
                format: uri
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
        '400':
          description: Bad Request
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

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && r.URL.Path == "/users" {
			// Parse request body
			var user struct {
				Name  string `json:"name"`
				Email string `json:"email"`
			}
			err := json.NewDecoder(r.Body).Decode(&user)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{
					"error": "Invalid JSON",
				})
				return
			}

			// Validate request
			if user.Name == "" || user.Email == "" {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{
					"error": "Name and email are required",
				})
				return
			}

			// Create response
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Location", "/users/1")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":    1,
				"name":  user.Name,
				"email": user.Email,
			})
		}
	}))
	defer server.Close()

	// Test cases
	tests := []struct {
		name           string
		requestBody    interface{}
		expectedStatus int
		expectError    bool
	}{
		{
			name: "valid request and response",
			requestBody: map[string]string{
				"name":  "John Doe",
				"email": "john@example.com",
			},
			expectedStatus: http.StatusCreated,
			expectError:    false,
		},
		{
			name: "invalid request - missing email",
			requestBody: map[string]string{
				"name": "John Doe",
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    false, // Error response is valid according to spec
		},
		{
			name: "invalid request - name too short",
			requestBody: map[string]string{
				"name":  "Jo",
				"email": "john@example.com",
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    false, // Error response is valid according to spec
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Make request
			requestBody, err := json.Marshal(tt.requestBody)
			require.NoError(t, err)

			req, err := http.NewRequest("POST", server.URL+"/users", strings.NewReader(string(requestBody)))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			// Create request file for validation
			requestData := map[string]interface{}{
				"method": "POST",
				"path":   "/users",
				"headers": map[string]string{
					"Content-Type": "application/json",
				},
				"body": requestBody,
			}
			requestBytes, err := json.Marshal(requestData)
			require.NoError(t, err)

			requestPath := filepath.Join(t.TempDir(), "request.json")
			err = os.WriteFile(requestPath, requestBytes, 0644)
			require.NoError(t, err)

			// Create response file for validation
			var responseBody interface{}
			err = json.NewDecoder(resp.Body).Decode(&responseBody)
			require.NoError(t, err)

			responseData := map[string]interface{}{
				"status_code": resp.StatusCode,
				"headers": map[string]string{
					"Content-Type": resp.Header.Get("Content-Type"),
					"Location":    resp.Header.Get("Location"),
				},
				"body": responseBody,
			}
			responseBytes, err := json.Marshal(responseData)
			require.NoError(t, err)

			responsePath := filepath.Join(t.TempDir(), "response.json")
			err = os.WriteFile(responsePath, responseBytes, 0644)
			require.NoError(t, err)

			// Run validation command
			cmd := exec.Command("meridian", "validate",
				"--spec", specPath,
				"--request", requestPath,
				"--response", responsePath,
			)
			output, err := cmd.CombinedOutput()

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, string(output), "validation failed")
			} else {
				assert.NoError(t, err)
				assert.Contains(t, string(output), "✅")
			}
		})
	}
}

func TestValidateWithComplexSchema(t *testing.T) {
	// Skip if meridian binary is not available
	if _, err := exec.LookPath("meridian"); err != nil {
		t.Skip("Skipping test: meridian binary not found in PATH. Run 'go install' first.")
	}

	// Create test OpenAPI spec with complex validation rules
	specYAML := `
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /products:
    post:
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required: [name, price, categories, attributes]
              properties:
                name:
                  type: string
                  minLength: 3
                  maxLength: 100
                  pattern: "^[a-zA-Z0-9\\s-]+$"
                price:
                  type: number
                  minimum: 0.01
                  maximum: 999999.99
                  multipleOf: 0.01
                categories:
                  type: array
                  minItems: 1
                  maxItems: 5
                  uniqueItems: true
                  items:
                    type: string
                    enum: [electronics, clothing, books, food, toys]
                attributes:
                  type: object
                  minProperties: 1
                  maxProperties: 10
                  additionalProperties:
                    type: string
                created_at:
                  type: string
                  format: date-time
                metadata:
                  type: object
                  properties:
                    weight:
                      type: number
                      minimum: 0
                    dimensions:
                      type: object
                      required: [length, width, height]
                      properties:
                        length:
                          type: number
                          minimum: 0
                        width:
                          type: number
                          minimum: 0
                        height:
                          type: number
                          minimum: 0
                    tags:
                      type: array
                      items:
                        type: string
                        minLength: 2
`
	specPath := filepath.Join(t.TempDir(), "openapi.yaml")
	err := os.WriteFile(specPath, []byte(specYAML), 0644)
	require.NoError(t, err)

	tests := []struct {
		name        string
		request     interface{}
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid complex request",
			request: map[string]interface{}{
				"name":  "Gaming Laptop XYZ-123",
				"price": 999.99,
				"categories": []string{
					"electronics",
					"toys",
				},
				"attributes": map[string]string{
					"brand":  "TechCo",
					"color":  "black",
					"model":  "XYZ-123",
				},
				"created_at": "2024-01-01T12:00:00Z",
				"metadata": map[string]interface{}{
					"weight": 2.5,
					"dimensions": map[string]interface{}{
						"length": 35.5,
						"width":  25.0,
						"height": 2.5,
					},
					"tags": []string{
						"gaming",
						"laptop",
						"high-performance",
					},
				},
			},
			expectError: false,
		},
		{
			name: "invalid name pattern",
			request: map[string]interface{}{
				"name":  "Gaming Laptop @#$",
				"price": 999.99,
				"categories": []string{
					"electronics",
				},
				"attributes": map[string]string{
					"brand": "TechCo",
				},
			},
			expectError: true,
			errorMsg:    "does not match pattern",
		},
		{
			name: "invalid price - not multiple of 0.01",
			request: map[string]interface{}{
				"name":  "Gaming Laptop XYZ-123",
				"price": 999.999,
				"categories": []string{
					"electronics",
				},
				"attributes": map[string]string{
					"brand": "TechCo",
				},
			},
			expectError: true,
			errorMsg:    "not multiple of",
		},
		{
			name: "invalid categories - duplicate items",
			request: map[string]interface{}{
				"name":  "Gaming Laptop XYZ-123",
				"price": 999.99,
				"categories": []string{
					"electronics",
					"electronics",
				},
				"attributes": map[string]string{
					"brand": "TechCo",
				},
			},
			expectError: true,
			errorMsg:    "duplicate items not allowed",
		},
		{
			name: "invalid metadata - negative dimensions",
			request: map[string]interface{}{
				"name":  "Gaming Laptop XYZ-123",
				"price": 999.99,
				"categories": []string{
					"electronics",
				},
				"attributes": map[string]string{
					"brand": "TechCo",
				},
				"metadata": map[string]interface{}{
					"dimensions": map[string]interface{}{
						"length": -1,
						"width":  25.0,
						"height": 2.5,
					},
				},
			},
			expectError: true,
			errorMsg:    "less than minimum",
		},
		{
			name: "invalid metadata - short tag",
			request: map[string]interface{}{
				"name":  "Gaming Laptop XYZ-123",
				"price": 999.99,
				"categories": []string{
					"electronics",
				},
				"attributes": map[string]string{
					"brand": "TechCo",
				},
				"metadata": map[string]interface{}{
					"tags": []string{
						"a",
					},
				},
			},
			expectError: true,
			errorMsg:    "string too short",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request file
			requestData := map[string]interface{}{
				"method": "POST",
				"path":   "/products",
				"headers": map[string]string{
					"Content-Type": "application/json",
				},
				"body": tt.request,
			}
			requestBytes, err := json.Marshal(requestData)
			require.NoError(t, err)

			requestPath := filepath.Join(t.TempDir(), "request.json")
			err = os.WriteFile(requestPath, requestBytes, 0644)
			require.NoError(t, err)

			// Run validation command
			cmd := exec.Command("meridian", "validate",
				"--spec", specPath,
				"--request", requestPath,
			)
			output, err := cmd.CombinedOutput()

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, string(output), tt.errorMsg)
			} else {
				assert.NoError(t, err)
				assert.Contains(t, string(output), "✅")
			}
		})
	}
} 