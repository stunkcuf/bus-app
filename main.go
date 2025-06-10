package main

import (
	"encoding/json"
	"fmt"
	git "github.com/go-git/go-git/v5"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"os"
	"os/exec" //run start.sh
	"strconv"  // add to importblock
	"strings"
	"time"
)

type User struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

type Attendance struct {
	Date    string `json:"date"`
	Driver  string `json:"driver"`
	Route   string `json:"route"`
	Present int    `json:"present"`
}

type Mileage struct {
	Date   string  `json:"date"`
	Driver string  `json:"driver"`
	Route  string  `json:"route"`
	Miles  float64 `json:"miles"`
}

type Activity struct {
	Date       string  `json:"date"`
	Driver     string  `json:"driver"`
	TripName   string  `json:"trip_name"`
	Attendance int     `json:"attendance"`
	Miles      float64 `json:"miles"`
	Notes      string  `json:"notes"`
}

type DriverSummary struct {
	Name              string
	TotalMorning      int
	TotalEvening      int
	TotalMiles        float64
	MonthlyAvgMiles   float64
	MonthlyAttendance int
}

type RouteStats struct {
	RouteName       string
	TotalMiles      float64
	AvgMiles        float64
	AttendanceDay   int
	AttendanceWeek  int
	AttendanceMonth int
}

type Route struct {
	RouteID   string `json:"route_id"`
	RouteName string `json:"route_name"`
	Positions []struct {
		Position int    `json:"position"`
		Student  string `json:"student"`
	} `json:"positions"`
}

type Bus struct {
	BusNumber        string `json:"bus_number"`
	Status           string `json:"status"` // active, maintenance, out_of_service
	Model            string `json:"model"`
	Capacity         int    `json:"capacity"`
	OilStatus        string `json:"oil_status"`        // good, due, overdue
	TireStatus       string `json:"tire_status"`       // good, worn, replace
	MaintenanceNotes string `json:"maintenance_notes"`
}

type Student struct {
	StudentID       string   `json:"student_id"`
	Name            string   `json:"name"`
	Locations       []Location `json:"locations"`
	PhoneNumber     string   `json:"phone_number"`
	AltPhoneNumber  string   `json:"alt_phone_number"`
	Guardian        string   `json:"guardian"`
	PickupTime      string   `json:"pickup_time"`
	DropoffTime     string   `json:"dropoff_time"`
	PositionNumber  int      `json:"position_number"`
	RouteID         string   `json:"route_id"`
	Driver          string   `json:"driver"`
	Active          bool     `json:"active"`
}

type Location struct {
	Type        string `json:"type"` // "pickup" or "dropoff"
	Address     string `json:"address"`
	Description string `json:"description"`
}

type RouteAssignment struct {
	Driver       string `json:"driver"`
	BusNumber    string `json:"bus_number"`
	RouteID      string `json:"route_id"`
	RouteName    string `json:"route_name"`
	AssignedDate string `json:"assigned_date"`
}

type DriverLog struct {
	Driver     string `json:"driver"`
	BusNumber  string `json:"bus_number"`
	RouteID    string `json:"route_id"`
	Date       string `json:"date"`
	Period     string `json:"period"`
	Departure  string `json:"departure_time"`
	Arrival    string `json:"arrival_time"`
	Mileage    float64 `json:"mileage"`
	Attendance []struct {
		Position   int    `json:"position"`
		Present    bool   `json:"present"`
		PickupTime string `json:"pickup_time,omitempty"`
	} `json:"attendance"`
}

type DashboardData struct {
	User            *User
	Role            string
	DriverSummaries []*DriverSummary
	RouteStats      []*RouteStats
	Activities      []Activity
	Routes          []Route
	Users           []User
	Buses           []*Bus
}

type AssignRouteData struct {
	User            *User
	Assignments     []RouteAssignment
	Drivers         []User
	AvailableRoutes []Route
	AvailableBuses  []*Bus
}

