package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"
	
	"github.com/lib/pq"
	"github.com/jmoiron/sqlx"
)

// Transaction utility functions
type TransactionFunc func(*sqlx.Tx) error

// withTransaction executes a function within a database transaction
func withTransaction(fn TransactionFunc) error {
	if db == nil {
		return fmt.Errorf("database not initialized")
	}
	
	tx, err := db.Beginx()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	
	// Setup defer to rollback on panic or error
	committed := false
	defer func() {
		if !committed {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				log.Printf("WARNING: Failed to rollback transaction: %v", rollbackErr)
			}
		}
	}()
	
	// Execute the function
	if err := fn(tx); err != nil {
		return err
	}
	
	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	
	committed = true
	return nil
}

// Transaction-enabled helper functions
func updateLastServiceMileageInTx(tx *sqlx.Tx, vehicleID, serviceType string, mileage int) error {
	// Update last service mileage in vehicles/buses table based on service type
	var busQuery, vehicleQuery string
	
	if serviceType == "oil_change" {
		busQuery = `UPDATE buses SET last_oil_change = $2 WHERE bus_id = $1`
		vehicleQuery = `UPDATE vehicles SET last_oil_change = $2 WHERE vehicle_id = $1`
	} else if serviceType == "tire_service" {
		busQuery = `UPDATE buses SET last_tire_service = $2 WHERE bus_id = $1`
		vehicleQuery = `UPDATE vehicles SET last_tire_service = $2 WHERE vehicle_id = $1`
	} else {
		return nil // No update needed for unknown service types
	}
	
	// Try to update buses table first
	result, err := tx.Exec(busQuery, vehicleID, mileage)
	if err != nil {
		return fmt.Errorf("failed to update bus last service mileage: %w", err)
	}
	
	// Check if any rows were affected
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		// No bus found, try vehicles table
		_, err = tx.Exec(vehicleQuery, vehicleID, mileage)
		if err != nil {
			return fmt.Errorf("failed to update vehicle last service mileage: %w", err)
		}
	}
	
	return nil
}

func updateVehicleMileageInTx(tx *sqlx.Tx, vehicleID string, mileage int) error {
	// Try to update buses table first
	busQuery := `UPDATE buses SET current_mileage = $2 WHERE bus_id = $1`
	result, err := tx.Exec(busQuery, vehicleID, mileage)
	if err != nil {
		return fmt.Errorf("failed to update bus mileage: %w", err)
	}
	
	// Check if any rows were affected
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		// No bus found, try vehicles table
		vehicleQuery := `UPDATE vehicles SET current_mileage = $2 WHERE vehicle_id = $1`
		_, err = tx.Exec(vehicleQuery, vehicleID, mileage)
		if err != nil {
			return fmt.Errorf("failed to update vehicle mileage: %w", err)
		}
	}
	
	return nil
}

func updateMaintenanceStatusBasedOnMileageInTx(tx *sqlx.Tx, vehicleID string) error {
	// Try to update buses table first
	busQuery := `
		UPDATE buses SET 
			oil_status = CASE 
				WHEN (current_mileage - COALESCE(last_oil_change, 0)) > 5000 THEN 'overdue'
				WHEN (current_mileage - COALESCE(last_oil_change, 0)) > 4500 THEN 'due'
				ELSE 'good'
			END,
			tire_status = CASE 
				WHEN (current_mileage - COALESCE(last_tire_service, 0)) > 15000 THEN 'overdue'
				WHEN (current_mileage - COALESCE(last_tire_service, 0)) > 12000 THEN 'due'
				ELSE 'good'
			END
		WHERE bus_id = $1
	`
	result, err := tx.Exec(busQuery, vehicleID)
	if err != nil {
		return fmt.Errorf("failed to update bus maintenance status: %w", err)
	}
	
	// Check if any rows were affected
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		// No bus found, try vehicles table
		vehicleQuery := `
			UPDATE vehicles SET 
				oil_status = CASE 
					WHEN (current_mileage - COALESCE(last_oil_change, 0)) > 5000 THEN 'overdue'
					WHEN (current_mileage - COALESCE(last_oil_change, 0)) > 4500 THEN 'due'
					ELSE 'good'
				END,
				tire_status = CASE 
					WHEN (current_mileage - COALESCE(last_tire_service, 0)) > 15000 THEN 'overdue'
					WHEN (current_mileage - COALESCE(last_tire_service, 0)) > 12000 THEN 'due'
					ELSE 'good'
				END
			WHERE vehicle_id = $1
		`
		_, err = tx.Exec(vehicleQuery, vehicleID)
		if err != nil {
			return fmt.Errorf("failed to update vehicle maintenance status: %w", err)
		}
	}
	
	return nil
}

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

	log.Printf("DEBUG: Loading buses from database")
	var buses []Bus
	// FIXED: Use explicit columns to avoid ID mapping issue
	err := db.Select(&buses, `
		SELECT bus_id, status, model, capacity, oil_status, tire_status, 
		       maintenance_notes, current_mileage, last_oil_change, 
		       last_tire_service, updated_at, created_at 
		FROM buses 
		ORDER BY bus_id
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to load buses: %w", err)
	}

	log.Printf("DEBUG: Successfully loaded %d buses from database", len(buses))
	for i, bus := range buses {
		if i < 3 { // Log first 3 buses for debugging
			log.Printf("DEBUG: Bus %d: ID=%s, Status=%s, Model=%s", i, bus.BusID, bus.Status, bus.GetModel())
		}
	}

	return buses, nil
}

// loadBusesFromDBPaginated loads buses with pagination
func loadBusesFromDBPaginated(pagination PaginationParams) ([]Bus, error) {
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	log.Printf("DEBUG: Loading buses from database with pagination (page=%d, perPage=%d)", pagination.Page, pagination.PerPage)
	
	var buses []Bus
	query := fmt.Sprintf(`
		SELECT bus_id, status, model, capacity, oil_status, tire_status, 
		       maintenance_notes, current_mileage, last_oil_change, 
		       last_tire_service, updated_at, created_at 
		FROM buses 
		ORDER BY bus_id
		LIMIT %d OFFSET %d
	`, pagination.PerPage, pagination.Offset)
	
	err := db.Select(&buses, query)
	if err != nil {
		return nil, fmt.Errorf("failed to load buses: %w", err)
	}

	log.Printf("DEBUG: Successfully loaded %d buses from database", len(buses))
	return buses, nil
}

// getBusCount returns total number of buses
func getBusCount() (int, error) {
	if db == nil {
		return 0, fmt.Errorf("database not initialized")
	}
	
	var count int
	err := db.Get(&count, "SELECT COUNT(*) FROM buses")
	return count, err
}

// getVehicleCount returns total number of vehicles
func getVehicleCount() (int, error) {
	if db == nil {
		return 0, fmt.Errorf("database not initialized")
	}
	
	var count int
	err := db.Get(&count, "SELECT COUNT(*) FROM vehicles")
	return count, err
}

func loadVehiclesFromDB() ([]Vehicle, error) {
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var vehicles []Vehicle
	// FIXED: Use explicit columns to avoid ID mapping issue
	err := db.Select(&vehicles, `
		SELECT vehicle_id, model, description, year, tire_size, license, 
		       oil_status, tire_status, status, maintenance_notes, 
		       serial_number, base, service_interval, current_mileage, 
		       last_oil_change, last_tire_service, updated_at, created_at, import_id
		FROM vehicles 
		ORDER BY vehicle_id
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to load vehicles: %w", err)
	}

	return vehicles, nil
}

