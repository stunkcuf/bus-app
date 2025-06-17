package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"

	git "github.com/go-git/go-git/v5"
)

//go:embed templates/*.html
var tmplFS embed.FS

var templates *template.Template

func init() {
	var err error

	// Create function map for templates
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

	// Parse templates from embedded filesystem
	templates, err = template.New("").Funcs(funcMap).ParseFS(tmplFS, "templates/*.html")
	if err != nil {
		log.Fatalf("Template parsing failed: %v", err)
	}

	log.Println("Templates loaded successfully")
}

// =============================================================================
// HTTP HANDLERS
// =============================================================================

func newUserPage(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	if r.Method == http.MethodPost {
		r.ParseForm()
		username := r.FormValue("username")
		password := r.FormValue("password")
		role := r.FormValue("role")

		newUser := User{Username: username, Password: password, Role: role}
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
	user := getUserFromSession(r)
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
		r.ParseForm()
		newPassword := r.FormValue("password")
		newRole := r.FormValue("role")

		users := loadUsers()
		for i, u := range users {
			if u.Username == username {
				users[i].Password = newPassword
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
	user := getUserFromSession(r)
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
		} else if driverLog.Period == "evening" {
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

	data := DashboardData{
		User:            user,
		Role:            user.Role,
		DriverSummaries: driverSummaries,
		RouteStats:      routeStats,
		Activities:      activities,
		Routes:          routes,
		Users:           users,
		Buses:           buses,
	}

	executeTemplate(w, "dashboard.html", data)
}

func driverProfileHandler(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/driver/")
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	// Lookup logs or summaries for the driver
	logs, _ := loadDriverLogs()
	var driverLogs []DriverLog
	for _, l := range logs {
		if l.Driver == name {
			driverLogs = append(driverLogs, l)
		}
	}

	data := struct {
		User *User
		Name string
		Logs []DriverLog
	}{
		User: user,
		Name: name,
		Logs: driverLogs,
	}

	executeTemplate(w, "driver_profile.html", data)
}

func driverDashboard(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
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

	data := DriverDashboardData{
		User:       user,
		Date:       date,
		Period:     period,
		Route:      driverRoute,
		DriverLog:  driverLog,
		Bus:        assignedBus,
		RecentLogs: recentLogs,
	}

	executeTemplate(w, "driver_dashboard.html", data)
}

func vehicleMaintenancePage(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	// Get vehicle ID from query parameter or URL path
	vehicleID := r.URL.Query().Get("vehicle_id")
	if vehicleID == "" {
		pathParts := strings.Split(r.URL.Path, "/")
		if len(pathParts) >= 3 {
			vehicleID = pathParts[2]
		}
	}

	if vehicleID == "" {
		http.Error(w, "Vehicle ID required", http.StatusBadRequest)
		return
	}

	log.Printf("Fetching maintenance records for vehicle ID: %s", vehicleID)

	// Prepare template data structure
	data := struct {
		Vehicle         interface{}
		MaintenanceLogs []MaintenanceLog
		TotalRecords    int
		TotalCost       float64
		AverageCost     float64
	}{}

	// Get maintenance logs from database
	busLogs, err := getBusMaintenanceRecords(vehicleID)
	if err != nil {
		log.Printf("Error loading maintenance records: %v", err)
	}

	// Convert BusMaintenanceLog to MaintenanceLog format
	var maintenanceLogs []MaintenanceLog
	var totalCost float64
	
	for _, busLog := range busLogs {
		var mileagePtr *int
		if busLog.Mileage > 0 {
			mileagePtr = &busLog.Mileage
		}
		
		maintenanceLogs = append(maintenanceLogs, MaintenanceLog{
			ServiceDate: busLog.Date,
			Mileage:     mileagePtr,
			WorkDone:    fmt.Sprintf("%s: %s", busLog.Category, busLog.Notes),
		})
	}

	// Sort by date (newest first)
	sort.Slice(maintenanceLogs, func(i, j int) bool {
		return maintenanceLogs[i].ServiceDate > maintenanceLogs[j].ServiceDate
	})

	// Try to find the vehicle/bus details
	buses := loadBuses()
	var foundBus *Bus
	for _, bus := range buses {
		if bus.BusID == vehicleID {
			foundBus = bus
			break
		}
	}

	if foundBus != nil {
		// Create a vehicle struct for template compatibility
		vehicleForTemplate := struct {
			VehicleNumber int
			Make          string
			Model         string
			Year          string
			VIN           string
			Description   string
		}{
			VehicleNumber: func() int {
				if num, err := strconv.Atoi(strings.TrimPrefix(vehicleID, "BUS")); err == nil {
					return num
				}
				return 0
			}(),
			Make:        "Bus Fleet",
			Model:       foundBus.Model,
			Year:        "",
			VIN:         vehicleID,
			Description: fmt.Sprintf("Capacity: %d passengers", foundBus.Capacity),
		}
		data.Vehicle = vehicleForTemplate
	} else {
		// Try company vehicles
		vehicles := loadVehicles()
		var foundVehicle *Vehicle
		for _, vehicle := range vehicles {
			if vehicle.VehicleID == vehicleID {
				foundVehicle = &vehicle
				break
			}
		}

		if foundVehicle == nil {
			log.Printf("Vehicle/Bus not found with ID: %s", vehicleID)
			http.Error(w, "Vehicle not found", http.StatusNotFound)
			return
		}

		vehicleForTemplate := struct {
			VehicleNumber int
			Make          string
			Model         string
			Year          string
			VIN           string
			Description   string
		}{
			VehicleNumber: func() int {
				if num, err := strconv.Atoi(strings.TrimPrefix(vehicleID, "VEH")); err == nil {
					return num
				}
				return 0
			}(),
			Make:        foundVehicle.Model,
			Model:       foundVehicle.Model,
			Year:        foundVehicle.Year,
			VIN:         vehicleID,
			Description: foundVehicle.Description,
		}
		data.Vehicle = vehicleForTemplate
	}

	data.MaintenanceLogs = maintenanceLogs
	data.TotalRecords = len(maintenanceLogs)
	data.TotalCost = totalCost
	if data.TotalRecords > 0 && data.TotalCost > 0 {
		data.AverageCost = data.TotalCost / float64(data.TotalRecords)
	}

	executeTemplate(w, "vehicle_maintenance.html", data)
}

func loginPage(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		r.ParseForm()
		username := r.FormValue("username")
		password := r.FormValue("password")
		for _, u := range loadUsers() {
			if u.Username == username && u.Password == password {
				http.SetCookie(w, &http.Cookie{
					Name:  "session_user",
					Value: username,
					Path:  "/",
				})

				if u.Role == "manager" {
					http.Redirect(w, r, "/manager-dashboard", http.StatusFound)
				} else if u.Role == "driver" {
					http.Redirect(w, r, "/driver-dashboard", http.StatusFound)
				} else {
					http.Redirect(w, r, "/", http.StatusFound)
				}
				return
			}
		}
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}
	executeTemplate(w, "login.html", nil)
}

func pullLatest() string {
	repo, err := git.PlainOpen(".")
	if err != nil {
		return "❌ Failed to open repo: " + err.Error()
	}

	w, err := repo.Worktree()
	if err != nil {
		return "❌ Failed to get worktree: " + err.Error()
	}

	err = w.Pull(&git.PullOptions{
		RemoteName: "origin",
		Auth:       nil,
		Force:      true,
	})
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return "❌ Git pull failed: " + err.Error()
	}
	return "✅ Git pull complete"
}

func runPullHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST allowed", http.StatusMethodNotAllowed)
		return
	}

	if r.Header.Get("x-trigger-source") != "cloudflare" {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	output := pullLatest()

	go func() {
		time.Sleep(1 * time.Second)
		exec.Command("bash", "restart_app.sh").Run()
	}()

	w.Write([]byte("✅ Git pulled and app restarted\n" + output))
}

