package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/url"  // ADDED: For URL decoding
	"path/filepath"
	"strconv"  // ADDED: For string to int conversion
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
	"github.com/xuri/excelize/v2"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

// ====================
// Session Management
// ====================

var (
	sessions = make(map[string]*Session)
	mu       sync.RWMutex
)

// Session represents a user session
type Session struct {
	Username string
	Role     string
	Expires  time.Time
}

// ====================
// Data Models
// ====================

// RouteLog represents a driver's daily route log
type RouteLog struct {
	ID         int
	Driver     string
	Date       string
	Period     string
	RouteID    string
	BusID      string
	Mileage    float64
	Departure  string
	Arrival    string
	Attendance []StudentAttendance
}

// StudentAttendance represents student attendance on a route
type StudentAttendance struct {
	Position   int
	Present    bool
	PickupTime string
}

// MaintenanceRecord represents a vehicle maintenance log entry
type MaintenanceRecord struct {
	VehicleID string
	Date      string
	Category  string
	Mileage   int
	Cost      float64
	Notes     string
	CreatedAt time.Time
}

// MileageReport represents a monthly mileage report for a vehicle
type MileageReport struct {
	ReportMonth    string
	ReportYear     int
	VehicleYear    int
	MakeModel      string
	LicensePlate   string
	VehicleID      string
	Location       string
	BeginningMiles int
	EndingMiles    int
	TotalMiles     int
	Status         string
}

// ====================
// Handlers
// ====================

// loginHandler handles user login
func loginHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		data := map[string]interface{}{
			"CSRFToken": generateCSRFToken(),
		}
		executeTemplate(w, "login.html", data)
	case "POST":
		r.ParseForm()
		username := r.FormValue("username")
		password := r.FormValue("password")

		user, err := authenticateUser(username, password)
		if err != nil {
			data := map[string]interface{}{
				"Error":     "Invalid username or password",
				"CSRFToken": generateCSRFToken(),
			}
			executeTemplate(w, "login.html", data)
			return
		}

		// Create session
		sessionID := generateSessionID()
		mu.Lock()
		sessions[sessionID] = &Session{
			Username: user.Username,
			Role:     user.Role,
			Expires:  time.Now().Add(24 * time.Hour),
		}
		mu.Unlock()

		log.Printf("Created session for user %s with role %s", user.Username, user.Role)

		// Set cookie
		http.SetCookie(w, &http.Cookie{
			Name:     "session",
			Value:    sessionID,
			Expires:  time.Now().Add(24 * time.Hour),
			HttpOnly: true,
			SameSite: http.SameSiteStrictMode,
		})

		// Redirect based on role
		if user.Role == "manager" {
			http.Redirect(w, r, "/manager-dashboard", http.StatusSeeOther)
		} else {
			http.Redirect(w, r, "/driver-dashboard", http.StatusSeeOther)
		}
	}
}

// dashboardHandler displays the appropriate dashboard based on user role
func dashboardHandler(w http.ResponseWriter, r *http.Request) {
	if !isLoggedIn(r) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	user := getUser(r)

	if user.Role == "manager" {
		users := loadUsersFromCache()
		buses := loadBusesFromCache()
		routes := loadRoutesFromCache()

		// Count pending users
		pendingCount := 0
		for _, u := range users {
			if u.Status == "pending" {
				pendingCount++
			}
		}

		data := map[string]interface{}{
			"User":         user,
			"Role":         user.Role,
			"Users":        users,
			"Buses":        buses,
			"Routes":       routes,
			"PendingUsers": pendingCount,
			"CSRFToken":    generateCSRFToken(),
		}
		executeTemplate(w, "dashboard.html", data)
	} else {
		// Driver dashboard
		data := map[string]interface{}{
			"User":      user,
			"Role":      user.Role,
			"CSRFToken": generateCSRFToken(),
		}
		executeTemplate(w, "dashboard.html", data)
	}
}

// driverDashboardHandler handles the driver's route logging page
func driverDashboardHandler(w http.ResponseWriter, r *http.Request) {
	if !isLoggedIn(r) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	user := getUser(r)
	if user.Role != "driver" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
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
	var assignment DriverAssignment
	var route Route
	var bus Bus
	hasAssignment := false

	err := db.QueryRow(`
		SELECT da.driver, da.route_id, da.bus_id, r.route_name, r.description
		FROM driver_assignments da
		JOIN routes r ON da.route_id = r.route_id
		WHERE da.driver = $1
	`, user.Username).Scan(&assignment.Driver, &assignment.RouteID, 
		&assignment.BusID, &route.RouteName, &route.Description)

	if err == nil {
		hasAssignment = true
		route.RouteID = assignment.RouteID
		
		// Get bus details
		err = db.QueryRow(`
			SELECT bus_id, model, capacity, status
			FROM buses
			WHERE bus_id = $1
		`, assignment.BusID).Scan(&bus.BusID, &bus.Model, &bus.Capacity, &bus.Status)
		
		if err != nil {
			log.Printf("Error loading bus details: %v", err)
		}
	}

	// Get students on this route
	var students []Student
	if hasAssignment {
		rows, err := db.Query(`
			SELECT s.student_id, s.name, s.phone_number, s.alt_phone_number, 
				   s.guardian, s.route_id, s.position_number, s.active,
				   s.pickup_time, s.dropoff_time
			FROM students s
			WHERE s.route_id = $1 AND s.active = true
			ORDER BY 
				CASE WHEN $2 = 'morning' THEN s.pickup_time ELSE s.dropoff_time END
		`, route.RouteID, period)
		
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var s Student
				var pickupTime, dropoffTime sql.NullString
				err := rows.Scan(&s.StudentID, &s.Name, &s.PhoneNumber, &s.AltPhoneNumber,
					&s.Guardian, &s.RouteID, &s.PositionNumber, &s.Active,
					&pickupTime, &dropoffTime)
				if err != nil {
					log.Printf("Error scanning student: %v", err)
					continue
				}
				s.PickupTime = pickupTime.String
				s.DropoffTime = dropoffTime.String
				students = append(students, s)
			}
		}
	}

	// Get existing log for this date/period if any
	var existingLog RouteLog
	err = db.QueryRow(`
		SELECT id, driver, date, period, route_id, bus_id, mileage, departure, arrival
		FROM driver_logs
		WHERE driver = $1 AND date = $2 AND period = $3
	`, user.Username, date, period).Scan(&existingLog.ID, &existingLog.Driver,
		&existingLog.Date, &existingLog.Period, &existingLog.RouteID,
		&existingLog.BusID, &existingLog.Mileage, &existingLog.Departure,
		&existingLog.Arrival)

	if err == nil {
		// Load attendance for existing log
		rows, err := db.Query(`
			SELECT position, present, pickup_time
			FROM student_attendance
			WHERE log_id = $1
		`, existingLog.ID)
		
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var att StudentAttendance
				err := rows.Scan(&att.Position, &att.Present, &att.PickupTime)
				if err == nil {
					existingLog.Attendance = append(existingLog.Attendance, att)
				}
			}
		}
	}

	// Get recent logs
	var recentLogs []RouteLog
	rows, err := db.Query(`
		SELECT id, date, period, mileage, departure, arrival
		FROM driver_logs
		WHERE driver = $1
		ORDER BY date DESC, period DESC
		LIMIT 5
	`, user.Username)
	
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var log RouteLog
			err := rows.Scan(&log.ID, &log.Date, &log.Period, 
				&log.Mileage, &log.Departure, &log.Arrival)
			if err == nil {
				// Count attendance
				var count int
				db.QueryRow(`
					SELECT COUNT(*) FROM student_attendance 
					WHERE log_id = $1 AND present = true
				`, log.ID).Scan(&count)
				
				log.Attendance = make([]StudentAttendance, count)
				recentLogs = append(recentLogs, log)
			}
		}
	}

	data := map[string]interface{}{
		"User":       user,
		"Date":       date,
		"Period":     period,
		"Route":      route,
		"Bus":        bus,
		"Students":   students,
		"DriverLog":  existingLog,
		"RecentLogs": recentLogs,
		"CSRFToken":  generateCSRFToken(),
	}

	executeTemplate(w, "driver_dashboard.html", data)
}

