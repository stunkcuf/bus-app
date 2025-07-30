package main

import (
	"log"
)

// FixNotificationsTable adds missing columns to the notifications table
func FixNotificationsTable() error {
	log.Println("Fixing notifications table...")

	// Add missing columns to notifications table
	alterQueries := []string{
		`ALTER TABLE notifications ADD COLUMN IF NOT EXISTS priority VARCHAR(20) DEFAULT 'normal'`,
		`ALTER TABLE notifications ADD COLUMN IF NOT EXISTS recipients JSONB DEFAULT '[]'::jsonb`,
		`ALTER TABLE notifications ADD COLUMN IF NOT EXISTS channels JSONB DEFAULT '[]'::jsonb`,
		`ALTER TABLE notifications ADD COLUMN IF NOT EXISTS data JSONB DEFAULT '{}'::jsonb`,
		`ALTER TABLE notifications ADD COLUMN IF NOT EXISTS status VARCHAR(20) DEFAULT 'pending'`,
		`ALTER TABLE notifications ADD COLUMN IF NOT EXISTS scheduled_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP`,
		`ALTER TABLE notifications ADD COLUMN IF NOT EXISTS subject VARCHAR(255)`,
		`ALTER TABLE notifications ADD COLUMN IF NOT EXISTS message TEXT`,
		`ALTER TABLE notifications ADD COLUMN IF NOT EXISTS type VARCHAR(50)`,
	}

	for _, query := range alterQueries {
		_, err := db.Exec(query)
		if err != nil {
			log.Printf("Warning: Failed to execute: %s - Error: %v", query, err)
			// Continue with other queries even if one fails
		}
	}

	// Add missing columns to in_app_notifications table
	inAppQueries := []string{
		`ALTER TABLE in_app_notifications ADD COLUMN IF NOT EXISTS priority VARCHAR(20) DEFAULT 'normal'`,
		`ALTER TABLE in_app_notifications ADD COLUMN IF NOT EXISTS data JSONB DEFAULT '{}'::jsonb`,
	}

	for _, query := range inAppQueries {
		_, err := db.Exec(query)
		if err != nil {
			log.Printf("Warning: Failed to execute: %s - Error: %v", query, err)
			// Continue with other queries even if one fails
		}
	}

	log.Println("Notifications table fix completed!")
	return nil
}