// loadVehiclesFromDBPaginated loads vehicles with pagination
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

// loadConsolidatedVehiclesFromDB loads all vehicles from the buses and vehicles tables
func loadConsolidatedVehiclesFromDB() ([]ConsolidatedVehicle, error) {
	// Just use the existing function that already does this
	return loadAllFleetVehiclesFromDB()
}

// loadConsolidatedBusesFromDB loads only buses from the buses table
func loadConsolidatedBusesFromDB() ([]ConsolidatedVehicle, error) {
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	
	// Load buses
	buses, err := loadBusesFromDB()
	if err != nil {
		return nil, fmt.Errorf("failed to load buses: %w", err)
	}
	
	var consolidatedBuses []ConsolidatedVehicle
	
	// Collect all bus IDs for batch loading assignments
	var busIDs []string
	for _, bus := range buses {
		busIDs = append(busIDs, bus.BusID)
	}
	
	// Batch load all assignments
	assignments, err := getVehicleAssignments(busIDs)
	if err != nil {
		log.Printf("Error loading vehicle assignments: %v", err)
		assignments = make(map[string]*RouteAssignment)
	}
	
	// Convert buses to ConsolidatedVehicle
	for _, bus := range buses {
		vehicle := ConsolidatedVehicle{
			ID:               bus.BusID,
			VehicleID:        bus.BusID,
			VehicleType:      "bus",
			Status:           bus.Status,
			Model:            bus.Model,
			Capacity:         bus.Capacity,
			OilStatus:        bus.OilStatus,
			TireStatus:       bus.TireStatus,
			MaintenanceNotes: bus.MaintenanceNotes,
			UpdatedAt:        bus.UpdatedAt,
			CreatedAt:        bus.CreatedAt,
			BusID:            bus.BusID, // For backward compatibility
			Assignment:       assignments[bus.BusID],
		}
		consolidatedBuses = append(consolidatedBuses, vehicle)
	}
	
	log.Printf("Loaded %d buses", len(consolidatedBuses))
	return consolidatedBuses, nil
}

