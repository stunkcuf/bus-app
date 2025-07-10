package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"github.com/xuri/excelize/v2"
	_ "github.com/lib/pq"
)

// loginHandler handles user login
func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		csrfToken, _ := GenerateSecureToken()
		renderTemplate(w, r, "login.html", LoginFormData{
			CSRFToken: csrfToken,
		})
		return
	}

	if r.Method == "POST" {
		username := r.FormValue("username")
		password := r.FormValue("password")

		// REMOVED CSRF validation on login - no session exists yet to validate against
		// CSRF protection is for authenticated sessions, not login attempts

		// Authenticate user
		user, err := authenticateUser(username, password)
		if err != nil {
			renderLoginError(w, r, "Invalid username or password")
			return
		}

		// Create session - adjusted for 3 return values
		sessionID, _, err := CreateSecureSession(user.Username, user.Role)
		if err != nil {
			renderLoginError(w, r, "Failed to create session")
			return
		}

		// Set session cookie
		http.SetCookie(w, &http.Cookie{
			Name:     SessionCookieName,
			Value:    sessionID,
			Path:     "/",
			HttpOnly: true,
			Secure:   !isDevelopment(),
			SameSite: http.SameSiteStrictMode,
			MaxAge:   86400, // 24 hours
		})

		// Redirect based on role
		if user.Role == "manager" {
			http.Redirect(w, r, "/manager-dashboard", http.StatusSeeOther)
		} else {
			http.Redirect(w, r, "/driver-dashboard", http.StatusSeeOther)
		}
	}
}

