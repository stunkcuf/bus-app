// database.go - Complete PostgreSQL Database Operations with Registration System
package main

import (
    "database/sql"
    "fmt"
    "log"
    "os"
    "sort"
    "strconv"
    
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
    -- Users table with registration support
    CREATE TABLE IF NOT EXISTS users (
        id SERIAL PRIMARY KEY,
        username VARCHAR(50) UNIQUE NOT NULL,
        password VARCHAR(255) NOT NULL,
        role VARCHAR(20) NOT NULL CHECK (role IN ('driver', 'manager', 'driver_pending')),
        status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'pending', 'suspended')),
        registration_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
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
    
    -- Service records table (legacy support)
    CREATE TABLE IF NOT EXISTS service_records (
        id SERIAL PRIMARY KEY,
        vehicle_number INTEGER,
        maintenance_date DATE,
        category VARCHAR(50),
        notes TEXT,
        work_done TEXT,
        mileage INTEGER,
        cost DECIMAL(10,2),
        po_number VARCHAR(50),
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    );

    -- Maintenance records table (unified maintenance tracking)
    CREATE TABLE IF NOT EXISTS maintenance_records (
        id SERIAL PRIMARY KEY,
        vehicle_id VARCHAR(20) NOT NULL,
        vehicle_type VARCHAR(20) DEFAULT 'vehicle' CHECK (vehicle_type IN ('bus', 'vehicle')),
        date DATE NOT NULL,
        category VARCHAR(50) NOT NULL,
        notes TEXT,
        mileage INTEGER,
        cost DECIMAL(10,2),
        po_number VARCHAR(50),
        work_done TEXT,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
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
    tables := []string{"users", "buses", "routes", "students", "route_assignments", "vehicles", "maintenance_records"}
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
        "CREATE INDEX IF NOT EXISTS idx_users_role ON users(role)",
        "CREATE INDEX IF NOT EXISTS idx_users_status ON users(status)",
        "CREATE INDEX IF NOT EXISTS idx_maintenance_records_vehicle ON maintenance_records(vehicle_id)",
        "CREATE INDEX IF NOT EXISTS idx_service_records_vehicle ON service_records(vehicle_number)",
    }
    
    for _, idx := range indexes {
        if _, err := db.Exec(idx); err != nil {
            log.Printf("Warning: Failed to create index: %v", err)
        }
    }
    
    // Update existing tables for registration system
    if err := updateUsersTableForRegistration(); err != nil {
        log.Printf("Warning: Failed to update users table for registration: %v", err)
    }
    
    // Ensure updated_at columns exist
    if err := ensureUpdatedAtColumn(); err != nil {
        log.Printf("Warning: Failed to ensure updated_at columns: %v", err)
    }
    
    // Ensure route names are populated in route assignments
    if err := fixRouteAssignmentNames(); err != nil {
        log.Printf("Warning: Failed to fix route assignment names: %v", err)
    }
    
    // Ensure routes table has proper structure
    if err := ensureRoutesTableStructure(); err != nil {
        log.Printf("Warning: Failed to ensure routes table structure: %v", err)
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

// updateUsersTableForRegistration adds registration support to existing users table
func updateUsersTableForRegistration() error {
    // First, check if we need to add the driver_pending role
    var constraintExists bool
    err := db.Get(&constraintExists, `
        SELECT EXISTS (
            SELECT 1 FROM information_schema.check_constraints 
            WHERE constraint_name LIKE 'users_role_check%'
            AND check_clause LIKE '%driver_pending%'
        )
    `)
    
    if err != nil {
        log.Printf("Warning: Could not check role constraint: %v", err)
    }
    
    if !constraintExists {
        // We need to drop and recreate the constraint to include driver_pending
        _, err = db.Exec(`
            ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_check;
            ALTER TABLE users ADD CONSTRAINT users_role_check 
            CHECK (role IN ('driver', 'manager', 'driver_pending'));
        `)
        
        if err != nil {
            log.Printf("Warning: Failed to update role constraint: %v", err)
        } else {
            log.Println("Updated users role constraint to include driver_pending")
        }
    }
    
    // Check if status column exists
    var statusExists bool
    err = db.Get(&statusExists, `
        SELECT EXISTS (
            SELECT 1 FROM information_schema.columns 
            WHERE table_name = 'users' 
            AND column_name = 'status'
        )
    `)
    
    if err != nil {
        return fmt.Errorf("failed to check status column existence: %v", err)
    }
    
    if !statusExists {
        // Add status column to existing users table
        _, err = db.Exec(`
            ALTER TABLE users 
            ADD COLUMN status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'pending', 'suspended'))
        `)
        
        if err != nil {
            return fmt.Errorf("failed to add status column: %v", err)
        }
        
        log.Println("Added status column to users table")
        
        // Update existing users to be active
        _, err = db.Exec(`UPDATE users SET status = 'active' WHERE status IS NULL`)
        if err != nil {
            return fmt.Errorf("failed to update existing users status: %v", err)
        }
    }
    
    // Check if registration_date column exists
    var regDateExists bool
    err = db.Get(&regDateExists, `
        SELECT EXISTS (
            SELECT 1 FROM information_schema.columns 
            WHERE table_name = 'users' 
            AND column_name = 'registration_date'
        )
    `)
    
    if err != nil {
        return fmt.Errorf("failed to check registration_date existence: %v", err)
    }
    
    if !regDateExists {
        _, err = db.Exec(`
            ALTER TABLE users 
            ADD COLUMN registration_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        `)
        
        if err != nil {
            return fmt.Errorf("failed to add registration_date column: %v", err)
        }
        
        log.Println("Added registration_date column to users table")
        
        // Update existing users with current timestamp
        _, err = db.Exec(`UPDATE users SET registration_date = CURRENT_TIMESTAMP WHERE registration_date IS NULL`)
        if err != nil {
            log.Printf("Warning: Failed to update registration_date for existing users: %v", err)
        }
    }
    
    return nil
}

// ensureUpdatedAtColumn adds updated_at column to tables that need it
func ensureUpdatedAtColumn() error {
	tables := []string{"users", "buses", "routes", "students", "route_assignments", "vehicles"}
	
	for _, table := range tables {
		// Check if updated_at column exists
		var columnExists bool
		err := db.QueryRow(`
			SELECT EXISTS (
				SELECT 1 FROM information_schema.columns 
				WHERE table_name = $1 
				AND column_name = 'updated_at'
			)
		`, table).Scan(&columnExists)
		
		if err != nil {
			log.Printf("Warning: Failed to check updated_at column for %s: %v", table, err)
			continue
		}
		
		if !columnExists {
			// Add updated_at column
			_, err = db.Exec(fmt.Sprintf(`
				ALTER TABLE %s 
				ADD COLUMN updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			`, table))
			
			if err != nil {
				log.Printf("Warning: Failed to add updated_at column to %s: %v", table, err)
				continue
			}
			
			log.Printf("Added updated_at column to %s table", table)
			
			// Update existing rows
			_, err = db.Exec(fmt.Sprintf(`
				UPDATE %s SET updated_at = CURRENT_TIMESTAMP WHERE updated_at IS NULL
			`, table))
			
			if err != nil {
				log.Printf("Warning: Failed to update updated_at for existing rows in %s: %v", table, err)
			}
		}
	}
	
	return nil
}

// createDefaultAdminUser creates a default admin with hashed password
// NOTE: The admin account is a system account and is hidden from the UI
// It should only be used for initial system setup and emergency access
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
        INSERT INTO users (username, password, role, status) 
        VALUES ('admin', $1, 'manager', 'active')
    `, string(hashedPassword))
    
    if err != nil {
        return fmt.Errorf("failed to insert admin user: %v", err)
    }
    
    log.Println("Created default admin user with username: admin, password: adminpass")
    log.Println("NOTE: This account is hidden from the UI and should only be used for system administration")
    return nil
}

// autoMigratePasswords automatically migrates any plain text passwords to bcrypt
func autoMigratePasswords() error {
    log.Println("Checking for plain text passwords to migrate...")
    
    // Get all users
    rows, err := db.Query("SELECT username, password FROM users WHERE role != 'driver_pending'")
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

// fixRouteAssignmentNames ensures all route assignments have proper route names
func fixRouteAssignmentNames() error {
	log.Println("Checking and fixing route assignment names...")
	
	// First check if there are any assignments with NULL or empty route names
	var count int
	err := db.Get(&count, `
		SELECT COUNT(*) FROM route_assignments 
		WHERE route_name IS NULL OR route_name = ''
	`)
	
	if err != nil {
		return fmt.Errorf("failed to check route assignments: %v", err)
	}
	
	if count == 0 {
		log.Println("All route assignments have proper route names")
		return nil
	}
	
	log.Printf("Found %d route assignments with missing route names, fixing...", count)
	
	// Update route names based on route_id
	_, err = db.Exec(`
		UPDATE route_assignments ra
		SET route_name = r.route_name
		FROM routes r
		WHERE ra.route_id = r.route_id
		AND (ra.route_name IS NULL OR ra.route_name = '')
	`)
	
	if err != nil {
		return fmt.Errorf("failed to update route names: %v", err)
	}
	
	log.Println("Successfully fixed route assignment names")
	return nil
}

// ensureRoutesTableStructure ensures the routes table has all necessary columns
func ensureRoutesTableStructure() error {
	// Check if positions column exists
	var columnExists bool
	err := db.Get(&columnExists, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.columns 
			WHERE table_name = 'routes' 
			AND column_name = 'positions'
		)
	`)
	
	if err != nil {
		log.Printf("Warning: Could not check routes table structure: %v", err)
		return nil // Don't fail the whole migration
	}
	
	if !columnExists {
		log.Println("Adding positions column to routes table...")
		_, err = db.Exec(`
			ALTER TABLE routes 
			ADD COLUMN positions JSONB DEFAULT '[]'::jsonb
		`)
		
		if err != nil {
			return fmt.Errorf("failed to add positions column: %v", err)
		}
		
		log.Println("Successfully added positions column to routes table")
	}
	
	return nil
}

