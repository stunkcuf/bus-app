package main

import (
	"fmt"
	"github.com/joho/godotenv"
	"os"
)

// setupDatabase initializes the database connection
func setupDatabase() error {
	// Load .env file in development
	if os.Getenv("APP_ENV") != "production" {
		if err := godotenv.Load(); err != nil {
			// .env file is optional, so we don't fail if it doesn't exist
			fmt.Println("No .env file found, using system environment variables")
		}
	}

	// Get database URL from environment
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		// Fallback to individual environment variables
		host := os.Getenv("PGHOST")
		port := os.Getenv("PGPORT")
		user := os.Getenv("PGUSER")
		password := os.Getenv("PGPASSWORD")
		dbname := os.Getenv("PGDATABASE")

		if host == "" || port == "" || user == "" || password == "" || dbname == "" {
			return fmt.Errorf("database connection parameters not set")
		}

		// Construct the database URL
		dbURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=require",
			user, password, host, port, dbname)
	}

	// Initialize the database connection
	return InitDB(dbURL)
}

// closeDatabase closes the database connection
func closeDatabase() {
	if db != nil {
		db.Close()
		fmt.Println("Database connection closed")
	}
}