// saveLogHandler saves driver's route log
func saveLogHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !isLoggedIn(r) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	user := getUser(r)
	if user.Role != "driver" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	r.ParseForm()
	
	date := r.FormValue("date")
	period := r.FormValue("period")
	routeID := r.FormValue("route_id")
	busID := r.FormValue("bus_id")
	mileage, _ := strconv.ParseFloat(r.FormValue("mileage"), 64)
	departure := r.FormValue("departure")
	arrival := r.FormValue("arrival")

	// Start transaction
	tx, err := db.Begin()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	// Check if log already exists
	var logID int
	err = tx.QueryRow(`
		SELECT id FROM driver_logs 
		WHERE driver = $1 AND date = $2 AND period = $3
	`, user.Username, date, period).Scan(&logID)

	if err == sql.ErrNoRows {
		// Insert new log
		err = tx.QueryRow(`
			INSERT INTO driver_logs (driver, date, period, route_id, bus_id, mileage, departure, arrival)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			RETURNING id
		`, user.Username, date, period, routeID, busID, mileage, departure, arrival).Scan(&logID)
	} else {
		// Update existing log
		_, err = tx.Exec(`
			UPDATE driver_logs 
			SET route_id = $1, bus_id = $2, mileage = $3, departure = $4, arrival = $5, updated_at = CURRENT_TIMESTAMP
			WHERE id = $6
		`, routeID, busID, mileage, departure, arrival, logID)
	}

	if err != nil {
		log.Printf("Error saving driver log: %v", err)
		http.Error(w, "Error saving log", http.StatusInternalServerError)
		return
	}

	// Delete existing attendance records
	_, err = tx.Exec(`DELETE FROM student_attendance WHERE log_id = $1`, logID)
	if err != nil {
		log.Printf("Error deleting old attendance: %v", err)
	}

	// Save attendance records
	for key, values := range r.Form {
		if strings.HasPrefix(key, "present_") {
			positionStr := strings.TrimPrefix(key, "present_")
			position, _ := strconv.Atoi(positionStr)
			pickupTime := r.FormValue("pickup_" + positionStr)
			
			_, err = tx.Exec(`
				INSERT INTO student_attendance (log_id, position, present, pickup_time)
				VALUES ($1, $2, $3, $4)
			`, logID, position, true, pickupTime)
			
			if err != nil {
				log.Printf("Error saving attendance: %v", err)
			}
		}
	}

	// Commit transaction
	err = tx.Commit()
	if err != nil {
		http.Error(w, "Error saving data", http.StatusInternalServerError)
		return
	}

	// Redirect back to driver dashboard
	http.Redirect(w, r, "/driver-dashboard?date="+date+"&period="+period, http.StatusSeeOther)
}

// fleetHandler manages bus fleet
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

	buses := loadBusesFromCache()
	
	// Get recent maintenance logs
	var maintenanceLogs []MaintenanceRecord
	rows, err := db.Query(`
		SELECT vehicle_id, date, category, mileage, cost, notes, created_at
		FROM maintenance_logs
		WHERE vehicle_id LIKE 'BUS%'
		ORDER BY created_at DESC
		LIMIT 10
	`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var log MaintenanceRecord
			err := rows.Scan(&log.VehicleID, &log.Date, &log.Category, 
				&log.Mileage, &log.Cost, &log.Notes, &log.CreatedAt)
			if err == nil {
				maintenanceLogs = append(maintenanceLogs, log)
			}
		}
	}

	data := map[string]interface{}{
		"Buses":           buses,
		"MaintenanceLogs": maintenanceLogs,
		"Today":           time.Now().Format("2006-01-02"),
		"CSRFToken":       generateCSRFToken(),
	}
	executeTemplate(w, "fleet.html", data)
}

// companyFleetHandler shows all company vehicles (not just buses)
func companyFleetHandler(w http.ResponseWriter, r *http.Request) {
	if !isLoggedIn(r) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	user := getUser(r)
	if user.Role != "manager" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Load all vehicles from cache
	vehicles := loadVehiclesFromCache()

	data := map[string]interface{}{
		"Vehicles":  vehicles,
		"CSRFToken": generateCSRFToken(),
	}
	executeTemplate(w, "company_fleet.html", data)
}

// loadVehiclesFromCache loads all vehicles with caching
func loadVehiclesFromCache() []Vehicle {
	cacheKey := "all_vehicles"
	if cached, found := cache.Get(cacheKey); found {
		return cached.([]Vehicle)
	}

	log.Println("Cache miss: loading vehicles from database")
	
	rows, err := db.Query(`
		SELECT vehicle_id, bus_id, description, year, model, tire_size, 
			   license, maintenance_notes, oil_status, tire_status, status, created_at
		FROM vehicles
		ORDER BY vehicle_id
	`)
	if err != nil {
		log.Printf("Error loading vehicles: %v", err)
		return []Vehicle{}
	}
	defer rows.Close()

	var vehicles []Vehicle
	for rows.Next() {
		var v Vehicle
		var yearStr sql.NullString  // CHANGED: Use NullString to handle invalid year values
		var model, tireSize, license, maintenanceNotes, oilStatus, tireStatus, status sql.NullString
		
		err := rows.Scan(&v.VehicleID, &v.BusID, &v.Description, &yearStr, &model, &tireSize, 
			&license, &maintenanceNotes, &oilStatus, &tireStatus, &status, &v.CreatedAt)
		if err != nil {
			log.Printf("Error scanning vehicle: %v", err)
			continue
		}
		
		// CHANGED: Handle year conversion with validation
		if yearStr.Valid && yearStr.String != "" && yearStr.String != "nan" {
			if year, err := strconv.Atoi(yearStr.String); err == nil {
				v.Year = year
			}
		}
		
		v.Model = model.String
		v.TireSize = tireSize.String
		v.License = license.String
		v.MaintenanceNotes = maintenanceNotes.String
		v.OilStatus = oilStatus.String
		v.TireStatus = tireStatus.String
		v.Status = status.String
		
		// Set default values if empty
		if v.OilStatus == "" {
			v.OilStatus = "good"
		}
		if v.TireStatus == "" {
			v.TireStatus = "good"
		}
		if v.Status == "" {
			v.Status = "active"
		}
		
		vehicles = append(vehicles, v)
	}

	cache.Set(cacheKey, vehicles, 5*time.Minute)
	return vehicles
}

