package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// executeTemplate executes a template with error handling
func executeTemplate(w http.ResponseWriter, name string, data interface{}) {
	// Add CSP nonce from context if available
	if r, ok := w.(*responseWriter); ok && r.request != nil {
		if nonce, ok := r.request.Context().Value("csp-nonce").(string); ok {
			// If data is a struct, we need to add the nonce
			// This is a simplified version - in production, you'd want a more robust solution
			log.Printf("CSP nonce available for template: %s", name)
		}
	}
	
	if err := templates.ExecuteTemplate(w, name, data); err != nil {
		log.Printf("Error executing template %s: %v", name, err)
		
		// In development mode, show detailed error
		if isDevelopment() {
			http.Error(w, fmt.Sprintf("Template error in %s: %v", name, err), http.StatusInternalServerError)
		} else {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	}
}

// Custom response writer to capture request context
type responseWriter struct {
	http.ResponseWriter
	request *http.Request
}

// isDevelopment checks if the app is running in development mode
func isDevelopment() bool {
	return os.Getenv("APP_ENV") == "development"
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// formatDate formats a date string for display
func formatDate(date string) string {
	// Add any date formatting logic here
	return date
}

// formatTime formats a time string for display
func formatTime(time string) string {
	// Add any time formatting logic here
	return time
}

// calculateMileage calculates mileage between two readings
func calculateMileage(start, end int) int {
	if end < start {
		return 0 // Invalid reading
	}
	return end - start
}

// generateUniqueID generates a unique ID with a prefix
func generateUniqueID(prefix string, count int) string {
	return fmt.Sprintf("%s%03d", prefix, count+1)
}

// truncateString safely truncates a string to a maximum length
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// isValidEmail checks if an email address is valid
func isValidEmail(email string) bool {
	// Simple email validation
	return len(email) > 3 && len(email) < 255 && 
		   emailRegex.MatchString(email)
}

// isValidPhone checks if a phone number is valid
func isValidPhone(phone string) bool {
	// Remove common formatting characters
	cleaned := phoneRegex.ReplaceAllString(phone, "")
	return len(cleaned) >= 10 && len(cleaned) <= 15
}

// Helper regex patterns
var (
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	phoneRegex = regexp.MustCompile(`[^\d]`)
)

// logError logs an error with context
func logError(context string, err error) {
	if err != nil {
		log.Printf("[ERROR] %s: %v", context, err)
	}
}

// logInfo logs an informational message
func logInfo(format string, args ...interface{}) {
	log.Printf("[INFO] "+format, args...)
}

// logDebug logs a debug message (only in development)
func logDebug(format string, args ...interface{}) {
	if isDevelopment() {
		log.Printf("[DEBUG] "+format, args...)
	}
}

// getEnv gets an environment variable with a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// mustGetEnv gets an environment variable or panics
func mustGetEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		panic(fmt.Sprintf("Environment variable %s is required", key))
	}
	return value
}

// stringInSlice checks if a string is in a slice
func stringInSlice(str string, slice []string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}

// removeFromSlice removes a string from a slice
func removeFromSlice(slice []string, str string) []string {
	result := make([]string, 0, len(slice))
	for _, s := range slice {
		if s != str {
			result = append(result, s)
		}
	}
	return result
}

// copyMap creates a shallow copy of a map
func copyMap(m map[string]string) map[string]string {
	result := make(map[string]string, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
}

// mergeErrors combines multiple errors into one
func mergeErrors(errors []error) error {
	if len(errors) == 0 {
		return nil
	}
	
	if len(errors) == 1 {
		return errors[0]
	}
	
	var msg string
	for i, err := range errors {
		if i > 0 {
			msg += "; "
		}
		msg += err.Error()
	}
	return fmt.Errorf(msg)
}

// sanitizeFilename removes potentially dangerous characters from filenames
func sanitizeFilename(filename string) string {
	// Remove any path separators
	filename = strings.ReplaceAll(filename, "/", "_")
	filename = strings.ReplaceAll(filename, "\\", "_")
	filename = strings.ReplaceAll(filename, "..", "_")
	
	// Keep only safe characters
	safe := regexp.MustCompile(`[^a-zA-Z0-9._-]`)
	return safe.ReplaceAllString(filename, "_")
}

// parseIntOrDefault parses an integer with a default value
func parseIntOrDefault(s string, defaultValue int) int {
	if val, err := strconv.Atoi(s); err == nil {
		return val
	}
	return defaultValue
}

// parseFloatOrDefault parses a float with a default value
func parseFloatOrDefault(s string, defaultValue float64) float64 {
	if val, err := strconv.ParseFloat(s, 64); err == nil {
		return val
	}
	return defaultValue
}
