# Go Package Structure Plan

## Current State
- 213 Go files all in the root directory
- Everything in `package main`
- Difficult to maintain and understand

## Proposed Package Structure

```
hs-bus/
├── cmd/
│   └── hs-bus/
│       └── main.go          # Application entry point
├── internal/               # Private packages
│   ├── auth/              # Authentication & sessions
│   │   ├── sessions.go
│   │   ├── middleware.go
│   │   └── csrf.go
│   ├── database/          # Database connection & core
│   │   ├── connection.go
│   │   ├── migrations.go
│   │   └── pool.go
│   ├── models/            # Data models
│   │   ├── user.go
│   │   ├── bus.go
│   │   ├── vehicle.go
│   │   ├── student.go
│   │   ├── route.go
│   │   ├── maintenance.go
│   │   └── ecse.go
│   ├── handlers/          # HTTP handlers
│   │   ├── auth.go
│   │   ├── bus.go
│   │   ├── vehicle.go
│   │   ├── student.go
│   │   ├── route.go
│   │   ├── maintenance.go
│   │   ├── ecse.go
│   │   ├── reports.go
│   │   └── dashboard.go
│   ├── services/          # Business logic
│   │   ├── import.go
│   │   ├── export.go
│   │   ├── notification.go
│   │   ├── analytics.go
│   │   ├── backup.go
│   │   └── monitoring.go
│   ├── api/              # API-specific code
│   │   ├── mobile.go
│   │   ├── routes.go
│   │   └── middleware.go
│   ├── utils/            # Utilities
│   │   ├── validation.go
│   │   ├── logger.go
│   │   ├── errors.go
│   │   └── helpers.go
│   └── templates/        # Template handling
│       ├── cache.go
│       └── functions.go
├── web/                  # Static files
│   ├── static/
│   └── templates/
├── migrations/           # SQL migrations
├── docs/                # Documentation
├── scripts/             # Build/deploy scripts
└── go.mod

## File Categorization

### Authentication/Security (→ internal/auth/)
- sessions.go
- middleware.go  
- middleware_auth.go
- middleware_csrf.go
- middleware_security.go
- secure_headers.go
- csrf.go

### Database Core (→ internal/database/)
- database.go
- db_pool_config.go
- db_pool_handlers.go
- run_migrations.go
- backup_recovery.go
- secure_query.go

### Models (→ internal/models/)
- models.go
- models_*.go files
- User, Bus, Vehicle, Student, Route structs

### Handlers by Feature (→ internal/handlers/)
**Auth/User:**
- handlers_auth.go
- handlers_login.go
- handlers_profile.go

**Fleet Management:**
- handlers_bus.go
- handlers_fleet.go
- handlers_fleet_edit.go
- fleet_handler_clean.go

**Student Management:**
- handlers_students.go
- handlers_ecse.go

**Route Management:**
- handlers_routes.go
- handlers_assign_routes.go
- route_*.go files

**Maintenance:**
- handlers_maintenance.go
- handlers_fuel.go
- maintenance_*.go files

**Reports/Analytics:**
- handlers_reports.go
- report_*.go files
- handlers_analytics.go

### Services (→ internal/services/)
**Import/Export:**
- import_*.go files
- export_*.go files
- wizard_handlers.go

**Notifications:**
- notification_*.go files
- handlers_notifications.go

**Monitoring:**
- monitoring_*.go files
- metrics_*.go files

### API (→ internal/api/)
- mobile_*.go files
- api_handlers.go
- lazy_loading.go

### Utilities (→ internal/utils/)
- utils.go
- errors.go
- validation.go
- logger.go

### Templates (→ internal/templates/)
- template_*.go files

## Migration Strategy

### Phase 1: Create Directory Structure
1. Create all package directories
2. Create package declaration files

### Phase 2: Move Core Components
1. Move database files
2. Move models
3. Move utilities
4. Update imports

### Phase 3: Move Handlers
1. Group handlers by feature
2. Move to appropriate packages
3. Update imports

### Phase 4: Move Services
1. Move business logic
2. Update imports

### Phase 5: Refactor main.go
1. Move to cmd/hs-bus/
2. Update to use packages
3. Clean up initialization

### Phase 6: Testing & Validation
1. Fix compilation errors
2. Update import paths
3. Test all functionality

## Benefits
- Clear separation of concerns
- Easier to test individual components
- Better code reusability
- Reduced coupling
- Easier onboarding for new developers
- Follows Go best practices