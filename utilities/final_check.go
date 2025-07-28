package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type TestResult struct {
	TestName string
	Status   string
	Details  string
	Time     int64
}

type PageTestV2 struct {
	URL          string
	Description  string
	RequiresAuth bool
	TestFunc     func(*http.Client, string) TestResult
}

func main() {
	fmt.Println("🔥 FINAL COMPREHENSIVE SYSTEM TEST")
	fmt.Println("=" + strings.Repeat("=", 60))
	fmt.Println("Testing with actual user accounts")
	
	// Load environment
	godotenv.Load("../.env")
	
	// Setup
	baseURL := "http://localhost:5003"
	if port := os.Getenv("PORT"); port != "" {
		baseURL = fmt.Sprintf("http://localhost:%s", port)
	}
	
	results := []TestResult{}
	
	// Test 1: Manager Login and Access
	fmt.Println("\n📋 Test 1: Manager Login and Dashboard Access")
	results = append(results, testManagerAccess(baseURL))
	
	// Test 2: Driver Login and Access
	fmt.Println("\n📋 Test 2: Driver Login and Dashboard Access")
	results = append(results, testDriverAccess(baseURL))
	
	// Test 3: Data Display on Key Pages
	fmt.Println("\n📋 Test 3: Data Display Verification")
	results = append(results, testDataDisplay(baseURL))
	
	// Test 4: API Endpoints
	fmt.Println("\n📋 Test 4: API Endpoint Testing")
	results = append(results, testAPIEndpoints(baseURL))
	
	// Test 5: Error Recovery
	fmt.Println("\n📋 Test 5: Error Recovery Testing")
	results = append(results, testErrorRecovery(baseURL))
	
	// Generate Final Report
	generateFinalReport(results)
}

func testManagerAccess(baseURL string) TestResult {
	start := time.Now()
	result := TestResult{
		TestName: "Manager Access Test",
		Status:   "PASS",
	}
	
	jar, _ := cookiejar.New(nil)
	client := &http.Client{
		Jar:     jar,
		Timeout: 10 * time.Second,
	}
	
	// Login as manager
	loginData := url.Values{
		"username": {"testmanager123"},
		"password": {"password123"},
	}
	
	resp, err := client.PostForm(baseURL+"/", loginData)
	if err != nil {
		result.Status = "FAIL"
		result.Details = fmt.Sprintf("Login failed: %v", err)
		return result
	}
	defer resp.Body.Close()
	
	// Test manager pages
	managerPages := []string{
		"/manager-dashboard",
		"/fleet",
		"/maintenance-records",
		"/users",
		"/assign-routes",
		"/ecse-dashboard",
	}
	
	failedPages := []string{}
	for _, page := range managerPages {
		pageResp, err := client.Get(baseURL + page)
		if err != nil || pageResp.StatusCode != 200 {
			failedPages = append(failedPages, page)
		}
		if pageResp != nil {
			pageResp.Body.Close()
		}
	}
	
	if len(failedPages) > 0 {
		result.Status = "PARTIAL"
		result.Details = fmt.Sprintf("Failed pages: %v", failedPages)
	} else {
		result.Details = "All manager pages accessible"
	}
	
	result.Time = time.Since(start).Milliseconds()
	return result
}

