// Fix for assign routes page
document.addEventListener('DOMContentLoaded', function() {
    console.log('Fixing assign routes page...');
    
    // Fix 1: Allow selecting the same bus for multiple routes for the same driver
    const driverSelect = document.querySelector('select[name="driver"]');
    const busSelect = document.querySelector('select[name="bus_id"]');
    const routeCheckboxes = document.querySelectorAll('input[name="route_ids[]"]');
    
    if (driverSelect && busSelect) {
        // Store original bus options
        const originalBusOptions = Array.from(busSelect.options).map(opt => ({
            value: opt.value,
            text: opt.text,
            dataset: {...opt.dataset}
        }));
        
        driverSelect.addEventListener('change', function() {
            const selectedDriver = this.value;
            
            // Reset bus options
            busSelect.innerHTML = '';
            
            // Add default option
            const defaultOpt = document.createElement('option');
            defaultOpt.value = '';
            defaultOpt.textContent = 'Select a bus...';
            busSelect.appendChild(defaultOpt);
            
            // Add all bus options
            originalBusOptions.forEach(bus => {
                if (bus.value) {
                    const opt = document.createElement('option');
                    opt.value = bus.value;
                    opt.textContent = bus.text;
                    Object.assign(opt.dataset, bus.dataset);
                    busSelect.appendChild(opt);
                }
            });
            
            // If driver already has assignments, pre-select their bus
            const existingAssignments = document.querySelectorAll(`[data-driver="${selectedDriver}"]`);
            if (existingAssignments.length > 0) {
                const existingBus = existingAssignments[0].dataset.busId;
                if (existingBus && busSelect.querySelector(`option[value="${existingBus}"]`)) {
                    busSelect.value = existingBus;
                }
            }
        });
    }
    
    // Fix 2: Fix the edit route modal
    document.querySelectorAll('.js-edit-route').forEach(button => {
        // Remove any existing listeners
        const newButton = button.cloneNode(true);
        button.parentNode.replaceChild(newButton, button);
        
        newButton.addEventListener('click', function(e) {
            e.preventDefault();
            e.stopPropagation();
            
            console.log('Edit button clicked', this.dataset);
            
            const routeId = this.dataset.routeId || '';
            const routeName = this.dataset.routeName || '';
            const description = this.dataset.description || '';
            
            // Wait for next tick to ensure modal is ready
            setTimeout(() => {
                const modalEl = document.getElementById('editRouteModal');
                if (!modalEl) {
                    console.error('Modal element not found');
                    return;
                }
                
                // Set form values
                const routeIdField = document.getElementById('edit_route_id');
                const routeNameField = document.getElementById('edit_route_name');
                const descriptionField = document.getElementById('edit_description');
                
                if (routeIdField) routeIdField.value = routeId;
                if (routeNameField) routeNameField.value = routeName;
                if (descriptionField) descriptionField.value = description;
                
                // Show modal using Bootstrap 5
                try {
                    let modal = bootstrap.Modal.getInstance(modalEl);
                    if (!modal) {
                        modal = new bootstrap.Modal(modalEl, {
                            backdrop: true,
                            keyboard: true,
                            focus: true
                        });
                    }
                    modal.show();
                } catch (err) {
                    console.error('Error showing modal:', err);
                }
            }, 100);
        });
    });
    
    // Fix 3: Enable multiple route selection
    if (routeCheckboxes.length > 0) {
        // Convert route selection to checkboxes if not already
        const routeSelectContainer = document.querySelector('.route-select-container');
        if (routeSelectContainer && !routeSelectContainer.querySelector('input[type="checkbox"]')) {
            const routes = document.querySelectorAll('[data-route-id]');
            
            routeSelectContainer.innerHTML = '<div class="route-checkboxes">';
            routes.forEach(route => {
                const routeId = route.dataset.routeId;
                const routeName = route.dataset.routeName;
                
                if (routeId && routeName) {
                    const checkboxDiv = document.createElement('div');
                    checkboxDiv.className = 'form-check mb-2';
                    checkboxDiv.innerHTML = `
                        <input class="form-check-input" type="checkbox" 
                               name="route_ids[]" value="${routeId}" 
                               id="route_${routeId}">
                        <label class="form-check-label" for="route_${routeId}">
                            ${routeName}
                        </label>
                    `;
                    routeSelectContainer.appendChild(checkboxDiv);
                }
            });
            routeSelectContainer.innerHTML += '</div>';
        }
    }
    
    // Fix 4: Ensure modal can be closed
    const modalCloseButtons = document.querySelectorAll('[data-bs-dismiss="modal"]');
    modalCloseButtons.forEach(btn => {
        btn.addEventListener('click', function() {
            const modals = document.querySelectorAll('.modal.show');
            modals.forEach(modal => {
                const bsModal = bootstrap.Modal.getInstance(modal);
                if (bsModal) {
                    bsModal.hide();
                }
            });
        });
    });
    
    // Fix 5: Handle ESC key to close modals
    document.addEventListener('keydown', function(e) {
        if (e.key === 'Escape') {
            const modals = document.querySelectorAll('.modal.show');
            modals.forEach(modal => {
                const bsModal = bootstrap.Modal.getInstance(modal);
                if (bsModal) {
                    bsModal.hide();
                }
            });
        }
    });
    
    console.log('Assign routes page fixes applied');
});