package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"time"
)

// EmergencyAlert represents an emergency alert
type EmergencyAlert struct {
	ID             int                    `json:"id"`
	AlertID        string                 `json:"alert_id"`
	Type           string                 `json:"type"` // breakdown, accident, medical, security, weather, other
	Severity       string                 `json:"severity"` // critical, high, medium, low
	Status         string                 `json:"status"` // active, acknowledged, resolved, cancelled
	Location       LocationData           `json:"location"`
	Title          string                 `json:"title"`
	Description    string                 `json:"description"`
	ReportedBy     int                    `json:"reported_by"`
	ReporterName   string                 `json:"reporter_name"`
	VehicleID      string                 `json:"vehicle_id,omitempty"`
	RouteID        string                 `json:"route_id,omitempty"`
	StudentIDs     []string               `json:"student_ids,omitempty"`
	Attachments    []EmergencyAttachment  `json:"attachments,omitempty"`
	Responders     []EmergencyResponder   `json:"responders"`
	Timeline       []EmergencyEvent       `json:"timeline"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
	ResolvedAt     *time.Time             `json:"resolved_at,omitempty"`
	Metadata       map[string]interface{} `json:"metadata"`
}

// LocationData represents location information
type LocationData struct {
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
	Address     string  `json:"address,omitempty"`
	Description string  `json:"description,omitempty"`
}

// EmergencyResponder represents a person responding to an emergency
type EmergencyResponder struct {
	ID           int       `json:"id"`
	UserID       int       `json:"user_id"`
	Name         string    `json:"name"`
	Role         string    `json:"role"`
	Status       string    `json:"status"` // assigned, en_route, on_scene, completed
	AssignedAt   time.Time `json:"assigned_at"`
	ArrivedAt    *time.Time `json:"arrived_at,omitempty"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
}

// EmergencyEvent represents a timeline event
type EmergencyEvent struct {
	ID          int       `json:"id"`
	Type        string    `json:"type"` // created, updated, status_change, responder_assigned, etc.
	Description string    `json:"description"`
	UserID      int       `json:"user_id"`
	UserName    string    `json:"user_name"`
	Timestamp   time.Time `json:"timestamp"`
}

// EmergencyAttachment represents a file attachment
type EmergencyAttachment struct {
	ID         int       `json:"id"`
	Type       string    `json:"type"` // photo, document, audio
	URL        string    `json:"url"`
	Filename   string    `json:"filename"`
	Size       int64     `json:"size"`
	UploadedBy int       `json:"uploaded_by"`
	UploadedAt time.Time `json:"uploaded_at"`
}

// EmergencyProtocol represents predefined emergency procedures
type EmergencyProtocol struct {
	ID        int                    `json:"id"`
	Type      string                 `json:"type"`
	Name      string                 `json:"name"`
	Steps     []ProtocolStep         `json:"steps"`
	Contacts  []EmergencyContact     `json:"contacts"`
	Resources []string               `json:"resources"`
	IsActive  bool                   `json:"is_active"`
}

