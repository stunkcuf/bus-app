// Main application JavaScript entry point
// This file imports and initializes all core application functionality

// Import utility modules
import '../css/main.css';

// Core application initialization
document.addEventListener('DOMContentLoaded', function() {
    console.log('Fleet Management System initialized');
    
    // Initialize global app features
    if (window.AccessibleForms) {
        window.AccessibleForms.init();
    }
    
    if (window.EnhancedHelpSystem) {
        window.EnhancedHelpSystem.init();
    }
    
    // Initialize auto-save for forms
    if (window.AutoSave) {
        window.AutoSave.init();
    }
    
    // Initialize confirmation dialogs
    if (window.ConfirmationDialogs) {
        window.ConfirmationDialogs.init();
    }
    
    // Initialize loading indicators
    if (window.LoadingIndicators) {
        window.LoadingIndicators.init();
    }
    
    // Initialize responsive tables
    if (window.ResponsiveTables) {
        window.ResponsiveTables.init();
    }
    
    // Initialize form validation
    if (window.FormValidation) {
        window.FormValidation.init();
    }
});

// Export for global access
window.FleetApp = {
    version: '1.0.0',
    initialized: true
};