type FleetData struct {
	User  *User
	Buses []*Bus
}

type MaintenanceLog struct {
		BusNumber string `json:"bus_number"`
		Date      string `json:"date"`      // YYYY‑MM‑DD
		Category  string `json:"category"`  // oil, tires, brakes, etc.
		Notes     string `json:"notes"`
		Mileage   int    `json:"mileage"`   // optional
}

type StudentData struct {
	User     *User
	Students []Student
	Routes   []Route
}

var templates *template.Template

func init() {
	var err error
	templates, err = template.New("").Funcs(template.FuncMap{
		"json": func(v interface{}) template.JS {
			b, err := json.Marshal(v)
			if err != nil {
				log.Printf("JSON marshal error: %v", err)
				return template.JS("{}")
			}
			return template.JS(b)
		},
	}).ParseGlob("templates/*.html")

	if err != nil {
		log.Printf("Template parsing failed: %v", err)
		// Create a fallback template to prevent crashes
		templates = template.New("fallback")
	}
}

func ensureDataFiles() {
	os.MkdirAll("data", os.ModePerm)
	if _, err := os.Stat("data/users.json"); os.IsNotExist(err) {
		defaultUsers := []User{{"admin", "adminpass", "manager"}}
		f, _ := os.Create("data/users.json")
		json.NewEncoder(f).Encode(defaultUsers)
		f.Close()
	}
	if _, err := os.Stat("data/route_assignments.json"); os.IsNotExist(err) {
		f, _ := os.Create("data/route_assignments.json")
		json.NewEncoder(f).Encode([]RouteAssignment{})
		f.Close()
	}
}

func loadUsers() []User {
	f, err := os.Open("data/users.json")
	if err != nil {
		return nil
	}
	defer f.Close()
	var users []User
	json.NewDecoder(f).Decode(&users)
	return users
}

func loadRoutes() ([]Route, error) {
	return loadJSON[Route]("data/routes.json")
}

func loadDriverLogs() ([]DriverLog, error) {
	return loadJSON[DriverLog]("data/driver_logs.json")
}

func loadJSON[T any](filename string) ([]T, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var data []T
	err = json.NewDecoder(f).Decode(&data)
	return data, err
}

func seedJSON[T any](path string, defaultData T) error {
		if _, err := os.Stat(path); err == nil {
				return nil // already present
		} else if !os.IsNotExist(err) {
				return fmt.Errorf("stat %s: %w", path, err)
		}

		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				return fmt.Errorf("mkdir: %w", err)
		}
		f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if err != nil {
				return fmt.Errorf("create: %w", err)
		}
		defer f.Close()

		enc := json.NewEncoder(f)
		enc.SetIndent("", "  ")
		if err := enc.Encode(defaultData); err != nil {
				return fmt.Errorf("encode: %w", err)
		}
		log.Printf("Seeded %s", path)
		return nil
}

func loadRouteAssignments() ([]RouteAssignment, error) {
	return loadJSON[RouteAssignment]("data/route_assignments.json")
}

func saveRouteAssignments(assignments []RouteAssignment) error {
	f, err := os.Create("data/route_assignments.json")
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(assignments)
}

func loadMaintenanceLogs() []MaintenanceLog {
	logs, _ := loadJSON[MaintenanceLog]("data/maintenance.json")
	return logs
}

func saveMaintenanceLogs(logs []MaintenanceLog) error {
	f, err := os.Create("data/maintenance.json")
	if err != nil { return err }
	defer f.Close()
	return json.NewEncoder(f).Encode(logs)
}

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

		users := loadUsers()
		users = append(users, User{Username: username, Password: password, Role: role})

		f, _ := os.Create("data/users.json")
		defer f.Close()
		json.NewEncoder(f).Encode(users)

		http.Redirect(w, r, "/manager-dashboard", http.StatusFound)
		return
	}

	templates.ExecuteTemplate(w, "new_user.html", nil)
}