func handleWebhook(w http.ResponseWriter, r *http.Request) {
	cmd := exec.Command("git", "pull", "origin", "main")
	output, err := cmd.CombinedOutput()
	if err != nil {
		http.Error(w, "Git pull failed: "+err.Error(), 500)
		return
	}
	exec.Command("kill", "1").Run()
	fmt.Fprintf(w, "Updated:\n%s", string(output))
}

func saveDriverLogHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	user := getUserFromSession(r)
	if user == nil || user.Role != "driver" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	r.ParseForm()
	date := r.FormValue("date")
	period := r.FormValue("period")
	busID := r.FormValue("bus_id")
	departure := r.FormValue("departure")
	arrival := r.FormValue("arrival")
	mileage, _ := strconv.ParseFloat(r.FormValue("mileage"), 64)

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
		pickup := r.FormValue("pickup_" + strconv.Itoa(p.Position))
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

func dashboardRouter(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
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

func assignRoutesPage(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	assignments, _ := loadRouteAssignments()
	routes, _ := loadRoutes()
	users := loadUsers()
	buses := loadBuses()

	// Filter drivers only
	var drivers []User
	for _, u := range users {
		if u.Role == "driver" {
			drivers = append(drivers, u)
		}
	}

	// Find assigned items
	assignedRouteIDs := make(map[string]bool)
	assignedBusIDs := make(map[string]bool)
	assignedDrivers := make(map[string]bool)
	for _, a := range assignments {
		assignedRouteIDs[a.RouteID] = true
		assignedBusIDs[a.BusID] = true
		assignedDrivers[a.Driver] = true
	}

	// Filter available routes (not assigned)
	var availableRoutes []Route
	for _, route := range routes {
		if !assignedRouteIDs[route.RouteID] {
			availableRoutes = append(availableRoutes, route)
		}
	}

	// Filter available buses (active and not assigned)
	var availableBuses []*Bus
	for _, bus := range buses {
		if bus.Status == "active" && !assignedBusIDs[bus.BusID] {
			availableBuses = append(availableBuses, bus)
		}
	}

	// Filter available drivers (not assigned)
	var availableDrivers []User
	for _, driver := range drivers {
		if !assignedDrivers[driver.Username] {
			availableDrivers = append(availableDrivers, driver)
		}
	}

	data := AssignRouteData{
		User:            user,
		Assignments:     assignments,
		Drivers:         availableDrivers,
		AvailableRoutes: availableRoutes,
		AvailableBuses:  availableBuses,
	}

	executeTemplate(w, "assign_routes.html", data)
}

func addRoute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		log.Printf("addRoute: Unauthorized access attempt")
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	r.ParseForm()
	routeName := r.FormValue("route_name")
	description := r.FormValue("description")

	log.Printf("addRoute: Attempting to add route '%s' with description '%s'", routeName, description)

	if routeName == "" {
		log.Printf("addRoute: Empty route name provided")
		http.Error(w, "Route name is required", http.StatusBadRequest)
		return
	}

	// Load existing routes
	routes, err := loadRoutes()
	if err != nil {
		log.Printf("addRoute: Error loading routes: %v", err)
		http.Error(w, "Unable to load existing routes", http.StatusInternalServerError)
		return
	}

	// Check for duplicate route names
	for _, existingRoute := range routes {
		if existingRoute.RouteName == routeName {
			log.Printf("addRoute: Duplicate route name '%s'", routeName)
			http.Error(w, "A route with this name already exists", http.StatusBadRequest)
			return
		}
	}

	// Generate unique route ID
	existingIDs := make([]string, len(routes))
	for i, r := range routes {
		existingIDs[i] = r.RouteID
	}
	routeID := ensureUniqueID("RT", existingIDs)

	// Create new route
	newRoute := Route{
		RouteID:     routeID,
		RouteName:   routeName,
		Description: description,
		Positions: []struct {
			Position int    `json:"position"`
			Student  string `json:"student"`
		}{},
	}

	log.Printf("addRoute: Creating new route with ID %s", routeID)

	// Save to database
	if err := saveRoute(newRoute); err != nil {
		log.Printf("addRoute: Error saving route: %v", err)
		http.Error(w, "Unable to save route", http.StatusInternalServerError)
		return
	}

	log.Printf("addRoute: Route %s added successfully", routeID)
	http.Redirect(w, r, "/assign-routes", http.StatusSeeOther)
}

