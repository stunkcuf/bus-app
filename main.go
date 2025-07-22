package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// Constants for better maintainability
const (
	DefaultPort       = "5000"
	SessionCookieName = "session_id"
	CSRFTokenHeader   = "X-CSRF-Token"
	TemplateGlob      = "templates/*.html"

	// Timeouts
	ReadTimeout    = 30 * time.Second
	WriteTimeout   = 60 * time.Second
	IdleTimeout    = 120 * time.Second
	MaxHeaderBytes = 1 << 20

	// Roles
	RoleManager       = "manager"
	RoleDriver        = "driver"
	RoleDriverPending = "driver_pending"

	// Status
	StatusActive  = "active"
	StatusPending = "pending"

	// Date format
	DateFormat = "2006-01-02"

	// Minimum password length
	MinPasswordLength = 6
)

// Templates variable
var templates *template.Template
var templateCache *TemplateCache

func init() {
	// Complete function map with all required functions including mul
	funcMap := template.FuncMap{
		// JSON marshaling
		"json": jsonMarshal,

		// ADDED: seq function for generating number sequences (needed for year dropdowns)
		"seq": func(start, end int) []int {
			var result []int
			for i := start; i <= end; i++ {
				result = append(result, i)
			}
			return result
		},

		// ADDED: formatNumber for formatting numbers with commas
		"formatNumber": func(n interface{}) string {
			var num int
			switch v := n.(type) {
			case int:
				num = v
			case int64:
				num = int(v)
			case float64:
				num = int(v)
			default:
				return fmt.Sprintf("%v", n)
			}

			// Format with commas
			str := fmt.Sprintf("%d", num)
			if num < 0 {
				return str // Handle negative numbers simply for now
			}
			if num < 1000 {
				return str
			}

			// Add commas every 3 digits from the right
			var result []string
			for i := len(str); i > 0; i -= 3 {
				start := i - 3
				if start < 0 {
					start = 0
				}
				result = append([]string{str[start:i]}, result...)
			}
			return strings.Join(result, ",")
		},

		// ADDED: multiply function (handles interface{} types)
		"multiply": func(a, b interface{}) float64 {
			return toFloat64(a) * toFloat64(b)
		},

		// Mathematical operations
		"add": func(a, b interface{}) interface{} {
			// Support both float and int addition
			switch a.(type) {
			case int:
				return int(toFloat64(a) + toFloat64(b))
			default:
				return toFloat64(a) + toFloat64(b)
			}
		},
		"sub": func(a, b interface{}) float64 {
			return toFloat64(a) - toFloat64(b)
		},
		"mul": func(a, b interface{}) float64 {
			return toFloat64(a) * toFloat64(b)
		},
		"div": func(a, b interface{}) float64 {
			bVal := toFloat64(b)
			if bVal == 0 {
				return 0
			}
			return toFloat64(a) / bVal
		},
		"mod": func(a, b interface{}) int {
			return toInt(a) % toInt(b)
		},

		// Comparison functions
		"eq": func(a, b interface{}) bool {
			return a == b
		},
		"ne": func(a, b interface{}) bool {
			return a != b
		},
		"lt": func(a, b interface{}) bool {
			return toFloat64(a) < toFloat64(b)
		},
		"le": func(a, b interface{}) bool {
			return toFloat64(a) <= toFloat64(b)
		},
		"gt": func(a, b interface{}) bool {
			return toFloat64(a) > toFloat64(b)
		},
		"ge": func(a, b interface{}) bool {
			return toFloat64(a) >= toFloat64(b)
		},

		// Utility functions
		"len":    getLength,
		"printf": fmt.Sprintf,

		// Number formatting
		"formatFloat": func(f interface{}, decimals int) string {
			format := fmt.Sprintf("%%.%df", decimals)
			return fmt.Sprintf(format, toFloat64(f))
		},
		"formatCurrency": func(amount interface{}) string {
			// UPDATED: Better currency formatting with commas
			f := toFloat64(amount)

			// Format with 2 decimal places
			str := fmt.Sprintf("%.2f", f)
			parts := strings.Split(str, ".")

			// Handle the integer part
			intPart := parts[0]
			negative := false
			if strings.HasPrefix(intPart, "-") {
				negative = true
				intPart = intPart[1:]
			}

			// Add commas to integer part
			var result []string
			for i := len(intPart); i > 0; i -= 3 {
				start := i - 3
				if start < 0 {
					start = 0
				}
				result = append([]string{intPart[start:i]}, result...)
			}

			formattedInt := strings.Join(result, ",")
			if negative {
				formattedInt = "-" + formattedInt
			}

			// Combine with decimal part
			if len(parts) > 1 {
				return formattedInt + "." + parts[1]
			}
			return formattedInt + ".00"
		},
		"formatPercent": func(value interface{}) string {
			return fmt.Sprintf("%.0f%%", toFloat64(value))
		},

		// Date formatting
		"formatDate": func(date string) string {
			t, err := time.Parse("2006-01-02", date)
			if err != nil {
				return date
			}
			return t.Format("Jan 2, 2006")
		},

		// String functions
		"hasPrefix": func(s, prefix string) bool {
			return len(s) >= len(prefix) && s[:len(prefix)] == prefix
		},
		"hasSuffix": func(s, suffix string) bool {
			return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
		},
		"title": func(s string) string {
			// Simple title case implementation
			words := strings.Fields(s)
			for i, word := range words {
				if len(word) > 0 {
					words[i] = strings.ToUpper(string(word[0])) + strings.ToLower(word[1:])
				}
			}
			return strings.Join(words, " ")
		},
		"upper": func(s string) string {
			return strings.ToUpper(s)
		},
		"lower": func(s string) string {
			return strings.ToLower(s)
		},
		"formatBytes": func(bytes int64) string {
			const unit = 1024
			if bytes < unit {
				return fmt.Sprintf("%d B", bytes)
			}
			div, exp := int64(unit), 0
			for n := bytes / unit; n >= unit; n /= unit {
				div *= unit
				exp++
			}
			return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
		},
		"formatDateTime": func(t time.Time) string {
			return t.Format("Jan 2, 2006 3:04 PM")
		},
		"dayOfWeek": func(day int) string {
			days := []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}
			if day >= 0 && day < len(days) {
				return days[day]
			}
			return ""
		},
		"truncate": func(s string, n int) string {
			if len(s) <= n {
				return s
			}
			if n <= 3 {
				return s[:n]
			}
			return s[:n-3] + "..."
		},
		"substr": func(s string, start, length int) string {
			if start < 0 || start >= len(s) {
				return ""
			}
			end := start + length
			if end > len(s) {
				end = len(s)
			}
			return s[start:end]
		},

		// Navigation functions for accessible design
		"getNavigation": func(user *User, currentPage string) NavigationData {
			if user == nil {
				return NavigationData{CurrentPage: currentPage}
			}
			return getNavigationData(user, currentPage)
		},
		"isActive": func(current, page string) bool {
			return strings.Contains(current, page)
		},
		"getBadgeClass": func(color string) string {
			switch color {
			case "primary":
				return "badge-primary"
			case "success":
				return "badge-success"
			case "warning":
				return "badge-warning"
			case "danger":
				return "badge-danger"
			default:
				return "badge-secondary"
			}
		},
		
		// dict creates a map from alternating key/value pairs
		"dict": func(values ...interface{}) (map[string]interface{}, error) {
			if len(values)%2 != 0 {
				return nil, fmt.Errorf("dict requires an even number of arguments")
			}
			dict := make(map[string]interface{})
			for i := 0; i < len(values); i += 2 {
				key, ok := values[i].(string)
				if !ok {
					return nil, fmt.Errorf("dict keys must be strings")
				}
				dict[key] = values[i+1]
			}
			return dict, nil
		},
		// list creates a slice from the provided values
		"list": func(values ...interface{}) []interface{} {
			return values
		},
	}

	// Load templates with optimization
	var err error
	if os.Getenv("APP_ENV") == "production" {
		// Use optimized template cache in production
		log.Printf("Loading templates with optimization...")
		templateCache, err = PrecompileTemplates("templates")
		if err != nil {
			log.Fatalf("Failed to load template cache: %v", err)
		}
		log.Printf("Successfully loaded optimized templates")
	} else {
		// Use standard templates in development for hot reload
		log.Printf("Loading templates from: %s", TemplateGlob)
		templates = template.New("").Funcs(funcMap)
		
		// First load component templates
		componentGlob := "templates/components/*.html"
		templates, err = templates.ParseGlob(componentGlob)
		if err != nil {
			log.Printf("Warning: Failed to load component templates: %v", err)
		}
		
		// Then load page templates
		templates, err = templates.ParseGlob(TemplateGlob)
		if err != nil {
			log.Fatalf("Failed to load templates: %v", err)
		}
		log.Printf("Successfully loaded %d templates", len(templates.Templates()))
	}

	// Start session cleanup in security.go
	go periodicSessionCleanup()
}

