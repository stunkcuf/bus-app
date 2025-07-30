package main

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"strings"
)

// VideoTutorial represents a video tutorial
type VideoTutorial struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Duration    string   `json:"duration"`
	Category    string   `json:"category"`
	Role        string   `json:"role"` // "all", "manager", "driver"
	Thumbnail   string   `json:"thumbnail"`
	VideoURL    string   `json:"video_url"`
	Transcript  string   `json:"transcript"`
	Topics      []string `json:"topics"`
	Order       int      `json:"order"`
	Views       int      `json:"views"`
}

// VideoCategory represents a category of videos
type VideoCategory struct {
	ID          string
	Name        string
	Description string
	Icon        string
	Videos      []VideoTutorial
}

func videoTutorialsHandler(w http.ResponseWriter, r *http.Request) {
	session := getUserFromSession(r)
	if session == nil {
		log.Printf("Video tutorials access without login")
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// Get category filter from query params
	categoryFilter := r.URL.Query().Get("category")
	
	// Get all video categories
	categories := getVideoCategories(session.Role)
	
	// Filter videos by category if specified
	var selectedCategory *VideoCategory
	if categoryFilter != "" {
		for i := range categories {
			if categories[i].ID == categoryFilter {
				selectedCategory = &categories[i]
				break
			}
		}
	}

	data := struct {
		Title            string
		Username         string
		UserType         string
		CSPNonce         string
		Categories       []VideoCategory
		SelectedCategory *VideoCategory
		AllVideos        []VideoTutorial
		SearchEnabled    bool
	}{
		Title:            "Video Tutorials",
		Username:         session.Username,
		UserType:         session.Role,
		CSPNonce:         generateNonce(),
		Categories:       categories,
		SelectedCategory: selectedCategory,
		AllVideos:        getAllVideos(session.Role),
		SearchEnabled:    true,
	}

	tmpl := template.Must(template.ParseFiles("templates/video_tutorials.html"))
	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("Error rendering video tutorials: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

func videoPlayerHandler(w http.ResponseWriter, r *http.Request) {
	session := getUserFromSession(r)
	if session == nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// Extract video ID from URL
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 3 {
		http.Error(w, "Invalid video ID", http.StatusBadRequest)
		return
	}
	videoID := parts[len(parts)-1]

	// Get video details
	video := getVideoByID(videoID)
	if video == nil {
		http.Error(w, "Video not found", http.StatusNotFound)
		return
	}

	// Check role permissions
	if video.Role != "all" && !strings.Contains(session.Role, strings.Title(video.Role)) {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	// Get related videos
	relatedVideos := getRelatedVideos(videoID, video.Category, 4)

	data := struct {
		Title         string
		Username      string
		UserType      string
		CSPNonce      string
		Video         *VideoTutorial
		RelatedVideos []VideoTutorial
		NextVideo     *VideoTutorial
	}{
		Title:         video.Title,
		Username:      session.Username,
		UserType:      session.Role,
		CSPNonce:      generateNonce(),
		Video:         video,
		RelatedVideos: relatedVideos,
		NextVideo:     getNextVideo(video),
	}

	tmpl := template.Must(template.ParseFiles("templates/video_player.html"))
	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("Error rendering video player: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

// API endpoint for video search
func videoSearchHandler(w http.ResponseWriter, r *http.Request) {
	session := getUserFromSession(r)
	if session == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	query := r.URL.Query().Get("q")
	if query == "" {
		json.NewEncoder(w).Encode([]VideoTutorial{})
		return
	}

	// Search videos
	results := searchVideos(query, session.Role)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

// Video data functions
func getVideoCategories(userType string) []VideoCategory {
	categories := []VideoCategory{
		{
			ID:          "getting-started",
			Name:        "Getting Started",
			Description: "Introduction and basic navigation",
			Icon:        "bi-play-circle",
			Videos:      filterVideosByCategory("getting-started", userType),
		},
		{
			ID:          "daily-tasks",
			Name:        "Daily Tasks",
			Description: "Common daily operations",
			Icon:        "bi-calendar-check",
			Videos:      filterVideosByCategory("daily-tasks", userType),
		},
		{
			ID:          "fleet-management",
			Name:        "Fleet Management",
			Description: "Managing vehicles and maintenance",
			Icon:        "bi-bus-front",
			Videos:      filterVideosByCategory("fleet-management", userType),
		},
		{
			ID:          "student-management",
			Name:        "Student Management",
			Description: "Working with student data",
			Icon:        "bi-people",
			Videos:      filterVideosByCategory("student-management", userType),
		},
		{
			ID:          "reporting",
			Name:        "Reports & Analytics",
			Description: "Generating and understanding reports",
			Icon:        "bi-graph-up",
			Videos:      filterVideosByCategory("reporting", userType),
		},
		{
			ID:          "troubleshooting",
			Name:        "Troubleshooting",
			Description: "Solving common problems",
			Icon:        "bi-tools",
			Videos:      filterVideosByCategory("troubleshooting", userType),
		},
	}

	return categories
}

func getAllVideos(userType string) []VideoTutorial {
	allVideos := []VideoTutorial{
		// Getting Started Videos
		{
			ID:          "welcome-tour",
			Title:       "Welcome to Fleet Management",
			Description: "A complete overview of the system and its capabilities",
			Duration:    "3:45",
			Category:    "getting-started",
			Role:        "all",
			Thumbnail:   "/static/images/video-thumb-welcome.jpg",
			VideoURL:    "https://example.com/videos/welcome-tour.mp4",
			Topics:      []string{"Overview", "Navigation", "Dashboard"},
			Order:       1,
		},
		{
			ID:          "first-login",
			Title:       "Your First Login",
			Description: "Step-by-step guide to logging in and initial setup",
			Duration:    "2:30",
			Category:    "getting-started",
			Role:        "all",
			Thumbnail:   "/static/images/video-thumb-login.jpg",
			VideoURL:    "https://example.com/videos/first-login.mp4",
			Topics:      []string{"Login", "Password", "Profile"},
			Order:       2,
		},
		{
			ID:          "navigation-basics",
			Title:       "Navigating the System",
			Description: "Learn how to find and use all features efficiently",
			Duration:    "4:15",
			Category:    "getting-started",
			Role:        "all",
			Thumbnail:   "/static/images/video-thumb-nav.jpg",
			VideoURL:    "https://example.com/videos/navigation.mp4",
			Topics:      []string{"Menu", "Search", "Shortcuts"},
			Order:       3,
		},

		// Driver-Specific Videos
		{
			ID:          "driver-morning-routine",
			Title:       "Driver Morning Routine",
			Description: "Complete walkthrough of morning route procedures",
			Duration:    "6:20",
			Category:    "daily-tasks",
			Role:        "driver",
			Thumbnail:   "/static/images/video-thumb-morning.jpg",
			VideoURL:    "https://example.com/videos/morning-routine.mp4",
			Topics:      []string{"Pre-trip", "Attendance", "Logging"},
			Order:       4,
		},
		{
			ID:          "logging-trips",
			Title:       "How to Log Your Trips",
			Description: "Detailed guide on recording trip information accurately",
			Duration:    "5:00",
			Category:    "daily-tasks",
			Role:        "driver",
			Thumbnail:   "/static/images/video-thumb-logging.jpg",
			VideoURL:    "https://example.com/videos/trip-logging.mp4",
			Topics:      []string{"Mileage", "Times", "Students"},
			Order:       5,
		},
		{
			ID:          "student-attendance",
			Title:       "Taking Student Attendance",
			Description: "Best practices for accurate attendance tracking",
			Duration:    "3:30",
			Category:    "student-management",
			Role:        "driver",
			Thumbnail:   "/static/images/video-thumb-attendance.jpg",
			VideoURL:    "https://example.com/videos/attendance.mp4",
			Topics:      []string{"Attendance", "Absences", "Notes"},
			Order:       6,
		},

		// Manager-Specific Videos
		{
			ID:          "manager-overview",
			Title:       "Manager Dashboard Overview",
			Description: "Understanding your command center",
			Duration:    "7:00",
			Category:    "getting-started",
			Role:        "manager",
			Thumbnail:   "/static/images/video-thumb-manager.jpg",
			VideoURL:    "https://example.com/videos/manager-overview.mp4",
			Topics:      []string{"Dashboard", "Metrics", "Monitoring"},
			Order:       7,
		},
		{
			ID:          "route-assignment",
			Title:       "Assigning Routes to Drivers",
			Description: "How to create and manage route assignments",
			Duration:    "5:45",
			Category:    "fleet-management",
			Role:        "manager",
			Thumbnail:   "/static/images/video-thumb-routes.jpg",
			VideoURL:    "https://example.com/videos/route-assignment.mp4",
			Topics:      []string{"Routes", "Assignments", "Conflicts"},
			Order:       8,
		},
		{
			ID:          "user-management",
			Title:       "Managing User Accounts",
			Description: "Creating and managing driver accounts",
			Duration:    "4:30",
			Category:    "fleet-management",
			Role:        "manager",
			Thumbnail:   "/static/images/video-thumb-users.jpg",
			VideoURL:    "https://example.com/videos/user-management.mp4",
			Topics:      []string{"Users", "Permissions", "Approval"},
			Order:       9,
		},
		{
			ID:          "generating-reports",
			Title:       "Creating Custom Reports",
			Description: "Use the report builder to get the data you need",
			Duration:    "6:15",
			Category:    "reporting",
			Role:        "manager",
			Thumbnail:   "/static/images/video-thumb-reports.jpg",
			VideoURL:    "https://example.com/videos/reports.mp4",
			Topics:      []string{"Reports", "Filters", "Export"},
			Order:       10,
		},

		// Fleet Management Videos
		{
			ID:          "vehicle-maintenance",
			Title:       "Scheduling Vehicle Maintenance",
			Description: "Keep your fleet in top condition",
			Duration:    "5:30",
			Category:    "fleet-management",
			Role:        "all",
			Thumbnail:   "/static/images/video-thumb-maintenance.jpg",
			VideoURL:    "https://example.com/videos/maintenance.mp4",
			Topics:      []string{"Maintenance", "Scheduling", "Tracking"},
			Order:       11,
		},
		{
			ID:          "adding-vehicles",
			Title:       "Adding New Vehicles",
			Description: "Step-by-step guide to adding buses to your fleet",
			Duration:    "3:45",
			Category:    "fleet-management",
			Role:        "manager",
			Thumbnail:   "/static/images/video-thumb-add-bus.jpg",
			VideoURL:    "https://example.com/videos/add-vehicle.mp4",
			Topics:      []string{"Vehicles", "Setup", "Documentation"},
			Order:       12,
		},

		// Student Management Videos
		{
			ID:          "adding-students",
			Title:       "Adding New Students",
			Description: "How to add students to your route",
			Duration:    "4:00",
			Category:    "student-management",
			Role:        "all",
			Thumbnail:   "/static/images/video-thumb-add-student.jpg",
			VideoURL:    "https://example.com/videos/add-student.mp4",
			Topics:      []string{"Students", "Information", "Routes"},
			Order:       13,
		},
		{
			ID:          "ecse-management",
			Title:       "Managing ECSE Students",
			Description: "Special considerations for special education transport",
			Duration:    "6:00",
			Category:    "student-management",
			Role:        "all",
			Thumbnail:   "/static/images/video-thumb-ecse.jpg",
			VideoURL:    "https://example.com/videos/ecse.mp4",
			Topics:      []string{"ECSE", "IEP", "Services"},
			Order:       14,
		},

		// Troubleshooting Videos
		{
			ID:          "common-issues",
			Title:       "Solving Common Problems",
			Description: "Quick fixes for frequent issues",
			Duration:    "5:00",
			Category:    "troubleshooting",
			Role:        "all",
			Thumbnail:   "/static/images/video-thumb-troubleshoot.jpg",
			VideoURL:    "https://example.com/videos/troubleshooting.mp4",
			Topics:      []string{"Errors", "Solutions", "Tips"},
			Order:       15,
		},
		{
			ID:          "getting-help",
			Title:       "How to Get Help",
			Description: "Using the help system and contacting support",
			Duration:    "2:45",
			Category:    "troubleshooting",
			Role:        "all",
			Thumbnail:   "/static/images/video-thumb-help.jpg",
			VideoURL:    "https://example.com/videos/getting-help.mp4",
			Topics:      []string{"Help", "Support", "Documentation"},
			Order:       16,
		},
	}

	// Filter videos based on user role
	var filtered []VideoTutorial
	for _, video := range allVideos {
		if video.Role == "all" || strings.Contains(userType, strings.Title(video.Role)) {
			filtered = append(filtered, video)
		}
	}

	return filtered
}

func filterVideosByCategory(category string, userType string) []VideoTutorial {
	var videos []VideoTutorial
	allVideos := getAllVideos(userType)
	
	for _, video := range allVideos {
		if video.Category == category {
			videos = append(videos, video)
		}
	}
	
	return videos
}

func getVideoByID(id string) *VideoTutorial {
	allVideos := getAllVideos("all")
	for _, video := range allVideos {
		if video.ID == id {
			return &video
		}
	}
	return nil
}

func getRelatedVideos(currentID string, category string, limit int) []VideoTutorial {
	var related []VideoTutorial
	allVideos := getAllVideos("all")
	
	// First, get videos from same category
	for _, video := range allVideos {
		if video.ID != currentID && video.Category == category {
			related = append(related, video)
			if len(related) >= limit {
				break
			}
		}
	}
	
	// If not enough, add videos from other categories
	if len(related) < limit {
		for _, video := range allVideos {
			if video.ID != currentID && video.Category != category {
				related = append(related, video)
				if len(related) >= limit {
					break
				}
			}
		}
	}
	
	return related
}

func getNextVideo(current *VideoTutorial) *VideoTutorial {
	allVideos := getAllVideos("all")
	for i, video := range allVideos {
		if video.ID == current.ID && i < len(allVideos)-1 {
			return &allVideos[i+1]
		}
	}
	return nil
}

func searchVideos(query string, userType string) []VideoTutorial {
	var results []VideoTutorial
	query = strings.ToLower(query)
	allVideos := getAllVideos(userType)
	
	for _, video := range allVideos {
		// Search in title, description, and topics
		if strings.Contains(strings.ToLower(video.Title), query) ||
		   strings.Contains(strings.ToLower(video.Description), query) {
			results = append(results, video)
			continue
		}
		
		// Search in topics
		for _, topic := range video.Topics {
			if strings.Contains(strings.ToLower(topic), query) {
				results = append(results, video)
				break
			}
		}
	}
	
	return results
}