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

type DataSeeder struct {
	db *sql.DB
}

func main() {
	fmt.Println("🌱 DATA SEEDER - Fleet Management System")
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
	
	seeder := &DataSeeder{db: db}
	
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
	
	// 1. Ensure test users exist
	if err := seeder.seedUsers(); err != nil {
		log.Printf("Failed to seed users: %v", err)
	}
	
	// 2. Seed additional buses if needed
	if err := seeder.seedBuses(); err != nil {
		log.Printf("Failed to seed buses: %v", err)
	}
	
	// 3. Seed vehicles
	if err := seeder.seedVehicles(); err != nil {
		log.Printf("Failed to seed vehicles: %v", err)
	}
	
	// 4. Seed routes
	if err := seeder.seedRoutes(); err != nil {
		log.Printf("Failed to seed routes: %v", err)
	}
	
	// 5. Seed students
	if err := seeder.seedStudents(); err != nil {
		log.Printf("Failed to seed students: %v", err)
	}
	
	// 6. Seed route assignments
	if err := seeder.seedRouteAssignments(); err != nil {
		log.Printf("Failed to seed route assignments: %v", err)
	}
	
	// 7. Seed maintenance records
	if err := seeder.seedMaintenanceRecords(); err != nil {
		log.Printf("Failed to seed maintenance records: %v", err)
	}
	
	// 8. Seed fuel records
	if err := seeder.seedFuelRecords(); err != nil {
		log.Printf("Failed to seed fuel records: %v", err)
	}
	
	// 9. Seed driver logs
	if err := seeder.seedDriverLogs(); err != nil {
		log.Printf("Failed to seed driver logs: %v", err)
	}
	
	// 10. Seed monthly mileage reports
	if err := seeder.seedMileageReports(); err != nil {
		log.Printf("Failed to seed mileage reports: %v", err)
	}
	
	fmt.Println("\n✅ Data seeding complete!")
	
	// Show final status
	fmt.Println("\n📊 Final Data Status:")
	seeder.checkDataStatus()
}

func (s *DataSeeder) checkDataStatus() {
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

func (s *DataSeeder) seedUsers() error {
	fmt.Println("\n👥 Seeding users...")
	
	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	
	users := []struct {
		username string
		email    string
		role     string
		fullName string
	}{
		{"testmanager", "manager@test.com", "manager", "Test Manager"},
		{"testdriver1", "driver1@test.com", "driver", "Test Driver One"},
		{"testdriver2", "driver2@test.com", "driver", "Test Driver Two"},
		{"testdriver3", "driver3@test.com", "driver", "Test Driver Three"},
	}
	
	for _, user := range users {
		// Check if user exists
		var exists bool
		err := s.db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE username = $1)", user.username).Scan(&exists)
		if err != nil {
			return err
		}
		
		if !exists {
			_, err = s.db.Exec(`
				INSERT INTO users (username, email, password_hash, role, full_name, status, created_at)
				VALUES ($1, $2, $3, $4, $5, 'active', CURRENT_TIMESTAMP)
			`, user.username, user.email, string(hashedPassword), user.role, user.fullName)
			
			if err != nil {
				log.Printf("Failed to create user %s: %v", user.username, err)
			} else {
				fmt.Printf("✅ Created user: %s\n", user.username)
			}
		}
	}
	
	return nil
}

func (s *DataSeeder) seedBuses() error {
	fmt.Println("\n🚌 Seeding buses...")
	
	// Check current bus count
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM buses").Scan(&count)
	if err != nil {
		return err
	}
	
	if count >= 20 {
		fmt.Println("Already have sufficient buses")
		return nil
	}
	
	// Add more buses
	models := []string{"CHEVROLET MIDCO", "FORD TRANSIT", "BLUEBIRD VISION", "THOMAS BUILT"}
	statuses := []string{"active", "active", "active", "maintenance"}
	
	for i := count + 1; i <= 20; i++ {
		busID := fmt.Sprintf("BUS-%03d", i)
		model := models[rand.Intn(len(models))]
		status := statuses[rand.Intn(len(statuses))]
		capacity := 30 + rand.Intn(40) // 30-70 seats
		
		_, err = s.db.Exec(`
			INSERT INTO buses (bus_id, model, status, capacity, oil_status, tire_status, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, CURRENT_TIMESTAMP)
			ON CONFLICT (bus_id) DO NOTHING
		`, busID, model, status, capacity, "good", "good")
		
		if err != nil {
			log.Printf("Failed to create bus %s: %v", busID, err)
		} else {
			fmt.Printf("✅ Created bus: %s\n", busID)
		}
	}
	
	return nil
}

