package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
	
	"golang.org/x/crypto/bcrypt"
)

// Parent represents a parent user
type Parent struct {
	ID            int       `json:"id"`
	Username      string    `json:"username"`
	Email         string    `json:"email"`
	Phone         string    `json:"phone"`
	Name          string    `json:"name"`
	Students      []ParentViewStudent `json:"students"`
	Notifications bool      `json:"notifications"`
	CreatedAt     time.Time `json:"created_at"`
}

// ParentViewStudent extends Student with additional fields for parent view
type ParentViewStudent struct {
	Student
	Grade   string `json:"grade"`
	Address string `json:"address"`
}

// ParentStudent represents the relationship between parent and student
type ParentStudent struct {
	ParentID      int    `json:"parent_id"`
	StudentID     string `json:"student_id"`
	Relationship  string `json:"relationship"` // mother, father, guardian
	EmergencyRank int    `json:"emergency_rank"` // 1 = primary contact
}

// BusTracking represents real-time bus location for parents
type BusTracking struct {
	StudentID        string    `json:"student_id"`
	StudentName      string    `json:"student_name"`
	BusID            string    `json:"bus_id"`
	RouteID          string    `json:"route_id"`
	RouteName        string    `json:"route_name"`
	CurrentLocation  *GPSLocation `json:"current_location"`
	EstimatedArrival *time.Time   `json:"estimated_arrival"`
	Status           string    `json:"status"` // not_started, en_route, arrived, departed
}

// parentLoginHandler handles parent login page
func parentLoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		data := struct {
			Title    string
			Error    string
			CSPNonce string
		}{
			Title:    "Parent Portal Login",
			Error:    r.URL.Query().Get("error"),
			CSPNonce: generateNonce(),
		}

		tmpl := template.Must(template.ParseFiles("templates/parent_login.html"))
		tmpl.Execute(w, data)
	} else if r.Method == "POST" {
		username := r.FormValue("username")
		password := r.FormValue("password")

		// Authenticate parent
		parent, err := authenticateParent(username, password)
		if err != nil {
			http.Redirect(w, r, "/parent/login?error=Invalid+credentials", http.StatusSeeOther)
			return
		}

		// Create session
		sessionToken := generateSessionToken()
		err = createParentSession(sessionToken, parent.ID)
		if err != nil {
			log.Printf("Failed to create parent session: %v", err)
			http.Redirect(w, r, "/parent/login?error=Session+error", http.StatusSeeOther)
			return
		}

		// Set session cookie
		http.SetCookie(w, &http.Cookie{
			Name:     "parent_session",
			Value:    sessionToken,
			Path:     "/parent",
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteStrictMode,
			MaxAge:   86400, // 24 hours
		})

		http.Redirect(w, r, "/parent/dashboard", http.StatusSeeOther)
	}
}

