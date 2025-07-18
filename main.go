package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

// Constants for better maintainability
const (
	DefaultPort         = "5000"
	SessionCookieName   = "session_id"
	CSRFTokenHeader     = "X-CSRF-Token"
	TemplateGlob        = "templates/*.html"
	
	// Timeouts
	ReadTimeout    = 30 * time.Second
	WriteTimeout   = 60 * time.Second
	IdleTimeout    = 120 * time.Second
	MaxHeaderBytes = 1 << 20
	
	// Roles
	RoleManager       = "manager"
	RoleDriver        = "driver"
	RoleDriverPending = "driver_pending"
	
	// Status
	StatusActive  = "active"
	StatusPending = "pending"
	
	// Date format
	DateFormat = "2006-01-02"
	
	// Minimum password length
	MinPasswordLength = 6
)


// Templates variable
var templates *template.Template

func init() {
	// Complete function map with all required functions including mul
	funcMap := template.FuncMap{
		// JSON marshaling
		"json": jsonMarshal,
		
		// ADDED: seq function for generating number sequences (needed for year dropdowns)
		"seq": func(start, end int) []int {
			var result []int
			for i := start; i <= end; i++ {
				result = append(result, i)
			}
			return result
		},
		
		// ADDED: formatNumber for formatting numbers with commas
		"formatNumber": func(n interface{}) string {
			var num int
			switch v := n.(type) {
			case int:
				num = v
			case int64:
				num = int(v)
			case float64:
				num = int(v)
			default:
				return fmt.Sprintf("%v", n)
			}
			
			// Format with commas
			str := fmt.Sprintf("%d", num)
			if num < 0 {
				return str // Handle negative numbers simply for now
			}
			if num < 1000 {
				return str
			}
			
			// Add commas every 3 digits from the right
			var result []string
			for i := len(str); i > 0; i -= 3 {
				start := i - 3
				if start < 0 {
					start = 0
				}
				result = append([]string{str[start:i]}, result...)
			}
			return strings.Join(result, ",")
		},
		
		// ADDED: multiply function (handles interface{} types)
		"multiply": func(a, b interface{}) float64 {
			return toFloat64(a) * toFloat64(b)
		},
		
		// Mathematical operations
		"add": func(a, b interface{}) float64 {
			return toFloat64(a) + toFloat64(b)
		},
		"sub": func(a, b interface{}) float64 {
			return toFloat64(a) - toFloat64(b)
		},
		"mul": func(a, b interface{}) float64 {
			return toFloat64(a) * toFloat64(b)
		},
		"div": func(a, b interface{}) float64 {
			bVal := toFloat64(b)
			if bVal == 0 {
				return 0
			}
			return toFloat64(a) / bVal
		},
		"mod": func(a, b interface{}) int {
			return toInt(a) % toInt(b)
		},
		
		// Comparison functions
		"eq": func(a, b interface{}) bool {
			return a == b
		},
		"ne": func(a, b interface{}) bool {
			return a != b
		},
		"lt": func(a, b interface{}) bool {
			return toFloat64(a) < toFloat64(b)
		},
		"le": func(a, b interface{}) bool {
			return toFloat64(a) <= toFloat64(b)
		},
		"gt": func(a, b interface{}) bool {
			return toFloat64(a) > toFloat64(b)
		},
		"ge": func(a, b interface{}) bool {
			return toFloat64(a) >= toFloat64(b)
		},
		
		// Utility functions
		"len":    getLength,
		"printf": fmt.Sprintf,
		
		// Number formatting
		"formatFloat": func(f interface{}, decimals int) string {
			format := fmt.Sprintf("%%.%df", decimals)
			return fmt.Sprintf(format, toFloat64(f))
		},
		"formatCurrency": func(amount interface{}) string {
			// UPDATED: Better currency formatting with commas
			f := toFloat64(amount)
			
			// Format with 2 decimal places
			str := fmt.Sprintf("%.2f", f)
			parts := strings.Split(str, ".")
			
			// Handle the integer part
			intPart := parts[0]
			negative := false
			if strings.HasPrefix(intPart, "-") {
				negative = true
				intPart = intPart[1:]
			}
			
			// Add commas to integer part
			var result []string
			for i := len(intPart); i > 0; i -= 3 {
				start := i - 3
				if start < 0 {
					start = 0
				}
				result = append([]string{intPart[start:i]}, result...)
			}
			
			formattedInt := strings.Join(result, ",")
			if negative {
				formattedInt = "-" + formattedInt
			}
			
			// Combine with decimal part
			if len(parts) > 1 {
				return formattedInt + "." + parts[1]
			}
			return formattedInt + ".00"
		},
		"formatPercent": func(value interface{}) string {
			return fmt.Sprintf("%.0f%%", toFloat64(value))
		},
		
		// Date formatting
		"formatDate": func(date string) string {
			t, err := time.Parse("2006-01-02", date)
			if err != nil {
				return date
			}
			return t.Format("Jan 2, 2006")
		},
		
		// String functions
		"hasPrefix": func(s, prefix string) bool {
			return len(s) >= len(prefix) && s[:len(prefix)] == prefix
		},
		"hasSuffix": func(s, suffix string) bool {
			return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
		},
		"title": func(s string) string {
			// Simple title case implementation
			words := strings.Fields(s)
			for i, word := range words {
				if len(word) > 0 {
					words[i] = strings.ToUpper(string(word[0])) + strings.ToLower(word[1:])
				}
			}
			return strings.Join(words, " ")
		},
		"formatBytes": func(bytes int64) string {
			const unit = 1024
			if bytes < unit {
				return fmt.Sprintf("%d B", bytes)
			}
			div, exp := int64(unit), 0
			for n := bytes / unit; n >= unit; n /= unit {
				div *= unit
				exp++
			}
			return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
		},
	}

	var err error
	log.Printf("Loading templates from: %s", TemplateGlob)
	templates, err = template.New("").Funcs(funcMap).ParseGlob(TemplateGlob)
	if err != nil {
		log.Fatalf("Failed to load templates: %v", err)
	}
	log.Printf("Successfully loaded %d templates", len(templates.Templates()))
	
	// Start session cleanup in security.go
	go periodicSessionCleanup()
}

