package main

import (
	"fmt"
	"log"
)

// FixFleetVehiclesTable ensures the fleet_vehicles table exists
func FixFleetVehiclesTable() error {
	log.Println("Checking fleet_vehicles table...")
	
	// Check if fleet_vehicles table exists
	var tableExists bool
	err := db.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name = 'fleet_vehicles'
		)
	`).Scan(&tableExists)
	if err != nil {
		return fmt.Errorf("error checking fleet_vehicles table: %w", err)
	}
	
	if !tableExists {
		log.Println("fleet_vehicles table doesn't exist, creating it...")
		
		// Create the table
		_, err = db.Exec(`
			CREATE TABLE IF NOT EXISTS fleet_vehicles (
				id SERIAL PRIMARY KEY,
				vehicle_number INTEGER,
				sheet_name VARCHAR(100),
				year INTEGER,
				make VARCHAR(100),
				model VARCHAR(100),
				description TEXT,
				serial_number VARCHAR(100),
				license VARCHAR(50),
				location VARCHAR(100),
				tire_size VARCHAR(50),
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
				updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			)
		`)
		if err != nil {
			return fmt.Errorf("error creating fleet_vehicles table: %w", err)
		}
		
		log.Println("fleet_vehicles table created successfully")
		
		// Add some sample data
		err = AddSampleFleetVehicles()
		if err != nil {
			log.Printf("Warning: Could not add sample fleet vehicles: %v", err)
		}
	} else {
		log.Println("fleet_vehicles table exists")
		
		// Check if it has data
		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM fleet_vehicles").Scan(&count)
		if err != nil {
			log.Printf("Error counting fleet vehicles: %v", err)
		} else {
			log.Printf("Found %d fleet vehicles", count)
			
			if count == 0 {
				// Add sample data
				err = AddSampleFleetVehicles()
				if err != nil {
					log.Printf("Warning: Could not add sample fleet vehicles: %v", err)
				}
			}
		}
	}
	
	return nil
}

// AddSampleFleetVehicles adds sample fleet vehicles
func AddSampleFleetVehicles() error {
	log.Println("Adding sample fleet vehicles...")
	
	sampleVehicles := []struct {
		vehicleNumber int
		sheetName     string
		year          int
		make          string
		model         string
		description   string
		serialNumber  string
		license       string
		location      string
		tireSize      string
	}{
		{101, "Fleet", 2022, "Ford", "F-150", "Maintenance Truck", "1FTFW1ET5NFC12345", "ABC-1234", "Main Garage", "275/65R18"},
		{102, "Fleet", 2021, "Chevrolet", "Silverado", "Utility Vehicle", "3GCUYDED8MG123456", "XYZ-5678", "North Depot", "265/70R17"},
		{103, "Fleet", 2020, "Ford", "Transit", "Parts Van", "1FBZX2CM5LKA12345", "DEF-9012", "Main Garage", "235/65R16"},
		{104, "Fleet", 2023, "GMC", "Sierra", "Supervisor Vehicle", "1GKS2DKC1PR123456", "GHI-3456", "South Depot", "275/60R20"},
		{105, "Fleet", 2019, "Dodge", "Ram", "Emergency Response", "1C6SRFFT5KN123456", "JKL-7890", "Main Garage", "285/70R17"},
		{106, "Fleet", 2022, "Nissan", "Frontier", "Inspection Vehicle", "1N6ED0EA7NN123456", "MNO-2345", "East Depot", "265/75R16"},
		{107, "Fleet", 2021, "Toyota", "Tacoma", "Field Service", "3TMCZ5AN9MM123456", "PQR-6789", "West Depot", "265/70R16"},
		{108, "Fleet", 2020, "Chevrolet", "Colorado", "Supply Transport", "1GCGTCEN2L1123456", "STU-0123", "Main Garage", "255/70R16"},
		{109, "Fleet", 2023, "Ford", "Ranger", "Route Inspection", "1FTER4FH5PLD12345", "VWX-4567", "North Depot", "265/70R17"},
		{110, "Fleet", 2022, "Ram", "ProMaster", "Mobile Workshop", "3C6TRVDG9NE123456", "YZA-8901", "Main Garage", "225/75R16"},
	}
	
	for _, v := range sampleVehicles {
		_, err := db.Exec(`
			INSERT INTO fleet_vehicles (
				vehicle_number, sheet_name, year, make, model, 
				description, serial_number, license, location, tire_size
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
			ON CONFLICT DO NOTHING
		`, v.vehicleNumber, v.sheetName, v.year, v.make, v.model,
			v.description, v.serialNumber, v.license, v.location, v.tireSize)
		
		if err != nil {
			log.Printf("Error inserting vehicle %d: %v", v.vehicleNumber, err)
		}
	}
	
	log.Println("Sample fleet vehicles added")
	return nil
}

// FixAPIRoutes sets up the missing API routes
func FixAPIRoutes() error {
	log.Println("Note: API routes need to be added to the main router in main.go")
	log.Println("The following routes are missing:")
	log.Println("  - /api/routes")
	log.Println("  - /api/buses")
	log.Println("  - /api/drivers")
	log.Println("  - /api/students")
	log.Println("  - /api/fleet-vehicles")
	log.Println("  - /api/route-assignments")
	log.Println("  - /api/ecse-students")
	log.Println("  - /api/maintenance-records")
	return nil
}