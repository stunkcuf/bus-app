package main

import (
	"fmt"
	"log"
	"os"

	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

// Vehicle struct from models.go
type Vehicle struct {
	VehicleID        string    `json:"vehicle_id" db:"vehicle_id"`
	Model            string    `json:"model" db:"model"`
	Description      string    `json:"description" db:"description"`
	Year             int       `json:"year" db:"year"`
	TireSize         string    `json:"tire_size" db:"tire_size"`
	License          string    `json:"license" db:"license"`
	OilStatus        string    `json:"oil_status" db:"oil_status"`
	TireStatus       string    `json:"tire_status" db:"tire_status"`
	Status           *string   `json:"status" db:"status"`
	MaintenanceNotes string    `json:"maintenance_notes" db:"maintenance_notes"`
	SerialNumber     string    `json:"serial_number" db:"serial_number"`
	Base             string    `json:"base" db:"base"`
	ServiceInterval  int       `json:"service_interval" db:"service_interval"`
	CurrentMileage   *float64  `json:"current_mileage" db:"current_mileage"`
	LastOilChange    *string   `json:"last_oil_change" db:"last_oil_change"`
	LastTireService  *string   `json:"last_tire_service" db:"last_tire_service"`
	UpdatedAt        string    `json:"updated_at" db:"updated_at"`
	CreatedAt        string    `json:"created_at" db:"created_at"`
	ImportID         *string   `json:"import_id" db:"import_id"`
}

// MaintenanceRecord struct from models.go
type MaintenanceRecord struct {
	ID              int      `json:"id" db:"id"`
	VehicleNumber   *int     `json:"vehicle_number" db:"vehicle_number"`
	ServiceDate     *string  `json:"service_date" db:"service_date"`
	Mileage         *int     `json:"mileage" db:"mileage"`
	PONumber        *string  `json:"po_number" db:"po_number"`
	Cost            *float64 `json:"cost" db:"cost"`
	WorkDescription *string  `json:"work_description" db:"work_description"`
	RawData         *string  `json:"raw_data" db:"raw_data"`
	CreatedAt       string   `json:"created_at" db:"created_at"`
	UpdatedAt       string   `json:"updated_at" db:"updated_at"`
	VehicleID       string   `json:"vehicle_id" db:"vehicle_id"`
	Date            *string  `json:"date" db:"date"`
}

// ServiceRecord struct from models.go
type ServiceRecord struct {
	ID              int    `json:"id" db:"id"`
	Unnamed0        string `json:"unnamed_0" db:"unnamed_0"`
	Unnamed1        string `json:"unnamed_1" db:"unnamed_1"`
	Unnamed2        string `json:"unnamed_2" db:"unnamed_2"`
	Unnamed3        string `json:"unnamed_3" db:"unnamed_3"`
	Unnamed4        string `json:"unnamed_4" db:"unnamed_4"`
	Unnamed5        string `json:"unnamed_5" db:"unnamed_5"`
	Unnamed6        string `json:"unnamed_6" db:"unnamed_6"`
	Unnamed7        string `json:"unnamed_7" db:"unnamed_7"`
	Unnamed8        string `json:"unnamed_8" db:"unnamed_8"`
	Unnamed9        string `json:"unnamed_9" db:"unnamed_9"`
	Unnamed10       string `json:"unnamed_10" db:"unnamed_10"`
	Unnamed11       string `json:"unnamed_11" db:"unnamed_11"`
	Unnamed12       string `json:"unnamed_12" db:"unnamed_12"`
	Unnamed13       string `json:"unnamed_13" db:"unnamed_13"`
	CreatedAt       string `json:"created_at" db:"created_at"`
	UpdatedAt       string `json:"updated_at" db:"updated_at"`
	MaintenanceDate string `json:"maintenance_date" db:"maintenance_date"`
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

	fmt.Println("=== Testing Handler Functions ===")

	// Test loadVehiclesFromDBPaginated (similar to what fleet handler uses)
	fmt.Println("\n1. Testing loadVehiclesFromDBPaginated...")
	vehicles, err := loadVehiclesFromDBPaginated(PaginationParams{
		Page:       1,
		PerPage:    50,
		Offset:     0,
	})
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
	} else {
		fmt.Printf("SUCCESS: Loaded %d vehicles\n", len(vehicles))
		for i, v := range vehicles[:min(5, len(vehicles))] {
			status := "NULL"
			if v.Status != nil {
				status = *v.Status
			}
			fmt.Printf("  Vehicle %d: ID=%s, Model=%s, Status=%s\n", i+1, v.VehicleID, v.Model, status)
		}
	}

	// Test loadMaintenanceRecordsFromDB
	fmt.Println("\n2. Testing loadMaintenanceRecordsFromDB...")
	maintenanceRecords, err := loadMaintenanceRecordsFromDB()
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
	} else {
		fmt.Printf("SUCCESS: Loaded %d maintenance records\n", len(maintenanceRecords))
		for i, r := range maintenanceRecords[:min(5, len(maintenanceRecords))] {
			vn := "NULL"
			if r.VehicleNumber != nil {
				vn = fmt.Sprintf("%d", *r.VehicleNumber)
			}
			fmt.Printf("  Record %d: ID=%d, VehicleNumber=%s, VehicleID=%s\n", i+1, r.ID, vn, r.VehicleID)
		}
	}

	// Test loadServiceRecordsFromDB
	fmt.Println("\n3. Testing loadServiceRecordsFromDB...")
	serviceRecords, err := loadServiceRecordsFromDB()
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
	} else {
		fmt.Printf("SUCCESS: Loaded %d service records\n", len(serviceRecords))
		for i, r := range serviceRecords[:min(5, len(serviceRecords))] {
			fmt.Printf("  Record %d: ID=%d, MaintenanceDate=%s\n", i+1, r.ID, r.MaintenanceDate)
		}
	}

	// Test getRouteAssignments
	fmt.Println("\n4. Testing getRouteAssignments...")
	assignments, err := getRouteAssignments()
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
	} else {
		fmt.Printf("SUCCESS: Loaded %d route assignments\n", len(assignments))
		for i, a := range assignments {
			fmt.Printf("  Assignment %d: Driver=%s, BusID=%s, RouteID=%s, RouteName=%s\n", i+1, a.Driver, a.BusID, a.RouteID, a.RouteName)
		}
	}

	fmt.Println("\n=== Test Complete ===")
}

