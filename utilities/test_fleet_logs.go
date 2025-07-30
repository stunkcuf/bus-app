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
	fmt.Println("Testing Fleet Page and Checking Logs...")
	fmt.Println("=======================================")

	// Create a cookie jar
	jar, _ := cookiejar.New(nil)
	client := &http.Client{
		Jar:     jar,
		Timeout: 10 * time.Second,
	}

	// First, login
	loginURL := "http://localhost:5003/login"
	formData := url.Values{
		"username": {"admin"},
		"password": {"admin123"},
	}

	fmt.Println("\n1. Attempting login...")
	resp, err := client.PostForm(loginURL, formData)
	if err != nil {
		fmt.Printf("Login failed: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusFound {
		fmt.Println("Login successful!")
	} else {
		fmt.Printf("Login returned status: %d\n", resp.StatusCode)
	}

	// Wait a moment
	time.Sleep(1 * time.Second)

	// Now access the fleet page
	fmt.Println("\n2. Accessing /fleet page...")
	fleetURL := "http://localhost:5003/fleet"
	resp2, err := client.Get(fleetURL)
	if err != nil {
		fmt.Printf("Failed to access fleet page: %v\n", err)
		return
	}
	defer resp2.Body.Close()

	fmt.Printf("Fleet page status: %d\n", resp2.StatusCode)

	// Read the response
	body, _ := ioutil.ReadAll(resp2.Body)
	bodyStr := string(body)

	// Check if we got redirected to login
	if strings.Contains(bodyStr, "Login") && strings.Contains(bodyStr, "password") {
		fmt.Println("ERROR: Got redirected to login page")
		return
	}

	// Look for bus data in the response
	busCount := strings.Count(bodyStr, "bus-maintenance/")
	fmt.Printf("\n3. Found %d buses in the HTML response\n", busCount)

	// Check for key elements
	if strings.Contains(bodyStr, "Fleet Overview") {
		fmt.Println("✓ Fleet Overview section found")
	}
	if strings.Contains(bodyStr, "Active Buses") {
		fmt.Println("✓ Active Buses stat found")
	}
	if strings.Contains(bodyStr, "table-modern") {
		fmt.Println("✓ Fleet table found")
	}

	// Count specific bus IDs
	busIDs := []string{"24", "25", "26", "52", "58", "59", "6", "60", "7", "8"}
	fmt.Println("\n4. Checking for specific bus IDs:")
	for _, id := range busIDs {
		if strings.Contains(bodyStr, fmt.Sprintf("bus-maintenance/%s", id)) {
			fmt.Printf("   ✓ Bus %s found\n", id)
		} else {
			fmt.Printf("   ✗ Bus %s NOT found\n", id)
		}
	}

	fmt.Println("\n5. Server Debug Logs:")
	fmt.Println("Check the server console output for these debug messages:")
	fmt.Println("- DEBUG: Fleet handler called by user: admin")
	fmt.Println("- DEBUG: Total bus count in database: X")
	fmt.Println("- DEBUG: Loaded X buses")
	fmt.Println("- DEBUG: Before rendering")
	fmt.Println("\nNote: The logs should appear in the terminal where the server is running")
}