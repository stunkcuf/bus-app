package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/crypto/bcrypt"
)

// MobileAPI handles mobile app endpoints
type MobileAPI struct {
	db         *sql.DB
	jwtSecret  []byte
	tokenExpiry time.Duration
}

// Mobile API models
type MobileLoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	DeviceID string `json:"device_id"`
	Platform string `json:"platform"` // ios, android
}

type MobileLoginResponse struct {
	Token        string      `json:"token"`
	RefreshToken string      `json:"refresh_token"`
	User         MobileUser  `json:"user"`
	ExpiresIn    int64       `json:"expires_in"`
}

type MobileUser struct {
	Username    string   `json:"username"`
	FullName    string   `json:"full_name"`
	Role        string   `json:"role"`
	Permissions []string `json:"permissions"`
	PhotoURL    string   `json:"photo_url,omitempty"`
}

type DriverStatus struct {
	Status      string    `json:"status"` // available, on_route, break, off_duty
	RouteID     string    `json:"route_id,omitempty"`
	LastUpdated time.Time `json:"last_updated"`
}

type RouteDetails struct {
	RouteID      string              `json:"route_id"`
	RouteName    string              `json:"route_name"`
	BusID        string              `json:"bus_id"`
	BusNumber    string              `json:"bus_number"`
	Students     []MobileStudentInfo `json:"students"`
	Stops        []RouteStop         `json:"stops"`
	StartTime    string              `json:"start_time"`
	EstimatedEnd string              `json:"estimated_end"`
}

// MobileStudentInfo for mobile API (different from import_ecse.go StudentInfo)
type MobileStudentInfo struct {
	StudentID     string `json:"student_id"`
	Name          string `json:"name"`
	Grade         string `json:"grade"`
	PickupStop    string `json:"pickup_stop"`
	DropoffStop   string `json:"dropoff_stop"`
	ContactInfo   string `json:"contact_info"`
	SpecialNeeds  string `json:"special_needs"`
}

type RouteStop struct {
	StopID       string    `json:"stop_id"`
	StopName     string    `json:"stop_name"`
	Address      string    `json:"address"`
	Latitude     float64   `json:"latitude"`
	Longitude    float64   `json:"longitude"`
	ScheduledTime string   `json:"scheduled_time"`
	StudentCount int       `json:"student_count"`
	Order        int       `json:"order"`
}

type AttendanceRecord struct {
	StudentID   string    `json:"student_id"`
	Status      string    `json:"status"` // present, absent, excused
	BoardedAt   time.Time `json:"boarded_at,omitempty"`
	DroppedAt   time.Time `json:"dropped_at,omitempty"`
	Notes       string    `json:"notes,omitempty"`
}

type LocationUpdate struct {
	Latitude  float64   `json:"latitude"`
	Longitude float64   `json:"longitude"`
	Speed     float64   `json:"speed"`
	Heading   float64   `json:"heading"`
	Accuracy  float64   `json:"accuracy"`
	Timestamp time.Time `json:"timestamp"`
}

type PreTripInspection struct {
	BusID            string                 `json:"bus_id"`
	InspectionDate   time.Time              `json:"inspection_date"`
	Mileage          int                    `json:"mileage"`
	FuelLevel        string                 `json:"fuel_level"`
	Items            []InspectionItem       `json:"items"`
	Issues           []string               `json:"issues"`
	SafeToDrive      bool                   `json:"safe_to_drive"`
	DriverSignature  string                 `json:"driver_signature"`
	Photos           []string               `json:"photos,omitempty"`
}

type InspectionItem struct {
	Category    string `json:"category"`
	Item        string `json:"item"`
	Status      string `json:"status"` // pass, fail, needs_attention
	Notes       string `json:"notes,omitempty"`
}

// JWT Claims
type MobileClaims struct {
	Username string `json:"username"`
	Role     string `json:"role"`
	DeviceID string `json:"device_id"`
	jwt.StandardClaims
}

