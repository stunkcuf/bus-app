package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

// ExportTemplate represents an Excel template configuration
type ExportTemplate struct {
	Name         string
	ImportType   ImportType
	Description  string
	Headers      []string
	SampleData   [][]string
	Instructions string
}

// getExportTemplates returns all available export templates
func getExportTemplates() []ExportTemplate {
	return []ExportTemplate{
		{
			Name:        "Mileage Import Template",
			ImportType:  ImportTypeMileage,
			Description: "Template for importing vehicle mileage data",
			Headers:     []string{"Vehicle ID", "Beginning Mileage", "Ending Mileage", "Date", "Notes"},
			SampleData: [][]string{
				{"BUS001", "45000", "45250", "2025-01-01", "Regular route"},
				{"BUS002", "32100", "32400", "2025-01-01", "Extended route"},
				{"VEH003", "15000", "15150", "2025-01-01", "Office errands"},
			},
			Instructions: "Instructions:\n1. Vehicle ID must match existing vehicles in the system\n2. Mileage values must be positive numbers\n3. Ending mileage must be greater than beginning mileage\n4. Date format: YYYY-MM-DD\n5. Notes field is optional",
		},
		{
			Name:        "ECSE Student Import Template",
			ImportType:  ImportTypeECSE,
			Description: "Template for importing Early Childhood Special Education students",
			Headers:     []string{"Name", "Date of Birth", "Phone", "Address", "IEP Status", "Speech Therapy", "Occupational Therapy", "Physical Therapy"},
			SampleData: [][]string{
				{"John Doe", "2020-03-15", "(555) 123-4567", "123 Main St, Anytown, ST 12345", "Yes", "Yes", "No", "No"},
				{"Jane Smith", "2019-08-22", "(555) 987-6543", "456 Oak Ave, Somewhere, ST 54321", "Yes", "No", "Yes", "Yes"},
				{"Bobby Johnson", "2020-11-30", "(555) 555-5555", "789 Pine Rd, Elsewhere, ST 99999", "No", "No", "No", "No"},
			},
			Instructions: "Instructions:\n1. Name: Full name of the student (required)\n2. Date of Birth: Format YYYY-MM-DD or MM/DD/YYYY (required)\n3. Phone: Primary contact number (required)\n4. Address: Full street address\n5. Service fields: Use Yes/No or Y/N\n6. All students must be age 0-21",
		},
		{
			Name:        "Student Import Template",
			ImportType:  ImportTypeStudent,
			Description: "Template for importing general student roster",
			Headers:     []string{"Name", "Grade", "Address", "Phone", "Guardian", "Pickup Time", "Dropoff Time"},
			SampleData: [][]string{
				{"Alice Brown", "3", "321 Elm St, Town, ST 11111", "(555) 111-2222", "Mary Brown", "7:30 AM", "3:45 PM"},
				{"Charlie Davis", "5", "654 Maple Dr, City, ST 22222", "(555) 333-4444", "David Davis", "7:45 AM", "4:00 PM"},
				{"Emma Wilson", "K", "987 Cedar Ln, Village, ST 33333", "(555) 666-7777", "Sarah Wilson", "8:00 AM", "3:30 PM"},
			},
			Instructions: "Instructions:\n1. Name: Full name of the student (required)\n2. Grade: K, PK, or 1-12 (required)\n3. Address: Full street address (required)\n4. Phone: Primary contact number (required)\n5. Guardian: Parent or guardian name\n6. Times: Use AM/PM format (e.g., 7:30 AM)",
		},
		{
			Name:        "Vehicle Import Template",
			ImportType:  ImportTypeVehicle,
			Description: "Template for importing vehicle fleet information",
			Headers:     []string{"Vehicle ID", "Year", "Make", "Model", "VIN", "License Plate", "Status"},
			SampleData: [][]string{
				{"BUS001", "2020", "Blue Bird", "Vision", "1BAKBCKA0LF123456", "ABC-1234", "active"},
				{"BUS002", "2019", "Thomas", "Saf-T-Liner C2", "1T7HT4B25K1234567", "XYZ-5678", "active"},
				{"VEH001", "2021", "Ford", "Transit", "1FTBW2XM7MKA12345", "DEF-9012", "maintenance"},
			},
			Instructions: "Instructions:\n1. Vehicle ID: Unique identifier (required)\n2. Year: 4-digit year (required)\n3. Make: Vehicle manufacturer (required)\n4. Model: Vehicle model (required)\n5. VIN: 17-character Vehicle Identification Number\n6. License Plate: Current registration\n7. Status: active, maintenance, or out_of_service",
		},
	}
}

