package main

import (
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

// approveUsersHandler shows pending user approvals
func approveUsersHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	// Get pending users
	var pendingUsers []User
	err := db.Select(&pendingUsers, `
		SELECT id, username, password, role, status, registration_date, created_at 
		FROM users 
		WHERE status = 'pending' 
		ORDER BY created_at DESC
	`)
	if err != nil {
		log.Printf("Error loading pending users: %v", err)
		http.Error(w, "Failed to load pending users", http.StatusInternalServerError)
		return
	}

	// Use regular template rendering
	data := map[string]interface{}{
		"Title":     "Approve Users",
		"User":      user,
		"CSRFToken": getSessionCSRFToken(r),
		"Data": map[string]interface{}{
			"PendingUsers": pendingUsers,
			"CSRFToken":    getSessionCSRFToken(r),
		},
	}

	renderTemplate(w, r, "approve_users.html", data)
}

// viewECSEStudentHandler displays ECSE student details
func viewECSEStudentHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	studentID := r.URL.Query().Get("id")
	if studentID == "" {
		http.Error(w, "Student ID required", http.StatusBadRequest)
		return
	}

	// Load student data
	var student ECSEStudent
	err := db.Get(&student, "SELECT * FROM ecse_students WHERE student_id = $1", studentID)
	if err != nil {
		log.Printf("Error loading ECSE student: %v", err)
		http.Error(w, "Student not found", http.StatusNotFound)
		return
	}

	// Use TemplateData structure
	templateData := TemplateData{
		Title: fmt.Sprintf("%s %s - ECSE Details", student.FirstName, student.LastName),
		User:  user,
		Data: map[string]interface{}{
			"Student":   student,
			"CSRFToken": getSessionCSRFToken(r),
		},
		CSRFToken: getSessionCSRFToken(r),
		CSPNonce:  getNonce(r),
	}

	renderTemplateData(w, r, "view_ecse_student.html", templateData)
}

// approveUserHandler handles user approval
func approveUserHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("APPROVE USER: Method=%s", r.Method)
	
	if r.Method != http.MethodPost {
		log.Printf("APPROVE USER: Invalid method: %s", r.Method)
		SendError(w, ErrMethodNotAllowed("Only POST method allowed"))
		return
	}

	// Parse form to get values
	if err := r.ParseForm(); err != nil {
		log.Printf("APPROVE USER: Failed to parse form: %v", err)
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	// Validate CSRF token
	if !validateCSRF(r) {
		log.Printf("APPROVE USER: Invalid CSRF token")
		http.Error(w, "Invalid CSRF token", http.StatusForbidden)
		return
	}

	username := r.FormValue("username")
	action := r.FormValue("action")
	
	log.Printf("APPROVE USER: username=%s, action=%s", username, action)
	
	if username == "" || action == "" {
		log.Printf("APPROVE USER: Missing username or action")
		http.Error(w, "Username and action required", http.StatusBadRequest)
		return
	}

	var query string
	if action == "approve" {
		query = "UPDATE users SET status = 'active' WHERE username = $1 AND status = 'pending'"
	} else if action == "reject" {
		query = "DELETE FROM users WHERE username = $1 AND status = 'pending'"
	} else {
		log.Printf("APPROVE USER: Invalid action: %s", action)
		http.Error(w, "Invalid action", http.StatusBadRequest)
		return
	}

	// Execute the action
	result, err := db.Exec(query, username)
	if err != nil {
		log.Printf("APPROVE USER: Database error %s user %s: %v", action, username, err)
		http.Error(w, fmt.Sprintf("Failed to %s user", action), http.StatusInternalServerError)
		return
	}

	// Check if any rows were affected
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("APPROVE USER: Error checking rows affected: %v", err)
	} else {
		log.Printf("APPROVE USER: %d rows affected for %s action on user %s", rowsAffected, action, username)
	}

	// Clear user cache to force reload
	if dataCache != nil {
		dataCache.clear()
		log.Printf("APPROVE USER: Cache cleared after %s action", action)
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
	for i, u := range users {
		log.Printf("User %d: %s, Role: %s, Status: %s", i, u.Username, u.Role, u.Status)
	}

	data := map[string]interface{}{
		"User":      user,
		"Users":     users,
		"CSRFToken": getSessionCSRFToken(r),
	}

	renderTemplate(w, r, "manage_users.html", data)
}

