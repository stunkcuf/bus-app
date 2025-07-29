package main

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
)

// VerifyUnusedTables checks for tables that have no data or references
func VerifyUnusedTables(db *sql.DB) error {
	log.Println("🔍 Verifying unused tables...")

	// List of potentially unused tables based on our audit
	suspectedUnused := []string{
		"fleet_vehicles",     // Migrated to vehicles table
		"import_logs",        // Legacy import system
		"import_mappings",    // Legacy import system
		"import_templates",   // Legacy import system
		"data_imports",       // Legacy import system
		"import_history",     // Legacy import system
		"import_configurations", // Legacy import system
		"excel_imports",      // Legacy import system
	}

	unusedTables := []string{}
	tablesWithData := []string{}

	for _, table := range suspectedUnused {
		// Check if table exists
		var exists bool
		err := db.QueryRow(`
			SELECT EXISTS (
				SELECT FROM information_schema.tables 
				WHERE table_schema = 'public' 
				AND table_name = $1
			)`, table).Scan(&exists)
		
		if err != nil {
			log.Printf("Error checking table %s: %v", table, err)
			continue
		}

		if !exists {
			log.Printf("Table %s does not exist (already removed)", table)
			continue
		}

		// Check row count
		var count int
		err = db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&count)
		if err != nil {
			log.Printf("Error counting rows in %s: %v", table, err)
			continue
		}

		if count == 0 {
			unusedTables = append(unusedTables, table)
			log.Printf("✓ Table %s is empty (0 rows)", table)
		} else {
			tablesWithData = append(tablesWithData, table)
			log.Printf("⚠ Table %s has data (%d rows)", table, count)
		}
	}

	// Check for foreign key dependencies
	log.Println("\n🔗 Checking foreign key dependencies...")
	for _, table := range unusedTables {
		var fkCount int
		err := db.QueryRow(`
			SELECT COUNT(*)
			FROM information_schema.table_constraints
			WHERE constraint_type = 'FOREIGN KEY'
			AND (table_name = $1 OR constraint_name LIKE '%' || $1 || '%')
		`, table).Scan(&fkCount)

		if err == nil && fkCount > 0 {
			log.Printf("⚠ Table %s has %d foreign key constraints", table, fkCount)
		}
	}

	// Generate summary
	fmt.Println("\n📊 UNUSED TABLES SUMMARY")
	fmt.Println("========================")
	fmt.Printf("Total suspected unused: %d\n", len(suspectedUnused))
	fmt.Printf("Empty tables (safe to remove): %d\n", len(unusedTables))
	fmt.Printf("Tables with data: %d\n", len(tablesWithData))

	if len(unusedTables) > 0 {
		fmt.Println("\n🗑️ Tables safe to remove:")
		for _, table := range unusedTables {
			fmt.Printf("  - %s\n", table)
		}
	}

	if len(tablesWithData) > 0 {
		fmt.Println("\n⚠️ Tables with data (need review):")
		for _, table := range tablesWithData {
			fmt.Printf("  - %s\n", table)
		}
	}

	// Check code references
	fmt.Println("\n📝 Code reference check needed for:")
	for _, table := range append(unusedTables, tablesWithData...) {
		fmt.Printf("  grep -r '%s' . --include='*.go'\n", table)
	}

	return nil
}

// RemoveUnusedTables removes tables that are verified as unused
func RemoveUnusedTables(db *sql.DB, tablesToRemove []string, dryRun bool) error {
	if dryRun {
		log.Println("🔍 DRY RUN - No changes will be made")
	}

	for _, table := range tablesToRemove {
		// Double-check table is empty
		var count int
		err := db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&count)
		if err != nil {
			log.Printf("Error checking %s: %v", table, err)
			continue
		}

		if count > 0 {
			log.Printf("⚠️ SKIPPING %s - table has %d rows", table, count)
			continue
		}

		// Check for views depending on this table
		var viewCount int
		err = db.QueryRow(`
			SELECT COUNT(*)
			FROM information_schema.view_table_usage
			WHERE table_name = $1
		`, table).Scan(&viewCount)

		if viewCount > 0 {
			log.Printf("⚠️ SKIPPING %s - %d views depend on it", table, viewCount)
			continue
		}

		if dryRun {
			log.Printf("Would drop table: %s", table)
		} else {
			log.Printf("Dropping table: %s", table)
			_, err = db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", table))
			if err != nil {
				log.Printf("❌ Error dropping %s: %v", table, err)
			} else {
				log.Printf("✅ Successfully dropped %s", table)
			}
		}
	}

	return nil
}

// AnalyzeTableUsage provides detailed analysis of table usage
func AnalyzeTableUsage(db *sql.DB) error {
	query := `
		SELECT 
			schemaname,
			tablename,
			pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) as size,
			n_tup_ins as inserts,
			n_tup_upd as updates,
			n_tup_del as deletes,
			n_live_tup as live_rows,
			n_dead_tup as dead_rows,
			last_vacuum,
			last_analyze
		FROM pg_stat_user_tables
		WHERE schemaname = 'public'
		ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC
	`

	rows, err := db.Query(query)
	if err != nil {
		return fmt.Errorf("failed to analyze table usage: %v", err)
	}
	defer rows.Close()

	fmt.Println("\n📊 TABLE USAGE ANALYSIS")
	fmt.Println("======================")
	fmt.Printf("%-30s %-10s %-10s %-10s %-10s %-10s\n", 
		"Table", "Size", "Rows", "Inserts", "Updates", "Deletes")
	fmt.Println(strings.Repeat("-", 80))

	for rows.Next() {
		var schema, table, size string
		var inserts, updates, deletes, liveRows, deadRows int64
		var lastVacuum, lastAnalyze sql.NullTime

		err := rows.Scan(&schema, &table, &size, &inserts, &updates, 
			&deletes, &liveRows, &deadRows, &lastVacuum, &lastAnalyze)
		if err != nil {
			continue
		}

		fmt.Printf("%-30s %-10s %-10d %-10d %-10d %-10d\n",
			table, size, liveRows, inserts, updates, deletes)
	}

	return nil
}