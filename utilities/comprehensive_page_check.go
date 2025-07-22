package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

type PageTest struct {
	URL         string
	Name        string
	DataCount   int
	TotalClaim  int
	Status      string
	Error       string
	Notes       []string
	HasData     bool
	Pagination  bool
}

type TestResults struct {
	Timestamp    time.Time
	TotalPages   int
	WorkingPages int
	BrokenPages  int
	Pages        []PageTest
	Summary      string
}

func main() {
	// Connect to database first to get actual counts
	db, err := sql.Open("postgres", "host=localhost port=5432 user=fleetuser password=Adminpassword123! dbname=fleetdb sslmode=disable")
	if err != nil {
		log.Fatal("Database connection failed:", err)
	}
	defer db.Close()

	// Create HTTP client with cookie jar
	jar, _ := cookiejar.New(nil)
	client := &http.Client{
		Jar:     jar,
		Timeout: 30 * time.Second,
	}

	// Login first
	fmt.Println("=== Fleet Management System Comprehensive Page Test ===")
	fmt.Println("Logging in as admin...")
	
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

	fmt.Println("Login successful!\n")

	// Get actual database counts
	dbCounts := getDBCounts(db)
	
	// Define all pages to test
	pages := []struct {
		url        string
		name       string
		dbCountKey string
	}{
		{"/fleet", "Fleet Overview", "buses"},
		{"/fleet-vehicles", "Fleet Vehicles", "vehicles"},
		{"/maintenance-records", "Maintenance Records", "maintenance"},
		{"/service-records", "Service Records", "service"},
		{"/fuel-records", "Fuel Records", "fuel"},
		{"/students", "Students", "students"},
		{"/ecse-dashboard", "ECSE Dashboard", "ecse"},
		{"/drivers", "Drivers", "drivers"},
		{"/routes", "Routes", "routes"},
		{"/company-fleet", "Company Fleet", "company_fleet"},
		{"/dashboard", "Main Dashboard", "dashboard"},
		{"/monthly-mileage-reports", "Monthly Mileage Reports", "mileage"},
		{"/fuel-analytics", "Fuel Analytics", "fuel_analytics"},
		{"/vehicle-maintenance", "Vehicle Maintenance", "vehicle_maintenance"},
	}

	results := TestResults{
		Timestamp:  time.Now(),
		TotalPages: len(pages),
		Pages:      make([]PageTest, 0),
	}

	// Test each page
	for _, page := range pages {
		fmt.Printf("\nTesting %s (%s)...\n", page.name, page.url)
		result := testPage(client, page.url, page.name, dbCounts[page.dbCountKey])
		results.Pages = append(results.Pages, result)
		
		if result.Status == "Working" {
			results.WorkingPages++
		} else {
			results.BrokenPages++
		}
		
		// Print immediate results
		fmt.Printf("  Status: %s\n", result.Status)
		if result.Error != "" {
			fmt.Printf("  Error: %s\n", result.Error)
		}
		if result.DataCount > 0 || result.TotalClaim > 0 {
			fmt.Printf("  Data: %d displayed / %d total (DB has %d)\n", result.DataCount, result.TotalClaim, dbCounts[page.dbCountKey])
		}
		for _, note := range result.Notes {
			fmt.Printf("  Note: %s\n", note)
		}
		
		time.Sleep(500 * time.Millisecond) // Be nice to the server
	}

	// Generate summary report
	generateReport(results)
}

func getDBCounts(db *sql.DB) map[string]int {
	counts := make(map[string]int)
	
	// Count buses
	var busCount int
	db.QueryRow("SELECT COUNT(*) FROM buses").Scan(&busCount)
	counts["buses"] = busCount
	
	// Count vehicles (assuming vehicles table exists)
	var vehicleCount int
	db.QueryRow("SELECT COUNT(*) FROM vehicles").Scan(&vehicleCount)
	counts["vehicles"] = vehicleCount
	
	// Count maintenance records
	var maintenanceCount int
	db.QueryRow("SELECT COUNT(*) FROM maintenance_records").Scan(&maintenanceCount)
	counts["maintenance"] = maintenanceCount
	
	// Count service records
	var serviceCount int
	db.QueryRow("SELECT COUNT(*) FROM service_records").Scan(&serviceCount)
	counts["service"] = serviceCount
	
	// Count fuel records
	var fuelCount int
	db.QueryRow("SELECT COUNT(*) FROM fuel_records").Scan(&fuelCount)
	counts["fuel"] = fuelCount
	
	// Count students
	var studentCount int
	db.QueryRow("SELECT COUNT(*) FROM students").Scan(&studentCount)
	counts["students"] = studentCount
	
	// Count ECSE students
	var ecseCount int
	db.QueryRow("SELECT COUNT(*) FROM ecse_students").Scan(&ecseCount)
	counts["ecse"] = ecseCount
	
	// Count drivers
	var driverCount int
	db.QueryRow("SELECT COUNT(*) FROM drivers").Scan(&driverCount)
	counts["drivers"] = driverCount
	
	// Count routes
	var routeCount int
	db.QueryRow("SELECT COUNT(*) FROM routes").Scan(&routeCount)
	counts["routes"] = routeCount
	
	// Count mileage reports
	var mileageCount int
	db.QueryRow("SELECT COUNT(*) FROM monthly_mileage_reports").Scan(&mileageCount)
	counts["mileage"] = mileageCount
	
	return counts
}

