// Maintenance Logging Wizard with Auto-Suggestions
function initMaintenanceWizard(vehicleType, vehicleId) {
    // Load suggestions when wizard starts
    let maintenanceSuggestions = [];
    loadMaintenanceSuggestions(vehicleType, vehicleId);
    
    const wizard = new StepWizard({
        containerId: 'wizardContainer',
        steps: [
            {
                title: 'Maintenance Suggestions',
                description: 'Review recommended maintenance for this vehicle.',
                render: function(data) {
                    return `
                        <div id="maintenanceSuggestions">
                            <div class="text-center">
                                <div class="spinner-border"></div>
                                <p>Loading maintenance suggestions...</p>
                            </div>
                        </div>
                        <div class="mt-4">
                            <h6>Or select a maintenance type:</h6>
                            <div class="maintenance-type-grid">
                                <button class="maintenance-type-card" onclick="selectMaintenanceType('oil_change')">
                                    <span class="icon">🛢️</span>
                                    <span>Oil Change</span>
                                </button>
                                <button class="maintenance-type-card" onclick="selectMaintenanceType('tire_service')">
                                    <span class="icon">🚙</span>
                                    <span>Tire Service</span>
                                </button>
                                <button class="maintenance-type-card" onclick="selectMaintenanceType('inspection')">
                                    <span class="icon">🔍</span>
                                    <span>Inspection</span>
                                </button>
                                <button class="maintenance-type-card" onclick="selectMaintenanceType('repair')">
                                    <span class="icon">🔧</span>
                                    <span>Repair</span>
                                </button>
                                <button class="maintenance-type-card" onclick="selectMaintenanceType('other')">
                                    <span class="icon">📋</span>
                                    <span>Other</span>
                                </button>
                            </div>
                        </div>
                        <style>
                            .maintenance-type-grid {
                                display: grid;
                                grid-template-columns: repeat(auto-fit, minmax(120px, 1fr));
                                gap: 15px;
                                margin-top: 15px;
                            }
                            .maintenance-type-card {
                                padding: 20px;
                                border: 2px solid #ddd;
                                border-radius: 8px;
                                background: white;
                                cursor: pointer;
                                transition: all 0.3s;
                                text-align: center;
                            }
                            .maintenance-type-card:hover {
                                border-color: #007bff;
                                transform: translateY(-2px);
                                box-shadow: 0 4px 8px rgba(0,0,0,0.1);
                            }
                            .maintenance-type-card .icon {
                                display: block;
                                font-size: 2em;
                                margin-bottom: 5px;
                            }
                            .suggestion-card {
                                cursor: pointer;
                                transition: all 0.3s;
                            }
                            .suggestion-card:hover {
                                transform: translateY(-2px);
                                box-shadow: 0 4px 8px rgba(0,0,0,0.1);
                            }
                        </style>
                    `;
                },
                init: function(wizard) {
                    // Display suggestions once loaded
                    setTimeout(() => {
                        const container = document.getElementById('maintenanceSuggestions');
                        if (container && maintenanceSuggestions.length > 0) {
                            container.innerHTML = renderMaintenanceSuggestions(maintenanceSuggestions);
                        } else if (container) {
                            container.innerHTML = '<p class="text-muted">No maintenance suggestions at this time.</p>';
                        }
                    }, 500);
                },
                collect: function(wizard) {
                    // Collect selected maintenance type
                    if (!wizard.data.category) {
                        wizard.data.category = 'other';
                    }
                },
                help: 'Review suggested maintenance based on mileage, time, and service history.'
            },
            {
                title: 'Maintenance Type',
                description: 'What type of maintenance are you logging?',
                fields: [
                    {
                        id: 'category',
                        type: 'radio',
                        label: 'Maintenance Category',
                        required: true,
                        options: [
                            { value: 'oil_change', text: '🛢️ Oil Change' },
                            { value: 'tire_service', text: '🚙 Tire Service' },
                            { value: 'inspection', text: '🔍 Inspection' },
                            { value: 'repair', text: '🔧 Repair' },
                            { value: 'other', text: '📋 Other' }
                        ]
                    }
                ],
                help: 'Select the category that best describes the maintenance performed.'
            },
            {
                title: 'Maintenance Details',
                description: 'Provide details about the maintenance performed.',
                fields: [
                    {
                        id: 'date',
                        type: 'date',
                        label: 'Date of Service',
                        required: true,
                        placeholder: 'Select date'
                    },
                    {
                        id: 'mileage',
                        type: 'number',
                        label: 'Current Mileage',
                        required: true,
                        placeholder: 'Enter current mileage',
                        help: 'Record the vehicle\'s mileage at the time of service.'
                    },
                    {
                        id: 'description',
                        type: 'text',
                        label: 'Description',
                        required: true,
                        placeholder: 'Start typing for suggestions...',
                        autocomplete: true
                    },
                    {
                        id: 'notes',
                        type: 'textarea',
                        label: 'Additional Notes',
                        required: false,
                        placeholder: 'Any additional details or observations...'
                    }
                ],
                init: function(wizard) {
                    // Set today's date as default
                    const dateInput = document.getElementById('date');
                    dateInput.value = new Date().toISOString().split('T')[0];
                    
                    // Initialize autocomplete for description field
                    const descriptionField = document.getElementById('description');
                    if (descriptionField) {
                        initMaintenanceAutocomplete('description', wizard.data.category || 'other');
                    }
                    
                    // Load last known mileage and suggest next value
                    fetch(`/api/vehicle-mileage/${vehicleType}/${vehicleId}`)
                        .then(response => response.json())
                        .then(data => {
                            if (data.lastMileage) {
                                const mileageInput = document.getElementById('mileage');
                                mileageInput.min = data.lastMileage;
                                mileageInput.placeholder = `Enter mileage (last recorded: ${data.lastMileage})`;
                            }
                        })
                        .catch(error => console.error('Failed to load mileage data'));
                    
                    // Auto-suggestions based on category
                    const notesTextarea = document.getElementById('notes');
                    const categoryInputs = document.querySelectorAll('input[name="category"]');
                    
                    categoryInputs.forEach(input => {
                        input.addEventListener('change', function() {
                            const suggestions = {
                                'oil_change': 'Changed oil and filter. Next service due at ',
                                'tire_service': 'Tire rotation performed. Tire pressure checked and adjusted. ',
                                'inspection': 'Annual inspection completed. ',
                                'repair': 'Repaired ',
                                'other': ''
                            };
                            
                            if (suggestions[this.value] && notesTextarea.value === '') {
                                notesTextarea.value = suggestions[this.value];
                                notesTextarea.focus();
                                notesTextarea.setSelectionRange(notesTextarea.value.length, notesTextarea.value.length);
                            }
                        });
                    });
                },
                validate: function(data) {
                    const mileage = parseInt(data.mileage);
                    if (isNaN(mileage) || mileage <= 0) {
                        return 'Please enter a valid mileage';
                    }
                    
                    const selectedDate = new Date(data.date);
                    const today = new Date();
                    if (selectedDate > today) {
                        return 'Service date cannot be in the future';
                    }
                    
                    if (!data.notes || data.notes.trim().length < 10) {
                        return 'Please provide more detailed service notes (at least 10 characters)';
                    }
                    
                    return true;
                }
            },
            {
                title: 'Service Cost',
                description: 'Enter the cost information for this maintenance.',
                fields: [
                    {
                        id: 'cost',
                        type: 'number',
                        label: 'Total Cost',
                        required: true,
                        placeholder: '0.00',
                        help: 'Enter the total cost including parts and labor.'
                    },
                    {
                        id: 'vendor',
                        type: 'text',
                        label: 'Service Provider',
                        required: false,
                        placeholder: 'Shop name or technician'
                    },
                    {
                        id: 'invoice',
                        type: 'text',
                        label: 'Invoice/Work Order #',
                        required: false,
                        placeholder: 'Optional reference number'
                    }
                ],
                init: function(wizard) {
                    // Format cost input
                    const costInput = document.getElementById('cost');
                    costInput.addEventListener('blur', function() {
                        if (this.value) {
                            this.value = parseFloat(this.value).toFixed(2);
                        }
                    });
                    
                    // Load common vendors for auto-complete
                    fetch('/api/maintenance-vendors')
                        .then(response => response.json())
                        .then(vendors => {
                            const vendorInput = document.getElementById('vendor');
                            const datalist = document.createElement('datalist');
                            datalist.id = 'vendorList';
                            
                            vendors.forEach(vendor => {
                                const option = document.createElement('option');
                                option.value = vendor;
                                datalist.appendChild(option);
                            });
                            
                            vendorInput.setAttribute('list', 'vendorList');
                            vendorInput.parentNode.appendChild(datalist);
                        })
                        .catch(error => console.error('Failed to load vendors'));
                },
                validate: function(data) {
                    const cost = parseFloat(data.cost);
                    if (isNaN(cost) || cost < 0) {
                        return 'Please enter a valid cost amount';
                    }
                    return true;
                }
            },
            {
                title: 'Review & Submit',
                description: 'Please review the maintenance record before submitting.',
                render: function(data) {
                    const cost = parseFloat(data.cost || 0).toFixed(2);
                    const categoryNames = {
                        'oil_change': 'Oil Change',
                        'tire_service': 'Tire Service',
                        'inspection': 'Inspection',
                        'repair': 'Repair',
                        'other': 'Other'
                    };
                    
                    return `
                        <div class="wizard-summary">
                            <h4>Maintenance Summary</h4>
                            <table>
                                <tr>
                                    <td>Vehicle:</td>
                                    <td>${vehicleType.toUpperCase()} ${vehicleId}</td>
                                </tr>
                                <tr>
                                    <td>Category:</td>
                                    <td>${categoryNames[data.category] || data.category}</td>
                                </tr>
                                <tr>
                                    <td>Date:</td>
                                    <td>${new Date(data.date).toLocaleDateString()}</td>
                                </tr>
                                <tr>
                                    <td>Mileage:</td>
                                    <td>${parseInt(data.mileage).toLocaleString()} miles</td>
                                </tr>
                                <tr>
                                    <td>Cost:</td>
                                    <td>$${cost}</td>
                                </tr>
                                ${data.vendor ? `
                                <tr>
                                    <td>Service Provider:</td>
                                    <td>${data.vendor}</td>
                                </tr>` : ''}
                                ${data.invoice ? `
                                <tr>
                                    <td>Invoice #:</td>
                                    <td>${data.invoice}</td>
                                </tr>` : ''}
                            </table>
                            
                            <div class="mt-3">
                                <h5>Service Notes:</h5>
                                <div class="p-3 bg-light rounded">${data.notes}</div>
                            </div>
                        </div>
                        
                        <div class="form-check mt-3">
                            <input class="form-check-input" type="checkbox" id="confirmAccuracy">
                            <label class="form-check-label" for="confirmAccuracy">
                                I confirm that this maintenance information is accurate and complete
                            </label>
                        </div>
                    `;
                },
                validate: function(data) {
                    const confirmBox = document.getElementById('confirmAccuracy');
                    if (!confirmBox.checked) {
                        return 'Please confirm the accuracy of the maintenance information';
                    }
                    return true;
                }
            }
        ],
        onComplete: function(data) {
            // Submit the maintenance record
            const wizardContainer = document.getElementById('wizardContainer');
            wizardContainer.innerHTML = '<div class="wizard-loading"><div class="spinner-border"></div><p>Saving maintenance record...</p></div>';
            
            // Get CSRF token
            const csrfToken = document.querySelector('input[name="csrf_token"]').value;
            
            // Prepare form data
            const formData = new URLSearchParams();
            formData.append('vehicle_type', vehicleType);
            formData.append('vehicle_id', vehicleId);
            formData.append('date', data.date);
            formData.append('category', data.category);
            formData.append('notes', data.notes);
            formData.append('mileage', data.mileage);
            formData.append('cost', data.cost);
            if (data.vendor) formData.append('vendor', data.vendor);
            if (data.invoice) formData.append('invoice', data.invoice);
            formData.append('csrf_token', csrfToken);
            
            fetch('/save-maintenance-record', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/x-www-form-urlencoded',
                },
                body: formData.toString()
            })
            .then(response => {
                if (response.ok) {
                    wizardContainer.innerHTML = `
                        <div class="wizard-success">
                            <i class="bi bi-check-circle"></i>
                            <h3>Maintenance Record Saved!</h3>
                            <p>The maintenance record has been successfully added to the vehicle's history.</p>
                            <div class="mt-3">
                                <button class="btn btn-primary" onclick="location.reload()">
                                    <i class="bi bi-arrow-clockwise"></i> View Maintenance History
                                </button>
                                <button class="btn btn-outline-primary ms-2" onclick="initMaintenanceWizard('${vehicleType}', '${vehicleId}')">
                                    <i class="bi bi-plus-circle"></i> Add Another Record
                                </button>
                            </div>
                        </div>
                    `;
                } else {
                    throw new Error('Failed to save maintenance record');
                }
            })
            .catch(error => {
                wizardContainer.innerHTML = `
                    <div class="alert alert-danger">
                        <i class="bi bi-exclamation-triangle"></i> Failed to save maintenance record. Please try again.
                        <button class="btn btn-primary mt-3" onclick="location.reload()">
                            <i class="bi bi-arrow-left"></i> Go Back
                        </button>
                    </div>
                `;
            });
        },
        onCancel: function() {
            // Close the wizard
            document.getElementById('wizardContainer').style.display = 'none';
        }
    });
    
    window.wizard = wizard;
}

