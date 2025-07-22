package main

import (
	"fmt"
	"log"
	"time"
	
	"github.com/lib/pq"
)

// getMaintenanceAlertsForDriver gets maintenance alerts for vehicles assigned to a driver
func getMaintenanceAlertsForDriver(driverUsername string) ([]MaintenanceAlert, error) {
	alerts := []MaintenanceAlert{}
	
	// Get driver's assigned vehicles
	assignments, err := getDriverAssignments(driverUsername)
	if err != nil {
		return alerts, fmt.Errorf("failed to get driver assignments: %w", err)
	}
	
	if len(assignments) == 0 {
		return alerts, nil
	}
	
	// Use the optimized batch loading function
	var busIDs []string
	for _, assignment := range assignments {
		busIDs = append(busIDs, assignment.BusID)
	}
	
	return getMaintenanceAlertsForBuses(busIDs)
}

// getMaintenanceAlertsForBuses gets maintenance alerts for multiple buses in a single query
func getMaintenanceAlertsForBuses(busIDs []string) ([]MaintenanceAlert, error) {
	if len(busIDs) == 0 {
		return []MaintenanceAlert{}, nil
	}
	
	alerts := []MaintenanceAlert{}
	
	// Get all buses and their maintenance info in a single query
	query := `
		WITH bus_maintenance AS (
			SELECT 
				bus_id, 
				MAX(created_at) as last_maintenance_date,
				COUNT(*) as maintenance_count
			FROM bus_maintenance_logs
			WHERE bus_id = ANY($1)
			GROUP BY bus_id
		)
		SELECT 
			b.*,
			COALESCE(bm.last_maintenance_date, '1970-01-01'::timestamp) as last_maintenance,
			EXTRACT(DAY FROM (NOW() - COALESCE(bm.last_maintenance_date, '1970-01-01'::timestamp))) as days_since_maintenance
		FROM buses b
		LEFT JOIN bus_maintenance bm ON b.bus_id = bm.bus_id
		WHERE b.bus_id = ANY($1)
	`
	
	rows, err := db.Query(query, pq.StringArray(busIDs))
	if err != nil {
		return alerts, fmt.Errorf("failed to get buses with maintenance info: %w", err)
	}
	defer rows.Close()
	
	for rows.Next() {
		var bus Bus
		var lastMaintenance time.Time
		var daysSinceMaintenance float64
		
		err := rows.Scan(
			&bus.BusID, &bus.Status, &bus.Model, &bus.Capacity,
			&bus.OilStatus, &bus.TireStatus, &bus.MaintenanceNotes,
			&bus.CurrentMileage, &bus.LastOilChange, &bus.LastTireService,
			&bus.UpdatedAt, &bus.CreatedAt,
			&lastMaintenance, &daysSinceMaintenance,
		)
		if err != nil {
			log.Printf("Error scanning bus data: %v", err)
			continue
		}
		
		// Check oil status
		if bus.OilStatus.Valid && (bus.OilStatus.String == "due" || bus.OilStatus.String == "overdue") {
			alerts = append(alerts, MaintenanceAlert{
				VehicleID:    bus.BusID,
				VehicleType:  "bus",
				AlertType:    "oil_change",
				ItemName:     "Oil Change",
				Severity:     getSeverity(bus.OilStatus.String),
				Message:      fmt.Sprintf("Bus %s oil change is %s", bus.BusID, bus.OilStatus.String),
				MilesOverdue: 0,
			})
		}
		
		// Check tire status
		if bus.TireStatus.Valid && (bus.TireStatus.String == "fair" || bus.TireStatus.String == "poor") {
			severity := "warning"
			if bus.TireStatus.String == "poor" {
				severity = "overdue"
			}
			alerts = append(alerts, MaintenanceAlert{
				VehicleID:    bus.BusID,
				VehicleType:  "bus",
				AlertType:    "tire_check",
				ItemName:     "Tire Check",
				Severity:     severity,
				Message:      fmt.Sprintf("Bus %s tires are in %s condition", bus.BusID, bus.TireStatus.String),
				MilesOverdue: 0,
			})
		}
		
		// Check maintenance notes
		if bus.MaintenanceNotes.Valid && len(bus.MaintenanceNotes.String) > 0 {
			alerts = append(alerts, MaintenanceAlert{
				VehicleID:    bus.BusID,
				VehicleType:  "bus",
				AlertType:    "maintenance_note",
				ItemName:     "Maintenance Note",
				Severity:     "warning",
				Message:      fmt.Sprintf("Bus %s has maintenance notes: %s", bus.BusID, bus.MaintenanceNotes.String),
				MilesOverdue: 0,
			})
		}
		
		// Check last maintenance date
		if daysSinceMaintenance > 90 {
			alerts = append(alerts, MaintenanceAlert{
				VehicleID:    bus.BusID,
				VehicleType:  "bus",
				AlertType:    "maintenance_overdue",
				ItemName:     "Routine Maintenance",
				Severity:     "overdue",
				Message:      fmt.Sprintf("Bus %s hasn't had maintenance in %d days", bus.BusID, int(daysSinceMaintenance)),
				MilesOverdue: 0,
			})
		} else if daysSinceMaintenance > 60 {
			alerts = append(alerts, MaintenanceAlert{
				VehicleID:    bus.BusID,
				VehicleType:  "bus",
				AlertType:    "maintenance_due",
				ItemName:     "Routine Maintenance",
				Severity:     "due",
				Message:      fmt.Sprintf("Bus %s maintenance due soon (%d days since last service)", bus.BusID, int(daysSinceMaintenance)),
				MilesOverdue: 0,
			})
		}
	}
	
	return alerts, nil
}

