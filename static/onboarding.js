// User Onboarding and Training System
// Provides interactive tours, progress tracking, and contextual guidance

class OnboardingTour {
    constructor(options = {}) {
        this.steps = options.steps || [];
        this.currentStep = 0;
        this.onComplete = options.onComplete || function() {};
        this.onSkip = options.onSkip || function() {};
        this.tourId = options.tourId || 'default-tour';
        this.allowSkip = options.allowSkip !== false;
        this.showProgress = options.showProgress !== false;
        this.backdrop = null;
        this.tooltip = null;
        this.isActive = false;
        
        // Load saved progress
        this.loadProgress();
        
        // Initialize styles
        this.injectStyles();
    }
    
    injectStyles() {
        if (document.getElementById('onboarding-styles')) return;
        
        const style = document.createElement('style');
        style.id = 'onboarding-styles';
        style.textContent = `
            .onboarding-backdrop {
                position: fixed;
                top: 0;
                left: 0;
                width: 100%;
                height: 100%;
                background: rgba(0, 0, 0, 0.5);
                z-index: 9998;
                transition: opacity 0.3s ease;
            }
            
            .onboarding-highlight {
                position: relative;
                z-index: 9999;
                box-shadow: 0 0 0 4px var(--color-primary, #667eea),
                           0 0 0 8px rgba(102, 126, 234, 0.3);
                border-radius: 8px;
                transition: all 0.3s ease;
            }
            
            .onboarding-tooltip {
                position: absolute;
                background: white;
                border-radius: 12px;
                padding: 24px;
                max-width: 400px;
                box-shadow: 0 10px 40px rgba(0, 0, 0, 0.2);
                z-index: 10000;
                animation: fadeInScale 0.3s ease;
            }
            
            .onboarding-tooltip-arrow {
                position: absolute;
                width: 0;
                height: 0;
                border-style: solid;
            }
            
            .onboarding-tooltip-arrow.top {
                bottom: -10px;
                left: 50%;
                transform: translateX(-50%);
                border-width: 10px 10px 0 10px;
                border-color: white transparent transparent transparent;
            }
            
            .onboarding-tooltip-arrow.bottom {
                top: -10px;
                left: 50%;
                transform: translateX(-50%);
                border-width: 0 10px 10px 10px;
                border-color: transparent transparent white transparent;
            }
            
            .onboarding-tooltip-arrow.left {
                right: -10px;
                top: 50%;
                transform: translateY(-50%);
                border-width: 10px 0 10px 10px;
                border-color: transparent transparent transparent white;
            }
            
            .onboarding-tooltip-arrow.right {
                left: -10px;
                top: 50%;
                transform: translateY(-50%);
                border-width: 10px 10px 10px 0;
                border-color: transparent white transparent transparent;
            }
            
            .onboarding-header {
                display: flex;
                justify-content: space-between;
                align-items: center;
                margin-bottom: 16px;
            }
            
            .onboarding-title {
                font-size: 20px;
                font-weight: 700;
                color: var(--color-text-primary, #2d3748);
                margin: 0;
            }
            
            .onboarding-close {
                background: none;
                border: none;
                font-size: 24px;
                color: var(--color-text-secondary, #718096);
                cursor: pointer;
                padding: 0;
                width: 32px;
                height: 32px;
                display: flex;
                align-items: center;
                justify-content: center;
                border-radius: 50%;
                transition: all 0.2s ease;
            }
            
            .onboarding-close:hover {
                background: var(--color-bg-secondary, #f7fafc);
                color: var(--color-text-primary, #2d3748);
            }
            
            .onboarding-content {
                font-size: 16px;
                line-height: 1.6;
                color: var(--color-text-secondary, #4a5568);
                margin-bottom: 20px;
            }
            
            .onboarding-image {
                width: 100%;
                max-height: 200px;
                object-fit: cover;
                border-radius: 8px;
                margin-bottom: 16px;
            }
            
            .onboarding-actions {
                display: flex;
                justify-content: space-between;
                align-items: center;
                gap: 12px;
            }
            
            .onboarding-progress {
                display: flex;
                gap: 6px;
                flex: 1;
            }
            
            .onboarding-progress-dot {
                width: 8px;
                height: 8px;
                border-radius: 50%;
                background: var(--color-border, #e2e8f0);
                transition: all 0.3s ease;
            }
            
            .onboarding-progress-dot.active {
                background: var(--color-primary, #667eea);
                transform: scale(1.2);
            }
            
            .onboarding-buttons {
                display: flex;
                gap: 8px;
            }
            
            .onboarding-btn {
                padding: 8px 16px;
                border: none;
                border-radius: 8px;
                font-size: 14px;
                font-weight: 600;
                cursor: pointer;
                transition: all 0.2s ease;
                min-height: 40px;
            }
            
            .onboarding-btn-primary {
                background: var(--color-primary, #667eea);
                color: white;
            }
            
            .onboarding-btn-primary:hover {
                background: var(--color-primary-dark, #5a67d8);
                transform: translateY(-1px);
            }
            
            .onboarding-btn-secondary {
                background: var(--color-bg-secondary, #f7fafc);
                color: var(--color-text-primary, #2d3748);
            }
            
            .onboarding-btn-secondary:hover {
                background: var(--color-bg-tertiary, #e2e8f0);
            }
            
            @keyframes fadeInScale {
                from {
                    opacity: 0;
                    transform: scale(0.9);
                }
                to {
                    opacity: 1;
                    transform: scale(1);
                }
            }
            
            @media (max-width: 768px) {
                .onboarding-tooltip {
                    max-width: calc(100vw - 32px);
                    margin: 16px;
                }
            }
        `;
        document.head.appendChild(style);
    }
    
