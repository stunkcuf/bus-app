package main

import (
	"encoding/json"
	"fmt"
	"html"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// Validation constants (non-duplicated)
const (
	MaxUsernameLength    = 50
	MinUsernameLength    = 3
	MaxPasswordLength    = 128
	MaxNameLength        = 200
	MaxPhoneLength       = 20
	MaxEmailLength       = 100
	MaxAddressLength     = 500
	MaxNotesLength       = 1000
	MaxMultipartMemory   = 10 << 20 // 10MB
)

// Regular expressions for validation (non-duplicated)
var (
	usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	busIDRegex    = regexp.MustCompile(`^[A-Z0-9\-]+$`)
	routeIDRegex  = regexp.MustCompile(`^[A-Z0-9\-]+$`)
	timeRegex     = regexp.MustCompile(`^([01]?[0-9]|2[0-3]):[0-5][0-9]$`)
	dateRegex     = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
)

// AllowedFileTypes defines allowed file extensions for uploads
var AllowedFileTypes = map[string][]string{
	"excel":  {".xlsx", ".xls", ".csv"},
	"image":  {".jpg", ".jpeg", ".png", ".gif", ".bmp"},
	"document": {".pdf", ".doc", ".docx"},
}

// ValidationRule represents a validation rule
type ValidationRule struct {
	Field      string
	Value      interface{}
	Rules      []Validator
	Optional   bool
}

// Validator is a function that validates a value
type Validator func(value interface{}, field string) *AppError

// ValidationMiddleware provides input validation for requests
func ValidationMiddleware(rules map[string][]ValidationRule) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// Get rules for this endpoint
			endpointRules, ok := rules[r.URL.Path]
			if !ok {
				// No validation rules for this endpoint
				next(w, r)
				return
			}

			// Parse form if needed
			if r.Method == "POST" || r.Method == "PUT" || r.Method == "PATCH" {
				if strings.Contains(r.Header.Get("Content-Type"), "multipart/form-data") {
					if err := r.ParseMultipartForm(MaxMultipartMemory); err != nil {
						SendError(w, ErrBadRequest("Failed to parse form data"))
						return
					}
				} else if strings.Contains(r.Header.Get("Content-Type"), "application/x-www-form-urlencoded") {
					if err := r.ParseForm(); err != nil {
						SendError(w, ErrBadRequest("Failed to parse form data"))
						return
					}
				}
			}

			// Validate input
			var errors []*AppError
			for _, rule := range endpointRules {
				value := getValueFromRequest(r, rule.Field)
				
				// Check if field is required
				if !rule.Optional && (value == nil || value == "") {
					errors = append(errors, ErrValidation(fmt.Sprintf("%s is required", rule.Field)).WithField(rule.Field))
					continue
				}

				// Skip validation if optional and empty
				if rule.Optional && (value == nil || value == "") {
					continue
				}

				// Apply validation rules
				for _, validator := range rule.Rules {
					if err := validator(value, rule.Field); err != nil {
						errors = append(errors, err)
					}
				}
			}

			if len(errors) > 0 {
				SendValidationErrors(w, errors)
				return
			}

			// Sanitize input
			sanitizeRequest(r)

			next(w, r)
		}
	}
}

// getValueFromRequest extracts value from request based on field name
func getValueFromRequest(r *http.Request, field string) interface{} {
	// Check URL parameters
	if val := r.URL.Query().Get(field); val != "" {
		return val
	}

	// Check form values
	if val := r.FormValue(field); val != "" {
		return val
	}

	// Check JSON body
	if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		var data map[string]interface{}
		body, _ := io.ReadAll(r.Body)
		r.Body = io.NopCloser(strings.NewReader(string(body)))
		
		if err := json.Unmarshal(body, &data); err == nil {
			if val, ok := data[field]; ok {
				return val
			}
		}
	}

	// Check file uploads
	if r.MultipartForm != nil && r.MultipartForm.File != nil {
		if files, ok := r.MultipartForm.File[field]; ok && len(files) > 0 {
			return files[0]
		}
	}

	return nil
}

