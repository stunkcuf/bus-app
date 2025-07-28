package utils

import (
	"fmt"
	"log"
	"os"
	"time"
)

// Logger levels
const (
	INFO  = "INFO"
	WARN  = "WARN"
	ERROR = "ERROR"
	FATAL = "FATAL"
	DEBUG = "DEBUG"
)

// LogInfo logs an informational message
func LogInfo(message string) {
	log.Printf("[%s] %s", INFO, message)
}

// LogWarn logs a warning message
func LogWarn(message string) {
	log.Printf("[%s] %s", WARN, message)
}

// LogError logs an error message
func LogError(message string, err error) {
	if err != nil {
		log.Printf("[%s] %s: %v", ERROR, message, err)
	} else {
		log.Printf("[%s] %s", ERROR, message)
	}
}

// LogFatal logs a fatal error and exits
func LogFatal(message string, err error) {
	if err != nil {
		log.Fatalf("[%s] %s: %v", FATAL, message, err)
	} else {
		log.Fatalf("[%s] %s", FATAL, message)
	}
}

// LogDebug logs a debug message
func LogDebug(message string) {
	if os.Getenv("DEBUG") == "true" {
		log.Printf("[%s] %s", DEBUG, message)
	}
}

// LogWithFields logs a message with structured fields
func LogWithFields(level string, message string, fields map[string]interface{}) {
	fieldStr := ""
	for k, v := range fields {
		fieldStr += fmt.Sprintf(" %s=%v", k, v)
	}
	log.Printf("[%s] %s%s", level, message, fieldStr)
}

// SetupLogger configures the logger
func SetupLogger() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	
	// Create logs directory if it doesn't exist
	if err := os.MkdirAll("logs", 0755); err != nil {
		log.Printf("Failed to create logs directory: %v", err)
		return
	}
	
	// Create log file
	logFile := fmt.Sprintf("logs/app_%s.log", time.Now().Format("2006-01-02"))
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Printf("Failed to open log file: %v", err)
		return
	}
	
	// Set output to file
	log.SetOutput(file)
}