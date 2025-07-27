package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/xuri/excelize/v2"
)

// FileAnalysisResult represents the result of analyzing an uploaded file
type FileAnalysisResult struct {
	FileID      string                   `json:"file_id"`
	FileName    string                   `json:"file_name"`
	FileSize    int64                    `json:"file_size"`
	Columns     []string                 `json:"columns"`
	RowCount    int                      `json:"row_count"`
	SampleData  []map[string]interface{} `json:"sample_data"`
	ImportType  string                   `json:"import_type"`
}

// ValidationResult represents the result of validating import data
type ValidationResult struct {
	TotalRecords   int                      `json:"total_records"`
	ValidRecords   int                      `json:"valid_records"`
	InvalidRecords int                      `json:"invalid_records"`
	Warnings       []string                 `json:"warnings"`
	Errors         []string                 `json:"errors"`
	Preview        []map[string]interface{} `json:"preview"`
}

// ImportResult represents the result of an import operation
type ImportResult struct {
	Total    int      `json:"total"`
	Imported int      `json:"imported"`
	Skipped  int      `json:"skipped"`
	Errors   int      `json:"errors"`
	Details  []string `json:"details"`
}

// TempFileStore stores temporary upload information
var TempFileStore = make(map[string]*FileAnalysisResult)

// enhancedImportAnalyzeHandler analyzes an uploaded Excel file with validation
func enhancedImportAnalyzeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check authentication
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		SendError(w, ErrUnauthorized("Access denied"))
		return
	}

	// Parse multipart form
	err := r.ParseMultipartForm(10 << 20) // 10 MB limit
	if err != nil {
		SendError(w, ErrValidation(fmt.Sprintf("File too large: %v", err)))
		return
	}

	// Get file
	file, header, err := r.FormFile("file")
	if err != nil {
		SendError(w, ErrValidation(fmt.Sprintf("No file uploaded: %v", err)))
		return
	}
	defer file.Close()

	// Get import type
	importType := r.FormValue("type")
	if importType == "" {
		SendError(w, ErrValidation("Import type not specified"))
		return
	}

	// Create temporary file
	tempDir := os.TempDir()
	fileID := uuid.New().String()
	tempPath := filepath.Join(tempDir, fileID+filepath.Ext(header.Filename))

	tempFile, err := os.Create(tempPath)
	if err != nil {
		SendError(w, ErrInternal("Failed to create temp file", err))
		return
	}
	defer tempFile.Close()

	// Copy file content
	_, err = io.Copy(tempFile, file)
	if err != nil {
		SendError(w, ErrInternal("Failed to save file", err))
		return
	}

	// Analyze Excel file
	xlsx, err := excelize.OpenFile(tempPath)
	if err != nil {
		SendError(w, ErrValidation(fmt.Sprintf("Invalid Excel file: %v", err)))
		return
	}
	defer xlsx.Close()

	// Get first sheet
	sheets := xlsx.GetSheetList()
	if len(sheets) == 0 {
		SendError(w, ErrValidation("Excel file has no sheets"))
		return
	}

	sheetName := sheets[0]
	rows, err := xlsx.GetRows(sheetName)
	if err != nil {
		SendError(w, ErrValidation(fmt.Sprintf("Failed to read Excel data: %v", err)))
		return
	}

	if len(rows) < 2 {
		SendError(w, ErrValidation("Excel file must have at least a header row and one data row"))
		return
	}

	// Get columns from first row
	columns := rows[0]

	// Get sample data (first 10 rows)
	sampleData := []map[string]interface{}{}
	for i := 1; i < len(rows) && i <= 10; i++ {
		rowData := make(map[string]interface{})
		for j, col := range columns {
			if j < len(rows[i]) {
				rowData[col] = rows[i][j]
			} else {
				rowData[col] = ""
			}
		}
		sampleData = append(sampleData, rowData)
	}

	// Store analysis result
	result := &FileAnalysisResult{
		FileID:     fileID,
		FileName:   header.Filename,
		FileSize:   header.Size,
		Columns:    columns,
		RowCount:   len(rows) - 1,
		SampleData: sampleData,
		ImportType: importType,
	}

	TempFileStore[fileID] = result

	// Clean up old temp files after 1 hour
	go func() {
		time.Sleep(1 * time.Hour)
		delete(TempFileStore, fileID)
		os.Remove(tempPath)
	}()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// enhancedImportValidateHandler validates the data with column mappings
func enhancedImportValidateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check authentication
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		SendError(w, ErrUnauthorized("Access denied"))
		return
	}

	// Parse request
	var req struct {
		Type     string            `json:"type"`
		FileID   string            `json:"file_id"`
		Mappings map[string]string `json:"mappings"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, ErrValidation(fmt.Sprintf("Invalid request: %v", err)))
		return
	}

	// Get file analysis result
	fileResult, ok := TempFileStore[req.FileID]
	if !ok {
		SendError(w, ErrValidation("File not found. Please upload again."))
		return
	}

	// Open Excel file
	tempPath := filepath.Join(os.TempDir(), req.FileID+filepath.Ext(fileResult.FileName))
	xlsx, err := excelize.OpenFile(tempPath)
	if err != nil {
		SendError(w, ErrInternal("Failed to open file", err))
		return
	}
	defer xlsx.Close()

	// Validate based on type
	result := validateImportData(xlsx, req.Type, req.Mappings)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// enhancedImportExecuteHandler executes the actual import
func enhancedImportExecuteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check authentication
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		SendError(w, ErrUnauthorized("Access denied"))
		return
	}

	// Parse request
	var req struct {
		Type           string            `json:"type"`
		FileID         string            `json:"file_id"`
		Mappings       map[string]string `json:"mappings"`
		SkipDuplicates bool              `json:"skip_duplicates"`
		UpdateExisting bool              `json:"update_existing"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, ErrValidation(fmt.Sprintf("Invalid request: %v", err)))
		return
	}

	// Get file analysis result
	fileResult, ok := TempFileStore[req.FileID]
	if !ok {
		SendError(w, ErrValidation("File not found. Please upload again."))
		return
	}

	// Open Excel file
	tempPath := filepath.Join(os.TempDir(), req.FileID+filepath.Ext(fileResult.FileName))
	xlsx, err := excelize.OpenFile(tempPath)
	if err != nil {
		SendError(w, ErrInternal("Failed to open file", err))
		return
	}
	defer xlsx.Close()

	// Execute import based on type
	var result *ImportResult
	switch req.Type {
	case "students":
		result = importStudentsWithValidation(xlsx, req.Mappings, req.SkipDuplicates, req.UpdateExisting)
	case "mileage":
		result = importMileageWithValidation(xlsx, req.Mappings, req.SkipDuplicates, req.UpdateExisting)
	case "ecse":
		result = importECSEWithValidation(xlsx, req.Mappings, req.SkipDuplicates, req.UpdateExisting)
	case "maintenance":
		result = importMaintenanceWithValidation(xlsx, req.Mappings, req.SkipDuplicates, req.UpdateExisting)
	default:
		SendError(w, ErrValidation("Invalid import type"))
		return
	}

	// Clean up temp file
	go func() {
		delete(TempFileStore, req.FileID)
		os.Remove(tempPath)
	}()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// validateImportData validates the import data based on type
