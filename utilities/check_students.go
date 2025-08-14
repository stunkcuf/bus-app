package main

import (
	"fmt"
	"log"
	"os"
	_ "github.com/lib/pq"
	"github.com/jmoiron/sqlx"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgresql://postgres:jTKOWGEzlprRGbkPBCgGxsgnwyLeGoDL@shortline.proxy.rlwy.net:40148/railway"
	}

	db, err := sqlx.Connect("postgres", dbURL)
	if err != nil {
		log.Fatal("Failed to connect:", err)
	}
	defer db.Close()

	// Count students
	var count int
	err = db.Get(&count, "SELECT COUNT(*) FROM students")
	if err != nil {
		log.Printf("Error counting students: %v", err)
	} else {
		fmt.Printf("Total students in database: %d\n", count)
	}

	// Get recent students
	var students []struct {
		Name  string `db:"name"`
		Grade string `db:"grade"`
	}
	
	err = db.Select(&students, "SELECT name, grade FROM students ORDER BY id DESC LIMIT 5")
	if err != nil {
		log.Printf("Error getting students: %v", err)
	} else {
		fmt.Println("\nRecent students:")
		for _, s := range students {
			fmt.Printf("  - %s (Grade %s)\n", s.Name, s.Grade)
		}
	}

	// Check for "Test Student"
	var testCount int
	err = db.Get(&testCount, "SELECT COUNT(*) FROM students WHERE name LIKE '%Test%'")
	if err != nil {
		log.Printf("Error checking test student: %v", err)
	} else {
		fmt.Printf("\nStudents with 'Test' in name: %d\n", testCount)
	}
}