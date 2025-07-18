// +build ignore

package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

func main() {
	// Use the same connection string as in production
	dbURL := "postgresql://postgres:jTKOWGEzlprRGbkPBCgGxsgnwyLeGoDL@shortline.proxy.rlwy.net:40148/railway"

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

	fmt.Println("Connected to database successfully!")

	// Check buses table
	var busCount int
	err = db.QueryRow("SELECT COUNT(*) FROM buses").Scan(&busCount)
	if err != nil {
		log.Printf("Error counting buses: %v", err)
	} else {
		fmt.Printf("\nNumber of buses in database: %d\n", busCount)
	}

	// Check vehicles table
	var vehicleCount int
	err = db.QueryRow("SELECT COUNT(*) FROM vehicles").Scan(&vehicleCount)
	if err != nil {
		log.Printf("Error counting vehicles: %v", err)
	} else {
		fmt.Printf("Number of vehicles in database: %d\n", vehicleCount)
	}

	// Check the column structure of buses table
	fmt.Println("\nBuses table columns:")
	rows, err := db.Query(`
		SELECT column_name, data_type 
		FROM information_schema.columns 
		WHERE table_name = 'buses' 
		ORDER BY ordinal_position
	`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var colName, dataType string
			rows.Scan(&colName, &dataType)
			fmt.Printf("  - %s (%s)\n", colName, dataType)
		}
	}

	// Check the column structure of vehicles table
	fmt.Println("\nVehicles table columns:")
	rows, err = db.Query(`
		SELECT column_name, data_type 
		FROM information_schema.columns 
		WHERE table_name = 'vehicles' 
		ORDER BY ordinal_position
	`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var colName, dataType string
			rows.Scan(&colName, &dataType)
			fmt.Printf("  - %s (%s)\n", colName, dataType)
		}
	}

	// If no data, offer to add sample data
	if busCount == 0 && vehicleCount == 0 {
		fmt.Println("\nNo data found in buses or vehicles tables!")
		fmt.Println("The database tables exist but are empty.")
	}
}