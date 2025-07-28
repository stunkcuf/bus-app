package main

import (
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"

	_ "github.com/lib/pq"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
)

type DataSeederV2 struct {
	db *sql.DB
}

func main() {
	fmt.Println("🌱 DATA SEEDER V2 - Fleet Management System")
	fmt.Println("=" + strings.Repeat("=", 60))
	fmt.Println("Updated to match actual database schema")
	
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
	
	seeder := &DataSeederV2{db: db}
	
	// Check current data status
	fmt.Println("\n📊 Current Data Status:")
	seeder.checkDataStatus()
	
	// Ask for confirmation
	fmt.Println("\n⚠️  WARNING: This will add sample data to your database.")
	fmt.Print("Continue? (y/N): ")
	
	var response string
	fmt.Scanln(&response)
	if strings.ToLower(response) != "y" {
		fmt.Println("Cancelled.")
		return
	}
	
	// Seed data
	fmt.Println("\n🚀 Seeding data...")
	
	// 1. Seed additional drivers
	if err := seeder.seedDrivers(); err != nil {
		log.Printf("Failed to seed drivers: %v", err)
	}
	
	// 2. Seed fuel records (matching actual schema)
	if err := seeder.seedFuelRecords(); err != nil {
		log.Printf("Failed to seed fuel records: %v", err)
	}
	
	// 3. Seed students (matching actual schema)
	if err := seeder.seedStudents(); err != nil {
		log.Printf("Failed to seed students: %v", err)
	}
	
	// 4. Seed route assignments
	if err := seeder.seedRouteAssignments(); err != nil {
		log.Printf("Failed to seed route assignments: %v", err)
	}
	
	// 5. Seed driver logs
	if err := seeder.seedDriverLogs(); err != nil {
		log.Printf("Failed to seed driver logs: %v", err)
	}
	
	fmt.Println("\n✅ Data seeding complete!")
	
	// Show final status
	fmt.Println("\n📊 Final Data Status:")
	seeder.checkDataStatus()
}

func (s *DataSeederV2) checkDataStatus() {
	tables := []string{
		"users", "buses", "vehicles", "routes", "students",
		"route_assignments", "maintenance_records", "fuel_records",
		"driver_logs", "monthly_mileage_reports",
	}
	
	for _, table := range tables {
		var count int
		err := s.db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&count)
		if err != nil {
			fmt.Printf("❌ %-25s Error: %v\n", table+":", err)
		} else {
			fmt.Printf("📋 %-25s %d records\n", table+":", count)
		}
	}
}

func (s *DataSeederV2) seedDrivers() error {
	fmt.Println("\n👥 Seeding additional drivers...")
	
	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("driver123"), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	
	drivers := []string{
		"driver_north", "driver_south", "driver_east", "driver_west", "driver_central",
	}
	
	for _, username := range drivers {
		// Check if user exists
		var exists bool
		err := s.db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE username = $1)", username).Scan(&exists)
		if err != nil || exists {
			continue
		}
		
		_, err = s.db.Exec(`
			INSERT INTO users (username, password, role, status)
			VALUES ($1, $2, 'driver', 'active')
		`, username, string(hashedPassword))
		
		if err != nil {
			log.Printf("Failed to create driver %s: %v", username, err)
		} else {
			fmt.Printf("✅ Created driver: %s (password: driver123)\n", username)
		}
	}
	
	return nil
}

