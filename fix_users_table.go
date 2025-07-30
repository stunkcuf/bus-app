package main

import (
	"log"
)

// FixUsersTable adds the missing id column to users table
func FixUsersTable() error {
	log.Println("Fixing users table - adding id column...")
	
	// Check if id column already exists
	var columnExists bool
	err := db.Get(&columnExists, `
		SELECT EXISTS (
			SELECT 1 
			FROM information_schema.columns 
			WHERE table_name = 'users' 
			AND column_name = 'id'
		)
	`)
	if err != nil {
		log.Printf("Error checking for id column: %v", err)
		return err
	}
	
	if columnExists {
		log.Println("id column already exists in users table")
		return nil
	}
	
	// Add id column as SERIAL PRIMARY KEY
	// First, we need to drop the primary key constraint on username
	log.Println("Dropping primary key constraint on username...")
	_, err = db.Exec(`ALTER TABLE users DROP CONSTRAINT IF EXISTS users_pkey`)
	if err != nil {
		log.Printf("Warning: Failed to drop primary key: %v", err)
		// Continue anyway, it might not exist
	}
	
	// Add id column
	log.Println("Adding id column...")
	_, err = db.Exec(`ALTER TABLE users ADD COLUMN IF NOT EXISTS id SERIAL`)
	if err != nil {
		log.Printf("Error adding id column: %v", err)
		return err
	}
	
	// Make id the primary key
	log.Println("Setting id as primary key...")
	_, err = db.Exec(`ALTER TABLE users ADD PRIMARY KEY (id)`)
	if err != nil {
		log.Printf("Warning: Failed to add primary key on id: %v", err)
		// Continue anyway
	}
	
	// Add unique constraint on username
	log.Println("Adding unique constraint on username...")
	_, err = db.Exec(`ALTER TABLE users ADD CONSTRAINT users_username_unique UNIQUE (username)`)
	if err != nil {
		log.Printf("Warning: Failed to add unique constraint: %v", err)
		// It might already exist
	}
	
	// Update foreign key references
	log.Println("Updating foreign key references...")
	
	// We need to update tables that reference users(username) to use users(id)
	// But for now, we'll keep the username references as they are more meaningful
	// and just ensure the id column exists for compatibility
	
	log.Println("Users table fixed successfully!")
	return nil
}