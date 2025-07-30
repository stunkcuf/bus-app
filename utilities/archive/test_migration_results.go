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

	fmt.Println("=== POST-MIGRATION DATA VERIFICATION ===\n")

	// Test 1: Fleet vehicles
	fmt.Println("1. Fleet Vehicles Test:")
	var fleetCount int
	var busCount int
	var otherCount int
	
	db.QueryRow("SELECT COUNT(*) FROM fleet_vehicles").Scan(&fleetCount)
	db.QueryRow("SELECT COUNT(*) FROM fleet_vehicles WHERE vehicle_type = 'bus'").Scan(&busCount)
	db.QueryRow("SELECT COUNT(*) FROM fleet_vehicles WHERE vehicle_type != 'bus'").Scan(&otherCount)
	
	fmt.Printf("   Total vehicles: %d\n", fleetCount)
	fmt.Printf("   Buses: %d\n", busCount)
	fmt.Printf("   Other vehicles: %d\n", otherCount)
	
	// Show sample vehicles
	fmt.Println("\n   Sample vehicles:")
	rows, _ := db.Query(`
		SELECT vehicle_number, vehicle_type, make, model, license 
		FROM fleet_vehicles 
		ORDER BY vehicle_type, vehicle_number 
		LIMIT 5
	`)
	defer rows.Close()
	
	for rows.Next() {
		var vehNum sql.NullInt32
		var vType, make, model, license sql.NullString
		rows.Scan(&vehNum, &vType, &make, &model, &license)
		fmt.Printf("   - #%d: %s %s (%s) - License: %s\n", 
			vehNum.Int32, 
			getValue(make), 
			getValue(model), 
			getValue(vType),
			getValue(license))
	}

	// Test 2: Maintenance records
	fmt.Println("\n2. Maintenance Records Test:")
	var maintCount int
	db.QueryRow("SELECT COUNT(*) FROM maintenance_records").Scan(&maintCount)
	fmt.Printf("   Total records: %d\n", maintCount)
	
	// Test 3: ECSE Students
	fmt.Println("\n3. ECSE Students Test:")
	var ecseCount int
	db.QueryRow("SELECT COUNT(*) FROM ecse_students").Scan(&ecseCount)
	fmt.Printf("   Total students: %d\n", ecseCount)
	
	// Test 4: Monthly mileage
	fmt.Println("\n4. Monthly Mileage Reports Test:")
	var mileageCount int
	db.QueryRow("SELECT COUNT(*) FROM monthly_mileage_reports").Scan(&mileageCount)
	fmt.Printf("   Total reports: %d\n", mileageCount)
	
	// Test 5: Check for old tables (should not exist)
	fmt.Println("\n5. Old Tables Check (should be removed):")
	oldTables := []string{
		"school_buses",
		"agency_vehicles", 
		"all_vehicle_mileage",
		"bus_maintenance_logs",
		"vehicle_maintenance_logs",
		"mileage_reports",
		"mileage_records",
	}
	
	for _, table := range oldTables {
		var exists bool
		err := db.QueryRow(`
			SELECT EXISTS(
				SELECT 1 FROM information_schema.tables 
				WHERE table_name = $1
			)`, table).Scan(&exists)
		
		if err == nil {
			if exists {
				fmt.Printf("   ⚠️  %s: Still exists (should be removed)\n", table)
			} else {
				fmt.Printf("   ✅ %s: Removed successfully\n", table)
			}
		}
	}
	
	// Test 6: Foreign key updates
	fmt.Println("\n6. Foreign Key Updates Test:")
	var routeVehicleCount int
	var driverVehicleCount int
	
	db.QueryRow("SELECT COUNT(*) FROM route_assignments WHERE vehicle_id IS NOT NULL").Scan(&routeVehicleCount)
	db.QueryRow("SELECT COUNT(*) FROM driver_logs WHERE vehicle_id IS NOT NULL").Scan(&driverVehicleCount)
	
	fmt.Printf("   route_assignments with vehicle_id: %d\n", routeVehicleCount)
	fmt.Printf("   driver_logs with vehicle_id: %d\n", driverVehicleCount)
	
	fmt.Println("\n=== SUMMARY ===")
	if fleetCount > 0 && maintCount > 0 && ecseCount > 0 {
		fmt.Println("✅ All major tables have data and are accessible!")
		fmt.Println("✅ Database migration was successful!")
	} else {
		fmt.Println("❌ Some tables may be missing data")
	}
}

func getValue(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return "N/A"
}