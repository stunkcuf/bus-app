// Table Enhancement Library - Sortable columns, filters, and export
(function() {
    'use strict';
    
    // Make tables sortable
    window.makeTableSortable = function(tableId) {
        const table = document.getElementById(tableId);
        if (!table) return;
        
        const headers = table.querySelectorAll('thead th');
        let sortColumn = -1;
        let sortDirection = 'asc';
        
        headers.forEach((header, index) => {
            // Skip action columns
            if (header.textContent.toLowerCase().includes('action')) return;
            
            header.style.cursor = 'pointer';
            header.style.userSelect = 'none';
            header.innerHTML = `
                ${header.textContent}
                <span class="sort-indicator ms-1" style="opacity: 0.3;">
                    <i class="bi bi-arrow-down-up"></i>
                </span>
            `;
            
            header.addEventListener('click', () => {
                sortTable(table, index, header);
            });
        });
        
        function sortTable(table, column, header) {
            const tbody = table.querySelector('tbody');
            const rows = Array.from(tbody.querySelectorAll('tr'));
            
            // Toggle sort direction
            if (sortColumn === column) {
                sortDirection = sortDirection === 'asc' ? 'desc' : 'asc';
            } else {
                sortDirection = 'asc';
                sortColumn = column;
            }
            
            // Update sort indicators
            table.querySelectorAll('.sort-indicator').forEach(indicator => {
                indicator.style.opacity = '0.3';
                indicator.innerHTML = '<i class="bi bi-arrow-down-up"></i>';
            });
            
            const indicator = header.querySelector('.sort-indicator');
            indicator.style.opacity = '1';
            indicator.innerHTML = sortDirection === 'asc' 
                ? '<i class="bi bi-arrow-up"></i>' 
                : '<i class="bi bi-arrow-down"></i>';
            
            // Sort rows
            rows.sort((a, b) => {
                const aValue = a.cells[column].textContent.trim();
                const bValue = b.cells[column].textContent.trim();
                
                // Check if numeric
                const aNum = parseFloat(aValue.replace(/[^0-9.-]/g, ''));
                const bNum = parseFloat(bValue.replace(/[^0-9.-]/g, ''));
                
                if (!isNaN(aNum) && !isNaN(bNum)) {
                    return sortDirection === 'asc' ? aNum - bNum : bNum - aNum;
                }
                
                // Date comparison
                const aDate = new Date(aValue);
                const bDate = new Date(bValue);
                if (!isNaN(aDate) && !isNaN(bDate)) {
                    return sortDirection === 'asc' ? aDate - bDate : bDate - aDate;
                }
                
                // String comparison
                if (sortDirection === 'asc') {
                    return aValue.localeCompare(bValue);
                } else {
                    return bValue.localeCompare(aValue);
                }
            });
            
            // Reorder rows in table
            rows.forEach(row => tbody.appendChild(row));
        }
    };
    
    // Add filter to table
    window.addTableFilter = function(tableId, placeholder = 'Filter table...') {
        const table = document.getElementById(tableId);
        if (!table) return;
        
        // Create filter input
        const filterContainer = document.createElement('div');
        filterContainer.className = 'mb-3';
        filterContainer.innerHTML = `
            <div class="input-group">
                <span class="input-group-text">
                    <i class="bi bi-funnel"></i>
                </span>
                <input type="text" class="form-control" id="${tableId}-filter" 
                       placeholder="${placeholder}">
                <button class="btn btn-outline-secondary" type="button" 
                        onclick="clearTableFilter('${tableId}')">
                    <i class="bi bi-x-circle"></i> Clear
                </button>
            </div>
        `;
        
        table.parentNode.insertBefore(filterContainer, table);
        
        // Add filter functionality
        const filterInput = document.getElementById(`${tableId}-filter`);
        filterInput.addEventListener('keyup', function() {
            const filter = this.value.toLowerCase();
            const tbody = table.querySelector('tbody');
            const rows = tbody.querySelectorAll('tr');
            
            rows.forEach(row => {
                const text = row.textContent.toLowerCase();
                row.style.display = text.includes(filter) ? '' : 'none';
            });
            
            // Update count
            updateFilteredCount(tableId);
        });
    };
    
    // Clear filter
    window.clearTableFilter = function(tableId) {
        const filterInput = document.getElementById(`${tableId}-filter`);
        if (filterInput) {
            filterInput.value = '';
            filterInput.dispatchEvent(new Event('keyup'));
        }
    };
    
    // Update filtered count
    function updateFilteredCount(tableId) {
        const table = document.getElementById(tableId);
        const tbody = table.querySelector('tbody');
        const totalRows = tbody.querySelectorAll('tr').length;
        const visibleRows = tbody.querySelectorAll('tr:not([style*="display: none"])').length;
        
        let countEl = document.getElementById(`${tableId}-count`);
        if (!countEl) {
            countEl = document.createElement('div');
            countEl.id = `${tableId}-count`;
            countEl.className = 'text-muted small mt-2';
            table.parentNode.insertBefore(countEl, table.nextSibling);
        }
        
        countEl.textContent = visibleRows < totalRows 
            ? `Showing ${visibleRows} of ${totalRows} rows`
            : `${totalRows} rows`;
    }
    
    // Export table to CSV
    window.exportTableToCSV = function(tableId, filename = 'export.csv') {
        const table = document.getElementById(tableId);
        if (!table) return;
        
        let csv = [];
        
        // Get headers
        const headers = [];
        table.querySelectorAll('thead th').forEach(header => {
            // Skip action columns
            if (!header.textContent.toLowerCase().includes('action')) {
                headers.push('"' + header.textContent.trim().replace(/"/g, '""') + '"');
            }
        });
        csv.push(headers.join(','));
        
        // Get visible rows
        const rows = table.querySelectorAll('tbody tr:not([style*="display: none"])');
        rows.forEach(row => {
            const rowData = [];
            row.querySelectorAll('td').forEach((cell, index) => {
                // Skip action columns (usually last)
                if (index < headers.length) {
                    rowData.push('"' + cell.textContent.trim().replace(/"/g, '""') + '"');
                }
            });
            csv.push(rowData.join(','));
        });
        
        // Download CSV
        const csvContent = csv.join('\n');
        const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' });
        const link = document.createElement('a');
        const url = URL.createObjectURL(blob);
        
        link.setAttribute('href', url);
        link.setAttribute('download', filename);
        link.style.visibility = 'hidden';
        document.body.appendChild(link);
        link.click();
        document.body.removeChild(link);
        
        // Show success message
        showNotification('Table exported successfully!', 'success');
    };
    
    // Export table to Excel (using simple HTML method)
    window.exportTableToExcel = function(tableId, filename = 'export') {
        const table = document.getElementById(tableId);
        if (!table) return;
        
        // Clone table and remove action columns
        const tableClone = table.cloneNode(true);
        
        // Remove action header
        const actionHeader = Array.from(tableClone.querySelectorAll('thead th'))
            .find(th => th.textContent.toLowerCase().includes('action'));
        if (actionHeader) {
            const actionIndex = Array.from(actionHeader.parentNode.children).indexOf(actionHeader);
            actionHeader.remove();
            
            // Remove action cells
            tableClone.querySelectorAll('tbody tr').forEach(row => {
                if (row.cells[actionIndex]) {
                    row.cells[actionIndex].remove();
                }
            });
        }
        
        // Remove hidden rows
        tableClone.querySelectorAll('tbody tr[style*="display: none"]').forEach(row => {
            row.remove();
        });
        
        // Create Excel file content
        const html = `
            <html xmlns:x="urn:schemas-microsoft-com:office:excel">
            <head>
                <meta charset="UTF-8">
                <style>
                    table { border-collapse: collapse; width: 100%; }
                    th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
                    th { background-color: #4CAF50; color: white; font-weight: bold; }
                    tr:nth-child(even) { background-color: #f2f2f2; }
                </style>
            </head>
            <body>
                ${tableClone.outerHTML}
            </body>
            </html>
        `;
        
        const blob = new Blob([html], { type: 'application/vnd.ms-excel' });
        const link = document.createElement('a');
        const url = URL.createObjectURL(blob);
        
        link.setAttribute('href', url);
        link.setAttribute('download', filename + '.xls');
        link.style.visibility = 'hidden';
        document.body.appendChild(link);
        link.click();
        document.body.removeChild(link);
        
        // Show success message
        showNotification('Table exported to Excel successfully!', 'success');
    };
    
    // Add export buttons to table
    window.addExportButtons = function(tableId, tableName = 'data') {
        const table = document.getElementById(tableId);
        if (!table) return;
        
        const exportContainer = document.createElement('div');
        exportContainer.className = 'mb-3 d-flex gap-2 justify-content-end';
        exportContainer.innerHTML = `
            <button class="btn btn-sm btn-outline-success" 
                    onclick="exportTableToCSV('${tableId}', '${tableName}_${new Date().toISOString().split('T')[0]}.csv')">
                <i class="bi bi-file-earmark-csv"></i> Export CSV
            </button>
            <button class="btn btn-sm btn-outline-success" 
                    onclick="exportTableToExcel('${tableId}', '${tableName}_${new Date().toISOString().split('T')[0]}')">
                <i class="bi bi-file-earmark-excel"></i> Export Excel
            </button>
        `;
        
        table.parentNode.insertBefore(exportContainer, table);
    };
    
    // Show notification
    function showNotification(message, type = 'info') {
        const notification = document.createElement('div');
        notification.className = `alert alert-${type} position-fixed top-0 end-0 m-3`;
        notification.style.zIndex = '9999';
        notification.innerHTML = `
            <div class="d-flex align-items-center">
                <i class="bi bi-check-circle me-2"></i>
                ${message}
            </div>
        `;
        
        document.body.appendChild(notification);
        
        setTimeout(() => {
            notification.style.opacity = '0';
            notification.style.transition = 'opacity 0.5s';
            setTimeout(() => notification.remove(), 500);
        }, 3000);
    }
    
    // Initialize all tables on page load
    window.initializeEnhancedTables = function() {
        // Find all data tables
        const tables = document.querySelectorAll('table[data-enhanced="true"]');
        tables.forEach(table => {
            if (table.id) {
                makeTableSortable(table.id);
                addTableFilter(table.id);
                addExportButtons(table.id, table.dataset.name || 'data');
            }
        });
    };
    
    // Auto-initialize on DOM ready
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', initializeEnhancedTables);
    } else {
        initializeEnhancedTables();
    }
})();