type PaginationParams struct {
	Page       int
	PerPage    int
	Offset     int
}

type RouteAssignment struct {
	Driver       string `json:"driver" db:"driver"`
	BusID        string `json:"bus_id" db:"bus_id"`
	RouteID      string `json:"route_id" db:"route_id"`
	RouteName    string `json:"route_name" db:"route_name"`
	AssignedDate string `json:"assigned_date" db:"assigned_date"`
}

func loadVehiclesFromDBPaginated(pagination PaginationParams) ([]Vehicle, error) {
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	query := fmt.Sprintf(`
		SELECT vehicle_id, model, description, year, tire_size, license, 
		       oil_status, tire_status, status, maintenance_notes, 
		       serial_number, base, service_interval, current_mileage, 
		       last_oil_change, last_tire_service, updated_at, created_at, import_id
		FROM vehicles 
		ORDER BY vehicle_id
		LIMIT %d OFFSET %d
	`, pagination.PerPage, pagination.Offset)

	var vehicles []Vehicle
	err := db.Select(&vehicles, query)
	if err != nil {
		return nil, fmt.Errorf("failed to load vehicles: %w", err)
	}

	return vehicles, nil
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

func getRouteAssignments() ([]RouteAssignment, error) {
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	query := `
		SELECT ra.driver, ra.bus_id, ra.route_id, r.route_name, ra.assigned_date
		FROM route_assignments ra
		JOIN routes r ON ra.route_id = r.route_id
		ORDER BY r.route_name
	`

	var assignments []RouteAssignment
	err := db.Select(&assignments, query)
	return assignments, err
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}