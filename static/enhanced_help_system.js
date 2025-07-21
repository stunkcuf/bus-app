// Enhanced Help System for Fleet Management
// =========================================
// Provides comprehensive help features for older and non-technical users

class EnhancedHelpSystem {
  constructor() {
    this.isHelpMode = false;
    this.currentTutorial = null;
    this.helpContent = new Map();
    this.tooltips = new Map();
    this.init();
  }

  init() {
    // Create help button
    this.createHelpButton();
    
    // Create help panel
    this.createHelpPanel();
    
    // Initialize help content
    this.loadHelpContent();
    
    // Set up event listeners
    this.setupEventListeners();
    
    // Initialize tooltips
    this.initializeTooltips();
  }

  createHelpButton() {
    const button = document.createElement('button');
    button.className = 'help-toggle-btn';
    button.innerHTML = '<i class="bi bi-question-circle-fill" aria-hidden="true"></i>';
    button.setAttribute('aria-label', 'Toggle help mode');
    button.setAttribute('title', 'Click for help');
    document.body.appendChild(button);
    this.helpButton = button;
  }

  createHelpPanel() {
    const panel = document.createElement('div');
    panel.className = 'help-panel';
    panel.innerHTML = `
      <div class="help-panel-header">
        <h2 class="help-panel-title">
          <i class="bi bi-life-preserver" aria-hidden="true"></i>
          Help & Support
        </h2>
        <button class="help-panel-close" aria-label="Close help panel">
          <i class="bi bi-x-lg" aria-hidden="true"></i>
        </button>
      </div>
      <div class="help-panel-content">
        <div class="help-search">
          <i class="bi bi-search help-search-icon" aria-hidden="true"></i>
          <input type="text" 
                 class="help-search-input" 
                 placeholder="Search for help..."
                 aria-label="Search help topics">
          <div class="help-search-results"></div>
        </div>
        
        <div class="help-section">
          <h3 class="help-section-title">
            <i class="bi bi-book" aria-hidden="true"></i>
            Quick Start Guide
          </h3>
          <div id="quickStartContent"></div>
        </div>
        
        <div class="help-section">
          <h3 class="help-section-title">
            <i class="bi bi-play-circle" aria-hidden="true"></i>
            Video Tutorials
          </h3>
          <div id="videoTutorials"></div>
        </div>
        
        <div class="help-section">
          <h3 class="help-section-title">
            <i class="bi bi-question-circle" aria-hidden="true"></i>
            Frequently Asked Questions
          </h3>
          <div id="faqContent"></div>
        </div>
        
        <div class="help-section">
          <h3 class="help-section-title">
            <i class="bi bi-telephone" aria-hidden="true"></i>
            Contact Support
          </h3>
          <div class="help-card">
            <div class="help-card-title">
              <i class="bi bi-envelope" aria-hidden="true"></i>
              Email Support
            </div>
            <div class="help-card-description">
              support@fleetmanagement.com<br>
              Response time: Within 24 hours
            </div>
          </div>
          <div class="help-card">
            <div class="help-card-title">
              <i class="bi bi-telephone" aria-hidden="true"></i>
              Phone Support
            </div>
            <div class="help-card-description">
              1-800-FLEET-HELP<br>
              Monday-Friday, 8 AM - 6 PM EST
            </div>
          </div>
        </div>
      </div>
    `;
    document.body.appendChild(panel);
    this.helpPanel = panel;
  }

