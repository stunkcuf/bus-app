package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

func main() {
	// Test different ways of sending login data
	
	// Method 1: PostForm
	fmt.Println("=== Testing with PostForm ===")
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
	fmt.Printf("Content-Type sent: application/x-www-form-urlencoded\n")
	
	// Method 2: Manual request
	fmt.Println("\n=== Testing with manual request ===")
	data := "username=admin&password=Headstart1"
	req, _ := http.NewRequest("POST", "http://localhost:5003/", strings.NewReader(data))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	
	client := &http.Client{}
	resp2, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer resp2.Body.Close()
	
	fmt.Printf("Status Code: %d\n", resp2.StatusCode)
	fmt.Printf("Location Header: %s\n", resp2.Header.Get("Location"))
	
	// Read body to check for errors
	body, _ := ioutil.ReadAll(resp2.Body)
	if strings.Contains(string(body), "Invalid username or password") {
		fmt.Println("❌ Authentication error message found")
	} else if strings.Contains(string(body), "pending approval") {
		fmt.Println("❌ Account pending approval")
	} else if strings.Contains(string(body), "alert-danger") {
		fmt.Println("❌ Some error occurred")
		// Extract error
		bodyStr := string(body)
		idx := strings.Index(bodyStr, "alert-danger")
		if idx > 0 {
			end := strings.Index(bodyStr[idx:], "</div>")
			if end > 0 && end < 200 {
				fmt.Printf("Error: %s\n", bodyStr[idx:idx+end])
			}
		}
	}
}