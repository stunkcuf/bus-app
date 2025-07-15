package main

import (
	"fmt"
	"log"
	"mime/multipart"
	"regexp"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

// processECSEExcelFile processes transportation Excel files with student lists
func processECSEExcelFile(file multipart.File, filename string) (int, error) {
	// Read Excel file
	f, err := excelize.OpenReader(file)
	if err != nil {
		return 0, fmt.Errorf("failed to open Excel file: %w", err)
	}
	defer f.Close()

	// Get sheet list
	sheets := f.GetSheetList()
	log.Printf("ECSE Excel file has %d sheets: %v", len(sheets), sheets)

	totalImported := 0

	// Process each sheet (each represents a school district)
	for _, sheet := range sheets {
		log.Printf("\n=== Processing ECSE sheet: '%s' ===", sheet)
		
		imported, err := processTransportationSheet(f, sheet)
		if err != nil {
			log.Printf("Error processing sheet %s: %v", sheet, err)
			continue
		}
		
		totalImported += imported
		log.Printf("Sheet '%s' - Imported: %d student records", sheet, imported)
	}

	return totalImported, nil
}

// processTransportationSheet processes a transportation report sheet
func processTransportationSheet(f *excelize.File, sheetName string) (int, error) {
	// Get all rows
	rows, err := f.GetRows(sheetName)
	if err != nil {
		return 0, fmt.Errorf("failed to get rows: %w", err)
	}

	if len(rows) == 0 {
		log.Printf("Sheet '%s' is empty", sheetName)
		return 0, nil
	}

	// Extract school district from sheet name
	schoolDistrict := strings.TrimSpace(sheetName)
	if strings.HasSuffix(schoolDistrict, "-SD") {
		schoolDistrict = strings.TrimSuffix(schoolDistrict, "-SD") + " School District"
	}

	// Find the month/year
	monthYear := ""
	for _, row := range rows {
		if len(row) > 0 && strings.Contains(strings.ToUpper(row[0]), "MONTH:") {
			parts := strings.Split(row[0], ":")
			if len(parts) > 1 {
				monthYear = strings.TrimSpace(parts[1])
			}
			break
		}
	}

	log.Printf("Processing %s for %s", schoolDistrict, monthYear)

	// Find the transportation table header row
	headerRowIndex := -1
	for i, row := range rows {
		if containsTransportHeaders(row) {
			headerRowIndex = i
			log.Printf("Found header row at index %d", i)
			break
		}
	}

	if headerRowIndex == -1 {
		log.Printf("No transportation table found in sheet '%s'", sheetName)
		// Log first 10 rows to debug
		log.Printf("First 10 rows of sheet:")
		for i := 0; i < len(rows) && i < 10; i++ {
			log.Printf("Row %d: %v", i, rows[i])
		}
		return 0, nil
	}

	// Parse routes from the table
	routes := parseTransportationRoutes(rows, headerRowIndex)
	log.Printf("Found %d routes", len(routes))

	// Find student names section (usually after the table)
	// Calculate where student section might start
	studentsStartIndex := headerRowIndex + len(routes) + 2
	
	// Look for student section more intelligently
	for i := studentsStartIndex; i < len(rows) && i < studentsStartIndex + 10; i++ {
		if i < len(rows) && len(rows[i]) > 0 {
			firstCell := strings.TrimSpace(rows[i][0])
			if isProgramHeader(firstCell) || looksLikeStudentName(firstCell) {
				studentsStartIndex = i
				log.Printf("Found student section starting at row %d", i)
				break
			}
		}
	}
	
	students := extractStudentNames(rows, studentsStartIndex)
	log.Printf("Found %d students total", len(students))

	// Import students
	imported := 0
	for i, studentData := range students {
		student := ECSEStudent{
			StudentID:              fmt.Sprintf("ECSE-%s-%s-%04d", getDistrictCode(schoolDistrict), time.Now().Format("0102"), i+1),
			FirstName:              studentData.FirstName,
			LastName:               studentData.LastName,
			Grade:                  "ECSE", // Default grade for ECSE students
			EnrollmentStatus:       "Active",
			TransportationRequired: studentData.HasTransportation,
			BusRoute:               studentData.Route,
			Notes:                  fmt.Sprintf("Program: %s, District: %s, Month: %s", studentData.Program, schoolDistrict, monthYear),
			CreatedAt:              time.Now(),
			UpdatedAt:              time.Now(),
		}

		err := saveECSEStudent(student)
		if err != nil {
			log.Printf("Error saving student %s: %v", student.StudentID, err)
			continue
		}
		imported++
	}

	return imported, nil
}

// containsTransportHeaders checks if a row contains transportation table headers
func containsTransportHeaders(row []string) bool {
	requiredHeaders := []string{"Center", "Driver", "Students", "Cost", "Miles"}
	foundCount := 0
	
	rowText := strings.Join(row, " ")
	for _, header := range requiredHeaders {
		if strings.Contains(rowText, header) {
			foundCount++
		}
	}
	
	return foundCount >= 3
}

// TransportationRoute represents a route from the Excel file
type TransportationRoute struct {
	Center               string
	Driver               string
	TotalStudents        int
	ECSEStudents         int
	CostPerMile          float64
	MilesPerRoute        float64
	CostPerRoute         float64
	RoutePercentage      int
	DistrictResponsibility string
	DistrictCostPerRoute float64
	NumberOfRoutes       int
	TotalDistrictCost    float64
}

// parseTransportationRoutes extracts route data from the table
func parseTransportationRoutes(rows [][]string, headerIndex int) []TransportationRoute {
	var routes []TransportationRoute
	
	// Process rows after header until we hit an empty row or student section
	for i := headerIndex + 1; i < len(rows); i++ {
		row := rows[i]
		
		// Stop if we hit an empty row or a row with less than expected columns
		if len(row) < 8 || strings.TrimSpace(row[0]) == "" {
			break
		}
		
		// Skip if this looks like a student name row
		if looksLikeStudentName(row[0]) {
			break
		}
		
		route := TransportationRoute{
			Center: strings.TrimSpace(row[0]),
		}
		
		// Extract driver name (column 1)
		if len(row) > 1 {
			route.Driver = strings.TrimSpace(row[1])
		}
		
		// Extract ECSE students count (usually column 3)
		if len(row) > 3 {
			fmt.Sscanf(row[3], "%d", &route.ECSEStudents)
		}
		
		// Log the route for debugging
		log.Printf("Route: %s, Driver: %s, ECSE Students: %d", route.Center, route.Driver, route.ECSEStudents)
		
		routes = append(routes, route)
	}
	
	return routes
}

// StudentInfo holds parsed student information
type StudentInfo struct {
	Name              string
	FirstName         string
	LastName          string
	Program           string
	Route             string
	HasTransportation bool
}

// extractStudentNames finds and extracts student names from the sheet
func extractStudentNames(rows [][]string, startIndex int) []StudentInfo {
	var students []StudentInfo
	currentProgram := ""
	
	// Look for program headers and student names
	for i := startIndex; i < len(rows); i++ {
		row := rows[i]
		
		// Skip empty rows
		if len(row) == 0 || (len(row) == 1 && strings.TrimSpace(row[0]) == "") {
			continue
		}
		
		// Check if this is a program header (e.g., "VICTORY SQUARE 2", "HCSR 3")
		firstCell := strings.TrimSpace(row[0])
		if isProgramHeader(firstCell) {
			currentProgram = firstCell
			log.Printf("Found program: %s", currentProgram)
			continue
		}
		
		// Extract student names from the row
		for _, cell := range row {
			name := strings.TrimSpace(cell)
			if name != "" && looksLikeStudentName(name) {
				studentInfo := parseStudentName(name)
				studentInfo.Program = currentProgram
				
				// Determine if student has transportation based on program
				studentInfo.HasTransportation = !strings.Contains(strings.ToUpper(currentProgram), "NO TRANSPORT")
				studentInfo.Route = currentProgram
				
				students = append(students, studentInfo)
			}
		}
	}
	
	return students
}

// isProgramHeader checks if a string is a program header
func isProgramHeader(text string) bool {
	// Common program patterns
	programPatterns := []string{
		"VICTORY SQUARE", "HCSR", "AWDC", "CWEL", "VS-", "RH-D",
		"PROGRAM", "CENTER", "ROUTE",
	}
	
	text = strings.ToUpper(text)
	
	// Check if it matches any pattern
	for _, pattern := range programPatterns {
		if strings.Contains(text, pattern) {
			return true
		}
	}
	
	// Also check if it's a short code that might be a program (e.g., "VS 2", "HCSR 3")
	if len(text) <= 20 && !looksLikeStudentName(text) && strings.Contains(text, " ") {
		// Check if it has a number at the end (common for program codes)
		parts := strings.Fields(text)
		if len(parts) >= 2 {
			lastPart := parts[len(parts)-1]
			if _, err := fmt.Sscanf(lastPart, "%d", new(int)); err == nil {
				return true
			}
		}
	}
	
	return false
}

// looksLikeStudentName checks if a string appears to be a student name
func looksLikeStudentName(text string) bool {
	// Skip if too short or too long
	if len(text) < 3 || len(text) > 50 {
		return false
	}
	
	// Skip if it's a number or contains certain keywords
	skipPatterns := []string{
		"$", "%", "TRANSPORT", "TOTAL", "COST", "MILES", "ROUTE",
		"DISTRICT", "CENTER", "DRIVER", "STUDENTS",
	}
	
	textUpper := strings.ToUpper(text)
	for _, pattern := range skipPatterns {
		if strings.Contains(textUpper, pattern) {
			return false
		}
	}
	
	// Check if it contains at least one space (likely first and last name)
	// or follows common name patterns
	return strings.Contains(text, " ") || isLikelyName(text)
}

// isLikelyName uses simple heuristics to determine if a string is likely a name
func isLikelyName(text string) bool {
	// Must start with a letter
	if len(text) == 0 || !isLetter(rune(text[0])) {
		return false
	}
	
	// Count letters vs non-letters
	letterCount := 0
	for _, r := range text {
		if isLetter(r) || r == ' ' || r == '-' || r == '\'' {
			letterCount++
		}
	}
	
	// At least 80% should be letters or acceptable characters
	return float64(letterCount)/float64(len(text)) > 0.8
}

// isLetter checks if a rune is a letter
func isLetter(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

// parseStudentName splits a full name into first and last name
func parseStudentName(fullName string) StudentInfo {
	info := StudentInfo{
		Name: fullName,
	}
	
	// Handle various name formats
	// Remove extra spaces
	fullName = regexp.MustCompile(`\s+`).ReplaceAllString(fullName, " ")
	fullName = strings.TrimSpace(fullName)
	
	parts := strings.Split(fullName, " ")
	
	if len(parts) >= 2 {
		// Assume first part is first name, last part is last name
		info.FirstName = parts[0]
		info.LastName = parts[len(parts)-1]
		
		// Handle middle names or complex last names
		if len(parts) > 2 {
			// Could be "First Middle Last" or "First Last-Part1 Last-Part2"
			// For now, take everything after first as last name
			info.LastName = strings.Join(parts[1:], " ")
		}
	} else if len(parts) == 1 {
		// Only one name part - use as last name
		info.LastName = parts[0]
		info.FirstName = "Unknown"
	}
	
	return info
}

// generateStudentID creates a unique student ID
func generateStudentID(name, district string) string {
	// Create initials from district
	districtCode := ""
	for _, word := range strings.Fields(district) {
		if len(word) > 0 {
			districtCode += string(word[0])
		}
	}
	if len(districtCode) > 3 {
		districtCode = districtCode[:3]
	}
	
	// Create a unique ID with timestamp and random component
	timestamp := time.Now().UnixNano() % 1000000
	return fmt.Sprintf("ECSE-%s-%06d", strings.ToUpper(districtCode), timestamp)
}

// getDistrictCode extracts a short code from district name
func getDistrictCode(district string) string {
	// Remove common suffixes
	district = strings.TrimSuffix(district, " School District")
	district = strings.TrimSuffix(district, "-SD")
	
	// Create initials
	districtCode := ""
	for _, word := range strings.Fields(district) {
		if len(word) > 0 {
			districtCode += string(word[0])
		}
	}
	
	// Ensure we have at least 3 characters
	if len(districtCode) < 3 && len(district) >= 3 {
		districtCode = strings.ToUpper(district[:3])
	} else if len(districtCode) > 3 {
		districtCode = districtCode[:3]
	}
	
	return strings.ToUpper(districtCode)
}

// saveECSEStudent saves a student to the database
func saveECSEStudent(student ECSEStudent) error {
	if db == nil {
		return fmt.Errorf("database not initialized")
	}

	// Use CAST to handle date type properly
	_, err := db.Exec(`
		INSERT INTO ecse_students (
			student_id, first_name, last_name, date_of_birth, grade,
			enrollment_status, iep_status, primary_disability, service_minutes,
			transportation_required, bus_route, parent_name, parent_phone,
			parent_email, address, city, state, zip_code, notes, created_at, updated_at
		) VALUES (
			$1, $2, $3, 
			CASE WHEN $4 = '' THEN NULL ELSE $4::DATE END, 
			$5, $6, 
			NULLIF($7, ''), NULLIF($8, ''), 
			$9, $10, 
			NULLIF($11, ''), NULLIF($12, ''), NULLIF($13, ''), NULLIF($14, ''), 
			NULLIF($15, ''), NULLIF($16, ''), NULLIF($17, ''), NULLIF($18, ''), 
			NULLIF($19, ''), 
			$20, $21
		)
		ON CONFLICT (student_id) DO UPDATE SET
			first_name = EXCLUDED.first_name,
			last_name = EXCLUDED.last_name,
			grade = EXCLUDED.grade,
			enrollment_status = EXCLUDED.enrollment_status,
			transportation_required = EXCLUDED.transportation_required,
			bus_route = EXCLUDED.bus_route,
			notes = EXCLUDED.notes,
			updated_at = EXCLUDED.updated_at
	`, student.StudentID, student.FirstName, student.LastName, student.DateOfBirth,
		student.Grade, student.EnrollmentStatus, student.IEPStatus, student.PrimaryDisability,
		student.ServiceMinutes, student.TransportationRequired, student.BusRoute,
		student.ParentName, student.ParentPhone, student.ParentEmail,
		student.Address, student.City, student.State, student.ZipCode, student.Notes,
		student.CreatedAt, student.UpdatedAt)

	return err
}
