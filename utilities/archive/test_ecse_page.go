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

	fmt.Println("Testing ECSE Reports Page")
	fmt.Println("========================")

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
	resp.Body.Close()

	// Access ECSE reports page
	fmt.Println("\n2. Accessing /view-ecse-reports...")
	resp, err = client.Get("http://localhost:5003/view-ecse-reports")
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		return
	}
	
	body, _ := ioutil.ReadAll(resp.Body)
	bodyStr := string(body)
	resp.Body.Close()
	
	fmt.Printf("Status: %d\n", resp.StatusCode)
	
	// Check what we got
	if strings.Contains(bodyStr, "ECSE Reports") || strings.Contains(bodyStr, "ECSE Students") {
		fmt.Println("✅ ECSE reports page loaded successfully!")
		
		// Check for data
		if strings.Contains(bodyStr, "No Students Found") {
			fmt.Println("   ⚠️  No ECSE students found")
		} else if strings.Contains(bodyStr, "student-row") {
			fmt.Println("   ✅ ECSE data is displaying")
			
			// Count students
			studentCount := strings.Count(bodyStr, "student-row")
			if studentCount > 0 {
				fmt.Printf("   ✅ Found %d ECSE students\n", studentCount)
			}
		}
		
		// Check styling
		if strings.Contains(bodyStr, "background: #0f0c29") {
			fmt.Println("   ✅ Dark theme is applied")
		}
		
		// Check for specific student data
		if strings.Contains(bodyStr, "IEP") {
			fmt.Println("   ✅ IEP status field present")
		}
		if strings.Contains(bodyStr, "Transportation") {
			fmt.Println("   ✅ Transportation field present")
		}
	} else {
		fmt.Println("❌ Unexpected response")
		// Show preview
		if len(bodyStr) > 300 {
			fmt.Printf("Preview: %s...\n", bodyStr[:300])
		}
	}
}