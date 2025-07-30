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

	fmt.Println("🔍 Checking Driver Accounts")
	fmt.Println("===========================")

	// Get all driver accounts
	rows, err := db.Query("SELECT username, password, status FROM users WHERE role = 'driver' ORDER BY username")
	if err != nil {
		log.Fatalf("Failed to query drivers: %v", err)
	}
	defer rows.Close()

	fmt.Println("\nDriver Accounts:")
	for rows.Next() {
		var username, password, status string
		err := rows.Scan(&username, &password, &status)
		if err != nil {
			fmt.Printf("Error scanning: %v\n", err)
			continue
		}
		
		// Check if password is hashed
		isHashed := len(password) > 50 && password[:4] == "$2a$"
		
		fmt.Printf("\n• Username: %s\n", username)
		fmt.Printf("  Status: %s\n", status)
		fmt.Printf("  Password hashed: %v\n", isHashed)
		
		if !isHashed {
			fmt.Printf("  ⚠️  Plain text password: %s\n", password)
			
			// Try to hash it
			hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
			if err == nil {
				fmt.Printf("  📝 UPDATE users SET password = '%s' WHERE username = '%s';\n", string(hash), username)
			}
		}
	}

	// Check route assignments
	fmt.Println("\n\nRoute Assignments:")
	rows, err = db.Query("SELECT driver, bus_id, route_id FROM route_assignments ORDER BY driver")
	if err == nil {
		defer rows.Close()
		count := 0
		for rows.Next() {
			var driver, busID, routeID string
			rows.Scan(&driver, &busID, &routeID)
			fmt.Printf("• %s -> Bus: %s, Route: %s\n", driver, busID, routeID)
			count++
		}
		if count == 0 {
			fmt.Println("  ⚠️  No route assignments found")
		}
	}

	// Check students
	fmt.Println("\n\nStudent Assignments:")
	var studentCount int
	err = db.QueryRow("SELECT COUNT(*) FROM students WHERE driver IS NOT NULL AND driver != ''").Scan(&studentCount)
	if err == nil {
		fmt.Printf("• Total students assigned to drivers: %d\n", studentCount)
	}
	
	// Show sample student assignments
	rows, err = db.Query("SELECT driver, COUNT(*) FROM students WHERE driver IS NOT NULL AND driver != '' GROUP BY driver")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var driver string
			var count int
			rows.Scan(&driver, &count)
			fmt.Printf("  - %s: %d students\n", driver, count)
		}
	}
}