func getUserFromSession(r *http.Request) *User {
	cookie, err := r.Cookie("session_user")
	if err != nil {
		return nil
	}
	uname := cookie.Value
	for _, u := range loadUsers() {
		if u.Username == uname {
			return &u
		}
	}
	return nil
}

func managerDashboard(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	attendance, _ := loadJSON[Attendance]("data/attendance.json")
	mileage, _ := loadJSON[Mileage]("data/mileage.json")
	activities, _ := loadJSON[Activity]("data/activities.json")
	users := loadUsers()
	routes, _ := loadRoutes()
	buses := loadBuses()

	// Filtered active routes
	activeRouteNames := map[string]bool{
		"Victory Square": true,
		"Airportway":     true,
		"NELC":           true,
		"Irrigon":        true,
		"PELC":           true,
		"Umatilla":       true,
	}
	filteredRoutes := []Route{}
	for _, r := range routes {
		if activeRouteNames[r.RouteName] {
			filteredRoutes = append(filteredRoutes, r)
		}
	}

	// Prepare name map
	nameMap := make(map[string]string)
	for _, u := range users {
		if u.Role == "driver" {
			nameMap[strings.ToLower(u.Username)] = u.Username
		}
	}

	driverData := make(map[string]*DriverSummary)
	// Pre-populate all known drivers
	for _, u := range users {
		if u.Role == "driver" {
			driverData[u.Username] = &DriverSummary{Name: u.Username}
		}
	}

	routeData := make(map[string]*RouteStats)
	now := time.Now()

	for _, att := range attendance {
		displayName := nameMap[strings.ToLower(att.Driver)]
		if displayName == "" {
			displayName = att.Driver
		}

		s := driverData[displayName]
		if s == nil {
			s = &DriverSummary{Name: displayName}
			driverData[displayName] = s
		}

		if strings.Contains(strings.ToLower(att.Route), "morning") {
			s.TotalMorning += att.Present
		} else {
			s.TotalEvening += att.Present
		}

		route := routeData[att.Route]
		if route == nil {
			route = &RouteStats{RouteName: att.Route}
			routeData[att.Route] = route
		}
		route.AttendanceMonth += att.Present

		parsed, err := time.Parse("2006-01-02", att.Date)
		if err != nil {
			log.Println("Failed to parse date:", att.Date, err)
			continue
		}
		if now.Sub(parsed).Hours() < 24 {
			route.AttendanceDay += att.Present
		}
		if now.Sub(parsed).Hours() < 168 {
			route.AttendanceWeek += att.Present
		}
	}

	for _, m := range mileage {
		displayName := nameMap[strings.ToLower(m.Driver)]
		if displayName == "" {
			displayName = m.Driver
		}

		s := driverData[displayName]
		if s == nil {
			s = &DriverSummary{Name: displayName}
			driverData[displayName] = s
		}
		s.TotalMiles += m.Miles

		parsed, _ := time.Parse("2006-01-02", m.Date)
		if parsed.Month() == now.Month() && parsed.Year() == now.Year() {
			s.MonthlyAvgMiles += m.Miles
		}

		route := routeData[m.Route]
		if route == nil {
			route = &RouteStats{RouteName: m.Route}
			routeData[m.Route] = route
		}
		route.TotalMiles += m.Miles
	}

	for _, s := range driverData {
		s.MonthlyAvgMiles = s.MonthlyAvgMiles / float64(30)
	}
	for _, r := range routeData {
		r.AvgMiles = r.TotalMiles / float64(30)
	}

	// Build slices for template
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
		Routes:          filteredRoutes, // ✅ Only active routes passed to template
		Users:           users,
		Buses:           buses,
	}

	templates.ExecuteTemplate(w, "dashboard.html", data)
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
		User   *User
		Name   string
		Logs   []DriverLog
	}{
		User: user,
		Name: name,
		Logs: driverLogs,
	}

	templates.ExecuteTemplate(w, "driver_profile.html", data)
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
	assignments, _ := loadRouteAssignments()

	var driverLog *DriverLog
	for _, log := range logs {
		if log.Driver == user.Username && log.Date == date && log.Period == period {
			driverLog = &log
			break
		}
	}

	type PageData struct {
		User      *User
		Date      string
		Period    string
		Route     *Route
		DriverLog *DriverLog
		Bus       *Bus // Include bus information
	}

	var driverRoute *Route
	var assignedBus *Bus

	// First check if driver has an assigned route
	var assignedBusNumber string
	var assignedRouteID string
	for _, a := range assignments {
		if a.Driver == user.Username {
			assignedBusNumber = a.BusNumber
			assignedRouteID = a.RouteID
			break
		}
	}

	// Load all buses
	buses := loadBuses()

	// Find the route for the assigned bus or from existing log
	for _, r := range routes {
		if assignedRouteID != "" && r.RouteID == assignedRouteID {
			driverRoute = &r
			break
		} else if driverLog != nil && r.RouteID == driverLog.RouteID {
			driverRoute = &r
			break
		}
	}

	//Find the assigned bus
	for _, b := range buses {
		if assignedBusNumber != "" && b.BusNumber == assignedBusNumber {
			assignedBus = b
			break
		} else if driverLog != nil && b.BusNumber == driverLog.BusNumber {
			assignedBus = b
			break
		}
	}

	data := PageData{
		User:      user,
		Date:      date,
		Period:    period,
		Route:     driverRoute,
		DriverLog: driverLog,
		Bus:       assignedBus, // Pass the bus data
	}

	if driverRoute == nil && driverLog != nil {
		log.Printf("Warning: No route found for route ID %s", driverLog.RouteID)
	}

	templates.ExecuteTemplate(w, "driver_dashboard.html", data)
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
	templates.ExecuteTemplate(w, "login.html", nil)
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
		Auth:       nil, // Add credentials if needed
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
	// Optional: Validate GitHub signature
	cmd := exec.Command("git", "pull", "origin", "main")
	output, err := cmd.CombinedOutput()
	if err != nil {
		http.Error(w, "Git pull failed: "+err.Error(), 500)
		return
	}
	exec.Command("kill", "1").Run() // triggers a Replit restart
	fmt.Fprintf(w, "Updated:\n%s", string(output))
}

