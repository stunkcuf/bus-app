package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
	
	"github.com/xuri/excelize/v2"
)

// databaseStatsHandler provides database optimization statistics
func databaseStatsHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		SendError(w, ErrUnauthorized("Access denied"))
		return
	}

	optimizer := InitializeQueryOptimizer()
	if optimizer == nil {
		SendError(w, ErrInternal("Query optimizer not available", nil))
		return
	}

	ctx := r.Context()
	stats, err := optimizer.GetDatabaseStats(ctx)
	if err != nil {
		log.Printf("Failed to get database stats: %v", err)
		SendError(w, ErrDatabase("get database statistics", err))
		return
	}

	recommendations, err := optimizer.GetQueryRecommendations(ctx)
	if err != nil {
		log.Printf("Failed to get recommendations: %v", err)
	}

	response := map[string]interface{}{
		"stats":           stats,
		"recommendations": recommendations,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// optimizeDatabaseHandler performs database optimization
func optimizeDatabaseHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		SendError(w, ErrMethodNotAllowed("Only POST method allowed"))
		return
	}

	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		SendError(w, ErrUnauthorized("Access denied"))
		return
	}

	if !validateCSRF(r) {
		SendError(w, ErrForbidden("Invalid CSRF token"))
		return
	}

	optimizer := InitializeQueryOptimizer()
	if optimizer == nil {
		http.Error(w, "Query optimizer not available", http.StatusInternalServerError)
		return
	}

	// Parse request
	var req struct {
		Action string `json:"action"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	var result string
	var err error

	switch req.Action {
	case "vacuum":
		err = optimizer.VacuumDatabase(ctx, false)
		result = "Database vacuum completed"
	case "vacuum_full":
		err = optimizer.VacuumDatabase(ctx, true)
		result = "Full database vacuum completed"
	case "create_indexes":
		err = optimizer.CreateMissingIndexes(ctx)
		result = "Missing indexes created"
	default:
		http.Error(w, "Invalid action", http.StatusBadRequest)
		return
	}

	if err != nil {
		log.Printf("Optimization action %s failed: %v", req.Action, err)
		http.Error(w, fmt.Sprintf("Optimization failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": result,
	})
}

// dashboardHandler redirects to the appropriate dashboard based on user role
func dashboardHandler(w http.ResponseWriter, r *http.Request) {
	// Get username from session
	cookie, err := r.Cookie(SessionCookieName)
	if err != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	session, err := GetSecureSession(cookie.Value)
	if err != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	var user User
	err = db.Get(&user, "SELECT * FROM users WHERE username = $1", session.Username)
	if err != nil {
		http.Error(w, "User not found", http.StatusInternalServerError)
		return
	}

	// Redirect based on role
	switch user.Role {
	case "manager":
		http.Redirect(w, r, "/manager-dashboard", http.StatusSeeOther)
	case "driver":
		http.Redirect(w, r, "/driver-dashboard", http.StatusSeeOther)
	default:
		http.Error(w, "Invalid user role", http.StatusForbidden)
	}
}

// approveUsersHandler shows pending users for approval
func approveUsersHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	var pendingUsers []User

	// Get pending users - need all fields for User struct
	query := `SELECT username, password, role, status, registration_date, created_at 
	          FROM users WHERE status = 'pending' ORDER BY registration_date DESC`
	if err := db.Select(&pendingUsers, query); err != nil {
		log.Printf("Error loading pending users: %v", err)
		http.Error(w, "Failed to get pending users", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"User":         user,
		"PendingUsers": pendingUsers,
		"CSRFToken":    getSessionCSRFToken(r),
	}

	renderTemplate(w, r, "approve_users.html", data)
}

// approveUserHandler approves a pending user
func approveUserHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendError(w, ErrMethodNotAllowed("Only POST method allowed"))
		return
	}

	username := r.FormValue("username")
	if username == "" {
		http.Error(w, "Username required", http.StatusBadRequest)
		return
	}

	// Update user status
	_, err := db.Exec("UPDATE users SET status = 'active' WHERE username = $1 AND status = 'pending'", username)
	if err != nil {
		http.Error(w, "Failed to approve user", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/approve-users", http.StatusSeeOther)
}

// manageUsersHandler shows all users for management
func manageUsersHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	// Direct database query - skip cache entirely to avoid issues
	var users []User
	query := `SELECT username, password, role, status, registration_date, created_at 
	          FROM users ORDER BY created_at DESC`
	
	err := db.Select(&users, query)
	if err != nil {
		log.Printf("Error loading users from database: %v", err)
		// Don't fail completely - show page with empty user list
		users = []User{}
		
		// Try alternate query without all fields
		err2 := db.Select(&users, "SELECT * FROM users ORDER BY username")
		if err2 != nil {
			log.Printf("Alternate query also failed: %v", err2)
		}
	}
	
	log.Printf("Loaded %d users for management page", len(users))

	data := map[string]interface{}{
		"User":      user,
		"Users":     users,
		"CSRFToken": getSessionCSRFToken(r),
	}

	renderTemplate(w, r, "users.html", data)
}

// editUserHandler handles user editing
func editUserHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendError(w, ErrMethodNotAllowed("Only POST method allowed"))
		return
	}

	username := r.FormValue("username")
	role := r.FormValue("role")
	status := r.FormValue("status")

	if username == "" {
		http.Error(w, "Username required", http.StatusBadRequest)
		return
	}

	// Update user
	_, err := db.Exec("UPDATE users SET role = $1, status = $2 WHERE username = $3", role, status, username)
	if err != nil {
		http.Error(w, "Failed to update user", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/manage-users", http.StatusSeeOther)
}

// deleteUserHandler handles user deletion
func deleteUserHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendError(w, ErrMethodNotAllowed("Only POST method allowed"))
		return
	}

	username := r.FormValue("username")
	if username == "" {
		http.Error(w, "Username required", http.StatusBadRequest)
		return
	}

	// Delete user
	_, err := db.Exec("DELETE FROM users WHERE username = $1", username)
	if err != nil {
		http.Error(w, "Failed to delete user", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/manage-users", http.StatusSeeOther)
}

// importECSEHandler handles ECSE data import
func importECSEHandler(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"CSRFToken": getSessionCSRFToken(r),
	}

	renderTemplate(w, r, "import_ecse.html", data)
}

// viewECSEReportsHandler shows ECSE reports
func viewECSEReportsHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	// For now, using a simple struct that matches the template expectations
	type ECSEDisplayStudent struct {
		StudentID              string `db:"student_id"`
		FirstName              string `db:"first_name"`
		LastName               string `db:"last_name"`
		Grade                  string `db:"grade"`
		EnrollmentStatus       string `db:"enrollment_status"`
		IEPStatus              string `db:"iep_status"`
		ServiceCount           int    `db:"service_count"`
		TransportationRequired bool   `db:"transportation_required"`
		BusRoute               string `db:"bus_route"`
		ParentPhone            string `db:"parent_phone"`
	}
	
	var students []ECSEDisplayStudent
	
	query := `SELECT 
		student_id,
		first_name,
		last_name,
		COALESCE(grade, '') as grade,
		COALESCE(enrollment_status, 'Unknown') as enrollment_status,
		COALESCE(iep_status, '') as iep_status,
		0 as service_count,
		COALESCE(transportation_required, false) as transportation_required,
		COALESCE(bus_route, '') as bus_route,
		COALESCE(parent_phone, '') as parent_phone
	FROM ecse_students 
	ORDER BY last_name, first_name
	LIMIT 100`
	
	if err := db.Select(&students, query); err != nil {
		log.Printf("Error loading ECSE students: %v", err)
		students = []ECSEDisplayStudent{}
	}
	
	log.Printf("Loaded %d ECSE students for display", len(students))
	
	// Debug: Show first student if any
	if len(students) > 0 {
		log.Printf("First student: ID=%s, Name=%s %s", 
			students[0].StudentID, students[0].FirstName, students[0].LastName)
	}

	// Ensure students is never nil for template
	if students == nil {
		students = []ECSEDisplayStudent{}
	}
	
	data := map[string]interface{}{
		"User": user,
		"Data": map[string]interface{}{
			"Students":     students,
			"HasStudents":  len(students) > 0,
			"StudentCount": len(students),
			"CSRFToken":    getSessionCSRFToken(r),
		},
		"CSRFToken": getSessionCSRFToken(r),
	}

	renderTemplate(w, r, "view_ecse_reports.html", data)
}

// viewECSEStudentHandler shows individual ECSE student details
func viewECSEStudentHandler(w http.ResponseWriter, r *http.Request) {
	// Extract student ID from URL path
	path := strings.TrimPrefix(r.URL.Path, "/ecse-student/")
	studentID := strings.TrimSuffix(path, "/")

	if studentID == "" {
		http.Error(w, "Student ID required", http.StatusBadRequest)
		return
	}

	var student ECSEStudent
	var services []ECSEService
	var assessments []ECSEAssessment
	var attendance []ECSEAttendance

	// Get student details
	if err := db.Get(&student, "SELECT * FROM ecse_students WHERE student_id = $1", studentID); err != nil {
		http.Error(w, "Student not found", http.StatusNotFound)
		return
	}

	// Get services
	db.Select(&services, "SELECT * FROM ecse_services WHERE student_id = $1", studentID)

	// Get assessments
	db.Select(&assessments, "SELECT * FROM ecse_assessments WHERE student_id = $1 ORDER BY assessment_date DESC", studentID)

	// Get attendance
	db.Select(&attendance, "SELECT * FROM ecse_attendance WHERE student_id = $1 ORDER BY date DESC LIMIT 30", studentID)

	data := map[string]interface{}{
		"Student":     student,
		"Services":    services,
		"Assessments": assessments,
		"Attendance":  attendance,
		"CSRFToken":   getSessionCSRFToken(r),
	}

	renderTemplate(w, r, "view_ecse_student.html", data)
}

// editECSEStudentHandler handles editing ECSE student information
func editECSEStudentHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		// Show edit form
		studentID := r.URL.Query().Get("id")
		if studentID == "" {
			http.Error(w, "Student ID required", http.StatusBadRequest)
			return
		}

		var student ECSEStudent
		if err := db.Get(&student, "SELECT * FROM ecse_students WHERE student_id = $1", studentID); err != nil {
			http.Error(w, "Student not found", http.StatusNotFound)
			return
		}

		data := map[string]interface{}{
			"Student":   student,
			"CSRFToken": getSessionCSRFToken(r),
		}

		renderTemplate(w, r, "edit_ecse_student.html", data)
		return
	}

	if r.Method == "POST" {
		// Parse form
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			SendError(w, ErrBadRequest("Failed to parse form data"))
			return
		}

		// Validate CSRF token
		if !validateCSRF(r) {
			SendError(w, ErrForbidden("Invalid CSRF token"))
			return
		}

		// Get form values
		studentID := r.FormValue("student_id")
		firstName := r.FormValue("first_name")
		lastName := r.FormValue("last_name")
		dateOfBirth := r.FormValue("date_of_birth")
		grade := r.FormValue("grade")
		enrollmentStatus := r.FormValue("enrollment_status")
		iepStatus := r.FormValue("iep_status")
		primaryDisability := r.FormValue("primary_disability")
		serviceMinutesStr := r.FormValue("service_minutes")
		transportationRequired := r.FormValue("transportation_required") == "true"
		busRoute := r.FormValue("bus_route")
		parentName := r.FormValue("parent_name")
		parentPhone := r.FormValue("parent_phone")
		parentEmail := r.FormValue("parent_email")
		city := r.FormValue("city")
		state := r.FormValue("state")
		zipCode := r.FormValue("zip_code")
		notes := r.FormValue("notes")

		// Parse service minutes
		serviceMinutes := 0
		if serviceMinutesStr != "" {
			fmt.Sscanf(serviceMinutesStr, "%d", &serviceMinutes)
		}

		// Update student
		query := `
			UPDATE ecse_students SET
				first_name = $2, last_name = $3, date_of_birth = $4, grade = $5,
				enrollment_status = $6, iep_status = $7, primary_disability = $8,
				service_minutes = $9, transportation_required = $10, bus_route = $11,
				parent_name = $12, parent_phone = $13, parent_email = $14,
				city = $15, state = $16, zip_code = $17, notes = $18
			WHERE student_id = $1
		`

		_, err := db.Exec(query, studentID, firstName, lastName, dateOfBirth, grade,
			enrollmentStatus, iepStatus, primaryDisability, serviceMinutes,
			transportationRequired, busRoute, parentName, parentPhone, parentEmail,
			city, state, zipCode, notes)

		if err != nil {
			log.Printf("Error updating ECSE student: %v", err)
			http.Error(w, "Failed to update student", http.StatusInternalServerError)
			return
		}

		// Redirect back to student view
		http.Redirect(w, r, "/ecse-student/"+studentID, http.StatusSeeOther)
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// exportECSEHandler exports ECSE data
func exportECSEHandler(w http.ResponseWriter, r *http.Request) {
	// Get all ECSE students
	var students []ECSEStudent
	query := `SELECT * FROM ecse_students ORDER BY last_name, first_name`
	if err := db.Select(&students, query); err != nil {
		http.Error(w, "Failed to get ECSE students", http.StatusInternalServerError)
		return
	}

	// Set response headers for JSON download
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename=ecse_export.json")

	// Write JSON response
	json.NewEncoder(w).Encode(students)
}

// studentsHandler shows student management page
func studentsHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "driver" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	// Check if we should show inactive students
	showInactive := r.URL.Query().Get("show_inactive") == "true"
	
	// Get driver's assignments to filter students
	assignments, err := getDriverAssignments(user.Username)
	if err != nil {
		log.Printf("Error loading driver assignments: %v", err)
	}

	// Get all students for the driver's routes (batch loading to avoid N+1 queries)
	var routeIDs []string
	for _, assignment := range assignments {
		routeIDs = append(routeIDs, assignment.RouteID)
	}
	
	var studentsMap map[string][]Student
	var students []Student
	
	if showInactive {
		studentsMap, err = getStudentsByMultipleRoutesIncludingInactive(routeIDs)
	} else {
		studentsMap, err = getStudentsByMultipleRoutes(routeIDs)
	}
	
	if err != nil {
		log.Printf("Error loading students: %v", err)
		studentsMap = make(map[string][]Student)
	}
	
	// Flatten the map into a single list
	for _, routeStudents := range studentsMap {
		students = append(students, routeStudents...)
	}

	// Get routes from cache
	routes, err := dataCache.getRoutes()
	if err != nil {
		log.Printf("Error loading routes: %v", err)
		routes = []Route{} // Empty slice on error
	}

	data := map[string]interface{}{
		"User":         user,
		"Students":     students,
		"Routes":       routes,
		"Assignments":  assignments,
		"ShowInactive": showInactive,
		"CSRFToken":    getSessionCSRFToken(r),
	}

	renderTemplate(w, r, "students.html", data)
}

// assignRoutesHandler handles route assignment page
func assignRoutesHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	// Get data from cache
	allUsers, err := dataCache.getUsers()
	if err != nil {
		log.Printf("Error loading users: %v", err)
	}

	// Filter for active drivers
	var drivers []User
	for _, u := range allUsers {
		if u.Role == "driver" && u.Status == "active" {
			drivers = append(drivers, u)
		}
	}

	buses, err := dataCache.getBuses()
	if err != nil {
		log.Printf("Error loading buses: %v", err)
	}

	// Filter for active buses
	var activeBuses []Bus
	for _, b := range buses {
		if b.Status == "active" {
			activeBuses = append(activeBuses, b)
		}
	}

	routes, err := dataCache.getRoutes()
	if err != nil {
		log.Printf("Error loading routes: %v", err)
	}

	// Get current assignments
	assignments, err := getRouteAssignments()
	if err != nil {
		log.Printf("Error loading assignments: %v", err)
		assignments = []RouteAssignment{} // Empty slice on error
	}

	// Get student counts per route
	studentCounts, err := getStudentCountsByRoute()
	if err != nil {
		log.Printf("Error loading student counts: %v", err)
		studentCounts = make(map[string]int)
	}

	data := map[string]interface{}{
		"User":          user,
		"Drivers":       drivers,
		"Buses":         activeBuses,
		"Routes":        routes,
		"Assignments":   assignments,
		"StudentCounts": studentCounts,
		"CSRFToken":     getSessionCSRFToken(r),
	}

	renderTemplate(w, r, "assign_routes.html", data)
}

// importMileageHandler handles mileage import page
func importMileageHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	if r.Method == "GET" {
		data := map[string]interface{}{
			"User": user,
			"Data": map[string]interface{}{
				"CSRFToken": getSessionCSRFToken(r),
			},
		}
		renderTemplate(w, r, "import_mileage_simple.html", data)
		return
	}

	if r.Method == "POST" {
		// Parse the multipart form (10MB max)
		err := r.ParseMultipartForm(10 << 20)
		if err != nil {
			SendError(w, ErrBadRequest("Failed to parse form data"))
			return
		}

		// Validate CSRF token
		if !validateCSRF(r) {
			SendError(w, ErrForbidden("Invalid CSRF token"))
			return
		}

		// Get the file from form
		file, header, err := r.FormFile("excel_file")
		if err != nil {
			log.Printf("Error getting file: %v", err)
			data := map[string]interface{}{
				"User": user,
				"Data": map[string]interface{}{
					"CSRFToken": getSessionCSRFToken(r),
					"Error":     "Please select a file to upload",
				},
			}
			renderTemplate(w, r, "import_mileage_simple.html", data)
			return
		}
		defer file.Close()

		// Process the Excel file
		log.Printf("Processing mileage file: %s", header.Filename)
		recordsImported, err := processEnhancedMileageExcelFile(file, header.Filename)
		
		if err != nil {
			log.Printf("Error processing mileage file: %v", err)
			data := map[string]interface{}{
				"User": user,
				"Data": map[string]interface{}{
					"CSRFToken": getSessionCSRFToken(r),
					"Error":     fmt.Sprintf("Error processing file: %v", err),
				},
			}
			renderTemplate(w, r, "import_mileage_simple.html", data)
			return
		}

		// Clear the data cache to reflect new imports
		if dataCache != nil {
			dataCache.clear()
		}

		// Show success message
		data := map[string]interface{}{
			"User": user,
			"Data": map[string]interface{}{
				"CSRFToken": getSessionCSRFToken(r),
				"Success":   fmt.Sprintf("Successfully imported %d mileage records", recordsImported),
			},
		}
		renderTemplate(w, r, "import_mileage_simple.html", data)
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// viewMileageReportsHandler shows mileage reports
func viewMileageReportsHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("viewMileageReportsHandler called for %s", r.URL.Path)
	
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	var reports []MonthlyMileageReport

	// Get mileage reports from monthly_mileage_reports table
	query := `SELECT * FROM monthly_mileage_reports ORDER BY report_year DESC, report_month DESC, bus_id`
	if err := db.Select(&reports, query); err != nil {
		log.Printf("Error loading mileage reports: %v", err)
		// Return empty list instead of error
		reports = []MonthlyMileageReport{}
	}

	// Get summary from driver logs for current month
	var currentMonthTotal float64
	var vehicleCount int
	now := time.Now()
	err := db.QueryRow(`
		SELECT COALESCE(SUM(mileage), 0), COUNT(DISTINCT bus_id)
		FROM driver_logs
		WHERE EXTRACT(YEAR FROM date::date) = $1 
		AND EXTRACT(MONTH FROM date::date) = $2
	`, now.Year(), int(now.Month())).Scan(&currentMonthTotal, &vehicleCount)
	
	if err != nil {
		log.Printf("Error getting driver log summary: %v", err)
		currentMonthTotal = 0
		vehicleCount = 0
	}

	// Calculate totals
	var totalMiles int
	activeVehicles := make(map[string]bool)
	for _, report := range reports {
		totalMiles += report.TotalMiles
		if report.BusID != "" {
			activeVehicles[report.BusID] = true
		}
	}

	// Simple pagination
	page := 1
	perPage := 50
	totalPages := (len(reports) + perPage - 1) / perPage
	
	// Get page from query
	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	// Slice reports for current page
	start := (page - 1) * perPage
	end := start + perPage
	if end > len(reports) {
		end = len(reports)
	}
	pagedReports := reports[start:end]

	data := map[string]interface{}{
		"Title":             "Monthly Mileage Reports",
		"User":              user,
		"Reports":           pagedReports,
		"TotalReports":      len(reports),
		"TotalMiles":        totalMiles,
		"ActiveVehicles":    len(activeVehicles),
		"CurrentMonthTotal": currentMonthTotal,
		"VehicleCount":      vehicleCount,
		"CurrentMonth":      now.Format("January 2006"),
		"CSRFToken":         getSessionCSRFToken(r),
		"Pagination": map[string]interface{}{
			"Page":       page,
			"PerPage":    perPage,
			"TotalPages": totalPages,
			"HasPrev":    page > 1,
			"HasNext":    page < totalPages,
			"Pages":      generatePageNumbers(page, totalPages),
		},
	}

	log.Printf("Rendering monthly_mileage_reports.html with %d reports", len(reports))
	renderTemplate(w, r, "monthly_mileage_reports.html", data)
}

// mileageReportGeneratorHandler shows mileage report generator
func mileageReportGeneratorHandler(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"CSRFToken": getSessionCSRFToken(r),
	}

	renderTemplate(w, r, "mileage-report-generator.html", data)
}

// driverProfileHandler shows driver profile
func driverProfileHandler(w http.ResponseWriter, r *http.Request) {
	// Get username from session
	cookie, err := r.Cookie(SessionCookieName)
	if err != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	session, err := GetSecureSession(cookie.Value)
	if err != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	var driver User
	var recentLogs []DriverLog

	// Get driver info
	if err := db.Get(&driver, "SELECT * FROM users WHERE username = $1", session.Username); err != nil {
		http.Error(w, "Driver not found", http.StatusNotFound)
		return
	}

	// Get recent logs
	query := `SELECT * FROM driver_logs WHERE driver = $1 ORDER BY date DESC, period DESC LIMIT 10`
	db.Select(&recentLogs, query, session.Username)

	data := map[string]interface{}{
		"Driver":     driver,
		"RecentLogs": recentLogs,
		"CSRFToken":  getSessionCSRFToken(r),
	}

	renderTemplate(w, r, "driver_profile.html", data)
}

// assignRouteHandler assigns a route to a driver
func assignRouteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendError(w, ErrMethodNotAllowed("Only POST method allowed"))
		return
	}

	driver := r.FormValue("driver")
	busID := r.FormValue("bus_id")
	routeID := r.FormValue("route_id")

	if driver == "" || busID == "" || routeID == "" {
		http.Error(w, "All fields required", http.StatusBadRequest)
		return
	}

	_, err := db.Exec(`INSERT INTO route_assignments (driver, bus_id, route_id) 
		VALUES ($1, $2, $3) ON CONFLICT (driver, route_id) DO UPDATE SET bus_id = $2`,
		driver, busID, routeID)
	if err != nil {
		http.Error(w, "Failed to assign route", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/assign-routes", http.StatusSeeOther)
}

// unassignRouteHandler removes a route assignment
func unassignRouteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendError(w, ErrMethodNotAllowed("Only POST method allowed"))
		return
	}

	id := r.FormValue("id")
	if id == "" {
		http.Error(w, "Assignment ID required", http.StatusBadRequest)
		return
	}

	_, err := db.Exec("DELETE FROM route_assignments WHERE id = $1", id)
	if err != nil {
		http.Error(w, "Failed to unassign route", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/assign-routes", http.StatusSeeOther)
}

// addRouteHandler adds a new route
func addRouteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendError(w, ErrMethodNotAllowed("Only POST method allowed"))
		return
	}

	routeID := r.FormValue("route_id")
	routeName := r.FormValue("route_name")
	description := r.FormValue("description")

	if routeID == "" || routeName == "" {
		http.Error(w, "Route ID and name required", http.StatusBadRequest)
		return
	}

	_, err := db.Exec("INSERT INTO routes (route_id, route_name, description) VALUES ($1, $2, $3)",
		routeID, routeName, description)
	if err != nil {
		http.Error(w, "Failed to add route", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/assign-routes", http.StatusSeeOther)
}

// editRouteHandler edits an existing route
func editRouteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendError(w, ErrMethodNotAllowed("Only POST method allowed"))
		return
	}

	routeID := r.FormValue("route_id")
	routeName := r.FormValue("route_name")
	description := r.FormValue("description")

	if routeID == "" {
		http.Error(w, "Route ID required", http.StatusBadRequest)
		return
	}

	_, err := db.Exec("UPDATE routes SET route_name = $1, description = $2 WHERE route_id = $3",
		routeName, description, routeID)
	if err != nil {
		http.Error(w, "Failed to update route", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/assign-routes", http.StatusSeeOther)
}

// deleteRouteHandler deletes a route
func deleteRouteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendError(w, ErrMethodNotAllowed("Only POST method allowed"))
		return
	}

	routeID := r.FormValue("route_id")
	if routeID == "" {
		http.Error(w, "Route ID required", http.StatusBadRequest)
		return
	}

	_, err := db.Exec("DELETE FROM routes WHERE route_id = $1", routeID)
	if err != nil {
		http.Error(w, "Failed to delete route", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/assign-routes", http.StatusSeeOther)
}

// exportMileageHandler exports mileage data
func exportMileageHandler(w http.ResponseWriter, r *http.Request) {
	var reports []MileageReport

	query := `SELECT * FROM mileage_reports ORDER BY year DESC, month DESC, vehicle_id`
	if err := db.Select(&reports, query); err != nil {
		http.Error(w, "Failed to get mileage reports", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename=mileage_export.json")

	json.NewEncoder(w).Encode(reports)
}

// addStudentWizardHandler displays the add student wizard
func addStudentWizardHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	// Load routes and drivers for the wizard
	routes, _ := dataCache.getRoutes()
	users, _ := dataCache.getUsers()
	
	// Filter to get only drivers
	var drivers []User
	for _, user := range users {
		if user.Role == "driver" && user.Status == "active" {
			drivers = append(drivers, user)
		}
	}

	data := map[string]interface{}{
		"User":      user,
		"CSRFToken": getSessionCSRFToken(r),
		"Routes":    routes,
		"Drivers":   drivers,
	}
	renderTemplate(w, r, "add_student_wizard.html", data)
}

// addStudentHandler adds a new student
func addStudentHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendError(w, ErrMethodNotAllowed("Only POST method allowed"))
		return
	}

	studentID := r.FormValue("student_id")
	name := r.FormValue("name")
	routeID := r.FormValue("route_id")

	if studentID == "" || name == "" {
		http.Error(w, "Student ID and name required", http.StatusBadRequest)
		return
	}

	_, err := db.Exec(`INSERT INTO students (student_id, name, route_id) VALUES ($1, $2, $3)`,
		studentID, name, routeID)
	if err != nil {
		http.Error(w, "Failed to add student", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/students", http.StatusSeeOther)
}

// editStudentHandler edits student information
func editStudentHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendError(w, ErrMethodNotAllowed("Only POST method allowed"))
		return
	}

	studentID := r.FormValue("student_id")
	name := r.FormValue("name")
	routeID := r.FormValue("route_id")

	if studentID == "" {
		http.Error(w, "Student ID required", http.StatusBadRequest)
		return
	}

	_, err := db.Exec("UPDATE students SET name = $1, route_id = $2 WHERE student_id = $3",
		name, routeID, studentID)
	if err != nil {
		http.Error(w, "Failed to update student", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/students", http.StatusSeeOther)
}

// removeStudentHandler removes a student
func removeStudentHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendError(w, ErrMethodNotAllowed("Only POST method allowed"))
		return
	}

	studentID := r.FormValue("student_id")
	if studentID == "" {
		http.Error(w, "Student ID required", http.StatusBadRequest)
		return
	}

	_, err := db.Exec("DELETE FROM students WHERE student_id = $1", studentID)
	if err != nil {
		http.Error(w, "Failed to remove student", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/students", http.StatusSeeOther)
}

// addBusWizardHandler displays the add bus wizard
func addBusWizardHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	data := map[string]interface{}{
		"User":      user,
		"CSRFToken": getSessionCSRFToken(r),
	}
	renderTemplate(w, r, "add_bus_wizard.html", data)
}

// addBusHandler handles adding a new bus to the fleet
func addBusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		SendError(w, ErrMethodNotAllowed("Only POST method allowed"))
		return
	}

	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	// Parse form for multipart data
	err := r.ParseMultipartForm(10 << 20) // 10MB max
	if err != nil {
		SendError(w, ErrBadRequest("Failed to parse form data"))
		return
	}

	// Validate CSRF token
	if !validateCSRF(r) {
		SendError(w, ErrForbidden("Invalid CSRF token"))
		return
	}

	// Get form values
	busID := r.FormValue("bus_id")
	model := r.FormValue("model")
	capacityStr := r.FormValue("capacity")
	status := r.FormValue("status")

	// Validate required fields
	if busID == "" || model == "" || capacityStr == "" {
		SendError(w, ErrValidation("Missing required fields"))
		return
	}

	// Parse capacity
	var capacity int
	_, err = fmt.Sscanf(capacityStr, "%d", &capacity)
	if err != nil || capacity <= 0 {
		http.Error(w, "Invalid capacity", http.StatusBadRequest)
		return
	}

	// Set default status if not provided
	if status == "" {
		status = "active"
	}

	// Create bus in consolidated fleet_vehicles table
	query := `
		INSERT INTO fleet_vehicles (vehicle_id, vehicle_type, model, capacity, status, oil_status, tire_status)
		VALUES ($1, 'bus', $2, $3, $4, 'good', 'good')
	`

	_, err = db.Exec(query, busID, model, capacity, status)
	if err != nil && strings.Contains(err.Error(), "fleet_vehicles") {
		// Try to insert into old buses table as fallback
		fallbackQuery := `
			INSERT INTO buses (bus_id, model, capacity, status, oil_status, tire_status)
			VALUES ($1, $2, $3, $4, 'good', 'good')
		`
		_, err = db.Exec(fallbackQuery, busID, model, capacity, status)
	}
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			http.Error(w, "Bus ID already exists", http.StatusConflict)
			return
		}
		log.Printf("Error adding bus: %v", err)
		http.Error(w, "Failed to add bus", http.StatusInternalServerError)
		return
	}

	// Invalidate cache
	dataCache.invalidateBuses()

	// Redirect to fleet page
	http.Redirect(w, r, "/fleet", http.StatusSeeOther)
}

// routeAssignmentWizardHandler displays the route assignment wizard
func routeAssignmentWizardHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	// Get routes, drivers, and buses from cache
	routes, _ := dataCache.getRoutes()
	allusers, _ := dataCache.getUsers()
	buses, _ := dataCache.getBuses()
	
	// Filter for active drivers
	var drivers []User
	for _, u := range allusers {
		if u.Role == "driver" && u.Status == "active" {
			drivers = append(drivers, u)
		}
	}
	
	// Get current assignments to show conflicts
	assignments, _ := getRouteAssignments()
	assignmentMap := make(map[string]RouteAssignment)
	for _, a := range assignments {
		assignmentMap[a.Driver] = a
		assignmentMap[a.BusID] = a
	}
	
	// Add assignment info to drivers and buses
	for i := range drivers {
		if assignment, exists := assignmentMap[drivers[i].Username]; exists {
			drivers[i].Assignment = &assignment
		}
	}
	
	for i := range buses {
		if assignment, exists := assignmentMap[buses[i].BusID]; exists {
			buses[i].Assignment = &assignment
		}
	}

	data := map[string]interface{}{
		"User":      user,
		"Routes":    routes,
		"Drivers":   drivers,
		"Buses":     buses,
		"CSRFToken": getSessionCSRFToken(r),
	}

	renderTemplate(w, r, "route_assignment_wizard.html", data)
}

// assignRouteWizardHandler handles the wizard form submission
func assignRouteWizardHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendError(w, ErrMethodNotAllowed("Only POST method allowed"))
		return
	}

	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	// Parse form
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		SendError(w, ErrBadRequest("Failed to parse form data"))
		return
	}

	// Validate CSRF token
	if !validateCSRF(r) {
		SendError(w, ErrForbidden("Invalid CSRF token"))
		return
	}

	// Get form values
	routeID := r.FormValue("route_id")
	driver := r.FormValue("driver")
	busID := r.FormValue("bus_id")
	notes := r.FormValue("notes")

	if routeID == "" || driver == "" || busID == "" {
		http.Error(w, "All fields are required", http.StatusBadRequest)
		return
	}

	// Begin transaction
	tx, err := db.Begin()
	if err != nil {
		log.Printf("Failed to begin transaction: %v", err)
		http.Error(w, "Failed to process assignment", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	// Remove any existing assignments for this driver or bus
	_, err = tx.Exec("DELETE FROM route_assignments WHERE driver = $1 OR bus_id = $2", driver, busID)
	if err != nil {
		log.Printf("Failed to remove existing assignments: %v", err)
		http.Error(w, "Failed to process assignment", http.StatusInternalServerError)
		return
	}

	// Create new assignment
	_, err = tx.Exec(`
		INSERT INTO route_assignments (driver, bus_id, route_id, notes) 
		VALUES ($1, $2, $3, $4)`,
		driver, busID, routeID, notes)
	if err != nil {
		log.Printf("Failed to create assignment: %v", err)
		http.Error(w, "Failed to create assignment", http.StatusInternalServerError)
		return
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		log.Printf("Failed to commit transaction: %v", err)
		http.Error(w, "Failed to save assignment", http.StatusInternalServerError)
		return
	}

	// Redirect to assignments page
	http.Redirect(w, r, "/assign-routes", http.StatusSeeOther)
}

// maintenanceWizardHandler displays the maintenance logging wizard
func maintenanceWizardHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	// Get buses and vehicles from cache
	buses, _ := dataCache.getBuses()
	vehicles, _ := dataCache.getVehicles()

	data := map[string]interface{}{
		"User":      user,
		"Buses":     buses,
		"Vehicles":  vehicles,
		"Today":     time.Now().Format("2006-01-02"),
		"CSRFToken": getSessionCSRFToken(r),
	}

	renderTemplate(w, r, "maintenance_wizard.html", data)
}

// saveMaintenanceWizardHandler handles the wizard form submission
func saveMaintenanceWizardHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendError(w, ErrMethodNotAllowed("Only POST method allowed"))
		return
	}

	user := getUserFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	// Parse form
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		SendError(w, ErrBadRequest("Failed to parse form data"))
		return
	}

	// Validate CSRF token
	if !validateCSRF(r) {
		SendError(w, ErrForbidden("Invalid CSRF token"))
		return
	}

	// Get form values
	vehicleType := r.FormValue("vehicle_type")
	busID := r.FormValue("bus_id")
	vehicleID := r.FormValue("vehicle_id")
	category := r.FormValue("category")
	date := r.FormValue("date")
	mileageStr := r.FormValue("mileage")
	notes := r.FormValue("notes")
	costStr := r.FormValue("cost")
	// performedBy := r.FormValue("performed_by") // Not used yet
	// invoiceNumber := r.FormValue("invoice_number") // Not used yet

	// Validate required fields
	if vehicleType == "" || category == "" || date == "" || mileageStr == "" || notes == "" {
		SendError(w, ErrValidation("Missing required fields"))
		return
	}

	// Parse numeric values
	mileage, err := strconv.Atoi(mileageStr)
	if err != nil {
		http.Error(w, "Invalid mileage value", http.StatusBadRequest)
		return
	}

	cost := 0.0
	if costStr != "" {
		cost, _ = strconv.ParseFloat(costStr, 64)
	}

	// Determine which vehicle ID to use
	var targetID string
	if vehicleType == "bus" {
		targetID = busID
	} else {
		targetID = vehicleID
	}

	if targetID == "" {
		http.Error(w, "No vehicle selected", http.StatusBadRequest)
		return
	}

	// Save maintenance record to consolidated maintenance_records table
	// Combine category and notes for work_description
	workDescription := category
	if notes != "" {
		workDescription = category + ": " + notes
	}
	
	_, err = db.Exec(`
		INSERT INTO maintenance_records (vehicle_id, service_date, work_description, mileage, cost, created_at, updated_at) 
		VALUES ($1, $2, $3, $4, $5, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
		targetID, date, workDescription, mileage, cost)

	if err != nil {
		log.Printf("Failed to save maintenance record: %v", err)
		http.Error(w, "Failed to save maintenance record", http.StatusInternalServerError)
		return
	}

	// Update vehicle status if needed
	if category == "oil" {
		// Update in consolidated fleet_vehicles table first
		_, err := db.Exec("UPDATE fleet_vehicles SET oil_status = 'good' WHERE vehicle_id = $1 AND vehicle_type = $2", targetID, vehicleType)
		if err != nil {
			// Fallback to old tables
			if vehicleType == "bus" {
				db.Exec("UPDATE buses SET oil_status = 'good' WHERE bus_id = $1", targetID)
				dataCache.invalidateBuses()
			} else {
				db.Exec("UPDATE vehicles SET oil_status = 'good' WHERE vehicle_id = $1", targetID)
				dataCache.invalidateVehicles()
			}
		}
	} else if category == "tires" {
		// Update in consolidated fleet_vehicles table first
		_, err := db.Exec("UPDATE fleet_vehicles SET tire_status = 'good' WHERE vehicle_id = $1 AND vehicle_type = $2", targetID, vehicleType)
		if err != nil {
			// Fallback to old tables
			if vehicleType == "bus" {
				db.Exec("UPDATE buses SET tire_status = 'good' WHERE bus_id = $1", targetID)
				dataCache.invalidateBuses()
			} else {
				db.Exec("UPDATE vehicles SET tire_status = 'good' WHERE vehicle_id = $1", targetID)
				dataCache.invalidateVehicles()
			}
		}
	}

	// Redirect based on vehicle type
	if vehicleType == "bus" {
		http.Redirect(w, r, "/fleet", http.StatusSeeOther)
	} else {
		http.Redirect(w, r, "/company-fleet", http.StatusSeeOther)
	}
}