func (s *DataSeederV2) seedFuelRecords() error {
	fmt.Println("\n⛽ Seeding fuel records...")
	
	// Get buses
	var buses []string
	rows, err := s.db.Query("SELECT bus_id FROM buses WHERE status = 'active' LIMIT 10")
	if err != nil {
		return err
	}
	defer rows.Close()
	
	for rows.Next() {
		var bus string
		rows.Scan(&bus)
		buses = append(buses, bus)
	}
	
	if len(buses) == 0 {
		return fmt.Errorf("no active buses found")
	}
	
	// Generate fuel records for last 30 days
	locations := []string{"Fleet Fuel Station", "Shell Station", "BP Station", "Chevron Station"}
	count := 0
	
	for _, bus := range buses {
		// 2-3 fuel-ups per week
		for d := 0; d < 30; d += 3 + rand.Intn(2) {
			date := time.Now().AddDate(0, 0, -d)
			gallons := float64(20 + rand.Intn(30))
			pricePerGallon := 3.0 + rand.Float64()
			totalCost := gallons * pricePerGallon
			odometer := 50000 + (30-d)*200 + rand.Intn(100)
			location := locations[rand.Intn(len(locations))]
			
			_, err = s.db.Exec(`
				INSERT INTO fuel_records (
					vehicle_id, date, gallons, cost, price_per_gallon,
					odometer, location, created_at
				) VALUES ($1, $2, $3, $4, $5, $6, $7, CURRENT_TIMESTAMP)
			`, bus, date, gallons, totalCost, pricePerGallon, odometer, location)
			
			if err == nil {
				count++
			}
		}
	}
	
	fmt.Printf("✅ Created %d fuel records\n", count)
	return nil
}

func (s *DataSeederV2) seedStudents() error {
	fmt.Println("\n👨‍🎓 Seeding students...")
	
	// Check current student count
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM students").Scan(&count)
	if err != nil {
		return err
	}
	
	if count >= 50 {
		fmt.Println("Already have sufficient students")
		return nil
	}
	
	// Get routes
	var routes []string
	rows, err := s.db.Query("SELECT id FROM routes")
	if err != nil {
		return err
	}
	defer rows.Close()
	
	for rows.Next() {
		var route string
		rows.Scan(&route)
		routes = append(routes, route)
	}
	
	if len(routes) == 0 {
		return fmt.Errorf("no routes found")
	}
	
	// Generate students
	names := []string{
		"Emma Smith", "Liam Johnson", "Olivia Williams", "Noah Brown", "Ava Jones",
		"Ethan Garcia", "Sophia Miller", "Mason Davis", "Isabella Rodriguez", "William Martinez",
		"Mia Anderson", "James Taylor", "Charlotte Thomas", "Benjamin Hernandez", "Amelia Moore",
		"Lucas Martin", "Harper Jackson", "Henry Thompson", "Evelyn White", "Alexander Lopez",
	}
	
	locations := []string{
		`{"pickup": "123 Main St", "dropoff": "School"}`,
		`{"pickup": "456 Oak Ave", "dropoff": "School"}`,
		`{"pickup": "789 Elm Dr", "dropoff": "School"}`,
		`{"pickup": "321 Park Rd", "dropoff": "School"}`,
		`{"pickup": "654 School Ln", "dropoff": "School"}`,
	}
	
	added := 0
	for i := 0; i < min(50-count, len(names)); i++ {
		studentID := fmt.Sprintf("STU-%04d", count+i+1)
		name := names[i%len(names)]
		route := routes[rand.Intn(len(routes))]
		location := locations[rand.Intn(len(locations))]
		phone := fmt.Sprintf("555-%04d", 1000+rand.Intn(9000))
		
		_, err = s.db.Exec(`
			INSERT INTO students (
				student_id, name, locations, phone_number,
				route_id, active, created_at
			) VALUES ($1, $2, $3::jsonb, $4, $5, true, CURRENT_TIMESTAMP)
		`, studentID, name, location, phone, route)
		
		if err == nil {
			added++
		}
	}
	
	fmt.Printf("✅ Created %d students\n", added)
	return nil
}

