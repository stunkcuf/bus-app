package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// Global search handler
func globalSearchHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	query := r.URL.Query().Get("q")
	if query == "" {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"results": []interface{}{},
			"count":   0,
		})
		return
	}

	results := performGlobalSearch(query, user.Role)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func performGlobalSearch(query string, role string) map[string]interface{} {
	searchPattern := "%" + strings.ToLower(query) + "%"
	results := map[string]interface{}{
		"students": []map[string]interface{}{},
		"buses":    []map[string]interface{}{},
		"routes":   []map[string]interface{}{},
		"drivers":  []map[string]interface{}{},
	}

	// Search students
	studentRows, err := db.Query(`
		SELECT student_id, name, locations, phone_number
		FROM students
		WHERE LOWER(name) LIKE $1 
		   OR LOWER(student_id) LIKE $1
		   OR LOWER(locations) LIKE $1
		LIMIT 5
	`, searchPattern)
	
	if err == nil {
		defer studentRows.Close()
		students := []map[string]interface{}{}
		for studentRows.Next() {
			var s Student
			studentRows.Scan(&s.StudentID, &s.Name, &s.Locations, &s.PhoneNumber)
			students = append(students, map[string]interface{}{
				"id":    s.StudentID,
				"name":  s.Name,
				"type":  "student",
				"link":  fmt.Sprintf("/students?search=%s", s.StudentID),
			})
		}
		results["students"] = students
	}

	// Search buses
	busRows, err := db.Query(`
		SELECT bus_number, capacity, status
		FROM buses
		WHERE LOWER(bus_number) LIKE $1
		   OR LOWER(status) LIKE $1
		LIMIT 5
	`, searchPattern)
	
	if err == nil {
		defer busRows.Close()
		buses := []map[string]interface{}{}
		for busRows.Next() {
			var busNumber, status string
			var capacity int
			busRows.Scan(&busNumber, &capacity, &status)
			buses = append(buses, map[string]interface{}{
				"id":     busNumber,
				"name":   fmt.Sprintf("Bus %s", busNumber),
				"type":   "bus",
				"status": status,
				"link":   fmt.Sprintf("/fleet?bus=%s", busNumber),
			})
		}
		results["buses"] = buses
	}

	// Search routes
	routeRows, err := db.Query(`
		SELECT id, name, description
		FROM routes
		WHERE LOWER(name) LIKE $1
		   OR LOWER(description) LIKE $1
		LIMIT 5
	`, searchPattern)
	
	if err == nil {
		defer routeRows.Close()
		routes := []map[string]interface{}{}
		for routeRows.Next() {
			var id int
			var name, description string
			routeRows.Scan(&id, &name, &description)
			routes = append(routes, map[string]interface{}{
				"id":   id,
				"name": name,
				"type": "route",
				"link": fmt.Sprintf("/routes?id=%d", id),
			})
		}
		results["routes"] = routes
	}

	// Search drivers (managers only)
	if role == "manager" {
		driverRows, err := db.Query(`
			SELECT id, username, email
			FROM users
			WHERE role = 'driver'
			  AND (LOWER(username) LIKE $1
			   OR LOWER(email) LIKE $1)
			LIMIT 5
		`, searchPattern)
		
		if err == nil {
			defer driverRows.Close()
			drivers := []map[string]interface{}{}
			for driverRows.Next() {
				var id int
				var username, email string
				driverRows.Scan(&id, &username, &email)
				drivers = append(drivers, map[string]interface{}{
					"id":   id,
					"name": username,
					"type": "driver",
					"link": fmt.Sprintf("/driver-profile?driver=%s", username),
				})
			}
			results["drivers"] = drivers
		}
	}

	// Count total results
	totalCount := len(results["students"].([]map[string]interface{})) +
		len(results["buses"].([]map[string]interface{})) +
		len(results["routes"].([]map[string]interface{})) +
		len(results["drivers"].([]map[string]interface{}))

	return map[string]interface{}{
		"results": results,
		"count":   totalCount,
		"query":   query,
	}
}

// Search page handler
func searchPageHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	query := r.URL.Query().Get("q")
	var searchResults map[string]interface{}
	
	if query != "" {
		searchResults = performGlobalSearch(query, user.Role)
	}

	renderTemplate(w, r, "search.html", map[string]interface{}{
		"User":    user,
		"Title":   "Search",
		"Query":   query,
		"Results": searchResults,
	})
}