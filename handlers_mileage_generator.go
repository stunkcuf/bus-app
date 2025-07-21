package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"time"
)

// generateMileageReportsFromLogsHandler generates mileage reports from driver logs
func generateMileageReportsFromLogsHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	if db == nil {
		http.Error(w, "Database not initialized", http.StatusInternalServerError)
		return
	}

	// Generate reports for the last 3 months
	now := time.Now()
	generated := 0

	for i := 0; i < 3; i++ {
		targetDate := now.AddDate(0, -i, 0)
		year := targetDate.Year()
		month := int(targetDate.Month())

		// Aggregate mileage from driver logs by vehicle and month
		query := `
			SELECT 
				bus_id as vehicle_id,
				$1 as year,
				$2 as month,
				SUM(mileage) as total_mileage,
				COUNT(DISTINCT date) as days_operated,
				COUNT(DISTINCT driver) as drivers_count,
				MIN(date) as first_trip_date,
				MAX(date) as last_trip_date
			FROM driver_logs
			WHERE 
				EXTRACT(YEAR FROM date::date) = $1 
				AND EXTRACT(MONTH FROM date::date) = $2
				AND mileage > 0
			GROUP BY bus_id
		`

		rows, err := db.Query(query, year, month)
		if err != nil {
			log.Printf("Error querying driver logs: %v", err)
			continue
		}

		for rows.Next() {
			var vehicleID string
			var yearVal, monthVal, daysOperated, driversCount int
			var totalMileage float64
			var firstTripDate, lastTripDate sql.NullString

			err := rows.Scan(&vehicleID, &yearVal, &monthVal, &totalMileage, 
				&daysOperated, &driversCount, &firstTripDate, &lastTripDate)
			if err != nil {
				log.Printf("Error scanning row: %v", err)
				continue
			}

			// Calculate averages
			avgDailyMileage := totalMileage / float64(daysOperated)
			
			// Get fuel data if available
			var fuelGallons, fuelCost float64
			err = db.QueryRow(`
				SELECT COALESCE(SUM(gallons), 0), COALESCE(SUM(total_cost), 0)
				FROM fuel_records
				WHERE vehicle_id = $1 
				AND EXTRACT(YEAR FROM date::date) = $2 
				AND EXTRACT(MONTH FROM date::date) = $3
			`, vehicleID, year, month).Scan(&fuelGallons, &fuelCost)
			
			if err != nil {
				log.Printf("Error getting fuel data: %v", err)
				fuelGallons = 0
				fuelCost = 0
			}

			// Calculate MPG if fuel data available
			var mpg float64
			if fuelGallons > 0 {
				mpg = totalMileage / fuelGallons
			}

			// Calculate cost per mile
			var costPerMile float64
			if totalMileage > 0 {
				costPerMile = fuelCost / totalMileage
			}

			// Insert or update mileage report
			_, err = db.Exec(`
				INSERT INTO mileage_reports (
					vehicle_id, year, month, total_mileage, fuel_gallons,
					fuel_cost, mpg, cost_per_mile, days_operated, avg_daily_mileage,
					start_odometer, end_odometer, notes, created_at, updated_at
				) VALUES (
					$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, 0, 0, $11, $12, $12
				) ON CONFLICT (vehicle_id, year, month) DO UPDATE SET
					total_mileage = EXCLUDED.total_mileage,
					fuel_gallons = EXCLUDED.fuel_gallons,
					fuel_cost = EXCLUDED.fuel_cost,
					mpg = EXCLUDED.mpg,
					cost_per_mile = EXCLUDED.cost_per_mile,
					days_operated = EXCLUDED.days_operated,
					avg_daily_mileage = EXCLUDED.avg_daily_mileage,
					notes = EXCLUDED.notes,
					updated_at = EXCLUDED.updated_at
			`, vehicleID, year, month, totalMileage, fuelGallons,
				fuelCost, mpg, costPerMile, daysOperated, avgDailyMileage,
				fmt.Sprintf("Generated from driver logs. %d drivers operated this vehicle.", driversCount),
				time.Now())

			if err != nil {
				log.Printf("Error inserting mileage report: %v", err)
			} else {
				generated++
			}
		}
		rows.Close()
	}

	// Also add some sample driver logs if none exist
	var logCount int
	err := db.QueryRow("SELECT COUNT(*) FROM driver_logs").Scan(&logCount)
	if err != nil || logCount < 10 {
		addSampleDriverLogs()
	}

	log.Printf("Generated %d mileage reports from driver logs", generated)
	
	// Redirect to view mileage reports
	http.Redirect(w, r, "/view-mileage-reports", http.StatusSeeOther)
}

// addSampleDriverLogs adds sample driver logs for demonstration
func addSampleDriverLogs() {
	buses := []string{"B001", "B002", "B003", "B004", "B005", "B010", "B015", "B020"}
	drivers := []string{"jsmith", "mjohnson", "dwilliams", "sbrown"}
	routes := []string{"R001", "R002", "R003", "R004", "R005"}
	
	// Generate logs for the past 30 days
	now := time.Now()
	for d := 0; d < 30; d++ {
		date := now.AddDate(0, 0, -d).Format("2006-01-02")
		
		// Morning and afternoon shifts for each bus
		for i, busID := range buses {
			driver := drivers[i%len(drivers)]
			route := routes[i%len(routes)]
			
			// Morning shift
			morningMileage := float64(25 + (i * 3))
			_, err := db.Exec(`
				INSERT INTO driver_logs (
					driver, bus_id, route_id, date, period,
					departure_time, arrival_time, mileage, attendance, created_at
				) VALUES (
					$1, $2, $3, $4, 'morning', '07:00', '09:00', $5, '[]'::jsonb, $6
				) ON CONFLICT DO NOTHING
			`, driver, busID, route, date, morningMileage, time.Now())
			
			if err != nil {
				log.Printf("Error inserting morning log: %v", err)
			}
			
			// Afternoon shift (skip weekends)
			dayOfWeek := now.AddDate(0, 0, -d).Weekday()
			if dayOfWeek != time.Saturday && dayOfWeek != time.Sunday {
				afternoonMileage := float64(28 + (i * 2))
				_, err = db.Exec(`
					INSERT INTO driver_logs (
						driver, bus_id, route_id, date, period,
						departure_time, arrival_time, mileage, attendance, created_at
					) VALUES (
						$1, $2, $3, $4, 'afternoon', '14:00', '16:30', $5, '[]'::jsonb, $6
					) ON CONFLICT DO NOTHING
				`, driver, busID, route, date, afternoonMileage, time.Now())
				
				if err != nil {
					log.Printf("Error inserting afternoon log: %v", err)
				}
			}
		}
	}
	
	// Add some fuel records too
	for _, busID := range buses {
		for w := 0; w < 4; w++ {
			date := now.AddDate(0, 0, -(w * 7)).Format("2006-01-02")
			gallons := 25.0 + float64(w*2)
			pricePerGallon := 3.50 + (float64(w) * 0.10)
			totalCost := gallons * pricePerGallon
			
			_, err := db.Exec(`
				INSERT INTO fuel_records (
					vehicle_id, date, gallons, price_per_gallon, total_cost,
					location, odometer, driver, created_at
				) VALUES (
					$1, $2, $3, $4, $5, 'Main Depot', 0, 'fleet_manager', $6
				) ON CONFLICT DO NOTHING
			`, busID, date, gallons, pricePerGallon, totalCost, time.Now())
			
			if err != nil {
				log.Printf("Error inserting fuel record: %v", err)
			}
		}
	}
}