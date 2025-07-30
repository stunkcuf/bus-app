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
)

func main() {
	baseURL := "http://localhost:5000"
	if len(os.Args) > 1 {
		baseURL = os.Args[1]
	}

	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}

	fmt.Println("Testing Fixed Endpoints...")
	fmt.Println("==========================")

	// First login as admin/manager
	fmt.Println("\n1. Logging in as admin...")
	loginData := url.Values{
		"username": {"admin"},
		"password": {"admin123"}, // Default password
	}
	
	resp, err := client.PostForm(baseURL+"/", loginData)
	if err != nil {
		fmt.Printf("❌ Login failed: %v\n", err)
		return
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	
	if resp.StatusCode != 200 && resp.StatusCode != 303 && resp.StatusCode != 302 {
		fmt.Printf("❌ Login failed with status %d\n", resp.StatusCode)
		fmt.Printf("Response: %s\n", string(body))
		return
	}
	fmt.Println("✅ Login successful")

	// Test the fixed endpoints
	endpoints := []struct {
		name        string
		url         string
		method      string
		expectJSON  bool
		checkFields []string
	}{
		{
			name:   "Users Page",
			url:    "/users",
			method: "GET",
		},
		{
			name:       "API Dashboard Stats",
			url:        "/api/dashboard/stats",
			method:     "GET",
			expectJSON: true,
			checkFields: []string{"activeBuses", "activeDrivers", "totalRoutes", "totalStudents"},
		},
		{
			name:       "API Fleet Status",
			url:        "/api/fleet-status",
			method:     "GET",
			expectJSON: true,
			checkFields: []string{"active", "maintenance", "inactive", "total"},
		},
		{
			name:   "Students Page (Manager Access)",
			url:    "/students",
			method: "GET",
		},
	}

	for _, endpoint := range endpoints {
		fmt.Printf("\n%s (%s %s)...\n", endpoint.name, endpoint.method, endpoint.url)
		
		req, _ := http.NewRequest(endpoint.method, baseURL+endpoint.url, nil)
		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("❌ Request failed: %v\n", err)
			continue
		}
		
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		
		fmt.Printf("Status: %d\n", resp.StatusCode)
		
		if resp.StatusCode == 200 {
			if endpoint.expectJSON {
				var result map[string]interface{}
				if err := json.Unmarshal(body, &result); err != nil {
					fmt.Printf("❌ Invalid JSON response: %v\n", err)
					fmt.Printf("Response: %s\n", string(body))
				} else {
					fmt.Println("✅ Valid JSON response")
					
					// Check for expected fields
					if data, ok := result["data"].(map[string]interface{}); ok {
						for _, field := range endpoint.checkFields {
							if _, exists := data[field]; exists {
								fmt.Printf("  ✓ Field '%s' present\n", field)
							} else {
								fmt.Printf("  ✗ Field '%s' missing\n", field)
							}
						}
					}
					
					// Print sample data
					fmt.Printf("Response data: %v\n", result["data"])
				}
			} else {
				// Check HTML response
				bodyStr := string(body)
				if strings.Contains(bodyStr, "<title>") && strings.Contains(bodyStr, "</title>") {
					fmt.Println("✅ Valid HTML response")
					
					// Extract title
					start := strings.Index(bodyStr, "<title>") + 7
					end := strings.Index(bodyStr, "</title>")
					if start > 6 && end > start {
						fmt.Printf("Page title: %s\n", bodyStr[start:end])
					}
				} else {
					fmt.Printf("⚠️  Response doesn't look like HTML\n")
				}
			}
		} else if resp.StatusCode == 401 {
			fmt.Printf("❌ Unauthorized (401) - Check authentication\n")
		} else if resp.StatusCode == 404 {
			fmt.Printf("❌ Not Found (404) - Endpoint not registered\n")
		} else {
			fmt.Printf("❌ Unexpected status code: %d\n", resp.StatusCode)
			if len(body) < 1000 {
				fmt.Printf("Response: %s\n", string(body))
			}
		}
	}

	// Now test driver dashboard with a driver account
	fmt.Println("\n\nTesting Driver Dashboard...")
	fmt.Println("===========================")
	
	// First logout
	client.Get(baseURL + "/logout")
	
	// Try to login as a driver (if exists)
	fmt.Println("\nTrying to login as driver1...")
	loginData = url.Values{
		"username": {"driver1"},
		"password": {"driver123"},
	}
	
	resp, err = client.PostForm(baseURL+"/", loginData)
	if err == nil && (resp.StatusCode == 200 || resp.StatusCode == 303 || resp.StatusCode == 302) {
		fmt.Println("✅ Driver login successful")
		
		// Test driver dashboard
		resp, err = client.Get(baseURL + "/driver-dashboard")
		if err == nil {
			fmt.Printf("Driver Dashboard Status: %d\n", resp.StatusCode)
			if resp.StatusCode == 200 {
				fmt.Println("✅ Driver dashboard accessible by driver")
			}
		}
		
		// Test students page as driver
		resp, err = client.Get(baseURL + "/students")
		if err == nil {
			fmt.Printf("Students Page Status (as driver): %d\n", resp.StatusCode)
			if resp.StatusCode == 200 {
				fmt.Println("✅ Students page accessible by driver")
			}
		}
	} else {
		fmt.Println("⚠️  No driver account available for testing")
		fmt.Println("   Create a driver account to test driver-specific pages")
	}
	
	fmt.Println("\n✅ Test completed!")
}