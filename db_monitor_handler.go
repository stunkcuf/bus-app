package main

import (
	"net/http"
)

// dbMonitorHandler serves the database connection pool monitoring page
func dbMonitorHandler(w http.ResponseWriter, r *http.Request) {
	// Check if user is authenticated and is a manager
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	
	data := map[string]interface{}{
		"User":      user,
		"CSRFToken": getSessionCSRFToken(r),
	}
	
	renderTemplate(w, r, "db_monitor.html", data)
}