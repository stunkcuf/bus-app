package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

var db *sqlx.DB

// maskConnectionString masks sensitive parts of connection string for logging
func maskConnectionString(connStr string) string {
	// Simple masking - just show the host
	if strings.Contains(connStr, "@") {
		parts := strings.Split(connStr, "@")
		if len(parts) > 1 {
			hostPart := parts[1]
			if strings.Contains(hostPart, "/") {
				hostPart = strings.Split(hostPart, "/")[0]
			}
			return fmt.Sprintf("postgres://****:****@%s/****", hostPart)
		}
	}
	return "postgres://****:****@****:****/****"
}

// InitDB initializes the database connection
func InitDB(dataSourceName string) error {
	log.Printf("Initializing database connection...")
	log.Printf("Database URL format check: %s", maskConnectionString(dataSourceName))

	var err error
	db, err = sqlx.Open("postgres", dataSourceName)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Apply optimized connection pool configuration
	poolConfig := LoadPoolConfigFromEnv()
	if err := ConfigureDBPool(db, poolConfig); err != nil {
		log.Printf("Warning: Failed to configure optimal pool settings: %v", err)
		// Fallback to default settings
		db.SetMaxOpenConns(25)
		db.SetMaxIdleConns(10)
		db.SetConnMaxLifetime(15 * time.Minute)
		db.SetConnMaxIdleTime(5 * time.Minute)
	}

	// Test the connection
	log.Println("Testing database connection...")
	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}
	log.Println("Database connection successful!")

	// Run migrations
	log.Println("Running database migrations...")
	if err := runMigrations(); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	// Ensure admin user exists
	log.Println("Ensuring admin user exists...")
	if err := ensureAdminUser(); err != nil {
		log.Printf("Warning: Failed to ensure admin user: %v", err)
		// Don't fail startup, but log the warning
	}

	// Create performance indexes
	if err := createPerformanceIndexes(); err != nil {
		log.Printf("Warning: Failed to create some performance indexes: %v", err)
		// Don't fail startup for index creation errors
	}

	// Fix database sync issues
	log.Println("Fixing database sync issues...")
	if err := FixDatabaseSyncIssues(); err != nil {
		log.Printf("Warning: Failed to fix some database sync issues: %v", err)
		// Don't fail startup, but log the warning
	}

	// Clean up invalid data
	log.Println("Cleaning up invalid data...")
	if err := CleanupInvalidData(); err != nil {
		log.Printf("Warning: Failed to clean up some invalid data: %v", err)
		// Don't fail startup, but log the warning
	}

	// Fix notifications table
	log.Println("Fixing notifications table...")
	if err := FixNotificationsTable(); err != nil {
		log.Printf("Warning: Failed to fix notifications table: %v", err)
		// Don't fail startup, but log the warning
	}

	// Fix users table
	log.Println("Fixing users table...")
	if err := FixUsersTable(); err != nil {
		log.Printf("Warning: Failed to fix users table: %v", err)
		// Don't fail startup, but log the warning
	}

	// Fix routes display
	log.Println("Fixing routes display...")
	if err := FixRoutesDisplay(); err != nil {
		log.Printf("Warning: Failed to fix routes display: %v", err)
		// Don't fail startup, but log the warning
	}

	// Fix route assignments
	if err := FixRouteAssignments(); err != nil {
		log.Printf("Warning: Failed to fix route assignments: %v", err)
		// Don't fail startup, but log the warning
	}

	log.Println("Database initialization complete!")
	return nil
}

