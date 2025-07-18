package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// loginHandler handles the login page and authentication
func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		data := map[string]interface{}{
			"CSRFToken": generateCSRFToken(),
		}
		renderTemplate(w, r, "login.html", data)
		return
	}

	if r.Method == "POST" {
		// NOTE: Login page doesn't have a session yet, so CSRF validation 
		// would always fail. Common practice is to skip CSRF for login
		// but use other protections like rate limiting.
		
		// Parse form data
		if err := r.ParseForm(); err != nil {
			log.Printf("Failed to parse form: %v", err)
			http.Error(w, "Failed to parse form", http.StatusBadRequest)
			return
		}
		
		username := r.FormValue("username")
		password := r.FormValue("password")
		
		// Log for debugging
		log.Printf("Login attempt for username: %s from IP: %s", username, getClientIP(r))
		
		if !rateLimiter.Allow(getClientIP(r)) {
			log.Printf("Rate limit exceeded for IP: %s", getClientIP(r))
			http.Error(w, "Too many login attempts. Please try again later.", http.StatusTooManyRequests)
			return
		}
		
		user, err := authenticateUser(username, password)
		if err != nil {
			log.Printf("Authentication failed for %s: %v", username, err)
			data := map[string]interface{}{
				"Error":     "Invalid username or password",
				"CSRFToken": generateCSRFToken(),
			}
			renderTemplate(w, r, "login.html", data)
			return
		}
		
		if user.Status != "active" {
			log.Printf("User %s is not active, status: %s", username, user.Status)
			data := map[string]interface{}{
				"Error":     "Your account is pending approval",
				"CSRFToken": generateCSRFToken(),
			}
			renderTemplate(w, r, "login.html", data)
			return
		}
		
		// Authentication successful
		log.Printf("Login successful for user: %s (role: %s)", username, user.Role)
		
		sessionToken := generateSessionToken()
		storeSession(sessionToken, user)
		log.Printf("Session created with token: %s for user: %s", sessionToken[:8]+"...", username)
		
		// Detect if we're on HTTPS
		isHTTPS := r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"
		
		http.SetCookie(w, &http.Cookie{
			Name:     SessionCookieName,
			Value:    sessionToken,
			Path:     "/",
			HttpOnly: true,
			Secure:   isHTTPS, // Set based on protocol
			SameSite: http.SameSiteLaxMode,
			MaxAge:   86400,
		})
		
		log.Printf("Cookie set, redirecting user %s (role: %s)", username, user.Role)
		
		if user.Role == "manager" {
			log.Printf("Redirecting to manager dashboard")
			http.Redirect(w, r, "/manager-dashboard", http.StatusSeeOther)
		} else {
			log.Printf("Redirecting to driver dashboard")
			http.Redirect(w, r, "/driver-dashboard", http.StatusSeeOther)
		}
	}
}

// logoutHandler handles user logout
func logoutHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	cookie, err := r.Cookie(SessionCookieName)
	if err == nil {
		deleteSession(cookie.Value)
	}
	
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})
	
	http.Redirect(w, r, "/", http.StatusFound)
}

// registerHandler handles user registration
func registerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		data := map[string]interface{}{
			"CSRFToken": generateCSRFToken(),
		}
		renderTemplate(w, r, "register.html", data)
		return
	}
	
	if r.Method == "POST" {
		username := r.FormValue("username")
		password := r.FormValue("password")
		confirmPassword := r.FormValue("confirm_password")
		
		if password != confirmPassword {
			data := map[string]interface{}{
				"Error":     "Passwords do not match",
				"CSRFToken": generateCSRFToken(),
			}
			renderTemplate(w, r, "register.html", data)
			return
		}
		
		err := createUser(username, password, "driver", "pending")
		if err != nil {
			data := map[string]interface{}{
				"Error":     "Username already exists",
				"CSRFToken": generateCSRFToken(),
			}
			renderTemplate(w, r, "register.html", data)
			return
		}
		
		data := map[string]interface{}{
			"Success":   true,
			"CSRFToken": generateCSRFToken(),
		}
		renderTemplate(w, r, "register.html", data)
	}
}

