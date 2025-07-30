package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"

	_ "github.com/lib/pq"
	"github.com/joho/godotenv"
)

func main() {
	fmt.Println("🔍 CHECKING SERVICE RECORDS DATA")
	fmt.Println("=" + strings.Repeat("=", 60))

	// Load environment
	godotenv.Load("../.env")

	// Connect to database
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL not set")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Check if service_records table exists
	fmt.Println("\n📋 Checking service_records table...")
	var tableExists bool
	err = db.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables 
			WHERE table_name = 'service_records'
		)
	`).Scan(&tableExists)
	
	if err != nil {
		log.Fatal("Failed to check table existence:", err)
	}
	
	if !tableExists {
		fmt.Println("❌ service_records table does not exist!")
		return
	}
	
	fmt.Println("✅ service_records table exists")

	// Get column information
	fmt.Println("\n📊 Table Structure:")
	rows, err := db.Query(`
		SELECT column_name, data_type, is_nullable
		FROM information_schema.columns 
		WHERE table_name = 'service_records'
		ORDER BY ordinal_position
	`)
	if err != nil {
		log.Fatal("Failed to get columns:", err)
	}
	defer rows.Close()

	for rows.Next() {
		var colName, dataType, isNullable string
		rows.Scan(&colName, &dataType, &isNullable)
		fmt.Printf("  • %-20s %s %s\n", colName, dataType, isNullable)
	}

	// Count records
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM service_records").Scan(&count)
	if err != nil {
		log.Fatal("Failed to count records:", err)
	}
	
	fmt.Printf("\n📈 Total Records: %d\n", count)

	// Sample some records
	if count > 0 {
		fmt.Println("\n📝 Sample Records (first 5):")
		rows, err := db.Query(`
			SELECT id, COALESCE(vehicle_info, ''), 
			       COALESCE(field_1, ''), COALESCE(field_2, ''), 
			       COALESCE(field_3, ''), COALESCE(field_4, '')
			FROM service_records 
			LIMIT 5
		`)
		if err != nil {
			log.Fatal("Failed to query records:", err)
		}
		defer rows.Close()

		for rows.Next() {
			var id int
			var vehicleInfo, f1, f2, f3, f4 string
			rows.Scan(&id, &vehicleInfo, &f1, &f2, &f3, &f4)
			
			fmt.Printf("\n  Record #%d:\n", id)
			if vehicleInfo != "" {
				fmt.Printf("    Vehicle: %s\n", vehicleInfo)
			}
			
			fields := []string{f1, f2, f3, f4}
			hasData := false
			for i, field := range fields {
				if field != "" {
					fmt.Printf("    Field %d: %s\n", i+1, field)
					hasData = true
				}
			}
			
			if !hasData && vehicleInfo == "" {
				fmt.Println("    (No data)")
			}
		}
	}

	// Check for records with meaningful data
	var recordsWithData int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM service_records 
		WHERE vehicle_info IS NOT NULL AND vehicle_info != ''
		   OR field_1 IS NOT NULL AND field_1 != ''
		   OR field_2 IS NOT NULL AND field_2 != ''
		   OR field_3 IS NOT NULL AND field_3 != ''
		   OR field_4 IS NOT NULL AND field_4 != ''
	`).Scan(&recordsWithData)
	
	if err == nil {
		fmt.Printf("\n✅ Records with data: %d (%.1f%%)\n", recordsWithData, float64(recordsWithData)/float64(count)*100)
	}

	// Check for unique vehicles
	var uniqueVehicles int
	err = db.QueryRow(`
		SELECT COUNT(DISTINCT vehicle_info) 
		FROM service_records 
		WHERE vehicle_info IS NOT NULL AND vehicle_info != ''
	`).Scan(&uniqueVehicles)
	
	if err == nil {
		fmt.Printf("🚗 Unique vehicles: %d\n", uniqueVehicles)
	}

	// Check for maintenance dates
	var recordsWithDates int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM service_records 
		WHERE maintenance_date IS NOT NULL
	`).Scan(&recordsWithDates)
	
	if err == nil {
		fmt.Printf("📅 Records with maintenance dates: %d\n", recordsWithDates)
	}
}