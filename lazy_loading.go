package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// Note on SQL Safety in this file:
// All SQL queries in this file use parameterized queries with placeholders ($1, $2, etc.)
// The countQuery pattern that concatenates strings is safe because:
// 1. The base query uses parameterized placeholders for all user inputs
// 2. The concatenation only wraps the parameterized query in a COUNT(*) subquery
// 3. The same args array is passed to both the count and main queries
// No user input is ever directly concatenated into the SQL strings.

// LazyLoadConfig defines configuration for lazy loading
type LazyLoadConfig struct {
	DefaultPageSize int
	MaxPageSize     int
	InitialLoad     int
}

// DefaultLazyLoadConfig returns sensible defaults for lazy loading
func DefaultLazyLoadConfig() LazyLoadConfig {
	return LazyLoadConfig{
		DefaultPageSize: 25,
		MaxPageSize:     100,
		InitialLoad:     10,
	}
}

// LazyLoadResponse represents a paginated response with metadata
type LazyLoadResponse struct {
	Data        interface{} `json:"data"`
	Page        int         `json:"page"`
	PerPage     int         `json:"per_page"`
	Total       int         `json:"total"`
	TotalPages  int         `json:"total_pages"`
	HasNext     bool        `json:"has_next"`
	HasPrevious bool        `json:"has_previous"`
	LoadTime    string      `json:"load_time"`
}

