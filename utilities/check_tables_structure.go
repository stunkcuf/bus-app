package main

import (
	"database/sql"
	"fmt"
	"log"
	
	_ "github.com/lib/pq"
)

func main() {
	dbURL := "postgresql://postgres:jTKOWGEzlprRGbkPBCgGxsgnwyLeGoDL@shortline.proxy.rlwy.net:40148/railway"
	
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Check buses table
	fmt.Println("=== BUSES TABLE ===")
	rows, err := db.Query(`
		SELECT column_name 
		FROM information_schema.columns 
		WHERE table_name = 'buses'
		ORDER BY ordinal_position
	`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var col string
			rows.Scan(&col)
			fmt.Printf("  - %s\n", col)
		}
	}

	// Check if buses have status column
	var hasStatus bool
	db.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM information_schema.columns 
			WHERE table_name = 'buses' AND column_name = 'status'
		)
	`).Scan(&hasStatus)
	fmt.Printf("\nBuses table has 'status' column: %v\n", hasStatus)

	// Check active buses
	fmt.Println("\n=== ACTIVE BUSES ===")
	var query string
	if hasStatus {
		query = `SELECT bus_id FROM buses WHERE status = 'active' ORDER BY bus_id`
	} else {
		query = `SELECT bus_id FROM buses ORDER BY bus_id`
	}
	
	busRows, err := db.Query(query)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		defer busRows.Close()
		count := 0
		for busRows.Next() {
			var busID string
			busRows.Scan(&busID)
			if count < 10 { // Show first 10
				fmt.Printf("  Bus %s\n", busID)
			}
			count++
		}
		fmt.Printf("  Total: %d buses\n", count)
	}

	// Check users table
	fmt.Println("\n=== DRIVERS ===")
	var hasStatusUser bool
	db.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM information_schema.columns 
			WHERE table_name = 'users' AND column_name = 'status'
		)
	`).Scan(&hasStatusUser)
	
	if hasStatusUser {
		query = `SELECT username FROM users WHERE role = 'driver' AND status = 'active' ORDER BY username`
	} else {
		query = `SELECT username FROM users WHERE role = 'driver' ORDER BY username`
	}
	
	driverRows, err := db.Query(query)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		defer driverRows.Close()
		count := 0
		for driverRows.Next() {
			var driver string
			driverRows.Scan(&driver)
			fmt.Printf("  - %s\n", driver)
			count++
		}
		fmt.Printf("  Total: %d drivers\n", count)
	}
}