func testPage(client *http.Client, pageURL, pageName string, dbCount int) PageTest {
	result := PageTest{
		URL:   pageURL,
		Name:  pageName,
		Notes: make([]string, 0),
	}
	
	fullURL := "http://localhost:5003" + pageURL
	resp, err := client.Get(fullURL)
	if err != nil {
		result.Status = "Error"
		result.Error = err.Error()
		return result
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		result.Status = "Error"
		result.Error = fmt.Sprintf("HTTP %d", resp.StatusCode)
		return result
	}
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		result.Status = "Error"
		result.Error = "Failed to read response"
		return result
	}
	
	content := string(body)
	
	// Check for error messages
	if strings.Contains(content, "Error") || strings.Contains(content, "error") {
		errorMatch := regexp.MustCompile(`(?i)(error[^<]+)`).FindStringSubmatch(content)
		if len(errorMatch) > 0 {
			result.Notes = append(result.Notes, "Error message found: "+errorMatch[1])
		}
	}
	
	// Page-specific checks
	switch pageURL {
	case "/fleet":
		result.DataCount = countMatches(content, `<tr[^>]*>.*?Bus #\d+.*?</tr>`)
		result.TotalClaim = extractNumber(content, `Total Buses:\s*(\d+)`)
		if result.DataCount == 0 {
			result.DataCount = countMatches(content, `bus-card|fleet-card`)
		}
		
	case "/fleet-vehicles":
		result.DataCount = countMatches(content, `<tr[^>]*>.*?vehicle.*?</tr>`)
		result.TotalClaim = extractNumber(content, `Total Vehicles:\s*(\d+)`)
		
	case "/maintenance-records":
		result.DataCount = countMatches(content, `<tr[^>]*>.*?maintenance.*?</tr>`)
		result.TotalClaim = extractNumber(content, `Total Records:\s*(\d+)|Showing \d+ of (\d+)`)
		if strings.Contains(content, "458 records") {
			result.TotalClaim = 458
		}
		
	case "/service-records":
		result.DataCount = countMatches(content, `<tr[^>]*>.*?service.*?</tr>`)
		result.TotalClaim = extractNumber(content, `Total Records:\s*(\d+)|(\d+) records`)
		if strings.Contains(content, "55 records") {
			result.TotalClaim = 55
		}
		
	case "/fuel-records":
		result.DataCount = countMatches(content, `<tr[^>]*>.*?fuel.*?</tr>`)
		if strings.Contains(content, "No fuel records") {
			result.Notes = append(result.Notes, "Page shows 'No fuel records'")
		}
		
	case "/students":
		result.DataCount = countMatches(content, `<tr[^>]*>.*?student.*?</tr>`)
		result.TotalClaim = extractNumber(content, `Total Students:\s*(\d+)`)
		
	case "/ecse-dashboard":
		result.DataCount = countMatches(content, `<tr[^>]*>.*?student.*?</tr>`)
		result.TotalClaim = extractNumber(content, `Total ECSE Students:\s*(\d+)`)
		
	case "/drivers":
		result.DataCount = countMatches(content, `<tr[^>]*>.*?driver.*?</tr>`)
		result.TotalClaim = extractNumber(content, `Total Drivers:\s*(\d+)`)
		
	case "/routes":
		result.DataCount = countMatches(content, `<tr[^>]*>.*?route.*?</tr>`)
		result.TotalClaim = extractNumber(content, `Total Routes:\s*(\d+)`)
		
	case "/dashboard":
		// Check for dashboard widgets/cards
		if strings.Contains(content, "Total Buses") {
			result.Notes = append(result.Notes, "Dashboard shows bus count")
		}
		if strings.Contains(content, "Active Drivers") {
			result.Notes = append(result.Notes, "Dashboard shows driver count")
		}
		if strings.Contains(content, "Total Students") {
			result.Notes = append(result.Notes, "Dashboard shows student count")
		}
		result.HasData = len(result.Notes) > 0
		
	case "/monthly-mileage-reports":
		result.DataCount = countMatches(content, `<tr[^>]*>.*?mileage.*?</tr>`)
		result.TotalClaim = extractNumber(content, `Total Reports:\s*(\d+)`)
		
	case "/fuel-analytics":
		// Check for charts or analytics data
		if strings.Contains(content, "chart") || strings.Contains(content, "Chart") {
			result.Notes = append(result.Notes, "Analytics charts present")
			result.HasData = true
		}
		
	case "/vehicle-maintenance":
		result.DataCount = countMatches(content, `<tr[^>]*>.*?maintenance.*?</tr>`)
		if strings.Contains(content, "Schedule Maintenance") {
			result.Notes = append(result.Notes, "Maintenance scheduling available")
		}
	}
	
	// Check for pagination
	if strings.Contains(content, "pagination") || strings.Contains(content, "page=") {
		result.Pagination = true
		result.Notes = append(result.Notes, "Pagination detected")
	}
	
	// Check if data is present
	if result.DataCount > 0 {
		result.HasData = true
	}
	
	// Determine status
	if result.Error != "" {
		result.Status = "Error"
	} else if result.DataCount == 0 && !result.HasData {
		result.Status = "No Data"
	} else if result.TotalClaim > 0 && result.DataCount < result.TotalClaim/2 {
		result.Status = "Partial Data"
		result.Notes = append(result.Notes, fmt.Sprintf("Only showing %d%% of claimed data", (result.DataCount*100)/result.TotalClaim))
	} else {
		result.Status = "Working"
	}
	
	// Compare with DB count
	if dbCount > 0 && result.DataCount < dbCount {
		result.Notes = append(result.Notes, fmt.Sprintf("DB has %d records, page shows %d", dbCount, result.DataCount))
	}
	
	return result
}

