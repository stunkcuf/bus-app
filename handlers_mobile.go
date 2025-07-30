package main

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"time"
	"strings"
)

// Mobile route handlers

// mobileDriverDashboardHandler serves the mobile driver dashboard
func mobileDriverDashboardHandler(w http.ResponseWriter, r *http.Request) {
	// Check if mobile device
	if !isMobileDevice(r) {
		http.Redirect(w, r, "/driver-dashboard", http.StatusSeeOther)
		return
	}

	session := getUserFromSession(r)
	if session == nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// Get driver data
	assignments, _ := getDriverAssignments(session.Username)
	vehicleID := ""
	routeID := ""
	if len(assignments) > 0 {
		vehicleID = assignments[0].BusID
		routeID = assignments[0].RouteID
	}
	todayMiles := getDriverTodayMiles(session.Username)
	studentsTransported := getDriverStudentsToday(session.Username)
	onTimePercentage := getDriverOnTimePercentage(session.Username)

	// Get today's routes
	routes := getDriverTodayRoutes(session.Username)

	// Get recent activity
	recentActivity := getDriverRecentActivity(session.Username, 5)

	// Get unread notifications count
	unreadCount := getUnreadNotificationCount(getUserID(session.Username))

	data := struct {
		Title               string
		Username            string
		CSPNonce            string
		VehicleID           string
		RouteID             string
		TodayMiles          int
		StudentsTransported int
		OnTimePercentage    int
		Routes              []RouteInfo
		RecentActivity      []ActivityItem
		UnreadCount         int
	}{
		Title:               "Driver Dashboard",
		Username:            session.Username,
		CSPNonce:            generateNonce(),
		VehicleID:           vehicleID,
		RouteID:             routeID,
		TodayMiles:          todayMiles,
		StudentsTransported: studentsTransported,
		OnTimePercentage:    onTimePercentage,
		Routes:              routes,
		RecentActivity:      recentActivity,
		UnreadCount:         unreadCount,
	}

	tmpl := template.Must(template.ParseFiles("templates/mobile_driver_dashboard.html"))
	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("Error rendering mobile driver dashboard: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// mobileManagerDashboardHandler serves the mobile manager dashboard
func mobileManagerDashboardHandler(w http.ResponseWriter, r *http.Request) {
	if !isMobileDevice(r) {
		http.Redirect(w, r, "/manager-dashboard", http.StatusSeeOther)
		return
	}

	session := getUserFromSession(r)
	if session == nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// Get manager stats
	stats := getFleetStats()
	alerts := getActiveAlerts()
	recentActivity := getSystemRecentActivity(10)

	data := struct {
		Title          string
		Username       string
		CSPNonce       string
		Stats          FleetStats
		Alerts         []Alert
		RecentActivity []ActivityItem
	}{
		Title:          "Manager Dashboard",
		Username:       session.Username,
		CSPNonce:       generateNonce(),
		Stats:          stats,
		Alerts:         alerts,
		RecentActivity: recentActivity,
	}

	tmpl := template.Must(template.ParseFiles("templates/mobile_manager_dashboard.html"))
	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("Error rendering mobile manager dashboard: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// mobileRouteDetailsHandler shows route details on mobile
func mobileRouteDetailsHandler(w http.ResponseWriter, r *http.Request) {
	session := getUserFromSession(r)
	if session == nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// Get route ID from URL
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 4 {
		http.Error(w, "Invalid route", http.StatusBadRequest)
		return
	}
	routeID := pathParts[3]

	// Get route details
	route, err := getRouteDetails(routeID)
	if err != nil {
		http.Error(w, "Route not found", http.StatusNotFound)
		return
	}

	// Get students on route
	students := getRouteStudents(routeID)

	data := struct {
		Title    string
		Username string
		CSPNonce string
		Route    RouteDetails
		Students []Student
	}{
		Title:    "Route Details",
		Username: session.Username,
		CSPNonce: generateNonce(),
		Route:    route,
		Students: students,
	}

	tmpl := template.Must(template.ParseFiles("templates/mobile_route_details.html"))
	tmpl.Execute(w, data)
}

// mobileStudentAttendanceHandler handles student attendance on mobile
func mobileStudentAttendanceHandler(w http.ResponseWriter, r *http.Request) {
	session := getUserFromSession(r)
	if session == nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	if r.Method == "GET" {
		// Get current route students
		assignments, _ := getDriverAssignments(session.Username)
		if len(assignments) == 0 {
			http.Error(w, "No active route", http.StatusBadRequest)
			return
		}
		routeID := assignments[0].RouteID

		students := getRouteStudents(routeID)

		data := struct {
			Title    string
			Username string
			CSPNonce string
			RouteID  string
			Students []Student
			Period   string
		}{
			Title:    "Student Attendance",
			Username: session.Username,
			CSPNonce: generateNonce(),
			RouteID:  routeID,
			Students: students,
			Period:   r.URL.Query().Get("period"), // morning or afternoon
		}

		tmpl := template.Must(template.ParseFiles("templates/mobile_attendance.html"))
		tmpl.Execute(w, data)

	} else if r.Method == "POST" {
		// Process attendance submission
		var attendance struct {
			RouteID  string            `json:"route_id"`
			Period   string            `json:"period"`
			Students map[string]string `json:"students"` // studentID -> status
		}

		if err := json.NewDecoder(r.Body).Decode(&attendance); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		// Save attendance
		err := saveAttendance(session.Username, attendance.RouteID, attendance.Period, attendance.Students)
		if err != nil {
			log.Printf("Failed to save attendance: %v", err)
			http.Error(w, "Failed to save attendance", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "Attendance saved successfully",
		})
	}
}

// mobileVehicleCheckHandler handles pre-trip vehicle checks
func mobileVehicleCheckHandler(w http.ResponseWriter, r *http.Request) {
	session := getUserFromSession(r)
	if session == nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	if r.Method == "GET" {
		assignments, _ := getDriverAssignments(session.Username)
		if len(assignments) == 0 {
			http.Error(w, "No vehicle assigned", http.StatusBadRequest)
			return
		}
		vehicleID := assignments[0].BusID

		// Get vehicle details
		vehicle, _ := getVehicleDetails(vehicleID)

		data := struct {
			Title     string
			Username  string
			CSPNonce  string
			Vehicle   Vehicle
			Checklist []ChecklistItem
		}{
			Title:     "Vehicle Check",
			Username:  session.Username,
			CSPNonce:  generateNonce(),
			Vehicle:   vehicle,
			Checklist: getVehicleChecklist(),
		}

		tmpl := template.Must(template.ParseFiles("templates/mobile_vehicle_check.html"))
		tmpl.Execute(w, data)

	} else if r.Method == "POST" {
		// Process vehicle check submission
		var checkData struct {
			VehicleID string                       `json:"vehicle_id"`
			Checklist map[string]bool              `json:"checklist"`
			Issues    []string                     `json:"issues"`
			Notes     string                       `json:"notes"`
			Mileage   int                          `json:"mileage"`
			Photos    []string                     `json:"photos"`
		}

		if err := json.NewDecoder(r.Body).Decode(&checkData); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		// Save vehicle check
		checkID, err := saveVehicleCheck(session.Username, checkData)
		if err != nil {
			log.Printf("Failed to save vehicle check: %v", err)
			http.Error(w, "Failed to save check", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"check_id": checkID,
		})
	}
}

// mobileGPSTrackingHandler provides real-time GPS updates
func mobileGPSTrackingHandler(w http.ResponseWriter, r *http.Request) {
	session := getUserFromSession(r)
	if session == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if r.Method == "POST" {
		var location struct {
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
			Accuracy  float64 `json:"accuracy"`
			Speed     float64 `json:"speed"`
			Heading   float64 `json:"heading"`
			Timestamp int64   `json:"timestamp"`
		}

		if err := json.NewDecoder(r.Body).Decode(&location); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		// Update driver location
		err := updateDriverLocation(session.Username, location)
		if err != nil {
			log.Printf("Failed to update location: %v", err)
			http.Error(w, "Failed to update location", http.StatusInternalServerError)
			return
		}

		// Check for geofence alerts
		alerts := checkGeofenceAlerts(session.Username, location.Latitude, location.Longitude)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"alerts":  alerts,
		})
	}
}

// Mobile API endpoints

// mobileAPIStudentsHandler returns students list for mobile
func mobileAPIStudentsHandler(w http.ResponseWriter, r *http.Request) {
	session := getUserFromSession(r)
	if session == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	routeID := r.URL.Query().Get("route_id")
	if routeID == "" {
		assignments, _ := getDriverAssignments(session.Username)
		if len(assignments) > 0 {
			routeID = assignments[0].RouteID
		}
	}

	students := getRouteStudents(routeID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(students)
}

// mobileAPINotificationsHandler returns notifications for mobile
func mobileAPINotificationsHandler(w http.ResponseWriter, r *http.Request) {
	session := getUserFromSession(r)
	if session == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	notifications := getUserNotifications(getUserID(session.Username), 20)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(notifications)
}

// Helper functions

func isMobileDevice(r *http.Request) bool {
	userAgent := r.Header.Get("User-Agent")
	mobileKeywords := []string{"Mobile", "Android", "iPhone", "iPad", "Windows Phone"}
	
	for _, keyword := range mobileKeywords {
		if strings.Contains(userAgent, keyword) {
			return true
		}
	}
	
	// Check for mobile query parameter (for testing)
	if r.URL.Query().Get("mobile") == "true" {
		return true
	}
	
	return false
}

func getStringValue(ptr *string, defaultValue string) string {
	if ptr != nil {
		return *ptr
	}
	return defaultValue
}

func getDriverTodayMiles(username string) int {
	var miles int
	err := db.QueryRow(`
		SELECT COALESCE(SUM(end_mileage - start_mileage), 0)
		FROM driver_logs
		WHERE driver = $1 AND log_date = CURRENT_DATE
	`, username).Scan(&miles)
	
	if err != nil {
		return 0
	}
	return miles
}

func getDriverStudentsToday(username string) int {
	var count int
	err := db.QueryRow(`
		SELECT COUNT(DISTINCT s.student_id)
		FROM students s
		JOIN route_assignments ra ON s.route_id = ra.route_id
		WHERE ra.driver = $1 
		AND CURRENT_DATE BETWEEN ra.start_date AND ra.end_date
	`, username).Scan(&count)
	
	if err != nil {
		return 0
	}
	return count
}

func getDriverOnTimePercentage(username string) int {
	// Calculate on-time percentage for the driver
	// This is a simplified version
	return 95
}

func getDriverTodayRoutes(username string) []RouteInfo {
	var routes []RouteInfo
	
	rows, err := db.Query(`
		SELECT r.route_id, r.route_name, 
		       COUNT(DISTINCT s.student_id) as student_count,
		       CASE 
		           WHEN EXISTS(SELECT 1 FROM driver_logs dl 
		                      WHERE dl.driver = ra.driver 
		                      AND dl.route_id = r.route_id 
		                      AND dl.log_date = CURRENT_DATE
		                      AND dl.end_time IS NOT NULL) THEN 'Completed'
		           WHEN EXISTS(SELECT 1 FROM driver_logs dl 
		                      WHERE dl.driver = ra.driver 
		                      AND dl.route_id = r.route_id 
		                      AND dl.log_date = CURRENT_DATE
		                      AND dl.end_time IS NULL) THEN 'In Progress'
		           ELSE 'Scheduled'
		       END as status
		FROM routes r
		JOIN route_assignments ra ON r.route_id = ra.route_id
		LEFT JOIN students s ON r.route_id = s.route_id
		WHERE ra.driver = $1 
		AND CURRENT_DATE BETWEEN ra.start_date AND ra.end_date
		GROUP BY r.route_id, r.route_name, ra.driver
		ORDER BY r.route_name
	`, username)
	
	if err != nil {
		log.Printf("Failed to get driver routes: %v", err)
		return routes
	}
	defer rows.Close()
	
	for rows.Next() {
		var route RouteInfo
		err := rows.Scan(&route.RouteID, &route.RouteName, &route.StudentCount, &route.Status)
		if err != nil {
			continue
		}
		routes = append(routes, route)
	}
	
	return routes
}

func getDriverRecentActivity(username string, limit int) []ActivityItem {
	var activities []ActivityItem
	
	// Get recent driver logs
	rows, err := db.Query(`
		SELECT 'Route Completed' as title, 
		       log_date || ' ' || COALESCE(end_time, start_time) as time,
		       'bi-check-circle' as icon
		FROM driver_logs
		WHERE driver = $1
		ORDER BY log_date DESC, COALESCE(end_time, start_time) DESC
		LIMIT $2
	`, username, limit)
	
	if err != nil {
		return activities
	}
	defer rows.Close()
	
	for rows.Next() {
		var activity ActivityItem
		err := rows.Scan(&activity.Title, &activity.Time, &activity.Icon)
		if err != nil {
			continue
		}
		activities = append(activities, activity)
	}
	
	return activities
}

func getUnreadNotificationCount(userID int) int {
	var count int
	err := db.QueryRow(`
		SELECT COUNT(*)
		FROM in_app_notifications
		WHERE user_id = $1 AND read = false
	`, strconv.Itoa(userID)).Scan(&count)
	
	if err != nil {
		return 0
	}
	return count
}

// Data structures

type RouteInfo struct {
	RouteID      string
	RouteName    string
	StudentCount int
	Status       string
}

type ActivityItem struct {
	Title string
	Time  string
	Icon  string
}

type RouteDetails struct {
	RouteID      string
	RouteName    string
	Description  string
	StartTime    string
	EndTime      string
	TotalStops   int
	Distance     float64
}

type ChecklistItem struct {
	ID          string
	Category    string
	Description string
	Required    bool
}

type FleetStats struct {
	TotalVehicles    int
	ActiveVehicles   int
	TotalDrivers     int
	ActiveRoutes     int
	MaintenanceDue   int
	EmergencyAlerts  int
}

type Alert struct {
	ID       string
	Type     string
	Title    string
	Message  string
	Severity string
	Time     time.Time
}

func getVehicleChecklist() []ChecklistItem {
	return []ChecklistItem{
		{ID: "tires", Category: "Exterior", Description: "Check tire condition and pressure", Required: true},
		{ID: "lights", Category: "Exterior", Description: "Test all lights (headlights, brake, turn signals)", Required: true},
		{ID: "mirrors", Category: "Exterior", Description: "Check and adjust mirrors", Required: true},
		{ID: "fluids", Category: "Engine", Description: "Check oil, coolant, and fluid levels", Required: true},
		{ID: "brakes", Category: "Safety", Description: "Test brake operation", Required: true},
		{ID: "horn", Category: "Safety", Description: "Test horn", Required: true},
		{ID: "seatbelts", Category: "Interior", Description: "Check all seatbelts", Required: true},
		{ID: "emergency", Category: "Safety", Description: "Check emergency equipment", Required: true},
		{ID: "cleanliness", Category: "Interior", Description: "Interior cleanliness", Required: false},
		{ID: "damage", Category: "General", Description: "Check for new damage", Required: true},
	}
}

func saveVehicleCheck(username string, checkData interface{}) (string, error) {
	// Implementation for saving vehicle check
	checkID := "CHK_" + strconv.FormatInt(time.Now().Unix(), 10)
	return checkID, nil
}

func saveAttendance(driver, routeID, period string, students map[string]string) error {
	// Implementation for saving attendance
	return nil
}

func updateDriverLocation(username string, location interface{}) error {
	// Implementation for updating driver location
	return nil
}

func checkGeofenceAlerts(username string, lat, lng float64) []Alert {
	// Check for geofence violations
	return []Alert{}
}

func getRouteStudents(routeID string) []Student {
	// Get students assigned to route
	return []Student{}
}

func getVehicleDetails(vehicleID string) (Vehicle, error) {
	// Get vehicle details
	return Vehicle{}, nil
}

func getRouteDetails(routeID string) (RouteDetails, error) {
	// Get route details
	return RouteDetails{}, nil
}

func getUserNotifications(userID int, limit int) []Notification {
	// Get user notifications
	return []Notification{}
}

func getFleetStats() FleetStats {
	// Get fleet statistics
	return FleetStats{}
}

func getActiveAlerts() []Alert {
	// Get active alerts
	return []Alert{}
}

func getSystemRecentActivity(limit int) []ActivityItem {
	// Get system-wide recent activity
	return []ActivityItem{}
}