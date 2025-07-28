# Fleet Management System - Advanced Features Guide

## 🚀 Overview

This guide documents the advanced features that have been added to the Fleet Management System to transform it into a production-ready, enterprise-grade application.

## 📋 Table of Contents

1. [Real-Time Dashboard](#real-time-dashboard)
2. [WebSocket Communication](#websocket-communication)
3. [Mobile API](#mobile-api)
4. [Advanced Analytics](#advanced-analytics)
5. [Notification System](#notification-system)
6. [Performance Monitoring](#performance-monitoring)
7. [Backup & Recovery](#backup-recovery)
8. [Security Enhancements](#security-enhancements)
9. [Production Configuration](#production-configuration)

## 🎯 Real-Time Dashboard

### Overview
The real-time dashboard provides live updates of fleet operations without page refreshes.

### Access
- **URL**: `/realtime-dashboard`
- **Role**: Manager only
- **Features**:
  - Live vehicle tracking on map
  - Real-time system metrics
  - Active alerts and notifications
  - Operations chat
  - Live activity feed

### Key Components
```javascript
// Connect to WebSocket
ws://your-domain.com/ws

// Message types:
- system_metrics: System performance data
- driver_location: GPS updates
- maintenance_alert: Vehicle issues
- route_update: Route status changes
- emergency: Emergency notifications
```

## 🔄 WebSocket Communication

### Overview
Enables real-time bidirectional communication between server and clients.

### Connection
```javascript
const ws = new WebSocket('ws://localhost:5003/ws');

ws.onmessage = (event) => {
    const message = JSON.parse(event.data);
    // Handle different message types
};
```

### Message Types
1. **System Metrics**
   ```json
   {
     "type": "system_metrics",
     "data": {
       "active_buses": 15,
       "active_routes": 8,
       "database": { "connections": 10 }
     }
   }
   ```

2. **Driver Location**
   ```json
   {
     "type": "driver_location",
     "data": {
       "driver": "john_doe",
       "latitude": 40.7128,
       "longitude": -74.0060
     }
   }
   ```

## 📱 Mobile API

### Overview
RESTful API for mobile applications (iOS/Android) used by drivers.

### Authentication
```bash
POST /api/mobile/v1/login
Content-Type: application/json

{
  "username": "driver123",
  "password": "password",
  "device_id": "unique-device-id",
  "platform": "ios"
}
```

### Key Endpoints

#### Get Current Route
```bash
GET /api/mobile/v1/driver/route
Authorization: Bearer <token>
```

#### Update Location
```bash
POST /api/mobile/v1/driver/location
Authorization: Bearer <token>

{
  "latitude": 40.7128,
  "longitude": -74.0060,
  "speed": 35.5,
  "heading": 180.0
}
```

#### Submit Attendance
```bash
POST /api/mobile/v1/driver/attendance
Authorization: Bearer <token>

[
  {
    "student_id": "STU001",
    "status": "present",
    "boarded_at": "2024-01-23T07:30:00Z"
  }
]
```

#### Pre-Trip Inspection
```bash
POST /api/mobile/v1/driver/inspection
Authorization: Bearer <token>

{
  "bus_id": "BUS-001",
  "mileage": 45000,
  "fuel_level": "3/4",
  "items": [
    {
      "category": "Brakes",
      "item": "Brake Pedal",
      "status": "pass"
    }
  ],
  "safe_to_drive": true
}
```

## 📊 Advanced Analytics

### Overview
Comprehensive analytics engine providing insights into fleet operations.

### Access Analytics
```bash
GET /api/analytics/fleet
Authorization: Bearer <token>
```

### Analytics Categories

1. **Operational Metrics**
   - Fleet utilization rate
   - Average route times
   - Driver utilization
   - Route efficiency scores

2. **Financial Metrics**
   - Total operating costs
   - Cost per mile
   - Fuel cost trends
   - Maintenance cost analysis

3. **Safety Metrics**
   - Accident rates
   - Maintenance compliance
   - Driver safety scores
   - Vehicle health scores

4. **Predictive Insights**
   - Maintenance forecasts
   - Fuel cost projections
   - Fleet expansion needs
   - Risk assessments

### Export Reports
```bash
GET /api/analytics/export?format=pdf
```

## 🔔 Notification System

### Overview
Multi-channel notification system supporting email, SMS, push, and in-app notifications.

### Configuration
```env
# Email
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USERNAME=your-email@gmail.com
SMTP_PASSWORD=your-app-password

# SMS (Twilio example)
SMS_PROVIDER=twilio
SMS_ACCOUNT_SID=your-account-sid
SMS_AUTH_TOKEN=your-auth-token
SMS_FROM_NUMBER=+1234567890

# Push Notifications
FCM_SERVER_KEY=your-fcm-key
```

### Sending Notifications
```go
notification := BuildMaintenanceNotification(vehicle, dueDate)
notification.Recipients = []Recipient{
    {UserID: "123", Email: "manager@company.com"},
}
notification.Channels = []string{"email", "push", "in-app"}

notificationSystem.Send(notification)
```

### Notification Types
- `maintenance_due`: Vehicle maintenance reminders
- `route_change`: Route assignment changes
- `emergency`: Emergency alerts
- `attendance_issue`: Student attendance problems
- `vehicle_issue`: Vehicle problems reported
- `schedule_reminder`: Schedule reminders
- `system_alert`: System-wide alerts

## 📈 Performance Monitoring

### Overview
Real-time performance monitoring and optimization tools.

### Access Metrics
```bash
GET /api/performance/metrics
```

### Features
1. **Query Performance**
   - Tracks slow queries (>100ms)
   - Identifies optimization opportunities
   - Monitors database connection pool

2. **Endpoint Performance**
   - Response time tracking
   - Error rate monitoring
   - Throughput analysis

3. **Resource Usage**
   - Memory consumption
   - CPU utilization
   - Goroutine monitoring

### View Slow Queries
```bash
GET /api/performance/slow-queries
```

## 💾 Backup & Recovery

### Overview
Automated backup system with point-in-time recovery capabilities.

### Manual Backup
```bash
POST /api/backup/create
```

### List Backups
```bash
GET /api/backup/list
```

### Restore Backup
```bash
POST /api/backup/restore
{
  "backup_file": "fleet_backup_20240123_020000.zip"
}
```

### Automatic Backups
- **Schedule**: Daily at 2 AM
- **Retention**: 7 days
- **Location**: `./backups/` or `BACKUP_PATH` env variable

### Recovery Features
1. **Auto-Recovery**
   - Database connection recovery
   - Cache recovery
   - Session recovery

2. **Manual Recovery**
   ```bash
   POST /api/recovery/trigger?component=database
   ```

## 🔒 Security Enhancements

### Input Validation
All user inputs are validated and sanitized:
```go
// Example validation rules
rules := map[string][]ValidationRule{
    "email": {Required(), Email()},
    "phone": {Required(), Phone()},
    "mileage": {Required(), Numeric(), Min(0)},
}
```

### Rate Limiting
Enhanced rate limiting per IP:
- **Default**: 100 requests/minute
- **Login**: 5 attempts/15 minutes
- **API**: 1000 requests/hour

### Secure Queries
All database queries use parameterized statements:
```go
// Safe query example
query := "SELECT * FROM users WHERE username = $1"
db.QueryRow(query, username)
```

## ⚡ Production Configuration

### Environment Variables
```env
# Application
APP_ENV=production
PORT=5003

# Database
DB_MAX_CONNECTIONS=25
DB_MAX_IDLE=5
DB_QUERY_TIMEOUT=30

# Security
SESSION_TIMEOUT_HOURS=24
MAX_LOGIN_ATTEMPTS=5
LOCKOUT_DURATION_MINUTES=15

# Performance
CACHE_TTL_MINUTES=10
METRICS_ENABLED=true

# Backups
BACKUP_PATH=/var/backups/fleet
```

### Production Optimizations
1. **Database Indices**: Automatically created for performance
2. **Query Caching**: Common queries cached for 10 minutes
3. **Connection Pooling**: Optimized database connections
4. **Static Asset Caching**: Browser caching headers
5. **Gzip Compression**: Response compression

### Health Checks
```bash
GET /api/health

# Response
{
  "status": "healthy",
  "database": "connected",
  "uptime": 86400,
  "version": "2.0.0"
}
```

## 🚀 Quick Start

### 1. Run Database Migration
```bash
psql -U your_user -d your_database -f migrations/004_advanced_features.sql
```

### 2. Configure Environment
```bash
cp .env.example .env
# Edit .env with your configuration
```

### 3. Start Server
```bash
go run .
```

### 4. Access Features
- Main App: http://localhost:5003
- Real-time Dashboard: http://localhost:5003/realtime-dashboard
- Mobile API Docs: http://localhost:5003/api/docs

## 📝 Best Practices

1. **Monitor Performance**
   - Check `/api/performance/metrics` regularly
   - Review slow queries weekly
   - Monitor error rates

2. **Backup Strategy**
   - Verify daily backups
   - Test restore process monthly
   - Keep off-site backup copies

3. **Security**
   - Rotate JWT secrets regularly
   - Review audit logs
   - Update dependencies monthly

4. **Mobile App**
   - Use refresh tokens
   - Implement offline mode
   - Cache route data locally

## 🆘 Troubleshooting

### WebSocket Connection Issues
```javascript
// Add reconnection logic
let reconnectInterval = null;
ws.onclose = () => {
    reconnectInterval = setInterval(() => {
        connectWebSocket();
    }, 5000);
};
```

### Performance Issues
1. Check slow queries: `/api/performance/slow-queries`
2. Review database connections: `/api/db/stats`
3. Check memory usage in monitoring dashboard

### Notification Delivery
1. Verify SMTP settings
2. Check notification status: Query `notifications` table
3. Review delivery logs: `notification_deliveries` table

## 🎯 Future Enhancements

1. **AI-Powered Route Optimization**
2. **Predictive Maintenance ML Models**
3. **Voice Commands for Drivers**
4. **Blockchain-based Audit Trail**
5. **IoT Sensor Integration**

---

*For additional support, contact the development team or check the issue tracker.*