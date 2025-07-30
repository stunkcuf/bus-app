# Fleet Management System - API Usage Examples

This document provides practical examples of using the Fleet Management System API endpoints. All examples use standard HTTP requests that can be executed using curl, Postman, or any HTTP client.

## Table of Contents

1. [Authentication](#authentication)
2. [Fleet Management](#fleet-management)
3. [Student Management](#student-management)
4. [Route Management](#route-management)
5. [Maintenance Records](#maintenance-records)
6. [Reporting](#reporting)
7. [Search and Autocomplete](#search-and-autocomplete)
8. [Import/Export](#importexport)
9. [Progress Tracking](#progress-tracking)
10. [Error Handling](#error-handling)

## Prerequisites

- Base URL: `https://your-domain.com` (replace with actual domain)
- Valid session cookie from login
- CSRF token from session

## Authentication

### Login
```bash
curl -X POST https://your-domain.com/login \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "username=johndoe&password=yourpassword" \
  -c cookies.txt
```

**Response:**
- Success: Redirects to dashboard with session cookie
- Failure: Returns to login page with error

### Logout
```bash
curl -X GET https://your-domain.com/logout \
  -b cookies.txt
```

## Fleet Management

### Get All Vehicles
```bash
curl -X GET https://your-domain.com/api/vehicles \
  -H "Accept: application/json" \
  -b cookies.txt
```

**Response:**
```json
{
  "vehicles": [
    {
      "id": 1,
      "bus_number": "1001",
      "make": "BlueBird",
      "model": "Vision",
      "year": 2020,
      "capacity": 72,
      "license_plate": "ABC-1234",
      "current_mileage": 45000,
      "last_maintenance": "2025-01-15"
    }
  ],
  "total": 10
}
```

### Add New Vehicle
```bash
curl -X POST https://your-domain.com/add-bus \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -H "X-CSRF-Token: your-csrf-token" \
  -b cookies.txt \
  -d "bus_number=1015&make=BlueBird&model=Vision&year=2025&capacity=72&license_plate=XYZ-5678&vin=1HGBH41JXMN109186"
```

### Update Vehicle Mileage
```bash
curl -X POST https://your-domain.com/api/vehicle-mileage/1001 \
  -H "Accept: application/json" \
  -b cookies.txt
```

**Response:**
```json
{
  "current_mileage": 45250,
  "last_updated": "2025-01-29T10:30:00Z"
}
```

## Student Management

### Search Students
```bash
curl -X GET "https://your-domain.com/api/search-students?q=john" \
  -H "Accept: application/json" \
  -b cookies.txt
```

**Response:**
```json
{
  "students": [
    {
      "id": 123,
      "name": "John Smith",
      "grade": "5th",
      "route": "Route A",
      "address": "123 Main St",
      "guardian": "Jane Smith",
      "guardian_phone": "(555) 123-4567"
    }
  ]
}
```

### Add New Student (Wizard API)
```bash
# Step 1: Start wizard session
curl -X POST https://your-domain.com/add-student-wizard \
  -H "X-CSRF-Token: your-csrf-token" \
  -b cookies.txt

# Step 2: Submit student data
curl -X POST https://your-domain.com/add-student-wizard \
  -H "Content-Type: application/json" \
  -H "X-CSRF-Token: your-csrf-token" \
  -b cookies.txt \
  -d '{
    "first_name": "Emily",
    "last_name": "Johnson",
    "grade": "3rd",
    "address": "456 Oak Avenue",
    "guardian_name": "Robert Johnson",
    "guardian_phone": "(555) 987-6543",
    "route_id": 5,
    "pickup_time": "07:30",
    "dropoff_time": "15:45"
  }'
```

## Route Management

### Get Available Routes
```bash
curl -X GET https://your-domain.com/api/available-routes \
  -H "Accept: application/json" \
  -b cookies.txt
```

**Response:**
```json
{
  "routes": [
    {
      "id": 1,
      "name": "North Elementary",
      "stops": 15,
      "distance": 12.5,
      "estimated_time": 45,
      "assigned_driver": null,
      "assigned_bus": null
    }
  ]
}
```

### Check Route Assignment Conflicts
```bash
curl -X POST https://your-domain.com/api/check-assignment-conflicts \
  -H "Content-Type: application/json" \
  -H "X-CSRF-Token: your-csrf-token" \
  -b cookies.txt \
  -d '{
    "driver_id": 5,
    "bus_id": 1001,
    "route_id": 3,
    "period": "morning"
  }'
```

**Response:**
```json
{
  "has_conflicts": false,
  "conflicts": [],
  "suggestions": {
    "alternative_drivers": [2, 7],
    "alternative_buses": [1003, 1005]
  }
}
```

## Maintenance Records

### Get Maintenance Suggestions
```bash
curl -X GET "https://your-domain.com/api/maintenance/suggestions?vehicle_id=1001" \
  -H "Accept: application/json" \
  -b cookies.txt
```

**Response:**
```json
{
  "suggestions": [
    {
      "service_type": "Oil Change",
      "priority": "high",
      "reason": "5,000 miles since last oil change",
      "estimated_cost": 75.00
    },
    {
      "service_type": "Tire Rotation",
      "priority": "medium",
      "reason": "10,000 miles since last rotation",
      "estimated_cost": 40.00
    }
  ]
}
```

### Log Maintenance Record
```bash
curl -X POST https://your-domain.com/save-maintenance-record \
  -H "Content-Type: application/json" \
  -H "X-CSRF-Token: your-csrf-token" \
  -b cookies.txt \
  -d '{
    "vehicle_id": 1001,
    "service_date": "2025-01-29",
    "service_type": "Oil Change",
    "description": "Routine oil change - 5W-30 synthetic",
    "mileage": 45250,
    "cost": 75.00,
    "vendor": "Quick Lube Express",
    "next_service_due": "2025-04-29"
  }'
```

## Reporting

### Generate PDF Report
```bash
curl -X POST https://your-domain.com/api/reports/pdf \
  -H "Content-Type: application/json" \
  -H "X-CSRF-Token: your-csrf-token" \
  -b cookies.txt \
  -d '{
    "report_type": "fleet_summary",
    "date_range": {
      "start": "2025-01-01",
      "end": "2025-01-31"
    },
    "filters": {
      "include_maintenance": true,
      "include_mileage": true
    }
  }' \
  -o fleet_report.pdf
```

### Export Mileage Data
```bash
curl -X GET "https://your-domain.com/export-mileage?month=2025-01&format=excel" \
  -b cookies.txt \
  -o mileage_report.xlsx
```

### Custom Report Builder
```bash
curl -X POST https://your-domain.com/api/reports/custom \
  -H "Content-Type: application/json" \
  -H "X-CSRF-Token: your-csrf-token" \
  -b cookies.txt \
  -d '{
    "name": "Driver Performance Report",
    "columns": ["driver_name", "total_miles", "on_time_percentage", "safety_score"],
    "filters": {
      "date_range": "last_month",
      "min_miles": 1000
    },
    "sort_by": "safety_score",
    "sort_order": "desc"
  }'
```

## Search and Autocomplete

### Bus Search with Autocomplete
```bash
curl -X GET "https://your-domain.com/api/search-buses?q=10&limit=5" \
  -H "Accept: application/json" \
  -b cookies.txt
```

**Response:**
```json
{
  "buses": [
    {"bus_number": "1001", "id": 1},
    {"bus_number": "1002", "id": 2},
    {"bus_number": "1003", "id": 3}
  ]
}
```

### Address Autocomplete
```bash
curl -X GET "https://your-domain.com/api/search-addresses?q=123+main" \
  -H "Accept: application/json" \
  -b cookies.txt
```

**Response:**
```json
{
  "addresses": [
    "123 Main Street",
    "123 Main Avenue",
    "123 Main Boulevard"
  ]
}
```

## Import/Export

### Analyze Import File
```bash
curl -X POST https://your-domain.com/api/import/analyze \
  -H "X-CSRF-Token: your-csrf-token" \
  -b cookies.txt \
  -F "file=@students.xlsx" \
  -F "type=students"
```

**Response:**
```json
{
  "analysis": {
    "total_rows": 150,
    "valid_rows": 148,
    "errors": [
      {
        "row": 45,
        "field": "grade",
        "error": "Invalid grade format"
      }
    ],
    "columns_detected": ["name", "grade", "address", "guardian", "phone"],
    "preview": [
      ["John Smith", "5th", "123 Main St", "Jane Smith", "(555) 123-4567"]
    ]
  }
}
```

### Execute Import
```bash
curl -X POST https://your-domain.com/api/import/execute \
  -H "Content-Type: application/json" \
  -H "X-CSRF-Token: your-csrf-token" \
  -b cookies.txt \
  -d '{
    "session_id": "import_abc123",
    "mapping": {
      "name": "student_name",
      "grade": "grade_level",
      "address": "home_address"
    },
    "options": {
      "skip_duplicates": true,
      "update_existing": false
    }
  }'
```

## Progress Tracking

### Track Feature Usage
```bash
curl -X POST https://your-domain.com/api/progress \
  -H "Content-Type: application/json" \
  -H "X-CSRF-Token: your-csrf-token" \
  -b cookies.txt \
  -d '{
    "feature": "route_assignment",
    "action": "completed",
    "metadata": {
      "routes_assigned": 5,
      "time_spent": 300
    }
  }'
```

### Get User Progress
```bash
curl -X GET https://your-domain.com/api/progress \
  -H "Accept: application/json" \
  -b cookies.txt
```

**Response:**
```json
{
  "overall_progress": 75,
  "features": {
    "onboarding": 100,
    "basic_operations": 80,
    "advanced_features": 45
  },
  "recent_activities": [
    {
      "feature": "route_assignment",
      "timestamp": "2025-01-29T10:30:00Z",
      "completion_status": "completed"
    }
  ]
}
```

## Error Handling

All API endpoints return consistent error responses:

### Validation Error
```json
{
  "error": "validation_failed",
  "message": "Invalid input data",
  "details": {
    "bus_number": "Bus number already exists",
    "capacity": "Capacity must be a positive number"
  }
}
```

### Authentication Error
```json
{
  "error": "unauthorized",
  "message": "Authentication required",
  "redirect": "/login"
}
```

### Permission Error
```json
{
  "error": "forbidden",
  "message": "You don't have permission to perform this action",
  "required_role": "manager"
}
```

### Server Error
```json
{
  "error": "internal_server_error",
  "message": "An unexpected error occurred",
  "request_id": "req_abc123",
  "timestamp": "2025-01-29T10:30:00Z"
}
```

## Best Practices

1. **Always include CSRF token** for POST/PUT/DELETE requests
2. **Handle session expiration** - Re-authenticate if you receive 401
3. **Use appropriate Accept headers** - `application/json` for API responses
4. **Implement retry logic** for transient failures
5. **Rate limiting** - Maximum 100 requests per minute per session
6. **Pagination** - Use `?page=1&limit=20` for list endpoints
7. **Filtering** - Most list endpoints support query parameters for filtering

## Common Integration Patterns

### JavaScript/Fetch Example
```javascript
async function addStudent(studentData) {
  const response = await fetch('/add-student', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'X-CSRF-Token': document.querySelector('[name="csrf_token"]').value
    },
    credentials: 'same-origin',
    body: JSON.stringify(studentData)
  });

  if (!response.ok) {
    const error = await response.json();
    throw new Error(error.message);
  }

  return await response.json();
}
```

### Python Example
```python
import requests

session = requests.Session()

# Login
login_data = {'username': 'johndoe', 'password': 'password'}
session.post('https://your-domain.com/login', data=login_data)

# Get CSRF token (implement based on your method)
csrf_token = get_csrf_token(session)

# Make API call
headers = {'X-CSRF-Token': csrf_token}
response = session.get('https://your-domain.com/api/vehicles', headers=headers)
vehicles = response.json()
```

### Go Example
```go
package main

import (
    "bytes"
    "encoding/json"
    "net/http"
    "net/http/cookiejar"
)

func main() {
    // Create client with cookie jar
    jar, _ := cookiejar.New(nil)
    client := &http.Client{Jar: jar}
    
    // Login
    loginData := url.Values{
        "username": {"johndoe"},
        "password": {"password"},
    }
    client.PostForm("https://your-domain.com/login", loginData)
    
    // Make API request
    resp, err := client.Get("https://your-domain.com/api/vehicles")
    if err != nil {
        log.Fatal(err)
    }
    defer resp.Body.Close()
    
    var vehicles []Vehicle
    json.NewDecoder(resp.Body).Decode(&vehicles)
}
```

## Testing Endpoints

For testing and development, you can use the practice mode endpoints:

```bash
# Enable practice mode
curl -X POST https://your-domain.com/api/practice-data \
  -H "X-CSRF-Token: your-csrf-token" \
  -b cookies.txt \
  -d "action=enable"

# Disable practice mode
curl -X POST https://your-domain.com/api/practice-data \
  -H "X-CSRF-Token: your-csrf-token" \
  -b cookies.txt \
  -d "action=disable"
```

## Support

For API support and questions:
- Email: api-support@fleetmanagement.com
- Documentation: https://your-domain.com/api-docs
- Status Page: https://status.fleetmanagement.com