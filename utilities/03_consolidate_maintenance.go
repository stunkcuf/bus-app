package main

import (
	"database/sql"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"time"
	_ "github.com/lib/pq"
)

func main() {
	dbURL := "postgresql://postgres:jTKOWGEzlprRGbkPBCgGxsgnwyLeGoDL@shortline.proxy.rlwy.net:40148/railway"

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer db.Close()

	fmt.Println("=== MAINTENANCE TABLE CONSOLIDATION ===")
	fmt.Println("This will consolidate service_records into maintenance_records")
	fmt.Println()

	// Step 1: Analyze service_records structure
	fmt.Println("Step 1: Analyzing service_records data...")
	analyzeServiceRecords(db)

	// Step 2: Import service_records into maintenance_records
	fmt.Println("\nStep 2: Importing service_records into maintenance_records...")
	serviceCount := importServiceRecords(db)
	fmt.Printf("✓ Imported %d service records\n", serviceCount)

	// Step 3: Import maintenance_sheets if useful
	fmt.Println("\nStep 3: Checking maintenance_sheets...")
	sheetCount := importMaintenanceSheets(db)
	fmt.Printf("✓ Imported %d maintenance sheet records\n", sheetCount)

	// Step 4: Show final statistics
	fmt.Println("\nStep 4: Final statistics...")
	showMaintenanceStats(db)

	fmt.Println("\n=== CONSOLIDATION COMPLETE ===")
	fmt.Println("\nNext steps:")
	fmt.Println("1. Verify data in maintenance_records table")
	fmt.Println("2. Update application to use maintenance_records only")
	fmt.Println("3. Drop old tables: service_records, maintenance_sheets (after verification)")
}

func analyzeServiceRecords(db *sql.DB) {
	// Show sample data to understand the unnamed columns
	rows, err := db.Query(`
		SELECT unnamed_0, unnamed_1, unnamed_2, unnamed_3, unnamed_4,
		       unnamed_5, unnamed_6, unnamed_7, unnamed_8, unnamed_9
		FROM service_records
		WHERE unnamed_1 IS NOT NULL
		LIMIT 10
	`)
	if err != nil {
		log.Printf("Error reading service_records: %v", err)
		return
	}
	defer rows.Close()

	fmt.Println("Sample service_records data:")
	fmt.Println("Column mapping discovered:")
	fmt.Println("  unnamed_0: Vehicle Description (e.g., '2012 CHEVY IMPALA')")
	fmt.Println("  unnamed_1: Vehicle Number")
	fmt.Println("  unnamed_2: Location/Base")
	fmt.Println("  unnamed_3: Current/Service Mileage")
	fmt.Println("  unnamed_4: Last Service Mileage")
	fmt.Println("  unnamed_5: Next Service Mileage")
	fmt.Println("  unnamed_7: Miles to Next Service")
	fmt.Println("  unnamed_8: Service Interval")

	// Count total records
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM service_records WHERE unnamed_1 ~ '^[0-9]+$'").Scan(&count)
	if err == nil {
		fmt.Printf("\nFound %d valid service records to import\n", count)
	}
}

func importServiceRecords(db *sql.DB) int {
	rows, err := db.Query(`
		SELECT unnamed_0, unnamed_1, unnamed_2, unnamed_3, unnamed_4,
		       unnamed_5, unnamed_6, unnamed_7, unnamed_8, unnamed_9,
		       unnamed_10, created_at, maintenance_date
		FROM service_records
		WHERE unnamed_1 ~ '^[0-9]+$'  -- Only records with numeric vehicle numbers
	`)
	if err != nil {
		log.Printf("Error reading service_records: %v", err)
		return 0
	}
	defer rows.Close()

	imported := 0
	for rows.Next() {
		var vehDesc, vehNum, location, currMileage, lastMileage sql.NullString
		var nextMileage, field6, milesToService, serviceInterval, field9, field10 sql.NullString
		var createdAt time.Time
		var maintDate sql.NullTime

		err := rows.Scan(&vehDesc, &vehNum, &location, &currMileage, &lastMileage,
			&nextMileage, &field6, &milesToService, &serviceInterval, &field9,
			&field10, &createdAt, &maintDate)
		if err != nil {
			log.Printf("Error scanning service record: %v", err)
			continue
		}

		// Parse vehicle number
		vehicleNumber, err := strconv.Atoi(vehNum.String)
		if err != nil {
			continue
		}

		// Parse mileages
		var currentMileageInt sql.NullInt32
		if currMileage.Valid && currMileage.String != "" {
			if m, err := strconv.Atoi(currMileage.String); err == nil {
				currentMileageInt = sql.NullInt32{Int32: int32(m), Valid: true}
			}
		}

		// Determine service date
		serviceDate := maintDate
		if !serviceDate.Valid {
			serviceDate = sql.NullTime{Time: createdAt, Valid: true}
		}

		// Build work description
		workDesc := "Service Record Import"
		if vehDesc.Valid && vehDesc.String != "" {
			workDesc = fmt.Sprintf("Vehicle: %s", vehDesc.String)
		}
		if nextMileage.Valid && nextMileage.String != "" {
			workDesc += fmt.Sprintf(" - Next service at %s miles", nextMileage.String)
		}
		if milesToService.Valid && milesToService.String != "" {
			workDesc += fmt.Sprintf(" (%s miles until service)", milesToService.String)
		}

		// Check if this record might already exist
		var exists bool
		err = db.QueryRow(`
			SELECT EXISTS(
				SELECT 1 FROM maintenance_records 
				WHERE vehicle_number = $1 
				AND ABS(COALESCE(mileage, 0) - COALESCE($2, 0)) < 100
				AND service_date::date = $3::date
			)`, vehicleNumber, currentMileageInt, serviceDate).Scan(&exists)
		
		if exists {
			fmt.Printf("  Skipping duplicate service record for vehicle #%d\n", vehicleNumber)
			continue
		}

		// Insert into maintenance_records
		_, err = db.Exec(`
			INSERT INTO maintenance_records (
				vehicle_number, vehicle_id, service_date, mileage,
				work_description, raw_data, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())`,
			vehicleNumber,
			fmt.Sprintf("%d", vehicleNumber),
			serviceDate,
			currentMileageInt,
			workDesc,
			fmt.Sprintf("Last: %s, Next: %s, Interval: %s", 
				getStringValue(lastMileage), 
				getStringValue(nextMileage), 
				getStringValue(serviceInterval)),
			createdAt,
		)
		if err != nil {
			log.Printf("Error inserting service record for vehicle %d: %v", vehicleNumber, err)
			continue
		}

		imported++
		if imported%10 == 0 {
			fmt.Printf("  Imported %d records...\n", imported)
		}
	}

	return imported
}

