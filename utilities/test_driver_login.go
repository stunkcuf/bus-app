package main

import (
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
	"github.com/joho/godotenv"
)

func main() {
	fmt.Println("🔐 TESTING DRIVER LOGIN")
	fmt.Println("=" + strings.Repeat("=", 60))

	// Load environment
	godotenv.Load("../.env")

	// Connect to database
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL not set")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Get drivers from database
	fmt.Println("\n📋 Finding drivers in database...")
	rows, err := db.Query(`
		SELECT username, role 
		FROM users 
		WHERE role = 'driver' 
		ORDER BY username
	`)
	if err != nil {
		log.Fatal("Failed to query users:", err)
	}
	defer rows.Close()

	var drivers []string
	fmt.Println("\nDrivers found:")
	for rows.Next() {
		var username, role string
		rows.Scan(&username, &role)
		drivers = append(drivers, username)
		fmt.Printf("• %s (role: %s)\n", username, role)
	}

	if len(drivers) == 0 {
		fmt.Println("❌ No drivers found in database")
		return
	}

	// Setup HTTP client
	baseURL := "http://localhost:5003"
	if port := os.Getenv("PORT"); port != "" {
		baseURL = fmt.Sprintf("http://localhost:%s", port)
	}

	// Test login for first driver
	fmt.Printf("\n🧪 Testing login for driver: %s\n", drivers[0])
	
	jar, _ := cookiejar.New(nil)
	client := &http.Client{
		Jar: jar,
	}

	// Try login with common password
	loginData := url.Values{
		"username": {drivers[0]},
		"password": {"password"}, // Common test password
	}

	resp, err := client.PostForm(baseURL+"/", loginData)
	if err != nil {
		log.Printf("Login request failed: %v", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("Login response status: %d\n", resp.StatusCode)
	
	// Check if redirected to dashboard
	if resp.StatusCode == 302 || resp.StatusCode == 303 {
		location := resp.Header.Get("Location")
		fmt.Printf("Redirected to: %s\n", location)
		
		if strings.Contains(location, "dashboard") {
			fmt.Println("✅ Login successful!")
			
			// Now test accessing driver dashboard
			fmt.Println("\n📄 Testing driver dashboard access...")
			dashResp, err := client.Get(baseURL + "/driver-dashboard")
			if err != nil {
				log.Printf("Dashboard request failed: %v", err)
				return
			}
			defer dashResp.Body.Close()
			
			fmt.Printf("Dashboard response status: %d\n", dashResp.StatusCode)
			
			if dashResp.StatusCode == 200 {
				body, _ := io.ReadAll(dashResp.Body)
				fmt.Println("✅ Driver dashboard accessible!")
				
				// Check for key elements
				if strings.Contains(string(body), "Welcome") {
					fmt.Println("✅ Dashboard displays welcome message")
				}
				if strings.Contains(string(body), "Route") {
					fmt.Println("✅ Dashboard displays route information")
				}
			} else {
				fmt.Printf("❌ Driver dashboard returned status %d\n", dashResp.StatusCode)
			}
		}
	} else {
		body, _ := io.ReadAll(resp.Body)
		if strings.Contains(string(body), "Invalid username or password") {
			fmt.Println("❌ Login failed - incorrect password")
			fmt.Println("\nTry creating a test driver with known password")
		}
	}
}