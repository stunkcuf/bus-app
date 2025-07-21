// User-Friendly Error Messages
// ============================
// Converts technical errors into clear, helpful messages for non-technical users

class FriendlyErrorSystem {
  constructor() {
    this.errorMappings = this.initializeErrorMappings();
    this.init();
  }

  init() {
    // Add styles
    this.addStyles();
    
    // Create error container
    this.createErrorContainer();
    
    // Intercept errors
    this.interceptErrors();
  }

  initializeErrorMappings() {
    return {
      // Network errors
      'NetworkError': {
        title: 'Connection Problem',
        message: 'We\'re having trouble connecting to the server. Please check your internet connection and try again.',
        icon: 'bi-wifi-off',
        suggestions: [
          'Check if you\'re connected to the internet',
          'Try refreshing the page',
          'Wait a moment and try again'
        ]
      },
      
      // Database errors
      'duplicate key': {
        title: 'Item Already Exists',
        message: 'This item already exists in the system. Please use a different ID or name.',
        icon: 'bi-exclamation-diamond',
        suggestions: [
          'Check if the item was already added',
          'Use a different ID number',
          'Contact support if you believe this is an error'
        ]
      },
      
      'foreign key': {
        title: 'Cannot Complete Action',
        message: 'This item is being used elsewhere in the system and cannot be removed.',
        icon: 'bi-link-45deg',
        suggestions: [
          'Check if this item is assigned to routes or students',
          'Remove assignments before deleting',
          'Contact your manager for help'
        ]
      },
      
      // Validation errors
      'required field': {
        title: 'Missing Information',
        message: 'Please fill in all required fields before continuing.',
        icon: 'bi-pencil-square',
        suggestions: [
          'Look for fields marked with a red asterisk (*)',
          'Make sure all information is complete',
          'Check for any highlighted fields'
        ]
      },
      
      'invalid format': {
        title: 'Incorrect Format',
        message: 'The information doesn\'t look quite right. Please check the format.',
        icon: 'bi-input-cursor-text',
        suggestions: [
          'Phone numbers should be like: 555-1234',
          'Dates should be: MM/DD/YYYY',
          'Email should include @ symbol'
        ]
      },
      
      // Permission errors
      'unauthorized': {
        title: 'Access Denied',
        message: 'You don\'t have permission to do this. Please contact your manager.',
        icon: 'bi-shield-x',
        suggestions: [
          'Make sure you\'re logged in',
          'Check if your account is active',
          'Ask your manager for access'
        ]
      },
      
      // Session errors
      'session expired': {
        title: 'Session Timed Out',
        message: 'You\'ve been logged out for security. Please log in again.',
        icon: 'bi-clock-history',
        suggestions: [
          'Click here to go to login page',
          'Your work was saved automatically',
          'This happens after 24 hours of inactivity'
        ]
      },
      
      // File errors
      'file too large': {
        title: 'File Too Big',
        message: 'The file you\'re trying to upload is too large. Maximum size is 10MB.',
        icon: 'bi-file-earmark-x',
        suggestions: [
          'Try compressing the file',
          'Split large Excel files into smaller ones',
          'Remove unnecessary data or images'
        ]
      },
      
      'invalid file type': {
        title: 'Wrong File Type',
        message: 'Please upload an Excel file (.xlsx or .xls).',
        icon: 'bi-file-earmark-excel',
        suggestions: [
          'Save your file as Excel format',
          'File should end with .xlsx or .xls',
          'Don\'t use .csv or .txt files'
        ]
      },
      
      // Generic errors
      '404': {
        title: 'Page Not Found',
        message: 'We couldn\'t find the page you\'re looking for.',
        icon: 'bi-search',
        suggestions: [
          'Check if the address is correct',
          'Go back to the dashboard',
          'Use the menu to navigate'
        ]
      },
      
      '500': {
        title: 'Something Went Wrong',
        message: 'We encountered an unexpected problem. Our team has been notified.',
        icon: 'bi-tools',
        suggestions: [
          'Try refreshing the page',
          'Your data should be safe',
          'Contact support if this continues'
        ]
      }
    };
  }