// Common validators

// Required validates that a field is not empty
func Required() Validator {
	return func(value interface{}, field string) *AppError {
		if value == nil || fmt.Sprintf("%v", value) == "" {
			return ErrValidation(fmt.Sprintf("%s is required", field)).WithField(field)
		}
		return nil
	}
}

// StringLength validates string length
func StringLength(min, max int) Validator {
	return func(value interface{}, field string) *AppError {
		str := fmt.Sprintf("%v", value)
		length := utf8.RuneCountInString(str)
		
		if length < min {
			return ErrValidation(fmt.Sprintf("%s must be at least %d characters", field, min)).WithField(field)
		}
		if max > 0 && length > max {
			return ErrValidation(fmt.Sprintf("%s must be at most %d characters", field, max)).WithField(field)
		}
		return nil
	}
}

// Pattern validates against a regular expression
func Pattern(regex *regexp.Regexp, message string) Validator {
	return func(value interface{}, field string) *AppError {
		str := fmt.Sprintf("%v", value)
		if !regex.MatchString(str) {
			return ErrValidation(message).WithField(field)
		}
		return nil
	}
}

// Username validates username format
func Username() Validator {
	return func(value interface{}, field string) *AppError {
		username := fmt.Sprintf("%v", value)
		
		if len(username) < MinUsernameLength || len(username) > MaxUsernameLength {
			return ErrValidation(fmt.Sprintf("Username must be between %d and %d characters", MinUsernameLength, MaxUsernameLength)).WithField(field)
		}
		
		if !usernameRegex.MatchString(username) {
			return ErrValidation("Username can only contain letters, numbers, underscores, and hyphens").WithField(field)
		}
		
		return nil
	}
}

// Email validates email format
func Email() Validator {
	return func(value interface{}, field string) *AppError {
		email := fmt.Sprintf("%v", value)
		
		if len(email) > MaxEmailLength {
			return ErrValidation("Email address is too long").WithField(field)
		}
		
		if !emailRegex.MatchString(email) {
			return ErrValidation("Invalid email format").WithField(field)
		}
		
		return nil
	}
}

// Phone validates phone number format
func Phone() Validator {
	return func(value interface{}, field string) *AppError {
		phone := fmt.Sprintf("%v", value)
		
		if len(phone) > MaxPhoneLength {
			return ErrValidation("Phone number is too long").WithField(field)
		}
		
		if !phoneRegex.MatchString(phone) {
			return ErrValidation("Invalid phone number format").WithField(field)
		}
		
		return nil
	}
}

// Integer validates integer values
func Integer(min, max int) Validator {
	return func(value interface{}, field string) *AppError {
		var intVal int
		
		switch v := value.(type) {
		case int:
			intVal = v
		case float64:
			intVal = int(v)
		case string:
			var err error
			intVal, err = strconv.Atoi(v)
			if err != nil {
				return ErrValidation(fmt.Sprintf("%s must be a valid number", field)).WithField(field)
			}
		default:
			return ErrValidation(fmt.Sprintf("%s must be a valid number", field)).WithField(field)
		}
		
		if intVal < min {
			return ErrValidation(fmt.Sprintf("%s must be at least %d", field, min)).WithField(field)
		}
		if max > 0 && intVal > max {
			return ErrValidation(fmt.Sprintf("%s must be at most %d", field, max)).WithField(field)
		}
		
		return nil
	}
}