// createPerformanceIndexes creates database indexes for optimal performance
func createPerformanceIndexes() error {
	indexes := []struct {
		name  string
		query string
	}{
		// Monthly Mileage Reports Indexes
		{
			name: "monthly_mileage_reports_year_month",
			query: `CREATE INDEX IF NOT EXISTS idx_monthly_mileage_reports_year_month 
			 ON monthly_mileage_reports(report_year DESC, report_month)`,
		},
		{
			name: "monthly_mileage_reports_bus_id",
			query: `CREATE INDEX IF NOT EXISTS idx_monthly_mileage_reports_bus_id 
			 ON monthly_mileage_reports(bus_id)`,
		},
		// Maintenance Records Indexes - FIXED: use correct columns
		{
			name: "maintenance_records_vehicle_id",
			query: `CREATE INDEX IF NOT EXISTS idx_maintenance_records_vehicle_id 
			 ON maintenance_records(vehicle_id)`,
		},
		{
			name: "maintenance_records_service_date",
			query: `CREATE INDEX IF NOT EXISTS idx_maintenance_records_service_date 
			 ON maintenance_records(service_date DESC) WHERE service_date IS NOT NULL`,
		},
		{
			name: "maintenance_records_date",
			query: `CREATE INDEX IF NOT EXISTS idx_maintenance_records_date 
			 ON maintenance_records(date DESC) WHERE date IS NOT NULL`,
		},
		// Fleet Vehicles Indexes - COMMENTED OUT: migrating to vehicles table
		// {
		// 	name: "fleet_vehicles_make_model",
		// 	query: `CREATE INDEX IF NOT EXISTS idx_fleet_vehicles_make_model 
		// 	 ON fleet_vehicles(make, model)`,
		// },
		// Buses table indexes
		{
			name: "buses_status",
			query: `CREATE INDEX IF NOT EXISTS idx_buses_status 
			 ON buses(status)`,
		},
		// Vehicles table indexes
		{
			name: "vehicles_status",
			query: `CREATE INDEX IF NOT EXISTS idx_vehicles_status 
			 ON vehicles(status) WHERE status IS NOT NULL`,
		},
		// Service Records Indexes - FIXED: remove non-existent column
		{
			name: "service_records_maintenance_date",
			query: `CREATE INDEX IF NOT EXISTS idx_service_records_maintenance_date 
			 ON service_records(maintenance_date DESC) WHERE maintenance_date IS NOT NULL`,
		},
		// Composite indexes for common queries
		{
			name: "maintenance_records_vehicle_service_date",
			query: `CREATE INDEX IF NOT EXISTS idx_maintenance_records_vehicle_service_date 
			 ON maintenance_records(vehicle_id, service_date DESC) WHERE service_date IS NOT NULL`,
		},
		{
			name: "monthly_reports_bus_year_month",
			query: `CREATE INDEX IF NOT EXISTS idx_monthly_reports_bus_year_month 
			 ON monthly_mileage_reports(bus_id, report_year DESC, report_month)`,
		},
		// Add indexes for fuel records
		{
			name: "fuel_records_vehicle_date",
			query: `CREATE INDEX IF NOT EXISTS idx_fuel_records_vehicle_date 
			 ON fuel_records(vehicle_id, date DESC)`,
		},
		// Additional performance indexes
		{
			name: "students_route_id",
			query: `CREATE INDEX IF NOT EXISTS idx_students_route_id 
			 ON students(route_id)`,
		},
		{
			name: "students_driver",
			query: `CREATE INDEX IF NOT EXISTS idx_students_driver 
			 ON students(driver)`,
		},
		{
			name: "route_assignments_driver",
			query: `CREATE INDEX IF NOT EXISTS idx_route_assignments_driver 
			 ON route_assignments(driver)`,
		},
		{
			name: "route_assignments_bus_id",
			query: `CREATE INDEX IF NOT EXISTS idx_route_assignments_bus_id 
			 ON route_assignments(bus_id)`,
		},
		{
			name: "driver_logs_driver",
			query: `CREATE INDEX IF NOT EXISTS idx_driver_logs_driver 
			 ON driver_logs(driver)`,
		},
		{
			name: "driver_logs_date",
			query: `CREATE INDEX IF NOT EXISTS idx_driver_logs_date 
			 ON driver_logs(date)`,
		},
	}

	for _, idx := range indexes {
		log.Printf("Creating index: %s", idx.name)
		if _, err := db.Exec(idx.query); err != nil {
			// Check if it's a column doesn't exist error
			errStr := err.Error()
			if strings.Contains(errStr, "does not exist") {
				log.Printf("Warning: Skipping index %s - column doesn't exist: %v", idx.name, err)
				continue
			}
			if strings.Contains(errStr, "already exists") {
				log.Printf("Index %s already exists, skipping", idx.name)
				continue
			}
			log.Printf("Warning: Failed to create index %s: %v", idx.name, err)
			// Continue with other indexes
		}
	}

	log.Printf("Performance indexes creation completed")
	return nil
}
// Add this function to your database.go file after the runMigrations function

func fixDatabaseCompatibilityIssues() error {
	log.Println("Fixing database compatibility issues...")
	
	// Fix 1: Add log_date column to driver_logs for compatibility
	_, err := db.Exec(`
		DO $$ 
		BEGIN
			-- Add log_date column if it doesn't exist
			IF NOT EXISTS (
				SELECT 1 FROM information_schema.columns 
				WHERE table_name = 'driver_logs' AND column_name = 'log_date'
			) THEN
				ALTER TABLE driver_logs ADD COLUMN log_date DATE;
				-- Copy data from date column
				UPDATE driver_logs SET log_date = date WHERE log_date IS NULL;
				-- Set default
				ALTER TABLE driver_logs ALTER COLUMN log_date SET DEFAULT CURRENT_DATE;
				-- Create index
				CREATE INDEX IF NOT EXISTS idx_driver_logs_log_date ON driver_logs(log_date);
				RAISE NOTICE 'Added log_date column to driver_logs';
			END IF;
			
			-- Fix 2: Set default for metrics metadata column
			ALTER TABLE metrics ALTER COLUMN metadata SET DEFAULT '{}';
		EXCEPTION
			WHEN others THEN
				RAISE NOTICE 'Some compatibility fixes may have failed: %', SQLERRM;
		END $$;
	`)
	
	if err != nil {
		log.Printf("Warning: Failed to apply some compatibility fixes: %v", err)
		// Don't fail startup
	}
	
	return nil
}

// Call this function in InitDB after runMigrations:
// Add this after line 65 in InitDB function:
// 
// // Fix compatibility issues
// log.Println("Fixing compatibility issues...")
// if err := fixDatabaseCompatibilityIssues(); err != nil {
//     log.Printf("Warning: Failed to fix compatibility issues: %v", err)
// }

