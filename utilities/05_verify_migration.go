package main

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	_ "github.com/lib/pq"
)

type ValidationResult struct {
	Category string
	Check    string
	Status   string
	Details  string
}

func main() {
	dbURL := "postgresql://postgres:jTKOWGEzlprRGbkPBCgGxsgnwyLeGoDL@shortline.proxy.rlwy.net:40148/railway"

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer db.Close()

	fmt.Println("=== DATABASE MIGRATION VERIFICATION ===")
	fmt.Println()

	var results []ValidationResult

	// Check 1: Vehicle consolidation
	results = append(results, checkVehicleConsolidation(db)...)

	// Check 2: Maintenance consolidation
	results = append(results, checkMaintenanceConsolidation(db)...)

	// Check 3: Column naming issues
	results = append(results, checkColumnNaming(db)...)

	// Check 4: Foreign key consistency
	results = append(results, checkForeignKeys(db)...)

	// Check 5: Empty tables
	results = append(results, checkEmptyTables(db)...)

	// Check 6: Data integrity
	results = append(results, checkDataIntegrity(db)...)

	// Print results
	printResults(results)
	
	// Generate summary
	generateSummary(results)
}

func checkVehicleConsolidation(db *sql.DB) []ValidationResult {
	var results []ValidationResult

	// Check fleet_vehicles has all data
	var fleetCount int
	err := db.QueryRow("SELECT COUNT(*) FROM fleet_vehicles").Scan(&fleetCount)
	if err != nil {
		results = append(results, ValidationResult{
			Category: "Vehicle Consolidation",
			Check:    "fleet_vehicles table",
			Status:   "ERROR",
			Details:  fmt.Sprintf("Cannot access table: %v", err),
		})
	} else {
		results = append(results, ValidationResult{
			Category: "Vehicle Consolidation",
			Check:    "fleet_vehicles record count",
			Status:   "PASS",
			Details:  fmt.Sprintf("%d vehicles found", fleetCount),
		})
	}

	// Check vehicle_type column exists and is populated
	var nullTypeCount int
	err = db.QueryRow("SELECT COUNT(*) FROM fleet_vehicles WHERE vehicle_type IS NULL").Scan(&nullTypeCount)
	if err == nil {
		if nullTypeCount > 0 {
			results = append(results, ValidationResult{
				Category: "Vehicle Consolidation",
				Check:    "vehicle_type population",
				Status:   "WARNING",
				Details:  fmt.Sprintf("%d vehicles without type", nullTypeCount),
			})
		} else {
			results = append(results, ValidationResult{
				Category: "Vehicle Consolidation",
				Check:    "vehicle_type population",
				Status:   "PASS",
				Details:  "All vehicles have type assigned",
			})
		}
	}

	// Check if old tables still exist
	oldTables := []string{"buses", "vehicles", "school_buses", "agency_vehicles"}
	for _, table := range oldTables {
		var exists bool
		err := db.QueryRow(`
			SELECT EXISTS(
				SELECT 1 FROM information_schema.tables 
				WHERE table_name = $1
			)`, table).Scan(&exists)
		
		if err == nil && exists {
			results = append(results, ValidationResult{
				Category: "Vehicle Consolidation",
				Check:    fmt.Sprintf("Old table: %s", table),
				Status:   "WARNING",
				Details:  "Table still exists - should be removed after verification",
			})
		}
	}

	return results
}

func checkMaintenanceConsolidation(db *sql.DB) []ValidationResult {
	var results []ValidationResult

	// Check maintenance_records count
	var maintCount int
	err := db.QueryRow("SELECT COUNT(*) FROM maintenance_records").Scan(&maintCount)
	if err == nil {
		results = append(results, ValidationResult{
			Category: "Maintenance Consolidation",
			Check:    "maintenance_records count",
			Status:   "PASS",
			Details:  fmt.Sprintf("%d maintenance records", maintCount),
		})
	}

	// Check for records without vehicle_number
	var noVehicleCount int
	err = db.QueryRow("SELECT COUNT(*) FROM maintenance_records WHERE vehicle_number IS NULL").Scan(&noVehicleCount)
	if err == nil && noVehicleCount > 0 {
		results = append(results, ValidationResult{
			Category: "Maintenance Consolidation",
			Check:    "vehicle_number population",
			Status:   "WARNING",
			Details:  fmt.Sprintf("%d records without vehicle_number", noVehicleCount),
		})
	}

	// Check if service_records still has data
	var serviceCount int
	err = db.QueryRow("SELECT COUNT(*) FROM service_records").Scan(&serviceCount)
	if err == nil && serviceCount > 0 {
		results = append(results, ValidationResult{
			Category: "Maintenance Consolidation",
			Check:    "service_records migration",
			Status:   "INFO",
			Details:  fmt.Sprintf("%d records still in service_records", serviceCount),
		})
	}

	return results
}

