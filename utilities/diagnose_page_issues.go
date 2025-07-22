package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"

	_ "github.com/lib/pq"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgresql://postgres:jTKOWGEzlprRGbkPBCgGxsgnwyLeGoDL@shortline.proxy.rlwy.net:40148/railway"
		fmt.Println("Using hardcoded database URL")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer db.Close()

	fmt.Println("🔍 DIAGNOSING PAGE DATA ISSUES")
	fmt.Println("==============================")

	// 1. Check users and roles
	fmt.Println("\n1. USERS & ROLES:")
	fmt.Println("-----------------")
	rows, err := db.Query("SELECT username, role, status FROM users ORDER BY role, username")
	if err != nil {
		fmt.Printf("❌ Error querying users: %v\n", err)
	} else {
		defer rows.Close()
		userCount := map[string]int{"manager": 0, "driver": 0}
		for rows.Next() {
			var username, role, status string
			rows.Scan(&username, &role, &status)
			fmt.Printf("• %s (role: %s, status: %s)\n", username, role, status)
			userCount[role]++
		}
		fmt.Printf("\nSummary: %d managers, %d drivers\n", userCount["manager"], userCount["driver"])
	}

	// 2. Check students data
	fmt.Println("\n2. STUDENTS DATA:")
	fmt.Println("-----------------")
	var studentCount int
	err = db.QueryRow("SELECT COUNT(*) FROM students").Scan(&studentCount)
	if err != nil {
		fmt.Printf("❌ Error counting students: %v\n", err)
	} else {
		fmt.Printf("✅ Total students: %d\n", studentCount)
		
		// Check active students
		var activeCount int
		err = db.QueryRow("SELECT COUNT(*) FROM students WHERE active = true").Scan(&activeCount)
		if err == nil {
			fmt.Printf("✅ Active students: %d\n", activeCount)
		}
		
		// Check students with routes
		var routedCount int
		err = db.QueryRow("SELECT COUNT(*) FROM students WHERE route_id IS NOT NULL AND route_id != ''").Scan(&routedCount)
		if err == nil {
			fmt.Printf("✅ Students with routes: %d\n", routedCount)
		}
		
		// Check students with drivers
		var driverCount int
		err = db.QueryRow("SELECT COUNT(*) FROM students WHERE driver IS NOT NULL AND driver != ''").Scan(&driverCount)
		if err == nil {
			fmt.Printf("✅ Students assigned to drivers: %d\n", driverCount)
		}
	}

	// 3. Check ECSE data
	fmt.Println("\n3. ECSE DATA:")
	fmt.Println("--------------")
	var ecseCount int
	err = db.QueryRow("SELECT COUNT(*) FROM ecse_students").Scan(&ecseCount)
	if err != nil {
		// Table might not exist
		fmt.Printf("❌ ECSE table might not exist: %v\n", err)
		
		// Try alternative table names
		err = db.QueryRow("SELECT COUNT(*) FROM ecse_student").Scan(&ecseCount)
		if err != nil {
			fmt.Printf("❌ No ECSE data found\n")
		} else {
			fmt.Printf("✅ ECSE students (ecse_student table): %d\n", ecseCount)
		}
	} else {
		fmt.Printf("✅ ECSE students: %d\n", ecseCount)
	}

	// 4. Check mileage data
	fmt.Println("\n4. MILEAGE DATA:")
	fmt.Println("----------------")
	
	// Check monthly mileage reports
	var monthlyCount int
	err = db.QueryRow("SELECT COUNT(*) FROM monthly_mileage_reports").Scan(&monthlyCount)
	if err != nil {
		fmt.Printf("❌ Error counting monthly mileage: %v\n", err)
	} else {
		fmt.Printf("✅ Monthly mileage reports: %d\n", monthlyCount)
	}
	
	// Check mileage reports
	var mileageCount int
	err = db.QueryRow("SELECT COUNT(*) FROM mileage_reports").Scan(&mileageCount)
	if err != nil {
		fmt.Printf("❌ Mileage reports table might not exist: %v\n", err)
	} else {
		fmt.Printf("✅ Mileage reports: %d\n", mileageCount)
	}
	
	// Check driver logs (which contain mileage)
	var driverLogCount int
	err = db.QueryRow("SELECT COUNT(*) FROM driver_logs").Scan(&driverLogCount)
	if err == nil {
		fmt.Printf("✅ Driver logs (with mileage): %d\n", driverLogCount)
		
		// Check logs with actual mileage data
		var mileageLogCount int
		err = db.QueryRow("SELECT COUNT(*) FROM driver_logs WHERE mileage > 0").Scan(&mileageLogCount)
		if err == nil {
			fmt.Printf("✅ Driver logs with mileage data: %d\n", mileageLogCount)
		}
	}

	// 5. Check route assignments (needed for driver access)
	fmt.Println("\n5. ROUTE ASSIGNMENTS:")
	fmt.Println("---------------------")
	var assignmentCount int
	err = db.QueryRow("SELECT COUNT(*) FROM route_assignments").Scan(&assignmentCount)
	if err != nil {
		fmt.Printf("❌ Error counting route assignments: %v\n", err)
	} else {
		fmt.Printf("✅ Route assignments: %d\n", assignmentCount)
		
		// Show actual assignments
		rows, err := db.Query("SELECT driver, bus_id, route_id FROM route_assignments LIMIT 5")
		if err == nil {
			defer rows.Close()
			fmt.Println("\nSample assignments:")
			for rows.Next() {
				var driver, busID, routeID string
				rows.Scan(&driver, &busID, &routeID)
				fmt.Printf("  • Driver: %s, Bus: %s, Route: %s\n", driver, busID, routeID)
			}
		}
	}

	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("DIAGNOSIS SUMMARY:")
	fmt.Println(strings.Repeat("=", 50))
	
	if studentCount == 0 {
		fmt.Println("⚠️  No students in database - /students page will be empty")
	}
	if ecseCount == 0 {
		fmt.Println("⚠️  No ECSE data - /view-ecse-reports will be empty")
	}
	if monthlyCount == 0 && mileageCount == 0 {
		fmt.Println("⚠️  No mileage data - /view-mileage-reports will be empty")
	}
	// userCount was defined inside the users section, so we need to check differently
	var driverUserCount int
	db.QueryRow("SELECT COUNT(*) FROM users WHERE role = 'driver'").Scan(&driverUserCount)
	if driverUserCount == 0 {
		fmt.Println("⚠️  No driver users - driver pages won't be accessible")
	}
}