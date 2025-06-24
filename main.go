package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"time"
)

// Templates variable
var templates *template.Template

func init() {
	funcMap := template.FuncMap{
		"json": func(v interface{}) template.JS {
			b, err := json.Marshal(v)
			if err != nil {
				log.Printf("JSON marshal error: %v", err)
				return template.JS("{}")
			}
			return template.JS(b)
		},
		"add": func(a, b int) int { return a + b },
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
		"printf": fmt.Sprintf,
	}

	var err error
	templates, err = template.New("").Funcs(funcMap).ParseGlob("templates/*.html")
	if err != nil {
		log.Fatalf("Failed to load templates: %v", err)
	}
}

func main() {
	// Database setup
	log.Println("🗄️  Setting up PostgreSQL database...")
	setupDatabase()
	defer closeDatabase()

	mux := http.NewServeMux()
	
	// Public registration routes
	mux.HandleFunc("/register", withRecovery(RateLimitMiddleware(registerHandler)))
	
	// Manager routes for approving users
	mux.HandleFunc("/approve-users", withRecovery(requireAuth(requireRole("manager")(approveUsersHandler))))
	mux.HandleFunc("/approve-user", withRecovery(requireAuth(requireRole("manager")(approveUserHandler))))
	
	// Replace the existing login handler with the new one that checks for pending status
	mux.HandleFunc("/", withRecovery(RateLimitMiddleware(loginHandlerWithApproval)))
	
	// Public routes
	mux.HandleFunc("/", withRecovery(RateLimitMiddleware(loginHandler)))
	mux.HandleFunc("/logout", withRecovery(logout))
	mux.HandleFunc("/health", withRecovery(healthCheck))

	// Protected routes - Using the middleware approach
	mux.HandleFunc("/new-user", withRecovery(requireAuth(requireRole("manager")(newUserHandler))))
	mux.HandleFunc("/edit-user", withRecovery(requireAuth(requireRole("manager")(editUserHandler))))
	mux.HandleFunc("/dashboard", withRecovery(requireAuth(dashboardRouter)))
	mux.HandleFunc("/manager-dashboard", withRecovery(requireAuth(requireRole("manager")(managerDashboard))))
	mux.HandleFunc("/driver-dashboard", withRecovery(requireAuth(requireRole("driver")(driverDashboard))))
	mux.HandleFunc("/save-log", withRecovery(requireAuth(requireRole("driver")(saveDriverLogHandler))))
	mux.HandleFunc("/remove-user", withRecovery(requireAuth(requireRole("manager")(removeUserHandler))))
	
	// Student routes
	mux.HandleFunc("/students", withRecovery(requireAuth(requireRole("driver")(studentsPage))))
	mux.HandleFunc("/add-student", withRecovery(requireAuth(requireRole("driver")(addStudentHandler))))
	mux.HandleFunc("/edit-student", withRecovery(requireAuth(requireRole("driver")(editStudentHandler))))
	mux.HandleFunc("/remove-student", withRecovery(requireAuth(requireRole("driver")(removeStudentHandler))))
	
	// Fleet routes
	mux.HandleFunc("/fleet", withRecovery(requireAuth(requireRole("manager")(fleetPage))))
	mux.HandleFunc("/company-fleet", withRecovery(requireAuth(requireRole("manager")(companyFleetPage))))
	mux.HandleFunc("/update-vehicle-status", withRecovery(requireAuth(requireRole("manager")(updateVehicleStatus))))
	
	// Route assignment routes
	mux.HandleFunc("/assign-routes", withRecovery(requireAuth(requireRole("manager")(assignRoutesPage))))
	mux.HandleFunc("/assign-route", withRecovery(requireAuth(requireRole("manager")(assignRouteHandler))))
	mux.HandleFunc("/unassign-route", withRecovery(requireAuth(requireRole("manager")(unassignRouteHandler))))
	mux.HandleFunc("/assign-routes/add", withRecovery(requireAuth(requireRole("manager")(addRouteHandler))))
	mux.HandleFunc("/assign-routes/edit", withRecovery(requireAuth(requireRole("manager")(editRouteHandler))))
	mux.HandleFunc("/assign-routes/delete", withRecovery(requireAuth(requireRole("manager")(deleteRouteHandler))))
	
	// Driver profile
	mux.HandleFunc("/driver/", withRecovery(requireAuth(requireRole("manager")(driverProfileHandler))))

	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}

	server := &http.Server{
		Addr:           fmt.Sprintf(":%s", port),
		Handler:        SecurityHeaders(mux),
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   60 * time.Second,
		IdleTimeout:    120 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	log.Printf("🚀 Server starting on port %s", port)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
// Add these handlers to your main.go file

// ============= REGISTRATION HANDLERS =============

func registerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		data := struct {
			Error string
		}{}
		executeTemplate(w, "register.html", data)
		return
	}

	// Handle POST - new registration
	username := SanitizeFormValue(r, "username")
	password := r.FormValue("password")

	// Validate username format
	if !ValidateUsername(username) {
		data := struct {
			Error string
		}{
			Error: "Invalid username format. Use 3-20 characters, letters and numbers only.",
		}
		executeTemplate(w, "register.html", data)
		return
	}

	// Validate password length
	if len(password) < 6 {
		data := struct {
			Error string
		}{
			Error: "Password must be at least 6 characters long.",
		}
		executeTemplate(w, "register.html", data)
		return
	}

	// Check if username already exists
	users := loadUsers()
	for _, user := range users {
		if user.Username == username {
			data := struct {
				Error string
			}{
				Error: "Username already exists. Please choose another.",
			}
			executeTemplate(w, "register.html", data)
			return
		}
	}

	// Hash password
	hashedPassword, err := HashPassword(password)
	if err != nil {
		http.Error(w, "Failed to process registration", http.StatusInternalServerError)
		return
	}

	// Create pending user (driver by default, pending approval)
	newUser := User{
		Username: username,
		Password: hashedPassword,
		Role:     "driver_pending", // Special role for pending approval
	}

	if err := saveUser(newUser); err != nil {
		data := struct {
			Error string
		}{
			Error: "Failed to create account. Please try again.",
		}
		executeTemplate(w, "register.html", data)
		return
	}

	// Show success page
	executeTemplate(w, "registration_success.html", nil)
}

func approveUsersHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	// Get all pending users
	allUsers := loadUsers()
	var pendingUsers []struct {
		Username  string
		CreatedAt string
	}

	for _, u := range allUsers {
		if u.Role == "driver_pending" {
			pendingUsers = append(pendingUsers, struct {
				Username  string
				CreatedAt string
			}{
				Username:  u.Username,
				CreatedAt: "Recently", // You could add timestamp to User struct
			})
		}
	}

	// Get CSRF token from session
	cookie, _ := r.Cookie("session_id")
	session, _ := GetSecureSession(cookie.Value)

	data := struct {
		PendingUsers []struct {
			Username  string
			CreatedAt string
		}
		CSRFToken string
	}{
		PendingUsers: pendingUsers,
		CSRFToken:    session.CSRFToken,
	}

	executeTemplate(w, "approve_users.html", data)
}

func approveUserHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Error(w, "Unauthorized", http.StatusForbidden)
		return
	}

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Validate CSRF token
	cookie, _ := r.Cookie("session_id")
	if !ValidateCSRFToken(cookie.Value, r.FormValue("csrf_token")) {
		http.Error(w, "Invalid CSRF token", http.StatusForbidden)
		return
	}

	username := r.FormValue("username")
	action := r.FormValue("action")

	if username == "" || (action != "approve" && action != "reject") {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Load all users
	users := loadUsers()
	updated := false

	for i, u := range users {
		if u.Username == username && u.Role == "driver_pending" {
			if action == "approve" {
				users[i].Role = "driver" // Change to active driver
			} else {
				// For reject, we'll delete the user
				if err := deleteUser(username); err != nil {
					http.Error(w, "Failed to process request", http.StatusInternalServerError)
					return
				}
				http.Redirect(w, r, "/approve-users", http.StatusFound)
				return
			}
			updated = true
			break
		}
	}

	if !updated {
		http.Error(w, "User not found or already processed", http.StatusNotFound)
		return
	}

	// Save the updated users
	if err := saveUsers(users); err != nil {
		http.Error(w, "Failed to update user", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/approve-users", http.StatusFound)
}

// Update your loginHandler to check for pending users
func loginHandlerWithApproval(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		// Check if already logged in
		cookie, err := r.Cookie("session_id")
		if err == nil {
			if session, exists := GetSecureSession(cookie.Value); exists {
				if session.Role == "manager" {
					http.Redirect(w, r, "/manager-dashboard", http.StatusFound)
				} else {
					http.Redirect(w, r, "/driver-dashboard", http.StatusFound)
				}
				return
			}
		}
		
		csrfToken, _ := GenerateSecureToken()
		data := LoginFormData{
			CSRFToken: csrfToken,
		}
		executeTemplate(w, "login.html", data)
		return
	}

	// Handle POST
	username := SanitizeFormValue(r, "username")
	password := r.FormValue("password")

	// Validate username format
	if !ValidateUsername(username) {
		csrfToken, _ := GenerateSecureToken()
		data := LoginFormData{
			Error:     "Invalid username format",
			CSRFToken: csrfToken,
		}
		executeTemplate(w, "login.html", data)
		return
	}

	// Check credentials
	users := loadUsers()
	for _, user := range users {
		if user.Username == username && CheckPasswordHash(password, user.Password) {
			// Check if user is pending approval
			if user.Role == "driver_pending" {
				csrfToken, _ := GenerateSecureToken()
				data := LoginFormData{
					Error:     "Your account is pending approval. Please wait for a manager to approve your registration.",
					CSRFToken: csrfToken,
				}
				executeTemplate(w, "login.html", data)
				return
			}

			// Create session for approved users only
			sessionID, _, err := CreateSecureSession(username, user.Role)
			if err != nil {
				http.Error(w, "Session creation failed", http.StatusInternalServerError)
				return
			}
			
			// Set session cookie
			SetSecureCookie(w, "session_id", sessionID)
			
			// Redirect based on role
			if user.Role == "manager" {
				http.Redirect(w, r, "/manager-dashboard", http.StatusFound)
			} else {
				http.Redirect(w, r, "/driver-dashboard", http.StatusFound)
			}
			return
		}
	}

	// Invalid credentials
	csrfToken, _ := GenerateSecureToken()
	data := LoginFormData{
		Error:     "Invalid username or password",
		CSRFToken: csrfToken,
	}
	executeTemplate(w, "login.html", data)
}



// ============= HANDLERS =============

func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok","service":"bus-fleet-management","timestamp":"` + time.Now().Format(time.RFC3339) + `"}`))
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		// Check if already logged in
		cookie, err := r.Cookie("session_id")
		if err == nil {
			if session, exists := GetSecureSession(cookie.Value); exists {
				if session.Role == "manager" {
					http.Redirect(w, r, "/manager-dashboard", http.StatusFound)
				} else {
					http.Redirect(w, r, "/driver-dashboard", http.StatusFound)
				}
				return
			}
		}
		
		csrfToken, _ := GenerateSecureToken()
		data := LoginFormData{
			CSRFToken: csrfToken,
		}
		executeTemplate(w, "login.html", data)
		return
	}

	// Handle POST
	username := SanitizeFormValue(r, "username")
	password := r.FormValue("password")

	// Validate username format
	if !ValidateUsername(username) {
		csrfToken, _ := GenerateSecureToken()
		data := LoginFormData{
			Error:     "Invalid username format",
			CSRFToken: csrfToken,
		}
		executeTemplate(w, "login.html", data)
		return
	}

	// Check credentials
	users := loadUsers()
	for _, user := range users {
		if user.Username == username && CheckPasswordHash(password, user.Password) {
			// Create session
			sessionID, _, err := CreateSecureSession(username, user.Role)
			if err != nil {
				http.Error(w, "Session creation failed", http.StatusInternalServerError)
				return
			}
			
			// Set session cookie
			SetSecureCookie(w, "session_id", sessionID)
			
			// Redirect based on role
			if user.Role == "manager" {
				http.Redirect(w, r, "/manager-dashboard", http.StatusFound)
			} else {
				http.Redirect(w, r, "/driver-dashboard", http.StatusFound)
			}
			return
		}
	}

	// Invalid credentials
	csrfToken, _ := GenerateSecureToken()
	data := LoginFormData{
		Error:     "Invalid username or password",
		CSRFToken: csrfToken,
	}
	executeTemplate(w, "login.html", data)
}

