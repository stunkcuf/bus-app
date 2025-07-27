package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"
)

// ecseDashboardHandler shows ECSE student overview
func ecseDashboardHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	
	// Check database connection first
	if db == nil {
		log.Printf("ERROR: Database is nil in ecseDashboardHandler")
		http.Error(w, "Database connection error", http.StatusInternalServerError)
		return
	}
	
	// Test query directly
	var testCount int
	err := db.Get(&testCount, "SELECT COUNT(*) FROM ecse_students")
	if err != nil {
		log.Printf("ERROR: Failed to count ECSE students: %v", err)
	} else {
		log.Printf("Direct query shows %d ECSE students in database", testCount)
	}

	// Load ECSE students
	students, err := loadECSEStudents()
	if err != nil {
		log.Printf("Error loading ECSE students: %v", err)
		students = []ECSEStudent{}
	}
	log.Printf("Loaded %d ECSE students for dashboard", len(students))
	
	// Log sample student for debugging
	if len(students) > 0 {
		log.Printf("Sample student: %+v", students[0])
		log.Printf("Sample student Grade: %s", students[0].GetGrade())
		log.Printf("Sample student IEP Status: %s", students[0].GetIEPStatus())
	}

	// Load ECSE services
	services, err := loadECSEServices()
	if err != nil {
		log.Printf("Error loading ECSE services: %v", err)
		services = []ECSEService{}
	}

	// Calculate statistics
	totalStudents := len(students)
	activeIEPs := 0
	transportationRequired := 0
	upcomingAssessments := 0

	for _, student := range students {
		if student.GetIEPStatus() == "active" {
			activeIEPs++
		}
		if student.GetTransportationRequired() {
			transportationRequired++
		}
	}

	// Calculate upcoming assessments (reviews due in next 30 days)
	if db != nil {
		var count int
		err := db.Get(&count, `
			SELECT COUNT(*) FROM ecse_assessments 
			WHERE next_review_date IS NOT NULL 
			AND next_review_date BETWEEN CURRENT_DATE AND CURRENT_DATE + INTERVAL '30 days'
		`)
		if err == nil {
			upcomingAssessments = count
		} else {
			log.Printf("Error counting upcoming assessments: %v", err)
		}
	}

	// Group services by type
	servicesByType := make(map[string]int)
	for _, service := range services {
		servicesByType[service.ServiceType]++
	}

	data := map[string]interface{}{
		"User":                   user,
		"CSRFToken":              getSessionCSRFToken(r),
		"TotalStudents":          totalStudents,
		"ActiveIEPs":             activeIEPs,
		"TransportationRequired": transportationRequired,
		"UpcomingAssessments":    upcomingAssessments,
		"Students":               students,
		"ServicesByType":         servicesByType,
		"CurrentDate":            time.Now().Format("Monday, January 2, 2006"),
	}
	
	log.Printf("ECSE Dashboard data - Total Students: %d, Students array length: %d", totalStudents, len(students))

	renderTemplate(w, r, "ecse_dashboard.html", data)
}

// ecseStudentDetailsHandler shows individual ECSE student details
func ecseStudentDetailsHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	studentID := r.URL.Query().Get("id")
	if studentID == "" {
		http.Error(w, "Student ID required", http.StatusBadRequest)
		return
	}

	// Load student details
	student, err := getECSEStudent(studentID)
	if err != nil {
		log.Printf("Error loading ECSE student %s: %v", studentID, err)
		http.Error(w, "Student not found", http.StatusNotFound)
		return
	}

	// Load student services
	services, err := getECSEStudentServices(studentID)
	if err != nil {
		log.Printf("Error loading services for student %s: %v", studentID, err)
		services = []ECSEService{}
	}

	// Load assessments
	assessments, err := getECSEStudentAssessments(studentID)
	if err != nil {
		log.Printf("Error loading assessments for student %s: %v", studentID, err)
		assessments = []ECSEAssessment{}
	}

	// Load attendance
	attendance, err := getECSEStudentAttendance(studentID)
	if err != nil {
		log.Printf("Error loading attendance for student %s: %v", studentID, err)
		attendance = []ECSEAttendance{}
	}

	data := map[string]interface{}{
		"User":        user,
		"CSRFToken":   getSessionCSRFToken(r),
		"Student":     student,
		"Services":    services,
		"Assessments": assessments,
		"Attendance":  attendance,
	}

	renderTemplate(w, r, "ecse_student_details.html", data)
}

