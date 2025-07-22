package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// MaintenanceRecord struct - matches actual database schema
type MaintenanceRecord struct {
	ID              int            `json:"id" db:"id"`
	VehicleNumber   sql.NullInt32  `json:"vehicle_number" db:"vehicle_number"`
	ServiceDate     sql.NullTime   `json:"service_date" db:"service_date"`
	Mileage         sql.NullInt32  `json:"mileage" db:"mileage"`
	PONumber        sql.NullString `json:"po_number" db:"po_number"`
	Cost            sql.NullString `json:"cost" db:"cost"` // Stored as string in DB due to varying formats
	WorkDescription sql.NullString `json:"work_description" db:"work_description"`
	RawData         sql.NullString `json:"raw_data" db:"raw_data"`
	CreatedAt       time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at" db:"updated_at"`
	VehicleID       sql.NullString `json:"vehicle_id" db:"vehicle_id"`
	Date            sql.NullTime   `json:"date" db:"date"`
}

var db *sqlx.DB

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
	db, err = sqlx.Connect("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	fmt.Println("✓ Successfully connected to database")

	// Test 1: Load records using the same function as the handler
	records, err := loadMaintenanceRecordsFromDB()
	if err != nil {
		log.Fatalf("Failed to load maintenance records: %v", err)
	}

	fmt.Printf("\n📊 Loaded %d maintenance records using db.Select\n", len(records))

	// Test 2: Check if the records have the expected data
	if len(records) > 0 {
		fmt.Println("\n🔍 Sample record data:")
		record := records[0]
		fmt.Printf("  ID: %d\n", record.ID)
		if record.VehicleNumber.Valid {
			fmt.Printf("  Vehicle Number: %d\n", record.VehicleNumber.Int32)
		}
		if record.ServiceDate.Valid {
			fmt.Printf("  Service Date: %s\n", record.ServiceDate.Time.Format("2006-01-02"))
		}
		if record.WorkDescription.Valid {
			fmt.Printf("  Work Description: %s\n", record.WorkDescription.String)
		}
	}

	// Test 3: Simulate pagination like the handler does
	page := 1
	perPage := 25
	totalRecords := len(records)
	totalPages := (totalRecords + perPage - 1) / perPage

	start := (page - 1) * perPage
	end := start + perPage
	if end > totalRecords {
		end = totalRecords
	}

	var paginatedRecords []MaintenanceRecord
	if start < totalRecords {
		paginatedRecords = records[start:end]
	}

	fmt.Printf("\n📄 Pagination test:\n")
	fmt.Printf("  Total records: %d\n", totalRecords)
	fmt.Printf("  Total pages: %d\n", totalPages)
	fmt.Printf("  Current page: %d\n", page)
	fmt.Printf("  Records on this page: %d\n", len(paginatedRecords))
	fmt.Printf("  Showing records %d-%d\n", start+1, end)

	// Test 4: Check for any nil/empty data issues
	emptyDescCount := 0
	nilVehicleCount := 0
	for _, r := range records {
		if !r.WorkDescription.Valid || r.WorkDescription.String == "" {
			emptyDescCount++
		}
		if !r.VehicleNumber.Valid {
			nilVehicleCount++
		}
	}

	fmt.Printf("\n⚠️  Data quality check:\n")
	fmt.Printf("  Records with empty work description: %d\n", emptyDescCount)
	fmt.Printf("  Records with nil vehicle number: %d\n", nilVehicleCount)

	// Test 5: Try a simple query to verify data
	var count int
	err = db.Get(&count, "SELECT COUNT(*) FROM maintenance_records")
	if err != nil {
		log.Printf("Failed to count records: %v", err)
	} else {
		fmt.Printf("\n✓ Direct SQL count: %d records\n", count)
	}
}

// loadMaintenanceRecordsFromDB - using the exact function from data.go
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