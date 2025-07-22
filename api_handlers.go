package main

import (
    "encoding/json"
    "fmt"
    "net/http"
    "strings"
    "time"
)

// invalidateCache clears cached data to ensure fresh data is loaded
func invalidateCache() {
    // For now, this is a placeholder - implement based on your caching strategy
    // If using a global cache, clear it here
    // Example: if globalCache != nil { globalCache.Clear() }
}

// API Response structure
type APIResponse struct {
    Success bool        `json:"success"`
    Message string      `json:"message"`
    Data    interface{} `json:"data,omitempty"`
    Error   string      `json:"error,omitempty"`
}

// Dashboard Stats API
func apiDashboardStatsHandler(w http.ResponseWriter, r *http.Request) {
    user := getUserFromSession(r)
    if user == nil {
        sendAPIError(w, "Unauthorized", http.StatusUnauthorized)
        return
    }

    stats := make(map[string]interface{})
    
    // Get counts from database
    var busCount, activeDrivers, totalRoutes, totalStudents int
    db.Get(&busCount, "SELECT COUNT(*) FROM buses WHERE status = 'active'")
    db.Get(&activeDrivers, "SELECT COUNT(DISTINCT driver) FROM route_assignments")
    db.Get(&totalRoutes, "SELECT COUNT(*) FROM routes")
    db.Get(&totalStudents, "SELECT COUNT(*) FROM students WHERE active = true")
    
    stats["activeBuses"] = busCount
    stats["activeDrivers"] = activeDrivers
    stats["totalRoutes"] = totalRoutes
    stats["totalStudents"] = totalStudents
    
    // Get maintenance alerts
    var maintenanceAlerts int
    db.Get(&maintenanceAlerts, `
        SELECT COUNT(*) FROM buses 
        WHERE oil_status IN ('needs_service', 'overdue') 
        OR tire_status IN ('worn', 'replace')
    `)
    stats["maintenanceAlerts"] = maintenanceAlerts
    
    sendAPIResponse(w, "Stats retrieved", stats)
}

// Search API
func apiSearchHandler(w http.ResponseWriter, r *http.Request) {
    user := getUserFromSession(r)
    if user == nil {
        sendAPIError(w, "Unauthorized", http.StatusUnauthorized)
        return
    }

    query := r.URL.Query().Get("q")
    if query == "" {
        sendAPIError(w, "Query parameter required", http.StatusBadRequest)
        return
    }

    results := make(map[string]interface{})
    searchTerm := "%" + strings.ToLower(query) + "%"
    
    // Search buses
    var buses []Bus
    db.Select(&buses, `
        SELECT * FROM buses 
        WHERE LOWER(bus_id) LIKE $1 
        OR LOWER(model) LIKE $1 
        OR LOWER(status) LIKE $1
        LIMIT 10
    `, searchTerm)
    results["buses"] = buses
    
    // Search drivers
    var drivers []User
    db.Select(&drivers, `
        SELECT username, role, status FROM users 
        WHERE LOWER(username) LIKE $1 
        AND role = 'driver'
        LIMIT 10
    `, searchTerm)
    results["drivers"] = drivers
    
    // Search students
    var students []Student
    db.Select(&students, `
        SELECT * FROM students 
        WHERE LOWER(name) LIKE $1 
        OR LOWER(student_id) LIKE $1
        LIMIT 10
    `, searchTerm)
    results["students"] = students
    
    sendAPIResponse(w, fmt.Sprintf("Found results for '%s'", query), results)
}

// Vehicle Status Update API
func apiUpdateVehicleStatusHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != "POST" {
        sendAPIError(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    user := getUserFromSession(r)
    if user == nil || user.Role != "manager" {
        sendAPIError(w, "Unauthorized", http.StatusUnauthorized)
        return
    }

    var req struct {
        VehicleID   string `json:"vehicle_id"`
        StatusType  string `json:"status_type"`
        NewStatus   string `json:"new_status"`
    }

    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        sendAPIError(w, "Invalid request body", http.StatusBadRequest)
        return
    }

    // Update based on status type
    var query string
    switch req.StatusType {
    case "oil":
        query = "UPDATE buses SET oil_status = $1, updated_at = $2 WHERE bus_id = $3"
    case "tire":
        query = "UPDATE buses SET tire_status = $1, updated_at = $2 WHERE bus_id = $3"
    case "status":
        query = "UPDATE buses SET status = $1, updated_at = $2 WHERE bus_id = $3"
    default:
        sendAPIError(w, "Invalid status type", http.StatusBadRequest)
        return
    }

    _, err := db.Exec(query, req.NewStatus, time.Now(), req.VehicleID)
    if err != nil {
        sendAPIError(w, "Failed to update status", http.StatusInternalServerError)
        return
    }

    // Invalidate cache
    invalidateCache()

    sendAPIResponse(w, "Status updated successfully", nil)
}

// Route Assignment API
func apiAssignRouteHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != "POST" {
        sendAPIError(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    user := getUserFromSession(r)
    if user == nil || user.Role != "manager" {
        sendAPIError(w, "Unauthorized", http.StatusUnauthorized)
        return
    }

    var req struct {
        Driver  string `json:"driver"`
        BusID   string `json:"bus_id"`
        RouteID string `json:"route_id"`
    }

    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        sendAPIError(w, "Invalid request body", http.StatusBadRequest)
        return
    }

    // Validate assignment doesn't conflict
    var existingCount int
    db.Get(&existingCount, `
        SELECT COUNT(*) FROM route_assignments 
        WHERE (driver = $1 OR bus_id = $2) AND route_id != $3
    `, req.Driver, req.BusID, req.RouteID)

    if existingCount > 0 {
        sendAPIError(w, "Driver or bus already assigned to another route", http.StatusConflict)
        return
    }

    // Create assignment
    _, err := db.Exec(`
        INSERT INTO route_assignments (driver, bus_id, route_id, assigned_date)
        VALUES ($1, $2, $3, $4)
        ON CONFLICT (driver, bus_id) DO UPDATE 
        SET route_id = $3, assigned_date = $4
    `, req.Driver, req.BusID, req.RouteID, time.Now())

    if err != nil {
        sendAPIError(w, "Failed to create assignment", http.StatusInternalServerError)
        return
    }

    invalidateCache()
    sendAPIResponse(w, "Route assigned successfully", nil)
}

