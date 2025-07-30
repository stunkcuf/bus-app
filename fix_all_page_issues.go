package main

import (
	"fmt"
	"log"
	"strings"
	"time"
)

// FixAllPageIssues runs all fixes to ensure pages display data correctly
func FixAllPageIssues() error {
	log.Println("=== Starting comprehensive page fixes ===")

	// 1. Fix routes and route assignments
	if err := FixRoutesDisplay(); err != nil {
		log.Printf("Error fixing routes display: %v", err)
	}

	if err := FixRouteAssignments(); err != nil {
		log.Printf("Error fixing route assignments: %v", err)
	}

	// 2. Ensure sample data exists for testing
	if err := EnsureSampleData(); err != nil {
		log.Printf("Error ensuring sample data: %v", err)
	}

	// 3. Fix fleet vehicles table
	if err := FixFleetVehiclesTable(); err != nil {
		log.Printf("Error fixing fleet vehicles table: %v", err)
	}
	
	// 4. Fix NULL values in all tables
	if err := FixAllNullValues(); err != nil {
		log.Printf("Error fixing NULL values: %v", err)
	}

	// 4. Clear all caches to force data reload
	if dataCache != nil {
		dataCache.mu.Lock()
		dataCache.routes = nil
		dataCache.buses = nil
		dataCache.users = nil
		dataCache.vehicles = nil
		dataCache.lastFetch = make(map[string]time.Time)
		dataCache.mu.Unlock()
		log.Println("All caches cleared")
	}

	log.Println("=== Page fixes completed ===")
	return nil
}

// EnsureSampleData creates sample data if tables are empty
func EnsureSampleData() error {
	log.Println("Checking for sample data...")

	// Check and create sample drivers
	var driverCount int
	err := db.QueryRow("SELECT COUNT(*) FROM users WHERE role = 'driver' AND status = 'active'").Scan(&driverCount)
	if err != nil {
		return err
	}

	if driverCount == 0 {
		log.Println("Creating sample drivers...")
		sampleDrivers := []struct {
			Username string
			Name     string
		}{
			{"driver1", "John Smith"},
			{"driver2", "Jane Doe"},
			{"driver3", "Bob Johnson"},
			{"driver4", "Alice Williams"},
			{"driver5", "Charlie Brown"},
		}

		for _, driver := range sampleDrivers {
			hashedPassword, _ := hashPassword("password123")
			_, err = db.Exec(`
				INSERT INTO users (username, password, role, status, registration_date, created_at)
				VALUES ($1, $2, 'driver', 'active', CURRENT_DATE, CURRENT_TIMESTAMP)
				ON CONFLICT (username) DO UPDATE SET status = 'active'
			`, driver.Username, hashedPassword)
			if err != nil {
				log.Printf("Error creating driver %s: %v", driver.Username, err)
			}
		}
	}

	// Check and create sample buses
	var busCount int
	err = db.QueryRow("SELECT COUNT(*) FROM buses WHERE status = 'active'").Scan(&busCount)
	if err != nil {
		return err
	}

	if busCount == 0 {
		log.Println("Creating sample buses...")
		for i := 1; i <= 10; i++ {
			busID := fmt.Sprintf("BUS%03d", i)
			_, err = db.Exec(`
				INSERT INTO buses (bus_id, bus_number, capacity, status, last_maintenance, mileage, created_at)
				VALUES ($1, $2, 72, 'active', CURRENT_DATE - INTERVAL '30 days', $3, CURRENT_TIMESTAMP)
				ON CONFLICT (bus_id) DO UPDATE SET status = 'active'
			`, busID, fmt.Sprintf("%d", 100+i), 10000+i*1000)
			if err != nil {
				log.Printf("Error creating bus %s: %v", busID, err)
			}
		}
	}

	// Create sample route assignments
	if err := CreateSampleAssignments(); err != nil {
		log.Printf("Error creating sample assignments: %v", err)
	}

	return nil
}

// FixAllNullValues fixes NULL values across all tables
func FixAllNullValues() error {
	log.Println("Fixing NULL values in all tables...")

	fixes := []struct {
		table  string
		column string
		value  string
	}{
		// Routes table
		{"routes", "description", "''"},
		{"routes", "positions", "'[]'::jsonb"},
		
		// Buses table
		{"buses", "notes", "''"},
		
		// Users table
		{"users", "registration_date", "CURRENT_DATE"},
		
		// Students table
		{"students", "guardian", "''"},
		{"students", "phone_number", "''"},
		{"students", "alt_phone_number", "''"},
		
		// ECSE students table
		{"ecse_students", "notes", "''"},
		{"ecse_students", "grade", "''"},
		{"ecse_students", "city", "''"},
		{"ecse_students", "state", "''"},
		{"ecse_students", "zip_code", "''"},
		{"ecse_students", "address", "''"},
		
		// Fleet vehicles table
		{"fleet_vehicles", "description", "''"},
		{"fleet_vehicles", "location", "''"},
		{"fleet_vehicles", "tire_size", "''"},
	}

	for _, fix := range fixes {
		query := fmt.Sprintf(
			"UPDATE %s SET %s = %s WHERE %s IS NULL",
			fix.table, fix.column, fix.value, fix.column,
		)
		
		_, err := db.Exec(query)
		if err != nil {
			// Don't fail on individual fixes, just log
			if !strings.Contains(err.Error(), "does not exist") {
				log.Printf("Warning: Could not fix NULL values in %s.%s: %v", 
					fix.table, fix.column, err)
			}
		}
	}

	return nil
}

// VerifyDataAccess checks if data is accessible from all tables
func VerifyDataAccess() error {
	log.Println("=== Verifying data access ===")

	tables := []struct {
		name   string
		query  string
	}{
		{"routes", "SELECT COUNT(*) FROM routes"},
		{"route_assignments", "SELECT COUNT(*) FROM route_assignments"},
		{"buses", "SELECT COUNT(*) FROM buses WHERE status = 'active'"},
		{"users (drivers)", "SELECT COUNT(*) FROM users WHERE role = 'driver' AND status = 'active'"},
		{"students", "SELECT COUNT(*) FROM students WHERE active = true"},
		{"ecse_students", "SELECT COUNT(*) FROM ecse_students"},
		{"fleet_vehicles", "SELECT COUNT(*) FROM fleet_vehicles"},
		{"maintenance_records", "SELECT COUNT(*) FROM maintenance_records"},
		{"service_records", "SELECT COUNT(*) FROM service_records"},
		{"monthly_mileage_reports", "SELECT COUNT(*) FROM monthly_mileage_reports"},
	}

	for _, table := range tables {
		var count int
		err := db.QueryRow(table.query).Scan(&count)
		if err != nil {
			log.Printf("❌ Error accessing %s: %v", table.name, err)
		} else {
			status := "✓"
			if count == 0 {
				status = "⚠️"
			}
			log.Printf("%s %s: %d records", status, table.name, count)
		}
	}

	return nil
}

// RunComprehensiveFix runs all fixes and verifies the system
func RunComprehensiveFix() {
	log.Println("=== Starting comprehensive system fix ===")
	
	// Run all fixes
	if err := FixAllPageIssues(); err != nil {
		log.Printf("Error in page fixes: %v", err)
	}

	// Verify data access
	if err := VerifyDataAccess(); err != nil {
		log.Printf("Error verifying data access: %v", err)
	}

	log.Println("=== Comprehensive fix completed ===")
}