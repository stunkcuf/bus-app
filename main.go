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
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
	"mime/multipart"
	"regexp"
	"github.com/xuri/excelize/v2"
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
	mux.HandleFunc("/", withRecovery(RateLimitMiddleware(loginHandlerWithApproval)))
	mux.HandleFunc("/register", withRecovery(RateLimitMiddleware(registerHandler)))
	mux.HandleFunc("/logout", withRecovery(logout))
	mux.HandleFunc("/health", withRecovery(healthCheck))

	// Manager-only routes
	setupManagerRoutes(mux)

	// Driver routes
	setupDriverRoutes(mux)
	
	// Common protected routes
	mux.HandleFunc("/dashboard", withRecovery(requireAuth(requireDatabase(dashboardRouter))))

	return mux
}

// setupManagerRoutes configures manager-specific routes
func setupManagerRoutes(mux *http.ServeMux) {
	// User management
	mux.HandleFunc("/approve-users", withRecovery(requireAuth(requireRole("manager")(requireDatabase(approveUsersHandler)))))
	mux.HandleFunc("/approve-user", withRecovery(requireAuth(requireRole("manager")(requireDatabase(approveUserHandler)))))
	mux.HandleFunc("/new-user", withRecovery(requireAuth(requireRole("manager")(requireDatabase(newUserHandler)))))
	mux.HandleFunc("/edit-user", withRecovery(requireAuth(requireRole("manager")(requireDatabase(editUserHandler)))))
	mux.HandleFunc("/remove-user", withRecovery(requireAuth(requireRole("manager")(requireDatabase(removeUserHandler)))))
	
	// Dashboard
	mux.HandleFunc("/manager-dashboard", withRecovery(requireAuth(requireRole("manager")(requireDatabase(managerDashboard)))))
	
	// Fleet management
	mux.HandleFunc("/fleet", withRecovery(requireAuth(requireRole("manager")(requireDatabase(fleetPage)))))
	mux.HandleFunc("/company-fleet", withRecovery(requireAuth(requireRole("manager")(requireDatabase(companyFleetPage)))))
	mux.HandleFunc("/update-vehicle-status", withRecovery(requireAuth(requireRole("manager")(requireDatabase(updateVehicleStatus)))))
	
	// Maintenance
	mux.HandleFunc("/debug-vehicle/", withRecovery(requireAuth(requireRole("manager")(requireDatabase(debugVehicleHandler)))))
	mux.HandleFunc("/bus-maintenance/", withRecovery(requireAuth(requireRole("manager")(requireDatabase(busMaintenanceHandler)))))
	mux.HandleFunc("/vehicle-maintenance/", withRecovery(requireAuth(requireRole("manager")(requireDatabase(vehicleMaintenanceHandler)))))
	mux.HandleFunc("/save-maintenance-record", withRecovery(requireAuth(requireRole("manager")(requireDatabase(saveMaintenanceRecordHandler)))))
	
	// Route management
	mux.HandleFunc("/assign-routes", withRecovery(requireAuth(requireRole("manager")(requireDatabase(assignRoutesPage)))))
	mux.HandleFunc("/assign-route", withRecovery(requireAuth(requireRole("manager")(requireDatabase(assignRouteHandler)))))
	mux.HandleFunc("/unassign-route", withRecovery(requireAuth(requireRole("manager")(requireDatabase(unassignRouteHandler)))))
	mux.HandleFunc("/assign-routes/add", withRecovery(requireAuth(requireRole("manager")(requireDatabase(addRouteHandler)))))
	mux.HandleFunc("/assign-routes/edit", withRecovery(requireAuth(requireRole("manager")(requireDatabase(editRouteHandler)))))
	mux.HandleFunc("/assign-routes/delete", withRecovery(requireAuth(requireRole("manager")(requireDatabase(deleteRouteHandler)))))
	
	// API endpoints for route assignment
	mux.HandleFunc("/api/route-assignment", withRecovery(requireAuth(requireRole("manager")(requireDatabase(handleSaveRouteAssignment)))))
	mux.HandleFunc("/api/check-driver-bus", withRecovery(requireAuth(requireRole("manager")(requireDatabase(handleCheckDriverBus)))))
	
	// Mileage reports
	mux.HandleFunc("/import-mileage", withRecovery(requireAuth(requireRole("manager")(requireDatabase(importMileageHandler)))))
	mux.HandleFunc("/view-mileage-reports", withRecovery(requireAuth(requireRole("manager")(requireDatabase(viewEnhancedMileageReportsHandler)))))
	
	// Driver profile
	mux.HandleFunc("/driver/", withRecovery(requireAuth(requireRole("manager")(requireDatabase(driverProfileHandler)))))
}

