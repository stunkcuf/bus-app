# Fleet Management System - Claude Development Guide

## Project Overview

The Fleet Management System is a comprehensive web-based platform for managing school transportation operations, including bus fleets, driver assignments, student ridership, and vehicle maintenance.

### Key Value Propositions
- **Operational Efficiency**: Streamlined route assignments and driver management
- **Safety & Compliance**: Comprehensive maintenance tracking and student attendance monitoring
- **Cost Management**: Detailed mileage tracking and fuel cost analysis
- **Special Education Support**: Dedicated ECSE student tracking
- **Data-Driven Insights**: Comprehensive reporting and analytics

## Technical Architecture

### Current Tech Stack
- **Backend**: Go (Golang) with standard `net/http` library
- **Database**: PostgreSQL with `sqlx` and `lib/pq` driver
- **Frontend**: Server-side rendered HTML templates with vanilla JavaScript
- **Authentication**: Session-based with secure cookies and CSRF tokens
- **File Processing**: Excelize for Excel import/export
- **Security**: bcrypt for password hashing, custom session management
- **Deployment**: Railway.app with PostgreSQL

### Key Technical Implementation
- CSRF protection implemented via session tokens
- Role-based access control (manager/driver roles)
- Session-based authentication with 24-hour expiration
- Server-side rendering with Go html/template
- Responsive design using inline CSS and minimal JavaScript
- Rate limiting on login attempts
- Secure headers middleware (CSP, HSTS, etc.)

## Core Modules & Features

### 1. User Management & Authentication
```go
type User struct {
    Username         string    `json:"username" db:"username"`
    Password         string    `json:"password,omitempty" db:"password"`
    Role             string    `json:"role" db:"role"`              // "manager" or "driver"
    Status           string    `json:"status" db:"status"`          // "active" or "pending"
    RegistrationDate string    `json:"registration_date" db:"registration_date"`
    CreatedAt        time.Time `json:"created_at" db:"created_at"`
}
```

**Key Features:**
- Self-service registration with manager approval workflow
- Bcrypt password hashing (cost factor: 12)
- Session-based authentication with 24-hour expiration
- CSRF token protection on all forms
- Rate limiting on login attempts (5 per 15 minutes)

### 2. Fleet & Vehicle Management
```go
type Bus struct {
    BusID            string    `json:"bus_id" db:"bus_id"`
    Status           string    `json:"status" db:"status"`
    Model            string    `json:"model" db:"model"`
    Capacity         int       `json:"capacity" db:"capacity"`
    OilStatus        string    `json:"oil_status" db:"oil_status"`
    TireStatus       string    `json:"tire_status" db:"tire_status"`
    MaintenanceNotes string    `json:"maintenance_notes" db:"maintenance_notes"`
    UpdatedAt        time.Time `json:"updated_at" db:"updated_at"`
}

type Vehicle struct {
    VehicleID        string    `json:"vehicle_id" db:"vehicle_id"`
    Model            string    `json:"model" db:"model"`
    Description      string    `json:"description" db:"description"`
    Year             int       `json:"year" db:"year"`
    TireSize         string    `json:"tire_size" db:"tire_size"`
    License          string    `json:"license" db:"license"`
    OilStatus        string    `json:"oil_status" db:"oil_status"`
    TireStatus       string    `json:"tire_status" db:"tire_status"`
    Status           string    `json:"status" db:"status"`
    MaintenanceNotes string    `json:"maintenance_notes" db:"maintenance_notes"`
    SerialNumber     string    `json:"serial_number" db:"serial_number"`
    Base             string    `json:"base" db:"base"`
    ServiceInterval  int       `json:"service_interval" db:"service_interval"`
}

type BusMaintenanceLog struct {
    ID        int       `json:"id" db:"id"`
    BusID     string    `json:"bus_id" db:"bus_id"`
    Date      string    `json:"date" db:"date"`
    Category  string    `json:"category" db:"category"`
    Notes     string    `json:"notes" db:"notes"`
    Mileage   int       `json:"mileage" db:"mileage"`
    Cost      float64   `json:"cost" db:"cost"`
    CreatedAt time.Time `json:"created_at" db:"created_at"`
}
```

**Visual Status Indicators:**
- Green: Active/Good condition
- Yellow: Maintenance due soon  
- Red: Out of service/Immediate attention needed

