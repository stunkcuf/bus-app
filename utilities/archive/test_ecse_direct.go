package main

import (
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file if it exists
	godotenv.Load("../.env")

	// Get database URL from environment
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable is not set")
	}

	// Connect to database using sqlx
	db, err := sqlx.Connect("postgres", dbURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	fmt.Println("Connected to database successfully!")

	// Test the exact query from loadECSEStudents
	var count int
	err = db.Get(&count, "SELECT COUNT(*) FROM ecse_students")
	if err != nil {
		log.Fatal("Failed to count students:", err)
	}
	
	fmt.Printf("Total ECSE students in database: %d\n", count)
	
	// Try a simple query without struct mapping
	rows, err := db.Query(`
		SELECT student_id, first_name, last_name 
		FROM ecse_students 
		ORDER BY last_name, first_name
		LIMIT 5
	`)
	if err != nil {
		log.Fatal("Failed to query students:", err)
	}
	defer rows.Close()
	
	fmt.Println("\nFirst 5 students:")
	i := 0
	for rows.Next() {
		var studentID, firstName, lastName string
		err := rows.Scan(&studentID, &firstName, &lastName)
		if err != nil {
			log.Printf("Error scanning: %v", err)
			continue
		}
		i++
		fmt.Printf("%d. %s %s (ID: %s)\n", i, firstName, lastName, studentID)
	}
	
	fmt.Printf("\nSuccessfully retrieved %d students\n", i)
}