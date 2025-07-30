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
	
	fmt.Println("=== Checking User Statuses ===\n")
	
	// Check all users and their statuses
	rows, err := db.Query(`
		SELECT username, role, status, created_at
		FROM users
		ORDER BY status, created_at DESC
	`)
	if err != nil {
		log.Fatal("Failed to query users:", err)
	}
	defer rows.Close()
	
	pendingCount := 0
	activeCount := 0
	
	fmt.Println("All Users:")
	fmt.Println("----------")
	for rows.Next() {
		var username, role, status string
		var createdAt sql.NullTime
		
		err := rows.Scan(&username, &role, &status, &createdAt)
		if err != nil {
			log.Printf("Error scanning row: %v", err)
			continue
		}
		
		fmt.Printf("%-20s %-10s %-10s %s\n", username, role, status, createdAt.Time.Format("2006-01-02"))
		
		if status == "pending" {
			pendingCount++
		} else if status == "active" {
			activeCount++
		}
	}
	
	fmt.Printf("\n\nSummary:\n")
	fmt.Printf("--------\n")
	fmt.Printf("Active users:  %d\n", activeCount)
	fmt.Printf("Pending users: %d\n", pendingCount)
	
	// Check the specific pending approval query
	fmt.Println("\n\nPending Approval Query Result:")
	fmt.Println("------------------------------")
	rows2, err := db.Query(`
		SELECT username, role, created_at 
		FROM users 
		WHERE status = 'pending' 
		ORDER BY created_at DESC
	`)
	if err != nil {
		log.Fatal("Failed to query pending users:", err)
	}
	defer rows2.Close()
	
	count := 0
	for rows2.Next() {
		var username, role string
		var createdAt sql.NullTime
		
		err := rows2.Scan(&username, &role, &createdAt)
		if err != nil {
			log.Printf("Error scanning row: %v", err)
			continue
		}
		
		count++
		fmt.Printf("%d. %-20s %-10s %s\n", count, username, role, createdAt.Time.Format("2006-01-02 15:04:05"))
	}
	
	if count == 0 {
		fmt.Println("No pending users found in the approval query")
	}
}