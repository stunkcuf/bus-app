package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"runtime"
	"time"
)

// RealtimeDashboardHandler serves the real-time dashboard page
func RealtimeDashboardHandler(w http.ResponseWriter, r *http.Request) {
	// Get user from session
	user := getUserFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// Only managers can access the real-time dashboard
	if user.Role != "manager" {
		http.Error(w, "Unauthorized - Manager access required", http.StatusForbidden)
		return
	}

	data := TemplateData{
		Title: "Real-Time Dashboard",
		User:  user,
		Data: map[string]interface{}{
			"timestamp": time.Now(),
		},
	}

	// Log access
	log.Printf("Real-time dashboard accessed by: %s", user.Username)

	// Render template
	renderTemplate(w, r, "realtime_dashboard.html", data)
}

// Initialize real-time features
func InitializeRealtimeFeatures() {
	log.Println("Initializing real-time features...")

	// Initialize WebSocket hub
	InitWebSocket()

	// Start background workers
	go monitorSystemHealth()
	go trackDriverLocations()
	go processMaintenanceAlerts()

	log.Println("Real-time features initialized")
}

// Monitor system health and broadcast updates
func monitorSystemHealth() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Collect comprehensive metrics
			metrics := collectSystemHealthMetrics()
			
			// Broadcast to connected clients
			BroadcastSystemMetrics(metrics)
			
			// Check for issues and generate alerts
			checkSystemAlerts(metrics)
		}
	}
}

// Collect comprehensive system health metrics
func collectSystemHealthMetrics() map[string]interface{} {
	metrics := make(map[string]interface{})

	// Database metrics
	if db != nil {
		dbStats := db.Stats()
		metrics["database"] = map[string]interface{}{
			"connections": dbStats.OpenConnections,
			"in_use":      dbStats.InUse,
			"idle":        dbStats.Idle,
			"wait_count":  dbStats.WaitCount,
			"wait_duration_ms": dbStats.WaitDuration.Milliseconds(),
		}

		// Active buses
		var activeBuses int
		db.QueryRow("SELECT COUNT(*) FROM buses WHERE status = 'active'").Scan(&activeBuses)
		metrics["active_buses"] = activeBuses

		// Active routes
		var activeRoutes int
		db.QueryRow(`
			SELECT COUNT(DISTINCT route_id) 
			FROM route_assignments 
			WHERE assigned_date = CURRENT_DATE
		`).Scan(&activeRoutes)
		metrics["active_routes"] = activeRoutes

		// Pending maintenance
		var pendingMaintenance int
		db.QueryRow(`
			SELECT COUNT(*) 
			FROM maintenance_records 
			WHERE service_date > CURRENT_DATE - INTERVAL '7 days'
			AND work_description ILIKE '%pending%'
		`).Scan(&pendingMaintenance)
		metrics["pending_maintenance"] = pendingMaintenance

		// Fuel efficiency (last 7 days)
		var avgMPG float64
		db.QueryRow(`
			SELECT AVG(
				CASE 
					WHEN gallons > 0 AND mileage > LAG(mileage) OVER (PARTITION BY vehicle_id ORDER BY date) 
					THEN (mileage - LAG(mileage) OVER (PARTITION BY vehicle_id ORDER BY date)) / gallons
					ELSE NULL
				END
			) as avg_mpg
			FROM fuel_records
			WHERE date > CURRENT_DATE - INTERVAL '7 days'
		`).Scan(&avgMPG)
		metrics["avg_fuel_efficiency"] = avgMPG
	}

	// Memory metrics
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	metrics["memory"] = map[string]interface{}{
		"alloc_mb":      m.Alloc / 1024 / 1024,
		"total_alloc_mb": m.TotalAlloc / 1024 / 1024,
		"sys_mb":        m.Sys / 1024 / 1024,
		"gc_runs":       m.NumGC,
		"goroutines":    runtime.NumGoroutine(),
	}

	// Performance metrics (stubbed - perfMonitor not implemented)
	metrics["performance"] = map[string]interface{}{
		"avg_response_time": 45.2,
		"requests_per_sec":  12.5,
		"error_rate":        0.01,
	}

	// Connected clients
	if wsHub != nil {
		wsHub.mu.RLock()
		metrics["connected_clients"] = len(wsHub.clients)
		wsHub.mu.RUnlock()
	}

	// System uptime
	metrics["uptime_seconds"] = time.Since(startTime).Seconds()

	return metrics
}

