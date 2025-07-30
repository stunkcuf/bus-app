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
		log.Fatal("DATABASE_URL environment variable not set")
	}

	db, err := sql.Open("postgres", dbURL)
	if err \!= nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	fmt.Println("🔍 Checking database schema...")

	// Check service_records
	rows, err := db.Query(`SELECT column_name FROM information_schema.columns WHERE table_name = 'service_records' ORDER BY ordinal_position`)
	if err \!= nil {
		log.Fatalf("Query failed: %v", err)
	}
	defer rows.Close()

	fmt.Println("SERVICE_RECORDS columns:")
	serviceRecordsHasMaintenanceDate := false
	for rows.Next() {
		var colName string
		rows.Scan(&colName)
		fmt.Printf("  - %s\n", colName)
		if colName == "maintenance_date" {
			serviceRecordsHasMaintenanceDate = true
		}
	}

	// Check maintenance_records  
	rows2, err := db.Query(`SELECT column_name FROM information_schema.columns WHERE table_name = 'maintenance_records' ORDER BY ordinal_position`)
	if err \!= nil {
		log.Fatalf("Query failed: %v", err)
	}
	defer rows2.Close()

	fmt.Println("MAINTENANCE_RECORDS columns:")
	maintenanceRecordsHasServiceDate := false
	for rows2.Next() {
		var colName string
		rows2.Scan(&colName)
		fmt.Printf("  - %s\n", colName)
		if colName == "service_date" {
			maintenanceRecordsHasServiceDate = true
		}
	}

	// Summary
	fmt.Printf("\nservice_records has maintenance_date: %v\n", serviceRecordsHasMaintenanceDate)
	fmt.Printf("maintenance_records has service_date: %v\n", maintenanceRecordsHasServiceDate)

	// Check data counts
	var serviceCount, maintenanceCount int
	db.QueryRow("SELECT COUNT(*) FROM service_records").Scan(&serviceCount)
	db.QueryRow("SELECT COUNT(*) FROM maintenance_records").Scan(&maintenanceCount)
	fmt.Printf("service_records: %d records\n", serviceCount)  
	fmt.Printf("maintenance_records: %d records\n", maintenanceCount)
}
