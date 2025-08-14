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

// MultiRouteAssignment represents multiple routes for a driver
type MultiRouteAssignment struct {
	Driver    string   `json:"driver"`
	BusID     string   `json:"bus_id"`
	RouteIDs  []string `json:"route_ids"`
	StartTime string   `json:"start_time"`
	EndTime   string   `json:"end_time"`
}

// Handle multiple route assignments for a single driver
func handleMultiRouteAssignment(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	if r.Method == http.MethodPost {
		// Parse the multi-route assignment
		r.ParseForm()
		
		driver := r.FormValue("driver")
		busID := r.FormValue("bus_id")
		routeIDs := r.Form["route_ids[]"] // Multiple route selections
		
		if driver == "" || busID == "" || len(routeIDs) == 0 {
			http.Error(w, "Missing required fields", http.StatusBadRequest)
			return
		}

		// Begin transaction
		tx, err := db.Begin()
		if err != nil {
			log.Printf("Error starting transaction: %v", err)
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
		defer tx.Rollback()

		// First, remove existing assignments for this driver
		_, err = tx.Exec(`
			DELETE FROM route_assignments 
			WHERE driver = $1
		`, driver)
		if err != nil {
			log.Printf("Error removing old assignments: %v", err)
			http.Error(w, "Failed to update assignments", http.StatusInternalServerError)
			return
		}

		// Insert new assignments for each route
		for _, routeID := range routeIDs {
			_, err = tx.Exec(`
				INSERT INTO route_assignments (driver, bus_id, route_id, assigned_date)
				VALUES ($1, $2, $3, $4)
			`, driver, busID, routeID, time.Now())
			
			if err != nil {
				log.Printf("Error inserting route assignment: %v", err)
				http.Error(w, "Failed to create assignment", http.StatusInternalServerError)
				return
			}
		}

		// Commit transaction
		if err = tx.Commit(); err != nil {
			log.Printf("Error committing transaction: %v", err)
			http.Error(w, "Failed to save assignments", http.StatusInternalServerError)
			return
		}

		// Return success
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": fmt.Sprintf("Assigned %d routes to driver %s", len(routeIDs), driver),
		})
		return
	}

	// GET request - show current assignments
	assignments := getMultiRouteAssignments()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(assignments)
}

// Get all multi-route assignments grouped by driver
func getMultiRouteAssignments() map[string][]RouteAssignment {
	assignments := make(map[string][]RouteAssignment)
	
	query := `
		SELECT 
			ra.driver,
			ra.bus_id,
			ra.route_id,
			r.name as route_name,
			ra.assigned_date
		FROM route_assignments ra
		LEFT JOIN routes r ON ra.route_id = r.id
		ORDER BY ra.driver, ra.route_id
	`
	
	rows, err := db.Query(query)
	if err != nil {
		log.Printf("Error fetching assignments: %v", err)
		return assignments
	}
	defer rows.Close()
	
	for rows.Next() {
		var assignment RouteAssignment
		var routeName sql.NullString
		
		err := rows.Scan(
			&assignment.Driver,
			&assignment.BusID,
			&assignment.RouteID,
			&routeName,
			&assignment.AssignedDate,
		)
		if err != nil {
			log.Printf("Error scanning assignment: %v", err)
			continue
		}
		
		if routeName.Valid {
			assignment.RouteName = routeName.String
		}
		
		assignments[assignment.Driver] = append(assignments[assignment.Driver], assignment)
	}
	
	return assignments
}

