package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"
	"time"
)

func main() {
	// Create cookie jar to handle session cookies
	jar, err := cookiejar.New(nil)
	if err != nil {
		log.Fatal("Failed to create cookie jar:", err)
	}

	client := &http.Client{
		Jar:     jar,
		Timeout: 10 * time.Second,
	}

	baseURL := "http://localhost:5000"
	if len(os.Args) > 1 {
		baseURL = os.Args[1]
	}

	// Step 1: Login
	loginURL := baseURL + "/"
	loginData := url.Values{
		"username": {"admin"},
		"password": {"SecureAdminPass123!"},
	}

	fmt.Println("Testing Fleet Management System Content")
	fmt.Println("=======================================")
	fmt.Println("\n1. Logging in as admin...")
	
	resp, err := client.PostForm(loginURL, loginData)
	if err != nil {
		log.Fatal("Failed to login:", err)
	}
	defer resp.Body.Close()

	// Check if login was successful
	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusSeeOther {
		fmt.Println("✓ Login successful")
	} else {
		log.Fatal("✗ Login failed with status:", resp.StatusCode)
	}

	// Test Manager Dashboard
	fmt.Println("\n2. Testing Manager Dashboard Content...")
	dashboardTests := []struct {
		url      string
		expected []string
		name     string
	}{
		{
			url:  "/manager-dashboard",
			name: "Manager Dashboard",
			expected: []string{
				"Fleet Overview",      // h2 heading
				"Quick Actions",       // h2 heading
				"Total Buses",         // metric label
				"Active Drivers",      // metric label
				"Total Students",      // metric label
				"Active Routes",       // metric label
				"Manage Fleet",        // quick action
				"Assign Routes",       // quick action
				"Manage Students",     // quick action
				"Recent Activity",     // h2 heading
			},
		},
		{
			url:  "/students",
			name: "Students Page",
			expected: []string{
				"Student Management",  // h1 heading
				"All Students",        // h2 heading
				"Total Students",      // stat label
				"Active Students",     // stat label
				"Morning Pickups",     // stat label
				"Afternoon Dropoffs",  // stat label
				"Add Student",         // button text
			},
		},
	}

	for _, test := range dashboardTests {
		fmt.Printf("\n   Testing %s (%s)...\n", test.name, test.url)
		resp, err := client.Get(baseURL + test.url)
		if err != nil {
			fmt.Printf("   ✗ Failed to fetch %s: %v\n", test.url, err)
			continue
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("   ✗ Failed to read response: %v\n", err)
			continue
		}

		content := string(body)
		
		// Check for expected content
		fmt.Println("   Checking for expected content:")
		for _, expected := range test.expected {
			if strings.Contains(content, expected) {
				fmt.Printf("   ✓ Found: '%s'\n", expected)
			} else {
				fmt.Printf("   ✗ Missing: '%s'\n", expected)
			}
		}

		// Save the actual HTML for inspection
		filename := strings.ReplaceAll(test.url[1:], "/", "_") + "_actual.html"
		err = os.WriteFile(filename, body, 0644)
		if err != nil {
			fmt.Printf("   ! Failed to save HTML: %v\n", err)
		} else {
			fmt.Printf("   → HTML saved to: %s\n", filename)
		}
	}

	fmt.Println("\n3. Summary")
	fmt.Println("===========")
	fmt.Println("The test script has checked for the following content:")
	fmt.Println("- Manager Dashboard: 'Fleet Overview' section with metrics")
	fmt.Println("- Manager Dashboard: 'Quick Actions' section with action buttons")
	fmt.Println("- Students Page: 'Student Management' heading")
	fmt.Println("- Students Page: 'All Students' section")
	fmt.Println("- Students Page: Student statistics labels")
	fmt.Println("\nIf the server is running, the actual HTML has been saved for inspection.")
}