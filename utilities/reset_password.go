package main

import (
	"fmt"
	"log"
	"os"

	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	// Load .env file
	godotenv.Load()

	// Get database URL
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL not set")
	}

	// Connect to database
	db, err := sqlx.Connect("postgres", dbURL)
	if err != nil {
		log.Fatal("Failed to connect:", err)
	}
	defer db.Close()

	// Check current password format
	fmt.Println("Checking password format in database...")

	var sample struct {
		Username string `db:"username"`
		Password string `db:"password"`
	}

	err = db.Get(&sample, "SELECT username, password FROM users WHERE username = 'admin'")
	if err != nil {
		log.Fatal("Failed to get admin user:", err)
	}

	fmt.Printf("Admin password hash: %s\n", sample.Password)
	fmt.Printf("Hash length: %d\n", len(sample.Password))

	// Check if it's a bcrypt hash
	if len(sample.Password) == 60 && sample.Password[:4] == "$2a$" {
		fmt.Println("Password appears to be bcrypt encoded")

		// Test with common default password
		testPasswords := []string{"admin", "password", "123456", "admin123"}
		for _, pw := range testPasswords {
			err := bcrypt.CompareHashAndPassword([]byte(sample.Password), []byte(pw))
			if err == nil {
				fmt.Printf("Found working password: %s\n", pw)
				break
			}
		}
	} else {
		fmt.Println("Password is NOT bcrypt encoded - might be plain text or different encoding")
	}

	// Reset admin password to 'admin123'
	fmt.Println("\nResetting admin password to 'admin123'...")

	newPassword := "admin123"
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal("Failed to hash password:", err)
	}

	_, err = db.Exec("UPDATE users SET password = $1 WHERE username = 'admin'", string(hashedPassword))
	if err != nil {
		log.Fatal("Failed to update password:", err)
	}

	fmt.Println("Password reset successful!")
	fmt.Println("\nYou can now login with:")
	fmt.Println("Username: admin")
	fmt.Println("Password: admin123")
}
