package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
)

func main() {
	// Get database URL from environment
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable not set")
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

	fmt.Println("Connected to database successfully!")

	// Check buses table
	var busCount int
	err = db.QueryRow("SELECT COUNT(*) FROM buses").Scan(&busCount)
	if err != nil {
		log.Printf("Error counting buses: %v", err)
	} else {
		fmt.Printf("Number of buses in database: %d\n", busCount)
	}

	// Check vehicles table
	var vehicleCount int
	err = db.QueryRow("SELECT COUNT(*) FROM vehicles").Scan(&vehicleCount)
	if err != nil {
		log.Printf("Error counting vehicles: %v", err)
	} else {
		fmt.Printf("Number of vehicles in database: %d\n", vehicleCount)
	}

	// Show sample bus data if any
	if busCount > 0 {
		fmt.Println("\nSample bus data:")
		rows, err := db.Query("SELECT bus_id, model, status FROM buses LIMIT 3")
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var busID, model, status string
				rows.Scan(&busID, &model, &status)
				fmt.Printf("- Bus %s: %s (Status: %s)\n", busID, model, status)
			}
		}
	}

	// Show sample vehicle data if any
	if vehicleCount > 0 {
		fmt.Println("\nSample vehicle data:")
		rows, err := db.Query("SELECT vehicle_id, model, status FROM vehicles LIMIT 3")
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var vehicleID, model, status string
				rows.Scan(&vehicleID, &model, &status)
				fmt.Printf("- Vehicle %s: %s (Status: %s)\n", vehicleID, model, status)
			}
		}
	}
}