// Helper function to convert interface{} to float64
func toFloat64(v interface{}) float64 {
	switch x := v.(type) {
	case int:
		return float64(x)
	case int32:
		return float64(x)
	case int64:
		return float64(x)
	case uint:
		return float64(x)
	case uint32:
		return float64(x)
	case uint64:
		return float64(x)
	case float32:
		return float64(x)
	case float64:
		return x
	case string:
		// Handle percentage strings like "75%" 
		if len(x) > 0 && x[len(x)-1] == '%' {
			var val float64
			if _, err := fmt.Sscanf(x[:len(x)-1], "%f", &val); err == nil {
				return val
			}
		}
		// Try to parse as float
		var val float64
		if _, err := fmt.Sscanf(x, "%f", &val); err == nil {
			return val
		}
		return 0
	default:
		return 0
	}
}

// Helper function to convert interface{} to int
func toInt(v interface{}) int {
	switch x := v.(type) {
	case int:
		return x
	case int32:
		return int(x)
	case int64:
		return int(x)
	case uint:
		return int(x)
	case uint32:
		return int(x)
	case uint64:
		return int(x)
	case float32:
		return int(x)
	case float64:
		return int(x)
	default:
		return 0
	}
}

// Template helper functions
func jsonMarshal(v interface{}) template.JS {
	b, err := json.Marshal(v)
	if err != nil {
		log.Printf("JSON marshal error: %v", err)
		return template.JS("{}")
	}
	return template.JS(b)
}

// Replace the getLength function in main.go with this updated version

