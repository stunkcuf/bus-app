package main

import (
	"log"
	"net/http"
)

// testFleetHandler is a simplified fleet handler for debugging
func testFleetHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	log.Printf("DEBUG: Test fleet handler called by user: %s", user.Username)

	// Simple data structure for testing
	data := map[string]interface{}{
		"User":      user,
		"CSRFToken": getSessionCSRFToken(r),
		"PageTitle": "Fleet Overview Test",
		"Debug":     true,
	}

	// Try to get buses directly
	if db != nil {
		var busCount int
		err := db.Get(&busCount, "SELECT COUNT(*) FROM buses")
		if err != nil {
			log.Printf("ERROR counting buses: %v", err)
			data["BusError"] = err.Error()
		} else {
			data["BusCount"] = busCount
			
			// Get sample buses
			var buses []Bus
			err = db.Select(&buses, "SELECT * FROM buses LIMIT 10")
			if err != nil {
				log.Printf("ERROR loading buses: %v", err)
				data["LoadError"] = err.Error()
			} else {
				data["Buses"] = buses
				log.Printf("DEBUG: Loaded %d buses", len(buses))
			}
		}

		// Get vehicle count
		var vehicleCount int
		err = db.Get(&vehicleCount, "SELECT COUNT(*) FROM vehicles")
		if err != nil {
			log.Printf("ERROR counting vehicles: %v", err)
			data["VehicleError"] = err.Error()
		} else {
			data["VehicleCount"] = vehicleCount
			
			// Get sample vehicles
			var vehicles []Vehicle
			err = db.Select(&vehicles, "SELECT * FROM vehicles WHERE vehicle_type != 'Bus' LIMIT 10")
			if err != nil {
				log.Printf("ERROR loading vehicles: %v", err)
			} else {
				data["Vehicles"] = vehicles
				log.Printf("DEBUG: Loaded %d non-bus vehicles", len(vehicles))
			}
		}
	} else {
		data["DatabaseError"] = "Database not connected"
	}

	// Use a simple template for testing
	renderTemplate(w, r, "test_fleet.html", data)
}