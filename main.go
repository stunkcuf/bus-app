package main

import (
	"crypto/subtle"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

//go:embed templates/*.html
var tmplFS embed.FS

var templates *template.Template

func init() {
	var err error

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
			case []*Bus:
				return len(s)
			case []Bus:
				return len(s)
			default:
				return 0
			}
		},
	}

	templates, err = template.New("templates").Funcs(funcMap).ParseFS(tmplFS, "templates/*.html")
	if err != nil {
		log.Fatalf("Template parsing failed: %v", err)
	}

	log.Println("Templates loaded successfully")
}

// =============================================================================
// SECURE SESSION MANAGEMENT
// =============================================================================

type SessionManager struct {
	sessions map[string]*SessionData
	mu       sync.RWMutex
}

var sessionMgr = &SessionManager{
	sessions: make(map[string]*SessionData),
}

// SessionData defined once below

type SessionData struct {
	Username  string
	Role      string
	CSRFToken string
	ExpiresAt time.Time
}

// =============================================================================
// HTTP HANDLERS
// =============================================================================

func loginPage(w http.ResponseWriter, r *http.Request) {
	// ... existing loginPage implementation ...
}

func newUserPage(w http.ResponseWriter, r *http.Request) {
	// ... existing newUserPage implementation ...
}

func editUserPage(w http.ResponseWriter, r *http.Request) {
	// ... existing editUserPage implementation ...
}

func managerDashboard(w http.ResponseWriter, r *http.Request) {
	// ... existing managerDashboard implementation ...
}

func driverDashboard(w http.ResponseWriter, r *http.Request) {
	// ... existing driverDashboard implementation ...
}

func saveDriverLogHandler(w http.ResponseWriter, r *http.Request) {
	// ... existing saveDriverLogHandler implementation ...
}

func removeUser(w http.ResponseWriter, r *http.Request) {
	// ... existing removeUser implementation ...
}

func logout(w http.ResponseWriter, r *http.Request) {
	// ... existing logout implementation ...
}

func dashboardRouter(w http.ResponseWriter, r *http.Request) {
	// ... existing dashboardRouter implementation ...
}

// healthCheck handler was missing; added here
func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}

// =============================================================================
// HELPER FUNCTIONS (only unique implementations kept)
// =============================================================================

func getSecureUser(r *http.Request) *User {
	// ... existing getSecureUser implementation ...
}

func SanitizeInput(input string) string {
	return strings.TrimSpace(input)
}

func ValidateUsername(username string) bool {
	return len(username) > 2
}

func ValidateDate(date string) bool {
	_, err := time.Parse("2006-01-02", date)
	return err == nil
}

func SetSecureCookie(w http.ResponseWriter, name, value string) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		HttpOnly: true,
		Path:     "/",
		Expires:  time.Now().Add(24 * time.Hour),
		Secure:   false,
		SameSite: http.SameSiteStrictMode,
	})
}

func executeTemplate(w http.ResponseWriter, tmpl string, data interface{}) {
	err := templates.ExecuteTemplate(w, tmpl, data)
	if err != nil {
		http.Error(w, "Template rendering error", http.StatusInternalServerError)
		log.Printf("Template execution error for %s: %v", tmpl, err)
	}
}

func CreateSecureSession(username, role string) (string, string, error) {
	sessionID := fmt.Sprintf("%d_%s", time.Now().UnixNano(), username)
	csrfToken := fmt.Sprintf("%x", time.Now().UnixNano())
	return sessionID, csrfToken, nil
}

func getCSRFToken(r *http.Request) string {
	cookie, err := r.Cookie("csrf_token")
	if err != nil {
		return ""
	}
	return cookie.Value
}

func verifyCSRFToken(r *http.Request) bool {
	sessionCookie, err := r.Cookie("session_id")
	if err != nil {
		return false
	}

	csrfToken := r.FormValue("csrf_token")
	if csrfToken == "" {
		csrfToken = r.Header.Get("X-CSRF-Token")
	}

	sessionMgr.mu.RLock()
	session, exists := sessionMgr.sessions[sessionCookie.Value]
	sessionMgr.mu.RUnlock()

	if !exists {
		return false
	}

	return subtle.ConstantTimeCompare([]byte(session.CSRFToken), []byte(csrfToken)) == 1
}

// =============================================================================
// MAIN FUNCTION
// =============================================================================

func main() {
	// Setup database
	log.Println("🗄️  Setting up PostgreSQL database...")
	setupDatabase()
	defer closeDatabase()
	log.Println("✅ Database setup complete")

	// Setup HTTP routes with security middleware
	mux := http.NewServeMux()

	// Public routes
	mux.HandleFunc("/", withRecovery(RateLimitMiddleware(loginPage)))
	mux.HandleFunc("/logout", withRecovery(logout))
	mux.HandleFunc("/health", withRecovery(healthCheck))

	// Protected routes
	mux.HandleFunc("/new-user", withRecovery(SecurityHeaders(newUserPage)))
	mux.HandleFunc("/edit-user", withRecovery(SecurityHeaders(editUserPage)))
	mux.HandleFunc("/dashboard", withRecovery(SecurityHeaders(dashboardRouter)))
	mux.HandleFunc("/manager-dashboard", withRecovery(SecurityHeaders(managerDashboard)))
	mux.HandleFunc("/driver-dashboard", withRecovery(SecurityHeaders(driverDashboard)))
	mux.HandleFunc("/save-log", withRecovery(SecurityHeaders(saveDriverLogHandler)))
	mux.HandleFunc("/remove-user", withRecovery(SecurityHeaders(removeUser)))

	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}

	server := &http.Server{
		Addr:           "0.0.0.0:" + port,
		Handler:        mux,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   60 * time.Second,
		IdleTimeout:    120 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	log.Printf("Server starting on port %s", port)
	log.Printf("Server accessible at: http://0.0.0.0:%s", port)

	if err := server.ListenAndServe(); err != nil {
		if err == http.ErrServerClosed {
			log.Println("Server was closed")
		} else {
			log.Printf("Server failed to start: %v", err)
			os.Exit(1)
		}
	}
}
