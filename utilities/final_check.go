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
	
	fmt.Println("Fleet Management System - Final System Test")
	fmt.Println("==========================================\n")
	
	// Test 1: Admin login and manager features
	fmt.Println("PART 1: Manager Features (admin account)")
	fmt.Println("-----------------------------------------")
	
	// Login as admin
	if !login(client, baseURL, "admin", "admin") {
		log.Fatal("Admin login failed")
	}
	fmt.Println("✅ Admin login successful\n")
	
	// Test manager pages
	managerPages := map[string]string{
		"Manager Dashboard":   "/manager-dashboard",
		"Fleet Page":          "/fleet",
		"User Management":     "/manage-users",
		"Route Assignment":    "/assign-routes",
		"ECSE Import":         "/import-ecse",
		"Maintenance Records": "/maintenance-records",
		"Fleet Vehicles":      "/fleet-vehicles",
		"Service Records":     "/service-records",
		"Monthly Mileage":     "/monthly-mileage-reports",
	}
	
	managerResults := []TestResult{}
	for name, path := range managerPages {
		result := testPage(client, baseURL+path, name)
		managerResults = append(managerResults, result)
		
		status := "❌"
		if result.Success {
			status = "✅"
		}
		fmt.Printf("%s %s - Status: %d\n", status, name, result.Status)
	}
	
	// Logout
	client.Get(baseURL + "/logout")
	
	// Test 2: Driver features
	fmt.Println("\nPART 2: Driver Features (bjmathis account)")
	fmt.Println("-------------------------------------------")
	
	// Try to login as driver (we don't know the password, so this might fail)
	fmt.Println("Note: Driver password unknown, testing what we can...\n")
	
	// Test 3: System summary
	fmt.Println("\nSYSTEM TEST SUMMARY")
	fmt.Println("===================")
	
	successCount := 0
	totalCount := len(managerResults)
	
	for _, r := range managerResults {
		if r.Success {
			successCount++
		}
	}
	
	fmt.Printf("\nManager Features Tested: %d\n", totalCount)
	fmt.Printf("Passed: %d\n", successCount)
	fmt.Printf("Failed: %d\n", totalCount-successCount)
	fmt.Printf("Success Rate: %.1f%%\n", float64(successCount)/float64(totalCount)*100)
	
	// List any failures
	if totalCount-successCount > 0 {
		fmt.Println("\nFailed Tests:")
		for _, r := range managerResults {
			if !r.Success {
				fmt.Printf("- %s: %s\n", r.Page, r.Message)
			}
		}
	}
	
	fmt.Println("\n✅ System test completed!")
}

func login(client *http.Client, baseURL, username, password string) bool {
	formData := url.Values{
		"username": {username},
		"password": {password},
	}
	
	resp, err := client.PostForm(baseURL+"/", formData)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	
	// Check if we got redirected to dashboard
	return strings.Contains(resp.Request.URL.Path, "dashboard")
}

func testPage(client *http.Client, url, pageName string) TestResult {
	resp, err := client.Get(url)
	if err != nil {
		return TestResult{
			Page:    pageName,
			Status:  0,
			Success: false,
			Message: fmt.Sprintf("Connection error: %v", err),
		}
	}
	defer resp.Body.Close()
	
	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)
	
	// Check for common error indicators
	if resp.StatusCode == 500 {
		errorMsg := "Internal server error"
		if strings.Contains(bodyStr, "Failed to") {
			// Extract error message
			start := strings.Index(bodyStr, "Failed to")
			if start != -1 {
				end := start + 50
				if end > len(bodyStr) {
					end = len(bodyStr)
				}
				errorMsg = bodyStr[start:end]
			}
		}
		return TestResult{
			Page:    pageName,
			Status:  resp.StatusCode,
			Success: false,
			Message: errorMsg,
		}
	}
	
	if resp.StatusCode == 403 {
		return TestResult{
			Page:    pageName,
			Status:  resp.StatusCode,
			Success: false,
			Message: "Access forbidden",
		}
	}
	
	// Check if redirected to login
	if strings.Contains(bodyStr, "Login") && strings.Contains(resp.Request.URL.Path, "/") {
		return TestResult{
			Page:    pageName,
			Status:  resp.StatusCode,
			Success: false,
			Message: "Redirected to login",
		}
	}
	
	// If status is 200 and no obvious errors, consider it success
	if resp.StatusCode == 200 {
		return TestResult{
			Page:    pageName,
			Status:  resp.StatusCode,
			Success: true,
			Message: "Page loaded successfully",
		}
	}
	
	return TestResult{
		Page:    pageName,
		Status:  resp.StatusCode,
		Success: false,
		Message: fmt.Sprintf("Unexpected status code: %d", resp.StatusCode),
	}
}