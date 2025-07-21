package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
)

// LogLevel represents the severity of a log entry
type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
	LogLevelFatal
)

var logLevelNames = map[LogLevel]string{
	LogLevelDebug: "DEBUG",
	LogLevelInfo:  "INFO",
	LogLevelWarn:  "WARN",
	LogLevelError: "ERROR",
	LogLevelFatal: "FATAL",
}

// LogEntry represents a structured log entry
type LogEntry struct {
	Timestamp  time.Time              `json:"timestamp"`
	Level      string                 `json:"level"`
	Message    string                 `json:"message"`
	Error      string                 `json:"error,omitempty"`
	RequestID  string                 `json:"request_id,omitempty"`
	UserID     string                 `json:"user_id,omitempty"`
	Method     string                 `json:"method,omitempty"`
	Path       string                 `json:"path,omitempty"`
	StatusCode int                    `json:"status_code,omitempty"`
	Duration   float64                `json:"duration_ms,omitempty"`
	IP         string                 `json:"ip,omitempty"`
	UserAgent  string                 `json:"user_agent,omitempty"`
	Fields     map[string]interface{} `json:"fields,omitempty"`
	Caller     string                 `json:"caller,omitempty"`
}

// Logger is the main logger instance
type Logger struct {
	mu     sync.RWMutex
	level  LogLevel
	output io.Writer
	json   bool
	fields map[string]interface{}
}

// Global logger instance
var logger *Logger

// InitLogger initializes the global logger
func InitLogger() {
	logLevel := LogLevelInfo
	jsonFormat := false

	// Set log level from environment
	switch strings.ToUpper(getEnv("LOG_LEVEL", "INFO")) {
	case "DEBUG":
		logLevel = LogLevelDebug
	case "WARN":
		logLevel = LogLevelWarn
	case "ERROR":
		logLevel = LogLevelError
	}

	// Enable JSON logging in production
	if isProduction() {
		jsonFormat = true
	}

	logger = &Logger{
		level:  logLevel,
		output: os.Stdout,
		json:   jsonFormat,
		fields: make(map[string]interface{}),
	}

	// Also set standard logger
	log.SetOutput(logger)
	log.SetFlags(0) // We handle formatting ourselves
}

// Write implements io.Writer interface for compatibility with standard log package
func (l *Logger) Write(p []byte) (n int, err error) {
	message := strings.TrimSpace(string(p))
	if message != "" {
		l.Info(message)
	}
	return len(p), nil
}

// WithField adds a field to the logger context
func (l *Logger) WithField(key string, value interface{}) *Logger {
	newLogger := &Logger{
		level:  l.level,
		output: l.output,
		json:   l.json,
		fields: make(map[string]interface{}),
	}

	// Copy existing fields
	for k, v := range l.fields {
		newLogger.fields[k] = v
	}

	// Add new field
	newLogger.fields[key] = value

	return newLogger
}

// WithFields adds multiple fields to the logger context
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	newLogger := &Logger{
		level:  l.level,
		output: l.output,
		json:   l.json,
		fields: make(map[string]interface{}),
	}

	// Copy existing fields
	for k, v := range l.fields {
		newLogger.fields[k] = v
	}

	// Add new fields
	for k, v := range fields {
		newLogger.fields[k] = v
	}

	return newLogger
}

// WithRequest adds request context to the logger
func (l *Logger) WithRequest(r *http.Request) *Logger {
	fields := map[string]interface{}{
		"method":     r.Method,
		"path":       r.URL.Path,
		"ip":         getClientIP(r),
		"user_agent": r.UserAgent(),
	}

	// Add request ID if available
	if requestID := r.Context().Value("requestID"); requestID != nil {
		fields["request_id"] = requestID
	}

	// Add user if available
	if user := getUserFromSession(r); user != nil {
		fields["user_id"] = user.Username
	}

	return l.WithFields(fields)
}

