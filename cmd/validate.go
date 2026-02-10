package cmd

import (
	"encoding/json"
	"fmt"
	"math"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate requests and responses",
	Long:  `Validate HTTP requests and responses against the OpenAPI specification.`,
	RunE:  runValidate,
}

func init() {
	rootCmd.AddCommand(validateCmd)
	validateCmd.Flags().StringP("spec", "s", "openapi.yaml", "Path to OpenAPI specification file")
	validateCmd.Flags().StringP("request", "r", "", "Path to request file (JSON)")
	validateCmd.Flags().StringP("response", "p", "", "Path to response file (JSON)")
	validateCmd.Flags().BoolP("verbose", "v", false, "Show detailed validation results")
}

type RequestData struct {
	Method  string            `json:"method"`
	Path    string            `json:"path"`
	Headers map[string]string `json:"headers"`
	Query   string            `json:"query"`
	Body    json.RawMessage   `json:"body"`
}

type ResponseData struct {
	StatusCode int               `json:"status_code"`
	Headers    map[string]string `json:"headers"`
	Body       json.RawMessage   `json:"body"`
}

func runValidate(cmd *cobra.Command, args []string) error {
	specPath, _ := cmd.Flags().GetString("spec")
	requestPath, _ := cmd.Flags().GetString("request")
	responsePath, _ := cmd.Flags().GetString("response")
	verbose, _ := cmd.Flags().GetBool("verbose")

	loader := openapi3.NewLoader()
	spec, err := loader.LoadFromFile(specPath)
	if err != nil {
		return fmt.Errorf("failed to load OpenAPI spec: %w", err)
	}

	if requestPath != "" {
		if err := validateRequest(spec, requestPath, verbose); err != nil {
			return fmt.Errorf("request validation failed: %w", err)
		}
	}

	if responsePath != "" {
		if err := validateResponse(spec, responsePath, verbose); err != nil {
			return fmt.Errorf("response validation failed: %w", err)
		}
	}

	if requestPath == "" && responsePath == "" {
		return fmt.Errorf("either --request or --response must be provided")
	}

	return nil
}

func validateRequest(spec *openapi3.T, requestPath string, verbose bool) error {
	data, err := os.ReadFile(requestPath)
	if err != nil {
		return fmt.Errorf("failed to read request file: %w", err)
	}

	var request RequestData
	if err := json.Unmarshal(data, &request); err != nil {
		return fmt.Errorf("failed to parse request data: %w", err)
	}

	pathItem := spec.Paths.Find(request.Path)
	if pathItem == nil {
		return fmt.Errorf("path not found in API specification: %s", request.Path)
	}

	var operation *openapi3.Operation
	switch strings.ToUpper(request.Method) {
	case "GET":
		operation = pathItem.Get
	case "POST":
		operation = pathItem.Post
	case "PUT":
		operation = pathItem.Put
	case "DELETE":
		operation = pathItem.Delete
	case "PATCH":
		operation = pathItem.Patch
	case "HEAD":
		operation = pathItem.Head
	case "OPTIONS":
		operation = pathItem.Options
	default:
		return fmt.Errorf("unsupported HTTP method: %s", request.Method)
	}

	if operation == nil {
		return fmt.Errorf("method %s not allowed for path %s", request.Method, request.Path)
	}

	headers := make(map[string][]string)
	for k, v := range request.Headers {
		headers[k] = []string{v}
	}

	query, err := url.ParseQuery(request.Query)
	if err != nil {
		return fmt.Errorf("failed to parse query string: %w", err)
	}

	if err := validateRequestData(operation, headers, query, request.Body); err != nil {
		if verbose {
			fmt.Fprintln(os.Stderr, "Request validation errors:")
			for _, e := range strings.Split(err.Error(), "; ") {
				fmt.Fprintf(os.Stderr, "  - %s\n", e)
			}
		}
		return fmt.Errorf("invalid request: %w", err)
	}

	fmt.Println("✅ Request is valid")
	return nil
}

func validateResponse(spec *openapi3.T, responsePath string, verbose bool) error {
	data, err := os.ReadFile(responsePath)
	if err != nil {
		return fmt.Errorf("failed to read response file: %w", err)
	}

	var response ResponseData
	if err := json.Unmarshal(data, &response); err != nil {
		return fmt.Errorf("failed to parse response data: %w", err)
	}

	headers := make(http.Header)
	for k, v := range response.Headers {
		headers.Set(k, v)
	}

	if err := validateResponseData(spec, response.StatusCode, headers, response.Body); err != nil {
		if verbose {
			fmt.Fprintln(os.Stderr, "Response validation errors:")
			for _, e := range strings.Split(err.Error(), "; ") {
				fmt.Fprintf(os.Stderr, "  - %s\n", e)
			}
		}
		return fmt.Errorf("invalid response: %w", err)
	}

	fmt.Println("✅ Response is valid")
	return nil
}

