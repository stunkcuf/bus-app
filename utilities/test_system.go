package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
)

func main() {
	// Create a cookie jar to store session cookies
	jar, err := cookiejar.New(nil)
	if err != nil {
		log.Fatal(err)
	}
	
	client := &http.Client{
		Jar: jar,
	}
	
	baseURL := "http://localhost:5003"
	
	// Step 1: Test if server is running
	fmt.Println("1. Testing server connectivity...")
	resp, err := client.Get(baseURL + "/health")
	if err != nil {
		log.Fatal("Server not running:", err)
	}
	defer resp.Body.Close()
	
	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("   Health check: %s\n\n", string(body))
	
	// Step 2: Get login page (to get any CSRF token if needed)
	fmt.Println("2. Getting login page...")
	resp, err = client.Get(baseURL + "/")
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode == 200 {
		fmt.Println("   Login page loaded successfully")
	}
	
	// Step 3: Attempt login
	fmt.Println("\n3. Attempting login with admin/admin...")
	formData := url.Values{
		"username": {"admin"},
		"password": {"admin"},
	}
	
	resp, err = client.PostForm(baseURL+"/", formData)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	
	// Check if we got redirected (successful login)
	fmt.Printf("   Response status: %d\n", resp.StatusCode)
	fmt.Printf("   Response URL: %s\n", resp.Request.URL.Path)
	
	// Check if we have a session cookie
	cookies := jar.Cookies(resp.Request.URL)
	fmt.Printf("   Cookies received: %d\n", len(cookies))
	for _, cookie := range cookies {
		fmt.Printf("   - %s: %s\n", cookie.Name, cookie.Value)
	}
	
	// Step 4: Try to access fleet page
	fmt.Println("\n4. Attempting to access fleet page...")
	resp, err = client.Get(baseURL + "/fleet")
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	
	fmt.Printf("   Fleet page status: %d\n", resp.StatusCode)
	fmt.Printf("   Response URL: %s\n", resp.Request.URL.Path)
	
	// Read some of the response to check content
	body, _ = io.ReadAll(resp.Body)
	bodyStr := string(body)
	
	if strings.Contains(bodyStr, "Login") {
		fmt.Println("   ❌ Still on login page - authentication failed")
	} else if strings.Contains(bodyStr, "Fleet") || strings.Contains(bodyStr, "Vehicles") {
		fmt.Println("   ✅ Fleet page accessed successfully")
		
		// Count vehicles
		vehicleCount := strings.Count(bodyStr, "vehicle-card") + strings.Count(bodyStr, "bus-card")
		fmt.Printf("   Vehicle cards found: %d\n", vehicleCount)
		
		// Look for specific indicators
		if strings.Contains(bodyStr, "54") || strings.Contains(bodyStr, "Total: 54") {
			fmt.Println("   ✅ Found indication of 54 vehicles")
		}
	}
	
	// Step 5: Summary
	fmt.Println("\n5. Test Summary:")
	if len(cookies) > 0 && !strings.Contains(bodyStr, "Login") {
		fmt.Println("   ✅ Login successful")
		fmt.Println("   ✅ Session management working")
		fmt.Println("   ✅ Fleet page accessible")
	} else {
		fmt.Println("   ❌ Login failed or session not maintained")
		fmt.Println("   Please check:")
		fmt.Println("   - Server logs for error messages")
		fmt.Println("   - Database connection")
		fmt.Println("   - Password hashing (admin password should be hashed)")
	}
}