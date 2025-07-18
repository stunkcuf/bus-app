# Fleet Management System - Planning Document

## Vision Statement

### Mission
To revolutionize school transportation management by providing a comprehensive, user-friendly digital platform that ensures student safety, optimizes operational efficiency, and reduces administrative burden for school districts.

### Vision
To become the leading fleet management solution for educational institutions, setting the standard for safe, efficient, and transparent school transportation operations while prioritizing student welfare and environmental sustainability.

### Core Values
- **Safety First**: Every feature designed with student safety as the top priority
- **Operational Excellence**: Streamline processes to maximize efficiency
- **Data Transparency**: Provide real-time insights for informed decision-making
- **User Empowerment**: Intuitive interfaces that require minimal training
- **Continuous Innovation**: Evolve with emerging technologies and user needs

---

## System Architecture

### Architecture Overview
```
┌─────────────────────────────────────────────────────────────────┐
│                        Client Layer                              │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐            │
│  │   Browser   │  │   Mobile    │  │   Tablet    │            │
│  │  (Desktop)  │  │   Browser   │  │  (Driver)   │            │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘            │
└─────────┼─────────────────┼─────────────────┼──────────────────┘
          │                 │                 │
          └─────────────────┴─────────────────┘
                            │
                     HTTPS (TLS 1.3)
                            │
┌─────────────────────────────────────────────────────────────────┐
│                    Application Layer                             │
│  ┌────────────────────────────────────────────────────────┐    │
│  │                  Go HTTP Server                         │    │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐           │    │
│  │  │  Router  │  │Middleware│  │ Handlers │           │    │
│  │  └──────────┘  └──────────┘  └──────────┘           │    │
│  └────────────────────────────────────────────────────────┘    │
│  ┌────────────────────────────────────────────────────────┐    │
│  │              Business Logic Layer                       │    │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐           │    │
│  │  │   Auth   │  │  Fleet   │  │  Student │           │    │
│  │  │  Service │  │  Service │  │  Service │           │    │
│  │  └──────────┘  └──────────┘  └──────────┘           │    │
│  └────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────┘
                            │
┌─────────────────────────────────────────────────────────────────┐
│                      Data Layer                                  │
│  ┌────────────────────────────────────────────────────────┐    │
│  │                  Data Access Layer                      │    │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐           │    │
│  │  │   sqlx   │  │  Cache   │  │  Import  │           │    │
│  │  │  Driver  │  │  Layer   │  │  Export  │           │    │
│  │  └──────────┘  └──────────┘  └──────────┘           │    │
│  └────────────────────────────────────────────────────────┘    │
│  ┌────────────────────────────────────────────────────────┐    │
│  │                PostgreSQL Database                      │    │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐           │    │
│  │  │  Tables  │  │  JSONB   │  │  Indexes │           │    │
│  │  │          │  │  Columns │  │          │           │    │
│  │  └──────────┘  └──────────┘  └──────────┘           │    │
│  └────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────┘
```

### Component Architecture

#### 1. **Presentation Layer**
- **Server-Side Rendering**: Go templates for fast initial page loads
- **Progressive Enhancement**: JavaScript for interactive features
- **Responsive Design**: Mobile-first approach with Bootstrap
- **Accessibility**: WCAG 2.1 AA compliance

#### 2. **Application Layer**
- **HTTP Server**: Standard library net/http for reliability
- **Routing**: URL pattern matching with middleware chain
- **Session Management**: In-memory session store with mutex protection
- **Security Middleware**: CSRF, rate limiting, security headers

#### 3. **Business Logic Layer**
- **Service Pattern**: Modular services for each domain
- **Domain Models**: Clean separation of concerns
- **Validation**: Input validation at service boundaries
- **Caching**: In-memory cache with TTL for performance

#### 4. **Data Access Layer**
- **Database Driver**: sqlx for enhanced database operations
- **Connection Pooling**: Optimized for concurrent access
- **Transaction Management**: ACID compliance for data integrity
- **Migration System**: Version-controlled schema changes

### Security Architecture

