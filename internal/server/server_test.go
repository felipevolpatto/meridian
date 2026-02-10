package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/felipevolpatto/meridian/internal/config"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestSpec() *openapi3.T {
	paths := openapi3.NewPaths()

	// Helper to create schema with type
	stringSchema := func() *openapi3.Schema {
		s := &openapi3.Schema{}
		s.Type = "string"
		return s
	}

	objectSchema := func() *openapi3.Schema {
		s := &openapi3.Schema{}
		s.Type = "object"
		return s
	}

	arraySchema := func(items *openapi3.Schema) *openapi3.Schema {
		s := &openapi3.Schema{}
		s.Type = "array"
		s.Items = &openapi3.SchemaRef{Value: items}
		return s
	}

	userSchema := func() *openapi3.Schema {
		s := objectSchema()
		s.Properties = map[string]*openapi3.SchemaRef{
			"id":   {Value: stringSchema()},
			"name": {Value: stringSchema()},
		}
		return s
	}

	// Create response schemas
	responses200 := openapi3.NewResponses()
	responses200.Set("200", &openapi3.ResponseRef{
		Value: &openapi3.Response{
			Content: openapi3.Content{
				"application/json": &openapi3.MediaType{
					Schema: &openapi3.SchemaRef{Value: arraySchema(userSchema())},
				},
			},
		},
	})

	responses201 := openapi3.NewResponses()
	responses201.Set("201", &openapi3.ResponseRef{
		Value: &openapi3.Response{
			Content: openapi3.Content{
				"application/json": &openapi3.MediaType{
					Schema: &openapi3.SchemaRef{Value: userSchema()},
				},
			},
		},
	})

	responses404 := openapi3.NewResponses()
	responses404.Set("404", &openapi3.ResponseRef{
		Value: &openapi3.Response{
			Content: openapi3.Content{
				"application/json": &openapi3.MediaType{
					Schema: &openapi3.SchemaRef{Value: objectSchema()},
				},
			},
		},
	})

	// User input schema
	userInputSchema := func() *openapi3.Schema {
		s := objectSchema()
		s.Properties = map[string]*openapi3.SchemaRef{
			"name": {Value: stringSchema()},
		}
		s.Required = []string{"name"}
		return s
	}

	// /users endpoint
	paths.Set("/users", &openapi3.PathItem{
		Get: &openapi3.Operation{
			OperationID: "listUsers",
			Responses:   responses200,
		},
		Post: &openapi3.Operation{
			OperationID: "createUser",
			RequestBody: &openapi3.RequestBodyRef{
				Value: &openapi3.RequestBody{
					Content: openapi3.Content{
						"application/json": &openapi3.MediaType{
							Schema: &openapi3.SchemaRef{Value: userInputSchema()},
						},
					},
				},
			},
			Responses: responses201,
		},
	})

	// /users/{id} endpoint
	paths.Set("/users/{id}", &openapi3.PathItem{
		Get: &openapi3.Operation{
			OperationID: "getUser",
			Responses:   responses200,
		},
		Put: &openapi3.Operation{
			OperationID: "updateUser",
			RequestBody: &openapi3.RequestBodyRef{
				Value: &openapi3.RequestBody{
					Content: openapi3.Content{
						"application/json": &openapi3.MediaType{
							Schema: &openapi3.SchemaRef{Value: userInputSchema()},
						},
					},
				},
			},
			Responses: responses200,
		},
		Delete: &openapi3.Operation{
			OperationID: "deleteUser",
			Responses:   responses404,
		},
	})

	return &openapi3.T{
		Paths: paths,
	}
}

func createTestConfig(dbPath string) *config.Config {
	return &config.Config{
		OpenAPI: "test.yaml",
		Server: config.ServerConfig{
			Address: "localhost",
			Port:    8080,
		},
		State: config.StateConfig{
			Persistence: dbPath,
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
}

func TestNewServer(t *testing.T) {
	// Create temp db file
	tmpFile, err := os.CreateTemp("", "test-*.db")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	spec := createTestSpec()
	cfg := createTestConfig(tmpFile.Name())

	server := NewServer(spec, cfg)

	assert.NotNil(t, server)
	assert.Equal(t, spec, server.spec)
	assert.Equal(t, cfg, server.cfg)
	assert.NotNil(t, server.validator)
	assert.NotNil(t, server.stateManager)
}

func TestServerHandlers(t *testing.T) {
	// Create temp db file
	tmpFile, err := os.CreateTemp("", "test-*.db")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	testSpec := createTestSpec()
	cfg := createTestConfig(tmpFile.Name())
	server := NewServer(testSpec, cfg)
	require.NotNil(t, server)

	handler := server.createHandler()

	t.Run("GET /users - empty collection", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/users", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

		var response []interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.NotNil(t, response)
		assert.Len(t, response, 0)
	})

	t.Run("POST /users - create new user", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"id":   "1",
			"name": "Test User",
		}
		reqBytes, err := json.Marshal(reqBody)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewReader(reqBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "Test User", response["name"])
		assert.Equal(t, "1", response["id"])
	})

	t.Run("GET /users - after creation", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/users", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response []map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Len(t, response, 1)
		assert.Equal(t, "Test User", response[0]["name"])
	})

	t.Run("GET /users/{id} - get specific user", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/users/1", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "Test User", response["name"])
	})

	t.Run("GET /users/{id} - not found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/users/999", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("PUT /users/{id} - update user", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"name": "Updated User",
		}
		reqBytes, err := json.Marshal(reqBody)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPut, "/users/1", bytes.NewReader(reqBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "Updated User", response["name"])
	})

	t.Run("DELETE /users/{id} - delete user", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/users/1", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
	})

	t.Run("GET /users - after deletion", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/users", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		var response []interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Len(t, response, 0)
	})
}

func TestAdminEndpoints(t *testing.T) {
	// Create temp db file
	tmpFile, err := os.CreateTemp("", "test-*.db")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	testSpec := createTestSpec()
	cfg := createTestConfig(tmpFile.Name())
	server := NewServer(testSpec, cfg)
	handler := server.createHandler()

	t.Run("GET /_meridian/status", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/_meridian/status", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "online", response["status"])
	})

	t.Run("GET /_meridian/state", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/_meridian/state", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Contains(t, response, "version")
	})
}

func TestCORSMiddleware_Integration(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-*.db")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	testSpec := createTestSpec()
	cfg := createTestConfig(tmpFile.Name())
	cfg.Behavior.CORS.Enabled = true
	cfg.Behavior.CORS.AllowedOrigins = []string{"*"}

	server := NewServer(testSpec, cfg)
	handler := server.createHandler()

	t.Run("OPTIONS request returns CORS headers", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/users", nil)
		req.Header.Set("Origin", "http://example.com")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
		assert.NotEmpty(t, w.Header().Get("Access-Control-Allow-Origin"))
	})

	t.Run("GET request includes CORS headers", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/users", nil)
		req.Header.Set("Origin", "http://example.com")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		assert.NotEmpty(t, w.Header().Get("Access-Control-Allow-Origin"))
	})
}
