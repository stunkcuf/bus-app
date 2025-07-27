# Fleet Management System - New Features Guide

## Table of Contents
1. [GPS Tracking System](#gps-tracking-system)
2. [Notification System](#notification-system)
3. [Mobile API](#mobile-api)
4. [Real-Time Dashboard](#real-time-dashboard)
5. [Budget Management](#budget-management)
6. [Enhanced Import/Export](#enhanced-importexport)
7. [Student Attendance Tracking](#student-attendance-tracking)

---

## GPS Tracking System

### Overview
Real-time vehicle tracking with location, speed, and status monitoring.

### Features
- Live vehicle location updates
- Speed and heading tracking
- Driver status (available, on_route, break, off_duty)
- Emergency reporting system
- Historical route playback

### Usage
1. **Access GPS Tracking**: Navigate to `/gps-tracking` (managers only)
2. **View Live Locations**: See all active vehicles on the map
3. **Track Specific Vehicle**: Click on a vehicle for detailed information
4. **Emergency Response**: Emergency alerts appear in real-time

### Technical Details
- Updates every 30 seconds (configurable)
- Stores 30 days of history
- WebSocket support for real-time updates

---

## Notification System

### Overview
Comprehensive notification system supporting email, SMS, and in-app notifications.

### Features
- Email notifications
- SMS alerts (with provider configuration)
- In-app notifications
- User preference management
- Template-based messages

### Notification Types
1. **Maintenance Alerts**
   - Oil change due
   - Tire service needed
   - Inspection reminders

2. **Route Notifications**
   - Driver assignments
   - Route changes
   - Delays or cancellations

3. **Emergency Alerts**
   - Vehicle breakdowns
   - Accidents
   - Emergency reports

### Configuration
Navigate to `/notification-preferences` to:
- Enable/disable notification types
- Set email preferences
- Configure alert thresholds

---

## Mobile API

### Overview
RESTful API for mobile applications supporting drivers and parents.

### Endpoints

#### Authentication
- `POST /api/mobile/login` - Mobile device login
- `POST /api/mobile/logout` - Logout
- `GET /api/mobile/profile` - User profile

#### Driver Features
- `GET /api/mobile/driver/routes` - Assigned routes
- `POST /api/mobile/driver/location` - Update location
- `GET /api/mobile/driver/students` - Student roster
- `POST /api/mobile/driver/attendance` - Mark attendance
- `POST /api/mobile/driver/emergency` - Report emergency

#### Parent Features
- `GET /api/mobile/parent/students` - Child information
- `GET /api/mobile/parent/bus-location` - Real-time bus tracking
- `GET /api/mobile/parent/notifications` - Alerts and updates

### Authentication
Uses JWT tokens with 24-hour expiration. Include token in Authorization header:
```
Authorization: Bearer <token>
```

---

## Real-Time Dashboard

### Overview
Live monitoring dashboard with WebSocket support for instant updates.

### Features
- Real-time fleet status
- Live vehicle locations
- Active route monitoring
- Emergency alerts
- Performance metrics

### Access
Navigate to `/realtime-dashboard` (managers only)

### Components
1. **Fleet Overview**: Active/idle/maintenance vehicles
2. **Live Map**: Real-time vehicle positions
3. **Alerts Panel**: Current issues and emergencies
4. **Metrics Display**: Speed, fuel, performance data

---

## Budget Management

### Overview
Comprehensive budget tracking and expense management system.

### Features
- Budget creation by category
- Expense tracking
- Budget vs. actual reporting
- Forecast projections
- Export to Excel/PDF

### Usage
1. **Create Budget**: `/budget/create`
   - Set annual budgets
   - Define categories (fuel, maintenance, insurance, etc.)
   - Set alert thresholds

2. **Track Expenses**: Automatically tracked from:
   - Fuel records
   - Maintenance logs
   - Manual entries

3. **View Reports**: `/budget/report`
   - Monthly/quarterly/annual views
   - Category breakdown
   - Variance analysis

---

## Enhanced Import/Export

### Overview
Advanced data import/export with validation and error handling.

### Import Features
- Excel file support (.xlsx, .xls)
- CSV file support
- Data validation before import
- Preview and mapping
- Error reporting
- Rollback capability

### Import Process
1. **Upload File**: `/import-data-wizard`
2. **Select Type**: Students, vehicles, routes, etc.
3. **Map Columns**: Match file columns to database fields
4. **Validate**: Review validation results
5. **Preview**: See data before import
6. **Execute**: Import with progress tracking

### Export Features
- Predefined templates
- Custom report builder
- Multiple formats (Excel, CSV, PDF)
- Scheduled exports
- Email delivery

### Export Process
1. **Choose Template**: `/export-templates`
2. **Set Filters**: Date ranges, categories, etc.
3. **Select Format**: Excel, CSV, or PDF
4. **Download or Email**: Immediate or scheduled

---

## Student Attendance Tracking

### Overview
Digital attendance system with parent notifications.

### Features
- Digital check-in/check-out
- Timestamp recording
- Absence tracking
- Parent notifications
- Attendance reports

### Driver Workflow
1. **Morning Route**:
   - View student roster
   - Mark present/absent
   - Note any issues
   - Submit attendance

2. **Afternoon Route**:
   - Verify morning attendance
   - Mark drop-offs
   - Update any changes

### Parent Features
- Real-time notifications
- Absence alerts
- Pickup/drop-off confirmations
- Historical attendance view

### Manager Reports
- Daily attendance summaries
- Absence patterns
- Route efficiency metrics
- Export capabilities

---

## Configuration Requirements

### GPS Tracking
```env
GPS_UPDATE_INTERVAL=30        # seconds
GPS_HISTORY_RETENTION=30      # days
GPS_ACCURACY_THRESHOLD=50     # meters
```

### Notifications
```env
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USERNAME=your-email@gmail.com
SMTP_PASSWORD=your-app-password
SMS_PROVIDER=twilio
SMS_ACCOUNT_SID=your-account-sid
SMS_AUTH_TOKEN=your-auth-token
```

### Mobile API
```env
JWT_SECRET=your-secret-key
JWT_EXPIRATION=24h
API_RATE_LIMIT=100
API_RATE_WINDOW=60
```

### WebSocket
```env
WEBSOCKET_PORT=8081
WEBSOCKET_PING_INTERVAL=30
WEBSOCKET_ALLOWED_ORIGINS=*
```

---

## Troubleshooting

### GPS Not Updating
1. Check GPS_UPDATE_INTERVAL setting
2. Verify driver mobile app permissions
3. Check network connectivity
4. Review GPS accuracy threshold

### Notifications Not Sending
1. Verify SMTP settings
2. Check spam folders
3. Review notification preferences
4. Check email queue in logs

### Mobile API Issues
1. Verify JWT token not expired
2. Check API rate limits
3. Review CORS settings
4. Check mobile app version

### WebSocket Connection Issues
1. Verify WebSocket port open
2. Check firewall settings
3. Review allowed origins
4. Check SSL/TLS configuration

---

## Security Considerations

1. **GPS Privacy**: Only authorized users can view locations
2. **API Security**: JWT tokens, rate limiting, HTTPS required
3. **Notification Security**: No sensitive data in emails
4. **Data Encryption**: All sensitive data encrypted at rest

---

## Support

For additional help:
- Check system logs at `/monitoring-dashboard`
- Review error details in `/api/health`
- Contact system administrator

**Last Updated**: January 27, 2025