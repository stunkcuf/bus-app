package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"os"
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
		os.Exit(1)
	}
	resp.Body.Close()

	// Test students page
	resp, err = client.Get("http://127.0.0.1:8080/students")
	if err != nil {
		fmt.Printf("Error accessing students page: %v\n", err)
		os.Exit(1)
	}
	
	body, _ := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	
	bodyStr := string(body)
	fmt.Printf("Status: %d\n", resp.StatusCode)
	
	// Check for key indicators
	if strings.Contains(bodyStr, "No Students Added Yet") {
		fmt.Println("❌ Page shows: No Students Added Yet")
	} else {
		fmt.Println("✓ Page does not show 'No Students Added Yet'")
	}
	
	if strings.Contains(bodyStr, "student-card") {
		fmt.Println("✓ Page contains student cards")
	} else {
		fmt.Println("❌ Page does not contain student cards")
	}
	
	// Check for specific students we know exist
	// Try both exact and partial matches
	studentNames := []string{
		"Driver Student One",
		"mychal bert",
		"mychal",
		"bert",
		"Test Student",
		"Student Driver",
		"Student Three",
	}
	
	fmt.Println("\nChecking for known students:")
	foundCount := 0
	for _, name := range studentNames {
		if strings.Contains(bodyStr, name) {
			fmt.Printf("  ✓ Found: %s\n", name)
			foundCount++
		} else {
			fmt.Printf("  ❌ Missing: %s\n", name)
		}
	}
	
	fmt.Printf("\nFound %d out of %d expected students\n", foundCount, len(studentNames))
}