// database.go - Complete PostgreSQL Database Operations
package main

import (
    "database/sql"
    "fmt"
    "log"
    "os"
    
    "github.com/jmoiron/sqlx"
    _ "github.com/lib/pq"
    "golang.org/x/crypto/bcrypt"
)

var db *sqlx.DB

// setupDatabase initializes the database connection
func setupDatabase() {
    if err := initDatabase(); err != nil {
        log.Fatalf("Failed to initialize database: %v", err)
    }
    
    log.Println("Database initialized successfully")
    
    // Run migrations
    if err := runMigrations(); err != nil {
        log.Fatalf("Failed to run migrations: %v", err)
    }
    
    // Check database health
    if err := checkDatabaseHealth(); err != nil {
        log.Fatalf("Database health check failed: %v", err)
    }
}

// Initialize database connection to your Railway PostgreSQL
func initDatabase() error {
    // Use environment variables for Railway connection
    dbURL := os.Getenv("DATABASE_URL")
    if dbURL == "" {
        // Fallback to individual environment variables
        host := os.Getenv("PGHOST")
        port := os.Getenv("PGPORT")
        user := os.Getenv("PGUSER")
        password := os.Getenv("PGPASSWORD")
        dbname := os.Getenv("PGDATABASE")
        
        if host == "" || port == "" || user == "" || password == "" || dbname == "" {
            return fmt.Errorf("database connection parameters not set")
        }
        
        dbURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=require",
            user, password, host, port, dbname)
    }
    
    var err error
    db, err = sqlx.Connect("postgres", dbURL)
    if err != nil {
        return fmt.Errorf("failed to connect to database: %v", err)
    }
    
    // Configure connection pool
    db.SetMaxOpenConns(25)
    db.SetMaxIdleConns(5)
    
    // Test the connection
    if err = db.Ping(); err != nil {
        return fmt.Errorf("failed to ping database: %v", err)
    }
    
    log.Println("Successfully connected to Railway PostgreSQL database")
    return nil
}

