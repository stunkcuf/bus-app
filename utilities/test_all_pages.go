package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type PageTest struct {
	URL            string
	Method         string
	RequiresAuth   bool
	RequiredRole   string
	ExpectedStatus int
	DataCheck      func(body string) PageTestResult
}

type PageTestResult struct {
	URL           string    `json:"url"`
	Method        string    `json:"method"`
	Status        string    `json:"status"`
	StatusCode    int       `json:"status_code"`
	ResponseTime  int64     `json:"response_time_ms"`
	DataFound     bool      `json:"data_found"`
	RecordCount   int       `json:"record_count"`
	ErrorMessage  string    `json:"error,omitempty"`
	Details       string    `json:"details,omitempty"`
	Timestamp     time.Time `json:"timestamp"`
}

func main() {
	fmt.Println("🧪 COMPREHENSIVE PAGE TESTING")
	fmt.Println("=" + strings.Repeat("=", 60))
	
	// Load environment
	godotenv.Load("../.env")
	
	// Setup
	baseURL := "http://localhost:5003"
	if port := os.Getenv("PORT"); port != "" {
		baseURL = fmt.Sprintf("http://localhost:%s", port)
	}
	
	// Create HTTP client with cookie jar
	jar, _ := cookiejar.New(nil)
	client := &http.Client{
		Jar:     jar,
		Timeout: 10 * time.Second,
	}
	
	// Test authentication first
	fmt.Println("\n🔐 Testing Authentication...")
	if !testLogin(client, baseURL) {
		log.Fatal("Authentication failed - cannot proceed with tests")
	}
	
	// Define all pages to test
	pages := getPageTests()
	
	// Run tests
	results := []PageTestResult{}
	fmt.Println("\n📄 Testing Pages...")
	fmt.Println(strings.Repeat("-", 80))
	
	for _, page := range pages {
		result := testPage(client, baseURL, page)
		results = append(results, result)
		
		// Print result
		statusIcon := "✅"
		if result.Status != "success" {
			if result.Status == "warning" {
				statusIcon = "⚠️"
			} else {
				statusIcon = "❌"
			}
		}
		
		fmt.Printf("%s %-40s %d (%dms)", statusIcon, result.URL, result.StatusCode, result.ResponseTime)
		if result.DataFound {
			fmt.Printf(" [%d records]", result.RecordCount)
		}
		if result.ErrorMessage != "" {
			fmt.Printf(" - %s", result.ErrorMessage)
		}
		fmt.Println()
	}
	
	// Generate summary
	generateSummary(results)
	
	// Save detailed report
	saveReport(results)
}

func testLogin(client *http.Client, baseURL string) bool {
	// Try admin login
	loginData := url.Values{
		"username": {"admin"},
		"password": {"Headstart1"},
	}
	
	resp, err := client.PostForm(baseURL+"/", loginData)
	if err != nil {
		log.Printf("Login request failed: %v", err)
		return false
	}
	defer resp.Body.Close()
	
	// Check if redirected to dashboard
	if resp.StatusCode == 302 || resp.StatusCode == 303 {
		location := resp.Header.Get("Location")
		if strings.Contains(location, "dashboard") {
			fmt.Println("✅ Login successful")
			return true
		}
	}
	
	// Check response body for success indicators
	body, _ := io.ReadAll(resp.Body)
	if strings.Contains(string(body), "dashboard") || strings.Contains(string(body), "Welcome") {
		fmt.Println("✅ Login successful")
		return true
	}
	
	fmt.Println("❌ Login failed")
	return false
}

