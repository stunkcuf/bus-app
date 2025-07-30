package main

import (
	"database/sql"
	"log"
	"net/http"
	"strconv"
	"strings"
)

// fleetVehicleEditHandler handles editing a fleet vehicle
func fleetVehicleEditHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	// Only managers can edit fleet vehicles
	if user.Role != "manager" {
		http.Error(w, "Unauthorized", http.StatusForbidden)
		return
	}

	// Get vehicle ID from URL
	vehicleID := strings.TrimPrefix(r.URL.Path, "/fleet-vehicle/edit/")
	log.Printf("Fleet vehicle edit handler - extracted ID: '%s' from path: '%s'", vehicleID, r.URL.Path)
	if vehicleID == "" {
		http.Error(w, "Vehicle ID required", http.StatusBadRequest)
		return
	}

	if r.Method == "GET" {
		// Load vehicle data
		vehicle, err := loadFleetVehicleByID(vehicleID)
		if err != nil {
			log.Printf("Error loading fleet vehicle %s: %v", vehicleID, err)
			http.Error(w, "Vehicle not found", http.StatusNotFound)
			return
		}

		data := map[string]interface{}{
			"Title":     "Edit Fleet Vehicle",
			"User":      user,
			"Vehicle":   vehicle,
			"CSRFToken": getSessionCSRFToken(r),
			"CSPNonce":  r.Context().Value("cspNonce"),
		}

		renderTemplate(w, r, "fleet_vehicle_edit.html", data)
		return
	}

	if r.Method == "POST" {
		// Validate CSRF
		if !validateCSRF(r) {
			http.Error(w, "Invalid CSRF token", http.StatusForbidden)
			return
		}

		// Parse form data
		vehicle := FleetVehicle{
			VehicleNumber:   parseIntOrNull(r.FormValue("vehicle_number")),
			Year:            parseIntOrNull(r.FormValue("year")),
			Make:            sql.NullString{String: r.FormValue("make"), Valid: r.FormValue("make") != ""},
			Model:           sql.NullString{String: r.FormValue("model"), Valid: r.FormValue("model") != ""},
			Description:     sql.NullString{String: r.FormValue("description"), Valid: r.FormValue("description") != ""},
			SerialNumber:    sql.NullString{String: r.FormValue("serial_number"), Valid: r.FormValue("serial_number") != ""},
			License:         sql.NullString{String: r.FormValue("license"), Valid: r.FormValue("license") != ""},
			Location:        sql.NullString{String: r.FormValue("location"), Valid: r.FormValue("location") != ""},
			TireSize:        sql.NullString{String: r.FormValue("tire_size"), Valid: r.FormValue("tire_size") != ""},
			SheetName:       sql.NullString{String: r.FormValue("sheet_name"), Valid: r.FormValue("sheet_name") != ""},
		}

		// Parse the ID
		idInt, err := strconv.Atoi(vehicleID)
		if err != nil {
			http.Error(w, "Invalid vehicle ID", http.StatusBadRequest)
			return
		}
		vehicle.ID = idInt

		// Update vehicle
		err = updateFleetVehicle(&vehicle)
		if err != nil {
			log.Printf("Error updating fleet vehicle: %v", err)
			data := map[string]interface{}{
				"Title":     "Edit Fleet Vehicle",
				"User":      user,
				"Vehicle":   vehicle,
				"CSRFToken": getSessionCSRFToken(r),
				"CSPNonce":  r.Context().Value("cspNonce"),
				"Error":     "Failed to update vehicle. Please try again.",
			}
			renderTemplate(w, r, "fleet_vehicle_edit.html", data)
			return
		}

		// Success - redirect to fleet vehicles page
		http.Redirect(w, r, "/fleet-vehicles?success=Vehicle+updated+successfully", http.StatusFound)
	}
}

