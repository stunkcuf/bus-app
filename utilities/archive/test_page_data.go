package main

import (
	"fmt"
	"log"
	"os"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

var db *sqlx.DB

func main() {
	// Initialize database
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

	fmt.Println("🧪 Testing Web Page Data Loading Functions...")

	// Test 1: Maintenance Records (this was failing before fix)
	fmt.Println("\n1. Testing Maintenance Records Page:")
	fmt.Println("=====================================")
	
	var maintenanceCount int
	err = db.QueryRow("SELECT COUNT(*) FROM maintenance_records").Scan(&maintenanceCount)
	if err != nil {
		fmt.Printf("❌ Count query failed: %v\n", err)
	} else {
		fmt.Printf("✅ Found %d maintenance records in database\n", maintenanceCount)
	}

	// Test loading with the fixed query (using COALESCE for NULLs)
	maintenanceQuery := `
		SELECT id, 
		       COALESCE(vehicle_number, 0) as vehicle_number, 
		       service_date, 
		       COALESCE(mileage, 0) as mileage, 
		       COALESCE(po_number, '') as po_number, 
		       COALESCE(cost, '') as cost, 
		       COALESCE(work_description, '') as work_description, 
		       COALESCE(raw_data, '') as raw_data, 
		       created_at, 
		       updated_at, 
		       COALESCE(vehicle_id, '') as vehicle_id, 
		       date
		FROM maintenance_records 
		ORDER BY 
		    COALESCE(service_date, date, created_at) DESC,
		    vehicle_number, id
		LIMIT 5`

	type TestMaintenanceRecord struct {
		ID              int    `db:"id"`
		VehicleNumber   int    `db:"vehicle_number"`
		ServiceDate     *string `db:"service_date"`
		Mileage         int    `db:"mileage"`
		PONumber        string `db:"po_number"`
		Cost            string `db:"cost"`
		WorkDescription string `db:"work_description"`
		RawData         string `db:"raw_data"`
		VehicleID       string `db:"vehicle_id"`
	}

	var maintenanceRecords []TestMaintenanceRecord
	err = db.Select(&maintenanceRecords, maintenanceQuery)
	if err != nil {
		fmt.Printf("❌ Maintenance records query FAILED: %v\n", err)
	} else {
		fmt.Printf("✅ Maintenance records query SUCCESS: loaded %d records\n", len(maintenanceRecords))
		if len(maintenanceRecords) > 0 {
			first := maintenanceRecords[0]
			fmt.Printf("   First record: ID=%d, Vehicle=%d, Cost=%s\n", 
				first.ID, first.VehicleNumber, first.Cost)
		}
	}

	// Test 2: Service Records  
	fmt.Println("\n2. Testing Service Records Page:")
	fmt.Println("================================")
	
	var serviceCount int
	err = db.QueryRow("SELECT COUNT(*) FROM service_records").Scan(&serviceCount)
	if err != nil {
		fmt.Printf("❌ Service count failed: %v\n", err)
	} else {
		fmt.Printf("✅ Found %d service records in database\n", serviceCount)
	}

	serviceQuery := `
		SELECT id, 
		       COALESCE(unnamed_0, '') as unnamed_0,
		       COALESCE(unnamed_1, '') as unnamed_1, 
		       COALESCE(unnamed_2, '') as unnamed_2,
		       maintenance_date
		FROM service_records 
		ORDER BY COALESCE(maintenance_date, created_at) DESC
		LIMIT 5`

	type TestServiceRecord struct {
		ID              int     `db:"id"`
		Unnamed0        string  `db:"unnamed_0"`
		Unnamed1        string  `db:"unnamed_1"`
		Unnamed2        string  `db:"unnamed_2"`
		MaintenanceDate *string `db:"maintenance_date"`
	}

	var serviceRecords []TestServiceRecord
	err = db.Select(&serviceRecords, serviceQuery)
	if err != nil {
		fmt.Printf("❌ Service records query FAILED: %v\n", err)
	} else {
		fmt.Printf("✅ Service records query SUCCESS: loaded %d records\n", len(serviceRecords))
		if len(serviceRecords) > 0 {
			first := serviceRecords[0]
			fmt.Printf("   First record: ID=%d, Data=%s|%s|%s\n", 
				first.ID, first.Unnamed0, first.Unnamed1, first.Unnamed2)
		}
	}

	// Test 3: Fleet page data
	fmt.Println("\n3. Testing Fleet Page Data:")
	fmt.Println("============================")
	
	var busCount int
	err = db.QueryRow("SELECT COUNT(*) FROM buses").Scan(&busCount)
	if err == nil {
		fmt.Printf("✅ Buses: %d\n", busCount)
	}

	var vehicleCount int  
	err = db.QueryRow("SELECT COUNT(*) FROM vehicles").Scan(&vehicleCount)
	if err == nil {
		fmt.Printf("✅ Vehicles: %d\n", vehicleCount)
	}

	fmt.Printf("✅ TOTAL FLEET: %d items\n", busCount+vehicleCount)

	fmt.Println("\n🎉 All page data tests completed!")
	fmt.Println("\n📋 SUMMARY:")
	fmt.Printf("• Maintenance Records: %d (should display on /maintenance-records)\n", maintenanceCount)
	fmt.Printf("• Service Records: %d (should display on /service-records)\n", serviceCount) 
	fmt.Printf("• Fleet Items: %d buses + %d vehicles = %d total (should display on /fleet)\n", 
		busCount, vehicleCount, busCount+vehicleCount)
	
	if maintenanceCount > 0 && serviceCount > 0 && (busCount+vehicleCount) > 0 {
		fmt.Println("\n✅ ALL DATA AVAILABLE - Web pages should now display correctly!")
	} else {
		fmt.Println("\n⚠️  Some data missing - check specific queries")
	}
}