package main

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"time"
)

// routeMonitoringHandler shows the route monitoring dashboard
func routeMonitoringHandler(w http.ResponseWriter, r *http.Request) {
	session := getUserFromSession(r)
	if session == nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// Get active routes with monitoring status
	activeRoutes := getActiveMonitoredRoutes()
	
	// Get recent deviations
	recentDeviations := getRecentDeviations(24 * time.Hour)
	
	// Get deviation statistics
	stats := getDeviationStats()

	data := struct {
		Title            string
		Username         string
		UserType         string
		CSPNonce         string
		ActiveRoutes     []MonitoredRoute
		RecentDeviations []RouteDeviation
		Stats            DeviationStats
	}{
		Title:            "Route Monitoring",
		Username:         session.Username,
		UserType:         session.Role,
		CSPNonce:         generateNonce(),
		ActiveRoutes:     activeRoutes,
		RecentDeviations: recentDeviations,
		Stats:            stats,
	}

	tmpl := template.Must(template.ParseFiles("templates/route_monitoring.html"))
	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("Error rendering route monitoring page: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// routePlannerHandler shows the route planning interface
func routePlannerHandler(w http.ResponseWriter, r *http.Request) {
	session := getUserFromSession(r)
	if session == nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// Get all routes
	routes := getAllActiveRoutes()

	data := struct {
		Title    string
		Username string
		UserType string
		CSPNonce string
		Routes   []Route
	}{
		Title:    "Route Planner",
		Username: session.Username,
		UserType: session.Role,
		CSPNonce: generateNonce(),
		Routes:   routes,
	}

	tmpl := template.Must(template.ParseFiles("templates/route_planner.html"))
	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("Error rendering route planner page: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// API Endpoints

// startRouteMonitoringHandler starts monitoring a route
func startRouteMonitoringHandler(w http.ResponseWriter, r *http.Request) {
	session := getUserFromSession(r)
	if session == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		VehicleID string `json:"vehicle_id"`
		RouteID   string `json:"route_id"`
		DriverID  string `json:"driver_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Start monitoring
	if routeMonitor != nil {
		if err := routeMonitor.StartRouteMonitoring(req.VehicleID, req.RouteID, req.DriverID); err != nil {
			log.Printf("Failed to start route monitoring: %v", err)
			http.Error(w, "Failed to start monitoring", http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Route monitoring started",
	})
}

// stopRouteMonitoringHandler stops monitoring a route
func stopRouteMonitoringHandler(w http.ResponseWriter, r *http.Request) {
	session := getUserFromSession(r)
	if session == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	vehicleID := r.URL.Query().Get("vehicle_id")
	if vehicleID == "" {
		http.Error(w, "Vehicle ID required", http.StatusBadRequest)
		return
	}

	// Stop monitoring
	if routeMonitor != nil {
		routeMonitor.StopRouteMonitoring(vehicleID)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Route monitoring stopped",
	})
}

// getRouteDeviationsHandler returns deviations for a route
func getRouteDeviationsHandler(w http.ResponseWriter, r *http.Request) {
	session := getUserFromSession(r)
	if session == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	vehicleID := r.URL.Query().Get("vehicle_id")
	hours := 24
	if h := r.URL.Query().Get("hours"); h != "" {
		if parsed, err := strconv.Atoi(h); err == nil {
			hours = parsed
		}
	}

	deviations := getVehicleDeviations(vehicleID, time.Duration(hours)*time.Hour)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(deviations)
}

// saveRoutePlanHandler saves or updates a route plan
func saveRoutePlanHandler(w http.ResponseWriter, r *http.Request) {
	session := getUserFromSession(r)
	if session == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		RouteID string      `json:"route_id"`
		Stops   []RouteStop `json:"stops"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Save route plan
	if err := saveRoutePlan(req.RouteID, req.Stops); err != nil {
		log.Printf("Failed to save route plan: %v", err)
		http.Error(w, "Failed to save route plan", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Route plan saved",
	})
}

// getRoutePlanHandler returns the route plan
func getRoutePlanHandler(w http.ResponseWriter, r *http.Request) {
	session := getUserFromSession(r)
	if session == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	routeID := r.URL.Query().Get("route_id")
	if routeID == "" {
		http.Error(w, "Route ID required", http.StatusBadRequest)
		return
	}

	stops, err := loadRoutePlan(routeID)
	if err != nil {
		log.Printf("Failed to load route plan: %v", err)
		http.Error(w, "Failed to load route plan", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"route_id": routeID,
		"stops":    stops,
	})
}

// updateDeviationSettingsHandler updates deviation monitoring settings
func updateDeviationSettingsHandler(w http.ResponseWriter, r *http.Request) {
	session := getUserFromSession(r)
	if session == nil || session.Role != "manager" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var settings struct {
		DeviationRadius float64       `json:"deviation_radius"` // meters
		StopDuration    int           `json:"stop_duration"`    // minutes
		CheckInterval   int           `json:"check_interval"`   // seconds
	}

	if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Update settings
	if routeMonitor != nil {
		routeMonitor.mu.Lock()
		routeMonitor.deviationRadius = settings.DeviationRadius
		routeMonitor.stopDuration = time.Duration(settings.StopDuration) * time.Minute
		routeMonitor.checkInterval = time.Duration(settings.CheckInterval) * time.Second
		routeMonitor.mu.Unlock()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Settings updated",
	})
}

// Helper functions

type MonitoredRoute struct {
	RouteID          string    `json:"route_id"`
	RouteName        string    `json:"route_name"`
	VehicleID        string    `json:"vehicle_id"`
	DriverName       string    `json:"driver_name"`
	Status           string    `json:"status"`
	StartTime        time.Time `json:"start_time"`
	CompletedStops   int       `json:"completed_stops"`
	TotalStops       int       `json:"total_stops"`
	CurrentDeviation *RouteDeviation `json:"current_deviation,omitempty"`
}

type DeviationStats struct {
	TotalToday       int            `json:"total_today"`
	ActiveDeviations int            `json:"active_deviations"`
	ByType           map[string]int `json:"by_type"`
	BySeverity       map[string]int `json:"by_severity"`
	TopVehicles      []VehicleDeviationCount `json:"top_vehicles"`
}

type VehicleDeviationCount struct {
	VehicleID string `json:"vehicle_id"`
	Count     int    `json:"count"`
}

func getActiveMonitoredRoutes() []MonitoredRoute {
	var routes []MonitoredRoute
	
	if routeMonitor == nil {
		return routes
	}

	routeMonitor.mu.RLock()
	defer routeMonitor.mu.RUnlock()

	for vehicleID, activeRoute := range routeMonitor.activeRoutes {
		monitored := MonitoredRoute{
			RouteID:        activeRoute.RouteID,
			VehicleID:      vehicleID,
			Status:         activeRoute.Status,
			StartTime:      activeRoute.StartTime,
			CompletedStops: len(activeRoute.CompletedStops),
			TotalStops:     len(activeRoute.PlannedStops),
		}

		// Get route name
		var routeName string
		db.QueryRow("SELECT route_name FROM routes WHERE route_id = $1", activeRoute.RouteID).Scan(&routeName)
		monitored.RouteName = routeName

		// Get driver name
		var driverName string
		db.QueryRow("SELECT username FROM users WHERE username = $1", activeRoute.DriverID).Scan(&driverName)
		monitored.DriverName = driverName

		// Get current deviation
		if dev, exists := routeMonitor.deviations[vehicleID]; exists {
			monitored.CurrentDeviation = dev
		}

		routes = append(routes, monitored)
	}

	return routes
}

func getRecentDeviations(duration time.Duration) []RouteDeviation {
	var deviations []RouteDeviation
	
	since := time.Now().Add(-duration)
	
	query := `
		SELECT id, vehicle_id, route_id, driver_id, deviation_type, severity,
		       location, expected_location, distance, duration, description,
		       auto_resolved, resolved_at, created_at, metadata
		FROM route_deviations
		WHERE created_at > $1
		ORDER BY created_at DESC
		LIMIT 100
	`
	
	rows, err := db.Query(query, since)
	if err != nil {
		log.Printf("Failed to get recent deviations: %v", err)
		return deviations
	}
	defer rows.Close()

	for rows.Next() {
		var dev RouteDeviation
		var locationJSON, expectedLocationJSON, metadataJSON []byte
		var durationMS int64
		
		err := rows.Scan(&dev.ID, &dev.VehicleID, &dev.RouteID, &dev.DriverID,
			&dev.DeviationType, &dev.Severity, &locationJSON, &expectedLocationJSON,
			&dev.Distance, &durationMS, &dev.Description, &dev.AutoResolved,
			&dev.ResolvedAt, &dev.CreatedAt, &metadataJSON)
		
		if err != nil {
			continue
		}

		json.Unmarshal(locationJSON, &dev.Location)
		if len(expectedLocationJSON) > 0 {
			var expectedLoc GPSLocation
			json.Unmarshal(expectedLocationJSON, &expectedLoc)
			dev.ExpectedLocation = &expectedLoc
		}
		json.Unmarshal(metadataJSON, &dev.Metadata)
		dev.Duration = time.Duration(durationMS) * time.Millisecond

		deviations = append(deviations, dev)
	}

	return deviations
}

func getVehicleDeviations(vehicleID string, duration time.Duration) []RouteDeviation {
	var deviations []RouteDeviation
	
	since := time.Now().Add(-duration)
	
	query := `
		SELECT id, vehicle_id, route_id, driver_id, deviation_type, severity,
		       location, expected_location, distance, duration, description,
		       auto_resolved, resolved_at, created_at, metadata
		FROM route_deviations
		WHERE vehicle_id = $1 AND created_at > $2
		ORDER BY created_at DESC
	`
	
	rows, err := db.Query(query, vehicleID, since)
	if err != nil {
		return deviations
	}
	defer rows.Close()

	for rows.Next() {
		var dev RouteDeviation
		var locationJSON, expectedLocationJSON, metadataJSON []byte
		var durationMS int64
		
		err := rows.Scan(&dev.ID, &dev.VehicleID, &dev.RouteID, &dev.DriverID,
			&dev.DeviationType, &dev.Severity, &locationJSON, &expectedLocationJSON,
			&dev.Distance, &durationMS, &dev.Description, &dev.AutoResolved,
			&dev.ResolvedAt, &dev.CreatedAt, &metadataJSON)
		
		if err != nil {
			continue
		}

		json.Unmarshal(locationJSON, &dev.Location)
		if len(expectedLocationJSON) > 0 {
			var expectedLoc GPSLocation
			json.Unmarshal(expectedLocationJSON, &expectedLoc)
			dev.ExpectedLocation = &expectedLoc
		}
		json.Unmarshal(metadataJSON, &dev.Metadata)
		dev.Duration = time.Duration(durationMS) * time.Millisecond

		deviations = append(deviations, dev)
	}

	return deviations
}

func getDeviationStats() DeviationStats {
	var stats DeviationStats
	stats.ByType = make(map[string]int)
	stats.BySeverity = make(map[string]int)

	// Get today's stats
	db.QueryRow(`
		SELECT COUNT(*) FROM route_deviations 
		WHERE created_at::date = CURRENT_DATE
	`).Scan(&stats.TotalToday)

	// Get active deviations
	db.QueryRow(`
		SELECT COUNT(*) FROM route_deviations 
		WHERE resolved_at IS NULL
	`).Scan(&stats.ActiveDeviations)

	// Get by type
	rows, _ := db.Query(`
		SELECT deviation_type, COUNT(*) 
		FROM route_deviations 
		WHERE created_at > CURRENT_DATE - INTERVAL '7 days'
		GROUP BY deviation_type
	`)
	defer rows.Close()

	for rows.Next() {
		var devType string
		var count int
		rows.Scan(&devType, &count)
		stats.ByType[devType] = count
	}

	// Get by severity
	rows2, _ := db.Query(`
		SELECT severity, COUNT(*) 
		FROM route_deviations 
		WHERE created_at > CURRENT_DATE - INTERVAL '7 days'
		GROUP BY severity
	`)
	defer rows2.Close()

	for rows2.Next() {
		var severity string
		var count int
		rows2.Scan(&severity, &count)
		stats.BySeverity[severity] = count
	}

	// Get top vehicles with deviations
	rows3, _ := db.Query(`
		SELECT vehicle_id, COUNT(*) as count
		FROM route_deviations 
		WHERE created_at > CURRENT_DATE - INTERVAL '30 days'
		GROUP BY vehicle_id
		ORDER BY count DESC
		LIMIT 5
	`)
	defer rows3.Close()

	for rows3.Next() {
		var vc VehicleDeviationCount
		rows3.Scan(&vc.VehicleID, &vc.Count)
		stats.TopVehicles = append(stats.TopVehicles, vc)
	}

	return stats
}

func saveRoutePlan(routeID string, stops []RouteStop) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Delete existing stops
	_, err = tx.Exec("DELETE FROM route_plans WHERE route_id = $1", routeID)
	if err != nil {
		return err
	}

	// Insert new stops
	for _, stop := range stops {
		_, err = tx.Exec(`
			INSERT INTO route_plans 
			(route_id, stop_number, stop_name, latitude, longitude,
			 planned_arrival, planned_departure, stop_radius)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`, routeID, stop.StopNumber, stop.Name, stop.Latitude, stop.Longitude,
		   stop.ArrivalTime, stop.DepartureTime, stop.StopRadius)
		
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func getAllActiveRoutes() []Route {
	var routes []Route
	
	query := `
		SELECT route_id, route_name
		FROM routes
		WHERE active = true OR active IS NULL
		ORDER BY route_name
	`
	
	rows, err := db.Query(query)
	if err != nil {
		log.Printf("Failed to get routes: %v", err)
		return routes
	}
	defer rows.Close()
	
	for rows.Next() {
		var route Route
		err := rows.Scan(&route.RouteID, &route.RouteName)
		if err != nil {
			continue
		}
		routes = append(routes, route)
	}
	
	return routes
}

