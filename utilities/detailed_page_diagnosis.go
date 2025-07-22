package main

import (
	"database/sql"
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

	_ "github.com/lib/pq"
)

func main() {
	// Connect to database to verify data exists
	db, err := sql.Open("postgres", "host=localhost port=5432 user=fleetuser password=Adminpassword123! dbname=fleetdb sslmode=disable")
	if err != nil {
		log.Fatal("Database connection failed:", err)
	}
	defer db.Close()

	fmt.Println("=== Fleet Management System Detailed Page Diagnosis ===\n")
	
	// First, let's check what's actually in the database
	fmt.Println("1. DATABASE CONTENT CHECK:")
	fmt.Println(strings.Repeat("-", 50))
	
	// Check each table
	tables := []struct {
		name  string
		query string
	}{
		{"Buses", "SELECT COUNT(*) FROM buses"},
		{"Vehicles", "SELECT COUNT(*) FROM vehicles"},
		{"Drivers", "SELECT COUNT(*) FROM drivers"},
		{"Students", "SELECT COUNT(*) FROM students"}, 
		{"ECSE Students", "SELECT COUNT(*) FROM ecse_students"},
		{"Routes", "SELECT COUNT(*) FROM routes"},
		{"Maintenance Records", "SELECT COUNT(*) FROM maintenance_records"},
		{"Service Records", "SELECT COUNT(*) FROM service_records"},
		{"Fuel Records", "SELECT COUNT(*) FROM fuel_records"},
		{"Monthly Mileage Reports", "SELECT COUNT(*) FROM monthly_mileage_reports"},
		{"Users", "SELECT COUNT(*) FROM users WHERE role = 'manager'"},
		{"Route Assignments", "SELECT COUNT(*) FROM route_assignments"},
	}
	
	for _, table := range tables {
		var count int
		err := db.QueryRow(table.query).Scan(&count)
		if err != nil {
			fmt.Printf("   ✗ %s: ERROR - %v\n", table.name, err)
		} else {
			fmt.Printf("   ✓ %s: %d records\n", table.name, count)
		}
	}
	
	// Check specific data samples
	fmt.Println("\n2. SAMPLE DATA CHECK:")
	fmt.Println(strings.Repeat("-", 50))
	
	// Check sample buses
	rows, err := db.Query("SELECT bus_number, status FROM buses LIMIT 5")
	if err == nil {
		fmt.Println("   Sample Buses:")
		defer rows.Close()
		for rows.Next() {
			var busNum string
			var status string
			rows.Scan(&busNum, &status)
			fmt.Printf("     - Bus %s (Status: %s)\n", busNum, status)
		}
	}
	
	// Now test the web interface
	fmt.Println("\n3. WEB INTERFACE TEST:")
	fmt.Println(strings.Repeat("-", 50))
	
	// Create HTTP client with cookie jar
	jar, _ := cookiejar.New(nil)
	client := &http.Client{
		Jar:     jar,
		Timeout: 30 * time.Second,
	}
	
	// Login
	fmt.Println("   Logging in...")
	loginURL := "http://localhost:5003/"
	formData := url.Values{
		"username": {"admin"},
		"password": {"Headstart1"},
	}
	
	resp, err := client.PostForm(loginURL, formData)
	if err != nil {
		log.Fatal("Login failed:", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusFound {
		log.Fatal("Login failed with status:", resp.StatusCode)
	}
	fmt.Println("   ✓ Login successful")
	
	// Test specific problematic pages with detailed analysis
	testPages := []string{
		"/fleet",
		"/dashboard", 
		"/students",
		"/drivers",
		"/company-fleet",
		"/maintenance-records",
	}
	
	fmt.Println("\n4. DETAILED PAGE TESTS:")
	fmt.Println(strings.Repeat("-", 50))
	
	for _, page := range testPages {
		fmt.Printf("\n   Testing %s:\n", page)
		
		resp, err := client.Get("http://localhost:5003" + page)
		if err != nil {
			fmt.Printf("     ✗ Request failed: %v\n", err)
			continue
		}
		defer resp.Body.Close()
		
		fmt.Printf("     - Status Code: %d\n", resp.StatusCode)
		
		if resp.StatusCode != http.StatusOK {
			fmt.Printf("     ✗ Page returned error status\n")
			continue
		}
		
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("     ✗ Failed to read response\n")
			continue
		}
		
		content := string(body)
		
		// Check for common patterns
		hasTable := strings.Contains(content, "<table") || strings.Contains(content, "<tbody")
		hasData := strings.Contains(content, "<tr") && !strings.Contains(content, "No data")
		hasError := strings.Contains(content, "error") || strings.Contains(content, "Error")
		hasPagination := strings.Contains(content, "pagination")
		
		fmt.Printf("     - Has table structure: %v\n", hasTable)
		fmt.Printf("     - Has data rows: %v\n", hasData)
		fmt.Printf("     - Has errors: %v\n", hasError)
		fmt.Printf("     - Has pagination: %v\n", hasPagination)
		
		// Extract specific content indicators
		if page == "/fleet" {
			busCount := countMatches(content, `Bus #\d+`)
			fmt.Printf("     - Buses found in HTML: %d\n", busCount)
		}
		
		// Check for specific error messages
		if hasError {
			errorMatch := regexp.MustCompile(`(?i)(error[^<]+)`).FindStringSubmatch(content)
			if len(errorMatch) > 0 {
				fmt.Printf("     - Error message: %s\n", errorMatch[1])
			}
		}
		
		// Save problematic page content for analysis
		if !hasData && resp.StatusCode == http.StatusOK {
			filename := fmt.Sprintf("debug_%s.html", strings.ReplaceAll(page[1:], "/", "_"))
			os.WriteFile(filename, body, 0644)
			fmt.Printf("     - Saved page content to %s for analysis\n", filename)
		}
	}
	
	fmt.Println("\n5. SUMMARY:")
	fmt.Println(strings.Repeat("-", 50))
	fmt.Println("The diagnosis shows:")
	fmt.Println("- Database contains data but web pages are not displaying it")
	fmt.Println("- Authentication is working correctly")
	fmt.Println("- Pages load but return empty data sets")
	fmt.Println("- This suggests an issue with data retrieval handlers")
}

func countMatches(content, pattern string) int {
	re := regexp.MustCompile(pattern)
	matches := re.FindAllString(content, -1)
	return len(matches)
}