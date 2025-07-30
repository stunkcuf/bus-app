package main

import (
	"database/sql"
	"fmt"
	"log"
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

	fmt.Println("=== FIXING VEHICLE TYPES ===\n")

	// First, show current vehicle type distribution
	fmt.Println("Current vehicle type distribution:")
	rows, _ := db.Query(`
		SELECT vehicle_type, COUNT(*) 
		FROM fleet_vehicles 
		GROUP BY vehicle_type
	`)
	defer rows.Close()
	
	for rows.Next() {
		var vType sql.NullString
		var count int
		rows.Scan(&vType, &count)
		fmt.Printf("  %s: %d\n", getValue(vType), count)
	}

	// Update buses based on various criteria
	fmt.Println("\nUpdating vehicle types based on patterns...")
	
	// Pattern 1: License plate starts with number (typical for buses)
	result, err := db.Exec(`
		UPDATE fleet_vehicles 
		SET vehicle_type = 'bus'
		WHERE license ~ '^\d+$' 
		AND (vehicle_type IS NULL OR vehicle_type = 'other')
	`)
	if err == nil {
		affected, _ := result.RowsAffected()
		fmt.Printf("  Updated %d vehicles with numeric license plates to 'bus'\n", affected)
	}

	// Pattern 2: Vehicle number in typical bus range (1-100)
	result, err = db.Exec(`
		UPDATE fleet_vehicles 
		SET vehicle_type = 'bus'
		WHERE vehicle_number BETWEEN 1 AND 100
		AND license ~ '^\d+$'
		AND (vehicle_type IS NULL OR vehicle_type = 'other')
	`)
	if err == nil {
		affected, _ := result.RowsAffected()
		fmt.Printf("  Updated %d vehicles in bus number range to 'bus'\n", affected)
	}

	// Pattern 3: Update based on model/description containing vehicle type keywords
	result, err = db.Exec(`
		UPDATE fleet_vehicles 
		SET vehicle_type = CASE
			WHEN LOWER(model) LIKE '%van%' OR LOWER(description) LIKE '%van%' THEN 'van'
			WHEN LOWER(model) LIKE '%impala%' OR LOWER(model) LIKE '%car%' THEN 'car'
			WHEN LOWER(model) LIKE '%truck%' OR LOWER(model) LIKE '%pickup%' THEN 'truck'
			WHEN LOWER(model) LIKE '%suburban%' OR LOWER(model) LIKE '%tahoe%' THEN 'suv'
			ELSE vehicle_type
		END
		WHERE vehicle_type = 'other'
	`)
	if err == nil {
		affected, _ := result.RowsAffected()
		fmt.Printf("  Updated %d vehicles based on model/description keywords\n", affected)
	}

	// Show updated distribution
	fmt.Println("\nUpdated vehicle type distribution:")
	rows2, _ := db.Query(`
		SELECT vehicle_type, COUNT(*) 
		FROM fleet_vehicles 
		GROUP BY vehicle_type
		ORDER BY COUNT(*) DESC
	`)
	defer rows2.Close()
	
	for rows2.Next() {
		var vType sql.NullString
		var count int
		rows2.Scan(&vType, &count)
		fmt.Printf("  %s: %d\n", getValue(vType), count)
	}

	// Show some examples
	fmt.Println("\nSample vehicles by type:")
	types := []string{"bus", "car", "van", "truck", "suv", "other"}
	
	for _, vType := range types {
		fmt.Printf("\n%s vehicles:\n", strings.Title(vType))
		
		sampleRows, _ := db.Query(`
			SELECT vehicle_number, make, model, license 
			FROM fleet_vehicles 
			WHERE vehicle_type = $1
			LIMIT 3
		`, vType)
		
		hasRows := false
		for sampleRows.Next() {
			var vNum sql.NullInt32
			var make, model, license sql.NullString
			sampleRows.Scan(&vNum, &make, &model, &license)
			fmt.Printf("  #%d: %s %s (License: %s)\n", 
				vNum.Int32, getValue(make), getValue(model), getValue(license))
			hasRows = true
		}
		sampleRows.Close()
		
		if !hasRows {
			fmt.Printf("  (none)\n")
		}
	}

	// Clean up the 4 tables that should have been dropped
	fmt.Println("\n=== CLEANING UP REMAINING TABLES ===")
	tablesToDrop := []string{
		"bus_maintenance_logs",
		"vehicle_maintenance_logs", 
		"mileage_reports",
		"mileage_records",
	}
	
	for _, table := range tablesToDrop {
		_, err := db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", table))
		if err != nil {
			fmt.Printf("  ✗ Failed to drop %s: %v\n", table, err)
		} else {
			fmt.Printf("  ✓ Dropped %s\n", table)
		}
	}

	fmt.Println("\n✅ Vehicle type fixing complete!")
}

func getValue(ns sql.NullString) string {
	if ns.Valid && ns.String != "" {
		return ns.String
	}
	return "(null)"
}