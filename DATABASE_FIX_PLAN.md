# Database Fix Plan - Fleet Management System

## Executive Summary
The database has significant structural issues:
- 17 out of 30 tables (57%) are empty
- Multiple overlapping tables for vehicles and maintenance
- Generic column names (unnamed_0 through unnamed_13)
- Inconsistent foreign key references (vehicle_id vs bus_id)

## Phase 1: Data Consolidation (Priority: HIGH)

### 1.1 Vehicle Tables Consolidation
**Current State:**
- `buses` (10 rows) - Basic bus data
- `vehicles` (44 rows) - Company vehicles 
- `fleet_vehicles` (70 rows) - Detailed vehicle data with proper columns
- `school_buses` (0 rows) - Empty duplicate
- `agency_vehicles` (0 rows) - Empty duplicate
- `all_vehicle_mileage` (0 rows) - Empty duplicate

**Action Plan:**
1. **PRIMARY TABLE**: Use `fleet_vehicles` as the master vehicle table
2. **MERGE**: Import data from `buses` and `vehicles` into `fleet_vehicles`
3. **STANDARDIZE**: Add `vehicle_type` column ('bus', 'van', 'car', etc.)
4. **DELETE**: Remove empty tables: `school_buses`, `agency_vehicles`, `all_vehicle_mileage`
5. **UPDATE**: All foreign keys to reference `fleet_vehicles.vehicle_number`

### 1.2 Maintenance Tables Consolidation
**Current State:**
- `maintenance_records` (409 rows) - Active, has real data
- `service_records` (55 rows) - Has data but with unnamed columns
- `maintenance_sheets` (10 rows) - Minimal data, unnamed columns
- `bus_maintenance_logs` (0 rows) - Empty
- `vehicle_maintenance_logs` (0 rows) - Empty

**Action Plan:**
1. **PRIMARY TABLE**: Use `maintenance_records` as master
2. **TRANSFORM**: Convert `service_records` data:
   - Map unnamed_0 → vehicle_description
   - Map unnamed_1 → vehicle_number
   - Map unnamed_3 → service_mileage
   - Map unnamed_8 → next_service_mileage
3. **MERGE**: Import transformed data into `maintenance_records`
4. **DELETE**: Remove empty tables and merged tables

### 1.3 Mileage Tables Consolidation
**Current State:**
- `monthly_mileage_reports` (1723 rows) - Active data
- `mileage_reports` (0 rows) - Empty
- `mileage_records` (0 rows) - Empty
- `driver_logs` (1 row) - Minimal data

**Action Plan:**
1. **PRIMARY TABLE**: Use `monthly_mileage_reports`
2. **ENHANCE**: Add missing columns for daily tracking
3. **DELETE**: Remove empty mileage tables

## Phase 2: Schema Improvements (Priority: HIGH)

### 2.1 Fix Column Names
**Tables with unnamed columns:**
- `service_records`: 14 unnamed columns
- `maintenance_sheets`: 4 unnamed columns

**Action Plan:**
1. Create mapping tables before transformation
2. Rename columns based on sample data analysis
3. Validate data after renaming

### 2.2 Standardize Foreign Keys
**Current Issues:**
- Some tables use `vehicle_id`, others use `bus_id`
- No consistent naming convention

**Action Plan:**
1. Standardize on `vehicle_id` for all vehicle references
2. Update all foreign key columns
3. Add proper constraints

## Phase 3: Empty Table Cleanup (Priority: MEDIUM)

### 3.1 Tables to Remove (17 empty tables)
```sql
-- ECSE related (keep structure for future use)
ecse_services
ecse_assessments  
ecse_attendance

-- Remove completely (redundant)
school_buses
agency_vehicles
all_vehicle_mileage
bus_maintenance_logs
vehicle_maintenance_logs
mileage_reports
mileage_records

-- System tables (keep for future features)
sessions
import_history
import_errors
scheduled_exports
activities
program_staff
```

## Phase 4: Data Migration Scripts

### 4.1 Vehicle Consolidation Script
```sql
-- Step 1: Add vehicle_type to fleet_vehicles
ALTER TABLE fleet_vehicles ADD COLUMN IF NOT EXISTS vehicle_type VARCHAR(50);

-- Step 2: Import buses data
INSERT INTO fleet_vehicles (vehicle_number, make, model, year, license, vehicle_type)
SELECT 
    CAST(SUBSTRING(bus_id FROM '\d+') AS INTEGER),
    CASE 
        WHEN model LIKE '%Ford%' THEN 'Ford'
        WHEN model LIKE '%Chevy%' THEN 'Chevrolet'
        ELSE 'Unknown'
    END,
    model,
    EXTRACT(YEAR FROM CURRENT_DATE),
    bus_id,
    'bus'
FROM buses
WHERE bus_id NOT IN (SELECT license FROM fleet_vehicles WHERE license IS NOT NULL);

-- Step 3: Import vehicles data
INSERT INTO fleet_vehicles (vehicle_number, make, model, year, description, vehicle_type)
SELECT 
    CAST(SUBSTRING(vehicle_id FROM '\d+') AS INTEGER),
    SUBSTRING(model FROM '^\w+'),
    model,
    CAST(year AS INTEGER),
    description,
    'vehicle'
FROM vehicles
WHERE vehicle_id NOT IN (SELECT license FROM fleet_vehicles WHERE license IS NOT NULL);
```

### 4.2 Maintenance Consolidation Script
```sql
-- Transform service_records to maintenance_records
INSERT INTO maintenance_records (
    vehicle_number,
    service_date,
    mileage,
    work_description,
    created_at
)
SELECT 
    CAST(unnamed_1 AS INTEGER),
    CURRENT_DATE,
    CAST(NULLIF(unnamed_3, '') AS INTEGER),
    CONCAT('Service needed at ', unnamed_8, ' miles'),
    CURRENT_TIMESTAMP
FROM service_records
WHERE unnamed_1 ~ '^\d+$';
```

## Phase 5: Application Updates

### 5.1 Model Updates
- Update Go structs to match new schema
- Remove references to deleted tables
- Standardize on `vehicle_id` field names

### 5.2 Handler Updates
- Update all queries to use consolidated tables
- Remove handlers for deleted tables
- Add proper error handling for migrations

## Implementation Timeline

### Week 1: Analysis & Backup
- [x] Complete database analysis
- [ ] Create full database backup
- [ ] Document current table relationships
- [ ] Create rollback plan

### Week 2: Consolidation
- [ ] Execute vehicle table consolidation
- [ ] Execute maintenance table consolidation
- [ ] Execute mileage table consolidation
- [ ] Verify data integrity

### Week 3: Schema Updates
- [ ] Fix column naming issues
- [ ] Standardize foreign keys
- [ ] Add missing indexes
- [ ] Update constraints

### Week 4: Application Updates
- [ ] Update Go models
- [ ] Update database queries
- [ ] Test all functionality
- [ ] Deploy updates

## Risk Mitigation

### Backup Strategy
1. Full database backup before each phase
2. Transaction-based migrations
3. Ability to rollback each phase independently

### Testing Plan
1. Create test database with production data copy
2. Run all migrations on test first
3. Validate data integrity after each step
4. User acceptance testing before production

## Success Metrics
- Reduce table count from 30 to ~15
- 100% of tables have proper column names
- 100% consistent foreign key references
- 0 empty tables (except planned future features)
- All application features working with new schema