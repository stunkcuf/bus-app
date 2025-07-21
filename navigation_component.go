package main

import (
	"strings"
)

// NavigationItem represents a navigation menu item
type NavigationItem struct {
	Title       string `json:"title"`
	URL         string `json:"url"`
	Icon        string `json:"icon"`
	Description string `json:"description"`
	Active      bool   `json:"active"`
	Role        string `json:"role"`        // "manager", "driver", or "both"
	Badge       string `json:"badge"`       // Optional badge text
	BadgeColor  string `json:"badge_color"` // "primary", "success", "warning", "danger"
}

// Breadcrumb represents a breadcrumb navigation item
type Breadcrumb struct {
	Title string `json:"title"`
	URL   string `json:"url"`
	Icon  string `json:"icon"`
}

// NavigationData contains all navigation-related data for templates
type NavigationData struct {
	User        *User            `json:"user"`
	Breadcrumbs []Breadcrumb     `json:"breadcrumbs"`
	MainNav     []NavigationItem `json:"main_nav"`
	Items       []NavigationItem `json:"items"` // Alias for MainNav for backward compatibility
	QuickLinks  []NavigationItem `json:"quick_links"`
	CurrentPage string           `json:"current_page"`
	ShowBack    bool             `json:"show_back"`
	BackURL     string           `json:"back_url"`
	BackTitle   string           `json:"back_title"`
	PageTitle   string           `json:"page_title"`
	PageIcon    string           `json:"page_icon"`
}

// getNavigationData creates navigation data based on user role and current page
func getNavigationData(user *User, currentPage string) NavigationData {
	nav := NavigationData{
		User:        user,
		CurrentPage: currentPage,
		ShowBack:    true,
	}

	// Set breadcrumbs based on current page
	nav.Breadcrumbs = getBreadcrumbs(currentPage, user)

	// Set page title and icon
	nav.PageTitle, nav.PageIcon = getPageTitleAndIcon(currentPage)

	// Set back button URL and title based on breadcrumbs
	if len(nav.Breadcrumbs) > 1 {
		// Go back to the previous breadcrumb
		prevBreadcrumb := nav.Breadcrumbs[len(nav.Breadcrumbs)-2]
		nav.BackURL = prevBreadcrumb.URL
		nav.BackTitle = prevBreadcrumb.Title
	} else {
		// If only home breadcrumb, hide back button on dashboard
		nav.ShowBack = false
	}

	// Dashboard pages don't need back button
	if currentPage == "manager-dashboard" || currentPage == "driver-dashboard" {
		nav.ShowBack = false
	}

	// Create main navigation based on user role
	if user.Role == "manager" {
		nav.MainNav = getManagerNavigation(currentPage)
		nav.QuickLinks = getManagerQuickLinks()
	} else {
		nav.MainNav = getDriverNavigation(currentPage)
		nav.QuickLinks = getDriverQuickLinks()
	}
	
	// Set Items as alias for MainNav for backward compatibility
	nav.Items = nav.MainNav

	return nav
}

