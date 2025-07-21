package main

import (
	"net/http"
	"log"
)

// profileHandler handles the user profile page
func profileHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	if r.Method == "GET" {
		data := map[string]interface{}{
			"User":      user,
			"CSRFToken": getSessionCSRFToken(r),
			"Navigation": getNavigation(user, "Profile", ""),
		}
		
		// For now, use a simple profile display
		renderTemplate(w, r, "profile.html", data)
	} else if r.Method == "POST" {
		// Handle profile updates
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Failed to parse form", http.StatusBadRequest)
			return
		}

		if !validateCSRF(r) {
			http.Error(w, "Invalid CSRF token", http.StatusForbidden)
			return
		}

		// Update user profile (placeholder for now)
		log.Printf("Profile update requested by user: %s", user.Username)
		
		// Redirect back to profile with success message
		http.Redirect(w, r, "/profile?success=1", http.StatusSeeOther)
	}
}

// settingsHandler handles the settings page
func settingsHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	// Only managers can access settings
	if user.Role != "manager" {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	data := map[string]interface{}{
		"User":      user,
		"CSRFToken": getSessionCSRFToken(r),
		"Navigation": getNavigation(user, "Settings", ""),
	}
	
	renderTemplate(w, r, "settings.html", data)
}

// helpDemoHandler shows the help system demonstration
func helpDemoHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	data := map[string]interface{}{
		"User":      user,
		"CSRFToken": getSessionCSRFToken(r),
	}
	
	renderTemplate(w, r, "help_demo.html", data)
}