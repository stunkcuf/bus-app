package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"
)

// maintenanceReportExportHandler handles export of maintenance reports in various formats
func maintenanceReportExportHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil {
		SendError(w, ErrUnauthorized("Not authenticated"))
		return
	}
	
	// Get vehicle ID from query params
	vehicleIDStr := r.URL.Query().Get("vehicle_id")
	if vehicleIDStr == "" {
		SendError(w, ErrBadRequest("Vehicle ID required"))
		return
	}
	
	vehicleID, err := strconv.Atoi(vehicleIDStr)
	if err != nil {
		SendError(w, ErrBadRequest("Invalid vehicle ID"))
		return
	}
	
	// Get format (default to PDF)
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "pdf"
	}
	
	// Initialize report generator
	reportGen := NewReportGenerator(db)
	
	// Generate report
	data, contentType, err := reportGen.GenerateMaintenanceReport(vehicleID, format)
	if err != nil {
		SendError(w, ErrInternal("Failed to generate report", err))
		return
	}
	
	// Get vehicle info for filename
	var busNumber string
	err = db.Get(&busNumber, "SELECT COALESCE(bus_number::text, 'vehicle_' || id::text) FROM vehicles WHERE id = $1", vehicleID)
	if err != nil {
		busNumber = "unknown"
	}
	
	// Set response headers
	filename := fmt.Sprintf("maintenance_report_%s_%s.%s", 
		busNumber, 
		time.Now().Format("20060102"), 
		format)
	
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	
	// Write the file
	w.Write(data)
}

// fleetReportExportHandler handles export of fleet overview reports
func fleetReportExportHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		SendError(w, ErrUnauthorized("Access denied"))
		return
	}
	
	// Get format (default to PDF)
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "pdf"
	}
	
	// Initialize report generator
	reportGen := NewReportGenerator(db)
	
	// Generate report
	data, contentType, err := reportGen.GenerateFleetReport(format)
	if err != nil {
		SendError(w, ErrInternal("Failed to generate report", err))
		return
	}
	
	// Set response headers
	filename := fmt.Sprintf("fleet_overview_%s.%s", 
		time.Now().Format("20060102"), 
		format)
	
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	
	// Write the file
	w.Write(data)
}

// analyticsReportExportHandler handles export of analytics reports
func analyticsReportExportHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		SendError(w, ErrUnauthorized("Access denied"))
		return
	}
	
	// Get report type and format
	reportType := r.URL.Query().Get("type")
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "pdf"
	}
	
	// Get date range
	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")
	
	// Generate appropriate report based on type
	var data []byte
	var contentType string
	var err error
	
	switch reportType {
	case "maintenance":
		data, contentType, err = generateMaintenanceAnalyticsReport(format, startDate, endDate)
	case "fuel":
		data, contentType, err = generateFuelAnalyticsReport(format, startDate, endDate)
	case "driver":
		data, contentType, err = generateDriverAnalyticsReport(format, startDate, endDate)
	case "route":
		data, contentType, err = generateRouteAnalyticsReport(format, startDate, endDate)
	default:
		// Generate comprehensive analytics report
		data, contentType, err = generateComprehensiveAnalyticsReport(format, startDate, endDate)
	}
	
	if err != nil {
		SendError(w, ErrInternal("Failed to generate report", err))
		return
	}
	
	// Set response headers
	filename := fmt.Sprintf("%s_analytics_%s.%s", 
		reportType,
		time.Now().Format("20060102"), 
		format)
	
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	
	// Write the file
	w.Write(data)
}

// Helper functions for specific analytics reports
func generateMaintenanceAnalyticsReport(format, startDate, endDate string) ([]byte, string, error) {
	// This would generate a maintenance-focused analytics report
	// For now, return a basic implementation
	if analyticsEngine == nil {
		return nil, "", fmt.Errorf("analytics engine not initialized")
	}
	
	analytics, err := analyticsEngine.GenerateFleetAnalytics()
	if err != nil {
		return nil, "", err
	}
	
	// For now, export as JSON until we implement specific PDF/Excel templates
	if format == "json" {
		data, err := json.MarshalIndent(analytics, "", "  ")
		return data, "application/json", err
	}
	
	return nil, "", fmt.Errorf("format %s not yet implemented for maintenance analytics", format)
}

func generateFuelAnalyticsReport(format, startDate, endDate string) ([]byte, string, error) {
	// This would generate a fuel-focused analytics report
	if analyticsEngine == nil {
		return nil, "", fmt.Errorf("analytics engine not initialized")
	}
	
	analytics, err := analyticsEngine.GenerateFleetAnalytics()
	if err != nil {
		return nil, "", err
	}
	
	// For now, export as JSON
	if format == "json" {
		data, err := json.MarshalIndent(analytics.FinancialMetrics, "", "  ")
		return data, "application/json", err
	}
	
	return nil, "", fmt.Errorf("format %s not yet implemented for fuel analytics", format)
}

func generateDriverAnalyticsReport(format, startDate, endDate string) ([]byte, string, error) {
	// This would generate a driver-focused analytics report
	if analyticsEngine == nil {
		return nil, "", fmt.Errorf("analytics engine not initialized")
	}
	
	analytics, err := analyticsEngine.GenerateFleetAnalytics()
	if err != nil {
		return nil, "", err
	}
	
	// For now, export as JSON
	if format == "json" {
		data, err := json.MarshalIndent(analytics.OperationalMetrics, "", "  ")
		return data, "application/json", err
	}
	
	return nil, "", fmt.Errorf("format %s not yet implemented for driver analytics", format)
}

func generateRouteAnalyticsReport(format, startDate, endDate string) ([]byte, string, error) {
	// This would generate a route-focused analytics report
	if analyticsEngine == nil {
		return nil, "", fmt.Errorf("analytics engine not initialized")
	}
	
	analytics, err := analyticsEngine.GenerateFleetAnalytics()
	if err != nil {
		return nil, "", err
	}
	
	// For now, export as JSON
	if format == "json" {
		data, err := json.MarshalIndent(analytics.EfficiencyMetrics, "", "  ")
		return data, "application/json", err
	}
	
	return nil, "", fmt.Errorf("format %s not yet implemented for route analytics", format)
}

func generateComprehensiveAnalyticsReport(format, startDate, endDate string) ([]byte, string, error) {
	// This would generate a comprehensive analytics report
	if analyticsEngine == nil {
		return nil, "", fmt.Errorf("analytics engine not initialized")
	}
	
	analytics, err := analyticsEngine.GenerateFleetAnalytics()
	if err != nil {
		return nil, "", err
	}
	
	// For now, export as JSON
	if format == "json" {
		data, err := json.MarshalIndent(analytics, "", "  ")
		return data, "application/json", err
	}
	
	// Use the existing export functionality for PDF/Excel
	filename, err := ExportAnalyticsReport(format)
	if err != nil {
		return nil, "", err
	}
	
	// Read the generated file
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, "", err
	}
	
	// Clean up the file
	os.Remove(filename)
	
	contentType := "application/octet-stream"
	if format == "pdf" {
		contentType = "application/pdf"
	} else if format == "excel" || format == "xlsx" {
		contentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	}
	
	return data, contentType, nil
}