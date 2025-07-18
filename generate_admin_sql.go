package main

import (
	"fmt"
	"log"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	password := "Headstart1"
	
	// Generate bcrypt hash
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal("Failed to hash password:", err)
	}
	
	fmt.Println("=== SQL Script to Create Admin User ===")
	fmt.Println()
	fmt.Println("-- First, check if admin exists:")
	fmt.Println("SELECT username, role, status FROM users WHERE username = 'admin';")
	fmt.Println()
	fmt.Println("-- If admin doesn't exist, run this INSERT:")
	fmt.Printf(`INSERT INTO users (username, password, role, status, registration_date, created_at)
VALUES (
    'admin',
    '%s',
    'manager',
    'active',
    CURRENT_DATE,
    CURRENT_TIMESTAMP
);`, string(hashedPassword))
	fmt.Println()
	fmt.Println()
	fmt.Println("-- If admin exists but you need to reset password, run this UPDATE:")
	fmt.Printf(`UPDATE users 
SET password = '%s',
    role = 'manager',
    status = 'active'
WHERE username = 'admin';`, string(hashedPassword))
	fmt.Println()
	fmt.Println()
	fmt.Println("-- Verify the user was created/updated:")
	fmt.Println("SELECT username, role, status, created_at FROM users WHERE username = 'admin';")
}