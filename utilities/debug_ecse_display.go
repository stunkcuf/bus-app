package main

import (
	"database/sql"
	"fmt"
	"log"
	_ "github.com/lib/pq"
)

func main() {
	dbURL := "postgresql://postgres:jTKOWGEzlprRGbkPBCgGxsgnwyLeGoDL@shortline.proxy.rlwy.net:40148/railway"

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer db.Close()

	fmt.Println("=== ECSE DISPLAY DEBUG ===\n")

	// Check if ECSE students exist
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM ecse_students").Scan(&count)
	if err != nil {
		fmt.Printf("Error counting ECSE students: %v\n", err)
		return
	}
	fmt.Printf("Total ECSE students in database: %d\n\n", count)

	// Check first few students
	fmt.Println("Sample ECSE students:")
	rows, err := db.Query(`
		SELECT student_id, first_name, last_name, grade, iep_status
		FROM ecse_students 
		LIMIT 5
	`)
	if err != nil {
		fmt.Printf("Error querying students: %v\n", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var studentID, firstName, lastName string
		var grade, iepStatus sql.NullString
		
		err := rows.Scan(&studentID, &firstName, &lastName, &grade, &iepStatus)
		if err != nil {
			fmt.Printf("Error scanning: %v\n", err)
			continue
		}
		
		gradeStr := "NULL"
		if grade.Valid {
			gradeStr = grade.String
		}
		
		iepStr := "NULL"
		if iepStatus.Valid {
			iepStr = iepStatus.String
		}
		
		fmt.Printf("  %s: %s %s (Grade: %s, IEP: %s)\n", 
			studentID, firstName, lastName, gradeStr, iepStr)
	}

	// Check for any errors in the ECSE table structure
	fmt.Println("\nChecking ECSE table structure:")
	colRows, err := db.Query(`
		SELECT column_name, data_type, is_nullable
		FROM information_schema.columns
		WHERE table_name = 'ecse_students'
		AND column_name IN ('student_id', 'first_name', 'last_name', 'grade', 'iep_status')
		ORDER BY ordinal_position
	`)
	if err == nil {
		defer colRows.Close()
		for colRows.Next() {
			var colName, dataType, nullable string
			colRows.Scan(&colName, &dataType, &nullable)
			fmt.Printf("  %s: %s (nullable: %s)\n", colName, dataType, nullable)
		}
	}

	// Check if there's a route issue
	fmt.Println("\nChecking if manager dashboard links to ECSE:")
	fmt.Println("  Expected route: /ecse-dashboard")
	fmt.Println("  Handler: ecseDashboardHandler")
	fmt.Println("  Template: ecse_dashboard_modern.html")
}