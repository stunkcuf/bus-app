package main

import (
	"net/http"
)

// V1 API Handlers
// These handlers are specifically for API version 1

// healthV1Handler handles v1 health check
func healthV1Handler(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"status": "healthy",
		"service": "Fleet Management System",
		"database": "connected",
	}
	
	sendVersionedAPIResponse(w, r, data, "Service is healthy")
}

// dashboardStatsV1Handler handles v1 dashboard stats
func dashboardStatsV1Handler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil {
		sendVersionedAPIError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	stats := make(map[string]interface{})
	
	// Get counts from database
	var busCount, activeDrivers, totalRoutes, totalStudents int
	db.Get(&busCount, "SELECT COUNT(*) FROM buses WHERE status = 'active'")
	db.Get(&activeDrivers, "SELECT COUNT(DISTINCT driver) FROM route_assignments")
	db.Get(&totalRoutes, "SELECT COUNT(*) FROM routes")
	db.Get(&totalStudents, "SELECT COUNT(*) FROM students WHERE active = true")
	
	stats["activeBuses"] = busCount
	stats["activeDrivers"] = activeDrivers
	stats["totalRoutes"] = totalRoutes
	stats["totalStudents"] = totalStudents
	
	// Add version-specific data for v1
	stats["apiVersion"] = "1.0"
	stats["features"] = []string{"basic_stats", "dashboard_overview"}
	
	sendVersionedAPIResponse(w, r, stats, "Dashboard statistics retrieved successfully")
}