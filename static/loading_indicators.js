// Loading Indicators System
// =========================
// Provides clear visual feedback during operations

class LoadingIndicatorSystem {
  constructor() {
    this.activeIndicators = new Map();
    this.init();
  }

  init() {
    // Add styles
    this.addStyles();
    
    // Create global loading overlay
    this.createGlobalOverlay();
    
    // Intercept form submissions
    this.interceptFormSubmissions();
    
    // Intercept button clicks
    this.interceptButtonClicks();
    
    // Monitor fetch requests
    this.monitorFetchRequests();
  }

  addStyles() {
    const style = document.createElement('style');
    style.textContent = `
      /* Global loading overlay */
      .global-loading-overlay {
        position: fixed;
        top: 0;
        left: 0;
        width: 100%;
        height: 100%;
        background: rgba(0, 0, 0, 0.5);
        display: none;
        align-items: center;
        justify-content: center;
        z-index: 9999;
      }
      
      .global-loading-overlay.show {
        display: flex;
      }
      
      .loading-modal {
        background: var(--color-bg-primary);
        border-radius: var(--border-radius-lg);
        padding: var(--space-xl);
        box-shadow: var(--shadow-xl);
        text-align: center;
        min-width: 300px;
      }
      
      .loading-spinner {
        width: 60px;
        height: 60px;
        border: 5px solid var(--color-bg-tertiary);
        border-top: 5px solid var(--color-primary);
        border-radius: 50%;
        animation: spin 1s linear infinite;
        margin: 0 auto var(--space-lg);
      }
      
      .loading-text {
        font-size: var(--font-size-lg);
        color: var(--color-text-primary);
        margin-bottom: var(--space-sm);
      }
      
      .loading-subtext {
        font-size: var(--font-size-base);
        color: var(--color-text-secondary);
      }
      
      /* Button loading state */
      .btn-loading {
        position: relative;
        color: transparent !important;
        pointer-events: none;
      }
      
      .btn-loading::after {
        content: '';
        position: absolute;
        width: 20px;
        height: 20px;
        top: 50%;
        left: 50%;
        margin-left: -10px;
        margin-top: -10px;
        border: 2px solid var(--color-bg-primary);
        border-top-color: transparent;
        border-radius: 50%;
        animation: spin 0.8s linear infinite;
      }
      
      /* Inline loading indicator */
      .inline-loader {
        display: inline-flex;
        align-items: center;
        gap: var(--space-sm);
        padding: var(--space-sm) var(--space-md);
        background: var(--color-info-light);
        color: var(--color-info-dark);
        border-radius: var(--border-radius-base);
        font-size: var(--font-size-base);
      }
      
      .inline-loader-spinner {
        width: 16px;
        height: 16px;
        border: 2px solid var(--color-info);
        border-top-color: transparent;
        border-radius: 50%;
        animation: spin 0.8s linear infinite;
      }
      
      /* Table loading overlay */
      .table-loading-overlay {
        position: absolute;
        top: 0;
        left: 0;
        width: 100%;
        height: 100%;
        background: rgba(255, 255, 255, 0.9);
        display: flex;
        align-items: center;
        justify-content: center;
        z-index: 10;
      }
      
      .table-loading-content {
        text-align: center;
      }
      
      /* Progress bar loader */
      .progress-loader {
        position: fixed;
        top: 0;
        left: 0;
        width: 100%;
        height: 4px;
        background: var(--color-bg-tertiary);
        z-index: 10000;
        overflow: hidden;
        opacity: 0;
        transition: opacity 0.3s ease;
      }
      
      .progress-loader.show {
        opacity: 1;
      }
      
      .progress-loader-bar {
        height: 100%;
        background: var(--color-primary);
        animation: progress 2s ease-in-out infinite;
      }
      
      /* Skeleton loader */
      .skeleton-loader {
        background: linear-gradient(90deg, 
          var(--color-bg-tertiary) 25%, 
          var(--color-bg-secondary) 50%, 
          var(--color-bg-tertiary) 75%);
        background-size: 200% 100%;
        animation: skeleton 1.5s ease-in-out infinite;
        border-radius: var(--border-radius-base);
      }
      
      .skeleton-text {
        height: 20px;
        margin-bottom: var(--space-sm);
      }
      
      .skeleton-title {
        height: 30px;
        width: 50%;
        margin-bottom: var(--space-md);
      }
      
      .skeleton-button {
        height: 44px;
        width: 120px;
        border-radius: var(--border-radius-base);
      }
      
      /* Animations */
      @keyframes spin {
        0% { transform: rotate(0deg); }
        100% { transform: rotate(360deg); }
      }
      
      @keyframes progress {
        0% {
          transform: translateX(-100%);
        }
        50% {
          transform: translateX(0);
        }
        100% {
          transform: translateX(100%);
        }
      }
      
      @keyframes skeleton {
        0% {
          background-position: -200% 0;
        }
        100% {
          background-position: 200% 0;
        }
      }
      
      /* Toast notifications for completion */
      .loading-toast {
        position: fixed;
        bottom: 20px;
        left: 50%;
        transform: translateX(-50%) translateY(100px);
        background: var(--color-success);
        color: var(--color-text-inverse);
        padding: var(--space-md) var(--space-lg);
        border-radius: var(--border-radius-lg);
        box-shadow: var(--shadow-lg);
        display: flex;
        align-items: center;
        gap: var(--space-sm);
        opacity: 0;
        transition: all 0.3s ease;
        z-index: 10001;
      }
      
      .loading-toast.show {
        transform: translateX(-50%) translateY(0);
        opacity: 1;
      }
      
      .loading-toast.error {
        background: var(--color-danger);
      }
    `;
    document.head.appendChild(style);
  }