### 3. Student Management
```go
type Student struct {
    StudentID      string     `json:"student_id" db:"student_id"`
    Name           string     `json:"name" db:"name"`
    Locations      []Location `json:"locations" db:"locations"`        // JSONB
    PhoneNumber    string     `json:"phone_number" db:"phone_number"`
    AltPhoneNumber string     `json:"alt_phone_number" db:"alt_phone_number"`
    Guardian       string     `json:"guardian" db:"guardian"`
    PickupTime     string     `json:"pickup_time" db:"pickup_time"`    // TIME
    DropoffTime    string     `json:"dropoff_time" db:"dropoff_time"`  // TIME
    PositionNumber int        `json:"position_number" db:"position_number"`
    RouteID        string     `json:"route_id" db:"route_id"`
    Driver         string     `json:"driver" db:"driver"`
    Active         bool       `json:"active" db:"active"`
}

type Location struct {
    LocationID  string `json:"location_id"`
    Type        string `json:"type"`        // "pickup" or "dropoff"
    Address     string `json:"address"`
    Description string `json:"description"`
}
```

### 4. Route Management
```go
type Route struct {
    RouteID     string `json:"route_id" db:"route_id"`
    RouteName   string `json:"route_name" db:"route_name"`
    Description string `json:"description" db:"description"`
    Positions   []struct {
        Position int    `json:"position"`
        Student  string `json:"student"`
    } `json:"positions" db:"positions"` // JSONB
}

type RouteAssignment struct {
    Driver       string `json:"driver" db:"driver"`
    BusID        string `json:"bus_id" db:"bus_id"`
    RouteID      string `json:"route_id" db:"route_id"`
    RouteName    string `json:"route_name" db:"route_name"`
    AssignedDate string `json:"assigned_date" db:"assigned_date"`
}
```

**Validation Rules:**
- One driver per bus per route
- No double assignments
- Routes cannot be modified while assigned

### 5. Daily Operations
```go
type DriverLog struct {
    ID         int                 `json:"id" db:"id"`
    Driver     string              `json:"driver" db:"driver"`
    BusID      string              `json:"bus_id" db:"bus_id"`
    RouteID    string              `json:"route_id" db:"route_id"`
    Date       string              `json:"date" db:"date"`
    Period     string              `json:"period" db:"period"` // "morning" or "afternoon"
    Departure  string              `json:"departure_time" db:"departure_time"`
    Arrival    string              `json:"arrival_time" db:"arrival_time"`
    Mileage    float64             `json:"mileage" db:"mileage"`
    Attendance []StudentAttendance `json:"attendance"` // JSONB
    CreatedAt  time.Time           `json:"created_at" db:"created_at"`
}

type StudentAttendance struct {
    Position   int    `json:"position"`
    Present    bool   `json:"present"`
    PickupTime string `json:"pickup_time"`
}
```

### 6. ECSE (Special Education) Support
```go
type ECSEStudent struct {
    StudentID              string    `json:"student_id" db:"student_id"`
    FirstName              string    `json:"first_name" db:"first_name"`
    LastName               string    `json:"last_name" db:"last_name"`
    DateOfBirth            string    `json:"date_of_birth" db:"date_of_birth"`
    Grade                  string    `json:"grade" db:"grade"`
    EnrollmentStatus       string    `json:"enrollment_status" db:"enrollment_status"`
    IEPStatus              string    `json:"iep_status" db:"iep_status"`
    PrimaryDisability      string    `json:"primary_disability" db:"primary_disability"`
    ServiceMinutes         int       `json:"service_minutes" db:"service_minutes"`
    TransportationRequired bool      `json:"transportation_required" db:"transportation_required"`
    BusRoute               string    `json:"bus_route" db:"bus_route"`
    ParentName             string    `json:"parent_name" db:"parent_name"`
    ParentPhone            string    `json:"parent_phone" db:"parent_phone"`
    ParentEmail            string    `json:"parent_email" db:"parent_email"`
    Notes                  string    `json:"notes" db:"notes"`
}

type ECSEService struct {
    ID          int    `json:"id" db:"id"`
    StudentID   string `json:"student_id" db:"student_id"`
    ServiceType string `json:"service_type" db:"service_type"` // speech, OT, PT, behavioral
    Frequency   string `json:"frequency" db:"frequency"`
    Duration    int    `json:"duration" db:"duration"`
    Provider    string `json:"provider" db:"provider"`
    StartDate   string `json:"start_date" db:"start_date"`
    EndDate     string `json:"end_date" db:"end_date"`
}
```

