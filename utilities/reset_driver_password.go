package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"

	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	// Get database URL from environment or use default
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		// Check .env file
		data, err := os.ReadFile("../.env")
		if err == nil {
			for _, line := range strings.Split(string(data), "\n") {
				if strings.HasPrefix(line, "DATABASE_URL=") {
					dbURL = strings.TrimPrefix(line, "DATABASE_URL=")
					dbURL = strings.TrimSpace(dbURL)
					dbURL = strings.TrimSuffix(dbURL, "\r")
					break
				}
			}
		}
	}

	if dbURL == "" {
		log.Fatal("DATABASE_URL not set")
	}

	// Connect to database
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Show available driver accounts
	fmt.Println("Available driver accounts:")
	rows, err := db.Query("SELECT username, role, status FROM users WHERE role = 'driver'")
	if err != nil {
		log.Fatal("Failed to query users:", err)
	}
	defer rows.Close()

	var drivers []string
	for rows.Next() {
		var username, role, status string
		rows.Scan(&username, &role, &status)
		fmt.Printf("- %s (status: %s)\n", username, status)
		drivers = append(drivers, username)
	}

	if len(drivers) == 0 {
		fmt.Println("No driver accounts found!")
		return
	}

	// Reset password for first driver
	driverUsername := drivers[0]
	newPassword := "driver123"

	fmt.Printf("\nResetting password for driver: %s\n", driverUsername)
	fmt.Printf("New password will be: %s\n", newPassword)

	// Generate bcrypt hash
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), 10)
	if err != nil {
		log.Fatal("Failed to generate password hash:", err)
	}

	// Update the password
	result, err := db.Exec("UPDATE users SET password = $1 WHERE username = $2", string(hash), driverUsername)
	if err != nil {
		log.Fatal("Failed to update password:", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected > 0 {
		fmt.Printf("\n✅ Password successfully reset for %s\n", driverUsername)
		fmt.Printf("You can now login with:\n")
		fmt.Printf("  Username: %s\n", driverUsername)
		fmt.Printf("  Password: %s\n", newPassword)
	} else {
		fmt.Println("\n❌ No rows updated - user might not exist")
	}

	// Verify the password works
	var storedHash string
	err = db.QueryRow("SELECT password FROM users WHERE username = $1", driverUsername).Scan(&storedHash)
	if err != nil {
		log.Fatal("Failed to verify password:", err)
	}

	err = bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(newPassword))
	if err == nil {
		fmt.Println("\n✅ Password verification successful!")
	} else {
		fmt.Println("\n❌ Password verification failed:", err)
	}
}