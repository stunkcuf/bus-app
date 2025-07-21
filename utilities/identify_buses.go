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

	fmt.Println("=== IDENTIFYING BUSES IN FLEET_VEHICLES ===\n")

	// Look for vehicles that might be buses based on various criteria
	fmt.Println("Vehicles with numeric license plates (potential buses):")
	rows, _ := db.Query(`
		SELECT vehicle_number, make, model, license, description, sheet_name
		FROM fleet_vehicles 
		WHERE license ~ '^\d+$'
		ORDER BY CAST(license AS INTEGER)
		LIMIT 20
	`)
	defer rows.Close()
	
	busCount := 0
	for rows.Next() {
		var vNum sql.NullInt32
		var make, model, license, desc, sheet sql.NullString
		rows.Scan(&vNum, &make, &model, &license, &desc, &sheet)
		fmt.Printf("  Vehicle #%d: License=%s, Model=%s, Desc=%s, Sheet=%s\n", 
			vNum.Int32, getValue(license), getValue(model), getValue(desc), getValue(sheet))
		busCount++
	}
	
	if busCount == 0 {
		fmt.Println("  (none found)")
	}

	// Check the old buses table to see what pattern to look for
	fmt.Println("\nChecking original buses table for patterns:")
	busRows, err := db.Query(`
		SELECT bus_id, model, status 
		FROM buses 
		LIMIT 10
	`)
	if err == nil {
		defer busRows.Close()
		for busRows.Next() {
			var busID, model, status sql.NullString
			busRows.Scan(&busID, &model, &status)
			fmt.Printf("  Bus: %s, Model: %s, Status: %s\n", 
				getValue(busID), getValue(model), getValue(status))
		}
	} else {
		fmt.Printf("  Error reading buses table: %v\n", err)
	}

	// Look for vehicles with low numbers (1-100) which are typically buses
	fmt.Println("\nVehicles with numbers 1-100:")
	rows2, _ := db.Query(`
		SELECT vehicle_number, make, model, license, description, vehicle_type
		FROM fleet_vehicles 
		WHERE vehicle_number BETWEEN 1 AND 100
		ORDER BY vehicle_number
		LIMIT 20
	`)
	defer rows2.Close()
	
	for rows2.Next() {
		var vNum sql.NullInt32
		var make, model, license, desc, vType sql.NullString
		rows2.Scan(&vNum, &make, &model, &license, &desc, &vType)
		fmt.Printf("  #%d: %s %s, License=%s, Type=%s\n", 
			vNum.Int32, getValue(make), getValue(model), 
			getValue(license), getValue(vType))
	}

	// Update buses based on the buses table
	fmt.Println("\nUpdating fleet_vehicles based on buses table...")
	result, err := db.Exec(`
		UPDATE fleet_vehicles fv
		SET vehicle_type = 'bus'
		FROM buses b
		WHERE (fv.license = b.bus_id OR fv.sheet_name = b.bus_id)
		AND (fv.vehicle_type IS NULL OR fv.vehicle_type = 'other')
	`)
	if err == nil {
		affected, _ := result.RowsAffected()
		fmt.Printf("  Updated %d vehicles to 'bus' based on buses table match\n", affected)
	} else {
		fmt.Printf("  Error: %v\n", err)
	}

	// Also update based on vehicle number matching
	result, err = db.Exec(`
		UPDATE fleet_vehicles fv
		SET vehicle_type = 'bus'
		FROM buses b
		WHERE fv.vehicle_number::text = REGEXP_REPLACE(b.bus_id, '[^0-9]', '', 'g')
		AND (fv.vehicle_type IS NULL OR fv.vehicle_type = 'other')
		AND b.bus_id IS NOT NULL
	`)
	if err == nil {
		affected, _ := result.RowsAffected()
		fmt.Printf("  Updated %d more vehicles to 'bus' based on number extraction\n", affected)
	}

	// Show final distribution
	fmt.Println("\nFinal vehicle type distribution:")
	rows3, _ := db.Query(`
		SELECT vehicle_type, COUNT(*) 
		FROM fleet_vehicles 
		GROUP BY vehicle_type
		ORDER BY COUNT(*) DESC
	`)
	defer rows3.Close()
	
	total := 0
	for rows3.Next() {
		var vType sql.NullString
		var count int
		rows3.Scan(&vType, &count)
		fmt.Printf("  %s: %d\n", getValue(vType), count)
		total += count
	}
	fmt.Printf("  TOTAL: %d\n", total)
}

func getValue(ns sql.NullString) string {
	if ns.Valid && ns.String != "" {
		return ns.String
	}
	return "(null)"
}