// Legacy function - keeping the old implementation for compatibility
func getMaintenanceAlertsForDriverLegacy(driverUsername string) ([]MaintenanceAlert, error) {
	alerts := []MaintenanceAlert{}
	
	// Get driver's assigned vehicles
	assignments, err := getDriverAssignments(driverUsername)
	if err != nil {
		return alerts, fmt.Errorf("failed to get driver assignments: %w", err)
	}
	
	for _, assignment := range assignments {
		// Get bus details
		var bus Bus
		err := db.Get(&bus, "SELECT * FROM buses WHERE bus_id = $1", assignment.BusID)
		if err != nil {
			log.Printf("Error getting bus %s: %v", assignment.BusID, err)
			continue
		}
		
		// Check oil status
		if bus.OilStatus.Valid && (bus.OilStatus.String == "due" || bus.OilStatus.String == "overdue") {
			alerts = append(alerts, MaintenanceAlert{
				VehicleID:    bus.BusID,
				VehicleType:  "bus",
				AlertType:    "oil_change",
				ItemName:     "Oil Change",
				Severity:     getSeverity(bus.OilStatus.String),
				Message:      fmt.Sprintf("Bus %s oil change is %s", bus.BusID, bus.OilStatus.String),
				MilesOverdue: 0,
			})
		}
		
		// Check tire status
		if bus.TireStatus.Valid && (bus.TireStatus.String == "fair" || bus.TireStatus.String == "poor") {
			severity := "warning"
			if bus.TireStatus.String == "poor" {
				severity = "overdue"
			}
			alerts = append(alerts, MaintenanceAlert{
				VehicleID:    bus.BusID,
				VehicleType:  "bus",
				AlertType:    "tire_check",
				ItemName:     "Tire Check",
				Severity:     severity,
				Message:      fmt.Sprintf("Bus %s tires are in %s condition", bus.BusID, bus.TireStatus.String),
				MilesOverdue: 0,
			})
		}
		
		// Check maintenance notes
		if bus.MaintenanceNotes.Valid && len(bus.MaintenanceNotes.String) > 0 {
			alerts = append(alerts, MaintenanceAlert{
				VehicleID:    bus.BusID,
				VehicleType:  "bus",
				AlertType:    "maintenance_note",
				ItemName:     "Maintenance Note",
				Severity:     "warning",
				Message:      fmt.Sprintf("Bus %s has maintenance notes: %s", bus.BusID, bus.MaintenanceNotes.String),
				MilesOverdue: 0,
			})
		}
		
		// Check last maintenance date
		var lastMaintenance time.Time
		err = db.QueryRow(`
			SELECT MAX(created_at) 
			FROM bus_maintenance_logs 
			WHERE bus_id = $1
		`, bus.BusID).Scan(&lastMaintenance)
		
		if err == nil {
			daysSinceMaintenance := int(time.Since(lastMaintenance).Hours() / 24)
			if daysSinceMaintenance > 90 {
				alerts = append(alerts, MaintenanceAlert{
					VehicleID:    bus.BusID,
					VehicleType:  "bus",
					AlertType:    "maintenance_overdue",
					ItemName:     "Routine Maintenance",
					Severity:     "overdue",
					Message:      fmt.Sprintf("Bus %s hasn't had maintenance in %d days", bus.BusID, daysSinceMaintenance),
					MilesOverdue: 0,
				})
			} else if daysSinceMaintenance > 60 {
				alerts = append(alerts, MaintenanceAlert{
					VehicleID:    bus.BusID,
					VehicleType:  "bus",
					AlertType:    "maintenance_due",
					ItemName:     "Routine Maintenance",
					Severity:     "due",
					Message:      fmt.Sprintf("Bus %s maintenance due soon (%d days since last service)", bus.BusID, daysSinceMaintenance),
					MilesOverdue: 0,
				})
			}
		}
	}
	
	return alerts, nil
}

