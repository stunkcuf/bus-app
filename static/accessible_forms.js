/* Accessible Forms JavaScript - Enhanced Validation and UX
   Optimized for older and non-technical users */

class AccessibleForm {
  constructor(formElement, options = {}) {
    this.form = formElement;
    this.options = {
      realTimeValidation: true,
      showProgressOnMultiStep: true,
      enableAutoComplete: true,
      validateOnBlur: true,
      enableProgressiveSave: false,
      customValidators: {},
      ...options
    };
    
    this.errors = {};
    this.touchedFields = new Set();
    this.isValid = true;
    
    this.init();
  }

  init() {
    this.setupForm();
    this.setupValidation();
    this.setupProgressiveDisclosure();
    this.setupAutoComplete();
    this.setupFileUploads();
    this.setupAccessibilityFeatures();
    
    if (this.options.showProgressOnMultiStep) {
      this.setupProgressIndicator();
    }
  }

  setupForm() {
    // Add accessible form class
    this.form.classList.add('accessible-form');
    
    // Add form attributes for accessibility
    this.form.setAttribute('novalidate', 'true');
    
    // Prevent double submission
    this.form.addEventListener('submit', (e) => {
      if (this.form.dataset.submitting === 'true') {
        e.preventDefault();
        return false;
      }
      
      if (!this.validateForm()) {
        e.preventDefault();
        this.showErrorSummary();
        return false;
      }
      
      this.form.dataset.submitting = 'true';
      this.setLoadingState(true);
    });
  }

  setupValidation() {
    const fields = this.form.querySelectorAll('input, textarea, select');
    
    fields.forEach(field => {
      // Add accessibility attributes
      if (field.hasAttribute('required')) {
        field.setAttribute('aria-required', 'true');
      }
      
      // Real-time validation
      if (this.options.realTimeValidation) {
        field.addEventListener('input', (e) => {
          if (this.touchedFields.has(field.name)) {
            this.validateField(field);
          }
        });
      }
      
      // Validate on blur
      if (this.options.validateOnBlur) {
        field.addEventListener('blur', (e) => {
          this.touchedFields.add(field.name);
          this.validateField(field);
        });
      }
      
      // Focus handling
      field.addEventListener('focus', (e) => {
        this.clearFieldError(field);
      });
    });
  }

  validateField(field) {
    const value = field.value.trim();
    const fieldName = field.name;
    const fieldLabel = this.getFieldLabel(field);
    
    // Clear previous errors
    delete this.errors[fieldName];
    
    // Required field validation
    if (field.hasAttribute('required') && !value) {
      this.errors[fieldName] = `${fieldLabel} is required`;
      this.showFieldError(field, this.errors[fieldName]);
      return false;
    }
    
    // Type-specific validation
    if (value) {
      switch (field.type) {
        case 'email':
          if (!this.isValidEmail(value)) {
            this.errors[fieldName] = `Please enter a valid email address`;
            this.showFieldError(field, this.errors[fieldName]);
            return false;
          }
          break;
          
        case 'tel':
          if (!this.isValidPhone(value)) {
            this.errors[fieldName] = `Please enter a valid phone number`;
            this.showFieldError(field, this.errors[fieldName]);
            return false;
          }
          break;
          
        case 'number':
          const min = field.getAttribute('min');
          const max = field.getAttribute('max');
          const numValue = parseFloat(value);
          
          if (isNaN(numValue)) {
            this.errors[fieldName] = `Please enter a valid number`;
            this.showFieldError(field, this.errors[fieldName]);
            return false;
          }
          
          if (min !== null && numValue < parseFloat(min)) {
            this.errors[fieldName] = `Value must be at least ${min}`;
            this.showFieldError(field, this.errors[fieldName]);
            return false;
          }
          
          if (max !== null && numValue > parseFloat(max)) {
            this.errors[fieldName] = `Value must not exceed ${max}`;
            this.showFieldError(field, this.errors[fieldName]);
            return false;
          }
          break;
          
        case 'date':
          if (!this.isValidDate(value)) {
            this.errors[fieldName] = `Please enter a valid date`;
            this.showFieldError(field, this.errors[fieldName]);
            return false;
          }
          break;
      }
      
      // Pattern validation
      const pattern = field.getAttribute('pattern');
      if (pattern && !new RegExp(pattern).test(value)) {
        const patternTitle = field.getAttribute('title') || 'Please enter a valid format';
        this.errors[fieldName] = patternTitle;
        this.showFieldError(field, this.errors[fieldName]);
        return false;
      }
      
      // Custom validation
      if (this.options.customValidators[fieldName]) {
        const customError = this.options.customValidators[fieldName](value, field);
        if (customError) {
          this.errors[fieldName] = customError;
          this.showFieldError(field, this.errors[fieldName]);
          return false;
        }
      }
    }
    
    // Show success state
    this.showFieldSuccess(field);
    return true;
  }

