# Database Migration Execution Plan

## Overview
This document provides step-by-step instructions for executing database migrations required for the new features.

## Pre-Migration Checklist

- [ ] **BACKUP DATABASE** - Critical before any migration
- [ ] Verify database connection
- [ ] Check current schema state
- [ ] Review migration scripts
- [ ] Plan rollback strategy

## Migration Files

### Required Migrations (in order):
1. `consolidate_vehicles_tables.sql` - Vehicle table consolidation
2. `004_advanced_features.sql` - New feature tables
3. `012_create_import_history.sql` - Import tracking

## Step-by-Step Execution

### Step 1: Backup Current Database
```bash
# Local backup
pg_dump -h localhost -U postgres -d fleet_management > backup_$(date +%Y%m%d_%H%M%S).sql

# Railway backup (if applicable)
railway run pg_dump $DATABASE_URL > backup_$(date +%Y%m%d_%H%M%S).sql
```

### Step 2: Verify Current Schema
```sql
-- Check existing tables
SELECT table_name 
FROM information_schema.tables 
WHERE table_schema = 'public' 
ORDER BY table_name;

-- Check migrations table
SELECT * FROM migrations ORDER BY executed_at DESC;
```

### Step 3: Execute Migrations

#### Option A: Automatic (Using Go Migration System)
```bash
# The application will run migrations on startup
go run .

# Or in production
./fleet-management
```

#### Option B: Manual Execution
```bash
# Connect to database
psql $DATABASE_URL

# Run each migration
\i migrations/consolidate_vehicles_tables.sql
\i migrations/004_advanced_features.sql
\i migrations/012_create_import_history.sql

# Record in migrations table
INSERT INTO migrations (filename, executed_at, success) 
VALUES 
  ('consolidate_vehicles_tables.sql', NOW(), true),
  ('004_advanced_features.sql', NOW(), true),
  ('012_create_import_history.sql', NOW(), true);
```

### Step 4: Verify Migration Success

```sql
-- Check new tables exist
SELECT COUNT(*) as new_tables FROM information_schema.tables 
WHERE table_schema = 'public' 
AND table_name IN (
  'driver_locations',
  'driver_status',
  'emergency_reports',
  'student_attendance',
  'notifications',
  'notification_preferences',
  'notification_queue',
  'notification_log',
  'budget_categories',
  'budget_entries',
  'gps_locations',
  'gps_history',
  'import_history',
  'api_tokens',
  'websocket_connections'
);

-- Should return: 15

-- Verify no data loss
SELECT 
  (SELECT COUNT(*) FROM buses) as bus_count,
  (SELECT COUNT(*) FROM vehicles) as vehicle_count,
  (SELECT COUNT(*) FROM students) as student_count,
  (SELECT COUNT(*) FROM users) as user_count;
```

### Step 5: Test Application

```bash
# Run application and check logs
go run .

# Test key endpoints
curl http://localhost:8080/health
curl http://localhost:8080/fleet (requires auth)
```

## Rollback Plan

### If Migration Fails:

1. **Stop Application Immediately**
   ```bash
   # Kill the process
   pkill fleet-management
   ```

2. **Restore from Backup**
   ```bash
   # Drop and recreate database
   psql -c "DROP DATABASE fleet_management;"
   psql -c "CREATE DATABASE fleet_management;"
   
   # Restore backup
   psql fleet_management < backup_YYYYMMDD_HHMMSS.sql
   ```

3. **Verify Restoration**
   ```sql
   -- Check tables restored
   SELECT COUNT(*) FROM information_schema.tables 
   WHERE table_schema = 'public';
   
   -- Check data restored
   SELECT 
     (SELECT COUNT(*) FROM buses) as buses,
     (SELECT COUNT(*) FROM vehicles) as vehicles;
   ```

## Post-Migration Tasks

1. **Update Application Configuration**
   - Set new environment variables
   - Enable feature flags
   - Configure email/SMS settings

2. **Initialize New Features**
   ```sql
   -- Add default notification preferences for existing users
   INSERT INTO notification_preferences (user_id, email_enabled, sms_enabled)
   SELECT username, true, false FROM users
   ON CONFLICT DO NOTHING;
   
   -- Create default budget categories
   INSERT INTO budget_categories (name, description, annual_budget) VALUES
   ('Fuel', 'Fuel costs for all vehicles', 50000),
   ('Maintenance', 'Regular maintenance and repairs', 30000),
   ('Insurance', 'Vehicle insurance', 20000),
   ('Other', 'Miscellaneous expenses', 10000);
   ```

3. **Test New Features**
   - Send test notification
   - Create test budget entry
   - Verify GPS tracking endpoint
   - Test mobile API authentication

## Migration Status Tracking

```sql
-- View migration history
SELECT 
  filename,
  executed_at,
  success,
  error_message
FROM migrations
ORDER BY executed_at DESC;

-- Check table creation dates
SELECT 
  tablename,
  obj_description(oid) as description
FROM pg_tables t
JOIN pg_class c ON c.relname = t.tablename
WHERE schemaname = 'public'
ORDER BY c.relcreatedat DESC
LIMIT 20;
```

## Troubleshooting

### Common Issues:

1. **Permission Denied**
   - Ensure database user has CREATE TABLE permission
   - Grant: `GRANT ALL ON DATABASE fleet_management TO your_user;`

2. **Table Already Exists**
   - Check if partial migration occurred
   - Drop specific tables and retry

3. **Foreign Key Violations**
   - Ensure migrations run in correct order
   - Check data integrity before migration

4. **Connection Timeout**
   - Increase statement timeout for large migrations
   - `SET statement_timeout = '10min';`

## Success Criteria

- [ ] All 15 new tables created
- [ ] No data loss in existing tables
- [ ] Application starts without errors
- [ ] Fleet page shows all 54 vehicles
- [ ] Health check endpoint responds 200 OK
- [ ] No errors in application logs

## Emergency Contacts

- Database Admin: [Contact Info]
- System Admin: [Contact Info]
- On-call Developer: [Contact Info]

---

**Created**: January 27, 2025
**Last Updated**: January 27, 2025
**Status**: Ready for Execution