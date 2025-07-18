package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
)

// dashboardHandler redirects to the appropriate dashboard based on user role
func dashboardHandler(w http.ResponseWriter, r *http.Request) {
	// Get username from session
	cookie, err := r.Cookie(SessionCookieName)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	
	session, err := GetSecureSession(cookie.Value)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
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
	var pendingUsers []User
	
	// Get pending users
	query := `SELECT username, role, status, registration_date FROM users WHERE status = 'pending' ORDER BY registration_date DESC`
	if err := db.Select(&pendingUsers, query); err != nil {
		http.Error(w, "Failed to get pending users", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"PendingUsers": pendingUsers,
		"CSRFToken":    getSessionCSRFToken(r),
	}
	
	renderTemplate(w, r, "approve_users.html", data)
}

// approveUserHandler approves a pending user
func approveUserHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
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
	var users []User
	
	// Get all users
	query := `SELECT username, role, status, registration_date FROM users ORDER BY username`
	if err := db.Select(&users, query); err != nil {
		http.Error(w, "Failed to get users", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Users":     users,
		"CSRFToken": getSessionCSRFToken(r),
	}
	
	renderTemplate(w, r, "users.html", data)
}

// editUserHandler handles user editing
func editUserHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
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
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
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
	var students []ECSEStudentView
	
	// Get ECSE students
	query := `SELECT * FROM ecse_students ORDER BY last_name, first_name`
	if err := db.Select(&students, query); err != nil {
		http.Error(w, "Failed to get ECSE students", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Students":  students,
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
			http.Error(w, "Failed to parse form", http.StatusBadRequest)
			return
		}

		// Validate CSRF token
		if !validateCSRF(r) {
			http.Error(w, "Invalid CSRF token", http.StatusForbidden)
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
		address := r.FormValue("address")
		city := r.FormValue("city")
		state := r.FormValue("state")
		zipCode := r.FormValue("zip_code")

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
				address = $15, city = $16, state = $17, zip_code = $18
			WHERE student_id = $1
		`

		_, err := db.Exec(query, studentID, firstName, lastName, dateOfBirth, grade,
			enrollmentStatus, iepStatus, primaryDisability, serviceMinutes,
			transportationRequired, busRoute, parentName, parentPhone, parentEmail,
			address, city, state, zipCode)

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
	var students []Student
	var routes []Route
	
	// Get students
	query := `SELECT * FROM students ORDER BY route_id, position_number, name`
	if err := db.Select(&students, query); err != nil {
		http.Error(w, "Failed to get students", http.StatusInternalServerError)
		return
	}

	// Get routes
	if err := db.Select(&routes, "SELECT * FROM routes ORDER BY route_name"); err != nil {
		http.Error(w, "Failed to get routes", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Students":  students,
		"Routes":    routes,
		"CSRFToken": getSessionCSRFToken(r),
	}

	renderTemplate(w, r, "students.html", data)
}

// assignRoutesHandler handles route assignment page
func assignRoutesHandler(w http.ResponseWriter, r *http.Request) {
	var drivers []User
	var buses []Bus
	var routes []Route
	var assignments []RouteAssignment
	
	// Get active drivers
	db.Select(&drivers, "SELECT * FROM users WHERE role = 'driver' AND status = 'active' ORDER BY username")
	
	// Get active buses
	db.Select(&buses, "SELECT * FROM buses WHERE status = 'active' ORDER BY bus_id")
	
	// Get routes
	db.Select(&routes, "SELECT * FROM routes ORDER BY route_name")
	
	// Get current assignments
	query := `SELECT ra.*, u.username as driver_name, b.bus_id as bus_name, r.route_name 
			  FROM route_assignments ra
			  JOIN users u ON ra.driver = u.username
			  JOIN buses b ON ra.bus_id = b.bus_id
			  JOIN routes r ON ra.route_id = r.route_id
			  ORDER BY ra.assigned_date DESC`
	db.Select(&assignments, query)

	data := map[string]interface{}{
		"Drivers":     drivers,
		"Buses":       buses,
		"Routes":      routes,
		"Assignments": assignments,
		"CSRFToken":   getSessionCSRFToken(r),
	}

	renderTemplate(w, r, "assign_routes.html", data)
}

// importMileageHandler handles mileage import page
func importMileageHandler(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"CSRFToken": getSessionCSRFToken(r),
	}

	renderTemplate(w, r, "import_mileage.html", data)
}

// viewMileageReportsHandler shows mileage reports
func viewMileageReportsHandler(w http.ResponseWriter, r *http.Request) {
	var reports []MileageReport
	
	// Get mileage reports
	query := `SELECT * FROM mileage_reports ORDER BY year DESC, month DESC, vehicle_id`
	if err := db.Select(&reports, query); err != nil {
		http.Error(w, "Failed to get mileage reports", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Reports":   reports,
		"CSRFToken": getSessionCSRFToken(r),
	}

	renderTemplate(w, r, "view_mileage_reports.html", data)
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
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	
	session, err := GetSecureSession(cookie.Value)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
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
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
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
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
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
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
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
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
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
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
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

// addStudentHandler adds a new student
func addStudentHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
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
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
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
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
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

// addBusHandler handles adding a new bus to the fleet
func addBusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
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
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	// Validate CSRF token
	if !validateCSRF(r) {
		http.Error(w, "Invalid CSRF token", http.StatusForbidden)
		return
	}

	// Get form values
	busID := r.FormValue("bus_id")
	model := r.FormValue("model")
	capacityStr := r.FormValue("capacity")
	status := r.FormValue("status")

	// Validate required fields
	if busID == "" || model == "" || capacityStr == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
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

	// Create bus in database
	query := `
		INSERT INTO buses (bus_id, model, capacity, status, oil_status, tire_status)
		VALUES ($1, $2, $3, $4, 'good', 'good')
	`
	
	_, err = db.Exec(query, busID, model, capacity, status)
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