  createGlobalOverlay() {
    const overlay = document.createElement('div');
    overlay.className = 'global-loading-overlay';
    overlay.id = 'global-loading-overlay';
    overlay.innerHTML = `
      <div class="loading-modal">
        <div class="loading-spinner"></div>
        <div class="loading-text">Please Wait</div>
        <div class="loading-subtext" id="loading-message">Processing your request...</div>
      </div>
    `;
    document.body.appendChild(overlay);
    
    // Create progress bar
    const progressBar = document.createElement('div');
    progressBar.className = 'progress-loader';
    progressBar.id = 'progress-loader';
    progressBar.innerHTML = '<div class="progress-loader-bar"></div>';
    document.body.appendChild(progressBar);
  }

  interceptFormSubmissions() {
    document.addEventListener('submit', (e) => {
      const form = e.target;
      
      // Skip if form has no-loading attribute
      if (form.hasAttribute('data-no-loading')) return;
      
      // Get custom loading message
      const loadingMessage = form.getAttribute('data-loading-message') || 'Saving your information...';
      
      // Show loading on submit button
      const submitBtn = form.querySelector('button[type="submit"]');
      if (submitBtn) {
        this.showButtonLoading(submitBtn);
      }
      
      // Show global loading for important forms
      if (form.classList.contains('important-form') || form.hasAttribute('data-show-overlay')) {
        this.showGlobalLoading(loadingMessage);
      }
    });
  }

  interceptButtonClicks() {
    document.addEventListener('click', (e) => {
      const button = e.target.closest('button, .btn');
      
      if (button && !button.hasAttribute('data-no-loading')) {
        // Check if it's an action button
        const isAction = button.hasAttribute('data-action') || 
                        button.onclick || 
                        (button.type === 'submit' && button.form);
        
        if (isAction) {
          const loadingText = button.getAttribute('data-loading-text') || 'Processing...';
          
          // For AJAX actions
          if (button.hasAttribute('data-ajax')) {
            this.showButtonLoading(button, loadingText);
          }
        }
      }
    });
  }

  monitorFetchRequests() {
    // Track active fetch requests
    let activeRequests = 0;
    
    const originalFetch = window.fetch;
    window.fetch = async (...args) => {
      activeRequests++;
      
      // Show progress bar for multiple requests
      if (activeRequests === 1) {
        this.showProgressBar();
      }
      
      try {
        const response = await originalFetch(...args);
        return response;
      } finally {
        activeRequests--;
        if (activeRequests === 0) {
          this.hideProgressBar();
        }
      }
    };
  }

  showGlobalLoading(message = 'Processing...') {
    const overlay = document.getElementById('global-loading-overlay');
    const messageEl = document.getElementById('loading-message');
    
    if (messageEl) {
      messageEl.textContent = message;
    }
    
    overlay.classList.add('show');
    
    // Store indicator
    const id = 'global-' + Date.now();
    this.activeIndicators.set(id, { type: 'global', element: overlay });
    
    return id;
  }

  hideGlobalLoading(id) {
    const overlay = document.getElementById('global-loading-overlay');
    overlay.classList.remove('show');
    
    if (id) {
      this.activeIndicators.delete(id);
    }
  }

  showButtonLoading(button, text = null) {
    // Store original content
    button.dataset.originalContent = button.innerHTML;
    
    // Add loading class
    button.classList.add('btn-loading');
    button.disabled = true;
    
    if (text) {
      button.textContent = text;
    }
    
    // Store indicator
    const id = 'button-' + Date.now();
    this.activeIndicators.set(id, { type: 'button', element: button });
    
    return id;
  }

  hideButtonLoading(button) {
    if (button.dataset.originalContent) {
      button.innerHTML = button.dataset.originalContent;
      delete button.dataset.originalContent;
    }
    
    button.classList.remove('btn-loading');
    button.disabled = false;
  }

