package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

// availableDriversHandler returns drivers available for route assignment
func availableDriversHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		SendError(w, ErrMethodNotAllowed("Only GET method allowed"))
		return
	}

	// Check authentication
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		SendError(w, ErrUnauthorized("Access denied"))
		return
	}

	// Get drivers who are active (can be assigned to multiple routes)
	query := `
		SELECT u.username, u.status 
		FROM users u 
		WHERE u.role = 'driver' 
			AND u.status = 'active'
		ORDER BY u.username
	`

	var drivers []struct {
		Username string `json:"username"`
		Status   string `json:"status"`
	}

	err := db.Select(&drivers, query)
	if err != nil {
		logError(&AppError{
			Type:       ErrorTypeDatabase,
			Message:    "Failed to load available drivers",
			Detail:     err.Error(),
			StatusCode: http.StatusInternalServerError,
			Internal:   err,
		})
		SendError(w, ErrDatabase("Failed to load drivers", err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(drivers)
}

// availableBusesHandler returns buses available for route assignment
func availableBusesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		SendError(w, ErrMethodNotAllowed("Only GET method allowed"))
		return
	}

	// Check authentication
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		SendError(w, ErrUnauthorized("Access denied"))
		return
	}

	// Get all active buses (can be assigned to multiple routes)
	query := `
		SELECT b.bus_id, b.model, b.capacity, b.status 
		FROM buses b 
		WHERE b.status = 'active'
		ORDER BY b.bus_id
	`

	var buses []struct {
		BusID    string `json:"bus_id"`
		Model    string `json:"model"`
		Capacity int    `json:"capacity"`
		Status   string `json:"status"`
	}

	err := db.Select(&buses, query)
	if err != nil {
		logError(&AppError{
			Type:       ErrorTypeDatabase,
			Message:    "Failed to load available buses",
			Detail:     err.Error(),
			StatusCode: http.StatusInternalServerError,
			Internal:   err,
		})
		SendError(w, ErrDatabase("Failed to load buses", err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(buses)
}

// availableRoutesHandler returns routes available for assignment
func availableRoutesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		SendError(w, ErrMethodNotAllowed("Only GET method allowed"))
		return
	}

	// Check authentication
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		SendError(w, ErrUnauthorized("Access denied"))
		return
	}

	// Get routes that are not currently assigned
	query := `
		SELECT r.route_id, r.route_name, r.description 
		FROM routes r 
		WHERE r.route_id NOT IN (
			SELECT DISTINCT route_id 
			FROM route_assignments 
			WHERE route_id IS NOT NULL
		)
		ORDER BY r.route_name
	`

	var routes []struct {
		RouteID     string `json:"route_id"`
		RouteName   string `json:"route_name"`
		Description string `json:"description"`
	}

	err := db.Select(&routes, query)
	if err != nil {
		logError(&AppError{
			Type:       ErrorTypeDatabase,
			Message:    "Failed to load available routes",
			Detail:     err.Error(),
			StatusCode: http.StatusInternalServerError,
			Internal:   err,
		})
		SendError(w, ErrDatabase("Failed to load routes", err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(routes)
}

// checkAssignmentConflictsHandler checks for conflicts in route assignments
func checkAssignmentConflictsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		SendError(w, ErrMethodNotAllowed("Only GET method allowed"))
		return
	}

	// Check authentication
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		SendError(w, ErrUnauthorized("Access denied"))
		return
	}

	var data struct {
		Driver  string `json:"driver"`
		BusID   string `json:"busId"`
		RouteID string `json:"routeId"`
	}

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		SendError(w, ErrValidation("Invalid request data"))
		return
	}

	conflicts := []string{}

	// Check if this exact combination already exists
	var existingCount int
	err := db.Get(&existingCount, 
		"SELECT COUNT(*) FROM route_assignments WHERE driver = $1 AND bus_id = $2 AND route_id = $3", 
		data.Driver, data.BusID, data.RouteID)
	if err == nil && existingCount > 0 {
		conflicts = append(conflicts, "This exact driver-bus-route combination already exists")
	}

	// Optional: Check if driver is already assigned to this specific route with a different bus
	var driverRouteCount int
	err = db.Get(&driverRouteCount, 
		"SELECT COUNT(*) FROM route_assignments WHERE driver = $1 AND route_id = $2 AND bus_id != $3", 
		data.Driver, data.RouteID, data.BusID)
	if err == nil && driverRouteCount > 0 {
		conflicts = append(conflicts, fmt.Sprintf("Driver %s is already assigned to route %s with a different bus", data.Driver, data.RouteID))
	}

	response := map[string]interface{}{
		"conflicts": conflicts,
		"hasConflicts": len(conflicts) > 0,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// vehicleMileageHandler returns the last recorded mileage for a vehicle
func vehicleMileageHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		SendError(w, ErrMethodNotAllowed("Only GET method allowed"))
		return
	}

	// Check authentication
	user := getUserFromSession(r)
	if user == nil {
		SendError(w, ErrUnauthorized("Access denied"))
		return
	}

	// Extract vehicle type and ID from URL
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 5 {
		SendError(w, ErrValidation("Invalid URL format"))
		return
	}

	vehicleType := parts[3]
	vehicleID := parts[4]

	var lastMileage int
	var query string

	if vehicleType == "bus" {
		query = `
			SELECT COALESCE(MAX(mileage), 0)
			FROM bus_maintenance_logs
			WHERE bus_id = $1
		`
	} else {
		query = `
			SELECT COALESCE(MAX(mileage), 0)
			FROM vehicle_maintenance_logs
			WHERE vehicle_id = $1
		`
	}

	err := db.Get(&lastMileage, query, vehicleID)
	if err != nil {
		logError(&AppError{
			Type:       ErrorTypeDatabase,
			Message:    "Failed to get vehicle mileage",
			Detail:     err.Error(),
			StatusCode: http.StatusInternalServerError,
			Internal:   err,
		})
		lastMileage = 0
	}

	response := map[string]interface{}{
		"lastMileage": lastMileage,
		"vehicleType": vehicleType,
		"vehicleId":   vehicleID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// maintenanceVendorsHandler returns common maintenance vendors
func maintenanceVendorsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		SendError(w, ErrMethodNotAllowed("Only GET method allowed"))
		return
	}

	// Check authentication
	user := getUserFromSession(r)
	if user == nil {
		SendError(w, ErrUnauthorized("Access denied"))
		return
	}

	// Get distinct vendors from maintenance logs
	var vendors []string
	
	query := `
		SELECT DISTINCT vendor FROM (
			SELECT vendor FROM bus_maintenance_logs WHERE vendor IS NOT NULL AND vendor != ''
			UNION
			SELECT vendor FROM vehicle_maintenance_logs WHERE vendor IS NOT NULL AND vendor != ''
		) AS all_vendors
		ORDER BY vendor
		LIMIT 20
	`

	err := db.Select(&vendors, query)
	if err != nil {
		// Return empty list on error
		vendors = []string{}
	}

	// Add some common vendors if list is short
	if len(vendors) < 5 {
		commonVendors := []string{
			"Fleet Services Inc",
			"Quick Lube Express",
			"Tire Pros",
			"District Maintenance Shop",
			"Mobile Mechanic Services",
		}
		vendors = append(vendors, commonVendors...)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(vendors)
}

