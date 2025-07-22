# Fleet Management System - Testing Guide

## Quick Start Testing

### 1. Build and Start the Application
```bash
go build -o hs-bus.exe .
./hs-bus.exe
```

The application runs on port 5003 by default.

### 2. Login Credentials
- **Admin User**: username: `admin`, password: `admin`
- **Test Driver**: username: `driver1`, password: `password` (if exists)

### 3. Main Pages to Test

#### After Login (http://localhost:5003/)
1. **Dashboard** - Should show after successful login
2. **Fleet Page** (http://localhost:5003/fleet)
   - Should display all buses and vehicles
   - Expected: 10 buses + 44 vehicles = 54 total
   - Each vehicle should show status, model, oil/tire status

3. **Other Key Pages**:
   - Students: http://localhost:5003/students
   - Routes: http://localhost:5003/assign-routes
   - ECSE Import: http://localhost:5003/import-ecse
   - User Management: http://localhost:5003/manage-users

## Debugging Fleet Issues

### If Fleet Page Shows "Unable to load fleet data"

1. **Check Server Console**
   - Look for DEBUG/ERROR messages
   - Common errors:
     - "database not initialized"
     - "failed to load buses/vehicles"
     - SQL syntax errors

2. **Direct Database Test**
   Connect to database and run:
   ```sql
   SELECT COUNT(*) FROM buses;        -- Should return 10
   SELECT COUNT(*) FROM vehicles;     -- Should return 44
   ```

3. **Check fleet_handler_clean.go**
   - This file contains the working fleet handler
   - It loads buses and vehicles separately
   - Handles nullable fields properly

### Common Issues and Solutions

1. **"Login required" on all pages**
   - Clear browser cookies
   - Ensure you're logged in as admin
   - Check session expiration (24 hours)

2. **Fleet shows 0 vehicles**
   - Database connection issue
   - Check DATABASE_URL environment variable
   - Verify tables exist with data

3. **"Internal Server Error"**
   - Check server console for panic messages
   - Usually indicates nil pointer or type conversion error
   - Review recent code changes

## Testing Workflow

### Manual Testing Steps
1. Start application
2. Login with admin/admin
3. Navigate to Fleet page
4. Verify vehicle count (should be 54)
5. Click on a vehicle for maintenance history
6. Test Add Bus functionality
7. Test status updates (active/maintenance/out-of-service)

### What to Look For
- ✅ All pages load without errors
- ✅ Data displays correctly
- ✅ Forms submit successfully
- ✅ Navigation works between pages
- ✅ Logout works and redirects to login

## Database Verification

Run these queries to verify data:
```sql
-- Check user exists
SELECT username, role, status FROM users WHERE username = 'admin';

-- Check fleet data
SELECT bus_id, status, model FROM buses LIMIT 5;
SELECT vehicle_id, status, model FROM vehicles LIMIT 5;

-- Check assignments
SELECT * FROM route_assignments LIMIT 5;
```

## If Everything Fails

1. **Restart Fresh**
   ```bash
   go build -o hs-bus.exe .
   ./hs-bus.exe
   ```

2. **Check Environment**
   - DATABASE_URL must be set
   - Port 5003 must be available

3. **Review Logs**
   - All errors are logged to console
   - Look for first ERROR message
   - DEBUG messages show execution flow

## Test Endpoints (Development Only)

When ENABLE_TEST_SERVER=true, these bypass authentication:
- http://localhost:5004/ - Test server status
- http://localhost:5004/status - Database status
- http://localhost:5004/fleet-test - Fleet loading test
- http://localhost:5004/db-test - Direct database test

## Expected Results

### Fleet Page
- Shows 54 total vehicles (10 buses + 44 company vehicles)
- Each vehicle has ID, status, model, oil/tire status
- Status indicators: green (active), yellow (maintenance), red (out-of-service)
- Maintenance history links work

### Dashboard
- Shows summary statistics
- Recent activity
- Quick links to main functions

### Data Management
- Can add/edit buses
- Can assign routes to drivers
- Can track maintenance records
- Can import ECSE student data

Remember: The CLAUDE.md file contains full system documentation including:
- CSRF token requirements
- Session management details
- Handler patterns
- Security implementation