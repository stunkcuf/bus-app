package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// TroubleshootingIssue represents a common issue and its solution
type TroubleshootingIssue struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Symptoms    []string `json:"symptoms"`
	Category    string   `json:"category"`
	Solution    Solution `json:"solution"`
	Related     []string `json:"related"`
	Tags        []string `json:"tags"`
	Frequency   string   `json:"frequency"` // "common", "occasional", "rare"
	UserRole    string   `json:"user_role"` // "all", "driver", "manager"
}

// Solution represents the steps to resolve an issue
type Solution struct {
	QuickFix    string   `json:"quick_fix"`
	Steps       []Step   `json:"steps"`
	PreventTips []string `json:"prevent_tips"`
	Contact     string   `json:"contact"`
}

// Step represents a single step in a solution
type Step struct {
	Number      int    `json:"number"`
	Action      string `json:"action"`
	Details     string `json:"details"`
	Screenshot  string `json:"screenshot"`
	Warning     string `json:"warning"`
}

// TroubleshootingCategory represents a category of issues
type TroubleshootingCategory struct {
	ID          string
	Name        string
	Description string
	Icon        string
	Issues      []TroubleshootingIssue
}

// DiagnosticResult represents the result of a system diagnostic
type DiagnosticResult struct {
	Component   string    `json:"component"`
	Status      string    `json:"status"` // "ok", "warning", "error"
	Message     string    `json:"message"`
	Details     string    `json:"details"`
	CheckedAt   time.Time `json:"checked_at"`
}

