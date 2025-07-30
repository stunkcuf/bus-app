// Session Timeout Warning System
// ==============================
// Warns users before their session expires to prevent data loss

class SessionTimeoutWarning {
    constructor() {
        // Configuration
        this.sessionTimeout = 24 * 60 * 60 * 1000; // 24 hours in milliseconds
        this.warningTime = 30 * 60 * 1000; // Show warning 30 minutes before timeout
        this.checkInterval = 60 * 1000; // Check every minute
        
        // State
        this.lastActivityTime = Date.now();
        this.warningShown = false;
        this.timeoutTimer = null;
        this.warningTimer = null;
        
        this.init();
    }
    
    init() {
        // Add styles
        this.addStyles();
        
        // Create warning modal
        this.createWarningModal();
        
        // Track user activity
        this.trackActivity();
        
        // Start monitoring
        this.startMonitoring();
        
        // Check for existing session info
        this.checkSessionStatus();
    }
    
    addStyles() {
        const style = document.createElement('style');
        style.textContent = `
            /* Session Warning Modal */
            .session-warning-overlay {
                position: fixed;
                top: 0;
                left: 0;
                right: 0;
                bottom: 0;
                background: rgba(0, 0, 0, 0.8);
                display: none;
                align-items: center;
                justify-content: center;
                z-index: 10000;
                backdrop-filter: blur(5px);
            }
            
            .session-warning-overlay.show {
                display: flex;
            }
            
            .session-warning-modal {
                background: rgba(30, 30, 60, 0.95);
                border-radius: 20px;
                padding: 40px;
                max-width: 500px;
                width: 90%;
                text-align: center;
                box-shadow: 0 20px 60px rgba(0, 0, 0, 0.5);
                border: 2px solid rgba(255, 193, 7, 0.5);
                animation: pulse 2s ease-in-out infinite;
            }
            
            @keyframes pulse {
                0%, 100% {
                    border-color: rgba(255, 193, 7, 0.5);
                    box-shadow: 0 20px 60px rgba(0, 0, 0, 0.5);
                }
                50% {
                    border-color: rgba(255, 193, 7, 0.8);
                    box-shadow: 0 20px 60px rgba(255, 193, 7, 0.2);
                }
            }
            
            .session-warning-icon {
                font-size: 80px;
                color: #ffc107;
                margin-bottom: 20px;
                animation: shake 1s ease-in-out infinite;
            }
            
            @keyframes shake {
                0%, 100% { transform: translateX(0); }
                25% { transform: translateX(-5px); }
                75% { transform: translateX(5px); }
            }
            
            .session-warning-title {
                font-size: 28px;
                font-weight: 700;
                color: white;
                margin-bottom: 20px;
            }
            
            .session-warning-message {
                font-size: 18px;
                color: rgba(255, 255, 255, 0.9);
                margin-bottom: 30px;
                line-height: 1.6;
            }
            
            .session-warning-countdown {
                font-size: 36px;
                font-weight: 700;
                color: #ffc107;
                margin: 20px 0;
                font-family: 'Courier New', monospace;
            }
            
            .session-warning-actions {
                display: flex;
                gap: 15px;
                justify-content: center;
                flex-wrap: wrap;
            }
            
            .session-warning-actions .btn {
                padding: 12px 30px;
                font-size: 16px;
                font-weight: 600;
                border-radius: 50px;
                transition: all 0.3s ease;
                min-width: 150px;
            }
            
            .btn-stay-logged-in {
                background: linear-gradient(135deg, #10b981 0%, #059669 100%);
                color: white;
                border: none;
            }
            
            .btn-stay-logged-in:hover {
                transform: translateY(-2px);
                box-shadow: 0 10px 30px rgba(16, 185, 129, 0.4);
            }
            
            .btn-logout-now {
                background: rgba(255, 255, 255, 0.1);
                color: white;
                border: 2px solid rgba(255, 255, 255, 0.3);
            }
            
            .btn-logout-now:hover {
                background: rgba(255, 255, 255, 0.2);
                border-color: rgba(255, 255, 255, 0.5);
            }
            
            /* Session timer in navbar */
            .session-timer {
                display: none;
                align-items: center;
                gap: 8px;
                padding: 8px 16px;
                background: rgba(255, 193, 7, 0.1);
                border: 1px solid rgba(255, 193, 7, 0.3);
                border-radius: 50px;
                color: #ffc107;
                font-size: 14px;
                font-weight: 600;
            }
            
            .session-timer.show {
                display: flex;
            }
            
            .session-timer.warning {
                animation: blink 1s ease-in-out infinite;
            }
            
            @keyframes blink {
                0%, 100% { opacity: 1; }
                50% { opacity: 0.5; }
            }
            
            /* Auto-save reminder */
            .autosave-reminder {
                position: fixed;
                bottom: 20px;
                right: 20px;
                background: rgba(59, 130, 246, 0.9);
                color: white;
                padding: 16px 24px;
                border-radius: 12px;
                display: none;
                align-items: center;
                gap: 12px;
                box-shadow: 0 10px 30px rgba(0, 0, 0, 0.3);
                animation: slideIn 0.5s ease;
            }
            
            .autosave-reminder.show {
                display: flex;
            }
            
            @keyframes slideIn {
                from {
                    transform: translateX(100%);
                    opacity: 0;
                }
                to {
                    transform: translateX(0);
                    opacity: 1;
                }
            }
        `;
        document.head.appendChild(style);
    }
    
