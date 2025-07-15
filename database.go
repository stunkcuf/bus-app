package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/lib/pq"
	"github.com/jmoiron/sqlx"
	"github.com/xuri/excelize/v2"
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
		// ECSE Tables
		`CREATE TABLE IF NOT EXISTS ecse_students (
		    student_id VARCHAR(50) PRIMARY KEY,
		    first_name VARCHAR(100) NOT NULL,
		    last_name VARCHAR(100) NOT NULL,
		    date_of_birth DATE,
		    grade VARCHAR(20),
		    enrollment_status VARCHAR(50) DEFAULT 'Active',
		    iep_status VARCHAR(50),
		    primary_disability VARCHAR(100),
		    service_minutes INTEGER DEFAULT 0,
		    transportation_required BOOLEAN DEFAULT false,
		    bus_route VARCHAR(50),
		    parent_name VARCHAR(200),
		    parent_phone VARCHAR(20),
		    parent_email VARCHAR(100),
		    address VARCHAR(200),
		    city VARCHAR(100),
		    state VARCHAR(2),
		    zip_code VARCHAR(10),
		    notes TEXT,
		    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		
		`CREATE TABLE IF NOT EXISTS ecse_services (
		    id SERIAL PRIMARY KEY,
		    student_id VARCHAR(50) NOT NULL,
		    service_type VARCHAR(100) NOT NULL,
		    frequency VARCHAR(50),
		    duration INTEGER DEFAULT 0,
		    provider VARCHAR(100),
		    start_date DATE,
		    end_date DATE,
		    goals TEXT,
		    progress TEXT,
		    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		    FOREIGN KEY (student_id) REFERENCES ecse_students(student_id) ON DELETE CASCADE
		)`,
		
		`CREATE TABLE IF NOT EXISTS ecse_assessments (
		    id SERIAL PRIMARY KEY,
		    student_id VARCHAR(50) NOT NULL,
		    assessment_type VARCHAR(100) NOT NULL,
		    assessment_date DATE,
		    score VARCHAR(50),
		    evaluator VARCHAR(100),
		    notes TEXT,
		    next_review_date DATE,
		    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		    FOREIGN KEY (student_id) REFERENCES ecse_students(student_id) ON DELETE CASCADE
		)`,
		
		`CREATE TABLE IF NOT EXISTS ecse_attendance (
		    id SERIAL PRIMARY KEY,
		    student_id VARCHAR(50) NOT NULL,
		    attendance_date DATE NOT NULL,
		    status VARCHAR(20) NOT NULL CHECK (status IN ('Present', 'Absent', 'Tardy', 'Excused')),
		    arrival_time TIME,
		    departure_time TIME,
		    notes TEXT,
		    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		    FOREIGN KEY (student_id) REFERENCES ecse_students(student_id) ON DELETE CASCADE,
		    UNIQUE(student_id, attendance_date)
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
		
		// Unified all_vehicle_mileage table - ensure it exists with proper structure
		`CREATE TABLE IF NOT EXISTS all_vehicle_mileage (
			id SERIAL PRIMARY KEY,
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
			status VARCHAR(50) DEFAULT 'active',
			notes TEXT,
			vehicle_type VARCHAR(20) NOT NULL DEFAULT 'agency',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(report_month, report_year, vehicle_id)
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
		`CREATE INDEX IF NOT EXISTS idx_activities_date ON activities(date)`,
		`CREATE INDEX IF NOT EXISTS idx_activities_driver ON activities(driver)`,
		`CREATE INDEX IF NOT EXISTS idx_maintenance_records_vehicle ON maintenance_records(vehicle_id)`,
		`CREATE INDEX IF NOT EXISTS idx_maintenance_records_date ON maintenance_records(date)`,
		// ECSE indexes for better performance
		`CREATE INDEX IF NOT EXISTS idx_ecse_students_enrollment ON ecse_students(enrollment_status)`,
		`CREATE INDEX IF NOT EXISTS idx_ecse_students_transportation ON ecse_students(transportation_required)`,
		`CREATE INDEX IF NOT EXISTS idx_ecse_students_bus_route ON ecse_students(bus_route)`,
		`CREATE INDEX IF NOT EXISTS idx_ecse_services_student ON ecse_services(student_id)`,
		`CREATE INDEX IF NOT EXISTS idx_ecse_assessments_student ON ecse_assessments(student_id)`,
		`CREATE INDEX IF NOT EXISTS idx_ecse_attendance_date ON ecse_attendance(attendance_date)`,
		// All vehicle mileage indexes
		`CREATE INDEX IF NOT EXISTS idx_all_vehicle_mileage_type ON all_vehicle_mileage(vehicle_type)`,
		`CREATE INDEX IF NOT EXISTS idx_all_vehicle_mileage_month_year ON all_vehicle_mileage(report_month, report_year)`,
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
		
		// Add maintenance_date to service_records if missing
		`ALTER TABLE service_records ADD COLUMN IF NOT EXISTS maintenance_date DATE`,
		
		// Add missing columns to all_vehicle_mileage table if it exists
		`ALTER TABLE all_vehicle_mileage ADD COLUMN IF NOT EXISTS id SERIAL`,
		`ALTER TABLE all_vehicle_mileage ADD COLUMN IF NOT EXISTS vehicle_year INTEGER`,
		`ALTER TABLE all_vehicle_mileage ADD COLUMN IF NOT EXISTS make_model VARCHAR(100)`,
		`ALTER TABLE all_vehicle_mileage ADD COLUMN IF NOT EXISTS license_plate VARCHAR(50)`,
		`ALTER TABLE all_vehicle_mileage ADD COLUMN IF NOT EXISTS location VARCHAR(100)`,
		`ALTER TABLE all_vehicle_mileage ADD COLUMN IF NOT EXISTS beginning_miles INTEGER DEFAULT 0`,
		`ALTER TABLE all_vehicle_mileage ADD COLUMN IF NOT EXISTS ending_miles INTEGER DEFAULT 0`,
		`ALTER TABLE all_vehicle_mileage ADD COLUMN IF NOT EXISTS total_miles INTEGER DEFAULT 0`,
		`ALTER TABLE all_vehicle_mileage ADD COLUMN IF NOT EXISTS status VARCHAR(50) DEFAULT 'active'`,
		`ALTER TABLE all_vehicle_mileage ADD COLUMN IF NOT EXISTS notes TEXT`,
		`ALTER TABLE all_vehicle_mileage ADD COLUMN IF NOT EXISTS vehicle_type VARCHAR(20) DEFAULT 'agency'`,
		`ALTER TABLE all_vehicle_mileage ADD COLUMN IF NOT EXISTS created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP`,
		`ALTER TABLE all_vehicle_mileage ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP`,
		
		// Add constraint if not exists (more complex, needs checking)
		`DO $ 
		BEGIN
			IF NOT EXISTS (
				SELECT 1 FROM pg_constraint 
				WHERE conname = 'all_vehicle_mileage_vehicle_type_check'
			) THEN
				ALTER TABLE all_vehicle_mileage 
				ADD CONSTRAINT all_vehicle_mileage_vehicle_type_check 
				CHECK (vehicle_type IN ('agency', 'bus'));
			END IF;
		END $;`,
	}

	for _, migration := range migrations {
		if _, err := db.Exec(migration); err != nil {
			log.Printf("Migration warning: %v", err)
			// Continue with other migrations
		}
	}

	return nil
}

// =============================================================================
// DATABASE QUERY FUNCTIONS WITH COMPREHENSIVE LOG RETRIEVAL
// =============================================================================

// getDriverBusForRoute gets the bus assigned to a driver for a specific route
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

// =============================================================================
// ENHANCED LOG RETRIEVAL FUNCTIONS
// =============================================================================

// getRecentMaintenanceActivity gets recent maintenance records across all tables
func getRecentMaintenanceActivity(limit int) ([]BusMaintenanceLog, error) {
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	
	query := `
		SELECT * FROM (
			-- Bus maintenance logs
			SELECT id, bus_id as vehicle_id, date::text, category, notes, mileage, cost, created_at
			FROM bus_maintenance_logs
			
			UNION ALL
			
			-- Maintenance records
			SELECT id, vehicle_id, date::text, category, notes, mileage, cost, created_at
			FROM maintenance_records
			
			UNION ALL
			
			-- Service records (legacy)
			SELECT id, COALESCE(vehicle_id, vehicle_number) as vehicle_id, 
			       COALESCE(maintenance_date::text, date::text, '') as date,
			       COALESCE(service_type, 'service') as category,
			       notes, 0 as mileage, 0 as cost, created_at
			FROM service_records
			WHERE vehicle_id IS NOT NULL OR vehicle_number IS NOT NULL
		) combined
		ORDER BY created_at DESC
		LIMIT $1
	`
	
	var logs []BusMaintenanceLog
	err := db.Select(&logs, query, limit)
	
	return logs, err
}

// getAllVehicleMaintenanceRecords gets all maintenance records for a vehicle
func getAllVehicleMaintenanceRecords(vehicleID string) ([]BusMaintenanceLog, error) {
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	
	query := `
		SELECT * FROM (
			-- Bus maintenance logs
			SELECT id, bus_id, date::text, category, notes, mileage, cost, created_at
			FROM bus_maintenance_logs
			WHERE bus_id = $1
			
			UNION ALL
			
			-- Maintenance records
			SELECT id, vehicle_id as bus_id, date::text, category, notes, mileage, cost, created_at
			FROM maintenance_records
			WHERE vehicle_id = $1
			
			UNION ALL
			
			-- Service records (legacy)
			SELECT id, COALESCE(vehicle_id, vehicle_number) as bus_id,
			       COALESCE(maintenance_date::text, date::text, '') as date,
			       COALESCE(service_type, 'service') as category,
			       notes, 0 as mileage, 0 as cost, created_at
			FROM service_records
			WHERE vehicle_id = $1 OR vehicle_number = $1
		) combined
		WHERE date != ''
		ORDER BY date DESC, created_at DESC
	`
	
	var records []BusMaintenanceLog
	err := db.Select(&records, query, vehicleID)
	
	if err != nil && err != sql.ErrNoRows {
		return nil, err
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
		(vehicle_id, vehicle_type, date, category, notes, mileage, cost) 
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, log.BusID, vehicleType, log.Date, log.Category, log.Notes, log.Mileage, log.Cost)
	
	return err
}

// getDriverLogSummary gets a summary of driver logs
func getDriverLogSummary(driver string, days int) (map[string]interface{}, error) {
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	
	startDate := time.Now().AddDate(0, 0, -days).Format("2006-01-02")
	summary := make(map[string]interface{})
	
	// Total miles driven
	var totalMiles sql.NullFloat64
	err := db.QueryRow(`
		SELECT SUM(mileage) FROM driver_logs 
		WHERE driver = $1 AND date >= $2
	`, driver, startDate).Scan(&totalMiles)
	
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	summary["total_miles"] = totalMiles.Float64
	
	// Number of trips
	var tripCount int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM driver_logs 
		WHERE driver = $1 AND date >= $2
	`, driver, startDate).Scan(&tripCount)
	
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	summary["trip_count"] = tripCount
	
	// Average students per trip (based on attendance)
	rows, err := db.Query(`
		SELECT attendance FROM driver_logs 
		WHERE driver = $1 AND date >= $2 AND attendance IS NOT NULL
	`, driver, startDate)
	
	if err == nil {
		defer rows.Close()
		totalStudents := 0
		validTrips := 0
		
		for rows.Next() {
			var attendanceJSON []byte
			if err := rows.Scan(&attendanceJSON); err == nil && len(attendanceJSON) > 0 {
				var attendance []StudentAttendance
				if json.Unmarshal(attendanceJSON, &attendance) == nil {
					presentCount := 0
					for _, a := range attendance {
						if a.Present {
							presentCount++
						}
					}
					totalStudents += presentCount
					validTrips++
				}
			}
		}
		
		if validTrips > 0 {
			summary["avg_students_per_trip"] = float64(totalStudents) / float64(validTrips)
		} else {
			summary["avg_students_per_trip"] = 0
		}
	}
	
	// Recent activity count
	var recentCount int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM driver_logs 
		WHERE driver = $1 AND date >= $2
	`, driver, time.Now().AddDate(0, 0, -7).Format("2006-01-02")).Scan(&recentCount)
	
	if err == nil {
		summary["recent_trips_7_days"] = recentCount
	}
	
	// Special trips/activities
	var activityCount int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM activities 
		WHERE driver = $1 AND date >= $2
	`, driver, startDate).Scan(&activityCount)
	
	if err == nil {
		summary["special_trips"] = activityCount
	}
	
	return summary, nil
}

