# Environment Variables Documentation

This document describes all environment variables used by the Fleet Management System.

## Overview

The Fleet Management System uses environment variables for configuration to support different deployment environments (development, staging, production) without code changes. This follows the [12-Factor App](https://12factor.net/config) methodology.

## Core Application Variables

### Required Variables

#### `DATABASE_URL`
- **Type**: String (PostgreSQL connection string)
- **Required**: Yes
- **Description**: Complete PostgreSQL database connection string
- **Format**: `postgresql://username:password@host:port/database?options`
- **Example**: `postgresql://postgres:password@localhost:5432/fleet_management`
- **Railway**: Automatically provided by Railway PostgreSQL addon
- **Notes**: 
  - Must include `sslmode=require` for production deployments
  - Takes precedence over individual PG* variables

### Optional Core Variables

#### `APP_ENV`
- **Type**: String
- **Required**: No
- **Default**: `development`
- **Values**: `development`, `production`
- **Description**: Application environment mode
- **Effects**:
  - **Development**: Enables debug logging, detailed error pages, serves static files from `/static/`
  - **Production**: Optimized logging, generic error pages, serves built assets from `/dist/`
- **Example**: `APP_ENV=production`

#### `PORT`
- **Type**: Integer
- **Required**: No
- **Default**: `5000`
- **Description**: HTTP server port
- **Example**: `PORT=8080`
- **Railway**: Automatically set by Railway platform
- **Range**: 1024-65535 (recommended: 3000-9999)

## Database Configuration (Alternative)

If `DATABASE_URL` is not set, the system will attempt to use individual PostgreSQL variables:

#### `PGHOST`
- **Type**: String
- **Required**: Only if `DATABASE_URL` not set
- **Description**: PostgreSQL server hostname or IP address
- **Example**: `PGHOST=localhost`

#### `PGPORT`
- **Type**: Integer
- **Required**: Only if `DATABASE_URL` not set
- **Default**: `5432`
- **Description**: PostgreSQL server port
- **Example**: `PGPORT=5432`

#### `PGUSER`
- **Type**: String
- **Required**: Only if `DATABASE_URL` not set
- **Description**: PostgreSQL username
- **Example**: `PGUSER=postgres`

#### `PGPASSWORD`
- **Type**: String
- **Required**: Only if `DATABASE_URL` not set
- **Description**: PostgreSQL password
- **Example**: `PGPASSWORD=secretpassword`
- **Security**: Use strong passwords, avoid logging

#### `PGDATABASE`
- **Type**: String
- **Required**: Only if `DATABASE_URL` not set
- **Description**: PostgreSQL database name
- **Example**: `PGDATABASE=fleet_management`

## Security Variables

#### `ADMIN_USERNAME`
- **Type**: String
- **Required**: No
- **Default**: `admin`
- **Description**: Default admin user username created on first startup
- **Example**: `ADMIN_USERNAME=administrator`
- **Notes**: Only used if admin user doesn't exist

#### `ADMIN_PASSWORD`
- **Type**: String
- **Required**: Recommended for production
- **Description**: Password for default admin user
- **Example**: `ADMIN_PASSWORD=SecurePassword123!`
- **Security**: 
  - **CRITICAL**: Must be set in production
  - Use strong passwords (12+ characters, mixed case, numbers, symbols)
  - Admin user won't be created if not set
  - Change immediately after first login

#### `SESSION_SECRET`
- **Type**: String
- **Required**: Recommended for production
- **Description**: Secret key for session encryption and signing
- **Example**: `SESSION_SECRET=your-256-bit-random-secret-key`
- **Security**:
  - Should be cryptographically secure random string
  - Minimum 32 characters recommended
  - Change this if sessions are compromised
  - Keep this secret and rotate periodically

#### `SESSION_STORE_FILE`
- **Type**: String
- **Required**: No
- **Default**: `sessions.json`
- **Description**: File path for persistent session storage
- **Example**: `SESSION_STORE_FILE=/var/lib/fleet/sessions.json`
- **Notes**: 
  - File will be created automatically
  - Ensure directory is writable by application
  - Consider using absolute paths in production

## Performance and Feature Flags

#### `DISABLE_COMPRESSION`
- **Type**: Boolean (string)
- **Required**: No
- **Default**: `false` (compression enabled)
- **Values**: `true`, `false`
- **Description**: Disable HTTP response compression
- **Example**: `DISABLE_COMPRESSION=true`
- **Use Cases**: 
  - Disable if using external compression (nginx, CDN)
  - Debug network issues
  - Reduce CPU usage in high-traffic scenarios

#### `LOG_LEVEL`
- **Type**: String
- **Required**: No
- **Default**: `INFO`
- **Values**: `DEBUG`, `INFO`, `WARN`, `ERROR`, `FATAL`
- **Description**: Minimum logging level
- **Example**: `LOG_LEVEL=DEBUG`
- **Effects**:
  - **DEBUG**: All messages including debug info
  - **INFO**: General application information
  - **WARN**: Warning messages and above
  - **ERROR**: Error messages and above
  - **FATAL**: Only fatal errors

## Frontend Build Variables

#### `NODE_ENV`
- **Type**: String
- **Required**: No (frontend build only)
- **Default**: `development`
- **Values**: `development`, `production`
- **Description**: Node.js environment for frontend build process
- **Example**: `NODE_ENV=production`
- **Usage**: Controls webpack build optimization

## Environment-Specific Configurations

### Development Environment

```bash
# Basic development setup
APP_ENV=development
DATABASE_URL=postgresql://postgres:password@localhost:5432/fleet_development
PORT=5000
LOG_LEVEL=DEBUG

# Optional for testing
ADMIN_USERNAME=admin
ADMIN_PASSWORD=devpassword123
SESSION_SECRET=dev-session-secret-not-for-production
```

### Production Environment

```bash
# Essential production variables
APP_ENV=production
DATABASE_URL=postgresql://username:password@host:port/database?sslmode=require
PORT=5000

# Security (REQUIRED)
ADMIN_USERNAME=administrator
ADMIN_PASSWORD=CHANGE_ME_SECURE_PASSWORD_123!
SESSION_SECRET=cryptographically-secure-random-256-bit-key

# Logging
LOG_LEVEL=INFO

# Performance
# DISABLE_COMPRESSION=true  # If using nginx compression
```

### Railway Deployment

```bash
# Railway automatically provides
DATABASE_URL=postgresql://...
PORT=5000

# You must set these
APP_ENV=production
ADMIN_PASSWORD=YourSecurePassword123!
SESSION_SECRET=your-secure-session-secret
```

## Validation and Defaults

The application validates environment variables at startup:

### Validation Rules

1. **Database Connection**: `DATABASE_URL` or complete PG* set required
2. **Port Range**: PORT must be 1-65535 if specified
3. **Admin Security**: Warning logged if `ADMIN_PASSWORD` not set
4. **Environment**: Only `development` and `production` supported for `APP_ENV`

### Default Behaviors

- Missing optional variables use documented defaults
- Invalid values fallback to defaults with warnings
- Critical missing variables cause startup failure

## Security Best Practices

### Secret Management

1. **Never commit secrets to version control**
   ```bash
   # Good: Use .env files (gitignored)
   echo "ADMIN_PASSWORD=secret" >> .env
   
   # Bad: Hardcode in source
   const password = "hardcoded"; // DON'T DO THIS
   ```

2. **Use strong passwords and secrets**
   ```bash
   # Generate secure session secret
   openssl rand -hex 32
   
   # Generate strong password
   openssl rand -base64 32
   ```

3. **Rotate secrets regularly**
   - Change `SESSION_SECRET` invalidates all sessions
   - Change `ADMIN_PASSWORD` and update admin user

### Production Deployment

1. **Use environment-specific values**
   ```bash
   # Development
   DATABASE_URL=postgresql://localhost/fleet_dev
   
   # Production  
   DATABASE_URL=postgresql://prod-host/fleet_prod
   ```

2. **Set secure file permissions**
   ```bash
   chmod 600 .env
   chown app:app .env
   ```

3. **Use secret management services**
   - Railway: Environment variables in dashboard
   - Docker: Docker secrets or environment files
   - Kubernetes: ConfigMaps and Secrets

## Troubleshooting

### Common Issues

#### Database Connection Failed
```bash
# Check variables
echo $DATABASE_URL
echo $PGHOST $PGPORT $PGUSER $PGDATABASE

# Test connection
psql $DATABASE_URL -c "SELECT version();"
```

#### Admin User Not Created
```bash
# Check if password is set
echo $ADMIN_PASSWORD

# Set password and restart
export ADMIN_PASSWORD="SecurePassword123"
./hs-bus
```

#### Sessions Not Persisting
```bash
# Check session file permissions
ls -la sessions.json

# Check custom session file
echo $SESSION_STORE_FILE
ls -la $SESSION_STORE_FILE
```

#### Port Already in Use
```bash
# Check if port is available
netstat -tlnp | grep :5000

# Use different port
export PORT=5001
```

### Debug Mode

Enable debug logging to troubleshoot configuration:

```bash
export LOG_LEVEL=DEBUG
export APP_ENV=development
./hs-bus
```

## Configuration Templates

### `.env` File Template

```bash
# .env - Copy and customize for your environment
# Do not commit this file to version control

# Application
APP_ENV=development
PORT=5000
LOG_LEVEL=INFO

# Database
DATABASE_URL=postgresql://postgres:password@localhost:5432/fleet_management

# Security (CHANGE THESE)
ADMIN_USERNAME=admin
ADMIN_PASSWORD=CHANGE_ME_SECURE_PASSWORD
SESSION_SECRET=CHANGE_ME_RANDOM_256_BIT_SECRET

# Optional
SESSION_STORE_FILE=sessions.json
DISABLE_COMPRESSION=false
```

### Docker Compose Template

```yaml
version: '3.8'
services:
  app:
    environment:
      - APP_ENV=production
      - DATABASE_URL=postgresql://postgres:password@db:5432/fleet
      - PORT=5000
      - ADMIN_USERNAME=admin
      - ADMIN_PASSWORD=SecurePassword123!
      - SESSION_SECRET=secure-random-session-secret
      - LOG_LEVEL=INFO
```

### Railway Template

Set these in Railway Dashboard → Variables:

```
APP_ENV=production
ADMIN_USERNAME=admin
ADMIN_PASSWORD=SecurePassword123!
SESSION_SECRET=cryptographically-secure-random-key
LOG_LEVEL=INFO
```

## Migration and Updates

### Changing Database

```bash
# Backup current data
pg_dump $OLD_DATABASE_URL > backup.sql

# Update variable
export DATABASE_URL=$NEW_DATABASE_URL

# Import data
psql $DATABASE_URL < backup.sql

# Restart application
./hs-bus
```

### Environment Migration

```bash
# Development to production
export APP_ENV=production
export LOG_LEVEL=INFO
# Update other variables...

# Test configuration
./hs-bus --validate-config  # If implemented
```

## Monitoring and Observability

### Health Check Endpoint

The application provides a health check endpoint that verifies configuration:

```bash
curl http://localhost:5000/health
```

Response includes database connectivity status based on configured variables.

### Logging Configuration Status

At startup, the application logs configuration status:

```
[INFO] Starting Fleet Management System
[INFO] Environment: production
[INFO] Database: Connected to PostgreSQL
[INFO] Admin user: Enabled
[INFO] Session storage: File-based (sessions.json)
[INFO] Compression: Enabled
[INFO] Log level: INFO
[INFO] Server starting on port 5000
```

## External References

- [12-Factor App Configuration](https://12factor.net/config)
- [PostgreSQL Connection Strings](https://www.postgresql.org/docs/current/libpq-connect.html#LIBPQ-CONNSTRING)
- [Railway Environment Variables](https://docs.railway.app/deploy/variables)
- [Go Environment Variables](https://golang.org/pkg/os/#Getenv)