# Database Migration Execution Guide

## Overview
This guide provides step-by-step instructions to fix the database issues identified in the comprehensive analysis.

## Current Issues Summary
- **17 empty tables** (57% of all tables)
- **Multiple overlapping vehicle tables** (buses, vehicles, fleet_vehicles)
- **4 maintenance-related tables** with only 1 containing data
- **Unnamed columns** in service_records and maintenance_sheets
- **Inconsistent foreign keys** (vehicle_id vs bus_id)

## Migration Scripts Created
1. `01_backup_database.go` - Creates full backup
2. `02_consolidate_vehicles.go` - Consolidates vehicle tables
3. `03_consolidate_maintenance.go` - Consolidates maintenance tables
4. `04_cleanup_database.go` - Removes empty tables and fixes issues
5. `05_verify_migration.go` - Validates migration success

## Execution Steps

### Step 1: Create Backup (MANDATORY)
```bash
cd utilities
go run 01_backup_database.go
```
This creates a timestamped backup directory with:
- Table structure SQL files
- Table data CSV files
- Backup summary

**Expected output:**
- Creates `backup_YYYYMMDD_HHMMSS/` directory
- Backs up all 30 tables
- Total time: ~2-3 minutes

### Step 2: Run Database Analysis (Optional)
```bash
go run database_comprehensive_analysis.go > analysis_before.txt
```
Save the current state for comparison.

### Step 3: Consolidate Vehicle Tables
```bash
go run 02_consolidate_vehicles.go
```
This will:
- Add `vehicle_type` column to fleet_vehicles
- Import 10 buses → fleet_vehicles
- Import 44 vehicles → fleet_vehicles
- Set vehicle types (bus, car, van, truck, suv)

**Expected results:**
- fleet_vehicles increases from 70 to ~124 records
- All vehicles have proper types assigned
- Old tables remain for verification

### Step 4: Consolidate Maintenance Tables
```bash
go run 03_consolidate_maintenance.go
```
This will:
- Import service_records (55) → maintenance_records
- Import useful maintenance_sheets data → maintenance_records
- Preserve existing 409 maintenance records

**Expected results:**
- maintenance_records increases from 409 to ~470 records
- Service intervals and mileage data preserved

### Step 5: Clean Up Database
```bash
go run 04_cleanup_database.go
```
**WARNING:** This permanently deletes empty tables!

This will:
- Drop 7 empty redundant tables
- Standardize foreign keys (bus_id → vehicle_id)
- Create compatibility views
- Add performance indexes

**User confirmation required** before deletion.

### Step 6: Verify Migration
```bash
go run 05_verify_migration.go
```
This runs comprehensive checks:
- Vehicle consolidation success
- Maintenance consolidation success
- Column naming issues
- Foreign key consistency
- Data integrity
- Empty table count

**Expected output:**
- All checks should show PASS or INFO
- Warnings are acceptable (can be addressed later)
- No ERROR status should appear

## Post-Migration Application Updates

### 1. Update Go Models
```go
// In models.go - Update to use fleet_vehicles
type FleetVehicle struct {
    ID            int    `db:"id"`
    VehicleNumber int    `db:"vehicle_number"`
    VehicleType   string `db:"vehicle_type"` // New field
    // ... rest of fields
}
```

### 2. Update Handlers
Replace references:
- `buses` table → `fleet_vehicles WHERE vehicle_type = 'bus'`
- `vehicles` table → `fleet_vehicles WHERE vehicle_type != 'bus'`
- `bus_id` → `vehicle_id` in all queries

### 3. Update Routes
Consolidate vehicle management:
- `/fleet` - Shows all fleet_vehicles
- `/fleet?type=bus` - Shows only buses
- `/fleet?type=vehicle` - Shows non-bus vehicles

## Rollback Plan

If issues occur:

### Option 1: Restore from backup
```bash
# Use the backup files created in Step 1
# Restore each table from CSV files
```

### Option 2: Use compatibility views
The migration creates views that maintain old table names:
- `buses` view → maps to fleet_vehicles
- `vehicles` view → maps to fleet_vehicles

### Option 3: Partial rollback
Keep consolidated data but recreate old table structure if needed.

## Testing Checklist

After migration, test:
- [ ] Login functionality works
- [ ] Vehicle/bus lists display correctly
- [ ] Maintenance records show for all vehicles
- [ ] ECSE students display (825 records)
- [ ] Can add new vehicles
- [ ] Can add new maintenance records
- [ ] Reports generate correctly
- [ ] No error messages in logs

## Timeline Estimate

- Backup: 5 minutes
- Vehicle consolidation: 2 minutes
- Maintenance consolidation: 3 minutes
- Cleanup: 2 minutes
- Verification: 1 minute
- **Total: ~15 minutes**

Plus testing: 30-60 minutes

## Success Metrics

Before migration:
- 30 tables (17 empty)
- Duplicate vehicle data across 3 tables
- Unnamed columns in 2 tables
- Inconsistent foreign keys

After migration:
- ~23 tables (10 or fewer empty)
- Single vehicle table (fleet_vehicles)
- No unnamed columns
- Consistent vehicle_id references
- All data preserved and accessible

## Emergency Contact

If critical issues arise:
1. Stop the migration immediately
2. Check the backup files are intact
3. Document the error messages
4. Restore from backup if needed

## Next Steps After Success

1. Monitor application for 24 hours
2. Remove old table references from code
3. Drop compatibility views after 1 week
4. Document new database schema
5. Update API documentation