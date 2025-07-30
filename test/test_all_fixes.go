package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"
)

func main() {
	TestAllFixes()
}

// TestAllFixes simulates page visits and checks for errors
func TestAllFixes() {
	baseURL := "http://localhost:8080"
	
	// Create HTTP client with cookie jar for session management
	jar, err := cookiejar.New(nil)
	if err != nil {
		log.Fatal("Failed to create cookie jar:", err)
	}
	
	client := &http.Client{
		Jar:     jar,
		Timeout: 10 * time.Second,
	}

	// Test login first
	fmt.Println("=== Testing Login ===")
	loginData := url.Values{
		"username": {"admin"},
		"password": {"admin123"},
	}
	
	resp, err := client.PostForm(baseURL+"/", loginData)
	if err != nil {
		log.Printf("Login error: %v", err)
		return
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusFound {
		log.Printf("Login failed with status: %d", resp.StatusCode)
		return
	}
	
	fmt.Println("✓ Login successful")

	// Pages to test
	pages := []struct {
		name string
		url  string
	}{
		{"Manager Dashboard", "/manager-dashboard"},
		{"Fleet Overview", "/fleet"},
		{"Assign Routes", "/assign-routes"},
		{"Fleet Vehicles", "/fleet-vehicles"},
		{"Maintenance Records", "/maintenance-records"},
		{"Service Records", "/service-records"},
		{"Monthly Mileage Reports", "/monthly-mileage-reports"},
		{"ECSE Dashboard", "/ecse-dashboard"},
		{"Students", "/students"},
		{"Manage Users", "/manage-users"},
		{"Company Fleet", "/company-fleet"},
	}

	// Test each page
	fmt.Println("\n=== Testing Pages ===")
	for _, page := range pages {
		fmt.Printf("Testing %s...", page.name)
		
		resp, err := client.Get(baseURL + page.url)
		if err != nil {
			fmt.Printf(" ❌ Error: %v\n", err)
			continue
		}
		defer resp.Body.Close()
		
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf(" ❌ Error reading body: %v\n", err)
			continue
		}
		
		// Check for common error indicators
		bodyStr := string(body)
		hasError := false
		errorMsg := ""
		
		if resp.StatusCode != http.StatusOK {
			hasError = true
			errorMsg = fmt.Sprintf("Status code: %d", resp.StatusCode)
		} else if strings.Contains(bodyStr, "Error:") || strings.Contains(bodyStr, "error:") {
			hasError = true
			errorMsg = "Page contains error message"
		} else if strings.Contains(bodyStr, "No Routes Defined") && page.url == "/assign-routes" {
			// Check if it's actually empty or just the default message
			if !strings.Contains(bodyStr, "<tr>") || strings.Count(bodyStr, "<tr>") < 2 {
				hasError = true
				errorMsg = "No routes displayed (empty table)"
			}
		}
		
		if hasError {
			fmt.Printf(" ❌ %s\n", errorMsg)
			
			// Save error page for debugging
			filename := fmt.Sprintf("error_%s_%d.html", 
				strings.ReplaceAll(page.name, " ", "_"), 
				time.Now().Unix())
			ioutil.WriteFile(filename, body, 0644)
			fmt.Printf("   Saved error page to: %s\n", filename)
		} else {
			// Check for data presence
			dataIndicators := []string{
				"<tr>", // Table rows
				"<td>", // Table data
				"card", // Card elements
			}
			
			hasData := false
			for _, indicator := range dataIndicators {
				if strings.Count(bodyStr, indicator) > 5 { // More than just headers
					hasData = true
					break
				}
			}
			
			if hasData {
				fmt.Printf(" ✓ OK (has data)\n")
			} else {
				fmt.Printf(" ⚠️  OK (no data displayed)\n")
			}
		}
		
		// Small delay between requests
		time.Sleep(100 * time.Millisecond)
	}

	// Test API endpoints
	fmt.Println("\n=== Testing API Endpoints ===")
	apiEndpoints := []string{
		"/api/routes",
		"/api/buses",
		"/api/drivers",
		"/api/students",
		"/api/fleet-vehicles",
	}

	for _, endpoint := range apiEndpoints {
		fmt.Printf("Testing %s...", endpoint)
		
		resp, err := client.Get(baseURL + endpoint)
		if err != nil {
			fmt.Printf(" ❌ Error: %v\n", err)
			continue
		}
		defer resp.Body.Close()
		
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf(" ❌ Error reading body: %v\n", err)
			continue
		}
		
		if resp.StatusCode == http.StatusOK {
			// Check if response is JSON array
			bodyStr := strings.TrimSpace(string(body))
			if strings.HasPrefix(bodyStr, "[") && strings.HasSuffix(bodyStr, "]") {
				if len(bodyStr) > 2 { // Not empty array
					fmt.Printf(" ✓ OK (has data)\n")
				} else {
					fmt.Printf(" ⚠️  OK (empty array)\n")
				}
			} else {
				fmt.Printf(" ❌ Invalid JSON response\n")
			}
		} else {
			fmt.Printf(" ❌ Status code: %d\n", resp.StatusCode)
		}
		
		time.Sleep(100 * time.Millisecond)
	}

	fmt.Println("\n=== Test Complete ===")
}