  showTableLoading(table) {
    const wrapper = table.closest('.table-responsive') || table.parentElement;
    wrapper.style.position = 'relative';
    
    const overlay = document.createElement('div');
    overlay.className = 'table-loading-overlay';
    overlay.innerHTML = `
      <div class="table-loading-content">
        <div class="loading-spinner"></div>
        <div class="loading-text">Loading data...</div>
      </div>
    `;
    
    wrapper.appendChild(overlay);
    
    const id = 'table-' + Date.now();
    this.activeIndicators.set(id, { type: 'table', element: overlay });
    
    return id;
  }

  hideTableLoading(id) {
    const indicator = this.activeIndicators.get(id);
    if (indicator && indicator.type === 'table') {
      indicator.element.remove();
      this.activeIndicators.delete(id);
    }
  }

  showInlineLoader(container, text = 'Loading...') {
    const loader = document.createElement('div');
    loader.className = 'inline-loader';
    loader.innerHTML = `
      <div class="inline-loader-spinner"></div>
      <span>${text}</span>
    `;
    
    container.appendChild(loader);
    
    const id = 'inline-' + Date.now();
    this.activeIndicators.set(id, { type: 'inline', element: loader });
    
    return id;
  }

  hideInlineLoader(id) {
    const indicator = this.activeIndicators.get(id);
    if (indicator && indicator.type === 'inline') {
      indicator.element.remove();
      this.activeIndicators.delete(id);
    }
  }

  showProgressBar() {
    const progressBar = document.getElementById('progress-loader');
    progressBar.classList.add('show');
  }

  hideProgressBar() {
    const progressBar = document.getElementById('progress-loader');
    progressBar.classList.remove('show');
  }

  showToast(message, type = 'success', duration = 3000) {
    const toast = document.createElement('div');
    toast.className = `loading-toast ${type}`;
    toast.innerHTML = `
      <i class="bi bi-${type === 'success' ? 'check-circle' : 'x-circle'}" aria-hidden="true"></i>
      <span>${message}</span>
    `;
    
    document.body.appendChild(toast);
    
    // Show toast
    setTimeout(() => toast.classList.add('show'), 10);
    
    // Hide and remove
    setTimeout(() => {
      toast.classList.remove('show');
      setTimeout(() => toast.remove(), 300);
    }, duration);
  }

  // Create skeleton loaders for dynamic content
  createSkeletonLoader(type = 'text', count = 3) {
    const container = document.createElement('div');
    container.className = 'skeleton-container';
    
    for (let i = 0; i < count; i++) {
      const skeleton = document.createElement('div');
      skeleton.className = `skeleton-loader skeleton-${type}`;
      container.appendChild(skeleton);
    }
    
    return container;
  }

  // Helper method for AJAX operations
  async withLoading(asyncFunction, options = {}) {
    const {
      showGlobal = false,
      message = 'Processing...',
      successMessage = 'Operation completed successfully',
      errorMessage = 'An error occurred'
    } = options;
    
    let loadingId;
    
    if (showGlobal) {
      loadingId = this.showGlobalLoading(message);
    } else {
      this.showProgressBar();
    }
    
    try {
      const result = await asyncFunction();
      
      // Show success toast
      if (successMessage) {
        this.showToast(successMessage, 'success');
      }
      
      return result;
    } catch (error) {
      // Show error toast
      if (errorMessage) {
        this.showToast(errorMessage, 'error');
      }
      
      throw error;
    } finally {
      if (showGlobal && loadingId) {
        this.hideGlobalLoading(loadingId);
      } else {
        this.hideProgressBar();
      }
    }
  }
}

// Initialize loading indicator system
const loadingIndicators = new LoadingIndicatorSystem();

// Export for use in other scripts
window.loadingIndicators = loadingIndicators;

// Convenience functions
window.showLoading = (message) => loadingIndicators.showGlobalLoading(message);
window.hideLoading = () => loadingIndicators.hideGlobalLoading();

// Auto-add loading indicators to common elements
document.addEventListener('DOMContentLoaded', function() {
  // Add loading messages to forms
  const formMessages = {
    'login': 'Signing you in...',
    'register': 'Creating your account...',
    'add-bus': 'Adding bus to fleet...',
    'add-student': 'Adding student...',
    'assign-route': 'Assigning route...',
    'save-log': 'Saving trip log...',
    'import': 'Importing data...'
  };
  
  Object.entries(formMessages).forEach(([keyword, message]) => {
    document.querySelectorAll(`form[action*="${keyword}"]`).forEach(form => {
      form.setAttribute('data-loading-message', message);
    });
  });
  
  // Add loading text to buttons
  document.querySelectorAll('button[type="submit"]').forEach(button => {
    const text = button.textContent.toLowerCase();
    if (text.includes('save')) {
      button.setAttribute('data-loading-text', 'Saving...');
    } else if (text.includes('delete') || text.includes('remove')) {
      button.setAttribute('data-loading-text', 'Removing...');
    } else if (text.includes('add') || text.includes('create')) {
      button.setAttribute('data-loading-text', 'Creating...');
    } else if (text.includes('update')) {
      button.setAttribute('data-loading-text', 'Updating...');
    }
  });
});