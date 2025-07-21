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

	// Generate new hash for 'admin' password
	newHash, err := bcrypt.GenerateFromPassword([]byte("admin"), 10)
	if err != nil {
		log.Fatal("Failed to generate hash:", err)
	}

	fmt.Printf("Generated new hash for 'admin': %s\n", string(newHash))

	// Update the admin password
	result, err := db.Exec("UPDATE users SET password = $1 WHERE username = 'admin'", string(newHash))
	if err != nil {
		log.Fatal("Failed to update password:", err)
	}

	rowsAffected, _ := result.RowsAffected()
	fmt.Printf("Rows updated: %d\n", rowsAffected)

	// Verify the update
	var storedHash string
	err = db.QueryRow("SELECT password FROM users WHERE username = 'admin'").Scan(&storedHash)
	if err != nil {
		log.Fatal("Failed to verify update:", err)
	}

	// Test the new password
	err = bcrypt.CompareHashAndPassword([]byte(storedHash), []byte("admin"))
	if err == nil {
		fmt.Println("✅ Password verification successful! You can now login with admin/admin")
	} else {
		fmt.Println("❌ Password verification failed:", err)
	}
}