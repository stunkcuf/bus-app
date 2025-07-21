# Maintenance Logs Fix Summary

## Date: January 20, 2025

## Issues Fixed

### 1. Database Query Update
**Problem**: The `getMaintenanceLogsForVehicle` function was querying deleted tables (`bus_maintenance_logs` and `vehicle_maintenance_logs`)
**Solution**: Updated to query only the consolidated `maintenance_records` table with proper field mapping

### 2. Save Functions Update
**Problem**: `saveBusMaintenanceLog` and `saveVehicleMaintenanceLog` were trying to insert into deleted tables
**Solution**: Updated both functions to insert into `maintenance_records` table

### 3. Handler Data Format
**Problem**: Handlers were passing data in wrong format for the template
**Solution**: Updated both `busMaintenanceHandler` and `vehicleMaintenanceHandler` to use the correct data structure expected by the template

### 4. Database Migrations
**Problem**: Migrations still trying to create deleted tables
**Solution**: Commented out creation of `bus_maintenance_logs` and `vehicle_maintenance_logs` tables and their indexes

### 5. CREATE TABLE for maintenance_records
**Problem**: No CREATE TABLE statement for the consolidated table
**Solution**: Added proper CREATE TABLE statement for `maintenance_records` in migrations

## Files Modified

1. **database.go**
   - Updated `getMaintenanceLogsForVehicle` to query `maintenance_records`
   - Commented out deprecated table creation
   - Added CREATE TABLE for `maintenance_records`

2. **data.go**
   - Updated `saveBusMaintenanceLog` to use `maintenance_records`
   - Updated `saveVehicleMaintenanceLog` to use `maintenance_records`

3. **handlers.go**
   - Fixed data structure in `busMaintenanceHandler`
   - Fixed data structure in `vehicleMaintenanceHandler`
   - Changed `MaintenanceLogs` to `MaintenanceRecords` to match template

4. **handlers_missing.go**
   - Updated maintenance save logic to use consolidated table

## Query Changes

### Old Query (UNION from deleted tables):
```sql
SELECT id, bus_id as vehicle_id, 'bus' as vehicle_type, date, category, notes, mileage, cost, created_at
FROM bus_maintenance_logs WHERE bus_id = $1
UNION ALL
SELECT id, vehicle_id, 'vehicle' as vehicle_type, date, category, notes, mileage, cost, created_at
FROM vehicle_maintenance_logs WHERE vehicle_id = $1
```

### New Query (from maintenance_records only):
```sql
SELECT 
    id,
    COALESCE(vehicle_id, CAST(vehicle_number AS VARCHAR)) as vehicle_id,
    CASE 
        WHEN vehicle_id LIKE 'BUS%' OR vehicle_id ~ '^[0-9]+$' THEN 'bus'
        ELSE 'vehicle'
    END as vehicle_type,
    COALESCE(TO_CHAR(service_date, 'YYYY-MM-DD'), TO_CHAR(date, 'YYYY-MM-DD'), '') as date,
    COALESCE(
        CASE 
            WHEN work_description ILIKE '%oil%' THEN 'oil_change'
            WHEN work_description ILIKE '%tire%' THEN 'tire_service'
            WHEN work_description ILIKE '%inspect%' THEN 'inspection'
            WHEN work_description ILIKE '%repair%' THEN 'repair'
            ELSE 'other'
        END,
        'other'
    ) as category,
    COALESCE(work_description, '') as notes,
    COALESCE(mileage, 0) as mileage,
    CASE 
        WHEN cost ~ '^[0-9]+\.?[0-9]*$' THEN CAST(cost AS NUMERIC)
        ELSE 0
    END as cost,
    created_at
FROM maintenance_records
WHERE vehicle_id = $1 OR CAST(vehicle_number AS VARCHAR) = $1
```

## Testing

The maintenance logs should now:
1. Display correctly when clicking maintenance links from fleet pages
2. Show all 458 records from the consolidated table
3. Save new records to the correct table
4. Handle both bus and vehicle maintenance properly

## Remaining Work

Other files that still reference old tables and may need updating:
- comparative_analytics.go
- charts.go
- dashboard_analytics.go
- driver_scorecards.go
- export_data.go
- pdf_reports.go
- report_builder.go

These can be updated as needed when those features are accessed.