/* Responsive Tables JavaScript - Mobile and Tablet Optimization
   Converts standard tables to accessible card-based layouts on mobile */

class ResponsiveTable {
  constructor(tableElement, options = {}) {
    this.table = tableElement;
    this.options = {
      breakpoint: 768,
      enableSearch: true,
      enableSort: true,
      enablePagination: false,
      rowsPerPage: 10,
      cardTemplate: null,
      ...options
    };
    
    this.init();
  }

  init() {
    this.setupContainer();
    this.createMobileCards();
    this.setupSearch();
    this.setupSort();
    this.setupResponsiveHandling();
    
    if (this.options.enablePagination) {
      this.setupPagination();
    }
  }

  setupContainer() {
    // Wrap table in responsive container if not already wrapped
    if (!this.table.closest('.responsive-table-container')) {
      const container = document.createElement('div');
      container.className = 'responsive-table-container';
      
      this.table.parentNode.insertBefore(container, this.table);
      container.appendChild(this.table);
    }
    
    // Add responsive class to table
    this.table.classList.add('responsive-table');
  }

  createMobileCards() {
    const cardsContainer = document.createElement('div');
    cardsContainer.className = 'table-cards';
    
    // Get headers for mobile cards
    const headers = Array.from(this.table.querySelectorAll('thead th')).map(th => ({
      text: th.textContent.trim(),
      key: th.getAttribute('data-key') || th.textContent.toLowerCase().replace(/\s+/g, '_')
    }));
    
    // Create cards from table rows
    const rows = this.table.querySelectorAll('tbody tr');
    rows.forEach((row, index) => {
      const card = this.createCard(row, headers, index);
      cardsContainer.appendChild(card);
    });
    
    // Insert cards after table
    this.table.parentNode.appendChild(cardsContainer);
    this.cardsContainer = cardsContainer;
  }

  createCard(row, headers, index) {
    const cells = Array.from(row.querySelectorAll('td'));
    const card = document.createElement('div');
    card.className = 'table-card';
    card.setAttribute('data-row-index', index);
    
    // Use custom template if provided
    if (this.options.cardTemplate) {
      card.innerHTML = this.options.cardTemplate(cells, headers, row);
      return card;
    }
    
    // Default card template
    let cardHTML = '';
    
    // Card header with first cell as title
    if (cells.length > 0) {
      const titleCell = cells[0];
      const statusCell = cells.find(cell => 
        cell.classList.contains('status') || 
        cell.querySelector('.table-status')
      );
      
      cardHTML += `
        <div class="table-card-header">
          <h3 class="table-card-title">${titleCell.textContent.trim()}</h3>
          ${statusCell ? `<div class="table-card-status">${statusCell.innerHTML}</div>` : ''}
        </div>
      `;
    }
    
    // Card body with other fields
    cardHTML += '<div class="table-card-body">';
    
    cells.forEach((cell, cellIndex) => {
      if (cellIndex === 0) return; // Skip title cell
      
      const header = headers[cellIndex];
      const cellContent = cell.innerHTML.trim();
      
      if (cellContent && !cell.classList.contains('actions')) {
        cardHTML += `
          <div class="table-card-field">
            <div class="table-card-label">${header ? header.text : `Field ${cellIndex + 1}`}</div>
            <div class="table-card-value">${cellContent}</div>
          </div>
        `;
      }
    });
    
    cardHTML += '</div>';
    
    // Card actions
    const actionsCell = cells.find(cell => cell.classList.contains('actions'));
    if (actionsCell) {
      cardHTML += `
        <div class="table-card-actions">
          ${actionsCell.innerHTML}
        </div>
      `;
    }
    
    card.innerHTML = cardHTML;
    return card;
  }

  setupSearch() {
    if (!this.options.enableSearch) return;
    
    const container = this.table.closest('.responsive-table-container');
    
    // Create search controls if they don't exist
    let controlsDiv = container.querySelector('.table-controls');
    if (!controlsDiv) {
      controlsDiv = document.createElement('div');
      controlsDiv.className = 'table-controls';
      container.insertBefore(controlsDiv, container.firstChild);
    }
    
    // Add search input if it doesn't exist
    let searchDiv = controlsDiv.querySelector('.table-search');
    if (!searchDiv) {
      searchDiv = document.createElement('div');
      searchDiv.className = 'table-search';
      searchDiv.innerHTML = `
        <label for="table-search-${Date.now()}" class="sr-only">Search table</label>
        <i class="bi bi-search" aria-hidden="true" style="color: var(--color-text-muted);"></i>
        <input type="text" 
               id="table-search-${Date.now()}"
               class="table-search-input" 
               placeholder="Search records..." 
               aria-label="Search table data">
      `;
      controlsDiv.appendChild(searchDiv);
    }
    
    // Add table info
    let infoDiv = controlsDiv.querySelector('.table-info');
    if (!infoDiv) {
      infoDiv = document.createElement('div');
      infoDiv.className = 'table-info';
      controlsDiv.appendChild(infoDiv);
    }
    
    // Setup search functionality
    const searchInput = searchDiv.querySelector('.table-search-input');
    searchInput.addEventListener('input', (e) => {
      this.filterTable(e.target.value);
    });
    
    // Update table info
    this.updateTableInfo();
  }

