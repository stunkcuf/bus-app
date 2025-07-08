package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

// withRecovery middleware recovers from panics
func withRecovery(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("Panic recovered: %v", err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
			}
		}()
		next(w, r)
	}
}

// CSPMiddleware adds CSP nonce to request context
func CSPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Generate nonce
		b := make([]byte, 16)
		if _, err := rand.Read(b); err != nil {
			log.Printf("Error generating nonce: %v", err)
			next.ServeHTTP(w, r)
			return
		}
		nonce := base64.StdEncoding.EncodeToString(b)
		
		// Add to context
		ctx := context.WithValue(r.Context(), "csp-nonce", nonce)
		r = r.WithContext(ctx)
		
		// Set CSP header with nonce
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
		
		next.ServeHTTP(w, r)
	})
}

// RateLimitMiddleware limits requests per IP
func RateLimitMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get client IP
		ip := r.RemoteAddr
		if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
			ip = strings.Split(forwarded, ",")[0]
		}
		
		// Only rate limit POST requests
		if r.Method == "POST" {
			if !loginLimiter.checkRateLimit(ip) {
				http.Error(w, "Too many requests. Please try again later.", http.StatusTooManyRequests)
				return
			}
		}
		
		next(w, r)
	}
}

// requireAuth ensures user is authenticated
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
		
		// Add user info to context
		ctx := context.WithValue(r.Context(), "user", &User{
			Username: session.Username,
			Role:     session.Role,
		})
		
		next(w, r.WithContext(ctx))
	}
}

// requireRole ensures user has a specific role
func requireRole(role string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			user := getUserFromSession(r)
			if user == nil || user.Role != role {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}
			next(w, r)
		}
	}
}

// requireDatabase middleware to ensure DB connection
func requireDatabase(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if db == nil {
			http.Error(w, "Database unavailable", http.StatusServiceUnavailable)
			return
		}
		
		// Ping database to ensure connection is alive
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		
		if err := db.PingContext(ctx); err != nil {
			log.Printf("Database ping failed: %v", err)
			http.Error(w, "Database connection lost", http.StatusServiceUnavailable)
			return
		}
		
		next(w, r)
	}
}

// loggingMiddleware logs HTTP requests
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		// Wrap the ResponseWriter to capture status code
		wrapped := &responseWriter{
			ResponseWriter: w,
			statusCode:    http.StatusOK,
		}
		
		next.ServeHTTP(wrapped, r)
		
		log.Printf("%s %s %s %d %v",
			r.RemoteAddr,
			r.Method,
			r.URL.Path,
			wrapped.statusCode,
			time.Since(start))
	})
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// corsMiddleware adds CORS headers
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-CSRF-Token")
		
		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

// timeoutMiddleware adds request timeout
func timeoutMiddleware(timeout time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()
			
			r = r.WithContext(ctx)
			
			done := make(chan struct{})
			go func() {
				next.ServeHTTP(w, r)
				close(done)
			}()
			
			select {
			case <-done:
				return
			case <-ctx.Done():
				http.Error(w, "Request timeout", http.StatusRequestTimeout)
			}
		})
	}
}

// contentTypeMiddleware ensures proper content type
func contentTypeMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set default content type if not set
		if w.Header().Get("Content-Type") == "" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
		}
		next.ServeHTTP(w, r)
	})
}
