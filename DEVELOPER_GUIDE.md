# Fleet Management System - Developer Guide

## 🚀 Welcome, Developer!

This guide will help you get up to speed with the Fleet Management System codebase. Whether you're fixing bugs, adding features, or maintaining the system, this guide has you covered.

## 📋 Table of Contents

1. [System Overview](#system-overview)
2. [Development Setup](#development-setup)
3. [Architecture](#architecture)
4. [Key Technologies](#key-technologies)
5. [Project Structure](#project-structure)
6. [Core Concepts](#core-concepts)
7. [Development Workflow](#development-workflow)
8. [Testing](#testing)
9. [Deployment](#deployment)
10. [Common Tasks](#common-tasks)
11. [Troubleshooting](#troubleshooting)
12. [Best Practices](#best-practices)

## 🏗️ System Overview

The Fleet Management System is a web-based application for managing school bus operations, including:
- Vehicle fleet management
- Driver assignments and tracking
- Student transportation management
- Route planning and optimization
- Maintenance tracking
- Reporting and analytics

### Key Features
- Role-based access control (Managers and Drivers)
- Real-time progress tracking
- Mobile-responsive design
- Practice mode for training
- Comprehensive help system
- ECSE (Early Childhood Special Education) support

## 💻 Development Setup

### Prerequisites
- Go 1.23.0 or higher
- PostgreSQL 13+ 
- Git
- A code editor (VS Code recommended)
- Docker (optional, for containerized development)

### Initial Setup

1. **Clone the Repository**
   ```bash
   git clone <repository-url>
   cd hs-bus
   ```

2. **Install Go Dependencies**
   ```bash
   go mod download
   ```

3. **Set Up PostgreSQL Database**
   ```bash
   # Create database
   createdb fleet_management
   
   # Run migrations (tables are created on startup)
   # The application automatically creates tables if they don't exist
   ```

4. **Configure Environment Variables**
   Create a `.env` file in the project root:
   ```env
   DATABASE_URL=postgres://username:password@localhost:5432/fleet_management?sslmode=disable
   PORT=5000
   SESSION_SECRET=your-secret-key-here
   ENVIRONMENT=development
   ```

5. **Run the Application**
   ```bash
   go run .
   ```

6. **Access the Application**
   Open http://localhost:5000 in your browser

### Development Tools

- **Air** - Live reload for Go apps
  ```bash
  go install github.com/cosmtrek/air@latest
  air
  ```

- **golangci-lint** - Linter aggregator
  ```bash
  golangci-lint run
  ```

## 🏛️ Architecture

### High-Level Architecture
```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│                 │     │                 │     │                 │
│   Web Browser   │────▶│    Go Server    │────▶│   PostgreSQL    │
│                 │     │                 │     │                 │
└─────────────────┘     └─────────────────┘     └─────────────────┘
         │                       │
         │                       ├── Handlers (HTTP endpoints)
         │                       ├── Middleware (Auth, CSRF, etc.)
         │                       ├── Models (Data structures)
         │                       └── Templates (HTML views)
         │
         └── Static files (CSS, JS)
```

### Request Flow
1. Browser sends request
2. Middleware chain processes request (auth, CSRF, etc.)
3. Handler function executes business logic
4. Database queries via direct SQL
5. Template renders with data
6. Response sent to browser

## 🛠️ Key Technologies

### Backend
- **Go 1.23** - Main programming language
- **net/http** - HTTP server and routing
- **database/sql** - Database interface
- **lib/pq** - PostgreSQL driver
- **bcrypt** - Password hashing
- **html/template** - Server-side templating

### Frontend
- **Bootstrap 5.3** - UI framework
- **Bootstrap Icons** - Icon library
- **Vanilla JavaScript** - No framework dependencies
- **CSS3** - Custom styling with glassmorphism effects

### Database
- **PostgreSQL** - Primary database
- **29 tables** - Complete schema for all features
- **Indexes** - Optimized for common queries

## 📁 Project Structure

```
hs-bus/
├── main.go                 # Application entry point and routing
├── handlers_*.go          # HTTP request handlers (grouped by feature)
├── models.go              # Data structures and types
├── database.go            # Database connection and core queries
├── middleware.go          # HTTP middleware functions
├── security.go            # Authentication and security functions
├── utils.go               # Utility functions
├── validation.go          # Input validation
├── errors.go              # Error handling
├── templates/             # HTML templates
│   ├── *.html            # Page templates
│   └── components/       # Reusable components
├── static/               # Static assets
│   ├── *.js             # JavaScript files
│   ├── *.css            # Stylesheets
│   └── images/          # Images
├── docs/                # Documentation
├── utilities/           # Utility scripts
└── go.mod              # Go module definition
```

### Key Files

#### Handlers (handlers_*.go)
- `handlers.go` - Core handlers (login, dashboard, etc.)
- `handlers_fleet.go` - Fleet management endpoints
- `handlers_students.go` - Student management
- `handlers_routes.go` - Route assignments
- `handlers_reports.go` - Reporting functionality
- `handlers_help.go` - Help system
- `handlers_wizards.go` - Multi-step forms

#### Core Systems
- `security.go` - Authentication, sessions, CSRF
- `middleware.go` - Request processing pipeline
- `database.go` - Database connection and helpers
- `validation.go` - Input validation rules
- `cache.go` - Caching implementation

#### New Features (Phase 3.5)
- `handlers_progress_tracking.go` - User progress tracking
- `handlers_practice_mode.go` - Practice mode with sample data
- `handlers_getting_started.go` - Role-specific guides
- `handlers_quick_reference.go` - Printable guides
- `handlers_user_manual.go` - Comprehensive documentation

## 🔑 Core Concepts

### Authentication Flow
1. User submits login form
2. Password verified with bcrypt
3. Session created and stored
4. Session token set in cookie
5. All requests validate session

### Session Management
```go
// Check authentication
session, err := getUserFromSession(r, w)
if err != nil {
    http.Redirect(w, r, "/", http.StatusSeeOther)
    return
}
```

### CSRF Protection
All forms must include CSRF token:
```html
<input type="hidden" name="csrf_token" value="{{.CSRFToken}}">
```

### Database Patterns
```go
// Query pattern
rows, err := db.Query(`
    SELECT id, name, email 
    FROM users 
    WHERE role = $1
`, role)
defer rows.Close()

// Transaction pattern
tx, err := db.Begin()
// ... operations ...
tx.Commit() // or tx.Rollback()
```

### Template Rendering
```go
tmpl := template.Must(template.ParseFiles("templates/page.html"))
data := struct {
    Title    string
    User     User
    CSRFToken string
}{
    Title:    "Page Title",
    User:     session,
    CSRFToken: getCSPNonce(r),
}
tmpl.Execute(w, data)
```

## 🔄 Development Workflow

### 1. Creating a New Feature

1. **Plan the Feature**
   - Review requirements in TASKS.md
   - Design database schema if needed
   - Plan UI/UX flow

2. **Create Handler File**
   ```go
   // handlers_feature.go
   package main
   
   func featureHandler(w http.ResponseWriter, r *http.Request) {
       // Implementation
   }
   ```

3. **Add Routes**
   ```go
   // In main.go
   mux.HandleFunc("/feature", withRecovery(requireAuth(featureHandler)))
   ```

4. **Create Template**
   ```html
   <!-- templates/feature.html -->
   <!DOCTYPE html>
   <html>
   <!-- Template content -->
   </html>
   ```

5. **Add Navigation**
   Update relevant dashboards and menus

### 2. Database Changes

1. **Add Migration Function**
   ```go
   func createFeatureTable(db *sql.DB) error {
       query := `
       CREATE TABLE IF NOT EXISTS feature (
           id SERIAL PRIMARY KEY,
           name VARCHAR(255) NOT NULL,
           created_at TIMESTAMP DEFAULT NOW()
       )`
       _, err := db.Exec(query)
       return err
   }
   ```

2. **Call in main.go init**
   ```go
   if err := createFeatureTable(db); err != nil {
       log.Printf("Failed to create feature table: %v", err)
   }
   ```

### 3. Frontend Development

1. **Create JavaScript Module**
   ```javascript
   // static/feature.js
   class FeatureManager {
       constructor() {
           this.init();
       }
       
       init() {
           // Setup event listeners
       }
   }
   ```

2. **Add Styles**
   ```css
   /* In template or separate CSS file */
   .feature-component {
       /* Styles */
   }
   ```

## 🧪 Testing

### Running Tests
```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific test
go test -run TestFunctionName
```

### Writing Tests
```go
// handlers_test.go
func TestFeatureHandler(t *testing.T) {
    // Setup
    req, err := http.NewRequest("GET", "/feature", nil)
    if err != nil {
        t.Fatal(err)
    }
    
    // Execute
    rr := httptest.NewRecorder()
    handler := http.HandlerFunc(featureHandler)
    handler.ServeHTTP(rr, req)
    
    // Assert
    if status := rr.Code; status != http.StatusOK {
        t.Errorf("handler returned wrong status code: got %v want %v",
            status, http.StatusOK)
    }
}
```

### Load Testing
```bash
# Using the built-in load test
go test -run TestLoadEndpoints
```

## 🚀 Deployment

### Railway Deployment

1. **Ensure Dockerfile is updated**
   ```dockerfile
   FROM golang:1.23-alpine AS builder
   # Build steps...
   ```

2. **Push to GitHub**
   ```bash
   git add .
   git commit -m "Feature: description"
   git push origin main
   ```

3. **Railway Auto-Deploy**
   - Railway automatically builds and deploys on push
   - Monitor deployment in Railway dashboard

### Manual Deployment

1. **Build Binary**
   ```bash
   go build -o fleet-management
   ```

2. **Set Environment Variables**
   ```bash
   export DATABASE_URL=...
   export PORT=5000
   ```

3. **Run Application**
   ```bash
   ./fleet-management
   ```

## 📝 Common Tasks

### Adding a New Page

1. Create handler function
2. Add route in main.go
3. Create HTML template
4. Add navigation link
5. Test functionality

### Adding a Database Table

1. Design schema
2. Create migration function
3. Add to init sequence
4. Create model struct
5. Implement CRUD operations

### Adding an API Endpoint

1. Create handler function
2. Add route with /api/ prefix
3. Return JSON response
4. Document in API_DOCUMENTATION.md

### Implementing a New Report

1. Create report handler
2. Design query for data
3. Create template for display
4. Add to report builder options
5. Test with sample data

## 🔧 Troubleshooting

### Common Issues

1. **Database Connection Failed**
   - Check DATABASE_URL format
   - Verify PostgreSQL is running
   - Check network/firewall settings

2. **Templates Not Found**
   - Ensure working directory is project root
   - Check template file names match code

3. **Session Issues**
   - Clear browser cookies
   - Check session storage implementation
   - Verify SESSION_SECRET is set

4. **JavaScript Not Loading**
   - Check browser console for errors
   - Verify script tags in templates
   - Check CSP nonce implementation

### Debugging Tips

1. **Enable Debug Logging**
   ```go
   log.Printf("DEBUG: Variable value: %v", variable)
   ```

2. **Check Database Queries**
   ```go
   log.Printf("SQL: %s, Args: %v", query, args)
   ```

3. **Inspect HTTP Requests**
   ```go
   log.Printf("Request: %s %s", r.Method, r.URL.Path)
   ```

## ✨ Best Practices

### Code Style
- Use `gofmt` for consistent formatting
- Follow Go naming conventions
- Keep functions focused and small
- Handle errors explicitly

### Security
- Always validate user input
- Use parameterized queries
- Include CSRF tokens in forms
- Hash passwords with bcrypt
- Sanitize output in templates

### Performance
- Use database indexes wisely
- Implement caching where appropriate
- Paginate large result sets
- Minimize database queries

### Maintenance
- Document complex logic
- Keep dependencies updated
- Monitor error logs
- Regular database backups

## 📚 Additional Resources

### Internal Documentation
- `CLAUDE.md` - AI assistant guidelines
- `TASKS.md` - Project roadmap and tasks
- `API_DOCUMENTATION.md` - API reference
- `PROJECT_DOCUMENTATION.md` - System overview

### External Resources
- [Go Documentation](https://golang.org/doc/)
- [PostgreSQL Docs](https://www.postgresql.org/docs/)
- [Bootstrap 5 Docs](https://getbootstrap.com/docs/5.3/)
- [Railway Docs](https://docs.railway.app/)

## 🤝 Getting Help

1. Check existing documentation
2. Search codebase for examples
3. Review git history for context
4. Test in practice mode first
5. Ask team members

---

Welcome to the team! We're excited to have you contributing to the Fleet Management System. Remember: prioritize security, write clean code, and always consider the end users - the hard-working people managing school transportation every day.