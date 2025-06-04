package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
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

func usersPage(w http.ResponseWriter, r *http.Request) {
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

		f, _ := os.Create("data/users.json")
		json.NewEncoder(f).Encode(users)
		f.Close()

		http.Redirect(w, r, "/users", http.StatusFound)
		return
	}
	templates.ExecuteTemplate(w, "add_user.html", nil)
}


func loginPage(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		r.ParseForm()
		username := r.FormValue("username")
		password := r.FormValue("password")
		for _, user := range loadUsers() {
			if user.Username == username && user.Password == password {
				http.Redirect(w, r, "/dashboard?role="+user.Role+"&user="+user.Username, http.StatusFound)
				return
			}
		}
		fmt.Fprintf(w, "Invalid credentials")
		return
	}
	templates.ExecuteTemplate(w, "login.html", nil)
}

func dashboardPage(w http.ResponseWriter, r *http.Request) {
	role := r.URL.Query().Get("role")
	user := r.URL.Query().Get("user")
	templates.ExecuteTemplate(w, "dashboard.html", map[string]string{
		"Role": role,
		"User": user,
	})
}

func main() {
	ensureDataFiles()
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	http.HandleFunc("/", loginPage)
	http.HandleFunc("/dashboard", dashboardPage)
	http.HandleFunc("/users", usersPage)
	http.HandleFunc("/add-user", addUserPage)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	fmt.Println("Server running on port:", port)
	http.ListenAndServe(":"+port, nil)
}
