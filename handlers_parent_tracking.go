package main

import (
	"database/sql"
	"log"
	"net/http"
)

// ParentChild represents a child's information for parent tracking
type ParentChild struct {
	ID        int    `json:"id" db:"id"`
	Name      string `json:"name" db:"name"`
	Grade     string `json:"grade" db:"grade"`
	BusID     string `json:"bus_id" db:"bus_id"`
	RouteID   string `json:"route_id" db:"route_id"`
	RouteName string `json:"route_name" db:"route_name"`
}

// Parent tracking page handler
func parentTrackingHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// For demo, create mock children data
	// In production, this would query the database for parent's children
	children := []ParentChild{
		{
			ID:        1,
			Name:      "Sarah Johnson",
			Grade:     "5th",
			BusID:     "101",
			RouteID:   "route_a",
			RouteName: "Route A - North District",
		},
		{
			ID:        2,
			Name:      "Michael Johnson",
			Grade:     "3rd",
			BusID:     "102",
			RouteID:   "route_b",
			RouteName: "Route B - East District",
		},
	}

	// Try to get real children from database
	if user.Role == "parent" {
		realChildren := getParentChildren(user.Username)
		if len(realChildren) > 0 {
			children = realChildren
		}
	}

	renderTemplate(w, r, "parent_tracking.html", map[string]interface{}{
		"User":       user,
		"ParentName": user.Username,
		"Children":   children,
	})
}

// Get children for a parent from database
func getParentChildren(parentUsername string) []ParentChild {
	var children []ParentChild

	query := `
		SELECT 
			s.id,
			s.name,
			s.grade,
			COALESCE(s.bus_id, ''),
			COALESCE(s.route_id, ''),
			COALESCE(r.name, 'No Route Assigned') as route_name
		FROM students s
		LEFT JOIN routes r ON s.route_id = r.id
		WHERE s.parent_email = $1 OR s.parent_phone = $1
		ORDER BY s.name
	`

	rows, err := db.Query(query, parentUsername)
	if err != nil {
		log.Printf("Error fetching parent's children: %v", err)
		return children
	}
	defer rows.Close()

	for rows.Next() {
		var child ParentChild
		var busID, routeID sql.NullString
		
		err := rows.Scan(
			&child.ID,
			&child.Name,
			&child.Grade,
			&busID,
			&routeID,
			&child.RouteName,
		)
		if err != nil {
			log.Printf("Error scanning child: %v", err)
			continue
		}

		if busID.Valid {
			child.BusID = busID.String
		}
		if routeID.Valid {
			child.RouteID = routeID.String
		}

		children = append(children, child)
	}

	return children
}

// Parent notification settings handler (enhanced version)
func parentNotificationSettingsHandlerEnhanced(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	if r.Method == http.MethodPost {
		// Handle notification settings update
		r.ParseForm()
		
		// Update settings in database
		_, err := db.Exec(`
			INSERT INTO parent_notifications (parent_id, bus_arrival, route_delays, emergency_alerts, notification_method)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (parent_id) DO UPDATE SET
				bus_arrival = $2,
				route_delays = $3,
				emergency_alerts = $4,
				notification_method = $5,
				updated_at = NOW()
		`,
			user.Username,
			r.FormValue("bus_arrival") == "on",
			r.FormValue("route_delays") == "on",
			r.FormValue("emergency_alerts") == "on",
			r.FormValue("notification_method"),
		)

		if err != nil {
			log.Printf("Error updating notification settings: %v", err)
			http.Error(w, "Failed to update settings", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/parent-notification-settings?success=1", http.StatusSeeOther)
		return
	}

	// Get current settings
	var settings struct {
		BusArrival       bool
		RouteDelays      bool
		EmergencyAlerts  bool
		NotificationMethod string
	}

	err := db.QueryRow(`
		SELECT bus_arrival, route_delays, emergency_alerts, notification_method
		FROM parent_notifications
		WHERE parent_id = $1
	`, user.Username).Scan(
		&settings.BusArrival,
		&settings.RouteDelays,
		&settings.EmergencyAlerts,
		&settings.NotificationMethod,
	)

	if err != nil && err != sql.ErrNoRows {
		log.Printf("Error fetching notification settings: %v", err)
	}

	// Default settings if none exist
	if err == sql.ErrNoRows {
		settings.BusArrival = true
		settings.RouteDelays = true
		settings.EmergencyAlerts = true
		settings.NotificationMethod = "email"
	}

	renderTemplate(w, r, "parent_notification_settings.html", map[string]interface{}{
		"User":     user,
		"Settings": settings,
		"Success":  r.URL.Query().Get("success") == "1",
	})
}

// Get bus ETA for parent
func getParentBusETAHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	busID := r.URL.Query().Get("bus_id")
	if busID == "" {
		http.Error(w, "Bus ID required", http.StatusBadRequest)
		return
	}

	// For demo, return mock ETA
	// In production, calculate based on real GPS position and route
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{
		"eta_minutes": 15,
		"distance_miles": 3.2,
		"next_stop": "Oak Street & Main",
		"stops_remaining": 4
	}`))
}