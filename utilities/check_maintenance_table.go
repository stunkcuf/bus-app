package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func main() {
	// Get DATABASE_URL from environment variable or use default
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		// Use the URL from the .env file
		dbURL = "postgresql://postgres:jTKOWGEzlprRGbkPBCgGxsgnwyLeGoDL@shortline.proxy.rlwy.net:40148/railway"
	}

	db, err := sqlx.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("Failed to connect:", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal("Failed to ping:", err)
	}
	fmt.Println("Connected to database")

	// Check maintenance_records table structure
	var columns []struct {
		ColumnName string `db:"column_name"`
		DataType   string `db:"data_type"`
		IsNullable string `db:"is_nullable"`
	}
	err = db.Select(&columns, `
		SELECT column_name, data_type, is_nullable
		FROM information_schema.columns 
		WHERE table_name = 'maintenance_records'
		ORDER BY ordinal_position
	`)
	if err != nil {
		log.Fatal("Failed to get columns:", err)
	}
	
	fmt.Println("\nmaintenance_records table structure:")
	for _, col := range columns {
		fmt.Printf("  %-20s %-20s %s\n", col.ColumnName, col.DataType, col.IsNullable)
	}

	// Check if the table has any data
	var count int
	err = db.Get(&count, "SELECT COUNT(*) FROM maintenance_records")
	if err != nil {
		log.Fatal("Failed to count records:", err)
	}
	fmt.Printf("\nTotal records: %d\n", count)

	// Get sample data
	if count > 0 {
		fmt.Println("\nSample data:")
		rows, err := db.Query(`
			SELECT id, vehicle_number, vehicle_id, service_date, date, 
			       work_description, mileage, cost
			FROM maintenance_records 
			LIMIT 3
		`)
		if err != nil {
			log.Fatal("Failed to query sample data:", err)
		}
		defer rows.Close()

		for rows.Next() {
			var id sql.NullInt64
			var vehicleNumber sql.NullInt64
			var vehicleID, workDesc sql.NullString
			var serviceDate, date sql.NullTime
			var mileage sql.NullInt64
			var cost sql.NullFloat64

			err := rows.Scan(&id, &vehicleNumber, &vehicleID, &serviceDate, &date, 
				&workDesc, &mileage, &cost)
			if err != nil {
				log.Printf("Scan error: %v", err)
				continue
			}

			fmt.Printf("\n  Record ID: %d\n", id.Int64)
			if vehicleNumber.Valid {
				fmt.Printf("  Vehicle Number: %d\n", vehicleNumber.Int64)
			}
			if vehicleID.Valid {
				fmt.Printf("  Vehicle ID: %s\n", vehicleID.String)
			}
			if serviceDate.Valid {
				fmt.Printf("  Service Date: %s\n", serviceDate.Time.Format("2006-01-02"))
			}
			if date.Valid {
				fmt.Printf("  Date: %s\n", date.Time.Format("2006-01-02"))
			}
			if workDesc.Valid {
				fmt.Printf("  Work: %s\n", workDesc.String)
			}
			if mileage.Valid {
				fmt.Printf("  Mileage: %d\n", mileage.Int64)
			}
			if cost.Valid {
				fmt.Printf("  Cost: %.2f\n", cost.Float64)
			}
		}
	}

	// Check distinct vehicle_ids
	fmt.Println("\n\nDistinct vehicle IDs in maintenance_records:")
	var vehicleIDs []sql.NullString
	err = db.Select(&vehicleIDs, "SELECT DISTINCT vehicle_id FROM maintenance_records WHERE vehicle_id IS NOT NULL LIMIT 10")
	if err != nil {
		log.Printf("Failed to get vehicle IDs: %v", err)
	} else {
		for _, vid := range vehicleIDs {
			if vid.Valid {
				fmt.Printf("  %s\n", vid.String)
			}
		}
	}

	// Check buses table
	fmt.Println("\n\nBuses in buses table:")
	var busIDs []string
	err = db.Select(&busIDs, "SELECT bus_id FROM buses LIMIT 10")
	if err != nil {
		log.Printf("Failed to get bus IDs: %v", err)
	} else {
		for _, bid := range busIDs {
			fmt.Printf("  %s\n", bid)
		}
	}
}