func getPageTests() []PageTest {
	return []PageTest{
		// Dashboard pages
		{
			URL:            "/manager-dashboard",
			Method:         "GET",
			RequiresAuth:   true,
			RequiredRole:   "manager",
			ExpectedStatus: 200,
			DataCheck:      checkDashboardData,
		},
		{
			URL:            "/driver-dashboard",
			Method:         "GET",
			RequiresAuth:   true,
			RequiredRole:   "driver",
			ExpectedStatus: 200,
			DataCheck:      checkDriverDashboard,
		},
		
		// Fleet Management
		{
			URL:            "/fleet",
			Method:         "GET",
			RequiresAuth:   true,
			ExpectedStatus: 200,
			DataCheck:      checkFleetData,
		},
		{
			URL:            "/fleet-vehicles",
			Method:         "GET",
			RequiresAuth:   true,
			ExpectedStatus: 200,
			DataCheck:      checkFleetVehicles,
		},
		{
			URL:            "/company-fleet",
			Method:         "GET",
			RequiresAuth:   true,
			ExpectedStatus: 200,
			DataCheck:      checkCompanyFleet,
		},
		
		// Maintenance
		{
			URL:            "/maintenance-records",
			Method:         "GET",
			RequiresAuth:   true,
			ExpectedStatus: 200,
			DataCheck:      checkMaintenanceRecords,
		},
		{
			URL:            "/service-records",
			Method:         "GET",
			RequiresAuth:   true,
			ExpectedStatus: 200,
			DataCheck:      checkServiceRecords,
		},
		{
			URL:            "/fuel-records",
			Method:         "GET",
			RequiresAuth:   true,
			ExpectedStatus: 200,
			DataCheck:      checkFuelRecords,
		},
		
		// Students & Routes
		{
			URL:            "/students",
			Method:         "GET",
			RequiresAuth:   true,
			ExpectedStatus: 200,
			DataCheck:      checkStudents,
		},
		{
			URL:            "/assign-routes",
			Method:         "GET",
			RequiresAuth:   true,
			RequiredRole:   "manager",
			ExpectedStatus: 200,
			DataCheck:      checkRouteAssignments,
		},
		
		// ECSE
		{
			URL:            "/ecse-dashboard",
			Method:         "GET",
			RequiresAuth:   true,
			RequiredRole:   "manager",
			ExpectedStatus: 200,
			DataCheck:      checkECSEDashboard,
		},
		{
			URL:            "/view-ecse-reports",
			Method:         "GET",
			RequiresAuth:   true,
			RequiredRole:   "manager",
			ExpectedStatus: 200,
			DataCheck:      checkECSEReports,
		},
		
		// Reports
		{
			URL:            "/monthly-mileage-reports",
			Method:         "GET",
			RequiresAuth:   true,
			ExpectedStatus: 200,
			DataCheck:      checkMileageReports,
		},
		{
			URL:            "/fuel-analytics",
			Method:         "GET",
			RequiresAuth:   true,
			ExpectedStatus: 200,
			DataCheck:      checkFuelAnalytics,
		},
		
		// User Management
		{
			URL:            "/users",
			Method:         "GET",
			RequiresAuth:   true,
			RequiredRole:   "manager",
			ExpectedStatus: 200,
			DataCheck:      checkUsers,
		},
		
		// API Endpoints
		{
			URL:            "/api/dashboard/stats",
			Method:         "GET",
			RequiresAuth:   true,
			ExpectedStatus: 200,
			DataCheck:      checkAPIResponse,
		},
		{
			URL:            "/api/fleet-status",
			Method:         "GET",
			RequiresAuth:   true,
			ExpectedStatus: 200,
			DataCheck:      checkAPIResponse,
		},
	}
}

func testPage(client *http.Client, baseURL string, test PageTest) PageTestResult {
	start := time.Now()
	result := PageTestResult{
		URL:       test.URL,
		Method:    test.Method,
		Timestamp: time.Now(),
	}
	
	// Create request
	req, err := http.NewRequest(test.Method, baseURL+test.URL, nil)
	if err != nil {
		result.Status = "error"
		result.ErrorMessage = fmt.Sprintf("Failed to create request: %v", err)
		return result
	}
	
	// Execute request
	resp, err := client.Do(req)
	if err != nil {
		result.Status = "error"
		result.ErrorMessage = fmt.Sprintf("Request failed: %v", err)
		return result
	}
	defer resp.Body.Close()
	
	result.StatusCode = resp.StatusCode
	result.ResponseTime = time.Since(start).Milliseconds()
	
	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		result.Status = "error"
		result.ErrorMessage = fmt.Sprintf("Failed to read response: %v", err)
		return result
	}
	
	// Check status code
	if resp.StatusCode != test.ExpectedStatus {
		if resp.StatusCode == 401 || resp.StatusCode == 403 {
			result.Status = "error"
			result.ErrorMessage = "Unauthorized"
		} else if resp.StatusCode == 404 {
			result.Status = "error"
			result.ErrorMessage = "Page not found"
		} else if resp.StatusCode == 500 {
			result.Status = "error"
			result.ErrorMessage = "Server error"
		} else {
			result.Status = "warning"
			result.ErrorMessage = fmt.Sprintf("Unexpected status: %d", resp.StatusCode)
		}
	} else {
		result.Status = "success"
	}
	
	// Run data check if provided
	if test.DataCheck != nil && result.Status == "success" {
		dataResult := test.DataCheck(string(body))
		result.DataFound = dataResult.DataFound
		result.RecordCount = dataResult.RecordCount
		if dataResult.ErrorMessage != "" {
			result.Status = "warning"
			result.ErrorMessage = dataResult.ErrorMessage
		}
		result.Details = dataResult.Details
	}
	
	return result
}

