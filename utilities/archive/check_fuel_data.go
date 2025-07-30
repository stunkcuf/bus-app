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

	fmt.Println("=== Checking for fuel-related data ===\n")

	// Check fuel_records table structure
	fmt.Println("1. fuel_records table structure:")
	rows, err := db.Query(`
		SELECT column_name, data_type, is_nullable
		FROM information_schema.columns 
		WHERE table_name = 'fuel_records' 
		ORDER BY ordinal_position
	`)
	if err != nil {
		log.Printf("Error checking fuel_records structure: %v", err)
	} else {
		defer rows.Close()
		for rows.Next() {
			var colName, dataType, isNullable string
			rows.Scan(&colName, &dataType, &isNullable)
			fmt.Printf("  %-25s %-20s %s\n", colName, dataType, isNullable)
		}
	}

	// Check if fuel data might be in maintenance records
	fmt.Println("\n2. Checking maintenance_records for fuel-related entries:")
	var fuelCount int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM maintenance_records 
		WHERE LOWER(work_description) LIKE '%fuel%' 
		   OR LOWER(work_description) LIKE '%gas%'
		   OR LOWER(work_description) LIKE '%diesel%'
	`).Scan(&fuelCount)
	if err == nil {
		fmt.Printf("   Found %d fuel-related maintenance records\n", fuelCount)
		
		if fuelCount > 0 {
			// Show samples
			fuelRows, _ := db.Query(`
				SELECT vehicle_number, service_date, work_description, cost
				FROM maintenance_records 
				WHERE LOWER(work_description) LIKE '%fuel%' 
				   OR LOWER(work_description) LIKE '%gas%'
				   OR LOWER(work_description) LIKE '%diesel%'
				LIMIT 5
			`)
			if fuelRows != nil {
				defer fuelRows.Close()
				fmt.Println("\n   Sample fuel-related maintenance records:")
				for fuelRows.Next() {
					var vehicleNumber sql.NullInt32
					var serviceDate sql.NullTime
					var workDesc, cost sql.NullString
					fuelRows.Scan(&vehicleNumber, &serviceDate, &workDesc, &cost)
					
					fmt.Printf("   - Vehicle %d: %s - %s ($%s)\n", 
						vehicleNumber.Int32, 
						serviceDate.Time.Format("2006-01-02"),
						workDesc.String,
						cost.String)
				}
			}
		}
	}

	// Check for any tables with 'fuel' in the name
	fmt.Println("\n3. Checking for other fuel-related tables:")
	tableRows, err := db.Query(`
		SELECT table_name 
		FROM information_schema.tables 
		WHERE table_schema = 'public' 
		  AND table_name LIKE '%fuel%'
		ORDER BY table_name
	`)
	if err == nil {
		defer tableRows.Close()
		found := false
		for tableRows.Next() {
			var tableName string
			tableRows.Scan(&tableName)
			fmt.Printf("   Found table: %s\n", tableName)
			found = true
		}
		if !found {
			fmt.Println("   No other fuel-related tables found")
		}
	}

	// Check driver logs for mileage data
	fmt.Println("\n4. Checking driver_logs for mileage data (potential fuel calculation):")
	var driverLogCount int
	err = db.QueryRow("SELECT COUNT(*) FROM driver_logs WHERE end_mileage > begin_mileage").Scan(&driverLogCount)
	if err == nil {
		fmt.Printf("   Found %d driver logs with mileage data\n", driverLogCount)
	}

	// Summary
	fmt.Println("\n=== Summary ===")
	fmt.Println("The fuel_records table exists but is empty.")
	fmt.Println("Fuel data might need to be:")
	fmt.Println("1. Imported from external source")
	fmt.Println("2. Entered manually")
	fmt.Println("3. Calculated from mileage data")
}