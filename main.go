package main

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
	"os/exec"  //run start.sh 
	"strconv" // add to importblock
	"fmt"

	git "github.com/go-git/go-git/v5"
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
	BusNumber string `json:"bus_number"`
	RouteName string `json:"route_name"`
	Positions []struct {
		Position int    `json:"position"`
		Student  string `json:"student"`
	} `json:"positions"`
}

type DriverLog struct {
	Driver    string `json:"driver"`
	BusNumber string `json:"bus_number"`
	Date      string `json:"date"`
	Period    string `json:"period"`
	Departure string `json:"departure_time"`
	Arrival   string `json:"arrival_time"`
	Mileage   float64
	Attendance []struct {
		Position   int    `json:"position"`
		Present    bool   `json:"present"`
		PickupTime string `json:"pickup_time,omitempty"`
	} `json:"attendance"`
}

type RouteAssignment struct {
	Driver    string `json:"driver"`
	BusNumber string `json:"bus_number"`
	RouteName string `json:"route_name"`
	AssignedDate string `json:"assigned_date"`
}

type DashboardData struct {
	User            *User
	Role            string
	DriverSummaries []*DriverSummary
	RouteStats      []*RouteStats
	Activities      []Activity
	Routes          []Route
	Users           []User
}

type AssignRouteData struct {
	User            *User
	Assignments     []RouteAssignment
	Drivers         []User
	AvailableRoutes []Route
}

var templates = template.Must(template.ParseGlob("templates/*.html"))

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
	}

	var driverRoute *Route
	
	// First check if driver has an assigned route
	var assignedBusNumber string
	for _, a := range assignments {
		if a.Driver == user.Username {
			assignedBusNumber = a.BusNumber
			break
		}
	}

	// Find the route for the assigned bus or from existing log
	for _, r := range routes {
		if assignedBusNumber != "" && r.BusNumber == assignedBusNumber {
			driverRoute = &r
			break
		} else if driverLog != nil && r.BusNumber == driverLog.BusNumber {
			driverRoute = &r
			break
		}
	}

	data := PageData{
		User:      user,
		Date:      date,
		Period:    period,
		Route:     driverRoute,
		DriverLog: driverLog,
	}

	if driverRoute == nil && driverLog != nil {
		log.Printf("Warning: No route found for bus number %s", driverLog.BusNumber)
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
	for _, rt := range routes {
		if rt.BusNumber == busNumber {
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
		logs = append(logs, DriverLog{
			Driver:     user.Username,
			BusNumber:  busNumber,
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

	// Filter drivers only
	var drivers []User
	for _, u := range users {
		if u.Role == "driver" {
			drivers = append(drivers, u)
		}
	}

	// Find available routes (not assigned)
	assignedBuses := make(map[string]bool)
	for _, a := range assignments {
		assignedBuses[a.BusNumber] = true
	}

	var availableRoutes []Route
	for _, r := range routes {
		if !assignedBuses[r.BusNumber] {
			availableRoutes = append(availableRoutes, r)
		}
	}

	data := AssignRouteData{
		User:            user,
		Assignments:     assignments,
		Drivers:         drivers,
		AvailableRoutes: availableRoutes,
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

	if driver == "" || busNumber == "" {
		http.Redirect(w, r, "/assign-routes", http.StatusFound)
		return
	}

	// Find route name
	routes, _ := loadRoutes()
	var routeName string
	for _, rt := range routes {
		if rt.BusNumber == busNumber {
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
			assignments[i].RouteName = routeName
			assignments[i].AssignedDate = time.Now().Format("2006-01-02")
			saveRouteAssignments(assignments)
			http.Redirect(w, r, "/assign-routes", http.StatusFound)
			return
		}
	}

	// Check if bus is already assigned
	for _, a := range assignments {
		if a.BusNumber == busNumber {
			http.Redirect(w, r, "/assign-routes", http.StatusFound)
			return
		}
	}

	// Add new assignment
	newAssignment := RouteAssignment{
		Driver:       driver,
		BusNumber:    busNumber,
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

func main() {
	ensureDataFiles()
	http.HandleFunc("/", withRecovery(loginPage))
	http.HandleFunc("/new-user", withRecovery(newUserPage))
	http.HandleFunc("/dashboard", withRecovery(dashboardRouter))
	http.HandleFunc("/manager-dashboard", withRecovery(managerDashboard))
	http.HandleFunc("/driver-dashboard", withRecovery(driverDashboard))
	http.HandleFunc("/driver/", withRecovery(driverProfileHandler))
	http.HandleFunc("/assign-routes", withRecovery(assignRoutesPage))
	http.HandleFunc("/assign-route", withRecovery(assignRoute))
	http.HandleFunc("/unassign-route", withRecovery(unassignRoute))
	http.HandleFunc("/webhook", withRecovery(handleWebhook))
	http.HandleFunc("/pull", withRecovery(runPullHandler))
	http.HandleFunc("/save-log", withRecovery(saveDriverLog))
	http.HandleFunc("/logout", withRecovery(logout))


	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}
	log.Println("Watching for changes...")
	log.Printf("Server starting on 0.0.0.0:%s", port)
	log.Fatal(http.ListenAndServe("0.0.0.0:"+port, nil))
}