func getLength(v interface{}) int {
	switch s := v.(type) {
	case []interface{}:
		return len(s)
	case []Bus:
		return len(s)
	case []User:
		return len(s)
	case []Student:
		return len(s)
	case []Route:
		return len(s)
	case []Vehicle:
		return len(s)
	case []ECSEStudentView:
		return len(s)
	case []ECSEStudent:
		return len(s)
	case []MaintenanceAlert:
		return len(s)
	case []CombinedMaintenanceLog:
		return len(s)
	case []MileageReport:
		return len(s)
	case []RouteAssignment:
		return len(s)
	case map[string]interface{}:
		return len(s)
	case map[string]int:
		return len(s)
	case map[string][]MaintenanceAlert:
		return len(s)
	case string:
		return len(s)
	default:
		return 0
	}
}
func main() {
	// Initialize logger
	InitLogger()
	
	// Database setup
	LogInfo("🗄️  Setting up PostgreSQL database...")
	if err := setupDatabase(); err != nil {
		LogFatal("Failed to setup database", err)
	}
	defer closeDatabase()
	
	// Reset rate limiter on startup to clear any previous blocks
	LogInfo("🔄 Resetting rate limiter...")
	rateLimiter.Reset()

	mux := setupRoutes()
	
	// Chain middlewares: CSP -> Security -> Router
	handler := CSPMiddleware(SecurityHeaders(mux))
	
	port := os.Getenv("PORT")
	if port == "" {
		port = DefaultPort
	}

	server := &http.Server{
		Addr:           fmt.Sprintf(":%s", port),
		Handler:        handler,
		ReadTimeout:    ReadTimeout,
		WriteTimeout:   WriteTimeout,
		IdleTimeout:    IdleTimeout,
		MaxHeaderBytes: MaxHeaderBytes,
	}

	// Start background jobs
	startScheduledExportsJob()
	
	// Graceful shutdown
	go gracefulShutdown(server)

	log.Printf("🚀 Server starting on port %s", port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}
}

// setupRoutes configures all application routes
func setupRoutes() *http.ServeMux {
	mux := http.NewServeMux()
	
	// Public routes
	mux.HandleFunc("/", withRecovery(RateLimitMiddleware(loginHandler)))
	mux.HandleFunc("/register", withRecovery(RateLimitMiddleware(registerHandler)))
	mux.HandleFunc("/logout", withRecovery(logoutHandler))
	mux.HandleFunc("/health", withRecovery(healthCheck))

	// Manager-only routes
	setupManagerRoutes(mux)

	// Driver routes
	setupDriverRoutes(mux)
	
	// API routes (accessible by both managers and drivers)
	setupAPIRoutes(mux)
	
	// Common protected routes
	mux.HandleFunc("/dashboard", withRecovery(requireAuth(requireDatabase(dashboardHandler))))

	return mux
}

// setupAPIRoutes configures API endpoints
func setupAPIRoutes(mux *http.ServeMux) {
	// Maintenance API routes
	mux.HandleFunc("/api/check-maintenance", withRecovery(requireAuth(requireDatabase(checkMaintenanceDueHandler))))
	mux.HandleFunc("/api/debug-maintenance", withRecovery(requireAuth(requireRole("manager")(requireDatabase(debugMaintenanceRecordsHandler)))))
	
	// Dashboard Analytics API routes
	mux.HandleFunc("/api/dashboard/analytics", withRecovery(requireAuth(requireRole("manager")(requireDatabase(dashboardAnalyticsHandler)))))
	mux.HandleFunc("/api/dashboard/fleet-status", withRecovery(requireAuth(requireDatabase(fleetStatusWidgetHandler))))
	mux.HandleFunc("/api/dashboard/maintenance-alerts", withRecovery(requireAuth(requireDatabase(maintenanceAlertsWidgetHandler))))
	mux.HandleFunc("/api/dashboard/route-efficiency", withRecovery(requireAuth(requireDatabase(routeEfficiencyWidgetHandler))))
	
	// Report Builder API routes
	mux.HandleFunc("/api/report-builder", withRecovery(requireAuth(requireRole("manager")(requireDatabase(reportBuilderAPIHandler)))))
	mux.HandleFunc("/api/report-data-sources", withRecovery(requireAuth(requireRole("manager")(requireDatabase(getReportDataSourcesHandler)))))
	mux.HandleFunc("/api/report-chart-types", withRecovery(requireAuth(requireRole("manager")(requireDatabase(reportChartTypesHandler)))))
}

