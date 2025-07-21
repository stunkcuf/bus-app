// Form Validation and Error Prevention System

class FormValidator {
    constructor(formId, rules) {
        this.form = document.getElementById(formId);
        this.rules = rules;
        this.errors = {};
        
        if (this.form) {
            this.init();
        }
    }
    
    init() {
        // Add real-time validation
        this.form.addEventListener('input', (e) => {
            if (e.target.name && this.rules[e.target.name]) {
                this.validateField(e.target);
            }
        });
        
        // Add blur validation for better UX
        this.form.addEventListener('blur', (e) => {
            if (e.target.name && this.rules[e.target.name]) {
                this.validateField(e.target);
            }
        }, true);
        
        // Validate on submit
        this.form.addEventListener('submit', (e) => {
            if (!this.validateForm()) {
                e.preventDefault();
                this.showErrorSummary();
            }
        });
    }
    
    validateField(field) {
        const name = field.name;
        const value = field.value;
        const rules = this.rules[name];
        
        // Clear previous errors
        this.clearFieldError(field);
        delete this.errors[name];
        
        if (!rules) return true;
        
        // Check each rule
        for (const rule of rules) {
            const error = this.checkRule(value, rule, field);
            if (error) {
                this.errors[name] = error;
                this.showFieldError(field, error);
                return false;
            }
        }
        
        // Show success state if field was previously invalid
        if (field.classList.contains('is-invalid')) {
            this.showFieldSuccess(field);
        }
        
        return true;
    }
    
    checkRule(value, rule, field) {
        switch (rule.type) {
            case 'required':
                if (!value || value.trim() === '') {
                    return rule.message || 'This field is required';
                }
                break;
                
            case 'minLength':
                if (value.length < rule.value) {
                    return rule.message || `Minimum ${rule.value} characters required`;
                }
                break;
                
            case 'maxLength':
                if (value.length > rule.value) {
                    return rule.message || `Maximum ${rule.value} characters allowed`;
                }
                break;
                
            case 'pattern':
                if (!rule.value.test(value)) {
                    return rule.message || 'Invalid format';
                }
                break;
                
            case 'email':
                const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
                if (!emailRegex.test(value)) {
                    return rule.message || 'Please enter a valid email address';
                }
                break;
                
            case 'match':
                const matchField = this.form.querySelector(`[name="${rule.value}"]`);
                if (matchField && value !== matchField.value) {
                    return rule.message || 'Fields do not match';
                }
                break;
                
            case 'custom':
                if (rule.validator && !rule.validator(value, field)) {
                    return rule.message || 'Invalid value';
                }
                break;
        }
        
        return null;
    }
    
    validateForm() {
        let isValid = true;
        
        // Validate all fields with rules
        Object.keys(this.rules).forEach(fieldName => {
            const field = this.form.querySelector(`[name="${fieldName}"]`);
            if (field && !this.validateField(field)) {
                isValid = false;
            }
        });
        
        return isValid;
    }
    
    showFieldError(field, message) {
        field.classList.add('is-invalid');
        field.classList.remove('is-valid');
        
        // Create or update error message
        let errorElement = field.parentElement.querySelector('.field-error');
        if (!errorElement) {
            errorElement = document.createElement('div');
            errorElement.className = 'field-error';
            field.parentElement.appendChild(errorElement);
        }
        errorElement.textContent = message;
        
        // Add ARIA attributes for accessibility
        field.setAttribute('aria-invalid', 'true');
        field.setAttribute('aria-describedby', errorElement.id || this.generateErrorId(field));
        if (!errorElement.id) {
            errorElement.id = this.generateErrorId(field);
        }
    }
    
    showFieldSuccess(field) {
        field.classList.add('is-valid');
        field.classList.remove('is-invalid');
        field.setAttribute('aria-invalid', 'false');
    }
    
    clearFieldError(field) {
        field.classList.remove('is-invalid', 'is-valid');
        const errorElement = field.parentElement.querySelector('.field-error');
        if (errorElement) {
            errorElement.remove();
        }
        field.removeAttribute('aria-invalid');
        field.removeAttribute('aria-describedby');
    }
    
    showErrorSummary() {
        // Remove existing summary
        const existingSummary = this.form.querySelector('.error-summary');
        if (existingSummary) {
            existingSummary.remove();
        }
        
        if (Object.keys(this.errors).length === 0) return;
        
        // Create error summary
        const summary = document.createElement('div');
        summary.className = 'error-summary alert alert-danger';
        summary.setAttribute('role', 'alert');
        summary.innerHTML = `
            <h4>Please fix the following errors:</h4>
            <ul>
                ${Object.entries(this.errors).map(([field, error]) => {
                    const label = this.getFieldLabel(field);
                    return `<li><a href="#${field}">${label}: ${error}</a></li>`;
                }).join('')}
            </ul>
        `;
        
        // Insert at the beginning of the form
        this.form.insertBefore(summary, this.form.firstChild);
        
        // Focus on summary for screen readers
        summary.focus();
        
        // Add click handlers to jump to fields
        summary.querySelectorAll('a').forEach(link => {
            link.addEventListener('click', (e) => {
                e.preventDefault();
                const fieldName = link.getAttribute('href').substring(1);
                const field = this.form.querySelector(`[name="${fieldName}"]`);
                if (field) {
                    field.focus();
                    field.scrollIntoView({ behavior: 'smooth', block: 'center' });
                }
            });
        });
    }
    
