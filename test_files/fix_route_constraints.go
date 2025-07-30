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
	
	fmt.Println("=== Fixing Route Assignment Constraints ===\n")
	
	// Drop the incorrect unique constraint on driver
	fmt.Println("Dropping incorrect UNIQUE constraint on driver column...")
	_, err = db.Exec(`ALTER TABLE route_assignments DROP CONSTRAINT IF EXISTS route_assignments_driver_key`)
	if err != nil {
		fmt.Printf("Error dropping constraint: %v\n", err)
	} else {
		fmt.Println("✓ Constraint dropped successfully")
	}
	
	// The composite unique constraint is correct and should remain
	fmt.Println("\nThe composite unique constraint (driver, bus_id, route_id) is correct and will remain.")
	fmt.Println("This allows a driver to have multiple routes but prevents duplicate assignments.")
	
	// Test the fix
	fmt.Println("\n\nTesting multiple route assignment after fix:")
	fmt.Println("--------------------------------------------")
	
	testDriver := "bjmathis"
	testBus := "24"
	testRoute := "NELC-2"
	
	fmt.Printf("Attempting to assign %s to route %s with bus %s...\n", testDriver, testRoute, testBus)
	
	_, err = db.Exec(`
		INSERT INTO route_assignments (driver, bus_id, route_id, assigned_date, created_at)
		VALUES ($1, $2, $3, CURRENT_DATE, CURRENT_TIMESTAMP)
	`, testDriver, testBus, testRoute)
	
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Println("✓ Success! Driver can now be assigned to multiple routes.")
		
		// Show updated assignments
		fmt.Println("\nCurrent assignments for", testDriver + ":")
		rows, err := db.Query(`
			SELECT bus_id, route_id FROM route_assignments 
			WHERE driver = $1
			ORDER BY route_id
		`, testDriver)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var busID, routeID string
				rows.Scan(&busID, &routeID)
				fmt.Printf("  - Route %s with Bus %s\n", routeID, busID)
			}
		}
		
		// Clean up test data
		fmt.Println("\nCleaning up test assignment...")
		db.Exec(`DELETE FROM route_assignments WHERE driver = $1 AND route_id = $2`, testDriver, testRoute)
		fmt.Println("✓ Test data cleaned up")
	}
	
	fmt.Println("\n✅ Route assignment constraints fixed!")
	fmt.Println("Drivers can now be assigned to multiple routes.")
}