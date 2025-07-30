# Fleet Management System - Boot Instructions

## Prerequisites Check

1. **Go Version**: Ensure Go 1.23+ is installed
   ```bash
   go version
   ```

2. **PostgreSQL Database**: Ensure your database is running and accessible

3. **Environment Variables**: Create a `.env` file if not in production:
   ```bash
   # Database connection
   DATABASE_URL=postgres://user:password@host:port/dbname?sslmode=require
   
   # Or individual variables
   PGHOST=your-db-host
   PGPORT=your-db-port
   PGUSER=your-db-user
   PGPASSWORD=your-db-password
   PGDATABASE=your-db-name
   
   # Application settings
   PORT=5000
   APP_ENV=development
   
   # Admin credentials (optional)
   ADMIN_USERNAME=admin
   ADMIN_PASSWORD=your-secure-password
   ```

## Boot Commands

### 1. Standard Boot
```bash
go run .
```

### 2. Build and Run
```bash
# Build the application
go build -o fleet-management

# Run the compiled binary
./fleet-management
```

### 3. Development Mode with Live Reload (if using air)
```bash
air
```

### 4. Production Mode
```bash
APP_ENV=production go run .
```

## Database Migrations

The application will automatically run migrations on startup, including:
- Database schema creation
- Performance indexes
- Database sync fixes (new)
- Invalid data cleanup (new)

### Manual Migration Commands
```bash
# Run migrations only
go run . migrate

# Run migrations with cleanup
go run . migrate cleanup

# Verify unused tables
go run . verify-unused

# Analyze table usage
go run . analyze-tables
```

## Startup Checklist

1. **Database Connection**: Watch for "Database connection successful!" in logs
2. **Migrations**: Look for "Database migrations completed successfully"
3. **Sync Fixes**: New! Look for "Fixing database sync issues..."
4. **Admin User**: Check for "Admin user ensured" message
5. **Server Start**: Look for "🚀 Server starting on port 5000"

## Expected Log Output

```
📝 Setting up log rotation...
🗄️  Setting up PostgreSQL database...
Initializing database connection...
Database connection successful!
Running database migrations...
Database migrations completed successfully
Ensuring admin user exists...
✅ Admin user ensured: username='admin'
Creating performance indexes...
Performance indexes creation completed
Fixing database sync issues...
Successfully completed migration: add_subject_to_notifications
Successfully completed migration: create_vehicle_health_checks_if_missing
Successfully completed migration: add_goals_to_ecse_services
Successfully completed migration: add_id_to_users
Database sync fixes completed successfully!
Cleaning up invalid data...
Data cleanup completed!
Database initialization complete!
🔐 Setting up session manager...
🚀 Setting up query cache...
🚀 Initializing advanced features...
💾 Initializing metrics storage...
📊 Starting metrics collection...
🔄 Resetting rate limiter...
📊 Starting metrics cleanup routine...
🚀 Server starting on port 5000
```

## Troubleshooting

### Database Connection Failed
- Check DATABASE_URL or individual PG* environment variables
- Verify database is running and accessible
- Check network connectivity to database

### Migration Errors
- Check logs for specific migration failures
- Migrations are designed to be safe and won't fail the startup
- Warnings about existing columns/tables are normal

### Port Already in Use
- Change PORT environment variable
- Or kill the process using port 5000:
  ```bash
  # Windows
  netstat -ano | findstr :5000
  taskkill /PID <PID> /F
  
  # Linux/Mac
  lsof -i :5000
  kill -9 <PID>
  ```

### Access the Application
Once booted successfully:
- Open browser to http://localhost:5000
- Login with admin credentials
- Check that pages load without errors

## Monitoring After Boot

1. **Check Error Logs**: Verify that previous errors are resolved:
   - No more "missing destination name subject"
   - No more "missing destination name mileage"
   - No more "missing destination name goals"
   - No more "missing destination name id"

2. **Test Key Functions**:
   - Login/logout
   - View fleet overview
   - Check maintenance records
   - View user management
   - Test ECSE services

3. **Performance Check**:
   - Pages should load quickly
   - No timeout errors
   - Database queries should be fast