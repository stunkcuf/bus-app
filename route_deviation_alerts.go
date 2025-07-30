package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"sync"
	"time"
)

// RouteDeviation represents a deviation from the planned route
type RouteDeviation struct {
	ID              int64             `json:"id"`
	VehicleID       string            `json:"vehicle_id"`
	RouteID         string            `json:"route_id"`
	DriverID        string            `json:"driver_id"`
	DeviationType   string            `json:"deviation_type"` // off_route, wrong_direction, stopped_too_long, skipped_stop, unauthorized_stop
	Severity        string            `json:"severity"`       // low, medium, high, critical
	Location        GPSLocation       `json:"location"`
	ExpectedLocation *GPSLocation     `json:"expected_location,omitempty"`
	Distance        float64           `json:"distance"`       // meters from expected route
	Duration        time.Duration     `json:"duration"`       // how long off route
	Description     string            `json:"description"`
	AutoResolved    bool              `json:"auto_resolved"`
	ResolvedAt      *time.Time        `json:"resolved_at,omitempty"`
	CreatedAt       time.Time         `json:"created_at"`
	Metadata        map[string]interface{} `json:"metadata"`
}

// RouteMonitor monitors vehicles for route deviations
type RouteMonitor struct {
	mu              sync.RWMutex
	activeRoutes    map[string]*ActiveRoute    // vehicleID -> active route
	deviations      map[string]*RouteDeviation // vehicleID -> current deviation
	alerts          chan RouteDeviation
	stopChan        chan struct{}
	checkInterval   time.Duration
	deviationRadius float64 // meters
	stopDuration    time.Duration
}

// ActiveRoute represents a currently active route being monitored
type ActiveRoute struct {
	RouteID         string          `json:"route_id"`
	VehicleID       string          `json:"vehicle_id"`
	DriverID        string          `json:"driver_id"`
	PlannedStops    []RouteStop     `json:"planned_stops"`
	CompletedStops  map[int]bool    `json:"completed_stops"`
	CurrentStopIndex int            `json:"current_stop_index"`
	StartTime       time.Time       `json:"start_time"`
	LastUpdate      time.Time       `json:"last_update"`
	LastLocation    *GPSLocation    `json:"last_location"`
	Status          string          `json:"status"` // on_route, deviated, stopped, completed
}

// RouteStop represents a planned stop on a route
type RouteStop struct {
	ID           int       `json:"id"`
	StopNumber   int       `json:"stop_number"`
	Name         string    `json:"name"`
	Latitude     float64   `json:"latitude"`
	Longitude    float64   `json:"longitude"`
	ArrivalTime  time.Time `json:"arrival_time"`
	DepartureTime time.Time `json:"departure_time"`
	StudentCount int       `json:"student_count"`
	StopRadius   float64   `json:"stop_radius"` // meters
}

var routeMonitor *RouteMonitor

// InitializeRouteMonitor sets up the route deviation monitoring system
func InitializeRouteMonitor() error {
	routeMonitor = &RouteMonitor{
		activeRoutes:    make(map[string]*ActiveRoute),
		deviations:      make(map[string]*RouteDeviation),
		alerts:          make(chan RouteDeviation, 100),
		stopChan:        make(chan struct{}),
		checkInterval:   10 * time.Second,
		deviationRadius: 200, // 200 meters default
		stopDuration:    5 * time.Minute,
	}

	// Create deviation tables
	if err := createDeviationTables(); err != nil {
		return err
	}

	// Start monitoring
	go routeMonitor.startMonitoring()
	go routeMonitor.processAlerts()

	log.Println("Route deviation monitoring initialized")
	return nil
}