func editRoute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	r.ParseForm()
	routeID := r.FormValue("route_id")
	routeName := r.FormValue("route_name")
	description := r.FormValue("description")

	if routeID == "" || routeName == "" {
		http.Error(w, "Route ID and name are required", http.StatusBadRequest)
		return
	}

	// Load existing routes
	routes, err := loadRoutes()
	if err != nil {
		log.Printf("Error loading routes: %v", err)
		http.Error(w, "Unable to load routes", http.StatusInternalServerError)
		return
	}

	// Find and update the route
	var routeToUpdate *Route
	for i, route := range routes {
		if route.RouteID == routeID {
			routes[i].RouteName = routeName
			routes[i].Description = description
			routeToUpdate = &routes[i]
			break
		}
	}

	if routeToUpdate == nil {
		http.Error(w, "Route not found", http.StatusNotFound)
		return
	}

	// Save to database
	if err := saveRoute(*routeToUpdate); err != nil {
		log.Printf("Error updating route: %v", err)
		http.Error(w, "Unable to update route", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/assign-routes", http.StatusSeeOther)
}

func deleteRouteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	r.ParseForm()
	routeID := r.FormValue("route_id")

	if routeID == "" {
		http.Error(w, "Route ID is required", http.StatusBadRequest)
		return
	}

	// Check if route is currently assigned
	assignments, err := loadRouteAssignments()
	if err == nil {
		for _, assignment := range assignments {
			if assignment.RouteID == routeID {
				http.Error(w, "Cannot delete route that is currently assigned to a driver", http.StatusBadRequest)
				return
			}
		}
	}

	// Delete from database
	if err := deleteRoute(routeID); err != nil {
		log.Printf("Error deleting route: %v", err)
		http.Error(w, "Unable to delete route", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/assign-routes", http.StatusSeeOther)
}

func assignRoute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	r.ParseForm()
	driver := r.FormValue("driver")
	busID := r.FormValue("bus_id")
	routeID := r.FormValue("route_id")

	if driver == "" || busID == "" || routeID == "" {
		http.Error(w, "All fields are required", http.StatusBadRequest)
		return
	}

	// Find route name
	routes, err := loadRoutes()
	if err != nil {
		log.Printf("Error loading routes: %v", err)
		http.Error(w, "Unable to load routes", http.StatusInternalServerError)
		return
	}

	var routeName string
	routeFound := false
	for _, rt := range routes {
		if rt.RouteID == routeID {
			routeName = rt.RouteName
			routeFound = true
			break
		}
	}

	if !routeFound {
		http.Error(w, "Route not found", http.StatusBadRequest)
		return
	}

	// Verify bus exists and is active
	buses := loadBuses()
	busFound := false
	for _, bus := range buses {
		if bus.BusID == busID {
			if bus.Status != "active" {
				http.Error(w, "Bus is not active", http.StatusBadRequest)
				return
			}
			busFound = true
			break
		}
	}

	if !busFound {
		http.Error(w, "Bus not found", http.StatusBadRequest)
		return
	}

	// Create or update assignment
	newAssignment := RouteAssignment{
		Driver:       driver,
		BusID:        busID,
		RouteID:      routeID,
		RouteName:    routeName,
		AssignedDate: time.Now().Format("2006-01-02"),
	}

	if err := saveRouteAssignment(newAssignment); err != nil {
		log.Printf("Error saving assignment: %v", err)
		http.Error(w, "Unable to save assignment", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/assign-routes", http.StatusFound)
}

func unassignRoute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	r.ParseForm()
	driver := r.FormValue("driver")

	if err := deleteRouteAssignment(driver); err != nil {
		log.Printf("Error deleting assignment: %v", err)
		http.Error(w, "Unable to remove assignment", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/assign-routes", http.StatusFound)
}

func fleetPage(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	buses := loadBuses()
	data := FleetData{
		User:  user,
		Buses: buses,
		Today: time.Now().Format("2006-01-02"),
	}

	executeTemplate(w, "fleet.html", data)
}

func addBus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	r.ParseForm()
	busID := r.FormValue("bus_id")
	status := r.FormValue("status")
	model := r.FormValue("model")
	capacity, _ := strconv.Atoi(r.FormValue("capacity"))
	oilStatus := r.FormValue("oil_status")
	tireStatus := r.FormValue("tire_status")
	maintenanceNotes := r.FormValue("maintenance_notes")

	// Check if bus ID already exists
	buses := loadBuses()
	for _, b := range buses {
		if b.BusID == busID {
			http.Error(w, "Bus ID already exists", http.StatusBadRequest)
			return
		}
	}

	newBus := &Bus{
		BusID:            busID,
		Status:           status,
		Model:            model,
		Capacity:         capacity,
		OilStatus:        oilStatus,
		TireStatus:       tireStatus,
		MaintenanceNotes: maintenanceNotes,
	}

	if err := saveBus(newBus); err != nil {
		log.Printf("Error saving bus: %v", err)
		http.Error(w, "Unable to save bus", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/fleet", http.StatusFound)
}

func editBus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	r.ParseForm()
	originalBusID := r.FormValue("original_bus_id")
	busID := r.FormValue("bus_id")
	status := r.FormValue("status")
	model := r.FormValue("model")
	capacity, _ := strconv.Atoi(r.FormValue("capacity"))
	oilStatus := r.FormValue("oil_status")
	tireStatus := r.FormValue("tire_status")
	maintenanceNotes := r.FormValue("maintenance_notes")

	buses := loadBuses()

	// Check if new bus ID conflicts with existing (unless it's the same bus)
	if busID != originalBusID {
		for _, b := range buses {
			if b.BusID == busID {
				http.Error(w, "Bus ID already exists", http.StatusBadRequest)
				return
			}
		}
	}

	// Find the original bus to check status change
	var originalBus *Bus
	for _, b := range buses {
		if b.BusID == originalBusID {
			originalBus = b
			break
		}
	}

	if originalBus == nil {
		log.Printf("EditBus: Bus not found with ID '%s'", originalBusID)
		http.Error(w, fmt.Sprintf("Bus not found with ID '%s'", originalBusID), http.StatusNotFound)
		return
	}

	// Check if status is changing from active to inactive
	statusChangingToInactive := originalBus.Status == "active" && (status == "maintenance" || status == "out_of_service")

	// If status is changing to inactive, check if bus is currently assigned
	if statusChangingToInactive {
		assignments, err := loadRouteAssignments()
		if err == nil {
			for _, assignment := range assignments {
				if assignment.BusID == originalBusID {
					// Bus is assigned to a driver/route, prompt for replacement bus selection
					http.Error(w, "REQUIRES_REPLACEMENT_BUS:"+assignment.Driver+":"+assignment.RouteName, http.StatusConflict)
					return
				}
			}
		}
	}

	// Update the bus
	updatedBus := &Bus{
		BusID:            busID,
		Status:           status,
		Model:            model,
		Capacity:         capacity,
		OilStatus:        oilStatus,
		TireStatus:       tireStatus,
		MaintenanceNotes: maintenanceNotes,
	}

	if err := saveBus(updatedBus); err != nil {
		log.Printf("Error saving bus: %v", err)
		http.Error(w, "Unable to save bus", http.StatusInternalServerError)
		return
	}

	// Auto-create maintenance log if status changed to maintenance or out_of_service
	if statusChangingToInactive || (status == "maintenance" && originalBus.Status != "maintenance") {
		logEntry := BusMaintenanceLog{
			BusID:    busID,
			Date:     time.Now().Format("2006-01-02"),
			Category: "status_change",
			Notes:    fmt.Sprintf("Bus status changed from '%s' to '%s'. %s", originalBus.Status, status, maintenanceNotes),
			Mileage:  0,
		}

		if err := saveMaintenanceLog(logEntry); err != nil {
			log.Printf("Warning: Failed to save maintenance log: %v", err)
		} else {
			log.Printf("Maintenance log created for bus %s status change", busID)
		}
	}

	http.Redirect(w, r, "/fleet", http.StatusFound)
}

func removeBus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	r.ParseForm()
	busID := r.FormValue("bus_id")

	// Check if bus is currently assigned
	assignments, err := loadRouteAssignments()
	if err == nil {
		for _, a := range assignments {
			if a.BusID == busID {
				http.Error(w, "Cannot remove bus that is currently assigned to a route", http.StatusBadRequest)
				return
			}
		}
	}

	if err := deleteBus(busID); err != nil {
		log.Printf("Error deleting bus: %v", err)
		http.Error(w, "Unable to remove bus", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/fleet", http.StatusFound)
}

func companyFleetPage(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	vehicles := loadVehicles()
	data := CompanyFleetData{
		User:     user,
		Vehicles: vehicles,
	}

	executeTemplate(w, "company_fleet.html", data)
}

func companyFleetDataHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	vehicles := loadVehicles()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(vehicles); err != nil {
		log.Printf("Error encoding vehicles: %v", err)
		http.Error(w, "Error encoding data", http.StatusInternalServerError)
	}
}

func importVehicleAsBus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	r.ParseForm()
	vehicleID := r.FormValue("vehicle_id")

	if vehicleID == "" {
		http.Error(w, "Vehicle ID is required", http.StatusBadRequest)
		return
	}

	// Load current vehicles to get full details
	vehicles := loadVehicles()
	var sourceVehicle *Vehicle
	for _, v := range vehicles {
		if v.VehicleID == vehicleID {
			sourceVehicle = &v
			break
		}
	}

	if sourceVehicle == nil {
		http.Error(w, "Vehicle not found in company fleet", http.StatusNotFound)
		return
	}

	// Check if already imported
	buses := loadBuses()
	for _, bus := range buses {
		if bus.BusID == vehicleID {
			http.Error(w, "Vehicle already imported as bus", http.StatusBadRequest)
			return
		}
	}

	// Determine capacity based on vehicle type
	capacity := 30 // Default
	description := strings.ToUpper(sourceVehicle.Description)

	// Try to determine capacity from description
	if strings.Contains(description, "EXPRESS") || strings.Contains(description, "STARCRAFT") {
		capacity = 25
	} else if strings.Contains(description, "MIDCO") {
		capacity = 20
	} else if strings.Contains(description, "CUTAWAY") {
		capacity = 15
	}

	// Use description as the model name since it's more descriptive
	modelName := sourceVehicle.Description
	if modelName == "" {
		modelName = sourceVehicle.Model
	}

	// Create new bus from vehicle data
	newBus := &Bus{
		BusID:            vehicleID,
		Status:           "active",
		Model:            modelName,
		Capacity:         capacity,
		OilStatus:        sourceVehicle.OilStatus,
		TireStatus:       sourceVehicle.TireStatus,
		MaintenanceNotes: fmt.Sprintf("Imported from company fleet. License: %s, Year: %s, Original Model: %s",
			sourceVehicle.License, sourceVehicle.Year, sourceVehicle.Model),
	}

	if err := saveBus(newBus); err != nil {
		log.Printf("Error saving bus: %v", err)
		http.Error(w, "Unable to save bus", http.StatusInternalServerError)
		return
	}

	log.Printf("Successfully imported vehicle %s (%s) as bus", vehicleID, modelName)
	http.Redirect(w, r, "/fleet", http.StatusFound)
}

func studentsPage(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "driver" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	students := loadStudents()
	routes, _ := loadRoutes()

	// Filter students for this driver
	var driverStudents []Student
	for _, s := range students {
		if s.Driver == user.Username {
			driverStudents = append(driverStudents, s)
		}
	}

	data := StudentData{
		User:     user,
		Students: driverStudents,
		Routes:   routes,
	}

	executeTemplate(w, "students.html", data)
}

func addStudent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	user := getUserFromSession(r)
	if user == nil || user.Role != "driver" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	r.ParseForm()
	name := r.FormValue("name")
	phoneNumber := r.FormValue("phone_number")
	altPhoneNumber := r.FormValue("alt_phone_number")
	guardian := r.FormValue("guardian")
	pickupTime := r.FormValue("pickup_time")
	dropoffTime := r.FormValue("dropoff_time")
	positionNumber, _ := strconv.Atoi(r.FormValue("position_number"))
	routeID := r.FormValue("route_id")

	// Parse locations
	var locations []Location
	pickupAddresses := r.Form["pickup_address"]
	pickupDescriptions := r.Form["pickup_description"]
	dropoffAddresses := r.Form["dropoff_address"]
	dropoffDescriptions := r.Form["dropoff_description"]

	for i, addr := range pickupAddresses {
		if addr != "" {
			desc := ""
			if i < len(pickupDescriptions) {
				desc = pickupDescriptions[i]
			}
			locations = append(locations, Location{
				Type:        "pickup",
				Address:     addr,
				Description: desc,
			})
		}
	}

	for i, addr := range dropoffAddresses {
		if addr != "" {
			desc := ""
			if i < len(dropoffDescriptions) {
				desc = dropoffDescriptions[i]
			}
			locations = append(locations, Location{
				Type:        "dropoff",
				Address:     addr,
				Description: desc,
			})
		}
	}

	// Generate student ID
	students := loadStudents()
	existingIDs := make([]string, len(students))
	for i, s := range students {
		existingIDs[i] = s.StudentID
	}
	studentID := ensureUniqueID("STU", existingIDs)

	newStudent := Student{
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

	if err := saveStudent(newStudent); err != nil {
		log.Printf("Error saving student: %v", err)
		http.Error(w, "Unable to save student", http.StatusInternalServerError)
		return
	}
	
	http.Redirect(w, r, "/students", http.StatusFound)
}

func editStudent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	user := getUserFromSession(r)
	if user == nil || user.Role != "driver" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	r.ParseForm()
	studentID := r.FormValue("student_id")
	name := r.FormValue("name")
	phoneNumber := r.FormValue("phone_number")
	altPhoneNumber := r.FormValue("alt_phone_number")
	guardian := r.FormValue("guardian")
	pickupTime := r.FormValue("pickup_time")
	dropoffTime := r.FormValue("dropoff_time")
	positionNumber, _ := strconv.Atoi(r.FormValue("position_number"))
	routeID := r.FormValue("route_id")
	active := r.FormValue("active") == "on"

	// Parse locations
	var locations []Location
	pickupAddresses := r.Form["pickup_address"]
	pickupDescriptions := r.Form["pickup_description"]
	dropoffAddresses := r.Form["dropoff_address"]
	dropoffDescriptions := r.Form["dropoff_description"]

	for i, addr := range pickupAddresses {
		if addr != "" {
			desc := ""
			if i < len(pickupDescriptions) {
				desc = pickupDescriptions[i]
			}
			locations = append(locations, Location{
				Type:        "pickup",
				Address:     addr,
				Description: desc,
			})
		}
	}

	for i, addr := range dropoffAddresses {
		if addr != "" {
			desc := ""
			if i < len(dropoffDescriptions) {
				desc = dropoffDescriptions[i]
			}
			locations = append(locations, Location{
				Type:        "dropoff",
				Address:     addr,
				Description: desc,
			})
		}
	}

	// Find and update student
	students := loadStudents()
	for i, s := range students {
		if s.StudentID == studentID && s.Driver == user.Username {
			students[i].Name = name
			students[i].Locations = locations
			students[i].PhoneNumber = phoneNumber
			students[i].AltPhoneNumber = altPhoneNumber
			students[i].Guardian = guardian
			students[i].PickupTime = pickupTime
			students[i].DropoffTime = dropoffTime
			students[i].PositionNumber = positionNumber
			students[i].RouteID = routeID
			students[i].Active = active
			
			if err := saveStudent(students[i]); err != nil {
				log.Printf("Error saving student: %v", err)
				http.Error(w, "Unable to save student", http.StatusInternalServerError)
				return
			}
			break
		}
	}

	http.Redirect(w, r, "/students", http.StatusFound)
}

func removeStudent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	user := getUserFromSession(r)
	if user == nil || user.Role != "driver" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	r.ParseForm()
	studentID := r.FormValue("student_id")

	// Verify the student belongs to this driver before deleting
	students := loadStudents()
	studentBelongsToDriver := false
	for _, s := range students {
		if s.StudentID == studentID && s.Driver == user.Username {
			studentBelongsToDriver = true
			break
		}
	}

	if !studentBelongsToDriver {
		http.Error(w, "Unauthorized", http.StatusForbidden)
		return
	}

	if err := deleteStudent(studentID); err != nil {
		log.Printf("Error deleting student: %v", err)
		http.Error(w, "Unable to remove student", http.StatusInternalServerError)
		return
	}
	
	http.Redirect(w, r, "/students", http.StatusFound)
}

func reassignDriverBus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	r.ParseForm()
	driverName := r.FormValue("driver")
	newBusID := r.FormValue("new_bus_id")

	if driverName == "" || newBusID == "" {
		http.Error(w, "Driver and new bus ID are required", http.StatusBadRequest)
		return
	}

	// Load assignments
	assignments, err := loadRouteAssignments()
	if err != nil {
		log.Printf("Error loading assignments: %v", err)
		http.Error(w, "Unable to load assignments", http.StatusInternalServerError)
		return
	}

	// Find and update the driver's assignment
	updated := false
	for _, assignment := range assignments {
		if assignment.Driver == driverName {
			assignment.BusID = newBusID
			if err := saveRouteAssignment(assignment); err != nil {
				log.Printf("Error saving assignment: %v", err)
				http.Error(w, "Unable to save assignment", http.StatusInternalServerError)
				return
			}
			updated = true
			break
		}
	}

	if !updated {
		http.Error(w, "Driver assignment not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Driver reassigned successfully"))
}

