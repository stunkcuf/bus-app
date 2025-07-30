// Contextual Help System
// Provides in-app help tooltips and guidance for all form fields and UI elements

class ContextualHelp {
    constructor() {
        this.helpData = {
            // Login Page
            'login-username': 'Enter your username provided by the administrator',
            'login-password': 'Enter your password. Contact admin if you forgot it',
            
            // User Management
            'user-role': 'Select whether this user is a Driver or Manager. Managers have full system access',
            'user-status': 'Active users can log in, Pending users need approval, Inactive users are disabled',
            
            // Fleet Management
            'bus-number': 'Enter the bus number as displayed on the vehicle (e.g., 101, 102)',
            'bus-status': 'Active: In service | Maintenance: Being serviced | Out of Service: Not available',
            'vehicle-mileage': 'Current odometer reading from the vehicle dashboard',
            'maintenance-type': 'Oil Change: Every 5,000 miles | Inspection: Annual | Tires: As needed | Repair: When issues arise',
            
            // Student Management
            'student-id': 'Unique identifier for the student (usually provided by school)',
            'studentID': 'Unique identifier for the student (usually provided by school)',
            'studentName': 'Enter the student\'s full name (First and Last name)',
            'positionNumber': 'Order in which this student is picked up/dropped off on the route',
            'guardianName': 'Name of parent or guardian responsible for the student',
            'phoneNumber': 'Primary contact number for emergencies (format: 555-123-4567)',
            'altPhoneNumber': 'Alternative contact number if primary is unavailable',
            'pickupAddress': 'Street address where the student will be picked up',
            'pickup-time': 'Time when student should be picked up in the morning',
            'dropoff-time': 'Time when student is dropped off in the afternoon',
            'guardian-phone': 'Primary contact number for emergencies',
            
            // Route Assignment
            'route-name': 'Descriptive name for the route (e.g., "North Elementary AM")',
            'route-driver': 'Select the driver responsible for this route',
            'route-bus': 'Select the bus to be used for this route',
            
            // Daily Operations
            'trip-date': 'Date of the trip (defaults to today)',
            'trip-period': 'Morning (AM) or Afternoon (PM) route',
            'begin-mileage': 'Odometer reading at the start of the trip',
            'end-mileage': 'Odometer reading at the end of the trip',
            'student-present': 'Check if the student was present for this trip',
            
            // ECSE Module
            'iep-status': 'Individualized Education Program status',
            'service-type': 'Speech: Speech therapy | OT: Occupational therapy | PT: Physical therapy',
            'therapy-frequency': 'How often the student receives this service',
            
            // Reporting
            'report-date-range': 'Select start and end dates for the report',
            'export-format': 'PDF: For printing | Excel: For data analysis | CSV: For other systems',
            'report-type': 'Summary: Overview | Detailed: All records | Custom: Selected fields',
            
            // Maintenance
            'service-date': 'Date when the maintenance was performed',
            'service-cost': 'Total cost of the maintenance (parts + labor)',
            'next-service': 'Recommended date or mileage for next service',
            'service-notes': 'Any additional details about the service performed'
        };
        
        this.init();
    }
    
    init() {
        // Add help icons to all form inputs and selects
        document.addEventListener('DOMContentLoaded', () => {
            this.attachHelpToElements();
            this.setupHelpTriggers();
            this.createHelpStyles();
        });
    }
    
    createHelpStyles() {
        const style = document.createElement('style');
        style.textContent = `
            .help-icon {
                display: inline-block;
                width: 18px;
                height: 18px;
                margin-left: 5px;
                cursor: help;
                vertical-align: middle;
                color: #667eea;
                transition: all 0.2s ease;
            }
            
            .help-icon:hover {
                color: #5a6fd8;
                transform: scale(1.1);
            }
            
            .help-tooltip {
                position: absolute;
                background: rgba(30, 30, 60, 0.95);
                color: white;
                padding: 12px 16px;
                border-radius: 8px;
                font-size: 14px;
                line-height: 1.4;
                max-width: 300px;
                box-shadow: 0 4px 20px rgba(0, 0, 0, 0.3);
                z-index: 9999;
                opacity: 0;
                pointer-events: none;
                transition: opacity 0.3s ease;
                backdrop-filter: blur(10px);
                border: 1px solid rgba(255, 255, 255, 0.1);
            }
            
            .help-tooltip.show {
                opacity: 1;
                pointer-events: auto;
            }
            
            .help-tooltip::before {
                content: '';
                position: absolute;
                top: -6px;
                left: 50%;
                transform: translateX(-50%);
                width: 12px;
                height: 12px;
                background: rgba(30, 30, 60, 0.95);
                transform: translateX(-50%) rotate(45deg);
                border-left: 1px solid rgba(255, 255, 255, 0.1);
                border-top: 1px solid rgba(255, 255, 255, 0.1);
            }
            
            .form-group-with-help {
                position: relative;
            }
            
            .help-quick-tip {
                display: inline-flex;
                align-items: center;
                gap: 8px;
                background: rgba(102, 126, 234, 0.1);
                color: #667eea;
                padding: 8px 12px;
                border-radius: 6px;
                font-size: 13px;
                margin-top: 5px;
                border: 1px solid rgba(102, 126, 234, 0.2);
            }
            
            .help-video-link {
                color: #4facfe;
                text-decoration: none;
                display: inline-flex;
                align-items: center;
                gap: 5px;
                margin-left: 10px;
                font-size: 13px;
            }
            
            .help-video-link:hover {
                text-decoration: underline;
            }
        `;
        document.head.appendChild(style);
    }
    