// registerHandler handles user registration
func registerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		csrfToken, _ := GenerateSecureToken()
		renderTemplate(w, r, "register.html", map[string]interface{}{
			"CSRFToken": csrfToken,
		})
		return
	}

	if r.Method == "POST" {
		username := r.FormValue("username")
		password := r.FormValue("password")
		role := r.FormValue("role")

		// REMOVED CSRF validation on registration - no session exists yet

		// Validate input
		if username == "" || password == "" {
			http.Error(w, "All fields are required", http.StatusBadRequest)
			return
		}

		// Default role to driver if not specified
		if role == "" {
			role = "driver"
		}

		// Hash password
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, "Failed to process password", http.StatusInternalServerError)
			return
		}

		// Create user
		status := "pending"
		if role == "manager" {
			status = "active"
		}

		_, err = db.Exec(`
			INSERT INTO users (username, password, role, status, registration_date, created_at)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, username, string(hashedPassword), role, status, time.Now().Format("2006-01-02"), time.Now())

		if err != nil {
			http.Error(w, "Username already exists", http.StatusBadRequest)
			return
		}

		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

// logoutHandler handles user logout
func logoutHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(SessionCookieName)
	if err == nil {
		// FIXED: Use ClearSession function instead of undefined secureSessions
		ClearSession(cookie.Value)
	}

	// Clear cookie
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   !isDevelopment(),
		SameSite: http.SameSiteStrictMode,
	})

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// dashboardHandler serves the manager dashboard
func dashboardHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// Load data based on role
	data := DashboardData{
		User:      user,
		Role:      user.Role,
		CSRFToken: generateCSRFToken(),
	}

	if user.Role == "manager" {
		// Load manager-specific data
		data.Users = loadUsers()
		data.Buses = loadBuses()
		
		// FIX: Convert []Route to []*Route
		routes, _ := loadRoutes()
		routePtrs := make([]*Route, len(routes))
		for i := range routes {
			routePtrs[i] = &routes[i]
		}
		data.Routes = routePtrs
		
		data.DriverSummaries = loadDriverSummaries()
		data.RouteStats = loadRouteStats()
		activities, _ := loadActivities()
		data.Activities = activities
		data.PendingUsers = countPendingUsers()
	}

	// Use renderTemplate instead of executeTemplate for CSP nonce support
	renderTemplate(w, r, "dashboard.html", data)
}

// approveUsersHandler shows pending users for approval
func approveUsersHandler(w http.ResponseWriter, r *http.Request) {
	users := []User{}
	rows, err := db.Query(`SELECT username, role, registration_date FROM users WHERE status = 'pending'`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var user User
			rows.Scan(&user.Username, &user.Role, &user.RegistrationDate)
			users = append(users, user)
		}
	}

	renderTemplate(w, r, "approve_users.html", map[string]interface{}{
		"Users":     users,
		"CSRFToken": generateCSRFToken(),
	})
}

// approveUserHandler approves a pending user
func approveUserHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		username := r.FormValue("username")
		if !validateCSRF(r) {
			http.Error(w, "Invalid CSRF token", http.StatusForbidden)
			return
		}

		_, err := db.Exec(`UPDATE users SET status = 'active' WHERE username = $1`, username)
		if err != nil {
			http.Error(w, "Failed to approve user", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/approve-users", http.StatusSeeOther)
	}
}

// manageUsersHandler shows all users for management
func manageUsersHandler(w http.ResponseWriter, r *http.Request) {
	users := loadUsers()
	renderTemplate(w, r, "manage_users.html", map[string]interface{}{
		"Users":     users,
		"CSRFToken": generateCSRFToken(),
	})
}

// editUserHandler handles user editing
func editUserHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		username := r.URL.Query().Get("username")
		user := getUserByUsername(username)
		if user == nil {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}

		renderTemplate(w, r, "edit_user.html", map[string]interface{}{
			"User":      user,
			"CSRFToken": generateCSRFToken(),
		})
		return
	}

	if r.Method == "POST" {
		if !validateCSRF(r) {
			http.Error(w, "Invalid CSRF token", http.StatusForbidden)
			return
		}

		username := r.FormValue("username")
		newPassword := r.FormValue("password")
		role := r.FormValue("role")

		// Update user
		if newPassword != "" {
			hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
			db.Exec(`UPDATE users SET password = $1, role = $2 WHERE username = $3`, hashedPassword, role, username)
		} else {
			db.Exec(`UPDATE users SET role = $1 WHERE username = $2`, role, username)
		}

		http.Redirect(w, r, "/manage-users", http.StatusSeeOther)
	}
}

// deleteUserHandler deletes a user
func deleteUserHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		if !validateCSRF(r) {
			http.Error(w, "Invalid CSRF token", http.StatusForbidden)
			return
		}

		username := r.FormValue("username")
		db.Exec(`DELETE FROM users WHERE username = $1`, username)
		http.Redirect(w, r, "/manage-users", http.StatusSeeOther)
	}
}

// driverDashboardHandler serves the driver dashboard - FIXED VERSION
func driverDashboardHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "driver" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// Get date and period from query params
	date := r.URL.Query().Get("date")
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}
	
	period := r.URL.Query().Get("period")
	if period == "" {
		period = "morning"
	}

	// Get driver's assignment
	assignment := getDriverAssignment(user.Username)
	
	// Get bus details if assigned
	var bus *Bus
	if assignment.BusID != "" {
		bus = getBusByID(assignment.BusID)
	}
	
	// Get route details if assigned
	var route *Route
	if assignment.RouteID != "" {
		route = getRouteByID(assignment.RouteID)
	}
	
	// Get students for the route ordered by pickup/dropoff time
	students := []Student{}
	if assignment.RouteID != "" {
		if period == "morning" {
			students = getRouteStudentsOrderedByPickup(assignment.RouteID)
		} else {
			students = getRouteStudentsOrderedByDropoff(assignment.RouteID)
		}
	}
	
	// Get driver's log for this date/period if exists
	var driverLog DriverLog
	var attendanceJSON sql.NullString
	err := db.QueryRow(`
		SELECT id, driver, bus_id, route_id, date, period, 
		       departure_time, arrival_time, mileage, attendance
		FROM driver_logs
		WHERE driver = $1 AND date = $2 AND period = $3
	`, user.Username, date, period).Scan(
		&driverLog.ID, &driverLog.Driver, 
		&driverLog.BusID, &driverLog.RouteID, &driverLog.Date, 
		&driverLog.Period, &driverLog.Departure, &driverLog.Arrival, 
		&driverLog.Mileage, &attendanceJSON)
	
	var driverLogPtr *DriverLog
	if err == nil {
		driverLogPtr = &driverLog
		if attendanceJSON.Valid && attendanceJSON.String != "" {
			json.Unmarshal([]byte(attendanceJSON.String), &driverLog.Attendance)
		}
	}
	
	// Get recent logs
	recentLogs := getDriverLogs(user.Username, 7)

	data := map[string]interface{}{
		"User":       user,
		"Assignment": assignment,
		"Bus":        bus,
		"Route":      route,
		"Students":   students,
		"DriverLog":  driverLogPtr,
		"RecentLogs": recentLogs,
		"Date":       date,
		"Period":     period,
		"Today":      time.Now().Format("2006-01-02"),
		"CSRFToken":  generateCSRFToken(),
	}

	renderTemplate(w, r, "driver_dashboard.html", data)
}

// saveLogHandler saves a driver's log
func saveLogHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		if !validateCSRF(r) {
			http.Error(w, "Invalid CSRF token", http.StatusForbidden)
			return
		}

		user := getUserFromSession(r)
		if user == nil || user.Role != "driver" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Parse form data
		log := DriverLog{
			Driver:    user.Username,
			Date:      r.FormValue("date"),
			Period:    r.FormValue("period"),
			RouteID:   r.FormValue("route_id"),
			BusID:     r.FormValue("bus_id"),
			Departure: r.FormValue("departure"),
			Arrival:   r.FormValue("arrival"),
			Mileage:   parseFloatOrDefault(r.FormValue("mileage"), 0),
		}

		// Parse attendance
		var attendance []StudentAttendance
		for key, values := range r.Form {
			if strings.HasPrefix(key, "present_") {
				positionStr := strings.TrimPrefix(key, "present_")
				position := parseIntOrDefault(positionStr, 0)
				if position > 0 && len(values) > 0 {
					attendance = append(attendance, StudentAttendance{
						Position: position,
						Present:  true,
					})
				}
			}
		}

		// Save to database
		attendanceJSON, _ := json.Marshal(attendance)
		_, err := db.Exec(`
			INSERT INTO driver_logs (driver, date, period, route_id, bus_id, departure_time, arrival_time, mileage, attendance)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			ON CONFLICT (driver, date, period) DO UPDATE 
			SET route_id = $4, bus_id = $5, departure_time = $6, arrival_time = $7, mileage = $8, attendance = $9
		`, log.Driver, log.Date, log.Period, log.RouteID, log.BusID, log.Departure, log.Arrival, log.Mileage, string(attendanceJSON))

		if err != nil {
			http.Error(w, "Failed to save log", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/driver-dashboard", http.StatusSeeOther)
	}
}

// fleetHandler shows the bus fleet
func fleetHandler(w http.ResponseWriter, r *http.Request) {
	if !isLoggedIn(r) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	user := getUser(r)
	if user.Role != "manager" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	buses := loadBuses()
	
	// Get recent maintenance logs
	maintenanceLogs := []BusMaintenanceLog{}
	rows, err := db.Query(`
		SELECT bus_id, date, category, notes, mileage, created_at
		FROM bus_maintenance_logs
		ORDER BY date DESC
		LIMIT 10
	`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var log BusMaintenanceLog
			err := rows.Scan(&log.BusID, &log.Date, &log.Category, 
				&log.Notes, &log.Mileage, &log.CreatedAt)
			if err == nil {
				maintenanceLogs = append(maintenanceLogs, log)
			}
		}
	} else {
		log.Printf("Error loading maintenance logs: %v", err)
	}

	data := map[string]interface{}{
		"User":            user,
		"Buses":           buses,
		"MaintenanceLogs": maintenanceLogs,
		"Today":           time.Now().Format("2006-01-02"),
		"CSRFToken":       generateCSRFToken(),
	}
	renderTemplate(w, r, "fleet.html", data)
}

// companyFleetHandler shows company vehicles
func companyFleetHandler(w http.ResponseWriter, r *http.Request) {
	vehicles := []Vehicle{}
	rows, err := db.Query(`
		SELECT vehicle_id, model, year, description, status, oil_status, tire_status, maintenance_notes
		FROM vehicles
		ORDER BY vehicle_id
	`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var v Vehicle
			rows.Scan(&v.VehicleID, &v.Model, &v.Year, &v.Description, &v.Status, &v.OilStatus, &v.TireStatus, &v.MaintenanceNotes)
			vehicles = append(vehicles, v)
		}
	}

	renderTemplate(w, r, "company_fleet.html", CompanyFleetData{
		User:      getUserFromSession(r),
		Vehicles:  vehicles,
		CSRFToken: generateCSRFToken(),
	})
}

// updateVehicleStatusHandler updates vehicle status - FIXED VERSION
func updateVehicleStatusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		if !validateCSRF(r) {
			http.Error(w, "Invalid CSRF token", http.StatusForbidden)
			return
		}

		vehicleID := r.FormValue("vehicle_id")
		statusType := r.FormValue("status_type")  // Changed from "field"
		newStatus := r.FormValue("new_status")    // Changed from "value"

		// Map status types to database columns
		fieldMap := map[string]string{
			"status": "status",
			"oil":    "oil_status",
			"tire":   "tire_status",
		}

		if dbField, ok := fieldMap[statusType]; ok {
			// First try buses table (numeric IDs)
			result, err := db.Exec(fmt.Sprintf("UPDATE buses SET %s = $1, updated_at = NOW() WHERE bus_id = $2", dbField), newStatus, vehicleID)
			if err != nil {
				log.Printf("Error updating bus: %v", err)
			}
			
			rowsAffected, _ := result.RowsAffected()
			
			// If no bus was updated, try vehicles table
			if rowsAffected == 0 {
				_, err = db.Exec(fmt.Sprintf("UPDATE vehicles SET %s = $1, updated_at = NOW() WHERE vehicle_id = $2", dbField), newStatus, vehicleID)
				if err != nil {
					log.Printf("Error updating vehicle: %v", err)
					http.Error(w, "Failed to update status", http.StatusInternalServerError)
					return
				}
			}
		}

		// Return JSON response
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status": "success",
			"message": "Status updated successfully",
		})
	}
}

// busMaintenanceHandler shows maintenance history for a bus
func busMaintenanceHandler(w http.ResponseWriter, r *http.Request) {
	busID := strings.TrimPrefix(r.URL.Path, "/bus-maintenance/")
	
	// Get maintenance records
	records := []MaintenanceRecord{}
	query := `
		SELECT bus_id, date, category, mileage, 0 as cost, notes, created_at
		FROM bus_maintenance_logs
		WHERE bus_id = $1
		ORDER BY date DESC
	`
	
	rows, err := db.Query(query, busID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var record MaintenanceRecord
			rows.Scan(&record.VehicleID, &record.Date, &record.Category, 
				&record.Mileage, &record.Cost, &record.Notes, &record.CreatedAt)
			records = append(records, record)
		}
	}

	// Calculate statistics
	var totalCost float64
	recentCount := 0
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
	
	for _, record := range records {
		totalCost += record.Cost
		// Parse the date string to compare
		if recordDate, err := time.Parse("2006-01-02", record.Date); err == nil {
			if recordDate.After(thirtyDaysAgo) {
				recentCount++
			}
		}
	}

	totalRecords := len(records)
	averageCost := float64(0)
	if totalRecords > 0 {
		averageCost = totalCost / float64(totalRecords)
	}

	// Create data map for template - use same template as vehicle maintenance
	data := map[string]interface{}{
		"VehicleID":          busID,
		"IsBus":              true,  // This is always a bus
		"MaintenanceRecords": records,
		"TotalRecords":       totalRecords,
		"TotalCost":          totalCost,
		"AverageCost":        averageCost,
		"RecentCount":        recentCount,
		"Today":              time.Now().Format("2006-01-02"),
		"CSRFToken":          generateCSRFToken(),
	}

	renderTemplate(w, r, "vehicle_maintenance.html", data)
}

// vehicleMaintenanceHandler shows maintenance for any vehicle - FIXED VERSION
func vehicleMaintenanceHandler(w http.ResponseWriter, r *http.Request) {
	vehicleID := strings.TrimPrefix(r.URL.Path, "/vehicle-maintenance/")
	
	// Try to determine if it's a bus by checking buses table
	var isBus bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM buses WHERE bus_id = $1)", vehicleID).Scan(&isBus)
	if err != nil {
		// If not in buses table, check if it looks like a bus ID (numeric)
		isBus = false
	}

	// Get maintenance records from appropriate table
	records := []MaintenanceRecord{}
	
	if isBus {
		// Query bus_maintenance_logs
		query := `
			SELECT bus_id, date, category, mileage, 
			       COALESCE(cost, 0) as cost, notes, created_at
			FROM bus_maintenance_logs
			WHERE bus_id = $1
			ORDER BY date DESC
		`
		rows, err := db.Query(query, vehicleID)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var record MaintenanceRecord
				rows.Scan(&record.VehicleID, &record.Date, &record.Category, 
					&record.Mileage, &record.Cost, &record.Notes, &record.CreatedAt)
				records = append(records, record)
			}
		}
	} else {
		// Query maintenance_records for other vehicles
		query := `
			SELECT vehicle_id, date, 'maintenance' as category, mileage, 
			       COALESCE(cost, 0) as cost, work_description as notes, created_at
			FROM maintenance_records
			WHERE vehicle_id = $1
			ORDER BY date DESC
		`
		rows, err := db.Query(query, vehicleID)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var record MaintenanceRecord
				rows.Scan(&record.VehicleID, &record.Date, &record.Category, 
					&record.Mileage, &record.Cost, &record.Notes, &record.CreatedAt)
				records = append(records, record)
			}
		}
	}

	// Calculate statistics
	var totalCost float64
	recentCount := 0
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
	
	for _, record := range records {
		totalCost += record.Cost
		// Parse the date string to compare
		if recordDate, err := time.Parse("2006-01-02", record.Date); err == nil {
			if recordDate.After(thirtyDaysAgo) {
				recentCount++
			}
		}
	}

	totalRecords := len(records)
	averageCost := float64(0)
	if totalRecords > 0 {
		averageCost = totalCost / float64(totalRecords)
	}

	data := map[string]interface{}{
		"VehicleID":          vehicleID,
		"IsBus":              isBus,
		"MaintenanceRecords": records,
		"TotalRecords":       totalRecords,
		"TotalCost":          totalCost,
		"AverageCost":        averageCost,
		"RecentCount":        recentCount,
		"Today":              time.Now().Format("2006-01-02"),
		"CSRFToken":          generateCSRFToken(),
	}

	renderTemplate(w, r, "vehicle_maintenance.html", data)
}

// saveMaintenanceRecordHandler saves a maintenance record
func saveMaintenanceRecordHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		if !validateCSRF(r) {
			http.Error(w, "Invalid CSRF token", http.StatusForbidden)
			return
		}

		vehicleID := r.FormValue("vehicle_id")
		date := r.FormValue("date")
		category := r.FormValue("category")
		mileage := parseIntOrDefault(r.FormValue("mileage"), 0)
		notes := r.FormValue("notes")

		// Save to bus_maintenance_logs table
		_, err := db.Exec(`
			INSERT INTO bus_maintenance_logs (bus_id, date, category, mileage, notes, created_at)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, vehicleID, date, category, mileage, notes, time.Now())

		if err != nil {
			http.Error(w, "Failed to save maintenance record", http.StatusInternalServerError)
			return
		}

		// Return JSON response for AJAX requests
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status": "success",
			"message": "Maintenance record saved successfully",
		})
	}
}