func saveDriverLog(w http.ResponseWriter, r *http.Request) {
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
	busNumber := r.FormValue("bus_number")
	departure := r.FormValue("departure")
	arrival := r.FormValue("arrival")
	mileage, _ := strconv.ParseFloat(r.FormValue("mileage"), 64)

	routes, _ := loadRoutes()
	var positions []struct {
		Position int    `json:"position"`
		Student  string `json:"student"`
	}
	var routeID string
	for _, rt := range routes {
		if rt.RouteID == routeID {
			positions = rt.Positions
			break
		}
	}

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

	logs, _ := loadDriverLogs()

	updated := false
	for i := range logs {
		if logs[i].Driver == user.Username && logs[i].Date == date && logs[i].Period == period {
			logs[i].BusNumber = busNumber
			logs[i].Departure = departure
			logs[i].Arrival = arrival
			logs[i].Mileage = mileage
			logs[i].Attendance = attendance
			updated = true
			break
		}
	}
	if !updated {
		//need to find route id based on bus number from assignment
		assignments, _ := loadRouteAssignments()
		var routeID string
		for _, assignment := range assignments {
			if assignment.BusNumber == busNumber && assignment.Driver == user.Username {
				routeID = assignment.RouteID
				break
			}
		}

		logs = append(logs, DriverLog{
			Driver:     user.Username,
			BusNumber:  busNumber,
			RouteID:    routeID,
			Date:       date,
			Period:     period,
			Departure:  departure,
			Arrival:    arrival,
			Mileage:    mileage,
			Attendance: attendance,
		})
	}

	f, _ := os.Create("data/driver_logs.json")
	defer f.Close()
	json.NewEncoder(f).Encode(logs)

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
	assignedBusNumbers := make(map[string]bool)
	assignedDrivers := make(map[string]bool)
	for _, a := range assignments {
		assignedRouteIDs[a.RouteID] = true
		assignedBusNumbers[a.BusNumber] = true
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
		if bus.Status == "active" && !assignedBusNumbers[bus.BusNumber] {
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

	templates.ExecuteTemplate(w, "assign_routes.html", data)
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
	busNumber := r.FormValue("bus_number")
	routeID := r.FormValue("route_id")

	if driver == "" || busNumber == "" || routeID == "" {
		http.Redirect(w, r, "/assign-routes", http.StatusFound)
		return
	}

	// Find route name
	routes, _ := loadRoutes()
	var routeName string
	for _, rt := range routes {
		if rt.RouteID == routeID {
			routeName = rt.RouteName
			break
		}
	}

	assignments, _ := loadRouteAssignments()

	// Check if driver already has an assignment
	for i, a := range assignments {
		if a.Driver == driver {
			// Update existing assignment
			assignments[i].BusNumber = busNumber
			assignments[i].RouteID = routeID
			assignments[i].RouteName = routeName
			assignments[i].AssignedDate = time.Now().Format("2006-01-02")
			saveRouteAssignments(assignments)
			http.Redirect(w, r, "/assign-routes", http.StatusFound)
			return
		}
	}

	// Check if route is already assigned
	for _, a := range assignments {
		if a.RouteID == routeID {
			http.Redirect(w, r, "/assign-routes", http.StatusFound)
			return
		}
	}

	// Add new assignment
	newAssignment := RouteAssignment{
		Driver:       driver,
		BusNumber:    busNumber,
		RouteID:      routeID,
		RouteName:    routeName,
		AssignedDate: time.Now().Format("2006-01-02"),
	}

	assignments = append(assignments, newAssignment)
	saveRouteAssignments(assignments)
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
	busNumber := r.FormValue("bus_number")

	assignments, _ := loadRouteAssignments()

	// Remove assignment
	var newAssignments []RouteAssignment
	for _, a := range assignments {
		if !(a.Driver == driver && a.BusNumber == busNumber) {
			newAssignments = append(newAssignments, a)
		}
	}

	saveRouteAssignments(newAssignments)
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
	}

	templates.ExecuteTemplate(w, "fleet.html", data)
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
	busNumber := r.FormValue("bus_number")
	status := r.FormValue("status")
	model := r.FormValue("model")
	capacity, _ := strconv.Atoi(r.FormValue("capacity"))
	oilStatus := r.FormValue("oil_status")
	tireStatus := r.FormValue("tire_status")
	maintenanceNotes := r.FormValue("maintenance_notes")

	buses := loadBuses()

	// Check if bus number already exists
	for _, b := range buses {
		if b.BusNumber == busNumber {
			http.Redirect(w, r, "/fleet", http.StatusFound)
			return
		}
	}

	newBus := &Bus{
		BusNumber:        busNumber,
		Status:           status,
		Model:            model,
		Capacity:         capacity,
		OilStatus:        oilStatus,
		TireStatus:       tireStatus,
		MaintenanceNotes: maintenanceNotes,
	}

	buses = append(buses, newBus)
	saveBuses(buses)
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
	originalBusNumber := r.FormValue("original_bus_number")
	busNumber := r.FormValue("bus_number")
	status := r.FormValue("status")
	model := r.FormValue("model")
	capacity, _ := strconv.Atoi(r.FormValue("capacity"))
	oilStatus := r.FormValue("oil_status")
	tireStatus := r.FormValue("tire_status")
	maintenanceNotes := r.FormValue("maintenance_notes")

	buses := loadBuses()

	for i, b := range buses {
		if b.BusNumber == originalBusNumber {
			buses[i].BusNumber = busNumber
			buses[i].Status = status
			buses[i].Model = model
			buses[i].Capacity = capacity
			buses[i].OilStatus = oilStatus
			buses[i].TireStatus = tireStatus
			buses[i].MaintenanceNotes = maintenanceNotes
			break
		}
	}

	saveBuses(buses)
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
	busNumber := r.FormValue("bus_number")

	buses := loadBuses()
	var newBuses []*Bus
	for _, b := range buses {
		if b.BusNumber != busNumber {
			newBuses = append(newBuses, b)
		}
	}

	saveBuses(newBuses)
	http.Redirect(w, r, "/fleet", http.StatusFound)
}

func saveBuses(buses []*Bus) error {
	f, err := os.Create("data/buses.json")
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(buses)
}

func loadStudents() []Student {
	f, err := os.Open("data/students.json")
	if err != nil {
		return []Student{}
	}
	defer f.Close()
	var students []Student
	json.NewDecoder(f).Decode(&students)
	return students
}

func saveStudents(students []Student) error {
	f, err := os.Create("data/students.json")
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(students)
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

	templates.ExecuteTemplate(w, "students.html", data)
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

	students := loadStudents()

	// Generate student ID
	studentID := fmt.Sprintf("STU_%d", len(students)+1)

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

	students = append(students, newStudent)
	saveStudents(students)
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
			break
		}
	}

	saveStudents(students)
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

	students := loadStudents()
	var newStudents []Student
	for _, s := range students {
		if !(s.StudentID == studentID && s.Driver == user.Username) {
			newStudents = append(newStudents, s)
		}
	}

	saveStudents(newStudents)
	http.Redirect(w, r, "/students", http.StatusFound)
}

func logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{Name: "session_user", Value: "", MaxAge: -1, Path: "/"})
	http.Redirect(w, r, "/", http.StatusFound)
}

func withRecovery(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("Recovered from panic in handler: %v", err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
			}
		}()
		h(w, r)
	}
}

// Load buses from json file
func loadBuses() []*Bus {
	f, err := os.Open("data/buses.json")
	if err != nil {
		// Handle error appropriately, e.g., log it and return an empty slice
		log.Printf("Error opening buses.json: %v", err)
		return []*Bus{} // Return an empty slice to avoid nil pointer dereference
	}
	defer f.Close()

	var buses []*Bus
	decoder := json.NewDecoder(f)
	if err := decoder.Decode(&buses); err != nil {
		// Handle decode error
		log.Printf("Error decoding buses.json: %v", err)
		return []*Bus{} // Return an empty slice
	}

	return buses
}

// Initialize data files
func initDataFiles() {
	// Create buses.json if it doesn't exist, and seed with some default data.
	if _, err := os.Stat("data/buses.json"); os.IsNotExist(err) {
		defaultBuses := []*Bus{
			{BusNumber: "1", Status: "active", Model: "Ford", Capacity: 20, OilStatus: "good", TireStatus: "good", MaintenanceNotes: ""},
			{BusNumber: "2", Status: "active", Model: "Chevy", Capacity: 25, OilStatus: "due", TireStatus: "good", MaintenanceNotes: "Oil change scheduled"},
			{BusNumber: "3", Status: "maintenance", Model: "Toyota", Capacity: 15, OilStatus: "good", TireStatus: "worn", MaintenanceNotes: "Brake inspection in progress"},
		}
		f, err := os.Create("data/buses.json")
		if err != nil {
			log.Fatalf("failed to create buses.json: %v", err)
		}
		defer f.Close()
		enc := json.NewEncoder(f)
		enc.SetIndent("", "  ") // Pretty print the JSON
		if err := enc.Encode(defaultBuses); err != nil {
			log.Fatalf("failed to encode buses to json: %v", err)
		}
		log.Println("Created and seeded data/buses.json")
	}

	// Create students.json if it doesn't exist
	if _, err := os.Stat("data/students.json"); os.IsNotExist(err) {
		defaultStudents := []Student{}
		f, err := os.Create("data/students.json")
		if err != nil {
			log.Fatalf("failed to create students.json: %v", err)
		}
		defer f.Close()
		enc := json.NewEncoder(f)
		enc.SetIndent("", "  ")
		if err := enc.Encode(defaultStudents); err != nil {
			log.Fatalf("failed to encode students to json: %v", err)
		}
		log.Println("Created data/students.json")
	}

	// Create routes.json if it doesn't exist, and seed with some default data.
	if _, err := os.Stat("data/routes.json"); os.IsNotExist(err) {
		routes := []Route{
			{
				RouteID: "1", 
				RouteName: "Victory Square", 
				Positions: []struct {
					Position int    `json:"position"`
					Student  string `json:"student"`
				}{{Position: 1, Student: "Alice"}, {Position: 2, Student: "Bob"}},
			},
			{
				RouteID: "2", 
				RouteName: "Airportway", 
				Positions: []struct {
					Position int    `json:"position"`
					Student  string `json:"student"`
				}{{Position: 1, Student: "Charlie"}, {Position: 2, Student: "David"}},
			},
		}
		f, err := os.Create("data/routes.json")
		if err != nil {
			log.Fatalf("failed to create routes.json: %v", err)
		}
		defer f.Close()
		enc := json.NewEncoder(f)
		enc.SetIndent("", "  ") // Pretty print the JSON
		if err := enc.Encode(routes); err != nil {
			log.Fatalf("failed to encode routes to json: %v", err)
		}
		log.Println("Created and seeded data/routes.json")
	}
}

