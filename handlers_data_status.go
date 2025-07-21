package main

import (
	"encoding/json"
	"log"
	"net/http"
)

// dataStatusHandler returns JSON with current data counts
func dataStatusHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if db == nil {
		http.Error(w, "Database not initialized", http.StatusInternalServerError)
		return
	}

	status := make(map[string]interface{})

	// Check ECSE students
	var ecseCount int
	err := db.QueryRow("SELECT COUNT(*) FROM ecse_students").Scan(&ecseCount)
	if err != nil {
		log.Printf("Error counting ECSE students: %v", err)
		status["ecse_students_error"] = err.Error()
	} else {
		status["ecse_students_count"] = ecseCount
	}

	// Check fuel records
	var fuelCount int
	err = db.QueryRow("SELECT COUNT(*) FROM fuel_records").Scan(&fuelCount)
	if err != nil {
		log.Printf("Error counting fuel records: %v", err)
		status["fuel_records_error"] = err.Error()
	} else {
		status["fuel_records_count"] = fuelCount
	}

	// Check maintenance logs
	var maintCount int
	err = db.QueryRow("SELECT COUNT(*) FROM bus_maintenance_logs").Scan(&maintCount)
	if err != nil {
		log.Printf("Error counting maintenance logs: %v", err)
		status["maintenance_logs_error"] = err.Error()
	} else {
		status["maintenance_logs_count"] = maintCount
	}

	// Check vehicles
	var vehicleCount int
	err = db.QueryRow("SELECT COUNT(*) FROM vehicles").Scan(&vehicleCount)
	if err != nil {
		log.Printf("Error counting vehicles: %v", err)
		status["vehicles_error"] = err.Error()
	} else {
		status["vehicles_count"] = vehicleCount
	}

	// Check fleet vehicles
	var fleetCount int
	err = db.QueryRow("SELECT COUNT(*) FROM fleet_vehicles").Scan(&fleetCount)
	if err != nil {
		log.Printf("Error counting fleet vehicles: %v", err)
		status["fleet_vehicles_error"] = err.Error()
	} else {
		status["fleet_vehicles_count"] = fleetCount
	}

	// Check buses
	var busCount int
	err = db.QueryRow("SELECT COUNT(*) FROM buses").Scan(&busCount)
	if err != nil {
		log.Printf("Error counting buses: %v", err)
		status["buses_error"] = err.Error()
	} else {
		status["buses_count"] = busCount
	}

	// Check driver logs
	var logCount int
	err = db.QueryRow("SELECT COUNT(*) FROM driver_logs").Scan(&logCount)
	if err != nil {
		log.Printf("Error counting driver logs: %v", err)
		status["driver_logs_error"] = err.Error()
	} else {
		status["driver_logs_count"] = logCount
	}

	// Check mileage reports
	var mileageCount int
	err = db.QueryRow("SELECT COUNT(*) FROM mileage_reports").Scan(&mileageCount)
	if err != nil {
		log.Printf("Error counting mileage reports: %v", err)
		status["mileage_reports_error"] = err.Error()
	} else {
		status["mileage_reports_count"] = mileageCount
	}

	// Return as JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}