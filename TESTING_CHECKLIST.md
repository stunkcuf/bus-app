# Fleet Management System - Local Testing Checklist

## Pre-Test Setup
1. Make sure the application is running: `go run .`
2. Ensure database is connected and running
3. Login as a manager account to access all features

## 🧪 Testing Checklist

### 1. Fleet Page - Vehicle Count
- [ ] Navigate to `/fleet`
- [ ] **Expected**: Should see ALL 91 vehicles (not just 10)
- [ ] **Check**: Vehicles are grouped by type (buses, trucks, vans, etc.)
- [ ] **Note**: Count the total number displayed

### 2. Manager Dashboard - Real Activity
- [ ] Navigate to `/manager-dashboard`
- [ ] **Expected**: Recent Activity section shows real data:
  - Driver logs (completed routes)
  - Maintenance records
  - New user registrations
- [ ] **Check**: Times show "X minutes/hours ago" format
- [ ] **Note**: Should NOT see "Bus #101 completed morning route" (old mock data)

### 3. Dashboard Statistics
- [ ] On Manager Dashboard, check:
  - [ ] **Total Drivers**: Should show actual count (not `users - 1`)
  - [ ] **Active Drivers**: Should show real number (not 0)
  - [ ] **Total Vehicles**: Should be 91
- [ ] **Note**: All counts should reflect real database data

### 4. ECSE Dashboard
- [ ] Navigate to `/ecse-dashboard` (link should be in manager dashboard)
- [ ] **Expected**: Dashboard loads with ECSE student data
- [ ] **Check**: "Upcoming Assessments" shows actual count (not 0)
- [ ] **Note**: If no assessments due, that's okay - just shouldn't be hardcoded 0

### 5. Maintenance Logs
- [ ] Go to `/fleet`
- [ ] Click maintenance link on any vehicle
- [ ] **Expected**: Maintenance history displays
- [ ] **Check**: Records show with dates, categories, and costs
- [ ] **Note**: Should see actual maintenance records, not empty list

### 6. Route Assignments - Student Counts
- [ ] Navigate to `/assign-routes`
- [ ] **Expected**: Each route shows student count
- [ ] **Check**: Numbers next to routes like "Route A (15 students)"
- [ ] **Note**: Counts should match actual students assigned

### 7. Import System Test
- [ ] Navigate to `/import-data` or import section
- [ ] **Test File Upload**:
  1. Click "Import Data"
  2. Select type (e.g., "student")
  3. Upload a test Excel file
- [ ] **Expected Results**:
  - File analysis shows columns and row count
  - Validation step shows preview of data
  - Can map columns to database fields
  - Import execution shows success/failure counts
- [ ] **Note**: Should NOT see mock data like "John Doe", "Jane Smith"

### 8. Mileage Data (if importing mileage)
- [ ] When viewing mileage reports
- [ ] **Check**: No negative mileage (ending < beginning)
- [ ] **Note**: System should auto-correct or flag invalid mileage

### 9. Average Daily Miles
- [ ] Check any dashboard showing daily averages
- [ ] **Expected**: Calculation based on weekdays only
- [ ] **Note**: Weekend days shouldn't inflate the average

### 10. Error Handling Check
- [ ] Try to access a non-existent record (e.g., `/bus-maintenance/INVALID`)
- [ ] **Expected**: Error message or redirect, not empty page
- [ ] **Note**: Some areas still show empty data on errors (documented for future fix)

## 🐛 Common Issues to Watch For

1. **Import File Upload Fails**
   - Check file size (max 10MB)
   - Ensure it's a valid Excel file (.xlsx)
   - Try a smaller test file

2. **Counts Show 0**
   - Refresh the page
   - Check database connection
   - Verify data exists in tables

3. **ECSE Dashboard Not Found**
   - Check URL: `/ecse-dashboard`
   - Ensure logged in as manager
   - Look for link in manager dashboard

4. **Maintenance Logs Empty**
   - Verify `maintenance_records` table has data
   - Check vehicle ID exists
   - Try different vehicles

## 📝 Test Results Template

```
Date: _____________
Tester: ___________

Fleet Page Vehicle Count: _____ / 91
Manager Dashboard Activity: Real Data? [ ] Yes [ ] No
Driver Count Accurate: [ ] Yes [ ] No
ECSE Dashboard Loads: [ ] Yes [ ] No
Maintenance Logs Display: [ ] Yes [ ] No
Import System Works: [ ] Yes [ ] No

Issues Found:
1. _________________________________
2. _________________________________
3. _________________________________

Notes:
___________________________________
___________________________________
```

## 🔍 Quick SQL Checks

If you need to verify data in the database:

```sql
-- Check vehicle count
SELECT COUNT(*) FROM fleet_vehicles;

-- Check driver count
SELECT COUNT(*) FROM users WHERE role = 'driver';

-- Check active drivers
SELECT COUNT(*) FROM users WHERE role = 'driver' AND status = 'active';

-- Check ECSE students
SELECT COUNT(*) FROM ecse_students;

-- Check maintenance records
SELECT COUNT(*) FROM maintenance_records;

-- Check upcoming assessments
SELECT COUNT(*) FROM ecse_assessments 
WHERE next_review_date BETWEEN CURRENT_DATE AND CURRENT_DATE + INTERVAL '30 days';
```

## ✅ Success Criteria

The system is working correctly if:
1. All vehicle counts match database (91 total)
2. Dashboard shows real activity, not mock data
3. Import system analyzes and processes real files
4. All calculations use proper business logic
5. No hardcoded zeros or mock data visible

## 🚨 If Tests Fail

1. Check application logs for errors
2. Verify database connection
3. Ensure all code changes were saved
4. Restart the application
5. Clear browser cache if needed

Good luck with testing! The system should now show real, accurate data throughout.