```
┌─────────────────────────────────────────┐
│            Security Layers              │
├─────────────────────────────────────────┤
│  1. Transport Security (HTTPS/TLS)      │
├─────────────────────────────────────────┤
│  2. Authentication (Session-based)      │
├─────────────────────────────────────────┤
│  3. Authorization (Role-based)          │
├─────────────────────────────────────────┤
│  4. CSRF Protection                     │
├─────────────────────────────────────────┤
│  5. Input Validation & Sanitization     │
├─────────────────────────────────────────┤
│  6. Rate Limiting                       │
├─────────────────────────────────────────┤
│  7. Security Headers (CSP, HSTS, etc.)  │
├─────────────────────────────────────────┤
│  8. SQL Injection Prevention            │
└─────────────────────────────────────────┘
```

### Data Flow Architecture

```
User Request → Load Balancer → Application Server
                                      ↓
                              Authentication Check
                                      ↓
                              Authorization Check
                                      ↓
                                Route Handler
                                      ↓
                              Business Logic Service
                                      ↓
                            Cache Check → Cache Hit → Return Data
                                ↓ (Cache Miss)
                            Database Query
                                      ↓
                              Update Cache
                                      ↓
                            Render Template
                                      ↓
                              HTTP Response
```

---

## Technology Stack

### Core Technologies

#### Backend
| Technology | Version | Purpose | Justification |
|------------|---------|---------|---------------|
| Go (Golang) | 1.21+ | Primary language | Performance, simplicity, built-in concurrency |
| PostgreSQL | 15+ | Primary database | JSONB support, reliability, ACID compliance |
| sqlx | 1.3.5 | Database toolkit | Enhanced database operations, struct scanning |
| lib/pq | 1.10.9 | PostgreSQL driver | Native Go driver, good performance |
| bcrypt | - | Password hashing | Industry standard for secure password storage |

#### Frontend
| Technology | Version | Purpose | Justification |
|------------|---------|---------|---------------|
| HTML5 | - | Markup | Semantic, accessible markup |
| Bootstrap | 5.3.0 | CSS Framework | Responsive design, component library |
| Vanilla JavaScript | ES6+ | Interactivity | No framework overhead, fast loading |
| Go Templates | - | Server-side rendering | Type-safe, fast rendering |

#### Infrastructure
| Technology | Version | Purpose | Justification |
|------------|---------|---------|---------------|
| Railway | - | Deployment platform | Simple deployment, integrated PostgreSQL |
| GitHub | - | Version control | Industry standard, CI/CD integration |
| Docker | 20+ | Containerization | Consistent environments (future) |

### Third-Party Libraries

#### Go Dependencies
```go
// go.mod dependencies
require (
    github.com/jmoiron/sqlx v1.3.5
    github.com/lib/pq v1.10.9
    github.com/xuri/excelize/v2 v2.8.0
    github.com/joho/godotenv v1.5.1
    golang.org/x/crypto v0.17.0
)
```

#### JavaScript Libraries (CDN)
- Bootstrap 5.3.0 (CSS & JS)
- Bootstrap Icons 1.11.0

### Database Schema

#### Core Tables
```sql
-- Users table
CREATE TABLE users (
    username VARCHAR(50) PRIMARY KEY,
    password VARCHAR(255) NOT NULL,
    role VARCHAR(20) NOT NULL,
    status VARCHAR(20) DEFAULT 'pending',
    registration_date DATE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Buses table
CREATE TABLE buses (
    bus_id VARCHAR(50) PRIMARY KEY,
    model VARCHAR(100),
    capacity INTEGER,
    status VARCHAR(20),
    oil_status VARCHAR(20),
    tire_status VARCHAR(20),
    maintenance_notes TEXT,
    current_mileage INTEGER DEFAULT 0,
    last_oil_change INTEGER DEFAULT 0,
    last_tire_service INTEGER DEFAULT 0,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Students table with JSONB for locations
CREATE TABLE students (
    student_id VARCHAR(50) PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    locations JSONB,
    phone_number VARCHAR(20),
    alt_phone_number VARCHAR(20),
    guardian VARCHAR(200),
    pickup_time TIME,
    dropoff_time TIME,
    position_number INTEGER,
    route_id VARCHAR(50),
    driver VARCHAR(50),
    active BOOLEAN DEFAULT true
);

-- Additional tables for routes, logs, ECSE, etc.
```

---

## Required Tools & Setup

### Development Environment

#### 1. **Core Development Tools**
- **Go**: Version 1.21 or higher
  ```bash
  # Install from https://go.dev/dl/
  go version  # Verify installation
  ```

