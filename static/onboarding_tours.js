// Onboarding Tours Configuration
// Defines specific tours for different features and user roles

// Manager Dashboard Tour
const managerDashboardTour = new OnboardingTour({
    tourId: 'manager-dashboard',
    name: 'Manager Dashboard Overview',
    description: 'Learn how to navigate and use the manager dashboard effectively',
    steps: [
        {
            target: '.page-header h1',
            title: 'Welcome to Your Dashboard!',
            content: `
                <p>This is your command center for managing the fleet. From here, you can:</p>
                <ul style="margin: 16px 0; padding-left: 20px;">
                    <li>View fleet statistics at a glance</li>
                    <li>Access all management features</li>
                    <li>Monitor daily operations</li>
                    <li>Respond to alerts and issues</li>
                </ul>
                <p>Let's take a quick tour to get you familiar with everything!</p>
            `,
            placement: 'bottom'
        },
        {
            target: '.dashboard-grid',
            title: 'Fleet Overview Metrics',
            content: `
                <p>These cards show your fleet's current status:</p>
                <ul style="margin: 16px 0; padding-left: 20px;">
                    <li><strong>Total Buses:</strong> All vehicles in your fleet</li>
                    <li><strong>Active Drivers:</strong> Drivers available for routes</li>
                    <li><strong>Total Students:</strong> Registered students</li>
                    <li><strong>Active Routes:</strong> Currently operating routes</li>
                </ul>
                <p>Click any metric for more details.</p>
            `,
            placement: 'top'
        },
        {
            target: '[data-help="quick-actions"]',
            title: 'Quick Actions',
            content: `
                <p>These buttons provide fast access to common tasks:</p>
                <ul style="margin: 16px 0; padding-left: 20px;">
                    <li><strong>Manage Fleet:</strong> View and maintain vehicles</li>
                    <li><strong>Assign Routes:</strong> Match drivers to buses and routes</li>
                    <li><strong>Manage Students:</strong> Add or update student information</li>
                    <li><strong>Import Data:</strong> Bulk upload from Excel files</li>
                </ul>
            `,
            placement: 'top'
        },
        {
            target: '[href="/help-demo"]',
            title: 'Getting Help',
            content: `
                <p>Need assistance? We're here to help!</p>
                <ul style="margin: 16px 0; padding-left: 20px;">
                    <li>Click the <strong>Help</strong> button for contextual guidance</li>
                    <li>Look for <i class="bi bi-question-circle"></i> icons for quick tips</li>
                    <li>Access video tutorials and documentation</li>
                </ul>
                <p>You can restart any tour from the Help menu.</p>
            `,
            placement: 'left'
        }
    ],
    onComplete: function() {
        console.log('Manager dashboard tour completed');
    }
});

// Fleet Management Tour
const fleetManagementTour = new OnboardingTour({
    tourId: 'fleet-management',
    name: 'Fleet Management',
    description: 'Learn how to manage your bus fleet and track maintenance',
    steps: [
        {
            target: '.page-actions',
            title: 'Fleet Management Actions',
            content: `
                <p>Key fleet management features:</p>
                <ul style="margin: 16px 0; padding-left: 20px;">
                    <li><strong>Add New Bus:</strong> Register a new vehicle</li>
                    <li><strong>Log Maintenance:</strong> Track service and repairs</li>
                    <li><strong>Company Vehicles:</strong> Manage support vehicles</li>
                </ul>
            `,
            placement: 'bottom'
        },
        {
            target: '.table thead',
            title: 'Fleet Information Table',
            content: `
                <p>This table shows all your vehicles with:</p>
                <ul style="margin: 16px 0; padding-left: 20px;">
                    <li><strong>Status indicators:</strong> Active, Maintenance, or Out of Service</li>
                    <li><strong>Oil & Tire status:</strong> Visual health indicators</li>
                    <li><strong>Quick actions:</strong> Edit details or view maintenance history</li>
                </ul>
                <p><strong>Tip:</strong> Click column headers to sort!</p>
            `,
            placement: 'bottom'
        },
        {
            target: '.status-indicator',
            title: 'Visual Status Indicators',
            content: `
                <p>Color codes help you quickly identify issues:</p>
                <ul style="margin: 16px 0; padding-left: 20px;">
                    <li><span style="color: #48bb78;">●</span> <strong>Green:</strong> Good condition</li>
                    <li><span style="color: #ed8936;">●</span> <strong>Yellow:</strong> Attention needed soon</li>
                    <li><span style="color: #f56565;">●</span> <strong>Red:</strong> Immediate action required</li>
                </ul>
                <p>Regular maintenance keeps your fleet in the green!</p>
            `,
            placement: 'left'
        }
    ]
});