// getBreadcrumbs creates breadcrumb navigation for the current page
func getBreadcrumbs(currentPage string, user *User) []Breadcrumb {
	breadcrumbs := []Breadcrumb{
		{Title: "Home", URL: getDashboardURL(user), Icon: "bi-house-door-fill"},
	}

	switch currentPage {
	case "fleet":
		breadcrumbs = append(breadcrumbs, Breadcrumb{Title: "Fleet Management", URL: "/fleet", Icon: "bi-bus-front"})
	case "company-fleet":
		breadcrumbs = append(breadcrumbs, Breadcrumb{Title: "Company Vehicles", URL: "/company-fleet", Icon: "bi-truck"})
	case "fleet-vehicles":
		breadcrumbs = append(breadcrumbs,
			Breadcrumb{Title: "Fleet Management", URL: "/fleet", Icon: "bi-bus-front"},
			Breadcrumb{Title: "Fleet Vehicles", URL: "/fleet-vehicles", Icon: "bi-truck"},
		)
	case "maintenance-records":
		breadcrumbs = append(breadcrumbs,
			Breadcrumb{Title: "Fleet Management", URL: "/fleet", Icon: "bi-bus-front"},
			Breadcrumb{Title: "Maintenance Records", URL: "/maintenance-records", Icon: "bi-wrench"},
		)
	case "monthly-mileage-reports":
		breadcrumbs = append(breadcrumbs,
			Breadcrumb{Title: "Reports", URL: "/view-mileage-reports", Icon: "bi-file-text"},
			Breadcrumb{Title: "Monthly Mileage", URL: "/monthly-mileage-reports", Icon: "bi-speedometer2"},
		)
	case "service-records":
		breadcrumbs = append(breadcrumbs,
			Breadcrumb{Title: "Fleet Management", URL: "/fleet", Icon: "bi-bus-front"},
			Breadcrumb{Title: "Service Records", URL: "/service-records", Icon: "bi-clipboard-check"},
		)
	case "students":
		breadcrumbs = append(breadcrumbs, Breadcrumb{Title: "Student Management", URL: "/students", Icon: "bi-people"})
	case "assign-routes":
		breadcrumbs = append(breadcrumbs, Breadcrumb{Title: "Route Assignment", URL: "/assign-routes", Icon: "bi-diagram-3"})
	case "import-ecse":
		breadcrumbs = append(breadcrumbs, Breadcrumb{Title: "Import ECSE", URL: "/import-ecse", Icon: "bi-file-earmark-excel"})
	case "view-ecse-reports":
		breadcrumbs = append(breadcrumbs, Breadcrumb{Title: "ECSE Reports", URL: "/view-ecse-reports", Icon: "bi-clipboard-data"})
	case "edit-ecse-student":
		breadcrumbs = append(breadcrumbs,
			Breadcrumb{Title: "ECSE Reports", URL: "/view-ecse-reports", Icon: "bi-clipboard-data"},
			Breadcrumb{Title: "Edit Student", URL: "", Icon: "bi-pencil"},
		)
	case "import-mileage":
		breadcrumbs = append(breadcrumbs, Breadcrumb{Title: "Import Mileage", URL: "/import-mileage", Icon: "bi-speedometer2"})
	case "view-mileage-reports":
		breadcrumbs = append(breadcrumbs, Breadcrumb{Title: "Mileage Reports", URL: "/view-mileage-reports", Icon: "bi-graph-up"})
	case "approve-users":
		breadcrumbs = append(breadcrumbs, Breadcrumb{Title: "Approve Users", URL: "/approve-users", Icon: "bi-person-check"})
	case "users":
		breadcrumbs = append(breadcrumbs, Breadcrumb{Title: "User Management", URL: "/users", Icon: "bi-people-fill"})
	case "report-builder":
		breadcrumbs = append(breadcrumbs, Breadcrumb{Title: "Report Builder", URL: "/report-builder", Icon: "bi-file-text"})
	case "scheduled-exports":
		breadcrumbs = append(breadcrumbs, Breadcrumb{Title: "Scheduled Exports", URL: "/scheduled-exports", Icon: "bi-calendar-check"})
	case "driver-dashboard":
		// Driver dashboard is home, no additional breadcrumb needed
	case "manager-dashboard":
		// Manager dashboard is home, no additional breadcrumb needed
	}

	// Handle dynamic pages with IDs
	if strings.Contains(currentPage, "ecse-student/") {
		breadcrumbs = append(breadcrumbs,
			Breadcrumb{Title: "ECSE Reports", URL: "/view-ecse-reports", Icon: "bi-clipboard-data"},
			Breadcrumb{Title: "Student Details", URL: "", Icon: "bi-person-badge"},
		)
	} else if strings.Contains(currentPage, "driver/") {
		breadcrumbs = append(breadcrumbs,
			Breadcrumb{Title: "User Management", URL: "/users", Icon: "bi-people-fill"},
			Breadcrumb{Title: "Driver Profile", URL: "", Icon: "bi-person"},
		)
	} else if strings.Contains(currentPage, "bus-maintenance/") {
		breadcrumbs = append(breadcrumbs,
			Breadcrumb{Title: "Fleet Management", URL: "/fleet", Icon: "bi-bus-front"},
			Breadcrumb{Title: "Bus Maintenance", URL: "", Icon: "bi-tools"},
		)
	} else if strings.Contains(currentPage, "vehicle-maintenance/") {
		breadcrumbs = append(breadcrumbs,
			Breadcrumb{Title: "Company Vehicles", URL: "/company-fleet", Icon: "bi-truck"},
			Breadcrumb{Title: "Vehicle Maintenance", URL: "", Icon: "bi-tools"},
		)
	}

	return breadcrumbs
}

// getManagerNavigation returns navigation items for managers
func getManagerNavigation(currentPage string) []NavigationItem {
	items := []NavigationItem{
		{
			Title:       "Fleet Management",
			URL:         "/fleet",
			Icon:        "truck",
			Description: "Manage buses and vehicles",
			Active:      strings.Contains(currentPage, "fleet"),
			Role:        "manager",
		},
		{
			Title:       "Route Assignment",
			URL:         "/assign-routes",
			Icon:        "map",
			Description: "Assign drivers to routes",
			Active:      currentPage == "assign-routes",
			Role:        "manager",
		},
		{
			Title:       "Student Management",
			URL:         "/students",
			Icon:        "people",
			Description: "Manage student information",
			Active:      currentPage == "students",
			Role:        "both",
		},
		{
			Title:       "ECSE Reports",
			URL:         "/view-ecse-reports",
			Icon:        "clipboard-data",
			Description: "Special education reports",
			Active:      strings.Contains(currentPage, "ecse"),
			Role:        "manager",
		},
		{
			Title:       "Mileage Reports",
			URL:         "/view-mileage-reports",
			Icon:        "speedometer2",
			Description: "View mileage and fuel data",
			Active:      strings.Contains(currentPage, "mileage"),
			Role:        "manager",
		},
		{
			Title:       "User Management",
			URL:         "/approve-users",
			Icon:        "person-check",
			Description: "Approve and manage users",
			Active:      strings.Contains(currentPage, "user"),
			Role:        "manager",
		},
	}

	return items
}

