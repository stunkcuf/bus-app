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

// FIXED: Better bus ID abbreviation system that avoids collisions
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
	
	port := os.Getenv("PORT")
	if port == "" {
		port = DefaultPort
	}

	server := &http.Server{
		Addr:           fmt.Sprintf(":%s", port),
		Handler:        SecurityHeaders(mux),
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
	renderTemplate(w, "login.html", LoginFormData{CSRFToken: csrfToken})
}

func handleLoginPost(w http.ResponseWriter, r *http.Request) {
	username := SanitizeFormValue(r, "username")
	password := r.FormValue("password")

	// Validate input
	if !ValidateUsername(username) {
		renderLoginError(w, "Invalid username format")
		return
	}

	// Find user and check credentials
	users, err := cache.GetUsers()
	if err != nil {
		log.Printf("Error loading users: %v", err)
		renderLoginError(w, "System error. Please try again.")
		return
	}
	
	for _, user := range users {
		if user.Username == username && CheckPasswordHash(password, user.Password) {
			if user.Role == RoleDriverPending {
				renderLoginError(w, "Your account is pending approval. Please wait for a manager to approve your registration.")
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

	renderLoginError(w, "Invalid username or password")
}

// Legacy mileage reports handler (still needed for compatibility)
func viewMileageReportsHandler(w http.ResponseWriter, r *http.Request) {
	// Redirect to enhanced version
	http.Redirect(w, r, "/view-mileage-reports", http.StatusFound)
}

func registerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		renderTemplate(w, "register.html", struct{ Error string }{})
		return
	}

	// Handle POST
	username := SanitizeFormValue(r, "username")
	password := r.FormValue("password")

	// Validate input
	if err := validateRegistration(username, password); err != nil {
		renderTemplate(w, "register.html", struct{ Error string }{Error: err.Error()})
		return
	}

	// Check if username exists
	exists, err := userExists(username)
	if err != nil {
		log.Printf("Error checking user existence: %v", err)
		renderTemplate(w, "register.html", struct{ Error string }{
			Error: "System error. Please try again.",
		})
		return
	}
	
	if exists {
		renderTemplate(w, "register.html", struct{ Error string }{
			Error: "Username already exists. Please choose another.",
		})
		return
	}

	// Create pending user
	if err := createPendingUser(username, password); err != nil {
		renderTemplate(w, "register.html", struct{ Error string }{
			Error: "Failed to create account. Please try again.",
		})
		return
	}

	// Clear user cache
	cache.InvalidateUsers()
	
	renderTemplate(w, "registration_success.html", nil)
}

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

	renderTemplate(w, "approve_users.html", data)
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
	
	renderTemplate(w, "dashboard.html", data)
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
	
	renderTemplate(w, "driver_dashboard.html", data)
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
		
		renderTemplate(w, "import_mileage.html", data)
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
			renderTemplate(w, "import_mileage.html", data)
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
			renderTemplate(w, "import_mileage.html", data)
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
			renderTemplate(w, "import_mileage.html", data)
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
		
		renderTemplate(w, "import_mileage.html", data)
	}
}

// ============= ENHANCED EXCEL PROCESSING FUNCTIONS =============
func processMileageExcelFile(file multipart.File, filename string) (int, error) {
	f, err := excelize.OpenReader(file)
	if err != nil {
		return 0, fmt.Errorf("failed to open Excel file: %v", err)
	}
	defer f.Close()
	
	sheets := f.GetSheetList()
	log.Printf("Excel file has %d sheets: %v", len(sheets), sheets)
	
	if len(sheets) == 0 {
		return 0, fmt.Errorf("no sheets found in Excel file")
	}
	
	totalImported := 0
	
	// Process each sheet
	for _, sheetName := range sheets {
		imported, err := processSheet(f, sheetName)
		if err != nil {
			log.Printf("Error processing sheet '%s': %v", sheetName, err)
			continue
		}
		totalImported += imported
	}
	
	return totalImported, nil
}

func processSheet(f *excelize.File, sheetName string) (int, error) {
	log.Printf("\n=== Processing sheet: '%s' ===", sheetName)
	
	rows, err := f.GetRows(sheetName)
	if err != nil {
		return 0, fmt.Errorf("error reading sheet: %v", err)
	}
	
	if len(rows) == 0 {
		return 0, nil
	}
	
	// Extract month and year from sheet name if possible
	reportMonth := sheetName
	reportYear := 2024 // Default, can be overridden
	
	// Try to extract year from sheet name (e.g., "January 2024")
	parts := strings.Split(sheetName, " ")
	if len(parts) > 1 {
		if year, err := strconv.Atoi(parts[len(parts)-1]); err == nil && year > 2000 && year < 2100 {
			reportYear = year
			reportMonth = strings.Join(parts[:len(parts)-1], " ")
		}
	}
	
	var agencyVehicles []AgencyVehicleRecord
	var schoolBuses []SchoolBusRecord
	var programStaff []ProgramStaffRecord
	
	currentSection := ""
	headerRowIndex := -1
	
	// Process rows
	for i, row := range rows {
		if len(row) == 0 {
			continue
		}
		
		firstCell := strings.TrimSpace(row[0])
		firstCellLower := strings.ToLower(firstCell)
		
		// Detect section headers
		if strings.Contains(firstCellLower, "agency vehicle") {
			currentSection = "agency"
			headerRowIndex = -1
			log.Printf("Found Agency Vehicles section at row %d", i+1)
			continue
		} else if strings.Contains(firstCellLower, "school bus") {
			currentSection = "school_bus"
			headerRowIndex = -1
			log.Printf("Found School Buses section at row %d", i+1)
			continue
		} else if strings.Contains(firstCellLower, "program") {
			currentSection = "program"
			headerRowIndex = -1
			log.Printf("Found Programs section at row %d", i+1)
			continue
		}
		
		// Look for header row
		if headerRowIndex == -1 && isHeaderRow(row) {
			headerRowIndex = i
			log.Printf("Found header row at index %d", i)
			continue
		}
		
		// Skip if we haven't found a section or header yet
		if currentSection == "" || headerRowIndex == -1 {
			continue
		}
		
		// Process data rows based on section
		switch currentSection {
		case "agency":
			if vehicle := parseAgencyVehicleRow(row, reportMonth, reportYear, i+1); vehicle != nil {
				agencyVehicles = append(agencyVehicles, *vehicle)
			}
		case "school_bus":
			if bus := parseSchoolBusRow(row, reportMonth, reportYear, i+1); bus != nil {
				schoolBuses = append(schoolBuses, *bus)
			}
		case "program":
			if staff := parseProgramStaffRow(row, reportMonth, reportYear, i+1); staff != nil {
				programStaff = append(programStaff, *staff)
			}
		}
	}
	
	// Insert records into database
	imported := 0
	
	if len(agencyVehicles) > 0 {
		count, err := insertAgencyVehicles(agencyVehicles)
		if err != nil {
			log.Printf("Error inserting agency vehicles: %v", err)
		} else {
			imported += count
		}
	}
	
	if len(schoolBuses) > 0 {
		count, err := insertSchoolBuses(schoolBuses)
		if err != nil {
			log.Printf("Error inserting school buses: %v", err)
		} else {
			imported += count
		}
	}
	
	if len(programStaff) > 0 {
		count, err := insertProgramStaff(programStaff)
		if err != nil {
			log.Printf("Error inserting program staff: %v", err)
		} else {
			imported += count
		}
	}
	
	log.Printf("Sheet '%s' - Imported: %d records", sheetName, imported)
	return imported, nil
}