// ProtocolStep represents a step in an emergency protocol
type ProtocolStep struct {
	Order       int    `json:"order"`
	Action      string `json:"action"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
}

// EmergencyContact represents an emergency contact
type EmergencyContact struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Role     string `json:"role"`
	Phone    string `json:"phone"`
	Email    string `json:"email"`
	Priority int    `json:"priority"`
}

// emergencyDashboardHandler serves the emergency management dashboard
func emergencyDashboardHandler(w http.ResponseWriter, r *http.Request) {
	session := getUserFromSession(r)
	if session == nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// Get active emergencies
	activeEmergencies, err := getActiveEmergencies()
	if err != nil {
		log.Printf("Failed to get active emergencies: %v", err)
	}

	// Get emergency history
	emergencyHistory, err := getEmergencyHistory(30)
	if err != nil {
		log.Printf("Failed to get emergency history: %v", err)
	}

	// Get protocols
	protocols, err := getEmergencyProtocols()
	if err != nil {
		log.Printf("Failed to get emergency protocols: %v", err)
	}

	data := struct {
		Title             string
		Username          string
		UserType          string
		CSPNonce          string
		ActiveEmergencies []EmergencyAlert
		EmergencyHistory  []EmergencyAlert
		Protocols         []EmergencyProtocol
		Stats             EmergencyStats
	}{
		Title:             "Emergency Management",
		Username:          session.Username,
		UserType:          session.Role,
		CSPNonce:          getCSPNonce(r.Context()),
		ActiveEmergencies: activeEmergencies,
		EmergencyHistory:  emergencyHistory,
		Protocols:         protocols,
		Stats:             getEmergencyStats(),
	}

	tmpl := template.Must(template.ParseFiles("templates/emergency_dashboard.html"))
	err = tmpl.Execute(w, data)
	if err != nil {
		log.Printf("Error rendering emergency dashboard: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// createEmergencyHandler handles creating new emergency alerts
func createEmergencyHandler(w http.ResponseWriter, r *http.Request) {
	session := getUserFromSession(r)
	if session == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		Type        string       `json:"type"`
		Severity    string       `json:"severity"`
		Title       string       `json:"title"`
		Description string       `json:"description"`
		Location    LocationData `json:"location"`
		VehicleID   string       `json:"vehicle_id,omitempty"`
		RouteID     string       `json:"route_id,omitempty"`
		StudentIDs  []string     `json:"student_ids,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Create emergency alert
	alert := &EmergencyAlert{
		AlertID:      generateEmergencyID(),
		Type:         req.Type,
		Severity:     req.Severity,
		Status:       "active",
		Title:        req.Title,
		Description:  req.Description,
		Location:     req.Location,
		ReportedBy:   getUserID(session.Username),
		ReporterName: session.Username,
		VehicleID:    req.VehicleID,
		RouteID:      req.RouteID,
		StudentIDs:   req.StudentIDs,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Save to database
	if err := saveEmergency(alert); err != nil {
		log.Printf("Failed to save emergency: %v", err)
		http.Error(w, "Failed to create emergency", http.StatusInternalServerError)
		return
	}

	// Add to timeline
	addEmergencyEvent(alert.AlertID, "created", fmt.Sprintf("Emergency alert created by %s", session.Username), getUserID(session.Username), session.Username)

	// Send notifications
	go sendEmergencyNotifications(alert)

	// Broadcast via WebSocket
	broadcastEmergencyUpdate(alert)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(alert)
}

// updateEmergencyHandler handles updating emergency status
func updateEmergencyHandler(w http.ResponseWriter, r *http.Request) {
	session := getUserFromSession(r)
	if session == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		AlertID     string `json:"alert_id"`
		Status      string `json:"status"`
		Description string `json:"description,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Update emergency status
	err := updateEmergencyStatus(req.AlertID, req.Status, getUserID(session.Username), session.Username, req.Description)
	if err != nil {
		log.Printf("Failed to update emergency: %v", err)
		http.Error(w, "Failed to update emergency", http.StatusInternalServerError)
		return
	}

	// Get updated alert
	alert, err := getEmergencyByID(req.AlertID)
	if err != nil {
		log.Printf("Failed to get emergency: %v", err)
		http.Error(w, "Emergency not found", http.StatusNotFound)
		return
	}

	// Broadcast update
	broadcastEmergencyUpdate(alert)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"alert":   alert,
	})
}

// emergencySOSHandler handles immediate SOS alerts from drivers
func emergencySOSHandler(w http.ResponseWriter, r *http.Request) {
	session := getUserFromSession(r)
	if session == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		Location  LocationData `json:"location"`
		VehicleID string       `json:"vehicle_id"`
		RouteID   string       `json:"route_id,omitempty"`
		Message   string       `json:"message,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Create critical SOS alert
	alert := &EmergencyAlert{
		AlertID:      generateEmergencyID(),
		Type:         "sos",
		Severity:     "critical",
		Status:       "active",
		Title:        "SOS - Driver Emergency",
		Description:  fmt.Sprintf("SOS activated by %s. %s", session.Username, req.Message),
		Location:     req.Location,
		ReportedBy:   getUserID(session.Username),
		ReporterName: session.Username,
		VehicleID:    req.VehicleID,
		RouteID:      req.RouteID,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		Metadata: map[string]interface{}{
			"sos": true,
		},
	}

	// Save to database
	if err := saveEmergency(alert); err != nil {
		log.Printf("Failed to save SOS: %v", err)
		http.Error(w, "Failed to create SOS alert", http.StatusInternalServerError)
		return
	}

	// Immediately notify all managers and emergency contacts
	go sendSOSNotifications(alert)

	// Broadcast critical alert
	broadcastSOSAlert(alert)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"alert":   alert,
		"message": "SOS alert sent successfully",
	})
}