// Helper function to convert interface{} to float64
func toFloat64(v interface{}) float64 {
	switch x := v.(type) {
	case int:
		return float64(x)
	case int32:
		return float64(x)
	case int64:
		return float64(x)
	case uint:
		return float64(x)
	case uint32:
		return float64(x)
	case uint64:
		return float64(x)
	case float32:
		return float64(x)
	case float64:
		return x
	case string:
		// Handle percentage strings like "75%"
		if len(x) > 0 && x[len(x)-1] == '%' {
			var val float64
			if _, err := fmt.Sscanf(x[:len(x)-1], "%f", &val); err == nil {
				return val
			}
		}
		// Try to parse as float
		var val float64
		if _, err := fmt.Sscanf(x, "%f", &val); err == nil {
			return val
		}
		return 0
	default:
		return 0
	}
}

// Helper function to convert interface{} to int
func toInt(v interface{}) int {
	switch x := v.(type) {
	case int:
		return x
	case int32:
		return int(x)
	case int64:
		return int(x)
	case uint:
		return int(x)
	case uint32:
		return int(x)
	case uint64:
		return int(x)
	case float32:
		return int(x)
	case float64:
		return int(x)
	default:
		return 0
	}
}

// Template helper functions
func jsonMarshal(v interface{}) template.JS {
	b, err := json.Marshal(v)
	if err != nil {
		log.Printf("JSON marshal error: %v", err)
		return template.JS("{}")
	}
	return template.JS(b)
}