    createWarningModal() {
        const modal = document.createElement('div');
        modal.className = 'session-warning-overlay';
        modal.id = 'sessionWarningModal';
        modal.innerHTML = `
            <div class="session-warning-modal">
                <i class="bi bi-clock-history session-warning-icon"></i>
                <h2 class="session-warning-title">Your Session is About to Expire</h2>
                <p class="session-warning-message">
                    You've been inactive for a while. Your session will expire in:
                </p>
                <div class="session-warning-countdown" id="sessionCountdown">30:00</div>
                <p class="session-warning-message">
                    Would you like to stay logged in?
                </p>
                <div class="session-warning-actions">
                    <button class="btn btn-stay-logged-in" onclick="sessionTimeout.extendSession()">
                        <i class="bi bi-check-circle"></i> Stay Logged In
                    </button>
                    <button class="btn btn-logout-now" onclick="sessionTimeout.logout()">
                        <i class="bi bi-box-arrow-right"></i> Logout Now
                    </button>
                </div>
            </div>
        `;
        document.body.appendChild(modal);
        
        // Create autosave reminder
        const reminder = document.createElement('div');
        reminder.className = 'autosave-reminder';
        reminder.id = 'autosaveReminder';
        reminder.innerHTML = `
            <i class="bi bi-info-circle"></i>
            <span>Your work is being auto-saved</span>
        `;
        document.body.appendChild(reminder);
    }
    
    trackActivity() {
        // Events that indicate user activity
        const activityEvents = [
            'mousedown', 'mousemove', 'keypress', 'scroll', 
            'touchstart', 'click', 'input', 'change'
        ];
        
        activityEvents.forEach(event => {
            document.addEventListener(event, () => {
                this.updateActivity();
            }, { passive: true });
        });
        
        // Track fetch requests as activity
        const originalFetch = window.fetch;
        window.fetch = (...args) => {
            this.updateActivity();
            return originalFetch.apply(window, args);
        };
    }
    
    updateActivity() {
        this.lastActivityTime = Date.now();
        
        // Hide warning if shown
        if (this.warningShown) {
            this.hideWarning();
        }
        
        // Reset timers
        this.resetTimers();
    }
    
    startMonitoring() {
        // Check session status every minute
        setInterval(() => {
            this.checkTimeout();
        }, this.checkInterval);
        
        // Initial check
        this.checkTimeout();
    }
    
    checkTimeout() {
        const now = Date.now();
        const timeSinceActivity = now - this.lastActivityTime;
        const timeUntilTimeout = this.sessionTimeout - timeSinceActivity;
        
        // Update session timer display
        this.updateSessionTimer(timeUntilTimeout);
        
        // Check if we should show warning
        if (timeUntilTimeout <= this.warningTime && !this.warningShown) {
            this.showWarning();
        }
        
        // Check if session has expired
        if (timeUntilTimeout <= 0) {
            this.handleTimeout();
        }
    }
    
    updateSessionTimer(timeRemaining) {
        // Find or create timer element in navbar
        let timer = document.getElementById('sessionTimer');
        if (!timer) {
            const navbar = document.querySelector('.navbar .d-flex');
            if (navbar) {
                timer = document.createElement('div');
                timer.id = 'sessionTimer';
                timer.className = 'session-timer';
                timer.innerHTML = '<i class="bi bi-clock"></i><span></span>';
                navbar.insertBefore(timer, navbar.firstChild);
            }
        }
        
        if (timer) {
            const hours = Math.floor(timeRemaining / (1000 * 60 * 60));
            const minutes = Math.floor((timeRemaining % (1000 * 60 * 60)) / (1000 * 60));
            
            if (hours < 1) {
                timer.classList.add('show', 'warning');
                timer.querySelector('span').textContent = `${minutes}m remaining`;
            } else if (hours < 2) {
                timer.classList.add('show');
                timer.classList.remove('warning');
                timer.querySelector('span').textContent = `${hours}h ${minutes}m`;
            } else {
                timer.classList.remove('show', 'warning');
            }
        }
    }
    