// Helper functions for ECSE data loading

func loadECSEStudents() ([]ECSEStudent, error) {
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

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

func loadECSEServices() ([]ECSEService, error) {
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var services []ECSEService
	err := db.Select(&services, `
		SELECT * FROM ecse_services 
		WHERE end_date IS NULL OR end_date > CURRENT_DATE
		ORDER BY student_id, service_type
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to load ECSE services: %w", err)
	}

	return services, nil
}

func getECSEStudent(studentID string) (*ECSEStudent, error) {
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var student ECSEStudent
	err := db.Get(&student, "SELECT * FROM ecse_students WHERE student_id = $1", studentID)
	if err != nil {
		return nil, err
	}

	return &student, nil
}

func getECSEStudentServices(studentID string) ([]ECSEService, error) {
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var services []ECSEService
	err := db.Select(&services, `
		SELECT * FROM ecse_services 
		WHERE student_id = $1 
		ORDER BY start_date DESC
	`, studentID)
	if err != nil {
		return nil, err
	}

	return services, nil
}

func getECSEStudentAssessments(studentID string) ([]ECSEAssessment, error) {
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var assessments []ECSEAssessment
	err := db.Select(&assessments, `
		SELECT * FROM ecse_assessments 
		WHERE student_id = $1 
		ORDER BY assessment_date DESC
	`, studentID)
	if err != nil {
		return nil, err
	}

	return assessments, nil
}

func getECSEStudentAttendance(studentID string) ([]ECSEAttendance, error) {
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	var attendance []ECSEAttendance
	err := db.Select(&attendance, `
		SELECT * FROM ecse_attendance 
		WHERE student_id = $1 
		ORDER BY service_date DESC 
		LIMIT 30
	`, studentID)
	if err != nil {
		return nil, err
	}

	return attendance, nil
}

// addECSEServiceHandler adds a new service for an ECSE student
func addECSEServiceHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse form
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	// Validate CSRF
	if !validateCSRF(r) {
		http.Error(w, "Invalid CSRF token", http.StatusForbidden)
		return
	}

	// Extract form data
	studentID := r.FormValue("student_id")
	serviceType := r.FormValue("service_type")
	frequency := r.FormValue("frequency")
	duration, _ := strconv.Atoi(r.FormValue("duration"))
	provider := r.FormValue("provider")
	startDate := r.FormValue("start_date")
	endDate := r.FormValue("end_date")

	// Insert new service
	_, err := db.Exec(`
		INSERT INTO ecse_services 
		(student_id, service_type, frequency, duration, provider, start_date, end_date)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, studentID, serviceType, frequency, duration, provider, startDate, endDate)

	if err != nil {
		log.Printf("Error adding ECSE service: %v", err)
		http.Error(w, "Failed to add service", http.StatusInternalServerError)
		return
	}

	// Redirect back to student details
	http.Redirect(w, r, "/ecse-student?id="+studentID, http.StatusSeeOther)
}

// testECSEHandler is a temporary test endpoint to debug ECSE dashboard
func testECSEHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("Test ECSE handler called")
	
	// Load ECSE students
	students, err := loadECSEStudents()
	if err != nil {
		log.Printf("Error loading ECSE students: %v", err)
		http.Error(w, fmt.Sprintf("Error loading students: %v", err), http.StatusInternalServerError)
		return
	}
	
	log.Printf("Loaded %d ECSE students in test handler", len(students))
	
	// Return simple response
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(w, "ECSE Test Handler\n")
	fmt.Fprintf(w, "================\n\n")
	fmt.Fprintf(w, "Total students loaded: %d\n\n", len(students))
	
	if len(students) > 0 {
		fmt.Fprintf(w, "First 5 students:\n")
		for i, student := range students {
			if i >= 5 {
				break
			}
			fmt.Fprintf(w, "%d. %s %s (ID: %s, Grade: %s)\n", 
				i+1, student.FirstName, student.LastName, 
				student.StudentID, student.GetGrade())
		}
	}
}
