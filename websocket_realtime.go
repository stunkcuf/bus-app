package main

import (
	"encoding/json"
	"log"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WebSocket upgrader
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// In production, check origin properly
		return true
	},
}

// Hub maintains active websocket connections
type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
}

// Client represents a websocket client
type Client struct {
	hub      *Hub
	conn     *websocket.Conn
	send     chan []byte
	user     *User
	clientID string
}

// Message types for websocket communication
type WSMessage struct {
	Type      string                 `json:"type"`
	Data      map[string]interface{} `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
}

var wsHub *Hub

// InitWebSocket initializes the websocket hub
func InitWebSocket() {
	wsHub = &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
	
	go wsHub.run()
	
	// Start broadcasting system updates
	go broadcastSystemUpdates()
}

// run handles websocket hub operations
func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			
			log.Printf("WebSocket client connected: %s (user: %s)", 
				client.clientID, client.user.Username)
			
			// Send welcome message
			welcome := WSMessage{
				Type: "welcome",
				Data: map[string]interface{}{
					"message": "Connected to Fleet Management System",
					"user":    client.user.Username,
				},
				Timestamp: time.Now(),
			}
			welcomeJSON, _ := json.Marshal(welcome)
			client.send <- welcomeJSON
			
		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				h.mu.Unlock()
				
				log.Printf("WebSocket client disconnected: %s", client.clientID)
			} else {
				h.mu.Unlock()
			}
			
		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					// Client's send channel is full, close it
					close(client.send)
					delete(h.clients, client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// WebSocketHandler handles websocket connections
func WebSocketHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	
	client := &Client{
		hub:      wsHub,
		conn:     conn,
		send:     make(chan []byte, 256),
		user:     user,
		clientID: generateSessionToken()[:8],
	}
	
	client.hub.register <- client
	
	// Start goroutines for reading and writing
	go client.writePump()
	go client.readPump()
}

// readPump handles incoming messages from client
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})
	
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}
		
		// Process incoming message
		c.processMessage(message)
	}
}

// writePump handles sending messages to client
func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	
	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			
			c.conn.WriteMessage(websocket.TextMessage, message)
			
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// processMessage handles incoming websocket messages
func (c *Client) processMessage(message []byte) {
	var msg WSMessage
	if err := json.Unmarshal(message, &msg); err != nil {
		log.Printf("Invalid websocket message: %v", err)
		return
	}
	
	switch msg.Type {
	case "subscribe":
		// Handle subscription to specific updates
		if channel, ok := msg.Data["channel"].(string); ok {
			c.handleSubscription(channel)
		}
		
	case "location_update":
		// Handle real-time location updates from drivers
		if c.user.Role == "driver" {
			c.handleLocationUpdate(msg.Data)
		}
		
	case "status_update":
		// Handle status updates
		if c.user.Role == "manager" || c.user.Role == "driver" {
			c.handleStatusUpdate(msg.Data)
		}
		
	case "chat":
		// Handle real-time chat messages
		c.handleChatMessage(msg.Data)
	}
}

// Broadcast functions

// BroadcastMaintenanceAlert sends maintenance alerts to all connected clients
func BroadcastMaintenanceAlert(vehicleID string, alertType string, details string) {
	if wsHub == nil {
		return
	}
	
	alert := WSMessage{
		Type: "maintenance_alert",
		Data: map[string]interface{}{
			"vehicle_id": vehicleID,
			"alert_type": alertType,
			"details":    details,
			"severity":   "high",
		},
		Timestamp: time.Now(),
	}
	
	alertJSON, _ := json.Marshal(alert)
	wsHub.broadcast <- alertJSON
}

// BroadcastRouteUpdate sends route updates to relevant clients
func BroadcastRouteUpdate(routeID string, updateType string, data map[string]interface{}) {
	if wsHub == nil {
		return
	}
	
	update := WSMessage{
		Type: "route_update",
		Data: map[string]interface{}{
			"route_id":    routeID,
			"update_type": updateType,
			"details":     data,
		},
		Timestamp: time.Now(),
	}
	
	updateJSON, _ := json.Marshal(update)
	
	// Send to specific clients based on route assignment
	wsHub.mu.RLock()
	defer wsHub.mu.RUnlock()
	
	for client := range wsHub.clients {
		// Check if client is assigned to this route
		if shouldReceiveRouteUpdate(client, routeID) {
			select {
			case client.send <- updateJSON:
			default:
				// Skip if channel is full
			}
		}
	}
}

// BroadcastSystemMetrics sends system metrics to managers
func BroadcastSystemMetrics(metrics map[string]interface{}) {
	if wsHub == nil {
		return
	}
	
	metricsMsg := WSMessage{
		Type: "system_metrics",
		Data: metrics,
		Timestamp: time.Now(),
	}
	
	metricsJSON, _ := json.Marshal(metricsMsg)
	
	// Send only to managers
	wsHub.mu.RLock()
	defer wsHub.mu.RUnlock()
	
	for client := range wsHub.clients {
		if client.user.Role == "manager" {
			select {
			case client.send <- metricsJSON:
			default:
				// Skip if channel is full
			}
		}
	}
}

// broadcastSystemUpdates sends periodic system updates
func broadcastSystemUpdates() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			// Collect current system stats
			stats := collectLiveStats()
			BroadcastSystemMetrics(stats)
		}
	}
}

// collectLiveStats gathers real-time system statistics
func collectLiveStats() map[string]interface{} {
	stats := make(map[string]interface{})
	
	// Database stats
	if db != nil {
		dbStats := db.Stats()
		stats["database"] = map[string]interface{}{
			"connections": dbStats.OpenConnections,
			"in_use":      dbStats.InUse,
			"idle":        dbStats.Idle,
		}
		
		// Active buses count
		var activeBuses int
		db.QueryRow("SELECT COUNT(*) FROM buses WHERE status = 'active'").Scan(&activeBuses)
		stats["active_buses"] = activeBuses
		
		// Routes in progress
		var activeRoutes int
		db.QueryRow(`
			SELECT COUNT(DISTINCT route_id) 
			FROM driver_logs 
			WHERE log_date = CURRENT_DATE 
			AND end_time IS NULL
		`).Scan(&activeRoutes)
		stats["active_routes"] = activeRoutes
	}
	
	// Memory stats
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	stats["memory"] = map[string]interface{}{
		"alloc_mb": m.Alloc / 1024 / 1024,
		"sys_mb":   m.Sys / 1024 / 1024,
	}
	
	// Connected clients
	wsHub.mu.RLock()
	stats["connected_clients"] = len(wsHub.clients)
	wsHub.mu.RUnlock()
	
	return stats
}

// Helper functions

func (c *Client) handleSubscription(channel string) {
	// Implement channel-specific subscriptions
	response := WSMessage{
		Type: "subscription_confirmed",
		Data: map[string]interface{}{
			"channel": channel,
			"status":  "subscribed",
		},
		Timestamp: time.Now(),
	}
	
	responseJSON, _ := json.Marshal(response)
	c.send <- responseJSON
}

func (c *Client) handleLocationUpdate(data map[string]interface{}) {
	// Store and broadcast driver location updates
	if lat, ok := data["latitude"].(float64); ok {
		if lng, ok := data["longitude"].(float64); ok {
			// Update driver location in database
			_, err := db.Exec(`
				UPDATE driver_locations 
				SET latitude = $1, longitude = $2, updated_at = CURRENT_TIMESTAMP
				WHERE driver_username = $3
			`, lat, lng, c.user.Username)
			
			if err != nil {
				log.Printf("Failed to update driver location: %v", err)
			}
			
			// Broadcast to managers
			BroadcastDriverLocation(c.user.Username, lat, lng)
		}
	}
}

func (c *Client) handleStatusUpdate(data map[string]interface{}) {
	// Handle various status updates
	if updateType, ok := data["type"].(string); ok {
		switch updateType {
		case "route_started":
			// Driver started route
			if routeID, ok := data["route_id"].(string); ok {
				BroadcastRouteUpdate(routeID, "started", map[string]interface{}{
					"driver": c.user.Username,
					"time":   time.Now(),
				})
			}
			
		case "route_completed":
			// Driver completed route
			if routeID, ok := data["route_id"].(string); ok {
				BroadcastRouteUpdate(routeID, "completed", map[string]interface{}{
					"driver": c.user.Username,
					"time":   time.Now(),
				})
			}
			
		case "emergency":
			// Emergency notification
			BroadcastEmergency(c.user.Username, data)
		}
	}
}

func (c *Client) handleChatMessage(data map[string]interface{}) {
	// Handle real-time chat between users
	if message, ok := data["message"].(string); ok {
		chatMsg := WSMessage{
			Type: "chat",
			Data: map[string]interface{}{
				"from":    c.user.Username,
				"message": message,
				"role":    c.user.Role,
			},
			Timestamp: time.Now(),
		}
		
		chatJSON, _ := json.Marshal(chatMsg)
		
		// Broadcast to relevant users
		wsHub.mu.RLock()
		defer wsHub.mu.RUnlock()
		
		for client := range wsHub.clients {
			// Send to managers and same-route drivers
			if shouldReceiveChat(client, c) {
				select {
				case client.send <- chatJSON:
				default:
				}
			}
		}
	}
}

func shouldReceiveRouteUpdate(client *Client, routeID string) bool {
	// Managers receive all updates
	if client.user.Role == "manager" {
		return true
	}
	
	// Drivers receive updates for their assigned routes
	if client.user.Role == "driver" {
		var assigned bool
		db.QueryRow(`
			SELECT EXISTS(
				SELECT 1 FROM route_assignments 
				WHERE driver = $1 AND route_id = $2
			)
		`, client.user.Username, routeID).Scan(&assigned)
		return assigned
	}
	
	return false
}

func shouldReceiveChat(recipient *Client, sender *Client) bool {
	// Managers can chat with everyone
	if recipient.user.Role == "manager" || sender.user.Role == "manager" {
		return true
	}
	
	// Drivers can chat if on same route
	var sameRoute bool
	db.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM route_assignments r1
			JOIN route_assignments r2 ON r1.route_id = r2.route_id
			WHERE r1.driver = $1 AND r2.driver = $2
		)
	`, recipient.user.Username, sender.user.Username).Scan(&sameRoute)
	
	return sameRoute
}

// Broadcast helper functions

func BroadcastDriverLocation(driver string, lat, lng float64) {
	location := WSMessage{
		Type: "driver_location",
		Data: map[string]interface{}{
			"driver":    driver,
			"latitude":  lat,
			"longitude": lng,
		},
		Timestamp: time.Now(),
	}
	
	locationJSON, _ := json.Marshal(location)
	
	// Send to managers only
	wsHub.mu.RLock()
	defer wsHub.mu.RUnlock()
	
	for client := range wsHub.clients {
		if client.user.Role == "manager" {
			select {
			case client.send <- locationJSON:
			default:
			}
		}
	}
}

func BroadcastEmergency(driver string, data map[string]interface{}) {
	emergency := WSMessage{
		Type: "emergency",
		Data: map[string]interface{}{
			"driver":  driver,
			"details": data,
		},
		Timestamp: time.Now(),
	}
	
	emergencyJSON, _ := json.Marshal(emergency)
	wsHub.broadcast <- emergencyJSON
}