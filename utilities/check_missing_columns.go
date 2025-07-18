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

	fmt.Println("Checking for missing columns in buses table...")

	// Check if current_mileage, last_oil_change, last_tire_service exist
	missingCols := []string{"current_mileage", "last_oil_change", "last_tire_service"}
	
	for _, col := range missingCols {
		var exists bool
		err := db.QueryRow(`
			SELECT EXISTS (
				SELECT 1 
				FROM information_schema.columns 
				WHERE table_name = 'buses' AND column_name = $1
			)
		`, col).Scan(&exists)
		
		if err != nil {
			log.Printf("Error checking column %s: %v", col, err)
		} else if !exists {
			fmt.Printf("Column %s is MISSING from buses table\n", col)
		} else {
			fmt.Printf("Column %s exists in buses table\n", col)
		}
	}

	fmt.Println("\nChecking for missing columns in vehicles table...")
	
	// Check vehicles table
	vehicleCols := []string{"current_mileage", "last_oil_change", "last_tire_service"}
	
	for _, col := range vehicleCols {
		var exists bool
		err := db.QueryRow(`
			SELECT EXISTS (
				SELECT 1 
				FROM information_schema.columns 
				WHERE table_name = 'vehicles' AND column_name = $1
			)
		`, col).Scan(&exists)
		
		if err != nil {
			log.Printf("Error checking column %s: %v", col, err)
		} else if !exists {
			fmt.Printf("Column %s is MISSING from vehicles table\n", col)
		} else {
			fmt.Printf("Column %s exists in vehicles table\n", col)
		}
	}

	// Show actual bus table structure
	fmt.Println("\nActual buses table structure:")
	rows, err := db.Query(`
		SELECT column_name, data_type, is_nullable, column_default
		FROM information_schema.columns 
		WHERE table_name = 'buses' 
		ORDER BY ordinal_position
	`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var colName, dataType, isNullable string
			var colDefault sql.NullString
			rows.Scan(&colName, &dataType, &isNullable, &colDefault)
			defaultStr := ""
			if colDefault.Valid {
				defaultStr = fmt.Sprintf(" DEFAULT %s", colDefault.String)
			}
			fmt.Printf("  - %s: %s (nullable: %s)%s\n", colName, dataType, isNullable, defaultStr)
		}
	}
}