package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"
)

// GPS tracking page handler
func liveTrackingHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	renderTemplate(w, r, "live_tracking.html", map[string]interface{}{
		"User":  user,
		"Title": "Live GPS Tracking",
	})
}

// SSE endpoint for GPS streaming
func gpsStreamHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Set headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Create SSE client
	client := &SSEClient{
		ID:       fmt.Sprintf("%s-%d", user.Username, time.Now().Unix()),
		Username: user.Username,
		Role:     user.Role,
		Events:   make(chan []byte, 10), // Buffered to prevent blocking
		Close:    make(chan bool),
	}

	// Register client
	sseHub.register <- client

	// Remove client on disconnect
	defer func() {
		sseHub.unregister <- client
	}()

	// Send initial connection message
	fmt.Fprintf(w, "event: connected\ndata: {\"status\":\"connected\",\"user\":\"%s\"}\n\n", user.Username)
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}

	// Listen for events
	for {
		select {
		case event := <-client.Events:
			// Send as gps_update event type for proper handling
			fmt.Fprintf(w, "event: gps_update\ndata: %s\n\n", event)
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
		case <-client.Close:
			return
		case <-r.Context().Done():
			return
		}
	}
}

// API endpoint to update GPS location (called by bus devices)
func updateGPSLocationHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var update GPSUpdate
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Validate update
	if update.VehicleID == "" {
		http.Error(w, "Vehicle ID required", http.StatusBadRequest)
		return
	}

	// Set timestamp if not provided
	if update.Timestamp.IsZero() {
		update.Timestamp = time.Now()
	}

	// Broadcast to all connected clients
	sseHub.broadcast <- update

	// Store in database (optional)
	storeGPSUpdate(update)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": "GPS location updated",
	})
}

// storeGPSUpdate is defined in gps_sse.go

// Get current bus locations
func getCurrentBusLocationsHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	locations := []GPSUpdate{}

	// Get latest location for each bus
	rows, err := db.Query(`
		SELECT DISTINCT ON (vehicle_id) 
			vehicle_id, latitude, longitude, speed, heading,
			driver_id, route_id, status, timestamp
		FROM gps_tracking
		WHERE timestamp > NOW() - INTERVAL '30 minutes'
		ORDER BY vehicle_id, timestamp DESC
	`)

	if err != nil {
		log.Printf("Error fetching bus locations: %v", err)
		// Return simulated data for demo
		locations = generateSimulatedLocations()
	} else {
		defer rows.Close()
		for rows.Next() {
			var loc GPSUpdate
			rows.Scan(&loc.VehicleID, &loc.Latitude, &loc.Longitude,
				&loc.Speed, &loc.Heading, &loc.DriverID,
				&loc.RouteID, &loc.Status, &loc.Timestamp)
			locations = append(locations, loc)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(locations)
}

// Generate simulated GPS data for demo
func generateSimulatedLocations() []GPSUpdate {
	buses := []string{"101", "102", "103", "104", "105"}
	routes := []string{"Route A", "Route B", "Route C"}
	drivers := []string{"John Smith", "Jane Doe", "Bob Johnson", "Alice Brown", "Charlie Wilson"}
	
	locations := []GPSUpdate{}
	baseTime := time.Now()
	
	for i, busID := range buses {
		status := "active"
		if i == 3 {
			status = "stopped"
		} else if i == 4 {
			status = "offline"
		}
		
		locations = append(locations, GPSUpdate{
			VehicleID: busID,
			Latitude:  40.7128 + (rand.Float64()-0.5)*0.1,
			Longitude: -74.0060 + (rand.Float64()-0.5)*0.1,
			Speed:     rand.Float64() * 30,
			Heading:   rand.Float64() * 360,
			Timestamp: baseTime.Add(-time.Duration(rand.Intn(300)) * time.Second),
			DriverID:  drivers[i],
			RouteID:   routes[i%len(routes)],
			Status:    status,
		})
	}
	
	return locations
}

// Calculate ETA for a bus
func calculateETAHandler(w http.ResponseWriter, r *http.Request) {
	busID := r.URL.Query().Get("bus_id")
	stopLat := r.URL.Query().Get("lat")
	stopLng := r.URL.Query().Get("lng")
	
	if busID == "" || stopLat == "" || stopLng == "" {
		http.Error(w, "Missing parameters", http.StatusBadRequest)
		return
	}
	
	// Get current bus location
	var busLat, busLng, speed float64
	err := db.QueryRow(`
		SELECT latitude, longitude, speed
		FROM gps_tracking
		WHERE vehicle_id = $1
		ORDER BY timestamp DESC
		LIMIT 1
	`, busID).Scan(&busLat, &busLng, &speed)
	
	if err != nil {
		// Return estimated time for demo
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"eta_minutes": 5 + rand.Intn(20),
			"distance_km": rand.Float64() * 10,
			"status":      "estimated",
		})
		return
	}
	
	// Calculate distance and ETA
	// This is a simplified calculation - in production, use a routing API
	stopLatFloat, _ := parseFloat(stopLat)
	stopLngFloat, _ := parseFloat(stopLng)
	distance := calculateDistance(busLat, busLng, stopLatFloat, stopLngFloat)
	
	// Assume average speed if current speed is 0
	if speed < 5 {
		speed = 25 // 25 mph average
	}
	
	etaMinutes := int(distance / (speed * 1.60934) * 60) // Convert mph to km/h
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"eta_minutes": etaMinutes,
		"distance_km": distance,
		"status":      "calculated",
	})
}

