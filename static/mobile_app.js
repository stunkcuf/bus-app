// Mobile App Framework for Fleet Management System

class MobileApp {
    constructor() {
        this.touchStartX = 0;
        this.touchStartY = 0;
        this.swipeThreshold = 50;
        this.pullRefreshThreshold = 80;
        this.isPulling = false;
        this.activeModal = null;
        this.hapticEnabled = true;
        
        this.init();
    }
    
    init() {
        // Check if running on mobile
        this.isMobile = /Android|webOS|iPhone|iPad|iPod|BlackBerry|IEMobile|Opera Mini/i.test(navigator.userAgent);
        
        // Add mobile class to body
        if (this.isMobile) {
            document.body.classList.add('mobile');
        }
        
        // Initialize features
        this.initServiceWorker();
        this.initTouchGestures();
        this.initPullToRefresh();
        this.initBottomNav();
        this.initModals();
        this.initHaptics();
        this.initOrientationHandler();
        this.initNetworkStatus();
        this.initBiometrics();
        
        // Setup event listeners
        this.setupEventListeners();
        
        // Initialize animations
        this.initAnimations();
    }
    
    // Service Worker for offline support
    initServiceWorker() {
        if ('serviceWorker' in navigator) {
            navigator.serviceWorker.register('/sw.js')
                .then(reg => console.log('Service Worker registered'))
                .catch(err => console.error('Service Worker registration failed:', err));
        }
    }
    
    // Touch gesture handling
    initTouchGestures() {
        document.addEventListener('touchstart', this.handleTouchStart.bind(this), { passive: true });
        document.addEventListener('touchmove', this.handleTouchMove.bind(this), { passive: false });
        document.addEventListener('touchend', this.handleTouchEnd.bind(this), { passive: true });
        
        // Prevent double-tap zoom
        let lastTouchEnd = 0;
        document.addEventListener('touchend', (e) => {
            const now = Date.now();
            if (now - lastTouchEnd <= 300) {
                e.preventDefault();
            }
            lastTouchEnd = now;
        }, false);
    }
    
    handleTouchStart(e) {
        this.touchStartX = e.touches[0].clientX;
        this.touchStartY = e.touches[0].clientY;
        
        // Add ripple effect
        const target = e.target.closest('.mobile-touchable');
        if (target) {
            this.createRipple(target, e.touches[0]);
        }
    }
    
    handleTouchMove(e) {
        if (!this.touchStartX || !this.touchStartY) return;
        
        const deltaX = e.touches[0].clientX - this.touchStartX;
        const deltaY = e.touches[0].clientY - this.touchStartY;
        
        // Handle pull to refresh
        if (this.isPullingEnabled() && deltaY > 0 && window.scrollY === 0) {
            e.preventDefault();
            this.handlePullRefresh(deltaY);
        }
        
        // Handle swipe actions
        const swipeContainer = e.target.closest('.mobile-swipe-container');
        if (swipeContainer && Math.abs(deltaX) > Math.abs(deltaY)) {
            e.preventDefault();
            this.handleSwipe(swipeContainer, deltaX);
        }
    }
    
    handleTouchEnd(e) {
        const deltaX = e.changedTouches[0].clientX - this.touchStartX;
        const deltaY = e.changedTouches[0].clientY - this.touchStartY;
        
        // Complete pull to refresh
        if (this.isPulling) {
            this.completePullRefresh();
        }
        
        // Complete swipe action
        const swipeContainer = e.target.closest('.mobile-swipe-container');
        if (swipeContainer) {
            this.completeSwipe(swipeContainer, deltaX);
        }
        
        // Reset touch coordinates
        this.touchStartX = 0;
        this.touchStartY = 0;
    }
    
    // Pull to refresh functionality
    initPullToRefresh() {
        // Create pull to refresh indicator
        const indicator = document.createElement('div');
        indicator.className = 'mobile-pull-refresh';
        indicator.innerHTML = '<div class="mobile-spinner"></div>';
        document.body.appendChild(indicator);
        this.pullIndicator = indicator;
    }
    
    isPullingEnabled() {
        return document.querySelector('[data-pull-refresh="true"]') !== null;
    }
    
    handlePullRefresh(deltaY) {
        this.isPulling = true;
        const progress = Math.min(deltaY / this.pullRefreshThreshold, 1);
        
        // Show and position indicator
        this.pullIndicator.classList.add('visible');
        this.pullIndicator.style.opacity = progress;
        this.pullIndicator.style.transform = `translateX(-50%) scale(${progress})`;
        
        // Haptic feedback at threshold
        if (progress === 1 && this.hapticEnabled) {
            this.triggerHaptic('light');
        }
    }
    
    completePullRefresh() {
        if (this.isPulling) {
            this.isPulling = false;
            
            // Trigger refresh
            const currentPage = document.querySelector('[data-pull-refresh="true"]');
            if (currentPage && currentPage.dataset.refreshAction) {
                window[currentPage.dataset.refreshAction]();
            }
            
            // Hide indicator
            setTimeout(() => {
                this.pullIndicator.classList.remove('visible');
            }, 500);
        }
    }
    
