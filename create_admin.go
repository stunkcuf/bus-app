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
	
	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %v", err)
	}
	
	// Check if user already exists
	var exists bool
	err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE username = $1)", username).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check if user exists: %v", err)
	}
	
	if exists {
		// Update existing user
		_, err = db.Exec(`
			UPDATE users 
			SET password = $1, role = 'manager', status = 'active'
			WHERE username = $2
		`, string(hashedPassword), username)
		if err != nil {
			return fmt.Errorf("failed to update admin user: %v", err)
		}
		log.Println("Admin user updated successfully")
		log.Printf("Admin login: username=%s, password=%s", username, password)
	} else {
		// Insert new user
		_, err = db.Exec(`
			INSERT INTO users (username, password, role, status)
			VALUES ($1, $2, 'manager', 'active')
		`, username, string(hashedPassword))
		if err != nil {
			return fmt.Errorf("failed to create admin user: %v", err)
		}
		log.Println("Admin user created successfully")
		log.Printf("Admin login: username=%s, password=%s", username, password)
	}
	
	return nil
}