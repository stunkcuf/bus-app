package main

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/exec"
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

var templates = template.Must(template.ParseGlob("templates/*.html"))

func ensureDataFiles() {
	os.MkdirAll("data", os.ModePerm)
	if _, err := os.Stat("data/users.json"); os.IsNotExist(err) {
		defaultUsers := []User{{"admin", "adminpass", "manager"}}
		f, _ := os.Create("data/users.json")
		json.NewEncoder(f).Encode(defaultUsers)
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

func dashboard(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	attendance, _ := loadJSON[Attendance]("data/attendance.json")
	mileage, _ := loadJSON[Mileage]("data/mileage.json")
	activities, _ := loadJSON[Activity]("data/activities.json")

	driverData := make(map[string]*DriverSummary)
	routeData := make(map[string]*RouteStats)
	now := time.Now()

	for _, att := range attendance {
		s := driverData[att.Driver]
		if s == nil {
			s = &DriverSummary{Name: att.Driver}
			driverData[att.Driver] = s
		}
		if att.Route == "morning" {
			s.TotalMorning += att.Present
		} else if att.Route == "evening" {
			s.TotalEvening += att.Present
		}
		parsed, _ := time.Parse("2006-01-02", att.Date)
		if parsed.Month() == now.Month() && parsed.Year() == now.Year() {
			s.MonthlyAttendance += att.Present
		}

		route := routeData[att.Route]
		if route == nil {
			route = &RouteStats{RouteName: att.Route}
			routeData[att.Route] = route
		}
		route.AttendanceMonth += att.Present
		if now.Sub(parsed).Hours() < 24 {
			route.AttendanceDay += att.Present
		}
		if now.Sub(parsed).Hours() < 168 {
			route.AttendanceWeek += att.Present
		}
	}

	for _, m := range mileage {
		s := driverData[m.Driver]
		if s == nil {
			s = &DriverSummary{Name: m.Driver}
			driverData[m.Driver] = s
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

	type DashboardData struct {
		User            *User
		Role            string
		DriverSummaries []*DriverSummary
		RouteStats      []*RouteStats
		Activities      []Activity
	}

	// Convert map to slices
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
	}

	templates.ExecuteTemplate(w, "dashboard.html", data)
}

func loginPage(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		r.ParseForm()
		username := r.FormValue("username")
		password := r.FormValue("password")
		for _, u := range loadUsers() {
			if u.Username == username && u.Password == password {
				http.SetCookie(w, &http.Cookie{Name: "session_user", Value: username, Path: "/"})
				http.Redirect(w, r, "/dashboard", http.StatusFound)
				return
			}
		}
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}
	templates.ExecuteTemplate(w, "login.html", nil)
}

func logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{Name: "session_user", Value: "", MaxAge: -1, Path: "/"})
	http.Redirect(w, r, "/", http.StatusFound)
}

func addUserPage(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		r.ParseForm()
		u := User{
			Username: r.FormValue("username"),
			Password: r.FormValue("password"),
			Role:     r.FormValue("role"),
		}
		users := loadUsers()
		users = append(users, u)
		saveUsers(users)
		http.Redirect(w, r, "/users", http.StatusFound)
		return
	}
	templates.ExecuteTemplate(w, "add_user.html", nil)
}

func saveUsers(users []User) {
	f, _ := os.Create("data/users.json")
	defer f.Close()
	json.NewEncoder(f).Encode(users)
}

func usersPage(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	users := loadUsers()
	templates.ExecuteTemplate(w, "users.html", users)
}

func editUserPage(w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("username")
	users := loadUsers()
	var target *User
	for i := range users {
		if users[i].Username == username {
			target = &users[i]
			break
		}
	}
	if target == nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	if r.Method == http.MethodPost {
		r.ParseForm()
		target.Password = r.FormValue("password")
		target.Role = r.FormValue("role")
		saveUsers(users)
		http.Redirect(w, r, "/users", http.StatusFound)
		return
	}

	templates.ExecuteTemplate(w, "edit_user.html", target)
}

func deleteUser(w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("username")
	users := loadUsers()
	newUsers := []User{}
	for _, u := range users {
		if u.Username != username {
			newUsers = append(newUsers, u)
		}
	}
	saveUsers(newUsers)
	http.Redirect(w, r, "/users", http.StatusFound)
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
	cmd := exec.Command("git", "pull", "origin", "main")
	output, err := cmd.CombinedOutput()
	if err != nil {
		http.Error(w, "Git pull failed:\n"+string(output), http.StatusInternalServerError)
		return
	}

	// Restart Go app
	go func() {
		time.Sleep(2 * time.Second)
		exec.Command("bash", "restart_app.sh").Run()
	}()

	w.Write([]byte("✅ Git pull complete:\n" + string(output)))
}

func main() {
	ensureDataFiles()
	http.HandleFunc("/", loginPage)
	http.HandleFunc("/dashboard", dashboard)
	http.HandleFunc("/logout", logout)
	http.HandleFunc("/users", usersPage)
	http.HandleFunc("/add-user", addUserPage)
	http.HandleFunc("/edit-user", editUserPage)
	http.HandleFunc("/delete-user", deleteUser)
	http.HandleFunc("/run-pull", runPullHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Println("Server running on port:", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))

}
