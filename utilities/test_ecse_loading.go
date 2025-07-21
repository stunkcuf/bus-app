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

	// Count ECSE students
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM ecse_students").Scan(&count)
	if err != nil {
		log.Fatalf("Failed to count ECSE students: %v", err)
	}

	fmt.Printf("Total ECSE students in database: %d\n\n", count)

	// Try to load a few students with the actual query used in the app
	rows, err := db.Query(`
		SELECT student_id, first_name, last_name, date_of_birth, grade, 
		       enrollment_status, iep_status, primary_disability, service_minutes,
		       transportation_required, bus_route, parent_name, parent_phone,
		       parent_email, address, city, state, zip_code, notes,
		       created_at, updated_at, import_id
		FROM ecse_students 
		ORDER BY student_id 
		LIMIT 5
	`)
	if err != nil {
		log.Fatalf("Failed to query ECSE students: %v", err)
	}
	defer rows.Close()

	fmt.Println("Sample ECSE students:")
	fmt.Println("====================================================")
	
	rowCount := 0
	for rows.Next() {
		var (
			studentID, firstName, lastName string
			dateOfBirth, grade, enrollmentStatus, iepStatus sql.NullString
			primaryDisability, busRoute, parentName sql.NullString
			parentPhone, parentEmail, address, city sql.NullString
			state, zipCode, notes, importID sql.NullString
			serviceMinutes sql.NullInt32
			transportationRequired sql.NullBool
			createdAt, updatedAt sql.NullTime
		)

		err := rows.Scan(
			&studentID, &firstName, &lastName, &dateOfBirth, &grade,
			&enrollmentStatus, &iepStatus, &primaryDisability, &serviceMinutes,
			&transportationRequired, &busRoute, &parentName, &parentPhone,
			&parentEmail, &address, &city, &state, &zipCode, &notes,
			&createdAt, &updatedAt, &importID,
		)
		if err != nil {
			log.Printf("Error scanning row: %v", err)
			continue
		}

		rowCount++
		fmt.Printf("Student %d:\n", rowCount)
		fmt.Printf("  ID: %s\n", studentID)
		fmt.Printf("  Name: %s %s\n", firstName, lastName)
		if grade.Valid {
			fmt.Printf("  Grade: %s\n", grade.String)
		}
		if iepStatus.Valid {
			fmt.Printf("  IEP Status: %s\n", iepStatus.String)
		}
		if transportationRequired.Valid {
			fmt.Printf("  Transportation: %v\n", transportationRequired.Bool)
		}
		if busRoute.Valid && busRoute.String != "" {
			fmt.Printf("  Bus Route: %s\n", busRoute.String)
		}
		fmt.Println()
	}

	if rowCount == 0 {
		fmt.Println("No ECSE students found!")
	} else {
		fmt.Printf("Successfully loaded %d sample students out of %d total\n", rowCount, count)
	}
}