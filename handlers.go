package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"github.com/xuri/excelize/v2"
	_ "github.com/lib/pq"
)

// loginHandler handles user login
func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		// For login page, generate a temporary token since no session exists
		csrfToken, _ := GenerateSecureToken()
		renderTemplate(w, r, "login.html", LoginFormData{
			CSRFToken: csrfToken,
		})
		return
	}

	if r.Method == "POST" {
		username := r.FormValue("username")
		password := r.FormValue("password")

		// NO CSRF validation on login - no session exists yet

		// Authenticate user
		user, err := authenticateUser(username, password)
		if err != nil {
			renderLoginError(w, r, "Invalid username or password")
			return
		}

		// Create session with CSRF token
		sessionID, _, err := CreateSecureSession(user.Username, user.Role)
		if err != nil {
			renderLoginError(w, r, "Failed to create session")
			return
		}

		// Set session cookie
		http.SetCookie(w, &http.Cookie{
			Name:     SessionCookieName,
			Value:    sessionID,
			Path:     "/",
			HttpOnly: true,
			Secure:   !isDevelopment(),
			SameSite: http.SameSiteStrictMode,
			MaxAge:   86400, // 24 hours
		})

		// Redirect based on role
		if user.Role == "manager" {
			http.Redirect(w, r, "/manager-dashboard", http.StatusSeeOther)
		} else {
			http.Redirect(w, r, "/driver-dashboard", http.StatusSeeOther)
		}
	}
}
// renderLoginError renders the login page with an error message
func renderLoginError(w http.ResponseWriter, r *http.Request, errorMsg string) {
	csrfToken, _ := GenerateSecureToken()
	renderTemplate(w, r, "login.html", LoginFormData{
		Error:     errorMsg,
		CSRFToken: csrfToken,
	})
}

