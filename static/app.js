// Fleet Management System - Main Application JavaScript
// This connects all the dots and makes everything work seamlessly
// Requires: logger.js for production-safe logging

class FleetManagementApp {
    constructor() {
        this.apiBase = window.location.origin;
        this.notifications = [];
        this.activeRequests = new Map();
        this.init();
    }

    init() {
        // Initialize all components
        this.initializeParticles();
        this.initializeNotifications();
        this.initializeInteractiveElements();
        this.initializeRealTimeUpdates();
        this.initializeFormValidation();
        this.initializeSearch();
        this.initializeTooltips();
        this.enhanceDataTables();
        this.setupKeyboardShortcuts();
        
        // Start background tasks
        this.startHealthCheck();
        this.startAutoSave();
    }

    // API Helper Methods
    async apiCall(endpoint, options = {}) {
        const requestId = Math.random().toString(36).substr(2, 9);
        this.activeRequests.set(requestId, true);

        try {
            const response = await fetch(`${this.apiBase}${endpoint}`, {
                ...options,
                headers: {
                    'Content-Type': 'application/json',
                    ...options.headers
                }
            });

            if (!response.ok) {
                throw new Error(`API Error: ${response.statusText}`);
            }

            const data = await response.json();
            this.activeRequests.delete(requestId);
            return data;
        } catch (error) {
            this.activeRequests.delete(requestId);
            this.showNotification('error', `Failed: ${error.message}`);
            throw error;
        }
    }

    // Particle System
    initializeParticles() {
        const particleContainer = document.createElement('div');
        particleContainer.className = 'particle-system';
        document.body.appendChild(particleContainer);

        for (let i = 0; i < 50; i++) {
            const particle = document.createElement('div');
            particle.className = 'particle';
            particle.style.left = Math.random() * 100 + '%';
            particle.style.animationDelay = Math.random() * 10 + 's';
            particle.style.animationDuration = (Math.random() * 10 + 10) + 's';
            particleContainer.appendChild(particle);
        }
    }

    // Notification System
    initializeNotifications() {
        const container = document.createElement('div');
        container.className = 'notification-container';
        container.id = 'notificationContainer';
        document.body.appendChild(container);
    }

    showNotification(type, message, duration = 5000) {
        const notification = document.createElement('div');
        notification.className = `notification notification-${type} animate__animated animate__slideInRight`;
        
        const icon = type === 'success' ? 'check-circle' : 'exclamation-circle';
        notification.innerHTML = `
            <i class="bi bi-${icon}-fill"></i>
            <span>${message}</span>
            <button class="btn-close btn-close-white ms-auto" onclick="this.parentElement.remove()"></button>
        `;

        document.getElementById('notificationContainer').appendChild(notification);

        setTimeout(() => {
            notification.classList.add('animate__slideOutRight');
            setTimeout(() => notification.remove(), 300);
        }, duration);
    }

    // Interactive Elements Enhancement
    initializeInteractiveElements() {
        // Enhance all cards with mouse tracking
        document.querySelectorAll('.data-card, .metric-card, .glass-card').forEach(card => {
            card.addEventListener('mousemove', (e) => {
                const rect = card.getBoundingClientRect();
                const x = e.clientX - rect.left;
                const y = e.clientY - rect.top;
                card.style.setProperty('--mouse-x', `${x}px`);
                card.style.setProperty('--mouse-y', `${y}px`);
            });
        });

        // Add ripple effect to buttons
        document.querySelectorAll('.btn').forEach(button => {
            button.addEventListener('click', function(e) {
                const ripple = document.createElement('span');
                ripple.className = 'ripple';
                this.appendChild(ripple);

                const rect = this.getBoundingClientRect();
                const size = Math.max(rect.width, rect.height);
                const x = e.clientX - rect.left - size / 2;
                const y = e.clientY - rect.top - size / 2;

                ripple.style.width = ripple.style.height = size + 'px';
                ripple.style.left = x + 'px';
                ripple.style.top = y + 'px';

                setTimeout(() => ripple.remove(), 600);
            });
        });
    }

    // Real-time Updates
    initializeRealTimeUpdates() {
        // Update dashboard stats every 30 seconds
        if (document.querySelector('.metric-card')) {
            setInterval(() => this.updateDashboardStats(), 30000);
        }

        // Live search functionality
        const searchInputs = document.querySelectorAll('input[type="search"], #searchInput');
        searchInputs.forEach(input => {
            input.addEventListener('input', debounce((e) => {
                this.performSearch(e.target.value);
            }, 300));
        });
    }

