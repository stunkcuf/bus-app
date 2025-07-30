package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
)

// exportECSEHandler handles ECSE data export
func exportECSEHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get export format
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "csv"
	}

	// Load ECSE students
	students, err := loadECSEStudentsFromDB()
	if err != nil {
		log.Printf("Error loading ECSE students for export: %v", err)
		http.Error(w, "Failed to load students", http.StatusInternalServerError)
		return
	}

	switch format {
	case "csv":
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", "attachment; filename=ecse_students.csv")
		
		// Write CSV header
		fmt.Fprintln(w, "StudentID,FirstName,LastName,Grade,DateOfBirth,IEPStatus,PrimaryDisability,ServiceMinutes")
		
		// Write student data
		for _, s := range students {
			fmt.Fprintf(w, "%s,%s,%s,%s,%s,%s,%s,%d\n",
				s.StudentID, s.FirstName, s.LastName,
				s.GetGrade(), s.GetDateOfBirth(), s.GetIEPStatus(),
				s.GetPrimaryDisability(), s.GetServiceMinutes())
		}
		
	case "json":
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", "attachment; filename=ecse_students.json")
		json.NewEncoder(w).Encode(students)
		
	default:
		http.Error(w, "Unsupported format", http.StatusBadRequest)
	}
}

// addBusHandler handles bus addition
func addBusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	// Parse form data
	busID := r.FormValue("bus_id")
	model := r.FormValue("model")
	capacityStr := r.FormValue("capacity")
	
	capacity := 0
	if capacityStr != "" {
		capacity, _ = strconv.Atoi(capacityStr)
	}

	// Insert new bus
	_, err := db.Exec(`
		INSERT INTO buses (bus_id, status, model, capacity, created_at)
		VALUES ($1, 'active', $2, $3, CURRENT_TIMESTAMP)
	`, busID, model, capacity)

	if err != nil {
		log.Printf("Error adding bus: %v", err)
		http.Error(w, "Failed to add bus", http.StatusInternalServerError)
		return
	}

	// Clear cache
	dataCache.clearBuses()

	http.Redirect(w, r, "/fleet", http.StatusSeeOther)
}

// addBusWizardHandler shows the add bus wizard
func addBusWizardHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	data := map[string]interface{}{
		"User":      user,
		"Title":     "Add New Bus",
		"CSRFToken": getSessionCSRFToken(r),
	}

	renderTemplate(w, r, "add_bus_wizard.html", data)
}

// maintenanceWizardHandler shows the maintenance wizard
func maintenanceWizardHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	// Get bus list for dropdown
	buses, err := dataCache.getBuses()
	if err != nil {
		log.Printf("Error loading buses: %v", err)
		buses = []Bus{}
	}

	data := map[string]interface{}{
		"User":      user,
		"Title":     "Maintenance Wizard",
		"Buses":     buses,
		"CSRFToken": getSessionCSRFToken(r),
	}

	renderTemplate(w, r, "maintenance_wizard.html", data)
}

// saveMaintenanceWizardHandler saves maintenance from wizard
func saveMaintenanceWizardHandler(w http.ResponseWriter, r *http.Request) {
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
	busID := r.FormValue("bus_id")
	serviceDate := r.FormValue("service_date")
	mileageStr := r.FormValue("mileage")
	cost := r.FormValue("cost")
	description := r.FormValue("description")
	
	mileage := 0
	if mileageStr != "" {
		mileage, _ = strconv.Atoi(mileageStr)
	}

	// Extract bus number from bus_id (format: "Bus #X")
	var busNumber int
	fmt.Sscanf(busID, "Bus #%d", &busNumber)

	// Insert maintenance record
	_, err := db.Exec(`
		INSERT INTO maintenance_records 
		(vehicle_number, service_date, mileage, cost, work_description, created_at)
		VALUES ($1, $2, $3, $4, $5, CURRENT_TIMESTAMP)
	`, busNumber, serviceDate, mileage, cost, description)

	if err != nil {
		log.Printf("Error saving maintenance record: %v", err)
		http.Error(w, "Failed to save maintenance record", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/maintenance-records", http.StatusSeeOther)
}

// routeAssignmentWizardHandler shows the route assignment wizard
func routeAssignmentWizardHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	// Load data for wizard
	drivers, _ := dataCache.getUsers()
	buses, _ := dataCache.getBuses()
	routes, _ := dataCache.getRoutes()
	assignments, _ := loadRouteAssignments()

	// Create maps for quick lookup
	driverAssignments := make(map[string][]RouteAssignment)
	busAssignments := make(map[string][]RouteAssignment)
	for _, a := range assignments {
		driverAssignments[a.Driver] = append(driverAssignments[a.Driver], a)
		busAssignments[a.BusID] = append(busAssignments[a.BusID], a)
	}

	// Filter active drivers
	var activeDrivers []User
	for _, d := range drivers {
		if d.Role == "driver" && d.Status == "active" {
			activeDrivers = append(activeDrivers, d)
		}
	}

	// Filter active buses
	var activeBuses []Bus
	for _, b := range buses {
		if b.Status == "active" {
			activeBuses = append(activeBuses, b)
		}
	}

	data := map[string]interface{}{
		"User":      user,
		"Title":     "Route Assignment Wizard",
		"CSRFToken": getSessionCSRFToken(r),
		"Data": map[string]interface{}{
			"Drivers":   activeDrivers,
			"Buses":     activeBuses,
			"Routes":    routes,
		},
	}

	renderTemplate(w, r, "route_assignment_wizard.html", data)
}

// assignRouteWizardHandler handles route assignment from wizard
func assignRouteWizardHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse form data
	driver := r.FormValue("driver")
	busID := r.FormValue("bus_id")
	routeID := r.FormValue("route_id")
	routeName := r.FormValue("route_name")

	// Validate inputs
	if driver == "" || busID == "" || routeID == "" {
		http.Error(w, "All fields are required", http.StatusBadRequest)
		return
	}

	// Check for existing assignment
	var exists bool
	err := db.Get(&exists, `
		SELECT EXISTS(
			SELECT 1 FROM route_assignments 
			WHERE driver = $1 OR bus_id = $2 OR route_id = $3
		)
	`, driver, busID, routeID)

	if err != nil {
		log.Printf("Error checking existing assignment: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	if exists {
		http.Error(w, "Driver, bus, or route already assigned", http.StatusConflict)
		return
	}

	// Create assignment
	_, err = db.Exec(`
		INSERT INTO route_assignments (driver, bus_id, route_id, route_name, assigned_date, created_at)
		VALUES ($1, $2, $3, $4, CURRENT_DATE, CURRENT_TIMESTAMP)
	`, driver, busID, routeID, routeName)

	if err != nil {
		log.Printf("Error creating assignment: %v", err)
		http.Error(w, "Failed to create assignment", http.StatusInternalServerError)
		return
	}

	// Return success
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Route assigned successfully",
	})
}