// registerHandler handles user registration
func registerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		csrfToken, _ := GenerateSecureToken()
		renderTemplate(w, r, "register.html", map[string]interface{}{
			"CSRFToken": csrfToken,
		})
		return
	}

	if r.Method == "POST" {
		username := r.FormValue("username")
		password := r.FormValue("password")
		role := r.FormValue("role")

		// NO CSRF validation on registration - no session exists yet

		// Validate input
		if username == "" || password == "" {
			http.Error(w, "All fields are required", http.StatusBadRequest)
			return
		}

		// Default role to driver if not specified
		if role == "" {
			role = "driver"
		}

		// Hash password
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, "Failed to process password", http.StatusInternalServerError)
			return
		}

		// Create user
		status := "pending"
		if role == "manager" {
			status = "active"
		}

		_, err = db.Exec(`
			INSERT INTO users (username, password, role, status, registration_date, created_at)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, username, string(hashedPassword), role, status, time.Now().Format("2006-01-02"), time.Now())

		if err != nil {
			http.Error(w, "Username already exists", http.StatusBadRequest)
			return
		}

		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

// logoutHandler handles user logout
func logoutHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(SessionCookieName)
	if err == nil {
		ClearSession(cookie.Value)
	}

	// Clear cookie
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   !isDevelopment(),
		SameSite: http.SameSiteStrictMode,
	})

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// dashboardHandler serves the manager dashboard
func dashboardHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// Debug logging
	log.Printf("Dashboard accessed by user: %s with role: %s", user.Username, user.Role)

	// Ensure managers go to manager dashboard
	if user.Role != "manager" {
		log.Printf("Non-manager user %s trying to access manager dashboard, redirecting", user.Username)
		http.Redirect(w, r, "/driver-dashboard", http.StatusSeeOther)
		return
	}

	// Load manager-specific data
	users := loadUsers()
	buses := loadBuses()
	
	// Convert []Route to []*Route
	routes, _ := loadRoutes()
	routePtrs := make([]*Route, len(routes))
	for i := range routes {
		routePtrs[i] = &routes[i]
	}
	
	driverSummaries := loadDriverSummaries()
	routeStats := loadRouteStats()
	activities, _ := loadActivities()
	pendingUsers := countPendingUsers()

	// Create the data structure that matches what the template expects
	data := map[string]interface{}{
		"Data": map[string]interface{}{
			"User":            user,
			"Role":            user.Role,
			"Users":           users,
			"Buses":           buses,
			"Routes":          routePtrs,
			"DriverSummaries": driverSummaries,
			"RouteStats":      routeStats,
			"Activities":      activities,
			"PendingUsers":    pendingUsers,
			"CSRFToken":       getSessionCSRFToken(r),
		},
		"CSPNonce": getCSPNonce(r),
	}

	// Debug log the data structure
	log.Printf("Rendering dashboard for manager %s with %d users, %d buses, %d routes", 
		user.Username, len(users), len(buses), len(routes))

	renderTemplate(w, r, "dashboard.html", data)
}

// approveUsersHandler shows pending users for approval
func approveUsersHandler(w http.ResponseWriter, r *http.Request) {
	users := []User{}
	rows, err := db.Query(`SELECT username, role, registration_date FROM users WHERE status = 'pending'`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var user User
			rows.Scan(&user.Username, &user.Role, &user.RegistrationDate)
			users = append(users, user)
		}
	}

	renderTemplate(w, r, "approve_users.html", map[string]interface{}{
		"Users":     users,
		"CSRFToken": getSessionCSRFToken(r),
	})
}

// approveUserHandler approves a pending user
func approveUserHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		username := r.FormValue("username")
		if !validateCSRF(r) {
			http.Error(w, "Invalid CSRF token", http.StatusForbidden)
			return
		}

		_, err := db.Exec(`UPDATE users SET status = 'active' WHERE username = $1`, username)
		if err != nil {
			http.Error(w, "Failed to approve user", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/approve-users", http.StatusSeeOther)
	}
}

// manageUsersHandler shows all users for management
func manageUsersHandler(w http.ResponseWriter, r *http.Request) {
	users := loadUsers()
	renderTemplate(w, r, "manage_users.html", map[string]interface{}{
		"Users":     users,
		"CSRFToken": getSessionCSRFToken(r),
	})
}

// editUserHandler handles user editing
func editUserHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		username := r.URL.Query().Get("username")
		user := getUserByUsername(username)
		if user == nil {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}

		renderTemplate(w, r, "edit_user.html", map[string]interface{}{
			"User":      user,
			"CSRFToken": getSessionCSRFToken(r),
		})
		return
	}

	if r.Method == "POST" {
		if !validateCSRF(r) {
			http.Error(w, "Invalid CSRF token", http.StatusForbidden)
			return
		}

		username := r.FormValue("username")
		newPassword := r.FormValue("password")
		role := r.FormValue("role")

		// Update user
		if newPassword != "" {
			hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
			db.Exec(`UPDATE users SET password = $1, role = $2 WHERE username = $3`, hashedPassword, role, username)
		} else {
			db.Exec(`UPDATE users SET role = $1 WHERE username = $2`, role, username)
		}

		http.Redirect(w, r, "/manage-users", http.StatusSeeOther)
	}
}

// deleteUserHandler deletes a user
func deleteUserHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		if !validateCSRF(r) {
			http.Error(w, "Invalid CSRF token", http.StatusForbidden)
			return
		}

		username := r.FormValue("username")
		db.Exec(`DELETE FROM users WHERE username = $1`, username)
		http.Redirect(w, r, "/manage-users", http.StatusSeeOther)
	}
}

// exportMileageHandler exports mileage data to Excel
func exportMileageHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	// Get filter parameters
	month := r.URL.Query().Get("month")
	year := r.URL.Query().Get("year")
	reportType := r.URL.Query().Get("type")
	
	if month == "" || year == "" {
		now := time.Now()
		month = now.Format("January")
		year = fmt.Sprintf("%d", now.Year())
	}
	
	yearInt, _ := strconv.Atoi(year)
	
	// Create Excel file
	f := excelize.NewFile()
	
	// Summary Sheet
	summarySheet := "Summary"
	f.SetSheetName("Sheet1", summarySheet)
	
	// Add headers
	headers := []string{"Metric", "Value"}
	for i, header := range headers {
		cell := fmt.Sprintf("%s1", string(rune('A'+i)))
		f.SetCellValue(summarySheet, cell, header)
	}
	
	// Calculate summary statistics
	stats := calculateMileageStatistics(month, yearInt)
	
	summaryData := [][]interface{}{
		{"Report Period", fmt.Sprintf("%s %d", month, yearInt)},
		{"Total Vehicles", stats.TotalVehicles},
		{"Total Miles Driven", stats.TotalMiles},
		{"Estimated Fuel Cost", fmt.Sprintf("$%.2f", stats.EstimatedCost)},
		{"Average Miles per Vehicle", stats.AvgMilesPerVehicle},
		{"Cost per Mile", fmt.Sprintf("$%.2f", stats.CostPerMile)},
		{"Vehicle Utilization", fmt.Sprintf("%.1f%%", stats.VehicleUtilization)},
	}
	
	for i, row := range summaryData {
		for j, value := range row {
			cell := fmt.Sprintf("%s%d", string(rune('A'+j)), i+2)
			f.SetCellValue(summarySheet, cell, value)
		}
	}
	
	// Vehicle Details Sheet
	if reportType == "all" || reportType == "agency" {
		sheetName := "Agency Vehicles"
		f.NewSheet(sheetName)
		
		// Headers
		headers := []string{"Vehicle ID", "Year", "Make/Model", "License", "Location", 
						  "Beginning Miles", "Ending Miles", "Total Miles", "Status"}
		for i, header := range headers {
			cell := fmt.Sprintf("%s1", string(rune('A'+i)))
			f.SetCellValue(sheetName, cell, header)
		}
		
		// Get data
		vehicles, _ := getAgencyVehicleReports(month, year)
		for i, v := range vehicles {
			row := i + 2
			f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), v.VehicleID)
			f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), v.VehicleYear)
			f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), v.MakeModel)
			f.SetCellValue(sheetName, fmt.Sprintf("D%d", row), v.LicensePlate)
			f.SetCellValue(sheetName, fmt.Sprintf("E%d", row), v.Location)
			f.SetCellValue(sheetName, fmt.Sprintf("F%d", row), v.BeginningMiles)
			f.SetCellValue(sheetName, fmt.Sprintf("G%d", row), v.EndingMiles)
			f.SetCellValue(sheetName, fmt.Sprintf("H%d", row), v.TotalMiles)
			f.SetCellValue(sheetName, fmt.Sprintf("I%d", row), v.Status)
		}
	}
	
	// Set response headers
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=mileage_report_%s_%s.xlsx", month, year))
	
	// Write file
	if err := f.Write(w); err != nil {
		http.Error(w, "Failed to generate Excel file", http.StatusInternalServerError)
		return
	}
}

// calculateMileageStatistics calculates comprehensive mileage statistics
func calculateMileageStatistics(month string, year int) *MileageStatistics {
	stats := &MileageStatistics{
		CostPerMile: 0.55, // Default IRS rate, should be configurable
	}
	
	// Get all vehicle data
	agencyVehicles, _ := getAgencyVehicleReports(month, fmt.Sprintf("%d", year))
	schoolBuses, _ := getSchoolBusReports(month, fmt.Sprintf("%d", year))
	
	// Calculate totals
	stats.TotalVehicles = len(agencyVehicles) + len(schoolBuses)
	
	for _, v := range agencyVehicles {
		stats.TotalMiles += v.TotalMiles
		if v.Status == "active" {
			stats.ActiveVehicles++
		}
	}
	
	for _, b := range schoolBuses {
		stats.TotalMiles += b.TotalMiles
		if b.Status == "active" {
			stats.ActiveVehicles++
		}
	}
	
	// Calculate derived statistics
	stats.EstimatedCost = float64(stats.TotalMiles) * stats.CostPerMile
	
	if stats.TotalVehicles > 0 {
		stats.AvgMilesPerVehicle = stats.TotalMiles / stats.TotalVehicles
		stats.VehicleUtilization = (float64(stats.ActiveVehicles) / float64(stats.TotalVehicles)) * 100
	}
	
	// Get driver statistics
	stats.DriverStats = calculateDriverStatistics(month, year)
	stats.RouteStats = calculateRouteStatistics(month, year)
	
	return stats
}

// calculateDriverStatistics aggregates driver performance data
func calculateDriverStatistics(month string, year int) []DriverStatistic {
	var stats []DriverStatistic
	
	// Get all drivers
	drivers := loadDrivers()
	
	for _, driver := range drivers {
		stat := DriverStatistic{
			DriverName: driver.Username,
		}
		
		// Get driver logs for the month
		startDate := fmt.Sprintf("%d-%02d-01", year, monthToNumber(month))
		endDate := fmt.Sprintf("%d-%02d-31", year, monthToNumber(month))
		
		logs, _ := getDriverLogsByDateRange(driver.Username, startDate, endDate)
		
		stat.TotalTrips = len(logs)
		
		for _, log := range logs {
			stat.TotalMiles += int(log.Mileage)
			
			// Calculate attendance
			presentCount := 0
			for _, attendance := range log.Attendance {
				if attendance.Present {
					presentCount++
				}
			}
			stat.StudentsTransported += presentCount
		}
		
		if stat.TotalTrips > 0 {
			stat.AvgMilesPerTrip = stat.TotalMiles / stat.TotalTrips
			stat.AttendanceRate = 95.0 // This should be calculated from actual data
			stat.EfficiencyScore = calculateEfficiencyScore(stat)
		}
		
		stats = append(stats, stat)
	}
	
	return stats
}

// calculateRouteStatistics calculates route performance metrics
func calculateRouteStatistics(month string, year int) []RouteStatistic {
	var stats []RouteStatistic
	
	// Get all routes
	routes, _ := loadRoutes()
	
	for _, route := range routes {
		stat := RouteStatistic{
			RouteName: route.RouteName,
		}
		
		// Get logs for this route
		startDate := fmt.Sprintf("%d-%02d-01", year, monthToNumber(month))
		endDate := fmt.Sprintf("%d-%02d-31", year, monthToNumber(month))
		
		// Query driver logs for this route
		rows, err := db.Query(`
			SELECT mileage, attendance 
			FROM driver_logs 
			WHERE route_id = $1 AND date BETWEEN $2 AND $3
		`, route.RouteID, startDate, endDate)
		
		if err == nil {
			defer rows.Close()
			
			totalStudents := 0
			for rows.Next() {
				var mileage float64
				var attendanceJSON []byte
				
				if err := rows.Scan(&mileage, &attendanceJSON); err == nil {
					stat.TotalMiles += int(mileage)
					stat.TotalRuns++
					
					// Count students
					var attendance []StudentAttendance
					if json.Unmarshal(attendanceJSON, &attendance) == nil {
						for _, a := range attendance {
							if a.Present {
								totalStudents++
							}
						}
					}
				}
			}
			
			if stat.TotalRuns > 0 {
				stat.AvgStudents = totalStudents / stat.TotalRuns
				stat.Efficiency = 85.0 // Calculate based on actual metrics
				if stat.AvgStudents > 0 {
					stat.CostPerStudent = (float64(stat.TotalMiles) * 0.55) / float64(totalStudents)
				}
			}
		}
		
		stats = append(stats, stat)
	}
	
	return stats
}

// monthToNumber converts month name to number
func monthToNumber(month string) int {
	months := map[string]int{
		"January":   1,
		"February":  2,
		"March":     3,
		"April":     4,
		"May":       5,
		"June":      6,
		"July":      7,
		"August":    8,
		"September": 9,
		"October":   10,
		"November":  11,
		"December":  12,
	}
	
	if num, ok := months[month]; ok {
		return num
	}
	return 1 // Default to January
}

// calculateEfficiencyScore calculates driver efficiency based on metrics
func calculateEfficiencyScore(stat DriverStatistic) int {
	score := 100
	
	// Deduct points for low attendance rate
	if stat.AttendanceRate < 90 {
		score -= 10
	}
	
	// Deduct points for excessive miles per trip
	if stat.AvgMilesPerTrip > 50 {
		score -= 5
	}
	
	// Bonus points for high student transport
	if stat.StudentsTransported > 100 {
		score += 5
	}
	
	// Keep score within bounds
	if score > 100 {
		score = 100
	} else if score < 0 {
		score = 0
	}
	
	return score
}

// driverDashboardHandler serves the driver dashboard
func driverDashboardHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// Ensure drivers go to driver dashboard
	if user.Role != "driver" {
		log.Printf("Non-driver user %s trying to access driver dashboard, redirecting", user.Username)
		http.Redirect(w, r, "/manager-dashboard", http.StatusSeeOther)
		return
	}

	// Get date and period from query params
	date := r.URL.Query().Get("date")
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}
	
	period := r.URL.Query().Get("period")
	if period == "" {
		period = "morning"
	}

	// Get driver's assignment
	assignment := getDriverAssignment(user.Username)
	
	// Get bus details if assigned
	var bus *Bus
	if assignment.BusID != "" {
		bus = getBusByID(assignment.BusID)
	}
	
	// Get route details if assigned
	var route *Route
	if assignment.RouteID != "" {
		route = getRouteByID(assignment.RouteID)
	}
	
	// Get students for the route ordered by pickup/dropoff time
	students := []Student{}
	if assignment.RouteID != "" {
		if period == "morning" {
			students = getRouteStudentsOrderedByPickup(assignment.RouteID)
		} else {
			students = getRouteStudentsOrderedByDropoff(assignment.RouteID)
		}
	}
	
	// Get driver's log for this date/period if exists
	var driverLog DriverLog
	var attendanceJSON sql.NullString
	err := db.QueryRow(`
		SELECT id, driver, bus_id, route_id, date, period, 
		       departure_time, arrival_time, mileage, attendance
		FROM driver_logs
		WHERE driver = $1 AND date = $2 AND period = $3
	`, user.Username, date, period).Scan(
		&driverLog.ID, &driverLog.Driver, 
		&driverLog.BusID, &driverLog.RouteID, &driverLog.Date, 
		&driverLog.Period, &driverLog.Departure, &driverLog.Arrival, 
		&driverLog.Mileage, &attendanceJSON)
	
	var driverLogPtr *DriverLog
	if err == nil {
		driverLogPtr = &driverLog
		if attendanceJSON.Valid && attendanceJSON.String != "" {
			json.Unmarshal([]byte(attendanceJSON.String), &driverLog.Attendance)
		}
	}
	
	// Get recent logs
	recentLogs := getDriverLogs(user.Username, 7)

	// Create data structure for driver dashboard template
	data := map[string]interface{}{
		"Data": map[string]interface{}{
			"User":       user,
			"Assignment": assignment,
			"Bus":        bus,
			"Route":      route,
			"Students":   students,
			"DriverLog":  driverLogPtr,
			"RecentLogs": recentLogs,
			"Date":       date,
			"Period":     period,
			"Today":      time.Now().Format("2006-01-02"),
			"CSRFToken":  getSessionCSRFToken(r),
		},
		"CSPNonce": getCSPNonce(r),
	}

	renderTemplate(w, r, "driver_dashboard.html", data)
}

// saveLogHandler saves a driver's log
func saveLogHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		if !validateCSRF(r) {
			http.Error(w, "Invalid CSRF token", http.StatusForbidden)
			return
		}

		user := getUserFromSession(r)
		if user == nil || user.Role != "driver" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Parse form data
		log := DriverLog{
			Driver:    user.Username,
			Date:      r.FormValue("date"),
			Period:    r.FormValue("period"),
			RouteID:   r.FormValue("route_id"),
			BusID:     r.FormValue("bus_id"),
			Departure: r.FormValue("departure"),
			Arrival:   r.FormValue("arrival"),
			Mileage:   parseFloatOrDefault(r.FormValue("mileage"), 0),
		}

		// Parse attendance
		var attendance []StudentAttendance
		for key, values := range r.Form {
			if strings.HasPrefix(key, "present_") {
				positionStr := strings.TrimPrefix(key, "present_")
				position := parseIntOrDefault(positionStr, 0)
				if position > 0 && len(values) > 0 {
					attendance = append(attendance, StudentAttendance{
						Position: position,
						Present:  true,
					})
				}
			}
		}

		// Save to database
		attendanceJSON, _ := json.Marshal(attendance)
		_, err := db.Exec(`
			INSERT INTO driver_logs (driver, date, period, route_id, bus_id, departure_time, arrival_time, mileage, attendance)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			ON CONFLICT (driver, date, period) DO UPDATE 
			SET route_id = $4, bus_id = $5, departure_time = $6, arrival_time = $7, mileage = $8, attendance = $9
		`, log.Driver, log.Date, log.Period, log.RouteID, log.BusID, log.Departure, log.Arrival, log.Mileage, string(attendanceJSON))

		if err != nil {
			http.Error(w, "Failed to save log", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/driver-dashboard", http.StatusSeeOther)
	}
}

// importECSEHandler handles ECSE report imports - FIXED with proper CSRF handling
func importECSEHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	if r.Method == "GET" {
		renderTemplate(w, r, "import_ecse.html", map[string]interface{}{
			"CSRFToken": getSessionCSRFToken(r),
		})
		return
	}

	if r.Method == "POST" {
		// IMPORTANT: Parse multipart form FIRST
		err := r.ParseMultipartForm(10 << 20) // 10 MB
		if err != nil {
			renderTemplate(w, r, "import_ecse.html", map[string]interface{}{
				"Error":     "Failed to parse form",
				"CSRFToken": getSessionCSRFToken(r),
			})
			return
		}

		// NOW validate CSRF after parsing the multipart form
		if !validateCSRF(r) {
			renderTemplate(w, r, "import_ecse.html", map[string]interface{}{
				"Error":     "Invalid CSRF token. Please try again.",
				"CSRFToken": getSessionCSRFToken(r),
			})
			return
		}

		file, header, err := r.FormFile("excel_file")
		if err != nil {
			renderTemplate(w, r, "import_ecse.html", map[string]interface{}{
				"Error":     "Failed to get file",
				"CSRFToken": getSessionCSRFToken(r),
			})
			return
		}
		defer file.Close()

		// Process ECSE Excel file
		imported, err := processECSEExcelFile(file, header.Filename)
		
		if err != nil {
			renderTemplate(w, r, "import_ecse.html", map[string]interface{}{
				"Error":     fmt.Sprintf("Import failed: %v", err),
				"CSRFToken": getSessionCSRFToken(r),
			})
			return
		}

		renderTemplate(w, r, "import_ecse.html", map[string]interface{}{
			"Success":   fmt.Sprintf("Successfully imported %d ECSE student records", imported),
			"CSRFToken": getSessionCSRFToken(r),
		})
	}
}

// viewECSEReportsHandler shows ECSE reports and data
func viewECSEReportsHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	// Get query parameters for filtering
	enrollmentStatus := r.URL.Query().Get("status")
	transportationOnly := r.URL.Query().Get("transportation") == "true"
	searchTerm := r.URL.Query().Get("search")

	// Build query
	query := `
		SELECT 
			s.student_id, s.first_name, s.last_name, s.date_of_birth::text,
			s.grade, s.enrollment_status, s.iep_status, s.primary_disability,
			s.service_minutes, s.transportation_required, s.bus_route,
			s.parent_name, s.parent_phone, s.parent_email,
			s.address, s.city, s.state, s.zip_code,
			COUNT(DISTINCT srv.id) as service_count,
			COUNT(DISTINCT a.id) as assessment_count
		FROM ecse_students s
		LEFT JOIN ecse_services srv ON s.student_id = srv.student_id
		LEFT JOIN ecse_assessments a ON s.student_id = a.student_id
		WHERE 1=1
	`
	
	args := []interface{}{}
	argCount := 0

	if enrollmentStatus != "" && enrollmentStatus != "all" {
		argCount++
		query += fmt.Sprintf(" AND s.enrollment_status = $%d", argCount)
		args = append(args, enrollmentStatus)
	}

	if transportationOnly {
		query += " AND s.transportation_required = true"
	}

	if searchTerm != "" {
		argCount++
		query += fmt.Sprintf(" AND (LOWER(s.first_name) LIKE LOWER($%d) OR LOWER(s.last_name) LIKE LOWER($%d) OR s.student_id LIKE $%d)", argCount, argCount, argCount)
		args = append(args, "%"+searchTerm+"%")
	}

	query += `
		GROUP BY s.student_id, s.first_name, s.last_name, s.date_of_birth,
			s.grade, s.enrollment_status, s.iep_status, s.primary_disability,
			s.service_minutes, s.transportation_required, s.bus_route,
			s.parent_name, s.parent_phone, s.parent_email,
			s.address, s.city, s.state, s.zip_code
		ORDER BY s.last_name, s.first_name
	`

	rows, err := db.Query(query, args...)
	
	var students []ECSEStudentView
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var student ECSEStudentView
			var dob sql.NullString
			
			err := rows.Scan(
				&student.StudentID, &student.FirstName, &student.LastName, &dob,
				&student.Grade, &student.EnrollmentStatus, &student.IEPStatus, &student.PrimaryDisability,
				&student.ServiceMinutes, &student.TransportationRequired, &student.BusRoute,
				&student.ParentName, &student.ParentPhone, &student.ParentEmail,
				&student.Address, &student.City, &student.State, &student.ZipCode,
				&student.ServiceCount, &student.AssessmentCount,
			)
			
			if err == nil {
				if dob.Valid {
					student.DateOfBirth = dob.String
				}
				students = append(students, student)
			}
		}
	}

	// Get summary statistics WITH PERCENTAGES
	stats := getECSEStatisticsWithPercentages()

	renderTemplate(w, r, "view_ecse_reports.html", map[string]interface{}{
		"User":               user,
		"Students":           students,
		"Stats":              stats,
		"EnrollmentStatus":   enrollmentStatus,
		"TransportationOnly": transportationOnly,
		"SearchTerm":         searchTerm,
		"CSRFToken":          getSessionCSRFToken(r),
	})
}

// viewECSEStudentHandler shows detailed view of a single ECSE student
func viewECSEStudentHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	studentID := strings.TrimPrefix(r.URL.Path, "/ecse-student/")
	
	// Get student details
	var student ECSEStudent
	err := db.QueryRow(`
		SELECT student_id, first_name, last_name, date_of_birth::text, grade,
			enrollment_status, iep_status, primary_disability, service_minutes,
			transportation_required, bus_route, parent_name, parent_phone,
			parent_email, address, city, state, zip_code, notes
		FROM ecse_students
		WHERE student_id = $1
	`, studentID).Scan(
		&student.StudentID, &student.FirstName, &student.LastName, &student.DateOfBirth,
		&student.Grade, &student.EnrollmentStatus, &student.IEPStatus, &student.PrimaryDisability,
		&student.ServiceMinutes, &student.TransportationRequired, &student.BusRoute,
		&student.ParentName, &student.ParentPhone, &student.ParentEmail,
		&student.Address, &student.City, &student.State, &student.ZipCode, &student.Notes,
	)
	
	if err != nil {
		http.Error(w, "Student not found", http.StatusNotFound)
		return
	}

	// Get services
	services := []ECSEService{}
	rows, err := db.Query(`
		SELECT id, service_type, frequency, duration, provider,
			start_date::text, end_date::text, goals, progress
		FROM ecse_services
		WHERE student_id = $1
		ORDER BY service_type
	`, studentID)
	
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var service ECSEService
			rows.Scan(&service.ID, &service.ServiceType, &service.Frequency,
				&service.Duration, &service.Provider, &service.StartDate,
				&service.EndDate, &service.Goals, &service.Progress)
			services = append(services, service)
		}
	}

	// Get assessments
	assessments := []ECSEAssessment{}
	rows, err = db.Query(`
		SELECT id, assessment_type, assessment_date::text, score,
			evaluator, notes, next_review_date::text
		FROM ecse_assessments
		WHERE student_id = $1
		ORDER BY assessment_date DESC
	`, studentID)
	
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var assessment ECSEAssessment
			rows.Scan(&assessment.ID, &assessment.AssessmentType, &assessment.AssessmentDate,
				&assessment.Score, &assessment.Evaluator, &assessment.Notes, &assessment.NextReviewDate)
			assessments = append(assessments, assessment)
		}
	}

	// Get recent attendance
	attendance := []ECSEAttendanceRecord{}
	rows, err = db.Query(`
		SELECT attendance_date::text, status, arrival_time::text, departure_time::text, notes
		FROM ecse_attendance
		WHERE student_id = $1
		ORDER BY attendance_date DESC
		LIMIT 30
	`, studentID)
	
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var record ECSEAttendanceRecord
			var arrival, departure sql.NullString
			rows.Scan(&record.Date, &record.Status, &arrival, &departure, &record.Notes)
			
			if arrival.Valid {
				record.ArrivalTime = arrival.String
			}
			if departure.Valid {
				record.DepartureTime = departure.String
			}
			
			attendance = append(attendance, record)
		}
	}

	renderTemplate(w, r, "view_ecse_student.html", map[string]interface{}{
		"User":        user,
		"Student":     student,
		"Services":    services,
		"Assessments": assessments,
		"Attendance":  attendance,
		"CSRFToken":   getSessionCSRFToken(r),
	})
}

// exportECSEHandler exports ECSE data to CSV
func exportECSEHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	// Set CSV headers
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=ecse_export_"+time.Now().Format("20060102")+".csv")

	// Get filter parameters
	enrollmentStatus := r.URL.Query().Get("status")
	transportationOnly := r.URL.Query().Get("transportation") == "true"
	searchTerm := r.URL.Query().Get("search")

	// Build query (similar to view handler)
	query := `
		SELECT 
			s.student_id, s.first_name, s.last_name, s.date_of_birth::text,
			s.grade, s.enrollment_status, s.iep_status, s.primary_disability,
			s.service_minutes, s.transportation_required, s.bus_route,
			s.parent_name, s.parent_phone, s.parent_email,
			s.address, s.city, s.state, s.zip_code
		FROM ecse_students s
		WHERE 1=1
	`
	
	args := []interface{}{}
	argCount := 0

	if enrollmentStatus != "" && enrollmentStatus != "all" {
		argCount++
		query += fmt.Sprintf(" AND s.enrollment_status = $%d", argCount)
		args = append(args, enrollmentStatus)
	}

	if transportationOnly {
		query += " AND s.transportation_required = true"
	}

	if searchTerm != "" {
		argCount++
		query += fmt.Sprintf(" AND (LOWER(s.first_name) LIKE LOWER($%d) OR LOWER(s.last_name) LIKE LOWER($%d) OR s.student_id LIKE $%d)", argCount, argCount, argCount)
		args = append(args, "%"+searchTerm+"%")
	}

	query += " ORDER BY s.last_name, s.first_name"

	rows, err := db.Query(query, args...)
	if err != nil {
		http.Error(w, "Failed to export data", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// Write CSV header
	fmt.Fprintf(w, "Student ID,First Name,Last Name,Date of Birth,Grade,Status,IEP Status,Primary Disability,")
	fmt.Fprintf(w, "Service Minutes,Transportation Required,Bus Route,Parent Name,Parent Phone,Parent Email,")
	fmt.Fprintf(w, "Address,City,State,Zip Code\n")

	// Write data rows
	for rows.Next() {
		var s struct {
			StudentID              string
			FirstName              string
			LastName               string
			DateOfBirth            sql.NullString
			Grade                  string
			EnrollmentStatus       string
			IEPStatus              sql.NullString
			PrimaryDisability      sql.NullString
			ServiceMinutes         int
			TransportationRequired bool
			BusRoute               sql.NullString
			ParentName             sql.NullString
			ParentPhone            sql.NullString
			ParentEmail            sql.NullString
			Address                sql.NullString
			City                   sql.NullString
			State                  sql.NullString
			ZipCode                sql.NullString
		}
		
		err := rows.Scan(&s.StudentID, &s.FirstName, &s.LastName, &s.DateOfBirth,
			&s.Grade, &s.EnrollmentStatus, &s.IEPStatus, &s.PrimaryDisability,
			&s.ServiceMinutes, &s.TransportationRequired, &s.BusRoute,
			&s.ParentName, &s.ParentPhone, &s.ParentEmail,
			&s.Address, &s.City, &s.State, &s.ZipCode)
		
		if err != nil {
			continue
		}
		
		// Write CSV row
		fmt.Fprintf(w, "%s,%s,%s,%s,%s,%s,%s,%s,%d,%t,%s,%s,%s,%s,%s,%s,%s,%s\n",
			s.StudentID, s.FirstName, s.LastName, 
			s.DateOfBirth.String, s.Grade, s.EnrollmentStatus,
			s.IEPStatus.String, s.PrimaryDisability.String,
			s.ServiceMinutes, s.TransportationRequired, s.BusRoute.String,
			s.ParentName.String, s.ParentPhone.String, s.ParentEmail.String,
			s.Address.String, s.City.String, s.State.String, s.ZipCode.String)
	}
}

// fleetHandler shows the bus fleet
func fleetHandler(w http.ResponseWriter, r *http.Request) {
	if !isLoggedIn(r) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	user := getUser(r)
	if user.Role != "manager" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	buses := loadBuses()
	
	// Get recent maintenance logs
	maintenanceLogs := []BusMaintenanceLog{}
	rows, err := db.Query(`
		SELECT bus_id, date, category, notes, mileage, created_at
		FROM bus_maintenance_logs
		ORDER BY date DESC
		LIMIT 10
	`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var log BusMaintenanceLog
			err := rows.Scan(&log.BusID, &log.Date, &log.Category, 
				&log.Notes, &log.Mileage, &log.CreatedAt)
			if err == nil {
				maintenanceLogs = append(maintenanceLogs, log)
			}
		}
	} else {
		log.Printf("Error loading maintenance logs: %v", err)
	}

	data := map[string]interface{}{
		"User":            user,
		"Buses":           buses,
		"MaintenanceLogs": maintenanceLogs,
		"Today":           time.Now().Format("2006-01-02"),
		"CSRFToken":       getSessionCSRFToken(r),
	}
	renderTemplate(w, r, "fleet.html", data)
}

// companyFleetHandler shows company vehicles
func companyFleetHandler(w http.ResponseWriter, r *http.Request) {
	vehicles := []Vehicle{}
	rows, err := db.Query(`
		SELECT vehicle_id, model, year, description, status, oil_status, tire_status, maintenance_notes
		FROM vehicles
		ORDER BY vehicle_id
	`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var v Vehicle
			rows.Scan(&v.VehicleID, &v.Model, &v.Year, &v.Description, &v.Status, &v.OilStatus, &v.TireStatus, &v.MaintenanceNotes)
			vehicles = append(vehicles, v)
		}
	}

	renderTemplate(w, r, "company_fleet.html", CompanyFleetData{
		User:      getUserFromSession(r),
		Vehicles:  vehicles,
		CSRFToken: getSessionCSRFToken(r),
	})
}

// updateVehicleStatusHandler updates vehicle status
func updateVehicleStatusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		// For AJAX requests, parse form data first
		r.ParseForm()
		
		if !validateCSRF(r) {
			http.Error(w, "Invalid CSRF token", http.StatusForbidden)
			return
		}

		vehicleID := r.FormValue("vehicle_id")
		statusType := r.FormValue("status_type")
		newStatus := r.FormValue("new_status")

		// Map status types to database columns
		fieldMap := map[string]string{
			"status": "status",
			"oil":    "oil_status",
			"tire":   "tire_status",
		}

		if dbField, ok := fieldMap[statusType]; ok {
			// First try buses table (numeric IDs)
			result, err := db.Exec(fmt.Sprintf("UPDATE buses SET %s = $1, updated_at = NOW() WHERE bus_id = $2", dbField), newStatus, vehicleID)
			if err != nil {
				log.Printf("Error updating bus: %v", err)
			}
			
			rowsAffected, _ := result.RowsAffected()
			
			// If no bus was updated, try vehicles table
			if rowsAffected == 0 {
				_, err = db.Exec(fmt.Sprintf("UPDATE vehicles SET %s = $1, updated_at = NOW() WHERE vehicle_id = $2", dbField), newStatus, vehicleID)
				if err != nil {
					log.Printf("Error updating vehicle: %v", err)
					http.Error(w, "Failed to update status", http.StatusInternalServerError)
					return
				}
			}
		}

		// Return JSON response
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status": "success",
			"message": "Status updated successfully",
		})
	}
}

// busMaintenanceHandler shows maintenance history for a bus
func busMaintenanceHandler(w http.ResponseWriter, r *http.Request) {
	busID := strings.TrimPrefix(r.URL.Path, "/bus-maintenance/")
	
	// Get maintenance records
	records := []MaintenanceRecord{}
	query := `
		SELECT bus_id, date, category, mileage, 0 as cost, notes, created_at
		FROM bus_maintenance_logs
		WHERE bus_id = $1
		ORDER BY date DESC
	`
	
	rows, err := db.Query(query, busID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var record MaintenanceRecord
			rows.Scan(&record.VehicleID, &record.Date, &record.Category, 
				&record.Mileage, &record.Cost, &record.Notes, &record.CreatedAt)
			records = append(records, record)
		}
	}

	// Calculate statistics
	var totalCost float64
	recentCount := 0
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
	
	for _, record := range records {
		totalCost += record.Cost
		// Parse the date string to compare
		if recordDate, err := time.Parse("2006-01-02", record.Date); err == nil {
			if recordDate.After(thirtyDaysAgo) {
				recentCount++
			}
		}
	}

	totalRecords := len(records)
	averageCost := float64(0)
	if totalRecords > 0 {
		averageCost = totalCost / float64(totalRecords)
	}

	// Create data map for template - use same template as vehicle maintenance
	data := map[string]interface{}{
		"VehicleID":          busID,
		"IsBus":              true,
		"MaintenanceRecords": records,
		"TotalRecords":       totalRecords,
		"TotalCost":          totalCost,
		"AverageCost":        averageCost,
		"RecentCount":        recentCount,
		"Today":              time.Now().Format("2006-01-02"),
		"CSRFToken":          getSessionCSRFToken(r),
	}

	renderTemplate(w, r, "vehicle_maintenance.html", data)
}

// vehicleMaintenanceHandler shows maintenance for any vehicle
func vehicleMaintenanceHandler(w http.ResponseWriter, r *http.Request) {
	vehicleID := strings.TrimPrefix(r.URL.Path, "/vehicle-maintenance/")
	
	// Try to determine if it's a bus by checking buses table
	var isBus bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM buses WHERE bus_id = $1)", vehicleID).Scan(&isBus)
	if err != nil {
		isBus = false
	}

	// Get maintenance records from appropriate table
	records := []MaintenanceRecord{}
	
	if isBus {
		// Query bus_maintenance_logs
		query := `
			SELECT bus_id, date, category, mileage, 
			       COALESCE(cost, 0) as cost, notes, created_at
			FROM bus_maintenance_logs
			WHERE bus_id = $1
			ORDER BY date DESC
		`
		rows, err := db.Query(query, vehicleID)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var record MaintenanceRecord
				rows.Scan(&record.VehicleID, &record.Date, &record.Category, 
					&record.Mileage, &record.Cost, &record.Notes, &record.CreatedAt)
				records = append(records, record)
			}
		}
	} else {
		// Query maintenance_records for other vehicles
		query := `
			SELECT vehicle_id, date, 'maintenance' as category, mileage, 
			       COALESCE(cost, 0) as cost, work_description as notes, created_at
			FROM maintenance_records
			WHERE vehicle_id = $1
			ORDER BY date DESC
		`
		rows, err := db.Query(query, vehicleID)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var record MaintenanceRecord
				rows.Scan(&record.VehicleID, &record.Date, &record.Category, 
					&record.Mileage, &record.Cost, &record.Notes, &record.CreatedAt)
				records = append(records, record)
			}
		}
	}

	// Calculate statistics
	var totalCost float64
	recentCount := 0
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
	
	for _, record := range records {
		totalCost += record.Cost
		// Parse the date string to compare
		if recordDate, err := time.Parse("2006-01-02", record.Date); err == nil {
			if recordDate.After(thirtyDaysAgo) {
				recentCount++
			}
		}
	}

	totalRecords := len(records)
	averageCost := float64(0)
	if totalRecords > 0 {
		averageCost = totalCost / float64(totalRecords)
	}

	data := map[string]interface{}{
		"VehicleID":          vehicleID,
		"IsBus":              isBus,
		"MaintenanceRecords": records,
		"TotalRecords":       totalRecords,
		"TotalCost":          totalCost,
		"AverageCost":        averageCost,
		"RecentCount":        recentCount,
		"Today":              time.Now().Format("2006-01-02"),
		"CSRFToken":          getSessionCSRFToken(r),
	}

	renderTemplate(w, r, "vehicle_maintenance.html", data)
}

// saveMaintenanceRecordHandler saves a maintenance record
func saveMaintenanceRecordHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		// Parse form data
		r.ParseForm()
		
		if !validateCSRF(r) {
			http.Error(w, "Invalid CSRF token", http.StatusForbidden)
			return
		}

		vehicleID := r.FormValue("vehicle_id")
		date := r.FormValue("date")
		category := r.FormValue("category")
		mileage := parseIntOrDefault(r.FormValue("mileage"), 0)
		notes := r.FormValue("notes")

		// Save to bus_maintenance_logs table
		_, err := db.Exec(`
			INSERT INTO bus_maintenance_logs (bus_id, date, category, mileage, notes, created_at)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, vehicleID, date, category, mileage, notes, time.Now())

		if err != nil {
			http.Error(w, "Failed to save maintenance record", http.StatusInternalServerError)
			return
		}

		// Return JSON response for AJAX requests
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status": "success",
			"message": "Maintenance record saved successfully",
		})
	}
}