// setupDriverRoutes configures driver-specific routes
func setupDriverRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/driver-dashboard", withRecovery(requireAuth(requireRole("driver")(requireDatabase(driverDashboard)))))
	mux.HandleFunc("/save-log", withRecovery(requireAuth(requireRole("driver")(requireDatabase(saveDriverLogHandler)))))
	
	// Student management
	mux.HandleFunc("/students", withRecovery(requireAuth(requireRole("driver")(requireDatabase(studentsPage)))))
	mux.HandleFunc("/add-student", withRecovery(requireAuth(requireRole("driver")(requireDatabase(addStudentHandler)))))
	mux.HandleFunc("/edit-student", withRecovery(requireAuth(requireRole("driver")(requireDatabase(editStudentHandler)))))
	mux.HandleFunc("/remove-student", withRecovery(requireAuth(requireRole("driver")(requireDatabase(removeStudentHandler)))))
}

// ============= AUTHENTICATION & REGISTRATION HANDLERS =============

func loginHandlerWithApproval(w http.ResponseWriter, r *http.Request) {
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

	// Find user and check credentials
	users, err := cache.GetUsers()
	if err != nil {
		log.Printf("Error loading users: %v", err)
		renderLoginError(w, r, "System error. Please try again.")
		return
	}
	
	for _, user := range users {
		if user.Username == username && CheckPasswordHash(password, user.Password) {
			if user.Role == RoleDriverPending {
				renderLoginError(w, r, "Your account is pending approval. Please wait for a manager to approve your registration.")
				return
			}

			// Create session
			sessionID, _, err := CreateSecureSession(username, user.Role)
			if err != nil {
				http.Error(w, "Session creation failed", http.StatusInternalServerError)
				return
			}
			
			SetSecureCookie(w, SessionCookieName, sessionID)
			redirectToDashboard(w, r, user.Role)
			return
		}
	}

	renderLoginError(w, r, "Invalid username or password")
}

// Legacy mileage reports handler (still needed for compatibility)
func viewMileageReportsHandler(w http.ResponseWriter, r *http.Request) {
	// Redirect to enhanced version
	http.Redirect(w, r, "/view-mileage-reports", http.StatusFound)
}

func registerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		renderTemplate(w, r, "register.html", struct{ Error string }{})
		return
	}

	// Handle POST
	username := SanitizeFormValue(r, "username")
	password := r.FormValue("password")

	// Validate input
	if err := validateRegistration(username, password); err != nil {
		renderTemplate(w, r, "register.html", struct{ Error string }{Error: err.Error()})
		return
	}

	// Check if username exists
	exists, err := userExists(username)
	if err != nil {
		log.Printf("Error checking user existence: %v", err)
		renderTemplate(w, r, "register.html", struct{ Error string }{
			Error: "System error. Please try again.",
		})
		return
	}
	
	if exists {
		renderTemplate(w, r, "register.html", struct{ Error string }{
			Error: "Username already exists. Please choose another.",
		})
		return
	}

	// Create pending user
	if err := createPendingUser(username, password); err != nil {
		renderTemplate(w, r, "register.html", struct{ Error string }{
			Error: "Failed to create account. Please try again.",
		})
		return
	}

	// Clear user cache
	cache.InvalidateUsers()
	
	renderTemplate(w, r, "registration_success.html", nil)
}
// Part 2 - Continuing from registerHandler...

func approveUsersHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != RoleManager {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	pendingUsers, err := getPendingUsers()
	if err != nil {
		log.Printf("Error getting pending users: %v", err)
		http.Error(w, "Failed to load pending users", http.StatusInternalServerError)
		return
	}
	
	csrfToken := getCSRFToken(r)

	data := struct {
		PendingUsers []struct {
			Username  string
			CreatedAt string
		}
		CSRFToken string
	}{
		PendingUsers: pendingUsers,
		CSRFToken:    csrfToken,
	}

	renderTemplate(w, r, "approve_users.html", data)
}

func approveUserHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Validate CSRF
	if !validateCSRF(r) {
		http.Error(w, "Invalid CSRF token", http.StatusForbidden)
		return
	}

	username := r.FormValue("username")
	action := r.FormValue("action")

	if err := validateApprovalRequest(username, action); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := processUserApproval(username, action); err != nil {
		http.Error(w, "Failed to process request", http.StatusInternalServerError)
		return
	}

	// Clear cache after user update
	cache.InvalidateUsers()
	
	http.Redirect(w, r, "/approve-users", http.StatusFound)
}

// ============= DASHBOARD HANDLERS =============

func dashboardRouter(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	
	redirectToDashboard(w, r, user.Role)
}

func managerDashboard(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != RoleManager {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	pendingCount, err := countPendingUsers()
	if err != nil {
		log.Printf("Error counting pending users: %v", err)
		pendingCount = 0
	}
	
	csrfToken := getCSRFToken(r)
	
	users, _ := cache.GetUsers()
	buses, _ := cache.GetBuses()
	routes, _ := cache.GetRoutes()
	
	data := DashboardData{
		User:            user,
		Role:            user.Role,
		Users:           users,
		Buses:           buses,
		Routes:          routes,
		DriverSummaries: []*DriverSummary{},
		RouteStats:      []*RouteStats{},
		Activities:      []Activity{},
		CSRFToken:       csrfToken,
		PendingUsers:    pendingCount,
	}
	
	renderTemplate(w, r, "dashboard.html", data)
}

func driverDashboard(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != RoleDriver {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	// Get parameters
	date, period := getDateAndPeriod(r)
	
	// Load driver's data
	assignment, _ := getDriverRouteAssignment(user.Username)
	route, bus := getRouteAndBus(assignment)
	routeStudents, err := getRouteStudents(route, user.Username, period)
	if err != nil {
		log.Printf("Error getting route students: %v", err)
		routeStudents = []Student{}
	}
	
	driverLog := getDriverLogForDatePeriod(user.Username, date, period)
	recentLogs, _ := getRecentDriverLogs(user.Username, 5)
	
	data := struct {
		User          *User
		Date          string
		Period        string
		Route         *Route
		Bus           *Bus
		Students      []Student
		DriverLog     *DriverLog
		RecentLogs    []DriverLog
		CSRFToken     string
	}{
		User:       user,
		Date:       date,
		Period:     period,
		Route:      route,
		Bus:        bus,
		Students:   routeStudents,
		DriverLog:  driverLog,
		RecentLogs: recentLogs,
		CSRFToken:  getCSRFToken(r),
	}
	
	renderTemplate(w, r, "driver_dashboard.html", data)
}

// ============= IMPORT MILEAGE HANDLER =============
func importMileageHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	
	if r.Method == "GET" {
		// Display the import form
		data := struct {
			User      *User
			CSRFToken string
			Error     string
			Success   string
		}{
			User:      user,
			CSRFToken: getCSRFToken(r),
		}
		
		renderTemplate(w, r, "import_mileage.html", data)
		return
	}
	
	// Handle POST - file upload
	if r.Method == "POST" {
		// Validate CSRF
		if !validateCSRF(r) {
			http.Error(w, "Invalid CSRF token", http.StatusForbidden)
			return
		}
		
		// Parse multipart form
		err := r.ParseMultipartForm(10 << 20) // 10 MB max
		if err != nil {
			data := struct {
				User      *User
				CSRFToken string
				Error     string
				Success   string
			}{
				User:      user,
				CSRFToken: getCSRFToken(r),
				Error:     "Failed to parse form data",
			}
			renderTemplate(w, r, "import_mileage.html", data)
			return
		}
		
		// Get the file
		file, header, err := r.FormFile("excel_file")
		if err != nil {
			data := struct {
				User      *User
				CSRFToken string
				Error     string
				Success   string
			}{
				User:      user,
				CSRFToken: getCSRFToken(r),
				Error:     "Failed to get uploaded file",
			}
			renderTemplate(w, r, "import_mileage.html", data)
			return
		}
		defer file.Close()
		
		// Log file info
		log.Printf("Uploaded File: %+v", header.Filename)
		log.Printf("File Size: %+v", header.Size)
		log.Printf("MIME Header: %+v", header.Header)
		
		// Process the Excel file
		importedCount, err := processMileageExcelFile(file, header.Filename)
		if err != nil {
			log.Printf("Error processing Excel file: %v", err)
			data := struct {
				User      *User
				CSRFToken string
				Error     string
				Success   string
			}{
				User:      user,
				CSRFToken: getCSRFToken(r),
				Error:     fmt.Sprintf("Failed to import file: %v", err),
			}
			renderTemplate(w, r, "import_mileage.html", data)
			return
		}
		
		// Success!
		data := struct {
			User      *User
			CSRFToken string
			Error     string
			Success   string
		}{
			User:      user,
			CSRFToken: getCSRFToken(r),
			Success:   fmt.Sprintf("Successfully imported %d records from '%s'! This includes agency vehicles, school buses, and program staff data.", importedCount, header.Filename),
		}
		
		renderTemplate(w, r, "import_mileage.html", data)
	}
}

// ============= USER MANAGEMENT HANDLERS =============

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

func editUserHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		handleEditUserGet(w, r)
		return
	}
	
	handleEditUserPost(w, r)
}

func removeUserHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	username := r.URL.Query().Get("username")
	
	if username == "" || username == user.Username {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	
	if err := deleteUser(username); err != nil {
		http.Error(w, "Failed to remove user", http.StatusInternalServerError)
		return
	}
	
	// Clear cache after user deletion
	cache.InvalidateUsers()
	
	http.Redirect(w, r, "/manager-dashboard", http.StatusFound)
}

// ============= DRIVER LOG HANDLER =============

func saveDriverLogHandler(w http.ResponseWriter, r *http.Request) {
	if !validateCSRF(r) {
		http.Error(w, "Invalid CSRF token", http.StatusForbidden)
		return
	}
	
	user := getUserFromSession(r)
	driverLog, err := parseDriverLog(r, user.Username)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	if err := saveDriverLog(driverLog); err != nil {
		http.Error(w, "Failed to save log", http.StatusInternalServerError)
		return
	}
	
	http.Redirect(w, r, fmt.Sprintf("/driver-dashboard?date=%s&period=%s", 
		driverLog.Date, driverLog.Period), http.StatusFound)
}

// ============= MAINTENANCE HANDLERS =============

func busMaintenanceHandler(w http.ResponseWriter, r *http.Request) {
	vehicleID := extractIDFromPath(r.URL.Path, "/bus-maintenance/")
	if vehicleID == "" {
		http.Error(w, "Bus ID required", http.StatusBadRequest)
		return
	}
	
	handleVehicleMaintenance(w, r, vehicleID, true)
}

