// Auto-Save System for Long Forms
// ================================
// Automatically saves form progress to prevent data loss

class AutoSaveSystem {
  constructor() {
    this.forms = new Map();
    this.saveInterval = 30000; // 30 seconds
    this.storagePrefix = 'autosave_';
    this.init();
  }

  init() {
    // Find all forms with auto-save enabled
    this.findAutoSaveForms();
    
    // Set up event listeners
    this.setupEventListeners();
    
    // Restore any saved data
    this.restoreSavedData();
    
    // Add visual indicators
    this.addStyles();
  }

  addStyles() {
    const style = document.createElement('style');
    style.textContent = `
      .autosave-indicator {
        position: fixed;
        bottom: 20px;
        left: 20px;
        background: var(--color-success);
        color: var(--color-text-inverse);
        padding: var(--space-sm) var(--space-md);
        border-radius: var(--border-radius-lg);
        display: flex;
        align-items: center;
        gap: var(--space-sm);
        opacity: 0;
        transform: translateY(20px);
        transition: all 0.3s ease;
        z-index: 1000;
        font-size: var(--font-size-small);
      }
      
      .autosave-indicator.show {
        opacity: 1;
        transform: translateY(0);
      }
      
      .autosave-indicator i {
        animation: spin 1s linear infinite;
      }
      
      @keyframes spin {
        from { transform: rotate(0deg); }
        to { transform: rotate(360deg); }
      }
      
      .form-restore-banner {
        background: var(--color-info-light);
        border: 1px solid var(--color-info);
        border-radius: var(--border-radius-base);
        padding: var(--space-md);
        margin-bottom: var(--space-lg);
        display: flex;
        align-items: center;
        justify-content: space-between;
        gap: var(--space-md);
      }
      
      .form-restore-banner-text {
        display: flex;
        align-items: center;
        gap: var(--space-md);
      }
      
      .form-restore-banner i {
        font-size: 1.5rem;
        color: var(--color-info-dark);
      }
      
      .form-restore-actions {
        display: flex;
        gap: var(--space-sm);
      }
      
      .autosave-status {
        font-size: var(--font-size-small);
        color: var(--color-text-secondary);
        margin-top: var(--space-xs);
        display: flex;
        align-items: center;
        gap: var(--space-xs);
      }
      
      .autosave-status.saved {
        color: var(--color-success);
      }
    `;
    document.head.appendChild(style);
    
    // Create indicator element
    const indicator = document.createElement('div');
    indicator.className = 'autosave-indicator';
    indicator.id = 'autosave-indicator';
    indicator.innerHTML = `
      <i class="bi bi-arrow-repeat" aria-hidden="true"></i>
      <span>Saving...</span>
    `;
    document.body.appendChild(indicator);
  }

  findAutoSaveForms() {
    // Look for forms with data-autosave attribute or long forms
    const forms = document.querySelectorAll('form[data-autosave], form.autosave-enabled');
    
    // Also auto-enable for forms with many fields
    document.querySelectorAll('form').forEach(form => {
      const inputs = form.querySelectorAll('input, textarea, select');
      if (inputs.length > 5 && !form.hasAttribute('data-no-autosave')) {
        forms.forEach.call([form], f => this.enableAutoSave(f));
      }
    });
    
    forms.forEach(form => this.enableAutoSave(form));
  }

  enableAutoSave(form) {
    const formId = form.id || this.generateFormId(form);
    form.id = formId;
    
    const formData = {
      form: form,
      fields: this.getFormFields(form),
      lastSave: null,
      timer: null,
      hasChanges: false
    };
    
    this.forms.set(formId, formData);
    
    // Add status indicator to form
    this.addStatusIndicator(form);
    
    // Start auto-save timer
    this.startAutoSaveTimer(formId);
  }

  generateFormId(form) {
    const action = form.action || 'form';
    const timestamp = Date.now();
    return `autosave_${action.replace(/[^a-zA-Z0-9]/g, '_')}_${timestamp}`;
  }

