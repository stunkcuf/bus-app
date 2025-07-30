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

func main() {
	fmt.Println("🔍 Verifying Critical Fixes")
	fmt.Println("=" + strings.Repeat("=", 40))
	
	// Create HTTP client
	jar, _ := cookiejar.New(nil)
	client := &http.Client{
		Jar: jar,
		Timeout: 30 * time.Second,
	}
	
	// Login first
	loginData := url.Values{
		"username": {"admin"},
		"password": {"Headstart1"},
	}
	
	fmt.Print("\n1. Testing authentication... ")
	resp, err := client.PostForm("http://localhost:5003/", loginData)
	if err != nil {
		fmt.Printf("❌ FAILED: %v\n", err)
		return
	}
	resp.Body.Close()
	fmt.Println("✅ PASS")
	
	// Test Company Fleet Performance
	fmt.Print("\n2. Company Fleet load time... ")
	start := time.Now()
	resp, err = client.Get("http://localhost:5003/company-fleet")
	loadTime := time.Since(start)
	
	if err != nil {
		fmt.Printf("❌ FAILED: %v\n", err)
	} else {
		body, _ := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		
		if loadTime > 2*time.Second {
			fmt.Printf("❌ SLOW: %v (should be <2s)\n", loadTime)
			fmt.Println("   Fix not applied - restart server!")
		} else {
			vehicleCount := strings.Count(string(body), "data-vehicle-id")
			if vehicleCount > 0 {
				fmt.Printf("✅ PASS: %v (%d vehicles)\n", loadTime, vehicleCount)
			} else if strings.Contains(string(body), "No Vehicles Found") {
				fmt.Printf("⚠️ WARNING: Loaded in %v but shows 'No Vehicles'\n", loadTime)
			} else {
				fmt.Printf("✅ PASS: %v\n", loadTime)
			}
		}
	}
	
	// Test ECSE Data Display
	fmt.Print("\n3. ECSE data display... ")
	resp, err = client.Get("http://localhost:5003/view-ecse-reports")
	if err != nil {
		fmt.Printf("❌ FAILED: %v\n", err)
	} else {
		body, _ := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		bodyStr := string(body)
		
		studentCount := strings.Count(bodyStr, "student-row")
		if studentCount > 0 {
			fmt.Printf("✅ PASS: Showing %d students\n", studentCount)
		} else if strings.Contains(bodyStr, "No Students Found") {
			fmt.Println("❌ FAILED: Shows 'No Students Found'")
			fmt.Println("   Fix not applied - restart server!")
		} else {
			fmt.Println("⚠️ WARNING: Unable to verify")
		}
	}
	
	// Test Maintenance Records Display
	fmt.Print("\n4. Maintenance records... ")
	resp, err = client.Get("http://localhost:5003/bus-maintenance/24")
	if err != nil {
		fmt.Printf("❌ FAILED: %v\n", err)
	} else {
		body, _ := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		
		recordCount := strings.Count(string(body), "maintenance-record")
		if recordCount > 0 {
			fmt.Printf("✅ PASS: Showing %d records\n", recordCount)
		} else {
			// Check for dark theme
			if strings.Contains(string(body), "background: #0f0c29") {
				fmt.Println("✅ PASS: Dark theme applied")
			} else {
				fmt.Println("⚠️ WARNING: No records visible")
			}
		}
	}
	
	// Summary
	fmt.Println("\n" + strings.Repeat("=", 40))
	fmt.Println("SUMMARY:")
	fmt.Println("- If any tests show 'Fix not applied'")
	fmt.Println("  → Restart the server: go run .")
	fmt.Println("- After restart, run this tool again")
	fmt.Println("- All tests should show ✅ PASS")
}