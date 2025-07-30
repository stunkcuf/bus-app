# Bus Fleet Management System v2.0

A secure web-based fleet management system for managing bus routes, drivers, students, and maintenance records. Built with Go and PostgreSQL.

📚 **For complete documentation, see [PROJECT_DOCUMENTATION.md](PROJECT_DOCUMENTATION.md)**

## 🔒 Security Updates (v2.0)

This version includes significant security improvements:
- **Password Hashing**: All passwords are now stored using bcrypt hashing
- **CSRF Protection**: All forms include CSRF token validation
- **Input Sanitization**: All user inputs are sanitized to prevent XSS attacks
- **Secure Sessions**: Session management with secure cookies
- **Rate Limiting**: Login attempts are rate-limited to prevent brute force attacks

## 🚀 Getting Started

### Prerequisites

- Go 1.23.0 or higher
- PostgreSQL database (Railway or similar)
- Environment variables configured

### Environment Variables

```bash
# Database connection (Railway provides DATABASE_URL)
DATABASE_URL=postgres://user:password@host:port/dbname?sslmode=require

# Or individual PostgreSQL variables
PGHOST=your-db-host
PGPORT=your-db-port
PGUSER=your-db-user
PGPASSWORD=your-db-password
PGDATABASE=your-db-name

# Application port (optional, defaults to 5000)
PORT=5000
```

### Installation

1. Clone the repository:
```bash
git clone <your-repo-url>
cd bus-app
```

2. Install dependencies:
```bash
go mod download
```

3. **IMPORTANT**: Migrate existing passwords to bcrypt hashes:
```bash
go run migrate_passwords.go
```

This will:
- Convert all plain text passwords to secure bcrypt hashes
- Create a temporary admin account if needed (username: `temp_admin`, password: `TempAdmin123!`)

4. Run the application:
```bash
go run .
```

## 🔐 Default Login

If you're starting fresh or after migration:
- **Username**: `admin`
- **Password**: `adminpass` (will be hashed on first run)

Or if migration created a temporary admin:
- **Username**: `temp_admin`
- **Password**: `TempAdmin123!`

**Important**: Change these default passwords immediately after first login!

## 📋 Features

### For Managers
- **User Management**: Create and manage driver/manager accounts
- **Route Management**: Define routes and assign them to drivers
- **Fleet Management**: Manage buses and their maintenance status
- **Driver Monitoring**: View driver performance and logs
- **Import Vehicles**: Import buses from company fleet inventory

### For Drivers
- **Daily Logs**: Record trip details, mileage, and student attendance
- **Student Management**: Manage student roster with pickup/dropoff locations
- **Route Information**: View assigned route and bus details
- **Recent Activity**: Track recent trips and attendance

### Security Features
- Secure password storage with bcrypt hashing
- CSRF protection on all forms
- Input validation and sanitization
- Session timeout after 24 hours
- Rate limiting on login attempts
- Secure HTTP headers

## 🛠️ Technical Details

### Database Schema
The system uses PostgreSQL with the following main tables:
- `users` - System users (drivers and managers)
- `buses` - Bus fleet information
- `routes` - Route definitions
- `students` - Student information
- `route_assignments` - Driver-bus-route assignments
- `driver_logs` - Daily trip logs
- `bus_maintenance_logs` - Maintenance records
- `vehicles` - Company vehicle inventory
- `activities` - Special trips/activities

### Technology Stack
- **Backend**: Go 1.23.0
- **Database**: PostgreSQL
- **Frontend**: Bootstrap 5.3.0, Vanilla JavaScript
- **Security**: bcrypt, CSRF tokens, secure sessions
- **Deployment**: Railway (recommended)

## 🚦 API Endpoints

### Public Endpoints
- `GET /` - Login page
- `POST /` - Login submission
- `GET /health` - Health check

### Protected Endpoints (Require Authentication)
- `GET/POST /new-user` - Create new user (managers only)
- `GET/POST /edit-user` - Edit user (managers only)
- `GET /dashboard` - Auto-routes to appropriate dashboard
- `GET /manager-dashboard` - Manager dashboard
- `GET /driver-dashboard` - Driver dashboard
- `POST /save-log` - Save driver log
- `GET /students` - Student management (drivers)
- `POST /add-student` - Add new student
- `POST /edit-student` - Edit student
- `POST /remove-student` - Remove student
- And many more...

## 🔧 Maintenance

### Regular Tasks
1. **Monitor logs** for any security issues or errors
2. **Review user accounts** periodically
3. **Update passwords** regularly
4. **Backup database** regularly
5. **Check for Go security updates**

### Troubleshooting

#### "Invalid credentials" error after migration
Run the password migration script:
```bash
go run migrate_passwords.go
```

#### Session expired errors
Sessions expire after 24 hours. Users need to log in again.

#### CSRF token errors
Clear browser cookies and cache, then try again.

## 📱 Browser Support

- Chrome/Edge (latest)
- Firefox (latest)
- Safari (latest)
- Mobile browsers supported

## 🤝 Contributing

1. Always hash passwords using bcrypt
2. Include CSRF tokens in all forms
3. Validate and sanitize all inputs
4. Use prepared statements for database queries
5. Test security features before deploying

## 📄 License

This project is designed for internal use by non-profit organizations.

## 🆘 Support

For issues or questions:
1. Check the logs in your deployment platform
2. Verify environment variables are set correctly
3. Ensure database migrations have run
4. Verify password migration completed successfully

---

**Remember**: Security is paramount. Always use strong passwords, keep the system updated, and monitor for suspicious activity.
