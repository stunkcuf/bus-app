package main

import (
	"embed"
	"encoding/json"
	"fmt"
	git "github.com/go-git/go-git/v5"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
	"database/sql"
	_ "github.com/lib/pq"
)

type User struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

type Attendance struct {
	Date    string `json:"date"`
	Driver  string `json:"driver"`
	Route   string `json:"route"`
	Present int    `json:"present"`
}

type Mileage struct {
	Date   string  `json:"date"`
	Driver string  `json:"driver"`
	Route  string  `json:"route"`
	Miles  float64 `json:"miles"`
}

type Activity struct {
	Date       string  `json:"date"`
	Driver     string  `json:"driver"`
	TripName   string  `json:"trip_name"`
	Attendance int     `json:"attendance"`
	Miles      float64 `json:"miles"`
	Notes      string  `json:"notes"`
}

type DriverSummary struct {
	Name              string
	TotalMorning      int
	TotalEvening      int
	TotalMiles        float64
	MonthlyAvgMiles   float64
	MonthlyAttendance int
}

type RouteStats struct {
	RouteName       string
	TotalMiles      float64
	AvgMiles        float64
	AttendanceDay   int
	AttendanceWeek  int
	AttendanceMonth int
}

type Route struct {
	RouteID     string `json:"route_id"`
	RouteName   string `json:"route_name"`
	Description string `json:"description"`  // Add this line
	Positions []struct {
		Position int    `json:"position"`
		Student  string `json:"student"`
	} `json:"positions"`
}

type Bus struct {
	BusID            string `json:"bus_id"`
	Status           string `json:"status"` // active, maintenance, out_of_service
	Model            string `json:"model"`
	Capacity         int    `json:"capacity"`
	OilStatus        string `json:"oil_status"`        // good, due, overdue
	TireStatus       string `json:"tire_status"`       // good, worn, replace
	MaintenanceNotes string `json:"maintenance_notes"`
}

type Vehicle struct {
	VehicleID        string `json:"vehicle_id"`
	Model            string `json:"model"`
	Description      string `json:"description"`
	Year             string `json:"year"`
	TireSize         string `json:"tire_size"`
	License          string `json:"license"`
	OilStatus        string `json:"oil_status"`
	TireStatus       string `json:"tire_status"`
	Status           string `json:"status"`
	MaintenanceNotes string `json:"maintenance_notes"`
}

type Student struct {
	StudentID       string     `json:"student_id"`
	Name            string     `json:"name"`
	Locations       []Location `json:"locations"`
	PhoneNumber     string     `json:"phone_number"`
	AltPhoneNumber  string     `json:"alt_phone_number"`
	Guardian        string     `json:"guardian"`
	PickupTime      string     `json:"pickup_time"`
	DropoffTime     string     `json:"dropoff_time"`
	PositionNumber  int        `json:"position_number"`
	RouteID         string     `json:"route_id"`
	Driver          string     `json:"driver"`
	Active          bool       `json:"active"`
}

type Location struct {
	Type        string `json:"type"` // "pickup" or "dropoff"
	Address     string `json:"address"`
	Description string `json:"description"`
}

type RouteAssignment struct {
	Driver       string `json:"driver"`
	BusID        string `json:"bus_id"`
	RouteID      string `json:"route_id"`
	RouteName    string `json:"route_name"`
	AssignedDate string `json:"assigned_date"`
}

type DriverLog struct {
	Driver     string `json:"driver"`
	BusID      string `json:"bus_id"`
	RouteID    string `json:"route_id"`
	Date       string `json:"date"`
	Period     string `json:"period"`
	Departure  string `json:"departure_time"`
	Arrival    string `json:"arrival_time"`
	Mileage    float64 `json:"mileage"`
	Attendance []struct {
		Position   int    `json:"position"`
		Present    bool   `json:"present"`
		PickupTime string `json:"pickup_time,omitempty"`
	} `json:"attendance"`
}

type DashboardData struct {
	User            *User
	Role            string
	DriverSummaries []*DriverSummary
	RouteStats      []*RouteStats
	Activities      []Activity
	Routes          []Route
	Users           []User
	Buses           []*Bus
}

type AssignRouteData struct {
	User            *User
	Assignments     []RouteAssignment
	Drivers         []User
	AvailableRoutes []Route
	AvailableBuses  []*Bus
}

type FleetData struct {
	User  *User
	Buses []*Bus
	Today string
}

type MaintenanceLog struct {
	BusID    string `json:"bus_id"`
	Date     string `json:"date"`      // YYYY‑MM‑DD
	Category string `json:"category"`  // oil, tires, brakes, etc.
	Notes    string `json:"notes"`
	Mileage  int    `json:"mileage"`   // optional
}

type StudentData struct {
	User     *User
	Students []Student
	Routes   []Route
}

type CompanyFleetData struct {
	User     *User
	Vehicles []Vehicle
}

//go:embed templates/*.html
var tmplFS embed.FS

var templates *template.Template

// PostgreSQL database connection
var db *sql.DB

func init() {
	var err error

	// Create function map for templates
	funcMap := template.FuncMap{
		"json": func(v interface{}) template.JS {
			b, err := json.Marshal(v)
			if err != nil {
				log.Printf("JSON marshal error: %v", err)
				return template.JS("{}")
			}
			return template.JS(b)
		},
		"add": func(a, b int) int {
			return a + b
		},
		"len": func(v interface{}) int {
			switch s := v.(type) {
			case []interface{}:
				return len(s)
			case []*Bus:
				return len(s)
			case []Bus:
				return len(s)
			default:
				return 0
			}
		},
	}

	// Parse templates from embedded filesystem
	templates, err = template.New("").Funcs(funcMap).ParseFS(tmplFS, "templates/*.html")
	if err != nil {
		log.Fatalf("Template parsing failed: %v", err)
	}

	log.Println("Templates loaded successfully")
}

// =============================================================================
// DATABASE FUNCTIONS
// =============================================================================

// Database initialization function
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

// Create all necessary tables
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

		// Routes table
		`CREATE TABLE IF NOT EXISTS routes (
			id SERIAL PRIMARY KEY,
			route_id VARCHAR(255) UNIQUE NOT NULL,
			route_name VARCHAR(255) NOT NULL,
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
	return nil
}

// Migration function to move JSON data to PostgreSQL
func migrateJSONToPostgreSQL() error {
	log.Println("🔄 Starting migration from JSON files to PostgreSQL...")

	// Migrate Users
	if err := migrateUsers(); err != nil {
		log.Printf("❌ Failed to migrate users: %v", err)
		return err
	}

	// Migrate Buses
	if err := migrateBuses(); err != nil {
		log.Printf("❌ Failed to migrate buses: %v", err)
		return err
	}

	// Migrate Routes
	if err := migrateRoutes(); err != nil {
		log.Printf("❌ Failed to migrate routes: %v", err)
		return err
	}

	// Migrate Students
	if err := migrateStudents(); err != nil {
		log.Printf("❌ Failed to migrate students: %v", err)
		return err
	}

	// Migrate Route Assignments
	if err := migrateRouteAssignments(); err != nil {
		log.Printf("❌ Failed to migrate route assignments: %v", err)
		return err
	}

	// Migrate Driver Logs
	if err := migrateDriverLogs(); err != nil {
		log.Printf("❌ Failed to migrate driver logs: %v", err)
		return err
	}

	// Migrate Maintenance Logs
	if err := migrateMaintenanceLogs(); err != nil {
		log.Printf("❌ Failed to migrate maintenance logs: %v", err)
		return err
	}

	// Migrate Vehicles
	if err := migrateVehicles(); err != nil {
		log.Printf("❌ Failed to migrate vehicles: %v", err)
		return err
	}

	log.Println("✅ Migration completed successfully!")
	return nil
}

func migrateUsers() error {
	users := loadUsersFromJSON()
	if len(users) == 0 {
		log.Println("📝 No users to migrate")
		return nil
	}

	for _, user := range users {
		_, err := db.Exec(`
			INSERT INTO users (username, password, role) 
			VALUES ($1, $2, $3) 
			ON CONFLICT (username) DO UPDATE SET 
				password = EXCLUDED.password,
				role = EXCLUDED.role
		`, user.Username, user.Password, user.Role)
		
		if err != nil {
			return fmt.Errorf("failed to insert user %s: %w", user.Username, err)
		}
	}

	log.Printf("✅ Migrated %d users", len(users))
	return nil
}

func migrateBuses() error {
	buses := loadBusesFromJSON()
	if len(buses) == 0 {
		log.Println("📝 No buses to migrate")
		return nil
	}

	for _, bus := range buses {
		_, err := db.Exec(`
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
			return fmt.Errorf("failed to insert bus %s: %w", bus.BusID, err)
		}
	}

	log.Printf("✅ Migrated %d buses", len(buses))
	return nil
}

func migrateRoutes() error {
	routes, err := loadJSON[Route]("data/routes.json")
	if err != nil {
		log.Println("📝 No routes to migrate")
		return nil
	}

	for _, route := range routes {
		positionsJSON, _ := json.Marshal(route.Positions)
		
		_, err := db.Exec(`
			INSERT INTO routes (route_id, route_name, positions) 
			VALUES ($1, $2, $3) 
			ON CONFLICT (route_id) DO UPDATE SET 
				route_name = EXCLUDED.route_name,
				positions = EXCLUDED.positions
		`, route.RouteID, route.RouteName, positionsJSON)
		
		if err != nil {
			return fmt.Errorf("failed to insert route %s: %w", route.RouteID, err)
		}
	}

	log.Printf("✅ Migrated %d routes", len(routes))
	return nil
}