// calculateDistance is defined in gps_tracking.go

// parseFloat is defined in compression.go

// Start GPS simulation for demo
func startGPSSimulation() {
	log.Println("Starting GPS simulation goroutine...")
	go func() {
		buses := []string{"101", "102", "103", "104", "105"}
		routes := []string{"Route A", "Route B", "Route C"}
		drivers := []string{"John Smith", "Jane Doe", "Bob Johnson", "Alice Brown", "Charlie Wilson"}
		log.Printf("GPS Simulation: Initialized with %d buses", len(buses))
		
		// Initial positions
		busPositions := make(map[string]*GPSUpdate)
		for i, busID := range buses {
			busPositions[busID] = &GPSUpdate{
				VehicleID: busID,
				Latitude:  40.7128 + (rand.Float64()-0.5)*0.1,
				Longitude: -74.0060 + (rand.Float64()-0.5)*0.1,
				Speed:     15 + rand.Float64()*20,
				Heading:   rand.Float64() * 360,
				DriverID:  drivers[i],
				RouteID:   routes[i%len(routes)],
				Status:    "active",
			}
			
			if i == 3 {
				busPositions[busID].Status = "stopped"
				busPositions[busID].Speed = 0
			} else if i == 4 {
				busPositions[busID].Status = "offline"
			}
		}
		
		// Simulate movement
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		
		log.Println("GPS Simulation: Starting position updates every 5 seconds")
		updateCount := 0
		for range ticker.C {
			updateCount++
			log.Printf("GPS Simulation: Update cycle #%d", updateCount)
			for busID, pos := range busPositions {
				if pos.Status == "active" {
					// Update position
					pos.Latitude += (rand.Float64() - 0.5) * 0.001
					pos.Longitude += (rand.Float64() - 0.5) * 0.001
					pos.Speed = 15 + rand.Float64()*20
					pos.Heading += (rand.Float64() - 0.5) * 10
					if pos.Heading < 0 {
						pos.Heading += 360
					} else if pos.Heading > 360 {
						pos.Heading -= 360
					}
					pos.Timestamp = time.Now()
					
					// Broadcast update
					log.Printf("GPS Simulation: Broadcasting update for bus %s", busID)
					select {
					case sseHub.broadcast <- *pos:
						log.Printf("GPS Simulation: Successfully sent update for bus %s", busID)
					default:
						log.Printf("GPS Simulation: Channel full, skipped update for bus %s", busID)
					}
				}
			}
		}
	}()
}