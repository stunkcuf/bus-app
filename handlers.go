package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
	
	"github.com/lib/pq"
	"github.com/jmoiron/sqlx"
)

// loginHandler handles the login page and authentication
func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		data := map[string]interface{}{}
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
			SendError(w, ErrBadRequest("Failed to parse form data"))
			return
		}

		username := r.FormValue("username")
		password := r.FormValue("password")

		// Log for debugging
		log.Printf("Login attempt for username: %s from IP: %s", username, getClientIP(r))
		log.Printf("DEBUG: Form values - username='%s', password length=%d", username, len(password))

		// Skip rate limiting in development mode
		if os.Getenv("APP_ENV") != "development" {
			if !rateLimiter.Allow(getClientIP(r)) {
				log.Printf("Rate limit exceeded for IP: %s", getClientIP(r))
				http.Error(w, "Too many login attempts. Please try again later.", http.StatusTooManyRequests)
				return
			}
		}

		user, err := authenticateUser(username, password)
		if err != nil {
			log.Printf("Authentication failed for %s: %v", username, err)
			data := map[string]interface{}{
				"Error": "Invalid username or password",
			}
			renderTemplate(w, r, "login.html", data)
			return
		}

		if user.Status != "active" {
			log.Printf("User %s is not active, status: %s", username, user.Status)
			data := map[string]interface{}{
				"Error": "Your account is pending approval",
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
		SendError(w, ErrMethodNotAllowed("Only POST method allowed"))
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
			"Data": map[string]interface{}{
				"CSRFToken": generateCSRFToken(),
			},
		}
		renderTemplate(w, r, "register.html", data)
		return
	}

	if r.Method == "POST" {
		username := r.FormValue("username")
		password := r.FormValue("password")

		err := createUser(username, password, "driver", "pending")
		if err != nil {
			data := map[string]interface{}{
				"Data": map[string]interface{}{
					"Error":     "Username already exists",
					"CSRFToken": generateCSRFToken(),
				},
			}
			renderTemplate(w, r, "register.html", data)
			return
		}

		renderTemplate(w, r, "registration_success.html", map[string]interface{}{
			"Username": username,
		})
	}
}

