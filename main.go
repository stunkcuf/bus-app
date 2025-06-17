package main

import (
	"crypto/subtle"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

//go:embed templates/*.html
var tmplFS embed.FS

var templates *template.Template

func init() {
	var err error

	funcMap := template.FuncMap{
		"json": func(v interface{}) template.JS {
			b, err := json.Marshal(v)
			if err != nil {
				log.Printf("JSON marshal error: %v", err)
				return template.JS("{}")
			}
			return template.JS(b)
		},
		"add": func(a, b int) int {
			return a + b
		},
		"len": func(v interface{}) int {
			switch s := v.(type) {
			case []interface{}:
				return len(s)
			case []*Bus:
				return len(s)
			case []Bus:
				return len(s)
			default:
				return 0
			}
		},
	}

	templates, err = template.New("").Funcs(funcMap).ParseFS(tmplFS, "templates/*.html")
	if err != nil {
		log.Fatalf("Template parsing failed: %v", err)
	}

	log.Println("Templates loaded successfully")
}

// =============================================================================
// SECURE SESSION MANAGEMENT
// =============================================================================

type SessionManager struct {
	sessions map[string]*SessionData
	mu       sync.RWMutex
}

var sessionMgr = &SessionManager{
	sessions: make(map[string]*SessionData),
}

type SessionData struct {
	Username  string
	Role      string
	CSRFToken string
	ExpiresAt time.Time
}

var sessionMgr = &SessionManager{
	sessions: make(map[string]*SessionData),
}

// =============================================================================
// HTTP HANDLERS
// =============================================================================

func loginPage(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		r.ParseForm()
		username := SanitizeInput(r.FormValue("username"))
		password := r.FormValue("password")

		// Validate username format
		if !ValidateUsername(username) {
			http.Error(w, "Invalid username format", http.StatusBadRequest)
			return
		}

		// Check credentials
		for _, u := range loadUsers() {
			if u.Username == username {
				// Check password hash
				if err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password)); err == nil {
					// Create secure session
					sessionID, csrfToken, err := CreateSecureSession(username, u.Role)
					if err != nil {
						http.Error(w, "Session creation failed", http.StatusInternalServerError)
						return
					}

					// Store session data
					sessionMgr.mu.Lock()
					sessionMgr.sessions[sessionID] = &SessionData{
						Username:  username,
						Role:      u.Role,
						CSRFToken: csrfToken,
						ExpiresAt: time.Now().Add(24 * time.Hour),
					}
					sessionMgr.mu.Unlock()

					// Set secure cookie
					SetSecureCookie(w, "session_id", sessionID)
					SetSecureCookie(w, "csrf_token", csrfToken)

					// Redirect based on role
					if u.Role == "manager" {
						http.Redirect(w, r, "/manager-dashboard", http.StatusFound)
					} else {
						http.Redirect(w, r, "/driver-dashboard", http.StatusFound)
					}
					return
				}
			}
		}
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}
	executeTemplate(w, "login.html", nil)
}

