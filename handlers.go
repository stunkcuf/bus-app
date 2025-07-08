package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

// getAgencyVehicles returns agency vehicles as JSON (for AJAX requests)
func getAgencyVehicles(w http.ResponseWriter, r *http.Request) {
	// Get query parameters
	month := r.URL.Query().Get("month")
	year := r.URL.Query().Get("year")
	
	if month == "" || year == "" {
		http.Error(w, "Month and year parameters required", http.StatusBadRequest)
		return
	}
	
	vehicles, err := getAgencyVehicleReports(month, year)
	if err != nil {
		log.Printf("Error getting agency vehicles: %v", err)
		http.Error(w, "Failed to get agency vehicles", http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(vehicles); err != nil {
		log.Printf("Error encoding agency vehicles: %v", err)
	}
}

// getSchoolBuses returns school buses as JSON (for AJAX requests)
func getSchoolBuses(w http.ResponseWriter, r *http.Request) {
	// Get query parameters
	month := r.URL.Query().Get("month")
	year := r.URL.Query().Get("year")
	
	if month == "" || year == "" {
		http.Error(w, "Month and year parameters required", http.StatusBadRequest)
		return
	}
	
	buses, err := getSchoolBusReports(month, year)
	if err != nil {
		log.Printf("Error getting school buses: %v", err)
		http.Error(w, "Failed to get school buses", http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(buses); err != nil {
		log.Printf("Error encoding school buses: %v", err)
	}
}

// exportMileageReport exports mileage data as CSV
func exportMileageReport(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	
	// Get parameters
	reportType := r.URL.Query().Get("type")
	month := r.URL.Query().Get("month")
	year := r.URL.Query().Get("year")
	
	if reportType == "" || month == "" || year == "" {
		http.Error(w, "Missing required parameters", http.StatusBadRequest)
		return
	}
	
	// Set CSV headers
	filename := fmt.Sprintf("%s_%s_%s.csv", reportType, month, year)
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	
	// Write CSV based on type
	switch reportType {
	case "agency":
		exportAgencyVehiclesCSV(w, month, year)
	case "school":
		exportSchoolBusesCSV(w, month, year)
	case "program":
		exportProgramStaffCSV(w, month, year)
	default:
		http.Error(w, "Invalid report type", http.StatusBadRequest)
	}
}

func exportAgencyVehiclesCSV(w http.ResponseWriter, month, year string) {
	vehicles, err := getAgencyVehicleReports(month, year)
	if err != nil {
		http.Error(w, "Failed to get data", http.StatusInternalServerError)
		return
	}
	
	// Write CSV header
	fmt.Fprintf(w, "Vehicle ID,Year,Make/Model,License Plate,Location,Beginning Miles,Ending Miles,Total Miles,Status,Notes\n")
	
	// Write data rows
	for _, v := range vehicles {
		fmt.Fprintf(w, "%s,%d,%s,%s,%s,%d,%d,%d,%s,%s\n",
			escapeCSV(v.VehicleID),
			v.VehicleYear,
			escapeCSV(v.MakeModel),
			escapeCSV(v.LicensePlate),
			escapeCSV(v.Location),
			v.BeginningMiles,
			v.EndingMiles,
			v.TotalMiles,
			escapeCSV(v.Status),
			escapeCSV(v.Notes))
	}
}

func exportSchoolBusesCSV(w http.ResponseWriter, month, year string) {
	buses, err := getSchoolBusReports(month, year)
	if err != nil {
		http.Error(w, "Failed to get data", http.StatusInternalServerError)
		return
	}
	
	// Write CSV header
	fmt.Fprintf(w, "Bus ID,Year,Make,License Plate,Location,Beginning Miles,Ending Miles,Total Miles,Status,Notes\n")
	
	// Write data rows
	for _, b := range buses {
		fmt.Fprintf(w, "%s,%d,%s,%s,%s,%d,%d,%d,%s,%s\n",
			escapeCSV(b.BusID),
			b.BusYear,
			escapeCSV(b.BusMake),
			escapeCSV(b.LicensePlate),
			escapeCSV(b.Location),
			b.BeginningMiles,
			b.EndingMiles,
			b.TotalMiles,
			escapeCSV(b.Status),
			escapeCSV(b.Notes))
	}
}

func exportProgramStaffCSV(w http.ResponseWriter, month, year string) {
	staff, err := getProgramStaffReports(month, year)
	if err != nil {
		http.Error(w, "Failed to get data", http.StatusInternalServerError)
		return
	}
	
	// Write CSV header
	fmt.Fprintf(w, "Program Type,Staff Count 1,Staff Count 2\n")
	
	// Write data rows
	for _, s := range staff {
		fmt.Fprintf(w, "%s,%d,%d\n",
			escapeCSV(s.ProgramType),
			s.StaffCount1,
			s.StaffCount2)
	}
}

// escapeCSV escapes special characters in CSV fields
func escapeCSV(s string) string {
	if strings.ContainsAny(s, ",\"\n") {
		s = strings.ReplaceAll(s, "\"", "\"\"")
		return fmt.Sprintf("\"%s\"", s)
	}
	return s
}

// studentDetailsAPI returns student details as JSON
func studentDetailsAPI(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	
	studentID := r.URL.Query().Get("id")
	if studentID == "" {
		http.Error(w, "Student ID required", http.StatusBadRequest)
		return
	}
	
	// Load students
	students, err := cache.GetStudents()
	if err != nil {
		http.Error(w, "Failed to load students", http.StatusInternalServerError)
		return
	}
	
	// Find the student
	for _, student := range students {
		if student.StudentID == studentID {
			// Check permissions
			if user.Role == "driver" && student.Driver != user.Username {
				http.Error(w, "Access denied", http.StatusForbidden)
				return
			}
			
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(student)
			return
		}
	}
	
	http.Error(w, "Student not found", http.StatusNotFound)
}

// routeDetailsAPI returns route details with all students
func routeDetailsAPI(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	
	routeID := r.URL.Query().Get("id")
	if routeID == "" {
		http.Error(w, "Route ID required", http.StatusBadRequest)
		return
	}
	
	// Get route
	routes, err := cache.GetRoutes()
	if err != nil {
		http.Error(w, "Failed to load routes", http.StatusInternalServerError)
		return
	}
	
	var route *Route
	for _, r := range routes {
		if r.RouteID == routeID {
			route = &r
			break
		}
	}
	
	if route == nil {
		http.Error(w, "Route not found", http.StatusNotFound)
		return
	}
	
	// Get students on this route
	students, err := cache.GetStudents()
	if err != nil {
		http.Error(w, "Failed to load students", http.StatusInternalServerError)
		return
	}
	
	var routeStudents []Student
	for _, s := range students {
		if s.RouteID == routeID && s.Active {
			// For drivers, only show their students
			if user.Role == "driver" && s.Driver != user.Username {
				continue
			}
			routeStudents = append(routeStudents, s)
		}
	}
	
	// Create response
	response := struct {
		Route    *Route    `json:"route"`
		Students []Student `json:"students"`
		Count    int       `json:"student_count"`
	}{
		Route:    route,
		Students: routeStudents,
		Count:    len(routeStudents),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// busStatusAPI returns current bus status
func busStatusAPI(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	
	busID := r.URL.Query().Get("id")
	if busID == "" {
		http.Error(w, "Bus ID required", http.StatusBadRequest)
		return
	}
	
	// Get bus
	buses, err := cache.GetBuses()
	if err != nil {
		http.Error(w, "Failed to load buses", http.StatusInternalServerError)
		return
	}
	
	for _, bus := range buses {
		if bus.BusID == busID {
			// Get maintenance records
			maintenanceRecords, _ := getAllVehicleMaintenanceRecords(busID)
			
			// Get current assignment
			var currentDriver string
			var currentRoute string
			assignments, _ := loadRouteAssignments()
			for _, a := range assignments {
				if a.BusID == busID {
					currentDriver = a.Driver
					currentRoute = a.RouteName
					break
				}
			}
			
			// Create response
			response := struct {
				Bus                *Bus                `json:"bus"`
				CurrentDriver      string              `json:"current_driver"`
				CurrentRoute       string              `json:"current_route"`
				MaintenanceRecords []BusMaintenanceLog `json:"maintenance_records"`
				RecentMaintenance  int                 `json:"recent_maintenance_count"`
			}{
				Bus:                bus,
				CurrentDriver:      currentDriver,
				CurrentRoute:       currentRoute,
				MaintenanceRecords: maintenanceRecords,
				RecentMaintenance:  countRecentMaintenance(maintenanceRecords, 30),
			}
			
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}
	}
	
	http.Error(w, "Bus not found", http.StatusNotFound)
}

// driverStatsAPI returns driver statistics
func driverStatsAPI(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	
	driverUsername := r.URL.Query().Get("driver")
	if driverUsername == "" {
		driverUsername = user.Username
	}
	
	// Check permissions
	if user.Role == "driver" && driverUsername != user.Username {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}
	
	// Get driver logs
	logs, err := getDriverLogs(driverUsername)
	if err != nil {
		http.Error(w, "Failed to load driver logs", http.StatusInternalServerError)
		return
	}
	
	// Calculate statistics
	stats := calculateDriverStats(logs)
	
	// Get current assignment
	assignment, _ := getDriverRouteAssignment(driverUsername)
	
	// Get student count
	students, _ := getDriverStudents(driverUsername)
	
	response := struct {
		Driver          string            `json:"driver"`
		TotalTrips      int               `json:"total_trips"`
		TotalMiles      float64           `json:"total_miles"`
		AverageMiles    float64           `json:"average_miles"`
		LastTripDate    string            `json:"last_trip_date"`
		CurrentRoute    string            `json:"current_route"`
		CurrentBus      string            `json:"current_bus"`
		StudentCount    int               `json:"student_count"`
		MonthlyStats    map[string]Stats  `json:"monthly_stats"`
	}{
		Driver:       driverUsername,
		TotalTrips:   stats.TotalTrips,
		TotalMiles:   stats.TotalMiles,
		AverageMiles: stats.AverageMiles,
		LastTripDate: stats.LastTripDate,
		StudentCount: len(students),
	}
	
	if assignment != nil {
		response.CurrentRoute = assignment.RouteName
		response.CurrentBus = assignment.BusID
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Helper structures
type Stats struct {
	Trips int     `json:"trips"`
	Miles float64 `json:"miles"`
}

type DriverStats struct {
	TotalTrips   int
	TotalMiles   float64
	AverageMiles float64
	LastTripDate string
}

// Helper functions

func countRecentMaintenance(records []BusMaintenanceLog, days int) int {
	cutoff := time.Now().AddDate(0, 0, -days).Format("2006-01-02")
	count := 0
	for _, record := range records {
		if record.Date >= cutoff {
			count++
		}
	}
	return count
}

func calculateDriverStats(logs []DriverLog) DriverStats {
	stats := DriverStats{}
	
	if len(logs) == 0 {
		return stats
	}
	
	for _, log := range logs {
		stats.TotalTrips++
		stats.TotalMiles += log.Mileage
		if log.Date > stats.LastTripDate {
			stats.LastTripDate = log.Date
		}
	}
	
	if stats.TotalTrips > 0 {
		stats.AverageMiles = stats.TotalMiles / float64(stats.TotalTrips)
	}
	
	return stats
}

// activityReportHandler shows activity reports
func activityReportHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	
	// Get date range
	startDate := r.URL.Query().Get("start")
	endDate := r.URL.Query().Get("end")
	
	if startDate == "" || endDate == "" {
		// Default to current month
		now := time.Now()
		startDate = now.AddDate(0, 0, -now.Day()+1).Format("2006-01-02")
		endDate = now.Format("2006-01-02")
	}
	
	// Load activities
	activities, err := loadActivitiesInRange(startDate, endDate)
	if err != nil {
		log.Printf("Error loading activities: %v", err)
		activities = []Activity{}
	}
	
	// Calculate totals
	totalMiles := 0.0
	totalAttendance := 0
	for _, a := range activities {
		totalMiles += a.Miles
		totalAttendance += a.Attendance
	}
	
	data := struct {
		User            *User
		Activities      []Activity
		StartDate       string
		EndDate         string
		TotalMiles      float64
		TotalAttendance int
		CSRFToken       string
	}{
		User:            user,
		Activities:      activities,
		StartDate:       startDate,
		EndDate:         endDate,
		TotalMiles:      totalMiles,
		TotalAttendance: totalAttendance,
		CSRFToken:       getCSRFToken(r),
	}
	
	renderTemplate(w, "activity_report.html", data)
}

func loadActivitiesInRange(startDate, endDate string) ([]Activity, error) {
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	
	var activities []Activity
	err := db.Select(&activities, `
		SELECT date::text, driver, trip_name, attendance, miles, notes
		FROM activities
		WHERE date >= $1 AND date <= $2
		ORDER BY date DESC, created_at DESC
	`, startDate, endDate)
	
	return activities, err
}

// systemHealthAPI returns system health metrics
func systemHealthAPI(w http.ResponseWriter, r *http.Request) {
	health := struct {
		Status       string                 `json:"status"`
		Timestamp    string                 `json:"timestamp"`
		Database     string                 `json:"database"`
		CacheStats   map[string]interface{} `json:"cache_stats"`
		SessionCount int                    `json:"active_sessions"`
		Version      string                 `json:"version"`
		Uptime       string                 `json:"uptime"`
	}{
		Status:       "healthy",
		Timestamp:    time.Now().Format(time.RFC3339),
		Database:     "connected",
		CacheStats:   cache.GetStats(),
		SessionCount: GetActiveSessionCount(),
		Version:      "2.0.0",
		Uptime:       getUptime(),
	}
	
	// Check database
	if db == nil || db.Ping() != nil {
		health.Status = "degraded"
		health.Database = "disconnected"
	}
	
	w.Header().Set("Content-Type", "application/json")
	if health.Status == "healthy" {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
	}
	
	json.NewEncoder(w).Encode(health)
}

var startTime = time.Now()

func getUptime() string {
	duration := time.Since(startTime)
	days := int(duration.Hours() / 24)
	hours := int(duration.Hours()) % 24
	minutes := int(duration.Minutes()) % 60
	
	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
	} else if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}
