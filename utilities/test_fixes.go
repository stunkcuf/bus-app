package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"strings"
)

func main() {
	// Create cookie jar
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}

	// Login as driver
	loginData := strings.NewReader("username=test&password=Headstart1")
	resp, err := client.Post("http://127.0.0.1:8080/", "application/x-www-form-urlencoded", loginData)
	if err != nil {
		fmt.Printf("Login failed: %v\n", err)
		return
	}
	resp.Body.Close()

	fmt.Println("=== Testing Session Timeout Warning ===")
	// Test students page for session timeout warning
	resp, err = client.Get("http://127.0.0.1:8080/students")
	if err != nil {
		fmt.Printf("Error accessing students page: %v\n", err)
		return
	}
	
	body, _ := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	bodyStr := string(body)
	
	// Check for session timer in navbar (should not be visible initially)
	if strings.Contains(bodyStr, "session-timer show") {
		fmt.Println("❌ Session timer is showing prematurely")
	} else {
		fmt.Println("✓ Session timer is not showing prematurely")
	}
	
	// Check for session warning modal (should not be visible)
	if strings.Contains(bodyStr, "session-warning-overlay show") {
		fmt.Println("❌ Session warning modal is showing prematurely")
	} else {
		fmt.Println("✓ Session warning modal is not showing prematurely")
	}

	fmt.Println("\n=== Testing Driver Dashboard Performance ===")
	// Test driver dashboard for real performance data
	resp, err = client.Get("http://127.0.0.1:8080/driver-dashboard")
	if err != nil {
		fmt.Printf("Error accessing driver dashboard: %v\n", err)
		return
	}
	
	body, _ = ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	bodyStr = string(body)
	
	// Check for hardcoded test data (should not be present)
	if strings.Contains(bodyStr, ">98%<") {
		fmt.Println("❌ Dashboard still shows hardcoded 98% on-time rate")
	} else {
		fmt.Println("✓ Dashboard does not show hardcoded 98% on-time rate")
	}
	
	if strings.Contains(bodyStr, ">15<") && strings.Contains(bodyStr, "Students Transported") {
		fmt.Println("❌ Dashboard still shows hardcoded 15 students transported")
	} else {
		fmt.Println("✓ Dashboard does not show hardcoded 15 students transported")
	}
	
	if strings.Contains(bodyStr, ">4.8<") && strings.Contains(bodyStr, "Safety Score") {
		fmt.Println("❌ Dashboard still shows hardcoded 4.8 safety score")
	} else {
		fmt.Println("✓ Dashboard does not show hardcoded 4.8 safety score")
	}
	
	// Check for template variables (shows real data is being passed)
	if strings.Contains(bodyStr, ".Data.Performance") {
		fmt.Println("✓ Dashboard has performance data structure")
	} else if strings.Contains(bodyStr, "Today's Performance") {
		fmt.Println("✓ Dashboard shows Today's Performance section")
	}
	
	fmt.Println("\n=== Testing Students Page ===")
	// Test students page again for functionality
	resp, err = client.Get("http://127.0.0.1:8080/students")
	if err != nil {
		fmt.Printf("Error accessing students page: %v\n", err)
		return
	}
	
	body, _ = ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	bodyStr = string(body)
	
	// Check for edit functionality
	if strings.Contains(bodyStr, "editStudentModal") {
		fmt.Println("✓ Edit student modal is present")
	} else {
		fmt.Println("❌ Edit student modal is missing")
	}
	
	// Check for route dropdown
	if strings.Contains(bodyStr, "select") && strings.Contains(bodyStr, "route") {
		fmt.Println("✓ Route dropdown is present")
	} else {
		fmt.Println("❌ Route dropdown is missing")
	}
	
	// Check for location display
	if !strings.Contains(bodyStr, "range can't iterate") {
		fmt.Println("✓ No template iteration errors")
	} else {
		fmt.Println("❌ Template still has iteration errors")
	}
	
	fmt.Println("\n=== All Tests Complete ===")
}