// exportTemplateHandler handles template download requests
func exportTemplateHandler(w http.ResponseWriter, r *http.Request) {
	importType := ImportType(r.URL.Query().Get("type"))
	if importType == "" {
		// Show template selection page
		renderTemplate(w, r, "export_templates.html", map[string]interface{}{
			"Templates": getExportTemplates(),
		})
		return
	}

	// Find the requested template
	var template *ExportTemplate
	for _, t := range getExportTemplates() {
		if t.ImportType == importType {
			template = &t
			break
		}
	}

	if template == nil {
		SendError(w, ErrNotFound("Template not found"))
		return
	}

	// Create Excel file
	f := excelize.NewFile()

	// Set up the main data sheet
	sheetName := "Data"
	f.SetSheetName("Sheet1", sheetName)

	// Style for headers
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold:  true,
			Size:  12,
			Color: "#FFFFFF",
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"#4472C4"},
			Pattern: 1,
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 1},
			{Type: "right", Color: "000000", Style: 1},
			{Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 1},
		},
	})

	// Style for sample data
	dataStyle, _ := f.NewStyle(&excelize.Style{
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"#E7E6E6"},
			Pattern: 1,
		},
		Font: &excelize.Font{
			Italic: true,
			Color:  "#666666",
		},
		Border: []excelize.Border{
			{Type: "left", Color: "CCCCCC", Style: 1},
			{Type: "right", Color: "CCCCCC", Style: 1},
			{Type: "top", Color: "CCCCCC", Style: 1},
			{Type: "bottom", Color: "CCCCCC", Style: 1},
		},
	})

	// Write headers
	for i, header := range template.Headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheetName, cell, header)
		f.SetCellStyle(sheetName, cell, cell, headerStyle)
	}

	// Write sample data
	for rowIdx, row := range template.SampleData {
		for colIdx, value := range row {
			cell, _ := excelize.CoordinatesToCellName(colIdx+1, rowIdx+2)
			f.SetCellValue(sheetName, cell, value)
			f.SetCellStyle(sheetName, cell, cell, dataStyle)
		}
	}

	// Auto-size columns
	for i := range template.Headers {
		col, _ := excelize.ColumnNumberToName(i + 1)
		f.SetColWidth(sheetName, col, col, 20)
	}

	// Add instructions sheet
	instructionsSheet := "Instructions"
	f.NewSheet(instructionsSheet)

	// Title style
	titleStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold: true,
			Size: 16,
		},
	})

	// Write instructions
	f.SetCellValue(instructionsSheet, "A1", template.Name)
	f.SetCellStyle(instructionsSheet, "A1", "A1", titleStyle)

	f.SetCellValue(instructionsSheet, "A3", "Description:")
	f.SetCellValue(instructionsSheet, "B3", template.Description)

	f.SetCellValue(instructionsSheet, "A5", "Instructions:")

	// Split instructions by line
	lines := strings.Split(template.Instructions, "\n")
	for i, line := range lines {
		f.SetCellValue(instructionsSheet, fmt.Sprintf("A%d", 6+i), line)
	}

	// Add metadata
	f.SetCellValue(instructionsSheet, "A20", "Template Information:")
	f.SetCellValue(instructionsSheet, "A21", fmt.Sprintf("Created: %s", time.Now().Format("2006-01-02")))
	f.SetCellValue(instructionsSheet, "A22", "Version: 1.0")
	f.SetCellValue(instructionsSheet, "A23", "System: Fleet Management System")

	// Set column width for instructions
	f.SetColWidth(instructionsSheet, "A", "A", 20)
	f.SetColWidth(instructionsSheet, "B", "B", 60)

	// Add data validation where applicable
	switch importType {
	case ImportTypeECSE, ImportTypeStudent:
		// Add Yes/No validation for boolean fields
		for i := 4; i <= 7; i++ { // Columns E-H for ECSE
			col, _ := excelize.ColumnNumberToName(i + 1)
			validation := &excelize.DataValidation{
				Type:         "list",
				Formula1:     "\"Yes,No\"",
				ShowDropDown: true,
			}
			f.AddDataValidation(fmt.Sprintf("%s2:%s1000", col, col), validation)
		}

	case ImportTypeVehicle:
		// Add status validation
		validation := &excelize.DataValidation{
			Type:         "list",
			Formula1:     "\"active,maintenance,out_of_service\"",
			ShowDropDown: true,
		}
		f.AddDataValidation("G2:G1000", validation)
	}

	// Set active sheet
	f.SetActiveSheet(0)

	// Set response headers
	filename := fmt.Sprintf("%s_template_%s.xlsx",
		strings.ToLower(string(importType)),
		time.Now().Format("20060102"))

	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))

	// Write file to response
	if err := f.Write(w); err != nil {
		LogRequest(r).Error("Failed to write Excel template", err)
		SendError(w, ErrInternal("Failed to generate template", err))
	}
}

// exportDataHandler handles data export requests
func exportDataHandler(w http.ResponseWriter, r *http.Request) {
	exportType := r.URL.Query().Get("type")
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "xlsx"
	}

	// Get date range
	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")

	if startDate == "" || endDate == "" {
		// Default to current month
		now := time.Now()
		startDate = now.Format("2006-01-02")
		endDate = now.AddDate(0, 1, -1).Format("2006-01-02")
	}

	switch exportType {
	case "mileage":
		exportMileageData(w, r, startDate, endDate, format)
	case "students":
		exportStudentData(w, r, format)
	case "vehicles":
		exportVehicleData(w, r, format)
	case "maintenance":
		exportMaintenanceData(w, r, startDate, endDate, format)
	default:
		SendError(w, ErrBadRequest("Invalid export type"))
	}
}

// Helper function to create consistent Excel styling
func createExcelStyles(f *excelize.File) (headerStyle, dataStyle int) {
	headerStyle, _ = f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold: true,
			Size: 11,
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"#D9E1F2"},
			Pattern: 1,
		},
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 1},
			{Type: "right", Color: "000000", Style: 1},
			{Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 1},
		},
	})

	dataStyle, _ = f.NewStyle(&excelize.Style{
		Border: []excelize.Border{
			{Type: "left", Color: "CCCCCC", Style: 1},
			{Type: "right", Color: "CCCCCC", Style: 1},
			{Type: "top", Color: "CCCCCC", Style: 1},
			{Type: "bottom", Color: "CCCCCC", Style: 1},
		},
	})

	return headerStyle, dataStyle
}
