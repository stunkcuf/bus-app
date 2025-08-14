package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL not set")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Check constraints
	fmt.Println("Checking route_assignments constraints...")
	
	query := `
		SELECT 
			con.conname AS constraint_name,
			con.contype AS constraint_type,
			pg_get_constraintdef(con.oid) AS definition
		FROM pg_constraint con
		JOIN pg_namespace nsp ON con.connamespace = nsp.oid
		JOIN pg_class cls ON con.conrelid = cls.oid
		WHERE cls.relname = 'route_assignments'
		AND nsp.nspname = 'public'
	`
	
	rows, err := db.Query(query)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	
	for rows.Next() {
		var name, ctype, def string
		err := rows.Scan(&name, &ctype, &def)
		if err != nil {
			log.Printf("Error scanning: %v", err)
			continue
		}
		fmt.Printf("\nConstraint: %s\nType: %s\nDefinition: %s\n", name, ctype, def)
	}
	
	// Check if we can assign multiple routes to same driver
	fmt.Println("\n\nTesting multiple route assignment...")
	
	// Try to insert test data
	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}
	defer tx.Rollback()
	
	// Clear test assignments
	_, err = tx.Exec("DELETE FROM route_assignments WHERE driver = 'test_multi_driver'")
	if err != nil {
		fmt.Printf("Error clearing test data: %v\n", err)
	}
	
	// Try to assign multiple routes
	_, err = tx.Exec(`
		INSERT INTO route_assignments (driver, bus_id, route_id, assigned_date)
		VALUES ('test_multi_driver', '999', 'RT001', CURRENT_DATE)
	`)
	if err != nil {
		fmt.Printf("Error inserting first route: %v\n", err)
	} else {
		fmt.Println("✓ First route assigned successfully")
	}
	
	_, err = tx.Exec(`
		INSERT INTO route_assignments (driver, bus_id, route_id, assigned_date)
		VALUES ('test_multi_driver', '999', 'RT002', CURRENT_DATE)
	`)
	if err != nil {
		fmt.Printf("✗ Error inserting second route: %v\n", err)
		fmt.Println("This indicates a constraint preventing multiple routes per driver")
	} else {
		fmt.Println("✓ Second route assigned successfully")
		fmt.Println("Multiple routes per driver ARE supported!")
	}
	
	// Don't commit test data
	fmt.Println("\nRolling back test data...")
}