func validateImportData(xlsx *excelize.File, importType string, mappings map[string]string) *ValidationResult {
	result := &ValidationResult{
		Warnings: []string{},
		Errors:   []string{},
		Preview:  []map[string]interface{}{},
	}

	// Get first sheet
	sheets := xlsx.GetSheetList()
	if len(sheets) == 0 {
		result.Errors = append(result.Errors, "No sheets found in Excel file")
		return result
	}

	rows, err := xlsx.GetRows(sheets[0])
	if err != nil {
		result.Errors = append(result.Errors, "Failed to read Excel data")
		return result
	}

	if len(rows) < 2 {
		result.Errors = append(result.Errors, "File must have at least a header row and one data row")
		return result
	}

	columns := rows[0]
	result.TotalRecords = len(rows) - 1

	// Create reverse mapping (field -> column)
	reverseMap := make(map[string]string)
	for col, field := range mappings {
		reverseMap[field] = col
	}

	// Check for duplicates across all data
	duplicateCheck := make(map[string][]int)

	// Validate each row
	for i := 1; i < len(rows); i++ {
		rowData := make(map[string]interface{})
		isValid := true

		// Map columns to fields
		for field, col := range reverseMap {
			colIndex := -1
			for j, c := range columns {
				if c == col {
					colIndex = j
					break
				}
			}

			if colIndex >= 0 && colIndex < len(rows[i]) {
				rowData[field] = strings.TrimSpace(rows[i][colIndex])
			} else {
				rowData[field] = ""
			}
		}

		// Type-specific validation
		switch importType {
		case "students":
			if err := validateStudentRow(rowData, i+1, result); err != nil {
				isValid = false
			} else {
				// Check for duplicates
				if id, ok := rowData["student_id"].(string); ok && id != "" {
					duplicateCheck[id] = append(duplicateCheck[id], i+1)
				}
			}
		case "mileage":
			if err := validateMileageRow(rowData, i+1, result); err != nil {
				isValid = false
			}
		case "ecse":
			if err := validateECSERow(rowData, i+1, result); err != nil {
				isValid = false
			}
		case "maintenance":
			if err := validateMaintenanceRow(rowData, i+1, result); err != nil {
				isValid = false
			}
		}

		if isValid {
			result.ValidRecords++
			// Add to preview (first 10 valid records)
			if len(result.Preview) < 10 {
				result.Preview = append(result.Preview, rowData)
			}
		} else {
			result.InvalidRecords++
		}
	}

	// Report duplicates
	for id, rows := range duplicateCheck {
		if len(rows) > 1 {
			result.Warnings = append(result.Warnings, 
				fmt.Sprintf("Duplicate ID '%s' found in rows: %v", id, rows))
		}
	}

	// Check existing database records
	if importType == "students" && result.ValidRecords > 0 {
		checkExistingStudents(result)
	}

	return result
}

// Validation functions for each type
func validateStudentRow(data map[string]interface{}, rowNum int, result *ValidationResult) error {
	hasError := false

	// Check required fields
	studentID, _ := data["student_id"].(string)
	name, _ := data["name"].(string)
	phone, _ := data["phone_number"].(string)

	if studentID == "" {
		result.Errors = append(result.Errors, fmt.Sprintf("Row %d: Missing student ID", rowNum))
		hasError = true
	} else if len(studentID) > 50 {
		result.Errors = append(result.Errors, fmt.Sprintf("Row %d: Student ID too long (max 50 characters)", rowNum))
		hasError = true
	}

	if name == "" {
		result.Errors = append(result.Errors, fmt.Sprintf("Row %d: Missing student name", rowNum))
		hasError = true
	} else if len(name) > 100 {
		result.Errors = append(result.Errors, fmt.Sprintf("Row %d: Student name too long (max 100 characters)", rowNum))
		hasError = true
	}

	if phone == "" {
		result.Warnings = append(result.Warnings, fmt.Sprintf("Row %d: Missing phone number", rowNum))
	} else if !importWizardIsValidPhone(phone) {
		result.Warnings = append(result.Warnings, fmt.Sprintf("Row %d: Invalid phone number format: %s", rowNum, phone))
	}

	if hasError {
		return fmt.Errorf("validation failed")
	}
	return nil
}

func validateMileageRow(data map[string]interface{}, rowNum int, result *ValidationResult) error {
	hasError := false

	vehicleID, _ := data["vehicle_id"].(string)
	dateStr, _ := data["date"].(string)
	mileageStr, _ := data["mileage"].(string)

	if vehicleID == "" {
		result.Errors = append(result.Errors, fmt.Sprintf("Row %d: Missing vehicle ID", rowNum))
		hasError = true
	}

	if dateStr == "" {
		result.Errors = append(result.Errors, fmt.Sprintf("Row %d: Missing date", rowNum))
		hasError = true
	} else if _, err := time.Parse("2006-01-02", dateStr); err != nil {
		// Try other common date formats
		if _, err := time.Parse("01/02/2006", dateStr); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Row %d: Invalid date format: %s", rowNum, dateStr))
			hasError = true
		}
	}

	if mileageStr == "" {
		result.Errors = append(result.Errors, fmt.Sprintf("Row %d: Missing mileage", rowNum))
		hasError = true
	} else if mileage, err := strconv.Atoi(mileageStr); err != nil || mileage < 0 {
		result.Errors = append(result.Errors, fmt.Sprintf("Row %d: Invalid mileage value: %s", rowNum, mileageStr))
		hasError = true
	}

	if hasError {
		return fmt.Errorf("validation failed")
	}
	return nil
}

