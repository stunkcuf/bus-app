package main

import (
	"database/sql"
	"encoding/json"
	"html/template"
	"net/http"
	"strings"
	"time"
)

// UserProgress represents a user's progress through system features
type UserProgress struct {
	UserID           int       `json:"user_id"`
	Feature          string    `json:"feature"`
	FirstAccessed    time.Time `json:"first_accessed"`
	LastAccessed     time.Time `json:"last_accessed"`
	CompletionStatus string    `json:"completion_status"` // "not_started", "in_progress", "completed"
	AccessCount      int       `json:"access_count"`
}

// ProgressMilestone represents a milestone in user onboarding
type ProgressMilestone struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Category    string    `json:"category"`
	Required    bool      `json:"required"`
	Order       int       `json:"order"`
	Completed   bool      `json:"completed"`
	CompletedAt time.Time `json:"completed_at,omitempty"`
}

// Track user progress for a feature
func trackUserProgress(db *sql.DB, userID int, feature string) error {
	// Check if progress record exists
	var exists bool
	err := db.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM user_progress 
			WHERE user_id = $1 AND feature = $2
		)`, userID, feature).Scan(&exists)
	if err != nil {
		return err
	}

	if exists {
		// Update existing record
		_, err = db.Exec(`
			UPDATE user_progress 
			SET last_accessed = NOW(), 
			    access_count = access_count + 1
			WHERE user_id = $1 AND feature = $2`,
			userID, feature)
	} else {
		// Create new record
		_, err = db.Exec(`
			INSERT INTO user_progress (user_id, feature, first_accessed, last_accessed, completion_status, access_count)
			VALUES ($1, $2, NOW(), NOW(), 'in_progress', 1)`,
			userID, feature)
	}

	return err
}

// Get user's overall progress
func getUserProgress(db *sql.DB, userID int, userType string) (map[string]interface{}, error) {
	// Get feature access data
	rows, err := db.Query(`
		SELECT feature, first_accessed, last_accessed, completion_status, access_count
		FROM user_progress
		WHERE user_id = $1
		ORDER BY last_accessed DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var features []UserProgress
	for rows.Next() {
		var up UserProgress
		up.UserID = userID
		err := rows.Scan(&up.Feature, &up.FirstAccessed, &up.LastAccessed, &up.CompletionStatus, &up.AccessCount)
		if err != nil {
			continue
		}
		features = append(features, up)
	}

	// Get milestones based on user type
	milestones := getMilestonesForRole(userType)
	
	// Mark completed milestones
	for i, milestone := range milestones {
		for _, feature := range features {
			if milestone.ID == feature.Feature && feature.CompletionStatus == "completed" {
				milestones[i].Completed = true
				milestones[i].CompletedAt = feature.LastAccessed
			}
		}
	}

	// Calculate overall progress
	totalMilestones := len(milestones)
	completedMilestones := 0
	requiredTotal := 0
	requiredCompleted := 0

	for _, m := range milestones {
		if m.Completed {
			completedMilestones++
			if m.Required {
				requiredCompleted++
			}
		}
		if m.Required {
			requiredTotal++
		}
	}

	progressPercent := 0
	if totalMilestones > 0 {
		progressPercent = (completedMilestones * 100) / totalMilestones
	}

	return map[string]interface{}{
		"features":                features,
		"milestones":             milestones,
		"total_milestones":       totalMilestones,
		"completed_milestones":   completedMilestones,
		"progress_percent":       progressPercent,
		"required_total":         requiredTotal,
		"required_completed":     requiredCompleted,
		"onboarding_complete":    requiredCompleted >= requiredTotal,
		"last_activity":          getLastActivity(features),
		"days_since_start":       getDaysSinceStart(features),
		"most_used_features":     getMostUsedFeatures(features, 5),
		"recommended_next_steps": getRecommendedNextSteps(milestones, userType),
	}, nil
}

