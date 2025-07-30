package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

func main() {
	// Test login directly without cookies
	loginData := url.Values{
		"username": {"admin"},
		"password": {"Headstart1"},
	}
	
	resp, err := http.PostForm("http://localhost:5003/", loginData)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer resp.Body.Close()
	
	fmt.Printf("Status Code: %d\n", resp.StatusCode)
	fmt.Printf("Location Header: %s\n", resp.Header.Get("Location"))
	
	body, _ := ioutil.ReadAll(resp.Body)
	bodyStr := string(body)
	
	// Check for error messages
	if strings.Contains(bodyStr, "Invalid username or password") {
		fmt.Println("❌ Login failed: Invalid credentials")
	} else if strings.Contains(bodyStr, "pending approval") {
		fmt.Println("❌ Login failed: Account pending approval")
	} else if strings.Contains(bodyStr, "Fleet Management System Login") {
		fmt.Println("❌ Still on login page")
		
		// Check if there's any other error
		if idx := strings.Index(bodyStr, "alert-danger"); idx > 0 {
			end := strings.Index(bodyStr[idx:], "</div>")
			if end > 0 {
				fmt.Printf("Error found: %s\n", bodyStr[idx:idx+end])
			}
		}
	} else if resp.StatusCode == 303 || resp.StatusCode == 302 {
		fmt.Println("✅ Login successful - redirecting")
	}
	
	// Check cookies
	fmt.Println("\nCookies received:")
	for _, cookie := range resp.Cookies() {
		fmt.Printf("  %s = %s\n", cookie.Name, cookie.Value)
	}
}