// loadConsolidatedNonBusVehiclesFromDB loads only non-bus vehicles from the vehicles table  
func loadConsolidatedNonBusVehiclesFromDB() ([]ConsolidatedVehicle, error) {
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	
	// Load vehicles
	vehicles, err := loadVehiclesFromDB()
	if err != nil {
		return nil, fmt.Errorf("failed to load vehicles: %w", err)
	}
	
	var consolidatedVehicles []ConsolidatedVehicle
	
	// Convert vehicles to ConsolidatedVehicle
	for _, veh := range vehicles {
		// Parse year as int
		year := 0
		if veh.Year.Valid && veh.Year.String != "" {
			fmt.Sscanf(veh.Year.String, "%d", &year)
		}
		
		// Convert year int to string  
		yearStr := sql.NullString{Valid: false}
		if year > 0 {
			yearStr = sql.NullString{String: fmt.Sprintf("%d", year), Valid: true}
		}
		
		vehicle := ConsolidatedVehicle{
			ID:               veh.VehicleID,
			VehicleID:        veh.VehicleID,
			VehicleType:      "vehicle",
			Status:           func() string {
				if veh.Status.Valid {
					return veh.Status.String
				}
				return "active"
			}(),
			Model:            veh.Model,
			Year:             yearStr,
			TireSize:         veh.TireSize,
			License:          veh.License,
			Description:      veh.Description,
			SerialNumber:     veh.SerialNumber,
			Base:             veh.Base,
			ServiceInterval:  veh.ServiceInterval,
			OilStatus:        veh.OilStatus,
			TireStatus:       veh.TireStatus,
			MaintenanceNotes: veh.MaintenanceNotes,
			UpdatedAt:        veh.UpdatedAt,
			CreatedAt:        veh.CreatedAt,
			BusID:            veh.VehicleID, // For backward compatibility
		}
		consolidatedVehicles = append(consolidatedVehicles, vehicle)
	}
	
	log.Printf("Loaded %d non-bus vehicles", len(consolidatedVehicles))
	return consolidatedVehicles, nil
}

// loadAllFleetVehiclesFromDB loads ALL vehicles from the buses and vehicles tables
func loadAllFleetVehiclesFromDB() ([]ConsolidatedVehicle, error) {
	log.Printf("DEBUG: loadAllFleetVehiclesFromDB - Starting")
	if db == nil {
		log.Printf("ERROR: loadAllFleetVehiclesFromDB - Database is nil")
		return nil, fmt.Errorf("database not initialized")
	}
	
	var allVehicles []ConsolidatedVehicle
	var allVehicleIDs []string
	
	// Load buses
	buses, err := loadBusesFromDB()
	if err != nil {
		return nil, fmt.Errorf("failed to load buses: %w", err)
	}
	
	// Collect all vehicle IDs for batch loading
	for _, bus := range buses {
		allVehicleIDs = append(allVehicleIDs, bus.BusID)
	}
	
	// Load vehicles
	vehicles, err := loadVehiclesFromDB()
	if err != nil {
		return nil, fmt.Errorf("failed to load vehicles: %w", err)
	}
	
	// Collect vehicle IDs too
	for _, veh := range vehicles {
		allVehicleIDs = append(allVehicleIDs, veh.VehicleID)
	}
	
	// Batch load all assignments
	assignments, err := getVehicleAssignments(allVehicleIDs)
	if err != nil {
		log.Printf("Error loading vehicle assignments: %v", err)
		assignments = make(map[string]*RouteAssignment)
	}
	
	// Convert buses to ConsolidatedVehicle
	for _, bus := range buses {
		vehicle := ConsolidatedVehicle{
			ID:               bus.BusID,
			VehicleID:        bus.BusID,
			VehicleType:      "bus",
			Status:           bus.Status,
			Model:            bus.Model,
			Capacity:         bus.Capacity,
			OilStatus:        bus.OilStatus,
			TireStatus:       bus.TireStatus,
			MaintenanceNotes: bus.MaintenanceNotes,
			UpdatedAt:        bus.UpdatedAt,
			CreatedAt:        bus.CreatedAt,
			BusID:            bus.BusID, // For backward compatibility
			Assignment:       assignments[bus.BusID],
		}
		allVehicles = append(allVehicles, vehicle)
	}
	
	// Convert vehicles to ConsolidatedVehicle
	for _, veh := range vehicles {
		// Parse year as int
		year := 0
		if veh.Year.Valid && veh.Year.String != "" {
			fmt.Sscanf(veh.Year.String, "%d", &year)
		}
		
		// Convert year int to string  
		yearStr := sql.NullString{Valid: false}
		if year > 0 {
			yearStr = sql.NullString{String: fmt.Sprintf("%d", year), Valid: true}
		}
		
		vehicle := ConsolidatedVehicle{
			ID:               veh.VehicleID,
			VehicleID:        veh.VehicleID,
			VehicleType:      "vehicle",
			Status:           func() string {
				if veh.Status.Valid {
					return veh.Status.String
				}
				return "active"
			}(),
			Model:            veh.Model,
			Year:             yearStr,
			TireSize:         veh.TireSize,
			License:          veh.License,
			Description:      veh.Description,
			SerialNumber:     veh.SerialNumber,
			Base:             veh.Base,
			ServiceInterval:  veh.ServiceInterval,
			OilStatus:        veh.OilStatus,
			TireStatus:       veh.TireStatus,
			MaintenanceNotes: veh.MaintenanceNotes,
			UpdatedAt:        veh.UpdatedAt,
			CreatedAt:        veh.CreatedAt,
			BusID:            veh.VehicleID, // For backward compatibility
			Assignment:       assignments[veh.VehicleID],
		}
		allVehicles = append(allVehicles, vehicle)
	}
	
	log.Printf("Loaded %d total vehicles (buses: %d, vehicles: %d)", len(allVehicles), len(buses), len(vehicles))
	return allVehicles, nil
}

