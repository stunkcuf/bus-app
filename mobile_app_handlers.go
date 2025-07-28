package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Enhanced attendance management with bulk operations
func (api *MobileAPI) GetStudentListHandler(w http.ResponseWriter, r *http.Request) {
	username := api.getUserFromToken(r)
	if username == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get route ID for the driver
	var routeID string
	err := api.db.QueryRow(`
		SELECT r.route_id 
		FROM route_assignments ra
		JOIN routes r ON ra.route_id = r.route_id
		WHERE ra.driver_username = $1 AND ra.status = 'active'
		LIMIT 1
	`, username).Scan(&routeID)

	if err != nil {
		http.Error(w, "No active route assigned", http.StatusNotFound)
		return
	}

	// Get students on the route with today's attendance
	rows, err := api.db.Query(`
		SELECT 
			s.student_id,
			s.name as student_name,
			COALESCE(s.grade, '') as grade,
			COALESCE(s.locations::text, '') as address,
			COALESCE(s.guardian, '') as parent_name,
			COALESCE(s.phone_number, '') as parent_phone,
			sa.status as attendance_status,
			sa.boarded_at,
			sa.dropped_at,
			sa.notes
		FROM students s
		LEFT JOIN student_attendance sa ON s.student_id = sa.student_id 
			AND sa.attendance_date = CURRENT_DATE
		WHERE s.route_id = $1
		ORDER BY s.name
	`, routeID)

	if err != nil {
		log.Printf("Failed to get student list: %v", err)
		http.Error(w, "Failed to retrieve students", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var students []map[string]interface{}
	for rows.Next() {
		var student struct {
			ID               string         `db:"student_id"`
			Name             string         `db:"student_name"`
			Grade            string         `db:"grade"`
			Address          string         `db:"address"`
			ParentName       string         `db:"parent_name"`
			ParentPhone      string         `db:"parent_phone"`
			AttendanceStatus sql.NullString `db:"attendance_status"`
			BoardedAt        sql.NullTime   `db:"boarded_at"`
			DroppedAt        sql.NullTime   `db:"dropped_at"`
			Notes            sql.NullString `db:"notes"`
		}

		if err := rows.Scan(&student.ID, &student.Name, &student.Grade,
			&student.Address, &student.ParentName, &student.ParentPhone,
			&student.AttendanceStatus, &student.BoardedAt, &student.DroppedAt,
			&student.Notes); err != nil {
			continue
		}

		studentMap := map[string]interface{}{
			"student_id":    student.ID,
			"name":         student.Name,
			"grade":        student.Grade,
			"address":      student.Address,
			"parent_name":  student.ParentName,
			"parent_phone": student.ParentPhone,
			"attendance": map[string]interface{}{
				"status":     student.AttendanceStatus.String,
				"boarded_at": nil,
				"dropped_at": nil,
				"notes":      student.Notes.String,
			},
		}

		if student.BoardedAt.Valid {
			studentMap["attendance"].(map[string]interface{})["boarded_at"] = student.BoardedAt.Time.Format("15:04")
		}
		if student.DroppedAt.Valid {
			studentMap["attendance"].(map[string]interface{})["dropped_at"] = student.DroppedAt.Time.Format("15:04")
		}

		students = append(students, studentMap)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"route_id": routeID,
		"date":     time.Now().Format("2006-01-02"),
		"students": students,
	})
}

// Get attendance history
func (api *MobileAPI) GetAttendanceHistoryHandler(w http.ResponseWriter, r *http.Request) {
	username := api.getUserFromToken(r)
	if username == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get query parameters
	studentID := r.URL.Query().Get("student_id")
	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")

	// Default to last 30 days if no dates provided
	if startDate == "" {
		startDate = time.Now().AddDate(0, 0, -30).Format("2006-01-02")
	}
	if endDate == "" {
		endDate = time.Now().Format("2006-01-02")
	}

	query := `
		SELECT 
			attendance_date,
			status,
			boarded_at,
			dropped_at,
			notes,
			recorded_by,
			created_at
		FROM student_attendance
		WHERE student_id = $1 
		AND attendance_date BETWEEN $2 AND $3
		ORDER BY attendance_date DESC
	`

	rows, err := api.db.Query(query, studentID, startDate, endDate)
	if err != nil {
		http.Error(w, "Failed to retrieve attendance history", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var history []map[string]interface{}
	for rows.Next() {
		var record struct {
			Date       time.Time      `db:"attendance_date"`
			Status     string         `db:"status"`
			BoardedAt  sql.NullTime   `db:"boarded_at"`
			DroppedAt  sql.NullTime   `db:"dropped_at"`
			Notes      sql.NullString `db:"notes"`
			RecordedBy string         `db:"recorded_by"`
			CreatedAt  time.Time      `db:"created_at"`
		}

		if err := rows.Scan(&record.Date, &record.Status, &record.BoardedAt,
			&record.DroppedAt, &record.Notes, &record.RecordedBy,
			&record.CreatedAt); err != nil {
			continue
		}

		entry := map[string]interface{}{
			"date":        record.Date.Format("2006-01-02"),
			"status":      record.Status,
			"notes":       record.Notes.String,
			"recorded_by": record.RecordedBy,
			"created_at":  record.CreatedAt.Format(time.RFC3339),
		}

		if record.BoardedAt.Valid {
			entry["boarded_at"] = record.BoardedAt.Time.Format("15:04")
		}
		if record.DroppedAt.Valid {
			entry["dropped_at"] = record.DroppedAt.Time.Format("15:04")
		}

		history = append(history, entry)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"student_id": studentID,
		"start_date": startDate,
		"end_date":   endDate,
		"history":    history,
	})
}

// Enhanced issue reporting with photo upload
func (api *MobileAPI) UploadIssuePhotoHandler(w http.ResponseWriter, r *http.Request) {
	username := api.getUserFromToken(r)
	if username == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse multipart form (10MB max)
	r.ParseMultipartForm(10 << 20)

	file, handler, err := r.FormFile("photo")
	if err != nil {
		http.Error(w, "Failed to get file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Create uploads directory if it doesn't exist
	uploadDir := filepath.Join("static", "uploads", "issues", time.Now().Format("2006-01"))
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		http.Error(w, "Failed to create upload directory", http.StatusInternalServerError)
		return
	}

	// Generate unique filename
	filename := fmt.Sprintf("%s_%d_%s", username, time.Now().Unix(), handler.Filename)
	filepath := filepath.Join(uploadDir, filename)

	// Create the file
	dst, err := os.Create(filepath)
	if err != nil {
		http.Error(w, "Failed to create file", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	// Copy the uploaded file to the destination
	if _, err := io.Copy(dst, file); err != nil {
		http.Error(w, "Failed to save file", http.StatusInternalServerError)
		return
	}

	// Return the file path
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "success",
		"file_path": strings.ReplaceAll(filepath, "\\", "/"),
		"file_size": handler.Size,
	})
}

// Get issue reports for driver
func (api *MobileAPI) GetIssueReportsHandler(w http.ResponseWriter, r *http.Request) {
	username := api.getUserFromToken(r)
	if username == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get query parameters
	status := r.URL.Query().Get("status")
	limit := r.URL.Query().Get("limit")
	if limit == "" {
		limit = "50"
	}

	query := `
		SELECT 
			issue_id,
			type,
			description,
			vehicle_id,
			route_id,
			severity,
			status,
			latitude,
			longitude,
			created_at,
			resolved_at,
			resolution_notes
		FROM issue_reports
		WHERE reported_by = $1
	`

	args := []interface{}{username}
	if status != "" {
		query += " AND status = $2"
		args = append(args, status)
	}

	query += " ORDER BY created_at DESC LIMIT " + limit

	rows, err := api.db.Query(query, args...)
	if err != nil {
		http.Error(w, "Failed to retrieve issues", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var issues []map[string]interface{}
	for rows.Next() {
		var issue struct {
			ID               int            `db:"issue_id"`
			Type            string         `db:"type"`
			Description     string         `db:"description"`
			VehicleID       sql.NullString `db:"vehicle_id"`
			RouteID         sql.NullString `db:"route_id"`
			Severity        string         `db:"severity"`
			Status          string         `db:"status"`
			Latitude        sql.NullFloat64 `db:"latitude"`
			Longitude       sql.NullFloat64 `db:"longitude"`
			CreatedAt       time.Time      `db:"created_at"`
			ResolvedAt      sql.NullTime   `db:"resolved_at"`
			ResolutionNotes sql.NullString `db:"resolution_notes"`
		}

		if err := rows.Scan(&issue.ID, &issue.Type, &issue.Description,
			&issue.VehicleID, &issue.RouteID, &issue.Severity, &issue.Status,
			&issue.Latitude, &issue.Longitude, &issue.CreatedAt,
			&issue.ResolvedAt, &issue.ResolutionNotes); err != nil {
			continue
		}

		issueMap := map[string]interface{}{
			"issue_id":     issue.ID,
			"type":         issue.Type,
			"description":  issue.Description,
			"vehicle_id":   issue.VehicleID.String,
			"route_id":     issue.RouteID.String,
			"severity":     issue.Severity,
			"status":       issue.Status,
			"created_at":   issue.CreatedAt.Format(time.RFC3339),
		}

		if issue.Latitude.Valid && issue.Longitude.Valid {
			issueMap["location"] = map[string]float64{
				"latitude":  issue.Latitude.Float64,
				"longitude": issue.Longitude.Float64,
			}
		}

		if issue.ResolvedAt.Valid {
			issueMap["resolved_at"] = issue.ResolvedAt.Time.Format(time.RFC3339)
			issueMap["resolution_notes"] = issue.ResolutionNotes.String
		}

		// Get attachments
		var attachments []string
		attRows, _ := api.db.Query(`
			SELECT file_path FROM issue_attachments 
			WHERE issue_id = $1
		`, issue.ID)
		if attRows != nil {
			defer attRows.Close()
			for attRows.Next() {
				var path string
				if attRows.Scan(&path) == nil {
					attachments = append(attachments, path)
				}
			}
		}
		issueMap["attachments"] = attachments

		issues = append(issues, issueMap)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"issues": issues,
		"count":  len(issues),
	})
}

// Update issue status (for managers via mobile)
func (api *MobileAPI) UpdateIssueStatusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "PUT" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	username := api.getUserFromToken(r)
	if username == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Check if user is a manager
	var role string
	err := api.db.QueryRow("SELECT role FROM users WHERE username = $1", username).Scan(&role)
	if err != nil || role != "manager" {
		http.Error(w, "Manager access required", http.StatusForbidden)
		return
	}

	// Get issue ID from URL
	issueIDStr := r.URL.Query().Get("issue_id")
	issueID, err := strconv.Atoi(issueIDStr)
	if err != nil {
		http.Error(w, "Invalid issue ID", http.StatusBadRequest)
		return
	}

	var update struct {
		Status          string `json:"status"`
		ResolutionNotes string `json:"resolution_notes"`
	}

	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Update issue
	query := `
		UPDATE issue_reports 
		SET status = $1, 
			resolution_notes = $2,
			resolved_by = $3,
			resolved_at = CASE WHEN $1 = 'resolved' THEN CURRENT_TIMESTAMP ELSE resolved_at END
		WHERE issue_id = $4
	`

	_, err = api.db.Exec(query, update.Status, update.ResolutionNotes, username, issueID)
	if err != nil {
		http.Error(w, "Failed to update issue", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"message": "Issue updated successfully",
	})
}

// Get dashboard statistics for mobile app
func (api *MobileAPI) GetMobileDashboardHandler(w http.ResponseWriter, r *http.Request) {
	username := api.getUserFromToken(r)
	if username == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get user role
	var role string
	err := api.db.QueryRow("SELECT role FROM users WHERE username = $1", username).Scan(&role)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	stats := make(map[string]interface{})

	if role == "driver" {
		// Get driver-specific stats
		var routeStats struct {
			RouteID      sql.NullString `db:"route_id"`
			RouteName    sql.NullString `db:"route_name"`
			StudentCount int            `db:"student_count"`
			VehicleID    sql.NullString `db:"vehicle_id"`
		}

		err := api.db.QueryRow(`
			SELECT 
				r.route_id,
				r.route_name,
				COUNT(DISTINCT s.student_id) as student_count,
				ra.bus_number as vehicle_id
			FROM route_assignments ra
			JOIN routes r ON ra.route_id = r.route_id
			LEFT JOIN students s ON s.route_id = r.route_id
			WHERE ra.driver_username = $1 AND ra.status = 'active'
			GROUP BY r.route_id, r.route_name, ra.bus_number
			LIMIT 1
		`, username).Scan(&routeStats.RouteID, &routeStats.RouteName, 
			&routeStats.StudentCount, &routeStats.VehicleID)

		if err == nil {
			stats["current_route"] = map[string]interface{}{
				"route_id":      routeStats.RouteID.String,
				"route_name":    routeStats.RouteName.String,
				"student_count": routeStats.StudentCount,
				"vehicle_id":    routeStats.VehicleID.String,
			}
		}

		// Get today's attendance stats
		var attendanceStats struct {
			Present int `db:"present"`
			Absent  int `db:"absent"`
			Late    int `db:"late"`
		}

		api.db.QueryRow(`
			SELECT 
				COUNT(CASE WHEN sa.status = 'present' THEN 1 END) as present,
				COUNT(CASE WHEN sa.status = 'absent' THEN 1 END) as absent,
				COUNT(CASE WHEN sa.status = 'late' THEN 1 END) as late
			FROM students s
			JOIN routes r ON s.route_id = r.route_id
			JOIN route_assignments ra ON r.route_id = ra.route_id
			LEFT JOIN student_attendance sa ON s.student_id = sa.student_id 
				AND sa.attendance_date = CURRENT_DATE
			WHERE ra.driver_username = $1 AND ra.status = 'active'
		`, username).Scan(&attendanceStats.Present, &attendanceStats.Absent, &attendanceStats.Late)

		stats["today_attendance"] = attendanceStats

		// Get recent issues count
		var issueCount int
		api.db.QueryRow(`
			SELECT COUNT(*) FROM issue_reports 
			WHERE reported_by = $1 AND status IN ('open', 'in_progress')
		`, username).Scan(&issueCount)

		stats["open_issues"] = issueCount

	} else if role == "manager" {
		// Get manager-specific stats
		var fleetStats struct {
			TotalVehicles   int `db:"total_vehicles"`
			ActiveVehicles  int `db:"active_vehicles"`
			TotalDrivers    int `db:"total_drivers"`
			TotalStudents   int `db:"total_students"`
			OpenIssues      int `db:"open_issues"`
			TodayAttendance float64 `db:"attendance_rate"`
		}

		err := api.db.QueryRow(`
			SELECT 
				(SELECT COUNT(*) FROM vehicles) as total_vehicles,
				(SELECT COUNT(*) FROM vehicles WHERE status = 'active') as active_vehicles,
				(SELECT COUNT(*) FROM users WHERE role = 'driver' AND status = 'active') as total_drivers,
				(SELECT COUNT(*) FROM students WHERE status = 'active') as total_students,
				(SELECT COUNT(*) FROM issue_reports WHERE status IN ('open', 'in_progress')) as open_issues,
				(SELECT 
					CASE WHEN COUNT(*) > 0 
					THEN CAST(COUNT(CASE WHEN status = 'present' THEN 1 END) AS FLOAT) / COUNT(*) * 100
					ELSE 0 END
				FROM student_attendance WHERE attendance_date = CURRENT_DATE) as attendance_rate
		`).Scan(&fleetStats.TotalVehicles, &fleetStats.ActiveVehicles,
			&fleetStats.TotalDrivers, &fleetStats.TotalStudents,
			&fleetStats.OpenIssues, &fleetStats.TodayAttendance)

		if err == nil {
			stats["fleet_overview"] = fleetStats
		}

		// Get issue breakdown by severity
		rows, _ := api.db.Query(`
			SELECT severity, COUNT(*) as count
			FROM issue_reports
			WHERE status IN ('open', 'in_progress')
			GROUP BY severity
		`)
		if rows != nil {
			defer rows.Close()
			severityBreakdown := make(map[string]int)
			for rows.Next() {
				var severity string
				var count int
				if rows.Scan(&severity, &count) == nil {
					severityBreakdown[severity] = count
				}
			}
			stats["issues_by_severity"] = severityBreakdown
		}
	}

	// Get recent notifications (for both roles)
	stats["recent_alerts"] = getRecentAlerts(username)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// Helper function to get recent alerts
func getRecentAlerts(username string) []map[string]interface{} {
	// This would integrate with your notification system
	// For now, return empty array
	return []map[string]interface{}{}
}

// Register enhanced mobile API routes
func RegisterEnhancedMobileAPIRoutes(mux *http.ServeMux, api *MobileAPI) {
	// Attendance endpoints
	mux.HandleFunc("/api/mobile/v1/attendance/students", api.GetStudentListHandler)
	mux.HandleFunc("/api/mobile/v1/attendance/history", api.GetAttendanceHistoryHandler)
	
	// Issue reporting endpoints
	mux.HandleFunc("/api/mobile/v1/issues/upload", api.UploadIssuePhotoHandler)
	mux.HandleFunc("/api/mobile/v1/issues/list", api.GetIssueReportsHandler)
	mux.HandleFunc("/api/mobile/v1/issues/update", api.UpdateIssueStatusHandler)
	
	// Dashboard endpoint
	mux.HandleFunc("/api/mobile/v1/dashboard", api.GetMobileDashboardHandler)
}