// getFleetMaintenanceOverview gets maintenance overview for all vehicles
func getFleetMaintenanceOverview() (map[string]interface{}, error) {
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	
	overview := make(map[string]interface{})
	
	// Vehicles needing attention (based on status)
	var needsAttention int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM (
			SELECT bus_id FROM buses 
			WHERE oil_status != 'good' OR tire_status != 'good' OR status != 'active'
			UNION
			SELECT vehicle_id FROM vehicles 
			WHERE oil_status != 'good' OR tire_status != 'good' OR status != 'active'
		) AS vehicles_needing_attention
	`).Scan(&needsAttention)
	
	if err == nil {
		overview["vehicles_needing_attention"] = needsAttention
	}
	
	// Recent maintenance (last 30 days)
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30).Format("2006-01-02")
	
	var recentMaintenanceCount int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM (
			SELECT id FROM bus_maintenance_logs WHERE date >= $1
			UNION ALL
			SELECT id FROM maintenance_records WHERE date >= $1
		) AS recent_maintenance
	`, thirtyDaysAgo).Scan(&recentMaintenanceCount)
	
	if err == nil {
		overview["recent_maintenance_count"] = recentMaintenanceCount
	}
	
	// Total maintenance cost (last 30 days)
	var totalCost sql.NullFloat64
	err = db.QueryRow(`
		SELECT SUM(cost) FROM (
			SELECT cost FROM bus_maintenance_logs WHERE date >= $1
			UNION ALL
			SELECT cost FROM maintenance_records WHERE date >= $1
		) AS maintenance_costs
	`, thirtyDaysAgo).Scan(&totalCost)
	
	if err == nil {
		overview["recent_maintenance_cost"] = totalCost.Float64
	}
	
	// Overdue maintenance (vehicles without maintenance in 90+ days)
	ninetyDaysAgo := time.Now().AddDate(0, 0, -90).Format("2006-01-02")
	
	rows, err := db.Query(`
		SELECT DISTINCT vehicle_id FROM (
			SELECT bus_id as vehicle_id FROM buses
			WHERE bus_id NOT IN (
				SELECT DISTINCT bus_id FROM bus_maintenance_logs WHERE date >= $1
			)
			UNION
			SELECT vehicle_id FROM vehicles
			WHERE vehicle_id NOT IN (
				SELECT DISTINCT vehicle_id FROM maintenance_records WHERE date >= $1
			)
		) AS overdue_vehicles
	`, ninetyDaysAgo)
	
	if err == nil {
		defer rows.Close()
		overdueCount := 0
		var overdueVehicles []string
		
		for rows.Next() {
			var vehicleID string
			if err := rows.Scan(&vehicleID); err == nil {
				overdueCount++
				overdueVehicles = append(overdueVehicles, vehicleID)
			}
		}
		
		overview["overdue_maintenance_count"] = overdueCount
		overview["overdue_vehicles"] = overdueVehicles
	}
	
	return overview, nil
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

