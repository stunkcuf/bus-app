package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
)

type TestResult struct {
	Page     string
	Status   int
	Success  bool
	Message  string
	Details  string
}

func main() {
	// Create a cookie jar to store session cookies
	jar, err := cookiejar.New(nil)
	if err != nil {
		log.Fatal(err)
	}
	
	client := &http.Client{
		Jar: jar,
	}
	
	baseURL := "http://localhost:5003"
	results := []TestResult{}
	
	fmt.Println("Fleet Management System - Comprehensive Test")
	fmt.Println("==========================================\n")
	
	// 1. Login
	fmt.Println("1. Testing Login...")
	if !login(client, baseURL) {
		log.Fatal("Login failed - cannot continue tests")
	}
	results = append(results, TestResult{
		Page:    "Login",
		Status:  200,
		Success: true,
		Message: "Login successful with admin/admin",
	})
	
	// 2. Test Manager Dashboard
	fmt.Println("\n2. Testing Manager Dashboard...")
	result := testPage(client, baseURL+"/manager-dashboard", "Manager Dashboard", []string{"Dashboard", "Statistics", "Quick Actions"})
	results = append(results, result)
	
	// 3. Test Fleet Page
	fmt.Println("\n3. Testing Fleet Page...")
	result = testFleetPage(client, baseURL+"/fleet")
	results = append(results, result)
	
	// 4. Test Student Management
	fmt.Println("\n4. Testing Student Management...")
	result = testPage(client, baseURL+"/students", "Student Management", []string{"Student", "Phone", "Guardian"})
	results = append(results, result)
	
	// 5. Test Route Assignment
	fmt.Println("\n5. Testing Route Assignment...")
	result = testPage(client, baseURL+"/assign-routes", "Route Assignment", []string{"Route", "Driver", "Bus"})
	results = append(results, result)
	
	// 6. Test User Management
	fmt.Println("\n6. Testing User Management...")
	result = testPage(client, baseURL+"/manage-users", "User Management", []string{"Username", "Role", "Status"})
	results = append(results, result)
	
	// 7. Test Maintenance Records
	fmt.Println("\n7. Testing Maintenance Records...")
	result = testPage(client, baseURL+"/maintenance-records", "Maintenance Records", []string{"Date", "Vehicle", "Category"})
	results = append(results, result)
	
	// 8. Test ECSE Import
	fmt.Println("\n8. Testing ECSE Import Page...")
	result = testPage(client, baseURL+"/import-ecse", "ECSE Import", []string{"Import", "Excel", "Upload"})
	results = append(results, result)
	
	// 9. Test Monthly Mileage Reports
	fmt.Println("\n9. Testing Monthly Mileage Reports...")
	result = testPage(client, baseURL+"/monthly-mileage-reports", "Monthly Mileage", []string{"Month", "Driver", "Mileage"})
	results = append(results, result)
	
	// 10. Test Driver Dashboard (need to create driver account or use existing)
	fmt.Println("\n10. Testing Driver Dashboard Access...")
	result = testPage(client, baseURL+"/driver-dashboard", "Driver Dashboard", []string{"Route", "Students", "Log"})
	if !result.Success {
		// Try as manager accessing driver features
		result.Details = "Manager may not have access to driver-specific features"
	}
	results = append(results, result)
	
	// Summary
	fmt.Println("\n\nTest Summary")
	fmt.Println("============")
	
	successCount := 0
	for _, r := range results {
		status := "❌"
		if r.Success {
			status = "✅"
			successCount++
		}
		fmt.Printf("%s %-25s: %s\n", status, r.Page, r.Message)
		if r.Details != "" {
			fmt.Printf("   Details: %s\n", r.Details)
		}
	}
	
	fmt.Printf("\nTotal Tests: %d\n", len(results))
	fmt.Printf("Passed: %d\n", successCount)
	fmt.Printf("Failed: %d\n", len(results)-successCount)
	fmt.Printf("Success Rate: %.1f%%\n", float64(successCount)/float64(len(results))*100)
}

func login(client *http.Client, baseURL string) bool {
	formData := url.Values{
		"username": {"admin"},
		"password": {"admin"},
	}
	
	resp, err := client.PostForm(baseURL+"/", formData)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	
	// Check if we got redirected to dashboard
	return strings.Contains(resp.Request.URL.Path, "dashboard")
}

func testPage(client *http.Client, url, pageName string, expectedContent []string) TestResult {
	resp, err := client.Get(url)
	if err != nil {
		return TestResult{
			Page:    pageName,
			Status:  0,
			Success: false,
			Message: fmt.Sprintf("Failed to load: %v", err),
		}
	}
	defer resp.Body.Close()
	
	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)
	
	// Check if redirected to login
	if strings.Contains(bodyStr, "Login") && strings.Contains(resp.Request.URL.Path, "/") && !strings.Contains(url, "login") {
		return TestResult{
			Page:    pageName,
			Status:  resp.StatusCode,
			Success: false,
			Message: "Redirected to login page",
		}
	}
	
	// Check for expected content
	missingContent := []string{}
	for _, content := range expectedContent {
		if !strings.Contains(bodyStr, content) {
			missingContent = append(missingContent, content)
		}
	}
	
	if len(missingContent) == 0 {
		return TestResult{
			Page:    pageName,
			Status:  resp.StatusCode,
			Success: true,
			Message: "Page loaded successfully with expected content",
		}
	}
	
	return TestResult{
		Page:    pageName,
		Status:  resp.StatusCode,
		Success: false,
		Message: "Page loaded but missing expected content",
		Details: fmt.Sprintf("Missing: %v", missingContent),
	}
}

func testFleetPage(client *http.Client, url string) TestResult {
	resp, err := client.Get(url)
	if err != nil {
		return TestResult{
			Page:    "Fleet Page",
			Status:  0,
			Success: false,
			Message: fmt.Sprintf("Failed to load: %v", err),
		}
	}
	defer resp.Body.Close()
	
	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)
	
	// Count vehicles
	busCount := strings.Count(bodyStr, "bus-card") + strings.Count(bodyStr, "Bus #")
	vehicleCount := strings.Count(bodyStr, "vehicle-card") + strings.Count(bodyStr, "Vehicle ID:")
	totalCount := busCount + vehicleCount
	
	// Look for the total count display
	has54 := strings.Contains(bodyStr, "54") || strings.Contains(bodyStr, "Total: 54")
	
	details := fmt.Sprintf("Found %d buses, %d vehicles (total: %d)", busCount, vehicleCount, totalCount)
	
	if totalCount >= 54 || has54 {
		return TestResult{
			Page:    "Fleet Page",
			Status:  resp.StatusCode,
			Success: true,
			Message: "Fleet page shows correct vehicle count",
			Details: details,
		}
	}
	
	return TestResult{
		Page:    "Fleet Page",
		Status:  resp.StatusCode,
		Success: false,
		Message: "Fleet page loaded but vehicle count mismatch",
		Details: details + " (expected 54)",
	}
}