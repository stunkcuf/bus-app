// Production-safe logging utility
// Include this file before any other scripts that use logging

window.Logger = (function() {
    // Check if we're in production (can be set by server template)
    const isProduction = window.APP_ENV === 'production';
    
    return {
        log: isProduction ? function() {} : console.log.bind(console),
        error: console.error.bind(console), // Always log errors  
        warn: isProduction ? function() {} : console.warn.bind(console),
        debug: isProduction ? function() {} : console.debug.bind(console),
        info: isProduction ? function() {} : console.info.bind(console)
    };
})();

// Also create a short alias for convenience
window.devLog = window.Logger;