// NewMobileAPI creates a new mobile API handler
func NewMobileAPI(db *sql.DB, jwtSecret string) *MobileAPI {
	return &MobileAPI{
		db:          db,
		jwtSecret:   []byte(jwtSecret),
		tokenExpiry: 24 * time.Hour,
	}
}

// Mobile authentication endpoint
func (api *MobileAPI) LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req MobileLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Validate credentials
	user, err := api.authenticateUser(req.Username, req.Password)
	if err != nil {
		log.Printf("Mobile login failed for %s: %v", req.Username, err)
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Generate tokens
	token, refreshToken, err := api.generateTokens(user, req.DeviceID)
	if err != nil {
		log.Printf("Token generation failed: %v", err)
		http.Error(w, "Authentication error", http.StatusInternalServerError)
		return
	}

	// Store device info
	api.storeDeviceInfo(user.Username, req.DeviceID, req.Platform)

	// Prepare response
	response := MobileLoginResponse{
		Token:        token,
		RefreshToken: refreshToken,
		User: MobileUser{
			Username: user.Username,
			FullName: user.Username, // Use username as full name since FirstName/LastName don't exist
			Role:     user.Role,
			Permissions: api.getUserPermissions(user.Role),
		},
		ExpiresIn: int64(api.tokenExpiry.Seconds()),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Get driver's current route
func (api *MobileAPI) GetCurrentRouteHandler(w http.ResponseWriter, r *http.Request) {
	username := api.getUserFromToken(r)
	if username == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get today's route assignment
	var route RouteDetails
	err := api.db.QueryRow(`
		SELECT 
			ra.route_id,
			r.route_name,
			ra.bus_id,
			b.bus_number,
			ra.start_time,
			ra.estimated_end_time
		FROM route_assignments ra
		JOIN routes r ON ra.route_id = r.route_id
		JOIN buses b ON ra.bus_id = b.bus_id
		WHERE ra.driver = $1 
		AND ra.assigned_date = CURRENT_DATE
		AND ra.status = 'active'
	`, username).Scan(
		&route.RouteID,
		&route.RouteName,
		&route.BusID,
		&route.BusNumber,
		&route.StartTime,
		&route.EstimatedEnd,
	)

	if err != nil {
		http.Error(w, "No active route found", http.StatusNotFound)
		return
	}

	// Get students on route
	studentRows, err := api.db.Query(`
		SELECT 
			s.student_id,
			s.name,
			s.grade,
			s.pickup_address,
			s.dropoff_address,
			s.contact_number,
			s.special_needs
		FROM students s
		WHERE s.route_id = $1
		AND s.active = true
		ORDER BY s.name
	`, route.RouteID)
	
	if err == nil {
		defer studentRows.Close()
		for studentRows.Next() {
			var student MobileStudentInfo
			var specialNeeds sql.NullString
			err := studentRows.Scan(
				&student.StudentID,
				&student.Name,
				&student.Grade,
				&student.PickupStop,
				&student.DropoffStop,
				&student.ContactInfo,
				&specialNeeds,
			)
			if err == nil {
				if specialNeeds.Valid {
					student.SpecialNeeds = specialNeeds.String
				}
				route.Students = append(route.Students, student)
			}
		}
	}

	// Get route stops (simulated)
	route.Stops = api.getRouteStops(route.RouteID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(route)
}

// Update driver status
func (api *MobileAPI) UpdateStatusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	username := api.getUserFromToken(r)
	if username == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var status DriverStatus
	if err := json.NewDecoder(r.Body).Decode(&status); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Update driver status
	_, err := api.db.Exec(`
		INSERT INTO driver_status (driver_username, status, route_id, updated_at)
		VALUES ($1, $2, $3, CURRENT_TIMESTAMP)
		ON CONFLICT (driver_username) 
		DO UPDATE SET status = $2, route_id = $3, updated_at = CURRENT_TIMESTAMP
	`, username, status.Status, status.RouteID)

	if err != nil {
		log.Printf("Failed to update driver status: %v", err)
		http.Error(w, "Failed to update status", http.StatusInternalServerError)
		return
	}

	// Broadcast status update
	if wsHub != nil {
		BroadcastRouteUpdate(status.RouteID, "driver_status", map[string]interface{}{
			"driver": username,
			"status": status.Status,
		})
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}

// Submit attendance
func (api *MobileAPI) SubmitAttendanceHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	username := api.getUserFromToken(r)
	if username == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var attendance []AttendanceRecord
	if err := json.NewDecoder(r.Body).Decode(&attendance); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Process attendance records
	tx, err := api.db.Begin()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	for _, record := range attendance {
		_, err := tx.Exec(`
			INSERT INTO student_attendance 
			(student_id, attendance_date, status, boarded_at, dropped_at, notes, recorded_by)
			VALUES ($1, CURRENT_DATE, $2, $3, $4, $5, $6)
			ON CONFLICT (student_id, attendance_date)
			DO UPDATE SET 
				status = $2, 
				boarded_at = $3, 
				dropped_at = $4, 
				notes = $5,
				recorded_by = $6,
				updated_at = CURRENT_TIMESTAMP
		`, record.StudentID, record.Status, record.BoardedAt, record.DroppedAt, 
		   record.Notes, username)
		
		if err != nil {
			log.Printf("Failed to record attendance: %v", err)
		}
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, "Failed to save attendance", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"recorded": len(attendance),
	})
}

// Update location
func (api *MobileAPI) UpdateLocationHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	username := api.getUserFromToken(r)
	if username == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var location LocationUpdate
	if err := json.NewDecoder(r.Body).Decode(&location); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Store location update
	_, err := api.db.Exec(`
		INSERT INTO driver_locations 
		(driver_username, latitude, longitude, speed, heading, accuracy, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (driver_username)
		DO UPDATE SET 
			latitude = $2, 
			longitude = $3, 
			speed = $4,
			heading = $5,
			accuracy = $6,
			updated_at = $7
	`, username, location.Latitude, location.Longitude, location.Speed, 
	   location.Heading, location.Accuracy, location.Timestamp)

	if err != nil {
		log.Printf("Failed to update location: %v", err)
		http.Error(w, "Failed to update location", http.StatusInternalServerError)
		return
	}

	// Broadcast location to managers
	if wsHub != nil {
		BroadcastDriverLocation(username, location.Latitude, location.Longitude)
	}

	w.WriteHeader(http.StatusOK)
}

