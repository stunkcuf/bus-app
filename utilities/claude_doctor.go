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
}

func main() {
	fmt.Println("🏥 CLAUDE DOCTOR - Fleet Management System Health Check")
	fmt.Println("=" + strings.Repeat("=", 60))
	fmt.Println()
	
	results := []TestResult{}
	
	// 1. Database Connection
	fmt.Println("📊 Checking Database Connection...")
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgresql://postgres:jTKOWGEzlprRGbkPBCgGxsgnwyLeGoDL@shortline.proxy.rlwy.net:40148/railway"
	}
	
	db, err := sqlx.Open("postgres", dbURL)
	if err != nil {
		results = append(results, TestResult{"Database Connection", "❌ FAILED", fmt.Sprintf("Error: %v", err)})
	} else {
		defer db.Close()
		err = db.Ping()
		if err != nil {
			results = append(results, TestResult{"Database Connection", "❌ FAILED", fmt.Sprintf("Ping failed: %v", err)})
		} else {
			results = append(results, TestResult{"Database Connection", "✅ PASS", "Connected successfully"})
		}
	}
	
	// 2. Check Tables
	fmt.Println("📋 Checking Database Tables...")
	tables := []string{"users", "buses", "students", "routes", "route_assignments", 
		"driver_logs", "maintenance_records", "ecse_students", "monthly_mileage_reports"}
	
	for _, table := range tables {
		var count int
		err := db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&count)
		if err != nil {
			results = append(results, TestResult{fmt.Sprintf("Table: %s", table), "❌ FAILED", fmt.Sprintf("Error: %v", err)})
		} else {
			results = append(results, TestResult{fmt.Sprintf("Table: %s", table), "✅ PASS", fmt.Sprintf("%d records", count)})
		}
	}
	
	// 3. Server Health
	fmt.Println("🌐 Checking Server Health...")
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar, Timeout: 10 * time.Second}
	
	resp, err := client.Get("http://localhost:5003/health")
	if err != nil {
		results = append(results, TestResult{"Server Health", "❌ FAILED", fmt.Sprintf("Error: %v", err)})
	} else {
		defer resp.Body.Close()
		if resp.StatusCode == 200 {
			results = append(results, TestResult{"Server Health", "✅ PASS", "Server is healthy"})
		} else {
			results = append(results, TestResult{"Server Health", "⚠️ WARNING", fmt.Sprintf("Status: %d", resp.StatusCode)})
		}
	}
	
	// 4. Authentication Test
	fmt.Println("🔐 Testing Authentication...")
	loginData := url.Values{
		"username": {"admin"},
		"password": {"Headstart1"},
	}
	
	resp, err = client.PostForm("http://localhost:5003/", loginData)
	if err != nil {
		results = append(results, TestResult{"Admin Login", "❌ FAILED", fmt.Sprintf("Error: %v", err)})
	} else {
		defer resp.Body.Close()
		if resp.StatusCode == 303 || resp.StatusCode == 302 {
			results = append(results, TestResult{"Admin Login", "✅ PASS", "Login successful"})
		} else {
			results = append(results, TestResult{"Admin Login", "❌ FAILED", fmt.Sprintf("Status: %d", resp.StatusCode)})
		}
	}
	
	// 5. Test Key Pages
	fmt.Println("📄 Testing Key Pages...")
	pages := []struct {
		name string
		url  string
	}{
		{"Manager Dashboard", "/manager-dashboard"},
		{"Fleet", "/fleet"},
		{"Company Fleet", "/company-fleet"},
		{"Route Assignments", "/assign-routes"},
		{"ECSE Reports", "/view-ecse-reports"},
		{"Mileage Reports", "/monthly-mileage-reports"},
		{"Import ECSE", "/import-ecse"},
		{"Import Mileage", "/import-mileage"},
	}
	
	for _, page := range pages {
		resp, err := client.Get("http://localhost:5003" + page.url)
		if err != nil {
			results = append(results, TestResult{page.name, "❌ FAILED", fmt.Sprintf("Error: %v", err)})
			continue
		}
		
		body, _ := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		
		if resp.StatusCode == 200 {
			// Check for error messages in page
			bodyStr := string(body)
			if strings.Contains(bodyStr, "Error") && !strings.Contains(bodyStr, "ErrorDocument") {
				results = append(results, TestResult{page.name, "⚠️ WARNING", "Page loads but contains error"})
			} else if strings.Contains(bodyStr, page.name) || strings.Contains(bodyStr, "<!DOCTYPE html>") {
				results = append(results, TestResult{page.name, "✅ PASS", "Page loads correctly"})
			} else {
				results = append(results, TestResult{page.name, "⚠️ WARNING", "Page loads but content uncertain"})
			}
		} else if resp.StatusCode == 302 || resp.StatusCode == 303 {
			results = append(results, TestResult{page.name, "⚠️ WARNING", "Redirected (auth required?)"})
		} else {
			results = append(results, TestResult{page.name, "❌ FAILED", fmt.Sprintf("Status: %d", resp.StatusCode)})
		}
	}
	
	// 6. Check Data Display
	fmt.Println("📊 Checking Data Display...")
	
	// Check ECSE data
	resp, _ = client.Get("http://localhost:5003/view-ecse-reports")
	body, _ := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if strings.Contains(string(body), "No Students Found") {
		results = append(results, TestResult{"ECSE Data Display", "⚠️ WARNING", "No data displayed (template issue?)"})
	} else if strings.Contains(string(body), "student-row") {
		results = append(results, TestResult{"ECSE Data Display", "✅ PASS", "Data displayed correctly"})
	}
	
	// Check Mileage data
	resp, _ = client.Get("http://localhost:5003/monthly-mileage-reports")
	body, _ = ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if strings.Contains(string(body), "No Reports Found") {
		results = append(results, TestResult{"Mileage Data Display", "⚠️ WARNING", "No data displayed"})
	} else if strings.Contains(string(body), "Total Miles") {
		results = append(results, TestResult{"Mileage Data Display", "✅ PASS", "Data displayed correctly"})
	}
	
	// 7. Check Dark Theme
	fmt.Println("🎨 Checking Dark Theme...")
	resp, _ = client.Get("http://localhost:5003/fleet")
	body, _ = ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if strings.Contains(string(body), "background: #0f0c29") {
		results = append(results, TestResult{"Dark Theme", "✅ PASS", "Dark theme applied"})
	} else {
		results = append(results, TestResult{"Dark Theme", "⚠️ WARNING", "Dark theme might be missing"})
	}
	
	// Print Summary
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("📋 DIAGNOSTIC SUMMARY")
	fmt.Println(strings.Repeat("=", 60))
	
	passCount := 0
	failCount := 0
	warnCount := 0
	
	for _, result := range results {
		fmt.Printf("%-30s %s  %s\n", result.Name, result.Status, result.Details)
		if strings.Contains(result.Status, "PASS") {
			passCount++
		} else if strings.Contains(result.Status, "FAILED") {
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
	
	// Recommendations
	fmt.Println("\n💡 RECOMMENDATIONS:")
	if failCount > 0 {
		fmt.Println("- Fix failed tests immediately")
	}
	if warnCount > 0 {
		fmt.Println("- Investigate warning conditions")
	}
	
	// Check specific known issues
	var ecseCount int
	db.QueryRow("SELECT COUNT(*) FROM ecse_students").Scan(&ecseCount)
	if ecseCount > 0 && strings.Contains(results[len(results)-3].Status, "WARNING") {
		fmt.Println("- ECSE data exists but not displaying - may need server restart")
	}
}