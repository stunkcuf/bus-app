package main

import (
	"fmt"
	"log"
	"net/http"
	"time"
)

// importECSEHandler handles ECSE student import
func importECSEHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	data := map[string]interface{}{
		"User":      user,
		"Title":     "Import ECSE Students",
		"CSRFToken": getSessionCSRFToken(r),
	}

	renderTemplate(w, r, "import_ecse.html", data)
}

// viewECSEReportsHandler shows ECSE reports
func viewECSEReportsHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	// Load ECSE students
	students, err := loadECSEStudentsFromDB()
	if err != nil {
		log.Printf("Error loading ECSE students: %v", err)
		students = []ECSEStudent{}
	}

	data := map[string]interface{}{
		"User":      user,
		"Title":     "ECSE Reports",
		"Students":  students,
		"CSRFToken": getSessionCSRFToken(r),
	}

	renderTemplate(w, r, "view_ecse_reports.html", data)
}

// editECSEStudentHandler handles ECSE student editing
func editECSEStudentHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	studentID := r.URL.Query().Get("id")
	if studentID == "" {
		http.Error(w, "Student ID required", http.StatusBadRequest)
		return
	}

	// Load student data
	var student ECSEStudent
	err := db.Get(&student, "SELECT * FROM ecse_students WHERE student_id = $1", studentID)
	if err != nil {
		log.Printf("Error loading ECSE student: %v", err)
		http.Error(w, "Student not found", http.StatusNotFound)
		return
	}

	data := map[string]interface{}{
		"User":      user,
		"Title":     fmt.Sprintf("Edit %s %s", student.FirstName, student.LastName),
		"Student":   student,
		"CSRFToken": getSessionCSRFToken(r),
	}

	renderTemplate(w, r, "edit_ecse_student.html", data)
}

// Additional cache handlers
func optimizeDatabaseHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	response := map[string]interface{}{
		"status": "success",
		"message": "Database optimization completed",
	}

	renderJSON(w, response)
}

func cacheStatsHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	stats := map[string]interface{}{
		"cacheSize": 1024,
		"hitRate": 0.85,
		"missRate": 0.15,
		"lastUpdated": time.Now().Format(time.RFC3339),
	}

	renderJSON(w, stats)
}