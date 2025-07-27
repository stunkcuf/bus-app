package main

import (
	"encoding/json"
	"net/http"
	"log"
)

// debugDataHandler provides a debug endpoint to check database data
func debugDataHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	debugInfo := make(map[string]interface{})

	// Check database connection
	if db == nil {
		debugInfo["database"] = "Not connected"
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(debugInfo)
		return
	}

	debugInfo["database"] = "Connected"

	// Get counts from each table
	tables := []string{"buses", "vehicles", "routes", "students", "users", "maintenance_records", "service_records"}
	counts := make(map[string]int)

	for _, table := range tables {
		var count int
		err := db.Get(&count, "SELECT COUNT(*) FROM " + table)
		if err != nil {
			log.Printf("Error counting %s: %v", table, err)
			counts[table] = -1
		} else {
			counts[table] = count
		}
	}
	debugInfo["table_counts"] = counts

	// Get sample buses
	var buses []Bus
	err := db.Select(&buses, "SELECT * FROM buses LIMIT 5")
	if err != nil {
		debugInfo["buses_error"] = err.Error()
	} else {
		debugInfo["sample_buses"] = buses
	}

	// Get sample vehicles
	var vehicles []Vehicle
	err = db.Select(&vehicles, "SELECT * FROM vehicles WHERE vehicle_type != 'Bus' LIMIT 5")
	if err != nil {
		debugInfo["vehicles_error"] = err.Error()
	} else {
		debugInfo["sample_vehicles"] = vehicles
	}

	// Check cache
	if dataCache != nil {
		busesFromCache, err := dataCache.getBuses()
		if err != nil {
			debugInfo["cache_error"] = err.Error()
		} else {
			debugInfo["cached_buses_count"] = len(busesFromCache)
		}
	} else {
		debugInfo["cache"] = "Not initialized"
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(debugInfo)
}