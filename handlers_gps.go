package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
	
	"github.com/gorilla/websocket"
)

var gpsUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Allow connections from same origin
		return true
	},
}

// gpsUpdateHandler handles GPS location updates from vehicles
func gpsUpdateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendError(w, ErrMethodNotAllowed("Only POST method allowed"))
		return
	}
	
	// Parse request body
	var location GPSLocation
	if err := json.NewDecoder(r.Body).Decode(&location); err != nil {
		SendError(w, ErrBadRequest("Invalid GPS data: "+err.Error()))
		return
	}
	
	// Set timestamp if not provided
	if location.Timestamp.IsZero() {
		location.Timestamp = time.Now()
	}
	
	// Update location
	if err := gpsTracker.UpdateLocation(&location); err != nil {
		SendError(w, ErrInternal("Failed to update GPS location", err))
		return
	}
	
	// Return success response
	SendJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "GPS location updated",
		"location_id": location.ID,
	})
}

// gpsTrackingHandler shows the real-time GPS tracking page
func gpsTrackingHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	
	// Get all vehicles with latest locations
	vehicles, err := dataCache.getVehicles()
	if err != nil {
		log.Printf("Error loading vehicles: %v", err)
		vehicles = []Vehicle{}
	}
	
	// Get latest locations for all vehicles
	locations := make(map[string]*GPSLocation)
	for _, vehicle := range vehicles {
		if loc, err := gpsTracker.GetLatestLocation(vehicle.VehicleID); err == nil && loc != nil {
			locations[vehicle.VehicleID] = loc
		}
	}
	
	// Get active routes
	routes, err := dataCache.getRoutes()
	if err != nil {
		log.Printf("Error loading routes: %v", err)
		routes = []Route{}
	}
	
	data := map[string]interface{}{
		"User":      user,
		"Vehicles":  vehicles,
		"Locations": locations,
		"Routes":    routes,
		"CSRFToken": getSessionCSRFToken(r),
		"MapAPIKey": getMapAPIKey(), // You'll need to implement this
	}
	
	renderTemplate(w, r, "gps_tracking.html", data)
}

// gpsWebSocketHandler handles WebSocket connections for real-time GPS updates
func gpsWebSocketHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	
	// Get vehicle ID from query params
	vehicleID := r.URL.Query().Get("vehicle_id")
	
	// Upgrade to WebSocket
	conn, err := gpsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()
	
	// Subscribe to updates
	if vehicleID != "" {
		gpsTracker.Subscribe(vehicleID, conn)
		defer gpsTracker.Unsubscribe(vehicleID, conn)
		
		// Send current location immediately
		if loc, err := gpsTracker.GetLatestLocation(vehicleID); err == nil && loc != nil {
			conn.WriteJSON(loc)
		}
	} else {
		// Subscribe to all vehicles
		vehicles, _ := dataCache.getVehicles()
		for _, vehicle := range vehicles {
			gpsTracker.Subscribe(vehicle.VehicleID, conn)
			defer gpsTracker.Unsubscribe(vehicle.VehicleID, conn)
		}
	}
	
	// Keep connection alive
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

// gpsHistoryHandler returns GPS history for a vehicle
func gpsHistoryHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil {
		SendError(w, ErrUnauthorized("Login required"))
		return
	}
	
	vehicleID := r.URL.Query().Get("vehicle_id")
	if vehicleID == "" {
		SendError(w, ErrBadRequest("Vehicle ID required"))
		return
	}
	
	// Parse date range
	startStr := r.URL.Query().Get("start")
	endStr := r.URL.Query().Get("end")
	
	var start, end time.Time
	var err error
	
	if startStr != "" {
		start, err = time.Parse("2006-01-02", startStr)
		if err != nil {
			SendError(w, ErrBadRequest("Invalid start date: "+err.Error()))
			return
		}
	} else {
		start = time.Now().AddDate(0, 0, -1) // Default: last 24 hours
	}
	
	if endStr != "" {
		end, err = time.Parse("2006-01-02", endStr)
		if err != nil {
			SendError(w, ErrBadRequest("Invalid end date: "+err.Error()))
			return
		}
		end = end.Add(24 * time.Hour) // Include entire end day
	} else {
		end = time.Now()
	}
	
	// Get history
	locations, err := gpsTracker.GetLocationHistory(vehicleID, start, end)
	if err != nil {
		SendError(w, ErrDatabase("Failed to load GPS history", err))
		return
	}
	
	// Calculate route statistics
	stats := calculateRouteStats(locations)
	
	SendJSON(w, http.StatusOK, map[string]interface{}{
		"success":   true,
		"vehicle_id": vehicleID,
		"start":     start,
		"end":       end,
		"locations": locations,
		"stats":     stats,
	})
}

