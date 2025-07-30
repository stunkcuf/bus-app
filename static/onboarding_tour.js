// Interactive Onboarding Tour System
// ==================================
// Guides new users through the system with step-by-step tutorials

class OnboardingTour {
    constructor() {
        this.currentStep = 0;
        this.tourData = {};
        this.overlay = null;
        this.tooltip = null;
        this.isActive = false;
        
        // Define tours for different user roles
        this.tours = {
            driver: this.getDriverTour(),
            manager: this.getManagerTour(),
            general: this.getGeneralTour()
        };
        
        this.init();
    }
    
    init() {
        // Add styles
        this.addStyles();
        
        // Create tour elements
        this.createOverlay();
        this.createTooltip();
        
        // Check if user needs onboarding
        this.checkOnboardingStatus();
    }
    
    addStyles() {
        const style = document.createElement('style');
        style.textContent = `
            /* Onboarding Overlay */
            .onboarding-overlay {
                position: fixed;
                top: 0;
                left: 0;
                right: 0;
                bottom: 0;
                background: rgba(0, 0, 0, 0.7);
                z-index: 9998;
                display: none;
                opacity: 0;
                transition: opacity 0.3s ease;
            }
            
            .onboarding-overlay.active {
                display: block;
                opacity: 1;
            }
            
            /* Highlighted Element */
            .onboarding-highlight {
                position: relative;
                z-index: 9999;
                box-shadow: 0 0 0 4px #667eea, 0 0 0 9999px rgba(0, 0, 0, 0.7);
                border-radius: 8px;
                animation: pulse 2s ease-in-out infinite;
            }
            
            @keyframes pulse {
                0%, 100% { box-shadow: 0 0 0 4px #667eea, 0 0 0 9999px rgba(0, 0, 0, 0.7); }
                50% { box-shadow: 0 0 0 8px #667eea, 0 0 0 9999px rgba(0, 0, 0, 0.7); }
            }
            
            /* Tour Tooltip */
            .onboarding-tooltip {
                position: fixed;
                background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
                color: white;
                padding: 24px;
                border-radius: 16px;
                box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
                max-width: 400px;
                z-index: 10000;
                display: none;
                opacity: 0;
                transform: translateY(20px);
                transition: all 0.3s ease;
            }
            
            .onboarding-tooltip.active {
                display: block;
                opacity: 1;
                transform: translateY(0);
            }
            
            .onboarding-tooltip::before {
                content: '';
                position: absolute;
                width: 20px;
                height: 20px;
                background: #667eea;
                transform: rotate(45deg);
            }
            
            .onboarding-tooltip.top::before {
                bottom: -10px;
                left: 50%;
                margin-left: -10px;
            }
            
            .onboarding-tooltip.bottom::before {
                top: -10px;
                left: 50%;
                margin-left: -10px;
            }
            
            .onboarding-tooltip.left::before {
                right: -10px;
                top: 50%;
                margin-top: -10px;
            }
            
            .onboarding-tooltip.right::before {
                left: -10px;
                top: 50%;
                margin-top: -10px;
            }
            
            /* Tooltip Content */
            .onboarding-step-number {
                font-size: 14px;
                opacity: 0.8;
                margin-bottom: 8px;
            }
            
            .onboarding-title {
                font-size: 24px;
                font-weight: 700;
                margin-bottom: 12px;
            }
            
            .onboarding-content {
                font-size: 16px;
                line-height: 1.6;
                margin-bottom: 20px;
            }
            
            .onboarding-actions {
                display: flex;
                justify-content: space-between;
                align-items: center;
            }
            
            .onboarding-progress {
                display: flex;
                gap: 6px;
            }
            
            .onboarding-progress-dot {
                width: 8px;
                height: 8px;
                border-radius: 50%;
                background: rgba(255, 255, 255, 0.3);
                transition: all 0.3s ease;
            }
            
            .onboarding-progress-dot.active {
                background: white;
                width: 24px;
                border-radius: 4px;
            }
            
            .onboarding-buttons {
                display: flex;
                gap: 12px;
            }
            
            .onboarding-btn {
                padding: 10px 20px;
                border: none;
                border-radius: 8px;
                font-weight: 600;
                cursor: pointer;
                transition: all 0.2s ease;
            }
            
            .onboarding-btn-skip {
                background: rgba(255, 255, 255, 0.2);
                color: white;
            }
            
            .onboarding-btn-skip:hover {
                background: rgba(255, 255, 255, 0.3);
            }
            
            .onboarding-btn-next {
                background: white;
                color: #667eea;
            }
            
            .onboarding-btn-next:hover {
                transform: translateY(-2px);
                box-shadow: 0 5px 20px rgba(0, 0, 0, 0.2);
            }
            
            /* Welcome Modal */
            .onboarding-welcome {
                position: fixed;
                top: 50%;
                left: 50%;
                transform: translate(-50%, -50%);
                background: white;
                padding: 40px;
                border-radius: 20px;
                box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
                max-width: 500px;
                text-align: center;
                z-index: 10001;
                color: #2d3748;
            }
            
            .onboarding-welcome h2 {
                color: #667eea;
                margin-bottom: 20px;
                font-size: 32px;
            }
            
            .onboarding-welcome p {
                font-size: 18px;
                line-height: 1.6;
                margin-bottom: 30px;
            }
            
            .onboarding-welcome .onboarding-btn {
                font-size: 18px;
                padding: 14px 32px;
            }
            
            /* Mobile Responsive */
            @media (max-width: 768px) {
                .onboarding-tooltip {
                    max-width: 90vw;
                    margin: 0 5vw;
                }
                
                .onboarding-welcome {
                    width: 90vw;
                    padding: 30px;
                }
            }
        `;
        document.head.appendChild(style);
    }
    
