// Fix bus selection when driver already has a bus assigned
document.addEventListener('DOMContentLoaded', function() {
    const driverSelect = document.querySelector('select[name="driver"]');
    const busSelect = document.querySelector('select[name="bus_id"]');
    
    if (!driverSelect || !busSelect) return;
    
    // Store all bus options
    const allBusOptions = Array.from(busSelect.options).map(opt => ({
        value: opt.value,
        text: opt.text,
        html: opt.outerHTML
    }));
    
    driverSelect.addEventListener('change', async function() {
        const selectedDriver = this.value;
        
        if (!selectedDriver) {
            // Reset to all buses if no driver selected
            busSelect.innerHTML = allBusOptions.map(opt => opt.html).join('');
            return;
        }
        
        try {
            // Check if driver already has assignments
            const response = await fetch(`/api/driver-assignments?driver=${selectedDriver}`);
            if (response.ok) {
                const assignments = await response.json();
                
                if (assignments && assignments.length > 0) {
                    // Driver has existing assignment - keep only their bus and auto-select it
                    const existingBusId = assignments[0].bus_id;
                    
                    // Clear bus select
                    busSelect.innerHTML = '';
                    
                    // Add the assigned bus (even if not in available list)
                    const option = document.createElement('option');
                    option.value = existingBusId;
                    option.text = `Bus #${existingBusId}`;
                    option.selected = true;
                    busSelect.appendChild(option);
                    
                    // Disable changing the bus since driver already has one assigned
                    busSelect.style.pointerEvents = 'none';
                    busSelect.style.opacity = '0.8';
                    
                    // Show info about existing routes with better visibility
                    let infoDiv = busSelect.parentNode.querySelector('.driver-info');
                    if (!infoDiv) {
                        infoDiv = document.createElement('div');
                        infoDiv.className = 'driver-info mt-2 p-2 rounded';
                        infoDiv.style.cssText = 'background: rgba(13, 202, 240, 0.2); border: 1px solid rgba(13, 202, 240, 0.5); color: white;';
                        busSelect.parentNode.appendChild(infoDiv);
                    }
                    
                    const routeNames = assignments.map(a => a.route_name || a.route_id).join(', ');
                    infoDiv.innerHTML = `
                        <div style="font-size: 14px;">
                            <i class="bi bi-info-circle"></i>
                            <strong>Current Assignment:</strong><br>
                            Driver: ${selectedDriver}<br>
                            Bus: ${existingBusId}<br>
                            Routes: ${routeNames}
                        </div>
                    `;
                } else {
                    // No existing assignment - show all available buses
                    busSelect.innerHTML = allBusOptions.map(opt => opt.html).join('');
                    
                    // Re-enable bus selection
                    busSelect.style.pointerEvents = 'auto';
                    busSelect.style.opacity = '1';
                    
                    // Remove info div if exists
                    const infoDiv = busSelect.parentNode.querySelector('.driver-info');
                    if (infoDiv) infoDiv.remove();
                }
            }
        } catch (error) {
            console.error('Error fetching driver assignments:', error);
            // On error, show all buses
            busSelect.innerHTML = allBusOptions.map(opt => opt.html).join('');
        }
    });
    
    // Trigger on page load if driver is already selected
    if (driverSelect.value) {
        driverSelect.dispatchEvent(new Event('change'));
    }
});