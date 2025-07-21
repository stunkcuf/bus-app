package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

func init() {
	// Direct fleet test - bypasses all complexity
	http.HandleFunc("/test-fleet-direct", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		
		fmt.Fprintf(w, "<h1>Direct Fleet Test</h1><pre>")
		
		// Test 1: Database
		fmt.Fprintf(w, "1. Database Check:\n")
		if db == nil {
			fmt.Fprintf(w, "   ERROR: Database is nil!\n")
			return
		}
		fmt.Fprintf(w, "   OK: Database connected\n\n")
		
		// Test 2: Direct SQL query buses
		fmt.Fprintf(w, "2. Direct SQL Query - Buses:\n")
		rows, err := db.Query("SELECT bus_id, status, model FROM buses LIMIT 5")
		if err != nil {
			fmt.Fprintf(w, "   ERROR: %v\n", err)
		} else {
			defer rows.Close()
			count := 0
			for rows.Next() {
				var busID, status string
				var model sql.NullString
				err := rows.Scan(&busID, &status, &model)
				if err != nil {
					fmt.Fprintf(w, "   Scan error: %v\n", err)
				} else {
					fmt.Fprintf(w, "   Bus: %s, Status: %s, Model: %s\n", busID, status, model.String)
					count++
				}
			}
			fmt.Fprintf(w, "   Total shown: %d\n\n", count)
		}
		
		// Test 3: Direct SQL query vehicles  
		fmt.Fprintf(w, "3. Direct SQL Query - Vehicles:\n")
		rows2, err := db.Query("SELECT vehicle_id, status, model FROM vehicles LIMIT 5")
		if err != nil {
			fmt.Fprintf(w, "   ERROR: %v\n", err)
		} else {
			defer rows2.Close()
			count := 0
			for rows2.Next() {
				var vehicleID string
				var status, model sql.NullString
				err := rows2.Scan(&vehicleID, &status, &model)
				if err != nil {
					fmt.Fprintf(w, "   Scan error: %v\n", err)
				} else {
					fmt.Fprintf(w, "   Vehicle: %s, Status: %s, Model: %s\n", 
						vehicleID, status.String, model.String)
					count++
				}
			}
			fmt.Fprintf(w, "   Total shown: %d\n\n", count)
		}
		
		// Test 4: loadBusesFromDB
		fmt.Fprintf(w, "4. Testing loadBusesFromDB():\n")
		buses, err := loadBusesFromDB()
		if err != nil {
			fmt.Fprintf(w, "   ERROR: %v\n", err)
		} else {
			fmt.Fprintf(w, "   SUCCESS: Loaded %d buses\n", len(buses))
			if len(buses) > 0 {
				fmt.Fprintf(w, "   Sample: %+v\n", buses[0])
			}
		}
		
		// Test 5: loadVehiclesFromDB
		fmt.Fprintf(w, "\n5. Testing loadVehiclesFromDB():\n")
		vehicles, err := loadVehiclesFromDB()
		if err != nil {
			fmt.Fprintf(w, "   ERROR: %v\n", err)
		} else {
			fmt.Fprintf(w, "   SUCCESS: Loaded %d vehicles\n", len(vehicles))
		}
		
		// Test 6: loadAllFleetVehiclesFromDB
		fmt.Fprintf(w, "\n6. Testing loadAllFleetVehiclesFromDB():\n")
		allVehicles, err := loadAllFleetVehiclesFromDB()
		if err != nil {
			fmt.Fprintf(w, "   ERROR: %v\n\n", err)
			fmt.Fprintf(w, "   This is likely why fleet page fails!\n")
		} else {
			fmt.Fprintf(w, "   SUCCESS: Loaded %d total vehicles\n", len(allVehicles))
		}
		
		fmt.Fprintf(w, "</pre>")
		
		// Add a button to test fleet handler
		fmt.Fprintf(w, `<hr>
<h2>Test Fleet Handler Directly</h2>
<button onclick="testFleet()">Test Fleet Handler</button>
<div id="result"></div>
<script>
function testFleet() {
    fetch('/fleet')
        .then(response => {
            document.getElementById('result').innerHTML = 
                'Status: ' + response.status + '<br>' +
                'Redirected: ' + response.redirected + '<br>' +
                'URL: ' + response.url;
            return response.text();
        })
        .then(text => {
            if (text.includes('Unable to load')) {
                document.getElementById('result').innerHTML += 
                    '<br><strong>ERROR FOUND IN RESPONSE!</strong>';
            }
        });
}
</script>`)
	})
}

// Also create a minimal fleet page that bypasses everything
func minimalFleetHandler(w http.ResponseWriter, r *http.Request) {
	// Skip all checks - just try to load data
	w.Header().Set("Content-Type", "text/html")
	
	fmt.Fprintf(w, "<h1>Minimal Fleet Test</h1>")
	
	// Try to load buses
	var buses []Bus
	err := db.Select(&buses, "SELECT * FROM buses")
	if err != nil {
		fmt.Fprintf(w, "<p>Error loading buses: %v</p>", err)
		return
	}
	
	// Try to load vehicles
	var vehicles []Vehicle  
	err = db.Select(&vehicles, "SELECT * FROM vehicles")
	if err != nil {
		fmt.Fprintf(w, "<p>Error loading vehicles: %v</p>", err)
		return
	}
	
	fmt.Fprintf(w, "<h2>Fleet Summary</h2>")
	fmt.Fprintf(w, "<p>Buses: %d</p>", len(buses))
	fmt.Fprintf(w, "<p>Vehicles: %d</p>", len(vehicles))
	fmt.Fprintf(w, "<p>Total: %d</p>", len(buses)+len(vehicles))
	
	fmt.Fprintf(w, "<h3>Sample Data</h3><pre>")
	if len(buses) > 0 {
		fmt.Fprintf(w, "First bus: %+v\n", buses[0])
	}
	if len(vehicles) > 0 {
		fmt.Fprintf(w, "First vehicle: %+v\n", vehicles[0])
	}
	fmt.Fprintf(w, "</pre>")
}

func init() {
	http.HandleFunc("/minimal-fleet", minimalFleetHandler)
}