- **PostgreSQL**: Version 15 or higher
  ```bash
  # Local development database
  # Install from https://www.postgresql.org/download/
  psql --version  # Verify installation
  ```

- **Git**: Version control
  ```bash
  git --version  # Verify installation
  ```

#### 2. **Code Editor/IDE**
Recommended options:
- **VS Code** with Go extension
  - Go extension by Google
  - PostgreSQL extension
  - GitLens
- **GoLand** (JetBrains)
- **Vim/Neovim** with Go plugins

#### 3. **Database Tools**
- **psql**: Command-line PostgreSQL client
- **pgAdmin 4**: GUI for PostgreSQL
- **DBeaver**: Universal database tool
- **TablePlus**: Modern database GUI

#### 4. **API Testing Tools**
- **curl**: Command-line HTTP client
- **Postman**: API development environment
- **HTTPie**: User-friendly HTTP client

#### 5. **Development Utilities**
```bash
# Go tools
go install golang.org/x/tools/gopls@latest  # Language server
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest  # Linter
go install github.com/cosmtrek/air@latest  # Live reload

# Database migrations
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

### Environment Setup

#### 1. **Project Setup**
```bash
# Clone repository
git clone <repository-url>
cd hs-bus

# Install dependencies
go mod download

# Copy environment template
cp .env.example .env
```

#### 2. **Environment Variables**
```env
# .env file
DATABASE_URL=postgresql://user:password@localhost:5432/fleet_management
PORT=5000
APP_ENV=development
SESSION_SECRET=your-secret-key-here
```

#### 3. **Database Setup**
```bash
# Create database
createdb fleet_management

# Run migrations (automatic on startup)
go run .

# Or manually
psql $DATABASE_URL < schema.sql
```

#### 4. **Development Workflow**
```bash
# Run with live reload
air

# Or run directly
go run .

# Build for production
go build -ldflags='-s -w' -o fleet-management .

# Run tests
go test ./...

# Format code
go fmt ./...

# Lint code
golangci-lint run
```

### Deployment Tools

#### 1. **Railway CLI**
```bash
# Install Railway CLI
npm install -g @railway/cli

# Login
railway login

# Deploy
railway up
```

#### 2. **Docker** (Future)
```dockerfile
# Dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -ldflags='-s -w' -o fleet-management .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/fleet-management .
CMD ["./fleet-management"]
```

### Monitoring & Debugging Tools

#### 1. **Application Monitoring**
- **pprof**: Built-in Go profiling
- **Prometheus**: Metrics collection (future)
- **Grafana**: Metrics visualization (future)

#### 2. **Log Management**
- **Structured logging**: Using log package
- **Log aggregation**: Railway logs
- **Error tracking**: Sentry (future)

#### 3. **Database Monitoring**
```sql
-- Useful queries for monitoring
SELECT * FROM pg_stat_activity;  -- Active connections
SELECT * FROM pg_stat_database;  -- Database statistics
SELECT * FROM pg_stat_user_tables;  -- Table statistics
```

### Security Tools

#### 1. **Security Scanning**
```bash
# Go security checker
go install github.com/securego/gosec/v2/cmd/gosec@latest
gosec ./...

# Dependency vulnerability scanning
go install golang.org/x/vuln/cmd/govulncheck@latest
govulncheck ./...
```

#### 2. **HTTPS/TLS Testing**
- **SSL Labs**: Online SSL test
- **OpenSSL**: Certificate verification
  ```bash
  openssl s_client -connect yourdomain.com:443
  ```

### Documentation Tools

#### 1. **Code Documentation**
```bash
# Generate Go documentation
godoc -http=:6060

# View at http://localhost:6060/pkg/your-module/
```

#### 2. **API Documentation**
- **Swagger/OpenAPI**: Future API documentation
- **Markdown**: Current documentation format

### Performance Testing Tools

#### 1. **Load Testing**
```bash
# Install hey
go install github.com/rakyll/hey@latest

# Basic load test
hey -n 1000 -c 50 https://your-app.railway.app/

# Install vegeta
go install github.com/tsenart/vegeta@latest

# Vegeta attack
echo "GET https://your-app.railway.app/" | vegeta attack -duration=30s | vegeta report
```

#### 2. **Database Performance**
```sql
-- Enable query timing
\timing

