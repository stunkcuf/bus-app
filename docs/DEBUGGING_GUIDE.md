# Fleet Management System - Debugging Guide

## Quick Status Checks

### 1. System Status (No Login Required)
URL: http://localhost:5003/status
- Shows database connection status
- Record counts for all major tables
- Tests key data loading functions
- No authentication required

### 2. Debug Fleet (No Login Required)
URL: http://localhost:5003/debug-fleet
- Detailed fleet data debugging
- Shows table record counts
- Tests all fleet loading functions
- Direct SQL query results
- Cache status

### 3. Test All Pages (Admin Login Required)
URL: http://localhost:5003/test-all-pages
- Tests all major pages
- Shows HTTP response codes
- Checks for error messages
- Database status summary
- Function call results

### 4. Test Pages Dashboard (Admin Login Required)
URL: http://localhost:5003/test-pages
- Interactive page testing
- Links to all pages
- Database statistics
- Known issues list

## Common Issues and Solutions

### Issue: "Unable to load fleet data"
1. Check http://localhost:5003/status
2. Verify database connection
3. Check buses and vehicles table counts
4. Look at loadAllFleetVehiclesFromDB result

### Issue: Login Required
1. Use admin/admin credentials
2. Check session cookie exists
3. Verify port 5003 is correct

### Issue: Page Not Loading
1. Check browser console for errors
2. Look at server logs
3. Use test-all-pages to identify specific issue
4. Check debug endpoints for data issues

## Key Data Expectations
- Buses: 10 records
- Vehicles: 44 records  
- Total Fleet: 54 records
- Users: 4 records
- ECSE Students: 825 records
- Maintenance Records: 458 records

## Testing Workflow
1. Start with /status - verify basics
2. Login as admin
3. Go to /test-all-pages - check all pages
4. If fleet fails, check /debug-fleet
5. Use browser developer tools for client-side issues
6. Check server console for detailed logs

## Server Logs
The application now includes detailed debug logging:
- DEBUG: Normal operation logs
- ERROR: Failure logs with details
- Look for "fleetHandler" and "loadAllFleetVehiclesFromDB" entries

## Direct Database Access
If needed, you can query the database directly:
```sql
-- Check buses
SELECT COUNT(*) FROM buses;
SELECT bus_id, status FROM buses LIMIT 5;

-- Check vehicles  
SELECT COUNT(*) FROM vehicles;
SELECT vehicle_id, status FROM vehicles LIMIT 5;

-- Check users
SELECT username, role, status FROM users;
```

## Fleet Page Specific Debugging
The fleet page requires:
1. Valid session (logged in)
2. Database connection
3. Buses table accessible
4. Vehicles table accessible
5. Both tables have data

Use /debug-fleet to verify all these requirements.