// runMigrations creates all necessary tables if they don't exist
func runMigrations() error {
    // Read and execute the schema
    schema := `
    -- Users table
    CREATE TABLE IF NOT EXISTS users (
        id SERIAL PRIMARY KEY,
        username VARCHAR(50) UNIQUE NOT NULL,
        password VARCHAR(255) NOT NULL,
        role VARCHAR(20) NOT NULL CHECK (role IN ('driver', 'manager')),
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    );

    -- Buses table
    CREATE TABLE IF NOT EXISTS buses (
        id SERIAL PRIMARY KEY,
        bus_id VARCHAR(20) UNIQUE NOT NULL,
        status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'maintenance', 'out_of_service')),
        model VARCHAR(100),
        capacity INTEGER DEFAULT 0,
        oil_status VARCHAR(20) DEFAULT 'good' CHECK (oil_status IN ('good', 'due', 'overdue')),
        tire_status VARCHAR(20) DEFAULT 'good' CHECK (tire_status IN ('good', 'worn', 'replace')),
        maintenance_notes TEXT,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    );

    -- Routes table
    CREATE TABLE IF NOT EXISTS routes (
        id SERIAL PRIMARY KEY,
        route_id VARCHAR(20) UNIQUE NOT NULL,
        route_name VARCHAR(100) NOT NULL,
        description TEXT,
        positions JSONB DEFAULT '[]'::jsonb,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    );

    -- Students table
    CREATE TABLE IF NOT EXISTS students (
        id SERIAL PRIMARY KEY,
        student_id VARCHAR(20) UNIQUE NOT NULL,
        name VARCHAR(100) NOT NULL,
        locations JSONB DEFAULT '[]'::jsonb,
        phone_number VARCHAR(20),
        alt_phone_number VARCHAR(20),
        guardian VARCHAR(100),
        pickup_time TIME,
        dropoff_time TIME,
        position_number INTEGER,
        route_id VARCHAR(20),
        driver VARCHAR(50),
        active BOOLEAN DEFAULT true,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    );

    -- Route assignments table
    CREATE TABLE IF NOT EXISTS route_assignments (
        id SERIAL PRIMARY KEY,
        driver VARCHAR(50) NOT NULL,
        bus_id VARCHAR(20) NOT NULL,
        route_id VARCHAR(20) NOT NULL,
        route_name VARCHAR(100),
        assigned_date DATE DEFAULT CURRENT_DATE,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        UNIQUE(driver)
    );

    -- Driver logs table
    CREATE TABLE IF NOT EXISTS driver_logs (
        id SERIAL PRIMARY KEY,
        driver VARCHAR(50) NOT NULL,
        bus_id VARCHAR(20) NOT NULL,
        route_id VARCHAR(20) NOT NULL,
        date DATE NOT NULL,
        period VARCHAR(20) NOT NULL CHECK (period IN ('morning', 'afternoon', 'evening')),
        departure_time TIME,
        arrival_time TIME,
        mileage DECIMAL(10,2),
        attendance JSONB DEFAULT '[]'::jsonb,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        UNIQUE(driver, date, period)
    );

    -- Bus maintenance logs table
    CREATE TABLE IF NOT EXISTS bus_maintenance_logs (
        id SERIAL PRIMARY KEY,
        bus_id VARCHAR(20) NOT NULL,
        date DATE NOT NULL,
        category VARCHAR(50) NOT NULL,
        notes TEXT,
        mileage INTEGER,
        cost DECIMAL(10,2),
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    );

    -- Company vehicles table
    CREATE TABLE IF NOT EXISTS vehicles (
        id SERIAL PRIMARY KEY,
        vehicle_id VARCHAR(20) UNIQUE NOT NULL,
        model VARCHAR(100),
        description TEXT,
        year VARCHAR(4),
        tire_size VARCHAR(50),
        license VARCHAR(20),
        oil_status VARCHAR(20) DEFAULT 'good' CHECK (oil_status IN ('good', 'needs_service', 'overdue')),
        tire_status VARCHAR(20) DEFAULT 'good' CHECK (tire_status IN ('good', 'worn', 'replace')),
        status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'maintenance', 'out_of_service')),
        maintenance_notes TEXT,
        serial_number VARCHAR(100),
        base VARCHAR(100),
        service_interval INTEGER,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    );

    -- Activities table
    CREATE TABLE IF NOT EXISTS activities (
        id SERIAL PRIMARY KEY,
        date DATE NOT NULL,
        driver VARCHAR(50) NOT NULL,
        trip_name VARCHAR(100) NOT NULL,
        attendance INTEGER DEFAULT 0,
        miles DECIMAL(10,2) DEFAULT 0,
        notes TEXT,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    );

    -- Create update triggers
    CREATE OR REPLACE FUNCTION update_updated_at_column()
    RETURNS TRIGGER AS $$
    BEGIN
        NEW.updated_at = CURRENT_TIMESTAMP;
        RETURN NEW;
    END;
    $$ language 'plpgsql';
    `
    
    // Execute the schema
    if _, err := db.Exec(schema); err != nil {
        return fmt.Errorf("failed to create schema: %v", err)
    }
    
    // Create triggers for each table that has updated_at
    tables := []string{"users", "buses", "routes", "students", "route_assignments", "vehicles"}
    for _, table := range tables {
        triggerSQL := fmt.Sprintf(`
            DROP TRIGGER IF EXISTS update_%s_updated_at ON %s;
            CREATE TRIGGER update_%s_updated_at BEFORE UPDATE ON %s
                FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
        `, table, table, table, table)
        
        if _, err := db.Exec(triggerSQL); err != nil {
            log.Printf("Warning: Failed to create trigger for %s: %v", table, err)
        }
    }
    
    // Create indexes
    indexes := []string{
        "CREATE INDEX IF NOT EXISTS idx_buses_status ON buses(status)",
        "CREATE INDEX IF NOT EXISTS idx_students_driver ON students(driver)",
        "CREATE INDEX IF NOT EXISTS idx_students_route ON students(route_id)",
        "CREATE INDEX IF NOT EXISTS idx_driver_logs_driver_date ON driver_logs(driver, date)",
        "CREATE INDEX IF NOT EXISTS idx_maintenance_bus_date ON bus_maintenance_logs(bus_id, date)",
        "CREATE INDEX IF NOT EXISTS idx_route_assignments_driver ON route_assignments(driver)",
    }
    
    for _, idx := range indexes {
        if _, err := db.Exec(idx); err != nil {
            log.Printf("Warning: Failed to create index: %v", err)
        }
    }
    
    // Create default admin user with HASHED password
    if err := createDefaultAdminUser(); err != nil {
        log.Printf("Warning: Failed to create default admin user: %v", err)
    }
    
    // Auto-migrate any plain text passwords
    if err := autoMigratePasswords(); err != nil {
        log.Printf("Warning: Failed to auto-migrate passwords: %v", err)
    }
    
    log.Println("Database migrations completed successfully")
    return nil
}

