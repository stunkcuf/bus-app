package main

import (
	"embed"
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
	BusID            string `json:"bus_id"`
	Status           string `json:"status"` // active, maintenance, out_of_service
	Model            string `json:"model"`
	Capacity         int    `json:"capacity"`
	OilStatus        string `json:"oil_status"`        // good, due, overdue
	TireStatus       string `json:"tire_status"`       // good, worn, replace
	MaintenanceNotes string `json:"maintenance_notes"`
}

type Student struct {
	StudentID       string     `json:"student_id"`
	Name            string     `json:"name"`
	Locations       []Location `json:"locations"`
	PhoneNumber     string     `json:"phone_number"`
	AltPhoneNumber  string     `json:"alt_phone_number"`
	Guardian        string     `json:"guardian"`
	PickupTime      string     `json:"pickup_time"`
	DropoffTime     string     `json:"dropoff_time"`
	PositionNumber  int        `json:"position_number"`
	RouteID         string     `json:"route_id"`
	Driver          string     `json:"driver"`
	Active          bool       `json:"active"`
}

type Location struct {
	Type        string `json:"type"` // "pickup" or "dropoff"
	Address     string `json:"address"`
	Description string `json:"description"`
}

type RouteAssignment struct {
	Driver       string `json:"driver"`
	BusID        string `json:"bus_id"`
	RouteID      string `json:"route_id"`
	RouteName    string `json:"route_name"`
	AssignedDate string `json:"assigned_date"`
}

type DriverLog struct {
	Driver     string `json:"driver"`
	BusID      string `json:"bus_id"`
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
	BusID    string `json:"bus_id"`
	Date     string `json:"date"`      // YYYY‑MM‑DD
	Category string `json:"category"`  // oil, tires, brakes, etc.
	Notes    string `json:"notes"`
	Mileage  int    `json:"mileage"`   // optional
}

type StudentData struct {
	User     *User
	Students []Student
	Routes   []Route
}

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
	}

	// Parse templates from embedded filesystem
	templates, err = template.New("").Funcs(funcMap).ParseFS(tmplFS, "templates/*.html")
	if err != nil {
		log.Fatalf("Template parsing failed: %v", err)
	}
	
	log.Println("Templates loaded successfully")
}

// Helper function to safely execute templates
func executeTemplate(w http.ResponseWriter, name string, data interface{}) {
	if err := templates.ExecuteTemplate(w, name, data); err != nil {
		log.Printf("Error executing template %s: %v", name, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
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
	assignments, err := loadJSON[RouteAssignment]("data/route_assignments.json")
	if err != nil {
		if os.IsNotExist(err) {
			// Return empty slice if file doesn't exist yet
			return []RouteAssignment{}, nil
		}
		return nil, fmt.Errorf("failed to load route assignments: %w", err)
	}
	return assignments, nil
}

func saveRouteAssignments(assignments []RouteAssignment) error {
	f, err := os.Create("data/route_assignments.json")
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(assignments)
}

func loadMaintenanceLogs() []MaintenanceLog {
	logs, _ := loadJSON[MaintenanceLog]("data/maintenance.json")
	return logs
}

func saveMaintenanceLogs(logs []MaintenanceLog) error {
	f, err := os.Create("data/maintenance.json")
	if err != nil { return err }
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(logs)
}

// Load buses from json file
func loadBuses() []*Bus {
	f, err := os.Open("data/buses.json")
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("buses.json not found, returning empty slice")
			return []*Bus{}
		}
		log.Printf("Error opening buses.json: %v", err)
		return []*Bus{}
	}
	defer f.Close()

	var buses []*Bus
	decoder := json.NewDecoder(f)
	if err := decoder.Decode(&buses); err != nil {
		log.Printf("Error decoding buses.json: %v", err)
		return []*Bus{}
	}

	return buses
}

func saveBuses(buses []*Bus) error {
	f, err := os.Create("data/buses.json")
	if err != nil {
		return fmt.Errorf("failed to create buses.json: %w", err)
	}
	defer f.Close()
	
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(buses); err != nil {
		return fmt.Errorf("failed to encode buses: %w", err)
	}
	
	return nil
}