func isHeaderRow(row []string) bool {
	// Check for common header keywords
	headerKeywords := []string{"year", "make", "lic", "id", "located", "beginning", "ending", "total", "miles"}
	
	rowText := strings.ToLower(strings.Join(row, " "))
	matchCount := 0
	
	for _, keyword := range headerKeywords {
		if strings.Contains(rowText, keyword) {
			matchCount++
		}
	}
	
	return matchCount >= 3
}

func parseAgencyVehicleRow(row []string, reportMonth string, reportYear int, rowNum int) *AgencyVehicleRecord {
	if len(row) < 7 {
		return nil
	}
	
	// Skip empty or invalid rows
	if isEmptyRow(row) {
		return nil
	}
	
	record := &AgencyVehicleRecord{
		ReportMonth: reportMonth,
		ReportYear:  reportYear,
	}
	
	// Parse year (column 0)
	if year := parseInt(row[0]); year > 1900 && year < 2100 {
		record.VehicleYear = year
	}
	
	// Parse make/model (column 1)
	if len(row) > 1 {
		record.MakeModel = cleanText(row[1])
	}
	
	// Parse license plate (column 2)
	if len(row) > 2 {
		record.LicensePlate = cleanText(row[2])
	}
	
	// Parse vehicle ID (column 3)
	if len(row) > 3 {
		record.VehicleID = cleanText(row[3])
		if record.VehicleID == "" {
			log.Printf("Skipping row %d: missing vehicle ID", rowNum)
			return nil
		}
	}
	
	// Validate required fields
	if record.ReportMonth == "" || record.ReportYear == 0 {
		log.Printf("Skipping row %d: missing report date", rowNum)
		return nil
	}
	
	// Parse location (column 4)
	if len(row) > 4 {
		record.Location = cleanText(row[4])
	}
	
	// Parse miles (columns 5, 6, 7)
	if len(row) > 5 {
		record.BeginningMiles = parseInt(row[5])
	}
	if len(row) > 6 {
		record.EndingMiles = parseInt(row[6])
	}
	if len(row) > 7 {
		record.TotalMiles = parseInt(row[7])
	}
	
	// Parse status/notes from the end of the row
	if len(row) > 8 {
		statusText := strings.ToLower(cleanText(row[8]))
		if strings.Contains(statusText, "for sale") {
			record.Status = "FOR SALE"
		} else if strings.Contains(statusText, "sold") {
			record.Status = "SOLD"
		} else if strings.Contains(statusText, "out of lease") {
			record.Status = "OUT OF LEASE"
		} else if strings.Contains(statusText, "no report") {
			record.Status = "NO REPORT"
		} else if strings.Contains(statusText, "repair") {
			record.Status = "REPAIRS"
		} else {
			record.Notes = cleanText(row[8])
		}
	}
	
	log.Printf("Parsed agency vehicle: ID=%s, Status=%s, Miles=%d", 
		record.VehicleID, record.Status, record.TotalMiles)
	
	return record
}

func parseSchoolBusRow(row []string, reportMonth string, reportYear int, rowNum int) *SchoolBusRecord {
	if len(row) < 7 {
		return nil
	}
	
	// Skip empty rows
	if isEmptyRow(row) {
		return nil
	}
	
	record := &SchoolBusRecord{
		ReportMonth: reportMonth,
		ReportYear:  reportYear,
	}
	
	// Column mapping for school buses:
	// 0: ID, 1: Location/Status, 2-3: Miles or Year/Make info
	
	// Parse bus ID (usually first column for school buses)
	if len(row) > 0 {
		record.BusID = cleanText(row[0])
		if record.BusID == "" {
			log.Printf("Skipping row %d: missing bus ID", rowNum)
			return nil
		}
	}
	
	// Validate required fields
	if record.ReportMonth == "" || record.ReportYear == 0 {
		log.Printf("Skipping row %d: missing report date", rowNum)
		return nil
	}
	
	// Parse location/status (column 1)
	if len(row) > 1 {
		locationStatus := cleanText(row[1])
		statusLower := strings.ToLower(locationStatus)
		
		if strings.Contains(statusLower, "spare") {
			record.Status = "SPARE"
			record.Location = "SPARE"
		} else if strings.Contains(statusLower, "slated for") {
			record.Status = "SLATED FOR"
			record.Location = locationStatus
		} else if strings.Contains(statusLower, "sub for") {
			record.Status = "SUBSTITUTE"
			record.Location = locationStatus
		} else {
			record.Location = locationStatus
		}
	}
	
	// Look for year and make in subsequent columns
	for i := 2; i < len(row) && i < 5; i++ {
		if year := parseInt(row[i]); year > 2000 && year < 2100 {
			record.BusYear = year
		} else if strings.Contains(strings.ToUpper(row[i]), "CHEV") {
			record.BusMake = cleanText(row[i])
		} else if strings.HasPrefix(strings.ToUpper(row[i]), "SC") {
			record.LicensePlate = cleanText(row[i])
		}
	}
	
	// Parse miles from the last columns
	if len(row) >= 7 {
		// Try to find miles in the last 3 columns
		for i := len(row) - 3; i < len(row); i++ {
			if i >= 0 && i < len(row) {
				miles := parseInt(row[i])
				if miles > 0 {
					if record.BeginningMiles == 0 {
						record.BeginningMiles = miles
					} else if record.EndingMiles == 0 {
						record.EndingMiles = miles
					} else {
						record.TotalMiles = miles
					}
				}
			}
		}
	}
	
	log.Printf("Parsed school bus: ID=%s, Status=%s, Location=%s", 
		record.BusID, record.Status, record.Location)
	
	return record
}