func (s *DataSeeder) seedVehicles() error {
	fmt.Println("\n🚗 Seeding vehicles...")
	
	// Check current vehicle count
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM vehicles").Scan(&count)
	if err != nil {
		return err
	}
	
	if count >= 10 {
		fmt.Println("Already have sufficient vehicles")
		return nil
	}
	
	// Add service vehicles
	vehicles := []struct {
		vehicleNumber int
		vehicleType   string
		status        string
	}{
		{101, "SERVICE", "active"},
		{102, "SERVICE", "active"},
		{103, "MAINTENANCE", "active"},
		{104, "ADMIN", "active"},
		{105, "SPARE", "out_of_service"},
	}
	
	for _, v := range vehicles {
		// Check if exists
		var exists bool
		err := s.db.QueryRow("SELECT EXISTS(SELECT 1 FROM vehicles WHERE vehicle_number = $1)", v.vehicleNumber).Scan(&exists)
		if err != nil || exists {
			continue
		}
		
		_, err = s.db.Exec(`
			INSERT INTO vehicles (vehicle_number, type, status, created_at)
			VALUES ($1, $2, $3, CURRENT_TIMESTAMP)
		`, v.vehicleNumber, v.vehicleType, v.status)
		
		if err != nil {
			log.Printf("Failed to create vehicle %d: %v", v.vehicleNumber, err)
		} else {
			fmt.Printf("✅ Created vehicle: %d\n", v.vehicleNumber)
		}
	}
	
	return nil
}

func (s *DataSeeder) seedRoutes() error {
	fmt.Println("\n🗺️ Seeding routes...")
	
	routes := []struct {
		id          string
		name        string
		description string
	}{
		{"RT-NORTH-01", "North Elementary Route", "Covers north side elementary schools"},
		{"RT-SOUTH-01", "South Elementary Route", "Covers south side elementary schools"},
		{"RT-EAST-01", "East Middle School Route", "East side middle school route"},
		{"RT-WEST-01", "West High School Route", "West side high school route"},
		{"RT-CENTRAL-01", "Central District Route", "Central district all schools"},
		{"RT-SPECIAL-01", "Special Needs Route", "Special education transport"},
	}
	
	for _, route := range routes {
		var exists bool
		err := s.db.QueryRow("SELECT EXISTS(SELECT 1 FROM routes WHERE id = $1)", route.id).Scan(&exists)
		if err != nil || exists {
			continue
		}
		
		_, err = s.db.Exec(`
			INSERT INTO routes (id, name, description, created_at)
			VALUES ($1, $2, $3, CURRENT_TIMESTAMP)
		`, route.id, route.name, route.description)
		
		if err != nil {
			log.Printf("Failed to create route %s: %v", route.id, err)
		} else {
			fmt.Printf("✅ Created route: %s\n", route.id)
		}
	}
	
	return nil
}

func (s *DataSeeder) seedStudents() error {
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
	firstNames := []string{"Emma", "Liam", "Olivia", "Noah", "Ava", "Ethan", "Sophia", "Mason", "Isabella", "William"}
	lastNames := []string{"Smith", "Johnson", "Williams", "Brown", "Jones", "Garcia", "Miller", "Davis", "Rodriguez", "Martinez"}
	grades := []string{"K", "1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11", "12"}
	
	for i := 0; i < 50-count; i++ {
		firstName := firstNames[rand.Intn(len(firstNames))]
		lastName := lastNames[rand.Intn(len(lastNames))]
		grade := grades[rand.Intn(len(grades))]
		route := routes[rand.Intn(len(routes))]
		
		// Generate address
		streetNum := 100 + rand.Intn(900)
		streets := []string{"Main St", "Oak Ave", "Elm Dr", "Park Rd", "School Ln"}
		street := streets[rand.Intn(len(streets))]
		address := fmt.Sprintf("%d %s", streetNum, street)
		
		_, err = s.db.Exec(`
			INSERT INTO students (
				first_name, last_name, grade, address, 
				route_id, is_active, created_at
			) VALUES ($1, $2, $3, $4, $5, 'Y', CURRENT_TIMESTAMP)
		`, firstName, lastName, grade, address, route)
		
		if err != nil {
			log.Printf("Failed to create student: %v", err)
		}
	}
	
	fmt.Printf("✅ Created %d students\n", 50-count)
	return nil
}

