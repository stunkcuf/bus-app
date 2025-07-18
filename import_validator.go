package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ImportValidator validates data for different import types
type ImportValidator struct {
	importType ImportType
	rules      map[string][]ValidationFunc
}

// ValidationFunc is a function that validates a value
type ValidationFunc func(value string, fieldName string) error

// NewImportValidator creates a new validator for the given import type
func NewImportValidator(importType ImportType) *ImportValidator {
	v := &ImportValidator{
		importType: importType,
		rules:      make(map[string][]ValidationFunc),
	}
	
	// Initialize rules based on import type
	switch importType {
	case ImportTypeMileage:
		v.initMileageRules()
	case ImportTypeECSE:
		v.initECSERules()
	case ImportTypeStudent:
		v.initStudentRules()
	case ImportTypeVehicle:
		v.initVehicleRules()
	}
	
	return v
}

// GetExpectedHeaders returns expected headers for the import type
func (v *ImportValidator) GetExpectedHeaders() []string {
	switch v.importType {
	case ImportTypeMileage:
		return []string{"vehicle", "beginning", "ending", "total", "miles", "id", "location"}
	case ImportTypeECSE:
		return []string{"name", "dob", "phone", "address", "iep", "speech", "ot", "pt"}
	case ImportTypeStudent:
		return []string{"name", "grade", "address", "phone", "guardian", "pickup", "dropoff"}
	case ImportTypeVehicle:
		return []string{"vehicle", "year", "make", "model", "vin", "license", "status"}
	default:
		return []string{}
	}
}

// GetRequiredColumns returns required columns for the import type
func (v *ImportValidator) GetRequiredColumns() []string {
	switch v.importType {
	case ImportTypeMileage:
		return []string{"vehicle_id", "beginning_mileage", "ending_mileage"}
	case ImportTypeECSE:
		return []string{"name", "dob", "phone"}
	case ImportTypeStudent:
		return []string{"name", "grade", "address", "phone"}
	case ImportTypeVehicle:
		return []string{"vehicle_id", "year", "make", "model"}
	default:
		return []string{}
	}
}

// ValidateField validates a single field
func (v *ImportValidator) ValidateField(fieldName, value string) error {
	rules, exists := v.rules[fieldName]
	if !exists {
		return nil // No rules for this field
	}
	
	for _, rule := range rules {
		if err := rule(value, fieldName); err != nil {
			return err
		}
	}
	
	return nil
}

// Initialize validation rules for mileage imports
func (v *ImportValidator) initMileageRules() {
	v.rules["vehicle_id"] = []ValidationFunc{
		requiredField,
		vehicleIDFormat,
	}
	
	v.rules["beginning_mileage"] = []ValidationFunc{
		requiredField,
		numericField,
		positiveNumber,
		maxMileage,
	}
	
	v.rules["ending_mileage"] = []ValidationFunc{
		requiredField,
		numericField,
		positiveNumber,
		maxMileage,
	}
	
	v.rules["date"] = []ValidationFunc{
		dateFormat,
	}
}

// Initialize validation rules for ECSE imports
func (v *ImportValidator) initECSERules() {
	v.rules["name"] = []ValidationFunc{
		requiredField,
		nameFormat,
		maxLength(100),
	}
	
	v.rules["dob"] = []ValidationFunc{
		requiredField,
		dateFormat,
		ageRange(0, 21), // ECSE students typically up to 21
	}
	
	v.rules["phone"] = []ValidationFunc{
		requiredField,
		phoneFormat,
	}
	
	v.rules["address"] = []ValidationFunc{
		maxLength(200),
	}
	
	v.rules["iep_status"] = []ValidationFunc{
		booleanField,
	}
	
	v.rules["speech_therapy"] = []ValidationFunc{
		booleanField,
	}
	
	v.rules["occupational_therapy"] = []ValidationFunc{
		booleanField,
	}
	
	v.rules["physical_therapy"] = []ValidationFunc{
		booleanField,
	}
}

// Initialize validation rules for student imports
func (v *ImportValidator) initStudentRules() {
	v.rules["name"] = []ValidationFunc{
		requiredField,
		nameFormat,
		maxLength(100),
	}
	
	v.rules["grade"] = []ValidationFunc{
		requiredField,
		gradeLevel,
	}
	
	v.rules["address"] = []ValidationFunc{
		requiredField,
		maxLength(200),
	}
	
	v.rules["phone"] = []ValidationFunc{
		requiredField,
		phoneFormat,
	}
	
	v.rules["guardian"] = []ValidationFunc{
		nameFormat,
		maxLength(100),
	}
	
	v.rules["pickup_time"] = []ValidationFunc{
		timeFormat,
	}
	
	v.rules["dropoff_time"] = []ValidationFunc{
		timeFormat,
	}
}