func parseProgramStaffRow(row []string, reportMonth string, reportYear int, rowNum int) *ProgramStaffRecord {
	if len(row) < 2 {
		return nil
	}
	
	// Look for program type in first column
	programType := ""
	firstCell := strings.ToUpper(cleanText(row[0]))
	
	if strings.Contains(firstCell, "HS") {
		programType = "HS"
	} else if strings.Contains(firstCell, "OPK") {
		programType = "OPK"
	} else if strings.Contains(firstCell, "EHS") {
		programType = "EHS"
	}
	
	if programType == "" {
		return nil
	}
	
	record := &ProgramStaffRecord{
		ReportMonth: reportMonth,
		ReportYear:  reportYear,
		ProgramType: programType,
	}
	
	// Look for staff counts in the row
	counts := []int{}
	for i := 1; i < len(row); i++ {
		if count := parseInt(row[i]); count > 0 {
			counts = append(counts, count)
		}
	}
	
	if len(counts) >= 1 {
		record.StaffCount1 = counts[0]
	}
	if len(counts) >= 2 {
		record.StaffCount2 = counts[1]
	}
	
	log.Printf("Parsed program staff: Type=%s, Count1=%d, Count2=%d", 
		record.ProgramType, record.StaffCount1, record.StaffCount2)
	
	return record
}

// Database insert functions
func insertAgencyVehicles(records []AgencyVehicleRecord) (int, error) {
	if db == nil {
		return 0, fmt.Errorf("database not initialized")
	}
	
	count := 0
	for _, record := range records {
		_, err := db.Exec(`
			INSERT INTO agency_vehicles 
			(report_month, report_year, vehicle_year, make_model, license_plate, 
			 vehicle_id, location, beginning_miles, ending_miles, total_miles, status, notes)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
			ON CONFLICT (report_month, report_year, vehicle_id) 
			DO UPDATE SET
				vehicle_year = EXCLUDED.vehicle_year,
				make_model = EXCLUDED.make_model,
				license_plate = EXCLUDED.license_plate,
				location = EXCLUDED.location,
				beginning_miles = EXCLUDED.beginning_miles,
				ending_miles = EXCLUDED.ending_miles,
				total_miles = EXCLUDED.total_miles,
				status = EXCLUDED.status,
				notes = EXCLUDED.notes,
				updated_at = CURRENT_TIMESTAMP
		`, record.ReportMonth, record.ReportYear, record.VehicleYear, record.MakeModel,
		   record.LicensePlate, record.VehicleID, record.Location, record.BeginningMiles,
		   record.EndingMiles, record.TotalMiles, record.Status, record.Notes)
		
		if err != nil {
			log.Printf("Error inserting agency vehicle %s: %v", record.VehicleID, err)
		} else {
			count++
		}
	}
	
	log.Printf("Successfully inserted %d agency vehicles", count)
	return count, nil
}

func insertSchoolBuses(records []SchoolBusRecord) (int, error) {
	if db == nil {
		return 0, fmt.Errorf("database not initialized")
	}
	
	count := 0
	for _, record := range records {
		_, err := db.Exec(`
			INSERT INTO school_buses 
			(report_month, report_year, bus_year, bus_make, license_plate, 
			 bus_id, location, beginning_miles, ending_miles, total_miles, status, notes)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
			ON CONFLICT (report_month, report_year, bus_id) 
			DO UPDATE SET
				bus_year = EXCLUDED.bus_year,
				bus_make = EXCLUDED.bus_make,
				license_plate = EXCLUDED.license_plate,
				location = EXCLUDED.location,
				beginning_miles = EXCLUDED.beginning_miles,
				ending_miles = EXCLUDED.ending_miles,
				total_miles = EXCLUDED.total_miles,
				status = EXCLUDED.status,
				notes = EXCLUDED.notes,
				updated_at = CURRENT_TIMESTAMP
		`, record.ReportMonth, record.ReportYear, record.BusYear, record.BusMake,
		   record.LicensePlate, record.BusID, record.Location, record.BeginningMiles,
		   record.EndingMiles, record.TotalMiles, record.Status, record.Notes)
		
		if err != nil {
			log.Printf("Error inserting school bus %s: %v", record.BusID, err)
		} else {
			count++
		}
	}
	
	log.Printf("Successfully inserted %d school buses", count)
	return count, nil
}

func insertProgramStaff(records []ProgramStaffRecord) (int, error) {
	if db == nil {
		return 0, fmt.Errorf("database not initialized")
	}
	
	count := 0
	for _, record := range records {
		_, err := db.Exec(`
			INSERT INTO program_staff 
			(report_month, report_year, program_type, staff_count_1, staff_count_2)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (report_month, report_year, program_type) 
			DO UPDATE SET
				staff_count_1 = EXCLUDED.staff_count_1,
				staff_count_2 = EXCLUDED.staff_count_2,
				updated_at = CURRENT_TIMESTAMP
		`, record.ReportMonth, record.ReportYear, record.ProgramType,
		   record.StaffCount1, record.StaffCount2)
		
		if err != nil {
			log.Printf("Error inserting program staff %s: %v", record.ProgramType, err)
		} else {
			count++
		}
	}
	
	log.Printf("Successfully inserted %d program staff records", count)
	return count, nil
}