func vehicleMaintenanceHandler(w http.ResponseWriter, r *http.Request) {
	// Extract vehicle ID from URL path
	path := r.URL.Path
	
	// Remove the prefix and any trailing slashes
	vehicleID := strings.TrimPrefix(path, "/vehicle-maintenance/")
	vehicleID = strings.TrimSuffix(vehicleID, "/")
	
	log.Printf("=== Vehicle Maintenance Handler ===")
	log.Printf("Full Path: %s", path)
	log.Printf("Extracted Vehicle ID: '%s'", vehicleID)
	
	if vehicleID == "" {
		log.Printf("ERROR: No vehicle ID in path")
		http.Error(w, "Vehicle ID required", http.StatusBadRequest)
		return
	}
	
	// Get vehicle info from vehicles table
	vehicles, err := cache.GetVehicles()
	if err != nil {
		log.Printf("Error loading vehicles: %v", err)
		http.Error(w, "Failed to load vehicles", http.StatusInternalServerError)
		return
	}
	
	var vehicle *Vehicle
	for i := range vehicles {
		if vehicles[i].VehicleID == vehicleID {
			vehicle = &vehicles[i]
			break
		}
	}
	
	if vehicle == nil {
		log.Printf("Vehicle not found: %s", vehicleID)
		http.Error(w, "Vehicle not found", http.StatusNotFound)
		return
	}
	
	// Get maintenance records
	allRecords, err := getAllVehicleMaintenanceRecords(vehicleID)
	if err != nil {
		log.Printf("Error getting maintenance records: %v", err)
		allRecords = []BusMaintenanceLog{}
	}
	
	// Calculate statistics
	stats := calculateMaintenanceStats(allRecords)
	
	// Create the template data
	data := struct {
		VehicleID          string
		IsBus              bool
		VehicleInfo        interface{}
		MaintenanceRecords []BusMaintenanceLog
		TotalRecords       int
		TotalCost          float64
		AverageCost        float64
		RecentCount        int
		Today              string
		CSRFToken          string
	}{
		VehicleID:          vehicleID,
		IsBus:              false,
		VehicleInfo:        vehicle,
		MaintenanceRecords: allRecords,
		TotalRecords:       stats.TotalRecords,
		TotalCost:          stats.TotalCost,
		AverageCost:        stats.AverageCost,
		RecentCount:        stats.RecentCount,
		Today:              time.Now().Format("2006-01-02"),
		CSRFToken:          getCSRFToken(r),
	}
	
	renderTemplate(w, r, "vehicle_maintenance.html", data)
}
// Part 3 - Continuing from vehicleMaintenanceHandler...

func fleetHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// Load buses
	buses, err := cache.GetBuses()
	if err != nil {
		log.Printf("Error loading buses: %v", err)
		buses = []*Bus{}
	}
	
	// Load recent maintenance logs
	recentMaintenanceLogs, err := getRecentMaintenanceActivity(5)
	if err != nil {
		log.Printf("Error loading recent maintenance logs: %v", err)
		recentMaintenanceLogs = []BusMaintenanceLog{}
	}
	
	log.Printf("Fleet handler: Found %d buses and %d recent maintenance logs", len(buses), len(recentMaintenanceLogs))
	
	data := FleetData{
		User:               user,
		Buses:              buses,
		Today:              time.Now().Format("2006-01-02"),
		CSRFToken:          getCSRFToken(r),
		MaintenanceLogs:    recentMaintenanceLogs,
	}
	
	renderTemplate(w, r, "fleet.html", data)
}

// ============= STUDENT MANAGEMENT HANDLERS =============

func studentsPage(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	driverStudents, err := getDriverStudents(user.Username)
	if err != nil {
		log.Printf("Error getting driver students: %v", err)
		driverStudents = []Student{}
	}
	
	routes, err := cache.GetRoutes()
	if err != nil {
		log.Printf("Error loading routes: %v", err)
		routes = []Route{}
	}
	
	data := StudentData{
		User:      user,
		Students:  driverStudents,
		Routes:    routes,
		CSRFToken: getCSRFToken(r),
	}
	
	renderTemplate(w, r, "students.html", data)
}

// ============= FLEET MANAGEMENT HANDLERS =============

