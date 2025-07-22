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
		fmt.Println("Using hardcoded database URL for testing")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer db.Close()

	fmt.Println("🔍 Diagnosing potential empty table issues...")

	// Define key tables and their purposes
	tables := map[string]string{
		"users":                "User accounts (manager/driver login)",
		"buses":                "School bus fleet data",
		"vehicles":             "Company vehicle data", 
		"routes":               "Bus route definitions",
		"students":             "Student roster and pickup info",
		"route_assignments":    "Driver-route-bus assignments",
		"driver_logs":          "Daily driver route logs",
		"maintenance_records":  "Vehicle maintenance history",
		"service_records":      "Service record history",
		"bus_maintenance_logs": "Bus-specific maintenance",
		"vehicle_maintenance_logs": "Vehicle-specific maintenance",
		"ecse_students":        "Special education student data",
		"mileage_reports":      "Monthly mileage reports",
		"fuel_records":         "Fuel consumption tracking",
	}

	fmt.Println("\n📊 Table Data Analysis:")
	fmt.Println("============================================================")
	
	for table, description := range tables {
		var count int
		var recentCount int
		
		// Get total count
		err := db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&count)
		if err != nil {
			fmt.Printf("❌ %-25s | ERROR: %v\n", table, err)
			continue
		}

		// Get recent count (last 30 days if created_at exists)
		recentQuery := fmt.Sprintf(`
			SELECT COUNT(*) FROM %s 
			WHERE created_at >= CURRENT_DATE - INTERVAL '30 days'
		`, table)
		
		err = db.QueryRow(recentQuery).Scan(&recentCount)
		if err != nil {
			recentCount = -1 // Indicates no created_at column or error
		}

		// Status indicator
		status := "✅"
		if count == 0 {
			status = "❌ EMPTY"
		} else if count < 5 {
			status = "⚠️  FEW"
		}

		if recentCount >= 0 {
			fmt.Printf("%s %-22s | Total: %4d | Recent: %3d | %s\n", 
				status, table, count, recentCount, description)
		} else {
			fmt.Printf("%s %-22s | Total: %4d | Recent: N/A | %s\n", 
				status, table, count, description)
		}
	}

	// Check specific scenarios that cause empty displays
	fmt.Println("\n🔍 Checking specific empty table scenarios...")

	// 1. Drivers with no route assignments
	var driversWithoutRoutes int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM users u
		WHERE u.role = 'driver' AND u.status = 'active'
		AND u.username NOT IN (SELECT driver FROM route_assignments WHERE driver IS NOT NULL)
	`).Scan(&driversWithoutRoutes)
	if err == nil {
		if driversWithoutRoutes > 0 {
			fmt.Printf("⚠️  %d active drivers have no route assignments\n", driversWithoutRoutes)
			fmt.Println("   → These drivers will see 'No Route Assignment' on dashboard")
		}
	}

	// 2. Routes with no students
	var routesWithoutStudents int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM routes r
		WHERE r.route_id NOT IN (SELECT DISTINCT route_id FROM students WHERE route_id IS NOT NULL AND active = true)
	`).Scan(&routesWithoutStudents)
	if err == nil {
		if routesWithoutStudents > 0 {
			fmt.Printf("⚠️  %d routes have no active students assigned\n", routesWithoutStudents)
			fmt.Println("   → Driver dashboards for these routes will show empty student lists")
		}
	}

	// 3. Vehicles with no maintenance records
	var vehiclesWithoutMaintenance int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM buses b
		WHERE b.bus_id NOT IN (SELECT DISTINCT vehicle_id FROM maintenance_records WHERE vehicle_id IS NOT NULL)
	`).Scan(&vehiclesWithoutMaintenance)
	if err == nil {
		if vehiclesWithoutMaintenance > 0 {
			fmt.Printf("⚠️  %d buses have no maintenance records\n", vehiclesWithoutMaintenance)
			fmt.Println("   → Fleet maintenance pages will show empty history for these vehicles")
		}
	}

	// 4. Check for recent activity
	fmt.Println("\n📈 Recent Activity Check (Last 7 days):")
	recentTables := []string{"driver_logs", "maintenance_records", "fuel_records"}
	
	for _, table := range recentTables {
		var recentActivity int
		err = db.QueryRow(fmt.Sprintf(`
			SELECT COUNT(*) FROM %s 
			WHERE created_at >= CURRENT_DATE - INTERVAL '7 days'
		`, table)).Scan(&recentActivity)
		
		if err == nil {
			status := "✅"
			if recentActivity == 0 {
				status = "⚠️  NO RECENT ACTIVITY"
			}
			fmt.Printf("  %s %-20s: %d records\n", status, table, recentActivity)
		}
	}

	// 5. Check for potential data loading issues
	fmt.Println("\n🔧 Checking for potential query issues...")
	
	// Check if maintenance queries will return data
	var maintenanceQueryResult int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM maintenance_records 
		WHERE vehicle_id IS NOT NULL 
		AND (service_date IS NOT NULL OR date IS NOT NULL)
	`).Scan(&maintenanceQueryResult)
	
	if err == nil {
		if maintenanceQueryResult == 0 {
			fmt.Println("❌ Maintenance records query will return empty - vehicle_id or dates missing")
		} else {
			fmt.Printf("✅ Maintenance records query should work - %d records available\n", maintenanceQueryResult)
		}
	}

	fmt.Println("\n🏁 Diagnosis complete!")
	fmt.Println("\nTo test driver experience:")
	fmt.Println("  1. Login as 'driver1' with password 'password'")
	fmt.Println("  2. Check if route assignment and students appear")
	fmt.Println("  3. Try filling out a daily route log")
}