package main

import (
	"bytes"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jung-kurt/gofpdf/v2"
	"github.com/xuri/excelize/v2"
	"github.com/jmoiron/sqlx"
)

// ReportGenerator handles generation of reports in various formats
type ReportGenerator struct {
	db *sqlx.DB
}

// NewReportGenerator creates a new report generator instance
func NewReportGenerator(db *sqlx.DB) *ReportGenerator {
	return &ReportGenerator{db: db}
}

// GenerateMaintenanceReport generates a maintenance report in the requested format
func (rg *ReportGenerator) GenerateMaintenanceReport(vehicleID int, format string) ([]byte, string, error) {
	// Fetch maintenance records
	query := `
		SELECT m.*, v.vehicle_id as bus_number, v.year, 'Vehicle' as make, v.model 
		FROM maintenance_records m
		JOIN vehicles v ON m.vehicle_id = v.vehicle_id
		WHERE m.vehicle_id = $1
		ORDER BY m.service_date DESC`
	
	var records []struct {
		MaintenanceRecord
		BusNumber string         `db:"bus_number"`
		Year      sql.NullString `db:"year"`
		Make      string         `db:"make"`
		Model     sql.NullString `db:"model"`
	}
	
	err := rg.db.Select(&records, query, vehicleID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to fetch maintenance records: %w", err)
	}
	
	if len(records) == 0 {
		return nil, "", fmt.Errorf("no maintenance records found for vehicle ID %d", vehicleID)
	}
	
	yearStr := ""
	if records[0].Year.Valid {
		yearStr = records[0].Year.String
	}
	modelStr := ""
	if records[0].Model.Valid {
		modelStr = records[0].Model.String
	}
	vehicleInfo := fmt.Sprintf("%s - %s %s %s", records[0].BusNumber, yearStr, records[0].Make, modelStr)
	
	switch format {
	case "pdf":
		data, err := rg.generateMaintenancePDF(records, vehicleInfo)
		return data, "application/pdf", err
	case "excel", "xlsx":
		data, err := rg.generateMaintenanceExcel(records, vehicleInfo)
		return data, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", err
	default:
		return nil, "", fmt.Errorf("unsupported format: %s", format)
	}
}

