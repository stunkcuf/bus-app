// Touch-Friendly Interactions
// =========================
// Enhanced touch support for mobile and tablet devices

class TouchFriendlySystem {
    constructor() {
        this.touchStartX = 0;
        this.touchStartY = 0;
        this.isTouchDevice = 'ontouchstart' in window;
        
        this.init();
    }
    
    init() {
        // Add touch class to body
        if (this.isTouchDevice) {
            document.body.classList.add('touch-device');
        }
        
        // Initialize touch enhancements
        this.enhanceButtons();
        this.enhanceForms();
        this.setupSwipeActions();
        this.preventDoubleTapZoom();
        this.addRippleEffect();
        this.enhanceDropdowns();
    }
    
    enhanceButtons() {
        // Add haptic feedback for buttons (if supported)
        document.addEventListener('click', (e) => {
            const button = e.target.closest('button, .btn, a.action-card');
            if (button && 'vibrate' in navigator) {
                navigator.vibrate(10); // Small haptic feedback
            }
        });
        
        // Prevent accidental double clicks
        let lastClickTime = 0;
        document.addEventListener('click', (e) => {
            const button = e.target.closest('button[type="submit"], .btn-primary');
            if (button) {
                const currentTime = new Date().getTime();
                if (currentTime - lastClickTime < 500) {
                    e.preventDefault();
                    e.stopPropagation();
                    return false;
                }
                lastClickTime = currentTime;
            }
        }, true);
    }
    
    enhanceForms() {
        // Auto-advance to next field on mobile
        const inputs = document.querySelectorAll('input[type="tel"], input[type="number"]');
        inputs.forEach((input, index) => {
            input.addEventListener('input', function() {
                if (this.value.length === this.maxLength && index < inputs.length - 1) {
                    inputs[index + 1].focus();
                }
            });
        });
        
        // Enhanced select dropdowns for touch
        const selects = document.querySelectorAll('select');
        selects.forEach(select => {
            // Add touch-friendly wrapper
            const wrapper = document.createElement('div');
            wrapper.className = 'select-touch-wrapper';
            select.parentNode.insertBefore(wrapper, select);
            wrapper.appendChild(select);
            
            // Add dropdown icon
            const icon = document.createElement('i');
            icon.className = 'bi bi-chevron-down select-icon';
            wrapper.appendChild(icon);
        });
    }
    
    setupSwipeActions() {
        // Add swipe-to-delete for table rows
        const swipeableElements = document.querySelectorAll('.swipeable-row, .table-card');
        
        swipeableElements.forEach(element => {
            let startX = 0;
            let currentX = 0;
            let startTime = 0;
            
            element.addEventListener('touchstart', (e) => {
                startX = e.touches[0].clientX;
                startTime = Date.now();
            }, { passive: true });
            
            element.addEventListener('touchmove', (e) => {
                currentX = e.touches[0].clientX;
                const diff = startX - currentX;
                
                // Only swipe if horizontal movement is significant
                if (Math.abs(diff) > 50) {
                    e.preventDefault();
                    
                    if (diff > 0) {
                        // Swiping left - show actions
                        element.style.transform = `translateX(-${Math.min(diff, 100)}px)`;
                    }
                }
            });
            
            element.addEventListener('touchend', (e) => {
                const diff = startX - currentX;
                const timeElapsed = Date.now() - startTime;
                
                if (Math.abs(diff) > 100 && timeElapsed < 300) {
                    // Quick swipe - show actions
                    element.classList.add('swiped');
                } else {
                    // Reset position
                    element.style.transform = '';
                    element.classList.remove('swiped');
                }
            });
        });
    }
    
    preventDoubleTapZoom() {
        // Prevent double-tap zoom on iOS
        let lastTouchEnd = 0;
        document.addEventListener('touchend', (e) => {
            const now = Date.now();
            if (now - lastTouchEnd <= 300) {
                e.preventDefault();
            }
            lastTouchEnd = now;
        }, false);
    }
    