func newUserPage(w http.ResponseWriter, r *http.Request) {
	user := getSecureUser(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	if r.Method == http.MethodPost {
		// Verify CSRF token
		if !verifyCSRFToken(r) {
			http.Error(w, "Invalid CSRF token", http.StatusForbidden)
			return
		}

		r.ParseForm()
		username := SanitizeInput(r.FormValue("username"))
		password := r.FormValue("password")
		role := SanitizeInput(r.FormValue("role"))

		// Validate inputs
		if !ValidateUsername(username) {
			http.Error(w, "Invalid username format", http.StatusBadRequest)
			return
		}

		if len(password) < 6 {
			http.Error(w, "Password must be at least 6 characters", http.StatusBadRequest)
			return
		}

		if role != "driver" && role != "manager" {
			http.Error(w, "Invalid role", http.StatusBadRequest)
			return
		}

		// Hash password
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 12)
		if err != nil {
			log.Printf("Error hashing password: %v", err)
			http.Error(w, "Unable to create user", http.StatusInternalServerError)
			return
		}

		newUser := User{
			Username: username,
			Password: string(hashedPassword),
			Role:     role,
		}

		if err := saveUser(newUser); err != nil {
			log.Printf("Error saving user: %v", err)
			http.Error(w, "Unable to save user", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/manager-dashboard", http.StatusFound)
		return
	}

	executeTemplate(w, "new_user.html", nil)
}

func editUserPage(w http.ResponseWriter, r *http.Request) {
	user := getSecureUser(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	username := r.URL.Query().Get("username")
	if username == "" {
		http.Error(w, "Username required", http.StatusBadRequest)
		return
	}

	if r.Method == http.MethodPost {
		// Verify CSRF token
		if !verifyCSRFToken(r) {
			http.Error(w, "Invalid CSRF token", http.StatusForbidden)
			return
		}

		r.ParseForm()
		newPassword := r.FormValue("password")
		newRole := SanitizeInput(r.FormValue("role"))

		if len(newPassword) < 6 {
			http.Error(w, "Password must be at least 6 characters", http.StatusBadRequest)
			return
		}

		if newRole != "driver" && newRole != "manager" {
			http.Error(w, "Invalid role", http.StatusBadRequest)
			return
		}

		// Hash new password
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), 12)
		if err != nil {
			http.Error(w, "Failed to hash password", http.StatusInternalServerError)
			return
		}

		users := loadUsers()
		for i, u := range users {
			if u.Username == username {
				users[i].Password = string(hashedPassword)
				users[i].Role = newRole
				break
			}
		}

		if err := saveUsers(users); err != nil {
			http.Error(w, "Failed to save user", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/manager-dashboard", http.StatusFound)
		return
	}

	// Find user to edit
	users := loadUsers()
	var editUser *User
	for _, u := range users {
		if u.Username == username {
			editUser = &u
			break
		}
	}

	if editUser == nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	executeTemplate(w, "edit_user.html", editUser)
}

func managerDashboard(w http.ResponseWriter, r *http.Request) {
	user := getSecureUser(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	// Load all data
	driverLogs, _ := loadDriverLogs()
	activities, _ := loadActivities()
	users := loadUsers()
	routes, _ := loadRoutes()
	buses := loadBuses()
	assignments, _ := loadRouteAssignments()

	// Initialize data structures
	driverData := make(map[string]*DriverSummary)
	routeData := make(map[string]*RouteStats)
	now := time.Now()

	// Pre-populate all known drivers
	for _, u := range users {
		if u.Role == "driver" {
			driverData[u.Username] = &DriverSummary{Name: u.Username}
		}
	}

	// Pre-populate all routes
	for _, r := range routes {
		routeData[r.RouteName] = &RouteStats{RouteName: r.RouteName}
	}

	// Process driver logs
	for _, driverLog := range driverLogs {
		// Get or create driver summary
		s := driverData[driverLog.Driver]
		if s == nil {
			s = &DriverSummary{Name: driverLog.Driver}
			driverData[driverLog.Driver] = s
		}

		// Add mileage
		s.TotalMiles += driverLog.Mileage

		// Calculate attendance from log
		presentCount := 0
		for _, att := range driverLog.Attendance {
			if att.Present {
				presentCount++
			}
		}

		// Add to morning/evening totals based on period
		if driverLog.Period == "morning" {
			s.TotalMorning += presentCount
		} else if driverLog.Period == "evening" || driverLog.Period == "afternoon" {
			s.TotalEvening += presentCount
		}

		// Parse date for time-based calculations
		parsed, err := time.Parse("2006-01-02", driverLog.Date)
		if err == nil {
			// Monthly calculations
			if parsed.Month() == now.Month() && parsed.Year() == now.Year() {
				s.MonthlyAttendance += presentCount
				s.MonthlyAvgMiles += driverLog.Mileage
			}

			// Find route name for this log
			var routeName string

			// First try to match by RouteID directly
			for _, r := range routes {
				if r.RouteID == driverLog.RouteID {
					routeName = r.RouteName
					break
				}
			}

			// If not found, try to get from driver's assignment
			if routeName == "" {
				for _, assignment := range assignments {
					if assignment.Driver == driverLog.Driver {
						routeName = assignment.RouteName
						break
					}
				}
			}

			// Update route statistics if we found a route
			if routeName != "" {
				route := routeData[routeName]
				if route == nil {
					route = &RouteStats{RouteName: routeName}
					routeData[routeName] = route
				}

				route.TotalMiles += driverLog.Mileage
				route.AttendanceMonth += presentCount

				// Time-based attendance (last 24 hours, last 7 days)
				if now.Sub(parsed).Hours() < 24 {
					route.AttendanceDay += presentCount
				}
				if now.Sub(parsed).Hours() < 168 { // 7 days
					route.AttendanceWeek += presentCount
				}
			}
		}
	}

	// Calculate averages for drivers
	for _, s := range driverData {
		if s.MonthlyAvgMiles > 0 {
			daysInMonth := float64(now.Day())
			if daysInMonth > 0 {
				s.MonthlyAvgMiles = s.MonthlyAvgMiles / daysInMonth
			}
		}
	}

	// Calculate averages for routes
	for _, r := range routeData {
		if r.TotalMiles > 0 {
			// Count logs for this route to calculate average
			logCount := 0
			for _, driverLog := range driverLogs {
				// Find route name for this log (same logic as above)
				var logRouteName string
				for _, route := range routes {
					if route.RouteID == driverLog.RouteID {
						logRouteName = route.RouteName
						break
					}
				}
				if logRouteName == "" {
					for _, assignment := range assignments {
						if assignment.Driver == driverLog.Driver {
							logRouteName = assignment.RouteName
							break
						}
					}
				}
				if logRouteName == r.RouteName {
					logCount++
				}
			}
			if logCount > 0 {
				r.AvgMiles = r.TotalMiles / float64(logCount)
			}
		}
	}

	// Convert maps to slices for template
	driverSummaries := []*DriverSummary{}
	for _, v := range driverData {
		driverSummaries = append(driverSummaries, v)
	}

	routeStats := []*RouteStats{}
	for _, v := range routeData {
		routeStats = append(routeStats, v)
	}

	// Get CSRF token for template
	csrfToken := getCSRFToken(r)

	data := DashboardData{
		User:            user,
		Role:            user.Role,
		DriverSummaries: driverSummaries,
		RouteStats:      routeStats,
		Activities:      activities,
		Routes:          routes,
		Users:           users,
		Buses:           buses,
		CSRFToken:       csrfToken,
	}

	executeTemplate(w, "dashboard.html", data)
}

func driverDashboard(w http.ResponseWriter, r *http.Request) {
	user := getSecureUser(r)
	if user == nil || user.Role != "driver" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	date := r.URL.Query().Get("date")
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}
	period := r.URL.Query().Get("period")
	if period == "" {
		period = "morning"
	}

	// Validate date format
	if !ValidateDate(date) {
		http.Error(w, "Invalid date format", http.StatusBadRequest)
		return
	}

	routes, _ := loadRoutes()
	logs, _ := loadDriverLogs()

	// Find current log for this date/period
	var driverLog *DriverLog
	for _, logEntry := range logs {
		if logEntry.Driver == user.Username && logEntry.Date == date && logEntry.Period == period {
			driverLog = &logEntry
			break
		}
	}

	// Get recent logs for this driver (last 5)
	var recentLogs []DriverLog
	count := 0
	for i := len(logs) - 1; i >= 0 && count < 5; i-- {
		if logs[i].Driver == user.Username {
			recentLogs = append(recentLogs, logs[i])
			count++
		}
	}

	var driverRoute *Route
	var assignedBus *Bus

	// Get the driver's current assignment
	assignment, err := getDriverRouteAssignment(user.Username)
	if err != nil {
		log.Printf("Warning: No assignment found for driver %s: %v", user.Username, err)
	}

	// Load all buses
	buses := loadBuses()

	// Find the route and bus based on assignment or existing log
	if assignment != nil {
		// Use assignment data (preferred)
		for _, r := range routes {
			if r.RouteID == assignment.RouteID || r.RouteName == assignment.RouteName {
				driverRoute = &r
				break
			}
		}

		for _, b := range buses {
			if b.BusID == assignment.BusID {
				assignedBus = b
				break
			}
		}
	} else if driverLog != nil {
		// Fall back to log data if no assignment
		for _, r := range routes {
			if r.RouteID == driverLog.RouteID {
				driverRoute = &r
				break
			}
		}

		for _, b := range buses {
			if b.BusID == driverLog.BusID {
				assignedBus = b
				break
			}
		}
	}

	// Load students and filter for this driver's active students on this route
	students := loadStudents()
	var activeStudentPositions []struct {
		Position int    `json:"position"`
		Student  string `json:"student"`
	}

	if driverRoute != nil {
		// Create a map of active students for this driver and route
		activeStudentMap := make(map[int]string)
		for _, student := range students {
			if student.Active && student.Driver == user.Username &&
				(student.RouteID == driverRoute.RouteID || (assignment != nil && student.RouteID == assignment.RouteID)) {
				activeStudentMap[student.PositionNumber] = student.Name
			}
		}

		// Build positions based on active students
		// Get all position numbers and sort them
		positions := make([]int, 0, len(activeStudentMap))
		for pos := range activeStudentMap {
			positions = append(positions, pos)
		}
		sort.Ints(positions)

		// Create the positions slice
		for _, pos := range positions {
			activeStudentPositions = append(activeStudentPositions, struct {
				Position int    `json:"position"`
				Student  string `json:"student"`
			}{
				Position: pos,
				Student:  activeStudentMap[pos],
			})
		}

		// Update the route with filtered positions
		if len(activeStudentPositions) > 0 {
			filteredRoute := *driverRoute
			filteredRoute.Positions = activeStudentPositions
			driverRoute = &filteredRoute
		} else {
			// If no active students, create empty route with same metadata
			filteredRoute := *driverRoute
			filteredRoute.Positions = []struct {
				Position int    `json:"position"`
				Student  string `json:"student"`
			}{}
			driverRoute = &filteredRoute
		}
	}

	// Get CSRF token for template
	csrfToken := getCSRFToken(r)

	data := DriverDashboardData{
		User:       user,
		Date:       date,
		Period:     period,
		Route:      driverRoute,
		DriverLog:  driverLog,
		Bus:        assignedBus,
		RecentLogs: recentLogs,
		CSRFToken:  csrfToken,
	}

	executeTemplate(w, "driver_dashboard.html", data)
}

func saveDriverLogHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	user := getSecureUser(r)
	if user == nil || user.Role != "driver" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	// Verify CSRF token
	if !verifyCSRFToken(r) {
		http.Error(w, "Invalid CSRF token", http.StatusForbidden)
		return
	}

	r.ParseForm()
	date := SanitizeInput(r.FormValue("date"))
	period := SanitizeInput(r.FormValue("period"))
	busID := SanitizeInput(r.FormValue("bus_id"))
	departure := SanitizeInput(r.FormValue("departure"))
	arrival := SanitizeInput(r.FormValue("arrival"))
	mileage, _ := strconv.ParseFloat(r.FormValue("mileage"), 64)

	// Validate inputs
	if !ValidateDate(date) {
		http.Error(w, "Invalid date format", http.StatusBadRequest)
		return
	}

	if period != "morning" && period != "afternoon" && period != "evening" {
		http.Error(w, "Invalid period", http.StatusBadRequest)
		return
	}

	// Get the driver's route assignment
	assignment, err := getDriverRouteAssignment(user.Username)
	if err != nil {
		log.Printf("Error getting driver assignment: %v", err)
		http.Error(w, "No route assignment found", http.StatusBadRequest)
		return
	}

	// Validate that the bus ID matches the assignment
	if busID != assignment.BusID {
		log.Printf("Bus ID mismatch: form=%s, assignment=%s", busID, assignment.BusID)
		http.Error(w, "Bus ID does not match assignment", http.StatusBadRequest)
		return
	}

	// Load route to get positions
	routes, err := loadRoutes()
	if err != nil {
		log.Printf("Error loading routes: %v", err)
		http.Error(w, "Unable to load routes", http.StatusInternalServerError)
		return
	}

	var positions []struct {
		Position int    `json:"position"`
		Student  string `json:"student"`
	}

	// Find the correct route using RouteID from assignment
	for _, rt := range routes {
		if rt.RouteID == assignment.RouteID || rt.RouteName == assignment.RouteName {
			positions = rt.Positions
			break
		}
	}

	// Build attendance data
	var attendance []struct {
		Position   int    `json:"position"`
		Present    bool   `json:"present"`
		PickupTime string `json:"pickup_time,omitempty"`
	}

	for _, p := range positions {
		present := r.FormValue("present_"+strconv.Itoa(p.Position)) == "on"
		pickup := SanitizeInput(r.FormValue("pickup_" + strconv.Itoa(p.Position)))
		attendance = append(attendance, struct {
			Position   int    `json:"position"`
			Present    bool   `json:"present"`
			PickupTime string `json:"pickup_time,omitempty"`
		}{p.Position, present, pickup})
	}

	// Create driver log
	driverLog := DriverLog{
		Driver:     user.Username,
		BusID:      busID,
		RouteID:    assignment.RouteID,
		Date:       date,
		Period:     period,
		Departure:  departure,
		Arrival:    arrival,
		Mileage:    mileage,
		Attendance: attendance,
	}

	// Save the log
	if err := saveDriverLog(driverLog); err != nil {
		log.Printf("Error saving driver log: %v", err)
		http.Error(w, "Unable to save log", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/driver-dashboard?date="+date+"&period="+period, http.StatusSeeOther)
}

func removeUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	user := getSecureUser(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	// Verify CSRF token
	if !verifyCSRFToken(r) {
		http.Error(w, "Invalid CSRF token", http.StatusForbidden)
		return
	}

	r.ParseForm()
	usernameToRemove := SanitizeInput(r.FormValue("username"))

	if usernameToRemove == "" {
		http.Error(w, "Username is required", http.StatusBadRequest)
		return
	}

	// Prevent removing yourself
	if usernameToRemove == user.Username {
		http.Error(w, "Cannot remove yourself", http.StatusBadRequest)
		return
	}

	// Check if user exists
	users := loadUsers()
	userFound := false
	for _, u := range users {
		if u.Username == usernameToRemove {
			userFound = true
			break
		}
	}

	if !userFound {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// If removing a driver, also remove their route assignments
	if err := deleteRouteAssignment(usernameToRemove); err != nil {
		log.Printf("Warning: Failed to delete route assignment for %s: %v", usernameToRemove, err)
	}

	// Delete the user
	if err := deleteUser(usernameToRemove); err != nil {
		log.Printf("Error deleting user: %v", err)
		http.Error(w, "Unable to delete user", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/manager-dashboard", http.StatusFound)
}

func logout(w http.ResponseWriter, r *http.Request) {
	// Clear session
	if cookie, err := r.Cookie("session_id"); err == nil {
		sessionMgr.mu.Lock()
		delete(sessionMgr.sessions, cookie.Value)
		sessionMgr.mu.Unlock()
	}

	// Clear cookies
	http.SetCookie(w, &http.Cookie{Name: "session_id", Value: "", MaxAge: -1, Path: "/"})
	http.SetCookie(w, &http.Cookie{Name: "csrf_token", Value: "", MaxAge: -1, Path: "/"})
	http.Redirect(w, r, "/", http.StatusFound)
}

func dashboardRouter(w http.ResponseWriter, r *http.Request) {
	user := getSecureUser(r)
	if user == nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	if user.Role == "manager" {
		managerDashboard(w, r)
	} else if user.Role == "driver" {
		driverDashboard(w, r)
	} else {
		http.Redirect(w, r, "/", http.StatusFound)
	}
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

func getSecureUser(r *http.Request) *User {
	cookie, err := r.Cookie("session_id")
	if err != nil {
		return nil
	}

	sessionMgr.mu.RLock()
	session, exists := sessionMgr.sessions[cookie.Value]
	sessionMgr.mu.RUnlock()

	if !exists || session.ExpiresAt.Before(time.Now()) {
		return nil
	}

	users := loadUsers()
	for _, u := range users {
		if u.Username == session.Username {
			return &u
		}
	}
	return nil
}

func SanitizeInput(input string) string {
	return strings.TrimSpace(input) // Add any HTML/entity escaping if needed
}

func ValidateUsername(username string) bool {
	return len(username) > 2 // Add more logic like regex if needed
}

func ValidateDate(date string) bool {
	_, err := time.Parse("2006-01-02", date)
	return err == nil
}

func SetSecureCookie(w http.ResponseWriter, name, value string) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		HttpOnly: true,
		Path:     "/",
		Expires:  time.Now().Add(24 * time.Hour),
		Secure:   false, // Set to true in production with HTTPS
		SameSite: http.SameSiteStrictMode,
	})
}

func executeTemplate(w http.ResponseWriter, tmpl string, data interface{}) {
	err := templates.ExecuteTemplate(w, tmpl, data)
	if err != nil {
		http.Error(w, "Template rendering error", http.StatusInternalServerError)
		log.Printf("Template execution error for %s: %v", tmpl, err)
	}
}

func CreateSecureSession(username, role string) (string, string, error) {
	sessionID := fmt.Sprintf("%d_%s", time.Now().UnixNano(), username)
	csrfToken := fmt.Sprintf("%x", time.Now().UnixNano())
	return sessionID, csrfToken, nil
}

func getCSRFToken(r *http.Request) string {
	cookie, err := r.Cookie("csrf_token")
	if err != nil {
		return ""
	}
	return cookie.Value
}

func verifyCSRFToken(r *http.Request) bool {
	sessionCookie, err := r.Cookie("session_id")
	if err != nil {
		return false
	}

	csrfToken := r.FormValue("csrf_token")
	if csrfToken == "" {
		csrfToken = r.Header.Get("X-CSRF-Token")
	}

	sessionMgr.mu.RLock()
	session, exists := sessionMgr.sessions[sessionCookie.Value]
	sessionMgr.mu.RUnlock()

	if !exists {
		return false
	}

	return subtle.ConstantTimeCompare([]byte(session.CSRFToken), []byte(csrfToken)) == 1
}

// =============================================================================
// MAIN FUNCTION
// =============================================================================

func main() {
	// Setup database
	log.Println("🗄️  Setting up PostgreSQL database...")
	setupDatabase()
	defer closeDatabase()
	
	log.Println("✅ Database setup complete")

	// Setup HTTP routes with security middleware
	mux := http.NewServeMux()
	
	// Public routes
	mux.HandleFunc("/", withRecovery(RateLimitMiddleware(loginPage)))
	mux.HandleFunc("/logout", withRecovery(logout))
	mux.HandleFunc("/health", withRecovery(healthCheck))

	// Protected routes
	mux.HandleFunc("/new-user", withRecovery(SecurityHeaders(newUserPage)))
	mux.HandleFunc("/edit-user", withRecovery(SecurityHeaders(editUserPage)))
	mux.HandleFunc("/dashboard", withRecovery(SecurityHeaders(dashboardRouter)))
	mux.HandleFunc("/manager-dashboard", withRecovery(SecurityHeaders(managerDashboard)))
	mux.HandleFunc("/driver-dashboard", withRecovery(SecurityHeaders(driverDashboard)))
	mux.HandleFunc("/save-log", withRecovery(SecurityHeaders(saveDriverLogHandler)))
	mux.HandleFunc("/remove-user", withRecovery(SecurityHeaders(removeUser)))
	
	// Add other routes...
	// (Include all other routes from original main.go with security middleware)

	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}

	server := &http.Server{
		Addr:           "0.0.0.0:" + port,
		Handler:        mux,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   60 * time.Second,
		IdleTimeout:    120 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1 MB
	}

	log.Printf("Server starting on port %s", port)
	log.Printf("Server will be accessible at: http://0.0.0.0:%s", port)

	if err := server.ListenAndServe(); err != nil {
		if err == http.ErrServerClosed {
			log.Println("Server was closed")
		} else {
			log.Printf("Server failed to start: %v", err)
			os.Exit(1)
		}
	}
}
