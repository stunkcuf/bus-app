package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

// MaintenanceRecord struct from models.go
type MaintenanceRecord struct {
	ID              int            `json:"id" db:"id"`
	VehicleNumber   sql.NullInt32  `json:"vehicle_number" db:"vehicle_number"`
	ServiceDate     sql.NullTime   `json:"service_date" db:"service_date"`
	Mileage         sql.NullInt32  `json:"mileage" db:"mileage"`
	PONumber        sql.NullString `json:"po_number" db:"po_number"`
	Cost            sql.NullString `json:"cost" db:"cost"`
	WorkDescription sql.NullString `json:"work_description" db:"work_description"`
	RawData         sql.NullString `json:"raw_data" db:"raw_data"`
	CreatedAt       time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at" db:"updated_at"`
	VehicleID       sql.NullString `json:"vehicle_id" db:"vehicle_id"`
	Date            sql.NullTime   `json:"date" db:"date"`
}

func (mr MaintenanceRecord) GetVehicleIdentifier() string {
	if mr.VehicleNumber.Valid && mr.VehicleNumber.Int32 > 0 {
		return fmt.Sprintf("Vehicle #%d", mr.VehicleNumber.Int32)
	}
	if mr.VehicleID.Valid && mr.VehicleID.String != "" {
		return mr.VehicleID.String
	}
	return fmt.Sprintf("Record #%d", mr.ID)
}

func (mr MaintenanceRecord) GetFormattedServiceDate() string {
	if mr.ServiceDate.Valid {
		return mr.ServiceDate.Time.Format("Jan 2, 2006")
	}
	if mr.Date.Valid {
		return mr.Date.Time.Format("Jan 2, 2006")
	}
	return "Unknown"
}

func (mr MaintenanceRecord) GetMileage() int {
	if mr.Mileage.Valid {
		return int(mr.Mileage.Int32)
	}
	return 0
}

func (mr MaintenanceRecord) GetCost() string {
	if mr.Cost.Valid {
		return mr.Cost.String
	}
	return ""
}

func (mr MaintenanceRecord) GetPONumber() string {
	if mr.PONumber.Valid {
		return mr.PONumber.String
	}
	return ""
}

func (mr MaintenanceRecord) GetWorkDescription() string {
	if mr.WorkDescription.Valid {
		return mr.WorkDescription.String
	}
	return ""
}

var db *sqlx.DB

func main() {
	// Load .env file
	godotenv.Load()

	// Initialize database
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		fmt.Println("DATABASE_URL not set")
		os.Exit(1)
	}

	var err error
	db, err = sqlx.Connect("postgres", connStr)
	if err != nil {
		log.Printf("Database connection failed: %v", err)
		os.Exit(1)
	}
	defer db.Close()

	fmt.Println("=== Testing Maintenance Handler Logic ===")

	// Simulate the maintenance records handler
	fmt.Println("\n1. Loading records...")
	records, err := loadMaintenanceRecordsFromDB()
	if err != nil {
		fmt.Printf("ERROR loading records: %v\n", err)
		return
	}
	fmt.Printf("Loaded %d total records\n", len(records))

	// Pagination setup (from handler)
	page := 1
	perPage := 25
	totalRecords := len(records)
	totalPages := (totalRecords + perPage - 1) / perPage

	fmt.Printf("Total pages: %d\n", totalPages)

	// Calculate pagination
	start := (page - 1) * perPage
	end := start + perPage
	if end > totalRecords {
		end = totalRecords
	}

	var paginatedRecords []MaintenanceRecord
	if start < totalRecords {
		paginatedRecords = records[start:end]
	}

	fmt.Printf("Paginated records: %d (start: %d, end: %d)\n", len(paginatedRecords), start, end)

	// Test template methods
	if len(paginatedRecords) > 0 {
		fmt.Println("\n2. Testing first 3 records:")
		for i, record := range paginatedRecords[:min(3, len(paginatedRecords))] {
			fmt.Printf("  Record %d:\n", i+1)
			fmt.Printf("    ID: %d\n", record.ID)
			fmt.Printf("    Vehicle Identifier: %s\n", record.GetVehicleIdentifier())
			fmt.Printf("    Service Date: %s\n", record.GetFormattedServiceDate())
			fmt.Printf("    Mileage: %d\n", record.GetMileage())
			fmt.Printf("    Cost: %s\n", record.GetCost())
			fmt.Printf("    Work Description: %s\n", record.GetWorkDescription())
		}
	} else {
		fmt.Println("\n2. No paginated records to display!")
	}

	fmt.Println("\n=== Test Complete ===")
}

func loadMaintenanceRecordsFromDB() ([]MaintenanceRecord, error) {
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var records []MaintenanceRecord
	query := `
		SELECT id, vehicle_number, service_date, mileage, po_number, cost, 
		       work_description, raw_data, created_at, updated_at, vehicle_id, date
		FROM maintenance_records 
		ORDER BY 
			COALESCE(service_date, date, created_at) DESC,
			vehicle_number, id`

	err := db.Select(&records, query)
	if err != nil {
		return nil, fmt.Errorf("failed to load maintenance records: %w", err)
	}

	return records, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}