func validateRequestData(operation *openapi3.Operation, headers map[string][]string, query url.Values, body json.RawMessage) error {
	for _, param := range operation.Parameters {
		if param.Value.In == "header" {
			if param.Value.Required {
				if _, ok := headers[param.Value.Name]; !ok {
					return fmt.Errorf("missing required header: %s", param.Value.Name)
				}
			}
			if values, ok := headers[param.Value.Name]; ok && len(values) > 0 && param.Value.Schema != nil {
				if err := validateParamValue(param.Value.Schema.Value, values[0], param.Value.Name); err != nil {
					return err
				}
			}
		} else if param.Value.In == "query" {
			if param.Value.Required {
				if _, ok := query[param.Value.Name]; !ok {
					return fmt.Errorf("missing required query parameter: %s", param.Value.Name)
				}
			}
			if values, ok := query[param.Value.Name]; ok && len(values) > 0 && param.Value.Schema != nil {
				if err := validateParamValue(param.Value.Schema.Value, values[0], param.Value.Name); err != nil {
					return err
				}
			}
		}
	}

	if operation.RequestBody != nil && operation.RequestBody.Value.Required {
		if len(body) == 0 {
			return fmt.Errorf("request body is required")
		}

		contentType := "application/json"
		if values, ok := headers["Content-Type"]; ok && len(values) > 0 {
			contentType = values[0]
		}

		mediaType := operation.RequestBody.Value.Content.Get(contentType)
		if mediaType == nil {
			return fmt.Errorf("unsupported content type: %s", contentType)
		}

		if mediaType.Schema != nil {
			var data interface{}
			if err := json.Unmarshal(body, &data); err != nil {
				return fmt.Errorf("invalid JSON in request body: %w", err)
			}

			if err := validateSchema(mediaType.Schema.Value, data); err != nil {
				return fmt.Errorf("request body validation failed: %w", err)
			}
		}
	}

	return nil
}

func validateResponseData(spec *openapi3.T, statusCode int, headers http.Header, body json.RawMessage) error {
	for _, pathItem := range spec.Paths.Map() {
		for _, op := range pathItem.Operations() {
			response := op.Responses.Status(statusCode)
			if response == nil {
				response = op.Responses.Default()
			}
			if response == nil {
				continue
			}

			if err := validateResponseHeaders(response.Value, headers); err != nil {
				return fmt.Errorf("invalid response headers: %w", err)
			}

			if err := validateResponseBody(response.Value, headers.Get("Content-Type"), body); err != nil {
				return fmt.Errorf("invalid response body: %w", err)
			}

			return nil
		}
	}

	return fmt.Errorf("no matching response found for status code %d", statusCode)
}