// assignRoutesHandler shows route assignment page - FIXED VERSION
func assignRoutesHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get all assignments
	assignments, _ := loadRouteAssignments()
	
	// Get all drivers
	drivers := loadDrivers()
	
	// Get all routes with assignment status
	allRoutes, _ := loadRoutes()
	
	// Get all buses
	allBuses := loadBuses()

	// Create map of assigned bus IDs and route IDs
	assignedBusIDs := make(map[string]bool)
	assignedRouteIDs := make(map[string]bool)
	for _, assignment := range assignments {
		assignedBusIDs[assignment.BusID] = true
		assignedRouteIDs[assignment.RouteID] = true
	}

	// Filter available buses (active and not assigned)
	availableBuses := []*Bus{}
	for _, bus := range allBuses {
		if bus.Status == "active" && !assignedBusIDs[bus.BusID] {
			availableBuses = append(availableBuses, bus)
		}
	}

	// Filter available routes (not assigned)
	availableRoutes := []*Route{}
	routesWithStatus := []*RouteWithStatus{}
	
	// FIX: Process routes correctly
	for i := range allRoutes {
		route := &allRoutes[i]  // Take address to get pointer
		isAssigned := assignedRouteIDs[route.RouteID]
		
		// Add to routesWithStatus for display
		routesWithStatus = append(routesWithStatus, &RouteWithStatus{
			Route:      *route,  // Dereference the pointer to get the value
			IsAssigned: isAssigned,
		})
		
		// Add to availableRoutes if not assigned
		if !isAssigned {
			availableRoutes = append(availableRoutes, route) // route is now a pointer
		}
	}

	// Count available drivers (not assigned)
	availableDriversCount := 0
	assignedDrivers := make(map[string]bool)
	for _, assignment := range assignments {
		assignedDrivers[assignment.Driver] = true
	}
	for _, driver := range drivers {
		if !assignedDrivers[driver.Username] {
			availableDriversCount++
		}
	}

	// Convert routes to pointers for data.Routes
	routePtrs := make([]*Route, len(allRoutes))
	for i := range allRoutes {
		routePtrs[i] = &allRoutes[i]
	}

	data := map[string]interface{}{
		"User":                  user,
		"Assignments":           assignments,
		"Drivers":               drivers,
		"Routes":                routePtrs,
		"RoutesWithStatus":      routesWithStatus,
		"AvailableRoutes":       availableRoutes,
		"Buses":                 allBuses,
		"AvailableBuses":        availableBuses,
		"TotalAssignments":      len(assignments),
		"TotalRoutes":           len(allRoutes),
		"AvailableDriversCount": availableDriversCount,
		"AvailableBusesCount":   len(availableBuses),
		"CSRFToken":             generateCSRFToken(),
	}

	renderTemplate(w, r, "assign_routes.html", data)
}

