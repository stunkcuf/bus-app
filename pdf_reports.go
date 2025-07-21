package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jung-kurt/gofpdf"
)

// PDFReportGenerator handles PDF report generation
type PDFReportGenerator struct {
	pdf    *gofpdf.Fpdf
	config PDFConfig
}

// PDFConfig contains PDF generation settings
type PDFConfig struct {
	PageSize    string
	Orientation string
	FontFamily  string
	FontSize    float64
	MarginLeft  float64
	MarginTop   float64
	MarginRight float64
	LineHeight  float64
}

// DefaultPDFConfig returns default PDF configuration
func DefaultPDFConfig() PDFConfig {
	return PDFConfig{
		PageSize:    "Letter",
		Orientation: "P", // Portrait
		FontFamily:  "Arial",
		FontSize:    10,
		MarginLeft:  10,
		MarginTop:   10,
		MarginRight: 10,
		LineHeight:  5,
	}
}

// NewPDFReportGenerator creates a new PDF report generator
func NewPDFReportGenerator(config PDFConfig) *PDFReportGenerator {
	pdf := gofpdf.New(config.Orientation, "mm", config.PageSize, "")
	pdf.SetMargins(config.MarginLeft, config.MarginTop, config.MarginRight)
	pdf.SetAutoPageBreak(true, 10)

	return &PDFReportGenerator{
		pdf:    pdf,
		config: config,
	}
}

