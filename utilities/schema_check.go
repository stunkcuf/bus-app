package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
)

func main() {
	// Try DATABASE_URL first, then fallback to hardcoded for testing
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgresql://postgres:jTKOWGEzlprRGbkPBCgGxsgnwyLeGoDL@shortline.proxy.rlwy.net:40148/railway"
		fmt.Println("Using hardcoded database URL for testing")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer db.Close()

	fmt.Println("🔍 Checking database schema...")

	// Check service_records
	rows, err := db.Query(`SELECT column_name FROM information_schema.columns WHERE table_name = 'service_records' ORDER BY ordinal_position`)
	if err != nil {
		log.Printf("Query failed: %v", err)
		return
	}
	defer rows.Close()

	fmt.Println("\nSERVICE_RECORDS columns:")
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
	if err != nil {
		log.Printf("Query failed: %v", err)
		return
	}
	defer rows2.Close()

	fmt.Println("\nMAINTENANCE_RECORDS columns:")
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
	fmt.Printf("\n✅ Analysis:")
	fmt.Printf("\n  - service_records has maintenance_date: %v", serviceRecordsHasMaintenanceDate)
	fmt.Printf("\n  - maintenance_records has service_date: %v", maintenanceRecordsHasServiceDate)

	// Check data counts
	var serviceCount, maintenanceCount int
	db.QueryRow("SELECT COUNT(*) FROM service_records").Scan(&serviceCount)
	db.QueryRow("SELECT COUNT(*) FROM maintenance_records").Scan(&maintenanceCount)
	fmt.Printf("\n  - service_records: %d records", serviceCount)  
	fmt.Printf("\n  - maintenance_records: %d records\n", maintenanceCount)
	
	if !serviceRecordsHasMaintenanceDate {
		fmt.Println("❌ Missing maintenance_date in service_records!")
	}
	if !maintenanceRecordsHasServiceDate {
		fmt.Println("❌ Missing service_date in maintenance_records!")
	}
}