// utils.go - Utility functions and helpers (PostgreSQL-only)
package main

import (
	"fmt"
	"log"
	"net/http"
)

// executeTemplate safely executes a template with error handling
func executeTemplate(w http.ResponseWriter, name string, data interface{}) {
	if err := templates.ExecuteTemplate(w, name, data); err != nil {
		log.Printf("Error executing template %s: %v", name, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// getDriverRouteAssignment returns the current route assignment for a driver
// NOTE: This returns the FIRST assignment found - drivers can have multiple routes
func getDriverRouteAssignment(driverUsername string) (*RouteAssignment, error) {
	assignments, err := loadRouteAssignments()
	if err != nil {
		return nil, fmt.Errorf("failed to load assignments: %w", err)
	}

	for _, assignment := range assignments {
		if assignment.Driver == driverUsername {
			return &assignment, nil
		}
	}

	return nil, fmt.Errorf("no assignment found for driver %s", driverUsername)
}

// getDriverRouteAssignments returns ALL route assignments for a driver
func getDriverRouteAssignments(driverUsername string) ([]RouteAssignment, error) {
	assignments, err := loadRouteAssignments()
	if err != nil {
		return nil, fmt.Errorf("failed to load assignments: %w", err)
	}

	var driverAssignments []RouteAssignment
	for _, assignment := range assignments {
		if assignment.Driver == driverUsername {
			driverAssignments = append(driverAssignments, assignment)
		}
	}

	return driverAssignments, nil
}

// validateRouteAssignment checks if a route assignment is valid
func validateRouteAssignment(assignment RouteAssignment) error {
	if assignment.Driver == "" {
		return fmt.Errorf("driver cannot be empty")
	}
	if assignment.BusID == "" {
		return fmt.Errorf("bus ID cannot be empty")
	}
	if assignment.RouteID == "" {
		return fmt.Errorf("route ID cannot be empty")
	}

	// Check if driver exists
	users := loadUsers()
	driverExists := false
	for _, u := range users {
		if u.Username == assignment.Driver && u.Role == "driver" {
			driverExists = true
			break
		}
	}
	if !driverExists {
		return fmt.Errorf("driver %s does not exist or is not a driver", assignment.Driver)
	}

	// Check if bus exists and is active
	buses := loadBuses()
	busExists := false
	for _, b := range buses {
		if b.BusID == assignment.BusID {
			if b.Status != "active" {
				return fmt.Errorf("bus %s is not active (status: %s)", assignment.BusID, b.Status)
			}
			busExists = true
			break
		}
	}
	if !busExists {
		return fmt.Errorf("bus %s does not exist", assignment.BusID)
	}

	// Check if route exists
	routes, err := loadRoutes()
	if err != nil {
		return fmt.Errorf("failed to load routes for validation: %w", err)
	}
	
	routeExists := false
	for _, r := range routes {
		if r.RouteID == assignment.RouteID {
			routeExists = true
			// Also ensure the route name matches if provided
			if assignment.RouteName != "" && r.RouteName != assignment.RouteName {
				log.Printf("Warning: Route name mismatch for route %s: expected '%s', got '%s'", 
					assignment.RouteID, r.RouteName, assignment.RouteName)
			}
			break
		}
	}
	if !routeExists {
		return fmt.Errorf("route %s does not exist", assignment.RouteID)
	}

	// Check if this exact assignment already exists (driver + route combination)
	existingAssignments, err := loadRouteAssignments()
	if err != nil {
		log.Printf("Warning: Could not check existing assignments: %v", err)
		// Continue anyway - the database constraint will catch duplicates
	} else {
		for _, existing := range existingAssignments {
			if existing.Driver == assignment.Driver && existing.RouteID == assignment.RouteID {
				return fmt.Errorf("driver %s is already assigned to route %s", 
					assignment.Driver, existing.RouteID)
			}
		}
	}

	// NOTE: We NO LONGER check if driver is assigned to other routes
	// Drivers can now have multiple route assignments

	return nil
}

// getUserFromSession retrieves the user from the secure session
func getUserFromSession(r *http.Request) *User {
	// Get session cookie
	cookie, err := r.Cookie("session_id")
	if err != nil {
		return nil
	}
	
	// Get session data
	session, exists := GetSecureSession(cookie.Value)
	if !exists {
		return nil
	}
	
	// Load user from database
	users := loadUsers()
	for _, u := range users {
		if u.Username == session.Username {
			return &u
		}
	}
	
	return nil
}

// generateID generates a unique ID for entities
func generateID(prefix string, count int) string {
	return fmt.Sprintf("%s%03d", prefix, count+1)
}

// ensureUniqueID ensures the generated ID is unique among existing IDs
func ensureUniqueID(prefix string, existingIDs []string) string {
	counter := len(existingIDs) + 1
	for {
		newID := fmt.Sprintf("%s%03d", prefix, counter)
		exists := false
		for _, id := range existingIDs {
			if id == newID {
				exists = true
				break
			}
		}
		if !exists {
			return newID
		}
		counter++
	}
}
