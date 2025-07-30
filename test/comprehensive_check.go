package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"
)

type TestResult struct {
	Page    string
	Status  string
	Message string
	Data    int
}

func main() {
	fmt.Println("=== Fleet Management System Comprehensive Test ===")
	fmt.Printf("Time: %s\n\n", time.Now().Format("2006-01-02 15:04:05"))
	
	results := []TestResult{}
	
	// Create HTTP client
	jar, _ := cookiejar.New(nil)
	client := &http.Client{
		Jar:     jar,
		Timeout: 10 * time.Second,
	}
	
	// 1. Test Login
	fmt.Println("1. Testing Login...")
	loginData := url.Values{
		"username": {"admin"},
		"password": {"Test123456!"},
	}
	
	resp, err := client.PostForm("http://localhost:8080/", loginData)
	if err != nil {
		results = append(results, TestResult{
			Page:    "Login",
			Status:  "❌ FAIL",
			Message: fmt.Sprintf("Error: %v", err),
		})
	} else {
		finalURL := resp.Request.URL.String()
		resp.Body.Close()
		
		if strings.Contains(finalURL, "dashboard") {
			results = append(results, TestResult{
				Page:    "Login",
				Status:  "✓ PASS",
				Message: "Successfully logged in as admin",
			})
			fmt.Println("   ✓ Login successful")
		} else {
			results = append(results, TestResult{
				Page:    "Login",
				Status:  "❌ FAIL",
				Message: "Login failed - still on login page",
			})
			fmt.Println("   ❌ Login failed")
			return
		}
	}
	
	// 2. Test Pages
	fmt.Println("\n2. Testing All Pages...")
	
	pages := []struct {
		name string
		url  string
	}{
		{"Manager Dashboard", "/manager-dashboard"},
		{"Fleet Overview", "/fleet"},
		{"Assign Routes", "/assign-routes"},
		{"Company Fleet", "/company-fleet"},
		{"Fleet Vehicles", "/fleet-vehicles"},
		{"Manage Users", "/manage-users"},
		{"ECSE Dashboard", "/ecse-dashboard"},
		{"Students", "/students"},
		{"Maintenance Records", "/maintenance-records"},
		{"Service Records", "/service-records"},
		{"Monthly Mileage Reports", "/monthly-mileage-reports"},
		{"User Activity Report", "/user-activity-report"},
		{"Reports", "/reports"},
		{"Approve Users", "/approve-users"},
		{"Add Fleet Vehicle", "/add-fleet-vehicle"},
		{"Add Student", "/add-student"},
		{"System Metrics", "/system-metrics"},
	}
	
	for i, page := range pages {
		fmt.Printf("\n   [%d/%d] Testing %s...\n", i+1, len(pages), page.name)
		
		resp, err := client.Get("http://localhost:8080" + page.url)
		if err != nil {
			results = append(results, TestResult{
				Page:    page.name,
				Status:  "❌ ERROR",
				Message: fmt.Sprintf("Request failed: %v", err),
			})
			continue
		}
		
		body, _ := ioutil.ReadAll(resp.Body)
		bodyStr := string(body)
		resp.Body.Close()
		
		result := TestResult{
			Page: page.name,
		}
		
		// Check status code
		if resp.StatusCode != http.StatusOK {
			result.Status = "❌ FAIL"
			result.Message = fmt.Sprintf("Status %d", resp.StatusCode)
			
			// Save error page
			filename := fmt.Sprintf("error_%s_%d.html", 
				strings.ReplaceAll(strings.ToLower(page.name), " ", "_"), 
				time.Now().Unix())
			ioutil.WriteFile(filename, body, 0644)
			fmt.Printf("      ❌ Status: %d (saved to %s)\n", resp.StatusCode, filename)
		} else {
			// Check for errors in content
			if strings.Contains(strings.ToLower(bodyStr), "error:") || 
			   strings.Contains(strings.ToLower(bodyStr), "failed to") {
				result.Status = "⚠️ WARN"
				
				// Extract error message
				lines := strings.Split(bodyStr, "\n")
				for _, line := range lines {
					if strings.Contains(strings.ToLower(line), "error:") || 
					   strings.Contains(strings.ToLower(line), "failed to") {
						result.Message = strings.TrimSpace(line)
						break
					}
				}
				fmt.Printf("      ⚠️ Page contains errors: %s\n", result.Message)
			} else {
				result.Status = "✓ PASS"
				
				// Count data
				tableRows := strings.Count(bodyStr, "<tr>") - strings.Count(bodyStr, "<thead>")
				if tableRows > 1 {
					result.Data = tableRows - 1
					result.Message = fmt.Sprintf("%d records", result.Data)
					fmt.Printf("      ✓ OK - %d records found\n", result.Data)
				} else {
					// Check for specific no data messages
					if strings.Contains(bodyStr, "No Routes Defined") ||
					   strings.Contains(bodyStr, "No Driver Assignments") ||
					   strings.Contains(bodyStr, "No vehicles found") ||
					   strings.Contains(bodyStr, "No data") ||
					   strings.Contains(bodyStr, "No records") {
						result.Message = "No data"
						fmt.Printf("      ✓ OK - No data\n")
					} else {
						result.Message = "Page loaded"
						fmt.Printf("      ✓ OK - Page loaded\n")
					}
				}
			}
		}
		
		results = append(results, result)
		time.Sleep(200 * time.Millisecond)
	}
	
	// 3. Test API Endpoints
	fmt.Println("\n\n3. Testing API Endpoints...")
	
	apis := []struct {
		name string
		url  string
	}{
		{"Routes API", "/api/routes"},
		{"Buses API", "/api/buses"},
		{"Drivers API", "/api/drivers"},
		{"Students API", "/api/students"},
		{"Fleet Vehicles API", "/api/fleet-vehicles"},
		{"Route Assignments API", "/api/route-assignments"},
		{"ECSE Students API", "/api/ecse-students"},
		{"Maintenance Records API", "/api/maintenance-records"},
	}
	
	for _, api := range apis {
		fmt.Printf("\n   Testing %s...\n", api.name)
		
		resp, err := client.Get("http://localhost:8080" + api.url)
		if err != nil {
			results = append(results, TestResult{
				Page:    api.name,
				Status:  "❌ ERROR",
				Message: fmt.Sprintf("Request failed: %v", err),
			})
			continue
		}
		
		body, _ := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		
		result := TestResult{
			Page: api.name,
		}
		
		if resp.StatusCode != http.StatusOK {
			result.Status = "❌ FAIL"
			result.Message = fmt.Sprintf("Status %d", resp.StatusCode)
			fmt.Printf("      ❌ Status: %d\n", resp.StatusCode)
		} else {
			bodyStr := strings.TrimSpace(string(body))
			if strings.HasPrefix(bodyStr, "[") {
				itemCount := strings.Count(bodyStr, "{")
				result.Status = "✓ PASS"
				result.Data = itemCount
				result.Message = fmt.Sprintf("%d items", itemCount)
				fmt.Printf("      ✓ OK - %d items\n", itemCount)
			} else if strings.HasPrefix(bodyStr, "{") {
				result.Status = "✓ PASS"
				result.Message = "Returns object"
				fmt.Printf("      ✓ OK - Returns object\n")
			} else {
				result.Status = "❌ FAIL"
				result.Message = "Invalid JSON"
				fmt.Printf("      ❌ Invalid JSON response\n")
			}
		}
		
		results = append(results, result)
	}
	
	// 4. Summary Report
	fmt.Println("\n\n=== TEST SUMMARY ===")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("%-30s %-12s %-40s\n", "Page/Endpoint", "Status", "Details")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	
	passCount := 0
	failCount := 0
	warnCount := 0
	
	for _, result := range results {
		fmt.Printf("%-30s %-12s %-40s\n", result.Page, result.Status, result.Message)
		
		if strings.Contains(result.Status, "PASS") {
			passCount++
		} else if strings.Contains(result.Status, "FAIL") || strings.Contains(result.Status, "ERROR") {
			failCount++
		} else if strings.Contains(result.Status, "WARN") {
			warnCount++
		}
	}
	
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("\nTotal Tests: %d\n", len(results))
	fmt.Printf("✓ Passed: %d\n", passCount)
	fmt.Printf("⚠️ Warnings: %d\n", warnCount)
	fmt.Printf("❌ Failed: %d\n", failCount)
	
	successRate := float64(passCount) / float64(len(results)) * 100
	fmt.Printf("\nSuccess Rate: %.1f%%\n", successRate)
	
	if failCount == 0 {
		fmt.Println("\n🎉 All tests passed!")
	} else {
		fmt.Printf("\n⚠️ %d tests failed - review error pages for details\n", failCount)
	}
	
	// Save summary
	summaryFile := fmt.Sprintf("test_summary_%s.txt", time.Now().Format("20060102_150405"))
	summary := fmt.Sprintf("Fleet Management System Test Summary\n")
	summary += fmt.Sprintf("Date: %s\n\n", time.Now().Format("2006-01-02 15:04:05"))
	summary += fmt.Sprintf("Total Tests: %d\n", len(results))
	summary += fmt.Sprintf("Passed: %d\n", passCount)
	summary += fmt.Sprintf("Warnings: %d\n", warnCount)
	summary += fmt.Sprintf("Failed: %d\n", failCount)
	summary += fmt.Sprintf("Success Rate: %.1f%%\n\n", successRate)
	summary += "Details:\n"
	for _, result := range results {
		summary += fmt.Sprintf("%-30s %s %s\n", result.Page, result.Status, result.Message)
	}
	
	ioutil.WriteFile(summaryFile, []byte(summary), 0644)
	fmt.Printf("\nTest summary saved to: %s\n", summaryFile)
}