// Replace the getLength function in main.go with this updated version

func getLength(v interface{}) int {
	switch s := v.(type) {
	case []interface{}:
		return len(s)
	case []Bus:
		return len(s)
	case []User:
		return len(s)
	case []Student:
		return len(s)
	case []Route:
		return len(s)
	case []Vehicle:
		return len(s)
	case []ECSEStudentView:
		return len(s)
	case []ECSEStudent:
		return len(s)
	case []MaintenanceAlert:
		return len(s)
	case []CombinedMaintenanceLog:
		return len(s)
	case []MileageReport:
		return len(s)
	case []RouteAssignment:
		return len(s)
	case map[string]interface{}:
		return len(s)
	case map[string]int:
		return len(s)
	case map[string][]MaintenanceAlert:
		return len(s)
	case string:
		return len(s)
	default:
		return 0
	}
}
func main() {
	// Initialize logger
	InitLogger()

	// Setup log rotation
	LogInfo("📝 Setting up log rotation...")
	if err := SetupLogRotation(); err != nil {
		LogError("Failed to setup log rotation", err)
		// Continue without log rotation
	}

	// Database setup
	LogInfo("🗄️  Setting up PostgreSQL database...")
	if err := setupDatabase(); err != nil {
		LogFatal("Failed to setup database", err)
	}
	defer closeDatabase()
	
	// Test server is handled by public_test_routes.go when needed

	// Initialize session manager
	LogInfo("🔐 Setting up session manager...")
	if err := initializeSessionManager(); err != nil {
		LogFatal("Failed to initialize session manager", err)
	}

	// Initialize query cache
	LogInfo("🚀 Setting up query cache...")
	initQueryCache()

	// Start database connection pool monitoring
	LogInfo("📊 Starting database connection pool monitoring...")
	startDBMonitoring()

	// Reset rate limiter on startup to clear any previous blocks
	LogInfo("🔄 Resetting rate limiter...")
	rateLimiter.Reset()

	mux := setupRoutes()

	// Configure compression
	compressionConfig := DefaultCompressionConfig()
	compressionConfig.Enabled = os.Getenv("DISABLE_COMPRESSION") != "true"

	// Chain middlewares: CSP -> Security -> Router (Compression disabled for now)
	handler := CSPMiddleware(SecurityHeaders(mux))

	port := os.Getenv("PORT")
	if port == "" {
		port = DefaultPort
	}

	server := &http.Server{
		Addr:           fmt.Sprintf(":%s", port),
		Handler:        handler,
		ReadTimeout:    ReadTimeout,
		WriteTimeout:   WriteTimeout,
		IdleTimeout:    IdleTimeout,
		MaxHeaderBytes: MaxHeaderBytes,
	}

	// Start background jobs
	startScheduledExportsJob()

	// Graceful shutdown
	go gracefulShutdown(server)

	log.Printf("🚀 Server starting on port %s", port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}
}

// setupRoutes configures all application routes
func setupRoutes() *http.ServeMux {
	mux := http.NewServeMux()

	// Static file server with proper content types
	mux.HandleFunc("/static/", func(w http.ResponseWriter, r *http.Request) {
		// Remove /static/ prefix
		path := r.URL.Path[8:]

		// Set cache control headers to prevent caching during development
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")

		// Set content type based on file extension
		switch {
		case strings.HasSuffix(path, ".css"):
			w.Header().Set("Content-Type", "text/css; charset=utf-8")
		case strings.HasSuffix(path, ".js"):
			w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
		case strings.HasSuffix(path, ".png"):
			w.Header().Set("Content-Type", "image/png")
		case strings.HasSuffix(path, ".jpg"), strings.HasSuffix(path, ".jpeg"):
			w.Header().Set("Content-Type", "image/jpeg")
		case strings.HasSuffix(path, ".svg"):
			w.Header().Set("Content-Type", "image/svg+xml")
		}

		// Serve the file
		http.ServeFile(w, r, filepath.Join("static", path))
	})

	// Register public test routes FIRST (no middleware)
	setupPublicTestRoutes(mux)
	
	// Create a special handler for root that checks path first
	mux.HandleFunc("/", withRecovery(func(w http.ResponseWriter, r *http.Request) {
		// Only handle exact "/" path, let other paths fall through
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		RateLimitMiddleware(loginHandler)(w, r)
	}))
	mux.HandleFunc("/register", withRecovery(RateLimitMiddleware(registerHandler)))
	mux.HandleFunc("/logout", withRecovery(logoutHandler))
	mux.HandleFunc("/health", withRecovery(healthCheck))
	mux.HandleFunc("/status", withRecovery(serverStatusHandler))
	
	// Test endpoint for ECSE dashboard (temporary - remove in production)
	mux.HandleFunc("/test-ecse", withRecovery(testECSEHandler))
	
	// Debug endpoints are available through /api/debug-* routes in development mode

	// Manager-only routes
	setupManagerRoutes(mux)

	// Driver routes
	setupDriverRoutes(mux)

	// API routes (accessible by both managers and drivers)
	setupAPIRoutes(mux)
	
	// Common protected routes for all authenticated users
	mux.HandleFunc("/profile", withRecovery(requireAuth(requireDatabase(profileHandler))))
	mux.HandleFunc("/settings", withRecovery(requireAuth(requireRole("manager")(requireDatabase(settingsHandler)))))
	mux.HandleFunc("/help-demo", withRecovery(requireAuth(requireDatabase(helpDemoHandler))))

	// Common protected routes
	mux.HandleFunc("/dashboard", withRecovery(requireAuth(requireDatabase(dashboardHandler))))

	return mux
}