// assignRoutesHandler shows route assignment page
func assignRoutesHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get all assignments
	assignments, _ := loadRouteAssignments()
	
	// Get all drivers
	drivers := loadDrivers()
	
	// Get all routes with assignment status
	allRoutes, _ := loadRoutes()
	
	// Get all buses
	allBuses := loadBuses()

	// Create map of assigned bus IDs and route IDs
	assignedBusIDs := make(map[string]bool)
	assignedRouteIDs := make(map[string]bool)
	for _, assignment := range assignments {
		assignedBusIDs[assignment.BusID] = true
		assignedRouteIDs[assignment.RouteID] = true
	}

	// Filter available buses (active and not assigned)
	availableBuses := []*Bus{}
	for _, bus := range allBuses {
		if bus.Status == "active" && !assignedBusIDs[bus.BusID] {
			availableBuses = append(availableBuses, bus)
		}
	}

	// Filter available routes (not assigned)
	availableRoutes := []*Route{}
	routesWithStatus := []*RouteWithStatus{}
	
	// Process routes correctly
	for i := range allRoutes {
		route := &allRoutes[i]
		isAssigned := assignedRouteIDs[route.RouteID]
		
		// Add to routesWithStatus for display
		routesWithStatus = append(routesWithStatus, &RouteWithStatus{
			Route:      *route,
			IsAssigned: isAssigned,
		})
		
		// Add to availableRoutes if not assigned
		if !isAssigned {
			availableRoutes = append(availableRoutes, route)
		}
	}

	// Count available drivers (not assigned)
	availableDriversCount := 0
	assignedDrivers := make(map[string]bool)
	for _, assignment := range assignments {
		assignedDrivers[assignment.Driver] = true
	}
	for _, driver := range drivers {
		if !assignedDrivers[driver.Username] {
			availableDriversCount++
		}
	}

	// Convert routes to pointers for data.Routes
	routePtrs := make([]*Route, len(allRoutes))
	for i := range allRoutes {
		routePtrs[i] = &allRoutes[i]
	}

	data := map[string]interface{}{
		"User":                  user,
		"Assignments":           assignments,
		"Drivers":               drivers,
		"Routes":                routePtrs,
		"RoutesWithStatus":      routesWithStatus,
		"AvailableRoutes":       availableRoutes,
		"Buses":                 allBuses,
		"AvailableBuses":        availableBuses,
		"TotalAssignments":      len(assignments),
		"TotalRoutes":           len(allRoutes),
		"AvailableDriversCount": availableDriversCount,
		"AvailableBusesCount":   len(availableBuses),
		"CSRFToken":             getSessionCSRFToken(r),
	}

	renderTemplate(w, r, "assign_routes.html", data)
}

