// migrate_passwords.go - Script to migrate plain text passwords to bcrypt hashes
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
	// Connect to database
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		// Fallback to individual environment variables
		host := os.Getenv("PGHOST")
		port := os.Getenv("PGPORT")
		user := os.Getenv("PGUSER")
		password := os.Getenv("PGPASSWORD")
		dbname := os.Getenv("PGDATABASE")
		
		if host == "" || port == "" || user == "" || password == "" || dbname == "" {
			log.Fatal("Database connection parameters not set")
		}
		
		dbURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=require",
			user, password, host, port, dbname)
	}
	
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()
	
	// Test connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	
	log.Println("Connected to database successfully")
	
	// Get all users
	rows, err := db.Query("SELECT username, password FROM users")
	if err != nil {
		log.Fatalf("Failed to query users: %v", err)
	}
	defer rows.Close()
	
	type UserToMigrate struct {
		Username string
		Password string
	}
	
	var users []UserToMigrate
	for rows.Next() {
		var user UserToMigrate
		if err := rows.Scan(&user.Username, &user.Password); err != nil {
			log.Printf("Error scanning user: %v", err)
			continue
		}
		users = append(users, user)
	}
	
	log.Printf("Found %d users to check", len(users))
	
	// Begin transaction
	tx, err := db.Begin()
	if err != nil {
		log.Fatalf("Failed to begin transaction: %v", err)
	}
	defer tx.Rollback()
	
	migratedCount := 0
	
	for _, user := range users {
		// Check if password is already hashed (bcrypt hashes start with $2a$, $2b$, or $2y$)
		if len(user.Password) > 4 && user.Password[0] == '$' && user.Password[1] == '2' {
			log.Printf("User %s already has hashed password, skipping", user.Username)
			continue
		}
		
		// Hash the plain text password
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), 12)
		if err != nil {
			log.Printf("Failed to hash password for user %s: %v", user.Username, err)
			continue
		}
		
		// Update the user's password
		_, err = tx.Exec("UPDATE users SET password = $1, updated_at = CURRENT_TIMESTAMP WHERE username = $2",
			string(hashedPassword), user.Username)
		if err != nil {
			log.Printf("Failed to update password for user %s: %v", user.Username, err)
			continue
		}
		
		log.Printf("Migrated password for user: %s", user.Username)
		migratedCount++
	}
	
	// Commit transaction
	if err := tx.Commit(); err != nil {
		log.Fatalf("Failed to commit transaction: %v", err)
	}
	
	log.Printf("Successfully migrated %d passwords", migratedCount)
	
	// If no passwords were migrated but there is an admin user, create a default hashed password
	if migratedCount == 0 && len(users) > 0 {
		log.Println("No passwords were migrated. You may need to reset passwords manually.")
		log.Println("Creating a temporary admin user with known password...")
		
		// Create a temporary admin user
		tempPassword := "TempAdmin123!"
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(tempPassword), 12)
		if err != nil {
			log.Fatalf("Failed to hash temp password: %v", err)
		}
		
		_, err = db.Exec(`
			INSERT INTO users (username, password, role) 
			VALUES ('temp_admin', $1, 'manager')
			ON CONFLICT (username) 
			DO UPDATE SET password = $1, updated_at = CURRENT_TIMESTAMP
		`, string(hashedPassword))
		
		if err != nil {
			log.Printf("Failed to create temp admin: %v", err)
		} else {
			log.Println("Created temporary admin user:")
			log.Println("  Username: temp_admin")
			log.Println("  Password: TempAdmin123!")
			log.Println("Please login and reset other user passwords, then delete this temp user.")
		}
	}
}
