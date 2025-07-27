package main

import (
	"database/sql"
	"fmt"
)

// Updated functions to work with consolidated vehicles table

// loadFleetVehicleByIDNew loads a fleet vehicle from the vehicles table
func loadFleetVehicleByIDNew(id string) (*FleetVehicle, error) {
	vehicleID := fmt.Sprintf("FV%s", id)
	query := `
		SELECT 
			CASE 
				WHEN vehicle_id LIKE 'FV%' THEN SUBSTRING(vehicle_id FROM 3)::INTEGER
				ELSE NULL
			END as id,
			vehicle_number, 
			NULL as sheet_name, 
			CASE 
				WHEN year ~ '^\d+$' THEN year::INTEGER
				ELSE NULL
			END as year,
			make, model, description, serial_number, license, location, tire_size,
			created_at, updated_at
		FROM vehicles
		WHERE vehicle_id = $1 AND vehicle_type = 'fleet'`

	var vehicle FleetVehicle
	err := db.QueryRow(query, vehicleID).Scan(
		&vehicle.ID,
		&vehicle.VehicleNumber,
		&vehicle.SheetName,
		&vehicle.Year,
		&vehicle.Make,
		&vehicle.Model,
		&vehicle.Description,
		&vehicle.SerialNumber,
		&vehicle.License,
		&vehicle.Location,
		&vehicle.TireSize,
		&vehicle.CreatedAt,
		&vehicle.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("vehicle not found")
	}
	if err != nil {
		return nil, err
	}

	return &vehicle, nil
}

// updateFleetVehicleNew updates a fleet vehicle in the vehicles table
func updateFleetVehicleNew(vehicle *FleetVehicle) error {
	vehicleID := fmt.Sprintf("FV%d", vehicle.ID)
	query := `
		UPDATE vehicles 
		SET vehicle_number = $2, 
			year = $3::text, 
			make = $4, 
			model = $5,
			description = $6, 
			serial_number = $7, 
			license = $8, 
			location = $9, 
			tire_size = $10,
			updated_at = CURRENT_TIMESTAMP
		WHERE vehicle_id = $1 AND vehicle_type = 'fleet'`

	_, err := db.Exec(query,
		vehicleID,
		vehicle.VehicleNumber,
		vehicle.Year,
		vehicle.Make,
		vehicle.Model,
		vehicle.Description,
		vehicle.SerialNumber,
		vehicle.License,
		vehicle.Location,
		vehicle.TireSize,
	)

	return err
}

// addFleetVehicleNew adds a new fleet vehicle to the vehicles table
func addFleetVehicleNew(vehicle *FleetVehicle) error {
	// Get next ID
	var maxID int
	err := db.QueryRow(`
		SELECT COALESCE(MAX(
			CASE 
				WHEN vehicle_id LIKE 'FV%' THEN SUBSTRING(vehicle_id FROM 3)::INTEGER
				ELSE 0
			END
		), 0) + 1
		FROM vehicles 
		WHERE vehicle_type = 'fleet'
	`).Scan(&maxID)
	
	if err != nil {
		return err
	}

	vehicleID := fmt.Sprintf("FV%d", maxID)
	
	query := `
		INSERT INTO vehicles (
			vehicle_id, vehicle_number, year, make, model, description,
			serial_number, license, location, tire_size, vehicle_type,
			status, created_at, updated_at
		) VALUES ($1, $2, $3::text, $4, $5, $6, $7, $8, $9, $10, 'fleet', 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		RETURNING 
			CASE 
				WHEN vehicle_id LIKE 'FV%' THEN SUBSTRING(vehicle_id FROM 3)::INTEGER
				ELSE NULL
			END`

	err = db.QueryRow(query,
		vehicleID,
		vehicle.VehicleNumber,
		vehicle.Year,
		vehicle.Make,
		vehicle.Model,
		vehicle.Description,
		vehicle.SerialNumber,
		vehicle.License,
		vehicle.Location,
		vehicle.TireSize,
	).Scan(&vehicle.ID)

	return err
}

// deleteFleetVehicleNew deletes a fleet vehicle from the vehicles table
func deleteFleetVehicleNew(id string) error {
	vehicleID := fmt.Sprintf("FV%s", id)
	query := `DELETE FROM vehicles WHERE vehicle_id = $1 AND vehicle_type = 'fleet'`
	_, err := db.Exec(query, vehicleID)
	return err
}

// UpdateAllFleetVehicleReferences updates all fleet_vehicles references to use vehicles table
// This function shows what needs to be changed in each file
func UpdateAllFleetVehicleReferences() {
	updates := []struct {
		File     string
		Function string
		Change   string
	}{
		{
			File:     "handlers_fleet_edit.go",
			Function: "loadFleetVehicleByID",
			Change:   "Replace with loadFleetVehicleByIDNew",
		},
		{
			File:     "handlers_fleet_edit.go",
			Function: "updateFleetVehicle",
			Change:   "Replace with updateFleetVehicleNew",
		},
		{
			File:     "handlers_fleet_edit.go",
			Function: "addFleetVehicle",
			Change:   "Replace with addFleetVehicleNew",
		},
		{
			File:     "handlers_fleet_edit.go",
			Function: "deleteFleetVehicle",
			Change:   "Replace with deleteFleetVehicleNew",
		},
		{
			File:     "data.go",
			Function: "loadFleetVehicles",
			Change:   "Already updated to use vehicles table",
		},
		{
			File:     "data.go",
			Function: "loadFleetVehiclesByFilters",
			Change:   "Already updated to use vehicles table",
		},
		{
			File:     "handlers.go",
			Function: "updateBusField/updateVehicleField",
			Change:   "Already updated to use correct tables",
		},
		{
			File:     "fleet_handler_clean.go",
			Function: "fleetDashboardHandler",
			Change:   "Already updated to use vehicles table",
		},
		{
			File:     "lazy_loading.go",
			Function: "LazyLoadFleetVehicles",
			Change:   "Needs update to use vehicles table",
		},
	}

	fmt.Println("Fleet Vehicles Reference Updates Required:")
	for _, update := range updates {
		fmt.Printf("- %s: %s - %s\n", update.File, update.Function, update.Change)
	}
}