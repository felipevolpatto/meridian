package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/felipevolpatto/meridian/internal/config"
	"github.com/getkin/kin-openapi/openapi3"
)

func TestParseNestedResource(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		pathParams map[string]string
		wantNested bool
		wantParent string
		wantChild  string
		wantPID    string
		wantCID    string
	}{
		{
			name:       "simple resource",
			path:       "/users",
			pathParams: map[string]string{},
			wantNested: false,
		},
		{
			name:       "simple resource with id",
			path:       "/users/123",
			pathParams: map[string]string{"id": "123"},
			wantNested: false,
		},
		{
			name:       "nested resource list",
			path:       "/users/123/posts",
			pathParams: map[string]string{"userId": "123"},
			wantNested: true,
			wantParent: "users",
			wantChild:  "posts",
			wantPID:    "123",
			wantCID:    "",
		},
		{
			name:       "nested resource with child id",
			path:       "/users/123/posts/456",
			pathParams: map[string]string{"userId": "123", "postId": "456"},
			wantNested: true,
			wantParent: "users",
			wantChild:  "posts",
			wantPID:    "123",
			wantCID:    "456",
		},
		{
			name:       "deeply nested resource",
			path:       "/organizations/org1/teams/team1/members",
			pathParams: map[string]string{"orgId": "org1", "teamId": "team1"},
			wantNested: true,
			wantParent: "organizations",
			wantChild:  "teams",
			wantPID:    "org1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := ParseNestedResource(tt.path, tt.pathParams)

			if info.IsNested != tt.wantNested {
				t.Errorf("IsNested = %v, want %v", info.IsNested, tt.wantNested)
			}

			if tt.wantNested {
				if info.ParentResource != tt.wantParent {
					t.Errorf("ParentResource = %v, want %v", info.ParentResource, tt.wantParent)
				}
				if info.ChildResource != tt.wantChild {
					t.Errorf("ChildResource = %v, want %v", info.ChildResource, tt.wantChild)
				}
				if info.ParentID != tt.wantPID {
					t.Errorf("ParentID = %v, want %v", info.ParentID, tt.wantPID)
				}
				if tt.wantCID != "" && info.ChildID != tt.wantCID {
					t.Errorf("ChildID = %v, want %v", info.ChildID, tt.wantCID)
				}
			}
		})
	}
}

func TestInferForeignKeyField(t *testing.T) {
	tests := []struct {
		parentResource string
		parentIDParam  string
		expected       string
	}{
		{"users", "userId", "user_id"},
		{"posts", "postId", "post_id"},
		{"categories", "categoryId", "category_id"},
		{"companies", "companyId", "company_id"},
	}

	for _, tt := range tests {
		t.Run(tt.parentResource, func(t *testing.T) {
			result := inferForeignKeyField(tt.parentResource, tt.parentIDParam)
			if result != tt.expected {
				t.Errorf("inferForeignKeyField(%q, %q) = %q, want %q",
					tt.parentResource, tt.parentIDParam, result, tt.expected)
			}
		})
	}
}

func TestIsResourceName(t *testing.T) {
	tests := []struct {
		part     string
		expected bool
	}{
		{"users", true},
		{"posts", true},
		{"my-resource", true},
		{"my_resource", true},
		{"{userId}", false},
		{"123", false},
		{"abc123", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.part, func(t *testing.T) {
			result := isResourceName(tt.part)
			if result != tt.expected {
				t.Errorf("isResourceName(%q) = %v, want %v", tt.part, result, tt.expected)
			}
		})
	}
}