// getEmergencyAlertsHandler returns list of emergency alerts
func getEmergencyAlertsHandler(w http.ResponseWriter, r *http.Request) {
	session := getUserFromSession(r)
	if session == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	status := r.URL.Query().Get("status")
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	alerts, err := getEmergencyAlerts(status, limit)
	if err != nil {
		log.Printf("Failed to get emergency alerts: %v", err)
		http.Error(w, "Failed to get alerts", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(alerts)
}

// assignResponderHandler assigns a responder to an emergency
func assignResponderHandler(w http.ResponseWriter, r *http.Request) {
	session := getUserFromSession(r)
	if session == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		AlertID string `json:"alert_id"`
		UserID  int    `json:"user_id"`
		Role    string `json:"role"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Assign responder
	responder := &EmergencyResponder{
		UserID:     req.UserID,
		Role:       req.Role,
		Status:     "assigned",
		AssignedAt: time.Now(),
	}

	err := assignResponder(req.AlertID, responder)
	if err != nil {
		log.Printf("Failed to assign responder: %v", err)
		http.Error(w, "Failed to assign responder", http.StatusInternalServerError)
		return
	}

	// Add to timeline
	addEmergencyEvent(req.AlertID, "responder_assigned", fmt.Sprintf("Responder assigned: %s", responder.Name), getUserID(session.Username), session.Username)

	// Notify responder
	go notifyResponder(req.AlertID, responder)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":   true,
		"responder": responder,
	})
}

// Database functions

func getActiveEmergencies() ([]EmergencyAlert, error) {
	query := `
		SELECT id, alert_id, type, severity, status, location, title, description,
		       reported_by, vehicle_id, route_id, created_at, updated_at
		FROM emergency_alerts
		WHERE status IN ('active', 'acknowledged')
		ORDER BY 
			CASE severity 
				WHEN 'critical' THEN 1 
				WHEN 'high' THEN 2 
				WHEN 'medium' THEN 3 
				WHEN 'low' THEN 4 
			END,
			created_at DESC
	`
	
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var alerts []EmergencyAlert
	for rows.Next() {
		var alert EmergencyAlert
		var locationJSON []byte
		var vehicleID, routeID sql.NullString
		
		err := rows.Scan(&alert.ID, &alert.AlertID, &alert.Type, &alert.Severity,
			&alert.Status, &locationJSON, &alert.Title, &alert.Description,
			&alert.ReportedBy, &vehicleID, &routeID, &alert.CreatedAt, &alert.UpdatedAt)
		
		if err != nil {
			continue
		}
		
		json.Unmarshal(locationJSON, &alert.Location)
		
		if vehicleID.Valid {
			alert.VehicleID = vehicleID.String
		}
		if routeID.Valid {
			alert.RouteID = routeID.String
		}
		
		// Get reporter name
		alert.ReporterName = getUsernameByID(alert.ReportedBy)
		
		// Get responders
		alert.Responders = getEmergencyResponders(alert.AlertID)
		
		// Get timeline
		alert.Timeline = getEmergencyTimeline(alert.AlertID)
		
		alerts = append(alerts, alert)
	}

	return alerts, nil
}

func getEmergencyHistory(days int) ([]EmergencyAlert, error) {
	query := `
		SELECT id, alert_id, type, severity, status, location, title, description,
		       reported_by, vehicle_id, route_id, created_at, updated_at, resolved_at
		FROM emergency_alerts
		WHERE created_at > CURRENT_TIMESTAMP - INTERVAL '%d days'
		ORDER BY created_at DESC
		LIMIT 100
	`
	
	rows, err := db.Query(fmt.Sprintf(query, days))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var alerts []EmergencyAlert
	for rows.Next() {
		var alert EmergencyAlert
		var locationJSON []byte
		var vehicleID, routeID sql.NullString
		var resolvedAt sql.NullTime
		
		err := rows.Scan(&alert.ID, &alert.AlertID, &alert.Type, &alert.Severity,
			&alert.Status, &locationJSON, &alert.Title, &alert.Description,
			&alert.ReportedBy, &vehicleID, &routeID, &alert.CreatedAt, 
			&alert.UpdatedAt, &resolvedAt)
		
		if err != nil {
			continue
		}
		
		json.Unmarshal(locationJSON, &alert.Location)
		
		if vehicleID.Valid {
			alert.VehicleID = vehicleID.String
		}
		if routeID.Valid {
			alert.RouteID = routeID.String
		}
		if resolvedAt.Valid {
			alert.ResolvedAt = &resolvedAt.Time
		}
		
		alerts = append(alerts, alert)
	}

	return alerts, nil
}

func getEmergencyProtocols() ([]EmergencyProtocol, error) {
	query := `
		SELECT id, type, name, steps, contacts, resources, is_active
		FROM emergency_protocols
		WHERE is_active = true
		ORDER BY type, name
	`
	
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var protocols []EmergencyProtocol
	for rows.Next() {
		var protocol EmergencyProtocol
		var stepsJSON, contactsJSON, resourcesJSON []byte
		
		err := rows.Scan(&protocol.ID, &protocol.Type, &protocol.Name,
			&stepsJSON, &contactsJSON, &resourcesJSON, &protocol.IsActive)
		
		if err != nil {
			continue
		}
		
		json.Unmarshal(stepsJSON, &protocol.Steps)
		json.Unmarshal(contactsJSON, &protocol.Contacts)
		json.Unmarshal(resourcesJSON, &protocol.Resources)
		
		protocols = append(protocols, protocol)
	}

	return protocols, nil
}

func saveEmergency(alert *EmergencyAlert) error {
	locationJSON, _ := json.Marshal(alert.Location)
	metadataJSON, _ := json.Marshal(alert.Metadata)
	
	query := `
		INSERT INTO emergency_alerts 
		(alert_id, type, severity, status, location, title, description,
		 reported_by, vehicle_id, route_id, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id
	`
	
	err := db.QueryRow(query,
		alert.AlertID, alert.Type, alert.Severity, alert.Status, locationJSON,
		alert.Title, alert.Description, alert.ReportedBy, alert.VehicleID,
		alert.RouteID, metadataJSON, alert.CreatedAt, alert.UpdatedAt,
	).Scan(&alert.ID)
	
	return err
}

func updateEmergencyStatus(alertID, status string, userID int, username, description string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Update status
	query := `
		UPDATE emergency_alerts 
		SET status = $1, updated_at = CURRENT_TIMESTAMP
		WHERE alert_id = $2
	`
	
	_, err = tx.Exec(query, status, alertID)
	if err != nil {
		return err
	}
	
	// Set resolved_at if resolved
	if status == "resolved" || status == "cancelled" {
		_, err = tx.Exec(`
			UPDATE emergency_alerts 
			SET resolved_at = CURRENT_TIMESTAMP 
			WHERE alert_id = $1
		`, alertID)
		if err != nil {
			return err
		}
	}
	
	// Add to timeline
	eventDesc := fmt.Sprintf("Status changed to %s", status)
	if description != "" {
		eventDesc += ": " + description
	}
	
	_, err = tx.Exec(`
		INSERT INTO emergency_timeline 
		(alert_id, type, description, user_id, timestamp)
		VALUES ($1, $2, $3, $4, CURRENT_TIMESTAMP)
	`, alertID, "status_change", eventDesc, userID)
	
	if err != nil {
		return err
	}
	
	return tx.Commit()
}

func getEmergencyByID(alertID string) (*EmergencyAlert, error) {
	var alert EmergencyAlert
	var locationJSON []byte
	var vehicleID, routeID sql.NullString
	var resolvedAt sql.NullTime
	
	query := `
		SELECT id, alert_id, type, severity, status, location, title, description,
		       reported_by, vehicle_id, route_id, created_at, updated_at, resolved_at
		FROM emergency_alerts
		WHERE alert_id = $1
	`
	
	err := db.QueryRow(query, alertID).Scan(
		&alert.ID, &alert.AlertID, &alert.Type, &alert.Severity,
		&alert.Status, &locationJSON, &alert.Title, &alert.Description,
		&alert.ReportedBy, &vehicleID, &routeID, &alert.CreatedAt,
		&alert.UpdatedAt, &resolvedAt,
	)
	
	if err != nil {
		return nil, err
	}
	
	json.Unmarshal(locationJSON, &alert.Location)
	
	if vehicleID.Valid {
		alert.VehicleID = vehicleID.String
	}
	if routeID.Valid {
		alert.RouteID = routeID.String
	}
	if resolvedAt.Valid {
		alert.ResolvedAt = &resolvedAt.Time
	}
	
	// Get reporter name
	alert.ReporterName = getUsernameByID(alert.ReportedBy)
	
	// Get responders
	alert.Responders = getEmergencyResponders(alert.AlertID)
	
	// Get timeline
	alert.Timeline = getEmergencyTimeline(alert.AlertID)
	
	return &alert, nil
}

func getEmergencyResponders(alertID string) []EmergencyResponder {
	var responders []EmergencyResponder
	
	query := `
		SELECT er.id, er.user_id, u.username, er.role, er.status,
		       er.assigned_at, er.arrived_at, er.completed_at
		FROM emergency_responders er
		JOIN users u ON er.user_id = u.id
		WHERE er.alert_id = $1
		ORDER BY er.assigned_at
	`
	
	rows, err := db.Query(query, alertID)
	if err != nil {
		return responders
	}
	defer rows.Close()

	for rows.Next() {
		var responder EmergencyResponder
		var arrivedAt, completedAt sql.NullTime
		
		err := rows.Scan(&responder.ID, &responder.UserID, &responder.Name,
			&responder.Role, &responder.Status, &responder.AssignedAt,
			&arrivedAt, &completedAt)
		
		if err != nil {
			continue
		}
		
		if arrivedAt.Valid {
			responder.ArrivedAt = &arrivedAt.Time
		}
		if completedAt.Valid {
			responder.CompletedAt = &completedAt.Time
		}
		
		responders = append(responders, responder)
	}

	return responders
}

func getEmergencyTimeline(alertID string) []EmergencyEvent {
	var events []EmergencyEvent
	
	query := `
		SELECT et.id, et.type, et.description, et.user_id, u.username, et.timestamp
		FROM emergency_timeline et
		LEFT JOIN users u ON et.user_id = u.id
		WHERE et.alert_id = $1
		ORDER BY et.timestamp DESC
	`
	
	rows, err := db.Query(query, alertID)
	if err != nil {
		return events
	}
	defer rows.Close()

	for rows.Next() {
		var event EmergencyEvent
		var userID sql.NullInt64
		var userName sql.NullString
		
		err := rows.Scan(&event.ID, &event.Type, &event.Description,
			&userID, &userName, &event.Timestamp)
		
		if err != nil {
			continue
		}
		
		if userID.Valid {
			event.UserID = int(userID.Int64)
		}
		if userName.Valid {
			event.UserName = userName.String
		}
		
		events = append(events, event)
	}

	return events
}

func addEmergencyEvent(alertID, eventType, description string, userID int, username string) error {
	_, err := db.Exec(`
		INSERT INTO emergency_timeline 
		(alert_id, type, description, user_id, timestamp)
		VALUES ($1, $2, $3, $4, CURRENT_TIMESTAMP)
	`, alertID, eventType, description, userID)
	
	return err
}

func assignResponder(alertID string, responder *EmergencyResponder) error {
	// Get user name
	var name string
	err := db.QueryRow("SELECT username FROM users WHERE id = $1", responder.UserID).Scan(&name)
	if err != nil {
		return err
	}
	responder.Name = name
	
	_, err = db.Exec(`
		INSERT INTO emergency_responders 
		(alert_id, user_id, role, status, assigned_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (alert_id, user_id) DO UPDATE SET
			role = EXCLUDED.role,
			status = EXCLUDED.status
		RETURNING id
	`, alertID, responder.UserID, responder.Role, responder.Status, responder.AssignedAt)
	
	return err
}

func getEmergencyAlerts(status string, limit int) ([]EmergencyAlert, error) {
	query := `
		SELECT id, alert_id, type, severity, status, location, title, description,
		       reported_by, vehicle_id, route_id, created_at, updated_at
		FROM emergency_alerts
	`
	
	if status != "" {
		query += fmt.Sprintf(" WHERE status = '%s'", status)
	}
	
	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT %d", limit)
	
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var alerts []EmergencyAlert
	for rows.Next() {
		var alert EmergencyAlert
		var locationJSON []byte
		var vehicleID, routeID sql.NullString
		
		err := rows.Scan(&alert.ID, &alert.AlertID, &alert.Type, &alert.Severity,
			&alert.Status, &locationJSON, &alert.Title, &alert.Description,
			&alert.ReportedBy, &vehicleID, &routeID, &alert.CreatedAt, &alert.UpdatedAt)
		
		if err != nil {
			continue
		}
		
		json.Unmarshal(locationJSON, &alert.Location)
		
		if vehicleID.Valid {
			alert.VehicleID = vehicleID.String
		}
		if routeID.Valid {
			alert.RouteID = routeID.String
		}
		
		alerts = append(alerts, alert)
	}

	return alerts, nil
}

// Helper functions

func generateEmergencyID() string {
	return fmt.Sprintf("EMRG-%d-%s", time.Now().Unix(), generateRandomString(6))
}

type EmergencyStats struct {
	ActiveCount    int            `json:"active_count"`
	TodayCount     int            `json:"today_count"`
	WeekCount      int            `json:"week_count"`
	ByType         map[string]int `json:"by_type"`
	BySeverity     map[string]int `json:"by_severity"`
	AverageResponse string        `json:"average_response"`
}

func getEmergencyStats() EmergencyStats {
	var stats EmergencyStats
	stats.ByType = make(map[string]int)
	stats.BySeverity = make(map[string]int)
	
	// Active count
	db.QueryRow("SELECT COUNT(*) FROM emergency_alerts WHERE status IN ('active', 'acknowledged')").Scan(&stats.ActiveCount)
	
	// Today count
	db.QueryRow("SELECT COUNT(*) FROM emergency_alerts WHERE DATE(created_at) = CURRENT_DATE").Scan(&stats.TodayCount)
	
	// Week count
	db.QueryRow("SELECT COUNT(*) FROM emergency_alerts WHERE created_at > CURRENT_TIMESTAMP - INTERVAL '7 days'").Scan(&stats.WeekCount)
	
	// By type
	rows, _ := db.Query("SELECT type, COUNT(*) FROM emergency_alerts WHERE created_at > CURRENT_TIMESTAMP - INTERVAL '30 days' GROUP BY type")
	defer rows.Close()
	for rows.Next() {
		var t string
		var count int
		rows.Scan(&t, &count)
		stats.ByType[t] = count
	}
	
	// By severity
	rows2, _ := db.Query("SELECT severity, COUNT(*) FROM emergency_alerts WHERE created_at > CURRENT_TIMESTAMP - INTERVAL '30 days' GROUP BY severity")
	defer rows2.Close()
	for rows2.Next() {
		var s string
		var count int
		rows2.Scan(&s, &count)
		stats.BySeverity[s] = count
	}
	
	// Average response time
	var avgMinutes sql.NullFloat64
	db.QueryRow(`
		SELECT AVG(EXTRACT(EPOCH FROM (MIN(er.assigned_at) - ea.created_at))/60)
		FROM emergency_alerts ea
		JOIN emergency_responders er ON ea.alert_id = er.alert_id
		WHERE ea.created_at > CURRENT_TIMESTAMP - INTERVAL '30 days'
	`).Scan(&avgMinutes)
	
	if avgMinutes.Valid {
		stats.AverageResponse = fmt.Sprintf("%.1f min", avgMinutes.Float64)
	} else {
		stats.AverageResponse = "N/A"
	}
	
	return stats
}

// Notification functions

func sendEmergencyNotifications(alert *EmergencyAlert) {
	notification := Notification{
		ID:       generateNotificationID(),
		Type:     "emergency",
		Priority: alert.Severity,
		Subject:  fmt.Sprintf("Emergency Alert: %s", alert.Title),
		Message:  alert.Description,
		Data: map[string]interface{}{
			"alert": alert,
		},
		Channels:  []string{"in-app", "email", "push"},
		CreatedAt: time.Now(),
	}
	
	// Get recipients based on alert type
	recipients := getEmergencyRecipients(alert)
	notification.Recipients = recipients
	
	if notificationSystem != nil {
		notificationSystem.Send(notification)
	}
}

func sendSOSNotifications(alert *EmergencyAlert) {
	notification := Notification{
		ID:       generateNotificationID(),
		Type:     "sos",
		Priority: "critical",
		Subject:  "🚨 SOS EMERGENCY ALERT",
		Message:  fmt.Sprintf("SOS activated by %s at %s", alert.ReporterName, alert.Location.Address),
		Data: map[string]interface{}{
			"alert":    alert,
			"sos":      true,
			"location": alert.Location,
		},
		Channels:  []string{"in-app", "email", "sms", "push", "voice"},
		CreatedAt: time.Now(),
	}
	
	// Get ALL managers and emergency contacts
	recipients := getAllEmergencyContacts()
	notification.Recipients = recipients
	
	if notificationSystem != nil {
		notificationSystem.Send(notification)
	}
	
	// Also trigger external emergency systems
	triggerExternalEmergencySystems(alert)
}

func notifyResponder(alertID string, responder *EmergencyResponder) {
	alert, err := getEmergencyByID(alertID)
	if err != nil {
		return
	}
	
	notification := Notification{
		ID:       generateNotificationID(),
		Type:     "responder_assignment",
		Priority: alert.Severity,
		Subject:  "Emergency Response Assignment",
		Message:  fmt.Sprintf("You have been assigned to respond to: %s", alert.Title),
		Data: map[string]interface{}{
			"alert":     alert,
			"responder": responder,
		},
		Recipients: []Recipient{{
			UserID:   strconv.Itoa(responder.UserID),
			Username: responder.Name,
		}},
		Channels:  []string{"in-app", "push", "sms"},
		CreatedAt: time.Now(),
	}
	
	if notificationSystem != nil {
		notificationSystem.Send(notification)
	}
}

// WebSocket broadcast functions

func broadcastEmergencyUpdate(alert *EmergencyAlert) {
	if wsHub == nil {
		return
	}
	
	message := WSMessage{
		Type: "emergency_update",
		Data: map[string]interface{}{
			"alert": alert,
		},
		Timestamp: time.Now(),
	}
	
	messageJSON, _ := json.Marshal(message)
	wsHub.broadcast <- messageJSON
}

func broadcastSOSAlert(alert *EmergencyAlert) {
	if wsHub == nil {
		return
	}
	
	message := WSMessage{
		Type: "sos_alert",
		Data: map[string]interface{}{
			"alert":    alert,
			"critical": true,
			"sound":    "emergency",
		},
		Timestamp: time.Now(),
	}
	
	messageJSON, _ := json.Marshal(message)
	
	// Send to all connected clients immediately
	wsHub.mu.RLock()
	defer wsHub.mu.RUnlock()
	
	for client := range wsHub.clients {
		select {
		case client.send <- messageJSON:
		default:
			// If channel is full, force send by clearing buffer
			select {
			case <-client.send:
				client.send <- messageJSON
			default:
			}
		}
	}
}

// External system integration

func triggerExternalEmergencySystems(alert *EmergencyAlert) {
	// In production, this would:
	// - Call 911 dispatch API
	// - Send alerts to school administration
	// - Activate building lockdown systems
	// - Notify law enforcement
	// - etc.
	
	log.Printf("CRITICAL: External emergency systems triggered for alert %s", alert.AlertID)
}

func getEmergencyRecipients(alert *EmergencyAlert) []Recipient {
	var recipients []Recipient
	
	// Always include all managers
	rows, _ := db.Query(`
		SELECT id, username, email FROM users 
		WHERE role = 'manager' AND approved = true
	`)
	defer rows.Close()
	
	for rows.Next() {
		var userID int
		var username, email string
		if err := rows.Scan(&userID, &username, &email); err == nil {
			recipients = append(recipients, Recipient{
				UserID:   strconv.Itoa(userID),
				Username: username,
				Email:    email,
			})
		}
	}
	
	// Add driver if vehicle-specific
	if alert.VehicleID != "" {
		// Get driver assigned to vehicle
		var driverUsername string
		err := db.QueryRow(`
			SELECT driver_id FROM route_assignments 
			WHERE bus_id = $1 AND CURRENT_DATE BETWEEN start_date AND end_date
			LIMIT 1
		`, alert.VehicleID).Scan(&driverUsername)
		
		if err == nil {
			var userID int
			var email string
			err = db.QueryRow(`
				SELECT id, email FROM users WHERE username = $1
			`, driverUsername).Scan(&userID, &email)
			
			if err == nil {
				recipients = append(recipients, Recipient{
					UserID:   strconv.Itoa(userID),
					Username: driverUsername,
					Email:    email,
				})
			}
		}
	}
	
	return recipients
}

func getAllEmergencyContacts() []Recipient {
	var recipients []Recipient
	
	// Get all managers
	rows, _ := db.Query(`
		SELECT id, username, email, phone FROM users 
		WHERE role = 'manager' AND approved = true
	`)
	defer rows.Close()
	
	for rows.Next() {
		var userID int
		var username, email string
		var phone sql.NullString
		if err := rows.Scan(&userID, &username, &email, &phone); err == nil {
			r := Recipient{
				UserID:   strconv.Itoa(userID),
				Username: username,
				Email:    email,
			}
			if phone.Valid {
				r.Phone = phone.String
			}
			recipients = append(recipients, r)
		}
	}
	
	// Get emergency contacts from database
	rows2, _ := db.Query(`
		SELECT name, email, phone FROM emergency_contacts 
		WHERE is_active = true 
		ORDER BY priority
	`)
	defer rows2.Close()
	
	for rows2.Next() {
		var name, email, phone string
		if err := rows2.Scan(&name, &email, &phone); err == nil {
			recipients = append(recipients, Recipient{
				UserID:   "contact",
				Username: name,
				Email:    email,
				Phone:    phone,
			})
		}
	}
	
	return recipients
}

// getUserID gets user ID from username
func getUserID(username string) int {
	var id int
	db.QueryRow("SELECT id FROM users WHERE username = $1", username).Scan(&id)
	return id
}

// getUsernameByID gets username from user ID
func getUsernameByID(userID int) string {
	var username string
	db.QueryRow("SELECT username FROM users WHERE id = $1", userID).Scan(&username)
	return username
}

// generateNotificationID creates a unique ID for notifications
func generateNotificationID() string {
	return fmt.Sprintf("notif_%d_%s", time.Now().Unix(), generateRandomString(8))
}