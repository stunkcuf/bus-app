# Fleet Management System - Data Display Issues Report

## Executive Summary
After thorough investigation of the codebase, I've identified several specific data display issues across various dashboards and components. These issues range from missing implementations to incorrect calculations and data filtering problems.

## 1. Driver Dashboard Issues

### Issue 1.1: Missing Driver-Specific Statistics
- **Location**: `driverDashboardHandler` in `handlers.go`
- **Problem**: Driver-specific stats (total miles driven, students transported, etc.) are not calculated
- **Impact**: Drivers don't see their individual performance metrics

### Issue 1.2: Maintenance Alerts Disabled
- **Location**: Line 321 in `handlers.go`
- **Problem**: Comment states "Maintenance checks disabled - required columns not in database"
- **Impact**: Drivers don't receive vehicle maintenance warnings

## 2. Route Assignment Display Issues

### Issue 2.1: No Validation for Double Assignments
- **Location**: `assignRouteHandler` in `handlers_missing.go`
- **Problem**: System allows multiple drivers to be assigned to the same bus/route
- **Impact**: Data integrity issues and scheduling conflicts

### Issue 2.2: Missing Assignment History
- **Location**: Route assignments table
- **Problem**: No tracking of historical assignments
- **Impact**: Cannot view past route assignments or driver history

## 3. Student Management Issues

### Issue 3.1: Active Student Filter
- **Location**: Line 409 in `data.go`
- **Problem**: Query filters only active students: `WHERE active = true`
- **Impact**: Inactive students are never displayed, even when needed for reports

### Issue 3.2: Missing Student Count per Route
- **Location**: `studentsHandler` in `handlers_missing.go`
- **Problem**: No aggregation of student counts per route
- **Impact**: Cannot see total students assigned to each route

## 4. Company Fleet Display Issues

### Issue 4.1: Mixed Vehicle Type Handling
- **Location**: `companyFleetHandler` in `handlers.go`
- **Problem**: Attempts to load from both old and new table structures
- **Impact**: Potential duplicate or missing vehicles in display

### Issue 4.2: Fallback Logic Confusion
- **Location**: Lines 456-487 in `handlers.go`
- **Problem**: Complex fallback logic between `fleet_vehicles` and old `vehicles` table
- **Impact**: Inconsistent data display depending on which table has data

## 5. User Management Issues

### Issue 5.1: Driver Count Calculation
- **Location**: Line 242 in `handlers.go`
- **Problem**: `TotalDrivers: len(users) - 1` assumes only one manager
- **Impact**: Incorrect driver count if multiple managers exist

### Issue 5.2: No Role Filtering in Some Queries
- **Location**: Various user queries
- **Problem**: Some queries don't filter by role when calculating statistics
- **Impact**: Managers might be counted as drivers in some metrics

## 6. Dashboard Statistics Issues

### Issue 6.1: Hardcoded Zero Values
- **Location**: Manager dashboard handler
- **Problem**: `ActiveDrivers` initially set to 0, then recalculated later
- **Impact**: Potential race condition or display of zero if calculation fails

### Issue 6.2: Maintenance Status Calculation
- **Location**: Lines 203-219 in `handlers.go`
- **Problem**: Only checks `oil_status` and `tire_status`, ignoring other maintenance types
- **Impact**: Incomplete maintenance alerts

## 7. ECSE Dashboard Issues

### Issue 7.1: Missing Upcoming Assessments
- **Location**: Line 61 in `handlers_ecse.go`
- **Problem**: `upcomingAssessments` is always 0 - no implementation
- **Impact**: ECSE dashboard shows 0 for upcoming assessments

### Issue 7.2: Transportation Required Filter
- **Location**: ECSE student queries
- **Problem**: No filtering for students who actually need transportation
- **Impact**: All ECSE students shown regardless of transportation needs

## 8. Mileage Report Issues

### Issue 8.1: No Validation for Mileage Data
- **Location**: Monthly mileage reports
- **Problem**: No validation that ending miles > beginning miles
- **Impact**: Negative or incorrect mileage calculations possible

### Issue 8.2: Missing Mileage Aggregations
- **Location**: Mileage report handlers
- **Problem**: No daily/weekly/monthly aggregations calculated
- **Impact**: Only raw data shown, no useful summaries

## 9. Maintenance Record Issues

### Issue 9.1: Inconsistent Date Handling
- **Location**: Line 685 in `data.go`
- **Problem**: Uses `COALESCE(service_date, date, created_at)` - multiple date fields
- **Impact**: Confusing date display in maintenance records

### Issue 9.2: Missing Cost Calculations
- **Location**: Maintenance record displays
- **Problem**: No total cost calculations or budget tracking
- **Impact**: Cannot see maintenance spending trends

## 10. General Data Loading Issues

### Issue 10.1: Silent Error Handling
- **Location**: Multiple handlers
- **Problem**: Errors logged but empty data returned: `logs = []SomeType{}`
- **Impact**: Users see empty lists instead of error messages

### Issue 10.2: Cache Invalidation Gaps
- **Location**: Various update handlers
- **Problem**: Some updates don't invalidate related caches
- **Impact**: Stale data displayed until cache expires

## 11. TODO/FIXME Items Found

### Issue 11.1: Import Functionality Stubs
- **Location**: Lines 1332, 1368, 1411 in `handlers_missing.go`
- **Problem**: Import analysis, validation, and processing are TODO stubs
- **Impact**: Import features don't actually work

### Issue 11.2: Report Builder Save
- **Location**: Line 316 in `report_builder.go`
- **Problem**: "TODO: Save to database" - reports aren't persisted
- **Impact**: Custom reports lost on page refresh

### Issue 11.3: Scheduled Export Edit
- **Location**: Line 111 in `scheduled_exports.go`
- **Problem**: Edit functionality returns "not yet implemented"
- **Impact**: Cannot modify scheduled exports once created

## Recommendations

### Immediate Fixes Needed:
1. Implement missing ECSE upcoming assessments calculation
2. Fix driver count calculation to properly filter by role
3. Implement proper mileage validation
4. Complete TODO stubs for import functionality
5. Fix maintenance alerts by ensuring required database columns exist

### Data Integrity Improvements:
1. Add validation for route assignments to prevent double-booking
2. Implement assignment history tracking
3. Add proper error messages instead of showing empty data
4. Validate mileage entries (ending > beginning)

### Performance Optimizations:
1. Fix cache invalidation to cover all related data
2. Consolidate vehicle data sources to avoid complex fallback logic
3. Implement proper data aggregations at query level

### User Experience Enhancements:
1. Show meaningful error messages when data loading fails
2. Add loading indicators for slow queries
3. Implement proper filtering options for inactive records
4. Add data export options for all major lists

## Conclusion

The system has numerous data display issues ranging from missing implementations to incorrect calculations. The most critical issues are:
1. Missing ECSE assessment tracking
2. Stub implementations for import features  
3. Incorrect user/driver counting
4. Missing maintenance alerts due to database schema issues

These issues significantly impact the system's usefulness for fleet management operations and should be addressed systematically, starting with the most critical user-facing features.