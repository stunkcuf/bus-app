package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

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
		SELECT id, vehicle_number, make, model, year, vin, 
		       license_plate, status, mileage, last_service, created_at
		FROM fleet_vehicles WHERE 1=1
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
