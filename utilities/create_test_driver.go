package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"

	_ "github.com/lib/pq"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	fmt.Println("🚗 CREATE TEST DRIVER")
	fmt.Println("=" + strings.Repeat("=", 60))

	// Load environment
	godotenv.Load("../.env")

	// Connect to database
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL not set")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Test driver credentials
	username := "testdriver123"
	password := "password123"
	role := "driver"
	status := "active"

	// Check if user already exists
	var exists bool
	err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE username = $1)", username).Scan(&exists)
	if err != nil {
		log.Fatal("Failed to check user existence:", err)
	}

	if exists {
		fmt.Printf("❌ User '%s' already exists\n", username)
		
		// Update the password instead
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			log.Fatal("Failed to hash password:", err)
		}

		_, err = db.Exec(`
			UPDATE users 
			SET password = $1, status = $2, role = $3
			WHERE username = $4
		`, string(hashedPassword), status, role, username)

		if err != nil {
			log.Fatal("Failed to update user:", err)
		}

		fmt.Printf("✅ Updated password for user: %s\n", username)
	} else {
		// Hash the password
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			log.Fatal("Failed to hash password:", err)
		}

		// Create the user
		_, err = db.Exec(`
			INSERT INTO users (username, password, role, status)
			VALUES ($1, $2, $3, $4)
		`, username, string(hashedPassword), role, status)

		if err != nil {
			log.Fatal("Failed to create user:", err)
		}

		fmt.Printf("✅ Created test driver: %s\n", username)
	}

	fmt.Println("\n📋 Driver Details:")
	fmt.Printf("  Username: %s\n", username)
	fmt.Printf("  Password: %s\n", password)
	fmt.Printf("  Role: %s\n", role)
	fmt.Printf("  Status: %s\n", status)

	// Create a route assignment for the test driver
	fmt.Println("\n🚌 Creating route assignment...")
	
	// Find an available bus
	var busID string
	err = db.QueryRow(`
		SELECT bus_id FROM buses 
		WHERE status = 'active' 
		AND bus_id NOT IN (SELECT bus_id FROM route_assignments WHERE bus_id IS NOT NULL)
		LIMIT 1
	`).Scan(&busID)
	
	if err != nil {
		fmt.Println("❌ No available buses for assignment")
	} else {
		// Find an available route
		var routeID string
		err = db.QueryRow(`
			SELECT id FROM routes 
			WHERE id NOT IN (SELECT route_id FROM route_assignments WHERE route_id IS NOT NULL)
			LIMIT 1
		`).Scan(&routeID)
		
		if err != nil {
			fmt.Println("❌ No available routes for assignment")
		} else {
			// Create assignment
			_, err = db.Exec(`
				INSERT INTO route_assignments (driver, bus_id, route_id, assigned_date)
				VALUES ($1, $2, $3, CURRENT_DATE)
				ON CONFLICT DO NOTHING
			`, username, busID, routeID)
			
			if err != nil {
				fmt.Printf("❌ Failed to create assignment: %v\n", err)
			} else {
				fmt.Printf("✅ Assigned bus %s to route %s for driver %s\n", busID, routeID, username)
			}
		}
	}

	// Also create a manager for testing
	fmt.Println("\n👔 Creating test manager...")
	managerUsername := "testmanager123"
	managerPassword := "password123"
	
	// Check if manager exists
	err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE username = $1)", managerUsername).Scan(&exists)
	if err != nil {
		log.Fatal("Failed to check manager existence:", err)
	}

	if exists {
		// Update the password
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(managerPassword), bcrypt.DefaultCost)
		if err != nil {
			log.Fatal("Failed to hash password:", err)
		}

		_, err = db.Exec(`
			UPDATE users 
			SET password = $1, status = 'active', role = 'manager'
			WHERE username = $2
		`, string(hashedPassword), managerUsername)

		if err != nil {
			log.Fatal("Failed to update manager:", err)
		}

		fmt.Printf("✅ Updated password for manager: %s\n", managerUsername)
	} else {
		// Hash the password
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(managerPassword), bcrypt.DefaultCost)
		if err != nil {
			log.Fatal("Failed to hash password:", err)
		}

		// Create the manager
		_, err = db.Exec(`
			INSERT INTO users (username, password, role, status)
			VALUES ($1, $2, 'manager', 'active')
		`, managerUsername, string(hashedPassword))

		if err != nil {
			log.Fatal("Failed to create manager:", err)
		}

		fmt.Printf("✅ Created test manager: %s\n", managerUsername)
	}

	fmt.Println("\n📋 Manager Details:")
	fmt.Printf("  Username: %s\n", managerUsername)
	fmt.Printf("  Password: %s\n", managerPassword)
	fmt.Printf("  Role: manager\n")
	fmt.Printf("  Status: active\n")

	fmt.Println("\n✅ Test accounts ready for use!")
}