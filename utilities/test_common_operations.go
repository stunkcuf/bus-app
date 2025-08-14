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

	fmt.Println("Testing common database operations that might fail...")
	
	// Test 1: Check if students table has grade column (earlier error)
	fmt.Print("\n1. Students table 'grade' column: ")
	var hasGrade bool
	err = db.Get(&hasGrade, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.columns 
			WHERE table_name = 'students' AND column_name = 'grade'
		)
	`)
	if err != nil {
		fmt.Printf("ERROR - %v\n", err)
	} else if !hasGrade {
		fmt.Println("MISSING - This will cause failures when displaying students")
	} else {
		fmt.Println("OK")
	}

	// Test 2: Check if buses table has bus_number column (earlier error)
	fmt.Print("\n2. Buses table 'bus_number' column: ")
	var hasBusNumber bool
	err = db.Get(&hasBusNumber, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.columns 
			WHERE table_name = 'buses' AND column_name = 'bus_number'
		)
	`)
	if err != nil {
		fmt.Printf("ERROR - %v\n", err)
	} else if !hasBusNumber {
		fmt.Println("MISSING - This will cause failures in fleet displays")
	} else {
		fmt.Println("OK")
	}

	// Test 3: Check driver_logs mileage columns
	fmt.Print("\n3. Driver logs mileage columns: ")
	var hasBeginMileage bool
	err = db.Get(&hasBeginMileage, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.columns 
			WHERE table_name = 'driver_logs' AND column_name = 'begin_mileage'
		)
	`)
	if err != nil {
		fmt.Printf("ERROR checking - %v\n", err)
	} else if hasBeginMileage {
		fmt.Println("Has begin_mileage/end_mileage (OLD SCHEMA)")
	} else {
		fmt.Println("Has single 'mileage' column (CURRENT SCHEMA)")
	}

	// Test 4: Try to query students
	fmt.Print("\n4. Query students: ")
	var studentCount int
	err = db.Get(&studentCount, "SELECT COUNT(*) FROM students")
	if err != nil {
		fmt.Printf("ERROR - %v\n", err)
	} else {
		fmt.Printf("OK - %d students found\n", studentCount)
	}

	// Test 5: Try to query buses with correct columns
	fmt.Print("\n5. Query buses: ")
	var busCount int
	err = db.Get(&busCount, "SELECT COUNT(*) FROM buses WHERE status = 'active'")
	if err != nil {
		fmt.Printf("ERROR - %v\n", err)
	} else {
		fmt.Printf("OK - %d active buses found\n", busCount)
	}

	fmt.Println("\n=== SUMMARY ===")
	fmt.Println("If you see MISSING columns above, those will cause page load failures.")
	fmt.Println("The application expects certain columns that don't exist in the database.")
}