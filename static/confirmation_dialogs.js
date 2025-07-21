// Confirmation Dialog System for Error Prevention
class ConfirmationDialog {
    constructor() {
        this.init();
    }

    init() {
        // Create dialog container
        this.createDialogContainer();
        
        // Bind to all delete buttons and dangerous actions
        document.addEventListener('DOMContentLoaded', () => {
            this.attachToElements();
        });
    }

    createDialogContainer() {
        const container = document.createElement('div');
        container.id = 'confirmation-dialog';
        container.className = 'confirmation-dialog-overlay';
        container.innerHTML = `
            <div class="confirmation-dialog">
                <div class="dialog-header">
                    <i class="bi bi-exclamation-triangle dialog-icon"></i>
                    <h3 id="dialog-title">Confirm Action</h3>
                </div>
                <div class="dialog-body">
                    <p id="dialog-message">Are you sure you want to proceed?</p>
                    <div id="dialog-details" class="dialog-details"></div>
                </div>
                <div class="dialog-footer">
                    <button type="button" class="btn btn-secondary" id="dialog-cancel">
                        <i class="bi bi-x-circle me-2"></i>Cancel
                    </button>
                    <button type="button" class="btn btn-danger" id="dialog-confirm">
                        <i class="bi bi-check-circle me-2"></i>Confirm
                    </button>
                </div>
            </div>
        `;
        container.style.display = 'none';
        document.body.appendChild(container);
        
        // Bind dialog buttons
        document.getElementById('dialog-cancel').addEventListener('click', () => this.hide());
        document.getElementById('dialog-confirm').addEventListener('click', () => this.confirm());
        
        // Close on escape key
        document.addEventListener('keydown', (e) => {
            if (e.key === 'Escape' && this.isVisible()) {
                this.hide();
            }
        });
    }

    attachToElements() {
        // Attach to delete buttons
        const deleteButtons = document.querySelectorAll('[data-confirm-delete]');
        deleteButtons.forEach(button => {
            button.addEventListener('click', (e) => {
                e.preventDefault();
                const itemName = button.getAttribute('data-confirm-delete');
                const action = button.getAttribute('data-action') || 'delete';
                const details = button.getAttribute('data-details') || '';
                
                this.show({
                    title: 'Confirm Deletion',
                    message: `Are you sure you want to delete "${itemName}"?`,
                    details: details || 'This action cannot be undone.',
                    confirmText: 'Delete',
                    confirmClass: 'btn-danger',
                    onConfirm: () => {
                        // Submit the form or perform the action
                        const form = button.closest('form');
                        if (form) {
                            form.submit();
                        } else if (button.href) {
                            window.location.href = button.href;
                        }
                    }
                });
            });
        });
        
        // Attach to forms with confirmation
        const confirmForms = document.querySelectorAll('form[data-confirm]');
        confirmForms.forEach(form => {
            form.addEventListener('submit', (e) => {
                e.preventDefault();
                
                const message = form.getAttribute('data-confirm');
                const details = form.getAttribute('data-confirm-details') || '';
                
                this.show({
                    title: 'Confirm Action',
                    message: message,
                    details: details,
                    confirmText: 'Proceed',
                    confirmClass: 'btn-primary',
                    onConfirm: () => {
                        form.submit();
                    }
                });
            });
        });
        
        // Attach to dangerous buttons
        const dangerButtons = document.querySelectorAll('[data-confirm-action]');
        dangerButtons.forEach(button => {
            button.addEventListener('click', (e) => {
                e.preventDefault();
                
                const action = button.getAttribute('data-confirm-action');
                const message = button.getAttribute('data-confirm-message') || `Are you sure you want to ${action}?`;
                const details = button.getAttribute('data-confirm-details') || '';
                
                this.show({
                    title: `Confirm ${action}`,
                    message: message,
                    details: details,
                    confirmText: action,
                    confirmClass: button.classList.contains('btn-danger') ? 'btn-danger' : 'btn-warning',
                    onConfirm: () => {
                        // Trigger the original action
                        if (button.form) {
                            button.form.submit();
                        } else if (button.href) {
                            window.location.href = button.href;
                        } else {
                            // Trigger click event without confirmation
                            button.removeAttribute('data-confirm-action');
                            button.click();
                            button.setAttribute('data-confirm-action', action);
                        }
                    }
                });
            });
        });
    }

    show(options) {
        this.currentOptions = options;
        
        // Update dialog content
        document.getElementById('dialog-title').textContent = options.title || 'Confirm Action';
        document.getElementById('dialog-message').textContent = options.message || 'Are you sure?';
        
        const detailsEl = document.getElementById('dialog-details');
        if (options.details) {
            detailsEl.textContent = options.details;
            detailsEl.style.display = 'block';
        } else {
            detailsEl.style.display = 'none';
        }
        
        const confirmBtn = document.getElementById('dialog-confirm');
        confirmBtn.textContent = options.confirmText || 'Confirm';
        confirmBtn.className = `btn ${options.confirmClass || 'btn-danger'}`;
        
        // Add icon to confirm button
        const icon = document.createElement('i');
        icon.className = 'bi bi-check-circle me-2';
        confirmBtn.prepend(icon);
        
        // Show dialog
        const container = document.getElementById('confirmation-dialog');
        container.style.display = 'flex';
        
        // Focus cancel button for safety
        setTimeout(() => {
            document.getElementById('dialog-cancel').focus();
        }, 100);
        
        // Announce to screen readers
        this.announceDialog(options.message);
    }

    hide() {
        const container = document.getElementById('confirmation-dialog');
        container.style.display = 'none';
        this.currentOptions = null;
    }

    confirm() {
        if (this.currentOptions && this.currentOptions.onConfirm) {
            this.currentOptions.onConfirm();
        }
        this.hide();
    }

    isVisible() {
        const container = document.getElementById('confirmation-dialog');
        return container.style.display !== 'none';
    }

    announceDialog(message) {
        const announcement = document.createElement('div');
        announcement.setAttribute('role', 'alert');
        announcement.setAttribute('aria-live', 'assertive');
        announcement.className = 'sr-only';
        announcement.textContent = `Confirmation required: ${message}`;
        
        document.body.appendChild(announcement);
        setTimeout(() => announcement.remove(), 1000);
    }
}

// Initialize system
const confirmationDialog = new ConfirmationDialog();

// Export for use
window.ConfirmationDialog = ConfirmationDialog;