// generateMaintenancePDF creates a PDF report of maintenance records
func (rg *ReportGenerator) generateMaintenancePDF(records []struct {
	MaintenanceRecord
	BusNumber string         `db:"bus_number"`
	Year      sql.NullString `db:"year"`
	Make      string         `db:"make"`
	Model     sql.NullString `db:"model"`
}, vehicleInfo string) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetAutoPageBreak(true, 15)
	
	// Add page
	pdf.AddPage()
	
	// Title
	pdf.SetFont("Arial", "B", 16)
	pdf.Cell(0, 10, "Maintenance History Report")
	pdf.Ln(10)
	
	// Vehicle info
	pdf.SetFont("Arial", "", 12)
	pdf.Cell(0, 8, "Vehicle: "+vehicleInfo)
	pdf.Ln(5)
	pdf.Cell(0, 8, fmt.Sprintf("Generated: %s", time.Now().Format("January 2, 2006")))
	pdf.Ln(5)
	pdf.Cell(0, 8, fmt.Sprintf("Total Records: %d", len(records)))
	pdf.Ln(10)
	
	// Table header
	pdf.SetFont("Arial", "B", 10)
	pdf.SetFillColor(230, 230, 230)
	pdf.CellFormat(30, 8, "Date", "1", 0, "C", true, 0, "")
	pdf.CellFormat(35, 8, "Category", "1", 0, "C", true, 0, "")
	pdf.CellFormat(25, 8, "Mileage", "1", 0, "C", true, 0, "")
	pdf.CellFormat(100, 8, "Description", "1", 0, "C", true, 0, "")
	pdf.Ln(-1)
	
	// Table content
	pdf.SetFont("Arial", "", 9)
	pdf.SetFillColor(255, 255, 255)
	
	for i, record := range records {
		// Alternate row colors
		if i%2 == 1 {
			pdf.SetFillColor(245, 245, 245)
		} else {
			pdf.SetFillColor(255, 255, 255)
		}
		
		// Date
		dateStr := "--"
		if record.Date.Valid {
			dateStr = record.Date.Time.Format("2006-01-02")
		}
		pdf.CellFormat(30, 7, dateStr, "1", 0, "C", true, 0, "")
		
		// Category/Work Description
		description := "--"
		if record.WorkDescription.Valid {
			description = record.WorkDescription.String
			if len(description) > 20 {
				description = description[:20] + "..."
			}
		}
		pdf.CellFormat(35, 7, description, "1", 0, "C", true, 0, "")
		
		// Mileage
		mileageStr := "--"
		if record.Mileage.Valid && record.Mileage.Int32 > 0 {
			mileageStr = fmt.Sprintf("%d", record.Mileage.Int32)
		}
		pdf.CellFormat(25, 7, mileageStr, "1", 0, "C", true, 0, "")
		
		// Notes - handle long text
		notes := "--"
		if record.WorkDescription.Valid {
			notes = record.WorkDescription.String
		}
		if len(notes) > 60 {
			notes = notes[:57] + "..."
		}
		pdf.CellFormat(100, 7, notes, "1", 0, "L", true, 0, "")
		pdf.Ln(-1)
		
		// Check if we need a new page
		if pdf.GetY() > 265 {
			pdf.AddPage()
			
			// Repeat header
			pdf.SetFont("Arial", "B", 10)
			pdf.SetFillColor(230, 230, 230)
			pdf.CellFormat(30, 8, "Date", "1", 0, "C", true, 0, "")
			pdf.CellFormat(35, 8, "Category", "1", 0, "C", true, 0, "")
			pdf.CellFormat(25, 8, "Mileage", "1", 0, "C", true, 0, "")
			pdf.CellFormat(100, 8, "Description", "1", 0, "C", true, 0, "")
			pdf.Ln(-1)
			pdf.SetFont("Arial", "", 9)
		}
	}
	
	// Summary statistics
	pdf.Ln(10)
	pdf.SetFont("Arial", "B", 12)
	pdf.Cell(0, 8, "Summary Statistics")
	pdf.Ln(8)
	
	// Calculate statistics
	categoryCount := make(map[string]int)
	totalCost := 0.0
	var minMileage, maxMileage int
	
	for i, record := range records {
		categoryCount[record.WorkDescription.String]++
		if record.Cost.Valid {
			// Parse cost string to float
			costStr := strings.ReplaceAll(record.Cost.String, "$", "")
			costStr = strings.ReplaceAll(costStr, ",", "")
			if costVal, err := strconv.ParseFloat(costStr, 64); err == nil && costVal > 0 {
				totalCost += costVal
			}
		}
		if i == 0 || (record.Mileage.Valid && record.Mileage.Int32 > 0 && int(record.Mileage.Int32) < minMileage) {
			minMileage = int(record.Mileage.Int32)
		}
		if record.Mileage.Valid && int(record.Mileage.Int32) > maxMileage {
			maxMileage = int(record.Mileage.Int32)
		}
	}
	
	pdf.SetFont("Arial", "", 10)
	y := pdf.GetY()
	
	// Left column
	pdf.SetXY(10, y)
	pdf.Cell(0, 6, "Maintenance by Category:")
	pdf.Ln(6)
	
	for category, count := range categoryCount {
		catName := strings.ReplaceAll(category, "_", " ")
		catName = strings.Title(catName)
		pdf.Cell(0, 5, fmt.Sprintf("  • %s: %d", catName, count))
		pdf.Ln(5)
	}
	
	// Right column
	pdf.SetXY(110, y)
	pdf.Cell(0, 6, "Mileage Range:")
	pdf.SetXY(110, y+6)
	pdf.Cell(0, 5, fmt.Sprintf("  Min: %d", minMileage))
	pdf.SetXY(110, y+11)
	pdf.Cell(0, 5, fmt.Sprintf("  Max: %d", maxMileage))
	
	if totalCost > 0 {
		pdf.SetXY(110, y+16)
		pdf.Cell(0, 5, fmt.Sprintf("  Total Cost: $%.2f", totalCost))
	}
	
	// Footer
	pdf.SetY(-15)
	pdf.SetFont("Arial", "I", 8)
	pdf.Cell(0, 10, fmt.Sprintf("Page %d", pdf.PageNo()))
	
	var buf bytes.Buffer
	err := pdf.Output(&buf)
	return buf.Bytes(), err
}