    attachHelpToElements() {
        // Find all form inputs, selects, and textareas
        const formElements = document.querySelectorAll('input[id], select[id], textarea[id]');
        
        formElements.forEach(element => {
            const helpKey = this.getHelpKey(element);
            if (this.helpData[helpKey]) {
                this.addHelpIcon(element, helpKey);
            }
        });
        
        // Also add help to specific UI elements
        this.addSpecialHelp();
    }
    
    getHelpKey(element) {
        // Try to match by ID first
        const id = element.id;
        if (this.helpData[id]) return id;
        
        // Try to match by name
        const name = element.name;
        if (this.helpData[name]) return name;
        
        // Try to match by partial ID
        for (const key in this.helpData) {
            if (id.includes(key) || (name && name.includes(key))) {
                return key;
            }
        }
        
        return null;
    }
    
    addHelpIcon(element, helpKey) {
        const wrapper = element.closest('.form-group') || element.parentElement;
        if (!wrapper) return;
        
        wrapper.classList.add('form-group-with-help');
        
        const helpIcon = document.createElement('i');
        helpIcon.className = 'bi bi-question-circle-fill help-icon';
        helpIcon.setAttribute('data-help-key', helpKey);
        
        // Insert help icon after label or element
        const label = wrapper.querySelector('label');
        if (label) {
            label.appendChild(helpIcon);
        } else {
            element.parentNode.insertBefore(helpIcon, element.nextSibling);
        }
    }
    
    addSpecialHelp() {
        // Add help to buttons
        const saveButtons = document.querySelectorAll('button[type="submit"]');
        saveButtons.forEach(button => {
            if (button.textContent.includes('Save')) {
                this.addQuickTip(button, 'Changes are saved immediately. No need to confirm.');
            }
        });
        
        // Add help to navigation items
        const navItems = document.querySelectorAll('.nav-link, .action-card');
        navItems.forEach(item => {
            const text = item.textContent.toLowerCase();
            if (text.includes('report')) {
                item.setAttribute('data-help', 'View and generate various reports');
            } else if (text.includes('fleet')) {
                item.setAttribute('data-help', 'Manage buses and vehicles');
            } else if (text.includes('student')) {
                item.setAttribute('data-help', 'Manage student information and assignments');
            }
        });
    }
    
    addQuickTip(element, tip) {
        const tipElement = document.createElement('div');
        tipElement.className = 'help-quick-tip';
        tipElement.innerHTML = `<i class="bi bi-lightbulb"></i> ${tip}`;
        element.parentNode.insertBefore(tipElement, element.nextSibling);
    }
    
    setupHelpTriggers() {
        // Click/tap to show help
        document.addEventListener('click', (e) => {
            if (e.target.classList.contains('help-icon')) {
                e.preventDefault();
                e.stopPropagation();
                this.showHelp(e.target);
            }
        });
        
        // Hover to show help (desktop)
        document.addEventListener('mouseover', (e) => {
            if (e.target.classList.contains('help-icon')) {
                this.showHelp(e.target);
            }
        });
        
        document.addEventListener('mouseout', (e) => {
            if (e.target.classList.contains('help-icon')) {
                this.hideHelp();
            }
        });
        
        // Hide help when clicking elsewhere
        document.addEventListener('click', (e) => {
            if (!e.target.classList.contains('help-icon')) {
                this.hideHelp();
            }
        });
        
        // Keyboard support
        document.addEventListener('keydown', (e) => {
            if (e.key === 'F1') {
                e.preventDefault();
                this.showNearestHelp();
            }
            if (e.key === 'Escape') {
                this.hideHelp();
            }
        });
    }
    