// getAllVehicleMaintenanceRecords gets maintenance records from ALL tables
func getAllVehicleMaintenanceRecords(vehicleID string) ([]BusMaintenanceLog, error) {
    if db == nil {
        return nil, fmt.Errorf("database connection not available")
    }
    
    var records []BusMaintenanceLog
    
    // 1. Get records from bus_maintenance_logs table
    query1 := `
        SELECT bus_id, date, category, notes, mileage, cost
        FROM bus_maintenance_logs 
        WHERE bus_id = $1 
        ORDER BY date DESC`
    
    rows1, err := db.Query(query1, vehicleID)
    if err != nil {
        log.Printf("Error querying bus_maintenance_logs: %v", err)
    } else {
        defer rows1.Close()
        for rows1.Next() {
            var record BusMaintenanceLog
            var date sql.NullTime
            var cost sql.NullFloat64
            
            if err := rows1.Scan(&record.BusID, &date, &record.Category, 
                &record.Notes, &record.Mileage, &cost); err != nil {
                log.Printf("Error scanning bus maintenance record: %v", err)
                continue
            }
            
            if date.Valid {
                record.Date = date.Time.Format("2006-01-02")
            }
            
            records = append(records, record)
        }
    }
    
    // 2. Get records from maintenance_records table
    query2 := `
        SELECT vehicle_id, date, category, notes, mileage, cost
        FROM maintenance_records 
        WHERE vehicle_id = $1 
        ORDER BY date DESC`
    
    rows2, err := db.Query(query2, vehicleID)
    if err != nil {
        log.Printf("Error querying maintenance_records: %v", err)
    } else {
        defer rows2.Close()
        for rows2.Next() {
            var record BusMaintenanceLog
            var date sql.NullTime
            var cost sql.NullFloat64
            
            if err := rows2.Scan(&record.BusID, &date, &record.Category, 
                &record.Notes, &record.Mileage, &cost); err != nil {
                log.Printf("Error scanning maintenance record: %v", err)
                continue
            }
            
            if date.Valid {
                record.Date = date.Time.Format("2006-01-02")
            }
            
            records = append(records, record)
        }
    }
    
    // 3. Get records from service_records table
    // Try multiple approaches to handle different ID formats and column names
    
    // First, check which column exists (vehicle_number, vehicle_id, or unnamed_1)
    var columnName string
    err = db.QueryRow(`
        SELECT column_name 
        FROM information_schema.columns 
        WHERE table_name = 'service_records' 
        AND column_name IN ('vehicle_number', 'vehicle_id', 'unnamed_1')
        LIMIT 1
    `).Scan(&columnName)
    
    if err != nil {
        log.Printf("Warning: Could not determine vehicle ID column name in service_records: %v", err)
        columnName = "vehicle_number" // Default fallback
    }
    
    log.Printf("Using column '%s' for vehicle ID in service_records table", columnName)
    
    // Build dynamic query based on column name
    query3 := fmt.Sprintf(`
        SELECT 
            $1 as vehicle_id,
            maintenance_date,
            COALESCE(category, 'service') as category,
            COALESCE(work_done, notes, '') as notes,
            mileage,
            cost
        FROM service_records 
        WHERE %s::TEXT = $1
        ORDER BY maintenance_date DESC`, columnName)
    
    rows3, err := db.Query(query3, vehicleID)
    if err != nil {
        log.Printf("Error querying service_records as string with column %s: %v", columnName, err)
        
        // If that fails and the ID looks numeric, try as integer
        if vehicleNum, err2 := strconv.Atoi(vehicleID); err2 == nil {
            query4 := fmt.Sprintf(`
                SELECT 
                    $1 as vehicle_id,
                    maintenance_date,
                    COALESCE(category, 'service') as category,
                    COALESCE(work_done, notes, '') as notes,
                    mileage,
                    cost
                FROM service_records 
                WHERE %s = $2
                ORDER BY maintenance_date DESC`, columnName)
            
            rows3, err = db.Query(query4, vehicleID, vehicleNum)
            if err != nil {
                log.Printf("Error querying service_records as int with column %s: %v", columnName, err)
                
                // Last attempt: try with COALESCE for all possible column names
                query5 := `
                    SELECT 
                        $1 as vehicle_id,
                        maintenance_date,
                        COALESCE(category, 'service') as category,
                        COALESCE(work_done, notes, '') as notes,
                        mileage,
                        cost
                    FROM service_records 
                    WHERE COALESCE(vehicle_number::TEXT, vehicle_id::TEXT, unnamed_1::TEXT) = $1
                    ORDER BY maintenance_date DESC`
                
                rows3, err = db.Query(query5, vehicleID)
                if err != nil {
                    log.Printf("Error querying service_records with COALESCE: %v", err)
                }
            }
        }
    }
    
    if rows3 != nil {
        defer rows3.Close()
        for rows3.Next() {
            var record BusMaintenanceLog
            var date sql.NullTime
            var cost sql.NullFloat64
            var mileage sql.NullInt64
            
            if err := rows3.Scan(&record.BusID, &date, &record.Category, 
                &record.Notes, &mileage, &cost); err != nil {
                log.Printf("Error scanning service record: %v", err)
                continue
            }
            
            if date.Valid {
                record.Date = date.Time.Format("2006-01-02")
            }
            
            if mileage.Valid {
                record.Mileage = int(mileage.Int64)
            }
            
            // Mark that this came from service_records
            record.Category = "service-" + record.Category
            
            records = append(records, record)
        }
    }
    
    // Sort all records by date (newest first)
    sort.Slice(records, func(i, j int) bool {
        return records[i].Date > records[j].Date
    })
    
    log.Printf("Found %d total maintenance records for vehicle %s", len(records), vehicleID)
    return records, nil
}

