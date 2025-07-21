package main

import (
	"database/sql"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	_ "github.com/lib/pq"
)

func main() {
	dbURL := "postgresql://postgres:jTKOWGEzlprRGbkPBCgGxsgnwyLeGoDL@shortline.proxy.rlwy.net:40148/railway"

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer db.Close()

	fmt.Println("=== VEHICLE TABLE CONSOLIDATION ===")
	fmt.Println("This will consolidate buses, vehicles into fleet_vehicles")
	fmt.Println()

	// Step 1: Add vehicle_type column if it doesn't exist
	fmt.Println("Step 1: Adding vehicle_type column to fleet_vehicles...")
	_, err = db.Exec(`
		ALTER TABLE fleet_vehicles 
		ADD COLUMN IF NOT EXISTS vehicle_type VARCHAR(50)
	`)
	if err != nil {
		log.Printf("Warning: Could not add vehicle_type column: %v", err)
	} else {
		fmt.Println("✓ vehicle_type column ready")
	}

	// Step 2: Analyze current data
	fmt.Println("\nStep 2: Analyzing current data...")
	analyzeCurrentData(db)

	// Step 3: Import buses into fleet_vehicles
	fmt.Println("\nStep 3: Importing buses into fleet_vehicles...")
	busCount := importBuses(db)
	fmt.Printf("✓ Imported %d buses\n", busCount)

	// Step 4: Import vehicles into fleet_vehicles
	fmt.Println("\nStep 4: Importing vehicles into fleet_vehicles...")
	vehicleCount := importVehicles(db)
	fmt.Printf("✓ Imported %d vehicles\n", vehicleCount)

	// Step 5: Update existing fleet_vehicles with vehicle_type
	fmt.Println("\nStep 5: Updating vehicle types for existing records...")
	updateExistingTypes(db)

	// Step 6: Show final statistics
	fmt.Println("\nStep 6: Final statistics...")
	showFinalStats(db)

	fmt.Println("\n=== CONSOLIDATION COMPLETE ===")
	fmt.Println("\nNext steps:")
	fmt.Println("1. Verify data in fleet_vehicles table")
	fmt.Println("2. Update application to use fleet_vehicles only")
	fmt.Println("3. Drop old tables: buses, vehicles (after verification)")
}

func analyzeCurrentData(db *sql.DB) {
	// Count records in each table
	tables := map[string]string{
		"buses":          "SELECT COUNT(*) FROM buses",
		"vehicles":       "SELECT COUNT(*) FROM vehicles",
		"fleet_vehicles": "SELECT COUNT(*) FROM fleet_vehicles",
	}

	for table, query := range tables {
		var count int
		err := db.QueryRow(query).Scan(&count)
		if err != nil {
			fmt.Printf("  %s: Error - %v\n", table, err)
		} else {
			fmt.Printf("  %s: %d records\n", table, count)
		}
	}
}