    addRippleEffect() {
        // Material Design-like ripple effect
        document.addEventListener('click', function(e) {
            const button = e.target.closest('.btn, button, .action-card');
            if (!button) return;
            
            const ripple = document.createElement('span');
            ripple.className = 'ripple';
            
            const rect = button.getBoundingClientRect();
            const size = Math.max(rect.width, rect.height);
            const x = e.clientX - rect.left - size / 2;
            const y = e.clientY - rect.top - size / 2;
            
            ripple.style.width = ripple.style.height = size + 'px';
            ripple.style.left = x + 'px';
            ripple.style.top = y + 'px';
            
            button.style.position = 'relative';
            button.style.overflow = 'hidden';
            button.appendChild(ripple);
            
            setTimeout(() => {
                ripple.remove();
            }, 600);
        });
        
        // Add ripple styles
        const style = document.createElement('style');
        style.textContent = `
            .ripple {
                position: absolute;
                border-radius: 50%;
                transform: scale(0);
                animation: ripple 0.6s ease-out;
                background: rgba(255, 255, 255, 0.5);
                pointer-events: none;
            }
            
            @keyframes ripple {
                to {
                    transform: scale(4);
                    opacity: 0;
                }
            }
        `;
        document.head.appendChild(style);
    }
    
    enhanceDropdowns() {
        // Make dropdowns more touch-friendly
        const dropdowns = document.querySelectorAll('.dropdown');
        
        dropdowns.forEach(dropdown => {
            const toggle = dropdown.querySelector('.dropdown-toggle');
            const menu = dropdown.querySelector('.dropdown-menu');
            
            if (toggle && menu) {
                // Prevent dropdown from closing on touch inside
                menu.addEventListener('touchend', (e) => {
                    e.stopPropagation();
                });
                
                // Close dropdown when touching outside
                document.addEventListener('touchstart', (e) => {
                    if (!dropdown.contains(e.target)) {
                        dropdown.classList.remove('show');
                        menu.classList.remove('show');
                    }
                });
            }
        });
    }
    
    // Helper method to create floating action button
    createFAB(options = {}) {
        const {
            icon = 'bi-plus',
            onClick = () => {},
            position = { bottom: '24px', right: '24px' }
        } = options;
        
        const fab = document.createElement('button');
        fab.className = 'fab';
        fab.innerHTML = `<i class="bi ${icon}"></i>`;
        fab.style.bottom = position.bottom;
        fab.style.right = position.right;
        
        fab.addEventListener('click', onClick);
        
        document.body.appendChild(fab);
        return fab;
    }
    
    // Helper to create segmented control
    createSegmentedControl(options = {}) {
        const {
            container,
            options: controlOptions = [],
            onChange = () => {}
        } = options;
        
        const control = document.createElement('div');
        control.className = 'segmented-control';
        
        controlOptions.forEach((option, index) => {
            const button = document.createElement('button');
            button.className = 'btn';
            button.textContent = option.label;
            button.dataset.value = option.value;
            
            if (index === 0) {
                button.classList.add('active');
            }
            
            button.addEventListener('click', function() {
                control.querySelectorAll('.btn').forEach(b => b.classList.remove('active'));
                this.classList.add('active');
                onChange(option.value);
            });
            
            control.appendChild(button);
        });
        
        if (container) {
            container.appendChild(control);
        }
        
        return control;
    }
    
    // Long press detection
    addLongPress(element, callback, duration = 500) {
        let pressTimer;
        let longPressTriggered = false;
        
        const start = (e) => {
            longPressTriggered = false;
            pressTimer = setTimeout(() => {
                longPressTriggered = true;
                callback(e);
                
                // Haptic feedback
                if ('vibrate' in navigator) {
                    navigator.vibrate(50);
                }
            }, duration);
        };
        
        const cancel = () => {
            clearTimeout(pressTimer);
        };
        
        const click = (e) => {
            if (longPressTriggered) {
                e.preventDefault();
                e.stopPropagation();
            }
        };
        
        element.addEventListener('touchstart', start);
        element.addEventListener('mousedown', start);
        element.addEventListener('touchend', cancel);
        element.addEventListener('mouseup', cancel);
        element.addEventListener('touchmove', cancel);
        element.addEventListener('mousemove', cancel);
        element.addEventListener('click', click);
    }
}

// Initialize touch-friendly system
const touchFriendly = new TouchFriendlySystem();

// Export for use in other modules
window.touchFriendly = touchFriendly;