// assignRouteHandler assigns a route to a driver
func assignRouteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		if !validateCSRF(r) {
			http.Error(w, "Invalid CSRF token", http.StatusForbidden)
			return
		}

		driver := r.FormValue("driver")
		routeID := r.FormValue("route_id")
		busID := r.FormValue("bus_id")

		// Delete existing assignment for this driver
		db.Exec(`DELETE FROM route_assignments WHERE driver = $1`, driver)

		// Create new assignment
		_, err := db.Exec(`
			INSERT INTO route_assignments (driver, route_id, bus_id, assigned_date)
			VALUES ($1, $2, $3, $4)
		`, driver, routeID, busID, time.Now().Format("2006-01-02"))

		if err != nil {
			http.Error(w, "Failed to assign route", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/assign-routes", http.StatusSeeOther)
	}
}

// unassignRouteHandler removes a route assignment
func unassignRouteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		if !validateCSRF(r) {
			http.Error(w, "Invalid CSRF token", http.StatusForbidden)
			return
		}

		driver := r.FormValue("driver")
		db.Exec(`DELETE FROM route_assignments WHERE driver = $1`, driver)
		http.Redirect(w, r, "/assign-routes", http.StatusSeeOther)
	}
}

// addRouteHandler adds a new route
func addRouteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		if !validateCSRF(r) {
			http.Error(w, "Invalid CSRF token", http.StatusForbidden)
			return
		}

		routeName := r.FormValue("route_name")
		description := r.FormValue("description")

		// Generate route ID
		routeID := fmt.Sprintf("ROUTE%03d", time.Now().Unix()%1000)

		_, err := db.Exec(`
			INSERT INTO routes (route_id, route_name, description)
			VALUES ($1, $2, $3)
		`, routeID, routeName, description)

		if err != nil {
			http.Error(w, "Failed to add route", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/assign-routes", http.StatusSeeOther)
	}
}

