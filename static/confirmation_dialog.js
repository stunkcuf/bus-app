// Confirmation Dialog System
// Provides consistent confirmation dialogs for destructive actions

class ConfirmationDialog {
    constructor() {
        this.dialogId = 'confirmationDialog';
        this.createDialogElement();
        this.setupEventListeners();
    }

    createDialogElement() {
        // Check if dialog already exists
        if (document.getElementById(this.dialogId)) {
            return;
        }

        const dialogHTML = `
        <div id="${this.dialogId}" class="confirmation-overlay" style="display: none;">
            <div class="confirmation-dialog">
                <div class="confirmation-header">
                    <h5 class="confirmation-title">
                        <i class="bi bi-exclamation-triangle-fill text-warning me-2"></i>
                        <span id="confirmTitle">Confirm Action</span>
                    </h5>
                </div>
                <div class="confirmation-body">
                    <p id="confirmMessage" class="confirmation-message">Are you sure you want to proceed?</p>
                    <div id="confirmDetails" class="confirmation-details" style="display: none;">
                        <!-- Additional details will be inserted here -->
                    </div>
                </div>
                <div class="confirmation-footer">
                    <button type="button" class="btn btn-secondary" id="confirmCancel">
                        <i class="bi bi-x-circle me-1"></i> Cancel
                    </button>
                    <button type="button" class="btn btn-danger" id="confirmProceed">
                        <i class="bi bi-check-circle me-1"></i> <span id="confirmButtonText">Proceed</span>
                    </button>
                </div>
            </div>
        </div>
        `;

        document.body.insertAdjacentHTML('beforeend', dialogHTML);
    }

    setupEventListeners() {
        // Cancel button
        document.getElementById('confirmCancel').addEventListener('click', () => {
            this.hide();
            if (this.onCancel) {
                this.onCancel();
            }
        });

        // Proceed button
        document.getElementById('confirmProceed').addEventListener('click', () => {
            this.hide();
            if (this.onConfirm) {
                this.onConfirm();
            }
        });

        // Click outside to close
        document.getElementById(this.dialogId).addEventListener('click', (e) => {
            if (e.target.id === this.dialogId) {
                this.hide();
                if (this.onCancel) {
                    this.onCancel();
                }
            }
        });

        // ESC key to close
        document.addEventListener('keydown', (e) => {
            if (e.key === 'Escape' && this.isVisible()) {
                this.hide();
                if (this.onCancel) {
                    this.onCancel();
                }
            }
        });
    }

    show(options = {}) {
        const {
            title = 'Confirm Action',
            message = 'Are you sure you want to proceed?',
            details = null,
            confirmText = 'Proceed',
            confirmClass = 'btn-danger',
            onConfirm = null,
            onCancel = null
        } = options;

        // Set dialog content
        document.getElementById('confirmTitle').textContent = title;
        document.getElementById('confirmMessage').textContent = message;
        document.getElementById('confirmButtonText').textContent = confirmText;

        // Set button class
        const confirmButton = document.getElementById('confirmProceed');
        confirmButton.className = `btn ${confirmClass}`;

        // Show/hide details
        const detailsElement = document.getElementById('confirmDetails');
        if (details) {
            detailsElement.innerHTML = details;
            detailsElement.style.display = 'block';
        } else {
            detailsElement.style.display = 'none';
        }

        // Store callbacks
        this.onConfirm = onConfirm;
        this.onCancel = onCancel;

        // Show dialog
        const dialog = document.getElementById(this.dialogId);
        dialog.style.display = 'flex';
        dialog.offsetHeight; // Force reflow
        dialog.classList.add('show');

        // Focus on cancel button for safety
        document.getElementById('confirmCancel').focus();
    }

    hide() {
        const dialog = document.getElementById(this.dialogId);
        dialog.classList.remove('show');
        setTimeout(() => {
            dialog.style.display = 'none';
        }, 300);

        // Clear callbacks
        this.onConfirm = null;
        this.onCancel = null;
    }

    isVisible() {
        const dialog = document.getElementById(this.dialogId);
        return dialog && dialog.style.display !== 'none';
    }
}

// Initialize global instance
const confirmDialog = new ConfirmationDialog();

