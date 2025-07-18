package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// Session represents a user session
type Session struct {
	Username    string
	UserRole    string
	CSRFToken   string
	CreatedAt   time.Time
	LastAccess  time.Time
}

// Rate limiter
type RateLimiter struct {
	mu       sync.Mutex
	attempts map[string][]time.Time
	limit    int
	window   time.Duration
}

// Global variables
var (
	sessions      = make(map[string]*Session)
	sessionsMutex sync.RWMutex
	rateLimiter   = NewRateLimiter(20, 15*time.Minute) // 20 attempts per 15 minutes (increased for development)
)

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		attempts: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}
	// Start cleanup routine
	go rl.cleanup()
	return rl
}

// Allow checks if the request should be allowed
func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-rl.window)

	// Get attempts for this IP
	attempts, exists := rl.attempts[ip]
	if !exists {
		rl.attempts[ip] = []time.Time{now}
		return true
	}

	// Filter out old attempts
	var validAttempts []time.Time
	for _, attempt := range attempts {
		if attempt.After(windowStart) {
			validAttempts = append(validAttempts, attempt)
		}
	}

	// Check if under limit
	if len(validAttempts) < rl.limit {
		validAttempts = append(validAttempts, now)
		rl.attempts[ip] = validAttempts
		return true
	}

	rl.attempts[ip] = validAttempts
	return false
}

// Reset clears all rate limit records
func (rl *RateLimiter) Reset() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.attempts = make(map[string][]time.Time)
}

// ResetIP clears rate limit records for a specific IP
func (rl *RateLimiter) ResetIP(ip string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	delete(rl.attempts, ip)
}

// cleanup removes old entries periodically
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		windowStart := now.Add(-rl.window)

		for ip, attempts := range rl.attempts {
			var validAttempts []time.Time
			for _, attempt := range attempts {
				if attempt.After(windowStart) {
					validAttempts = append(validAttempts, attempt)
				}
			}
			if len(validAttempts) == 0 {
				delete(rl.attempts, ip)
			} else {
				rl.attempts[ip] = validAttempts
			}
		}
		rl.mu.Unlock()
	}
}

// getClientIP extracts the client IP address from the request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first (for proxies)
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		// Take the first IP in the chain
		ips := strings.Split(forwarded, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check X-Real-IP header
	realIP := r.Header.Get("X-Real-IP")
	if realIP != "" {
		return realIP
	}

	// Fall back to RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// authenticateUser verifies username and password
func authenticateUser(username, password string) (*User, error) {
	log.Printf("DEBUG: authenticateUser called for username: %s", username)
	
	if db == nil {
		log.Printf("ERROR: Authentication failed - database connection is nil")
		return nil, fmt.Errorf("database not initialized")
	}
	
	// Test database connection
	if err := db.Ping(); err != nil {
		log.Printf("ERROR: Database ping failed: %v", err)
		return nil, fmt.Errorf("database connection lost")
	}
	
	// First, let's check if the users table exists
	var tableExists bool
	tableCheckErr := db.Get(&tableExists, `
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name = 'users'
		)
	`)
	if tableCheckErr != nil {
		log.Printf("ERROR: Failed to check if users table exists: %v", tableCheckErr)
	} else {
		log.Printf("DEBUG: Users table exists: %v", tableExists)
	}
	
	// Try to count total users
	var userCount int
	countErr := db.Get(&userCount, "SELECT COUNT(*) FROM users")
	if countErr != nil {
		log.Printf("ERROR: Failed to count users: %v", countErr)
	} else {
		log.Printf("DEBUG: Total users in database: %d", userCount)
	}

	var user User
	// Use specific column selection to avoid any schema mismatch issues
	err := db.Get(&user, `
		SELECT username, password, role, status, registration_date, created_at 
		FROM users 
		WHERE username = $1
	`, username)
	if err != nil {
		log.Printf("ERROR: Database query failed for user %s: %v", username, err)
		log.Printf("ERROR: SQL Error Type: %T", err)
		
		// Try a simpler query to debug - just get password and role
		var debugUser struct {
			Username string `db:"username"`
			Password string `db:"password"`
			Role     string `db:"role"`
		}
		debugErr := db.Get(&debugUser, "SELECT username, password, role FROM users WHERE username = $1", username)
		if debugErr != nil {
			log.Printf("ERROR: Even simple query failed: %v", debugErr)
			
			// Let's list all users to see what's in the database
			var allUsers []string
			listErr := db.Select(&allUsers, "SELECT username FROM users")
			if listErr != nil {
				log.Printf("ERROR: Failed to list users: %v", listErr)
			} else {
				log.Printf("DEBUG: All users in database: %v", allUsers)
			}
		} else {
			log.Printf("DEBUG: Found user via simple query: %s (role: %s)", debugUser.Username, debugUser.Role)
			// Use the debug user data for authentication
			user.Username = debugUser.Username
			user.Password = debugUser.Password
			user.Role = debugUser.Role
			user.Status = "active" // Default to active for now
			err = nil // Clear the error since we got the data we need
		}
		
		if err != nil {
			return nil, fmt.Errorf("invalid credentials")
		}
	}

	log.Printf("User %s found, checking password (hashed: %v)", username, strings.HasPrefix(user.Password, "$2"))

	// Check if password is hashed (bcrypt hashes start with $2)
	if strings.HasPrefix(user.Password, "$2") {
		// Verify bcrypt password
		err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
		if err != nil {
			log.Printf("Authentication failed for %s: incorrect password", username)
			return nil, fmt.Errorf("invalid credentials")
		}
	} else {
		// Legacy plain text password (should be migrated)
		log.Printf("WARNING: User %s has plain text password, needs migration", username)
		if user.Password != password {
			return nil, fmt.Errorf("invalid credentials")
		}
		// Optionally hash the password here for migration
		go migrateUserPassword(username, password)
	}

	return &user, nil
}

// migrateUserPassword updates a plain text password to bcrypt
func migrateUserPassword(username, plainPassword string) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(plainPassword), bcrypt.DefaultCost)
	if err != nil {
		return
	}

	_, _ = db.Exec("UPDATE users SET password = $1 WHERE username = $2", string(hashedPassword), username)
}