// Route Assignment Tour
const routeAssignmentTour = new OnboardingTour({
    tourId: 'route-assignment',
    name: 'Route Assignment',
    description: 'Learn how to assign drivers to buses and routes',
    steps: [
        {
            target: '[onclick="showAssignmentWizard()"]',
            title: 'Assignment Wizard',
            content: `
                <p>The <strong>Assignment Wizard</strong> makes route assignment easy!</p>
                <p>It guides you through:</p>
                <ol style="margin: 16px 0; padding-left: 20px;">
                    <li>Selecting an available driver</li>
                    <li>Choosing a bus in good condition</li>
                    <li>Assigning a route</li>
                    <li>Confirming the assignment</li>
                </ol>
                <p>The wizard checks for conflicts automatically!</p>
            `,
            placement: 'bottom'
        },
        {
            target: '.assignment-row',
            title: 'Current Assignments',
            content: `
                <p>This table shows all active route assignments:</p>
                <ul style="margin: 16px 0; padding-left: 20px;">
                    <li>Driver name and assigned bus</li>
                    <li>Route information</li>
                    <li>Assignment date</li>
                    <li>Quick unassign option</li>
                </ul>
                <p><strong>Note:</strong> Drivers can only have one active route.</p>
            `,
            placement: 'top'
        }
    ]
});

// Student Management Tour  
const studentManagementTour = new OnboardingTour({
    tourId: 'student-management',
    name: 'Student Management',
    description: 'Learn how to add and manage student information',
    steps: [
        {
            target: '[onclick="document.getElementById(\'addStudentModal\').style.display = \'block\';"]',
            title: 'Adding New Students',
            content: `
                <p>Click here to add a new student to the system.</p>
                <p>You'll need:</p>
                <ul style="margin: 16px 0; padding-left: 20px;">
                    <li>Student's full name</li>
                    <li>Route assignment</li>
                    <li>Contact phone numbers</li>
                    <li>Pickup and dropoff locations</li>
                </ul>
                <p><strong>Tip:</strong> The form has auto-complete for addresses!</p>
            `,
            placement: 'bottom'
        },
        {
            target: '.student-card',
            title: 'Student Information Cards',
            content: `
                <p>Each card displays:</p>
                <ul style="margin: 16px 0; padding-left: 20px;">
                    <li><strong>Student details:</strong> Name, ID, and guardian</li>
                    <li><strong>Route info:</strong> Assigned route and driver</li>
                    <li><strong>Locations:</strong> Pickup and dropoff addresses</li>
                    <li><strong>Status:</strong> Active or inactive</li>
                </ul>
                <p>Use the action buttons to edit or remove students.</p>
            `,
            placement: 'right'
        },
        {
            target: '.search-box',
            title: 'Finding Students',
            content: `
                <p>Quickly find students by:</p>
                <ul style="margin: 16px 0; padding-left: 20px;">
                    <li>Name</li>
                    <li>Student ID</li>
                    <li>Route</li>
                    <li>Phone number</li>
                </ul>
                <p>The search updates as you type!</p>
            `,
            placement: 'bottom'
        }
    ]
});

// Import Data Tour
const importDataTour = new OnboardingTour({
    tourId: 'import-data',
    name: 'Data Import',
    description: 'Learn how to import data from Excel files',
    steps: [
        {
            target: '[onclick="showImportWizard()"]',
            title: 'Import Wizard',
            content: `
                <p>The <strong>Import Wizard</strong> helps you upload data safely:</p>
                <ol style="margin: 16px 0; padding-left: 20px;">
                    <li><strong>Select file:</strong> Choose your Excel file</li>
                    <li><strong>Analyze:</strong> Review detected columns</li>
                    <li><strong>Map columns:</strong> Match Excel columns to system fields</li>
                    <li><strong>Preview:</strong> Check data before importing</li>
                    <li><strong>Import:</strong> Process the data</li>
                </ol>
                <p>The wizard validates everything to prevent errors!</p>
            `,
            placement: 'bottom'
        },
        {
            target: '.file-upload-area',
            title: 'File Upload',
            content: `
                <p>Upload Excel files by:</p>
                <ul style="margin: 16px 0; padding-left: 20px;">
                    <li><strong>Drag & Drop:</strong> Drag files directly here</li>
                    <li><strong>Browse:</strong> Click to select files</li>
                </ul>
                <p><strong>Supported formats:</strong> .xlsx and .xls files up to 10MB</p>
            `,
            placement: 'top'
        },
        {
            target: '.sample-format',
            title: 'Excel Format Guide',
            content: `
                <p>This table shows the required Excel format.</p>
                <p><strong>Important:</strong></p>
                <ul style="margin: 16px 0; padding-left: 20px;">
                    <li>First row must contain column headers</li>
                    <li>Data starts from row 2</li>
                    <li>Follow the exact column order shown</li>
                    <li>Bus IDs must match existing vehicles</li>
                </ul>
            `,
            placement: 'top'
        }
    ]
});

