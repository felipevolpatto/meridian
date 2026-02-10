package server

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/felipevolpatto/meridian/internal/config"
	"github.com/felipevolpatto/meridian/internal/state"
	"github.com/felipevolpatto/meridian/internal/validation"
	"github.com/getkin/kin-openapi/openapi3"
)

//go:embed web_content/*
var webContent embed.FS

type Server struct {
	spec         *openapi3.T
	cfg          *config.Config
	validator    *validation.RequestValidator
	httpServer   *http.Server
	stateManager *state.Manager
	pathMatchers []pathMatcher
	handler      http.Handler
}

type pathMatcher struct {
	pattern    *regexp.Regexp
	template   string
	paramNames []string
	pathItem   *openapi3.PathItem
}

func NewServer(spec *openapi3.T, cfg *config.Config) *Server {
	manager, err := state.New(cfg.State.Persistence)
	if err != nil {
		log.Fatalf("Failed to create state manager: %v", err)
	}

	s := &Server{
		spec:         spec,
		cfg:          cfg,
		validator:    validation.NewRequestValidator(spec),
		stateManager: manager,
	}

	s.compilePaths()
	s.handler = s.createHandler()

	return s
}

func (s *Server) compilePaths() {
	s.pathMatchers = make([]pathMatcher, 0)

	for path, pathItem := range s.spec.Paths.Map() {
		paramNames := make([]string, 0)
		regexPattern := "^"

		parts := strings.Split(path, "/")
		for _, part := range parts {
			if part == "" {
				continue
			}
			regexPattern += "/"
			if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
				paramName := part[1 : len(part)-1]
				paramNames = append(paramNames, paramName)
				regexPattern += `([^/]+)`
			} else {
				regexPattern += regexp.QuoteMeta(part)
			}
		}
		regexPattern += "$"

		compiled, err := regexp.Compile(regexPattern)
		if err != nil {
			log.Printf("Warning: failed to compile path pattern %s: %v", path, err)
			continue
		}

		s.pathMatchers = append(s.pathMatchers, pathMatcher{
			pattern:    compiled,
			template:   path,
			paramNames: paramNames,
			pathItem:   pathItem,
		})
	}
}

func (s *Server) matchPath(requestPath string) (*openapi3.PathItem, map[string]string) {
	for _, matcher := range s.pathMatchers {
		matches := matcher.pattern.FindStringSubmatch(requestPath)
		if matches != nil {
			params := make(map[string]string)
			for i, name := range matcher.paramNames {
				if i+1 < len(matches) {
					params[name] = matches[i+1]
				}
			}
			return matcher.pathItem, params
		}
	}
	return nil, nil
}

func StartServer(spec *openapi3.T, cfg *config.Config) error {
	s := NewServer(spec, cfg)

	addr := fmt.Sprintf("%s:%d", cfg.Server.Address, cfg.Server.Port)
	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      s.createHandler(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("Starting Meridian mock server on http://%s", addr)
	log.Printf("Web interface available at http://%s/_meridian/", addr)

	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	if s.httpServer == nil {
		return nil
	}
	return s.httpServer.Shutdown(ctx)
}

func (s *Server) createHandler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/_meridian/status", s.handleStatus)
	mux.HandleFunc("/_meridian/state", s.handleStateAPI)
	mux.HandleFunc("/_meridian/spec", s.handleSpec)
	mux.HandleFunc("/_meridian/", s.handleWebUI)
	mux.HandleFunc("/", s.handleAPI)

	var handler http.Handler = mux

	if s.cfg.Behavior.Compression {
		handler = s.compressionMiddleware(handler)
	}

	if s.cfg.Behavior.Caching.Enabled {
		handler = s.cachingMiddleware(handler)
	}

	if s.cfg.Behavior.RateLimit.Enabled {
		handler = s.rateLimitMiddleware(handler)
	}

	if s.cfg.Behavior.Errors.Enabled {
		handler = s.errorSimulationMiddleware(handler)
	}

	if s.cfg.Behavior.Latency.Enabled {
		handler = s.latencyMiddleware(handler)
	}

	if s.cfg.Behavior.CORS.Enabled {
		handler = s.corsMiddleware(handler)
	}

	return handler
}

