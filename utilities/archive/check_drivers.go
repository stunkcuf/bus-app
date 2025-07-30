package main

import (
	"database/sql"
	"fmt"
	"log"
	
	_ "github.com/lib/pq"
)

func main() {
	// Hardcode the database URL for this utility
	databaseURL := "postgresql://postgres:jTKOWGEzlprRGbkPBCgGxsgnwyLeGoDL@shortline.proxy.rlwy.net:40148/railway"
	
	// Connect to database
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()
	
	// Test connection
	err = db.Ping()
	if err != nil {
		log.Fatal("Failed to ping database:", err)
	}
	
	fmt.Println("Connected to database successfully!")
	fmt.Println("\nChecking for driver accounts...")
	fmt.Println("==================================================")
	
	// Query for driver accounts
	rows, err := db.Query("SELECT username, role, status FROM users WHERE role = 'driver' ORDER BY username")
	if err != nil {
		log.Fatal("Failed to query users:", err)
	}
	defer rows.Close()
	
	count := 0
	fmt.Printf("%-20s %-10s %-10s\n", "Username", "Role", "Status")
	fmt.Println("----------------------------------------")
	
	for rows.Next() {
		var username, role, status string
		err := rows.Scan(&username, &role, &status)
		if err != nil {
			log.Printf("Error scanning row: %v", err)
			continue
		}
		fmt.Printf("%-20s %-10s %-10s\n", username, role, status)
		count++
	}
	
	if err = rows.Err(); err != nil {
		log.Fatal("Error iterating rows:", err)
	}
	
	fmt.Println("----------------------------------------")
	fmt.Printf("\nTotal driver accounts found: %d\n", count)
	
	// If no drivers found, suggest creating one
	if count == 0 {
		fmt.Println("\nNo driver accounts found. You may want to:")
		fmt.Println("1. Register a new driver account through the web interface")
		fmt.Println("2. Create one manually using SQL")
		fmt.Println("\nExample SQL to create a driver account:")
		fmt.Println("INSERT INTO users (username, password, role, status, registration_date, created_at)")
		fmt.Println("VALUES ('testdriver', '$2a$12$[hashed_password]', 'driver', 'active', '2025-01-20', NOW());")
	}
}