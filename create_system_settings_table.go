package main

import (
	"log"
)

// CreateSystemSettingsTable creates the system_settings table if it doesn't exist
func CreateSystemSettingsTable() error {
	query := `
	CREATE TABLE IF NOT EXISTS system_settings (
		key VARCHAR(100) PRIMARY KEY,
		value TEXT,
		description TEXT,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_by VARCHAR(50)
	);
	
	-- Insert default GPS setting if it doesn't exist
	INSERT INTO system_settings (key, value, description, updated_by)
	VALUES ('gps_enabled', 'false', 'Enable or disable GPS tracking system-wide', 'system')
	ON CONFLICT (key) DO NOTHING;
	`
	
	_, err := db.Exec(query)
	if err != nil {
		log.Printf("Error creating system_settings table: %v", err)
		return err
	}
	
	log.Println("System settings table ready")
	return nil
}

// CreateErrorLogsTable creates the error_logs table for tracking panics and errors
func CreateErrorLogsTable() error {
	query := `
	CREATE TABLE IF NOT EXISTS error_logs (
		id SERIAL PRIMARY KEY,
		timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		url VARCHAR(500),
		method VARCHAR(10),
		error TEXT,
		stack_trace TEXT,
		username VARCHAR(100),
		user_agent TEXT,
		resolved BOOLEAN DEFAULT FALSE,
		notes TEXT
	);
	`
	
	_, err := db.Exec(query)
	if err != nil {
		log.Printf("Error creating error_logs table: %v", err)
		return err
	}
	
	// Create indexes for better performance
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_error_logs_timestamp ON error_logs(timestamp DESC)",
		"CREATE INDEX IF NOT EXISTS idx_error_logs_username ON error_logs(username)",
		"CREATE INDEX IF NOT EXISTS idx_error_logs_resolved ON error_logs(resolved)",
	}
	
	for _, idx := range indexes {
		if _, err := db.Exec(idx); err != nil {
			log.Printf("Warning: Failed to create index: %v", err)
		}
	}
	
	log.Println("Error logs table ready")
	return nil
}