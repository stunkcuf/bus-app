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

// MaintenanceRecord struct - matches actual database schema
type MaintenanceRecord struct {
	ID              int
	VehicleNumber   *int
	ServiceDate     *time.Time
	Mileage         *int
	PONumber        *string
	Cost            *float64
	WorkDescription *string
	RawData         *string
	CreatedAt       *time.Time
	UpdatedAt       *time.Time
	VehicleID       *string
	Date            *time.Time
}

// Vehicle struct - minimal version for testing
type Vehicle struct {
	ID     int
	Name   string
	Number string
}

var db *sql.DB

func main() {
	// Load environment variables
	if err := godotenv.Load("../.env"); err != nil {
		log.Printf("Warning: .env file not found: %v", err)
	}

	// Connect to database
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable is not set")
	}

	var err error
	db, err = sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Test connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	fmt.Println("✓ Successfully connected to database")

	// Load maintenance records
	records, err := loadMaintenanceRecordsFromDB()
	if err != nil {
		log.Fatalf("Failed to load maintenance records: %v", err)
	}

	// Print summary
	fmt.Printf("\n📊 Loaded %d maintenance records\n", len(records))

	// Show first few records
	if len(records) > 0 {
		fmt.Println("\n📋 First few records:")
		fmt.Println("================================================================================")
		
		limit := 5
		if len(records) < limit {
			limit = len(records)
		}
		
		for i := 0; i < limit; i++ {
			record := records[i]
			fmt.Printf("\nRecord #%d (ID: %d):\n", i+1, record.ID)
			if record.VehicleNumber != nil {
				fmt.Printf("  Vehicle Number: %d\n", *record.VehicleNumber)
			}
			if record.VehicleID != nil {
				fmt.Printf("  Vehicle ID: %s\n", *record.VehicleID)
			}
			if record.ServiceDate != nil {
				fmt.Printf("  Service Date: %s\n", record.ServiceDate.Format("2006-01-02"))
			}
			if record.Mileage != nil {
				fmt.Printf("  Mileage: %d\n", *record.Mileage)
			}
			if record.PONumber != nil {
				fmt.Printf("  PO Number: %s\n", *record.PONumber)
			}
			if record.Cost != nil {
				fmt.Printf("  Cost: $%.2f\n", *record.Cost)
			}
			if record.WorkDescription != nil {
				fmt.Printf("  Work Description: %s\n", *record.WorkDescription)
			}
			if record.RawData != nil {
				fmt.Printf("  Raw Data: %s\n", *record.RawData)
			}
		}
	}

	// Check for any records with missing vehicle IDs
	missingVehicleCount := 0
	missingVehicleNumberCount := 0
	for _, record := range records {
		if record.VehicleID == nil || *record.VehicleID == "" {
			missingVehicleCount++
		}
		if record.VehicleNumber == nil {
			missingVehicleNumberCount++
		}
	}
	if missingVehicleCount > 0 {
		fmt.Printf("\n⚠️  Warning: %d records have missing vehicle IDs\n", missingVehicleCount)
	}
	if missingVehicleNumberCount > 0 {
		fmt.Printf("⚠️  Warning: %d records have missing vehicle numbers\n", missingVehicleNumberCount)
	}

	// Show cost distribution
	fmt.Println("\n💰 Cost Analysis:")
	recordsWithCost := 0
	totalCost := 0.0
	for _, record := range records {
		if record.Cost != nil {
			recordsWithCost++
			totalCost += *record.Cost
		}
	}
	fmt.Printf("  Records with cost: %d/%d\n", recordsWithCost, len(records))
	if recordsWithCost > 0 {
		fmt.Printf("  Total cost: $%.2f\n", totalCost)
		fmt.Printf("  Average cost: $%.2f\n", totalCost/float64(recordsWithCost))
	}

	// Show vehicles with most maintenance records
	fmt.Println("\n🚌 Top 5 Vehicles by Maintenance Count:")
	vehicleCounts := make(map[int]int)
	for _, record := range records {
		if record.VehicleNumber != nil {
			vehicleCounts[*record.VehicleNumber]++
		}
	}
	
	// Convert to slice for sorting
	type vehicleCount struct {
		VehicleID int
		Count     int
	}
	var vcSlice []vehicleCount
	for vid, count := range vehicleCounts {
		vcSlice = append(vcSlice, vehicleCount{VehicleID: vid, Count: count})
	}
	
	// Sort by count (descending)
	for i := 0; i < len(vcSlice)-1; i++ {
		for j := i + 1; j < len(vcSlice); j++ {
			if vcSlice[j].Count > vcSlice[i].Count {
				vcSlice[i], vcSlice[j] = vcSlice[j], vcSlice[i]
			}
		}
	}
	
	// Show top 5
	limit := 5
	if len(vcSlice) < limit {
		limit = len(vcSlice)
	}
	for i := 0; i < limit; i++ {
		fmt.Printf("  Vehicle ID %d: %d maintenance records\n", vcSlice[i].VehicleID, vcSlice[i].Count)
	}
}

// loadMaintenanceRecordsFromDB - matches actual database schema
func loadMaintenanceRecordsFromDB() ([]MaintenanceRecord, error) {
	query := `
        SELECT id, vehicle_number, service_date, mileage, po_number, 
               cost, work_description, raw_data, created_at, updated_at, vehicle_id, date
        FROM maintenance_records
        ORDER BY service_date DESC
    `
	
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query maintenance records: %v", err)
	}
	defer rows.Close()

	var records []MaintenanceRecord
	for rows.Next() {
		var r MaintenanceRecord
		err := rows.Scan(
			&r.ID, &r.VehicleNumber, &r.ServiceDate, &r.Mileage,
			&r.PONumber, &r.Cost, &r.WorkDescription, &r.RawData,
			&r.CreatedAt, &r.UpdatedAt, &r.VehicleID, &r.Date,
		)
		if err != nil {
			log.Printf("Error scanning maintenance record: %v", err)
			continue
		}
		records = append(records, r)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating maintenance records: %v", err)
	}

	return records, nil
}