    createOverlay() {
        this.overlay = document.createElement('div');
        this.overlay.className = 'onboarding-overlay';
        document.body.appendChild(this.overlay);
    }
    
    createTooltip() {
        this.tooltip = document.createElement('div');
        this.tooltip.className = 'onboarding-tooltip';
        this.tooltip.innerHTML = `
            <div class="onboarding-step-number">Step <span id="currentStep">1</span> of <span id="totalSteps">5</span></div>
            <h3 class="onboarding-title" id="tourTitle">Welcome!</h3>
            <div class="onboarding-content" id="tourContent">Let's get started.</div>
            <div class="onboarding-actions">
                <div class="onboarding-progress" id="tourProgress"></div>
                <div class="onboarding-buttons">
                    <button class="onboarding-btn onboarding-btn-skip" onclick="onboardingTour.skipTour()">Skip Tour</button>
                    <button class="onboarding-btn onboarding-btn-next" onclick="onboardingTour.nextStep()">
                        <span id="nextBtnText">Next</span> →
                    </button>
                </div>
            </div>
        `;
        document.body.appendChild(this.tooltip);
    }
    
    checkOnboardingStatus() {
        // Check if user has completed onboarding
        const hasCompletedTour = localStorage.getItem('onboardingCompleted');
        const userRole = this.getUserRole();
        
        // Only show tour if explicitly requested via URL hash
        if (window.location.hash === '#tour') {
            setTimeout(() => {
                this.showWelcome(userRole);
            }, 1000);
        }
        
        // Add help menu item for tour
        this.addTourMenuItem();
    }
    
    getUserRole() {
        // Detect user role from page content or session
        const isDashboard = window.location.pathname.includes('dashboard');
        const isManager = window.location.pathname.includes('manager') || 
                         document.body.textContent.includes('Manager Dashboard');
        const isDriver = window.location.pathname.includes('driver') || 
                        document.body.textContent.includes('Driver Portal');
        
        if (isManager) return 'manager';
        if (isDriver) return 'driver';
        return 'general';
    }
    
