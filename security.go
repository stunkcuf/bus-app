package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// Session represents a user session
type Session struct {
	SessionID    string
	Username     string
	Role         string
	CSRFToken    string
	CreatedAt    time.Time
	LastAccessed time.Time
	ExpiresAt    time.Time
}

// Session storage
var (
	sessions    = make(map[string]*Session)
	sessionsMux sync.RWMutex
)

// Constants for security
const (
	SessionDuration = 24 * time.Hour
	CSRFTokenLength = 32
	TokenLength     = 32
)

// FIXED: Generate nonce for CSP headers
func generateNonce() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		log.Printf("Error generating nonce: %v", err)
		return ""
	}
	return base64.StdEncoding.EncodeToString(b)
}

// FIXED: SecurityHeaders middleware with proper CSP implementation
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Generate nonce for this request
		nonce := generateNonce()
		
		// Store nonce in context for templates
		ctx := context.WithValue(r.Context(), "csp-nonce", nonce)
		r = r.WithContext(ctx)
		
		// Security headers
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		
		// FIXED: Proper CSP without unsafe-inline
		csp := fmt.Sprintf(
			"default-src 'self'; "+
				"script-src 'self' 'nonce-%s' https://cdn.jsdelivr.net https://unpkg.com; "+
				"style-src 'self' 'nonce-%s' https://cdn.jsdelivr.net https://unpkg.com; "+
				"font-src 'self' https://cdn.jsdelivr.net; "+
				"img-src 'self' data: https:; "+
				"connect-src 'self'; "+
				"frame-ancestors 'none'; "+
				"base-uri 'self'; "+
				"form-action 'self'",
			nonce, nonce)
		
		w.Header().Set("Content-Security-Policy", csp)
		
		// HSTS for HTTPS
		if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		
		next.ServeHTTP(w, r)
	})
}

// FIXED: Single cleanup goroutine to avoid race conditions
func periodicSessionCleanup() {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	
	for range ticker.C {
		cleanupExpiredSessions()
	}
}

// CreateSecureSession creates a new session with CSRF protection
func CreateSecureSession(username, role string) (string, string, error) {
	sessionID, err := GenerateSecureToken()
	if err != nil {
		return "", "", err
	}
	
	csrfToken, err := GenerateSecureToken()
	if err != nil {
		return "", "", err
	}
	
	now := time.Now()
	session := &Session{
		SessionID:    sessionID,
		Username:     username,
		Role:         role,
		CSRFToken:    csrfToken,
		CreatedAt:    now,
		LastAccessed: now,
		ExpiresAt:    now.Add(SessionDuration),
	}
	
	sessionsMux.Lock()
	sessions[sessionID] = session
	sessionsMux.Unlock()
	
	log.Printf("Created session for user %s with role %s", username, role)
	
	return sessionID, csrfToken, nil
}

// GetSecureSession retrieves and validates a session
func GetSecureSession(sessionID string) (*Session, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("empty session ID")
	}
	
	sessionsMux.Lock()
	defer sessionsMux.Unlock()
	
	session, exists := sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session not found")
	}
	
	// Check expiration
	if time.Now().After(session.ExpiresAt) {
		delete(sessions, sessionID)
		return nil, fmt.Errorf("session expired")
	}
	
	// Update last accessed time
	session.LastAccessed = time.Now()
	
	// Extend session if it's been active
	if time.Since(session.CreatedAt) < SessionDuration/2 {
		session.ExpiresAt = time.Now().Add(SessionDuration)
	}
	
	return session, nil
}

// ValidateCSRFToken validates a CSRF token for a session
func ValidateCSRFToken(sessionID, token string) bool {
	if sessionID == "" || token == "" {
		return false
	}
	
	session, err := GetSecureSession(sessionID)
	if err != nil {
		return false
	}
	
	return session.CSRFToken == token
}

// ClearSession removes a session
func ClearSession(sessionID string) {
	sessionsMux.Lock()
	defer sessionsMux.Unlock()
	
	if session, exists := sessions[sessionID]; exists {
		log.Printf("Clearing session for user %s", session.Username)
		delete(sessions, sessionID)
	}
}

