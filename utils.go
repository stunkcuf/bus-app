// utils.go - Utility functions and helpers
package main

import (
	"encoding/json"
	"fmt"
	"html/template"
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
				return fmt.Errorf("bus %s is not active", assignment.BusID)
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
		return fmt.Errorf("failed to load routes: %w", err)
	}
	routeExists := false
	for _, r := range routes {
		// Check both RouteID and RouteName for flexibility
		if r.RouteID == assignment.RouteID || r.RouteName == assignment.RouteName {
			routeExists = true
			break
		}
	}
	if !routeExists {
		return fmt.Errorf("route %s does not exist", assignment.RouteID)
	}

	return nil
}

// getUserFromSession retrieves the user from the session cookie
func getUserFromSession(r *http.Request) *User {
	cookie, err := r.Cookie("session_user")
	if err != nil {
		return nil
	}
	
	uname := cookie.Value
	for _, u := range loadUsers() {
		if u.Username == uname {
			return &u
		}
	}
	return nil
}

// initDataFiles creates initial data files with proper ID structure
func initDataFiles() {
	// Ensure data directory exists with proper permissions
	if err := os.MkdirAll("data", 0755); err != nil {
		log.Printf("Warning: failed to create data directory: %v", err)
		return
	}

	// Create buses.json if it doesn't exist
	if _, err := os.Stat("data/buses.json"); os.IsNotExist(err) {
		defaultBuses := []*Bus{
			{BusID: "BUS001", Status: "active", Model: "Ford Transit", Capacity: 20, OilStatus: "good", TireStatus: "good", MaintenanceNotes: ""},
			{BusID: "BUS002", Status: "active", Model: "Chevrolet Express", Capacity: 25, OilStatus: "due", TireStatus: "good", MaintenanceNotes: "Oil change scheduled"},
			{BusID: "BUS003", Status: "maintenance", Model: "Toyota Coaster", Capacity: 15, OilStatus: "good", TireStatus: "worn", MaintenanceNotes: "Brake inspection in progress"},
		}
		f, err := os.OpenFile("data/buses.json", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			log.Printf("Warning: failed to create buses.json: %v", err)
			return
		}
		defer f.Close()
		enc := json.NewEncoder(f)
		enc.SetIndent("", "  ")
		if err := enc.Encode(defaultBuses); err != nil {
			log.Printf("Warning: failed to encode buses to json: %v", err)
			return
		}
		log.Println("Created and seeded data/buses.json with ID-based structure")
	}

	// Create vehicle.json if it doesn't exist
	if _, err := os.Stat("data/vehicle.json"); os.IsNotExist(err) {
		defaultVehicles := []Vehicle{
			{VehicleID: "VEH001", Model: "Ford F-150", Year: "2022", License: "ABC123", Status: "active", OilStatus: "good", TireStatus: "good"},
			{VehicleID: "VEH002", Model: "Chevrolet Silverado", Year: "2021", License: "XYZ789", Status: "active", OilStatus: "needs_service", TireStatus: "worn"},
		}
		f, err := os.OpenFile("data/vehicle.json", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			log.Printf("Warning: failed to create vehicle.json: %v", err)
			return
		}
		defer f.Close()
		enc := json.NewEncoder(f)
		enc.SetIndent("", "  ")
		if err := enc.Encode(defaultVehicles); err != nil {
			log.Printf("Warning: failed to encode vehicles to json: %v", err)
			return
		}
		log.Println("Created and seeded data/vehicle.json")
	}

	// Create students.json if it doesn't exist
	if _, err := os.Stat("data/students.json"); os.IsNotExist(err) {
		defaultStudents := []Student{}
		f, err := os.OpenFile("data/students.json", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			log.Printf("Warning: failed to create students.json: %v", err)
			return
		}
		defer f.Close()
		enc := json.NewEncoder(f)
		enc.SetIndent("", "  ")
		if err := enc.Encode(defaultStudents); err != nil {
			log.Printf("Warning: failed to encode students to json: %v", err)
			return
		}
		log.Println("Created data/students.json")
	}

	// Create routes.json if it doesn't exist
	if _, err := os.Stat("data/routes.json"); os.IsNotExist(err) {
		routes := []Route{
			{
				RouteID:     "RT001",
				RouteName:   "Victory Square",
				Description: "Downtown Victory Square route",
				Positions: []struct {
					Position int    `json:"position"`
					Student  string `json:"student"`
				}{{Position: 1, Student: "Alice Johnson"}, {Position: 2, Student: "Bob Smith"}},
			},
			{
				RouteID:     "RT002",
				RouteName:   "Airportway",
				Description: "Airport way business district",
				Positions: []struct {
					Position int    `json:"position"`
					Student  string `json:"student"`
				}{{Position: 1, Student: "Charlie Brown"}, {Position: 2, Student: "David Wilson"}},
			},
			{
				RouteID:     "RT003",
				RouteName:   "NELC",
				Description: "Northeast Learning Center route",
				Positions: []struct {
					Position int    `json:"position"`
					Student  string `json:"student"`
				}{{Position: 1, Student: "Emma Davis"}, {Position: 2, Student: "Frank Miller"}},
			},
			{
				RouteID:     "RT004",
				RouteName:   "Irrigon",
				Description: "Irrigon community route",
				Positions: []struct {
					Position int    `json:"position"`
					Student  string `json:"student"`
				}{{Position: 1, Student: "Grace Lee"}, {Position: 2, Student: "Henry Clark"}},
			},
			{
				RouteID:     "RT005",
				RouteName:   "PELC",
				Description: "Pacific Educational Learning Center",
				Positions: []struct {
					Position int    `json:"position"`
					Student  string `json:"student"`
				}{{Position: 1, Student: "Ivy Rodriguez"}, {Position: 2, Student: "Jack Thompson"}},
			},
			{
				RouteID:     "RT006",
				RouteName:   "Umatilla",
				Description: "Umatilla district route",
				Positions: []struct {
					Position int    `json:"position"`
					Student  string `json:"student"`
				}{{Position: 1, Student: "Kate Anderson"}, {Position: 2, Student: "Liam Garcia"}},
			},
		}
		f, err := os.OpenFile("data/routes.json", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			log.Printf("Warning: failed to create routes.json: %v", err)
			return
		}
		defer f.Close()
		enc := json.NewEncoder(f)
		enc.SetIndent("", "  ")
		if err := enc.Encode(routes); err != nil {
			log.Printf("Warning: failed to encode routes to json: %v", err)
			return
		}
		log.Println("Created and seeded data/routes.json with RouteID structure")
	}

	// Create other empty JSON files
	createEmptyJSONIfNotExists("data/route_assignments.json", []RouteAssignment{})
	createEmptyJSONIfNotExists("data/maintenance.json", []MaintenanceLog{})
	createEmptyJSONIfNotExists("data/driver_logs.json", []DriverLog{})
}

// createEmptyJSONIfNotExists creates an empty JSON array file if it doesn't exist
func createEmptyJSONIfNotExists(filename string, emptyData interface{}) {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			log.Printf("Warning: failed to create %s: %v", filename, err)
			return
		}
		defer f.Close()
		enc := json.NewEncoder(f)
		enc.SetIndent("", "  ")
		if err := enc.Encode(emptyData); err != nil {
			log.Printf("Warning: failed to encode to %s: %v", filename, err)
			return
		}
		log.Printf("Created %s", filename)
	}
}
