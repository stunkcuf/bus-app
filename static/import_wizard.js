// Import Data Wizard
function initImportWizard(importType) {
    const wizard = new StepWizard({
        containerId: 'wizardContainer',
        steps: [
            {
                title: 'Select File',
                description: 'Choose the Excel file you want to import.',
                render: function(data) {
                    return `
                        <div class="form-group">
                            <label for="importFile" class="form-label">
                                Excel File (.xlsx or .xls)
                                <span class="text-danger">*</span>
                            </label>
                            <input type="file" class="form-control" id="importFile" 
                                   accept=".xlsx,.xls" required>
                            <small class="form-text text-muted">
                                Maximum file size: 10MB
                            </small>
                        </div>
                        
                        <div class="mt-3">
                            <h5>File Requirements:</h5>
                            <ul class="text-muted">
                                ${getImportRequirements(importType)}
                            </ul>
                        </div>
                        
                        <div class="mt-3">
                            <a href="/templates/download/${importType}" class="btn btn-outline-primary">
                                <i class="bi bi-download"></i> Download Template
                            </a>
                        </div>
                    `;
                },
                collect: function(wizard) {
                    const fileInput = document.getElementById('importFile');
                    if (fileInput.files.length > 0) {
                        wizard.data.file = fileInput.files[0];
                        wizard.data.fileName = fileInput.files[0].name;
                    }
                },
                validate: function(data) {
                    if (!data.file) {
                        return 'Please select a file to import';
                    }
                    
                    const maxSize = 10 * 1024 * 1024; // 10MB
                    if (data.file.size > maxSize) {
                        return 'File size exceeds 10MB limit';
                    }
                    
                    const validExtensions = ['.xlsx', '.xls'];
                    const extension = data.fileName.substring(data.fileName.lastIndexOf('.')).toLowerCase();
                    if (!validExtensions.includes(extension)) {
                        return 'Please select a valid Excel file (.xlsx or .xls)';
                    }
                    
                    return true;
                }
            },
            {
                title: 'Column Mapping',
                description: 'Map the columns in your file to the system fields.',
                render: function(data) {
                    return `
                        <div id="columnMappingContainer">
                            <div class="wizard-loading">
                                <div class="spinner-border"></div>
                                <p>Analyzing file structure...</p>
                            </div>
                        </div>
                    `;
                },
                init: function(wizard) {
                    // Upload file and get column headers
                    const formData = new FormData();
                    formData.append('file', wizard.data.file);
                    formData.append('type', importType);
                    
                    fetch('/api/analyze-import-file', {
                        method: 'POST',
                        body: formData
                    })
                    .then(response => response.json())
                    .then(result => {
                        if (result.error) {
                            throw new Error(result.error);
                        }
                        
                        wizard.data.columns = result.columns;
                        wizard.data.sampleData = result.sampleData;
                        wizard.data.requiredFields = result.requiredFields;
                        
                        renderColumnMapping(wizard);
                    })
                    .catch(error => {
                        document.getElementById('columnMappingContainer').innerHTML = `
                            <div class="alert alert-danger">
                                <i class="bi bi-exclamation-triangle"></i> ${error.message || 'Failed to analyze file'}
                            </div>
                        `;
                    });
                },
                collect: function(wizard) {
                    // Collect column mappings
                    wizard.data.columnMappings = {};
                    const selects = document.querySelectorAll('.column-mapping-select');
                    selects.forEach(select => {
                        if (select.value) {
                            wizard.data.columnMappings[select.dataset.field] = select.value;
                        }
                    });
                },
                validate: function(data) {
                    // Check that all required fields are mapped
                    for (const field of data.requiredFields) {
                        if (!data.columnMappings[field]) {
                            return `Please map the required field: ${field}`;
                        }
                    }
                    return true;
                }
            },
            {
                title: 'Preview & Validate',
                description: 'Review the data that will be imported.',
                render: function(data) {
                    return `
                        <div id="previewContainer">
                            <div class="wizard-loading">
                                <div class="spinner-border"></div>
                                <p>Validating data...</p>
                            </div>
                        </div>
                    `;
                },
                init: function(wizard) {
                    // Validate and preview data
                    const formData = new FormData();
                    formData.append('file', wizard.data.file);
                    formData.append('type', importType);
                    formData.append('mappings', JSON.stringify(wizard.data.columnMappings));
                    
                    fetch('/api/preview-import', {
                        method: 'POST',
                        body: formData
                    })
                    .then(response => response.json())
                    .then(result => {
                        if (result.error) {
                            throw new Error(result.error);
                        }
                        
                        wizard.data.validRecords = result.validRecords;
                        wizard.data.invalidRecords = result.invalidRecords;
                        wizard.data.warnings = result.warnings;
                        
                        renderPreview(wizard);
                    })
                    .catch(error => {
                        document.getElementById('previewContainer').innerHTML = `
                            <div class="alert alert-danger">
                                <i class="bi bi-exclamation-triangle"></i> ${error.message || 'Failed to validate data'}
                            </div>
                        `;
                    });
                },
                validate: function(data) {
                    if (data.validRecords === 0) {
                        return 'No valid records found to import';
                    }
                    
                    if (data.invalidRecords > data.validRecords) {
                        const proceed = confirm(`Warning: ${data.invalidRecords} invalid records found (${data.validRecords} valid). Do you want to proceed with importing only the valid records?`);
                        if (!proceed) {
                            return 'Import cancelled due to too many invalid records';
                        }
                    }
                    
                    return true;
                }
            },
            {
                title: 'Import Options',
                description: 'Choose how to handle the import.',
                fields: [
                    {
                        id: 'importMode',
                        type: 'radio',
                        label: 'Import Mode',
                        required: true,
                        options: [
                            { 
                                value: 'append', 
                                text: 'Add to existing data (recommended)' 
                            },
                            { 
                                value: 'update', 
                                text: 'Update existing records (match by ID)' 
                            },
                            { 
                                value: 'replace', 
                                text: '⚠️ Replace all existing data (dangerous)' 
                            }
                        ]
                    },
                    {
                        id: 'skipDuplicates',
                        type: 'checkbox',
                        checkLabel: 'Skip duplicate records',
                        value: true
                    },
                    {
                        id: 'validateReferences',
                        type: 'checkbox',
                        checkLabel: 'Validate all references (e.g., driver names, route IDs)',
                        value: true
                    }
                ],
                render: function(data) {
                    let html = wizard.renderStepContent(this);
                    
                    html += `
                        <div class="mt-4 p-3 bg-light rounded">
                            <h5>Import Summary:</h5>
                            <ul class="mb-0">
                                <li><strong>${data.validRecords}</strong> records will be imported</li>
                                ${data.invalidRecords > 0 ? `<li><strong>${data.invalidRecords}</strong> records will be skipped due to errors</li>` : ''}
                                ${data.warnings > 0 ? `<li><strong>${data.warnings}</strong> warnings found (non-critical)</li>` : ''}
                            </ul>
                        </div>
                    `;
                    
                    return html;
                }
            },
            {
                title: 'Confirm Import',
                description: 'Review all settings before starting the import.',
                render: function(data) {
                    const modeDescriptions = {
                        'append': 'Add new records to existing data',
                        'update': 'Update existing records where IDs match',
                        'replace': 'Delete all existing data and import new data'
                    };
                    
                    return `
                        <div class="wizard-summary">
                            <h4>Import Configuration</h4>
                            <table>
                                <tr>
                                    <td>Import Type:</td>
                                    <td>${importType.toUpperCase()}</td>
                                </tr>
                                <tr>
                                    <td>File:</td>
                                    <td>${data.fileName}</td>
                                </tr>
                                <tr>
                                    <td>Records to Import:</td>
                                    <td>${data.validRecords}</td>
                                </tr>
                                <tr>
                                    <td>Import Mode:</td>
                                    <td>${modeDescriptions[data.importMode]}</td>
                                </tr>
                                <tr>
                                    <td>Skip Duplicates:</td>
                                    <td>${data.skipDuplicates ? 'Yes' : 'No'}</td>
                                </tr>
                                <tr>
                                    <td>Validate References:</td>
                                    <td>${data.validateReferences ? 'Yes' : 'No'}</td>
                                </tr>
                            </table>
                        </div>
                        
                        ${data.importMode === 'replace' ? `
                        <div class="alert alert-danger mt-3">
                            <i class="bi bi-exclamation-triangle"></i> 
                            <strong>Warning:</strong> Replace mode will permanently delete all existing ${importType} data!
                        </div>
                        ` : ''}
                        
                        <div class="form-check mt-3">
                            <input class="form-check-input" type="checkbox" id="confirmImport">
                            <label class="form-check-label" for="confirmImport">
                                I understand the import settings and want to proceed
                            </label>
                        </div>
                    `;
                },
                validate: function(data) {
                    const confirmBox = document.getElementById('confirmImport');
                    if (!confirmBox.checked) {
                        return 'Please confirm that you want to proceed with the import';
                    }
                    return true;
                }
            }
        ],
        onComplete: function(data) {
            // Start the import
            const wizardContainer = document.getElementById('wizardContainer');
            wizardContainer.innerHTML = `
                <div class="wizard-loading">
                    <div class="spinner-border"></div>
                    <p>Importing data...</p>
                    <div class="progress mt-3" style="height: 20px;">
                        <div id="importProgress" class="progress-bar progress-bar-striped progress-bar-animated" 
                             role="progressbar" style="width: 0%">0%</div>
                    </div>
                    <p id="importStatus" class="mt-2 text-muted">Preparing import...</p>
                </div>
            `;
            
            // Get CSRF token
            const csrfToken = document.querySelector('input[name="csrf_token"]').value;
            
            // Prepare form data
            const formData = new FormData();
            formData.append('file', data.file);
            formData.append('type', importType);
            formData.append('mappings', JSON.stringify(data.columnMappings));
            formData.append('mode', data.importMode);
            formData.append('skipDuplicates', data.skipDuplicates);
            formData.append('validateReferences', data.validateReferences);
            formData.append('csrf_token', csrfToken);
            
            // Start import with progress tracking
            startImportWithProgress(formData, wizardContainer);
        },
        onCancel: function() {
            // Close the wizard
            document.getElementById('wizardContainer').style.display = 'none';
        }
    });
    
    window.wizard = wizard;
}