  validateForm() {
    this.errors = {};
    this.isValid = true;
    
    const fields = this.form.querySelectorAll('input, textarea, select');
    
    fields.forEach(field => {
      if (!this.validateField(field)) {
        this.isValid = false;
      }
    });
    
    return this.isValid;
  }

  showFieldError(field, message) {
    // Remove success classes
    field.classList.remove('is-valid');
    field.classList.add('is-invalid');
    
    // Update ARIA attributes
    field.setAttribute('aria-invalid', 'true');
    
    // Get or create error element
    let errorElement = this.getFieldErrorElement(field);
    if (!errorElement) {
      errorElement = this.createFieldErrorElement(field);
    }
    
    // Update error message
    errorElement.innerHTML = `<i class="bi bi-exclamation-circle" aria-hidden="true"></i> ${message}`;
    errorElement.style.display = 'flex';
    
    // Update aria-describedby
    const describedBy = field.getAttribute('aria-describedby') || '';
    if (!describedBy.includes(errorElement.id)) {
      field.setAttribute('aria-describedby', 
        describedBy ? `${describedBy} ${errorElement.id}` : errorElement.id
      );
    }
  }

  showFieldSuccess(field) {
    // Remove error classes
    field.classList.remove('is-invalid');
    field.classList.add('is-valid');
    
    // Update ARIA attributes
    field.setAttribute('aria-invalid', 'false');
    
    // Hide error message
    const errorElement = this.getFieldErrorElement(field);
    if (errorElement) {
      errorElement.style.display = 'none';
    }
    
    // Show success message for important fields
    if (field.hasAttribute('required') || field.type === 'email') {
      let successElement = this.getFieldSuccessElement(field);
      if (!successElement) {
        successElement = this.createFieldSuccessElement(field);
      }
      
      successElement.innerHTML = `<i class="bi bi-check-circle" aria-hidden="true"></i> Looks good!`;
      successElement.style.display = 'flex';
      
      // Hide success message after 3 seconds
      setTimeout(() => {
        if (successElement) {
          successElement.style.display = 'none';
        }
      }, 3000);
    }
  }

  clearFieldError(field) {
    const errorElement = this.getFieldErrorElement(field);
    if (errorElement) {
      errorElement.style.display = 'none';
    }
    
    const successElement = this.getFieldSuccessElement(field);
    if (successElement) {
      successElement.style.display = 'none';
    }
    
    field.classList.remove('is-invalid', 'is-valid');
    field.setAttribute('aria-invalid', 'false');
  }

  getFieldErrorElement(field) {
    return document.getElementById(`${field.name}-error`);
  }

  getFieldSuccessElement(field) {
    return document.getElementById(`${field.name}-success`);
  }

  createFieldErrorElement(field) {
    const errorElement = document.createElement('div');
    errorElement.id = `${field.name}-error`;
    errorElement.className = 'invalid-feedback';
    errorElement.style.display = 'none';
    
    const fieldGroup = field.closest('.form-group');
    if (fieldGroup) {
      fieldGroup.appendChild(errorElement);
    } else {
      field.parentNode.appendChild(errorElement);
    }
    
    return errorElement;
  }

  createFieldSuccessElement(field) {
    const successElement = document.createElement('div');
    successElement.id = `${field.name}-success`;
    successElement.className = 'valid-feedback';
    successElement.style.display = 'none';
    
    const fieldGroup = field.closest('.form-group');
    if (fieldGroup) {
      fieldGroup.appendChild(successElement);
    } else {
      field.parentNode.appendChild(successElement);
    }
    
    return successElement;
  }

