// Maintenance Logging Wizard
function initMaintenanceWizard(vehicleType, vehicleId) {
    const wizard = new StepWizard({
        containerId: 'wizardContainer',
        steps: [
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
                        id: 'notes',
                        type: 'textarea',
                        label: 'Service Notes',
                        required: true,
                        placeholder: 'Describe the maintenance performed...'
                    }
                ],
                init: function(wizard) {
                    // Set today's date as default
                    const dateInput = document.getElementById('date');
                    dateInput.value = new Date().toISOString().split('T')[0];
                    
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