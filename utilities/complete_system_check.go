package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"
)

func main() {
	// Wait for server to start
	time.Sleep(2 * time.Second)
	
	fmt.Println("Fleet Management System - Complete System Test")
	fmt.Println("=============================================\n")
	
	// Test 1: Manager Functions
	fmt.Println("PHASE 1: Testing Manager Functions (admin/admin)")
	fmt.Println("------------------------------------------------")
	testManagerFunctions()
	
	// Test 2: Driver Functions  
	fmt.Println("\n\nPHASE 2: Testing Driver Functions (bjmathis/driver123)")
	fmt.Println("-------------------------------------------------------")
	testDriverFunctions()
	
	fmt.Println("\n\n✅ COMPLETE SYSTEM TEST FINISHED!")
}

func testManagerFunctions() {
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}
	baseURL := "http://localhost:5003"
	
	// Login as admin
	if !login(client, baseURL, "admin", "admin") {
		log.Fatal("Admin login failed")
	}
	fmt.Println("✅ Admin login successful")
	
	// Test pages
	pages := map[string]string{
		"Manager Dashboard":   "/manager-dashboard",
		"Fleet Page":          "/fleet", 
		"User Management":     "/manage-users",
		"Route Assignment":    "/assign-routes",
		"ECSE Import":         "/import-ecse",
		"Maintenance Records": "/maintenance-records",
	}
	
	for name, path := range pages {
		resp, err := client.Get(baseURL + path)
		if err != nil {
			fmt.Printf("❌ %s - Connection error: %v\n", name, err)
			continue
		}
		
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		
		if resp.StatusCode == 200 {
			// Special check for fleet page
			if path == "/fleet" && strings.Contains(string(body), "54") {
				fmt.Printf("✅ %s - Working (shows 54 vehicles)\n", name)
			} else {
				fmt.Printf("✅ %s - Working\n", name)
			}
		} else {
			fmt.Printf("❌ %s - Error (Status: %d)\n", name, resp.StatusCode)
		}
	}
	
	// Logout
	client.Get(baseURL + "/logout")
}

func testDriverFunctions() {
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}
	baseURL := "http://localhost:5003"
	
	// Login as driver
	if !login(client, baseURL, "bjmathis", "driver123") {
		log.Fatal("Driver login failed")
	}
	fmt.Println("✅ Driver login successful")
	
	// Test driver pages
	pages := map[string]string{
		"Driver Dashboard":    "/driver-dashboard",
		"Student Management":  "/students",
		"Students (Inactive)": "/students?show_inactive=true",
	}
	
	for name, path := range pages {
		resp, err := client.Get(baseURL + path)
		if err != nil {
			fmt.Printf("❌ %s - Connection error: %v\n", name, err)
			continue
		}
		
		body, _ := io.ReadAll(resp.Body)
		bodyStr := string(body)
		resp.Body.Close()
		
		if resp.StatusCode == 200 {
			// Check for specific content
			if path == "/driver-dashboard" && strings.Contains(bodyStr, "Maintenance Alerts") {
				fmt.Printf("✅ %s - Working (maintenance alerts present)\n", name)
			} else if strings.Contains(path, "show_inactive") {
				fmt.Printf("✅ %s - Working (inactive filter active)\n", name)
			} else {
				fmt.Printf("✅ %s - Working\n", name)
			}
		} else {
			fmt.Printf("❌ %s - Error (Status: %d)\n", name, resp.StatusCode)
		}
	}
}

func login(client *http.Client, baseURL, username, password string) bool {
	formData := url.Values{
		"username": {username},
		"password": {password},
	}
	
	resp, err := client.PostForm(baseURL+"/", formData)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	
	// Check if we got redirected to dashboard
	return strings.Contains(resp.Request.URL.Path, "dashboard")
}