// Lazy Loading JavaScript Module
(function() {
    'use strict';

    // LazyLoader class for handling pagination and infinite scrolling
    class LazyLoader {
        constructor(options) {
            this.container = document.querySelector(options.container);
            this.endpoint = options.endpoint;
            this.itemTemplate = options.itemTemplate || this.defaultItemTemplate;
            this.loadingIndicator = options.loadingIndicator || this.createLoadingIndicator();
            this.filters = options.filters || {};
            this.searchInput = options.searchInput || null;
            this.filterInputs = options.filterInputs || [];
            
            this.page = 1;
            this.perPage = options.perPage || 25;
            this.loading = false;
            this.hasMore = true;
            this.totalItems = 0;
            this.items = [];
            
            this.init();
        }

        init() {
            // Add scroll event listener for infinite scrolling
            if (this.container) {
                const scrollContainer = this.container.closest('.table-container') || window;
                scrollContainer.addEventListener('scroll', this.handleScroll.bind(this));
            }

            // Add search input listener
            if (this.searchInput) {
                const searchElement = document.querySelector(this.searchInput);
                if (searchElement) {
                    let searchTimeout;
                    searchElement.addEventListener('input', (e) => {
                        clearTimeout(searchTimeout);
                        searchTimeout = setTimeout(() => {
                            this.filters.search = e.target.value;
                            this.reset();
                            this.loadMore();
                        }, 300);
                    });
                }
            }

            // Add filter input listeners
            this.filterInputs.forEach(filter => {
                const element = document.querySelector(filter.selector);
                if (element) {
                    element.addEventListener('change', (e) => {
                        this.filters[filter.name] = e.target.value;
                        this.reset();
                        this.loadMore();
                    });
                }
            });

            // Load initial data
            this.loadMore();
        }

        handleScroll(e) {
            const scrollElement = e.target === document ? document.documentElement : e.target;
            const scrollTop = scrollElement.scrollTop;
            const scrollHeight = scrollElement.scrollHeight;
            const clientHeight = scrollElement.clientHeight;

            if (scrollTop + clientHeight >= scrollHeight - 100 && !this.loading && this.hasMore) {
                this.loadMore();
            }
        }

        async loadMore() {
            if (this.loading || !this.hasMore) return;

            this.loading = true;
            this.showLoading();

            try {
                const params = new URLSearchParams({
                    page: this.page,
                    per_page: this.perPage,
                    ...this.filters
                });

                const response = await fetch(`${this.endpoint}?${params}`, {
                    credentials: 'same-origin',
                    headers: {
                        'Accept': 'application/json'
                    }
                });

                if (!response.ok) {
                    throw new Error(`HTTP error! status: ${response.status}`);
                }

                const data = await response.json();
                
                this.totalItems = data.total;
                this.hasMore = data.has_next;
                this.items = this.items.concat(data.data);
                
                this.renderItems(data.data);
                this.updateStatus();
                
                this.page++;
            } catch (error) {
                console.error('Error loading data:', error);
                this.showError('Failed to load data. Please try again.');
            } finally {
                this.loading = false;
                this.hideLoading();
            }
        }

        renderItems(items) {
            if (!this.container) return;

            const tbody = this.container.querySelector('tbody') || this.container;
            
            items.forEach(item => {
                const row = this.itemTemplate(item);
                tbody.insertAdjacentHTML('beforeend', row);
            });

            // Re-initialize any dynamic elements (tooltips, etc.)
            this.initializeDynamicElements();
        }

        defaultItemTemplate(item) {
            // Default template - should be overridden for specific use cases
            const cells = Object.values(item).map(value => `<td>${this.escapeHtml(value || '')}</td>`).join('');
            return `<tr>${cells}</tr>`;
        }

        reset() {
            this.page = 1;
            this.hasMore = true;
            this.items = [];
            if (this.container) {
                const tbody = this.container.querySelector('tbody') || this.container;
                tbody.innerHTML = '';
            }
        }

        showLoading() {
            if (this.loadingIndicator && this.container) {
                this.container.parentElement.appendChild(this.loadingIndicator);
            }
        }

        hideLoading() {
            if (this.loadingIndicator && this.loadingIndicator.parentElement) {
                this.loadingIndicator.parentElement.removeChild(this.loadingIndicator);
            }
        }

        createLoadingIndicator() {
            const indicator = document.createElement('div');
            indicator.className = 'lazy-loading-indicator';
            indicator.innerHTML = `
                <div class="spinner-border text-primary" role="status">
                    <span class="sr-only">Loading...</span>
                </div>
                <p>Loading more items...</p>
            `;
            indicator.style.cssText = `
                text-align: center;
                padding: 20px;
                font-size: 14px;
                color: #666;
            `;
            return indicator;
        }

        updateStatus() {
            const statusElement = document.querySelector('.lazy-load-status');
            if (statusElement) {
                const showing = Math.min(this.items.length, this.totalItems);
                statusElement.textContent = `Showing ${showing} of ${this.totalItems} items`;
            }
        }

        showError(message) {
            const errorDiv = document.createElement('div');
            errorDiv.className = 'alert alert-danger';
            errorDiv.textContent = message;
            if (this.container && this.container.parentElement) {
                this.container.parentElement.insertBefore(errorDiv, this.container);
                setTimeout(() => errorDiv.remove(), 5000);
            }
        }

        escapeHtml(text) {
            const div = document.createElement('div');
            div.textContent = text;
            return div.innerHTML;
        }

        initializeDynamicElements() {
            // Initialize tooltips if using Bootstrap
            if (typeof $ !== 'undefined' && $.fn.tooltip) {
                $('[data-toggle="tooltip"]').tooltip();
            }

            // Dispatch custom event for other initializations
            document.dispatchEvent(new CustomEvent('lazyLoadItemsAdded'));
        }
    }

    // Template functions for different data types
    const Templates = {
        student: (student) => {
            const activeClass = student.active ? 'success' : 'secondary';
            const activeText = student.active ? 'Active' : 'Inactive';
            return `
                <tr>
                    <td>${student.name || ''}</td>
                    <td>${student.guardian || ''}</td>
                    <td>${student.phone_number || ''}</td>
                    <td>${student.pickup_time || ''}</td>
                    <td>${student.driver || 'Unassigned'}</td>
                    <td>
                        <span class="badge badge-${activeClass}">${activeText}</span>
                    </td>
                    <td>
                        <a href="/student/${student.student_id}" class="btn btn-sm btn-info">
                            <i class="fas fa-eye"></i> View
                        </a>
                    </td>
                </tr>
            `;
        },

        bus: (bus) => {
            const statusClass = bus.status === 'active' ? 'success' : 
                               bus.status === 'maintenance' ? 'warning' : 'danger';
            return `
                <tr>
                    <td>${bus.bus_id}</td>
                    <td>${bus.model || 'N/A'}</td>
                    <td>${bus.capacity || 0}</td>
                    <td>
                        <span class="badge badge-${statusClass}">${bus.status}</span>
                    </td>
                    <td>${bus.current_mileage || 0}</td>
                    <td>
                        <a href="/bus/${bus.bus_id}" class="btn btn-sm btn-info">
                            <i class="fas fa-eye"></i> View
                        </a>
                    </td>
                </tr>
            `;
        },

        driverLog: (log) => {
            const mileage = log.end_mileage - log.begin_mileage;
            return `
                <tr>
                    <td>${log.date}</td>
                    <td>${log.driver}</td>
                    <td>${log.bus_id}</td>
                    <td>${log.period}</td>
                    <td>${log.departure_time || ''} - ${log.arrival_time || ''}</td>
                    <td>${mileage} miles</td>
                    <td>${log.attendance || 0}</td>
                </tr>
            `;
        },

        maintenanceRecord: (record) => {
            return `
                <tr>
                    <td>${record.maintenance_date}</td>
                    <td>${record.vehicle_id}</td>
                    <td>${record.category}</td>
                    <td>${record.description}</td>
                    <td>$${record.cost || 0}</td>
                    <td>${record.mechanic || ''}</td>
                    <td>
                        <span class="badge badge-${record.status === 'completed' ? 'success' : 'warning'}">
                            ${record.status}
                        </span>
                    </td>
                </tr>
            `;
        },

        fleetVehicle: (vehicle) => {
            const statusClass = vehicle.status === 'active' ? 'success' : 
                               vehicle.status === 'maintenance' ? 'warning' : 'danger';
            return `
                <tr>
                    <td>${vehicle.vehicle_number}</td>
                    <td>${vehicle.make} ${vehicle.model}</td>
                    <td>${vehicle.year}</td>
                    <td>${vehicle.license_plate}</td>
                    <td>
                        <span class="badge badge-${statusClass}">${vehicle.status}</span>
                    </td>
                    <td>${vehicle.mileage || 0}</td>
                    <td>${vehicle.last_service || 'N/A'}</td>
                </tr>
            `;
        }
    };

    // Export for use
    window.LazyLoader = LazyLoader;
    window.LazyLoadTemplates = Templates;

    // Auto-initialize lazy loaders based on data attributes
    document.addEventListener('DOMContentLoaded', function() {
        const lazyLoadContainers = document.querySelectorAll('[data-lazy-load]');
        
        lazyLoadContainers.forEach(container => {
            const endpoint = container.getAttribute('data-lazy-load-endpoint');
            const type = container.getAttribute('data-lazy-load-type');
            const perPage = parseInt(container.getAttribute('data-lazy-load-per-page') || '25');
            
            if (endpoint && type && Templates[type]) {
                new LazyLoader({
                    container: `#${container.id}`,
                    endpoint: endpoint,
                    itemTemplate: Templates[type],
                    perPage: perPage,
                    searchInput: container.getAttribute('data-lazy-load-search'),
                    filterInputs: JSON.parse(container.getAttribute('data-lazy-load-filters') || '[]')
                });
            }
        });
    });
})();