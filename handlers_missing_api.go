package main

import (
	"encoding/json"
	"net/http"
	"time"
)

// API endpoint for notifications list
func apiNotificationsHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get notifications for user - simplified since table might not exist
	notifications := []map[string]interface{}{
		{
		"id":         1,
		"message":    "Welcome to the Fleet Management System",
		"type":       "info",
		"is_read":    false,
		"created_at": time.Now().AddDate(0, 0, -1),
		},
		{
		"id":         2,
		"message":    "Your monthly report is ready",
		"type":       "success",
		"is_read":    true,
		"created_at": time.Now().AddDate(0, 0, -2),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"notifications": notifications,
		"count":        len(notifications),
	})
}

// API endpoint for searching students
func apiSearchStudentsHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	searchQuery := r.URL.Query().Get("q")
	if searchQuery == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
		"students": []interface{}{},
		"count":    0,
		})
		return
	}

	// Search students by name or ID
	query := `
		SELECT student_id, name, locations, phone_number, guardian, route_id
		FROM students
		WHERE LOWER(name) LIKE LOWER($1) 
		   OR LOWER(student_id) LIKE LOWER($1)
		ORDER BY name
		LIMIT 20
	`

	searchPattern := "%" + searchQuery + "%"
	rows, err := db.Query(query, searchPattern)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	students := []map[string]interface{}{}
	for rows.Next() {
		var s Student
		if err := rows.Scan(&s.StudentID, &s.Name, &s.Locations, &s.PhoneNumber, &s.Guardian, &s.RouteID); err != nil {
		continue
		}
		students = append(students, map[string]interface{}{
		"student_id": s.StudentID,
		"name":       s.Name,
		"locations":  s.Locations,
		"phone":      s.PhoneNumber,
		"guardian":   s.Guardian,
		"route_id":   s.RouteID,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"students": students,
		"count":    len(students),
		"query":    searchQuery,
	})
}

// API endpoint for fleet summary
func apiFleetSummaryHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get fleet summary statistics
	var stats struct {
		TotalBuses       int `json:"total_buses"`
		ActiveBuses      int `json:"active_buses"`
		InactiveBuses    int `json:"inactive_buses"`
		MaintenanceDue   int `json:"maintenance_due"`
		TotalRoutes      int `json:"total_routes"`
		ActiveRoutes     int `json:"active_routes"`
		TotalDrivers     int `json:"total_drivers"`
		TotalStudents    int `json:"total_students"`
	}

	// Count buses
	db.QueryRow("SELECT COUNT(*) FROM buses").Scan(&stats.TotalBuses)
	db.QueryRow("SELECT COUNT(*) FROM buses WHERE status = 'active'").Scan(&stats.ActiveBuses)
	db.QueryRow("SELECT COUNT(*) FROM buses WHERE status = 'inactive'").Scan(&stats.InactiveBuses)

	// Count maintenance due
	db.QueryRow(`
		SELECT COUNT(DISTINCT vehicle_id) 
		FROM maintenance_records 
		WHERE next_service_date <= CURRENT_DATE + INTERVAL '30 days'
	`).Scan(&stats.MaintenanceDue)

	// Count routes
	db.QueryRow("SELECT COUNT(*) FROM routes").Scan(&stats.TotalRoutes)
	db.QueryRow("SELECT COUNT(*) FROM routes WHERE active = true").Scan(&stats.ActiveRoutes)

	// Count drivers
	db.QueryRow("SELECT COUNT(*) FROM users WHERE role = 'driver'").Scan(&stats.TotalDrivers)

	// Count students
	db.QueryRow("SELECT COUNT(*) FROM students").Scan(&stats.TotalStudents)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// Missing dashboard handlers
func budgetDashboardPageHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// Get budget data
	var budgetData struct {
		TotalBudget     float64
		SpentAmount     float64
		RemainingAmount float64
		FuelCosts       float64
		MaintenanceCosts float64
		OtherCosts      float64
	}

	// Calculate costs from maintenance records
	db.QueryRow(`
		SELECT COALESCE(SUM(cost), 0) 
		FROM maintenance_records 
		WHERE EXTRACT(YEAR FROM service_date) = EXTRACT(YEAR FROM CURRENT_DATE)
	`).Scan(&budgetData.MaintenanceCosts)

	// Calculate fuel costs
	db.QueryRow(`
		SELECT COALESCE(SUM(cost), 0) 
		FROM fuel_records 
		WHERE EXTRACT(YEAR FROM date) = EXTRACT(YEAR FROM CURRENT_DATE)
	`).Scan(&budgetData.FuelCosts)

	budgetData.TotalBudget = 500000 // Example budget
	budgetData.SpentAmount = budgetData.FuelCosts + budgetData.MaintenanceCosts
	budgetData.RemainingAmount = budgetData.TotalBudget - budgetData.SpentAmount

	renderTemplate(w, r, "budget_dashboard.html", map[string]interface{}{
		"User":   user,
		"Title":  "Budget Dashboard",
		"Budget": budgetData,
	})
}

func progressDashboardPageHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// Get progress metrics
	var metrics struct {
		RoutesCompleted   int
		StudentsPickedUp  int
		MilesDrivern      float64
		OnTimePercentage  float64
	}

	// Get today's completed routes
	db.QueryRow(`
		SELECT COUNT(*) 
		FROM driver_logs 
		WHERE DATE(created_at) = CURRENT_DATE 
		AND status = 'completed'
	`).Scan(&metrics.RoutesCompleted)

	// Get today's mileage
	db.QueryRow(`
		SELECT COALESCE(SUM(end_mileage - start_mileage), 0)
		FROM driver_logs
		WHERE DATE(created_at) = CURRENT_DATE
	`).Scan(&metrics.MilesDrivern)

	metrics.OnTimePercentage = 95.5 // Example
	metrics.StudentsPickedUp = 450  // Example

	renderTemplate(w, r, "progress_dashboard.html", map[string]interface{}{
		"User":    user,
		"Title":   "Progress Dashboard",
		"Metrics": metrics,
	})
}