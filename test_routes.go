package main

import (
	"database/sql"
	"log"
)

func testRoutesDirectly() {
	log.Println("🔍 Testing routes directly from database...")
	
	// Test 1: Count routes
	var count int
	err := db.Get(&count, "SELECT COUNT(*) FROM routes")
	if err != nil {
		log.Printf("❌ Error counting routes: %v", err)
	} else {
		log.Printf("✅ Total routes in database: %d", count)
	}
	
	// Test 2: Check positions column
	rows, err := db.Query("SELECT route_id, route_name, positions FROM routes LIMIT 5")
	if err != nil {
		log.Printf("❌ Error querying routes: %v", err)
		return
	}
	defer rows.Close()
	
	log.Println("📋 Sample routes:")
	for rows.Next() {
		var routeID, routeName string
		var positions sql.NullString
		
		err := rows.Scan(&routeID, &routeName, &positions)
		if err != nil {
			log.Printf("❌ Error scanning row: %v", err)
			continue
		}
		
		posStr := "NULL"
		if positions.Valid {
			posStr = positions.String
		}
		
		log.Printf("  Route: %s - %s (positions: %s)", routeID, routeName, posStr)
	}
	
	// Test 3: Test loadRoutesFromDB function
	routes, err := loadRoutesFromDB()
	if err != nil {
		log.Printf("❌ loadRoutesFromDB failed: %v", err)
	} else {
		log.Printf("✅ loadRoutesFromDB returned %d routes", len(routes))
		if len(routes) > 0 {
			log.Printf("  First route: %+v", routes[0])
		}
	}
}