    showHelp(helpIcon) {
        this.hideHelp(); // Hide any existing tooltip
        
        const helpKey = helpIcon.getAttribute('data-help-key');
        const helpText = this.helpData[helpKey];
        
        if (!helpText) return;
        
        const tooltip = document.createElement('div');
        tooltip.className = 'help-tooltip';
        tooltip.textContent = helpText;
        
        // Add video link if available
        const videoLinks = {
            'route-assignment': '/help/videos/route-assignment',
            'student-management': '/help/videos/student-management',
            'daily-operations': '/help/videos/daily-operations'
        };
        
        for (const [key, link] of Object.entries(videoLinks)) {
            if (helpKey.includes(key)) {
                const videoLink = document.createElement('a');
                videoLink.className = 'help-video-link';
                videoLink.href = link;
                videoLink.innerHTML = '<i class="bi bi-play-circle"></i> Watch video';
                videoLink.onclick = (e) => {
                    e.preventDefault();
                    this.openVideoHelp(link);
                };
                tooltip.appendChild(videoLink);
                break;
            }
        }
        
        document.body.appendChild(tooltip);
        
        // Position the tooltip
        const iconRect = helpIcon.getBoundingClientRect();
        const tooltipRect = tooltip.getBoundingClientRect();
        
        let top = iconRect.bottom + 10;
        let left = iconRect.left + (iconRect.width / 2) - (tooltipRect.width / 2);
        
        // Adjust if tooltip goes off screen
        if (left < 10) left = 10;
        if (left + tooltipRect.width > window.innerWidth - 10) {
            left = window.innerWidth - tooltipRect.width - 10;
        }
        
        tooltip.style.top = `${top}px`;
        tooltip.style.left = `${left}px`;
        
        // Show tooltip
        requestAnimationFrame(() => {
            tooltip.classList.add('show');
        });
        
        this.currentTooltip = tooltip;
    }
    
    hideHelp() {
        if (this.currentTooltip) {
            this.currentTooltip.classList.remove('show');
            setTimeout(() => {
                if (this.currentTooltip && this.currentTooltip.parentNode) {
                    this.currentTooltip.parentNode.removeChild(this.currentTooltip);
                }
                this.currentTooltip = null;
            }, 300);
        }
    }
    
    showNearestHelp() {
        // Find the nearest help icon to the focused element
        const focused = document.activeElement;
        if (!focused) return;
        
        const helpIcon = focused.parentElement.querySelector('.help-icon');
        if (helpIcon) {
            this.showHelp(helpIcon);
        }
    }
    
    openVideoHelp(link) {
        // Open video help in a modal or new window
        window.location.href = link;
    }
    
    // Interactive "Show me how" functionality
    showMeHow(task) {
        const guides = {
            'add-student': [
                { element: '#add-student-btn', text: 'Click the "Add Student" button', action: 'highlight' },
                { element: '#studentName', text: 'Enter the student\'s full name', action: 'focus' },
                { element: '#studentID', text: 'Enter the unique student ID', action: 'focus' },
                { element: '#guardianName', text: 'Enter parent/guardian name', action: 'focus' },
                { element: '#phoneNumber', text: 'Enter contact phone number', action: 'focus' },
                { element: '#pickupAddress', text: 'Enter the pickup address', action: 'focus' },
                { element: 'button[type="submit"]', text: 'Click Save to add the student', action: 'highlight' }
            ],
            'log-trip': [
                { element: '.action-card:contains("Morning")', text: 'Click on Morning or Afternoon trip', action: 'highlight' },
                { element: '#trip-date', text: 'Verify the trip date', action: 'focus' },
                { element: '#begin-mileage', text: 'Enter starting odometer reading', action: 'focus' },
                { element: '.student-attendance', text: 'Mark each student present/absent', action: 'highlight' },
                { element: '#end-mileage', text: 'Enter ending odometer reading', action: 'focus' },
                { element: 'button[type="submit"]', text: 'Click Submit to save the trip', action: 'highlight' }
            ],
            'assign-route': [
                { element: '#driver', text: 'Select a driver from the dropdown', action: 'focus' },
                { element: '#route_id', text: 'Choose the route to assign', action: 'focus' },
                { element: '#bus_id', text: 'Select the bus for this route', action: 'focus' },
                { element: 'button[type="submit"]', text: 'Click Create Assignment', action: 'highlight' }
            ],
            'generate-report': [
                { element: 'a[href*="report"]', text: 'Click on Reports in the menu', action: 'highlight' },
                { element: '#report-type', text: 'Select the type of report', action: 'focus' },
                { element: '#date-range', text: 'Choose the date range', action: 'focus' },
                { element: '#export-format', text: 'Select export format (PDF/Excel)', action: 'focus' },
                { element: '#generate-btn', text: 'Click Generate Report', action: 'highlight' }
            ]
        };
        
        const steps = guides[task];
        if (!steps) return;
        
        this.startInteractiveGuide(steps);
    }
    