func checkColumnNaming(db *sql.DB) []ValidationResult {
	var results []ValidationResult

	// Check for tables with unnamed columns
	query := `
		SELECT t.table_name, COUNT(*) as unnamed_count
		FROM information_schema.columns c
		JOIN information_schema.tables t ON c.table_name = t.table_name
		WHERE t.table_schema = 'public' 
		AND t.table_type = 'BASE TABLE'
		AND c.column_name LIKE 'unnamed_%'
		GROUP BY t.table_name
	`

	rows, err := db.Query(query)
	if err != nil {
		results = append(results, ValidationResult{
			Category: "Column Naming",
			Check:    "Unnamed columns",
			Status:   "ERROR",
			Details:  fmt.Sprintf("Cannot check: %v", err),
		})
		return results
	}
	defer rows.Close()

	hasUnnamed := false
	for rows.Next() {
		var tableName string
		var count int
		rows.Scan(&tableName, &count)
		
		results = append(results, ValidationResult{
			Category: "Column Naming",
			Check:    fmt.Sprintf("Table: %s", tableName),
			Status:   "WARNING",
			Details:  fmt.Sprintf("%d unnamed columns", count),
		})
		hasUnnamed = true
	}

	if !hasUnnamed {
		results = append(results, ValidationResult{
			Category: "Column Naming",
			Check:    "Unnamed columns",
			Status:   "PASS",
			Details:  "No unnamed columns found",
		})
	}

	return results
}

func checkForeignKeys(db *sql.DB) []ValidationResult {
	var results []ValidationResult

	// Check for inconsistent vehicle references
	tables := []struct {
		table  string
		column string
	}{
		{"route_assignments", "bus_id"},
		{"route_assignments", "vehicle_id"},
		{"driver_logs", "bus_id"},
		{"driver_logs", "vehicle_id"},
		{"fuel_records", "vehicle_id"},
		{"monthly_mileage_reports", "bus_id"},
	}

	for _, t := range tables {
		var exists bool
		err := db.QueryRow(`
			SELECT EXISTS(
				SELECT 1 FROM information_schema.columns 
				WHERE table_name = $1 AND column_name = $2
			)`, t.table, t.column).Scan(&exists)
		
		if err == nil && exists {
			var nonNullCount int
			db.QueryRow(fmt.Sprintf(
				"SELECT COUNT(*) FROM %s WHERE %s IS NOT NULL", 
				t.table, t.column,
			)).Scan(&nonNullCount)
			
			if nonNullCount > 0 {
				status := "INFO"
				if strings.Contains(t.column, "bus_id") {
					status = "WARNING"
				}
				
				results = append(results, ValidationResult{
					Category: "Foreign Keys",
					Check:    fmt.Sprintf("%s.%s", t.table, t.column),
					Status:   status,
					Details:  fmt.Sprintf("%d non-null values", nonNullCount),
				})
			}
		}
	}

	return results
}

