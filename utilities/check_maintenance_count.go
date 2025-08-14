package main

import (
	"fmt"
	"log"
	"os"
	"time"
	_ "github.com/lib/pq"
	"github.com/jmoiron/sqlx"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgresql://postgres:jTKOWGEzlprRGbkPBCgGxsgnwyLeGoDL@shortline.proxy.rlwy.net:40148/railway"
	}

	db, err := sqlx.Connect("postgres", dbURL)
	if err != nil {
		log.Fatal("Failed to connect:", err)
	}
	defer db.Close()

	// Count total records
	var totalCount int
	err = db.Get(&totalCount, "SELECT COUNT(*) FROM maintenance_records")
	if err != nil {
		log.Fatal("Failed to count records:", err)
	}
	fmt.Printf("Total maintenance records: %d\n", totalCount)

	// Check if any records have huge raw_data
	var avgRawDataSize float64
	err = db.Get(&avgRawDataSize, "SELECT AVG(LENGTH(raw_data::text)) FROM maintenance_records WHERE raw_data IS NOT NULL")
	if err != nil {
		log.Printf("Error checking raw_data size: %v", err)
	} else {
		fmt.Printf("Average raw_data size: %.0f bytes\n", avgRawDataSize)
	}

	// Check largest raw_data
	var maxRawDataSize int
	err = db.Get(&maxRawDataSize, "SELECT MAX(LENGTH(raw_data::text)) FROM maintenance_records WHERE raw_data IS NOT NULL")
	if err != nil {
		log.Printf("Error checking max raw_data size: %v", err)
	} else {
		fmt.Printf("Largest raw_data: %d bytes\n", maxRawDataSize)
	}

	// Test the actual query performance
	fmt.Println("\nTesting query performance...")
	start := time.Now()
	
	rows, err := db.Query(`
		SELECT id, vehicle_number, service_date, mileage, po_number, cost,
		       work_description, raw_data, created_at, updated_at, vehicle_id, date
		FROM maintenance_records 
		ORDER BY 
			COALESCE(service_date, date, created_at) DESC,
			vehicle_number, id
		LIMIT 100
	`)
	
	if err != nil {
		log.Fatal("Query failed:", err)
	}
	defer rows.Close()
	
	count := 0
	for rows.Next() {
		count++
	}
	
	elapsed := time.Since(start)
	fmt.Printf("Query for 100 records took: %v\n", elapsed)
	fmt.Printf("Records fetched: %d\n", count)

	// Check for problematic data
	fmt.Println("\nChecking for problematic data...")
	var nullServiceDateCount int
	err = db.Get(&nullServiceDateCount, "SELECT COUNT(*) FROM maintenance_records WHERE service_date IS NULL AND date IS NULL")
	if err != nil {
		log.Printf("Error checking null dates: %v", err)
	} else {
		fmt.Printf("Records with both service_date and date NULL: %d\n", nullServiceDateCount)
	}
}