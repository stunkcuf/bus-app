package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
)

func main() {
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}

	fmt.Println("Testing Mileage Reports Page")
	fmt.Println("===========================")

	// Login as manager
	fmt.Println("\n1. Logging in as admin...")
	loginData := url.Values{
		"username": {"admin"},
		"password": {"admin123"},
	}
	
	resp, err := client.PostForm("http://localhost:5003/", loginData)
	if err != nil {
		fmt.Printf("❌ Login error: %v\n", err)
		return
	}
	
	body, _ := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	
	if strings.Contains(string(body), "Manager Dashboard") {
		fmt.Println("✅ Successfully logged in as manager")
	} else {
		fmt.Println("❌ Login failed")
		return
	}

	// Access mileage reports page
	fmt.Println("\n2. Accessing /view-mileage-reports...")
	resp, err = client.Get("http://localhost:5003/view-mileage-reports")
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		return
	}
	
	body, _ = ioutil.ReadAll(resp.Body)
	bodyStr := string(body)
	resp.Body.Close()
	
	fmt.Printf("Status: %d\n", resp.StatusCode)
	
	// Check what we got
	if strings.Contains(bodyStr, "Monthly Mileage Reports") {
		fmt.Println("✅ Mileage reports page loaded successfully!")
		
		// Check for data
		if strings.Contains(bodyStr, "No Reports Found") {
			fmt.Println("   ⚠️  No mileage reports found")
		} else if strings.Contains(bodyStr, "tbody") && strings.Contains(bodyStr, "Total Miles") {
			fmt.Println("   ✅ Mileage data is displaying")
			
			// Count reports
			reportCount := strings.Count(bodyStr, "<tr>") - 1 // Subtract header row
			if reportCount > 0 {
				fmt.Printf("   ✅ Found %d mileage reports\n", reportCount)
			}
		}
		
		// Check styling
		if strings.Contains(bodyStr, "background: #0f0c29") {
			fmt.Println("   ✅ Dark theme is applied")
		}
		
		// Check summary cards
		if strings.Contains(bodyStr, "summary-card") {
			fmt.Println("   ✅ Summary statistics are present")
		}
	} else if strings.Contains(bodyStr, "Error") || strings.Contains(bodyStr, "error") {
		fmt.Println("❌ Error loading page")
		// Extract error
		if idx := strings.Index(bodyStr, "Error:"); idx != -1 {
			end := strings.Index(bodyStr[idx:], "</")
			if end > 0 {
				fmt.Printf("   %s\n", bodyStr[idx:idx+end])
			}
		}
	} else {
		fmt.Println("❌ Unexpected response")
		// Show preview
		if len(bodyStr) > 300 {
			fmt.Printf("Preview: %s...\n", bodyStr[:300])
		}
	}
}