func logout(w http.ResponseWriter, r *http.Request) {
	// Clear session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})
	
	http.Redirect(w, r, "/", http.StatusFound)
}

func dashboardRouter(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	
	if user.Role == "manager" {
		http.Redirect(w, r, "/manager-dashboard", http.StatusFound)
	} else {
		http.Redirect(w, r, "/driver-dashboard", http.StatusFound)
	}
}

func managerDashboard(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	// Get CSRF token from session
	cookie, _ := r.Cookie("session_id")
	session, _ := GetSecureSession(cookie.Value)
	
	data := DashboardData{
		User:            user,
		Role:            user.Role,
		Users:           loadUsers(),
		Buses:           loadBuses(),
		Routes:          []Route{}, // We'll load these separately
		DriverSummaries: []*DriverSummary{},
		RouteStats:      []*RouteStats{},
		Activities:      []Activity{},
		CSRFToken:       session.CSRFToken,
	}
	
	// Load routes
	routes, err := loadRoutes()
	if err != nil {
		log.Printf("Error loading routes: %v", err)
	} else {
		data.Routes = routes
	}
	
	executeTemplate(w, "dashboard.html", data)
}

func driverDashboard(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "driver" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	// Get date and period from query params
	date := r.URL.Query().Get("date")
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}
	
	period := r.URL.Query().Get("period")
	if period == "" {
		if time.Now().Hour() < 12 {
			period = "morning"
		} else {
			period = "afternoon"
		}
	}
	
	// Get driver's route assignment
	assignment, err := getDriverRouteAssignment(user.Username)
	
	var route *Route
	var bus *Bus
	
	if err == nil && assignment != nil {
		// Get route details
		routes, _ := loadRoutes()
		for _, r := range routes {
			if r.RouteID == assignment.RouteID {
				route = &r
				break
			}
		}
		
		// Get bus details
		buses := loadBuses()
		for _, b := range buses {
			if b.BusID == assignment.BusID {
				bus = b
				break
			}
		}
	}
	
	// Get existing log for this date/period
	logs, _ := loadDriverLogs()
	var driverLog *DriverLog
	for _, log := range logs {
		if log.Driver == user.Username && log.Date == date && log.Period == period {
			driverLog = &log
			break
		}
	}
	
	// Get recent logs
	var recentLogs []DriverLog
	for _, log := range logs {
		if log.Driver == user.Username {
			recentLogs = append(recentLogs, log)
			if len(recentLogs) >= 5 { // Show last 5 logs
				break
			}
		}
	}
	
	// Get CSRF token from session
	cookie, _ := r.Cookie("session_id")
	session, _ := GetSecureSession(cookie.Value)
	
	data := DriverDashboardData{
		User:       user,
		Date:       date,
		Period:     period,
		Route:      route,
		Bus:        bus,
		DriverLog:  driverLog,
		RecentLogs: recentLogs,
		CSRFToken:  session.CSRFToken,
	}
	
	executeTemplate(w, "driver_dashboard.html", data)
}

func newUserHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	if r.Method == "GET" {
		// Get CSRF token
		cookie, _ := r.Cookie("session_id")
		session, _ := GetSecureSession(cookie.Value)
		
		data := UserFormData{
			CSRFToken: session.CSRFToken,
		}
		executeTemplate(w, "new_user.html", data)
		return
	}

	// Handle POST
	// Validate CSRF token
	cookie, _ := r.Cookie("session_id")
	if !ValidateCSRFToken(cookie.Value, r.FormValue("csrf_token")) {
		http.Error(w, "Invalid CSRF token", http.StatusForbidden)
		return
	}
	
	username := SanitizeFormValue(r, "username")
	password := r.FormValue("password")
	role := SanitizeFormValue(r, "role")
	
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
	hashedPassword, err := HashPassword(password)
	if err != nil {
		http.Error(w, "Failed to hash password", http.StatusInternalServerError)
		return
	}
	
	// Create user
	newUser := User{
		Username: username,
		Password: hashedPassword,
		Role:     role,
	}
	
	if err := saveUser(newUser); err != nil {
		http.Error(w, "Failed to create user", http.StatusInternalServerError)
		return
	}
	
	http.Redirect(w, r, "/manager-dashboard", http.StatusFound)
}

func editUserHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	if r.Method == "GET" {
		username := r.URL.Query().Get("username")
		if username == "" {
			http.Error(w, "Username required", http.StatusBadRequest)
			return
		}
		
		// Find user
		users := loadUsers()
		var targetUser *User
		for _, u := range users {
			if u.Username == username {
				targetUser = &u
				break
			}
		}
		
		if targetUser == nil {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		
		// Get CSRF token
		cookie, _ := r.Cookie("session_id")
		session, _ := GetSecureSession(cookie.Value)
		
		data := struct {
			Username  string
			Role      string
			CSRFToken string
		}{
			Username:  targetUser.Username,
			Role:      targetUser.Role,
			CSRFToken: session.CSRFToken,
		}
		
		executeTemplate(w, "users.html", data)
		return
	}

	// Handle POST
	// Validate CSRF token
	cookie, _ := r.Cookie("session_id")
	if !ValidateCSRFToken(cookie.Value, r.FormValue("csrf_token")) {
		http.Error(w, "Invalid CSRF token", http.StatusForbidden)
		return
	}
	
	username := r.FormValue("username")
	password := r.FormValue("password")
	role := r.FormValue("role")
	
	if len(password) < 6 {
		http.Error(w, "Password must be at least 6 characters", http.StatusBadRequest)
		return
	}
	
	// Hash password
	hashedPassword, err := HashPassword(password)
	if err != nil {
		http.Error(w, "Failed to hash password", http.StatusInternalServerError)
		return
	}
	
	// Update user
	users := loadUsers()
	updated := false
	for i, u := range users {
		if u.Username == username {
			users[i].Password = hashedPassword
			users[i].Role = role
			updated = true
			break
		}
	}
	
	if !updated {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}
	
	if err := saveUsers(users); err != nil {
		http.Error(w, "Failed to update user", http.StatusInternalServerError)
		return
	}
	
	http.Redirect(w, r, "/manager-dashboard", http.StatusFound)
}

func removeUserHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	username := r.URL.Query().Get("username")
	if username == "" || username == user.Username {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	
	if err := deleteUser(username); err != nil {
		http.Error(w, "Failed to remove user", http.StatusInternalServerError)
		return
	}
	
	http.Redirect(w, r, "/manager-dashboard", http.StatusFound)
}

func saveDriverLogHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "driver" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	// Parse form
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}
	
	// Validate CSRF token
	cookie, _ := r.Cookie("session_id")
	if !ValidateCSRFToken(cookie.Value, r.FormValue("csrf_token")) {
		http.Error(w, "Invalid CSRF token", http.StatusForbidden)
		return
	}
	
	// Get form values
	date := r.FormValue("date")
	period := r.FormValue("period")
	departure := r.FormValue("departure")
	arrival := r.FormValue("arrival")
	mileageStr := r.FormValue("mileage")
	routeID := r.FormValue("route_id")
	busID := r.FormValue("bus_id")
	
	// Convert mileage
	var mileage float64
	fmt.Sscanf(mileageStr, "%f", &mileage)
	
	// Build attendance records
	var attendance []struct {
		Position   int    `json:"position"`
		Present    bool   `json:"present"`
		PickupTime string `json:"pickup_time,omitempty"`
	}
	
	// Process attendance checkboxes
	for key, _ := range r.Form {
		if len(key) > 8 && key[:8] == "present_" {
			var position int
			fmt.Sscanf(key[8:], "%d", &position)
			
			pickupTime := r.FormValue(fmt.Sprintf("pickup_%d", position))
			
			attendance = append(attendance, struct {
				Position   int    `json:"position"`
				Present    bool   `json:"present"`
				PickupTime string `json:"pickup_time,omitempty"`
			}{
				Position:   position,
				Present:    true,
				PickupTime: pickupTime,
			})
		}
	}
	
	// Create driver log
	driverLog := DriverLog{
		Driver:     user.Username,
		BusID:      busID,
		RouteID:    routeID,
		Date:       date,
		Period:     period,
		Departure:  departure,
		Arrival:    arrival,
		Mileage:    mileage,
		Attendance: attendance,
	}
	
	// Save log
	if err := saveDriverLog(driverLog); err != nil {
		http.Error(w, "Failed to save log", http.StatusInternalServerError)
		return
	}
	
	// Redirect back to dashboard
	http.Redirect(w, r, fmt.Sprintf("/driver-dashboard?date=%s&period=%s", date, period), http.StatusFound)
}

func studentsPage(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "driver" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	// Get all students for this driver
	allStudents := loadStudents()
	var driverStudents []Student
	for _, s := range allStudents {
		if s.Driver == user.Username {
			driverStudents = append(driverStudents, s)
		}
	}
	
	// Get routes for the dropdown
	routes, _ := loadRoutes()
	
	// Get CSRF token
	cookie, _ := r.Cookie("session_id")
	session, _ := GetSecureSession(cookie.Value)
	
	data := StudentData{
		User:      user,
		Students:  driverStudents,
		Routes:    routes,
		CSRFToken: session.CSRFToken,
	}
	
	executeTemplate(w, "students.html", data)
}

func addStudentHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "driver" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// Parse form
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}
	
	// Generate student ID
	students := loadStudents()
	studentID := fmt.Sprintf("STU%03d", len(students)+1)
	
	// Get form values
	name := SanitizeFormValue(r, "name")
	guardian := SanitizeFormValue(r, "guardian")
	phoneNumber := SanitizeFormValue(r, "phone_number")
	altPhoneNumber := SanitizeFormValue(r, "alt_phone_number")
	pickupTime := r.FormValue("pickup_time")
	dropoffTime := r.FormValue("dropoff_time")
	routeID := r.FormValue("route_id")
	
	var positionNumber int
	fmt.Sscanf(r.FormValue("position_number"), "%d", &positionNumber)
	
	// Build locations
	var locations []Location
	
	// Process pickup locations
	pickupAddresses := r.Form["pickup_address"]
	pickupDescriptions := r.Form["pickup_description"]
	for i := range pickupAddresses {
		if pickupAddresses[i] != "" {
			desc := ""
			if i < len(pickupDescriptions) {
				desc = pickupDescriptions[i]
			}
			locations = append(locations, Location{
				Type:        "pickup",
				Address:     pickupAddresses[i],
				Description: desc,
			})
		}
	}
	
	// Process dropoff locations
	dropoffAddresses := r.Form["dropoff_address"]
	dropoffDescriptions := r.Form["dropoff_description"]
	for i := range dropoffAddresses {
		if dropoffAddresses[i] != "" {
			desc := ""
			if i < len(dropoffDescriptions) {
				desc = dropoffDescriptions[i]
			}
			locations = append(locations, Location{
				Type:        "dropoff",
				Address:     dropoffAddresses[i],
				Description: desc,
			})
		}
	}
	
	// Create student
	student := Student{
		StudentID:      studentID,
		Name:           name,
		Locations:      locations,
		PhoneNumber:    phoneNumber,
		AltPhoneNumber: altPhoneNumber,
		Guardian:       guardian,
		PickupTime:     pickupTime,
		DropoffTime:    dropoffTime,
		PositionNumber: positionNumber,
		RouteID:        routeID,
		Driver:         user.Username,
		Active:         true,
	}
	
	// Save student
	if err := saveStudent(student); err != nil {
		http.Error(w, "Failed to save student", http.StatusInternalServerError)
		return
	}
	
	http.Redirect(w, r, "/students", http.StatusFound)
}

func editStudentHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "driver" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// Parse form
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}
	
	studentID := r.FormValue("student_id")
	
	// Find existing student
	students := loadStudents()
	var student *Student
	for i := range students {
		if students[i].StudentID == studentID && students[i].Driver == user.Username {
			student = &students[i]
			break
		}
	}
	
	if student == nil {
		http.Error(w, "Student not found", http.StatusNotFound)
		return
	}
	
	// Update fields
	student.Name = SanitizeFormValue(r, "name")
	student.Guardian = SanitizeFormValue(r, "guardian")
	student.PhoneNumber = SanitizeFormValue(r, "phone_number")
	student.AltPhoneNumber = SanitizeFormValue(r, "alt_phone_number")
	student.PickupTime = r.FormValue("pickup_time")
	student.DropoffTime = r.FormValue("dropoff_time")
	student.RouteID = r.FormValue("route_id")
	student.Active = r.FormValue("active") == "on"
	
	fmt.Sscanf(r.FormValue("position_number"), "%d", &student.PositionNumber)
	
	// Rebuild locations
	student.Locations = []Location{}
	
	// Process pickup locations
	pickupAddresses := r.Form["pickup_address"]
	pickupDescriptions := r.Form["pickup_description"]
	for i := range pickupAddresses {
		if pickupAddresses[i] != "" {
			desc := ""
			if i < len(pickupDescriptions) {
				desc = pickupDescriptions[i]
			}
			student.Locations = append(student.Locations, Location{
				Type:        "pickup",
				Address:     pickupAddresses[i],
				Description: desc,
			})
		}
	}
	
	// Process dropoff locations
	dropoffAddresses := r.Form["dropoff_address"]
	dropoffDescriptions := r.Form["dropoff_description"]
	for i := range dropoffAddresses {
		if dropoffAddresses[i] != "" {
			desc := ""
			if i < len(dropoffDescriptions) {
				desc = dropoffDescriptions[i]
			}
			student.Locations = append(student.Locations, Location{
				Type:        "dropoff",
				Address:     dropoffAddresses[i],
				Description: desc,
			})
		}
	}
	
	// Save updated student
	if err := saveStudent(*student); err != nil {
		http.Error(w, "Failed to update student", http.StatusInternalServerError)
		return
	}
	
	http.Redirect(w, r, "/students", http.StatusFound)
}

func removeStudentHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "driver" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	studentID := r.FormValue("student_id")
	if studentID == "" {
		http.Error(w, "Student ID required", http.StatusBadRequest)
		return
	}
	
	// Verify student belongs to this driver
	students := loadStudents()
	found := false
	for _, s := range students {
		if s.StudentID == studentID && s.Driver == user.Username {
			found = true
			break
		}
	}
	
	if !found {
		http.Error(w, "Student not found", http.StatusNotFound)
		return
	}
	
	if err := deleteStudent(studentID); err != nil {
		http.Error(w, "Failed to remove student", http.StatusInternalServerError)
		return
	}
	
	http.Redirect(w, r, "/students", http.StatusFound)
}

func fleetPage(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	buses := loadBuses()
	
	// Get CSRF token
	cookie, _ := r.Cookie("session_id")
	session, _ := GetSecureSession(cookie.Value)
	
	data := FleetData{
		User:      user,
		Buses:     buses,
		Today:     time.Now().Format("2006-01-02"),
		CSRFToken: session.CSRFToken,
	}
	
	executeTemplate(w, "fleet.html", data)
}

func companyFleetPage(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	vehicles := loadVehicles()
	
	// Get CSRF token
	cookie, _ := r.Cookie("session_id")
	session, _ := GetSecureSession(cookie.Value)
	
	data := CompanyFleetData{
		User:      user,
		Vehicles:  vehicles,
		CSRFToken: session.CSRFToken,
	}
	
	executeTemplate(w, "company_fleet.html", data)
}

func updateVehicleStatus(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// Parse form
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}
	
	vehicleID := r.FormValue("vehicle_id")
	statusType := r.FormValue("status_type")
	newStatus := r.FormValue("new_status")
	
	log.Printf("Update vehicle status: ID=%s, Type=%s, Status=%s", vehicleID, statusType, newStatus)
	
	if vehicleID == "" || statusType == "" || newStatus == "" {
		http.Error(w, "Missing required parameters", http.StatusBadRequest)
		return
	}
	
	// Load vehicles
	vehicles := loadVehicles()
	
	// Find and update vehicle
	updated := false
	for i := range vehicles {
		if vehicles[i].VehicleID == vehicleID {
			switch statusType {
			case "oil":
				vehicles[i].OilStatus = newStatus
			case "tire":
				vehicles[i].TireStatus = newStatus
			case "status":
				vehicles[i].Status = newStatus
			default:
				http.Error(w, "Invalid status type", http.StatusBadRequest)
				return
			}
			
			// Save individual vehicle
			if err := saveVehicle(vehicles[i]); err != nil {
				log.Printf("Failed to save vehicle: %v", err)
				http.Error(w, "Failed to update vehicle", http.StatusInternalServerError)
				return
			}
			
			updated = true
			break
		}
	}
	
	if !updated {
		http.Error(w, "Vehicle not found", http.StatusNotFound)
		return
	}
	
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func assignRoutesPage(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	// Load data
	assignments, _ := loadRouteAssignments()
	routes, _ := loadRoutes()
	buses := loadBuses()
	users := loadUsers()
	
	// Filter drivers
	var drivers []User
	for _, u := range users {
		if u.Role == "driver" {
			drivers = append(drivers, u)
		}
	}
	
	// Get CSRF token
	cookie, _ := r.Cookie("session_id")
	session, _ := GetSecureSession(cookie.Value)
	
	data := AssignRouteData{
		User:            user,
		Assignments:     assignments,
		Drivers:         drivers,
		AvailableRoutes: routes,
		AvailableBuses:  buses,
		CSRFToken:       session.CSRFToken,
	}
	
	executeTemplate(w, "assign_routes.html", data)
}

func assignRouteHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// Parse form
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}
	
	driver := r.FormValue("driver")
	busID := r.FormValue("bus_id")
	routeID := r.FormValue("route_id")
	
	// Get route name
	routes, _ := loadRoutes()
	routeName := ""
	for _, r := range routes {
		if r.RouteID == routeID {
			routeName = r.RouteName
			break
		}
	}
	
	assignment := RouteAssignment{
		Driver:       driver,
		BusID:        busID,
		RouteID:      routeID,
		RouteName:    routeName,
		AssignedDate: time.Now().Format("2006-01-02"),
	}
	
	// Validate assignment
	if err := validateRouteAssignment(assignment); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	// Save assignment
	if err := saveRouteAssignment(assignment); err != nil {
		http.Error(w, "Failed to save assignment", http.StatusInternalServerError)
		return
	}
	
	http.Redirect(w, r, "/assign-routes", http.StatusFound)
}

func unassignRouteHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	driver := r.FormValue("driver")
	if driver == "" {
		http.Error(w, "Driver required", http.StatusBadRequest)
		return
	}
	
	if err := deleteRouteAssignment(driver); err != nil {
		http.Error(w, "Failed to unassign route", http.StatusInternalServerError)
		return
	}
	
	http.Redirect(w, r, "/assign-routes", http.StatusFound)
}

func addRouteHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	routeName := SanitizeFormValue(r, "route_name")
	description := SanitizeFormValue(r, "description")
	
	if routeName == "" {
		http.Error(w, "Route name required", http.StatusBadRequest)
		return
	}
	
	// Generate route ID
	routes, _ := loadRoutes()
	routeID := fmt.Sprintf("RT%03d", len(routes)+1)
	
	route := Route{
		RouteID:     routeID,
		RouteName:   routeName,
		Description: description,
		Positions:   []struct {
			Position int    `json:"position"`
			Student  string `json:"student"`
		}{},
	}
	
	if err := saveRoute(route); err != nil {
		http.Error(w, "Failed to save route", http.StatusInternalServerError)
		return
	}
	
	http.Redirect(w, r, "/assign-routes", http.StatusFound)
}

func editRouteHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	routeID := r.FormValue("route_id")
	routeName := SanitizeFormValue(r, "route_name")
	description := SanitizeFormValue(r, "description")
	
	if routeID == "" || routeName == "" {
		http.Error(w, "Route ID and name required", http.StatusBadRequest)
		return
	}
	
	// Find and update route
	routes, _ := loadRoutes()
	updated := false
	for i := range routes {
		if routes[i].RouteID == routeID {
			routes[i].RouteName = routeName
			routes[i].Description = description
			
			if err := saveRoute(routes[i]); err != nil {
				http.Error(w, "Failed to update route", http.StatusInternalServerError)
				return
			}
			updated = true
			break
		}
	}
	
	if !updated {
		http.Error(w, "Route not found", http.StatusNotFound)
		return
	}
	
	http.Redirect(w, r, "/assign-routes", http.StatusFound)
}

func deleteRouteHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	routeID := r.FormValue("route_id")
	if routeID == "" {
		http.Error(w, "Route ID required", http.StatusBadRequest)
		return
	}
	
	// Check if route is assigned
	assignments, _ := loadRouteAssignments()
	for _, a := range assignments {
		if a.RouteID == routeID {
			http.Error(w, "Cannot delete route that is currently assigned", http.StatusBadRequest)
			return
		}
	}
	
	if err := deleteRoute(routeID); err != nil {
		http.Error(w, "Failed to delete route", http.StatusInternalServerError)
		return
	}
	
	http.Redirect(w, r, "/assign-routes", http.StatusFound)
}

func driverProfileHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	// Extract driver username from URL path
	path := r.URL.Path
	driverUsername := path[len("/driver/"):]
	
	if driverUsername == "" {
		http.Error(w, "Driver username required", http.StatusBadRequest)
		return
	}
	
	// Get driver logs
	allLogs, _ := loadDriverLogs()
	var driverLogs []DriverLog
	for _, log := range allLogs {
		if log.Driver == driverUsername {
			driverLogs = append(driverLogs, log)
		}
	}
	
	data := struct {
		Name string
		Logs []DriverLog
	}{
		Name: driverUsername,
		Logs: driverLogs,
	}
	
	executeTemplate(w, "driver_profile.html", data)
}
