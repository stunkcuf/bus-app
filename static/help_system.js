/* Help System JavaScript - Accessible Help for Older Users
   Provides contextual help, tooltips, and guided assistance */

class HelpSystem {
  constructor() {
    this.helpMode = false;
    this.currentHelp = null;
    this.init();
  }

  init() {
    this.createHelpToggle();
    this.setupEventListeners();
    this.initializeHelpSections();
    this.setupKeyboardNavigation();
  }

  // Create the floating help toggle button
  createHelpToggle() {
    const helpToggle = document.createElement('button');
    helpToggle.className = 'help-toggle';
    helpToggle.innerHTML = '<i class="bi bi-question-circle-fill" aria-hidden="true"></i>';
    helpToggle.setAttribute('aria-label', 'Toggle help mode');
    helpToggle.setAttribute('title', 'Click to show/hide help information');
    
    helpToggle.addEventListener('click', () => {
      this.toggleHelpMode();
    });

    document.body.appendChild(helpToggle);
    this.helpToggle = helpToggle;
  }

  // Toggle help mode on/off
  toggleHelpMode() {
    this.helpMode = !this.helpMode;
    document.body.classList.toggle('help-mode', this.helpMode);
    
    if (this.helpMode) {
      this.helpToggle.classList.add('active');
      this.helpToggle.setAttribute('aria-label', 'Hide help information');
      this.helpToggle.innerHTML = '<i class="bi bi-check-circle-fill" aria-hidden="true"></i>';
      this.showWelcomeMessage();
    } else {
      this.helpToggle.classList.remove('active');
      this.helpToggle.setAttribute('aria-label', 'Show help information');
      this.helpToggle.innerHTML = '<i class="bi bi-question-circle-fill" aria-hidden="true"></i>';
      this.hideWelcomeMessage();
    }
  }

  // Show welcome message when help mode is activated
  showWelcomeMessage() {
    if (document.querySelector('.help-welcome')) return;

    const welcomeDiv = document.createElement('div');
    welcomeDiv.className = 'help-welcome help-panel';
    welcomeDiv.innerHTML = `
      <div class="help-panel-header">
        <div class="help-panel-icon">
          <i class="bi bi-lightbulb-fill"></i>
        </div>
        <h3 class="help-panel-title">Help Mode Activated</h3>
      </div>
      <div class="help-panel-content">
        <p><strong>Great!</strong> Help mode is now active. You'll see helpful information and tips throughout the page.</p>
        <ul>
          <li><strong>Tooltips:</strong> All help tooltips are now visible</li>
          <li><strong>Instructions:</strong> Look for detailed step-by-step guides</li>
          <li><strong>Tips:</strong> Green boxes show helpful tips</li>
          <li><strong>Warnings:</strong> Yellow boxes highlight important information</li>
        </ul>
        <p>Click the help button again to hide this information.</p>
      </div>
    `;

    // Insert at the top of main content
    const mainContent = document.querySelector('main') || document.querySelector('.container') || document.body;
    const firstChild = mainContent.firstElementChild;
    if (firstChild) {
      mainContent.insertBefore(welcomeDiv, firstChild);
    } else {
      mainContent.appendChild(welcomeDiv);
    }

    // Auto-hide after 10 seconds
    setTimeout(() => {
      if (welcomeDiv.parentNode) {
        welcomeDiv.style.opacity = '0.7';
      }
    }, 10000);
  }

  // Hide welcome message
  hideWelcomeMessage() {
    const welcomeDiv = document.querySelector('.help-welcome');
    if (welcomeDiv) {
      welcomeDiv.remove();
    }
  }

  // Initialize collapsible help sections
  initializeHelpSections() {
    const helpSections = document.querySelectorAll('.help-section');
    
    helpSections.forEach(section => {
      const header = section.querySelector('.help-section-header');
      const content = section.querySelector('.help-section-content');
      
      if (header && content) {
        header.addEventListener('click', () => {
          const isExpanded = section.classList.contains('expanded');
          
          // Close other sections
          helpSections.forEach(otherSection => {
            if (otherSection !== section) {
              otherSection.classList.remove('expanded');
            }
          });
          
          // Toggle current section
          section.classList.toggle('expanded', !isExpanded);
          
          // Update ARIA attributes
          header.setAttribute('aria-expanded', (!isExpanded).toString());
          content.setAttribute('aria-hidden', isExpanded.toString());
        });

        // Set initial ARIA attributes
        header.setAttribute('aria-expanded', 'false');
        header.setAttribute('role', 'button');
        header.setAttribute('tabindex', '0');
        content.setAttribute('aria-hidden', 'true');
      }
    });
  }

  // Setup keyboard navigation for help system
  setupKeyboardNavigation() {
    document.addEventListener('keydown', (e) => {
      // Press F1 to toggle help mode
      if (e.key === 'F1') {
        e.preventDefault();
        this.toggleHelpMode();
      }
      
      // Press Escape to exit help mode
      if (e.key === 'Escape' && this.helpMode) {
        this.toggleHelpMode();
      }

      // Enter/Space to activate help section headers
      if ((e.key === 'Enter' || e.key === ' ') && e.target.classList.contains('help-section-header')) {
        e.preventDefault();
        e.target.click();
      }
    });
  }

