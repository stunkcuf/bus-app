// Fix for popup elements appearing at bottom of page
(function() {
    'use strict';
    
    // Wait for DOM to be ready
    document.addEventListener('DOMContentLoaded', function() {
        console.log('Fixing popup positioning...');
        
        // Fix notification containers
        fixNotificationContainer();
        
        // Fix loading overlays
        fixLoadingOverlay();
        
        // Fix session timeout modal
        fixSessionTimeoutModal();
        
        // Fix onboarding tooltips
        fixOnboardingTooltips();
        
        // Fix progress tracker
        fixProgressTracker();
        
        // Add global styles to ensure popups work correctly
        addGlobalPopupStyles();
    });
    
    function fixNotificationContainer() {
        // Find all notification-related elements
        const notifications = document.querySelectorAll(
            '.notification-container, .notification-toast, .realtime-notification'
        );
        
        notifications.forEach(el => {
            if (el && !el.style.position) {
                el.style.position = 'fixed';
                el.style.zIndex = '10000';
                el.style.top = '20px';
                el.style.right = '20px';
            }
        });
    }
    
    function fixLoadingOverlay() {
        const loadingElements = document.querySelectorAll(
            '.loading-overlay, .global-loading-overlay, .loading-indicator'
        );
        
        loadingElements.forEach(el => {
            if (el) {
                el.style.position = 'fixed';
                el.style.top = '0';
                el.style.left = '0';
                el.style.width = '100%';
                el.style.height = '100%';
                el.style.zIndex = '9999';
                el.style.display = 'none'; // Hide by default
            }
        });
    }
    
    function fixSessionTimeoutModal() {
        const modals = document.querySelectorAll(
            '.session-timeout-modal, .session-warning, .session-timeout-warning'
        );
        
        modals.forEach(el => {
            if (el) {
                el.style.position = 'fixed';
                el.style.top = '50%';
                el.style.left = '50%';
                el.style.transform = 'translate(-50%, -50%)';
                el.style.zIndex = '10001';
                el.style.display = 'none'; // Hide by default
            }
        });
    }
    
    function fixOnboardingTooltips() {
        const tooltips = document.querySelectorAll(
            '.onboarding-tooltip, .onboarding-overlay, .tour-tooltip'
        );
        
        tooltips.forEach(el => {
            if (el) {
                if (el.classList.contains('onboarding-overlay')) {
                    el.style.position = 'fixed';
                    el.style.top = '0';
                    el.style.left = '0';
                    el.style.width = '100%';
                    el.style.height = '100%';
                    el.style.zIndex = '9998';
                } else {
                    el.style.position = 'absolute';
                    el.style.zIndex = '10000';
                }
                el.style.display = 'none'; // Hide by default
            }
        });
    }
    
    function fixProgressTracker() {
        const trackers = document.querySelectorAll(
            '.progress-tracker, .progress-widget, .progress-indicator'
        );
        
        trackers.forEach(el => {
            if (el) {
                el.style.position = 'fixed';
                el.style.bottom = '20px';
                el.style.right = '20px';
                el.style.zIndex = '1000';
                el.style.display = 'none'; // Hide by default
            }
        });
    }
    
    function addGlobalPopupStyles() {
        const style = document.createElement('style');
        style.textContent = `
            /* Ensure all popup elements are properly positioned */
            [class*="notification-"], 
            [class*="loading-"], 
            [class*="session-"], 
            [class*="onboarding-"],
            [class*="progress-"] {
                box-sizing: border-box;
            }
            
            /* Hide elements that shouldn't be visible initially */
            .loading-overlay:not(.active),
            .session-timeout-modal:not(.active),
            .onboarding-overlay:not(.active),
            .progress-widget:not(.active) {
                display: none !important;
            }
            
            /* Ensure popups appear above everything else */
            .notification-container {
                position: fixed !important;
                top: 20px !important;
                right: 20px !important;
                z-index: 10000 !important;
                max-width: 400px;
            }
            
            /* Fix for elements appearing at bottom */
            body > div[style*="position: relative"] {
                position: fixed !important;
            }
            
            /* Remove any test/demo content */
            .activity-item:has(.activity-title:contains("New driver registered")),
            .activity-item:has(.activity-title:contains("Jane Smith")) {
                display: none;
            }
        `;
        document.head.appendChild(style);
    }
    
    // Monitor for dynamically added elements
    const observer = new MutationObserver(function(mutations) {
        mutations.forEach(function(mutation) {
            mutation.addedNodes.forEach(function(node) {
                if (node.nodeType === 1) { // Element node
                    // Check if it's a popup element
                    const className = node.className || '';
                    if (className.includes('notification') || 
                        className.includes('loading') || 
                        className.includes('session') ||
                        className.includes('onboarding') ||
                        className.includes('progress')) {
                        
                        // Apply fixes
                        setTimeout(() => {
                            fixNotificationContainer();
                            fixLoadingOverlay();
                            fixSessionTimeoutModal();
                            fixOnboardingTooltips();
                            fixProgressTracker();
                        }, 100);
                    }
                }
            });
        });
    });
    
    // Start observing
    observer.observe(document.body, {
        childList: true,
        subtree: true
    });
    
    console.log('Popup positioning fixes applied');
})();