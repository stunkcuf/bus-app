package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"runtime/debug"
)

// withRecovery wraps a handler to recover from panics
func withRecovery(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("Panic recovered: %v\n%s", err, debug.Stack())
				SendError(w, ErrInternal("An unexpected error occurred", nil))
			}
		}()
		h(w, r)
	}
}

// requireAuth ensures the user is authenticated
func requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(SessionCookieName)
		if err != nil {
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}

		session, err := GetSecureSession(cookie.Value)
		if err != nil {
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}

		// Check if user exists in database
		var exists bool
		err = db.Get(&exists, "SELECT EXISTS(SELECT 1 FROM users WHERE username = $1)", session.Username)
		if err != nil || !exists {
			deleteSession(cookie.Value)
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}

		next(w, r)
	}
}

// requireRole ensures the user has the required role
func requireRole(role string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			user := getUserFromSession(r)
			if user == nil || user.Role != role {
				SendError(w, ErrUnauthorized("Access denied"))
				return
			}
			next(w, r)
		}
	}
}

// requireDatabase ensures database is connected
func requireDatabase(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if db == nil {
			SendError(w, ErrDatabase("Database not available", nil))
			return
		}

		// Test database connection
		if err := db.Ping(); err != nil {
			log.Printf("Database ping failed: %v", err)
			SendError(w, ErrDatabase("Database connection lost", nil))
			return
		}

		next(w, r)
	}
}

// RateLimitMiddleware applies rate limiting
func RateLimitMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip := getClientIP(r)
		if !rateLimiter.Allow(ip) {
			SendError(w, ErrRateLimit("Too many requests"))
			return
		}
		next(w, r)
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

		// HSTS (only on HTTPS)
		if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		next.ServeHTTP(w, r)
	})
}

// CSPMiddleware adds Content Security Policy headers
func CSPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if CSP header is already set by a handler
		if w.Header().Get("Content-Security-Policy") != "" {
			// CSP already set, don't override
			next.ServeHTTP(w, r)
			return
		}

		// Generate nonce for inline scripts
		nonce := generateNonce()

		// Set CSP header WITHOUT 'unsafe-inline' for scripts
		// Using nonce makes 'unsafe-inline' ignored anyway, but it's cleaner to remove it
		csp := fmt.Sprintf(
			"default-src 'self'; "+
				"script-src 'self' 'nonce-%s' https://cdnjs.cloudflare.com https://cdn.jsdelivr.net; "+
				"style-src 'self' 'unsafe-inline' 'nonce-%s' https://cdnjs.cloudflare.com https://cdn.jsdelivr.net; "+
				"img-src 'self' data: https:; "+
				"font-src 'self' https://cdnjs.cloudflare.com https://cdn.jsdelivr.net; "+
				"connect-src 'self'; "+
				"frame-ancestors 'none'; "+
				"base-uri 'self'; "+
				"form-action 'self'",
			nonce, nonce,
		)
		w.Header().Set("Content-Security-Policy", csp)

		// Store nonce in request context for template use
		ctx := r.Context()
		ctx = setCSPNonce(ctx, nonce)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// generateNonce generates a random nonce for CSP
func generateNonce() string {
	b := make([]byte, 16)
	rand.Read(b)
	return base64.StdEncoding.EncodeToString(b)
}

// GenerateCSPNonce generates a CSP nonce for external use
func GenerateCSPNonce() string {
	return generateNonce()
}

// Context keys
type contextKey string

const cspNonceKey contextKey = "csp-nonce"

// setCSPNonce stores the CSP nonce in context
func setCSPNonce(ctx context.Context, nonce string) context.Context {
	return context.WithValue(ctx, cspNonceKey, nonce)
}

// getCSPNonce retrieves the CSP nonce from context
func getCSPNonce(ctx context.Context) string {
	if nonce, ok := ctx.Value(cspNonceKey).(string); ok {
		return nonce
	}
	return ""
}

// MetricsHandler wraps MetricsMiddleware to work with http.Handler
func MetricsHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Use MetricsMiddleware by wrapping the ServeHTTP method
		MetricsMiddleware(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		})(w, r)
	})
}
