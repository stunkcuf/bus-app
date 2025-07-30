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

	db, err := sqlx.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer db.Close()

	// Count ECSE students
	var count int
	err = db.Get(&count, "SELECT COUNT(*) FROM ecse_students")
	if err != nil {
		fmt.Printf("Error counting: %v\n", err)
		return
	}
	
	fmt.Printf("Total ECSE students: %d\n", count)
	
	// Try the exact query from handler
	type ECSEDisplayStudent struct {
		StudentID              string `db:"student_id"`
		FirstName              string `db:"first_name"`
		LastName               string `db:"last_name"`
		Grade                  string `db:"grade"`
		EnrollmentStatus       string `db:"enrollment_status"`
		IEPStatus              string `db:"iep_status"`
		ServiceCount           int    `db:"service_count"`
		TransportationRequired bool   `db:"transportation_required"`
		BusRoute               string `db:"bus_route"`
		ParentPhone            string `db:"parent_phone"`
	}
	
	var students []ECSEDisplayStudent
	
	query := `SELECT 
		student_id,
		first_name,
		last_name,
		COALESCE(grade, '') as grade,
		COALESCE(enrollment_status, 'Unknown') as enrollment_status,
		COALESCE(iep_status, '') as iep_status,
		0 as service_count,
		COALESCE(transportation_required, false) as transportation_required,
		COALESCE(bus_route, '') as bus_route,
		COALESCE(parent_phone, '') as parent_phone
	FROM ecse_students 
	ORDER BY last_name, first_name
	LIMIT 100`
	
	err = db.Select(&students, query)
	if err != nil {
		fmt.Printf("❌ Query failed: %v\n", err)
		return
	}
	
	fmt.Printf("✅ Query returned %d students\n", len(students))
	
	// Show first 3 students
	for i, s := range students {
		if i >= 3 {
			break
		}
		fmt.Printf("\nStudent %d:\n", i+1)
		fmt.Printf("  ID: %s\n", s.StudentID)
		fmt.Printf("  Name: %s %s\n", s.FirstName, s.LastName)
		fmt.Printf("  Grade: %s\n", s.Grade)
		fmt.Printf("  Enrollment: %s\n", s.EnrollmentStatus)
	}
}