// generateMaintenanceExcel creates an Excel report of maintenance records
func (rg *ReportGenerator) generateMaintenanceExcel(records []struct {
	MaintenanceRecord
	BusNumber string         `db:"bus_number"`
	Year      sql.NullString `db:"year"`
	Make      string         `db:"make"`
	Model     sql.NullString `db:"model"`
}, vehicleInfo string) ([]byte, error) {
	f := excelize.NewFile()
	sheet := "Maintenance History"
	
	// Rename default sheet
	f.SetSheetName("Sheet1", sheet)
	
	// Title and header
	f.SetCellValue(sheet, "A1", "Maintenance History Report")
	f.SetCellValue(sheet, "A2", "Vehicle: "+vehicleInfo)
	f.SetCellValue(sheet, "A3", "Generated: "+time.Now().Format("January 2, 2006"))
	f.SetCellValue(sheet, "A4", fmt.Sprintf("Total Records: %d", len(records)))
	
	// Column headers
	headers := []string{"Date", "Category", "Mileage", "Description", "Cost", "Vendor", "Invoice #"}
	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 6)
		f.SetCellValue(sheet, cell, header)
	}
	
	// Style for headers
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#E0E0E0"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})
	f.SetCellStyle(sheet, "A6", "G6", headerStyle)
	
	// Title style
	titleStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 14},
	})
	f.SetCellStyle(sheet, "A1", "A1", titleStyle)
	
	// Data rows
	for i, record := range records {
		row := i + 7
		
		// Date
		dateStr := "--"
		if record.Date.Valid {
			dateStr = record.Date.Time.Format("2006-01-02")
		}
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), dateStr)
		
		// Category
		category := strings.ReplaceAll(record.WorkDescription.String, "_", " ")
		category = strings.Title(category)
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), category)
		
		// Mileage
		if record.Mileage.Valid && record.Mileage.Int32 > 0 {
			f.SetCellValue(sheet, fmt.Sprintf("C%d", row), record.Mileage)
		} else {
			f.SetCellValue(sheet, fmt.Sprintf("C%d", row), "--")
		}
		
		// Description
		f.SetCellValue(sheet, fmt.Sprintf("D%d", row), record.WorkDescription.String)
		
		// Cost
		if record.Cost.Valid {
			f.SetCellValue(sheet, fmt.Sprintf("E%d", row), record.Cost.String)
		}
		
		// PO Number
		if record.PONumber.Valid {
			f.SetCellValue(sheet, fmt.Sprintf("F%d", row), record.PONumber.String)
		}
		
	}
	
	// Format cost column as currency
	currencyStyle, _ := f.NewStyle(&excelize.Style{
		NumFmt: 2, // $#,##0.00
	})
	f.SetCellStyle(sheet, fmt.Sprintf("E7"), fmt.Sprintf("E%d", len(records)+6), currencyStyle)
	
	// Auto-fit columns
	cols := []string{"A", "B", "C", "D", "E", "F", "G"}
	widths := []float64{12, 15, 10, 50, 12, 20, 15}
	for i, col := range cols {
		f.SetColWidth(sheet, col, col, widths[i])
	}
	
	// Add filters
	f.AutoFilter(sheet, fmt.Sprintf("A6:G%d", len(records)+6), nil)
	
	// Summary sheet
	summarySheet := "Summary"
	f.NewSheet(summarySheet)
	
	// Summary title
	f.SetCellValue(summarySheet, "A1", "Maintenance Summary")
	f.SetCellStyle(summarySheet, "A1", "A1", titleStyle)
	
	// Category breakdown
	f.SetCellValue(summarySheet, "A3", "Category")
	f.SetCellValue(summarySheet, "B3", "Count")
	f.SetCellValue(summarySheet, "C3", "Total Cost")
	f.SetCellStyle(summarySheet, "A3", "C3", headerStyle)
	
	categoryStats := make(map[string]struct {
		count int
		cost  float64
	})
	
	for _, record := range records {
		stat := categoryStats[record.WorkDescription.String]
		stat.count++
		if record.Cost.Valid {
			// Parse cost string to float
			costStr := strings.ReplaceAll(record.Cost.String, "$", "")
			costStr = strings.ReplaceAll(costStr, ",", "")
			if costVal, err := strconv.ParseFloat(costStr, 64); err == nil {
				stat.cost += costVal
			}
		}
		categoryStats[record.WorkDescription.String] = stat
	}
	
	row := 4
	for category, stat := range categoryStats {
		catName := strings.ReplaceAll(category, "_", " ")
		catName = strings.Title(catName)
		f.SetCellValue(summarySheet, fmt.Sprintf("A%d", row), catName)
		f.SetCellValue(summarySheet, fmt.Sprintf("B%d", row), stat.count)
		f.SetCellValue(summarySheet, fmt.Sprintf("C%d", row), stat.cost)
		row++
	}
	
	// Format summary cost column
	f.SetCellStyle(summarySheet, "C4", fmt.Sprintf("C%d", row-1), currencyStyle)
	
	// Auto-fit summary columns
	f.SetColWidth(summarySheet, "A", "A", 20)
	f.SetColWidth(summarySheet, "B", "B", 10)
	f.SetColWidth(summarySheet, "C", "C", 15)
	
	// Save to buffer
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, err
	}
	
	return buf.Bytes(), nil
}

