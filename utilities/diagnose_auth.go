package main

import (
	"bufio"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

// loadEnvFile loads .env file from parent directory
func loadEnvFile() error {
	envPath := filepath.Join("..", ".env")
	file, err := os.Open(envPath)
	if err != nil {
		return fmt.Errorf("could not open .env file: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			os.Setenv(key, value)
		}
	}
	
	return scanner.Err()
}

func main() {
	// Load .env file from parent directory
	if err := loadEnvFile(); err != nil {
		log.Printf("Warning: %v", err)
	}
	
	// Get database URL
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL not set. Please ensure .env file exists in parent directory or set DATABASE_URL environment variable")
	}
	
	fmt.Println("=== Authentication Diagnostics ===")
	fmt.Println()
	
	// Connect to database
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()
	
	// Test connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	fmt.Println("✅ Database connection successful")
	
	// Check if users table exists
	var tableExists bool
	err = db.QueryRow(`
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name = 'users'
		)
	`).Scan(&tableExists)
	if err != nil {
		log.Fatalf("Failed to check tables: %v", err)
	}
	fmt.Printf("✅ Users table exists: %v\n", tableExists)
	
	// List all columns in users table
	fmt.Println("\nColumns in users table:")
	rows, err := db.Query(`
		SELECT column_name, data_type, is_nullable 
		FROM information_schema.columns 
		WHERE table_name = 'users' 
		ORDER BY ordinal_position
	`)
	if err != nil {
		log.Printf("Failed to list columns: %v", err)
	} else {
		defer rows.Close()
		for rows.Next() {
			var colName, dataType, nullable string
			rows.Scan(&colName, &dataType, &nullable)
			fmt.Printf("  - %s (%s) nullable=%s\n", colName, dataType, nullable)
		}
	}
	
	// Count users
	var userCount int
	err = db.QueryRow("SELECT COUNT(*) FROM users").Scan(&userCount)
	if err != nil {
		log.Printf("Failed to count users: %v", err)
	} else {
		fmt.Printf("\n✅ Total users: %d\n", userCount)
	}
	
	// List all users
	fmt.Println("\nAll users in database:")
	userRows, err := db.Query("SELECT username, role, status FROM users ORDER BY username")
	if err != nil {
		log.Printf("Failed to list users: %v", err)
	} else {
		defer userRows.Close()
		for userRows.Next() {
			var username, role, status string
			userRows.Scan(&username, &role, &status)
			fmt.Printf("  - %s (role=%s, status=%s)\n", username, role, status)
		}
	}
	
	// Check admin specifically
	fmt.Println("\n=== Admin User Check ===")
	var adminUser struct {
		Username string
		Password string
		Role     string
		Status   string
	}
	
	err = db.QueryRow("SELECT username, password, role, status FROM users WHERE username = 'admin'").
		Scan(&adminUser.Username, &adminUser.Password, &adminUser.Role, &adminUser.Status)
	
	if err == sql.ErrNoRows {
		fmt.Println("❌ Admin user NOT FOUND")
	} else if err != nil {
		log.Printf("❌ Error querying admin: %v", err)
	} else {
		fmt.Printf("✅ Admin found: %s (role=%s, status=%s)\n", 
			adminUser.Username, adminUser.Role, adminUser.Status)
		
		// Check password format
		if len(adminUser.Password) > 0 && adminUser.Password[0] == '$' {
			fmt.Println("✅ Password is hashed (bcrypt)")
			
			// Test password
			err = bcrypt.CompareHashAndPassword([]byte(adminUser.Password), []byte("Headstart1"))
			if err == nil {
				fmt.Println("✅ Password 'Headstart1' is CORRECT")
			} else {
				fmt.Println("❌ Password 'Headstart1' is INCORRECT")
			}
		} else {
			fmt.Println("⚠️  Password is NOT hashed (plain text)")
		}
	}
	
	// Try the exact query used in authentication
	fmt.Println("\n=== Testing Authentication Query ===")
	var testUser struct {
		Username         string
		Password         string
		Role             string
		Status           string
		RegistrationDate sql.NullTime
		CreatedAt        sql.NullTime
	}
	
	err = db.QueryRow(`
		SELECT username, password, role, status, registration_date, created_at 
		FROM users 
		WHERE username = $1
	`, "admin").Scan(
		&testUser.Username,
		&testUser.Password,
		&testUser.Role,
		&testUser.Status,
		&testUser.RegistrationDate,
		&testUser.CreatedAt,
	)
	
	if err == sql.ErrNoRows {
		fmt.Println("❌ Auth query returned no rows for admin")
	} else if err != nil {
		fmt.Printf("❌ Auth query failed: %v\n", err)
	} else {
		fmt.Println("✅ Auth query successful")
		fmt.Printf("   Username: %s\n", testUser.Username)
		fmt.Printf("   Role: %s\n", testUser.Role)
		fmt.Printf("   Status: %s\n", testUser.Status)
	}
}