// Initialize validation rules for vehicle imports
func (v *ImportValidator) initVehicleRules() {
	v.rules["vehicle_id"] = []ValidationFunc{
		requiredField,
		vehicleIDFormat,
	}
	
	v.rules["year"] = []ValidationFunc{
		requiredField,
		numericField,
		yearRange(1900, time.Now().Year()+2),
	}
	
	v.rules["make"] = []ValidationFunc{
		requiredField,
		maxLength(50),
	}
	
	v.rules["model"] = []ValidationFunc{
		requiredField,
		maxLength(50),
	}
	
	v.rules["vin"] = []ValidationFunc{
		vinFormat,
	}
	
	v.rules["license_plate"] = []ValidationFunc{
		licensePlateFormat,
		maxLength(20),
	}
	
	v.rules["status"] = []ValidationFunc{
		vehicleStatus,
	}
}

// Validation functions

func requiredField(value, fieldName string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s is required", fieldName)
	}
	return nil
}

func numericField(value, fieldName string) error {
	if value == "" {
		return nil
	}
	if _, err := strconv.Atoi(value); err != nil {
		return fmt.Errorf("%s must be a number", fieldName)
	}
	return nil
}

func positiveNumber(value, fieldName string) error {
	if value == "" {
		return nil
	}
	num, err := strconv.Atoi(value)
	if err != nil {
		return nil // Let numericField handle this
	}
	if num < 0 {
		return fmt.Errorf("%s must be positive", fieldName)
	}
	return nil
}

func maxMileage(value, fieldName string) error {
	if value == "" {
		return nil
	}
	num, err := strconv.Atoi(value)
	if err != nil {
		return nil
	}
	if num > 999999 {
		return fmt.Errorf("%s exceeds maximum allowed value", fieldName)
	}
	return nil
}

func vehicleIDFormat(value, fieldName string) error {
	if value == "" {
		return nil
	}
	// Allow alphanumeric with dashes and underscores
	if !regexp.MustCompile(`^[A-Za-z0-9_-]+$`).MatchString(value) {
		return fmt.Errorf("%s contains invalid characters", fieldName)
	}
	return nil
}

func dateFormat(value, fieldName string) error {
	if value == "" {
		return nil
	}
	
	// Try common date formats
	formats := []string{
		"2006-01-02",
		"01/02/2006",
		"01-02-2006",
		"1/2/2006",
		"1-2-2006",
	}
	
	for _, format := range formats {
		if _, err := time.Parse(format, value); err == nil {
			return nil
		}
	}
	
	return fmt.Errorf("%s must be a valid date (MM/DD/YYYY or YYYY-MM-DD)", fieldName)
}

func timeFormat(value, fieldName string) error {
	if value == "" {
		return nil
	}
	
	// Try common time formats
	formats := []string{
		"15:04",
		"3:04 PM",
		"3:04PM",
		"15:04:05",
	}
	
	for _, format := range formats {
		if _, err := time.Parse(format, value); err == nil {
			return nil
		}
	}
	
	return fmt.Errorf("%s must be a valid time (HH:MM or HH:MM AM/PM)", fieldName)
}

func phoneFormat(value, fieldName string) error {
	if value == "" {
		return nil
	}
	
	// Remove common formatting characters
	cleaned := regexp.MustCompile(`[^\d]`).ReplaceAllString(value, "")
	
	// Check length
	if len(cleaned) != 10 && len(cleaned) != 11 {
		return fmt.Errorf("%s must be a valid phone number", fieldName)
	}
	
	return nil
}

func nameFormat(value, fieldName string) error {
	if value == "" {
		return nil
	}
	
	// Check for invalid characters
	if regexp.MustCompile(`[0-9<>\"'%;()&+]`).MatchString(value) {
		return fmt.Errorf("%s contains invalid characters", fieldName)
	}
	
	return nil
}

func gradeLevel(value, fieldName string) error {
	if value == "" {
		return nil
	}
	
	// Allow K, PK, numbers 1-12
	upperValue := strings.ToUpper(value)
	if upperValue == "K" || upperValue == "PK" || upperValue == "PREK" {
		return nil
	}
	
	if grade, err := strconv.Atoi(value); err == nil {
		if grade >= 1 && grade <= 12 {
			return nil
		}
	}
	
	return fmt.Errorf("%s must be a valid grade level (PK, K, 1-12)", fieldName)
}

func vinFormat(value, fieldName string) error {
	if value == "" {
		return nil
	}
	
	// VIN should be 17 characters
	if len(value) != 17 {
		return fmt.Errorf("%s must be 17 characters", fieldName)
	}
	
	// VIN should be alphanumeric (excluding I, O, Q)
	if !regexp.MustCompile(`^[A-HJ-NPR-Z0-9]{17}$`).MatchString(strings.ToUpper(value)) {
		return fmt.Errorf("%s contains invalid VIN characters", fieldName)
	}
	
	return nil
}

