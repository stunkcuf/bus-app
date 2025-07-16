package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

var db *sqlx.DB

// InitDB initializes the database connection
func InitDB(dataSourceName string) error {
	var err error
	db, err = sqlx.Open("postgres", dataSourceName)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Test the connection
	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	// Run migrations
	if err := runMigrations(); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

// runMigrations runs database migrations
func runMigrations() error {
	migrations := []string{
		// Users table
		`CREATE TABLE IF NOT EXISTS users (
			username VARCHAR(50) PRIMARY KEY,
			password VARCHAR(255) NOT NULL,
			role VARCHAR(20) NOT NULL DEFAULT 'driver',
			status VARCHAR(20) NOT NULL DEFAULT 'pending',
			registration_date DATE DEFAULT CURRENT_DATE,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// Buses table with maintenance tracking
		`CREATE TABLE IF NOT EXISTS buses (
			bus_id VARCHAR(50) PRIMARY KEY,
			status VARCHAR(20) DEFAULT 'active',
			model VARCHAR(100),
			capacity INTEGER,
			oil_status VARCHAR(20) DEFAULT 'good',
			tire_status VARCHAR(20) DEFAULT 'good',
			maintenance_notes TEXT,
			current_mileage INTEGER DEFAULT 0,
			last_oil_change INTEGER DEFAULT 0,
			last_tire_service INTEGER DEFAULT 0,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// Vehicles table with maintenance tracking
		`CREATE TABLE IF NOT EXISTS vehicles (
			vehicle_id VARCHAR(50) PRIMARY KEY,
			model VARCHAR(100),
			description TEXT,
			year INTEGER,
			tire_size VARCHAR(50),
			license VARCHAR(50),
			oil_status VARCHAR(20) DEFAULT 'good',
			tire_status VARCHAR(20) DEFAULT 'good',
			status VARCHAR(20) DEFAULT 'active',
			maintenance_notes TEXT,
			serial_number VARCHAR(100),
			base VARCHAR(100),
			service_interval INTEGER DEFAULT 5000,
			current_mileage INTEGER DEFAULT 0,
			last_oil_change INTEGER DEFAULT 0,
			last_tire_service INTEGER DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// Bus maintenance logs
		`CREATE TABLE IF NOT EXISTS bus_maintenance_logs (
			id SERIAL PRIMARY KEY,
			bus_id VARCHAR(50) REFERENCES buses(bus_id),
			date DATE NOT NULL,
			category VARCHAR(50) NOT NULL,
			notes TEXT,
			mileage INTEGER,
			cost DECIMAL(10,2),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// Vehicle maintenance logs
		`CREATE TABLE IF NOT EXISTS vehicle_maintenance_logs (
			id SERIAL PRIMARY KEY,
			vehicle_id VARCHAR(50) REFERENCES vehicles(vehicle_id),
			date DATE NOT NULL,
			category VARCHAR(50) NOT NULL,
			notes TEXT,
			mileage INTEGER,
			cost DECIMAL(10,2),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// Routes table
		`CREATE TABLE IF NOT EXISTS routes (
			route_id VARCHAR(50) PRIMARY KEY,
			route_name VARCHAR(100) NOT NULL,
			description TEXT,
			positions JSONB DEFAULT '[]'::jsonb,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// Route assignments
		`CREATE TABLE IF NOT EXISTS route_assignments (
			driver VARCHAR(50) REFERENCES users(username),
			bus_id VARCHAR(50) REFERENCES buses(bus_id),
			route_id VARCHAR(50) REFERENCES routes(route_id),
			assigned_date DATE DEFAULT CURRENT_DATE,
			PRIMARY KEY (driver, bus_id, route_id)
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
			position_number INTEGER,
			route_id VARCHAR(50),
			driver VARCHAR(50),
			active BOOLEAN DEFAULT true,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// Driver logs
		`CREATE TABLE IF NOT EXISTS driver_logs (
			id SERIAL PRIMARY KEY,
			driver VARCHAR(50) REFERENCES users(username),
			bus_id VARCHAR(50),
			route_id VARCHAR(50),
			date DATE NOT NULL,
			period VARCHAR(20),
			departure_time TIME,
			arrival_time TIME,
			begin_mileage DECIMAL(10,2),
			end_mileage DECIMAL(10,2),
			attendance JSONB DEFAULT '[]'::jsonb,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// ECSE students
		`CREATE TABLE IF NOT EXISTS ecse_students (
			student_id VARCHAR(50) PRIMARY KEY,
			first_name VARCHAR(100),
			last_name VARCHAR(100),
			date_of_birth DATE,
			grade VARCHAR(20),
			enrollment_status VARCHAR(50),
			iep_status VARCHAR(50),
			primary_disability VARCHAR(100),
			service_minutes INTEGER,
			transportation_required BOOLEAN DEFAULT false,
			bus_route VARCHAR(50),
			parent_name VARCHAR(100),
			parent_phone VARCHAR(20),
			parent_email VARCHAR(100),
			address TEXT,
			last_assessment_date DATE,
			next_assessment_date DATE,
			notes TEXT,
			active BOOLEAN DEFAULT true,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// ECSE services
		`CREATE TABLE IF NOT EXISTS ecse_services (
			id SERIAL PRIMARY KEY,
			student_id VARCHAR(50) REFERENCES ecse_students(student_id),
			service_type VARCHAR(50),
			frequency VARCHAR(50),
			duration INTEGER,
			provider VARCHAR(100),
			start_date DATE,
			end_date DATE,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// Mileage reports
		`CREATE TABLE IF NOT EXISTS mileage_reports (
			id SERIAL PRIMARY KEY,
			unit VARCHAR(50),
			vehicle_no VARCHAR(50),
			driver VARCHAR(100),
			month VARCHAR(20),
			year INTEGER,
			begin_miles INTEGER,
			end_miles INTEGER,
			total_miles INTEGER,
			daily_miles TEXT,
			utilization DECIMAL(5,2),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// Add maintenance tracking columns if they don't exist
		`DO $$ 
		BEGIN 
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
				WHERE table_name = 'buses' AND column_name = 'current_mileage') THEN
				ALTER TABLE buses ADD COLUMN current_mileage INTEGER DEFAULT 0;
			END IF;
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
				WHERE table_name = 'buses' AND column_name = 'last_oil_change') THEN
				ALTER TABLE buses ADD COLUMN last_oil_change INTEGER DEFAULT 0;
			END IF;
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
				WHERE table_name = 'buses' AND column_name = 'last_tire_service') THEN
				ALTER TABLE buses ADD COLUMN last_tire_service INTEGER DEFAULT 0;
			END IF;
		END $$;`,

		`DO $$ 
		BEGIN 
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
				WHERE table_name = 'vehicles' AND column_name = 'current_mileage') THEN
				ALTER TABLE vehicles ADD COLUMN current_mileage INTEGER DEFAULT 0;
			END IF;
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
				WHERE table_name = 'vehicles' AND column_name = 'last_oil_change') THEN
				ALTER TABLE vehicles ADD COLUMN last_oil_change INTEGER DEFAULT 0;
			END IF;
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
				WHERE table_name = 'vehicles' AND column_name = 'last_tire_service') THEN
				ALTER TABLE vehicles ADD COLUMN last_tire_service INTEGER DEFAULT 0;
			END IF;
		END $$;`,

		// Create indexes for better performance
		`CREATE INDEX IF NOT EXISTS idx_bus_maintenance_logs_bus_id ON bus_maintenance_logs(bus_id);`,
		`CREATE INDEX IF NOT EXISTS idx_vehicle_maintenance_logs_vehicle_id ON vehicle_maintenance_logs(vehicle_id);`,
		`CREATE INDEX IF NOT EXISTS idx_driver_logs_driver ON driver_logs(driver);`,
		`CREATE INDEX IF NOT EXISTS idx_driver_logs_date ON driver_logs(date);`,
		`CREATE INDEX IF NOT EXISTS idx_students_route_id ON students(route_id);`,
		`CREATE INDEX IF NOT EXISTS idx_route_assignments_driver ON route_assignments(driver);`,
	}

	for _, migration := range migrations {
		if _, err := db.Exec(migration); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}

	return nil
}

// Database query functions

// GetMaintenanceLogsForVehicle retrieves all maintenance logs for a vehicle (bus or regular vehicle)
func getMaintenanceLogsForVehicle(vehicleID string) ([]CombinedMaintenanceLog, error) {
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	query := `
		SELECT 
			id,
			bus_id as vehicle_id,
			'bus' as vehicle_type,
			date,
			category,
			notes,
			mileage,
			cost,
			created_at
		FROM bus_maintenance_logs
		WHERE bus_id = $1
		
		UNION ALL
		
		SELECT 
			id,
			vehicle_id,
			'vehicle' as vehicle_type,
			date,
			category,
			notes,
			mileage,
			cost,
			created_at
		FROM vehicle_maintenance_logs
		WHERE vehicle_id = $1
		
		ORDER BY date DESC, created_at DESC
	`

	var logs []CombinedMaintenanceLog
	err := db.Select(&logs, query, vehicleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get maintenance logs: %w", err)
	}

	return logs, nil
}

// GetVehicleCurrentMileage gets the current mileage for any vehicle
func getVehicleCurrentMileage(vehicleID string) (int, error) {
	if db == nil {
		return 0, fmt.Errorf("database not initialized")
	}

	var mileage int
	
	// Try buses table first
	err := db.Get(&mileage, "SELECT current_mileage FROM buses WHERE bus_id = $1", vehicleID)
	if err == nil {
		return mileage, nil
	}
	
	// Try vehicles table
	err = db.Get(&mileage, "SELECT current_mileage FROM vehicles WHERE vehicle_id = $1", vehicleID)
	if err != nil {
		return 0, fmt.Errorf("failed to get vehicle mileage: %w", err)
	}
	
	return mileage, nil
}

// UpdateVehicleMileage updates the current mileage for a vehicle
func updateVehicleMileage(vehicleID string, newMileage int) error {
	if db == nil {
		return fmt.Errorf("database not initialized")
	}

	// Try updating bus first
	result, err := db.Exec("UPDATE buses SET current_mileage = $1 WHERE bus_id = $2", newMileage, vehicleID)
	if err == nil {
		rowsAffected, _ := result.RowsAffected()
		if rowsAffected > 0 {
			return nil
		}
	}

	// Try updating vehicle
	result, err = db.Exec("UPDATE vehicles SET current_mileage = $1 WHERE vehicle_id = $2", newMileage, vehicleID)
	if err != nil {
		return fmt.Errorf("failed to update vehicle mileage: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("vehicle not found")
	}

	return nil
}

// UpdateVehicleMaintenanceStatus updates oil and tire status based on mileage
func updateVehicleMaintenanceStatus(vehicleID string, oilStatus, tireStatus string) error {
	if db == nil {
		return fmt.Errorf("database not initialized")
	}

	// Try updating bus first
	result, err := db.Exec(
		"UPDATE buses SET oil_status = $1, tire_status = $2, updated_at = CURRENT_TIMESTAMP WHERE bus_id = $3",
		oilStatus, tireStatus, vehicleID,
	)
	if err == nil {
		rowsAffected, _ := result.RowsAffected()
		if rowsAffected > 0 {
			return nil
		}
	}

	// Try updating vehicle
	result, err = db.Exec(
		"UPDATE vehicles SET oil_status = $1, tire_status = $2, updated_at = CURRENT_TIMESTAMP WHERE vehicle_id = $3",
		oilStatus, tireStatus, vehicleID,
	)
	if err != nil {
		return fmt.Errorf("failed to update vehicle status: %w", err)
	}

	return nil
}

// GetVehicleMaintenanceInfo gets maintenance information for a vehicle
func getVehicleMaintenanceInfo(vehicleID string) (currentMileage, lastOilChange, lastTireService int, err error) {
	if db == nil {
		return 0, 0, 0, fmt.Errorf("database not initialized")
	}

	// Try buses table first
	err = db.QueryRow(
		"SELECT current_mileage, last_oil_change, last_tire_service FROM buses WHERE bus_id = $1",
		vehicleID,
	).Scan(&currentMileage, &lastOilChange, &lastTireService)
	
	if err == nil {
		return currentMileage, lastOilChange, lastTireService, nil
	}

	// Try vehicles table
	err = db.QueryRow(
		"SELECT current_mileage, last_oil_change, last_tire_service FROM vehicles WHERE vehicle_id = $1",
		vehicleID,
	).Scan(&currentMileage, &lastOilChange, &lastTireService)
	
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to get vehicle maintenance info: %w", err)
	}

	return currentMileage, lastOilChange, lastTireService, nil
}

// UpdateLastServiceMileage updates the last service mileage for a specific maintenance type
func updateLastServiceMileage(vehicleID string, serviceType string, mileage int) error {
	if db == nil {
		return fmt.Errorf("database not initialized")
	}

	var query string
	switch serviceType {
	case "oil_change":
		query = "UPDATE buses SET last_oil_change = $1 WHERE bus_id = $2"
	case "tire_service":
		query = "UPDATE buses SET last_tire_service = $1 WHERE bus_id = $2"
	default:
		return fmt.Errorf("unknown service type: %s", serviceType)
	}

	// Try updating bus first
	result, err := db.Exec(query, mileage, vehicleID)
	if err == nil {
		rowsAffected, _ := result.RowsAffected()
		if rowsAffected > 0 {
			return nil
		}
	}

	// Try updating vehicle
	switch serviceType {
	case "oil_change":
		query = "UPDATE vehicles SET last_oil_change = $1 WHERE vehicle_id = $2"
	case "tire_service":
		query = "UPDATE vehicles SET last_tire_service = $1 WHERE vehicle_id = $2"
	}

	_, err = db.Exec(query, mileage, vehicleID)
	if err != nil {
		return fmt.Errorf("failed to update last service mileage: %w", err)
	}

	return nil
}

// GetLastMileageForVehicle gets the last recorded mileage for a vehicle from driver logs
func getLastMileageForVehicle(vehicleID string) (float64, error) {
	if db == nil {
		return 0, fmt.Errorf("database not initialized")
	}

	var lastMileage sql.NullFloat64
	query := `
		SELECT MAX(end_mileage) 
		FROM driver_logs 
		WHERE bus_id = $1 AND end_mileage IS NOT NULL
	`
	
	err := db.Get(&lastMileage, query, vehicleID)
	if err != nil {
		return 0, fmt.Errorf("failed to get last mileage: %w", err)
	}

	if !lastMileage.Valid {
		return 0, nil
	}

	return lastMileage.Float64, nil
}
