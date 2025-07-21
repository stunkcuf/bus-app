# Data Integration Fix Plan

## Overview
The database migration was successful, but the website handlers are not properly integrated with the new consolidated tables. This plan addresses all identified issues.

## Critical Issues to Fix

### 1. Fleet Page (Shows 10 instead of 91)
**Problem**: Fleet page only shows buses, not all vehicles
**Root Cause**: Handler loads only buses with `vehicle_type = 'bus'`
**Fix Required**:
- Update `fleetHandler` to load ALL vehicles from `fleet_vehicles`
- Update template to handle different vehicle types
- Fix `loadFleetVehiclesFromDB` query to use correct columns

### 2. ECSE Dashboard Not Showing
**Problem**: 825 ECSE students not displaying
**Root Cause**: Missing helper methods for SQL null types in templates
**Fix Required**:
- The helper methods already exist in `models_helpers.go`
- Need to ensure template uses these methods
- Verify handler is passing data correctly

### 3. Maintenance Logs Issues
**Problem**: Maintenance records not showing properly, ID conflicts
**Root Cause**: Old query still using deleted tables
**Fix Required**:
- Update queries to use `maintenance_records` table only
- Remove references to `bus_maintenance_logs` and `vehicle_maintenance_logs`
- Fix ID handling in maintenance display

### 4. Company Fleet Issues
**Problem**: Duplicate/incorrect vehicle loading
**Root Cause**: Loading from both old and new sources
**Fix Required**:
- Remove duplicate data loading
- Use only consolidated `fleet_vehicles` table

### 5. Dashboard Issues
**Problem**: Manager dashboard shows mock data and wrong counts
**Root Cause**: Using old cache methods and mock data
**Fix Required**:
- Update to use consolidated tables
- Remove mock recent activity
- Fix vehicle/bus counts

## Detailed Fixes by File

### handlers.go
1. **fleetHandler** (line 372):
   - Change to load ALL vehicles, not just buses
   - Group by vehicle_type for display
   
2. **companyFleetHandler** (line 468):
   - Remove duplicate loading
   - Use only consolidated vehicles
   
3. **managerDashboardHandler** (line 181):
   - Update vehicle counts to use consolidated table
   - Remove mock recent activity
   - Load real maintenance data

### data.go
1. **loadFleetVehiclesFromDB** (line 509):
   - Fix query to match actual fleet_vehicles columns
   - Remove incorrect column references
   
2. **Add new function**:
   - `loadAllConsolidatedVehicles()` for complete fleet view

### handlers_ecse.go
1. **ecseDashboardHandler** (line 12):
   - Verify data is being passed to template
   - Check template is using helper methods

### database.go
1. **getMaintenanceLogsForVehicle** (line 610):
   - Update to query only `maintenance_records` table
   - Remove UNION with deleted tables
   
2. **getVehicleMaintenanceInfo** (line 662):
   - Update column references for new structure

### cache.go
1. **Update cache methods**:
   - `getBuses()` should load from consolidated table
   - `getVehicles()` should load from consolidated table

### Template Updates Required
1. **fleet.html**:
   - Handle all vehicle types, not just buses
   - Show proper counts
   
2. **ecse_dashboard_modern.html**:
   - Ensure using helper methods for null fields
   
3. **manager_dashboard.html**:
   - Update to show real data, not mocks

## Implementation Order

### Phase 1: Fix Critical Display Issues (1-2 hours)
1. Fix fleet page to show all 91 vehicles
2. Fix ECSE dashboard display
3. Fix maintenance logs query

### Phase 2: Update Data Loading (1 hour)
1. Update all load functions in data.go
2. Fix cache methods
3. Remove duplicate loading

### Phase 3: Handler Updates (2 hours)
1. Update all handlers to use consolidated tables
2. Remove mock data
3. Fix dashboard counts

### Phase 4: Template Verification (1 hour)
1. Verify all templates use correct field names
2. Test each page for proper display
3. Fix any remaining display issues

### Phase 5: Testing & Validation (1 hour)
1. Test all pages systematically
2. Verify counts match database
3. Check all CRUD operations

## Quick Wins (Do First)

### 1. Fleet Page Fix
```go
// In fleetHandler, change:
buses, err := loadConsolidatedBusesFromDB()
// To:
vehicles, err := loadAllConsolidatedVehicles()
```

### 2. ECSE Dashboard Fix
- Verify template is using `{{.GetGrade}}` instead of `{{.Grade}}`
- Check all nullable field access

### 3. Maintenance Query Fix
```sql
-- Change FROM bus_maintenance_logs UNION vehicle_maintenance_logs
-- To: FROM maintenance_records only
```

## Expected Results
- Fleet page: Shows all 91 vehicles
- ECSE dashboard: Shows all 825 students
- Maintenance: Shows all 458 records
- Dashboards: Show real counts, not mocks
- All pages: Display accurate data from consolidated tables

## Testing Checklist
- [ ] Fleet shows 91 vehicles (10 buses, 81 others)
- [ ] ECSE shows 825 students
- [ ] Maintenance shows 458 records
- [ ] Manager dashboard shows real counts
- [ ] Driver dashboard works correctly
- [ ] All CRUD operations work
- [ ] No errors in console/logs

## Rollback Plan
- Code changes only, no database changes needed
- Can revert handlers individually if issues arise
- Original backup still available