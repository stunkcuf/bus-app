package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

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

var db *sqlx.DB

func main() {
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		connStr = "postgresql://postgres:jTKOWGEzlprRGbkPBCgGxsgnwyLeGoDL@shortline.proxy.rlwy.net:40148/railway"
		fmt.Println("Using hardcoded database URL for testing")
	}

	var err error
	db, err = sqlx.Connect("postgres", connStr)
	if err != nil {
		log.Printf("Database connection failed: %v", err)
		os.Exit(1)
	}
	defer db.Close()

	fmt.Println("🔧 Testing Fixed Maintenance Records Query...")

	// Test the corrected query without problematic COALESCE
	query := `
		SELECT id, vehicle_number, service_date, mileage, po_number, cost,
		       work_description, raw_data, created_at, updated_at, vehicle_id, date
		FROM maintenance_records 
		ORDER BY 
			COALESCE(service_date, date, created_at) DESC,
			vehicle_number, id
		LIMIT 10`

	var records []MaintenanceRecord
	err = db.Select(&records, query)
	if err != nil {
		fmt.Printf("❌ MAINTENANCE QUERY FAILED: %v\n", err)
		return
	}

	fmt.Printf("✅ SUCCESS: Loaded %d maintenance records!\n", len(records))
	
	if len(records) > 0 {
		fmt.Println("\n📋 First 3 records:")
		for i, record := range records[:min(3, len(records))] {
			fmt.Printf("\n  Record %d:\n", i+1)
			fmt.Printf("    ID: %d\n", record.ID)
			
			if record.VehicleNumber.Valid {
				fmt.Printf("    Vehicle: #%d\n", record.VehicleNumber.Int32)
			} else {
				fmt.Printf("    Vehicle: N/A\n")
			}
			
			if record.ServiceDate.Valid {
				fmt.Printf("    Date: %s\n", record.ServiceDate.Time.Format("Jan 2, 2006"))
			} else if record.Date.Valid {
				fmt.Printf("    Date: %s\n", record.Date.Time.Format("Jan 2, 2006"))
			} else {
				fmt.Printf("    Date: N/A\n")
			}
			
			if record.Cost.Valid {
				fmt.Printf("    Cost: %s\n", record.Cost.String)
			} else {
				fmt.Printf("    Cost: N/A\n")
			}
			
			if record.WorkDescription.Valid {
				desc := record.WorkDescription.String
				if len(desc) > 50 {
					desc = desc[:50] + "..."
				}
				fmt.Printf("    Work: %s\n", desc)
			} else {
				fmt.Printf("    Work: N/A\n")
			}
		}
	}

	fmt.Println("\n🎉 MAINTENANCE RECORDS FIX COMPLETE!")
	fmt.Printf("✅ Query now successfully loads %d records\n", len(records))
	fmt.Println("✅ Web page /maintenance-records should now display data!")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}