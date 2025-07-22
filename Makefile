# Fleet Management System Makefile

# Variables
BINARY_NAME=hs-bus
COVERAGE_FILE=coverage.out
COVERAGE_HTML=coverage.html
FRONTEND_DIST_DIR=dist
NODE_MODULES_DIR=node_modules

# Go commands
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt
GOVET=$(GOCMD) vet

# Node.js commands
NPM=npm
NPMBUILD=$(NPM) run build
NPMDEV=$(NPM) run build:dev
NPMWATCH=$(NPM) run watch
NPMINSTALL=$(NPM) install
NPMCLEAN=$(NPM) run clean

# Build flags
LDFLAGS=-ldflags="-s -w"

# Default target
all: frontend-install frontend test build

# Frontend targets

# Install Node.js dependencies
frontend-install:
	@echo "Installing frontend dependencies..."
	@if [ ! -d "$(NODE_MODULES_DIR)" ]; then $(NPMINSTALL); fi

# Build frontend assets for production
frontend:
	@echo "Building frontend assets for production..."
	$(NPMBUILD)

# Build frontend assets for development
frontend-dev:
	@echo "Building frontend assets for development..."
	$(NPMDEV)

# Watch frontend changes
frontend-watch:
	@echo "Watching frontend changes..."
	$(NPMWATCH)

# Clean frontend build artifacts
frontend-clean:
	@echo "Cleaning frontend build artifacts..."
	rm -rf $(FRONTEND_DIST_DIR)
	$(NPMCLEAN) || true

# Full application targets

# Build the binary
build:
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) -v

# Build everything for production
build-prod: frontend-install frontend build
	@echo "Production build complete!"

# Build everything for development  
build-dev: frontend-install frontend-dev build
	@echo "Development build complete!"

# Build for Windows
build-windows:
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME).exe -v

# Build for Linux
build-linux:
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME)-linux -v

# Build for macOS
build-darwin:
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME)-darwin -v

# Clean build artifacts
clean: frontend-clean
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_NAME).exe
	rm -f $(BINARY_NAME)-linux
	rm -f $(BINARY_NAME)-darwin
	rm -f $(COVERAGE_FILE)
	rm -f $(COVERAGE_HTML)

# Run tests
test:
	$(GOTEST) -v ./...

# Run tests with race detector
test-race:
	$(GOTEST) -race -v ./...

# Run tests with coverage
test-coverage:
	$(GOTEST) -v -coverprofile=$(COVERAGE_FILE) -covermode=atomic ./...
	$(GOCMD) tool cover -func=$(COVERAGE_FILE)

# Generate HTML coverage report
coverage-html: test-coverage
	$(GOCMD) tool cover -html=$(COVERAGE_FILE) -o $(COVERAGE_HTML)
	@echo "Coverage report generated at $(COVERAGE_HTML)"

# Run integration tests
test-integration:
	$(GOTEST) -v -tags=integration ./...

# Run all tests (unit + integration)
test-all: test test-integration

# Run end-to-end tests
test-e2e:
	$(GOTEST) -v -tags=e2e ./...

# Run load tests
test-load:
	$(GOTEST) -v -tags=load -timeout=30m ./...

# Format code
fmt:
	$(GOFMT) ./...

# Run go vet
vet:
	$(GOVET) ./...

# Run static analysis
lint:
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run

# Download dependencies
deps:
	$(GOMOD) download
	$(GOMOD) tidy

# Update dependencies
deps-update:
	$(GOMOD) tidy
	$(GOGET) -u ./...
	$(GOMOD) tidy

# Run the application
run:
	$(GOBUILD) -o $(BINARY_NAME) -v
	./$(BINARY_NAME)

# Run with environment variables
run-dev:
	APP_ENV=development $(GOCMD) run .

# Development mode with frontend watching
dev: frontend-install frontend-dev
	@echo "Starting development mode with frontend watching..."
	@echo "Frontend assets will rebuild automatically on changes"
	@echo "Press Ctrl+C to stop"
	$(NPMWATCH) &
	APP_ENV=development $(GOCMD) run .

# Development with Air hot reloading (if available)
dev-air: frontend-install frontend-dev
	@echo "Starting development with Air hot reloading..."
	$(NPMWATCH) &
	air || APP_ENV=development $(GOCMD) run .

