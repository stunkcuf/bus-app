package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"runtime"
	"strings"
	"time"
)

// ErrorType represents the type of error
type ErrorType string

const (
	// Client errors (4xx)
	ErrorTypeValidation   ErrorType = "VALIDATION_ERROR"
	ErrorTypeUnauthorized ErrorType = "UNAUTHORIZED"
	ErrorTypeForbidden    ErrorType = "FORBIDDEN"
	ErrorTypeNotFound     ErrorType = "NOT_FOUND"
	ErrorTypeConflict     ErrorType = "CONFLICT"
	ErrorTypeRateLimit    ErrorType = "RATE_LIMIT"
	ErrorTypeBadRequest   ErrorType = "BAD_REQUEST"

	// Server errors (5xx)
	ErrorTypeInternal     ErrorType = "INTERNAL_ERROR"
	ErrorTypeDatabase     ErrorType = "DATABASE_ERROR"
	ErrorTypeExternal     ErrorType = "EXTERNAL_SERVICE_ERROR"
	ErrorTypeConfig       ErrorType = "CONFIGURATION_ERROR"
	ErrorTypeUnavailable  ErrorType = "SERVICE_UNAVAILABLE"
)

// AppError represents a structured application error
type AppError struct {
	Type       ErrorType   `json:"type"`
	Message    string      `json:"message"`
	Detail     string      `json:"detail,omitempty"`
	Field      string      `json:"field,omitempty"`
	Code       string      `json:"code,omitempty"`
	StatusCode int         `json:"-"`
	Internal   error       `json:"-"`
	Stack      string      `json:"-"`
	Timestamp  time.Time   `json:"timestamp"`
	RequestID  string      `json:"request_id,omitempty"`
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Internal != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Internal)
	}
	return e.Message
}

// NewAppError creates a new application error
func NewAppError(errType ErrorType, message string, internal error) *AppError {
	// Capture stack trace
	buf := make([]byte, 2048)
	n := runtime.Stack(buf, false)
	stack := string(buf[:n])

	// Determine status code based on error type
	statusCode := getStatusCodeForErrorType(errType)

	return &AppError{
		Type:       errType,
		Message:    message,
		StatusCode: statusCode,
		Internal:   internal,
		Stack:      stack,
		Timestamp:  time.Now(),
	}
}

// WithDetail adds additional detail to the error
func (e *AppError) WithDetail(detail string) *AppError {
	e.Detail = detail
	return e
}

// WithField adds a field name to the error (useful for validation errors)
func (e *AppError) WithField(field string) *AppError {
	e.Field = field
	return e
}

// WithCode adds an error code
func (e *AppError) WithCode(code string) *AppError {
	e.Code = code
	return e
}

// WithRequestID adds a request ID for tracking
func (e *AppError) WithRequestID(requestID string) *AppError {
	e.RequestID = requestID
	return e
}

