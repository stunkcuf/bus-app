package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
	"github.com/jmoiron/sqlx"
)

func main() {
	// Connect to database
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgresql://postgres:jTKOWGEzlprRGbkPBCgGxsgnwyLeGoDL@shortline.proxy.rlwy.net:40148/railway"
	}

	db, err := sqlx.Connect("postgres", dbURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// First, check what columns exist in the vehicles table
	fmt.Println("Checking vehicles table structure...")
	var columns []struct {
		ColumnName string `db:"column_name"`
		DataType   string `db:"data_type"`
	}
	
	err = db.Select(&columns, `
		SELECT column_name, data_type 
		FROM information_schema.columns 
		WHERE table_name = 'vehicles' 
		ORDER BY ordinal_position
	`)
	
	if err != nil {
		log.Fatal("Failed to get columns:", err)
	}
	
	fmt.Println("Vehicles table columns:")
	for _, col := range columns {
		fmt.Printf("  - %s (%s)\n", col.ColumnName, col.DataType)
	}

	// Try the query
	fmt.Println("\nTesting fleet vehicles query...")
	query := `
		SELECT 
			CASE 
				WHEN vehicle_id LIKE 'FV%' THEN SUBSTRING(vehicle_id FROM 3)::INTEGER
				ELSE 1
			END as id,
			NULL::INTEGER as vehicle_number, 
			NULL as sheet_name, 
			CASE 
				WHEN year ~ '^\d+$' THEN year::INTEGER
				ELSE NULL
			END as year,
			make, model, 
		    description, serial_number, license, base as location, tire_size,
		    created_at, updated_at
		FROM vehicles 
		WHERE status = 'active'
		ORDER BY 
			vehicle_id,
			year DESC, model
		LIMIT 5`

	rows, err := db.Query(query)
	if err != nil {
		log.Fatal("Query failed:", err)
	}
	defer rows.Close()

	fmt.Println("Query executed successfully! First 5 results:")
	
	count := 0
	for rows.Next() {
		var id int
		var vehicleNumber sql.NullInt32
		var sheetName, year sql.NullString
		var make, model, description, serialNumber, license, location, tireSize sql.NullString
		var createdAt, updatedAt string
		
		err := rows.Scan(&id, &vehicleNumber, &sheetName, &year, 
			&make, &model, &description, &serialNumber, &license, 
			&location, &tireSize, &createdAt, &updatedAt)
		
		if err != nil {
			log.Printf("Row scan error: %v", err)
			continue
		}
		
		count++
		fmt.Printf("%d. Model: %s, Year: %s, License: %s\n", 
			count, model.String, year.String, license.String)
	}
	
	if count == 0 {
		fmt.Println("No vehicles found in database")
	} else {
		fmt.Printf("Total displayed: %d vehicles\n", count)
	}
}