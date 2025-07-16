package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

var db *sqlx.DB

// InitDB initializes the database connection
func InitDB(dataSourceName string) error {
	var err error
	db, err = sqlx.Open("postgres", dataSourceName)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Test the connection
	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	// Run migrations
	if err := runMigrations(); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

// runMigrations runs database migrations
		// Add city, state, zip_code columns to ecse_students if they don't exist
		`DO $$ 
		BEGIN 
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
				WHERE table_name = 'ecse_students' AND column_name = 'city') THEN
				ALTER TABLE ecse_students ADD COLUMN city VARCHAR(100);
			END IF;
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
				WHERE table_name = 'ecse_students' AND column_name = 'state') THEN
				ALTER TABLE ecse_students ADD COLUMN state VARCHAR(50);
			END IF;
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
				WHERE table_name = 'ecse_students' AND column_name = 'zip_code') THEN
				ALTER TABLE ecse_students ADD COLUMN zip_code VARCHAR(20);
			END IF;
		END $$;`,

// Database query functions

// GetMaintenanceLogsForVehicle retrieves all maintenance logs for a vehicle (bus or regular vehicle)
func getMaintenanceLogsForVehicle(vehicleID string) ([]CombinedMaintenanceLog, error) {
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	query := `
		SELECT 
			id,
			bus_id as vehicle_id,
			'bus' as vehicle_type,
			date,
			category,
			notes,
			mileage,
			cost,
			created_at
		FROM bus_maintenance_logs
		WHERE bus_id = $1
		
		UNION ALL
		
		SELECT 
			id,
			vehicle_id,
			'vehicle' as vehicle_type,
			date,
			category,
			notes,
			mileage,
			cost,
			created_at
		FROM vehicle_maintenance_logs
		WHERE vehicle_id = $1
		
		ORDER BY date DESC, created_at DESC
	`

	var logs []CombinedMaintenanceLog
	err := db.Select(&logs, query, vehicleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get maintenance logs: %w", err)
	}

	return logs, nil
}

// GetVehicleCurrentMileage gets the current mileage for any vehicle
func getVehicleCurrentMileage(vehicleID string) (int, error) {
	if db == nil {
		return 0, fmt.Errorf("database not initialized")
	}

	var mileage int
	
	// Try buses table first
	err := db.Get(&mileage, "SELECT current_mileage FROM buses WHERE bus_id = $1", vehicleID)
	if err == nil {
		return mileage, nil
	}
	
	// Try vehicles table
	err = db.Get(&mileage, "SELECT current_mileage FROM vehicles WHERE vehicle_id = $1", vehicleID)
	if err != nil {
		return 0, fmt.Errorf("failed to get vehicle mileage: %w", err)
	}
	
	return mileage, nil
}

// UpdateVehicleMileage updates the current mileage for a vehicle
func updateVehicleMileage(vehicleID string, newMileage int) error {
	if db == nil {
		return fmt.Errorf("database not initialized")
	}

	// Try updating bus first
	result, err := db.Exec("UPDATE buses SET current_mileage = $1 WHERE bus_id = $2", newMileage, vehicleID)
	if err == nil {
		rowsAffected, _ := result.RowsAffected()
		if rowsAffected > 0 {
			return nil
		}
	}

	// Try updating vehicle
	result, err = db.Exec("UPDATE vehicles SET current_mileage = $1 WHERE vehicle_id = $2", newMileage, vehicleID)
	if err != nil {
		return fmt.Errorf("failed to update vehicle mileage: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("vehicle not found")
	}

	return nil
}

// UpdateVehicleMaintenanceStatus updates oil and tire status based on mileage
func updateVehicleMaintenanceStatus(vehicleID string, oilStatus, tireStatus string) error {
	if db == nil {
		return fmt.Errorf("database not initialized")
	}

	// Try updating bus first
	result, err := db.Exec(
		"UPDATE buses SET oil_status = $1, tire_status = $2, updated_at = CURRENT_TIMESTAMP WHERE bus_id = $3",
		oilStatus, tireStatus, vehicleID,
	)
	if err == nil {
		rowsAffected, _ := result.RowsAffected()
		if rowsAffected > 0 {
			return nil
		}
	}

	// Try updating vehicle
	result, err = db.Exec(
		"UPDATE vehicles SET oil_status = $1, tire_status = $2, updated_at = CURRENT_TIMESTAMP WHERE vehicle_id = $3",
		oilStatus, tireStatus, vehicleID,
	)
	if err != nil {
		return fmt.Errorf("failed to update vehicle status: %w", err)
	}

	return nil
}

// GetVehicleMaintenanceInfo gets maintenance information for a vehicle
func getVehicleMaintenanceInfo(vehicleID string) (currentMileage, lastOilChange, lastTireService int, err error) {
	if db == nil {
		return 0, 0, 0, fmt.Errorf("database not initialized")
	}

	// Try buses table first
	err = db.QueryRow(
		"SELECT current_mileage, last_oil_change, last_tire_service FROM buses WHERE bus_id = $1",
		vehicleID,
	).Scan(&currentMileage, &lastOilChange, &lastTireService)
	
	if err == nil {
		return currentMileage, lastOilChange, lastTireService, nil
	}

	// Try vehicles table
	err = db.QueryRow(
		"SELECT current_mileage, last_oil_change, last_tire_service FROM vehicles WHERE vehicle_id = $1",
		vehicleID,
	).Scan(&currentMileage, &lastOilChange, &lastTireService)
	
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to get vehicle maintenance info: %w", err)
	}

	return currentMileage, lastOilChange, lastTireService, nil
}

// UpdateLastServiceMileage updates the last service mileage for a specific maintenance type
func updateLastServiceMileage(vehicleID string, serviceType string, mileage int) error {
	if db == nil {
		return fmt.Errorf("database not initialized")
	}

	var query string
	switch serviceType {
	case "oil_change":
		query = "UPDATE buses SET last_oil_change = $1 WHERE bus_id = $2"
	case "tire_service":
		query = "UPDATE buses SET last_tire_service = $1 WHERE bus_id = $2"
	default:
		return fmt.Errorf("unknown service type: %s", serviceType)
	}

	// Try updating bus first
	result, err := db.Exec(query, mileage, vehicleID)
	if err == nil {
		rowsAffected, _ := result.RowsAffected()
		if rowsAffected > 0 {
			return nil
		}
	}

	// Try updating vehicle
	switch serviceType {
	case "oil_change":
		query = "UPDATE vehicles SET last_oil_change = $1 WHERE vehicle_id = $2"
	case "tire_service":
		query = "UPDATE vehicles SET last_tire_service = $1 WHERE vehicle_id = $2"
	}

	_, err = db.Exec(query, mileage, vehicleID)
	if err != nil {
		return fmt.Errorf("failed to update last service mileage: %w", err)
	}

	return nil
}

// GetLastMileageForVehicle gets the last recorded mileage for a vehicle from driver logs
func getLastMileageForVehicle(vehicleID string) (float64, error) {
	if db == nil {
		return 0, fmt.Errorf("database not initialized")
	}

	var lastMileage sql.NullFloat64
	query := `
		SELECT MAX(end_mileage) 
		FROM driver_logs 
		WHERE bus_id = $1 AND end_mileage IS NOT NULL
	`
	
	err := db.Get(&lastMileage, query, vehicleID)
	if err != nil {
		return 0, fmt.Errorf("failed to get last mileage: %w", err)
	}

	if !lastMileage.Valid {
		return 0, nil
	}

	return lastMileage.Float64, nil
}
