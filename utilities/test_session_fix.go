package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"time"
)

func main() {
	fmt.Println("=== Testing Session Timeout Fix ===")
	fmt.Println()
	
	// Create HTTP client with cookie jar
	jar, _ := cookiejar.New(nil)
	client := &http.Client{
		Jar: jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	
	baseURL := "http://localhost:8080"
	
	// Test 1: Login and get session
	fmt.Println("Test 1: Creating new session...")
	loginData := url.Values{
		"username": {"admin"},
		"password": {"Headstart1"},
	}
	
	resp, err := client.PostForm(baseURL+"/login", loginData)
	if err != nil {
		log.Fatal("Login failed:", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode == 302 || resp.StatusCode == 303 {
		fmt.Println("✓ Login successful, session created")
		
		// Get session cookie
		cookies := jar.Cookies(&url.URL{Scheme: "http", Host: "localhost:8080"})
		var sessionCookie *http.Cookie
		for _, cookie := range cookies {
			if cookie.Name == "hsbus_session" {
				sessionCookie = cookie
				break
			}
		}
		
		if sessionCookie != nil {
			fmt.Printf("✓ Session cookie: %s...\n", sessionCookie.Value[:20])
			
			// Read sessions.json to check the session details
			sessionData, err := ioutil.ReadFile("sessions.json")
			if err == nil {
				var sessions map[string]map[string]interface{}
				json.Unmarshal(sessionData, &sessions)
				
				if session, exists := sessions[sessionCookie.Value]; exists {
					fmt.Printf("✓ Session found in storage\n")
					fmt.Printf("  - Expires at: %v\n", session["expires_at"])
					fmt.Printf("  - Last access: %v\n", session["last_access"])
				}
			}
		}
	} else {
		fmt.Printf("✗ Login failed with status: %d\n", resp.StatusCode)
	}
	
	fmt.Println("\nTest 2: Accessing protected page (should update last_access)...")
	time.Sleep(2 * time.Second)
	
	resp2, err := client.Get(baseURL + "/manager-dashboard")
	if err != nil {
		log.Printf("Failed to access dashboard: %v", err)
	} else {
		defer resp2.Body.Close()
		if resp2.StatusCode == 200 {
			fmt.Println("✓ Successfully accessed protected page")
			
			// Check if session was updated
			cookies := jar.Cookies(&url.URL{Scheme: "http", Host: "localhost:8080"})
			for _, cookie := range cookies {
				if cookie.Name == "hsbus_session" {
					// Read sessions.json again
					sessionData, err := ioutil.ReadFile("sessions.json")
					if err == nil {
						var sessions map[string]map[string]interface{}
						json.Unmarshal(sessionData, &sessions)
						
						if session, exists := sessions[cookie.Value]; exists {
							fmt.Printf("✓ Session updated in storage\n")
							fmt.Printf("  - New expires at: %v\n", session["expires_at"])
							fmt.Printf("  - New last access: %v\n", session["last_access"])
							
							// Parse the expiration time
							if expiresStr, ok := session["expires_at"].(string); ok {
								expires, err := time.Parse(time.RFC3339Nano, expiresStr)
								if err == nil {
									timeUntilExpiry := expires.Sub(time.Now())
									fmt.Printf("  - Time until expiry: %.1f hours\n", timeUntilExpiry.Hours())
									
									if timeUntilExpiry.Hours() > 23 {
										fmt.Println("✓ Session expiration extended correctly (sliding window working)")
									} else {
										fmt.Println("✗ Session expiration not extended properly")
									}
								}
							}
						}
					}
					break
				}
			}
		} else if resp2.StatusCode == 302 {
			fmt.Printf("✗ Redirected to login (session expired or invalid): %d\n", resp2.StatusCode)
		}
	}
	
	fmt.Println("\n=== Session Test Complete ===")
}