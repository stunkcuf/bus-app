// Enhanced JavaScript for assign routes page
document.addEventListener('DOMContentLoaded', function() {
    console.log('Initializing enhanced assign routes functionality...');
    
    // Initialize edit route modal functionality
    function initializeEditModal() {
        // Find all edit buttons
        const editButtons = document.querySelectorAll('[data-route-id]');
        console.log(`Found ${editButtons.length} route edit buttons`);
        
        editButtons.forEach(button => {
            // Check if this is an edit button
            if (button.classList.contains('js-edit-route') || button.tagName === 'BUTTON') {
                button.addEventListener('click', function(e) {
                    e.preventDefault();
                    e.stopPropagation();
                    
                    const routeId = this.dataset.routeId;
                    const routeName = this.dataset.routeName || '';
                    const description = this.dataset.description || '';
                    
                    console.log('Opening edit modal for route:', routeId, routeName);
                    
                    // Find or create modal
                    let modal = document.getElementById('editRouteModal');
                    if (!modal) {
                        console.log('Creating modal element...');
                        modal = createEditModal();
                        document.body.appendChild(modal);
                    }
                    
                    // Set form values
                    const routeIdField = modal.querySelector('#edit_route_id');
                    const routeNameField = modal.querySelector('#edit_route_name');
                    const descriptionField = modal.querySelector('#edit_description');
                    
                    if (routeIdField) routeIdField.value = routeId;
                    if (routeNameField) routeNameField.value = routeName;
                    if (descriptionField) descriptionField.value = description;
                    
                    // Show modal
                    if (typeof bootstrap !== 'undefined') {
                        const bsModal = new bootstrap.Modal(modal);
                        bsModal.show();
                    } else {
                        // Fallback if Bootstrap isn't loaded
                        modal.style.display = 'block';
                        modal.classList.add('show');
                    }
                });
            }
        });
    }
    
    // Create edit modal if it doesn't exist
    function createEditModal() {
        const modalHTML = `
            <div class="modal fade" id="editRouteModal" tabindex="-1">
                <div class="modal-dialog">
                    <div class="modal-content">
                        <div class="modal-header">
                            <h5 class="modal-title">Edit Route</h5>
                            <button type="button" class="btn-close" data-bs-dismiss="modal"></button>
                        </div>
                        <form method="POST" action="/update-route">
                            <div class="modal-body">
                                <input type="hidden" id="edit_route_id" name="route_id">
                                <div class="mb-3">
                                    <label for="edit_route_name" class="form-label">Route Name</label>
                                    <input type="text" class="form-control" id="edit_route_name" name="route_name" required>
                                </div>
                                <div class="mb-3">
                                    <label for="edit_description" class="form-label">Description</label>
                                    <textarea class="form-control" id="edit_description" name="description" rows="3"></textarea>
                                </div>
                            </div>
                            <div class="modal-footer">
                                <button type="button" class="btn btn-secondary" data-bs-dismiss="modal">Cancel</button>
                                <button type="submit" class="btn btn-primary">Save Changes</button>
                            </div>
                        </form>
                    </div>
                </div>
            </div>
        `;
        
        const div = document.createElement('div');
        div.innerHTML = modalHTML;
        return div.firstElementChild;
    }
    
    // Enhanced multi-route assignment
    function enhanceMultiRouteForm() {
        const form = document.querySelector('form[action="/multi-route-assign"]');
        if (!form) {
            console.log('Multi-route form not found, checking for regular form...');
            return;
        }
        
        console.log('Enhancing multi-route form...');
        
        // Add validation
        form.addEventListener('submit', function(e) {
            const checkboxes = form.querySelectorAll('input[name="route_ids[]"]:checked');
            if (checkboxes.length === 0) {
                e.preventDefault();
                alert('Please select at least one route to assign.');
                return false;
            }
            
            const driver = form.querySelector('select[name="driver"]').value;
            const busId = form.querySelector('select[name="bus_id"]').value;
            
            if (!driver || !busId) {
                e.preventDefault();
                alert('Please select both a driver and a bus.');
                return false;
            }
            
            console.log(`Assigning ${checkboxes.length} routes to driver ${driver} with bus ${busId}`);
        });
        
        // Add select all/none functionality
        const routeCheckboxContainer = form.querySelector('.route-checkboxes');
        if (routeCheckboxContainer) {
            const selectAllBtn = document.createElement('button');
            selectAllBtn.type = 'button';
            selectAllBtn.className = 'btn btn-sm btn-outline-primary me-2 mb-2';
            selectAllBtn.textContent = 'Select All';
            selectAllBtn.onclick = () => {
                form.querySelectorAll('input[name="route_ids[]"]').forEach(cb => cb.checked = true);
            };
            
            const selectNoneBtn = document.createElement('button');
            selectNoneBtn.type = 'button';
            selectNoneBtn.className = 'btn btn-sm btn-outline-secondary mb-2';
            selectNoneBtn.textContent = 'Clear Selection';
            selectNoneBtn.onclick = () => {
                form.querySelectorAll('input[name="route_ids[]"]').forEach(cb => cb.checked = false);
            };
            
            const buttonDiv = document.createElement('div');
            buttonDiv.className = 'mb-2';
            buttonDiv.appendChild(selectAllBtn);
            buttonDiv.appendChild(selectNoneBtn);
            
            routeCheckboxContainer.parentNode.insertBefore(buttonDiv, routeCheckboxContainer);
        }
    }
    
    // Initialize driver-bus relationship
    function initializeDriverBusSync() {
        const driverSelect = document.querySelector('select[name="driver"]');
        const busSelect = document.querySelector('select[name="bus_id"]');
        
        if (!driverSelect || !busSelect) return;
        
        driverSelect.addEventListener('change', async function() {
            const driver = this.value;
            if (!driver) return;
            
            // Check if driver already has assignments
            try {
                const response = await fetch(`/api/driver-assignments?driver=${driver}`);
                if (response.ok) {
                    const assignments = await response.json();
                    if (assignments && assignments.length > 0) {
                        // Pre-select the bus if driver already has one
                        const existingBus = assignments[0].bus_id;
                        if (existingBus && busSelect.querySelector(`option[value="${existingBus}"]`)) {
                            busSelect.value = existingBus;
                            
                            // Show which routes are already assigned
                            const assignedRoutes = assignments.map(a => a.route_id);
                            const routeInfo = document.createElement('div');
                            routeInfo.className = 'alert alert-info mt-2';
                            routeInfo.innerHTML = `
                                <small>
                                    <i class="bi bi-info-circle"></i>
                                    Driver ${driver} is currently assigned to bus ${existingBus} 
                                    with ${assignments.length} route(s): ${assignments.map(a => a.route_name || a.route_id).join(', ')}
                                </small>
                            `;
                            
                            // Remove any existing info
                            const existingInfo = busSelect.parentNode.querySelector('.alert-info');
                            if (existingInfo) existingInfo.remove();
                            
                            busSelect.parentNode.appendChild(routeInfo);
                        }
                    }
                }
            } catch (error) {
                console.error('Error fetching driver assignments:', error);
            }
        });
    }
    
    // Run all initializations
    initializeEditModal();
    enhanceMultiRouteForm();
    initializeDriverBusSync();
    
    console.log('Enhanced assign routes initialization complete');
});