// Helper function to save driver logs
func saveDriverLogs(logs []DriverLog) error {
	f, err := os.Create("data/driver_logs.json")
	if err != nil {
		return fmt.Errorf("failed to create driver logs file: %w", err)
	}
	defer f.Close()
	
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(logs); err != nil {
		return fmt.Errorf("failed to encode driver logs: %w", err)
	}
	
	return nil
}

// getDriverRouteAssignment returns the current route assignment for a driver
func getDriverRouteAssignment(driverUsername string) (*RouteAssignment, error) {
	assignments, err := loadRouteAssignments()
	if err != nil {
		return nil, fmt.Errorf("failed to load assignments: %w", err)
	}
	
	for _, assignment := range assignments {
		if assignment.Driver == driverUsername {
			return &assignment, nil
		}
	}
	
	return nil, fmt.Errorf("no assignment found for driver %s", driverUsername)
}

// validateRouteAssignment checks if a route assignment is valid
func validateRouteAssignment(assignment RouteAssignment) error {
	if assignment.Driver == "" {
		return fmt.Errorf("driver cannot be empty")
	}
	if assignment.BusID == "" {
		return fmt.Errorf("bus ID cannot be empty")
	}
	if assignment.RouteID == "" {
		return fmt.Errorf("route ID cannot be empty")
	}
	
	// Check if driver exists
	users := loadUsers()
	driverExists := false
	for _, u := range users {
		if u.Username == assignment.Driver && u.Role == "driver" {
			driverExists = true
			break
		}
	}
	if !driverExists {
		return fmt.Errorf("driver %s does not exist", assignment.Driver)
	}
	
	// Check if bus exists and is active
	buses := loadBuses()
	busExists := false
	for _, b := range buses {
		if b.BusID == assignment.BusID {
			if b.Status != "active" {
				return fmt.Errorf("bus %s is not active", assignment.BusID)
			}
			busExists = true
			break
		}
	}
	if !busExists {
		return fmt.Errorf("bus %s does not exist", assignment.BusID)
	}
	
	// Check if route exists
	routes, err := loadRoutes()
	if err != nil {
		return fmt.Errorf("failed to load routes: %w", err)
	}
	routeExists := false
	for _, r := range routes {
		if r.RouteID == assignment.RouteID {
			routeExists = true
			break
		}
	}
	if !routeExists {
		return fmt.Errorf("route %s does not exist", assignment.RouteID)
	}
	
	return nil
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

	executeTemplate(w, "new_user.html", nil)
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
		User   *User
		Name   string
		Logs   []DriverLog
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
		Bus       *Bus
	}

	var driverRoute *Route
	var assignedBus *Bus

	// Get the driver's current assignment
	assignment, err := getDriverRouteAssignment(user.Username)
	if err != nil {
		log.Printf("Warning: No assignment found for driver %s: %v", user.Username, err)
		// Continue without assignment - driver might not be assigned yet
	}

	// Load all buses
	buses := loadBuses()

	// Find the route and bus based on assignment or existing log
	if assignment != nil {
		// Use assignment data (preferred)
		for _, r := range routes {
			if r.RouteID == assignment.RouteID {
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

	data := PageData{
		User:      user,
		Date:      date,
		Period:    period,
		Route:     driverRoute,
		DriverLog: driverLog,
		Bus:       assignedBus,
	}

	if driverRoute == nil && assignment != nil {
		log.Printf("Warning: No route found for route ID %s", assignment.RouteID)
	}
	if assignedBus == nil && assignment != nil {
		log.Printf("Warning: No bus found for bus ID %s", assignment.BusID)
	}

	executeTemplate(w, "driver_dashboard.html", data)
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
	busID := r.FormValue("bus_id")  // Changed from bus_number to bus_id
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
		if rt.RouteID == assignment.RouteID {
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

	// Load existing logs
	logs, err := loadDriverLogs()
	if err != nil {
		log.Printf("Error loading driver logs: %v", err)
		// Continue with empty slice if file doesn't exist
		logs = []DriverLog{}
	}

	// Check if we're updating an existing log
	updated := false
	for i := range logs {
		if logs[i].Driver == user.Username && logs[i].Date == date && logs[i].Period == period {
			logs[i].BusID = busID
			logs[i].RouteID = assignment.RouteID
			logs[i].Departure = departure
			logs[i].Arrival = arrival
			logs[i].Mileage = mileage
			logs[i].Attendance = attendance
			updated = true
			break
		}
	}

	// If not updating, create new log entry
	if !updated {
		logs = append(logs, DriverLog{
			Driver:     user.Username,
			BusID:      busID,
			RouteID:    assignment.RouteID,
			Date:       date,
			Period:     period,
			Departure:  departure,
			Arrival:    arrival,
			Mileage:    mileage,
			Attendance: attendance,
		})
	}

	// Save the logs
	if err := saveDriverLogs(logs); err != nil {
		log.Printf("Error saving driver logs: %v", err)
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
	busID := r.FormValue("bus_id")  // Changed from bus_number
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

	assignments, err := loadRouteAssignments()
	if err != nil {
		log.Printf("Error loading assignments: %v", err)
		assignments = []RouteAssignment{}
	}

	// Check if driver already has an assignment
	for i, a := range assignments {
		if a.Driver == driver {
			// Update existing assignment
			assignments[i].BusID = busID
			assignments[i].RouteID = routeID
			assignments[i].RouteName = routeName
			assignments[i].AssignedDate = time.Now().Format("2006-01-02")
			
			if err := saveRouteAssignments(assignments); err != nil {
				log.Printf("Error saving assignments: %v", err)
				http.Error(w, "Unable to save assignment", http.StatusInternalServerError)
				return
			}
			
			http.Redirect(w, r, "/assign-routes", http.StatusFound)
			return
		}
	}

	// Check if route or bus is already assigned
	for _, a := range assignments {
		if a.RouteID == routeID {
			http.Error(w, "Route is already assigned", http.StatusBadRequest)
			return
		}
		if a.BusID == busID {
			http.Error(w, "Bus is already assigned", http.StatusBadRequest)
			return
		}
	}

	// Add new assignment
	newAssignment := RouteAssignment{
		Driver:       driver,
		BusID:        busID,
		RouteID:      routeID,
		RouteName:    routeName,
		AssignedDate: time.Now().Format("2006-01-02"),
	}

	assignments = append(assignments, newAssignment)
	if err := saveRouteAssignments(assignments); err != nil {
		log.Printf("Error saving assignments: %v", err)
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
	busID := r.FormValue("bus_id")  // Changed from bus_number

	assignments, err := loadRouteAssignments()
	if err != nil {
		log.Printf("Error loading assignments: %v", err)
		http.Error(w, "Unable to load assignments", http.StatusInternalServerError)
		return
	}

	// Remove assignment
	var newAssignments []RouteAssignment
	found := false
	for _, a := range assignments {
		if !(a.Driver == driver && a.BusID == busID) {
			newAssignments = append(newAssignments, a)
		} else {
			found = true
		}
	}

	if !found {
		http.Error(w, "Assignment not found", http.StatusNotFound)
		return
	}

	if err := saveRouteAssignments(newAssignments); err != nil {
		log.Printf("Error saving assignments: %v", err)
		http.Error(w, "Unable to save assignments", http.StatusInternalServerError)
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
	busID := r.FormValue("bus_id")  // Changed from bus_number
	status := r.FormValue("status")
	model := r.FormValue("model")
	capacity, _ := strconv.Atoi(r.FormValue("capacity"))
	oilStatus := r.FormValue("oil_status")
	tireStatus := r.FormValue("tire_status")
	maintenanceNotes := r.FormValue("maintenance_notes")

	buses := loadBuses()

	// Check if bus ID already exists
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

	buses = append(buses, newBus)
	if err := saveBuses(buses); err != nil {
		log.Printf("Error saving buses: %v", err)
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
	originalBusID := r.FormValue("original_bus_id")  // Changed from original_bus_number
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

	updated := false
	for i, b := range buses {
		if b.BusID == originalBusID {
			buses[i].BusID = busID
			buses[i].Status = status
			buses[i].Model = model
			buses[i].Capacity = capacity
			buses[i].OilStatus = oilStatus
			buses[i].TireStatus = tireStatus
			buses[i].MaintenanceNotes = maintenanceNotes
			updated = true
			break
		}
	}

	if !updated {
		http.Error(w, "Bus not found", http.StatusNotFound)
		return
	}

	if err := saveBuses(buses); err != nil {
		log.Printf("Error saving buses: %v", err)
		http.Error(w, "Unable to save bus", http.StatusInternalServerError)
		return
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
	busID := r.FormValue("bus_id")  // Changed from bus_number

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

	buses := loadBuses()
	var newBuses []*Bus
	found := false
	for _, b := range buses {
		if b.BusID != busID {
			newBuses = append(newBuses, b)
		} else {
			found = true
		}
	}

	if !found {
		http.Error(w, "Bus not found", http.StatusNotFound)
		return
	}

	if err := saveBuses(newBuses); err != nil {
		log.Printf("Error saving buses: %v", err)
		http.Error(w, "Unable to save buses", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/fleet", http.StatusFound)
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

// Updated maintenance log function to use BusID
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
		BusID:    r.FormValue("bus_id"), // Changed from bus_number
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

	logs := loadMaintenanceLogs()
	logs = append(logs, logEntry)
	if err := saveMaintenanceLogs(logs); err != nil {
		log.Printf("Error saving maintenance logs: %v", err)
		http.Error(w, "Unable to save", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/fleet", http.StatusFound)
}

func logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{Name: "session_user", Value: "", MaxAge: -1, Path: "/"})
	http.Redirect(w, r, "/", http.StatusFound)
}

func withRecovery(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("Recovered from panic in handler %s %s: %v", r.Method, r.URL.Path, err)
				
				// Set headers to prevent caching of error responses
				w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
				w.Header().Set("Pragma", "no-cache")
				w.Header().Set("Expires", "0")
				
				if !isResponseWritten(w) {
					http.Error(w, "Internal server error", http.StatusInternalServerError)
				}
			}
		}()
		
		// Log the request for debugging
		log.Printf("Handling request: %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)
		h(w, r)
	}
}

// Helper to check if response has been written
func isResponseWritten(w http.ResponseWriter) bool {
	// This is a simple check - in production you might want a more sophisticated approach
	return false
}

// Migration helper functions to convert from old structure to new ID-based structure

// migrateBusData converts old BusNumber fields to BusID
func migrateBusData() error {
	f, err := os.Open("data/buses.json")
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No file to migrate
		}
		return err
	}
	defer f.Close()

	var rawData []map[string]interface{}
	if err := json.NewDecoder(f).Decode(&rawData); err != nil {
		return fmt.Errorf("failed to decode buses.json: %w", err)
	}

	// Convert BusNumber to BusID if needed
	migrated := false
	for _, bus := range rawData {
		if busNumber, exists := bus["bus_number"]; exists {
			bus["bus_id"] = busNumber
			delete(bus, "bus_number")
			migrated = true
		}
	}

	if migrated {
		f, err := os.Create("data/buses.json")
		if err != nil {
			return fmt.Errorf("failed to create migrated buses.json: %w", err)
		}
		defer f.Close()

		enc := json.NewEncoder(f)
		enc.SetIndent("", "  ")
		if err := enc.Encode(rawData); err != nil {
			return fmt.Errorf("failed to encode migrated buses: %w", err)
		}

		log.Println("Migrated buses.json: BusNumber -> BusID")
	}

	return nil
}

// migrateRouteAssignments converts old BusNumber fields to BusID
func migrateRouteAssignments() error {
	f, err := os.Open("data/route_assignments.json")
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No file to migrate
		}
		return err
	}
	defer f.Close()

	var rawData []map[string]interface{}
	if err := json.NewDecoder(f).Decode(&rawData); err != nil {
		return fmt.Errorf("failed to decode route_assignments.json: %w", err)
	}

	// Convert BusNumber to BusID if needed
	migrated := false
	for _, assignment := range rawData {
		if busNumber, exists := assignment["bus_number"]; exists {
			assignment["bus_id"] = busNumber
			delete(assignment, "bus_number")
			migrated = true
		}
	}

	if migrated {
		f, err := os.Create("data/route_assignments.json")
		if err != nil {
			return fmt.Errorf("failed to create migrated route_assignments.json: %w", err)
		}
		defer f.Close()

		enc := json.NewEncoder(f)
		enc.SetIndent("", "  ")
		if err := enc.Encode(rawData); err != nil {
			return fmt.Errorf("failed to encode migrated assignments: %w", err)
		}

		log.Println("Migrated route_assignments.json: BusNumber -> BusID")
	}

	return nil
}

// migrateDriverLogs converts old BusNumber fields to BusID
func migrateDriverLogs() error {
	f, err := os.Open("data/driver_logs.json")
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No file to migrate
		}
		return err
	}
	defer f.Close()

	var rawData []map[string]interface{}
	if err := json.NewDecoder(f).Decode(&rawData); err != nil {
		return fmt.Errorf("failed to decode driver_logs.json: %w", err)
	}

	// Convert BusNumber to BusID if needed
	migrated := false
	for _, log := range rawData {
		if busNumber, exists := log["bus_number"]; exists {
			log["bus_id"] = busNumber
			delete(log, "bus_number")
			migrated = true
		}
	}

	if migrated {
		f, err := os.Create("data/driver_logs.json")
		if err != nil {
			return fmt.Errorf("failed to create migrated driver_logs.json: %w", err)
		}
		defer f.Close()

		enc := json.NewEncoder(f)
		enc.SetIndent("", "  ")
		if err := enc.Encode(rawData); err != nil {
			return fmt.Errorf("failed to encode migrated logs: %w", err)
		}

		log.Println("Migrated driver_logs.json: BusNumber -> BusID")
	}

	return nil
}

// migrateMaintenanceLogs converts old BusNumber fields to BusID
func migrateMaintenanceLogs() error {
	f, err := os.Open("data/maintenance.json")
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No file to migrate
		}
		return err
	}
	defer f.Close()

	var rawData []map[string]interface{}
	if err := json.NewDecoder(f).Decode(&rawData); err != nil {
		return fmt.Errorf("failed to decode maintenance.json: %w", err)
	}

	// Convert BusNumber to BusID if needed
	migrated := false
	for _, log := range rawData {
		if busNumber, exists := log["bus_number"]; exists {
			log["bus_id"] = busNumber
			delete(log, "bus_number")
			migrated = true
		}
	}

	if migrated {
		f, err := os.Create("data/maintenance.json")
		if err != nil {
			return fmt.Errorf("failed to create migrated maintenance.json: %w", err)
		}
		defer f.Close()

		enc := json.NewEncoder(f)
		enc.SetIndent("", "  ")
		if err := enc.Encode(rawData); err != nil {
			return fmt.Errorf("failed to encode migrated maintenance: %w", err)
		}

		log.Println("Migrated maintenance.json: BusNumber -> BusID")
	}

	return nil
}

