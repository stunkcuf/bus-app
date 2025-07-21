package main

import (
	"fmt"
	"net/http"
)

func init() {
	http.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		
		fmt.Fprintf(w, "System Status\n")
		fmt.Fprintf(w, "=============\n\n")
		
		// Database check
		if db == nil {
			fmt.Fprintf(w, "Database: NOT CONNECTED\n")
			return
		}
		fmt.Fprintf(w, "Database: Connected\n")
		
		// Count records
		var busCount, vehCount, userCount int
		db.QueryRow("SELECT COUNT(*) FROM buses").Scan(&busCount)
		db.QueryRow("SELECT COUNT(*) FROM vehicles").Scan(&vehCount)
		db.QueryRow("SELECT COUNT(*) FROM users").Scan(&userCount)
		
		fmt.Fprintf(w, "\nRecord Counts:\n")
		fmt.Fprintf(w, "- Buses: %d\n", busCount)
		fmt.Fprintf(w, "- Vehicles: %d\n", vehCount)
		fmt.Fprintf(w, "- Total Fleet: %d\n", busCount + vehCount)
		fmt.Fprintf(w, "- Users: %d\n", userCount)
		
		// Test key functions
		fmt.Fprintf(w, "\nFunction Tests:\n")
		
		buses, err := loadBusesFromDB()
		fmt.Fprintf(w, "- loadBusesFromDB: %d records (error: %v)\n", len(buses), err)
		
		vehicles, err := loadVehiclesFromDB()
		fmt.Fprintf(w, "- loadVehiclesFromDB: %d records (error: %v)\n", len(vehicles), err)
		
		allVehicles, err := loadAllFleetVehiclesFromDB()
		fmt.Fprintf(w, "- loadAllFleetVehiclesFromDB: %d records (error: %v)\n", len(allVehicles), err)
		
		fmt.Fprintf(w, "\nServer Port: 5003 (default)\n")
	})
}