// GenerateMileageReport creates a PDF mileage report
func (p *PDFReportGenerator) GenerateMileageReport(startDate, endDate string) (*bytes.Buffer, error) {
	p.pdf.AddPage()

	// Header
	p.addHeader("Fleet Mileage Report", fmt.Sprintf("%s to %s", startDate, endDate))

	// Query mileage data
	query := `
		SELECT m.vehicle_id, m.month, m.year, m.beginning_mileage, 
		       m.ending_mileage, m.total_miles, m.driver
		FROM mileage_reports m
		WHERE (m.year || '-' || LPAD(m.month::text, 2, '0') || '-01')::date 
		      BETWEEN $1::date AND $2::date
		ORDER BY m.year, m.month, m.vehicle_id
	`

	rows, err := db.Query(query, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Table headers
	headers := []string{"Vehicle ID", "Month/Year", "Start Miles", "End Miles", "Total Miles", "Driver"}
	widths := []float64{30, 30, 30, 30, 30, 40}
	p.addTableHeader(headers, widths)

	// Table data
	totalMiles := 0
	rowCount := 0

	for rows.Next() {
		var vehicleID, driver string
		var month, year, beginMileage, endMileage, totalMilesRow int

		err := rows.Scan(&vehicleID, &month, &year, &beginMileage, &endMileage, &totalMilesRow, &driver)
		if err != nil {
			continue
		}

		monthYear := fmt.Sprintf("%02d/%d", month, year)
		data := []string{
			vehicleID,
			monthYear,
			fmt.Sprintf("%d", beginMileage),
			fmt.Sprintf("%d", endMileage),
			fmt.Sprintf("%d", totalMilesRow),
			driver,
		}

		p.addTableRow(data, widths)
		totalMiles += totalMilesRow
		rowCount++
	}

	// Summary
	p.pdf.Ln(10)
	p.pdf.SetFont(p.config.FontFamily, "B", 12)
	p.pdf.Cell(0, 10, fmt.Sprintf("Total Vehicles: %d", rowCount))
	p.pdf.Ln(10)
	p.pdf.Cell(0, 10, fmt.Sprintf("Total Miles: %s", formatNumber(totalMiles)))

	// Footer
	p.addFooter()

	// Generate PDF
	var buf bytes.Buffer
	err = p.pdf.Output(&buf)
	if err != nil {
		return nil, err
	}

	return &buf, nil
}

// GenerateStudentReport creates a PDF student roster report
func (p *PDFReportGenerator) GenerateStudentReport() (*bytes.Buffer, error) {
	p.pdf.AddPage()

	// Header
	p.addHeader("Student Roster Report", time.Now().Format("January 2, 2006"))

	// Query student data
	query := `
		SELECT s.name, s.grade, s.address, s.phone, s.guardian_name,
		       s.pickup_time, s.dropoff_time, s.driver, r.name as route_name
		FROM students s
		LEFT JOIN routes r ON s.route_id = r.id
		WHERE s.active = true
		ORDER BY r.name, s.name
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Group by route
	currentRoute := ""
	studentCount := 0
	widths := []float64{40, 20, 30, 40, 25, 25}

	for rows.Next() {
		var name, grade, address, phone, guardian, pickupTime, dropoffTime, driver string
		var route *string

		err := rows.Scan(&name, &grade, &address, &phone, &guardian,
			&pickupTime, &dropoffTime, &driver, &route)
		if err != nil {
			continue
		}

		routeName := "Unassigned"
		if route != nil {
			routeName = *route
		}

		// New route section
		if routeName != currentRoute {
			if currentRoute != "" {
				p.pdf.Ln(5)
			}
			currentRoute = routeName

			p.pdf.SetFont(p.config.FontFamily, "B", 14)
			p.pdf.SetFillColor(240, 240, 240)
			p.pdf.CellFormat(0, 8, fmt.Sprintf("Route: %s", routeName), "", 1, "L", true, 0, "")
			p.pdf.Ln(2)

			// Table headers for this route
			headers := []string{"Name", "Grade", "Phone", "Guardian", "Pickup", "Dropoff"}
			p.addTableHeader(headers, widths)
		}

		// Student data
		data := []string{
			name,
			grade,
			phone,
			guardian,
			pickupTime,
			dropoffTime,
		}

		p.addTableRow(data, widths)
		studentCount++
	}

	// Summary
	p.pdf.Ln(10)
	p.pdf.SetFont(p.config.FontFamily, "B", 12)
	p.pdf.Cell(0, 10, fmt.Sprintf("Total Students: %d", studentCount))

	// Footer
	p.addFooter()

	// Generate PDF
	var buf bytes.Buffer
	err = p.pdf.Output(&buf)
	if err != nil {
		return nil, err
	}

	return &buf, nil
}

// GenerateMaintenanceReport creates a PDF maintenance report
func (p *PDFReportGenerator) GenerateMaintenanceReport(startDate, endDate string) (*bytes.Buffer, error) {
	p.pdf.AddPage()

	// Header
	p.addHeader("Maintenance Report", fmt.Sprintf("%s to %s", startDate, endDate))

	// Bus maintenance section
	p.pdf.SetFont(p.config.FontFamily, "B", 14)
	p.pdf.Cell(0, 10, "Bus Maintenance")
	p.pdf.Ln(10)

	// Query bus maintenance
	busQuery := `
		SELECT date, bus_id, maintenance_type, description, mileage, cost, performed_by
		FROM bus_maintenance_logs
		WHERE date BETWEEN $1::date AND $2::date
		ORDER BY date DESC
	`

	rows, err := db.Query(busQuery, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Table headers
	headers := []string{"Date", "Bus ID", "Type", "Description", "Mileage", "Cost", "By"}
	widths := []float64{25, 20, 25, 50, 25, 25, 20}
	p.addTableHeader(headers, widths)

	totalCost := 0.0
	maintenanceCount := 0

	for rows.Next() {
		var date, busID, mainType, description, performedBy string
		var mileage int
		var cost float64

		err := rows.Scan(&date, &busID, &mainType, &description, &mileage, &cost, &performedBy)
		if err != nil {
			continue
		}

		data := []string{
			date[:10], // Just date, no time
			busID,
			mainType,
			truncateString(description, 30),
			fmt.Sprintf("%d", mileage),
			fmt.Sprintf("$%.2f", cost),
			performedBy,
		}

		p.addTableRow(data, widths)
		totalCost += cost
		maintenanceCount++
	}

	// Vehicle maintenance section
	p.pdf.Ln(10)
	p.pdf.SetFont(p.config.FontFamily, "B", 14)
	p.pdf.Cell(0, 10, "Vehicle Maintenance")
	p.pdf.Ln(10)

	// Query vehicle maintenance
	vehicleQuery := `
		SELECT date, vehicle_id, maintenance_type, description, mileage, cost, performed_by
		FROM vehicle_maintenance_logs
		WHERE date BETWEEN $1::date AND $2::date
		ORDER BY date DESC
	`

	rows, err = db.Query(vehicleQuery, startDate, endDate)
	if err == nil {
		defer rows.Close()

		// Add headers again
		p.addTableHeader(headers, widths)

		for rows.Next() {
			var date, vehicleID, mainType, description, performedBy string
			var mileage int
			var cost float64

			err := rows.Scan(&date, &vehicleID, &mainType, &description, &mileage, &cost, &performedBy)
			if err != nil {
				continue
			}

			data := []string{
				date[:10],
				vehicleID,
				mainType,
				truncateString(description, 30),
				fmt.Sprintf("%d", mileage),
				fmt.Sprintf("$%.2f", cost),
				performedBy,
			}

			p.addTableRow(data, widths)
			totalCost += cost
			maintenanceCount++
		}
	}

	// Summary
	p.pdf.Ln(10)
	p.pdf.SetFont(p.config.FontFamily, "B", 12)
	p.pdf.Cell(0, 10, fmt.Sprintf("Total Maintenance Items: %d", maintenanceCount))
	p.pdf.Ln(10)
	p.pdf.Cell(0, 10, fmt.Sprintf("Total Cost: $%s", formatNumber(int(totalCost))))

	// Footer
	p.addFooter()

	// Generate PDF
	var buf bytes.Buffer
	err = p.pdf.Output(&buf)
	if err != nil {
		return nil, err
	}

	return &buf, nil
}

// GenerateFleetStatusReport creates a PDF fleet status report
func (p *PDFReportGenerator) GenerateFleetStatusReport() (*bytes.Buffer, error) {
	p.pdf.AddPage()

	// Header
	p.addHeader("Fleet Status Report", time.Now().Format("January 2, 2006"))

	// Buses section
	p.pdf.SetFont(p.config.FontFamily, "B", 14)
	p.pdf.Cell(0, 10, "School Buses")
	p.pdf.Ln(10)

	// Query buses
	busQuery := `
		SELECT b.bus_id, b.model, b.status, b.current_mileage, 
		       b.last_oil_change, b.last_tire_service,
		       r.name as route_name, ra.driver
		FROM buses b
		LEFT JOIN route_assignments ra ON b.bus_id = ra.bus_id
		LEFT JOIN routes r ON ra.route_id = r.id
		ORDER BY b.bus_id
	`

	rows, err := db.Query(busQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Table headers
	headers := []string{"Bus ID", "Model", "Status", "Mileage", "Route", "Driver"}
	widths := []float64{25, 35, 25, 30, 35, 40}
	p.addTableHeader(headers, widths)

	busCount := 0
	activeBuses := 0

	for rows.Next() {
		var busID, model, status string
		var mileage, oilChange, tireService int
		var route, driver *string

		err := rows.Scan(&busID, &model, &status, &mileage, &oilChange, &tireService, &route, &driver)
		if err != nil {
			continue
		}

		routeName := "Unassigned"
		if route != nil {
			routeName = *route
		}

		driverName := "Unassigned"
		if driver != nil {
			driverName = *driver
		}

		data := []string{
			busID,
			model,
			status,
			fmt.Sprintf("%d", mileage),
			routeName,
			driverName,
		}

		p.addTableRow(data, widths)
		busCount++
		if status == "active" {
			activeBuses++
		}
	}

	// Company vehicles section
	p.pdf.Ln(10)
	p.pdf.SetFont(p.config.FontFamily, "B", 14)
	p.pdf.Cell(0, 10, "Company Vehicles")
	p.pdf.Ln(10)

	// Query vehicles
	vehicleQuery := `
		SELECT vehicle_id, year, model, license, status, current_mileage
		FROM vehicles
		ORDER BY vehicle_id
	`

	rows, err = db.Query(vehicleQuery)
	if err == nil {
		defer rows.Close()

		// Table headers for vehicles
		vHeaders := []string{"Vehicle ID", "Year", "Model", "License", "Status", "Mileage"}
		vWidths := []float64{30, 20, 40, 30, 30, 30}
		p.addTableHeader(vHeaders, vWidths)

		vehicleCount := 0

		for rows.Next() {
			var vehicleID, model, license, status string
			var year, mileage int

			err := rows.Scan(&vehicleID, &year, &model, &license, &status, &mileage)
			if err != nil {
				continue
			}

			data := []string{
				vehicleID,
				fmt.Sprintf("%d", year),
				model,
				license,
				status,
				fmt.Sprintf("%d", mileage),
			}

			p.addTableRow(data, vWidths)
			vehicleCount++
		}

		// Summary
		p.pdf.Ln(10)
		p.pdf.SetFont(p.config.FontFamily, "B", 12)
		p.pdf.Cell(0, 10, fmt.Sprintf("Total Buses: %d (Active: %d)", busCount, activeBuses))
		p.pdf.Ln(10)
		p.pdf.Cell(0, 10, fmt.Sprintf("Total Company Vehicles: %d", vehicleCount))
	}

	// Footer
	p.addFooter()

	// Generate PDF
	var buf bytes.Buffer
	err = p.pdf.Output(&buf)
	if err != nil {
		return nil, err
	}

	return &buf, nil
}

// GenerateCustomReport creates a PDF from custom report data
func (p *PDFReportGenerator) GenerateCustomReport(title string, headers []string, data [][]string) (*bytes.Buffer, error) {
	p.pdf.AddPage()

	// Header
	p.addHeader(title, time.Now().Format("January 2, 2006"))

	// Calculate column widths
	pageWidth := 190.0 // Letter width minus margins
	colCount := len(headers)
	colWidth := pageWidth / float64(colCount)
	widths := make([]float64, colCount)
	for i := range widths {
		widths[i] = colWidth
	}

	// Table
	p.addTableHeader(headers, widths)

	for _, row := range data {
		p.addTableRow(row, widths)
	}

	// Summary
	p.pdf.Ln(10)
	p.pdf.SetFont(p.config.FontFamily, "B", 12)
	p.pdf.Cell(0, 10, fmt.Sprintf("Total Records: %d", len(data)))

	// Footer
	p.addFooter()

	// Generate PDF
	var buf bytes.Buffer
	err := p.pdf.Output(&buf)
	if err != nil {
		return nil, err
	}

	return &buf, nil
}

// Helper methods

func (p *PDFReportGenerator) addHeader(title, subtitle string) {
	// Logo/Company name
	p.pdf.SetFont(p.config.FontFamily, "B", 16)
	p.pdf.Cell(0, 10, "Fleet Management System")
	p.pdf.Ln(10)

	// Report title
	p.pdf.SetFont(p.config.FontFamily, "B", 20)
	p.pdf.Cell(0, 10, title)
	p.pdf.Ln(10)

	// Subtitle/date range
	p.pdf.SetFont(p.config.FontFamily, "", 12)
	p.pdf.SetTextColor(100, 100, 100)
	p.pdf.Cell(0, 10, subtitle)
	p.pdf.SetTextColor(0, 0, 0)
	p.pdf.Ln(15)
}

func (p *PDFReportGenerator) addTableHeader(headers []string, widths []float64) {
	p.pdf.SetFont(p.config.FontFamily, "B", 10)
	p.pdf.SetFillColor(240, 240, 240)

	for i, header := range headers {
		p.pdf.CellFormat(widths[i], 7, header, "1", 0, "C", true, 0, "")
	}
	p.pdf.Ln(-1)
}

func (p *PDFReportGenerator) addTableRow(data []string, widths []float64) {
	p.pdf.SetFont(p.config.FontFamily, "", 9)

	for i, cell := range data {
		p.pdf.CellFormat(widths[i], 6, cell, "1", 0, "L", false, 0, "")
	}
	p.pdf.Ln(-1)
}

func (p *PDFReportGenerator) addFooter() {
	p.pdf.SetY(-15)
	p.pdf.SetFont(p.config.FontFamily, "I", 8)
	p.pdf.SetTextColor(128, 128, 128)
	p.pdf.CellFormat(0, 10, fmt.Sprintf("Generated on %s - Page %d",
		time.Now().Format("January 2, 2006 at 3:04 PM"),
		p.pdf.PageNo()), "", 0, "C", false, 0, "")
	p.pdf.SetTextColor(0, 0, 0)
}

// Utility functions

func formatNumber(n int) string {
	str := fmt.Sprintf("%d", n)
	if n < 1000 {
		return str
	}

	var result []string
	for i := len(str); i > 0; i -= 3 {
		start := i - 3
		if start < 0 {
			start = 0
		}
		result = append([]string{str[start:i]}, result...)
	}
	return strings.Join(result, ",")
}

// HTTP Handlers

// pdfReportHandler handles PDF report generation requests
func pdfReportHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	reportType := r.URL.Query().Get("type")
	if reportType == "" {
		http.Error(w, "Report type required", http.StatusBadRequest)
		return
	}

	generator := NewPDFReportGenerator(DefaultPDFConfig())
	var buf *bytes.Buffer
	var err error
	var filename string

	switch reportType {
	case "mileage":
		startDate := r.URL.Query().Get("start_date")
		endDate := r.URL.Query().Get("end_date")
		if startDate == "" || endDate == "" {
			http.Error(w, "Start and end dates required", http.StatusBadRequest)
			return
		}

		buf, err = generator.GenerateMileageReport(startDate, endDate)
		filename = fmt.Sprintf("mileage_report_%s.pdf", time.Now().Format("20060102"))

	case "students":
		buf, err = generator.GenerateStudentReport()
		filename = fmt.Sprintf("student_roster_%s.pdf", time.Now().Format("20060102"))

	case "maintenance":
		startDate := r.URL.Query().Get("start_date")
		endDate := r.URL.Query().Get("end_date")
		if startDate == "" || endDate == "" {
			http.Error(w, "Start and end dates required", http.StatusBadRequest)
			return
		}

		buf, err = generator.GenerateMaintenanceReport(startDate, endDate)
		filename = fmt.Sprintf("maintenance_report_%s.pdf", time.Now().Format("20060102"))

	case "fleet":
		buf, err = generator.GenerateFleetStatusReport()
		filename = fmt.Sprintf("fleet_status_%s.pdf", time.Now().Format("20060102"))

	default:
		http.Error(w, "Invalid report type", http.StatusBadRequest)
		return
	}

	if err != nil {
		log.Printf("Failed to generate PDF report: %v", err)
		http.Error(w, "Failed to generate report", http.StatusInternalServerError)
		return
	}

	// Set headers for PDF download
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.Header().Set("Content-Length", strconv.Itoa(buf.Len()))

	// Write PDF to response
	w.Write(buf.Bytes())
}

// pdfCustomReportHandler handles custom report PDF generation
func pdfCustomReportHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if user.Role != "manager" {
		http.Error(w, "Manager access required", http.StatusForbidden)
		return
	}

	// Parse request body
	var request struct {
		Title   string     `json:"title"`
		Headers []string   `json:"headers"`
		Data    [][]string `json:"data"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request data", http.StatusBadRequest)
		return
	}

	if request.Title == "" || len(request.Headers) == 0 {
		http.Error(w, "Title and headers required", http.StatusBadRequest)
		return
	}

	// Generate PDF
	generator := NewPDFReportGenerator(DefaultPDFConfig())
	buf, err := generator.GenerateCustomReport(request.Title, request.Headers, request.Data)
	if err != nil {
		log.Printf("Failed to generate custom PDF report: %v", err)
		http.Error(w, "Failed to generate report", http.StatusInternalServerError)
		return
	}

	// Set headers for PDF download
	filename := fmt.Sprintf("custom_report_%s.pdf", time.Now().Format("20060102_150405"))
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.Header().Set("Content-Length", strconv.Itoa(buf.Len()))

	// Write PDF to response
	w.Write(buf.Bytes())
}