// Driver Dashboard Tour
const driverDashboardTour = new OnboardingTour({
    tourId: 'driver-dashboard',
    name: 'Driver Dashboard',
    description: 'Learn how to use the driver dashboard for daily operations',
    steps: [
        {
            target: '.route-info-card',
            title: 'Your Route Information',
            content: `
                <p>This card shows your assigned route details:</p>
                <ul style="margin: 16px 0; padding-left: 20px;">
                    <li><strong>Route name</strong> and number</li>
                    <li><strong>Assigned bus</strong> with status</li>
                    <li><strong>Total students</strong> on your route</li>
                </ul>
                <p>Check this every morning before starting!</p>
            `,
            placement: 'bottom'
        },
        {
            target: '[data-bs-target="#morningLogModal"]',
            title: 'Morning Log',
            content: `
                <p>Start each day by completing your morning log:</p>
                <ol style="margin: 16px 0; padding-left: 20px;">
                    <li>Record departure time</li>
                    <li>Note starting mileage</li>
                    <li>Mark student attendance</li>
                    <li>Submit the log</li>
                </ol>
                <p><strong>Remember:</strong> Accurate logs are required!</p>
            `,
            placement: 'left'
        },
        {
            target: '.student-list',
            title: 'Student Roster',
            content: `
                <p>Your complete student list with:</p>
                <ul style="margin: 16px 0; padding-left: 20px;">
                    <li><strong>Pickup order:</strong> Students in route sequence</li>
                    <li><strong>Contact info:</strong> Guardian phone numbers</li>
                    <li><strong>Locations:</strong> Pickup and dropoff addresses</li>
                    <li><strong>Special notes:</strong> Important information</li>
                </ul>
                <p><strong>Tip:</strong> Print this list for offline reference!</p>
            `,
            placement: 'top'
        }
    ]
});

// First Time User Tour
const firstTimeUserTour = new OnboardingTour({
    tourId: 'first-time-user',
    name: 'Getting Started',
    description: 'Essential overview for new users',
    steps: [
        {
            target: 'body',
            title: 'Welcome to Fleet Management System!',
            content: `
                <p>We're excited to have you on board! This system helps you:</p>
                <ul style="margin: 16px 0; padding-left: 20px;">
                    <li>Manage your bus fleet efficiently</li>
                    <li>Track student transportation</li>
                    <li>Schedule and monitor routes</li>
                    <li>Maintain vehicle service records</li>
                </ul>
                <p>This quick tour will show you the basics. Ready to begin?</p>
            `,
            placement: 'center'
        },
        {
            target: '.navigation-component',
            title: 'Navigation Bar',
            content: `
                <p>The navigation bar is your gateway to all features:</p>
                <ul style="margin: 16px 0; padding-left: 20px;">
                    <li><strong>Home:</strong> Return to dashboard</li>
                    <li><strong>Menu items:</strong> Access different sections</li>
                    <li><strong>User menu:</strong> Profile and logout</li>
                    <li><strong>Help:</strong> Get assistance anytime</li>
                </ul>
            `,
            placement: 'bottom'
        },
        {
            target: '[data-help]',
            title: 'Help System',
            content: `
                <p>We've built help into every page:</p>
                <ul style="margin: 16px 0; padding-left: 20px;">
                    <li>Look for <i class="bi bi-question-circle"></i> icons</li>
                    <li>Hover over elements for tooltips</li>
                    <li>Click Help for detailed guides</li>
                    <li>Access tours from the Help menu</li>
                </ul>
                <p><strong>Remember:</strong> Help is always one click away!</p>
            `,
            placement: 'left'
        },
        {
            target: '.btn-primary',
            title: 'Taking Action',
            content: `
                <p>Throughout the system:</p>
                <ul style="margin: 16px 0; padding-left: 20px;">
                    <li><strong>Blue buttons:</strong> Primary actions</li>
                    <li><strong>Gray buttons:</strong> Secondary options</li>
                    <li><strong>Red buttons:</strong> Delete or remove</li>
                    <li><strong>Icons:</strong> Visual cues for actions</li>
                </ul>
                <p>Don't worry - we'll always confirm before making changes!</p>
            `,
            placement: 'top'
        }
    ],
    onComplete: function() {
        // Mark user as onboarded
        localStorage.setItem('user-onboarded', 'true');
    }
});