    startInteractiveGuide(steps) {
        let currentStep = 0;
        
        // Create guide overlay
        const overlay = document.createElement('div');
        overlay.className = 'guide-overlay';
        overlay.style.cssText = `
            position: fixed;
            top: 0;
            left: 0;
            right: 0;
            bottom: 0;
            background: rgba(0, 0, 0, 0.7);
            z-index: 9998;
            cursor: pointer;
        `;
        
        const guideBox = document.createElement('div');
        guideBox.className = 'guide-box';
        guideBox.style.cssText = `
            position: fixed;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            padding: 20px;
            border-radius: 20px;
            z-index: 9999;
            max-width: 400px;
            box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
        `;
        
        const showStep = () => {
            if (currentStep >= steps.length) {
                // Guide complete
                overlay.remove();
                guideBox.remove();
                this.showCompletionMessage();
                return;
            }
            
            const step = steps[currentStep];
            const element = document.querySelector(step.element);
            
            if (!element) {
                // Element not found, skip to next step
                currentStep++;
                showStep();
                return;
            }
            
            // Highlight element
            element.style.position = 'relative';
            element.style.zIndex = '10000';
            element.style.boxShadow = '0 0 0 4px #667eea, 0 0 0 9999px rgba(0, 0, 0, 0.7)';
            
            // Update guide box
            guideBox.innerHTML = `
                <h4 style="margin-bottom: 10px;">Step ${currentStep + 1} of ${steps.length}</h4>
                <p style="margin-bottom: 20px;">${step.text}</p>
                <div style="display: flex; justify-content: space-between;">
                    <button onclick="contextualHelp.skipGuide()" style="
                        background: rgba(255, 255, 255, 0.2);
                        border: none;
                        color: white;
                        padding: 10px 20px;
                        border-radius: 20px;
                        cursor: pointer;
                    ">Skip</button>
                    <button onclick="contextualHelp.nextGuideStep()" style="
                        background: white;
                        border: none;
                        color: #667eea;
                        padding: 10px 20px;
                        border-radius: 20px;
                        cursor: pointer;
                        font-weight: bold;
                    ">Next →</button>
                </div>
            `;
            
            // Position guide box near element
            const rect = element.getBoundingClientRect();
            let top = rect.bottom + 20;
            let left = rect.left;
            
            if (top + 200 > window.innerHeight) {
                top = rect.top - 200;
            }
            if (left + 400 > window.innerWidth) {
                left = window.innerWidth - 420;
            }
            
            guideBox.style.top = `${top}px`;
            guideBox.style.left = `${left}px`;
            
            // Perform action
            if (step.action === 'focus' && element.focus) {
                element.focus();
            }
            
            // Clean up previous step
            if (currentStep > 0) {
                const prevStep = steps[currentStep - 1];
                const prevElement = document.querySelector(prevStep.element);
                if (prevElement) {
                    prevElement.style.boxShadow = '';
                    prevElement.style.zIndex = '';
                }
            }
        };
        
        // Store guide state for external control
        this.currentGuide = {
            overlay,
            guideBox,
            steps,
            currentStep,
            showStep,
            next: () => {
                currentStep++;
                showStep();
            },
            skip: () => {
                overlay.remove();
                guideBox.remove();
                // Clean up any highlighted elements
                steps.forEach(step => {
                    const element = document.querySelector(step.element);
                    if (element) {
                        element.style.boxShadow = '';
                        element.style.zIndex = '';
                    }
                });
                this.currentGuide = null;
            }
        };
        
        document.body.appendChild(overlay);
        document.body.appendChild(guideBox);
        showStep();
    }
    
    nextGuideStep() {
        if (this.currentGuide) {
            this.currentGuide.next();
        }
    }
    
    skipGuide() {
        if (this.currentGuide) {
            this.currentGuide.skip();
        }
    }
    
    showCompletionMessage() {
        const message = document.createElement('div');
        message.style.cssText = `
            position: fixed;
            top: 50%;
            left: 50%;
            transform: translate(-50%, -50%);
            background: linear-gradient(135deg, #10b981 0%, #059669 100%);
            color: white;
            padding: 30px;
            border-radius: 20px;
            text-align: center;
            z-index: 10000;
            box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
        `;
        message.innerHTML = `
            <i class="bi bi-check-circle" style="font-size: 48px; margin-bottom: 10px; display: block;"></i>
            <h3>Guide Complete!</h3>
            <p>You've successfully completed this task guide.</p>
            <button onclick="this.parentElement.remove()" style="
                background: white;
                color: #10b981;
                border: none;
                padding: 10px 30px;
                border-radius: 20px;
                margin-top: 10px;
                cursor: pointer;
                font-weight: bold;
            ">Got it!</button>
        `;
        document.body.appendChild(message);
    }
}

// Initialize the contextual help system
const contextualHelp = new ContextualHelp();