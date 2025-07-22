package main

import (
	"fmt"
	"log"
)

// checkDatabaseSchema verifies table schemas and identifies missing columns
func checkDatabaseSchema() error {
	if db == nil {
		return fmt.Errorf("database not initialized")
	}

	fmt.Println("🔍 Checking database schema...")

	// Check service_records table structure
	fmt.Println("\n📋 SERVICE_RECORDS table columns:")
	rows, err := db.Query(`
		SELECT column_name, data_type, is_nullable 
		FROM information_schema.columns 
		WHERE table_name = 'service_records' 
		ORDER BY ordinal_position
	`)
	if err != nil {
		return fmt.Errorf("failed to query service_records columns: %w", err)
	}
	defer rows.Close()

	serviceRecordsHasMaintenanceDate := false
	for rows.Next() {
		var colName, dataType, nullable string
		err := rows.Scan(&colName, &dataType, &nullable)
		if err != nil {
			continue
		}
		fmt.Printf("  - %s (%s) nullable: %s\n", colName, dataType, nullable)
		if colName == "maintenance_date" {
			serviceRecordsHasMaintenanceDate = true
		}
	}

	// Check maintenance_records table structure
	fmt.Println("\n📋 MAINTENANCE_RECORDS table columns:")
	rows2, err := db.Query(`
		SELECT column_name, data_type, is_nullable 
		FROM information_schema.columns 
		WHERE table_name = 'maintenance_records' 
		ORDER BY ordinal_position
	`)
	if err != nil {
		return fmt.Errorf("failed to query maintenance_records columns: %w", err)
	}
	defer rows2.Close()

	maintenanceRecordsHasServiceDate := false
	for rows2.Next() {
		var colName, dataType, nullable string
		err := rows2.Scan(&colName, &dataType, &nullable)
		if err != nil {
			continue
		}
		fmt.Printf("  - %s (%s) nullable: %s\n", colName, dataType, nullable)
		if colName == "service_date" {
			maintenanceRecordsHasServiceDate = true
		}
	}

	// Check existing indexes
	fmt.Println("\n🗂️ Existing indexes:")
	rows3, err := db.Query(`
		SELECT indexname, tablename 
		FROM pg_indexes 
		WHERE tablename IN ('service_records', 'maintenance_records')
		ORDER BY tablename, indexname
	`)
	if err != nil {
		return fmt.Errorf("failed to query indexes: %w", err)
	}
	defer rows3.Close()

	for rows3.Next() {
		var indexName, tableName string
		err := rows3.Scan(&indexName, &tableName)
		if err != nil {
			continue
		}
		fmt.Printf("  - %s on %s\n", indexName, tableName)
	}

	// Check record counts
	fmt.Println("\n📊 Record counts:")
	var serviceCount, maintenanceCount int
	db.QueryRow("SELECT COUNT(*) FROM service_records").Scan(&serviceCount)
	db.QueryRow("SELECT COUNT(*) FROM maintenance_records").Scan(&maintenanceCount)
	fmt.Printf("  - service_records: %d records\n", serviceCount)
	fmt.Printf("  - maintenance_records: %d records\n", maintenanceCount)

	// Summary
	fmt.Println("\n✅ Schema Analysis:")
	fmt.Printf("  - service_records has maintenance_date: %v\n", serviceRecordsHasMaintenanceDate)
	fmt.Printf("  - maintenance_records has service_date: %v\n", maintenanceRecordsHasServiceDate)

	if !serviceRecordsHasMaintenanceDate {
		fmt.Println("❌ ISSUE: service_records missing maintenance_date column")
	}
	if !maintenanceRecordsHasServiceDate {
		fmt.Println("❌ ISSUE: maintenance_records missing service_date column")
	}

	return nil
}

// fixDatabaseSchema attempts to fix schema issues
func fixDatabaseSchema() error {
	fmt.Println("🔧 Attempting to fix database schema issues...")

	// Fix 1: Ensure service_records has maintenance_date column
	_, err := db.Exec(`
		DO $$ 
		BEGIN 
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
				WHERE table_name = 'service_records' AND column_name = 'maintenance_date') THEN
				ALTER TABLE service_records ADD COLUMN maintenance_date DATE;
				RAISE NOTICE 'Added maintenance_date column to service_records';
			END IF;
		END $$
	`)
	if err != nil {
		log.Printf("Warning: Could not add maintenance_date to service_records: %v", err)
	}

	// Fix 2: Ensure maintenance_records has service_date column
	_, err = db.Exec(`
		DO $$ 
		BEGIN 
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
				WHERE table_name = 'maintenance_records' AND column_name = 'service_date') THEN
				ALTER TABLE maintenance_records ADD COLUMN service_date DATE;
				RAISE NOTICE 'Added service_date column to maintenance_records';
			END IF;
		END $$
	`)
	if err != nil {
		log.Printf("Warning: Could not add service_date to maintenance_records: %v", err)
	}

	// Fix 3: Create missing indexes (safe with IF NOT EXISTS)
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_service_records_maintenance_date 
		 ON service_records(maintenance_date DESC) WHERE maintenance_date IS NOT NULL`,
		
		`CREATE INDEX IF NOT EXISTS idx_maintenance_records_service_date 
		 ON maintenance_records(service_date DESC) WHERE service_date IS NOT NULL`,
		
		`CREATE INDEX IF NOT EXISTS idx_maintenance_records_vehicle_service_date 
		 ON maintenance_records(vehicle_id, service_date DESC) WHERE service_date IS NOT NULL`,
	}

	for _, indexSQL := range indexes {
		_, err := db.Exec(indexSQL)
		if err != nil {
			log.Printf("Warning: Could not create index: %v", err)
		} else {
			fmt.Printf("✅ Created index successfully\n")
		}
	}

	fmt.Println("🔧 Schema fix completed")
	return nil
}

func main() {
	// Initialize database connection
	setupDatabase()
	
	// Check schema
	if err := checkDatabaseSchema(); err != nil {
		log.Fatalf("Schema check failed: %v", err)
	}

	// Fix schema issues
	if err := fixDatabaseSchema(); err != nil {
		log.Fatalf("Schema fix failed: %v", err)
	}

	// Re-check after fixes
	fmt.Println("\n🔄 Re-checking schema after fixes...")
	if err := checkDatabaseSchema(); err != nil {
		log.Fatalf("Schema re-check failed: %v", err)
	}
}