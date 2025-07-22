package main

import (
	"fmt"
	"net/http"
)

func init() {
	// Add debug endpoint for fleet
	http.HandleFunc("/debug-fleet", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		
		fmt.Fprintf(w, "Fleet Debug Information\n")
		fmt.Fprintf(w, "======================\n\n")
		
		// Test 1: Database connection
		fmt.Fprintf(w, "1. Database Connection Test:\n")
		if db == nil {
			fmt.Fprintf(w, "   ERROR: Database is nil!\n")
			return
		}
		fmt.Fprintf(w, "   ✓ Database connected\n\n")
		
		// Test 2: Count records in tables
		fmt.Fprintf(w, "2. Table Record Counts:\n")
		var busCount, vehCount, fleetCount int
		
		err := db.QueryRow("SELECT COUNT(*) FROM buses").Scan(&busCount)
		if err != nil {
			fmt.Fprintf(w, "   ERROR counting buses: %v\n", err)
		} else {
			fmt.Fprintf(w, "   Buses table: %d records\n", busCount)
		}
		
		err = db.QueryRow("SELECT COUNT(*) FROM vehicles").Scan(&vehCount)
		if err != nil {
			fmt.Fprintf(w, "   ERROR counting vehicles: %v\n", err)
		} else {
			fmt.Fprintf(w, "   Vehicles table: %d records\n", vehCount)
		}
		
		// Check if fleet_vehicles exists
		err = db.QueryRow("SELECT COUNT(*) FROM fleet_vehicles").Scan(&fleetCount)
		if err != nil {
			fmt.Fprintf(w, "   Fleet_vehicles table: ERROR - %v\n", err)
		} else {
			fmt.Fprintf(w, "   Fleet_vehicles table: %d records\n", fleetCount)
		}
		
		fmt.Fprintf(w, "   Total expected: %d\n\n", busCount + vehCount)
		
		// Test 3: Try loading buses
		fmt.Fprintf(w, "3. Loading Buses Test:\n")
		buses, err := loadBusesFromDB()
		if err != nil {
			fmt.Fprintf(w, "   ERROR: %v\n", err)
		} else {
			fmt.Fprintf(w, "   ✓ Loaded %d buses\n", len(buses))
			if len(buses) > 0 {
				fmt.Fprintf(w, "   Sample: %s (Status: %s)\n", buses[0].BusID, buses[0].Status)
			}
		}
		
		// Test 4: Try loading vehicles
		fmt.Fprintf(w, "\n4. Loading Vehicles Test:\n")
		vehicles, err := loadVehiclesFromDB()
		if err != nil {
			fmt.Fprintf(w, "   ERROR: %v\n", err)
		} else {
			fmt.Fprintf(w, "   ✓ Loaded %d vehicles\n", len(vehicles))
			if len(vehicles) > 0 {
				fmt.Fprintf(w, "   Sample: %s (Status: %s)\n", vehicles[0].VehicleID, vehicles[0].Status.String)
			}
		}
		
		// Test 5: Try loadAllFleetVehiclesFromDB
		fmt.Fprintf(w, "\n5. Loading All Fleet Vehicles Test:\n")
		allVehicles, err := loadAllFleetVehiclesFromDB()
		if err != nil {
			fmt.Fprintf(w, "   ERROR: %v\n", err)
		} else {
			fmt.Fprintf(w, "   ✓ Loaded %d total vehicles\n", len(allVehicles))
			
			busTypeCount := 0
			vehTypeCount := 0
			for _, v := range allVehicles {
				if v.VehicleType == "bus" {
					busTypeCount++
				} else if v.VehicleType == "vehicle" {
					vehTypeCount++
				}
			}
			fmt.Fprintf(w, "   - Buses: %d\n", busTypeCount)
			fmt.Fprintf(w, "   - Vehicles: %d\n", vehTypeCount)
		}
		
		// Test 6: Check cache
		fmt.Fprintf(w, "\n6. Cache Status:\n")
		if dataCache != nil {
			cachedBuses, err := dataCache.getBuses()
			if err == nil {
				fmt.Fprintf(w, "   Cached buses: %d\n", len(cachedBuses))
			} else {
				fmt.Fprintf(w, "   Cache error: %v\n", err)
			}
		} else {
			fmt.Fprintf(w, "   Cache is nil\n")
		}
		
		// Test 7: Check which handler is mapped to /fleet
		fmt.Fprintf(w, "\n7. Route Mapping:\n")
		fmt.Fprintf(w, "   /fleet -> fleetHandler (should display all vehicles)\n")
		fmt.Fprintf(w, "   /company-fleet -> companyFleetHandler (non-bus vehicles)\n")
		fmt.Fprintf(w, "   /fleet-vehicles -> fleetVehiclesHandler (different feature)\n")
		
		// Test 8: Sample SQL queries
		fmt.Fprintf(w, "\n8. Direct SQL Test:\n")
		rows, err := db.Query("SELECT bus_id, status FROM buses LIMIT 3")
		if err != nil {
			fmt.Fprintf(w, "   ERROR querying buses: %v\n", err)
		} else {
			defer rows.Close()
			fmt.Fprintf(w, "   Sample buses:\n")
			for rows.Next() {
				var busID, status string
				rows.Scan(&busID, &status)
				fmt.Fprintf(w, "   - %s (status: %s)\n", busID, status)
			}
		}
	})
}