// log writes a log entry
func (l *Logger) log(level LogLevel, message string, err error) {
	if level < l.level {
		return
	}

	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     logLevelNames[level],
		Message:   message,
		Fields:    l.fields,
	}

	// Add error if provided
	if err != nil {
		entry.Error = err.Error()
	}

	// Add caller information
	if level >= LogLevelError {
		_, file, line, ok := runtime.Caller(2)
		if ok {
			parts := strings.Split(file, "/")
			entry.Caller = fmt.Sprintf("%s:%d", parts[len(parts)-1], line)
		}
	}

	// Format and write log entry
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.json {
		json.NewEncoder(l.output).Encode(entry)
	} else {
		// Human-readable format
		fmt.Fprintf(l.output, "[%s] %s %s",
			entry.Timestamp.Format("2006-01-02 15:04:05"),
			entry.Level,
			entry.Message,
		)

		if entry.Error != "" {
			fmt.Fprintf(l.output, " error=%q", entry.Error)
		}

		if entry.RequestID != "" {
			fmt.Fprintf(l.output, " request_id=%s", entry.RequestID)
		}

		if entry.UserID != "" {
			fmt.Fprintf(l.output, " user=%s", entry.UserID)
		}

		if entry.Method != "" && entry.Path != "" {
			fmt.Fprintf(l.output, " %s %s", entry.Method, entry.Path)
		}

		if entry.StatusCode > 0 {
			fmt.Fprintf(l.output, " status=%d", entry.StatusCode)
		}

		if entry.Duration > 0 {
			fmt.Fprintf(l.output, " duration=%.2fms", entry.Duration)
		}

		if entry.Caller != "" {
			fmt.Fprintf(l.output, " caller=%s", entry.Caller)
		}

		// Add custom fields
		for k, v := range entry.Fields {
			if k != "method" && k != "path" && k != "request_id" && k != "user_id" {
				fmt.Fprintf(l.output, " %s=%v", k, v)
			}
		}

		fmt.Fprintln(l.output)
	}
}

// Debug logs a debug message
func (l *Logger) Debug(message string) {
	l.log(LogLevelDebug, message, nil)
}

// Debugf logs a formatted debug message
func (l *Logger) Debugf(format string, args ...interface{}) {
	l.log(LogLevelDebug, fmt.Sprintf(format, args...), nil)
}

// Info logs an info message
func (l *Logger) Info(message string) {
	l.log(LogLevelInfo, message, nil)
}

// Infof logs a formatted info message
func (l *Logger) Infof(format string, args ...interface{}) {
	l.log(LogLevelInfo, fmt.Sprintf(format, args...), nil)
}

// Warn logs a warning message
func (l *Logger) Warn(message string) {
	l.log(LogLevelWarn, message, nil)
}

// Warnf logs a formatted warning message
func (l *Logger) Warnf(format string, args ...interface{}) {
	l.log(LogLevelWarn, fmt.Sprintf(format, args...), nil)
}

// Error logs an error message
func (l *Logger) Error(message string, err error) {
	l.log(LogLevelError, message, err)
}

// Errorf logs a formatted error message
func (l *Logger) Errorf(format string, err error, args ...interface{}) {
	l.log(LogLevelError, fmt.Sprintf(format, args...), err)
}

// Fatal logs a fatal message and exits
func (l *Logger) Fatal(message string, err error) {
	l.log(LogLevelFatal, message, err)
	os.Exit(1)
}

// Global logger functions for convenience
func LogDebug(message string) {
	if logger != nil {
		logger.Debug(message)
	}
}

func LogDebugf(format string, args ...interface{}) {
	if logger != nil {
		logger.Debugf(format, args...)
	}
}

func LogInfo(message string) {
	if logger != nil {
		logger.Info(message)
	}
}

func LogInfof(format string, args ...interface{}) {
	if logger != nil {
		logger.Infof(format, args...)
	}
}

func LogWarn(message string) {
	if logger != nil {
		logger.Warn(message)
	}
}

func LogWarnf(format string, args ...interface{}) {
	if logger != nil {
		logger.Warnf(format, args...)
	}
}

func LogError(message string, err error) {
	if logger != nil {
		logger.Error(message, err)
	}
}

func LogErrorf(format string, err error, args ...interface{}) {
	if logger != nil {
		logger.Errorf(format, err, args...)
	}
}

func LogFatal(message string, err error) {
	if logger != nil {
		logger.Fatal(message, err)
	}
}

func LogRequest(r *http.Request) *Logger {
	if logger != nil {
		return logger.WithRequest(r)
	}
	return logger
}

// LoggingMiddleware logs HTTP requests
func LoggingMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create response writer wrapper to capture status code
		wrapped := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// Log request start
		LogRequest(r).Debug("Request started")

		// Call the handler
		next(wrapped, r)

		// Log request completion
		duration := time.Since(start).Milliseconds()
		LogRequest(r).WithFields(map[string]interface{}{
			"status_code": wrapped.statusCode,
			"duration_ms": duration,
		}).Info("Request completed")
	}
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