  filterTable(searchTerm) {
    const term = searchTerm.toLowerCase().trim();
    const rows = this.table.querySelectorAll('tbody tr');
    const cards = this.cardsContainer.querySelectorAll('.table-card');
    
    let visibleCount = 0;
    
    rows.forEach((row, index) => {
      const text = row.textContent.toLowerCase();
      const isVisible = !term || text.includes(term);
      
      row.style.display = isVisible ? '' : 'none';
      cards[index].style.display = isVisible ? '' : 'none';
      
      if (isVisible) visibleCount++;
    });
    
    this.updateTableInfo(visibleCount);
  }

  updateTableInfo(filteredCount = null) {
    const infoDiv = this.table.closest('.responsive-table-container').querySelector('.table-info');
    if (!infoDiv) return;
    
    const totalRows = this.table.querySelectorAll('tbody tr').length;
    const displayCount = filteredCount !== null ? filteredCount : totalRows;
    
    if (filteredCount !== null && filteredCount < totalRows) {
      infoDiv.textContent = `Showing ${displayCount} of ${totalRows} records`;
    } else {
      infoDiv.textContent = `${totalRows} record${totalRows !== 1 ? 's' : ''}`;
    }
  }

  setupSort() {
    if (!this.options.enableSort) return;
    
    const headers = this.table.querySelectorAll('thead th');
    headers.forEach((header, index) => {
      if (header.classList.contains('no-sort')) return;
      
      header.classList.add('sortable-header');
      header.setAttribute('tabindex', '0');
      header.setAttribute('role', 'button');
      header.setAttribute('aria-label', `Sort by ${header.textContent.trim()}`);
      
      header.addEventListener('click', () => {
        this.sortTable(index, header);
      });
      
      header.addEventListener('keydown', (e) => {
        if (e.key === 'Enter' || e.key === ' ') {
          e.preventDefault();
          this.sortTable(index, header);
        }
      });
    });
  }

  sortTable(columnIndex, headerElement) {
    const tbody = this.table.querySelector('tbody');
    const rows = Array.from(tbody.querySelectorAll('tr'));
    
    // Determine sort direction
    const currentSort = headerElement.getAttribute('data-sort');
    const isAscending = currentSort !== 'asc';
    
    // Clear other sort indicators
    this.table.querySelectorAll('thead th').forEach(th => {
      th.classList.remove('sorted-asc', 'sorted-desc');
      th.removeAttribute('data-sort');
    });
    
    // Set sort indicator
    headerElement.classList.add(isAscending ? 'sorted-asc' : 'sorted-desc');
    headerElement.setAttribute('data-sort', isAscending ? 'asc' : 'desc');
    headerElement.setAttribute('aria-label', 
      `Sorted by ${headerElement.textContent.trim()} ${isAscending ? 'ascending' : 'descending'}`
    );
    
    // Sort rows
    rows.sort((a, b) => {
      const aText = a.cells[columnIndex].textContent.trim();
      const bText = b.cells[columnIndex].textContent.trim();
      
      // Try to sort as numbers first
      const aNum = parseFloat(aText.replace(/[^0-9.-]/g, ''));
      const bNum = parseFloat(bText.replace(/[^0-9.-]/g, ''));
      
      if (!isNaN(aNum) && !isNaN(bNum)) {
        return isAscending ? aNum - bNum : bNum - aNum;
      }
      
      // Sort as text
      return isAscending 
        ? aText.localeCompare(bText)
        : bText.localeCompare(aText);
    });
    
    // Reorder DOM
    rows.forEach(row => tbody.appendChild(row));
    
    // Re-create mobile cards with new order
    this.recreateMobileCards();
  }

  recreateMobileCards() {
    // Remove existing cards
    this.cardsContainer.innerHTML = '';
    
    // Get headers
    const headers = Array.from(this.table.querySelectorAll('thead th')).map(th => ({
      text: th.textContent.trim(),
      key: th.getAttribute('data-key') || th.textContent.toLowerCase().replace(/\s+/g, '_')
    }));
    
    // Recreate cards
    const rows = this.table.querySelectorAll('tbody tr');
    rows.forEach((row, index) => {
      const card = this.createCard(row, headers, index);
      this.cardsContainer.appendChild(card);
    });
  }

