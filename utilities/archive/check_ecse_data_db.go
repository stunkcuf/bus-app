package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
	
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

	fmt.Println("🔍 Checking ECSE Data")
	fmt.Println("=" + strings.Repeat("=", 50))

	// Check table structure
	fmt.Println("\n1. ECSE Students Table Structure:")
	rows, err := db.Query(`
		SELECT column_name, data_type, is_nullable
		FROM information_schema.columns
		WHERE table_name = 'ecse_students'
		ORDER BY ordinal_position
	`)
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
	} else {
		defer rows.Close()
		for rows.Next() {
			var col, dtype, nullable string
			rows.Scan(&col, &dtype, &nullable)
			fmt.Printf("  - %s (%s) %s\n", col, dtype, nullable)
		}
	}

	// Count records
	fmt.Println("\n2. ECSE Students Count:")
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM ecse_students").Scan(&count)
	if err != nil {
		fmt.Printf("❌ Error counting: %v\n", err)
	} else {
		fmt.Printf("✅ Total ECSE students: %d\n", count)
	}

	// Show sample data
	if count > 0 {
		fmt.Println("\n3. Sample ECSE Students:")
		rows, err := db.Query(`
			SELECT student_id, first_name, last_name, grade, enrollment_status, 
			       transportation_required, iep_status
			FROM ecse_students 
			LIMIT 5
		`)
		if err != nil {
			fmt.Printf("❌ Error loading samples: %v\n", err)
		} else {
			defer rows.Close()
			for rows.Next() {
				var studentID, firstName, lastName, grade, enrollment string
				var transport bool
				var iepStatus sql.NullString
				
				err := rows.Scan(&studentID, &firstName, &lastName, &grade, 
					&enrollment, &transport, &iepStatus)
				if err != nil {
					fmt.Printf("  ❌ Scan error: %v\n", err)
					continue
				}
				
				iep := "None"
				if iepStatus.Valid {
					iep = iepStatus.String
				}
				
				fmt.Printf("  • %s: %s %s (Grade %s) - Status: %s, Transport: %v, IEP: %s\n",
					studentID, firstName, lastName, grade, enrollment, transport, iep)
			}
		}
	}

	// Check for column case sensitivity issues
	fmt.Println("\n4. Column Name Check:")
	rows, err = db.Query(`
		SELECT column_name 
		FROM information_schema.columns 
		WHERE table_name = 'ecse_students' 
		AND column_name IN ('student_id', 'studentid', 'StudentID')
	`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var col string
			rows.Scan(&col)
			fmt.Printf("  Found ID column: %s\n", col)
		}
	}

	// Test the actual query the handler uses
	fmt.Println("\n5. Testing Handler Query:")
	rows, err = db.Query("SELECT * FROM ecse_students ORDER BY last_name, first_name LIMIT 3")
	if err != nil {
		fmt.Printf("❌ Handler query failed: %v\n", err)
	} else {
		defer rows.Close()
		cols, _ := rows.Columns()
		fmt.Printf("✅ Query successful, columns: %v\n", cols)
	}
}