// GenerateFleetReport generates a fleet overview report
func (rg *ReportGenerator) GenerateFleetReport(format string) ([]byte, string, error) {
	// Fetch fleet data
	query := `
		SELECT v.*, 
		       COUNT(DISTINCT mr.id) as maintenance_count,
		       COALESCE(MAX(mr.mileage), v.current_mileage) as latest_mileage,
		       COALESCE(SUM(mr.cost), 0) as total_maintenance_cost
		FROM vehicles v
		LEFT JOIN maintenance_records mr ON v.vehicle_id = mr.vehicle_id
		GROUP BY v.vehicle_id, v.model, v.description, v.year, v.tire_size, v.license, 
		         v.oil_status, v.tire_status, v.status, v.maintenance_notes, 
		         v.serial_number, v.base, v.service_interval, v.current_mileage, 
		         v.last_oil_change, v.last_service_date, v.next_service_due, 
		         v.created_at, v.updated_at
		ORDER BY v.vehicle_id`
	
	var buses []struct {
		Vehicle
		MaintenanceCount     int     `db:"maintenance_count"`
		LatestMileage        int     `db:"latest_mileage"`
		TotalMaintenanceCost float64 `db:"total_maintenance_cost"`
	}
	
	err := rg.db.Select(&buses, query)
	if err != nil {
		return nil, "", fmt.Errorf("failed to fetch fleet data: %w", err)
	}
	
	switch format {
	case "pdf":
		data, err := rg.generateFleetPDF(buses)
		return data, "application/pdf", err
	case "excel", "xlsx":
		data, err := rg.generateFleetExcel(buses)
		return data, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", err
	default:
		return nil, "", fmt.Errorf("unsupported format: %s", format)
	}
}