// createDefaultAdminUser creates a default admin with hashed password
func createDefaultAdminUser() error {
    // Check if admin already exists
    var count int
    err := db.Get(&count, "SELECT COUNT(*) FROM users WHERE username = 'admin'")
    if err != nil {
        return fmt.Errorf("failed to check for admin user: %v", err)
    }
    
    if count > 0 {
        log.Println("Admin user already exists")
        return nil
    }
    
    // Hash the default password
    hashedPassword, err := bcrypt.GenerateFromPassword([]byte("adminpass"), 12)
    if err != nil {
        return fmt.Errorf("failed to hash default password: %v", err)
    }
    
    // Insert the admin user
    _, err = db.Exec(`
        INSERT INTO users (username, password, role) 
        VALUES ('admin', $1, 'manager')
    `, string(hashedPassword))
    
    if err != nil {
        return fmt.Errorf("failed to insert admin user: %v", err)
    }
    
    log.Println("Created default admin user with username: admin, password: adminpass")
    return nil
}

// autoMigratePasswords automatically migrates any plain text passwords to bcrypt
func autoMigratePasswords() error {
    log.Println("Checking for plain text passwords to migrate...")
    
    // Get all users
    rows, err := db.Query("SELECT username, password FROM users")
    if err != nil {
        return fmt.Errorf("failed to query users: %v", err)
    }
    defer rows.Close()
    
    type UserToCheck struct {
        Username string
        Password string
    }
    
    var users []UserToCheck
    for rows.Next() {
        var user UserToCheck
        if err := rows.Scan(&user.Username, &user.Password); err != nil {
            log.Printf("Error scanning user: %v", err)
            continue
        }
        users = append(users, user)
    }
    
    migratedCount := 0
    
    // Begin transaction
    tx, err := db.Begin()
    if err != nil {
        return fmt.Errorf("failed to begin transaction: %v", err)
    }
    defer tx.Rollback()
    
    for _, user := range users {
        // Check if password is already hashed (bcrypt hashes start with $2a$, $2b$, or $2y$)
        if len(user.Password) > 4 && user.Password[0] == '$' && user.Password[1] == '2' {
            continue // Already hashed
        }
        
        // Hash the plain text password
        hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), 12)
        if err != nil {
            log.Printf("Failed to hash password for user %s: %v", user.Username, err)
            continue
        }
        
        // Update the user's password
        _, err = tx.Exec("UPDATE users SET password = $1, updated_at = CURRENT_TIMESTAMP WHERE username = $2",
            string(hashedPassword), user.Username)
        if err != nil {
            log.Printf("Failed to update password for user %s: %v", user.Username, err)
            continue
        }
        
        log.Printf("Migrated password for user: %s", user.Username)
        migratedCount++
    }
    
    // Commit transaction
    if err := tx.Commit(); err != nil {
        return fmt.Errorf("failed to commit transaction: %v", err)
    }
    
    if migratedCount > 0 {
        log.Printf("Successfully migrated %d passwords to bcrypt", migratedCount)
    } else {
        log.Println("No plain text passwords found to migrate")
    }
    
    return nil
}

// Get vehicle list for the fleet overview page (legacy function for compatibility)
func getVehicleList() ([]VehicleWithStats, error) {
    // This function is not used in the new system
    return []VehicleWithStats{}, nil
}

// Get a specific vehicle by vehicle number (legacy function)
func getVehicle(vehicleNumber int) (*Vehicle, error) {
    // This function is not used in the new system
    return nil, fmt.Errorf("legacy function not implemented")
}

// Get maintenance records for a specific bus
func getBusMaintenanceRecords(busID string) ([]BusMaintenanceLog, error) {
    query := `
        SELECT bus_id, date, category, notes, mileage, cost
        FROM bus_maintenance_logs 
        WHERE bus_id = $1 
        ORDER BY date DESC`
    
    var records []BusMaintenanceLog
    rows, err := db.Query(query, busID)
    if err != nil {
        return nil, fmt.Errorf("failed to fetch maintenance records: %v", err)
    }
    defer rows.Close()
    
    for rows.Next() {
        var record BusMaintenanceLog
        var date sql.NullTime
        var cost sql.NullFloat64
        
        if err := rows.Scan(&record.BusID, &date, &record.Category, 
            &record.Notes, &record.Mileage, &cost); err != nil {
            log.Printf("Error scanning maintenance record: %v", err)
            continue
        }
        
        if date.Valid {
            record.Date = date.Time.Format("2006-01-02")
        }
        
        records = append(records, record)
    }
    
    return records, nil
}