  getFormFields(form) {
    return form.querySelectorAll('input:not([type="submit"]):not([type="button"]):not([type="hidden"]):not([type="csrf_token"]), textarea, select');
  }

  setupEventListeners() {
    // Listen for form changes
    document.addEventListener('input', (e) => {
      const form = e.target.form;
      if (form && this.forms.has(form.id)) {
        this.markAsChanged(form.id);
      }
    });
    
    document.addEventListener('change', (e) => {
      const form = e.target.form;
      if (form && this.forms.has(form.id)) {
        this.markAsChanged(form.id);
      }
    });
    
    // Save before unload
    window.addEventListener('beforeunload', (e) => {
      this.saveAllForms();
    });
    
    // Clear saved data on successful form submission
    document.addEventListener('submit', (e) => {
      const form = e.target;
      if (this.forms.has(form.id)) {
        // Add flag to clear on successful submission
        form.addEventListener('load', () => {
          this.clearSavedData(form.id);
        });
      }
    });
  }

  markAsChanged(formId) {
    const formData = this.forms.get(formId);
    if (formData) {
      formData.hasChanges = true;
    }
  }

  startAutoSaveTimer(formId) {
    const formData = this.forms.get(formId);
    if (!formData) return;
    
    // Clear existing timer
    if (formData.timer) {
      clearInterval(formData.timer);
    }
    
    // Set new timer
    formData.timer = setInterval(() => {
      if (formData.hasChanges) {
        this.saveForm(formId);
      }
    }, this.saveInterval);
  }

  saveForm(formId) {
    const formData = this.forms.get(formId);
    if (!formData || !formData.hasChanges) return;
    
    const data = {};
    
    // Collect form data
    formData.fields.forEach(field => {
      const name = field.name;
      if (!name) return;
      
      if (field.type === 'checkbox') {
        data[name] = field.checked;
      } else if (field.type === 'radio') {
        if (field.checked) {
          data[name] = field.value;
        }
      } else {
        data[name] = field.value;
      }
    });
    
    // Save to localStorage
    const saveData = {
      data: data,
      timestamp: Date.now(),
      url: window.location.href
    };
    
    try {
      localStorage.setItem(this.storagePrefix + formId, JSON.stringify(saveData));
      
      // Update UI
      this.showSaveIndicator();
      this.updateStatusIndicator(formId, true);
      
      // Reset change flag
      formData.hasChanges = false;
      formData.lastSave = Date.now();
    } catch (e) {
      console.error('Auto-save failed:', e);
    }
  }

  saveAllForms() {
    this.forms.forEach((formData, formId) => {
      if (formData.hasChanges) {
        this.saveForm(formId);
      }
    });
  }

  restoreSavedData() {
    this.forms.forEach((formData, formId) => {
      const savedDataKey = this.storagePrefix + formId;
      const savedDataStr = localStorage.getItem(savedDataKey);
      
      if (savedDataStr) {
        try {
          const savedData = JSON.parse(savedDataStr);
          
          // Check if saved data is recent (within 24 hours)
          const age = Date.now() - savedData.timestamp;
          if (age < 24 * 60 * 60 * 1000) {
            this.showRestoreBanner(formId, savedData);
          } else {
            // Clear old data
            localStorage.removeItem(savedDataKey);
          }
        } catch (e) {
          console.error('Failed to parse saved data:', e);
          localStorage.removeItem(savedDataKey);
        }
      }
    });
  }

  showRestoreBanner(formId, savedData) {
    const formData = this.forms.get(formId);
    if (!formData) return;
    
    const banner = document.createElement('div');
    banner.className = 'form-restore-banner';
    banner.innerHTML = `
      <div class="form-restore-banner-text">
        <i class="bi bi-info-circle-fill" aria-hidden="true"></i>
        <div>
          <strong>Unsaved data found</strong><br>
          <small>Saved ${this.formatTimeAgo(savedData.timestamp)}</small>
        </div>
      </div>
      <div class="form-restore-actions">
        <button type="button" class="btn btn-sm btn-info" onclick="autoSave.restoreData('${formId}')">
          Restore
        </button>
        <button type="button" class="btn btn-sm btn-secondary" onclick="autoSave.discardData('${formId}')">
          Discard
        </button>
      </div>
    `;
    
    formData.form.insertBefore(banner, formData.form.firstChild);
  }

