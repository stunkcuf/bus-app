package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
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

	// Connect to database
	db, err := sql.Open("postgres", dbURL)
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

	// Count total ECSE students
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM ecse_students").Scan(&count)
	if err != nil {
		log.Fatal("Failed to count ECSE students:", err)
	}
	fmt.Printf("Total ECSE students: %d\n", count)

	// Show sample of ECSE students
	rows, err := db.Query(`
		SELECT student_id, first_name, last_name, grade, iep_status, enrollment_status 
		FROM ecse_students 
		LIMIT 5
	`)
	if err != nil {
		log.Fatal("Failed to query ECSE students:", err)
	}
	defer rows.Close()

	fmt.Println("\nSample ECSE students:")
	fmt.Println("----------------------------------------")
	for rows.Next() {
		var studentID, firstName, lastName string
		var grade, iepStatus, enrollmentStatus sql.NullString
		
		err := rows.Scan(&studentID, &firstName, &lastName, &grade, &iepStatus, &enrollmentStatus)
		if err != nil {
			log.Printf("Error scanning row: %v", err)
			continue
		}
		
		gradeStr := "N/A"
		if grade.Valid {
			gradeStr = grade.String
		}
		
		iepStr := "N/A"
		if iepStatus.Valid {
			iepStr = iepStatus.String
		}
		
		enrollStr := "N/A"
		if enrollmentStatus.Valid {
			enrollStr = enrollmentStatus.String
		}
		
		fmt.Printf("ID: %s, Name: %s %s, Grade: %s, IEP: %s, Status: %s\n", 
			studentID, firstName, lastName, gradeStr, iepStr, enrollStr)
	}

	// Check for any NULL student_id values
	var nullCount int
	err = db.QueryRow("SELECT COUNT(*) FROM ecse_students WHERE student_id IS NULL").Scan(&nullCount)
	if err != nil {
		log.Printf("Failed to check NULL student_ids: %v", err)
	} else {
		fmt.Printf("\nStudents with NULL student_id: %d\n", nullCount)
	}

	// Check for empty student_id values
	var emptyCount int
	err = db.QueryRow("SELECT COUNT(*) FROM ecse_students WHERE student_id = ''").Scan(&emptyCount)
	if err != nil {
		log.Printf("Failed to check empty student_ids: %v", err)
	} else {
		fmt.Printf("Students with empty student_id: %d\n", emptyCount)
	}
}