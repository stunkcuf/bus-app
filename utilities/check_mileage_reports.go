package main

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func main() {
	// Use the URL from the .env file
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

	// Check monthly_mileage_reports table structure
	var columns []struct {
		ColumnName string `db:"column_name"`
		DataType   string `db:"data_type"`
		IsNullable string `db:"is_nullable"`
	}
	err = db.Select(&columns, `
		SELECT column_name, data_type, is_nullable
		FROM information_schema.columns 
		WHERE table_name = 'monthly_mileage_reports'
		ORDER BY ordinal_position
	`)
	if err != nil {
		log.Fatal("Failed to get columns:", err)
	}
	
	fmt.Println("\nmonthly_mileage_reports table structure:")
	for _, col := range columns {
		fmt.Printf("  %-20s %-20s %s\n", col.ColumnName, col.DataType, col.IsNullable)
	}

	// Check if the table has any data
	var count int
	err = db.Get(&count, "SELECT COUNT(*) FROM monthly_mileage_reports")
	if err != nil {
		log.Fatal("Failed to count records:", err)
	}
	fmt.Printf("\nTotal records: %d\n", count)

	// Get sample data
	if count > 0 {
		fmt.Println("\nSample data:")
		rows, err := db.Query(`
			SELECT id, report_month, report_year, bus_year, bus_make, 
			       license_plate, bus_id, located_at, beginning_miles, 
			       ending_miles, total_miles
			FROM monthly_mileage_reports 
			ORDER BY report_year DESC, report_month DESC
			LIMIT 5
		`)
		if err != nil {
			log.Fatal("Failed to query sample data:", err)
		}
		defer rows.Close()

		for rows.Next() {
			var id sql.NullInt64
			var reportMonth sql.NullString
			var reportYear, busYear sql.NullInt64
			var busMake, licensePlate, busID, locatedAt sql.NullString
			var beginMiles, endMiles, totalMiles sql.NullInt64

			err := rows.Scan(&id, &reportMonth, &reportYear, &busYear, &busMake,
				&licensePlate, &busID, &locatedAt, &beginMiles, &endMiles, &totalMiles)
			if err != nil {
				log.Printf("Scan error: %v", err)
				continue
			}

			fmt.Printf("\n  Record ID: %d\n", id.Int64)
			fmt.Printf("  Month/Year: %s/%d\n", reportMonth.String, reportYear.Int64)
			fmt.Printf("  Bus: %s - %d %s (License: %s)\n", busID.String, busYear.Int64, busMake.String, licensePlate.String)
			fmt.Printf("  Location: %s\n", locatedAt.String)
			fmt.Printf("  Miles: %d -> %d (Total: %d)\n", beginMiles.Int64, endMiles.Int64, totalMiles.Int64)
		}
	}

	// Check for NULL values in key columns
	fmt.Println("\n\nChecking for missing data:")
	
	columns_to_check := []string{"report_month", "report_year", "bus_id", "beginning_miles", "ending_miles", "total_miles"}
	
	for _, col := range columns_to_check {
		var nullCount int
		err = db.Get(&nullCount, fmt.Sprintf("SELECT COUNT(*) FROM monthly_mileage_reports WHERE %s IS NULL", col))
		if err == nil && nullCount > 0 {
			fmt.Printf("  Records with NULL %s: %d\n", col, nullCount)
		}
	}

	// Check unique bus_ids
	fmt.Println("\n\nUnique buses in reports:")
	var busIDs []sql.NullString
	err = db.Select(&busIDs, "SELECT DISTINCT bus_id FROM monthly_mileage_reports WHERE bus_id IS NOT NULL ORDER BY bus_id LIMIT 20")
	if err != nil {
		log.Printf("Failed to get bus IDs: %v", err)
	} else {
		for _, bid := range busIDs {
			if bid.Valid {
				fmt.Printf("  %s\n", bid.String)
			}
		}
	}
}