// analyzeImportFileHandler analyzes an Excel file for import
func analyzeImportFileHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		SendError(w, ErrMethodNotAllowed("Only GET method allowed"))
		return
	}

	// Check authentication
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		SendError(w, ErrUnauthorized("Access denied"))
		return
	}

	// Parse multipart form
	err := r.ParseMultipartForm(10 << 20) // 10MB limit
	if err != nil {
		sendJSONError(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		sendJSONError(w, "No file provided", http.StatusBadRequest)
		return
	}
	defer file.Close()

	importType := r.FormValue("type")
	if importType == "" {
		sendJSONError(w, "Import type not specified", http.StatusBadRequest)
		return
	}

	// Create temporary file
	tempFile := filepath.Join(".", "temp_"+header.Filename)
	out, err := createFile(tempFile)
	if err != nil {
		sendJSONError(w, "Failed to create temporary file", http.StatusInternalServerError)
		return
	}
	defer removeFile(tempFile)
	defer out.Close()

	// Copy uploaded file
	_, err = copyFile(out, file)
	if err != nil {
		sendJSONError(w, "Failed to save file", http.StatusInternalServerError)
		return
	}

	// Open Excel file
	f, err := excelize.OpenFile(tempFile)
	if err != nil {
		sendJSONError(w, "Invalid Excel file", http.StatusBadRequest)
		return
	}
	defer f.Close()

	// Get first sheet
	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		sendJSONError(w, "No sheets found in Excel file", http.StatusBadRequest)
		return
	}

	// Get column headers (first row)
	rows, err := f.GetRows(sheets[0])
	if err != nil || len(rows) == 0 {
		sendJSONError(w, "Failed to read Excel file", http.StatusBadRequest)
		return
	}

	columns := rows[0]
	
	// Get sample data (next 5 rows)
	sampleData := make(map[string][]string)
	for i, col := range columns {
		samples := []string{}
		for j := 1; j < len(rows) && j <= 5; j++ {
			if i < len(rows[j]) && rows[j][i] != "" {
				samples = append(samples, rows[j][i])
			}
		}
		sampleData[col] = samples
	}

	// Get required fields based on import type
	requiredFields := getRequiredFields(importType)

	response := map[string]interface{}{
		"columns":        columns,
		"sampleData":     sampleData,
		"requiredFields": requiredFields,
		"totalRows":      len(rows) - 1,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Helper functions

func getRequiredFields(importType string) []string {
	switch importType {
	case "students":
		return []string{"student_id", "name", "phone_number"}
	case "ecse":
		return []string{"student_id", "first_name", "last_name", "date_of_birth"}
	case "mileage":
		return []string{"date", "driver", "bus_id", "start_mileage", "end_mileage"}
	default:
		return []string{}
	}
}

func validateImportDataFromFile(filename string, importType string, mappings map[string]string) (int, int, int) {
	// This is a simplified validation - in production, you'd have more comprehensive checks
	f, err := excelize.OpenFile(filename)
	if err != nil {
		return 0, 0, 0
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return 0, 0, 0
	}

	rows, err := f.GetRows(sheets[0])
	if err != nil || len(rows) <= 1 {
		return 0, 0, 0
	}

	// Simple validation: count non-empty rows
	validCount := 0
	invalidCount := 0
	warningCount := 0

	// Skip header row
	for i := 1; i < len(rows); i++ {
		row := rows[i]
		if len(row) == 0 {
			continue
		}

		// Check if required fields have values
		hasAllRequired := true
		requiredFields := getRequiredFields(importType)
		
		for _, field := range requiredFields {
			if mappedCol, ok := mappings[field]; ok {
				// Find column index
				colIndex := -1
				for j, col := range rows[0] {
					if col == mappedCol {
						colIndex = j
						break
					}
				}

				if colIndex == -1 || colIndex >= len(row) || row[colIndex] == "" {
					hasAllRequired = false
					break
				}
			} else {
				hasAllRequired = false
				break
			}
		}

		if hasAllRequired {
			validCount++
		} else {
			invalidCount++
		}
	}

	return validCount, invalidCount, warningCount
}

func sendJSONError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{
		"error": message,
	})
}