func TestExtractResourceInfo(t *testing.T) {
	tests := []struct {
		name         string
		path         string
		pathParams   map[string]string
		wantResource string
		wantNested   bool
	}{
		{
			name:         "simple resource",
			path:         "/users",
			pathParams:   map[string]string{},
			wantResource: "users",
			wantNested:   false,
		},
		{
			name:         "nested resource",
			path:         "/users/123/posts",
			pathParams:   map[string]string{"userId": "123"},
			wantResource: "posts",
			wantNested:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resourceName, nestedInfo := ExtractResourceInfo(tt.path, tt.pathParams)

			if resourceName != tt.wantResource {
				t.Errorf("resourceName = %q, want %q", resourceName, tt.wantResource)
			}
			if nestedInfo.IsNested != tt.wantNested {
				t.Errorf("IsNested = %v, want %v", nestedInfo.IsNested, tt.wantNested)
			}
		})
	}
}

func TestNestedResourcesIntegration(t *testing.T) {
	spec := &openapi3.T{
		OpenAPI: "3.0.0",
		Info: &openapi3.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Paths: &openapi3.Paths{},
	}

	// Add paths for nested resources
	spec.Paths.Set("/users", &openapi3.PathItem{
		Get:  &openapi3.Operation{OperationID: "listUsers"},
		Post: &openapi3.Operation{OperationID: "createUser"},
	})
	spec.Paths.Set("/users/{userId}", &openapi3.PathItem{
		Get:    &openapi3.Operation{OperationID: "getUser"},
		Put:    &openapi3.Operation{OperationID: "updateUser"},
		Delete: &openapi3.Operation{OperationID: "deleteUser"},
	})
	spec.Paths.Set("/users/{userId}/posts", &openapi3.PathItem{
		Get:  &openapi3.Operation{OperationID: "listUserPosts"},
		Post: &openapi3.Operation{OperationID: "createUserPost"},
	})
	spec.Paths.Set("/users/{userId}/posts/{postId}", &openapi3.PathItem{
		Get:    &openapi3.Operation{OperationID: "getUserPost"},
		Put:    &openapi3.Operation{OperationID: "updateUserPost"},
		Delete: &openapi3.Operation{OperationID: "deleteUserPost"},
	})

	cfg := &config.Config{
		Server: config.ServerConfig{
			Address: "localhost",
			Port:    8080,
		},
		State: config.StateConfig{
			Persistence: "",
		},
		Behavior: config.BehaviorConfig{
			CORS: config.CORSConfig{Enabled: false},
		},
	}

	server := NewServer(spec, cfg)

	// Create a user first
	userBody := `{"id": "user-1", "name": "John Doe"}`
	req := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(userBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("Failed to create user: %d - %s", rr.Code, rr.Body.String())
	}

	// Create a post for the user via nested route
	postBody := `{"id": "post-1", "title": "My First Post", "content": "Hello World"}`
	req = httptest.NewRequest(http.MethodPost, "/users/user-1/posts", strings.NewReader(postBody))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("Failed to create post: %d - %s", rr.Code, rr.Body.String())
	}

	// Verify the post has user_id set automatically
	var createdPost map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &createdPost)
	if createdPost["user_id"] != "user-1" {
		t.Errorf("Expected user_id = 'user-1', got %v", createdPost["user_id"])
	}

	// List posts for user-1
	req = httptest.NewRequest(http.MethodGet, "/users/user-1/posts", nil)
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Failed to list posts: %d - %s", rr.Code, rr.Body.String())
	}

	var posts []map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &posts)
	if len(posts) != 1 {
		t.Errorf("Expected 1 post, got %d", len(posts))
	}

	// Get specific post
	req = httptest.NewRequest(http.MethodGet, "/users/user-1/posts/post-1", nil)
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Failed to get post: %d - %s", rr.Code, rr.Body.String())
	}

	// Create another user
	user2Body := `{"id": "user-2", "name": "Jane Doe"}`
	req = httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(user2Body))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	// Create a post for user-2
	post2Body := `{"id": "post-2", "title": "Jane's Post"}`
	req = httptest.NewRequest(http.MethodPost, "/users/user-2/posts", strings.NewReader(post2Body))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	// List posts for user-1 should still return only 1
	req = httptest.NewRequest(http.MethodGet, "/users/user-1/posts", nil)
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	json.Unmarshal(rr.Body.Bytes(), &posts)
	if len(posts) != 1 {
		t.Errorf("Expected 1 post for user-1, got %d", len(posts))
	}

	// Try to get user-2's post via user-1's route - should fail
	req = httptest.NewRequest(http.MethodGet, "/users/user-1/posts/post-2", nil)
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("Expected 404 for cross-user post access, got %d", rr.Code)
	}

	// Update post via nested route
	updateBody := `{"title": "Updated Title"}`
	req = httptest.NewRequest(http.MethodPut, "/users/user-1/posts/post-1", strings.NewReader(updateBody))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Failed to update post: %d - %s", rr.Code, rr.Body.String())
	}

	// Verify foreign key is preserved after update
	var updatedPost map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &updatedPost)
	if updatedPost["user_id"] != "user-1" {
		t.Errorf("Expected user_id preserved after update, got %v", updatedPost["user_id"])
	}

	// Delete post via nested route
	req = httptest.NewRequest(http.MethodDelete, "/users/user-1/posts/post-1", nil)
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Errorf("Failed to delete post: %d - %s", rr.Code, rr.Body.String())
	}

	// Verify post is deleted
	req = httptest.NewRequest(http.MethodGet, "/users/user-1/posts/post-1", nil)
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("Expected 404 after delete, got %d", rr.Code)
	}
}