// setupManagerRoutes configures manager-specific routes
func setupManagerRoutes(mux *http.ServeMux) {
	// User management
	mux.HandleFunc("/approve-users", withRecovery(requireAuth(requireRole("manager")(requireDatabase(approveUsersHandler)))))
	mux.HandleFunc("/approve-user", withRecovery(requireAuth(requireRole("manager")(requireDatabase(approveUserHandler)))))
	mux.HandleFunc("/manage-users", withRecovery(requireAuth(requireRole("manager")(requireDatabase(manageUsersHandler)))))
	mux.HandleFunc("/edit-user", withRecovery(requireAuth(requireRole("manager")(requireDatabase(editUserHandler)))))
	mux.HandleFunc("/delete-user", withRecovery(requireAuth(requireRole("manager")(requireDatabase(deleteUserHandler)))))
	
	// ECSE Management Routes
	mux.HandleFunc("/import-ecse", withRecovery(requireAuth(requireRole("manager")(requireDatabase(importECSEHandler)))))
	mux.HandleFunc("/view-ecse-reports", withRecovery(requireAuth(requireRole("manager")(requireDatabase(viewECSEReportsHandler)))))
	mux.HandleFunc("/ecse-student/", withRecovery(requireAuth(requireRole("manager")(requireDatabase(viewECSEStudentHandler)))))
	mux.HandleFunc("/edit-ecse-student", withRecovery(requireAuth(requireRole("manager")(requireDatabase(editECSEStudentHandler)))))
	mux.HandleFunc("/export-ecse", withRecovery(requireAuth(requireRole("manager")(requireDatabase(exportECSEHandler)))))
	
	// Dashboard
	mux.HandleFunc("/manager-dashboard", withRecovery(requireAuth(requireRole("manager")(requireDatabase(managerDashboardHandler)))))
	mux.HandleFunc("/analytics-dashboard", withRecovery(requireAuth(requireRole("manager")(requireDatabase(analyticsDashboardHandler)))))
	mux.HandleFunc("/report-builder", withRecovery(requireAuth(requireRole("manager")(requireDatabase(reportBuilderHandler)))))
	
	// Fleet management - Available to both managers and drivers with proper permissions
	mux.HandleFunc("/fleet", withRecovery(requireAuth(requireDatabase(fleetHandler))))
	mux.HandleFunc("/company-fleet", withRecovery(requireAuth(requireDatabase(companyFleetHandler))))
	mux.HandleFunc("/update-vehicle-status", withRecovery(requireAuth(requireDatabase(updateVehicleStatusHandler))))
	mux.HandleFunc("/add-bus", withRecovery(requireAuth(requireRole("manager")(requireDatabase(addBusHandler)))))
	
	// Maintenance - Available to both managers and drivers
	mux.HandleFunc("/bus-maintenance/", withRecovery(requireAuth(requireDatabase(busMaintenanceHandler))))
	mux.HandleFunc("/vehicle-maintenance/", withRecovery(requireAuth(requireDatabase(vehicleMaintenanceHandler))))
	mux.HandleFunc("/save-maintenance-record", withRecovery(requireAuth(requireDatabase(saveMaintenanceRecordHandler))))
	
	// Route management
	mux.HandleFunc("/assign-routes", withRecovery(requireAuth(requireRole("manager")(requireDatabase(assignRoutesHandler)))))
	mux.HandleFunc("/assign-route", withRecovery(requireAuth(requireRole("manager")(requireDatabase(assignRouteHandler)))))
	mux.HandleFunc("/unassign-route", withRecovery(requireAuth(requireRole("manager")(requireDatabase(unassignRouteHandler)))))
	mux.HandleFunc("/add-route", withRecovery(requireAuth(requireRole("manager")(requireDatabase(addRouteHandler)))))
	mux.HandleFunc("/edit-route", withRecovery(requireAuth(requireRole("manager")(requireDatabase(editRouteHandler)))))
	mux.HandleFunc("/delete-route", withRecovery(requireAuth(requireRole("manager")(requireDatabase(deleteRouteHandler)))))
	
	// Mileage reports
	mux.HandleFunc("/import-mileage", withRecovery(requireAuth(requireRole("manager")(requireDatabase(importMileageHandler)))))
	mux.HandleFunc("/view-mileage-reports", withRecovery(requireAuth(requireRole("manager")(requireDatabase(viewMileageReportsHandler)))))
	mux.HandleFunc("/export-mileage", withRecovery(requireAuth(requireRole("manager")(requireDatabase(exportMileageHandler)))))
	mux.HandleFunc("/mileage-report-generator", withRecovery(requireAuth(requireRole("manager")(requireDatabase(mileageReportGeneratorHandler)))))
	
	// Enhanced Import System (temporarily disabled)
	// mux.HandleFunc("/import", withRecovery(requireAuth(requireRole("manager")(requireDatabase(importHandler)))))
	// mux.HandleFunc("/import/mapping", withRecovery(requireAuth(requireRole("manager")(requireDatabase(importMappingHandler)))))
	// mux.HandleFunc("/import/preview", withRecovery(requireAuth(requireRole("manager")(requireDatabase(importPreviewHandler)))))
	// mux.HandleFunc("/import/execute", withRecovery(requireAuth(requireRole("manager")(requireDatabase(importExecuteHandler)))))
	// mux.HandleFunc("/import/history", withRecovery(requireAuth(requireRole("manager")(requireDatabase(importHistoryHandler)))))
	// mux.HandleFunc("/import/details", withRecovery(requireAuth(requireRole("manager")(requireDatabase(importDetailsHandler)))))
	// mux.HandleFunc("/import/rollback", withRecovery(requireAuth(requireRole("manager")(requireDatabase(importRollbackHandler)))))
	// mux.HandleFunc("/api/import", withRecovery(requireAuth(requireRole("manager")(requireDatabase(importAPIHandler)))))
	// mux.HandleFunc("/api/import/auto-map", withRecovery(requireAuth(requireRole("manager")(requireDatabase(autoMapHandler)))))
	
	// Export System
	mux.HandleFunc("/export/templates", withRecovery(requireAuth(requireRole("manager")(requireDatabase(exportTemplateHandler)))))
	mux.HandleFunc("/export/template", withRecovery(requireAuth(requireRole("manager")(requireDatabase(exportTemplateHandler)))))
	mux.HandleFunc("/export/data", withRecovery(requireAuth(requireRole("manager")(requireDatabase(exportDataHandler)))))
	mux.HandleFunc("/export/scheduled", withRecovery(requireAuth(requireRole("manager")(requireDatabase(scheduledExportsHandler)))))
	mux.HandleFunc("/export/scheduled/edit", withRecovery(requireAuth(requireRole("manager")(requireDatabase(scheduledExportEditHandler)))))
	mux.HandleFunc("/export/scheduled/delete", withRecovery(requireAuth(requireRole("manager")(requireDatabase(scheduledExportDeleteHandler)))))
	mux.HandleFunc("/export/scheduled/run", withRecovery(requireAuth(requireRole("manager")(requireDatabase(scheduledExportRunHandler)))))
	
	// PDF Reports
	mux.HandleFunc("/api/reports/pdf", withRecovery(requireAuth(requireDatabase(pdfReportHandler))))
	mux.HandleFunc("/api/reports/pdf/custom", withRecovery(requireAuth(requireRole("manager")(requireDatabase(pdfCustomReportHandler)))))
	
	// Driver profile
	mux.HandleFunc("/driver/", withRecovery(requireAuth(requireRole("manager")(requireDatabase(driverProfileHandler)))))
}

