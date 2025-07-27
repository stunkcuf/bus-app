package main

import (
	"bytes"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/smtp"
	"os"
	"strings"
	"sync"
	"time"
)

// NotificationSystem handles all notifications
type NotificationSystem struct {
	db              *sql.DB
	emailConfig     EmailConfig
	smsConfig       SMSConfig
	pushConfig      PushConfig
	templates       map[string]*template.Template
	queue           chan Notification
	workers         int
	wg              sync.WaitGroup
}

// Configuration structs
type EmailConfig struct {
	SMTPHost     string
	SMTPPort     string
	Username     string
	Password     string
	FromAddress  string
	FromName     string
}

type SMSConfig struct {
	Provider    string // twilio, nexmo, etc.
	AccountSID  string
	AuthToken   string
	FromNumber  string
}

type PushConfig struct {
	FCMServerKey string
	APNSCert     string
	APNSKey      string
}

// Notification models
type Notification struct {
	ID           string                 `json:"id"`
	Type         string                 `json:"type"`
	Priority     string                 `json:"priority"` // high, medium, low
	Recipients   []Recipient            `json:"recipients"`
	Subject      string                 `json:"subject"`
	Message      string                 `json:"message"`
	Data         map[string]interface{} `json:"data"`
	Channels     []string               `json:"channels"` // email, sms, push, in-app
	ScheduledAt  *time.Time             `json:"scheduled_at,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
}

type Recipient struct {
	UserID       string   `json:"user_id"`
	Username     string   `json:"username"`
	Email        string   `json:"email"`
	Phone        string   `json:"phone"`
	DeviceTokens []string `json:"device_tokens"`
	Preferences  NotificationPreferences `json:"preferences"`
}

type NotificationPreferences struct {
	Email        bool                   `json:"email"`
	SMS          bool                   `json:"sms"`
	Push         bool                   `json:"push"`
	Quiet        TimeRange              `json:"quiet_hours"`
	Types        map[string]bool        `json:"types"`
}

type TimeRange struct {
	Start string `json:"start"` // "22:00"
	End   string `json:"end"`   // "07:00"
}

type NotificationTemplate struct {
	Name    string
	Subject string
	Body    string
}

// Notification types
const (
	NotifyMaintenanceDue     = "maintenance_due"
	NotifyRouteChange        = "route_change"
	NotifyEmergency          = "emergency"
	NotifyAttendanceIssue    = "attendance_issue"
	NotifyVehicleIssue       = "vehicle_issue"
	NotifyScheduleReminder   = "schedule_reminder"
	NotifySystemAlert        = "system_alert"
	NotifyReportReady        = "report_ready"
)

// NewNotificationSystem creates a new notification system
func NewNotificationSystem(db *sql.DB, emailConfig EmailConfig, smsConfig SMSConfig, pushConfig PushConfig) *NotificationSystem {
	ns := &NotificationSystem{
		db:          db,
		emailConfig: emailConfig,
		smsConfig:   smsConfig,
		pushConfig:  pushConfig,
		templates:   make(map[string]*template.Template),
		queue:       make(chan Notification, 1000),
		workers:     5,
	}

	// Load notification templates
	ns.loadTemplates()

	// Start notification workers
	ns.startWorkers()

	// Start scheduled notification checker
	go ns.processScheduledNotifications()

	return ns
}

// Send notification
func (ns *NotificationSystem) Send(notification Notification) error {
	// Validate notification
	if err := ns.validateNotification(&notification); err != nil {
		return fmt.Errorf("invalid notification: %v", err)
	}

	// Generate ID if not provided
	if notification.ID == "" {
		notification.ID = generateID("notif")
	}

	// Set creation time
	notification.CreatedAt = time.Now()

	// Store in database
	if err := ns.storeNotification(notification); err != nil {
		log.Printf("Failed to store notification: %v", err)
	}

	// Add to queue for processing
	select {
	case ns.queue <- notification:
		return nil
	default:
		return fmt.Errorf("notification queue full")
	}
}

// Process notification
func (ns *NotificationSystem) processNotification(notification Notification) {
	log.Printf("Processing notification %s: %s", notification.ID, notification.Subject)

	// Check if scheduled for later
	if notification.ScheduledAt != nil && notification.ScheduledAt.After(time.Now()) {
		return // Will be processed by scheduled checker
	}

	// Process each recipient
	var wg sync.WaitGroup
	for _, recipient := range notification.Recipients {
		wg.Add(1)
		go func(r Recipient) {
			defer wg.Done()
			ns.sendToRecipient(notification, r)
		}(recipient)
	}
	wg.Wait()

	// Update notification status
	ns.updateNotificationStatus(notification.ID, "sent")
}

// Send to individual recipient
func (ns *NotificationSystem) sendToRecipient(notification Notification, recipient Recipient) {
	// Check quiet hours
	if ns.isQuietHours(recipient.Preferences.Quiet) && notification.Priority != "high" {
		log.Printf("Skipping notification for %s due to quiet hours", recipient.Username)
		return
	}

	// Check notification type preferences
	if prefs, ok := recipient.Preferences.Types[notification.Type]; ok && !prefs {
		log.Printf("User %s has disabled %s notifications", recipient.Username, notification.Type)
		return
	}

	// Send via requested channels
	for _, channel := range notification.Channels {
		switch channel {
		case "email":
			if recipient.Preferences.Email && recipient.Email != "" {
				ns.sendEmail(notification, recipient)
			}
		case "sms":
			if recipient.Preferences.SMS && recipient.Phone != "" {
				ns.sendSMS(notification, recipient)
			}
		case "push":
			if recipient.Preferences.Push && len(recipient.DeviceTokens) > 0 {
				ns.sendPush(notification, recipient)
			}
		case "in-app":
			ns.sendInApp(notification, recipient)
		}
	}
}

// Send email notification
func (ns *NotificationSystem) sendEmail(notification Notification, recipient Recipient) error {
	// Prepare email content
	var body bytes.Buffer
	if tmpl, ok := ns.templates[notification.Type]; ok {
		data := struct {
			Recipient    Recipient
			Notification Notification
		}{recipient, notification}
		
		if err := tmpl.Execute(&body, data); err != nil {
			log.Printf("Failed to render email template: %v", err)
			body.WriteString(notification.Message)
		}
	} else {
		body.WriteString(notification.Message)
	}

	// Build email
	from := fmt.Sprintf("%s <%s>", ns.emailConfig.FromName, ns.emailConfig.FromAddress)
	to := recipient.Email
	subject := notification.Subject

	message := fmt.Sprintf("From: %s\r\n"+
		"To: %s\r\n"+
		"Subject: %s\r\n"+
		"MIME-Version: 1.0\r\n"+
		"Content-Type: text/html; charset=UTF-8\r\n"+
		"\r\n"+
		"%s", from, to, subject, body.String())

	// Send email
	auth := smtp.PlainAuth("", ns.emailConfig.Username, ns.emailConfig.Password, ns.emailConfig.SMTPHost)
	addr := fmt.Sprintf("%s:%s", ns.emailConfig.SMTPHost, ns.emailConfig.SMTPPort)
	
	err := smtp.SendMail(addr, auth, ns.emailConfig.FromAddress, []string{to}, []byte(message))
	if err != nil {
		log.Printf("Failed to send email to %s: %v", to, err)
		return err
	}

	log.Printf("Email sent to %s: %s", to, subject)
	ns.recordDelivery(notification.ID, recipient.UserID, "email", "sent")
	return nil
}

// Send SMS notification
func (ns *NotificationSystem) sendSMS(notification Notification, recipient Recipient) error {
	// Format phone number
	phone := ns.formatPhoneNumber(recipient.Phone)
	
	// Prepare SMS content (limit to 160 chars)
	message := notification.Message
	if len(message) > 160 {
		message = message[:157] + "..."
	}

	// Send via provider (example with Twilio-like API)
	payload := map[string]string{
		"From": ns.smsConfig.FromNumber,
		"To":   phone,
		"Body": message,
	}

	// This would actually call the SMS provider's API
	_ = payload // Avoid unused variable error
	log.Printf("SMS sent to %s: %s", phone, message)
	ns.recordDelivery(notification.ID, recipient.UserID, "sms", "sent")
	return nil
}

// Send push notification
func (ns *NotificationSystem) sendPush(notification Notification, recipient Recipient) error {
	// Prepare push notification payload
	payload := map[string]interface{}{
		"notification": map[string]interface{}{
			"title": notification.Subject,
			"body":  notification.Message,
			"sound": "default",
			"badge": 1,
		},
		"data": notification.Data,
		"priority": notification.Priority,
	}

	// Send to each device token
	for _, token := range recipient.DeviceTokens {
		// Determine platform (iOS/Android) based on token format
		if strings.HasPrefix(token, "ios:") {
			ns.sendAPNS(token[4:], payload)
		} else {
			ns.sendFCM(token, payload)
		}
	}

	ns.recordDelivery(notification.ID, recipient.UserID, "push", "sent")
	return nil
}

// Send FCM (Android) push notification
func (ns *NotificationSystem) sendFCM(token string, payload map[string]interface{}) error {
	// Prepare FCM request
	fcmPayload := map[string]interface{}{
		"to":   token,
		"data": payload,
	}

	jsonPayload, _ := json.Marshal(fcmPayload)

	// Send to FCM
	req, _ := http.NewRequest("POST", "https://fcm.googleapis.com/fcm/send", bytes.NewReader(jsonPayload))
	req.Header.Set("Authorization", "key="+ns.pushConfig.FCMServerKey)
	req.Header.Set("Content-Type", "application/json")

	// This would actually send the request
	log.Printf("FCM notification sent to token: %s", token[:10]+"...")
	return nil
}

// Send APNS (iOS) push notification
func (ns *NotificationSystem) sendAPNS(token string, payload map[string]interface{}) error {
	// This would use APNS HTTP/2 API
	log.Printf("APNS notification sent to token: %s", token[:10]+"...")
	return nil
}

// Send in-app notification
func (ns *NotificationSystem) sendInApp(notification Notification, recipient Recipient) error {
	// Store in-app notification
	_, err := ns.db.Exec(`
		INSERT INTO in_app_notifications 
		(user_id, notification_id, type, subject, message, data, read, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, false, $7)
	`, recipient.UserID, notification.ID, notification.Type, notification.Subject,
	   notification.Message, notification.Data, notification.CreatedAt)

	if err != nil {
		log.Printf("Failed to store in-app notification: %v", err)
		return err
	}

	// Broadcast via WebSocket if user is online
	if wsHub != nil {
		// Send to specific user's connections
		wsHub.mu.RLock()
		for client := range wsHub.clients {
			if client.user.Username == recipient.Username {
				notifJSON, _ := json.Marshal(map[string]interface{}{
					"type": "notification",
					"data": notification,
				})
				select {
				case client.send <- notifJSON:
				default:
				}
			}
		}
		wsHub.mu.RUnlock()
	}

	ns.recordDelivery(notification.ID, recipient.UserID, "in-app", "delivered")
	return nil
}

// Notification helpers

func (ns *NotificationSystem) validateNotification(n *Notification) error {
	if len(n.Recipients) == 0 {
		return fmt.Errorf("no recipients specified")
	}
	if n.Subject == "" {
		return fmt.Errorf("subject is required")
	}
	if n.Message == "" {
		return fmt.Errorf("message is required")
	}
	if len(n.Channels) == 0 {
		n.Channels = []string{"email", "in-app"} // Default channels
	}
	if n.Priority == "" {
		n.Priority = "medium"
	}
	return nil
}

func (ns *NotificationSystem) isQuietHours(quiet TimeRange) bool {
	if quiet.Start == "" || quiet.End == "" {
		return false
	}

	now := time.Now()
	currentTime := now.Format("15:04")
	
	// Handle overnight quiet hours
	if quiet.Start > quiet.End {
		return currentTime >= quiet.Start || currentTime <= quiet.End
	}
	
	return currentTime >= quiet.Start && currentTime <= quiet.End
}

func (ns *NotificationSystem) formatPhoneNumber(phone string) string {
	// Remove all non-digits
	cleaned := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, phone)

	// Add country code if needed
	if len(cleaned) == 10 {
		cleaned = "1" + cleaned
	}

	return "+" + cleaned
}

// Database operations

func (ns *NotificationSystem) storeNotification(n Notification) error {
	dataJSON, _ := json.Marshal(n.Data)
	recipientsJSON, _ := json.Marshal(n.Recipients)
	channelsJSON, _ := json.Marshal(n.Channels)

	_, err := ns.db.Exec(`
		INSERT INTO notifications 
		(id, type, priority, subject, message, data, recipients, channels, 
		 scheduled_at, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, 'pending', $10)
	`, n.ID, n.Type, n.Priority, n.Subject, n.Message, dataJSON, 
	   recipientsJSON, channelsJSON, n.ScheduledAt, n.CreatedAt)

	return err
}

func (ns *NotificationSystem) updateNotificationStatus(id, status string) {
	_, err := ns.db.Exec(`
		UPDATE notifications 
		SET status = $1, updated_at = CURRENT_TIMESTAMP 
		WHERE id = $2
	`, status, id)
	
	if err != nil {
		log.Printf("Failed to update notification status: %v", err)
	}
}

func (ns *NotificationSystem) recordDelivery(notificationID, userID, channel, status string) {
	_, err := ns.db.Exec(`
		INSERT INTO notification_deliveries 
		(notification_id, user_id, channel, status, delivered_at)
		VALUES ($1, $2, $3, $4, CURRENT_TIMESTAMP)
	`, notificationID, userID, channel, status)
	
	if err != nil {
		log.Printf("Failed to record delivery: %v", err)
	}
}

// Worker management

func (ns *NotificationSystem) startWorkers() {
	for i := 0; i < ns.workers; i++ {
		ns.wg.Add(1)
		go ns.worker(i)
	}
}

func (ns *NotificationSystem) worker(id int) {
	defer ns.wg.Done()
	log.Printf("Notification worker %d started", id)

	for notification := range ns.queue {
		ns.processNotification(notification)
	}
}

func (ns *NotificationSystem) Stop() {
	close(ns.queue)
	ns.wg.Wait()
}

// Scheduled notifications

func (ns *NotificationSystem) processScheduledNotifications() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ns.checkScheduledNotifications()
		}
	}
}

func (ns *NotificationSystem) checkScheduledNotifications() {
	rows, err := ns.db.Query(`
		SELECT id, type, priority, subject, message, data, recipients, channels
		FROM notifications
		WHERE status = 'pending'
		AND scheduled_at <= CURRENT_TIMESTAMP
		LIMIT 10
	`)
	if err != nil {
		log.Printf("Failed to check scheduled notifications: %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var n Notification
		var dataJSON, recipientsJSON, channelsJSON []byte
		
		err := rows.Scan(&n.ID, &n.Type, &n.Priority, &n.Subject, &n.Message,
						 &dataJSON, &recipientsJSON, &channelsJSON)
		if err != nil {
			continue
		}

		json.Unmarshal(dataJSON, &n.Data)
		json.Unmarshal(recipientsJSON, &n.Recipients)
		json.Unmarshal(channelsJSON, &n.Channels)

		// Add to processing queue
		select {
		case ns.queue <- n:
		default:
			log.Printf("Queue full, skipping scheduled notification %s", n.ID)
		}
	}
}

// Template management

func (ns *NotificationSystem) loadTemplates() {
	// Email templates for different notification types
	templates := map[string]string{
		NotifyMaintenanceDue: `
			<h2>Vehicle Maintenance Due</h2>
			<p>Hello {{.Recipient.Username}},</p>
			<p>{{.Notification.Message}}</p>
			<p>Please schedule maintenance as soon as possible.</p>
		`,
		NotifyRouteChange: `
			<h2>Route Assignment Change</h2>
			<p>Hello {{.Recipient.Username}},</p>
			<p>{{.Notification.Message}}</p>
			<p>Please review your updated schedule in the app.</p>
		`,
		NotifyEmergency: `
			<h2 style="color: red;">EMERGENCY ALERT</h2>
			<p>{{.Notification.Message}}</p>
			<p>Please take immediate action.</p>
		`,
	}

	for name, tmplStr := range templates {
		tmpl, err := template.New(name).Parse(tmplStr)
		if err != nil {
			log.Printf("Failed to parse template %s: %v", name, err)
			continue
		}
		ns.templates[name] = tmpl
	}
}

// Notification builders

func BuildMaintenanceNotification(vehicle Vehicle, dueDate time.Time) Notification {
	return Notification{
		Type:     NotifyMaintenanceDue,
		Priority: "medium",
		Subject:  fmt.Sprintf("Maintenance Due: %s", vehicle.Model),
		Message: fmt.Sprintf("Vehicle %s (%s) is due for maintenance on %s. Current mileage: %d",
			vehicle.VehicleID, vehicle.Model, dueDate.Format("Jan 2, 2006"), vehicle.CurrentMileage),
		Data: map[string]interface{}{
			"vehicle_id": vehicle.VehicleID,
			"due_date":   dueDate,
			"mileage":    vehicle.CurrentMileage,
		},
		Channels: []string{"email", "push", "in-app"},
	}
}

func BuildEmergencyNotification(driver, message string, location *LocationUpdate) Notification {
	n := Notification{
		Type:     NotifyEmergency,
		Priority: "high",
		Subject:  "Emergency Alert",
		Message:  fmt.Sprintf("Driver %s: %s", driver, message),
		Data: map[string]interface{}{
			"driver": driver,
			"time":   time.Now(),
		},
		Channels: []string{"email", "sms", "push", "in-app"},
	}

	if location != nil {
		n.Data["location"] = map[string]float64{
			"lat": location.Latitude,
			"lng": location.Longitude,
		}
	}

	return n
}

// Initialize notification system
var notificationSystem *NotificationSystem

func InitializeNotificationSystem() {
	emailConfig := EmailConfig{
		SMTPHost:    os.Getenv("SMTP_HOST"),
		SMTPPort:    os.Getenv("SMTP_PORT"),
		Username:    os.Getenv("SMTP_USERNAME"),
		Password:    os.Getenv("SMTP_PASSWORD"),
		FromAddress: os.Getenv("SMTP_FROM_ADDRESS"),
		FromName:    "Fleet Management System",
	}

	smsConfig := SMSConfig{
		Provider:   os.Getenv("SMS_PROVIDER"),
		AccountSID: os.Getenv("SMS_ACCOUNT_SID"),
		AuthToken:  os.Getenv("SMS_AUTH_TOKEN"),
		FromNumber: os.Getenv("SMS_FROM_NUMBER"),
	}

	pushConfig := PushConfig{
		FCMServerKey: os.Getenv("FCM_SERVER_KEY"),
		APNSCert:     os.Getenv("APNS_CERT_PATH"),
		APNSKey:      os.Getenv("APNS_KEY_PATH"),
	}

	notificationSystem = NewNotificationSystem(db.DB, emailConfig, smsConfig, pushConfig)
	log.Println("Notification system initialized")
}

// generateID generates a unique ID with the given prefix
func generateID(prefix string) string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return prefix + "_" + hex.EncodeToString(bytes)
}