// Register all tours
window.addEventListener('DOMContentLoaded', function() {
    // Register tours with the manager
    window.onboardingManager.registerTour('manager-dashboard', managerDashboardTour);
    window.onboardingManager.registerTour('fleet-management', fleetManagementTour);
    window.onboardingManager.registerTour('route-assignment', routeAssignmentTour);
    window.onboardingManager.registerTour('student-management', studentManagementTour);
    window.onboardingManager.registerTour('import-data', importDataTour);
    window.onboardingManager.registerTour('driver-dashboard', driverDashboardTour);
    window.onboardingManager.registerTour('first-time-user', firstTimeUserTour);
    
    // Auto-start first time user tour if applicable
    const isFirstTime = window.onboardingManager.checkFirstTimeUser();
    const autoStartEnabled = window.onboardingManager.userProfile.preferences.autoStart;
    const currentPath = window.location.pathname;
    
    if (isFirstTime && autoStartEnabled && currentPath === '/manager-dashboard') {
        setTimeout(() => {
            firstTimeUserTour.start();
        }, 1000);
    }
});

// Contextual tour triggers
function startContextualTour() {
    const currentPath = window.location.pathname;
    const tourMap = {
        '/manager-dashboard': 'manager-dashboard',
        '/fleet': 'fleet-management',
        '/assign-routes': 'route-assignment',
        '/students': 'student-management',
        '/import-mileage': 'import-data',
        '/import-ecse': 'import-data',
        '/driver-dashboard': 'driver-dashboard'
    };
    
    const tourId = tourMap[currentPath];
    if (tourId && window.onboardingManager.tours[tourId]) {
        const tour = window.onboardingManager.tours[tourId];
        if (!tour.isCompleted()) {
            // Show prompt to start tour
            showTourPrompt(tourId);
        }
    }
}

// Tour prompt helper
function showTourPrompt(tourId) {
    const prompt = document.createElement('div');
    prompt.style.cssText = `
        position: fixed;
        bottom: 20px;
        left: 20px;
        background: var(--color-primary, #667eea);
        color: white;
        padding: 16px 20px;
        border-radius: 12px;
        box-shadow: 0 4px 20px rgba(0, 0, 0, 0.15);
        max-width: 350px;
        z-index: 1000;
        animation: slideInLeft 0.3s ease;
    `;
    
    prompt.innerHTML = `
        <div style="display: flex; align-items: start; gap: 12px;">
            <i class="bi bi-lightbulb" style="font-size: 24px; flex-shrink: 0;"></i>
            <div style="flex: 1;">
                <p style="margin: 0 0 8px 0; font-weight: 600;">New to this page?</p>
                <p style="margin: 0 0 12px 0; font-size: 14px; opacity: 0.9;">
                    Take a quick tour to learn about the features available here.
                </p>
                <div style="display: flex; gap: 8px;">
                    <button onclick="startTourFromPrompt('${tourId}')" style="
                        background: white;
                        color: var(--color-primary, #667eea);
                        border: none;
                        padding: 6px 16px;
                        border-radius: 6px;
                        font-weight: 600;
                        cursor: pointer;
                    ">Start Tour</button>
                    <button onclick="dismissTourPrompt()" style="
                        background: transparent;
                        color: white;
                        border: 1px solid rgba(255, 255, 255, 0.3);
                        padding: 6px 16px;
                        border-radius: 6px;
                        cursor: pointer;
                    ">Not Now</button>
                </div>
            </div>
        </div>
    `;
    
    document.body.appendChild(prompt);
    
    // Auto-dismiss after 10 seconds
    setTimeout(() => {
        if (document.body.contains(prompt)) {
            prompt.style.animation = 'slideOutLeft 0.3s ease';
            setTimeout(() => prompt.remove(), 300);
        }
    }, 10000);
    
    window.currentTourPrompt = prompt;
}

window.startTourFromPrompt = function(tourId) {
    if (window.currentTourPrompt) {
        window.currentTourPrompt.remove();
    }
    window.onboardingManager.startTour(tourId);
};

window.dismissTourPrompt = function() {
    if (window.currentTourPrompt) {
        window.currentTourPrompt.style.animation = 'slideOutLeft 0.3s ease';
        setTimeout(() => window.currentTourPrompt.remove(), 300);
    }
};

// Add animations
const animationStyles = document.createElement('style');
animationStyles.textContent = `
    @keyframes slideInLeft {
        from {
            transform: translateX(-100%);
            opacity: 0;
        }
        to {
            transform: translateX(0);
            opacity: 1;
        }
    }
    
    @keyframes slideOutLeft {
        from {
            transform: translateX(0);
            opacity: 1;
        }
        to {
            transform: translateX(-100%);
            opacity: 0;
        }
    }
    
    @keyframes slideInRight {
        from {
            transform: translateX(100%);
            opacity: 0;
        }
        to {
            transform: translateX(0);
            opacity: 1;
        }
    }
    
    @keyframes slideOutRight {
        from {
            transform: translateX(0);
            opacity: 1;
        }
        to {
            transform: translateX(100%);
            opacity: 0;
        }
    }
`;
document.head.appendChild(animationStyles);