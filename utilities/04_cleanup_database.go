package main

import (
	"database/sql"
	"fmt"
	"log"
	_ "github.com/lib/pq"
)

func main() {
	dbURL := "postgresql://postgres:jTKOWGEzlprRGbkPBCgGxsgnwyLeGoDL@shortline.proxy.rlwy.net:40148/railway"

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer db.Close()

	fmt.Println("=== DATABASE CLEANUP ===")
	fmt.Println("This will remove empty redundant tables and fix column names")
	fmt.Println()

	// Step 1: Show tables to be removed
	fmt.Println("Step 1: Tables marked for removal (empty and redundant):")
	tablesToRemove := []string{
		"school_buses",          // Empty, redundant with buses
		"agency_vehicles",       // Empty, redundant with vehicles
		"all_vehicle_mileage",   // Empty, redundant with mileage tables
		"bus_maintenance_logs",  // Empty, redundant with maintenance_records
		"vehicle_maintenance_logs", // Empty, redundant with maintenance_records
		"mileage_reports",       // Empty, redundant with monthly_mileage_reports
		"mileage_records",       // Empty, redundant with monthly_mileage_reports
	}

	for _, table := range tablesToRemove {
		var count int
		err := db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&count)
		if err != nil {
			fmt.Printf("  ✗ %s - Error: %v\n", table, err)
		} else {
			fmt.Printf("  • %s (%d rows)\n", table, count)
		}
	}

	// Step 2: Confirm before proceeding
	fmt.Println("\nWARNING: This will permanently delete the above tables!")
	fmt.Println("Press Enter to continue or Ctrl+C to cancel...")
	fmt.Scanln()

	// Step 3: Drop tables
	fmt.Println("\nStep 2: Dropping empty tables...")
	for _, table := range tablesToRemove {
		_, err := db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", table))
		if err != nil {
			fmt.Printf("  ✗ Failed to drop %s: %v\n", table, err)
		} else {
			fmt.Printf("  ✓ Dropped %s\n", table)
		}
	}

	// Step 4: Update foreign key references
	fmt.Println("\nStep 3: Standardizing foreign key references...")
	updateForeignKeys(db)

	// Step 5: Create views for backward compatibility
	fmt.Println("\nStep 4: Creating compatibility views...")
	createCompatibilityViews(db)

	// Step 6: Add indexes for performance
	fmt.Println("\nStep 5: Adding performance indexes...")
	addIndexes(db)

	fmt.Println("\n=== CLEANUP COMPLETE ===")
	showFinalSummary(db)
}

func updateForeignKeys(db *sql.DB) {
	// Update tables that use bus_id to use vehicle_id instead
	updates := []struct {
		table    string
		oldCol   string
		newCol   string
		addCol   bool
	}{
		{"route_assignments", "bus_id", "vehicle_id", true},
		{"driver_logs", "bus_id", "vehicle_id", true},
		{"monthly_mileage_reports", "bus_id", "vehicle_id", true},
	}

	for _, update := range updates {
		if update.addCol {
			// First add the new column if it doesn't exist
			_, err := db.Exec(fmt.Sprintf(`
				ALTER TABLE %s 
				ADD COLUMN IF NOT EXISTS %s VARCHAR(50)
			`, update.table, update.newCol))
			if err != nil {
				fmt.Printf("  ✗ Failed to add %s to %s: %v\n", update.newCol, update.table, err)
				continue
			}
		}

		// Copy data from old column to new column
		result, err := db.Exec(fmt.Sprintf(`
			UPDATE %s 
			SET %s = %s 
			WHERE %s IS NULL AND %s IS NOT NULL
		`, update.table, update.newCol, update.oldCol, update.newCol, update.oldCol))
		
		if err != nil {
			fmt.Printf("  ✗ Failed to update %s.%s: %v\n", update.table, update.newCol, err)
		} else {
			affected, _ := result.RowsAffected()
			fmt.Printf("  ✓ Updated %s.%s (%d rows)\n", update.table, update.newCol, affected)
		}
	}
}