// setupAPIRoutes configures API endpoints
func setupAPIRoutes(mux *http.ServeMux) {
	// Versioned API routes
	setupV1APIRoutes(mux)
	
	// Core API routes (from api_handlers.go)
	registerAPIRoutes()
	
	// Maintenance API routes
	mux.HandleFunc("/api/check-maintenance", withRecovery(requireAuth(requireDatabase(checkMaintenanceDueHandler))))
	mux.HandleFunc("/api/debug-maintenance", withRecovery(requireAuth(requireRole("manager")(requireDatabase(debugMaintenanceRecordsHandler)))))

	// Dashboard Analytics API routes
	mux.HandleFunc("/api/dashboard/analytics", withRecovery(requireAuth(requireRole("manager")(requireDatabase(dashboardAnalyticsHandler)))))
	mux.HandleFunc("/api/dashboard/fleet-status", withRecovery(requireAuth(requireDatabase(fleetStatusWidgetHandler))))
	mux.HandleFunc("/api/dashboard/maintenance-alerts", withRecovery(requireAuth(requireDatabase(maintenanceAlertsWidgetHandler))))
	mux.HandleFunc("/api/dashboard/route-efficiency", withRecovery(requireAuth(requireDatabase(routeEfficiencyWidgetHandler))))

	// Report Builder API routes
	mux.HandleFunc("/api/report-builder", withRecovery(requireAuth(requireRole("manager")(requireDatabase(reportBuilderAPIHandler)))))
	mux.HandleFunc("/api/report-data-sources", withRecovery(requireAuth(requireRole("manager")(requireDatabase(getReportDataSourcesHandler)))))
	mux.HandleFunc("/api/report-chart-types", withRecovery(requireAuth(requireRole("manager")(requireDatabase(reportChartTypesHandler)))))

	// Database Optimization API routes
	mux.HandleFunc("/api/database/stats", withRecovery(requireAuth(requireRole("manager")(requireDatabase(databaseStatsHandler)))))
	mux.HandleFunc("/api/database/optimize", withRecovery(requireAuth(requireRole("manager")(requireDatabase(optimizeDatabaseHandler)))))
	mux.HandleFunc("/api/cache/stats", withRecovery(requireAuth(requireRole("manager")(cacheStatsHandler))))
	
	// Database Connection Pool Monitoring routes
	mux.HandleFunc("/api/db/stats", withRecovery(requireAuth(requireRole("manager")(dbStatsHandler))))
	mux.HandleFunc("/api/db/metrics", withRecovery(requireAuth(requireRole("manager")(dbMetricsHandler))))
	mux.HandleFunc("/api/db/health", withRecovery(dbHealthCheckHandler)) // No auth for monitoring tools

	// Chart/Visualization API routes
	mux.HandleFunc("/api/charts/data", withRecovery(requireAuth(requireDatabase(chartDataHandler))))
	mux.HandleFunc("/api/charts/available", withRecovery(requireAuth(requireDatabase(availableChartsHandler))))

	// Lazy Loading API routes
	mux.HandleFunc("/api/lazy/monthly-mileage-reports", withRecovery(requireAuth(requireRole("manager")(requireDatabase(monthlyMileageReportsAPIHandler)))))
	mux.HandleFunc("/api/lazy/maintenance-records", withRecovery(requireAuth(requireRole("manager")(requireDatabase(maintenanceRecordsAPIHandler)))))
	mux.HandleFunc("/api/lazy/fleet-vehicles", withRecovery(requireAuth(requireRole("manager")(requireDatabase(fleetVehiclesAPIHandler)))))

	// Comparative Analytics API routes
	mux.HandleFunc("/api/analytics/comparison", withRecovery(requireAuth(requireRole("manager")(requireDatabase(comparativeAnalyticsHandler)))))
	mux.HandleFunc("/api/analytics/trend", withRecovery(requireAuth(requireRole("manager")(requireDatabase(trendAnalysisHandler)))))

	// Fuel Efficiency API routes
	mux.HandleFunc("/api/fuel/record", withRecovery(requireAuth(requireDatabase(saveFuelRecordHandler))))
	mux.HandleFunc("/api/fuel/efficiency", withRecovery(requireAuth(requireDatabase(vehicleFuelEfficiencyHandler))))
	mux.HandleFunc("/api/fuel/summary", withRecovery(requireAuth(requireRole("manager")(requireDatabase(fleetFuelSummaryHandler)))))
	mux.HandleFunc("/api/fuel/trend-chart", withRecovery(requireAuth(requireDatabase(fuelTrendChartHandler))))

	// Driver Scorecard API routes
	mux.HandleFunc("/api/scorecard/driver", withRecovery(requireAuth(requireDatabase(driverScorecardHandler))))
	mux.HandleFunc("/api/scorecard/all", withRecovery(requireAuth(requireRole("manager")(requireDatabase(allDriverScorecardsHandler)))))
}

