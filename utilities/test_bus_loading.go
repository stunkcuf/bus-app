package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

// Test Bus struct matching the one in models.go
type TestBus struct {
	BusID            string         `json:"bus_id" db:"bus_id"`
	Status           string         `json:"status" db:"status"`
	Model            sql.NullString `json:"model" db:"model"`
	Capacity         sql.NullInt32  `json:"capacity" db:"capacity"`
	OilStatus        sql.NullString `json:"oil_status" db:"oil_status"`
	TireStatus       sql.NullString `json:"tire_status" db:"tire_status"`
	MaintenanceNotes sql.NullString `json:"maintenance_notes" db:"maintenance_notes"`
	CurrentMileage   sql.NullInt32  `json:"current_mileage" db:"current_mileage"`
	LastOilChange    sql.NullInt32  `json:"last_oil_change" db:"last_oil_change"`
	LastTireService  sql.NullInt32  `json:"last_tire_service" db:"last_tire_service"`
	UpdatedAt        sql.NullTime   `json:"updated_at" db:"updated_at"`
	CreatedAt        sql.NullTime   `json:"created_at" db:"created_at"`
}

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

	// Connect using sqlx (same as the app)
	db, err := sqlx.Connect("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	fmt.Println("Successfully connected to database using sqlx")

	// Test 1: Try the exact query and struct from the app
	fmt.Println("\nTest 1: Using sqlx.Select with app's Bus struct")
	var buses []TestBus
	query := `
		SELECT bus_id, status, model, capacity, oil_status, tire_status, 
		       maintenance_notes, current_mileage, last_oil_change, 
		       last_tire_service, updated_at, created_at 
		FROM buses 
		ORDER BY bus_id
		LIMIT 10 OFFSET 0
	`
	
	err = db.Select(&buses, query)
	if err != nil {
		log.Printf("ERROR: db.Select failed: %v", err)
		
		// Try with Get to see if it's a scanning issue
		fmt.Println("\nTrying db.Get for a single bus:")
		var singleBus TestBus
		err = db.Get(&singleBus, "SELECT * FROM buses LIMIT 1")
		if err != nil {
			log.Printf("ERROR: db.Get also failed: %v", err)
		}
	} else {
		fmt.Printf("SUCCESS: Loaded %d buses using db.Select\n", len(buses))
		for i, bus := range buses {
			fmt.Printf("  Bus %d: ID=%s, Status=%s, Capacity=%v\n", 
				i+1, bus.BusID, bus.Status, bus.Capacity.Int32)
		}
	}

	// Test 2: Check if it's an int32 vs int64 issue
	fmt.Println("\nTest 2: Checking data types")
	row := db.QueryRow("SELECT capacity FROM buses WHERE capacity IS NOT NULL LIMIT 1")
	var capacityValue interface{}
	err = row.Scan(&capacityValue)
	if err != nil {
		log.Printf("ERROR scanning capacity: %v", err)
	} else {
		fmt.Printf("Capacity value type: %T, value: %v\n", capacityValue, capacityValue)
	}

	// Test 3: Try with sql.NullInt64 instead
	fmt.Println("\nTest 3: Using NullInt64 for integer fields")
	type TestBusInt64 struct {
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

	var busesInt64 []TestBusInt64
	err = db.Select(&busesInt64, query)
	if err != nil {
		log.Printf("ERROR: db.Select with Int64 failed: %v", err)
	} else {
		fmt.Printf("SUCCESS: Loaded %d buses using Int64 fields\n", len(busesInt64))
	}

	// Test 4: Check what the loadBusesFromDBPaginated function would see
	fmt.Println("\nTest 4: Simulating loadBusesFromDBPaginated behavior")
	fmt.Printf("Using formatted query like the function does...\n")
	
	formattedQuery := fmt.Sprintf(`
		SELECT bus_id, status, model, capacity, oil_status, tire_status, 
		       maintenance_notes, current_mileage, last_oil_change, 
		       last_tire_service, updated_at, created_at 
		FROM buses 
		ORDER BY bus_id
		LIMIT %d OFFSET %d
	`, 10, 0)
	
	var appBuses []TestBus
	err = db.Select(&appBuses, formattedQuery)
	if err != nil {
		log.Printf("ERROR: Formatted query failed: %v", err)
	} else {
		fmt.Printf("SUCCESS: Formatted query loaded %d buses\n", len(appBuses))
	}
}