package main

import (
	"fmt"
	"log"
	"golang.org/x/crypto/bcrypt"
)

// CreateAdminUser creates an admin user with specified credentials
// This should be run once to set up the initial admin account
func CreateAdminUser() error {
	username := "admin"
	password := "Headstart1"
	
	log.Println("Checking for admin user...")
	
	// Check if user already exists
	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE username = $1)", username).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check if user exists: %v", err)
	}
	
	if exists {
		log.Println("Admin user already exists - skipping creation")
		// Don't update the password if user already exists
		// This preserves any password changes made in production
		return nil
	}
	
	// Only create if doesn't exist
	log.Println("Admin user not found, creating...")
	
	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %v", err)
	}
	
	// Insert new user
	_, err = db.Exec(`
		INSERT INTO users (username, password, role, status)
		VALUES ($1, $2, 'manager', 'active')
	`, username, string(hashedPassword))
	if err != nil {
		return fmt.Errorf("failed to create admin user: %v", err)
	}
	log.Println("Admin user created successfully")
	log.Printf("Default admin login: username=%s, password=%s", username, password)
	
	return nil
}