package main

import (
	"database/sql"
	"fmt"
	"log"
	"regexp"
	"strconv"
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

	// First, let's see what the data looks like
	fmt.Println("\nAnalyzing current data issues...")
	
	rows, err := db.Query(`
		SELECT id, bus_id, located_at, beginning_miles, ending_miles, total_miles
		FROM monthly_mileage_reports
		WHERE (beginning_miles = 0 AND ending_miles = 0 AND total_miles = 0)
		   OR bus_id LIKE 'BUS%'
		LIMIT 20
	`)
	if err != nil {
		log.Fatal("Error querying data:", err)
	}
	defer rows.Close()

	fmt.Println("\nSample problematic records:")
	for rows.Next() {
		var id int
		var busID, locatedAt sql.NullString
		var beginMiles, endMiles, totalMiles sql.NullInt64

		err := rows.Scan(&id, &busID, &locatedAt, &beginMiles, &endMiles, &totalMiles)
		if err != nil {
			log.Printf("Scan error: %v", err)
			continue
		}

		fmt.Printf("ID: %d, BusID: %s, Location: %s, Miles: %d->%d (Total: %d)\n",
			id, busID.String, locatedAt.String, beginMiles.Int64, endMiles.Int64, totalMiles.Int64)
	}

	// Fix bus_id format
	fmt.Println("\nFixing bus_id format...")
	result, err := db.Exec(`
		UPDATE monthly_mileage_reports
		SET bus_id = 
			CASE 
				WHEN bus_id LIKE 'BUS%' THEN 
					REGEXP_REPLACE(bus_id, '^BUS', '')
				ELSE bus_id
			END
		WHERE bus_id LIKE 'BUS%'
	`)
	if err != nil {
		log.Printf("Error fixing bus_id: %v", err)
	} else {
		count, _ := result.RowsAffected()
		fmt.Printf("Fixed %d bus_id values\n", count)
	}

	// Check if located_at contains mileage data
	fmt.Println("\nChecking if mileage data is in wrong columns...")
	
	rows2, err := db.Query(`
		SELECT id, bus_id, located_at
		FROM monthly_mileage_reports
		WHERE located_at ~ '^[0-9,]+$'
		   AND beginning_miles = 0
		LIMIT 10
	`)
	if err == nil {
		defer rows2.Close()
		
		fixCount := 0
		for rows2.Next() {
			var id int
			var busID, locatedAt sql.NullString
			
			err := rows2.Scan(&id, &busID, &locatedAt)
			if err != nil {
				continue
			}
			
			// Extract number from located_at
			mileageStr := strings.ReplaceAll(locatedAt.String, ",", "")
			if mileage, err := strconv.Atoi(mileageStr); err == nil && mileage > 0 {
				// This looks like ending mileage
				_, err = db.Exec(`
					UPDATE monthly_mileage_reports
					SET ending_miles = $1,
					    total_miles = $1 - beginning_miles,
					    located_at = NULL
					WHERE id = $2
				`, mileage, id)
				
				if err == nil {
					fixCount++
					fmt.Printf("Fixed record %d: moved %d from located_at to ending_miles\n", id, mileage)
				}
			}
		}
		fmt.Printf("Fixed %d records with mileage in wrong column\n", fixCount)
	}

	// Fix records where bus_id contains comma-separated numbers
	fmt.Println("\nFixing bus_id with commas...")
	_, err = db.Exec(`
		UPDATE monthly_mileage_reports
		SET bus_id = REPLACE(bus_id, ',', '')
		WHERE bus_id LIKE '%,%'
	`)
	if err != nil {
		log.Printf("Error removing commas from bus_id: %v", err)
	}

	// Look for patterns where mileage might be stored
	fmt.Println("\nLooking for mileage patterns in other fields...")
	
	// Try to extract mileage from bus_make or other fields if they contain numbers
	rows3, err := db.Query(`
		SELECT id, bus_id, bus_make, license_plate, located_at
		FROM monthly_mileage_reports
		WHERE beginning_miles = 0 
		  AND ending_miles = 0
		  AND (bus_make ~ '[0-9]{4,}' OR license_plate ~ '[0-9]{4,}')
		LIMIT 20
	`)
	if err == nil {
		defer rows3.Close()
		
		for rows3.Next() {
			var id int
			var busID, busMake, licensePlate, locatedAt sql.NullString
			
			err := rows3.Scan(&id, &busID, &busMake, &licensePlate, &locatedAt)
			if err != nil {
				continue
			}
			
			fmt.Printf("Potential mileage data - ID: %d, BusID: %s, Make: %s, License: %s, Location: %s\n",
				id, busID.String, busMake.String, licensePlate.String, locatedAt.String)
			
			// Extract numbers from fields
			re := regexp.MustCompile(`\b(\d{4,})\b`)
			
			// Check bus_make for mileage
			if matches := re.FindAllString(busMake.String, -1); len(matches) >= 2 {
				begin, _ := strconv.Atoi(matches[0])
				end, _ := strconv.Atoi(matches[1])
				if begin > 0 && end > begin {
					_, err = db.Exec(`
						UPDATE monthly_mileage_reports
						SET beginning_miles = $1,
						    ending_miles = $2,
						    total_miles = $3
						WHERE id = $4
					`, begin, end, end-begin, id)
					
					if err == nil {
						fmt.Printf("  -> Updated with miles: %d -> %d\n", begin, end)
					}
				}
			}
		}
	}

	// Final summary
	fmt.Println("\nFinal data summary:")
	
	var totalRecords, recordsWithMileage int
	var totalMilesSum sql.NullInt64
	
	err = db.QueryRow(`
		SELECT COUNT(*), 
		       COUNT(CASE WHEN total_miles > 0 THEN 1 END),
		       SUM(total_miles)
		FROM monthly_mileage_reports
	`).Scan(&totalRecords, &recordsWithMileage, &totalMilesSum)
	
	if err == nil {
		fmt.Printf("Total records: %d\n", totalRecords)
		fmt.Printf("Records with mileage: %d (%.1f%%)\n", recordsWithMileage, 
			float64(recordsWithMileage)/float64(totalRecords)*100)
		fmt.Printf("Total miles: %d\n", totalMilesSum.Int64)
	}

	// Show some successful records
	fmt.Println("\nSample records with mileage:")
	rows4, err := db.Query(`
		SELECT bus_id, report_month, report_year, beginning_miles, ending_miles, total_miles
		FROM monthly_mileage_reports
		WHERE total_miles > 0
		ORDER BY total_miles DESC
		LIMIT 10
	`)
	if err == nil {
		defer rows4.Close()
		
		for rows4.Next() {
			var busID, month sql.NullString
			var year sql.NullInt64
			var begin, end, total sql.NullInt64
			
			err := rows4.Scan(&busID, &month, &year, &begin, &end, &total)
			if err == nil {
				fmt.Printf("  %s - %s %d: %d -> %d (Total: %d miles)\n",
					busID.String, month.String, year.Int64,
					begin.Int64, end.Int64, total.Int64)
			}
		}
	}

	fmt.Println("\nData fix complete!")
}