package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"
)

func main() {
	fmt.Println("🔍 Checking Students Page Access")
	fmt.Println("=" + strings.Repeat("=", 50))

	// Create a cookie jar to maintain session
	jar, _ := cookiejar.New(nil)
	client := &http.Client{
		Jar: jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			fmt.Printf("  🔄 Redirect: %s -> %s\n", via[len(via)-1].URL, req.URL)
			return nil
		},
	}

	baseURL := "http://localhost:5003"
	if envURL := os.Getenv("BASE_URL"); envURL != "" {
		baseURL = envURL
	}

	// Test 1: Direct access without login
	fmt.Println("\n1. Testing direct access to /students (no auth):")
	resp, err := client.Get(baseURL + "/students")
	if err != nil {
		fmt.Printf("   ❌ Error: %v\n", err)
	} else {
		fmt.Printf("   Final URL: %s\n", resp.Request.URL)
		fmt.Printf("   Status: %d %s\n", resp.StatusCode, resp.Status)
		resp.Body.Close()
	}

	// Test 2: Login as driver
	fmt.Println("\n2. Testing driver login:")
	loginData := url.Values{
		"username": {"bjmathis"},
		"password": {"password123"},
	}
	
	resp, err = client.PostForm(baseURL+"/", loginData)
	if err != nil {
		fmt.Printf("   ❌ Login error: %v\n", err)
		return
	}
	defer resp.Body.Close()
	
	body, _ := ioutil.ReadAll(resp.Body)
	if strings.Contains(string(body), "Driver Dashboard") {
		fmt.Println("   ✅ Successfully logged in as driver")
	} else if strings.Contains(string(body), "Invalid username or password") {
		fmt.Println("   ❌ Login failed - invalid credentials")
		return
	} else {
		fmt.Printf("   ⚠️  Unexpected response after login\n")
	}

	// Test 3: Access students page as driver
	fmt.Println("\n3. Testing /students access as driver:")
	resp, err = client.Get(baseURL + "/students")
	if err != nil {
		fmt.Printf("   ❌ Error: %v\n", err)
	} else {
		fmt.Printf("   Status: %d %s\n", resp.StatusCode, resp.Status)
		body, _ := ioutil.ReadAll(resp.Body)
		bodyStr := string(body)
		
		if strings.Contains(bodyStr, "Student Management") {
			fmt.Println("   ✅ Successfully loaded students page")
			
			// Check for students data
			if strings.Contains(bodyStr, "No students assigned") {
				fmt.Println("   ⚠️  No students assigned to this driver")
			} else if strings.Contains(bodyStr, "student-row") {
				fmt.Println("   ✅ Students data is displaying")
			}
		} else if strings.Contains(bodyStr, "Fleet Management System Login") {
			fmt.Println("   ❌ Redirected to login page - session issue")
		} else {
			fmt.Println("   ❌ Unexpected response")
			// Print first 500 chars to debug
			if len(bodyStr) > 500 {
				fmt.Printf("   Response preview: %s...\n", bodyStr[:500])
			} else {
				fmt.Printf("   Response: %s\n", bodyStr)
			}
		}
		resp.Body.Close()
	}

	// Test 4: Login as manager and try to access
	fmt.Println("\n4. Testing manager access to /students:")
	
	// First logout
	client.Get(baseURL + "/logout")
	
	// Login as manager
	loginData = url.Values{
		"username": {"admin"},
		"password": {"admin123"},
	}
	
	resp, err = client.PostForm(baseURL+"/", loginData)
	if err != nil {
		fmt.Printf("   ❌ Manager login error: %v\n", err)
		return
	}
	resp.Body.Close()
	
	// Try to access students page
	resp, err = client.Get(baseURL + "/students")
	if err != nil {
		fmt.Printf("   ❌ Error: %v\n", err)
	} else {
		fmt.Printf("   Status: %d\n", resp.StatusCode)
		if resp.Request.URL.Path == "/" {
			fmt.Println("   ✅ Correctly redirected to login (managers can't access)")
		} else {
			fmt.Printf("   Final URL: %s\n", resp.Request.URL)
		}
		resp.Body.Close()
	}

	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("SUMMARY:")
	fmt.Println("- Students page requires driver role")
	fmt.Println("- Make sure driver account exists and has correct password")
	fmt.Println("- Check if driver has assigned routes/students")
}