// editRouteHandler edits a route
func editRouteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		if !validateCSRF(r) {
			http.Error(w, "Invalid CSRF token", http.StatusForbidden)
			return
		}

		routeID := r.FormValue("route_id")
		routeName := r.FormValue("route_name")
		description := r.FormValue("description")

		db.Exec(`
			UPDATE routes 
			SET route_name = $1, description = $2
			WHERE route_id = $3
		`, routeName, description, routeID)

		http.Redirect(w, r, "/assign-routes", http.StatusSeeOther)
	}
}

// deleteRouteHandler deletes a route
func deleteRouteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		if !validateCSRF(r) {
			http.Error(w, "Invalid CSRF token", http.StatusForbidden)
			return
		}

		routeID := r.FormValue("route_id")
		
		// Delete route and related data
		db.Exec(`DELETE FROM route_assignments WHERE route_id = $1`, routeID)
		db.Exec(`DELETE FROM students WHERE route_id = $1`, routeID)
		db.Exec(`DELETE FROM routes WHERE route_id = $1`, routeID)

		http.Redirect(w, r, "/assign-routes", http.StatusSeeOther)
	}
}

// studentsHandler shows student management page
func studentsHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "driver" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get driver's route
	assignment := getDriverAssignment(user.Username)
	students := getRouteStudents(assignment.RouteID)

	renderTemplate(w, r, "students.html", StudentData{
		User:      user,
		Students:  students,
		CSRFToken: getSessionCSRFToken(r),
	})
}

