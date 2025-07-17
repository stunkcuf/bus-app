package main

import (
	"fmt"
	"log"
)

// Define maintenance schedules
var maintenanceSchedules = []MaintenanceSchedule{
	{
		ItemName:      "Oil & Filter Change",
		Interval:      5000,
		WarningMiles:  500,
		CriticalMiles: 1000,
	},
	{
		ItemName:      "Tire Rotation",
		Interval:      10000,
		WarningMiles:  1000,
		CriticalMiles: 2000,
	},
	{
		ItemName:      "Air Filter",
		Interval:      15000,
		WarningMiles:  1000,
		CriticalMiles: 3000,
	},
	{
		ItemName:      "Brake Inspection",
		Interval:      20000,
		WarningMiles:  2000,
		CriticalMiles: 5000,
	},
}

// CheckMaintenanceDue checks if maintenance is due for a vehicle
func checkMaintenanceDue(vehicleID string) ([]MaintenanceAlert, error) {
	// Get current maintenance info
	currentMileage, lastOilChange, lastTireService, err := getVehicleMaintenanceInfo(vehicleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get vehicle info: %w", err)
	}

	var alerts []MaintenanceAlert

	// Check oil change
	oilMilesSince := currentMileage - lastOilChange
	if oilMilesSince >= 5000 {
		alert := MaintenanceAlert{
			VehicleID:    vehicleID,
			AlertType:    "maintenance",
			ItemName:     "Oil & Filter Change",
			Severity:     "overdue",
			MilesOverdue: oilMilesSince - 5000,
			Message:      fmt.Sprintf("Oil change overdue by %d miles", oilMilesSince-5000),
		}
		alerts = append(alerts, alert)
	} else if oilMilesSince >= 4500 {
		alert := MaintenanceAlert{
			VehicleID: vehicleID,
			AlertType: "maintenance",
			ItemName:  "Oil & Filter Change",
			Severity:  "due",
			Message:   fmt.Sprintf("Oil change due in %d miles", 5000-oilMilesSince),
		}
		alerts = append(alerts, alert)
	}

	// Check tire service
	tireMilesSince := currentMileage - lastTireService
	if tireMilesSince >= 40000 {
		alert := MaintenanceAlert{
			VehicleID:    vehicleID,
			AlertType:    "maintenance",
			ItemName:     "Tire Service",
			Severity:     "overdue",
			MilesOverdue: tireMilesSince - 40000,
			Message:      fmt.Sprintf("Tire service overdue by %d miles", tireMilesSince-40000),
		}
		alerts = append(alerts, alert)
	} else if tireMilesSince >= 35000 {
		alert := MaintenanceAlert{
			VehicleID: vehicleID,
			AlertType: "maintenance",
			ItemName:  "Tire Service",
			Severity:  "warning",
			Message:   fmt.Sprintf("Tire service due in %d miles", 40000-tireMilesSince),
		}
		alerts = append(alerts, alert)
	}

	return alerts, nil
}

// ValidateMileageEntry validates a new mileage entry
func validateMileageEntry(vehicleID string, newMileage float64) MileageValidation {
	// Get last recorded mileage
	lastMileage, err := getLastMileageForVehicle(vehicleID)
	if err != nil {
		log.Printf("Error getting last mileage: %v", err)
		// Continue with validation if we can't get last mileage
	}

	// Check if mileage is going backwards
	if lastMileage > 0 && newMileage < lastMileage {
		return MileageValidation{
			Valid: false,
			Error: fmt.Sprintf("Mileage cannot go backwards. Previous: %.0f, New: %.0f", lastMileage, newMileage),
		}
	}

	// Check for unrealistic jumps
	if lastMileage > 0 {
		mileageDiff := newMileage - lastMileage
		
		// Warning for large jumps (>1000 miles)
		if mileageDiff > 1000 {
			return MileageValidation{
				Valid:   true,
				Warning: fmt.Sprintf("Large mileage increase detected: %.0f miles. Please verify this is correct.", mileageDiff),
			}
		}
		
		// Warning for suspicious daily mileage (>500 miles in one day)
		if mileageDiff > 500 {
			return MileageValidation{
				Valid:   true,
				Warning: fmt.Sprintf("High daily mileage: %.0f miles. This is unusual for a school bus route.", mileageDiff),
			}
		}
	}

	return MileageValidation{Valid: true}
}

// UpdateMaintenanceStatusBasedOnMileage updates oil and tire status based on current mileage
func updateMaintenanceStatusBasedOnMileage(vehicleID string) error {
	// Get maintenance info
	currentMileage, lastOilChange, lastTireService, err := getVehicleMaintenanceInfo(vehicleID)
	if err != nil {
		return fmt.Errorf("failed to get maintenance info: %w", err)
	}

	// Calculate miles since last service
	oilMilesSince := currentMileage - lastOilChange
	tireMilesSince := currentMileage - lastTireService

	// Determine oil status
	var oilStatus string
	if oilMilesSince >= 5000 {
		oilStatus = "overdue"
	} else if oilMilesSince >= 4500 {
		oilStatus = "needs_service"
	} else {
		oilStatus = "good"
	}

	// Determine tire status
	var tireStatus string
	if tireMilesSince >= 40000 {
		tireStatus = "replace"
	} else if tireMilesSince >= 35000 {
		tireStatus = "worn"
	} else {
		tireStatus = "good"
	}

	// Update the status in the database
	return updateVehicleMaintenanceStatus(vehicleID, oilStatus, tireStatus)
}