// Helper functions
function getImportRequirements(type) {
    const requirements = {
        'students': `
            <li>Student ID (unique identifier)</li>
            <li>Student Name</li>
            <li>Phone Number</li>
            <li>Guardian Name</li>
            <li>Route ID (optional)</li>
            <li>Driver Username (optional)</li>
        `,
        'ecse': `
            <li>Student ID (unique identifier)</li>
            <li>First Name</li>
            <li>Last Name</li>
            <li>Date of Birth (MM/DD/YYYY)</li>
            <li>IEP Status</li>
            <li>Primary Disability</li>
        `,
        'mileage': `
            <li>Date (MM/DD/YYYY)</li>
            <li>Driver Username</li>
            <li>Bus ID</li>
            <li>Start Mileage</li>
            <li>End Mileage</li>
            <li>Route ID</li>
        `
    };
    
    return requirements[type] || '<li>Please check the template for required fields</li>';
}

function renderColumnMapping(wizard) {
    const container = document.getElementById('columnMappingContainer');
    const systemFields = getSystemFields(importType);
    
    let html = `
        <div class="alert alert-info">
            <i class="bi bi-info-circle"></i> 
            Map each system field to a column in your Excel file. Required fields are marked with *.
        </div>
        
        <div class="table-responsive">
            <table class="table table-sm">
                <thead>
                    <tr>
                        <th>System Field</th>
                        <th>Excel Column</th>
                        <th>Sample Data</th>
                    </tr>
                </thead>
                <tbody>
    `;
    
    systemFields.forEach(field => {
        const isRequired = wizard.data.requiredFields.includes(field.id);
        html += `
            <tr>
                <td>
                    ${field.name} ${isRequired ? '<span class="text-danger">*</span>' : ''}
                </td>
                <td>
                    <select class="form-select form-select-sm column-mapping-select" 
                            data-field="${field.id}">
                        <option value="">-- Select Column --</option>
                        ${wizard.data.columns.map(col => {
                            const selected = field.autoMatch && col.toLowerCase().includes(field.autoMatch) ? 'selected' : '';
                            return `<option value="${col}" ${selected}>${col}</option>`;
                        }).join('')}
                    </select>
                </td>
                <td>
                    <small class="text-muted sample-data" data-field="${field.id}">-</small>
                </td>
            </tr>
        `;
    });
    
    html += `
                </tbody>
            </table>
        </div>
    `;
    
    container.innerHTML = html;
    
    // Update sample data when column is selected
    container.querySelectorAll('.column-mapping-select').forEach(select => {
        select.addEventListener('change', function() {
            updateSampleData(this, wizard.data.sampleData);
        });
        // Trigger initial sample data update
        if (select.value) {
            updateSampleData(select, wizard.data.sampleData);
        }
    });
}

