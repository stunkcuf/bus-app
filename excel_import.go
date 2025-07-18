package main

import (
	"database/sql"
	"fmt"
	"math/rand"
	"mime/multipart"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

// ImportType represents the type of Excel import
type ImportType string

const (
	ImportTypeMileage ImportType = "mileage"
	ImportTypeECSE    ImportType = "ecse"
	ImportTypeStudent ImportType = "student"
	ImportTypeVehicle ImportType = "vehicle"
	
	// MaxFileSize is the maximum allowed Excel file size (10MB)
	MaxFileSize = 10 * 1024 * 1024
)

// ImportError represents a detailed import error
type ImportError struct {
	Row         int         `json:"row"`
	Column      string      `json:"column"`
	Sheet       string      `json:"sheet"`
	Value       interface{} `json:"value"`
	Error       string      `json:"error"`
	ErrorType   string      `json:"error_type"`
	Severity    string      `json:"severity"` // "error", "warning", "info"
}

// ImportResult represents the result of an import operation
type ImportResult struct {
	TotalRows      int            `json:"total_rows"`
	ProcessedRows  int            `json:"processed_rows"`
	SuccessfulRows int            `json:"successful_rows"`
	FailedRows     int            `json:"failed_rows"`
	WarningCount   int            `json:"warning_count"`
	Errors         []ImportError  `json:"errors"`
	Warnings       []ImportError  `json:"warnings"`
	Summary        string         `json:"summary"`
	StartTime      time.Time      `json:"start_time"`
	EndTime        time.Time      `json:"end_time"`
	Duration       string         `json:"duration"`
	ImportID       string         `json:"import_id"`
	ImportType     ImportType     `json:"import_type"`
	FileName       string         `json:"file_name"`
	FileSize       int64          `json:"file_size"`
	Sheets         []string       `json:"sheets"`
	RollbackInfo   *RollbackInfo  `json:"rollback_info,omitempty"`
}

// RollbackInfo contains information needed to rollback an import
type RollbackInfo struct {
	TransactionID string    `json:"transaction_id"`
	TableName     string    `json:"table_name"`
	RecordIDs     []string  `json:"record_ids"`
	BackupTable   string    `json:"backup_table"`
	CanRollback   bool      `json:"can_rollback"`
	RollbackUntil time.Time `json:"rollback_until"`
}

// ExcelImporter handles Excel file imports with enhanced error handling
type ExcelImporter struct {
	db             *sql.DB
	importType     ImportType
	validator      *ImportValidator
	result         *ImportResult
	tx             *sql.Tx
	logger         *Logger
	SkipInvalid    bool
	StopOnError    bool
	IgnoreWarnings bool
	ColumnMap      map[string]int
}

// NewExcelImporter creates a new Excel importer
func NewExcelImporter(db *sql.DB, importType ImportType) *ExcelImporter {
	return &ExcelImporter{
		db:         db,
		importType: importType,
		validator:  NewImportValidator(importType),
		logger:     logger,
	}
}

// ImportFile imports an Excel file with comprehensive error handling
func (ei *ExcelImporter) ImportFile(file multipart.File, header *multipart.FileHeader) (*ImportResult, error) {
	// Initialize result
	ei.result = &ImportResult{
		ImportType: ei.importType,
		FileName:   header.Filename,
		FileSize:   header.Size,
		StartTime:  time.Now(),
		ImportID:   generateImportID(),
		Errors:     []ImportError{},
		Warnings:   []ImportError{},
	}

	// Validate file
	if err := ei.validateFile(header); err != nil {
		ei.result.Errors = append(ei.result.Errors, ImportError{
			Error:     err.Error(),
			ErrorType: "FILE_VALIDATION",
			Severity:  "error",
		})
		return ei.finishImport(false), err
	}

	// Open Excel file
	xlsx, err := excelize.OpenReader(file)
	if err != nil {
		ei.result.Errors = append(ei.result.Errors, ImportError{
			Error:     fmt.Sprintf("Failed to open Excel file: %v", err),
			ErrorType: "FILE_OPEN",
			Severity:  "error",
		})
		return ei.finishImport(false), err
	}
	defer xlsx.Close()

	// Get sheets
	sheets := xlsx.GetSheetList()
	ei.result.Sheets = sheets

	if len(sheets) == 0 {
		ei.result.Errors = append(ei.result.Errors, ImportError{
			Error:     "No sheets found in Excel file",
			ErrorType: "NO_SHEETS",
			Severity:  "error",
		})
		return ei.finishImport(false), fmt.Errorf("no sheets found")
	}

	// Start transaction
	tx, err := ei.db.Begin()
	if err != nil {
		ei.result.Errors = append(ei.result.Errors, ImportError{
			Error:     fmt.Sprintf("Failed to start transaction: %v", err),
			ErrorType: "TRANSACTION",
			Severity:  "error",
		})
		return ei.finishImport(false), err
	}
	ei.tx = tx

	// Create rollback info
	ei.result.RollbackInfo = &RollbackInfo{
		TransactionID: ei.result.ImportID,
		TableName:     ei.getTableName(),
		RecordIDs:     []string{},
		CanRollback:   true,
		RollbackUntil: time.Now().Add(24 * time.Hour),
	}

	// Process each sheet
	success := true
	for _, sheet := range sheets {
		if err := ei.processSheet(xlsx, sheet); err != nil {
			ei.logger.WithField("sheet", sheet).Error("Failed to process sheet", err)
			success = false
			// Continue with other sheets instead of failing completely
		}
	}

	// Commit or rollback transaction
	if success && ei.result.FailedRows == 0 {
		if err := tx.Commit(); err != nil {
			ei.result.Errors = append(ei.result.Errors, ImportError{
				Error:     fmt.Sprintf("Failed to commit transaction: %v", err),
				ErrorType: "TRANSACTION_COMMIT",
				Severity:  "error",
			})
			tx.Rollback()
			return ei.finishImport(false), err
		}
		ei.result.RollbackInfo.CanRollback = true
	} else {
		tx.Rollback()
		ei.result.RollbackInfo.CanRollback = false
	}

	return ei.finishImport(success), nil
}

// validateFile validates the uploaded file
func (ei *ExcelImporter) validateFile(header *multipart.FileHeader) error {
	// Check file extension
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".xlsx" && ext != ".xls" {
		return fmt.Errorf("invalid file type: %s (expected .xlsx or .xls)", ext)
	}

	// Check file size
	if header.Size > MaxFileSize {
		return fmt.Errorf("file too large: %d bytes (max %d bytes)", header.Size, MaxFileSize)
	}

	// Check MIME type
	contentType := header.Header.Get("Content-Type")
	validTypes := []string{
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		"application/vnd.ms-excel",
		"application/octet-stream", // Sometimes Excel files are detected as this
	}
	
	valid := false
	for _, validType := range validTypes {
		if strings.Contains(contentType, validType) {
			valid = true
			break
		}
	}
	
	if !valid && contentType != "" {
		ei.result.Warnings = append(ei.result.Warnings, ImportError{
			Error:     fmt.Sprintf("Unexpected content type: %s", contentType),
			ErrorType: "MIME_TYPE",
			Severity:  "warning",
		})
	}

	return nil
}

// processSheet processes a single sheet
func (ei *ExcelImporter) processSheet(xlsx *excelize.File, sheet string) error {
	rows, err := xlsx.GetRows(sheet)
	if err != nil {
		ei.result.Errors = append(ei.result.Errors, ImportError{
			Sheet:     sheet,
			Error:     fmt.Sprintf("Failed to read sheet: %v", err),
			ErrorType: "SHEET_READ",
			Severity:  "error",
		})
		return err
	}

	if len(rows) == 0 {
		ei.result.Warnings = append(ei.result.Warnings, ImportError{
			Sheet:     sheet,
			Error:     "Sheet is empty",
			ErrorType: "EMPTY_SHEET",
			Severity:  "warning",
		})
		return nil
	}

	ei.result.TotalRows += len(rows)

	// Find header row
	headerRow, headerIndex := ei.findHeaderRow(rows, sheet)
	if headerIndex == -1 {
		ei.result.Errors = append(ei.result.Errors, ImportError{
			Sheet:     sheet,
			Error:     "Could not find header row",
			ErrorType: "NO_HEADER",
			Severity:  "error",
		})
		return fmt.Errorf("no header row found")
	}

	// Map columns
	columnMap := ei.mapColumns(headerRow, sheet)
	if len(columnMap) == 0 {
		ei.result.Errors = append(ei.result.Errors, ImportError{
			Sheet:     sheet,
			Row:       headerIndex + 1,
			Error:     "No valid columns found in header",
			ErrorType: "INVALID_HEADER",
			Severity:  "error",
		})
		return fmt.Errorf("invalid header")
	}

	// Process data rows
	for i := headerIndex + 1; i < len(rows); i++ {
		row := rows[i]
		ei.result.ProcessedRows++

		// Skip empty rows
		if ei.isEmptyRow(row) {
			continue
		}

		// Process row based on import type
		switch ei.importType {
		case ImportTypeMileage:
			err = ei.processMileageRow(row, i+1, sheet, columnMap)
		case ImportTypeECSE:
			err = ei.processECSERow(row, i+1, sheet, columnMap)
		case ImportTypeStudent:
			err = ei.processStudentRow(row, i+1, sheet, columnMap)
		case ImportTypeVehicle:
			err = ei.processVehicleRow(row, i+1, sheet, columnMap)
		}

		if err != nil {
			ei.result.FailedRows++
			// Error already added in process function
		} else {
			ei.result.SuccessfulRows++
		}
	}

	return nil
}

// findHeaderRow finds the header row in the sheet
func (ei *ExcelImporter) findHeaderRow(rows [][]string, sheet string) ([]string, int) {
	expectedHeaders := ei.validator.GetExpectedHeaders()
	
	for i, row := range rows {
		if ei.isHeaderRow(row, expectedHeaders) {
			return row, i
		}
	}
	
	return nil, -1
}

// isHeaderRow checks if a row is a header row
func (ei *ExcelImporter) isHeaderRow(row []string, expectedHeaders []string) bool {
	matches := 0
	for _, cell := range row {
		cellLower := strings.ToLower(strings.TrimSpace(cell))
		for _, expected := range expectedHeaders {
			if strings.Contains(cellLower, strings.ToLower(expected)) {
				matches++
				break
			}
		}
	}
	
	// Consider it a header if at least 50% of expected headers are found
	return matches >= len(expectedHeaders)/2
}

// mapColumns creates a mapping of column names to indices
func (ei *ExcelImporter) mapColumns(headerRow []string, sheet string) map[string]int {
	columnMap := make(map[string]int)
	
	for i, header := range headerRow {
		normalized := ei.normalizeHeader(header)
		if normalized != "" {
			columnMap[normalized] = i
		}
	}
	
	// Log unmapped required columns
	required := ei.validator.GetRequiredColumns()
	for _, col := range required {
		if _, ok := columnMap[col]; !ok {
			ei.result.Warnings = append(ei.result.Warnings, ImportError{
				Sheet:     sheet,
				Column:    col,
				Error:     fmt.Sprintf("Required column '%s' not found", col),
				ErrorType: "MISSING_COLUMN",
				Severity:  "warning",
			})
		}
	}
	
	return columnMap
}

// normalizeHeader normalizes a header string
func (ei *ExcelImporter) normalizeHeader(header string) string {
	// Remove special characters and normalize
	normalized := strings.ToLower(strings.TrimSpace(header))
	normalized = strings.ReplaceAll(normalized, " ", "_")
	normalized = strings.ReplaceAll(normalized, "-", "_")
	normalized = strings.ReplaceAll(normalized, ".", "")
	normalized = strings.ReplaceAll(normalized, "#", "number")
	
	// Map common variations
	mappings := map[string]string{
		"vehicle_id":     "vehicle_id",
		"vehicle_number": "vehicle_id",
		"bus_id":         "bus_id",
		"bus_number":     "bus_id",
		"student_name":   "name",
		"full_name":      "name",
		"phone":          "phone_number",
		"telephone":      "phone_number",
	}
	
	if mapped, ok := mappings[normalized]; ok {
		return mapped
	}
	
	return normalized
}

// isEmptyRow checks if a row is empty
func (ei *ExcelImporter) isEmptyRow(row []string) bool {
	for _, cell := range row {
		if strings.TrimSpace(cell) != "" {
			return false
		}
	}
	return true
}

// processMileageRow processes a mileage data row
func (ei *ExcelImporter) processMileageRow(row []string, rowNum int, sheet string, columnMap map[string]int) error {
	// Extract values
	vehicleID := ei.getStringValue(row, columnMap, "vehicle_id")
	if vehicleID == "" {
		ei.addError(rowNum, "vehicle_id", sheet, "", "Vehicle ID is required", "REQUIRED_FIELD")
		return fmt.Errorf("missing vehicle ID")
	}

	// Validate and extract other fields
	beginMileage := ei.getIntValue(row, columnMap, "beginning_mileage", rowNum, sheet)
	endMileage := ei.getIntValue(row, columnMap, "ending_mileage", rowNum, sheet)
	
	if beginMileage < 0 || endMileage < 0 {
		return fmt.Errorf("invalid mileage values")
	}
	
	if endMileage < beginMileage {
		ei.addError(rowNum, "ending_mileage", sheet, endMileage, 
			"Ending mileage cannot be less than beginning mileage", "VALIDATION")
		return fmt.Errorf("invalid mileage range")
	}

	// Insert into database
	query := `INSERT INTO mileage_records (vehicle_id, begin_mileage, end_mileage, import_id) 
	          VALUES ($1, $2, $3, $4) RETURNING id`
	
	var recordID string
	err := ei.tx.QueryRow(query, vehicleID, beginMileage, endMileage, ei.result.ImportID).Scan(&recordID)
	if err != nil {
		ei.addError(rowNum, "", sheet, row, fmt.Sprintf("Database error: %v", err), "DATABASE")
		return err
	}
	
	ei.result.RollbackInfo.RecordIDs = append(ei.result.RollbackInfo.RecordIDs, recordID)
	return nil
}

// Helper methods for getting values with validation
func (ei *ExcelImporter) getStringValue(row []string, columnMap map[string]int, column string) string {
	if idx, ok := columnMap[column]; ok && idx < len(row) {
		return strings.TrimSpace(row[idx])
	}
	return ""
}

func (ei *ExcelImporter) getIntValue(row []string, columnMap map[string]int, column string, rowNum int, sheet string) int {
	str := ei.getStringValue(row, columnMap, column)
	if str == "" {
		return 0
	}
	
	val, err := strconv.Atoi(str)
	if err != nil {
		ei.addError(rowNum, column, sheet, str, "Invalid number format", "FORMAT")
		return -1
	}
	
	return val
}

// addError adds an error to the result
func (ei *ExcelImporter) addError(row int, column, sheet string, value interface{}, error, errorType string) {
	ei.result.Errors = append(ei.result.Errors, ImportError{
		Row:       row,
		Column:    column,
		Sheet:     sheet,
		Value:     value,
		Error:     error,
		ErrorType: errorType,
		Severity:  "error",
	})
}

// finishImport finalizes the import result
func (ei *ExcelImporter) finishImport(success bool) *ImportResult {
	ei.result.EndTime = time.Now()
	ei.result.Duration = ei.result.EndTime.Sub(ei.result.StartTime).String()
	
	if success && ei.result.FailedRows == 0 {
		ei.result.Summary = fmt.Sprintf("Import completed successfully. %d records imported.", 
			ei.result.SuccessfulRows)
	} else if ei.result.SuccessfulRows > 0 {
		ei.result.Summary = fmt.Sprintf("Import partially completed. %d succeeded, %d failed.", 
			ei.result.SuccessfulRows, ei.result.FailedRows)
	} else {
		ei.result.Summary = "Import failed. No records were imported."
	}
	
	// Save import history
	ei.saveImportHistory()
	
	return ei.result
}

// saveImportHistory saves the import history to database
func (ei *ExcelImporter) saveImportHistory() {
	query := `INSERT INTO import_history 
	          (import_id, import_type, file_name, file_size, total_rows, successful_rows, 
	           failed_rows, error_count, warning_count, summary, start_time, end_time) 
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`
	
	_, err := ei.db.Exec(query, 
		ei.result.ImportID,
		ei.result.ImportType,
		ei.result.FileName,
		ei.result.FileSize,
		ei.result.TotalRows,
		ei.result.SuccessfulRows,
		ei.result.FailedRows,
		len(ei.result.Errors),
		len(ei.result.Warnings),
		ei.result.Summary,
		ei.result.StartTime,
		ei.result.EndTime,
	)
	
	if err != nil {
		ei.logger.Error("Failed to save import history", err)
	}
}

// getTableName returns the table name for the import type
func (ei *ExcelImporter) getTableName() string {
	switch ei.importType {
	case ImportTypeMileage:
		return "mileage_records"
	case ImportTypeECSE:
		return "ecse_students"
	case ImportTypeStudent:
		return "students"
	case ImportTypeVehicle:
		return "vehicles"
	default:
		return ""
	}
}

// generateImportID generates a unique import ID
func generateImportID() string {
	return fmt.Sprintf("IMP_%d_%s", time.Now().Unix(), generateRandomString(8))
}

// generateRandomString generates a random alphanumeric string of specified length
func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

// processECSERow processes an ECSE student data row
func (ei *ExcelImporter) processECSERow(row []string, rowNum int, sheet string, columnMap map[string]int) error {
	// Extract required fields
	name := ei.getStringValue(row, columnMap, "name")
	if name == "" {
		ei.addError(rowNum, "name", sheet, "", "Student name is required", "REQUIRED_FIELD")
		return fmt.Errorf("missing student name")
	}
	
	// Validate name
	if err := ei.validator.ValidateField("name", name); err != nil {
		ei.addError(rowNum, "name", sheet, name, err.Error(), "VALIDATION")
		return err
	}
	
	// Extract and validate DOB
	dobStr := ei.getStringValue(row, columnMap, "dob")
	if dobStr == "" {
		ei.addError(rowNum, "dob", sheet, "", "Date of birth is required", "REQUIRED_FIELD")
		return fmt.Errorf("missing date of birth")
	}
	
	dob, err := ParseDate(dobStr)
	if err != nil {
		ei.addError(rowNum, "dob", sheet, dobStr, "Invalid date format", "FORMAT")
		return err
	}
	
	// Extract phone
	phone := ei.getStringValue(row, columnMap, "phone")
	if phone == "" {
		ei.addError(rowNum, "phone", sheet, "", "Phone number is required", "REQUIRED_FIELD")
		return fmt.Errorf("missing phone number")
	}
	
	// Normalize and validate phone
	phone = NormalizePhone(phone)
	if err := ei.validator.ValidateField("phone", phone); err != nil {
		ei.addError(rowNum, "phone", sheet, phone, err.Error(), "VALIDATION")
		return err
	}
	
	// Extract optional fields
	address := ei.getStringValue(row, columnMap, "address")
	iepStatus := ParseBoolean(ei.getStringValue(row, columnMap, "iep_status"))
	speechTherapy := ParseBoolean(ei.getStringValue(row, columnMap, "speech_therapy"))
	occupationalTherapy := ParseBoolean(ei.getStringValue(row, columnMap, "occupational_therapy"))
	physicalTherapy := ParseBoolean(ei.getStringValue(row, columnMap, "physical_therapy"))
	
	// Insert into database
	query := `INSERT INTO ecse_students 
	          (name, dob, phone, address, iep_status, speech_therapy, occupational_therapy, 
	           physical_therapy, import_id, created_at) 
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, CURRENT_TIMESTAMP) 
	          RETURNING id`
	
	var recordID string
	err = ei.tx.QueryRow(query, 
		name, dob, phone, address, iepStatus, speechTherapy, 
		occupationalTherapy, physicalTherapy, ei.result.ImportID).Scan(&recordID)
	
	if err != nil {
		ei.addError(rowNum, "", sheet, row, fmt.Sprintf("Database error: %v", err), "DATABASE")
		return err
	}
	
	ei.result.RollbackInfo.RecordIDs = append(ei.result.RollbackInfo.RecordIDs, recordID)
	return nil
}

// processStudentRow processes a student data row
func (ei *ExcelImporter) processStudentRow(row []string, rowNum int, sheet string, columnMap map[string]int) error {
	// Extract required fields
	name := ei.getStringValue(row, columnMap, "name")
	if name == "" {
		ei.addError(rowNum, "name", sheet, "", "Student name is required", "REQUIRED_FIELD")
		return fmt.Errorf("missing student name")
	}
	
	// Validate name
	if err := ei.validator.ValidateField("name", name); err != nil {
		ei.addError(rowNum, "name", sheet, name, err.Error(), "VALIDATION")
		return err
	}
	
	// Extract and validate grade
	grade := ei.getStringValue(row, columnMap, "grade")
	if grade == "" {
		ei.addError(rowNum, "grade", sheet, "", "Grade is required", "REQUIRED_FIELD")
		return fmt.Errorf("missing grade")
	}
	
	if err := ei.validator.ValidateField("grade", grade); err != nil {
		ei.addError(rowNum, "grade", sheet, grade, err.Error(), "VALIDATION")
		return err
	}
	
	// Extract address
	address := ei.getStringValue(row, columnMap, "address")
	if address == "" {
		ei.addError(rowNum, "address", sheet, "", "Address is required", "REQUIRED_FIELD")
		return fmt.Errorf("missing address")
	}
	
	// Extract and validate phone
	phone := ei.getStringValue(row, columnMap, "phone")
	if phone == "" {
		ei.addError(rowNum, "phone", sheet, "", "Phone number is required", "REQUIRED_FIELD")
		return fmt.Errorf("missing phone number")
	}
	
	phone = NormalizePhone(phone)
	if err := ei.validator.ValidateField("phone", phone); err != nil {
		ei.addError(rowNum, "phone", sheet, phone, err.Error(), "VALIDATION")
		return err
	}
	
	// Extract optional fields
	guardian := ei.getStringValue(row, columnMap, "guardian")
	pickupTimeStr := ei.getStringValue(row, columnMap, "pickup_time")
	dropoffTimeStr := ei.getStringValue(row, columnMap, "dropoff_time")
	
	// Parse times if provided
	var pickupTime, dropoffTime *time.Time
	if pickupTimeStr != "" {
		if pt, err := ParseTime(pickupTimeStr); err == nil {
			pickupTime = &pt
		} else {
			ei.addWarning(rowNum, "pickup_time", sheet, pickupTimeStr, "Invalid time format", "FORMAT")
		}
	}
	
	if dropoffTimeStr != "" {
		if dt, err := ParseTime(dropoffTimeStr); err == nil {
			dropoffTime = &dt
		} else {
			ei.addWarning(rowNum, "dropoff_time", sheet, dropoffTimeStr, "Invalid time format", "FORMAT")
		}
	}
	
	// Insert into database
	query := `INSERT INTO students 
	          (name, grade, address, phone, guardian, pickup_time, dropoff_time, 
	           active, import_id, created_at) 
	          VALUES ($1, $2, $3, $4, $5, $6, $7, true, $8, CURRENT_TIMESTAMP) 
	          RETURNING id`
	
	var recordID string
	err := ei.tx.QueryRow(query, 
		name, grade, address, phone, guardian, pickupTime, dropoffTime, 
		ei.result.ImportID).Scan(&recordID)
	
	if err != nil {
		ei.addError(rowNum, "", sheet, row, fmt.Sprintf("Database error: %v", err), "DATABASE")
		return err
	}
	
	ei.result.RollbackInfo.RecordIDs = append(ei.result.RollbackInfo.RecordIDs, recordID)
	return nil
}

// processVehicleRow processes a vehicle data row
func (ei *ExcelImporter) processVehicleRow(row []string, rowNum int, sheet string, columnMap map[string]int) error {
	// Extract required fields
	vehicleID := ei.getStringValue(row, columnMap, "vehicle_id")
	if vehicleID == "" {
		ei.addError(rowNum, "vehicle_id", sheet, "", "Vehicle ID is required", "REQUIRED_FIELD")
		return fmt.Errorf("missing vehicle ID")
	}
	
	// Validate vehicle ID
	if err := ei.validator.ValidateField("vehicle_id", vehicleID); err != nil {
		ei.addError(rowNum, "vehicle_id", sheet, vehicleID, err.Error(), "VALIDATION")
		return err
	}
	
	// Extract and validate year
	yearStr := ei.getStringValue(row, columnMap, "year")
	if yearStr == "" {
		ei.addError(rowNum, "year", sheet, "", "Year is required", "REQUIRED_FIELD")
		return fmt.Errorf("missing year")
	}
	
	year := ei.getIntValue(row, columnMap, "year", rowNum, sheet)
	if year < 0 {
		return fmt.Errorf("invalid year")
	}
	
	if err := ei.validator.ValidateField("year", yearStr); err != nil {
		ei.addError(rowNum, "year", sheet, yearStr, err.Error(), "VALIDATION")
		return err
	}
	
	// Extract make and model
	make := ei.getStringValue(row, columnMap, "make")
	if make == "" {
		ei.addError(rowNum, "make", sheet, "", "Make is required", "REQUIRED_FIELD")
		return fmt.Errorf("missing make")
	}
	
	model := ei.getStringValue(row, columnMap, "model")
	if model == "" {
		ei.addError(rowNum, "model", sheet, "", "Model is required", "REQUIRED_FIELD")
		return fmt.Errorf("missing model")
	}
	
	// Extract optional fields
	vin := ei.getStringValue(row, columnMap, "vin")
	if vin != "" {
		if err := ei.validator.ValidateField("vin", vin); err != nil {
			ei.addWarning(rowNum, "vin", sheet, vin, err.Error(), "VALIDATION")
			vin = "" // Clear invalid VIN
		}
	}
	
	licensePlate := ei.getStringValue(row, columnMap, "license_plate")
	if licensePlate != "" {
		if err := ei.validator.ValidateField("license_plate", licensePlate); err != nil {
			ei.addWarning(rowNum, "license_plate", sheet, licensePlate, err.Error(), "VALIDATION")
		}
	}
	
	// Extract and validate status
	status := ei.getStringValue(row, columnMap, "status")
	if status == "" {
		status = "active" // Default status
	} else {
		if err := ei.validator.ValidateField("status", status); err != nil {
			ei.addWarning(rowNum, "status", sheet, status, err.Error(), "VALIDATION")
			status = "active" // Use default if invalid
		}
	}
	
	// Determine vehicle type based on ID pattern
	vehicleType := "company"
	if strings.HasPrefix(strings.ToLower(vehicleID), "bus") || 
	   strings.Contains(strings.ToLower(vehicleID), "bus") {
		vehicleType = "bus"
	}
	
	// Insert into database
	query := `INSERT INTO vehicles 
	          (vehicle_id, year, make, model, vin, license_plate, status, 
	           type, import_id, created_at) 
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, CURRENT_TIMESTAMP) 
	          ON CONFLICT (vehicle_id) 
	          DO UPDATE SET
	              year = EXCLUDED.year,
	              make = EXCLUDED.make,
	              model = EXCLUDED.model,
	              vin = EXCLUDED.vin,
	              license_plate = EXCLUDED.license_plate,
	              status = EXCLUDED.status,
	              updated_at = CURRENT_TIMESTAMP
	          RETURNING id`
	
	var recordID string
	err := ei.tx.QueryRow(query, 
		vehicleID, year, make, model, vin, licensePlate, status, 
		vehicleType, ei.result.ImportID).Scan(&recordID)
	
	if err != nil {
		ei.addError(rowNum, "", sheet, row, fmt.Sprintf("Database error: %v", err), "DATABASE")
		return err
	}
	
	ei.result.RollbackInfo.RecordIDs = append(ei.result.RollbackInfo.RecordIDs, recordID)
	return nil
}

// addWarning adds a warning to the result
func (ei *ExcelImporter) addWarning(row int, column, sheet string, value interface{}, warning, warningType string) {
	ei.result.Warnings = append(ei.result.Warnings, ImportError{
		Row:       row,
		Column:    column,
		Sheet:     sheet,
		Value:     value,
		Error:     warning,
		ErrorType: warningType,
		Severity:  "warning",
	})
	ei.result.WarningCount++
}