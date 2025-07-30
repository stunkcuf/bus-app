package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found: %v", err)
	}

	// Get database URL
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL not set")
	}

	// Connect to database
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	fmt.Println("Fleet Data Debug Check")
	fmt.Println("=====================")

	// 1. Check total bus count
	var busCount int
	err = db.QueryRow("SELECT COUNT(*) FROM buses").Scan(&busCount)
	if err != nil {
		fmt.Printf("ERROR getting bus count: %v\n", err)
	} else {
		fmt.Printf("\n1. Total buses in database: %d\n", busCount)
	}

	// 2. Check bus status distribution
	fmt.Println("\n2. Bus status distribution:")
	rows, err := db.Query(`
		SELECT status, COUNT(*) as count 
		FROM buses 
		GROUP BY status 
		ORDER BY count DESC
	`)
	if err != nil {
		fmt.Printf("ERROR querying bus status: %v\n", err)
	} else {
		defer rows.Close()
		for rows.Next() {
			var status sql.NullString
			var count int
			if err := rows.Scan(&status, &count); err == nil {
				if status.Valid {
					fmt.Printf("   - %s: %d buses\n", status.String, count)
				} else {
					fmt.Printf("   - (NULL): %d buses\n", count)
				}
			}
		}
	}

	// 3. List first 10 buses
	fmt.Println("\n3. First 10 buses (same as fleet page):")
	rows, err = db.Query(`
		SELECT bus_id, model, capacity, status, oil_status, tire_status 
		FROM buses 
		ORDER BY bus_id 
		LIMIT 10
	`)
	if err != nil {
		fmt.Printf("ERROR querying buses: %v\n", err)
	} else {
		defer rows.Close()
		i := 0
		for rows.Next() {
			var busID string
			var model, status, oilStatus, tireStatus sql.NullString
			var capacity sql.NullInt64
			
			if err := rows.Scan(&busID, &model, &capacity, &status, &oilStatus, &tireStatus); err == nil {
				i++
				fmt.Printf("\n   Bus #%d:\n", i)
				fmt.Printf("   - ID: %s\n", busID)
				if model.Valid {
					fmt.Printf("   - Model: %s\n", model.String)
				}
				if capacity.Valid {
					fmt.Printf("   - Capacity: %d\n", capacity.Int64)
				}
				if status.Valid {
					fmt.Printf("   - Status: %s\n", status.String)
				}
				if oilStatus.Valid {
					fmt.Printf("   - Oil Status: %s\n", oilStatus.String)
				}
				if tireStatus.Valid {
					fmt.Printf("   - Tire Status: %s\n", tireStatus.String)
				}
			}
		}
		if i == 0 {
			fmt.Println("   No buses found!")
		}
	}

	// 4. Check vehicle count
	var vehicleCount int
	err = db.QueryRow("SELECT COUNT(*) FROM vehicles").Scan(&vehicleCount)
	if err != nil {
		fmt.Printf("\nERROR getting vehicle count: %v\n", err)
	} else {
		fmt.Printf("\n4. Total vehicles (non-bus) in database: %d\n", vehicleCount)
	}

	// 5. Check if fleet handler would see data
	fmt.Println("\n5. Simulating fleet handler query:")
	
	// This matches the query in fleet_handler_clean.go
	testQuery := `
		SELECT COUNT(*) FROM buses 
		WHERE 1=1
	`
	var testCount int
	err = db.QueryRow(testQuery).Scan(&testCount)
	if err != nil {
		fmt.Printf("ERROR in test query: %v\n", err)
	} else {
		fmt.Printf("   Fleet handler would see: %d buses\n", testCount)
	}

	fmt.Println("\n6. Debug messages that should appear in server logs:")
	fmt.Println("   - DEBUG: Fleet handler called by user: [username]")
	fmt.Printf("   - DEBUG: Total bus count in database: %d\n", busCount)
	fmt.Println("   - DEBUG: About to call loadBusesFromDBPaginated")
	fmt.Println("   - SUCCESS: Loaded 10 buses (page 1 of X)")
	fmt.Println("   - DEBUG: Before rendering - Data.Buses count: 10")
	fmt.Println("\nNote: These messages should appear in the server console when accessing /fleet")
}