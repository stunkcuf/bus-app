# Mobile App API Documentation

## Overview
The Fleet Management System provides a comprehensive REST API for mobile applications to enable drivers and managers to perform their duties on the go.

## Authentication

### Login
```
POST /api/mobile/v1/login
Content-Type: application/json

{
  "username": "driver123",
  "password": "password",
  "device_id": "unique-device-id",
  "platform": "ios" // or "android"
}

Response:
{
  "token": "jwt-token",
  "refresh_token": "refresh-token",
  "user": {
    "username": "driver123",
    "full_name": "John Doe",
    "role": "driver",
    "permissions": ["view_routes", "submit_attendance"]
  },
  "expires_in": 3600
}
```

### Refresh Token
```
POST /api/mobile/v1/refresh
Authorization: Bearer {refresh-token}
```

## Driver Endpoints

### Get Current Route
```
GET /api/mobile/v1/driver/route
Authorization: Bearer {token}

Response:
{
  "route_id": "R001",
  "route_name": "North District Route",
  "vehicle_id": "BUS-001",
  "student_count": 25,
  "stops": [...]
}
```

### Get Student List
```
GET /api/mobile/v1/attendance/students
Authorization: Bearer {token}

Response:
{
  "route_id": "R001",
  "date": "2024-01-25",
  "students": [
    {
      "student_id": "STU001",
      "name": "Jane Smith",
      "grade": "5",
      "address": "123 Main St",
      "parent_name": "John Smith",
      "parent_phone": "555-0123",
      "attendance": {
        "status": "present",
        "boarded_at": "07:45",
        "dropped_at": null,
        "notes": ""
      }
    }
  ]
}
```

### Submit Attendance
```
POST /api/mobile/v1/driver/attendance
Authorization: Bearer {token}
Content-Type: application/json

[
  {
    "student_id": "STU001",
    "status": "present",
    "boarded_at": "07:45",
    "dropped_at": "15:30",
    "notes": ""
  },
  {
    "student_id": "STU002",
    "status": "absent",
    "notes": "Parent called - sick"
  }
]
```

### Get Attendance History
```
GET /api/mobile/v1/attendance/history?student_id=STU001&start_date=2024-01-01&end_date=2024-01-31
Authorization: Bearer {token}

Response:
{
  "student_id": "STU001",
  "start_date": "2024-01-01",
  "end_date": "2024-01-31",
  "history": [
    {
      "date": "2024-01-25",
      "status": "present",
      "boarded_at": "07:45",
      "dropped_at": "15:30",
      "notes": "",
      "recorded_by": "driver123"
    }
  ]
}
```

### Update GPS Location
```
POST /api/mobile/v1/driver/location
Authorization: Bearer {token}
Content-Type: application/json

{
  "latitude": 40.7128,
  "longitude": -74.0060,
  "speed": 25.5,
  "heading": 180,
  "accuracy": 10,
  "timestamp": "2024-01-25T08:30:00Z"
}
```

### Report Issue
```
POST /api/mobile/v1/driver/issue
Authorization: Bearer {token}
Content-Type: application/json

{
  "type": "mechanical",
  "description": "Engine making unusual noise",
  "vehicle_id": "BUS-001",
  "route_id": "R001",
  "severity": "medium",
  "location": {
    "latitude": 40.7128,
    "longitude": -74.0060
  }
}
```

### Upload Issue Photo
```
POST /api/mobile/v1/issues/upload
Authorization: Bearer {token}
Content-Type: multipart/form-data

photo: [binary file data]

Response:
{
  "status": "success",
  "file_path": "static/uploads/issues/2024-01/driver123_1706169600_photo.jpg",
  "file_size": 2048576
}
```

