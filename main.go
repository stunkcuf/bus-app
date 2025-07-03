package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
	"mime/multipart"
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

// MileageRecord represents a row from the Excel file
type MileageRecord struct {
	ReportMonth    string
	ReportYear     int
	BusYear        int
	BusMake        string
	LicensePlate   string
	BusID          string
	LocatedAt      string
	BeginningMiles int
	EndingMiles    int
	TotalMiles     int
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

// ADD THIS FUNCTION TO FIX THE BUSID LENGTH ISSUE
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
	
	// If still too long and not in our map, truncate to 10 chars
	if len(busID) > 10 {
		log.Printf("Warning: Truncating BusID '%s' to '%s'", busID, busID[:10])
		return busID[:10]
	}
	
	return busID
}

func main() {
	// Database setup
	log.Println("🗄️  Setting up PostgreSQL database...")
	setupDatabase()
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

	log.Printf("🚀 Server starting on port %s", port)
	if err := server.ListenAndServe(); err != nil {
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
	mux.HandleFunc("/dashboard", withRecovery(requireAuth(dashboardRouter)))

	return mux
}

// setupManagerRoutes configures manager-specific routes
func setupManagerRoutes(mux *http.ServeMux) {
	// User management
	mux.HandleFunc("/approve-users", withRecovery(requireAuth(requireRole("manager")(approveUsersHandler))))
	mux.HandleFunc("/approve-user", withRecovery(requireAuth(requireRole("manager")(approveUserHandler))))
	mux.HandleFunc("/new-user", withRecovery(requireAuth(requireRole("manager")(newUserHandler))))
	mux.HandleFunc("/edit-user", withRecovery(requireAuth(requireRole("manager")(editUserHandler))))
	mux.HandleFunc("/remove-user", withRecovery(requireAuth(requireRole("manager")(removeUserHandler))))
	
	// Dashboard
	mux.HandleFunc("/manager-dashboard", withRecovery(requireAuth(requireRole("manager")(managerDashboard))))
	
	// Fleet management
	mux.HandleFunc("/fleet", withRecovery(requireAuth(requireRole("manager")(fleetPage))))
	mux.HandleFunc("/company-fleet", withRecovery(requireAuth(requireRole("manager")(companyFleetPage))))
	mux.HandleFunc("/update-vehicle-status", withRecovery(requireAuth(requireRole("manager")(updateVehicleStatus))))
	
	// Maintenance
	mux.HandleFunc("/debug-vehicle/", withRecovery(requireAuth(requireRole("manager")(debugVehicleHandler))))
	mux.HandleFunc("/bus-maintenance/", withRecovery(requireAuth(requireRole("manager")(busMaintenanceHandler))))
	mux.HandleFunc("/vehicle-maintenance/", withRecovery(requireAuth(requireRole("manager")(vehicleMaintenanceHandler))))
	mux.HandleFunc("/save-maintenance-record", withRecovery(requireAuth(requireRole("manager")(saveMaintenanceRecordHandler))))
	
	// Route management
	mux.HandleFunc("/assign-routes", withRecovery(requireAuth(requireRole("manager")(assignRoutesPage))))
	mux.HandleFunc("/assign-route", withRecovery(requireAuth(requireRole("manager")(assignRouteHandler))))
	mux.HandleFunc("/unassign-route", withRecovery(requireAuth(requireRole("manager")(unassignRouteHandler))))
	mux.HandleFunc("/assign-routes/add", withRecovery(requireAuth(requireRole("manager")(addRouteHandler))))
	mux.HandleFunc("/assign-routes/edit", withRecovery(requireAuth(requireRole("manager")(editRouteHandler))))
	mux.HandleFunc("/assign-routes/delete", withRecovery(requireAuth(requireRole("manager")(deleteRouteHandler))))
	
	// API endpoints for route assignment
	mux.HandleFunc("/api/route-assignment", withRecovery(requireAuth(requireRole("manager")(handleSaveRouteAssignment))))
	mux.HandleFunc("/api/check-driver-bus", withRecovery(requireAuth(requireRole("manager")(handleCheckDriverBus))))
	
	// In setupManagerRoutes function, add:
	mux.HandleFunc("/import-mileage", withRecovery(requireAuth(requireRole("manager")(importMileageHandler))))
	mux.HandleFunc("/view-mileage-reports", withRecovery(requireAuth(requireRole("manager")(viewMileageReportsHandler))))
	
	// Driver profile
	mux.HandleFunc("/driver/", withRecovery(requireAuth(requireRole("manager")(driverProfileHandler))))
}

// setupDriverRoutes configures driver-specific routes
func setupDriverRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/driver-dashboard", withRecovery(requireAuth(requireRole("driver")(driverDashboard))))
	mux.HandleFunc("/save-log", withRecovery(requireAuth(requireRole("driver")(saveDriverLogHandler))))
	
	// Student management
	mux.HandleFunc("/students", withRecovery(requireAuth(requireRole("driver")(studentsPage))))
	mux.HandleFunc("/add-student", withRecovery(requireAuth(requireRole("driver")(addStudentHandler))))
	mux.HandleFunc("/edit-student", withRecovery(requireAuth(requireRole("driver")(editStudentHandler))))
	mux.HandleFunc("/remove-student", withRecovery(requireAuth(requireRole("driver")(removeStudentHandler))))
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
	users := loadUsers()
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
// ============= MILEAGE REPORTS HANDLER =============
func viewMileageReportsHandler(w http.ResponseWriter, r *http.Request) {
    user := getUserFromSession(r)
    if user == nil || user.Role != "manager" {
        http.Redirect(w, r, "/", http.StatusFound)
        return
    }
    
    // Get query parameters for filtering
    month := r.URL.Query().Get("month")
    year := r.URL.Query().Get("year")
    busID := r.URL.Query().Get("bus_id")
    
    // Load mileage reports from database
    reports, err := getMileageReports(month, year, busID)
    if err != nil {
        log.Printf("Error loading mileage reports: %v", err)
        reports = []MileageRecord{}
    }
    
    // Get summary statistics
    stats := getMileageStats(reports)
    
    data := struct {
        User      *User
        Reports   []MileageRecord
        Stats     MileageStats
        CSRFToken string
        // Filter values
        FilterMonth string
        FilterYear  string
        FilterBusID string
    }{
        User:        user,
        Reports:     reports,
        Stats:       stats,
        CSRFToken:   getCSRFToken(r),
        FilterMonth: month,
        FilterYear:  year,
        FilterBusID: busID,
    }
    
    renderTemplate(w, "view_mileage_reports.html", data)
}

// Add these helper functions
type MileageStats struct {
    TotalRecords   int
    TotalMiles     int
    AverageMiles   float64
    UniqueBuses    int
}

func getMileageReports(month, year, busID string) ([]MileageRecord, error) {
    query := `
        SELECT report_month, report_year, bus_year, bus_make, 
               license_plate, bus_id, located_at, 
               beginning_miles, ending_miles, total_miles
        FROM monthly_mileage_reports
        WHERE 1=1
    `
    args := []interface{}{}
    argCount := 0
    
    if month != "" {
        argCount++
        query += fmt.Sprintf(" AND report_month = $%d", argCount)
        args = append(args, month)
    }
    
    if year != "" {
        argCount++
        query += fmt.Sprintf(" AND report_year = $%d", argCount)
        args = append(args, year)
    }
    
    if busID != "" {
        argCount++
        query += fmt.Sprintf(" AND bus_id = $%d", argCount)
        args = append(args, busID)
    }
    
    query += " ORDER BY report_year DESC, report_month, bus_id"
    
    rows, err := db.Query(query, args...)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    var reports []MileageRecord
    for rows.Next() {
        var r MileageRecord
        err := rows.Scan(&r.ReportMonth, &r.ReportYear, &r.BusYear, 
                        &r.BusMake, &r.LicensePlate, &r.BusID, 
                        &r.LocatedAt, &r.BeginningMiles, 
                        &r.EndingMiles, &r.TotalMiles)
        if err != nil {
            log.Printf("Error scanning mileage record: %v", err)
            continue
        }
        reports = append(reports, r)
    }
    
    return reports, nil
}

func getMileageStats(reports []MileageRecord) MileageStats {
    stats := MileageStats{
        TotalRecords: len(reports),
    }
    
    busMap := make(map[string]bool)
    totalMiles := 0
    
    for _, r := range reports {
        totalMiles += r.TotalMiles
        busMap[r.BusID] = true
    }
    
    stats.TotalMiles = totalMiles
    stats.UniqueBuses = len(busMap)
    if stats.TotalRecords > 0 {
        stats.AverageMiles = float64(totalMiles) / float64(stats.TotalRecords)
    }
    
    return stats
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
	if userExists(username) {
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

	renderTemplate(w, "registration_success.html", nil)
}

func approveUsersHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != RoleManager {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	pendingUsers := getPendingUsers()
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

	pendingCount := countPendingUsers()
	csrfToken := getCSRFToken(r)
	
	data := DashboardData{
		User:            user,
		Role:            user.Role,
		Users:           loadUsers(),
		Buses:           loadBuses(),
		Routes:          loadRoutesWithDefault(),
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
	routeStudents := getRouteStudents(route, user.Username, period)
	driverLog := getDriverLogForDatePeriod(user.Username, date, period)
	recentLogs := getRecentDriverLogs(user.Username, 5)
	
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
            Success:   fmt.Sprintf("Successfully imported %d mileage records from '%s'!", importedCount, header.Filename),
        }
        
        renderTemplate(w, "import_mileage.html", data)
    }
}

// Replace the existing processMileageExcelFile function in main.go with:
func processMileageExcelFile(file multipart.File, filename string) (int, error) {
    return processEnhancedMileageExcelFile(file, filename)
}

// Update the route setup in setupManagerRoutes to use the enhanced handler:
func setupManagerRoutes(mux *http.ServeMux) {
    // ... existing routes ...
    
    // Replace the existing view-mileage-reports route with:
    mux.HandleFunc("/view-mileage-reports", withRecovery(requireAuth(requireRole("manager")(viewEnhancedMileageReportsHandler))))
    
    // ... rest of routes ...
}

// Update the importMileageHandler success message to be more specific:
func importMileageHandler(w http.ResponseWriter, r *http.Request) {
    // ... existing code ...
    
    // After successful import, update the success message:
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

// Add this helper function to validate vehicle IDs don't exceed database limits:
func validateVehicleID(id string) string {
    // Ensure vehicle ID doesn't exceed 20 characters (your database limit)
    if len(id) > 20 {
        // Try to intelligently shorten it
        if strings.HasPrefix(strings.ToUpper(id), "BUS") && len(id) > 20 {
            // Remove "BUS" prefix and re-add it to fit
            baseID := strings.TrimPrefix(strings.ToUpper(id), "BUS")
            if len(baseID) <= 17 {
                return "BUS" + baseID
            }
            // If still too long, truncate
            return "BUS" + baseID[:17]
        }
        // For other IDs, just truncate
        log.Printf("Warning: Vehicle ID '%s' truncated to '%s'", id, id[:20])
        return id[:20]
    }
    return id
}

// Update the database global variable declaration if needed:
var db *sql.DB // Make sure this is declared at package level

// Add this new enhanced parser function
func parseEnhancedMileageRow(row []string, sheetName string, section string) *MileageRecord {
    // Skip rows that don't have enough columns
    if len(row) < 6 {
        return nil
    }
    
    var record MileageRecord
    record.ReportMonth = sheetName
    record.ReportYear = 2024 // Default, adjust as needed
    
    // Log the row for debugging
    log.Printf("Parsing row with %d columns: %v", len(row), row)
    
    // Try to find bus ID and mileage data
    busID := ""
    var beginMiles, endMiles int
    
    // Strategy 1: Look for patterns in specific positions
    // Common Excel layouts have bus info in columns 3-5 and mileage in later columns
    
    // Find bus ID
    for i := 0; i < len(row) && i < 6; i++ {
        cell := strings.TrimSpace(row[i])
        if cell != "" && cell != "#REF!" {
            // Check if this could be a bus identifier
            if strings.Contains(strings.ToUpper(cell), "BUS") {
                busID = cell
                break
            } else if i >= 3 && i <= 5 {
                // Check if it's a number that could be a bus ID
                if num := parseInt(cell); num > 0 && num < 1000 {
                    busID = fmt.Sprintf("%d", num)
                } else if len(cell) <= 10 && !strings.Contains(cell, " ") {
                    // Short identifier without spaces
                    busID = cell
                }
            }
        }
    }
    
    // If no bus ID found yet, try alternative positions
    if busID == "" {
        for i := 2; i < len(row) && i < 7; i++ {
            cell := strings.TrimSpace(row[i])
            if cell != "" && cell != "#REF!" && len(cell) <= 20 {
                // Take the first non-empty, reasonable length value
                busID = cell
                break
            }
        }
    }
    
    if busID == "" {
        return nil
    }
    
    // Find mileage data - typically in the last columns
    mileageValues := []int{}
    for i := max(6, len(row)-10); i < len(row); i++ {
        if val := parseInt(row[i]); val > 1000 && val < 999999 {
            mileageValues = append(mileageValues, val)
        }
    }
    
    // Assign beginning and ending miles
    if len(mileageValues) >= 2 {
        // Assume first is beginning, second is ending
        beginMiles = mileageValues[0]
        endMiles = mileageValues[1]
        
        // Sanity check - ending should be greater than beginning
        if endMiles < beginMiles {
            beginMiles, endMiles = endMiles, beginMiles
        }
    } else if len(mileageValues) == 1 {
        // Only one mileage value - might be ending miles
        endMiles = mileageValues[0]
    }
    
    // Clean up and standardize bus ID
    busID = strings.TrimSpace(busID)
    if !strings.HasPrefix(strings.ToUpper(busID), "BUS") {
        busID = "BUS" + busID
    }
    busID = abbreviateBusID(busID)
    record.BusID = busID
    
    // Set mileage values
    record.BeginningMiles = beginMiles
    record.EndingMiles = endMiles
    if beginMiles > 0 && endMiles > beginMiles {
        record.TotalMiles = endMiles - beginMiles
    }
    
    // Parse other fields
    if len(row) > 0 {
        if year := parseInt(row[0]); year > 1900 && year < 2030 {
            record.BusYear = year
        }
    }
    
    if len(row) > 1 {
        record.BusMake = strings.TrimSpace(row[1])
    }
    
    if len(row) > 2 {
        record.LicensePlate = strings.TrimSpace(row[2])
    }
    
    // Location - try multiple positions
    for i := 4; i < min(7, len(row)); i++ {
        loc := strings.TrimSpace(row[i])
        if loc != "" && loc != "#REF!" && !strings.Contains(loc, "BUS") && len(loc) < 50 {
            record.LocatedAt = loc
            break
        }
    }
    
    // Add section prefix to location
    if section != "" {
        switch section {
        case "school_bus":
            if record.LocatedAt == "" {
                record.LocatedAt = "School Bus"
            } else if !strings.Contains(strings.ToLower(record.LocatedAt), "school") {
                record.LocatedAt = "School Bus - " + record.LocatedAt
            }
        case "agency_vehicle":
            if record.LocatedAt == "" {
                record.LocatedAt = "Agency Vehicle"
            } else if !strings.Contains(strings.ToLower(record.LocatedAt), "agency") {
                record.LocatedAt = "Agency - " + record.LocatedAt
            }
        case "program":
            if record.LocatedAt == "" {
                record.LocatedAt = "Program Vehicle"
            } else if !strings.Contains(strings.ToLower(record.LocatedAt), "program") {
                record.LocatedAt = "Program - " + record.LocatedAt
            }
        }
    }
    
    return &record
}

// Add this helper function if you don't have it
func max(a, b int) int {
    if a > b {
        return a
    }
    return b
}

// Add this debug function to help understand the Excel structure
func debugExcelStructure(file multipart.File, filename string) error {
    f, err := excelize.OpenReader(file)
    if err != nil {
        return fmt.Errorf("failed to open Excel file: %v", err)
    }
    defer f.Close()
    
    sheets := f.GetSheetList()
    log.Printf("\n=== EXCEL STRUCTURE DEBUG ===")
    log.Printf("File: %s", filename)
    log.Printf("Sheets: %v", sheets)
    
    for _, sheetName := range sheets {
        log.Printf("\n--- Sheet: %s ---", sheetName)
        
        rows, err := f.GetRows(sheetName)
        if err != nil {
            log.Printf("Error reading sheet: %v", err)
            continue
        }
        
        // Print first 20 rows to understand structure
        for i := 0; i < min(20, len(rows)); i++ {
            if len(rows[i]) > 0 {
                log.Printf("Row %d (%d cols): %v", i+1, len(rows[i]), rows[i])
            }
        }
    }
    
    return nil
}

func parseMileageRow(row []string, sheetName string, section string) *MileageRecord {
    // Skip rows that don't have enough columns
    if len(row) < 8 {
        return nil
    }
    
    // Try to identify the structure based on content
    // Common patterns:
    // Pattern 1: Year, Make/Model, License, Bus#, Location, Miles columns...
    // Pattern 2: Bus info in first columns, mileage data in later columns
    
    var record MileageRecord
    record.ReportMonth = sheetName
    record.ReportYear = 2024 // Default, adjust as needed
    
    // Try to find bus ID (might be labeled as "Bus #", "Unit", etc.)
    busID := ""
    var beginIdx, endIdx int = -1, -1
    
    // First pass - identify column positions
    for idx, cell := range row {
        cellTrimmed := strings.TrimSpace(cell)
        
        // Look for bus identifier
        if busID == "" && cellTrimmed != "" && cellTrimmed != "#REF!" {
            // Check if this looks like a bus number
            if strings.Contains(strings.ToUpper(cellTrimmed), "BUS") {
                busID = cellTrimmed
            } else if idx >= 3 && idx <= 5 { // Common position for bus ID
                // Could be just a number
                if _, err := strconv.Atoi(cellTrimmed); err == nil {
                    busID = cellTrimmed
                }
            }
        }
        
        // Look for mileage columns (usually towards the end)
        if idx >= len(row)-6 { // Last 6 columns likely contain mileage
            if val := parseInt(cellTrimmed); val > 1000 && val < 999999 { // Reasonable mileage range
                if beginIdx == -1 {
                    beginIdx = idx
                    record.BeginningMiles = val
                } else if endIdx == -1 && idx > beginIdx {
                    endIdx = idx
                    record.EndingMiles = val
                }
            }
        }
    }
    
    // If we didn't find bus ID in expected places, try other positions
    if busID == "" {
        // Sometimes bus info is in column 4 or 5
        for i := 3; i < min(6, len(row)); i++ {
            cell := strings.TrimSpace(row[i])
            if cell != "" && cell != "#REF!" {
                busID = cell
                break
            }
        }
    }
    
    if busID == "" {
        return nil
    }
    
    // Clean up and standardize bus ID
    busID = strings.TrimSpace(busID)
    if !strings.HasPrefix(strings.ToUpper(busID), "BUS") {
        busID = "BUS" + busID
    }
    busID = abbreviateBusID(busID)
    record.BusID = busID
    
    // Parse other fields
    if len(row) > 0 {
        // Try to extract year from first column
        if year := parseInt(row[0]); year > 1900 && year < 2030 {
            record.BusYear = year
        }
    }
    
    if len(row) > 1 {
        record.BusMake = strings.TrimSpace(row[1])
    }
    
    if len(row) > 2 {
        record.LicensePlate = strings.TrimSpace(row[2])
    }
    
    // Location - try multiple positions
    for i := 4; i < min(7, len(row)); i++ {
        loc := strings.TrimSpace(row[i])
        if loc != "" && loc != "#REF!" && !strings.Contains(loc, "BUS") {
            record.LocatedAt = loc
            break
        }
    }
    
    // If we couldn't find mileage in the standard way, try a different approach
    if record.BeginningMiles == 0 && record.EndingMiles == 0 {
        // Sometimes mileage is in specific column positions
        // Try columns 7-12 for mileage data
        for i := 7; i < min(len(row), 13); i++ {
            val := parseInt(row[i])
            if val > 1000 && val < 999999 {
                if record.BeginningMiles == 0 {
                    record.BeginningMiles = val
                } else if record.EndingMiles == 0 && val != record.BeginningMiles {
                    record.EndingMiles = val
                    break
                }
            }
        }
    }
    
    // Calculate total miles if we have both values
    if record.BeginningMiles > 0 && record.EndingMiles > record.BeginningMiles {
        record.TotalMiles = record.EndingMiles - record.BeginningMiles
    }
    
    // Add section type to location if not already present
    if section != "" && record.LocatedAt != "" {
        switch section {
        case "school_bus":
            if !strings.Contains(strings.ToLower(record.LocatedAt), "school") {
                record.LocatedAt = "School Bus - " + record.LocatedAt
            }
        case "agency_vehicle":
            if !strings.Contains(strings.ToLower(record.LocatedAt), "agency") {
                record.LocatedAt = "Agency - " + record.LocatedAt
            }
        case "program":
            if !strings.Contains(strings.ToLower(record.LocatedAt), "program") {
                record.LocatedAt = "Program - " + record.LocatedAt
            }
        }
    }
    
    log.Printf("DEBUG: Parsed row - BusID: %s, Begin: %d, End: %d, Total: %d", 
        record.BusID, record.BeginningMiles, record.EndingMiles, record.TotalMiles)
    
    return &record
}

func min(a, b int) int {
    if a < b {
        return a
    }
    return b
}

func parseInt(s string) int {
    val, _ := strconv.Atoi(strings.TrimSpace(s))
    return val
}

func insertMileageRecords(records []MileageRecord) (int, error) {
    if db == nil {
        return 0, fmt.Errorf("database not initialized")
    }
    
    log.Printf("Attempting to insert %d records into database", len(records))
    
    count := 0
    for i, record := range records {
        _, err := db.Exec(`
            INSERT INTO monthly_mileage_reports 
            (report_month, report_year, bus_year, bus_make, license_plate, 
             bus_id, located_at, beginning_miles, ending_miles, total_miles)
            VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
            ON CONFLICT (report_month, report_year, bus_id) 
            DO UPDATE SET
                bus_year = EXCLUDED.bus_year,
                bus_make = EXCLUDED.bus_make,
                license_plate = EXCLUDED.license_plate,
                located_at = EXCLUDED.located_at,
                beginning_miles = EXCLUDED.beginning_miles,
                ending_miles = EXCLUDED.ending_miles,
                total_miles = EXCLUDED.total_miles,
                updated_at = CURRENT_TIMESTAMP
        `, record.ReportMonth, record.ReportYear, record.BusYear, record.BusMake,
           record.LicensePlate, record.BusID, record.LocatedAt, record.BeginningMiles,
           record.EndingMiles, record.TotalMiles)
        
        if err != nil {
            log.Printf("Error inserting record %d (BusID=%s): %v", i+1, record.BusID, err)
        } else {
            count++
            log.Printf("Successfully inserted record %d (BusID=%s)", i+1, record.BusID)
        }
    }
    
    log.Printf("Successfully inserted %d out of %d records", count, len(records))
    
    return count, nil
}

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
    // When URL is /vehicle-maintenance/11, we need to get "11"
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
    var vehicle Vehicle
    err := db.QueryRow(`
        SELECT vehicle_id, model, description, year, tire_size, 
               license, oil_status, tire_status, status, maintenance_notes,
               serial_number, base, COALESCE(service_interval, 0)
        FROM vehicles 
        WHERE vehicle_id = $1
        LIMIT 1
    `, vehicleID).Scan(
        &vehicle.VehicleID,
        &vehicle.Model,
        &vehicle.Description,
        &vehicle.Year,
        &vehicle.TireSize,
        &vehicle.License,
        &vehicle.OilStatus,
        &vehicle.TireStatus,
        &vehicle.Status,
        &vehicle.MaintenanceNotes,
        &vehicle.SerialNumber,
        &vehicle.Base,
        &vehicle.ServiceInterval,
    )
    
    if err != nil {
        log.Printf("Error fetching vehicle info for ID %s: %v", vehicleID, err)
    }
    
    var allRecords []BusMaintenanceLog
    totalCost := 0.0
    
    // Try to get maintenance records
    vehicleNum, err := strconv.Atoi(vehicleID)
    if err == nil {
        log.Printf("Querying maintenance_records for vehicle_number: %d", vehicleNum)
        
        rows, err := db.Query(`
            SELECT vehicle_number, 
                   COALESCE(service_date::text, created_at::text, ''), 
                   COALESCE(mileage, 0),
                   COALESCE(work_description, ''),
                   COALESCE(cost, 0)
            FROM maintenance_records 
            WHERE vehicle_number = $1
            ORDER BY COALESCE(service_date, created_at) DESC
        `, vehicleNum)
        
        if err != nil {
            log.Printf("Error querying maintenance_records: %v", err)
        } else {
            defer rows.Close()
            for rows.Next() {
                var record BusMaintenanceLog
                var vehicleNum int
                var cost float64
                err := rows.Scan(&vehicleNum, &record.Date, &record.Mileage, &record.Notes, &cost)
                if err == nil {
                    record.Category = "service"
                    record.BusID = strconv.Itoa(vehicleNum)
                    allRecords = append(allRecords, record)
                    totalCost += cost
                }
            }
            log.Printf("Found %d records in maintenance_records", len(allRecords))
        }
    }
    
    // Also check service_records
    rows2, err := db.Query(`
        SELECT COALESCE(unnamed_1, ''), 
               COALESCE(unnamed_2, ''),
               COALESCE(unnamed_3, ''),
               COALESCE(unnamed_4, ''),
               COALESCE(created_at::text, '')
        FROM service_records 
        WHERE unnamed_1 = $1
        ORDER BY created_at DESC
        LIMIT 20
    `, vehicleID)
    
    if err != nil {
        log.Printf("Error querying service_records: %v", err)
    } else {
        defer rows2.Close()
        serviceCount := 0
        for rows2.Next() {
            var vehicleID, vendor, serviceNum, mileageStr, createdAt string
            err := rows2.Scan(&vehicleID, &vendor, &serviceNum, &mileageStr, &createdAt)
            if err == nil {
                mileage := 0
                if m, err := strconv.Atoi(mileageStr); err == nil {
                    mileage = m
                }
                
                dateStr := createdAt
                if len(createdAt) >= 10 {
                    dateStr = createdAt[:10]
                }
                
                record := BusMaintenanceLog{
                    BusID:    vehicleID,
                    Date:     dateStr,
                    Category: "service",
                    Notes:    fmt.Sprintf("Service by %s - Invoice #%s", vendor, serviceNum),
                    Mileage:  mileage,
                }
                allRecords = append(allRecords, record)
                serviceCount++
            }
        }
        log.Printf("Found %d records in service_records", serviceCount)
    }
    
    log.Printf("Total maintenance records for vehicle %s: %d", vehicleID, len(allRecords))
    
    // Calculate statistics
    avgCost := 0.0
    if len(allRecords) > 0 && totalCost > 0 {
        avgCost = totalCost / float64(len(allRecords))
    }
    
    recentCount := 0
    thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
    for _, record := range allRecords {
        if recordDate, err := time.Parse("2006-01-02", record.Date); err == nil {
            if recordDate.After(thirtyDaysAgo) {
                recentCount++
            }
        }
    }
    
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
        TotalRecords:       len(allRecords),
        TotalCost:          totalCost,
        AverageCost:        avgCost,
        RecentCount:        recentCount,
        Today:              time.Now().Format("2006-01-02"),
        CSRFToken:          getCSRFToken(r),
    }
    
    executeTemplate(w, "vehicle_maintenance.html", data)
}

