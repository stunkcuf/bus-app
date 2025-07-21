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
	dbURL := "postgresql://postgres:jTKOWGEzlprRGbkPBCgGxsgnwyLeGoDL@shortline.proxy.rlwy.net:40148/railway"

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer db.Close()

	timestamp := time.Now().Format("20060102_150405")
	backupDir := fmt.Sprintf("backup_%s", timestamp)
	
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		log.Fatalf("Failed to create backup directory: %v", err)
	}

	fmt.Printf("Creating backup in directory: %s\n", backupDir)

	// Get all tables
	rows, err := db.Query(`
		SELECT table_name 
		FROM information_schema.tables 
		WHERE table_schema = 'public' 
		AND table_type = 'BASE TABLE'
		ORDER BY table_name
	`)
	if err != nil {
		log.Fatalf("Failed to get tables: %v", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			continue
		}
		tables = append(tables, table)
	}

	fmt.Printf("Found %d tables to backup\n", len(tables))

	// Backup each table structure and data
	for _, table := range tables {
		fmt.Printf("Backing up %s...", table)
		
		// Get table structure
		structureFile := fmt.Sprintf("%s/%s_structure.sql", backupDir, table)
		if err := backupTableStructure(db, table, structureFile); err != nil {
			log.Printf("Error backing up structure for %s: %v", table, err)
			continue
		}
		
		// Get table data
		dataFile := fmt.Sprintf("%s/%s_data.csv", backupDir, table)
		rowCount, err := backupTableData(db, table, dataFile)
		if err != nil {
			log.Printf("Error backing up data for %s: %v", table, err)
			continue
		}
		
		fmt.Printf(" %d rows\n", rowCount)
	}

	// Create summary file
	summaryFile := fmt.Sprintf("%s/backup_summary.txt", backupDir)
	if err := createSummary(tables, summaryFile); err != nil {
		log.Printf("Error creating summary: %v", err)
	}

	fmt.Printf("\nBackup completed successfully in %s\n", backupDir)
}

func backupTableStructure(db *sql.DB, tableName, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	// Get CREATE TABLE statement (PostgreSQL specific)
	query := fmt.Sprintf(`
		SELECT 
			'CREATE TABLE ' || table_name || ' (' ||
			string_agg(
				column_name || ' ' || data_type || 
				CASE WHEN character_maximum_length IS NOT NULL 
					THEN '(' || character_maximum_length || ')' 
					ELSE '' 
				END ||
				CASE WHEN is_nullable = 'NO' 
					THEN ' NOT NULL' 
					ELSE '' 
				END ||
				CASE WHEN column_default IS NOT NULL 
					THEN ' DEFAULT ' || column_default 
					ELSE '' 
				END,
				', '
			) || ');'
		FROM information_schema.columns
		WHERE table_name = '%s'
		GROUP BY table_name
	`, tableName)

	var createStmt string
	err = db.QueryRow(query).Scan(&createStmt)
	if err != nil {
		// Fallback to column listing
		fmt.Fprintf(file, "-- Table: %s\n", tableName)
		fmt.Fprintf(file, "-- Failed to get CREATE statement, listing columns instead\n\n")
		
		cols, err := db.Query(`
			SELECT column_name, data_type, is_nullable, column_default
			FROM information_schema.columns
			WHERE table_name = $1
			ORDER BY ordinal_position
		`, tableName)
		if err != nil {
			return err
		}
		defer cols.Close()
		
		for cols.Next() {
			var colName, dataType, isNullable string
			var colDefault sql.NullString
			cols.Scan(&colName, &dataType, &isNullable, &colDefault)
			fmt.Fprintf(file, "-- %s %s %s\n", colName, dataType, isNullable)
		}
		return nil
	}

	fmt.Fprintf(file, "-- Table: %s\n", tableName)
	fmt.Fprintf(file, "%s\n", createStmt)
	return nil
}

func backupTableData(db *sql.DB, tableName, filename string) (int, error) {
	// Get row count first
	var count int
	err := db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)).Scan(&count)
	if err != nil {
		return 0, err
	}

	if count == 0 {
		// Create empty file to indicate empty table
		file, _ := os.Create(filename)
		file.Close()
		return 0, nil
	}

	file, err := os.Create(filename)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	// Get all data
	rows, err := db.Query(fmt.Sprintf("SELECT * FROM %s", tableName))
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return 0, err
	}

	// Write header
	for i, col := range columns {
		if i > 0 {
			fmt.Fprint(file, ",")
		}
		fmt.Fprintf(file, `"%s"`, col)
	}
	fmt.Fprintln(file)

	// Write data
	rowCount := 0
	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	for i := range columns {
		valuePtrs[i] = &values[i]
	}

	for rows.Next() {
		err := rows.Scan(valuePtrs...)
		if err != nil {
			continue
		}

		for i, val := range values {
			if i > 0 {
				fmt.Fprint(file, ",")
			}
			
			switch v := val.(type) {
			case nil:
				fmt.Fprint(file, "NULL")
			case []byte:
				fmt.Fprintf(file, `"%s"`, string(v))
			case string:
				fmt.Fprintf(file, `"%s"`, v)
			case time.Time:
				fmt.Fprintf(file, `"%s"`, v.Format("2006-01-02 15:04:05"))
			default:
				fmt.Fprintf(file, "%v", v)
			}
		}
		fmt.Fprintln(file)
		rowCount++
	}

	return rowCount, nil
}

func createSummary(tables []string, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	fmt.Fprintf(file, "Database Backup Summary\n")
	fmt.Fprintf(file, "=======================\n")
	fmt.Fprintf(file, "Backup Date: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Fprintf(file, "Total Tables: %d\n\n", len(tables))
	fmt.Fprintf(file, "Tables Backed Up:\n")
	for _, table := range tables {
		fmt.Fprintf(file, "- %s\n", table)
	}

	return nil
}