  addStyles() {
    const style = document.createElement('style');
    style.textContent = `
      .friendly-error-container {
        position: fixed;
        top: 20px;
        right: 20px;
        max-width: 400px;
        z-index: 10000;
      }
      
      .friendly-error {
        background: var(--color-bg-primary);
        border: 2px solid var(--color-danger);
        border-radius: var(--border-radius-lg);
        box-shadow: var(--shadow-xl);
        margin-bottom: var(--space-md);
        animation: slideIn 0.3s ease;
        overflow: hidden;
      }
      
      .friendly-error-header {
        background: var(--color-danger);
        color: var(--color-text-inverse);
        padding: var(--space-md) var(--space-lg);
        display: flex;
        align-items: center;
        justify-content: space-between;
      }
      
      .friendly-error-title {
        display: flex;
        align-items: center;
        gap: var(--space-sm);
        font-size: var(--font-size-lg);
        font-weight: 600;
      }
      
      .friendly-error-close {
        background: none;
        border: none;
        color: var(--color-text-inverse);
        font-size: 1.5rem;
        cursor: pointer;
        padding: var(--space-xs);
        border-radius: var(--border-radius-base);
        transition: background var(--transition-base);
      }
      
      .friendly-error-close:hover {
        background: rgba(255, 255, 255, 0.1);
      }
      
      .friendly-error-body {
        padding: var(--space-lg);
      }
      
      .friendly-error-message {
        font-size: var(--font-size-base);
        line-height: 1.6;
        color: var(--color-text-primary);
        margin-bottom: var(--space-md);
      }
      
      .friendly-error-suggestions {
        background: var(--color-bg-tertiary);
        border-radius: var(--border-radius-base);
        padding: var(--space-md);
        margin-bottom: var(--space-md);
      }
      
      .friendly-error-suggestions-title {
        font-weight: 600;
        margin-bottom: var(--space-sm);
        display: flex;
        align-items: center;
        gap: var(--space-xs);
      }
      
      .friendly-error-suggestions ul {
        margin: 0;
        padding-left: var(--space-lg);
        list-style: none;
      }
      
      .friendly-error-suggestions li {
        position: relative;
        padding-left: var(--space-md);
        margin-bottom: var(--space-xs);
      }
      
      .friendly-error-suggestions li:before {
        content: "→";
        position: absolute;
        left: 0;
        color: var(--color-primary);
      }
      
      .friendly-error-actions {
        display: flex;
        gap: var(--space-sm);
      }
      
      .friendly-error-details {
        margin-top: var(--space-md);
        padding-top: var(--space-md);
        border-top: 1px solid var(--border-color);
      }
      
      .friendly-error-details-toggle {
        background: none;
        border: none;
        color: var(--color-primary);
        cursor: pointer;
        font-size: var(--font-size-small);
        display: flex;
        align-items: center;
        gap: var(--space-xs);
      }
      
      .friendly-error-details-content {
        display: none;
        margin-top: var(--space-sm);
        padding: var(--space-sm);
        background: var(--color-bg-tertiary);
        border-radius: var(--border-radius-base);
        font-family: monospace;
        font-size: var(--font-size-small);
        color: var(--color-text-secondary);
        max-height: 200px;
        overflow-y: auto;
      }
      
      .friendly-error-details-content.show {
        display: block;
      }
      
      @keyframes slideIn {
        from {
          transform: translateX(100%);
          opacity: 0;
        }
        to {
          transform: translateX(0);
          opacity: 1;
        }
      }
      
      /* Inline error messages */
      .field-error {
        background: var(--color-danger-light);
        border: 1px solid var(--color-danger);
        color: var(--color-danger-dark);
        padding: var(--space-sm) var(--space-md);
        border-radius: var(--border-radius-base);
        margin-top: var(--space-xs);
        font-size: var(--font-size-base);
        display: flex;
        align-items: center;
        gap: var(--space-sm);
      }
      
      .field-error i {
        font-size: 1.2rem;
      }
    `;
    document.head.appendChild(style);
  }

  createErrorContainer() {
    const container = document.createElement('div');
    container.className = 'friendly-error-container';
    container.id = 'friendly-error-container';
    document.body.appendChild(container);
  }

  interceptErrors() {
    // Override window.onerror
    const originalOnError = window.onerror;
    window.onerror = (message, source, lineno, colno, error) => {
      this.handleError(error || new Error(message));
      if (originalOnError) {
        return originalOnError(message, source, lineno, colno, error);
      }
      return true;
    };

    // Listen for unhandled promise rejections
    window.addEventListener('unhandledrejection', (event) => {
      this.handleError(new Error(event.reason));
      event.preventDefault();
    });

    // Intercept fetch errors
    const originalFetch = window.fetch;
    window.fetch = async (...args) => {
      try {
        const response = await originalFetch(...args);
        
        if (!response.ok) {
          // Check for specific HTTP errors
          if (response.status === 404) {
            this.showError('404');
          } else if (response.status === 401) {
            this.showError('unauthorized');
          } else if (response.status === 500) {
            this.showError('500');
          } else {
            // Try to get error message from response
            try {
              const data = await response.json();
              if (data.error) {
                this.handleError(new Error(data.error));
              }
            } catch (e) {
              // Fallback to status text
              this.handleError(new Error(response.statusText));
            }
          }
        }
        
        return response;
      } catch (error) {
        if (error.name === 'TypeError' && error.message.includes('fetch')) {
          this.showError('NetworkError');
        } else {
          this.handleError(error);
        }
        throw error;
      }
    };
  }