// loadFleetVehiclesByType loads vehicles grouped by type
func loadFleetVehiclesByType() (map[string][]ConsolidatedVehicle, error) {
	vehicles, err := loadAllFleetVehiclesFromDB()
	if err != nil {
		return nil, err
	}
	
	// Group by vehicle type
	grouped := make(map[string][]ConsolidatedVehicle)
	for _, v := range vehicles {
		vType := v.VehicleType
		if vType == "" {
			vType = "other"
		}
		grouped[vType] = append(grouped[vType], v)
	}
	
	// Log grouped counts
	for vType, vehicles := range grouped {
		log.Printf("Vehicle type '%s': %d vehicles", vType, len(vehicles))
	}
	
	return grouped, nil
}

// loadRouteAssignments loads all route assignments from the database
func loadRouteAssignments() ([]RouteAssignment, error) {
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	
	var assignments []RouteAssignment
	err := db.Select(&assignments, `
		SELECT ra.id, ra.driver, ra.bus_id, ra.route_id, r.route_name, ra.assigned_date, ra.created_at
		FROM route_assignments ra
		JOIN routes r ON ra.route_id = r.route_id
		ORDER BY ra.assigned_date DESC
	`)
	
	if err != nil {
		return nil, fmt.Errorf("failed to load route assignments: %w", err)
	}
	
	return assignments, nil
}

// getVehicleAssignment gets the current route assignment for a vehicle
func getVehicleAssignment(vehicleID string) *RouteAssignment {
	if db == nil {
		return nil
	}
	
	var assignment RouteAssignment
	err := db.Get(&assignment, `
		SELECT ra.id, ra.driver, ra.bus_id, ra.route_id, r.route_name, ra.assigned_date, ra.created_at
		FROM route_assignments ra
		JOIN routes r ON ra.route_id = r.route_id
		WHERE ra.bus_id = $1
		LIMIT 1
	`, vehicleID)
	
	if err != nil {
		return nil
	}
	
	return &assignment
}

// getVehicleAssignments gets current route assignments for multiple vehicles in a single query
func getVehicleAssignments(vehicleIDs []string) (map[string]*RouteAssignment, error) {
	if db == nil || len(vehicleIDs) == 0 {
		return make(map[string]*RouteAssignment), nil
	}
	
	query := `
		SELECT ra.id, ra.driver, ra.bus_id, ra.route_id, r.route_name, ra.assigned_date, ra.created_at
		FROM route_assignments ra
		JOIN routes r ON ra.route_id = r.route_id
		WHERE ra.bus_id = ANY($1)
	`
	
	var assignments []RouteAssignment
	err := db.Select(&assignments, query, pq.StringArray(vehicleIDs))
	if err != nil {
		return nil, err
	}
	
	// Create map of vehicle ID to assignment
	assignmentMap := make(map[string]*RouteAssignment)
	for i := range assignments {
		assignmentMap[assignments[i].BusID] = &assignments[i]
	}
	
	return assignmentMap, nil
}

func loadRoutesFromDB() ([]Route, error) {
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var routes []Route
	// FIXED: Select only columns that exist in the Route struct
	err := db.Select(&routes, `
		SELECT route_id, route_name, description, positions, created_at 
		FROM routes 
		ORDER BY route_id
	`)
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
	// FIXED: Select only columns that exist in the User struct, including id
	err := db.Select(&users, `
		SELECT id, username, password, role, status, registration_date, created_at 
		FROM users 
		ORDER BY username
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to load users: %w", err)
	}

	return users, nil
}

// loadECSEStudentsFromDB loads ECSE students from database
func loadECSEStudentsFromDB() ([]ECSEStudent, error) {
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	
	var students []ECSEStudent
	err := db.Select(&students, "SELECT * FROM ecse_students ORDER BY last_name, first_name")
	if err != nil {
		return nil, fmt.Errorf("failed to load ECSE students: %w", err)
	}
	return students, nil
}

func loadStudentsFromDB() ([]Student, error) {
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var students []Student
	// FIXED: Use COALESCE to handle NULL values
	err := db.Select(&students, `
		SELECT 
			COALESCE(student_id, '') as student_id,
			COALESCE(name, '') as name,
			COALESCE(locations::text, '[]') as locations,
			COALESCE(phone_number, '') as phone_number,
			COALESCE(alt_phone_number, '') as alt_phone_number,
			COALESCE(guardian, '') as guardian,
			COALESCE(pickup_time::text, '') as pickup_time,
			COALESCE(dropoff_time::text, '') as dropoff_time,
			COALESCE(position_number, 0) as position_number,
			COALESCE(route_id, '') as route_id,
			COALESCE(driver, '') as driver,
			COALESCE(active, false) as active,
			COALESCE(created_at, CURRENT_TIMESTAMP) as created_at
		FROM students 
		WHERE active = true 
		ORDER BY name
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to load students: %w", err)
	}

	return students, nil
}

// loadStudentsFromDBPaginated loads active students with pagination
func loadStudentsFromDBPaginated(pagination PaginationParams, includeInactive bool) ([]Student, error) {
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var students []Student
	whereClause := ""
	if !includeInactive {
		whereClause = "WHERE active = true"
	}
	
	query := fmt.Sprintf(`
		SELECT 
			COALESCE(student_id, '') as student_id,
			COALESCE(name, '') as name,
			COALESCE(locations::text, '[]') as locations,
			COALESCE(phone_number, '') as phone_number,
			COALESCE(alt_phone_number, '') as alt_phone_number,
			COALESCE(guardian, '') as guardian,
			COALESCE(pickup_time::text, '') as pickup_time,
			COALESCE(dropoff_time::text, '') as dropoff_time,
			COALESCE(position_number, 0) as position_number,
			COALESCE(route_id, '') as route_id,
			COALESCE(driver, '') as driver,
			COALESCE(active, false) as active,
			COALESCE(created_at, CURRENT_TIMESTAMP) as created_at
		FROM students 
		%s 
		ORDER BY name
		LIMIT %d OFFSET %d
	`, whereClause, pagination.PerPage, pagination.Offset)
	
	err := db.Select(&students, query)
	if err != nil {
		return nil, fmt.Errorf("failed to load students: %w", err)
	}

	return students, nil
}