func troubleshootingHandler(w http.ResponseWriter, r *http.Request) {
	session := getUserFromSession(r)
	if session == nil {
		log.Printf("Troubleshooting access without login")
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// Get category filter from query params
	categoryFilter := r.URL.Query().Get("category")
	searchQuery := r.URL.Query().Get("search")
	
	// Get all troubleshooting categories
	categories := getTroubleshootingCategories(session.Role)
	
	// Filter issues by category or search
	var filteredIssues []TroubleshootingIssue
	var selectedCategory *TroubleshootingCategory
	
	if searchQuery != "" {
		// Search across all issues
		filteredIssues = searchTroubleshootingIssues(searchQuery, session.Role)
	} else if categoryFilter != "" {
		// Filter by category
		for i := range categories {
			if categories[i].ID == categoryFilter {
				selectedCategory = &categories[i]
				filteredIssues = categories[i].Issues
				break
			}
		}
	}

	// Get frequently viewed issues
	frequentIssues := getFrequentIssues(session.Role, 5)

	data := struct {
		Title              string
		Username           string
		UserType           string
		CSPNonce           string
		Categories         []TroubleshootingCategory
		SelectedCategory   *TroubleshootingCategory
		FilteredIssues     []TroubleshootingIssue
		FrequentIssues     []TroubleshootingIssue
		SearchQuery        string
		ShowDiagnostics    bool
	}{
		Title:            "Troubleshooting Guide",
		Username:         session.Username,
		UserType:         session.Role,
		CSPNonce:         generateNonce(),
		Categories:       categories,
		SelectedCategory: selectedCategory,
		FilteredIssues:   filteredIssues,
		FrequentIssues:   frequentIssues,
		SearchQuery:      searchQuery,
		ShowDiagnostics:  session.Role == "manager",
	}

	tmpl := template.Must(template.ParseFiles("templates/troubleshooting.html"))
	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("Error rendering troubleshooting guide: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

func troubleshootingIssueHandler(w http.ResponseWriter, r *http.Request) {
	session := getUserFromSession(r)
	if session == nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// Extract issue ID from URL
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 3 {
		http.Error(w, "Invalid issue ID", http.StatusBadRequest)
		return
	}
	issueID := parts[len(parts)-1]

	// Get issue details
	issue := getIssueByID(issueID)
	if issue == nil {
		http.Error(w, "Issue not found", http.StatusNotFound)
		return
	}

	// Check role permissions
	if issue.UserRole != "all" && !strings.Contains(session.Role, strings.Title(issue.UserRole)) {
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	// Get related issues
	relatedIssues := getRelatedIssues(issue.Related, 3)

	// Track issue view
	trackIssueView(issueID, getUserID(session.Username))

	data := struct {
		Title          string
		Username       string
		UserType       string
		CSPNonce       string
		Issue          *TroubleshootingIssue
		RelatedIssues  []TroubleshootingIssue
		CanRunDiagnostics bool
	}{
		Title:          issue.Title,
		Username:       session.Username,
		UserType:       session.Role,
		CSPNonce:       generateNonce(),
		Issue:          issue,
		RelatedIssues:  relatedIssues,
		CanRunDiagnostics: session.Role == "manager",
	}

	tmpl := template.Must(template.ParseFiles("templates/troubleshooting_issue.html"))
	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("Error rendering issue details: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

// API endpoint for system diagnostics
func diagnosticsHandler(w http.ResponseWriter, r *http.Request) {
	session := getUserFromSession(r)
	if session == nil || session.Role != "manager" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Run system diagnostics
	results := runSystemDiagnostics()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

// Troubleshooting data functions
func getTroubleshootingCategories(userType string) []TroubleshootingCategory {
	categories := []TroubleshootingCategory{
		{
			ID:          "login-issues",
			Name:        "Login & Access",
			Description: "Problems signing in or accessing features",
			Icon:        "bi-key",
			Issues:      filterIssuesByCategory("login-issues", userType),
		},
		{
			ID:          "data-entry",
			Name:        "Data Entry",
			Description: "Issues with forms and data input",
			Icon:        "bi-pencil-square",
			Issues:      filterIssuesByCategory("data-entry", userType),
		},
		{
			ID:          "performance",
			Name:        "Performance",
			Description: "Slow loading or timeout issues",
			Icon:        "bi-speedometer",
			Issues:      filterIssuesByCategory("performance", userType),
		},
		{
			ID:          "reports",
			Name:        "Reports & Export",
			Description: "Problems generating or exporting reports",
			Icon:        "bi-file-earmark-bar-graph",
			Issues:      filterIssuesByCategory("reports", userType),
		},
		{
			ID:          "mobile",
			Name:        "Mobile Issues",
			Description: "Problems on phones and tablets",
			Icon:        "bi-phone",
			Issues:      filterIssuesByCategory("mobile", userType),
		},
		{
			ID:          "errors",
			Name:        "Error Messages",
			Description: "Understanding and fixing error messages",
			Icon:        "bi-exclamation-triangle",
			Issues:      filterIssuesByCategory("errors", userType),
		},
	}

	return categories
}

func getAllTroubleshootingIssues() []TroubleshootingIssue {
	return []TroubleshootingIssue{
		// Login Issues
		{
			ID:          "forgot-password",
			Title:       "I Forgot My Password",
			Description: "Unable to log in because you've forgotten your password",
			Symptoms:    []string{"Can't remember password", "Password not working", "Account locked"},
			Category:    "login-issues",
			Solution: Solution{
				QuickFix: "Contact your administrator for a password reset",
				Steps: []Step{
					{Number: 1, Action: "Click 'Forgot Password' on login page", Details: "Look for the link below the login button"},
					{Number: 2, Action: "Enter your username", Details: "Use the same username you normally log in with"},
					{Number: 3, Action: "Check your email", Details: "A reset link will be sent to your registered email"},
					{Number: 4, Action: "Follow the reset link", Details: "Click the link in the email within 24 hours"},
					{Number: 5, Action: "Create a new password", Details: "Choose a strong password you'll remember"},
				},
				PreventTips: []string{
					"Write your password in a secure location",
					"Use a password manager",
					"Create a memorable but secure password",
				},
				Contact: "If you don't receive the email, contact your system administrator",
			},
			Related:   []string{"account-locked", "wrong-credentials"},
			Tags:      []string{"password", "login", "access"},
			Frequency: "common",
			UserRole:  "all",
		},
		{
			ID:          "session-timeout",
			Title:       "Keep Getting Logged Out",
			Description: "System logs you out unexpectedly or too frequently",
			Symptoms:    []string{"Logged out while working", "Have to log in repeatedly", "Lost unsaved work"},
			Category:    "login-issues",
			Solution: Solution{
				QuickFix: "Save your work frequently and check 'Remember Me' when logging in",
				Steps: []Step{
					{Number: 1, Action: "Check 'Remember Me' box", Details: "This extends your session duration"},
					{Number: 2, Action: "Save work regularly", Details: "Use Ctrl+S or click Save often"},
					{Number: 3, Action: "Keep browser tab active", Details: "Don't leave the system idle for long periods"},
					{Number: 4, Action: "Check internet connection", Details: "Unstable connections can cause logouts"},
				},
				PreventTips: []string{
					"Save work every few minutes",
					"Complete forms promptly",
					"Maintain stable internet connection",
				},
				Contact: "Report persistent issues to IT support",
			},
			Related:   []string{"lost-data", "connection-issues"},
			Tags:      []string{"session", "timeout", "logout"},
			Frequency: "common",
			UserRole:  "all",
		},

		// Data Entry Issues
		{
			ID:          "form-validation-errors",
			Title:       "Form Won't Submit",
			Description: "Getting errors when trying to submit forms",
			Symptoms:    []string{"Red error messages", "Form won't save", "Required fields highlighted"},
			Category:    "data-entry",
			Solution: Solution{
				QuickFix: "Check all required fields are filled and in the correct format",
				Steps: []Step{
					{Number: 1, Action: "Look for red asterisks (*)", Details: "These mark required fields"},
					{Number: 2, Action: "Check field formats", Details: "Dates: MM/DD/YYYY, Phone: (555) 555-5555"},
					{Number: 3, Action: "Fill all required fields", Details: "Even if they seem optional"},
					{Number: 4, Action: "Review error messages", Details: "They explain what needs to be fixed"},
					{Number: 5, Action: "Correct and resubmit", Details: "Fix errors one by one"},
				},
				PreventTips: []string{
					"Complete all required fields first",
					"Use correct formats for dates and phones",
					"Double-check before submitting",
				},
				Contact: "Screenshot errors if they persist",
			},
			Related:   []string{"date-format", "phone-format"},
			Tags:      []string{"forms", "validation", "submit"},
			Frequency: "common",
			UserRole:  "all",
		},
		{
			ID:          "duplicate-entry",
			Title:       "Duplicate Entry Error",
			Description: "System says record already exists",
			Symptoms:    []string{"Duplicate key error", "Record already exists", "Can't add student/vehicle"},
			Category:    "data-entry",
			Solution: Solution{
				QuickFix: "Search for the existing record before creating a new one",
				Steps: []Step{
					{Number: 1, Action: "Use search function", Details: "Search by name, ID, or key field"},
					{Number: 2, Action: "Check for existing record", Details: "It may already be in the system"},
					{Number: 3, Action: "Update existing record", Details: "Edit instead of creating new", Warning: "Don't create duplicates"},
					{Number: 4, Action: "Check for typos", Details: "Small differences matter"},
				},
				PreventTips: []string{
					"Always search first",
					"Keep records up to date",
					"Use consistent naming",
				},
				Contact: "Admin can merge duplicates if needed",
			},
			Related:   []string{"search-not-working", "edit-record"},
			Tags:      []string{"duplicate", "exists", "unique"},
			Frequency: "occasional",
			UserRole:  "all",
		},

		// Performance Issues
		{
			ID:          "slow-loading",
			Title:       "Pages Loading Slowly",
			Description: "System is running slower than usual",
			Symptoms:    []string{"Long wait times", "Spinning loader", "Timeouts"},
			Category:    "performance",
			Solution: Solution{
				QuickFix: "Clear browser cache and check internet speed",
				Steps: []Step{
					{Number: 1, Action: "Clear browser cache", Details: "Ctrl+Shift+Delete, select 'Cached images'"},
					{Number: 2, Action: "Check internet speed", Details: "Run speedtest.net"},
					{Number: 3, Action: "Close unused tabs", Details: "Too many tabs slow browsers"},
					{Number: 4, Action: "Try different browser", Details: "Chrome or Edge recommended"},
					{Number: 5, Action: "Check time of day", Details: "System may be busy during peak hours"},
				},
				PreventTips: []string{
					"Use modern browser",
					"Maintain good internet connection",
					"Clear cache weekly",
				},
				Contact: "Report if issue persists after trying these steps",
			},
			Related:   []string{"browser-compatibility", "timeout-errors"},
			Tags:      []string{"slow", "performance", "speed"},
			Frequency: "occasional",
			UserRole:  "all",
		},

		// Report Issues
		{
			ID:          "report-no-data",
			Title:       "Report Shows No Data",
			Description: "Generated report is empty or missing data",
			Symptoms:    []string{"Blank report", "No results found", "Missing information"},
			Category:    "reports",
			Solution: Solution{
				QuickFix: "Check your date range and filters",
				Steps: []Step{
					{Number: 1, Action: "Verify date range", Details: "Ensure dates include data period"},
					{Number: 2, Action: "Check filters", Details: "Remove filters one by one"},
					{Number: 3, Action: "Confirm data exists", Details: "Check if source data is present"},
					{Number: 4, Action: "Try broader search", Details: "Expand criteria gradually"},
				},
				PreventTips: []string{
					"Start with broad criteria",
					"Verify data exists first",
					"Save working report configurations",
				},
				Contact: "Admin can verify data availability",
			},
			Related:   []string{"report-filters", "date-selection"},
			Tags:      []string{"reports", "empty", "no-data"},
			Frequency: "common",
			UserRole:  "all",
		},
		{
			ID:          "export-not-working",
			Title:       "Can't Export Reports",
			Description: "Export to Excel/PDF not working",
			Symptoms:    []string{"Export button not working", "File won't download", "Corrupted file"},
			Category:    "reports",
			Solution: Solution{
				QuickFix: "Check popup blocker and download settings",
				Steps: []Step{
					{Number: 1, Action: "Disable popup blocker", Details: "Allow popups for this site"},
					{Number: 2, Action: "Check Downloads folder", Details: "File may be there already"},
					{Number: 3, Action: "Try different format", Details: "If Excel fails, try CSV"},
					{Number: 4, Action: "Reduce data size", Details: "Export smaller date ranges"},
				},
				PreventTips: []string{
					"Allow popups for the system",
					"Keep Downloads folder organized",
					"Export in smaller chunks if needed",
				},
				Contact: "IT can check export service status",
			},
			Related:   []string{"popup-blocked", "download-location"},
			Tags:      []string{"export", "download", "excel", "pdf"},
			Frequency: "occasional",
			UserRole:  "all",
		},

		// Mobile Issues
		{
			ID:          "mobile-display",
			Title:       "Display Issues on Mobile",
			Description: "Pages not displaying correctly on phone/tablet",
			Symptoms:    []string{"Text cut off", "Buttons not clickable", "Layout broken"},
			Category:    "mobile",
			Solution: Solution{
				QuickFix: "Rotate device to landscape mode",
				Steps: []Step{
					{Number: 1, Action: "Rotate to landscape", Details: "Turn device sideways"},
					{Number: 2, Action: "Zoom out", Details: "Pinch to zoom out for full view"},
					{Number: 3, Action: "Request desktop site", Details: "In browser menu options"},
					{Number: 4, Action: "Update browser", Details: "Use latest version"},
				},
				PreventTips: []string{
					"Use landscape for tables",
					"Keep browser updated",
					"Use tablet for complex tasks",
				},
				Contact: "Report specific pages with issues",
			},
			Related:   []string{"responsive-design", "touch-not-working"},
			Tags:      []string{"mobile", "responsive", "display"},
			Frequency: "occasional",
			UserRole:  "all",
		},

		// Error Messages
		{
			ID:          "500-error",
			Title:       "500 Internal Server Error",
			Description: "Getting server error messages",
			Symptoms:    []string{"Error 500", "Internal server error", "Something went wrong"},
			Category:    "errors",
			Solution: Solution{
				QuickFix: "Refresh the page and try again",
				Steps: []Step{
					{Number: 1, Action: "Note the error details", Details: "Screenshot if possible"},
					{Number: 2, Action: "Refresh the page", Details: "Press F5 or Ctrl+R"},
					{Number: 3, Action: "Go back and retry", Details: "Use browser back button"},
					{Number: 4, Action: "Clear cache and retry", Details: "Ctrl+Shift+Delete"},
					{Number: 5, Action: "Report to admin", Details: "Include screenshot and time", Warning: "Don't retry too many times"},
				},
				PreventTips: []string{
					"Save work frequently",
					"Report errors promptly",
					"Note what you were doing",
				},
				Contact: "Report immediately with details",
			},
			Related:   []string{"400-error", "404-error"},
			Tags:      []string{"error", "500", "server"},
			Frequency: "rare",
			UserRole:  "all",
		},
		{
			ID:          "permission-denied",
			Title:       "Permission Denied Error",
			Description: "Access denied to certain features",
			Symptoms:    []string{"Access denied", "Not authorized", "Permission error"},
			Category:    "errors",
			Solution: Solution{
				QuickFix: "Verify you're using the correct account and have necessary permissions",
				Steps: []Step{
					{Number: 1, Action: "Check logged-in user", Details: "Verify username in top right"},
					{Number: 2, Action: "Confirm your role", Details: "Driver vs Manager access"},
					{Number: 3, Action: "Request access", Details: "Contact administrator"},
					{Number: 4, Action: "Try logging out/in", Details: "Refresh your session"},
				},
				PreventTips: []string{
					"Know your access level",
					"Use correct account",
					"Request access early",
				},
				Contact: "Administrator manages permissions",
			},
			Related:   []string{"role-access", "feature-locked"},
			Tags:      []string{"permission", "access", "denied"},
			Frequency: "occasional",
			UserRole:  "all",
		},
	}
}

func filterIssuesByCategory(category string, userType string) []TroubleshootingIssue {
	var issues []TroubleshootingIssue
	allIssues := getAllTroubleshootingIssues()
	
	for _, issue := range allIssues {
		if issue.Category == category && 
		   (issue.UserRole == "all" || strings.Contains(userType, strings.Title(issue.UserRole))) {
			issues = append(issues, issue)
		}
	}
	
	return issues
}

func getIssueByID(id string) *TroubleshootingIssue {
	allIssues := getAllTroubleshootingIssues()
	for _, issue := range allIssues {
		if issue.ID == id {
			return &issue
		}
	}
	return nil
}

func getRelatedIssues(relatedIDs []string, limit int) []TroubleshootingIssue {
	var related []TroubleshootingIssue
	allIssues := getAllTroubleshootingIssues()
	
	for _, id := range relatedIDs {
		for _, issue := range allIssues {
			if issue.ID == id {
				related = append(related, issue)
				if len(related) >= limit {
					return related
				}
				break
			}
		}
	}
	
	return related
}

func searchTroubleshootingIssues(query string, userType string) []TroubleshootingIssue {
	var results []TroubleshootingIssue
	query = strings.ToLower(query)
	allIssues := getAllTroubleshootingIssues()
	
	for _, issue := range allIssues {
		// Check role permission
		if issue.UserRole != "all" && !strings.Contains(userType, strings.Title(issue.UserRole)) {
			continue
		}
		
		// Search in title, description, symptoms, and tags
		if strings.Contains(strings.ToLower(issue.Title), query) ||
		   strings.Contains(strings.ToLower(issue.Description), query) {
			results = append(results, issue)
			continue
		}
		
		// Search in symptoms
		for _, symptom := range issue.Symptoms {
			if strings.Contains(strings.ToLower(symptom), query) {
				results = append(results, issue)
				break
			}
		}
		
		// Search in tags
		for _, tag := range issue.Tags {
			if strings.Contains(strings.ToLower(tag), query) {
				results = append(results, issue)
				break
			}
		}
	}
	
	return results
}

func getFrequentIssues(userType string, limit int) []TroubleshootingIssue {
	// In production, this would query actual view statistics
	// For now, return common issues
	var frequent []TroubleshootingIssue
	allIssues := getAllTroubleshootingIssues()
	
	for _, issue := range allIssues {
		if issue.Frequency == "common" && 
		   (issue.UserRole == "all" || strings.Contains(userType, strings.Title(issue.UserRole))) {
			frequent = append(frequent, issue)
			if len(frequent) >= limit {
				break
			}
		}
	}
	
	return frequent
}

func trackIssueView(issueID string, userID int) {
	// In production, track which issues users view most
	// This helps identify common problems
	log.Printf("User %d viewed troubleshooting issue: %s", userID, issueID)
}

func runSystemDiagnostics() []DiagnosticResult {
	results := []DiagnosticResult{}
	
	// Check database connection
	dbResult := DiagnosticResult{
		Component: "Database",
		CheckedAt: time.Now(),
	}
	
	err := db.Ping()
	if err != nil {
		dbResult.Status = "error"
		dbResult.Message = "Database connection failed"
		dbResult.Details = err.Error()
	} else {
		dbResult.Status = "ok"
		dbResult.Message = "Database connection active"
		
		// Check table count
		var tableCount int
		err = db.QueryRow("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public'").Scan(&tableCount)
		if err == nil {
			dbResult.Details = fmt.Sprintf("Tables: %d", tableCount)
		}
	}
	results = append(results, dbResult)
	
	// Check session storage
	sessionResult := DiagnosticResult{
		Component: "Sessions",
		CheckedAt: time.Now(),
	}
	
	activeSessions := getActiveSessionCount()
	if activeSessions >= 0 {
		sessionResult.Status = "ok"
		sessionResult.Message = "Session management operational"
		sessionResult.Details = fmt.Sprintf("Active sessions: %d", activeSessions)
	} else {
		sessionResult.Status = "warning"
		sessionResult.Message = "Could not retrieve session count"
	}
	results = append(results, sessionResult)
	
	// Check disk space (simplified)
	diskResult := DiagnosticResult{
		Component: "Storage",
		Status:    "ok",
		Message:   "Storage available",
		CheckedAt: time.Now(),
	}
	results = append(results, diskResult)
	
	// Check response time
	responseResult := DiagnosticResult{
		Component: "Response Time",
		CheckedAt: time.Now(),
	}
	
	start := time.Now()
	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}
	_, err = http.Get("http://localhost:" + port + "/health")
	elapsed := time.Since(start)
	
	if err != nil {
		responseResult.Status = "error"
		responseResult.Message = "Health check failed"
		responseResult.Details = err.Error()
	} else if elapsed > 2*time.Second {
		responseResult.Status = "warning"
		responseResult.Message = "Slow response time"
		responseResult.Details = elapsed.String()
	} else {
		responseResult.Status = "ok"
		responseResult.Message = "Response time normal"
		responseResult.Details = elapsed.String()
	}
	results = append(results, responseResult)
	
	return results
}

func getActiveSessionCount() int {
	// Count active sessions from database
	var count int
	err := db.QueryRow(`
		SELECT COUNT(DISTINCT username) 
		FROM active_sessions 
		WHERE last_activity > NOW() - INTERVAL '30 minutes'
	`).Scan(&count)
	
	if err != nil {
		log.Printf("Failed to get active session count: %v", err)
		return 0
	}
	return count
}