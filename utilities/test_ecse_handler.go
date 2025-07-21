package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
)

// ECSEStudent model
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

func loadECSEStudents(db *sqlx.DB) ([]ECSEStudent, error) {
	var students []ECSEStudent
	err := db.Select(&students, `
		SELECT * FROM ecse_students 
		ORDER BY last_name, first_name
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to load ECSE students: %w", err)
	}
	return students, nil
}

func main() {
	// Load .env file if it exists
	godotenv.Load("../.env")

	// Get database URL from environment
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable is not set")
	}

	// Connect to database using sqlx
	db, err := sqlx.Connect("postgres", dbURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	fmt.Println("Connected to database successfully!")

	// Test the same function as the handler
	students, err := loadECSEStudents(db)
	if err != nil {
		log.Printf("Error loading ECSE students: %v", err)
		students = []ECSEStudent{}
	}
	
	fmt.Printf("\nLoaded %d ECSE students\n", len(students))
	
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

	fmt.Printf("\nStatistics (same as dashboard):\n")
	fmt.Printf("TotalStudents: %d\n", totalStudents)
	fmt.Printf("ActiveIEPs: %d\n", activeIEPs)  
	fmt.Printf("TransportationRequired: %d\n", transportationRequired)
	
	// Show what would be in the template data
	fmt.Printf("\nTemplate data would contain:\n")
	fmt.Printf("- Students array with %d items\n", len(students))
	if len(students) > 0 {
		fmt.Printf("- First student: %s %s\n", students[0].FirstName, students[0].LastName)
	}
}