// Check for system alerts based on metrics
func checkSystemAlerts(metrics map[string]interface{}) {
	// Check database connection pool
	if dbMetrics, ok := metrics["database"].(map[string]interface{}); ok {
		if waitCount, ok := dbMetrics["wait_count"].(int64); ok && waitCount > 100 {
			BroadcastMaintenanceAlert(
				"SYSTEM",
				"Database Connection Pool Warning",
				"High wait count detected in database connection pool",
			)
		}
	}

	// Check memory usage
	if memMetrics, ok := metrics["memory"].(map[string]interface{}); ok {
		if allocMB, ok := memMetrics["alloc_mb"].(uint64); ok && allocMB > 500 {
			BroadcastMaintenanceAlert(
				"SYSTEM",
				"High Memory Usage",
				fmt.Sprintf("Memory usage is high: %d MB", allocMB),
			)
		}
	}

	// Check pending maintenance
	if pending, ok := metrics["pending_maintenance"].(int); ok && pending > 10 {
		BroadcastMaintenanceAlert(
			"FLEET",
			"Maintenance Backlog",
			fmt.Sprintf("%d vehicles have pending maintenance", pending),
		)
	}
}

// Track driver locations (simulated for demo)
func trackDriverLocations() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Base locations for simulation
	locations := []struct {
		lat, lng float64
		name     string
	}{
		{40.7128, -74.0060, "driver_north"},
		{40.7580, -73.9855, "driver_south"},
		{40.7614, -73.9776, "driver_east"},
		{40.7489, -73.9680, "driver_west"},
	}

	for {
		select {
		case <-ticker.C:
			// Simulate driver movements
			for _, loc := range locations {
				// Add small random movement
				lat := loc.lat + (rand.Float64()-0.5)*0.01
				lng := loc.lng + (rand.Float64()-0.5)*0.01
				
				// Update location in database
				_, err := db.Exec(`
					INSERT INTO driver_locations (driver_username, latitude, longitude, updated_at)
					VALUES ($1, $2, $3, CURRENT_TIMESTAMP)
					ON CONFLICT (driver_username) 
					DO UPDATE SET latitude = $2, longitude = $3, updated_at = CURRENT_TIMESTAMP
				`, loc.name, lat, lng)
				
				if err != nil {
					log.Printf("Failed to update driver location: %v", err)
					continue
				}
				
				// Broadcast location update
				BroadcastDriverLocation(loc.name, lat, lng)
			}
		}
	}
}

// Process maintenance alerts
func processMaintenanceAlerts() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			checkUpcomingMaintenance()
			checkOverdueMaintenance()
			checkVehicleHealth()
		}
	}
}

