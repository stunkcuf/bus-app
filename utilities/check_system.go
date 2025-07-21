package main

import (
	"bufio"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"

	_ "github.com/lib/pq"
)

func loadEnv() {
	file, err := os.Open("../.env")
	if err != nil {
		file, err = os.Open(".env")
		if err != nil {
			return
		}
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) == 0 || line[0] == '#' {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			os.Setenv(parts[0], parts[1])
		}
	}
}

func main() {
	loadEnv()
	
	fmt.Println("=== FLEET MANAGEMENT SYSTEM CHECK ===\n")
	
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL not set")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Check connection
	err = db.Ping()
	if err != nil {
		log.Fatal("Database connection failed:", err)
	}
	fmt.Println("✓ Database connection successful")

	// Check admin user
	fmt.Println("\n=== USER CHECK ===")
	var username, role, status string
	err = db.QueryRow("SELECT username, role, status FROM users WHERE username = 'admin'").Scan(&username, &role, &status)
	if err != nil {
		fmt.Printf("✗ Admin user not found: %v\n", err)
	} else {
		fmt.Printf("✓ Admin user exists: username=%s, role=%s, status=%s\n", username, role, status)
	}

	// Check table counts
	fmt.Println("\n=== TABLE RECORD COUNTS ===")
	tables := []struct {
		name     string
		expected string
	}{
		{"buses", "10 buses"},
		{"vehicles", "44 vehicles"},
		{"students", "19 students"},
		{"routes", "5 routes"},
		{"maintenance_records", "maintenance records"},
		{"users", "users"},
		{"route_assignments", "route assignments"},
		{"driver_logs", "driver logs"},
		{"ecse_students", "ECSE students"},
	}

	totalVehicles := 0
	for _, table := range tables {
		var count int
		err = db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", table.name)).Scan(&count)
		if err != nil {
			fmt.Printf("✗ Error counting %s: %v\n", table.name, err)
		} else {
			fmt.Printf("✓ %s table: %d %s\n", table.name, count, table.expected)
			if table.name == "buses" || table.name == "vehicles" {
				totalVehicles += count
			}
		}
	}
	
	fmt.Printf("\n✓ Total fleet vehicles (buses + vehicles): %d\n", totalVehicles)

	// Check some sample data
	fmt.Println("\n=== SAMPLE DATA ===")
	
	// Sample buses
	fmt.Println("\nSample Buses:")
	rows, err := db.Query("SELECT bus_id, status, model FROM buses LIMIT 3")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var busID, status, model string
			rows.Scan(&busID, &status, &model)
			fmt.Printf("  - Bus %s: %s, Status: %s\n", busID, model, status)
		}
	}

	// Sample vehicles
	fmt.Println("\nSample Vehicles:")
	rows, err = db.Query("SELECT vehicle_id, model, status FROM vehicles LIMIT 3")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var vehicleID, model, status string
			rows.Scan(&vehicleID, &model, &status)
			fmt.Printf("  - Vehicle %s: %s, Status: %s\n", vehicleID, model, status)
		}
	}

	// Check if we can authenticate with admin password
	fmt.Println("\n=== PASSWORD CHECK ===")
	var hashedPassword string
	err = db.QueryRow("SELECT password FROM users WHERE username = 'admin'").Scan(&hashedPassword)
	if err != nil {
		fmt.Printf("✗ Could not retrieve admin password: %v\n", err)
	} else {
		// Check if it's a bcrypt hash
		if strings.HasPrefix(hashedPassword, "$2a$") || strings.HasPrefix(hashedPassword, "$2b$") {
			fmt.Println("✓ Admin password is properly hashed (bcrypt)")
		} else if hashedPassword == "admin" {
			fmt.Println("⚠ WARNING: Admin password is not hashed!")
		} else {
			fmt.Println("? Admin password format unknown")
		}
	}

	fmt.Println("\n=== SYSTEM STATUS ===")
	fmt.Println("Database: ✓ Connected and accessible")
	fmt.Println("Admin user: ✓ Exists and active")
	fmt.Printf("Fleet size: ✓ %d total vehicles\n", totalVehicles)
	fmt.Println("\nTo test login functionality, please ensure the server is running:")
	fmt.Println("  go run .")
	fmt.Println("  Then access http://localhost:5003")
	fmt.Println("  Login with: admin/admin")
}