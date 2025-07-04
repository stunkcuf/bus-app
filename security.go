// security.go - Standalone security utilities
package main

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"html"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
	
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/time/rate"
)

// ==============================================================
// RATE LIMITING - Prevents brute force attacks
// ==============================================================

type RateLimiter struct {
	visitors map[string]*rate.Limiter
	mu       sync.Mutex
	r        rate.Limit
	b        int
}

func NewRateLimiter(r rate.Limit, b int) *RateLimiter {
	return &RateLimiter{
		visitors: make(map[string]*rate.Limiter),
		r:        r,
		b:        b,
	}
}

func (rl *RateLimiter) GetVisitor(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	limiter, exists := rl.visitors[ip]
	if !exists {
		limiter = rate.NewLimiter(rl.r, rl.b)
		rl.visitors[ip] = limiter
		
		// Clean up old entries after 1 hour
		go func(ip string) {
			time.Sleep(time.Hour)
			rl.mu.Lock()
			delete(rl.visitors, ip)
			rl.mu.Unlock()
		}(ip)
	}

	return limiter
}

// Global rate limiter: 10 requests per second, burst of 20
var rateLimiter = NewRateLimiter(10, 20)

// ==============================================================
// INPUT VALIDATION - Prevents SQL injection and XSS
// ==============================================================

