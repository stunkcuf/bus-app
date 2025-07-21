// Step-by-Step Wizard Framework
class StepWizard {
    constructor(options) {
        this.containerId = options.containerId;
        this.steps = options.steps || [];
        this.currentStep = 0;
        this.data = {};
        this.onComplete = options.onComplete || function() {};
        this.onCancel = options.onCancel || function() {};
        
        this.init();
    }
    
    init() {
        this.container = document.getElementById(this.containerId);
        if (!this.container) {
            console.error(`Container with ID ${this.containerId} not found`);
            return;
        }
        
        this.render();
    }
    
    render() {
        const step = this.steps[this.currentStep];
        
        this.container.innerHTML = `
            <div class="wizard-container">
                <div class="wizard-header">
                    <h2>${step.title}</h2>
                    <div class="wizard-progress">
                        <div class="progress-bar">
                            <div class="progress-fill" style="width: ${((this.currentStep + 1) / this.steps.length) * 100}%"></div>
                        </div>
                        <span class="progress-text">Step ${this.currentStep + 1} of ${this.steps.length}</span>
                    </div>
                </div>
                
                <div class="wizard-body">
                    <div class="wizard-description">${step.description || ''}</div>
                    <div class="wizard-content" id="wizard-step-content">
                        ${this.renderStepContent(step)}
                    </div>
                    ${step.help ? `<div class="wizard-help"><i class="bi bi-info-circle"></i> ${step.help}</div>` : ''}
                </div>
                
                <div class="wizard-footer">
                    <button type="button" class="btn btn-secondary" onclick="wizard.cancel()">
                        <i class="bi bi-x-circle"></i> Cancel
                    </button>
                    ${this.currentStep > 0 ? `
                        <button type="button" class="btn btn-outline-primary" onclick="wizard.previousStep()">
                            <i class="bi bi-arrow-left"></i> Previous
                        </button>
                    ` : ''}
                    ${this.currentStep < this.steps.length - 1 ? `
                        <button type="button" class="btn btn-primary" onclick="wizard.nextStep()">
                            Next <i class="bi bi-arrow-right"></i>
                        </button>
                    ` : `
                        <button type="button" class="btn btn-success" onclick="wizard.complete()">
                            <i class="bi bi-check-circle"></i> Complete
                        </button>
                    `}
                </div>
            </div>
        `;
        
        // Set up any step-specific initialization
        if (step.init) {
            step.init(this);
        }
    }
    
    renderStepContent(step) {
        if (step.render) {
            return step.render(this.data);
        }
        
        // Default rendering for common field types
        let html = '<div class="form-group">';
        
        if (step.fields) {
            step.fields.forEach(field => {
                html += this.renderField(field);
            });
        }
        
        html += '</div>';
        return html;
    }
    
    renderField(field) {
        let html = `<div class="mb-3">`;
        
        if (field.label) {
            html += `<label for="${field.id}" class="form-label">${field.label}`;
            if (field.required) html += ' <span class="text-danger">*</span>';
            html += '</label>';
        }
        
        switch (field.type) {
            case 'select':
                html += `<select class="form-control" id="${field.id}" ${field.required ? 'required' : ''}>`;
                html += `<option value="">Select ${field.label}...</option>`;
                if (field.options) {
                    field.options.forEach(opt => {
                        const selected = this.data[field.id] === opt.value ? 'selected' : '';
                        html += `<option value="${opt.value}" ${selected}>${opt.text}</option>`;
                    });
                }
                html += '</select>';
                break;
                
            case 'text':
            case 'number':
            case 'date':
            case 'time':
                const value = this.data[field.id] || '';
                html += `<input type="${field.type}" class="form-control" id="${field.id}" 
                         value="${value}" ${field.required ? 'required' : ''} 
                         ${field.placeholder ? `placeholder="${field.placeholder}"` : ''}>`;
                break;
                
            case 'textarea':
                const textValue = this.data[field.id] || '';
                html += `<textarea class="form-control" id="${field.id}" rows="3" 
                         ${field.required ? 'required' : ''}
                         ${field.placeholder ? `placeholder="${field.placeholder}"` : ''}>${textValue}</textarea>`;
                break;
                
            case 'radio':
                if (field.options) {
                    field.options.forEach(opt => {
                        const checked = this.data[field.id] === opt.value ? 'checked' : '';
                        html += `
                            <div class="form-check">
                                <input class="form-check-input" type="radio" name="${field.id}" 
                                       id="${field.id}_${opt.value}" value="${opt.value}" ${checked}>
                                <label class="form-check-label" for="${field.id}_${opt.value}">
                                    ${opt.text}
                                </label>
                            </div>
                        `;
                    });
                }
                break;
                
            case 'checkbox':
                const isChecked = this.data[field.id] ? 'checked' : '';
                html += `
                    <div class="form-check">
                        <input class="form-check-input" type="checkbox" id="${field.id}" ${isChecked}>
                        <label class="form-check-label" for="${field.id}">
                            ${field.checkLabel || field.label}
                        </label>
                    </div>
                `;
                break;
        }
        
        if (field.help) {
            html += `<small class="form-text text-muted">${field.help}</small>`;
        }
        
        html += '</div>';
        return html;
    }
    
