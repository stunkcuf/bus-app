# Database Table Connection Tasks

## Overview
Currently only 19 out of 29 tables (66%) are properly connected to the application. This document outlines the work needed to fully integrate all database tables.

## Priority 1: Tables with Existing Data (Need Immediate Connection)

### 1. fleet_vehicles (70 rows)
- [ ] Add CREATE TABLE statement in database.go
- [ ] Create FleetVehicle struct in models.go
- [ ] Add loadFleetVehiclesFromDB() in data.go
- [ ] Create fleetVehiclesHandler in handlers.go
- [ ] Create fleet_vehicles.html template
- [ ] Add route in main.go
- [ ] Add menu item in navigation

### 2. maintenance_records (409 rows)
- [ ] Add CREATE TABLE statement in database.go
- [ ] Create MaintenanceRecord struct in models.go
- [ ] Add loadMaintenanceRecordsFromDB() in data.go
- [ ] Create maintenanceRecordsHandler in handlers.go
- [ ] Create maintenance_records.html template
- [ ] Add route in main.go
- [ ] Link from maintenance pages

### 3. monthly_mileage_reports (1,723 rows)
- [ ] Add CREATE TABLE statement in database.go
- [ ] Create MonthlyMileageReport struct in models.go
- [ ] Add loadMonthlyMileageReportsFromDB() in data.go
- [ ] Create monthlyMileageReportsHandler in handlers.go
- [ ] Create monthly_mileage_reports.html template
- [ ] Add route in main.go
- [ ] Add to reports menu

### 4. maintenance_sheets (10 rows)
- [ ] Add CREATE TABLE statement in database.go
- [ ] Create MaintenanceSheet struct in models.go
- [ ] Add loadMaintenanceSheetsFromDB() in data.go
- [ ] Create maintenanceSheetsHandler in handlers.go
- [ ] Create maintenance_sheets.html template
- [ ] Add route in main.go

### 5. service_records (55 rows)
- [ ] Add CREATE TABLE statement in database.go
- [ ] Create ServiceRecord struct in models.go
- [ ] Add loadServiceRecordsFromDB() in data.go
- [ ] Create serviceRecordsHandler in handlers.go
- [ ] Create service_records.html template
- [ ] Add route in main.go

## Priority 2: Import-Related Tables (Partially Implemented)

### 6. agency_vehicles (0 rows)
- [ ] Add CREATE TABLE statement in database.go
- [ ] Fix import logic in import_mileage.go
- [ ] Create viewing handler
- [ ] Create agency_vehicles.html template
- [ ] Add route in main.go

### 7. school_buses (0 rows)
- [ ] Add CREATE TABLE statement in database.go
- [ ] Fix import logic in import_mileage.go
- [ ] Create viewing handler
- [ ] Create school_buses.html template
- [ ] Add route in main.go

### 8. program_staff (0 rows)
- [ ] Add CREATE TABLE statement in database.go
- [ ] Fix import logic in import_mileage.go
- [ ] Create viewing handler
- [ ] Create program_staff.html template
- [ ] Add route in main.go

## Priority 3: Empty Tables (May Need Implementation)

### 9. activities (0 rows)
- [ ] Determine purpose/usage
- [ ] Add CREATE TABLE statement if needed
- [ ] Create Activity struct
- [ ] Create handlers and templates
- [ ] Add routes

### 10. all_vehicle_mileage (0 rows)
- [ ] Determine if this is a view or table
- [ ] Create if needed for reporting
- [ ] Add viewing capabilities

## Existing Issues to Fix

### Import System (Currently Disabled)
The import handlers are commented out in main.go (lines 559-568):
```go
// Disabled routes:
// mux.HandleFunc("/import", ...)
// mux.HandleFunc("/import-history", ...)
// mux.HandleFunc("/import-details", ...)
// etc.
```

- [ ] Re-enable import routes
- [ ] Test import functionality
- [ ] Ensure import_history and import_errors tables are properly used

## Implementation Plan

### Phase 1: Connect Tables with Existing Data (Week 1)
1. fleet_vehicles (70 rows)
2. maintenance_records (409 rows)
3. monthly_mileage_reports (1,723 rows)
4. maintenance_sheets (10 rows)
5. service_records (55 rows)

### Phase 2: Fix Import System (Week 2)
1. Re-enable import routes
2. Fix agency_vehicles import
3. Fix school_buses import
4. Fix program_staff import

### Phase 3: Evaluate Empty Tables (Week 3)
1. Determine which empty tables are needed
2. Implement necessary functionality
3. Consider removing truly unused tables

## Testing Checklist

For each connected table:
- [ ] Verify CREATE TABLE statement works
- [ ] Test data loading from database
- [ ] Verify HTML template displays correctly
- [ ] Test create/read/update/delete operations
- [ ] Check navigation and menu items
- [ ] Verify data relationships with other tables
- [ ] Test with existing production data

## Notes

1. The application currently shows data from some tables (like fleet_vehicles with 70 rows) that don't have proper handlers or routes
2. Some tables might be views or legacy tables that should be removed
3. Priority should be given to tables with existing data
4. Consider creating a database diagram to understand relationships