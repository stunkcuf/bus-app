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
	client := &http.Client{
		Jar: jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			fmt.Printf("Redirect: %s -> %s (Status: %d)\n", via[len(via)-1].URL, req.URL, req.Response.StatusCode)
			return nil
		},
	}
	
	// Test login
	fmt.Println("Testing login...")
	loginData := url.Values{
		"username": {"admin"},
		"password": {"Test123456!"},
	}
	
	resp, err := client.PostForm("http://localhost:8080/", loginData)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer resp.Body.Close()
	
	fmt.Printf("Final status: %d\n", resp.StatusCode)
	fmt.Printf("Final URL: %s\n", resp.Request.URL)
	
	body, _ := ioutil.ReadAll(resp.Body)
	bodyStr := string(body)
	
	// Check what we got
	if strings.Contains(bodyStr, "Dashboard") {
		fmt.Println("✓ Successfully reached dashboard!")
		
		// Extract dashboard type
		if strings.Contains(bodyStr, "Manager Dashboard") {
			fmt.Println("  - Manager Dashboard")
		} else if strings.Contains(bodyStr, "Driver Dashboard") {
			fmt.Println("  - Driver Dashboard")
		}
		
		// Check for data
		fmt.Printf("  - Table rows: %d\n", strings.Count(bodyStr, "<tr>"))
		fmt.Printf("  - Cards: %d\n", strings.Count(bodyStr, "class=\"card"))
	} else if strings.Contains(bodyStr, "Invalid username or password") {
		fmt.Println("❌ Invalid credentials")
	} else if strings.Contains(bodyStr, "login") {
		fmt.Println("❌ Still on login page")
	} else {
		fmt.Printf("? Unknown response (length: %d)\n", len(bodyStr))
		// Save for inspection
		ioutil.WriteFile("login_response.html", body, 0644)
		fmt.Println("  Saved to login_response.html")
	}
}