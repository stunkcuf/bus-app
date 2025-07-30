package main

import (
	"html/template"
	"log"
	"net/http"
	"strings"
)

// Chapter represents a chapter in the user manual
type Chapter struct {
	ID          string
	Title       string
	Icon        string
	Description string
	Sections    []Section
	Order       int
}

// Section represents a section within a chapter
type Section struct {
	ID          string
	Title       string
	Content     template.HTML
	Screenshots []Screenshot
	Tips        []string
	Warnings    []string
}

// Screenshot represents an image in the manual
type Screenshot struct {
	URL     string
	Caption string
	Alt     string
}

func userManualHandler(w http.ResponseWriter, r *http.Request) {
	session := getUserFromSession(r)
	if session == nil {
		log.Printf("User manual access without login")
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// Get chapter from URL
	chapterID := r.URL.Query().Get("chapter")
	
	// Get all chapters
	chapters := getAllChapters(session.Role)
	
	// Find selected chapter
	var selectedChapter *Chapter
	if chapterID != "" {
		for _, ch := range chapters {
			if ch.ID == chapterID {
				selectedChapter = &ch
				break
			}
		}
	}
	
	// Default to first chapter if none selected
	if selectedChapter == nil && len(chapters) > 0 {
		selectedChapter = &chapters[0]
	}

	data := struct {
		Title           string
		Username        string
		UserType        string
		CSPNonce        string
		Chapters        []Chapter
		SelectedChapter *Chapter
		TableOfContents []Chapter
		SearchEnabled   bool
	}{
		Title:           "User Manual",
		Username:        session.Username,
		UserType:        session.Role,
		CSPNonce:        generateNonce(),
		Chapters:        chapters,
		SelectedChapter: selectedChapter,
		TableOfContents: chapters,
		SearchEnabled:   true,
	}

	tmpl := template.Must(template.ParseFiles("templates/user_manual.html"))
	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("Error rendering user manual: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

func getAllChapters(userType string) []Chapter {
	chapters := []Chapter{
		{
			ID:          "getting-started",
			Title:       "Getting Started",
			Icon:        "bi-rocket-takeoff",
			Description: "Learn the basics of the Fleet Management System",
			Order:       1,
			Sections: []Section{
				{
					ID:    "first-login",
					Title: "Your First Login",
					Content: template.HTML(`
						<p>Welcome to the Fleet Management System! This guide will help you get started with your first login.</p>
						
						<h4>Login Steps:</h4>
						<ol>
							<li><strong>Navigate to the Login Page</strong>
								<p>Open your web browser and go to the system URL provided by your administrator.</p>
							</li>
							<li><strong>Enter Your Credentials</strong>
								<p>Type your username and password. These should have been provided by your manager.</p>
							</li>
							<li><strong>Click "Login"</strong>
								<p>After entering your credentials, click the blue "Login" button.</p>
							</li>
							<li><strong>Dashboard Access</strong>
								<p>You'll be automatically directed to your role-specific dashboard (Manager or Driver).</p>
							</li>
						</ol>
						
						<div class="alert alert-info mt-3">
							<i class="bi bi-info-circle me-2"></i>
							<strong>First Time Users:</strong> You'll see a welcome tour prompt. We recommend taking the tour to learn about key features.
						</div>
					`),
					Tips: []string{
						"Bookmark the login page for quick access",
						"Use a password manager to securely store your credentials",
						"Enable 'Remember Me' for convenience (on personal devices only)",
					},
					Warnings: []string{
						"Never share your login credentials with anyone",
						"Always log out when using shared computers",
					},
				},
				{
					ID:    "navigation",
					Title: "Navigating the System",
					Content: template.HTML(`
						<p>The Fleet Management System uses an intuitive navigation structure designed for ease of use.</p>
						
						<h4>Main Navigation Elements:</h4>
						
						<h5>1. Top Navigation Bar</h5>
						<ul>
							<li><strong>Logo/Home:</strong> Click to return to your dashboard</li>
							<li><strong>Getting Started:</strong> Access role-specific guides</li>
							<li><strong>Help:</strong> Open the help center</li>
							<li><strong>User Menu:</strong> Access profile, progress, and settings</li>
							<li><strong>Logout:</strong> Securely exit the system</li>
						</ul>
						
						<h5>2. Dashboard Cards</h5>
						<p>Your dashboard displays card-based navigation for quick access to features:</p>
						<ul>
							<li>Click any card to navigate to that feature</li>
							<li>Cards show relevant statistics and status</li>
							<li>Color coding indicates urgency or status</li>
						</ul>
						
						<h5>3. Breadcrumb Navigation</h5>
						<p>Shows your current location and allows easy backtracking:</p>
						<ul>
							<li>Located below the main navigation</li>
							<li>Click any breadcrumb to go back to that level</li>
						</ul>
					`),
					Tips: []string{
						"Use keyboard shortcuts (Alt + key) for faster navigation",
						"The dashboard always shows your most important information first",
						"Hover over icons to see tooltips explaining their function",
					},
				},
				{
					ID:    "user-interface",
					Title: "Understanding the Interface",
					Content: template.HTML(`
						<p>The system uses a modern, accessible interface designed for users of all technical levels.</p>
						
						<h4>Interface Elements:</h4>
						
						<h5>Color Coding System</h5>
						<ul>
							<li><span class="badge bg-success">Green</span> - Active, completed, or good status</li>
							<li><span class="badge bg-warning">Yellow</span> - Warning, pending, or needs attention</li>
							<li><span class="badge bg-danger">Red</span> - Error, inactive, or urgent</li>
							<li><span class="badge bg-info">Blue</span> - Informational or neutral</li>
						</ul>
						
						<h5>Common Icons</h5>
						<ul>
							<li><i class="bi bi-bus-front"></i> Bus/Vehicle related</li>
							<li><i class="bi bi-people"></i> Student/Person related</li>
							<li><i class="bi bi-diagram-3"></i> Route related</li>
							<li><i class="bi bi-tools"></i> Maintenance related</li>
							<li><i class="bi bi-calendar"></i> Schedule/Date related</li>
							<li><i class="bi bi-graph-up"></i> Reports/Analytics</li>
						</ul>
						
						<h5>Form Elements</h5>
						<ul>
							<li><strong>Required Fields:</strong> Marked with a red asterisk (*)</li>
							<li><strong>Help Icons:</strong> Click <i class="bi bi-question-circle"></i> for field-specific help</li>
							<li><strong>Validation:</strong> Real-time feedback on input errors</li>
							<li><strong>Auto-save:</strong> Forms save progress automatically</li>
						</ul>
					`),
					Tips: []string{
						"Dark backgrounds with white text are used throughout for better readability",
						"All clickable elements change appearance on hover",
						"Loading indicators show when actions are processing",
					},
				},
			},
		},
		{
			ID:          "daily-operations",
			Title:       "Daily Operations",
			Icon:        "bi-calendar-check",
			Description: "Learn how to perform your daily tasks efficiently",
			Order:       2,
			Sections: []Section{
				{
					ID:    "driver-daily-tasks",
					Title: "Driver Daily Tasks",
					Content: template.HTML(`
						<p>As a driver, your daily tasks revolve around safe transportation of students and accurate record-keeping.</p>
						
						<h4>Morning Routine</h4>
						
						<h5>1. Pre-Trip Inspection</h5>
						<ol>
							<li>Walk around the bus checking:
								<ul>
									<li>Tires for proper inflation and wear</li>
									<li>All lights (headlights, brake lights, turn signals)</li>
									<li>Mirrors for proper adjustment</li>
									<li>Emergency exits functionality</li>
									<li>First aid kit and fire extinguisher presence</li>
								</ul>
							</li>
							<li>Start the engine and check:
								<ul>
									<li>Fuel level</li>
									<li>Oil pressure</li>
									<li>Brake operation</li>
									<li>Warning lights</li>
								</ul>
							</li>
						</ol>
						
						<h5>2. System Check-In</h5>
						<ol>
							<li>Log into the Fleet Management System</li>
							<li>Navigate to your Driver Dashboard</li>
							<li>Click "Morning Route" quick action</li>
							<li>Verify your route assignment and bus number</li>
							<li>Review student roster for any changes</li>
						</ol>
						
						<h5>3. Route Logging</h5>
						<ol>
							<li>Record departure time when leaving the depot</li>
							<li>Enter beginning odometer reading</li>
							<li>At each stop:
								<ul>
									<li>Mark students present/absent</li>
									<li>Record actual pickup time if different from scheduled</li>
									<li>Note any issues or parent messages</li>
								</ul>
							</li>
							<li>Upon arrival at school:
								<ul>
									<li>Record arrival time</li>
									<li>Enter ending odometer reading</li>
									<li>Add any route notes</li>
									<li>Submit the log</li>
								</ul>
							</li>
						</ol>
					`),
					Tips: []string{
						"Use the mobile-friendly interface to update attendance in real-time",
						"Pre-fill common values to save time",
						"Add detailed notes for substitute drivers",
					},
					Warnings: []string{
						"Never skip the pre-trip inspection",
						"Always verify student identity before allowing boarding",
						"Report any mechanical issues immediately",
					},
				},
				{
					ID:    "manager-daily-tasks",
					Title: "Manager Daily Tasks",
					Content: template.HTML(`
						<p>Managers oversee the entire fleet operation, ensuring smooth daily operations and compliance.</p>
						
						<h4>Daily Management Workflow</h4>
						
						<h5>1. Morning Review (6:00 AM - 7:00 AM)</h5>
						<ol>
							<li><strong>Check Driver Attendance</strong>
								<ul>
									<li>Review driver check-ins on dashboard</li>
									<li>Identify any absent drivers</li>
									<li>Assign substitute drivers if needed</li>
								</ul>
							</li>
							<li><strong>Fleet Status Check</strong>
								<ul>
									<li>Verify all buses are operational</li>
									<li>Check for maintenance alerts</li>
									<li>Review fuel levels</li>
								</ul>
							</li>
							<li><strong>Route Coverage</strong>
								<ul>
									<li>Ensure all routes have assigned drivers</li>
									<li>Check for any route modifications</li>
									<li>Communicate changes to affected drivers</li>
								</ul>
							</li>
						</ol>
						
						<h5>2. Ongoing Monitoring (Throughout Day)</h5>
						<ul>
							<li><strong>Real-time Tracking:</strong> Monitor bus locations (if GPS enabled)</li>
							<li><strong>Communication Hub:</strong> Handle driver calls and parent inquiries</li>
							<li><strong>Issue Resolution:</strong> Address problems as they arise</li>
							<li><strong>Documentation:</strong> Log all incidents and resolutions</li>
						</ul>
						
						<h5>3. End-of-Day Tasks</h5>
						<ol>
							<li><strong>Trip Log Review</strong>
								<ul>
									<li>Verify all drivers submitted logs</li>
									<li>Check for missing data</li>
									<li>Review any reported issues</li>
								</ul>
							</li>
							<li><strong>Next Day Preparation</strong>
								<ul>
									<li>Review tomorrow's driver assignments</li>
									<li>Check for scheduled maintenance</li>
									<li>Prepare substitute list if needed</li>
								</ul>
							</li>
							<li><strong>Reporting</strong>
								<ul>
									<li>Generate daily summary report</li>
									<li>Email updates to administration</li>
									<li>Update maintenance schedules</li>
								</ul>
							</li>
						</ol>
					`),
					Tips: []string{
						"Set up dashboard widgets for at-a-glance monitoring",
						"Use the activity feed to track all system changes",
						"Create recurring reports for regular updates",
					},
				},
			},
		},
		{
			ID:          "fleet-management",
			Title:       "Fleet Management",
			Icon:        "bi-bus-front",
			Description: "Managing your vehicle fleet effectively",
			Order:       3,
			Sections: []Section{
				{
					ID:    "vehicle-management",
					Title: "Vehicle Management",
					Content: template.HTML(`
						<p>Proper fleet management ensures safe, reliable transportation for students.</p>
						
						<h4>Managing Your Fleet</h4>
						
						<h5>Adding a New Vehicle</h5>
						<ol>
							<li>Navigate to Fleet Management from the dashboard</li>
							<li>Click "Add New Vehicle" button</li>
							<li>Enter vehicle information:
								<ul>
									<li>Bus/Vehicle number (unique identifier)</li>
									<li>Make and model</li>
									<li>Year of manufacture</li>
									<li>License plate number</li>
									<li>Seating capacity</li>
									<li>Current mileage</li>
									<li>Fuel type</li>
								</ul>
							</li>
							<li>Set initial status (Active/Maintenance/Out of Service)</li>
							<li>Upload relevant documents (registration, insurance)</li>
							<li>Save the vehicle record</li>
						</ol>
						
						<h5>Vehicle Status Management</h5>
						<p>Keep vehicle status updated for accurate fleet availability:</p>
						<ul>
							<li><strong>Active:</strong> Available for daily routes</li>
							<li><strong>Maintenance:</strong> Undergoing scheduled or repair maintenance</li>
							<li><strong>Out of Service:</strong> Not available for use</li>
							<li><strong>Reserve:</strong> Backup vehicle for emergencies</li>
						</ul>
						
						<h5>Fleet Overview Features</h5>
						<ul>
							<li><strong>Status Dashboard:</strong> Visual representation of fleet status</li>
							<li><strong>Quick Filters:</strong> Filter by status, type, or assignment</li>
							<li><strong>Bulk Actions:</strong> Update multiple vehicles at once</li>
							<li><strong>Export Options:</strong> Download fleet data for reports</li>
						</ul>
					`),
					Tips: []string{
						"Keep photos of each vehicle for easy identification",
						"Set up maintenance reminders based on mileage or time",
						"Document all modifications or equipment additions",
					},
				},
				{
					ID:    "maintenance-tracking",
					Title: "Maintenance Tracking",
					Content: template.HTML(`
						<p>Regular maintenance is crucial for safety and compliance. The system helps track all maintenance activities.</p>
						
						<h4>Maintenance Management</h4>
						
						<h5>Scheduling Maintenance</h5>
						<ol>
							<li>Go to Maintenance Records from the dashboard</li>
							<li>Select the vehicle needing maintenance</li>
							<li>Click "Schedule Maintenance"</li>
							<li>Choose maintenance type:
								<ul>
									<li>Oil change</li>
									<li>Tire rotation/replacement</li>
									<li>Brake inspection</li>
									<li>Annual inspection</li>
									<li>Repair work</li>
									<li>Other (specify)</li>
								</ul>
							</li>
							<li>Set the scheduled date</li>
							<li>Add notes for the maintenance team</li>
							<li>Save the maintenance schedule</li>
						</ol>
						
						<h5>Recording Completed Maintenance</h5>
						<ol>
							<li>Locate the scheduled maintenance item</li>
							<li>Click "Mark Complete"</li>
							<li>Enter completion details:
								<ul>
									<li>Actual completion date</li>
									<li>Work performed description</li>
									<li>Parts replaced</li>
									<li>Labor hours</li>
									<li>Total cost</li>
									<li>Technician name</li>
								</ul>
							</li>
							<li>Upload invoices or work orders</li>
							<li>Set next service date if recurring</li>
							<li>Save the record</li>
						</ol>
						
						<h5>Maintenance Alerts</h5>
						<p>The system automatically generates alerts for:</p>
						<ul>
							<li>Overdue maintenance items</li>
							<li>Upcoming scheduled maintenance (7-day warning)</li>
							<li>Mileage-based service intervals</li>
							<li>Inspection expiration dates</li>
							<li>Warranty expiration notices</li>
						</ul>
					`),
					Tips: []string{
						"Take before/after photos for major repairs",
						"Keep digital copies of all maintenance receipts",
						"Set up recurring maintenance schedules for routine items",
					},
					Warnings: []string{
						"Never delay safety-critical maintenance",
						"Document all maintenance for compliance audits",
					},
				},
			},
		},
		{
			ID:          "student-management",
			Title:       "Student Management",
			Icon:        "bi-people",
			Description: "Managing student information and transportation needs",
			Order:       4,
			Sections: []Section{
				{
					ID:    "student-roster",
					Title: "Managing Student Rosters",
					Content: template.HTML(`
						<p>Accurate student information ensures safe and efficient transportation.</p>
						
						<h4>Student Information Management</h4>
						
						<h5>Adding a New Student</h5>
						<ol>
							<li>Navigate to Students section</li>
							<li>Click "Add Student" button</li>
							<li>Enter student information:
								<ul>
									<li>Full name (Last, First Middle)</li>
									<li>Grade level</li>
									<li>School attending</li>
									<li>Home address</li>
									<li>Pickup address (if different)</li>
									<li>Dropoff address (if different)</li>
								</ul>
							</li>
							<li>Add guardian information:
								<ul>
									<li>Primary guardian name</li>
									<li>Relationship to student</li>
									<li>Primary phone number</li>
									<li>Emergency phone number</li>
									<li>Email address</li>
									<li>Alternate guardians</li>
								</ul>
							</li>
							<li>Set transportation details:
								<ul>
									<li>AM pickup time</li>
									<li>PM dropoff time</li>
									<li>Assigned route</li>
									<li>Special instructions</li>
									<li>Medical alerts</li>
								</ul>
							</li>
							<li>Save the student record</li>
						</ol>
						
						<h5>Managing Special Needs</h5>
						<p>For students with special requirements:</p>
						<ul>
							<li><strong>Medical Needs:</strong> Document allergies, medications, conditions</li>
							<li><strong>Mobility Needs:</strong> Note wheelchair, walker, or assistance required</li>
							<li><strong>Behavioral Needs:</strong> Include relevant behavioral plans</li>
							<li><strong>Communication Needs:</strong> Language preferences or communication methods</li>
						</ul>
						
						<h5>Updating Student Information</h5>
						<ol>
							<li>Search for the student by name or ID</li>
							<li>Click on the student record</li>
							<li>Select "Edit" button</li>
							<li>Update necessary fields</li>
							<li>Add notes about the change</li>
							<li>Save updates</li>
						</ol>
					`),
					Tips: []string{
						"Verify guardian identity when making pickup changes",
						"Keep photos of authorized pickup persons on file",
						"Review and update student info at the start of each school year",
					},
					Warnings: []string{
						"Always verify authorization before releasing students to alternate pickups",
						"Protect student privacy - never share information without authorization",
					},
				},
				{
					ID:    "ecse-students",
					Title: "ECSE Student Management",
					Content: template.HTML(`
						<p>Early Childhood Special Education (ECSE) students require additional tracking and reporting.</p>
						
						<h4>ECSE-Specific Features</h4>
						
						<h5>ECSE Student Setup</h5>
						<ol>
							<li>When adding a student, check "ECSE Student" box</li>
							<li>Enter additional ECSE information:
								<ul>
									<li>IEP status and review date</li>
									<li>Service types (Speech, OT, PT, etc.)</li>
									<li>Service frequency and duration</li>
									<li>Equipment needs</li>
									<li>Behavioral plan summary</li>
									<li>Therapist contact information</li>
								</ul>
							</li>
							<li>Upload relevant documents:
								<ul>
									<li>IEP summary (transportation section)</li>
									<li>Medical action plans</li>
									<li>Emergency care plans</li>
								</ul>
							</li>
						</ol>
						
						<h5>Daily ECSE Tracking</h5>
						<ul>
							<li><strong>Service Delivery:</strong> Track which services were provided</li>
							<li><strong>Attendance:</strong> Special codes for therapy days</li>
							<li><strong>Behavior Tracking:</strong> Note significant behaviors</li>
							<li><strong>Equipment Check:</strong> Verify special equipment is present</li>
						</ul>
						
						<h5>ECSE Reporting</h5>
						<p>Generate specialized reports for:</p>
						<ul>
							<li>Monthly service summaries</li>
							<li>IEP transportation compliance</li>
							<li>Attendance patterns</li>
							<li>Equipment usage</li>
							<li>Incident reports</li>
						</ul>
					`),
					Tips: []string{
						"Maintain close communication with ECSE coordinators",
						"Document all deviations from normal routines",
						"Keep emergency plans easily accessible",
					},
				},
			},
		},
		{
			ID:          "reporting",
			Title:       "Reports & Analytics",
			Icon:        "bi-graph-up",
			Description: "Generate insights from your fleet data",
			Order:       5,
			Sections: []Section{
				{
					ID:    "standard-reports",
					Title: "Standard Reports",
					Content: template.HTML(`
						<p>The system provides various pre-built reports for common needs.</p>
						
						<h4>Available Standard Reports</h4>
						
						<h5>1. Daily Operations Report</h5>
						<ul>
							<li>Routes completed</li>
							<li>Student attendance summary</li>
							<li>Mileage totals</li>
							<li>Incidents or issues</li>
							<li>Driver performance metrics</li>
						</ul>
						
						<h5>2. Monthly Mileage Report</h5>
						<ul>
							<li>Total miles by vehicle</li>
							<li>Fuel consumption estimates</li>
							<li>Cost per mile calculations</li>
							<li>Route efficiency metrics</li>
							<li>Comparison to previous months</li>
						</ul>
						
						<h5>3. Maintenance Summary</h5>
						<ul>
							<li>Completed maintenance items</li>
							<li>Upcoming maintenance schedule</li>
							<li>Maintenance costs by vehicle</li>
							<li>Downtime analysis</li>
							<li>Parts inventory usage</li>
						</ul>
						
						<h5>4. Student Transportation Report</h5>
						<ul>
							<li>Active student count</li>
							<li>Route assignments</li>
							<li>Attendance patterns</li>
							<li>Special needs summary</li>
							<li>Guardian contact list</li>
						</ul>
						
						<h5>Generating Standard Reports</h5>
						<ol>
							<li>Navigate to Reports section</li>
							<li>Select report type from dropdown</li>
							<li>Choose date range or period</li>
							<li>Select filters (optional):
								<ul>
									<li>Specific vehicles</li>
									<li>Specific drivers</li>
									<li>Specific routes</li>
								</ul>
							</li>
							<li>Click "Generate Report"</li>
							<li>Choose format (PDF, Excel, CSV)</li>
							<li>Download or email report</li>
						</ol>
					`),
					Tips: []string{
						"Schedule recurring reports for automatic generation",
						"Save report templates for consistent formatting",
						"Use filters to focus on specific areas of interest",
					},
				},
				{
					ID:    "custom-reports",
					Title: "Custom Report Builder",
					Content: template.HTML(`
						<p>Create custom reports tailored to your specific needs using the Report Builder.</p>
						
						<h4>Using the Report Builder</h4>
						
						<h5>Step 1: Select Data Source</h5>
						<ul>
							<li>Vehicle data</li>
							<li>Driver information</li>
							<li>Student records</li>
							<li>Route data</li>
							<li>Maintenance logs</li>
							<li>Trip logs</li>
						</ul>
						
						<h5>Step 2: Choose Fields</h5>
						<p>Drag and drop fields you want to include:</p>
						<ul>
							<li>Identification fields (names, numbers)</li>
							<li>Date/time fields</li>
							<li>Numeric fields (mileage, costs)</li>
							<li>Status fields</li>
							<li>Custom fields</li>
						</ul>
						
						<h5>Step 3: Apply Filters</h5>
						<ul>
							<li>Date ranges</li>
							<li>Status conditions</li>
							<li>Numeric thresholds</li>
							<li>Text matching</li>
							<li>Combination filters</li>
						</ul>
						
						<h5>Step 4: Configure Grouping</h5>
						<ul>
							<li>Group by vehicle</li>
							<li>Group by driver</li>
							<li>Group by date</li>
							<li>Group by route</li>
							<li>Multi-level grouping</li>
						</ul>
						
						<h5>Step 5: Add Calculations</h5>
						<ul>
							<li>Sums and totals</li>
							<li>Averages</li>
							<li>Counts</li>
							<li>Min/Max values</li>
							<li>Percentages</li>
						</ul>
						
						<h5>Step 6: Format and Export</h5>
						<ol>
							<li>Preview the report</li>
							<li>Adjust column widths</li>
							<li>Add headers/footers</li>
							<li>Include charts/graphs</li>
							<li>Save as template</li>
							<li>Export in desired format</li>
						</ol>
					`),
					Tips: []string{
						"Start with a standard report and modify it",
						"Test with small date ranges first",
						"Save successful report configurations as templates",
					},
				},
			},
		},
	}

	// Add manager-specific chapters
	if strings.Contains(userType, "Manager") {
		chapters = append(chapters, Chapter{
			ID:          "administration",
			Title:       "Administration",
			Icon:        "bi-gear",
			Description: "System administration and user management",
			Order:       6,
			Sections: []Section{
				{
					ID:    "user-management",
					Title: "User Management",
					Content: template.HTML(`
						<p>Managers can create and manage user accounts for drivers and other staff.</p>
						
						<h4>Managing User Accounts</h4>
						
						<h5>Creating New Users</h5>
						<ol>
							<li>Navigate to "Manage Users" from dashboard</li>
							<li>Click "Add New User" button</li>
							<li>Fill in user details:
								<ul>
									<li>Username (must be unique)</li>
									<li>Full name</li>
									<li>Email address</li>
									<li>Phone number</li>
									<li>Role (Driver or Manager)</li>
									<li>Temporary password</li>
								</ul>
							</li>
							<li>Set account options:
								<ul>
									<li>Account active/inactive</li>
									<li>Require password change on first login</li>
									<li>Account expiration date (if temporary)</li>
								</ul>
							</li>
							<li>Save the new user</li>
							<li>System sends welcome email with login instructions</li>
						</ol>
						
						<h5>Approving Driver Registrations</h5>
						<ol>
							<li>Check "Pending Approvals" section regularly</li>
							<li>Review driver application details</li>
							<li>Verify credentials and references</li>
							<li>Click "Approve" or "Reject"</li>
							<li>Add notes for the decision</li>
							<li>Approved drivers receive activation email</li>
						</ol>
						
						<h5>Managing Existing Users</h5>
						<ul>
							<li><strong>Reset Password:</strong> Generate new temporary password</li>
							<li><strong>Lock/Unlock:</strong> Temporarily disable access</li>
							<li><strong>Change Role:</strong> Promote driver to manager or vice versa</li>
							<li><strong>View Activity:</strong> See login history and actions</li>
							<li><strong>Delete User:</strong> Permanently remove (archives data)</li>
						</ul>
					`),
					Tips: []string{
						"Use strong temporary passwords",
						"Regularly review user list for inactive accounts",
						"Document reason for account changes",
					},
					Warnings: []string{
						"Only delete users who no longer need any system access",
						"Changing roles immediately affects user permissions",
					},
				},
				{
					ID:    "system-settings",
					Title: "System Configuration",
					Content: template.HTML(`
						<p>Configure system-wide settings to match your organization's needs.</p>
						
						<h4>Configuration Areas</h4>
						
						<h5>Organization Settings</h5>
						<ul>
							<li>Organization name and logo</li>
							<li>Contact information</li>
							<li>Time zone settings</li>
							<li>Business hours</li>
							<li>Holiday calendar</li>
						</ul>
						
						<h5>Operational Settings</h5>
						<ul>
							<li>Default pickup/dropoff times</li>
							<li>Route buffer times</li>
							<li>Mileage calculation methods</li>
							<li>Fuel cost estimates</li>
							<li>Maintenance intervals</li>
						</ul>
						
						<h5>Notification Settings</h5>
						<ul>
							<li>Email notification preferences</li>
							<li>SMS alert configuration</li>
							<li>System announcement broadcasts</li>
							<li>Automated report scheduling</li>
							<li>Alert thresholds</li>
						</ul>
						
						<h5>Security Settings</h5>
						<ul>
							<li>Password complexity requirements</li>
							<li>Session timeout duration</li>
							<li>Login attempt limits</li>
							<li>Two-factor authentication</li>
							<li>IP whitelist/blacklist</li>
						</ul>
					`),
					Tips: []string{
						"Test notification settings before enabling",
						"Document all configuration changes",
						"Review security settings quarterly",
					},
				},
			},
		})
	}

	// Add troubleshooting chapter for all users
	chapters = append(chapters, Chapter{
		ID:          "troubleshooting",
		Title:       "Troubleshooting",
		Icon:        "bi-tools",
		Description: "Common issues and their solutions",
		Order:       10,
		Sections: []Section{
			{
				ID:    "common-issues",
				Title: "Common Issues",
				Content: template.HTML(`
					<p>Solutions to frequently encountered problems.</p>
					
					<h4>Login Issues</h4>
					
					<h5>Problem: Cannot log in</h5>
					<p><strong>Solutions:</strong></p>
					<ul>
						<li>Verify username and password are correct</li>
						<li>Check if Caps Lock is on</li>
						<li>Clear browser cache and cookies</li>
						<li>Try a different browser</li>
						<li>Contact manager to verify account is active</li>
					</ul>
					
					<h5>Problem: "Session Expired" messages</h5>
					<p><strong>Solutions:</strong></p>
					<ul>
						<li>Log in again - sessions expire after 24 hours</li>
						<li>Check if multiple tabs are open</li>
						<li>Ensure stable internet connection</li>
						<li>Disable browser extensions that might interfere</li>
					</ul>
					
					<h4>Data Entry Issues</h4>
					
					<h5>Problem: Cannot save forms</h5>
					<p><strong>Solutions:</strong></p>
					<ul>
						<li>Check all required fields are filled</li>
						<li>Look for validation error messages</li>
						<li>Ensure dates are in correct format</li>
						<li>Verify internet connection</li>
						<li>Try saving after refreshing the page</li>
					</ul>
					
					<h5>Problem: Data not appearing</h5>
					<p><strong>Solutions:</strong></p>
					<ul>
						<li>Refresh the page (F5)</li>
						<li>Check filters aren't hiding data</li>
						<li>Verify correct date range selected</li>
						<li>Ensure you have permission to view the data</li>
						<li>Clear browser cache</li>
					</ul>
					
					<h4>Performance Issues</h4>
					
					<h5>Problem: System running slowly</h5>
					<p><strong>Solutions:</strong></p>
					<ul>
						<li>Close unnecessary browser tabs</li>
						<li>Clear browser cache and history</li>
						<li>Check internet connection speed</li>
						<li>Try during off-peak hours</li>
						<li>Disable browser extensions</li>
						<li>Update browser to latest version</li>
					</ul>
				`),
				Tips: []string{
					"Keep your browser updated for best performance",
					"Use Chrome, Firefox, or Edge for best compatibility",
					"Save your work frequently to prevent data loss",
				},
			},
			{
				ID:    "error-messages",
				Title: "Understanding Error Messages",
				Content: template.HTML(`
					<p>Common error messages and what they mean.</p>
					
					<h4>Error Message Guide</h4>
					
					<table class="table table-bordered">
						<thead>
							<tr>
								<th>Error Message</th>
								<th>Meaning</th>
								<th>Solution</th>
							</tr>
						</thead>
						<tbody>
							<tr>
								<td>"Invalid credentials"</td>
								<td>Username or password incorrect</td>
								<td>Double-check login information</td>
							</tr>
							<tr>
								<td>"Session expired"</td>
								<td>You've been logged out for security</td>
								<td>Log in again to continue</td>
							</tr>
							<tr>
								<td>"Permission denied"</td>
								<td>You don't have access to this feature</td>
								<td>Contact your manager for access</td>
							</tr>
							<tr>
								<td>"Duplicate entry"</td>
								<td>This record already exists</td>
								<td>Check if record was already created</td>
							</tr>
							<tr>
								<td>"Required field missing"</td>
								<td>A mandatory field is empty</td>
								<td>Fill in all fields marked with *</td>
							</tr>
							<tr>
								<td>"Invalid date format"</td>
								<td>Date entered incorrectly</td>
								<td>Use MM/DD/YYYY format</td>
							</tr>
							<tr>
								<td>"Connection timeout"</td>
								<td>Server didn't respond in time</td>
								<td>Check internet and try again</td>
							</tr>
						</tbody>
					</table>
					
					<h4>When to Contact Support</h4>
					<p>Contact support if you encounter:</p>
					<ul>
						<li>Error messages not listed above</li>
						<li>Repeated errors after trying solutions</li>
						<li>Data inconsistencies or missing records</li>
						<li>System features not working as expected</li>
						<li>Security concerns or suspicious activity</li>
					</ul>
					
					<h4>Information to Provide Support</h4>
					<ul>
						<li>Your username and role</li>
						<li>Exact error message</li>
						<li>What you were trying to do</li>
						<li>Steps that led to the error</li>
						<li>Browser and version you're using</li>
						<li>Screenshot of the error (if possible)</li>
					</ul>
				`),
				Tips: []string{
					"Take screenshots of errors before refreshing",
					"Note the time when errors occur",
					"Keep a log of recurring issues",
				},
			},
		},
	})

	return chapters
}