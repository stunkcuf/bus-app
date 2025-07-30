package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgresql://postgres:jTKOWGEzlprRGbkPBCgGxsgnwyLeGoDL@shortline.proxy.rlwy.net:40148/railway"
		fmt.Println("Using hardcoded database URL for testing")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer db.Close()

	fmt.Println("🧪 Testing data loading functions that webpages use...")

	// Test 1: Maintenance Records Loading
	fmt.Println("\n1. Testing Maintenance Records Query:")
	fmt.Println("=====================================")
	
	maintenanceQuery := `
		SELECT id, vehicle_number, service_date, mileage, po_number, cost, 
		       work_description, raw_data, created_at, updated_at, vehicle_id, date
		FROM maintenance_records 
		ORDER BY 
			COALESCE(service_date, date, created_at) DESC,
			vehicle_number, id
		LIMIT 5`
	
	rows, err := db.Query(maintenanceQuery)
	if err != nil {
		fmt.Printf("❌ MAINTENANCE QUERY FAILED: %v\n", err)
	} else {
		defer rows.Close()
		count := 0
		for rows.Next() {
			var id, vehicleNumber, mileage int
			var poNumber, workDescription, rawData, vehicleID string
			var serviceDate, date, createdAt, updatedAt sql.NullTime
			var cost sql.NullFloat64
			
			err := rows.Scan(&id, &vehicleNumber, &serviceDate, &mileage, &poNumber, 
				&cost, &workDescription, &rawData, &createdAt, &updatedAt, &vehicleID, &date)
			if err != nil {
				fmt.Printf("❌ Row scan error: %v\n", err)
				break
			}
			count++
			
			// Show first record details
			if count == 1 {
				fmt.Printf("✅ First Record: ID=%d, Vehicle=%d, VehicleID=%s\n", id, vehicleNumber, vehicleID)
				if serviceDate.Valid {
					fmt.Printf("   Service Date: %v\n", serviceDate.Time.Format("2006-01-02"))
				} else {
					fmt.Printf("   Service Date: NULL\n")
				}
				fmt.Printf("   Work: %s\n", workDescription)
			}
		}
		fmt.Printf("✅ MAINTENANCE QUERY SUCCESS: %d records found\n", count)
	}

	// Test 2: Service Records Loading  
	fmt.Println("\n2. Testing Service Records Query:")
	fmt.Println("==================================")
	
	serviceQuery := `
		SELECT id, unnamed_0, unnamed_1, unnamed_2, unnamed_3, unnamed_4, unnamed_5, 
		       unnamed_6, unnamed_7, unnamed_8, unnamed_9, unnamed_10, unnamed_11, 
		       unnamed_12, unnamed_13, created_at, updated_at, maintenance_date
		FROM service_records 
		ORDER BY 
			COALESCE(maintenance_date, created_at) DESC,
			id
		LIMIT 5`
	
	rows2, err := db.Query(serviceQuery)
	if err != nil {
		fmt.Printf("❌ SERVICE QUERY FAILED: %v\n", err)
	} else {
		defer rows2.Close()
		count := 0
		for rows2.Next() {
			var id int
			var unnamed0, unnamed1, unnamed2, unnamed3, unnamed4, unnamed5 sql.NullString
			var unnamed6, unnamed7, unnamed8, unnamed9, unnamed10, unnamed11 sql.NullString 
			var unnamed12, unnamed13 sql.NullString
			var createdAt, updatedAt, maintenanceDate sql.NullTime
			
			err := rows2.Scan(&id, &unnamed0, &unnamed1, &unnamed2, &unnamed3, &unnamed4, 
				&unnamed5, &unnamed6, &unnamed7, &unnamed8, &unnamed9, &unnamed10, 
				&unnamed11, &unnamed12, &unnamed13, &createdAt, &updatedAt, &maintenanceDate)
			if err != nil {
				fmt.Printf("❌ Row scan error: %v\n", err)
				break
			}
			count++
			
			// Show first record details
			if count == 1 {
				fmt.Printf("✅ First Record: ID=%d\n", id)
				if maintenanceDate.Valid {
					fmt.Printf("   Maintenance Date: %v\n", maintenanceDate.Time.Format("2006-01-02"))
				} else {
					fmt.Printf("   Maintenance Date: NULL\n")
				}
				fmt.Printf("   Data: %s, %s, %s\n", 
					getStringValue(unnamed0), getStringValue(unnamed1), getStringValue(unnamed2))
			}
		}
		fmt.Printf("✅ SERVICE QUERY SUCCESS: %d records found\n", count)
	}

	// Test 3: Vehicle Loading (the one that works vs doesn't work)
	fmt.Println("\n3. Testing Vehicle Loading:")
	fmt.Println("============================")
	
	// Test buses (this works)
	var busCount int
	err = db.QueryRow("SELECT COUNT(*) FROM buses").Scan(&busCount)
	if err != nil {
		fmt.Printf("❌ Bus count failed: %v\n", err)
	} else {
		fmt.Printf("✅ Bus count: %d\n", busCount)
	}
	
	// Test vehicles (this might be failing)
	var vehicleCount int  
	err = db.QueryRow("SELECT COUNT(*) FROM vehicles").Scan(&vehicleCount)
	if err != nil {
		fmt.Printf("❌ Vehicle count failed: %v\n", err)
	} else {
		fmt.Printf("✅ Vehicle count: %d\n", vehicleCount)
	}
	
	// Test vehicle query that might be failing
	vehicleQuery := `SELECT vehicle_id, model, description, year FROM vehicles LIMIT 3`
	rows3, err := db.Query(vehicleQuery)
	if err != nil {
		fmt.Printf("❌ VEHICLE QUERY FAILED: %v\n", err)
	} else {
		defer rows3.Close()
		count := 0
		for rows3.Next() {
			var vehicleID, model, description sql.NullString
			var year sql.NullString // This might be the problem - "nan" values
			
			err := rows3.Scan(&vehicleID, &model, &description, &year)
			if err != nil {
				fmt.Printf("❌ Vehicle row scan error: %v\n", err)
				break
			}
			count++
			
			fmt.Printf("   Vehicle: %s, Model: %s, Year: %s\n", 
				getStringValue(vehicleID), getStringValue(model), getStringValue(year))
		}
		fmt.Printf("✅ VEHICLE DETAIL QUERY: %d records scanned\n", count)
	}

	// Test 4: Route assignments (driver dashboard issue)
	fmt.Println("\n4. Testing Route Assignments:")
	fmt.Println("==============================")
	
	var routeAssignmentCount int
	err = db.QueryRow("SELECT COUNT(*) FROM route_assignments").Scan(&routeAssignmentCount)
	if err != nil {
		fmt.Printf("❌ Route assignment count failed: %v\n", err)
	} else {
		fmt.Printf("✅ Route assignments: %d\n", routeAssignmentCount)
		
		// Show actual assignments
		assignmentQuery := `SELECT driver, bus_id, route_id, route_name FROM route_assignments LIMIT 5`
		rows4, err := db.Query(assignmentQuery)
		if err != nil {
			fmt.Printf("❌ Assignment detail query failed: %v\n", err)
		} else {
			defer rows4.Close()
			for rows4.Next() {
				var driver, busID, routeID, routeName string
				err := rows4.Scan(&driver, &busID, &routeID, &routeName)
				if err == nil {
					fmt.Printf("   - Driver: %s, Bus: %s, Route: %s (%s)\n", 
						driver, busID, routeID, routeName)
				}
			}
		}
	}

	fmt.Println("\n🏁 Data loading test completed!")
	fmt.Println("\nIf queries work here but not in webpages, the issue is likely:")
	fmt.Println("1. Template rendering problems")
	fmt.Println("2. Handler error handling masking issues") 
	fmt.Println("3. Data structure marshaling problems")
	fmt.Println("4. JavaScript errors hiding content")
}

func getStringValue(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return "NULL"
}