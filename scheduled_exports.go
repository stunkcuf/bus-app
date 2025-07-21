package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// ScheduledExport represents a scheduled export configuration
type ScheduledExport struct {
	ID         int        `json:"id" db:"id"`
	Name       string     `json:"name" db:"name"`
	ExportType string     `json:"export_type" db:"export_type"`
	Schedule   string     `json:"schedule" db:"schedule"`         // daily, weekly, monthly
	DayOfWeek  int        `json:"day_of_week" db:"day_of_week"`   // 0-6 for weekly
	DayOfMonth int        `json:"day_of_month" db:"day_of_month"` // 1-31 for monthly
	Time       string     `json:"time" db:"time"`                 // HH:MM format
	Format     string     `json:"format" db:"format"`             // xlsx, csv
	Recipients string     `json:"recipients" db:"recipients"`     // comma-separated emails
	Enabled    bool       `json:"enabled" db:"enabled"`
	LastRun    *time.Time `json:"last_run" db:"last_run"`
	NextRun    time.Time  `json:"next_run" db:"next_run"`
	CreatedBy  string     `json:"created_by" db:"created_by"`
	CreatedAt  time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at" db:"updated_at"`
}

// scheduledExportsHandler manages scheduled exports
func scheduledExportsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		// List scheduled exports
		exports, err := getScheduledExports()
		if err != nil {
			LogRequest(r).Error("Failed to get scheduled exports", err)
			SendError(w, ErrInternal("Failed to load scheduled exports", err))
			return
		}

		renderTemplate(w, r, "scheduled_exports.html", map[string]interface{}{
			"Exports": exports,
		})

	case "POST":
		// Create new scheduled export
		var export ScheduledExport

		// Parse form data
		export.Name = r.FormValue("name")
		export.ExportType = r.FormValue("export_type")
		export.Schedule = r.FormValue("schedule")
		export.Time = r.FormValue("time")
		export.Format = r.FormValue("format")
		export.Recipients = r.FormValue("recipients")
		export.Enabled = r.FormValue("enabled") == "on"

		// Parse schedule-specific fields
		if export.Schedule == "weekly" {
			fmt.Sscanf(r.FormValue("day_of_week"), "%d", &export.DayOfWeek)
		} else if export.Schedule == "monthly" {
			fmt.Sscanf(r.FormValue("day_of_month"), "%d", &export.DayOfMonth)
		}

		// Set creator
		user := getUserFromSession(r)
		if user != nil {
			export.CreatedBy = user.Username
		}

		// Calculate next run time
		export.NextRun = calculateNextRun(export)

		// Save to database
		err := createScheduledExport(&export)
		if err != nil {
			LogRequest(r).Error("Failed to create scheduled export", err)
			SendError(w, ErrInternal("Failed to create scheduled export", err))
			return
		}

		// Redirect to list
		http.Redirect(w, r, "/export/scheduled", http.StatusSeeOther)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// scheduledExportEditHandler handles editing scheduled exports
func scheduledExportEditHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		SendError(w, ErrBadRequest("Export ID required"))
		return
	}

	var id int
	fmt.Sscanf(idStr, "%d", &id)

	switch r.Method {
	case "GET":
		// Show edit form
		export, err := getScheduledExport(id)
		if err != nil {
			SendError(w, ErrNotFound("Export not found"))
			return
		}

		user := getUserFromSession(r)
		if user == nil {
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}

		data := map[string]interface{}{
			"User":      user,
			"CSRFToken": getSessionCSRFToken(r),
			"Export":    export,
			"ExportTypes": []string{"fleet", "students", "maintenance", "mileage", "ecse"},
			"Schedules":   []string{"daily", "weekly", "monthly"},
			"Formats":     []string{"xlsx", "csv", "pdf"},
		}

		renderTemplate(w, r, "scheduled_export_edit.html", data)

	case "POST":
		// Update export
		export, err := getScheduledExport(id)
		if err != nil {
			SendError(w, ErrNotFound("Export not found"))
			return
		}

		// Update fields
		export.Name = r.FormValue("name")
		export.ExportType = r.FormValue("export_type")
		export.Schedule = r.FormValue("schedule")
		export.Time = r.FormValue("time")
		export.Format = r.FormValue("format")
		export.Recipients = r.FormValue("recipients")
		export.Enabled = r.FormValue("enabled") == "on"

		// Update schedule-specific fields
		if export.Schedule == "weekly" {
			fmt.Sscanf(r.FormValue("day_of_week"), "%d", &export.DayOfWeek)
		} else if export.Schedule == "monthly" {
			fmt.Sscanf(r.FormValue("day_of_month"), "%d", &export.DayOfMonth)
		}

		// Recalculate next run time
		export.NextRun = calculateNextRun(*export)

		// Save changes
		err = updateScheduledExport(export)
		if err != nil {
			LogRequest(r).Error("Failed to update scheduled export", err)
			SendError(w, ErrInternal("Failed to update scheduled export", err))
			return
		}

		// Redirect to list
		http.Redirect(w, r, "/export/scheduled", http.StatusSeeOther)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// scheduledExportDeleteHandler handles deletion
func scheduledExportDeleteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idStr := r.FormValue("id")
	var id int
	fmt.Sscanf(idStr, "%d", &id)

	err := deleteScheduledExport(id)
	if err != nil {
		LogRequest(r).Error("Failed to delete scheduled export", err)
		SendError(w, ErrInternal("Failed to delete scheduled export", err))
		return
	}

	http.Redirect(w, r, "/export/scheduled", http.StatusSeeOther)
}

// scheduledExportRunHandler manually runs a scheduled export
func scheduledExportRunHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idStr := r.FormValue("id")
	var id int
	fmt.Sscanf(idStr, "%d", &id)

	export, err := getScheduledExport(id)
	if err != nil {
		SendError(w, ErrNotFound("Export not found"))
		return
	}

	// Run the export
	err = runScheduledExport(export)
	if err != nil {
		LogRequest(r).Error("Failed to run scheduled export", err)
		SendError(w, ErrInternal("Failed to run export", err))
		return
	}

	// Return success
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Export completed successfully",
	})
}

// Database functions

func getScheduledExports() ([]ScheduledExport, error) {
	query := `
		SELECT id, name, export_type, schedule, day_of_week, day_of_month, 
		       time, format, recipients, enabled, last_run, next_run, 
		       created_by, created_at, updated_at
		FROM scheduled_exports
		ORDER BY name
	`

	var exports []ScheduledExport
	err := db.Select(&exports, query)
	return exports, err
}

func getScheduledExport(id int) (*ScheduledExport, error) {
	var export ScheduledExport
	query := `
		SELECT id, name, export_type, schedule, day_of_week, day_of_month, 
		       time, format, recipients, enabled, last_run, next_run, 
		       created_by, created_at, updated_at
		FROM scheduled_exports
		WHERE id = $1
	`

	err := db.Get(&export, query, id)
	return &export, err
}

func createScheduledExport(export *ScheduledExport) error {
	query := `
		INSERT INTO scheduled_exports 
		(name, export_type, schedule, day_of_week, day_of_month, time, 
		 format, recipients, enabled, next_run, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		RETURNING id
	`

	return db.QueryRow(query,
		export.Name, export.ExportType, export.Schedule, export.DayOfWeek,
		export.DayOfMonth, export.Time, export.Format, export.Recipients,
		export.Enabled, export.NextRun, export.CreatedBy).Scan(&export.ID)
}

func updateScheduledExport(export *ScheduledExport) error {
	query := `
		UPDATE scheduled_exports 
		SET name = $2, export_type = $3, schedule = $4, day_of_week = $5, 
		    day_of_month = $6, time = $7, format = $8, recipients = $9, 
		    enabled = $10, next_run = $11, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`

	_, err := db.Exec(query,
		export.ID, export.Name, export.ExportType, export.Schedule,
		export.DayOfWeek, export.DayOfMonth, export.Time, export.Format,
		export.Recipients, export.Enabled, export.NextRun)
	return err
}