// API endpoint for progress tracking
func progressTrackingHandler(w http.ResponseWriter, r *http.Request) {
	session := getUserFromSession(r)
	if session == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	switch r.Method {
	case "GET":
		// Get user's progress
		if db == nil {
			http.Error(w, "Database connection unavailable", http.StatusInternalServerError)
			return
		}

		progress, err := getUserProgress(db.DB, getUserID(session.Username), session.Role)
		if err != nil {
			// log.Printf("Error getting user progress: %v", err)
			http.Error(w, "Failed to get progress", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(progress)

	case "POST":
		// Track feature usage
		var data struct {
			Feature string `json:"feature"`
			Status  string `json:"status,omitempty"`
		}

		if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		if db == nil {
			http.Error(w, "Database connection unavailable", http.StatusInternalServerError)
			return
		}

		// Track the feature access
		err := trackUserProgress(db.DB, getUserID(session.Username), data.Feature)
		if err != nil {
			// log.Printf("Error tracking progress: %v", err)
			http.Error(w, "Failed to track progress", http.StatusInternalServerError)
			return
		}

		// Update completion status if provided
		if data.Status != "" {
			_, err := db.Exec(`
				UPDATE user_progress 
				SET completion_status = $1
				WHERE user_id = $2 AND feature = $3`,
				data.Status, getUserID(session.Username), data.Feature)
			if err != nil {
				// log.Printf("Error updating status: %v", err)
			}
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "tracked"})
	}
}

// Progress dashboard handler
func progressDashboardHandler(w http.ResponseWriter, r *http.Request) {
	session := getUserFromSession(r)
	if session == nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	if db == nil {
		http.Error(w, "Database connection unavailable", http.StatusInternalServerError)
		return
	}

	progress, err := getUserProgress(db.DB, getUserID(session.Username), session.Role)
	if err != nil {
		// log.Printf("Error getting progress for dashboard: %v", err)
		progress = map[string]interface{}{
			"error": "Failed to load progress data",
		}
	}

	// Add template functions
	funcMap := template.FuncMap{
		"mult": func(a, b float64) float64 { return a * b },
		"div":  func(a, b float64) float64 { return a / b },
		"sub":  func(a, b float64) float64 { return a - b },
	}

	data := struct {
		Title    string
		Username string
		UserType string
		CSPNonce string
		Progress map[string]interface{}
	}{
		Title:    "My Progress",
		Username: session.Username,
		UserType: session.Role,
		CSPNonce: generateNonce(),
		Progress: progress,
	}

	tmpl := template.Must(template.New("progress_dashboard.html").Funcs(funcMap).ParseFiles("templates/progress_dashboard.html"))
	err = tmpl.Execute(w, data)
	if err != nil {
		// log.Printf("Error rendering progress dashboard: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

// Helper functions
func getMilestonesForRole(userType string) []ProgressMilestone {
	var milestones []ProgressMilestone

	// Common milestones for all users
	common := []ProgressMilestone{
		{ID: "first_login", Name: "First Login", Description: "Successfully logged into the system", Category: "Getting Started", Required: true, Order: 1},
		{ID: "profile_updated", Name: "Profile Updated", Description: "Updated your profile information", Category: "Getting Started", Required: false, Order: 2},
		{ID: "help_accessed", Name: "Help Center Visited", Description: "Accessed the help center", Category: "Getting Started", Required: false, Order: 3},
		{ID: "tour_completed", Name: "Tour Completed", Description: "Completed the interactive tour", Category: "Getting Started", Required: true, Order: 4},
	}

	milestones = append(milestones, common...)

	// Role-specific milestones
	if strings.Contains(userType, "Manager") {
		manager := []ProgressMilestone{
			{ID: "fleet_viewed", Name: "Fleet Reviewed", Description: "Viewed the fleet management page", Category: "Core Features", Required: true, Order: 5},
			{ID: "route_assigned", Name: "Route Assigned", Description: "Assigned a driver to a route", Category: "Core Features", Required: true, Order: 6},
			{ID: "user_approved", Name: "User Approved", Description: "Approved a new driver", Category: "Management", Required: false, Order: 7},
			{ID: "report_generated", Name: "Report Generated", Description: "Generated your first report", Category: "Analytics", Required: true, Order: 8},
			{ID: "maintenance_scheduled", Name: "Maintenance Scheduled", Description: "Scheduled vehicle maintenance", Category: "Fleet", Required: false, Order: 9},
			{ID: "ecse_managed", Name: "ECSE Reviewed", Description: "Accessed ECSE student data", Category: "Students", Required: false, Order: 10},
		}
		milestones = append(milestones, manager...)
	} else if strings.Contains(userType, "Driver") {
		driver := []ProgressMilestone{
			{ID: "route_viewed", Name: "Route Checked", Description: "Viewed your route assignment", Category: "Core Features", Required: true, Order: 5},
			{ID: "students_viewed", Name: "Student Roster Reviewed", Description: "Viewed your student roster", Category: "Core Features", Required: true, Order: 6},
			{ID: "trip_logged", Name: "First Trip Logged", Description: "Completed your first trip log", Category: "Core Features", Required: true, Order: 7},
			{ID: "attendance_taken", Name: "Attendance Recorded", Description: "Recorded student attendance", Category: "Daily Tasks", Required: true, Order: 8},
			{ID: "mileage_logged", Name: "Mileage Tracked", Description: "Recorded trip mileage", Category: "Daily Tasks", Required: true, Order: 9},
			{ID: "student_added", Name: "Student Added", Description: "Added a new student to roster", Category: "Management", Required: false, Order: 10},
		}
		milestones = append(milestones, driver...)
	}

	return milestones
}

func getLastActivity(features []UserProgress) time.Time {
	if len(features) == 0 {
		return time.Time{}
	}
	return features[0].LastAccessed // Already sorted by last_accessed DESC
}

func getDaysSinceStart(features []UserProgress) int {
	if len(features) == 0 {
		return 0
	}
	
	// Find earliest access
	earliest := time.Now()
	for _, f := range features {
		if f.FirstAccessed.Before(earliest) {
			earliest = f.FirstAccessed
		}
	}
	
	return int(time.Since(earliest).Hours() / 24)
}

func getMostUsedFeatures(features []UserProgress, limit int) []UserProgress {
	// Sort by access count
	sorted := make([]UserProgress, len(features))
	copy(sorted, features)
	
	// Simple bubble sort for small datasets
	for i := 0; i < len(sorted)-1; i++ {
		for j := 0; j < len(sorted)-i-1; j++ {
			if sorted[j].AccessCount < sorted[j+1].AccessCount {
				sorted[j], sorted[j+1] = sorted[j+1], sorted[j]
			}
		}
	}
	
	if len(sorted) > limit {
		return sorted[:limit]
	}
	return sorted
}

func getRecommendedNextSteps(milestones []ProgressMilestone, userType string) []string {
	var recommendations []string
	
	// Find incomplete required milestones first
	for _, m := range milestones {
		if m.Required && !m.Completed {
			switch m.ID {
			case "tour_completed":
				recommendations = append(recommendations, "Take the interactive tour to learn the system")
			case "fleet_viewed":
				recommendations = append(recommendations, "Review your fleet status")
			case "route_assigned":
				recommendations = append(recommendations, "Assign drivers to their routes")
			case "trip_logged":
				recommendations = append(recommendations, "Log your first trip")
			case "students_viewed":
				recommendations = append(recommendations, "Review your student roster")
			}
			
			if len(recommendations) >= 3 {
				return recommendations
			}
		}
	}
	
	// Add some optional but helpful milestones
	for _, m := range milestones {
		if !m.Required && !m.Completed {
			switch m.ID {
			case "practice_mode":
				recommendations = append(recommendations, "Try Practice Mode to explore safely")
			case "quick_reference":
				recommendations = append(recommendations, "Print the Quick Reference guide")
			case "help_video":
				recommendations = append(recommendations, "Watch help videos for visual guidance")
			}
			
			if len(recommendations) >= 3 {
				return recommendations
			}
		}
	}
	
	// If all done, suggest advanced features
	if len(recommendations) == 0 {
		if strings.Contains(userType, "Manager") {
			recommendations = append(recommendations, "Explore advanced analytics features")
			recommendations = append(recommendations, "Set up automated reports")
		} else {
			recommendations = append(recommendations, "Review your performance metrics")
			recommendations = append(recommendations, "Update emergency contact information")
		}
	}
	
	return recommendations
}

// Create the user_progress table if it doesn't exist
func createProgressTable(db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS user_progress (
		id SERIAL PRIMARY KEY,
		user_id INTEGER NOT NULL REFERENCES users(id),
		feature VARCHAR(100) NOT NULL,
		first_accessed TIMESTAMP NOT NULL DEFAULT NOW(),
		last_accessed TIMESTAMP NOT NULL DEFAULT NOW(),
		completion_status VARCHAR(20) DEFAULT 'not_started',
		access_count INTEGER DEFAULT 0,
		created_at TIMESTAMP DEFAULT NOW(),
		UNIQUE(user_id, feature)
	);

	CREATE INDEX IF NOT EXISTS idx_user_progress_user_id ON user_progress(user_id);
	CREATE INDEX IF NOT EXISTS idx_user_progress_feature ON user_progress(feature);
	`
	
	_, err := db.Exec(query)
	return err
}