package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"
	"time"
)

func main() {
	TestPages()
}

// TestPages simulates user interaction with the system
func TestPages() {
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

	// Get login page first to extract CSRF token
	fmt.Println("=== Getting Login Page ===")
	resp, err := client.Get(baseURL + "/")
	if err != nil {
		log.Printf("Failed to get login page: %v", err)
		return
	}
	defer resp.Body.Close()
	
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read login page: %v", err)
		return
	}
	
	// Extract CSRF token
	re := regexp.MustCompile(`<input type="hidden" name="csrf_token" value="([^"]*)"`)
	matches := re.FindSubmatch(body)
	var csrfToken string
	if len(matches) > 1 {
		csrfToken = string(matches[1])
		fmt.Printf("✓ Found CSRF token: %s\n", csrfToken)
	} else {
		fmt.Println("⚠️ No CSRF token found, continuing anyway")
	}
	
	// Test login
	fmt.Println("\n=== Testing Login ===")
	loginData := url.Values{
		"username":   {"admin"},
		"password":   {"Headstart1"},
		"csrf_token": {csrfToken},
	}
	
	resp, err = client.PostForm(baseURL+"/", loginData)
	if err != nil {
		log.Printf("Login error: %v", err)
		return
	}
	defer resp.Body.Close()
	
	if resp.StatusCode == http.StatusFound {
		location := resp.Header.Get("Location")
		fmt.Printf("✓ Login successful, redirecting to: %s\n", location)
	} else {
		body, _ := ioutil.ReadAll(resp.Body)
		log.Printf("Login failed with status: %d", resp.StatusCode)
		if strings.Contains(string(body), "Invalid username or password") {
			log.Printf("Invalid credentials")
		}
		ioutil.WriteFile("login_error.html", body, 0644)
		return
	}

	// Pages to test
	pages := []struct {
		name string
		url  string
		checkFor string
	}{
		{"Manager Dashboard", "/manager-dashboard", "Dashboard"},
		{"Fleet Overview", "/fleet", "Fleet Overview"},
		{"Assign Routes", "/assign-routes", "Route Assignments"},
		{"Fleet Vehicles", "/fleet-vehicles", "Fleet Vehicles"},
		{"Maintenance Records", "/maintenance-records", "Maintenance Records"},
		{"Service Records", "/service-records", "Service Records"},
		{"Monthly Mileage Reports", "/monthly-mileage-reports", "Mileage Reports"},
		{"ECSE Dashboard", "/ecse-dashboard", "ECSE Dashboard"},
		{"Students", "/students", "Students"},
		{"Manage Users", "/manage-users", "Manage Users"},
		{"Company Fleet", "/company-fleet", "Company Fleet"},
	}

	// Test each page
	fmt.Println("\n=== Testing Pages ===")
	errorCount := 0
	for _, page := range pages {
		fmt.Printf("\nTesting %s (%s)...\n", page.name, page.url)
		
		resp, err := client.Get(baseURL + page.url)
		if err != nil {
			fmt.Printf("  ❌ Error: %v\n", err)
			errorCount++
			continue
		}
		defer resp.Body.Close()
		
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("  ❌ Error reading body: %v\n", err)
			errorCount++
			continue
		}
		
		bodyStr := string(body)
		
		// Check status code
		if resp.StatusCode != http.StatusOK {
			fmt.Printf("  ❌ Status code: %d\n", resp.StatusCode)
			errorCount++
			filename := fmt.Sprintf("error_%s_%d.html", 
				strings.ReplaceAll(page.name, " ", "_"), 
				time.Now().Unix())
			ioutil.WriteFile(filename, body, 0644)
			fmt.Printf("  Saved error page to: %s\n", filename)
			continue
		}
		
		// Check for expected content
		if !strings.Contains(bodyStr, page.checkFor) {
			fmt.Printf("  ⚠️ Expected content '%s' not found\n", page.checkFor)
		}
		
		// Check for errors
		if strings.Contains(strings.ToLower(bodyStr), "error:") || 
		   strings.Contains(strings.ToLower(bodyStr), "failed to") {
			fmt.Printf("  ❌ Page contains error messages\n")
			errorCount++
		}
		
		// Check for data
		hasData := false
		dataCount := 0
		
		// Count table rows (excluding header)
		tableRows := strings.Count(bodyStr, "<tr>") - strings.Count(bodyStr, "<thead>")
		if tableRows > 1 {
			hasData = true
			dataCount = tableRows - 1 // Subtract header row
		}
		
		// Count cards
		cardCount := strings.Count(bodyStr, "class=\"card")
		if cardCount > 0 {
			hasData = true
			if dataCount == 0 {
				dataCount = cardCount
			}
		}
		
		// Special check for routes page
		if page.url == "/assign-routes" {
			if strings.Contains(bodyStr, "No Routes Defined") {
				fmt.Printf("  ⚠️ No routes defined message displayed\n")
				hasData = false
			}
			if strings.Contains(bodyStr, "No Driver Assignments") {
				fmt.Printf("  ⚠️ No driver assignments message displayed\n")
			}
		}
		
		if hasData {
			fmt.Printf("  ✓ OK - Has data (found %d items)\n", dataCount)
		} else {
			fmt.Printf("  ⚠️ OK - No data displayed\n")
		}
		
		// Small delay between requests
		time.Sleep(500 * time.Millisecond)
	}

	// Test API endpoints
	fmt.Println("\n=== Testing API Endpoints ===")
	apiEndpoints := []struct {
		name string
		url  string
	}{
		{"Routes API", "/api/routes"},
		{"Buses API", "/api/buses"},
		{"Drivers API", "/api/drivers"},
		{"Students API", "/api/students"},
		{"Fleet Vehicles API", "/api/fleet-vehicles"},
		{"Route Assignments API", "/api/route-assignments"},
	}

	for _, endpoint := range apiEndpoints {
		fmt.Printf("\nTesting %s...\n", endpoint.name)
		
		resp, err := client.Get(baseURL + endpoint.url)
		if err != nil {
			fmt.Printf("  ❌ Error: %v\n", err)
			errorCount++
			continue
		}
		defer resp.Body.Close()
		
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("  ❌ Error reading body: %v\n", err)
			errorCount++
			continue
		}
		
		if resp.StatusCode == http.StatusOK {
			bodyStr := strings.TrimSpace(string(body))
			if strings.HasPrefix(bodyStr, "[") && strings.HasSuffix(bodyStr, "]") {
				// Count items in array
				itemCount := strings.Count(bodyStr, "{")
				if itemCount > 0 {
					fmt.Printf("  ✓ OK - Returns %d items\n", itemCount)
				} else {
					fmt.Printf("  ⚠️ OK - Empty array\n")
				}
			} else if strings.HasPrefix(bodyStr, "{") {
				fmt.Printf("  ✓ OK - Returns object\n")
			} else {
				fmt.Printf("  ❌ Invalid JSON response\n")
				errorCount++
			}
		} else {
			fmt.Printf("  ❌ Status code: %d\n", resp.StatusCode)
			errorCount++
		}
		
		time.Sleep(200 * time.Millisecond)
	}

	fmt.Printf("\n=== Test Complete ===\n")
	fmt.Printf("Total errors found: %d\n", errorCount)
	if errorCount == 0 {
		fmt.Println("✅ All tests passed!")
	} else {
		fmt.Printf("❌ %d tests failed\n", errorCount)
	}
}