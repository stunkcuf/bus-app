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
	// Get database URL from environment or command line
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" && len(os.Args) > 1 {
		dbURL = os.Args[1]
	}
	
	if dbURL == "" {
		log.Fatal("Please provide DATABASE_URL as environment variable or first argument")
	}
	
	log.Printf("Connecting to database...")
	
	// Connect to database
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()
	
	// Test connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	log.Println("Connected successfully!")
	
	// Check if users table exists
	var tableExists bool
	err = db.QueryRow(`
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name = 'users'
		)
	`).Scan(&tableExists)
	if err != nil {
		log.Fatalf("Failed to check if users table exists: %v", err)
	}
	
	if !tableExists {
		log.Println("Users table does not exist! Creating it now...")
		
		// Create users table
		_, err = db.Exec(`
			CREATE TABLE users (
				username VARCHAR(50) PRIMARY KEY,
				password VARCHAR(255) NOT NULL,
				role VARCHAR(20) NOT NULL CHECK (role IN ('manager', 'driver')),
				status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('active', 'pending')),
				registration_date DATE NOT NULL DEFAULT CURRENT_DATE,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			)
		`)
		if err != nil {
			log.Fatalf("Failed to create users table: %v", err)
		}
		log.Println("Users table created successfully!")
	}
	
	// Check if admin exists
	var adminExists bool
	err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE username = 'admin')").Scan(&adminExists)
	if err != nil {
		log.Fatalf("Failed to check if admin exists: %v", err)
	}
	
	if adminExists {
		log.Println("Admin user already exists!")
		
		// Show admin details
		var username, role, status string
		var createdAt sql.NullTime
		err = db.QueryRow("SELECT username, role, status, created_at FROM users WHERE username = 'admin'").
			Scan(&username, &role, &status, &createdAt)
		if err != nil {
			log.Printf("Failed to get admin details: %v", err)
		} else {
			log.Printf("Admin details: username=%s, role=%s, status=%s, created=%v", 
				username, role, status, createdAt.Time)
		}
		
		// Ask if we should reset password
		fmt.Print("\nDo you want to reset the admin password to 'Headstart1'? (yes/no): ")
		var response string
		fmt.Scanln(&response)
		
		if response == "yes" {
			// Hash the password
			hashedPassword, err := bcrypt.GenerateFromPassword([]byte("Headstart1"), bcrypt.DefaultCost)
			if err != nil {
				log.Fatalf("Failed to hash password: %v", err)
			}
			
			// Update password
			_, err = db.Exec("UPDATE users SET password = $1 WHERE username = 'admin'", string(hashedPassword))
			if err != nil {
				log.Fatalf("Failed to update password: %v", err)
			}
			log.Println("Admin password reset successfully!")
		}
	} else {
		log.Println("Admin user does not exist. Creating now...")
		
		// Hash the password
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte("Headstart1"), bcrypt.DefaultCost)
		if err != nil {
			log.Fatalf("Failed to hash password: %v", err)
		}
		
		// Insert admin user
		_, err = db.Exec(`
			INSERT INTO users (username, password, role, status, registration_date)
			VALUES ('admin', $1, 'manager', 'active', CURRENT_DATE)
		`, string(hashedPassword))
		if err != nil {
			log.Fatalf("Failed to create admin user: %v", err)
		}
		log.Println("Admin user created successfully!")
		log.Println("Login credentials: username=admin, password=Headstart1")
	}
	
	// Show all users
	log.Println("\nAll users in database:")
	rows, err := db.Query("SELECT username, role, status, created_at FROM users ORDER BY created_at")
	if err != nil {
		log.Printf("Failed to list users: %v", err)
	} else {
		defer rows.Close()
		for rows.Next() {
			var username, role, status string
			var createdAt sql.NullTime
			if err := rows.Scan(&username, &role, &status, &createdAt); err == nil {
				log.Printf("  - %s (role=%s, status=%s, created=%v)", 
					username, role, status, createdAt.Time)
			}
		}
	}
}