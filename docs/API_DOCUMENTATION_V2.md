# Fleet Management System API Documentation

## Overview

The Fleet Management System provides a comprehensive REST API for managing school transportation operations. All API endpoints require authentication unless otherwise specified.

## Base URL

```
http://localhost:5003
```

## Authentication

The system uses session-based authentication. To authenticate:

1. Login via POST to `/` with username and password
2. Session cookie will be set automatically
3. Include session cookie in all subsequent requests

### Login

```http
POST /
Content-Type: application/x-www-form-urlencoded

username=admin&password=Headstart1
```

Response: Redirect to appropriate dashboard

## API Endpoints

### Health & Status

#### System Health Check
```http
GET /health
```

Response:
```json
{
  "status": "ok",
  "timestamp": "2024-01-20T10:30:00Z",
  "services": {
    "database": "healthy",
    "cache": "ok",
    "sessions": "ok"
  },
  "recovery": {
    "healthy": true,
    "last_check": "2024-01-20T10:29:30Z",
    "recovery_attempts": 0
  }
}
```

#### Server Status
```http
GET /status
```

Returns detailed server status including version, uptime, and system metrics.

### Dashboard APIs

#### Dashboard Statistics
```http
GET /api/dashboard/stats
Authorization: Required (Manager)
```

Response:
```json
{
  "total_buses": 10,
  "active_buses": 8,
  "total_students": 150,
  "active_routes": 5,
  "maintenance_due": 3,
  "fuel_cost_mtd": 2500.50
}
```

#### Fleet Status Overview
```http
GET /api/fleet-status
Authorization: Required
```

Response:
```json
{
  "buses": [
    {
      "bus_id": "BUS-001",
      "status": "active",
      "current_route": "RT-NORTH-01",
      "driver": "john.doe",
      "oil_status": "good",
      "tire_status": "due"
    }
  ],
  "summary": {
    "total": 10,
    "active": 8,
    "maintenance": 1,
    "out_of_service": 1
  }
}
```

### Fleet Management

#### Get All Buses
```http
GET /api/buses
Authorization: Required
```

Query Parameters:
- `status` - Filter by status (active, maintenance, out_of_service)
- `page` - Page number (default: 1)
- `per_page` - Items per page (default: 20)

#### Get Single Bus
```http
GET /api/bus/{bus_id}
Authorization: Required
```

#### Update Bus
```http
PUT /api/bus/{bus_id}
Authorization: Required (Manager)
Content-Type: application/json
```

Request Body:
```json
{
  "status": "maintenance",
  "oil_status": "overdue",
  "tire_status": "good",
  "maintenance_notes": "Oil change required"
}
```

#### Add New Bus
```http
POST /api/buses
Authorization: Required (Manager)
Content-Type: application/json
```

Request Body:
```json
{
  "bus_id": "BUS-021",
  "model": "BLUEBIRD VISION",
  "capacity": 65,
  "status": "active"
}
```

### Maintenance Management

#### Get Maintenance Records
```http
GET /api/maintenance-records
Authorization: Required
```

Query Parameters:
- `vehicle_id` - Filter by vehicle
- `start_date` - Start date (YYYY-MM-DD)
- `end_date` - End date (YYYY-MM-DD)
- `page` - Page number
- `per_page` - Items per page

#### Add Maintenance Record
```http
POST /api/maintenance-records
Authorization: Required (Manager)
Content-Type: application/json
```

Request Body:
```json
{
  "vehicle_id": "BUS-001",
  "service_date": "2024-01-15",
  "mileage": 125000,
  "cost": 350.00,
  "work_description": "Oil change and filter replacement",
  "po_number": "PO-2024-001"
}
```

### Student Management

#### Get Students
```http
GET /api/students
Authorization: Required
```

Query Parameters:
- `route_id` - Filter by route
- `grade` - Filter by grade
- `active` - Filter by active status (true/false)
- `search` - Search by name

#### Add Student
```http
POST /api/students
Authorization: Required (Driver/Manager)
Content-Type: application/json
```

Request Body:
```json
{
  "first_name": "Jane",
  "last_name": "Smith",
  "grade": "5",
  "address": "123 Main St",
  "route_id": "RT-NORTH-01",
  "parent_name": "John Smith",
  "parent_phone": "555-1234",
  "emergency_contact": "555-5678"
}
```

#### Update Student
```http
PUT /api/student/{student_id}
Authorization: Required (Driver/Manager)
```

### Route Management

#### Get All Routes
```http
GET /api/routes
Authorization: Required
```

#### Get Route Assignments
```http
GET /api/route-assignments
Authorization: Required
```

#### Create Route Assignment
```http
POST /api/route-assignments
Authorization: Required (Manager)
Content-Type: application/json
```

Request Body:
```json
{
  "route_id": "RT-NORTH-01",
  "bus_id": "BUS-001",
  "driver_username": "john.doe",
  "effective_date": "2024-01-20"
}
```