// File handling utilities
func createFile(filename string) (*os.File, error) {
	return os.Create(filename)
}

func removeFile(filename string) error {
	return os.Remove(filename)
}

func copyFile(dst io.Writer, src io.Reader) (int64, error) {
	return io.Copy(dst, src)
}

// lastMaintenanceHandlerOLD returns the last maintenance date for a vehicle
func lastMaintenanceHandlerOLD(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		SendError(w, ErrMethodNotAllowed("Only GET method allowed"))
		return
	}

	// Check authentication
	user := getUserFromSession(r)
	if user == nil {
		SendError(w, ErrUnauthorized("Access denied"))
		return
	}

	// Extract vehicle type and ID from URL
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 5 {
		SendError(w, ErrValidation("Invalid URL format"))
		return
	}

	vehicleType := parts[3]
	vehicleID := parts[4]

	var lastMaintenanceDate string
	var query string

	if vehicleType == "bus" {
		query = `
			SELECT COALESCE(MAX(date), '') as last_date
			FROM bus_maintenance_logs
			WHERE bus_id = $1
		`
	} else {
		query = `
			SELECT COALESCE(MAX(date), '') as last_date
			FROM vehicle_maintenance_logs
			WHERE vehicle_id = $1
		`
	}

	err := db.Get(&lastMaintenanceDate, query, vehicleID)
	if err != nil {
		logError(&AppError{
			Type:       ErrorTypeDatabase,
			Message:    "Failed to get last maintenance date",
			Detail:     err.Error(),
			StatusCode: http.StatusInternalServerError,
			Internal:   err,
		})
		lastMaintenanceDate = ""
	}

	var formattedDate string
	if lastMaintenanceDate != "" {
		if date, err := time.Parse("2006-01-02", lastMaintenanceDate); err == nil {
			daysSince := int(time.Since(date).Hours() / 24)
			if daysSince == 0 {
				formattedDate = "Today"
			} else if daysSince == 1 {
				formattedDate = "Yesterday"
			} else {
				formattedDate = fmt.Sprintf("%d days ago", daysSince)
			}
		} else {
			formattedDate = "Unknown"
		}
	} else {
		formattedDate = "No maintenance recorded"
	}

	response := map[string]interface{}{
		"lastMaintenanceDate": formattedDate,
		"vehicleType":         vehicleType,
		"vehicleId":          vehicleID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}