func createCompatibilityViews(db *sql.DB) {
	// Create views that map old table names to new consolidated tables
	views := []struct {
		viewName string
		query    string
	}{
		{
			"buses_view",
			`CREATE OR REPLACE VIEW buses AS
			 SELECT vehicle_number::text as bus_id,
			        COALESCE(description, 'Active') as status,
			        model, 
			        50 as capacity,
			        'good' as oil_status,
			        'good' as tire_status,
			        '' as maintenance_notes,
			        updated_at,
			        created_at
			 FROM fleet_vehicles 
			 WHERE vehicle_type = 'bus'`,
		},
		{
			"vehicles_view",
			`CREATE OR REPLACE VIEW vehicles AS
			 SELECT vehicle_number::text as vehicle_id,
			        model,
			        description,
			        year::text as year,
			        tire_size,
			        license,
			        'good' as oil_status,
			        'good' as tire_status,
			        'active' as status,
			        '' as maintenance_notes,
			        serial_number,
			        location as base,
			        3000 as service_interval,
			        updated_at,
			        created_at,
			        vehicle_number::text as import_id
			 FROM fleet_vehicles 
			 WHERE vehicle_type != 'bus'`,
		},
	}

	for _, view := range views {
		_, err := db.Exec(view.query)
		if err != nil {
			fmt.Printf("  ✗ Failed to create view %s: %v\n", view.viewName, err)
		} else {
			fmt.Printf("  ✓ Created compatibility view: %s\n", view.viewName)
		}
	}
}

func addIndexes(db *sql.DB) {
	indexes := []struct {
		table   string
		column  string
		name    string
	}{
		{"fleet_vehicles", "vehicle_number", "idx_fleet_vehicles_number"},
		{"fleet_vehicles", "vehicle_type", "idx_fleet_vehicles_type"},
		{"fleet_vehicles", "license", "idx_fleet_vehicles_license"},
		{"maintenance_records", "vehicle_number", "idx_maintenance_vehicle"},
		{"maintenance_records", "service_date", "idx_maintenance_date"},
		{"monthly_mileage_reports", "bus_id", "idx_mileage_vehicle"},
		{"monthly_mileage_reports", "report_month", "idx_mileage_month"},
		{"ecse_students", "student_id", "idx_ecse_student_id"},
		{"fuel_records", "vehicle_id", "idx_fuel_vehicle"},
		{"fuel_records", "date", "idx_fuel_date"},
	}

	for _, idx := range indexes {
		_, err := db.Exec(fmt.Sprintf(`
			CREATE INDEX IF NOT EXISTS %s ON %s (%s)
		`, idx.name, idx.table, idx.column))
		
		if err != nil {
			fmt.Printf("  ✗ Failed to create index %s: %v\n", idx.name, err)
		} else {
			fmt.Printf("  ✓ Created index: %s\n", idx.name)
		}
	}
}

func showFinalSummary(db *sql.DB) {
	fmt.Println("\n=== FINAL DATABASE SUMMARY ===")
	
	// Count remaining tables
	var tableCount int
	err := db.QueryRow(`
		SELECT COUNT(*) 
		FROM information_schema.tables 
		WHERE table_schema = 'public' 
		AND table_type = 'BASE TABLE'
	`).Scan(&tableCount)
	
	if err == nil {
		fmt.Printf("\nRemaining tables: %d\n", tableCount)
	}

	// Show main tables with counts
	mainTables := []string{
		"fleet_vehicles",
		"maintenance_records", 
		"monthly_mileage_reports",
		"ecse_students",
		"fuel_records",
		"users",
		"students",
		"routes",
		"route_assignments",
	}

	fmt.Println("\nMain tables:")
	for _, table := range mainTables {
		var count int
		err := db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&count)
		if err == nil {
			fmt.Printf("  %-25s: %d records\n", table, count)
		}
	}

	fmt.Println("\nRecommended next steps:")
	fmt.Println("1. Test application with new database structure")
	fmt.Println("2. Update Go models to use fleet_vehicles")
	fmt.Println("3. Update handlers to use standardized vehicle_id")
	fmt.Println("4. Remove references to dropped tables in code")
}