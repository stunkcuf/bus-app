// middleware.go - HTTP middleware functions
package main

import (
	"log"
	"net/http"
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

// Note: getUserFromSession is already in utils.go, so we don't need it here

// You can add more middleware functions here as you develop them:

// Example: requireAuth middleware (when you're ready to use it)
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

// Example: requireRole middleware
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

// Example: logging middleware
func loggingMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		log.Printf("Started %s %s", r.Method, r.URL.Path)
		next(w, r)
		log.Printf("Completed %s %s in %v", r.Method, r.URL.Path, time.Since(start))
	}
}
