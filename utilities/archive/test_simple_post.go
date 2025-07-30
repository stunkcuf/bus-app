package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

func main() {
	// Simple POST with form data
	formData := "username=admin&password=Headstart1"
	
	req, err := http.NewRequest("POST", "http://localhost:5003/", strings.NewReader(formData))
	if err != nil {
		fmt.Printf("Error creating request: %v\n", err)
		return
	}
	
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Content-Length", fmt.Sprintf("%d", len(formData)))
	
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // Don't follow redirects
		},
	}
	
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error sending request: %v\n", err)
		return
	}
	defer resp.Body.Close()
	
	fmt.Printf("Status Code: %d\n", resp.StatusCode)
	fmt.Printf("Location Header: '%s'\n", resp.Header.Get("Location"))
	
	// Print all headers
	fmt.Println("\nResponse Headers:")
	for k, v := range resp.Header {
		fmt.Printf("%s: %v\n", k, v)
	}
	
	// Check body
	body, _ := ioutil.ReadAll(resp.Body)
	bodyStr := string(body)
	
	if len(bodyStr) > 0 {
		fmt.Printf("\nBody length: %d\n", len(bodyStr))
		if strings.Contains(bodyStr, "Invalid") {
			fmt.Println("Found 'Invalid' in response")
		}
		if strings.Contains(bodyStr, "Login") {
			fmt.Println("Found 'Login' in response")
		}
		if strings.Contains(bodyStr, "Dashboard") {
			fmt.Println("Found 'Dashboard' in response")
		}
	}
}