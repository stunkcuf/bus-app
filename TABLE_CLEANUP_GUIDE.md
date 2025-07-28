# Database Table Cleanup Guide

## Overview
This guide documents the process for cleaning up unused database tables in the Fleet Management System.

## Commands Available

### 1. Verify Unused Tables
```bash
./hs-bus verify-unused
```
This command will:
- Check all suspected unused tables
- Count rows in each table
- Check for foreign key dependencies
- Generate a summary report

### 2. Analyze Table Usage
```bash
./hs-bus analyze-tables
```
This command provides detailed statistics:
- Table sizes
- Row counts
- Insert/Update/Delete activity
- Last vacuum and analyze times

### 3. Cleanup Tables (Dry Run)
```bash
./hs-bus cleanup-tables
```
This runs in dry-run mode by default and will:
- Show which tables would be removed
- Verify tables are empty before removal
- Check for dependent views

### 4. Cleanup Tables (Force)
```bash
./hs-bus cleanup-tables --force
```
This will actually remove the unused tables.

## Tables Identified for Removal

### Legacy Import System Tables
These tables are from the old import system that has been replaced:
- `import_logs`
- `import_mappings`
- `import_templates`
- `data_imports`
- `import_history`
- `import_configurations`
- `excel_imports`

### Migrated Tables
- `fleet_vehicles` - Data migrated to `vehicles` table with `vehicle_type = 'fleet'`

## Tables to Keep

### ECSE Tables (In Active Use)
- `ecse_services` - Used by ECSE service tracking
- `ecse_assessments` - Used for ECSE student assessments
- `ecse_attendance` - Used for ECSE attendance tracking
- `ecse_students` - Main ECSE student table

## Cleanup Process

1. **First, run the vehicle consolidation migration:**
   ```bash
   ./hs-bus migrate
   ```

2. **Verify unused tables:**
   ```bash
   ./hs-bus verify-unused
   ```

3. **Analyze table usage:**
   ```bash
   ./hs-bus analyze-tables
   ```

4. **Run cleanup in dry-run mode:**
   ```bash
   ./hs-bus cleanup-tables
   ```

5. **If everything looks good, run with --force:**
   ```bash
   ./hs-bus cleanup-tables --force
   ```

## Post-Cleanup Verification

After cleanup, verify the system is working correctly:
1. Check all pages load without errors
2. Verify import functionality works with the new unified system
3. Check that ECSE features continue to work
4. Monitor logs for any database errors

## Rollback Plan

If issues arise after cleanup:
1. The `fleet_vehicles` data is preserved in the migration backup
2. Import tables can be recreated from the database.go schema if needed
3. Always backup the database before running cleanup commands