  loadHelpContent() {
    // Load context-specific help content based on current page
    const page = this.getCurrentPage();
    
    // Quick Start content
    const quickStart = {
      'manager-dashboard': [
        { 
          title: 'View Fleet Status', 
          description: 'Check the status of all buses and vehicles at a glance',
          icon: 'bi-bus-front'
        },
        { 
          title: 'Assign Routes', 
          description: 'Assign drivers to specific bus routes for the day',
          icon: 'bi-diagram-3'
        },
        { 
          title: 'Manage Students', 
          description: 'Add, edit, or remove students from the system',
          icon: 'bi-people'
        }
      ],
      'driver-dashboard': [
        { 
          title: 'Log Daily Trip', 
          description: 'Record your daily trip information including mileage',
          icon: 'bi-journal-text'
        },
        { 
          title: 'Take Attendance', 
          description: 'Mark which students were present on your route',
          icon: 'bi-check2-square'
        },
        { 
          title: 'Report Issues', 
          description: 'Report any vehicle maintenance issues immediately',
          icon: 'bi-exclamation-triangle'
        }
      ]
    };

    // FAQ content
    const faqs = {
      'general': [
        {
          question: 'How do I reset my password?',
          answer: 'Click on your username in the top right, then select "Change Password" from the menu. You\'ll need to enter your current password and then your new password twice.'
        },
        {
          question: 'What browsers are supported?',
          answer: 'The system works best with Chrome, Firefox, Safari, and Edge. Make sure your browser is up to date for the best experience.'
        },
        {
          question: 'How do I print reports?',
          answer: 'Open the report you want to print, then click the "Print" button at the top of the page. You can also use Ctrl+P (Windows) or Cmd+P (Mac).'
        }
      ]
    };

    this.quickStartContent = quickStart[page] || quickStart['general'] || [];
    this.faqContent = faqs[page] || faqs['general'] || [];
    
    this.renderQuickStart();
    this.renderFAQ();
  }

  renderQuickStart() {
    const container = document.getElementById('quickStartContent');
    if (!container) return;
    
    container.innerHTML = this.quickStartContent.map(item => `
      <div class="help-card" onclick="helpSystem.showTutorial('${item.title}')">
        <div class="help-card-title">
          <i class="${item.icon}" aria-hidden="true"></i>
          ${item.title}
        </div>
        <div class="help-card-description">${item.description}</div>
      </div>
    `).join('');
  }

  renderFAQ() {
    const container = document.getElementById('faqContent');
    if (!container) return;
    
    container.innerHTML = this.faqContent.map((item, index) => `
      <div class="help-card" onclick="helpSystem.toggleFAQ(${index})">
        <div class="help-card-title">
          <i class="bi bi-question-circle" aria-hidden="true"></i>
          ${item.question}
        </div>
        <div class="help-card-description" id="faq-${index}" style="display: none;">
          ${item.answer}
        </div>
      </div>
    `).join('');
  }

  setupEventListeners() {
    // Help button click
    this.helpButton.addEventListener('click', () => this.toggleHelpMode());
    
    // Help panel close
    const closeBtn = this.helpPanel.querySelector('.help-panel-close');
    closeBtn.addEventListener('click', () => this.closeHelpPanel());
    
    // Help search
    const searchInput = this.helpPanel.querySelector('.help-search-input');
    searchInput.addEventListener('input', (e) => this.handleSearch(e.target.value));
    
    // Keyboard shortcuts
    document.addEventListener('keydown', (e) => {
      // F1 for help
      if (e.key === 'F1') {
        e.preventDefault();
        this.toggleHelpMode();
      }
      // Escape to close help
      if (e.key === 'Escape' && this.isHelpMode) {
        this.toggleHelpMode();
      }
    });
    
    // Click outside tooltip to close
    document.addEventListener('click', (e) => {
      if (!e.target.closest('.help-tooltip') && !e.target.closest('[data-help]')) {
        this.hideAllTooltips();
      }
    });
  }

  initializeTooltips() {
    // Find all elements with data-help attribute
    const helpElements = document.querySelectorAll('[data-help]');
    
    helpElements.forEach(element => {
      const helpKey = element.getAttribute('data-help');
      const helpTitle = element.getAttribute('data-help-title') || 'Help';
      const helpContent = element.getAttribute('data-help-content') || this.getHelpContent(helpKey);
      
      // Add help indicator
      if (!element.querySelector('.help-indicator')) {
        const indicator = document.createElement('span');
        indicator.className = 'help-indicator';
        indicator.innerHTML = '<i class="bi bi-info-circle" aria-hidden="true"></i>';
        indicator.style.cssText = `
          position: absolute;
          top: -8px;
          right: -8px;
          background: var(--color-info);
          color: white;
          width: 20px;
          height: 20px;
          border-radius: 50%;
          display: flex;
          align-items: center;
          justify-content: center;
          font-size: 12px;
          cursor: help;
          z-index: 1;
          opacity: 0;
          transition: opacity 0.3s;
        `;
        element.style.position = 'relative';
        element.appendChild(indicator);
      }
      
      // Show indicator in help mode
      element.addEventListener('mouseenter', () => {
        if (this.isHelpMode) {
          const indicator = element.querySelector('.help-indicator');
          if (indicator) indicator.style.opacity = '1';
        }
      });
      
      element.addEventListener('mouseleave', () => {
        const indicator = element.querySelector('.help-indicator');
        if (indicator) indicator.style.opacity = '0';
      });
      
      // Click to show tooltip
      element.addEventListener('click', (e) => {
        if (this.isHelpMode) {
          e.preventDefault();
          e.stopPropagation();
          this.showTooltip(element, helpTitle, helpContent);
        }
      });
    });
  }

