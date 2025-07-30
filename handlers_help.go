package main

import (
	"encoding/json"
	"net/http"
	"strings"
)

// helpCenterHandler displays the main help center page
func helpCenterHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	data := map[string]interface{}{
		"Title":       "Help Center",
		"User":        user,
		"CSRFToken":   getSessionCSRFToken(r),
		"Navigation":  getNavigation(user, "Help", ""),
		"CurrentPath": r.URL.Path,
	}

	renderTemplate(w, r, "help_center.html", data)
}

// helpArticleHandler displays a specific help article
func helpArticleHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	// Extract article ID from URL
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 3 {
		http.Error(w, "Invalid article ID", http.StatusBadRequest)
		return
	}
	articleID := parts[len(parts)-1]

	// Help articles content
	articles := map[string]struct {
		Title   string
		Content string
	}{
		"first-login": {
			Title: "Your First Login",
			Content: `
				<h2>Welcome to the Fleet Management System!</h2>
				<p>Follow these steps for your first login:</p>
				<ol>
					<li><strong>Enter Your Username:</strong> This was provided by your administrator</li>
					<li><strong>Enter Your Password:</strong> Use the temporary password given to you</li>
					<li><strong>Click "Sign In":</strong> You'll be taken to your dashboard</li>
					<li><strong>Change Your Password:</strong> For security, change your password on first login by clicking your name in the top right and selecting "Profile"</li>
				</ol>
				<div class="alert alert-info mt-3">
					<i class="bi bi-info-circle"></i> <strong>Tip:</strong> If you forget your password, contact your administrator for a reset.
				</div>
			`,
		},
		"navigation": {
			Title: "Navigating the System",
			Content: `
				<h2>Understanding the Dashboard and Menu</h2>
				<p>The Fleet Management System is organized into several main areas:</p>
				<h3>Dashboard</h3>
				<ul>
					<li><strong>Manager Dashboard:</strong> Overview of fleet operations, quick stats, and alerts</li>
					<li><strong>Driver Dashboard:</strong> Daily tasks, assigned students, and trip logs</li>
				</ul>
				<h3>Main Menu Items</h3>
				<ul>
					<li><strong>Fleet:</strong> View and manage buses and vehicles</li>
					<li><strong>Students:</strong> Manage student information and assignments</li>
					<li><strong>Routes:</strong> Create and assign routes to drivers</li>
					<li><strong>Reports:</strong> Generate various operational reports</li>
					<li><strong>Maintenance:</strong> Track vehicle service and repairs</li>
				</ul>
			`,
		},
		"daily-log": {
			Title: "Recording Daily Trips",
			Content: `
				<h2>How to Log Your Daily Routes</h2>
				<h3>Morning Route</h3>
				<ol>
					<li>Go to your Driver Dashboard</li>
					<li>Click "Log Morning Trip"</li>
					<li>Enter beginning odometer reading</li>
					<li>Record departure time</li>
					<li>Mark each student as present/absent</li>
					<li>Enter arrival time and ending odometer</li>
					<li>Add any notes about the trip</li>
					<li>Click "Save Trip Log"</li>
				</ol>
				<h3>Afternoon Route</h3>
				<p>Follow the same process, but select "Afternoon" as the trip period.</p>
				<div class="alert alert-warning mt-3">
					<i class="bi bi-exclamation-triangle"></i> <strong>Important:</strong> Always double-check your odometer readings before saving.
				</div>
			`,
		},
		"add-student": {
			Title: "Adding New Students",
			Content: `
				<h2>Step-by-Step Guide to Enroll Students</h2>
				<ol>
					<li><strong>Navigate to Students:</strong> Click "Students" in the main menu</li>
					<li><strong>Click "Add New Student":</strong> Look for the green button</li>
					<li><strong>Fill in Required Information:</strong>
						<ul>
							<li>Student name (first and last)</li>
							<li>Grade level</li>
							<li>Home address</li>
							<li>Guardian contact information</li>
							<li>Pickup and dropoff times</li>
						</ul>
					</li>
					<li><strong>Assign to Route:</strong> Select the appropriate route from the dropdown</li>
					<li><strong>Review and Save:</strong> Check all information is correct before saving</li>
				</ol>
			`,
		},
	}

	article, exists := articles[articleID]
	if !exists {
		http.Error(w, "Article not found", http.StatusNotFound)
		return
	}

	// Return JSON for API calls
	if r.Header.Get("Accept") == "application/json" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(article)
		return
	}

	// For now, redirect back to help center
	// In a full implementation, you'd create a help_article.html template
	http.Redirect(w, r, "/help-center#"+articleID, http.StatusFound)
}

// helpSearchHandler handles search queries in the help system
func helpSearchHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "Search query required", http.StatusBadRequest)
		return
	}

	// Simple search implementation - in production, you'd use a proper search engine
	searchResults := []struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		URL         string `json:"url"`
		Category    string `json:"category"`
	}{
		{
			Title:       "Recording Daily Trips",
			Description: "Learn how to log your morning and afternoon routes",
			URL:         "/help/article/daily-log",
			Category:    "Daily Operations",
		},
		{
			Title:       "Adding New Students",
			Description: "Step-by-step guide to enroll students in the system",
			URL:         "/help/article/add-student",
			Category:    "Student Management",
		},
	}

	// Filter results based on query
	var filteredResults []interface{}
	queryLower := strings.ToLower(query)
	for _, result := range searchResults {
		if strings.Contains(strings.ToLower(result.Title), queryLower) ||
			strings.Contains(strings.ToLower(result.Description), queryLower) {
			filteredResults = append(filteredResults, result)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"query":   query,
		"results": filteredResults,
		"count":   len(filteredResults),
	})
}

// helpVideoHandler serves video tutorial information
func helpVideoHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	// Extract video ID from URL
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 3 {
		http.Error(w, "Invalid video ID", http.StatusBadRequest)
		return
	}
	videoID := parts[len(parts)-1]

	// Video catalog - in production, this would come from a database
	videos := map[string]struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Duration    string `json:"duration"`
		URL         string `json:"url"`
		Thumbnail   string `json:"thumbnail"`
	}{
		"getting-started": {
			Title:       "Getting Started Guide",
			Description: "Introduction to the Fleet Management System",
			Duration:    "5 minutes",
			URL:         "/static/videos/getting-started.mp4", // Placeholder
			Thumbnail:   "/static/images/video-thumb-1.jpg",
		},
		"daily-operations": {
			Title:       "Daily Operations Walkthrough",
			Description: "Complete guide to daily driver tasks",
			Duration:    "8 minutes",
			URL:         "/static/videos/daily-operations.mp4",
			Thumbnail:   "/static/images/video-thumb-2.jpg",
		},
	}

	video, exists := videos[videoID]
	if !exists {
		http.Error(w, "Video not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(video)
}