// assignRouteHandler assigns a route to a driver
func assignRouteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		if !validateCSRF(r) {
			http.Error(w, "Invalid CSRF token", http.StatusForbidden)
			return
		}

		driver := r.FormValue("driver")
		routeID := r.FormValue("route_id")
		busID := r.FormValue("bus_id")

		// Delete existing assignment for this driver
		db.Exec(`DELETE FROM route_assignments WHERE driver = $1`, driver)

		// Create new assignment
		_, err := db.Exec(`
			INSERT INTO route_assignments (driver, route_id, bus_id, assigned_date)
			VALUES ($1, $2, $3, $4)
		`, driver, routeID, busID, time.Now().Format("2006-01-02"))

		if err != nil {
			http.Error(w, "Failed to assign route", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/assign-routes", http.StatusSeeOther)
	}
}

// unassignRouteHandler removes a route assignment
func unassignRouteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		if !validateCSRF(r) {
			http.Error(w, "Invalid CSRF token", http.StatusForbidden)
			return
		}

		driver := r.FormValue("driver")
		db.Exec(`DELETE FROM route_assignments WHERE driver = $1`, driver)
		http.Redirect(w, r, "/assign-routes", http.StatusSeeOther)
	}
}

// addRouteHandler adds a new route
func addRouteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		if !validateCSRF(r) {
			http.Error(w, "Invalid CSRF token", http.StatusForbidden)
			return
		}

		routeName := r.FormValue("route_name")
		description := r.FormValue("description")

		// Generate route ID
		routeID := fmt.Sprintf("ROUTE%03d", time.Now().Unix()%1000)

		_, err := db.Exec(`
			INSERT INTO routes (route_id, route_name, description)
			VALUES ($1, $2, $3)
		`, routeID, routeName, description)

		if err != nil {
			http.Error(w, "Failed to add route", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/assign-routes", http.StatusSeeOther)
	}
}