    showWarning() {
        this.warningShown = true;
        const modal = document.getElementById('sessionWarningModal');
        modal.classList.add('show');
        
        // Show autosave reminder
        const reminder = document.getElementById('autosaveReminder');
        reminder.classList.add('show');
        setTimeout(() => {
            reminder.classList.remove('show');
        }, 5000);
        
        // Start countdown
        this.startCountdown();
    }
    
    hideWarning() {
        this.warningShown = false;
        const modal = document.getElementById('sessionWarningModal');
        modal.classList.remove('show');
        this.stopCountdown();
    }
    
    startCountdown() {
        const updateCountdown = () => {
            const now = Date.now();
            const timeSinceActivity = now - this.lastActivityTime;
            const timeUntilTimeout = this.sessionTimeout - timeSinceActivity;
            
            if (timeUntilTimeout <= 0) {
                this.handleTimeout();
                return;
            }
            
            const minutes = Math.floor(timeUntilTimeout / (1000 * 60));
            const seconds = Math.floor((timeUntilTimeout % (1000 * 60)) / 1000);
            
            const countdown = document.getElementById('sessionCountdown');
            if (countdown) {
                countdown.textContent = `${minutes}:${seconds.toString().padStart(2, '0')}`;
            }
        };
        
        updateCountdown();
        this.countdownInterval = setInterval(updateCountdown, 1000);
    }
    
    stopCountdown() {
        if (this.countdownInterval) {
            clearInterval(this.countdownInterval);
            this.countdownInterval = null;
        }
    }
    
    extendSession() {
        // Make a request to extend the session
        fetch('/api/extend-session', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            }
        }).then(() => {
            this.updateActivity();
            this.showNotification('Session extended for another 24 hours', 'success');
        }).catch(() => {
            // Even if the request fails, reset the client-side timer
            this.updateActivity();
        });
    }
    
    logout() {
        // Save any pending data
        if (window.autoSave) {
            window.autoSave.saveAllForms();
        }
        
        // Redirect to logout
        window.location.href = '/logout';
    }
    
    handleTimeout() {
        // Save any pending data
        if (window.autoSave) {
            window.autoSave.saveAllForms();
        }
        
        // Show timeout message
        this.showNotification('Your session has expired. Redirecting to login...', 'warning');
        
        // Redirect to login after a short delay
        setTimeout(() => {
            window.location.href = '/login?timeout=1';
        }, 2000);
    }
    
    resetTimers() {
        // Clear existing timers
        if (this.warningTimer) {
            clearTimeout(this.warningTimer);
        }
        if (this.timeoutTimer) {
            clearTimeout(this.timeoutTimer);
        }
        
        // Set new timers
        this.warningTimer = setTimeout(() => {
            this.showWarning();
        }, this.sessionTimeout - this.warningTime);
        
        this.timeoutTimer = setTimeout(() => {
            this.handleTimeout();
        }, this.sessionTimeout);
    }
    
    checkSessionStatus() {
        // Check with server for actual session status
        fetch('/api/session-status')
            .then(response => response.json())
            .then(data => {
                if (data.remainingTime) {
                    // Sync with server time
                    const serverTimeout = data.remainingTime * 1000;
                    if (serverTimeout < this.sessionTimeout) {
                        this.sessionTimeout = serverTimeout;
                        this.resetTimers();
                    }
                }
            })
            .catch(() => {
                // Ignore errors, use client-side tracking
            });
    }
    
    showNotification(message, type = 'info') {
        // Create or use existing notification system
        const notification = document.createElement('div');
        notification.className = `loading-toast ${type} show`;
        notification.innerHTML = `
            <i class="bi bi-${type === 'success' ? 'check-circle' : 'exclamation-circle'}"></i>
            <span>${message}</span>
        `;
        document.body.appendChild(notification);
        
        setTimeout(() => {
            notification.remove();
        }, 5000);
    }
}

// Initialize session timeout warning system
const sessionTimeout = new SessionTimeoutWarning();

// Export for global access
window.sessionTimeout = sessionTimeout;