func addMaintenanceLog(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	r.ParseForm()
	mileage, _ := strconv.Atoi(r.FormValue("mileage"))
	logEntry := BusMaintenanceLog{
		BusID:    r.FormValue("bus_id"),
		Date:     r.FormValue("date"),
		Category: r.FormValue("category"),
		Notes:    r.FormValue("notes"),
		Mileage:  mileage,
	}

	// Validate bus exists
	buses := loadBuses()
	busExists := false
	for _, bus := range buses {
		if bus.BusID == logEntry.BusID {
			busExists = true
			break
		}
	}

	if !busExists {
		http.Error(w, "Bus not found", http.StatusBadRequest)
		return
	}

	if err := saveMaintenanceLog(logEntry); err != nil {
		log.Printf("Error saving maintenance log: %v", err)
		http.Error(w, "Unable to save maintenance log", http.StatusInternalServerError)
		return
	}
	
	http.Redirect(w, r, "/fleet", http.StatusFound)
}

func removeUser(w http.ResponseWriter, r *http.Request) {
	// Accept both GET and POST for debugging
	var usernameToRemove string

	if r.Method == http.MethodGet {
		// Parse from URL query for GET requests
		usernameToRemove = r.URL.Query().Get("username")
		log.Printf("DEBUG: Received GET request for removing user: %s", usernameToRemove)
	} else if r.Method == http.MethodPost {
		// Parse form data for POST requests
		r.ParseForm()
		usernameToRemove = r.FormValue("username")
		log.Printf("DEBUG: Received POST request for removing user: %s", usernameToRemove)
	} else {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check if user is logged in and is a manager
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	// Check if username was provided
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

func updateVehicleStatus(w http.ResponseWriter, r *http.Request) {
	log.Printf("🔍 updateVehicleStatus: Request received - Method: %s, URL: %s", r.Method, r.URL.String())

	if r.Method != http.MethodPost {
		log.Printf("❌ updateVehicleStatus: Invalid method %s", r.Method)
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check user session
	user := getUserFromSession(r)
	log.Printf("🔍 updateVehicleStatus: User from session: %+v", user)
	if user == nil {
		log.Printf("❌ updateVehicleStatus: No user in session")
		http.Error(w, "Unauthorized - No session", http.StatusUnauthorized)
		return
	}
	if user.Role != "manager" {
		log.Printf("❌ updateVehicleStatus: User %s has role %s, expected manager", user.Username, user.Role)
		http.Error(w, "Unauthorized - Not a manager", http.StatusUnauthorized)
		return
	}
	log.Printf("✅ updateVehicleStatus: User %s is authorized as manager", user.Username)

	// Parse form data
	if err := r.ParseForm(); err != nil {
		log.Printf("❌ updateVehicleStatus: Error parsing form: %v", err)
		http.Error(w, "Error parsing form data", http.StatusBadRequest)
		return
	}

	log.Printf("🔍 updateVehicleStatus: Raw form data: %+v", r.Form)

	// Extract form values
	vehicleID := r.FormValue("vehicle_id")
	statusType := r.FormValue("status_type")
	newStatus := r.FormValue("new_status")

	log.Printf("🔍 updateVehicleStatus: Extracted values:")
	log.Printf("  - vehicle_id: '%s' (len=%d)", vehicleID, len(vehicleID))
	log.Printf("  - status_type: '%s' (len=%d)", statusType, len(statusType))
	log.Printf("  - new_status: '%s' (len=%d)", newStatus, len(newStatus))

	// Check for missing parameters
	if vehicleID == "" || statusType == "" || newStatus == "" {
		log.Printf("❌ updateVehicleStatus: Missing required parameters")
		http.Error(w, "Missing required parameters", http.StatusBadRequest)
		return
	}

	log.Printf("✅ updateVehicleStatus: All parameters present, proceeding with update")

	// Load current vehicles
	vehicles := loadVehicles()
	log.Printf("🔍 updateVehicleStatus: Loaded %d vehicles from database", len(vehicles))

	// Find and update the vehicle
	updated := false
	for i, vehicle := range vehicles {
		if vehicle.VehicleID == vehicleID {
			log.Printf("✅ updateVehicleStatus: Found matching vehicle at index %d", i)

			switch statusType {
			case "oil":
				log.Printf("🔍 updateVehicleStatus: Updating oil status from '%s' to '%s'", vehicles[i].OilStatus, newStatus)
				vehicles[i].OilStatus = newStatus
			case "tire":
				log.Printf("🔍 updateVehicleStatus: Updating tire status from '%s' to '%s'", vehicles[i].TireStatus, newStatus)
				vehicles[i].TireStatus = newStatus
			case "status":
				log.Printf("🔍 updateVehicleStatus: Updating vehicle status from '%s' to '%s'", vehicles[i].Status, newStatus)
				vehicles[i].Status = newStatus
			default:
				log.Printf("❌ updateVehicleStatus: Invalid status type: '%s'", statusType)
				http.Error(w, "Invalid status type", http.StatusBadRequest)
				return
			}
			
			// Save individual vehicle
			if err := saveVehicle(vehicles[i]); err != nil {
				log.Printf("❌ updateVehicleStatus: Error saving vehicle: %v", err)
				http.Error(w, "Failed to save changes", http.StatusInternalServerError)
				return
			}
			
			updated = true
			log.Printf("✅ updateVehicleStatus: Vehicle updated successfully")
			break
		}
	}

	if !updated {
		log.Printf("❌ updateVehicleStatus: Vehicle not found with ID: '%s'", vehicleID)
		http.Error(w, "Vehicle not found", http.StatusNotFound)
		return
	}

	log.Printf("✅ updateVehicleStatus: Successfully updated vehicle %s", vehicleID)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Status updated successfully"))
}

func updateBusStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	r.ParseForm()
	busID := r.FormValue("bus_id")
	statusType := r.FormValue("status_type")
	newStatus := r.FormValue("new_status")

	log.Printf("Updating bus %s: %s status to %s", busID, statusType, newStatus)

	if busID == "" || statusType == "" || newStatus == "" {
		http.Error(w, "Missing required parameters", http.StatusBadRequest)
		return
	}

	// Load current buses
	buses := loadBuses()
	updated := false

	// Find and update the bus
	for i, bus := range buses {
		if bus.BusID == busID {
			switch statusType {
			case "oil":
				buses[i].OilStatus = newStatus
			case "tire":
				buses[i].TireStatus = newStatus
			case "status":
				buses[i].Status = newStatus
			default:
				http.Error(w, "Invalid status type", http.StatusBadRequest)
				return
			}
			
			// Save individual bus
			if err := saveBus(buses[i]); err != nil {
				log.Printf("Error saving bus: %v", err)
				http.Error(w, "Failed to save changes", http.StatusInternalServerError)
				return
			}
			
			updated = true
			log.Printf("Updated bus %s: %s status to %s", busID, statusType, newStatus)
			break
		}
	}

	if !updated {
		log.Printf("Bus not found: %s", busID)
		http.Error(w, "Bus not found", http.StatusNotFound)
		return
	}

	log.Printf("Successfully updated and saved bus %s", busID)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Status updated successfully"))
}

func logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{Name: "session_user", Value: "", MaxAge: -1, Path: "/"})
	http.Redirect(w, r, "/", http.StatusFound)
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}

