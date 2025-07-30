package main

import (
	"database/sql"
	"log"
	"net/http"
	"strconv"
)

// editBusHandler displays the edit bus form
func editBusHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	busID := r.URL.Query().Get("id")
	if busID == "" {
		http.Error(w, "Bus ID required", http.StatusBadRequest)
		return
	}

	// Load bus data
	var bus Bus
	err := db.QueryRow(`
		SELECT bus_id, COALESCE(model, ''), COALESCE(capacity, 0), status, 
			   COALESCE(current_mileage, 0), COALESCE(last_oil_change_miles, 0), 
			   COALESCE(last_tire_change_miles, 0), created_at
		FROM buses 
		WHERE bus_id = $1
	`, busID).Scan(&bus.BusID, &bus.Model, &bus.Capacity, &bus.Status,
		&bus.CurrentMileage, &bus.LastOilChange, &bus.LastTireService,
		&bus.CreatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Bus not found", http.StatusNotFound)
			return
		}
		log.Printf("Error loading bus: %v", err)
		http.Error(w, "Failed to load bus", http.StatusInternalServerError)
		return
	}

	// Handle POST request to update bus
	if r.Method == http.MethodPost {
		// Parse form
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Failed to parse form", http.StatusBadRequest)
			return
		}

		// Validate CSRF token
		if !validateCSRF(r) {
			http.Error(w, "Invalid CSRF token", http.StatusForbidden)
			return
		}

		// Get form values
		model := r.FormValue("model")
		capacity := r.FormValue("capacity")
		status := r.FormValue("status")
		currentMileage := r.FormValue("current_mileage")
		lastOilChange := r.FormValue("last_oil_change_miles")
		lastTireChange := r.FormValue("last_tire_change_miles")

		// Validate inputs
		capacityInt, err := strconv.Atoi(capacity)
		if err != nil || capacityInt < 1 {
			http.Error(w, "Invalid capacity", http.StatusBadRequest)
			return
		}

		currentMileageInt := 0
		if currentMileage != "" {
			currentMileageInt, err = strconv.Atoi(currentMileage)
			if err != nil || currentMileageInt < 0 {
				http.Error(w, "Invalid current mileage", http.StatusBadRequest)
				return
			}
		}

		lastOilChangeInt := 0
		if lastOilChange != "" {
			lastOilChangeInt, err = strconv.Atoi(lastOilChange)
			if err != nil || lastOilChangeInt < 0 {
				http.Error(w, "Invalid last oil change mileage", http.StatusBadRequest)
				return
			}
		}

		lastTireChangeInt := 0
		if lastTireChange != "" {
			lastTireChangeInt, err = strconv.Atoi(lastTireChange)
			if err != nil || lastTireChangeInt < 0 {
				http.Error(w, "Invalid last tire change mileage", http.StatusBadRequest)
				return
			}
		}

		// Update bus
		_, err = db.Exec(`
			UPDATE buses 
			SET model = $1, 
				capacity = $2, 
				status = $3,
				current_mileage = $4,
				last_oil_change_miles = $5,
				last_tire_change_miles = $6,
				updated_at = NOW()
			WHERE bus_id = $7
		`, model, capacityInt, status, currentMileageInt, lastOilChangeInt, lastTireChangeInt, busID)

		if err != nil {
			log.Printf("Error updating bus: %v", err)
			http.Error(w, "Failed to update bus", http.StatusInternalServerError)
			return
		}

		// Log activity (if function exists)
		// logActivity(db, &Activity{
		// 	UserID:      user.Username,
		// 	Action:      "update_bus",
		// 	Description: fmt.Sprintf("Updated bus %s", busID),
		// 	Timestamp:   time.Now(),
		// })

		// Redirect back to fleet page
		http.Redirect(w, r, "/fleet", http.StatusSeeOther)
		return
	}

	// Prepare page data for GET request
	data := map[string]interface{}{
		"Title":     "Edit Bus",
		"User":      user,
		"CSRFToken": generateCSRFToken(),
		"Data": map[string]interface{}{
			"Bus": bus,
		},
	}

	renderTemplate(w, r, "edit_bus.html", data)
}