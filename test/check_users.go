package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	
	_ "github.com/lib/pq"
)

func main() {
	// Get database URL from environment
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgresql://postgres:jTKOWGEzlprRGbkPBCgGxsgnwyLeGoDL@shortline.proxy.rlwy.net:40148/railway?sslmode=require"
	}
	
	// Connect to database
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()
	
	// Query users
	fmt.Println("=== Users in Database ===")
	rows, err := db.Query(`
		SELECT username, role, status, 
		       CASE WHEN password IS NOT NULL AND password != '' THEN 'SET' ELSE 'NOT SET' END as has_password,
		       created_at
		FROM users 
		ORDER BY role, username
	`)
	if err != nil {
		log.Fatal("Failed to query users:", err)
	}
	defer rows.Close()
	
	fmt.Printf("%-20s %-10s %-10s %-12s %-20s\n", "Username", "Role", "Status", "Has Password", "Created")
	fmt.Println("-----------------------------------------------------------------------")
	
	for rows.Next() {
		var username, role, status, hasPassword string
		var createdAt sql.NullTime
		
		err := rows.Scan(&username, &role, &status, &hasPassword, &createdAt)
		if err != nil {
			log.Printf("Error scanning row: %v", err)
			continue
		}
		
		created := "N/A"
		if createdAt.Valid {
			created = createdAt.Time.Format("2006-01-02 15:04")
		}
		
		fmt.Printf("%-20s %-10s %-10s %-12s %-20s\n", username, role, status, hasPassword, created)
	}
}

