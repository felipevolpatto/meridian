package validation

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

// SecurityValidator handles validation of security requirements
type SecurityValidator struct {
	spec *openapi3.T
}

// NewSecurityValidator creates a new security validator
func NewSecurityValidator(spec *openapi3.T) *SecurityValidator {
	return &SecurityValidator{spec: spec}
}

// ValidateSecurity validates the security requirements for a request
func (v *SecurityValidator) ValidateSecurity(op *openapi3.Operation, headers http.Header, query map[string][]string) ValidationErrors {
	var errors ValidationErrors

	// If no security requirements are defined, access is allowed
	if op.Security == nil && v.spec.Security == nil {
		return errors
	}

	// Use operation-specific security requirements if defined, otherwise use global requirements
	var requirements openapi3.SecurityRequirements
	if op.Security != nil {
		requirements = *op.Security
	} else {
		requirements = v.spec.Security
	}

	// No security requirements means no security needed
	if len(requirements) == 0 {
		return errors
	}

	// Try each security requirement (OR relationship between requirements)
	var requirementErrors ValidationErrors
	for _, requirement := range requirements {
		requirementErrs := v.validateSecurityRequirement(requirement, headers, query)
		if len(requirementErrs) == 0 {
			// If any requirement is met, security validation passes
			return errors
		}
		requirementErrors = append(requirementErrors, requirementErrs...)
	}

	// If we get here, no security requirement was met
	return requirementErrors
}

func (v *SecurityValidator) validateSecurityRequirement(requirement openapi3.SecurityRequirement, headers http.Header, query map[string][]string) ValidationErrors {
	var errors ValidationErrors

	// Each name/scope pair in the requirement must be satisfied (AND relationship)
	for schemeName, scopes := range requirement {
		scheme := v.spec.Components.SecuritySchemes[schemeName].Value
		if scheme == nil {
			errors = append(errors, &ValidationError{
				Message: fmt.Sprintf("Security scheme %s not found", schemeName),
				Code:    "security_scheme_not_found",
			})
			continue
		}

		var schemeErrors ValidationErrors
		switch scheme.Type {
		case "apiKey":
			schemeErrors = v.validateAPIKey(scheme, headers, query)
		case "http":
			schemeErrors = v.validateHTTP(scheme, headers)
		case "oauth2":
			schemeErrors = v.validateOAuth2(scheme, headers, scopes)
		case "openIdConnect":
			schemeErrors = v.validateOpenIDConnect(scheme, headers, scopes)
		default:
			schemeErrors = append(schemeErrors, &ValidationError{
				Message: fmt.Sprintf("Unsupported security scheme type: %s", scheme.Type),
				Code:    "unsupported_security_scheme",
			})
		}

		if len(schemeErrors) > 0 {
			errors = append(errors, schemeErrors...)
		}
	}

	return errors
}

func (v *SecurityValidator) validateAPIKey(scheme *openapi3.SecurityScheme, headers http.Header, query map[string][]string) ValidationErrors {
	var errors ValidationErrors

	var value string
	switch scheme.In {
	case "header":
		value = headers.Get(scheme.Name)
	case "query":
		if values, ok := query[scheme.Name]; ok && len(values) > 0 {
			value = values[0]
		}
	}

	if value == "" {
		errors = append(errors, &ValidationError{
			Field:   scheme.Name,
			Message: fmt.Sprintf("Required %s API key not provided", scheme.In),
			Code:    "missing_api_key",
		})
	}

	return errors
}

func (v *SecurityValidator) validateHTTP(scheme *openapi3.SecurityScheme, headers http.Header) ValidationErrors {
	var errors ValidationErrors

	auth := headers.Get("Authorization")
	if auth == "" {
		errors = append(errors, &ValidationError{
			Field:   "Authorization",
			Message: "Authorization header is required",
			Code:    "missing_authorization",
		})
		return errors
	}

	switch scheme.Scheme {
	case "basic":
		if !strings.HasPrefix(auth, "Basic ") {
			errors = append(errors, &ValidationError{
				Field:   "Authorization",
				Message: "Basic authentication required",
				Code:    "invalid_auth_scheme",
			})
			return errors
		}

		credentials := strings.TrimPrefix(auth, "Basic ")
		decoded, err := base64.StdEncoding.DecodeString(credentials)
		if err != nil {
			errors = append(errors, &ValidationError{
				Field:   "Authorization",
				Message: "Invalid Basic authentication credentials",
				Code:    "invalid_basic_auth",
			})
			return errors
		}

		parts := strings.SplitN(string(decoded), ":", 2)
		if len(parts) != 2 {
			errors = append(errors, &ValidationError{
				Field:   "Authorization",
				Message: "Invalid Basic authentication format",
				Code:    "invalid_basic_auth_format",
			})
		}

	case "bearer":
		if !strings.HasPrefix(auth, "Bearer ") {
			errors = append(errors, &ValidationError{
				Field:   "Authorization",
				Message: "Bearer authentication required",
				Code:    "invalid_auth_scheme",
			})
			return errors
		}

		token := strings.TrimPrefix(auth, "Bearer ")
		if token == "" {
			errors = append(errors, &ValidationError{
				Field:   "Authorization",
				Message: "Bearer token is required",
				Code:    "missing_bearer_token",
			})
		}

	default:
		errors = append(errors, &ValidationError{
			Message: fmt.Sprintf("Unsupported HTTP authentication scheme: %s", scheme.Scheme),
			Code:    "unsupported_auth_scheme",
		})
	}

	return errors
}

func (v *SecurityValidator) validateOAuth2(scheme *openapi3.SecurityScheme, headers http.Header, requiredScopes []string) ValidationErrors {
	var errors ValidationErrors

	auth := headers.Get("Authorization")
	if auth == "" {
		errors = append(errors, &ValidationError{
			Field:   "Authorization",
			Message: "Authorization header is required for OAuth2",
			Code:    "missing_authorization",
		})
		return errors
	}

	if !strings.HasPrefix(auth, "Bearer ") {
		errors = append(errors, &ValidationError{
			Field:   "Authorization",
			Message: "Bearer token required for OAuth2",
			Code:    "invalid_auth_scheme",
		})
		return errors
	}

	// Note: Actual token validation and scope checking would be done by the security handler
	// This validator only checks the presence and format of the token

	return errors
}

func (v *SecurityValidator) validateOpenIDConnect(scheme *openapi3.SecurityScheme, headers http.Header, requiredScopes []string) ValidationErrors {
	// OpenID Connect validation is similar to OAuth2
	return v.validateOAuth2(scheme, headers, requiredScopes)
} 