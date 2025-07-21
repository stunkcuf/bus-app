// Route Assignment Wizard
function initRouteAssignmentWizard() {
    const wizard = new StepWizard({
        containerId: 'wizardContainer',
        steps: [
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
                init: function(wizard) {
                    // Load available drivers
                    fetch('/api/available-drivers')
                        .then(response => response.json())
                        .then(drivers => {
                            const select = document.getElementById('driver');
                            select.innerHTML = '<option value="">Select a driver...</option>';
                            drivers.forEach(driver => {
                                const option = document.createElement('option');
                                option.value = driver.username;
                                option.textContent = `${driver.username} - ${driver.status}`;
                                select.appendChild(option);
                            });
                        })
                        .catch(error => {
                            wizard.showError('Failed to load drivers');
                        });
                },
                help: 'Only active drivers who are not currently assigned to a route will appear in this list.'
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