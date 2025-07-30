package main

import (
	"fmt"
	"log"
)

// FixRoutesDisplay ensures routes are properly initialized and displayed
func FixRoutesDisplay() error {
	log.Println("Starting routes display fix...")

	// 1. Check if routes table exists
	var tableExists bool
	err := db.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name = 'routes'
		)
	`).Scan(&tableExists)
	if err != nil {
		return fmt.Errorf("error checking routes table: %w", err)
	}

	if !tableExists {
		log.Println("Routes table doesn't exist, creating it...")
		err = createRoutesTable()
		if err != nil {
			return fmt.Errorf("error creating routes table: %w", err)
		}
	}

	// 2. Check route count
	var routeCount int
	err = db.QueryRow("SELECT COUNT(*) FROM routes").Scan(&routeCount)
	if err != nil {
		return fmt.Errorf("error counting routes: %w", err)
	}

	log.Printf("Current route count: %d", routeCount)

	// 3. If no routes exist, create sample routes
	if routeCount == 0 {
		log.Println("No routes found, creating sample routes...")
		err = createSampleRoutes()
		if err != nil {
			return fmt.Errorf("error creating sample routes: %w", err)
		}
	}

	// 4. Fix any NULL or invalid data in routes
	log.Println("Fixing NULL values in routes...")
	_, err = db.Exec(`
		UPDATE routes 
		SET description = '' 
		WHERE description IS NULL
	`)
	if err != nil {
		log.Printf("Warning: Could not fix NULL descriptions: %v", err)
	}

	_, err = db.Exec(`
		UPDATE routes 
		SET positions = '[]'::jsonb 
		WHERE positions IS NULL
	`)
	if err != nil {
		log.Printf("Warning: Could not fix NULL positions: %v", err)
	}

	// 5. Clear the cache to force reload
	if dataCache != nil {
		dataCache.clearRoutes()
		log.Println("Routes cache cleared")
	}

	// 6. Test loading routes
	routes, err := loadRoutesFromDB()
	if err != nil {
		return fmt.Errorf("error loading routes after fix: %w", err)
	}

	log.Printf("Successfully loaded %d routes after fix", len(routes))
	for i, route := range routes {
		if i < 5 { // Show first 5 routes
			log.Printf("  - %s: %s", route.RouteID, route.RouteName)
		}
	}

	return nil
}

func createRoutesTable() error {
	query := `
		CREATE TABLE IF NOT EXISTS routes (
			route_id VARCHAR(50) PRIMARY KEY,
			route_name VARCHAR(100) NOT NULL,
			description TEXT DEFAULT '',
			positions JSONB DEFAULT '[]'::jsonb,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`
	_, err := db.Exec(query)
	return err
}

func createSampleRoutes() error {
	sampleRoutes := []struct {
		ID          string
		Name        string
		Description string
	}{
		{"ROUTE-001", "North Elementary", "Morning and afternoon route for North Elementary School"},
		{"ROUTE-002", "South Middle School", "Covers South Middle School area"},
		{"ROUTE-003", "East High School", "East High School morning route"},
		{"ROUTE-004", "West Elementary", "West Elementary and surrounding neighborhoods"},
		{"ROUTE-005", "Central District", "Central district schools combined route"},
		{"ROUTE-006", "Rural North", "Rural areas north of town"},
		{"ROUTE-007", "Rural South", "Rural areas south of town"},
		{"ROUTE-008", "Downtown Schools", "Downtown area schools"},
		{"ROUTE-009", "Special Ed Route", "Special education dedicated route"},
		{"ROUTE-010", "Activity Bus", "After-school activities and sports"},
	}

	for _, route := range sampleRoutes {
		_, err := db.Exec(`
			INSERT INTO routes (route_id, route_name, description, created_at)
			VALUES ($1, $2, $3, CURRENT_TIMESTAMP)
			ON CONFLICT (route_id) DO UPDATE
			SET route_name = EXCLUDED.route_name,
			    description = EXCLUDED.description
		`, route.ID, route.Name, route.Description)
		if err != nil {
			log.Printf("Error inserting route %s: %v", route.ID, err)
			// Continue with other routes
		}
	}

	log.Printf("Created %d sample routes", len(sampleRoutes))
	return nil
}

// FixRouteAssignments ensures route assignments are valid
func FixRouteAssignments() error {
	log.Println("Fixing route assignments...")

	// 1. Remove assignments with invalid route IDs
	result, err := db.Exec(`
		DELETE FROM route_assignments ra
		WHERE NOT EXISTS (
			SELECT 1 FROM routes r 
			WHERE r.route_id = ra.route_id
		)
	`)
	if err != nil {
		log.Printf("Warning: Could not clean invalid assignments: %v", err)
	} else {
		affected, _ := result.RowsAffected()
		if affected > 0 {
			log.Printf("Removed %d invalid route assignments", affected)
		}
	}

	// 2. Remove assignments with invalid bus IDs
	result, err = db.Exec(`
		DELETE FROM route_assignments ra
		WHERE NOT EXISTS (
			SELECT 1 FROM buses b 
			WHERE b.bus_id = ra.bus_id
		)
	`)
	if err != nil {
		log.Printf("Warning: Could not clean invalid bus assignments: %v", err)
	} else {
		affected, _ := result.RowsAffected()
		if affected > 0 {
			log.Printf("Removed %d assignments with invalid buses", affected)
		}
	}

	// 3. Remove assignments with invalid drivers
	result, err = db.Exec(`
		DELETE FROM route_assignments ra
		WHERE NOT EXISTS (
			SELECT 1 FROM users u 
			WHERE u.username = ra.driver 
			AND u.role = 'driver'
		)
	`)
	if err != nil {
		log.Printf("Warning: Could not clean invalid driver assignments: %v", err)
	} else {
		affected, _ := result.RowsAffected()
		if affected > 0 {
			log.Printf("Removed %d assignments with invalid drivers", affected)
		}
	}

	return nil
}

// CreateSampleAssignments creates sample route assignments if none exist
func CreateSampleAssignments() error {
	var assignmentCount int
	err := db.QueryRow("SELECT COUNT(*) FROM route_assignments").Scan(&assignmentCount)
	if err != nil {
		return err
	}

	if assignmentCount > 0 {
		log.Printf("Found %d existing assignments, skipping sample creation", assignmentCount)
		return nil
	}

	log.Println("Creating sample route assignments...")

	// Get available drivers, buses, and routes
	var drivers []string
	err = db.Select(&drivers, "SELECT username FROM users WHERE role = 'driver' AND status = 'active' LIMIT 5")
	if err != nil || len(drivers) == 0 {
		log.Println("No active drivers found for sample assignments")
		return nil
	}

	var buses []string
	err = db.Select(&buses, "SELECT bus_id FROM buses WHERE status = 'active' LIMIT 5")
	if err != nil || len(buses) == 0 {
		log.Println("No active buses found for sample assignments")
		return nil
	}

	var routes []string
	err = db.Select(&routes, "SELECT route_id FROM routes LIMIT 5")
	if err != nil || len(routes) == 0 {
		log.Println("No routes found for sample assignments")
		return nil
	}

	// Create assignments
	for i := 0; i < len(drivers) && i < len(buses) && i < len(routes); i++ {
		_, err = db.Exec(`
			INSERT INTO route_assignments (driver, bus_id, route_id, assigned_date, created_at)
			VALUES ($1, $2, $3, CURRENT_DATE, CURRENT_TIMESTAMP)
			ON CONFLICT ON CONSTRAINT route_assignments_unique_assignment DO NOTHING
		`, drivers[i], buses[i], routes[i])
		if err != nil {
			log.Printf("Error creating assignment: %v", err)
		}
	}

	log.Println("Sample assignments created")
	return nil
}