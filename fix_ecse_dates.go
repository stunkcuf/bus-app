package main

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"
)

// FixECSEDateIssues fixes invalid date formats in ECSE-related tables
func FixECSEDateIssues() error {
	log.Println("[ECSE FIX] Starting ECSE date format fixes...")
	
	// Check if ecse_students table exists
	var tableExists bool
	err := db.Get(&tableExists, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables 
			WHERE table_name = 'ecse_students'
		)
	`)
	
	if err != nil {
		return fmt.Errorf("failed to check if ecse_students table exists: %w", err)
	}
	
	if !tableExists {
		log.Println("[ECSE FIX] ecse_students table does not exist, skipping fixes")
		return nil
	}
	
	// Fix birthdate column issues
	fixes := []struct {
		name  string
		query string
	}{
		{
			name: "Fix empty string birthdates",
			query: `
				UPDATE ecse_students 
				SET birthdate = NULL 
				WHERE birthdate = '' OR birthdate = '0000-00-00'
			`,
		},
		{
			name: "Fix invalid date formats",
			query: `
				UPDATE ecse_students 
				SET birthdate = NULL 
				WHERE birthdate IS NOT NULL 
				AND birthdate !~ '^\d{4}-\d{2}-\d{2}$'
			`,
		},
		{
			name: "Fix enrollment dates",
			query: `
				UPDATE ecse_students 
				SET enrollment_date = NULL 
				WHERE enrollment_date = '' OR enrollment_date = '0000-00-00'
			`,
		},
		{
			name: "Fix assessment dates",
			query: `
				UPDATE ecse_assessments 
				SET assessment_date = NULL 
				WHERE assessment_date = '' OR assessment_date = '0000-00-00'
			`,
		},
		{
			name: "Fix service dates",
			query: `
				UPDATE ecse_services 
				SET service_date = NULL 
				WHERE service_date = '' OR service_date = '0000-00-00'
			`,
		},
	}
	
	for _, fix := range fixes {
		log.Printf("[ECSE FIX] Applying: %s", fix.name)
		
		// Check if table exists before running update
		tableName := extractTableName(fix.query)
		if tableName != "" && tableName != "ecse_students" {
			var exists bool
			err := db.Get(&exists, `
				SELECT EXISTS (
					SELECT 1 FROM information_schema.tables 
					WHERE table_name = $1
				)
			`, tableName)
			
			if err != nil || !exists {
				log.Printf("[ECSE FIX] Table %s does not exist, skipping", tableName)
				continue
			}
		}
		
		result, err := db.Exec(fix.query)
		if err != nil {
			log.Printf("[ECSE FIX] Warning: Failed to apply %s: %v", fix.name, err)
			continue
		}
		
		rowsAffected, _ := result.RowsAffected()
		log.Printf("[ECSE FIX] Fixed %d rows for: %s", rowsAffected, fix.name)
	}
	
	// Add constraints to prevent future issues
	constraints := []struct {
		name  string
		query string
	}{
		{
			name: "Add birthdate check constraint",
			query: `
				ALTER TABLE ecse_students 
				ADD CONSTRAINT check_birthdate_format 
				CHECK (birthdate IS NULL OR birthdate ~ '^\d{4}-\d{2}-\d{2}$')
			`,
		},
		{
			name: "Add enrollment_date check constraint",
			query: `
				ALTER TABLE ecse_students 
				ADD CONSTRAINT check_enrollment_date_format 
				CHECK (enrollment_date IS NULL OR enrollment_date ~ '^\d{4}-\d{2}-\d{2}$')
			`,
		},
	}
	
	for _, constraint := range constraints {
		log.Printf("[ECSE FIX] Adding constraint: %s", constraint.name)
		
		// Check if constraint already exists
		var exists bool
		constraintName := extractConstraintName(constraint.query)
		if constraintName != "" {
			err := db.Get(&exists, `
				SELECT EXISTS (
					SELECT 1 FROM information_schema.table_constraints 
					WHERE constraint_name = $1
				)
			`, constraintName)
			
			if err == nil && exists {
				log.Printf("[ECSE FIX] Constraint %s already exists, skipping", constraintName)
				continue
			}
		}
		
		if _, err := db.Exec(constraint.query); err != nil {
			log.Printf("[ECSE FIX] Warning: Failed to add constraint %s: %v", constraint.name, err)
		}
	}
	
	log.Println("[ECSE FIX] ECSE date format fixes completed")
	return nil
}

// ValidateECSEDate validates and formats a date string
func ValidateECSEDate(dateStr string) (sql.NullString, error) {
	result := sql.NullString{String: "", Valid: false}
	
	if dateStr == "" || dateStr == "0000-00-00" {
		return result, nil
	}
	
	// Try to parse the date
	formats := []string{
		"2006-01-02",
		"01/02/2006",
		"1/2/2006",
		"2006/01/02",
	}
	
	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			result.String = t.Format("2006-01-02")
			result.Valid = true
			return result, nil
		}
	}
	
	return result, fmt.Errorf("invalid date format: %s", dateStr)
}

// extractTableName extracts table name from UPDATE query
func extractTableName(query string) string {
	query = strings.ToLower(query)
	if strings.Contains(query, "update ") {
		parts := strings.Fields(query)
		for i, part := range parts {
			if part == "update" && i+1 < len(parts) {
				return strings.TrimSpace(parts[i+1])
			}
		}
	}
	return ""
}

// extractConstraintName extracts constraint name from ALTER TABLE query
func extractConstraintName(query string) string {
	query = strings.ToLower(query)
	if strings.Contains(query, "constraint ") {
		parts := strings.Fields(query)
		for i, part := range parts {
			if part == "constraint" && i+1 < len(parts) {
				return strings.TrimSpace(parts[i+1])
			}
		}
	}
	return ""
}

// CreateECSETables creates ECSE-related tables if they don't exist
func CreateECSETables() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS ecse_students (
			student_id VARCHAR(50) PRIMARY KEY,
			first_name VARCHAR(100),
			last_name VARCHAR(100),
			birthdate DATE,
			iep_status VARCHAR(50),
			disability_category VARCHAR(100),
			enrollment_date DATE,
			parent_contact VARCHAR(100),
			address TEXT,
			notes TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		
		`CREATE TABLE IF NOT EXISTS ecse_services (
			service_id SERIAL PRIMARY KEY,
			student_id VARCHAR(50) REFERENCES ecse_students(student_id),
			service_type VARCHAR(100),
			frequency VARCHAR(50),
			duration VARCHAR(50),
			provider VARCHAR(100),
			service_date DATE,
			notes TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		
		`CREATE TABLE IF NOT EXISTS ecse_assessments (
			assessment_id SERIAL PRIMARY KEY,
			student_id VARCHAR(50) REFERENCES ecse_students(student_id),
			assessment_type VARCHAR(100),
			assessment_date DATE,
			score VARCHAR(50),
			evaluator VARCHAR(100),
			notes TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		
		`CREATE TABLE IF NOT EXISTS ecse_attendance (
			attendance_id SERIAL PRIMARY KEY,
			student_id VARCHAR(50) REFERENCES ecse_students(student_id),
			date DATE,
			status VARCHAR(20),
			notes TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
	}
	
	for i, query := range queries {
		if _, err := db.Exec(query); err != nil {
			return fmt.Errorf("failed to create ECSE table %d: %w", i, err)
		}
	}
	
	// Create indexes
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_ecse_students_name ON ecse_students(last_name, first_name)",
		"CREATE INDEX IF NOT EXISTS idx_ecse_services_student ON ecse_services(student_id)",
		"CREATE INDEX IF NOT EXISTS idx_ecse_assessments_student ON ecse_assessments(student_id)",
		"CREATE INDEX IF NOT EXISTS idx_ecse_attendance_student ON ecse_attendance(student_id, date)",
	}
	
	for _, idx := range indexes {
		if _, err := db.Exec(idx); err != nil {
			log.Printf("Warning: Failed to create index: %v", err)
		}
	}
	
	return nil
}