    showWelcome(role) {
        const welcomeModal = document.createElement('div');
        welcomeModal.className = 'onboarding-welcome';
        welcomeModal.innerHTML = `
            <button style="position: absolute; top: 10px; right: 10px; background: none; border: none; font-size: 30px; cursor: pointer; color: #999;" 
                    onclick="document.querySelector('.onboarding-welcome').remove(); document.querySelector('.onboarding-overlay').classList.remove('active'); localStorage.setItem('onboardingCompleted', 'true');">
                ×
            </button>
            <i class="bi bi-rocket-takeoff" style="font-size: 64px; color: #667eea; margin-bottom: 20px;"></i>
            <h2>Welcome to Fleet Management!</h2>
            <p>We'll guide you through the key features of the system with a quick interactive tour.</p>
            <p><strong>This will only take 2-3 minutes.</strong></p>
            <div style="display: flex; gap: 16px; justify-content: center;">
                <button class="onboarding-btn onboarding-btn-skip" 
                        onclick="document.querySelector('.onboarding-welcome').remove(); document.querySelector('.onboarding-overlay').classList.remove('active'); localStorage.setItem('onboardingCompleted', 'true');">
                    Maybe Later
                </button>
                <button class="onboarding-btn onboarding-btn-next" style="background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); color: white;" 
                        onclick="onboardingTour.startTour('${role}', this.parentElement.parentElement)">
                    Start Tour
                </button>
            </div>
        `;
        
        this.overlay.classList.add('active');
        document.body.appendChild(welcomeModal);
    }
    
    dismissWelcome(modal) {
        modal.remove();
        this.overlay.classList.remove('active');
    }
    
    startTour(role, welcomeModal) {
        if (welcomeModal) {
            welcomeModal.remove();
        }
        
        this.tourData = this.tours[role] || this.tours.general;
        this.currentStep = 0;
        this.isActive = true;
        
        this.showStep();
    }
    
    showStep() {
        const step = this.tourData.steps[this.currentStep];
        if (!step) {
            this.completeTour();
            return;
        }
        
        // Update tooltip content
        document.getElementById('currentStep').textContent = this.currentStep + 1;
        document.getElementById('totalSteps').textContent = this.tourData.steps.length;
        document.getElementById('tourTitle').textContent = step.title;
        document.getElementById('tourContent').innerHTML = step.content;
        
        // Update next button text
        const isLastStep = this.currentStep === this.tourData.steps.length - 1;
        document.getElementById('nextBtnText').textContent = isLastStep ? 'Finish' : 'Next';
        
        // Update progress dots
        this.updateProgress();
        
        // Highlight element if specified
        this.highlightElement(step.element);
        
        // Position tooltip
        this.positionTooltip(step.element, step.position);
        
        // Show tooltip
        this.tooltip.classList.add('active');
        
        // Execute any step actions
        if (step.action) {
            step.action();
        }
    }
    
    highlightElement(selector) {
        // Remove previous highlights
        document.querySelectorAll('.onboarding-highlight').forEach(el => {
            el.classList.remove('onboarding-highlight');
        });
        
        if (selector) {
            const element = document.querySelector(selector);
            if (element) {
                element.classList.add('onboarding-highlight');
                element.scrollIntoView({ behavior: 'smooth', block: 'center' });
            }
        }
    }
    