// Helper functions for Excel processing
func cleanText(s string) string {
	// Remove strikethrough markers (~~text~~)
	s = regexp.MustCompile(`~~(.+?)~~`).ReplaceAllString(s, "$1")
	// Remove extra spaces and trim
	s = strings.TrimSpace(s)
	s = regexp.MustCompile(`\s+`).ReplaceAllString(s, " ")
	return s
}

func parseInt(s string) int {
	s = cleanText(s)
	// Remove commas from numbers
	s = strings.ReplaceAll(s, ",", "")
	val, _ := strconv.Atoi(s)
	return val
}

func isEmptyRow(row []string) bool {
	for _, cell := range row {
		if cleanText(cell) != "" && cleanText(cell) != "-" {
			return false
		}
	}
	return true
}

// ============= REST OF YOUR HANDLERS =============

// ============= USER MANAGEMENT HANDLERS =============

func newUserHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		data := UserFormData{CSRFToken: getCSRFToken(r)}
		renderTemplate(w, "new_user.html", data)
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
	
	renderTemplate(w, "vehicle_maintenance.html", data)
}

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
	
	renderTemplate(w, "fleet.html", data)
}

func debugVehicleHandler(w http.ResponseWriter, r *http.Request) {
	vehicleID := extractIDFromPath(r.URL.Path, "/debug-vehicle/")
	if vehicleID == "" {
		http.Error(w, "Vehicle ID required", http.StatusBadRequest)
		return
	}
	
	// Call debug function from database.go
	debugMaintenanceTables(vehicleID)
	
	// Get statistics
	stats := getMaintenanceStats(vehicleID)
	
	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(struct {
		VehicleID        string
		DatabaseStatus   string
		MaintenanceStats interface{}
	}{
		VehicleID:        vehicleID,
		DatabaseStatus:   "Check server logs for detailed debug output",
		MaintenanceStats: stats,
	})
}

func saveMaintenanceRecordHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	if !validateCSRF(r) {
		http.Error(w, "Invalid CSRF token", http.StatusForbidden)
		return
	}
	
	record, err := parseMaintenanceRecord(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	if err := saveMaintenanceRecordToDB(record); err != nil {
		http.Error(w, "Failed to save maintenance record", http.StatusInternalServerError)
		return
	}
	
	sendJSONResponse(w, map[string]string{
		"status":  "success",
		"message": "Maintenance record saved successfully",
	})
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
	
	renderTemplate(w, "students.html", data)
}

func addStudentHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	if !validateCSRF(r) {
		http.Error(w, "Invalid CSRF token", http.StatusForbidden)
		return
	}
	
	user := getUserFromSession(r)
	student, err := parseStudentForm(r, user.Username, "")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	if err := saveStudent(student); err != nil {
		http.Error(w, "Failed to save student", http.StatusInternalServerError)
		return
	}
	
	http.Redirect(w, r, "/students", http.StatusFound)
}

func editStudentHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	if !validateCSRF(r) {
		http.Error(w, "Invalid CSRF token", http.StatusForbidden)
		return
	}
	
	user := getUserFromSession(r)
	studentID := r.FormValue("student_id")
	
	// Verify ownership
	if !verifyStudentOwnership(studentID, user.Username) {
		http.Error(w, "Student not found", http.StatusNotFound)
		return
	}
	
	student, err := parseStudentForm(r, user.Username, studentID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	if err := saveStudent(student); err != nil {
		http.Error(w, "Failed to update student", http.StatusInternalServerError)
		return
	}
	
	http.Redirect(w, r, "/students", http.StatusFound)
}

func removeStudentHandler(w http.ResponseWriter, r *http.Request) {
	if !validateCSRF(r) {
		http.Error(w, "Invalid CSRF token", http.StatusForbidden)
		return
	}
	
	user := getUserFromSession(r)
	studentID := r.FormValue("student_id")
	
	if studentID == "" {
		http.Error(w, "Student ID required", http.StatusBadRequest)
		return
	}
	
	if !verifyStudentOwnership(studentID, user.Username) {
		http.Error(w, "Student not found", http.StatusNotFound)
		return
	}
	
	if err := deleteStudent(studentID); err != nil {
		http.Error(w, "Failed to remove student", http.StatusInternalServerError)
		return
	}
	
	http.Redirect(w, r, "/students", http.StatusFound)
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
	
	renderTemplate(w, "fleet.html", data)
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
	
	renderTemplate(w, "company_fleet.html", data)
}

func updateVehicleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// CSRF validation is optional for AJAX calls
	csrfToken := r.FormValue("csrf_token")
	if csrfToken == "" {
		csrfToken = r.Header.Get(CSRFTokenHeader)
	}
	if csrfToken != "" && !validateCSRFToken(r, csrfToken) {
		http.Error(w, "Invalid CSRF token", http.StatusForbidden)
		return
	}
	
	status, err := parseVehicleStatusUpdate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	if err := updateVehicleStatusInDB(status); err != nil {
		http.Error(w, "Failed to update vehicle", http.StatusInternalServerError)
		return
	}
	
	// Clear cache after vehicle update
	cache.InvalidateVehicles()
	
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
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
	
	renderTemplate(w, "assign_routes.html", assignmentData)
}

func assignRouteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	if !validateCSRF(r) {
		http.Error(w, "Invalid CSRF token", http.StatusForbidden)
		return
	}
	
	assignment, err := parseRouteAssignment(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	if err := validateRouteAssignment(assignment); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	if err := saveRouteAssignment(assignment); err != nil {
		http.Error(w, "Failed to save assignment", http.StatusInternalServerError)
		return
	}
	
	http.Redirect(w, r, "/assign-routes", http.StatusFound)
}

func unassignRouteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	if !validateCSRF(r) {
		http.Error(w, "Invalid CSRF token", http.StatusForbidden)
		return
	}
	
	driver := r.FormValue("driver")
	if driver == "" {
		http.Error(w, "Driver required", http.StatusBadRequest)
		return
	}
	
	if err := deleteRouteAssignment(driver); err != nil {
		http.Error(w, "Failed to unassign route", http.StatusInternalServerError)
		return
	}
	
	http.Redirect(w, r, "/assign-routes", http.StatusFound)
}

func addRouteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	if !validateCSRF(r) {
		http.Error(w, "Invalid CSRF token", http.StatusForbidden)
		return
	}
	
	route, err := parseNewRoute(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	if err := saveRoute(route); err != nil {
		http.Error(w, "Failed to save route", http.StatusInternalServerError)
		return
	}
	
	// Clear cache after route creation
	cache.InvalidateRoutes()
	
	http.Redirect(w, r, "/assign-routes", http.StatusFound)
}

func editRouteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	if !validateCSRF(r) {
		http.Error(w, "Invalid CSRF token", http.StatusForbidden)
		return
	}
	
	route, err := parseRouteUpdate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	if err := updateRoute(route); err != nil {
		http.Error(w, "Failed to update route", http.StatusInternalServerError)
		return
	}
	
	// Clear cache after route update
	cache.InvalidateRoutes()
	
	http.Redirect(w, r, "/assign-routes", http.StatusFound)
}

func deleteRouteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	if !validateCSRF(r) {
		http.Error(w, "Invalid CSRF token", http.StatusForbidden)
		return
	}
	
	routeID := r.FormValue("route_id")
	if routeID == "" {
		http.Error(w, "Route ID required", http.StatusBadRequest)
		return
	}
	
	if err := validateRouteDelete(routeID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	if err := deleteRoute(routeID); err != nil {
		http.Error(w, "Failed to delete route", http.StatusInternalServerError)
		return
	}
	
	// Clear cache after route deletion
	cache.InvalidateRoutes()
	
	http.Redirect(w, r, "/assign-routes", http.StatusFound)
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
	
	renderTemplate(w, "driver_profile.html", data)
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

func logout(w http.ResponseWriter, r *http.Request) {
	// Get username from session before clearing
	if user := getUserFromSession(r); user != nil {
		// Clear all sessions for this user
		ClearUserSessions(user.Username)
	}
	
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})
	
	http.Redirect(w, r, "/", http.StatusFound)
}

// ============= HELPER FUNCTIONS =============

func renderTemplate(w http.ResponseWriter, name string, data interface{}) {
	executeTemplate(w, name, data)
}

func renderLoginError(w http.ResponseWriter, errorMsg string) {
	csrfToken, _ := GenerateSecureToken()
	renderTemplate(w, "login.html", LoginFormData{
		Error:     errorMsg,
		CSRFToken: csrfToken,
	})
}

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

func countPendingUsers() (int, error) {
	users, err := cache.GetUsers()
	if err != nil {
		return 0, err
	}
	
	count := 0
	for _, u := range users {
		if u.Role == RoleDriverPending {
			count++
		}
	}
	return count, nil
}

func validateApprovalRequest(username, action string) error {
	if username == "" || (action != "approve" && action != "reject") {
		return fmt.Errorf("Invalid request")
	}
	return nil
}

func processUserApproval(username, action string) error {
	if action == "reject" {
		return deleteUser(username)
	}
	
	// Approve user
	users, err := loadUsersFromDB()
	if err != nil {
		return err
	}
	
	for i, u := range users {
		if u.Username == username && u.Role == RoleDriverPending {
			users[i].Role = RoleDriver
			users[i].Status = StatusActive
			return updateUser(users[i])
		}
	}
	
	return fmt.Errorf("User not found or already processed")
}

func getRouteAndBus(assignment *RouteAssignment) (*Route, *Bus) {
	if assignment == nil {
		return nil, nil
	}
	
	// Get route
	routes, _ := cache.GetRoutes()
	var route *Route
	for _, r := range routes {
		if r.RouteID == assignment.RouteID {
			route = &r
			break
		}
	}
	
	// Get bus
	buses, _ := cache.GetBuses()
	var bus *Bus
	for _, b := range buses {
		if b.BusID == assignment.BusID {
			bus = b
			break
		}
	}
	
	return route, bus
}

func getRouteStudents(route *Route, driverUsername, period string) ([]Student, error) {
	if route == nil {
		return []Student{}, nil
	}
	
	students, err := loadStudentsFromDB()
	if err != nil {
		return nil, err
	}
	
	var routeStudents []Student
	for _, s := range students {
		if s.RouteID == route.RouteID && s.Driver == driverUsername && s.Active {
			routeStudents = append(routeStudents, s)
		}
	}
	
	// Sort by pickup/dropoff time
	sort.Slice(routeStudents, func(i, j int) bool {
		if period == "morning" {
			if routeStudents[i].PickupTime == "" {
				return false
			}
			if routeStudents[j].PickupTime == "" {
				return true
			}
			return routeStudents[i].PickupTime < routeStudents[j].PickupTime
		} else {
			if routeStudents[i].DropoffTime == "" {
				return false
			}
			if routeStudents[j].DropoffTime == "" {
				return true
			}
			return routeStudents[i].DropoffTime < routeStudents[j].DropoffTime
		}
	})
	
	return routeStudents, nil
}

func getDriverLogForDatePeriod(driver, date, period string) *DriverLog {
	logs, err := loadDriverLogsFromDB()
	if err != nil {
		log.Printf("Error loading driver logs: %v", err)
		return nil
	}
	
	for _, log := range logs {
		if log.Driver == driver && log.Date == date && log.Period == period {
			return &log
		}
	}
	return nil
}

