// User Progress Tracking System
// ============================
// Automatically tracks user interactions with features

class ProgressTracker {
    constructor() {
        this.tracked = new Set(); // Prevent duplicate tracking in same session
        this.queuedEvents = [];
        this.isOnline = navigator.onLine;
        
        this.init();
    }
    
    init() {
        // Track page loads
        this.trackPageView();
        
        // Track feature interactions
        this.setupEventListeners();
        
        // Handle offline/online events
        window.addEventListener('online', () => {
            this.isOnline = true;
            this.flushQueue();
        });
        
        window.addEventListener('offline', () => {
            this.isOnline = false;
        });
        
        // Track important milestones
        this.checkMilestones();
        
        // Show progress widget if enabled
        this.initProgressWidget();
    }
    
    trackPageView() {
        const path = window.location.pathname;
        let feature = this.getFeatureFromPath(path);
        
        if (feature) {
            this.trackFeature(feature);
        }
    }
    
    getFeatureFromPath(path) {
        const featureMap = {
            '/fleet': 'fleet_viewed',
            '/assign-routes': 'routes_page_viewed',
            '/manage-users': 'users_page_viewed',
            '/students': 'students_viewed',
            '/driver-dashboard': 'dashboard_accessed',
            '/manager-dashboard': 'dashboard_accessed',
            '/help-center': 'help_accessed',
            '/getting-started': 'getting_started_viewed',
            '/practice-mode': 'practice_mode_accessed',
            '/quick-reference': 'quick_reference_viewed',
            '/analytics-dashboard': 'analytics_viewed',
            '/report-builder': 'reports_accessed',
            '/ecse-dashboard': 'ecse_viewed',
            '/maintenance-records': 'maintenance_viewed',
            '/gps-tracking': 'gps_accessed',
            '/profile': 'profile_viewed',
            '/change-password': 'password_change_attempted'
        };
        
        return featureMap[path] || null;
    }
    
    setupEventListeners() {
        // Track form submissions
        document.addEventListener('submit', (e) => {
            const form = e.target;
            const formId = form.id || form.getAttribute('data-feature');
            
            if (formId) {
                this.trackFeature(formId + '_submitted');
            }
            
            // Specific form tracking
            if (form.action.includes('/save-log')) {
                this.trackFeature('trip_logged', 'completed');
            } else if (form.action.includes('/add-student')) {
                this.trackFeature('student_added', 'completed');
            } else if (form.action.includes('/assign-route')) {
                this.trackFeature('route_assigned', 'completed');
            } else if (form.action.includes('/approve-user')) {
                this.trackFeature('user_approved', 'completed');
            }
        });
        
        // Track button clicks
        document.addEventListener('click', (e) => {
            const button = e.target.closest('button, a');
            if (!button) return;
            
            const feature = button.getAttribute('data-track');
            if (feature) {
                this.trackFeature(feature);
            }
            
            // Track specific actions
            if (button.textContent.includes('Generate') && window.location.pathname.includes('report')) {
                this.trackFeature('report_generated', 'completed');
            } else if (button.textContent.includes('Export')) {
                this.trackFeature('data_exported');
            } else if (button.textContent.includes('Print')) {
                this.trackFeature('print_requested');
            } else if (button.classList.contains('attendance-checkbox')) {
                this.trackFeature('attendance_taken', 'in_progress');
            }
        });
        
        // Track tour completion
        document.addEventListener('tourCompleted', () => {
            this.trackFeature('tour_completed', 'completed');
        });
        
        // Track practice mode
        if (window.isPracticeMode && window.isPracticeMode()) {
            this.trackFeature('practice_mode_used');
        }
    }
    
    checkMilestones() {
        // Check for first login (if no previous tracking data)
        if (!localStorage.getItem('hasLoggedInBefore')) {
            this.trackFeature('first_login', 'completed');
            localStorage.setItem('hasLoggedInBefore', 'true');
        }
        
        // Check if profile is complete
        const profileFields = document.querySelectorAll('[data-profile-field]');
        if (profileFields.length > 0) {
            let allFilled = true;
            profileFields.forEach(field => {
                if (!field.value || field.value.trim() === '') {
                    allFilled = false;
                }
            });
            
            if (allFilled) {
                this.trackFeature('profile_updated', 'completed');
            }
        }
    }
    
    trackFeature(feature, status = 'in_progress') {
        // Don't track the same feature twice in one session
        const trackingKey = `${feature}_${status}`;
        if (this.tracked.has(trackingKey)) {
            return;
        }
        
        this.tracked.add(trackingKey);
        
        const data = {
            feature: feature,
            status: status,
            timestamp: new Date().toISOString()
        };
        
        if (this.isOnline) {
            this.sendTrackingData(data);
        } else {
            this.queuedEvents.push(data);
        }
    }
    
