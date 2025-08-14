# Technical Planning & Architecture
## HS Bus Fleet Management System

---

## System Architecture

### Overview
A monolithic Go web application with server-side rendering, designed for simplicity, reliability, and ease of deployment.

```
┌─────────────────────────────────────────────────┐
│             Client Layer (Browsers)              │
│         Desktop | Tablet | Mobile                │
└─────────────────────┬───────────────────────────┘
                      │ HTTPS
┌─────────────────────▼───────────────────────────┐
│              Go Application Server               │
│  ┌─────────────────────────────────────────┐   │
│  │  HTTP Router → Middleware → Handlers    │   │
│  └─────────────────────────────────────────┘   │
│  ┌─────────────────────────────────────────┐   │
│  │  Business Logic & Data Validation       │   │
│  └─────────────────────────────────────────┘   │
│  ┌─────────────────────────────────────────┐   │
│  │  Template Engine & Session Management   │   │
│  └─────────────────────────────────────────┘   │
└─────────────────────┬───────────────────────────┘
                      │ SQL
┌─────────────────────▼───────────────────────────┐
│            PostgreSQL Database                   │
│         Tables | Indexes | JSONB                 │
└──────────────────────────────────────────────────┘
```

### Component Details

#### 1. **Presentation Layer**
- Server-side HTML rendering with Go templates
- Bootstrap 5.3 for responsive design
- Progressive enhancement with vanilla JavaScript
- Mobile-first approach for tablet optimization

#### 2. **Application Layer**
- Standard library `net/http` for HTTP server
- Middleware chain for authentication, CSRF, logging
- Handler functions organized by feature domain
- In-memory session store with database backup

#### 3. **Data Layer**
- PostgreSQL with `sqlx` for enhanced operations
- Connection pooling optimized for concurrent access
- JSONB columns for flexible data structures
- Prepared statements to prevent SQL injection

---

## Technology Stack

### Core Dependencies
```go
// go.mod
module hs-bus

go 1.21

require (
    github.com/jmoiron/sqlx v1.3.5      // Enhanced database operations
    github.com/lib/pq v1.10.9           // PostgreSQL driver
    golang.org/x/crypto v0.17.0         // bcrypt for passwords
    github.com/xuri/excelize/v2 v2.8.0  // Excel import/export
)
```

### Frontend Assets
- Bootstrap 5.3.0 (CSS framework)
- Bootstrap Icons 1.11.0
- Custom CSS for accessibility enhancements
- Vanilla JavaScript for interactivity

---

## Database Schema

### Core Tables

```sql
-- Users (authentication & authorization)
users (
    username VARCHAR(50) PRIMARY KEY,
    password VARCHAR(255),  -- bcrypt hashed
    role VARCHAR(20),       -- 'manager' or 'driver'
    status VARCHAR(20),     -- 'active' or 'pending'
    created_at TIMESTAMP
)

-- Buses (fleet management)
buses (
    bus_id VARCHAR(50) PRIMARY KEY,
    model VARCHAR(100),
    capacity INTEGER,
    status VARCHAR(20),
    current_mileage INTEGER,
    maintenance_notes TEXT
)

-- Students (rider management)
students (
    student_id VARCHAR(50) PRIMARY KEY,
    name VARCHAR(200),
    locations JSONB,        -- Flexible location data
    guardian VARCHAR(200),
    route_id VARCHAR(50),
    driver VARCHAR(50)
)

-- Routes (transportation planning)
routes (
    route_id VARCHAR(50) PRIMARY KEY,
    route_name VARCHAR(100),
    description TEXT,
    created_at TIMESTAMP
)

-- Route Assignments (driver-bus-route linking)
route_assignments (
    id SERIAL PRIMARY KEY,
    driver VARCHAR(50),
    bus_id VARCHAR(50),
    route_id VARCHAR(50),
    UNIQUE(driver, bus_id, route_id)
)
```

### Supporting Tables
- `driver_logs` - Daily trip records
- `maintenance_records` - Service history
- `vehicles` - Non-bus fleet vehicles
- `ecse_students` - Special education tracking
- `mileage_reports` - Monthly summaries

---

## Security Architecture

### Authentication & Authorization
- Session-based authentication with secure cookies
- Role-based access control (Manager/Driver)
- CSRF tokens on all state-changing operations
- Password hashing with bcrypt (cost factor 10)

### Data Protection
- Input validation and sanitization
- Parameterized SQL queries
- XSS prevention through template escaping
- Rate limiting on login attempts

### Security Headers
```go
// Applied via middleware
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
X-XSS-Protection: 1; mode=block
Content-Security-Policy: default-src 'self'
```

---

## Development Setup

### Prerequisites
- Go 1.21+ installed
- PostgreSQL 15+ running
- Git for version control