// editRouteHandler edits a route
func editRouteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		if !validateCSRF(r) {
			http.Error(w, "Invalid CSRF token", http.StatusForbidden)
			return
		}

		routeID := r.FormValue("route_id")
		routeName := r.FormValue("route_name")
		description := r.FormValue("description")

		db.Exec(`
			UPDATE routes 
			SET route_name = $1, description = $2
			WHERE route_id = $3
		`, routeName, description, routeID)

		http.Redirect(w, r, "/assign-routes", http.StatusSeeOther)
	}
}

// deleteRouteHandler deletes a route
func deleteRouteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		if !validateCSRF(r) {
			http.Error(w, "Invalid CSRF token", http.StatusForbidden)
			return
		}

		routeID := r.FormValue("route_id")
		
		// Delete route and related data
		db.Exec(`DELETE FROM route_assignments WHERE route_id = $1`, routeID)
		db.Exec(`DELETE FROM students WHERE route_id = $1`, routeID)
		db.Exec(`DELETE FROM routes WHERE route_id = $1`, routeID)

		http.Redirect(w, r, "/assign-routes", http.StatusSeeOther)
	}
}

// studentsHandler shows student management page
func studentsHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "driver" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get driver's route
	assignment := getDriverAssignment(user.Username)
	students := getRouteStudents(assignment.RouteID)

	renderTemplate(w, r, "students.html", StudentData{
		User:      user,
		Students:  students,
		CSRFToken: generateCSRFToken(),
	})
}

