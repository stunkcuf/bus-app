package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found: %v", err)
	}

	// Get database connection string
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable is not set")
	}

	// Connect using sqlx (same as the app)
	db, err := sqlx.Connect("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	fmt.Println("Testing Fleet Page Data Flow")
	fmt.Println("============================")

	// Step 1: Test getBusCount
	fmt.Println("\n1. Testing getBusCount function:")
	var count int
	err = db.Get(&count, "SELECT COUNT(*) FROM buses")
	if err != nil {
		log.Printf("ERROR: getBusCount query failed: %v", err)
	} else {
		fmt.Printf("   SUCCESS: getBusCount returned %d\n", count)
	}

	// Step 2: Test loadBusesFromDBPaginated
	fmt.Println("\n2. Testing loadBusesFromDBPaginated:")
	
	type Bus struct {
		BusID            string         `json:"bus_id" db:"bus_id"`
		Status           string         `json:"status" db:"status"`
		Model            sql.NullString `json:"model" db:"model"`
		Capacity         sql.NullInt32  `json:"capacity" db:"capacity"`
		OilStatus        sql.NullString `json:"oil_status" db:"oil_status"`
		TireStatus       sql.NullString `json:"tire_status" db:"tire_status"`
		MaintenanceNotes sql.NullString `json:"maintenance_notes" db:"maintenance_notes"`
		CurrentMileage   sql.NullInt32  `json:"current_mileage" db:"current_mileage"`
		LastOilChange    sql.NullInt32  `json:"last_oil_change" db:"last_oil_change"`
		LastTireService  sql.NullInt32  `json:"last_tire_service" db:"last_tire_service"`
		UpdatedAt        sql.NullTime   `json:"updated_at" db:"updated_at"`
		CreatedAt        sql.NullTime   `json:"created_at" db:"created_at"`
	}

	var buses []Bus
	query := fmt.Sprintf(`
		SELECT bus_id, status, model, capacity, oil_status, tire_status, 
		       maintenance_notes, current_mileage, last_oil_change, 
		       last_tire_service, updated_at, created_at 
		FROM buses 
		ORDER BY bus_id
		LIMIT %d OFFSET %d
	`, 50, 0)
	
	err = db.Select(&buses, query)
	if err != nil {
		log.Printf("ERROR: loadBusesFromDBPaginated failed: %v", err)
	} else {
		fmt.Printf("   SUCCESS: Loaded %d buses\n", len(buses))
		if len(buses) > 0 {
			fmt.Println("   Sample buses:")
			for i := 0; i < 3 && i < len(buses); i++ {
				fmt.Printf("     - Bus %s: Status=%s, Model=%s\n", 
					buses[i].BusID, buses[i].Status, buses[i].Model.String)
			}
		}
	}

	// Step 3: Simulate the ConsolidatedVehicle conversion
	fmt.Println("\n3. Testing ConsolidatedVehicle conversion:")
	type ConsolidatedVehicle struct {
		ID               string
		VehicleID        string
		BusID            string
		VehicleType      string
		Status           string
		Model            sql.NullString
		Capacity         sql.NullInt32
		OilStatus        sql.NullString
		TireStatus       sql.NullString
		MaintenanceNotes sql.NullString
		UpdatedAt        sql.NullTime
		CreatedAt        sql.NullTime
	}

	allVehicles := []ConsolidatedVehicle{}
	vehiclesByType := make(map[string][]ConsolidatedVehicle)
	
	for _, bus := range buses {
		cv := ConsolidatedVehicle{
			ID:               bus.BusID,
			VehicleID:        bus.BusID,
			BusID:            bus.BusID,
			VehicleType:      "bus",
			Status:           bus.Status,
			Model:            bus.Model,
			Capacity:         bus.Capacity,
			OilStatus:        bus.OilStatus,
			TireStatus:       bus.TireStatus,
			MaintenanceNotes: bus.MaintenanceNotes,
			UpdatedAt:        bus.UpdatedAt,
			CreatedAt:        bus.CreatedAt,
		}
		allVehicles = append(allVehicles, cv)
		vehiclesByType["bus"] = append(vehiclesByType["bus"], cv)
	}
	
	fmt.Printf("   Total vehicles converted: %d\n", len(allVehicles))
	fmt.Printf("   Buses in vehiclesByType: %d\n", len(vehiclesByType["bus"]))

	// Step 4: Calculate statistics
	fmt.Println("\n4. Calculating statistics:")
	busesSlice := vehiclesByType["bus"]
	if busesSlice == nil {
		busesSlice = []ConsolidatedVehicle{}
	}
	
	activeBuses := 0
	maintenanceBuses := 0
	outOfServiceBuses := 0
	
	for _, bus := range busesSlice {
		switch bus.Status {
		case "active":
			activeBuses++
		case "maintenance":
			maintenanceBuses++
		case "out-of-service", "out_of_service":
			outOfServiceBuses++
		default:
			activeBuses++
		}
	}
	
	fmt.Printf("   Active buses: %d\n", activeBuses)
	fmt.Printf("   Maintenance buses: %d\n", maintenanceBuses)
	fmt.Printf("   Out of service buses: %d\n", outOfServiceBuses)

	// Step 5: Final data that would be passed to template
	fmt.Println("\n5. Final template data:")
	fmt.Printf("   Buses (for template): %d items\n", len(busesSlice))
	fmt.Printf("   AllVehicles: %d items\n", len(allVehicles))
	fmt.Printf("   ActiveBuses: %d\n", activeBuses)
	fmt.Printf("   MaintenanceBuses: %d\n", maintenanceBuses)
	fmt.Printf("   OutOfServiceBuses: %d\n", outOfServiceBuses)
	
	// Check if the issue is with empty slice
	if len(busesSlice) == 0 {
		fmt.Println("\n⚠️  WARNING: busesSlice is empty! This would cause 'No Buses in Fleet' to display")
	} else {
		fmt.Println("\n✓ busesSlice has data - template should show the fleet table")
	}
}