# Database migrations
migrate-up:
	@echo "Running database migrations..."
	$(GOCMD) run . -migrate

# Generate mocks (if using mockgen)
mocks:
	@which mockgen > /dev/null || (echo "Installing mockgen..." && go install github.com/golang/mock/mockgen@latest)
	go generate ./...

# Security scan
security:
	@which gosec > /dev/null || (echo "Installing gosec..." && go install github.com/securego/gosec/v2/cmd/gosec@latest)
	gosec -fmt=json -out=security-report.json ./...
	@echo "Security report generated at security-report.json"

# Check for outdated dependencies
check-deps:
	@which go-mod-outdated > /dev/null || (echo "Installing go-mod-outdated..." && go install github.com/psampaz/go-mod-outdated@latest)
	go list -u -m -json all | go-mod-outdated -direct

# Performance profiling
profile-cpu:
	$(GOTEST) -cpuprofile=cpu.prof -bench=. ./...
	$(GOCMD) tool pprof cpu.prof

profile-mem:
	$(GOTEST) -memprofile=mem.prof -bench=. ./...
	$(GOCMD) tool pprof mem.prof

# Docker build
docker-build:
	docker build -t hs-bus:latest .

# Docker run
docker-run:
	docker run -p 5000:5000 --env-file .env hs-bus:latest

# CI/CD helpers
ci-test:
	$(GOTEST) -v -coverprofile=$(COVERAGE_FILE) -covermode=atomic ./...
	$(GOCMD) tool cover -func=$(COVERAGE_FILE) | grep total | awk '{print "Total Coverage: " $$3}'

# Pre-commit checks
pre-commit: fmt vet lint test

# Help
help:
	@echo "Fleet Management System Build Commands:"
	@echo ""
	@echo "Frontend Build:"
	@echo "  make frontend-install  - Install Node.js dependencies"
	@echo "  make frontend         - Build frontend assets (production)"
	@echo "  make frontend-dev     - Build frontend assets (development)"
	@echo "  make frontend-watch   - Watch frontend changes"
	@echo "  make frontend-clean   - Clean frontend build artifacts"
	@echo ""
	@echo "Application Build:"
	@echo "  make build           - Build backend binary"
	@echo "  make build-prod      - Build complete production application"
	@echo "  make build-dev       - Build complete development application"
	@echo "  make build-windows   - Build for Windows"
	@echo "  make build-linux     - Build for Linux"
	@echo "  make build-darwin    - Build for macOS"
	@echo ""
	@echo "Development:"
	@echo "  make dev             - Start development with frontend watching"
	@echo "  make dev-air         - Start development with Air hot reloading"
	@echo "  make run             - Build and run"
	@echo "  make run-dev         - Run in development mode"
	@echo ""
	@echo "Testing:"
	@echo "  make test            - Run unit tests"
	@echo "  make test-race       - Run tests with race detector"
	@echo "  make test-coverage   - Run tests with coverage"
	@echo "  make coverage-html   - Generate HTML coverage report"
	@echo "  make test-integration - Run integration tests"
	@echo "  make test-all        - Run all tests"
	@echo "  make test-e2e        - Run end-to-end tests"
	@echo "  make test-load       - Run load tests"
	@echo ""
	@echo "Code Quality:"
	@echo "  make fmt             - Format code"
	@echo "  make vet             - Run go vet"
	@echo "  make lint            - Run linter"
	@echo "  make security        - Run security scan"
	@echo "  make pre-commit      - Run pre-commit checks"
	@echo ""
	@echo "Maintenance:"
	@echo "  make clean           - Clean all build artifacts"
	@echo "  make deps            - Download dependencies"
	@echo "  make deps-update     - Update dependencies"
	@echo "  make check-deps      - Check for outdated dependencies"

.PHONY: all build build-prod build-dev build-windows build-linux build-darwin clean test test-race test-coverage coverage-html test-integration test-all fmt vet lint deps deps-update run run-dev dev dev-air migrate-up mocks security check-deps profile-cpu profile-mem docker-build docker-run ci-test pre-commit help frontend-install frontend frontend-dev frontend-watch frontend-clean