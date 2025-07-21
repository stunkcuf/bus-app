package main

import (
	"database/sql"
	"log"
	"os"

	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	// Get database URL from environment
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	// Connect to database
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Test connection
	if err := db.Ping(); err != nil {
		log.Fatal("Failed to ping database:", err)
	}

	username := "admin"
	password := "Headstart1"

	log.Printf("Ensuring admin user exists with correct password...")

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal("Failed to hash password:", err)
	}

	// Check if user exists
	var exists bool
	err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE username = $1)", username).Scan(&exists)
	if err != nil {
		log.Fatal("Failed to check user existence:", err)
	}

	if exists {
		// Update password and ensure active status
		result, err := db.Exec(`
			UPDATE users 
			SET password = $1, role = 'manager', status = 'active'
			WHERE username = $2
		`, string(hashedPassword), username)
		if err != nil {
			log.Fatal("Failed to update admin user:", err)
		}

		rows, _ := result.RowsAffected()
		log.Printf("Admin user updated (rows affected: %d)", rows)
	} else {
		// Create user
		_, err = db.Exec(`
			INSERT INTO users (username, password, role, status, registration_date)
			VALUES ($1, $2, 'manager', 'active', CURRENT_DATE)
		`, username, string(hashedPassword))
		if err != nil {
			log.Fatal("Failed to create admin user:", err)
		}
		log.Println("Admin user created")
	}

	// Verify the password works
	var storedPassword string
	err = db.QueryRow("SELECT password FROM users WHERE username = $1", username).Scan(&storedPassword)
	if err != nil {
		log.Fatal("Failed to retrieve password:", err)
	}

	err = bcrypt.CompareHashAndPassword([]byte(storedPassword), []byte(password))
	if err != nil {
		log.Fatal("Password verification failed:", err)
	}

	log.Println("✅ Success! Admin user is ready")
	log.Printf("Username: %s", username)
	log.Printf("Password: %s", password)
	log.Println("You can now login at hs-bus.org")
}
