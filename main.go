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
	"sync"
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

// Cleanup management
var cleanupOnce sync.Once

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
	
	// Initialize session cleanup
	cleanupOnce.Do(func() {
		go periodicSessionCleanup()
	})
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

// Better bus ID abbreviation system that avoids collisions
func abbreviateBusID(busID string) string {
	// Define abbreviations for known long BusIDs
	abbreviations := map[string]string{
		"BUSLA GRANDE":          "BUSLG",
		"BUSMAIN OFFICE":        "BUSMAIN",
		"BUSUMATILLA":          "BUSUMAT",
		"BUSMILTON FREEWATER":   "BUSMF",
		"BUSVICTORY SQ":        "BUSVS", 
		"BUSENTERPRISE":        "BUSENT",
		"BUSWIC HERMISTON":     "BUSWH",
		"BUSBOARDMAN":          "BUSBOARD",
		"BUSPINE TREE":         "BUSPT",
		"BUSSLATED FOR MILTON": "BUSSFM",
		"BUSVICTORY 1":         "BUSV1",
		"BUSVICTORY 2":         "BUSV2",
		"BUSsub for victory 2": "BUSSUBV2",
		"BUSROCKY HTS.":        "BUSRH",
		"BUSPENDLETON":         "BUSPEND",
		"BUSMO/ JOHN DAY":      "BUSMJD",
		"BUSAWOC-2":            "BUSAWOC2",
	}
	
	// Check if we have a predefined abbreviation
	if shortened, exists := abbreviations[busID]; exists {
		return shortened
	}
	
	// If still too long, generate a unique short ID
	if len(busID) > 10 {
		// Use first 7 chars + 3-digit hash to avoid collisions
		hash := hashString(busID) % 1000
		return fmt.Sprintf("%s%03d", busID[:min(7, len(busID))], hash)
	}
	
	return busID
}

func hashString(s string) uint32 {
	h := uint32(0)
	for _, c := range s {
		h = h*31 + uint32(c)
	}
	return h
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
		Handler:        handler,  // Use the chained handler
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
	mux.HandleFunc("/new-user", withRecovery(requireAuth(requireRole("manager")(requireDatabase(newUserHandler)))))
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

// ============= AUTHENTICATION HANDLER =============

func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		handleLoginGet(w, r)
		return
	}
	
	handleLoginPost(w, r)
}

func handleLoginGet(w http.ResponseWriter, r *http.Request) {
	// Check if already logged in
	if user := getUserFromSession(r); user != nil {
		redirectToDashboard(w, r, user.Role)
		return
	}
	
	csrfToken, _ := GenerateSecureToken()
	renderTemplate(w, r, "login.html", LoginFormData{CSRFToken: csrfToken})
}

func handleLoginPost(w http.ResponseWriter, r *http.Request) {
	username := SanitizeFormValue(r, "username")
	password := r.FormValue("password")

	// Validate input
	if !ValidateUsername(username) {
		renderLoginError(w, r, "Invalid username format")
		return
	}

	// Authenticate user
	user, err := authenticateUser(username, password)
	if err != nil {
		renderLoginError(w, r, "Invalid username or password")
		return
	}

	// Check if pending
	if user.Status == StatusPending {
		renderLoginError(w, r, "Your account is pending approval. Please wait for a manager to approve your registration.")
		return
	}

	// Create session
	sessionID := generateSessionID()
	mu.Lock()
	sessions[sessionID] = &Session{
		Username: user.Username,
		Role:     user.Role,
		Expires:  time.Now().Add(24 * time.Hour),
	}
	mu.Unlock()
	
	SetSecureCookie(w, SessionCookieName, sessionID)
	redirectToDashboard(w, r, user.Role)
}

// ============= USER MANAGEMENT HANDLER =============

func newUserHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		data := UserFormData{CSRFToken: getCSRFToken(r)}
		renderTemplate(w, r, "new_user.html", data)
		return
	}

	// Handle POST
	if !validateCSRF(r) {
		http.Error(w, "Invalid CSRF token", http.StatusForbidden)
		return
	}
	
	username := SanitizeFormValue(r, "username")
	password := r.FormValue("password")
	role := SanitizeFormValue(r, "role")
	
	if err := validateNewUser(username, password, role); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	if err := createUser(username, password, role); err != nil {
		http.Error(w, "Failed to create user", http.StatusInternalServerError)
		return
	}
	
	// Clear cache after user creation
	cache.InvalidateUsers()
	
	http.Redirect(w, r, "/manager-dashboard", http.StatusFound)
}

// ============= UTILITY HANDLERS =============

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

func redirectToDashboard(w http.ResponseWriter, r *http.Request, role string) {
	path := "/driver-dashboard"
	if role == RoleManager {
		path = "/manager-dashboard"
	}
	http.Redirect(w, r, path, http.StatusFound)
}

func getCSRFToken(r *http.Request) string {
	cookie, _ := r.Cookie(SessionCookieName)
	if cookie != nil {
		mu.RLock()
		if session, exists := sessions[cookie.Value]; exists {
			mu.RUnlock()
			return generateCSRFToken()
		}
		mu.RUnlock()
	}
	return ""
}

func validateCSRF(r *http.Request) bool {
	// For now, just check if token is present
	// In production, you'd validate against session-stored token
	token := r.FormValue("csrf_token")
	return token != ""
}

func getUserFromSession(r *http.Request) *User {
	cookie, err := r.Cookie(SessionCookieName)
	if err != nil {
		return nil
	}

	mu.RLock()
	session, exists := sessions[cookie.Value]
	mu.RUnlock()

	if !exists || time.Now().After(session.Expires) {
		return nil
	}

	return &User{
		Username: session.Username,
		Role:     session.Role,
		Status:   StatusActive,
	}
}

func renderLoginError(w http.ResponseWriter, r *http.Request, errorMsg string) {
	csrfToken, _ := GenerateSecureToken()
	data := LoginFormData{
		Error:     errorMsg,
		CSRFToken: csrfToken,
	}
	renderTemplate(w, r, "login.html", data)
}

func extractIDFromPath(path, prefix string) string {
	if len(path) <= len(prefix) {
		return ""
	}
	return path[len(prefix):]
}

func sendJSONResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(data)
}

func getDateAndPeriod(r *http.Request) (string, string) {
	date := r.URL.Query().Get("date")
	if date == "" {
		date = time.Now().Format(DateFormat)
	}
	
	period := r.URL.Query().Get("period")
	if period == "" {
		if time.Now().Hour() < 12 {
			period = "morning"
		} else {
			period = "afternoon"
		}
	}
	
	return date, period
}

func validateNewUser(username, password, role string) error {
	if !ValidateUsername(username) {
		return fmt.Errorf("Invalid username format")
	}
	
	if len(password) < MinPasswordLength {
		return fmt.Errorf("Password must be at least %d characters", MinPasswordLength)
	}
	
	if role != RoleDriver && role != RoleManager {
		return fmt.Errorf("Invalid role")
	}
	
	return nil
}

func createUser(username, password, role string) error {
	hashedPassword, err := HashPassword(password)
	if err != nil {
		return err
	}
	
	newUser := User{
		Username: username,
		Password: hashedPassword,
		Role:     role,
		Status:   StatusActive,
	}
	
	return saveUser(newUser)
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

// GetActiveSessionCount returns the number of active sessions
func GetActiveSessionCount() int {
	mu.RLock()
	defer mu.RUnlock()
	
	count := 0
	now := time.Now()
	for _, session := range sessions {
		if session != nil && now.Before(session.Expires) {
			count++
		}
	}
	return count
}

// Helper functions
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Development mode check
func isDevelopment() bool {
	return os.Getenv("APP_ENV") == "development"
}

// periodicSessionCleanup runs periodically to clean expired sessions
func periodicSessionCleanup() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	
	for range ticker.C {
		mu.Lock()
		now := time.Now()
		for id, session := range sessions {
			if session != nil && now.After(session.Expires) {
				delete(sessions, id)
			}
		}
		mu.Unlock()
	}
}
