// handlers.go - HTTP handlers for route assignment fixes
package main

import (
    "encoding/json"
    "log"
    "net/http"
)

// NEW HANDLER: Check if driver has existing bus
func handleCheckDriverBus(w http.ResponseWriter, r *http.Request) {
    driver := r.URL.Query().Get("driver")
    if driver == "" {
        http.Error(w, "Driver parameter required", http.StatusBadRequest)
        return
    }
    
    busID, err := getDriverAssignedBus(driver)
    if err != nil {
        log.Printf("Error checking driver bus: %v", err)
        http.Error(w, "Failed to check driver bus", http.StatusInternalServerError)
        return
    }
    
    response := map[string]string{
        "bus_id": busID,
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

// UPDATED HANDLER: Handle route assignment with optional bus
func handleSaveRouteAssignment(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }
    
    var assignment RouteAssignment
    if err := json.NewDecoder(r.Body).Decode(&assignment); err != nil {
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }
    
    // Validate required fields (bus_id is now optional)
    if assignment.Driver == "" || assignment.RouteID == "" {
        http.Error(w, "Driver and route_id are required", http.StatusBadRequest)
        return
    }
    
    if err := saveRouteAssignment(assignment); err != nil {
        log.Printf("Error saving route assignment: %v", err)
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]bool{"success": true})
}