// generateSessionToken creates a new session token
func generateSessionToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

// generateCSRFToken creates a new CSRF token
func generateCSRFToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

// storeSession stores a session for a user
func storeSession(token string, user *User) {
	sessionsMutex.Lock()
	defer sessionsMutex.Unlock()

	sessions[token] = &Session{
		Username:   user.Username,
		UserRole:   user.Role,
		CSRFToken:  generateCSRFToken(),
		CreatedAt:  time.Now(),
		LastAccess: time.Now(),
	}
}

// GetSecureSession retrieves a session by token
func GetSecureSession(token string) (*Session, error) {
	sessionsMutex.RLock()
	defer sessionsMutex.RUnlock()

	session, exists := sessions[token]
	if !exists {
		return nil, fmt.Errorf("session not found")
	}

	// Check if session is expired (24 hours)
	if time.Since(session.CreatedAt) > 24*time.Hour {
		return nil, fmt.Errorf("session expired")
	}

	// Update last access time
	session.LastAccess = time.Now()
	return session, nil
}

// GetSession retrieves a session from HTTP request
func GetSession(r *http.Request) (*Session, error) {
	cookie, err := r.Cookie(SessionCookieName)
	if err != nil {
		return nil, fmt.Errorf("no session cookie")
	}
	
	return GetSecureSession(cookie.Value)
}

// deleteSession removes a session
func deleteSession(token string) {
	sessionsMutex.Lock()
	defer sessionsMutex.Unlock()
	delete(sessions, token)
}

// getUserFromSession gets the user from the current session
func getUserFromSession(r *http.Request) *User {
	cookie, err := r.Cookie(SessionCookieName)
	if err != nil {
		return nil
	}

	session, err := GetSecureSession(cookie.Value)
	if err != nil {
		return nil
	}

	return &User{
		Username: session.Username,
		Role:     session.UserRole,
	}
}

// createUser creates a new user
func createUser(username, password, role, status string) error {
	if db == nil {
		return fmt.Errorf("database not initialized")
	}

	// Validate username
	if len(username) < 3 || len(username) > 20 {
		return fmt.Errorf("username must be between 3 and 20 characters")
	}

	// Validate password
	if len(password) < 6 {
		return fmt.Errorf("password must be at least 6 characters")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Insert user
	_, err = db.Exec(`
		INSERT INTO users (username, password, role, status, registration_date)
		VALUES ($1, $2, $3, $4, CURRENT_DATE)
	`, username, string(hashedPassword), role, status)
	
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique constraint") {
			return fmt.Errorf("username already exists")
		}
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// GetActiveSessionCount returns the number of active sessions
func GetActiveSessionCount() int {
	sessionsMutex.RLock()
	defer sessionsMutex.RUnlock()
	
	count := 0
	now := time.Now()
	for _, session := range sessions {
		if now.Sub(session.CreatedAt) < 24*time.Hour {
			count++
		}
	}
	return count
}

// periodicSessionCleanup removes expired sessions
func periodicSessionCleanup() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		sessionsMutex.Lock()
		now := time.Now()
		for token, session := range sessions {
			if now.Sub(session.CreatedAt) > 24*time.Hour {
				delete(sessions, token)
			}
		}
		sessionsMutex.Unlock()
	}
}