func importMaintenanceSheets(db *sql.DB) int {
	// Check if maintenance_sheets has any useful data
	rows, err := db.Query(`
		SELECT vehicle_id, description, unnamed_1, unnamed_2, 
		       unnamed_3, unnamed_4, created_at
		FROM maintenance_sheets
		WHERE vehicle_id IS NOT NULL 
		  AND (unnamed_1 IS NOT NULL OR unnamed_2 IS NOT NULL 
		       OR unnamed_3 IS NOT NULL OR unnamed_4 IS NOT NULL)
	`)
	if err != nil {
		log.Printf("Error reading maintenance_sheets: %v", err)
		return 0
	}
	defer rows.Close()

	imported := 0
	for rows.Next() {
		var vehicleID, description, field1, field2, field3, field4 sql.NullString
		var createdAt time.Time

		err := rows.Scan(&vehicleID, &description, &field1, &field2, 
			&field3, &field4, &createdAt)
		if err != nil {
			continue
		}

		// Extract vehicle number
		vehicleNumber := extractVehicleNumber(vehicleID.String)
		if vehicleNumber == 0 {
			continue
		}

		// Build work description from available fields
		workDesc := "Maintenance Sheet Import"
		if description.Valid && description.String != "" {
			workDesc = description.String
		}
		
		// Append non-null fields
		details := ""
		if field1.Valid && field1.String != "" {
			details += field1.String + " "
		}
		if field2.Valid && field2.String != "" {
			details += field2.String + " "
		}
		if field3.Valid && field3.String != "" {
			details += field3.String + " "
		}
		if field4.Valid && field4.String != "" {
			details += field4.String
		}
		
		if details != "" {
			workDesc += " - " + details
		}

		// Insert into maintenance_records
		_, err = db.Exec(`
			INSERT INTO maintenance_records (
				vehicle_number, vehicle_id, service_date,
				work_description, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, NOW())`,
			vehicleNumber,
			fmt.Sprintf("%d", vehicleNumber),
			createdAt,
			workDesc,
			createdAt,
		)
		if err != nil {
			log.Printf("Error inserting maintenance sheet: %v", err)
			continue
		}

		imported++
	}

	return imported
}

func showMaintenanceStats(db *sql.DB) {
	// Count total maintenance records
	var total int
	err := db.QueryRow("SELECT COUNT(*) FROM maintenance_records").Scan(&total)
	if err == nil {
		fmt.Printf("\nTotal maintenance records: %d\n", total)
	}

	// Count by vehicle
	fmt.Println("\nTop 10 vehicles by maintenance records:")
	rows, err := db.Query(`
		SELECT vehicle_number, COUNT(*) as count
		FROM maintenance_records
		WHERE vehicle_number IS NOT NULL
		GROUP BY vehicle_number
		ORDER BY count DESC
		LIMIT 10
	`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var vehNum sql.NullInt32
			var count int
			rows.Scan(&vehNum, &count)
			if vehNum.Valid {
				fmt.Printf("  Vehicle #%d: %d records\n", vehNum.Int32, count)
			}
		}
	}

	// Date range
	var minDate, maxDate sql.NullTime
	err = db.QueryRow(`
		SELECT MIN(COALESCE(service_date, date)), MAX(COALESCE(service_date, date))
		FROM maintenance_records
	`).Scan(&minDate, &maxDate)
	if err == nil && minDate.Valid && maxDate.Valid {
		fmt.Printf("\nDate range: %s to %s\n", 
			minDate.Time.Format("2006-01-02"),
			maxDate.Time.Format("2006-01-02"))
	}
}

// Helper functions
func extractVehicleNumber(s string) int {
	if s == "" {
		return 0
	}
	
	// Try to extract number from string
	re := regexp.MustCompile(`\d+`)
	matches := re.FindString(s)
	if matches != "" {
		if num, err := strconv.Atoi(matches); err == nil {
			return num
		}
	}
	return 0
}

func getStringValue(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return "N/A"
}