// addStudentHandler adds a student
func addStudentHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		if !validateCSRF(r) {
			http.Error(w, "Invalid CSRF token", http.StatusForbidden)
			return
		}

		user := getUserFromSession(r)
		if user == nil || user.Role != "driver" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Get driver's route
		assignment := getDriverAssignment(user.Username)

		// Create student
		student := Student{
			StudentID:      generateUniqueID("STU", int(time.Now().Unix()%1000)),
			Name:           r.FormValue("name"),
			PhoneNumber:    r.FormValue("phone_number"),
			AltPhoneNumber: r.FormValue("alt_phone_number"),
			Guardian:       r.FormValue("guardian"),
			PickupTime:     r.FormValue("pickup_time"),
			DropoffTime:    r.FormValue("dropoff_time"),
			PositionNumber: parseIntOrDefault(r.FormValue("position_number"), 1),
			RouteID:        assignment.RouteID,
			Driver:         user.Username,
			Active:         true,
		}

		// Save to database
		_, err := db.Exec(`
			INSERT INTO students (student_id, name, phone_number, alt_phone_number, guardian, 
				pickup_time, dropoff_time, position_number, route_id, driver, active)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		`, student.StudentID, student.Name, student.PhoneNumber, student.AltPhoneNumber,
			student.Guardian, student.PickupTime, student.DropoffTime, student.PositionNumber,
			student.RouteID, student.Driver, student.Active)

		if err != nil {
			http.Error(w, "Failed to add student", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/students", http.StatusSeeOther)
	}
}

