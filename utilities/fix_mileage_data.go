package main

import (
	"database/sql"
	"fmt"
	"log"
	"strings"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func main() {
	// Connect to database
	dbURL := "postgresql://postgres:jTKOWGEzlprRGbkPBCgGxsgnwyLeGoDL@shortline.proxy.rlwy.net:40148/railway"

	db, err := sqlx.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("Failed to connect:", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal("Failed to ping:", err)
	}
	fmt.Println("Connected to database")

	// Clear existing bad data from monthly_mileage_reports
	fmt.Println("\nClearing existing malformed data...")
	_, err = db.Exec("DELETE FROM monthly_mileage_reports WHERE beginning_miles = 0 AND ending_miles = 0 AND total_miles = 0")
	if err != nil {
		log.Printf("Error clearing data: %v", err)
	}

	// Migrate data from school_buses table
	fmt.Println("\nMigrating school bus data...")
	migrated := 0
	
	rows, err := db.Query(`
		SELECT report_month, report_year, bus_year, bus_make, license_plate,
		       bus_id, location, beginning_miles, ending_miles, total_miles
		FROM school_buses
		WHERE beginning_miles > 0 OR ending_miles > 0 OR total_miles > 0
	`)
	if err != nil {
		log.Printf("Error querying school_buses: %v", err)
	} else {
		defer rows.Close()
		
		for rows.Next() {
			var reportMonth, busMake, licensePlate, busID, location sql.NullString
			var reportYear, busYear sql.NullInt64
			var beginMiles, endMiles, totalMiles sql.NullInt64

			err := rows.Scan(&reportMonth, &reportYear, &busYear, &busMake, &licensePlate,
				&busID, &location, &beginMiles, &endMiles, &totalMiles)
			if err != nil {
				log.Printf("Scan error: %v", err)
				continue
			}

			// Clean up bus_id - remove "BUS" prefix if present
			cleanBusID := busID.String
			if strings.HasPrefix(cleanBusID, "BUS") {
				cleanBusID = strings.TrimPrefix(cleanBusID, "BUS")
			}

			// Insert into monthly_mileage_reports
			_, err = db.Exec(`
				INSERT INTO monthly_mileage_reports 
				(report_month, report_year, bus_year, bus_make, license_plate,
				 bus_id, located_at, beginning_miles, ending_miles, total_miles)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
				ON CONFLICT (report_month, report_year, bus_id) DO UPDATE SET
					bus_year = EXCLUDED.bus_year,
					bus_make = EXCLUDED.bus_make,
					license_plate = EXCLUDED.license_plate,
					located_at = EXCLUDED.located_at,
					beginning_miles = EXCLUDED.beginning_miles,
					ending_miles = EXCLUDED.ending_miles,
					total_miles = EXCLUDED.total_miles,
					updated_at = CURRENT_TIMESTAMP
			`, reportMonth, reportYear, busYear, busMake, licensePlate,
				cleanBusID, location, beginMiles, endMiles, totalMiles)

			if err != nil {
				log.Printf("Error inserting bus %s: %v", busID.String, err)
			} else {
				migrated++
			}
		}
	}
	
	fmt.Printf("Migrated %d school bus records\n", migrated)

	// Migrate data from agency_vehicles table
	fmt.Println("\nMigrating agency vehicle data...")
	agencyMigrated := 0
	
	rows2, err := db.Query(`
		SELECT report_month, report_year, vehicle_year, make_model, license_plate,
		       vehicle_id, location, beginning_miles, ending_miles, total_miles
		FROM agency_vehicles
		WHERE beginning_miles > 0 OR ending_miles > 0 OR total_miles > 0
	`)
	if err != nil {
		log.Printf("Error querying agency_vehicles: %v", err)
	} else {
		defer rows2.Close()
		
		for rows2.Next() {
			var reportMonth, makeModel, licensePlate, vehicleID, location sql.NullString
			var reportYear, vehicleYear sql.NullInt64
			var beginMiles, endMiles, totalMiles sql.NullInt64

			err := rows2.Scan(&reportMonth, &reportYear, &vehicleYear, &makeModel, &licensePlate,
				&vehicleID, &location, &beginMiles, &endMiles, &totalMiles)
			if err != nil {
				log.Printf("Scan error: %v", err)
				continue
			}

			// Insert into monthly_mileage_reports
			_, err = db.Exec(`
				INSERT INTO monthly_mileage_reports 
				(report_month, report_year, bus_year, bus_make, license_plate,
				 bus_id, located_at, beginning_miles, ending_miles, total_miles)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
				ON CONFLICT (report_month, report_year, bus_id) DO UPDATE SET
					bus_year = EXCLUDED.bus_year,
					bus_make = EXCLUDED.bus_make,
					license_plate = EXCLUDED.license_plate,
					located_at = EXCLUDED.located_at,
					beginning_miles = EXCLUDED.beginning_miles,
					ending_miles = EXCLUDED.ending_miles,
					total_miles = EXCLUDED.total_miles,
					updated_at = CURRENT_TIMESTAMP
			`, reportMonth, reportYear, vehicleYear, makeModel, licensePlate,
				vehicleID, location, beginMiles, endMiles, totalMiles)

			if err != nil {
				log.Printf("Error inserting vehicle %s: %v", vehicleID.String, err)
			} else {
				agencyMigrated++
			}
		}
	}
	
	fmt.Printf("Migrated %d agency vehicle records\n", agencyMigrated)

	// Check the results
	fmt.Println("\nChecking monthly_mileage_reports after migration:")
	
	var count int
	var totalMilesSum sql.NullInt64
	err = db.QueryRow(`
		SELECT COUNT(*), SUM(total_miles)
		FROM monthly_mileage_reports
		WHERE total_miles > 0
	`).Scan(&count, &totalMilesSum)
	
	if err == nil {
		fmt.Printf("Records with mileage: %d\n", count)
		fmt.Printf("Total miles: %d\n", totalMilesSum.Int64)
	}

	// Show sample records
	fmt.Println("\nSample records after migration:")
	rows3, err := db.Query(`
		SELECT bus_id, bus_make, report_month, report_year, 
		       beginning_miles, ending_miles, total_miles
		FROM monthly_mileage_reports
		WHERE total_miles > 0
		ORDER BY report_year DESC, report_month DESC
		LIMIT 10
	`)
	if err == nil {
		defer rows3.Close()
		
		for rows3.Next() {
			var busID, busMake, month sql.NullString
			var year sql.NullInt64
			var begin, end, total sql.NullInt64
			
			err := rows3.Scan(&busID, &busMake, &month, &year, &begin, &end, &total)
			if err == nil {
				fmt.Printf("  %s (%s) - %s %d: %d -> %d (Total: %d)\n",
					busID.String, busMake.String, month.String, year.Int64,
					begin.Int64, end.Int64, total.Int64)
			}
		}
	}

	fmt.Println("\nMigration complete!")
}