### Fuel Management

#### Get Fuel Records
```http
GET /api/fuel-records
Authorization: Required
```

Query Parameters:
- `vehicle_id` - Filter by vehicle
- `start_date` - Start date
- `end_date` - End date

#### Add Fuel Record
```http
POST /api/fuel-records
Authorization: Required
Content-Type: application/json
```

Request Body:
```json
{
  "vehicle_id": "BUS-001",
  "fuel_date": "2024-01-20",
  "gallons": 45.5,
  "price_per_gallon": 3.29,
  "total_cost": 149.70,
  "odometer_reading": 125500,
  "location": "Fleet Fuel Station"
}
```

### Reporting

#### Generate Mileage Report
```http
GET /api/reports/mileage
Authorization: Required (Manager)
```

Query Parameters:
- `bus_id` - Specific bus (optional)
- `month` - Month (1-12)
- `year` - Year (YYYY)

#### Export Data
```http
GET /api/export/{type}
Authorization: Required (Manager)
```

Types:
- `students` - Export student roster
- `fleet` - Export fleet inventory
- `maintenance` - Export maintenance records
- `fuel` - Export fuel records

Format Options (query parameter):
- `format=csv` (default)
- `format=xlsx`
- `format=pdf`

### ECSE Management

#### Get ECSE Students
```http
GET /api/ecse/students
Authorization: Required (Manager)
```

#### Get ECSE Student Details
```http
GET /api/ecse/student/{student_id}
Authorization: Required (Manager)
```

#### Update ECSE Student
```http
PUT /api/ecse/student/{student_id}
Authorization: Required (Manager)
```

### User Management

#### Get Users
```http
GET /api/users
Authorization: Required (Manager)
```

#### Create User
```http
POST /api/users
Authorization: Required (Manager)
Content-Type: application/json
```

Request Body:
```json
{
  "username": "new.driver",
  "email": "driver@school.com",
  "password": "securepassword",
  "role": "driver",
  "full_name": "New Driver"
}
```

#### Update User
```http
PUT /api/user/{user_id}
Authorization: Required (Manager)
```

#### Delete User
```http
DELETE /api/user/{user_id}
Authorization: Required (Manager)
```

### Monitoring & Recovery

#### System Metrics
```http
GET /api/monitoring/metrics
Authorization: Required (Manager)
```

Returns real-time system metrics including database performance, memory usage, and request rates.

#### System Alerts
```http
GET /api/monitoring/alerts
Authorization: Required (Manager)
```

Returns current system alerts and warnings.

#### Trigger Recovery
```http
POST /api/recovery
Authorization: Required (Manager)
```

Manually triggers system recovery procedures.

## Error Responses

All error responses follow this format:

```json
{
  "error": {
    "code": "ERROR_CODE",
    "message": "Human readable error message",
    "details": {
      "field": "Additional context"
    }
  },
  "timestamp": "2024-01-20T10:30:00Z"
}
```

Common HTTP Status Codes:
- `200 OK` - Request successful
- `201 Created` - Resource created
- `400 Bad Request` - Invalid request data
- `401 Unauthorized` - Authentication required
- `403 Forbidden` - Insufficient permissions
- `404 Not Found` - Resource not found
- `409 Conflict` - Resource conflict
- `422 Unprocessable Entity` - Validation error
- `500 Internal Server Error` - Server error

## Rate Limiting

API endpoints are rate limited to prevent abuse:
- 100 requests per minute per IP for authenticated users
- 20 requests per minute per IP for unauthenticated endpoints
- Login attempts limited to 5 per minute

Rate limit headers:
- `X-RateLimit-Limit` - Request limit
- `X-RateLimit-Remaining` - Remaining requests
- `X-RateLimit-Reset` - Reset timestamp

## Pagination

List endpoints support pagination using these query parameters:
- `page` - Page number (default: 1)
- `per_page` - Items per page (default: 20, max: 100)

Pagination metadata is included in responses:
```json
{
  "data": [...],
  "pagination": {
    "page": 1,
    "per_page": 20,
    "total_items": 150,
    "total_pages": 8,
    "has_next": true,
    "has_prev": false
  }
}
```

## Webhooks (Future)

The system will support webhooks for real-time notifications:
- Maintenance alerts
- Route changes
- System events

## API Versioning

The API uses URL-based versioning:
- Current version: `/api/v1/`
- Legacy endpoints: `/api/` (deprecated)

Version information is included in response headers:
- `X-API-Version: 1.0`

## SDK Support

Official SDKs are planned for:
- JavaScript/TypeScript
- Python
- Go

## Support

For API support:
- Documentation: https://docs.fleetmanagement.com/api
- Issues: https://github.com/fleetmanagement/api/issues
- Email: api-support@fleetmanagement.com