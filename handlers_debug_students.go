package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

// debugStudentsHandler is a test endpoint to debug the students loading
func debugStudentsHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil {
		http.Error(w, "Not authenticated", http.StatusUnauthorized)
		return
	}

	log.Printf("DEBUG STUDENTS: Starting for user=%s, role=%s", user.Username, user.Role)
	
	result := map[string]interface{}{
		"user":     user.Username,
		"role":     user.Role,
		"students": []map[string]interface{}{},
	}
	
	// Test direct query
	if user.Role == "driver" {
		var count int
		err := db.Get(&count, "SELECT COUNT(*) FROM students WHERE driver = $1 AND active = true", user.Username)
		result["count_query"] = fmt.Sprintf("count=%d, error=%v", count, err)
		
		// Try simple query
		rows, err := db.Query(`
			SELECT student_id, name, driver, active 
			FROM students 
			WHERE driver = $1 AND active = true
			LIMIT 10
		`, user.Username)
		
		if err != nil {
			result["query_error"] = err.Error()
		} else {
			defer rows.Close()
			
			students := []map[string]interface{}{}
			for rows.Next() {
				var studentID, name, driver string
				var active bool
				if err := rows.Scan(&studentID, &name, &driver, &active); err != nil {
					result["scan_error"] = err.Error()
					break
				}
				students = append(students, map[string]interface{}{
					"student_id": studentID,
					"name":       name,
					"driver":     driver,
					"active":     active,
				})
			}
			result["students"] = students
			result["students_count"] = len(students)
		}
		
		// Also try the complex query with struct
		var studentsStruct []Student
		err2 := db.Select(&studentsStruct, `
			SELECT 
				COALESCE(student_id, '') as student_id,
				COALESCE(name, '') as name,
				COALESCE(locations::text, '[]') as locations,
				COALESCE(phone_number, '') as phone_number,
				COALESCE(alt_phone_number, '') as alt_phone_number,
				COALESCE(guardian, '') as guardian,
				COALESCE(pickup_time::text, '') as pickup_time,
				COALESCE(dropoff_time::text, '') as dropoff_time,
				COALESCE(position_number, 0) as position_number,
				COALESCE(route_id, '') as route_id,
				COALESCE(driver, '') as driver,
				COALESCE(active, false) as active,
				COALESCE(created_at, CURRENT_TIMESTAMP) as created_at
			FROM students 
			WHERE driver = $1 AND active = true
			ORDER BY name
		`, user.Username)
		
		result["struct_query_error"] = fmt.Sprintf("%v", err2)
		result["struct_query_count"] = len(studentsStruct)
		
		// Try without COALESCE
		var studentsSimple []Student
		err3 := db.Select(&studentsSimple, `
			SELECT student_id, name, locations, phone_number, alt_phone_number, 
			       guardian, pickup_time, dropoff_time, position_number, 
			       route_id, driver, active, created_at 
			FROM students 
			WHERE driver = $1 AND active = true
			ORDER BY name
		`, user.Username)
		
		result["simple_query_error"] = fmt.Sprintf("%v", err3)
		result["simple_query_count"] = len(studentsSimple)
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}