// utils.go - Utility functions and helpers
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

// executeTemplate safely executes a template with error handling
func executeTemplate(w http.ResponseWriter, name string, data interface{}) {
	if err := templates.ExecuteTemplate(w, name, data); err != nil {
		log.Printf("Error executing template %s: %v", name, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// ensureDataFiles creates the data directory and default files if they don't exist
func ensureDataFiles() {
	os.MkdirAll("data", os.ModePerm)
	
	// Create default users.json if it doesn't exist
	if _, err := os.Stat("data/users.json"); os.IsNotExist(err) {
		defaultUsers := []User{{"admin", "adminpass", "manager"}}
		f, _ := os.Create("data/users.json")
		json.NewEncoder(f).Encode(defaultUsers)
		f.Close()
	}
	
	// Create empty route_assignments.json if it doesn't exist
	if _, err := os.Stat("data/route_assignments.json"); os.IsNotExist(err) {
		f, _ := os.Create("data/route_assignments.json")
		json.NewEncoder(f).Encode([]RouteAssignment{})
		f.Close()
	}
}

// loadJSON is a generic function to load JSON data from a file
func loadJSON[T any](filename string) ([]T, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	
	var data []T
	err = json.NewDecoder(f).Decode(&data)
	return data, err
}

// seedJSON creates a JSON file with default data if it doesn't exist
func seedJSON[T any](path string, defaultData T) error {
	if _, err := os.Stat(path); err == nil {
		return nil // already present
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("stat %s: %w", path, err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return fmt.Errorf("create: %w", err)
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(defaultData); err != nil {
		return fmt.Errorf("encode: %w", err)
	}
	
	log.Printf("Seeded %s", path)
	return nil
}

// getDriverRouteAssignment returns the current route assignment for a driver
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
		return fmt.Errorf("driver %s does not exist", assignment.Driver)
	}

	// Check if bus exists and is active
	buses := loadBuses()
	busExists := false
	for _, b := range buses {
		if b.BusID == assignment.BusID {
			if b.Status != "active" {
				return fmt.Errorf("bus %s is not active", ass