// Data check functions
func checkDashboardData(body string) PageTestResult {
	result := PageTestResult{}
	
	// Check for key dashboard elements
	if strings.Contains(body, "Total Buses") || strings.Contains(body, "Active Buses") {
		result.DataFound = true
		
		// Try to extract bus count
		re := regexp.MustCompile(`(\d+)\s*buses`)
		if matches := re.FindStringSubmatch(body); len(matches) > 1 {
			fmt.Sscanf(matches[1], "%d", &result.RecordCount)
		}
	}
	
	return result
}

func checkDriverDashboard(body string) PageTestResult {
	result := PageTestResult{}
	
	if strings.Contains(body, "Route Assignments") || strings.Contains(body, "My Routes") {
		result.DataFound = true
	}
	
	return result
}

func checkFleetData(body string) PageTestResult {
	result := PageTestResult{}
	
	// Check for bus data
	busCount := strings.Count(body, "Bus #") + strings.Count(body, "BUS-")
	if busCount > 0 {
		result.DataFound = true
		result.RecordCount = busCount
	}
	
	// Check for "No Buses in Fleet" message
	if strings.Contains(body, "No Buses in Fleet") {
		result.DataFound = false
		result.ErrorMessage = "No buses displayed"
	}
	
	return result
}

func checkFleetVehicles(body string) PageTestResult {
	result := PageTestResult{}
	
	// Extract vehicle count
	re := regexp.MustCompile(`(\d+)\s+vehicles`)
	if matches := re.FindStringSubmatch(body); len(matches) > 1 {
		fmt.Sscanf(matches[1], "%d", &result.RecordCount)
		result.DataFound = result.RecordCount > 0
	}
	
	return result
}

func checkCompanyFleet(body string) PageTestResult {
	result := PageTestResult{}
	
	if strings.Contains(body, "Company Vehicles") || strings.Contains(body, "Fleet Overview") {
		result.DataFound = true
		
		// Count vehicle entries
		vehicleCount := strings.Count(body, "Vehicle #")
		if vehicleCount > 0 {
			result.RecordCount = vehicleCount
		}
	}
	
	return result
}

func checkMaintenanceRecords(body string) PageTestResult {
	result := PageTestResult{}
	
	// Look for total records indicator
	re := regexp.MustCompile(`(\d+)\s+Total Records`)
	if matches := re.FindStringSubmatch(body); len(matches) > 1 {
		fmt.Sscanf(matches[1], "%d", &result.RecordCount)
		result.DataFound = result.RecordCount > 0
	}
	
	// Also check for actual record entries
	if strings.Contains(body, "Work Description") || strings.Contains(body, "Service Date") {
		result.DataFound = true
	}
	
	return result
}

func checkServiceRecords(body string) PageTestResult {
	result := PageTestResult{}
	
	// Look for service record indicators
	re := regexp.MustCompile(`(\d+)\s+Total Records`)
	if matches := re.FindStringSubmatch(body); len(matches) > 1 {
		fmt.Sscanf(matches[1], "%d", &result.RecordCount)
		result.DataFound = result.RecordCount > 0
	}
	
	return result
}

func checkFuelRecords(body string) PageTestResult {
	result := PageTestResult{}
	
	// Check for fuel data
	if strings.Contains(body, "Total Cost") || strings.Contains(body, "gallons") {
		result.DataFound = true
		
		// Extract record count
		re := regexp.MustCompile(`(\d+)\s+records`)
		if matches := re.FindStringSubmatch(body); len(matches) > 1 {
			fmt.Sscanf(matches[1], "%d", &result.RecordCount)
		}
	}
	
	return result
}

func checkStudents(body string) PageTestResult {
	result := PageTestResult{}
	
	// Count student entries
	studentCount := strings.Count(body, "student-item") + strings.Count(body, "Student:")
	if studentCount > 0 {
		result.DataFound = true
		result.RecordCount = studentCount
	}
	
	// Check for student names
	if strings.Contains(body, "First Name") || strings.Contains(body, "Last Name") {
		result.DataFound = true
	}
	
	return result
}

func checkRouteAssignments(body string) PageTestResult {
	result := PageTestResult{}
	
	if strings.Contains(body, "Route Assignments") || strings.Contains(body, "Assign Routes") {
		result.DataFound = true
		
		// Count assignments
		assignmentCount := strings.Count(body, "assignment-item") + strings.Count(body, "Route:")
		if assignmentCount > 0 {
			result.RecordCount = assignmentCount
		}
	}
	
	return result
}

func checkECSEDashboard(body string) PageTestResult {
	result := PageTestResult{}
	
	// Look for ECSE student count
	re := regexp.MustCompile(`(\d+)\s+ECSE Students`)
	if matches := re.FindStringSubmatch(body); len(matches) > 1 {
		fmt.Sscanf(matches[1], "%d", &result.RecordCount)
		result.DataFound = result.RecordCount > 0
	}
	
	if strings.Contains(body, "Active IEPs") || strings.Contains(body, "Transportation Required") {
		result.DataFound = true
	}
	
	return result
}

