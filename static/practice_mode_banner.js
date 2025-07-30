// Practice Mode Banner System
// ===========================
// Shows a banner when the user is in practice mode

class PracticeModeBanner {
    constructor() {
        this.banner = null;
        this.init();
    }
    
    init() {
        // Check if practice mode is active
        if (this.isPracticeModeActive()) {
            this.createBanner();
            this.showBanner();
        }
    }
    
    isPracticeModeActive() {
        // Check for practice mode cookie
        const cookies = document.cookie.split(';');
        for (let cookie of cookies) {
            const [name, value] = cookie.trim().split('=');
            if (name === 'practice_mode' && value === 'active') {
                return true;
            }
        }
        
        // Also check URL parameter
        const urlParams = new URLSearchParams(window.location.search);
        return urlParams.get('practice') === '1';
    }
    
    createBanner() {
        // Create banner element
        this.banner = document.createElement('div');
        this.banner.className = 'practice-mode-banner';
        this.banner.innerHTML = `
            <div class="practice-banner-content">
                <div class="practice-banner-icon">
                    <i class="bi bi-mortarboard"></i>
                </div>
                <div class="practice-banner-text">
                    <strong>Practice Mode Active</strong> - You're working with sample data. Changes won't affect real records.
                </div>
                <div class="practice-banner-actions">
                    <a href="/practice-mode" class="practice-banner-link">
                        <i class="bi bi-gear"></i> Manage
                    </a>
                    <button class="practice-banner-close" onclick="practiceModeBanner.hideBanner()">
                        <i class="bi bi-x-lg"></i>
                    </button>
                </div>
            </div>
        `;
        
        // Add styles
        this.addStyles();
        
        // Add to page
        document.body.insertBefore(this.banner, document.body.firstChild);
    }
    
    addStyles() {
        const style = document.createElement('style');
        style.textContent = `
            .practice-mode-banner {
                position: fixed;
                top: 0;
                left: 0;
                right: 0;
                background: linear-gradient(135deg, #10b981 0%, #059669 100%);
                color: white;
                padding: 12px 20px;
                z-index: 10000;
                box-shadow: 0 2px 10px rgba(0, 0, 0, 0.2);
                transform: translateY(-100%);
                transition: transform 0.3s ease;
            }
            
            .practice-mode-banner.show {
                transform: translateY(0);
            }
            
            .practice-banner-content {
                max-width: 1400px;
                margin: 0 auto;
                display: flex;
                align-items: center;
                gap: 15px;
                flex-wrap: wrap;
            }
            
            .practice-banner-icon {
                font-size: 24px;
                display: flex;
                align-items: center;
            }
            
            .practice-banner-text {
                flex: 1;
                font-size: 14px;
                line-height: 1.4;
            }
            
            .practice-banner-actions {
                display: flex;
                align-items: center;
                gap: 15px;
            }
            
            .practice-banner-link {
                color: white;
                text-decoration: none;
                padding: 6px 12px;
                background: rgba(255, 255, 255, 0.2);
                border-radius: 5px;
                font-size: 14px;
                transition: all 0.2s ease;
                display: flex;
                align-items: center;
                gap: 5px;
            }
            
            .practice-banner-link:hover {
                background: rgba(255, 255, 255, 0.3);
                color: white;
                text-decoration: none;
            }
            
            .practice-banner-close {
                background: none;
                border: none;
                color: white;
                font-size: 20px;
                cursor: pointer;
                padding: 5px;
                opacity: 0.8;
                transition: opacity 0.2s ease;
            }
            
            .practice-banner-close:hover {
                opacity: 1;
            }
            
            /* Adjust page content when banner is visible */
            body.practice-mode-active {
                padding-top: 60px;
            }
            
            /* Animation for attention */
            @keyframes pulse {
                0%, 100% { transform: scale(1); }
                50% { transform: scale(1.1); }
            }
            
            .practice-banner-icon {
                animation: pulse 2s ease-in-out infinite;
            }
            
            /* Mobile responsive */
            @media (max-width: 768px) {
                .practice-banner-content {
                    font-size: 13px;
                }
                
                .practice-banner-icon {
                    font-size: 20px;
                }
                
                .practice-banner-text strong {
                    display: block;
                    margin-bottom: 2px;
                }
            }
        `;
        document.head.appendChild(style);
    }
    
    showBanner() {
        // Add class to body for spacing
        document.body.classList.add('practice-mode-active');
        
        // Show banner with animation
        setTimeout(() => {
            this.banner.classList.add('show');
        }, 100);
        
        // Check if user has hidden banner this session
        const hidden = sessionStorage.getItem('practice_banner_hidden');
        if (hidden === 'true') {
            this.hideBanner();
        }
    }
    
    hideBanner() {
        if (this.banner) {
            this.banner.classList.remove('show');
            sessionStorage.setItem('practice_banner_hidden', 'true');
        }
    }
    
    // API for checking practice mode status
    isPracticeMode() {
        return this.isPracticeModeActive();
    }
    
    // Get practice data via API
    async getPracticeData(type = 'summary') {
        try {
            const response = await fetch(`/api/practice-data?type=${type}`);
            const data = await response.json();
            return data;
        } catch (error) {
            console.error('Failed to fetch practice data:', error);
            return null;
        }
    }
}

// Initialize practice mode banner
const practiceModeBanner = new PracticeModeBanner();

// Export for global access
window.practiceModeBanner = practiceModeBanner;

// Utility function for other scripts to check practice mode
window.isPracticeMode = function() {
    return practiceModeBanner.isPracticeMode();
};