package main

import (
	"database/sql"
	"log"
	"os"
	_ "github.com/lib/pq"
)

func main() {
	// Get database URL
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:Savage1995!@viaduct.proxy.rlwy.net:51688/railway?sslmode=disable"
	}

	// Connect to database
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Check table columns
	query := `
		SELECT column_name, data_type 
		FROM information_schema.columns 
		WHERE table_name = 'ecse_students' 
		ORDER BY ordinal_position
	`
	
	rows, err := db.Query(query)
	if err != nil {
		log.Fatalf("Failed to query columns: %v", err)
	}
	defer rows.Close()

	log.Println("ECSE Students table columns:")
	for rows.Next() {
		var columnName, dataType string
		if err := rows.Scan(&columnName, &dataType); err != nil {
			log.Printf("Error scanning row: %v", err)
			continue
		}
		log.Printf("  - %s (%s)", columnName, dataType)
	}

	// Check for any data
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM ecse_students").Scan(&count)
	if err != nil {
		log.Printf("Error counting records: %v", err)
	} else {
		log.Printf("\nTotal ECSE students in table: %d", count)
	}

	// Check fuel_records table
	log.Println("\nFuel Records table columns:")
	rows2, err := db.Query(`
		SELECT column_name, data_type 
		FROM information_schema.columns 
		WHERE table_name = 'fuel_records' 
		ORDER BY ordinal_position
	`)
	if err != nil {
		log.Printf("Failed to query fuel_records columns: %v", err)
	} else {
		defer rows2.Close()
		for rows2.Next() {
			var columnName, dataType string
			if err := rows2.Scan(&columnName, &dataType); err != nil {
				log.Printf("Error scanning row: %v", err)
				continue
			}
			log.Printf("  - %s (%s)", columnName, dataType)
		}
	}

	// Check fuel records count
	err = db.QueryRow("SELECT COUNT(*) FROM fuel_records").Scan(&count)
	if err != nil {
		log.Printf("Error counting fuel records: %v", err)
	} else {
		log.Printf("\nTotal fuel records in table: %d", count)
	}
}