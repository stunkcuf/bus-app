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

// Global cache for performance
var cache = &DataCache{
	ttl: 5 * time.Minute,
}

// Templates variable
var templates *template.Template

func init() {
	funcMap := template.FuncMap{
		"json": jsonMarshal,
		"add":  func(a, b int) int { return a + b },
		"len":  getLength,
		"printf": fmt.Sprintf,
	}

	var err error
	templates, err = template.New("").Funcs(funcMap).ParseGlob(TemplateGlob)
	if err != nil {
		log.Fatalf("Failed to load templates: %v", err)
	}
	
	// Start session cleanup in security.go
	go periodicSessionCleanup()
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

func getLength(v interface{}) int {
	switch s := v.(type) {
	case []interface{}:
		return len(s)
	case []*Bus:
		return len(s)
	case []Bus:
		return len(s)
	default:
		return 0
	}
}

func main() {
	// Database setup
	log.Println("🗄️  Setting up PostgreSQL database...")
	if err := setupDatabase(); err != nil {
		log.Fatalf("Failed to setup database: %v", err)
	}
	defer closeDatabase()

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
	
	// Common protected routes
	mux.HandleFunc("/dashboard", withRecovery(requireAuth(requireDatabase(dashboardHandler))))

	return mux
}

// setupManagerRoutes configures manager-specific routes
func setupManagerRoutes(mux *http.ServeMux) {
	// User management
	mux.HandleFunc("/approve-users", withRecovery(requireAuth(requireRole("manager")(requireDatabase(approveUsersHandler)))))
	mux.HandleFunc("/approve-user", withRecovery(requireAuth(requireRole("manager")(requireDatabase(approveUserHandler)))))
	mux.HandleFunc("/manage-users", withRecovery(requireAuth(requireRole("manager")(requireDatabase(manageUsersHandler)))))
	mux.HandleFunc("/edit-user", withRecovery(requireAuth(requireRole("manager")(requireDatabase(editUserHandler)))))
	mux.HandleFunc("/delete-user", withRecovery(requireAuth(requireRole("manager")(requireDatabase(deleteUserHandler)))))
	
	// Dashboard
	mux.HandleFunc("/manager-dashboard", withRecovery(requireAuth(requireRole("manager")(requireDatabase(dashboardHandler)))))
	
	// Fleet management
	mux.HandleFunc("/fleet", withRecovery(requireAuth(requireRole("manager")(requireDatabase(fleetHandler)))))
	mux.HandleFunc("/company-fleet", withRecovery(requireAuth(requireRole("manager")(requireDatabase(companyFleetHandler)))))
	mux.HandleFunc("/update-vehicle-status", withRecovery(requireAuth(requireRole("manager")(requireDatabase(updateVehicleStatusHandler)))))
	
	// Maintenance
	mux.HandleFunc("/bus-maintenance/", withRecovery(requireAuth(requireRole("manager")(requireDatabase(busMaintenanceHandler)))))
	mux.HandleFunc("/vehicle-maintenance/", withRecovery(requireAuth(requireRole("manager")(requireDatabase(vehicleMaintenanceHandler)))))
	mux.HandleFunc("/save-maintenance-record", withRecovery(requireAuth(requireRole("manager")(requireDatabase(saveMaintenanceRecordHandler)))))
	
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

func getCSRFToken(r *http.Request) string {
	cookie, _ := r.Cookie(SessionCookieName)
	if cookie != nil {
		if session, _ := GetSecureSession(cookie.Value); session != nil {
			return session.CSRFToken
		}
	}
	return ""
}

func validateCSRF(r *http.Request) bool {
	cookie, _ := r.Cookie(SessionCookieName)
	return cookie != nil && ValidateCSRFToken(cookie.Value, r.FormValue("csrf_token"))
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