var (
	usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]{3,20}$`)
	busIDRegex    = regexp.MustCompile(`^BUS[0-9]{3}$`)
	routeIDRegex  = regexp.MustCompile(`^RT[0-9]{3}$`)
	phoneRegex    = regexp.MustCompile(`^[\d\s\-\(\)\+]{10,20}$`)
	dateRegex     = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
)

// ValidateUsername checks if username is safe
func ValidateUsername(username string) bool {
	return usernameRegex.MatchString(username)
}

// ValidateBusID checks if bus ID matches expected format
func ValidateBusID(busID string) bool {
	return busIDRegex.MatchString(busID)
}

// ValidateRouteID checks if route ID matches expected format  
func ValidateRouteID(routeID string) bool {
	return routeIDRegex.MatchString(routeID)
}

// ValidatePhone checks if phone number is reasonable
func ValidatePhone(phone string) bool {
	if phone == "" {
		return true // optional field
	}
	return phoneRegex.MatchString(phone)
}

// ValidateDate checks if date is in YYYY-MM-DD format
func ValidateDate(date string) bool {
	return dateRegex.MatchString(date)
}

// SanitizeInput removes dangerous characters and limits length
func SanitizeInput(input string) string {
	// Remove null bytes
	input = strings.ReplaceAll(input, "\x00", "")
	
	// Trim whitespace
	input = strings.TrimSpace(input)
	
	// Escape HTML to prevent XSS
	input = html.EscapeString(input)
	
	// Limit length to prevent DOS
	if len(input) > 500 {
		input = input[:500]
	}
	
	return input
}

// SanitizeFormValue is a helper to sanitize form inputs
func SanitizeFormValue(r *http.Request, key string) string {
	return SanitizeInput(r.FormValue(key))
}

// ==============================================================
// PASSWORD HASHING - Replaces plain text passwords
// ==============================================================

// HashPassword creates a bcrypt hash of the password
func HashPassword(password string) (string, error) {
	// Use cost of 12 (good balance of security and speed)
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	return string(bytes), err
}

// CheckPasswordHash compares password with hash
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// ==============================================================
// SESSION MANAGEMENT - More secure than plain cookies
// ==============================================================

type SecureSession struct {
	sessions map[string]*SessionData
	mu       sync.RWMutex
}

type SessionData struct {
	Username  string
	Role      string
	CSRFToken string
	ExpiresAt time.Time
}

var sessionStore = &SecureSession{
	sessions: make(map[string]*SessionData),
}

// GenerateSecureToken creates a cryptographically secure random token
func GenerateSecureToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// CreateSecureSession creates a new secure session
func CreateSecureSession(username, role string) (sessionID, csrfToken string, err error) {
	sessionID, err = GenerateSecureToken()
	if err != nil {
		return "", "", err
	}
	
	csrfToken, err = GenerateSecureToken()
	if err != nil {
		return "", "", err
	}
	
	sessionStore.mu.Lock()
	sessionStore.sessions[sessionID] = &SessionData{
		Username:  username,
		Role:      role,
		CSRFToken: csrfToken,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	sessionStore.mu.Unlock()
	
	// Clean up expired sessions periodically
	go cleanupExpiredSessions()
	
	return sessionID, csrfToken, nil
}

// GetSecureSession retrieves session data
func GetSecureSession(sessionID string) (*SessionData, bool) {
	sessionStore.mu.RLock()
	defer sessionStore.mu.RUnlock()
	
	session, exists := sessionStore.sessions[sessionID]
	if !exists || session.ExpiresAt.Before(time.Now()) {
		return nil, false
	}
	
	return session, true
}

// ValidateCSRFToken checks if the CSRF token is valid for the session
func ValidateCSRFToken(sessionID, token string) bool {
	session, exists := GetSecureSession(sessionID)
	if !exists {
		return false
	}
	
	// Use constant time comparison to prevent timing attacks
	return subtle.ConstantTimeCompare([]byte(session.CSRFToken), []byte(token)) == 1
}

// cleanupExpiredSessions removes expired sessions every hour
func cleanupExpiredSessions() {
	time.Sleep(time.Hour)
	
	sessionStore.mu.Lock()
	defer sessionStore.mu.Unlock()
	
	now := time.Now()
	for id, session := range sessionStore.sessions {
		if session.ExpiresAt.Before(now) {
			delete(sessionStore.sessions, id)
		}
	}
}

// SetSecureCookie sets a secure HTTP-only cookie
func SetSecureCookie(w http.ResponseWriter, name, value string) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		Secure:   true, // Railway provides HTTPS
		SameSite: http.SameSiteStrictMode,
		MaxAge:   86400, // 24 hours
	})
}

// ==============================================================
// SECURE MIDDLEWARE - Drop-in authentication middleware
// ==============================================================

// SecureAuthMiddleware checks if user is authenticated
func SecureAuthMiddleware(requiredRole string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get session cookie
		cookie, err := r.Cookie("session_id")
		if err != nil {
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}
		
		// Validate session
		session, exists := GetSecureSession(cookie.Value)
		if !exists {
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}
		
		// Check role if specified
		if requiredRole != "" && session.Role != requiredRole {
			http.Error(w, "Unauthorized", http.StatusForbidden)
			return
		}
		
		// For POST requests, validate CSRF token
		if r.Method == "POST" {
			csrfToken := r.FormValue("csrf_token")
			if csrfToken == "" {
				csrfToken = r.Header.Get("X-CSRF-Token")
			}
			
			if !ValidateCSRFToken(cookie.Value, csrfToken) {
				http.Error(w, "Invalid CSRF token", http.StatusForbidden)
				return
			}
		}
		
		next(w, r)
	}
}

// ==============================================================
// HELPER FUNCTIONS
// ==============================================================

// GetActiveSessionCount returns the number of active sessions (for metrics)
func GetActiveSessionCount() int {
	sessionStore.mu.RLock()
	defer sessionStore.mu.RUnlock()
	
	count := 0
	now := time.Now()
	for _, session := range sessionStore.sessions {
		if session.ExpiresAt.After(now) {
			count++
		}
	}
	return count
}

// ClearUserSessions removes all sessions for a specific user
func ClearUserSessions(username string) {
	sessionStore.mu.Lock()
	defer sessionStore.mu.Unlock()
	
	for id, session := range sessionStore.sessions {
		if session.Username == username {
			delete(sessionStore.sessions, id)
		}
	}
}

// SecurityHeaders adds security headers to responses
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Security headers
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		
		// HSTS for HTTPS (Railway provides HTTPS)
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		
		// Basic CSP with img-src to allow data URIs for Bootstrap icons
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline' https://cdn.jsdelivr.net; style-src 'self' 'unsafe-inline' https://cdn.jsdelivr.net; font-src 'self' https://cdn.jsdelivr.net; img-src 'self' data: https:;")
		
		next.ServeHTTP(w, r)
	})
}

// ==============================================================
// EXAMPLE HANDLER CREATORS - Use these in your main.go
// ==============================================================

// CreateSecureLoginHandler creates a login handler using your existing functions
func CreateSecureLoginHandler(loginPageFunc func(http.ResponseWriter, *http.Request), 
	loadUsersFunc func() []User,
	checkUserFunc func(username, password string) (*User, bool)) http.HandlerFunc {
	
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			loginPageFunc(w, r)
			return
		}
		
		// Rate limiting check
		if !rateLimiter.GetVisitor(r.RemoteAddr).Allow() {
			http.Error(w, "Too many login attempts", http.StatusTooManyRequests)
			return
		}
		
		// Sanitize inputs
		username := SanitizeFormValue(r, "username")
		password := r.FormValue("password")
		
		// Validate username format
		if !ValidateUsername(username) {
			http.Error(w, "Invalid username format", http.StatusBadRequest)
			return
		}
		
		// Check user credentials
		user, valid := checkUserFunc(username, password)
		if !valid || user == nil {
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			return
		}
		
		// Create secure session
		sessionID, _, err := CreateSecureSession(username, user.Role)
		if err != nil {
			http.Error(w, "Session creation failed", http.StatusInternalServerError)
			return
		}
		
		// Set secure cookie
		SetSecureCookie(w, "session_id", sessionID)
		
		// Redirect based on role
		if user.Role == "manager" {
			http.Redirect(w, r, "/manager-dashboard", http.StatusFound)
		} else {
			http.Redirect(w, r, "/driver-dashboard", http.StatusFound)
		}
	}
}