// editUserHandler handles user editing
func editUserHandler(w http.ResponseWriter, r *http.Request) {
	// Handle GET request to show edit form
	if r.Method == http.MethodGet {
		username := r.URL.Query().Get("username")
		if username == "" {
			http.Error(w, "Username required", http.StatusBadRequest)
			return
		}
		
		// Load user data
		var user struct {
			Username string
			Role     string
			Status   string
			Email    string
			Phone    string
		}
		
		err := db.QueryRow(`
			SELECT username, role, status, COALESCE(email, ''), COALESCE(phone, '')
			FROM users WHERE username = $1
		`, username).Scan(&user.Username, &user.Role, &user.Status, &user.Email, &user.Phone)
		
		if err != nil {
			log.Printf("Error loading user: %v", err)
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		
		data := map[string]interface{}{
			"Title":     "Edit User",
			"User":      getUserFromSession(r),
			"CSRFToken": generateCSRFToken(),
			"Data":      user,
		}
		
		renderTemplate(w, r, "edit_user.html", data)
		return
	}
	
	// Handle POST request
	if r.Method != http.MethodPost {
		SendError(w, ErrMethodNotAllowed("Only GET or POST method allowed"))
		return
	}

	username := r.FormValue("username")
	action := r.FormValue("action")

	if username == "" {
		http.Error(w, "Username required", http.StatusBadRequest)
		return
	}

	switch action {
	case "update_role":
		role := r.FormValue("role")
		if role != "driver" && role != "manager" {
			http.Error(w, "Invalid role", http.StatusBadRequest)
			return
		}
		
		_, err := db.Exec("UPDATE users SET role = $1 WHERE username = $2", role, username)
		if err != nil {
			log.Printf("Error updating user role: %v", err)
			http.Error(w, "Failed to update role", http.StatusInternalServerError)
			return
		}
		
	case "reset_password":
		password := r.FormValue("password")
		if password == "" || len(password) < 6 {
			http.Error(w, "Password must be at least 6 characters", http.StatusBadRequest)
			return
		}
		
		hashedPassword, err := hashPassword(password)
		if err != nil {
			log.Printf("Error hashing password: %v", err)
			http.Error(w, "Failed to reset password", http.StatusInternalServerError)
			return
		}
		
		_, err = db.Exec("UPDATE users SET password = $1 WHERE username = $2", hashedPassword, username)
		if err != nil {
			log.Printf("Error resetting password: %v", err)
			http.Error(w, "Failed to reset password", http.StatusInternalServerError)
			return
		}
		
	default:
		http.Error(w, "Invalid action", http.StatusBadRequest)
		return
	}

	// Redirect back to manager dashboard
	http.Redirect(w, r, "/manager-dashboard", http.StatusSeeOther)
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
	log.Printf("DEBUG: Total users loaded: %d", len(allUsers))

	// Filter for active drivers
	var drivers []User
	for _, u := range allUsers {
		if u.Role == "driver" && u.Status == "active" {
			drivers = append(drivers, u)
		}
	}
	log.Printf("DEBUG: Active drivers found: %d", len(drivers))

	buses, err := dataCache.getBuses()
	if err != nil {
		log.Printf("Error loading buses: %v", err)
	}
	log.Printf("DEBUG: Total buses loaded: %d", len(buses))

	// Filter for active buses
	var activeBuses []Bus
	for _, b := range buses {
		if b.Status == "active" {
			activeBuses = append(activeBuses, b)
		}
	}
	log.Printf("DEBUG: Active buses found: %d", len(activeBuses))

	routes, err := dataCache.getRoutes()
	if err != nil {
		log.Printf("Error loading routes: %v", err)
	}
	log.Printf("DEBUG: Total routes loaded: %d", len(routes))

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

	// Calculate statistics
	totalAssignments := len(assignments)
	totalRoutes := len(routes)
	
	// Count available drivers (not assigned)
	assignedDrivers := make(map[string]bool)
	for _, a := range assignments {
		assignedDrivers[a.Driver] = true
	}
	availableDriversCount := 0
	for _, d := range drivers {
		if !assignedDrivers[d.Username] {
			availableDriversCount++
		}
	}
	
	// Count available buses (not assigned)
	assignedBuses := make(map[string]bool)
	for _, a := range assignments {
		assignedBuses[a.BusID] = true
	}
	availableBusesCount := 0
	for _, b := range activeBuses {
		if !assignedBuses[b.BusID] {
			availableBusesCount++
		}
	}
	
	// Build routes with status
	assignedRoutes := make(map[string]bool)
	for _, a := range assignments {
		assignedRoutes[a.RouteID] = true
	}
	
	type RouteWithStatus struct {
		Route
		IsAssigned bool
	}
	
	var routesWithStatus []RouteWithStatus
	var availableRoutes []Route
	for _, r := range routes {
		routesWithStatus = append(routesWithStatus, RouteWithStatus{
			Route:      r,
			IsAssigned: assignedRoutes[r.RouteID],
		})
		// Add to available routes if not assigned
		if !assignedRoutes[r.RouteID] {
			availableRoutes = append(availableRoutes, r)
		}
	}
	
	// Build available buses list (those not assigned)
	var availableBuses []Bus
	for _, b := range activeBuses {
		if !assignedBuses[b.BusID] {
			availableBuses = append(availableBuses, b)
		}
	}

	// Debug logging to verify data
	log.Printf("DEBUG: RoutesWithStatus count: %d", len(routesWithStatus))
	if len(routesWithStatus) > 0 {
		log.Printf("DEBUG: First route: %+v", routesWithStatus[0])
	}
	log.Printf("DEBUG: Assignments count: %d", len(assignments))
	
	data := map[string]interface{}{
		"User": user,
		"Data": map[string]interface{}{
			"Drivers":               drivers,
			"Buses":                activeBuses,
			"Routes":               routes,
			"RoutesWithStatus":     routesWithStatus,
			"AvailableRoutes":      availableRoutes,
			"AvailableBuses":       availableBuses,
			"Assignments":          assignments,
			"StudentCounts":        studentCounts,
			"TotalAssignments":     totalAssignments,
			"TotalRoutes":          totalRoutes,
			"AvailableDriversCount": availableDriversCount,
			"AvailableBusesCount":  availableBusesCount,
			"CSRFToken":           getSessionCSRFToken(r),
		},
		"CSRFToken": getSessionCSRFToken(r),
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
			"CSRFToken": getSessionCSRFToken(r),
		}
		renderTemplate(w, r, "import_mileage_simple.html", data)
		return
	}

	// Handle POST (file upload)
	err := r.ParseMultipartForm(10 << 20) // 10 MB max
	if err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	file, handler, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Failed to get file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	log.Printf("Uploading file: %s", handler.Filename)

	// Process the file
	if strings.HasSuffix(handler.Filename, ".xlsx") || strings.HasSuffix(handler.Filename, ".xls") {
		processExcelFile(w, r, file, handler.Filename)
	} else if strings.HasSuffix(handler.Filename, ".csv") {
		// CSV import has been removed - redirect to unified import
		http.Error(w, "CSV import is no longer supported. Please use the unified import system at /import", http.StatusBadRequest)
		return
	} else {
		http.Error(w, "Unsupported file type. Please upload Excel or CSV files.", http.StatusBadRequest)
		return
	}
}