// managerDashboardHandler shows the manager dashboard with maintenance overview
func managerDashboardHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	// Get various statistics
	buses, _ := dataCache.getBuses()
	vehicles, _ := dataCache.getVehicles()
	users, _ := dataCache.getUsers()
	routes, _ := dataCache.getRoutes()

	// Count vehicles needing maintenance
	maintenanceNeeded := 0
	
	// Check buses
	for _, bus := range buses {
		if bus.OilStatus == "needs_service" || bus.OilStatus == "overdue" ||
			bus.TireStatus == "worn" || bus.TireStatus == "replace" {
			maintenanceNeeded++
		}
	}
	
	// Check vehicles
	for _, vehicle := range vehicles {
		if vehicle.OilStatus == "needs_service" || vehicle.OilStatus == "overdue" ||
			vehicle.TireStatus == "worn" || vehicle.TireStatus == "replace" {
			maintenanceNeeded++
		}
	}

	// Get pending users
	pendingUsers := 0
	for _, u := range users {
		if u.Status == "pending" {
			pendingUsers++
		}
	}

	data := map[string]interface{}{
		"User":              user,
		"CSRFToken":         getSessionCSRFToken(r),
		"TotalBuses":        len(buses),
		"TotalVehicles":     len(vehicles),
		"TotalDrivers":      len(users) - 1, // Exclude manager
		"TotalRoutes":       len(routes),
		"MaintenanceNeeded": maintenanceNeeded,
		"PendingUsers":      pendingUsers,
	}

	renderTemplate(w, r, "dashboard.html", data)
}

// driverDashboardHandler with maintenance alerts
func driverDashboardHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	// Get driver's assignments
	assignments, err := getDriverAssignments(user.Username)
	if err != nil {
		log.Printf("Error loading assignments: %v", err)
	}

	// Check for maintenance alerts on assigned buses
	var maintenanceAlerts []MaintenanceAlert
	for _, assignment := range assignments {
		alerts, err := checkMaintenanceDue(assignment.BusID)
		if err != nil {
			log.Printf("Error checking maintenance for bus %s: %v", assignment.BusID, err)
			continue
		}
		maintenanceAlerts = append(maintenanceAlerts, alerts...)
	}

	// Get students for assigned routes
	studentsMap := make(map[string][]Student)
	for _, assignment := range assignments {
		students, err := getStudentsByRoute(assignment.RouteID)
		if err != nil {
			log.Printf("Error loading students for route %s: %v", assignment.RouteID, err)
			continue
		}
		studentsMap[assignment.RouteID] = students
	}

	// Check for success message
	success := r.URL.Query().Get("success") == "true"

	data := map[string]interface{}{
		"User":               user,
		"CSRFToken":          getSessionCSRFToken(r),
		"Assignments":        assignments,
		"StudentsMap":        studentsMap,
		"MaintenanceAlerts":  maintenanceAlerts,
		"Success":            success,
		"CurrentDate":        time.Now().Format("2006-01-02"),
	}

	renderTemplate(w, r, "driver_dashboard.html", data)
}

// fleetHandler shows the fleet overview with maintenance alerts
func fleetHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	buses, err := dataCache.getBuses()
	if err != nil {
		log.Printf("Error loading buses: %v", err)
		http.Error(w, "Failed to load fleet data", http.StatusInternalServerError)
		return
	}

	// Get maintenance alerts for all buses
	allAlerts := make(map[string][]MaintenanceAlert)
	for _, bus := range buses {
		alerts, err := checkMaintenanceDue(bus.BusID)
		if err != nil {
			log.Printf("Error checking maintenance for bus %s: %v", bus.BusID, err)
			continue
		}
		if len(alerts) > 0 {
			allAlerts[bus.BusID] = alerts
		}
	}

	data := map[string]interface{}{
		"User":               user,
		"CSRFToken":          getSessionCSRFToken(r),
		"Buses":              buses,
		"MaintenanceAlerts":  allAlerts,
	}

	renderTemplate(w, r, "fleet.html", data)
}