  showErrorSummary() {
    // Remove existing error summary
    const existingSummary = this.form.querySelector('.form-errors');
    if (existingSummary) {
      existingSummary.remove();
    }
    
    if (Object.keys(this.errors).length === 0) return;
    
    // Create error summary
    const errorSummary = document.createElement('div');
    errorSummary.className = 'form-errors';
    errorSummary.setAttribute('role', 'alert');
    errorSummary.setAttribute('aria-live', 'polite');
    
    let errorHTML = `
      <div class="form-errors-header">
        <i class="bi bi-exclamation-triangle-fill form-errors-icon" aria-hidden="true"></i>
        <h3 class="form-errors-title">Please fix the following errors:</h3>
      </div>
      <ul class="form-errors-list">
    `;
    
    Object.entries(this.errors).forEach(([field, message]) => {
      errorHTML += `
        <li>
          <i class="bi bi-arrow-right" aria-hidden="true"></i>
          <a href="#${field}" onclick="document.getElementById('${field}').focus(); return false;">
            ${message}
          </a>
        </li>
      `;
    });
    
    errorHTML += '</ul>';
    errorSummary.innerHTML = errorHTML;
    
    // Insert at top of form
    this.form.insertBefore(errorSummary, this.form.firstChild);
    
    // Focus error summary
    errorSummary.focus();
    errorSummary.scrollIntoView({ behavior: 'smooth', block: 'center' });
  }

  setupProgressiveDisclosure() {
    const sections = this.form.querySelectorAll('.form-section');
    
    sections.forEach(section => {
      const header = section.querySelector('.form-section-header');
      const content = section.querySelector('.form-section-content');
      
      if (header && content) {
        header.addEventListener('click', () => {
          const isExpanded = section.classList.contains('expanded');
          
          // Toggle section
          section.classList.toggle('expanded');
          
          // Update ARIA attributes
          header.setAttribute('aria-expanded', (!isExpanded).toString());
          content.setAttribute('aria-hidden', isExpanded.toString());
          
          // Focus first field when expanded
          if (!isExpanded) {
            const firstField = content.querySelector('input, textarea, select');
            if (firstField) {
              setTimeout(() => firstField.focus(), 300);
            }
          }
        });
        
        // Set initial ARIA attributes
        header.setAttribute('aria-expanded', 'false');
        header.setAttribute('role', 'button');
        header.setAttribute('tabindex', '0');
        content.setAttribute('aria-hidden', 'true');
        
        // Keyboard support
        header.addEventListener('keydown', (e) => {
          if (e.key === 'Enter' || e.key === ' ') {
            e.preventDefault();
            header.click();
          }
        });
      }
    });
  }

  setupAutoComplete() {
    if (!this.options.enableAutoComplete) return;
    
    const fields = this.form.querySelectorAll('input[type="text"], input[type="email"], input[type="tel"]');
    
    fields.forEach(field => {
      // Add appropriate autocomplete attributes
      const fieldName = field.name.toLowerCase();
      
      if (fieldName.includes('email')) {
        field.setAttribute('autocomplete', 'email');
      } else if (fieldName.includes('phone') || fieldName.includes('tel')) {
        field.setAttribute('autocomplete', 'tel');
      } else if (fieldName.includes('name')) {
        if (fieldName.includes('first')) {
          field.setAttribute('autocomplete', 'given-name');
        } else if (fieldName.includes('last')) {
          field.setAttribute('autocomplete', 'family-name');
        } else {
          field.setAttribute('autocomplete', 'name');
        }
      } else if (fieldName.includes('address')) {
        field.setAttribute('autocomplete', 'street-address');
      } else if (fieldName.includes('city')) {
        field.setAttribute('autocomplete', 'address-level2');
      } else if (fieldName.includes('state')) {
        field.setAttribute('autocomplete', 'address-level1');
      } else if (fieldName.includes('zip') || fieldName.includes('postal')) {
        field.setAttribute('autocomplete', 'postal-code');
      }
    });
  }

  setupFileUploads() {
    const fileInputs = this.form.querySelectorAll('input[type="file"]');
    
    fileInputs.forEach(input => {
      const label = input.nextElementSibling;
      
      input.addEventListener('change', (e) => {
        const files = e.target.files;
        
        if (files.length > 0) {
          const fileName = files[0].name;
          const fileSize = this.formatFileSize(files[0].size);
          
          // Update label text
          if (label) {
            label.innerHTML = `
              <i class="bi bi-file-earmark-check" aria-hidden="true"></i>
              <span>${fileName} (${fileSize})</span>
            `;
          }
          
          // Show file info
          let fileInfo = input.parentNode.querySelector('.form-file-info');
          if (!fileInfo) {
            fileInfo = document.createElement('div');
            fileInfo.className = 'form-file-info';
            input.parentNode.appendChild(fileInfo);
          }
          
          fileInfo.innerHTML = `
            <i class="bi bi-info-circle" aria-hidden="true"></i>
            Selected: ${fileName} (${fileSize})
          `;
        }
      });
    });
  }