// runMigrations executes all necessary data migrations
func runMigrations() error {
	log.Println("Running data migrations...")
	
	if err := migrateBusData(); err != nil {
		return fmt.Errorf("bus migration failed: %w", err)
	}
	
	if err := migrateRouteAssignments(); err != nil {
		return fmt.Errorf("route assignment migration failed: %w", err)
	}
	
	if err := migrateDriverLogs(); err != nil {
		return fmt.Errorf("driver logs migration failed: %w", err)
	}
	
	if err := migrateMaintenanceLogs(); err != nil {
		return fmt.Errorf("maintenance logs migration failed: %w", err)
	}
	
	log.Println("Data migrations completed successfully")
	return nil
}

// Updated initialization with proper ID structure
func initDataFiles() {
	// Ensure data directory exists with proper permissions
	if err := os.MkdirAll("data", 0755); err != nil {
		log.Printf("Warning: failed to create data directory: %v", err)
		return
	}

	// Create buses.json if it doesn't exist, and seed with ID-based data
	if _, err := os.Stat("data/buses.json"); os.IsNotExist(err) {
		defaultBuses := []*Bus{
			{BusID: "BUS001", Status: "active", Model: "Ford Transit", Capacity: 20, OilStatus: "good", TireStatus: "good", MaintenanceNotes: ""},
			{BusID: "BUS002", Status: "active", Model: "Chevrolet Express", Capacity: 25, OilStatus: "due", TireStatus: "good", MaintenanceNotes: "Oil change scheduled"},
			{BusID: "BUS003", Status: "maintenance", Model: "Toyota Coaster", Capacity: 15, OilStatus: "good", TireStatus: "worn", MaintenanceNotes: "Brake inspection in progress"},
		}
		f, err := os.OpenFile("data/buses.json", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			log.Printf("Warning: failed to create buses.json: %v", err)
			return
		}
		defer f.Close()
		enc := json.NewEncoder(f)
		enc.SetIndent("", "  ")
		if err := enc.Encode(defaultBuses); err != nil {
			log.Printf("Warning: failed to encode buses to json: %v", err)
			return
		}
		log.Println("Created and seeded data/buses.json with ID-based structure")
	}

	// Create students.json if it doesn't exist
	if _, err := os.Stat("data/students.json"); os.IsNotExist(err) {
		defaultStudents := []Student{}
		f, err := os.OpenFile("data/students.json", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			log.Printf("Warning: failed to create students.json: %v", err)
			return
		}
		defer f.Close()
		enc := json.NewEncoder(f)
		enc.SetIndent("", "  ")
		if err := enc.Encode(defaultStudents); err != nil {
			log.Printf("Warning: failed to encode students to json: %v", err)
			return
		}
		log.Println("Created data/students.json")
	}

	// Create routes.json if it doesn't exist, and seed with RouteID-based data
	if _, err := os.Stat("data/routes.json"); os.IsNotExist(err) {
		routes := []Route{
			{
				RouteID:   "RT001",
				RouteName: "Victory Square",
				Positions: []struct {
					Position int    `json:"position"`
					Student  string `json:"student"`
				}{{Position: 1, Student: "Alice Johnson"}, {Position: 2, Student: "Bob Smith"}},
			},
			{
				RouteID:   "RT002",
				RouteName: "Airportway",
				Positions: []struct {
					Position int    `json:"position"`
					Student  string `json:"student"`
				}{{Position: 1, Student: "Charlie Brown"}, {Position: 2, Student: "David Wilson"}},
			},
			{
				RouteID:   "RT003",
				RouteName: "NELC",
				Positions: []struct {
					Position int    `json:"position"`
					Student  string `json:"student"`
				}{{Position: 1, Student: "Emma Davis"}, {Position: 2, Student: "Frank Miller"}},
			},
			{
				RouteID:   "RT004",
				RouteName: "Irrigon",
				Positions: []struct {
					Position int    `json:"position"`
					Student  string `json:"student"`
				}{{Position: 1, Student: "Grace Lee"}, {Position: 2, Student: "Henry Clark"}},
			},
			{
				RouteID:   "RT005",
				RouteName: "PELC",
				Positions: []struct {
					Position int    `json:"position"`
					Student  string `json:"student"`
				}{{Position: 1, Student: "Ivy Rodriguez"}, {Position: 2, Student: "Jack Thompson"}},
			},
			{
				RouteID:   "RT006",
				RouteName: "Umatilla",
				Positions: []struct {
					Position int    `json:"position"`
					Student  string `json:"student"`
				}{{Position: 1, Student: "Kate Anderson"}, {Position: 2, Student: "Liam Garcia"}},
			},
		}
		f, err := os.OpenFile("data/routes.json", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			log.Printf("Warning: failed to create routes.json: %v", err)
			return
		}
		defer f.Close()
		enc := json.NewEncoder(f)
		enc.SetIndent("", "  ")
		if err := enc.Encode(routes); err != nil {
			log.Printf("Warning: failed to encode routes to json: %v", err)
			return
		}
		log.Println("Created and seeded data/routes.json with RouteID structure")
	}

	// Create route_assignments.json if it doesn't exist
	if _, err := os.Stat("data/route_assignments.json"); os.IsNotExist(err) {
		defaultAssignments := []RouteAssignment{}
		f, err := os.OpenFile("data/route_assignments.json", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			log.Printf("Warning: failed to create route_assignments.json: %v", err)
			return
		}
		defer f.Close()
		enc := json.NewEncoder(f)
		enc.SetIndent("", "  ")
		if err := enc.Encode(defaultAssignments); err != nil {
			log.Printf("Warning: failed to encode assignments to json: %v", err)
			return
		}
		log.Println("Created data/route_assignments.json")
	}

	// Create maintenance.json if it doesn't exist
	if _, err := os.Stat("data/maintenance.json"); os.IsNotExist(err) {
		defaultMaintenance := []MaintenanceLog{}
		f, err := os.OpenFile("data/maintenance.json", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			log.Printf("Warning: failed to create maintenance.json: %v", err)
			return
		}
		defer f.Close()
		enc := json.NewEncoder(f)
		enc.SetIndent("", "  ")
		if err := enc.Encode(defaultMaintenance); err != nil {
			log.Printf("Warning: failed to encode maintenance to json: %v", err)
			return
		}
		log.Println("Created data/maintenance.json")
	}

	// Create driver_logs.json if it doesn't exist
	if _, err := os.Stat("data/driver_logs.json"); os.IsNotExist(err) {
		defaultLogs := []DriverLog{}
		f, err := os.OpenFile("data/driver_logs.json", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			log.Printf("Warning: failed to create driver_logs.json: %v", err)
			return
		}
		defer f.Close()
		enc := json.NewEncoder(f)
		enc.SetIndent("", "  ")
		if err := enc.Encode(defaultLogs); err != nil {
			log.Printf("Warning: failed to encode driver logs to json: %v", err)
			return
		}
		log.Println("Created data/driver_logs.json")
	}
}