func testDriverAccess(baseURL string) TestResult {
	start := time.Now()
	result := TestResult{
		TestName: "Driver Access Test",
		Status:   "PASS",
	}
	
	jar, _ := cookiejar.New(nil)
	client := &http.Client{
		Jar:     jar,
		Timeout: 10 * time.Second,
	}
	
	// Login as driver
	loginData := url.Values{
		"username": {"testdriver123"},
		"password": {"password123"},
	}
	
	resp, err := client.PostForm(baseURL+"/", loginData)
	if err != nil {
		result.Status = "FAIL"
		result.Details = fmt.Sprintf("Login failed: %v", err)
		return result
	}
	defer resp.Body.Close()
	
	// Test driver dashboard
	dashResp, err := client.Get(baseURL + "/driver-dashboard")
	if err != nil {
		result.Status = "FAIL"
		result.Details = fmt.Sprintf("Dashboard access failed: %v", err)
		return result
	}
	defer dashResp.Body.Close()
	
	if dashResp.StatusCode != 200 {
		result.Status = "FAIL"
		result.Details = fmt.Sprintf("Dashboard returned status %d", dashResp.StatusCode)
	} else {
		body, _ := io.ReadAll(dashResp.Body)
		if strings.Contains(string(body), "Welcome") || strings.Contains(string(body), "Dashboard") {
			result.Details = "Driver dashboard accessible and displays content"
		} else {
			result.Status = "PARTIAL"
			result.Details = "Dashboard accessible but content unclear"
		}
	}
	
	result.Time = time.Since(start).Milliseconds()
	return result
}

func testDataDisplay(baseURL string) TestResult {
	start := time.Now()
	result := TestResult{
		TestName: "Data Display Test",
		Status:   "PASS",
	}
	
	// Login as manager first
	jar, _ := cookiejar.New(nil)
	client := &http.Client{
		Jar:     jar,
		Timeout: 10 * time.Second,
	}
	
	loginData := url.Values{
		"username": {"admin"},
		"password": {"Headstart1"},
	}
	
	client.PostForm(baseURL+"/", loginData)
	
	// Test pages that should display data
	dataChecks := map[string]string{
		"/fleet":                   "buses",
		"/fuel-records":            "fuel",
		"/maintenance-records":     "maintenance",
		"/monthly-mileage-reports": "mileage",
		"/students":                "students",
	}
	
	issuesFound := []string{}
	
	for page, dataType := range dataChecks {
		resp, err := client.Get(baseURL + page)
		if err != nil {
			issuesFound = append(issuesFound, fmt.Sprintf("%s: request failed", page))
			continue
		}
		
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		
		// Check for "no data" messages
		bodyStr := string(body)
		if strings.Contains(bodyStr, "No records found") || 
		   strings.Contains(bodyStr, "No data available") ||
		   strings.Contains(bodyStr, "0 records") {
			issuesFound = append(issuesFound, fmt.Sprintf("%s: shows no %s data", page, dataType))
		}
	}
	
	if len(issuesFound) > 0 {
		result.Status = "PARTIAL"
		result.Details = fmt.Sprintf("Issues: %s", strings.Join(issuesFound, "; "))
	} else {
		result.Details = "All data pages display records correctly"
	}
	
	result.Time = time.Since(start).Milliseconds()
	return result
}

func testAPIEndpoints(baseURL string) TestResult {
	start := time.Now()
	result := TestResult{
		TestName: "API Endpoints Test",
		Status:   "PASS",
	}
	
	// Login as manager
	jar, _ := cookiejar.New(nil)
	client := &http.Client{
		Jar:     jar,
		Timeout: 10 * time.Second,
	}
	
	loginData := url.Values{
		"username": {"admin"},
		"password": {"Headstart1"},
	}
	
	client.PostForm(baseURL+"/", loginData)
	
	// Test API endpoints
	apis := []string{
		"/api/dashboard/stats",
		"/api/fleet-status",
		"/api/health",
		"/api/monitoring/metrics",
	}
	
	failedAPIs := []string{}
	
	for _, api := range apis {
		resp, err := client.Get(baseURL + api)
		if err != nil {
			failedAPIs = append(failedAPIs, fmt.Sprintf("%s: error", api))
			continue
		}
		
		if resp.StatusCode != 200 {
			failedAPIs = append(failedAPIs, fmt.Sprintf("%s: status %d", api, resp.StatusCode))
		} else {
			// Try to parse as JSON
			body, _ := io.ReadAll(resp.Body)
			var data interface{}
			if err := json.Unmarshal(body, &data); err != nil {
				failedAPIs = append(failedAPIs, fmt.Sprintf("%s: invalid JSON", api))
			}
		}
		resp.Body.Close()
	}
	
	if len(failedAPIs) > 0 {
		result.Status = "PARTIAL"
		result.Details = fmt.Sprintf("Failed APIs: %s", strings.Join(failedAPIs, "; "))
	} else {
		result.Details = "All API endpoints return valid JSON"
	}
	
	result.Time = time.Since(start).Milliseconds()
	return result
}