## Development Guidelines

### Code Organization
```
/
  main.go              # Application entry point and routing
  handlers.go          # HTTP request handlers
  database.go          # Database connection and query functions
  data.go              # Data loading and saving functions
  models.go            # Data structures and models
  security.go          # Authentication and security functions
  middleware.go        # HTTP middleware functions
  utils.go             # Helper functions
  cache.go             # Caching implementation
  import_mileage.go    # Mileage report import functionality
  import_ecse.go       # ECSE student import functionality
  migrate_passwords.go # Password migration script
  /templates           # HTML templates
    login.html
    dashboard.html
    driver_dashboard.html
    fleet.html
    students.html
    (etc.)
```

### HTTP Routes Structure
```
# Public Routes
GET    /                        # Login page
POST   /                        # Login handler
GET    /register                # Registration page
POST   /register                # Registration handler
POST   /logout                  # Logout handler
GET    /health                  # Health check endpoint

# Manager Routes
GET    /manager-dashboard       # Manager dashboard
GET    /approve-users           # Pending users list
POST   /approve-user            # Approve a user
GET    /manage-users            # User management
GET    /edit-user              # Edit user form
POST   /edit-user              # Update user
POST   /delete-user            # Delete user

# Fleet Management
GET    /fleet                   # Bus fleet overview
GET    /company-fleet           # Company vehicles
POST   /update-vehicle-status   # Update vehicle status
GET    /bus-maintenance/{id}    # Bus maintenance history
GET    /vehicle-maintenance/{id} # Vehicle maintenance history
POST   /save-maintenance-record # Save maintenance record

# Route Management
GET    /assign-routes           # Route assignment page
POST   /assign-route            # Assign route to driver
POST   /unassign-route          # Remove route assignment
POST   /add-route               # Add new route
POST   /edit-route              # Update route
POST   /delete-route            # Delete route

# ECSE Management
GET    /import-ecse             # ECSE import page
POST   /import-ecse             # Import ECSE Excel file
GET    /view-ecse-reports       # ECSE reports overview
GET    /ecse-student/{id}       # ECSE student details
GET    /export-ecse             # Export ECSE data to CSV

# Mileage Reports
GET    /import-mileage          # Mileage import page
POST   /import-mileage          # Import mileage Excel
GET    /view-mileage-reports    # View mileage reports
GET    /export-mileage          # Export mileage to Excel

# Driver Routes
GET    /driver-dashboard        # Driver dashboard
POST   /save-log                # Save driver log
GET    /students                # Student management
POST   /add-student             # Add student
POST   /edit-student            # Update student
POST   /remove-student          # Remove student

# Profile
GET    /driver/{username}       # Driver profile (manager view)
```

### Security Implementation
1. **Authentication**: Session-based with secure cookies
   - 24-hour session expiration
   - HTTPOnly, Secure, SameSite=Strict cookies
   - Session tokens stored in memory with mutex protection

2. **Password Security**: 
   - Bcrypt hashing with cost factor 12
   - Migration script for legacy plain-text passwords
   - Minimum 6 character requirement

3. **CSRF Protection**: 
   - Token generated per session
   - Validated on all POST requests
   - Tokens embedded in forms via template

4. **Rate Limiting**: 
   - 5 login attempts per IP per 15 minutes
   - IP-based tracking with cleanup

5. **Security Headers**:
   - Content Security Policy with nonces
   - X-Frame-Options: DENY
   - X-Content-Type-Options: nosniff
   - HSTS enabled on HTTPS
   - Strict referrer policy

6. **Input Validation**:
   - Username: 3-20 alphanumeric characters
   - HTML tag stripping
   - Length limits on all inputs
   - SQL injection prevention via parameterized queries

### Performance Features
1. **In-Memory Caching**: 
   - DataCache with 5-minute TTL
   - Thread-safe with RWMutex
   - Caches users, buses, routes, vehicles, students
   - Automatic invalidation on updates

2. **Database Optimization**:
   - Connection pooling (25 max connections, 5 idle)
   - Prepared statements via sqlx
   - Indexed columns for frequent queries
   - JSONB for flexible data structures