// companyFleetHandler shows company vehicles with maintenance alerts
func companyFleetHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	vehicles, err := dataCache.getVehicles()
	if err != nil {
		log.Printf("Error loading vehicles: %v", err)
		http.Error(w, "Failed to load vehicle data", http.StatusInternalServerError)
		return
	}

	// Get maintenance alerts for all vehicles
	allAlerts := make(map[string][]MaintenanceAlert)
	for _, vehicle := range vehicles {
		alerts, err := checkMaintenanceDue(vehicle.VehicleID)
		if err != nil {
			log.Printf("Error checking maintenance for vehicle %s: %v", vehicle.VehicleID, err)
			continue
		}
		if len(alerts) > 0 {
			allAlerts[vehicle.VehicleID] = alerts
		}
	}

	data := map[string]interface{}{
		"User":              user,
		"CSRFToken":         getSessionCSRFToken(r),
		"Vehicles":          vehicles,
		"MaintenanceAlerts": allAlerts,
	}

	renderTemplate(w, r, "company_fleet.html", data)
}

// busMaintenanceHandler shows maintenance history for a bus
func busMaintenanceHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	busID := strings.TrimPrefix(r.URL.Path, "/bus-maintenance/")
	if busID == "" {
		http.Error(w, "Bus ID required", http.StatusBadRequest)
		return
	}

	// Get maintenance logs using the fixed function
	logs, err := getMaintenanceLogsForVehicle(busID)
	if err != nil {
		log.Printf("Error loading maintenance logs: %v", err)
		logs = []CombinedMaintenanceLog{} // Show empty list instead of error
	}

	// Get bus details
	buses, err := dataCache.getBuses()
	if err != nil {
		http.Error(w, "Failed to load bus data", http.StatusInternalServerError)
		return
	}

	var bus *Bus
	for _, b := range buses {
		if b.BusID == busID {
			bus = &b
			break
		}
	}

	if bus == nil {
		http.Error(w, "Bus not found", http.StatusNotFound)
		return
	}

	// Check for maintenance alerts
	alerts, err := checkMaintenanceDue(busID)
	if err != nil {
		log.Printf("Error checking maintenance due: %v", err)
		alerts = []MaintenanceAlert{}
	}

	data := map[string]interface{}{
		"User":             user,
		"CSRFToken":        getSessionCSRFToken(r),
		"Bus":              bus,
		"MaintenanceLogs":  logs,
		"MaintenanceAlerts": alerts,
	}

	renderTemplate(w, r, "bus_maintenance.html", data)
}

// vehicleMaintenanceHandler shows maintenance history for a vehicle
func vehicleMaintenanceHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	vehicleID := strings.TrimPrefix(r.URL.Path, "/vehicle-maintenance/")
	if vehicleID == "" {
		http.Error(w, "Vehicle ID required", http.StatusBadRequest)
		return
	}

	// Get maintenance logs using the fixed function
	logs, err := getMaintenanceLogsForVehicle(vehicleID)
	if err != nil {
		log.Printf("Error loading maintenance logs: %v", err)
		logs = []CombinedMaintenanceLog{}
	}

	// Get vehicle details
	vehicles, err := dataCache.getVehicles()
	if err != nil {
		http.Error(w, "Failed to load vehicle data", http.StatusInternalServerError)
		return
	}

	var vehicle *Vehicle
	for _, v := range vehicles {
		if v.VehicleID == vehicleID {
			vehicle = &v
			break
		}
	}

	if vehicle == nil {
		http.Error(w, "Vehicle not found", http.StatusNotFound)
		return
	}

	// Check for maintenance alerts
	alerts, err := checkMaintenanceDue(vehicleID)
	if err != nil {
		log.Printf("Error checking maintenance due: %v", err)
		alerts = []MaintenanceAlert{}
	}

	data := map[string]interface{}{
		"User":              user,
		"CSRFToken":         getSessionCSRFToken(r),
		"Vehicle":           vehicle,
		"MaintenanceLogs":   logs,
		"MaintenanceAlerts": alerts,
	}

	renderTemplate(w, r, "vehicle_maintenance.html", data)
}

