package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	// Read the PostgreSQL backup file
	content, err := os.ReadFile("railway_backup.sql")
	if err != nil {
		log.Fatal("Failed to read backup file:", err)
	}

	// Create SQLite database
	os.Remove("fleet_management.db") // Remove if exists
	db, err := sql.Open("sqlite3", "fleet_management.db")
	if err != nil {
		log.Fatal("Failed to create SQLite database:", err)
	}
	defer db.Close()

	// Split content into statements
	statements := strings.Split(string(content), ";\n")
	
	successCount := 0
	errorCount := 0
	
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" || strings.HasPrefix(stmt, "--") {
			continue
		}
		
		// Convert PostgreSQL syntax to SQLite
		stmt = convertToSQLite(stmt)
		
		// Execute statement
		_, err := db.Exec(stmt)
		if err != nil {
			if !strings.Contains(err.Error(), "already exists") {
				log.Printf("Error executing: %v\n", err)
				if len(stmt) > 100 {
					log.Printf("Statement: %s...\n", stmt[:100])
				} else {
					log.Printf("Statement: %s\n", stmt)
				}
				errorCount++
			}
		} else {
			successCount++
		}
	}
	
	fmt.Printf("\nSQLite database created: fleet_management.db\n")
	fmt.Printf("Successful statements: %d\n", successCount)
	fmt.Printf("Failed statements: %d\n", errorCount)
	
	// Test the database
	var busCount int
	err = db.QueryRow("SELECT COUNT(*) FROM buses").Scan(&busCount)
	if err == nil {
		fmt.Printf("\nBuses in database: %d\n", busCount)
	}
	
	var maintenanceCount int
	err = db.QueryRow("SELECT COUNT(*) FROM maintenance_records").Scan(&maintenanceCount)
	if err == nil {
		fmt.Printf("Maintenance records: %d\n", maintenanceCount)
	}
}

func convertToSQLite(stmt string) string {
	// Remove PostgreSQL specific syntax
	stmt = strings.ReplaceAll(stmt, "SERIAL", "INTEGER")
	stmt = strings.ReplaceAll(stmt, "DOUBLE PRECISION", "REAL")
	stmt = strings.ReplaceAll(stmt, "TIMESTAMP", "DATETIME")
	stmt = strings.ReplaceAll(stmt, "JSONB", "TEXT")
	stmt = strings.ReplaceAll(stmt, "BOOLEAN", "INTEGER")
	stmt = strings.ReplaceAll(stmt, "TRUE", "1")
	stmt = strings.ReplaceAll(stmt, "FALSE", "0")
	stmt = strings.ReplaceAll(stmt, " IF NOT EXISTS", "")
	stmt = strings.ReplaceAll(stmt, "NUMERIC", "REAL")
	
	// Remove array_to_string and other PostgreSQL functions
	if strings.Contains(stmt, "array_to_string") {
		return ""
	}
	
	return stmt
}