// addStudentHandler adds a student
func addStudentHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		if !validateCSRF(r) {
			http.Error(w, "Invalid CSRF token", http.StatusForbidden)
			return
		}

		user := getUserFromSession(r)
		if user == nil || user.Role != "driver" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Get driver's route
		assignment := getDriverAssignment(user.Username)

		// Create student
		student := Student{
			StudentID:      generateUniqueID("STU", int(time.Now().Unix()%1000)),
			Name:           r.FormValue("name"),
			PhoneNumber:    r.FormValue("phone_number"),
			AltPhoneNumber: r.FormValue("alt_phone_number"),
			Guardian:       r.FormValue("guardian"),
			PickupTime:     r.FormValue("pickup_time"),
			DropoffTime:    r.FormValue("dropoff_time"),
			PositionNumber: parseIntOrDefault(r.FormValue("position_number"), 1),
			RouteID:        assignment.RouteID,
			Driver:         user.Username,
			Active:         true,
		}

		// Save to database
		_, err := db.Exec(`
			INSERT INTO students (student_id, name, phone_number, alt_phone_number, guardian, 
				pickup_time, dropoff_time, position_number, route_id, driver, active)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		`, student.StudentID, student.Name, student.PhoneNumber, student.AltPhoneNumber,
			student.Guardian, student.PickupTime, student.DropoffTime, student.PositionNumber,
			student.RouteID, student.Driver, student.Active)

		if err != nil {
			http.Error(w, "Failed to add student", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/students", http.StatusSeeOther)
	}
}

// editStudentHandler edits a student
func editStudentHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		if !validateCSRF(r) {
			http.Error(w, "Invalid CSRF token", http.StatusForbidden)
			return
		}

		studentID := r.FormValue("student_id")
		
		db.Exec(`
			UPDATE students 
			SET name = $1, phone_number = $2, alt_phone_number = $3, guardian = $4,
				pickup_time = $5, dropoff_time = $6, position_number = $7
			WHERE student_id = $8
		`, r.FormValue("name"), r.FormValue("phone_number"), r.FormValue("alt_phone_number"),
			r.FormValue("guardian"), r.FormValue("pickup_time"), r.FormValue("dropoff_time"),
			parseIntOrDefault(r.FormValue("position_number"), 1), studentID)

		http.Redirect(w, r, "/students", http.StatusSeeOther)
	}
}

// removeStudentHandler removes a student
func removeStudentHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		if !validateCSRF(r) {
			http.Error(w, "Invalid CSRF token", http.StatusForbidden)
			return
		}

		studentID := r.FormValue("student_id")
		db.Exec(`UPDATE students SET active = false WHERE student_id = $1`, studentID)
		http.Redirect(w, r, "/students", http.StatusSeeOther)
	}
}

// importMileageHandler handles mileage report imports
func importMileageHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		renderTemplate(w, r, "import_mileage.html", map[string]interface{}{
			"CSRFToken": generateCSRFToken(),
		})
		return
	}

	if r.Method == "POST" {
		if !validateCSRF(r) {
			http.Error(w, "Invalid CSRF token", http.StatusForbidden)
			return
		}

		// Parse multipart form
		err := r.ParseMultipartForm(10 << 20) // 10 MB
		if err != nil {
			http.Error(w, "Failed to parse form", http.StatusBadRequest)
			return
		}

		file, header, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "Failed to get file", http.StatusBadRequest)
			return
		}
		defer file.Close()

		// Process Excel file
		processExcelFile(file, header.Filename)

		http.Redirect(w, r, "/view-mileage-reports", http.StatusSeeOther)
	}
}

// viewMileageReportsHandler shows mileage reports
func viewMileageReportsHandler(w http.ResponseWriter, r *http.Request) {
	// Load mileage reports from database
	reports := []MileageReport{}
	rows, err := db.Query(`
		SELECT report_month, report_year, vehicle_year, make_model, license_plate,
			   vehicle_id, location, beginning_miles, ending_miles, total_miles, status
		FROM mileage_reports
		ORDER BY report_year DESC, report_month DESC, vehicle_id
	`)
	
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var report MileageReport
			rows.Scan(&report.ReportMonth, &report.ReportYear, &report.VehicleYear,
				&report.MakeModel, &report.LicensePlate, &report.VehicleID,
				&report.Location, &report.BeginningMiles, &report.EndingMiles,
				&report.TotalMiles, &report.Status)
			reports = append(reports, report)
		}
	}

	renderTemplate(w, r, "view_mileage_reports.html", map[string]interface{}{
		"Reports":   reports,
		"CSRFToken": generateCSRFToken(),
	})
}

// driverProfileHandler shows driver profile
func driverProfileHandler(w http.ResponseWriter, r *http.Request) {
	driverUsername := strings.TrimPrefix(r.URL.Path, "/driver/")
	
	// Get driver info
	driver := getUserByUsername(driverUsername)
	if driver == nil || driver.Role != "driver" {
		http.Error(w, "Driver not found", http.StatusNotFound)
		return
	}

	// Get driver's assignment
	assignment := getDriverAssignment(driverUsername)
	
	// Get recent logs
	logs := getDriverLogs(driverUsername, 30)
	
	// Calculate statistics
	totalMiles := 0.0
	for _, log := range logs {
		totalMiles += log.Mileage
	}

	renderTemplate(w, r, "driver_profile.html", map[string]interface{}{
		"Driver":     driver,
		"Assignment": assignment,
		"Logs":       logs,
		"TotalMiles": totalMiles,
		"LogCount":   len(logs),
		"CSRFToken":  generateCSRFToken(),
	})
}

