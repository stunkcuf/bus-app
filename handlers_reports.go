package main

import (
	"net/http"
	"log"
)

// reportsHandler shows reports page for both drivers and managers
func reportsHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	// Get driver's recent logs if they're a driver
	var driverLogs []DriverLog
	if user.Role == "driver" {
		if db != nil {
			query := `
				SELECT driver, bus_id, route_id, date, period, departure_time, arrival_time, 
				       begin_mileage, end_mileage, attendance, created_at
				FROM driver_logs
				WHERE driver = $1
				ORDER BY date DESC, created_at DESC
				LIMIT 20
			`
			if err := db.Select(&driverLogs, query, user.Username); err != nil {
				log.Printf("Error loading driver logs: %v", err)
			}
		}
	}

	// Get statistics for managers
	var totalBuses, totalDrivers, totalRoutes, totalStudents int
	if user.Role == "manager" {
		// Get counts from cache
		buses, _ := dataCache.getBuses()
		totalBuses = len(buses)
		
		routes, _ := dataCache.getRoutes()
		totalRoutes = len(routes)
		
		users, _ := dataCache.getUsers()
		for _, u := range users {
			if u.Role == "driver" && u.Status == "active" {
				totalDrivers++
			}
		}
		
		students, _ := loadStudentsFromDB()
		totalStudents = len(students)
	}

	data := map[string]interface{}{
		"User":          user,
		"Title":         "Reports",
		"CSRFToken":     getSessionCSRFToken(r),
		"DriverLogs":    driverLogs,
		"TotalBuses":    totalBuses,
		"TotalDrivers":  totalDrivers,
		"TotalRoutes":   totalRoutes,
		"TotalStudents": totalStudents,
	}

	// Use different template based on role
	if user.Role == "driver" {
		renderTemplate(w, r, "driver_reports.html", data)
	} else {
		renderTemplate(w, r, "manager_reports.html", data)
	}
}

// mileageReportGeneratorHandler shows the mileage report generator page
func mileageReportGeneratorHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	data := map[string]interface{}{
		"User":      user,
		"Title":     "Mileage Report Generator",
		"CSRFToken": getSessionCSRFToken(r),
	}

	renderTemplate(w, r, "mileage-report-generator.html", data)
}

// importDataWizardHandler shows the import data wizard
func importDataWizardHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	data := map[string]interface{}{
		"User":      user,
		"Title":     "Import Data Wizard",
		"CSRFToken": getSessionCSRFToken(r),
	}

	renderTemplate(w, r, "import_data_wizard.html", data)
}

// importAnalyzeHandler analyzes import file - delegates to enhanced version
func importAnalyzeHandler(w http.ResponseWriter, r *http.Request) {
	enhancedImportAnalyzeHandler(w, r)
}