    start() {
        if (this.steps.length === 0) return;
        
        this.isActive = true;
        this.currentStep = 0;
        this.createBackdrop();
        this.showStep(this.currentStep);
    }
    
    createBackdrop() {
        this.backdrop = document.createElement('div');
        this.backdrop.className = 'onboarding-backdrop';
        this.backdrop.addEventListener('click', (e) => {
            if (e.target === this.backdrop && this.allowSkip) {
                this.skip();
            }
        });
        document.body.appendChild(this.backdrop);
    }
    
    showStep(index) {
        if (index >= this.steps.length) {
            this.complete();
            return;
        }
        
        const step = this.steps[index];
        const targetEl = document.querySelector(step.target);
        
        if (!targetEl) {
            console.warn(`Onboarding target not found: ${step.target}`);
            this.next();
            return;
        }
        
        // Highlight target element
        this.highlightElement(targetEl);
        
        // Show tooltip
        this.showTooltip(targetEl, step);
        
        // Execute step callback if provided
        if (step.onShow) {
            step.onShow();
        }
    }
    
    highlightElement(element) {
        // Remove previous highlight
        document.querySelectorAll('.onboarding-highlight').forEach(el => {
            el.classList.remove('onboarding-highlight');
        });
        
        // Add highlight to new element
        element.classList.add('onboarding-highlight');
        
        // Scroll element into view
        element.scrollIntoView({
            behavior: 'smooth',
            block: 'center',
            inline: 'center'
        });
    }
    