func importBuses(db *sql.DB) int {
	// First, check what buses we have
	rows, err := db.Query(`
		SELECT bus_id, status, model, capacity, oil_status, 
		       tire_status, maintenance_notes
		FROM buses
	`)
	if err != nil {
		log.Printf("Error reading buses: %v", err)
		return 0
	}
	defer rows.Close()

	imported := 0
	for rows.Next() {
		var busID, status, model, oilStatus, tireStatus, maintNotes sql.NullString
		var capacity sql.NullInt32

		err := rows.Scan(&busID, &status, &model, &capacity, 
			&oilStatus, &tireStatus, &maintNotes)
		if err != nil {
			log.Printf("Error scanning bus: %v", err)
			continue
		}

		if !busID.Valid || busID.String == "" {
			continue
		}

		// Extract vehicle number from bus_id (e.g., "Bus-2" -> 2)
		vehicleNum := extractNumber(busID.String)
		if vehicleNum == 0 {
			// Try to use the bus_id as-is
			vehicleNum = hashStringToNumber(busID.String)
		}

		// Check if already exists
		var exists bool
		err = db.QueryRow(`
			SELECT EXISTS(
				SELECT 1 FROM fleet_vehicles 
				WHERE vehicle_number = $1 OR license = $2
			)`, vehicleNum, busID.String).Scan(&exists)
		
		if exists {
			fmt.Printf("  Skipping bus %s (already exists)\n", busID.String)
			continue
		}

		// Extract make from model if possible
		make := "Unknown"
		if model.Valid {
			make = extractMake(model.String)
		}

		// Insert into fleet_vehicles
		_, err = db.Exec(`
			INSERT INTO fleet_vehicles (
				vehicle_number, sheet_name, make, model, 
				description, license, vehicle_type, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())`,
			vehicleNum,
			busID.String,
			make,
			model,
			fmt.Sprintf("Bus - %s", getStringValue(model)),
			busID.String,
			"bus",
		)
		if err != nil {
			log.Printf("Error inserting bus %s: %v", busID.String, err)
			continue
		}

		imported++
		fmt.Printf("  Imported bus: %s -> vehicle #%d\n", busID.String, vehicleNum)
	}

	return imported
}

func importVehicles(db *sql.DB) int {
	rows, err := db.Query(`
		SELECT vehicle_id, model, description, year, tire_size, 
		       license, oil_status, tire_status, status, 
		       maintenance_notes, serial_number, base
		FROM vehicles
	`)
	if err != nil {
		log.Printf("Error reading vehicles: %v", err)
		return 0
	}
	defer rows.Close()

	imported := 0
	for rows.Next() {
		var vehicleID, model, description, year, tireSize, license sql.NullString
		var oilStatus, tireStatus, status, maintNotes, serial, base sql.NullString

		err := rows.Scan(&vehicleID, &model, &description, &year, 
			&tireSize, &license, &oilStatus, &tireStatus, &status,
			&maintNotes, &serial, &base)
		if err != nil {
			log.Printf("Error scanning vehicle: %v", err)
			continue
		}

		if !vehicleID.Valid || vehicleID.String == "" {
			continue
		}

		// Extract vehicle number
		vehicleNum := extractNumber(vehicleID.String)
		if vehicleNum == 0 {
			vehicleNum = hashStringToNumber(vehicleID.String)
		}

		// Check if already exists
		var exists bool
		licenseCheck := license.String
		if !license.Valid || license.String == "" {
			licenseCheck = vehicleID.String
		}
		
		err = db.QueryRow(`
			SELECT EXISTS(
				SELECT 1 FROM fleet_vehicles 
				WHERE vehicle_number = $1 OR license = $2
			)`, vehicleNum, licenseCheck).Scan(&exists)
		
		if exists {
			fmt.Printf("  Skipping vehicle %s (already exists)\n", vehicleID.String)
			continue
		}

		// Determine vehicle type
		vehicleType := determineVehicleType(model.String, description.String)
		
		// Extract make
		make := "Unknown"
		if model.Valid {
			make = extractMake(model.String)
		}

		// Convert year string to int
		var yearInt sql.NullInt32
		if year.Valid && year.String != "" {
			if y, err := strconv.Atoi(year.String); err == nil {
				yearInt = sql.NullInt32{Int32: int32(y), Valid: true}
			}
		}

		// Insert into fleet_vehicles
		_, err = db.Exec(`
			INSERT INTO fleet_vehicles (
				vehicle_number, sheet_name, year, make, model, 
				description, serial_number, license, location,
				tire_size, vehicle_type, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW(), NOW())`,
			vehicleNum,
			vehicleID.String,
			yearInt,
			make,
			model,
			description,
			serial,
			licenseCheck,
			base,
			tireSize,
			vehicleType,
		)
		if err != nil {
			log.Printf("Error inserting vehicle %s: %v", vehicleID.String, err)
			continue
		}

		imported++
		fmt.Printf("  Imported vehicle: %s -> vehicle #%d (%s)\n", 
			vehicleID.String, vehicleNum, vehicleType)
	}

	return imported
}

