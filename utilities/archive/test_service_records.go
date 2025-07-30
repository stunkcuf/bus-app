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

// ServiceRecord struct from models.go - correct version
type ServiceRecord struct {
	ID              int            `json:"id" db:"id"`
	Unnamed0        sql.NullString `json:"unnamed_0" db:"unnamed_0"`
	Unnamed1        sql.NullString `json:"unnamed_1" db:"unnamed_1"`
	Unnamed2        sql.NullString `json:"unnamed_2" db:"unnamed_2"`
	Unnamed3        sql.NullString `json:"unnamed_3" db:"unnamed_3"`
	Unnamed4        sql.NullString `json:"unnamed_4" db:"unnamed_4"`
	Unnamed5        sql.NullString `json:"unnamed_5" db:"unnamed_5"`
	Unnamed6        sql.NullString `json:"unnamed_6" db:"unnamed_6"`
	Unnamed7        sql.NullString `json:"unnamed_7" db:"unnamed_7"`
	Unnamed8        sql.NullString `json:"unnamed_8" db:"unnamed_8"`
	Unnamed9        sql.NullString `json:"unnamed_9" db:"unnamed_9"`
	Unnamed10       sql.NullString `json:"unnamed_10" db:"unnamed_10"`
	Unnamed11       sql.NullString `json:"unnamed_11" db:"unnamed_11"`
	Unnamed12       sql.NullString `json:"unnamed_12" db:"unnamed_12"`
	Unnamed13       sql.NullString `json:"unnamed_13" db:"unnamed_13"`
	CreatedAt       time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at" db:"updated_at"`
	MaintenanceDate sql.NullTime   `json:"maintenance_date" db:"maintenance_date"`
}

func (sr ServiceRecord) GetMaintenanceDate() string {
	if sr.MaintenanceDate.Valid {
		return sr.MaintenanceDate.Time.Format("2006-01-02")
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

	fmt.Println("=== Testing Service Records Fix ===")

	// Test loadServiceRecordsFromDB with corrected struct
	fmt.Println("\n1. Testing loadServiceRecordsFromDB...")
	records, err := loadServiceRecordsFromDB()
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
	} else {
		fmt.Printf("SUCCESS: Loaded %d service records\n", len(records))
		for i, r := range records[:min(5, len(records))] {
			fmt.Printf("  Record %d: ID=%d, MaintenanceDate=%s\n", i+1, r.ID, r.GetMaintenanceDate())
		}
	}

	fmt.Println("\n=== Test Complete ===")
}

func loadServiceRecordsFromDB() ([]ServiceRecord, error) {
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var records []ServiceRecord
	query := `
		SELECT id, unnamed_0, unnamed_1, unnamed_2, unnamed_3, unnamed_4, unnamed_5, 
		       unnamed_6, unnamed_7, unnamed_8, unnamed_9, unnamed_10, unnamed_11, 
		       unnamed_12, unnamed_13, created_at, updated_at, maintenance_date
		FROM service_records 
		ORDER BY 
			COALESCE(maintenance_date, created_at) DESC,
			id`

	err := db.Select(&records, query)
	if err != nil {
		return nil, fmt.Errorf("failed to load service records: %w", err)
	}

	return records, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}