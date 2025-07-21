package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"

	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	// Get database URL from environment or use default
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		// Check .env file
		data, err := os.ReadFile("../.env")
		if err == nil {
			for _, line := range strings.Split(string(data), "\n") {
				if strings.HasPrefix(line, "DATABASE_URL=") {
					dbURL = strings.TrimPrefix(line, "DATABASE_URL=")
					dbURL = strings.TrimSpace(dbURL)
					dbURL = strings.TrimSuffix(dbURL, "\r")
					break
				}
			}
		}
	}

	if dbURL == "" {
		log.Fatal("DATABASE_URL not set")
	}

	// Connect to database
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Get admin password hash
	var passwordHash string
	err = db.QueryRow("SELECT password FROM users WHERE username = 'admin'").Scan(&passwordHash)
	if err != nil {
		log.Fatal("Failed to get admin password:", err)
	}

	fmt.Printf("Admin password hash: %s\n", passwordHash)

	// Test password verification
	testPasswords := []string{"admin", "Admin", "admin123", "password", "123456"}
	
	fmt.Println("\nTesting password verification:")
	for _, testPass := range testPasswords {
		err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(testPass))
		if err == nil {
			fmt.Printf("✅ Password '%s' matches!\n", testPass)
		} else {
			fmt.Printf("❌ Password '%s' does not match\n", testPass)
		}
	}

	// Also try to generate a new hash for 'admin' to verify
	newHash, err := bcrypt.GenerateFromPassword([]byte("admin"), 10)
	if err != nil {
		log.Fatal("Failed to generate hash:", err)
	}
	fmt.Printf("\nNew hash for 'admin': %s\n", string(newHash))
	
	// Verify the new hash works
	err = bcrypt.CompareHashAndPassword(newHash, []byte("admin"))
	if err == nil {
		fmt.Println("✅ New hash verification successful")
	} else {
		fmt.Println("❌ New hash verification failed")
	}
}