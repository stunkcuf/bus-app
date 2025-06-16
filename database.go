// database.go - All database operations and data loading/saving
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"sync"
	"time"
	
	_ "github.com/lib/pq"
)

// =============================================================================
// GLOBAL DATABASE CONNECTION
// =============================================================================

// PostgreSQL database connection
var db *sql.DB

// Cache variables
var userCache []User
var userCacheTime time.Time
var userCacheMutex sync.RWMutex

// =============================================================================
// DATABASE INITIALIZATION
// =============================================================================

// initDatabase initializes the PostgreSQL connection
func initDatabase() error {
	// Railway automatically provides DATABASE_URL environment variable
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return fmt.Errorf("DATABASE_URL environment variable not set")
	}

	var err error
	db, err = sql.Open("postgres", databaseURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	if err = db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	log.Println("✅ Connected to PostgreSQL database")
	return createTables()
}

// createTables creates all necessary database tables
func createTables() error {
	queries := []string{
		// Users table
		`CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			username VARCHAR(255) UNIQUE NOT NULL,
			password VARCHAR(255) NOT NULL,
			role VARCHAR(50) NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// Buses table
		`CREATE TABLE IF NOT EXISTS buses (
			id SERIAL PRIMARY KEY,
			bus_id VARCHAR(255) UNIQUE NOT NULL,
			status VARCHAR(50) NOT NULL DEFAULT 'active',
			model VARCHAR(255),
			capacity INTEGER DEFAULT 0,
			oil_status VARCHAR(50) DEFAULT 'good',
			tire_status VARCHAR(50) DEFAULT 'good',
			maintenance_notes TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// Routes table - WITH DESCRIPTION
		`CREATE TABLE IF NOT EXISTS routes (
			id SERIAL PRIMARY KEY,
			route_id VARCHAR(255) UNIQUE NOT NULL,
			route_name VARCHAR(255) NOT NULL,
			description TEXT,
			positions JSONB,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// Students table
		`CREATE TABLE IF NOT EXISTS students (
			id SERIAL PRIMARY KEY,
			student_id VARCHAR(255) UNIQUE NOT NULL,
			name VARCHAR(255) NOT NULL,
			locations JSONB,
			phone_number VARCHAR(50),
			alt_phone_number VARCHAR(50),
			guardian VARCHAR(255),
			pickup_time TIME,
			dropoff_time TIME,
			position_number INTEGER,
			route_id VARCHAR(255),
			driver VARCHAR(255),
			active BOOLEAN DEFAULT true,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// Route assignments table
		`CREATE TABLE IF NOT EXISTS route_assignments (
			id SERIAL PRIMARY KEY,
			driver VARCHAR(255) NOT NULL,
			bus_id VARCHAR(255) NOT NULL,
			route_id VARCHAR(255) NOT NULL,
			route_name VARCHAR(255),
			assigned_date DATE,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(driver)
		)`,

		// Driver logs table
		`CREATE TABLE IF NOT EXISTS driver_logs (
			id SERIAL PRIMARY KEY,
			driver VARCHAR(255) NOT NULL,
			bus_id VARCHAR(255),
			route_id VARCHAR(255),
			date DATE NOT NULL,
			period VARCHAR(50) NOT NULL,
			departure_time VARCHAR(10),
			arrival_time VARCHAR(10),
			mileage DECIMAL(10,2) DEFAULT 0,
			attendance JSONB,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(driver, date, period)
		)`,

		// Maintenance logs table
		`CREATE TABLE IF NOT EXISTS maintenance_logs (
			id SERIAL PRIMARY KEY,
			bus_id VARCHAR(255) NOT NULL,
			date DATE NOT NULL,
			category VARCHAR(100),
			notes TEXT,
			mileage INTEGER DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// Vehicles table (for company fleet)
		`CREATE TABLE IF NOT EXISTS vehicles (
			id SERIAL PRIMARY KEY,
			vehicle_id VARCHAR(255) UNIQUE NOT NULL,
			model VARCHAR(255),
			description TEXT,
			year VARCHAR(4),
			tire_size VARCHAR(50),
			license VARCHAR(50),
			oil_status VARCHAR(50) DEFAULT 'good',
			tire_status VARCHAR(50) DEFAULT 'good',
			status VARCHAR(50) DEFAULT 'active',
			maintenance_notes TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
	}

	for _, query := range queries {
		if _, err := db.Exec(query); err != nil {
			return fmt.Errorf("failed to create table: %w", err)
		}
	}

	log.Println("✅ Database tables created successfully")
	
	// Add missing columns if they don't exist (for existing databases)
	if err := ensureSchemaUpdates(); err != nil {
		log.Printf("⚠️  Warning: Schema updates failed: %v", err)
		// Don't fail completely, just log the warning
	}
	
	return nil
}

// ensureSchemaUpdates handles schema evolution for existing databases
func ensureSchemaUpdates() error {
	log.Println("🔧 Checking for schema updates...")
	
	// Add missing columns if they don't exist
	schemaUpdates := []struct {
		table      string
		column     string
		definition string
	}{
		{"routes", "description", "ALTER TABLE routes ADD COLUMN IF NOT EXISTS description TEXT"},
		{"vehicles", "serial_number", "ALTER TABLE vehicles ADD COLUMN IF NOT EXISTS serial_number VARCHAR(255)"},
		{"vehicles", "base", "ALTER TABLE vehicles ADD COLUMN IF NOT EXISTS base VARCHAR(255)"},
		{"vehicles", "service_interval", "ALTER TABLE vehicles ADD COLUMN IF NOT EXISTS service_interval INTEGER DEFAULT 5000"},
		{"buses", "updated_at", "ALTER TABLE buses ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP"},
	}

	for _, update := range schemaUpdates {
		// PostgreSQL specific: ADD COLUMN IF NOT EXISTS
		if _, err := db.Exec(update.definition); err != nil {
			log.Printf("⚠️  Could not add column %s.%s: %v", update.table, update.column, err)
			// Continue with other updates
		} else {
			log.Printf("✅ Ensured column %s.%s exists", update.table, update.column)
		}
	}

	return nil
}

// setupDatabase initializes database with migration support
func setupDatabase() {
	log.Println("🗄️  Setting up database...")
	
	if err := initDatabase(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Run migration if JSON files exist
	if _, err := os.Stat("data"); err == nil {
		if err := migrateJSONToPostgreSQL(); err != nil {
			log.Printf("⚠️  Migration failed: %v", err)
			log.Println("Continuing with database setup...")
		}
	}
}

// =============================================================================
// USER OPERATIONS
// =============================================================================

// loadUsers loads from PostgreSQL first, JSON fallback
func loadUsers() []User {
	if db != nil {
		rows, err := db.Query("SELECT username, password, role FROM users ORDER BY username")
		if err != nil {
			log.Printf("Error loading users from database: %v", err)
			return loadUsersFromJSON() // Fallback
		}
		defer rows.Close()

		var users []User
		for rows.Next() {
			var user User
			err := rows.Scan(&user.Username, &user.Password, &user.Role)
			if err != nil {
				log.Printf("Error scanning user: %v", err)
				continue
			}
			users = append(users, user)
		}
		return users
	}
	return loadUsersFromJSON()
}

// loadUsersFromJSON is the JSON fallback for users
func loadUsersFromJSON() []User {
	userCacheMutex.RLock()
	if time.Since(userCacheTime) < 30*time.Second && userCache != nil {
		defer userCacheMutex.RUnlock()
		return userCache
	}
	userCacheMutex.RUnlock()

	userCacheMutex.Lock()
	defer userCacheMutex.Unlock()

	f, err := os.Open("data/users.json")
	if err != nil {
		return nil
	}
	defer f.Close()
	var users []User
	json.NewDecoder(f).Decode(&users)
	userCache = users
	userCacheTime = time.Now()
	return users
}

// saveUsers saves to PostgreSQL first, JSON fallback
func saveUsers(users []User) error {
	if db != nil {
		tx, err := db.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback()

		// Clear existing users
		_, err = tx.Exec("DELETE FROM users")
		if err != nil {
			return err
		}

		// Insert new users
		for _, user := range users {
			_, err = tx.Exec("INSERT INTO users (username, password, role) VALUES ($1, $2, $3)",
				user.Username, user.Password, user.Role)
			if err != nil {
				return err
			}
		}

		return tx.Commit()
	}
	return saveUsersToJSON(users)
}

func saveUsersToJSON(users []User) error {
	f, err := os.Create("data/users.json")
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(users)
}

// =============================================================================
// BUS OPERATIONS
// =============================================================================

// loadBuses - PostgreSQL first, JSON fallback - NEVER returns nil
func loadBuses() []*Bus {
	// Initialize with empty slice to ensure we never return nil
	buses := make([]*Bus, 0)
	
	if db != nil {
		rows, err := db.Query(`SELECT bus_id, status, model, capacity, oil_status, tire_status, maintenance_notes 
			FROM buses ORDER BY bus_id`)
		if err != nil {
			log.Printf("Error loading buses from database: %v", err)
			// Try JSON fallback
			jsonBuses := loadBusesFromJSON()
			if jsonBuses != nil {
				return jsonBuses
			}
			// Return empty slice, not nil
			return buses
		}
		defer rows.Close()

		for rows.Next() {
			bus := &Bus{}
			err := rows.Scan(&bus.BusID, &bus.Status, &bus.Model, &bus.Capacity, 
				&bus.OilStatus, &bus.TireStatus, &bus.MaintenanceNotes)
			if err != nil {
				log.Printf("Error scanning bus: %v", err)
				continue
			}
			buses = append(buses, bus)
		}
		
		log.Printf("Loaded %d buses from database", len(buses))
		return buses
	}
	
	// No database, use JSON
	jsonBuses := loadBusesFromJSON()
	if jsonBuses != nil {
		return jsonBuses
	}
	
	// Always return at least an empty slice
	return buses
}

func loadBusesFromJSON() []*Bus {
	// Initialize with empty slice
	buses := make([]*Bus, 0)
	
	f, err := os.Open("data/buses.json")
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("buses.json not found, returning empty slice")
			return buses // Return empty slice, not nil
		}
		log.Printf("Error opening buses.json: %v", err)
		return buses // Return empty slice, not nil
	}
	defer f.Close()

	decoder := json.NewDecoder(f)
	if err := decoder.Decode(&buses); err != nil {
		log.Printf("Error decoding buses.json: %v", err)
		// Reset to empty slice on decode error
		buses = make([]*Bus, 0)
		return buses
	}
	
	log.Printf("Loaded %d buses from JSON", len(buses))
	return buses
}

// saveBuses - PostgreSQL first, JSON fallback
func saveBuses(buses []*Bus) error {
	if db != nil {
		tx, err := db.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback()

		for _, bus := range buses {
			_, err = tx.Exec(`
				INSERT INTO buses (bus_id, status, model, capacity, oil_status, tire_status, maintenance_notes) 
				VALUES ($1, $2, $3, $4, $5, $6, $7) 
				ON CONFLICT (bus_id) DO UPDATE SET 
					status = EXCLUDED.status,
					model = EXCLUDED.model,
					capacity = EXCLUDED.capacity,
					oil_status = EXCLUDED.oil_status,
					tire_status = EXCLUDED.tire_status,
					maintenance_notes = EXCLUDED.maintenance_notes,
					updated_at = CURRENT_TIMESTAMP
			`, bus.BusID, bus.Status, bus.Model, bus.Capacity, bus.OilStatus, bus.TireStatus, bus.MaintenanceNotes)
			
			if err != nil {
				return err
			}
		}

		return tx.Commit()
	}
	return saveBusesToJSON(buses)
}

func saveBusesToJSON(buses []*Bus) error {
	f, err := os.Create("data/buses.json")
	if err != nil {
		return fmt.Errorf("failed to create buses.json: %w", err)
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(buses); err != nil {
		return fmt.Errorf("failed to encode buses: %w", err)
	}

	return nil
}

// =============================================================================
// ROUTE OPERATIONS
// =============================================================================

// loadRoutes - PostgreSQL first, JSON fallback
func loadRoutes() ([]Route, error) {
	if db != nil {
		rows, err := db.Query("SELECT route_id, route_name, description, positions FROM routes ORDER BY route_id")
		if err != nil {
			log.Printf("loadRoutes: Database query error: %v, falling back to JSON", err)
			return loadRoutesFromJSON()
		}
		defer rows.Close()

		var routes []Route
		for rows.Next() {
			var route Route
			var positionsJSON []byte
			var description sql.NullString
			err := rows.Scan(&route.RouteID, &route.RouteName, &description, &positionsJSON)
			if err != nil {
				log.Printf("Error scanning route: %v", err)
				continue
			}

			if description.Valid {
				route.Description = description.String
			}

			// Parse positions JSON
			if len(positionsJSON) > 0 {
				err = json.Unmarshal(positionsJSON, &route.Positions)
				if err != nil {
					log.Printf("Error parsing positions JSON: %v", err)
					route.Positions = []struct {
						Position int    `json:"position"`
						Student  string `json:"student"`
					}{}
				}
			}

			routes = append(routes, route)
		}
		return routes, nil
	}
	return loadRoutesFromJSON()
}

// loadRoutesFromJSON helper function to load routes from JSON file
func loadRoutesFromJSON() ([]Route, error) {
	log.Printf("loadRoutesFromJSON: Attempting to load routes from JSON file")
	
	// Check if file exists
	if _, err := os.Stat("data/routes.json"); os.IsNotExist(err) {
		log.Printf("loadRoutesFromJSON: routes.json does not exist, returning empty array")
		return []Route{}, nil
	}
	
	f, err := os.Open("data/routes.json")
	if err != nil {
		log.Printf("loadRoutesFromJSON: Error opening file: %v", err)
		if os.IsPermission(err) {
			return nil, fmt.Errorf("permission denied accessing routes.json")
		}
		return nil, fmt.Errorf("failed to open routes file: %w", err)
	}
	defer f.Close()

	var routes []Route
	decoder := json.NewDecoder(f)
	if err := decoder.Decode(&routes); err != nil {
		log.Printf("loadRoutesFromJSON: Error decoding JSON: %v", err)
		// Check if it's empty file
		if err == io.EOF {
			log.Printf("loadRoutesFromJSON: Empty file, returning empty array")
			return []Route{}, nil
		}
		return nil, fmt.Errorf("invalid JSON in routes file: %w", err)
	}
	
	log.Printf("loadRoutesFromJSON: Successfully loaded %d routes from JSON", len(routes))
	return routes, nil
}

// =============================================================================
// ROUTE ASSIGNMENT OPERATIONS
// =============================================================================

// loadRouteAssignments - PostgreSQL first, JSON fallback
func loadRouteAssignments() ([]RouteAssignment, error) {
	if db != nil {
		rows, err := db.Query(`SELECT driver, bus_id, route_id, route_name, assigned_date 
			FROM route_assignments ORDER BY driver`)
		if err != nil {
			return loadRouteAssignmentsFromJSON() // Fallback
		}
		defer rows.Close()

		var assignments []RouteAssignment
		for rows.Next() {
			var assignment RouteAssignment
			err := rows.Scan(&assignment.Driver, &assignment.BusID, &assignment.RouteID,
				&assignment.RouteName, &assignment.AssignedDate)
			if err != nil {
				log.Printf("Error scanning assignment: %v", err)
				continue
			}
			assignments = append(assignments, assignment)
		}
		return assignments, nil
	}
	return loadRouteAssignmentsFromJSON()
}

func loadRouteAssignmentsFromJSON() ([]RouteAssignment, error) {
	assignments, err := loadJSON[RouteAssignment]("data/route_assignments.json")
	if err != nil {
		if os.IsNotExist(err) {
			return []RouteAssignment{}, nil
		}
		return nil, fmt.Errorf("failed to load route assignments: %w", err)
	}
	return assignments, nil
}

// saveRouteAssignments - PostgreSQL first, JSON fallback
func saveRouteAssignments(assignments []RouteAssignment) error {
	if db != nil {
		tx, err := db.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback()

		// Clear existing assignments
		_, err = tx.Exec("DELETE FROM route_assignments")
		if err != nil {
			return err
		}

		// Insert assignments
		for _, assignment := range assignments {
			_, err = tx.Exec(`
				INSERT INTO route_assignments (driver, bus_id, route_id, route_name, assigned_date) 
				VALUES ($1, $2, $3, $4, $5)
			`, assignment.Driver, assignment.BusID, assignment.RouteID, assignment.RouteName, assignment.AssignedDate)
			
			if err != nil {
				return err
			}
		}

		return tx.Commit()
	}
	return saveRouteAssignmentsToJSON(assignments)
}

func saveRouteAssignmentsToJSON(assignments []RouteAssignment) error {
	f, err := os.Create("data/route_assignments.json")
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(assignments)
}

// =============================================================================
// DRIVER LOG OPERATIONS
// =============================================================================

// loadDriverLogs - PostgreSQL first, JSON fallback
func loadDriverLogs() ([]DriverLog, error) {
	if db != nil {
		rows, err := db.Query(`SELECT driver, bus_id, route_id, date, period, departure_time, 
			arrival_time, mileage, attendance FROM driver_logs ORDER BY date DESC, driver`)
		if err != nil {
			return loadJSON[DriverLog]("data/driver_logs.json")
		}
		defer rows.Close()

		var logs []DriverLog
		for rows.Next() {
			var driverLog DriverLog
			var attendanceJSON []byte
			err := rows.Scan(&driverLog.Driver, &driverLog.BusID, &driverLog.RouteID, &driverLog.Date, &driverLog.Period,
				&driverLog.Departure, &driverLog.Arrival, &driverLog.Mileage, &attendanceJSON)
			if err != nil {
				log.Printf("Error scanning driver log: %v", err)
				continue
			}

			// Parse attendance JSON
			err = json.Unmarshal(attendanceJSON, &driverLog.Attendance)
			if err != nil {
				log.Printf("Error parsing attendance JSON: %v", err)
				continue
			}

			logs = append(logs, driverLog)
		}
		return logs, nil
	}
	return loadJSON[DriverLog]("data/driver_logs.json")
}

// saveDriverLogs - PostgreSQL first, JSON fallback
func saveDriverLogs(logs []DriverLog) error {
	if db != nil {
		tx, err := db.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback()

		for _, driverLog := range logs {
			attendanceJSON, _ := json.Marshal(driverLog.Attendance)
			_, err = tx.Exec(`
				INSERT INTO driver_logs (driver, bus_id, route_id, date, period, departure_time, 
					arrival_time, mileage, attendance) 
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) 
				ON CONFLICT (driver, date, period) DO UPDATE SET 
					bus_id = EXCLUDED.bus_id,
					route_id = EXCLUDED.route_id,
					departure_time = EXCLUDED.departure_time,
					arrival_time = EXCLUDED.arrival_time,
					mileage = EXCLUDED.mileage,
					attendance = EXCLUDED.attendance
			`, driverLog.Driver, driverLog.BusID, driverLog.RouteID, driverLog.Date, driverLog.Period, driverLog.Departure,
			   driverLog.Arrival, driverLog.Mileage, attendanceJSON)
			
			if err != nil {
				return err
			}
		}

		return tx.Commit()
	}
	return saveDriverLogsToJSON(logs)
}

func saveDriverLogsToJSON(logs []DriverLog) error {
	f, err := os.Create("data/driver_logs.json")
	if err != nil {
		return fmt.Errorf("failed to create driver logs file: %w", err)
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(logs); err != nil {
		return fmt.Errorf("failed to encode driver logs: %w", err)
	}

	return nil
}

// =============================================================================
// STUDENT OPERATIONS
// =============================================================================

// loadStudents - PostgreSQL first, JSON fallback
func loadStudents() []Student {
	if db != nil {
		rows, err := db.Query(`SELECT student_id, name, locations, phone_number, alt_phone_number, 
			guardian, pickup_time, dropoff_time, position_number, route_id, driver, active 
			FROM students ORDER BY name`)
		if err != nil {
			log.Printf("Error loading students from database: %v", err)
			return loadStudentsFromJSON()
		}
		defer rows.Close()

		var students []Student
		for rows.Next() {
			var student Student
			var locationsJSON []byte
			err := rows.Scan(&student.StudentID, &student.Name, &locationsJSON, &student.PhoneNumber,
				&student.AltPhoneNumber, &student.Guardian, &student.PickupTime, &student.DropoffTime,
				&student.PositionNumber, &student.RouteID, &student.Driver, &student.Active)
			if err != nil {
				log.Printf("Error scanning student: %v", err)
				continue
			}

			// Parse locations JSON
			err = json.Unmarshal(locationsJSON, &student.Locations)
			if err != nil {
				log.Printf("Error parsing locations JSON: %v", err)
				continue
			}

			students = append(students, student)
		}
		return students
	}
	return loadStudentsFromJSON()
}

func loadStudentsFromJSON() []Student {
	f, err := os.Open("data/students.json")
	if err != nil {
		return []Student{}
	}
	defer f.Close()
	var students []Student
	json.NewDecoder(f).Decode(&students)
	return students
}

// saveStudents - PostgreSQL first, JSON fallback
func saveStudents(students []Student) error {
	if db != nil {
		tx, err := db.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback()

		// Clear existing students
		_, err = tx.Exec("DELETE FROM students")
		if err != nil {
			return err
		}

		// Insert students
		for _, student := range students {
			locationsJSON, _ := json.Marshal(student.Locations)
			_, err = tx.Exec(`
				INSERT INTO students (student_id, name, locations, phone_number, alt_phone_number, 
					guardian, pickup_time, dropoff_time, position_number, route_id, driver, active) 
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
			`, student.StudentID, student.Name, locationsJSON, student.PhoneNumber, student.AltPhoneNumber,
			   student.Guardian, student.PickupTime, student.DropoffTime, student.PositionNumber,
			   student.RouteID, student.Driver, student.Active)
			
			if err != nil {
				return err
			}
		}

		return tx.Commit()
	}
	return saveStudentsToJSON(students)
}

func saveStudentsToJSON(students []Student) error {
	f, err := os.Create("data/students.json")
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(students)
}

// =============================================================================
// MAINTENANCE LOG OPERATIONS
// =============================================================================

// loadMaintenanceLogs - PostgreSQL first, JSON fallback
func loadMaintenanceLogs() []MaintenanceLog {
	if db != nil {
		rows, err := db.Query("SELECT bus_id, date, category, notes, mileage FROM maintenance_logs ORDER BY date DESC")
		if err != nil {
			log.Printf("Error loading maintenance logs: %v", err)
			logs, _ := loadJSON[MaintenanceLog]("data/maintenance.json")
			return logs
		}
		defer rows.Close()

		var logs []MaintenanceLog
		for rows.Next() {
			var maintLog MaintenanceLog
			err := rows.Scan(&maintLog.BusID, &maintLog.Date, &maintLog.Category, &maintLog.Notes, &maintLog.Mileage)
			if err != nil {
				log.Printf("Error scanning maintenance log: %v", err)
				continue
			}
			logs = append(logs, maintLog)
		}
		return logs
	}
	logs, _ := loadJSON[MaintenanceLog]("data/maintenance.json")
	return logs
}

// saveMaintenanceLogs - PostgreSQL first, JSON fallback
func saveMaintenanceLogs(logs []MaintenanceLog) error {
	if db != nil {
		tx, err := db.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback()

		for _, maintLog := range logs {
			_, err = tx.Exec(`
				INSERT INTO maintenance_logs (bus_id, date, category, notes, mileage) 
				VALUES ($1, $2, $3, $4, $5)
			`, maintLog.BusID, maintLog.Date, maintLog.Category, maintLog.Notes, maintLog.Mileage)
			
			if err != nil {
				return err
			}
		}

		return tx.Commit()
	}
	return saveMaintenanceLogsToJSON(logs)
}

func saveMaintenanceLogsToJSON(logs []MaintenanceLog) error {
	f, err := os.Create("data/maintenance.json")
	if err != nil { 
		return err 
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(logs)
}

// =============================================================================
// VEHICLE OPERATIONS
// =============================================================================

// loadVehicles - PostgreSQL first, JSON fallback
func loadVehicles() []Vehicle {
	if db != nil {
		rows, err := db.Query(`
			SELECT vehicle_id, model, description, year, tire_size, license, 
			       oil_status, tire_status, status, maintenance_notes,
			       COALESCE(serial_number, ''), COALESCE(base, ''),
			       COALESCE(service_interval, 5000)
			FROM vehicles 
			ORDER BY vehicle_id
		`)
		if err != nil {
			log.Printf("Error loading vehicles from database: %v", err)
			return loadVehiclesFromJSON()
		}
		defer rows.Close()

		var vehicles []Vehicle
		for rows.Next() {
			var vehicle Vehicle
			err := rows.Scan(
				&vehicle.VehicleID, &vehicle.Model, &vehicle.Description, 
				&vehicle.Year, &vehicle.TireSize, &vehicle.License,
				&vehicle.OilStatus, &vehicle.TireStatus, &vehicle.Status, 
				&vehicle.MaintenanceNotes, &vehicle.SerialNumber, 
				&vehicle.Base, &vehicle.ServiceInterval,
			)
			if err != nil {
				log.Printf("Error scanning vehicle: %v", err)
				continue
			}
			vehicles = append(vehicles, vehicle)
		}
		return vehicles
	}
	return loadVehiclesFromJSON()
}

func loadVehiclesFromJSON() []Vehicle {
	f, err := os.Open("data/vehicle.json")
	if err != nil {
		log.Printf("Error loading vehicles: %v", err)
		return []Vehicle{}
	}
	defer f.Close()
	var vehicles []Vehicle
	json.NewDecoder(f).Decode(&vehicles)
	return vehicles
}

// saveVehicles - PostgreSQL first, JSON fallback
func saveVehicles(vehicles []Vehicle) error {
	if db != nil {
		tx, err := db.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback()

		for _, vehicle := range vehicles {
			_, err = tx.Exec(`
				INSERT INTO vehicles (
					vehicle_id, model, description, year, tire_size, license, 
					oil_status, tire_status, status, maintenance_notes,
					serial_number, base, service_interval
				) 
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13) 
				ON CONFLICT (vehicle_id) DO UPDATE SET 
					model = EXCLUDED.model,
					description = EXCLUDED.description,
					year = EXCLUDED.year,
					tire_size = EXCLUDED.tire_size,
					license = EXCLUDED.license,
					oil_status = EXCLUDED.oil_status,
					tire_status = EXCLUDED.tire_status,
					status = EXCLUDED.status,
					maintenance_notes = EXCLUDED.maintenance_notes,
					serial_number = EXCLUDED.serial_number,
					base = EXCLUDED.base,
					service_interval = EXCLUDED.service_interval
			`, vehicle.VehicleID, vehicle.Model, vehicle.Description, vehicle.Year, 
			   vehicle.TireSize, vehicle.License, vehicle.OilStatus, vehicle.TireStatus,
			   vehicle.Status, vehicle.MaintenanceNotes, vehicle.SerialNumber, 
			   vehicle.Base, vehicle.ServiceInterval)
			
			if err != nil {
				return err
			}
		}

		return tx.Commit()
	}
	return saveVehiclesToJSON(vehicles)
}

func saveVehiclesToJSON(vehicles []Vehicle) error {
	f, err := os.Create("data/vehicle.json")
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(vehicles)
}

// =============================================================================
// MIGRATION FUNCTIONS
// =============================================================================

// Move all your migration functions here:
// migrateJSONToPostgreSQL, migrateUsers, migrateBuses, migrateRoutes, 
// migrateStudents, migrateRouteAssignments, migrateDriverLogs, 
// migrateMaintenanceLogs, migrateVehicles

// Add the migration functions from your main.go here...