  showTooltip(element, title, content) {
    // Hide other tooltips
    this.hideAllTooltips();
    
    // Create tooltip
    const tooltip = document.createElement('div');
    tooltip.className = 'help-tooltip';
    tooltip.innerHTML = `
      <div class="help-tooltip-header">
        <i class="bi bi-lightbulb" aria-hidden="true"></i>
        ${title}
      </div>
      <div class="help-tooltip-content">${content}</div>
      <div class="help-tooltip-actions">
        <button onclick="helpSystem.hideTooltip(this.parentElement.parentElement)">Got it</button>
        <button onclick="helpSystem.showMoreHelp('${title}')">Learn more</button>
      </div>
    `;
    
    document.body.appendChild(tooltip);
    
    // Position tooltip
    const rect = element.getBoundingClientRect();
    const tooltipRect = tooltip.getBoundingClientRect();
    
    // Determine best position
    let top = rect.bottom + 10;
    let left = rect.left;
    let position = 'bottom';
    
    // Check if tooltip goes off screen
    if (top + tooltipRect.height > window.innerHeight) {
      top = rect.top - tooltipRect.height - 10;
      position = 'top';
    }
    
    if (left + tooltipRect.width > window.innerWidth) {
      left = window.innerWidth - tooltipRect.width - 20;
    }
    
    tooltip.style.top = `${top}px`;
    tooltip.style.left = `${left}px`;
    tooltip.classList.add(position);
    
    // Show tooltip with animation
    setTimeout(() => tooltip.classList.add('show'), 10);
    
    // Highlight element
    element.classList.add('help-highlight');
  }

  hideTooltip(tooltip) {
    tooltip.classList.remove('show');
    setTimeout(() => tooltip.remove(), 300);
    
    // Remove highlights
    document.querySelectorAll('.help-highlight').forEach(el => {
      el.classList.remove('help-highlight');
    });
  }

  hideAllTooltips() {
    document.querySelectorAll('.help-tooltip').forEach(tooltip => {
      this.hideTooltip(tooltip);
    });
  }

  toggleHelpMode() {
    this.isHelpMode = !this.isHelpMode;
    
    if (this.isHelpMode) {
      document.body.classList.add('help-mode-active');
      this.helpButton.classList.add('active');
      this.showHelpPanel();
      this.showHelpModeMessage();
    } else {
      document.body.classList.remove('help-mode-active');
      this.helpButton.classList.remove('active');
      this.closeHelpPanel();
      this.hideAllTooltips();
    }
  }

  showHelpModeMessage() {
    const message = document.createElement('div');
    message.className = 'help-mode-message';
    message.style.cssText = `
      position: fixed;
      top: 20px;
      left: 50%;
      transform: translateX(-50%);
      background: var(--color-info);
      color: white;
      padding: 1rem 2rem;
      border-radius: var(--border-radius-lg);
      font-size: 1.1rem;
      z-index: 10002;
      animation: slideDown 0.3s ease;
    `;
    message.innerHTML = `
      <i class="bi bi-info-circle" aria-hidden="true"></i>
      Help Mode Active - Click on any element to see help
    `;
    document.body.appendChild(message);
    
    setTimeout(() => {
      message.style.animation = 'slideUp 0.3s ease';
      setTimeout(() => message.remove(), 300);
    }, 3000);
  }

  showHelpPanel() {
    this.helpPanel.classList.add('show');
    
    // Create overlay
    const overlay = document.createElement('div');
    overlay.className = 'help-mode-overlay';
    overlay.addEventListener('click', () => this.closeHelpPanel());
    document.body.appendChild(overlay);
  }

  closeHelpPanel() {
    this.helpPanel.classList.remove('show');
    const overlay = document.querySelector('.help-mode-overlay');
    if (overlay) overlay.remove();
  }