// getStatusCodeForErrorType returns the appropriate HTTP status code for an error type
func getStatusCodeForErrorType(errType ErrorType) int {
	switch errType {
	case ErrorTypeValidation, ErrorTypeBadRequest:
		return http.StatusBadRequest
	case ErrorTypeUnauthorized:
		return http.StatusUnauthorized
	case ErrorTypeForbidden:
		return http.StatusForbidden
	case ErrorTypeNotFound:
		return http.StatusNotFound
	case ErrorTypeConflict:
		return http.StatusConflict
	case ErrorTypeRateLimit:
		return http.StatusTooManyRequests
	case ErrorTypeUnavailable:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}

// Common error creators
func ErrValidation(message string) *AppError {
	return NewAppError(ErrorTypeValidation, message, nil)
}

func ErrBadRequest(message string) *AppError {
	return NewAppError(ErrorTypeValidation, message, nil)
}

func ErrUnauthorized(message string) *AppError {
	return NewAppError(ErrorTypeUnauthorized, message, nil)
}

func ErrForbidden(message string) *AppError {
	return NewAppError(ErrorTypeForbidden, message, nil)
}

func ErrNotFound(resource string) *AppError {
	return NewAppError(ErrorTypeNotFound, fmt.Sprintf("%s not found", resource), nil)
}

func ErrConflict(message string) *AppError {
	return NewAppError(ErrorTypeConflict, message, nil)
}

func ErrDatabase(operation string, err error) *AppError {
	message := fmt.Sprintf("Database error during %s", operation)
	return NewAppError(ErrorTypeDatabase, message, err)
}

func ErrInternal(message string, err error) *AppError {
	return NewAppError(ErrorTypeInternal, message, err)
}

func ErrMethodNotAllowed(message string) *AppError {
	return NewAppError(ErrorTypeValidation, message, nil)
}

// ErrorResponse represents the JSON error response
type ErrorResponse struct {
	Success bool        `json:"success"`
	Error   *AppError   `json:"error"`
	Errors  []*AppError `json:"errors,omitempty"` // For multiple validation errors
}

// SendError sends a structured error response
func SendError(w http.ResponseWriter, err error) {
	// Type assertion to check if it's an AppError
	appErr, ok := err.(*AppError)
	if !ok {
		// Convert regular error to AppError
		appErr = ErrInternal("An unexpected error occurred", err)
	}

	// Log error with context
	logError(appErr)

	// Prepare response
	response := ErrorResponse{
		Success: false,
		Error:   appErr,
	}

	// Don't expose internal error details in production
	if isProduction() && appErr.StatusCode >= 500 {
		appErr.Message = "An internal server error occurred"
		appErr.Detail = ""
		appErr.Internal = nil
	}

	// Send JSON response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(appErr.StatusCode)
	
	if err := json.NewEncoder(w).Encode(response); err != nil {
		// Fallback to plain text if JSON encoding fails
		http.Error(w, appErr.Message, appErr.StatusCode)
	}
}

// SendValidationErrors sends multiple validation errors
func SendValidationErrors(w http.ResponseWriter, errors []*AppError) {
	response := ErrorResponse{
		Success: false,
		Errors:  errors,
	}

	if len(errors) > 0 {
		response.Error = errors[0] // Set the first error as the main error
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(response)
}

// logError logs error with full context
func logError(err *AppError) {
	// Build log message
	var logMsg strings.Builder
	logMsg.WriteString(fmt.Sprintf("[ERROR] Type: %s, Status: %d\n", err.Type, err.StatusCode))
	logMsg.WriteString(fmt.Sprintf("Message: %s\n", err.Message))
	
	if err.Detail != "" {
		logMsg.WriteString(fmt.Sprintf("Detail: %s\n", err.Detail))
	}
	
	if err.Field != "" {
		logMsg.WriteString(fmt.Sprintf("Field: %s\n", err.Field))
	}
	
	if err.Code != "" {
		logMsg.WriteString(fmt.Sprintf("Code: %s\n", err.Code))
	}
	
	if err.RequestID != "" {
		logMsg.WriteString(fmt.Sprintf("RequestID: %s\n", err.RequestID))
	}
	
	if err.Internal != nil {
		logMsg.WriteString(fmt.Sprintf("Internal: %v\n", err.Internal))
	}
	
	// Log stack trace for server errors
	if err.StatusCode >= 500 {
		logMsg.WriteString(fmt.Sprintf("Stack:\n%s\n", err.Stack))
	}
	
	log.Print(logMsg.String())
}

// isProduction checks if the app is running in production
func isProduction() bool {
	env := getEnv("APP_ENV", "development")
	return env == "production"
}

// RecoverMiddleware recovers from panics and returns appropriate error
func RecoverMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				// Log panic with stack trace
				buf := make([]byte, 4096)
				n := runtime.Stack(buf, false)
				stack := string(buf[:n])
				
				log.Printf("[PANIC] %v\nStack:\n%s", err, stack)
				
				// Send error response
				appErr := NewAppError(
					ErrorTypeInternal,
					"An unexpected error occurred",
					fmt.Errorf("panic: %v", err),
				)
				SendError(w, appErr)
			}
		}()
		
		next(w, r)
	}
}

// ValidateRequired validates required fields
func ValidateRequired(fields map[string]string) []*AppError {
	var errors []*AppError
	
	for field, value := range fields {
		if strings.TrimSpace(value) == "" {
			err := ErrValidation(fmt.Sprintf("%s is required", field)).WithField(field)
			errors = append(errors, err)
		}
	}
	
	return errors
}

// ValidateEmail validates email format
func ValidateEmail(email string) *AppError {
	// Simple email validation
	if !strings.Contains(email, "@") || !strings.Contains(email, ".") {
		return ErrValidation("Invalid email format").WithField("email")
	}
	return nil
}

// ValidateLength validates string length
func ValidateLength(field, value string, min, max int) *AppError {
	length := len(value)
	if length < min {
		return ErrValidation(fmt.Sprintf("%s must be at least %d characters", field, min)).WithField(field)
	}
	if max > 0 && length > max {
		return ErrValidation(fmt.Sprintf("%s must be at most %d characters", field, max)).WithField(field)
	}
	return nil
}

// ValidateNumericRange validates numeric values
func ValidateNumericRange(field string, value, min, max float64) *AppError {
	if value < min {
		return ErrValidation(fmt.Sprintf("%s must be at least %.2f", field, min)).WithField(field)
	}
	if max > 0 && value > max {
		return ErrValidation(fmt.Sprintf("%s must be at most %.2f", field, max)).WithField(field)
	}
	return nil
}

// WrapError wraps an error with additional context
func WrapError(err error, message string) error {
	if err == nil {
		return nil
	}
	
	// If it's already an AppError, add detail
	if appErr, ok := err.(*AppError); ok {
		return appErr.WithDetail(message)
	}
	
	// Otherwise create new AppError
	return ErrInternal(message, err)
}