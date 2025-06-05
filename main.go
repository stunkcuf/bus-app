// main.go
package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"strings"
)

type User struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Role     string `json:"role"`
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
	RouteName        string
	TotalMiles       float64
	AvgMiles         float64
	AttendanceDay    int
	AttendanceWeek   int
	AttendanceMonth  int
}

type Activity struct {
	Date       string
	Driver     string
	TripName   string
	Attendance int
	Miles      float64
	Notes      string
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
	uname := strings.TrimSpace(cookie.Value)
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
		username := strings.TrimSpace(r.FormValue("username"))
		password := r.FormValue("password")
		for _, user := range loadUsers() {
			if user.Username == username && user.Password == password {
				http.SetCookie(w, &http.Cookie{Name: "session_user", Value: username, Path: "/", HttpOnly: true})
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
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	driverSummaries := []DriverSummary{
		{"Alice Johnson", 20, 18, 145.6, 72.8, 38},
		{"Bob Smith", 22, 22, 160.0, 80.0, 44},
	}

	routeStats := []RouteStats{
		{"Route A", 300.5, 30.1, 12, 55, 230},
		{"Route B", 280.0, 28.0, 10, 48, 210},
	}

	activities := []Activity{
		{"2025-06-03", "Alice Johnson", "Field Trip - Zoo", 30, 14.2, "Went well"},
		{"2025-06-04", "Bob Smith", "Sports Event", 25, 18.6, "Late return"},
	}

	templates.ExecuteTemplate(w, "dashboard.html", struct {
		Username        string
		Role            string
		DriverSummaries []DriverSummary
		RouteStats      []RouteStats
		Activities      []Activity
	}{
		Username:        user.Username,
		Role:            user.Role,
		DriverSummaries: driverSummaries,
		RouteStats:      routeStats,
		Activities:      activities,
	})
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
	currentUser := getUserFromSession(r)
	if currentUser == nil || currentUser.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	if r.Method == http.MethodPost {
		r.ParseForm()
		username := strings.TrimSpace(r.FormValue("username"))
		password := r.FormValue("password")
		role := r.FormValue("role")

		if username == "" || password == "" {
			http.Error(w, "Username and password cannot be empty", http.StatusBadRequest)
			return
		}

		users := loadUsers()
		for _, u := range users {
			if u.Username == username {
				http.Error(w, "Username already exists", http.StatusConflict)
				return
			}
		}
		users = append(users, User{Username: username, Password: password, Role: role})
		saveUsers(users)
		http.Redirect(w, r, "/users", http.StatusFound)
		return
	}
	templates.ExecuteTemplate(w, "add_user.html", nil)
}

func deleteUser(w http.ResponseWriter, r *http.Request) {
	currentUser := getUserFromSession(r)
	if currentUser == nil || currentUser.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	username := r.URL.Query().Get("username")
	if username == "admin" {
		http.Error(w, "Cannot delete admin user", http.StatusForbidden)
		return
	}
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
	currentUser := getUserFromSession(r)
	if currentUser == nil || currentUser.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	username := r.URL.Query().Get("username")
	users := loadUsers()
	var userToEdit *User
	for i := range users {
		if users[i].Username == username {
			userToEdit = &users[i]
			break
		}
	}
	if userToEdit == nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	if r.Method == http.MethodPost {
		r.ParseForm()
		password := r.FormValue("password")
		role := r.FormValue("role")
		userToEdit.Password = password
		userToEdit.Role = role
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
	fmt.Println("Server running on port:", port)
	http.ListenAndServe(":"+port, nil)
}
