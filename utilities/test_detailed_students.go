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
		dbURL = "postgresql://postgres:jTKOWGEzlprRGbkPBCgGxsgnwyLeGoDL@shortline.proxy.rlwy.net:40148/railway"
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("Failed to connect:", err)
	}
	defer db.Close()

	// Test the exact query from the handler
	fmt.Println("Testing exact query from studentsHandler for driver 'test':")
	fmt.Println("===========================================================")
	
	rows, err := db.Query(`
		SELECT student_id, name, locations, phone_number, alt_phone_number, 
		       guardian, pickup_time, dropoff_time, position_number, 
		       route_id, driver, active, created_at 
		FROM students 
		WHERE driver = $1 AND active = true
		ORDER BY name
	`, "test")
	
	if err != nil {
		log.Fatal("Query failed:", err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var studentID, name, phoneNumber, altPhoneNumber, guardian, routeID, driver sql.NullString
		var locations sql.NullString 
		var pickupTime, dropoffTime sql.NullString
		var positionNumber sql.NullInt32
		var active bool
		var createdAt sql.NullTime
		
		err := rows.Scan(&studentID, &name, &locations, &phoneNumber, &altPhoneNumber, 
			&guardian, &pickupTime, &dropoffTime, &positionNumber, 
			&routeID, &driver, &active, &createdAt)
		if err != nil {
			log.Printf("Error scanning row: %v", err)
			continue
		}
		
		count++
		fmt.Printf("%d. ID: %s, Name: %s, Driver: %s, Active: %v\n", 
			count, studentID.String, name.String, driver.String, active)
	}
	
	fmt.Printf("\nTotal rows returned: %d\n", count)
	
	// Now check what the actual Student struct columns are
	fmt.Println("\nChecking column structure of students table:")
	rows2, err := db.Query(`
		SELECT column_name, data_type, is_nullable 
		FROM information_schema.columns 
		WHERE table_name = 'students' 
		ORDER BY ordinal_position
	`)
	if err == nil {
		defer rows2.Close()
		for rows2.Next() {
			var colName, dataType, nullable string
			rows2.Scan(&colName, &dataType, &nullable)
			fmt.Printf("  - %s (%s) nullable=%s\n", colName, dataType, nullable)
		}
	}
}