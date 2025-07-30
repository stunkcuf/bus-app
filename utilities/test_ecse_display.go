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
	// Create client with cookie jar
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}

	// Login as admin
	loginData := url.Values{
		"username": {"admin"},
		"password": {"Headstart1"},
	}
	
	fmt.Println("Logging in...")
	resp, _ := client.PostForm("http://localhost:5003/", loginData)
	resp.Body.Close()

	// Access ECSE reports
	fmt.Println("\nAccessing ECSE reports page...")
	resp, err := client.Get("http://localhost:5003/view-ecse-reports")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer resp.Body.Close()
	
	body, _ := ioutil.ReadAll(resp.Body)
	bodyStr := string(body)
	
	fmt.Printf("Status code: %d\n", resp.StatusCode)
	fmt.Printf("Page length: %d bytes\n", len(bodyStr))
	
	// Check for key elements
	if strings.Contains(bodyStr, "ECSE Student Reports") {
		fmt.Println("✅ ECSE reports page loaded")
	}
	
	if strings.Contains(bodyStr, "No Students Found") {
		fmt.Println("⚠️ 'No Students Found' message present")
	}
	
	// Count student rows
	studentCount := strings.Count(bodyStr, "student-row")
	fmt.Printf("Found %d student rows\n", studentCount)
	
	// Check for specific student data
	if strings.Contains(bodyStr, "StudentID") {
		fmt.Println("✅ Student ID column found")
	}
	
	// Look for template errors
	if strings.Contains(bodyStr, "{{") && strings.Contains(bodyStr, "}}") {
		fmt.Println("❌ Template rendering error - found unprocessed template tags")
	}
	
	// Check if data object exists
	if strings.Contains(bodyStr, "script") {
		// Look for any JavaScript that might show data
		idx := strings.Index(bodyStr, "<script>")
		if idx > 0 {
			endIdx := strings.Index(bodyStr[idx:], "</script>")
			if endIdx > 0 && endIdx < 1000 {
				scriptContent := bodyStr[idx : idx+endIdx]
				if strings.Contains(scriptContent, "console.log") {
					fmt.Println("Found console.log in script")
				}
			}
		}
	}
}