func validateECSERow(data map[string]interface{}, rowNum int, result *ValidationResult) error {
	hasError := false

	studentID, _ := data["student_id"].(string)
	firstName, _ := data["first_name"].(string)
	lastName, _ := data["last_name"].(string)
	dobStr, _ := data["date_of_birth"].(string)

	if studentID == "" {
		result.Errors = append(result.Errors, fmt.Sprintf("Row %d: Missing student ID", rowNum))
		hasError = true
	}

	if firstName == "" {
		result.Errors = append(result.Errors, fmt.Sprintf("Row %d: Missing first name", rowNum))
		hasError = true
	}

	if lastName == "" {
		result.Errors = append(result.Errors, fmt.Sprintf("Row %d: Missing last name", rowNum))
		hasError = true
	}

	if dobStr != "" {
		if _, err := time.Parse("2006-01-02", dobStr); err != nil {
			if _, err := time.Parse("01/02/2006", dobStr); err != nil {
				result.Warnings = append(result.Warnings, fmt.Sprintf("Row %d: Invalid date of birth format: %s", rowNum, dobStr))
			}
		}
	}

	if hasError {
		return fmt.Errorf("validation failed")
	}
	return nil
}

func validateMaintenanceRow(data map[string]interface{}, rowNum int, result *ValidationResult) error {
	hasError := false

	vehicleID, _ := data["vehicle_id"].(string)
	dateStr, _ := data["date"].(string)
	category, _ := data["category"].(string)
	description, _ := data["description"].(string)
	costStr, _ := data["cost"].(string)

	if vehicleID == "" {
		result.Errors = append(result.Errors, fmt.Sprintf("Row %d: Missing vehicle ID", rowNum))
		hasError = true
	}

	if dateStr == "" {
		result.Errors = append(result.Errors, fmt.Sprintf("Row %d: Missing date", rowNum))
		hasError = true
	} else if _, err := time.Parse("2006-01-02", dateStr); err != nil {
		if _, err := time.Parse("01/02/2006", dateStr); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Row %d: Invalid date format: %s", rowNum, dateStr))
			hasError = true
		}
	}

	if category == "" {
		result.Errors = append(result.Errors, fmt.Sprintf("Row %d: Missing category", rowNum))
		hasError = true
	} else {
		validCategories := []string{"oil_change", "tire_service", "inspection", "repair", "other"}
		isValidCategory := false
		for _, validCat := range validCategories {
			if strings.ToLower(category) == validCat {
				isValidCategory = true
				break
			}
		}
		if !isValidCategory {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Row %d: Unknown category '%s', will be set to 'other'", rowNum, category))
		}
	}

	if description == "" {
		result.Warnings = append(result.Warnings, fmt.Sprintf("Row %d: Missing description", rowNum))
	}

	if costStr != "" {
		if _, err := strconv.ParseFloat(costStr, 64); err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Row %d: Invalid cost value: %s", rowNum, costStr))
		}
	}

	if hasError {
		return fmt.Errorf("validation failed")
	}
	return nil
}

// Helper functions
func importWizardIsValidPhone(phone string) bool {
	// Remove common formatting characters
	cleaned := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, phone)

	// Check if it's a valid phone number length
	return len(cleaned) >= 10 && len(cleaned) <= 15
}

func checkExistingStudents(result *ValidationResult) {
	// Get IDs from preview
	var ids []string
	for _, row := range result.Preview {
		if id, ok := row["student_id"].(string); ok && id != "" {
			ids = append(ids, id)
		}
	}

	if len(ids) == 0 {
		return
	}

	// Check database
	query := fmt.Sprintf("SELECT student_id FROM students WHERE student_id IN ('%s')", 
		strings.Join(ids, "','"))
	
	rows, err := db.Query(query)
	if err != nil {
		result.Warnings = append(result.Warnings, "Could not check for existing students in database")
		return
	}
	defer rows.Close()

	var existingCount int
	for rows.Next() {
		existingCount++
	}

	if existingCount > 0 {
		result.Warnings = append(result.Warnings, 
			fmt.Sprintf("%d students already exist in the database", existingCount))
	}
}

