package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
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

	// Check if data exists
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM ecse_students").Scan(&count)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	
	fmt.Printf("ECSE students in database: %d\n", count)
	
	// Check a sample record
	var id, firstName, lastName string
	err = db.QueryRow("SELECT student_id, first_name, last_name FROM ecse_students LIMIT 1").
		Scan(&id, &firstName, &lastName)
	
	if err != nil {
		fmt.Printf("Error getting sample: %v\n", err)
	} else {
		fmt.Printf("Sample student: %s - %s %s\n", id, firstName, lastName)
	}
	
	// Test if COALESCE is causing issues
	var testGrade string
	err = db.QueryRow("SELECT COALESCE(grade, '') FROM ecse_students LIMIT 1").Scan(&testGrade)
	if err != nil {
		fmt.Printf("COALESCE test failed: %v\n", err)
	} else {
		fmt.Printf("COALESCE test passed, grade: '%s'\n", testGrade)
	}
}