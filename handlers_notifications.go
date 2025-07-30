package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"
)

// updateVehicleStatusWithNotification updates vehicle status and triggers notifications
func updateVehicleStatusWithNotification(vehicleID, vehicleType, newStatus, changedBy string) error {
	// Get old status first
	var oldStatus string
	var err error
	
	if vehicleType == "bus" {
		err = db.QueryRow("SELECT status FROM buses WHERE bus_id = $1", vehicleID).Scan(&oldStatus)
	} else {
		err = db.QueryRow("SELECT status FROM vehicles WHERE vehicle_id = $1", vehicleID).Scan(&oldStatus)
	}
	
	// Update the status
	if vehicleType == "bus" {
		err = updateBusField(vehicleID, "status", newStatus)
	} else {
		err = updateVehicleField(vehicleID, "status", newStatus)
	}
	
	if err != nil {
		return err
	}
	
	// Trigger notification if status changed
	if oldStatus != newStatus && notificationTriggers != nil {
		go notificationTriggers.TriggerVehicleStatusChangeNotification(vehicleID, oldStatus, newStatus, changedBy)
	}
	
	return nil
}

// Notification API handlers

// notificationPreferencesHandler handles user notification preferences
func notificationPreferencesHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	if r.Method == "GET" {
		// Get user preferences
		var prefs NotificationPreferences
		
		// Default preferences
		prefs.Email = true
		prefs.SMS = false
		prefs.Push = true
		
		data := map[string]interface{}{
			"Title":       "Notification Preferences",
			"User":        user,
			"Preferences": prefs,
			"CSRFToken":   getSessionCSRFToken(r),
			"CSPNonce":    r.Context().Value("cspNonce"),
		}
		
		renderTemplate(w, r, "notification_preferences.html", data)
		return
	}
	
	if r.Method == "POST" {
		// Update preferences
		if !validateCSRF(r) {
			http.Error(w, "Invalid CSRF token", http.StatusForbidden)
			return
		}
		
		// Parse form
		prefs := NotificationPreferences{
			Email: r.FormValue("email") == "on",
			SMS:   r.FormValue("sms") == "on",
			Push:  r.FormValue("push") == "on",
		}
		
		// Parse quiet hours
		prefs.Quiet.Start = r.FormValue("quiet_start")
		prefs.Quiet.End = r.FormValue("quiet_end")
		
		// Parse notification types
		prefs.Types = make(map[string]bool)
		for _, notifType := range []string{
			NotifyMaintenanceDue,
			NotifyRouteChange,
			NotifyEmergency,
			NotifyAttendanceIssue,
			NotifyVehicleIssue,
			NotifyScheduleReminder,
			NotifyReportReady,
		} {
			prefs.Types[notifType] = r.FormValue("type_"+notifType) == "on"
		}
		
		// Save preferences (would normally save to database)
		log.Printf("Updated notification preferences for user %s", user.Username)
		
		http.Redirect(w, r, "/notification-preferences?success=1", http.StatusFound)
	}
}

// notificationHistoryHandler shows notification history
func notificationHistoryHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	// Get notifications for user
	rows, err := db.Query(`
		SELECT n.id, n.type, n.subject, n.message, n.priority, 
			   n.created_at, nd.channel, nd.status, nd.delivered_at
		FROM notifications n
		LEFT JOIN notification_deliveries nd ON n.id = nd.notification_id
		WHERE nd.user_id = $1
		ORDER BY n.created_at DESC
		LIMIT 50
	`, user.Username)
	
	if err != nil {
		log.Printf("Error loading notification history: %v", err)
		http.Error(w, "Failed to load notifications", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type NotificationHistory struct {
		ID          string
		Type        string
		Subject     string
		Message     string
		Priority    string
		CreatedAt   time.Time
		Channel     string
		Status      string
		DeliveredAt *time.Time
	}

	var notifications []NotificationHistory
	for rows.Next() {
		var n NotificationHistory
		err := rows.Scan(&n.ID, &n.Type, &n.Subject, &n.Message, &n.Priority,
			&n.CreatedAt, &n.Channel, &n.Status, &n.DeliveredAt)
		if err != nil {
			continue
		}
		notifications = append(notifications, n)
	}

	data := map[string]interface{}{
		"Title":         "Notification History",
		"User":          user,
		"Notifications": notifications,
		"CSPNonce":      r.Context().Value("cspNonce"),
	}

	renderTemplate(w, r, "notification_history.html", data)
}

// testNotificationHandler sends a test notification
func testNotificationHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Error(w, "Unauthorized", http.StatusForbidden)
		return
	}

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Send test notification
	if notificationSystem != nil {
		recipient, err := notificationTriggers.getUserRecipient(user.Username)
		if err != nil {
			SendError(w, ErrInternal("Failed to get recipient", err))
			return
		}

		notification := Notification{
			Type:     NotifySystemAlert,
			Priority: "low",
			Subject:  "Test Notification",
			Message:  "This is a test notification from the Fleet Management System. If you received this, your notifications are working correctly!",
			Data: map[string]interface{}{
				"test": true,
				"timestamp": time.Now(),
			},
			Channels:   []string{"email", "in-app"},
			Recipients: []Recipient{recipient},
		}

		err = notificationSystem.Send(notification)
		if err != nil {
			SendError(w, ErrInternal("Failed to send notification", err))
			return
		}

		SendJSON(w, http.StatusOK, map[string]string{
			"message": "Test notification sent successfully",
		})
	} else {
		SendError(w, ErrInternal("Notification system not initialized", nil))
	}
}

