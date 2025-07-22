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

	fmt.Println("Testing ECSE Handler Query")
	fmt.Println("=========================")

	// Test the exact query from the handler
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
	LIMIT 10`
	
	if err := db.Select(&students, query); err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		return
	}
	
	fmt.Printf("✅ Query successful! Found %d students\n", len(students))
	
	for i, s := range students {
		fmt.Printf("\n%d. %s %s (ID: %s)\n", i+1, s.FirstName, s.LastName, s.StudentID)
		fmt.Printf("   Grade: %s, Status: %s\n", s.Grade, s.EnrollmentStatus)
		fmt.Printf("   Transport: %v, Route: %s\n", s.TransportationRequired, s.BusRoute)
	}
}