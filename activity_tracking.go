package main

import (
	"fmt"
	"time"
)

// getRecentActivity retrieves recent system activity from various sources
func getRecentActivity() []map[string]interface{} {
	activities := []map[string]interface{}{}
	
	if db == nil {
		return activities
	}
	
	// Get recent driver logs (completed routes)
	var recentLogs []struct {
		Driver   string    `db:"driver"`
		BusID    string    `db:"bus_id"`
		RouteID  string    `db:"route_id"`
		Period   string    `db:"period"`
		Date     string    `db:"date"`
		CreatedAt time.Time `db:"created_at"`
	}
	
	err := db.Select(&recentLogs, `
		SELECT driver, bus_id, route_id, period, date, created_at 
		FROM driver_logs 
		ORDER BY created_at DESC 
		LIMIT 5
	`)
	
	if err == nil {
		for _, log := range recentLogs {
			activities = append(activities, map[string]interface{}{
				"Type":    "success",
				"Icon":    "check-circle",
				"Message": fmt.Sprintf("%s completed %s route on bus %s", log.Driver, log.Period, log.BusID),
				"Time":    formatTimeAgo(log.CreatedAt),
			})
		}
	}
	
	// Get recent maintenance records
	var recentMaintenance []struct {
		VehicleNumber int       `db:"vehicle_number"`
		Category      string    `db:"category"`
		CreatedAt     time.Time `db:"created_at"`
	}
	
	err = db.Select(&recentMaintenance, `
		SELECT vehicle_number, work_description as category, created_at 
		FROM maintenance_records 
		WHERE created_at > NOW() - INTERVAL '7 days'
		ORDER BY created_at DESC 
		LIMIT 5
	`)
	
	if err == nil {
		for _, maint := range recentMaintenance {
			activities = append(activities, map[string]interface{}{
				"Type":    "info",
				"Icon":    "tools",
				"Message": fmt.Sprintf("Vehicle #%d: %s", maint.VehicleNumber, maint.Category),
				"Time":    formatTimeAgo(maint.CreatedAt),
			})
		}
	}
	
	// Get recent user registrations
	var recentUsers []struct {
		Username     string    `db:"username"`
		Status       string    `db:"status"`
		CreatedAt    time.Time `db:"created_at"`
	}
	
	err = db.Select(&recentUsers, `
		SELECT username, status, created_at 
		FROM users 
		WHERE created_at > NOW() - INTERVAL '7 days'
		ORDER BY created_at DESC 
		LIMIT 5
	`)
	
	if err == nil {
		for _, user := range recentUsers {
			icon := "person-plus"
			message := fmt.Sprintf("New driver %s registered", user.Username)
			if user.Status == "pending" {
				icon = "person-exclamation"
				message += " (pending approval)"
			}
			
			activities = append(activities, map[string]interface{}{
				"Type":    "info",
				"Icon":    icon,
				"Message": message,
				"Time":    formatTimeAgo(user.CreatedAt),
			})
		}
	}
	
	// Check for vehicles needing maintenance soon
	var maintenanceAlerts []struct {
		VehicleNumber int    `db:"vehicle_number"`
		NextService   int    `db:"next_service"`
		CurrentMiles  int    `db:"current_miles"`
	}
	
	// This is a simplified query - you might need to adjust based on your actual data structure
	err = db.Select(&maintenanceAlerts, `
		SELECT 
			mr.vehicle_number,
			MAX(mr.mileage) + 5000 as next_service,
			MAX(mr.mileage) as current_miles
		FROM maintenance_records mr
		GROUP BY mr.vehicle_number
		HAVING MAX(mr.mileage) + 5000 - MAX(mr.mileage) < 1000
		LIMIT 5
	`)
	
	if err == nil {
		for _, alert := range maintenanceAlerts {
			milesUntil := alert.NextService - alert.CurrentMiles
			activities = append(activities, map[string]interface{}{
				"Type":    "warning",
				"Icon":    "exclamation-triangle",
				"Message": fmt.Sprintf("Vehicle #%d maintenance due in %d miles", alert.VehicleNumber, milesUntil),
				"Time":    "Upcoming",
			})
		}
	}
	
	// Sort by time and limit to 10 most recent
	if len(activities) > 10 {
		activities = activities[:10]
	}
	
	// If no activities, return a default message
	if len(activities) == 0 {
		activities = append(activities, map[string]interface{}{
			"Type":    "info",
			"Icon":    "info-circle",
			"Message": "No recent activity to display",
			"Time":    "Now",
		})
	}
	
	return activities
}

// formatTimeAgo converts a time to a human-readable "ago" format
func formatTimeAgo(t time.Time) string {
	duration := time.Since(t)
	
	if duration < time.Minute {
		return "Just now"
	} else if duration < time.Hour {
		minutes := int(duration.Minutes())
		if minutes == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", minutes)
	} else if duration < 24*time.Hour {
		hours := int(duration.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	} else if duration < 7*24*time.Hour {
		days := int(duration.Hours() / 24)
		if days == 1 {
			return "Yesterday"
		}
		return fmt.Sprintf("%d days ago", days)
	} else {
		return t.Format("Jan 2, 2006")
	}
}