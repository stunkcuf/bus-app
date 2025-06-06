package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"os/exec"
)

type User struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Role     string `json:"role"`
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
	templates.ExecuteTemplate(w, "dashboard.html", user)
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

// ✅ GitHub Pull Triggered by Cloudflare Worker
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
	w.Write([]byte("✅ Git pull complete:\n" + string(output)))
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

	// ✅ Add pull sync endpoint
	http.HandleFunc("/run-pull", runPullHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	fmt.Println("Server running on port:", port)
	http.ListenAndServe(":"+port, nil)
}
