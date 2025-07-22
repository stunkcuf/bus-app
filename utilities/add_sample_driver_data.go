package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgresql://postgres:jTKOWGEzlprRGbkPBCgGxsgnwyLeGoDL@shortline.proxy.rlwy.net:40148/railway"
		fmt.Println("Using hardcoded database URL for testing")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer db.Close()

	fmt.Println("🚌 Adding sample data for driver dashboard testing...")

	// 1. First check if we have any drivers
	var driverCount int
	err = db.QueryRow("SELECT COUNT(*) FROM users WHERE role = 'driver'").Scan(&driverCount)
	if err != nil {
		log.Printf("Error checking drivers: %v", err)
		return
	}
	fmt.Printf("Current drivers: %d\n", driverCount)

	// 2. Add a sample driver if none exist
	if driverCount == 0 {
		_, err = db.Exec(`
			INSERT INTO users (username, password, role, status, created_at) 
			VALUES ('driver1', '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewdBPxRW0.2h1CrO', 'driver', 'active', NOW())
			ON CONFLICT (username) DO NOTHING
		`)
		if err != nil {
			log.Printf("Error adding sample driver: %v", err)
		} else {
			fmt.Println("✅ Added sample driver: driver1 (password: password)")
		}
	}

	// 3. Add sample buses if none exist
	var busCount int
	db.QueryRow("SELECT COUNT(*) FROM buses").Scan(&busCount)
	if busCount == 0 {
		buses := []string{
			"INSERT INTO buses (bus_id, model, capacity, status, created_at) VALUES ('101', 'Blue Bird Vision', 72, 'active', NOW())",
			"INSERT INTO buses (bus_id, model, capacity, status, created_at) VALUES ('102', 'Thomas C2', 48, 'active', NOW())",
			"INSERT INTO buses (bus_id, model, capacity, status, created_at) VALUES ('103', 'IC Bus CE Series', 84, 'active', NOW())",
		}
		
		for _, busSQL := range buses {
			_, err = db.Exec(busSQL + " ON CONFLICT (bus_id) DO NOTHING")
			if err != nil {
				log.Printf("Error adding bus: %v", err)
			}
		}
		fmt.Println("✅ Added 3 sample buses")
	}

	// 4. Add sample routes if none exist
	var routeCount int
	db.QueryRow("SELECT COUNT(*) FROM routes").Scan(&routeCount)
	if routeCount == 0 {
		routes := []string{
			"INSERT INTO routes (route_id, route_name, description) VALUES ('R001', 'Elementary North', 'North side elementary schools')",
			"INSERT INTO routes (route_id, route_name, description) VALUES ('R002', 'Elementary South', 'South side elementary schools')",
			"INSERT INTO routes (route_id, route_name, description) VALUES ('R003', 'High School Express', 'Direct route to high school')",
		}
		
		for _, routeSQL := range routes {
			_, err = db.Exec(routeSQL + " ON CONFLICT (route_id) DO NOTHING")
			if err != nil {
				log.Printf("Error adding route: %v", err)
			}
		}
		fmt.Println("✅ Added 3 sample routes")
	}

	// 5. Add sample route assignment for driver1
	var assignmentCount int
	db.QueryRow("SELECT COUNT(*) FROM route_assignments WHERE driver = 'driver1'").Scan(&assignmentCount)
	if assignmentCount == 0 {
		_, err = db.Exec(`
			INSERT INTO route_assignments (driver, bus_id, route_id, route_name, assigned_date) 
			VALUES ('driver1', '101', 'R001', 'Elementary North', CURRENT_DATE)
			ON CONFLICT DO NOTHING
		`)
		if err != nil {
			log.Printf("Error adding route assignment: %v", err)
		} else {
			fmt.Println("✅ Assigned driver1 to route R001 with bus 101")
		}
	}

	// 6. Add sample students to the route
	var studentCount int
	db.QueryRow("SELECT COUNT(*) FROM students WHERE route_id = 'R001'").Scan(&studentCount)
	if studentCount == 0 {
		students := []string{
			`INSERT INTO students (student_id, name, locations, phone_number, guardian, pickup_time, dropoff_time, position_number, route_id, driver, active) 
			 VALUES ('S001', 'Emma Johnson', '[{"type":"pickup","address":"123 Oak St","description":"Blue house"}]', '555-0101', 'Sarah Johnson', '07:30', '15:45', 1, 'R001', 'driver1', true)`,
			
			`INSERT INTO students (student_id, name, locations, phone_number, guardian, pickup_time, dropoff_time, position_number, route_id, driver, active) 
			 VALUES ('S002', 'Liam Smith', '[{"type":"pickup","address":"456 Pine Ave","description":"Red brick house"}]', '555-0102', 'Mike Smith', '07:35', '15:40', 2, 'R001', 'driver1', true)`,
			
			`INSERT INTO students (student_id, name, locations, phone_number, guardian, pickup_time, dropoff_time, position_number, route_id, driver, active) 
			 VALUES ('S003', 'Sofia Garcia', '[{"type":"pickup","address":"789 Elm Dr","description":"White fence"}]', '555-0103', 'Maria Garcia', '07:40', '15:35', 3, 'R001', 'driver1', true)`,
		}
		
		for _, studentSQL := range students {
			_, err = db.Exec(studentSQL + " ON CONFLICT (student_id) DO NOTHING")
			if err != nil {
				log.Printf("Error adding student: %v", err)
			}
		}
		fmt.Println("✅ Added 3 sample students to route R001")
	}

	// 7. Summary
	fmt.Println("\n📊 Current data summary:")
	db.QueryRow("SELECT COUNT(*) FROM users WHERE role = 'driver'").Scan(&driverCount)
	db.QueryRow("SELECT COUNT(*) FROM buses").Scan(&busCount)  
	db.QueryRow("SELECT COUNT(*) FROM routes").Scan(&routeCount)
	db.QueryRow("SELECT COUNT(*) FROM route_assignments").Scan(&assignmentCount)
	db.QueryRow("SELECT COUNT(*) FROM students WHERE active = true").Scan(&studentCount)

	fmt.Printf("  - Drivers: %d\n", driverCount)
	fmt.Printf("  - Buses: %d\n", busCount)
	fmt.Printf("  - Routes: %d\n", routeCount)
	fmt.Printf("  - Route assignments: %d\n", assignmentCount)
	fmt.Printf("  - Active students: %d\n", studentCount)

	fmt.Println("\n🎉 Sample data setup complete!")
	fmt.Println("You can now log in as:")
	fmt.Println("  Username: driver1")
	fmt.Println("  Password: password")
	fmt.Println("  Role: driver")
}