// ensureAdminUser ensures that an admin user exists in the database
func ensureAdminUser() error {
	// Get admin credentials from environment or use secure defaults
	username := os.Getenv("ADMIN_USERNAME")
	if username == "" {
		username = "admin"
	}
	
	password := os.Getenv("ADMIN_PASSWORD")
	if password == "" {
		// Generate a random password if not provided
		log.Println("Warning: ADMIN_PASSWORD not set. Admin user creation skipped.")
		log.Println("Set ADMIN_PASSWORD environment variable and restart to create admin user.")
		return nil
	}
	
	// Hash the password
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}
	hashedPassword := string(hashedBytes)
	
	// Check if admin user exists
	var exists bool
	err = db.Get(&exists, "SELECT EXISTS(SELECT 1 FROM users WHERE username = $1)", username)
	if err != nil {
		return fmt.Errorf("failed to check admin existence: %w", err)
	}
	
	if exists {
		// Update password and ensure active status
		_, err = db.Exec(`
			UPDATE users 
			SET password = $1, role = 'manager', status = 'active'
			WHERE username = $2
		`, hashedPassword, username)
		if err != nil {
			return fmt.Errorf("failed to update admin user: %w", err)
		}
		log.Printf("Admin user updated successfully")
	} else {
		// Create admin user
		_, err = db.Exec(`
			INSERT INTO users (username, password, role, status, registration_date, created_at)
			VALUES ($1, $2, 'manager', 'active', CURRENT_DATE, CURRENT_TIMESTAMP)
		`, username, hashedPassword)
		if err != nil {
			return fmt.Errorf("failed to create admin user: %w", err)
		}
		log.Printf("Admin user created successfully")
	}
	
	log.Printf("✅ Admin user ensured: username='%s'", username)
	return nil
}