// getSeverity converts status strings to severity levels
func getSeverity(status string) string {
	switch status {
	case "overdue", "poor", "critical":
		return "overdue"
	case "due", "fair":
		return "due"
	default:
		return "warning"
	}
}

// getMaintenanceAlertsByVehicle gets all maintenance alerts for a specific vehicle
func getMaintenanceAlertsByVehicle(vehicleID string) ([]MaintenanceAlert, error) {
	alerts := []MaintenanceAlert{}
	
	// Try to get as bus first
	var bus Bus
	err := db.Get(&bus, "SELECT * FROM buses WHERE bus_id = $1", vehicleID)
	if err == nil {
		// Check oil status
		if bus.OilStatus.Valid && (bus.OilStatus.String == "due" || bus.OilStatus.String == "overdue") {
			alerts = append(alerts, MaintenanceAlert{
				VehicleID:    bus.BusID,
				VehicleType:  "bus",
				AlertType:    "oil_change",
				ItemName:     "Oil Change",
				Severity:     getSeverity(bus.OilStatus.String),
				Message:      fmt.Sprintf("Oil change is %s", bus.OilStatus.String),
				MilesOverdue: 0,
			})
		}
		
		// Check tire status
		if bus.TireStatus.Valid && (bus.TireStatus.String == "fair" || bus.TireStatus.String == "poor") {
			severity := "warning"
			if bus.TireStatus.String == "poor" {
				severity = "overdue"
			}
			alerts = append(alerts, MaintenanceAlert{
				VehicleID:    bus.BusID,
				VehicleType:  "bus",
				AlertType:    "tire_check",
				ItemName:     "Tire Check",
				Severity:     severity,
				Message:      fmt.Sprintf("Tires are in %s condition", bus.TireStatus.String),
				MilesOverdue: 0,
			})
		}
		
		return alerts, nil
	}
	
	// Try as vehicle
	var vehicle Vehicle
	err = db.Get(&vehicle, "SELECT * FROM vehicles WHERE vehicle_id = $1", vehicleID)
	if err == nil {
		// Check oil status
		if vehicle.OilStatus.Valid && (vehicle.OilStatus.String == "due" || vehicle.OilStatus.String == "overdue") {
			alerts = append(alerts, MaintenanceAlert{
				VehicleID:    vehicle.VehicleID,
				VehicleType:  "vehicle",
				AlertType:    "oil_change",
				ItemName:     "Oil Change",
				Severity:     getSeverity(vehicle.OilStatus.String),
				Message:      fmt.Sprintf("Oil change is %s", vehicle.OilStatus.String),
				MilesOverdue: 0,
			})
		}
		
		// Check tire status
		if vehicle.TireStatus.Valid && (vehicle.TireStatus.String == "fair" || vehicle.TireStatus.String == "poor") {
			severity := "warning"
			if vehicle.TireStatus.String == "poor" {
				severity = "overdue"
			}
			alerts = append(alerts, MaintenanceAlert{
				VehicleID:    vehicle.VehicleID,
				VehicleType:  "vehicle",
				AlertType:    "tire_check",
				ItemName:     "Tire Check",
				Severity:     severity,
				Message:      fmt.Sprintf("Tires are in %s condition", vehicle.TireStatus.String),
				MilesOverdue: 0,
			})
		}
	}
	
	return alerts, nil
}