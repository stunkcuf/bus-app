package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	_ "github.com/lib/pq"
	"github.com/joho/godotenv"
)

type HealthCheckResult struct {
	Component   string        `json:"component"`
	Status      string        `json:"status"`
	Message     string        `json:"message"`
	Details     interface{}   `json:"details,omitempty"`
	Duration    time.Duration `json:"duration"`
	Timestamp   time.Time     `json:"timestamp"`
}

type PageTestResult struct {
	URL        string `json:"url"`
	StatusCode int    `json:"status_code"`
	DataFound  bool   `json:"data_found"`
	Error      string `json:"error,omitempty"`
	DataCount  int    `json:"data_count,omitempty"`
}

type SystemHealthReport struct {
	OverallStatus string                   `json:"overall_status"`
	Timestamp     time.Time                `json:"timestamp"`
	Components    []HealthCheckResult      `json:"components"`
	Pages         []PageTestResult         `json:"pages"`
	DataIntegrity map[string]interface{}   `json:"data_integrity"`
}

func main() {
	fmt.Println("🏥 SYSTEM HEALTH CHECK - Fleet Management System")
	fmt.Println("=" + strings.Repeat("=", 60))
	
	// Load environment
	godotenv.Load("../.env")
	
	report := SystemHealthReport{
		Timestamp:     time.Now(),
		Components:    []HealthCheckResult{},
		Pages:         []PageTestResult{},
		DataIntegrity: make(map[string]interface{}),
	}
	
	// 1. Database Connectivity
	dbResult := checkDatabase()
	report.Components = append(report.Components, dbResult)
	
	if dbResult.Status != "healthy" {
		report.OverallStatus = "critical"
		printReport(report)
		return
	}
	
	// 2. Table Health Checks
	tableResults := checkTables()
	report.Components = append(report.Components, tableResults...)
	
	// 3. Data Integrity Checks
	integrityResults := checkDataIntegrity()
	report.DataIntegrity = integrityResults
	
	// 4. API Endpoint Tests
	apiResults := testAPIEndpoints()
	report.Components = append(report.Components, apiResults...)
	
	// 5. Page Load Tests
	pageResults := testPageLoads()
	report.Pages = pageResults
	
	// Calculate overall status
	report.OverallStatus = calculateOverallStatus(report)
	
	// Print and save report
	printReport(report)
	saveReport(report)
}

func checkDatabase() HealthCheckResult {
	start := time.Now()
	result := HealthCheckResult{
		Component: "Database Connection",
		Timestamp: time.Now(),
	}
	
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		result.Status = "critical"
		result.Message = "DATABASE_URL not configured"
		result.Duration = time.Since(start)
		return result
	}
	
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		result.Status = "critical"
		result.Message = fmt.Sprintf("Failed to connect: %v", err)
		result.Duration = time.Since(start)
		return result
	}
	defer db.Close()
	
	// Test connection
	err = db.Ping()
	if err != nil {
		result.Status = "critical"
		result.Message = fmt.Sprintf("Database ping failed: %v", err)
		result.Duration = time.Since(start)
		return result
	}
	
	// Get connection stats
	var dbName, dbVersion string
	err = db.QueryRow("SELECT current_database(), version()").Scan(&dbName, &dbVersion)
	if err == nil {
		result.Details = map[string]string{
			"database": dbName,
			"version":  strings.Split(dbVersion, " ")[0] + " " + strings.Split(dbVersion, " ")[1],
		}
	}
	
	result.Status = "healthy"
	result.Message = "Database connection successful"
	result.Duration = time.Since(start)
	return result
}

func checkTables() []HealthCheckResult {
	results := []HealthCheckResult{}
	
	tables := []struct {
		name     string
		critical bool
	}{
		{"users", true},
		{"buses", true},
		{"vehicles", true},
		{"students", true},
		{"routes", true},
		{"route_assignments", true},
		{"maintenance_records", false},
		{"service_records", false},
		{"fuel_records", false},
		{"ecse_students", false},
		{"monthly_mileage_reports", false},
		{"driver_logs", false},
	}
	
	dbURL := os.Getenv("DATABASE_URL")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return results
	}
	defer db.Close()
	
	for _, table := range tables {
		start := time.Now()
		result := HealthCheckResult{
			Component: fmt.Sprintf("Table: %s", table.name),
			Timestamp: time.Now(),
		}
		
		// Check if table exists and get count
		var count int
		err := db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", table.name)).Scan(&count)
		if err != nil {
			if table.critical {
				result.Status = "critical"
			} else {
				result.Status = "warning"
			}
			result.Message = fmt.Sprintf("Table check failed: %v", err)
		} else {
			result.Status = "healthy"
			result.Message = fmt.Sprintf("%d records", count)
			result.Details = map[string]int{"record_count": count}
			
			// Warning if critical table is empty
			if count == 0 && table.critical {
				result.Status = "warning"
				result.Message = "Critical table is empty"
			}
		}
		
		result.Duration = time.Since(start)
		results = append(results, result)
	}
	
	return results
}

