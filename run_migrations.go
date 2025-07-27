package main

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"time"
)

// RunMigrations executes database migrations
func RunMigrations(db *sql.DB) error {
	log.Println("Starting database migrations...")

	// Create migrations tracking table if it doesn't exist
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS migrations (
			id SERIAL PRIMARY KEY,
			filename VARCHAR(255) UNIQUE NOT NULL,
			executed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			success BOOLEAN DEFAULT true,
			error_message TEXT
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Check if consolidate_vehicles migration has been run
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM migrations WHERE filename = $1", "consolidate_vehicles_tables.sql").Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check migration status: %w", err)
	}

	if count > 0 {
		log.Println("Vehicle consolidation migration already completed")
		return nil
	}

	// Run the consolidation migration
	log.Println("Running vehicle consolidation migration...")
	
	migrationPath := filepath.Join("migrations", "consolidate_vehicles_tables.sql")
	migrationSQL, err := ioutil.ReadFile(migrationPath)
	if err != nil {
		return fmt.Errorf("failed to read migration file: %w", err)
	}

	// Start transaction
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Execute migration
	_, err = tx.Exec(string(migrationSQL))
	if err != nil {
		// Log the error
		db.Exec("INSERT INTO migrations (filename, success, error_message) VALUES ($1, $2, $3)",
			"consolidate_vehicles_tables.sql", false, err.Error())
		return fmt.Errorf("migration failed: %w", err)
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit migration: %w", err)
	}

	// Record successful migration
	_, err = db.Exec("INSERT INTO migrations (filename, success) VALUES ($1, $2)",
		"consolidate_vehicles_tables.sql", true)
	if err != nil {
		log.Printf("Warning: Migration succeeded but failed to record: %v", err)
	}

	log.Println("Vehicle consolidation migration completed successfully")

	// Generate migration report
	report, err := GenerateMigrationReport(db)
	if err != nil {
		log.Printf("Warning: Failed to generate report: %v", err)
	} else {
		log.Println("Migration Report:")
		log.Println(report)
	}

	return nil
}

// GenerateMigrationReport creates a summary of the migration
func GenerateMigrationReport(db *sql.DB) (string, error) {
	var report string
	
	// Count vehicles by type
	var busCount, fleetCount, totalCount int
	db.QueryRow("SELECT COUNT(*) FROM vehicles WHERE vehicle_type = 'bus'").Scan(&busCount)
	db.QueryRow("SELECT COUNT(*) FROM vehicles WHERE vehicle_type = 'fleet'").Scan(&fleetCount)
	db.QueryRow("SELECT COUNT(*) FROM vehicles").Scan(&totalCount)

	report = fmt.Sprintf(`
Vehicle Migration Summary:
- Total Vehicles: %d
- Buses: %d
- Fleet Vehicles: %d
- Migration Date: %s
`, totalCount, busCount, fleetCount, time.Now().Format("2006-01-02 15:04:05"))

	// Check for any issues
	var duplicateLicenses int
	db.QueryRow(`
		SELECT COUNT(*) FROM (
			SELECT license, COUNT(*) 
			FROM vehicles 
			WHERE license IS NOT NULL AND license != ''
			GROUP BY license 
			HAVING COUNT(*) > 1
		) as dups
	`).Scan(&duplicateLicenses)

	if duplicateLicenses > 0 {
		report += fmt.Sprintf("\nWarning: %d duplicate license plates found\n", duplicateLicenses)
	}

	return report, nil
}

// CleanupUnusedTables removes tables that are confirmed unused
func CleanupUnusedTables(db *sql.DB) error {
	log.Println("Checking for unused tables...")

	// Tables to check and potentially remove
	unusedTables := []struct {
		name     string
		checkSQL string
	}{
		{"ecse_services", "SELECT COUNT(*) FROM ecse_services"},
		{"ecse_assessments", "SELECT COUNT(*) FROM ecse_assessments"},
		{"ecse_attendance", "SELECT COUNT(*) FROM ecse_attendance"},
		{"scheduled_exports", "SELECT COUNT(*) FROM scheduled_exports"},
		{"saved_reports", "SELECT COUNT(*) FROM saved_reports"},
		{"program_staff", "SELECT COUNT(*) FROM program_staff"},
		{"import_history", "SELECT COUNT(*) FROM import_history"},
		{"import_errors", "SELECT COUNT(*) FROM import_errors"},
	}

	for _, table := range unusedTables {
		var count int
		err := db.QueryRow(table.checkSQL).Scan(&count)
		if err != nil {
			log.Printf("Error checking table %s: %v", table.name, err)
			continue
		}

		if count == 0 {
			log.Printf("Table %s is empty and can be removed", table.name)
			// For safety, we'll create a backup before dropping
			backupSQL := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s_backup_%s AS SELECT * FROM %s",
				table.name, time.Now().Format("20060102"), table.name)
			_, err = db.Exec(backupSQL)
			if err != nil {
				log.Printf("Failed to backup %s: %v", table.name, err)
				continue
			}

			// Now we can safely drop the table
			_, err = db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", table.name))
			if err != nil {
				log.Printf("Failed to drop %s: %v", table.name, err)
			} else {
				log.Printf("Successfully removed unused table: %s", table.name)
			}
		} else {
			log.Printf("Table %s has %d records - keeping it", table.name, count)
		}
	}

	return nil
}

// UpdateFleetVehicleReferences updates all code references from fleet_vehicles to vehicles
func UpdateFleetVehicleReferences() error {
	log.Println("Code references should be updated manually to ensure proper testing")
	log.Println("Main areas to update:")
	log.Println("1. Change all queries from 'fleet_vehicles' to 'vehicles'")
	log.Println("2. Add WHERE vehicle_type = 'fleet' conditions where needed")
	log.Println("3. Update any struct tags or model definitions")
	log.Println("4. Update handlers that specifically deal with fleet vehicles")
	return nil
}