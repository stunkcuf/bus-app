package main

import (
	"encoding/json"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type User struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

type Attendance struct {
	Driver  string  `json:"driver"`
	Route   string  `json:"route"`
	Date    string  `json:"date"`
	Mileage float64 `json:"mileage"`
	Type    string  `json:"type"`
}

type Activity struct {
	Date       string  `json:"date"`
	Driver     string  `json:"driver"`
	TripName   string  `json:"tripName"`
	Attendance int     `json:"attendance"`
	Miles      float64 `json:"miles"`
	Notes      string  `json:"notes"`
}

type DashboardData struct {
	User            string
	Role            string
	DriverSummaries []DriverSummary
	RouteStats      []RouteStat
	Activities      []Activity
}

type DriverSummary struct {
	Name              string
	TotalMorning      int
	TotalEvening      int
	TotalMiles        float64
	MonthlyAvgMiles   float64
	MonthlyAttendance int
}

type RouteStat struct {
	RouteName        string
	TotalMiles       float64
	AvgMiles         float64
	AttendanceDay    int
	AttendanceWeek   int
	AttendanceMonth  int
}

var templates = template.Must(template.ParseGlob("templates/*.html"))

func ensureDataFiles() {
	os.MkdirAll("data", os.ModePerm)
	if _, err := os.Stat("data/users.json"); os.IsNotExist(err) {
		defaultUsers := []User{{"admin", "adminpass", "manager"}}
		file, _ := os.Create("data/users.json")
		json.NewEncoder(file).Encode(defaultUsers)
		file.Close()
	}
}

func loadUsers() []User {
	file, err := os.Open("data/users.json")
	if err != nil {
		return nil
	}
	defer file.Close()
	var users []User
	json.NewDecoder(file).Decode(&users)
	return users
}

func saveUsers(users []User) {
	file, _ := os.Create("data/users.json")
	json.NewEncoder(file).Encode(users)
	file.Close()
}

func getUserFromSession(r *http.Request) *User {
	cookie, err := r.Cookie("session_user")
	if err != nil {
		return nil
	}
	uname := cookie.Value
	for _, user := range loadUsers() {
		if user.Username == uname {
			return &user
		}
	}
	return nil
}

func loginPage(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		r.ParseForm()
		username := r.FormValue("username")
		password := r.FormValue("password")
		for _, user := range loadUsers() {
			if user.Username == username && user.Password == password {
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

func dashboardPage(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	data := DashboardData{
		User: user.Username,
		Role: user.Role,
	}

	if user.Role == "manager" {
		attendancePath := filepath.Join("data", "attendance.json")
		activityPath := filepath.Join("data", "activities.json")

		var records []Attendance
		var activities []Activity

		if f, err := os.ReadFile(attendancePath); err == nil {
			_ = json.Unmarshal(f, &records)
		}

		if f, err := os.ReadFile(activityPath); err == nil {
			_ = json.Unmarshal(f, &activities)
		}

		driverMap := make(map[string]*DriverSummary)
		routeMap := make(map[string]*RouteStat)
		now := time.Now()

		for _, rec := range records {
			// Driver summary
			ds := driverMap[rec.Driver]
			if ds == nil {
				ds = &DriverSummary{Name: rec.Driver}
				driverMap[rec.Driver] = ds
			}
			if rec.Route == "morning" {
				ds.TotalMorning++
			}
			if rec.Route == "evening" {
				ds.TotalEvening++
			}
			ds.TotalMiles += rec.Mileage
			ds.MonthlyAttendance++

			// Route stats
			rs := routeMap[rec.Route]
			if rs == nil {
				rs = &RouteStat{RouteName: rec.Route}
				routeMap[rec.Route] = rs
			}
			rs.TotalMiles += rec.Mileage
			rs.AttendanceMonth++
			if recDate, err := time.Parse("2006-01-02", rec.Date); err == nil {
				if now.Format("2006-01-02") == recDate.Format("2006-01-02") {
					rs.AttendanceDay++
				}
				if now.Sub(recDate).Hours() <= 7*24 {
					rs.AttendanceWeek++
				}
			}
		}

		for _, ds := range driverMap {
			ds.MonthlyAvgMiles = ds.TotalMiles / float64(ds.MonthlyAttendance)
			data.DriverSummaries = append(data.DriverSummaries, *ds)
		}

		for _, rs := range routeMap {
			if rs.AttendanceMonth > 0 {
				rs.AvgMiles = rs.TotalMiles / float64(rs.AttendanceMonth)
			}
			data.RouteStats = append(data.RouteStats, *rs)
		}

		data.Activities = activities
	}

	templates.ExecuteTemplate(w, "dashboard.html", data)
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

func addUserPage(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		r.ParseForm()
		newUser := User{
			Username: r.FormValue("username"),
			Password: r.FormValue("password"),
			Role:     r.FormValue("role"),
		}
		users := loadUsers()
		users = append(users, newUser)
		saveUsers(users)
		http.Redirect(w, r, "/users", http.StatusFound)
		return
	}
	templates.ExecuteTemplate(w, "add_user.html", nil)
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

func editUserPage(w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("username")
	users := loadUsers()
	var userToEdit *User
	for _, u := range users {
		if u.Username == username {
			userToEdit = &u
			break
		}
	}
	if userToEdit == nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	if r.Method == http.MethodPost {
		r.ParseForm()
		for i, u := range users {
			if u.Username == username {
				users[i].Password = r.FormValue("password")
				users[i].Role = r.FormValue("role")
				break
			}
		}
		saveUsers(users)
		http.Redirect(w, r, "/users", http.StatusFound)
		return
	}
	templates.ExecuteTemplate(w, "edit_user.html", userToEdit)
}

func main() {
	ensureDataFiles()
	http.HandleFunc("/", loginPage)
	http.HandleFunc("/dashboard", dashboardPage)
	http.HandleFunc("/logout", logout)
	http.HandleFunc("/users", usersPage)
	http.HandleFunc("/add-user", addUserPage)
	http.HandleFunc("/edit-user", editUserPage)
	http.HandleFunc("/delete-user", deleteUser)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	http.ListenAndServe(":"+port, nil)
}
