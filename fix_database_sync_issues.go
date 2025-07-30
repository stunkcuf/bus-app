package main

import (
	"fmt"
	"log"
	"strings"
)

// FixDatabaseSyncIssues adds missing columns and fixes struct mapping issues
func FixDatabaseSyncIssues() error {
	log.Println("Starting database sync fixes...")

	// Define migrations to fix missing columns
	migrations := []struct {
		name        string
		query       string
		description string
	}{
		// Fix 1: Add subject column to in_app_notifications table
		{
			name: "add_subject_to_notifications",
			query: `
				DO $$ 
				BEGIN 
					IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
						WHERE table_name = 'in_app_notifications' AND column_name = 'subject') THEN
						ALTER TABLE in_app_notifications ADD COLUMN subject VARCHAR(255) NOT NULL DEFAULT 'Notification';
						UPDATE in_app_notifications SET subject = 
							CASE 
								WHEN type = 'maintenance_due' THEN 'Maintenance Due'
								WHEN type = 'route_change' THEN 'Route Change'
								WHEN type = 'emergency' THEN 'Emergency Alert'
								WHEN type = 'vehicle_issue' THEN 'Vehicle Issue'
								ELSE 'System Notification'
							END
						WHERE subject = 'Notification';
					END IF;
				END $$;`,
			description: "Adding subject column to in_app_notifications table",
		},
		// Fix 1b: Also add subject to notifications table
		{
			name: "add_subject_to_notifications_main",
			query: `
				DO $$ 
				BEGIN 
					IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
						WHERE table_name = 'notifications' AND column_name = 'subject') THEN
						ALTER TABLE notifications ADD COLUMN subject VARCHAR(255) NOT NULL DEFAULT 'Notification';
					END IF;
				END $$;`,
			description: "Adding subject column to notifications table",
		},
		// Fix 1c: Add priority column to notifications table
		{
			name: "add_priority_to_notifications",
			query: `
				DO $$ 
				BEGIN 
					IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
						WHERE table_name = 'notifications' AND column_name = 'priority') THEN
						ALTER TABLE notifications ADD COLUMN priority VARCHAR(20) NOT NULL DEFAULT 'medium';
					END IF;
				END $$;`,
			description: "Adding priority column to notifications table",
		},
		// Fix 1d: Add data column to notifications table
		{
			name: "add_data_to_notifications",
			query: `
				DO $$ 
				BEGIN 
					IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
						WHERE table_name = 'notifications' AND column_name = 'data') THEN
						ALTER TABLE notifications ADD COLUMN data JSONB;
					END IF;
				END $$;`,
			description: "Adding data column to notifications table",
		},
		// Fix 2: Add mileage column to vehicle health checks (if such table exists)
		// First check if there's a vehicle_health_checks table
		{
			name: "create_vehicle_health_checks_if_missing",
			query: `
				CREATE TABLE IF NOT EXISTS vehicle_health_checks (
					id SERIAL PRIMARY KEY,
					vehicle_id VARCHAR(50) NOT NULL,
					check_date DATE NOT NULL,
					mileage INTEGER NOT NULL,
					oil_check BOOLEAN DEFAULT FALSE,
					tire_check BOOLEAN DEFAULT FALSE,
					brake_check BOOLEAN DEFAULT FALSE,
					fluid_check BOOLEAN DEFAULT FALSE,
					notes TEXT,
					checked_by VARCHAR(50),
					created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
				);`,
			description: "Creating vehicle_health_checks table if missing",
		},
		// Fix 3: Add goals column to ecse_services table
		{
			name: "add_goals_to_ecse_services",
			query: `
				DO $$ 
				BEGIN 
					IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
						WHERE table_name = 'ecse_services' AND column_name = 'goals') THEN
						ALTER TABLE ecse_services ADD COLUMN goals TEXT;
					END IF;
				END $$;`,
			description: "Adding goals column to ecse_services table",
		},
		// Fix 4: Add id column to users table (auto-incrementing)
		{
			name: "add_id_to_users",
			query: `
				DO $$ 
				BEGIN 
					IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
						WHERE table_name = 'users' AND column_name = 'id') THEN
						ALTER TABLE users ADD COLUMN id SERIAL;
						-- Create unique index on id
						CREATE UNIQUE INDEX IF NOT EXISTS idx_users_id ON users(id);
					END IF;
				END $$;`,
			description: "Adding id column to users table",
		},
		// Fix 5: Update foreign key references to use username instead of id where needed
		{
			name: "fix_conversation_participants_fk",
			query: `
				DO $$ 
				BEGIN 
					-- Drop the existing foreign key constraint if it references users(id)
					IF EXISTS (
						SELECT 1 FROM information_schema.table_constraints 
						WHERE constraint_name = 'conversation_participants_user_id_fkey'
						AND table_name = 'conversation_participants'
					) THEN
						ALTER TABLE conversation_participants DROP CONSTRAINT conversation_participants_user_id_fkey;
					END IF;
					
					-- Change user_id column to VARCHAR to reference username
					IF EXISTS (
						SELECT 1 FROM information_schema.columns 
						WHERE table_name = 'conversation_participants' 
						AND column_name = 'user_id' 
						AND data_type = 'integer'
					) THEN
						-- Create temporary column
						ALTER TABLE conversation_participants ADD COLUMN user_username VARCHAR(50);
						
						-- Copy data converting id to username
						UPDATE conversation_participants cp
						SET user_username = u.username
						FROM users u
						WHERE cp.user_id = u.id;
						
						-- Drop old column and rename new one
						ALTER TABLE conversation_participants DROP COLUMN user_id;
						ALTER TABLE conversation_participants RENAME COLUMN user_username TO user_id;
						
						-- Add foreign key to username
						ALTER TABLE conversation_participants 
						ADD CONSTRAINT conversation_participants_user_id_fkey 
						FOREIGN KEY (user_id) REFERENCES users(username) ON DELETE CASCADE;
					END IF;
				END $$;`,
			description: "Fixing conversation_participants foreign key reference",
		},
		// Fix 6: Similar fix for messages table
		{
			name: "fix_messages_fk",
			query: `
				DO $$ 
				BEGIN 
					-- Fix sender_id column
					IF EXISTS (
						SELECT 1 FROM information_schema.columns 
						WHERE table_name = 'messages' 
						AND column_name = 'sender_id' 
						AND data_type = 'integer'
					) THEN
						-- Create temporary column
						ALTER TABLE messages ADD COLUMN sender_username VARCHAR(50);
						
						-- Copy data converting id to username
						UPDATE messages m
						SET sender_username = u.username
						FROM users u
						WHERE m.sender_id = u.id;
						
						-- Drop old column and rename new one
						ALTER TABLE messages DROP COLUMN sender_id;
						ALTER TABLE messages RENAME COLUMN sender_username TO sender_id;
					END IF;
					
					-- Fix recipient_id column
					IF EXISTS (
						SELECT 1 FROM information_schema.columns 
						WHERE table_name = 'messages' 
						AND column_name = 'recipient_id' 
						AND data_type = 'integer'
					) THEN
						-- Create temporary column
						ALTER TABLE messages ADD COLUMN recipient_username VARCHAR(50);
						
						-- Copy data converting id to username
						UPDATE messages m
						SET recipient_username = u.username
						FROM users u
						WHERE m.recipient_id = u.id;
						
						-- Drop old column and rename new one
						ALTER TABLE messages DROP COLUMN recipient_id;
						ALTER TABLE messages RENAME COLUMN recipient_username TO recipient_id;
					END IF;
					
					-- Add foreign keys if they don't exist
					IF NOT EXISTS (
						SELECT 1 FROM information_schema.table_constraints 
						WHERE constraint_name = 'messages_sender_id_fkey'
					) THEN
						ALTER TABLE messages 
						ADD CONSTRAINT messages_sender_id_fkey 
						FOREIGN KEY (sender_id) REFERENCES users(username) ON DELETE SET NULL;
					END IF;
					
					IF NOT EXISTS (
						SELECT 1 FROM information_schema.table_constraints 
						WHERE constraint_name = 'messages_recipient_id_fkey'
					) THEN
						ALTER TABLE messages 
						ADD CONSTRAINT messages_recipient_id_fkey 
						FOREIGN KEY (recipient_id) REFERENCES users(username) ON DELETE SET NULL;
					END IF;
				END $$;`,
			description: "Fixing messages table foreign key references",
		},
		// Fix 7: Similar fix for emergency_alerts table
		{
			name: "fix_emergency_alerts_fk",
			query: `
				DO $$ 
				BEGIN 
					IF EXISTS (
						SELECT 1 FROM information_schema.columns 
						WHERE table_name = 'emergency_alerts' 
						AND column_name = 'reporter_id' 
						AND data_type = 'integer'
					) THEN
						-- Create temporary column
						ALTER TABLE emergency_alerts ADD COLUMN reporter_username VARCHAR(50);
						
						-- Copy data converting id to username
						UPDATE emergency_alerts ea
						SET reporter_username = u.username
						FROM users u
						WHERE ea.reporter_id = u.id;
						
						-- Drop old column and rename new one
						ALTER TABLE emergency_alerts DROP COLUMN reporter_id;
						ALTER TABLE emergency_alerts RENAME COLUMN reporter_username TO reporter_id;
						
						-- Add foreign key
						ALTER TABLE emergency_alerts 
						ADD CONSTRAINT emergency_alerts_reporter_id_fkey 
						FOREIGN KEY (reporter_id) REFERENCES users(username) ON DELETE SET NULL;
					END IF;
				END $$;`,
			description: "Fixing emergency_alerts table foreign key reference",
		},
		// Fix 8: Similar fix for emergency_responders table
		{
			name: "fix_emergency_responders_fk",
			query: `
				DO $$ 
				BEGIN 
					IF EXISTS (
						SELECT 1 FROM information_schema.columns 
						WHERE table_name = 'emergency_responders' 
						AND column_name = 'user_id' 
						AND data_type = 'integer'
					) THEN
						-- Create temporary column
						ALTER TABLE emergency_responders ADD COLUMN user_username VARCHAR(50);
						
						-- Copy data converting id to username
						UPDATE emergency_responders er
						SET user_username = u.username
						FROM users u
						WHERE er.user_id = u.id;
						
						-- Drop old column and rename new one
						ALTER TABLE emergency_responders DROP COLUMN user_id;
						ALTER TABLE emergency_responders RENAME COLUMN user_username TO user_id;
						
						-- Add foreign key
						ALTER TABLE emergency_responders 
						ADD CONSTRAINT emergency_responders_user_id_fkey 
						FOREIGN KEY (user_id) REFERENCES users(username) ON DELETE CASCADE;
					END IF;
				END $$;`,
			description: "Fixing emergency_responders table foreign key reference",
		},
		// Fix 9: Add indexes for performance
		{
			name: "add_performance_indexes",
			query: `
				CREATE INDEX IF NOT EXISTS idx_vehicle_health_checks_vehicle_date 
				ON vehicle_health_checks(vehicle_id, check_date DESC);
				
				CREATE INDEX IF NOT EXISTS idx_ecse_services_student 
				ON ecse_services(student_id);
				
				CREATE INDEX IF NOT EXISTS idx_users_username_lower 
				ON users(LOWER(username));`,
			description: "Adding performance indexes",
		},
	}

	// Execute each migration
	for _, migration := range migrations {
		log.Printf("Executing migration: %s - %s", migration.name, migration.description)
		
		if _, err := db.Exec(migration.query); err != nil {
			// Check if it's a harmless error we can ignore
			errStr := err.Error()
			if strings.Contains(errStr, "already exists") {
				log.Printf("Migration %s: Column/constraint already exists, skipping", migration.name)
				continue
			}
			if strings.Contains(errStr, "does not exist") && strings.Contains(migration.name, "fix_") {
				log.Printf("Migration %s: Table doesn't exist, skipping fix", migration.name)
				continue
			}
			
			return fmt.Errorf("failed to execute migration %s: %w", migration.name, err)
		}
		
		log.Printf("Successfully completed migration: %s", migration.name)
	}

	log.Println("Database sync fixes completed successfully!")
	return nil
}

