package main

import (
	"database/sql"
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	// Load .env file
	godotenv.Load()

	// Get connection string from environment
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		fmt.Println("DATABASE_URL environment variable not set")
		os.Exit(1)
	}

	// Connect to database
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		fmt.Printf("Failed to open database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// Query admin user
	var username, password, role, status string
	err = db.QueryRow("SELECT username, password, role, status FROM users WHERE username = 'admin'").Scan(&username, &password, &role, &status)
	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Println("No admin user found in database")
		} else {
			fmt.Printf("Error querying database: %v\n", err)
		}
		os.Exit(1)
	}

	// Display results
	fmt.Printf("\nAdmin User Details:\n")
	fmt.Printf("Username: %s\n", username)
	fmt.Printf("Role: %s\n", role)
	fmt.Printf("Status: %s\n", status)
	fmt.Printf("\nPassword Field:\n")
	fmt.Printf("Value: %s\n", password)
	fmt.Printf("Length: %d characters\n", len(password))
	
	// Check if it's bcrypt hashed
	if strings.HasPrefix(password, "$2a$") || strings.HasPrefix(password, "$2b$") || strings.HasPrefix(password, "$2y$") {
		fmt.Println("Type: Bcrypt hashed password")
	} else if len(password) == 60 && strings.Contains(password, "$") {
		fmt.Println("Type: Likely bcrypt hashed (based on length and format)")
	} else {
		fmt.Println("Type: Plain text password (NOT HASHED)")
	}
}