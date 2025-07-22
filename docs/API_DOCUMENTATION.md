# Fleet Management System API Documentation

## Table of Contents
1. [Authentication](#authentication)
2. [Common Headers](#common-headers)
3. [Error Responses](#error-responses)
4. [Public Endpoints](#public-endpoints)
5. [Dashboard API](#dashboard-api)
6. [Fleet Management API](#fleet-management-api)
7. [Route Management API](#route-management-api)
8. [Student Management API](#student-management-api)
9. [Maintenance API](#maintenance-api)
10. [Analytics API](#analytics-api)
11. [Import/Export API](#importexport-api)
12. [Reporting API](#reporting-api)

---

## Authentication

The API uses session-based authentication with CSRF protection. All authenticated requests must include:
- A valid session cookie (`session_id`)
- CSRF token for POST/PUT/DELETE requests

### Login
```http
POST /
Content-Type: application/x-www-form-urlencoded

username=myuser&password=mypassword
```

**Response:**
- Success: 303 redirect to appropriate dashboard
- Failure: 200 with error message in HTML

### Logout
```http
POST /logout
```

**Response:** 303 redirect to login page

---

## Common Headers

### Request Headers
```http
Cookie: session_id=<session_token>
Content-Type: application/json (for JSON endpoints)
X-CSRF-Token: <csrf_token> (for state-changing operations)
```

### Response Headers
```http
Content-Type: application/json
Content-Encoding: gzip (if compression enabled)
Cache-Control: no-cache, no-store, must-revalidate
```

---

## Error Responses

All API errors follow this format:

```json
{
  "error": {
    "code": "ERROR_CODE",
    "message": "Human-readable error message",
    "details": {} // Optional additional details
  }
}
```

### Common Error Codes
- `UNAUTHORIZED` - Not authenticated
- `FORBIDDEN` - Insufficient permissions
- `VALIDATION_ERROR` - Invalid input
- `NOT_FOUND` - Resource not found
- `DATABASE_ERROR` - Database operation failed
- `INTERNAL_ERROR` - Server error

---

## Public Endpoints

### Health Check
```http
GET /health
```

**Response:**
```json
{
  "status": "healthy",
  "service": "fleet-management",
  "timestamp": "2025-01-18T10:00:00Z",
  "database": "connected",
  "cache": "active",
  "session_store": "database"
}
```

---

## Dashboard API

### Dashboard Analytics
```http
GET /api/dashboard/analytics
```

**Required Role:** Manager

**Response:**
```json
{
  "fleet": {
    "total_buses": 45,
    "active_buses": 38,
    "maintenance_buses": 5,
    "out_of_service": 2,
    "utilization_rate": 84.4
  },
  "routes": {
    "total_routes": 25,
    "active_routes": 23,
    "students_transported": 1250,
    "avg_route_efficiency": 92.5
  },
  "mileage": {
    "total_miles_today": 2340,
    "total_miles_month": 45600,
    "avg_miles_per_bus": 120.5,
    "fuel_efficiency": 6.8
  },
  "maintenance": {
    "overdue_count": 3,
    "due_soon_count": 8,
    "completed_this_month": 15,
    "avg_maintenance_cost": 250.50
  },
  "drivers": {
    "total_drivers": 50,
    "active_today": 45,
    "avg_performance_score": 88.5
  },
  "trends": {
    "mileage_trend": [/* daily data points */],
    "maintenance_trend": [/* monthly data points */],
    "cost_trend": [/* monthly data points */]
  }
}
```

### Fleet Status Widget
```http
GET /api/dashboard/fleet-status
```

**Response:**
```json
{
  "summary": {
    "total": 45,
    "active": 38,
    "maintenance": 5,
    "out_of_service": 2
  },
  "buses": [
    {
      "bus_id": "BUS001",
      "status": "active",
      "current_route": "ROUTE_A",
      "driver": "John Doe",
      "last_location": "North Elementary"
    }
  ]
}
```

### Maintenance Alerts Widget
```http
GET /api/dashboard/maintenance-alerts
```

**Response:**
```json
{
  "alerts": [
    {
      "id": 1,
      "vehicle_id": "BUS001",
      "vehicle_type": "bus",
      "alert_type": "maintenance_due",
      "category": "oil_change",
      "message": "Oil change due at 25000 miles",
      "severity": "warning",
      "due_mileage": 25000,
      "current_miles": 24750,
      "created_at": "2025-01-18T08:00:00Z"
    }
  ],
  "summary": {
    "critical": 2,
    "warning": 8,
    "info": 5
  }
}
```

### Route Efficiency Widget
```http
GET /api/dashboard/route-efficiency
```

**Response:**
```json
{
  "routes": [
    {
      "route_id": "ROUTE001",
      "route_name": "North Elementary",
      "efficiency_score": 95.5,
      "on_time_percentage": 98.0,
      "capacity_utilization": 85.0,
      "avg_trip_duration": 45
    }
  ]
}
```

---

## Fleet Management API

### Update Vehicle Status
```http
POST /api/update-vehicle-status
Content-Type: application/json

{
  "vehicle_id": "BUS001",
  "vehicle_type": "bus",
  "field_name": "status",
  "field_value": "maintenance",
  "csrf_token": "<csrf_token>"
}
```

**Response:**
```json
{
  "success": true,
  "message": "Vehicle status updated successfully"
}
```

### Add New Bus
```http
POST /add-bus
Content-Type: application/json

{
  "bus_id": "BUS100",
  "model": "Blue Bird Vision",
  "capacity": 72,
  "status": "active",
  "oil_status": "good",
  "tire_status": "good",
  "maintenance_notes": "",
  "csrf_token": "<csrf_token>"
}
```

**Required Role:** Manager

**Response:**
```json
{
  "success": true,
  "message": "Bus added successfully",
  "bus_id": "BUS100"
}
```

### Check Maintenance Due
```http
GET /api/check-maintenance?vehicle_id=BUS001&vehicle_type=bus&current_mileage=24500
```

**Response:**
```json
{
  "maintenance_due": [
    {
      "category": "oil_change",
      "due_at": 25000,
      "overdue": false,
      "last_service": 20000
    },
    {
      "category": "tire_rotation",
      "due_at": 30000,
      "overdue": false,
      "last_service": 20000
    }
  ]
}
```

---

## Route Management API

### Assign Route
```http
POST /assign-route
Content-Type: application/x-www-form-urlencoded

driver=johndoe&bus_id=BUS001&route_id=ROUTE001&csrf_token=<token>
```

**Required Role:** Manager

**Response:**
- Success: 303 redirect with success message
- Failure: 200 with error message

### Unassign Route
```http
POST /unassign-route
Content-Type: application/x-www-form-urlencoded

driver=johndoe&bus_id=BUS001&route_id=ROUTE001&csrf_token=<token>
```

**Required Role:** Manager

### Add Route
```http
POST /add-route
Content-Type: application/x-www-form-urlencoded

route_id=ROUTE100&route_name=New+Route&description=Test+route&csrf_token=<token>
```

**Required Role:** Manager

---

## Student Management API

### Add Student
```http
POST /add-student
Content-Type: application/x-www-form-urlencoded

student_id=STU001&name=John+Doe&phone_number=555-1234&guardian=Jane+Doe&pickup_time=07:30&dropoff_time=15:30&position_number=1&csrf_token=<token>
```

**Required Role:** Driver or Manager

### Edit Student
```http
POST /edit-student
Content-Type: application/x-www-form-urlencoded

student_id=STU001&name=John+Doe+Jr&phone_number=555-5678&csrf_token=<token>
```

### Remove Student
```http
POST /remove-student
Content-Type: application/x-www-form-urlencoded

student_id=STU001&csrf_token=<token>
```

---

## Maintenance API

### Save Maintenance Record
```http
POST /save-maintenance-record
Content-Type: application/x-www-form-urlencoded

vehicle_id=BUS001&vehicle_type=bus&category=oil_change&notes=Regular+oil+change&mileage=25000&cost=150.00&date=2025-01-18&csrf_token=<token>
```

**Response:**
- Success: 303 redirect to maintenance history
- Failure: 200 with error message

### Debug Maintenance Records
```http
GET /api/debug-maintenance?vehicle_id=BUS001&vehicle_type=bus
```

**Required Role:** Manager

**Response:**
```json
{
  "vehicle": {
    "id": "BUS001",
    "type": "bus",
    "current_mileage": 24500
  },
  "maintenance_records": [
    {
      "id": 1,
      "category": "oil_change",
      "date": "2025-01-01",
      "mileage": 20000,
      "cost": 150.00,
      "notes": "Regular oil change"
    }
  ],
  "alerts": []
}
```

---

## Analytics API

### Comparative Analytics
```http
GET /api/analytics/comparison?period=month
```

**Parameters:**
- `period`: `month` or `year`

**Required Role:** Manager

**Response:**
```json
{
  "period": "month",
  "current_period": {
    "start_date": "2025-01-01",
    "end_date": "2025-01-31",
    "metrics": {
      "total_miles": 45600,
      "total_trips": 2300,
      "maintenance_costs": 3500.00,
      "fuel_costs": 8900.00,
      "active_buses": 38,
      "students_transported": 35000
    }
  },
  "previous_period": {
    "start_date": "2024-12-01",
    "end_date": "2024-12-31",
    "metrics": {/* same structure */}
  },
  "comparison": {
    "total_miles": {
      "current": 45600,
      "previous": 43200,
      "change": 2400,
      "change_percent": 5.6,
      "trend": "up"
    }
    // ... other metrics
  }
}
```

### Trend Analysis
```http
GET /api/analytics/trend?metric=mileage&period=6months
```

**Parameters:**
- `metric`: `mileage`, `costs`, `maintenance`, `fuel_efficiency`
- `period`: `3months`, `6months`, `1year`

**Response:**
```json
{
  "metric": "mileage",
  "period": "6months",
  "data_points": [
    {
      "date": "2024-08-01",
      "value": 42000,
      "label": "August 2024"
    }
    // ... more data points
  ],
  "summary": {
    "total": 265000,
    "average": 44166,
    "min": 41000,
    "max": 48000,
    "trend": "increasing"
  }
}
```

---

## Import/Export API

### Export Students
```http
GET /api/export/students?format=csv
```

**Parameters:**
- `format`: `csv` or `excel`
- `active_only`: `true` or `false` (optional)

**Response:** File download with appropriate content type

### Export Fleet Data
```http
GET /api/export/fleet?format=excel
```

**Response:** Excel file with fleet inventory

### Import Preview
```http
POST /api/import/preview
Content-Type: multipart/form-data

file=<file>&type=students&csrf_token=<token>
```

**Response:**
```json
{
  "preview": {
    "total_rows": 100,
    "valid_rows": 95,
    "invalid_rows": 5,
    "columns_detected": ["student_id", "name", "phone", "guardian"],
    "sample_data": [
      // First 5 rows
    ],
    "validation_errors": [
      {
        "row": 15,
        "errors": ["Invalid phone number format"]
      }
    ]
  }
}
```

### Import Data
```http
POST /api/import/process
Content-Type: application/json

{
  "import_id": "IMP_20250118_001",
  "type": "students",
  "column_mapping": {
    "A": "student_id",
    "B": "name",
    "C": "phone_number"
  },
  "options": {
    "update_existing": true,
    "skip_errors": false
  },
  "csrf_token": "<token>"
}
```

**Response:**
```json
{
  "success": true,
  "summary": {
    "total_processed": 100,
    "imported": 95,
    "updated": 20,
    "skipped": 5,
    "errors": 0
  },
  "import_id": "IMP_20250118_001"
}
```

---

## Reporting API

### Generate Report
```http
POST /api/report-builder
Content-Type: application/json

{
  "name": "Monthly Fleet Report",
  "description": "Fleet status and maintenance for January",
  "data_source": "fleet_overview",
  "filters": {
    "date_range": {
      "start": "2025-01-01",
      "end": "2025-01-31"
    },
    "status": ["active", "maintenance"]
  },
  "columns": ["bus_id", "status", "total_miles", "maintenance_costs"],
  "grouping": "status",
  "sorting": {
    "column": "total_miles",
    "direction": "desc"
  },
  "aggregations": [
    {
      "column": "total_miles",
      "function": "sum",
      "alias": "total_fleet_miles"
    }
  ],
  "chart_config": {
    "type": "bar",
    "x_axis": "bus_id",
    "y_axis": "total_miles"
  },
  "csrf_token": "<token>"
}
```

**Response:**
```json
{
  "report_id": "RPT_20250118_001",
  "data": {
    "rows": [/* report data */],
    "totals": {
      "total_fleet_miles": 45600
    }
  },
  "chart_data": {
    "labels": ["BUS001", "BUS002"],
    "datasets": [{
      "label": "Total Miles",
      "data": [1200, 1350]
    }]
  }
}
```

### Get Report Data Sources
```http
GET /api/report-data-sources
```

**Response:**
```json
{
  "data_sources": [
    {
      "id": "fleet_overview",
      "name": "Fleet Overview",
      "description": "Complete fleet inventory with status",
      "available_columns": [
        {
          "name": "bus_id",
          "type": "string",
          "label": "Bus ID"
        },
        {
          "name": "total_miles",
          "type": "number",
          "label": "Total Miles"
        }
      ]
    }
  ]
}
```

### Schedule Report Export
```http
POST /api/scheduled-exports
Content-Type: application/json

{
  "name": "Weekly Fleet Report",
  "export_type": "fleet_summary",
  "format": "excel",
  "frequency": "weekly",
  "day_of_week": 1,
  "time": "08:00",
  "email_recipients": ["manager@school.edu"],
  "enabled": true,
  "csrf_token": "<token>"
}
```

**Response:**
```json
{
  "success": true,
  "export_id": "EXP_20250118_001",
  "next_run": "2025-01-25T08:00:00Z"
}
```

---

## Pagination

List endpoints support pagination with these parameters:
- `page`: Page number (default: 1)
- `per_page`: Items per page (default: 20, max: 100)

**Example:**
```http
GET /api/students?page=2&per_page=50
```

**Paginated Response:**
```json
{
  "data": [/* items */],
  "pagination": {
    "page": 2,
    "per_page": 50,
    "total": 245,
    "total_pages": 5,
    "has_prev": true,
    "has_next": true
  }
}
```

---

## Rate Limiting

API endpoints are rate-limited to prevent abuse:
- Authentication endpoints: 20 requests per 15 minutes per IP
- API endpoints: 100 requests per minute per user
- Import/Export endpoints: 10 requests per hour per user

Rate limit headers:
```http
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1737205200
```

---

## Webhooks (Future)

Webhook support is planned for:
- Maintenance alerts
- Route completion notifications
- Import/export completion
- System alerts

---

## API Versioning

Currently, all endpoints are unversioned (v1 implied). Future versions will use URL versioning:
- Current: `/api/dashboard/analytics`
- Future: `/api/v2/dashboard/analytics`

---

## CORS Policy

CORS is configured for:
- Allowed origins: Configured domains only
- Allowed methods: GET, POST, PUT, DELETE, OPTIONS
- Allowed headers: Content-Type, X-CSRF-Token
- Credentials: Supported

---

## Best Practices

1. **Always include CSRF tokens** for state-changing operations
2. **Use appropriate HTTP methods** (GET for reads, POST for writes)
3. **Handle errors gracefully** - check error responses
4. **Implement exponential backoff** for retries
5. **Cache responses** where appropriate
6. **Compress request bodies** for large payloads
7. **Use pagination** for list endpoints
8. **Include correlation IDs** in requests for debugging

---

## Support

For API support:
- Documentation: This document
- Issues: GitHub repository issues
- Email: support@fleetmanagement.com

---

Last Updated: January 18, 2025
API Version: 1.0