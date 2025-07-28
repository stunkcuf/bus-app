# Lazy Loading Implementation Guide

## Overview

The lazy loading system provides efficient data loading for large datasets by loading content incrementally as users scroll. This improves initial page load times and reduces server load.

## Features

- **Infinite Scrolling**: Automatically loads more data as users scroll
- **Search Integration**: Real-time search with debouncing
- **Filter Support**: Multiple filter options per dataset
- **Performance Optimized**: Only loads visible data
- **Mobile Responsive**: Works seamlessly on all devices
- **Error Handling**: Graceful error recovery
- **Loading Indicators**: Visual feedback during data loading

## Available Endpoints

### 1. Students (`/api/lazy/students`)
- **Filters**: status (active/inactive), driver, search
- **Default page size**: 25 items
- **Sorting**: By position_number, name

### 2. Buses (`/api/lazy/buses`)
- **Filters**: status, search (bus_id, model)
- **Default page size**: 25 items
- **Sorting**: By bus_id

### 3. Driver Logs (`/api/lazy/driver-logs`)
- **Filters**: driver, bus_id, date_from, date_to, period
- **Default page size**: 25 items
- **Sorting**: By date DESC, created_at DESC

### 4. Maintenance Records (`/api/lazy/maintenance-records`)
- **Filters**: vehicle_id, category
- **Default page size**: 25 items
- **Manager access only**

### 5. Fleet Vehicles (`/api/lazy/fleet-vehicles`)
- **Filters**: status, make
- **Default page size**: 25 items
- **Manager access only**

### 6. Monthly Mileage Reports (`/api/lazy/monthly-mileage-reports`)
- **Filters**: year, month, bus_id
- **Default page size**: 25 items
- **Manager access only**

## Usage Examples

### Basic HTML Setup

```html
<!-- Include CSS and JavaScript -->
<link rel="stylesheet" href="/static/lazy_loading.css">
<script src="/static/lazy_loading.js"></script>

<!-- Container with data attributes -->
<div class="lazy-load-container" 
     id="students-table"
     data-lazy-load="true"
     data-lazy-load-endpoint="/api/lazy/students"
     data-lazy-load-type="student"
     data-lazy-load-per-page="25">
    <table class="table">
        <thead>
            <tr>
                <th>Name</th>
                <th>Guardian</th>
                <th>Status</th>
            </tr>
        </thead>
        <tbody>
            <!-- Items loaded here -->
        </tbody>
    </table>
</div>
```

### With Search and Filters

```html
<!-- Search input -->
<input type="text" id="search-students" placeholder="Search...">

<!-- Filter select -->
<select id="status-filter">
    <option value="">All</option>
    <option value="active">Active</option>
    <option value="inactive">Inactive</option>
</select>

<!-- Container with search and filter configuration -->
<div class="lazy-load-container" 
     id="students-table"
     data-lazy-load="true"
     data-lazy-load-endpoint="/api/lazy/students"
     data-lazy-load-type="student"
     data-lazy-load-search="#search-students"
     data-lazy-load-filters='[{"name": "status", "selector": "#status-filter"}]'>
    <!-- Table content -->
</div>
```

### JavaScript API

```javascript
// Manual initialization
const loader = new LazyLoader({
    container: '#students-table',
    endpoint: '/api/lazy/students',
    perPage: 50,
    itemTemplate: (student) => {
        return `
            <tr>
                <td>${student.name}</td>
                <td>${student.guardian}</td>
                <td>${student.active ? 'Active' : 'Inactive'}</td>
            </tr>
        `;
    },
    searchInput: '#search-students',
    filterInputs: [
        { name: 'status', selector: '#status-filter' },
        { name: 'driver', selector: '#driver-filter' }
    ]
});

// Reset and reload
loader.reset();
loader.loadMore();

// Access loaded items
console.log(loader.items);
console.log(loader.totalItems);
```

## API Response Format

All lazy loading endpoints return data in this format:

```json
{
    "data": [...],          // Array of items
    "page": 1,              // Current page number
    "per_page": 25,         // Items per page
    "total": 150,           // Total items available
    "total_pages": 6,       // Total number of pages
    "has_next": true,       // Whether more pages exist
    "has_previous": false,  // Whether previous pages exist
    "load_time": "12.5ms"   // Server processing time
}
```

## Query Parameters

All endpoints accept these query parameters:

- `page` - Page number (default: 1)
- `per_page` - Items per page (default: 25, max: 100)
- Additional filters specific to each endpoint

## Performance Considerations

1. **Initial Load**: Only loads the first page (25 items by default)
2. **Scroll Trigger**: New data loads when user scrolls within 100px of bottom
3. **Debounced Search**: 300ms delay on search input to reduce API calls
4. **Caching**: Browser caches responses for better performance
5. **Connection Pooling**: Efficient database queries with proper indexing

## Customization

### Custom Templates

Create custom item templates for different data types:

```javascript
window.LazyLoadTemplates.customType = (item) => {
    return `
        <tr>
            <td>${item.field1}</td>
            <td>${item.field2}</td>
            <td>
                <button onclick="handleAction(${item.id})">
                    Action
                </button>
            </td>
        </tr>
    `;
};
```

### Event Handling

Listen for lazy loading events:

```javascript
document.addEventListener('lazyLoadItemsAdded', function(e) {
    // Re-initialize tooltips, click handlers, etc.
    $('[data-toggle="tooltip"]').tooltip();
});
```

### Error Handling

Handle loading errors gracefully:

```javascript
loader.showError = function(message) {
    // Custom error display
    showNotification(message, 'error');
};
```

## Browser Support

- Chrome 60+
- Firefox 55+
- Safari 11+
- Edge 79+
- Mobile browsers (iOS Safari, Chrome Android)

## Troubleshooting

### Common Issues

1. **No data loading**: Check browser console for API errors
2. **Infinite loading**: Verify endpoint returns proper pagination metadata
3. **Search not working**: Ensure search input selector is correct
4. **Filters not applying**: Check filter name matches API parameter

### Debug Mode

Enable debug logging:

```javascript
// In browser console
localStorage.setItem('lazyLoadDebug', 'true');
```

## Future Enhancements

1. Virtual scrolling for extremely large datasets
2. Bi-directional infinite scrolling
3. Offline support with service workers
4. Advanced caching strategies
5. Bulk operations on loaded items