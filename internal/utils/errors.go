package utils

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// AppError represents an application error
type AppError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
	Status  int    `json:"-"`
}

// Error implements the error interface
func (e AppError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Common error constructors
func ErrNotFound(message string) AppError {
	return AppError{
		Code:    "NOT_FOUND",
		Message: message,
		Status:  http.StatusNotFound,
	}
}

func ErrBadRequest(message string) AppError {
	return AppError{
		Code:    "BAD_REQUEST",
		Message: message,
		Status:  http.StatusBadRequest,
	}
}

func ErrUnauthorized(message string) AppError {
	return AppError{
		Code:    "UNAUTHORIZED",
		Message: message,
		Status:  http.StatusUnauthorized,
	}
}

func ErrForbidden(message string) AppError {
	return AppError{
		Code:    "FORBIDDEN",
		Message: message,
		Status:  http.StatusForbidden,
	}
}

func ErrConflict(message string) AppError {
	return AppError{
		Code:    "CONFLICT",
		Message: message,
		Status:  http.StatusConflict,
	}
}

func ErrDatabase(message string, err error) AppError {
	details := ""
	if err != nil {
		details = err.Error()
	}
	return AppError{
		Code:    "DATABASE_ERROR",
		Message: message,
		Details: details,
		Status:  http.StatusInternalServerError,
	}
}

func ErrInternal(message string) AppError {
	return AppError{
		Code:    "INTERNAL_ERROR",
		Message: message,
		Status:  http.StatusInternalServerError,
	}
}

// SendError sends an error response
func SendError(w http.ResponseWriter, err AppError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(err.Status)
	json.NewEncoder(w).Encode(err)
}

// SendJSON sends a JSON response
func SendJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		LogError("Failed to encode JSON response", err)
	}
}