func (s *Server) handleWebUI(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/_meridian")
	if path == "" || path == "/" {
		path = "/index.html"
	}

	content, err := fs.Sub(webContent, "web_content")
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	filePath := strings.TrimPrefix(path, "/")
	file, err := content.Open(filePath)
	if err != nil {
		filePath = "index.html"
		file, err = content.Open(filePath)
		if err != nil {
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	contentType := "text/plain"
	switch {
	case strings.HasSuffix(filePath, ".html"):
		contentType = "text/html; charset=utf-8"
	case strings.HasSuffix(filePath, ".css"):
		contentType = "text/css; charset=utf-8"
	case strings.HasSuffix(filePath, ".js"):
		contentType = "application/javascript; charset=utf-8"
	case strings.HasSuffix(filePath, ".json"):
		contentType = "application/json; charset=utf-8"
	case strings.HasSuffix(filePath, ".svg"):
		contentType = "image/svg+xml"
	case strings.HasSuffix(filePath, ".png"):
		contentType = "image/png"
	case strings.HasSuffix(filePath, ".ico"):
		contentType = "image/x-icon"
	}
	w.Header().Set("Content-Type", contentType)

	data := make([]byte, stat.Size())
	if _, err := file.Read(data); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Write(data)
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	specInfo := map[string]interface{}{}
	if s.spec != nil && s.spec.Info != nil {
		specInfo["title"] = s.spec.Info.Title
		specInfo["version"] = s.spec.Info.Version
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "online",
		"version": "1.0.0",
		"spec":    specInfo,
	})
}

func (s *Server) handleStateAPI(w http.ResponseWriter, r *http.Request) {
	data, err := s.stateManager.Export()
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to export state: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func (s *Server) handleSpec(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.spec)
}

func (s *Server) handleAPI(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	method := r.Method

	pathItem, pathParams := s.matchPath(path)
	if pathItem == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "Path not found",
			"code":  "not_found",
			"path":  path,
		})
		return
	}

	var op *openapi3.Operation
	switch method {
	case http.MethodGet:
		op = pathItem.Get
	case http.MethodPost:
		op = pathItem.Post
	case http.MethodPut:
		op = pathItem.Put
	case http.MethodDelete:
		op = pathItem.Delete
	case http.MethodPatch:
		op = pathItem.Patch
	case http.MethodHead:
		op = pathItem.Head
	case http.MethodOptions:
		op = pathItem.Options
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if op == nil {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	resourceName, nestedInfo := ExtractResourceInfo(path, pathParams)
	if resourceName == "" {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	switch method {
	case http.MethodGet:
		s.handleGet(w, r, resourceName, pathParams, nestedInfo)
	case http.MethodPost:
		s.handlePost(w, r, resourceName, nestedInfo)
	case http.MethodPut:
		s.handlePut(w, r, resourceName, pathParams, nestedInfo)
	case http.MethodDelete:
		s.handleDelete(w, r, resourceName, pathParams, nestedInfo)
	case http.MethodPatch:
		s.handlePatch(w, r, resourceName, pathParams, nestedInfo)
	}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.handler.ServeHTTP(w, r)
}

func (s *Server) handleGet(w http.ResponseWriter, r *http.Request, resourceName string, pathParams map[string]string, nestedInfo *NestedResourceInfo) {
	// For nested resources, use the child ID if present
	var resourceID string
	if nestedInfo.IsNested && nestedInfo.ChildID != "" {
		resourceID = nestedInfo.ChildID
	} else {
		resourceID = pathParams["id"]
		if resourceID == "" {
			for key, value := range pathParams {
				if strings.HasSuffix(strings.ToLower(key), "id") && key != nestedInfo.ParentIDParam {
					resourceID = value
					break
				}
			}
		}
	}

	if resourceID == "" {
		data, err := s.stateManager.GetResources(resourceName)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to get resources: %v", err), http.StatusInternalServerError)
			return
		}

		if data == nil {
			data = []interface{}{}
		}

		// Filter by parent ID for nested resources
		if nestedInfo.IsNested && nestedInfo.ParentID != "" {
			data = s.filterByParentID(data, nestedInfo)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(data)
		return
	}

	data, err := s.stateManager.GetResource(resourceName, resourceID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "Resource not found",
			"code":  "not_found",
		})
		return
	}

	// Verify the resource belongs to the parent for nested resources
	if nestedInfo.IsNested && nestedInfo.ParentID != "" {
		if !s.belongsToParent(data, nestedInfo) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": "Resource not found",
				"code":  "not_found",
			})
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