// setupV1APIRoutes configures Version 1 API endpoints
func setupV1APIRoutes(mux *http.ServeMux) {
	// Health check endpoint for v1
	mux.HandleFunc("/api/v1/health", withRecovery(withAPIVersion(APIVersion1, healthV1Handler)))
	
	// Dashboard endpoints for v1
	mux.HandleFunc("/api/v1/dashboard/stats", withRecovery(requireAuth(requireRole("manager")(withAPIVersion(APIVersion1, dashboardStatsV1Handler)))))
	
	// Future v1 endpoints can be added here...
	
	// Backward compatibility routes (legacy endpoints redirect to v1)
	mux.HandleFunc("/api/health", withRecovery(withAPIVersion(APIVersion1, healthV1Handler)))
}

// setupManagerRoutes configures manager-specific routes
func setupManagerRoutes(mux *http.ServeMux) {
	// User management
	mux.HandleFunc("/approve-users", withRecovery(requireAuth(requireRole("manager")(requireDatabase(approveUsersHandler)))))
	mux.HandleFunc("/approve-user", withRecovery(requireAuth(requireRole("manager")(requireDatabase(approveUserHandler)))))
	mux.HandleFunc("/manage-users", withRecovery(requireAuth(requireRole("manager")(requireDatabase(manageUsersHandler)))))
	mux.HandleFunc("/edit-user", withRecovery(requireAuth(requireRole("manager")(requireDatabase(editUserHandler)))))
	mux.HandleFunc("/delete-user", withRecovery(requireAuth(requireRole("manager")(requireDatabase(deleteUserHandler)))))

	// ECSE Management Routes
	mux.HandleFunc("/import-ecse", withRecovery(requireAuth(requireRole("manager")(requireDatabase(importECSEHandler)))))
	mux.HandleFunc("/view-ecse-reports", withRecovery(requireAuth(requireRole("manager")(requireDatabase(viewECSEReportsHandler)))))
	mux.HandleFunc("/ecse-student/", withRecovery(requireAuth(requireRole("manager")(requireDatabase(viewECSEStudentHandler)))))
	mux.HandleFunc("/edit-ecse-student", withRecovery(requireAuth(requireRole("manager")(requireDatabase(editECSEStudentHandler)))))
	mux.HandleFunc("/export-ecse", withRecovery(requireAuth(requireRole("manager")(requireDatabase(exportECSEHandler)))))

	// Dashboard
	mux.HandleFunc("/manager-dashboard", withRecovery(requireAuth(requireRole("manager")(requireDatabase(managerDashboardHandler)))))
	mux.HandleFunc("/analytics-dashboard", withRecovery(requireAuth(requireRole("manager")(requireDatabase(analyticsDashboardHandler)))))
	mux.HandleFunc("/report-builder", withRecovery(requireAuth(requireRole("manager")(requireDatabase(reportBuilderHandler)))))
	mux.HandleFunc("/db-monitor", withRecovery(requireAuth(requireRole("manager")(dbMonitorHandler))))
	
	// ECSE Management
	mux.HandleFunc("/ecse-dashboard", withRecovery(requireAuth(requireRole("manager")(requireDatabase(ecseDashboardHandler)))))
	mux.HandleFunc("/ecse-student", withRecovery(requireAuth(requireRole("manager")(requireDatabase(ecseStudentDetailsHandler)))))
	mux.HandleFunc("/add-ecse-service", withRecovery(requireAuth(requireRole("manager")(requireDatabase(addECSEServiceHandler)))))
	mux.HandleFunc("/add-sample-ecse-data", withRecovery(requireAuth(requireRole("manager")(requireDatabase(addSampleECSEDataHandler)))))
	mux.HandleFunc("/add-sample-fleet-data", withRecovery(requireAuth(requireRole("manager")(requireDatabase(addSampleFleetDataHandler)))))
	mux.HandleFunc("/add-sample-fuel-data", withRecovery(requireAuth(requireRole("manager")(requireDatabase(addSampleFuelDataHandler)))))
	mux.HandleFunc("/generate-mileage-reports", withRecovery(requireAuth(requireRole("manager")(requireDatabase(generateMileageReportsFromLogsHandler)))))
	mux.HandleFunc("/fix-tables", withRecovery(requireAuth(requireRole("manager")(requireDatabase(fixTablesHandler)))))
	mux.HandleFunc("/data-status", withRecovery(requireAuth(requireRole("manager")(requireDatabase(dataStatusHandler)))))
	mux.HandleFunc("/check-db", withRecovery(requireAuth(requireRole("manager")(requireDatabase(checkDatabaseHandler)))))
	
	// Fuel Management
	mux.HandleFunc("/fuel-records", withRecovery(requireAuth(requireDatabase(fuelRecordsHandler))))
	mux.HandleFunc("/fuel-tracking", withRecovery(requireAuth(requireDatabase(fuelRecordsHandler))))
	mux.HandleFunc("/add-fuel-record", withRecovery(requireAuth(requireDatabase(addFuelRecordHandler))))
	mux.HandleFunc("/fuel-analytics", withRecovery(requireAuth(requireRole("manager")(requireDatabase(fuelAnalyticsHandler)))))

	// Fleet management - Available to both managers and drivers with proper permissions
	mux.HandleFunc("/fleet", withRecovery(requireAuth(requireDatabase(fleetHandler))))
	mux.HandleFunc("/company-fleet", withRecovery(requireAuth(requireDatabase(companyFleetHandler))))
	mux.HandleFunc("/fleet-vehicles", withRecovery(requireAuth(requireRole("manager")(requireDatabase(fleetVehiclesHandler)))))
	mux.HandleFunc("/update-vehicle-status", withRecovery(requireAuth(requireDatabase(updateVehicleStatusHandler))))
	mux.HandleFunc("/add-bus", withRecovery(requireAuth(requireRole("manager")(requireDatabase(addBusHandler)))))
	mux.HandleFunc("/add-bus-wizard", withRecovery(requireAuth(requireRole("manager")(requireDatabase(addBusWizardHandler)))))

	// Maintenance - Available to both managers and drivers
	mux.HandleFunc("/bus-maintenance/", withRecovery(requireAuth(requireDatabase(busMaintenanceHandler))))
	mux.HandleFunc("/vehicle-maintenance/", withRecovery(requireAuth(requireDatabase(vehicleMaintenanceHandler))))
	mux.HandleFunc("/maintenance-records", withRecovery(requireAuth(requireRole("manager")(requireDatabase(maintenanceRecordsHandler)))))
	mux.HandleFunc("/service-records", withRecovery(requireAuth(requireRole("manager")(requireDatabase(serviceRecordsHandler)))))
	mux.HandleFunc("/save-maintenance-record", withRecovery(requireAuth(requireDatabase(saveMaintenanceRecordHandler))))
	mux.HandleFunc("/maintenance-wizard", withRecovery(requireAuth(requireDatabase(maintenanceWizardHandler))))
	mux.HandleFunc("/save-maintenance-wizard", withRecovery(requireAuth(requireDatabase(saveMaintenanceWizardHandler))))

	// Route management
	mux.HandleFunc("/assign-routes", withRecovery(requireAuth(requireRole("manager")(requireDatabase(assignRoutesHandler)))))
	mux.HandleFunc("/route-assignment-wizard", withRecovery(requireAuth(requireRole("manager")(requireDatabase(routeAssignmentWizardHandler)))))
	mux.HandleFunc("/assign-route-wizard", withRecovery(requireAuth(requireRole("manager")(requireDatabase(assignRouteWizardHandler)))))
	mux.HandleFunc("/assign-route", withRecovery(requireAuth(requireRole("manager")(requireDatabase(assignRouteHandler)))))
	mux.HandleFunc("/unassign-route", withRecovery(requireAuth(requireRole("manager")(requireDatabase(unassignRouteHandler)))))
	mux.HandleFunc("/add-route", withRecovery(requireAuth(requireRole("manager")(requireDatabase(addRouteHandler)))))
	mux.HandleFunc("/edit-route", withRecovery(requireAuth(requireRole("manager")(requireDatabase(editRouteHandler)))))
	mux.HandleFunc("/delete-route", withRecovery(requireAuth(requireRole("manager")(requireDatabase(deleteRouteHandler)))))
	
	// Wizard API endpoints
	mux.HandleFunc("/api/available-drivers", withRecovery(requireAuth(requireRole("manager")(requireDatabase(availableDriversHandler)))))
	mux.HandleFunc("/api/available-buses", withRecovery(requireAuth(requireRole("manager")(requireDatabase(availableBusesHandler)))))
	mux.HandleFunc("/api/available-routes", withRecovery(requireAuth(requireRole("manager")(requireDatabase(availableRoutesHandler)))))
	mux.HandleFunc("/api/check-assignment-conflicts", withRecovery(requireAuth(requireRole("manager")(requireDatabase(checkAssignmentConflictsHandler)))))
	mux.HandleFunc("/api/vehicle-mileage/", withRecovery(requireAuth(requireDatabase(vehicleMileageHandler))))
	mux.HandleFunc("/api/last-maintenance/", withRecovery(requireAuth(requireDatabase(lastMaintenanceHandler))))
	mux.HandleFunc("/api/maintenance-vendors", withRecovery(requireAuth(requireDatabase(maintenanceVendorsHandler))))
	mux.HandleFunc("/api/analyze-import-file", withRecovery(requireAuth(requireRole("manager")(requireDatabase(analyzeImportFileHandler)))))
	mux.HandleFunc("/api/preview-import", withRecovery(requireAuth(requireRole("manager")(requireDatabase(previewImportHandler)))))
	
	// Auto-complete API endpoints
	mux.HandleFunc("/api/search-buses", withRecovery(requireAuth(requireDatabase(searchBusesHandler))))
	mux.HandleFunc("/api/search-drivers", withRecovery(requireAuth(requireDatabase(searchDriversHandler))))
	mux.HandleFunc("/api/search-students", withRecovery(requireAuth(requireDatabase(searchStudentsHandler))))
	mux.HandleFunc("/api/search-addresses", withRecovery(requireAuth(requireDatabase(searchAddressesHandler))))
	mux.HandleFunc("/api/suggest-models", withRecovery(requireAuth(requireDatabase(suggestModelsHandler))))

	// Mileage reports
	mux.HandleFunc("/import-mileage", withRecovery(requireAuth(requireRole("manager")(requireDatabase(importMileageHandler)))))
	mux.HandleFunc("/view-mileage-reports", withRecovery(requireAuth(requireRole("manager")(requireDatabase(viewMileageReportsHandler)))))
	mux.HandleFunc("/export-mileage", withRecovery(requireAuth(requireRole("manager")(requireDatabase(exportMileageHandler)))))
	mux.HandleFunc("/mileage-report-generator", withRecovery(requireAuth(requireRole("manager")(requireDatabase(mileageReportGeneratorHandler)))))
	mux.HandleFunc("/monthly-mileage-reports", withRecovery(requireAuth(requireRole("manager")(requireDatabase(monthlyMileageReportsHandler)))))

	// Enhanced Import System
	mux.HandleFunc("/import-data-wizard", withRecovery(requireAuth(requireRole("manager")(requireDatabase(importDataWizardHandler)))))
	mux.HandleFunc("/api/import/analyze", withRecovery(requireAuth(requireRole("manager")(requireDatabase(importAnalyzeHandler)))))
	mux.HandleFunc("/api/import/validate", withRecovery(requireAuth(requireRole("manager")(requireDatabase(importValidateHandler)))))
	mux.HandleFunc("/api/import/execute", withRecovery(requireAuth(requireRole("manager")(requireDatabase(importExecuteHandler)))))
	
	// Legacy Import System (temporarily disabled)
	// mux.HandleFunc("/import", withRecovery(requireAuth(requireRole("manager")(requireDatabase(importHandler)))))
	// mux.HandleFunc("/import/mapping", withRecovery(requireAuth(requireRole("manager")(requireDatabase(importMappingHandler)))))
	// mux.HandleFunc("/import/preview", withRecovery(requireAuth(requireRole("manager")(requireDatabase(importPreviewHandler)))))
	// mux.HandleFunc("/import/execute", withRecovery(requireAuth(requireRole("manager")(requireDatabase(importExecuteHandler)))))
	// mux.HandleFunc("/import/history", withRecovery(requireAuth(requireRole("manager")(requireDatabase(importHistoryHandler)))))
	// mux.HandleFunc("/import/details", withRecovery(requireAuth(requireRole("manager")(requireDatabase(importDetailsHandler)))))
	// mux.HandleFunc("/import/rollback", withRecovery(requireAuth(requireRole("manager")(requireDatabase(importRollbackHandler)))))
	// mux.HandleFunc("/api/import", withRecovery(requireAuth(requireRole("manager")(requireDatabase(importAPIHandler)))))
	// mux.HandleFunc("/api/import/auto-map", withRecovery(requireAuth(requireRole("manager")(requireDatabase(autoMapHandler)))))

	// Export System
	mux.HandleFunc("/export/templates", withRecovery(requireAuth(requireRole("manager")(requireDatabase(exportTemplateHandler)))))
	mux.HandleFunc("/export/template", withRecovery(requireAuth(requireRole("manager")(requireDatabase(exportTemplateHandler)))))
	mux.HandleFunc("/export/data", withRecovery(requireAuth(requireRole("manager")(requireDatabase(exportDataHandler)))))
	mux.HandleFunc("/export/scheduled", withRecovery(requireAuth(requireRole("manager")(requireDatabase(scheduledExportsHandler)))))
	mux.HandleFunc("/export/scheduled/edit", withRecovery(requireAuth(requireRole("manager")(requireDatabase(scheduledExportEditHandler)))))
	mux.HandleFunc("/export/scheduled/delete", withRecovery(requireAuth(requireRole("manager")(requireDatabase(scheduledExportDeleteHandler)))))
	mux.HandleFunc("/export/scheduled/run", withRecovery(requireAuth(requireRole("manager")(requireDatabase(scheduledExportRunHandler)))))

	// PDF Reports
	mux.HandleFunc("/api/reports/pdf", withRecovery(requireAuth(requireDatabase(pdfReportHandler))))
	mux.HandleFunc("/api/reports/pdf/custom", withRecovery(requireAuth(requireRole("manager")(requireDatabase(pdfCustomReportHandler)))))

	// Driver profile
	mux.HandleFunc("/driver/", withRecovery(requireAuth(requireRole("manager")(requireDatabase(driverProfileHandler)))))
}