    showTooltip(targetEl, step) {
        // Remove existing tooltip
        if (this.tooltip) {
            this.tooltip.remove();
        }
        
        // Create tooltip
        this.tooltip = document.createElement('div');
        this.tooltip.className = 'onboarding-tooltip';
        
        // Header
        const header = document.createElement('div');
        header.className = 'onboarding-header';
        
        const title = document.createElement('h3');
        title.className = 'onboarding-title';
        title.textContent = step.title || `Step ${this.currentStep + 1}`;
        
        const closeBtn = document.createElement('button');
        closeBtn.className = 'onboarding-close';
        closeBtn.innerHTML = '×';
        closeBtn.setAttribute('aria-label', 'Close tour');
        closeBtn.onclick = () => this.skip();
        
        header.appendChild(title);
        if (this.allowSkip) {
            header.appendChild(closeBtn);
        }
        
        // Image (if provided)
        if (step.image) {
            const img = document.createElement('img');
            img.className = 'onboarding-image';
            img.src = step.image;
            img.alt = step.imageAlt || '';
            this.tooltip.appendChild(img);
        }
        
        // Content
        const content = document.createElement('div');
        content.className = 'onboarding-content';
        content.innerHTML = step.content || '';
        
        // Actions
        const actions = document.createElement('div');
        actions.className = 'onboarding-actions';
        
        // Progress dots
        if (this.showProgress && this.steps.length > 1) {
            const progress = document.createElement('div');
            progress.className = 'onboarding-progress';
            
            for (let i = 0; i < this.steps.length; i++) {
                const dot = document.createElement('div');
                dot.className = 'onboarding-progress-dot';
                if (i === this.currentStep) {
                    dot.classList.add('active');
                }
                progress.appendChild(dot);
            }
            
            actions.appendChild(progress);
        }
        
        // Buttons
        const buttons = document.createElement('div');
        buttons.className = 'onboarding-buttons';
        
        if (this.currentStep > 0) {
            const prevBtn = document.createElement('button');
            prevBtn.className = 'onboarding-btn onboarding-btn-secondary';
            prevBtn.textContent = 'Previous';
            prevBtn.onclick = () => this.previous();
            buttons.appendChild(prevBtn);
        }
        
        const nextBtn = document.createElement('button');
        nextBtn.className = 'onboarding-btn onboarding-btn-primary';
        nextBtn.textContent = this.currentStep === this.steps.length - 1 ? 'Complete' : 'Next';
        nextBtn.onclick = () => this.next();
        buttons.appendChild(nextBtn);
        
        actions.appendChild(buttons);
        
        // Assemble tooltip
        this.tooltip.appendChild(header);
        this.tooltip.appendChild(content);
        this.tooltip.appendChild(actions);
        
        // Add arrow
        const arrow = document.createElement('div');
        arrow.className = 'onboarding-tooltip-arrow';
        this.tooltip.appendChild(arrow);
        
        // Add to page
        document.body.appendChild(this.tooltip);
        
        // Position tooltip
        this.positionTooltip(targetEl, this.tooltip, arrow, step.placement);
    }
    
    positionTooltip(targetEl, tooltip, arrow, preferredPlacement = 'bottom') {
        const targetRect = targetEl.getBoundingClientRect();
        const tooltipRect = tooltip.getBoundingClientRect();
        
        const placements = {
            top: () => {
                tooltip.style.left = `${targetRect.left + targetRect.width / 2 - tooltipRect.width / 2}px`;
                tooltip.style.top = `${targetRect.top - tooltipRect.height - 20}px`;
                arrow.className = 'onboarding-tooltip-arrow bottom';
            },
            bottom: () => {
                tooltip.style.left = `${targetRect.left + targetRect.width / 2 - tooltipRect.width / 2}px`;
                tooltip.style.top = `${targetRect.bottom + 20}px`;
                arrow.className = 'onboarding-tooltip-arrow top';
            },
            left: () => {
                tooltip.style.left = `${targetRect.left - tooltipRect.width - 20}px`;
                tooltip.style.top = `${targetRect.top + targetRect.height / 2 - tooltipRect.height / 2}px`;
                arrow.className = 'onboarding-tooltip-arrow right';
            },
            right: () => {
                tooltip.style.left = `${targetRect.right + 20}px`;
                tooltip.style.top = `${targetRect.top + targetRect.height / 2 - tooltipRect.height / 2}px`;
                arrow.className = 'onboarding-tooltip-arrow left';
            }
        };
        
        // Try preferred placement
        if (placements[preferredPlacement]) {
            placements[preferredPlacement]();
        }
        
        // Adjust if tooltip goes off screen
        const viewportWidth = window.innerWidth;
        const viewportHeight = window.innerHeight;
        
        if (tooltip.offsetLeft < 0) {
            tooltip.style.left = '10px';
        } else if (tooltip.offsetLeft + tooltipRect.width > viewportWidth) {
            tooltip.style.left = `${viewportWidth - tooltipRect.width - 10}px`;
        }
        
        if (tooltip.offsetTop < 0) {
            placements.bottom();
        } else if (tooltip.offsetTop + tooltipRect.height > viewportHeight) {
            placements.top();
        }
    }
    
    next() {
        if (this.currentStep < this.steps.length - 1) {
            this.currentStep++;
            this.showStep(this.currentStep);
            this.saveProgress();
        } else {
            this.complete();
        }
    }
    
    previous() {
        if (this.currentStep > 0) {
            this.currentStep--;
            this.showStep(this.currentStep);
            this.saveProgress();
        }
    }
    