// getStudentCount returns total number of students
func getStudentCount(includeInactive bool) (int, error) {
	if db == nil {
		return 0, fmt.Errorf("database not initialized")
	}
	
	var count int
	query := "SELECT COUNT(*) FROM students"
	if !includeInactive {
		query += " WHERE active = true"
	}
	
	err := db.Get(&count, query)
	return count, err
}

// Save functions

func saveBusMaintenanceLog(busLog BusMaintenanceLog) error {
	return withTransaction(func(tx *sqlx.Tx) error {
		// Insert into the consolidated maintenance_records table
		query := `
			INSERT INTO maintenance_records (vehicle_id, service_date, work_description, mileage, cost, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		`

		// Combine category and notes for work_description
		workDescription := busLog.Category
		if busLog.Notes != "" {
			workDescription = busLog.Category + ": " + busLog.Notes
		}

		_, err := tx.Exec(query, busLog.BusID, busLog.Date, workDescription, busLog.Mileage, busLog.Cost)
		if err != nil {
			return fmt.Errorf("failed to save bus maintenance log: %w", err)
		}

		// Update last service mileage if applicable
		if busLog.Category == "oil_change" && busLog.Mileage > 0 {
			if err := updateLastServiceMileageInTx(tx, busLog.BusID, "oil_change", busLog.Mileage); err != nil {
				return fmt.Errorf("failed to update last oil change mileage: %w", err)
			}
		} else if busLog.Category == "tire_service" && busLog.Mileage > 0 {
			if err := updateLastServiceMileageInTx(tx, busLog.BusID, "tire_service", busLog.Mileage); err != nil {
				return fmt.Errorf("failed to update last tire service mileage: %w", err)
			}
		}

		// Update vehicle status based on new mileage
		if busLog.Mileage > 0 {
			if err := updateVehicleMileageInTx(tx, busLog.BusID, busLog.Mileage); err != nil {
				return fmt.Errorf("failed to update vehicle mileage: %w", err)
			}
			if err := updateMaintenanceStatusBasedOnMileageInTx(tx, busLog.BusID); err != nil {
				return fmt.Errorf("failed to update maintenance status: %w", err)
			}
		}

		// Invalidate cache after successful transaction
		dataCache.invalidateBuses()
		
		return nil
	})
}

func saveVehicleMaintenanceLog(vehicleLog VehicleMaintenanceLog) error {
	return withTransaction(func(tx *sqlx.Tx) error {
		// Insert into the consolidated maintenance_records table
		query := `
			INSERT INTO maintenance_records (vehicle_id, service_date, work_description, mileage, cost, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		`

		// Combine category and notes for work_description
		workDescription := vehicleLog.Category
		if vehicleLog.Notes != "" {
			workDescription = vehicleLog.Category + ": " + vehicleLog.Notes
		}

		_, err := tx.Exec(query, vehicleLog.VehicleID, vehicleLog.Date, workDescription, vehicleLog.Mileage, vehicleLog.Cost)
		if err != nil {
			return fmt.Errorf("failed to save vehicle maintenance log: %w", err)
		}

		// Update last service mileage if applicable
		if vehicleLog.Category == "oil_change" && vehicleLog.Mileage > 0 {
			if err := updateLastServiceMileageInTx(tx, vehicleLog.VehicleID, "oil_change", vehicleLog.Mileage); err != nil {
				return fmt.Errorf("failed to update last oil change mileage: %w", err)
			}
		} else if vehicleLog.Category == "tire_service" && vehicleLog.Mileage > 0 {
			if err := updateLastServiceMileageInTx(tx, vehicleLog.VehicleID, "tire_service", vehicleLog.Mileage); err != nil {
				return fmt.Errorf("failed to update last tire service mileage: %w", err)
			}
		}

		// Update vehicle status based on new mileage
		if vehicleLog.Mileage > 0 {
			if err := updateVehicleMileageInTx(tx, vehicleLog.VehicleID, vehicleLog.Mileage); err != nil {
				return fmt.Errorf("failed to update vehicle mileage: %w", err)
			}
			if err := updateMaintenanceStatusBasedOnMileageInTx(tx, vehicleLog.VehicleID); err != nil {
				return fmt.Errorf("failed to update maintenance status: %w", err)
			}
		}

		// Invalidate cache after successful transaction
		dataCache.invalidateVehicles()

		return nil
	})
}

