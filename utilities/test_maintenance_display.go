package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	// Load .env file
	if err := godotenv.Load("../.env"); err != nil {
		log.Printf("Warning: .env file not found")
	}

	// Get database URL
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL not set")
	}

	// Connect to database
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Test connection
	if err := db.Ping(); err != nil {
		log.Fatal("Failed to ping database:", err)
	}

	fmt.Println("=== MAINTENANCE RECORDS DATA CHECK ===")
	fmt.Println()

	// 1. Check total records
	var totalCount int
	err = db.QueryRow("SELECT COUNT(*) FROM maintenance_records").Scan(&totalCount)
	if err != nil {
		log.Fatal("Failed to count records:", err)
	}
	fmt.Printf("Total maintenance records: %d\n\n", totalCount)

	// 2. Check first 10 records to see data structure
	fmt.Println("Sample maintenance records:")
	fmt.Println("=" + repeatString("=", 120))

	rows, err := db.Query(`
		SELECT 
			id,
			vehicle_number,
			vehicle_id,
			service_date,
			date,
			mileage,
			po_number,
			cost,
			work_description,
			created_at
		FROM maintenance_records 
		ORDER BY COALESCE(service_date, date, created_at) DESC
		LIMIT 10
	`)
	if err != nil {
		log.Fatal("Failed to query records:", err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var id int
		var vehicleNumber sql.NullInt32
		var vehicleID, poNumber, cost, workDescription sql.NullString
		var serviceDate, date sql.NullTime
		var mileage sql.NullInt32
		var createdAt time.Time

		err := rows.Scan(&id, &vehicleNumber, &vehicleID, &serviceDate, &date, 
			&mileage, &poNumber, &cost, &workDescription, &createdAt)
		if err != nil {
			log.Printf("Error scanning row: %v", err)
			continue
		}

		count++
		fmt.Printf("Record #%d (ID: %d):\n", count, id)
		
		// Vehicle info
		if vehicleNumber.Valid {
			fmt.Printf("  Vehicle Number: %d\n", vehicleNumber.Int32)
		} else {
			fmt.Printf("  Vehicle Number: NULL\n")
		}
		
		if vehicleID.Valid {
			fmt.Printf("  Vehicle ID: %s\n", vehicleID.String)
		} else {
			fmt.Printf("  Vehicle ID: NULL\n")
		}

		// Date info
		if serviceDate.Valid {
			fmt.Printf("  Service Date: %s\n", serviceDate.Time.Format("2006-01-02"))
		} else if date.Valid {
			fmt.Printf("  Date: %s\n", date.Time.Format("2006-01-02"))
		} else {
			fmt.Printf("  Date: NULL (Created: %s)\n", createdAt.Format("2006-01-02"))
		}

		// Other fields
		if mileage.Valid {
			fmt.Printf("  Mileage: %d\n", mileage.Int32)
		} else {
			fmt.Printf("  Mileage: NULL\n")
		}

		if cost.Valid {
			fmt.Printf("  Cost: %s\n", cost.String)
		} else {
			fmt.Printf("  Cost: NULL\n")
		}

		if poNumber.Valid {
			fmt.Printf("  PO Number: %s\n", poNumber.String)
		} else {
			fmt.Printf("  PO Number: NULL\n")
		}

		if workDescription.Valid {
			desc := workDescription.String
			if len(desc) > 60 {
				desc = desc[:60] + "..."
			}
			fmt.Printf("  Work: %s\n", desc)
		} else {
			fmt.Printf("  Work: NULL\n")
		}
		
		fmt.Println()
	}

	// 3. Check for common issues
	fmt.Println("\n=== DATA QUALITY CHECK ===")
	
	// Records with no vehicle identifier
	var noVehicle int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM maintenance_records 
		WHERE vehicle_number IS NULL AND vehicle_id IS NULL
	`).Scan(&noVehicle)
	if err == nil {
		fmt.Printf("Records with no vehicle info: %d\n", noVehicle)
	}

	// Records with no date
	var noDate int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM maintenance_records 
		WHERE service_date IS NULL AND date IS NULL
	`).Scan(&noDate)
	if err == nil {
		fmt.Printf("Records with no date: %d\n", noDate)
	}

	// Check vehicle_id format
	fmt.Println("\n=== VEHICLE ID FORMATS ===")
	rows2, err := db.Query(`
		SELECT DISTINCT vehicle_id 
		FROM maintenance_records 
		WHERE vehicle_id IS NOT NULL 
		LIMIT 20
	`)
	if err == nil {
		defer rows2.Close()
		for rows2.Next() {
			var vid sql.NullString
			if err := rows2.Scan(&vid); err == nil && vid.Valid {
				fmt.Printf("  %s\n", vid.String)
			}
		}
	}

	// Check what GetVehicleIdentifier would return
	fmt.Println("\n=== VEHICLE IDENTIFIER LOGIC TEST ===")
	rows3, err := db.Query(`
		SELECT 
			vehicle_number,
			vehicle_id,
			CASE 
				WHEN vehicle_number IS NOT NULL THEN 'Vehicle #' || vehicle_number::text
				WHEN vehicle_id IS NOT NULL THEN vehicle_id
				ELSE 'Unknown'
			END as display_identifier
		FROM maintenance_records 
		LIMIT 10
	`)
	if err == nil {
		defer rows3.Close()
		for rows3.Next() {
			var vehicleNumber sql.NullInt32
			var vehicleID sql.NullString
			var displayID string
			if err := rows3.Scan(&vehicleNumber, &vehicleID, &displayID); err == nil {
				fmt.Printf("VehicleNumber: %v, VehicleID: %v => Display: %s\n", 
					vehicleNumber.Valid, vehicleID.Valid, displayID)
			}
		}
	}
}

func repeatString(s string, count int) string {
	result := ""
	for i := 0; i < count; i++ {
		result += s
	}
	return result
}