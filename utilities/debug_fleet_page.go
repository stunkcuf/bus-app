package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found: %v", err)
	}

	// Get database connection string
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable is not set")
	}

	// Connect to database
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Test connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	fmt.Println("Successfully connected to database")

	// 1. Count buses in the buses table
	var busCount int
	err = db.QueryRow("SELECT COUNT(*) FROM buses").Scan(&busCount)
	if err != nil {
		log.Printf("Error counting buses: %v", err)
	} else {
		fmt.Printf("\n1. Total buses in database: %d\n", busCount)
	}

	// 2. Get sample data from buses table using correct column names
	fmt.Println("\n2. Sample bus data:")
	rows, err := db.Query("SELECT id, bus_id, status, model, capacity, oil_status, tire_status, maintenance_notes FROM buses LIMIT 5")
	if err != nil {
		log.Printf("Error querying buses: %v", err)
	} else {
		defer rows.Close()
		for rows.Next() {
			var id int
			var busID, status string
			var model, oilStatus, tireStatus, maintenanceNotes sql.NullString
			var capacity sql.NullInt64
			
			err := rows.Scan(&id, &busID, &status, &model, &capacity, &oilStatus, &tireStatus, &maintenanceNotes)
			if err != nil {
				log.Printf("Error scanning row: %v", err)
				continue
			}
			
			fmt.Printf("   ID: %d, Bus ID: %s, Status: %s, Model: %s, Capacity: %v, Oil: %s, Tire: %s\n", 
				id, busID, status, model.String, capacity.Int64, oilStatus.String, tireStatus.String)
		}
	}

	// 3. Run the exact query from loadBusesFromDBPaginated
	fmt.Println("\n3. Running loadBusesFromDBPaginated query:")
	
	// First, let's check what columns exist in the buses table
	fmt.Println("\n   Checking bus table structure:")
	rows, err = db.Query(`
		SELECT column_name, data_type, is_nullable 
		FROM information_schema.columns 
		WHERE table_name = 'buses' 
		ORDER BY ordinal_position
	`)
	if err != nil {
		log.Printf("Error checking table structure: %v", err)
	} else {
		defer rows.Close()
		for rows.Next() {
			var columnName, dataType, isNullable string
			rows.Scan(&columnName, &dataType, &isNullable)
			fmt.Printf("   - %s (%s) nullable: %s\n", columnName, dataType, isNullable)
		}
	}

	// 4. Now run the actual query used in the fleet page (from loadBusesFromDBPaginated)
	fmt.Println("\n4. Simulating fleet page query (with pagination):")
	offset := 0
	limit := 10
	
	// Use the exact query from loadBusesFromDBPaginated
	query := `
		SELECT bus_id, status, model, capacity, oil_status, tire_status, 
		       maintenance_notes, current_mileage, last_oil_change, 
		       last_tire_service, updated_at, created_at 
		FROM buses 
		ORDER BY bus_id
		LIMIT $1 OFFSET $2
	`
	
	rows, err = db.Query(query, limit, offset)
	if err != nil {
		log.Printf("Error running paginated query: %v", err)
	}
	
	if rows != nil {
		defer rows.Close()
		count := 0
		for rows.Next() {
			count++
			// Scan the columns that loadBusesFromDBPaginated expects
			var busID, status string
			var model, oilStatus, tireStatus, maintenanceNotes sql.NullString
			var capacity, currentMileage, lastOilChange, lastTireService sql.NullInt64
			var updatedAt, createdAt sql.NullTime
			
			err := rows.Scan(&busID, &status, &model, &capacity, &oilStatus, &tireStatus, 
				&maintenanceNotes, &currentMileage, &lastOilChange, &lastTireService, &updatedAt, &createdAt)
			if err != nil {
				log.Printf("   Error scanning row: %v", err)
				continue
			}
			
			fmt.Printf("   Bus %s - Status: %s, Model: %s, Capacity: %v, Oil: %s, Tire: %s, Mileage: %v\n", 
				busID, status, model.String, capacity.Int64, oilStatus.String, tireStatus.String, currentMileage.Int64)
		}
		fmt.Printf("\n   Total buses retrieved: %d\n", count)
	}

	// 5. Test using sqlx Select like the app does
	fmt.Println("\n5. Testing sqlx Select method (same as app uses):")
	
	// Import sqlx if not already
	type Bus struct {
		BusID            string         `db:"bus_id"`
		Status           string         `db:"status"`
		Model            sql.NullString `db:"model"`
		Capacity         sql.NullInt64  `db:"capacity"`
		OilStatus        sql.NullString `db:"oil_status"`
		TireStatus       sql.NullString `db:"tire_status"`
		MaintenanceNotes sql.NullString `db:"maintenance_notes"`
		CurrentMileage   sql.NullInt64  `db:"current_mileage"`
		LastOilChange    sql.NullInt64  `db:"last_oil_change"`
		LastTireService  sql.NullInt64  `db:"last_tire_service"`
		UpdatedAt        sql.NullTime   `db:"updated_at"`
		CreatedAt        sql.NullTime   `db:"created_at"`
	}
	
	// Try a simple query first
	simpleQuery := "SELECT * FROM buses LIMIT 5"
	rows, err = db.Query(simpleQuery)
	if err != nil {
		log.Printf("Simple SELECT * failed: %v", err)
	} else {
		defer rows.Close()
		columns, _ := rows.Columns()
		fmt.Printf("   Columns returned by SELECT *: %v\n", columns)
	}
	
	// 6. Check if there's any issue with the database connection in the app
	fmt.Println("\n6. Additional diagnostics:")
	
	// Check for any errors in the logs table if it exists
	var hasLogsTable bool
	err = db.QueryRow("SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'logs')").Scan(&hasLogsTable)
	if err == nil && hasLogsTable {
		fmt.Println("   Checking recent error logs...")
		rows, err := db.Query("SELECT created_at, level, message FROM logs WHERE level = 'error' ORDER BY created_at DESC LIMIT 5")
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var createdAt sql.NullTime
				var level, message string
				rows.Scan(&createdAt, &level, &message)
				fmt.Printf("   [%s] %s: %s\n", createdAt.Time.Format("2006-01-02 15:04:05"), level, message)
			}
		}
	}

	// Check current database name and schema
	var dbName string
	err = db.QueryRow("SELECT current_database()").Scan(&dbName)
	if err == nil {
		fmt.Printf("\n   Current database: %s\n", dbName)
	}

	var schema string
	err = db.QueryRow("SELECT current_schema()").Scan(&schema)
	if err == nil {
		fmt.Printf("   Current schema: %s\n", schema)
	}

	fmt.Println("\nDebug complete!")
}