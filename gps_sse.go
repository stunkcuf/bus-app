package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// GPSUpdate represents a GPS location update
type GPSUpdate struct {
	VehicleID string    `json:"vehicle_id"`
	Latitude  float64   `json:"latitude"`
	Longitude float64   `json:"longitude"`
	Speed     float64   `json:"speed"`
	Heading   float64   `json:"heading"`
	Timestamp time.Time `json:"timestamp"`
	DriverID  string    `json:"driver_id"`
	RouteID   string    `json:"route_id"`
	Status    string    `json:"status"` // "active", "stopped", "offline"
}

// SSEClient represents a Server-Sent Events client
type SSEClient struct {
	ID       string
	Username string
	Role     string
	Events   chan []byte
	Close    chan bool
}

// SSEHub manages all SSE connections
type SSEHub struct {
	clients    map[string]*SSEClient
	register   chan *SSEClient
	unregister chan *SSEClient
	broadcast  chan GPSUpdate
}

var sseHub = &SSEHub{
	clients:    make(map[string]*SSEClient),
	register:   make(chan *SSEClient),
	unregister: make(chan *SSEClient),
	broadcast:  make(chan GPSUpdate, 100), // Buffered channel to prevent blocking
}

// Run starts the SSE hub
func (h *SSEHub) Run() {
	log.Println("SSE Hub: Started and running")
	for {
		select {
		case client := <-h.register:
			h.clients[client.ID] = client
			log.Printf("SSE client registered: %s (user: %s)", client.ID, client.Username)

		case client := <-h.unregister:
			if _, ok := h.clients[client.ID]; ok {
				delete(h.clients, client.ID)
				close(client.Events)
				log.Printf("SSE client unregistered: %s", client.ID)
			}

		case update := <-h.broadcast:
			log.Printf("SSE Hub: Received broadcast for vehicle %s", update.VehicleID)
			data, err := json.Marshal(update)
			if err != nil {
				log.Printf("Error marshaling GPS update: %v", err)
				continue
			}

			log.Printf("SSE Hub: Broadcasting to %d clients", len(h.clients))
			// Send to all connected clients
			for clientID, client := range h.clients {
				// Only send GPS updates to managers or the driver of the vehicle
				if client.Role == "manager" || client.Username == update.DriverID {
					log.Printf("SSE Hub: Sending to client %s", clientID)
					select {
					case client.Events <- data:
						log.Printf("SSE Hub: Sent to client %s", clientID)
					default:
						log.Printf("SSE Hub: Client %s channel full, removing", clientID)
						// Client's channel is full, close it
						close(client.Events)
						delete(h.clients, clientID)
					}
				}
			}
		}
	}
}

// gpsSSEHandler handles Server-Sent Events for GPS tracking
func gpsSSEHandler(w http.ResponseWriter, r *http.Request) {
	// Check authentication
	session, err := GetSession(r)
	if err != nil || session == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	username := session.Username
	role := session.Role
	if username == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Check if GPS is enabled (for managers)
	if role == "manager" {
		enabled, err := isGPSEnabled()
		if err != nil {
			log.Printf("Error checking GPS status: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		if !enabled {
			http.Error(w, "GPS tracking is disabled", http.StatusServiceUnavailable)
			return
		}
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Create client
	client := &SSEClient{
		ID:       generateSessionToken()[:8],
		Username: username,
		Role:     role,
		Events:   make(chan []byte),
		Close:    make(chan bool),
	}

	// Register client
	sseHub.register <- client

	// Clean up on disconnect
	defer func() {
		sseHub.unregister <- client
	}()

	// Send initial connection message
	fmt.Fprintf(w, "event: connected\ndata: {\"message\":\"GPS tracking connected\",\"client_id\":\"%s\"}\n\n", client.ID)
	w.(http.Flusher).Flush()

	// Send heartbeat and GPS updates
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			// Client disconnected
			return

		case data := <-client.Events:
			// Send GPS update
			fmt.Fprintf(w, "event: gps_update\ndata: %s\n\n", data)
			w.(http.Flusher).Flush()

		case <-ticker.C:
			// Send heartbeat to keep connection alive
			fmt.Fprintf(w, "event: heartbeat\ndata: {\"timestamp\":\"%s\"}\n\n", time.Now().Format(time.RFC3339))
			w.(http.Flusher).Flush()

		case <-client.Close:
			// Server is closing this connection
			return
		}
	}
}

// gpsUpdateSSEHandler receives GPS updates from vehicles and broadcasts them
func gpsUpdateSSEHandler(w http.ResponseWriter, r *http.Request) {
	// This would typically be called by the GPS device/app
	session, err := GetSession(r)
	if err != nil || session == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	username := session.Username
	if username == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse GPS update
	var update GPSUpdate
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Set timestamp
	update.Timestamp = time.Now()
	update.DriverID = username

	// Store in database (optional)
	if err := storeGPSUpdate(update); err != nil {
		log.Printf("Error storing GPS update: %v", err)
	}

	// Broadcast to connected clients
	sseHub.broadcast <- update

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// isGPSEnabled checks if GPS tracking is enabled
func isGPSEnabled() (bool, error) {
	var value string
	err := db.Get(&value, "SELECT value FROM system_settings WHERE key = 'gps_enabled'")
	if err != nil {
		// If setting doesn't exist, default to false
		if err.Error() == "sql: no rows in result set" {
			return false, nil
		}
		return false, err
	}
	// Convert string value to boolean
	return value == "true", nil
}

// toggleGPSHandler enables/disables GPS tracking
func toggleGPSHandler(w http.ResponseWriter, r *http.Request) {
	session, err := GetSession(r)
	if err != nil || session == nil || session.Role != "manager" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse request
	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Update setting in database
	_, err = db.Exec(`
		INSERT INTO system_settings (key, value, updated_at, updated_by)
		VALUES ('gps_enabled', $1, NOW(), $2)
		ON CONFLICT (key) DO UPDATE
		SET value = $1, updated_at = NOW(), updated_by = $2
	`, fmt.Sprintf("%t", req.Enabled), session.Username)

	if err != nil {
		log.Printf("Error updating GPS setting: %v", err)
		http.Error(w, "Failed to update setting", http.StatusInternalServerError)
		return
	}

	// Log the change
	log.Printf("GPS tracking %s by %s", map[bool]string{true: "enabled", false: "disabled"}[req.Enabled], session.Username)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"enabled": req.Enabled,
	})
}

// storeGPSUpdate stores GPS update in database
func storeGPSUpdate(update GPSUpdate) error {
	_, err := db.Exec(`
		INSERT INTO gps_tracking (
			vehicle_id, latitude, longitude, speed, heading,
			timestamp, driver_id, route_id, status
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, update.VehicleID, update.Latitude, update.Longitude, update.Speed,
		update.Heading, update.Timestamp, update.DriverID, update.RouteID, update.Status)
	return err
}

// InitSSE initializes the SSE hub
func InitSSE() {
	log.Println("Initializing Server-Sent Events for GPS tracking...")
	go sseHub.Run()
}