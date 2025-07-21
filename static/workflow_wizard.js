// Workflow Wizard System
// Provides step-by-step guidance for common tasks

class WorkflowWizard {
    constructor(containerId, steps) {
        this.container = document.getElementById(containerId);
        this.steps = steps;
        this.currentStep = 0;
        this.data = {};
        this.init();
    }

    init() {
        this.render();
        this.attachEventListeners();
    }

    render() {
        const step = this.steps[this.currentStep];
        const totalSteps = this.steps.length;
        
        this.container.innerHTML = `
            <div class="wizard-container">
                <div class="wizard-header">
                    <h2>${step.title}</h2>
                    <div class="wizard-progress">
                        <div class="progress-bar">
                            <div class="progress-fill" style="width: ${((this.currentStep + 1) / totalSteps) * 100}%"></div>
                        </div>
                        <span class="progress-text">Step ${this.currentStep + 1} of ${totalSteps}</span>
                    </div>
                </div>
                
                <div class="wizard-body">
                    <p class="wizard-description">${step.description}</p>
                    ${this.renderStepContent(step)}
                </div>
                
                <div class="wizard-footer">
                    ${this.currentStep > 0 ? '<button class="btn btn-secondary" id="wizard-prev">Previous</button>' : ''}
                    ${this.currentStep < totalSteps - 1 ? 
                        '<button class="btn btn-primary" id="wizard-next">Next</button>' : 
                        '<button class="btn btn-success" id="wizard-complete">Complete</button>'
                    }
                </div>
            </div>
        `;
    }

    renderStepContent(step) {
        switch (step.type) {
            case 'form':
                return this.renderForm(step.fields);
            case 'checklist':
                return this.renderChecklist(step.items);
            case 'info':
                return this.renderInfo(step.content);
            case 'confirmation':
                return this.renderConfirmation(step.summary);
            default:
                return '';
        }
    }

    renderForm(fields) {
        return fields.map(field => `
            <div class="form-group">
                <label for="${field.id}" class="form-label ${field.required ? 'required' : ''}">
                    ${field.label}
                </label>
                ${field.help ? `
                    <div class="form-help">
                        <i class="bi bi-info-circle" aria-hidden="true"></i>
                        ${field.help}
                    </div>
                ` : ''}
                ${this.renderField(field)}
            </div>
        `).join('');
    }

    renderField(field) {
        switch (field.type) {
            case 'text':
            case 'number':
            case 'date':
            case 'time':
                return `<input type="${field.type}" id="${field.id}" name="${field.id}" 
                    class="form-control" ${field.required ? 'required' : ''} 
                    ${field.placeholder ? `placeholder="${field.placeholder}"` : ''}
                    value="${this.data[field.id] || ''}">`;
            
            case 'select':
                return `
                    <select id="${field.id}" name="${field.id}" class="form-control" ${field.required ? 'required' : ''}>
                        <option value="">Select ${field.label}</option>
                        ${field.options.map(opt => `
                            <option value="${opt.value}" ${this.data[field.id] === opt.value ? 'selected' : ''}>
                                ${opt.label}
                            </option>
                        `).join('')}
                    </select>
                `;
            
            case 'textarea':
                return `<textarea id="${field.id}" name="${field.id}" 
                    class="form-control" rows="4" ${field.required ? 'required' : ''}
                    ${field.placeholder ? `placeholder="${field.placeholder}"` : ''}>${this.data[field.id] || ''}</textarea>`;
            
            default:
                return '';
        }
    }

    renderChecklist(items) {
        return `
            <div class="wizard-checklist">
                ${items.map((item, index) => `
                    <div class="checklist-item">
                        <input type="checkbox" id="check-${index}" name="check-${index}" 
                            ${this.data[`check-${index}`] ? 'checked' : ''}>
                        <label for="check-${index}">${item}</label>
                    </div>
                `).join('')}
            </div>
        `;
    }

    renderInfo(content) {
        return `
            <div class="wizard-info">
                ${content}
            </div>
        `;
    }

    renderConfirmation(summary) {
        return `
            <div class="wizard-confirmation">
                <h3>Please Review Your Information</h3>
                <dl class="confirmation-list">
                    ${Object.entries(summary).map(([key, label]) => `
                        <dt>${label}:</dt>
                        <dd>${this.data[key] || 'Not provided'}</dd>
                    `).join('')}
                </dl>
            </div>
        `;
    }

    attachEventListeners() {
        this.container.addEventListener('click', (e) => {
            if (e.target.id === 'wizard-next') {
                this.handleNext();
            } else if (e.target.id === 'wizard-prev') {
                this.handlePrev();
            } else if (e.target.id === 'wizard-complete') {
                this.handleComplete();
            }
        });

        // Save form data on input
        this.container.addEventListener('input', (e) => {
            if (e.target.name) {
                this.data[e.target.name] = e.target.type === 'checkbox' ? 
                    e.target.checked : e.target.value;
            }
        });
    }