// Get maintenance records for a specific vehicle (legacy compatibility)
func getVehicleMaintenanceRecords(vehicleNumber int) ([]MaintenanceLog, error) {
    // Convert to new system - this is for compatibility
    return []MaintenanceLog{}, nil
}

// Add a new maintenance record (legacy compatibility)
func addMaintenanceRecord(record MaintenanceLog) error {
    // Convert to new system - this is for compatibility
    return fmt.Errorf("use saveMaintenanceLog instead")
}

// Get fleet statistics
func getFleetStats() (*FleetStats, error) {
    var stats FleetStats
    
    // Get total buses
    err := db.Get(&stats.TotalVehicles, "SELECT COUNT(*) FROM buses")
    if err != nil {
        return nil, fmt.Errorf("failed to get bus count: %v", err)
    }
    
    // Get total maintenance records
    err = db.Get(&stats.TotalMaintenanceRecords, "SELECT COUNT(*) FROM bus_maintenance_logs")
    if err != nil {
        return nil, fmt.Errorf("failed to get maintenance count: %v", err)
    }
    
    // Get total maintenance cost
    err = db.Get(&stats.TotalMaintenanceCost, 
        "SELECT COALESCE(SUM(cost), 0) FROM bus_maintenance_logs WHERE cost IS NOT NULL")
    if err != nil {
        return nil, fmt.Errorf("failed to get total cost: %v", err)
    }
    
    // Calculate average cost
    if stats.TotalMaintenanceRecords > 0 {
        stats.AverageMaintenanceCost = stats.TotalMaintenanceCost / float64(stats.TotalMaintenanceRecords)
    }
    
    return &stats, nil
}

// Get recent maintenance activity
func getRecentMaintenanceActivity(limit int) ([]BusMaintenanceLog, error) {
    query := `
        SELECT bus_id, date, category, notes, mileage, cost
        FROM bus_maintenance_logs
        ORDER BY date DESC, created_at DESC
        LIMIT $1`
    
    var records []BusMaintenanceLog
    rows, err := db.Query(query, limit)
    if err != nil {
        return nil, fmt.Errorf("failed to fetch recent maintenance: %v", err)
    }
    defer rows.Close()
    
    for rows.Next() {
        var record BusMaintenanceLog
        var date sql.NullTime
        var cost sql.NullFloat64
        
        if err := rows.Scan(&record.BusID, &date, &record.Category, 
            &record.Notes, &record.Mileage, &cost); err != nil {
            log.Printf("Error scanning maintenance record: %v", err)
            continue
        }
        
        if date.Valid {
            record.Date = date.Time.Format("2006-01-02")
        }
        
        records = append(records, record)
    }
    
    return records, nil
}

// Search vehicles by ID, model, or description
func searchVehicles(searchTerm string) ([]Vehicle, error) {
    searchPattern := "%" + searchTerm + "%"
    query := `
        SELECT vehicle_id, model, description, year, tire_size, license,
            oil_status, tire_status, status, maintenance_notes, serial_number, base, service_interval
        FROM vehicles
        WHERE 
            vehicle_id ILIKE $1 OR
            model ILIKE $1 OR 
            description ILIKE $1
        ORDER BY vehicle_id ASC`
    
    var vehicles []Vehicle
    err := db.Select(&vehicles, query, searchPattern)
    if err != nil {
        return nil, fmt.Errorf("failed to search vehicles: %v", err)
    }
    
    return vehicles, nil
}

// Check database health
func checkDatabaseHealth() error {
    if db == nil {
        return fmt.Errorf("database connection is nil")
    }
    
    if err := db.Ping(); err != nil {
        return fmt.Errorf("database ping failed: %v", err)
    }
    
    // Verify tables exist
    tables := []string{"users", "buses", "routes", "students", "route_assignments", 
                      "driver_logs", "bus_maintenance_logs", "vehicles", "activities"}
    
    for _, table := range tables {
        var exists bool
        err := db.Get(&exists, `
            SELECT EXISTS (
                SELECT FROM information_schema.tables 
                WHERE table_schema = 'public' 
                AND table_name = $1
            )`, table)
        
        if err != nil || !exists {
            return fmt.Errorf("table %s does not exist", table)
        }
    }
    
    log.Println("Database health check passed")
    return nil
}

// Cleanup function to close database connection
func closeDatabase() {
    if db != nil {
        db.Close()
        log.Println("Database connection closed")
    }
}