  handleSearch(query) {
    const resultsContainer = this.helpPanel.querySelector('.help-search-results');
    
    if (!query.trim()) {
      resultsContainer.classList.remove('show');
      return;
    }
    
    // Search through help content
    const results = this.searchHelpContent(query.toLowerCase());
    
    if (results.length > 0) {
      resultsContainer.innerHTML = results.map(result => `
        <div class="help-search-result" onclick="helpSystem.showHelpTopic('${result.id}')">
          <strong>${result.title}</strong><br>
          <small>${result.preview}</small>
        </div>
      `).join('');
      resultsContainer.classList.add('show');
    } else {
      resultsContainer.innerHTML = `
        <div class="help-search-result">
          <em>No results found for "${query}"</em>
        </div>
      `;
      resultsContainer.classList.add('show');
    }
  }

  searchHelpContent(query) {
    // Simulated search - in production, this would search a help database
    const allContent = [
      { id: 'add-bus', title: 'How to Add a New Bus', preview: 'Learn how to add a new bus to your fleet...' },
      { id: 'assign-route', title: 'Assigning Routes to Drivers', preview: 'Step-by-step guide to assign routes...' },
      { id: 'student-roster', title: 'Managing Student Roster', preview: 'Add, edit, and remove students...' },
      { id: 'reports', title: 'Generating Reports', preview: 'Create and export various reports...' },
      { id: 'maintenance', title: 'Vehicle Maintenance', preview: 'Track and schedule maintenance...' }
    ];
    
    return allContent.filter(item => 
      item.title.toLowerCase().includes(query) || 
      item.preview.toLowerCase().includes(query)
    );
  }

  showTutorial(topic) {
    // Start interactive tutorial for specific topic
    const tutorials = {
      'View Fleet Status': [
        {
          element: '.quick-action[href="/fleet"]',
          title: 'Access Fleet Management',
          content: 'Click here to view all your buses and vehicles.',
          position: 'bottom'
        },
        {
          element: '.bus-status',
          title: 'Understanding Status Colors',
          content: 'Green means active, yellow means maintenance due, red means out of service.',
          position: 'right'
        }
      ]
    };
    
    const steps = tutorials[topic];
    if (steps) {
      this.startTutorial(steps);
    }
  }

  startTutorial(steps) {
    this.currentTutorial = {
      steps: steps,
      currentStep: 0
    };
    
    this.closeHelpPanel();
    this.showTutorialStep(0);
  }

  showTutorialStep(stepIndex) {
    const step = this.currentTutorial.steps[stepIndex];
    if (!step) return;
    
    const element = document.querySelector(step.element);
    if (!element) {
      console.warn(`Tutorial element not found: ${step.element}`);
      return;
    }
    
    // Create spotlight
    const spotlight = document.createElement('div');
    spotlight.className = 'tutorial-spotlight';
    const rect = element.getBoundingClientRect();
    spotlight.style.cssText = `
      top: ${rect.top - 10}px;
      left: ${rect.left - 10}px;
      width: ${rect.width + 20}px;
      height: ${rect.height + 20}px;
    `;
    document.body.appendChild(spotlight);
    
    // Create tutorial bubble
    const bubble = document.createElement('div');
    bubble.className = 'tutorial-bubble';
    bubble.innerHTML = `
      <div class="tutorial-bubble-header">
        <div>
          <div class="tutorial-bubble-title">${step.title}</div>
          <div class="tutorial-bubble-step">Step ${stepIndex + 1} of ${this.currentTutorial.steps.length}</div>
        </div>
        <button onclick="helpSystem.endTutorial()" aria-label="End tutorial">
          <i class="bi bi-x" aria-hidden="true"></i>
        </button>
      </div>
      <div class="tutorial-bubble-content">${step.content}</div>
      <div class="tutorial-bubble-actions">
        <div class="tutorial-progress">
          ${this.currentTutorial.steps.map((_, i) => 
            `<div class="tutorial-progress-dot ${i === stepIndex ? 'active' : ''}"></div>`
          ).join('')}
        </div>
        <div>
          ${stepIndex > 0 ? 
            `<button class="btn btn-secondary" onclick="helpSystem.previousTutorialStep()">Previous</button>` : 
            ''
          }
          ${stepIndex < this.currentTutorial.steps.length - 1 ? 
            `<button class="btn btn-primary" onclick="helpSystem.nextTutorialStep()">Next</button>` :
            `<button class="btn btn-success" onclick="helpSystem.endTutorial()">Finish</button>`
          }
        </div>
      </div>
    `;
    
    document.body.appendChild(bubble);
    
    // Position bubble
    this.positionTutorialBubble(bubble, element, step.position);
    
    // Show with animation
    setTimeout(() => {
      spotlight.style.transition = 'all 0.5s ease';
      bubble.classList.add('show');
    }, 10);
    
    this.currentTutorialElements = { spotlight, bubble };
  }

