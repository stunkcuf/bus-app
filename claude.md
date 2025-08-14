# Development Guide for AI Assistants
## HS Bus Fleet Management System

---

## 🚨 Important: Start of Session

When beginning work on this project, always:
1. Check current git status for uncommitted changes
2. Review TASKS.md for current priorities
3. Run the application to verify it's working
4. Check for any error logs or issues

---

## Project Context

### What This Is
A production fleet management system for school transportation, used daily by 45+ drivers and managers to track buses, students, routes, and maintenance.

### Technology Stack
- **Backend**: Go 1.24 with sqlx
- **Database**: PostgreSQL (Railway hosted)
- **Frontend**: Server-side templates, Bootstrap 5.3, vanilla JS
- **Deployment**: Railway.app

### Key Files
- `main.go` - Route definitions and server setup
- `handlers*.go` - Business logic (70+ handler files)
- `database.go` - Database operations
- `models.go` - Data structures
- `templates/` - HTML templates
- `static/` - CSS and JavaScript

---

## Development Guidelines

### Code Style
```go
// GOOD: Clear, simple, idiomatic Go
func getUser(username string) (*User, error) {
    var user User
    err := db.Get(&user, "SELECT * FROM users WHERE username = $1", username)
    if err != nil {
        return nil, fmt.Errorf("getting user %s: %w", username, err)
    }
    return &user, nil
}

// BAD: Don't add unnecessary complexity
```

### Database Queries
- Always use parameterized queries
- Handle NULL values with sql.Null types
- Use transactions for multi-step operations
- Check for sql.ErrNoRows explicitly

### Error Handling
```go
// Always wrap errors with context
if err != nil {
    return fmt.Errorf("operation failed: %w", err)
}

// Log errors but don't expose internals to users
log.Printf("ERROR: Database query failed: %v", err)
http.Error(w, "An error occurred", http.StatusInternalServerError)
```

### Security Requirements
- Never store passwords in plain text
- Always validate user input
- Use CSRF tokens for state-changing operations
- Check authentication on every protected route
- Sanitize data before rendering in templates

---

## Common Tasks

### Adding a New Page

1. Create handler function:
```go
// handlers_feature.go
func handleFeaturePage(w http.ResponseWriter, r *http.Request) {
    username, role := getSessionUser(r)
    if username == "" {
        http.Redirect(w, r, "/", http.StatusSeeOther)
        return
    }
    
    // Business logic here
    
    renderTemplate(w, "feature.html", map[string]interface{}{
        "Username": username,
        "Role":     role,
        // Add data
    })
}
```

2. Add route in main.go:
```go
http.HandleFunc("/feature", handleFeaturePage)
```

3. Create template in templates/feature.html

4. Add navigation link if needed

### Database Schema Changes

1. Create migration file in migrations/
2. Add new model fields in models.go
3. Update relevant handlers
4. Test with existing data

### Adding Excel Import/Export

Use the existing patterns:
- Import: See `handleImportMileage` 
- Export: See `handleExportMileage`
- Validation: Always validate data before importing

---

## Testing Checklist

Before committing changes:
- [ ] Application starts without errors
- [ ] Login works for both manager and driver
- [ ] New features work on mobile/tablet
- [ ] No console errors in browser
- [ ] Database queries are optimized
- [ ] Error cases are handled gracefully

---

## Deployment Process

### Local Testing
```bash
# Set environment variables
set DATABASE_URL=postgresql://...
# PORT defaults to 8080 if not set

# Run application
go run .

# Or build and run
go build -o fleet.exe
./fleet.exe

# Access at http://localhost:8080
```

### Production Deployment
The application auto-deploys to Railway on push to main branch.

---

## Common Issues & Solutions

### Issue: "Invalid credentials" 
**Solution**: Passwords are bcrypt hashed. 
- Default manager login: admin/Headstart1
- Default driver login: test/Headstart1

### Issue: Session expired
**Solution**: Sessions timeout after 24 hours. This is normal.

### Issue: Excel import fails
**Solution**: Check file size (<10MB) and format. Use provided templates.

### Issue: Slow page load
**Solution**: Check database indexes and query optimization.

---

## Database Information

### Main Tables (29 total)
- `users` - Authentication
- `buses` - Fleet vehicles  
- `students` - Rider information
- `routes` - Transportation routes
- `route_assignments` - Driver/bus/route links
- `driver_logs` - Daily trip records
- `maintenance_records` - Service history
- `vehicles` - Non-bus fleet
- `ecse_students` - Special education

### Data Volume
- ~45 active users
- ~100 vehicles
- ~1000+ students
- ~400+ maintenance records
- ~1700+ mileage reports

---

## UI/UX Considerations

The system is used by non-technical staff, often older users:
- Use large, clear buttons (44px minimum)
- Provide confirmation dialogs for destructive actions
- Include help tooltips on complex features
- Maintain consistent navigation patterns
- Optimize for tablet use in vehicles
- Support offline capabilities where possible

---

## Performance Notes

### Current Metrics
- Page load: ~1.3 seconds average
- Database queries: <100ms for most operations
- Session storage: In-memory with JSON backup
- Concurrent users: Handles 100+ easily

### Optimization Opportunities
- Implement Redis for session caching
- Add database query result caching
- Use CDN for static assets
- Implement lazy loading for large datasets

---

## Security Considerations

### Current Implementation
- bcrypt password hashing (cost 10)
- CSRF tokens on all forms
- Session-based authentication
- Input validation and sanitization
- SQL injection prevention via parameterized queries

### Future Enhancements
- Two-factor authentication
- API rate limiting
- Audit logging
- Field-level encryption for sensitive data

---

## Development Workflow

### Making Changes
1. Create feature branch
2. Make changes following guidelines
3. Test thoroughly
4. Update relevant documentation
5. Commit with clear message
6. Push and create PR

### Commit Messages
```
feat: Add maintenance cost tracking
fix: Resolve session timeout issue  
docs: Update API documentation
perf: Optimize database queries
```

---

## Getting Help

### Resources
- Go Documentation: https://go.dev/doc/
- PostgreSQL Docs: https://postgresql.org/docs/
- Bootstrap: https://getbootstrap.com/
- Railway Docs: https://docs.railway.app/

### Project Files
- [README.md](README.md) - Quick start
- [PRD.md](PRD.md) - Requirements
- [PLANNING.md](PLANNING.md) - Architecture
- [TASKS.md](TASKS.md) - Current work

---

## Important Reminders

1. **Production System**: This is used daily by real users
2. **Data Integrity**: Never delete or modify data without backups
3. **User Experience**: Keep interfaces simple and intuitive
4. **Performance**: Test with realistic data volumes
5. **Security**: Follow security best practices always

---

**Remember**: The goal is a reliable, user-friendly system that makes transportation management easier for school staff. Keep solutions simple, secure, and maintainable.