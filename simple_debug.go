package main

import (
	"fmt"
	"net/http"
)

func init() {
	// This runs when the program starts
	http.HandleFunc("/test-db", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		
		fmt.Fprintf(w, "Database Test Page\n")
		fmt.Fprintf(w, "==================\n\n")
		
		// Test 1: Is database connected?
		if db == nil {
			fmt.Fprintf(w, "ERROR: Database is NOT connected!\n")
			return
		}
		fmt.Fprintf(w, "✓ Database is connected\n\n")
		
		// Test 2: Can we query anything?
		var test int
		err := db.Get(&test, "SELECT 1")
		if err != nil {
			fmt.Fprintf(w, "ERROR: Cannot query database: %v\n", err)
			return
		}
		fmt.Fprintf(w, "✓ Database queries work\n\n")
		
		// Test 3: Count buses
		var busCount int
		err = db.Get(&busCount, "SELECT COUNT(*) FROM buses")
		if err != nil {
			fmt.Fprintf(w, "ERROR counting buses: %v\n", err)
		} else {
			fmt.Fprintf(w, "✓ Buses table has %d records\n", busCount)
		}
		
		// Test 4: Count vehicles
		var vehCount int
		err = db.Get(&vehCount, "SELECT COUNT(*) FROM vehicles")
		if err != nil {
			fmt.Fprintf(w, "ERROR counting vehicles: %v\n", err)
		} else {
			fmt.Fprintf(w, "✓ Vehicles table has %d records\n", vehCount)
		}
		
		// Test 5: List all tables
		fmt.Fprintf(w, "\nAll tables in database:\n")
		rows, err := db.Query("SELECT table_name FROM information_schema.tables WHERE table_schema = 'public' ORDER BY table_name")
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var name string
				if rows.Scan(&name) == nil {
					fmt.Fprintf(w, "  - %s\n", name)
				}
			}
		}
	})
}