// createDeviationTables creates the necessary database tables
func createDeviationTables() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS route_deviations (
			id BIGSERIAL PRIMARY KEY,
			vehicle_id VARCHAR(50) NOT NULL,
			route_id VARCHAR(50) NOT NULL,
			driver_id VARCHAR(50) NOT NULL,
			deviation_type VARCHAR(50) NOT NULL,
			severity VARCHAR(20) NOT NULL,
			location JSONB NOT NULL,
			expected_location JSONB,
			distance DOUBLE PRECISION,
			duration BIGINT, -- milliseconds
			description TEXT,
			auto_resolved BOOLEAN DEFAULT FALSE,
			resolved_at TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			metadata JSONB
		)`,
		
		`CREATE INDEX IF NOT EXISTS idx_route_deviations_vehicle 
		 ON route_deviations(vehicle_id, created_at DESC)`,
		
		`CREATE INDEX IF NOT EXISTS idx_route_deviations_unresolved 
		 ON route_deviations(vehicle_id) WHERE resolved_at IS NULL`,
		
		`CREATE TABLE IF NOT EXISTS route_plans (
			id SERIAL PRIMARY KEY,
			route_id VARCHAR(50) NOT NULL REFERENCES routes(route_id),
			stop_number INTEGER NOT NULL,
			stop_name VARCHAR(255),
			latitude DOUBLE PRECISION NOT NULL,
			longitude DOUBLE PRECISION NOT NULL,
			planned_arrival TIME,
			planned_departure TIME,
			stop_duration INTEGER DEFAULT 60, -- seconds
			stop_radius DOUBLE PRECISION DEFAULT 50, -- meters
			metadata JSONB,
			UNIQUE(route_id, stop_number)
		)`,
		
		`CREATE INDEX IF NOT EXISTS idx_route_plans_route 
		 ON route_plans(route_id, stop_number)`,
	}

	for _, query := range queries {
		if _, err := db.Exec(query); err != nil {
			return fmt.Errorf("failed to create deviation tables: %w", err)
		}
	}

	return nil
}

// StartRouteMonitoring starts monitoring a vehicle on a route
func (rm *RouteMonitor) StartRouteMonitoring(vehicleID, routeID, driverID string) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	// Load route plan
	stops, err := loadRoutePlan(routeID)
	if err != nil {
		return fmt.Errorf("failed to load route plan: %w", err)
	}

	activeRoute := &ActiveRoute{
		RouteID:          routeID,
		VehicleID:        vehicleID,
		DriverID:         driverID,
		PlannedStops:     stops,
		CompletedStops:   make(map[int]bool),
		CurrentStopIndex: 0,
		StartTime:        time.Now(),
		LastUpdate:       time.Now(),
		Status:           "on_route",
	}

	rm.activeRoutes[vehicleID] = activeRoute
	
	log.Printf("Started route monitoring for vehicle %s on route %s", vehicleID, routeID)
	return nil
}

// StopRouteMonitoring stops monitoring a vehicle
func (rm *RouteMonitor) StopRouteMonitoring(vehicleID string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	delete(rm.activeRoutes, vehicleID)
	delete(rm.deviations, vehicleID)
	
	log.Printf("Stopped route monitoring for vehicle %s", vehicleID)
}

// startMonitoring continuously monitors all active routes
func (rm *RouteMonitor) startMonitoring() {
	ticker := time.NewTicker(rm.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rm.checkAllRoutes()
		case <-rm.stopChan:
			return
		}
	}
}

// checkAllRoutes checks all active routes for deviations
func (rm *RouteMonitor) checkAllRoutes() {
	rm.mu.RLock()
	vehicles := make([]string, 0, len(rm.activeRoutes))
	for vehicleID := range rm.activeRoutes {
		vehicles = append(vehicles, vehicleID)
	}
	rm.mu.RUnlock()

	for _, vehicleID := range vehicles {
		if err := rm.checkVehicleRoute(vehicleID); err != nil {
			log.Printf("Error checking route for vehicle %s: %v", vehicleID, err)
		}
	}
}

// checkVehicleRoute checks a specific vehicle for route deviations
func (rm *RouteMonitor) checkVehicleRoute(vehicleID string) error {
	rm.mu.RLock()
	activeRoute, exists := rm.activeRoutes[vehicleID]
	rm.mu.RUnlock()

	if !exists {
		return nil
	}

	// Get current location
	location, err := gpsTracker.GetLatestLocation(vehicleID)
	if err != nil || location == nil {
		return fmt.Errorf("failed to get vehicle location: %w", err)
	}

	// Update last location
	activeRoute.LastLocation = location
	activeRoute.LastUpdate = time.Now()

	// Check various deviation types
	rm.checkOffRoute(activeRoute, location)
	rm.checkStoppedTooLong(activeRoute, location)
	rm.checkSkippedStop(activeRoute, location)
	rm.checkUnauthorizedStop(activeRoute, location)
	rm.checkWrongDirection(activeRoute, location)

	return nil
}

// checkOffRoute checks if vehicle is off the planned route
func (rm *RouteMonitor) checkOffRoute(route *ActiveRoute, location *GPSLocation) {
	// Find nearest point on route
	nearestStop, distance := rm.findNearestPlannedStop(route, location.Latitude, location.Longitude)
	
	if distance > rm.deviationRadius {
		deviation := &RouteDeviation{
			VehicleID:     route.VehicleID,
			RouteID:       route.RouteID,
			DriverID:      route.DriverID,
			DeviationType: "off_route",
			Severity:      rm.calculateSeverity(distance),
			Location:      *location,
			ExpectedLocation: &GPSLocation{
				Latitude:  nearestStop.Latitude,
				Longitude: nearestStop.Longitude,
			},
			Distance:    distance,
			Description: fmt.Sprintf("Vehicle is %.0f meters off the planned route", distance),
			CreatedAt:   time.Now(),
		}

		rm.handleDeviation(deviation)
	} else {
		// Vehicle is back on route, resolve any existing deviation
		rm.resolveDeviation(route.VehicleID, "off_route")
	}
}

// checkStoppedTooLong checks if vehicle has been stopped too long
func (rm *RouteMonitor) checkStoppedTooLong(route *ActiveRoute, location *GPSLocation) {
	if location.Speed > 5 { // Vehicle is moving
		rm.resolveDeviation(route.VehicleID, "stopped_too_long")
		return
	}

	// Check if stopped at a planned stop
	_, distance := rm.findNearestPlannedStop(route, location.Latitude, location.Longitude)
	if distance <= 50 { // At a planned stop
		return
	}

	// Check existing deviation
	rm.mu.RLock()
	existingDev, exists := rm.deviations[route.VehicleID]
	rm.mu.RUnlock()

	if exists && existingDev.DeviationType == "stopped_too_long" {
		// Update duration
		existingDev.Duration = time.Since(existingDev.CreatedAt)
		
		if existingDev.Duration > rm.stopDuration && existingDev.Severity != "high" {
			existingDev.Severity = "high"
			rm.alerts <- *existingDev
		}
	} else {
		// Create new stopped deviation
		deviation := &RouteDeviation{
			VehicleID:     route.VehicleID,
			RouteID:       route.RouteID,
			DriverID:      route.DriverID,
			DeviationType: "stopped_too_long",
			Severity:      "low",
			Location:      *location,
			Description:   "Vehicle stopped at unauthorized location",
			CreatedAt:     time.Now(),
		}

		rm.handleDeviation(deviation)
	}
}

// checkSkippedStop checks if vehicle skipped a planned stop
func (rm *RouteMonitor) checkSkippedStop(route *ActiveRoute, location *GPSLocation) {
	if route.CurrentStopIndex >= len(route.PlannedStops) {
		return
	}

	currentStop := route.PlannedStops[route.CurrentStopIndex]
	distance := calculateDistance(location.Latitude, location.Longitude, 
		currentStop.Latitude, currentStop.Longitude)

	// If vehicle has moved past the stop without stopping
	if route.CurrentStopIndex < len(route.PlannedStops)-1 {
		nextStop := route.PlannedStops[route.CurrentStopIndex+1]
		distanceToNext := calculateDistance(location.Latitude, location.Longitude,
			nextStop.Latitude, nextStop.Longitude)

		if distanceToNext < distance && !route.CompletedStops[currentStop.StopNumber] {
			deviation := &RouteDeviation{
				VehicleID:     route.VehicleID,
				RouteID:       route.RouteID,
				DriverID:      route.DriverID,
				DeviationType: "skipped_stop",
				Severity:      "high",
				Location:      *location,
				ExpectedLocation: &GPSLocation{
					Latitude:  currentStop.Latitude,
					Longitude: currentStop.Longitude,
				},
				Description: fmt.Sprintf("Skipped stop: %s", currentStop.Name),
				CreatedAt:   time.Now(),
				Metadata: map[string]interface{}{
					"stop_number": currentStop.StopNumber,
					"stop_name":   currentStop.Name,
				},
			}

			rm.handleDeviation(deviation)
			route.CurrentStopIndex++ // Move to next stop
		}
	}

	// Check if at current stop
	if distance <= currentStop.StopRadius {
		route.CompletedStops[currentStop.StopNumber] = true
		if route.CurrentStopIndex < len(route.PlannedStops)-1 {
			route.CurrentStopIndex++
		}
	}
}

// checkUnauthorizedStop checks for stops at unauthorized locations
func (rm *RouteMonitor) checkUnauthorizedStop(route *ActiveRoute, location *GPSLocation) {
	if location.Speed > 5 { // Vehicle is moving
		return
	}

	// Check if at a planned stop
	nearestStop, distance := rm.findNearestPlannedStop(route, location.Latitude, location.Longitude)
	if distance <= nearestStop.StopRadius {
		return // At authorized stop
	}

	// Check for existing unauthorized stop
	rm.mu.RLock()
	existingDev, exists := rm.deviations[route.VehicleID]
	rm.mu.RUnlock()

	if !exists || existingDev.DeviationType != "unauthorized_stop" {
		deviation := &RouteDeviation{
			VehicleID:     route.VehicleID,
			RouteID:       route.RouteID,
			DriverID:      route.DriverID,
			DeviationType: "unauthorized_stop",
			Severity:      "medium",
			Location:      *location,
			Distance:      distance,
			Description:   fmt.Sprintf("Unauthorized stop %.0f meters from nearest planned stop", distance),
			CreatedAt:     time.Now(),
		}

		rm.handleDeviation(deviation)
	}
}

// checkWrongDirection checks if vehicle is going in wrong direction
func (rm *RouteMonitor) checkWrongDirection(route *ActiveRoute, location *GPSLocation) {
	if route.CurrentStopIndex >= len(route.PlannedStops) {
		return
	}

	// Calculate expected heading to next stop
	nextStop := route.PlannedStops[route.CurrentStopIndex]
	expectedHeading := calculateBearing(location.Latitude, location.Longitude,
		nextStop.Latitude, nextStop.Longitude)

	// Calculate heading difference
	headingDiff := math.Abs(location.Heading - expectedHeading)
	if headingDiff > 180 {
		headingDiff = 360 - headingDiff
	}

	// If heading is more than 90 degrees off and vehicle is moving
	if headingDiff > 90 && location.Speed > 10 {
		deviation := &RouteDeviation{
			VehicleID:     route.VehicleID,
			RouteID:       route.RouteID,
			DriverID:      route.DriverID,
			DeviationType: "wrong_direction",
			Severity:      "medium",
			Location:      *location,
			Description:   fmt.Sprintf("Vehicle heading wrong direction (%.0f° off course)", headingDiff),
			CreatedAt:     time.Now(),
			Metadata: map[string]interface{}{
				"expected_heading": expectedHeading,
				"actual_heading":   location.Heading,
				"heading_diff":     headingDiff,
			},
		}

		rm.handleDeviation(deviation)
	} else {
		rm.resolveDeviation(route.VehicleID, "wrong_direction")
	}
}

// handleDeviation processes a new or updated deviation
func (rm *RouteMonitor) handleDeviation(deviation *RouteDeviation) {
	rm.mu.Lock()
	existingDev, exists := rm.deviations[deviation.VehicleID]
	
	if !exists || existingDev.DeviationType != deviation.DeviationType {
		// New deviation
		rm.deviations[deviation.VehicleID] = deviation
		rm.mu.Unlock()
		
		// Save to database
		if err := saveDeviation(deviation); err != nil {
			log.Printf("Failed to save deviation: %v", err)
		}
		
		// Send alert
		rm.alerts <- *deviation
	} else {
		// Update existing
		existingDev.Duration = time.Since(existingDev.CreatedAt)
		existingDev.Location = deviation.Location
		existingDev.Distance = deviation.Distance
		rm.mu.Unlock()
	}
}

// resolveDeviation marks a deviation as resolved
func (rm *RouteMonitor) resolveDeviation(vehicleID, deviationType string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if dev, exists := rm.deviations[vehicleID]; exists && dev.DeviationType == deviationType {
		dev.AutoResolved = true
		now := time.Now()
		dev.ResolvedAt = &now
		
		// Update in database
		if err := updateDeviationResolved(dev.ID, true, &now); err != nil {
			log.Printf("Failed to update deviation resolution: %v", err)
		}
		
		delete(rm.deviations, vehicleID)
	}
}

// processAlerts handles deviation alerts
func (rm *RouteMonitor) processAlerts() {
	for deviation := range rm.alerts {
		// Log deviation
		log.Printf("Route Deviation Alert: Vehicle %s - %s (%s severity)",
			deviation.VehicleID, deviation.Description, deviation.Severity)
		
		// Send notifications based on severity
		if deviation.Severity == "high" || deviation.Severity == "critical" {
			sendDeviationNotification(&deviation)
		}
		
		// Broadcast to WebSocket clients
		broadcastDeviation(&deviation)
	}
}

// Helper functions

func (rm *RouteMonitor) findNearestPlannedStop(route *ActiveRoute, lat, lng float64) (RouteStop, float64) {
	if len(route.PlannedStops) == 0 {
		return RouteStop{}, math.MaxFloat64
	}

	nearestStop := route.PlannedStops[0]
	minDistance := calculateDistance(lat, lng, nearestStop.Latitude, nearestStop.Longitude)

	for _, stop := range route.PlannedStops[1:] {
		distance := calculateDistance(lat, lng, stop.Latitude, stop.Longitude)
		if distance < minDistance {
			minDistance = distance
			nearestStop = stop
		}
	}

	return nearestStop, minDistance
}

func (rm *RouteMonitor) calculateSeverity(distance float64) string {
	switch {
	case distance < 500:
		return "low"
	case distance < 1000:
		return "medium"
	case distance < 2000:
		return "high"
	default:
		return "critical"
	}
}

// calculateDistance is already defined in gps_tracking.go

// calculateBearing calculates bearing from point 1 to point 2 in degrees
func calculateBearing(lat1, lng1, lat2, lng2 float64) float64 {
	lat1Rad := lat1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	deltaLng := (lng2 - lng1) * math.Pi / 180
	
	y := math.Sin(deltaLng) * math.Cos(lat2Rad)
	x := math.Cos(lat1Rad)*math.Sin(lat2Rad) -
		math.Sin(lat1Rad)*math.Cos(lat2Rad)*math.Cos(deltaLng)
	
	bearing := math.Atan2(y, x) * 180 / math.Pi
	
	// Normalize to 0-360
	return math.Mod(bearing+360, 360)
}

// Database functions

func loadRoutePlan(routeID string) ([]RouteStop, error) {
	query := `
		SELECT stop_number, stop_name, latitude, longitude,
		       planned_arrival, planned_departure, stop_radius
		FROM route_plans
		WHERE route_id = $1
		ORDER BY stop_number
	`
	
	rows, err := db.Query(query, routeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var stops []RouteStop
	for rows.Next() {
		var stop RouteStop
		var arrivalTime, departureTime sql.NullTime
		
		err := rows.Scan(&stop.StopNumber, &stop.Name, &stop.Latitude, &stop.Longitude,
			&arrivalTime, &departureTime, &stop.StopRadius)
		if err != nil {
			continue
		}
		
		if arrivalTime.Valid {
			stop.ArrivalTime = arrivalTime.Time
		}
		if departureTime.Valid {
			stop.DepartureTime = departureTime.Time
		}
		
		stops = append(stops, stop)
	}
	
	return stops, nil
}

func saveDeviation(deviation *RouteDeviation) error {
	locationJSON, _ := json.Marshal(deviation.Location)
	var expectedLocationJSON []byte
	if deviation.ExpectedLocation != nil {
		expectedLocationJSON, _ = json.Marshal(deviation.ExpectedLocation)
	}
	metadataJSON, _ := json.Marshal(deviation.Metadata)
	
	query := `
		INSERT INTO route_deviations 
		(vehicle_id, route_id, driver_id, deviation_type, severity,
		 location, expected_location, distance, duration, description,
		 auto_resolved, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id
	`
	
	err := db.QueryRow(query,
		deviation.VehicleID, deviation.RouteID, deviation.DriverID,
		deviation.DeviationType, deviation.Severity, locationJSON,
		expectedLocationJSON, deviation.Distance, int64(deviation.Duration/time.Millisecond),
		deviation.Description, deviation.AutoResolved, metadataJSON, deviation.CreatedAt,
	).Scan(&deviation.ID)
	
	return err
}

func updateDeviationResolved(deviationID int64, autoResolved bool, resolvedAt *time.Time) error {
	query := `
		UPDATE route_deviations 
		SET auto_resolved = $1, resolved_at = $2
		WHERE id = $3
	`
	
	_, err := db.Exec(query, autoResolved, resolvedAt, deviationID)
	return err
}

// Notification functions

func sendDeviationNotification(deviation *RouteDeviation) {
	notification := Notification{
		ID:       generateNotificationID(),
		Type:     "route_deviation",
		Priority: deviation.Severity,
		Subject:  fmt.Sprintf("Route Deviation: %s", deviation.VehicleID),
		Message:  deviation.Description,
		Data: map[string]interface{}{
			"deviation": deviation,
		},
		Channels:  []string{"in-app", "push"},
		CreatedAt: time.Now(),
	}
	
	// Get managers to notify
	recipients := getManagerRecipients()
	notification.Recipients = recipients
	
	if notificationSystem != nil {
		notificationSystem.Send(notification)
	}
}

func broadcastDeviation(deviation *RouteDeviation) {
	if wsHub == nil {
		return
	}
	
	message := WSMessage{
		Type: "route_deviation",
		Data: map[string]interface{}{
			"deviation": deviation,
		},
		Timestamp: time.Now(),
	}
	
	messageJSON, _ := json.Marshal(message)
	
	wsHub.mu.RLock()
	defer wsHub.mu.RUnlock()
	
	for client := range wsHub.clients {
		select {
		case client.send <- messageJSON:
		default:
		}
	}
}

func getManagerRecipients() []Recipient {
	var recipients []Recipient
	
	rows, _ := db.Query(`
		SELECT id, username, email 
		FROM users 
		WHERE role = 'manager' AND approved = true
	`)
	defer rows.Close()
	
	for rows.Next() {
		var userID int
		var username, email string
		if err := rows.Scan(&userID, &username, &email); err == nil {
			recipients = append(recipients, Recipient{
				UserID:   fmt.Sprintf("%d", userID),
				Username: username,
				Email:    email,
			})
		}
	}
	
	return recipients
}