    async updateDashboardStats() {
        try {
            const stats = await this.apiCall('/api/dashboard-stats');
            
            // Animate number changes
            document.querySelectorAll('[data-stat]').forEach(element => {
                const stat = element.dataset.stat;
                if (stats[stat] !== undefined) {
                    this.animateValue(element, parseInt(element.textContent), stats[stat], 1000);
                }
            });
        } catch (error) {
            devLog.error('Failed to update stats:', error);
        }
    }

    animateValue(element, start, end, duration) {
        const startTime = performance.now();
        const updateValue = (currentTime) => {
            const elapsed = currentTime - startTime;
            const progress = Math.min(elapsed / duration, 1);
            const value = Math.floor(start + (end - start) * progress);
            element.textContent = value;
            
            if (progress < 1) {
                requestAnimationFrame(updateValue);
            }
        };
        requestAnimationFrame(updateValue);
    }

    // Form Validation
    initializeFormValidation() {
        document.querySelectorAll('form').forEach(form => {
            form.addEventListener('submit', async (e) => {
                if (!form.checkValidity()) {
                    e.preventDefault();
                    e.stopPropagation();
                    this.showFormErrors(form);
                    return;
                }

                // Handle AJAX form submission
                if (form.dataset.ajax === 'true') {
                    e.preventDefault();
                    await this.submitFormAjax(form);
                }
            });

            // Real-time validation
            form.querySelectorAll('input, select, textarea').forEach(input => {
                input.addEventListener('blur', () => this.validateField(input));
            });
        });
    }

    validateField(field) {
        const isValid = field.checkValidity();
        const parent = field.closest('.form-group') || field.parentElement;
        
        if (isValid) {
            parent.classList.remove('has-error');
            parent.classList.add('has-success');
        } else {
            parent.classList.add('has-error');
            parent.classList.remove('has-success');
            this.showFieldError(field);
        }
    }

    showFieldError(field) {
        const error = field.validationMessage;
        let errorElement = field.parentElement.querySelector('.error-message');
        
        if (!errorElement) {
            errorElement = document.createElement('div');
            errorElement.className = 'error-message text-danger small mt-1';
            field.parentElement.appendChild(errorElement);
        }
        
        errorElement.textContent = error;
    }

    async submitFormAjax(form) {
        const submitBtn = form.querySelector('[type="submit"]');
        const originalText = submitBtn.innerHTML;
        
        submitBtn.disabled = true;
        submitBtn.innerHTML = '<span class="spinner-border spinner-border-sm me-2"></span>Processing...';

        try {
            const formData = new FormData(form);
            const data = Object.fromEntries(formData);
            
            const response = await this.apiCall(form.action, {
                method: form.method || 'POST',
                body: JSON.stringify(data)
            });

            this.showNotification('success', response.message || 'Operation successful!');
            
            if (form.dataset.redirect) {
                setTimeout(() => {
                    window.location.href = form.dataset.redirect;
                }, 1000);
            }
        } catch (error) {
            this.showNotification('error', error.message);
        } finally {
            submitBtn.disabled = false;
            submitBtn.innerHTML = originalText;
        }
    }

    // Search Functionality
    initializeSearch() {
        const searchBox = document.querySelector('#globalSearch');
        if (!searchBox) return;

        searchBox.addEventListener('input', debounce(async (e) => {
            const query = e.target.value;
            if (query.length < 2) return;

            const results = await this.apiCall(`/api/search?q=${encodeURIComponent(query)}`);
            this.displaySearchResults(results);
        }, 300));
    }

    performSearch(query) {
        const tables = document.querySelectorAll('table');
        tables.forEach(table => {
            const rows = table.querySelectorAll('tbody tr');
            rows.forEach(row => {
                const text = row.textContent.toLowerCase();
                row.style.display = text.includes(query.toLowerCase()) ? '' : 'none';
            });
        });

        // Update result count
        const visibleRows = document.querySelectorAll('tbody tr:not([style*="display: none"])').length;
        this.updateSearchStatus(`Found ${visibleRows} results`);
    }

    updateSearchStatus(message) {
        let statusElement = document.querySelector('.search-status');
        if (!statusElement) {
            statusElement = document.createElement('div');
            statusElement.className = 'search-status text-muted small mt-2';
            document.querySelector('.search-container')?.appendChild(statusElement);
        }
        statusElement.textContent = message;
    }

    // Tooltip System
    initializeTooltips() {
        document.querySelectorAll('[data-tooltip]').forEach(element => {
            const tooltip = document.createElement('div');
            tooltip.className = 'tooltip-glass';
            tooltip.textContent = element.dataset.tooltip;
            document.body.appendChild(tooltip);

            element.addEventListener('mouseenter', (e) => {
                const rect = e.target.getBoundingClientRect();
                tooltip.style.left = rect.left + rect.width / 2 - tooltip.offsetWidth / 2 + 'px';
                tooltip.style.top = rect.top - tooltip.offsetHeight - 10 + 'px';
                tooltip.classList.add('show');
            });

            element.addEventListener('mouseleave', () => {
                tooltip.classList.remove('show');
            });
        });
    }