// processExcelFile handles Excel file processing for mileage import
func processExcelFile(w http.ResponseWriter, r *http.Request, file multipart.File, filename string) {
	// Create temp file
	tempFile, err := os.CreateTemp("", "mileage-*.xlsx")
	if err != nil {
		http.Error(w, "Failed to create temp file", http.StatusInternalServerError)
		return
	}
	defer os.Remove(tempFile.Name())

	// Copy uploaded file to temp file
	_, err = io.Copy(tempFile, file)
	if err != nil {
		http.Error(w, "Failed to save file", http.StatusInternalServerError)
		return
	}
	tempFile.Close()

	// Open Excel file
	f, err := excelize.OpenFile(tempFile.Name())
	if err != nil {
		http.Error(w, "Failed to open Excel file", http.StatusInternalServerError)
		return
	}
	defer f.Close()

	// Get first sheet
	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		http.Error(w, "No sheets found in Excel file", http.StatusBadRequest)
		return
	}

	rows, err := f.GetRows(sheets[0])
	if err != nil {
		http.Error(w, "Failed to read Excel data", http.StatusInternalServerError)
		return
	}

	// Process rows
	var imported, failed int
	for i, row := range rows {
		if i == 0 {
			// Skip header row
			continue
		}

		if len(row) < 4 {
			failed++
			continue
		}

		// Parse row data
		date := row[0]
		busNumber := row[1]
		startMileage := row[2]
		endMileage := row[3]

		// Convert to appropriate types
		busNum, _ := strconv.Atoi(busNumber)
		startMile, _ := strconv.Atoi(startMileage)
		endMile, _ := strconv.Atoi(endMileage)

		// Insert into database
		_, err := db.Exec(`
			INSERT INTO mileage_records (date, bus_number, start_mileage, end_mileage, created_at)
			VALUES ($1, $2, $3, $4, $5)
		`, date, busNum, startMile, endMile, time.Now())

		if err != nil {
			log.Printf("Failed to import row %d: %v", i+1, err)
			failed++
		} else {
			imported++
		}
	}

	// Return result
	data := map[string]interface{}{
		"User": getUserFromSession(r),
		"Data": map[string]interface{}{
			"Success":  true,
			"Imported": imported,
			"Failed":   failed,
			"Total":    len(rows) - 1,
			"CSRFToken": getSessionCSRFToken(r),
		},
		"CSRFToken": getSessionCSRFToken(r),
	}

	renderTemplate(w, r, "import_mileage_simple.html", data)
}

