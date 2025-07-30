package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// assignRouteHandler assigns a route to a driver and bus
func assignRouteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse form data
	driver := r.FormValue("driver")
	busID := r.FormValue("bus_id")
	routeID := r.FormValue("route_id")

	// Validate inputs
	if driver == "" || busID == "" || routeID == "" {
		log.Printf("Missing required fields: driver=%s, bus_id=%s, route_id=%s", driver, busID, routeID)
		http.Error(w, "All fields are required", http.StatusBadRequest)
		return
	}

	// Create assignment (removed route_name as it's not in the table)
	_, err := db.Exec(`
		INSERT INTO route_assignments (driver, bus_id, route_id, assigned_date, created_at)
		VALUES ($1, $2, $3, CURRENT_DATE, CURRENT_TIMESTAMP)
		ON CONFLICT ON CONSTRAINT route_assignments_unique_assignment DO UPDATE
		SET assigned_date = CURRENT_DATE, created_at = CURRENT_TIMESTAMP
	`, driver, busID, routeID)

	if err != nil {
		log.Printf("Error creating route assignment: %v", err)
		http.Error(w, "Failed to assign route", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/assign-routes", http.StatusSeeOther)
}

// unassignRouteHandler removes a route assignment
func unassignRouteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse form data
	driver := r.FormValue("driver")
	busID := r.FormValue("bus_id")
	routeID := r.FormValue("route_id")

	// Delete specific assignment
	query := `DELETE FROM route_assignments WHERE driver = $1 AND bus_id = $2`
	args := []interface{}{driver, busID}
	
	// If route_id is provided, delete specific route assignment
	if routeID != "" {
		query += ` AND route_id = $3`
		args = append(args, routeID)
	}
	
	_, err := db.Exec(query, args...)

	if err != nil {
		log.Printf("Error removing route assignment: %v", err)
		http.Error(w, "Failed to unassign route", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/assign-routes", http.StatusSeeOther)
}

// addRouteHandler adds a new route
func addRouteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse form data
	routeName := r.FormValue("route_name")
	description := r.FormValue("description")

	// Generate route ID
	routeID := fmt.Sprintf("ROUTE-%d", time.Now().Unix())

	// Insert new route
	_, err := db.Exec(`
		INSERT INTO routes (route_id, route_name, description, created_at)
		VALUES ($1, $2, $3, CURRENT_TIMESTAMP)
	`, routeID, routeName, description)

	if err != nil {
		log.Printf("Error adding route: %v", err)
		http.Error(w, "Failed to add route", http.StatusInternalServerError)
		return
	}

	// Clear cache
	dataCache.clearRoutes()

	http.Redirect(w, r, "/assign-routes", http.StatusSeeOther)
}

// editRouteHandler updates a route
func editRouteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse JSON request
	var req struct {
		RouteID     string `json:"route_id"`
		RouteName   string `json:"route_name"`
		Description string `json:"description"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Update route
	_, err := db.Exec(`
		UPDATE routes 
		SET route_name = $2, description = $3
		WHERE route_id = $1
	`, req.RouteID, req.RouteName, req.Description)

	if err != nil {
		log.Printf("Error updating route: %v", err)
		http.Error(w, "Failed to update route", http.StatusInternalServerError)
		return
	}

	// Clear cache
	dataCache.clearRoutes()

	// Return success
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Route updated successfully",
	})
}

// deleteRouteHandler deletes a route
func deleteRouteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse JSON request
	var req struct {
		RouteID string `json:"route_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Check if route is assigned
	var isAssigned bool
	err := db.Get(&isAssigned, `
		SELECT EXISTS(
			SELECT 1 FROM route_assignments WHERE route_id = $1
		)
	`, req.RouteID)

	if err != nil {
		log.Printf("Error checking route assignment: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	if isAssigned {
		http.Error(w, "Cannot delete assigned route", http.StatusConflict)
		return
	}

	// Delete route
	_, err = db.Exec("DELETE FROM routes WHERE route_id = $1", req.RouteID)
	if err != nil {
		log.Printf("Error deleting route: %v", err)
		http.Error(w, "Failed to delete route", http.StatusInternalServerError)
		return
	}

	// Clear cache
	dataCache.clearRoutes()

	// Return success
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Route deleted successfully",
	})
}

// viewMileageReportsHandler shows mileage reports
func viewMileageReportsHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	// Get filter parameters
	busNumber := r.URL.Query().Get("bus_number")
	month := r.URL.Query().Get("month")
	year := r.URL.Query().Get("year")

	// Build query
	query := `SELECT * FROM mileage_records WHERE 1=1`
	args := []interface{}{}
	argCount := 0

	if busNumber != "" {
		argCount++
		query += fmt.Sprintf(" AND bus_number = $%d", argCount)
		args = append(args, busNumber)
	}

	if month != "" && year != "" {
		argCount++
		query += fmt.Sprintf(" AND EXTRACT(MONTH FROM date) = $%d", argCount)
		args = append(args, month)
		
		argCount++
		query += fmt.Sprintf(" AND EXTRACT(YEAR FROM date) = $%d", argCount)
		args = append(args, year)
	}

	query += " ORDER BY date DESC"

	// Load records
	var records []MileageRecord
	err := db.Select(&records, query, args...)
	if err != nil {
		log.Printf("Error loading mileage records: %v", err)
		records = []MileageRecord{}
	}

	// Get bus list for filter
	var busNumbers []int
	err = db.Select(&busNumbers, "SELECT DISTINCT bus_number FROM mileage_records ORDER BY bus_number")
	if err != nil {
		log.Printf("Error loading bus numbers: %v", err)
	}

	data := map[string]interface{}{
		"User":       user,
		"Title":      "Mileage Reports",
		"Records":    records,
		"BusNumbers": busNumbers,
		"CSRFToken":  getSessionCSRFToken(r),
		"FilterBus":  busNumber,
		"FilterMonth": month,
		"FilterYear": year,
	}

	renderTemplate(w, r, "view_mileage_reports.html", data)
}

// exportMileageHandler exports mileage data
func exportMileageHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get export format
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "csv"
	}

	// Load all mileage records
	var records []MileageRecord
	err := db.Select(&records, "SELECT * FROM mileage_records ORDER BY date DESC")
	if err != nil {
		log.Printf("Error loading mileage records for export: %v", err)
		http.Error(w, "Failed to load records", http.StatusInternalServerError)
		return
	}

	switch format {
	case "csv":
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", "attachment; filename=mileage_records.csv")
		
		// Write CSV header
		fmt.Fprintln(w, "Date,BusNumber,StartMileage,EndMileage,TotalMiles,Driver")
		
		// Write records
		for _, r := range records {
			totalMiles := 0
			if r.EndMileage > 0 && r.StartMileage > 0 {
				totalMiles = r.EndMileage - r.StartMileage
			}
			
			driver := r.Driver
			if driver == "" {
				driver = "N/A"
			}
			
			fmt.Fprintf(w, "%s,%d,%d,%d,%d,%s\n",
				r.Date, r.BusNumber, r.StartMileage, r.EndMileage, totalMiles, driver)
		}
		
	case "json":
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", "attachment; filename=mileage_records.json")
		json.NewEncoder(w).Encode(records)
		
	default:
		http.Error(w, "Unsupported format", http.StatusBadRequest)
	}
}