func checkDataIntegrity() map[string]interface{} {
	integrity := make(map[string]interface{})
	
	dbURL := os.Getenv("DATABASE_URL")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		integrity["error"] = err.Error()
		return integrity
	}
	defer db.Close()
	
	// Check for orphaned records
	checks := []struct {
		name  string
		query string
	}{
		{
			"orphaned_students",
			`SELECT COUNT(*) FROM students s 
			 LEFT JOIN routes r ON s.route_id = r.id 
			 WHERE s.route_id IS NOT NULL AND r.id IS NULL`,
		},
		{
			"unassigned_buses",
			`SELECT COUNT(*) FROM buses b 
			 LEFT JOIN route_assignments ra ON b.bus_id = ra.bus_id 
			 WHERE ra.bus_id IS NULL AND b.status = 'active'`,
		},
		{
			"invalid_maintenance_vehicles",
			`SELECT COUNT(*) FROM maintenance_records mr 
			 WHERE mr.vehicle_id NOT IN (
				SELECT bus_id FROM buses 
				UNION 
				SELECT CAST(vehicle_number AS VARCHAR) FROM vehicles
			 )`,
		},
	}
	
	for _, check := range checks {
		var count int
		err := db.QueryRow(check.query).Scan(&count)
		if err != nil {
			integrity[check.name] = map[string]interface{}{
				"status": "error",
				"error":  err.Error(),
			}
		} else {
			status := "ok"
			if count > 0 {
				status = "warning"
			}
			integrity[check.name] = map[string]interface{}{
				"status": status,
				"count":  count,
			}
		}
	}
	
	return integrity
}

func testAPIEndpoints() []HealthCheckResult {
	results := []HealthCheckResult{}
	
	// Get base URL
	port := os.Getenv("PORT")
	if port == "" {
		port = "5003"
	}
	baseURL := fmt.Sprintf("http://localhost:%s", port)
	
	// Test endpoints
	endpoints := []struct {
		path   string
		method string
	}{
		{"/health", "GET"},
		{"/api/health", "GET"},
		{"/api/v1/health", "GET"},
		{"/api/dashboard/stats", "GET"},
		{"/api/fleet-status", "GET"},
	}
	
	client := &http.Client{
		Timeout: 5 * time.Second,
	}
	
	for _, endpoint := range endpoints {
		start := time.Now()
		result := HealthCheckResult{
			Component: fmt.Sprintf("API: %s %s", endpoint.method, endpoint.path),
			Timestamp: time.Now(),
		}
		
		req, err := http.NewRequest(endpoint.method, baseURL+endpoint.path, nil)
		if err != nil {
			result.Status = "error"
			result.Message = fmt.Sprintf("Failed to create request: %v", err)
			result.Duration = time.Since(start)
			results = append(results, result)
			continue
		}
		
		// Add auth cookie if needed
		// cookie := &http.Cookie{Name: "session", Value: "test-session"}
		// req.AddCookie(cookie)
		
		resp, err := client.Do(req)
		if err != nil {
			result.Status = "error"
			result.Message = fmt.Sprintf("Request failed: %v", err)
		} else {
			defer resp.Body.Close()
			
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				result.Status = "healthy"
				result.Message = fmt.Sprintf("Status: %d", resp.StatusCode)
			} else if resp.StatusCode == 401 || resp.StatusCode == 403 {
				result.Status = "warning"
				result.Message = fmt.Sprintf("Auth required: %d", resp.StatusCode)
			} else {
				result.Status = "error"
				result.Message = fmt.Sprintf("Status: %d", resp.StatusCode)
			}
			
			result.Details = map[string]int{"status_code": resp.StatusCode}
		}
		
		result.Duration = time.Since(start)
		results = append(results, result)
	}
	
	return results
}

func testPageLoads() []PageTestResult {
	results := []PageTestResult{}
	
	// Pages to test
	pages := []string{
		"/",
		"/fleet",
		"/maintenance-records",
		"/service-records",
		"/fuel-records",
		"/students",
		"/ecse-dashboard",
		"/monthly-mileage-reports",
		"/manager-dashboard",
		"/driver-dashboard",
	}
	
	port := os.Getenv("PORT")
	if port == "" {
		port = "5003"
	}
	baseURL := fmt.Sprintf("http://localhost:%s", port)
	
	client := &http.Client{
		Timeout: 5 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // Don't follow redirects
		},
	}
	
	for _, page := range pages {
		result := PageTestResult{
			URL: page,
		}
		
		resp, err := client.Get(baseURL + page)
		if err != nil {
			result.Error = err.Error()
			results = append(results, result)
			continue
		}
		defer resp.Body.Close()
		
		result.StatusCode = resp.StatusCode
		
		// Check if it's a redirect (likely to login)
		if resp.StatusCode == 302 || resp.StatusCode == 303 {
			location := resp.Header.Get("Location")
			if location == "/" || strings.Contains(location, "login") {
				result.Error = "Requires authentication"
			}
		}
		
		results = append(results, result)
	}
	
	return results
}

