package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"
)

func main() {
	// Create HTTP client with cookie jar
	jar, _ := cookiejar.New(nil)
	client := &http.Client{
		Jar: jar,
	}

	fmt.Println("=== Fleet Management System Manual Page Check ===\n")

	// Login
	fmt.Println("1. Logging in as admin...")
	loginURL := "http://localhost:5003/"
	formData := url.Values{
		"username": {"admin"},
		"password": {"Headstart1"},
	}

	resp, err := client.PostForm(loginURL, formData)
	if err != nil {
		log.Fatal("Login failed:", err)
	}
	resp.Body.Close()

	fmt.Println("   ✓ Login successful\n")

	// Test each page manually
	pages := []struct {
		url         string
		name        string
		searchFor   string
		countRegex  string
	}{
		{"/fleet", "Fleet Overview", "Bus #", `Bus #\d+`},
		{"/fleet-vehicles", "Fleet Vehicles", "vehicle", `<tr[^>]*>.*?</tr>`},
		{"/maintenance-records", "Maintenance Records", "maintenance", `<tr[^>]*>.*?</tr>`},
		{"/service-records", "Service Records", "service", `<tr[^>]*>.*?</tr>`},
		{"/fuel-records", "Fuel Records", "fuel", `<tr[^>]*>.*?</tr>`},
		{"/students", "Students", "student", `<tr[^>]*>.*?</tr>`},
		{"/ecse-dashboard", "ECSE Dashboard", "student", `<tr[^>]*>.*?</tr>`},
		{"/monthly-mileage-reports", "Monthly Mileage Reports", "mileage", `<tr[^>]*>.*?</tr>`},
		{"/dashboard", "Main Dashboard", "Total", `Total`},
	}

	fmt.Println("2. Testing Individual Pages:")
	fmt.Println(strings.Repeat("-", 80))
	fmt.Printf("%-30s %-10s %-15s %-25s\n", "Page", "Status", "Data Found", "Notes")
	fmt.Println(strings.Repeat("-", 80))

	for _, page := range pages {
		resp, err := client.Get("http://localhost:5003" + page.url)
		status := "OK"
		dataFound := "No"
		notes := ""

		if err != nil {
			status = "ERROR"
			notes = err.Error()
		} else {
			if resp.StatusCode != 200 {
				status = fmt.Sprintf("HTTP %d", resp.StatusCode)
			}

			body, _ := io.ReadAll(resp.Body)
			content := string(body)
			resp.Body.Close()

			// Check for data
			if strings.Contains(content, page.searchFor) {
				// Count occurrences
				re := regexp.MustCompile(page.countRegex)
				matches := re.FindAllString(content, -1)
				if len(matches) > 0 {
					dataFound = fmt.Sprintf("Yes (%d)", len(matches))
				}
			}

			// Check for specific indicators
			if strings.Contains(content, "No data") || strings.Contains(content, "No records") {
				notes = "Empty message shown"
			}
			if strings.Contains(content, "Error") && !strings.Contains(content, "errorMessage") {
				notes = "Error on page"
			}
			if strings.Contains(content, "pagination") {
				if notes != "" {
					notes += ", "
				}
				notes += "Has pagination"
			}

			// Special checks
			if page.url == "/maintenance-records" && strings.Contains(content, "458 records") {
				if notes != "" {
					notes += ", "
				}
				notes += "Claims 458 records"
			}
			if page.url == "/service-records" && strings.Contains(content, "55 records") {
				if notes != "" {
					notes += ", "
				}
				notes += "Claims 55 records"
			}
		}

		fmt.Printf("%-30s %-10s %-15s %-25s\n", page.name, status, dataFound, notes)
	}

	fmt.Println(strings.Repeat("-", 80))
	
	// Test a specific API endpoint
	fmt.Println("\n3. Testing API Endpoints:")
	fmt.Println(strings.Repeat("-", 50))
	
	apiEndpoints := []string{
		"/api/dashboard/fleet-status",
		"/api/fuel/summary",
	}
	
	for _, endpoint := range apiEndpoints {
		resp, err := client.Get("http://localhost:5003" + endpoint)
		if err != nil {
			fmt.Printf("   %s: ERROR - %v\n", endpoint, err)
			continue
		}
		
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		
		fmt.Printf("   %s: HTTP %d\n", endpoint, resp.StatusCode)
		if resp.StatusCode == 200 && len(body) > 0 {
			// Try to show first 100 chars of response
			preview := string(body)
			if len(preview) > 100 {
				preview = preview[:100] + "..."
			}
			fmt.Printf("     Response: %s\n", preview)
		}
	}
	
	fmt.Println("\n4. SUMMARY:")
	fmt.Println(strings.Repeat("-", 50))
	fmt.Println("Check the table above to see which pages are working vs broken.")
	fmt.Println("Pages showing 'No' in Data Found column need investigation.")
}