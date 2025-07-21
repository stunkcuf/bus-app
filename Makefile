# Fleet Management System Makefile

# Variables
BINARY_NAME=hs-bus
COVERAGE_FILE=coverage.out
COVERAGE_HTML=coverage.html

# Go commands
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt
GOVET=$(GOCMD) vet

# Build flags
LDFLAGS=-ldflags="-s -w"

# Default target
all: test build

# Build the binary
build:
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) -v

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
clean:
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
	@echo "Available targets:"
	@echo "  make build          - Build the binary"
	@echo "  make build-windows  - Build for Windows"
	@echo "  make build-linux    - Build for Linux"
	@echo "  make build-darwin   - Build for macOS"
	@echo "  make clean          - Clean build artifacts"
	@echo "  make test           - Run unit tests"
	@echo "  make test-race      - Run tests with race detector"
	@echo "  make test-coverage  - Run tests with coverage"
	@echo "  make coverage-html  - Generate HTML coverage report"
	@echo "  make test-integration - Run integration tests"
	@echo "  make test-all       - Run all tests"
	@echo "  make test-e2e       - Run end-to-end tests"
	@echo "  make test-load      - Run load tests"
	@echo "  make fmt            - Format code"
	@echo "  make vet            - Run go vet"
	@echo "  make lint           - Run linter"
	@echo "  make deps           - Download dependencies"
	@echo "  make deps-update    - Update dependencies"
	@echo "  make run            - Build and run"
	@echo "  make run-dev        - Run in development mode"
	@echo "  make security       - Run security scan"
	@echo "  make check-deps     - Check for outdated dependencies"
	@echo "  make pre-commit     - Run pre-commit checks"

.PHONY: all build build-windows build-linux build-darwin clean test test-race test-coverage coverage-html test-integration test-all fmt vet lint deps deps-update run run-dev migrate-up mocks security check-deps profile-cpu profile-mem docker-build docker-run ci-test pre-commit help