// managerDashboardHandler shows the manager dashboard with maintenance overview
func managerDashboardHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("Manager dashboard accessed by IP: %s", getClientIP(r))
	
	user := getUserFromSession(r)
	if user == nil {
		log.Printf("No user session found, redirecting to login")
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	
	if user.Role != "manager" {
		log.Printf("User %s has role %s, not manager, redirecting", user.Username, user.Role)
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	
	log.Printf("Manager dashboard accessed by user: %s", user.Username)

	// Get various statistics
	buses, _ := dataCache.getBuses()
	vehicles, _ := dataCache.getVehicles()
	users, _ := dataCache.getUsers()
	routes, _ := dataCache.getRoutes()

	// Count vehicles needing maintenance and status
	maintenanceNeeded := 0
	activeBuses := 0
	busesMaintenanceDue := 0
	busesOutOfService := 0

	// Check buses
	for _, bus := range buses {
		if bus.Status == "active" {
			activeBuses++
		} else if bus.Status == "out_of_service" {
			busesOutOfService++
		}

		oilStatus := bus.GetOilStatus()
		tireStatus := bus.GetTireStatus()
		if oilStatus == "due_soon" || oilStatus == "overdue" ||
			tireStatus == "due_soon" || tireStatus == "overdue" {
			maintenanceNeeded++
			busesMaintenanceDue++
		}
	}

	// Check vehicles
	for _, vehicle := range vehicles {
		oilStatus := vehicle.GetOilStatus()
		tireStatus := vehicle.GetTireStatus()
		if oilStatus == "due_soon" || oilStatus == "overdue" ||
			tireStatus == "due_soon" || tireStatus == "overdue" {
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

	// Count route assignments
	assignedRoutes := 0
	if assignments, err := getRouteAssignments(); err == nil {
		assignedRoutes = len(assignments)
	}
	unassignedRoutes := len(routes) - assignedRoutes

	// Count actual drivers (users with role="driver")
	driverCount := 0
	for _, u := range users {
		if u.Role == "driver" {
			driverCount++
		}
	}

	data := map[string]interface{}{
		"User":                user,
		"CSRFToken":           getSessionCSRFToken(r),
		"TotalBuses":          len(buses),
		"TotalVehicles":       len(vehicles),
		"TotalDrivers":        driverCount,
		"TotalRoutes":         len(routes),
		"MaintenanceNeeded":   maintenanceNeeded,
		"PendingUsers":        pendingUsers,
		"ActiveBuses":         activeBuses,
		"BusesMaintenanceDue": busesMaintenanceDue,
		"BusesOutOfService":   busesOutOfService,
		"AssignedRoutes":      assignedRoutes,
		"UnassignedRoutes":    unassignedRoutes,
	}

	// Add more data for modern dashboard
	data["CurrentDate"] = time.Now().Format("Monday, January 2, 2006")
	
	// Count active drivers (status = 'active' and role = 'driver')
	activeDriverCount := 0
	for _, u := range users {
		if u.Role == "driver" && u.Status == "active" {
			activeDriverCount++
		}
	}
	data["ActiveDrivers"] = activeDriverCount
	data["TodayRoutes"] = assignedRoutes
	data["PendingAlerts"] = maintenanceNeeded + pendingUsers
	
	// Calculate percentages
	if len(buses) > 0 {
		data["ActiveBusesPercent"] = (activeBuses * 100) / len(buses)
	} else {
		data["ActiveBusesPercent"] = 0
	}
	
	// Get real recent activity
	data["RecentActivity"] = getRecentActivity()
	
	// Get maintenance alerts
	var maintenanceAlerts []MaintenanceAlert
	for _, bus := range buses {
		oilStatus := bus.GetOilStatus()
		
		if oilStatus == "overdue" {
			maintenanceAlerts = append(maintenanceAlerts, MaintenanceAlert{
				VehicleID: bus.BusID,
				Severity:  "danger",
				Message:   "Oil change overdue",
			})
		} else if oilStatus == "due_soon" {
			maintenanceAlerts = append(maintenanceAlerts, MaintenanceAlert{
				VehicleID: bus.BusID,
				Severity:  "warning",
				Message:   "Oil change due soon",
			})
		}
		
		if len(maintenanceAlerts) >= 5 {
			break // Limit to 5 alerts
		}
	}
	data["MaintenanceAlerts"] = maintenanceAlerts
	
	// Count active drivers and fix naming
	activeDrivers := 0
	for _, u := range users {
		if u.Role == "driver" && u.Status == "active" {
			activeDrivers++
		}
	}
	data["TotalDrivers"] = activeDrivers  // Changed from ActiveDrivers to TotalDrivers to match template
	
	// Get total students
	students, err := loadStudentsFromDB()
	totalStudents := 0
	if err == nil {
		totalStudents = len(students)
	}
	data["TotalStudents"] = totalStudents
	
	// Count active routes (routes with assignments)
	assignments, _ := getRouteAssignments()
	activeRoutesMap := make(map[string]bool)
	for _, assignment := range assignments {
		activeRoutesMap[assignment.RouteID] = true
	}
	data["ActiveRoutes"] = len(activeRoutesMap)
	
	// Use the regular manager dashboard template
	renderTemplate(w, r, "manager_dashboard.html", data)
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

	// Get maintenance alerts for driver's vehicles
	maintenanceAlerts, err := getMaintenanceAlertsForDriver(user.Username)
	if err != nil {
		log.Printf("Error loading maintenance alerts: %v", err)
		maintenanceAlerts = []MaintenanceAlert{} // Empty slice on error
	}

	// Get students for assigned routes (batch loading to avoid N+1 queries)
	var routeIDs []string
	for _, assignment := range assignments {
		routeIDs = append(routeIDs, assignment.RouteID)
	}
	
	studentsMap, err := getStudentsByMultipleRoutes(routeIDs)
	if err != nil {
		log.Printf("Error loading students for routes: %v", err)
		// Fallback to empty map on error
		studentsMap = make(map[string][]Student)
	}

	// Check for success message
	success := r.URL.Query().Get("success") == "true"

	data := map[string]interface{}{
		"User":              user,
		"CSRFToken":         getSessionCSRFToken(r),
		"Assignments":       assignments,
		"StudentsMap":       studentsMap,
		"MaintenanceAlerts": maintenanceAlerts,
		"Success":           success,
		"CurrentDate":       time.Now().Format("2006-01-02"),
	}

	// Changed from driver_dashboard_modern.html to driver_dashboard.html
	renderTemplate(w, r, "driver_dashboard.html", data)
}

// companyFleetHandler shows company vehicles with maintenance alerts
func companyFleetHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	log.Printf("Company fleet page requested by %s", user.Username)
	start := time.Now()

	// Load all vehicles (non-bus vehicles from vehicles table)
	vehicles, err := loadVehiclesFromDB()
	if err != nil {
		log.Printf("Error loading vehicles after %v: %v", time.Since(start), err)
		vehicles = []Vehicle{}
	} else {
		log.Printf("Loaded %d vehicles in %v", len(vehicles), time.Since(start))
	}

	// Convert Vehicle structs to ConsolidatedVehicle for template compatibility
	consolidatedVehicles := make([]ConsolidatedVehicle, len(vehicles))
	for i, v := range vehicles {
		// Safe status handling
		status := "active"
		if v.Status.Valid {
			status = v.Status.String
		}

		consolidatedVehicles[i] = ConsolidatedVehicle{
			ID:               v.VehicleID,
			VehicleID:        v.VehicleID,
			BusID:            v.VehicleID, // For backward compatibility
			VehicleType:      "vehicle",
			Status:           status,
			Model:            v.Model,
			Year:             v.Year,
			TireSize:         v.TireSize,
			License:          v.License,
			OilStatus:        v.OilStatus,
			TireStatus:       v.TireStatus,
			Description:      v.Description,
			SerialNumber:     v.SerialNumber,
			Base:             v.Base,
			ServiceInterval:  v.ServiceInterval,
			MaintenanceNotes: v.MaintenanceNotes,
			UpdatedAt:        v.UpdatedAt,
			CreatedAt:        v.CreatedAt,
		}
	}

	// Calculate vehicle statistics correctly
	activeCount := 0
	maintenanceCount := 0
	outOfServiceCount := 0

	for _, v := range consolidatedVehicles {
		switch v.Status {
		case "active":
			activeCount++
		case "maintenance":
			maintenanceCount++
		case "out_of_service", "out-of-service":
			outOfServiceCount++
		default:
			// Don't default to active - count as unknown
			log.Printf("Unknown vehicle status: %s for vehicle %s", v.Status, v.VehicleID)
		}
	}

	// Pagination
	totalVehicles := len(consolidatedVehicles)
	pagination := GetPaginationParams(r, totalVehicles, 20) // 20 vehicles per page

	// Apply pagination to vehicles
	paginatedVehicles := []ConsolidatedVehicle{}
	if len(consolidatedVehicles) > 0 {
		end := pagination.Offset + pagination.PerPage
		if end > len(consolidatedVehicles) {
			end = len(consolidatedVehicles)
		}
		if pagination.Offset < len(consolidatedVehicles) {
			paginatedVehicles = consolidatedVehicles[pagination.Offset:end]
		}
	}

	// Skip maintenance alerts for now - they're causing timeouts
	// These can be loaded asynchronously via AJAX if needed
	allAlerts := make(map[string][]MaintenanceAlert)

	log.Printf("Preparing template data: %d total vehicles, %d paginated", len(consolidatedVehicles), len(paginatedVehicles))
	
	// Prepare data for template - wrap in Data structure like template expects
	data := map[string]interface{}{
		"User":      user,
		"CSRFToken": getSessionCSRFToken(r),
		"Data": map[string]interface{}{
			"Vehicles":          paginatedVehicles,
			"CSRFToken":         getSessionCSRFToken(r),
			"Pagination":        pagination,
			"AllVehicles":       consolidatedVehicles,
			"MaintenanceAlerts": allAlerts,
			"TotalVehicles":     totalVehicles,
		},
		"ActiveCount":       activeCount,
		"MaintenanceCount":  maintenanceCount,
		"OutOfServiceCount": outOfServiceCount,
		"TotalVehicles":     totalVehicles,
		// Remove FleetVehicles - we don't need it
		"FleetVehicles":    []FleetVehicle{},
		"AllFleetVehicles": []FleetVehicle{},
	}

	log.Printf("Company Fleet: Total=%d, Active=%d, Maintenance=%d, OutOfService=%d",
		totalVehicles, activeCount, maintenanceCount, outOfServiceCount)

	// Changed from company_fleet_modern.html to company_fleet.html
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

	// Calculate statistics
	var totalCost float64
	var last30DaysCost float64
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
	
	for _, log := range logs {
		totalCost += log.Cost
		
		// Parse date for 30-day calculation
		if log.Date != "" {
			if logDate, err := time.Parse("2006-01-02", log.Date); err == nil {
				if logDate.After(thirtyDaysAgo) {
					last30DaysCost += log.Cost
				}
			}
		}
	}
	
	avgCost := float64(0)
	if len(logs) > 0 {
		avgCost = totalCost / float64(len(logs))
	}

	data := map[string]interface{}{
		"User":      user,
		"CSRFToken": getSessionCSRFToken(r),
		"Data": map[string]interface{}{
			"VehicleID":          busID,
			"IsBus":              true,
			"Vehicle":            bus,
			"MaintenanceRecords": logs,
			"MaintenanceAlerts":  alerts,
			"TotalRecords":       len(logs),
			"TotalCost":          totalCost,
			"AverageCost":        avgCost,
			"Last30DaysCost":     last30DaysCost,
		},
	}

	renderTemplate(w, r, "vehicle_maintenance.html", data)
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

	// Calculate statistics
	var totalCost float64
	var last30DaysCost float64
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
	
	for _, log := range logs {
		totalCost += log.Cost
		
		// Parse date for 30-day calculation
		if log.Date != "" {
			if logDate, err := time.Parse("2006-01-02", log.Date); err == nil {
				if logDate.After(thirtyDaysAgo) {
					last30DaysCost += log.Cost
				}
			}
		}
	}
	
	avgCost := float64(0)
	if len(logs) > 0 {
		avgCost = totalCost / float64(len(logs))
	}

	data := map[string]interface{}{
		"User":      user,
		"CSRFToken": getSessionCSRFToken(r),
		"Data": map[string]interface{}{
			"VehicleID":          vehicleID,
			"IsBus":              false,
			"Vehicle":            vehicle,
			"MaintenanceRecords": logs,
			"MaintenanceAlerts":  alerts,
			"TotalRecords":       len(logs),
			"TotalCost":          totalCost,
			"AverageCost":        avgCost,
			"Last30DaysCost":     last30DaysCost,
		},
	}

	renderTemplate(w, r, "vehicle_maintenance.html", data)
}

// saveMaintenanceRecordHandler handles saving maintenance records
func saveMaintenanceRecordHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		SendError(w, ErrMethodNotAllowed("Only POST method allowed"))
		return
	}

	user := getUserFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	if !validateCSRF(r) {
		SendError(w, ErrForbidden("Invalid CSRF token"))
		return
	}

	// Parse form
	err := r.ParseForm()
	if err != nil {
		SendError(w, ErrBadRequest("Failed to parse form data"))
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
		SendError(w, ErrMethodNotAllowed("Only POST method allowed"))
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
		SendError(w, ErrMethodNotAllowed("Only POST method allowed"))
		return
	}

	user := getUserFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	if !validateCSRF(r) {
		SendError(w, ErrForbidden("Invalid CSRF token"))
		return
	}

	// Parse form
	err := r.ParseForm()
	if err != nil {
		SendError(w, ErrBadRequest("Failed to parse form data"))
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

	// Save to database using transaction
	err = withTransaction(func(tx *sqlx.Tx) error {
		// Save driver log
		query := `
			INSERT INTO driver_logs (driver, bus_id, route_id, date, period, departure_time, arrival_time, begin_mileage, end_mileage, attendance)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		`

		_, err := tx.Exec(query, driverLog.Driver, driverLog.BusID, driverLog.RouteID,
			driverLog.Date, driverLog.Period, driverLog.Departure, driverLog.Arrival,
			driverLog.BeginMileage, driverLog.EndMileage, driverLog.Attendance)

		if err != nil {
			return fmt.Errorf("failed to save driver log: %w", err)
		}

		// Update vehicle mileage
		if err := updateVehicleMileageInTx(tx, busID, int(endMileage)); err != nil {
			return fmt.Errorf("failed to update vehicle mileage: %w", err)
		}

		// Update maintenance status based on mileage
		if err := updateMaintenanceStatusBasedOnMileageInTx(tx, busID); err != nil {
			return fmt.Errorf("failed to update maintenance status: %w", err)
		}

		return nil
	})

	if err != nil {
		log.Printf("Error saving driver log transaction: %v", err)
		http.Error(w, "Failed to save log", http.StatusInternalServerError)
		return
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

func getRouteAssignments() ([]RouteAssignment, error) {
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	query := `
		SELECT ra.driver, ra.bus_id, ra.route_id, r.route_name, ra.assigned_date
		FROM route_assignments ra
		JOIN routes r ON ra.route_id = r.route_id
		ORDER BY r.route_name
	`

	var assignments []RouteAssignment
	err := db.Select(&assignments, query)
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

// getStudentsByMultipleRoutes gets all active students for multiple routes in a single query
func getStudentsByMultipleRoutes(routeIDs []string) (map[string][]Student, error) {
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	
	if len(routeIDs) == 0 {
		return make(map[string][]Student), nil
	}

	// Build query with proper parameter placeholders
	query := `
		SELECT * FROM students 
		WHERE route_id = ANY($1) AND active = true 
		ORDER BY route_id, position_number, pickup_time
	`

	var students []Student
	err := db.Select(&students, query, pq.StringArray(routeIDs))
	if err != nil {
		return nil, err
	}

	// Group students by route ID
	studentsMap := make(map[string][]Student)
	for _, student := range students {
		studentsMap[student.RouteID] = append(studentsMap[student.RouteID], student)
	}

	return studentsMap, nil
}

// getStudentsByMultipleRoutesIncludingInactive gets all students for multiple routes in a single query (including inactive)
func getStudentsByMultipleRoutesIncludingInactive(routeIDs []string) (map[string][]Student, error) {
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	
	if len(routeIDs) == 0 {
		return make(map[string][]Student), nil
	}

	// Build query with proper parameter placeholders
	query := `
		SELECT * FROM students 
		WHERE route_id = ANY($1)
		ORDER BY route_id, active DESC, position_number, pickup_time
	`

	var students []Student
	err := db.Select(&students, query, pq.StringArray(routeIDs))
	if err != nil {
		return nil, err
	}

	// Group students by route ID
	studentsMap := make(map[string][]Student)
	for _, student := range students {
		studentsMap[student.RouteID] = append(studentsMap[student.RouteID], student)
	}

	return studentsMap, nil
}

// getStudentsByRouteIncludingInactive gets all students for a route including inactive ones
func getStudentsByRouteIncludingInactive(routeID string) ([]Student, error) {
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	query := `
		SELECT * FROM students 
		WHERE route_id = $1
		ORDER BY active DESC, position_number, pickup_time
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

	// Use parameterized queries based on field name
	var query string
	switch fieldName {
	case "status":
		query = "UPDATE fleet_vehicles SET status = $1, updated_at = CURRENT_TIMESTAMP WHERE vehicle_id = $2 AND vehicle_type = 'bus'"
	case "oil_status":
		query = "UPDATE fleet_vehicles SET oil_status = $1, updated_at = CURRENT_TIMESTAMP WHERE vehicle_id = $2 AND vehicle_type = 'bus'"
	case "tire_status":
		query = "UPDATE fleet_vehicles SET tire_status = $1, updated_at = CURRENT_TIMESTAMP WHERE vehicle_id = $2 AND vehicle_type = 'bus'"
	case "maintenance_notes":
		query = "UPDATE fleet_vehicles SET maintenance_notes = $1, updated_at = CURRENT_TIMESTAMP WHERE vehicle_id = $2 AND vehicle_type = 'bus'"
	default:
		return fmt.Errorf("invalid field name: %s", fieldName)
	}
	
	_, err := db.Exec(query, fieldValue, busID)
	if err != nil {
		// Fallback to old buses table
		var oldQuery string
		switch fieldName {
		case "status":
			oldQuery = "UPDATE buses SET status = $1, updated_at = CURRENT_TIMESTAMP WHERE bus_id = $2"
		case "oil_status":
			oldQuery = "UPDATE buses SET oil_status = $1, updated_at = CURRENT_TIMESTAMP WHERE bus_id = $2"
		case "tire_status":
			oldQuery = "UPDATE buses SET tire_status = $1, updated_at = CURRENT_TIMESTAMP WHERE bus_id = $2"
		case "maintenance_notes":
			oldQuery = "UPDATE buses SET maintenance_notes = $1, updated_at = CURRENT_TIMESTAMP WHERE bus_id = $2"
		}
		_, oldErr := db.Exec(oldQuery, fieldValue, busID)
		return oldErr
	}
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

	// Use parameterized queries based on field name
	var query string
	switch fieldName {
	case "status":
		query = "UPDATE fleet_vehicles SET status = $1, updated_at = CURRENT_TIMESTAMP WHERE vehicle_id = $2 AND vehicle_type = 'vehicle'"
	case "oil_status":
		query = "UPDATE fleet_vehicles SET oil_status = $1, updated_at = CURRENT_TIMESTAMP WHERE vehicle_id = $2 AND vehicle_type = 'vehicle'"
	case "tire_status":
		query = "UPDATE fleet_vehicles SET tire_status = $1, updated_at = CURRENT_TIMESTAMP WHERE vehicle_id = $2 AND vehicle_type = 'vehicle'"
	case "maintenance_notes":
		query = "UPDATE fleet_vehicles SET maintenance_notes = $1, updated_at = CURRENT_TIMESTAMP WHERE vehicle_id = $2 AND vehicle_type = 'vehicle'"
	default:
		return fmt.Errorf("invalid field name: %s", fieldName)
	}
	
	_, err := db.Exec(query, fieldValue, vehicleID)
	if err != nil {
		// Fallback to old vehicles table
		var oldQuery string
		switch fieldName {
		case "status":
			oldQuery = "UPDATE vehicles SET status = $1, updated_at = CURRENT_TIMESTAMP WHERE vehicle_id = $2"
		case "oil_status":
			oldQuery = "UPDATE vehicles SET oil_status = $1, updated_at = CURRENT_TIMESTAMP WHERE vehicle_id = $2"
		case "tire_status":
			oldQuery = "UPDATE vehicles SET tire_status = $1, updated_at = CURRENT_TIMESTAMP WHERE vehicle_id = $2"
		case "maintenance_notes":
			oldQuery = "UPDATE vehicles SET maintenance_notes = $1, updated_at = CURRENT_TIMESTAMP WHERE vehicle_id = $2"
		}
		_, oldErr := db.Exec(oldQuery, fieldValue, vehicleID)
		return oldErr
	}
	return err
}

// monthlyMileageReportsHandler displays monthly mileage reports
func monthlyMileageReportsHandler(w http.ResponseWriter, r *http.Request) {
	// Check authentication
	user := getUserFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	// Parse query parameters for filtering
	yearStr := r.URL.Query().Get("year")
	month := r.URL.Query().Get("month")
	busID := r.URL.Query().Get("bus_id")

	// Parse pagination parameters
	pageStr := r.URL.Query().Get("page")
	perPageStr := r.URL.Query().Get("per_page")

	page := 1
	perPage := 50

	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	if perPageStr != "" {
		if pp, err := strconv.Atoi(perPageStr); err == nil && pp > 0 && pp <= 200 {
			perPage = pp
		}
	}

	var reports []MonthlyMileageReport
	var err error
	var year int

	if yearStr != "" {
		if y, parseErr := strconv.Atoi(yearStr); parseErr == nil {
			year = y
		}
	}

	// Load reports (filtered or all)
	if year > 0 || month != "" || busID != "" {
		reports, err = loadMonthlyMileageReportsByFilters(year, month, busID)
	} else {
		reports, err = loadMonthlyMileageReportsFromDB()
	}

	if err != nil {
		LogError("Failed to load monthly mileage reports", err)
		SendError(w, ErrDatabase("Failed to load reports", err))
		return
	}

	// Calculate pagination
	totalReports := len(reports)
	totalPages := (totalReports + perPage - 1) / perPage

	pagination := struct {
		Page       int
		PerPage    int
		TotalPages int
		HasPrev    bool
		HasNext    bool
		Pages      []int
	}{
		Page:       page,
		PerPage:    perPage,
		TotalPages: totalPages,
		HasPrev:    page > 1,
		HasNext:    page < totalPages,
		Pages:      make([]int, 0),
	}

	// Apply pagination
	start := (page - 1) * perPage
	end := start + perPage
	if end > totalReports {
		end = totalReports
	}

	paginatedReports := []MonthlyMileageReport{}
	if start < totalReports {
		paginatedReports = reports[start:end]
	}

	// Get unique years for filter dropdown
	yearSet := make(map[int]bool)
	for _, report := range reports {
		yearSet[report.ReportYear] = true
	}

	var years []int
	for y := range yearSet {
		years = append(years, y)
	}

	// Sort years descending
	for i := 0; i < len(years); i++ {
		for j := i + 1; j < len(years); j++ {
			if years[i] < years[j] {
				years[i], years[j] = years[j], years[i]
			}
		}
	}

	// Get unique bus IDs for filter dropdown
	busIDSet := make(map[string]bool)
	for _, report := range reports {
		if report.BusID != "" {
			busIDSet[report.BusID] = true
		}
	}

	var busIDs []string
	for id := range busIDSet {
		busIDs = append(busIDs, id)
	}

	// Calculate summary statistics
	totalMiles := 0
	activeVehicles := make(map[string]bool)

	for _, report := range reports {
		totalMiles += report.TotalMiles
		if report.BusID != "" {
			activeVehicles[report.BusID] = true
		}
	}

	data := map[string]interface{}{
		"User":         user,
		"CSRFToken":    getSessionCSRFToken(r),
		"Reports":      paginatedReports,
		"Pagination":   pagination,
		"TotalReports": totalReports,
		"Years":        years,
		"Months": []string{
			"January", "February", "March", "April", "May", "June",
			"July", "August", "September", "October", "November", "December",
		},
		"BusIDs":         busIDs,
		"SelectedYear":   yearStr,
		"SelectedMonth":  month,
		"SelectedBusID":  busID,
		"TotalMiles":     totalMiles,
		"ActiveVehicles": len(activeVehicles),
		"Title":          "Monthly Mileage Reports",
	}

	renderTemplate(w, r, "monthly_mileage_reports.html", data)
}

// fleetVehiclesHandler displays fleet vehicles from the fleet_vehicles table
func fleetVehiclesHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	// Get filter parameters
	yearStr := r.URL.Query().Get("year")
	makeFilter := r.URL.Query().Get("make")
	location := r.URL.Query().Get("location")

	// Parse year
	var year int
	if yearStr != "" {
		if y, err := strconv.Atoi(yearStr); err == nil {
			year = y
		}
	}

	// Get pagination parameters
	pageStr := r.URL.Query().Get("page")
	page := 1
	if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
		page = p
	}

	// Load vehicles based on filters
	var vehicles []FleetVehicle
	var err error

	if year > 0 || makeFilter != "" || location != "" {
		vehicles, err = loadFleetVehiclesByFilters(year, makeFilter, location)
	} else {
		vehicles, err = loadFleetVehiclesFromDB()
	}

	if err != nil {
		log.Printf("Error loading fleet vehicles: %v", err)
		http.Error(w, "Failed to load fleet vehicles", http.StatusInternalServerError)
		return
	}

	// Pagination setup
	perPage := 20
	totalVehicles := len(vehicles)
	totalPages := (totalVehicles + perPage - 1) / perPage

	// Calculate pagination
	start := (page - 1) * perPage
	end := start + perPage
	if end > totalVehicles {
		end = totalVehicles
	}

	var paginatedVehicles []FleetVehicle
	if start < totalVehicles {
		paginatedVehicles = vehicles[start:end]
	}

	// Create pagination object
	pagination := struct {
		Page       int
		PerPage    int
		TotalPages int
		HasPrev    bool
		HasNext    bool
		Pages      []int
	}{
		Page:       page,
		PerPage:    perPage,
		TotalPages: totalPages,
		HasPrev:    page > 1,
		HasNext:    page < totalPages,
		Pages:      make([]int, 0),
	}

	// Generate page numbers for pagination
	start_page := max(1, page-2)
	end_page := min(totalPages, page+2)
	for i := start_page; i <= end_page; i++ {
		pagination.Pages = append(pagination.Pages, i)
	}

	// Get unique years for filter dropdown
	yearSet := make(map[int]bool)
	for _, vehicle := range vehicles {
		if vehicle.GetYear() > 0 {
			yearSet[vehicle.GetYear()] = true
		}
	}

	var years []int
	for y := range yearSet {
		years = append(years, y)
	}

	// Sort years descending
	for i := 0; i < len(years); i++ {
		for j := i + 1; j < len(years); j++ {
			if years[i] < years[j] {
				years[i], years[j] = years[j], years[i]
			}
		}
	}

	// Get unique makes for filter dropdown
	makeSet := make(map[string]bool)
	for _, vehicle := range vehicles {
		if vehicle.GetMake() != "" {
			makeSet[vehicle.GetMake()] = true
		}
	}

	var makes []string
	for m := range makeSet {
		makes = append(makes, m)
	}

	// Get unique locations for filter dropdown
	locationSet := make(map[string]bool)
	for _, vehicle := range vehicles {
		if vehicle.GetLocation() != "" {
			locationSet[vehicle.GetLocation()] = true
		}
	}

	var locations []string
	for l := range locationSet {
		locations = append(locations, l)
	}

	// Calculate summary statistics
	totalWithYear := 0
	totalWithSerial := 0
	totalWithLicense := 0

	for _, vehicle := range vehicles {
		if vehicle.GetYear() > 0 {
			totalWithYear++
		}
		if vehicle.GetSerialNumber() != "" {
			totalWithSerial++
		}
		if vehicle.GetLicense() != "" {
			totalWithLicense++
		}
	}

	data := map[string]interface{}{
		"User":             user,
		"CSRFToken":        getSessionCSRFToken(r),
		"Vehicles":         paginatedVehicles,
		"Pagination":       pagination,
		"TotalVehicles":    totalVehicles,
		"Years":            years,
		"Makes":            makes,
		"Locations":        locations,
		"SelectedYear":     yearStr,
		"SelectedMake":     makeFilter,
		"SelectedLocation": location,
		"TotalWithYear":    totalWithYear,
		"TotalWithSerial":  totalWithSerial,
		"TotalWithLicense": totalWithLicense,
		"Title":            "Fleet Vehicles",
	}

	renderTemplate(w, r, "fleet_vehicles.html", data)
}

// maintenanceRecordsHandler displays maintenance records from the maintenance_records table
func maintenanceRecordsHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	// Get filter parameters
	vehicleNumberStr := r.URL.Query().Get("vehicle_number")
	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")

	// Parse vehicle number
	var vehicleNumber int
	if vehicleNumberStr != "" {
		if vn, err := strconv.Atoi(vehicleNumberStr); err == nil {
			vehicleNumber = vn
		}
	}

	// Get pagination parameters
	pageStr := r.URL.Query().Get("page")
	page := 1
	if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
		page = p
	}

	// Load records based on filters
	var records []MaintenanceRecord
	var err error

	if vehicleNumber > 0 || startDate != "" || endDate != "" {
		records, err = loadMaintenanceRecordsByFilters(vehicleNumber, startDate, endDate)
		log.Printf("DEBUG: Loading maintenance records with filters - vehicle: %d, start: %s, end: %s", vehicleNumber, startDate, endDate)
	} else {
		records, err = loadMaintenanceRecordsFromDB()
		log.Printf("DEBUG: Loading all maintenance records")
	}

	if err != nil {
		log.Printf("Error loading maintenance records: %v", err)
		http.Error(w, "Failed to load maintenance records", http.StatusInternalServerError)
		return
	}
	
	log.Printf("DEBUG: Loaded %d maintenance records", len(records))

	// Pagination setup
	perPage := 25
	totalRecords := len(records)
	totalPages := (totalRecords + perPage - 1) / perPage

	// Calculate pagination
	start := (page - 1) * perPage
	end := start + perPage
	if end > totalRecords {
		end = totalRecords
	}

	var paginatedRecords []MaintenanceRecord
	if start < totalRecords {
		paginatedRecords = records[start:end]
	}
	log.Printf("DEBUG: Pagination - page %d, showing records %d-%d of %d total", page, start+1, end, totalRecords)
	log.Printf("DEBUG: Paginated records count: %d", len(paginatedRecords))

	// Create pagination object
	pagination := struct {
		Page       int
		PerPage    int
		TotalPages int
		HasPrev    bool
		HasNext    bool
		Pages      []int
	}{
		Page:       page,
		PerPage:    perPage,
		TotalPages: totalPages,
		HasPrev:    page > 1,
		HasNext:    page < totalPages,
		Pages:      make([]int, 0),
	}

	// Generate page numbers for pagination
	start_page := max(1, page-2)
	end_page := min(totalPages, page+2)
	for i := start_page; i <= end_page; i++ {
		pagination.Pages = append(pagination.Pages, i)
	}

	// Get unique vehicle numbers for filter dropdown
	vehicleNumberSet := make(map[int]bool)
	for _, record := range records {
		if record.GetVehicleNumber() > 0 {
			vehicleNumberSet[record.GetVehicleNumber()] = true
		}
	}

	var vehicleNumbers []int
	for vn := range vehicleNumberSet {
		vehicleNumbers = append(vehicleNumbers, vn)
	}

	// Sort vehicle numbers
	for i := 0; i < len(vehicleNumbers); i++ {
		for j := i + 1; j < len(vehicleNumbers); j++ {
			if vehicleNumbers[i] > vehicleNumbers[j] {
				vehicleNumbers[i], vehicleNumbers[j] = vehicleNumbers[j], vehicleNumbers[i]
			}
		}
	}

	// Calculate summary statistics
	totalCost := 0.0
	recordsWithCost := 0
	recordsWithMileage := 0
	uniqueVehicles := make(map[int]bool)

	for _, record := range records {
		cost := record.GetCostAsFloat()
		if cost > 0 {
			totalCost += cost
			recordsWithCost++
		}
		if record.GetMileage() > 0 {
			recordsWithMileage++
		}
		if record.GetVehicleNumber() > 0 {
			uniqueVehicles[record.GetVehicleNumber()] = true
		}
	}

	data := map[string]interface{}{
		"User":                  user,
		"CSRFToken":             getSessionCSRFToken(r),
		"Records":               paginatedRecords,
		"Pagination":            pagination,
		"TotalRecords":          totalRecords,
		"VehicleNumbers":        vehicleNumbers,
		"SelectedVehicleNumber": vehicleNumberStr,
		"SelectedStartDate":     startDate,
		"SelectedEndDate":       endDate,
		"TotalCost":             totalCost,
		"RecordsWithCost":       recordsWithCost,
		"RecordsWithMileage":    recordsWithMileage,
		"UniqueVehicles":        len(uniqueVehicles),
		"Title":                 "Maintenance Records",
	}

	renderTemplate(w, r, "maintenance_records.html", data)
}

// serviceRecordsHandler displays service records from the service_records table
func serviceRecordsHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	// Get filter parameters
	vehicleFilter := r.URL.Query().Get("vehicle_filter")
	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")

	// Get pagination parameters
	pageStr := r.URL.Query().Get("page")
	page := 1
	if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
		page = p
	}

	// Load records based on filters
	var records []ServiceRecord
	var err error

	if vehicleFilter != "" || startDate != "" || endDate != "" {
		records, err = loadServiceRecordsByFilters(vehicleFilter, startDate, endDate)
	} else {
		records, err = loadServiceRecordsFromDB()
	}

	if err != nil {
		log.Printf("Error loading service records: %v", err)
		http.Error(w, "Failed to load service records", http.StatusInternalServerError)
		return
	}

	// Pagination setup
	perPage := 20
	totalRecords := len(records)
	totalPages := (totalRecords + perPage - 1) / perPage

	// Calculate pagination
	start := (page - 1) * perPage
	end := start + perPage
	if end > totalRecords {
		end = totalRecords
	}

	var paginatedRecords []ServiceRecord
	if start < totalRecords {
		paginatedRecords = records[start:end]
	}

	// Create pagination object
	pagination := struct {
		Page       int
		PerPage    int
		TotalPages int
		HasPrev    bool
		HasNext    bool
		Pages      []int
	}{
		Page:       page,
		PerPage:    perPage,
		TotalPages: totalPages,
		HasPrev:    page > 1,
		HasNext:    page < totalPages,
		Pages:      make([]int, 0),
	}

	// Generate page numbers for pagination
	start_page := max(1, page-2)
	end_page := min(totalPages, page+2)
	for i := start_page; i <= end_page; i++ {
		pagination.Pages = append(pagination.Pages, i)
	}

	// Calculate summary statistics
	recordsWithData := 0
	recordsWithMaintDate := 0
	uniqueVehicles := make(map[string]bool)

	for _, record := range records {
		// Count records with meaningful data (non-header)
		fields := record.GetAllFields()
		if len(fields) > 0 {
			recordsWithData++
		}

		if record.GetMaintenanceDate() != "" {
			recordsWithMaintDate++
		}

		vehicleInfo := record.GetVehicleInfo()
		if vehicleInfo != "" && vehicleInfo != fmt.Sprintf("Record #%d", record.ID) {
			uniqueVehicles[vehicleInfo] = true
		}
	}

	data := map[string]interface{}{
		"User":                  user,
		"CSRFToken":             getSessionCSRFToken(r),
		"Records":               paginatedRecords,
		"Pagination":            pagination,
		"TotalRecords":          totalRecords,
		"SelectedVehicleFilter": vehicleFilter,
		"SelectedStartDate":     startDate,
		"SelectedEndDate":       endDate,
		"RecordsWithData":       recordsWithData,
		"RecordsWithMaintDate":  recordsWithMaintDate,
		"UniqueVehicles":        len(uniqueVehicles),
		"Title":                 "Service Records",
	}

	renderTemplate(w, r, "service_records.html", data)
}