// generateFleetPDF creates a PDF report of the fleet
func (rg *ReportGenerator) generateFleetPDF(buses []struct {
	Vehicle
	MaintenanceCount     int     `db:"maintenance_count"`
	LatestMileage        int     `db:"latest_mileage"`
	TotalMaintenanceCost float64 `db:"total_maintenance_cost"`
}) ([]byte, error) {
	pdf := gofpdf.New("L", "mm", "A4", "") // Landscape orientation
	pdf.SetAutoPageBreak(true, 15)
	
	// Add page
	pdf.AddPage()
	
	// Title
	pdf.SetFont("Arial", "B", 16)
	pdf.Cell(0, 10, "Fleet Overview Report")
	pdf.Ln(10)
	
	// Report info
	pdf.SetFont("Arial", "", 12)
	pdf.Cell(0, 8, fmt.Sprintf("Generated: %s", time.Now().Format("January 2, 2006")))
	pdf.Ln(5)
	pdf.Cell(0, 8, fmt.Sprintf("Total Buses: %d", len(buses)))
	pdf.Ln(10)
	
	// Table header
	pdf.SetFont("Arial", "B", 9)
	pdf.SetFillColor(230, 230, 230)
	pdf.CellFormat(20, 8, "Bus #", "1", 0, "C", true, 0, "")
	pdf.CellFormat(25, 8, "Year", "1", 0, "C", true, 0, "")
	pdf.CellFormat(30, 8, "Make", "1", 0, "C", true, 0, "")
	pdf.CellFormat(30, 8, "Model", "1", 0, "C", true, 0, "")
	pdf.CellFormat(25, 8, "Status", "1", 0, "C", true, 0, "")
	pdf.CellFormat(30, 8, "Mileage", "1", 0, "C", true, 0, "")
	pdf.CellFormat(25, 8, "Services", "1", 0, "C", true, 0, "")
	pdf.CellFormat(35, 8, "Maint. Cost", "1", 0, "C", true, 0, "")
	pdf.CellFormat(50, 8, "Assignment", "1", 0, "C", true, 0, "")
	pdf.Ln(-1)
	
	// Table content
	pdf.SetFont("Arial", "", 8)
	
	var totalMaintCost float64
	activeCount := 0
	
	for i, bus := range buses {
		// Alternate row colors
		if i%2 == 1 {
			pdf.SetFillColor(245, 245, 245)
		} else {
			pdf.SetFillColor(255, 255, 255)
		}
		
		// Vehicle ID
		pdf.CellFormat(20, 7, bus.VehicleID, "1", 0, "C", true, 0, "")
		// Year
		yearStr := "--"
		if bus.Year.Valid {
			yearStr = bus.Year.String
		}
		pdf.CellFormat(25, 7, yearStr, "1", 0, "C", true, 0, "")
		// Make - not available in Vehicle, use model instead
		modelStr := "--"
		if bus.Model.Valid {
			modelStr = bus.Model.String
		}
		pdf.CellFormat(30, 7, modelStr, "1", 0, "C", true, 0, "")
		pdf.CellFormat(30, 7, modelStr, "1", 0, "C", true, 0, "")
		
		// Status with color
		statusStr := "--"
		if bus.Status.Valid {
			statusStr = strings.Title(bus.Status.String)
		}
		pdf.CellFormat(25, 7, statusStr, "1", 0, "C", true, 0, "")
		
		pdf.CellFormat(30, 7, fmt.Sprintf("%d", bus.LatestMileage), "1", 0, "C", true, 0, "")
		pdf.CellFormat(25, 7, strconv.Itoa(bus.MaintenanceCount), "1", 0, "C", true, 0, "")
		pdf.CellFormat(35, 7, fmt.Sprintf("$%.2f", bus.TotalMaintenanceCost), "1", 0, "C", true, 0, "")
		
		// Assignment - not available in Vehicle table
		pdf.CellFormat(50, 7, "N/A", "1", 0, "C", true, 0, "")
		pdf.Ln(-1)
		
		totalMaintCost += bus.TotalMaintenanceCost
		if bus.Status.Valid && bus.Status.String == "active" {
			activeCount++
		}
	}
	
	// Summary
	pdf.Ln(10)
	pdf.SetFont("Arial", "B", 12)
	pdf.Cell(0, 8, "Fleet Summary")
	pdf.Ln(8)
	
	pdf.SetFont("Arial", "", 10)
	pdf.Cell(60, 6, fmt.Sprintf("Active Buses: %d", activeCount))
	pdf.Cell(60, 6, fmt.Sprintf("Inactive Buses: %d", len(buses)-activeCount))
	pdf.Cell(0, 6, fmt.Sprintf("Total Maintenance Cost: $%.2f", totalMaintCost))
	
	// Footer
	pdf.SetY(-15)
	pdf.SetFont("Arial", "I", 8)
	pdf.Cell(0, 10, fmt.Sprintf("Page %d", pdf.PageNo()))
	
	var buf bytes.Buffer
	err := pdf.Output(&buf)
	return buf.Bytes(), err
}

