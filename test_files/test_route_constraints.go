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
		dbURL = "postgresql://postgres:jTKOWGEzlprRGbkPBCgGxsgnwyLeGoDL@shortline.proxy.rlwy.net:40148/railway?sslmode=require"
	}
	
	// Connect to database
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()
	
	fmt.Println("=== Testing Route Assignment Constraints ===\n")
	
	// Check current route assignments
	fmt.Println("Current Route Assignments:")
	fmt.Println("--------------------------")
	rows, err := db.Query(`
		SELECT driver, bus_id, route_id, assigned_date 
		FROM route_assignments 
		ORDER BY driver, route_id
	`)
	if err != nil {
		log.Fatal("Failed to query route assignments:", err)
	}
	defer rows.Close()
	
	for rows.Next() {
		var driver, busID, routeID string
		var assignedDate sql.NullTime
		
		err := rows.Scan(&driver, &busID, &routeID, &assignedDate)
		if err != nil {
			log.Printf("Error scanning row: %v", err)
			continue
		}
		
		fmt.Printf("Driver: %-15s Bus: %-10s Route: %-10s Date: %s\n", 
			driver, busID, routeID, assignedDate.Time.Format("2006-01-02"))
	}
	
	// Test if we can assign a driver to multiple routes
	fmt.Println("\n\nTesting Multiple Route Assignment:")
	fmt.Println("----------------------------------")
	
	// Try to add a second route for bjmathis
	testDriver := "bjmathis"
	testBus := "24"  // Same bus she already has
	testRoute := "NELC-2"  // Different route
	
	fmt.Printf("Attempting to assign %s to route %s with bus %s...\n", testDriver, testRoute, testBus)
	
	_, err = db.Exec(`
		INSERT INTO route_assignments (driver, bus_id, route_id, assigned_date, created_at)
		VALUES ($1, $2, $3, CURRENT_DATE, CURRENT_TIMESTAMP)
		ON CONFLICT ON CONSTRAINT route_assignments_unique_assignment DO UPDATE
		SET assigned_date = CURRENT_DATE, created_at = CURRENT_TIMESTAMP
	`, testDriver, testBus, testRoute)
	
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Println("Success! Driver can be assigned to multiple routes.")
		
		// Show updated assignments
		fmt.Println("\nUpdated assignments for", testDriver + ":")
		rows2, err := db.Query(`
			SELECT bus_id, route_id FROM route_assignments 
			WHERE driver = $1
		`, testDriver)
		if err == nil {
			defer rows2.Close()
			for rows2.Next() {
				var busID, routeID string
				rows2.Scan(&busID, &routeID)
				fmt.Printf("  - Route %s with Bus %s\n", routeID, busID)
			}
		}
		
		// Clean up test data
		fmt.Println("\nCleaning up test assignment...")
		db.Exec(`DELETE FROM route_assignments WHERE driver = $1 AND route_id = $2`, testDriver, testRoute)
	}
	
	// Check table constraints
	fmt.Println("\n\nTable Constraints:")
	fmt.Println("------------------")
	rows3, err := db.Query(`
		SELECT conname, pg_get_constraintdef(oid) 
		FROM pg_constraint 
		WHERE conrelid = 'route_assignments'::regclass
	`)
	if err == nil {
		defer rows3.Close()
		for rows3.Next() {
			var name, def string
			rows3.Scan(&name, &def)
			fmt.Printf("%s: %s\n", name, def)
		}
	}
}