// geofenceHandler manages geofences
func geofenceHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		SendError(w, ErrUnauthorized("Manager access required"))
		return
	}
	
	switch r.Method {
	case http.MethodGet:
		// List all geofences
		query := `
			SELECT id, name, type, center_latitude, center_longitude, 
			       radius_meters, metadata, active, created_at
			FROM geofences
			ORDER BY name
		`
		
		var geofences []struct {
			ID              int             `json:"id" db:"id"`
			Name            string          `json:"name" db:"name"`
			Type            string          `json:"type" db:"type"`
			CenterLatitude  float64         `json:"center_latitude" db:"center_latitude"`
			CenterLongitude float64         `json:"center_longitude" db:"center_longitude"`
			RadiusMeters    float64         `json:"radius_meters" db:"radius_meters"`
			Metadata        json.RawMessage `json:"metadata" db:"metadata"`
			Active          bool            `json:"active" db:"active"`
			CreatedAt       time.Time       `json:"created_at" db:"created_at"`
		}
		
		if err := db.Select(&geofences, query); err != nil {
			SendError(w, ErrDatabase("Failed to load geofences", err))
			return
		}
		
		SendJSON(w, http.StatusOK, map[string]interface{}{
			"success":   true,
			"geofences": geofences,
		})
		
	case http.MethodPost:
		// Create new geofence
		var req struct {
			Name            string          `json:"name"`
			Type            string          `json:"type"`
			CenterLatitude  float64         `json:"center_latitude"`
			CenterLongitude float64         `json:"center_longitude"`
			RadiusMeters    float64         `json:"radius_meters"`
			Metadata        json.RawMessage `json:"metadata"`
		}
		
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			SendError(w, ErrBadRequest("Invalid request data: "+err.Error()))
			return
		}
		
		// Validate
		if req.Name == "" || req.Type == "" {
			SendError(w, ErrBadRequest("Name and type are required"))
			return
		}
		
		if req.RadiusMeters <= 0 {
			SendError(w, ErrBadRequest("Radius must be positive"))
			return
		}
		
		// Insert geofence
		var id int
		query := `
			INSERT INTO geofences (name, type, center_latitude, center_longitude, radius_meters, metadata)
			VALUES ($1, $2, $3, $4, $5, $6)
			RETURNING id
		`
		
		err := db.QueryRow(
			query, req.Name, req.Type, req.CenterLatitude,
			req.CenterLongitude, req.RadiusMeters, req.Metadata,
		).Scan(&id)
		
		if err != nil {
			SendError(w, ErrDatabase("Failed to create geofence", err))
			return
		}
		
		SendJSON(w, http.StatusCreated, map[string]interface{}{
			"success": true,
			"message": "Geofence created successfully",
			"id":      id,
		})
		
	case http.MethodDelete:
		// Delete geofence
		idStr := r.URL.Query().Get("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			SendError(w, ErrBadRequest("Invalid geofence ID: "+err.Error()))
			return
		}
		
		_, err = db.Exec("DELETE FROM geofences WHERE id = $1", id)
		if err != nil {
			SendError(w, ErrDatabase("Failed to delete geofence", err))
			return
		}
		
		SendJSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"message": "Geofence deleted successfully",
		})
		
	default:
		SendError(w, ErrMethodNotAllowed("Method not allowed"))
	}
}

// calculateRouteStats calculates statistics from GPS locations
func calculateRouteStats(locations []GPSLocation) map[string]interface{} {
	if len(locations) == 0 {
		return map[string]interface{}{
			"total_distance": 0,
			"average_speed":  0,
			"max_speed":      0,
			"duration":       0,
			"stops":          0,
		}
	}
	
	var totalDistance float64
	var totalSpeed float64
	var maxSpeed float64
	stops := 0
	
	for i := 1; i < len(locations); i++ {
		prev := locations[i-1]
		curr := locations[i]
		
		// Calculate distance
		distance := calculateDistance(
			prev.Latitude, prev.Longitude,
			curr.Latitude, curr.Longitude,
		)
		totalDistance += distance
		
		// Track speeds
		totalSpeed += curr.Speed
		if curr.Speed > maxSpeed {
			maxSpeed = curr.Speed
		}
		
		// Count stops (speed < 5 km/h for more than 1 minute)
		if curr.Speed < 5 && i > 0 {
			timeDiff := curr.Timestamp.Sub(prev.Timestamp)
			if timeDiff > time.Minute {
				stops++
			}
		}
	}
	
	duration := locations[len(locations)-1].Timestamp.Sub(locations[0].Timestamp)
	averageSpeed := totalSpeed / float64(len(locations))
	
	return map[string]interface{}{
		"total_distance": fmt.Sprintf("%.2f", totalDistance/1000), // Convert to km
		"average_speed":  fmt.Sprintf("%.1f", averageSpeed),
		"max_speed":      fmt.Sprintf("%.1f", maxSpeed),
		"duration":       duration.String(),
		"stops":          stops,
	}
}

// getMapAPIKey returns the Google Maps API key from environment
func getMapAPIKey() string {
	// You should store this in environment variables
	return os.Getenv("GOOGLE_MAPS_API_KEY")
}

// gpsVehiclesHandler returns current locations for all vehicles
func gpsVehiclesHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil {
		SendError(w, ErrUnauthorized("Login required"))
		return
	}
	
	// Get all vehicles with latest locations
	vehicles, err := dataCache.getVehicles()
	if err != nil {
		SendError(w, ErrDatabase("Failed to load vehicles", err))
		return
	}
	
	// Get latest locations for all vehicles
	locations := make([]GPSLocation, 0)
	for _, vehicle := range vehicles {
		if loc, err := gpsTracker.GetLatestLocation(vehicle.VehicleID); err == nil && loc != nil {
			locations = append(locations, *loc)
		}
	}
	
	SendJSON(w, http.StatusOK, map[string]interface{}{
		"success":   true,
		"locations": locations,
	})
}