// Submit pre-trip inspection
func (api *MobileAPI) SubmitInspectionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	username := api.getUserFromToken(r)
	if username == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var inspection PreTripInspection
	if err := json.NewDecoder(r.Body).Decode(&inspection); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Store inspection
	tx, err := api.db.Begin()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	// Insert main inspection record
	var inspectionID int
	err = tx.QueryRow(`
		INSERT INTO pre_trip_inspections 
		(bus_id, driver_username, inspection_date, mileage, fuel_level, 
		 safe_to_drive, driver_signature, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, CURRENT_TIMESTAMP)
		RETURNING inspection_id
	`, inspection.BusID, username, inspection.InspectionDate, inspection.Mileage,
	   inspection.FuelLevel, inspection.SafeToDrive, inspection.DriverSignature).Scan(&inspectionID)

	if err != nil {
		log.Printf("Failed to create inspection: %v", err)
		http.Error(w, "Failed to save inspection", http.StatusInternalServerError)
		return
	}

	// Insert inspection items
	for _, item := range inspection.Items {
		_, err := tx.Exec(`
			INSERT INTO inspection_items 
			(inspection_id, category, item, status, notes)
			VALUES ($1, $2, $3, $4, $5)
		`, inspectionID, item.Category, item.Item, item.Status, item.Notes)
		
		if err != nil {
			log.Printf("Failed to save inspection item: %v", err)
		}
	}

	// Create maintenance alerts for failed items
	if !inspection.SafeToDrive || len(inspection.Issues) > 0 {
		issueDetails := strings.Join(inspection.Issues, "; ")
		BroadcastMaintenanceAlert(
			inspection.BusID,
			"Pre-Trip Inspection Failed",
			fmt.Sprintf("Driver %s reported issues: %s", username, issueDetails),
		)
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, "Failed to save inspection", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"inspection_id": inspectionID,
	})
}