    positionTooltip(elementSelector, preferredPosition = 'bottom') {
        const tooltip = this.tooltip;
        tooltip.className = 'onboarding-tooltip active';
        
        if (!elementSelector) {
            // Center the tooltip
            tooltip.style.top = '50%';
            tooltip.style.left = '50%';
            tooltip.style.transform = 'translate(-50%, -50%)';
            return;
        }
        
        const element = document.querySelector(elementSelector);
        if (!element) return;
        
        const rect = element.getBoundingClientRect();
        const tooltipRect = tooltip.getBoundingClientRect();
        
        // Calculate positions
        const positions = {
            bottom: {
                top: rect.bottom + 20,
                left: rect.left + (rect.width - tooltipRect.width) / 2
            },
            top: {
                top: rect.top - tooltipRect.height - 20,
                left: rect.left + (rect.width - tooltipRect.width) / 2
            },
            right: {
                top: rect.top + (rect.height - tooltipRect.height) / 2,
                left: rect.right + 20
            },
            left: {
                top: rect.top + (rect.height - tooltipRect.height) / 2,
                left: rect.left - tooltipRect.width - 20
            }
        };
        
        // Use preferred position or find best fit
        let position = preferredPosition;
        const pos = positions[position];
        
        // Adjust if tooltip goes off screen
        if (pos.left < 10) pos.left = 10;
        if (pos.left + tooltipRect.width > window.innerWidth - 10) {
            pos.left = window.innerWidth - tooltipRect.width - 10;
        }
        if (pos.top < 10) pos.top = 10;
        if (pos.top + tooltipRect.height > window.innerHeight - 10) {
            position = 'top';
        }
        
        tooltip.style.top = pos.top + 'px';
        tooltip.style.left = pos.left + 'px';
        tooltip.style.transform = 'none';
        tooltip.classList.add(position);
    }
    
    updateProgress() {
        const progressContainer = document.getElementById('tourProgress');
        progressContainer.innerHTML = '';
        
        for (let i = 0; i < this.tourData.steps.length; i++) {
            const dot = document.createElement('div');
            dot.className = 'onboarding-progress-dot';
            if (i === this.currentStep) {
                dot.classList.add('active');
            }
            progressContainer.appendChild(dot);
        }
    }
    
    nextStep() {
        this.currentStep++;
        this.showStep();
    }
    
    previousStep() {
        if (this.currentStep > 0) {
            this.currentStep--;
            this.showStep();
        }
    }
    
    skipTour() {
        if (confirm('Are you sure you want to skip the tour? You can restart it anytime from the Help menu.')) {
            this.completeTour();
        }
    }
    
    completeTour() {
        // Mark tour as completed
        localStorage.setItem('onboardingCompleted', 'true');
        localStorage.setItem('onboardingCompletedDate', new Date().toISOString());
        
        // Hide tour elements
        this.tooltip.classList.remove('active');
        this.overlay.classList.remove('active');
        document.querySelectorAll('.onboarding-highlight').forEach(el => {
            el.classList.remove('onboarding-highlight');
        });
        
        this.isActive = false;
        
        // Show completion message
        this.showCompletionMessage();
    }
    
    showCompletionMessage() {
        const message = document.createElement('div');
        message.className = 'onboarding-welcome';
        message.innerHTML = `
            <i class="bi bi-check-circle" style="font-size: 64px; color: #10b981; margin-bottom: 20px;"></i>
            <h2>Tour Complete!</h2>
            <p>Great job! You've completed the onboarding tour.</p>
            <p>Remember, you can always:</p>
            <ul style="text-align: left; margin: 20px 0;">
                <li>Access the Help Center for detailed guides</li>
                <li>Restart the tour from the Help menu</li>
                <li>Contact support if you need assistance</li>
            </ul>
            <button class="onboarding-btn onboarding-btn-next" style="background: linear-gradient(135deg, #10b981 0%, #059669 100%); color: white;" 
                    onclick="this.parentElement.remove(); document.querySelector('.onboarding-overlay').classList.remove('active');">
                Get Started
            </button>
        `;
        document.body.appendChild(message);
    }
    
    addTourMenuItem() {
        // Add "Take Tour" option to help menu or navbar
        const helpMenu = document.querySelector('a[href="/help-center"]');
        if (helpMenu && helpMenu.parentElement) {
            const tourLink = document.createElement('a');
            tourLink.href = '#tour';
            tourLink.className = 'btn btn-outline-light btn-sm ms-2';
            tourLink.innerHTML = '<i class="bi bi-play-circle"></i> Take Tour';
            tourLink.onclick = (e) => {
                e.preventDefault();
                this.startTour(this.getUserRole());
            };
            helpMenu.parentElement.insertBefore(tourLink, helpMenu.nextSibling);
        }
    }
    