// setupDriverRoutes configures driver-specific routes
func setupDriverRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/driver-dashboard", withRecovery(requireAuth(requireRole("driver")(requireDatabase(driverDashboardHandler)))))
	mux.HandleFunc("/save-log", withRecovery(requireAuth(requireRole("driver")(requireDatabase(saveLogHandler)))))
	
	// Student management
	mux.HandleFunc("/students", withRecovery(requireAuth(requireRole("driver")(requireDatabase(studentsHandler)))))
	mux.HandleFunc("/add-student", withRecovery(requireAuth(requireRole("driver")(requireDatabase(addStudentHandler)))))
	mux.HandleFunc("/edit-student", withRecovery(requireAuth(requireRole("driver")(requireDatabase(editStudentHandler)))))
	mux.HandleFunc("/remove-student", withRecovery(requireAuth(requireRole("driver")(requireDatabase(removeStudentHandler)))))
}

// ============= UTILITY HANDLER =============

func healthCheck(w http.ResponseWriter, r *http.Request) {
	health := struct {
		Status      string `json:"status"`
		Service     string `json:"service"`
		Timestamp   string `json:"timestamp"`
		Database    string `json:"database"`
		Version     string `json:"version"`
		SessionCount int   `json:"active_sessions"`
	}{
		Status:      "ok",
		Service:     "bus-fleet-management",
		Timestamp:   time.Now().Format(time.RFC3339),
		Database:    "connected",
		Version:     "2.0.0",
		SessionCount: GetActiveSessionCount(),
	}
	
	// Check database connection
	if db == nil || db.Ping() != nil {
		health.Status = "degraded"
		health.Database = "disconnected"
	}
	
	w.Header().Set("Content-Type", "application/json")
	
	if health.Status == "ok" {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
	}
	
	json.NewEncoder(w).Encode(health)
}