// filterByParentID filters a list of resources by parent ID
func (s *Server) filterByParentID(data []interface{}, nestedInfo *NestedResourceInfo) []interface{} {
	if nestedInfo == nil || !nestedInfo.IsNested || nestedInfo.ParentID == "" {
		return data
	}

	filtered := make([]interface{}, 0)
	for _, item := range data {
		if s.belongsToParent(item, nestedInfo) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

// belongsToParent checks if a resource belongs to the specified parent
func (s *Server) belongsToParent(item interface{}, nestedInfo *NestedResourceInfo) bool {
	if nestedInfo == nil || !nestedInfo.IsNested || nestedInfo.ParentID == "" {
		return true
	}

	obj, ok := item.(map[string]interface{})
	if !ok {
		return false
	}

	// Check common foreign key patterns
	foreignKeyFields := []string{
		nestedInfo.ForeignKeyField,
		nestedInfo.ParentResource + "_id",
		nestedInfo.ParentResource + "Id",
		nestedInfo.ParentIDParam,
	}

	for _, field := range foreignKeyFields {
		if val, exists := obj[field]; exists {
			if fmt.Sprintf("%v", val) == nestedInfo.ParentID {
				return true
			}
		}
	}

	return false
}

func (s *Server) handlePost(w http.ResponseWriter, r *http.Request, resourceName string, nestedInfo *NestedResourceInfo) {
	var data map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "Failed to parse request body",
			"code":  "invalid_json",
		})
		return
	}

	if _, ok := data["id"]; !ok {
		data["id"] = fmt.Sprintf("%d", time.Now().UnixNano())
	}

	// Automatically set foreign key for nested resources
	if nestedInfo.IsNested && nestedInfo.ParentID != "" {
		data[nestedInfo.ForeignKeyField] = nestedInfo.ParentID
	}

	if err := s.stateManager.AddResource(resourceName, data); err != nil {
		http.Error(w, fmt.Sprintf("failed to add resource: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(data)
}

func (s *Server) handlePut(w http.ResponseWriter, r *http.Request, resourceName string, pathParams map[string]string, nestedInfo *NestedResourceInfo) {
	var resourceID string
	if nestedInfo.IsNested && nestedInfo.ChildID != "" {
		resourceID = nestedInfo.ChildID
	} else {
		resourceID = pathParams["id"]
		if resourceID == "" {
			for key, value := range pathParams {
				if strings.HasSuffix(strings.ToLower(key), "id") && key != nestedInfo.ParentIDParam {
					resourceID = value
					break
				}
			}
		}
	}

	if resourceID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "Resource ID required",
			"code":  "missing_id",
		})
		return
	}

	// Verify the resource belongs to the parent for nested resources
	if nestedInfo.IsNested && nestedInfo.ParentID != "" {
		existing, err := s.stateManager.GetResource(resourceName, resourceID)
		if err == nil && !s.belongsToParent(existing, nestedInfo) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": "Resource not found",
				"code":  "not_found",
			})
			return
		}
	}

	var data map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "Failed to parse request body",
			"code":  "invalid_json",
		})
		return
	}

	data["id"] = resourceID

	// Preserve foreign key for nested resources
	if nestedInfo.IsNested && nestedInfo.ParentID != "" {
		data[nestedInfo.ForeignKeyField] = nestedInfo.ParentID
	}

	if err := s.stateManager.UpdateResource(resourceName, resourceID, data); err != nil {
		if err.Error() == "resource not found" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": "Resource not found",
				"code":  "not_found",
			})
			return
		}
		http.Error(w, fmt.Sprintf("failed to update resource: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func (s *Server) handlePatch(w http.ResponseWriter, r *http.Request, resourceName string, pathParams map[string]string, nestedInfo *NestedResourceInfo) {
	var resourceID string
	if nestedInfo.IsNested && nestedInfo.ChildID != "" {
		resourceID = nestedInfo.ChildID
	} else {
		resourceID = pathParams["id"]
		if resourceID == "" {
			for key, value := range pathParams {
				if strings.HasSuffix(strings.ToLower(key), "id") && key != nestedInfo.ParentIDParam {
					resourceID = value
					break
				}
			}
		}
	}

	if resourceID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "Resource ID required",
			"code":  "missing_id",
		})
		return
	}

	existing, err := s.stateManager.GetResource(resourceName, resourceID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "Resource not found",
			"code":  "not_found",
		})
		return
	}

	// Verify the resource belongs to the parent for nested resources
	if nestedInfo.IsNested && nestedInfo.ParentID != "" {
		if !s.belongsToParent(existing, nestedInfo) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": "Resource not found",
				"code":  "not_found",
			})
			return
		}
	}

	var patchData map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&patchData); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "Failed to parse request body",
			"code":  "invalid_json",
		})
		return
	}

	existingMap, ok := existing.(map[string]interface{})
	if !ok {
		http.Error(w, "invalid resource format", http.StatusInternalServerError)
		return
	}

	for key, value := range patchData {
		existingMap[key] = value
	}

	// Preserve foreign key for nested resources
	if nestedInfo.IsNested && nestedInfo.ParentID != "" {
		existingMap[nestedInfo.ForeignKeyField] = nestedInfo.ParentID
	}

	if err := s.stateManager.UpdateResource(resourceName, resourceID, existingMap); err != nil {
		http.Error(w, fmt.Sprintf("failed to update resource: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(existingMap)
}

func (s *Server) handleDelete(w http.ResponseWriter, r *http.Request, resourceName string, pathParams map[string]string, nestedInfo *NestedResourceInfo) {
	var resourceID string
	if nestedInfo.IsNested && nestedInfo.ChildID != "" {
		resourceID = nestedInfo.ChildID
	} else {
		resourceID = pathParams["id"]
		if resourceID == "" {
			for key, value := range pathParams {
				if strings.HasSuffix(strings.ToLower(key), "id") && key != nestedInfo.ParentIDParam {
					resourceID = value
					break
				}
			}
		}
	}

	if resourceID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "Resource ID required",
			"code":  "missing_id",
		})
		return
	}

	// Verify the resource belongs to the parent for nested resources
	if nestedInfo.IsNested && nestedInfo.ParentID != "" {
		existing, err := s.stateManager.GetResource(resourceName, resourceID)
		if err == nil && !s.belongsToParent(existing, nestedInfo) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": "Resource not found",
				"code":  "not_found",
			})
			return
		}
	}

	if err := s.stateManager.DeleteResource(resourceName, resourceID); err != nil {
		if err.Error() == "resource not found" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": "Resource not found",
				"code":  "not_found",
			})
			return
		}
		http.Error(w, fmt.Sprintf("failed to delete resource: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