    skip() {
        if (confirm('Are you sure you want to skip this tour? You can restart it from the help menu.')) {
            this.cleanup();
            this.onSkip();
            this.markAsCompleted();
        }
    }
    
    complete() {
        this.cleanup();
        this.onComplete();
        this.markAsCompleted();
        
        // Show completion message
        this.showCompletionMessage();
    }
    
    cleanup() {
        // Remove backdrop
        if (this.backdrop) {
            this.backdrop.remove();
            this.backdrop = null;
        }
        
        // Remove tooltip
        if (this.tooltip) {
            this.tooltip.remove();
            this.tooltip = null;
        }
        
        // Remove highlights
        document.querySelectorAll('.onboarding-highlight').forEach(el => {
            el.classList.remove('onboarding-highlight');
        });
        
        this.isActive = false;
    }
    
    saveProgress() {
        const progress = JSON.parse(localStorage.getItem('onboarding-progress') || '{}');
        progress[this.tourId] = {
            currentStep: this.currentStep,
            lastUpdated: new Date().toISOString()
        };
        localStorage.setItem('onboarding-progress', JSON.stringify(progress));
    }
    
    loadProgress() {
        const progress = JSON.parse(localStorage.getItem('onboarding-progress') || '{}');
        if (progress[this.tourId]) {
            this.currentStep = progress[this.tourId].currentStep || 0;
        }
    }
    
    markAsCompleted() {
        const completed = JSON.parse(localStorage.getItem('onboarding-completed') || '[]');
        if (!completed.includes(this.tourId)) {
            completed.push(this.tourId);
            localStorage.setItem('onboarding-completed', JSON.stringify(completed));
        }
    }
    
    isCompleted() {
        const completed = JSON.parse(localStorage.getItem('onboarding-completed') || '[]');
        return completed.includes(this.tourId);
    }
    
    reset() {
        const progress = JSON.parse(localStorage.getItem('onboarding-progress') || '{}');
        delete progress[this.tourId];
        localStorage.setItem('onboarding-progress', JSON.stringify(progress));
        
        const completed = JSON.parse(localStorage.getItem('onboarding-completed') || '[]');
        const index = completed.indexOf(this.tourId);
        if (index > -1) {
            completed.splice(index, 1);
            localStorage.setItem('onboarding-completed', JSON.stringify(completed));
        }
        
        this.currentStep = 0;
    }
    
    showCompletionMessage() {
        const message = document.createElement('div');
        message.style.cssText = `
            position: fixed;
            bottom: 20px;
            right: 20px;
            background: var(--color-success, #48bb78);
            color: white;
            padding: 16px 24px;
            border-radius: 8px;
            box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
            font-weight: 600;
            z-index: 10000;
            animation: slideInRight 0.3s ease;
        `;
        message.textContent = '✓ Tour completed successfully!';
        
        document.body.appendChild(message);
        
        setTimeout(() => {
            message.style.animation = 'slideOutRight 0.3s ease';
            setTimeout(() => message.remove(), 300);
        }, 3000);
    }
}

// Onboarding Manager - coordinates multiple tours and tracks user progress
class OnboardingManager {
    constructor() {
        this.tours = {};
        this.userProfile = this.loadUserProfile();
        this.checkFirstTimeUser();
    }
    
    registerTour(id, tour) {
        this.tours[id] = tour;
    }
    
    startTour(id) {
        if (this.tours[id]) {
            this.tours[id].start();
        }
    }
    
    loadUserProfile() {
        return JSON.parse(localStorage.getItem('onboarding-profile') || JSON.stringify({
            firstVisit: new Date().toISOString(),
            toursCompleted: 0,
            lastActivity: new Date().toISOString(),
            preferences: {
                autoStart: true,
                showHints: true
            }
        }));
    }
    
    saveUserProfile() {
        this.userProfile.lastActivity = new Date().toISOString();
        localStorage.setItem('onboarding-profile', JSON.stringify(this.userProfile));
    }
    
    checkFirstTimeUser() {
        const isFirstTime = !localStorage.getItem('onboarding-not-first-time');
        if (isFirstTime) {
            localStorage.setItem('onboarding-not-first-time', 'true');
            return true;
        }
        return false;
    }
    