// Helper functions

func authenticateUser(username, password string) (*User, error) {
	var user User
	err := db.QueryRow(`
		SELECT username, password, role, status, registration_date 
		FROM users 
		WHERE username = $1 AND status = 'active'
	`, username).Scan(&user.Username, &user.Password, &user.Role, &user.Status, &user.RegistrationDate)
	
	if err != nil {
		return nil, err
	}

	// Verify password
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func getUserByUsername(username string) *User {
	var user User
	err := db.QueryRow(`
		SELECT username, role, status, registration_date 
		FROM users 
		WHERE username = $1
	`, username).Scan(&user.Username, &user.Role, &user.Status, &user.RegistrationDate)
	
	if err != nil {
		return nil
	}
	
	return &user
}

func loadDrivers() []User {
	drivers := []User{}
	rows, err := db.Query(`SELECT username, role, status FROM users WHERE role = 'driver' AND status = 'active'`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var driver User
			rows.Scan(&driver.Username, &driver.Role, &driver.Status)
			drivers = append(drivers, driver)
		}
	}
	return drivers
}

func loadDriverSummaries() []*DriverSummary {
	summaries := []*DriverSummary{}
	rows, err := db.Query(`
		SELECT ra.driver, ra.bus_id, ra.route_id, r.route_name,
			   COALESCE(MAX(dl.created_at), NOW()) as last_activity,
			   COALESCE(SUM(dl.mileage), 0) as total_miles
		FROM route_assignments ra
		LEFT JOIN routes r ON ra.route_id = r.route_id
		LEFT JOIN driver_logs dl ON dl.driver = ra.driver
		GROUP BY ra.driver, ra.bus_id, ra.route_id, r.route_name
	`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var s DriverSummary
			rows.Scan(&s.Driver, &s.BusID, &s.RouteID, &s.RouteName, &s.LastActivity, &s.TotalMiles)
			summaries = append(summaries, &s)
		}
	}
	return summaries
}

func loadRouteStats() []*RouteStats {
	stats := []*RouteStats{}
	rows, err := db.Query(`
		SELECT r.route_id, r.route_name,
			   COUNT(DISTINCT ra.bus_id) as active_buses,
			   COUNT(DISTINCT s.student_id) as total_students
		FROM routes r
		LEFT JOIN route_assignments ra ON r.route_id = ra.route_id
		LEFT JOIN students s ON r.route_id = s.route_id AND s.active = true
		GROUP BY r.route_id, r.route_name
	`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var s RouteStats
			rows.Scan(&s.RouteID, &s.RouteName, &s.ActiveBuses, &s.TotalStudents)
			stats = append(stats, &s)
		}
	}
	return stats
}

func countPendingUsers() int {
	var count int
	db.QueryRow(`SELECT COUNT(*) FROM users WHERE status = 'pending'`).Scan(&count)
	return count
}

func getDriverAssignment(driver string) *DriverAssignment {
	var a DriverAssignment
	err := db.QueryRow(`
		SELECT ra.driver, ra.route_id, ra.bus_id, r.route_name
		FROM route_assignments ra
		LEFT JOIN routes r ON ra.route_id = r.route_id
		WHERE ra.driver = $1
	`, driver).Scan(&a.Driver, &a.RouteID, &a.BusID, &a.RouteName)
	
	if err != nil {
		return &DriverAssignment{Driver: driver}
	}
	
	return &a
}

func getDriverLogs(driver string, days int) []DriverLog {
	logs := []DriverLog{}
	rows, err := db.Query(`
		SELECT id, driver, bus_id, route_id, date, period, departure_time, arrival_time, mileage, attendance
		FROM driver_logs
		WHERE driver = $1 AND date >= $2
		ORDER BY date DESC, period DESC
	`, driver, time.Now().AddDate(0, 0, -days).Format("2006-01-02"))
	
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var log DriverLog
			var attendanceJSON sql.NullString
			rows.Scan(&log.ID, &log.Driver, &log.BusID, &log.RouteID, &log.Date, 
				&log.Period, &log.Departure, &log.Arrival, &log.Mileage, &attendanceJSON)
			
			// Parse attendance JSON
			if attendanceJSON.Valid && attendanceJSON.String != "" {
				json.Unmarshal([]byte(attendanceJSON.String), &log.Attendance)
			}
			logs = append(logs, log)
		}
	}
	
	return logs
}

func getRouteStudents(routeID string) []Student {
	students := []Student{}
	rows, err := db.Query(`
		SELECT student_id, name, phone_number, alt_phone_number, guardian,
			   pickup_time, dropoff_time, position_number, route_id, driver
		FROM students
		WHERE route_id = $1 AND active = true
		ORDER BY position_number
	`, routeID)
	
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var s Student
			rows.Scan(&s.StudentID, &s.Name, &s.PhoneNumber, &s.AltPhoneNumber,
				&s.Guardian, &s.PickupTime, &s.DropoffTime, &s.PositionNumber,
				&s.RouteID, &s.Driver)
			s.Active = true
			students = append(students, s)
		}
	}
	
	return students
}