func deleteScheduledExport(id int) error {
	_, err := db.Exec("DELETE FROM scheduled_exports WHERE id = $1", id)
	return err
}

// calculateNextRun calculates the next run time for a scheduled export
func calculateNextRun(export ScheduledExport) time.Time {
	now := time.Now()

	// Parse the time
	timeParts := strings.Split(export.Time, ":")
	hour, minute := 0, 0
	if len(timeParts) == 2 {
		fmt.Sscanf(timeParts[0], "%d", &hour)
		fmt.Sscanf(timeParts[1], "%d", &minute)
	}

	switch export.Schedule {
	case "daily":
		// Next occurrence of the specified time
		next := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())
		if next.Before(now) {
			next = next.AddDate(0, 0, 1)
		}
		return next

	case "weekly":
		// Next occurrence of the specified day and time
		daysUntil := (export.DayOfWeek - int(now.Weekday()) + 7) % 7
		if daysUntil == 0 {
			// Today is the day, check time
			next := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())
			if next.After(now) {
				return next
			}
			daysUntil = 7
		}
		return time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location()).AddDate(0, 0, daysUntil)

	case "monthly":
		// Next occurrence of the specified day of month and time
		next := time.Date(now.Year(), now.Month(), export.DayOfMonth, hour, minute, 0, 0, now.Location())
		if next.Before(now) {
			next = next.AddDate(0, 1, 0)
		}
		return next

	default:
		return now.AddDate(0, 0, 1) // Default to tomorrow
	}
}

// runScheduledExport executes a scheduled export
func runScheduledExport(export *ScheduledExport) error {
	// Generate the export based on type
	var data []byte
	var filename string
	var err error

	switch export.ExportType {
	case "mileage":
		data, filename, err = generateMileageExport(export.Format)
	case "students":
		data, filename, err = generateStudentExport(export.Format)
	case "vehicles":
		data, filename, err = generateVehicleExport(export.Format)
	case "maintenance":
		data, filename, err = generateMaintenanceExport(export.Format)
	default:
		return fmt.Errorf("unknown export type: %s", export.ExportType)
	}

	if err != nil {
		return fmt.Errorf("failed to generate export: %v", err)
	}

	// Send email with attachment
	recipients := strings.Split(export.Recipients, ",")
	for i, r := range recipients {
		recipients[i] = strings.TrimSpace(r)
	}

	// In a real implementation, send email with attachment
	// For now, just log it
	LogInfo("Scheduled export completed: " + export.Name + " (" + filename + ") - " + fmt.Sprintf("%d bytes", len(data)))

	// Update last run time
	_, err = db.Exec("UPDATE scheduled_exports SET last_run = CURRENT_TIMESTAMP WHERE id = $1", export.ID)
	if err != nil {
		return fmt.Errorf("failed to update last run time: %v", err)
	}

	// Calculate and update next run time
	export.NextRun = calculateNextRun(*export)
	_, err = db.Exec("UPDATE scheduled_exports SET next_run = $2 WHERE id = $1", export.ID, export.NextRun)

	return err
}

// Background job to run scheduled exports
func runScheduledExportsJob() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		// Get exports due to run
		query := `
			SELECT id, name, export_type, schedule, day_of_week, day_of_month, 
			       time, format, recipients, enabled, last_run, next_run, 
			       created_by, created_at, updated_at
			FROM scheduled_exports
			WHERE enabled = true AND next_run <= CURRENT_TIMESTAMP
		`

		var exports []ScheduledExport
		err := db.Select(&exports, query)
		if err != nil {
			LogError("Failed to get due exports", err)
			continue
		}

		// Run each due export
		for _, export := range exports {
			go func(e ScheduledExport) {
				if err := runScheduledExport(&e); err != nil {
					LogError("Failed to run scheduled export", err)
				}
			}(export)
		}
	}
}