// editStudentHandler edits a student
func editStudentHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		if !validateCSRF(r) {
			http.Error(w, "Invalid CSRF token", http.StatusForbidden)
			return
		}

		studentID := r.FormValue("student_id")
		
		db.Exec(`
			UPDATE students 
			SET name = $1, phone_number = $2, alt_phone_number = $3, guardian = $4,
				pickup_time = $5, dropoff_time = $6, position_number = $7
			WHERE student_id = $8
		`, r.FormValue("name"), r.FormValue("phone_number"), r.FormValue("alt_phone_number"),
			r.FormValue("guardian"), r.FormValue("pickup_time"), r.FormValue("dropoff_time"),
			parseIntOrDefault(r.FormValue("position_number"), 1), studentID)

		http.Redirect(w, r, "/students", http.StatusSeeOther)
	}
}

// removeStudentHandler removes a student
func removeStudentHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		if !validateCSRF(r) {
			http.Error(w, "Invalid CSRF token", http.StatusForbidden)
			return
		}

		studentID := r.FormValue("student_id")
		db.Exec(`UPDATE students SET active = false WHERE student_id = $1`, studentID)
		http.Redirect(w, r, "/students", http.StatusSeeOther)
	}
}

// importMileageHandler handles mileage report imports - FIXED
func importMileageHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	if r.Method == "GET" {
		renderTemplate(w, r, "import_mileage.html", map[string]interface{}{
			"CSRFToken": getSessionCSRFToken(r),
		})
		return
	}

	if r.Method == "POST" {
		// Parse multipart form FIRST
		err := r.ParseMultipartForm(10 << 20) // 10 MB
		if err != nil {
			renderTemplate(w, r, "import_mileage.html", map[string]interface{}{
				"Error":     "Failed to parse form",
				"CSRFToken": getSessionCSRFToken(r),
			})
			return
		}

		// THEN validate CSRF
		if !validateCSRF(r) {
			renderTemplate(w, r, "import_mileage.html", map[string]interface{}{
				"Error":     "Invalid CSRF token. Please try again.",
				"CSRFToken": getSessionCSRFToken(r),
			})
			return
		}

		file, header, err := r.FormFile("file")
		if err != nil {
			renderTemplate(w, r, "import_mileage.html", map[string]interface{}{
				"Error":     "Failed to get file",
				"CSRFToken": getSessionCSRFToken(r),
			})
			return
		}
		defer file.Close()

		// Process Excel file
		imported, err := processEnhancedMileageExcelFile(file, header.Filename)
		
		if err != nil {
			renderTemplate(w, r, "import_mileage.html", map[string]interface{}{
				"Error":     fmt.Sprintf("Import failed: %v", err),
				"CSRFToken": getSessionCSRFToken(r),
			})
			return
		}

		renderTemplate(w, r, "import_mileage.html", map[string]interface{}{
			"Success":   fmt.Sprintf("Successfully imported %d mileage records", imported),
			"CSRFToken": getSessionCSRFToken(r),
		})
	}
}

// viewMileageReportsHandler shows mileage reports
func viewMileageReportsHandler(w http.ResponseWriter, r *http.Request) {
	// Use the enhanced mileage reports handler from database.go
	viewEnhancedMileageReportsHandler(w, r)
}

// driverProfileHandler shows driver profile
func driverProfileHandler(w http.ResponseWriter, r *http.Request) {
	driverUsername := strings.TrimPrefix(r.URL.Path, "/driver/")
	
	// Get driver info
	driver := getUserByUsername(driverUsername)
	if driver == nil || driver.Role != "driver" {
		http.Error(w, "Driver not found", http.StatusNotFound)
		return
	}

	// Get driver's assignment
	assignment := getDriverAssignment(driverUsername)
	
	// Get recent logs
	logs := getDriverLogs(driverUsername, 30)
	
	// Calculate statistics
	totalMiles := 0.0
	for _, log := range logs {
		totalMiles += log.Mileage
	}

	renderTemplate(w, r, "driver_profile.html", map[string]interface{}{
		"Driver":     driver,
		"Assignment": assignment,
		"Logs":       logs,
		"TotalMiles": totalMiles,
		"LogCount":   len(logs),
		"CSRFToken":  getSessionCSRFToken(r),
	})
}

// Helper functions

func authenticateUser(username, password string) (*User, error) {
	var user User
	err := db.QueryRow(`
		SELECT username, password, role, status, registration_date 
		FROM users 
		WHERE username = $1 AND status = 'active'
	`, username).Scan(&user.Username, &user.Password, &user.Role, &user.Status, &user.RegistrationDate)
	
	if err != nil {
		return nil, err
	}

	// Verify password
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func getUserByUsername(username string) *User {
	var user User
	err := db.QueryRow(`
		SELECT username, role, status, registration_date 
		FROM users 
		WHERE username = $1
	`, username).Scan(&user.Username, &user.Role, &user.Status, &user.RegistrationDate)
	
	if err != nil {
		return nil
	}
	
	return &user
}

func loadDrivers() []User {
	drivers := []User{}
	rows, err := db.Query(`SELECT username, role, status FROM users WHERE role = 'driver' AND status = 'active'`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var driver User
			rows.Scan(&driver.Username, &driver.Role, &driver.Status)
			drivers = append(drivers, driver)
		}
	}
	return drivers
}

func loadDriverSummaries() []*DriverSummary {
	summaries := []*DriverSummary{}
	rows, err := db.Query(`
		SELECT ra.driver, ra.bus_id, ra.route_id, r.route_name,
			   COALESCE(MAX(dl.created_at), NOW()) as last_activity,
			   COALESCE(SUM(dl.mileage), 0) as total_miles
		FROM route_assignments ra
		LEFT JOIN routes r ON ra.route_id = r.route_id
		LEFT JOIN driver_logs dl ON dl.driver = ra.driver
		GROUP BY ra.driver, ra.bus_id, ra.route_id, r.route_name
	`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var s DriverSummary
			rows.Scan(&s.Driver, &s.BusID, &s.RouteID, &s.RouteName, &s.LastActivity, &s.TotalMiles)
			summaries = append(summaries, &s)
		}
	}
	return summaries
}

func loadRouteStats() []*RouteStats {
	stats := []*RouteStats{}
	rows, err := db.Query(`
		SELECT r.route_id, r.route_name,
			   COUNT(DISTINCT ra.bus_id) as active_buses,
			   COUNT(DISTINCT s.student_id) as total_students
		FROM routes r
		LEFT JOIN route_assignments ra ON r.route_id = ra.route_id
		LEFT JOIN students s ON r.route_id = s.route_id AND s.active = true
		GROUP BY r.route_id, r.route_name
	`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var s RouteStats
			rows.Scan(&s.RouteID, &s.RouteName, &s.ActiveBuses, &s.TotalStudents)
			stats = append(stats, &s)
		}
	}
	return stats
}

func countPendingUsers() int {
	var count int
	db.QueryRow(`SELECT COUNT(*) FROM users WHERE status = 'pending'`).Scan(&count)
	return count
}

func getDriverAssignment(driver string) *DriverAssignment {
	var a DriverAssignment
	err := db.QueryRow(`
		SELECT ra.driver, ra.route_id, ra.bus_id, r.route_name
		FROM route_assignments ra
		LEFT JOIN routes r ON ra.route_id = r.route_id
		WHERE ra.driver = $1
	`, driver).Scan(&a.Driver, &a.RouteID, &a.BusID, &a.RouteName)
	
	if err != nil {
		return &DriverAssignment{Driver: driver}
	}
	
	return &a
}