// saveMaintenanceRecordHandler handles saving maintenance records
func saveMaintenanceRecordHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user := getUserFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	if !validateCSRF(r) {
		http.Error(w, "Invalid CSRF token", http.StatusForbidden)
		return
	}

	// Parse form
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	vehicleID := r.FormValue("vehicle_id")
	vehicleType := r.FormValue("vehicle_type")
	date := r.FormValue("date")
	category := r.FormValue("category")
	notes := r.FormValue("notes")
	mileageStr := r.FormValue("mileage")
	costStr := r.FormValue("cost")

	// Validate required fields
	if vehicleID == "" || date == "" || category == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	// Parse mileage
	var mileage int
	if mileageStr != "" {
		mileage, err = strconv.Atoi(mileageStr)
		if err != nil {
			http.Error(w, "Invalid mileage", http.StatusBadRequest)
			return
		}

		// Validate mileage entry
		validation := validateMileageEntry(vehicleID, float64(mileage))
		if !validation.Valid {
			http.Error(w, validation.Error, http.StatusBadRequest)
			return
		}
		// Note: We could show warnings to the user here if needed
	}

	// Parse cost
	var cost float64
	if costStr != "" {
		cost, err = strconv.ParseFloat(costStr, 64)
		if err != nil {
			http.Error(w, "Invalid cost", http.StatusBadRequest)
			return
		}
	}

	// Save based on vehicle type
	if vehicleType == "bus" {
		log := BusMaintenanceLog{
			BusID:    vehicleID,
			Date:     date,
			Category: category,
			Notes:    notes,
			Mileage:  mileage,
			Cost:     cost,
		}
		err = saveBusMaintenanceLog(log)
	} else {
		log := VehicleMaintenanceLog{
			VehicleID: vehicleID,
			Date:      date,
			Category:  category,
			Notes:     notes,
			Mileage:   mileage,
			Cost:      cost,
		}
		err = saveVehicleMaintenanceLog(log)
	}

	if err != nil {
		log.Printf("Error saving maintenance record: %v", err)
		http.Error(w, "Failed to save maintenance record", http.StatusInternalServerError)
		return
	}

	// Redirect back to maintenance page
	if vehicleType == "bus" {
		http.Redirect(w, r, "/bus-maintenance/"+vehicleID, http.StatusSeeOther)
	} else {
		http.Redirect(w, r, "/vehicle-maintenance/"+vehicleID, http.StatusSeeOther)
	}
}