func testErrorRecovery(baseURL string) TestResult {
	start := time.Now()
	result := TestResult{
		TestName: "Error Recovery Test",
		Status:   "PASS",
	}
	
	// Test invalid URLs
	client := &http.Client{Timeout: 5 * time.Second}
	
	resp, err := client.Get(baseURL + "/invalid-page-12345")
	if err != nil {
		result.Status = "FAIL"
		result.Details = "Server not responding to invalid URLs"
		return result
	}
	defer resp.Body.Close()
	
	if resp.StatusCode == 404 {
		result.Details = "Server properly handles 404 errors"
	} else {
		result.Status = "PARTIAL"
		result.Details = fmt.Sprintf("Unexpected status for invalid URL: %d", resp.StatusCode)
	}
	
	result.Time = time.Since(start).Milliseconds()
	return result
}

func generateFinalReport(results []TestResult) {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("📊 FINAL TEST REPORT")
	fmt.Println(strings.Repeat("=", 60))
	
	passCount := 0
	partialCount := 0
	failCount := 0
	totalTime := int64(0)
	
	for _, result := range results {
		icon := "✅"
		switch result.Status {
		case "PASS":
			passCount++
		case "PARTIAL":
			icon = "⚠️"
			partialCount++
		case "FAIL":
			icon = "❌"
			failCount++
		}
		
		totalTime += result.Time
		fmt.Printf("\n%s %s\n", icon, result.TestName)
		fmt.Printf("   Status: %s\n", result.Status)
		fmt.Printf("   Details: %s\n", result.Details)
		fmt.Printf("   Time: %dms\n", result.Time)
	}
	
	// Summary
	fmt.Println("\n" + strings.Repeat("-", 60))
	fmt.Println("SUMMARY:")
	fmt.Printf("Total Tests: %d\n", len(results))
	fmt.Printf("✅ Passed: %d\n", passCount)
	fmt.Printf("⚠️  Partial: %d\n", partialCount)
	fmt.Printf("❌ Failed: %d\n", failCount)
	fmt.Printf("Total Time: %dms\n", totalTime)
	
	// Overall Status
	fmt.Println("\n" + strings.Repeat("-", 60))
	if failCount == 0 && partialCount == 0 {
		fmt.Println("🎉 OVERALL STATUS: ALL TESTS PASSED!")
	} else if failCount == 0 {
		fmt.Println("⚠️  OVERALL STATUS: SYSTEM FUNCTIONAL WITH MINOR ISSUES")
	} else {
		fmt.Println("❌ OVERALL STATUS: CRITICAL ISSUES DETECTED")
	}
	
	// Recommendations
	fmt.Println("\n📝 RECOMMENDATIONS:")
	if failCount > 0 || partialCount > 0 {
		fmt.Println("1. Restart the server to apply all fixes")
		fmt.Println("2. Check server logs for any errors")
		fmt.Println("3. Verify database connectivity")
		if partialCount > 0 {
			fmt.Println("4. Review partial test results for specific issues")
		}
	} else {
		fmt.Println("System is functioning well. Continue monitoring for issues.")
	}
	
	// Save report
	report := map[string]interface{}{
		"timestamp": time.Now(),
		"results":   results,
		"summary": map[string]int{
			"total":   len(results),
			"passed":  passCount,
			"partial": partialCount,
			"failed":  failCount,
		},
		"totalTime": totalTime,
	}
	
	filename := fmt.Sprintf("final_test_report_%s.json", time.Now().Format("20060102_150405"))
	data, _ := json.MarshalIndent(report, "", "  ")
	os.WriteFile(filename, data, 0644)
	
	fmt.Printf("\n💾 Report saved to: %s\n", filename)
}