func updateExistingTypes(db *sql.DB) {
	// Update vehicles without a type based on their description/model
	result, err := db.Exec(`
		UPDATE fleet_vehicles 
		SET vehicle_type = CASE
			WHEN LOWER(description) LIKE '%bus%' OR LOWER(model) LIKE '%bus%' THEN 'bus'
			WHEN LOWER(description) LIKE '%van%' OR LOWER(model) LIKE '%van%' THEN 'van'
			WHEN LOWER(description) LIKE '%truck%' OR LOWER(model) LIKE '%truck%' THEN 'truck'
			WHEN LOWER(description) LIKE '%car%' OR LOWER(model) LIKE '%impala%' OR 
			     LOWER(model) LIKE '%sedan%' THEN 'car'
			WHEN LOWER(description) LIKE '%suv%' OR LOWER(model) LIKE '%suburban%' OR
			     LOWER(model) LIKE '%tahoe%' THEN 'suv'
			ELSE 'other'
		END
		WHERE vehicle_type IS NULL
	`)
	if err != nil {
		log.Printf("Error updating vehicle types: %v", err)
		return
	}

	affected, _ := result.RowsAffected()
	fmt.Printf("✓ Updated %d records with vehicle type\n", affected)
}

func showFinalStats(db *sql.DB) {
	// Count by vehicle type
	rows, err := db.Query(`
		SELECT vehicle_type, COUNT(*) as count
		FROM fleet_vehicles
		GROUP BY vehicle_type
		ORDER BY count DESC
	`)
	if err != nil {
		log.Printf("Error getting stats: %v", err)
		return
	}
	defer rows.Close()

	fmt.Println("\nFleet vehicles by type:")
	total := 0
	for rows.Next() {
		var vType sql.NullString
		var count int
		rows.Scan(&vType, &count)
		if vType.Valid {
			fmt.Printf("  %s: %d\n", vType.String, count)
		} else {
			fmt.Printf("  (untyped): %d\n", count)
		}
		total += count
	}
	fmt.Printf("  TOTAL: %d vehicles\n", total)
}

// Helper functions
func extractNumber(s string) int {
	re := regexp.MustCompile(`\d+`)
	matches := re.FindString(s)
	if matches != "" {
		if num, err := strconv.Atoi(matches); err == nil {
			return num
		}
	}
	return 0
}

func hashStringToNumber(s string) int {
	hash := 0
	for _, c := range s {
		hash = hash*31 + int(c)
	}
	// Keep it positive and reasonable
	if hash < 0 {
		hash = -hash
	}
	return (hash % 9000) + 1000 // Returns number between 1000-9999
}

func extractMake(model string) string {
	model = strings.ToUpper(model)
	makes := []string{
		"FORD", "CHEVROLET", "CHEVY", "GMC", "DODGE", 
		"TOYOTA", "HONDA", "NISSAN", "FREIGHTLINER",
		"INTERNATIONAL", "BLUEBIRD", "THOMAS",
	}
	
	for _, make := range makes {
		if strings.Contains(model, make) {
			if make == "CHEVY" {
				return "Chevrolet"
			}
			return strings.Title(strings.ToLower(make))
		}
	}
	return "Unknown"
}

func determineVehicleType(model, description string) string {
	combined := strings.ToLower(model + " " + description)
	
	if strings.Contains(combined, "bus") {
		return "bus"
	} else if strings.Contains(combined, "van") {
		return "van"
	} else if strings.Contains(combined, "truck") {
		return "truck"
	} else if strings.Contains(combined, "impala") || 
	          strings.Contains(combined, "sedan") ||
	          strings.Contains(combined, "car") {
		return "car"
	} else if strings.Contains(combined, "suburban") || 
	          strings.Contains(combined, "tahoe") ||
	          strings.Contains(combined, "suv") {
		return "suv"
	}
	return "other"
}

func getStringValue(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}