### Get Issue Reports
```
GET /api/mobile/v1/issues/list?status=open&limit=50
Authorization: Bearer {token}

Response:
{
  "issues": [
    {
      "issue_id": 123,
      "type": "mechanical",
      "description": "Engine making unusual noise",
      "vehicle_id": "BUS-001",
      "severity": "medium",
      "status": "open",
      "created_at": "2024-01-25T08:30:00Z",
      "location": {
        "latitude": 40.7128,
        "longitude": -74.0060
      },
      "attachments": ["static/uploads/issues/2024-01/photo1.jpg"]
    }
  ],
  "count": 1
}
```

### Update Driver Status
```
POST /api/mobile/v1/driver/status
Authorization: Bearer {token}
Content-Type: application/json

{
  "status": "on_route" // available, on_route, break, off_duty
}
```

### Submit Vehicle Inspection
```
POST /api/mobile/v1/driver/inspection
Authorization: Bearer {token}
Content-Type: application/json

{
  "vehicle_id": "BUS-001",
  "checklist": {
    "tires": "pass",
    "brakes": "pass",
    "lights": "fail",
    "mirrors": "pass"
  },
  "notes": "Left turn signal not working",
  "odometer": 45678
}
```

## Manager Endpoints (Mobile)

### Get Mobile Dashboard
```
GET /api/mobile/v1/dashboard
Authorization: Bearer {token}

Response (for managers):
{
  "fleet_overview": {
    "total_vehicles": 44,
    "active_vehicles": 40,
    "total_drivers": 25,
    "total_students": 1200,
    "open_issues": 5,
    "attendance_rate": 95.5
  },
  "issues_by_severity": {
    "low": 2,
    "medium": 2,
    "high": 1,
    "critical": 0
  },
  "recent_alerts": []
}

Response (for drivers):
{
  "current_route": {
    "route_id": "R001",
    "route_name": "North District",
    "student_count": 25,
    "vehicle_id": "BUS-001"
  },
  "today_attendance": {
    "present": 23,
    "absent": 1,
    "late": 1
  },
  "open_issues": 2,
  "recent_alerts": []
}
```

### Update Issue Status (Manager only)
```
PUT /api/mobile/v1/issues/update?issue_id=123
Authorization: Bearer {token}
Content-Type: application/json

{
  "status": "resolved",
  "resolution_notes": "Replaced turn signal bulb"
}
```

## Error Responses

All endpoints return standard error responses:

```json
{
  "error": "Error message",
  "code": 400
}
```

Common HTTP status codes:
- 200: Success
- 201: Created
- 400: Bad Request
- 401: Unauthorized
- 403: Forbidden
- 404: Not Found
- 500: Internal Server Error

## Rate Limiting

API requests are rate limited to:
- 100 requests per minute for authenticated users
- 10 requests per minute for login endpoints

## Best Practices

1. **Authentication**: Always include the Bearer token in the Authorization header
2. **Timestamps**: Use ISO 8601 format for all timestamps
3. **GPS Updates**: Send location updates every 30 seconds when on route
4. **Offline Support**: Store attendance data locally and sync when connection is restored
5. **Photo Uploads**: Compress images before uploading (max 10MB)
6. **Error Handling**: Implement retry logic with exponential backoff

## WebSocket Connection (Real-time Updates)

```javascript
const ws = new WebSocket('wss://api.fleetmanagement.com/ws/mobile');

ws.onopen = () => {
  ws.send(JSON.stringify({
    type: 'authenticate',
    token: 'your-jwt-token'
  }));
};

ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  // Handle real-time updates
};
```

## Sample Mobile App Flow

1. **Login**: Authenticate and store tokens securely
2. **Get Route**: Fetch assigned route and student list
3. **Start Route**: Update status to "on_route"
4. **Track Location**: Send GPS updates every 30 seconds
5. **Record Attendance**: Mark students as they board
6. **Handle Issues**: Report and photograph any problems
7. **Complete Route**: Mark students as dropped off
8. **End Route**: Update status to "available"

## Security Considerations

1. Store tokens securely using platform-specific secure storage
2. Implement certificate pinning for HTTPS connections
3. Validate all server responses
4. Implement app-level authentication (PIN/biometric)
5. Clear sensitive data on logout
6. Use encrypted communication for all API calls