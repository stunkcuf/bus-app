package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
)

func main() {
	// Create HTTP client with cookie jar
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}
	
	// First login
	fmt.Println("1. Logging in...")
	loginData := url.Values{
		"username": {"admin"},
		"password": {"Test123456!"},
	}
	
	resp, err := client.PostForm("http://localhost:8080/", loginData)
	if err != nil {
		fmt.Printf("Login error: %v\n", err)
		return
	}
	
	finalURL := resp.Request.URL.String()
	resp.Body.Close()
	
	if finalURL == "http://localhost:8080/manager-dashboard" {
		fmt.Println("   ✓ Login successful")
	} else {
		fmt.Println("   ❌ Login failed")
		return
	}
	
	// Test API endpoints
	fmt.Println("\n2. Testing API endpoints...")
	
	endpoints := []string{
		"/api/routes",
		"/api/buses",
		"/api/drivers",
		"/api/students",
		"/api/fleet-vehicles",
		"/api/route-assignments",
		"/api/ecse-students",
		"/api/maintenance-records",
	}
	
	for _, endpoint := range endpoints {
		fmt.Printf("\n   Testing %s...\n", endpoint)
		
		resp, err := client.Get("http://localhost:8080" + endpoint)
		if err != nil {
			fmt.Printf("   ❌ Error: %v\n", err)
			continue
		}
		
		body, _ := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		
		fmt.Printf("   Status: %d\n", resp.StatusCode)
		if resp.StatusCode == 200 {
			fmt.Printf("   Response length: %d bytes\n", len(body))
			fmt.Printf("   First 100 chars: %.100s\n", string(body))
		} else {
			fmt.Printf("   Response: %s\n", string(body))
		}
	}
}