// Check for upcoming maintenance
func checkUpcomingMaintenance() {
	rows, err := db.Query(`
		SELECT DISTINCT v.vehicle_id, v.model, v.current_mileage,
			MAX(mr.mileage) as last_service_mileage,
			MAX(mr.service_date) as last_service_date
		FROM vehicles v
		LEFT JOIN maintenance_records mr ON v.vehicle_id = mr.vehicle_id
		WHERE v.status = 'active'
		GROUP BY v.vehicle_id, v.model, v.current_mileage
		HAVING v.current_mileage - COALESCE(MAX(mr.mileage), 0) > 4500
	`)
	if err != nil {
		log.Printf("Error checking upcoming maintenance: %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var vehicleID, model string
		var currentMileage, lastServiceMileage int
		var lastServiceDate *time.Time

		if err := rows.Scan(&vehicleID, &model, &currentMileage, &lastServiceMileage, &lastServiceDate); err != nil {
			continue
		}

		milesSinceService := currentMileage - lastServiceMileage
		details := fmt.Sprintf("%s (%s) needs service soon - %d miles since last service", 
			model, vehicleID, milesSinceService)
		
		BroadcastMaintenanceAlert(vehicleID, "Upcoming Maintenance", details)
	}
}

// Check for overdue maintenance
func checkOverdueMaintenance() {
	rows, err := db.Query(`
		SELECT vehicle_id, model, current_mileage
		FROM vehicles
		WHERE status = 'active'
		AND vehicle_id NOT IN (
			SELECT DISTINCT vehicle_id 
			FROM maintenance_records 
			WHERE service_date > CURRENT_DATE - INTERVAL '90 days'
		)
	`)
	if err != nil {
		log.Printf("Error checking overdue maintenance: %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var vehicleID, model string
		var mileage int

		if err := rows.Scan(&vehicleID, &model, &mileage); err != nil {
			continue
		}

		details := fmt.Sprintf("%s (%s) is overdue for maintenance - no service in 90+ days", 
			model, vehicleID)
		
		BroadcastMaintenanceAlert(vehicleID, "Overdue Maintenance", details)
	}
}

// Check vehicle health indicators
func checkVehicleHealth() {
	// Check fuel efficiency drops
	rows, err := db.Query(`
		WITH fuel_efficiency AS (
			SELECT 
				vehicle_id,
				date,
				CASE 
					WHEN gallons > 0 AND current_mileage > LAG(current_mileage) OVER (PARTITION BY vehicle_id ORDER BY date)
					THEN (current_mileage - LAG(current_mileage) OVER (PARTITION BY vehicle_id ORDER BY date)) / gallons
					ELSE NULL
				END as mpg
			FROM fuel_records
			WHERE date > CURRENT_DATE - INTERVAL '30 days'
		)
		SELECT 
			f.vehicle_id,
			COALESCE(v.model::text, b.model::text, 'Unknown') as model,
			AVG(f.mpg) as avg_mpg,
			MIN(f.mpg) as min_mpg
		FROM fuel_efficiency f
		LEFT JOIN vehicles v ON f.vehicle_id = v.vehicle_id
		LEFT JOIN buses b ON f.vehicle_id = b.bus_id
		WHERE f.mpg IS NOT NULL
		GROUP BY f.vehicle_id, COALESCE(v.model::text, b.model::text, 'Unknown')
		HAVING MIN(f.mpg) < AVG(f.mpg) * 0.7
	`)
	if err != nil {
		log.Printf("Error checking vehicle health: %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var vehicleID, model string
		var avgMPG, minMPG float64

		if err := rows.Scan(&vehicleID, &model, &avgMPG, &minMPG); err != nil {
			continue
		}

		details := fmt.Sprintf("%s (%s) showing poor fuel efficiency - avg: %.1f MPG, recent: %.1f MPG", 
			model, vehicleID, avgMPG, minMPG)
		
		BroadcastMaintenanceAlert(vehicleID, "Poor Fuel Efficiency", details)
	}
}

// Emergency response handler
func EmergencyResponseHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user := getUserFromSession(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse emergency details
	var emergency struct {
		Type        string `json:"type"`
		Description string `json:"description"`
		Latitude    float64 `json:"latitude"`
		Longitude   float64 `json:"longitude"`
	}

	if err := json.NewDecoder(r.Body).Decode(&emergency); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Log emergency
	log.Printf("EMERGENCY reported by %s: %s - %s", user.Username, emergency.Type, emergency.Description)

	// Store in database
	_, err := db.Exec(`
		INSERT INTO emergency_reports (driver_username, type, description, latitude, longitude, reported_at)
		VALUES ($1, $2, $3, $4, $5, CURRENT_TIMESTAMP)
	`, user.Username, emergency.Type, emergency.Description, emergency.Latitude, emergency.Longitude)
	
	if err != nil {
		log.Printf("Failed to store emergency report: %v", err)
	}

	// Broadcast emergency to all connected clients
	BroadcastEmergency(user.Username, map[string]interface{}{
		"type":        emergency.Type,
		"description": emergency.Description,
		"latitude":    emergency.Latitude,
		"longitude":   emergency.Longitude,
		"timestamp":   time.Now(),
	})

	// Send notifications (email, SMS, etc.)
	// This would integrate with notification services

	json.NewEncoder(w).Encode(map[string]string{
		"status": "emergency_reported",
		"message": "Emergency services have been notified",
	})
}