package main

import (
	"embed"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"time"

	"golang.org/x/crypto/bcrypt"

	"yourproject/security"
	"yourproject/utils"
)

//go:embed templates/*.html
var tmplFS embed.FS

var templates *template.Template

func init() {
	funcMap := template.FuncMap{
		"json": func(v interface{}) template.JS {
			b, err := json.Marshal(v)
			if err != nil {
				log.Printf("JSON marshal error: %v", err)
				return template.JS("{}")
			}
			return template.JS(b)
		},
		"add": func(a, b int) int { return a + b },
		"len": func(v interface{}) int {
			switch s := v.(type) {
			case []interface{}:
				return len(s)
			case []*security.Bus:
				return len(s)
			case []security.Bus:
				return len(s)
			default:
				return 0
			}
		},
	}

	templates = template.Must(template.New("templates").Funcs(funcMap).ParseFS(tmplFS, "templates/*.html"))
}

// Secure session management uses security package
var sessionMgr = security.NewSessionManager()

// HTTP handlers delegate to security package
func loginPage(w http.ResponseWriter, r *http.Request)  { security.LoginPage(w, r, templates) }
func newUserPage(w http.ResponseWriter, r *http.Request) { security.NewUserPage(w, r, templates) }
func editUserPage(w http.ResponseWriter, r *http.Request) { security.EditUserPage(w, r, templates) }
func managerDashboard(w http.ResponseWriter, r *http.Request) { security.ManagerDashboard(w, r, templates) }
func driverDashboard(w http.ResponseWriter, r *http.Request) { security.DriverDashboard(w, r, templates) }
func saveDriverLogHandler(w http.ResponseWriter, r *http.Request) { security.SaveDriverLog(w, r) }
func removeUser(w http.ResponseWriter, r *http.Request)     { security.RemoveUser(w, r) }
func logout(w http.ResponseWriter, r *http.Request)         { security.Logout(w, r) }
func dashboardRouter(w http.ResponseWriter, r *http.Request) { security.DashboardRouter(w, r, templates) }

func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}

func main() {
	// Database setup
	log.Println("🗄️  Setting up PostgreSQL database...")
	if err := utils.SetupDatabase(); err != nil {
		log.Fatalf("Database setup failed: %v", err)
	}
	defer utils.CloseDatabase()

	mux := http.NewServeMux()

	// Public routes
	mux.HandleFunc("/", utils.WithRecovery(utils.RateLimit(loginPage)))
	mux.HandleFunc("/logout", utils.WithRecovery(logout))
	mux.HandleFunc("/health", utils.WithRecovery(healthCheck))

	// Protected routes
	routes := map[string]http.HandlerFunc{
		"/new-user":          newUserPage,
		"/edit-user":         editUserPage,
		"/dashboard":         dashboardRouter,
		"/manager-dashboard": managerDashboard,
		"/driver-dashboard":  driverDashboard,
		"/save-log":          saveDriverLogHandler,
		"/remove-user":       removeUser,
	}
	for path, handler := range routes {
		mux.HandleFunc(path, utils.WithRecovery(utils.SecurityHeaders(handler)))
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}

	server := &http.Server{
		Addr:           fmt.Sprintf(":%s", port),
		Handler:        mux,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   60 * time.Second,
		IdleTimeout:    120 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	log.Printf("Server starting on port %s", port)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