func (s *DataSeederV2) seedRouteAssignments() error {
	fmt.Println("\n🚌 Seeding route assignments...")
	
	// Get drivers
	var drivers []string
	rows, err := s.db.Query("SELECT username FROM users WHERE role = 'driver' AND status = 'active'")
	if err != nil {
		return err
	}
	defer rows.Close()
	
	for rows.Next() {
		var driver string
		rows.Scan(&driver)
		drivers = append(drivers, driver)
	}
	
	// Get unassigned buses
	var buses []string
	rows, err = s.db.Query(`
		SELECT bus_id FROM buses 
		WHERE status = 'active' 
		AND bus_id NOT IN (SELECT bus_id FROM route_assignments WHERE bus_id IS NOT NULL)
		LIMIT 10
	`)
	if err != nil {
		return err
	}
	defer rows.Close()
	
	for rows.Next() {
		var bus string
		rows.Scan(&bus)
		buses = append(buses, bus)
	}
	
	// Get unassigned routes
	var routes []string
	rows, err = s.db.Query(`
		SELECT id FROM routes 
		WHERE id NOT IN (SELECT route_id FROM route_assignments WHERE route_id IS NOT NULL)
	`)
	if err != nil {
		return err
	}
	defer rows.Close()
	
	for rows.Next() {
		var route string
		rows.Scan(&route)
		routes = append(routes, route)
	}
	
	// Create assignments
	count := 0
	maxAssignments := min(len(drivers), len(buses), len(routes))
	
	for i := 0; i < maxAssignments; i++ {
		_, err = s.db.Exec(`
			INSERT INTO route_assignments (
				driver, bus_id, route_id, assigned_date, created_at
			) VALUES ($1, $2, $3, CURRENT_DATE, CURRENT_TIMESTAMP)
			ON CONFLICT DO NOTHING
		`, drivers[i], buses[i], routes[i])
		
		if err == nil {
			count++
			fmt.Printf("✅ Assigned %s to route %s with bus %s\n", drivers[i], routes[i], buses[i])
		}
	}
	
	return nil
}

func (s *DataSeederV2) seedDriverLogs() error {
	fmt.Println("\n📝 Seeding driver logs...")
	
	// Get route assignments
	rows, err := s.db.Query(`
		SELECT driver, bus_id, route_id
		FROM route_assignments
		WHERE driver IS NOT NULL AND bus_id IS NOT NULL AND route_id IS NOT NULL
	`)
	if err != nil {
		return err
	}
	defer rows.Close()
	
	type assignment struct {
		driver string
		bus    string
		route  string
	}
	
	var assignments []assignment
	for rows.Next() {
		var a assignment
		rows.Scan(&a.driver, &a.bus, &a.route)
		assignments = append(assignments, a)
	}
	
	if len(assignments) == 0 {
		fmt.Println("No route assignments found")
		return nil
	}
	
	// Generate logs for last 7 days
	count := 0
	for _, a := range assignments {
		for d := 0; d < 7; d++ {
			// Skip weekends
			date := time.Now().AddDate(0, 0, -d)
			if date.Weekday() == time.Saturday || date.Weekday() == time.Sunday {
				continue
			}
			
			// Morning route
			startMileage := 100000 + d*50
			endMileage := startMileage + 25 + rand.Intn(10)
			
			_, err = s.db.Exec(`
				INSERT INTO driver_logs (
					driver, log_date, bus_id, route_id,
					start_time, end_time, start_mileage, end_mileage,
					notes, created_at
				) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, CURRENT_TIMESTAMP)
			`, a.driver, date, a.bus, a.route, 
				"07:00", "08:30", startMileage, endMileage,
				"Morning route completed")
			
			if err == nil {
				count++
			}
			
			// Afternoon route
			startMileage = endMileage
			endMileage = startMileage + 25 + rand.Intn(10)
			
			_, err = s.db.Exec(`
				INSERT INTO driver_logs (
					driver, log_date, bus_id, route_id,
					start_time, end_time, start_mileage, end_mileage,
					notes, created_at
				) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, CURRENT_TIMESTAMP)
			`, a.driver, date, a.bus, a.route,
				"14:30", "16:00", startMileage, endMileage,
				"Afternoon route completed")
			
			if err == nil {
				count++
			}
		}
	}
	
	fmt.Printf("✅ Created %d driver logs\n", count)
	return nil
}

func min(nums ...int) int {
	if len(nums) == 0 {
		return 0
	}
	min := nums[0]
	for _, n := range nums[1:] {
		if n < min {
			min = n
		}
	}
	return min
}