3. **Request Handling**:
   - 30s read timeout, 60s write timeout
   - Graceful shutdown with 30s timeout
   - Recovery middleware for panic handling
   - Context-based request cancellation

4. **Excel Processing**:
   - Streaming file processing with Excelize
   - Batch inserts for large imports
   - Transaction-based imports for consistency

5. **Template Caching**:
   - Pre-compiled templates at startup
   - Efficient template execution

## Common Development Tasks

### Adding a New Feature
1. Define struct in `models.go`
2. Add database table/migration in `database.go`
3. Create data access functions in `data.go`
4. Implement HTTP handlers in `handlers.go`
5. Add routes in `main.go`
6. Create/update HTML templates
7. Add caching support in `cache.go` if needed
8. Write tests (when implemented)

### Database Operations Pattern
```go
// Loading data with error handling
func loadItemsFromDB() ([]Item, error) {
    if db == nil {
        return nil, fmt.Errorf("database not initialized")
    }
    
    var items []Item
    err := db.Select(&items, "SELECT * FROM items ORDER BY id")
    if err != nil {
        return nil, fmt.Errorf("failed to load items: %w", err)
    }
    
    return items, nil
}

// Saving with transaction
func saveItems(items []Item) error {
    tx, err := db.Begin()
    if err != nil {
        return fmt.Errorf("failed to begin transaction: %w", err)
    }
    
    success := false
    defer func() {
        if !success {
            tx.Rollback()
        }
    }()
    
    // Perform operations...
    
    if err := tx.Commit(); err != nil {
        return fmt.Errorf("failed to commit: %w", err)
    }
    
    success = true
    return nil
}
```

### Handler Pattern
```go
func myHandler(w http.ResponseWriter, r *http.Request) {
    // Check authentication
    user := getUserFromSession(r)
    if user == nil {
        http.Redirect(w, r, "/", http.StatusFound)
        return
    }
    
    if r.Method == "GET" {
        // Render template
        data := map[string]interface{}{
            "User":      user,
            "CSRFToken": getSessionCSRFToken(r),
            // Other data...
        }
        renderTemplate(w, r, "template.html", data)
        return
    }
    
    if r.Method == "POST" {
        // Validate CSRF
        if !validateCSRF(r) {
            http.Error(w, "Invalid CSRF token", http.StatusForbidden)
            return
        }
        
        // Process form...
    }
}
```

## Testing Strategy
- **Unit Tests**: Use Go's built-in testing package
- **Integration Tests**: Test database operations with test database
- **Handler Tests**: Use httptest package for HTTP handler testing
- **Load Tests**: Consider using tools like hey or vegeta
- **Security Tests**: Validate CSRF, session handling, rate limiting

## Deployment Configuration
- **Platform**: Railway.app
- **Database**: PostgreSQL on Railway
- **Environment Variables**:
  - `DATABASE_URL`: PostgreSQL connection string
  - `PORT`: Server port (default 5000)
  - `APP_ENV`: Environment (development/production)
- **Build Command**: `go mod download && go build -ldflags='-s -w' -o main .`
- **Start Command**: `./main`
- **Health Check**: GET /health endpoint

## Migration Requirements
- **Password Migration**: Run `go run -tags migrate migrate_passwords.go` to convert plain-text passwords to bcrypt
- **Database Migrations**: Handled automatically in `runMigrations()` function
- **Legacy Data**: Import scripts handle Excel files from old system

## Important Notes for Development
- **CSRF Tokens**: Always include in forms via `{{ .CSRFToken }}`
- **Multipart Forms**: Parse with `r.ParseMultipartForm()` before CSRF validation
- **Session Management**: Sessions stored in memory, lost on restart
- **File Uploads**: Limited to 10MB by default
- **Date/Time Format**: Use "2006-01-02" for dates, "15:04" for times
- **Error Handling**: Always wrap errors with context using fmt.Errorf
- **Database Nulls**: Use sql.NullString, sql.NullTime for nullable columns
- **JSON Storage**: Use JSONB columns for flexible data (locations, positions)
- **Concurrent Access**: Use mutex for shared data structures
- **Template Security**: HTML is auto-escaped by default
- **HTTP Methods**: Always check r.Method in handlers
- **Redirects**: Use http.StatusSeeOther (303) after POST
- **Static Files**: Currently served inline, consider CDN for production