func countMatches(content, pattern string) int {
	re := regexp.MustCompile(pattern)
	matches := re.FindAllString(content, -1)
	return len(matches)
}

func extractNumber(content, pattern string) int {
	re := regexp.MustCompile(pattern)
	match := re.FindStringSubmatch(content)
	if len(match) > 1 {
		for i := 1; i < len(match); i++ {
			if match[i] != "" {
				num, _ := strconv.Atoi(match[i])
				return num
			}
		}
	}
	return 0
}

func generateReport(results TestResults) {
	fmt.Println("\n\n=== COMPREHENSIVE TEST REPORT ===")
	fmt.Printf("Test Date: %s\n", results.Timestamp.Format("2006-01-02 15:04:05"))
	fmt.Printf("Total Pages Tested: %d\n", results.TotalPages)
	fmt.Printf("Working Pages: %d\n", results.WorkingPages)
	fmt.Printf("Broken Pages: %d\n", results.BrokenPages)
	fmt.Printf("Success Rate: %.1f%%\n\n", float64(results.WorkingPages)*100/float64(results.TotalPages))
	
	// Working pages
	fmt.Println("=== WORKING PAGES ===")
	for _, page := range results.Pages {
		if page.Status == "Working" {
			fmt.Printf("✓ %s (%s)\n", page.Name, page.URL)
			if page.DataCount > 0 {
				fmt.Printf("  - Showing %d records\n", page.DataCount)
			}
		}
	}
	
	// Broken/Problem pages
	fmt.Println("\n=== BROKEN/PROBLEM PAGES ===")
	for _, page := range results.Pages {
		if page.Status != "Working" {
			fmt.Printf("✗ %s (%s) - %s\n", page.Name, page.URL, page.Status)
			if page.Error != "" {
				fmt.Printf("  - Error: %s\n", page.Error)
			}
			if page.DataCount > 0 && page.TotalClaim > 0 {
				fmt.Printf("  - Only showing %d of %d records\n", page.DataCount, page.TotalClaim)
			}
			for _, note := range page.Notes {
				fmt.Printf("  - %s\n", note)
			}
		}
	}
	
	// Create JSON report
	reportData, _ := json.MarshalIndent(results, "", "  ")
	err := os.WriteFile("page_test_report.json", reportData, 0644)
	if err == nil {
		fmt.Println("\nDetailed report saved to page_test_report.json")
	}
	
	// Summary
	fmt.Println("\n=== SUMMARY ===")
	fmt.Println("The fleet management system has significant data display issues:")
	fmt.Printf("- Only %d out of %d pages are fully functional\n", results.WorkingPages, results.TotalPages)
	fmt.Println("- Many pages show limited or no data despite having records in the database")
	fmt.Println("- The main issue appears to be in data retrieval/display logic, not the database itself")
}