// ============= HELPER FUNCTIONS =============

// getSessionCSRFToken gets the CSRF token from the current session
func getSessionCSRFToken(r *http.Request) string {
	cookie, err := r.Cookie(SessionCookieName)
	if err != nil {
		return ""
	}
	
	session, err := GetSecureSession(cookie.Value)
	if err != nil {
		return ""
	}
	
	return session.CSRFToken
}

// validateCSRF validates the CSRF token - FIXED for multipart forms
func validateCSRF(r *http.Request) bool {
	// Get session cookie
	cookie, err := r.Cookie(SessionCookieName)
	if err != nil {
		// No session cookie = no authenticated user
		log.Printf("CSRF validation failed: no session cookie")
		return false
	}
	
	// Get session
	session, err := GetSecureSession(cookie.Value)
	if err != nil {
		// Invalid session = fail validation
		log.Printf("CSRF validation failed: invalid session")
		return false
	}
	
	// Get submitted token from form or header
	submittedToken := r.FormValue("csrf_token")
	if submittedToken == "" {
		// Also check header for AJAX requests
		submittedToken = r.Header.Get("X-CSRF-Token")
	}
	
	// Debug logging
	if submittedToken == "" {
		log.Printf("CSRF validation failed: no token submitted")
		return false
	}
	
	if session.CSRFToken != submittedToken {
		log.Printf("CSRF validation failed: token mismatch - expected: %s, got: %s", session.CSRFToken, submittedToken)
		return false
	}
	
	return true
}

// Graceful shutdown
func gracefulShutdown(server *http.Server) {
	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan
	
	log.Println("Shutting down server...")
	
	// Give connections 30 seconds to finish
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}
	
	log.Println("Server shutdown complete")
}

// Helper functions
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Development mode check
func isDevelopment() bool {
	return os.Getenv("APP_ENV") == "development"
}