// updateVehicleStatusHandler updates vehicle status, oil status, or tire status
func updateVehicleStatusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !isLoggedIn(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	user := getUser(r)
	if user.Role != "manager" {
		http.Error(w, "Unauthorized - Manager access required", http.StatusUnauthorized)
		return
	}

	r.ParseForm()
	vehicleID := r.FormValue("vehicle_id")
	statusType := r.FormValue("status_type")
	newStatus := r.FormValue("new_status")

	if vehicleID == "" || statusType == "" || newStatus == "" {
		http.Error(w, "Missing required parameters", http.StatusBadRequest)
		return
	}

	var query string
	switch statusType {
	case "status":
		query = "UPDATE vehicles SET status = $1 WHERE vehicle_id = $2"
	case "oil":
		query = "UPDATE vehicles SET oil_status = $1 WHERE vehicle_id = $2"
	case "tire":
		query = "UPDATE vehicles SET tire_status = $1 WHERE vehicle_id = $2"
	default:
		http.Error(w, "Invalid status type", http.StatusBadRequest)
		return
	}

	_, err := db.Exec(query, newStatus, vehicleID)
	if err != nil {
		log.Printf("Error updating vehicle status: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	// Clear cache
	cache.Delete("all_vehicles")

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "success",
		"message": "Vehicle status updated successfully",
	})
}