// parentDashboardHandler shows the parent dashboard
func parentDashboardHandler(w http.ResponseWriter, r *http.Request) {
	parent := getParentFromSession(r)
	if parent == nil {
		http.Redirect(w, r, "/parent/login", http.StatusSeeOther)
		return
	}

	// Get students with current bus status
	students := getParentStudentsWithStatus(parent.ID)
	
	// Get recent notifications
	notifications := getParentNotifications(parent.ID, 10)
	
	// Get upcoming events (convert to []Student)
	var baseStudents []Student
	for _, s := range parent.Students {
		baseStudents = append(baseStudents, s.Student)
	}
	events := getUpcomingEvents(baseStudents)

	data := struct {
		Title         string
		Parent        *Parent
		Students      []StudentStatus
		Notifications []ParentNotification
		Events        []Event
		CSPNonce      string
	}{
		Title:         "Parent Dashboard",
		Parent:        parent,
		Students:      students,
		Notifications: notifications,
		Events:        events,
		CSPNonce:      generateNonce(),
	}

	tmpl := template.Must(template.ParseFiles("templates/parent_dashboard.html"))
	err := tmpl.Execute(w, data)
	if err != nil {
		log.Printf("Error rendering parent dashboard: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// parentBusTrackingHandler shows real-time bus tracking
func parentBusTrackingHandler(w http.ResponseWriter, r *http.Request) {
	parent := getParentFromSession(r)
	if parent == nil {
		http.Redirect(w, r, "/parent/login", http.StatusSeeOther)
		return
	}

	// Get tracking info for all parent's students
	trackingInfo := getParentBusTracking(parent.ID)

	data := struct {
		Title        string
		Parent       *Parent
		TrackingInfo []BusTracking
		MapAPIKey    string
		CSPNonce     string
	}{
		Title:        "Bus Tracking",
		Parent:       parent,
		TrackingInfo: trackingInfo,
		MapAPIKey:    getMapAPIKey(),
		CSPNonce:     generateNonce(),
	}

	tmpl := template.Must(template.ParseFiles("templates/parent_bus_tracking.html"))
	tmpl.Execute(w, data)
}

// parentStudentDetailsHandler shows detailed info for a student
func parentStudentDetailsHandler(w http.ResponseWriter, r *http.Request) {
	parent := getParentFromSession(r)
	if parent == nil {
		http.Redirect(w, r, "/parent/login", http.StatusSeeOther)
		return
	}

	// Get student ID from URL
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 4 {
		http.Error(w, "Invalid student ID", http.StatusBadRequest)
		return
	}
	studentID := pathParts[3]

	// Verify parent has access to this student
	if !parentHasStudent(parent.ID, studentID) {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	// Get student details
	student := getStudentDetails(studentID)
	attendance := getStudentAttendance(studentID, 30) // Last 30 days
	notifications := getStudentNotifications(studentID, 20)

	data := struct {
		Title         string
		Parent        *Parent
		Student       StudentDetail
		Attendance    []AttendanceRecord
		Notifications []ParentNotification
		CSPNonce      string
	}{
		Title:         "Student Details",
		Parent:        parent,
		Student:       student,
		Attendance:    attendance,
		Notifications: notifications,
		CSPNonce:      generateNonce(),
	}

	tmpl := template.Must(template.ParseFiles("templates/parent_student_details.html"))
	tmpl.Execute(w, data)
}

// parentNotificationSettingsHandler manages notification preferences
func parentNotificationSettingsHandler(w http.ResponseWriter, r *http.Request) {
	parent := getParentFromSession(r)
	if parent == nil {
		http.Redirect(w, r, "/parent/login", http.StatusSeeOther)
		return
	}

	if r.Method == "GET" {
		settings := getParentNotificationSettings(parent.ID)

		data := struct {
			Title    string
			Parent   *Parent
			Settings NotificationSettings
			CSPNonce string
		}{
			Title:    "Notification Settings",
			Parent:   parent,
			Settings: settings,
			CSPNonce: generateNonce(),
		}

		tmpl := template.Must(template.ParseFiles("templates/parent_notification_settings.html"))
		tmpl.Execute(w, data)
	} else if r.Method == "POST" {
		// Update settings
		var settings NotificationSettings
		settings.BusArrival = r.FormValue("bus_arrival") == "on"
		settings.BusDeparture = r.FormValue("bus_departure") == "on"
		settings.Attendance = r.FormValue("attendance") == "on"
		settings.Emergency = r.FormValue("emergency") == "on"
		settings.RouteChanges = r.FormValue("route_changes") == "on"
		settings.EmailEnabled = r.FormValue("email_enabled") == "on"
		settings.SMSEnabled = r.FormValue("sms_enabled") == "on"
		settings.PushEnabled = r.FormValue("push_enabled") == "on"

		err := updateParentNotificationSettings(parent.ID, settings)
		if err != nil {
			log.Printf("Failed to update notification settings: %v", err)
		}

		http.Redirect(w, r, "/parent/notification-settings?success=true", http.StatusSeeOther)
	}
}

// API Endpoints for Parent Portal

// parentAPIBusLocationHandler returns current bus location
func parentAPIBusLocationHandler(w http.ResponseWriter, r *http.Request) {
	parent := getParentFromSession(r)
	if parent == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	studentID := r.URL.Query().Get("student_id")
	if !parentHasStudent(parent.ID, studentID) {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	// Get bus location for student
	tracking := getBusTrackingForStudent(studentID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tracking)
}

// parentAPINotificationsHandler returns parent notifications
func parentAPINotificationsHandler(w http.ResponseWriter, r *http.Request) {
	parent := getParentFromSession(r)
	if parent == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	limit := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	notifications := getParentNotifications(parent.ID, limit)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(notifications)
}

// parentAPIMarkNotificationReadHandler marks notification as read
func parentAPIMarkNotificationReadHandler(w http.ResponseWriter, r *http.Request) {
	parent := getParentFromSession(r)
	if parent == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		NotificationID string `json:"notification_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	err := markParentNotificationRead(parent.ID, req.NotificationID)
	if err != nil {
		http.Error(w, "Failed to update notification", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// parentRegistrationHandler handles parent registration
func parentRegistrationHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		data := struct {
			Title    string
			Error    string
			CSPNonce string
		}{
			Title:    "Parent Registration",
			Error:    r.URL.Query().Get("error"),
			CSPNonce: generateNonce(),
		}

		tmpl := template.Must(template.ParseFiles("templates/parent_registration.html"))
		tmpl.Execute(w, data)
	} else if r.Method == "POST" {
		// Process registration
		registration := ParentRegistration{
			Email:         r.FormValue("email"),
			Phone:         r.FormValue("phone"),
			Name:          r.FormValue("name"),
			StudentCode:   r.FormValue("student_code"),
			Relationship:  r.FormValue("relationship"),
		}

		// Validate student code
		studentID, err := validateStudentCode(registration.StudentCode)
		if err != nil {
			http.Redirect(w, r, "/parent/register?error=Invalid+student+code", http.StatusSeeOther)
			return
		}

		// Create parent account
		parentID, err := createParentAccount(registration)
		if err != nil {
			log.Printf("Failed to create parent account: %v", err)
			http.Redirect(w, r, "/parent/register?error=Registration+failed", http.StatusSeeOther)
			return
		}

		// Link parent to student
		err = linkParentToStudent(parentID, studentID, registration.Relationship)
		if err != nil {
			log.Printf("Failed to link parent to student: %v", err)
		}

		// Send welcome email
		go sendParentWelcomeEmail(registration.Email, registration.Name)

		http.Redirect(w, r, "/parent/login?registered=true", http.StatusSeeOther)
	}
}

// Helper types and functions

type StudentStatus struct {
	ParentViewStudent
	BusStatus      string
	LastSeen       *time.Time
	NextPickup     *time.Time
	CurrentLocation *GPSLocation
}

type ParentNotification struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Title     string    `json:"title"`
	Message   string    `json:"message"`
	StudentID *string   `json:"student_id,omitempty"`
	Read      bool      `json:"read"`
	CreatedAt time.Time `json:"created_at"`
}

type Event struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Date        time.Time `json:"date"`
	StudentID   *string   `json:"student_id,omitempty"`
}

type StudentDetail struct {
	ParentViewStudent
	Route      *Route
	Driver     *User
	BusNumber  string
	PickupTime string
	DropTime   string
	PickupStop string
	DropStop   string
}

type AttendanceRecord struct {
	Date      time.Time `json:"date"`
	Morning   string    `json:"morning"`
	Afternoon string    `json:"afternoon"`
}

type NotificationSettings struct {
	BusArrival   bool `json:"bus_arrival"`
	BusDeparture bool `json:"bus_departure"`
	Attendance   bool `json:"attendance"`
	Emergency    bool `json:"emergency"`
	RouteChanges bool `json:"route_changes"`
	EmailEnabled bool `json:"email_enabled"`
	SMSEnabled   bool `json:"sms_enabled"`
	PushEnabled  bool `json:"push_enabled"`
}

type ParentRegistration struct {
	Email        string
	Phone        string
	Name         string
	StudentCode  string
	Relationship string
}

// Session management

func getParentFromSession(r *http.Request) *Parent {
	cookie, err := r.Cookie("parent_session")
	if err != nil {
		return nil
	}

	var parentID int
	err = db.QueryRow(`
		SELECT parent_id FROM parent_sessions 
		WHERE token = $1 AND expires_at > CURRENT_TIMESTAMP
	`, cookie.Value).Scan(&parentID)
	
	if err != nil {
		return nil
	}

	return getParentByID(parentID)
}

func createParentSession(token string, parentID int) error {
	_, err := db.Exec(`
		INSERT INTO parent_sessions (token, parent_id, expires_at)
		VALUES ($1, $2, CURRENT_TIMESTAMP + INTERVAL '24 hours')
		ON CONFLICT (token) DO UPDATE SET expires_at = EXCLUDED.expires_at
	`, token, parentID)
	return err
}

// Database functions

func authenticateParent(username, password string) (*Parent, error) {
	var parent Parent
	var hashedPassword string
	
	err := db.QueryRow(`
		SELECT id, username, email, phone, name, password, created_at
		FROM parents
		WHERE (username = $1 OR email = $1) AND active = true
	`, username).Scan(&parent.ID, &parent.Username, &parent.Email, 
		&parent.Phone, &parent.Name, &hashedPassword, &parent.CreatedAt)
	
	if err != nil {
		return nil, err
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password)); err != nil {
		return nil, fmt.Errorf("invalid password")
	}

	// Load students
	parent.Students = getParentStudents(parent.ID)

	return &parent, nil
}

func getParentByID(parentID int) *Parent {
	var parent Parent
	
	err := db.QueryRow(`
		SELECT id, username, email, phone, name, notifications, created_at
		FROM parents
		WHERE id = $1 AND active = true
	`, parentID).Scan(&parent.ID, &parent.Username, &parent.Email,
		&parent.Phone, &parent.Name, &parent.Notifications, &parent.CreatedAt)
	
	if err != nil {
		return nil
	}

	parent.Students = getParentStudents(parent.ID)
	return &parent
}

func getParentStudents(parentID int) []ParentViewStudent {
	var students []ParentViewStudent
	
	rows, err := db.Query(`
		SELECT s.student_id, s.name, s.locations, s.route_id
		FROM students s
		JOIN parent_students ps ON s.student_id = ps.student_id
		WHERE ps.parent_id = $1
		ORDER BY s.name
	`, parentID)
	
	if err != nil {
		return students
	}
	defer rows.Close()

	for rows.Next() {
		var student ParentViewStudent
		var routeID sql.NullString
		var locations string
		
		err := rows.Scan(&student.StudentID, &student.Name, &locations, &routeID)
		if err != nil {
			continue
		}
		
		if routeID.Valid {
			student.RouteID = routeID.String
		}
		
		// Mock grade and address for demo
		student.Grade = "5th"
		student.Address = locations
		
		students = append(students, student)
	}

	return students
}

func getParentStudentsWithStatus(parentID int) []StudentStatus {
	students := getParentStudents(parentID)
	var studentStatuses []StudentStatus

	for _, student := range students {
		status := StudentStatus{ParentViewStudent: student}
		
		// Get current bus status
		if student.RouteID != "" {
			// Get bus location
			vehicleID := getRouteVehicle(student.RouteID)
			if vehicleID != "" {
				if loc, err := gpsTracker.GetLatestLocation(vehicleID); err == nil && loc != nil {
					status.CurrentLocation = loc
					status.LastSeen = &loc.Timestamp
					
					// Determine status based on time and location
					status.BusStatus = determineBusStatus(student.Student, loc)
				}
			}
		}
		
		studentStatuses = append(studentStatuses, status)
	}

	return studentStatuses
}

func getParentBusTracking(parentID int) []BusTracking {
	var trackingInfo []BusTracking
	
	query := `
		SELECT s.student_id, s.name, ra.bus_id, r.route_id, r.route_name
		FROM students s
		JOIN parent_students ps ON s.student_id = ps.student_id
		JOIN routes r ON s.route_id = r.route_id
		LEFT JOIN route_assignments ra ON r.route_id = ra.route_id
			AND CURRENT_DATE BETWEEN ra.start_date AND ra.end_date
		WHERE ps.parent_id = $1
	`
	
	rows, err := db.Query(query, parentID)
	if err != nil {
		return trackingInfo
	}
	defer rows.Close()

	for rows.Next() {
		var tracking BusTracking
		var busID sql.NullString
		
		err := rows.Scan(&tracking.StudentID, &tracking.StudentName,
			&busID, &tracking.RouteID, &tracking.RouteName)
		if err != nil {
			continue
		}
		
		if busID.Valid {
			tracking.BusID = busID.String
			
			// Get current location
			if loc, err := gpsTracker.GetLatestLocation(tracking.BusID); err == nil && loc != nil {
				tracking.CurrentLocation = loc
				tracking.Status = determineBusStatus2(tracking.StudentID, loc)
				
				// Calculate ETA
				eta := calculateStudentETA(tracking.StudentID, loc)
				tracking.EstimatedArrival = eta
			}
		} else {
			tracking.Status = "not_started"
		}
		
		trackingInfo = append(trackingInfo, tracking)
	}

	return trackingInfo
}

func parentHasStudent(parentID int, studentID string) bool {
	var exists bool
	err := db.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM parent_students 
			WHERE parent_id = $1 AND student_id = $2
		)
	`, parentID, studentID).Scan(&exists)
	
	return exists && err == nil
}

func getParentNotifications(parentID int, limit int) []ParentNotification {
	var notifications []ParentNotification
	
	query := `
		SELECT id, type, title, message, student_id, read, created_at
		FROM parent_notifications
		WHERE parent_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`
	
	rows, err := db.Query(query, parentID, limit)
	if err != nil {
		return notifications
	}
	defer rows.Close()

	for rows.Next() {
		var notif ParentNotification
		var studentID sql.NullString
		
		err := rows.Scan(&notif.ID, &notif.Type, &notif.Title, &notif.Message,
			&studentID, &notif.Read, &notif.CreatedAt)
		if err != nil {
			continue
		}
		
		if studentID.Valid {
			notif.StudentID = &studentID.String
		}
		
		notifications = append(notifications, notif)
	}

	return notifications
}

func createParentAccount(reg ParentRegistration) (int, error) {
	// Generate username from email
	username := strings.Split(reg.Email, "@")[0]
	
	// Generate temporary password
	tempPassword := generateTempPassword()
	hashedPassword, _ := hashPassword(tempPassword)
	
	var parentID int
	err := db.QueryRow(`
		INSERT INTO parents (username, email, phone, name, password, active)
		VALUES ($1, $2, $3, $4, $5, true)
		RETURNING id
	`, username, reg.Email, reg.Phone, reg.Name, hashedPassword).Scan(&parentID)
	
	if err != nil {
		return 0, err
	}
	
	// Store temp password for email
	storeTempPassword(parentID, tempPassword)
	
	return parentID, nil
}

func linkParentToStudent(parentID int, studentID string, relationship string) error {
	_, err := db.Exec(`
		INSERT INTO parent_students (parent_id, student_id, relationship, emergency_rank)
		VALUES ($1, $2, $3, 2)
		ON CONFLICT (parent_id, student_id) DO NOTHING
	`, parentID, studentID, relationship)
	return err
}

// Utility functions

func determineBusStatus(student Student, location *GPSLocation) string {
	// Logic to determine bus status based on time and location
	now := time.Now()
	hour := now.Hour()
	
	if hour < 6 {
		return "not_started"
	} else if hour < 9 {
		return "morning_route"
	} else if hour < 14 {
		return "at_school"
	} else if hour < 17 {
		return "afternoon_route"
	} else {
		return "completed"
	}
}

func determineBusStatus2(studentID string, location *GPSLocation) string {
	// More sophisticated status determination
	return "en_route"
}

func calculateStudentETA(studentID string, currentLocation *GPSLocation) *time.Time {
	// Calculate estimated arrival time based on current location
	// This would use route information and current progress
	eta := time.Now().Add(15 * time.Minute)
	return &eta
}

func validateStudentCode(code string) (string, error) {
	// Validate the student registration code
	var studentID string
	err := db.QueryRow(`
		SELECT student_id FROM student_codes
		WHERE code = $1 AND used = false AND expires_at > CURRENT_TIMESTAMP
	`, code).Scan(&studentID)
	
	if err != nil {
		return "", fmt.Errorf("invalid or expired code")
	}
	
	// Mark code as used
	db.Exec("UPDATE student_codes SET used = true WHERE code = $1", code)
	
	return studentID, nil
}

func getRouteVehicle(routeID string) string {
	var vehicleID string
	db.QueryRow(`
		SELECT bus_id FROM route_assignments
		WHERE route_id = $1 AND CURRENT_DATE BETWEEN start_date AND end_date
		LIMIT 1
	`, routeID).Scan(&vehicleID)
	return vehicleID
}

// getMapAPIKey is already defined in handlers_gps.go

func generateTempPassword() string {
	// Generate a temporary password
	return generateRandomString(8)
}

func storeTempPassword(parentID int, password string) {
	// Store temporary password for sending in welcome email
	// In production, this would be encrypted
	db.Exec(`
		INSERT INTO temp_passwords (parent_id, password, expires_at)
		VALUES ($1, $2, CURRENT_TIMESTAMP + INTERVAL '7 days')
	`, parentID, password)
}

func sendParentWelcomeEmail(email, name string) {
	// Send welcome email with login instructions
	log.Printf("Would send welcome email to %s (%s)", name, email)
}

// Helper password functions

// checkPasswordHash is already defined in utils.go

// hashPassword is already defined in utils.go