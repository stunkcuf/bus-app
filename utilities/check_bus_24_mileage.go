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
		dbURL = "postgresql://postgres:jTKOWGEzlprRGbkPBCgGxsgnwyLeGoDL@shortline.proxy.rlwy.net:40148/railway"
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	fmt.Println("=== CHECKING BUS 24 DATA ===\n")

	// Check bus table
	fmt.Println("1. Bus Table Data:")
	var busID, status string
	var currentMileage sql.NullInt64
	err = db.QueryRow(`
		SELECT bus_id, current_mileage, status 
		FROM buses 
		WHERE bus_id = '24'
	`).Scan(&busID, &currentMileage, &status)
	
	if err != nil {
		fmt.Printf("Error fetching bus: %v\n", err)
	} else {
		fmt.Printf("   Bus ID: %s\n", busID)
		fmt.Printf("   Current Mileage: %v (Valid: %v)\n", currentMileage.Int64, currentMileage.Valid)
		fmt.Printf("   Status: %s\n\n", status)
	}

	// Check maintenance records
	fmt.Println("2. Maintenance Records (last 5):")
	rows, err := db.Query(`
		SELECT 
			date,
			service_type,
			mileage,
			description
		FROM maintenance_records 
		WHERE vehicle_id = '24'
		ORDER BY date DESC
		LIMIT 5
	`)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		defer rows.Close()
		count := 0
		for rows.Next() {
			var date, serviceType, description string
			var maintenanceMileage sql.NullInt64
			rows.Scan(&date, &serviceType, &maintenanceMileage, &description)
			count++
			fmt.Printf("   [%d] Date: %s, Type: %s\n", count, date, serviceType)
			fmt.Printf("       Mileage: %v (Valid: %v)\n", maintenanceMileage.Int64, maintenanceMileage.Valid)
			fmt.Printf("       Description: %s\n\n", description)
		}
		if count == 0 {
			fmt.Println("   No maintenance records found")
		}
	}

	// Check monthly mileage reports
	fmt.Println("3. Monthly Mileage Reports (last 3):")
	rows2, err := db.Query(`
		SELECT 
			year,
			month,
			starting_mileage,
			ending_mileage,
			total_miles
		FROM monthly_mileage_reports 
		WHERE bus_id = '24'
		ORDER BY year DESC, month DESC
		LIMIT 3
	`)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		defer rows2.Close()
		count := 0
		for rows2.Next() {
			var year, month int
			var startMileage, endMileage, totalMiles sql.NullInt64
			rows2.Scan(&year, &month, &startMileage, &endMileage, &totalMiles)
			count++
			fmt.Printf("   [%d] %d/%d:\n", count, month, year)
			fmt.Printf("       Start: %v, End: %v, Total: %v\n\n", 
				startMileage.Int64, endMileage.Int64, totalMiles.Int64)
		}
		if count == 0 {
			fmt.Println("   No mileage reports found")
		}
	}

	// Check if we need to sync mileage
	fmt.Println("4. Suggested Fix:")
	
	// Get max mileage from maintenance records
	var maxMileage sql.NullInt64
	err = db.QueryRow(`
		SELECT MAX(mileage) 
		FROM maintenance_records 
		WHERE vehicle_id = '24' AND mileage IS NOT NULL
	`).Scan(&maxMileage)
	
	if err == nil && maxMileage.Valid {
		fmt.Printf("   Max mileage from maintenance: %d\n", maxMileage.Int64)
		
		if !currentMileage.Valid || currentMileage.Int64 < maxMileage.Int64 {
			fmt.Printf("   Bus table mileage should be updated to: %d\n", maxMileage.Int64)
			fmt.Println("\n   Updating bus mileage...")
			
			_, err = db.Exec(`
				UPDATE buses 
				SET current_mileage = $1 
				WHERE bus_id = '24'
			`, maxMileage.Int64)
			
			if err != nil {
				fmt.Printf("   Error updating: %v\n", err)
			} else {
				fmt.Println("   ✅ Bus mileage updated successfully!")
			}
		} else {
			fmt.Println("   Bus mileage appears correct")
		}
	}
}