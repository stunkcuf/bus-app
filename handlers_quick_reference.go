package main

import (
	"html/template"
	"log"
	"net/http"
	"time"
)

func quickReferenceHandler(w http.ResponseWriter, r *http.Request) {
	session := getUserFromSession(r)
	if session == nil {
		log.Printf("Quick reference access without login")
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// Get guide type from URL parameter
	guideType := r.URL.Query().Get("type")
	if guideType == "" {
		// Determine type from user role
		if session.Role == "manager" {
			guideType = "manager"
		} else if session.Role == "driver" {
			guideType = "driver"
		} else {
			guideType = "general"
		}
	}

	// Data structure for the reference
	data := struct {
		Title           string
		GuideType       string
		UserType        string
		Username        string
		CSPNonce        string
		Sections        []ReferenceSection
		KeyboardShortcuts []Shortcut
		EmergencyContacts []Contact
		PrintDate       string
	}{
		Title:     "Quick Reference Guide",
		GuideType: guideType,
		UserType:  session.Role,
		Username:  session.Username,
		CSPNonce:  generateNonce(),
		PrintDate: time.Now().Format("January 2, 2006"),
	}

	// Populate content based on role
	switch guideType {
	case "manager":
		data.Title = "Manager Quick Reference"
		data.Sections = getManagerReferenceSections()
		data.KeyboardShortcuts = getManagerShortcuts()
		data.EmergencyContacts = getEmergencyContacts()
	case "driver":
		data.Title = "Driver Quick Reference"
		data.Sections = getDriverReferenceSections()
		data.KeyboardShortcuts = getDriverShortcuts()
		data.EmergencyContacts = getEmergencyContacts()
	default:
		data.Title = "Quick Reference Guide"
		data.Sections = getGeneralReferenceSections()
		data.KeyboardShortcuts = getGeneralShortcuts()
		data.EmergencyContacts = getEmergencyContacts()
	}

	tmpl := template.Must(template.ParseFiles("templates/quick_reference.html"))
	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("Error rendering quick reference: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

// ReferenceSection represents a section in the quick reference
type ReferenceSection struct {
	Title    string
	Icon     string
	Items    []ReferenceItem
	Priority string // "critical", "important", "helpful"
}

// ReferenceItem represents an item in a reference section
type ReferenceItem struct {
	Action      string
	Steps       []string
	Note        string
	Warning     string
	Shortcut    string
}

// Shortcut represents a keyboard shortcut
type Shortcut struct {
	Keys        string
	Description string
	Context     string // "global", "form", "list"
}

// Contact represents an emergency contact
type Contact struct {
	Name        string
	Role        string
	Phone       string
	Email       string
	Available   string
	Priority    string // "primary", "secondary"
}

// Manager reference content
func getManagerReferenceSections() []ReferenceSection {
	return []ReferenceSection{
		{
			Title:    "Daily Tasks",
			Icon:     "bi-calendar-check",
			Priority: "critical",
			Items: []ReferenceItem{
				{
					Action: "Check Pending Driver Approvals",
					Steps: []string{
						"Go to Manage Users",
						"Look for 'Pending' status",
						"Click 'Approve' or 'Reject'",
					},
					Shortcut: "From dashboard: Quick Actions → Manage Users",
				},
				{
					Action: "Review Daily Operations",
					Steps: []string{
						"Open Manager Dashboard",
						"Check Recent Activity section",
						"Review driver logs for the day",
					},
					Note: "Check for any missed routes or incidents",
				},
				{
					Action: "Monitor Fleet Status",
					Steps: []string{
						"Navigate to Fleet page",
						"Check bus status indicators",
						"Note any buses in maintenance",
					},
					Warning: "Red status requires immediate attention",
				},
			},
		},
		{
			Title:    "Route Management",
			Icon:     "bi-diagram-3",
			Priority: "important",
			Items: []ReferenceItem{
				{
					Action: "Assign Driver to Route",
					Steps: []string{
						"Go to Assign Routes",
						"Select driver from dropdown",
						"Choose bus and route",
						"Click 'Assign'",
					},
					Note: "System prevents double-booking automatically",
				},
				{
					Action: "Handle Route Changes",
					Steps: []string{
						"Remove current assignment first",
						"Wait for confirmation",
						"Create new assignment",
					},
					Warning: "Changes affect student pickups immediately",
				},
			},
		},
		{
			Title:    "Reports & Analytics",
			Icon:     "bi-graph-up",
			Priority: "helpful",
			Items: []ReferenceItem{
				{
					Action: "Generate Monthly Report",
					Steps: []string{
						"Open Report Builder",
						"Select report type",
						"Choose date range",
						"Click 'Generate'",
						"Export as PDF or Excel",
					},
					Shortcut: "Quick Actions → Reports",
				},
				{
					Action: "Export ECSE Data",
					Steps: []string{
						"Navigate to ECSE Dashboard",
						"Click 'Export' button",
						"Select format (Excel/CSV)",
						"Save to computer",
					},
					Note: "Include IEP status for compliance",
				},
			},
		},
		{
			Title:    "Emergency Procedures",
			Icon:     "bi-exclamation-triangle",
			Priority: "critical",
			Items: []ReferenceItem{
				{
					Action: "Bus Breakdown",
					Steps: []string{
						"Contact driver immediately",
						"Arrange replacement bus",
						"Notify affected parents",
						"Update route status",
						"Document incident",
					},
					Warning: "Priority: Student safety first",
				},
				{
					Action: "Driver Absence",
					Steps: []string{
						"Check substitute driver list",
						"Reassign route temporarily",
						"Notify school administration",
						"Update system assignments",
					},
					Note: "Keep substitute list updated",
				},
			},
		},
	}
}

func getManagerShortcuts() []Shortcut {
	return []Shortcut{
		{Keys: "Alt + D", Description: "Go to Dashboard", Context: "global"},
		{Keys: "Alt + F", Description: "Fleet Management", Context: "global"},
		{Keys: "Alt + R", Description: "Assign Routes", Context: "global"},
		{Keys: "Alt + U", Description: "Manage Users", Context: "global"},
		{Keys: "Ctrl + S", Description: "Save Changes", Context: "form"},
		{Keys: "Ctrl + F", Description: "Search Page", Context: "global"},
		{Keys: "Escape", Description: "Cancel/Close Dialog", Context: "global"},
		{Keys: "Tab", Description: "Next Field", Context: "form"},
		{Keys: "Shift + Tab", Description: "Previous Field", Context: "form"},
	}
}

// Driver reference content
func getDriverReferenceSections() []ReferenceSection {
	return []ReferenceSection{
		{
			Title:    "Pre-Trip Checklist",
			Icon:     "bi-clipboard-check",
			Priority: "critical",
			Items: []ReferenceItem{
				{
					Action: "Vehicle Inspection",
					Steps: []string{
						"Check tire condition and pressure",
						"Test all lights (headlights, turn signals, brake lights)",
						"Verify mirrors are clean and adjusted",
						"Check fuel level",
						"Test brakes at low speed",
						"Ensure first aid kit is present",
						"Verify fire extinguisher is accessible",
					},
					Warning: "Do not operate if any safety item fails",
				},
				{
					Action: "System Check-In",
					Steps: []string{
						"Log into Driver Dashboard",
						"Verify correct route assignment",
						"Check student roster",
						"Note any special instructions",
					},
					Note: "Report issues to dispatch immediately",
				},
			},
		},
		{
			Title:    "Daily Route Logging",
			Icon:     "bi-journal-text",
			Priority: "critical",
			Items: []ReferenceItem{
				{
					Action: "Morning Route",
					Steps: []string{
						"Open Driver Dashboard",
						"Click 'Morning Route'",
						"Enter departure time",
						"Record beginning mileage",
						"Mark student attendance",
						"Enter actual pickup times",
						"Record arrival time and ending mileage",
						"Save log",
					},
					Shortcut: "Dashboard → Quick Actions → Morning Route",
				},
				{
					Action: "Afternoon Route",
					Steps: []string{
						"Same as morning route",
						"Pay attention to different drop-off times",
						"Note any parent pickups",
					},
					Warning: "Never leave until all students are accounted for",
				},
			},
		},
		{
			Title:    "Student Management",
			Icon:     "bi-people",
			Priority: "important",
			Items: []ReferenceItem{
				{
					Action: "Take Attendance",
					Steps: []string{
						"Call each student by name",
						"Check the 'Present' box",
						"Enter actual pickup time",
						"Note any absences",
					},
					Note: "Update in real-time during route",
				},
				{
					Action: "Handle No-Shows",
					Steps: []string{
						"Wait 2 minutes at stop",
						"Leave student unchecked",
						"Add note in trip log",
						"Continue route on schedule",
					},
					Warning: "Do not skip stops without waiting",
				},
			},
		},
		{
			Title:    "Emergency Procedures",
			Icon:     "bi-exclamation-triangle",
			Priority: "critical",
			Items: []ReferenceItem{
				{
					Action: "Medical Emergency",
					Steps: []string{
						"Pull over safely",
						"Call 911 immediately",
						"Contact dispatch",
						"Provide first aid if trained",
						"Keep other students calm",
						"Wait for emergency services",
					},
					Warning: "Student safety is top priority",
				},
				{
					Action: "Vehicle Breakdown",
					Steps: []string{
						"Move to safe location",
						"Turn on hazard lights",
						"Contact dispatch",
						"Keep students on bus",
						"Wait for assistance",
						"Complete incident report",
					},
					Note: "Never leave students unattended",
				},
				{
					Action: "Discipline Issues",
					Steps: []string{
						"Use calm voice",
						"Pull over if necessary",
						"Document behavior",
						"Report to supervisor",
						"Follow district policy",
					},
					Warning: "Never use physical restraint",
				},
			},
		},
	}
}

func getDriverShortcuts() []Shortcut {
	return []Shortcut{
		{Keys: "Alt + D", Description: "Go to Dashboard", Context: "global"},
		{Keys: "Alt + S", Description: "Student Management", Context: "global"},
		{Keys: "Alt + M", Description: "Morning Route", Context: "dashboard"},
		{Keys: "Alt + A", Description: "Afternoon Route", Context: "dashboard"},
		{Keys: "Space", Description: "Check/Uncheck Student", Context: "attendance"},
		{Keys: "Tab", Description: "Next Student", Context: "attendance"},
		{Keys: "Ctrl + S", Description: "Save Log", Context: "form"},
		{Keys: "F1", Description: "Help", Context: "global"},
	}
}

// General reference content
func getGeneralReferenceSections() []ReferenceSection {
	return []ReferenceSection{
		{
			Title:    "Getting Help",
			Icon:     "bi-question-circle",
			Priority: "important",
			Items: []ReferenceItem{
				{
					Action: "Access Help Center",
					Steps: []string{
						"Click 'Help' in navigation",
						"Browse categories",
						"Or search for topic",
					},
					Shortcut: "Available on every page",
				},
				{
					Action: "Contact Support",
					Steps: []string{
						"Go to Help Center",
						"Click 'Contact Support'",
						"Fill out request form",
						"Submit ticket",
					},
					Note: "Include screenshots if possible",
				},
			},
		},
		{
			Title:    "Account Management",
			Icon:     "bi-person-circle",
			Priority: "helpful",
			Items: []ReferenceItem{
				{
					Action: "Change Password",
					Steps: []string{
						"Click your username",
						"Select 'Profile'",
						"Click 'Change Password'",
						"Enter current and new password",
						"Save changes",
					},
					Warning: "Use strong passwords (8+ characters)",
				},
			},
		},
	}
}

func getGeneralShortcuts() []Shortcut {
	return []Shortcut{
		{Keys: "F1", Description: "Help", Context: "global"},
		{Keys: "Ctrl + F", Description: "Search", Context: "global"},
		{Keys: "Escape", Description: "Close Dialog", Context: "global"},
	}
}

func getEmergencyContacts() []Contact {
	return []Contact{
		{
			Name:      "Dispatch",
			Role:      "Primary Contact",
			Phone:     "555-0001",
			Email:     "dispatch@schooldistrict.edu",
			Available: "24/7 during school days",
			Priority:  "primary",
		},
		{
			Name:      "Transportation Director",
			Role:      "Management",
			Phone:     "555-0002",
			Email:     "transport.director@schooldistrict.edu",
			Available: "M-F 7AM-5PM",
			Priority:  "primary",
		},
		{
			Name:      "Maintenance Shop",
			Role:      "Vehicle Issues",
			Phone:     "555-0003",
			Email:     "maintenance@schooldistrict.edu",
			Available: "M-F 6AM-4PM",
			Priority:  "secondary",
		},
		{
			Name:      "IT Support",
			Role:      "System Issues",
			Phone:     "555-0004",
			Email:     "it.support@schooldistrict.edu",
			Available: "M-F 8AM-5PM",
			Priority:  "secondary",
		},
	}
}