-- Explain query plan
EXPLAIN ANALYZE SELECT * FROM students WHERE driver = 'john_doe';
```

---

## Development Best Practices

### Code Organization
1. **Single Responsibility**: Each file/function has one clear purpose
2. **Consistent Naming**: Follow Go naming conventions
3. **Error Handling**: Always check and wrap errors with context
4. **Documentation**: Comment exported functions and complex logic

### Security Practices
1. **Input Validation**: Validate all user inputs
2. **SQL Injection Prevention**: Use parameterized queries
3. **Authentication**: Check auth on every protected route
4. **Secrets Management**: Never commit secrets to version control

### Performance Guidelines
1. **Database Queries**: Use indexes, avoid N+1 queries
2. **Caching**: Cache expensive operations
3. **Connection Pooling**: Reuse database connections
4. **Concurrent Operations**: Leverage Go's goroutines wisely

### Testing Strategy
1. **Unit Tests**: Test individual functions
2. **Integration Tests**: Test database operations
3. **End-to-End Tests**: Test complete workflows
4. **Performance Tests**: Regular load testing

---

## Future Technology Considerations

### Mobile Applications
- **React Native**: Cross-platform mobile development
- **Flutter**: Alternative cross-platform option
- **Progressive Web App**: Enhanced mobile web experience

### Real-Time Features
- **WebSockets**: Real-time updates
- **Server-Sent Events**: One-way real-time communication
- **GraphQL Subscriptions**: Real-time data synchronization

### Machine Learning Integration
- **Route Optimization**: ML-based route planning
- **Predictive Maintenance**: Predict vehicle maintenance needs
- **Attendance Patterns**: Analyze student attendance trends

### Cloud Services
- **AWS/GCP/Azure**: Enterprise cloud deployment
- **Kubernetes**: Container orchestration
- **Microservices**: Service decomposition for scale

---

## Project Milestones

### Phase 1: Foundation (Completed)
- ✅ Core authentication system
- ✅ Basic fleet management
- ✅ Student roster management
- ✅ Driver daily logs

### Phase 2: Enhancement (Current)
- ✅ Route optimization
- ✅ Maintenance scheduling
- ✅ ECSE module
- ✅ Basic reporting
- 🔄 Excel import/export improvements

### Phase 3: Advanced Features (Q2 2025)
- 📅 Mobile application
- 📅 Real-time GPS tracking
- 📅 Parent portal
- 📅 Advanced analytics

### Phase 4: Enterprise Features (Q3-Q4 2025)
- 📅 Multi-district support
- 📅 API for third-party integration
- 📅 Advanced reporting suite
- 📅 AI/ML features

---

## Resource Planning

### Team Structure (Recommended)
1. **Backend Developer**: Go expertise, database design
2. **Frontend Developer**: HTML/CSS/JS, responsive design
3. **DevOps Engineer**: Deployment, monitoring, security
4. **QA Engineer**: Testing, quality assurance
5. **Product Manager**: Requirements, user feedback

### Infrastructure Costs
- **Development**: Local environment (minimal cost)
- **Staging**: Railway Hobby plan ($5-20/month)
- **Production**: Railway Pro plan ($20-100/month based on usage)
- **Future Scale**: Cloud infrastructure (varies by usage)

### Training Requirements
1. **Go Programming**: 2-4 weeks for proficiency
2. **PostgreSQL**: 1-2 weeks for basics
3. **System Architecture**: Ongoing learning
4. **Security Best Practices**: Regular updates

---

## Risk Management

### Technical Risks
1. **Scalability**: Plan for horizontal scaling
2. **Data Loss**: Regular backups, disaster recovery
3. **Security Breaches**: Regular security audits
4. **Technical Debt**: Regular refactoring cycles

### Mitigation Strategies
1. **Code Reviews**: Mandatory peer reviews
2. **Automated Testing**: Comprehensive test suite
3. **Documentation**: Keep documentation current
4. **Monitoring**: Proactive performance monitoring

---

## Conclusion

This planning document provides a comprehensive roadmap for the Fleet Management System development. By following these architectural principles, utilizing the recommended technology stack, and setting up the proper development environment, the team can build a robust, scalable, and secure application that meets the needs of school transportation departments.

Regular updates to this document should reflect changes in technology choices, architectural decisions, and project milestones.