// Import functions with validation
func importStudentsWithValidation(xlsx *excelize.File, mappings map[string]string, skipDuplicates, updateExisting bool) *ImportResult {
	result := &ImportResult{
		Details: []string{},
	}

	// Get first sheet
	sheets := xlsx.GetSheetList()
	rows, _ := xlsx.GetRows(sheets[0])
	
	if len(rows) < 2 {
		result.Errors = 1
		result.Details = append(result.Details, "No data rows found")
		return result
	}
	
	columns := rows[0]

	// Create reverse mapping
	reverseMap := make(map[string]string)
	for col, field := range mappings {
		reverseMap[field] = col
	}

	// Import each row
	for i := 1; i < len(rows); i++ {
		result.Total++
		rowData := make(map[string]string)

		// Map columns to fields
		for field, col := range reverseMap {
			colIndex := -1
			for j, c := range columns {
				if c == col {
					colIndex = j
					break
				}
			}

			if colIndex >= 0 && colIndex < len(rows[i]) {
				rowData[field] = strings.TrimSpace(rows[i][colIndex])
			}
		}

		// Skip empty rows
		if rowData["student_id"] == "" && rowData["name"] == "" {
			result.Skipped++
			continue
		}

		// Check if student exists
		var existingID string
		err := db.Get(&existingID, "SELECT student_id FROM students WHERE student_id = $1", rowData["student_id"])
		
		if err == nil && skipDuplicates && !updateExisting {
			result.Skipped++
			result.Details = append(result.Details, fmt.Sprintf("Skipped duplicate student ID: %s", rowData["student_id"]))
			continue
		}

		if err == nil && updateExisting {
			// Update existing student
			_, err = db.Exec(`
				UPDATE students 
				SET name = $2, phone_number = $3, guardian = $4
				WHERE student_id = $1`,
				rowData["student_id"], rowData["name"], 
				rowData["phone_number"], rowData["guardian"])
			
			if err != nil {
				result.Errors++
				result.Details = append(result.Details, 
					fmt.Sprintf("Failed to update student %s: %v", rowData["student_id"], err))
			} else {
				result.Imported++
			}
		} else {
			// Insert new student
			_, err = db.Exec(`
				INSERT INTO students (student_id, name, phone_number, guardian, active)
				VALUES ($1, $2, $3, $4, true)`,
				rowData["student_id"], rowData["name"], 
				rowData["phone_number"], rowData["guardian"])
			
			if err != nil {
				result.Errors++
				result.Details = append(result.Details, 
					fmt.Sprintf("Failed to insert student %s: %v", rowData["student_id"], err))
			} else {
				result.Imported++
			}
		}
	}

	return result
}

func importMileageWithValidation(xlsx *excelize.File, mappings map[string]string, skipDuplicates, updateExisting bool) *ImportResult {
	result := &ImportResult{
		Details: []string{},
	}

	// Get first sheet
	sheets := xlsx.GetSheetList()
	rows, _ := xlsx.GetRows(sheets[0])
	
	if len(rows) < 2 {
		result.Errors = 1
		result.Details = append(result.Details, "No data rows found")
		return result
	}
	
	columns := rows[0]

	// Create reverse mapping
	reverseMap := make(map[string]string)
	for col, field := range mappings {
		reverseMap[field] = col
	}

	// Import each row
	for i := 1; i < len(rows); i++ {
		result.Total++
		rowData := make(map[string]string)

		// Map columns to fields
		for field, col := range reverseMap {
			colIndex := -1
			for j, c := range columns {
				if c == col {
					colIndex = j
					break
				}
			}

			if colIndex >= 0 && colIndex < len(rows[i]) {
				rowData[field] = strings.TrimSpace(rows[i][colIndex])
			}
		}

		// Parse date
		dateStr := rowData["date"]
		var date time.Time
		var err error
		
		// Try different date formats
		for _, format := range []string{"2006-01-02", "01/02/2006", "1/2/2006"} {
			date, err = time.Parse(format, dateStr)
			if err == nil {
				break
			}
		}
		
		if err != nil {
			result.Errors++
			result.Details = append(result.Details, 
				fmt.Sprintf("Row %d: Invalid date format: %s", i+1, dateStr))
			continue
		}

		// Parse mileage
		mileage, err := strconv.Atoi(rowData["mileage"])
		if err != nil {
			result.Errors++
			result.Details = append(result.Details, 
				fmt.Sprintf("Row %d: Invalid mileage: %s", i+1, rowData["mileage"]))
			continue
		}

		// Insert mileage record
		_, err = db.Exec(`
			INSERT INTO monthly_mileage_reports (vehicle_id, report_date, total_mileage)
			VALUES ($1, $2, $3)
			ON CONFLICT (vehicle_id, report_date) 
			DO UPDATE SET total_mileage = $3`,
			rowData["vehicle_id"], date, mileage)
		
		if err != nil {
			result.Errors++
			result.Details = append(result.Details, 
				fmt.Sprintf("Failed to insert mileage for vehicle %s: %v", 
					rowData["vehicle_id"], err))
		} else {
			result.Imported++
		}
	}

	return result
}

