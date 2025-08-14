# HS Bus - Fleet Management System

A comprehensive web-based fleet management system for school transportation operations. Built with Go and PostgreSQL.

## Features

### Core Functionality
- **Fleet Management**: Track buses and vehicles with maintenance schedules
- **Driver Management**: User accounts, route assignments, and performance tracking  
- **Student Tracking**: Roster management, attendance, and ECSE support
- **Route Optimization**: Assign drivers to buses and routes efficiently
- **Reporting**: Mileage reports, maintenance logs, and analytics dashboards

### Key Capabilities
- 📱 Mobile-responsive design for tablet use in vehicles
- 🔒 Secure authentication with role-based access (Manager/Driver)
- 📊 Real-time dashboards with data visualization
- 📥 Excel import/export for bulk data operations
- 🚸 Special education (ECSE) student tracking
- 📈 Advanced analytics and PDF report generation

## Quick Start

### Prerequisites
- Go 1.21+ 
- PostgreSQL database
- Environment variables configured

### Installation

1. **Clone and setup**
```bash
git clone <repository>
cd hs-bus
go mod download
```

2. **Configure environment**
```bash
# Create .env file with:
DATABASE_URL=postgresql://user:password@host:port/dbname
PORT=8080  # Optional, defaults to 8080 if not set
```

3. **Run the application**
```bash
go run .
# Or build and run:
go build -o fleet.exe && ./fleet.exe
```

4. **Access the system**
- Navigate to `http://localhost:8080` (default port)
- Default manager login: `admin` / `Headstart1`
- Default driver login: `test` / `Headstart1`

## Project Structure

```
hs-bus/
├── main.go                 # Application entry point
├── handlers*.go            # HTTP request handlers
├── models.go               # Data models
├── database.go             # Database connection and queries
├── templates/              # HTML templates
│   ├── components/         # Reusable UI components  
│   └── *.html             # Page templates
├── static/                 # CSS, JavaScript, images
└── migrations/             # Database schema files
```

## Technology Stack

- **Backend**: Go 1.24+, sqlx, lib/pq
- **Database**: PostgreSQL 15+
- **Frontend**: Bootstrap 5.3, Vanilla JavaScript
- **Security**: bcrypt, CSRF protection, secure sessions
- **Deployment**: Railway, Docker support

## Development

### Running locally
```bash
# Start with auto-reload (requires Air)
air

# Run tests
go test ./...

# Build for production
go build -ldflags='-s -w' -o fleet.exe .
```

### Key Files
- `main.go` - Routes and server configuration
- `handlers*.go` - Business logic for each feature
- `database.go` - Database operations
- `models.go` - Data structures

## Documentation

- [Product Requirements](PRD.md) - Feature specifications
- [Planning Document](PLANNING.md) - Architecture and technical details  
- [Task List](TASKS.md) - Development roadmap and progress
- [Development Guide](CLAUDE.md) - Instructions for AI assistants

## Current Status

**Phase 3.5**: User Experience & Accessibility (January 2025)
- ✅ Core fleet management operational
- ✅ Advanced reporting and analytics  
- ✅ Testing infrastructure with CI/CD
- ✅ Performance optimizations
- 🔄 Enhanced UI/UX for non-technical users

## License

Proprietary - For internal use by educational institutions only.

## Support

For issues or questions, check the logs in your deployment platform and verify environment variables are correctly configured.