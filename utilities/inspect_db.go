package main

import (
	"database/sql"
	"fmt"
	"log"
	_ "github.com/lib/pq"
)

func main() {
	// Use the actual database connection
	dbURL := "postgresql://postgres:jTKOWGEzlprRGbkPBCgGxsgnwyLeGoDL@shortline.proxy.rlwy.net:40148/railway"

	// Connect to database
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Test connection
	err = db.Ping()
	if err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	fmt.Println("Connected to database successfully!")
	fmt.Println("\n=== Database Tables ===")

	// Get all tables
	rows, err := db.Query(`
		SELECT table_name 
		FROM information_schema.tables 
		WHERE table_schema = 'public' 
		AND table_type = 'BASE TABLE'
		ORDER BY table_name
	`)
	if err != nil {
		log.Fatalf("Failed to query tables: %v", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			log.Printf("Error scanning table: %v", err)
			continue
		}
		tables = append(tables, tableName)
		
		// Get row count
		var count int
		err = db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)).Scan(&count)
		if err != nil {
			fmt.Printf("%-30s: Error counting rows: %v\n", tableName, err)
		} else {
			fmt.Printf("%-30s: %d rows\n", tableName, count)
		}
	}

	// Check for ECSE data specifically
	fmt.Println("\n=== ECSE Related Data ===")
	
	// Check ecse_records table
	var ecseCount int
	err = db.QueryRow("SELECT COUNT(*) FROM ecse_records").Scan(&ecseCount)
	if err == nil {
		fmt.Printf("ecse_records table has %d rows\n", ecseCount)
		
		// Show sample ECSE records
		if ecseCount > 0 {
			fmt.Println("\nSample ECSE records:")
			sampleRows, err := db.Query("SELECT * FROM ecse_records LIMIT 3")
			if err == nil {
				defer sampleRows.Close()
				
				// Get column names
				columns, _ := sampleRows.Columns()
				fmt.Printf("Columns: %v\n", columns)
			}
		}
	}

	// Check students table for ECSE data
	err = db.QueryRow(`
		SELECT COUNT(*) FROM students 
		WHERE student_id LIKE '%ECSE%' OR route_id LIKE '%ECSE%'
	`).Scan(&ecseCount)
	if err == nil && ecseCount > 0 {
		fmt.Printf("\nFound %d ECSE-related entries in students table\n", ecseCount)
	}

	// Check for fuel_records
	var fuelCount int
	err = db.QueryRow("SELECT COUNT(*) FROM fuel_records").Scan(&fuelCount)
	if err == nil {
		fmt.Printf("\nfuel_records table has %d rows\n", fuelCount)
		
		// Check column structure
		cols, err := db.Query(`
			SELECT column_name, data_type 
			FROM information_schema.columns 
			WHERE table_name = 'fuel_records' 
			ORDER BY ordinal_position
		`)
		if err == nil {
			defer cols.Close()
			fmt.Println("Fuel records columns:")
			for cols.Next() {
				var colName, dataType string
				cols.Scan(&colName, &dataType)
				fmt.Printf("  - %s (%s)\n", colName, dataType)
			}
		}
	}

	// Check maintenance tables
	fmt.Println("\n=== Maintenance Tables ===")
	maintTables := []string{"bus_maintenance_logs", "vehicle_maintenance_logs", "maintenance_records", "bus_maintenance_log"}
	for _, table := range maintTables {
		var count int
		err = db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&count)
		if err == nil {
			fmt.Printf("%s: %d rows\n", table, count)
		}
	}
}