func fleetPage(w http.ResponseWriter, r *http.Request) {
	buses, err := cache.GetBuses()
	if err != nil {
		log.Printf("Error loading buses: %v", err)
		buses = []*Bus{}
	}
	
	data := FleetData{
		User:      getUserFromSession(r),
		Buses:     buses,
		Today:     time.Now().Format(DateFormat),
		CSRFToken: getCSRFToken(r),
	}
	
	renderTemplate(w, r, "fleet.html", data)
}

func companyFleetPage(w http.ResponseWriter, r *http.Request) {
	vehicles, err := cache.GetVehicles()
	if err != nil {
		log.Printf("Error loading vehicles: %v", err)
		vehicles = []Vehicle{}
	}
	
	data := CompanyFleetData{
		User:      getUserFromSession(r),
		Vehicles:  vehicles,
		CSRFToken: getCSRFToken(r),
	}
	
	renderTemplate(w, r, "company_fleet.html", data)
}

// ============= ROUTE ASSIGNMENT HANDLERS =============

func assignRoutesPage(w http.ResponseWriter, r *http.Request) {
	assignments, err := loadRouteAssignments()
	if err != nil {
		log.Printf("Error loading route assignments: %v", err)
		assignments = []RouteAssignment{}
	}
	
	routes, _ := cache.GetRoutes()
	buses, _ := cache.GetBuses()
	users, _ := cache.GetUsers()
	
	// Calculate available resources
	assignmentData := calculateAssignmentData(assignments, routes, buses, users)
	assignmentData.CSRFToken = getCSRFToken(r)
	assignmentData.User = getUserFromSession(r)
	
	renderTemplate(w, r, "assign_routes.html", assignmentData)
}

// ============= PROFILE HANDLERS =============

func driverProfileHandler(w http.ResponseWriter, r *http.Request) {
	driverUsername := extractIDFromPath(r.URL.Path, "/driver/")
	if driverUsername == "" {
		http.Error(w, "Driver username required", http.StatusBadRequest)
		return
	}
	
	driverLogs, err := getDriverLogs(driverUsername)
	if err != nil {
		log.Printf("Error getting driver logs: %v", err)
		driverLogs = []DriverLog{}
	}
	
	data := struct {
		Name string
		Logs []DriverLog
	}{
		Name: driverUsername,
		Logs: driverLogs,
	}
	
	renderTemplate(w, r, "driver_profile.html", data)
}

// ============= ACTIVITY REPORT HANDLER =============

func activityReportHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	
	// Get date range
	startDate := r.URL.Query().Get("start")
	endDate := r.URL.Query().Get("end")
	
	if startDate == "" || endDate == "" {
		// Default to current month
		now := time.Now()
		startDate = now.AddDate(0, 0, -now.Day()+1).Format("2006-01-02")
		endDate = now.Format("2006-01-02")
	}
	
	// Load activities
	activities, err := loadActivitiesInRange(startDate, endDate)
	if err != nil {
		log.Printf("Error loading activities: %v", err)
		activities = []Activity{}
	}
	
	// Calculate totals
	totalMiles := 0.0
	totalAttendance := 0
	for _, a := range activities {
		totalMiles += a.Miles
		totalAttendance += a.Attendance
	}
	
	data := struct {
		User            *User
		Activities      []Activity
		StartDate       string
		EndDate         string
		TotalMiles      float64
		TotalAttendance int
		CSRFToken       string
	}{
		User:            user,
		Activities:      activities,
		StartDate:       startDate,
		EndDate:         endDate,
		TotalMiles:      totalMiles,
		TotalAttendance: totalAttendance,
		CSRFToken:       getCSRFToken(r),
	}
	
	renderTemplate(w, r, "activity_report.html", data)
}

// ============= HELPER FUNCTIONS (Updated for CSP) =============

// REMOVED: renderTemplate is now in utils.go
// REMOVED: renderLoginError is now in utils.go

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