func getRecentDriverLogs(driver string, limit int) ([]DriverLog, error) {
	logs, err := loadDriverLogsFromDB()
	if err != nil {
		return nil, err
	}
	
	var recentLogs []DriverLog
	for _, log := range logs {
		if log.Driver == driver {
			recentLogs = append(recentLogs, log)
			if len(recentLogs) >= limit {
				break
			}
		}
	}
	return recentLogs, nil
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
	
	renderTemplate(w, "users.html", data)
}

func handleEditUserPost(w http.ResponseWriter, r *http.Request) {
	if !validateCSRF(r) {
		http.Error(w, "Invalid CSRF token", http.StatusForbidden)
		return
	}
	
	username := r.FormValue("username")
	action := r.FormValue("action")
	role := r.FormValue("role")
	password := r.FormValue("password")
	
	if username == "" {
		http.Error(w, "Username required", http.StatusBadRequest)
		return
	}
	
	// Find existing user
	users, err := loadUsersFromDB()
	if err != nil {
		http.Error(w, "Failed to load users", http.StatusInternalServerError)
		return
	}
	
	var existingUser *User
	for i := range users {
		if users[i].Username == username {
			existingUser = &users[i]
			break
		}
	}
	
	if existingUser == nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}
	
	// Handle different actions
	switch action {
	case "update_role":
		if role != RoleDriver && role != RoleManager {
			http.Error(w, "Invalid role", http.StatusBadRequest)
			return
		}
		existingUser.Role = role
		
	case "reset_password":
		if len(password) < MinPasswordLength {
			http.Error(w, fmt.Sprintf("Password must be at least %d characters", MinPasswordLength), http.StatusBadRequest)
			return
		}
		
		hashedPassword, err := HashPassword(password)
		if err != nil {
			log.Printf("Failed to hash password: %v", err)
			http.Error(w, "Failed to hash password", http.StatusInternalServerError)
			return
		}
		existingUser.Password = hashedPassword
		
	default:
		// Legacy behavior
		if role != "" {
			if role != RoleDriver && role != RoleManager {
				http.Error(w, "Invalid role", http.StatusBadRequest)
				return
			}
			existingUser.Role = role
		}
		
		if password != "" {
			if len(password) < MinPasswordLength {
				http.Error(w, fmt.Sprintf("Password must be at least %d characters", MinPasswordLength), http.StatusBadRequest)
				return
			}
			
			hashedPassword, err := HashPassword(password)
			if err != nil {
				log.Printf("Failed to hash password: %v", err)
				http.Error(w, "Failed to hash password", http.StatusInternalServerError)
				return
			}
			existingUser.Password = hashedPassword
		}
	}
	
	if err := updateUser(*existingUser); err != nil {
		log.Printf("Failed to update user %s: %v", username, err)
		http.Error(w, "Failed to update user", http.StatusInternalServerError)
		return
	}
	
	// Clear cache after user update
	cache.InvalidateUsers()
	
	http.Redirect(w, r, "/manager-dashboard", http.StatusFound)
}

func parseDriverLog(r *http.Request, driverUsername string) (DriverLog, error) {
	if err := r.ParseForm(); err != nil {
		return DriverLog{}, fmt.Errorf("Failed to parse form")
	}
	
	date := r.FormValue("date")
	period := r.FormValue("period")
	departure := r.FormValue("departure")
	arrival := r.FormValue("arrival")
	mileageStr := r.FormValue("mileage")
	routeID := r.FormValue("route_id")
	busID := r.FormValue("bus_id")
	
	var mileage float64
	fmt.Sscanf(mileageStr, "%f", &mileage)
	
	// Build attendance records
	var attendance []struct {
		Position   int    `json:"position"`
		Present    bool   `json:"present"`
		PickupTime string `json:"pickup_time,omitempty"`
	}
	
	for key := range r.Form {
		if len(key) > 8 && key[:8] == "present_" {
			var position int
			fmt.Sscanf(key[8:], "%d", &position)
			
			pickupTime := r.FormValue(fmt.Sprintf("pickup_%d", position))
			
			attendance = append(attendance, struct {
				Position   int    `json:"position"`
				Present    bool   `json:"present"`
				PickupTime string `json:"pickup_time,omitempty"`
			}{
				Position:   position,
				Present:    true,
				PickupTime: pickupTime,
			})
		}
	}
	
	return DriverLog{
		Driver:     driverUsername,
		BusID:      busID,
		RouteID:    routeID,
		Date:       date,
		Period:     period,
		Departure:  departure,
		Arrival:    arrival,
		Mileage:    mileage,
		Attendance: attendance,
	}, nil
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
	
	renderTemplate(w, "vehicle_maintenance.html", data)
}

func getMaintenanceStats(vehicleID string) interface{} {
	stats := struct {
		BusMaintenanceLogs  int
		MaintenanceRecords  int
		ServiceRecords      int
		TotalRecords        int
	}{}
	
	if db != nil {
		db.QueryRow("SELECT COUNT(*) FROM bus_maintenance_logs WHERE bus_id = $1", vehicleID).Scan(&stats.BusMaintenanceLogs)
		db.QueryRow("SELECT COUNT(*) FROM maintenance_records WHERE vehicle_id = $1", vehicleID).Scan(&stats.MaintenanceRecords)
		db.QueryRow(`
			SELECT COUNT(*) FROM service_records 
			WHERE COALESCE(vehicle_number::VARCHAR, vehicle_id::VARCHAR, unnamed_1::VARCHAR) = $1
		`, vehicleID).Scan(&stats.ServiceRecords)
		
		stats.TotalRecords = stats.BusMaintenanceLogs + stats.MaintenanceRecords + stats.ServiceRecords
	}
	
	return stats
}

func calculateMaintenanceStats(records []BusMaintenanceLog) struct {
	TotalRecords int
	TotalCost    float64
	AverageCost  float64
	RecentCount  int
} {
	stats := struct {
		TotalRecords int
		TotalCost    float64
		AverageCost  float64
		RecentCount  int
	}{
		TotalRecords: len(records),
	}
	
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30).Format(DateFormat)
	for _, record := range records {
		if record.Date >= thirtyDaysAgo {
			stats.RecentCount++
		}
	}
	
	if stats.TotalRecords > 0 && stats.TotalCost > 0 {
		stats.AverageCost = stats.TotalCost / float64(stats.TotalRecords)
	}
	
	return stats
}