func TestNestedResourcesPatch(t *testing.T) {
	spec := &openapi3.T{
		OpenAPI: "3.0.0",
		Info: &openapi3.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Paths: &openapi3.Paths{},
	}

	spec.Paths.Set("/users", &openapi3.PathItem{
		Post: &openapi3.Operation{OperationID: "createUser"},
	})
	spec.Paths.Set("/users/{userId}/posts", &openapi3.PathItem{
		Post: &openapi3.Operation{OperationID: "createUserPost"},
	})
	spec.Paths.Set("/users/{userId}/posts/{postId}", &openapi3.PathItem{
		Patch: &openapi3.Operation{OperationID: "patchUserPost"},
	})

	cfg := &config.Config{
		Server: config.ServerConfig{Address: "localhost", Port: 8080},
		State:  config.StateConfig{Persistence: ""},
	}

	server := NewServer(spec, cfg)

	// Create user and post
	req := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(`{"id": "u1"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	req = httptest.NewRequest(http.MethodPost, "/users/u1/posts", strings.NewReader(`{"id": "p1", "title": "Original"}`))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	// Patch the post
	req = httptest.NewRequest(http.MethodPatch, "/users/u1/posts/p1", strings.NewReader(`{"title": "Patched"}`))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("PATCH failed: %d - %s", rr.Code, rr.Body.String())
	}

	var patched map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &patched)

	if patched["title"] != "Patched" {
		t.Errorf("Expected title = 'Patched', got %v", patched["title"])
	}
	if patched["user_id"] != "u1" {
		t.Errorf("Expected user_id preserved, got %v", patched["user_id"])
	}
}

func TestBuildAndParseNestedResourceKey(t *testing.T) {
	key := BuildNestedResourceKey("users", "123", "posts")
	expected := "users:123:posts"
	if key != expected {
		t.Errorf("BuildNestedResourceKey = %q, want %q", key, expected)
	}

	parent, parentID, child, ok := ParseNestedResourceKey(key)
	if !ok {
		t.Fatal("ParseNestedResourceKey returned false")
	}
	if parent != "users" || parentID != "123" || child != "posts" {
		t.Errorf("ParseNestedResourceKey = (%q, %q, %q), want (users, 123, posts)", parent, parentID, child)
	}

	// Invalid key
	_, _, _, ok = ParseNestedResourceKey("invalid")
	if ok {
		t.Error("Expected ParseNestedResourceKey to return false for invalid key")
	}
}