// Load maintenance suggestions from the server
async function loadMaintenanceSuggestions(vehicleType, vehicleId) {
    try {
        const response = await fetch(`/api/maintenance/suggestions?vehicle_type=${vehicleType}&vehicle_id=${vehicleId}`);
        if (response.ok) {
            maintenanceSuggestions = await response.json();
        }
    } catch (error) {
        console.error('Error loading maintenance suggestions:', error);
    }
}

// Render maintenance suggestions
function renderMaintenanceSuggestions(suggestions) {
    let html = '<div class="maintenance-suggestions">';
    html += '<h6><i class="bi bi-clipboard-check"></i> Recommended Maintenance</h6>';
    
    if (suggestions.length === 0) {
        html += '<p class="text-muted">No maintenance recommendations at this time.</p>';
    } else {
        suggestions.forEach((suggestion, index) => {
            const priorityClass = suggestion.priority === 'high' ? 'danger' : 
                                 suggestion.priority === 'medium' ? 'warning' : 'secondary';
            
            html += `
                <div class="suggestion-card mb-3 p-3 border rounded" onclick="useSuggestion(${index})">
                    <div class="d-flex justify-content-between align-items-start">
                        <div>
                            <h6 class="mb-1">
                                <span class="me-2">${suggestion.icon}</span>
                                ${suggestion.title}
                            </h6>
                            <p class="mb-2 text-muted">${suggestion.description}</p>
                            <div>
                                <span class="badge bg-${priorityClass}">
                                    ${suggestion.priority.toUpperCase()} Priority
                                </span>
                                ${suggestion.estimated_cost ? `
                                    <span class="badge bg-info ms-1">
                                        Est. Cost: $${suggestion.estimated_cost.toFixed(2)}
                                    </span>
                                ` : ''}
                                ${suggestion.mileage_threshold ? `
                                    <span class="badge bg-secondary ms-1">
                                        At ${suggestion.mileage_threshold.toLocaleString()} miles
                                    </span>
                                ` : ''}
                            </div>
                        </div>
                        <button class="btn btn-sm btn-outline-primary">
                            Use This
                        </button>
                    </div>
                </div>
            `;
        });
    }
    
    html += '</div>';
    return html;
}

