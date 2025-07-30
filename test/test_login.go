package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"
)

func main() {
	baseURL := "http://localhost:8080"
	
	// Create HTTP client with cookie jar
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}
	
	// Test credentials
	credentials := []struct {
		username string
		password string
	}{
		{"admin", "Headstart1"},
		{"admin", "admin123"},
		{"admin", "admin"},
		{"testmanager", "Test123456!"},
		{"testmanager123", "password123"},
		{"driver1", "password123"},
	}
	
	for _, cred := range credentials {
		fmt.Printf("\nTesting %s / %s...\n", cred.username, cred.password)
		
		// Get login page for CSRF token
		resp, err := client.Get(baseURL + "/")
		if err != nil {
			fmt.Printf("Error getting login page: %v\n", err)
			continue
		}
		
		body, _ := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		
		// Extract CSRF token
		re := regexp.MustCompile(`<input type="hidden" name="csrf_token" value="([^"]*)"`)
		matches := re.FindSubmatch(body)
		csrfToken := ""
		if len(matches) > 1 {
			csrfToken = string(matches[1])
		}
		
		// Try login
		loginData := url.Values{
			"username":   {cred.username},
			"password":   {cred.password},
			"csrf_token": {csrfToken},
		}
		
		resp, err = client.PostForm(baseURL+"/", loginData)
		if err != nil {
			fmt.Printf("  ❌ Error: %v\n", err)
			continue
		}
		
		if resp.StatusCode == http.StatusFound {
			location := resp.Header.Get("Location")
			fmt.Printf("  ✓ SUCCESS! Redirecting to: %s\n", location)
			
			// Follow redirect
			resp2, _ := client.Get(baseURL + location)
			body2, _ := ioutil.ReadAll(resp2.Body)
			resp2.Body.Close()
			
			// Check what page we got
			if strings.Contains(string(body2), "Dashboard") {
				fmt.Printf("  ✓ Successfully accessed dashboard\n")
			}
		} else {
			body, _ := ioutil.ReadAll(resp.Body)
			if strings.Contains(string(body), "Invalid username or password") {
				fmt.Printf("  ❌ Invalid credentials\n")
			} else {
				fmt.Printf("  ❌ Status: %d\n", resp.StatusCode)
			}
		}
		resp.Body.Close()
		
		// Clear cookies for next attempt
		jar, _ = cookiejar.New(nil)
		client.Jar = jar
	}
}