func (s *DataSeeder) seedRouteAssignments() error {
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
	
	// Get buses
	var buses []string
	rows, err = s.db.Query("SELECT bus_id FROM buses WHERE status = 'active' LIMIT 10")
	if err != nil {
		return err
	}
	defer rows.Close()
	
	for rows.Next() {
		var bus string
		rows.Scan(&bus)
		buses = append(buses, bus)
	}
	
	// Get routes
	var routes []string
	rows, err = s.db.Query("SELECT id FROM routes")
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
	for i := 0; i < len(routes) && i < len(drivers) && i < len(buses); i++ {
		// Check if assignment exists
		var exists bool
		err := s.db.QueryRow(
			"SELECT EXISTS(SELECT 1 FROM route_assignments WHERE route_id = $1)", 
			routes[i],
		).Scan(&exists)
		
		if err != nil || exists {
			continue
		}
		
		_, err = s.db.Exec(`
			INSERT INTO route_assignments (
				driver_username, bus_id, route_id, 
				assigned_date, is_active, created_at
			) VALUES ($1, $2, $3, CURRENT_DATE, true, CURRENT_TIMESTAMP)
		`, drivers[i%len(drivers)], buses[i], routes[i])
		
		if err != nil {
			log.Printf("Failed to create assignment: %v", err)
		} else {
			fmt.Printf("✅ Assigned %s to route %s with bus %s\n", drivers[i%len(drivers)], routes[i], buses[i])
		}
	}
	
	return nil
}

func (s *DataSeeder) seedMaintenanceRecords() error {
	fmt.Println("\n🔧 Seeding maintenance records...")
	
	// Check current count
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM maintenance_records").Scan(&count)
	if err != nil {
		return err
	}
	
	if count >= 100 {
		fmt.Println("Already have sufficient maintenance records")
		return nil
	}
	
	// Get buses
	var buses []string
	rows, err := s.db.Query("SELECT bus_id FROM buses")
	if err != nil {
		return err
	}
	defer rows.Close()
	
	for rows.Next() {
		var bus string
		rows.Scan(&bus)
		buses = append(buses, bus)
	}
	
	// Maintenance types
	workTypes := []string{
		"Oil Change",
		"Tire Rotation",
		"Brake Inspection",
		"Engine Service",
		"Transmission Check",
		"Battery Replacement",
		"Air Filter Change",
		"Coolant Flush",
	}
	
	// Generate records
	for i := 0; i < 100-count; i++ {
		bus := buses[rand.Intn(len(buses))]
		workType := workTypes[rand.Intn(len(workTypes))]
		
		// Random date in last year
		daysAgo := rand.Intn(365)
		serviceDate := time.Now().AddDate(0, 0, -daysAgo)
		
		// Random mileage
		mileage := 50000 + rand.Intn(100000)
		
		// Random cost
		cost := float64(50 + rand.Intn(500))
		
		_, err = s.db.Exec(`
			INSERT INTO maintenance_records (
				vehicle_id, service_date, mileage, cost,
				work_description, created_at
			) VALUES ($1, $2, $3, $4, $5, CURRENT_TIMESTAMP)
		`, bus, serviceDate, mileage, cost, workType)
		
		if err != nil {
			log.Printf("Failed to create maintenance record: %v", err)
		}
	}
	
	fmt.Printf("✅ Created %d maintenance records\n", 100-count)
	return nil
}

