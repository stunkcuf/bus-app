package main

import (
	"log"
	"net/http"
)

// importValidateHandler validates import data - delegates to enhanced version
func importValidateHandler(w http.ResponseWriter, r *http.Request) {
	enhancedImportValidateHandler(w, r)
}

// importExecuteHandler executes the import - delegates to enhanced version
func importExecuteHandler(w http.ResponseWriter, r *http.Request) {
	enhancedImportExecuteHandler(w, r)
}

// driverProfileHandler shows driver profile
func driverProfileHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	// Get driver info
	driverUsername := r.URL.Query().Get("driver")
	if driverUsername == "" {
		driverUsername = user.Username
	}

	// Only allow viewing own profile unless manager
	if driverUsername != user.Username && user.Role != "manager" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	data := map[string]interface{}{
		"User":      user,
		"Title":     "Driver Profile",
		"Driver":    driverUsername,
		"CSRFToken": getSessionCSRFToken(r),
	}

	renderTemplate(w, r, "driver_profile.html", data)
}

// studentsHandler shows students list
func studentsHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	// Load students based on user role
	var students []Student
	var err error
	
	if user.Role == "driver" {
		// Load only students assigned to this driver
		err = db.Select(&students, `
			SELECT * FROM students 
			WHERE driver = $1 AND active = true
			ORDER BY name
		`, user.Username)
	} else {
		// Managers see all students
		students, err = loadStudentsFromDB()
	}

	if err != nil {
		log.Printf("Error loading students: %v", err)
		students = []Student{}
	}

	data := map[string]interface{}{
		"User":      user,
		"Title":     "Students",
		"Students":  students,
		"CSRFToken": getSessionCSRFToken(r),
	}

	renderTemplate(w, r, "students.html", data)
}

// addStudentHandler handles student addition
func addStudentHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user := getUserFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	// Parse form data
	studentID := r.FormValue("student_id")
	name := r.FormValue("name")
	locations := r.FormValue("locations")
	phoneNumber := r.FormValue("phone_number")
	guardian := r.FormValue("guardian")
	routeID := r.FormValue("route_id")
	driver := r.FormValue("driver")

	// Drivers can only add students to their own routes
	if user.Role == "driver" {
		driver = user.Username
	}

	// Insert student
	_, err := db.Exec(`
		INSERT INTO students 
		(student_id, name, locations, phone_number, guardian, route_id, driver, active, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, true, CURRENT_TIMESTAMP)
	`, studentID, name, locations, phoneNumber, guardian, routeID, driver)

	if err != nil {
		log.Printf("Error adding student: %v", err)
		http.Error(w, "Failed to add student", http.StatusInternalServerError)
		return
	}

	// Clear cache
	dataCache.clearStudents()

	http.Redirect(w, r, "/students", http.StatusSeeOther)
}

// addStudentWizardHandler shows the add student wizard
func addStudentWizardHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	// Get routes for dropdown
	routes, err := dataCache.getRoutes()
	if err != nil {
		log.Printf("Error loading routes: %v", err)
		routes = []Route{}
	}

	// Get drivers for dropdown (managers only)
	var drivers []User
	if user.Role == "manager" {
		allUsers, _ := dataCache.getUsers()
		for _, u := range allUsers {
			if u.Role == "driver" && u.Status == "active" {
				drivers = append(drivers, u)
			}
		}
	}

	data := map[string]interface{}{
		"User":      user,
		"Title":     "Add New Student",
		"Routes":    routes,
		"Drivers":   drivers,
		"CSRFToken": getSessionCSRFToken(r),
	}

	renderTemplate(w, r, "add_student_wizard.html", data)
}

// lastMaintenanceHandler gets last maintenance date for a vehicle
func lastMaintenanceHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get vehicle ID from URL
	vehicleID := r.URL.Path[len("/api/last-maintenance/"):]
	
	// Get last maintenance date
	var lastDate string
	err := db.Get(&lastDate, `
		SELECT COALESCE(MAX(service_date), '')
		FROM maintenance_records 
		WHERE vehicle_id = $1
	`, vehicleID)

	if err != nil {
		log.Printf("Error getting last maintenance: %v", err)
		lastDate = ""
	}

	renderJSON(w, map[string]interface{}{
		"lastMaintenance": lastDate,
	})
}

// previewImportHandler previews import data
func previewImportHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// This would preview import data
	// For now return mock data
	renderJSON(w, map[string]interface{}{
		"success": true,
		"preview": []map[string]interface{}{
			{"type": "student", "name": "John Doe", "grade": "5"},
			{"type": "student", "name": "Jane Smith", "grade": "3"},
			{"type": "route", "name": "Route A", "driver": "driver1"},
		},
	})
}

// editStudentHandler handles student editing
func editStudentHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user := getUserFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	// Parse form data
	studentID := r.FormValue("student_id")
	name := r.FormValue("name")
	locations := r.FormValue("locations")
	phoneNumber := r.FormValue("phone_number")
	guardian := r.FormValue("guardian")
	routeID := r.FormValue("route_id")

	// Update student
	_, err := db.Exec(`
		UPDATE students 
		SET name = $2, locations = $3, phone_number = $4, guardian = $5, route_id = $6
		WHERE student_id = $1
	`, studentID, name, locations, phoneNumber, guardian, routeID)

	if err != nil {
		log.Printf("Error updating student: %v", err)
		http.Error(w, "Failed to update student", http.StatusInternalServerError)
		return
	}

	// Clear cache
	dataCache.clearStudents()

	http.Redirect(w, r, "/students", http.StatusSeeOther)
}

// removeStudentHandler handles student removal
func removeStudentHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user := getUserFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	studentID := r.FormValue("student_id")
	
	// Soft delete - just mark as inactive
	_, err := db.Exec("UPDATE students SET active = false WHERE student_id = $1", studentID)
	if err != nil {
		log.Printf("Error removing student: %v", err)
		http.Error(w, "Failed to remove student", http.StatusInternalServerError)
		return
	}

	// Clear cache
	dataCache.clearStudents()

	http.Redirect(w, r, "/students", http.StatusSeeOther)
}