func parseMaintenanceRecord(r *http.Request) (BusMaintenanceLog, error) {
	if err := r.ParseForm(); err != nil {
		return BusMaintenanceLog{}, fmt.Errorf("Failed to parse form")
	}
	
	vehicleID := r.FormValue("bus_id")
	if vehicleID == "" {
		vehicleID = r.FormValue("vehicle_id")
	}
	
	date := r.FormValue("date")
	category := r.FormValue("category")
	notes := r.FormValue("notes")
	mileageStr := r.FormValue("mileage")
	
	if vehicleID == "" || date == "" || category == "" || notes == "" {
		return BusMaintenanceLog{}, fmt.Errorf("Missing required fields")
	}
	
	var mileage int
	if mileageStr != "" {
		fmt.Sscanf(mileageStr, "%d", &mileage)
	}
	
	return BusMaintenanceLog{
		BusID:    vehicleID,
		Date:     date,
		Category: category,
		Notes:    notes,
		Mileage:  mileage,
	}, nil
}

func saveMaintenanceRecordToDB(maintenanceLog BusMaintenanceLog) error {
	// Determine vehicle type
	vehicleType := "vehicle"
	buses, _ := cache.GetBuses()
	for _, bus := range buses {
		if bus.BusID == maintenanceLog.BusID {
			vehicleType = "bus"
			break
		}
	}
	
	savedAny := false
	
	// Save to maintenance_records (unified table)
	if err := saveMaintenanceRecord(maintenanceLog, vehicleType); err != nil {
		log.Printf("Failed to save to maintenance_records: %v", err)
	} else {
		savedAny = true
	}
	
	// If it's a bus, also save to bus_maintenance_logs
	if vehicleType == "bus" {
		if err := saveMaintenanceLog(maintenanceLog); err != nil {
			log.Printf("Failed to save to bus_maintenance_logs: %v", err)
		} else {
			savedAny = true
		}
	}
	
	if !savedAny {
		return fmt.Errorf("Failed to save maintenance record")
	}
	
	return nil
}

func getDriverStudents(driverUsername string) ([]Student, error) {
	students, err := loadStudentsFromDB()
	if err != nil {
		return nil, err
	}
	
	var driverStudents []Student
	for _, s := range students {
		if s.Driver == driverUsername {
			driverStudents = append(driverStudents, s)
		}
	}
	return driverStudents, nil
}

func verifyStudentOwnership(studentID, driverUsername string) bool {
	students, err := loadStudentsFromDB()
	if err != nil {
		return false
	}
	
	for _, s := range students {
		if s.StudentID == studentID && s.Driver == driverUsername {
			return true
		}
	}
	return false
}

func parseStudentForm(r *http.Request, driverUsername, studentID string) (Student, error) {
	// Get form values
	name := SanitizeFormValue(r, "name")
	guardian := SanitizeFormValue(r, "guardian")
	phoneNumber := SanitizeFormValue(r, "phone_number")
	altPhoneNumber := SanitizeFormValue(r, "alt_phone_number")
	pickupTime := r.FormValue("pickup_time")
	dropoffTime := r.FormValue("dropoff_time")
	routeID := r.FormValue("route_id")
	
	// Generate student ID if new
	if studentID == "" {
		students, _ := loadStudentsFromDB()
		studentID = fmt.Sprintf("STU%03d", len(students)+1)
	}
	
	// Build locations
	var locations []Location
	
	// Process pickup locations
	pickupAddresses := r.Form["pickup_address"]
	pickupDescriptions := r.Form["pickup_description"]
	for i := range pickupAddresses {
		if pickupAddresses[i] != "" {
			desc := ""
			if i < len(pickupDescriptions) {
				desc = pickupDescriptions[i]
			}
			locations = append(locations, Location{
				Type:        "pickup",
				Address:     pickupAddresses[i],
				Description: desc,
			})
		}
	}
	
	// Process dropoff locations
	dropoffAddresses := r.Form["dropoff_address"]
	dropoffDescriptions := r.Form["dropoff_description"]
	for i := range dropoffAddresses {
		if dropoffAddresses[i] != "" {
			desc := ""
			if i < len(dropoffDescriptions) {
				desc = dropoffDescriptions[i]
			}
			locations = append(locations, Location{
				Type:        "dropoff",
				Address:     dropoffAddresses[i],
				Description: desc,
			})
		}
	}
	
	// Position number
	var positionNumber int
	if posStr := r.FormValue("position_number"); posStr != "" {
		fmt.Sscanf(posStr, "%d", &positionNumber)
	}
	
	// Active status - default to true for new students
	active := true
	if studentID != "" {
		// For existing students, check the form value
		activeValue := r.FormValue("active")
		active = activeValue == "on" || activeValue == "true"
	}
	
	log.Printf("DEBUG: Parsed student %s with %d locations", studentID, len(locations))
	
	return Student{
		StudentID:      studentID,
		Name:           name,
		Locations:      locations,
		PhoneNumber:    phoneNumber,
		AltPhoneNumber: altPhoneNumber,
		Guardian:       guardian,
		PickupTime:     pickupTime,
		DropoffTime:    dropoffTime,
		PositionNumber: positionNumber,
		RouteID:        routeID,
		Driver:         driverUsername,
		Active:         active,
	}, nil
}

func parseVehicleStatusUpdate(r *http.Request) (struct {
	VehicleID  string
	StatusType string
	NewStatus  string
}, error) {
	if err := r.ParseForm(); err != nil {
		return struct {
			VehicleID  string
			StatusType string
			NewStatus  string
		}{}, fmt.Errorf("Failed to parse form")
	}
	
	vehicleID := r.FormValue("vehicle_id")
	statusType := r.FormValue("status_type")
	newStatus := r.FormValue("new_status")
	
	if vehicleID == "" || statusType == "" || newStatus == "" {
		return struct {
			VehicleID  string
			StatusType string
			NewStatus  string
		}{}, fmt.Errorf("Missing required parameters")
	}
	
	return struct {
		VehicleID  string
		StatusType string
		NewStatus  string
	}{
		VehicleID:  vehicleID,
		StatusType: statusType,
		NewStatus:  newStatus,
	}, nil
}

