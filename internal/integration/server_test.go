package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/felipevolpatto/meridian/internal/config"
	"github.com/felipevolpatto/meridian/internal/server"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestServer(t *testing.T) (*server.Server, http.Handler, func()) {
	// Create a temporary OpenAPI spec file
	specContent := `
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /users:
    get:
      operationId: listUsers
      responses:
        '200':
          description: List of users
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: '#/components/schemas/User'
    post:
      operationId: createUser
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/UserInput'
      responses:
        '201':
          description: User created
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/User'
  /users/{id}:
    get:
      operationId: getUser
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: string
      responses:
        '200':
          description: User details
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/User'
        '404':
          description: User not found
    put:
      operationId: updateUser
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: string
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/UserInput'
      responses:
        '200':
          description: User updated
    delete:
      operationId: deleteUser
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: string
      responses:
        '204':
          description: User deleted
components:
  schemas:
    User:
      type: object
      properties:
        id:
          type: string
        name:
          type: string
        email:
          type: string
          format: email
      required:
        - id
        - name
        - email
    UserInput:
      type: object
      properties:
        name:
          type: string
        email:
          type: string
          format: email
      required:
        - name
        - email
`

	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "meridian-test-*")
	require.NoError(t, err)

	// Write OpenAPI spec to temporary file
	specFile := filepath.Join(tmpDir, "openapi.yaml")
	err = os.WriteFile(specFile, []byte(specContent), 0644)
	require.NoError(t, err)

	// Create temporary state file
	stateFile := filepath.Join(tmpDir, "test.db")

	// Create configuration
	cfg := &config.Config{
		OpenAPI: specFile,
		Server: config.ServerConfig{
			Address: "localhost",
			Port:    8080,
		},
		State: config.StateConfig{
			Persistence: stateFile,
		},
		Behavior: config.BehaviorConfig{
			Errors:  config.ErrorConfig{Enabled: false, Rate: 0},
			Latency: config.LatencyConfig{Enabled: false, Min: 0, Max: 0},
			CORS: config.CORSConfig{
				Enabled:        true,
				AllowedOrigins: []string{"*"},
				AllowedMethods: []string{"GET", "POST", "PUT", "DELETE"},
				AllowedHeaders: []string{"Content-Type"},
			},
		},
	}

	// Parse OpenAPI spec
	loader := openapi3.NewLoader()
	spec, err := loader.LoadFromFile(specFile)
	require.NoError(t, err)

	// Create server
	srv := server.NewServer(spec, cfg)
	require.NotNil(t, srv)

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return srv, srv, cleanup
}

func TestServerIntegration(t *testing.T) {
	_, handler, cleanup := setupTestServer(t)
	defer cleanup()

	t.Run("create and retrieve user", func(t *testing.T) {
		// Create a new user
		userInput := map[string]interface{}{
			"id":    "1",
			"name":  "John Doe",
			"email": "john@example.com",
		}
		inputBytes, err := json.Marshal(userInput)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewReader(inputBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

		var createdUser map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &createdUser)
		require.NoError(t, err)
		assert.Equal(t, "John Doe", createdUser["name"])
		assert.Equal(t, "john@example.com", createdUser["email"])
		assert.Equal(t, "1", createdUser["id"])

		// Retrieve the created user
		req = httptest.NewRequest(http.MethodGet, "/users/1", nil)
		w = httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

		var retrievedUser map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &retrievedUser)
		require.NoError(t, err)
		assert.Equal(t, createdUser["name"], retrievedUser["name"])
		assert.Equal(t, createdUser["email"], retrievedUser["email"])

		// List all users
		req = httptest.NewRequest(http.MethodGet, "/users", nil)
		w = httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

		var users []map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &users)
		require.NoError(t, err)
		assert.Len(t, users, 1)
	})

	t.Run("update user", func(t *testing.T) {
		// Update the user
		updateInput := map[string]interface{}{
			"name":  "Jane Doe",
			"email": "jane@example.com",
		}
		inputBytes, err := json.Marshal(updateInput)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPut, "/users/1", bytes.NewReader(inputBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var updatedUser map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &updatedUser)
		require.NoError(t, err)
		assert.Equal(t, "Jane Doe", updatedUser["name"])
	})

	t.Run("delete user", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/users/1", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)

		// Verify user is deleted
		req = httptest.NewRequest(http.MethodGet, "/users/1", nil)
		w = httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("get non-existent user returns 404", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/users/999", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestCORSIntegration(t *testing.T) {
	_, handler, cleanup := setupTestServer(t)
	defer cleanup()

	t.Run("preflight request", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/users", nil)
		req.Header.Set("Origin", "http://example.com")
		req.Header.Set("Access-Control-Request-Method", "POST")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
		assert.Contains(t, w.Header().Get("Access-Control-Allow-Origin"), "http://example.com")
	})
}