func rootHealthCheck(w http.ResponseWriter, r *http.Request) {
	// Always show login page for root path
	loginPage(w, r)
}

// =============================================================================
// MAIN FUNCTION
// =============================================================================

func main() {
	// Add defer to catch any panics
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Server crashed with panic: %v", r)
			os.Exit(1)
		}
	}()

	log.Println("Starting bus transportation app...")

	// Setup database - REQUIRED
	log.Println("🗄️  Setting up PostgreSQL database...")
	setupDatabase()
	defer closeDatabase()
	
	log.Println("✅ Database setup complete")

	// Setup HTTP routes with recovery middleware
	log.Println("Setting up HTTP routes...")
	http.HandleFunc("/", withRecovery(rootHealthCheck))
	http.HandleFunc("/new-user", withRecovery(newUserPage))
	http.HandleFunc("/edit-user", withRecovery(editUserPage))
	http.HandleFunc("/dashboard", withRecovery(dashboardRouter))
	http.HandleFunc("/manager-dashboard", withRecovery(managerDashboard))
	http.HandleFunc("/driver-dashboard", withRecovery(driverDashboard))
	http.HandleFunc("/driver/", withRecovery(driverProfileHandler))
	http.HandleFunc("/assign-routes", withRecovery(assignRoutesPage))
	http.HandleFunc("/assign-route", withRecovery(assignRoute))
	http.HandleFunc("/assign-routes/add", withRecovery(addRoute))
	http.HandleFunc("/assign-routes/edit", withRecovery(editRoute))
	http.HandleFunc("/assign-routes/deleteRouteHandler", withRecovery(deleteRouteHandler))
	http.HandleFunc("/unassign-route", withRecovery(unassignRoute))
	http.HandleFunc("/fleet", withRecovery(fleetPage))
	http.HandleFunc("/company-fleet", withRecovery(companyFleetPage))
	http.HandleFunc("/company-fleet-data", withRecovery(companyFleetDataHandler))
	http.HandleFunc("/vehicle-maintenance", withRecovery(vehicleMaintenancePage))
	http.HandleFunc("/import-vehicle-as-bus", withRecovery(importVehicleAsBus))
	http.HandleFunc("/update-vehicle-status", withRecovery(updateVehicleStatus))
	http.HandleFunc("/update-bus-status", withRecovery(updateBusStatus))
	http.HandleFunc("/add-bus", withRecovery(addBus))
	http.HandleFunc("/edit-bus", withRecovery(editBus))
	http.HandleFunc("/remove-bus", withRecovery(removeBus))
	http.HandleFunc("/webhook", withRecovery(handleWebhook))
	http.HandleFunc("/pull", withRecovery(runPullHandler))
	http.HandleFunc("/saveDriverLogHandler", withRecovery(saveDriverLogHandler))
	http.HandleFunc("/students", withRecovery(studentsPage))
	http.HandleFunc("/add-student", withRecovery(addStudent))
	http.HandleFunc("/edit-student", withRecovery(editStudent))
	http.HandleFunc("/remove-student", withRecovery(removeStudent))
	http.HandleFunc("/add-maint", withRecovery(addMaintenanceLog))
	http.HandleFunc("/reassign-driver-bus", withRecovery(reassignDriverBus))
	http.HandleFunc("/remove-user", withRecovery(removeUser))
	http.HandleFunc("/logout", withRecovery(logout))
	http.HandleFunc("/health", withRecovery(healthCheck))

	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}

	server := &http.Server{
		Addr:           "0.0.0.0:" + port,
		Handler:        http.DefaultServeMux,
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
