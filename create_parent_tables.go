package main

import (
	"database/sql"
	"fmt"
	"log"
)

// createParentTables creates all tables needed for parent portal functionality
func createParentTables(db *sql.DB) error {
	queries := []string{
		// Parents table
		`CREATE TABLE IF NOT EXISTS parents (
			id SERIAL PRIMARY KEY,
			username VARCHAR(100) UNIQUE NOT NULL,
			email VARCHAR(255) UNIQUE NOT NULL,
			phone VARCHAR(20),
			name VARCHAR(255) NOT NULL,
			password VARCHAR(255) NOT NULL,
			notifications BOOLEAN DEFAULT true,
			active BOOLEAN DEFAULT true,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			last_login TIMESTAMP,
			password_reset_token VARCHAR(255),
			password_reset_expires TIMESTAMP
		)`,
		
		// Parent-Student relationship
		`CREATE TABLE IF NOT EXISTS parent_students (
			parent_id INTEGER REFERENCES parents(id) ON DELETE CASCADE,
			student_id VARCHAR(50) REFERENCES students(student_id) ON DELETE CASCADE,
			relationship VARCHAR(50) NOT NULL,
			emergency_rank INTEGER DEFAULT 2,
			PRIMARY KEY (parent_id, student_id)
		)`,
		
		// Parent sessions
		`CREATE TABLE IF NOT EXISTS parent_sessions (
			token VARCHAR(255) PRIMARY KEY,
			parent_id INTEGER REFERENCES parents(id) ON DELETE CASCADE,
			expires_at TIMESTAMP NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		
		// Parent notifications
		`CREATE TABLE IF NOT EXISTS parent_notifications (
			id VARCHAR(100) PRIMARY KEY,
			parent_id INTEGER REFERENCES parents(id) ON DELETE CASCADE,
			type VARCHAR(50) NOT NULL,
			title VARCHAR(255) NOT NULL,
			message TEXT NOT NULL,
			student_id VARCHAR(50) REFERENCES students(student_id),
			read BOOLEAN DEFAULT false,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		
		// Parent notification settings
		`CREATE TABLE IF NOT EXISTS parent_notification_settings (
			parent_id INTEGER REFERENCES parents(id) ON DELETE CASCADE,
			bus_arrival BOOLEAN DEFAULT true,
			bus_departure BOOLEAN DEFAULT true,
			attendance BOOLEAN DEFAULT true,
			emergency BOOLEAN DEFAULT true,
			route_changes BOOLEAN DEFAULT true,
			email_enabled BOOLEAN DEFAULT true,
			sms_enabled BOOLEAN DEFAULT false,
			push_enabled BOOLEAN DEFAULT false,
			quiet_hours_start TIME,
			quiet_hours_end TIME,
			PRIMARY KEY (parent_id)
		)`,
		
		// Student registration codes
		`CREATE TABLE IF NOT EXISTS student_codes (
			code VARCHAR(20) PRIMARY KEY,
			student_id VARCHAR(50) REFERENCES students(student_id) ON DELETE CASCADE,
			used BOOLEAN DEFAULT false,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			expires_at TIMESTAMP NOT NULL,
			created_by VARCHAR(100)
		)`,
		
		// Temporary passwords for parent accounts
		`CREATE TABLE IF NOT EXISTS temp_passwords (
			parent_id INTEGER REFERENCES parents(id) ON DELETE CASCADE,
			password VARCHAR(255) NOT NULL,
			expires_at TIMESTAMP NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (parent_id)
		)`,
		
		// Parent activity log
		`CREATE TABLE IF NOT EXISTS parent_activity_log (
			id BIGSERIAL PRIMARY KEY,
			parent_id INTEGER REFERENCES parents(id) ON DELETE CASCADE,
			action VARCHAR(100) NOT NULL,
			details JSONB,
			ip_address VARCHAR(45),
			user_agent TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		
		// Create indexes
		`CREATE INDEX IF NOT EXISTS idx_parent_sessions_expires ON parent_sessions(expires_at)`,
		`CREATE INDEX IF NOT EXISTS idx_parent_notifications_parent ON parent_notifications(parent_id, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_parent_notifications_unread ON parent_notifications(parent_id) WHERE read = false`,
		`CREATE INDEX IF NOT EXISTS idx_student_codes_expires ON student_codes(expires_at)`,
		`CREATE INDEX IF NOT EXISTS idx_parent_activity_log_parent ON parent_activity_log(parent_id, created_at DESC)`,
	}

	for _, query := range queries {
		if _, err := db.Exec(query); err != nil {
			return fmt.Errorf("failed to create parent tables: %w", err)
		}
	}

	log.Println("Parent portal tables created successfully")
	return nil
}

// Helper functions for parent portal

func getParentNotificationSettings(parentID int) NotificationSettings {
	var settings NotificationSettings
	
	err := db.QueryRow(`
		SELECT bus_arrival, bus_departure, attendance, emergency, route_changes,
		       email_enabled, sms_enabled, push_enabled
		FROM parent_notification_settings
		WHERE parent_id = $1
	`, parentID).Scan(
		&settings.BusArrival, &settings.BusDeparture, &settings.Attendance,
		&settings.Emergency, &settings.RouteChanges, &settings.EmailEnabled,
		&settings.SMSEnabled, &settings.PushEnabled,
	)
	
	if err != nil {
		// Return default settings if none exist
		return NotificationSettings{
			BusArrival:   true,
			BusDeparture: true,
			Attendance:   true,
			Emergency:    true,
			RouteChanges: true,
			EmailEnabled: true,
			SMSEnabled:   false,
			PushEnabled:  false,
		}
	}
	
	return settings
}

func updateParentNotificationSettings(parentID int, settings NotificationSettings) error {
	_, err := db.Exec(`
		INSERT INTO parent_notification_settings 
		(parent_id, bus_arrival, bus_departure, attendance, emergency, route_changes,
		 email_enabled, sms_enabled, push_enabled)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (parent_id) DO UPDATE SET
			bus_arrival = EXCLUDED.bus_arrival,
			bus_departure = EXCLUDED.bus_departure,
			attendance = EXCLUDED.attendance,
			emergency = EXCLUDED.emergency,
			route_changes = EXCLUDED.route_changes,
			email_enabled = EXCLUDED.email_enabled,
			sms_enabled = EXCLUDED.sms_enabled,
			push_enabled = EXCLUDED.push_enabled
	`, parentID, settings.BusArrival, settings.BusDeparture, settings.Attendance,
	   settings.Emergency, settings.RouteChanges, settings.EmailEnabled,
	   settings.SMSEnabled, settings.PushEnabled)
	
	return err
}

func getStudentDetails(studentID string) StudentDetail {
	var detail StudentDetail
	
	// Get basic student info with route and driver
	query := `
		SELECT s.student_id, s.name, s.locations, s.route_id,
		       r.route_name, u.username as driver_name, ra.bus_id
		FROM students s
		LEFT JOIN routes r ON s.route_id = r.route_id
		LEFT JOIN route_assignments ra ON r.route_id = ra.route_id
			AND CURRENT_DATE BETWEEN ra.start_date AND ra.end_date
		LEFT JOIN users u ON ra.driver_id = u.username
		WHERE s.student_id = $1
	`
	
	var routeID, routeName, driverName, busNumber sql.NullString
	var locations string
	
	err := db.QueryRow(query, studentID).Scan(
		&detail.StudentID, &detail.Name, &locations,
		&routeID, &routeName, &driverName, &busNumber,
	)
	
	if err == nil {
		// Populate embedded Student fields
		detail.Locations = locations
		// Mock grade and address
		detail.Grade = "5th" 
		detail.Address = locations
		
		if routeID.Valid {
			detail.RouteID = routeID.String
			detail.Route = &Route{
				RouteID:   routeID.String,
				RouteName: routeName.String,
			}
		}
		if driverName.Valid {
			detail.Driver = &User{Username: driverName.String}
		}
		if busNumber.Valid {
			detail.BusNumber = busNumber.String
		}
		
		// Mock pickup/drop times - in production, these would come from route plans
		detail.PickupTime = "7:15 AM"
		detail.DropTime = "3:45 PM"
		detail.PickupStop = "Corner of Main St"
		detail.DropStop = "Corner of Main St"
	}
	
	return detail
}

func getStudentAttendance(studentID string, days int) []AttendanceRecord {
	// Mock attendance data - in production, this would query actual attendance records
	records := []AttendanceRecord{}
	
	// Generate mock data for the last 'days' days
	// In production, query the attendance table
	
	return records
}

func getStudentNotifications(studentID string, limit int) []ParentNotification {
	// Get notifications specific to this student
	// In production, join with actual notification data
	return []ParentNotification{}
}

func getUpcomingEvents(students []Student) []Event {
	// Mock upcoming events - in production, query from events table
	return []Event{}
}

func getBusTrackingForStudent(studentID string) *BusTracking {
	var tracking BusTracking
	
	query := `
		SELECT s.student_id, s.name, ra.bus_id, r.route_id, r.route_name
		FROM students s
		JOIN routes r ON s.route_id = r.route_id
		LEFT JOIN route_assignments ra ON r.route_id = ra.route_id
			AND CURRENT_DATE BETWEEN ra.start_date AND ra.end_date
		WHERE s.student_id = $1
	`
	
	var busID sql.NullString
	
	err := db.QueryRow(query, studentID).Scan(
		&tracking.StudentID, &tracking.StudentName,
		&busID, &tracking.RouteID, &tracking.RouteName,
	)
	
	if err != nil {
		return nil
	}
	
	if busID.Valid {
		tracking.BusID = busID.String
		
		// Get current location from GPS tracker
		if gpsTracker != nil {
			if loc, err := gpsTracker.GetLatestLocation(tracking.BusID); err == nil && loc != nil {
				tracking.CurrentLocation = loc
				tracking.Status = "en_route"
				
				// Calculate ETA (mock for now)
				eta := calculateStudentETA(tracking.StudentID, loc)
				tracking.EstimatedArrival = eta
			}
		}
	} else {
		tracking.Status = "not_started"
	}
	
	return &tracking
}

func markParentNotificationRead(parentID int, notificationID string) error {
	_, err := db.Exec(`
		UPDATE parent_notifications 
		SET read = true 
		WHERE parent_id = $1 AND id = $2
	`, parentID, notificationID)
	return err
}

func generateRandomString(length int) string {
	// Simple random string generator for demo
	// In production, use crypto/rand
	const charset = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[i%len(charset)]
	}
	return string(result)
}

// generateNotificationID is already defined in handlers_realtime_notifications.go