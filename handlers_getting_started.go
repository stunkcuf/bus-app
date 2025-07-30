package main

import (
	"html/template"
	"log"
	"net/http"
)

func gettingStartedHandler(w http.ResponseWriter, r *http.Request) {
	session := getUserFromSession(r)
	if session == nil {
		log.Printf("Getting started guide access without login: path=%s", r.URL.Path)
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// Get role from URL parameter or user session
	role := r.URL.Query().Get("role")
	if role == "" {
		// Determine role from user
		if session.Role == "manager" {
			role = "manager"
		} else if session.Role == "driver" {
			role = "driver"
		} else {
			role = "general"
		}
	}

	// Data structure for the guide
	data := struct {
		Title       string
		Role        string
		UserType    string
		Username    string
		CSPNonce    string
		Steps       []GuideStep
		QuickLinks  []QuickLink
		Resources   []Resource
		Tips        []string
	}{
		Title:    "Getting Started Guide",
		Role:     role,
		UserType: session.Role,
		Username: session.Username,
		CSPNonce: generateNonce(),
	}

	// Populate guide content based on role
	switch role {
	case "manager":
		data.Title = "Manager Getting Started Guide"
		data.Steps = getManagerGuideSteps()
		data.QuickLinks = getManagerQuickLinks()
		data.Resources = getManagerResources()
		data.Tips = getManagerTips()
	case "driver":
		data.Title = "Driver Getting Started Guide"
		data.Steps = getDriverGuideSteps()
		data.QuickLinks = getDriverQuickLinks()
		data.Resources = getDriverResources()
		data.Tips = getDriverTips()
	default:
		data.Title = "Getting Started Guide"
		data.Steps = getGeneralGuideSteps()
		data.QuickLinks = getGeneralQuickLinks()
		data.Resources = getGeneralResources()
		data.Tips = getGeneralTips()
	}

	tmpl := template.Must(template.ParseFiles("templates/getting_started.html"))
	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("Error rendering getting started guide: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

// GuideStep represents a step in the getting started guide
type GuideStep struct {
	Number      int
	Title       string
	Description string
	Action      string
	ActionURL   string
	Icon        string
	Completed   bool
}

// QuickLink represents a quick access link
type QuickLink struct {
	Title       string
	Description string
	URL         string
	Icon        string
}

// Resource represents a helpful resource
type Resource struct {
	Title       string
	Description string
	Type        string // "video", "document", "tutorial"
	URL         string
	Icon        string
}

// Manager guide content
func getManagerGuideSteps() []GuideStep {
	return []GuideStep{
		{
			Number:      1,
			Title:       "Review Your Fleet",
			Description: "Start by checking your current fleet status. See how many buses are active, in maintenance, or out of service.",
			Action:      "View Fleet",
			ActionURL:   "/fleet",
			Icon:        "bi-bus-front",
		},
		{
			Number:      2,
			Title:       "Check Pending Driver Approvals",
			Description: "Review and approve any pending driver registrations. New drivers need manager approval before they can access the system.",
			Action:      "Manage Users",
			ActionURL:   "/manage-users",
			Icon:        "bi-person-check",
		},
		{
			Number:      3,
			Title:       "Assign Routes to Drivers",
			Description: "Make sure all your drivers are assigned to appropriate routes with their buses. Use the visual assignment tool for easy management.",
			Action:      "Assign Routes",
			ActionURL:   "/assign-routes",
			Icon:        "bi-diagram-3",
		},
		{
			Number:      4,
			Title:       "Review Student Rosters",
			Description: "Check that all students are properly assigned to routes. Special education (ECSE) students require extra attention.",
			Action:      "ECSE Dashboard",
			ActionURL:   "/ecse-dashboard",
			Icon:        "bi-mortarboard",
		},
		{
			Number:      5,
			Title:       "Set Up Maintenance Schedules",
			Description: "Regular maintenance is crucial for safety. Review and schedule maintenance for all vehicles in your fleet.",
			Action:      "Maintenance Records",
			ActionURL:   "/maintenance-records",
			Icon:        "bi-tools",
		},
		{
			Number:      6,
			Title:       "Explore Analytics",
			Description: "Use the analytics dashboard to track mileage, fuel costs, and driver performance metrics.",
			Action:      "View Analytics",
			ActionURL:   "/analytics-dashboard",
			Icon:        "bi-graph-up",
		},
	}
}

func getManagerQuickLinks() []QuickLink {
	return []QuickLink{
		{
			Title:       "Daily Operations",
			Description: "Monitor today's trips and driver logs",
			URL:         "/manager-dashboard",
			Icon:        "bi-speedometer2",
		},
		{
			Title:       "Import Data",
			Description: "Bulk import ECSE students from Excel",
			URL:         "/import-ecse",
			Icon:        "bi-file-earmark-excel",
		},
		{
			Title:       "Generate Reports",
			Description: "Create custom reports for administration",
			URL:         "/report-builder",
			Icon:        "bi-file-earmark-bar-graph",
		},
		{
			Title:       "GPS Tracking",
			Description: "Real-time vehicle location tracking",
			URL:         "/gps-tracking",
			Icon:        "bi-geo-alt-fill",
		},
	}
}

func getManagerResources() []Resource {
	return []Resource{
		{
			Title:       "Fleet Management Best Practices",
			Description: "Learn effective strategies for managing your school bus fleet",
			Type:        "document",
			URL:         "/help/article/fleet-best-practices",
			Icon:        "bi-book",
		},
		{
			Title:       "Route Optimization Guide",
			Description: "Tips for creating efficient routes that save time and fuel",
			Type:        "document",
			URL:         "/help/article/route-optimization",
			Icon:        "bi-map",
		},
		{
			Title:       "Safety Compliance Checklist",
			Description: "Ensure your fleet meets all safety regulations",
			Type:        "document",
			URL:         "/help/article/safety-compliance",
			Icon:        "bi-shield-check",
		},
		{
			Title:       "Video: Managing ECSE Students",
			Description: "Special considerations for special education transportation",
			Type:        "video",
			URL:         "/help/videos/ecse-management",
			Icon:        "bi-play-circle",
		},
	}
}

func getManagerTips() []string {
	return []string{
		"Check the dashboard daily for driver activity and system alerts",
		"Regular maintenance prevents costly breakdowns - schedule inspections monthly",
		"Use the analytics dashboard to identify cost-saving opportunities",
		"Export reports weekly for administrative review",
		"Keep driver contact information up-to-date for emergencies",
	}
}

// Driver guide content
func getDriverGuideSteps() []GuideStep {
	return []GuideStep{
		{
			Number:      1,
			Title:       "Check Your Route Assignment",
			Description: "Make sure you know which route and bus you're assigned to. Contact your manager if you haven't been assigned yet.",
			Action:      "View Dashboard",
			ActionURL:   "/driver-dashboard",
			Icon:        "bi-speedometer2",
		},
		{
			Number:      2,
			Title:       "Review Your Student Roster",
			Description: "Familiarize yourself with the students on your route, their pickup/dropoff times, and guardian contact information.",
			Action:      "Manage Students",
			ActionURL:   "/students",
			Icon:        "bi-people",
		},
		{
			Number:      3,
			Title:       "Pre-Trip Inspection",
			Description: "Always perform a pre-trip inspection of your bus. Check tires, lights, mirrors, and safety equipment.",
			Action:      "Inspection Checklist",
			ActionURL:   "/help/article/pre-trip-inspection",
			Icon:        "bi-clipboard-check",
		},
		{
			Number:      4,
			Title:       "Log Your First Trip",
			Description: "After completing a route, log your trip details including mileage, attendance, and any incidents.",
			Action:      "Log Morning Trip",
			ActionURL:   "/driver-dashboard?period=morning",
			Icon:        "bi-journal-text",
		},
		{
			Number:      5,
			Title:       "Understand Emergency Procedures",
			Description: "Know what to do in case of emergencies, including evacuation procedures and emergency contacts.",
			Action:      "Emergency Guide",
			ActionURL:   "/help/article/emergency-procedures",
			Icon:        "bi-exclamation-triangle",
		},
	}
}

func getDriverQuickLinks() []QuickLink {
	return []QuickLink{
		{
			Title:       "Morning Route",
			Description: "Quick access to morning trip logging",
			URL:         "/driver-dashboard?period=morning",
			Icon:        "bi-sunrise",
		},
		{
			Title:       "Afternoon Route",
			Description: "Quick access to afternoon trip logging",
			URL:         "/driver-dashboard?period=afternoon",
			Icon:        "bi-sunset",
		},
		{
			Title:       "Student Roster",
			Description: "View and manage your students",
			URL:         "/students",
			Icon:        "bi-people",
		},
		{
			Title:       "Trip History",
			Description: "Review your past trip logs",
			URL:         "/reports",
			Icon:        "bi-clock-history",
		},
	}
}

func getDriverResources() []Resource {
	return []Resource{
		{
			Title:       "Safe Driving Practices",
			Description: "Essential safety tips for school bus drivers",
			Type:        "document",
			URL:         "/help/article/safe-driving",
			Icon:        "bi-shield-check",
		},
		{
			Title:       "Student Management Tips",
			Description: "Best practices for managing student behavior",
			Type:        "document",
			URL:         "/help/article/student-management",
			Icon:        "bi-people",
		},
		{
			Title:       "Video: Daily Route Logging",
			Description: "Step-by-step guide to logging your trips",
			Type:        "video",
			URL:         "/help/videos/route-logging",
			Icon:        "bi-play-circle",
		},
		{
			Title:       "Weather Driving Guide",
			Description: "Driving safely in various weather conditions",
			Type:        "document",
			URL:         "/help/article/weather-driving",
			Icon:        "bi-cloud-rain",
		},
	}
}

func getDriverTips() []string {
	return []string{
		"Always perform pre-trip inspections - safety first!",
		"Log trips immediately after completion for accuracy",
		"Keep emergency contact numbers readily accessible",
		"Communicate any bus issues to maintenance immediately",
		"Update student attendance in real-time during routes",
	}
}

// General guide content (for other users)
func getGeneralGuideSteps() []GuideStep {
	return []GuideStep{
		{
			Number:      1,
			Title:       "Explore the Dashboard",
			Description: "Familiarize yourself with the main dashboard and available features.",
			Action:      "Go to Dashboard",
			ActionURL:   "/dashboard",
			Icon:        "bi-speedometer2",
		},
		{
			Number:      2,
			Title:       "Update Your Profile",
			Description: "Keep your contact information and password up to date.",
			Action:      "Edit Profile",
			ActionURL:   "/profile",
			Icon:        "bi-person-circle",
		},
		{
			Number:      3,
			Title:       "Learn the System",
			Description: "Visit the help center to learn about all available features.",
			Action:      "Help Center",
			ActionURL:   "/help-center",
			Icon:        "bi-question-circle",
		},
	}
}

func getGeneralQuickLinks() []QuickLink {
	return []QuickLink{
		{
			Title:       "Help Center",
			Description: "Get help with any feature",
			URL:         "/help-center",
			Icon:        "bi-question-circle",
		},
		{
			Title:       "Contact Support",
			Description: "Reach out for assistance",
			URL:         "/help/article/contact-support",
			Icon:        "bi-headset",
		},
	}
}

func getGeneralResources() []Resource {
	return []Resource{
		{
			Title:       "System Overview",
			Description: "Understanding the fleet management system",
			Type:        "document",
			URL:         "/help/article/system-overview",
			Icon:        "bi-book",
		},
		{
			Title:       "FAQs",
			Description: "Frequently asked questions",
			Type:        "document",
			URL:         "/help/article/faqs",
			Icon:        "bi-question-square",
		},
	}
}

func getGeneralTips() []string {
	return []string{
		"Keep your login credentials secure",
		"Contact your administrator for access issues",
		"Check the help center for detailed guides",
	}
}