// Float validates float values
func Float(min, max float64) Validator {
	return func(value interface{}, field string) *AppError {
		var floatVal float64
		
		switch v := value.(type) {
		case float64:
			floatVal = v
		case int:
			floatVal = float64(v)
		case string:
			var err error
			floatVal, err = strconv.ParseFloat(v, 64)
			if err != nil {
				return ErrValidation(fmt.Sprintf("%s must be a valid number", field)).WithField(field)
			}
		default:
			return ErrValidation(fmt.Sprintf("%s must be a valid number", field)).WithField(field)
		}
		
		if floatVal < min {
			return ErrValidation(fmt.Sprintf("%s must be at least %.2f", field, min)).WithField(field)
		}
		if max > 0 && floatVal > max {
			return ErrValidation(fmt.Sprintf("%s must be at most %.2f", field, max)).WithField(field)
		}
		
		return nil
	}
}

// Date validates date format (YYYY-MM-DD)
func Date() Validator {
	return func(value interface{}, field string) *AppError {
		dateStr := fmt.Sprintf("%v", value)
		
		if !dateRegex.MatchString(dateStr) {
			return ErrValidation("Date must be in YYYY-MM-DD format").WithField(field)
		}
		
		_, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			return ErrValidation("Invalid date").WithField(field)
		}
		
		return nil
	}
}

// Time validates time format (HH:MM)
func Time() Validator {
	return func(value interface{}, field string) *AppError {
		timeStr := fmt.Sprintf("%v", value)
		
		if !timeRegex.MatchString(timeStr) {
			return ErrValidation("Time must be in HH:MM format").WithField(field)
		}
		
		return nil
	}
}

// BusID validates bus ID format
func BusID() Validator {
	return func(value interface{}, field string) *AppError {
		busID := fmt.Sprintf("%v", value)
		
		if len(busID) > 50 {
			return ErrValidation("Bus ID is too long").WithField(field)
		}
		
		if !busIDRegex.MatchString(busID) {
			return ErrValidation("Bus ID can only contain uppercase letters, numbers, and hyphens").WithField(field)
		}
		
		return nil
	}
}

// RouteID validates route ID format
func RouteID() Validator {
	return func(value interface{}, field string) *AppError {
		routeID := fmt.Sprintf("%v", value)
		
		if len(routeID) > 50 {
			return ErrValidation("Route ID is too long").WithField(field)
		}
		
		if !routeIDRegex.MatchString(routeID) {
			return ErrValidation("Route ID can only contain uppercase letters, numbers, and hyphens").WithField(field)
		}
		
		return nil
	}
}

// FileUpload validates file uploads
func FileUpload(fileType string, maxSize int64) Validator {
	return func(value interface{}, field string) *AppError {
		fileHeader, ok := value.(*multipart.FileHeader)
		if !ok {
			return ErrValidation("Invalid file upload").WithField(field)
		}
		
		// Check file size
		if fileHeader.Size > maxSize {
			return ErrValidation(fmt.Sprintf("File size must not exceed %d MB", maxSize/(1<<20))).WithField(field)
		}
		
		// Check file type
		ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
		allowedExts, ok := AllowedFileTypes[fileType]
		if !ok {
			return ErrValidation("Invalid file type category").WithField(field)
		}
		
		allowed := false
		for _, allowedExt := range allowedExts {
			if ext == allowedExt {
				allowed = true
				break
			}
		}
		
		if !allowed {
			return ErrValidation(fmt.Sprintf("File type must be one of: %s", strings.Join(allowedExts, ", "))).WithField(field)
		}
		
		return nil
	}
}

// OneOf validates that value is one of allowed values
func OneOf(values []string) Validator {
	return func(value interface{}, field string) *AppError {
		strVal := fmt.Sprintf("%v", value)
		
		for _, allowed := range values {
			if strVal == allowed {
				return nil
			}
		}
		
		return ErrValidation(fmt.Sprintf("%s must be one of: %s", field, strings.Join(values, ", "))).WithField(field)
	}
}

