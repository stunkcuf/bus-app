// Comprehensive Help Tooltip System
class HelpTooltipSystem {
    constructor() {
        this.tooltips = new Map();
        this.activeTooltip = null;
        this.init();
    }

    init() {
        // Initialize tooltip container
        this.createTooltipContainer();
        
        // Bind events
        document.addEventListener('DOMContentLoaded', () => {
            this.attachTooltips();
            this.bindGlobalEvents();
        });
    }

    createTooltipContainer() {
        const container = document.createElement('div');
        container.id = 'help-tooltip-container';
        container.className = 'help-tooltip-container';
        container.style.display = 'none';
        document.body.appendChild(container);
    }

    attachTooltips() {
        // Find all elements with data-help attribute
        const helpElements = document.querySelectorAll('[data-help]');
        
        helpElements.forEach(element => {
            const helpText = element.getAttribute('data-help');
            const helpTitle = element.getAttribute('data-help-title') || 'Help';
            
            // Add help icon if not already present
            if (!element.querySelector('.help-icon')) {
                const helpIcon = document.createElement('i');
                helpIcon.className = 'bi bi-question-circle help-icon';
                helpIcon.setAttribute('aria-label', 'Help');
                helpIcon.setAttribute('role', 'button');
                helpIcon.setAttribute('tabindex', '0');
                element.appendChild(helpIcon);
            }
            
            // Store tooltip data
            this.tooltips.set(element, {
                title: helpTitle,
                content: helpText
            });
            
            // Bind events
            this.bindTooltipEvents(element);
        });
    }

    bindTooltipEvents(element) {
        const helpIcon = element.querySelector('.help-icon');
        if (!helpIcon) return;

        // Mouse events
        helpIcon.addEventListener('mouseenter', (e) => this.showTooltip(element, e));
        helpIcon.addEventListener('mouseleave', () => this.hideTooltip());
        
        // Keyboard events for accessibility
        helpIcon.addEventListener('keydown', (e) => {
            if (e.key === 'Enter' || e.key === ' ') {
                e.preventDefault();
                this.toggleTooltip(element, e);
            }
            if (e.key === 'Escape') {
                this.hideTooltip();
            }
        });
        
        // Touch events for mobile
        helpIcon.addEventListener('touchstart', (e) => {
            e.preventDefault();
            this.toggleTooltip(element, e);
        });
    }

    bindGlobalEvents() {
        // Hide tooltip on scroll
        window.addEventListener('scroll', () => this.hideTooltip());
        
        // Hide tooltip on click outside
        document.addEventListener('click', (e) => {
            if (!e.target.closest('.help-icon') && !e.target.closest('#help-tooltip-container')) {
                this.hideTooltip();
            }
        });
        
        // Update position on window resize
        window.addEventListener('resize', () => {
            if (this.activeTooltip) {
                this.updateTooltipPosition(this.activeTooltip);
            }
        });
    }

    showTooltip(element, event) {
        const tooltipData = this.tooltips.get(element);
        if (!tooltipData) return;
        
        const container = document.getElementById('help-tooltip-container');
        
        // Create tooltip content
        container.innerHTML = `
            <div class="help-tooltip-arrow"></div>
            <div class="help-tooltip-header">
                <i class="bi bi-info-circle"></i>
                <span>${tooltipData.title}</span>
            </div>
            <div class="help-tooltip-content">${tooltipData.content}</div>
        `;
        
        // Show container
        container.style.display = 'block';
        
        // Position tooltip
        this.activeTooltip = element;
        this.updateTooltipPosition(element);
        
        // Announce to screen readers
        this.announceTooltip(tooltipData.content);
    }

    hideTooltip() {
        const container = document.getElementById('help-tooltip-container');
        container.style.display = 'none';
        this.activeTooltip = null;
    }

    toggleTooltip(element, event) {
        if (this.activeTooltip === element) {
            this.hideTooltip();
        } else {
            this.showTooltip(element, event);
        }
    }

    updateTooltipPosition(element) {
        const container = document.getElementById('help-tooltip-container');
        const helpIcon = element.querySelector('.help-icon');
        if (!helpIcon) return;
        
        const iconRect = helpIcon.getBoundingClientRect();
        const containerRect = container.getBoundingClientRect();
        
        // Calculate position
        let top = iconRect.top - containerRect.height - 10;
        let left = iconRect.left + (iconRect.width / 2) - (containerRect.width / 2);
        
        // Adjust if tooltip goes off screen
        if (top < 10) {
            top = iconRect.bottom + 10;
            container.classList.add('tooltip-bottom');
        } else {
            container.classList.remove('tooltip-bottom');
        }
        
        if (left < 10) {
            left = 10;
        } else if (left + containerRect.width > window.innerWidth - 10) {
            left = window.innerWidth - containerRect.width - 10;
        }
        
        container.style.top = top + 'px';
        container.style.left = left + 'px';
    }

    announceTooltip(content) {
        // Create screen reader announcement
        const announcement = document.createElement('div');
        announcement.setAttribute('role', 'status');
        announcement.setAttribute('aria-live', 'polite');
        announcement.className = 'sr-only';
        announcement.textContent = 'Help: ' + content;
        
        document.body.appendChild(announcement);
        setTimeout(() => announcement.remove(), 1000);
    }
}

// Initialize the help system
const helpTooltipSystem = new HelpTooltipSystem();

// Export for use in other modules
window.HelpTooltipSystem = HelpTooltipSystem;