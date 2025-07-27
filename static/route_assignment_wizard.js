// Route Assignment Wizard with Conflict Detection
function initRouteAssignmentWizard() {
    const wizard = new StepWizard({
        containerId: 'wizardContainer',
        steps: [
            {
                title: 'Select Route',
                description: 'Choose the route to assign.',
                fields: [
                    {
                        id: 'routeId',
                        type: 'select',
                        label: 'Route',
                        required: true,
                        options: [] // Will be populated dynamically
                    },
                    {
                        id: 'period',
                        type: 'radio',
                        label: 'Period',
                        required: true,
                        options: [
                            { value: 'morning', text: '🌅 Morning Route' },
                            { value: 'afternoon', text: '🌆 Afternoon Route' }
                        ]
                    }
                ],
                init: function(wizard) {
                    // Load available routes
                    fetch('/api/routes/unassigned')
                        .then(response => response.json())
                        .then(routes => {
                            const select = document.getElementById('routeId');
                            select.innerHTML = '<option value="">Select a route...</option>';
                            routes.forEach(route => {
                                const option = document.createElement('option');
                                option.value = route.id;
                                option.textContent = `${route.name} (${route.student_count} students)`;
                                select.appendChild(option);
                            });
                        })
                        .catch(error => {
                            wizard.showError('Failed to load routes');
                        });
                },
                help: 'Select the route and time period for this assignment.'
            },
            {
                title: 'Select Driver',
                description: 'Choose the driver who will be assigned to this route.',
                fields: [
                    {
                        id: 'driver',
                        type: 'select',
                        label: 'Driver',
                        required: true,
                        options: [] // Will be populated dynamically
                    }
                ],
                init: async function(wizard) {
                    const routeId = wizard.data.routeId;
                    const period = wizard.data.period;
                    
                    // Show loading indicator
                    const container = document.getElementById('driverSuggestions');
                    if (container) {
                        container.innerHTML = '<div class="text-center"><div class="spinner-border"></div><p>Loading suggestions...</p></div>';
                    }
                    
                    // Load available drivers
                    try {
                        const [driversResponse, suggestionsResponse] = await Promise.all([
                            fetch('/api/available-drivers'),
                            fetch(`/api/route-assignment/suggestions?route_id=${routeId}&period=${period}`)
                        ]);
                        
                        const drivers = await driversResponse.json();
                        const suggestions = await suggestionsResponse.json();
                        
                        const select = document.getElementById('driver');
                        select.innerHTML = '<option value="">Select a driver...</option>';
                        
                        // Add suggested drivers first
                        if (suggestions && suggestions.length > 0) {
                            const optgroup = document.createElement('optgroup');
                            optgroup.label = '⭐ Recommended Drivers';
                            
                            suggestions.forEach(suggestion => {
                                if (!wizard.data.busId || suggestion.bus_id === wizard.data.busId) {
                                    const option = document.createElement('option');
                                    option.value = suggestion.driver_id;
                                    option.textContent = `${suggestion.driver_name} (Score: ${(suggestion.score * 100).toFixed(0)}%)`;
                                    option.dataset.suggestion = JSON.stringify(suggestion);
                                    optgroup.appendChild(option);
                                }
                            });
                            
                            if (optgroup.children.length > 0) {
                                select.appendChild(optgroup);
                            }
                        }
                        
                        // Add all available drivers
                        const allGroup = document.createElement('optgroup');
                        allGroup.label = 'All Available Drivers';
                        
                        drivers.forEach(driver => {
                            const option = document.createElement('option');
                            option.value = driver.username;
                            option.textContent = `${driver.name || driver.username} - ${driver.status || 'Available'}`;
                            allGroup.appendChild(option);
                        });
                        
                        select.appendChild(allGroup);
                        
                        // Show suggestion details
                        if (container && suggestions && suggestions.length > 0) {
                            container.innerHTML = renderSuggestions(suggestions);
                        }
                        
                    } catch (error) {
                        wizard.showError('Failed to load drivers');
                    }
                },
                render: function(data) {
                    return `
                        <div class="form-group">
                            <label for="driver">Select Driver <span class="text-danger">*</span></label>
                            <select class="form-control" id="driver" required>
                                <option value="">Loading drivers...</option>
                            </select>
                        </div>
                        <div id="driverSuggestions" class="mt-3"></div>
                        <div id="conflictWarnings" class="mt-3"></div>
                    `;
                },
                onFieldChange: async function(fieldId, value, wizard) {
                    if (fieldId === 'driver' && value) {
                        await checkConflicts(wizard);
                    }
                },
                help: 'Drivers are ranked by experience, route familiarity, and availability.'
            },
            {
                title: 'Select Bus',
                description: 'Choose the bus that will be used for this route.',
                fields: [
                    {
                        id: 'busId',
                        type: 'select',
                        label: 'Bus',
                        required: true,
                        options: [] // Will be populated dynamically
                    }
                ],
                init: function(wizard) {
                    // Load available buses
                    fetch('/api/available-buses')
                        .then(response => response.json())
                        .then(buses => {
                            const select = document.getElementById('busId');
                            select.innerHTML = '<option value="">Select a bus...</option>';
                            buses.forEach(bus => {
                                const option = document.createElement('option');
                                option.value = bus.bus_id;
                                option.textContent = `${bus.bus_id} - ${bus.model} (Capacity: ${bus.capacity})`;
                                if (bus.status !== 'active') {
                                    option.disabled = true;
                                    option.textContent += ' - ' + bus.status.toUpperCase();
                                }
                                select.appendChild(option);
                            });
                        })
                        .catch(error => {
                            wizard.showError('Failed to load buses');
                        });
                },
                help: 'Only buses with "Active" status can be assigned to routes.'
            },
            {
                title: 'Select Route',
                description: 'Choose the route to assign.',
                fields: [
                    {
                        id: 'routeId',
                        type: 'select',
                        label: 'Route',
                        required: true,
                        options: [] // Will be populated dynamically
                    }
                ],
                init: function(wizard) {
                    // Load available routes
                    fetch('/api/available-routes')
                        .then(response => response.json())
                        .then(routes => {
                            const select = document.getElementById('routeId');
                            select.innerHTML = '<option value="">Select a route...</option>';
                            routes.forEach(route => {
                                const option = document.createElement('option');
                                option.value = route.route_id;
                                option.textContent = `${route.route_name} - ${route.description}`;
                                select.appendChild(option);
                            });
                        })
                        .catch(error => {
                            wizard.showError('Failed to load routes');
                        });
                },
                help: 'Routes that are already assigned will not appear in this list.'
            },
            {
                title: 'Review Assignment',
                description: 'Please review the assignment details before confirming.',
                render: function(data) {
                    return `
                        <div class="wizard-summary">
                            <h4>Assignment Summary</h4>
                            <table>
                                <tr>
                                    <td>Driver:</td>
                                    <td>${data.driver || 'Not selected'}</td>
                                </tr>
                                <tr>
                                    <td>Bus:</td>
                                    <td>${data.busId || 'Not selected'}</td>
                                </tr>
                                <tr>
                                    <td>Route:</td>
                                    <td>${data.routeId || 'Not selected'}</td>
                                </tr>
                                <tr>
                                    <td>Assignment Date:</td>
                                    <td>${new Date().toLocaleDateString()}</td>
                                </tr>
                            </table>
                        </div>
                        <div id="conflictCheck" class="mt-3"></div>
                    `;
                },
                init: function(wizard) {
                    // Check for conflicts
                    const conflictDiv = document.getElementById('conflictCheck');
                    conflictDiv.innerHTML = '<div class="wizard-loading"><div class="spinner-border"></div><p>Checking for conflicts...</p></div>';
                    
                    fetch('/api/check-assignment-conflicts', {
                        method: 'POST',
                        headers: {
                            'Content-Type': 'application/json',
                        },
                        body: JSON.stringify(wizard.data)
                    })
                    .then(response => response.json())
                    .then(result => {
                        if (result.conflicts && result.conflicts.length > 0) {
                            let html = '<div class="conflict-warning"><h4>⚠️ Potential Conflicts Detected</h4><ul>';
                            result.conflicts.forEach(conflict => {
                                html += `<li>${conflict}</li>`;
                            });
                            html += '</ul></div>';
                            conflictDiv.innerHTML = html;
                        } else {
                            conflictDiv.innerHTML = '<div class="alert alert-success"><i class="bi bi-check-circle"></i> No conflicts detected. Assignment is ready to proceed.</div>';
                        }
                    })
                    .catch(error => {
                        conflictDiv.innerHTML = '<div class="alert alert-warning">Unable to check for conflicts. Please verify manually.</div>';
                    });
                },
                validate: function(data) {
                    if (!data.driver) return 'Please select a driver';
                    if (!data.busId) return 'Please select a bus';
                    if (!data.routeId) return 'Please select a route';
                    return true;
                }
            }
        ],
        onComplete: function(data) {
            // Submit the assignment
            const wizardContainer = document.getElementById('wizardContainer');
            wizardContainer.innerHTML = '<div class="wizard-loading"><div class="spinner-border"></div><p>Creating assignment...</p></div>';
            
            // Get CSRF token
            const csrfToken = document.querySelector('input[name="csrf_token"]').value;
            
            fetch('/assign-route', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/x-www-form-urlencoded',
                },
                body: `driver=${encodeURIComponent(data.driver)}&bus_id=${encodeURIComponent(data.busId)}&route_id=${encodeURIComponent(data.routeId)}&csrf_token=${encodeURIComponent(csrfToken)}`
            })
            .then(response => {
                if (response.ok) {
                    wizardContainer.innerHTML = `
                        <div class="wizard-success">
                            <i class="bi bi-check-circle"></i>
                            <h3>Assignment Created Successfully!</h3>
                            <p>The route has been assigned to ${data.driver}.</p>
                            <button class="btn btn-primary mt-3" onclick="location.reload()">
                                <i class="bi bi-arrow-clockwise"></i> View Assignments
                            </button>
                        </div>
                    `;
                } else {
                    throw new Error('Assignment failed');
                }
            })
            .catch(error => {
                wizardContainer.innerHTML = `
                    <div class="alert alert-danger">
                        <i class="bi bi-exclamation-triangle"></i> Failed to create assignment. Please try again.
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

// Check for conflicts when driver and bus are selected
async function checkConflicts(wizard) {
    const container = document.getElementById('conflictWarnings');
    if (!container) return;
    
    const data = {
        driver_id: wizard.data.driver,
        bus_id: wizard.data.busId,
        route_id: wizard.data.routeId,
        period: wizard.data.period
    };
    
    try {
        const response = await fetch('/api/route-assignment/check-conflicts', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'X-CSRF-Token': getCSRFToken()
            },
            body: JSON.stringify(data)
        });
        
        const result = await response.json();
        
        container.innerHTML = renderConflictResults(result);
        
        // Update wizard based on conflicts
        if (!result.can_assign) {
            wizard.disableNext();
            wizard.showError('Cannot proceed due to conflicts. Please select different options.');
        } else {
            wizard.enableNext();
        }
        
    } catch (error) {
        console.error('Error checking conflicts:', error);
    }
}

// Render conflict checking results
function renderConflictResults(result) {
    let html = '';
    
    if (result.conflicts && result.conflicts.length > 0) {
        html += '<div class="alert alert-danger">';
        html += '<h6><i class="bi bi-exclamation-triangle"></i> Conflicts Detected</h6>';
        html += '<ul class="mb-0">';
        result.conflicts.forEach(conflict => {
            html += `<li>${conflict.description}</li>`;
        });
        html += '</ul>';
        html += '</div>';
    }
    
    if (result.warnings && result.warnings.length > 0) {
        html += '<div class="alert alert-warning">';
        html += '<h6><i class="bi bi-info-circle"></i> Warnings</h6>';
        html += '<ul class="mb-0">';
        result.warnings.forEach(warning => {
            html += `<li>${warning.description}</li>`;
        });
        html += '</ul>';
        html += '</div>';
    }
    
    if (result.can_assign && (!result.conflicts || result.conflicts.length === 0)) {
        html += '<div class="alert alert-success">';
        html += '<i class="bi bi-check-circle"></i> No conflicts detected. Assignment can proceed.';
        html += '</div>';
    }
    
    return html;
}

// Render driver/bus suggestions
function renderSuggestions(suggestions) {
    let html = '<div class="suggestions-container">';
    html += '<h6><i class="bi bi-lightbulb"></i> Recommended Assignments</h6>';
    
    suggestions.forEach((suggestion, index) => {
        const scoreClass = suggestion.score > 0.8 ? 'success' : suggestion.score > 0.6 ? 'warning' : 'secondary';
        
        html += `
            <div class="suggestion-card mb-2 p-3 border rounded">
                <div class="d-flex justify-content-between align-items-start">
                    <div>
                        <strong>${suggestion.driver_name}</strong> with <strong>Bus ${suggestion.bus_number}</strong>
                        <div class="mt-1">
                            <span class="badge bg-${scoreClass}">Match Score: ${(suggestion.score * 100).toFixed(0)}%</span>
                            ${suggestion.has_conflicts ? '<span class="badge bg-danger ms-1">Has Conflicts</span>' : ''}
                        </div>
                    </div>
                    <button class="btn btn-sm btn-outline-primary" onclick="applySuggestion(${index})">
                        Use This
                    </button>
                </div>
                ${suggestion.reasons && suggestion.reasons.length > 0 ? `
                    <div class="mt-2 small text-muted">
                        <strong>Reasons:</strong>
                        <ul class="mb-0">
                            ${suggestion.reasons.map(reason => `<li>${reason}</li>`).join('')}
                        </ul>
                    </div>
                ` : ''}
            </div>
        `;
    });
    
    html += '</div>';
    return html;
}

// Apply a suggestion
function applySuggestion(index) {
    // This function would be called when a suggestion is clicked
    // Implementation depends on how suggestions are stored
    console.log('Applying suggestion', index);
}

// Helper to get CSRF token
function getCSRFToken() {
    const tokenInput = document.querySelector('input[name="csrf_token"]');
    return tokenInput ? tokenInput.value : '';
}
}