func licensePlateFormat(value, fieldName string) error {
	if value == "" {
		return nil
	}
	
	// Allow alphanumeric with spaces and dashes
	if !regexp.MustCompile(`^[A-Za-z0-9 -]+$`).MatchString(value) {
		return fmt.Errorf("%s contains invalid characters", fieldName)
	}
	
	return nil
}

func vehicleStatus(value, fieldName string) error {
	if value == "" {
		return nil
	}
	
	validStatuses := []string{"active", "maintenance", "out_of_service", "retired", "for_sale", "sold"}
	valueLower := strings.ToLower(value)
	
	for _, status := range validStatuses {
		if valueLower == status {
			return nil
		}
	}
	
	return fmt.Errorf("%s must be one of: %s", fieldName, strings.Join(validStatuses, ", "))
}

func booleanField(value, fieldName string) error {
	if value == "" {
		return nil
	}
	
	valueLower := strings.ToLower(value)
	validValues := []string{"true", "false", "yes", "no", "y", "n", "1", "0"}
	
	for _, valid := range validValues {
		if valueLower == valid {
			return nil
		}
	}
	
	return fmt.Errorf("%s must be a boolean value (yes/no, true/false, 1/0)", fieldName)
}

// Helper functions to create validation functions with parameters

func maxLength(max int) ValidationFunc {
	return func(value, fieldName string) error {
		if len(value) > max {
			return fmt.Errorf("%s exceeds maximum length of %d characters", fieldName, max)
		}
		return nil
	}
}

func minLength(min int) ValidationFunc {
	return func(value, fieldName string) error {
		if value != "" && len(value) < min {
			return fmt.Errorf("%s must be at least %d characters", fieldName, min)
		}
		return nil
	}
}

func yearRange(min, max int) ValidationFunc {
	return func(value, fieldName string) error {
		if value == "" {
			return nil
		}
		year, err := strconv.Atoi(value)
		if err != nil {
			return nil // Let numericField handle this
		}
		if year < min || year > max {
			return fmt.Errorf("%s must be between %d and %d", fieldName, min, max)
		}
		return nil
	}
}

func ageRange(minAge, maxAge int) ValidationFunc {
	return func(value, fieldName string) error {
		if value == "" {
			return nil
		}
		
		// Parse date
		var dob time.Time
		formats := []string{"2006-01-02", "01/02/2006", "01-02-2006"}
		
		for _, format := range formats {
			if parsed, err := time.Parse(format, value); err == nil {
				dob = parsed
				break
			}
		}
		
		if dob.IsZero() {
			return nil // Let dateFormat handle this
		}
		
		// Calculate age
		now := time.Now()
		age := now.Year() - dob.Year()
		if now.YearDay() < dob.YearDay() {
			age--
		}
		
		if age < minAge || age > maxAge {
			return fmt.Errorf("age must be between %d and %d years", minAge, maxAge)
		}
		
		return nil
	}
}

// ParseBoolean converts various boolean representations to bool
func ParseBoolean(value string) bool {
	valueLower := strings.ToLower(strings.TrimSpace(value))
	return valueLower == "true" || valueLower == "yes" || valueLower == "y" || valueLower == "1"
}

// ParseDate attempts to parse a date string in various formats
func ParseDate(value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, fmt.Errorf("empty date")
	}
	
	formats := []string{
		"2006-01-02",
		"01/02/2006",
		"01-02-2006",
		"1/2/2006",
		"1-2-2006",
		"Jan 2, 2006",
		"January 2, 2006",
	}
	
	for _, format := range formats {
		if parsed, err := time.Parse(format, value); err == nil {
			return parsed, nil
		}
	}
	
	return time.Time{}, fmt.Errorf("unable to parse date: %s", value)
}

// ParseTime attempts to parse a time string in various formats
func ParseTime(value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, fmt.Errorf("empty time")
	}
	
	formats := []string{
		"15:04",
		"3:04 PM",
		"3:04PM",
		"15:04:05",
		"3:04:05 PM",
	}
	
	for _, format := range formats {
		if parsed, err := time.Parse(format, value); err == nil {
			return parsed, nil
		}
	}
	
	return time.Time{}, fmt.Errorf("unable to parse time: %s", value)
}

// NormalizePhone normalizes a phone number to a standard format
func NormalizePhone(phone string) string {
	// Remove all non-numeric characters
	cleaned := regexp.MustCompile(`[^\d]`).ReplaceAllString(phone, "")
	
	// Format as (XXX) XXX-XXXX if 10 digits
	if len(cleaned) == 10 {
		return fmt.Sprintf("(%s) %s-%s", cleaned[:3], cleaned[3:6], cleaned[6:])
	}
	
	// Format as X (XXX) XXX-XXXX if 11 digits (with country code)
	if len(cleaned) == 11 && cleaned[0] == '1' {
		return fmt.Sprintf("%s (%s) %s-%s", cleaned[0:1], cleaned[1:4], cleaned[4:7], cleaned[7:])
	}
	
	return phone // Return original if can't format
}