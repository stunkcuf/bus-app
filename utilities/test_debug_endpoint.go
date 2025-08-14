package main

import (
	"encoding/json"
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

	// Test debug endpoint
	resp, err = client.Get("http://127.0.0.1:8080/debug-students")
	if err != nil {
		fmt.Printf("Error accessing debug endpoint: %v\n", err)
		return
	}
	
	body, _ := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	
	fmt.Printf("Status: %d\n", resp.StatusCode)
	
	// Parse JSON response
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		fmt.Printf("Failed to parse JSON: %v\n", err)
		fmt.Printf("Raw body:\n%s\n", string(body))
		return
	}
	
	// Pretty print the result
	prettyJSON, _ := json.MarshalIndent(result, "", "  ")
	fmt.Printf("Debug result:\n%s\n", string(prettyJSON))
}