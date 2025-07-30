package main

import (
	"bufio"
	"database/sql"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"

	_ "github.com/lib/pq"
)

func loadEnv() {
	file, err := os.Open("../.env")
	if err != nil {
		// Try current directory too
		file, err = os.Open(".env")
		if err != nil {
			return
		}
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) == 0 || line[0] == '#' {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			os.Setenv(parts[0], parts[1])
		}
	}
}

func main() {
	// Load .env file
	loadEnv()
	
	// Database checks
	fmt.Println("=== DATABASE CHECKS ===")
	
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL not set")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Check admin user
	var username, role, status string
	err = db.QueryRow("SELECT username, role, status FROM users WHERE username = 'admin'").Scan(&username, &role, &status)
	if err != nil {
		fmt.Printf("Admin user check failed: %v\n", err)
	} else {
		fmt.Printf("Admin user found: username=%s, role=%s, status=%s\n", username, role, status)
	}

	// Check record counts
	tables := []string{"buses", "vehicles", "students", "routes", "maintenance_records"}
	for _, table := range tables {
		var count int
		err = db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&count)
		if err != nil {
			fmt.Printf("Error counting %s: %v\n", table, err)
		} else {
			fmt.Printf("%s table: %d records\n", table, count)
		}
	}

	// HTTP login test
	fmt.Println("\n=== LOGIN TEST ===")
	
	// Create HTTP client with cookie jar
	jar, err := cookiejar.New(nil)
	if err != nil {
		log.Fatal("Failed to create cookie jar:", err)
	}
	client := &http.Client{
		Jar: jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // Don't follow redirects automatically
		},
	}

	// Use port from environment or default to 5000
	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}
	baseURL := "http://localhost:" + port

	// Step 1: Get login page to obtain CSRF token
	resp, err := client.Get(baseURL + "/")
	if err != nil {
		log.Fatal("Failed to get login page:", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	// Extract CSRF token from the form
	csrfToken := ""
	bodyStr := string(body)
	if idx := strings.Index(bodyStr, `name="csrf_token" value="`); idx != -1 {
		start := idx + len(`name="csrf_token" value="`)
		end := strings.Index(bodyStr[start:], `"`)
		if end != -1 {
			csrfToken = bodyStr[start : start+end]
		}
	}
	fmt.Printf("CSRF token extracted: %v\n", csrfToken != "")

	// Step 2: Login with admin credentials
	loginData := url.Values{
		"username":   {"admin"},
		"password":   {"admin"},
		"csrf_token": {csrfToken},
	}

	resp, err = client.PostForm(baseURL+"/", loginData)
	if err != nil {
		log.Fatal("Failed to post login:", err)
	}
	defer resp.Body.Close()

	fmt.Printf("Login response status: %d\n", resp.StatusCode)
	
	// Check if we got a redirect (successful login)
	if resp.StatusCode == 303 || resp.StatusCode == 302 {
		location := resp.Header.Get("Location")
		fmt.Printf("Login successful! Redirected to: %s\n", location)
		
		// Follow the redirect
		resp, err = client.Get(baseURL + location)
		if err != nil {
			log.Fatal("Failed to follow redirect:", err)
		}
		resp.Body.Close()
	} else {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("Login failed. Response body sample: %.200s...\n", string(body))
	}

	// Step 3: Access fleet page
	fmt.Println("\n=== FLEET PAGE TEST ===")
	resp, err = client.Get(baseURL + "/fleet")
	if err != nil {
		log.Fatal("Failed to get fleet page:", err)
	}
	defer resp.Body.Close()

	body, _ = io.ReadAll(resp.Body)
	bodyStr = string(body)

	fmt.Printf("Fleet page status: %d\n", resp.StatusCode)
	
	if resp.StatusCode == 200 {
		// Count occurrences of bus entries (looking for class="bus-container")
		busCount := strings.Count(bodyStr, `class="bus-container"`)
		fmt.Printf("Found %d bus containers on the page\n", busCount)
		
		// Check if the expected text is present
		if strings.Contains(bodyStr, "Fleet Management") {
			fmt.Println("Fleet Management header found ✓")
		}
		
		// Sample some bus IDs
		fmt.Println("\nSample bus IDs found:")
		for i := 0; i < 5 && i < len(bodyStr); i++ {
			if idx := strings.Index(bodyStr, `<h3>Bus `); idx != -1 {
				endIdx := strings.Index(bodyStr[idx:], `</h3>`)
				if endIdx != -1 {
					busInfo := bodyStr[idx : idx+endIdx+5]
					fmt.Printf("  %s\n", busInfo)
					bodyStr = bodyStr[idx+endIdx:]
				}
			}
		}
	} else {
		fmt.Printf("Failed to access fleet page. Sample response: %.200s...\n", bodyStr)
	}

	fmt.Println("\n=== TEST COMPLETE ===")
}