// setupDriverRoutes configures driver-specific routes
func setupDriverRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/driver-dashboard", withRecovery(requireAuth(requireRole("driver")(requireDatabase(driverDashboardHandler)))))
	mux.HandleFunc("/save-log", withRecovery(requireAuth(requireRole("driver")(requireDatabase(saveLogHandler)))))

	// Student management
	mux.HandleFunc("/students", withRecovery(requireAuth(requireRole("driver")(requireDatabase(studentsHandler)))))
	mux.HandleFunc("/add-student", withRecovery(requireAuth(requireRole("driver")(requireDatabase(addStudentHandler)))))
	mux.HandleFunc("/add-student-wizard", withRecovery(requireAuth(requireRole("driver")(requireDatabase(addStudentWizardHandler)))))
	mux.HandleFunc("/edit-student", withRecovery(requireAuth(requireRole("driver")(requireDatabase(editStudentHandler)))))
	mux.HandleFunc("/remove-student", withRecovery(requireAuth(requireRole("driver")(requireDatabase(removeStudentHandler)))))
}

// ============= UTILITY HANDLER =============

func healthCheck(w http.ResponseWriter, r *http.Request) {
	health := struct {
		Status       string `json:"status"`
		Service      string `json:"service"`
		Timestamp    string `json:"timestamp"`
		Database     string `json:"database"`
		Version      string `json:"version"`
		SessionCount int    `json:"active_sessions"`
		UserCount    *int   `json:"user_count,omitempty"`
		AdminExists  *bool  `json:"admin_exists,omitempty"`
		DBError      string `json:"db_error,omitempty"`
		TableExists  *bool  `json:"users_table_exists,omitempty"`
	}{
		Status:       "ok",
		Service:      "bus-fleet-management",
		Timestamp:    time.Now().Format(time.RFC3339),
		Database:     "connected",
		Version:      "2.0.0",
		SessionCount: GetActiveSessionCount(),
	}

	// Check database connection
	if db == nil {
		health.Status = "degraded"
		health.Database = "not_initialized"
		health.DBError = "Database connection is nil"
	} else if err := db.Ping(); err != nil {
		health.Status = "degraded"
		health.Database = "disconnected"
		health.DBError = err.Error()
	} else {
		// Database is connected, run additional checks

		// Check if users table exists
		var tableExists bool
		if err := db.Get(&tableExists, `
			SELECT EXISTS (
				SELECT FROM information_schema.tables 
				WHERE table_schema = 'public' 
				AND table_name = 'users'
			)
		`); err == nil {
			health.TableExists = &tableExists
		}

		// Count users
		var userCount int
		if err := db.Get(&userCount, "SELECT COUNT(*) FROM users"); err == nil {
			health.UserCount = &userCount
		} else if health.DBError == "" {
			health.DBError = "Failed to count users: " + err.Error()
		}

		// Check if admin exists
		var adminExists bool
		if err := db.Get(&adminExists, "SELECT EXISTS(SELECT 1 FROM users WHERE username = 'admin')"); err == nil {
			health.AdminExists = &adminExists
		}
	}

	w.Header().Set("Content-Type", "application/json")

	if health.Status == "ok" {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	json.NewEncoder(w).Encode(health)
}

// ============= HELPER FUNCTIONS =============

// getSessionCSRFToken gets the CSRF token from the current session
func getSessionCSRFToken(r *http.Request) string {
	cookie, err := r.Cookie(SessionCookieName)
	if err != nil {
		return ""
	}

	session, err := GetSecureSession(cookie.Value)
	if err != nil {
		return ""
	}

	return session.CSRFToken
}

// validateCSRF validates the CSRF token - FIXED for multipart forms
func validateCSRF(r *http.Request) bool {
	// Get session cookie
	cookie, err := r.Cookie(SessionCookieName)
	if err != nil {
		// No session cookie = no authenticated user
		log.Printf("CSRF validation failed: no session cookie")
		return false
	}

	// Get session
	session, err := GetSecureSession(cookie.Value)
	if err != nil {
		// Invalid session = fail validation
		log.Printf("CSRF validation failed: invalid session")
		return false
	}

	// Get submitted token from form or header
	submittedToken := r.FormValue("csrf_token")
	if submittedToken == "" {
		// Also check header for AJAX requests
		submittedToken = r.Header.Get("X-CSRF-Token")
	}

	// Debug logging
	if submittedToken == "" {
		log.Printf("CSRF validation failed: no token submitted")
		return false
	}

	if session.CSRFToken != submittedToken {
		log.Printf("CSRF validation failed: token mismatch - expected: %s, got: %s", session.CSRFToken, submittedToken)
		return false
	}

	return true
}

// Graceful shutdown
func gracefulShutdown(server *http.Server) {
	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down server...")

	// Give connections 30 seconds to finish
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	log.Println("Server shutdown complete")
}

// Helper functions
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Development mode check
func isDevelopment() bool {
	return os.Getenv("APP_ENV") == "development"
}

