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

var templates = template.Must(template.ParseGlob("templates/*.html"))

func ensureDataFiles() {
	os.MkdirAll("data", os.ModePerm)
	if _, err := os.Stat("data/users.json"); os.IsNotExist(err) {
		defaultUsers := []User{{"admin", "adminpass", "manager"}}
		file, err := os.Create("data/users.json")
		if err != nil {
			panic("cannot create users.json")
		}
		defer file.Close()
		json.NewEncoder(file).Encode(defaultUsers)
	}
}

func loadUsers() []User {
	file, err := os.Open("data/users.json")
	if err != nil {
		return nil
	}
	defer file.Close()
	var users []User
	if err := json.NewDecoder(file).Decode(&users); err != nil {
		return nil
	}
	return users
}

func saveUsers(users []User) {
	file, err := os.Create("data/users.json")
	if err != nil {
		fmt.Println("Failed to save users:", err)
		return
	}
	defer file.Close()
	json.NewEncoder(file).Encode(users)
}

func getUserFromSession(r *http.Request) *User {
	cookie, err := r.Cookie("session_user")
	if err != nil {
		return nil
	}
	username := strings.TrimSpace(cookie.Value)
	for _, user := range loadUsers() {
		if user.Username == username {
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
				http.SetCookie(w, &http.Cookie{
					Name:     "session_user",
					Value:    username,
					Path:     "/",
					HttpOnly: true,
				})
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
	http.SetCookie(w, &http.Cookie{
		Name:   "session_user",
		Value:  "",
		MaxAge: -1,
		Path:   "/",
	})
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
	templates.ExecuteTemplate(w, "users.html", loadUsers())
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