// getDriverNavigation returns navigation items for drivers
func getDriverNavigation(currentPage string) []NavigationItem {
	items := []NavigationItem{
		{
			Title:       "Daily Logs",
			URL:         "/driver-dashboard",
			Icon:        "journal-text",
			Description: "Record daily trip information",
			Active:      currentPage == "driver-dashboard",
			Role:        "driver",
		},
		{
			Title:       "Student Management",
			URL:         "/students",
			Icon:        "people",
			Description: "View assigned students",
			Active:      currentPage == "students",
			Role:        "both",
		},
		{
			Title:       "Fleet Status",
			URL:         "/fleet",
			Icon:        "truck",
			Description: "View vehicle information",
			Active:      strings.Contains(currentPage, "fleet"),
			Role:        "both",
		},
	}

	return items
}

// getManagerQuickLinks returns quick action links for managers
func getManagerQuickLinks() []NavigationItem {
	return []NavigationItem{
		{
			Title:       "Add New Bus",
			URL:         "/fleet#add-bus",
			Icon:        "plus-circle",
			Description: "Register a new vehicle",
			Role:        "manager",
		},
		{
			Title:       "Import Data",
			URL:         "/import-mileage",
			Icon:        "upload",
			Description: "Import Excel files",
			Role:        "manager",
		},
		{
			Title:       "Generate Reports",
			URL:         "/report-builder",
			Icon:        "file-earmark-pdf",
			Description: "Create custom reports",
			Role:        "manager",
		},
	}
}

// getDriverQuickLinks returns quick action links for drivers
func getDriverQuickLinks() []NavigationItem {
	return []NavigationItem{
		{
			Title:       "Log Trip",
			URL:         "/driver-dashboard#log-trip",
			Icon:        "plus-circle",
			Description: "Record a new trip",
			Role:        "driver",
		},
		{
			Title:       "Report Issue",
			URL:         "/fleet#maintenance",
			Icon:        "exclamation-triangle",
			Description: "Report vehicle problems",
			Role:        "driver",
		},
	}
}

// getDashboardURL returns the appropriate dashboard URL for the user
func getDashboardURL(user *User) string {
	if user.Role == "manager" {
		return "/manager-dashboard"
	}
	return "/driver-dashboard"
}

// getPageTitleAndIcon returns the page title and icon for the current page
func getPageTitleAndIcon(currentPage string) (string, string) {
	switch currentPage {
	case "manager-dashboard":
		return "Manager Dashboard", "bi-speedometer2"
	case "driver-dashboard":
		return "Driver Dashboard", "bi-journal-text"
	case "fleet":
		return "Fleet Management", "bi-bus-front"
	case "company-fleet":
		return "Company Vehicles", "bi-truck"
	case "fleet-vehicles":
		return "Fleet Vehicles", "bi-truck"
	case "maintenance-records":
		return "Maintenance Records", "bi-wrench"
	case "monthly-mileage-reports":
		return "Monthly Mileage Reports", "bi-speedometer2"
	case "service-records":
		return "Service Records", "bi-clipboard-check"
	case "students":
		return "Student Management", "bi-people"
	case "assign-routes":
		return "Route Assignment", "bi-diagram-3"
	case "import-ecse":
		return "Import ECSE Data", "bi-file-earmark-excel"
	case "view-ecse-reports":
		return "ECSE Reports", "bi-clipboard-data"
	case "edit-ecse-student":
		return "Edit ECSE Student", "bi-pencil"
	case "import-mileage":
		return "Import Mileage", "bi-speedometer2"
	case "view-mileage-reports":
		return "Mileage Reports", "bi-graph-up"
	case "approve-users":
		return "Approve Users", "bi-person-check"
	case "users":
		return "User Management", "bi-people-fill"
	case "report-builder":
		return "Report Builder", "bi-file-text"
	case "scheduled-exports":
		return "Scheduled Exports", "bi-calendar-check"
	default:
		// Handle dynamic pages
		if strings.Contains(currentPage, "ecse-student/") {
			return "ECSE Student Details", "bi-person-badge"
		} else if strings.Contains(currentPage, "driver/") {
			return "Driver Profile", "bi-person"
		} else if strings.Contains(currentPage, "bus-maintenance/") {
			return "Bus Maintenance", "bi-tools"
		} else if strings.Contains(currentPage, "vehicle-maintenance/") {
			return "Vehicle Maintenance", "bi-tools"
		}
		return "Fleet Management System", "bi-bus-front"
	}
}

// getNavigation is a helper function that returns navigation data as a map
func getNavigation(user *User, activePage string, subPage string) map[string]interface{} {
	navData := getNavigationData(user, activePage)
	
	// Convert to map for template compatibility
	return map[string]interface{}{
		"ActivePage": activePage,
		"SubPage":    subPage,
		"User":       user,
		"Items":      navData.MainNav,
	}
}

// Note: Template functions are now integrated in main.go funcMap