// Enhanced assign routes handler for UI
func handleAssignRoutesEnhanced(w http.ResponseWriter, r *http.Request) {
	log.Printf("handleAssignRoutesEnhanced called - Method: %s", r.Method)
	user := getUserFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	if r.Method == http.MethodPost {
		// Handle form submission
		r.ParseForm()
		
		// Get form values
		driver := r.FormValue("driver")
		busID := r.FormValue("bus_id")
		routeIDsStr := r.FormValue("route_ids") // Comma-separated route IDs
		
		// Split route IDs
		routeIDs := strings.Split(routeIDsStr, ",")
		
		// Clean up route IDs
		cleanRouteIDs := []string{}
		for _, id := range routeIDs {
			trimmed := strings.TrimSpace(id)
			if trimmed != "" {
				cleanRouteIDs = append(cleanRouteIDs, trimmed)
			}
		}
		
		if driver == "" || busID == "" || len(cleanRouteIDs) == 0 {
			log.Printf("WARNING: Missing required fields for route assignment")
			http.Redirect(w, r, "/assign-routes?error=missing_fields", http.StatusSeeOther)
			return
		}

		// Begin transaction
		tx, err := db.Begin()
		if err != nil {
			log.Printf("ERROR: Error starting transaction: %v", err)
			http.Redirect(w, r, "/assign-routes?error=database", http.StatusSeeOther)
			return
		}
		defer tx.Rollback()

		// Only remove assignments for the specific routes being reassigned
		for _, routeID := range cleanRouteIDs {
			_, err = tx.Exec(`
				DELETE FROM route_assignments 
				WHERE route_id = $1
			`, routeID)
			if err != nil {
				log.Printf("WARNING: Error clearing route %s: %v", routeID, err)
			}
		}

		// Insert new assignments
		successCount := 0
		for _, routeID := range cleanRouteIDs {
			_, err = tx.Exec(`
				INSERT INTO route_assignments (driver, bus_id, route_id, assigned_date)
				VALUES ($1, $2, $3, $4)
				ON CONFLICT (driver, route_id) DO UPDATE SET
					bus_id = $2,
					assigned_date = $4
			`, driver, busID, routeID, time.Now())
			
			if err != nil {
				log.Printf("WARNING: Error inserting route %s: %v", routeID, err)
			} else {
				successCount++
			}
		}

		// Commit if at least one route was assigned
		if successCount > 0 {
			if err = tx.Commit(); err != nil {
				log.Printf("ERROR: Error committing transaction: %v", err)
				http.Redirect(w, r, "/assign-routes?error=save_failed", http.StatusSeeOther)
				return
			}
			
			log.Printf("INFO: Assigned %d routes to driver %s with bus %s", successCount, driver, busID)
			http.Redirect(w, r, fmt.Sprintf("/assign-routes?success=1&count=%d", successCount), http.StatusSeeOther)
		} else {
			http.Redirect(w, r, "/assign-routes?error=no_routes_assigned", http.StatusSeeOther)
		}
		return
	}

	// GET request - show the assignment page
	// Get all drivers
	drivers := []User{}
	driverRows, err := db.Query(`
		SELECT username, role, status 
		FROM users 
		WHERE role = 'driver' AND status = 'active'
		ORDER BY username
	`)
	if err != nil {
		log.Printf("Error fetching drivers: %v", err)
	} else {
		defer driverRows.Close()
		for driverRows.Next() {
			var driver User
			err := driverRows.Scan(&driver.Username, &driver.Role, &driver.Status)
			if err != nil {
				log.Printf("Error scanning driver: %v", err)
				continue
			}
			drivers = append(drivers, driver)
		}
		log.Printf("Loaded %d drivers", len(drivers))
	}

	// Get all buses
	buses := []Bus{}
	busRows, err := db.Query(`
		SELECT bus_id, status 
		FROM buses 
		WHERE status = 'active'
		ORDER BY bus_id
	`)
	if err != nil {
		log.Printf("Error fetching buses: %v", err)
	} else {
		defer busRows.Close()
		for busRows.Next() {
			var bus Bus
			err := busRows.Scan(&bus.BusID, &bus.Status)
			if err != nil {
				log.Printf("Error scanning bus: %v", err)
				continue
			}
			buses = append(buses, bus)
		}
		log.Printf("Loaded %d buses", len(buses))
	}

	// Get all routes
	routes := []Route{}
	routeRows, err := db.Query(`
		SELECT route_id, route_name, description 
		FROM routes 
		ORDER BY route_name
	`)
	if err != nil {
		log.Printf("Error fetching routes: %v", err)
	} else {
		defer routeRows.Close()
		for routeRows.Next() {
			var route Route
			var desc sql.NullString
			err := routeRows.Scan(&route.RouteID, &route.RouteName, &desc)
			if err != nil {
				log.Printf("Error scanning route: %v", err)
				continue
			}
			if desc.Valid {
				route.Description = desc.String
			}
			routes = append(routes, route)
		}
		log.Printf("Loaded %d routes", len(routes))
	}

	// Get current assignments grouped by driver
	multiAssignments := getMultiRouteAssignments()
	
	// Get current assignments for status calculation
	assignments, _ := getRouteAssignments()
	
	// Build routes with status
	assignedRoutes := make(map[string]bool)
	for _, a := range assignments {
		assignedRoutes[a.RouteID] = true
	}
	
	type RouteWithStatus struct {
		Route
		IsAssigned bool
	}
	
	var routesWithStatus []RouteWithStatus
	var availableRoutes []Route
	for _, r := range routes {
		routesWithStatus = append(routesWithStatus, RouteWithStatus{
			Route:      r,
			IsAssigned: assignedRoutes[r.RouteID],
		})
		// Add to available routes if not assigned
		if !assignedRoutes[r.RouteID] {
			availableRoutes = append(availableRoutes, r)
		}
	}
	
	// Build available buses list (those not assigned)
	assignedBuses := make(map[string]bool)
	for _, a := range assignments {
		assignedBuses[a.BusID] = true
	}
	var availableBuses []Bus
	for _, b := range buses {
		if !assignedBuses[b.BusID] {
			availableBuses = append(availableBuses, b)
		}
	}

	renderTemplate(w, r, "assign_routes.html", map[string]interface{}{
		"User": user,
		"Data": map[string]interface{}{
			"Drivers":             drivers,
			"Buses":               buses,
			"RoutesWithStatus":    routesWithStatus,
			"AvailableRoutes":     availableRoutes,
			"AvailableBuses":      availableBuses,
			"Assignments":         assignments,
			"MultiAssignments":    multiAssignments,
			"TotalRoutes":         len(routes),
			"TotalAssignments":    len(assignments),
			"AvailableDrivers":    len(drivers) - len(assignedRoutes),
			"AvailableBusesCount": len(availableBuses),
		},
		"Success":      r.URL.Query().Get("success") == "1",
		"SuccessCount": r.URL.Query().Get("count"),
		"Error":        r.URL.Query().Get("error"),
	})
}