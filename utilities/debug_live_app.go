package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
)

// Test handler to debug the live application
func debugFleetDataHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, "<html><body>")
	fmt.Fprintf(w, "<h1>Fleet Data Debug</h1>")
	fmt.Fprintf(w, "<pre>")
	
	// Check database connection
	if db == nil {
		fmt.Fprintf(w, "ERROR: Database is nil!\n")
	} else {
		fmt.Fprintf(w, "✓ Database connection exists\n")
		
		// Test ping
		if err := db.Ping(); err != nil {
			fmt.Fprintf(w, "ERROR: Database ping failed: %v\n", err)
		} else {
			fmt.Fprintf(w, "✓ Database ping successful\n")
		}
	}
	
	// Count buses
	var busCount int
	err := db.QueryRow("SELECT COUNT(*) FROM buses").Scan(&busCount)
	if err != nil {
		fmt.Fprintf(w, "ERROR counting buses: %v\n", err)
	} else {
		fmt.Fprintf(w, "✓ Total buses in database: %d\n", busCount)
	}
	
	// Try loadBusesFromDBPaginated
	fmt.Fprintf(w, "\nTesting loadBusesFromDBPaginated:\n")
	pagination := PaginationParams{
		Page:    1,
		PerPage: 10,
		Offset:  0,
	}
	
	buses, err := loadBusesFromDBPaginated(pagination)
	if err != nil {
		fmt.Fprintf(w, "ERROR: loadBusesFromDBPaginated failed: %v\n", err)
	} else {
		fmt.Fprintf(w, "✓ loadBusesFromDBPaginated returned %d buses\n", len(buses))
		for i, bus := range buses {
			fmt.Fprintf(w, "  Bus %d: ID=%s, Status=%s, Model=%s\n", 
				i+1, bus.BusID, bus.Status, bus.Model.String)
		}
	}
	
	// Try getBusCount
	count, err := getBusCount()
	if err != nil {
		fmt.Fprintf(w, "ERROR: getBusCount failed: %v\n", err)
	} else {
		fmt.Fprintf(w, "✓ getBusCount returned: %d\n", count)
	}
	
	// Check what fleetHandler would see
	fmt.Fprintf(w, "\nSimulating fleetHandler behavior:\n")
	allVehicles := []ConsolidatedVehicle{}
	
	// Load buses
	if buses != nil && len(buses) > 0 {
		for _, bus := range buses {
			cv := ConsolidatedVehicle{
				ID:               bus.BusID,
				VehicleID:        bus.BusID,
				BusID:            bus.BusID,
				VehicleType:      "bus",
				Status:           bus.Status,
				Model:            bus.Model,
				Capacity:         bus.Capacity,
				OilStatus:        bus.OilStatus,
				TireStatus:       bus.TireStatus,
				MaintenanceNotes: bus.MaintenanceNotes,
				UpdatedAt:        bus.UpdatedAt,
				CreatedAt:        bus.CreatedAt,
			}
			allVehicles = append(allVehicles, cv)
		}
	}
	
	fmt.Fprintf(w, "✓ Total vehicles for display: %d\n", len(allVehicles))
	
	fmt.Fprintf(w, "</pre>")
	fmt.Fprintf(w, "<p><a href='/fleet'>Go to Fleet Page</a></p>")
	fmt.Fprintf(w, "</body></html>")
}

func main() {
	// Load environment
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found: %v", err)
	}

	// Setup database
	log.Println("Setting up database...")
	if err := setupDatabase(); err != nil {
		log.Fatalf("Failed to setup database: %v", err)
	}
	defer closeDatabase()

	// Add debug route
	http.HandleFunc("/debug-fleet", debugFleetDataHandler)
	
	// Start simple server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	
	log.Printf("Debug server starting on port %s", port)
	log.Printf("Visit http://localhost:%s/debug-fleet to debug fleet data", port)
	
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}