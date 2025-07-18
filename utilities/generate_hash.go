package main

import (
	"fmt"
	"log"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	// Generate hash for 'admin123'
	password := "admin123"
	
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 12) // Using cost 12 like your existing hash
	if err != nil {
		log.Fatal("Failed to hash password:", err)
	}
	
	fmt.Printf("Password: %s\n", password)
	fmt.Printf("Hash: %s\n", string(hashedPassword))
	fmt.Println("\nSQL to update admin password:")
	fmt.Printf("UPDATE users SET password = '%s' WHERE username = 'admin';\n", string(hashedPassword))
}