// CleanupInvalidData cleans up NULL and invalid data in student records
func CleanupInvalidData() error {
	log.Println("Starting data cleanup...")

	cleanupQueries := []struct {
		name  string
		query string
	}{
		{
			name: "Fix NULL birthdates in ECSE students",
			query: `
				UPDATE ecse_students 
				SET date_of_birth = '2019-01-01' 
				WHERE date_of_birth IS NULL OR date_of_birth = '';`,
		},
		{
			name: "Fix NULL IEP status",
			query: `
				UPDATE ecse_students 
				SET iep_status = 'Pending Review' 
				WHERE iep_status IS NULL OR iep_status = '';`,
		},
		{
			name: "Fix NULL parent information",
			query: `
				UPDATE ecse_students 
				SET parent_name = 'Contact Required' 
				WHERE parent_name IS NULL OR parent_name = '';`,
		},
		{
			name: "Fix NULL addresses",
			query: `
				UPDATE ecse_students 
				SET address = 'Address Update Required',
				    city = 'Update Required',
				    state = 'TX',
				    zip_code = '00000'
				WHERE (address IS NULL OR address = '') 
				   OR (city IS NULL OR city = '')
				   OR (state IS NULL OR state = '')
				   OR (zip_code IS NULL OR zip_code = '');`,
		},
		{
			name: "Set default enrollment status",
			query: `
				UPDATE ecse_students 
				SET enrollment_status = 'Active' 
				WHERE enrollment_status IS NULL OR enrollment_status = '';`,
		},
	}

	for _, cleanup := range cleanupQueries {
		log.Printf("Executing cleanup: %s", cleanup.name)
		
		result, err := db.Exec(cleanup.query)
		if err != nil {
			log.Printf("Warning: Failed to execute cleanup %s: %v", cleanup.name, err)
			continue
		}
		
		rowsAffected, _ := result.RowsAffected()
		log.Printf("Cleanup %s: Updated %d rows", cleanup.name, rowsAffected)
	}

	log.Println("Data cleanup completed!")
	return nil
}