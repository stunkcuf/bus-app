// Auto-complete and Enhanced Data Entry System

class AutoComplete {
    constructor(inputId, options = {}) {
        this.input = document.getElementById(inputId);
        if (!this.input) return;
        
        this.options = {
            minChars: options.minChars || 2,
            delay: options.delay || 300,
            maxResults: options.maxResults || 10,
            source: options.source || [],
            onSelect: options.onSelect || null,
            template: options.template || null,
            placeholder: options.placeholder || 'Start typing to search...',
            noResultsText: options.noResultsText || 'No results found',
            loadingText: options.loadingText || 'Loading...'
        };
        
        this.results = [];
        this.selectedIndex = -1;
        this.isOpen = false;
        this.debounceTimer = null;
        
        this.init();
    }
    
    init() {
        // Create dropdown container
        this.createDropdown();
        
        // Add event listeners
        this.input.addEventListener('input', this.handleInput.bind(this));
        this.input.addEventListener('keydown', this.handleKeydown.bind(this));
        this.input.addEventListener('blur', this.handleBlur.bind(this));
        this.input.addEventListener('focus', this.handleFocus.bind(this));
        
        // Close on outside click
        document.addEventListener('click', (e) => {
            if (!this.input.contains(e.target) && !this.dropdown.contains(e.target)) {
                this.close();
            }
        });
    }
    
    createDropdown() {
        this.dropdown = document.createElement('div');
        this.dropdown.className = 'autocomplete-dropdown';
        this.dropdown.style.display = 'none';
        
        // Position relative to input
        const parent = this.input.parentElement;
        parent.style.position = 'relative';
        parent.appendChild(this.dropdown);
        
        // Set ARIA attributes
        this.input.setAttribute('role', 'combobox');
        this.input.setAttribute('aria-autocomplete', 'list');
        this.input.setAttribute('aria-expanded', 'false');
        this.input.setAttribute('aria-controls', `autocomplete-${this.input.id}`);
        
        this.dropdown.setAttribute('id', `autocomplete-${this.input.id}`);
        this.dropdown.setAttribute('role', 'listbox');
    }
    
    handleInput(e) {
        const value = e.target.value.trim();
        
        // Clear existing timer
        clearTimeout(this.debounceTimer);
        
        if (value.length < this.options.minChars) {
            this.close();
            return;
        }
        
        // Show loading state
        this.showLoading();
        
        // Debounce the search
        this.debounceTimer = setTimeout(() => {
            this.search(value);
        }, this.options.delay);
    }
    
    handleKeydown(e) {
        if (!this.isOpen) return;
        
        switch (e.key) {
            case 'ArrowDown':
                e.preventDefault();
                this.selectNext();
                break;
            case 'ArrowUp':
                e.preventDefault();
                this.selectPrev();
                break;
            case 'Enter':
                e.preventDefault();
                if (this.selectedIndex >= 0) {
                    this.selectItem(this.results[this.selectedIndex]);
                }
                break;
            case 'Escape':
                this.close();
                break;
        }
    }
    
    handleBlur() {
        // Delay to allow click on dropdown items
        setTimeout(() => this.close(), 200);
    }
    
    handleFocus() {
        if (this.input.value.length >= this.options.minChars) {
            this.search(this.input.value);
        }
    }
    
    async search(query) {
        try {
            let results;
            
            if (typeof this.options.source === 'function') {
                // Async data source
                results = await this.options.source(query);
            } else if (Array.isArray(this.options.source)) {
                // Static array - filter locally
                results = this.options.source.filter(item => {
                    const searchStr = typeof item === 'string' ? item : item.label || item.value;
                    return searchStr.toLowerCase().includes(query.toLowerCase());
                });
            } else if (typeof this.options.source === 'string') {
                // URL endpoint
                const response = await fetch(`${this.options.source}?q=${encodeURIComponent(query)}`);
                results = await response.json();
            }
            
            this.results = results.slice(0, this.options.maxResults);
            this.showResults();
            
        } catch (error) {
            console.error('AutoComplete search error:', error);
            this.showError();
        }
    }
    
