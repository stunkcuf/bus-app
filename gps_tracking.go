package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"
	
	"github.com/gorilla/websocket"
	"github.com/jmoiron/sqlx"
)

// GPSLocation represents a GPS coordinate with metadata
type GPSLocation struct {
	ID           int64     `json:"id" db:"id"`
	VehicleID    string    `json:"vehicle_id" db:"vehicle_id"`
	Latitude     float64   `json:"latitude" db:"latitude"`
	Longitude    float64   `json:"longitude" db:"longitude"`
	Speed        float64   `json:"speed" db:"speed"`         // km/h
	Heading      float64   `json:"heading" db:"heading"`     // degrees
	Accuracy     float64   `json:"accuracy" db:"accuracy"`   // meters
	Altitude     float64   `json:"altitude" db:"altitude"`   // meters
	Timestamp    time.Time `json:"timestamp" db:"timestamp"`
	DriverID     string    `json:"driver_id" db:"driver_id"`
	RouteID      string    `json:"route_id" db:"route_id"`
	Status       string    `json:"status" db:"status"`       // active, idle, offline
	BatteryLevel int       `json:"battery_level" db:"battery_level"` // percentage
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

// GPSTracker manages real-time GPS tracking
type GPSTracker struct {
	db          *sqlx.DB
	mu          sync.RWMutex
	locations   map[string]*GPSLocation // vehicleID -> latest location
	subscribers map[string]map[*websocket.Conn]bool // vehicleID -> connections
	broadcast   chan GPSBroadcast
}

// GPSBroadcast represents a location update to broadcast
type GPSBroadcast struct {
	VehicleID string
	Location  *GPSLocation
}

// Global GPS tracker instance
var gpsTracker *GPSTracker

// InitializeGPSTracking sets up the GPS tracking system
func InitializeGPSTracking(database *sqlx.DB) error {
	gpsTracker = &GPSTracker{
		db:          database,
		locations:   make(map[string]*GPSLocation),
		subscribers: make(map[string]map[*websocket.Conn]bool),
		broadcast:   make(chan GPSBroadcast, 100),
	}
	
	// Create GPS tracking tables
	if err := createGPSTrackingTables(database); err != nil {
		return fmt.Errorf("failed to create GPS tracking tables: %w", err)
	}
	
	// Load latest locations for active vehicles
	if err := gpsTracker.loadLatestLocations(); err != nil {
		log.Printf("Warning: Failed to load latest GPS locations: %v", err)
	}
	
	// Start broadcast handler
	go gpsTracker.handleBroadcasts()
	
	// Start cleanup routine for old GPS data
	go gpsTracker.startCleanupRoutine()
	
	return nil
}

// createGPSTrackingTables creates the necessary database tables
func createGPSTrackingTables(db *sqlx.DB) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS gps_locations (
			id BIGSERIAL PRIMARY KEY,
			vehicle_id VARCHAR(50) NOT NULL REFERENCES vehicles(vehicle_id),
			latitude DOUBLE PRECISION NOT NULL,
			longitude DOUBLE PRECISION NOT NULL,
			speed DOUBLE PRECISION DEFAULT 0,
			heading DOUBLE PRECISION DEFAULT 0,
			accuracy DOUBLE PRECISION DEFAULT 0,
			altitude DOUBLE PRECISION DEFAULT 0,
			timestamp TIMESTAMP NOT NULL,
			driver_id VARCHAR(50) REFERENCES users(username),
			route_id VARCHAR(50) REFERENCES routes(route_id),
			status VARCHAR(20) DEFAULT 'active',
			battery_level INTEGER DEFAULT 100,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			CONSTRAINT valid_latitude CHECK (latitude >= -90 AND latitude <= 90),
			CONSTRAINT valid_longitude CHECK (longitude >= -180 AND longitude <= 180),
			CONSTRAINT valid_speed CHECK (speed >= 0),
			CONSTRAINT valid_battery CHECK (battery_level >= 0 AND battery_level <= 100)
		)`,
		
		`CREATE INDEX IF NOT EXISTS idx_gps_locations_vehicle_timestamp 
		 ON gps_locations(vehicle_id, timestamp DESC)`,
		
		`CREATE INDEX IF NOT EXISTS idx_gps_locations_timestamp 
		 ON gps_locations(timestamp DESC)`,
		
		`CREATE INDEX IF NOT EXISTS idx_gps_locations_driver 
		 ON gps_locations(driver_id)`,
		
		// Table for storing GPS tracking sessions
		`CREATE TABLE IF NOT EXISTS gps_tracking_sessions (
			id SERIAL PRIMARY KEY,
			vehicle_id VARCHAR(50) NOT NULL REFERENCES vehicles(vehicle_id),
			driver_id VARCHAR(50) NOT NULL REFERENCES users(username),
			route_id VARCHAR(50) REFERENCES routes(route_id),
			start_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			end_time TIMESTAMP,
			start_location JSONB,
			end_location JSONB,
			total_distance DOUBLE PRECISION DEFAULT 0,
			average_speed DOUBLE PRECISION DEFAULT 0,
			max_speed DOUBLE PRECISION DEFAULT 0,
			total_stops INTEGER DEFAULT 0,
			status VARCHAR(20) DEFAULT 'active'
		)`,
		
		// Table for geofences (e.g., school zones, stops)
		`CREATE TABLE IF NOT EXISTS geofences (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			type VARCHAR(50) NOT NULL, -- school, stop, zone
			center_latitude DOUBLE PRECISION NOT NULL,
			center_longitude DOUBLE PRECISION NOT NULL,
			radius_meters DOUBLE PRECISION NOT NULL,
			metadata JSONB,
			active BOOLEAN DEFAULT true,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		
		// Table for geofence events
		`CREATE TABLE IF NOT EXISTS geofence_events (
			id BIGSERIAL PRIMARY KEY,
			vehicle_id VARCHAR(50) NOT NULL REFERENCES vehicles(vehicle_id),
			geofence_id INTEGER NOT NULL REFERENCES geofences(id),
			event_type VARCHAR(20) NOT NULL, -- enter, exit
			timestamp TIMESTAMP NOT NULL,
			location JSONB,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
	}
	
	for _, query := range queries {
		if _, err := db.Exec(query); err != nil {
			return fmt.Errorf("failed to execute query: %w", err)
		}
	}
	
	return nil
}

// loadLatestLocations loads the most recent location for each vehicle
func (gt *GPSTracker) loadLatestLocations() error {
	query := `
		SELECT DISTINCT ON (vehicle_id) 
			id, vehicle_id, latitude, longitude, speed, heading, 
			accuracy, altitude, timestamp, driver_id, route_id, 
			status, battery_level, created_at
		FROM gps_locations
		WHERE timestamp > NOW() - INTERVAL '24 hours'
		ORDER BY vehicle_id, timestamp DESC
	`
	
	var locations []GPSLocation
	if err := gt.db.Select(&locations, query); err != nil {
		return err
	}
	
	gt.mu.Lock()
	defer gt.mu.Unlock()
	
	for i := range locations {
		gt.locations[locations[i].VehicleID] = &locations[i]
	}
	
	log.Printf("Loaded %d latest GPS locations", len(locations))
	return nil
}

// UpdateLocation updates a vehicle's GPS location
func (gt *GPSTracker) UpdateLocation(location *GPSLocation) error {
	// Validate location
	if err := validateGPSLocation(location); err != nil {
		return fmt.Errorf("invalid GPS location: %w", err)
	}
	
	// Store in database
	query := `
		INSERT INTO gps_locations (
			vehicle_id, latitude, longitude, speed, heading, 
			accuracy, altitude, timestamp, driver_id, route_id, 
			status, battery_level
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
		) RETURNING id, created_at
	`
	
	err := gt.db.QueryRow(
		query,
		location.VehicleID, location.Latitude, location.Longitude,
		location.Speed, location.Heading, location.Accuracy,
		location.Altitude, location.Timestamp, location.DriverID,
		location.RouteID, location.Status, location.BatteryLevel,
	).Scan(&location.ID, &location.CreatedAt)
	
	if err != nil {
		return fmt.Errorf("failed to store GPS location: %w", err)
	}
	
	// Update in-memory cache
	gt.mu.Lock()
	gt.locations[location.VehicleID] = location
	gt.mu.Unlock()
	
	// Broadcast to subscribers
	gt.broadcast <- GPSBroadcast{
		VehicleID: location.VehicleID,
		Location:  location,
	}
	
	// Check geofences
	go gt.checkGeofences(location)
	
	return nil
}

// GetLatestLocation returns the most recent location for a vehicle
func (gt *GPSTracker) GetLatestLocation(vehicleID string) (*GPSLocation, error) {
	gt.mu.RLock()
	location, exists := gt.locations[vehicleID]
	gt.mu.RUnlock()
	
	if exists {
		return location, nil
	}
	
	// Try to load from database
	var loc GPSLocation
	query := `
		SELECT id, vehicle_id, latitude, longitude, speed, heading, 
		       accuracy, altitude, timestamp, driver_id, route_id, 
		       status, battery_level, created_at
		FROM gps_locations
		WHERE vehicle_id = $1
		ORDER BY timestamp DESC
		LIMIT 1
	`
	
	err := gt.db.Get(&loc, query, vehicleID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	
	// Cache it
	gt.mu.Lock()
	gt.locations[vehicleID] = &loc
	gt.mu.Unlock()
	
	return &loc, nil
}

// GetLocationHistory returns location history for a vehicle
func (gt *GPSTracker) GetLocationHistory(vehicleID string, start, end time.Time) ([]GPSLocation, error) {
	query := `
		SELECT id, vehicle_id, latitude, longitude, speed, heading, 
		       accuracy, altitude, timestamp, driver_id, route_id, 
		       status, battery_level, created_at
		FROM gps_locations
		WHERE vehicle_id = $1 AND timestamp BETWEEN $2 AND $3
		ORDER BY timestamp ASC
	`
	
	var locations []GPSLocation
	err := gt.db.Select(&locations, query, vehicleID, start, end)
	return locations, err
}

// Subscribe adds a WebSocket connection to receive updates for a vehicle
func (gt *GPSTracker) Subscribe(vehicleID string, conn *websocket.Conn) {
	gt.mu.Lock()
	defer gt.mu.Unlock()
	
	if gt.subscribers[vehicleID] == nil {
		gt.subscribers[vehicleID] = make(map[*websocket.Conn]bool)
	}
	gt.subscribers[vehicleID][conn] = true
}

// Unsubscribe removes a WebSocket connection
func (gt *GPSTracker) Unsubscribe(vehicleID string, conn *websocket.Conn) {
	gt.mu.Lock()
	defer gt.mu.Unlock()
	
	if subs, exists := gt.subscribers[vehicleID]; exists {
		delete(subs, conn)
		if len(subs) == 0 {
			delete(gt.subscribers, vehicleID)
		}
	}
}

// handleBroadcasts sends location updates to subscribers
func (gt *GPSTracker) handleBroadcasts() {
	for broadcast := range gt.broadcast {
		gt.mu.RLock()
		subscribers := gt.subscribers[broadcast.VehicleID]
		gt.mu.RUnlock()
		
		if len(subscribers) == 0 {
			continue
		}
		
		message, err := json.Marshal(broadcast.Location)
		if err != nil {
			log.Printf("Failed to marshal GPS location: %v", err)
			continue
		}
		
		// Send to all subscribers
		for conn := range subscribers {
			if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("Failed to send GPS update: %v", err)
				gt.Unsubscribe(broadcast.VehicleID, conn)
				conn.Close()
			}
		}
	}
}

// checkGeofences checks if a location triggers any geofence events
func (gt *GPSTracker) checkGeofences(location *GPSLocation) {
	query := `
		SELECT id, name, type, center_latitude, center_longitude, radius_meters
		FROM geofences
		WHERE active = true
	`
	
	var geofences []struct {
		ID              int     `db:"id"`
		Name            string  `db:"name"`
		Type            string  `db:"type"`
		CenterLatitude  float64 `db:"center_latitude"`
		CenterLongitude float64 `db:"center_longitude"`
		RadiusMeters    float64 `db:"radius_meters"`
	}
	
	if err := gt.db.Select(&geofences, query); err != nil {
		log.Printf("Failed to load geofences: %v", err)
		return
	}
	
	for _, gf := range geofences {
		distance := calculateDistance(
			location.Latitude, location.Longitude,
			gf.CenterLatitude, gf.CenterLongitude,
		)
		
		if distance <= gf.RadiusMeters {
			// Check if this is a new entry
			gt.recordGeofenceEvent(location.VehicleID, gf.ID, "enter", location)
		}
	}
}

// recordGeofenceEvent records a geofence entry/exit event
func (gt *GPSTracker) recordGeofenceEvent(vehicleID string, geofenceID int, eventType string, location *GPSLocation) {
	// Check if we already have a recent event
	var count int
	checkQuery := `
		SELECT COUNT(*) FROM geofence_events
		WHERE vehicle_id = $1 AND geofence_id = $2 
		AND event_type = $3 AND timestamp > NOW() - INTERVAL '5 minutes'
	`
	
	gt.db.Get(&count, checkQuery, vehicleID, geofenceID, eventType)
	if count > 0 {
		return // Already recorded
	}
	
	// Record the event
	locationJSON, _ := json.Marshal(map[string]interface{}{
		"latitude":  location.Latitude,
		"longitude": location.Longitude,
		"speed":     location.Speed,
	})
	
	insertQuery := `
		INSERT INTO geofence_events (vehicle_id, geofence_id, event_type, timestamp, location)
		VALUES ($1, $2, $3, $4, $5)
	`
	
	if _, err := gt.db.Exec(insertQuery, vehicleID, geofenceID, eventType, location.Timestamp, locationJSON); err != nil {
		log.Printf("Failed to record geofence event: %v", err)
	}
}

// startCleanupRoutine removes old GPS data periodically
func (gt *GPSTracker) startCleanupRoutine() {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()
	
	for range ticker.C {
		// Keep only last 30 days of GPS data
		query := `
			DELETE FROM gps_locations 
			WHERE timestamp < NOW() - INTERVAL '30 days'
		`
		
		result, err := gt.db.Exec(query)
		if err != nil {
			log.Printf("Failed to cleanup old GPS data: %v", err)
			continue
		}
		
		rows, _ := result.RowsAffected()
		if rows > 0 {
			log.Printf("Cleaned up %d old GPS records", rows)
		}
	}
}

// validateGPSLocation validates GPS location data
func validateGPSLocation(loc *GPSLocation) error {
	if loc.VehicleID == "" {
		return fmt.Errorf("vehicle ID is required")
	}
	
	if loc.Latitude < -90 || loc.Latitude > 90 {
		return fmt.Errorf("invalid latitude: %f", loc.Latitude)
	}
	
	if loc.Longitude < -180 || loc.Longitude > 180 {
		return fmt.Errorf("invalid longitude: %f", loc.Longitude)
	}
	
	if loc.Speed < 0 {
		return fmt.Errorf("speed cannot be negative: %f", loc.Speed)
	}
	
	if loc.BatteryLevel < 0 || loc.BatteryLevel > 100 {
		return fmt.Errorf("battery level must be 0-100: %d", loc.BatteryLevel)
	}
	
	return nil
}

// calculateDistance calculates distance between two GPS coordinates in meters
func calculateDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371000 // Earth's radius in meters
	
	lat1Rad := lat1 * (3.14159265359 / 180)
	lat2Rad := lat2 * (3.14159265359 / 180)
	deltaLat := (lat2 - lat1) * (3.14159265359 / 180)
	deltaLon := (lon2 - lon1) * (3.14159265359 / 180)
	
	a := (sin(deltaLat/2) * sin(deltaLat/2)) +
		(cos(lat1Rad) * cos(lat2Rad) * sin(deltaLon/2) * sin(deltaLon/2))
	c := 2 * atan2(sqrt(a), sqrt(1-a))
	
	return R * c
}

// Helper math functions
func sin(x float64) float64 {
	// Simple sine approximation
	return x - (x*x*x)/6 + (x*x*x*x*x)/120
}

func cos(x float64) float64 {
	// Simple cosine approximation
	return 1 - (x*x)/2 + (x*x*x*x)/24
}

func sqrt(x float64) float64 {
	// Newton's method for square root
	if x == 0 {
		return 0
	}
	guess := x
	for i := 0; i < 10; i++ {
		guess = (guess + x/guess) / 2
	}
	return guess
}

func atan2(y, x float64) float64 {
	// Simple atan2 approximation
	if x > 0 {
		return atan(y / x)
	}
	if x < 0 && y >= 0 {
		return atan(y/x) + 3.14159265359
	}
	if x < 0 && y < 0 {
		return atan(y/x) - 3.14159265359
	}
	if x == 0 && y > 0 {
		return 3.14159265359 / 2
	}
	if x == 0 && y < 0 {
		return -3.14159265359 / 2
	}
	return 0
}

func atan(x float64) float64 {
	// Simple arctangent approximation
	return x - (x*x*x)/3 + (x*x*x*x*x)/5
}