    getAvailableTours() {
        const completed = JSON.parse(localStorage.getItem('onboarding-completed') || '[]');
        return Object.keys(this.tours).map(id => ({
            id,
            completed: completed.includes(id),
            tour: this.tours[id]
        }));
    }
    
    resetAllTours() {
        localStorage.removeItem('onboarding-progress');
        localStorage.removeItem('onboarding-completed');
        localStorage.removeItem('onboarding-profile');
        localStorage.removeItem('onboarding-not-first-time');
        
        Object.values(this.tours).forEach(tour => tour.reset());
    }
    
    showTourMenu() {
        const menu = document.createElement('div');
        menu.style.cssText = `
            position: fixed;
            top: 50%;
            left: 50%;
            transform: translate(-50%, -50%);
            background: white;
            padding: 32px;
            border-radius: 16px;
            box-shadow: 0 20px 50px rgba(0, 0, 0, 0.2);
            max-width: 500px;
            width: 90%;
            max-height: 80vh;
            overflow-y: auto;
            z-index: 10001;
        `;
        
        const backdrop = document.createElement('div');
        backdrop.style.cssText = `
            position: fixed;
            top: 0;
            left: 0;
            width: 100%;
            height: 100%;
            background: rgba(0, 0, 0, 0.5);
            z-index: 10000;
        `;
        
        const title = document.createElement('h2');
        title.textContent = 'Interactive Tours';
        title.style.marginBottom = '24px';
        
        const tourList = document.createElement('div');
        tourList.style.cssText = 'display: flex; flex-direction: column; gap: 16px;';
        
        this.getAvailableTours().forEach(({ id, completed, tour }) => {
            const tourItem = document.createElement('div');
            tourItem.style.cssText = `
                padding: 16px;
                border: 2px solid var(--color-border, #e2e8f0);
                border-radius: 8px;
                cursor: pointer;
                transition: all 0.2s ease;
            `;
            
            tourItem.innerHTML = `
                <div style="display: flex; justify-content: space-between; align-items: center;">
                    <div>
                        <h4 style="margin: 0 0 8px 0;">${tour.name || id}</h4>
                        <p style="margin: 0; color: var(--color-text-secondary, #718096); font-size: 14px;">
                            ${tour.description || 'Learn how to use this feature'}
                        </p>
                    </div>
                    <div style="text-align: right;">
                        ${completed ? 
                            '<span style="color: var(--color-success, #48bb78);">✓ Completed</span>' : 
                            '<span style="color: var(--color-primary, #667eea);">Start →</span>'
                        }
                    </div>
                </div>
            `;
            
            tourItem.onmouseover = () => {
                tourItem.style.borderColor = 'var(--color-primary, #667eea)';
                tourItem.style.transform = 'translateY(-2px)';
            };
            
            tourItem.onmouseout = () => {
                tourItem.style.borderColor = 'var(--color-border, #e2e8f0)';
                tourItem.style.transform = 'translateY(0)';
            };
            
            tourItem.onclick = () => {
                document.body.removeChild(backdrop);
                document.body.removeChild(menu);
                if (completed) {
                    tour.reset();
                }
                tour.start();
            };
            
            tourList.appendChild(tourItem);
        });
        
        const closeBtn = document.createElement('button');
        closeBtn.textContent = 'Close';
        closeBtn.style.cssText = `
            margin-top: 24px;
            padding: 12px 24px;
            background: var(--color-bg-secondary, #f7fafc);
            border: none;
            border-radius: 8px;
            cursor: pointer;
            font-weight: 600;
            width: 100%;
        `;
        closeBtn.onclick = () => {
            document.body.removeChild(backdrop);
            document.body.removeChild(menu);
        };
        
        menu.appendChild(title);
        menu.appendChild(tourList);
        menu.appendChild(closeBtn);
        
        backdrop.onclick = () => {
            document.body.removeChild(backdrop);
            document.body.removeChild(menu);
        };
        
        document.body.appendChild(backdrop);
        document.body.appendChild(menu);
    }
}

// Create global onboarding manager
window.onboardingManager = new OnboardingManager();

// Export classes
window.OnboardingTour = OnboardingTour;
window.OnboardingManager = OnboardingManager;