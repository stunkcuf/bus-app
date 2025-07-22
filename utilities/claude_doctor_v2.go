package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"
	"time"

	_ "github.com/lib/pq"
	"github.com/jmoiron/sqlx"
)

type TestResult struct {
	Name    string
	Status  string
	Details string
	Time    time.Duration
}

func main() {
	fmt.Println("🏥 CLAUDE DOCTOR v2 - Comprehensive System Check")
	fmt.Println("=" + strings.Repeat("=", 60))
	fmt.Println()
	
	results := []TestResult{}
	
	// 1. Database Connection
	fmt.Println("📊 Phase 1: Database Connectivity...")
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgresql://postgres:jTKOWGEzlprRGbkPBCgGxsgnwyLeGoDL@shortline.proxy.rlwy.net:40148/railway"
	}
	
	start := time.Now()
	db, err := sqlx.Open("postgres", dbURL)
	if err != nil {
		results = append(results, TestResult{"Database Connection", "❌ FAILED", fmt.Sprintf("Error: %v", err), time.Since(start)})
		printResults(results)
		return
	}
	defer db.Close()
	
	err = db.Ping()
	connTime := time.Since(start)
	if err != nil {
		results = append(results, TestResult{"Database Connection", "❌ FAILED", fmt.Sprintf("Ping failed: %v", err), connTime})
	} else {
		results = append(results, TestResult{"Database Connection", "✅ PASS", fmt.Sprintf("Connected in %v", connTime), connTime})
	}
	
	// 2. Table Data Verification
	fmt.Println("\n📋 Phase 2: Data Integrity Check...")
	tables := map[string]struct{
		Name string
		MinExpected int
	}{
		"users": {"Users", 5},
		"buses": {"Buses", 10},
		"vehicles": {"Vehicles", 40},
		"students": {"Students", 50},
		"routes": {"Routes", 5},
		"route_assignments": {"Route Assignments", 0},
		"driver_logs": {"Driver Logs", 0},
		"maintenance_records": {"Maintenance Records", 400},
		"ecse_students": {"ECSE Students", 800},
		"monthly_mileage_reports": {"Mileage Reports", 0},
	}
	
	totalRecords := 0
	for table, info := range tables {
		var count int
		start = time.Now()
		err := db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&count)
		queryTime := time.Since(start)
		
		if err != nil {
			results = append(results, TestResult{info.Name, "❌ FAILED", fmt.Sprintf("Error: %v", err), queryTime})
		} else {
			totalRecords += count
			status := "✅ PASS"
			details := fmt.Sprintf("%d records", count)
			if count < info.MinExpected {
				status = "⚠️ WARNING"
				details = fmt.Sprintf("%d records (expected %d+)", count, info.MinExpected)
			}
			results = append(results, TestResult{info.Name, status, details, queryTime})
		}
	}
	
	// 3. Authentication Test
	fmt.Println("\n🔐 Phase 3: Authentication System...")
	jar, _ := cookiejar.New(nil)
	client := &http.Client{
		Jar: jar,
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // Don't follow redirects
		},
	}
	
	// Test login
	loginData := url.Values{
		"username": {"admin"},
		"password": {"Headstart1"},
	}
	
	start = time.Now()
	resp, err := client.PostForm("http://localhost:5003/", loginData)
	loginTime := time.Since(start)
	
	if err != nil {
		results = append(results, TestResult{"Admin Login", "❌ FAILED", fmt.Sprintf("Error: %v", err), loginTime})
	} else {
		defer resp.Body.Close()
		if resp.StatusCode == 303 || resp.StatusCode == 302 {
			results = append(results, TestResult{"Admin Login", "✅ PASS", fmt.Sprintf("Authenticated in %v", loginTime), loginTime})
		} else {
			body, _ := ioutil.ReadAll(resp.Body)
			if strings.Contains(string(body), "Invalid username or password") {
				results = append(results, TestResult{"Admin Login", "❌ FAILED", "Invalid credentials", loginTime})
			} else {
				results = append(results, TestResult{"Admin Login", "❌ FAILED", fmt.Sprintf("Status: %d", resp.StatusCode), loginTime})
			}
		}
	}
	
	// Re-enable redirect following for page tests
	client.CheckRedirect = nil
	
	// 4. Critical Page Load Tests
	fmt.Println("\n📄 Phase 4: Page Load Performance...")
	pages := []struct {
		name string
		url  string
		maxTime time.Duration
		checkFor string
	}{
		{"Manager Dashboard", "/manager-dashboard", 2 * time.Second, "Manager Dashboard"},
		{"Fleet", "/fleet", 2 * time.Second, "Fleet Management"},
		{"Company Fleet", "/company-fleet", 2 * time.Second, "Company Fleet"},
		{"ECSE Reports", "/view-ecse-reports", 2 * time.Second, "ECSE Student Reports"},
		{"Mileage Reports", "/monthly-mileage-reports", 2 * time.Second, "Monthly Mileage Reports"},
		{"Route Assignments", "/assign-routes", 2 * time.Second, "Route Assignments"},
	}
	
	for _, page := range pages {
		start = time.Now()
		resp, err := client.Get("http://localhost:5003" + page.url)
		loadTime := time.Since(start)
		
		if err != nil {
			if strings.Contains(err.Error(), "timeout") {
				results = append(results, TestResult{page.name, "❌ TIMEOUT", fmt.Sprintf("Exceeded %v", page.maxTime), loadTime})
			} else {
				results = append(results, TestResult{page.name, "❌ FAILED", fmt.Sprintf("Error: %v", err), loadTime})
			}
			continue
		}
		
		body, _ := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		bodyStr := string(body)
		
		if resp.StatusCode == 200 {
			status := "✅ PASS"
			details := fmt.Sprintf("Loaded in %v", loadTime)
			
			// Performance check
			if loadTime > page.maxTime {
				status = "⚠️ SLOW"
				details = fmt.Sprintf("Loaded in %v (max %v)", loadTime, page.maxTime)
			}
			
			// Content check
			if page.checkFor != "" && !strings.Contains(bodyStr, page.checkFor) {
				status = "⚠️ WARNING"
				details = fmt.Sprintf("Missing expected content (%v)", loadTime)
			}
			
			results = append(results, TestResult{page.name, status, details, loadTime})
		} else {
			results = append(results, TestResult{page.name, "❌ FAILED", fmt.Sprintf("Status %d", resp.StatusCode), loadTime})
		}
	}
	
	// 5. Data Display Verification
	fmt.Println("\n📊 Phase 5: Data Display Verification...")
	
	// Check ECSE data display
	resp, _ = client.Get("http://localhost:5003/view-ecse-reports")
	body, _ := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if strings.Contains(string(body), "No Students Found") && results[9].Status == "✅ PASS" {
		results = append(results, TestResult{"ECSE Data Display", "❌ FAILED", "Shows 'No Students' despite 825 in DB", 0})
	} else if strings.Contains(string(body), "student-row") {
		count := strings.Count(string(body), "student-row")
		results = append(results, TestResult{"ECSE Data Display", "✅ PASS", fmt.Sprintf("Displaying %d students", count), 0})
	} else {
		results = append(results, TestResult{"ECSE Data Display", "⚠️ WARNING", "Unable to verify display", 0})
	}
	
	// Check Company Fleet vehicles
	resp, _ = client.Get("http://localhost:5003/company-fleet")
	body, _ = ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if strings.Contains(string(body), "No Vehicles Found") && results[3].Status == "✅ PASS" {
		results = append(results, TestResult{"Fleet Data Display", "❌ FAILED", "Shows 'No Vehicles' despite 44 in DB", 0})
	} else if strings.Contains(string(body), "vehicle-") || strings.Contains(string(body), "tr data-vehicle-id") {
		results = append(results, TestResult{"Fleet Data Display", "✅ PASS", "Vehicles displayed", 0})
	} else {
		results = append(results, TestResult{"Fleet Data Display", "⚠️ WARNING", "Unable to verify display", 0})
	}
	
	// Print final results
	printResults(results)
	
	// Performance Summary
	fmt.Println("\n⚡ PERFORMANCE SUMMARY:")
	totalTime := time.Duration(0)
	slowPages := 0
	for _, r := range results {
		if r.Time > 0 {
			totalTime += r.Time
			if r.Time > 2*time.Second {
				slowPages++
			}
		}
	}
	fmt.Printf("Total test time: %v\n", totalTime)
	fmt.Printf("Slow pages (>2s): %d\n", slowPages)
	fmt.Printf("Total DB records: %d\n", totalRecords)
}