// Use a maintenance suggestion
function useSuggestion(index) {
    if (maintenanceSuggestions && maintenanceSuggestions[index]) {
        const suggestion = maintenanceSuggestions[index];
        
        // Set the maintenance type
        window.wizard.data.category = suggestion.type;
        window.wizard.data.suggestedDescription = suggestion.title;
        window.wizard.data.suggestedCost = suggestion.estimated_cost;
        
        // Move to next step
        window.wizard.nextStep();
        
        // Pre-fill the form if on the details step
        setTimeout(() => {
            const descriptionField = document.getElementById('description');
            if (descriptionField) {
                descriptionField.value = suggestion.title;
            }
            
            const costField = document.getElementById('cost');
            if (costField && suggestion.estimated_cost) {
                costField.value = suggestion.estimated_cost.toFixed(2);
            }
        }, 100);
    }
}

// Select maintenance type from grid
function selectMaintenanceType(type) {
    window.wizard.data.category = type;
    window.wizard.nextStep();
}

// Initialize autocomplete for description field
function initMaintenanceAutocomplete(fieldId, category) {
    const field = document.getElementById(fieldId);
    if (!field) return;
    
    let autocompleteContainer = document.getElementById(fieldId + '_autocomplete');
    if (!autocompleteContainer) {
        autocompleteContainer = document.createElement('div');
        autocompleteContainer.id = fieldId + '_autocomplete';
        autocompleteContainer.className = 'autocomplete-suggestions';
        autocompleteContainer.style.cssText = `
            position: absolute;
            background: white;
            border: 1px solid #ddd;
            border-radius: 4px;
            max-height: 200px;
            overflow-y: auto;
            display: none;
            z-index: 1000;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        `;
        field.parentNode.appendChild(autocompleteContainer);
    }
    
    let debounceTimer;
    
    field.addEventListener('input', function() {
        clearTimeout(debounceTimer);
        const value = this.value;
        
        if (value.length < 2) {
            autocompleteContainer.style.display = 'none';
            return;
        }
        
        debounceTimer = setTimeout(async () => {
            try {
                const response = await fetch(`/api/maintenance/autocomplete?q=${encodeURIComponent(value)}&category=${category}`);
                if (response.ok) {
                    const suggestions = await response.json();
                    displayAutocomplete(suggestions, autocompleteContainer, field);
                }
            } catch (error) {
                console.error('Autocomplete error:', error);
            }
        }, 300);
    });
    
    // Hide autocomplete on click outside
    document.addEventListener('click', function(e) {
        if (!field.contains(e.target) && !autocompleteContainer.contains(e.target)) {
            autocompleteContainer.style.display = 'none';
        }
    });
}