  restoreData(formId) {
    const savedDataStr = localStorage.getItem(this.storagePrefix + formId);
    if (!savedDataStr) return;
    
    try {
      const savedData = JSON.parse(savedDataStr);
      const formData = this.forms.get(formId);
      
      if (!formData) return;
      
      // Restore field values
      Object.entries(savedData.data).forEach(([name, value]) => {
        const field = formData.form.querySelector(`[name="${name}"]`);
        if (field) {
          if (field.type === 'checkbox') {
            field.checked = value;
          } else if (field.type === 'radio') {
            const radio = formData.form.querySelector(`[name="${name}"][value="${value}"]`);
            if (radio) radio.checked = true;
          } else {
            field.value = value;
          }
          
          // Trigger change event
          field.dispatchEvent(new Event('change', { bubbles: true }));
        }
      });
      
      // Remove banner
      const banner = formData.form.querySelector('.form-restore-banner');
      if (banner) banner.remove();
      
      // Show success message
      this.showMessage('Data restored successfully', 'success');
    } catch (e) {
      console.error('Failed to restore data:', e);
      this.showMessage('Failed to restore data', 'error');
    }
  }

  discardData(formId) {
    localStorage.removeItem(this.storagePrefix + formId);
    
    const formData = this.forms.get(formId);
    if (formData) {
      const banner = formData.form.querySelector('.form-restore-banner');
      if (banner) banner.remove();
    }
  }

  clearSavedData(formId) {
    localStorage.removeItem(this.storagePrefix + formId);
    this.updateStatusIndicator(formId, false);
  }

  showSaveIndicator() {
    const indicator = document.getElementById('autosave-indicator');
    indicator.classList.add('show');
    
    setTimeout(() => {
      indicator.classList.remove('show');
    }, 2000);
  }

  addStatusIndicator(form) {
    // Find or create a status container
    let statusContainer = form.querySelector('.form-actions');
    if (!statusContainer) {
      statusContainer = form.querySelector('button[type="submit"]')?.parentElement;
    }
    
    if (statusContainer) {
      const status = document.createElement('div');
      status.className = 'autosave-status';
      status.innerHTML = `
        <i class="bi bi-cloud" aria-hidden="true"></i>
        <span>Auto-save enabled</span>
      `;
      statusContainer.appendChild(status);
    }
  }

  updateStatusIndicator(formId, saved) {
    const formData = this.forms.get(formId);
    if (!formData) return;
    
    const status = formData.form.querySelector('.autosave-status');
    if (status) {
      if (saved) {
        status.classList.add('saved');
        status.innerHTML = `
          <i class="bi bi-cloud-check" aria-hidden="true"></i>
          <span>Saved ${this.formatTimeAgo(Date.now())}</span>
        `;
      } else {
        status.classList.remove('saved');
        status.innerHTML = `
          <i class="bi bi-cloud" aria-hidden="true"></i>
          <span>Auto-save enabled</span>
        `;
      }
    }
  }

  formatTimeAgo(timestamp) {
    const seconds = Math.floor((Date.now() - timestamp) / 1000);
    
    if (seconds < 60) return 'just now';
    if (seconds < 3600) return `${Math.floor(seconds / 60)} minutes ago`;
    if (seconds < 86400) return `${Math.floor(seconds / 3600)} hours ago`;
    return `${Math.floor(seconds / 86400)} days ago`;
  }

  showMessage(message, type = 'info') {
    // You can integrate this with your existing notification system
    console.log(`[AutoSave] ${type}: ${message}`);
  }
}

// Initialize auto-save system
const autoSave = new AutoSaveSystem();

// Expose for manual control
window.autoSave = autoSave;