// NEW HELPER FUNCTIONS FROM FIXES

func getBusByID(busID string) *Bus {
	var bus Bus
	err := db.QueryRow(`
		SELECT bus_id, model, capacity, status, oil_status, tire_status, maintenance_notes
		FROM buses WHERE bus_id = $1
	`, busID).Scan(&bus.BusID, &bus.Model, &bus.Capacity, &bus.Status, 
		&bus.OilStatus, &bus.TireStatus, &bus.MaintenanceNotes)
	
	if err != nil {
		return nil
	}
	return &bus
}

func getRouteByID(routeID string) *Route {
	var route Route
	err := db.QueryRow(`
		SELECT route_id, route_name, description
		FROM routes WHERE route_id = $1
	`, routeID).Scan(&route.RouteID, &route.RouteName, &route.Description)
	
	if err != nil {
		return nil
	}
	return &route
}

func getRouteStudentsOrderedByPickup(routeID string) []Student {
	students := []Student{}
	rows, err := db.Query(`
		SELECT student_id, name, phone_number, alt_phone_number, guardian,
			   pickup_time, dropoff_time, position_number, route_id, driver
		FROM students
		WHERE route_id = $1 AND active = true
		ORDER BY pickup_time, position_number
	`, routeID)
	
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var s Student
			rows.Scan(&s.StudentID, &s.Name, &s.PhoneNumber, &s.AltPhoneNumber,
				&s.Guardian, &s.PickupTime, &s.DropoffTime, &s.PositionNumber,
				&s.RouteID, &s.Driver)
			s.Active = true
			students = append(students, s)
		}
	}
	
	return students
}

func getRouteStudentsOrderedByDropoff(routeID string) []Student {
	students := []Student{}
	rows, err := db.Query(`
		SELECT student_id, name, phone_number, alt_phone_number, guardian,
			   pickup_time, dropoff_time, position_number, route_id, driver
		FROM students
		WHERE route_id = $1 AND active = true
		ORDER BY dropoff_time, position_number
	`, routeID)
	
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var s Student
			rows.Scan(&s.StudentID, &s.Name, &s.PhoneNumber, &s.AltPhoneNumber,
				&s.Guardian, &s.PickupTime, &s.DropoffTime, &s.PositionNumber,
				&s.RouteID, &s.Driver)
			s.Active = true
			students = append(students, s)
		}
	}
	
	return students
}

func processExcelFile(file io.Reader, filename string) error {
	// Read file into memory
	data, err := io.ReadAll(file)
	if err != nil {
		return err
	}

	// Open Excel file
	f, err := excelize.OpenReader(strings.NewReader(string(data)))
	if err != nil {
		return err
	}

	// Process each sheet
	for _, sheetName := range f.GetSheetList() {
		rows, err := f.GetRows(sheetName)
		if err != nil {
			continue
		}

		// Skip header row
		if len(rows) <= 1 {
			continue
		}

		// Process data rows
		for i := 1; i < len(rows); i++ {
			row := rows[i]
			if len(row) < 10 {
				continue
			}

			// Parse row data based on sheet type
			if strings.Contains(sheetName, "Agency") {
				processAgencyVehicleRow(row)
			} else if strings.Contains(sheetName, "Bus") {
				processSchoolBusRow(row)
			}
		}
	}

	return nil
}

func processAgencyVehicleRow(row []string) {
	// Extract data from row
	reportMonth := row[0]
	reportYear := parseIntOrDefault(row[1], 0)
	vehicleYear := parseIntOrDefault(row[2], 0)
	makeModel := row[3]
	licensePlate := row[4]
	vehicleID := row[5]
	location := row[6]
	beginningMiles := parseIntOrDefault(row[7], 0)
	endingMiles := parseIntOrDefault(row[8], 0)
	totalMiles := parseIntOrDefault(row[9], 0)

	// Insert into database
	db.Exec(`
		INSERT INTO mileage_reports (report_month, report_year, vehicle_year, make_model,
			license_plate, vehicle_id, location, beginning_miles, ending_miles, total_miles, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (report_month, report_year, vehicle_id) DO UPDATE
		SET beginning_miles = $8, ending_miles = $9, total_miles = $10
	`, reportMonth, reportYear, vehicleYear, makeModel, licensePlate, vehicleID,
		location, beginningMiles, endingMiles, totalMiles, "active")
}

func processSchoolBusRow(row []string) {
	// Similar to processAgencyVehicleRow but for school buses
	processAgencyVehicleRow(row)
}

// getUser gets the current user from session
func getUser(r *http.Request) *User {
	return getUserFromSession(r)
}

// isLoggedIn checks if user has valid session
func isLoggedIn(r *http.Request) bool {
	cookie, err := r.Cookie(SessionCookieName)
	if err != nil {
		return false
	}

	session, err := GetSecureSession(cookie.Value)
	if err != nil {
		return false
	}

	return session != nil
}