// vehicleMaintenanceHandler displays maintenance records for a specific vehicle
func vehicleMaintenanceHandler(w http.ResponseWriter, r *http.Request) {
	if !isLoggedIn(r) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	user := getUser(r)
	if user.Role != "manager" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Extract vehicle ID from URL path
	path := r.URL.Path
	vehicleID := strings.TrimPrefix(path, "/vehicle-maintenance/")
	
	// CHANGED: URL decode the vehicle ID to handle spaces and special characters
	decodedID, err := url.QueryUnescape(vehicleID)
	if err == nil {
		vehicleID = decodedID // Use decoded version if successful
	}
	
	log.Printf("=== Vehicle Maintenance Handler ===")
	log.Printf("Full Path: %s", path)
	log.Printf("Extracted Vehicle ID: '%s'", vehicleID)
	
	if vehicleID == "" {
		http.Error(w, "Vehicle ID required", http.StatusBadRequest)
		return
	}

	// Get maintenance records for the vehicle
	query := `
		SELECT vehicle_id, date, category, mileage, cost, notes, created_at
		FROM maintenance_logs
		WHERE vehicle_id = $1
		ORDER BY date DESC, created_at DESC
	`

	rows, err := db.Query(query, vehicleID)
	if err != nil {
		log.Printf("Error querying maintenance logs: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var records []MaintenanceRecord
	var totalCost float64
	for rows.Next() {
		var record MaintenanceRecord
		err := rows.Scan(&record.VehicleID, &record.Date, &record.Category, 
			&record.Mileage, &record.Cost, &record.Notes, &record.CreatedAt)
		if err != nil {
			log.Printf("Error scanning maintenance record: %v", err)
			continue
		}
		records = append(records, record)
		totalCost += record.Cost
	}

	// Calculate statistics
	totalRecords := len(records)
	averageCost := 0.0
	if totalRecords > 0 {
		averageCost = totalCost / float64(totalRecords)
	}

	// Count recent records (last 30 days)
	recentCount := 0
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
	for _, record := range records {
		recordDate, _ := time.Parse("2006-01-02", record.Date)
		if recordDate.After(thirtyDaysAgo) {
			recentCount++
		}
	}

	// Determine if this is a bus based on the vehicle ID
	isBus := strings.HasPrefix(vehicleID, "BUS")

	// CHANGED: Create a simple anonymous struct that matches what the template expects
	data := struct {
		CSPNonce           string
		VehicleID          string
		IsBus              bool
		MaintenanceRecords []MaintenanceRecord
		TotalRecords       int
		TotalCost          float64
		AverageCost        float64
		RecentCount        int
		Today              string
		CSRFToken          string
	}{
		CSPNonce:           generateCSPNonce(),
		VehicleID:          vehicleID,
		IsBus:              isBus,
		MaintenanceRecords: records,
		TotalRecords:       totalRecords,
		TotalCost:          totalCost,
		AverageCost:        averageCost,
		RecentCount:        recentCount,
		Today:              time.Now().Format("2006-01-02"),
		CSRFToken:          generateCSRFToken(),
	}

	executeTemplate(w, "vehicle_maintenance.html", data)
}

// busMaintenanceHandler is similar to vehicleMaintenanceHandler but specifically for buses
func busMaintenanceHandler(w http.ResponseWriter, r *http.Request) {
	if !isLoggedIn(r) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	user := getUser(r)
	if user.Role != "manager" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Extract bus ID from URL
	busID := strings.TrimPrefix(r.URL.Path, "/bus-maintenance/")
	if busID == "" {
		http.Error(w, "Bus ID required", http.StatusBadRequest)
		return
	}

	// Redirect to vehicle maintenance with BUS prefix
	http.Redirect(w, r, "/vehicle-maintenance/BUS"+busID, http.StatusMovedPermanently)
}

// saveMaintenanceRecordHandler saves a maintenance record
func saveMaintenanceRecordHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !isLoggedIn(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	user := getUser(r)
	if user.Role != "manager" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	r.ParseForm()
	
	vehicleID := r.FormValue("vehicle_id")
	if vehicleID == "" {
		vehicleID = r.FormValue("bus_id") // Support both field names
	}
	
	date := r.FormValue("date")
	category := r.FormValue("category")
	mileage, _ := strconv.Atoi(r.FormValue("mileage"))
	cost, _ := strconv.ParseFloat(r.FormValue("cost"), 64)
	notes := r.FormValue("notes")

	// Insert maintenance record
	_, err := db.Exec(`
		INSERT INTO maintenance_logs (vehicle_id, date, category, mileage, cost, notes)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, vehicleID, date, category, mileage, cost, notes)

	if err != nil {
		log.Printf("Error saving maintenance record: %v", err)
		http.Error(w, "Error saving maintenance record", http.StatusInternalServerError)
		return
	}

	// Update vehicle status based on maintenance type
	if category == "oil_change" {
		db.Exec("UPDATE vehicles SET oil_status = 'good' WHERE vehicle_id = $1", vehicleID)
		// Also update buses table if it's a bus
		if strings.HasPrefix(vehicleID, "BUS") {
			busID := strings.TrimPrefix(vehicleID, "BUS")
			db.Exec("UPDATE buses SET oil_status = 'good' WHERE bus_id = $1", busID)
		}
	}
	
	// Clear caches
	cache.Delete("all_vehicles")
	cache.Delete("buses")

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"message": "Maintenance record saved successfully",
	})
}

// viewMileageReportsHandler displays mileage reports with enhanced functionality
func viewMileageReportsHandler(w http.ResponseWriter, r *http.Request) {
	if !isLoggedIn(r) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	user := getUser(r)
	if user.Role != "manager" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get filter parameters
	reportType := r.URL.Query().Get("type")
	filterMonth := r.URL.Query().Get("month")
	filterYear := r.URL.Query().Get("year")
	filterVehicleID := r.URL.Query().Get("vehicle_id")

	// Base query
	baseQuery := `
		SELECT report_month, report_year, vehicle_year, make_model, 
			   license_plate, vehicle_id, location, beginning_miles, 
			   ending_miles, total_miles, status
		FROM mileage_reports
		WHERE 1=1
	`

	// Build query conditions
	var conditions []string
	var args []interface{}
	argCount := 1

	if filterMonth != "" {
		conditions = append(conditions, fmt.Sprintf("report_month = $%d", argCount))
		args = append(args, filterMonth)
		argCount++
	}

	if filterYear != "" {
		conditions = append(conditions, fmt.Sprintf("report_year = $%d", argCount))
		args = append(args, filterYear)
		argCount++
	}

	if filterVehicleID != "" {
		conditions = append(conditions, fmt.Sprintf("vehicle_id ILIKE $%d", argCount))
		args = append(args, "%"+filterVehicleID+"%")
		argCount++
	}

	// Add conditions to query
	if len(conditions) > 0 {
		baseQuery += " AND " + strings.Join(conditions, " AND ")
	}

	baseQuery += " ORDER BY report_year DESC, report_month, vehicle_id"

	// Execute query
	rows, err := db.Query(baseQuery, args...)
	if err != nil {
		log.Printf("Error querying mileage reports: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// Separate reports by type
	var agencyVehicles []MileageReport
	var schoolBuses []map[string]interface{}
	var programStaff []map[string]interface{}
	
	// Statistics
	stats := struct {
		TotalVehicles          int
		ActiveVehicles         int
		TotalMiles             int
		AverageMilesPerVehicle float64
		TotalProgramStaff      int
		VehiclesForSale        int
	}{}

	vehicleMap := make(map[string]bool)

	for rows.Next() {
		var report MileageReport
		err := rows.Scan(&report.ReportMonth, &report.ReportYear, &report.VehicleYear,
			&report.MakeModel, &report.LicensePlate, &report.VehicleID,
			&report.Location, &report.BeginningMiles, &report.EndingMiles,
			&report.TotalMiles, &report.Status)
		
		if err != nil {
			log.Printf("Error scanning mileage report: %v", err)
			continue
		}

		// Track unique vehicles
		vehicleMap[report.VehicleID] = true
		
		// Update statistics
		stats.TotalMiles += report.TotalMiles
		if report.TotalMiles > 0 {
			stats.ActiveVehicles++
		}
		if report.Status == "FOR SALE" {
			stats.VehiclesForSale++
		}

		// Categorize by vehicle type
		if strings.HasPrefix(report.VehicleID, "BUS") {
			// School bus
			busData := map[string]interface{}{
				"ReportMonth":    report.ReportMonth,
				"ReportYear":     report.ReportYear,
				"BusID":          report.VehicleID,
				"BusYear":        report.VehicleYear,
				"BusMake":        report.MakeModel,
				"LicensePlate":   report.LicensePlate,
				"Location":       report.Location,
				"BeginningMiles": report.BeginningMiles,
				"EndingMiles":    report.EndingMiles,
				"TotalMiles":     report.TotalMiles,
				"Status":         report.Status,
			}
			
			if reportType == "" || reportType == "all" || reportType == "school_bus" {
				schoolBuses = append(schoolBuses, busData)
			}
		} else if strings.Contains(strings.ToUpper(report.Location), "PROGRAM") {
			// Program staff vehicle
			staffData := map[string]interface{}{
				"ReportMonth": report.ReportMonth,
				"ReportYear":  report.ReportYear,
				"ProgramType": report.Location,
				"StaffCount1": 1, // Mock data
				"StaffCount2": 1,
			}
			
			if reportType == "" || reportType == "all" || reportType == "program" {
				programStaff = append(programStaff, staffData)
			}
		} else {
			// Agency vehicle
			if reportType == "" || reportType == "all" || reportType == "agency" {
				agencyVehicles = append(agencyVehicles, report)
			}
		}
	}

	// Calculate final statistics
	stats.TotalVehicles = len(vehicleMap)
	if stats.TotalVehicles > 0 {
		stats.AverageMilesPerVehicle = float64(stats.TotalMiles) / float64(stats.TotalVehicles)
	}
	stats.TotalProgramStaff = len(programStaff)

	// Prepare template data
	data := map[string]interface{}{
		"AgencyVehicles":   agencyVehicles,
		"SchoolBuses":      schoolBuses,
		"ProgramStaff":     programStaff,
		"Stats":            stats,
		"FilterType":       reportType,
		"FilterMonth":      filterMonth,
		"FilterYear":       filterYear,
		"FilterVehicleID":  filterVehicleID,
		"CSRFToken":        generateCSRFToken(),
	}

	// Use the enhanced template
	if reportType == "all" || reportType == "" {
		// CHANGED: Fixed template name from mileage_reports.html to view_mileage_reports.html
		executeTemplate(w, "view_mileage_reports.html", data)
	} else {
		// Show filtered results
		// CHANGED: Fixed template name from mileage_reports.html to view_mileage_reports.html
		executeTemplate(w, "view_mileage_reports.html", data)
	}
}

// importMileageHandler handles Excel file uploads for mileage reports
func importMileageHandler(w http.ResponseWriter, r *http.Request) {
	if !isLoggedIn(r) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	user := getUser(r)
	if user.Role != "manager" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	switch r.Method {
	case "GET":
		data := map[string]interface{}{
			"CSRFToken": generateCSRFToken(),
		}
		executeTemplate(w, "import_mileage.html", data)
		
	case "POST":
		// Parse multipart form
		err := r.ParseMultipartForm(10 << 20) // 10 MB max
		if err != nil {
			data := map[string]interface{}{
				"Error":     "File too large. Maximum size is 10MB.",
				"CSRFToken": generateCSRFToken(),
			}
			executeTemplate(w, "import_mileage.html", data)
			return
		}

		// Get file from form
		file, header, err := r.FormFile("excel_file")
		if err != nil {
			data := map[string]interface{}{
				"Error":     "No file uploaded",
				"CSRFToken": generateCSRFToken(),
			}
			executeTemplate(w, "import_mileage.html", data)
			return
		}
		defer file.Close()

		// Check file extension
		ext := filepath.Ext(header.Filename)
		if ext != ".xlsx" && ext != ".xls" {
			data := map[string]interface{}{
				"Error":     "Only Excel files (.xlsx, .xls) are allowed",
				"CSRFToken": generateCSRFToken(),
			}
			executeTemplate(w, "import_mileage.html", data)
			return
		}

		// Save uploaded file temporarily
		tempFile, err := io.ReadAll(file)
		if err != nil {
			data := map[string]interface{}{
				"Error":     "Error reading file",
				"CSRFToken": generateCSRFToken(),
			}
			executeTemplate(w, "import_mileage.html", data)
			return
		}

		// Process Excel file
		recordsImported, err := processExcelFile(tempFile)
		if err != nil {
			data := map[string]interface{}{
				"Error":     fmt.Sprintf("Error processing file: %v", err),
				"CSRFToken": generateCSRFToken(),
			}
			executeTemplate(w, "import_mileage.html", data)
			return
		}

		data := map[string]interface{}{
			"Success":   fmt.Sprintf("Successfully imported %d records", recordsImported),
			"CSRFToken": generateCSRFToken(),
		}
		executeTemplate(w, "import_mileage.html", data)
	}
}

// processExcelFile processes the uploaded Excel file
func processExcelFile(fileData []byte) (int, error) {
	// Open Excel file from bytes
	f, err := excelize.OpenReader(strings.NewReader(string(fileData)))
	if err != nil {
		return 0, fmt.Errorf("failed to open Excel file: %v", err)
	}
	defer f.Close()

	// Get first sheet
	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return 0, fmt.Errorf("no sheets found in Excel file")
	}

	rows, err := f.GetRows(sheets[0])
	if err != nil {
		return 0, fmt.Errorf("failed to read rows: %v", err)
	}

	if len(rows) < 2 {
		return 0, fmt.Errorf("Excel file must have header row and at least one data row")
	}

	// Skip header row and process data
	recordsImported := 0
	for i := 1; i < len(rows); i++ {
		row := rows[i]
		if len(row) < 10 {
			continue // Skip incomplete rows
		}

		// Parse row data
		reportMonth := row[0]
		reportYear, _ := strconv.Atoi(row[1])
		vehicleYear, _ := strconv.Atoi(row[2])
		makeModel := row[3]
		licensePlate := row[4]
		vehicleID := row[5]
		location := row[6]
		beginningMiles, _ := strconv.Atoi(row[7])
		endingMiles, _ := strconv.Atoi(row[8])
		totalMiles, _ := strconv.Atoi(row[9])
		
		// Handle optional status column
		status := ""
		if len(row) > 10 {
			status = row[10]
		}

		// Ensure vehicle ID has proper prefix
		if vehicleID != "" && !strings.HasPrefix(vehicleID, "BUS") && !strings.HasPrefix(vehicleID, "VEH") {
			// Add BUS prefix if it looks like a bus number
			if _, err := strconv.Atoi(vehicleID); err == nil {
				vehicleID = "BUS" + vehicleID
			}
		}

		// Insert or update record
		_, err = db.Exec(`
			INSERT INTO mileage_reports 
			(report_month, report_year, vehicle_year, make_model, license_plate, 
			 vehicle_id, location, beginning_miles, ending_miles, total_miles, status)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
			ON CONFLICT (report_month, report_year, vehicle_id) DO UPDATE
			SET vehicle_year = EXCLUDED.vehicle_year,
				make_model = EXCLUDED.make_model,
				license_plate = EXCLUDED.license_plate,
				location = EXCLUDED.location,
				beginning_miles = EXCLUDED.beginning_miles,
				ending_miles = EXCLUDED.ending_miles,
				total_miles = EXCLUDED.total_miles,
				status = EXCLUDED.status,
				updated_at = CURRENT_TIMESTAMP
		`, reportMonth, reportYear, vehicleYear, makeModel, licensePlate,
			vehicleID, location, beginningMiles, endingMiles, totalMiles, status)

		if err != nil {
			log.Printf("Error importing row %d: %v", i, err)
			continue
		}

		recordsImported++
	}

	return recordsImported, nil
}

// registerHandler handles new driver registration
func registerHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		data := map[string]interface{}{
			"CSRFToken": generateCSRFToken(),
		}
		executeTemplate(w, "register.html", data)
		
	case "POST":
		r.ParseForm()
		username := r.FormValue("username")
		password := r.FormValue("password")

		// Validate username
		if len(username) < 3 || len(username) > 20 {
			data := map[string]interface{}{
				"Error":     "Username must be between 3 and 20 characters",
				"CSRFToken": generateCSRFToken(),
			}
			executeTemplate(w, "register.html", data)
			return
		}

		// Check if username already exists
		var exists bool
		err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE username = $1)", username).Scan(&exists)
		if err != nil || exists {
			data := map[string]interface{}{
				"Error":     "Username already taken",
				"CSRFToken": generateCSRFToken(),
			}
			executeTemplate(w, "register.html", data)
			return
		}

		// Hash password
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, "Error creating account", http.StatusInternalServerError)
			return
		}

		// Create user with pending status
		_, err = db.Exec(`
			INSERT INTO users (username, password, role, status)
			VALUES ($1, $2, 'driver', 'pending')
		`, username, string(hashedPassword))

		if err != nil {
			http.Error(w, "Error creating account", http.StatusInternalServerError)
			return
		}

		// Clear users cache
		cache.Delete("users")

		// Show success page
		executeTemplate(w, "registration_success.html", nil)
	}
}

// approveUsersHandler displays pending user registrations
func approveUsersHandler(w http.ResponseWriter, r *http.Request) {
	if !isLoggedIn(r) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	user := getUser(r)
	if user.Role != "manager" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get pending users
	rows, err := db.Query(`
		SELECT username, created_at 
		FROM users 
		WHERE status = 'pending' 
		ORDER BY created_at DESC
	`)
	if err != nil {
		log.Printf("Error querying pending users: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var pendingUsers []map[string]string
	for rows.Next() {
		var username string
		var createdAt time.Time
		err := rows.Scan(&username, &createdAt)
		if err != nil {
			continue
		}
		pendingUsers = append(pendingUsers, map[string]string{
			"Username":  username,
			"CreatedAt": createdAt.Format("Jan 2, 2006 3:04 PM"),
		})
	}

	data := map[string]interface{}{
		"PendingUsers": pendingUsers,
		"CSRFToken":    generateCSRFToken(),
	}
	executeTemplate(w, "approve_users.html", data)
}

// approveUserHandler handles user approval/rejection
func approveUserHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !isLoggedIn(r) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	user := getUser(r)
	if user.Role != "manager" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	r.ParseForm()
	username := r.FormValue("username")
	action := r.FormValue("action")

	if action == "approve" {
		_, err := db.Exec(`
			UPDATE users SET status = 'active' 
			WHERE username = $1 AND status = 'pending'
		`, username)
		if err != nil {
			log.Printf("Error approving user: %v", err)
		}
	} else if action == "reject" {
		_, err := db.Exec(`
			DELETE FROM users 
			WHERE username = $1 AND status = 'pending'
		`, username)
		if err != nil {
			log.Printf("Error rejecting user: %v", err)
		}
	}

	// Clear users cache
	cache.Delete("users")

	http.Redirect(w, r, "/approve-users", http.StatusSeeOther)
}

// manageUsersHandler displays user management page
func manageUsersHandler(w http.ResponseWriter, r *http.Request) {
	if !isLoggedIn(r) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	user := getUser(r)
	if user.Role != "manager" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	users := loadUsersFromCache()
	data := map[string]interface{}{
		"Users":     users,
		"CSRFToken": generateCSRFToken(),
	}
	executeTemplate(w, "manage_users.html", data)
}

// editUserHandler handles user editing
func editUserHandler(w http.ResponseWriter, r *http.Request) {
	if !isLoggedIn(r) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	user := getUser(r)
	if user.Role != "manager" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	username := r.URL.Query().Get("username")
	if username == "" {
		http.Error(w, "Username required", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case "GET":
		// Get user details
		var u User
		err := db.QueryRow(`
			SELECT username, role, status, created_at 
			FROM users WHERE username = $1
		`, username).Scan(&u.Username, &u.Role, &u.Status, &u.CreatedAt)
		
		if err != nil {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}

		data := map[string]interface{}{
			"Username":  u.Username,
			"Role":      u.Role,
			"Status":    u.Status,
			"CSRFToken": generateCSRFToken(),
		}
		executeTemplate(w, "users.html", data)
		
	case "POST":
		r.ParseForm()
		action := r.FormValue("action")
		
		if action == "update_role" {
			newRole := r.FormValue("role")
			if newRole != "driver" && newRole != "manager" {
				http.Error(w, "Invalid role", http.StatusBadRequest)
				return
			}
			
			_, err := db.Exec(`
				UPDATE users SET role = $1 
				WHERE username = $2
			`, newRole, username)
			
			if err != nil {
				log.Printf("Error updating user role: %v", err)
				http.Error(w, "Error updating user", http.StatusInternalServerError)
				return
			}
		} else if action == "reset_password" {
			newPassword := r.FormValue("password")
			if len(newPassword) < 6 {
				http.Error(w, "Password must be at least 6 characters", http.StatusBadRequest)
				return
			}
			
			hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
			if err != nil {
				http.Error(w, "Error resetting password", http.StatusInternalServerError)
				return
			}
			
			_, err = db.Exec(`
				UPDATE users SET password = $1 
				WHERE username = $2
			`, string(hashedPassword), username)
			
			if err != nil {
				log.Printf("Error resetting password: %v", err)
				http.Error(w, "Error resetting password", http.StatusInternalServerError)
				return
			}
		}
		
		// Clear users cache
		cache.Delete("users")
		
		http.Redirect(w, r, "/manager-dashboard", http.StatusSeeOther)
	}
}

// deleteUserHandler handles user deletion
func deleteUserHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !isLoggedIn(r) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	user := getUser(r)
	if user.Role != "manager" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	r.ParseForm()
	username := r.FormValue("username")
	
	// Don't allow deleting own account
	if username == user.Username {
		http.Error(w, "Cannot delete your own account", http.StatusBadRequest)
		return
	}

	_, err := db.Exec("DELETE FROM users WHERE username = $1", username)
	if err != nil {
		log.Printf("Error deleting user: %v", err)
		http.Error(w, "Error deleting user", http.StatusInternalServerError)
		return
	}

	// Clear users cache
	cache.Delete("users")

	http.Redirect(w, r, "/manage-users", http.StatusSeeOther)
}

// assignRoutesHandler manages route assignments
func assignRoutesHandler(w http.ResponseWriter, r *http.Request) {
	if !isLoggedIn(r) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	user := getUser(r)
	if user.Role != "manager" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Load data
	routes := loadRoutesFromCache()
	buses := loadBusesFromCache()
	users := loadUsersFromCache()
	
	// Get drivers only
	var drivers []User
	for _, u := range users {
		if u.Role == "driver" && u.Status == "active" {
			drivers = append(drivers, u)
		}
	}
	
	// Get current assignments
	rows, err := db.Query(`
		SELECT da.driver, da.route_id, da.bus_id, r.route_name
		FROM driver_assignments da
		JOIN routes r ON da.route_id = r.route_id
		ORDER BY da.driver
	`)
	
	var assignments []DriverAssignment
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var a DriverAssignment
			err := rows.Scan(&a.Driver, &a.RouteID, &a.BusID, &a.RouteName)
			if err == nil {
				assignments = append(assignments, a)
			}
		}
	}
	
	// Get assigned bus IDs
	assignedBuses := make(map[string]bool)
	for _, a := range assignments {
		assignedBuses[a.BusID] = true
	}
	
	// Filter available buses
	var availableBuses []Bus
	for _, b := range buses {
		if b.Status == "active" && !assignedBuses[b.BusID] {
			availableBuses = append(availableBuses, b)
		}
	}

	// Get routes with assignment status
	var routesWithStatus []map[string]interface{}
	assignedRoutes := make(map[string]bool)
	
	for _, a := range assignments {
		assignedRoutes[a.RouteID] = true
	}
	
	for _, r := range routes {
		routeData := map[string]interface{}{
			"RouteID":     r.RouteID,
			"RouteName":   r.RouteName,
			"Description": r.Description,
			"IsAssigned":  assignedRoutes[r.RouteID],
		}
		routesWithStatus = append(routesWithStatus, routeData)
	}
	
	// Get available routes (not assigned)
	var availableRoutes []Route
	for _, r := range routes {
		if !assignedRoutes[r.RouteID] {
			availableRoutes = append(availableRoutes, r)
		}
	}

	// Calculate statistics
	totalAssignments := len(assignments)
	totalRoutes := len(routes)
	availableDriversCount := len(drivers) - len(assignments)
	availableBusesCount := len(availableBuses)

	data := map[string]interface{}{
		"Drivers":              drivers,
		"Routes":               routes,
		"AvailableRoutes":      availableRoutes,
		"RoutesWithStatus":     routesWithStatus,
		"Buses":                buses,
		"AvailableBuses":       availableBuses,
		"Assignments":          assignments,
		"TotalAssignments":     totalAssignments,
		"TotalRoutes":          totalRoutes,
		"AvailableDriversCount": availableDriversCount,
		"AvailableBusesCount":  availableBusesCount,
		"CSRFToken":            generateCSRFToken(),
	}
	
	executeTemplate(w, "assign_routes.html", data)
}

// assignRouteHandler creates a new route assignment
func assignRouteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !isLoggedIn(r) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	user := getUser(r)
	if user.Role != "manager" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	r.ParseForm()
	driver := r.FormValue("driver")
	routeID := r.FormValue("route_id")
	busID := r.FormValue("bus_id")

	// Validate inputs
	if driver == "" || routeID == "" || busID == "" {
		http.Error(w, "All fields are required", http.StatusBadRequest)
		return
	}

	// Check if driver already has an assignment
	var existingCount int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM driver_assignments WHERE driver = $1
	`, driver).Scan(&existingCount)
	
	if err == nil && existingCount > 0 {
		http.Error(w, "Driver already has an assignment", http.StatusBadRequest)
		return
	}

	// Create assignment
	_, err = db.Exec(`
		INSERT INTO driver_assignments (driver, route_id, bus_id)
		VALUES ($1, $2, $3)
	`, driver, routeID, busID)

	if err != nil {
		log.Printf("Error creating assignment: %v", err)
		http.Error(w, "Error creating assignment", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/assign-routes", http.StatusSeeOther)
}

// unassignRouteHandler removes a route assignment
func unassignRouteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !isLoggedIn(r) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	user := getUser(r)
	if user.Role != "manager" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	r.ParseForm()
	driver := r.FormValue("driver")
	busID := r.FormValue("bus_id")

	_, err := db.Exec(`
		DELETE FROM driver_assignments 
		WHERE driver = $1 AND bus_id = $2
	`, driver, busID)

	if err != nil {
		log.Printf("Error removing assignment: %v", err)
		http.Error(w, "Error removing assignment", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/assign-routes", http.StatusSeeOther)
}

// addRouteHandler adds a new route
func addRouteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !isLoggedIn(r) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	user := getUser(r)
	if user.Role != "manager" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	r.ParseForm()
	routeName := r.FormValue("route_name")
	description := r.FormValue("description")

	if routeName == "" {
		http.Error(w, "Route name is required", http.StatusBadRequest)
		return
	}

	// Generate route ID
	routeID := fmt.Sprintf("RT%03d", time.Now().Unix()%1000)

	_, err := db.Exec(`
		INSERT INTO routes (route_id, route_name, description)
		VALUES ($1, $2, $3)
	`, routeID, routeName, description)

	if err != nil {
		log.Printf("Error creating route: %v", err)
		http.Error(w, "Error creating route", http.StatusInternalServerError)
		return
	}

	// Clear routes cache
	cache.Delete("routes")

	http.Redirect(w, r, "/assign-routes", http.StatusSeeOther)
}

// editRouteHandler edits an existing route
func editRouteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !isLoggedIn(r) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	user := getUser(r)
	if user.Role != "manager" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	r.ParseForm()
	routeID := r.FormValue("route_id")
	routeName := r.FormValue("route_name")
	description := r.FormValue("description")

	if routeID == "" || routeName == "" {
		http.Error(w, "Route ID and name are required", http.StatusBadRequest)
		return
	}

	_, err := db.Exec(`
		UPDATE routes 
		SET route_name = $1, description = $2, updated_at = CURRENT_TIMESTAMP
		WHERE route_id = $3
	`, routeName, description, routeID)

	if err != nil {
		log.Printf("Error updating route: %v", err)
		http.Error(w, "Error updating route", http.StatusInternalServerError)
		return
	}

	// Clear routes cache
	cache.Delete("routes")

	http.Redirect(w, r, "/assign-routes", http.StatusSeeOther)
}

// deleteRouteHandler deletes a route
func deleteRouteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !isLoggedIn(r) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	user := getUser(r)
	if user.Role != "manager" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	r.ParseForm()
	routeID := r.FormValue("route_id")

	if routeID == "" {
		http.Error(w, "Route ID is required", http.StatusBadRequest)
		return
	}

	// Check if route has assignments
	var assignmentCount int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM driver_assignments WHERE route_id = $1
	`, routeID).Scan(&assignmentCount)

	if err == nil && assignmentCount > 0 {
		http.Error(w, "Cannot delete route with active assignments", http.StatusBadRequest)
		return
	}

	// Check if route has students
	var studentCount int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM students WHERE route_id = $1
	`, routeID).Scan(&studentCount)

	if err == nil && studentCount > 0 {
		http.Error(w, "Cannot delete route with assigned students", http.StatusBadRequest)
		return
	}

	// Delete route
	_, err = db.Exec(`DELETE FROM routes WHERE route_id = $1`, routeID)

	if err != nil {
		log.Printf("Error deleting route: %v", err)
		http.Error(w, "Error deleting route", http.StatusInternalServerError)
		return
	}

	// Clear routes cache
	cache.Delete("routes")

	http.Redirect(w, r, "/assign-routes", http.StatusSeeOther)
}

// studentsHandler manages student roster
func studentsHandler(w http.ResponseWriter, r *http.Request) {
	if !isLoggedIn(r) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	user := getUser(r)
	if user.Role != "driver" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get driver's route
	var routeID string
	err := db.QueryRow(`
		SELECT route_id FROM driver_assignments WHERE driver = $1
	`, user.Username).Scan(&routeID)

	// Get all routes for dropdown
	routes := loadRoutesFromCache()

	// Get students
	var students []Student
	query := `
		SELECT s.student_id, s.name, s.phone_number, s.alt_phone_number, 
			   s.guardian, s.route_id, s.position_number, s.active,
			   s.pickup_time, s.dropoff_time
		FROM students s
		WHERE s.driver = $1
		ORDER BY s.position_number, s.name
	`
	
	rows, err := db.Query(query, user.Username)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var s Student
			var altPhone sql.NullString
			var pickupTime, dropoffTime sql.NullString
			
			err := rows.Scan(&s.StudentID, &s.Name, &s.PhoneNumber, &altPhone,
				&s.Guardian, &s.RouteID, &s.PositionNumber, &s.Active,
				&pickupTime, &dropoffTime)
			if err != nil {
				log.Printf("Error scanning student: %v", err)
				continue
			}
			
			s.AltPhoneNumber = altPhone.String
			s.PickupTime = pickupTime.String
			s.DropoffTime = dropoffTime.String
			
			// Load student locations
			locRows, err := db.Query(`
				SELECT location_id, type, address, description
				FROM student_locations
				WHERE student_id = $1
				ORDER BY type, location_id
			`, s.StudentID)
			
			if err == nil {
				defer locRows.Close()
				for locRows.Next() {
					var loc Location
					err := locRows.Scan(&loc.LocationID, &loc.Type, &loc.Address, &loc.Description)
					if err == nil {
						s.Locations = append(s.Locations, loc)
					}
				}
			}
			
			students = append(students, s)
		}
	}

	data := map[string]interface{}{
		"User":      user,
		"Students":  students,
		"Routes":    routes,
		"CSRFToken": generateCSRFToken(),
	}
	executeTemplate(w, "students.html", data)
}

