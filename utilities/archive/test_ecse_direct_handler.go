package main

import (
	"fmt"
	"net/http"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"os"
	"log"
)

func main() {
	// Connect to database
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgresql://postgres:jTKOWGEzlprRGbkPBCgGxsgnwyLeGoDL@shortline.proxy.rlwy.net:40148/railway"
	}

	db, err := sqlx.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer db.Close()

	// Define the same struct as in the handler
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
	
	if err := db.Select(&students, query); err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	
	fmt.Printf("Loaded %d students\n", len(students))
	
	// Create a simple test handler
	http.HandleFunc("/test-ecse", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		
		fmt.Fprintf(w, "<html><body><h1>ECSE Test</h1>")
		fmt.Fprintf(w, "<p>Total students: %d</p>", len(students))
		
		if len(students) > 0 {
			fmt.Fprintf(w, "<h2>First 5 students:</h2><ul>")
			for i := 0; i < 5 && i < len(students); i++ {
				s := students[i]
				fmt.Fprintf(w, "<li>%s %s (ID: %s, Grade: %s)</li>", 
					s.FirstName, s.LastName, s.StudentID, s.Grade)
			}
			fmt.Fprintf(w, "</ul>")
		}
		
		// Test the data structure for template
		data := map[string]interface{}{
			"Data": map[string]interface{}{
				"Students": students,
			},
		}
		
		// Check if the structure works
		if dataMap, ok := data["Data"].(map[string]interface{}); ok {
			if studentList, ok := dataMap["Students"].([]ECSEDisplayStudent); ok {
				fmt.Fprintf(w, "<p>Data structure is correct. Students: %d</p>", len(studentList))
			} else {
				fmt.Fprintf(w, "<p>ERROR: Students is not the right type</p>")
			}
		} else {
			fmt.Fprintf(w, "<p>ERROR: Data is not a map</p>")
		}
		
		fmt.Fprintf(w, "</body></html>")
	})
	
	fmt.Println("Starting test server on :8080")
	fmt.Println("Visit http://localhost:8080/test-ecse")
	http.ListenAndServe(":8080", nil)
}