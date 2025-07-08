package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"  // Added missing import
	"os"
	"strconv"
	"time"

	_ "github.com/lib/pq"
	"github.com/jmoiron/sqlx"
)

var db *sqlx.DB

// setupDatabase initializes the PostgreSQL database connection and creates tables
func setupDatabase() error {
	// Get database URL from environment
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return fmt.Errorf("DATABASE_URL environment variable not set")
	}

	// Connect to database
	var err error
	db, err = sqlx.Connect("postgres", databaseURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Test connection
	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	log.Println("✅ Successfully connected to PostgreSQL database")

	// Create tables
	if err := createTables(); err != nil {
		return fmt.Errorf("failed to create tables: %w", err)
	}

	// Run migrations
	if err := runMigrations(); err != nil {
		log.Printf("Warning: Failed to run migrations: %v", err)
		// Don't fail completely, as tables might already exist
	}

	return nil
}

// closeDatabase closes the database connection
func closeDatabase() {
	if db != nil {
		if err := db.Close(); err != nil {
			log.Printf("Error closing database: %v", err)
		} else {
			log.Println("Database connection closed")
		}
	}
}

// createTables creates all necessary tables
func createTables() error {
	tables := []string{
		// Users table with new fields
		`CREATE TABLE IF NOT EXISTS users (
			username VARCHAR(50) PRIMARY KEY,
			password VARCHAR(255) NOT NULL,
			role VARCHAR(20) NOT NULL CHECK (role IN ('driver', 'manager', 'driver_pending')),
			status VARCHAR(20) DEFAULT 'active',
			registration_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// Buses table
		`CREATE TABLE IF NOT EXISTS buses (
			bus_id VARCHAR(50) PRIMARY KEY,
			status VARCHAR(20) DEFAULT 'active',
			model VARCHAR(100),
			capacity INTEGER DEFAULT 0,
			oil_status VARCHAR(20) DEFAULT 'OK',
			tire_status VARCHAR(20) DEFAULT 'OK',
			maintenance_notes TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// Routes table
		`CREATE TABLE IF NOT EXISTS routes (
			route_id VARCHAR(50) PRIMARY KEY,
			route_name VARCHAR(100) NOT NULL,
			description TEXT,
			positions JSONB DEFAULT '[]'::jsonb,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// Students table
		`CREATE TABLE IF NOT EXISTS students (
			student_id VARCHAR(50) PRIMARY KEY,
			name VARCHAR(100) NOT NULL,
			locations JSONB DEFAULT '[]'::jsonb,
			phone_number VARCHAR(20),
			alt_phone_number VARCHAR(20),
			guardian VARCHAR(100),
			pickup_time TIME,
			dropoff_time TIME,
			position_number INTEGER DEFAULT 0,
			route_id VARCHAR(50),
			driver VARCHAR(50),
			active BOOLEAN DEFAULT true,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (route_id) REFERENCES routes(route_id) ON DELETE SET NULL,
			FOREIGN KEY (driver) REFERENCES users(username) ON DELETE SET NULL
		)`,

		// Route assignments table
		`CREATE TABLE IF NOT EXISTS route_assignments (
			driver VARCHAR(50) PRIMARY KEY,
			bus_id VARCHAR(50) NOT NULL,
			route_id VARCHAR(50) NOT NULL,
			route_name VARCHAR(100),
			assigned_date DATE DEFAULT CURRENT_DATE,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (driver) REFERENCES users(username) ON DELETE CASCADE,
			FOREIGN KEY (bus_id) REFERENCES buses(bus_id) ON DELETE CASCADE,
			FOREIGN KEY (route_id) REFERENCES routes(route_id) ON DELETE CASCADE
		)`,

		// Driver logs table
		`CREATE TABLE IF NOT EXISTS driver_logs (
			driver VARCHAR(50) NOT NULL,
			bus_id VARCHAR(50),
			route_id VARCHAR(50),
			date DATE NOT NULL,
			period VARCHAR(20) NOT NULL CHECK (period IN ('morning', 'afternoon')),
			departure_time TIME,
			arrival_time TIME,
			mileage DECIMAL(10,2) DEFAULT 0,
			attendance JSONB DEFAULT '[]'::jsonb,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (driver, date, period),
			FOREIGN KEY (driver) REFERENCES users(username) ON DELETE CASCADE,
			FOREIGN KEY (bus_id) REFERENCES buses(bus_id) ON DELETE SET NULL,
			FOREIGN KEY (route_id) REFERENCES routes(route_id) ON DELETE SET NULL
		)`,

		// Bus maintenance logs table
		`CREATE TABLE IF NOT EXISTS bus_maintenance_logs (
			id SERIAL PRIMARY KEY,
			bus_id VARCHAR(50) NOT NULL,
			date DATE NOT NULL,
			category VARCHAR(50) NOT NULL,
			notes TEXT,
			mileage INTEGER DEFAULT 0,
			cost DECIMAL(10,2) DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (bus_id) REFERENCES buses(bus_id) ON DELETE CASCADE
		)`,

		// Vehicles table (for company fleet)
		`CREATE TABLE IF NOT EXISTS vehicles (
			vehicle_id VARCHAR(50) PRIMARY KEY,
			model VARCHAR(100),
			description TEXT,
			year INTEGER,
			tire_size VARCHAR(20),
			license VARCHAR(20),
			oil_status VARCHAR(20) DEFAULT 'OK',
			tire_status VARCHAR(20) DEFAULT 'OK',
			status VARCHAR(20) DEFAULT 'active',
			maintenance_notes TEXT,
			serial_number VARCHAR(100),
			base VARCHAR(100),
			service_interval INTEGER DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// Activities table
		`CREATE TABLE IF NOT EXISTS activities (
			id SERIAL PRIMARY KEY,
			date DATE NOT NULL,
			driver VARCHAR(50),
			trip_name VARCHAR(100),
			attendance INTEGER DEFAULT 0,
			miles DECIMAL(10,2) DEFAULT 0,
			notes TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (driver) REFERENCES users(username) ON DELETE SET NULL
		)`,

		// Unified maintenance records table
		`CREATE TABLE IF NOT EXISTS maintenance_records (
			id SERIAL PRIMARY KEY,
			vehicle_id VARCHAR(50) NOT NULL,
			vehicle_type VARCHAR(20) NOT NULL CHECK (vehicle_type IN ('bus', 'vehicle')),
			date DATE NOT NULL,
			category VARCHAR(50) NOT NULL,
			notes TEXT,
			mileage INTEGER DEFAULT 0,
			cost DECIMAL(10,2) DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// Mileage reporting tables
		`CREATE TABLE IF NOT EXISTS agency_vehicles (
			report_month VARCHAR(50) NOT NULL,
			report_year INTEGER NOT NULL,
			vehicle_year INTEGER,
			make_model VARCHAR(100),
			license_plate VARCHAR(50),
			vehicle_id VARCHAR(50) NOT NULL,
			location VARCHAR(100),
			beginning_miles INTEGER DEFAULT 0,
			ending_miles INTEGER DEFAULT 0,
			total_miles INTEGER DEFAULT 0,
			status VARCHAR(50),
			notes TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (report_month, report_year, vehicle_id)
		)`,

		`CREATE TABLE IF NOT EXISTS school_buses (
			report_month VARCHAR(50) NOT NULL,
			report_year INTEGER NOT NULL,
			bus_year INTEGER,
			bus_make VARCHAR(100),
			license_plate VARCHAR(50),
			bus_id VARCHAR(50) NOT NULL,
			location VARCHAR(100),
			beginning_miles INTEGER DEFAULT 0,
			ending_miles INTEGER DEFAULT 0,
			total_miles INTEGER DEFAULT 0,
			status VARCHAR(50),
			notes TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (report_month, report_year, bus_id)
		)`,

		`CREATE TABLE IF NOT EXISTS program_staff (
			report_month VARCHAR(50) NOT NULL,
			report_year INTEGER NOT NULL,
			program_type VARCHAR(20) NOT NULL CHECK (program_type IN ('HS', 'OPK', 'EHS')),
			staff_count_1 INTEGER DEFAULT 0,
			staff_count_2 INTEGER DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (report_month, report_year, program_type)
		)`,

		// Service records table (from legacy system)
		`CREATE TABLE IF NOT EXISTS service_records (
			id SERIAL PRIMARY KEY,
			vehicle_number VARCHAR(50),
			vehicle_id VARCHAR(50),
			unnamed_1 VARCHAR(100),
			date DATE,
			service_type VARCHAR(100),
			notes TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
	}

	// Execute each table creation
	for _, table := range tables {
		if _, err := db.Exec(table); err != nil {
			return fmt.Errorf("failed to create table: %w", err)
		}
	}

	// Create indexes for better performance
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_students_driver ON students(driver)`,
		`CREATE INDEX IF NOT EXISTS idx_students_route ON students(route_id)`,
		`CREATE INDEX IF NOT EXISTS idx_driver_logs_date ON driver_logs(date)`,
		`CREATE INDEX IF NOT EXISTS idx_maintenance_date ON bus_maintenance_logs(date)`,
		`CREATE INDEX IF NOT EXISTS idx_maintenance_bus ON bus_maintenance_logs(bus_id)`,
		`CREATE INDEX IF NOT EXISTS idx_route_assignments_bus ON route_assignments(bus_id)`,
		`CREATE INDEX IF NOT EXISTS idx_route_assignments_route ON route_assignments(route_id)`,
		`CREATE INDEX IF NOT EXISTS idx_vehicles_status ON vehicles(status)`,
		`CREATE INDEX IF NOT EXISTS idx_buses_status ON buses(status)`,
	}

	for _, index := range indexes {
		if _, err := db.Exec(index); err != nil {
			log.Printf("Warning: Failed to create index: %v", err)
			// Don't fail on index creation errors
		}
	}

	log.Println("✅ All tables created successfully")
	return nil
}

// runMigrations runs any pending database migrations
func runMigrations() error {
	// Add any new columns that might not exist
	migrations := []string{
		// Add status column to users if it doesn't exist
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS status VARCHAR(20) DEFAULT 'active'`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS registration_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP`,
		
		// Add route_name to route_assignments if it doesn't exist
		`ALTER TABLE route_assignments ADD COLUMN IF NOT EXISTS route_name VARCHAR(100)`,
		
		// Add cost column to bus_maintenance_logs if it doesn't exist
		`ALTER TABLE bus_maintenance_logs ADD COLUMN IF NOT EXISTS cost DECIMAL(10,2) DEFAULT 0`,
		
		// Add updated_at to various tables
		`ALTER TABLE buses ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP`,
		`ALTER TABLE routes ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP`,
		`ALTER TABLE students ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP`,
		`ALTER TABLE vehicles ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP`,
		`ALTER TABLE maintenance_records ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP`,
	}

	for _, migration := range migrations {
		if _, err := db.Exec(migration); err != nil {
			log.Printf("Migration warning: %v", err)
			// Continue with other migrations
		}
	}

	return nil
}

// Database query functions

// FIXED: getDriverBusForRoute gets the bus assigned to a driver for a specific route
func getDriverBusForRoute(driver, routeID string) (string, error) {
	if db == nil {
		return "", fmt.Errorf("database not initialized")
	}
	
	var busID string
	err := db.Get(&busID, `
		SELECT bus_id 
		FROM route_assignments 
		WHERE driver = $1 AND route_id = $2
	`, driver, routeID)
	
	if err == sql.ErrNoRows {
		return "", nil
	}
	return busID, err
}

// getDriverAssignedBus gets the bus assigned to a driver (legacy, returns first bus)
func getDriverAssignedBus(driver string) (string, error) {
	if db == nil {
		return "", fmt.Errorf("database not initialized")
	}
	
	var busID string
	err := db.Get(&busID, `
		SELECT bus_id 
		FROM route_assignments 
		WHERE driver = $1 
		ORDER BY created_at DESC 
		LIMIT 1
	`, driver)
	
	if err == sql.ErrNoRows {
		return "", nil
	}
	return busID, err
}

// getDriverRouteAssignment gets the route assignment for a driver
func getDriverRouteAssignment(driver string) (*RouteAssignment, error) {
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	
	var assignment RouteAssignment
	err := db.Get(&assignment, `
		SELECT driver, bus_id, route_id, 
		       COALESCE(route_name, '') as route_name, 
		       assigned_date::text
		FROM route_assignments 
		WHERE driver = $1
		ORDER BY created_at DESC
		LIMIT 1
	`, driver)
	
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	
	return &assignment, nil
}

// saveRouteAssignment saves or updates a route assignment
func saveRouteAssignment(assignment RouteAssignment) error {
	if db == nil {
		return fmt.Errorf("database not initialized")
	}
	
	// Validate assignment
	if err := validateRouteAssignment(assignment); err != nil {
		return err
	}
	
	_, err := db.Exec(`
		INSERT INTO route_assignments (driver, bus_id, route_id, route_name, assigned_date) 
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (driver) 
		DO UPDATE SET 
			bus_id = $2, 
			route_id = $3, 
			route_name = $4,
			assigned_date = $5
	`, assignment.Driver, assignment.BusID, assignment.RouteID, 
	   assignment.RouteName, assignment.AssignedDate)
	
	return err
}

// validateRouteAssignment checks if an assignment is valid
func validateRouteAssignment(assignment RouteAssignment) error {
	if assignment.Driver == "" || assignment.BusID == "" || assignment.RouteID == "" {
		return fmt.Errorf("missing required fields")
	}
	
	// Check if bus is already assigned to another driver
	var existingDriver string
	err := db.Get(&existingDriver, `
		SELECT driver FROM route_assignments 
		WHERE bus_id = $1 AND driver != $2
	`, assignment.BusID, assignment.Driver)
	
	if err == nil && existingDriver != "" {
		return fmt.Errorf("bus %s is already assigned to driver %s", assignment.BusID, existingDriver)
	}
	
	return nil
}

// getRecentMaintenanceActivity gets recent maintenance records
func getRecentMaintenanceActivity(limit int) ([]BusMaintenanceLog, error) {
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	
	var logs []BusMaintenanceLog
	err := db.Select(&logs, `
		SELECT bus_id, date::text, category, notes, mileage 
		FROM bus_maintenance_logs 
		ORDER BY date DESC, created_at DESC 
		LIMIT $1
	`, limit)
	
	return logs, err
}

// getAllVehicleMaintenanceRecords gets all maintenance records for a vehicle
func getAllVehicleMaintenanceRecords(vehicleID string) ([]BusMaintenanceLog, error) {
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	
	var records []BusMaintenanceLog
	
	// Get from bus_maintenance_logs
	err := db.Select(&records, `
		SELECT bus_id, date::text, category, notes, mileage 
		FROM bus_maintenance_logs 
		WHERE bus_id = $1 
		ORDER BY date DESC
	`, vehicleID)
	
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	
	// Also get from maintenance_records
	var additionalRecords []BusMaintenanceLog
	err = db.Select(&additionalRecords, `
		SELECT vehicle_id as bus_id, date::text, category, notes, mileage 
		FROM maintenance_records 
		WHERE vehicle_id = $1 
		ORDER BY date DESC
	`, vehicleID)
	
	if err == nil {
		records = append(records, additionalRecords...)
	}
	
	return records, nil
}

// saveMaintenanceRecord saves a maintenance record to the unified table
func saveMaintenanceRecord(log BusMaintenanceLog, vehicleType string) error {
	if db == nil {
		return fmt.Errorf("database not initialized")
	}
	
	_, err := db.Exec(`
		INSERT INTO maintenance_records 
		(vehicle_id, vehicle_type, date, category, notes, mileage) 
		VALUES ($1, $2, $3, $4, $5, $6)
	`, log.BusID, vehicleType, log.Date, log.Category, log.Notes, log.Mileage)
	
	return err
}

// debugMaintenanceTables helps debug maintenance table issues
func debugMaintenanceTables(vehicleID string) {
	if db == nil {
		log.Println("Database not initialized")
		return
	}
	
	log.Printf("\n=== Debugging maintenance tables for vehicle: %s ===", vehicleID)
	
	// Check bus_maintenance_logs
	var count1 int
	db.Get(&count1, "SELECT COUNT(*) FROM bus_maintenance_logs WHERE bus_id = $1", vehicleID)
	log.Printf("Records in bus_maintenance_logs: %d", count1)
	
	// Check maintenance_records
	var count2 int
	db.Get(&count2, "SELECT COUNT(*) FROM maintenance_records WHERE vehicle_id = $1", vehicleID)
	log.Printf("Records in maintenance_records: %d", count2)
	
	// Check service_records
	var count3 int
	db.Get(&count3, `
		SELECT COUNT(*) FROM service_records 
		WHERE COALESCE(vehicle_number::VARCHAR, vehicle_id::VARCHAR, unnamed_1::VARCHAR) = $1
	`, vehicleID)
	log.Printf("Records in service_records: %d", count3)
	
	// Sample records
	var sample BusMaintenanceLog
	err := db.Get(&sample, `
		SELECT bus_id, date::text, category, notes, mileage 
		FROM bus_maintenance_logs 
		WHERE bus_id = $1 
		ORDER BY date DESC 
		LIMIT 1
	`, vehicleID)
	
	if err == nil {
		log.Printf("Latest maintenance record: %+v", sample)
	} else {
		log.Printf("Error getting sample record: %v", err)
	}
	
	log.Printf("=== End debug ===\n")
}

// API handlers for route assignments

func handleSaveRouteAssignment(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var assignment RouteAssignment
	if err := json.NewDecoder(r.Body).Decode(&assignment); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	if err := saveRouteAssignment(assignment); err != nil {
		log.Printf("Error saving route assignment: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "success",
		"message": "Route assignment saved successfully",
	})
}

func handleCheckDriverBus(w http.ResponseWriter, r *http.Request) {
	driver := r.URL.Query().Get("driver")
	routeID := r.URL.Query().Get("route_id")
	
	if driver == "" {
		http.Error(w, "Driver parameter required", http.StatusBadRequest)
		return
	}
	
	var busID string
	var err error
	
	if routeID != "" {
		busID, err = getDriverBusForRoute(driver, routeID)
	} else {
		busID, err = getDriverAssignedBus(driver)
	}
	
	if err != nil {
		log.Printf("Error checking driver bus: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"bus_id": busID,
	})
}

// Enhanced mileage reports handler
func viewEnhancedMileageReportsHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	
	// Get query parameters
	reportMonth := r.URL.Query().Get("month")
	reportYear := r.URL.Query().Get("year")
	reportType := r.URL.Query().Get("type")
	
	// Default to current month/year if not specified
	if reportMonth == "" || reportYear == "" {
		now := time.Now()
		reportMonth = now.Format("January")
		reportYear = fmt.Sprintf("%d", now.Year())
	}
	
	// Get data based on report type
	var data interface{}
	var err error
	
	switch reportType {
	case "agency":
		data, err = getAgencyVehicleReports(reportMonth, reportYear)
	case "school":
		data, err = getSchoolBusReports(reportMonth, reportYear)
	case "program":
		data, err = getProgramStaffReports(reportMonth, reportYear)
	default:
		// Get all types
		agencyData, _ := getAgencyVehicleReports(reportMonth, reportYear)
		schoolData, _ := getSchoolBusReports(reportMonth, reportYear)
		programData, _ := getProgramStaffReports(reportMonth, reportYear)
		
		data = struct {
			AgencyVehicles []AgencyVehicleRecord
			SchoolBuses    []SchoolBusRecord
			ProgramStaff   []ProgramStaffRecord
		}{
			AgencyVehicles: agencyData,
			SchoolBuses:    schoolData,
			ProgramStaff:   programData,
		}
	}
	
	if err != nil {
		log.Printf("Error getting mileage reports: %v", err)
	}
	
	// Get available months/years
	availableReports, _ := getAvailableReports()
	
	templateData := struct {
		User             *User
		ReportMonth      string
		ReportYear       string
		ReportType       string
		Data             interface{}
		AvailableReports []string
		CSRFToken        string
	}{
		User:             user,
		ReportMonth:      reportMonth,
		ReportYear:       reportYear,
		ReportType:       reportType,
		Data:             data,
		AvailableReports: availableReports,
		CSRFToken:        getCSRFToken(r),
	}
	
	renderTemplate(w, r, "mileage_reports.html", templateData)
}

// Get agency vehicle reports
func getAgencyVehicleReports(month, year string) ([]AgencyVehicleRecord, error) {
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	
	yearInt, _ := strconv.Atoi(year)
	
	var records []AgencyVehicleRecord
	err := db.Select(&records, `
		SELECT * FROM agency_vehicles 
		WHERE report_month = $1 AND report_year = $2 
		ORDER BY vehicle_id
	`, month, yearInt)
	
	return records, err
}

// Get school bus reports
func getSchoolBusReports(month, year string) ([]SchoolBusRecord, error) {
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	
	yearInt, _ := strconv.Atoi(year)
	
	var records []SchoolBusRecord
	err := db.Select(&records, `
		SELECT * FROM school_buses 
		WHERE report_month = $1 AND report_year = $2 
		ORDER BY bus_id
	`, month, yearInt)
	
	return records, err
}

// Get program staff reports
func getProgramStaffReports(month, year string) ([]ProgramStaffRecord, error) {
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	
	yearInt, _ := strconv.Atoi(year)
	
	var records []ProgramStaffRecord
	err := db.Select(&records, `
		SELECT * FROM program_staff 
		WHERE report_month = $1 AND report_year = $2 
		ORDER BY program_type
	`, month, yearInt)
	
	return records, err
}

// Get available report months/years
func getAvailableReports() ([]string, error) {
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	
	var reports []string
	
	// Get unique month/year combinations
	err := db.Select(&reports, `
		SELECT DISTINCT report_month || ' ' || report_year::text as report
		FROM (
			SELECT report_month, report_year FROM agency_vehicles
			UNION
			SELECT report_month, report_year FROM school_buses
			UNION
			SELECT report_month, report_year FROM program_staff
		) combined
		ORDER BY report DESC
	`)
	
	return reports, err
}