// runMigrations runs database migrations
func runMigrations() error {
	log.Println("Starting database migrations...")

	migrations := []string{
		// Create users table
		`CREATE TABLE IF NOT EXISTS users (
			username VARCHAR(50) PRIMARY KEY,
			password VARCHAR(255) NOT NULL,
			role VARCHAR(20) NOT NULL CHECK (role IN ('manager', 'driver')),
			status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('active', 'pending')),
			registration_date DATE NOT NULL DEFAULT CURRENT_DATE,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// Create sessions table
		`CREATE TABLE IF NOT EXISTS sessions (
			token VARCHAR(255) PRIMARY KEY,
			username VARCHAR(50) NOT NULL REFERENCES users(username) ON DELETE CASCADE,
			csrf_token VARCHAR(255) NOT NULL,
			expires_at TIMESTAMP NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// Create password reset tokens table
		`CREATE TABLE IF NOT EXISTS password_reset_tokens (
			username VARCHAR(50) PRIMARY KEY REFERENCES users(username) ON DELETE CASCADE,
			token VARCHAR(255) NOT NULL,
			expires_at TIMESTAMP NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// Create buses table
		`CREATE TABLE IF NOT EXISTS buses (
			bus_id VARCHAR(50) PRIMARY KEY,
			status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'maintenance', 'out_of_service')),
			model VARCHAR(100),
			capacity INTEGER,
			oil_status VARCHAR(20) DEFAULT 'good' CHECK (oil_status IN ('good', 'due_soon', 'overdue')),
			tire_status VARCHAR(20) DEFAULT 'good' CHECK (tire_status IN ('good', 'due_soon', 'overdue')),
			maintenance_notes TEXT,
			current_mileage INTEGER DEFAULT 0,
			last_oil_change INTEGER DEFAULT 0,
			last_tire_service INTEGER DEFAULT 0,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// Create vehicles table
		`CREATE TABLE IF NOT EXISTS vehicles (
			vehicle_id VARCHAR(50) PRIMARY KEY,
			model VARCHAR(100),
			description TEXT,
			year VARCHAR(10),
			tire_size VARCHAR(50),
			license VARCHAR(50),
			oil_status VARCHAR(20) DEFAULT 'good' CHECK (oil_status IN ('good', 'due_soon', 'overdue')),
			tire_status VARCHAR(20) DEFAULT 'good' CHECK (tire_status IN ('good', 'due_soon', 'overdue')),
			status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'maintenance', 'out_of_service')),
			maintenance_notes TEXT,
			serial_number VARCHAR(100),
			base VARCHAR(100),
			service_interval INTEGER,
			current_mileage INTEGER DEFAULT 0,
			last_oil_change INTEGER DEFAULT 0,
			last_tire_service INTEGER DEFAULT 0,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// Create consolidated maintenance_records table
		`CREATE TABLE IF NOT EXISTS maintenance_records (
			id SERIAL PRIMARY KEY,
			vehicle_number INTEGER,
			service_date DATE,
			mileage INTEGER,
			cost NUMERIC,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			date DATE,
			po_number VARCHAR(50),
			vehicle_id VARCHAR(20),
			work_description TEXT,
			raw_data TEXT
		)`,

		// Create routes table
		`CREATE TABLE IF NOT EXISTS routes (
			route_id VARCHAR(50) PRIMARY KEY,
			route_name VARCHAR(100) NOT NULL,
			description TEXT,
			positions JSONB DEFAULT '[]',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// Create students table
		`CREATE TABLE IF NOT EXISTS students (
			student_id VARCHAR(50) PRIMARY KEY,
			name VARCHAR(100) NOT NULL,
			locations JSONB DEFAULT '[]',
			phone_number VARCHAR(20),
			alt_phone_number VARCHAR(20),
			guardian VARCHAR(100),
			pickup_time TIME,
			dropoff_time TIME,
			position_number INTEGER,
			route_id VARCHAR(50) REFERENCES routes(route_id) ON DELETE SET NULL,
			driver VARCHAR(50) REFERENCES users(username) ON DELETE SET NULL,
			active BOOLEAN DEFAULT true,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// Create route_assignments table
		`CREATE TABLE IF NOT EXISTS route_assignments (
			id SERIAL PRIMARY KEY,
			driver VARCHAR(50) NOT NULL REFERENCES users(username) ON DELETE CASCADE,
			bus_id VARCHAR(50) NOT NULL REFERENCES buses(bus_id) ON DELETE CASCADE,
			route_id VARCHAR(50) NOT NULL REFERENCES routes(route_id) ON DELETE CASCADE,
			assigned_date DATE NOT NULL DEFAULT CURRENT_DATE,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(driver, route_id),
			UNIQUE(bus_id, route_id)
		)`,

		// Create driver_logs table
		`CREATE TABLE IF NOT EXISTS driver_logs (
			id SERIAL PRIMARY KEY,
			driver VARCHAR(50) NOT NULL REFERENCES users(username) ON DELETE CASCADE,
			bus_id VARCHAR(50) NOT NULL REFERENCES buses(bus_id) ON DELETE CASCADE,
			route_id VARCHAR(50) NOT NULL REFERENCES routes(route_id) ON DELETE CASCADE,
			date DATE NOT NULL,
			period VARCHAR(20) NOT NULL CHECK (period IN ('morning', 'afternoon')),
			departure_time TIME,
			arrival_time TIME,
			start_mileage DOUBLE PRECISION,
			end_mileage DOUBLE PRECISION,
			attendance JSONB DEFAULT '[]',
			notes TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// Create ecse_students table
		`CREATE TABLE IF NOT EXISTS ecse_students (
			student_id VARCHAR(50) PRIMARY KEY,
			first_name VARCHAR(100) NOT NULL,
			last_name VARCHAR(100) NOT NULL,
			date_of_birth DATE,
			grade VARCHAR(20),
			enrollment_status VARCHAR(50),
			iep_status VARCHAR(50),
			primary_disability VARCHAR(100),
			service_minutes INTEGER,
			transportation_required BOOLEAN DEFAULT false,
			bus_route VARCHAR(100),
			parent_name VARCHAR(100),
			parent_phone VARCHAR(20),
			parent_email VARCHAR(100),
			city VARCHAR(100),
			state VARCHAR(50),
			zip_code VARCHAR(20),
			notes TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// Create ecse_services table
		`CREATE TABLE IF NOT EXISTS ecse_services (
			id SERIAL PRIMARY KEY,
			student_id VARCHAR(50) NOT NULL REFERENCES ecse_students(student_id) ON DELETE CASCADE,
			service_type VARCHAR(50) NOT NULL CHECK (service_type IN ('speech', 'OT', 'PT', 'behavioral', 'other')),
			frequency VARCHAR(100),
			duration INTEGER,
			provider VARCHAR(100),
			start_date DATE,
			end_date DATE,
			goals TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// Create ecse_assessments table
		`CREATE TABLE IF NOT EXISTS ecse_assessments (
			id SERIAL PRIMARY KEY,
			student_id VARCHAR(50) NOT NULL REFERENCES ecse_students(student_id) ON DELETE CASCADE,
			assessment_date DATE NOT NULL,
			assessment_type VARCHAR(100),
			results TEXT,
			evaluator VARCHAR(100),
			next_assessment_date DATE,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// Create ecse_attendance table
		`CREATE TABLE IF NOT EXISTS ecse_attendance (
			id SERIAL PRIMARY KEY,
			student_id VARCHAR(50) NOT NULL REFERENCES ecse_students(student_id) ON DELETE CASCADE,
			date DATE NOT NULL,
			status VARCHAR(20) NOT NULL CHECK (status IN ('present', 'absent', 'tardy', 'excused')),
			arrival_time TIME,
			departure_time TIME,
			notes TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(student_id, date)
		)`,

		// Create mileage_reports table
		`CREATE TABLE IF NOT EXISTS mileage_reports (
			id SERIAL PRIMARY KEY,
			vehicle_id VARCHAR(50) NOT NULL,
			driver VARCHAR(100),
			month INTEGER NOT NULL,
			year INTEGER NOT NULL,
			beginning_mileage DOUBLE PRECISION,
			ending_mileage DOUBLE PRECISION,
			total_miles DOUBLE PRECISION,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(vehicle_id, month, year)
		)`,

		// Create monthly_mileage_reports table if not exists
		`CREATE TABLE IF NOT EXISTS monthly_mileage_reports (
			id SERIAL PRIMARY KEY,
			report_month VARCHAR(20) NOT NULL,
			report_year INTEGER NOT NULL,
			bus_year INTEGER,
			bus_make VARCHAR(100),
			license_plate VARCHAR(50),
			bus_id VARCHAR(50),
			located_at VARCHAR(100),
			beginning_miles INTEGER,
			ending_miles INTEGER,
			total_miles INTEGER,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// COMMENTED OUT: fleet_vehicles table - migrating to vehicles table
		// `CREATE TABLE IF NOT EXISTS fleet_vehicles (
		// 	id SERIAL PRIMARY KEY,
		// 	vehicle_number INTEGER,
		// 	sheet_name VARCHAR(100),
		// 	year INTEGER,
		// 	make VARCHAR(100),
		// 	model VARCHAR(100),
		// 	description TEXT,
		// 	serial_number VARCHAR(100),
		// 	license VARCHAR(50),
		// 	location VARCHAR(100),
		// 	tire_size VARCHAR(50),
		// 	status VARCHAR(20) DEFAULT 'active',
		// 	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		// 	updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		// )`,

		// Create service_records table if not exists
		`CREATE TABLE IF NOT EXISTS service_records (
			id SERIAL PRIMARY KEY,
			unnamed_0 VARCHAR(255),
			unnamed_1 VARCHAR(255),
			unnamed_2 VARCHAR(255),
			unnamed_3 VARCHAR(255),
			unnamed_4 VARCHAR(255),
			unnamed_5 VARCHAR(255),
			unnamed_6 VARCHAR(255),
			unnamed_7 VARCHAR(255),
			unnamed_8 VARCHAR(255),
			unnamed_9 VARCHAR(255),
			unnamed_10 VARCHAR(255),
			unnamed_11 VARCHAR(255),
			unnamed_12 VARCHAR(255),
			unnamed_13 VARCHAR(255),
			maintenance_date DATE,
			vehicle_id VARCHAR(50),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// Add city, state, zip_code columns to ecse_students if they don't exist
		`DO $$ 
		BEGIN 
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
				WHERE table_name = 'ecse_students' AND column_name = 'city') THEN
				ALTER TABLE ecse_students ADD COLUMN city VARCHAR(100);
			END IF;
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
				WHERE table_name = 'ecse_students' AND column_name = 'state') THEN
				ALTER TABLE ecse_students ADD COLUMN state VARCHAR(50);
			END IF;
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
				WHERE table_name = 'ecse_students' AND column_name = 'zip_code') THEN
				ALTER TABLE ecse_students ADD COLUMN zip_code VARCHAR(20);
			END IF;
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
				WHERE table_name = 'ecse_students' AND column_name = 'address') THEN
				ALTER TABLE ecse_students ADD COLUMN address TEXT;
			END IF;
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
				WHERE table_name = 'ecse_students' AND column_name = 'updated_at') THEN
				ALTER TABLE ecse_students ADD COLUMN updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP;
			END IF;
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
				WHERE table_name = 'ecse_students' AND column_name = 'import_id') THEN
				ALTER TABLE ecse_students ADD COLUMN import_id VARCHAR(50);
			END IF;
		END $$;`,

		// Create mileage_records table if not exists
		`CREATE TABLE IF NOT EXISTS mileage_records (
			id SERIAL PRIMARY KEY,
			vehicle_id VARCHAR(50) NOT NULL,
			begin_mileage INTEGER NOT NULL,
			end_mileage INTEGER NOT NULL,
			import_id VARCHAR(50),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// COMMENTED OUT: import history table - legacy import system removed
		// `CREATE TABLE IF NOT EXISTS import_history (
		// 	id SERIAL PRIMARY KEY,
		// 	import_id VARCHAR(50) UNIQUE NOT NULL,
		// 	import_type VARCHAR(20) NOT NULL,
		// 	file_name VARCHAR(255) NOT NULL,
		// 	file_size BIGINT NOT NULL,
		// 	total_rows INTEGER DEFAULT 0,
		// 	successful_rows INTEGER DEFAULT 0,
		// 	failed_rows INTEGER DEFAULT 0,
		// 	error_count INTEGER DEFAULT 0,
		// 	warning_count INTEGER DEFAULT 0,
		// 	summary TEXT,
		// 	start_time TIMESTAMP NOT NULL,
		// 	end_time TIMESTAMP NOT NULL,
		// 	duration INTERVAL GENERATED ALWAYS AS (end_time - start_time) STORED,
		// 	imported_by VARCHAR(50) REFERENCES users(username),
		// 	rollback_available BOOLEAN DEFAULT TRUE,
		// 	rollback_expires_at TIMESTAMP,
		// 	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		// )`,

		// COMMENTED OUT: import errors table - legacy import system removed
		// `CREATE TABLE IF NOT EXISTS import_errors (
		// 	id SERIAL PRIMARY KEY,
		// 	import_id VARCHAR(50) REFERENCES import_history(import_id) ON DELETE CASCADE,
		// 	row_number INTEGER,
		// 	column_name VARCHAR(100),
		// 	sheet_name VARCHAR(100),
		// 	error_type VARCHAR(50),
		// 	error_message TEXT,
		// 	error_value TEXT,
		// 	severity VARCHAR(20) DEFAULT 'error',
		// 	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		// )`,

		// Add import_id columns to track imports
		`DO $$ 
		BEGIN 
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
				WHERE table_name = 'mileage_records' AND column_name = 'import_id') THEN
				ALTER TABLE mileage_records ADD COLUMN import_id VARCHAR(50);
			END IF;
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
				WHERE table_name = 'students' AND column_name = 'import_id') THEN
				ALTER TABLE students ADD COLUMN import_id VARCHAR(50);
			END IF;
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
				WHERE table_name = 'vehicles' AND column_name = 'import_id') THEN
				ALTER TABLE vehicles ADD COLUMN import_id VARCHAR(50);
			END IF;
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
				WHERE table_name = 'program_staff' AND column_name = 'import_id') THEN
				ALTER TABLE program_staff ADD COLUMN import_id VARCHAR(50);
			END IF;
			-- Year column fix for vehicles
			IF EXISTS (SELECT 1 FROM information_schema.columns 
				WHERE table_name = 'vehicles' AND column_name = 'year' AND data_type = 'integer') THEN
				ALTER TABLE vehicles ALTER COLUMN year TYPE VARCHAR(10) USING year::VARCHAR;
			END IF;
		END $$;`,

		// Create scheduled exports table
		`CREATE TABLE IF NOT EXISTS scheduled_exports (
			id SERIAL PRIMARY KEY,
			name VARCHAR(100) NOT NULL,
			export_type VARCHAR(50) NOT NULL,
			schedule VARCHAR(20) NOT NULL CHECK (schedule IN ('daily', 'weekly', 'monthly')),
			day_of_week INTEGER DEFAULT 0,
			day_of_month INTEGER DEFAULT 1,
			time VARCHAR(5) NOT NULL,
			format VARCHAR(10) NOT NULL DEFAULT 'xlsx',
			recipients TEXT,
			enabled BOOLEAN DEFAULT TRUE,
			last_run TIMESTAMP,
			next_run TIMESTAMP NOT NULL,
			created_by VARCHAR(50) REFERENCES users(username),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		
		// Create saved reports table
		`CREATE TABLE IF NOT EXISTS saved_reports (
			id SERIAL PRIMARY KEY,
			name VARCHAR(200) NOT NULL,
			description TEXT,
			data_source VARCHAR(100) NOT NULL,
			fields TEXT NOT NULL, -- JSON array of fields
			filters TEXT, -- JSON object of filters
			sort_by VARCHAR(100),
			sort_order VARCHAR(10) DEFAULT 'asc',
			chart_type VARCHAR(50),
			chart_config TEXT, -- JSON configuration
			created_by VARCHAR(50) REFERENCES users(username),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			last_run TIMESTAMP,
			is_public BOOLEAN DEFAULT FALSE
		)`,

		// Create fuel_records table
		`CREATE TABLE IF NOT EXISTS fuel_records (
			id SERIAL PRIMARY KEY,
			vehicle_id VARCHAR(50) NOT NULL,
			date DATE NOT NULL,
			gallons DECIMAL(10,2) NOT NULL,
			cost DECIMAL(10,2) NOT NULL,
			price_per_gallon DECIMAL(10,2) NOT NULL,
			odometer INTEGER NOT NULL,
			location VARCHAR(255),
			driver VARCHAR(100),
			notes TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			CONSTRAINT positive_gallons CHECK (gallons > 0),
			CONSTRAINT positive_cost CHECK (cost > 0),
			CONSTRAINT positive_odometer CHECK (odometer > 0)
		)`,

		// Create program_staff table if not exists
		`CREATE TABLE IF NOT EXISTS program_staff (
			id SERIAL PRIMARY KEY,
			report_month VARCHAR(20),
			report_year INTEGER,
			program_type VARCHAR(100),
			staff_count1 INTEGER,
			staff_count2 INTEGER,
			import_id VARCHAR(50),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// Create driver_locations table for real-time tracking
		`CREATE TABLE IF NOT EXISTS driver_locations (
			driver_username VARCHAR(50) PRIMARY KEY REFERENCES users(username) ON DELETE CASCADE,
			latitude DECIMAL(10,8),
			longitude DECIMAL(11,8),
			speed DECIMAL(5,2),
			heading DECIMAL(5,2),
			accuracy DECIMAL(6,2),
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// Create notifications table
		`CREATE TABLE IF NOT EXISTS notifications (
			id VARCHAR(100) PRIMARY KEY,
			type VARCHAR(50) NOT NULL,
			title VARCHAR(255) NOT NULL,
			message TEXT NOT NULL,
			priority VARCHAR(20) NOT NULL DEFAULT 'normal' CHECK (priority IN ('low', 'normal', 'high', 'urgent')),
			channels VARCHAR(255) NOT NULL DEFAULT 'email',
			scheduled_for TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'sent', 'failed', 'cancelled'))
		)`,

		// Create notification_recipients table
		`CREATE TABLE IF NOT EXISTS notification_recipients (
			id SERIAL PRIMARY KEY,
			notification_id VARCHAR(100) REFERENCES notifications(id) ON DELETE CASCADE,
			user_id VARCHAR(50),
			email VARCHAR(255),
			phone VARCHAR(20),
			UNIQUE(notification_id, user_id, email, phone)
		)`,

		// Create notification_deliveries table
		`CREATE TABLE IF NOT EXISTS notification_deliveries (
			id SERIAL PRIMARY KEY,
			notification_id VARCHAR(100) REFERENCES notifications(id) ON DELETE CASCADE,
			user_id VARCHAR(50),
			channel VARCHAR(20) NOT NULL,
			status VARCHAR(20) NOT NULL,
			delivered_at TIMESTAMP,
			error_message TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// Create in_app_notifications table
		`CREATE TABLE IF NOT EXISTS in_app_notifications (
			id SERIAL PRIMARY KEY,
			user_id VARCHAR(50) NOT NULL REFERENCES users(username) ON DELETE CASCADE,
			notification_id VARCHAR(100) NOT NULL,
			type VARCHAR(50) NOT NULL,
			subject VARCHAR(255) NOT NULL,
			message TEXT NOT NULL,
			data JSONB,
			read BOOLEAN DEFAULT FALSE,
			read_at TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(user_id, notification_id)
		)`,

		// Add email and phone to users table if not exists
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS email VARCHAR(255)`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS phone VARCHAR(20)`,

		// Drop unique constraints on route_assignments to allow multiple routes per driver/bus
		`ALTER TABLE route_assignments DROP CONSTRAINT IF EXISTS route_assignments_driver_route_id_key`,
		`ALTER TABLE route_assignments DROP CONSTRAINT IF EXISTS route_assignments_bus_id_route_id_key`,
		
		// Add a composite unique constraint to prevent duplicate assignments
		`ALTER TABLE route_assignments ADD CONSTRAINT route_assignments_unique_assignment 
		 UNIQUE(driver, bus_id, route_id)`,

		// Create budget tables
		`CREATE TABLE IF NOT EXISTS budgets (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			fiscal_year INTEGER NOT NULL,
			total_amount DECIMAL(15,2) NOT NULL DEFAULT 0,
			allocated_amount DECIMAL(15,2) NOT NULL DEFAULT 0,
			spent_amount DECIMAL(15,2) NOT NULL DEFAULT 0,
			status VARCHAR(20) NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'active', 'closed')),
			created_by VARCHAR(50) NOT NULL REFERENCES users(username),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		`CREATE TABLE IF NOT EXISTS budget_categories (
			id SERIAL PRIMARY KEY,
			budget_id INTEGER NOT NULL REFERENCES budgets(id) ON DELETE CASCADE,
			category_name VARCHAR(100) NOT NULL,
			category_type VARCHAR(50) NOT NULL,
			allocated_amount DECIMAL(15,2) NOT NULL DEFAULT 0,
			spent_amount DECIMAL(15,2) NOT NULL DEFAULT 0,
			description TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		`CREATE TABLE IF NOT EXISTS budget_transactions (
			id SERIAL PRIMARY KEY,
			budget_id INTEGER NOT NULL REFERENCES budgets(id) ON DELETE CASCADE,
			category_id INTEGER NOT NULL REFERENCES budget_categories(id),
			transaction_date DATE NOT NULL,
			amount DECIMAL(15,2) NOT NULL,
			transaction_type VARCHAR(20) NOT NULL CHECK (transaction_type IN ('expense', 'adjustment')),
			description TEXT,
			vehicle_id VARCHAR(50),
			reference_id VARCHAR(50),
			reference_type VARCHAR(50),
			created_by VARCHAR(50) NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		`CREATE TABLE IF NOT EXISTS budget_alerts (
			id SERIAL PRIMARY KEY,
			budget_id INTEGER NOT NULL REFERENCES budgets(id) ON DELETE CASCADE,
			category_id INTEGER REFERENCES budget_categories(id),
			alert_type VARCHAR(50) NOT NULL,
			threshold_pct DECIMAL(5,2),
			message TEXT NOT NULL,
			is_active BOOLEAN DEFAULT TRUE,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// Create indexes for budget tables
		`CREATE INDEX IF NOT EXISTS idx_budget_transactions_date ON budget_transactions(transaction_date DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_budget_transactions_category ON budget_transactions(category_id)`,
		`CREATE INDEX IF NOT EXISTS idx_budget_categories_budget ON budget_categories(budget_id)`,
		`CREATE INDEX IF NOT EXISTS idx_budget_alerts_active ON budget_alerts(budget_id, is_active) WHERE is_active = true`,

		// Messaging tables
		`CREATE TABLE IF NOT EXISTS conversations (
			id VARCHAR(100) PRIMARY KEY,
			type VARCHAR(20) NOT NULL DEFAULT 'direct', -- direct, group, broadcast
			name VARCHAR(255),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_conversations_updated ON conversations(updated_at DESC)`,
		
		`CREATE TABLE IF NOT EXISTS conversation_participants (
			conversation_id VARCHAR(100) REFERENCES conversations(id) ON DELETE CASCADE,
			user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
			joined_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			last_read_at TIMESTAMP,
			PRIMARY KEY (conversation_id, user_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_participants_user ON conversation_participants(user_id)`,
		
		`CREATE TABLE IF NOT EXISTS messages (
			id SERIAL PRIMARY KEY,
			conversation_id VARCHAR(100) REFERENCES conversations(id) ON DELETE CASCADE,
			sender_id INTEGER REFERENCES users(id),
			recipient_id INTEGER REFERENCES users(id), -- For direct messages
			content TEXT NOT NULL,
			message_type VARCHAR(20) DEFAULT 'text', -- text, location, emergency, system
			status VARCHAR(20) DEFAULT 'sent', -- sent, delivered, read
			metadata JSONB,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			read_at TIMESTAMP,
			CONSTRAINT check_message_type CHECK (message_type IN ('text', 'location', 'emergency', 'system')),
			CONSTRAINT check_status CHECK (status IN ('sent', 'delivered', 'read'))
		)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_conversation ON messages(conversation_id, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_sender ON messages(sender_id)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_unread ON messages(conversation_id, sender_id, read_at) WHERE read_at IS NULL`,
		
		`CREATE TABLE IF NOT EXISTS message_attachments (
			id SERIAL PRIMARY KEY,
			message_id INTEGER REFERENCES messages(id) ON DELETE CASCADE,
			type VARCHAR(20) NOT NULL, -- image, document, location
			url TEXT NOT NULL,
			filename VARCHAR(255),
			size BIGINT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_attachments_message ON message_attachments(message_id)`,
		
		// Trigger to update conversation timestamp when new message is added
		`CREATE OR REPLACE FUNCTION update_conversation_timestamp()
		RETURNS TRIGGER AS $$
		BEGIN
			UPDATE conversations 
			SET updated_at = CURRENT_TIMESTAMP 
			WHERE id = NEW.conversation_id;
			RETURN NEW;
		END;
		$$ LANGUAGE plpgsql`,
		
		`DROP TRIGGER IF EXISTS update_conversation_on_message ON messages`,
		
		`CREATE TRIGGER update_conversation_on_message
		AFTER INSERT ON messages
		FOR EACH ROW
		EXECUTE FUNCTION update_conversation_timestamp()`,
		
		// Emergency system tables
		`CREATE TABLE IF NOT EXISTS emergency_alerts (
			id SERIAL PRIMARY KEY,
			alert_id VARCHAR(100) UNIQUE NOT NULL,
			type VARCHAR(50) NOT NULL, -- breakdown, accident, medical, security, weather, sos, other
			severity VARCHAR(20) NOT NULL, -- critical, high, medium, low
			status VARCHAR(20) NOT NULL DEFAULT 'active', -- active, acknowledged, resolved, cancelled
			reporter_id INTEGER REFERENCES users(id),
			vehicle_id VARCHAR(50),
			route_id VARCHAR(50),
			location_lat DECIMAL(10,8),
			location_lng DECIMAL(11,8),
			location_address TEXT,
			title VARCHAR(255) NOT NULL,
			description TEXT,
			affected_users JSONB,
			timeline JSONB,
			metadata JSONB,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			acknowledged_at TIMESTAMP,
			resolved_at TIMESTAMP,
			CONSTRAINT check_emergency_type CHECK (type IN ('breakdown', 'accident', 'medical', 'security', 'weather', 'sos', 'other')),
			CONSTRAINT check_severity CHECK (severity IN ('critical', 'high', 'medium', 'low')),
			CONSTRAINT check_status CHECK (status IN ('active', 'acknowledged', 'resolved', 'cancelled'))
		)`,
		`CREATE INDEX IF NOT EXISTS idx_emergency_alerts_status ON emergency_alerts(status) WHERE status IN ('active', 'acknowledged')`,
		`CREATE INDEX IF NOT EXISTS idx_emergency_alerts_severity ON emergency_alerts(severity, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_emergency_alerts_reporter ON emergency_alerts(reporter_id)`,
		
		`CREATE TABLE IF NOT EXISTS emergency_responders (
			id SERIAL PRIMARY KEY,
			alert_id VARCHAR(100) REFERENCES emergency_alerts(alert_id) ON DELETE CASCADE,
			user_id INTEGER REFERENCES users(id),
			status VARCHAR(50) DEFAULT 'assigned', -- assigned, en_route, on_scene, completed
			assigned_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			eta TIMESTAMP,
			notes TEXT
		)`,
		`CREATE INDEX IF NOT EXISTS idx_emergency_responders_alert ON emergency_responders(alert_id)`,
		
		`CREATE TABLE IF NOT EXISTS emergency_protocols (
			id SERIAL PRIMARY KEY,
			type VARCHAR(50) NOT NULL,
			name VARCHAR(255) NOT NULL,
			steps JSONB NOT NULL,
			contacts JSONB,
			resources JSONB,
			is_active BOOLEAN DEFAULT TRUE,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		
		`CREATE TABLE IF NOT EXISTS emergency_contacts (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			role VARCHAR(100),
			phone VARCHAR(50),
			email VARCHAR(255),
			priority INTEGER DEFAULT 1,
			is_active BOOLEAN DEFAULT TRUE,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		
		`CREATE TABLE IF NOT EXISTS emergency_attachments (
			id SERIAL PRIMARY KEY,
			alert_id VARCHAR(100) REFERENCES emergency_alerts(alert_id) ON DELETE CASCADE,
			type VARCHAR(50) NOT NULL, -- photo, document, audio
			url TEXT NOT NULL,
			filename VARCHAR(255),
			size BIGINT,
			uploaded_by INTEGER REFERENCES users(id),
			uploaded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_emergency_attachments_alert ON emergency_attachments(alert_id)`,
	}

	for i, migration := range migrations {
		// Log which migration we're running
		tableName := "unknown"
		if strings.Contains(migration, "CREATE TABLE") {
			parts := strings.Split(migration, " ")
			for j, part := range parts {
				if strings.ToUpper(part) == "TABLE" && j+2 < len(parts) {
					tableName = strings.TrimSuffix(parts[j+2], "(")
					break
				}
			}
		} else if strings.Contains(migration, "CREATE INDEX") {
			tableName = "index"
		} else if strings.Contains(migration, "DO $$") {
			tableName = "ALTER TABLE"
		}

		log.Printf("Running migration %d: %s", i+1, tableName)

		if _, err := db.Exec(migration); err != nil {
			// Check if it's a duplicate table/constraint error
			errStr := err.Error()
			if strings.Contains(errStr, "already exists") {
				log.Printf("Migration %d: %s already exists, continuing", i+1, tableName)
				continue
			}

			// Log which migration failed
			log.Printf("Migration %d failed (%s): %v", i+1, tableName, err)

			// For index creation or ALTER TABLE, we can continue if it fails
			if strings.Contains(migration, "CREATE INDEX") || strings.Contains(migration, "ALTER TABLE") {
				log.Printf("Continuing despite error in %s", tableName)
				continue
			}

			return fmt.Errorf("failed to execute migration %d (%s): %w", i+1, tableName, err)
		}
	}

	log.Println("Database migrations completed successfully")
	return nil
}

// Database query functions

// GetMaintenanceLogsForVehicle retrieves all maintenance logs for a vehicle from the consolidated maintenance_records table
func getMaintenanceLogsForVehicle(vehicleID string) ([]CombinedMaintenanceLog, error) {
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	query := `
		SELECT 
			id,
			COALESCE(vehicle_id, CAST(vehicle_number AS VARCHAR)) as vehicle_id,
			CASE 
				WHEN vehicle_id ~ '^[0-9]+$' THEN 'bus'
				ELSE 'vehicle'
			END as vehicle_type,
			COALESCE(TO_CHAR(service_date, 'YYYY-MM-DD'), TO_CHAR(date, 'YYYY-MM-DD'), '') as date,
			COALESCE(
				CASE 
					WHEN work_description ILIKE '%oil%' THEN 'oil_change'
					WHEN work_description ILIKE '%tire%' THEN 'tire_service'
					WHEN work_description ILIKE '%inspect%' THEN 'inspection'
					WHEN work_description ILIKE '%repair%' THEN 'repair'
					ELSE 'other'
				END,
				'other'
			) as category,
			COALESCE(work_description, '') as notes,
			COALESCE(mileage, 0) as mileage,
			COALESCE(cost::numeric, 0) as cost,
			COALESCE(created_at, CURRENT_TIMESTAMP) as created_at
		FROM maintenance_records
		WHERE vehicle_id = $1 
		   OR CAST(vehicle_number AS VARCHAR) = $1
		ORDER BY COALESCE(service_date, date, created_at) DESC
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
	_, err = db.Exec(
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