func importECSEWithValidation(xlsx *excelize.File, mappings map[string]string, skipDuplicates, updateExisting bool) *ImportResult {
	result := &ImportResult{
		Details: []string{},
	}
	
	// Placeholder implementation
	result.Total = 0
	result.Details = append(result.Details, "ECSE import functionality coming soon")
	
	return result
}

func importMaintenanceWithValidation(xlsx *excelize.File, mappings map[string]string, skipDuplicates, updateExisting bool) *ImportResult {
	result := &ImportResult{
		Details: []string{},
	}
	
	// Get first sheet
	sheets := xlsx.GetSheetList()
	rows, _ := xlsx.GetRows(sheets[0])
	
	if len(rows) < 2 {
		result.Errors = 1
		result.Details = append(result.Details, "No data rows found")
		return result
	}
	
	columns := rows[0]

	// Create reverse mapping
	reverseMap := make(map[string]string)
	for col, field := range mappings {
		reverseMap[field] = col
	}

	// Import each row
	for i := 1; i < len(rows); i++ {
		result.Total++
		rowData := make(map[string]string)

		// Map columns to fields
		for field, col := range reverseMap {
			colIndex := -1
			for j, c := range columns {
				if c == col {
					colIndex = j
					break
				}
			}

			if colIndex >= 0 && colIndex < len(rows[i]) {
				rowData[field] = strings.TrimSpace(rows[i][colIndex])
			}
		}

		// Parse date
		dateStr := rowData["date"]
		var date time.Time
		var err error
		
		// Try different date formats
		for _, format := range []string{"2006-01-02", "01/02/2006", "1/2/2006"} {
			date, err = time.Parse(format, dateStr)
			if err == nil {
				break
			}
		}
		
		if err != nil {
			result.Errors++
			result.Details = append(result.Details, 
				fmt.Sprintf("Row %d: Invalid date format: %s", i+1, dateStr))
			continue
		}

		// Parse cost
		cost := 0.0
		if rowData["cost"] != "" {
			cost, _ = strconv.ParseFloat(rowData["cost"], 64)
		}

		// Normalize category
		category := strings.ToLower(rowData["category"])
		validCategories := map[string]bool{
			"oil_change": true,
			"tire_service": true,
			"inspection": true,
			"repair": true,
			"other": true,
		}
		
		if !validCategories[category] {
			category = "other"
		}

		// Determine vehicle type
		vehicleType := "bus"
		if strings.HasPrefix(rowData["vehicle_id"], "V") {
			vehicleType = "fleet"
		}

		// Insert maintenance record
		_, err = db.Exec(`
			INSERT INTO maintenance_records 
			(vehicle_id, vehicle_type, service_date, service_type, description, cost, mileage)
			VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			rowData["vehicle_id"], vehicleType, date, category, 
			rowData["description"], cost, 0)
		
		if err != nil {
			result.Errors++
			result.Details = append(result.Details, 
				fmt.Sprintf("Failed to insert maintenance for vehicle %s: %v", 
					rowData["vehicle_id"], err))
		} else {
			result.Imported++
		}
	}

	return result
}