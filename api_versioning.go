package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// APIVersion represents an API version
type APIVersion string

const (
	APIVersion1 APIVersion = "v1"
	APIVersion2 APIVersion = "v2"
	// Add more versions as needed
)

// VersionedAPIResponse extends the existing APIResponse with versioning
type VersionedAPIResponse struct {
	APIResponse
	Version   string `json:"version"`
	Timestamp string `json:"timestamp"`
}

// APIErrorResponse represents an API error response
type APIErrorResponse struct {
	Version   string `json:"version"`
	Success   bool   `json:"success"`
	Error     string `json:"error"`
	Code      string `json:"code,omitempty"`
	Timestamp string `json:"timestamp"`
}

// withAPIVersion wraps a handler with API versioning
func withAPIVersion(version APIVersion, handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Set response headers
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("API-Version", string(version))
		
		// Add version to request context
		ctx := r.Context()
		ctx = setAPIVersion(ctx, version)
		r = r.WithContext(ctx)
		
		// Call the original handler
		handler(w, r)
	}
}

// sendVersionedAPIResponse sends a versioned API response
func sendVersionedAPIResponse(w http.ResponseWriter, r *http.Request, data interface{}, message string) {
	version := getAPIVersionFromContext(r.Context())
	if version == "" {
		version = string(APIVersion1) // Default to v1
	}
	
	response := VersionedAPIResponse{
		APIResponse: APIResponse{
			Success: true,
			Data:    data,
			Message: message,
		},
		Version:   version,
		Timestamp: getCurrentTimestamp(),
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// sendVersionedAPIError sends a versioned API error response
func sendVersionedAPIError(w http.ResponseWriter, message string, statusCode int) {
	sendVersionedAPIErrorWithCode(w, message, "", statusCode)
}

// sendVersionedAPIErrorWithCode sends a versioned API error response with error code
func sendVersionedAPIErrorWithCode(w http.ResponseWriter, message, code string, statusCode int) {
	response := APIErrorResponse{
		Version:   string(APIVersion1), // Default version for errors
		Success:   false,
		Error:     message,
		Code:      code,
		Timestamp: getCurrentTimestamp(),
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

// extractAPIVersionFromPath extracts API version from URL path
func extractAPIVersionFromPath(path string) APIVersion {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) >= 2 && parts[0] == "api" {
		switch parts[1] {
		case "v1":
			return APIVersion1
		case "v2":
			return APIVersion2
		default:
			return APIVersion1 // Default to v1 for backward compatibility
		}
	}
	return APIVersion1
}

// getCurrentTimestamp returns current timestamp in ISO format
func getCurrentTimestamp() string {
	return time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
}

// Context key for API version
type apiVersionContextKey string

const apiVersionKey apiVersionContextKey = "api_version"

// setAPIVersion sets the API version in context
func setAPIVersion(ctx context.Context, version APIVersion) context.Context {
	return context.WithValue(ctx, apiVersionKey, string(version))
}

// getAPIVersionFromContext gets the API version from context
func getAPIVersionFromContext(ctx context.Context) string {
	if version, ok := ctx.Value(apiVersionKey).(string); ok {
		return version
	}
	return string(APIVersion1) // Default to v1
}

// Helper functions for setting up versioned routes
// These will be used in main.go where mux is available

// GetVersionedAPIPattern returns the pattern for a versioned API endpoint
func GetVersionedAPIPattern(basePath string, version APIVersion) string {
	return fmt.Sprintf("/api/%s%s", string(version), basePath)
}

// GetLegacyAPIPattern returns the pattern for legacy (unversioned) API endpoint
func GetLegacyAPIPattern(basePath string) string {
	return fmt.Sprintf("/api%s", basePath)
}