  setupAccessibilityFeatures() {
    // Add live region for announcements
    if (!document.getElementById('form-live-region')) {
      const liveRegion = document.createElement('div');
      liveRegion.id = 'form-live-region';
      liveRegion.className = 'sr-only';
      liveRegion.setAttribute('aria-live', 'polite');
      liveRegion.setAttribute('aria-atomic', 'true');
      document.body.appendChild(liveRegion);
    }
    
    // Announce form changes
    this.liveRegion = document.getElementById('form-live-region');
  }

  setupProgressIndicator() {
    // Look for multi-step form sections
    const sections = this.form.querySelectorAll('.form-section');
    if (sections.length <= 1) return;
    
    // Create progress indicator
    const progressDiv = document.createElement('div');
    progressDiv.className = 'form-progress';
    progressDiv.setAttribute('role', 'progressbar');
    progressDiv.setAttribute('aria-label', 'Form completion progress');
    
    let progressHTML = '';
    sections.forEach((section, index) => {
      const title = section.querySelector('.form-section-title')?.textContent || `Step ${index + 1}`;
      const stepClass = index === 0 ? 'active' : '';
      
      progressHTML += `
        <div class="form-progress-step ${stepClass}">
          <div class="form-progress-number">${index + 1}</div>
          <div class="form-progress-label">${title}</div>
        </div>
      `;
    });
    
    progressDiv.innerHTML = progressHTML;
    
    // Insert at top of form
    this.form.insertBefore(progressDiv, this.form.firstChild);
    
    // Update progress when sections are expanded
    sections.forEach((section, index) => {
      const header = section.querySelector('.form-section-header');
      if (header) {
        header.addEventListener('click', () => {
          setTimeout(() => {
            this.updateProgress(index);
          }, 100);
        });
      }
    });
  }

  updateProgress(activeIndex) {
    const steps = this.form.querySelectorAll('.form-progress-step');
    
    steps.forEach((step, index) => {
      step.classList.remove('active', 'completed');
      
      if (index < activeIndex) {
        step.classList.add('completed');
      } else if (index === activeIndex) {
        step.classList.add('active');
      }
    });
    
    // Announce progress
    if (this.liveRegion) {
      this.liveRegion.textContent = `Step ${activeIndex + 1} of ${steps.length}`;
    }
  }

  setLoadingState(loading) {
    if (loading) {
      this.form.classList.add('form-loading');
      
      // Disable all form controls
      const controls = this.form.querySelectorAll('input, textarea, select, button');
      controls.forEach(control => {
        control.disabled = true;
      });
      
      // Announce loading
      if (this.liveRegion) {
        this.liveRegion.textContent = 'Form is being submitted, please wait...';
      }
    } else {
      this.form.classList.remove('form-loading');
      this.form.dataset.submitting = 'false';
      
      // Re-enable form controls
      const controls = this.form.querySelectorAll('input, textarea, select, button');
      controls.forEach(control => {
        control.disabled = false;
      });
    }
  }

  // Utility methods
  getFieldLabel(field) {
    const label = this.form.querySelector(`label[for="${field.id}"]`);
    return label ? label.textContent.replace('*', '').trim() : field.name;
  }

  isValidEmail(email) {
    const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
    return emailRegex.test(email);
  }

  isValidPhone(phone) {
    const phoneRegex = /^[\+]?[\d\s\-\(\)]{10,}$/;
    return phoneRegex.test(phone);
  }

  isValidDate(dateString) {
    const date = new Date(dateString);
    return date instanceof Date && !isNaN(date);
  }

  formatFileSize(bytes) {
    if (bytes === 0) return '0 Bytes';
    const k = 1024;
    const sizes = ['Bytes', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  }

  // Public methods
  reset() {
    this.form.reset();
    this.errors = {};
    this.touchedFields.clear();
    this.isValid = true;
    
    // Remove validation classes
    const fields = this.form.querySelectorAll('input, textarea, select');
    fields.forEach(field => {
      this.clearFieldError(field);
    });
    
    // Remove error summary
    const errorSummary = this.form.querySelector('.form-errors');
    if (errorSummary) {
      errorSummary.remove();
    }
  }

  validate() {
    return this.validateForm();
  }

  // Static method to initialize all forms
  static initAll(selector = '.accessible-form, form', options = {}) {
    const forms = document.querySelectorAll(selector);
    const instances = [];
    
    forms.forEach(form => {
      const instance = new AccessibleForm(form, options);
      instances.push(instance);
      
      // Store instance on element
      form._accessibleFormInstance = instance;
    });
    
    return instances;
  }
}

// Auto-initialize on DOM ready
document.addEventListener('DOMContentLoaded', () => {
  AccessibleForm.initAll();
});

// Export for module use
if (typeof module !== 'undefined' && module.exports) {
  module.exports = AccessibleForm;
}