func migrateStudents() error {
	students := loadStudentsFromJSON()
	if len(students) == 0 {
		log.Println("📝 No students to migrate")
		return nil
	}

	for _, student := range students {
		locationsJSON, _ := json.Marshal(student.Locations)
		
		_, err := db.Exec(`
			INSERT INTO students (student_id, name, locations, phone_number, alt_phone_number, 
				guardian, pickup_time, dropoff_time, position_number, route_id, driver, active) 
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12) 
			ON CONFLICT (student_id) DO UPDATE SET 
				name = EXCLUDED.name,
				locations = EXCLUDED.locations,
				phone_number = EXCLUDED.phone_number,
				alt_phone_number = EXCLUDED.alt_phone_number,
				guardian = EXCLUDED.guardian,
				pickup_time = EXCLUDED.pickup_time,
				dropoff_time = EXCLUDED.dropoff_time,
				position_number = EXCLUDED.position_number,
				route_id = EXCLUDED.route_id,
				driver = EXCLUDED.driver,
				active = EXCLUDED.active
		`, student.StudentID, student.Name, locationsJSON, student.PhoneNumber, student.AltPhoneNumber,
		   student.Guardian, student.PickupTime, student.DropoffTime, student.PositionNumber, 
		   student.RouteID, student.Driver, student.Active)
		
		if err != nil {
			return fmt.Errorf("failed to insert student %s: %w", student.StudentID, err)
		}
	}

	log.Printf("✅ Migrated %d students", len(students))
	return nil
}

func migrateRouteAssignments() error {
	assignments, err := loadRouteAssignmentsFromJSON()
	if err != nil {
		log.Println("📝 No route assignments to migrate")
		return nil
	}

	for _, assignment := range assignments {
		_, err := db.Exec(`
			INSERT INTO route_assignments (driver, bus_id, route_id, route_name, assigned_date) 
			VALUES ($1, $2, $3, $4, $5) 
			ON CONFLICT (driver) DO UPDATE SET 
				bus_id = EXCLUDED.bus_id,
				route_id = EXCLUDED.route_id,
				route_name = EXCLUDED.route_name,
				assigned_date = EXCLUDED.assigned_date
		`, assignment.Driver, assignment.BusID, assignment.RouteID, assignment.RouteName, assignment.AssignedDate)
		
		if err != nil {
			return fmt.Errorf("failed to insert assignment for driver %s: %w", assignment.Driver, err)
		}
	}

	log.Printf("✅ Migrated %d route assignments", len(assignments))
	return nil
}