// Display autocomplete suggestions
function displayAutocomplete(suggestions, container, field) {
    if (suggestions.length === 0) {
        container.style.display = 'none';
        return;
    }
    
    let html = '';
    suggestions.forEach(suggestion => {
        html += `
            <div class="autocomplete-item p-2" style="cursor: pointer; border-bottom: 1px solid #eee;">
                <div>${suggestion.label}</div>
                ${suggestion.average_cost ? `
                    <small class="text-muted">Avg. Cost: $${suggestion.average_cost.toFixed(2)}</small>
                ` : ''}
            </div>
        `;
    });
    
    container.innerHTML = html;
    container.style.display = 'block';
    
    // Position below the field
    const rect = field.getBoundingClientRect();
    container.style.width = rect.width + 'px';
    container.style.top = (rect.bottom + window.scrollY) + 'px';
    container.style.left = rect.left + 'px';
    
    // Add click handlers
    container.querySelectorAll('.autocomplete-item').forEach((item, index) => {
        item.addEventListener('click', function() {
            field.value = suggestions[index].value;
            
            // Update cost field if available
            const costField = document.getElementById('cost');
            if (costField && suggestions[index].average_cost) {
                costField.value = suggestions[index].average_cost.toFixed(2);
            }
            
            container.style.display = 'none';
        });
        
        item.addEventListener('mouseenter', function() {
            this.style.backgroundColor = '#f0f0f0';
        });
        
        item.addEventListener('mouseleave', function() {
            this.style.backgroundColor = 'white';
        });
    });
}