// ClearUserSessions removes all sessions for a specific user
func ClearUserSessions(username string) {
	sessionsMux.Lock()
	defer sessionsMux.Unlock()
	
	for sessionID, session := range sessions {
		if session.Username == username {
			log.Printf("Clearing session %s for user %s", sessionID, username)
			delete(sessions, sessionID)
		}
	}
}

// cleanupExpiredSessions removes expired sessions
func cleanupExpiredSessions() {
	sessionsMux.Lock()
	defer sessionsMux.Unlock()
	
	now := time.Now()
	expired := 0
	
	for sessionID, session := range sessions {
		if now.After(session.ExpiresAt) {
			delete(sessions, sessionID)
			expired++
		}
	}
	
	if expired > 0 {
		log.Printf("Cleaned up %d expired sessions", expired)
	}
}

// GetActiveSessionCount returns the number of active sessions
func GetActiveSessionCount() int {
	sessionsMux.RLock()
	defer sessionsMux.RUnlock()
	return len(sessions)
}

// GenerateSecureToken generates a cryptographically secure random token
func GenerateSecureToken() (string, error) {
	bytes := make([]byte, TokenLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// HashPassword hashes a password using bcrypt
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// CheckPasswordHash compares a password with its hash
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// SetSecureCookie sets a secure HTTP-only cookie
func SetSecureCookie(w http.ResponseWriter, name, value string) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		MaxAge:   int(SessionDuration.Seconds()),
		HttpOnly: true,
		Secure:   true, // Always use secure in production
		SameSite: http.SameSiteStrictMode,
	})
}

// Input validation functions

// ValidateUsername validates username format
func ValidateUsername(username string) bool {
	if len(username) < 3 || len(username) > 20 {
		return false
	}
	// Only allow alphanumeric characters and underscores
	match, _ := regexp.MatchString("^[a-zA-Z0-9_]+$", username)
	return match
}

// SanitizeInput removes potentially dangerous characters from input
func SanitizeInput(input string) string {
	// Remove any HTML tags
	re := regexp.MustCompile("<[^>]*>")
	input = re.ReplaceAllString(input, "")
	
	// Trim whitespace
	input = strings.TrimSpace(input)
	
	// Limit length
	if len(input) > 1000 {
		input = input[:1000]
	}
	
	return input
}

// SanitizeFormValue gets and sanitizes a form value
func SanitizeFormValue(r *http.Request, key string) string {
	return SanitizeInput(r.FormValue(key))
}

// Rate limiting
type RateLimiter struct {
	attempts map[string][]time.Time
	mu       sync.Mutex
	limit    int
	window   time.Duration
}

var loginLimiter = &RateLimiter{
	attempts: make(map[string][]time.Time),
	limit:    5,
	window:   15 * time.Minute,
}

// checkRateLimit checks if an IP has exceeded the rate limit
func (rl *RateLimiter) checkRateLimit(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	
	now := time.Now()
	
	// Clean up old attempts
	if attempts, exists := rl.attempts[ip]; exists {
		validAttempts := []time.Time{}
		for _, attempt := range attempts {
			if now.Sub(attempt) < rl.window {
				validAttempts = append(validAttempts, attempt)
			}
		}
		rl.attempts[ip] = validAttempts
	}
	
	// Check if limit exceeded
	if len(rl.attempts[ip]) >= rl.limit {
		return false
	}
	
	// Record this attempt
	rl.attempts[ip] = append(rl.attempts[ip], now)
	return true
}

// getUserFromSession gets user from request context or session
func getUserFromSession(r *http.Request) *User {
	// Check context first
	if user, ok := r.Context().Value("user").(*User); ok {
		return user
	}
	
	// Fall back to session lookup
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
		Role:     session.Role,
	}
}

// CSRF token generation for forms
func generateCSRFToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		log.Printf("Error generating CSRF token: %v", err)
		return ""
	}
	return base64.URLEncoding.EncodeToString(b)
}

// Hash a string using SHA256 (for non-password use)
func hashSHA256(s string) string {
	h := sha256.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}
