// Temporary fix for stuck onboarding tour
document.addEventListener('DOMContentLoaded', function() {
    // Add escape key handler to dismiss any onboarding modal
    document.addEventListener('keydown', function(e) {
        if (e.key === 'Escape') {
            // Remove welcome modal
            const welcomeModal = document.querySelector('.onboarding-welcome');
            if (welcomeModal) {
                welcomeModal.remove();
            }
            
            // Remove overlay
            const overlay = document.querySelector('.onboarding-overlay');
            if (overlay) {
                overlay.classList.remove('active');
            }
            
            // Remove tooltip
            const tooltip = document.querySelector('.onboarding-tooltip');
            if (tooltip) {
                tooltip.classList.remove('active');
            }
            
            // Mark tour as skipped
            localStorage.setItem('onboardingCompleted', 'true');
        }
    });
    
    // Add a close button to any existing modal
    setTimeout(function() {
        const welcomeModal = document.querySelector('.onboarding-welcome');
        if (welcomeModal && !welcomeModal.querySelector('.close-x')) {
            const closeBtn = document.createElement('button');
            closeBtn.className = 'close-x';
            closeBtn.innerHTML = '×';
            closeBtn.style.cssText = `
                position: absolute;
                top: 15px;
                right: 15px;
                background: none;
                border: none;
                font-size: 32px;
                color: #999;
                cursor: pointer;
                width: 40px;
                height: 40px;
                display: flex;
                align-items: center;
                justify-content: center;
                border-radius: 50%;
                transition: all 0.2s ease;
            `;
            closeBtn.onmouseover = function() {
                this.style.background = '#f0f0f0';
                this.style.color = '#333';
            };
            closeBtn.onmouseout = function() {
                this.style.background = 'none';
                this.style.color = '#999';
            };
            closeBtn.onclick = function() {
                welcomeModal.remove();
                const overlay = document.querySelector('.onboarding-overlay');
                if (overlay) overlay.classList.remove('active');
                localStorage.setItem('onboardingCompleted', 'true');
            };
            welcomeModal.appendChild(closeBtn);
        }
    }, 100);
});

// Also add a global function to force close the tour
window.forceCloseTour = function() {
    // Remove all tour elements
    const elements = ['.onboarding-welcome', '.onboarding-overlay', '.onboarding-tooltip'];
    elements.forEach(selector => {
        const el = document.querySelector(selector);
        if (el) {
            if (selector === '.onboarding-overlay' || selector === '.onboarding-tooltip') {
                el.classList.remove('active');
            } else {
                el.remove();
            }
        }
    });
    
    // Mark as completed
    localStorage.setItem('onboardingCompleted', 'true');
    
    console.log('Tour closed successfully');
};