// fleetVehicleAddHandler handles adding a new fleet vehicle
func fleetVehicleAddHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	// Only managers can add fleet vehicles
	if user.Role != "manager" {
		http.Error(w, "Unauthorized", http.StatusForbidden)
		return
	}

	if r.Method == "GET" {
		data := map[string]interface{}{
			"Title":     "Add Fleet Vehicle",
			"User":      user,
			"CSRFToken": getSessionCSRFToken(r),
			"CSPNonce":  r.Context().Value("cspNonce"),
		}

		renderTemplate(w, r, "fleet_vehicle_add.html", data)
		return
	}

	if r.Method == "POST" {
		// Validate CSRF
		if !validateCSRF(r) {
			http.Error(w, "Invalid CSRF token", http.StatusForbidden)
			return
		}

		// Parse form data
		vehicle := FleetVehicle{
			VehicleNumber:   parseIntOrNull(r.FormValue("vehicle_number")),
			Year:            parseIntOrNull(r.FormValue("year")),
			Make:            sql.NullString{String: r.FormValue("make"), Valid: r.FormValue("make") != ""},
			Model:           sql.NullString{String: r.FormValue("model"), Valid: r.FormValue("model") != ""},
			Description:     sql.NullString{String: r.FormValue("description"), Valid: r.FormValue("description") != ""},
			SerialNumber:    sql.NullString{String: r.FormValue("serial_number"), Valid: r.FormValue("serial_number") != ""},
			License:         sql.NullString{String: r.FormValue("license"), Valid: r.FormValue("license") != ""},
			Location:        sql.NullString{String: r.FormValue("location"), Valid: r.FormValue("location") != ""},
			TireSize:        sql.NullString{String: r.FormValue("tire_size"), Valid: r.FormValue("tire_size") != ""},
			SheetName:       sql.NullString{String: r.FormValue("sheet_name"), Valid: r.FormValue("sheet_name") != ""},
		}

		// Add vehicle
		err := addFleetVehicle(&vehicle)
		if err != nil {
			log.Printf("Error adding fleet vehicle: %v", err)
			data := map[string]interface{}{
				"Title":     "Add Fleet Vehicle",
				"User":      user,
				"Vehicle":   vehicle,
				"CSRFToken": getSessionCSRFToken(r),
				"CSPNonce":  r.Context().Value("cspNonce"),
				"Error":     "Failed to add vehicle. Please try again.",
			}
			renderTemplate(w, r, "fleet_vehicle_add.html", data)
			return
		}

		// Success - redirect to fleet vehicles page
		http.Redirect(w, r, "/fleet-vehicles?success=Vehicle+added+successfully", http.StatusFound)
	}
}

// API endpoint for fleet vehicle operations
func apiFleetVehicleHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		SendError(w, ErrUnauthorized("Manager access required"))
		return
	}

	switch r.Method {
	case "DELETE":
		// Delete vehicle
		vehicleID := r.URL.Query().Get("id")
		if vehicleID == "" {
			SendError(w, ErrBadRequest("Vehicle ID required"))
			return
		}

		err := deleteFleetVehicle(vehicleID)
		if err != nil {
			SendError(w, ErrDatabase("Failed to delete vehicle", err))
			return
		}

		SendJSON(w, http.StatusOK, map[string]string{
			"message": "Vehicle deleted successfully",
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// Helper functions for database operations

func loadFleetVehicleByID(id string) (*FleetVehicle, error) {
	return loadFleetVehicleByIDNew(id)
}

func updateFleetVehicle(vehicle *FleetVehicle) error {
	return updateFleetVehicleNew(vehicle)
}

func addFleetVehicle(vehicle *FleetVehicle) error {
	return addFleetVehicleNew(vehicle)
}

func deleteFleetVehicle(id string) error {
	return deleteFleetVehicleNew(id)
}

// Helper function to parse int or return null
func parseIntOrNull(s string) sql.NullInt32 {
	if s == "" {
		return sql.NullInt32{Valid: false}
	}
	if i, err := strconv.Atoi(s); err == nil {
		return sql.NullInt32{Int32: int32(i), Valid: true}
	}
	return sql.NullInt32{Valid: false}
}