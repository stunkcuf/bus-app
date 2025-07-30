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

	fmt.Println("Testing Students Page Access")
	fmt.Println("===========================")

	// Login as driver
	fmt.Println("\n1. Logging in as bjmathis...")
	loginData := url.Values{
		"username": {"bjmathis"},
		"password": {"driver123"},
	}
	
	resp, err := client.PostForm("http://localhost:5003/", loginData)
	if err != nil {
		fmt.Printf("❌ Login error: %v\n", err)
		return
	}
	
	body, _ := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	
	if strings.Contains(string(body), "Driver Dashboard") {
		fmt.Println("✅ Successfully logged in as driver")
	} else {
		fmt.Println("❌ Login failed")
		if strings.Contains(string(body), "Invalid") {
			fmt.Println("   Invalid credentials")
		}
		return
	}

	// Access students page
	fmt.Println("\n2. Accessing /students...")
	resp, err = client.Get("http://localhost:5003/students")
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		return
	}
	
	body, _ = ioutil.ReadAll(resp.Body)
	bodyStr := string(body)
	resp.Body.Close()
	
	fmt.Printf("Status: %d\n", resp.StatusCode)
	fmt.Printf("URL: %s\n", resp.Request.URL)
	
	// Check what we got
	if strings.Contains(bodyStr, "Student Management") {
		fmt.Println("✅ Students page loaded successfully!")
		
		// Check for data
		if strings.Contains(bodyStr, "No students assigned") {
			fmt.Println("   ⚠️  No students assigned to this driver")
		} else if strings.Contains(bodyStr, "student-row") || strings.Contains(bodyStr, "tbody") {
			fmt.Println("   ✅ Students data is displaying")
		}
		
		// Check styling
		if strings.Contains(bodyStr, "background: #0f0c29") {
			fmt.Println("   ✅ Dark theme is applied")
		} else {
			fmt.Println("   ⚠️  Dark theme might be missing")
		}
	} else if strings.Contains(bodyStr, "Error") {
		fmt.Println("❌ Error loading page")
		// Extract error message
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