    handleNext() {
        if (this.validateCurrentStep()) {
            this.currentStep++;
            this.render();
        }
    }

    handlePrev() {
        this.currentStep--;
        this.render();
    }

    handleComplete() {
        if (this.validateCurrentStep()) {
            if (this.onComplete) {
                this.onComplete(this.data);
            }
        }
    }

    validateCurrentStep() {
        const step = this.steps[this.currentStep];
        
        if (step.type === 'form') {
            const requiredFields = step.fields.filter(f => f.required);
            for (const field of requiredFields) {
                const value = this.data[field.id];
                if (!value || value.trim() === '') {
                    alert(`Please fill in ${field.label}`);
                    document.getElementById(field.id)?.focus();
                    return false;
                }
            }
        }
        
        if (step.type === 'checklist' && step.requireAll) {
            const allChecked = step.items.every((_, index) => 
                this.data[`check-${index}`] === true
            );
            if (!allChecked) {
                alert('Please complete all checklist items before proceeding.');
                return false;
            }
        }
        
        return true;
    }
}

// Common workflow definitions
const workflows = {
    addBus: [
        {
            title: "Bus Information",
            description: "Enter the basic information for the new bus",
            type: "form",
            fields: [
                {
                    id: "bus_id",
                    label: "Bus ID/Number",
                    type: "text",
                    required: true,
                    placeholder: "e.g., BUS-001",
                    help: "Unique identifier for this bus"
                },
                {
                    id: "model",
                    label: "Bus Model",
                    type: "text",
                    required: true,
                    placeholder: "e.g., Blue Bird Vision"
                },
                {
                    id: "capacity",
                    label: "Seating Capacity",
                    type: "number",
                    required: true,
                    placeholder: "e.g., 72"
                }
            ]
        },
        {
            title: "Safety Checklist",
            description: "Confirm all safety requirements are met",
            type: "checklist",
            requireAll: true,
            items: [
                "Valid registration and insurance",
                "Recent safety inspection completed",
                "Emergency exits clearly marked",
                "First aid kit installed",
                "Fire extinguisher present and inspected",
                "Two-way radio or communication device installed"
            ]
        },
        {
            title: "Initial Status",
            description: "Set the initial maintenance status",
            type: "form",
            fields: [
                {
                    id: "oil_status",
                    label: "Oil Status",
                    type: "select",
                    required: true,
                    options: [
                        { value: "good", label: "Good - Recently Changed" },
                        { value: "due_soon", label: "Due Soon - Monitor" },
                        { value: "overdue", label: "Overdue - Change Immediately" }
                    ]
                },
                {
                    id: "tire_status",
                    label: "Tire Status",
                    type: "select",
                    required: true,
                    options: [
                        { value: "good", label: "Good - Adequate Tread" },
                        { value: "monitor", label: "Monitor - Some Wear" },
                        { value: "replace", label: "Replace - Worn" }
                    ]
                },
                {
                    id: "notes",
                    label: "Additional Notes",
                    type: "textarea",
                    required: false,
                    placeholder: "Any special notes about this bus..."
                }
            ]
        },
        {
            title: "Review & Confirm",
            description: "Review the information before adding the bus",
            type: "confirmation",
            summary: {
                bus_id: "Bus ID",
                model: "Model",
                capacity: "Capacity",
                oil_status: "Oil Status",
                tire_status: "Tire Status",
                notes: "Notes"
            }
        }
    ],
    
    assignRoute: [
        {
            title: "Select Driver",
            description: "Choose a driver for this route assignment",
            type: "form",
            fields: [
                {
                    id: "driver",
                    label: "Driver",
                    type: "select",
                    required: true,
                    options: [] // Will be populated dynamically
                }
            ]
        },
        {
            title: "Select Bus",
            description: "Choose an available bus for this route",
            type: "form",
            fields: [
                {
                    id: "bus_id",
                    label: "Bus",
                    type: "select",
                    required: true,
                    options: [] // Will be populated dynamically
                }
            ]
        },
        {
            title: "Select Route",
            description: "Choose the route to assign",
            type: "form",
            fields: [
                {
                    id: "route_id",
                    label: "Route",
                    type: "select",
                    required: true,
                    options: [] // Will be populated dynamically
                }
            ]
        },
        {
            title: "Pre-Assignment Checklist",
            description: "Confirm these items before assigning the route",
            type: "checklist",
            items: [
                "Driver has valid CDL and medical certificate",
                "Driver is familiar with the route",
                "Bus has been inspected today",
                "Route schedule has been reviewed with driver",
                "Emergency contact information is up to date"
            ]
        },
        {
            title: "Review Assignment",
            description: "Confirm the route assignment details",
            type: "confirmation",
            summary: {
                driver: "Driver",
                bus_id: "Bus",
                route_id: "Route"
            }
        }
    ]
};

// Export for use in other files
window.WorkflowWizard = WorkflowWizard;
window.workflows = workflows;