// =============================================================================
// API HANDLERS FOR LOGS AND REPORTS
// =============================================================================

// handleSaveRouteAssignment handles route assignment saving
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

// handleCheckDriverBus checks driver's assigned bus
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

// handleGetDriverLogs returns driver logs with optional filtering
func handleGetDriverLogs(w http.ResponseWriter, r *http.Request) {
	driver := r.URL.Query().Get("driver")
	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")
	limit := r.URL.Query().Get("limit")
	
	var logs []DriverLog
	var err error
	
	if driver != "" && startDate != "" && endDate != "" {
		logs, err = getDriverLogsByDateRange(driver, startDate, endDate)
	} else if driver != "" && limit != "" {
		limitInt, _ := strconv.Atoi(limit)
		logs, err = loadDriverLogsForDriver(driver, limitInt)
	} else {
		logs, err = loadDriverLogsFromDB()
	}
	
	if err != nil {
		log.Printf("Error getting driver logs: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logs)
}

// handleGetMaintenanceLogs returns maintenance logs for a vehicle
func handleGetMaintenanceLogs(w http.ResponseWriter, r *http.Request) {
	vehicleID := r.URL.Query().Get("vehicle_id")
	
	if vehicleID == "" {
		http.Error(w, "Vehicle ID required", http.StatusBadRequest)
		return
	}
	
	logs, err := getAllVehicleMaintenanceRecords(vehicleID)
	if err != nil {
		log.Printf("Error getting maintenance logs: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logs)
}

// handleGetLogSummary returns log summary for a driver
func handleGetLogSummary(w http.ResponseWriter, r *http.Request) {
	driver := r.URL.Query().Get("driver")
	days := r.URL.Query().Get("days")
	
	if driver == "" {
		http.Error(w, "Driver parameter required", http.StatusBadRequest)
		return
	}
	
	daysInt := 30 // default
	if days != "" {
		if d, err := strconv.Atoi(days); err == nil {
			daysInt = d
		}
	}
	
	summary, err := getDriverLogSummary(driver, daysInt)
	if err != nil {
		log.Printf("Error getting driver log summary: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
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
	
	// Initialize data structure that matches what the template expects
	type MileageReportData struct {
		AgencyVehicles []AgencyVehicleRecord
		SchoolBuses    []SchoolBusRecord
		ProgramStaff   []ProgramStaffRecord
		TotalVehicles  int
		TotalMiles     int
		TotalActive    int
	}
	
	// Get data based on report type
	var data MileageReportData
	var err error
	
	switch reportType {
	case "agency":
		data.AgencyVehicles, err = getAgencyVehicleReports(reportMonth, reportYear)
		data.TotalVehicles = len(data.AgencyVehicles)
		for _, v := range data.AgencyVehicles {
			data.TotalMiles += v.TotalMiles
			if v.Status == "active" {
				data.TotalActive++
			}
		}
	case "school":
		data.SchoolBuses, err = getSchoolBusReports(reportMonth, reportYear)
		data.TotalVehicles = len(data.SchoolBuses)
		for _, v := range data.SchoolBuses {
			data.TotalMiles += v.TotalMiles
			if v.Status == "active" {
				data.TotalActive++
			}
		}
	case "program":
		data.ProgramStaff, err = getProgramStaffReports(reportMonth, reportYear)
	default:
		// Get all types
		data.AgencyVehicles, _ = getAgencyVehicleReports(reportMonth, reportYear)
		data.SchoolBuses, _ = getSchoolBusReports(reportMonth, reportYear)
		data.ProgramStaff, _ = getProgramStaffReports(reportMonth, reportYear)
		
		// Calculate totals
		data.TotalVehicles = len(data.AgencyVehicles) + len(data.SchoolBuses)
		for _, v := range data.AgencyVehicles {
			data.TotalMiles += v.TotalMiles
			if v.Status == "active" {
				data.TotalActive++
			}
		}
		for _, v := range data.SchoolBuses {
			data.TotalMiles += v.TotalMiles
			if v.Status == "active" {
				data.TotalActive++
			}
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
		Data             MileageReportData
		AvailableReports []string
		CSRFToken        string
	}{
		User:             user,
		ReportMonth:      reportMonth,
		ReportYear:       reportYear,
		ReportType:       reportType,
		Data:             data,
		AvailableReports: availableReports,
		CSRFToken: getSessionCSRFToken(r),
	}
	
	renderTemplate(w, r, "mileage_reports.html", templateData)
}

// Get agency vehicle reports - Updated to use all_vehicle_mileage table
func getAgencyVehicleReports(month, year string) ([]AgencyVehicleRecord, error) {
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	
	yearInt, _ := strconv.Atoi(year)
	
	var records []AgencyVehicleRecord
	
	// First try the all_vehicle_mileage table
	err := db.Select(&records, `
		SELECT 
			report_month,
			report_year,
			vehicle_year,
			make_model,
			license_plate,
			vehicle_id,
			location,
			beginning_miles,
			ending_miles,
			total_miles,
			status,
			notes
		FROM all_vehicle_mileage
		WHERE report_month = $1 AND report_year = $2
		  AND vehicle_type = 'agency'
		ORDER BY vehicle_id
	`, month, yearInt)
	
	// If no records or table doesn't exist, try the legacy table
	if (err != nil || len(records) == 0) && err != sql.ErrNoRows {
		log.Printf("Falling back to agency_vehicles table: %v", err)
		err = db.Select(&records, `
			SELECT * FROM agency_vehicles 
			WHERE report_month = $1 AND report_year = $2 
			ORDER BY vehicle_id
		`, month, yearInt)
	}
	
	return records, err
}

// Get school bus reports - Updated to use all_vehicle_mileage table
func getSchoolBusReports(month, year string) ([]SchoolBusRecord, error) {
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	
	yearInt, _ := strconv.Atoi(year)
	
	var records []SchoolBusRecord
	
	// First try the all_vehicle_mileage table
	err := db.Select(&records, `
		SELECT 
			report_month,
			report_year,
			vehicle_year as bus_year,
			make_model as bus_make,
			license_plate,
			vehicle_id as bus_id,
			location,
			beginning_miles,
			ending_miles,
			total_miles,
			status,
			notes
		FROM all_vehicle_mileage
		WHERE report_month = $1 AND report_year = $2
		  AND vehicle_type = 'bus'
		ORDER BY vehicle_id
	`, month, yearInt)
	
	// If no records or table doesn't exist, try the legacy table
	if (err != nil || len(records) == 0) && err != sql.ErrNoRows {
		log.Printf("Falling back to school_buses table: %v", err)
		err = db.Select(&records, `
			SELECT * FROM school_buses 
			WHERE report_month = $1 AND report_year = $2 
			ORDER BY bus_id
		`, month, yearInt)
	}
	
	return records, err
}

// Alternative: Get all vehicles regardless of type
func getAllVehicleReports(month, year string) ([]AgencyVehicleRecord, error) {
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	
	yearInt, _ := strconv.Atoi(year)
	
	var records []AgencyVehicleRecord
	err := db.Select(&records, `
		SELECT 
			report_month,
			report_year,
			vehicle_year,
			make_model,
			license_plate,
			vehicle_id,
			location,
			beginning_miles,
			ending_miles,
			total_miles,
			status,
			notes
		FROM all_vehicle_mileage
		WHERE report_month = $1 AND report_year = $2
		ORDER BY vehicle_id
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
	
	// Get unique month/year combinations from all_vehicle_mileage and program_staff
	err := db.Select(&reports, `
		SELECT DISTINCT report_month || ' ' || report_year::text as report
		FROM (
			SELECT report_month, report_year FROM all_vehicle_mileage
			UNION
			SELECT report_month, report_year FROM program_staff
		) combined
		ORDER BY report DESC
	`)
	
	// If that fails, try the old tables as fallback
	if err != nil || len(reports) == 0 {
		err = db.Select(&reports, `
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
	}
	
	return reports, err
}

// getDriverLogsByDateRange gets driver logs within a date range
func getDriverLogsByDateRange(driver, startDate, endDate string) ([]DriverLog, error) {
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	
	var logs []DriverLog
	err := db.Select(&logs, `
		SELECT id, driver, bus_id, route_id, date::text, period, 
		       departure_time::text, arrival_time::text, mileage, attendance::text
		FROM driver_logs
		WHERE driver = $1 AND date BETWEEN $2 AND $3
		ORDER BY date DESC, period DESC
	`, driver, startDate, endDate)
	
	return logs, err
}

// loadDriverLogsForDriver loads recent logs for a specific driver
func loadDriverLogsForDriver(driver string, limit int) ([]DriverLog, error) {
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	
	var logs []DriverLog
	err := db.Select(&logs, `
		SELECT id, driver, bus_id, route_id, date::text, period, 
		       departure_time::text, arrival_time::text, mileage, attendance::text
		FROM driver_logs
		WHERE driver = $1
		ORDER BY date DESC, period DESC
		LIMIT $2
	`, driver, limit)
	
	return logs, err
}

// loadDriverLogsFromDB loads all driver logs (limited to prevent overload)
func loadDriverLogsFromDB() ([]DriverLog, error) {
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	
	var logs []DriverLog
	err := db.Select(&logs, `
		SELECT id, driver, bus_id, route_id, date::text, period, 
		       departure_time::text, arrival_time::text, mileage, attendance::text
		FROM driver_logs
		ORDER BY date DESC, period DESC
		LIMIT 100
	`)
	
	return logs, err
}
func loadUsers() []User {
	if db == nil {
		return []User{}
	}
	
	var users []User
	err := db.Select(&users, `
		SELECT username, role, status, registration_date::text 
		FROM users 
		ORDER BY username
	`)
	
	if err != nil {
		log.Printf("Error loading users: %v", err)
		return []User{}
	}
	
	return users
}

// loadBuses loads all buses from the database
func loadBuses() []*Bus {
	if db == nil {
		return []*Bus{}
	}
	
	var buses []*Bus
	err := db.Select(&buses, `
		SELECT bus_id, status, model, capacity, oil_status, tire_status, maintenance_notes
		FROM buses
		ORDER BY bus_id
	`)
	
	if err != nil {
		log.Printf("Error loading buses: %v", err)
		return []*Bus{}
	}
	
	return buses
}

// loadRoutes loads all routes from the database
func loadRoutes() ([]Route, error) {
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	
	var routes []Route
	err := db.Select(&routes, `
		SELECT route_id, route_name, description
		FROM routes
		ORDER BY route_name
	`)
	
	return routes, err
}

// loadActivities loads recent activities
func loadActivities() ([]Activity, error) {
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	
	var activities []Activity
	err := db.Select(&activities, `
		SELECT id, date::text, driver, trip_name, attendance, miles, notes
		FROM activities
		ORDER BY date DESC
		LIMIT 50
	`)
	
	return activities, err
}

// loadRouteAssignments loads all route assignments
func loadRouteAssignments() ([]RouteAssignment, error) {
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	
	var assignments []RouteAssignment
	err := db.Select(&assignments, `
		SELECT driver, bus_id, route_id, route_name, assigned_date::text
		FROM route_assignments
		ORDER BY driver
	`)
	
	return assignments, err
}
func getComprehensiveActivityLog(startDate, endDate string) ([]map[string]interface{}, error) {
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	
	query := `
		SELECT * FROM (
			-- Driver logs
			SELECT 
				dl.date::text as activity_date,
				dl.driver as user_name,
				'Route ' || dl.period as activity_type,
				r.route_name as description,
				dl.mileage as miles,
				jsonb_array_length(dl.attendance) as count,
				'driver_log' as source
			FROM driver_logs dl
			LEFT JOIN routes r ON dl.route_id = r.route_id
			WHERE dl.date BETWEEN $1 AND $2
			
			UNION ALL
			
			-- Activities
			SELECT 
				a.date::text as activity_date,
				a.driver as user_name,
				'Special Trip' as activity_type,
				a.trip_name as description,
				a.miles,
				a.attendance as count,
				'activity' as source
			FROM activities a
			WHERE a.date BETWEEN $1 AND $2
			
			UNION ALL
			
			-- Maintenance logs
			SELECT 
				m.date::text as activity_date,
				'Maintenance' as user_name,
				m.category as activity_type,
				m.bus_id || ': ' || m.notes as description,
				0 as miles,
				0 as count,
				'maintenance' as source
			FROM bus_maintenance_logs m
			WHERE m.date BETWEEN $1 AND $2
		) combined
		ORDER BY activity_date DESC, user_name
	`
	
	rows, err := db.Query(query, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get comprehensive activity log: %w", err)
	}
	defer rows.Close()
	
	var activities []map[string]interface{}
	for rows.Next() {
		var activityDate, userName, activityType, description, source string
		var miles float64
		var count int
		
		err := rows.Scan(&activityDate, &userName, &activityType, &description, &miles, &count, &source)
		if err != nil {
			log.Printf("Error scanning activity: %v", err)
			continue
		}
		
		activity := map[string]interface{}{
			"date":        activityDate,
			"user":        userName,
			"type":        activityType,
			"description": description,
			"miles":       miles,
			"count":       count,
			"source":      source,
		}
		
		activities = append(activities, activity)
	}
	
// processECSEExcelFile processes an ECSE Excel file import
func processECSEExcelFile(file io.Reader, filename string) (int, error) {
	// Read the Excel file
	f, err := excelize.OpenReader(file)
	if err != nil {
		return 0, fmt.Errorf("failed to open Excel file: %w", err)
	}
	defer f.Close()

	// Get the first sheet
	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return 0, fmt.Errorf("no sheets found in Excel file")
	}

	rows, err := f.GetRows(sheets[0])
	if err != nil {
		return 0, fmt.Errorf("failed to get rows: %w", err)
	}

	if len(rows) < 2 {
		return 0, fmt.Errorf("file has no data rows")
	}

	// Skip header row and process data
	imported := 0
	for i := 1; i < len(rows); i++ {
		row := rows[i]
		if len(row) < 10 { // Ensure minimum columns
			continue
		}

		// Parse date of birth
		var dob *time.Time
		if row[3] != "" {
			if parsedDOB, err := time.Parse("1/2/2006", row[3]); err == nil {
				dob = &parsedDOB
			}
		}

		// Parse boolean values
		transportRequired := strings.ToLower(row[9]) == "yes" || strings.ToLower(row[9]) == "true"
		
		// Parse service minutes
		serviceMinutes := 0
		if row[8] != "" {
			if mins, err := strconv.Atoi(row[8]); err == nil {
				serviceMinutes = mins
			}
		}

		// Insert or update student record
		_, err := db.Exec(`
			INSERT INTO ecse_students (
				student_id, first_name, last_name, date_of_birth, grade,
				enrollment_status, iep_status, primary_disability, service_minutes,
				transportation_required, bus_route, parent_name, parent_phone,
				parent_email, address, city, state, zip_code, notes
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
			ON CONFLICT (student_id) DO UPDATE SET
				first_name = EXCLUDED.first_name,
				last_name = EXCLUDED.last_name,
				date_of_birth = EXCLUDED.date_of_birth,
				grade = EXCLUDED.grade,
				enrollment_status = EXCLUDED.enrollment_status,
				iep_status = EXCLUDED.iep_status,
				primary_disability = EXCLUDED.primary_disability,
				service_minutes = EXCLUDED.service_minutes,
				transportation_required = EXCLUDED.transportation_required,
				bus_route = EXCLUDED.bus_route,
				parent_name = EXCLUDED.parent_name,
				parent_phone = EXCLUDED.parent_phone,
				parent_email = EXCLUDED.parent_email,
				address = EXCLUDED.address,
				city = EXCLUDED.city,
				state = EXCLUDED.state,
				zip_code = EXCLUDED.zip_code,
				notes = EXCLUDED.notes,
				updated_at = CURRENT_TIMESTAMP
		`, row[0], row[1], row[2], dob, row[4], row[5], row[6], row[7], serviceMinutes,
			transportRequired, row[10], row[11], row[12], row[13], row[14], row[15], row[16], row[17], row[18])

		if err != nil {
			log.Printf("Error importing student %s: %v", row[0], err)
			continue
		}
		imported++
	}

	return imported, nil
}

// processEnhancedMileageExcelFile processes a mileage Excel file import
func processEnhancedMileageExcelFile(file io.Reader, filename string) (int, error) {
	// Read the Excel file
	f, err := excelize.OpenReader(file)
	if err != nil {
		return 0, fmt.Errorf("failed to open Excel file: %w", err)
	}
	defer f.Close()

	// Get all sheets
	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return 0, fmt.Errorf("no sheets found in Excel file")
	}

	totalImported := 0

	// Process each sheet
	for _, sheet := range sheets {
		rows, err := f.GetRows(sheet)
		if err != nil {
			log.Printf("Error reading sheet %s: %v", sheet, err)
			continue
		}

		if len(rows) < 2 {
			continue // Skip empty sheets
		}

		// Determine sheet type based on headers or sheet name
		vehicleType := "agency"
		if strings.Contains(strings.ToLower(sheet), "bus") {
			vehicleType = "bus"
		}

		// Skip header row and process data
		for i := 1; i < len(rows); i++ {
			row := rows[i]
			if len(row) < 12 { // Ensure minimum columns
				continue
			}

			// Parse numeric values
			reportYear, _ := strconv.Atoi(row[1])
			vehicleYear, _ := strconv.Atoi(row[2])
			beginningMiles, _ := strconv.Atoi(row[7])
			endingMiles, _ := strconv.Atoi(row[8])
			totalMiles, _ := strconv.Atoi(row[9])

			// Insert into all_vehicle_mileage table
			_, err := db.Exec(`
				INSERT INTO all_vehicle_mileage (
					report_month, report_year, vehicle_year, make_model, license_plate,
					vehicle_id, location, beginning_miles, ending_miles, total_miles,
					status, notes, vehicle_type
				) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
				ON CONFLICT (report_month, report_year, vehicle_id) DO UPDATE SET
					vehicle_year = EXCLUDED.vehicle_year,
					make_model = EXCLUDED.make_model,
					license_plate = EXCLUDED.license_plate,
					location = EXCLUDED.location,
					beginning_miles = EXCLUDED.beginning_miles,
					ending_miles = EXCLUDED.ending_miles,
					total_miles = EXCLUDED.total_miles,
					status = EXCLUDED.status,
					notes = EXCLUDED.notes,
					vehicle_type = EXCLUDED.vehicle_type,
					updated_at = CURRENT_TIMESTAMP
			`, row[0], reportYear, vehicleYear, row[3], row[4], row[5], row[6],
				beginningMiles, endingMiles, totalMiles, row[10], row[11], vehicleType)

			if err != nil {
				log.Printf("Error importing vehicle %s: %v", row[5], err)
				continue
			}
			totalImported++
		}
	}

	return totalImported, nil
}

// getUserFromSession gets the user from the current session
func getUserFromSession(r *http.Request) *User {
	cookie, err := r.Cookie(SessionCookieName)
	if err != nil {
		return nil
	}
	
	session, err := GetSecureSession(cookie.Value)
	if err != nil || session == nil {
		return nil
	}
	
	return &User{
		Username: session.Username,
		Role:     session.Role,
		Status:   "active",
	}
}

// getSessionCSRFToken gets the CSRF token from the session
func getSessionCSRFToken(r *http.Request) string {
	cookie, err := r.Cookie(SessionCookieName)
	if err != nil {
		return ""
	}
	
	session, err := GetSecureSession(cookie.Value)
	if err != nil || session == nil {
		return ""
	}
	
	return session.CSRFToken
}
