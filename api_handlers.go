package main

import (
	"encoding/json"
	"log"
	"net/http"
)

// API Handlers for various endpoints

// apiRoutesHandler returns all routes as JSON
func apiRoutesHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	
	routes, err := loadRoutesFromDB()
	if err != nil {
		log.Printf("Error loading routes: %v", err)
		http.Error(w, "Failed to load routes", http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(routes)
}

// apiBusesHandler returns all buses as JSON
func apiBusesHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	
	buses, err := loadBusesFromDB()
	if err != nil {
		log.Printf("Error loading buses: %v", err)
		http.Error(w, "Failed to load buses", http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(buses)
}

// apiDriversHandler returns all drivers as JSON
func apiDriversHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	
	// Only managers can view all drivers
	if user.Role != "manager" {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	
	users, err := loadUsersFromDB()
	if err != nil {
		log.Printf("Error loading users: %v", err)
		http.Error(w, "Failed to load drivers", http.StatusInternalServerError)
		return
	}
	
	// Filter only drivers
	var drivers []User
	for _, u := range users {
		if u.Role == "driver" {
			drivers = append(drivers, u)
		}
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(drivers)
}

// apiStudentsHandler returns all students as JSON
func apiStudentsHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	
	students, err := loadStudentsFromDB()
	if err != nil {
		log.Printf("Error loading students: %v", err)
		http.Error(w, "Failed to load students", http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(students)
}

// apiFleetVehiclesHandler returns all fleet vehicles as JSON
func apiFleetVehiclesHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	
	vehicles, err := loadFleetVehiclesFromDB()
	if err != nil {
		log.Printf("Error loading fleet vehicles: %v", err)
		http.Error(w, "Failed to load fleet vehicles", http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(vehicles)
}

// apiRouteAssignmentsHandler returns all route assignments as JSON
func apiRouteAssignmentsHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	
	assignments, err := loadRouteAssignments() // Use the existing function
	if err != nil {
		log.Printf("Error loading route assignments: %v", err)
		http.Error(w, "Failed to load route assignments", http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(assignments)
}

// apiECSEStudentsHandler returns all ECSE students as JSON
func apiECSEStudentsHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	
	// Only managers can view ECSE students
	if user.Role != "manager" {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	
	students, err := loadECSEStudentsFromDB()
	if err != nil {
		log.Printf("Error loading ECSE students: %v", err)
		http.Error(w, "Failed to load ECSE students", http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(students)
}

// apiMaintenanceRecordsHandler returns all maintenance records as JSON
func apiMaintenanceRecordsHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	
	records, err := loadMaintenanceRecordsFromDB()
	if err != nil {
		log.Printf("Error loading maintenance records: %v", err)
		http.Error(w, "Failed to load maintenance records", http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(records)
}