    showLoading() {
        this.dropdown.innerHTML = `<div class="autocomplete-loading">${this.options.loadingText}</div>`;
        this.open();
    }
    
    showError() {
        this.dropdown.innerHTML = '<div class="autocomplete-error">Error loading results</div>';
        this.open();
    }
    
    showResults() {
        if (this.results.length === 0) {
            this.dropdown.innerHTML = `<div class="autocomplete-no-results">${this.options.noResultsText}</div>`;
            this.open();
            return;
        }
        
        const ul = document.createElement('ul');
        ul.className = 'autocomplete-results';
        
        this.results.forEach((item, index) => {
            const li = document.createElement('li');
            li.className = 'autocomplete-item';
            li.setAttribute('role', 'option');
            li.setAttribute('aria-selected', index === this.selectedIndex);
            
            if (index === this.selectedIndex) {
                li.classList.add('selected');
            }
            
            // Use custom template if provided
            if (this.options.template) {
                li.innerHTML = this.options.template(item);
            } else {
                li.textContent = typeof item === 'string' ? item : item.label || item.value;
            }
            
            li.addEventListener('click', () => this.selectItem(item));
            li.addEventListener('mouseenter', () => this.setSelected(index));
            
            ul.appendChild(li);
        });
        
        this.dropdown.innerHTML = '';
        this.dropdown.appendChild(ul);
        this.open();
    }
    
    selectItem(item) {
        const value = typeof item === 'string' ? item : item.value || item.label;
        this.input.value = value;
        
        if (this.options.onSelect) {
            this.options.onSelect(item);
        }
        
        this.close();
        this.input.focus();
    }
    
    selectNext() {
        this.setSelected(Math.min(this.selectedIndex + 1, this.results.length - 1));
    }
    
    selectPrev() {
        this.setSelected(Math.max(this.selectedIndex - 1, 0));
    }
    
    setSelected(index) {
        this.selectedIndex = index;
        
        // Update UI
        const items = this.dropdown.querySelectorAll('.autocomplete-item');
        items.forEach((item, i) => {
            item.classList.toggle('selected', i === index);
            item.setAttribute('aria-selected', i === index);
        });
        
        // Scroll into view if needed
        if (items[index]) {
            items[index].scrollIntoView({ block: 'nearest' });
        }
    }
    
    open() {
        this.isOpen = true;
        this.dropdown.style.display = 'block';
        this.input.setAttribute('aria-expanded', 'true');
        
        // Position dropdown
        const rect = this.input.getBoundingClientRect();
        this.dropdown.style.width = `${rect.width}px`;
    }
    
    close() {
        this.isOpen = false;
        this.dropdown.style.display = 'none';
        this.input.setAttribute('aria-expanded', 'false');
        this.selectedIndex = -1;
    }
}