// monthlyMileageReportsAPIHandler provides paginated API access to monthly mileage reports
func monthlyMileageReportsAPIHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// Check authentication
	user := getUserFromSession(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse query parameters
	pageStr := r.URL.Query().Get("page")
	perPageStr := r.URL.Query().Get("per_page")
	yearStr := r.URL.Query().Get("year")
	month := r.URL.Query().Get("month")
	busID := r.URL.Query().Get("bus_id")

	config := DefaultLazyLoadConfig()
	page := 1
	perPage := config.DefaultPageSize

	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	if perPageStr != "" {
		if pp, err := strconv.Atoi(perPageStr); err == nil && pp > 0 && pp <= config.MaxPageSize {
			perPage = pp
		}
	}

	// Build query with filters
	query := `
		SELECT id, report_month, report_year, bus_id, driver_name, 
		       total_miles, fuel_cost, maintenance_cost, notes, created_at
		FROM monthly_mileage_reports WHERE 1=1
	`
	args := []interface{}{}
	argIndex := 1

	if yearStr != "" {
		query += fmt.Sprintf(" AND report_year = $%d", argIndex)
		if year, err := strconv.Atoi(yearStr); err == nil {
			args = append(args, year)
			argIndex++
		}
	}

	if month != "" {
		query += fmt.Sprintf(" AND report_month = $%d", argIndex)
		args = append(args, month)
		argIndex++
	}

	if busID != "" {
		query += fmt.Sprintf(" AND bus_id = $%d", argIndex)
		args = append(args, busID)
		argIndex++
	}

	// Get total count
	countQuery := "SELECT COUNT(*) FROM (" + query + ") as count_query"
	var total int
	if err := db.Get(&total, countQuery, args...); err != nil {
		LogError("Failed to get total count for monthly mileage reports", err)
		http.Error(w, "Failed to get total count", http.StatusInternalServerError)
		return
	}

	// Add pagination to main query
	offset := (page - 1) * perPage
	query += fmt.Sprintf(" ORDER BY report_year DESC, report_month DESC LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, perPage, offset)

	// Get paginated data
	var reports []MonthlyMileageReport
	if err := db.Select(&reports, query, args...); err != nil {
		LogError("Failed to load monthly mileage reports", err)
		http.Error(w, "Failed to load reports", http.StatusInternalServerError)
		return
	}

	// Calculate pagination metadata
	totalPages := (total + perPage - 1) / perPage
	hasNext := page < totalPages
	hasPrevious := page > 1

	response := LazyLoadResponse{
		Data:        reports,
		Page:        page,
		PerPage:     perPage,
		Total:       total,
		TotalPages:  totalPages,
		HasNext:     hasNext,
		HasPrevious: hasPrevious,
		LoadTime:    fmt.Sprintf("%.2fms", float64(time.Since(start).Nanoseconds())/1000000),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// maintenanceRecordsAPIHandler provides paginated API access to maintenance records
func maintenanceRecordsAPIHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// Check authentication
	user := getUserFromSession(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse query parameters
	pageStr := r.URL.Query().Get("page")
	perPageStr := r.URL.Query().Get("per_page")
	vehicleID := r.URL.Query().Get("vehicle_id")
	category := r.URL.Query().Get("category")

	config := DefaultLazyLoadConfig()
	page := 1
	perPage := config.DefaultPageSize

	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	if perPageStr != "" {
		if pp, err := strconv.Atoi(perPageStr); err == nil && pp > 0 && pp <= config.MaxPageSize {
			perPage = pp
		}
	}

	// Build query with filters
	query := `
		SELECT id, vehicle_id, maintenance_date, category, description, 
		       cost, mileage, mechanic, status, created_at
		FROM maintenance_records WHERE 1=1
	`
	args := []interface{}{}
	argIndex := 1

	if vehicleID != "" {
		query += fmt.Sprintf(" AND vehicle_id = $%d", argIndex)
		args = append(args, vehicleID)
		argIndex++
	}

	if category != "" {
		query += fmt.Sprintf(" AND category = $%d", argIndex)
		args = append(args, category)
		argIndex++
	}

	// Get total count
	countQuery := "SELECT COUNT(*) FROM (" + query + ") as count_query"
	var total int
	if err := db.Get(&total, countQuery, args...); err != nil {
		LogError("Failed to get total count for maintenance records", err)
		http.Error(w, "Failed to get total count", http.StatusInternalServerError)
		return
	}

	// Add pagination to main query
	offset := (page - 1) * perPage
	query += fmt.Sprintf(" ORDER BY maintenance_date DESC LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, perPage, offset)

	// Get paginated data
	var records []MaintenanceRecord
	if err := db.Select(&records, query, args...); err != nil {
		LogError("Failed to load maintenance records", err)
		http.Error(w, "Failed to load records", http.StatusInternalServerError)
		return
	}

	// Calculate pagination metadata
	totalPages := (total + perPage - 1) / perPage
	hasNext := page < totalPages
	hasPrevious := page > 1

	response := LazyLoadResponse{
		Data:        records,
		Page:        page,
		PerPage:     perPage,
		Total:       total,
		TotalPages:  totalPages,
		HasNext:     hasNext,
		HasPrevious: hasPrevious,
		LoadTime:    fmt.Sprintf("%.2fms", float64(time.Since(start).Nanoseconds())/1000000),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// fleetVehiclesAPIHandler provides paginated API access to fleet vehicles
func fleetVehiclesAPIHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// Check authentication
	user := getUserFromSession(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse query parameters
	pageStr := r.URL.Query().Get("page")
	perPageStr := r.URL.Query().Get("per_page")
	status := r.URL.Query().Get("status")
	makeFilter := r.URL.Query().Get("make")

	config := DefaultLazyLoadConfig()
	page := 1
	perPage := config.DefaultPageSize

	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	if perPageStr != "" {
		if pp, err := strconv.Atoi(perPageStr); err == nil && pp > 0 && pp <= config.MaxPageSize {
			perPage = pp
		}
	}

	// Build query with filters
	query := `
		SELECT 
			CASE 
				WHEN vehicle_id LIKE 'FV%' THEN SUBSTRING(vehicle_id FROM 3)::INTEGER
				ELSE NULL
			END as id,
			vehicle_number, make, model, 
			CASE WHEN year ~ '^\d+$' THEN year::INTEGER ELSE NULL END as year,
			serial_number as vin, license as license_plate, 
			status, 
			CASE WHEN current_mileage ~ '^\d+$' THEN current_mileage::INTEGER ELSE NULL END as mileage,
			last_service, created_at
		FROM vehicles 
		WHERE vehicle_type = 'fleet'
	`
	args := []interface{}{}
	argIndex := 1

	if status != "" {
		query += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, status)
		argIndex++
	}

	if makeFilter != "" {
		query += fmt.Sprintf(" AND make ILIKE $%d", argIndex)
		args = append(args, "%"+makeFilter+"%")
		argIndex++
	}

	// Get total count
	countQuery := "SELECT COUNT(*) FROM (" + query + ") as count_query"
	var total int
	if err := db.Get(&total, countQuery, args...); err != nil {
		LogError("Failed to get total count for fleet vehicles", err)
		http.Error(w, "Failed to get total count", http.StatusInternalServerError)
		return
	}

	// Add pagination to main query
	offset := (page - 1) * perPage
	query += fmt.Sprintf(" ORDER BY vehicle_number LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, perPage, offset)

	// Get paginated data
	var vehicles []FleetVehicle
	if err := db.Select(&vehicles, query, args...); err != nil {
		LogError("Failed to load fleet vehicles", err)
		http.Error(w, "Failed to load vehicles", http.StatusInternalServerError)
		return
	}

	// Calculate pagination metadata
	totalPages := (total + perPage - 1) / perPage
	hasNext := page < totalPages
	hasPrevious := page > 1

	response := LazyLoadResponse{
		Data:        vehicles,
		Page:        page,
		PerPage:     perPage,
		Total:       total,
		TotalPages:  totalPages,
		HasNext:     hasNext,
		HasPrevious: hasPrevious,
		LoadTime:    fmt.Sprintf("%.2fms", float64(time.Since(start).Nanoseconds())/1000000),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// studentsAPIHandler provides paginated API access to students
func studentsAPIHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// Check authentication
	user := getUserFromSession(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse query parameters
	pageStr := r.URL.Query().Get("page")
	perPageStr := r.URL.Query().Get("per_page")
	driverFilter := r.URL.Query().Get("driver")
	status := r.URL.Query().Get("status")
	search := r.URL.Query().Get("search")

	config := DefaultLazyLoadConfig()
	page := 1
	perPage := config.DefaultPageSize

	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	if perPageStr != "" {
		if pp, err := strconv.Atoi(perPageStr); err == nil && pp > 0 && pp <= config.MaxPageSize {
			perPage = pp
		}
	}

	// Build query with filters
	query := `
		SELECT student_id, name, locations, phone_number, alt_phone_number, 
		       guardian, pickup_time, dropoff_time, position_number, 
		       route_id, driver, active, created_at
		FROM students WHERE 1=1
	`
	args := []interface{}{}
	argIndex := 1

	// Apply driver filter
	if driverFilter != "" {
		query += fmt.Sprintf(" AND driver = $%d", argIndex)
		args = append(args, driverFilter)
		argIndex++
	} else if user.Role == "driver" {
		// Drivers only see their own students
		query += fmt.Sprintf(" AND driver = $%d", argIndex)
		args = append(args, user.Username)
		argIndex++
	}

	// Apply status filter
	if status == "active" {
		query += " AND active = true"
	} else if status == "inactive" {
		query += " AND active = false"
	}

	// Apply search filter
	if search != "" {
		query += fmt.Sprintf(" AND (name ILIKE $%d OR guardian ILIKE $%d)", argIndex, argIndex+1)
		searchTerm := "%" + search + "%"
		args = append(args, searchTerm, searchTerm)
		argIndex += 2
	}

	// Get total count
	countQuery := "SELECT COUNT(*) FROM (" + query + ") as count_query"
	var total int
	if err := db.Get(&total, countQuery, args...); err != nil {
		LogError("Failed to get total count for students", err)
		http.Error(w, "Failed to get total count", http.StatusInternalServerError)
		return
	}

	// Add pagination to main query
	offset := (page - 1) * perPage
	query += fmt.Sprintf(" ORDER BY position_number, name LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, perPage, offset)

	// Get paginated data
	var students []Student
	if err := db.Select(&students, query, args...); err != nil {
		LogError("Failed to load students", err)
		http.Error(w, "Failed to load students", http.StatusInternalServerError)
		return
	}

	// Calculate pagination metadata
	totalPages := (total + perPage - 1) / perPage
	hasNext := page < totalPages
	hasPrevious := page > 1

	response := LazyLoadResponse{
		Data:        students,
		Page:        page,
		PerPage:     perPage,
		Total:       total,
		TotalPages:  totalPages,
		HasNext:     hasNext,
		HasPrevious: hasPrevious,
		LoadTime:    fmt.Sprintf("%.2fms", float64(time.Since(start).Nanoseconds())/1000000),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// driverLogsAPIHandler provides paginated API access to driver logs
func driverLogsAPIHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// Check authentication
	user := getUserFromSession(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse query parameters
	pageStr := r.URL.Query().Get("page")
	perPageStr := r.URL.Query().Get("per_page")
	driverFilter := r.URL.Query().Get("driver")
	busID := r.URL.Query().Get("bus_id")
	dateFrom := r.URL.Query().Get("date_from")
	dateTo := r.URL.Query().Get("date_to")
	period := r.URL.Query().Get("period")

	config := DefaultLazyLoadConfig()
	page := 1
	perPage := config.DefaultPageSize

	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	if perPageStr != "" {
		if pp, err := strconv.Atoi(perPageStr); err == nil && pp > 0 && pp <= config.MaxPageSize {
			perPage = pp
		}
	}

	// Build query with filters
	query := `
		SELECT id, driver, bus_id, route_id, date, period, 
		       departure_time, arrival_time, begin_mileage, end_mileage, 
		       mileage, attendance, notes, created_at
		FROM driver_logs WHERE 1=1
	`
	args := []interface{}{}
	argIndex := 1

	// Apply driver filter
	if driverFilter != "" {
		query += fmt.Sprintf(" AND driver = $%d", argIndex)
		args = append(args, driverFilter)
		argIndex++
	} else if user.Role == "driver" {
		// Drivers only see their own logs
		query += fmt.Sprintf(" AND driver = $%d", argIndex)
		args = append(args, user.Username)
		argIndex++
	}

	// Apply bus filter
	if busID != "" {
		query += fmt.Sprintf(" AND bus_id = $%d", argIndex)
		args = append(args, busID)
		argIndex++
	}

	// Apply date range filters
	if dateFrom != "" {
		query += fmt.Sprintf(" AND date >= $%d", argIndex)
		args = append(args, dateFrom)
		argIndex++
	}

	if dateTo != "" {
		query += fmt.Sprintf(" AND date <= $%d", argIndex)
		args = append(args, dateTo)
		argIndex++
	}

	// Apply period filter
	if period != "" {
		query += fmt.Sprintf(" AND period = $%d", argIndex)
		args = append(args, period)
		argIndex++
	}

	// Get total count
	countQuery := "SELECT COUNT(*) FROM (" + query + ") as count_query"
	var total int
	if err := db.Get(&total, countQuery, args...); err != nil {
		LogError("Failed to get total count for driver logs", err)
		http.Error(w, "Failed to get total count", http.StatusInternalServerError)
		return
	}

	// Add pagination to main query
	offset := (page - 1) * perPage
	query += fmt.Sprintf(" ORDER BY date DESC, created_at DESC LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, perPage, offset)

	// Get paginated data
	var logs []DriverLog
	if err := db.Select(&logs, query, args...); err != nil {
		LogError("Failed to load driver logs", err)
		http.Error(w, "Failed to load logs", http.StatusInternalServerError)
		return
	}

	// Calculate pagination metadata
	totalPages := (total + perPage - 1) / perPage
	hasNext := page < totalPages
	hasPrevious := page > 1

	response := LazyLoadResponse{
		Data:        logs,
		Page:        page,
		PerPage:     perPage,
		Total:       total,
		TotalPages:  totalPages,
		HasNext:     hasNext,
		HasPrevious: hasPrevious,
		LoadTime:    fmt.Sprintf("%.2fms", float64(time.Since(start).Nanoseconds())/1000000),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// busesAPIHandler provides paginated API access to buses
func busesAPIHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// Check authentication
	user := getUserFromSession(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse query parameters
	pageStr := r.URL.Query().Get("page")
	perPageStr := r.URL.Query().Get("per_page")
	status := r.URL.Query().Get("status")
	search := r.URL.Query().Get("search")

	config := DefaultLazyLoadConfig()
	page := 1
	perPage := config.DefaultPageSize

	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	if perPageStr != "" {
		if pp, err := strconv.Atoi(perPageStr); err == nil && pp > 0 && pp <= config.MaxPageSize {
			perPage = pp
		}
	}

	// Build query with filters
	query := `
		SELECT bus_id, status, model, capacity, oil_status, tire_status, 
		       maintenance_notes, current_mileage, last_oil_change, 
		       last_tire_rotation, created_at
		FROM buses WHERE 1=1
	`
	args := []interface{}{}
	argIndex := 1

	// Apply status filter
	if status != "" {
		query += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, status)
		argIndex++
	}

	// Apply search filter
	if search != "" {
		query += fmt.Sprintf(" AND (bus_id ILIKE $%d OR model ILIKE $%d)", argIndex, argIndex+1)
		searchTerm := "%" + search + "%"
		args = append(args, searchTerm, searchTerm)
		argIndex += 2
	}

	// Get total count
	countQuery := "SELECT COUNT(*) FROM (" + query + ") as count_query"
	var total int
	if err := db.Get(&total, countQuery, args...); err != nil {
		LogError("Failed to get total count for buses", err)
		http.Error(w, "Failed to get total count", http.StatusInternalServerError)
		return
	}

	// Add pagination to main query
	offset := (page - 1) * perPage
	query += fmt.Sprintf(" ORDER BY bus_id LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, perPage, offset)

	// Get paginated data
	var buses []Bus
	if err := db.Select(&buses, query, args...); err != nil {
		LogError("Failed to load buses", err)
		http.Error(w, "Failed to load buses", http.StatusInternalServerError)
		return
	}

	// Calculate pagination metadata
	totalPages := (total + perPage - 1) / perPage
	hasNext := page < totalPages
	hasPrevious := page > 1

	response := LazyLoadResponse{
		Data:        buses,
		Page:        page,
		PerPage:     perPage,
		Total:       total,
		TotalPages:  totalPages,
		HasNext:     hasNext,
		HasPrevious: hasPrevious,
		LoadTime:    fmt.Sprintf("%.2fms", float64(time.Since(start).Nanoseconds())/1000000),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