// updateVehicleStatusHandler handles status updates with validation
func updateVehicleStatusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user := getUserFromSession(r)
	if user == nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "Unauthorized"})
		return
	}

	// Parse JSON request
	var req struct {
		VehicleID   string `json:"vehicle_id"`
		VehicleType string `json:"vehicle_type"`
		FieldName   string `json:"field_name"`
		FieldValue  string `json:"field_value"`
		CSRFToken   string `json:"csrf_token"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request"})
		return
	}

	// Validate CSRF token
	sessionToken := getSessionCSRFToken(r)
	if req.CSRFToken != sessionToken {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid CSRF token"})
		return
	}

	// Update based on vehicle type
	var err error
	if req.VehicleType == "bus" {
		err = updateBusField(req.VehicleID, req.FieldName, req.FieldValue)
	} else {
		err = updateVehicleField(req.VehicleID, req.FieldName, req.FieldValue)
	}

	if err != nil {
		log.Printf("Error updating %s %s: %v", req.VehicleType, req.VehicleID, err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to update"})
		return
	}

	// If status was updated, check if we need to update maintenance status
	if req.FieldName == "status" || req.FieldName == "oil_status" || req.FieldName == "tire_status" {
		if err := updateMaintenanceStatusBasedOnMileage(req.VehicleID); err != nil {
			log.Printf("Warning: failed to update maintenance status: %v", err)
		}
	}

	// Invalidate cache
	if req.VehicleType == "bus" {
		dataCache.invalidateBuses()
	} else {
		dataCache.invalidateVehicles()
	}

	// Return updated status
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Updated successfully",
	})
}

// saveLogHandler with mileage validation
func saveLogHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user := getUserFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	if !validateCSRF(r) {
		http.Error(w, "Invalid CSRF token", http.StatusForbidden)
		return
	}

	// Parse form
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	// Extract form values
	busID := r.FormValue("bus_id")
	routeID := r.FormValue("route_id")
	date := r.FormValue("date")
	period := r.FormValue("period")
	departure := r.FormValue("departure_time")
	arrival := r.FormValue("arrival_time")
	beginMileageStr := r.FormValue("begin_mileage")
	endMileageStr := r.FormValue("end_mileage")

	// Validate required fields
	if busID == "" || routeID == "" || date == "" || period == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	// Parse mileage
	beginMileage, err := strconv.ParseFloat(beginMileageStr, 64)
	if err != nil {
		http.Error(w, "Invalid begin mileage", http.StatusBadRequest)
		return
	}

	endMileage, err := strconv.ParseFloat(endMileageStr, 64)
	if err != nil {
		http.Error(w, "Invalid end mileage", http.StatusBadRequest)
		return
	}

	// Validate mileage
	if endMileage < beginMileage {
		http.Error(w, "End mileage cannot be less than begin mileage", http.StatusBadRequest)
		return
	}

	// Validate against vehicle's current mileage
	validation := validateMileageEntry(busID, endMileage)
	if !validation.Valid {
		http.Error(w, validation.Error, http.StatusBadRequest)
		return
	}

	// Build attendance JSON
	var attendance []map[string]interface{}
	for key, values := range r.Form {
		if strings.HasPrefix(key, "present_") {
			posStr := strings.TrimPrefix(key, "present_")
			position, _ := strconv.Atoi(posStr)
			pickupTime := r.FormValue("pickup_time_" + posStr)
			
			attendanceRecord := map[string]interface{}{
				"position":    position,
				"present":     values[0] == "true",
				"pickup_time": pickupTime,
			}
			attendance = append(attendance, attendanceRecord)
		}
	}

	attendanceJSON, err := json.Marshal(attendance)
	if err != nil {
		http.Error(w, "Failed to process attendance", http.StatusInternalServerError)
		return
	}

	// Create driver log
	driverLog := DriverLog{
		Driver:       user.Username,
		BusID:        busID,
		RouteID:      routeID,
		Date:         date,
		Period:       period,
		Departure:    departure,
		Arrival:      arrival,
		BeginMileage: beginMileage,
		EndMileage:   endMileage,
		Attendance:   string(attendanceJSON),
	}

	// Save to database
	query := `
		INSERT INTO driver_logs (driver, bus_id, route_id, date, period, departure_time, arrival_time, begin_mileage, end_mileage, attendance)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	
	_, err = db.Exec(query, driverLog.Driver, driverLog.BusID, driverLog.RouteID, 
		driverLog.Date, driverLog.Period, driverLog.Departure, driverLog.Arrival,
		driverLog.BeginMileage, driverLog.EndMileage, driverLog.Attendance)
	
	if err != nil {
		log.Printf("Error saving driver log: %v", err)
		http.Error(w, "Failed to save log", http.StatusInternalServerError)
		return
	}

	// Update vehicle mileage and check maintenance status
	if err := updateVehicleMileage(busID, int(endMileage)); err != nil {
		log.Printf("Warning: failed to update vehicle mileage: %v", err)
	}
	
	if err := updateMaintenanceStatusBasedOnMileage(busID); err != nil {
		log.Printf("Warning: failed to update maintenance status: %v", err)
	}

	// Check if we should show any maintenance alerts
	alerts, err := checkMaintenanceDue(busID)
	if err == nil && len(alerts) > 0 {
		// Store alerts in session to show on next page
		// Note: In a real app, you might want to use a flash message system
		log.Printf("Vehicle %s has %d maintenance alerts", busID, len(alerts))
	}

	// Redirect to driver dashboard with success
	http.Redirect(w, r, "/driver-dashboard?success=true", http.StatusSeeOther)
}