func getDriverLogs(driver string, days int) []DriverLog {
	logs := []DriverLog{}
	rows, err := db.Query(`
		SELECT id, driver, bus_id, route_id, date, period, departure_time, arrival_time, mileage, attendance
		FROM driver_logs
		WHERE driver = $1 AND date >= $2
		ORDER BY date DESC, period DESC
	`, driver, time.Now().AddDate(0, 0, -days).Format("2006-01-02"))
	
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var log DriverLog
			var attendanceJSON sql.NullString
			rows.Scan(&log.ID, &log.Driver, &log.BusID, &log.RouteID, &log.Date, 
				&log.Period, &log.Departure, &log.Arrival, &log.Mileage, &attendanceJSON)
			
			// Parse attendance JSON
			if attendanceJSON.Valid && attendanceJSON.String != "" {
				json.Unmarshal([]byte(attendanceJSON.String), &log.Attendance)
			}
			logs = append(logs, log)
		}
	}
	
	return logs
}

func getRouteStudents(routeID string) []Student {
	students := []Student{}
	rows, err := db.Query(`
		SELECT student_id, name, phone_number, alt_phone_number, guardian,
			   pickup_time, dropoff_time, position_number, route_id, driver
		FROM students
		WHERE route_id = $1 AND active = true
		ORDER BY position_number
	`, routeID)
	
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var s Student
			rows.Scan(&s.StudentID, &s.Name, &s.PhoneNumber, &s.AltPhoneNumber,
				&s.Guardian, &s.PickupTime, &s.DropoffTime, &s.PositionNumber,
				&s.RouteID, &s.Driver)
			s.Active = true
			students = append(students, s)
		}
	}
	
	return students
}

// Helper functions for data retrieval

func getBusByID(busID string) *Bus {
	var bus Bus
	err := db.QueryRow(`
		SELECT bus_id, model, capacity, status, oil_status, tire_status, maintenance_notes
		FROM buses WHERE bus_id = $1
	`, busID).Scan(&bus.BusID, &bus.Model, &bus.Capacity, &bus.Status, 
		&bus.OilStatus, &bus.TireStatus, &bus.MaintenanceNotes)
	
	if err != nil {
		return nil
	}
	return &bus
}

func getRouteByID(routeID string) *Route {
	var route Route
	err := db.QueryRow(`
		SELECT route_id, route_name, description
		FROM routes WHERE route_id = $1
	`, routeID).Scan(&route.RouteID, &route.RouteName, &route.Description)
	
	if err != nil {
		return nil
	}
	return &route
}

func getRouteStudentsOrderedByPickup(routeID string) []Student {
	students := []Student{}
	rows, err := db.Query(`
		SELECT student_id, name, phone_number, alt_phone_number, guardian,
			   pickup_time, dropoff_time, position_number, route_id, driver
		FROM students
		WHERE route_id = $1 AND active = true
		ORDER BY pickup_time, position_number
	`, routeID)
	
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var s Student
			rows.Scan(&s.StudentID, &s.Name, &s.PhoneNumber, &s.AltPhoneNumber,
				&s.Guardian, &s.PickupTime, &s.DropoffTime, &s.PositionNumber,
				&s.RouteID, &s.Driver)
			s.Active = true
			students = append(students, s)
		}
	}
	
	return students
}

func getRouteStudentsOrderedByDropoff(routeID string) []Student {
	students := []Student{}
	rows, err := db.Query(`
		SELECT student_id, name, phone_number, alt_phone_number, guardian,
			   pickup_time, dropoff_time, position_number, route_id, driver
		FROM students
		WHERE route_id = $1 AND active = true
		ORDER BY dropoff_time, position_number
	`, routeID)
	
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var s Student
			rows.Scan(&s.StudentID, &s.Name, &s.PhoneNumber, &s.AltPhoneNumber,
				&s.Guardian, &s.PickupTime, &s.DropoffTime, &s.PositionNumber,
				&s.RouteID, &s.Driver)
			s.Active = true
			students = append(students, s)
		}
	}
	
	return students
}

// getUser gets the current user from session
func getUser(r *http.Request) *User {
	return getUserFromSession(r)
}

// isLoggedIn checks if user has valid session
func isLoggedIn(r *http.Request) bool {
	cookie, err := r.Cookie(SessionCookieName)
	if err != nil {
		return false
	}

	session, err := GetSecureSession(cookie.Value)
	if err != nil {
		return false
	}

	return session != nil
}

// getECSEStatisticsWithPercentages returns ECSE statistics including calculated percentages
func getECSEStatisticsWithPercentages() map[string]interface{} {
	stats := make(map[string]interface{})
	
	// Total students
	var totalStudents int
	db.QueryRow("SELECT COUNT(*) FROM ecse_students").Scan(&totalStudents)
	stats["TotalStudents"] = totalStudents
	
	// Active students
	var activeStudents int
	db.QueryRow("SELECT COUNT(*) FROM ecse_students WHERE enrollment_status = 'Active'").Scan(&activeStudents)
	stats["ActiveStudents"] = activeStudents
	
	// Students requiring transportation
	var transportationStudents int
	db.QueryRow("SELECT COUNT(*) FROM ecse_students WHERE transportation_required = true").Scan(&transportationStudents)
	stats["TransportationStudents"] = transportationStudents
	
	// Students with IEP
	var iepStudents int
	db.QueryRow("SELECT COUNT(*) FROM ecse_students WHERE iep_status IS NOT NULL AND iep_status != ''").Scan(&iepStudents)
	stats["IEPStudents"] = iepStudents
	
	// Total services
	var totalServices int
	db.QueryRow("SELECT COUNT(*) FROM ecse_services").Scan(&totalServices)
	stats["TotalServices"] = totalServices
	
	// Service types breakdown
	serviceTypes := make(map[string]int)
	rows, err := db.Query("SELECT service_type, COUNT(*) FROM ecse_services GROUP BY service_type")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var serviceType string
			var count int
			rows.Scan(&serviceType, &count)
			serviceTypes[serviceType] = count
		}
	}
	stats["ServiceTypes"] = serviceTypes
	
	// Calculate percentages
	if totalStudents > 0 {
		stats["ActivePercent"] = int((float64(activeStudents) / float64(totalStudents)) * 100)
		stats["IEPPercent"] = int((float64(iepStudents) / float64(totalStudents)) * 100)
		stats["TransportationPercent"] = int((float64(transportationStudents) / float64(totalStudents)) * 100)
	} else {
		stats["ActivePercent"] = 0
		stats["IEPPercent"] = 0
		stats["TransportationPercent"] = 0
	}
	
	return stats
}

// getCSPNonce gets the CSP nonce from the request context
func getCSPNonce(r *http.Request) string {
	if nonce, ok := r.Context().Value("csp-nonce").(string); ok {
		return nonce
	}
	return ""
}

// Add these missing helper functions if they don't exist elsewhere
func getDriverLogsByDateRange(driver, startDate, endDate string) ([]DriverLog, error) {
	logs := []DriverLog{}
	rows, err := db.Query(`
		SELECT id, driver, bus_id, route_id, date, period, departure_time, arrival_time, mileage, attendance
		FROM driver_logs
		WHERE driver = $1 AND date BETWEEN $2 AND $3
		ORDER BY date DESC, period DESC
	`, driver, startDate, endDate)
	
	if err != nil {
		return logs, err
	}
	defer rows.Close()
	
	for rows.Next() {
		var log DriverLog
		var attendanceJSON sql.NullString
		err := rows.Scan(&log.ID, &log.Driver, &log.BusID, &log.RouteID, &log.Date, 
			&log.Period, &log.Departure, &log.Arrival, &log.Mileage, &attendanceJSON)
		
		if err == nil {
			// Parse attendance JSON
			if attendanceJSON.Valid && attendanceJSON.String != "" {
				json.Unmarshal([]byte(attendanceJSON.String), &log.Attendance)
			}
			logs = append(logs, log)
		}
	}
	
	return logs, nil
}
