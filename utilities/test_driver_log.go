package main

import (
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
	"github.com/jmoiron/sqlx"
)

func main() {
	// Connect to database
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgresql://postgres:jTKOWGEzlprRGbkPBCgGxsgnwyLeGoDL@shortline.proxy.rlwy.net:40148/railway"
	}

	db, err := sqlx.Connect("postgres", dbURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Check if bus exists
	fmt.Println("Checking bus ID 7...")
	var busCount int
	err = db.Get(&busCount, "SELECT COUNT(*) FROM buses WHERE id = $1", 7)
	if err != nil {
		log.Printf("Error checking bus: %v", err)
	} else {
		fmt.Printf("Bus ID 7 exists: %v\n", busCount > 0)
	}

	// Check available buses
	var buses []struct {
		ID     int    `db:"id"`
		Number string `db:"bus_number"`
	}
	err = db.Select(&buses, "SELECT id, bus_number FROM buses LIMIT 5")
	if err != nil {
		log.Printf("Error getting buses: %v", err)
	} else {
		fmt.Println("\nAvailable buses:")
		for _, bus := range buses {
			fmt.Printf("  ID: %d, Number: %s\n", bus.ID, bus.Number)
		}
	}

	// Check route
	fmt.Println("\nChecking route ROUTE-1753845257...")
	var routeCount int
	err = db.Get(&routeCount, "SELECT COUNT(*) FROM routes WHERE route_id = $1", "ROUTE-1753845257")
	if err != nil {
		log.Printf("Error checking route: %v", err)
	} else {
		fmt.Printf("Route exists: %v\n", routeCount > 0)
	}

	// Check available routes
	var routes []struct {
		ID   string `db:"route_id"`
		Name string `db:"route_name"`
	}
	err = db.Select(&routes, "SELECT route_id, route_name FROM routes LIMIT 5")
	if err != nil {
		log.Printf("Error getting routes: %v", err)
	} else {
		fmt.Println("\nAvailable routes:")
		for _, route := range routes {
			fmt.Printf("  ID: %s, Name: %s\n", route.ID, route.Name)
		}
	}

	// Try a simpler insert directly
	fmt.Println("\nTrying direct insert into driver_logs...")
	_, err = db.Exec(`
		INSERT INTO driver_logs (driver, bus_id, route_id, date, period, departure_time, arrival_time, begin_mileage, end_mileage, attendance)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, "test", "1", "ROUTE-001", "2025-08-13", "morning", "07:00", "08:30", 100, 150, "[]")
	
	if err != nil {
		fmt.Printf("Direct insert failed: %v\n", err)
	} else {
		fmt.Println("Direct insert successful!")
	}
}