    // Swipe actions
    handleSwipe(container, deltaX) {
        const content = container.querySelector('.mobile-swipe-content');
        const actions = container.querySelector('.mobile-swipe-actions');
        
        if (content && actions) {
            const maxSwipe = actions.offsetWidth;
            const swipeAmount = Math.max(-maxSwipe, Math.min(0, deltaX));
            
            content.style.transform = `translateX(${swipeAmount}px)`;
            actions.style.transform = `translateX(${100 + (swipeAmount / maxSwipe * 100)}%)`;
        }
    }
    
    completeSwipe(container, deltaX) {
        const content = container.querySelector('.mobile-swipe-content');
        const actions = container.querySelector('.mobile-swipe-actions');
        
        if (content && actions) {
            const threshold = -this.swipeThreshold;
            
            if (deltaX < threshold) {
                // Show actions
                content.style.transform = `translateX(-${actions.offsetWidth}px)`;
                actions.style.transform = 'translateX(0)';
            } else {
                // Hide actions
                content.style.transform = 'translateX(0)';
                actions.style.transform = 'translateX(100%)';
            }
            
            // Add transition
            content.style.transition = 'transform 0.3s ease';
            actions.style.transition = 'transform 0.3s ease';
            
            // Remove transition after animation
            setTimeout(() => {
                content.style.transition = '';
                actions.style.transition = '';
            }, 300);
        }
    }
    
    // Bottom navigation
    initBottomNav() {
        const navItems = document.querySelectorAll('.mobile-nav-item');
        
        navItems.forEach(item => {
            item.addEventListener('click', (e) => {
                e.preventDefault();
                
                // Update active state
                navItems.forEach(nav => nav.classList.remove('active'));
                item.classList.add('active');
                
                // Haptic feedback
                if (this.hapticEnabled) {
                    this.triggerHaptic('light');
                }
                
                // Navigate
                const page = item.dataset.page;
                if (page) {
                    this.navigateToPage(page);
                }
            });
        });
    }
    
    // Modal handling
    initModals() {
        // Setup modal triggers
        document.addEventListener('click', (e) => {
            const trigger = e.target.closest('[data-modal]');
            if (trigger) {
                e.preventDefault();
                const modalId = trigger.dataset.modal;
                this.openModal(modalId);
            }
            
            const close = e.target.closest('[data-modal-close]');
            if (close) {
                e.preventDefault();
                this.closeModal();
            }
        });
        
        // Handle backdrop clicks
        document.addEventListener('click', (e) => {
            if (e.target.classList.contains('mobile-modal-backdrop')) {
                this.closeModal();
            }
        });
    }
    
    openModal(modalId) {
        const modal = document.getElementById(modalId);
        const backdrop = document.querySelector('.mobile-modal-backdrop');
        
        if (modal) {
            // Show modal
            modal.classList.add('active');
            backdrop?.classList.add('active');
            this.activeModal = modal;
            
            // Prevent body scroll
            document.body.style.overflow = 'hidden';
            
            // Haptic feedback
            if (this.hapticEnabled) {
                this.triggerHaptic('medium');
            }
        }
    }
    
    closeModal() {
        if (this.activeModal) {
            const backdrop = document.querySelector('.mobile-modal-backdrop');
            
            // Hide modal
            this.activeModal.classList.remove('active');
            backdrop?.classList.remove('active');
            
            // Restore body scroll
            document.body.style.overflow = '';
            
            this.activeModal = null;
        }
    }
    
    // Haptic feedback
    initHaptics() {
        // Check if haptics are supported
        this.hapticEnabled = 'vibrate' in navigator;
        
        // Add haptic feedback to buttons
        document.addEventListener('click', (e) => {
            const button = e.target.closest('.mobile-btn, .mobile-fab, .mobile-nav-btn');
            if (button && this.hapticEnabled) {
                this.triggerHaptic('light');
            }
        });
    }
    
    triggerHaptic(intensity = 'light') {
        if (!this.hapticEnabled) return;
        
        const patterns = {
            light: [10],
            medium: [20],
            heavy: [30],
            success: [10, 50, 10],
            warning: [30, 30, 30],
            error: [50, 100, 50]
        };
        
        const pattern = patterns[intensity] || patterns.light;
        navigator.vibrate(pattern);
    }
    
    // Orientation handling
    initOrientationHandler() {
        const handleOrientation = () => {
            const orientation = window.orientation || 0;
            document.body.dataset.orientation = 
                Math.abs(orientation) === 90 ? 'landscape' : 'portrait';
        };
        
        window.addEventListener('orientationchange', handleOrientation);
        handleOrientation();
    }
    
    // Network status
    initNetworkStatus() {
        const updateNetworkStatus = () => {
            const isOnline = navigator.onLine;
            document.body.dataset.networkStatus = isOnline ? 'online' : 'offline';
            
            // Show notification
            if (!isOnline) {
                this.showToast('You are offline', 'warning');
            }
        };
        
        window.addEventListener('online', updateNetworkStatus);
        window.addEventListener('offline', updateNetworkStatus);
        updateNetworkStatus();
    }
    