// Smart form field suggestions based on field type
class SmartFieldSuggestions {
    constructor() {
        this.patterns = {
            phone: {
                pattern: /^[\d-]*$/,
                format: (value) => {
                    const numbers = value.replace(/\D/g, '');
                    if (numbers.length <= 3) return numbers;
                    if (numbers.length <= 6) return `${numbers.slice(0, 3)}-${numbers.slice(3)}`;
                    return `${numbers.slice(0, 3)}-${numbers.slice(3, 6)}-${numbers.slice(6, 10)}`;
                },
                placeholder: '123-456-7890'
            },
            
            date: {
                pattern: /^[\d-\/]*$/,
                format: (value) => {
                    const numbers = value.replace(/\D/g, '');
                    if (numbers.length <= 2) return numbers;
                    if (numbers.length <= 4) return `${numbers.slice(0, 2)}/${numbers.slice(2)}`;
                    return `${numbers.slice(0, 2)}/${numbers.slice(2, 4)}/${numbers.slice(4, 8)}`;
                },
                placeholder: 'MM/DD/YYYY'
            },
            
            time: {
                pattern: /^[\d:]*$/,
                format: (value) => {
                    const numbers = value.replace(/\D/g, '');
                    if (numbers.length <= 2) return numbers;
                    return `${numbers.slice(0, 2)}:${numbers.slice(2, 4)}`;
                },
                placeholder: 'HH:MM'
            },
            
            busId: {
                pattern: /^[A-Z0-9-]*$/,
                format: (value) => value.toUpperCase().replace(/[^A-Z0-9-]/g, ''),
                suggestions: ['BUS-', 'MINI-', 'VAN-'],
                placeholder: 'BUS-001'
            },
            
            routeId: {
                pattern: /^[A-Z0-9-]*$/,
                format: (value) => value.toUpperCase().replace(/[^A-Z0-9-]/g, ''),
                suggestions: ['ROUTE-', 'AM-', 'PM-', 'SPECIAL-'],
                placeholder: 'ROUTE-01'
            }
        };
    }
    
    enhance(fieldId, type) {
        const field = document.getElementById(fieldId);
        if (!field || !this.patterns[type]) return;
        
        const pattern = this.patterns[type];
        
        // Set placeholder
        if (pattern.placeholder && !field.placeholder) {
            field.placeholder = pattern.placeholder;
        }
        
        // Add input formatting
        field.addEventListener('input', (e) => {
            let value = e.target.value;
            
            // Apply pattern validation
            if (pattern.pattern && !pattern.pattern.test(value)) {
                e.target.value = value.slice(0, -1);
                return;
            }
            
            // Apply formatting
            if (pattern.format) {
                const cursorPos = e.target.selectionStart;
                const formatted = pattern.format(value);
                e.target.value = formatted;
                
                // Maintain cursor position
                const diff = formatted.length - value.length;
                e.target.setSelectionRange(cursorPos + diff, cursorPos + diff);
            }
        });
        
        // Add suggestions if available
        if (pattern.suggestions) {
            new AutoComplete(fieldId, {
                source: pattern.suggestions,
                minChars: 0,
                onSelect: (value) => {
                    field.value = value;
                    field.focus();
                    field.setSelectionRange(value.length, value.length);
                }
            });
        }
    }
}

// Data validation helper
class DataValidator {
    static validate(value, rules) {
        const errors = [];
        
        for (const rule of rules) {
            switch (rule.type) {
                case 'required':
                    if (!value || value.trim() === '') {
                        errors.push(rule.message || 'This field is required');
                    }
                    break;
                    
                case 'email':
                    if (value && !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value)) {
                        errors.push(rule.message || 'Invalid email address');
                    }
                    break;
                    
                case 'phone':
                    if (value && !/^\d{3}-\d{3}-\d{4}$/.test(value)) {
                        errors.push(rule.message || 'Phone must be in format: 123-456-7890');
                    }
                    break;
                    
                case 'date':
                    if (value) {
                        const date = new Date(value);
                        if (isNaN(date.getTime())) {
                            errors.push(rule.message || 'Invalid date');
                        }
                    }
                    break;
                    
                case 'min':
                    if (value && parseFloat(value) < rule.value) {
                        errors.push(rule.message || `Minimum value is ${rule.value}`);
                    }
                    break;
                    
                case 'max':
                    if (value && parseFloat(value) > rule.value) {
                        errors.push(rule.message || `Maximum value is ${rule.value}`);
                    }
                    break;
                    
                case 'pattern':
                    if (value && !rule.value.test(value)) {
                        errors.push(rule.message || 'Invalid format');
                    }
                    break;
            }
        }
        
        return errors;
    }
}

// Export for use
window.AutoComplete = AutoComplete;
window.SmartFieldSuggestions = SmartFieldSuggestions;
window.DataValidator = DataValidator;