func checkEmptyTables(db *sql.DB) []ValidationResult {
	var results []ValidationResult

	// Get all tables with 0 rows
	query := `
		SELECT table_name 
		FROM information_schema.tables 
		WHERE table_schema = 'public' 
		AND table_type = 'BASE TABLE'
		ORDER BY table_name
	`

	rows, err := db.Query(query)
	if err != nil {
		return results
	}
	defer rows.Close()

	emptyCount := 0
	var emptyTables []string

	for rows.Next() {
		var tableName string
		rows.Scan(&tableName)
		
		var count int
		err := db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)).Scan(&count)
		if err == nil && count == 0 {
			emptyTables = append(emptyTables, tableName)
			emptyCount++
		}
	}

	if emptyCount > 0 {
		results = append(results, ValidationResult{
			Category: "Empty Tables",
			Check:    "Empty table count",
			Status:   "INFO",
			Details:  fmt.Sprintf("%d empty tables: %s", emptyCount, strings.Join(emptyTables, ", ")),
		})
	} else {
		results = append(results, ValidationResult{
			Category: "Empty Tables",
			Check:    "Empty table count",
			Status:   "PASS",
			Details:  "No empty tables found",
		})
	}

	return results
}

func checkDataIntegrity(db *sql.DB) []ValidationResult {
	var results []ValidationResult

	// Check vehicle number consistency in fleet_vehicles
	var duplicates int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM (
			SELECT vehicle_number, COUNT(*) as cnt
			FROM fleet_vehicles
			WHERE vehicle_number IS NOT NULL
			GROUP BY vehicle_number
			HAVING COUNT(*) > 1
		) as dups
	`).Scan(&duplicates)
	
	if err == nil {
		if duplicates > 0 {
			results = append(results, ValidationResult{
				Category: "Data Integrity",
				Check:    "Duplicate vehicle numbers",
				Status:   "WARNING",
				Details:  fmt.Sprintf("%d duplicate vehicle numbers found", duplicates),
			})
		} else {
			results = append(results, ValidationResult{
				Category: "Data Integrity",
				Check:    "Vehicle number uniqueness",
				Status:   "PASS",
				Details:  "All vehicle numbers are unique",
			})
		}
	}

	// Check for orphaned maintenance records
	var orphaned int
	err = db.QueryRow(`
		SELECT COUNT(*) 
		FROM maintenance_records m
		WHERE m.vehicle_number IS NOT NULL
		AND NOT EXISTS (
			SELECT 1 FROM fleet_vehicles f 
			WHERE f.vehicle_number = m.vehicle_number
		)
	`).Scan(&orphaned)
	
	if err == nil && orphaned > 0 {
		results = append(results, ValidationResult{
			Category: "Data Integrity",
			Check:    "Orphaned maintenance records",
			Status:   "WARNING",
			Details:  fmt.Sprintf("%d maintenance records reference non-existent vehicles", orphaned),
		})
	}

	return results
}

func printResults(results []ValidationResult) {
	fmt.Println("\n=== VALIDATION RESULTS ===\n")

	currentCategory := ""
	for _, result := range results {
		if result.Category != currentCategory {
			if currentCategory != "" {
				fmt.Println()
			}
			fmt.Printf("%s:\n", result.Category)
			currentCategory = result.Category
		}

		status := result.Status
		switch status {
		case "PASS":
			status = "✓ PASS"
		case "WARNING":
			status = "⚠ WARN"
		case "ERROR":
			status = "✗ ERROR"
		case "INFO":
			status = "ℹ INFO"
		}

		fmt.Printf("  %-8s %-40s %s\n", status, result.Check, result.Details)
	}
}

func generateSummary(results []ValidationResult) {
	passCount := 0
	warnCount := 0
	errorCount := 0
	
	for _, result := range results {
		switch result.Status {
		case "PASS":
			passCount++
		case "WARNING":
			warnCount++
		case "ERROR":
			errorCount++
		}
	}

	fmt.Println("\n=== SUMMARY ===")
	fmt.Printf("Total checks: %d\n", len(results))
	fmt.Printf("  ✓ Passed:  %d\n", passCount)
	fmt.Printf("  ⚠ Warnings: %d\n", warnCount)
	fmt.Printf("  ✗ Errors:   %d\n", errorCount)

	if errorCount == 0 && warnCount == 0 {
		fmt.Println("\n✅ Database migration completed successfully!")
	} else if errorCount == 0 {
		fmt.Println("\n⚠️  Database migration completed with warnings. Review and address as needed.")
	} else {
		fmt.Println("\n❌ Database migration has errors that need to be resolved.")
	}
}