// generateFleetExcel creates an Excel report of the fleet
func (rg *ReportGenerator) generateFleetExcel(buses []struct {
	Vehicle
	MaintenanceCount     int     `db:"maintenance_count"`
	LatestMileage        int     `db:"latest_mileage"`
	TotalMaintenanceCost float64 `db:"total_maintenance_cost"`
}) ([]byte, error) {
	f := excelize.NewFile()
	sheet := "Fleet Overview"
	
	// Rename default sheet
	f.SetSheetName("Sheet1", sheet)
	
	// Title
	f.SetCellValue(sheet, "A1", "Fleet Overview Report")
	f.SetCellValue(sheet, "A2", "Generated: "+time.Now().Format("January 2, 2006"))
	f.SetCellValue(sheet, "A3", fmt.Sprintf("Total Buses: %d", len(buses)))
	
	// Headers
	headers := []string{"Bus #", "Year", "Make", "Model", "VIN", "Status", "Current Mileage", 
		"Maintenance Count", "Total Maint. Cost", "Driver Assigned", "Route", "Insurance Exp", "Registration Exp"}
	
	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 5)
		f.SetCellValue(sheet, cell, header)
	}
	
	// Style for headers
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#E0E0E0"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})
	f.SetCellStyle(sheet, "A5", "M5", headerStyle)
	
	// Title style
	titleStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 14},
	})
	f.SetCellStyle(sheet, "A1", "A1", titleStyle)
	
	// Data rows
	for i, bus := range buses {
		row := i + 6
		
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), bus.VehicleID)
		yearStr := "--"
		if bus.Year.Valid {
			yearStr = bus.Year.String
		}
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), yearStr)
		modelStr := "--"
		if bus.Model.Valid {
			modelStr = bus.Model.String
		}
		f.SetCellValue(sheet, fmt.Sprintf("C%d", row), modelStr)
		f.SetCellValue(sheet, fmt.Sprintf("D%d", row), modelStr)
		// VIN not available in Vehicle table
		f.SetCellValue(sheet, fmt.Sprintf("E%d", row), "N/A")
		statusStr := "--"
		if bus.Status.Valid {
			statusStr = strings.Title(bus.Status.String)
		}
		f.SetCellValue(sheet, fmt.Sprintf("F%d", row), statusStr)
		f.SetCellValue(sheet, fmt.Sprintf("G%d", row), bus.LatestMileage)
		f.SetCellValue(sheet, fmt.Sprintf("H%d", row), bus.MaintenanceCount)
		f.SetCellValue(sheet, fmt.Sprintf("I%d", row), bus.TotalMaintenanceCost)
		
		// Assignment not available in Vehicle table
		f.SetCellValue(sheet, fmt.Sprintf("J%d", row), "N/A")
		
		// Route not available in Vehicle table
		f.SetCellValue(sheet, fmt.Sprintf("K%d", row), "N/A")
		
		// Insurance and Registration not available in Vehicle table
		f.SetCellValue(sheet, fmt.Sprintf("L%d", row), "N/A")
		f.SetCellValue(sheet, fmt.Sprintf("M%d", row), "N/A")
	}
	
	// Format cost column as currency
	currencyStyle, _ := f.NewStyle(&excelize.Style{
		NumFmt: 2, // $#,##0.00
	})
	f.SetCellStyle(sheet, "I6", fmt.Sprintf("I%d", len(buses)+5), currencyStyle)
	
	// Conditional formatting for status
	// Skip for now - needs proper ConditionalFormatOptions struct
	
	// Auto-fit columns
	cols := []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L", "M"}
	widths := []float64{10, 8, 15, 15, 20, 10, 15, 18, 18, 15, 12, 15, 15}
	for i, col := range cols {
		f.SetColWidth(sheet, col, col, widths[i])
	}
	
	// Add filters
	f.AutoFilter(sheet, fmt.Sprintf("A5:M%d", len(buses)+5), nil)
	
	// Save to buffer
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, err
	}
	
	return buf.Bytes(), nil
}