### Local Environment
```bash
# Clone repository
git clone <repository>
cd hs-bus

# Install dependencies
go mod download

# Setup environment
cp .env.example .env
# Edit .env with your database credentials

# Run application
go run .
# Or with auto-reload
air

# Access at http://localhost:8080
```

### Environment Variables
```env
DATABASE_URL=postgresql://user:pass@localhost/dbname
PORT=8080
SESSION_SECRET=random-32-char-string
APP_ENV=development
```

---

## Deployment Strategy

### Production Deployment (Railway)
```yaml
# railway.toml
[build]
builder = "nixpacks"
buildCommand = "go build -o fleet"

[deploy]
startCommand = "./fleet"
healthcheckPath = "/health"
healthcheckTimeout = 300

[variables]
PORT = "5000"
```

### Docker Deployment
```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -ldflags='-s -w' -o fleet

FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=builder /app/fleet /fleet
CMD ["/fleet"]
```

---

## Performance Considerations

### Database Optimization
- Indexes on frequently queried columns
- Connection pool tuning (25 max connections)
- Query result caching for read-heavy operations
- Pagination for large result sets

### Application Performance
- Template caching in production
- Static file serving with proper cache headers
- Gzip compression for responses
- Lazy loading for data tables

### Monitoring
- Health check endpoint `/health`
- Structured logging with levels
- Database query performance tracking
- Error rate monitoring

---

## Testing Strategy

### Test Coverage Goals
- Unit tests: 80% coverage
- Integration tests: Critical paths
- End-to-end tests: User workflows

### Test Organization
```
tests/
├── unit/          # Business logic tests
├── integration/   # Database operation tests
├── e2e/          # Full workflow tests
└── load/         # Performance tests
```

### CI/CD Pipeline
```yaml
# GitHub Actions workflow
- Run tests on push/PR
- Check code formatting
- Run security scanner
- Deploy to staging
- Manual approval for production
```

---

## Development Best Practices

### Code Organization
```
hs-bus/
├── main.go              # Entry point, routes
├── handlers*.go         # HTTP handlers by feature
├── models.go            # Data structures
├── database.go          # Database operations
├── middleware.go        # HTTP middleware
├── validation.go        # Input validation
├── utils.go            # Helper functions
├── templates/          # HTML templates
│   ├── components/     # Reusable parts
│   └── pages/         # Full pages
└── static/            # CSS, JS, images
```

### Coding Standards
- Follow Go conventions and idioms
- Use meaningful variable/function names
- Comment exported functions
- Handle all errors explicitly
- Write tests for new features

### Git Workflow
- Feature branches from main
- Descriptive commit messages
- Pull requests with reviews
- Squash merge to main
- Tag releases semantically

---

## Maintenance & Operations

### Regular Tasks
- Database backups (daily)
- Log rotation (weekly)
- Dependency updates (monthly)
- Security patches (as needed)
- Performance review (quarterly)

### Monitoring Checklist
- [ ] Application uptime
- [ ] Response times
- [ ] Error rates
- [ ] Database connections
- [ ] Disk usage
- [ ] Memory usage

### Troubleshooting Guide
1. **High memory usage**: Check for goroutine leaks
2. **Slow queries**: Review database indexes
3. **Login issues**: Verify session configuration
4. **Import failures**: Check file size and format
5. **UI issues**: Clear browser cache

---

## Future Considerations

### Scalability Path
1. **Vertical scaling**: Increase server resources
2. **Read replicas**: Separate read/write databases
3. **Caching layer**: Redis for session/data cache
4. **CDN**: Static asset delivery
5. **Microservices**: Split into domain services

### Technology Upgrades
- GraphQL API for flexible queries
- WebSockets for real-time updates
- React/Vue for dynamic UI
- Kubernetes for orchestration
- Elasticsearch for advanced search

---

## Risk Management

### Technical Risks
- **Database failure**: Regular backups, failover plan
- **Security breach**: Security audits, incident response
- **Performance degradation**: Monitoring, optimization
- **Data loss**: Transaction logs, audit trail

### Mitigation Strategies
- Comprehensive testing before deployment
- Gradual rollout with rollback capability
- Regular security assessments
- Performance benchmarking
- Documentation maintenance

---

## Resources & References

### Documentation
- [Go Documentation](https://go.dev/doc/)
- [PostgreSQL Manual](https://www.postgresql.org/docs/)
- [Bootstrap Documentation](https://getbootstrap.com/docs/)

### Tools
- [sqlx](https://github.com/jmoiron/sqlx) - Database toolkit
- [Air](https://github.com/cosmtrek/air) - Live reload
- [golangci-lint](https://golangci-lint.run/) - Linter

### Monitoring
- Application logs via deployment platform
- Database metrics via pg_stat views
- Performance profiling with pprof

---

**Last Updated**: January 2025  
**Next Review**: February 2025