func migrateDriverLogs() error {
	logs, err := loadJSON[DriverLog]("data/driver_logs.json")
	if err != nil {
		log.Println("📝 No driver logs to migrate")
		return nil
	}

	for _, driverLog := range logs {  // FIXED: renamed from 'log' to 'driverLog'
		attendanceJSON, _ := json.Marshal(driverLog.Attendance)
		
		_, err := db.Exec(`
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
			return fmt.Errorf("failed to insert driver log for %s: %w", driverLog.Driver, err)
		}
	}

	log.Printf("✅ Migrated %d driver logs", len(logs))
	return nil
}

func migrateMaintenanceLogs() error {
	logs, _ := loadJSON[MaintenanceLog]("data/maintenance.json")
	if len(logs) == 0 {
		log.Println("📝 No maintenance logs to migrate")
		return nil
	}

	for _, maintLog := range logs {  // FIXED: renamed from 'log' to 'maintLog'
		_, err := db.Exec(`
			INSERT INTO maintenance_logs (bus_id, date, category, notes, mileage) 
			VALUES ($1, $2, $3, $4, $5)
		`, maintLog.BusID, maintLog.Date, maintLog.Category, maintLog.Notes, maintLog.Mileage)
		
		if err != nil {
			return fmt.Errorf("failed to insert maintenance log for bus %s: %w", maintLog.BusID, err)
		}
	}

	log.Printf("✅ Migrated %d maintenance logs", len(logs))
	return nil
}

func migrateVehicles() error {
	vehicles := loadVehiclesFromJSON()
	if len(vehicles) == 0 {
		log.Println("📝 No vehicles to migrate")
		return nil
	}

	for _, vehicle := range vehicles {
		_, err := db.Exec(`
			INSERT INTO vehicles (vehicle_id, model, description, year, tire_size, license, 
				oil_status, tire_status, status, maintenance_notes) 
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) 
			ON CONFLICT (vehicle_id) DO UPDATE SET 
				model = EXCLUDED.model,
				description = EXCLUDED.description,
				year = EXCLUDED.year,
				tire_size = EXCLUDED.tire_size,
				license = EXCLUDED.license,
				oil_status = EXCLUDED.oil_status,
				tire_status = EXCLUDED.tire_status,
				status = EXCLUDED.status,
				maintenance_notes = EXCLUDED.maintenance_notes
		`, vehicle.VehicleID, vehicle.Model, vehicle.Description, vehicle.Year, vehicle.TireSize,
		   vehicle.License, vehicle.OilStatus, vehicle.TireStatus, vehicle.Status, vehicle.MaintenanceNotes)
		
		if err != nil {
			return fmt.Errorf("failed to insert vehicle %s: %w", vehicle.VehicleID, err)
		}
	}

	log.Printf("✅ Migrated %d vehicles", len(vehicles))
	return nil
}

// Setup database with migration
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
// LOAD/SAVE FUNCTIONS (PostgreSQL + JSON Fallback)
// =============================================================================

var userCache []User
var userCacheTime time.Time
var userCacheMutex sync.RWMutex

// Load users - PostgreSQL first, JSON fallback
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

// JSON fallback for users
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

// Save users - PostgreSQL first, JSON fallback
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

// Load buses - PostgreSQL first, JSON fallback
func loadBuses() []*Bus {
	if db != nil {
		rows, err := db.Query(`SELECT bus_id, status, model, capacity, oil_status, tire_status, maintenance_notes 
			FROM buses ORDER BY bus_id`)
		if err != nil {
			log.Printf("Error loading buses from database: %v", err)
			return loadBusesFromJSON()
		}
		defer rows.Close()

		var buses []*Bus
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
		return buses
	}
	return loadBusesFromJSON()
}

func loadBusesFromJSON() []*Bus {
	f, err := os.Open("data/buses.json")
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("buses.json not found, returning empty slice")
			return []*Bus{}
		}
		log.Printf("Error opening buses.json: %v", err)
		return []*Bus{}
	}
	defer f.Close()

	var buses []*Bus
	decoder := json.NewDecoder(f)
	if err := decoder.Decode(&buses); err != nil {
		log.Printf("Error decoding buses.json: %v", err)
		return []*Bus{}
	}
	return buses
}

// Save buses - PostgreSQL first, JSON fallback
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

// Load routes - PostgreSQL first, JSON fallback
func loadRoutes() ([]Route, error) {
	if db != nil {
		rows, err := db.Query("SELECT route_id, route_name, positions FROM routes ORDER BY route_id")
		if err != nil {
			return loadJSON[Route]("data/routes.json") // Fallback
		}
		defer rows.Close()

		var routes []Route
		for rows.Next() {
			var route Route
			var positionsJSON []byte
			err := rows.Scan(&route.RouteID, &route.RouteName, &positionsJSON)
			if err != nil {
				log.Printf("Error scanning route: %v", err)
				continue
			}

			// Parse positions JSON
			err = json.Unmarshal(positionsJSON, &route.Positions)
			if err != nil {
				log.Printf("Error parsing positions JSON: %v", err)
				continue
			}

			routes = append(routes, route)
		}
		return routes, nil
	}
	return loadJSON[Route]("data/routes.json")
}

// Load route assignments - PostgreSQL first, JSON fallback
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

// Save route assignments - PostgreSQL first, JSON fallback
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

// Load driver logs - PostgreSQL first, JSON fallback
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
			var driverLog DriverLog  // FIXED: renamed from 'log' to 'driverLog'
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

// Save driver logs - PostgreSQL first, JSON fallback
func saveDriverLogs(logs []DriverLog) error {
	if db != nil {
		tx, err := db.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback()

		for _, driverLog := range logs {  // FIXED: renamed from 'log' to 'driverLog'
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

// Load students - PostgreSQL first, JSON fallback
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

// Save students - PostgreSQL first, JSON fallback
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

// Load maintenance logs - PostgreSQL first, JSON fallback
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
			var maintLog MaintenanceLog  // FIXED: renamed from 'log' to 'maintLog'
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

// Save maintenance logs - PostgreSQL first, JSON fallback
func saveMaintenanceLogs(logs []MaintenanceLog) error {
	if db != nil {
		tx, err := db.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback()

		for _, maintLog := range logs {  // FIXED: renamed from 'log' to 'maintLog'
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

// Load vehicles - PostgreSQL first, JSON fallback
func loadVehicles() []Vehicle {
	if db != nil {
		rows, err := db.Query(`SELECT vehicle_id, model, description, year, tire_size, license, 
			oil_status, tire_status, status, maintenance_notes FROM vehicles ORDER BY vehicle_id`)
		if err != nil {
			log.Printf("Error loading vehicles from database: %v", err)
			return loadVehiclesFromJSON()
		}
		defer rows.Close()

		var vehicles []Vehicle
		for rows.Next() {
			var vehicle Vehicle
			err := rows.Scan(&vehicle.VehicleID, &vehicle.Model, &vehicle.Description, &vehicle.Year,
				&vehicle.TireSize, &vehicle.License, &vehicle.OilStatus, &vehicle.TireStatus,
				&vehicle.Status, &vehicle.MaintenanceNotes)
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

// Save vehicles - PostgreSQL first, JSON fallback
func saveVehicles(vehicles []Vehicle) error {
	if db != nil {
		tx, err := db.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback()

		for _, vehicle := range vehicles {
			_, err = tx.Exec(`
				INSERT INTO vehicles (vehicle_id, model, description, year, tire_size, license, 
					oil_status, tire_status, status, maintenance_notes) 
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) 
				ON CONFLICT (vehicle_id) DO UPDATE SET 
					model = EXCLUDED.model,
					description = EXCLUDED.description,
					year = EXCLUDED.year,
					tire_size = EXCLUDED.tire_size,
					license = EXCLUDED.license,
					oil_status = EXCLUDED.oil_status,
					tire_status = EXCLUDED.tire_status,
					status = EXCLUDED.status,
					maintenance_notes = EXCLUDED.maintenance_notes
			`, vehicle.VehicleID, vehicle.Model, vehicle.Description, vehicle.Year, vehicle.TireSize,
			   vehicle.License, vehicle.OilStatus, vehicle.TireStatus, vehicle.Status, vehicle.MaintenanceNotes)
			
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
// HELPER FUNCTIONS (Keep existing)
// =============================================================================

// Helper function to safely execute templates
func executeTemplate(w http.ResponseWriter, name string, data interface{}) {
	if err := templates.ExecuteTemplate(w, name, data); err != nil {
		log.Printf("Error executing template %s: %v", name, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func ensureDataFiles() {
	os.MkdirAll("data", os.ModePerm)
	if _, err := os.Stat("data/users.json"); os.IsNotExist(err) {
		defaultUsers := []User{{"admin", "adminpass", "manager"}}
		f, _ := os.Create("data/users.json")
		json.NewEncoder(f).Encode(defaultUsers)
		f.Close()
	}
	if _, err := os.Stat("data/route_assignments.json"); os.IsNotExist(err) {
		f, _ := os.Create("data/route_assignments.json")
		json.NewEncoder(f).Encode([]RouteAssignment{})
		f.Close()
	}
}

func loadJSON[T any](filename string) ([]T, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var data []T
	err = json.NewDecoder(f).Decode(&data)
	return data, err
}

func seedJSON[T any](path string, defaultData T) error {
	if _, err := os.Stat(path); err == nil {
		return nil // already present
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("stat %s: %w", path, err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return fmt.Errorf("create: %w", err)
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(defaultData); err != nil {
		return fmt.Errorf("encode: %w", err)
	}
	log.Printf("Seeded %s", path)
	return nil
}

// getDriverRouteAssignment returns the current route assignment for a driver
func getDriverRouteAssignment(driverUsername string) (*RouteAssignment, error) {
	assignments, err := loadRouteAssignments()
	if err != nil {
		return nil, fmt.Errorf("failed to load assignments: %w", err)
	}

	for _, assignment := range assignments {
		if assignment.Driver == driverUsername {
			return &assignment, nil
		}
	}

	return nil, fmt.Errorf("no assignment found for driver %s", driverUsername)
}

// validateRouteAssignment checks if a route assignment is valid
func validateRouteAssignment(assignment RouteAssignment) error {
	if assignment.Driver == "" {
		return fmt.Errorf("driver cannot be empty")
	}
	if assignment.BusID == "" {
		return fmt.Errorf("bus ID cannot be empty")
	}
	if assignment.RouteID == "" {
		return fmt.Errorf("route ID cannot be empty")
	}

	// Check if driver exists
	users := loadUsers()
	driverExists := false
	for _, u := range users {
		if u.Username == assignment.Driver && u.Role == "driver" {
			driverExists = true
			break
		}
	}
	if !driverExists {
		return fmt.Errorf("driver %s does not exist", assignment.Driver)
	}

	// Check if bus exists and is active
	buses := loadBuses()
	busExists := false
	for _, b := range buses {
		if b.BusID == assignment.BusID {
			if b.Status != "active" {
				return fmt.Errorf("bus %s is not active", assignment.BusID)
			}
			busExists = true
			break
		}
	}
	if !busExists {
		return fmt.Errorf("bus %s does not exist", assignment.BusID)
	}

	// Check if route exists
	routes, err := loadRoutes()
	if err != nil {
		return fmt.Errorf("failed to load routes: %w", err)
	}
	routeExists := false
	for _, r := range routes {
		// Check both RouteID and RouteName for flexibility
		if r.RouteID == assignment.RouteID || r.RouteName == assignment.RouteName {
			routeExists = true
			break
		}
	}
	if !routeExists {
		return fmt.Errorf("route %s does not exist", assignment.RouteID)
	}

	return nil
}

func getUserFromSession(r *http.Request) *User {
	cookie, err := r.Cookie("session_user")
	if err != nil {
		return nil
	}
	uname := cookie.Value
	for _, u := range loadUsers() {
		if u.Username == uname {
			return &u
		}
	}
	return nil
}

// =============================================================================
// HTTP HANDLERS (Keep all existing handlers)
// =============================================================================

func newUserPage(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	if r.Method == http.MethodPost {
		r.ParseForm()
		username := r.FormValue("username")
		password := r.FormValue("password")
		role := r.FormValue("role")

		users := loadUsers()
		users = append(users, User{Username: username, Password: password, Role: role})

		if err := saveUsers(users); err != nil {
			log.Printf("Error saving users: %v", err)
			http.Error(w, "Unable to save user", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/manager-dashboard", http.StatusFound)
		return
	}

	executeTemplate(w, "new_user.html", nil)
}

func editUserPage(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	username := r.URL.Query().Get("username")
	if username == "" {
		http.Error(w, "Username required", http.StatusBadRequest)
		return
	}

	if r.Method == http.MethodPost {
		r.ParseForm()
		newPassword := r.FormValue("password")
		newRole := r.FormValue("role")

		users := loadUsers()
		for i, u := range users {
			if u.Username == username {
				users[i].Password = newPassword
				users[i].Role = newRole
				break
			}
		}

		if err := saveUsers(users); err != nil {
			http.Error(w, "Failed to save user", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/manager-dashboard", http.StatusFound)
		return
	}

	// Find user to edit
	users := loadUsers()
	var editUser *User
	for _, u := range users {
		if u.Username == username {
			editUser = &u
			break
		}
	}

	if editUser == nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	executeTemplate(w, "edit_user.html", editUser)
}

func managerDashboard(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	// Load all data
	driverLogs, _ := loadDriverLogs()
	activities, _ := loadJSON[Activity]("data/activities.json")
	users := loadUsers()
	routes, _ := loadRoutes()
	buses := loadBuses()
	assignments, _ := loadRouteAssignments()

	// Initialize data structures
	driverData := make(map[string]*DriverSummary)
	routeData := make(map[string]*RouteStats)
	now := time.Now()

	// Pre-populate all known drivers
	for _, u := range users {
		if u.Role == "driver" {
			driverData[u.Username] = &DriverSummary{Name: u.Username}
		}
	}

	// Pre-populate all routes
	for _, r := range routes {
		routeData[r.RouteName] = &RouteStats{RouteName: r.RouteName}
	}

	// Process driver logs
	for _, driverLog := range driverLogs {  // FIXED: renamed from 'log' to 'driverLog'
		// Get or create driver summary
		s := driverData[driverLog.Driver]
		if s == nil {
			s = &DriverSummary{Name: driverLog.Driver}
			driverData[driverLog.Driver] = s
		}

		// Add mileage
		s.TotalMiles += driverLog.Mileage

		// Calculate attendance from log
		presentCount := 0
		for _, att := range driverLog.Attendance {
			if att.Present {
				presentCount++
			}
		}

		// Add to morning/evening totals based on period
		if driverLog.Period == "morning" {
			s.TotalMorning += presentCount
		} else if driverLog.Period == "evening" {
			s.TotalEvening += presentCount
		}

		// Parse date for time-based calculations
		parsed, err := time.Parse("2006-01-02", driverLog.Date)
		if err == nil {
			// Monthly calculations
			if parsed.Month() == now.Month() && parsed.Year() == now.Year() {
				s.MonthlyAttendance += presentCount
				s.MonthlyAvgMiles += driverLog.Mileage
			}

			// Find route name for this log
			var routeName string

			// First try to match by RouteID directly
			for _, r := range routes {
				if r.RouteID == driverLog.RouteID {
					routeName = r.RouteName
					break
				}
			}

			// If not found, try to get from driver's assignment
			if routeName == "" {
				for _, assignment := range assignments {
					if assignment.Driver == driverLog.Driver {
						routeName = assignment.RouteName
						break
					}
				}
			}

			// If still not found, check if it's a numeric ID that needs mapping
			if routeName == "" {
				for _, r := range routes {
					if (driverLog.RouteID == "1" && r.RouteID == "1") ||
						 (driverLog.RouteID == "2" && r.RouteID == "2") ||
						 (driverLog.RouteID == "3" && r.RouteID == "3") ||
						 (driverLog.RouteID == "4" && r.RouteID == "4") ||
						 (driverLog.RouteID == "5" && r.RouteID == "5") ||
						 (driverLog.RouteID == "6" && r.RouteID == "6") {
						routeName = r.RouteName
						break
					}
				}
			}

			// Update route statistics if we found a route
			if routeName != "" {
				route := routeData[routeName]
				if route == nil {
					route = &RouteStats{RouteName: routeName}
					routeData[routeName] = route
				}

				route.TotalMiles += driverLog.Mileage
				route.AttendanceMonth += presentCount

				// Time-based attendance (last 24 hours, last 7 days)
				if now.Sub(parsed).Hours() < 24 {
					route.AttendanceDay += presentCount
				}
				if now.Sub(parsed).Hours() < 168 { // 7 days
					route.AttendanceWeek += presentCount
				}
			}
		}
	}

	// Calculate averages for drivers
	for _, s := range driverData {
		if s.MonthlyAvgMiles > 0 {
			daysInMonth := float64(now.Day())
			if daysInMonth > 0 {
				s.MonthlyAvgMiles = s.MonthlyAvgMiles / daysInMonth
			}
		}
	}

	// Calculate averages for routes
	for _, r := range routeData {
		if r.TotalMiles > 0 {
			// Count logs for this route to calculate average
			logCount := 0
			for _, driverLog := range driverLogs {  // FIXED: renamed from 'log' to 'driverLog'
				// Find route name for this log (same logic as above)
				var logRouteName string
				for _, route := range routes {
					if route.RouteID == driverLog.RouteID {
						logRouteName = route.RouteName
						break
					}
				}
				if logRouteName == "" {
					for _, assignment := range assignments {
						if assignment.Driver == driverLog.Driver {
							logRouteName = assignment.RouteName
							break
						}
					}
				}
				if logRouteName == r.RouteName {
					logCount++
				}
			}
			if logCount > 0 {
				r.AvgMiles = r.TotalMiles / float64(logCount)
			}
		}
	}

	// Convert maps to slices for template
	driverSummaries := []*DriverSummary{}
	for _, v := range driverData {
		driverSummaries = append(driverSummaries, v)
	}

	routeStats := []*RouteStats{}
	for _, v := range routeData {
		routeStats = append(routeStats, v)
	}

	data := DashboardData{
		User:            user,
		Role:            user.Role,
		DriverSummaries: driverSummaries,
		RouteStats:      routeStats,
		Activities:      activities,
		Routes:          routes,
		Users:           users,
		Buses:           buses,
	}

	executeTemplate(w, "dashboard.html", data)
}

func driverProfileHandler(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/driver/")
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	// Lookup logs or summaries for the driver
	logs, _ := loadDriverLogs()
	var driverLogs []DriverLog
	for _, l := range logs {
		if l.Driver == name {
			driverLogs = append(driverLogs, l)
		}
	}

	data := struct {
		User   *User
		Name   string
		Logs   []DriverLog
	}{
		User: user,
		Name: name,
		Logs: driverLogs,
	}

	executeTemplate(w, "driver_profile.html", data)
}

func driverDashboard(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "driver" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	date := r.URL.Query().Get("date")
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}
	period := r.URL.Query().Get("period")
	if period == "" {
		period = "morning"
	}

	routes, _ := loadRoutes()
	logs, _ := loadDriverLogs()

	var driverLog *DriverLog
	for _, logEntry := range logs {
		if logEntry.Driver == user.Username && logEntry.Date == date && logEntry.Period == period {
			driverLog = &logEntry
			break
		}
	}

	type PageData struct {
		User      *User
		Date      string
		Period    string
		Route     *Route
		DriverLog *DriverLog
		Bus       *Bus
	}

	var driverRoute *Route
	var assignedBus *Bus

	// Get the driver's current assignment
	assignment, err := getDriverRouteAssignment(user.Username)
	if err != nil {
		log.Printf("Warning: No assignment found for driver %s: %v", user.Username, err)
		// Continue without assignment - driver might not be assigned yet
	}

	// Load all buses
	buses := loadBuses()

	// Find the route and bus based on assignment or existing log
	if assignment != nil {
		// Use assignment data (preferred)
		for _, r := range routes {
			// Try exact match first, then try by route name
			if r.RouteID == assignment.RouteID || r.RouteName == assignment.RouteName {
				driverRoute = &r
				break
			}
		}

		for _, b := range buses {
			if b.BusID == assignment.BusID {
				assignedBus = b
				break
			}
		}
	} else if driverLog != nil {
		// Fall back to log data if no assignment
		for _, r := range routes {
			if r.RouteID == driverLog.RouteID {
				driverRoute = &r
				break
			}
		}

		for _, b := range buses {
			if b.BusID == driverLog.BusID {
				assignedBus = b
				break
			}
		}
	}

	// If we still don't have a route but have an assignment, let's also check by converting route ID
	if driverRoute == nil && assignment != nil {
		// Try to match by numeric ID conversion (e.g., "1" -> "RT001")
		for _, r := range routes {
			if assignment.RouteID == "1" && r.RouteID == "RT001" ||
				 assignment.RouteID == "2" && r.RouteID == "RT002" ||
				 assignment.RouteID == "3" && r.RouteID == "RT003" ||
				 assignment.RouteID == "4" && r.RouteID == "RT004" ||
				 assignment.RouteID == "5" && r.RouteID == "RT005" ||
				 assignment.RouteID == "6" && r.RouteID == "RT006" {
				driverRoute = &r
				break
			}
		}
	}

	// Load students and filter for this driver's active students on this route
	students := loadStudents()
	var activeStudentPositions []struct {
		Position int    `json:"position"`
		Student  string `json:"student"`
	}

	if driverRoute != nil {
		// Create a map of active students for this driver and route
		activeStudentMap := make(map[int]string)
		for _, student := range students {
			if student.Active && student.Driver == user.Username && 
				 (student.RouteID == driverRoute.RouteID || (assignment != nil && student.RouteID == assignment.RouteID)) {
				activeStudentMap[student.PositionNumber] = student.Name
			}
		}

		// Filter route positions to only include active students
		for _, position := range driverRoute.Positions {
			if studentName, exists := activeStudentMap[position.Position]; exists {
				activeStudentPositions = append(activeStudentPositions, struct {
					Position int    `json:"position"`
					Student  string `json:"student"`
				}{
					Position: position.Position,
					Student:  studentName,
				})
			}
		}

		// Update the route with filtered positions
		if len(activeStudentPositions) > 0 {
			filteredRoute := *driverRoute
			filteredRoute.Positions = activeStudentPositions
			driverRoute = &filteredRoute
		} else {
			// If no active students, create empty route with same metadata
			filteredRoute := *driverRoute
			filteredRoute.Positions = []struct {
				Position int    `json:"position"`
				Student  string `json:"student"`
			}{}
			driverRoute = &filteredRoute
		}
	}

	data := PageData{
		User:      user,
		Date:      date,
		Period:    period,
		Route:     driverRoute,
		DriverLog: driverLog,
		Bus:       assignedBus,
	}

	if driverRoute == nil && assignment != nil {
		log.Printf("Warning: No route found for route ID %s", assignment.RouteID)
	}
	if assignedBus == nil && assignment != nil {
		log.Printf("Warning: No bus found for bus ID %s", assignment.BusID)
	}

	executeTemplate(w, "driver_dashboard.html", data)
}

func loginPage(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		r.ParseForm()
		username := r.FormValue("username")
		password := r.FormValue("password")
		for _, u := range loadUsers() {
			if u.Username == username && u.Password == password {
				http.SetCookie(w, &http.Cookie{
					Name:  "session_user",
					Value: username,
					Path:  "/",
				})

				if u.Role == "manager" {
					http.Redirect(w, r, "/manager-dashboard", http.StatusFound)
				} else if u.Role == "driver" {
					http.Redirect(w, r, "/driver-dashboard", http.StatusFound)
				} else {
					http.Redirect(w, r, "/", http.StatusFound)
				}
				return
			}
		}
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}
	executeTemplate(w, "login.html", nil)
}

func pullLatest() string {
	repo, err := git.PlainOpen(".")
	if err != nil {
		return "❌ Failed to open repo: " + err.Error()
	}

	w, err := repo.Worktree()
	if err != nil {
		return "❌ Failed to get worktree: " + err.Error()
	}

	err = w.Pull(&git.PullOptions{
		RemoteName: "origin",
		Auth:       nil, // Add credentials if needed
		Force:      true,
	})
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return "❌ Git pull failed: " + err.Error()
	}
	return "✅ Git pull complete"
}

func runPullHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST allowed", http.StatusMethodNotAllowed)
		return
	}

	if r.Header.Get("x-trigger-source") != "cloudflare" {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	output := pullLatest()

	go func() {
		time.Sleep(1 * time.Second)
		exec.Command("bash", "restart_app.sh").Run()
	}()

	w.Write([]byte("✅ Git pulled and app restarted\n" + output))
}

func handleWebhook(w http.ResponseWriter, r *http.Request) {
	// Optional: Validate GitHub signature
	cmd := exec.Command("git", "pull", "origin", "main")
	output, err := cmd.CombinedOutput()
	if err != nil {
		http.Error(w, "Git pull failed: "+err.Error(), 500)
		return
	}
	exec.Command("kill", "1").Run() // triggers a Replit restart
	fmt.Fprintf(w, "Updated:\n%s", string(output))
}

func saveDriverLog(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	user := getUserFromSession(r)
	if user == nil || user.Role != "driver" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	r.ParseForm()
	date := r.FormValue("date")
	period := r.FormValue("period")
	busID := r.FormValue("bus_id")  // Changed from bus_number to bus_id
	departure := r.FormValue("departure")
	arrival := r.FormValue("arrival")
	mileage, _ := strconv.ParseFloat(r.FormValue("mileage"), 64)

	// Get the driver's route assignment
	assignment, err := getDriverRouteAssignment(user.Username)
	if err != nil {
		log.Printf("Error getting driver assignment: %v", err)
		http.Error(w, "No route assignment found", http.StatusBadRequest)
		return
	}

	// Validate that the bus ID matches the assignment
	if busID != assignment.BusID {
		log.Printf("Bus ID mismatch: form=%s, assignment=%s", busID, assignment.BusID)
		http.Error(w, "Bus ID does not match assignment", http.StatusBadRequest)
		return
	}

	// Load route to get positions
	routes, err := loadRoutes()
	if err != nil {
		log.Printf("Error loading routes: %v", err)
		http.Error(w, "Unable to load routes", http.StatusInternalServerError)
		return
	}

	var positions []struct {
		Position int    `json:"position"`
		Student  string `json:"student"`
	}

	// Find the correct route using RouteID from assignment
	for _, rt := range routes {
		// Try exact match first, then by route name, then by ID mapping
		if rt.RouteID == assignment.RouteID || rt.RouteName == assignment.RouteName ||
			 (assignment.RouteID == "1" && rt.RouteID == "RT001") ||
			 (assignment.RouteID == "2" && rt.RouteID == "RT002") ||
			 (assignment.RouteID == "3" && rt.RouteID == "RT003") ||
			 (assignment.RouteID == "4" && rt.RouteID == "RT004") ||
			 (assignment.RouteID == "5" && rt.RouteID == "RT005") ||
			 (assignment.RouteID == "6" && rt.RouteID == "RT006") {
			positions = rt.Positions
			break
		}
	}

	// Build attendance data
	var attendance []struct {
		Position   int    `json:"position"`
		Present    bool   `json:"present"`
		PickupTime string `json:"pickup_time,omitempty"`
	}

	for _, p := range positions {
		present := r.FormValue("present_"+strconv.Itoa(p.Position)) == "on"
		pickup := r.FormValue("pickup_" + strconv.Itoa(p.Position))
		attendance = append(attendance, struct {
			Position   int    `json:"position"`
			Present    bool   `json:"present"`
			PickupTime string `json:"pickup_time,omitempty"`
		}{p.Position, present, pickup})
	}

	// Load existing logs
	logs, err := loadDriverLogs()
	if err != nil {
		log.Printf("Error loading driver logs: %v", err)
		// Continue with empty slice if file doesn't exist
		logs = []DriverLog{}
	}

	// Check if we're updating an existing log
	updated := false
	for i := range logs {
		if logs[i].Driver == user.Username && logs[i].Date == date && logs[i].Period == period {
			logs[i].BusID = busID
			logs[i].RouteID = assignment.RouteID
			logs[i].Departure = departure
			logs[i].Arrival = arrival
			logs[i].Mileage = mileage
			logs[i].Attendance = attendance
			updated = true
			break
		}
	}

	// If not updating, create new log entry
	if !updated {
		logs = append(logs, DriverLog{
			Driver:     user.Username,
			BusID:      busID,
			RouteID:    assignment.RouteID,
			Date:       date,
			Period:     period,
			Departure:  departure,
			Arrival:    arrival,
			Mileage:    mileage,
			Attendance: attendance,
		})
	}

	// Save the logs
	if err := saveDriverLogs(logs); err != nil {
		log.Printf("Error saving driver logs: %v", err)
		http.Error(w, "Unable to save log", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/driver-dashboard?date="+date+"&period="+period, http.StatusSeeOther)
}

func dashboardRouter(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	if user.Role == "manager" {
		managerDashboard(w, r)
	} else if user.Role == "driver" {
		driverDashboard(w, r)
	} else {
		http.Redirect(w, r, "/", http.StatusFound)
	}
}

func assignRoutesPage(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	assignments, _ := loadRouteAssignments()
	routes, _ := loadRoutes()
	users := loadUsers()
	buses := loadBuses()

	// Filter drivers only
	var drivers []User
	for _, u := range users {
		if u.Role == "driver" {
			drivers = append(drivers, u)
		}
	}

	// Find assigned items
	assignedRouteIDs := make(map[string]bool)
	assignedBusIDs := make(map[string]bool)
	assignedDrivers := make(map[string]bool)
	for _, a := range assignments {
		assignedRouteIDs[a.RouteID] = true
		assignedBusIDs[a.BusID] = true
		assignedDrivers[a.Driver] = true
	}

	// Filter available routes (not assigned)
	var availableRoutes []Route
	for _, route := range routes {
		if !assignedRouteIDs[route.RouteID] {
			availableRoutes = append(availableRoutes, route)
		}
	}

	// Filter available buses (active and not assigned)
	var availableBuses []*Bus
	for _, bus := range buses {
		if bus.Status == "active" && !assignedBusIDs[bus.BusID] {
			availableBuses = append(availableBuses, bus)
		}
	}

	// Filter available drivers (not assigned)
	var availableDrivers []User
	for _, driver := range drivers {
		if !assignedDrivers[driver.Username] {
			availableDrivers = append(availableDrivers, driver)
		}
	}

	data := AssignRouteData{
		User:            user,
		Assignments:     assignments,
		Drivers:         availableDrivers,
		AvailableRoutes: availableRoutes,
		AvailableBuses:  availableBuses,
	}

	executeTemplate(w, "assign_routes.html", data)
}

func assignRoute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	r.ParseForm()
	driver := r.FormValue("driver")
	busID := r.FormValue("bus_id")  // Changed from bus_number
	routeID := r.FormValue("route_id")

	if driver == "" || busID == "" || routeID == "" {
		http.Error(w, "All fields are required", http.StatusBadRequest)
		return
	}

	// Find route name
	routes, err := loadRoutes()
	if err != nil {
		log.Printf("Error loading routes: %v", err)
		http.Error(w, "Unable to load routes", http.StatusInternalServerError)
		return
	}

	var routeName string
	routeFound := false
	for _, rt := range routes {
		if rt.RouteID == routeID {
			routeName = rt.RouteName
			routeFound = true
			break
		}
	}

	if !routeFound {
		http.Error(w, "Route not found", http.StatusBadRequest)
		return
	}

	// Verify bus exists and is active
	buses := loadBuses()
	busFound := false
	for _, bus := range buses {
		if bus.BusID == busID {
			if bus.Status != "active" {
				http.Error(w, "Bus is not active", http.StatusBadRequest)
				return
			}
			busFound = true
			break
		}
	}

	if !busFound {
		http.Error(w, "Bus not found", http.StatusBadRequest)
		return
	}

	assignments, err := loadRouteAssignments()
	if err != nil {
		log.Printf("Error loading assignments: %v", err)
		assignments = []RouteAssignment{}
	}

	// Check if driver already has an assignment
	for i, a := range assignments {
		if a.Driver == driver {
			// Update existing assignment
			assignments[i].BusID = busID
			assignments[i].RouteID = routeID
			assignments[i].RouteName = routeName
			assignments[i].AssignedDate = time.Now().Format("2006-01-02")

			if err := saveRouteAssignments(assignments); err != nil {
				log.Printf("Error saving assignments: %v", err)
				http.Error(w, "Unable to save assignment", http.StatusInternalServerError)
				return
			}

			http.Redirect(w, r, "/assign-routes", http.StatusFound)
			return
		}
	}

	// Check if route or bus is already assigned
	for _, a := range assignments {
		if a.RouteID == routeID {
			http.Error(w, "Route is already assigned", http.StatusBadRequest)
			return
		}
		if a.BusID == busID {
			http.Error(w, "Bus is already assigned", http.StatusBadRequest)
			return
		}
	}

	// Add new assignment
	newAssignment := RouteAssignment{
		Driver:       driver,
		BusID:        busID,
		RouteID:      routeID,
		RouteName:    routeName,
		AssignedDate: time.Now().Format("2006-01-02"),
	}

	assignments = append(assignments, newAssignment)
	if err := saveRouteAssignments(assignments); err != nil {
		log.Printf("Error saving assignments: %v", err)
		http.Error(w, "Unable to save assignment", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/assign-routes", http.StatusFound)
}

func unassignRoute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	r.ParseForm()
	driver := r.FormValue("driver")
	busID := r.FormValue("bus_id")  // Changed from bus_number

	assignments, err := loadRouteAssignments()
	if err != nil {
		log.Printf("Error loading assignments: %v", err)
		http.Error(w, "Unable to load assignments", http.StatusInternalServerError)
		return
	}

	// Remove assignment
	var newAssignments []RouteAssignment
	found := false
	for _, a := range assignments {
		if !(a.Driver == driver && a.BusID == busID) {
			newAssignments = append(newAssignments, a)
		} else {
			found = true
		}
	}

	if !found {
		http.Error(w, "Assignment not found", http.StatusNotFound)
		return
	}

	if err := saveRouteAssignments(newAssignments); err != nil {
		log.Printf("Error saving assignments: %v", err)
		http.Error(w, "Unable to save assignments", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/assign-routes", http.StatusFound)
}

func fleetPage(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	buses := loadBuses()
	data := FleetData{
		User:  user,
		Buses: buses,
		Today: time.Now().Format("2006-01-02"),
	}

	executeTemplate(w, "fleet.html", data)
}

func addBus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	r.ParseForm()
	busID := r.FormValue("bus_id")  // Changed from bus_number
	status := r.FormValue("status")
	model := r.FormValue("model")
	capacity, _ := strconv.Atoi(r.FormValue("capacity"))
	oilStatus := r.FormValue("oil_status")
	tireStatus := r.FormValue("tire_status")
	maintenanceNotes := r.FormValue("maintenance_notes")

	buses := loadBuses()

	// Check if bus ID already exists
	for _, b := range buses {
		if b.BusID == busID {
			http.Error(w, "Bus ID already exists", http.StatusBadRequest)
			return
		}
	}

	newBus := &Bus{
		BusID:            busID,
		Status:           status,
		Model:            model,
		Capacity:         capacity,
		OilStatus:        oilStatus,
		TireStatus:       tireStatus,
		MaintenanceNotes: maintenanceNotes,
	}

	buses = append(buses, newBus)
	if err := saveBuses(buses); err != nil {
		log.Printf("Error saving buses: %v", err)
		http.Error(w, "Unable to save bus", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/fleet", http.StatusFound)
}

func addRoute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	r.ParseForm()
	routeName := r.FormValue("route_name")
	description := r.FormValue("description")

	if routeName == "" {
		http.Error(w, "Route name is required", http.StatusBadRequest)
		return
	}

	// Load existing routes
	routes, err := loadRoutes()
	if err != nil {
		log.Printf("Error loading routes: %v", err)
		routes = []Route{} // Start with empty if load fails
	}

	// Generate unique route ID
	routeID := fmt.Sprintf("RT%03d", len(routes)+1)

	// Create new route
	newRoute := Route{
		RouteID:     routeID,
		RouteName:   routeName,
		Description: description,
		Positions: []struct {
			Position int    `json:"position"`
			Student  string `json:"student"`
		}{}, // Empty positions initially
	}

	// Add to routes slice
	routes = append(routes, newRoute)

	// Save using your existing save system
	if db != nil {
		// Save to PostgreSQL - you'll need to add description column to your routes table
		positionsJSON, _ := json.Marshal(newRoute.Positions)
		_, err := db.Exec(`
			INSERT INTO routes (route_id, route_name, description, positions) 
			VALUES ($1, $2, $3, $4)
		`, newRoute.RouteID, newRoute.RouteName, newRoute.Description, positionsJSON)
		
		if err != nil {
			log.Printf("Error saving route to database: %v", err)
			http.Error(w, "Unable to save route", http.StatusInternalServerError)
			return
		}
	} else {
		// Fallback to JSON
		f, err := os.Create("data/routes.json")
		if err != nil {
			log.Printf("Error creating routes file: %v", err)
			http.Error(w, "Unable to save route", http.StatusInternalServerError)
			return
		}
		defer f.Close()
		
		enc := json.NewEncoder(f)
		enc.SetIndent("", "  ")
		if err := enc.Encode(routes); err != nil {
			log.Printf("Error encoding routes: %v", err)
			http.Error(w, "Unable to save route", http.StatusInternalServerError)
			return
		}
	}

	http.Redirect(w, r, "/assign-routes", http.StatusSeeOther)
}

func editRoute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
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

	// Load existing routes
	routes, err := loadRoutes()
	if err != nil {
		log.Printf("Error loading routes: %v", err)
		http.Error(w, "Unable to load routes", http.StatusInternalServerError)
		return
	}

	// Find and update the route
	updated := false
	for i, route := range routes {
		if route.RouteID == routeID {
			routes[i].RouteName = routeName
			routes[i].Description = description
			updated = true
			break
		}
	}

	if !updated {
		http.Error(w, "Route not found", http.StatusNotFound)
		return
	}

	// Save using your existing save system
	if db != nil {
		// Save to PostgreSQL
		_, err := db.Exec(`
			UPDATE routes 
			SET route_name = $1, description = $2 
			WHERE route_id = $3
		`, routeName, description, routeID)
		
		if err != nil {
			log.Printf("Error updating route in database: %v", err)
			http.Error(w, "Unable to update route", http.StatusInternalServerError)
			return
		}
	} else {
		// Fallback to JSON
		f, err := os.Create("data/routes.json")
		if err != nil {
			log.Printf("Error creating routes file: %v", err)
			http.Error(w, "Unable to save route", http.StatusInternalServerError)
			return
		}
		defer f.Close()
		
		enc := json.NewEncoder(f)
		enc.SetIndent("", "  ")
		if err := enc.Encode(routes); err != nil {
			log.Printf("Error encoding routes: %v", err)
			http.Error(w, "Unable to save route", http.StatusInternalServerError)
			return
		}
	}

	http.Redirect(w, r, "/assign-routes", http.StatusSeeOther)
}

func deleteRoute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	r.ParseForm()
	routeID := r.FormValue("route_id")

	if routeID == "" {
		http.Error(w, "Route ID is required", http.StatusBadRequest)
		return
	}

	// Check if route is currently assigned
	assignments, err := loadRouteAssignments()
	if err == nil {
		for _, assignment := range assignments {
			if assignment.RouteID == routeID {
				http.Error(w, "Cannot delete route that is currently assigned to a driver", http.StatusBadRequest)
				return
			}
		}
	}

	// Load existing routes
	routes, err := loadRoutes()
	if err != nil {
		log.Printf("Error loading routes: %v", err)
		http.Error(w, "Unable to load routes", http.StatusInternalServerError)
		return
	}

	// Find and remove the route
	var newRoutes []Route
	found := false
	for _, route := range routes {
		if route.RouteID != routeID {
			newRoutes = append(newRoutes, route)
		} else {
			found = true
		}
	}

	if !found {
		http.Error(w, "Route not found", http.StatusNotFound)
		return
	}

	// Save using your existing save system
	if db != nil {
		// Delete from PostgreSQL
		_, err := db.Exec("DELETE FROM routes WHERE route_id = $1", routeID)
		if err != nil {
			log.Printf("Error deleting route from database: %v", err)
			http.Error(w, "Unable to delete route", http.StatusInternalServerError)
			return
		}
	} else {
		// Fallback to JSON
		f, err := os.Create("data/routes.json")
		if err != nil {
			log.Printf("Error creating routes file: %v", err)
			http.Error(w, "Unable to save routes", http.StatusInternalServerError)
			return
		}
		defer f.Close()
		
		enc := json.NewEncoder(f)
		enc.SetIndent("", "  ")
		if err := enc.Encode(newRoutes); err != nil {
			log.Printf("Error encoding routes: %v", err)
			http.Error(w, "Unable to save routes", http.StatusInternalServerError)
			return
		}
	}

	http.Redirect(w, r, "/assign-routes", http.StatusSeeOther)
}

func editBus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	r.ParseForm()
	originalBusID := r.FormValue("original_bus_id")
	busID := r.FormValue("bus_id")
	status := r.FormValue("status")
	model := r.FormValue("model")
	capacity, _ := strconv.Atoi(r.FormValue("capacity"))
	oilStatus := r.FormValue("oil_status")
	tireStatus := r.FormValue("tire_status")
	maintenanceNotes := r.FormValue("maintenance_notes")

	// Debug logging
	log.Printf("EditBus: originalBusID='%s', newBusID='%s', status='%s'", originalBusID, busID, status)

	buses := loadBuses()
	log.Printf("EditBus: loaded %d buses", len(buses))

	// Check if new bus ID conflicts with existing (unless it's the same bus)
	if busID != originalBusID {
		for _, b := range buses {
			if b.BusID == busID {
				http.Error(w, "Bus ID already exists", http.StatusBadRequest)
				return
			}
		}
	}

	// Find the original bus to check status change
	var originalBus *Bus
	for _, b := range buses {
		log.Printf("EditBus: checking bus ID '%s' against original '%s'", b.BusID, originalBusID)
		if b.BusID == originalBusID {
			originalBus = b
			break
		}
	}

	if originalBus == nil {
		log.Printf("EditBus: Bus not found with ID '%s'", originalBusID)
		// List all available bus IDs for debugging
		busIDs := make([]string, len(buses))
		for i, b := range buses {
			busIDs[i] = b.BusID
		}
		log.Printf("EditBus: Available bus IDs: %v", busIDs)
		http.Error(w, fmt.Sprintf("Bus not found with ID '%s'", originalBusID), http.StatusNotFound)
		return
	}

	// Check if status is changing from active to inactive
	statusChangingToInactive := originalBus.Status == "active" && (status == "maintenance" || status == "out_of_service")

	// If status is changing to inactive, check if bus is currently assigned
	if statusChangingToInactive {
		assignments, err := loadRouteAssignments()
		if err == nil {
			for _, assignment := range assignments {
				if assignment.BusID == originalBusID {
					// Bus is assigned to a driver/route, prompt for replacement bus selection
					http.Error(w, "REQUIRES_REPLACEMENT_BUS:"+assignment.Driver+":"+assignment.RouteName, http.StatusConflict)
					return
				}
			}
		}
	}

	updated := false
	for i, b := range buses {
		if b.BusID == originalBusID {
			buses[i].BusID = busID
			buses[i].Status = status
			buses[i].Model = model
			buses[i].Capacity = capacity
			buses[i].OilStatus = oilStatus
			buses[i].TireStatus = tireStatus
			buses[i].MaintenanceNotes = maintenanceNotes
			updated = true
			break
		}
	}

	if !updated {
		http.Error(w, "Bus not found", http.StatusNotFound)
		return
	}

	if err := saveBuses(buses); err != nil {
		log.Printf("Error saving buses: %v", err)
		http.Error(w, "Unable to save bus", http.StatusInternalServerError)
		return
	}

	// Auto-create maintenance log if status changed to maintenance or out_of_service
	if statusChangingToInactive || (status == "maintenance" && originalBus.Status != "maintenance") {
		maintenanceLogs := loadMaintenanceLogs()

		logEntry := MaintenanceLog{
			BusID:    busID,
			Date:     time.Now().Format("2006-01-02"),
			Category: "status_change",
			Notes:    fmt.Sprintf("Bus status changed from '%s' to '%s'. %s", originalBus.Status, status, maintenanceNotes),
			Mileage:  0, // Could be enhanced to track mileage
		}

		maintenanceLogs = append(maintenanceLogs, logEntry)
		if err := saveMaintenanceLogs(maintenanceLogs); err != nil {
			log.Printf("Warning: Failed to save maintenance log: %v", err)
			// Don't fail the bus update for this
		} else {
			log.Printf("Maintenance log created for bus %s status change", busID)
		}
	}

	http.Redirect(w, r, "/fleet", http.StatusFound)
}

func removeBus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	r.ParseForm()
	busID := r.FormValue("bus_id")  // Changed from bus_number

	// Check if bus is currently assigned
	assignments, err := loadRouteAssignments()
	if err == nil {
		for _, a := range assignments {
			if a.BusID == busID {
				http.Error(w, "Cannot remove bus that is currently assigned to a route", http.StatusBadRequest)
				return
			}
		}
	}

	buses := loadBuses()
	var newBuses []*Bus
	found := false
	for _, b := range buses {
		if b.BusID != busID {
			newBuses = append(newBuses, b)
		} else {
			found = true
		}
	}

	if !found {
		http.Error(w, "Bus not found", http.StatusNotFound)
		return
	}

	if err := saveBuses(newBuses); err != nil {
		log.Printf("Error saving buses: %v", err)
		http.Error(w, "Unable to save buses", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/fleet", http.StatusFound)
}

func companyFleetPage(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	vehicles := loadVehicles()
	data := CompanyFleetData{
		User:     user,
		Vehicles: vehicles,
	}

	executeTemplate(w, "company_fleet.html", data)
}

func studentsPage(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "driver" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	students := loadStudents()
	routes, _ := loadRoutes()

	// Filter students for this driver
	var driverStudents []Student
	for _, s := range students {
		if s.Driver == user.Username {
			driverStudents = append(driverStudents, s)
		}
	}

	data := StudentData{
		User:     user,
		Students: driverStudents,
		Routes:   routes,
	}

	executeTemplate(w, "students.html", data)
}

func addStudent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	user := getUserFromSession(r)
	if user == nil || user.Role != "driver" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	r.ParseForm()
	name := r.FormValue("name")
	phoneNumber := r.FormValue("phone_number")
	altPhoneNumber := r.FormValue("alt_phone_number")
	guardian := r.FormValue("guardian")
	pickupTime := r.FormValue("pickup_time")
	dropoffTime := r.FormValue("dropoff_time")
	positionNumber, _ := strconv.Atoi(r.FormValue("position_number"))
	routeID := r.FormValue("route_id")

	// Parse locations
	var locations []Location
	pickupAddresses := r.Form["pickup_address"]
	pickupDescriptions := r.Form["pickup_description"]
	dropoffAddresses := r.Form["dropoff_address"]
	dropoffDescriptions := r.Form["dropoff_description"]

	for i, addr := range pickupAddresses {
		if addr != "" {
			desc := ""
			if i < len(pickupDescriptions) {
				desc = pickupDescriptions[i]
			}
			locations = append(locations, Location{
				Type:        "pickup",
				Address:     addr,
				Description: desc,
			})
		}
	}

	for i, addr := range dropoffAddresses {
		if addr != "" {
			desc := ""
			if i < len(dropoffDescriptions) {
				desc = dropoffDescriptions[i]
			}
			locations = append(locations, Location{
				Type:        "dropoff",
				Address:     addr,
				Description: desc,
			})
		}
	}

	students := loadStudents()

	// Generate student ID
	studentID := fmt.Sprintf("STU_%d", len(students)+1)

	newStudent := Student{
		StudentID:      studentID,
		Name:           name,
		Locations:      locations,
		PhoneNumber:    phoneNumber,
		AltPhoneNumber: altPhoneNumber,
		Guardian:       guardian,
		PickupTime:     pickupTime,
		DropoffTime:    dropoffTime,
		PositionNumber: positionNumber,
		RouteID:        routeID,
		Driver:         user.Username,
		Active:         true,
	}

	students = append(students, newStudent)
	if err := saveStudents(students); err != nil {
		log.Printf("Error saving students: %v", err)
		http.Error(w, "Unable to save student", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/students", http.StatusFound)
}

func editStudent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	user := getUserFromSession(r)
	if user == nil || user.Role != "driver" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	r.ParseForm()
	studentID := r.FormValue("student_id")
	name := r.FormValue("name")
	phoneNumber := r.FormValue("phone_number")
	altPhoneNumber := r.FormValue("alt_phone_number")
	guardian := r.FormValue("guardian")
	pickupTime := r.FormValue("pickup_time")
	dropoffTime := r.FormValue("dropoff_time")
	positionNumber, _ := strconv.Atoi(r.FormValue("position_number"))
	routeID := r.FormValue("route_id")
	active := r.FormValue("active") == "on"

	// Parse locations
	var locations []Location
	pickupAddresses := r.Form["pickup_address"]
	pickupDescriptions := r.Form["pickup_description"]
	dropoffAddresses := r.Form["dropoff_address"]
	dropoffDescriptions := r.Form["dropoff_description"]

	for i, addr := range pickupAddresses {
		if addr != "" {
			desc := ""
			if i < len(pickupDescriptions) {
				desc = pickupDescriptions[i]
			}
			locations = append(locations, Location{
				Type:        "pickup",
				Address:     addr,
				Description: desc,
			})
		}
	}

	for i, addr := range dropoffAddresses {
		if addr != "" {
			desc := ""
			if i < len(dropoffDescriptions) {
				desc = dropoffDescriptions[i]
			}
			locations = append(locations, Location{
				Type:        "dropoff",
				Address:     addr,
				Description: desc,
			})
		}
	}

	students := loadStudents()

	for i, s := range students {
		if s.StudentID == studentID && s.Driver == user.Username {
			students[i].Name = name
			students[i].Locations = locations
			students[i].PhoneNumber = phoneNumber
			students[i].AltPhoneNumber = altPhoneNumber
			students[i].Guardian = guardian
			students[i].PickupTime = pickupTime
			students[i].DropoffTime = dropoffTime
			students[i].PositionNumber = positionNumber
			students[i].RouteID = routeID
			students[i].Active = active
			break
		}
	}

	if err := saveStudents(students); err != nil {
		log.Printf("Error saving students: %v", err)
		http.Error(w, "Unable to save student", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/students", http.StatusFound)
}

func removeStudent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	user := getUserFromSession(r)
	if user == nil || user.Role != "driver" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	r.ParseForm()
	studentID := r.FormValue("student_id")

	students := loadStudents()
	var newStudents []Student
	for _, s := range students {
		if !(s.StudentID == studentID && s.Driver == user.Username) {
			newStudents = append(newStudents, s)
		}
	}

	if err := saveStudents(newStudents); err != nil {
		log.Printf("Error saving students: %v", err)
		http.Error(w, "Unable to save students", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/students", http.StatusFound)
}

func reassignDriverBus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	r.ParseForm()
	driverName := r.FormValue("driver")
	newBusID := r.FormValue("new_bus_id")

	if driverName == "" || newBusID == "" {
		http.Error(w, "Driver and new bus ID are required", http.StatusBadRequest)
		return
	}

	// Load assignments
	assignments, err := loadRouteAssignments()
	if err != nil {
		log.Printf("Error loading assignments: %v", err)
		http.Error(w, "Unable to load assignments", http.StatusInternalServerError)
		return
	}

	// Find and update the driver's assignment
	updated := false
	for i, assignment := range assignments {
		if assignment.Driver == driverName {
			assignments[i].BusID = newBusID
			updated = true
			break
		}
	}

	if !updated {
		http.Error(w, "Driver assignment not found", http.StatusNotFound)
		return
	}

	// Save updated assignments
	if err := saveRouteAssignments(assignments); err != nil {
		log.Printf("Error saving assignments: %v", err)
		http.Error(w, "Unable to save assignment", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Driver reassigned successfully"))
}

// Updated maintenance log function to use BusID
func addMaintenanceLog(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	r.ParseForm()
	mileage, _ := strconv.Atoi(r.FormValue("mileage"))
	logEntry := MaintenanceLog{
		BusID:    r.FormValue("bus_id"), // Changed from bus_number
		Date:     r.FormValue("date"),
		Category: r.FormValue("category"),
		Notes:    r.FormValue("notes"),
		Mileage:  mileage,
	}

	// Validate bus exists
	buses := loadBuses()
	busExists := false
	for _, bus := range buses {
		if bus.BusID == logEntry.BusID {
			busExists = true
			break
		}
	}

	if !busExists {
		http.Error(w, "Bus not found", http.StatusBadRequest)
		return
	}

	logs := loadMaintenanceLogs()
	logs = append(logs, logEntry)
	if err := saveMaintenanceLogs(logs); err != nil {
		log.Printf("Error saving maintenance logs: %v", err)
		http.Error(w, "Unable to save", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/fleet", http.StatusFound)
}

func removeUser(w http.ResponseWriter, r *http.Request) {
	// Accept both GET and POST for debugging
	var usernameToRemove string

	if r.Method == http.MethodGet {
		// Parse from URL query for GET requests
		usernameToRemove = r.URL.Query().Get("username")
		log.Printf("DEBUG: Received GET request for removing user: %s", usernameToRemove)
	} else if r.Method == http.MethodPost {
		// Parse form data for POST requests
		r.ParseForm()
		usernameToRemove = r.FormValue("username")
		log.Printf("DEBUG: Received POST request for removing user: %s", usernameToRemove)
	} else {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check if user is logged in and is a manager
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	// Check if username was provided
	if usernameToRemove == "" {
		http.Error(w, "Username is required", http.StatusBadRequest)
		return
	}

	// Prevent removing yourself
	if usernameToRemove == user.Username {
		http.Error(w, "Cannot remove yourself", http.StatusBadRequest)
		return
	}

	users := loadUsers()
	var newUsers []User
	userFound := false
	for _, u := range users {
		if u.Username != usernameToRemove {
			newUsers = append(newUsers, u)
		} else {
			userFound = true
		}
	}

	if !userFound {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// If removing a driver, also remove their route assignments
	if userFound {
		assignments, err := loadRouteAssignments()
		if err == nil {
			var newAssignments []RouteAssignment
			for _, assignment := range assignments {
				if assignment.Driver != usernameToRemove {
					newAssignments = append(newAssignments, assignment)
				}
			}
			saveRouteAssignments(newAssignments)
		}
	}

	// Save updated users list
	if err := saveUsers(newUsers); err != nil {
		log.Printf("Error saving users: %v", err)
		http.Error(w, "Unable to save users", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/manager-dashboard", http.StatusFound)
}

func updateVehicleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
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

	vehicles := loadVehicles()
	updated := false

	for i, vehicle := range vehicles {
		if vehicle.VehicleID == vehicleID {
			switch statusType {
			case "oil":
				vehicles[i].OilStatus = newStatus
			case "tire":
				vehicles[i].TireStatus = newStatus
			case "status":
				vehicles[i].Status = newStatus
			default:
				http.Error(w, "Invalid status type", http.StatusBadRequest)
				return
			}
			updated = true
			break
		}
	}

	if !updated {
		http.Error(w, "Vehicle not found", http.StatusNotFound)
		return
	}

	// Save updated vehicles
	if err := saveVehicles(vehicles); err != nil {
		log.Printf("Error saving vehicles: %v", err)
		http.Error(w, "Failed to save changes", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Status updated successfully"))
}

func updateBusStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	r.ParseForm()
	busID := r.FormValue("bus_id")
	statusType := r.FormValue("status_type")
	newStatus := r.FormValue("new_status")

	log.Printf("Updating bus %s: %s status to %s", busID, statusType, newStatus)

	if busID == "" || statusType == "" || newStatus == "" {
		http.Error(w, "Missing required parameters", http.StatusBadRequest)
		return
	}

	buses := loadBuses()
	updated := false

	for i, bus := range buses {
		if bus.BusID == busID {
			switch statusType {
			case "oil":
				buses[i].OilStatus = newStatus
			case "tire":
				buses[i].TireStatus = newStatus
			case "status":
				buses[i].Status = newStatus
			default:
				http.Error(w, "Invalid status type", http.StatusBadRequest)
				return
			}
			updated = true
			log.Printf("Updated bus %s: %s status to %s", busID, statusType, newStatus)
			break
		}
	}

	if !updated {
		log.Printf("Bus not found: %s", busID)
		http.Error(w, "Bus not found", http.StatusNotFound)
		return
	}

	// Save updated buses
	if err := saveBuses(buses); err != nil {
		log.Printf("Error saving buses: %v", err)
		http.Error(w, "Failed to save changes", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Status updated successfully"))
}

func logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{Name: "session_user", Value: "", MaxAge: -1, Path: "/"})
	http.Redirect(w, r, "/", http.StatusFound)
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}

func rootHealthCheck(w http.ResponseWriter, r *http.Request) {
	// Always show login page for root path
	loginPage(w, r)
}

func withRecovery(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		defer func() {
			duration := time.Since(start)
			log.Printf("%s %s - %v", r.Method, r.URL.Path, duration)
			if err := recover(); err != nil {
				log.Printf("Recovered from panic in handler %s: %v", r.URL.Path, err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
			}
		}()
		h(w, r)
	}
}

// Updated initialization with proper ID structure
func initDataFiles() {
	// Ensure data directory exists with proper permissions
	if err := os.MkdirAll("data", 0755); err != nil {
		log.Printf("Warning: failed to create data directory: %v", err)
		return
	}

	// Create buses.json if it doesn't exist, and seed with ID-based data
	if _, err := os.Stat("data/buses.json"); os.IsNotExist(err) {
		defaultBuses := []*Bus{
			{BusID: "BUS001", Status: "active", Model: "Ford Transit", Capacity: 20, OilStatus: "good", TireStatus: "good", MaintenanceNotes: ""},
			{BusID: "BUS002", Status: "active", Model: "Chevrolet Express", Capacity: 25, OilStatus: "due", TireStatus: "good", MaintenanceNotes: "Oil change scheduled"},
			{BusID: "BUS003", Status: "maintenance", Model: "Toyota Coaster", Capacity: 15, OilStatus: "good", TireStatus: "worn", MaintenanceNotes: "Brake inspection in progress"},
		}
		f, err := os.OpenFile("data/buses.json", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			log.Printf("Warning: failed to create buses.json: %v", err)
			return
		}
		defer f.Close()
		enc := json.NewEncoder(f)
		enc.SetIndent("", "  ")
		if err := enc.Encode(defaultBuses); err != nil {
			log.Printf("Warning: failed to encode buses to json: %v", err)
			return
		}
		log.Println("Created and seeded data/buses.json with ID-based structure")
	}

	// Create vehicle.json if it doesn't exist
	if _, err := os.Stat("data/vehicle.json"); os.IsNotExist(err) {
		defaultVehicles := []Vehicle{
			{VehicleID: "VEH001", Model: "Ford F-150", Year: "2022", License: "ABC123", Status: "active", OilStatus: "good", TireStatus: "good"},
			{VehicleID: "VEH002", Model: "Chevrolet Silverado", Year: "2021", License: "XYZ789", Status: "active", OilStatus: "needs_service", TireStatus: "worn"},
		}
		f, err := os.OpenFile("data/vehicle.json", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			log.Printf("Warning: failed to create vehicle.json: %v", err)
			return
		}
		defer f.Close()
		enc := json.NewEncoder(f)
		enc.SetIndent("", "  ")
		if err := enc.Encode(defaultVehicles); err != nil {
			log.Printf("Warning: failed to encode vehicles to json: %v", err)
			return
		}
		log.Println("Created and seeded data/vehicle.json")
	}

	// Create students.json if it doesn't exist
	if _, err := os.Stat("data/students.json"); os.IsNotExist(err) {
		defaultStudents := []Student{}
		f, err := os.OpenFile("data/students.json", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			log.Printf("Warning: failed to create students.json: %v", err)
			return
		}
		defer f.Close()
		enc := json.NewEncoder(f)
		enc.SetIndent("", "  ")
		if err := enc.Encode(defaultStudents); err != nil {
			log.Printf("Warning: failed to encode students to json: %v", err)
			return
		}
		log.Println("Created data/students.json")
	}

	// Create routes.json if it doesn't exist, and seed with RouteID-based data
	if _, err := os.Stat("data/routes.json"); os.IsNotExist(err) {
		routes := []Route{
			{
				RouteID:   "RT001",
				RouteName: "Victory Square",
				Positions: []struct {
					Position int    `json:"position"`
					Student  string `json:"student"`
				}{{Position: 1, Student: "Alice Johnson"}, {Position: 2, Student: "Bob Smith"}},
			},
			{
				RouteID:   "RT002",
				RouteName: "Airportway",
				Positions: []struct {
					Position int    `json:"position"`
					Student  string `json:"student"`
				}{{Position: 1, Student: "Charlie Brown"}, {Position: 2, Student: "David Wilson"}},
			},
			{
				RouteID:   "RT003",
				RouteName: "NELC",
				Positions: []struct {
					Position int    `json:"position"`
					Student  string `json:"student"`
				}{{Position: 1, Student: "Emma Davis"}, {Position: 2, Student: "Frank Miller"}},
			},
			{
				RouteID:   "RT004",
				RouteName: "Irrigon",
				Positions: []struct {
					Position int    `json:"position"`
					Student  string `json:"student"`
				}{{Position: 1, Student: "Grace Lee"}, {Position: 2, Student: "Henry Clark"}},
			},
			{
				RouteID:   "RT005",
				RouteName: "PELC",
				Positions: []struct {
					Position int    `json:"position"`
					Student  string `json:"student"`
				}{{Position: 1, Student: "Ivy Rodriguez"}, {Position: 2, Student: "Jack Thompson"}},
			},
			{
				RouteID:   "RT006",
				RouteName: "Umatilla",
				Positions: []struct {
					Position int    `json:"position"`
					Student  string `json:"student"`
				}{{Position: 1, Student: "Kate Anderson"}, {Position: 2, Student: "Liam Garcia"}},
			},
		}
		f, err := os.OpenFile("data/routes.json", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			log.Printf("Warning: failed to create routes.json: %v", err)
			return
		}
		defer f.Close()
		enc := json.NewEncoder(f)
		enc.SetIndent("", "  ")
		if err := enc.Encode(routes); err != nil {
			log.Printf("Warning: failed to encode routes to json: %v", err)
			return
		}
		log.Println("Created and seeded data/routes.json with RouteID structure")
	}

	// Create route_assignments.json if it doesn't exist
	if _, err := os.Stat("data/route_assignments.json"); os.IsNotExist(err) {
		defaultAssignments := []RouteAssignment{}
		f, err := os.OpenFile("data/route_assignments.json", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			log.Printf("Warning: failed to create route_assignments.json: %v", err)
			return
		}
		defer f.Close()
		enc := json.NewEncoder(f)
		enc.SetIndent("", "  ")
		if err := enc.Encode(defaultAssignments); err != nil {
			log.Printf("Warning: failed to encode assignments to json: %v", err)
			return
		}
		log.Println("Created data/route_assignments.json")
	}

	// Create maintenance.json if it doesn't exist
	if _, err := os.Stat("data/maintenance.json"); os.IsNotExist(err) {
		defaultMaintenance := []MaintenanceLog{}
		f, err := os.OpenFile("data/maintenance.json", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			log.Printf("Warning: failed to create maintenance.json: %v", err)
			return
		}
		defer f.Close()
		enc := json.NewEncoder(f)
		enc.SetIndent("", "  ")
		if err := enc.Encode(defaultMaintenance); err != nil {
			log.Printf("Warning: failed to encode maintenance to json: %v", err)
			return
		}
		log.Println("Created data/maintenance.json")
	}

	// Create driver_logs.json if it doesn't exist
	if _, err := os.Stat("data/driver_logs.json"); os.IsNotExist(err) {
		defaultLogs := []DriverLog{}
		f, err := os.OpenFile("data/driver_logs.json", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			log.Printf("Warning: failed to create driver_logs.json: %v", err)
			return
		}
		defer f.Close()
		enc := json.NewEncoder(f)
		enc.SetIndent("", "  ")
		if err := enc.Encode(defaultLogs); err != nil {
			log.Printf("Warning: failed to encode driver logs to json: %v", err)
			return
		}
		log.Println("Created data/driver_logs.json")
	}
}

func main() {
	// Add defer to catch any panics
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Server crashed with panic: %v", r)
			os.Exit(1)
		}
	}()

	log.Println("Starting bus transportation app...")

	// 🆕 NEW: Setup database first if DATABASE_URL exists
	if os.Getenv("DATABASE_URL") != "" {
		log.Println("🗄️  Setting up PostgreSQL database...")
		setupDatabase()
		log.Println("Using PostgreSQL database")
	} else {
		log.Println("No DATABASE_URL found, using local file storage")
		// Ensure basic data files exist
		log.Println("Initializing data files...")
		ensureDataFiles()
		// Initialize data files with proper structure
		initDataFiles()
		log.Println("Using local file storage only")
	}

	// Setup HTTP routes with recovery middleware
	log.Println("Setting up HTTP routes...")
	http.HandleFunc("/", withRecovery(rootHealthCheck))
	http.HandleFunc("/new-user", withRecovery(newUserPage))
	http.HandleFunc("/edit-user", withRecovery(editUserPage))
	http.HandleFunc("/dashboard", withRecovery(dashboardRouter))
	http.HandleFunc("/manager-dashboard", withRecovery(managerDashboard))
	http.HandleFunc("/driver-dashboard", withRecovery(driverDashboard))
	http.HandleFunc("/driver/", withRecovery(driverProfileHandler))
	http.HandleFunc("/assign-routes", withRecovery(assignRoutesPage))
	http.HandleFunc("/assign-route", withRecovery(assignRoute))
	http.HandleFunc("/delete-route", withRecovery(deleteRoute))
	http.HandleFunc("/unassign-route", withRecovery(unassignRoute))
	http.HandleFunc("/fleet", withRecovery(fleetPage))
	http.HandleFunc("/company-fleet", withRecovery(companyFleetPage))
	http.HandleFunc("/update-vehicle-status", withRecovery(updateVehicleStatus))
	http.HandleFunc("/update-bus-status", withRecovery(updateBusStatus))
	http.HandleFunc("/add-bus", withRecovery(addBus))
	http.HandleFunc("/edit-bus", withRecovery(editBus))
	http.HandleFunc("/remove-bus", withRecovery(removeBus))
	http.HandleFunc("/webhook", withRecovery(handleWebhook))
	http.HandleFunc("/pull", withRecovery(runPullHandler))
	http.HandleFunc("/save-log", withRecovery(saveDriverLog))
	http.HandleFunc("/students", withRecovery(studentsPage))
	http.HandleFunc("/add-student", withRecovery(addStudent))
	http.HandleFunc("/edit-student", withRecovery(editStudent))
	http.HandleFunc("/remove-student", withRecovery(removeStudent))
	http.HandleFunc("/add-maint", withRecovery(addMaintenanceLog))
	http.HandleFunc("/add-route", withRecovery(addRoute))
	http.HandleFunc("/reassign-driver-bus", withRecovery(reassignDriverBus))
	http.HandleFunc("/remove-user", withRecovery(removeUser))
	http.HandleFunc("/logout", withRecovery(logout))
	http.HandleFunc("/health", withRecovery(healthCheck))

	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}

	// Check if port is available before trying to bind
	log.Printf("Checking if port %s is available...", port)

	server := &http.Server{
		Addr:           "0.0.0.0:" + port,
		Handler:        http.DefaultServeMux,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   60 * time.Second,
		IdleTimeout:    120 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1 MB
	}

	log.Printf("Server starting on port %s with PostgreSQL migration support", port)
	log.Printf("Data structure: BusID, RouteID, StudentID for consistent identification")
	log.Printf("PostgreSQL: Auto-migration from JSON files on first run")
	log.Printf("Server will be accessible at: http://0.0.0.0:%s", port)
	log.Printf("Starting HTTP server...")

	go func() {
		time.Sleep(2 * time.Second)
		log.Printf("Server should be ready now - check the webview")
	}()

	if err := server.ListenAndServe(); err != nil {
		if err == http.ErrServerClosed {
			log.Println("Server was closed")
		} else {
			log.Printf("Server failed to start: %v", err)
			log.Printf("This usually means port %s is already in use", port)
			log.Println("Try running: pkill -f 'go run main.go' or lsof -ti:5000 | xargs kill -9")
			os.Exit(1)
		}
	}
}