// processCSVFile has been removed - use the unified import system at /import

// usersHandler handles user profile edit page
func usersHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	username := r.URL.Query().Get("username")
	if username == "" {
		http.Error(w, "Username required", http.StatusBadRequest)
		return
	}

	// Load user data
	var targetUser User
	err := db.Get(&targetUser, `
		SELECT username, password, role, status, registration_date, created_at 
		FROM users WHERE username = $1
	`, username)
	if err != nil {
		log.Printf("Error loading user: %v", err)
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	data := map[string]interface{}{
		"User":      user,
		"Username":  targetUser.Username,
		"Role":      targetUser.Role,
		"CSRFToken": getSessionCSRFToken(r),
	}

	renderTemplate(w, r, "users.html", data)
}

// deleteUserHandler handles user deletion
func deleteUserHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("DELETE USER: Method=%s", r.Method)
	
	if r.Method != http.MethodPost {
		log.Printf("DELETE USER: Invalid method: %s", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		log.Printf("DELETE USER: Unauthorized - user=%v", user)
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	// Parse form to get values
	if err := r.ParseForm(); err != nil {
		log.Printf("DELETE USER: Failed to parse form: %v", err)
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	// Validate CSRF token
	if !validateCSRF(r) {
		log.Printf("DELETE USER: Invalid CSRF token")
		http.Error(w, "Invalid CSRF token", http.StatusForbidden)
		return
	}

	username := r.FormValue("username")
	log.Printf("DELETE USER: Attempting to delete user: %s", username)
	
	if username == "" {
		log.Printf("DELETE USER: Empty username")
		http.Error(w, "Username required", http.StatusBadRequest)
		return
	}

	// Don't allow self-deletion
	if username == user.Username {
		log.Printf("DELETE USER: Attempted self-deletion by %s", username)
		http.Error(w, "Cannot delete your own account", http.StatusBadRequest)
		return
	}

	// Check if user exists
	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE username = $1)", username).Scan(&exists)
	if err != nil {
		log.Printf("DELETE USER: Error checking user existence: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	
	if !exists {
		log.Printf("DELETE USER: User %s not found", username)
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Delete the user
	result, err := db.Exec("DELETE FROM users WHERE username = $1", username)
	if err != nil {
		log.Printf("DELETE USER: Error deleting user %s: %v", username, err)
		http.Error(w, "Failed to delete user", http.StatusInternalServerError)
		return
	}
	
	rowsAffected, _ := result.RowsAffected()
	log.Printf("DELETE USER: Successfully deleted user %s (rows affected: %d)", username, rowsAffected)

	// Clear user cache to force reload
	if dataCache != nil {
		dataCache.clear()
		log.Printf("DELETE USER: Cache cleared after deletion")
	}

	http.Redirect(w, r, "/manage-users", http.StatusSeeOther)
}