// sanitizeRequest sanitizes request input
func sanitizeRequest(r *http.Request) {
	// Sanitize query parameters
	query := r.URL.Query()
	for key, values := range query {
		for i, value := range values {
			query[key][i] = sanitizeString(value)
		}
	}
	r.URL.RawQuery = query.Encode()
	
	// Sanitize form values
	if r.Form != nil {
		for key, values := range r.Form {
			for i, value := range values {
				r.Form[key][i] = sanitizeString(value)
			}
		}
	}
	
	// Sanitize multipart form
	if r.MultipartForm != nil && r.MultipartForm.Value != nil {
		for key, values := range r.MultipartForm.Value {
			for i, value := range values {
				r.MultipartForm.Value[key][i] = sanitizeString(value)
			}
		}
	}
}

// sanitizeString removes dangerous characters and HTML
func sanitizeString(input string) string {
	// Decode HTML entities
	input = html.UnescapeString(input)
	
	// Remove null bytes
	input = strings.ReplaceAll(input, "\x00", "")
	
	// Escape HTML
	input = html.EscapeString(input)
	
	// Trim whitespace
	input = strings.TrimSpace(input)
	
	return input
}

// Common validation rule sets

// LoginValidationRules defines validation for login
var LoginValidationRules = []ValidationRule{
	{
		Field: "username",
		Rules: []Validator{Required(), Username()},
	},
	{
		Field: "password",
		Rules: []Validator{Required(), StringLength(MinPasswordLength, MaxPasswordLength)},
	},
}

// RegisterValidationRules defines validation for registration
var RegisterValidationRules = []ValidationRule{
	{
		Field: "username",
		Rules: []Validator{Required(), Username()},
	},
	{
		Field: "password",
		Rules: []Validator{Required(), StringLength(MinPasswordLength, MaxPasswordLength)},
	},
	{
		Field: "confirm_password",
		Rules: []Validator{Required()},
	},
	{
		Field: "role",
		Rules: []Validator{Required(), OneOf([]string{"driver", "manager"})},
	},
}

// AddBusValidationRules defines validation for adding a bus
var AddBusValidationRules = []ValidationRule{
	{
		Field: "bus_id",
		Rules: []Validator{Required(), BusID()},
	},
	{
		Field: "capacity",
		Rules: []Validator{Required(), Integer(1, 100)},
	},
	{
		Field: "model",
		Rules: []Validator{StringLength(0, 100)},
		Optional: true,
	},
	{
		Field: "status",
		Rules: []Validator{OneOf([]string{"active", "maintenance", "out_of_service"})},
		Optional: true,
	},
}

// StudentValidationRules defines validation for student management
var StudentValidationRules = []ValidationRule{
	{
		Field: "name",
		Rules: []Validator{Required(), StringLength(2, MaxNameLength)},
	},
	{
		Field: "phone_number",
		Rules: []Validator{Phone()},
		Optional: true,
	},
	{
		Field: "alt_phone_number",
		Rules: []Validator{Phone()},
		Optional: true,
	},
	{
		Field: "guardian",
		Rules: []Validator{StringLength(2, MaxNameLength)},
		Optional: true,
	},
	{
		Field: "pickup_time",
		Rules: []Validator{Time()},
		Optional: true,
	},
	{
		Field: "dropoff_time",
		Rules: []Validator{Time()},
		Optional: true,
	},
}

// FileUploadValidationRules defines validation for file uploads
var FileUploadValidationRules = []ValidationRule{
	{
		Field: "file",
		Rules: []Validator{Required(), FileUpload("excel", MaxFileSize)},
	},
}

// CreateValidationRules creates a map of validation rules for endpoints
func CreateValidationRules() map[string][]ValidationRule {
	return map[string][]ValidationRule{
		"/":                    LoginValidationRules,
		"/register":            RegisterValidationRules,
		"/add-bus":             AddBusValidationRules,
		"/add-student":         StudentValidationRules,
		"/edit-student":        StudentValidationRules,
		"/import-mileage":      FileUploadValidationRules,
		"/import-ecse":         FileUploadValidationRules,
	}
}