    collectStepData() {
        const step = this.steps[this.currentStep];
        
        if (step.fields) {
            step.fields.forEach(field => {
                const element = document.getElementById(field.id);
                if (element) {
                    if (field.type === 'checkbox') {
                        this.data[field.id] = element.checked;
                    } else if (field.type === 'radio') {
                        const checked = document.querySelector(`input[name="${field.id}"]:checked`);
                        if (checked) {
                            this.data[field.id] = checked.value;
                        }
                    } else {
                        this.data[field.id] = element.value;
                    }
                }
            });
        }
        
        // Custom data collection
        if (step.collect) {
            step.collect(this);
        }
    }
    
    validateStep() {
        const step = this.steps[this.currentStep];
        
        // Custom validation
        if (step.validate) {
            const result = step.validate(this.data);
            if (result !== true) {
                this.showError(result);
                return false;
            }
        }
        
        // Default validation for required fields
        if (step.fields) {
            for (let field of step.fields) {
                if (field.required) {
                    const value = this.data[field.id];
                    if (!value || value.trim() === '') {
                        this.showError(`${field.label} is required`);
                        return false;
                    }
                }
            }
        }
        
        return true;
    }
    
    showError(message) {
        // Show error in wizard body
        const errorDiv = document.createElement('div');
        errorDiv.className = 'alert alert-danger alert-dismissible fade show mt-3';
        errorDiv.innerHTML = `
            <i class="bi bi-exclamation-triangle"></i> ${message}
            <button type="button" class="btn-close" data-bs-dismiss="alert"></button>
        `;
        
        const content = document.getElementById('wizard-step-content');
        content.insertBefore(errorDiv, content.firstChild);
        
        // Auto-dismiss after 5 seconds
        setTimeout(() => {
            errorDiv.remove();
        }, 5000);
    }
    
    showSuccess(message) {
        const successDiv = document.createElement('div');
        successDiv.className = 'alert alert-success alert-dismissible fade show mt-3';
        successDiv.innerHTML = `
            <i class="bi bi-check-circle"></i> ${message}
            <button type="button" class="btn-close" data-bs-dismiss="alert"></button>
        `;
        
        const content = document.getElementById('wizard-step-content');
        content.insertBefore(successDiv, content.firstChild);
        
        // Auto-dismiss after 3 seconds
        setTimeout(() => {
            successDiv.remove();
        }, 3000);
    }
    
    nextStep() {
        this.collectStepData();
        
        if (!this.validateStep()) {
            return;
        }
        
        if (this.currentStep < this.steps.length - 1) {
            this.currentStep++;
            this.render();
        }
    }
    
    previousStep() {
        if (this.currentStep > 0) {
            this.currentStep--;
            this.render();
        }
    }
    
    complete() {
        this.collectStepData();
        
        if (!this.validateStep()) {
            return;
        }
        
        this.onComplete(this.data);
    }
    
    cancel() {
        if (confirm('Are you sure you want to cancel? All progress will be lost.')) {
            this.onCancel();
        }
    }
}

// Export for use
window.StepWizard = StepWizard;