// loadMonthlyMileageReportsFromDB loads all monthly mileage reports from database
func loadMonthlyMileageReportsFromDB() ([]MonthlyMileageReport, error) {
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var reports []MonthlyMileageReport
	query := `
		SELECT id, report_month, report_year, bus_year, bus_make, 
		       license_plate, bus_id, located_at, beginning_miles, 
		       ending_miles, total_miles, created_at, updated_at
		FROM monthly_mileage_reports 
		ORDER BY report_year DESC, 
		         CASE report_month 
		             WHEN 'January' THEN 1 WHEN 'February' THEN 2 WHEN 'March' THEN 3
		             WHEN 'April' THEN 4 WHEN 'May' THEN 5 WHEN 'June' THEN 6
		             WHEN 'July' THEN 7 WHEN 'August' THEN 8 WHEN 'September' THEN 9
		             WHEN 'October' THEN 10 WHEN 'November' THEN 11 WHEN 'December' THEN 12
		             ELSE 0 
		         END DESC,
		         bus_id`

	err := db.Select(&reports, query)
	if err != nil {
		return nil, fmt.Errorf("failed to load monthly mileage reports: %w", err)
	}

	return reports, nil
}

// generateCurrentMonthMileageReports generates monthly mileage reports for the current month from driver logs
func generateCurrentMonthMileageReports() ([]MonthlyMileageReport, error) {
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	// Get current month and year
	now := time.Now()
	currentMonth := now.Month()
	currentYear := now.Year()
	monthName := currentMonth.String()

	// Get first and last day of current month
	firstDay := time.Date(currentYear, currentMonth, 1, 0, 0, 0, 0, time.Local)
	lastDay := firstDay.AddDate(0, 1, -1)

	// Query to get ALL vehicles with their mileage data for current month
	// This includes both buses and company vehicles
	query := `
		WITH all_vehicles AS (
			-- Get all buses
			SELECT 
				b.bus_id as id,
				b.bus_id as vehicle_id,
				EXTRACT(YEAR FROM NOW())::VARCHAR as year,
				'School Bus' as make,
				COALESCE(b.model::text, 'Standard') as model,
				'' as license_plate,
				'Bus' as vehicle_type,
				COALESCE(b.current_mileage, 0) as current_mileage
			FROM buses b
			WHERE b.status = 'active'
			
			UNION ALL
			
			-- Get all company vehicles
			SELECT 
				v.vehicle_id as id,
				v.vehicle_id,
				COALESCE(v.year, EXTRACT(YEAR FROM NOW())::VARCHAR) as year,
				'Vehicle' as make,
				COALESCE(v.model::text, 'Company Vehicle') as model,
				COALESCE(v.license::text, '') as license_plate,
				'Vehicle' as vehicle_type,
				COALESCE(v.current_mileage, 0) as current_mileage
			FROM vehicles v
			WHERE v.status = 'active'
		),
		monthly_data AS (
			-- Get mileage data from driver logs for current month
			SELECT 
				dl.bus_id as vehicle_id,
				MIN(dl.start_mileage) as beginning_miles,
				MAX(dl.end_mileage) as ending_miles,
				SUM(CASE 
					WHEN dl.end_mileage > dl.start_mileage 
					THEN dl.end_mileage - dl.start_mileage 
					ELSE 0 
				END) as total_miles
			FROM driver_logs dl
			WHERE dl.date >= $1 AND dl.date <= $2
				AND dl.bus_id IS NOT NULL
			GROUP BY dl.bus_id
		)
		SELECT 
			av.vehicle_id,
			COALESCE(md.beginning_miles, av.current_mileage) as beginning_miles,
			COALESCE(md.ending_miles, av.current_mileage) as ending_miles,
			COALESCE(md.total_miles, 0) as total_miles,
			av.year as vehicle_year,
			av.make as vehicle_make,
			av.model as vehicle_model,
			av.license_plate,
			av.vehicle_type
		FROM all_vehicles av
		LEFT JOIN monthly_data md ON av.id = md.vehicle_id
		ORDER BY av.vehicle_type, av.vehicle_id`

	rows, err := db.Query(query, firstDay.Format("2006-01-02"), lastDay.Format("2006-01-02"))
	if err != nil {
		return nil, fmt.Errorf("failed to query monthly mileage data: %w", err)
	}
	defer rows.Close()

	var reports []MonthlyMileageReport
	for rows.Next() {
		var report MonthlyMileageReport
		var vehicleModel, vehicleType string
		
		report.ReportMonth = monthName
		report.ReportYear = currentYear
		report.CreatedAt = now
		report.UpdatedAt = now

		err := rows.Scan(
			&report.BusID,
			&report.BeginningMiles,
			&report.EndingMiles,
			&report.TotalMiles,
			&report.BusYear,
			&report.BusMake,
			&vehicleModel,
			&report.LicensePlate,
			&vehicleType,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan monthly mileage row: %w", err)
		}
		
		// Combine make and model for display
		if vehicleModel != "" && report.BusMake != vehicleModel {
			report.BusMake = report.BusMake + " " + vehicleModel
		}
		
		// Set location based on vehicle type
		if report.LocatedAt == "" {
			report.LocatedAt = vehicleType
		}

		reports = append(reports, report)
	}

	// Return the reports sorted by vehicle type and ID
	return reports, nil
}