func (s *DataSeeder) seedFuelRecords() error {
	fmt.Println("\n⛽ Seeding fuel records...")
	
	// Get buses
	var buses []string
	rows, err := s.db.Query("SELECT bus_id FROM buses WHERE status = 'active'")
	if err != nil {
		return err
	}
	defer rows.Close()
	
	for rows.Next() {
		var bus string
		rows.Scan(&bus)
		buses = append(buses, bus)
	}
	
	// Generate fuel records for last 30 days
	for _, bus := range buses {
		// 2-3 fuel-ups per week
		for d := 0; d < 30; d += 3 + rand.Intn(2) {
			date := time.Now().AddDate(0, 0, -d)
			gallons := float64(20 + rand.Intn(30))
			pricePerGallon := 3.0 + rand.Float64()
			totalCost := gallons * pricePerGallon
			odometer := 50000 + (30-d)*200 + rand.Intn(100)
			
			_, err = s.db.Exec(`
				INSERT INTO fuel_records (
					vehicle_id, fuel_date, gallons, price_per_gallon,
					total_cost, odometer_reading, location, created_at
				) VALUES ($1, $2, $3, $4, $5, $6, $7, CURRENT_TIMESTAMP)
			`, bus, date, gallons, pricePerGallon, totalCost, odometer, "Fleet Fuel Station")
			
			if err != nil {
				log.Printf("Failed to create fuel record: %v", err)
			}
		}
	}
	
	fmt.Println("✅ Created fuel records")
	return nil
}

func (s *DataSeeder) seedDriverLogs() error {
	fmt.Println("\n📝 Seeding driver logs...")
	
	// Get drivers with assignments
	rows, err := s.db.Query(`
		SELECT DISTINCT ra.driver_username, ra.bus_id, ra.route_id
		FROM route_assignments ra
		JOIN users u ON u.username = ra.driver_username
		WHERE u.role = 'driver' AND ra.is_active = true
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
	
	// Generate logs for last 30 days
	for _, a := range assignments {
		for d := 0; d < 30; d++ {
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
					driver_username, log_date, bus_id, route_id,
					start_time, end_time, start_mileage, end_mileage,
					notes, created_at
				) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, CURRENT_TIMESTAMP)
			`, a.driver, date, a.bus, a.route, 
				"07:00", "08:30", startMileage, endMileage,
				"Morning route completed")
			
			// Afternoon route
			startMileage = endMileage
			endMileage = startMileage + 25 + rand.Intn(10)
			
			_, err = s.db.Exec(`
				INSERT INTO driver_logs (
					driver_username, log_date, bus_id, route_id,
					start_time, end_time, start_mileage, end_mileage,
					notes, created_at
				) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, CURRENT_TIMESTAMP)
			`, a.driver, date, a.bus, a.route,
				"14:30", "16:00", startMileage, endMileage,
				"Afternoon route completed")
			
			if err != nil {
				log.Printf("Failed to create driver log: %v", err)
			}
		}
	}
	
	fmt.Println("✅ Created driver logs")
	return nil
}

func (s *DataSeeder) seedMileageReports() error {
	fmt.Println("\n📊 Seeding monthly mileage reports...")
	
	// Get buses
	var buses []string
	rows, err := s.db.Query("SELECT bus_id FROM buses")
	if err != nil {
		return err
	}
	defer rows.Close()
	
	for rows.Next() {
		var bus string
		rows.Scan(&bus)
		buses = append(buses, bus)
	}
	
	// Generate reports for last 6 months
	now := time.Now()
	for m := 0; m < 6; m++ {
		reportDate := now.AddDate(0, -m, 0)
		year := reportDate.Year()
		month := int(reportDate.Month())
		
		for _, bus := range buses {
			// Check if report exists
			var exists bool
			err := s.db.QueryRow(
				"SELECT EXISTS(SELECT 1 FROM monthly_mileage_reports WHERE bus_id = $1 AND year = $2 AND month = $3)",
				bus, year, month,
			).Scan(&exists)
			
			if err != nil || exists {
				continue
			}
			
			// Random mileage data
			startMileage := 100000 + m*2000 + rand.Intn(500)
			endMileage := startMileage + 1500 + rand.Intn(1000)
			totalMiles := endMileage - startMileage
			
			_, err = s.db.Exec(`
				INSERT INTO monthly_mileage_reports (
					bus_id, year, month, total_miles,
					start_mileage, end_mileage, days_operated,
					avg_daily_miles, created_at
				) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, CURRENT_TIMESTAMP)
			`, bus, year, month, totalMiles, startMileage, endMileage,
				20+rand.Intn(5), totalMiles/22)
			
			if err != nil {
				log.Printf("Failed to create mileage report: %v", err)
			}
		}
	}
	
	fmt.Println("✅ Created monthly mileage reports")
	return nil
}