    async sendTrackingData(data) {
        try {
            const response = await fetch('/api/progress', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify(data)
            });
            
            if (!response.ok) {
                throw new Error('Failed to track progress');
            }
            
            // Update local progress cache
            this.updateLocalProgress(data);
            
        } catch (error) {
            console.error('Progress tracking error:', error);
            // Queue for retry
            this.queuedEvents.push(data);
        }
    }
    
    flushQueue() {
        if (this.queuedEvents.length === 0) return;
        
        const events = [...this.queuedEvents];
        this.queuedEvents = [];
        
        events.forEach(event => {
            this.sendTrackingData(event);
        });
    }
    
    updateLocalProgress(data) {
        const progress = JSON.parse(localStorage.getItem('userProgress') || '{}');
        
        if (!progress[data.feature]) {
            progress[data.feature] = {
                first_accessed: data.timestamp,
                access_count: 0
            };
        }
        
        progress[data.feature].last_accessed = data.timestamp;
        progress[data.feature].access_count++;
        progress[data.feature].status = data.status;
        
        localStorage.setItem('userProgress', JSON.stringify(progress));
        
        // Trigger progress update event
        window.dispatchEvent(new CustomEvent('progressUpdated', {
            detail: { feature: data.feature, progress: progress }
        }));
    }
    
    async getProgress() {
        try {
            const response = await fetch('/api/progress');
            const data = await response.json();
            return data;
        } catch (error) {
            console.error('Failed to fetch progress:', error);
            // Return local progress as fallback
            return this.getLocalProgress();
        }
    }
    
    getLocalProgress() {
        const progress = JSON.parse(localStorage.getItem('userProgress') || '{}');
        const features = Object.keys(progress).map(key => ({
            feature: key,
            ...progress[key]
        }));
        
        return {
            features: features,
            progress_percent: this.calculateProgressPercent(features)
        };
    }
    
    calculateProgressPercent(features) {
        // Basic calculation based on feature count
        const expectedFeatures = 10; // Adjust based on role
        const completedFeatures = features.filter(f => f.status === 'completed').length;
        return Math.min(100, Math.round((completedFeatures / expectedFeatures) * 100));
    }
    
    initProgressWidget() {
        // Check if widget should be shown
        const showWidget = localStorage.getItem('showProgressWidget') !== 'false';
        if (!showWidget) return;
        
        // Only show on dashboard pages
        if (!window.location.pathname.includes('dashboard')) return;
        
        // Create progress widget
        const widget = document.createElement('div');
        widget.id = 'progress-widget';
        widget.className = 'progress-widget';
        widget.innerHTML = `
            <div class="progress-widget-header">
                <span class="progress-widget-title">Your Progress</span>
                <button class="progress-widget-close" onclick="progressTracker.hideWidget()">
                    <i class="bi bi-x"></i>
                </button>
            </div>
            <div class="progress-widget-content">
                <div class="progress-bar-container">
                    <div class="progress-bar-fill" id="progress-bar-fill" style="width: 0%"></div>
                </div>
                <div class="progress-text" id="progress-text">Loading...</div>
                <a href="/progress" class="progress-link">View Details →</a>
            </div>
        `;
        
        // Add styles
        this.addWidgetStyles();
        
        // Add to page
        document.body.appendChild(widget);
        
        // Load progress data
        this.updateProgressWidget();
    }
    
    addWidgetStyles() {
        if (document.getElementById('progress-widget-styles')) return;
        
        const style = document.createElement('style');
        style.id = 'progress-widget-styles';
        style.textContent = `
            .progress-widget {
                position: fixed;
                bottom: 20px;
                right: 20px;
                background: white;
                border-radius: 12px;
                box-shadow: 0 4px 20px rgba(0, 0, 0, 0.15);
                width: 300px;
                z-index: 1000;
                transform: translateY(120%);
                transition: transform 0.3s ease;
            }
            
            .progress-widget.show {
                transform: translateY(0);
            }
            
            .progress-widget-header {
                display: flex;
                justify-content: space-between;
                align-items: center;
                padding: 15px;
                border-bottom: 1px solid #e9ecef;
            }
            
            .progress-widget-title {
                font-weight: 600;
                color: #495057;
            }
            
            .progress-widget-close {
                background: none;
                border: none;
                font-size: 20px;
                color: #6c757d;
                cursor: pointer;
                padding: 0;
                width: 24px;
                height: 24px;
                display: flex;
                align-items: center;
                justify-content: center;
            }
            
            .progress-widget-close:hover {
                color: #495057;
            }
            
            .progress-widget-content {
                padding: 15px;
            }
            
            .progress-bar-container {
                height: 8px;
                background: #e9ecef;
                border-radius: 4px;
                overflow: hidden;
                margin-bottom: 10px;
            }
            
            .progress-bar-fill {
                height: 100%;
                background: linear-gradient(90deg, #667eea, #764ba2);
                transition: width 0.5s ease;
            }
            
            .progress-text {
                font-size: 14px;
                color: #6c757d;
                margin-bottom: 10px;
            }
            
            .progress-link {
                font-size: 14px;
                color: #667eea;
                text-decoration: none;
                font-weight: 500;
            }
            
            .progress-link:hover {
                color: #764ba2;
            }
        `;
        document.head.appendChild(style);
    }
    
    async updateProgressWidget() {
        const widget = document.getElementById('progress-widget');
        if (!widget) return;
        
        try {
            const progress = await this.getProgress();
            const percent = progress.progress_percent || 0;
            
            document.getElementById('progress-bar-fill').style.width = percent + '%';
            document.getElementById('progress-text').textContent = 
                percent === 100 ? 'Onboarding Complete! 🎉' : 
                `${percent}% Complete - ${progress.completed_milestones || 0} of ${progress.total_milestones || 0} milestones`;
            
            // Show widget with animation
            setTimeout(() => widget.classList.add('show'), 100);
            
        } catch (error) {
            console.error('Failed to update progress widget:', error);
            widget.remove();
        }
    }
    
    hideWidget() {
        const widget = document.getElementById('progress-widget');
        if (widget) {
            widget.classList.remove('show');
            setTimeout(() => widget.remove(), 300);
            localStorage.setItem('showProgressWidget', 'false');
        }
    }
}

// Initialize progress tracker
const progressTracker = new ProgressTracker();

// Export for global access
window.progressTracker = progressTracker;