func fleetHandler(w http.ResponseWriter, r *http.Request) {
    user := getUserFromSession(r)
    if user == nil || user.Role != "manager" {
        http.Redirect(w, r, "/", http.StatusSeeOther)
        return
    }

    // Load buses
    buses := loadBuses()
    
    // Load recent maintenance logs from PostgreSQL only
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
    
    executeTemplate(w, "fleet.html", data)
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
	driverStudents := getDriverStudents(user.Username)
	routes, _ := loadRoutes()
	
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
	data := FleetData{
		User:      getUserFromSession(r),
		Buses:     loadBuses(),
		Today:     time.Now().Format(DateFormat),
		CSRFToken: getCSRFToken(r),
	}
	
	renderTemplate(w, "fleet.html", data)
}

func companyFleetPage(w http.ResponseWriter, r *http.Request) {
	data := CompanyFleetData{
		User:      getUserFromSession(r),
		Vehicles:  loadVehicles(),
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
	
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// ============= ROUTE ASSIGNMENT HANDLERS =============

func assignRoutesPage(w http.ResponseWriter, r *http.Request) {
	assignments, _ := loadRouteAssignments()
	routes, _ := loadRoutes()
	buses := loadBuses()
	users := loadUsers()
	
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
	
	http.Redirect(w, r, "/assign-routes", http.StatusFound)
}

// ============= PROFILE HANDLERS =============

func driverProfileHandler(w http.ResponseWriter, r *http.Request) {
	driverUsername := extractIDFromPath(r.URL.Path, "/driver/")
	if driverUsername == "" {
		http.Error(w, "Driver username required", http.StatusBadRequest)
		return
	}
	
	driverLogs := getDriverLogs(driverUsername)
	
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
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":    "ok",
		"service":   "bus-fleet-management",
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

func logout(w http.ResponseWriter, r *http.Request) {
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

func userExists(username string) bool {
	users := loadUsers()
	for _, user := range users {
		if user.Username == username {
			return true
		}
	}
	return false
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

func getPendingUsers() []struct {
	Username  string
	CreatedAt string
} {
	var pendingUsers []struct {
		Username  string
		CreatedAt string
	}
	
	for _, u := range loadUsers() {
		if u.Role == RoleDriverPending {
			pendingUsers = append(pendingUsers, struct {
				Username  string
				CreatedAt string
			}{
				Username:  u.Username,
				CreatedAt: "Recently", // You could add timestamp to User struct
			})
		}
	}
	
	return pendingUsers
}

func countPendingUsers() int {
	count := 0
	for _, u := range loadUsers() {
		if u.Role == RoleDriverPending {
			count++
		}
	}
	return count
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
	users := loadUsers()
	for i, u := range users {
		if u.Username == username && u.Role == RoleDriverPending {
			users[i].Role = RoleDriver
			users[i].Status = StatusActive
			return saveUsers(users)
		}
	}
	
	return fmt.Errorf("User not found or already processed")
}

func loadRoutesWithDefault() []Route {
	routes, err := loadRoutes()
	if err != nil {
		log.Printf("Error loading routes: %v", err)
		return []Route{}
	}
	return routes
}

func getRouteAndBus(assignment *RouteAssignment) (*Route, *Bus) {
	if assignment == nil {
		return nil, nil
	}
	
	// Get route
	routes, _ := loadRoutes()
	var route *Route
	for _, r := range routes {
		if r.RouteID == assignment.RouteID {
			route = &r
			break
		}
	}
	
	// Get bus
	buses := loadBuses()
	var bus *Bus
	for _, b := range buses {
		if b.BusID == assignment.BusID {
			bus = b
			break
		}
	}
	
	return route, bus
}

func getRouteStudents(route *Route, driverUsername, period string) []Student {
	if route == nil {
		return []Student{}
	}
	
	var routeStudents []Student
	for _, s := range loadStudents() {
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
	
	return routeStudents
}

func getDriverLogForDatePeriod(driver, date, period string) *DriverLog {
	logs, _ := loadDriverLogs()
	for _, log := range logs {
		if log.Driver == driver && log.Date == date && log.Period == period {
			return &log
		}
	}
	return nil
}

func getRecentDriverLogs(driver string, limit int) []DriverLog {
	logs, _ := loadDriverLogs()
	var recentLogs []DriverLog
	for _, log := range logs {
		if log.Driver == driver {
			recentLogs = append(recentLogs, log)
			if len(recentLogs) >= limit {
				break
			}
		}
	}
	return recentLogs
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
	var targetUser *User
	for _, u := range loadUsers() {
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
	users := loadUsers()
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
		buses := loadBuses()
		for _, bus := range buses {
			if bus.BusID == vehicleID {
				vehicleInfo = bus
				break
			}
		}
	} else {
		vehicles := loadVehicles()
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
	buses := loadBuses()
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

func getDriverStudents(driverUsername string) []Student {
	var driverStudents []Student
	for _, s := range loadStudents() {
		if s.Driver == driverUsername {
			driverStudents = append(driverStudents, s)
		}
	}
	return driverStudents
}

func verifyStudentOwnership(studentID, driverUsername string) bool {
	students := loadStudents()
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
		students := loadStudents()
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
	vehicles := loadVehicles()
	
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
	routes, _ := loadRoutes()
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
	routes, _ := loadRoutes()
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
	routes, _ := loadRoutes()
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
	routes, _ := loadRoutes()
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
	students := loadStudents()
	for _, s := range students {
		if s.RouteID == routeID && s.Active {
			return fmt.Errorf("Cannot delete route that has active students assigned")
		}
	}
	
	return nil
}

func getDriverLogs(driverUsername string) []DriverLog {
	allLogs, _ := loadDriverLogs()
	var driverLogs []DriverLog
	for _, log := range allLogs {
		if log.Driver == driverUsername {
			driverLogs = append(driverLogs, log)
		}
	}
	return driverLogs
}
