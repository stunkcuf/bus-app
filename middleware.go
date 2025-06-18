// middleware.go - HTTP middleware functions
package main

import (
	"log"
	"net/http"
	"strings"
	"time"
)

// withRecovery wraps handlers to recover from panics and log requests
func withRecovery(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		defer func() {
			duration := time.Since(start)
			log.Printf("%s %s - %v", r.Method, r.URL.Path, duration)
			if err := recover(); err != nil {
				log.Printf("Recovered from panic in handler %s: %v", r.URL.Path, err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
			}
		}()
		h(w, r)
	}
}

// RateLimitMiddleware uses the global rate limiter from security.go
func RateLimitMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr
		if forwardedIP := r.Header.Get("X-Forwarded-For"); forwardedIP != "" {
			// Take the first IP if there are multiple
			if idx := strings.Index(forwardedIP, ","); idx != -1 {
				ip = forwardedIP[:idx]
			} else {
				ip = forwardedIP
			}
		}

		limiter := rateLimiter.GetVisitor(ip)
		if !limiter.Allow() {
			http.Error(w, "Too many requests. Please try again later.", http.StatusTooManyRequests)
			return
		}

		next(w, r)
	}
}

// requireAuth middleware checks if user is authenticated
func requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := getUserFromSession(r)
		if user == nil {
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}
		next(w, r)
	}
}

// requireRole middleware checks if user has specific role
func requireRole(role string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			user := getUserFromSession(r)
			if user == nil || user.Role != role {
				http.Error(w, "Unauthorized", http.StatusForbidden)
				return
			}
			next(w, r)
		}
	}
}

// loggingMiddleware logs all requests
func loggingMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		log.Printf("Started %s %s", r.Method, r.URL.Path)
		next(w, r)
		log.Printf("Completed %s %s in %v", r.Method, r.URL.Path, time.Since(start))
	}
}
