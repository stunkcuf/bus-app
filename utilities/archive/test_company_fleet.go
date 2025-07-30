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
	// Create client with cookie jar
	jar, _ := cookiejar.New(nil)
	client := &http.Client{
		Jar:     jar,
		Timeout: 30 * time.Second, // 30 second timeout
	}

	// Login first
	loginData := url.Values{
		"username": {"admin"},
		"password": {"Headstart1"},
	}
	
	fmt.Println("Logging in...")
	resp, err := client.PostForm("http://localhost:5003/", loginData)
	if err != nil {
		fmt.Printf("Login error: %v\n", err)
		return
	}
	resp.Body.Close()

	// Now access company fleet
	fmt.Println("\nAccessing company fleet page...")
	start := time.Now()
	
	resp, err = client.Get("http://localhost:5003/company-fleet")
	elapsed := time.Since(start)
	
	if err != nil {
		fmt.Printf("❌ Error accessing company fleet: %v\n", err)
		fmt.Printf("   Time elapsed: %v\n", elapsed)
		if strings.Contains(err.Error(), "timeout") {
			fmt.Println("   This appears to be a timeout!")
		}
		return
	}
	defer resp.Body.Close()
	
	fmt.Printf("✅ Response received in %v\n", elapsed)
	fmt.Printf("   Status code: %d\n", resp.StatusCode)
	
	// Read body
	body, _ := ioutil.ReadAll(resp.Body)
	bodyStr := string(body)
	
	if strings.Contains(bodyStr, "Company Fleet") {
		fmt.Println("   Page loaded successfully")
		
		// Count vehicles
		vehicleCount := strings.Count(bodyStr, "vehicle-card")
		fmt.Printf("   Found %d vehicle cards\n", vehicleCount)
		
		if strings.Contains(bodyStr, "No Vehicles Found") {
			fmt.Println("   ⚠️ 'No Vehicles Found' message present")
		}
	} else if strings.Contains(bodyStr, "Login") {
		fmt.Println("   ❌ Redirected to login page")
	} else {
		fmt.Println("   ⚠️ Unknown response")
		if len(bodyStr) > 500 {
			fmt.Printf("   Preview: %s...\n", bodyStr[:500])
		}
	}
}