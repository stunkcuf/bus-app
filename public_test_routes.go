package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strings"
)

// Register public test routes that bypass all middleware
func setupPublicTestRoutes(mux *http.ServeMux) {
	log.Printf("DEBUG: Registering public test routes")
	
	// Simple test endpoint
	mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintf(w, "Test endpoint working!\n")
		fmt.Fprintf(w, "Database status: %v\n", db != nil)
		fmt.Fprintf(w, "Request path: %s\n", r.URL.Path)
		fmt.Fprintf(w, "Request method: %s\n", r.Method)
	})
	
	// Direct fleet test
	mux.HandleFunc("/test-fleet-simple", func(w http.ResponseWriter, r *http.Request) {
		// Skip all authentication and middleware
		user := &User{Username: "test", Role: "manager"}
		
		// Load buses
		buses, err := loadBusesFromDB()
		if err != nil {
			http.Error(w, fmt.Sprintf("Bus error: %v", err), 500)
			return
		}
		
		// Load vehicles
		vehicles, err := loadVehiclesFromDB()
		if err != nil {
			http.Error(w, fmt.Sprintf("Vehicle error: %v", err), 500)
			return
		}
		
		// Create simple HTML response
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, "<h1>Fleet Test (Simple)</h1>")
		fmt.Fprintf(w, "<p>User: %s (Role: %s)</p>", user.Username, user.Role)
		fmt.Fprintf(w, "<p>Buses loaded: %d</p>", len(buses))
		fmt.Fprintf(w, "<p>Vehicles loaded: %d</p>", len(vehicles))
		fmt.Fprintf(w, "<p>Total: %d</p>", len(buses)+len(vehicles))
		
		fmt.Fprintf(w, "<h2>Sample Data</h2>")
		if len(buses) > 0 {
			b := buses[0]
			fmt.Fprintf(w, "<p>First bus: ID=%s, Status=%s</p>", b.BusID, b.Status)
		}
		if len(vehicles) > 0 {
			v := vehicles[0]
			status := "NULL"
			if v.Status.Valid {
				status = v.Status.String
			}
			fmt.Fprintf(w, "<p>First vehicle: ID=%s, Status=%s</p>", v.VehicleID, status)
		}
	})
	
	// Public status endpoint
	mux.HandleFunc("/public/status", publicStatusHandler)
	mux.HandleFunc("/public/fleet-test", publicFleetTestHandler)
	mux.HandleFunc("/public/db-test", publicDBTestHandler)
	mux.HandleFunc("/public/fleet-render", publicFleetRenderHandler)
}

func publicStatusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, "<h1>System Status (Public)</h1><pre>")
	
	// Database check
	if db == nil {
		fmt.Fprintf(w, "ERROR: Database is nil\n")
		return
	}
	
	// Test query
	var test int
	err := db.QueryRow("SELECT 1").Scan(&test)
	if err != nil {
		fmt.Fprintf(w, "ERROR: Database query failed: %v\n", err)
		return
	}
	
	fmt.Fprintf(w, "✓ Database connected and working\n\n")
	
	// Count records
	var counts []struct {
		Table string
		Count int
		Error error
	}
	
	tables := []string{"buses", "vehicles", "users", "students", "routes", "ecse_students", "maintenance_records"}
	
	for _, table := range tables {
		var count int
		err := db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&count)
		counts = append(counts, struct {
			Table string
			Count int
			Error error
		}{table, count, err})
	}
	
	fmt.Fprintf(w, "Table Counts:\n")
	for _, c := range counts {
		if c.Error != nil {
			fmt.Fprintf(w, "  %-20s: ERROR - %v\n", c.Table, c.Error)
		} else {
			fmt.Fprintf(w, "  %-20s: %d records\n", c.Table, c.Count)
		}
	}
	
	fmt.Fprintf(w, "</pre>")
}

func publicDBTestHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, "<h1>Database Test (Public)</h1><pre>")
	
	// Test 1: Direct bus query
	fmt.Fprintf(w, "Test 1: Query buses table\n")
	rows, err := db.Query("SELECT id, bus_id, status, model FROM buses LIMIT 3")
	if err != nil {
		fmt.Fprintf(w, "ERROR: %v\n\n", err)
	} else {
		defer rows.Close()
		count := 0
		for rows.Next() {
			var id int
			var busID, status string
			var model sql.NullString
			err := rows.Scan(&id, &busID, &status, &model)
			if err != nil {
				fmt.Fprintf(w, "  Scan error: %v\n", err)
			} else {
				fmt.Fprintf(w, "  ID=%d, BusID=%s, Status=%s, Model=%s\n", 
					id, busID, status, model.String)
				count++
			}
		}
		fmt.Fprintf(w, "  Total: %d buses shown\n\n", count)
	}
	
	// Test 2: Direct vehicle query
	fmt.Fprintf(w, "Test 2: Query vehicles table\n")
	rows2, err := db.Query("SELECT id, vehicle_id, status, model FROM vehicles LIMIT 3")
	if err != nil {
		fmt.Fprintf(w, "ERROR: %v\n\n", err)
	} else {
		defer rows2.Close()
		count := 0
		for rows2.Next() {
			var id int
			var vehicleID string
			var status, model sql.NullString
			err := rows2.Scan(&id, &vehicleID, &status, &model)
			if err != nil {
				fmt.Fprintf(w, "  Scan error: %v\n", err)
			} else {
				statusStr := "NULL"
				if status.Valid {
					statusStr = status.String
				}
				modelStr := "NULL"
				if model.Valid {
					modelStr = model.String
				}
				fmt.Fprintf(w, "  ID=%d, VehicleID=%s, Status=%s, Model=%s\n", 
					id, vehicleID, statusStr, modelStr)
				count++
			}
		}
		fmt.Fprintf(w, "  Total: %d vehicles shown\n", count)
	}
	
	fmt.Fprintf(w, "</pre>")
}

func publicFleetTestHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	
	fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head>
<title>Fleet Test (Public)</title>
<style>
.error { color: red; }
.success { color: green; }
.data { background: #f0f0f0; padding: 10px; margin: 10px 0; }
</style>
</head>
<body>
<h1>Fleet Test (Public Access)</h1>
`)
	
	// Test loading functions
	fmt.Fprintf(w, "<h2>1. Testing loadBusesFromDB()</h2>")
	buses, err := loadBusesFromDB()
	if err != nil {
		fmt.Fprintf(w, "<p class='error'>ERROR: %v</p>", err)
	} else {
		fmt.Fprintf(w, "<p class='success'>SUCCESS: Loaded %d buses</p>", len(buses))
		if len(buses) > 0 {
			fmt.Fprintf(w, "<div class='data'>Sample bus: %+v</div>", buses[0])
		}
	}
	
	fmt.Fprintf(w, "<h2>2. Testing loadVehiclesFromDB()</h2>")
	vehicles, err := loadVehiclesFromDB()
	if err != nil {
		fmt.Fprintf(w, "<p class='error'>ERROR: %v</p>", err)
	} else {
		fmt.Fprintf(w, "<p class='success'>SUCCESS: Loaded %d vehicles</p>", len(vehicles))
		if len(vehicles) > 0 {
			// Show first vehicle safely
			v := vehicles[0]
			fmt.Fprintf(w, "<div class='data'>")
			fmt.Fprintf(w, "ID: %d<br>", v.ID)
			fmt.Fprintf(w, "VehicleID: %s<br>", v.VehicleID)
			fmt.Fprintf(w, "Status: %s<br>", func() string {
				if v.Status.Valid {
					return v.Status.String
				}
				return "NULL"
			}())
			fmt.Fprintf(w, "Model: %s<br>", func() string {
				if v.Model.Valid {
					return v.Model.String
				}
				return "NULL"
			}())
			fmt.Fprintf(w, "</div>")
		}
	}
	
	fmt.Fprintf(w, "<h2>3. Testing loadAllFleetVehiclesFromDB()</h2>")
	allVehicles, err := loadAllFleetVehiclesFromDB()
	if err != nil {
		fmt.Fprintf(w, "<p class='error'>ERROR: %v</p>", err)
		fmt.Fprintf(w, "<p>This is likely why the fleet page fails!</p>")
	} else {
		fmt.Fprintf(w, "<p class='success'>SUCCESS: Loaded %d total vehicles</p>", len(allVehicles))
	}
	
	fmt.Fprintf(w, "</body></html>")
}

func publicFleetRenderHandler(w http.ResponseWriter, r *http.Request) {
	// Create a minimal working fleet page without authentication
	
	// Load vehicles directly
	var allVehicles []ConsolidatedVehicle
	
	// Load buses
	buses, err := loadBusesFromDB()
	if err == nil {
		for _, bus := range buses {
			cv := ConsolidatedVehicle{
				ID:               bus.ID,
				VehicleID:        bus.BusID,
				BusID:            bus.BusID,
				VehicleType:      "bus",
				Status:           bus.Status,
				Model:            bus.Model,
				Capacity:         bus.Capacity,
				OilStatus:        bus.OilStatus,
				TireStatus:       bus.TireStatus,
				MaintenanceNotes: bus.MaintenanceNotes,
			}
			allVehicles = append(allVehicles, cv)
		}
	}
	
	// Load vehicles
	vehicles, err := loadVehiclesFromDB()
	if err == nil {
		for _, veh := range vehicles {
			status := "active"
			if veh.Status.Valid {
				status = veh.Status.String
			}
			
			cv := ConsolidatedVehicle{
				ID:               veh.ID,
				VehicleID:        veh.VehicleID,
				BusID:            veh.VehicleID,
				VehicleType:      "vehicle",
				Status:           status,
				Model:            veh.Model,
				Year:             veh.Year,
				TireSize:         veh.TireSize,
				License:          veh.License,
				OilStatus:        veh.OilStatus,
				TireStatus:       veh.TireStatus,
				MaintenanceNotes: veh.MaintenanceNotes,
			}
			allVehicles = append(allVehicles, cv)
		}
	}
	
	// Render simple HTML
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head>
<title>Fleet Render Test</title>
<style>
table { border-collapse: collapse; width: 100%%; }
th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
th { background-color: #f2f2f2; }
.bus { background-color: #e8f5e9; }
.vehicle { background-color: #e3f2fd; }
</style>
</head>
<body>
<h1>Fleet Vehicles (Public Test)</h1>
<p>Total vehicles: %d</p>
<table>
<tr>
<th>Type</th>
<th>ID</th>
<th>Status</th>
<th>Model</th>
<th>Oil Status</th>
<th>Tire Status</th>
</tr>
`, len(allVehicles))
	
	for _, v := range allVehicles {
		model := "N/A"
		if v.Model.Valid {
			model = v.Model.String
		}
		oilStatus := "N/A"
		if v.OilStatus.Valid {
			oilStatus = v.OilStatus.String
		}
		tireStatus := "N/A"
		if v.TireStatus.Valid {
			tireStatus = v.TireStatus.String
		}
		
		fmt.Fprintf(w, `<tr class="%s">
<td>%s</td>
<td>%s</td>
<td>%s</td>
<td>%s</td>
<td>%s</td>
<td>%s</td>
</tr>
`, v.VehicleType, strings.Title(v.VehicleType), v.VehicleID, v.Status, model, oilStatus, tireStatus)
	}
	
	fmt.Fprintf(w, "</table></body></html>")
}