package main

import (
	"log"
)

func CheckDataIssues() {
	log.Println("🔍 Checking data issues...")

	// Check drivers
	var driverCount int
	err := db.Get(&driverCount, `SELECT COUNT(*) FROM users WHERE role = 'driver'`)
	if err != nil {
		log.Printf("Error checking drivers: %v", err)
	} else {
		log.Printf("Total drivers in database: %d", driverCount)
	}

	// Check active drivers
	var activeDriverCount int
	err = db.Get(&activeDriverCount, `SELECT COUNT(*) FROM users WHERE role = 'driver' AND status = 'active'`)
	if err != nil {
		log.Printf("Error checking active drivers: %v", err)
	} else {
		log.Printf("Active drivers in database: %d", activeDriverCount)
	}

	// List all drivers
	type DriverInfo struct {
		Username string `db:"username"`
		Status   string `db:"status"`
		Role     string `db:"role"`
	}
	var drivers []DriverInfo
	err = db.Select(&drivers, `SELECT username, status, role FROM users WHERE role = 'driver' LIMIT 10`)
	if err != nil {
		log.Printf("Error listing drivers: %v", err)
	} else {
		for _, d := range drivers {
			log.Printf("Driver: %s (status: %s, role: %s)", d.Username, d.Status, d.Role)
		}
	}

	// Check routes
	var routeCount int
	err = db.Get(&routeCount, `SELECT COUNT(*) FROM routes`)
	if err != nil {
		log.Printf("Error checking routes: %v", err)
	} else {
		log.Printf("Total routes in database: %d", routeCount)
	}

	// Check buses
	var busCount int
	err = db.Get(&busCount, `SELECT COUNT(*) FROM buses WHERE status = 'active'`)
	if err != nil {
		log.Printf("Error checking buses: %v", err)
	} else {
		log.Printf("Active buses in database: %d", busCount)
	}

	// Check route assignments
	var assignmentCount int
	err = db.Get(&assignmentCount, `SELECT COUNT(*) FROM route_assignments`)
	if err != nil {
		log.Printf("Error checking assignments: %v", err)
	} else {
		log.Printf("Total route assignments: %d", assignmentCount)
	}

	// Check constraint issue
	log.Println("Checking route_assignments constraint...")
	var constraintExists bool
	err = db.Get(&constraintExists, `
		SELECT EXISTS (
			SELECT 1 FROM pg_constraint 
			WHERE conname = 'route_assignments_unique_assignment'
		)
	`)
	if err != nil {
		log.Printf("Error checking constraint: %v", err)
	} else {
		log.Printf("Constraint route_assignments_unique_assignment exists: %v", constraintExists)
	}
}