// debugMaintenanceTables helps debug what's in the maintenance tables
func debugMaintenanceTables(vehicleID string) {
    log.Printf("\n=== DEBUGGING MAINTENANCE DATA FOR VEHICLE %s ===", vehicleID)
    
    // Check vehicles table
    var exists bool
    err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM vehicles WHERE vehicle_id = $1)", vehicleID).Scan(&exists)
    if err != nil {
        log.Printf("Error checking vehicles table: %v", err)
    } else {
        log.Printf("Vehicle %s exists in vehicles table: %v", vehicleID, exists)
    }
    
    // Check bus_maintenance_logs
    var count1 int
    err = db.QueryRow("SELECT COUNT(*) FROM bus_maintenance_logs WHERE bus_id = $1", vehicleID).Scan(&count1)
    if err != nil {
        log.Printf("Error counting bus_maintenance_logs: %v", err)
    } else {
        log.Printf("Found %d records in bus_maintenance_logs", count1)
    }
    
    // Check maintenance_records
    var count2 int
    err = db.QueryRow("SELECT COUNT(*) FROM maintenance_records WHERE vehicle_id = $1", vehicleID).Scan(&count2)
    if err != nil {
        log.Printf("Error counting maintenance_records: %v", err)
    } else {
        log.Printf("Found %d records in maintenance_records", count2)
    }
    
    // Check service_records with all possible column names
    log.Println("\nChecking service_records with different column names:")
    
    // Check vehicle_number
    var count3a int
    err = db.QueryRow("SELECT COUNT(*) FROM service_records WHERE vehicle_number::VARCHAR = $1", vehicleID).Scan(&count3a)
    if err == nil {
        log.Printf("Found %d records using vehicle_number column (as string)", count3a)
    }
    
    // Check vehicle_id
    var count3b int
    err = db.QueryRow("SELECT COUNT(*) FROM service_records WHERE vehicle_id::VARCHAR = $1", vehicleID).Scan(&count3b)
    if err == nil {
        log.Printf("Found %d records using vehicle_id column", count3b)
    }
    
    // Check unnamed_1
    var count3c int
    err = db.QueryRow("SELECT COUNT(*) FROM service_records WHERE unnamed_1::VARCHAR = $1", vehicleID).Scan(&count3c)
    if err == nil {
        log.Printf("Found %d records using unnamed_1 column", count3c)
    }
    
    // Try COALESCE for all columns
    var count3d int
    err = db.QueryRow(`
        SELECT COUNT(*) FROM service_records 
        WHERE COALESCE(vehicle_number::VARCHAR, vehicle_id::VARCHAR, unnamed_1::VARCHAR) = $1
    `, vehicleID).Scan(&count3d)
    if err != nil {
        log.Printf("Error with COALESCE query: %v", err)
    } else {
        log.Printf("Found %d records using COALESCE of all columns", count3d)
    }
    
    // Try as integer if applicable
    if vehicleNum, err2 := strconv.Atoi(vehicleID); err2 == nil {
        var count3e int
        err = db.QueryRow("SELECT COUNT(*) FROM service_records WHERE vehicle_number = $1", vehicleNum).Scan(&count3e)
        if err == nil {
            log.Printf("Found %d records using vehicle_number as integer", count3e)
        }
    }
    
    // Show sample data from each table
    log.Println("\nSample maintenance_records:")
    rows, _ := db.Query("SELECT vehicle_id, date, category FROM maintenance_records WHERE vehicle_id = $1 LIMIT 3", vehicleID)
    if rows != nil {
        defer rows.Close()
        for rows.Next() {
            var vid, date, cat string
            rows.Scan(&vid, &date, &cat)
            log.Printf("  - %s | %s | %s", vid, date, cat)
        }
    }
    
    log.Println("\nSample service_records (trying different column names):")
    
    // First, find which column exists
    var columnName string
    err = db.QueryRow(`
        SELECT column_name 
        FROM information_schema.columns 
        WHERE table_name = 'service_records' 
        AND column_name IN ('vehicle_number', 'vehicle_id', 'unnamed_1')
        LIMIT 1
    `).Scan(&columnName)
    
    if err == nil && columnName != "" {
        log.Printf("Using column: %s", columnName)
        query := fmt.Sprintf(`
            SELECT %s, maintenance_date, COALESCE(work_done, notes, category, '') 
            FROM service_records 
            WHERE %s::VARCHAR = $1 OR (%s IS NOT NULL AND %s::INTEGER = $2)
            LIMIT 3
        `, columnName, columnName, columnName, columnName)
        
        vehicleNum, _ := strconv.Atoi(vehicleID)
        rows2, err := db.Query(query, vehicleID, vehicleNum)
        if err != nil {
            log.Printf("Error querying service_records: %v", err)
        } else if rows2 != nil {
            defer rows2.Close()
            for rows2.Next() {
                var vnum sql.NullString
                var date sql.NullTime
                var work string
                rows2.Scan(&vnum, &date, &work)
                log.Printf("  - %v | %v | %s", vnum.String, date.Time, work)
            }
        }
    } else {
        log.Printf("Could not determine vehicle ID column in service_records")
    }
    
    log.Println("=== END DEBUG ===\n")
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

// saveMaintenanceRecord saves a maintenance record
func saveMaintenanceRecord(record BusMaintenanceLog, vehicleType string) error {
    if db == nil {
        return fmt.Errorf("database connection not available")
    }
    
    if vehicleType == "" {
        vehicleType = "vehicle"
    }
    
    _, err := db.Exec(`
        INSERT INTO maintenance_records (vehicle_id, vehicle_type, date, category, notes, mileage, cost) 
        VALUES ($1, $2, $3, $4, $5, $6, $7)
    `, record.BusID, vehicleType, record.Date, record.Category, record.Notes, record.Mileage, 0.0)
    
    return err
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
                      "driver_logs", "bus_maintenance_logs", "vehicles", "activities",
                      "service_records", "maintenance_records"}
    
    for _, table := range tables {
        var exists bool
        err := db.Get(&exists, `
            SELECT EXISTS (
                SELECT FROM information_schema.tables 
                WHERE table_schema = 'public' 
                AND table_name = $1
            )`, table)
        
        if err != nil || !exists {
            log.Printf("Warning: table %s does not exist or error checking: %v", table, err)
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