// Get driver schedule
func (api *MobileAPI) GetScheduleHandler(w http.ResponseWriter, r *http.Request) {
	username := api.getUserFromToken(r)
	if username == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get date range (default: next 7 days)
	startDate := time.Now()
	endDate := startDate.AddDate(0, 0, 7)

	if start := r.URL.Query().Get("start"); start != "" {
		if parsed, err := time.Parse("2006-01-02", start); err == nil {
			startDate = parsed
		}
	}
	if end := r.URL.Query().Get("end"); end != "" {
		if parsed, err := time.Parse("2006-01-02", end); err == nil {
			endDate = parsed
		}
	}

	// Query schedule
	rows, err := api.db.Query(`
		SELECT 
			ra.assigned_date,
			ra.route_id,
			r.route_name,
			ra.bus_id,
			b.bus_number,
			ra.start_time,
			ra.estimated_end_time,
			ra.status
		FROM route_assignments ra
		JOIN routes r ON ra.route_id = r.route_id
		JOIN buses b ON ra.bus_id = b.bus_id
		WHERE ra.driver = $1
		AND ra.assigned_date BETWEEN $2 AND $3
		ORDER BY ra.assigned_date, ra.start_time
	`, username, startDate, endDate)

	if err != nil {
		log.Printf("Failed to get schedule: %v", err)
		http.Error(w, "Failed to retrieve schedule", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	schedule := []map[string]interface{}{}
	for rows.Next() {
		var date time.Time
		var routeID, routeName, busID, busNumber, startTime, endTime, status string
		
		err := rows.Scan(&date, &routeID, &routeName, &busID, &busNumber, 
						 &startTime, &endTime, &status)
		if err != nil {
			continue
		}

		schedule = append(schedule, map[string]interface{}{
			"date":       date.Format("2006-01-02"),
			"route_id":   routeID,
			"route_name": routeName,
			"bus_id":     busID,
			"bus_number": busNumber,
			"start_time": startTime,
			"end_time":   endTime,
			"status":     status,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(schedule)
}

// Report issue
func (api *MobileAPI) ReportIssueHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	username := api.getUserFromToken(r)
	if username == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var issue struct {
		Type        string   `json:"type"`
		Description string   `json:"description"`
		VehicleID   string   `json:"vehicle_id,omitempty"`
		RouteID     string   `json:"route_id,omitempty"`
		Severity    string   `json:"severity"`
		Photos      []string `json:"photos,omitempty"`
		Location    *LocationUpdate `json:"location,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&issue); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Store issue report
	var issueID int
	err := api.db.QueryRow(`
		INSERT INTO issue_reports 
		(reported_by, type, description, vehicle_id, route_id, severity, 
		 latitude, longitude, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'open', CURRENT_TIMESTAMP)
		RETURNING issue_id
	`, username, issue.Type, issue.Description, issue.VehicleID, issue.RouteID,
	   issue.Severity, 
	   sql.NullFloat64{Float64: issue.Location.Latitude, Valid: issue.Location != nil},
	   sql.NullFloat64{Float64: issue.Location.Longitude, Valid: issue.Location != nil}).Scan(&issueID)

	if err != nil {
		log.Printf("Failed to create issue report: %v", err)
		http.Error(w, "Failed to save issue", http.StatusInternalServerError)
		return
	}

	// Broadcast alert for high severity issues
	if issue.Severity == "high" || issue.Type == "safety" {
		BroadcastMaintenanceAlert(
			issue.VehicleID,
			fmt.Sprintf("Driver Issue Report - %s", issue.Type),
			fmt.Sprintf("%s reported by %s", issue.Description, username),
		)
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"issue_id": issueID,
	})
}

// Helper functions

func (api *MobileAPI) authenticateUser(username, password string) (*User, error) {
	var user User
	err := api.db.QueryRow(`
		SELECT username, password, role, status
		FROM users
		WHERE username = $1
	`, username).Scan(&user.Username, &user.Password, &user.Role, &user.Status)

	if err != nil {
		return nil, err
	}

	if user.Status != "active" {
		return nil, fmt.Errorf("account disabled")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, fmt.Errorf("invalid password")
	}

	return &user, nil
}

func (api *MobileAPI) generateTokens(user *User, deviceID string) (string, string, error) {
	// Access token
	claims := MobileClaims{
		Username: user.Username,
		Role:     user.Role,
		DeviceID: deviceID,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(api.tokenExpiry).Unix(),
			IssuedAt:  time.Now().Unix(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(api.jwtSecret)
	if err != nil {
		return "", "", err
	}

	// Refresh token (longer expiry)
	refreshClaims := claims
	refreshClaims.ExpiresAt = time.Now().Add(30 * 24 * time.Hour).Unix()
	
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshString, err := refreshToken.SignedString(api.jwtSecret)
	if err != nil {
		return "", "", err
	}

	return tokenString, refreshString, nil
}

func (api *MobileAPI) getUserFromToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return ""
	}

	token, err := jwt.ParseWithClaims(parts[1], &MobileClaims{}, func(token *jwt.Token) (interface{}, error) {
		return api.jwtSecret, nil
	})

	if err != nil || !token.Valid {
		return ""
	}

	if claims, ok := token.Claims.(*MobileClaims); ok {
		return claims.Username
	}

	return ""
}

func (api *MobileAPI) storeDeviceInfo(username, deviceID, platform string) {
	_, err := api.db.Exec(`
		INSERT INTO user_devices (username, device_id, platform, last_seen)
		VALUES ($1, $2, $3, CURRENT_TIMESTAMP)
		ON CONFLICT (username, device_id) 
		DO UPDATE SET platform = $3, last_seen = CURRENT_TIMESTAMP
	`, username, deviceID, platform)
	
	if err != nil {
		log.Printf("Failed to store device info: %v", err)
	}
}

func (api *MobileAPI) getUserPermissions(role string) []string {
	permissions := map[string][]string{
		"driver": {
			"view_route",
			"update_attendance",
			"submit_inspection",
			"report_issue",
			"update_location",
		},
		"manager": {
			"view_all_routes",
			"view_all_drivers",
			"view_analytics",
			"manage_schedule",
			"approve_issues",
		},
	}

	return permissions[role]
}

func (api *MobileAPI) getRouteStops(routeID string) []RouteStop {
	// This would fetch actual route stops from database
	// For now, return simulated data
	return []RouteStop{
		{
			StopID:       "STOP-001",
			StopName:     "Main Street & 1st Ave",
			Address:      "100 Main Street",
			Latitude:     40.7128,
			Longitude:    -74.0060,
			ScheduledTime: "07:00 AM",
			StudentCount: 5,
			Order:        1,
		},
		{
			StopID:       "STOP-002",
			StopName:     "Oak Drive & 2nd Street",
			Address:      "200 Oak Drive",
			Latitude:     40.7180,
			Longitude:    -74.0100,
			ScheduledTime: "07:10 AM",
			StudentCount: 3,
			Order:        2,
		},
	}
}

// Register mobile API routes
func RegisterMobileAPIRoutes(mux *http.ServeMux, api *MobileAPI) {
	// Authentication
	mux.HandleFunc("/api/mobile/v1/login", api.LoginHandler)
	
	// Driver endpoints
	mux.HandleFunc("/api/mobile/v1/driver/route", api.GetCurrentRouteHandler)
	mux.HandleFunc("/api/mobile/v1/driver/status", api.UpdateStatusHandler)
	mux.HandleFunc("/api/mobile/v1/driver/attendance", api.SubmitAttendanceHandler)
	mux.HandleFunc("/api/mobile/v1/driver/location", api.UpdateLocationHandler)
	mux.HandleFunc("/api/mobile/v1/driver/inspection", api.SubmitInspectionHandler)
	mux.HandleFunc("/api/mobile/v1/driver/schedule", api.GetScheduleHandler)
	mux.HandleFunc("/api/mobile/v1/driver/issue", api.ReportIssueHandler)
}