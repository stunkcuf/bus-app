package main

import (
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
	"github.com/jmoiron/sqlx"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgresql://postgres:jTKOWGEzlprRGbkPBCgGxsgnwyLeGoDL@shortline.proxy.rlwy.net:40148/railway"
	}

	db, err := sqlx.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer db.Close()

	fmt.Println("Checking indexes on vehicles table...")

	// Check indexes
	var indexes []struct {
		TableName string `db:"tablename"`
		IndexName string `db:"indexname"`
		IndexDef  string `db:"indexdef"`
	}
	
	err = db.Select(&indexes, `
		SELECT tablename, indexname, indexdef
		FROM pg_indexes
		WHERE tablename = 'vehicles'
		ORDER BY indexname
	`)
	
	if err != nil {
		fmt.Printf("Error getting indexes: %v\n", err)
		return
	}
	
	fmt.Printf("\nFound %d indexes:\n", len(indexes))
	for _, idx := range indexes {
		fmt.Printf("  - %s\n    %s\n", idx.IndexName, idx.IndexDef)
	}

	// Check table stats
	var stats struct {
		RelName   string `db:"relname"`
		RelTuples int    `db:"reltuples"`
		RelPages  int    `db:"relpages"`
	}
	
	err = db.Get(&stats, `
		SELECT relname, reltuples::int, relpages
		FROM pg_class
		WHERE relname = 'vehicles'
	`)
	
	if err == nil {
		fmt.Printf("\nTable stats:\n")
		fmt.Printf("  Estimated rows: %d\n", stats.RelTuples)
		fmt.Printf("  Pages: %d\n", stats.RelPages)
	}

	// Run EXPLAIN on the query
	fmt.Println("\nQuery plan:")
	rows, err := db.Query(`
		EXPLAIN ANALYZE
		SELECT vehicle_id, model, description, year, tire_size, license, 
		       oil_status, tire_status, status, maintenance_notes, 
		       serial_number, base, service_interval, current_mileage, 
		       last_oil_change, last_tire_service, updated_at, created_at, import_id
		FROM vehicles 
		ORDER BY vehicle_id
		LIMIT 5
	`)
	
	if err != nil {
		fmt.Printf("Error running EXPLAIN: %v\n", err)
		return
	}
	defer rows.Close()
	
	for rows.Next() {
		var plan string
		rows.Scan(&plan)
		fmt.Printf("  %s\n", plan)
	}
}