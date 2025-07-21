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
		log.Fatalf("Failed to connect: %v", err)
	}
	defer db.Close()

	// Check ECSE students table structure
	fmt.Println("=== ECSE Students Table Structure ===")
	rows, err := db.Query(`
		SELECT column_name, data_type, is_nullable
		FROM information_schema.columns 
		WHERE table_name = 'ecse_students' 
		ORDER BY ordinal_position
	`)
	if err != nil {
		log.Fatalf("Failed to query columns: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var colName, dataType, isNullable string
		rows.Scan(&colName, &dataType, &isNullable)
		fmt.Printf("%-25s %-20s %s\n", colName, dataType, isNullable)
	}

	// Show sample ECSE data
	fmt.Println("\n=== Sample ECSE Students ===")
	sampleRows, err := db.Query("SELECT * FROM ecse_students LIMIT 3")
	if err != nil {
		log.Printf("Error querying ECSE students: %v", err)
		return
	}
	defer sampleRows.Close()

	columns, _ := sampleRows.Columns()
	fmt.Printf("Columns: %v\n", columns)
	
	// Check maintenance_records structure
	fmt.Println("\n=== Maintenance Records Structure ===")
	rows2, err := db.Query(`
		SELECT column_name, data_type
		FROM information_schema.columns 
		WHERE table_name = 'maintenance_records' 
		ORDER BY ordinal_position
		LIMIT 10
	`)
	if err == nil {
		defer rows2.Close()
		for rows2.Next() {
			var colName, dataType string
			rows2.Scan(&colName, &dataType)
			fmt.Printf("%-25s %s\n", colName, dataType)
		}
	}
}