function updateSampleData(select, sampleData) {
    const field = select.dataset.field;
    const column = select.value;
    const sampleCell = document.querySelector(`.sample-data[data-field="${field}"]`);
    
    if (column && sampleData[column]) {
        sampleCell.textContent = sampleData[column].slice(0, 3).join(', ') + '...';
    } else {
        sampleCell.textContent = '-';
    }
}

function renderPreview(wizard) {
    const container = document.getElementById('previewContainer');
    
    let html = '';
    
    // Show validation results
    if (wizard.data.invalidRecords > 0) {
        html += `
            <div class="alert alert-warning">
                <i class="bi bi-exclamation-triangle"></i> 
                <strong>${wizard.data.invalidRecords} records</strong> contain errors and will be skipped.
                <button class="btn btn-sm btn-outline-warning ms-2" onclick="showValidationErrors()">
                    View Errors
                </button>
            </div>
        `;
    }
    
    if (wizard.data.warnings > 0) {
        html += `
            <div class="alert alert-info">
                <i class="bi bi-info-circle"></i> 
                <strong>${wizard.data.warnings} warnings</strong> found (non-critical).
            </div>
        `;
    }
    
    html += `
        <div class="d-flex justify-content-between align-items-center mb-3">
            <h5>Valid Records Preview (${wizard.data.validRecords} total)</h5>
            <span class="text-muted">Showing first 10 records</span>
        </div>
        
        <div class="table-responsive" style="max-height: 400px; overflow-y: auto;">
            <table class="table table-sm table-striped">
                <thead class="sticky-top bg-light">
                    <tr>
                        ${Object.keys(wizard.data.columnMappings).map(field => 
                            `<th>${field}</th>`
                        ).join('')}
                    </tr>
                </thead>
                <tbody>
                    <!-- Preview data would be rendered here -->
                    <tr>
                        <td colspan="${Object.keys(wizard.data.columnMappings).length}" class="text-center text-muted">
                            Preview data will be shown here...
                        </td>
                    </tr>
                </tbody>
            </table>
        </div>
    `;
    
    container.innerHTML = html;
}

