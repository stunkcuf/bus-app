package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
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

	log.Printf("STUDENTS HANDLER: Starting for user=%s, role=%s", user.Username, user.Role)

	// Load students based on user role
	var students []Student
	var err error
	
	if user.Role == "driver" {
		// Load only students assigned to this driver
		log.Printf("STUDENTS HANDLER: Loading students for driver %s", user.Username)
		
		// Debug: Test a simple count first
		var count int
		countErr := db.Get(&count, "SELECT COUNT(*) FROM students WHERE driver = $1 AND active = true", user.Username)
		log.Printf("STUDENTS HANDLER: Count query shows %d students for driver %s (error=%v)", count, user.Username, countErr)
		
		err = db.Select(&students, `
			SELECT 
				COALESCE(student_id, '') as student_id,
				COALESCE(name, '') as name,
				COALESCE(locations::text, '[]') as locations,
				COALESCE(phone_number, '') as phone_number,
				COALESCE(alt_phone_number, '') as alt_phone_number,
				COALESCE(guardian, '') as guardian,
				COALESCE(pickup_time::text, '') as pickup_time,
				COALESCE(dropoff_time::text, '') as dropoff_time,
				COALESCE(position_number, 0) as position_number,
				COALESCE(route_id, '') as route_id,
				COALESCE(driver, '') as driver,
				COALESCE(active, false) as active,
				COALESCE(created_at, CURRENT_TIMESTAMP) as created_at
			FROM students 
			WHERE driver = $1 AND active = true
			ORDER BY name
		`, user.Username)
		log.Printf("STUDENTS HANDLER: Query result for driver %s: %d students, error=%v", user.Username, len(students), err)
	} else {
		// Managers see all students
		log.Printf("STUDENTS HANDLER: Loading all students for manager")
		students, err = loadStudentsFromDB()
		log.Printf("STUDENTS HANDLER: Query result for manager: %d students, error=%v", len(students), err)
	}

	if err != nil {
		log.Printf("STUDENTS HANDLER ERROR: Loading students for user %s (role=%s): %v", user.Username, user.Role, err)
		// Ensure we have an empty slice, not nil
		students = make([]Student, 0)
	}
	
	// Ensure students is never nil
	if students == nil {
		log.Printf("STUDENTS HANDLER: students was nil, creating empty slice")
		students = make([]Student, 0)
	}
	
	log.Printf("STUDENTS HANDLER: Final count %d students for user %s (role=%s)", len(students), user.Username, user.Role)
	
	// Log first student if any to verify data
	if len(students) > 0 {
		log.Printf("STUDENTS HANDLER: First student: ID=%s, Name=%s", students[0].StudentID, students[0].Name)
	}

	// Get routes for dropdown
	var routes []Route
	if user.Role == "driver" {
		// For drivers, get only their assigned routes
		assignments, err := getDriverAssignments(user.Username)
		if err == nil {
			allRoutes, _ := dataCache.getRoutes()
			for _, assignment := range assignments {
				for _, route := range allRoutes {
					if route.RouteID == assignment.RouteID {
						routes = append(routes, route)
						break
					}
				}
			}
		}
		log.Printf("STUDENTS HANDLER: Driver %s has %d routes available", user.Username, len(routes))
	} else {
		// Managers see all routes
		routes, _ = dataCache.getRoutes()
		log.Printf("STUDENTS HANDLER: Manager has %d routes available", len(routes))
	}

	// If no routes available, add an empty option
	if len(routes) == 0 {
		routes = append(routes, Route{
			RouteID:   "",
			RouteName: "No Routes Available",
		})
	}

	data := map[string]interface{}{
		"User":      user,
		"Title":     "Students",
		"Data": map[string]interface{}{
			"Students": students,
			"Routes":   routes,
		},
		"CSRFToken": getSessionCSRFToken(r),
	}
	
	log.Printf("STUDENTS HANDLER: Rendering template with data structure: User=%s, Data.Students count=%d", user.Username, len(students))

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

	// Validate CSRF token
	if !validateCSRF(r) {
		http.Error(w, "Invalid CSRF token", http.StatusForbidden)
		return
	}

	// Parse form data
	studentID := r.FormValue("student_id")
	name := r.FormValue("name")
	phoneNumber := r.FormValue("phone_number")
	guardian := r.FormValue("guardian")
	routeID := r.FormValue("route_id")
	driver := r.FormValue("driver")
	
	// Handle locations - combine pickup and dropoff addresses
	pickupAddress := r.FormValue("pickup_address")
	dropoffAddress := r.FormValue("dropoff_address")
	locations := "[]" // Default empty JSON array
	if pickupAddress != "" || dropoffAddress != "" {
		// Create a JSON array with the addresses
		locations = fmt.Sprintf(`[{"type":"pickup","address":"%s"},{"type":"dropoff","address":"%s"}]`, 
			pickupAddress, dropoffAddress)
	}

	// Drivers automatically get assigned as the driver for students they add
	if user.Role == "driver" {
		driver = user.Username
		log.Printf("Driver %s adding student, setting driver field to %s", user.Username, driver)
	} else if driver == "" {
		// For managers, if no driver specified, leave it empty
		log.Printf("Manager %s adding student with driver field: %s", user.Username, driver)
	}

	// Validate required fields
	if studentID == "" || name == "" {
		log.Printf("Missing required fields: studentID=%s, name=%s", studentID, name)
		http.Error(w, "Student ID and Name are required", http.StatusBadRequest)
		return
	}

	// Insert student
	_, err := db.Exec(`
		INSERT INTO students 
		(student_id, name, locations, phone_number, guardian, route_id, driver, active, created_at)
		VALUES ($1, $2, $3::jsonb, $4, $5, $6, $7, true, CURRENT_TIMESTAMP)
	`, studentID, name, locations, phoneNumber, guardian, routeID, driver)

	if err != nil {
		log.Printf("Error adding student: %v (studentID=%s, name=%s, phone=%s, guardian=%s, route=%s, driver=%s)", 
			err, studentID, name, phoneNumber, guardian, routeID, driver)
		
		// Check for duplicate student ID
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique constraint") {
			http.Error(w, "Student ID already exists", http.StatusConflict)
			return
		}
		
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
	var routes []Route
	if user.Role == "driver" {
		// For drivers, get only their assigned routes
		assignments, err := getDriverAssignments(user.Username)
		if err == nil {
			allRoutes, _ := dataCache.getRoutes()
			for _, assignment := range assignments {
				for _, route := range allRoutes {
					if route.RouteID == assignment.RouteID {
						routes = append(routes, route)
						break
					}
				}
			}
		}
	} else {
		// Managers see all routes
		var err error
		routes, err = dataCache.getRoutes()
		if err != nil {
			log.Printf("Error loading routes: %v", err)
			routes = []Route{}
		}
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

	// Get file from form data
	err := r.ParseMultipartForm(10 << 20) // 10 MB
	if err != nil {
		renderJSON(w, map[string]interface{}{
			"success": false,
			"error":   "Failed to parse upload",
		})
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		renderJSON(w, map[string]interface{}{
			"success": false,
			"error":   "No file uploaded",
		})
		return
	}
	defer file.Close()

	// For now, return empty preview as parsing needs to be implemented
	renderJSON(w, map[string]interface{}{
		"success": true,
		"preview": []map[string]interface{}{},
		"message": "File uploaded successfully. Preview not yet implemented.",
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