  handleError(error) {
    const errorMessage = error.message || error.toString();
    
    // Find matching error mapping
    let errorConfig = null;
    for (const [key, config] of Object.entries(this.errorMappings)) {
      if (errorMessage.toLowerCase().includes(key.toLowerCase())) {
        errorConfig = config;
        break;
      }
    }
    
    // Default error if no match
    if (!errorConfig) {
      errorConfig = {
        title: 'Oops! Something went wrong',
        message: 'We encountered an unexpected issue. Please try again.',
        icon: 'bi-exclamation-circle',
        suggestions: [
          'Try refreshing the page',
          'Check your internet connection',
          'Contact support if the problem continues'
        ]
      };
    }
    
    this.displayError(errorConfig, error);
  }

  showError(errorKey) {
    const errorConfig = this.errorMappings[errorKey];
    if (errorConfig) {
      this.displayError(errorConfig);
    }
  }

  displayError(config, originalError = null) {
    const container = document.getElementById('friendly-error-container');
    
    const errorId = 'error-' + Date.now();
    const errorElement = document.createElement('div');
    errorElement.className = 'friendly-error';
    errorElement.id = errorId;
    
    errorElement.innerHTML = `
      <div class="friendly-error-header">
        <div class="friendly-error-title">
          <i class="bi ${config.icon}" aria-hidden="true"></i>
          ${config.title}
        </div>
        <button class="friendly-error-close" onclick="friendlyErrors.closeError('${errorId}')" aria-label="Close">
          <i class="bi bi-x" aria-hidden="true"></i>
        </button>
      </div>
      <div class="friendly-error-body">
        <p class="friendly-error-message">${config.message}</p>
        
        ${config.suggestions && config.suggestions.length > 0 ? `
          <div class="friendly-error-suggestions">
            <div class="friendly-error-suggestions-title">
              <i class="bi bi-lightbulb" aria-hidden="true"></i>
              What you can do:
            </div>
            <ul>
              ${config.suggestions.map(s => `<li>${s}</li>`).join('')}
            </ul>
          </div>
        ` : ''}
        
        <div class="friendly-error-actions">
          <button class="btn btn-primary" onclick="location.reload()">
            <i class="bi bi-arrow-clockwise" aria-hidden="true"></i>
            Refresh Page
          </button>
          <button class="btn btn-secondary" onclick="window.history.back()">
            <i class="bi bi-arrow-left" aria-hidden="true"></i>
            Go Back
          </button>
        </div>
        
        ${originalError ? `
          <div class="friendly-error-details">
            <button class="friendly-error-details-toggle" onclick="friendlyErrors.toggleDetails('${errorId}')">
              <i class="bi bi-chevron-right" aria-hidden="true"></i>
              Technical details
            </button>
            <div class="friendly-error-details-content" id="${errorId}-details">
              ${originalError.stack || originalError.message || originalError}
            </div>
          </div>
        ` : ''}
      </div>
    `;
    
    container.appendChild(errorElement);
    
    // Auto-remove after 30 seconds
    setTimeout(() => {
      this.closeError(errorId);
    }, 30000);
    
    // Log to console for debugging
    if (originalError) {
      console.error('Original error:', originalError);
    }
  }

  closeError(errorId) {
    const errorElement = document.getElementById(errorId);
    if (errorElement) {
      errorElement.style.animation = 'slideOut 0.3s ease';
      setTimeout(() => {
        errorElement.remove();
      }, 300);
    }
  }

  toggleDetails(errorId) {
    const details = document.getElementById(errorId + '-details');
    const toggle = document.querySelector(`#${errorId} .friendly-error-details-toggle i`);
    
    if (details) {
      details.classList.toggle('show');
      if (details.classList.contains('show')) {
        toggle.className = 'bi bi-chevron-down';
      } else {
        toggle.className = 'bi bi-chevron-right';
      }
    }
  }

  // Helper method for form validation
  showFieldError(field, message) {
    // Remove existing error
    const existingError = field.parentElement.querySelector('.field-error');
    if (existingError) {
      existingError.remove();
    }
    
    // Add new error
    const error = document.createElement('div');
    error.className = 'field-error';
    error.innerHTML = `
      <i class="bi bi-exclamation-circle" aria-hidden="true"></i>
      <span>${message}</span>
    `;
    
    field.parentElement.appendChild(error);
    field.setAttribute('aria-invalid', 'true');
    field.setAttribute('aria-describedby', error.id);
    
    // Focus on field
    field.focus();
  }

  clearFieldError(field) {
    const error = field.parentElement.querySelector('.field-error');
    if (error) {
      error.remove();
    }
    field.removeAttribute('aria-invalid');
    field.removeAttribute('aria-describedby');
  }
}

// Initialize friendly error system
const friendlyErrors = new FriendlyErrorSystem();

// Export for use in other scripts
window.friendlyErrors = friendlyErrors;

// Add CSS animation for slideOut
const style = document.createElement('style');
style.textContent = `
  @keyframes slideOut {
    from {
      transform: translateX(0);
      opacity: 1;
    }
    to {
      transform: translateX(100%);
      opacity: 0;
    }
  }
`;
document.head.appendChild(style);