// loadMonthlyMileageReportsByFilters loads monthly mileage reports with filters
func loadMonthlyMileageReportsByFilters(year int, month string, busID string) ([]MonthlyMileageReport, error) {
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var reports []MonthlyMileageReport
	var conditions []string
	var args []interface{}

	baseQuery := `
		SELECT id, report_month, report_year, bus_year, bus_make, 
		       license_plate, bus_id, located_at, beginning_miles, 
		       ending_miles, total_miles, created_at, updated_at
		FROM monthly_mileage_reports WHERE 1=1`

	if year > 0 {
		conditions = append(conditions, " AND report_year = $"+fmt.Sprintf("%d", len(args)+1))
		args = append(args, year)
	}

	if month != "" {
		conditions = append(conditions, " AND report_month = $"+fmt.Sprintf("%d", len(args)+1))
		args = append(args, month)
	}

	if busID != "" {
		conditions = append(conditions, " AND bus_id = $"+fmt.Sprintf("%d", len(args)+1))
		args = append(args, busID)
	}

	query := baseQuery
	for _, condition := range conditions {
		query += condition
	}

	query += ` ORDER BY report_year DESC, 
	              CASE report_month 
	                  WHEN 'January' THEN 1 WHEN 'February' THEN 2 WHEN 'March' THEN 3
	                  WHEN 'April' THEN 4 WHEN 'May' THEN 5 WHEN 'June' THEN 6
	                  WHEN 'July' THEN 7 WHEN 'August' THEN 8 WHEN 'September' THEN 9
	                  WHEN 'October' THEN 10 WHEN 'November' THEN 11 WHEN 'December' THEN 12
	                  ELSE 0 
	              END DESC,
	              bus_id`

	err := db.Select(&reports, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to load filtered monthly mileage reports: %w", err)
	}

	return reports, nil
}

// loadFleetVehiclesFromDB loads all fleet vehicles from database
func loadFleetVehiclesFromDB() ([]FleetVehicle, error) {
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var vehicles []FleetVehicle
	query := `
		SELECT 
			COALESCE(id, 1) as id,
			NULL::INTEGER as vehicle_number, 
			NULL as sheet_name, 
			CASE 
				WHEN year ~ '^\d+$' THEN year::INTEGER
				ELSE NULL
			END as year,
			NULL as make, model, 
		    description, serial_number, license, base as location, tire_size,
		    created_at, updated_at
		FROM vehicles 
		WHERE status = 'active'
		ORDER BY 
			vehicle_id,
			year DESC, model`

	err := db.Select(&vehicles, query)
	if err != nil {
		return nil, fmt.Errorf("failed to load fleet vehicles: %w", err)
	}

	return vehicles, nil
}

// loadFleetVehiclesByFilters loads filtered fleet vehicles
func loadFleetVehiclesByFilters(year int, make string, location string) ([]FleetVehicle, error) {
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var vehicles []FleetVehicle
	var conditions []string
	var args []interface{}

	baseQuery := `
		SELECT 
			COALESCE(id, 1) as id,
			NULL::INTEGER as vehicle_number, 
			NULL as sheet_name, 
			CASE 
				WHEN year ~ '^\d+$' THEN year::INTEGER
				ELSE NULL
			END as year,
			NULL as make, model, 
		    description, serial_number, license, base as location, tire_size,
		    created_at, updated_at
		FROM vehicles 
		WHERE status = 'active'`

	if year > 0 {
		conditions = append(conditions, " AND year = $"+fmt.Sprintf("%d", len(args)+1))
		args = append(args, year)
	}

	// Make filter disabled as vehicles table doesn't have make column
	// if make != "" {
	// 	conditions = append(conditions, " AND UPPER(make) LIKE UPPER($"+fmt.Sprintf("%d", len(args)+1)+")")
	// 	args = append(args, "%"+make+"%")
	// }

	if location != "" {
		conditions = append(conditions, " AND UPPER(base) LIKE UPPER($"+fmt.Sprintf("%d", len(args)+1)+")")
		args = append(args, "%"+location+"%")
	}

	query := baseQuery
	for _, condition := range conditions {
		query += condition
	}

	query += ` ORDER BY 
		id,
		year DESC, model`

	err := db.Select(&vehicles, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to load filtered fleet vehicles: %w", err)
	}

	return vehicles, nil
}

// loadMaintenanceRecordsFromDB loads all maintenance records from database
// Note: This loads ALL records - consider using loadMaintenanceRecordsFromDBPaginated for better performance
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

// loadMaintenanceRecordsByFilters loads filtered maintenance records
func loadMaintenanceRecordsByFilters(vehicleNumber int, startDate, endDate string) ([]MaintenanceRecord, error) {
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var records []MaintenanceRecord
	var conditions []string
	var args []interface{}

	baseQuery := `
		SELECT id, vehicle_number, service_date, mileage, po_number, cost, 
		       work_description, raw_data, created_at, updated_at, vehicle_id, date
		FROM maintenance_records WHERE 1=1`

	if vehicleNumber > 0 {
		conditions = append(conditions, " AND vehicle_number = $"+fmt.Sprintf("%d", len(args)+1))
		args = append(args, vehicleNumber)
	}

	if startDate != "" {
		conditions = append(conditions, " AND (service_date >= $"+fmt.Sprintf("%d", len(args)+1)+" OR date >= $"+fmt.Sprintf("%d", len(args)+1)+")")
		args = append(args, startDate)
	}

	if endDate != "" {
		conditions = append(conditions, " AND (service_date <= $"+fmt.Sprintf("%d", len(args)+1)+" OR date <= $"+fmt.Sprintf("%d", len(args)+1)+")")
		args = append(args, endDate)
	}

	query := baseQuery
	for _, condition := range conditions {
		query += condition
	}

	query += ` ORDER BY 
		COALESCE(service_date, date, created_at) DESC,
		vehicle_number, id`

	err := db.Select(&records, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to load filtered maintenance records: %w", err)
	}

	return records, nil
}

