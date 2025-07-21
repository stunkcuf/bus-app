package main

import (
	"fmt"
	"log"

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

	// Update total_miles where we have beginning and ending miles
	fmt.Println("\nCalculating total_miles from beginning and ending miles...")
	
	result, err := db.Exec(`
		UPDATE monthly_mileage_reports
		SET total_miles = ending_miles - beginning_miles
		WHERE ending_miles > 0 
		  AND beginning_miles >= 0
		  AND ending_miles > beginning_miles
		  AND (total_miles = 0 OR total_miles IS NULL)
	`)
	
	if err != nil {
		log.Printf("Error calculating total_miles: %v", err)
	} else {
		count, _ := result.RowsAffected()
		fmt.Printf("Updated %d records with calculated total_miles\n", count)
	}

	// For records with only ending_miles (beginning = 0), assume that's the total
	fmt.Println("\nSetting total_miles for records with only ending_miles...")
	
	result2, err := db.Exec(`
		UPDATE monthly_mileage_reports
		SET total_miles = ending_miles
		WHERE ending_miles > 0 
		  AND beginning_miles = 0
		  AND (total_miles = 0 OR total_miles IS NULL)
	`)
	
	if err != nil {
		log.Printf("Error setting total_miles: %v", err)
	} else {
		count, _ := result2.RowsAffected()
		fmt.Printf("Updated %d records where ending_miles = total_miles\n", count)
	}

	// Fix any negative total_miles
	fmt.Println("\nFixing negative total_miles...")
	
	result3, err := db.Exec(`
		UPDATE monthly_mileage_reports
		SET total_miles = ABS(total_miles)
		WHERE total_miles < 0
	`)
	
	if err != nil {
		log.Printf("Error fixing negative miles: %v", err)
	} else {
		count, _ := result3.RowsAffected()
		fmt.Printf("Fixed %d records with negative miles\n", count)
	}

	// Show summary
	fmt.Println("\nFinal summary:")
	
	var stats struct {
		TotalRecords       int
		RecordsWithMileage int
		TotalMilesSum      int64
		AvgMilesPerRecord  float64
	}
	
	err = db.Get(&stats, `
		SELECT 
			COUNT(*) as total_records,
			COUNT(CASE WHEN total_miles > 0 THEN 1 END) as records_with_mileage,
			COALESCE(SUM(total_miles), 0) as total_miles_sum,
			COALESCE(AVG(CASE WHEN total_miles > 0 THEN total_miles END), 0) as avg_miles_per_record
		FROM monthly_mileage_reports
	`)
	
	if err == nil {
		fmt.Printf("Total records: %d\n", stats.TotalRecords)
		fmt.Printf("Records with mileage: %d (%.1f%%)\n", 
			stats.RecordsWithMileage, 
			float64(stats.RecordsWithMileage)/float64(stats.TotalRecords)*100)
		fmt.Printf("Total miles across all records: %d\n", stats.TotalMilesSum)
		fmt.Printf("Average miles per record (with mileage): %.0f\n", stats.AvgMilesPerRecord)
	}

	// Show top 10 records by mileage
	fmt.Println("\nTop 10 records by mileage:")
	
	type Record struct {
		BusID      string `db:"bus_id"`
		Month      string `db:"report_month"`
		Year       int    `db:"report_year"`
		BeginMiles int    `db:"beginning_miles"`
		EndMiles   int    `db:"ending_miles"`
		TotalMiles int    `db:"total_miles"`
	}
	
	var records []Record
	err = db.Select(&records, `
		SELECT bus_id, report_month, report_year, 
		       beginning_miles, ending_miles, total_miles
		FROM monthly_mileage_reports
		WHERE total_miles > 0
		ORDER BY total_miles DESC
		LIMIT 10
	`)
	
	if err == nil {
		for i, r := range records {
			fmt.Printf("%d. Bus %s - %s %d: %d -> %d = %d miles\n",
				i+1, r.BusID, r.Month, r.Year,
				r.BeginMiles, r.EndMiles, r.TotalMiles)
		}
	}

	fmt.Println("\nMileage calculation complete!")
}