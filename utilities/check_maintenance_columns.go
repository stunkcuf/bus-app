package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	// Load environment variables
	if err := godotenv.Load("../.env"); err != nil {
		log.Printf("Warning: .env file not found: %v", err)
	}

	// Connect to database
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable is not set")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Query column information
	query := `
		SELECT column_name, data_type, is_nullable, column_default
		FROM information_schema.columns
		WHERE table_name = 'maintenance_records'
		ORDER BY ordinal_position;
	`

	rows, err := db.Query(query)
	if err != nil {
		log.Fatalf("Failed to query columns: %v", err)
	}
	defer rows.Close()

	fmt.Println("Columns in maintenance_records table:")
	fmt.Println("=====================================")
	
	for rows.Next() {
		var columnName, dataType, isNullable string
		var columnDefault sql.NullString
		
		err := rows.Scan(&columnName, &dataType, &isNullable, &columnDefault)
		if err != nil {
			log.Printf("Error scanning row: %v", err)
			continue
		}
		
		fmt.Printf("- %s (%s) nullable=%s", columnName, dataType, isNullable)
		if columnDefault.Valid {
			fmt.Printf(" default=%s", columnDefault.String)
		}
		fmt.Println()
	}

	// Also check first few records
	fmt.Println("\nFirst 3 records (raw data):")
	fmt.Println("===========================")
	
	dataQuery := `SELECT * FROM maintenance_records LIMIT 3`
	dataRows, err := db.Query(dataQuery)
	if err != nil {
		log.Printf("Failed to query data: %v", err)
		return
	}
	defer dataRows.Close()

	// Get column names
	columns, err := dataRows.Columns()
	if err != nil {
		log.Printf("Failed to get columns: %v", err)
		return
	}

	fmt.Printf("Columns: %v\n\n", columns)

	// Create a slice of interface{} to hold column values
	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	for i := range columns {
		valuePtrs[i] = &values[i]
	}

	recordNum := 1
	for dataRows.Next() {
		err := dataRows.Scan(valuePtrs...)
		if err != nil {
			log.Printf("Error scanning row: %v", err)
			continue
		}

		fmt.Printf("Record %d:\n", recordNum)
		for i, col := range columns {
			fmt.Printf("  %s: %v\n", col, values[i])
		}
		fmt.Println()
		recordNum++
	}
}