// Student Attendance API
func apiUpdateAttendanceHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != "POST" {
        sendAPIError(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    user := getUserFromSession(r)
    if user == nil {
        sendAPIError(w, "Unauthorized", http.StatusUnauthorized)
        return
    }

    var req struct {
        LogID      int                  `json:"log_id"`
        Attendance []StudentAttendance  `json:"attendance"`
    }

    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        sendAPIError(w, "Invalid request body", http.StatusBadRequest)
        return
    }

    // Update attendance
    attendanceJSON, _ := json.Marshal(req.Attendance)
    _, err := db.Exec(`
        UPDATE driver_logs 
        SET attendance = $1 
        WHERE id = $2 AND driver = $3
    `, attendanceJSON, req.LogID, user.Username)

    if err != nil {
        sendAPIError(w, "Failed to update attendance", http.StatusInternalServerError)
        return
    }

    sendAPIResponse(w, "Attendance updated successfully", nil)
}

// Maintenance Record API
func apiCreateMaintenanceRecordHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != "POST" {
        sendAPIError(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    user := getUserFromSession(r)
    if user == nil || user.Role != "manager" {
        sendAPIError(w, "Unauthorized", http.StatusUnauthorized)
        return
    }

    var req struct {
        VehicleID   string  `json:"vehicle_id"`
        Category    string  `json:"category"`
        Description string  `json:"description"`
        Cost        float64 `json:"cost"`
        Mileage     int     `json:"mileage"`
    }

    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        sendAPIError(w, "Invalid request body", http.StatusBadRequest)
        return
    }

    // Create maintenance record
    _, err := db.Exec(`
        INSERT INTO bus_maintenance_logs (bus_id, date, category, notes, mileage, cost, created_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7)
    `, req.VehicleID, time.Now(), req.Category, req.Description, req.Mileage, req.Cost, time.Now())

    if err != nil {
        sendAPIError(w, "Failed to create maintenance record", http.StatusInternalServerError)
        return
    }

    // Update vehicle status if needed
    if req.Category == "oil_change" {
        db.Exec("UPDATE buses SET oil_status = 'good', last_oil_change = $1 WHERE bus_id = $2", 
            req.Mileage, req.VehicleID)
    } else if req.Category == "tire_service" {
        db.Exec("UPDATE buses SET tire_status = 'good', last_tire_service = $1 WHERE bus_id = $2", 
            req.Mileage, req.VehicleID)
    }

    invalidateCache()
    sendAPIResponse(w, "Maintenance record created", nil)
}

// Real-time Vehicle Location API (for future GPS tracking)
func apiVehicleLocationHandler(w http.ResponseWriter, r *http.Request) {
    user := getUserFromSession(r)
    if user == nil {
        sendAPIError(w, "Unauthorized", http.StatusUnauthorized)
        return
    }

    vehicleID := r.URL.Query().Get("vehicle_id")
    if vehicleID == "" {
        // Return all vehicle locations
        locations := make(map[string]interface{})
        
        // Mock data for now - replace with actual GPS data
        locations["buses"] = []map[string]interface{}{
            {
                "vehicle_id": "24",
                "lat": 41.8781,
                "lng": -87.6298,
                "speed": 25,
                "heading": 180,
                "last_update": time.Now().Unix(),
            },
        }
        
        sendAPIResponse(w, "Vehicle locations retrieved", locations)
    } else {
        // Return specific vehicle location
        location := map[string]interface{}{
            "vehicle_id": vehicleID,
            "lat": 41.8781,
            "lng": -87.6298,
            "speed": 0,
            "heading": 0,
            "last_update": time.Now().Unix(),
        }
        
        sendAPIResponse(w, "Vehicle location retrieved", location)
    }
}

// Helper functions for API responses
func sendAPIResponse(w http.ResponseWriter, message string, data interface{}) {
    w.Header().Set("Content-Type", "application/json")
    response := APIResponse{
        Success: true,
        Message: message,
        Data:    data,
    }
    json.NewEncoder(w).Encode(response)
}

func sendAPIError(w http.ResponseWriter, message string, statusCode int) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(statusCode)
    response := APIResponse{
        Success: false,
        Error:   message,
    }
    json.NewEncoder(w).Encode(response)
}

// Register all API routes
func registerAPIRoutes() {
    http.HandleFunc("/api/dashboard-stats", apiDashboardStatsHandler)
    http.HandleFunc("/api/search", apiSearchHandler)
    http.HandleFunc("/api/update-vehicle-status", apiUpdateVehicleStatusHandler)
    http.HandleFunc("/api/assign-route", apiAssignRouteHandler)
    http.HandleFunc("/api/update-attendance", apiUpdateAttendanceHandler)
    http.HandleFunc("/api/create-maintenance", apiCreateMaintenanceRecordHandler)
    http.HandleFunc("/api/vehicle-locations", apiVehicleLocationHandler)
}