    // Tour definitions
    getDriverTour() {
        return {
            name: 'Driver Tour',
            steps: [
                {
                    title: 'Welcome to Your Driver Portal',
                    content: 'This is your personal dashboard where you can manage daily operations. Let\'s explore the key features.',
                    element: null,
                    position: 'center'
                },
                {
                    title: 'Quick Stats',
                    content: 'Here you can see your assigned students, today\'s routes, and recent activity at a glance.',
                    element: '.stats-grid',
                    position: 'bottom'
                },
                {
                    title: 'Log Your Trips',
                    content: 'Click these cards to log your morning or afternoon trips. This is where you\'ll spend most of your time.',
                    element: '.action-card',
                    position: 'right'
                },
                {
                    title: 'Recent Trip Logs',
                    content: 'View your recent trips here. You can see attendance, mileage, and any notes from previous routes.',
                    element: '.recent-trips',
                    position: 'top'
                },
                {
                    title: 'Need Help?',
                    content: 'Click the Help button anytime for guides, or use the floating + button for quick trip logging.',
                    element: 'a[href="/help-center"]',
                    position: 'bottom'
                }
            ]
        };
    }
    
    getManagerTour() {
        return {
            name: 'Manager Tour',
            steps: [
                {
                    title: 'Manager Dashboard Overview',
                    content: 'Welcome! This is your command center for managing the entire fleet. Let\'s explore the key features.',
                    element: null,
                    position: 'center'
                },
                {
                    title: 'Fleet Statistics',
                    content: 'Monitor your fleet status in real-time. See active buses, driver assignments, and student counts.',
                    element: '.stats-grid',
                    position: 'bottom'
                },
                {
                    title: 'Quick Actions',
                    content: 'Access frequently used features like fleet management, student roster, and route assignments.',
                    element: '.quick-actions',
                    position: 'top'
                },
                {
                    title: 'Recent Activity',
                    content: 'Stay informed about driver logs, maintenance updates, and system activities.',
                    element: '.activity-feed',
                    position: 'left'
                },
                {
                    title: 'Analytics & Reports',
                    content: 'Access comprehensive reports and analytics to make data-driven decisions.',
                    element: 'a[href*="analytics"]',
                    position: 'bottom'
                },
                {
                    title: 'User Management',
                    content: 'Approve new drivers and manage user accounts from here.',
                    element: 'a[href*="users"]',
                    position: 'bottom'
                },
                {
                    title: 'Get Support',
                    content: 'Access the Help Center for detailed guides, or contact support directly.',
                    element: 'a[href="/help-center"]',
                    position: 'bottom'
                }
            ]
        };
    }
    
    getGeneralTour() {
        return {
            name: 'General Tour',
            steps: [
                {
                    title: 'Welcome to Fleet Management',
                    content: 'Let\'s take a quick tour of the system to help you get started.',
                    element: null,
                    position: 'center'
                },
                {
                    title: 'Navigation',
                    content: 'Use the navigation menu to access different sections of the system.',
                    element: '.navbar',
                    position: 'bottom'
                },
                {
                    title: 'Your Profile',
                    content: 'Click on your username to access profile settings and change your password.',
                    element: '.bi-person-circle',
                    position: 'bottom'
                },
                {
                    title: 'Help & Support',
                    content: 'Need help? Click here to access guides, tutorials, and support.',
                    element: 'a[href="/help-center"]',
                    position: 'bottom'
                }
            ]
        };
    }
}

// Initialize onboarding tour
const onboardingTour = new OnboardingTour();

// Export for global access
window.onboardingTour = onboardingTour;