  setupResponsiveHandling() {
    const handleResize = () => {
      // This is handled by CSS, but we can add any JS-specific responsive logic here
      this.updateTableInfo();
    };
    
    window.addEventListener('resize', handleResize);
    
    // Store reference for cleanup
    this.resizeHandler = handleResize;
  }

  setupPagination() {
    if (!this.options.enablePagination) return;
    
    const container = this.table.closest('.responsive-table-container');
    
    // Create pagination container
    const paginationDiv = document.createElement('div');
    paginationDiv.className = 'table-pagination';
    
    paginationDiv.innerHTML = `
      <div class="pagination-info">
        <span id="pagination-info-${Date.now()}"></span>
      </div>
      <div class="pagination-controls">
        <button class="pagination-btn" id="prev-btn" aria-label="Previous page">
          <i class="bi bi-chevron-left" aria-hidden="true"></i>
        </button>
        <span class="pagination-numbers"></span>
        <button class="pagination-btn" id="next-btn" aria-label="Next page">
          <i class="bi bi-chevron-right" aria-hidden="true"></i>
        </button>
      </div>
    `;
    
    container.appendChild(paginationDiv);
    
    // Initialize pagination logic
    this.currentPage = 1;
    this.totalPages = Math.ceil(this.table.querySelectorAll('tbody tr').length / this.options.rowsPerPage);
    
    this.setupPaginationEvents(paginationDiv);
    this.updatePagination();
  }

  setupPaginationEvents(paginationDiv) {
    const prevBtn = paginationDiv.querySelector('#prev-btn');
    const nextBtn = paginationDiv.querySelector('#next-btn');
    
    prevBtn.addEventListener('click', () => {
      if (this.currentPage > 1) {
        this.currentPage--;
        this.updatePagination();
      }
    });
    
    nextBtn.addEventListener('click', () => {
      if (this.currentPage < this.totalPages) {
        this.currentPage++;
        this.updatePagination();
      }
    });
  }

  updatePagination() {
    const rows = this.table.querySelectorAll('tbody tr');
    const cards = this.cardsContainer.querySelectorAll('.table-card');
    const startIndex = (this.currentPage - 1) * this.options.rowsPerPage;
    const endIndex = startIndex + this.options.rowsPerPage;
    
    // Show/hide rows and cards
    rows.forEach((row, index) => {
      const isVisible = index >= startIndex && index < endIndex;
      row.style.display = isVisible ? '' : 'none';
      if (cards[index]) {
        cards[index].style.display = isVisible ? '' : 'none';
      }
    });
    
    // Update pagination info
    const infoSpan = document.querySelector('[id^="pagination-info-"]');
    if (infoSpan) {
      const start = startIndex + 1;
      const end = Math.min(endIndex, rows.length);
      infoSpan.textContent = `Showing ${start}-${end} of ${rows.length}`;
    }
    
    // Update buttons
    const container = this.table.closest('.responsive-table-container');
    const prevBtn = container.querySelector('#prev-btn');
    const nextBtn = container.querySelector('#next-btn');
    
    prevBtn.disabled = this.currentPage === 1;
    nextBtn.disabled = this.currentPage === this.totalPages;
  }

  // Public methods
  refresh() {
    this.recreateMobileCards();
    this.updateTableInfo();
    
    if (this.options.enablePagination) {
      this.totalPages = Math.ceil(this.table.querySelectorAll('tbody tr').length / this.options.rowsPerPage);
      this.updatePagination();
    }
  }

  destroy() {
    if (this.resizeHandler) {
      window.removeEventListener('resize', this.resizeHandler);
    }
    
    if (this.cardsContainer) {
      this.cardsContainer.remove();
    }
  }

  // Static method to initialize all tables
  static initAll(selector = '.table', options = {}) {
    const tables = document.querySelectorAll(selector);
    const instances = [];
    
    tables.forEach(table => {
      const instance = new ResponsiveTable(table, options);
      instances.push(instance);
      
      // Store instance on element for later access
      table._responsiveTableInstance = instance;
    });
    
    return instances;
  }
}

// Auto-initialize on DOM ready
document.addEventListener('DOMContentLoaded', () => {
  // Look for tables with responsive-table class
  ResponsiveTable.initAll('.responsive-table, .table');
});

// Export for module use
if (typeof module !== 'undefined' && module.exports) {
  module.exports = ResponsiveTable;
}