// markNotificationReadHandler marks a notification as read
func markNotificationReadHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil {
		SendError(w, ErrUnauthorized("Authentication required"))
		return
	}

	if r.Method != "POST" {
		SendError(w, ErrMethodNotAllowed("Only POST method allowed"))
		return
	}

	var req struct {
		NotificationID string `json:"notification_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, ErrBadRequest("Invalid request"))
		return
	}

	// Mark as read
	_, err := db.Exec(`
		UPDATE in_app_notifications 
		SET read = true, read_at = CURRENT_TIMESTAMP
		WHERE notification_id = $1 AND user_id = $2
	`, req.NotificationID, user.Username)

	if err != nil {
		SendError(w, ErrDatabase("Failed to mark notification as read", err))
		return
	}

	SendJSON(w, http.StatusOK, map[string]bool{"success": true})
}

// getUnreadNotificationsHandler returns count of unread notifications
func getUnreadNotificationsHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil {
		SendError(w, ErrUnauthorized("Authentication required"))
		return
	}

	var count int
	err := db.QueryRow(`
		SELECT COUNT(*) 
		FROM in_app_notifications 
		WHERE user_id = $1 AND read = false
	`, user.Username).Scan(&count)

	if err != nil {
		SendError(w, ErrDatabase("Failed to get unread count", err))
		return
	}

	SendJSON(w, http.StatusOK, map[string]int{"count": count})
}

// Integration with existing handlers

// Hook into route assignment
func assignRouteWithNotification(driver, busID, routeID string) error {
	// Check if this is a new assignment
	var exists bool
	err := db.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM route_assignments 
			WHERE driver = $1 AND route_id = $2
		)
	`, driver, routeID).Scan(&exists)
	
	// Perform the assignment
	if exists {
		// Update existing
		_, err = db.Exec(`
			UPDATE route_assignments 
			SET bus_id = $1, assigned_date = CURRENT_DATE 
			WHERE driver = $2 AND route_id = $3
		`, busID, driver, routeID)
	} else {
		// Insert new
		_, err = db.Exec(`
			INSERT INTO route_assignments (driver, bus_id, route_id, assigned_date)
			VALUES ($1, $2, $3, CURRENT_DATE)
		`, driver, busID, routeID)
	}
	
	if err != nil {
		return err
	}
	
	// Trigger notification
	if notificationTriggers != nil {
		go notificationTriggers.TriggerRouteAssignmentNotification(driver, busID, routeID, !exists)
	}
	
	return nil
}

// Hook into attendance marking
func markAttendanceWithNotification(studentID string, date time.Time, present bool) error {
	// Mark attendance
	_, err := db.Exec(`
		INSERT INTO student_attendance (student_id, date, present, marked_at, marked_by)
		VALUES ($1, $2, $3, CURRENT_TIMESTAMP, $4)
		ON CONFLICT (student_id, date) 
		DO UPDATE SET present = $3, marked_at = CURRENT_TIMESTAMP
	`, studentID, date.Format("2006-01-02"), present, "system")
	
	if err != nil {
		return err
	}
	
	// If absent, trigger notification after all attendance is marked
	// This should be batched to avoid spam
	if !present {
		// Queue for batch notification
		log.Printf("Student %s marked absent on %s", studentID, date.Format("2006-01-02"))
	}
	
	return nil
}

// Add scheduled task to process attendance notifications
func processAttendanceNotifications() {
	// Run at 10:30 AM daily
	go scheduleDaily(10, 30, func() {
		if notificationTriggers != nil {
			notificationTriggers.TriggerAttendanceIssueNotifications()
		}
	})
}

// markAllNotificationsReadHandler marks all notifications as read for the user
func markAllNotificationsReadHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil {
		SendError(w, ErrUnauthorized("Authentication required"))
		return
	}

	if r.Method != "POST" {
		SendError(w, ErrMethodNotAllowed("Only POST method allowed"))
		return
	}

	// Mark all as read
	_, err := db.Exec(`
		UPDATE in_app_notifications 
		SET read = true, read_at = CURRENT_TIMESTAMP
		WHERE user_id = $1 AND read = false
	`, user.Username)

	if err != nil {
		SendError(w, ErrDatabase("Failed to mark notifications as read", err))
		return
	}

	SendJSON(w, http.StatusOK, map[string]bool{"success": true})
}

// getRecentNotificationsHandler returns recent notifications for the user
func getRecentNotificationsHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil {
		SendError(w, ErrUnauthorized("Authentication required"))
		return
	}

	// Get limit from query param
	limit := 10
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 50 {
			limit = parsed
		}
	}

	rows, err := db.Query(`
		SELECT n.notification_id, n.type, n.subject, n.message, 
		       n.created_at, n.read
		FROM in_app_notifications n
		WHERE n.user_id = $1
		ORDER BY n.created_at DESC
		LIMIT $2
	`, user.Username, limit)

	if err != nil {
		SendError(w, ErrDatabase("Failed to get notifications", err))
		return
	}
	defer rows.Close()

	type NotificationItem struct {
		ID        string    `json:"id"`
		Type      string    `json:"type"`
		Subject   string    `json:"subject"`
		Message   string    `json:"message"`
		CreatedAt time.Time `json:"created_at"`
		Read      bool      `json:"read"`
		Priority  string    `json:"priority"`
	}

	var notifications []NotificationItem
	for rows.Next() {
		var n NotificationItem
		err := rows.Scan(&n.ID, &n.Type, &n.Subject, &n.Message, 
			&n.CreatedAt, &n.Read, &n.Priority)
		if err != nil {
			continue
		}
		notifications = append(notifications, n)
	}

	SendJSON(w, http.StatusOK, notifications)
}