// checkMaintenanceDueHandler API endpoint to check maintenance status
func checkMaintenanceDueHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	vehicleID := r.URL.Query().Get("vehicle_id")
	if vehicleID == "" {
		http.Error(w, "Vehicle ID required", http.StatusBadRequest)
		return
	}

	alerts, err := checkMaintenanceDue(vehicleID)
	if err != nil {
		log.Printf("Error checking maintenance: %v", err)
		http.Error(w, "Failed to check maintenance", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(alerts)
}

// debugMaintenanceRecordsHandler helps debug maintenance records
func debugMaintenanceRecordsHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	vehicleID := r.URL.Query().Get("vehicle_id")
	if vehicleID == "" {
		http.Error(w, "Vehicle ID required", http.StatusBadRequest)
		return
	}

	// Get maintenance logs
	logs, err := getMaintenanceLogsForVehicle(vehicleID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error: %v", err), http.StatusInternalServerError)
		return
	}

	// Get vehicle info
	currentMileage, lastOilChange, lastTireService, err := getVehicleMaintenanceInfo(vehicleID)
	if err != nil {
		log.Printf("Error getting vehicle info: %v", err)
	}

	// Check maintenance alerts
	alerts, err := checkMaintenanceDue(vehicleID)
	if err != nil {
		log.Printf("Error checking alerts: %v", err)
	}

	debug := map[string]interface{}{
		"vehicle_id":        vehicleID,
		"maintenance_logs":  logs,
		"current_mileage":   currentMileage,
		"last_oil_change":   lastOilChange,
		"last_tire_service": lastTireService,
		"alerts":            alerts,
		"log_count":         len(logs),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(debug)
}

// Helper functions
func getDriverAssignments(username string) ([]RouteAssignment, error) {
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	query := `
		SELECT ra.driver, ra.bus_id, ra.route_id, r.route_name, ra.assigned_date
		FROM route_assignments ra
		JOIN routes r ON ra.route_id = r.route_id
		WHERE ra.driver = $1
		ORDER BY r.route_name
	`

	var assignments []RouteAssignment
	err := db.Select(&assignments, query, username)
	return assignments, err
}

func getStudentsByRoute(routeID string) ([]Student, error) {
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	query := `
		SELECT * FROM students 
		WHERE route_id = $1 AND active = true 
		ORDER BY position_number, pickup_time
	`

	var students []Student
	err := db.Select(&students, query, routeID)
	return students, err
}

// Helper functions for updating vehicle fields
func updateBusField(busID, fieldName, fieldValue string) error {
	if db == nil {
		return fmt.Errorf("database not initialized")
	}

	allowedFields := map[string]bool{
		"status":            true,
		"oil_status":        true,
		"tire_status":       true,
		"maintenance_notes": true,
	}

	if !allowedFields[fieldName] {
		return fmt.Errorf("field update not allowed: %s", fieldName)
	}

	query := fmt.Sprintf("UPDATE buses SET %s = $1, updated_at = CURRENT_TIMESTAMP WHERE bus_id = $2", fieldName)
	_, err := db.Exec(query, fieldValue, busID)
	return err
}

func updateVehicleField(vehicleID, fieldName, fieldValue string) error {
	if db == nil {
		return fmt.Errorf("database not initialized")
	}

	allowedFields := map[string]bool{
		"status":            true,
		"oil_status":        true,
		"tire_status":       true,
		"maintenance_notes": true,
	}

	if !allowedFields[fieldName] {
		return fmt.Errorf("field update not allowed: %s", fieldName)
	}

	query := fmt.Sprintf("UPDATE vehicles SET %s = $1, updated_at = CURRENT_TIMESTAMP WHERE vehicle_id = $2", fieldName)
	_, err := db.Exec(query, fieldValue, vehicleID)
	return err
}

// Health check endpoint
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}