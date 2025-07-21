package main

import (
	"fmt"
	"log"
	"os"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func main() {
	// Use the Railway database URL
	dbURL := "postgresql://postgres:jTKOWGEzlprRGbkPBCgGxsgnwyLeGoDL@shortline.proxy.rlwy.net:40148/railway"

	db, err := sqlx.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("Failed to connect:", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal("Failed to ping:", err)
	}
	fmt.Println("Connected to Railway database")

	// Create output file
	file, err := os.Create("railway_backup.sql")
	if err != nil {
		log.Fatal("Failed to create export file:", err)
	}
	defer file.Close()

	// Write header
	file.WriteString("-- Railway Database Export\n")
	file.WriteString("-- Simple backup using pg_dump format\n\n")

	// List of critical tables to export
	tables := []string{
		"users",
		"buses", 
		"vehicles",
		"routes",
		"students",
		"maintenance_records",
		"route_assignments",
		"driver_logs",
		"ecse_students",
		"mileage_reports",
	}

	for _, table := range tables {
		fmt.Printf("Exporting table: %s\n", table)
		
		// Get CREATE TABLE statement using pg_dump style query
		var tableDef string
		err := db.Get(&tableDef, fmt.Sprintf(`
			SELECT 
				'CREATE TABLE IF NOT EXISTS %s (' || E'\n' ||
				array_to_string(
					array_agg(
						'    ' || column_name || ' ' ||
						CASE 
							WHEN data_type = 'character varying' THEN 
								'VARCHAR(' || COALESCE(character_maximum_length::text, '255') || ')'
							WHEN data_type = 'character' THEN 
								'CHAR(' || COALESCE(character_maximum_length::text, '1') || ')'
							WHEN data_type = 'numeric' THEN 
								'NUMERIC' || 
								CASE 
									WHEN numeric_precision IS NOT NULL THEN 
										'(' || numeric_precision || ',' || COALESCE(numeric_scale::text, '0') || ')'
									ELSE ''
								END
							WHEN data_type = 'timestamp without time zone' THEN 'TIMESTAMP'
							WHEN data_type = 'time without time zone' THEN 'TIME'
							WHEN data_type = 'text' THEN 'TEXT'
							WHEN data_type = 'integer' THEN 'INTEGER'
							WHEN data_type = 'smallint' THEN 'SMALLINT'
							WHEN data_type = 'bigint' THEN 'BIGINT'
							WHEN data_type = 'boolean' THEN 'BOOLEAN'
							WHEN data_type = 'date' THEN 'DATE'
							WHEN data_type = 'double precision' THEN 'DOUBLE PRECISION'
							WHEN data_type = 'jsonb' THEN 'JSONB'
							WHEN data_type = 'interval' THEN 'INTERVAL'
							ELSE data_type
						END ||
						CASE 
							WHEN is_nullable = 'NO' THEN ' NOT NULL'
							ELSE ''
						END ||
						CASE 
							WHEN column_default IS NOT NULL THEN ' DEFAULT ' || column_default
							ELSE ''
						END
						ORDER BY ordinal_position
					),
					',' || E'\n'
				) || E'\n);'
			FROM information_schema.columns
			WHERE table_name = '%s' AND table_schema = 'public'
		`, table, table))
		
		if err != nil {
			log.Printf("Failed to get structure for %s: %v", table, err)
			// Write a simple comment
			file.WriteString(fmt.Sprintf("\n-- Table %s (structure export failed)\n", table))
			continue
		}
		
		file.WriteString(fmt.Sprintf("\n-- Table: %s\n", table))
		file.WriteString(tableDef + "\n\n")
		
		// Export table data
		exportData(db, file, table)
	}
	
	// Add constraints and indexes
	file.WriteString("\n-- Primary Keys and Constraints\n")
	constraints := []string{
		"ALTER TABLE users ADD PRIMARY KEY (username) IF NOT EXISTS;",
		"ALTER TABLE buses ADD PRIMARY KEY (bus_id) IF NOT EXISTS;", 
		"ALTER TABLE vehicles ADD PRIMARY KEY (vehicle_id) IF NOT EXISTS;",
		"ALTER TABLE routes ADD PRIMARY KEY (route_id) IF NOT EXISTS;",
		"ALTER TABLE students ADD PRIMARY KEY (student_id) IF NOT EXISTS;",
		"ALTER TABLE ecse_students ADD PRIMARY KEY (student_id) IF NOT EXISTS;",
	}
	
	for _, constraint := range constraints {
		file.WriteString(constraint + "\n")
	}
	
	fmt.Println("\nExport completed to railway_backup.sql")
	fmt.Println("\nTo set up local PostgreSQL:")
	fmt.Println("1. Download PostgreSQL from https://www.postgresql.org/download/windows/")
	fmt.Println("2. Install with default settings")
	fmt.Println("3. Open pgAdmin or command prompt")
	fmt.Println("4. Create database: CREATE DATABASE fleet_management;")
	fmt.Println("5. Import data: psql -U postgres -d fleet_management -f railway_backup.sql")
}

func exportData(db *sqlx.DB, file *os.File, table string) {
	// Count rows
	var count int
	err := db.Get(&count, fmt.Sprintf("SELECT COUNT(*) FROM %s", table))
	if err != nil {
		log.Printf("Failed to count rows in %s: %v", table, err)
		return
	}
	
	if count == 0 {
		file.WriteString(fmt.Sprintf("-- No data in table %s\n\n", table))
		return
	}
	
	file.WriteString(fmt.Sprintf("-- Data for table %s (%d rows)\n", table, count))
	
	// For simplicity, use COPY format
	query := fmt.Sprintf("SELECT * FROM %s", table)
	
	rows, err := db.Query(query)
	if err != nil {
		log.Printf("Failed to query %s: %v", table, err)
		return
	}
	defer rows.Close()
	
	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		log.Printf("Failed to get columns for %s: %v", table, err)
		return
	}
	
	// Export rows as INSERT statements
	rowCount := 0
	for rows.Next() {
		// Create holders for values
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}
		
		err := rows.Scan(valuePtrs...)
		if err != nil {
			log.Printf("Failed to scan row in %s: %v", table, err)
			continue
		}
		
		// Format values for SQL
		valueStrings := make([]string, len(values))
		for i, v := range values {
			valueStrings[i] = formatSQLValue(v)
		}
		
		// Write INSERT statement
		file.WriteString(fmt.Sprintf("INSERT INTO %s VALUES (%s);\n", table, joinStrings(valueStrings)))
		rowCount++
		
		// Add periodic commits for large tables
		if rowCount%100 == 0 {
			file.WriteString("-- COMMIT;\n")
		}
	}
	
	file.WriteString(fmt.Sprintf("-- Exported %d rows\n\n", rowCount))
}

func formatSQLValue(v interface{}) string {
	switch v := v.(type) {
	case nil:
		return "NULL"
	case string:
		// Escape single quotes and backslashes
		escaped := v
		escaped = replaceAll(escaped, "\\", "\\\\")
		escaped = replaceAll(escaped, "'", "''")
		return fmt.Sprintf("'%s'", escaped)
	case []byte:
		// Handle byte arrays (like JSONB)
		escaped := string(v)
		escaped = replaceAll(escaped, "\\", "\\\\")
		escaped = replaceAll(escaped, "'", "''")
		return fmt.Sprintf("'%s'", escaped)
	case bool:
		if v {
			return "TRUE"
		}
		return "FALSE"
	default:
		return fmt.Sprintf("%v", v)
	}
}

func joinStrings(strs []string) string {
	result := ""
	for i, s := range strs {
		if i > 0 {
			result += ", "
		}
		result += s
	}
	return result
}

func replaceAll(s, old, new string) string {
	result := ""
	for i := 0; i < len(s); {
		if i+len(old) <= len(s) && s[i:i+len(old)] == old {
			result += new
			i += len(old)
		} else {
			result += string(s[i])
			i++
		}
	}
	return result
}