// maint handles maintenance log operations
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
		logEntry := MaintenanceLog{
				BusNumber: r.FormValue("bus_number"),
				Date:      r.FormValue("date"),
				Category:  r.FormValue("category"),
				Notes:     r.FormValue("notes"),
				Mileage:   mileage,
		}

		logs := loadMaintenanceLogs()
		logs = append(logs, logEntry)
		if err := saveMaintenanceLogs(logs); err != nil {
				http.Error(w, "Unable to save", http.StatusInternalServerError)
				return
		}
		http.Redirect(w, r, "/fleet", http.StatusFound)
}

func main() {
	ensureDataFiles()
	initDataFiles()

	http.HandleFunc("/", withRecovery(loginPage))
	http.HandleFunc("/new-user", withRecovery(newUserPage))
	http.HandleFunc("/dashboard", withRecovery(dashboardRouter))
	http.HandleFunc("/manager-dashboard", withRecovery(managerDashboard))
	http.HandleFunc("/driver-dashboard", withRecovery(driverDashboard))
	http.HandleFunc("/driver/", withRecovery(driverProfileHandler))
	http.HandleFunc("/assign-routes", withRecovery(assignRoutesPage))
	http.HandleFunc("/assign-route", withRecovery(assignRoute))
	http.HandleFunc("/unassign-route", withRecovery(unassignRoute))
	http.HandleFunc("/fleet", withRecovery(fleetPage))
	http.HandleFunc("/add-bus", withRecovery(addBus))
	http.HandleFunc("/edit-bus", withRecovery(editBus))
	http.HandleFunc("/remove-bus", withRecovery(removeBus))
	http.HandleFunc("/webhook", withRecovery(handleWebhook))
	http.HandleFunc("/pull", withRecovery(runPullHandler))
	http.HandleFunc("/save-log", withRecovery(saveDriverLog))
	http.HandleFunc("/students", withRecovery(studentsPage))
	http.HandleFunc("/add-student", withRecovery(addStudent))
	http.HandleFunc("/edit-student", withRecovery(editStudent))
	http.HandleFunc("/remove-student", withRecovery(removeStudent))
	http.HandleFunc("/add-maint", withRecovery(addMaintenanceLog))
	http.HandleFunc("/logout", withRecovery(logout))
	

	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}

	server := &http.Server{
		Addr:         "0.0.0.0:" + port,
		Handler:      nil,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("Server starting on port %s", port)
	log.Fatal(server.ListenAndServe())
}