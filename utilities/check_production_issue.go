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

	// Test the exact query that would be used
	fmt.Println("Testing SELECT * FROM vehicles query:")
	
	rows, err := db.Query("SELECT * FROM vehicles ORDER BY vehicle_id LIMIT 1")
	if err != nil {
		fmt.Printf("ERROR executing query: %v\n", err)
		return
	}
	defer rows.Close()

	// Get column names
	cols, err := rows.Columns()
	if err != nil {
		fmt.Printf("ERROR getting columns: %v\n", err)
		return
	}

	fmt.Printf("Number of columns returned: %d\n", len(cols))
	fmt.Println("Column names:")
	for i, col := range cols {
		fmt.Printf("  %d: %s\n", i+1, col)
	}

	// Try to scan one row
	if rows.Next() {
		// Create a slice of interface{} to hold the values
		values := make([]interface{}, len(cols))
		valuePtrs := make([]interface{}, len(cols))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		// Scan the row
		if err := rows.Scan(valuePtrs...); err != nil {
			fmt.Printf("ERROR scanning row: %v\n", err)
		} else {
			fmt.Println("\nFirst row data:")
			for i, col := range cols {
				fmt.Printf("  %s: %v\n", col, values[i])
			}
		}
	}

	// Check if there are NULL values that might cause issues
	fmt.Println("\nChecking for NULL values in critical columns:")
	var nullCount int
	err = db.QueryRow(`
		SELECT COUNT(*) 
		FROM vehicles 
		WHERE vehicle_id IS NULL 
		   OR model IS NULL 
		   OR status IS NULL
	`).Scan(&nullCount)
	if err == nil && nullCount > 0 {
		fmt.Printf("WARNING: Found %d vehicles with NULL in critical columns\n", nullCount)
	}
}