package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

// executeTemplate executes a template with error handling and CSP nonce support
func executeTemplate(w http.ResponseWriter, name string, data interface{}) {
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

// renderTemplate renders a template with CSP nonce support
func renderTemplate(w http.ResponseWriter, r *http.Request, name string, data interface{}) {
	// Get nonce from context
	nonce := ""
	if n, ok := r.Context().Value("csp-nonce").(string); ok {
		nonce = n
	}
	
	// Check if data is a map, if so add CSPNonce to it
	if mapData, ok := data.(map[string]interface{}); ok {
		mapData["CSPNonce"] = nonce
		executeTemplate(w, name, mapData)
		return
	}
	
	// For struct data, we need to use reflection to add CSPNonce
	// For now, just pass the data directly with CSPNonce in a wrapper
	// that preserves the original structure
	type TemplateData struct {
		CSPNonce string
	}
	
	// Create a map to hold all the data
	templateData := make(map[string]interface{})
	
	// Use reflection to convert struct to map
	if data != nil {
		dataType := reflect.TypeOf(data)
		dataValue := reflect.ValueOf(data)
		
		if dataType.Kind() == reflect.Struct {
			for i := 0; i < dataType.NumField(); i++ {
				field := dataType.Field(i)
				value := dataValue.Field(i).Interface()
				templateData[field.Name] = value
			}
		} else {
			// If it's not a struct, just add it as Data
			templateData["Data"] = data
		}
	}
	
	// Add CSPNonce
	templateData["CSPNonce"] = nonce
	
	executeTemplate(w, name, templateData)
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

// Helper functions (removed duplicates)

// Email and phone regex patterns
var (
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	phoneRegex = regexp.MustCompile(`[^\d]`)
)

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
func mergeErrors(errs []error) error {
	if len(errs) == 0 {
		return nil
	}
	
	if len(errs) == 1 {
		return errs[0]
	}
	
	var msg string
	for i, err := range errs {
		if i > 0 {
			msg += "; "
		}
		msg += err.Error()
	}
	return errors.New(msg)
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