func calculateOverallStatus(report SystemHealthReport) string {
	criticalCount := 0
	warningCount := 0
	
	for _, component := range report.Components {
		switch component.Status {
		case "critical":
			criticalCount++
		case "warning", "error":
			warningCount++
		}
	}
	
	// Check data integrity
	for _, check := range report.DataIntegrity {
		if checkMap, ok := check.(map[string]interface{}); ok {
			if status, ok := checkMap["status"].(string); ok {
				if status == "error" {
					criticalCount++
				} else if status == "warning" {
					warningCount++
				}
			}
		}
	}
	
	if criticalCount > 0 {
		return "critical"
	} else if warningCount > 2 {
		return "degraded"
	} else if warningCount > 0 {
		return "warning"
	}
	
	return "healthy"
}

func printReport(report SystemHealthReport) {
	fmt.Printf("\n📊 HEALTH CHECK REPORT\n")
	fmt.Printf("Timestamp: %s\n", report.Timestamp.Format("2006-01-02 15:04:05"))
	fmt.Printf("Overall Status: %s\n\n", strings.ToUpper(report.OverallStatus))
	
	// Component Status
	fmt.Println("🔧 COMPONENT STATUS:")
	fmt.Println(strings.Repeat("-", 80))
	for _, component := range report.Components {
		statusIcon := getStatusIcon(component.Status)
		fmt.Printf("%s %-40s %s (%v)\n", 
			statusIcon, 
			component.Component, 
			component.Message,
			component.Duration)
	}
	
	// Data Integrity
	fmt.Println("\n🔍 DATA INTEGRITY:")
	fmt.Println(strings.Repeat("-", 80))
	for check, result := range report.DataIntegrity {
		if resultMap, ok := result.(map[string]interface{}); ok {
			status := resultMap["status"].(string)
			statusIcon := getStatusIcon(status)
			fmt.Printf("%s %-40s", statusIcon, check)
			
			if count, ok := resultMap["count"]; ok {
				fmt.Printf(" Count: %v", count)
			}
			if err, ok := resultMap["error"]; ok {
				fmt.Printf(" Error: %v", err)
			}
			fmt.Println()
		}
	}
	
	// Page Tests
	fmt.Println("\n📄 PAGE ACCESSIBILITY:")
	fmt.Println(strings.Repeat("-", 80))
	for _, page := range report.Pages {
		statusIcon := "✅"
		if page.StatusCode != 200 {
			if page.StatusCode == 302 || page.StatusCode == 401 {
				statusIcon = "🔒"
			} else {
				statusIcon = "❌"
			}
		}
		
		fmt.Printf("%s %-30s Status: %d", statusIcon, page.URL, page.StatusCode)
		if page.Error != "" {
			fmt.Printf(" (%s)", page.Error)
		}
		fmt.Println()
	}
	
	// Summary
	fmt.Printf("\n📈 SUMMARY:\n")
	fmt.Println(strings.Repeat("-", 80))
	
	healthyCount := 0
	warningCount := 0
	criticalCount := 0
	
	for _, component := range report.Components {
		switch component.Status {
		case "healthy":
			healthyCount++
		case "warning":
			warningCount++
		case "critical", "error":
			criticalCount++
		}
	}
	
	fmt.Printf("✅ Healthy: %d\n", healthyCount)
	fmt.Printf("⚠️  Warning: %d\n", warningCount)
	fmt.Printf("❌ Critical: %d\n", criticalCount)
}

func getStatusIcon(status string) string {
	switch status {
	case "healthy", "ok":
		return "✅"
	case "warning":
		return "⚠️"
	case "critical", "error":
		return "❌"
	default:
		return "❓"
	}
}

func saveReport(report SystemHealthReport) {
	// Save JSON report
	filename := fmt.Sprintf("health_report_%s.json", time.Now().Format("20060102_150405"))
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		log.Printf("Failed to marshal report: %v", err)
		return
	}
	
	err = os.WriteFile(filename, data, 0644)
	if err != nil {
		log.Printf("Failed to save report: %v", err)
		return
	}
	
	fmt.Printf("\n💾 Report saved to: %s\n", filename)
}