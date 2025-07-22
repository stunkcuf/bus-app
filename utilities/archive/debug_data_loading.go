package main

import (
	"fmt"
	"log"
	"os"

	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	// Load .env file
	godotenv.Load()

	// Initialize database
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		fmt.Println("DATABASE_URL not set")
		os.Exit(1)
	}

	db, err := sqlx.Connect("postgres", connStr)
	if err != nil {
		log.Printf("Database connection failed: %v", err)
		os.Exit(1)
	}
	defer db.Close()

	fmt.Println("=== Database Data Loading Debug ===")
	
	// Check buses count
	var busCount int
	err = db.Get(&busCount, "SELECT COUNT(*) FROM buses")
	if err != nil {
		fmt.Printf("Error counting buses: %v\n", err)
	} else {
		fmt.Printf("Buses count: %d\n", busCount)
	}

	// Check vehicles count
	var vehicleCount int
	err = db.Get(&vehicleCount, "SELECT COUNT(*) FROM vehicles")
	if err != nil {
		fmt.Printf("Error counting vehicles: %v\n", err)
	} else {
		fmt.Printf("Vehicles count: %d\n", vehicleCount)
	}

	// Check maintenance_records count
	var maintCount int
	err = db.Get(&maintCount, "SELECT COUNT(*) FROM maintenance_records")
	if err != nil {
		fmt.Printf("Error counting maintenance_records: %v\n", err)
	} else {
		fmt.Printf("Maintenance records count: %d\n", maintCount)
	}

	// Check service_records count
	var serviceCount int
	err = db.Get(&serviceCount, "SELECT COUNT(*) FROM service_records")
	if err != nil {
		fmt.Printf("Error counting service_records: %v\n", err)
	} else {
		fmt.Printf("Service records count: %d\n", serviceCount)
	}

	// Check routes count
	var routeCount int
	err = db.Get(&routeCount, "SELECT COUNT(*) FROM routes")
	if err != nil {
		fmt.Printf("Error counting routes: %v\n", err)
	} else {
		fmt.Printf("Routes count: %d\n", routeCount)
	}

	// Check route_assignments count
	var assignmentCount int
	err = db.Get(&assignmentCount, "SELECT COUNT(*) FROM route_assignments")
	if err != nil {
		fmt.Printf("Error counting route_assignments: %v\n", err)
	} else {
		fmt.Printf("Route assignments count: %d\n", assignmentCount)
	}

	fmt.Println("\n=== Testing Data Loading Functions ===")

	// Test loading maintenance records
	var maintenanceRecords []struct {
		ID            int    `db:"id"`
		VehicleNumber *int   `db:"vehicle_number"`
		VehicleID     string `db:"vehicle_id"`
	}
	
	query := `
		SELECT id, vehicle_number, vehicle_id
		FROM maintenance_records 
		ORDER BY id DESC
		LIMIT 10`
	
	err = db.Select(&maintenanceRecords, query)
	if err != nil {
		fmt.Printf("Error loading maintenance records: %v\n", err)
	} else {
		fmt.Printf("Successfully loaded %d maintenance records (showing first 10)\n", len(maintenanceRecords))
		for i, record := range maintenanceRecords {
			vn := "NULL"
			if record.VehicleNumber != nil {
				vn = fmt.Sprintf("%d", *record.VehicleNumber)
			}
			fmt.Printf("  Record %d: ID=%d, VehicleNumber=%s, VehicleID=%s\n", i+1, record.ID, vn, record.VehicleID)
		}
	}

	// Test loading service records
	var serviceRecords []struct {
		ID              int    `db:"id"`
		MaintenanceDate string `db:"maintenance_date"`
	}
	
	serviceQuery := `
		SELECT id, maintenance_date
		FROM service_records 
		ORDER BY id DESC
		LIMIT 10`
	
	err = db.Select(&serviceRecords, serviceQuery)
	if err != nil {
		fmt.Printf("Error loading service records: %v\n", err)
	} else {
		fmt.Printf("Successfully loaded %d service records (showing first 10)\n", len(serviceRecords))
		for i, record := range serviceRecords {
			fmt.Printf("  Record %d: ID=%d, MaintenanceDate=%s\n", i+1, record.ID, record.MaintenanceDate)
		}
	}

	// Test vehicles loading
	var vehicles []struct {
		VehicleID string `db:"vehicle_id"`
		Model     string `db:"model"`
		Status    *string `db:"status"`
	}
	
	vehicleQuery := `
		SELECT vehicle_id, model, status
		FROM vehicles 
		ORDER BY vehicle_id DESC
		LIMIT 10`
	
	err = db.Select(&vehicles, vehicleQuery)
	if err != nil {
		fmt.Printf("Error loading vehicles: %v\n", err)
	} else {
		fmt.Printf("Successfully loaded %d vehicles (showing first 10)\n", len(vehicles))
		for i, vehicle := range vehicles {
			status := "NULL"
			if vehicle.Status != nil {
				status = *vehicle.Status
			}
			fmt.Printf("  Vehicle %d: ID=%s, Model=%s, Status=%s\n", i+1, vehicle.VehicleID, vehicle.Model, status)
		}
	}

	// Test routes loading
	var routes []struct {
		RouteID   string `db:"route_id"`
		RouteName string `db:"route_name"`
	}
	
	routeQuery := `
		SELECT route_id, route_name
		FROM routes 
		ORDER BY route_id DESC
		LIMIT 10`
	
	err = db.Select(&routes, routeQuery)
	if err != nil {
		fmt.Printf("Error loading routes: %v\n", err)
	} else {
		fmt.Printf("Successfully loaded %d routes (showing first 10)\n", len(routes))
		for i, route := range routes {
			fmt.Printf("  Route %d: ID=%s, Name=%s\n", i+1, route.RouteID, route.RouteName)
		}
	}

	fmt.Println("\n=== Debug Complete ===")
}