  // Setup event listeners
  setupEventListeners() {
    // Close tooltips when clicking outside
    document.addEventListener('click', (e) => {
      if (!e.target.closest('.help-tooltip')) {
        this.closeAllTooltips();
      }
    });

    // Handle tooltip focus
    document.addEventListener('focusin', (e) => {
      if (e.target.classList.contains('help-tooltip-trigger')) {
        this.showTooltip(e.target);
      }
    });

    document.addEventListener('focusout', (e) => {
      if (e.target.classList.contains('help-tooltip-trigger')) {
        setTimeout(() => {
          if (!e.target.closest('.help-tooltip:hover')) {
            this.hideTooltip(e.target);
          }
        }, 100);
      }
    });
  }

  // Show specific tooltip
  showTooltip(trigger) {
    const tooltip = trigger.parentNode.querySelector('.help-tooltip-content');
    if (tooltip) {
      tooltip.style.opacity = '1';
      tooltip.style.visibility = 'visible';
      tooltip.style.pointerEvents = 'auto';
    }
  }

  // Hide specific tooltip
  hideTooltip(trigger) {
    const tooltip = trigger.parentNode.querySelector('.help-tooltip-content');
    if (tooltip && !document.body.classList.contains('help-mode')) {
      tooltip.style.opacity = '0';
      tooltip.style.visibility = 'hidden';
      tooltip.style.pointerEvents = 'none';
    }
  }

  // Close all tooltips
  closeAllTooltips() {
    if (!this.helpMode) {
      const tooltips = document.querySelectorAll('.help-tooltip-content');
      tooltips.forEach(tooltip => {
        tooltip.style.opacity = '0';
        tooltip.style.visibility = 'hidden';
        tooltip.style.pointerEvents = 'none';
      });
    }
  }

  // Add contextual help to form fields
  static addFieldHelp(fieldId, helpText, options = {}) {
    const field = document.getElementById(fieldId);
    if (!field) return;

    const {
      type = 'tooltip', // 'tooltip' or 'inline'
      icon = 'question-circle',
      position = 'right'
    } = options;

    if (type === 'tooltip') {
      const helpTooltip = document.createElement('div');
      helpTooltip.className = 'help-tooltip';
      
      helpTooltip.innerHTML = `
        <button type="button" class="help-tooltip-trigger" aria-describedby="${fieldId}-help">
          <i class="bi bi-${icon}" aria-hidden="true"></i>
        </button>
        <div class="help-tooltip-content" id="${fieldId}-help" role="tooltip">
          ${helpText}
        </div>
      `;

      // Insert after the field
      field.parentNode.insertBefore(helpTooltip, field.nextSibling);
    } else {
      const helpInline = document.createElement('div');
      helpInline.className = 'form-help';
      helpInline.id = `${fieldId}-help`;
      helpInline.innerHTML = `
        <i class="bi bi-${icon} form-help-icon" aria-hidden="true"></i>
        <span>${helpText}</span>
      `;

      // Insert after the field
      field.parentNode.insertBefore(helpInline, field.nextSibling);
      
      // Update field aria-describedby
      const existingDescribedBy = field.getAttribute('aria-describedby') || '';
      field.setAttribute('aria-describedby', 
        existingDescribedBy ? `${existingDescribedBy} ${fieldId}-help` : `${fieldId}-help`
      );
    }
  }

  // Add help section to page
  static addHelpSection(title, content, options = {}) {
    const {
      icon = 'info-circle',
      expanded = false,
      insertAfter = null
    } = options;

    const helpSection = document.createElement('div');
    helpSection.className = `help-section ${expanded ? 'expanded' : ''}`;
    
    helpSection.innerHTML = `
      <div class="help-section-header" role="button" tabindex="0" aria-expanded="${expanded}">
        <i class="bi bi-${icon} help-section-icon" aria-hidden="true"></i>
        <h3 class="help-section-title">${title}</h3>
        <i class="bi bi-chevron-down help-section-toggle" aria-hidden="true"></i>
      </div>
      <div class="help-section-content" aria-hidden="${!expanded}">
        <div class="help-section-body">
          ${content}
        </div>
      </div>
    `;

    // Insert into page
    const target = insertAfter || document.querySelector('main') || document.querySelector('.container') || document.body;
    if (insertAfter) {
      target.parentNode.insertBefore(helpSection, target.nextSibling);
    } else {
      target.appendChild(helpSection);
    }

    return helpSection;
  }

  // Create step-by-step guide
  static createStepGuide(steps, title = "Step-by-Step Guide") {
    const stepsList = steps.map((step, index) => `
      <li class="help-step">
        <div class="help-step-number">${index + 1}</div>
        <div class="help-step-content">
          <div class="help-step-title">${step.title}</div>
          <div class="help-step-description">${step.description}</div>
        </div>
      </li>
    `).join('');

    return `
      <h4>${title}</h4>
      <ol class="help-steps">
        ${stepsList}
      </ol>
    `;
  }

  // Create warning message
  static createWarning(message) {
    return `
      <div class="help-warning">
        <i class="bi bi-exclamation-triangle-fill help-warning-icon" aria-hidden="true"></i>
        <div>${message}</div>
      </div>
    `;
  }

  // Create tip message
  static createTip(message) {
    return `
      <div class="help-tip">
        <i class="bi bi-lightbulb-fill help-tip-icon" aria-hidden="true"></i>
        <div>${message}</div>
      </div>
    `;
  }
}

// Initialize help system when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
  window.helpSystem = new HelpSystem();
});

// Export for use in other scripts
if (typeof module !== 'undefined' && module.exports) {
  module.exports = HelpSystem;
}