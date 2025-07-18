package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
)

func main() {
	// Get database URL from environment
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable is not set")
	}

	// Connect to database
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Test connection
	err = db.Ping()
	if err != nil {
		log.Fatal("Failed to ping database:", err)
	}

	fmt.Println("Connected to database successfully!")
	
	// Check if buses table is empty
	var busCount int
	err = db.QueryRow("SELECT COUNT(*) FROM buses").Scan(&busCount)
	if err != nil {
		fmt.Printf("Error counting buses: %v\n", err)
		return
	}
	
	if busCount == 0 {
		fmt.Println("\nNo buses found. Adding sample buses...")
		
		// Add sample buses
		buses := []struct {
			id       string
			status   string
			model    string
			capacity int
		}{
			{"BUS001", "active", "Blue Bird Vision", 72},
			{"BUS002", "active", "Thomas Saf-T-Liner C2", 65},
			{"BUS003", "maintenance", "IC Bus CE Series", 71},
			{"BUS004", "out_of_service", "Blue Bird All American", 84},
		}
		
		for _, bus := range buses {
			_, err = db.Exec(`
				INSERT INTO buses (bus_id, status, model, capacity, oil_status, tire_status, maintenance_notes, updated_at)
				VALUES ($1, $2, $3, $4, 'good', 'good', '', $5)
				ON CONFLICT (bus_id) DO NOTHING
			`, bus.id, bus.status, bus.model, bus.capacity, time.Now())
			
			if err != nil {
				fmt.Printf("Error inserting bus %s: %v\n", bus.id, err)
			} else {
				fmt.Printf("Added bus: %s\n", bus.id)
			}
		}
	} else {
		fmt.Printf("\nFound %d buses in the database.\n", busCount)
	}
	
	// Check if vehicles table is empty
	var vehicleCount int
	err = db.QueryRow("SELECT COUNT(*) FROM vehicles").Scan(&vehicleCount)
	if err != nil {
		fmt.Printf("Error counting vehicles: %v\n", err)
		return
	}
	
	if vehicleCount == 0 {
		fmt.Println("\nNo vehicles found. Adding sample vehicles...")
		
		// Add sample vehicles
		vehicles := []struct {
			id          string
			model       string
			description string
			year        int
			license     string
			status      string
		}{
			{"VEH001", "Ford F-250", "Maintenance Truck", 2022, "ABC-1234", "active"},
			{"VEH002", "Chevrolet Express 3500", "Parts Van", 2021, "XYZ-5678", "active"},
			{"VEH003", "Ford Transit", "Supervisor Vehicle", 2023, "DEF-9012", "maintenance"},
		}
		
		for _, vehicle := range vehicles {
			_, err = db.Exec(`
				INSERT INTO vehicles (vehicle_id, model, description, year, tire_size, license, 
					oil_status, tire_status, status, maintenance_notes, serial_number, base, service_interval)
				VALUES ($1, $2, $3, $4, '225/75R16', $5, 'good', 'good', $6, '', '', 'Main Depot', 5000)
				ON CONFLICT (vehicle_id) DO NOTHING
			`, vehicle.id, vehicle.model, vehicle.description, vehicle.year, vehicle.license, vehicle.status)
			
			if err != nil {
				fmt.Printf("Error inserting vehicle %s: %v\n", vehicle.id, err)
			} else {
				fmt.Printf("Added vehicle: %s\n", vehicle.id)
			}
		}
	} else {
		fmt.Printf("\nFound %d vehicles in the database.\n", vehicleCount)
	}
	
	fmt.Println("\nDatabase check complete!")
}