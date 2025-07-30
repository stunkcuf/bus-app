package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgresql://postgres:jTKOWGEzlprRGbkPBCgGxsgnwyLeGoDL@shortline.proxy.rlwy.net:40148/railway"
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer db.Close()

	fmt.Println("🔍 Checking Login Issues")
	fmt.Println("=======================")

	// List all users
	fmt.Println("\nAll Users in Database:")
	rows, err := db.Query("SELECT username, role, status FROM users ORDER BY username")
	if err != nil {
		log.Fatalf("Failed to query users: %v", err)
	}
	defer rows.Close()

	fmt.Println("\nUsername        | Role    | Status")
	fmt.Println("----------------|---------|--------")
	for rows.Next() {
		var username, role, status string
		rows.Scan(&username, &role, &status)
		fmt.Printf("%-15s | %-7s | %s\n", username, role, status)
	}

	// Test specific logins
	fmt.Println("\n\nTesting Common Logins:")
	fmt.Println("======================")
	
	testLogins := []struct{
		username string
		password string
	}{
		{"admin", "admin123"},
		{"bjmathis", "driver123"},
		{"testdriver", "test123"},
	}

	for _, test := range testLogins {
		fmt.Printf("\nTesting %s / %s:\n", test.username, test.password)
		
		var storedHash string
		var userStatus string
		err := db.QueryRow("SELECT password, status FROM users WHERE username = $1", test.username).Scan(&storedHash, &userStatus)
		if err != nil {
			fmt.Printf("  ❌ User not found\n")
			continue
		}
		
		fmt.Printf("  ✓ User exists (status: %s)\n", userStatus)
		
		// Check password
		err = bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(test.password))
		if err != nil {
			fmt.Printf("  ❌ Password does not match\n")
		} else {
			fmt.Printf("  ✅ Password is correct\n")
		}
	}
	
	fmt.Println("\n\nIf you're having trouble logging in:")
	fmt.Println("1. Make sure you're using the correct username (case-sensitive)")
	fmt.Println("2. Clear your browser cookies")
	fmt.Println("3. Try an incognito/private browser window")
	fmt.Println("4. Check that the user status is 'active'")
}