// addStudentHandler adds a new student
func addStudentHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !isLoggedIn(r) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	user := getUser(r)
	if user.Role != "driver" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	r.ParseForm()
	
	// Get form values
	name := r.FormValue("name")
	phoneNumber := r.FormValue("phone_number")
	altPhoneNumber := r.FormValue("alt_phone_number")
	guardian := r.FormValue("guardian")
	routeID := r.FormValue("route_id")
	pickupTime := r.FormValue("pickup_time")
	dropoffTime := r.FormValue("dropoff_time")

	// Validate required fields
	if name == "" || phoneNumber == "" || guardian == "" || routeID == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	// Generate student ID
	studentID := uuid.New().String()

	// Get next position number
	var maxPosition int
	err := db.QueryRow(`
		SELECT COALESCE(MAX(position_number), 0) 
		FROM students 
		WHERE driver = $1
	`, user.Username).Scan(&maxPosition)
	
	positionNumber := maxPosition + 1

	// Insert student
	_, err = db.Exec(`
		INSERT INTO students (student_id, name, phone_number, alt_phone_number, 
			guardian, route_id, position_number, driver, active, pickup_time, dropoff_time)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, studentID, name, phoneNumber, altPhoneNumber, guardian, routeID, 
		positionNumber, user.Username, true, pickupTime, dropoffTime)

	if err != nil {
		log.Printf("Error adding student: %v", err)
		http.Error(w, "Error adding student", http.StatusInternalServerError)
		return
	}

	// Add locations
	pickupAddresses := r.Form["pickup_address"]
	pickupDescriptions := r.Form["pickup_description"]
	
	for i := range pickupAddresses {
		if pickupAddresses[i] != "" {
			desc := ""
			if i < len(pickupDescriptions) {
				desc = pickupDescriptions[i]
			}
			
			_, err = db.Exec(`
				INSERT INTO student_locations (student_id, type, address, description)
				VALUES ($1, 'pickup', $2, $3)
			`, studentID, pickupAddresses[i], desc)
			
			if err != nil {
				log.Printf("Error adding pickup location: %v", err)
			}
		}
	}

	dropoffAddresses := r.Form["dropoff_address"]
	dropoffDescriptions := r.Form["dropoff_description"]
	
	for i := range dropoffAddresses {
		if dropoffAddresses[i] != "" {
			desc := ""
			if i < len(dropoffDescriptions) {
				desc = dropoffDescriptions[i]
			}
			
			_, err = db.Exec(`
				INSERT INTO student_locations (student_id, type, address, description)
				VALUES ($1, 'dropoff', $2, $3)
			`, studentID, dropoffAddresses[i], desc)
			
			if err != nil {
				log.Printf("Error adding dropoff location: %v", err)
			}
		}
	}

	http.Redirect(w, r, "/students", http.StatusSeeOther)
}

// editStudentHandler edits an existing student
func editStudentHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !isLoggedIn(r) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	user := getUser(r)
	if user.Role != "driver" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	r.ParseForm()
	
	// Get form values
	studentID := r.FormValue("student_id")
	name := r.FormValue("name")
	phoneNumber := r.FormValue("phone_number")
	altPhoneNumber := r.FormValue("alt_phone_number")
	guardian := r.FormValue("guardian")
	routeID := r.FormValue("route_id")
	active := r.FormValue("active") == "on"
	pickupTime := r.FormValue("pickup_time")
	dropoffTime := r.FormValue("dropoff_time")

	// Update student
	_, err := db.Exec(`
		UPDATE students 
		SET name = $1, phone_number = $2, alt_phone_number = $3, 
			guardian = $4, route_id = $5, active = $6, 
			pickup_time = $7, dropoff_time = $8, updated_at = CURRENT_TIMESTAMP
		WHERE student_id = $9 AND driver = $10
	`, name, phoneNumber, altPhoneNumber, guardian, routeID, active, 
		pickupTime, dropoffTime, studentID, user.Username)

	if err != nil {
		log.Printf("Error updating student: %v", err)
		http.Error(w, "Error updating student", http.StatusInternalServerError)
		return
	}

	// Delete existing locations
	_, err = db.Exec(`
		DELETE FROM student_locations WHERE student_id = $1
	`, studentID)

	// Add new locations
	pickupAddresses := r.Form["pickup_address"]
	pickupDescriptions := r.Form["pickup_description"]
	
	for i := range pickupAddresses {
		if pickupAddresses[i] != "" {
			desc := ""
			if i < len(pickupDescriptions) {
				desc = pickupDescriptions[i]
			}
			
			_, err = db.Exec(`
				INSERT INTO student_locations (student_id, type, address, description)
				VALUES ($1, 'pickup', $2, $3)
			`, studentID, pickupAddresses[i], desc)
		}
	}

	dropoffAddresses := r.Form["dropoff_address"]
	dropoffDescriptions := r.Form["dropoff_description"]
	
	for i := range dropoffAddresses {
		if dropoffAddresses[i] != "" {
			desc := ""
			if i < len(dropoffDescriptions) {
				desc = dropoffDescriptions[i]
			}
			
			_, err = db.Exec(`
				INSERT INTO student_locations (student_id, type, address, description)
				VALUES ($1, 'dropoff', $2, $3)
			`, studentID, dropoffAddresses[i], desc)
		}
	}

	http.Redirect(w, r, "/students", http.StatusSeeOther)
}

// removeStudentHandler removes a student
func removeStudentHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !isLoggedIn(r) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	user := getUser(r)
	if user.Role != "driver" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	r.ParseForm()
	studentID := r.FormValue("student_id")

	// Delete student locations first
	_, err := db.Exec(`
		DELETE FROM student_locations WHERE student_id = $1
	`, studentID)

	// Delete student
	_, err = db.Exec(`
		DELETE FROM students 
		WHERE student_id = $1 AND driver = $2
	`, studentID, user.Username)

	if err != nil {
		log.Printf("Error removing student: %v", err)
		http.Error(w, "Error removing student", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/students", http.StatusSeeOther)
}

// driverProfileHandler shows driver's trip history
func driverProfileHandler(w http.ResponseWriter, r *http.Request) {
	if !isLoggedIn(r) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	user := getUser(r)
	if user.Role != "manager" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get driver name from URL query
	driverName := r.URL.Query().Get("driver")
	if driverName == "" {
		http.Error(w, "Driver name required", http.StatusBadRequest)
		return
	}

	// Get driver's logs
	rows, err := db.Query(`
		SELECT id, date, period, route_id, bus_id, mileage, departure, arrival
		FROM driver_logs
		WHERE driver = $1
		ORDER BY date DESC, period DESC
	`, driverName)

	if err != nil {
		log.Printf("Error loading driver logs: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var logs []RouteLog
	for rows.Next() {
		var log RouteLog
		err := rows.Scan(&log.ID, &log.Date, &log.Period, 
			&log.RouteID, &log.BusID, &log.Mileage, 
			&log.Departure, &log.Arrival)
		if err != nil {
			continue
		}

		// Get attendance count
		var attendanceCount int
		db.QueryRow(`
			SELECT COUNT(*) FROM student_attendance 
			WHERE log_id = $1 AND present = true
		`, log.ID).Scan(&attendanceCount)
		
		// Create attendance slice with count
		log.Attendance = make([]StudentAttendance, attendanceCount)
		
		logs = append(logs, log)
	}

	data := map[string]interface{}{
		"Name": driverName,
		"Logs": logs,
		"CSRFToken": generateCSRFToken(),
	}
	
	executeTemplate(w, "driver_profile.html", data)
}

// logoutHandler handles user logout
func logoutHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session")
	if err == nil {
		mu.Lock()
		delete(sessions, cookie.Value)
		mu.Unlock()
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    "",
		MaxAge:   -1,
		HttpOnly: true,
	})

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// ====================
// Helper Functions
// ====================

// isLoggedIn checks if user has valid session
func isLoggedIn(r *http.Request) bool {
	cookie, err := r.Cookie("session")
	if err != nil {
		return false
	}

	mu.RLock()
	session, exists := sessions[cookie.Value]
	mu.RUnlock()

	if !exists || time.Now().After(session.Expires) {
		return false
	}

	return true
}

// getUser gets the current user from session
func getUser(r *http.Request) *User {
	cookie, err := r.Cookie("session")
	if err != nil {
		return nil
	}

	mu.RLock()
	session, exists := sessions[cookie.Value]
	mu.RUnlock()

	if !exists {
		return nil
	}

	return &User{
		Username: session.Username,
		Role:     session.Role,
	}
}

// authenticateUser verifies username and password
func authenticateUser(username, password string) (*User, error) {
	var user User
	var hashedPassword string

	err := db.QueryRow(`
		SELECT username, password, role, status 
		FROM users 
		WHERE username = $1 AND status = 'active'
	`, username).Scan(&user.Username, &hashedPassword, &user.Role, &user.Status)

	if err != nil {
		return nil, err
	}

	err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	if err != nil {
		return nil, err
	}

	return &user, nil
}

// generateSessionID generates a random session ID
func generateSessionID() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// generateCSRFToken generates a CSRF token
func generateCSRFToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// loadUsersFromCache loads users with caching
func loadUsersFromCache() []User {
	cacheKey := "users"
	if cached, found := cache.Get(cacheKey); found {
		return cached.([]User)
	}

	log.Println("Cache miss: loading users from database")
	
	rows, err := db.Query("SELECT username, role, status, created_at FROM users ORDER BY username")
	if err != nil {
		log.Printf("Error loading users: %v", err)
		return []User{}
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		err := rows.Scan(&u.Username, &u.Role, &u.Status, &u.CreatedAt)
		if err != nil {
			log.Printf("Error scanning user: %v", err)
			continue
		}
		users = append(users, u)
	}

	cache.Set(cacheKey, users, 5*time.Minute)
	return users
}

// loadBusesFromCache loads buses with caching
func loadBusesFromCache() []Bus {
	cacheKey := "buses"
	if cached, found := cache.Get(cacheKey); found {
		return cached.([]Bus)
	}

	log.Println("Cache miss: loading buses from database")
	
	rows, err := db.Query(`
		SELECT bus_id, model, capacity, status, oil_status, tire_status, maintenance_notes 
		FROM buses 
		ORDER BY bus_id
	`)
	if err != nil {
		log.Printf("Error loading buses: %v", err)
		return []Bus{}
	}
	defer rows.Close()

	var buses []Bus
	for rows.Next() {
		var b Bus
		var model, maintenanceNotes sql.NullString
		err := rows.Scan(&b.BusID, &model, &b.Capacity, &b.Status, 
			&b.OilStatus, &b.TireStatus, &maintenanceNotes)
		if err != nil {
			log.Printf("Error scanning bus: %v", err)
			continue
		}
		b.Model = model.String
		b.MaintenanceNotes = maintenanceNotes.String
		buses = append(buses, b)
	}

	cache.Set(cacheKey, buses, 5*time.Minute)
	return buses
}

// loadRoutesFromCache loads routes with caching
func loadRoutesFromCache() []Route {
	cacheKey := "routes"
	if cached, found := cache.Get(cacheKey); found {
		return cached.([]Route)
	}

	log.Println("Cache miss: loading routes from database")
	
	rows, err := db.Query("SELECT route_id, route_name, description FROM routes ORDER BY route_name")
	if err != nil {
		log.Printf("Error loading routes: %v", err)
		return []Route{}
	}
	defer rows.Close()

	var routes []Route
	for rows.Next() {
		var r Route
		var description sql.NullString
		err := rows.Scan(&r.RouteID, &r.RouteName, &description)
		if err != nil {
			log.Printf("Error scanning route: %v", err)
			continue
		}
		r.Description = description.String
		routes = append(routes, r)
	}

	cache.Set(cacheKey, routes, 5*time.Minute)
	return routes
}
