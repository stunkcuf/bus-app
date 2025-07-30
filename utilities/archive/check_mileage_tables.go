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
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer db.Close()

	fmt.Println("🔍 Checking Mileage Tables")
	fmt.Println("=" + strings.Repeat("=", 50))

	// Check if mileage_reports table exists
	fmt.Println("\n1. Checking mileage_reports table:")
	var exists bool
	err = db.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables 
			WHERE table_name = 'mileage_reports'
		)
	`).Scan(&exists)
	
	if exists {
		fmt.Println("   ✅ Table exists")
		
		// Get column info
		rows, err := db.Query(`
			SELECT column_name, data_type, is_nullable
			FROM information_schema.columns
			WHERE table_name = 'mileage_reports'
			ORDER BY ordinal_position
		`)
		if err == nil {
			defer rows.Close()
			fmt.Println("   Columns:")
			for rows.Next() {
				var col, dtype, nullable string
				rows.Scan(&col, &dtype, &nullable)
				fmt.Printf("     - %s (%s) %s\n", col, dtype, nullable)
			}
		}
		
		// Get row count
		var count int
		db.QueryRow("SELECT COUNT(*) FROM mileage_reports").Scan(&count)
		fmt.Printf("   Row count: %d\n", count)
	} else {
		fmt.Println("   ❌ Table does not exist")
	}

	// Check monthly_mileage_reports table
	fmt.Println("\n2. Checking monthly_mileage_reports table:")
	err = db.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables 
			WHERE table_name = 'monthly_mileage_reports'
		)
	`).Scan(&exists)
	
	if exists {
		fmt.Println("   ✅ Table exists")
		
		// Get column info
		rows, err := db.Query(`
			SELECT column_name, data_type, is_nullable
			FROM information_schema.columns
			WHERE table_name = 'monthly_mileage_reports'
			ORDER BY ordinal_position
		`)
		if err == nil {
			defer rows.Close()
			fmt.Println("   Columns:")
			for rows.Next() {
				var col, dtype, nullable string
				rows.Scan(&col, &dtype, &nullable)
				fmt.Printf("     - %s (%s) %s\n", col, dtype, nullable)
			}
		}
		
		// Get row count and sample data
		var count int
		db.QueryRow("SELECT COUNT(*) FROM monthly_mileage_reports").Scan(&count)
		fmt.Printf("   Row count: %d\n", count)
		
		if count > 0 {
			fmt.Println("\n   Sample data:")
			rows, err := db.Query("SELECT vehicle_id, year, month, total_mileage FROM monthly_mileage_reports LIMIT 5")
			if err == nil {
				defer rows.Close()
				for rows.Next() {
					var vehicleID string
					var year, month int
					var mileage float64
					rows.Scan(&vehicleID, &year, &month, &mileage)
					fmt.Printf("     - Vehicle %s: %d/%d = %.2f miles\n", vehicleID, month, year, mileage)
				}
			}
		}
	} else {
		fmt.Println("   ❌ Table does not exist")
	}

	// Check driver_logs for mileage data
	fmt.Println("\n3. Checking driver_logs mileage data:")
	var logCount int
	var totalMileage float64
	err = db.QueryRow(`
		SELECT COUNT(*), COALESCE(SUM(mileage), 0)
		FROM driver_logs 
		WHERE mileage > 0
	`).Scan(&logCount, &totalMileage)
	
	if err == nil {
		fmt.Printf("   Driver logs with mileage: %d\n", logCount)
		fmt.Printf("   Total mileage logged: %.2f miles\n", totalMileage)
	}

	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("RECOMMENDATION:")
	fmt.Println("The handler is looking for 'mileage_reports' but data is in 'monthly_mileage_reports'")
	fmt.Println("Need to either:")
	fmt.Println("1. Update the handler to use monthly_mileage_reports")
	fmt.Println("2. Create a view or migrate data to mileage_reports")
}