// importDataWizardHandler displays the import data wizard
func importDataWizardHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	data := map[string]interface{}{
		"User":      user,
		"CSRFToken": getSessionCSRFToken(r),
	}

	renderTemplate(w, r, "import_data_wizard.html", data)
}

// importAnalyzeHandler analyzes uploaded Excel file
func importAnalyzeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendError(w, ErrMethodNotAllowed("Only POST method allowed"))
		return
	}

	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		SendError(w, ErrUnauthorized("Access denied"))
		return
	}

	// Parse multipart form
	err := r.ParseMultipartForm(10 << 20) // 10MB
	if err != nil {
		SendError(w, ErrBadRequest("Failed to parse form data"))
		return
	}

	// Get file
	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "No file uploaded", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Get import type
	importType := r.FormValue("type")
	if importType == "" {
		http.Error(w, "Import type required", http.StatusBadRequest)
		return
	}

	// Save file temporarily
	tempFile, err := saveUploadedFile(file)
	if err != nil {
		log.Printf("Error saving uploaded file: %v", err)
		http.Error(w, "Failed to save file", http.StatusInternalServerError)
		return
	}
	defer tempFile.Close()

	// Open Excel file
	f, err := excelize.OpenFile(tempFile.Name())
	if err != nil {
		log.Printf("Error opening Excel file: %v", err)
		http.Error(w, "Failed to open Excel file", http.StatusBadRequest)
		return
	}
	defer f.Close()

	// Get sheets
	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		http.Error(w, "Excel file has no sheets", http.StatusBadRequest)
		return
	}

	// Analyze first sheet by default
	sheetName := sheets[0]
	rows, err := f.GetRows(sheetName)
	if err != nil {
		log.Printf("Error reading sheet %s: %v", sheetName, err)
		http.Error(w, "Failed to read Excel sheet", http.StatusInternalServerError)
		return
	}

	// Get columns from first row
	var columns []string
	if len(rows) > 0 {
		columns = rows[0]
	}

	// Create file ID for tracking
	fileID := fmt.Sprintf("import_%s_%d", importType, time.Now().Unix())
	
	// Store file path in session for later use
	sessionToken := getSessionToken(r)
	if sessionToken != "" {
		session, err := GetSecureSession(sessionToken)
		if err == nil && session != nil {
			if session.ImportFiles == nil {
				session.ImportFiles = make(map[string]string)
			}
			session.ImportFiles[fileID] = tempFile.Name()
			// Update session in the store
			if sessionManager != nil {
				sessionManager.store.Set(sessionToken, session)
			}
		}
	}

	response := map[string]interface{}{
		"file_id":      fileID,
		"import_type":  importType,
		"sheets":       sheets,
		"sheet":        sheetName,
		"columns":      columns,
		"rows":         len(rows) - 1, // Exclude header row
		"total_rows":   len(rows),
		"preview_data": getPreviewData(rows, 5), // First 5 rows for preview
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// importValidateHandler validates the data with column mappings
func importValidateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendError(w, ErrMethodNotAllowed("Only POST method allowed"))
		return
	}

	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		SendError(w, ErrUnauthorized("Access denied"))
		return
	}

	var req struct {
		Type     string            `json:"type"`
		Mappings map[string]string `json:"mappings"`
		FileID   string            `json:"file_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Get file from session
	session, err := GetSecureSession(getSessionToken(r))
	if err != nil || session == nil || session.ImportFiles == nil {
		http.Error(w, "Session expired or file not found", http.StatusBadRequest)
		return
	}

	filePath, exists := session.ImportFiles[req.FileID]
	if !exists {
		http.Error(w, "Import file not found", http.StatusBadRequest)
		return
	}

	// Open Excel file
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		http.Error(w, "Failed to open import file", http.StatusInternalServerError)
		return
	}
	defer f.Close()

	// Get first sheet
	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		http.Error(w, "No sheets found", http.StatusBadRequest)
		return
	}

	rows, err := f.GetRows(sheets[0])
	if err != nil {
		http.Error(w, "Failed to read sheet", http.StatusInternalServerError)
		return
	}

	// Validate based on import type
	validationResult := validateImportData(req.Type, rows, req.Mappings)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(validationResult)
}

// importExecuteHandler performs the actual import
func importExecuteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendError(w, ErrMethodNotAllowed("Only POST method allowed"))
		return
	}

	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		SendError(w, ErrUnauthorized("Access denied"))
		return
	}

	var req struct {
		Type           string            `json:"type"`
		FileID         string            `json:"file_id"`
		Mappings       map[string]string `json:"mappings"`
		SkipDuplicates bool              `json:"skip_duplicates"`
		UpdateExisting bool              `json:"update_existing"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Get file from session
	session, err := GetSecureSession(getSessionToken(r))
	if err != nil || session == nil || session.ImportFiles == nil {
		http.Error(w, "Session expired or file not found", http.StatusBadRequest)
		return
	}

	filePath, exists := session.ImportFiles[req.FileID]
	if !exists {
		http.Error(w, "Import file not found", http.StatusBadRequest)
		return
	}

	// Open Excel file
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		http.Error(w, "Failed to open import file", http.StatusInternalServerError)
		return
	}
	defer f.Close()

	// Get first sheet
	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		http.Error(w, "No sheets found", http.StatusBadRequest)
		return
	}

	rows, err := f.GetRows(sheets[0])
	if err != nil {
		http.Error(w, "Failed to read sheet", http.StatusInternalServerError)
		return
	}

	// Execute import based on type
	var result ImportResult
	
	switch req.Type {
	case "student":
		result = importStudentData(rows, req.Mappings, req.SkipDuplicates, req.UpdateExisting)
	case "vehicle":
		result = importVehicleData(rows, req.Mappings, req.SkipDuplicates, req.UpdateExisting)
	case "ecse":
		result = importECSEData(rows, req.Mappings, req.SkipDuplicates, req.UpdateExisting)
	case "mileage":
		result = importMileageData(rows, req.Mappings, req.SkipDuplicates, req.UpdateExisting)
	default:
		http.Error(w, "Invalid import type", http.StatusBadRequest)
		return
	}

	// Clean up temporary file
	os.Remove(filePath)
	delete(session.ImportFiles, req.FileID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// cacheStatsHandler provides cache performance statistics
func cacheStatsHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		SendError(w, ErrUnauthorized("Access denied"))
		return
	}

	stats := map[string]interface{}{
		"data_cache":  dataCache.getStats(),
		"query_cache": queryCache.Stats(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// Helper functions for import handlers

// saveUploadedFile saves an uploaded file to a temporary location
func saveUploadedFile(file multipart.File) (*os.File, error) {
	// Create temp file
	tempFile, err := os.CreateTemp("", "import_*.xlsx")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}

	// Copy file content
	_, err = io.Copy(tempFile, file)
	if err != nil {
		tempFile.Close()
		os.Remove(tempFile.Name())
		return nil, fmt.Errorf("failed to save file: %w", err)
	}

	// Reset file position
	tempFile.Seek(0, 0)
	return tempFile, nil
}

// getPreviewData extracts preview rows from Excel data
func getPreviewData(rows [][]string, limit int) []map[string]string {
	if len(rows) < 2 {
		return []map[string]string{}
	}

	headers := rows[0]
	preview := make([]map[string]string, 0, limit)

	for i := 1; i < len(rows) && i <= limit; i++ {
		row := rows[i]
		rowMap := make(map[string]string)
		
		for j, header := range headers {
			if j < len(row) {
				rowMap[header] = row[j]
			} else {
				rowMap[header] = ""
			}
		}
		
		preview = append(preview, rowMap)
	}

	return preview
}

// getSessionToken extracts session token from request
func getSessionToken(r *http.Request) string {
	cookie, err := r.Cookie(SessionCookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}

// validateImportData validates import data based on type
func validateImportData(importType string, rows [][]string, mappings map[string]string) map[string]interface{} {
	result := map[string]interface{}{
		"total_records": 0,
		"valid_records": 0,
		"warnings":      []string{},
		"errors":        []string{},
		"preview":       []map[string]string{},
	}

	if len(rows) < 2 {
		result["errors"] = append(result["errors"].([]string), "No data rows found")
		return result
	}

	headers := rows[0]
	dataRows := rows[1:]
	result["total_records"] = len(dataRows)

	// Validate based on type
	switch importType {
	case "student":
		result = validateStudentImport(headers, dataRows, mappings)
	case "vehicle":
		result = validateVehicleImport(headers, dataRows, mappings)
	case "ecse":
		result = validateECSEImport(headers, dataRows, mappings)
	case "mileage":
		result = validateMileageImport(headers, dataRows, mappings)
	}

	return result
}

// Import validation functions
func validateStudentImport(headers []string, rows [][]string, mappings map[string]string) map[string]interface{} {
	validCount := 0
	warnings := []string{}
	errors := []string{}
	preview := []map[string]string{}

	for i, row := range rows {
		if i < 5 { // Preview first 5 rows
			rowMap := make(map[string]string)
			for j, header := range headers {
				if j < len(row) {
					rowMap[header] = row[j]
				}
			}
			preview = append(preview, rowMap)
		}

		// Basic validation
		studentID := getValueByMapping(row, headers, mappings, "student_id")
		name := getValueByMapping(row, headers, mappings, "name")

		if studentID == "" {
			errors = append(errors, fmt.Sprintf("Row %d: Missing student ID", i+2))
			continue
		}
		if name == "" {
			warnings = append(warnings, fmt.Sprintf("Row %d: Missing student name", i+2))
		}

		validCount++
	}

	return map[string]interface{}{
		"total_records": len(rows),
		"valid_records": validCount,
		"warnings":      warnings,
		"errors":        errors,
		"preview":       preview,
	}
}

func validateVehicleImport(headers []string, rows [][]string, mappings map[string]string) map[string]interface{} {
	// Similar structure to student validation
	return validateStudentImport(headers, rows, mappings)
}

func validateECSEImport(headers []string, rows [][]string, mappings map[string]string) map[string]interface{} {
	// Similar structure to student validation
	return validateStudentImport(headers, rows, mappings)
}

func validateMileageImport(headers []string, rows [][]string, mappings map[string]string) map[string]interface{} {
	// Similar structure to student validation
	return validateStudentImport(headers, rows, mappings)
}

// Import execution functions
func importStudentData(rows [][]string, mappings map[string]string, skipDuplicates bool, updateExisting bool) ImportResult {
	result := ImportResult{
		TotalRows:    len(rows) - 1, // Exclude header
		StartTime:    time.Now(),
	}

	if len(rows) < 2 {
		result.Summary = "No data rows to import"
		return result
	}

	headers := rows[0]
	tx, err := db.Begin()
	if err != nil {
		result.Errors = append(result.Errors, ImportError{
			Row:      0,
			Error:    "Failed to start transaction",
			Severity: "error",
		})
		return result
	}
	defer tx.Rollback()

	for i, row := range rows[1:] {
		studentID := getValueByMapping(row, headers, mappings, "student_id")
		name := getValueByMapping(row, headers, mappings, "name")
		routeID := getValueByMapping(row, headers, mappings, "route_id")

		if studentID == "" {
			result.FailedRows++
			continue
		}

		// Check if exists
		var exists bool
		err := tx.QueryRow("SELECT EXISTS(SELECT 1 FROM students WHERE student_id = $1)", studentID).Scan(&exists)
		if err != nil {
			result.FailedRows++
			continue
		}

		if exists && skipDuplicates {
			result.ProcessedRows++
			continue
		}

		if exists && updateExisting {
			_, err = tx.Exec("UPDATE students SET name = $2, route_id = $3 WHERE student_id = $1",
				studentID, name, routeID)
		} else {
			_, err = tx.Exec("INSERT INTO students (student_id, name, route_id, active) VALUES ($1, $2, $3, true)",
				studentID, name, routeID)
		}

		if err != nil {
			result.FailedRows++
			result.Errors = append(result.Errors, ImportError{
				Row:      i + 2,
				Column:   "student_id",
				Value:    studentID,
				Error:    err.Error(),
				Severity: "error",
			})
		} else {
			result.SuccessfulRows++
		}
		result.ProcessedRows++
	}

	if err := tx.Commit(); err != nil {
		result.Summary = "Import failed: " + err.Error()
	} else {
		result.Summary = fmt.Sprintf("Import completed: %d successful, %d failed",
			result.SuccessfulRows, result.FailedRows)
		// Invalidate cache
		dataCache.invalidateStudents()
	}

	return result
}

func importVehicleData(rows [][]string, mappings map[string]string, skipDuplicates bool, updateExisting bool) ImportResult {
	// Similar structure to student import
	return ImportResult{
		TotalRows: len(rows) - 1,
		Summary:   "Vehicle import not yet implemented",
	}
}

func importECSEData(rows [][]string, mappings map[string]string, skipDuplicates bool, updateExisting bool) ImportResult {
	// Similar structure to student import
	return ImportResult{
		TotalRows: len(rows) - 1,
		Summary:   "ECSE import not yet implemented",
	}
}

func importMileageData(rows [][]string, mappings map[string]string, skipDuplicates bool, updateExisting bool) ImportResult {
	// Similar structure to student import
	return ImportResult{
		TotalRows: len(rows) - 1,
		Summary:   "Mileage import not yet implemented",
	}
}

// getValueByMapping retrieves value from row based on column mapping
func getValueByMapping(row []string, headers []string, mappings map[string]string, field string) string {
	mappedColumn, exists := mappings[field]
	if !exists {
		return ""
	}

	for i, header := range headers {
		if header == mappedColumn && i < len(row) {
			return strings.TrimSpace(row[i])
		}
	}
	return ""
}