// Helper functions for common actions
const confirmationHelpers = {
    // Delete confirmation
    confirmDelete: function(itemName, onConfirm) {
        confirmDialog.show({
            title: 'Delete Confirmation',
            message: `Are you sure you want to delete "${itemName}"?`,
            details: '<div class="alert alert-warning mb-0"><i class="bi bi-exclamation-triangle me-2"></i>This action cannot be undone.</div>',
            confirmText: 'Delete',
            confirmClass: 'btn-danger',
            onConfirm: onConfirm
        });
    },

    // Update confirmation
    confirmUpdate: function(itemName, changes, onConfirm) {
        let changesList = '<ul class="mb-0">';
        for (const [key, value] of Object.entries(changes)) {
            changesList += `<li><strong>${key}:</strong> ${value}</li>`;
        }
        changesList += '</ul>';

        confirmDialog.show({
            title: 'Update Confirmation',
            message: `Are you sure you want to update "${itemName}"?`,
            details: `<div class="changes-preview">${changesList}</div>`,
            confirmText: 'Update',
            confirmClass: 'btn-primary',
            onConfirm: onConfirm
        });
    },

    // Navigation confirmation (for unsaved changes)
    confirmNavigation: function(onConfirm) {
        confirmDialog.show({
            title: 'Unsaved Changes',
            message: 'You have unsaved changes. Are you sure you want to leave?',
            details: '<div class="alert alert-info mb-0"><i class="bi bi-info-circle me-2"></i>Your changes will be lost if you continue.</div>',
            confirmText: 'Leave Page',
            confirmClass: 'btn-warning',
            onConfirm: onConfirm
        });
    },

    // Logout confirmation
    confirmLogout: function(onConfirm) {
        confirmDialog.show({
            title: 'Logout Confirmation',
            message: 'Are you sure you want to logout?',
            confirmText: 'Logout',
            confirmClass: 'btn-primary',
            onConfirm: onConfirm
        });
    },

    // Batch action confirmation
    confirmBatchAction: function(action, itemCount, onConfirm) {
        confirmDialog.show({
            title: `${action} Multiple Items`,
            message: `Are you sure you want to ${action.toLowerCase()} ${itemCount} items?`,
            details: `<div class="alert alert-warning mb-0"><i class="bi bi-exclamation-triangle me-2"></i>This will affect ${itemCount} items.</div>`,
            confirmText: action,
            confirmClass: 'btn-danger',
            onConfirm: onConfirm
        });
    }
};

// Auto-attach to forms with confirmation requirement
document.addEventListener('DOMContentLoaded', function() {
    // Delete forms
    document.querySelectorAll('form[data-confirm-delete]').forEach(form => {
        form.addEventListener('submit', function(e) {
            e.preventDefault();
            const itemName = this.dataset.confirmDelete || 'this item';
            confirmationHelpers.confirmDelete(itemName, () => {
                this.submit();
            });
        });
    });

    // Update forms with changes
    document.querySelectorAll('form[data-confirm-update]').forEach(form => {
        form.addEventListener('submit', function(e) {
            if (this.dataset.hasChanges === 'true') {
                e.preventDefault();
                const itemName = this.dataset.confirmUpdate || 'this item';
                confirmationHelpers.confirmUpdate(itemName, {}, () => {
                    this.submit();
                });
            }
        });
    });

    // Logout buttons
    document.querySelectorAll('form[action="/logout"]').forEach(form => {
        form.addEventListener('submit', function(e) {
            e.preventDefault();
            confirmationHelpers.confirmLogout(() => {
                this.submit();
            });
        });
    });

    // Delete buttons
    document.querySelectorAll('[data-confirm-action]').forEach(element => {
        element.addEventListener('click', function(e) {
            e.preventDefault();
            const action = this.dataset.confirmAction;
            const message = this.dataset.confirmMessage || 'Are you sure?';
            const href = this.href;
            
            confirmDialog.show({
                title: 'Confirm Action',
                message: message,
                confirmText: action,
                onConfirm: () => {
                    if (href) {
                        window.location.href = href;
                    }
                }
            });
        });
    });
});

// Track unsaved changes
let hasUnsavedChanges = false;

function trackFormChanges() {
    const forms = document.querySelectorAll('form[data-track-changes]');
    
    forms.forEach(form => {
        const initialData = new FormData(form);
        
        form.addEventListener('change', function() {
            const currentData = new FormData(form);
            hasUnsavedChanges = !formDataEquals(initialData, currentData);
            
            if (hasUnsavedChanges) {
                form.dataset.hasChanges = 'true';
            } else {
                form.dataset.hasChanges = 'false';
            }
        });
    });
}

function formDataEquals(a, b) {
    const aEntries = Array.from(a.entries()).sort();
    const bEntries = Array.from(b.entries()).sort();
    
    if (aEntries.length !== bEntries.length) return false;
    
    for (let i = 0; i < aEntries.length; i++) {
        if (aEntries[i][0] !== bEntries[i][0] || aEntries[i][1] !== bEntries[i][1]) {
            return false;
        }
    }
    
    return true;
}

// Warn before leaving with unsaved changes
window.addEventListener('beforeunload', function(e) {
    if (hasUnsavedChanges) {
        e.preventDefault();
        e.returnValue = '';
    }
});

// Initialize form tracking
document.addEventListener('DOMContentLoaded', trackFormChanges);

// Export for use in other modules
window.confirmDialog = confirmDialog;
window.confirmationHelpers = confirmationHelpers;