    // Data Table Enhancements
    enhanceDataTables() {
        document.querySelectorAll('table').forEach(table => {
            // Add sorting functionality
            const headers = table.querySelectorAll('th');
            headers.forEach((header, index) => {
                header.style.cursor = 'pointer';
                header.addEventListener('click', () => this.sortTable(table, index));
            });

            // Add row selection
            const rows = table.querySelectorAll('tbody tr');
            rows.forEach(row => {
                row.addEventListener('click', (e) => {
                    if (e.target.tagName !== 'BUTTON' && e.target.tagName !== 'A') {
                        row.classList.toggle('selected');
                    }
                });
            });
        });
    }

    sortTable(table, columnIndex) {
        const tbody = table.querySelector('tbody');
        const rows = Array.from(tbody.querySelectorAll('tr'));
        const isAscending = table.dataset.sortOrder !== 'asc';
        
        rows.sort((a, b) => {
            const aValue = a.cells[columnIndex].textContent.trim();
            const bValue = b.cells[columnIndex].textContent.trim();
            
            if (!isNaN(aValue) && !isNaN(bValue)) {
                return isAscending ? aValue - bValue : bValue - aValue;
            }
            
            return isAscending ? 
                aValue.localeCompare(bValue) : 
                bValue.localeCompare(aValue);
        });
        
        rows.forEach(row => tbody.appendChild(row));
        table.dataset.sortOrder = isAscending ? 'asc' : 'desc';
    }

    // Keyboard Shortcuts
    setupKeyboardShortcuts() {
        document.addEventListener('keydown', (e) => {
            // Ctrl/Cmd + K for search
            if ((e.ctrlKey || e.metaKey) && e.key === 'k') {
                e.preventDefault();
                document.querySelector('#searchInput')?.focus();
            }
            
            // Escape to close modals
            if (e.key === 'Escape') {
                document.querySelector('.modal.show .btn-close')?.click();
            }
            
            // Ctrl/Cmd + S to save
            if ((e.ctrlKey || e.metaKey) && e.key === 's') {
                e.preventDefault();
                this.saveCurrentForm();
            }
        });
    }

    saveCurrentForm() {
        const activeForm = document.querySelector('form:not([data-no-autosave])');
        if (activeForm) {
            const submitBtn = activeForm.querySelector('[type="submit"]');
            submitBtn?.click();
        }
    }

    // Health Check
    startHealthCheck() {
        setInterval(async () => {
            try {
                const response = await fetch('/health');
                if (!response.ok) {
                    this.showNotification('error', 'Connection lost. Trying to reconnect...');
                }
            } catch (error) {
                devLog.error('Health check failed:', error);
            }
        }, 60000); // Every minute
    }

    // Auto-save Draft
    startAutoSave() {
        setInterval(() => {
            document.querySelectorAll('form[data-autosave="true"]').forEach(form => {
                const formData = new FormData(form);
                const data = Object.fromEntries(formData);
                localStorage.setItem(`draft_${form.id}`, JSON.stringify(data));
            });
        }, 30000); // Every 30 seconds
    }

    // Load drafts on page load
    loadDrafts() {
        document.querySelectorAll('form[data-autosave="true"]').forEach(form => {
            const draft = localStorage.getItem(`draft_${form.id}`);
            if (draft) {
                const data = JSON.parse(draft);
                Object.entries(data).forEach(([key, value]) => {
                    const field = form.querySelector(`[name="${key}"]`);
                    if (field) field.value = value;
                });
            }
        });
    }
}

// Utility Functions
function debounce(func, wait) {
    let timeout;
    return function executedFunction(...args) {
        const later = () => {
            clearTimeout(timeout);
            func(...args);
        };
        clearTimeout(timeout);
        timeout = setTimeout(later, wait);
    };
}

// Initialize when DOM is ready
document.addEventListener('DOMContentLoaded', () => {
    window.fleetApp = new FleetManagementApp();
    
    // Load any saved drafts
    window.fleetApp.loadDrafts();
    
    // Initialize tooltips
    const tooltipTriggerList = [].slice.call(document.querySelectorAll('[data-bs-toggle="tooltip"]'));
    tooltipTriggerList.map(function (tooltipTriggerEl) {
        return new bootstrap.Tooltip(tooltipTriggerEl);
    });
});

// Export for use in other modules
window.FleetManagementApp = FleetManagementApp;