// Add health check endpoint
func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"healthy","timestamp":"` + time.Now().Format(time.RFC3339) + `"}`))
}

// Add CORS headers to prevent cross-origin issues
func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		next(w, r)
	}
}

func main() {
	// Ensure basic data files exist
	ensureDataFiles()
	
	// Run migrations to convert old structure to new ID-based structure
	if err := runMigrations(); err != nil {
		log.Printf("Migration error: %v", err)
		// Don't fatal here, let the app continue even if migrations fail
	}
	
	// Initialize data files with proper structure
	initDataFiles()

	// Setup HTTP routes with recovery and CORS middleware
	http.HandleFunc("/health", corsMiddleware(withRecovery(healthCheck)))
	http.HandleFunc("/", corsMiddleware(withRecovery(loginPage)))
	http.HandleFunc("/new-user", corsMiddleware(withRecovery(newUserPage)))
	http.HandleFunc("/dashboard", corsMiddleware(withRecovery(dashboardRouter)))
	http.HandleFunc("/manager-dashboard", corsMiddleware(withRecovery(managerDashboard)))
	http.HandleFunc("/driver-dashboard", corsMiddleware(withRecovery(driverDashboard)))
	http.HandleFunc("/driver/", corsMiddleware(withRecovery(driverProfileHandler)))
	http.HandleFunc("/assign-routes", corsMiddleware(withRecovery(assignRoutesPage)))
	http.HandleFunc("/assign-route", corsMiddleware(withRecovery(assignRoute)))
	http.HandleFunc("/unassign-route", corsMiddleware(withRecovery(unassignRoute)))
	http.HandleFunc("/fleet", corsMiddleware(withRecovery(fleetPage)))
	http.HandleFunc("/add-bus", corsMiddleware(withRecovery(addBus)))
	http.HandleFunc("/edit-bus", corsMiddleware(withRecovery(editBus)))
	http.HandleFunc("/remove-bus", corsMiddleware(withRecovery(removeBus)))
	http.HandleFunc("/webhook", corsMiddleware(withRecovery(handleWebhook)))
	http.HandleFunc("/pull", corsMiddleware(withRecovery(runPullHandler)))
	http.HandleFunc("/save-log", corsMiddleware(withRecovery(saveDriverLog)))
	http.HandleFunc("/students", corsMiddleware(withRecovery(studentsPage)))
	http.HandleFunc("/add-student", corsMiddleware(withRecovery(addStudent)))
	http.HandleFunc("/edit-student", corsMiddleware(withRecovery(editStudent)))
	http.HandleFunc("/remove-student", corsMiddleware(withRecovery(removeStudent)))
	http.HandleFunc("/add-maint", corsMiddleware(withRecovery(addMaintenanceLog)))
	http.HandleFunc("/logout", corsMiddleware(withRecovery(logout)))

	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}

	server := &http.Server{
		Addr:         "0.0.0.0:" + port,
		Handler:      nil,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second, // Increased for better connection handling
	}

	log.Printf("Server starting on port %s with ID-based data structure", port)
	log.Printf("Data structure: BusID, RouteID, StudentID for consistent identification")
	log.Printf("Health check available at /health")
	
	// Graceful shutdown handling
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()
	
	// Keep the server running
	select {}
}