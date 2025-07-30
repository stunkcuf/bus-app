package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
	"github.com/jmoiron/sqlx"
)

type Vehicle struct {
	VehicleID        string         `json:"vehicle_id" db:"vehicle_id"`
	Model            sql.NullString `json:"model" db:"model"`
	Description      sql.NullString `json:"description" db:"description"`
	Year             sql.NullString `json:"year" db:"year"`
	TireSize         sql.NullString `json:"tire_size" db:"tire_size"`
	License          sql.NullString `json:"license" db:"license"`
	OilStatus        sql.NullString `json:"oil_status" db:"oil_status"`
	TireStatus       sql.NullString `json:"tire_status" db:"tire_status"`
	Status           sql.NullString `json:"status" db:"status"`
	MaintenanceNotes sql.NullString `json:"maintenance_notes" db:"maintenance_notes"`
	SerialNumber     sql.NullString `json:"serial_number" db:"serial_number"`
	Base             sql.NullString `json:"base" db:"base"`
	ServiceInterval  sql.NullInt32  `json:"service_interval" db:"service_interval"`
	CurrentMileage   sql.NullInt32  `json:"current_mileage" db:"current_mileage"`
	LastOilChange    sql.NullInt32  `json:"last_oil_change" db:"last_oil_change"`
	LastTireService  sql.NullInt32  `json:"last_tire_service" db:"last_tire_service"`
	UpdatedAt        sql.NullTime   `json:"updated_at" db:"updated_at"`
	CreatedAt        sql.NullTime   `json:"created_at" db:"created_at"`
	ImportID         sql.NullString `json:"import_id" db:"import_id"`
}

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgresql://postgres:jTKOWGEzlprRGbkPBCgGxsgnwyLeGoDL@shortline.proxy.rlwy.net:40148/railway"
	}

	db, err := sqlx.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer db.Close()

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	fmt.Println("Testing vehicles struct mapping...")

	// Test with explicit columns (no id)
	var vehicles []Vehicle
	start := time.Now()
	
	err = db.Select(&vehicles, `
		SELECT vehicle_id, model, description, year, tire_size, license, 
		       oil_status, tire_status, status, maintenance_notes, 
		       serial_number, base, service_interval, current_mileage, 
		       last_oil_change, last_tire_service, updated_at, created_at, import_id
		FROM vehicles 
		ORDER BY vehicle_id
		LIMIT 5
	`)
	
	elapsed := time.Since(start)
	
	if err != nil {
		fmt.Printf("❌ Query failed: %v\n", err)
		fmt.Printf("   Time taken: %v\n", elapsed)
		
		// Try with SELECT *
		fmt.Println("\nTrying SELECT * to see what happens...")
		err2 := db.Select(&vehicles, "SELECT * FROM vehicles LIMIT 1")
		if err2 != nil {
			fmt.Printf("❌ SELECT * also failed: %v\n", err2)
		}
		return
	}
	
	fmt.Printf("✅ Query successful in %v\n", elapsed)
	fmt.Printf("   Loaded %d vehicles\n", len(vehicles))
	
	// Show first vehicle
	if len(vehicles) > 0 {
		v := vehicles[0]
		fmt.Printf("\nFirst vehicle:\n")
		fmt.Printf("  VehicleID: %s\n", v.VehicleID)
		if v.Model.Valid {
			fmt.Printf("  Model: %s\n", v.Model.String)
		}
		if v.Status.Valid {
			fmt.Printf("  Status: %s\n", v.Status.String)
		}
	}
}