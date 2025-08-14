package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
)

func main() {
	// Get database URL from environment
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable is not set")
	}

	// Connect to database
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	fmt.Println("=== Optimizing Maintenance Records Query ===")
	fmt.Println()

	// Test current query performance
	fmt.Println("Testing current query performance...")
	start := time.Now()
	
	var count int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM maintenance_records
	`).Scan(&count)
	
	if err != nil {
		log.Fatal("Failed to count records:", err)
	}
	
	fmt.Printf("Total maintenance records: %d\n", count)
	fmt.Printf("Count query took: %v\n", time.Since(start))
	
	// Test the actual slow query
	fmt.Println("\nTesting main query performance (before optimization)...")
	start = time.Now()
	
	rows, err := db.Query(`
		SELECT id, vehicle_number, service_date, mileage, po_number, cost,
		       work_description, raw_data, created_at, updated_at, vehicle_id, date
		FROM maintenance_records 
		ORDER BY 
			COALESCE(service_date, date, created_at) DESC,
			vehicle_number, id
		LIMIT 25
	`)
	if err != nil {
		log.Printf("Query error: %v", err)
	} else {
		rows.Close()
		fmt.Printf("Main query took: %v\n", time.Since(start))
	}

	// Create optimized indexes
	fmt.Println("\n=== Creating Optimized Indexes ===")
	
	indexes := []struct {
		name  string
		query string
	}{
		{
			"idx_maintenance_dates",
			`CREATE INDEX IF NOT EXISTS idx_maintenance_dates 
			ON maintenance_records(service_date DESC NULLS LAST, date DESC NULLS LAST, created_at DESC)`,
		},
		{
			"idx_maintenance_vehicle_date",
			`CREATE INDEX IF NOT EXISTS idx_maintenance_vehicle_date 
			ON maintenance_records(vehicle_number, service_date DESC NULLS LAST)`,
		},
		{
			"idx_maintenance_composite",
			`CREATE INDEX IF NOT EXISTS idx_maintenance_composite 
			ON maintenance_records(vehicle_number, COALESCE(service_date, date, created_at) DESC)`,
		},
	}

	for _, idx := range indexes {
		fmt.Printf("\nCreating index: %s...", idx.name)
		start := time.Now()
		
		_, err := db.Exec(idx.query)
		if err != nil {
			fmt.Printf(" FAILED: %v\n", err)
		} else {
			fmt.Printf(" SUCCESS (took %v)\n", time.Since(start))
		}
	}

	// Analyze table for query planner
	fmt.Println("\nRunning ANALYZE on maintenance_records table...")
	_, err = db.Exec("ANALYZE maintenance_records")
	if err != nil {
		fmt.Printf("ANALYZE failed: %v\n", err)
	} else {
		fmt.Println("ANALYZE completed successfully")
	}

	// Test query performance after optimization
	fmt.Println("\n=== Testing Query Performance After Optimization ===")
	start = time.Now()
	
	rows, err = db.Query(`
		SELECT id, vehicle_number, service_date, mileage, po_number, cost,
		       work_description, raw_data, created_at, updated_at, vehicle_id, date
		FROM maintenance_records 
		ORDER BY 
			COALESCE(service_date, date, created_at) DESC,
			vehicle_number, id
		LIMIT 25
	`)
	if err != nil {
		log.Printf("Query error: %v", err)
	} else {
		rows.Close()
		fmt.Printf("Main query took: %v (after optimization)\n", time.Since(start))
	}

	// Show existing indexes
	fmt.Println("\n=== Current Indexes on maintenance_records ===")
	rows, err = db.Query(`
		SELECT indexname, indexdef
		FROM pg_indexes
		WHERE tablename = 'maintenance_records'
		ORDER BY indexname
	`)
	if err != nil {
		log.Printf("Failed to get indexes: %v", err)
	} else {
		defer rows.Close()
		for rows.Next() {
			var name, def string
			if err := rows.Scan(&name, &def); err == nil {
				fmt.Printf("- %s\n", name)
			}
		}
	}

	fmt.Println("\n✅ Optimization complete!")
	fmt.Println("\nRecommendations:")
	fmt.Println("1. The indexes have been created to optimize the ORDER BY clause")
	fmt.Println("2. Consider adding pagination at the database level instead of loading all records")
	fmt.Println("3. For better performance, consider caching frequently accessed data")
}