    // Biometric authentication
    initBiometrics() {
        if ('credentials' in navigator) {
            this.biometricsAvailable = true;
        }
    }
    
    async authenticateWithBiometrics() {
        if (!this.biometricsAvailable) {
            return false;
        }
        
        try {
            const credential = await navigator.credentials.get({
                publicKey: {
                    challenge: new Uint8Array(32),
                    timeout: 60000,
                    userVerification: "preferred"
                }
            });
            
            return credential !== null;
        } catch (error) {
            console.error('Biometric authentication failed:', error);
            return false;
        }
    }
    
    // Navigation
    navigateToPage(page) {
        // Add page transition animation
        const content = document.querySelector('.mobile-content');
        content.classList.add('mobile-fade-out');
        
        setTimeout(() => {
            window.location.href = page;
        }, 300);
    }
    
    // Utility functions
    createRipple(element, touch) {
        const ripple = document.createElement('span');
        ripple.className = 'mobile-ripple';
        
        const rect = element.getBoundingClientRect();
        const size = Math.max(rect.width, rect.height);
        const x = touch.clientX - rect.left - size / 2;
        const y = touch.clientY - rect.top - size / 2;
        
        ripple.style.width = ripple.style.height = size + 'px';
        ripple.style.left = x + 'px';
        ripple.style.top = y + 'px';
        
        element.appendChild(ripple);
        
        setTimeout(() => {
            ripple.remove();
        }, 600);
    }
    
    showToast(message, type = 'info', duration = 3000) {
        const toast = document.createElement('div');
        toast.className = `mobile-toast mobile-toast-${type}`;
        toast.textContent = message;
        
        document.body.appendChild(toast);
        
        // Animate in
        setTimeout(() => {
            toast.classList.add('show');
        }, 10);
        
        // Remove after duration
        setTimeout(() => {
            toast.classList.remove('show');
            setTimeout(() => {
                toast.remove();
            }, 300);
        }, duration);
    }
    
    // Event listeners
    setupEventListeners() {
        // Handle form submissions
        document.addEventListener('submit', (e) => {
            const form = e.target;
            if (form.classList.contains('mobile-form')) {
                e.preventDefault();
                this.handleFormSubmit(form);
            }
        });
        
        // Handle tab switching
        document.addEventListener('click', (e) => {
            const tab = e.target.closest('.mobile-tab');
            if (tab) {
                this.switchTab(tab);
            }
        });
        
        // Handle toggle switches
        document.addEventListener('click', (e) => {
            const toggle = e.target.closest('.mobile-toggle');
            if (toggle) {
                toggle.classList.toggle('active');
                const event = new CustomEvent('toggle', {
                    detail: { active: toggle.classList.contains('active') }
                });
                toggle.dispatchEvent(event);
            }
        });
    }
    
    handleFormSubmit(form) {
        const formData = new FormData(form);
        const submitBtn = form.querySelector('[type="submit"]');
        
        // Disable submit button
        if (submitBtn) {
            submitBtn.disabled = true;
            submitBtn.innerHTML = '<div class="mobile-spinner-small"></div>';
        }
        
        // Submit form (implement actual submission logic)
        console.log('Form submitted:', Object.fromEntries(formData));
        
        // Show success (demo)
        setTimeout(() => {
            this.showToast('Saved successfully', 'success');
            if (submitBtn) {
                submitBtn.disabled = false;
                submitBtn.textContent = 'Submit';
            }
        }, 1000);
    }
    
    switchTab(selectedTab) {
        const tabGroup = selectedTab.parentElement;
        const tabs = tabGroup.querySelectorAll('.mobile-tab');
        const tabContents = document.querySelectorAll(`[data-tab-group="${tabGroup.dataset.tabGroup}"]`);
        
        // Update active tab
        tabs.forEach(tab => tab.classList.remove('active'));
        selectedTab.classList.add('active');
        
        // Update content
        const targetContent = selectedTab.dataset.tabTarget;
        tabContents.forEach(content => {
            content.style.display = content.id === targetContent ? 'block' : 'none';
        });
        
        // Haptic feedback
        if (this.hapticEnabled) {
            this.triggerHaptic('light');
        }
    }
    
    // Animation initialization
    initAnimations() {
        // Observe elements for animations
        const observer = new IntersectionObserver((entries) => {
            entries.forEach(entry => {
                if (entry.isIntersecting) {
                    entry.target.classList.add('mobile-animated');
                }
            });
        }, { threshold: 0.1 });
        
        // Observe all animatable elements
        document.querySelectorAll('.mobile-animate').forEach(el => {
            observer.observe(el);
        });
    }
    
    // Public API
    static getInstance() {
        if (!window.mobileApp) {
            window.mobileApp = new MobileApp();
        }
        return window.mobileApp;
    }
}

// Initialize on DOM ready
if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', () => {
        MobileApp.getInstance();
    });
} else {
    MobileApp.getInstance();
}

// Export for use in other modules
window.MobileApp = MobileApp;