func validateResponseHeaders(response *openapi3.Response, headers http.Header) error {
	if response.Headers == nil {
		return nil
	}

	var errors []string

	for name, header := range response.Headers {
		if header.Value.Required {
			if _, exists := headers[name]; !exists {
				errors = append(errors, fmt.Sprintf("missing required header: %s", name))
				continue
			}
		}

		if values, exists := headers[name]; exists && header.Value.Schema != nil {
			for _, value := range values {
				if err := validateParamValue(header.Value.Schema.Value, value, name); err != nil {
					errors = append(errors, fmt.Sprintf("header %s: %v", name, err))
				}
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf(strings.Join(errors, "; "))
	}

	return nil
}

func validateResponseBody(response *openapi3.Response, contentType string, body json.RawMessage) error {
	if len(body) == 0 {
		return nil
	}

	if response.Content == nil {
		return fmt.Errorf("response body not allowed")
	}

	var mediaType *openapi3.MediaType
	for ct, mt := range response.Content {
		if matchContentType(ct, contentType) {
			mediaType = mt
			break
		}
	}

	if mediaType == nil {
		return fmt.Errorf("unsupported content type: %s", contentType)
	}

	if mediaType.Schema != nil {
		var data interface{}
		if err := json.Unmarshal(body, &data); err != nil {
			return fmt.Errorf("invalid JSON format: %w", err)
		}

		if err := validateSchema(mediaType.Schema.Value, data); err != nil {
			return fmt.Errorf("response body validation failed: %w", err)
		}
	}

	return nil
}

func validateSchema(schema *openapi3.Schema, data interface{}) error {
	if schema == nil {
		return nil
	}

	var errors []string

	if err := validateType(schema, data); err != nil {
		errors = append(errors, err.Error())
		return fmt.Errorf(strings.Join(errors, "; "))
	}

	switch schema.Type {
	case "object":
		if obj, ok := data.(map[string]interface{}); ok {
			for _, required := range schema.Required {
				if _, exists := obj[required]; !exists {
					errors = append(errors, fmt.Sprintf("missing required property: %s", required))
				}
			}

			for name, value := range obj {
				if prop, ok := schema.Properties[name]; ok {
					if err := validateSchema(prop.Value, value); err != nil {
						errors = append(errors, fmt.Sprintf("property %s: %v", name, err))
					}
				} else if schema.AdditionalProperties.Schema == nil {
					errors = append(errors, fmt.Sprintf("additional property not allowed: %s", name))
				}
			}

			if schema.MinProps > 0 && len(obj) < int(schema.MinProps) {
				errors = append(errors, fmt.Sprintf("too few properties, minimum %d", schema.MinProps))
			}
			if schema.MaxProps != nil && len(obj) > int(*schema.MaxProps) {
				errors = append(errors, fmt.Sprintf("too many properties, maximum %d", *schema.MaxProps))
			}
		}

	case "array":
		if arr, ok := data.([]interface{}); ok {
			if schema.MinItems > 0 && len(arr) < int(schema.MinItems) {
				errors = append(errors, fmt.Sprintf("too few items, minimum %d", schema.MinItems))
			}
			if schema.MaxItems != nil && len(arr) > int(*schema.MaxItems) {
				errors = append(errors, fmt.Sprintf("too many items, maximum %d", *schema.MaxItems))
			}

			if schema.Items != nil {
				for i, item := range arr {
					if err := validateSchema(schema.Items.Value, item); err != nil {
						errors = append(errors, fmt.Sprintf("item %d: %v", i, err))
					}
				}
			}

			if schema.UniqueItems {
				seen := make(map[string]bool)
				for _, item := range arr {
					key, err := json.Marshal(item)
					if err != nil {
						continue
					}
					if seen[string(key)] {
						errors = append(errors, "duplicate items not allowed")
						break
					}
					seen[string(key)] = true
				}
			}
		}

	case "string":
		if str, ok := data.(string); ok {
			if schema.MinLength > 0 && len(str) < int(schema.MinLength) {
				errors = append(errors, fmt.Sprintf("string too short, minimum %d", schema.MinLength))
			}

			if schema.MaxLength != nil && len(str) > int(*schema.MaxLength) {
				errors = append(errors, fmt.Sprintf("string too long, maximum %d", *schema.MaxLength))
			}

			if schema.Pattern != "" {
				if matched, err := regexp.MatchString(schema.Pattern, str); err == nil && !matched {
					errors = append(errors, fmt.Sprintf("string does not match pattern: %s", schema.Pattern))
				}
			}

			if err := validateFormat(schema.Format, str); err != nil {
				errors = append(errors, err.Error())
			}

			if schema.Enum != nil {
				valid := false
				for _, enum := range schema.Enum {
					if str == enum {
						valid = true
						break
					}
				}
				if !valid {
					errors = append(errors, fmt.Sprintf("value not in enum: %v", schema.Enum))
				}
			}
		}

	case "number", "integer":
		if num, ok := getNumber(data); ok {
			if schema.Min != nil && num < *schema.Min {
				errors = append(errors, fmt.Sprintf("value %v less than minimum %v", num, *schema.Min))
			}
			if schema.Max != nil && num > *schema.Max {
				errors = append(errors, fmt.Sprintf("value %v greater than maximum %v", num, *schema.Max))
			}

			if schema.MultipleOf != nil && math.Mod(num, *schema.MultipleOf) != 0 {
				errors = append(errors, fmt.Sprintf("value %v not multiple of %v", num, *schema.MultipleOf))
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf(strings.Join(errors, "; "))
	}

	return nil
}

func validateType(schema *openapi3.Schema, data interface{}) error {
	if schema.Type == "" {
		return nil
	}

	switch schema.Type {
	case "object":
		if _, ok := data.(map[string]interface{}); !ok {
			return fmt.Errorf("expected object, got %T", data)
		}
	case "array":
		if _, ok := data.([]interface{}); !ok {
			return fmt.Errorf("expected array, got %T", data)
		}
	case "string":
		if _, ok := data.(string); !ok {
			return fmt.Errorf("expected string, got %T", data)
		}
	case "number":
		if _, ok := getNumber(data); !ok {
			return fmt.Errorf("expected number, got %T", data)
		}
	case "integer":
		if num, ok := getNumber(data); !ok || math.Floor(num) != num {
			return fmt.Errorf("expected integer, got %T", data)
		}
	case "boolean":
		if _, ok := data.(bool); !ok {
			return fmt.Errorf("expected boolean, got %T", data)
		}
	case "null":
		if data != nil {
			return fmt.Errorf("expected null, got %T", data)
		}
	}

	return nil
}

func validateFormat(format, value string) error {
	switch format {
	case "date-time":
		if _, err := time.Parse(time.RFC3339, value); err != nil {
			return fmt.Errorf("invalid date-time format")
		}
	case "date":
		if _, err := time.Parse("2006-01-02", value); err != nil {
			return fmt.Errorf("invalid date format")
		}
	case "time":
		formats := []string{
			"15:04:05",
			"15:04:05.0",
			"15:04:05.00",
			"15:04:05.000",
			"15:04:05.0000",
			"15:04:05.00000",
			"15:04:05.000000",
		}
		valid := false
		for _, f := range formats {
			if _, err := time.Parse(f, value); err == nil {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("invalid time format, expected HH:MM:SS[.fff]")
		}
	case "email":
		if !strings.Contains(value, "@") {
			return fmt.Errorf("invalid email format")
		}
	case "ipv4":
		if ip := net.ParseIP(value); ip == nil || ip.To4() == nil {
			return fmt.Errorf("invalid IPv4 format")
		}
	case "ipv6":
		if ip := net.ParseIP(value); ip == nil || ip.To4() != nil {
			return fmt.Errorf("invalid IPv6 format")
		}
	case "uuid":
		if _, err := uuid.Parse(value); err != nil {
			return fmt.Errorf("invalid UUID format")
		}
	case "uri":
		if _, err := url.Parse(value); err != nil {
			return fmt.Errorf("invalid URI format")
		}
	}
	return nil
}

func getNumber(data interface{}) (float64, bool) {
	switch v := data.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	}
	return 0, false
}

func matchContentType(pattern, actual string) bool {
	pattern = strings.ToLower(strings.TrimSpace(pattern))
	actual = strings.ToLower(strings.TrimSpace(actual))

	patternParts := strings.Split(pattern, ";")
	actualParts := strings.Split(actual, ";")

	return patternParts[0] == actualParts[0] || patternParts[0] == "*/*"
}

func validateParamValue(schema *openapi3.Schema, value string, paramName string) error {
	if schema == nil {
		return nil
	}

	switch schema.Type {
	case "integer":
		intVal, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return fmt.Errorf("parameter %s must be an integer", paramName)
		}
		if schema.Min != nil && float64(intVal) < *schema.Min {
			return fmt.Errorf("parameter %s value %d is less than minimum %v", paramName, intVal, *schema.Min)
		}
		if schema.Max != nil && float64(intVal) > *schema.Max {
			return fmt.Errorf("parameter %s value %d is greater than maximum %v", paramName, intVal, *schema.Max)
		}

	case "number":
		numVal, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("parameter %s must be a number", paramName)
		}
		if schema.Min != nil && numVal < *schema.Min {
			return fmt.Errorf("parameter %s value %v is less than minimum %v", paramName, numVal, *schema.Min)
		}
		if schema.Max != nil && numVal > *schema.Max {
			return fmt.Errorf("parameter %s value %v is greater than maximum %v", paramName, numVal, *schema.Max)
		}

	case "string":
		if schema.MinLength > 0 && uint64(len(value)) < schema.MinLength {
			return fmt.Errorf("parameter %s length must be >= %d", paramName, schema.MinLength)
		}
		if schema.MaxLength != nil && uint64(len(value)) > *schema.MaxLength {
			return fmt.Errorf("parameter %s length must be <= %d", paramName, *schema.MaxLength)
		}
		if schema.Pattern != "" {
			if matched, _ := regexp.MatchString(schema.Pattern, value); !matched {
				return fmt.Errorf("parameter %s must match pattern: %s", paramName, schema.Pattern)
			}
		}
	}

	return nil
}