func validateCSRFToken(r *http.Request, token string) bool {
	cookie, _ := r.Cookie(SessionCookieName)
	return cookie != nil && ValidateCSRFToken(cookie.Value, token)
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

// Additional helper functions for data processing

func validateRegistration(username, password string) error {
	if !ValidateUsername(username) {
		return fmt.Errorf("Invalid username format. Use 3-20 characters, letters and numbers only.")
	}
	if len(password) < MinPasswordLength {
		return fmt.Errorf("Password must be at least %d characters long.", MinPasswordLength)
	}
	return nil
}

func userExists(username string) (bool, error) {
	users, err := cache.GetUsers()
	if err != nil {
		return false, err
	}
	
	for _, user := range users {
		if user.Username == username {
			return true, nil
		}
	}
	return false, nil
}

func createPendingUser(username, password string) error {
	hashedPassword, err := HashPassword(password)
	if err != nil {
		return err
	}
	
	newUser := User{
		Username: username,
		Password: hashedPassword,
		Role:     RoleDriverPending,
		Status:   StatusPending,
	}
	
	return saveUser(newUser)
}

func getPendingUsers() ([]struct {
	Username  string
	CreatedAt string
}, error) {
	var pendingUsers []struct {
		Username  string
		CreatedAt string
	}
	
	users, err := cache.GetUsers()
	if err != nil {
		return nil, err
	}
	
	for _, u := range users {
		if u.Role == RoleDriverPending {
			pendingUsers = append(pendingUsers, struct {
				Username  string
				CreatedAt string
			}{
				Username:  u.Username,
				CreatedAt: u.RegistrationDate,
			})
		}
	}
	
	return pendingUsers, nil
}

func handleEditUserGet(w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("username")
	if username == "" {
		http.Error(w, "Username required", http.StatusBadRequest)
		return
	}
	
	// Find user
	users, err := cache.GetUsers()
	if err != nil {
		http.Error(w, "Failed to load users", http.StatusInternalServerError)
		return
	}
	
	var targetUser *User
	for _, u := range users {
		if u.Username == username {
			targetUser = &u
			break
		}
	}
	
	if targetUser == nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}
	
	data := struct {
		Username  string
		Role      string
		CSRFToken string
	}{
		Username:  targetUser.Username,
		Role:      targetUser.Role,
		CSRFToken: getCSRFToken(r),
	}
	
	renderTemplate(w, r, "users.html", data)
}

func handleVehicleMaintenance(w http.ResponseWriter, r *http.Request, vehicleID string, isBus bool) {
	var vehicleInfo interface{}
	
	if isBus {
		buses, _ := cache.GetBuses()
		for _, bus := range buses {
			if bus.BusID == vehicleID {
				vehicleInfo = bus
				break
			}
		}
	} else {
		vehicles, _ := cache.GetVehicles()
		for i := range vehicles {
			if vehicles[i].VehicleID == vehicleID {
				vehicleInfo = &vehicles[i]
				break
			}
		}
	}
	
	if vehicleInfo == nil {
		http.Error(w, "Vehicle not found", http.StatusNotFound)
		return
	}
	
	records, err := getAllVehicleMaintenanceRecords(vehicleID)
	if err != nil {
		log.Printf("Error loading maintenance records: %v", err)
		records = []BusMaintenanceLog{}
	}
	
	// Calculate statistics
	stats := calculateMaintenanceStats(records)
	
	data := struct {
		VehicleID          string
		IsBus              bool
		VehicleInfo        interface{}
		MaintenanceRecords []BusMaintenanceLog
		TotalRecords       int
		TotalCost          float64
		AverageCost        float64
		RecentCount        int
		Today              string
		CSRFToken          string
	}{
		VehicleID:          vehicleID,
		IsBus:              isBus,
		VehicleInfo:        vehicleInfo,
		MaintenanceRecords: records,
		TotalRecords:       stats.TotalRecords,
		TotalCost:          stats.TotalCost,
		AverageCost:        stats.AverageCost,
		RecentCount:        stats.RecentCount,
		Today:              time.Now().Format(DateFormat),
		CSRFToken:          getCSRFToken(r),
	}
	
	renderTemplate(w, r, "vehicle_maintenance.html", data)
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
