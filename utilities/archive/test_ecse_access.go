package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
)

func main() {
	// Create a cookie jar to maintain session
	jar, err := cookiejar.New(nil)
	if err != nil {
		log.Fatal("Failed to create cookie jar:", err)
	}

	client := &http.Client{
		Jar: jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Don't follow redirects automatically
			if len(via) >= 1 {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}

	baseURL := "http://localhost:5003"

	// Step 1: Login
	fmt.Println("Step 1: Logging in...")
	loginData := url.Values{
		"username": {"admin"},
		"password": {"admin123"},
	}

	resp, err := client.PostForm(baseURL+"/", loginData)
	if err != nil {
		log.Fatal("Failed to login:", err)
	}
	defer resp.Body.Close()

	fmt.Printf("Login response status: %d\n", resp.StatusCode)
	
	// Check if we got redirected (successful login)
	if resp.StatusCode == 303 || resp.StatusCode == 302 {
		location := resp.Header.Get("Location")
		fmt.Printf("Redirected to: %s\n", location)
		
		// Follow the redirect
		resp2, err := client.Get(baseURL + location)
		if err != nil {
			log.Printf("Failed to follow redirect: %v", err)
		} else {
			resp2.Body.Close()
			fmt.Printf("Dashboard loaded successfully\n")
		}
	}

	// Step 2: Access ECSE Dashboard
	fmt.Println("\nStep 2: Accessing ECSE Dashboard...")
	resp, err = client.Get(baseURL + "/ecse-dashboard")
	if err != nil {
		log.Fatal("Failed to access ECSE dashboard:", err)
	}
	defer resp.Body.Close()

	fmt.Printf("ECSE Dashboard response status: %d\n", resp.StatusCode)

	// Read response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal("Failed to read response:", err)
	}

	// Check if we got the ECSE dashboard
	bodyStr := string(body)
	if strings.Contains(bodyStr, "ECSE Dashboard") {
		fmt.Println("Successfully loaded ECSE Dashboard!")
		
		// Look for debug info
		if strings.Contains(bodyStr, "Debug Information") {
			fmt.Println("\nFound debug information in response")
			// Extract some info
			if idx := strings.Index(bodyStr, "Total Students in data:"); idx != -1 {
				snippet := bodyStr[idx:min(idx+100, len(bodyStr))]
				fmt.Println(snippet)
			}
		}
		
		// Check if students are displayed
		studentCount := strings.Count(bodyStr, "ecse-student-card")
		fmt.Printf("\nFound %d student cards in the HTML\n", studentCount)
		
	} else if strings.Contains(bodyStr, "Login") {
		fmt.Println("ERROR: Got redirected to login page")
		fmt.Println("Session might not be maintained properly")
	} else {
		fmt.Println("ERROR: Unknown response")
		fmt.Printf("First 500 chars: %s\n", bodyStr[:min(500, len(bodyStr))])
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}