  positionTutorialBubble(bubble, targetElement, preferredPosition) {
    const targetRect = targetElement.getBoundingClientRect();
    const bubbleRect = bubble.getBoundingClientRect();
    
    let top, left;
    
    switch (preferredPosition) {
      case 'bottom':
        top = targetRect.bottom + 20;
        left = targetRect.left + (targetRect.width - bubbleRect.width) / 2;
        break;
      case 'top':
        top = targetRect.top - bubbleRect.height - 20;
        left = targetRect.left + (targetRect.width - bubbleRect.width) / 2;
        break;
      case 'right':
        top = targetRect.top + (targetRect.height - bubbleRect.height) / 2;
        left = targetRect.right + 20;
        break;
      case 'left':
        top = targetRect.top + (targetRect.height - bubbleRect.height) / 2;
        left = targetRect.left - bubbleRect.width - 20;
        break;
    }
    
    // Keep bubble on screen
    top = Math.max(20, Math.min(top, window.innerHeight - bubbleRect.height - 20));
    left = Math.max(20, Math.min(left, window.innerWidth - bubbleRect.width - 20));
    
    bubble.style.top = `${top}px`;
    bubble.style.left = `${left}px`;
  }

  nextTutorialStep() {
    this.clearTutorialElements();
    this.currentTutorial.currentStep++;
    this.showTutorialStep(this.currentTutorial.currentStep);
  }

  previousTutorialStep() {
    this.clearTutorialElements();
    this.currentTutorial.currentStep--;
    this.showTutorialStep(this.currentTutorial.currentStep);
  }

  endTutorial() {
    this.clearTutorialElements();
    this.currentTutorial = null;
  }

  clearTutorialElements() {
    if (this.currentTutorialElements) {
      this.currentTutorialElements.spotlight.remove();
      this.currentTutorialElements.bubble.remove();
      this.currentTutorialElements = null;
    }
  }

  toggleFAQ(index) {
    const answer = document.getElementById(`faq-${index}`);
    if (answer) {
      answer.style.display = answer.style.display === 'none' ? 'block' : 'none';
    }
  }

  showMoreHelp(topic) {
    // Open help panel with specific topic
    this.showHelpPanel();
    // In production, this would navigate to the specific help topic
  }

  getCurrentPage() {
    // Determine current page from URL or other means
    const path = window.location.pathname;
    if (path.includes('manager-dashboard')) return 'manager-dashboard';
    if (path.includes('driver-dashboard')) return 'driver-dashboard';
    if (path.includes('fleet')) return 'fleet';
    if (path.includes('students')) return 'students';
    return 'general';
  }

  getHelpContent(key) {
    // Default help content for common elements
    const content = {
      'quick-actions': 'These buttons provide quick access to frequently used features. Click any button to navigate to that section.',
      'fleet-status': 'View the current status of all buses in your fleet. Green indicates active, yellow means maintenance is due, and red means out of service.',
      'student-roster': 'Manage all students assigned to routes. You can add new students, edit existing information, or remove students who are no longer active.',
      'route-assignment': 'Assign drivers to specific bus routes. Ensure each route has a driver and bus assigned before the start of the day.',
      'reports': 'Generate various reports including mileage, maintenance, and student attendance. Reports can be viewed on screen or exported.',
      'user-profile': 'View and edit your profile information, change your password, and manage notification preferences.'
    };
    
    return content[key] || 'Click to learn more about this feature.';
  }
}

// Initialize help system when DOM is ready
document.addEventListener('DOMContentLoaded', function() {
  window.helpSystem = new EnhancedHelpSystem();
  
  // Add custom styles for animations
  const style = document.createElement('style');
  style.textContent = `
    @keyframes slideDown {
      from { transform: translate(-50%, -100%); opacity: 0; }
      to { transform: translate(-50%, 0); opacity: 1; }
    }
    
    @keyframes slideUp {
      from { transform: translate(-50%, 0); opacity: 1; }
      to { transform: translate(-50%, -100%); opacity: 0; }
    }
  `;
  document.head.appendChild(style);
});