func updateVehicleStatusInDB(status struct {
	VehicleID  string
	StatusType string
	NewStatus  string
}) error {
	vehicles, err := loadVehiclesFromDB()
	if err != nil {
		return err
	}
	
	for i := range vehicles {
		if vehicles[i].VehicleID == status.VehicleID {
			switch status.StatusType {
			case "oil":
				vehicles[i].OilStatus = status.NewStatus
			case "tire":
				vehicles[i].TireStatus = status.NewStatus
			case "status":
				vehicles[i].Status = status.NewStatus
			default:
				return fmt.Errorf("Invalid status type")
			}
			
			return saveVehicle(vehicles[i])
		}
	}
	
	return fmt.Errorf("Vehicle not found")
}

func calculateAssignmentData(assignments []RouteAssignment, routes []Route, buses []*Bus, users []User) AssignRouteData {
	// Track assigned resources
	assignedBuses := make(map[string]bool)
	assignedRoutes := make(map[string]bool)
	
	// Create a map to track which routes are assigned to each driver
	driverRoutes := make(map[string][]string)
	
	for _, assignment := range assignments {
		assignedBuses[assignment.BusID] = true
		assignedRoutes[assignment.RouteID] = true
		driverRoutes[assignment.Driver] = append(driverRoutes[assignment.Driver], assignment.RouteID)
	}
	
	// Filter available resources
	// NOTE: For drivers, we show ALL drivers since they can have multiple routes
	var availableDrivers []User
	for _, u := range users {
		if u.Role == RoleDriver {
			// Include ALL drivers, not just unassigned ones
			availableDrivers = append(availableDrivers, u)
		}
	}
	
	var availableBuses []*Bus
	for _, bus := range buses {
		if bus.Status == StatusActive && !assignedBuses[bus.BusID] {
			availableBuses = append(availableBuses, bus)
		}
	}
	
	// Create routes with status
	var routesWithStatus []struct {
		Route
		IsAssigned bool `json:"is_assigned"`
	}
	
	for _, route := range routes {
		routesWithStatus = append(routesWithStatus, struct {
			Route
			IsAssigned bool `json:"is_assigned"`
		}{
			Route:      route,
			IsAssigned: assignedRoutes[route.RouteID],
		})
	}
	
	return AssignRouteData{
		Assignments:           assignments,
		Drivers:               availableDrivers,
		AvailableRoutes:       routes,
		AvailableBuses:        availableBuses,
		RoutesWithStatus:      routesWithStatus,
		TotalAssignments:      len(assignments),
		TotalRoutes:           len(routes),
		AvailableDriversCount: len(availableDrivers),
		AvailableBusesCount:   len(availableBuses),
	}
}

func parseRouteAssignment(r *http.Request) (RouteAssignment, error) {
	driver := r.FormValue("driver")
	busID := r.FormValue("bus_id")
	routeID := r.FormValue("route_id")
	
	// Get route name
	routes, _ := cache.GetRoutes()
	routeName := ""
	for _, route := range routes {
		if route.RouteID == routeID {
			routeName = route.RouteName
			break
		}
	}
	
	return RouteAssignment{
		Driver:       driver,
		BusID:        busID,
		RouteID:      routeID,
		RouteName:    routeName,
		AssignedDate: time.Now().Format(DateFormat),
	}, nil
}

func parseNewRoute(r *http.Request) (Route, error) {
	routeName := SanitizeFormValue(r, "route_name")
	description := SanitizeFormValue(r, "description")
	
	if routeName == "" {
		return Route{}, fmt.Errorf("Route name required")
	}
	
	// Generate route ID
	routes, _ := cache.GetRoutes()
	routeID := fmt.Sprintf("RT%03d", len(routes)+1)
	
	return Route{
		RouteID:     routeID,
		RouteName:   routeName,
		Description: description,
		Positions:   []struct {
			Position int    `json:"position"`
			Student  string `json:"student"`
		}{},
	}, nil
}

func parseRouteUpdate(r *http.Request) (Route, error) {
	routeID := r.FormValue("route_id")
	routeName := SanitizeFormValue(r, "route_name")
	description := SanitizeFormValue(r, "description")
	
	if routeID == "" || routeName == "" {
		return Route{}, fmt.Errorf("Route ID and name required")
	}
	
	// Find existing route
	routes, _ := cache.GetRoutes()
	for _, route := range routes {
		if route.RouteID == routeID {
			route.RouteName = routeName
			route.Description = description
			return route, nil
		}
	}
	
	return Route{}, fmt.Errorf("Route not found")
}

func updateRoute(route Route) error {
	routes, err := loadRoutesFromDB()
	if err != nil {
		return err
	}
	
	for i := range routes {
		if routes[i].RouteID == route.RouteID {
			routes[i] = route
			return saveRoute(routes[i])
		}
	}
	return fmt.Errorf("Route not found")
}

func validateRouteDelete(routeID string) error {
	// Check if route is assigned
	assignments, _ := loadRouteAssignments()
	for _, a := range assignments {
		if a.RouteID == routeID {
			return fmt.Errorf("Cannot delete route that is currently assigned")
		}
	}
	
	// Check if students are on this route
	students, _ := loadStudentsFromDB()
	for _, s := range students {
		if s.RouteID == routeID && s.Active {
			return fmt.Errorf("Cannot delete route that has active students assigned")
		}
	}
	
	return nil
}

func getDriverLogs(driverUsername string) ([]DriverLog, error) {
	allLogs, err := loadDriverLogsFromDB()
	if err != nil {
		return nil, err
	}
	
	var driverLogs []DriverLog
	for _, log := range allLogs {
		if log.Driver == driverUsername {
			driverLogs = append(driverLogs, log)
		}
	}
	return driverLogs, nil
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
