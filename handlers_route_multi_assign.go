package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

// Handle assigning multiple routes to a driver with the same bus
func handleMultiRouteAssign(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse form data
	err := r.ParseForm()
	if err != nil {
		log.Printf("Error parsing form: %v", err)
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	driver := r.FormValue("driver")
	busID := r.FormValue("bus_id")
	routeIDs := r.Form["route_ids[]"] // Get all selected routes

	// If no array, try single route_id
	if len(routeIDs) == 0 {
		singleRoute := r.FormValue("route_id")
		if singleRoute != "" {
			routeIDs = []string{singleRoute}
		}
	}

	log.Printf("Multi-assign: Driver=%s, Bus=%s, Routes=%v", driver, busID, routeIDs)

	if driver == "" || busID == "" || len(routeIDs) == 0 {
		log.Printf("Missing fields: driver=%s, bus=%s, routes=%d", driver, busID, len(routeIDs))
		http.Redirect(w, r, "/assign-routes?error=missing_fields", http.StatusSeeOther)
		return
	}

	// Start transaction
	tx, err := db.Begin()
	if err != nil {
		log.Printf("Error starting transaction: %v", err)
		http.Redirect(w, r, "/assign-routes?error=database", http.StatusSeeOther)
		return
	}
	defer tx.Rollback()

	// First, get existing routes for this driver to preserve any we're not changing
	var existingRoutes []string
	rows, err := tx.Query(`
		SELECT route_id 
		FROM route_assignments 
		WHERE driver = $1 AND bus_id = $2
	`, driver, busID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var routeID string
			rows.Scan(&routeID)
			existingRoutes = append(existingRoutes, routeID)
		}
	}

	// Delete only the routes that are being reassigned
	for _, routeID := range routeIDs {
		_, err = tx.Exec(`
			DELETE FROM route_assignments 
			WHERE route_id = $1
		`, routeID)
		if err != nil {
			log.Printf("Error clearing route %s: %v", routeID, err)
		}
	}

	// Insert new assignments
	successCount := 0
	for _, routeID := range routeIDs {
		routeID = strings.TrimSpace(routeID)
		if routeID == "" {
			continue
		}

		_, err = tx.Exec(`
			INSERT INTO route_assignments (driver, bus_id, route_id, assigned_date)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (driver, route_id) DO UPDATE SET
				bus_id = EXCLUDED.bus_id,
				assigned_date = EXCLUDED.assigned_date
		`, driver, busID, routeID, time.Now())

		if err != nil {
			log.Printf("Error assigning route %s: %v", routeID, err)
			// Try without ON CONFLICT
			_, err2 := tx.Exec(`
				DELETE FROM route_assignments WHERE driver = $1 AND route_id = $2;
				INSERT INTO route_assignments (driver, bus_id, route_id, assigned_date)
				VALUES ($1, $3, $2, $4)
			`, driver, routeID, busID, time.Now())
			if err2 != nil {
				log.Printf("Fallback also failed for route %s: %v", routeID, err2)
			} else {
				successCount++
			}
		} else {
			successCount++
		}
	}

	// Commit transaction
	if successCount > 0 {
		err = tx.Commit()
		if err != nil {
			log.Printf("Error committing transaction: %v", err)
			http.Redirect(w, r, "/assign-routes?error=commit_failed", http.StatusSeeOther)
			return
		}
		log.Printf("Successfully assigned %d routes to driver %s with bus %s", successCount, driver, busID)
		http.Redirect(w, r, fmt.Sprintf("/assign-routes?success=1&count=%d", successCount), http.StatusSeeOther)
	} else {
		http.Redirect(w, r, "/assign-routes?error=no_assignments", http.StatusSeeOther)
	}
}

// API endpoint to get driver's current assignments
func getDriverAssignmentsAPI(w http.ResponseWriter, r *http.Request) {
	driver := r.URL.Query().Get("driver")
	if driver == "" {
		http.Error(w, "Driver required", http.StatusBadRequest)
		return
	}

	type Assignment struct {
		RouteID   string `json:"route_id"`
		RouteName string `json:"route_name"`
		BusID     string `json:"bus_id"`
	}

	var assignments []Assignment

	rows, err := db.Query(`
		SELECT ra.route_id, ra.bus_id, r.route_name
		FROM route_assignments ra
		LEFT JOIN routes r ON ra.route_id = r.route_id
		WHERE ra.driver = $1
		ORDER BY r.route_name
	`, driver)

	if err != nil {
		log.Printf("Error fetching driver assignments: %v", err)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]Assignment{})
		return
	}
	defer rows.Close()

	for rows.Next() {
		var a Assignment
		var routeName sql.NullString
		err := rows.Scan(&a.RouteID, &a.BusID, &routeName)
		if err != nil {
			continue
		}
		if routeName.Valid {
			a.RouteName = routeName.String
		}
		assignments = append(assignments, a)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(assignments)
}