// Load functions for caching

func loadBusesFromDB() ([]Bus, error) {
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var buses []Bus
	err := db.Select(&buses, "SELECT * FROM buses ORDER BY bus_id")
	if err != nil {
		return nil, fmt.Errorf("failed to load buses: %w", err)
	}

	return buses, nil
}

func loadVehiclesFromDB() ([]Vehicle, error) {
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var vehicles []Vehicle
	err := db.Select(&vehicles, "SELECT * FROM vehicles ORDER BY vehicle_id")
	if err != nil {
		return nil, fmt.Errorf("failed to load vehicles: %w", err)
	}

	return vehicles, nil
}

func loadRoutesFromDB() ([]Route, error) {
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var routes []Route
	err := db.Select(&routes, "SELECT * FROM routes ORDER BY route_id")
	if err != nil {
		return nil, fmt.Errorf("failed to load routes: %w", err)
	}

	return routes, nil
}

func loadUsersFromDB() ([]User, error) {
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var users []User
	err := db.Select(&users, "SELECT * FROM users ORDER BY username")
	if err != nil {
		return nil, fmt.Errorf("failed to load users: %w", err)
	}

	return users, nil
}

func loadStudentsFromDB() ([]Student, error) {
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var students []Student
	err := db.Select(&students, "SELECT * FROM students WHERE active = true ORDER BY name")
	if err != nil {
		return nil, fmt.Errorf("failed to load students: %w", err)
	}

	return students, nil
}

// Save functions

func saveBusMaintenanceLog(busLog BusMaintenanceLog) error {
	if db == nil {
		return fmt.Errorf("database not initialized")
	}

	query := `
		INSERT INTO bus_maintenance_logs (bus_id, date, category, notes, mileage, cost)
		VALUES (:bus_id, :date, :category, :notes, :mileage, :cost)
	`

	_, err := db.NamedExec(query, busLog)
	if err != nil {
		return fmt.Errorf("failed to save bus maintenance log: %w", err)
	}

	// Update last service mileage if applicable
	if busLog.Category == "oil_change" && busLog.Mileage > 0 {
		if err := updateLastServiceMileage(busLog.BusID, "oil_change", busLog.Mileage); err != nil {
			log.Printf("Warning: failed to update last oil change mileage: %v", err)
		}
	} else if busLog.Category == "tire_service" && busLog.Mileage > 0 {
		if err := updateLastServiceMileage(busLog.BusID, "tire_service", busLog.Mileage); err != nil {
			log.Printf("Warning: failed to update last tire service mileage: %v", err)
		}
	}

	// Update vehicle status based on new mileage
	if busLog.Mileage > 0 {
		if err := updateVehicleMileage(busLog.BusID, busLog.Mileage); err != nil {
			log.Printf("Warning: failed to update vehicle mileage: %v", err)
		}
		if err := updateMaintenanceStatusBasedOnMileage(busLog.BusID); err != nil {
			log.Printf("Warning: failed to update maintenance status: %v", err)
		}
	}

	// Invalidate cache
	dataCache.invalidateBuses()

	return nil
}

func saveVehicleMaintenanceLog(vehicleLog VehicleMaintenanceLog) error {
	if db == nil {
		return fmt.Errorf("database not initialized")
	}

	query := `
		INSERT INTO vehicle_maintenance_logs (vehicle_id, date, category, notes, mileage, cost)
		VALUES (:vehicle_id, :date, :category, :notes, :mileage, :cost)
	`

	_, err := db.NamedExec(query, vehicleLog)
	if err != nil {
		return fmt.Errorf("failed to save vehicle maintenance log: %w", err)
	}

	// Update last service mileage if applicable
	if vehicleLog.Category == "oil_change" && vehicleLog.Mileage > 0 {
		if err := updateLastServiceMileage(vehicleLog.VehicleID, "oil_change", vehicleLog.Mileage); err != nil {
			log.Printf("Warning: failed to update last oil change mileage: %v", err)
		}
	} else if vehicleLog.Category == "tire_service" && vehicleLog.Mileage > 0 {
		if err := updateLastServiceMileage(vehicleLog.VehicleID, "tire_service", vehicleLog.Mileage); err != nil {
			log.Printf("Warning: failed to update last tire service mileage: %v", err)
		}
	}

	// Update vehicle status based on new mileage
	if vehicleLog.Mileage > 0 {
		if err := updateVehicleMileage(vehicleLog.VehicleID, vehicleLog.Mileage); err != nil {
			log.Printf("Warning: failed to update vehicle mileage: %v", err)
		}
		if err := updateMaintenanceStatusBasedOnMileage(vehicleLog.VehicleID); err != nil {
			log.Printf("Warning: failed to update maintenance status: %v", err)
		}
	}

	// Invalidate cache
	dataCache.invalidateVehicles()

	return nil
}
