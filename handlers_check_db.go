package main

import (
	"encoding/json"
	"log"
	"net/http"
)

// checkDatabaseHandler shows what's actually in the database
func checkDatabaseHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if db == nil {
		http.Error(w, "Database not initialized", http.StatusInternalServerError)
		return
	}

	result := make(map[string]interface{})

	// Get all tables
	var tables []string
	err := db.Select(&tables, `
		SELECT table_name 
		FROM information_schema.tables 
		WHERE table_schema = 'public' 
		AND table_type = 'BASE TABLE'
		ORDER BY table_name
	`)
	if err != nil {
		log.Printf("Error getting tables: %v", err)
	}
	result["tables"] = tables

	// Check each table for count and sample data
	tableInfo := make(map[string]interface{})
	
	// Define a whitelist of allowed tables to prevent SQL injection
	allowedTables := map[string]bool{
		"buses": true, "vehicles": true, "fleet_vehicles": true,
		"students": true, "routes": true, "route_assignments": true,
		"driver_logs": true, "bus_maintenance_logs": true,
		"vehicle_maintenance_logs": true, "monthly_mileage_reports": true,
		"ecse_students": true, "ecse_services": true, "fuel_records": true,
		"maintenance_records": true, "service_records": true, "users": true,
		"saved_reports": true, "scheduled_exports": true, "sessions": true,
	}
	
	for _, table := range tables {
		info := make(map[string]interface{})
		
		// Validate table name against whitelist
		if !allowedTables[table] {
			info["error"] = "Table not in whitelist"
			tableInfo[table] = info
			continue
		}
		
		// Get count using parameterized query
		// Since we can't use placeholders for table names, we use the whitelist approach
		var count int
		query := "SELECT COUNT(*) FROM " + table
		err := db.QueryRow(query).Scan(&count)
		if err != nil {
			info["error"] = err.Error()
		} else {
			info["count"] = count
		}
		
		// Get columns
		var columns []struct {
			ColumnName string `db:"column_name"`
			DataType   string `db:"data_type"`
		}
		err = db.Select(&columns, `
			SELECT column_name, data_type 
			FROM information_schema.columns 
			WHERE table_name = $1 
			ORDER BY ordinal_position
		`, table)
		if err != nil {
			log.Printf("Error getting columns for %s: %v", table, err)
		} else {
			info["columns"] = columns
		}
		
		tableInfo[table] = info
	}
	
	result["table_info"] = tableInfo

	// Special check for ECSE related tables
	ecseTablesQuery := `
		SELECT table_name, 
		       (SELECT COUNT(*) FROM information_schema.tables t2 WHERE t2.table_name = t1.table_name) as count
		FROM information_schema.tables t1
		WHERE table_schema = 'public' 
		AND table_name LIKE '%ecse%'
		ORDER BY table_name
	`
	
	var ecseTables []struct {
		TableName string `db:"table_name"`
		Count     int    `db:"count"`
	}
	err = db.Select(&ecseTables, ecseTablesQuery)
	if err != nil {
		log.Printf("Error checking ECSE tables: %v", err)
	}
	result["ecse_tables"] = ecseTables

	// Check for any ECSE data in main tables
	var studentCount int
	err = db.QueryRow(`
		SELECT COUNT(*) 
		FROM students 
		WHERE student_id LIKE 'ECSE%' 
		OR notes LIKE '%ECSE%'
		OR notes LIKE '%special%'
	`).Scan(&studentCount)
	if err != nil {
		log.Printf("Error checking students for ECSE: %v", err)
	} else {
		result["ecse_in_students_table"] = studentCount
	}

	// Return JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}