package main

import (
	"database/sql"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
)

func main() {
	// Get database URL from environment or use default
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		// Try Railway environment
		host := os.Getenv("PGHOST")
		port := os.Getenv("PGPORT")
		user := os.Getenv("PGUSER")
		password := os.Getenv("PGPASSWORD")
		database := os.Getenv("PGDATABASE")
		
		if host != "" && user != "" {
			dbURL = "postgres://" + user + ":" + password + "@" + host + ":" + port + "/" + database + "?sslmode=require"
		} else {
			// Use the same default as main.go - without SSL
			dbURL = "postgres://postgres:Savage1995!@viaduct.proxy.rlwy.net:51688/railway?sslmode=disable"
		}
	}

	// Connect to database
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Test connection
	err = db.Ping()
	if err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	log.Println("Connected to database successfully")

	// Sample ECSE students data
	students := []struct {
		ID                     string
		FirstName              string
		LastName               string
		DOB                    string
		Grade                  string
		EnrollmentStatus       string
		IEPStatus              string
		PrimaryDisability      string
		ServiceMinutes         int
		TransportationRequired bool
		BusRoute               string
		ParentName             string
		ParentPhone            string
		ParentEmail            string
		City                   string
		State                  string
		ZipCode                string
		Notes                  string
	}{
		{
			ID: "ECSE001", FirstName: "Emma", LastName: "Johnson", DOB: "2019-03-15",
			Grade: "Pre-K", EnrollmentStatus: "active", IEPStatus: "active",
			PrimaryDisability: "Speech/Language Impairment", ServiceMinutes: 120,
			TransportationRequired: true, BusRoute: "ECSE-1", ParentName: "Sarah Johnson",
			ParentPhone: "555-0101", ParentEmail: "sarah.johnson@email.com",
			City: "Springfield", State: "IL", ZipCode: "62701",
			Notes: "Requires speech therapy 2x per week",
		},
		{
			ID: "ECSE002", FirstName: "Liam", LastName: "Smith", DOB: "2018-11-22",
			Grade: "Pre-K", EnrollmentStatus: "active", IEPStatus: "active",
			PrimaryDisability: "Autism Spectrum Disorder", ServiceMinutes: 180,
			TransportationRequired: true, BusRoute: "ECSE-1", ParentName: "Michael Smith",
			ParentPhone: "555-0102", ParentEmail: "m.smith@email.com",
			City: "Springfield", State: "IL", ZipCode: "62702",
			Notes: "Behavioral support plan in place",
		},
		{
			ID: "ECSE003", FirstName: "Sophia", LastName: "Williams", DOB: "2019-07-08",
			Grade: "Pre-K", EnrollmentStatus: "active", IEPStatus: "active",
			PrimaryDisability: "Developmental Delay", ServiceMinutes: 150,
			TransportationRequired: true, BusRoute: "ECSE-2", ParentName: "Jennifer Williams",
			ParentPhone: "555-0103", ParentEmail: "j.williams@email.com",
			City: "Springfield", State: "IL", ZipCode: "62703",
			Notes: "OT and PT services required",
		},
		{
			ID: "ECSE004", FirstName: "Noah", LastName: "Brown", DOB: "2019-01-30",
			Grade: "Pre-K", EnrollmentStatus: "active", IEPStatus: "evaluation",
			PrimaryDisability: "Other Health Impairment", ServiceMinutes: 90,
			TransportationRequired: false, BusRoute: "", ParentName: "David Brown",
			ParentPhone: "555-0104", ParentEmail: "d.brown@email.com",
			City: "Springfield", State: "IL", ZipCode: "62704",
			Notes: "Parent provides transportation",
		},
		{
			ID: "ECSE005", FirstName: "Olivia", LastName: "Davis", DOB: "2018-09-12",
			Grade: "Kindergarten", EnrollmentStatus: "active", IEPStatus: "active",
			PrimaryDisability: "Multiple Disabilities", ServiceMinutes: 240,
			TransportationRequired: true, BusRoute: "ECSE-1", ParentName: "Amanda Davis",
			ParentPhone: "555-0105", ParentEmail: "a.davis@email.com",
			City: "Springfield", State: "IL", ZipCode: "62705",
			Notes: "Requires full-time aide support",
		},
		{
			ID: "ECSE006", FirstName: "Ethan", LastName: "Miller", DOB: "2019-05-20",
			Grade: "Pre-K", EnrollmentStatus: "active", IEPStatus: "active",
			PrimaryDisability: "Intellectual Disability", ServiceMinutes: 180,
			TransportationRequired: true, BusRoute: "ECSE-2", ParentName: "Lisa Miller",
			ParentPhone: "555-0106", ParentEmail: "l.miller@email.com",
			City: "Springfield", State: "IL", ZipCode: "62706",
			Notes: "Adaptive PE services included",
		},
		{
			ID: "ECSE007", FirstName: "Ava", LastName: "Wilson", DOB: "2019-02-14",
			Grade: "Pre-K", EnrollmentStatus: "active", IEPStatus: "active",
			PrimaryDisability: "Hearing Impairment", ServiceMinutes: 120,
			TransportationRequired: true, BusRoute: "ECSE-1", ParentName: "Robert Wilson",
			ParentPhone: "555-0107", ParentEmail: "r.wilson@email.com",
			City: "Springfield", State: "IL", ZipCode: "62707",
			Notes: "Uses hearing aids, requires FM system",
		},
		{
			ID: "ECSE008", FirstName: "Mason", LastName: "Garcia", DOB: "2018-12-05",
			Grade: "Kindergarten", EnrollmentStatus: "active", IEPStatus: "active",
			PrimaryDisability: "Visual Impairment", ServiceMinutes: 150,
			TransportationRequired: true, BusRoute: "ECSE-2", ParentName: "Maria Garcia",
			ParentPhone: "555-0108", ParentEmail: "m.garcia@email.com",
			City: "Springfield", State: "IL", ZipCode: "62708",
			Notes: "Requires large print materials",
		},
	}

	// Insert students
	for _, s := range students {
		_, err := db.Exec(`
			INSERT INTO ecse_students (
				student_id, first_name, last_name, date_of_birth, grade,
				enrollment_status, iep_status, primary_disability, service_minutes,
				transportation_required, bus_route, parent_name, parent_phone,
				parent_email, city, state, zip_code, notes, created_at
			) VALUES (
				$1, $2, $3, $4::DATE, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19
			) ON CONFLICT (student_id) DO UPDATE SET
				first_name = EXCLUDED.first_name,
				last_name = EXCLUDED.last_name,
				grade = EXCLUDED.grade,
				enrollment_status = EXCLUDED.enrollment_status,
				iep_status = EXCLUDED.iep_status,
				transportation_required = EXCLUDED.transportation_required,
				bus_route = EXCLUDED.bus_route
		`, s.ID, s.FirstName, s.LastName, s.DOB, s.Grade, s.EnrollmentStatus,
			s.IEPStatus, s.PrimaryDisability, s.ServiceMinutes, s.TransportationRequired,
			s.BusRoute, s.ParentName, s.ParentPhone, s.ParentEmail, s.City, s.State,
			s.ZipCode, s.Notes, time.Now())

		if err != nil {
			log.Printf("Error inserting student %s: %v", s.ID, err)
		} else {
			log.Printf("Successfully added/updated ECSE student: %s %s", s.FirstName, s.LastName)
		}
	}

	// Add sample services for each student
	services := []struct {
		StudentID   string
		ServiceType string
		Frequency   string
		Duration    int
		Provider    string
		StartDate   string
	}{
		{"ECSE001", "speech", "2x per week", 30, "Ms. Thompson", "2024-08-01"},
		{"ECSE002", "behavioral", "Daily", 60, "Mr. Rodriguez", "2024-08-01"},
		{"ECSE002", "speech", "3x per week", 30, "Ms. Thompson", "2024-08-01"},
		{"ECSE003", "OT", "2x per week", 45, "Ms. Chen", "2024-08-01"},
		{"ECSE003", "PT", "1x per week", 45, "Mr. Johnson", "2024-08-01"},
		{"ECSE005", "speech", "Daily", 30, "Ms. Thompson", "2024-08-01"},
		{"ECSE005", "OT", "3x per week", 45, "Ms. Chen", "2024-08-01"},
		{"ECSE005", "PT", "2x per week", 45, "Mr. Johnson", "2024-08-01"},
		{"ECSE006", "other", "Adaptive PE 2x per week", 30, "Coach Williams", "2024-08-01"},
		{"ECSE007", "speech", "3x per week", 45, "Ms. Davis (ASL)", "2024-08-01"},
		{"ECSE008", "other", "Vision services", 60, "Ms. Anderson", "2024-08-01"},
	}

	// Insert services
	for _, srv := range services {
		_, err := db.Exec(`
			INSERT INTO ecse_services (
				student_id, service_type, frequency, duration, provider, start_date
			) VALUES ($1, $2, $3, $4, $5, $6::DATE)
			ON CONFLICT DO NOTHING
		`, srv.StudentID, srv.ServiceType, srv.Frequency, srv.Duration, srv.Provider, srv.StartDate)

		if err != nil {
			log.Printf("Error inserting service for %s: %v", srv.StudentID, err)
		}
	}

	// Add sample assessments
	assessments := []struct {
		StudentID      string
		Date           string
		Type           string
		Results        string
		Evaluator      string
		NextAssessment string
	}{
		{"ECSE001", "2024-09-15", "Speech/Language Evaluation", "Moderate delays in expressive language", "Ms. Thompson", "2025-09-15"},
		{"ECSE002", "2024-09-10", "Comprehensive Evaluation", "Autism diagnosis confirmed, requires intensive support", "Dr. Martinez", "2025-03-10"},
		{"ECSE003", "2024-09-20", "Developmental Assessment", "Global developmental delays, making progress", "Dr. Lee", "2025-03-20"},
		{"ECSE005", "2024-09-05", "Triennial Review", "Continues to qualify for services in all areas", "IEP Team", "2027-09-05"},
	}

	for _, a := range assessments {
		_, err := db.Exec(`
			INSERT INTO ecse_assessments (
				student_id, assessment_date, assessment_type, results, evaluator, next_assessment_date
			) VALUES ($1, $2::DATE, $3, $4, $5, $6::DATE)
			ON CONFLICT DO NOTHING
		`, a.StudentID, a.Date, a.Type, a.Results, a.Evaluator, a.NextAssessment)

		if err != nil {
			log.Printf("Error inserting assessment for %s: %v", a.StudentID, err)
		}
	}

	// Add sample attendance records
	today := time.Now()
	for i := 0; i < 5; i++ {
		date := today.AddDate(0, 0, -i).Format("2006-01-02")
		for _, s := range students {
			if s.TransportationRequired {
				status := "present"
				if i == 3 && s.ID == "ECSE002" {
					status = "absent"
				}
				_, err := db.Exec(`
					INSERT INTO ecse_attendance (
						student_id, date, status, arrival_time, departure_time
					) VALUES ($1, $2::DATE, $3, '08:30', '14:30')
					ON CONFLICT (student_id, date) DO NOTHING
				`, s.ID, date, status)

				if err != nil {
					log.Printf("Error inserting attendance: %v", err)
				}
			}
		}
	}

	log.Println("Sample ECSE data added successfully!")
}