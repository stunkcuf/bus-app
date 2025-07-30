package main

import (
	"log"
)

func FixSchemaIssues() error {
	log.Println("🔧 Fixing database schema issues...")

	// Drop and recreate the attendance_summary view
	queries := []string{
		// Drop the existing view if it exists
		`DROP VIEW IF EXISTS attendance_summary CASCADE`,

		// Recreate with correct column references
		`CREATE OR REPLACE VIEW attendance_summary AS
		SELECT 
			s.student_id,
			s.name as student_name,
			'' as grade,
			r.route_name,
			COUNT(CASE WHEN sa.status = 'present' THEN 1 END) as days_present,
			COUNT(CASE WHEN sa.status = 'absent' THEN 1 END) as days_absent,
			COUNT(CASE WHEN sa.status = 'late' THEN 1 END) as days_late,
			COUNT(sa.attendance_date) as total_days_recorded
		FROM students s
		LEFT JOIN student_attendance sa ON s.student_id = sa.student_id
		LEFT JOIN routes r ON s.route_id = r.route_id
		GROUP BY s.student_id, s.name, r.route_name`,

		// Add missing subject column to notifications table
		`ALTER TABLE notifications ADD COLUMN IF NOT EXISTS subject TEXT`,

		// Ensure all required columns exist
		`ALTER TABLE students ADD COLUMN IF NOT EXISTS student_name VARCHAR(100)`,
		
		// Update student_name from name column if it exists
		`UPDATE students SET student_name = name WHERE student_name IS NULL AND name IS NOT NULL`,
	}

	for i, query := range queries {
		log.Printf("Executing fix %d...", i+1)
		if _, err := db.Exec(query); err != nil {
			log.Printf("Warning: Query %d failed: %v", i+1, err)
			// Continue with other fixes
		}
	}

	log.Println("✅ Schema fixes applied")
	return nil
}