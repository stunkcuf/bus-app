package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
	"github.com/joho/godotenv"
)

// ECSEStudent model (simplified)
type ECSEStudent struct {
	StudentID              string         `json:"student_id" db:"student_id"`
	FirstName              string         `json:"first_name" db:"first_name"`
	LastName               string         `json:"last_name" db:"last_name"`
	DateOfBirth            sql.NullString `json:"date_of_birth" db:"date_of_birth"`
	Grade                  sql.NullString `json:"grade" db:"grade"`
	EnrollmentStatus       sql.NullString `json:"enrollment_status" db:"enrollment_status"`
	IEPStatus              sql.NullString `json:"iep_status" db:"iep_status"`
	PrimaryDisability      sql.NullString `json:"primary_disability" db:"primary_disability"`
	ServiceMinutes         sql.NullInt32  `json:"service_minutes" db:"service_minutes"`
	TransportationRequired sql.NullBool   `json:"transportation_required" db:"transportation_required"`
	BusRoute               sql.NullString `json:"bus_route" db:"bus_route"`
	ParentName             sql.NullString `json:"parent_name" db:"parent_name"`
	ParentPhone            sql.NullString `json:"parent_phone" db:"parent_phone"`
	ParentEmail            sql.NullString `json:"parent_email" db:"parent_email"`
}

// Helper methods
func (e ECSEStudent) GetGrade() string {
	if e.Grade.Valid {
		return e.Grade.String
	}
	return ""
}

func (e ECSEStudent) GetIEPStatus() string {
	if e.IEPStatus.Valid {
		return e.IEPStatus.String
	}
	return ""
}

func (e ECSEStudent) IsTransportationRequired() bool {
	if e.TransportationRequired.Valid {
		return e.TransportationRequired.Bool
	}
	return false
}

func (e ECSEStudent) GetServiceMinutes() int {
	if e.ServiceMinutes.Valid {
		return int(e.ServiceMinutes.Int32)
	}
	return 0
}

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

	// Test loadECSEStudents function
	fmt.Println("\nTesting loadECSEStudents function...")
	start := time.Now()
	
	rows, err := db.Query(`
		SELECT student_id, first_name, last_name, date_of_birth, grade, enrollment_status, 
		       iep_status, primary_disability, service_minutes, transportation_required, 
		       bus_route, parent_name, parent_phone, parent_email
		FROM ecse_students 
		ORDER BY last_name, first_name
	`)
	if err != nil {
		log.Fatal("Failed to query ECSE students:", err)
	}
	defer rows.Close()

	var students []ECSEStudent
	for rows.Next() {
		var student ECSEStudent
		err := rows.Scan(
			&student.StudentID, &student.FirstName, &student.LastName,
			&student.DateOfBirth, &student.Grade, &student.EnrollmentStatus,
			&student.IEPStatus, &student.PrimaryDisability, &student.ServiceMinutes,
			&student.TransportationRequired, &student.BusRoute, &student.ParentName,
			&student.ParentPhone, &student.ParentEmail,
		)
		if err != nil {
			log.Printf("Error scanning row: %v", err)
			continue
		}
		students = append(students, student)
	}

	elapsed := time.Since(start)
	fmt.Printf("Loaded %d ECSE students in %v\n", len(students), elapsed)

	// Calculate statistics
	totalStudents := len(students)
	activeIEPs := 0
	transportationRequired := 0

	for _, student := range students {
		if student.GetIEPStatus() == "active" {
			activeIEPs++
		}
		if student.IsTransportationRequired() {
			transportationRequired++
		}
	}

	fmt.Printf("\nStatistics:\n")
	fmt.Printf("Total Students: %d\n", totalStudents)
	fmt.Printf("Active IEPs: %d\n", activeIEPs)
	fmt.Printf("Transportation Required: %d\n", transportationRequired)

	// Show sample of students
	fmt.Println("\nSample students (first 5):")
	for i, student := range students {
		if i >= 5 {
			break
		}
		fmt.Printf("%d. %s %s - Grade: %s, IEP: %s, Services: %d min, Transport: %v\n",
			i+1, student.FirstName, student.LastName, 
			student.GetGrade(), student.GetIEPStatus(), 
			student.GetServiceMinutes(), student.IsTransportationRequired())
	}
}