    getFieldLabel(fieldName) {
        const field = this.form.querySelector(`[name="${fieldName}"]`);
        if (!field) return fieldName;
        
        // Try to find associated label
        const label = this.form.querySelector(`label[for="${field.id}"]`);
        if (label) {
            return label.textContent.replace('*', '').trim();
        }
        
        // Fallback to placeholder or name
        return field.placeholder || fieldName;
    }
    
    generateErrorId(field) {
        return `error-${field.name}-${Date.now()}`;
    }
}

// Common validation rules
const commonRules = {
    username: [
        { type: 'required', message: 'Username is required' },
        { type: 'minLength', value: 3, message: 'Username must be at least 3 characters' },
        { type: 'maxLength', value: 20, message: 'Username must be less than 20 characters' },
        { type: 'pattern', value: /^[a-zA-Z0-9_]+$/, message: 'Username can only contain letters, numbers, and underscores' }
    ],
    
    password: [
        { type: 'required', message: 'Password is required' },
        { type: 'minLength', value: 6, message: 'Password must be at least 6 characters' }
    ],
    
    confirm_password: [
        { type: 'required', message: 'Please confirm your password' },
        { type: 'match', value: 'password', message: 'Passwords do not match' }
    ],
    
    bus_id: [
        { type: 'required', message: 'Bus ID is required' },
        { type: 'pattern', value: /^[A-Z0-9-]+$/, message: 'Bus ID must contain only uppercase letters, numbers, and hyphens' }
    ],
    
    capacity: [
        { type: 'required', message: 'Capacity is required' },
        { type: 'custom', validator: (value) => {
            const num = parseInt(value);
            return !isNaN(num) && num > 0 && num <= 100;
        }, message: 'Capacity must be between 1 and 100' }
    ],
    
    phone: [
        { type: 'pattern', value: /^\d{3}-\d{3}-\d{4}$/, message: 'Phone must be in format: 123-456-7890' }
    ]
};

// Auto-save functionality for forms
class FormAutoSave {
    constructor(formId, storageKey) {
        this.form = document.getElementById(formId);
        this.storageKey = storageKey;
        this.saveTimeout = null;
        
        if (this.form) {
            this.init();
        }
    }
    
    init() {
        // Restore saved data
        this.restore();
        
        // Save on input with debounce
        this.form.addEventListener('input', () => {
            clearTimeout(this.saveTimeout);
            this.saveTimeout = setTimeout(() => this.save(), 1000);
        });
        
        // Clear on successful submit
        this.form.addEventListener('submit', () => {
            this.clear();
        });
        
        // Show restore prompt if data exists
        if (this.hasSavedData()) {
            this.showRestorePrompt();
        }
    }
    
    save() {
        const formData = new FormData(this.form);
        const data = {};
        
        for (const [key, value] of formData.entries()) {
            data[key] = value;
        }
        
        localStorage.setItem(this.storageKey, JSON.stringify({
            data: data,
            timestamp: Date.now()
        }));
        
        this.showSaveIndicator();
    }
    
    restore() {
        const saved = localStorage.getItem(this.storageKey);
        if (!saved) return;
        
        try {
            const { data, timestamp } = JSON.parse(saved);
            
            // Don't restore if data is older than 24 hours
            if (Date.now() - timestamp > 24 * 60 * 60 * 1000) {
                this.clear();
                return;
            }
            
            Object.entries(data).forEach(([key, value]) => {
                const field = this.form.querySelector(`[name="${key}"]`);
                if (field && field.type !== 'file') {
                    field.value = value;
                }
            });
        } catch (e) {
            console.error('Failed to restore form data:', e);
        }
    }
    
    clear() {
        localStorage.removeItem(this.storageKey);
    }
    
    hasSavedData() {
        return localStorage.getItem(this.storageKey) !== null;
    }
    
    showRestorePrompt() {
        const prompt = document.createElement('div');
        prompt.className = 'autosave-prompt alert alert-info';
        prompt.innerHTML = `
            <p>You have unsaved changes from a previous session. 
            <button type="button" class="btn btn-sm btn-primary" id="restore-data">Restore</button>
            <button type="button" class="btn btn-sm btn-secondary" id="discard-data">Discard</button>
            </p>
        `;
        
        this.form.insertBefore(prompt, this.form.firstChild);
        
        document.getElementById('restore-data').addEventListener('click', () => {
            prompt.remove();
        });
        
        document.getElementById('discard-data').addEventListener('click', () => {
            this.clear();
            prompt.remove();
            this.form.reset();
        });
    }
    
    showSaveIndicator() {
        let indicator = this.form.querySelector('.autosave-indicator');
        if (!indicator) {
            indicator = document.createElement('div');
            indicator.className = 'autosave-indicator';
            this.form.appendChild(indicator);
        }
        
        indicator.textContent = 'Saved';
        indicator.classList.add('show');
        
        setTimeout(() => {
            indicator.classList.remove('show');
        }, 2000);
    }
}

// Export for use
window.FormValidator = FormValidator;
window.FormAutoSave = FormAutoSave;
window.commonRules = commonRules;