// loadMaintenanceRecordsFromDBPaginated loads maintenance records with pagination
func loadMaintenanceRecordsFromDBPaginated(pagination PaginationParams, vehicleID string) ([]MaintenanceRecord, error) {
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var records []MaintenanceRecord
	whereClause := ""
	args := []interface{}{}
	
	if vehicleID != "" {
		whereClause = "WHERE vehicle_id = $1 OR vehicle_number = $1"
		args = append(args, vehicleID)
	}
	
	// Build query with placeholders
	baseQuery := fmt.Sprintf(`
		SELECT id, vehicle_number, service_date, mileage, po_number, cost, 
		       work_description, raw_data, created_at, updated_at, vehicle_id, date
		FROM maintenance_records 
		%s
		ORDER BY 
			COALESCE(service_date, date, created_at) DESC,
			vehicle_number, id
		LIMIT $%d OFFSET $%d`, whereClause, len(args)+1, len(args)+2)
	
	args = append(args, pagination.PerPage, pagination.Offset)
	
	err := db.Select(&records, baseQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to load maintenance records: %w", err)
	}

	return records, nil
}

// getMaintenanceRecordCount returns total number of maintenance records
func getMaintenanceRecordCount(vehicleID string) (int, error) {
	if db == nil {
		return 0, fmt.Errorf("database not initialized")
	}
	
	var count int
	if vehicleID != "" {
		err := db.Get(&count, "SELECT COUNT(*) FROM maintenance_records WHERE vehicle_id = $1 OR vehicle_number = $1", vehicleID)
		return count, err
	}
	
	err := db.Get(&count, "SELECT COUNT(*) FROM maintenance_records")
	return count, err
}

// loadServiceRecordsFromDB loads all service records from database
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

// loadServiceRecordsByFilters loads filtered service records
func loadServiceRecordsByFilters(vehicleFilter string, startDate, endDate string) ([]ServiceRecord, error) {
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var records []ServiceRecord
	var conditions []string
	var args []interface{}

	baseQuery := `
		SELECT id, unnamed_0, unnamed_1, unnamed_2, unnamed_3, unnamed_4, unnamed_5, 
		       unnamed_6, unnamed_7, unnamed_8, unnamed_9, unnamed_10, unnamed_11, 
		       unnamed_12, unnamed_13, created_at, updated_at, maintenance_date
		FROM service_records WHERE 1=1`

	if vehicleFilter != "" {
		// Search across multiple fields that might contain vehicle info
		conditions = append(conditions, " AND (UPPER(unnamed_0) LIKE UPPER($"+fmt.Sprintf("%d", len(args)+1)+") OR UPPER(unnamed_1) LIKE UPPER($"+fmt.Sprintf("%d", len(args)+1)+") OR UPPER(unnamed_2) LIKE UPPER($"+fmt.Sprintf("%d", len(args)+1)+"))")
		args = append(args, "%"+vehicleFilter+"%")
	}

	if startDate != "" {
		conditions = append(conditions, " AND (maintenance_date >= $"+fmt.Sprintf("%d", len(args)+1)+" OR created_at >= $"+fmt.Sprintf("%d", len(args)+1)+")")
		args = append(args, startDate)
	}

	if endDate != "" {
		conditions = append(conditions, " AND (maintenance_date <= $"+fmt.Sprintf("%d", len(args)+1)+" OR created_at <= $"+fmt.Sprintf("%d", len(args)+1)+")")
		args = append(args, endDate)
	}

	query := baseQuery
	for _, condition := range conditions {
		query += condition
	}

	query += ` ORDER BY 
		COALESCE(maintenance_date, created_at) DESC,
		id`

	err := db.Select(&records, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to load filtered service records: %w", err)
	}

	return records, nil
}

// loadFuelRecordsFromDB loads all fuel records from database
func loadFuelRecordsFromDB() ([]FuelRecord, error) {
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	var records []FuelRecord
	query := `
		SELECT id, vehicle_id, date, gallons, price_per_gallon, cost, 
		       odometer, location, driver, notes, created_at
		FROM fuel_records 
		ORDER BY date DESC, id DESC`
	err := db.Select(&records, query)
	if err != nil {
		return nil, fmt.Errorf("failed to load fuel records: %w", err)
	}
	return records, nil
}

// getStudentCountsByRoute returns a map of route IDs to student counts
func getStudentCountsByRoute() (map[string]int, error) {
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	studentCounts := make(map[string]int)
	
	// Get student counts per route
	rows, err := db.Query(`
		SELECT route_id, COUNT(*) as student_count
		FROM students
		WHERE route_id IS NOT NULL AND route_id != ''
		GROUP BY route_id
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to get student counts: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var routeID string
		var count int
		if err := rows.Scan(&routeID, &count); err != nil {
			log.Printf("Error scanning student count: %v", err)
			continue
		}
		studentCounts[routeID] = count
	}

	return studentCounts, nil
}
