package main

import (
	"fmt"
	"log"

	"github.com/jmoiron/sqlx"
)

// InitializeMobileAppTables creates database tables for mobile app features
func InitializeMobileAppTables(db *sqlx.DB) error {
	log.Println("📱 Creating mobile app database tables...")

	queries := []string{
		// Student attendance table
		`CREATE TABLE IF NOT EXISTS student_attendance (
			id SERIAL PRIMARY KEY,
			student_id VARCHAR(50) NOT NULL REFERENCES students(student_id),
			attendance_date DATE NOT NULL,
			status VARCHAR(20) NOT NULL CHECK (status IN ('present', 'absent', 'excused', 'late')),
			boarded_at TIME,
			dropped_at TIME,
			notes TEXT,
			recorded_by VARCHAR(50) REFERENCES users(username),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(student_id, attendance_date)
		)`,

		// Index for attendance queries
		`CREATE INDEX IF NOT EXISTS idx_attendance_date ON student_attendance(attendance_date)`,
		`CREATE INDEX IF NOT EXISTS idx_attendance_student ON student_attendance(student_id)`,
		`CREATE INDEX IF NOT EXISTS idx_attendance_recorded_by ON student_attendance(recorded_by)`,

		// Driver locations table for real-time tracking
		`CREATE TABLE IF NOT EXISTS driver_locations (
			driver_username VARCHAR(50) PRIMARY KEY REFERENCES users(username),
			latitude DOUBLE PRECISION NOT NULL,
			longitude DOUBLE PRECISION NOT NULL,
			speed DOUBLE PRECISION DEFAULT 0,
			heading DOUBLE PRECISION DEFAULT 0,
			accuracy DOUBLE PRECISION DEFAULT 0,
			updated_at TIMESTAMP NOT NULL,
			CHECK (latitude >= -90 AND latitude <= 90),
			CHECK (longitude >= -180 AND longitude <= 180)
		)`,

		// Issue reports table
		`CREATE TABLE IF NOT EXISTS issue_reports (
			issue_id SERIAL PRIMARY KEY,
			reported_by VARCHAR(50) NOT NULL REFERENCES users(username),
			type VARCHAR(50) NOT NULL,
			description TEXT NOT NULL,
			vehicle_id VARCHAR(50) REFERENCES vehicles(vehicle_id),
			route_id VARCHAR(50) REFERENCES routes(route_id),
			severity VARCHAR(20) NOT NULL CHECK (severity IN ('low', 'medium', 'high', 'critical')),
			status VARCHAR(20) DEFAULT 'open' CHECK (status IN ('open', 'in_progress', 'resolved', 'closed')),
			latitude DOUBLE PRECISION,
			longitude DOUBLE PRECISION,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			resolved_at TIMESTAMP,
			resolved_by VARCHAR(50) REFERENCES users(username),
			resolution_notes TEXT
		)`,

		// Index for issue queries
		`CREATE INDEX IF NOT EXISTS idx_issues_status ON issue_reports(status)`,
		`CREATE INDEX IF NOT EXISTS idx_issues_severity ON issue_reports(severity)`,
		`CREATE INDEX IF NOT EXISTS idx_issues_reported_by ON issue_reports(reported_by)`,
		`CREATE INDEX IF NOT EXISTS idx_issues_vehicle ON issue_reports(vehicle_id)`,
		`CREATE INDEX IF NOT EXISTS idx_issues_created ON issue_reports(created_at DESC)`,

		// Issue attachments table for photos
		`CREATE TABLE IF NOT EXISTS issue_attachments (
			attachment_id SERIAL PRIMARY KEY,
			issue_id INTEGER NOT NULL REFERENCES issue_reports(issue_id) ON DELETE CASCADE,
			file_path TEXT NOT NULL,
			file_type VARCHAR(50),
			file_size INTEGER,
			uploaded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// Mobile device sessions for push notifications
		`CREATE TABLE IF NOT EXISTS mobile_sessions (
			session_id SERIAL PRIMARY KEY,
			username VARCHAR(50) NOT NULL REFERENCES users(username),
			device_id VARCHAR(255) NOT NULL,
			platform VARCHAR(20) NOT NULL CHECK (platform IN ('ios', 'android')),
			push_token TEXT,
			app_version VARCHAR(20),
			last_active TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(username, device_id)
		)`,

		// Attendance summary view
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

		// Issue statistics view
		`CREATE OR REPLACE VIEW issue_statistics AS
		SELECT 
			DATE_TRUNC('month', created_at) as month,
			type,
			severity,
			COUNT(*) as issue_count,
			COUNT(CASE WHEN status = 'resolved' THEN 1 END) as resolved_count,
			AVG(EXTRACT(EPOCH FROM (resolved_at - created_at))/3600) as avg_resolution_hours
		FROM issue_reports
		GROUP BY DATE_TRUNC('month', created_at), type, severity`,
	}

	// Execute each query
	for _, query := range queries {
		if _, err := db.Exec(query); err != nil {
			return fmt.Errorf("failed to execute query: %w", err)
		}
	}

	log.Println("✅ Mobile app database tables created successfully")
	return nil
}

// CreateMobileAppTriggers creates database triggers for mobile app features
func CreateMobileAppTriggers(db *sqlx.DB) error {
	triggers := []string{
		// Auto-update timestamp trigger for attendance
		`CREATE OR REPLACE FUNCTION update_attendance_timestamp()
		RETURNS TRIGGER AS $$
		BEGIN
			NEW.updated_at = CURRENT_TIMESTAMP;
			RETURN NEW;
		END;
		$$ LANGUAGE plpgsql`,

		`DROP TRIGGER IF EXISTS update_attendance_timestamp ON student_attendance`,
		
		`CREATE TRIGGER update_attendance_timestamp
		BEFORE UPDATE ON student_attendance
		FOR EACH ROW EXECUTE FUNCTION update_attendance_timestamp()`,

		// Auto-update timestamp trigger for issues
		`CREATE OR REPLACE FUNCTION update_issue_timestamp()
		RETURNS TRIGGER AS $$
		BEGIN
			NEW.updated_at = CURRENT_TIMESTAMP;
			RETURN NEW;
		END;
		$$ LANGUAGE plpgsql`,

		`DROP TRIGGER IF EXISTS update_issue_timestamp ON issue_reports`,
		
		`CREATE TRIGGER update_issue_timestamp
		BEFORE UPDATE ON issue_reports
		FOR EACH ROW EXECUTE FUNCTION update_issue_timestamp()`,

		// Update last_active for mobile sessions
		`CREATE OR REPLACE FUNCTION update_mobile_session_activity()
		RETURNS TRIGGER AS $$
		BEGIN
			NEW.last_active = CURRENT_TIMESTAMP;
			RETURN NEW;
		END;
		$$ LANGUAGE plpgsql`,

		`DROP TRIGGER IF EXISTS update_mobile_session_activity ON mobile_sessions`,
		
		`CREATE TRIGGER update_mobile_session_activity
		BEFORE UPDATE ON mobile_sessions
		FOR EACH ROW EXECUTE FUNCTION update_mobile_session_activity()`,
	}

	for _, trigger := range triggers {
		if _, err := db.Exec(trigger); err != nil {
			log.Printf("Warning: Failed to create trigger: %v", err)
			// Continue with other triggers
		}
	}

	return nil
}