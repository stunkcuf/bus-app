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
	client := &http.Client{
		Jar: jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			fmt.Printf("Redirect detected: %s\n", req.URL)
			return nil
		},
	}

	// First GET the login page to check if server is up
	fmt.Println("=== Getting login page ===")
	resp, err := client.Get("http://localhost:5003/")
	if err != nil {
		fmt.Printf("Error getting login page: %v\n", err)
		return
	}
	body, _ := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	
	if strings.Contains(string(body), "Fleet Management System Login") {
		fmt.Println("✅ Login page loaded successfully")
	} else {
		fmt.Println("❌ Login page not found")
		return
	}

	// Check for CSRF token in form
	bodyStr := string(body)
	hasCSRF := strings.Contains(bodyStr, "csrf_token")
	fmt.Printf("CSRF token in form: %v\n", hasCSRF)

	// Now POST login
	fmt.Println("\n=== Posting login ===")
	loginData := url.Values{
		"username": {"admin"},
		"password": {"Headstart1"},
	}
	
	resp, err = client.PostForm("http://localhost:5003/", loginData)
	if err != nil {
		fmt.Printf("Error posting login: %v\n", err)
		return
	}
	defer resp.Body.Close()
	
	fmt.Printf("Status Code: %d\n", resp.StatusCode)
	fmt.Printf("Location Header: '%s'\n", resp.Header.Get("Location"))
	
	// Check cookies
	cookies := client.Jar.Cookies(&url.URL{Scheme: "http", Host: "localhost:5003"})
	fmt.Printf("Cookies received: %d\n", len(cookies))
	for _, cookie := range cookies {
		fmt.Printf("  - %s = %s (HttpOnly: %v, Secure: %v)\n", 
			cookie.Name, cookie.Value[:min(10, len(cookie.Value))], 
			cookie.HttpOnly, cookie.Secure)
	}
	
	// Check response body
	body2, _ := ioutil.ReadAll(resp.Body)
	bodyStr2 := string(body2)
	
	if strings.Contains(bodyStr2, "Invalid username or password") {
		fmt.Println("\n❌ Got 'Invalid username or password' error")
	} else if strings.Contains(bodyStr2, "pending approval") {
		fmt.Println("\n❌ Got 'pending approval' error")
	} else if strings.Contains(bodyStr2, "Fleet Management System Login") {
		fmt.Println("\n❌ Still showing login page")
	} else if strings.Contains(bodyStr2, "Manager Dashboard") || strings.Contains(bodyStr2, "Driver Dashboard") {
		fmt.Println("\n✅ Successfully logged in and showing dashboard!")
	} else {
		fmt.Println("\n⚠️ Unknown response")
		// Print first 500 chars
		if len(bodyStr2) > 500 {
			fmt.Printf("Response preview: %s...\n", bodyStr2[:500])
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}