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

	// Login
	loginData := url.Values{
		"username": {"admin"},
		"password": {"admin123"},
	}
	
	resp, _ := client.PostForm("http://localhost:5003/", loginData)
	resp.Body.Close()

	// Get ECSE page
	resp, err := client.Get("http://localhost:5003/view-ecse-reports")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	
	body, _ := ioutil.ReadAll(resp.Body)
	bodyStr := string(body)
	resp.Body.Close()
	
	// Find the table section
	tableStart := strings.Index(bodyStr, "<!-- Students Table -->")
	if tableStart > 0 {
		tableEnd := strings.Index(bodyStr[tableStart:], "</div>")
		if tableEnd > 0 {
			tableSection := bodyStr[tableStart:tableStart+tableEnd]
			
			// Check if the conditional is true
			if strings.Contains(tableSection, "{{if .Data.Students}}") {
				fmt.Println("Found template conditional: {{if .Data.Students}}")
			}
			
			// Check what's actually rendered
			if strings.Contains(tableSection, "<table") {
				fmt.Println("✅ Table is rendered")
			} else if strings.Contains(tableSection, "No Students Found") {
				fmt.Println("❌ 'No Students Found' is rendered")
				
				// Extract the relevant section
				noDataStart := strings.Index(tableSection, "{{else}}")
				if noDataStart > 0 {
					fmt.Println("\nTemplate else block is being executed")
					fmt.Println("This means .Data.Students is empty or false")
				}
			}
		}
	}
	
	// Check for any error messages in response
	if strings.Contains(bodyStr, "Error") && !strings.Contains(bodyStr, "ErrorDocument") {
		errorStart := strings.Index(bodyStr, "Error")
		errorEnd := errorStart + 100
		if errorEnd > len(bodyStr) {
			errorEnd = len(bodyStr)
		}
		fmt.Printf("\nFound error text: %s\n", bodyStr[errorStart:errorEnd])
	}
}