function getSystemFields(type) {
    const fields = {
        'students': [
            { id: 'student_id', name: 'Student ID', autoMatch: 'id' },
            { id: 'name', name: 'Student Name', autoMatch: 'name' },
            { id: 'phone_number', name: 'Phone Number', autoMatch: 'phone' },
            { id: 'guardian', name: 'Guardian Name', autoMatch: 'guardian' },
            { id: 'route_id', name: 'Route ID', autoMatch: 'route' },
            { id: 'driver', name: 'Driver Username', autoMatch: 'driver' }
        ],
        'ecse': [
            { id: 'student_id', name: 'Student ID', autoMatch: 'id' },
            { id: 'first_name', name: 'First Name', autoMatch: 'first' },
            { id: 'last_name', name: 'Last Name', autoMatch: 'last' },
            { id: 'date_of_birth', name: 'Date of Birth', autoMatch: 'birth' },
            { id: 'iep_status', name: 'IEP Status', autoMatch: 'iep' },
            { id: 'primary_disability', name: 'Primary Disability', autoMatch: 'disability' }
        ],
        'mileage': [
            { id: 'date', name: 'Date', autoMatch: 'date' },
            { id: 'driver', name: 'Driver', autoMatch: 'driver' },
            { id: 'bus_id', name: 'Bus ID', autoMatch: 'bus' },
            { id: 'start_mileage', name: 'Start Mileage', autoMatch: 'start' },
            { id: 'end_mileage', name: 'End Mileage', autoMatch: 'end' },
            { id: 'route_id', name: 'Route ID', autoMatch: 'route' }
        ]
    };
    
    return fields[type] || [];
}

function startImportWithProgress(formData, container) {
    // This would typically use WebSocket or Server-Sent Events for real-time progress
    // For now, we'll simulate with a regular POST
    
    fetch('/import-data', {
        method: 'POST',
        body: formData
    })
    .then(response => response.json())
    .then(result => {
        if (result.success) {
            container.innerHTML = `
                <div class="wizard-success">
                    <i class="bi bi-check-circle"></i>
                    <h3>Import Completed Successfully!</h3>
                    <div class="mt-3">
                        <ul class="list-unstyled">
                            <li><i class="bi bi-check text-success"></i> ${result.imported} records imported</li>
                            ${result.updated ? `<li><i class="bi bi-arrow-repeat text-info"></i> ${result.updated} records updated</li>` : ''}
                            ${result.skipped ? `<li><i class="bi bi-x text-warning"></i> ${result.skipped} records skipped</li>` : ''}
                        </ul>
                    </div>
                    <button class="btn btn-primary mt-3" onclick="location.href='/${importType}'">
                        <i class="bi bi-eye"></i> View Imported Data
                    </button>
                </div>
            `;
        } else {
            throw new Error(result.error || 'Import failed');
        }
    })
    .catch(error => {
        container.innerHTML = `
            <div class="alert alert-danger">
                <i class="bi bi-exclamation-triangle"></i> 
                <strong>Import Failed:</strong> ${error.message}
                <div class="mt-3">
                    <button class="btn btn-primary" onclick="location.reload()">
                        <i class="bi bi-arrow-left"></i> Try Again
                    </button>
                </div>
            </div>
        `;
    });
}