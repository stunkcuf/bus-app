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

	fmt.Println("Creating test driver account...")

	// Hash the password
	password := "test123"
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		log.Fatalf("Failed to hash password: %v", err)
	}

	// Create test driver
	_, err = db.Exec(`
		INSERT INTO users (username, password, role, status, registration_date, created_at)
		VALUES ($1, $2, 'driver', 'active', CURRENT_DATE, NOW())
		ON CONFLICT (username) DO UPDATE 
		SET password = $2, status = 'active'
	`, "testdriver", string(hash))

	if err != nil {
		log.Fatalf("Failed to create user: %v", err)
	}

	fmt.Println("✅ Created/updated test driver:")
	fmt.Println("   Username: testdriver")
	fmt.Println("   Password: test123")
	fmt.Println("   Role: driver")
	fmt.Println("   Status: active")

	// Also update bjmathis to a known password
	hash2, _ := bcrypt.GenerateFromPassword([]byte("driver123"), 12)
	_, err = db.Exec("UPDATE users SET password = $1 WHERE username = 'bjmathis'", string(hash2))
	if err == nil {
		fmt.Println("\n✅ Updated bjmathis password to: driver123")
	}
}