func printResults(results []TestResult) {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("📋 DIAGNOSTIC RESULTS")
	fmt.Println(strings.Repeat("=", 60))
	
	passCount := 0
	failCount := 0
	warnCount := 0
	
	for _, result := range results {
		fmt.Printf("%-25s %s  %s\n", result.Name, result.Status, result.Details)
		if strings.Contains(result.Status, "PASS") {
			passCount++
		} else if strings.Contains(result.Status, "FAILED") || strings.Contains(result.Status, "TIMEOUT") {
			failCount++
		} else {
			warnCount++
		}
	}
	
	fmt.Println("\n" + strings.Repeat("-", 60))
	fmt.Printf("Total Tests: %d\n", len(results))
	fmt.Printf("✅ Passed: %d\n", passCount)
	fmt.Printf("⚠️ Warnings: %d\n", warnCount)
	fmt.Printf("❌ Failed: %d\n", failCount)
	
	// Overall Health Score
	score := (passCount * 100) / len(results)
	fmt.Printf("\n🏥 SYSTEM HEALTH SCORE: %d%%\n", score)
	
	if score >= 90 {
		fmt.Println("💚 System is healthy!")
	} else if score >= 70 {
		fmt.Println("💛 System needs attention")
	} else {
		fmt.Println("❤️ System has critical issues")
	}
	
	// Specific Recommendations
	fmt.Println("\n💡 RECOMMENDATIONS:")
	if failCount > 0 {
		fmt.Println("- Fix all failed tests immediately")
	}
	if warnCount > 0 {
		fmt.Println("- Investigate warning conditions")
	}
	
	for _, r := range results {
		if strings.Contains(r.Name, "Company Fleet") && r.Time > 5*time.Second {
			fmt.Println("- Company Fleet page is very slow - restart server to apply optimization")
		}
		if strings.Contains(r.Name, "ECSE Data Display") && strings.Contains(r.Status, "FAILED") {
			fmt.Println("- ECSE data exists but not displaying - restart server to apply fix")
		}
	}
}