package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"
	"time"
)

func main() {
	baseURL := "http://localhost:8080"
	
	// Create HTTP client with cookie jar
	jar, _ := cookiejar.New(nil)
	client := &http.Client{
		Jar:     jar,
		Timeout: 10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Allow redirects but limit to 10
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}
	
	fmt.Println("=== Fleet Management System Test ===\n")
	
	// Step 1: Get login page and CSRF token
	fmt.Println("1. Getting login page...")
	resp, err := client.Get(baseURL + "/")
	if err != nil {
		fmt.Printf("   ❌ Error: %v\n", err)
		return
	}
	
	body, _ := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	
	// Extract CSRF token
	re := regexp.MustCompile(`<input type="hidden" name="csrf_token" value="([^"]*)"`)
	matches := re.FindSubmatch(body)
	csrfToken := ""
	if len(matches) > 1 {
		csrfToken = string(matches[1])
	}
	fmt.Printf("   ✓ Got login page (CSRF: %s)\n", csrfToken)
	
	// Step 2: Login
	fmt.Println("\n2. Logging in as admin...")
	loginData := url.Values{
		"username":   {"admin"},
		"password":   {"Test123456!"},
		"csrf_token": {csrfToken},
	}
	
	resp, err = client.PostForm(baseURL+"/", loginData)
	if err != nil {
		fmt.Printf("   ❌ Error: %v\n", err)
		return
	}
	
	if resp.StatusCode == http.StatusFound || resp.StatusCode == http.StatusSeeOther {
		location := resp.Header.Get("Location")
		fmt.Printf("   ✓ Login successful! Redirecting to: %s\n", location)
		resp.Body.Close()
		
		// Follow redirect
		resp, err = client.Get(baseURL + location)
		if err != nil {
			fmt.Printf("   ❌ Error following redirect: %v\n", err)
			return
		}
	} else {
		body, _ := ioutil.ReadAll(resp.Body)
		fmt.Printf("   ❌ Login failed (Status: %d)\n", resp.StatusCode)
		if strings.Contains(string(body), "Invalid") {
			fmt.Println("   Error: Invalid credentials")
		}
		resp.Body.Close()
		return
	}
	
	body, _ = ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	
	// Check what page we're on
	if strings.Contains(string(body), "Manager Dashboard") {
		fmt.Println("   ✓ Successfully reached Manager Dashboard")
	} else if strings.Contains(string(body), "Driver Dashboard") {
		fmt.Println("   ✓ Successfully reached Driver Dashboard")
	}
	
	// Step 3: Test key pages
	fmt.Println("\n3. Testing key pages...")
	
	pages := []struct {
		name     string
		url      string
		checkFor []string
	}{
		{
			"Assign Routes",
			"/assign-routes",
			[]string{"Route Assignments", "Available Routes", "Driver Assignments"},
		},
		{
			"Fleet Overview",
			"/fleet",
			[]string{"Fleet Overview", "Buses", "Vehicles"},
		},
		{
			"Company Fleet",
			"/company-fleet",
			[]string{"Company Fleet", "Vehicle Management"},
		},
		{
			"Manage Users",
			"/manage-users",
			[]string{"Manage Users", "User Management"},
		},
		{
			"ECSE Dashboard",
			"/ecse-dashboard",
			[]string{"ECSE", "Special Education"},
		},
		{
			"Maintenance Records",
			"/maintenance-records",
			[]string{"Maintenance", "Service History"},
		},
	}
	
	for _, page := range pages {
		fmt.Printf("\n   Testing %s (%s)...\n", page.name, page.url)
		
		resp, err := client.Get(baseURL + page.url)
		if err != nil {
			fmt.Printf("     ❌ Error: %v\n", err)
			continue
		}
		
		body, _ := ioutil.ReadAll(resp.Body)
		bodyStr := string(body)
		resp.Body.Close()
		
		if resp.StatusCode != http.StatusOK {
			fmt.Printf("     ❌ Status: %d\n", resp.StatusCode)
			// Save error page
			filename := fmt.Sprintf("error_%s.html", strings.ReplaceAll(page.name, " ", "_"))
			ioutil.WriteFile(filename, body, 0644)
			continue
		}
		
		fmt.Printf("     ✓ Page loaded (Status: 200)\n")
		
		// Check for expected content
		foundContent := false
		for _, check := range page.checkFor {
			if strings.Contains(bodyStr, check) {
				fmt.Printf("     ✓ Found: %s\n", check)
				foundContent = true
				break
			}
		}
		
		if !foundContent {
			fmt.Printf("     ⚠️ Expected content not found\n")
		}
		
		// Check for data
		tableRows := strings.Count(bodyStr, "<tr>")
		if tableRows > 1 {
			fmt.Printf("     ✓ Has data: %d table rows\n", tableRows-1)
		} else {
			// Check for specific "no data" messages
			if strings.Contains(bodyStr, "No Routes Defined") {
				fmt.Printf("     ⚠️ No routes defined\n")
			} else if strings.Contains(bodyStr, "No Driver Assignments") {
				fmt.Printf("     ⚠️ No driver assignments\n")
			} else if strings.Contains(bodyStr, "No vehicles found") {
				fmt.Printf("     ⚠️ No vehicles found\n")
			} else if strings.Contains(bodyStr, "No data") || strings.Contains(bodyStr, "No records") {
				fmt.Printf("     ⚠️ No data displayed\n")
			} else {
				fmt.Printf("     ℹ️ No table data found\n")
			}
		}
		
		// Check for errors
		if strings.Contains(strings.ToLower(bodyStr), "error:") || 
		   strings.Contains(strings.ToLower(bodyStr), "failed to") {
			fmt.Printf("     ❌ Page contains error messages\n")
			// Extract error message
			lines := strings.Split(bodyStr, "\n")
			for _, line := range lines {
				if strings.Contains(strings.ToLower(line), "error:") || 
				   strings.Contains(strings.ToLower(line), "failed to") {
					fmt.Printf("        Error: %s\n", strings.TrimSpace(line))
					break
				}
			}
		}
		
		time.Sleep(500 * time.Millisecond)
	}
	
	// Step 4: Test API endpoints
	fmt.Println("\n\n4. Testing API endpoints...")
	
	apis := []struct {
		name string
		url  string
	}{
		{"Routes", "/api/routes"},
		{"Buses", "/api/buses"},
		{"Drivers", "/api/drivers"},
		{"Fleet Vehicles", "/api/fleet-vehicles"},
		{"Route Assignments", "/api/route-assignments"},
	}
	
	for _, api := range apis {
		fmt.Printf("\n   %s API...\n", api.name)
		
		resp, err := client.Get(baseURL + api.url)
		if err != nil {
			fmt.Printf("     ❌ Error: %v\n", err)
			continue
		}
		
		body, _ := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		
		if resp.StatusCode != http.StatusOK {
			fmt.Printf("     ❌ Status: %d\n", resp.StatusCode)
			continue
		}
		
		bodyStr := strings.TrimSpace(string(body))
		if strings.HasPrefix(bodyStr, "[") {
			itemCount := strings.Count(bodyStr, "{")
			if itemCount > 0 {
				fmt.Printf("     ✓ Returns %d items\n", itemCount)
			} else {
				fmt.Printf("     ⚠️ Returns empty array\n")
			}
		} else if strings.HasPrefix(bodyStr, "{") {
			fmt.Printf("     ✓ Returns object\n")
		} else {
			fmt.Printf("     ❌ Invalid JSON response\n")
		}
	}
	
	fmt.Println("\n\n=== Test Complete ===")
}