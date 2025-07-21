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

	// Count maintenance records
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM maintenance_records").Scan(&count)
	if err != nil {
		log.Fatalf("Failed to count maintenance records: %v", err)
	}

	fmt.Printf("Total maintenance records in database: %d\n\n", count)

	// Load a few sample records
	rows, err := db.Query(`
		SELECT id, vehicle_number, service_date, mileage, po_number, cost, 
		       work_description, vehicle_id, date
		FROM maintenance_records 
		ORDER BY COALESCE(service_date, date, created_at) DESC
		LIMIT 5
	`)
	if err != nil {
		log.Fatalf("Failed to query maintenance records: %v", err)
	}
	defer rows.Close()

	fmt.Println("Sample maintenance records:")
	fmt.Println("====================================================")
	
	rowCount := 0
	for rows.Next() {
		var id int
		var vehicleNumber, mileage sql.NullInt32
		var serviceDate, date sql.NullTime
		var poNumber, cost, workDescription, vehicleID sql.NullString

		err := rows.Scan(
			&id, &vehicleNumber, &serviceDate, &mileage, &poNumber, 
			&cost, &workDescription, &vehicleID, &date,
		)
		if err != nil {
			log.Printf("Error scanning row: %v", err)
			continue
		}

		rowCount++
		fmt.Printf("Record %d:\n", rowCount)
		fmt.Printf("  ID: %d\n", id)
		if vehicleNumber.Valid {
			fmt.Printf("  Vehicle Number: %d\n", vehicleNumber.Int32)
		}
		if vehicleID.Valid && vehicleID.String != "" {
			fmt.Printf("  Vehicle ID: %s\n", vehicleID.String)
		}
		if serviceDate.Valid {
			fmt.Printf("  Service Date: %s\n", serviceDate.Time.Format("2006-01-02"))
		} else if date.Valid {
			fmt.Printf("  Date: %s\n", date.Time.Format("2006-01-02"))
		}
		if mileage.Valid {
			fmt.Printf("  Mileage: %d\n", mileage.Int32)
		}
		if workDescription.Valid && workDescription.String != "" {
			fmt.Printf("  Work: %s\n", workDescription.String)
		}
		if cost.Valid && cost.String != "" {
			fmt.Printf("  Cost: %s\n", cost.String)
		}
		fmt.Println()
	}

	if rowCount == 0 {
		fmt.Println("No maintenance records found!")
	} else {
		fmt.Printf("Successfully loaded %d sample records out of %d total\n", rowCount, count)
	}
}