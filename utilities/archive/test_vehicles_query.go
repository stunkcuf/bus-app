package main

import (
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
	"github.com/jmoiron/sqlx"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgresql://postgres:jTKOWGEzlprRGbkPBCgGxsgnwyLeGoDL@shortline.proxy.rlwy.net:40148/railway"
	}

	db, err := sqlx.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer db.Close()

	// Set connection pool settings like main app
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	fmt.Println("Testing vehicles query...")

	// First, check table structure
	var cols []struct {
		Name string `db:"column_name"`
		Type string `db:"data_type"`
	}
	
	err = db.Select(&cols, `
		SELECT column_name, data_type 
		FROM information_schema.columns 
		WHERE table_name = 'vehicles' 
		ORDER BY ordinal_position
	`)
	
	if err != nil {
		fmt.Printf("Error getting columns: %v\n", err)
	} else {
		fmt.Println("\nVehicles table structure:")
		for _, col := range cols {
			fmt.Printf("  - %s (%s)\n", col.Name, col.Type)
		}
	}

	// Test basic count
	var count int
	err = db.Get(&count, "SELECT COUNT(*) FROM vehicles")
	if err != nil {
		fmt.Printf("Error counting vehicles: %v\n", err)
	} else {
		fmt.Printf("\nTotal vehicles: %d\n", count)
	}

	// Test the actual query with timeout
	fmt.Println("\nTesting full query (10 second timeout)...")
	
	rows, err := db.Query(`
		SELECT vehicle_id, model, description, year, tire_size, license, 
		       oil_status, tire_status, status, maintenance_notes, 
		       serial_number, base, service_interval, current_mileage, 
		       last_oil_change, last_tire_service, updated_at, created_at, import_id
		FROM vehicles 
		ORDER BY vehicle_id
		LIMIT 5
	`)
	
	if err != nil {
		fmt.Printf("❌ Query failed: %v\n", err)
		return
	}
	defer rows.Close()

	// Count results
	rowCount := 0
	for rows.Next() {
		rowCount++
		var vehicleID string
		rows.Scan(&vehicleID)
		fmt.Printf("  Row %d: vehicle_id = %s\n", rowCount, vehicleID)
	}
	
	fmt.Printf("\n✅ Query successful - returned %d rows\n", rowCount)
}