func checkECSEReports(body string) PageTestResult {
	result := PageTestResult{}
	
	if strings.Contains(body, "ECSE Reports") || strings.Contains(body, "Student Reports") {
		result.DataFound = true
		
		// Count report entries
		reportCount := strings.Count(body, "report-item") + strings.Count(body, "View Report")
		if reportCount > 0 {
			result.RecordCount = reportCount
		}
	}
	
	return result
}

func checkMileageReports(body string) PageTestResult {
	result := PageTestResult{}
	
	// Look for mileage data
	if strings.Contains(body, "Total Miles") || strings.Contains(body, "Mileage Report") {
		result.DataFound = true
		
		// Extract report count
		re := regexp.MustCompile(`(\d+)\s+reports`)
		if matches := re.FindStringSubmatch(body); len(matches) > 1 {
			fmt.Sscanf(matches[1], "%d", &result.RecordCount)
		}
	}
	
	return result
}

func checkFuelAnalytics(body string) PageTestResult {
	result := PageTestResult{}
	
	if strings.Contains(body, "Fuel Analytics") || strings.Contains(body, "Cost Analysis") {
		result.DataFound = true
	}
	
	// Check for chart elements
	if strings.Contains(body, "canvas") || strings.Contains(body, "chart") {
		result.Details = "Charts present"
	}
	
	return result
}

func checkUsers(body string) PageTestResult {
	result := PageTestResult{}
	
	// Count user entries
	userCount := strings.Count(body, "user-item") + strings.Count(body, "@")
	if userCount > 0 {
		result.DataFound = true
		result.RecordCount = userCount
	}
	
	return result
}

func checkAPIResponse(body string) PageTestResult {
	result := PageTestResult{}
	
	// Try to parse as JSON
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(body), &data); err == nil {
		result.DataFound = true
		result.RecordCount = len(data)
		result.Details = "Valid JSON response"
	} else {
		result.ErrorMessage = "Invalid JSON response"
	}
	
	return result
}

func generateSummary(results []PageTestResult) {
	fmt.Println("\n📊 TEST SUMMARY")
	fmt.Println(strings.Repeat("=", 80))
	
	successCount := 0
	warningCount := 0
	errorCount := 0
	dataFoundCount := 0
	totalRecords := 0
	
	for _, result := range results {
		switch result.Status {
		case "success":
			successCount++
		case "warning":
			warningCount++
		case "error":
			errorCount++
		}
		
		if result.DataFound {
			dataFoundCount++
			totalRecords += result.RecordCount
		}
	}
	
	totalTests := len(results)
	successRate := float64(successCount) / float64(totalTests) * 100
	
	fmt.Printf("Total Tests: %d\n", totalTests)
	fmt.Printf("✅ Success: %d (%.1f%%)\n", successCount, successRate)
	fmt.Printf("⚠️  Warning: %d\n", warningCount)
	fmt.Printf("❌ Error: %d\n", errorCount)
	fmt.Printf("\n")
	fmt.Printf("📊 Pages with Data: %d/%d\n", dataFoundCount, totalTests)
	fmt.Printf("📈 Total Records Found: %d\n", totalRecords)
	
	// List problematic pages
	if errorCount > 0 || warningCount > 0 {
		fmt.Println("\n⚠️  PAGES REQUIRING ATTENTION:")
		fmt.Println(strings.Repeat("-", 80))
		
		for _, result := range results {
			if result.Status == "error" || result.Status == "warning" {
				fmt.Printf("• %s: %s\n", result.URL, result.ErrorMessage)
			}
		}
	}
	
	// List pages with no data
	fmt.Println("\n📭 PAGES WITH NO DATA:")
	fmt.Println(strings.Repeat("-", 80))
	
	noDataCount := 0
	for _, result := range results {
		if result.Status == "success" && !result.DataFound {
			fmt.Printf("• %s\n", result.URL)
			noDataCount++
		}
	}
	
	if noDataCount == 0 {
		fmt.Println("All successful pages display data ✅")
	}
}

func saveReport(results []PageTestResult) {
	// Create report
	report := map[string]interface{}{
		"timestamp": time.Now(),
		"results":   results,
		"summary": map[string]interface{}{
			"total_tests": len(results),
			"success":     countByStatus(results, "success"),
			"warning":     countByStatus(results, "warning"),
			"error":       countByStatus(results, "error"),
		},
	}
	
	// Save to file
	filename := fmt.Sprintf("page_test_report_%s.json", time.Now().Format("20060102_150405"))
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
	
	fmt.Printf("\n💾 Detailed report saved to: %s\n", filename)
}

func countByStatus(results []PageTestResult, status string) int {
	count := 0
	for _, result := range results {
		if result.Status == status {
			count++
		}
	}
	return count
}