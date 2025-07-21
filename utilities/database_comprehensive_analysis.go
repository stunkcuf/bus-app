package main

import (
	"database/sql"
	"fmt"
	"log"
	"sort"
	"strings"
	_ "github.com/lib/pq"
)

type TableInfo struct {
	Name        string
	RowCount    int
	Columns     []ColumnInfo
	SampleData  []map[string]interface{}
	Issues      []string
}

type ColumnInfo struct {
	Name         string
	DataType     string
	IsNullable   bool
	DefaultValue sql.NullString
}

func main() {
	dbURL := "postgresql://postgres:jTKOWGEzlprRGbkPBCgGxsgnwyLeGoDL@shortline.proxy.rlwy.net:40148/railway"

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer db.Close()

	fmt.Println("=== COMPREHENSIVE DATABASE ANALYSIS ===")
	fmt.Println("Database: railway")
	fmt.Println("=====================================\n")

	// Get all tables
	tables, err := getAllTables(db)
	if err != nil {
		log.Fatalf("Failed to get tables: %v", err)
	}

	// Analyze each table
	tableInfos := make(map[string]*TableInfo)
	for _, table := range tables {
		info, err := analyzeTable(db, table)
		if err != nil {
			log.Printf("Error analyzing table %s: %v", table, err)
			continue
		}
		tableInfos[table] = info
	}

	// Print analysis results
	printTableSummary(tableInfos)
	printEmptyTables(tableInfos)
	printDuplicateTables(tableInfos)
	printColumnNamingIssues(tableInfos)
	printDataQualityIssues(tableInfos)
	printRelationshipAnalysis(tableInfos)
	
	// Generate recommendations
	printRecommendations(tableInfos)
}

func getAllTables(db *sql.DB) ([]string, error) {
	rows, err := db.Query(`
		SELECT table_name 
		FROM information_schema.tables 
		WHERE table_schema = 'public' 
		AND table_type = 'BASE TABLE'
		ORDER BY table_name
	`)
	if err != nil {
		return nil, err
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
	return tables, nil
}

func analyzeTable(db *sql.DB, tableName string) (*TableInfo, error) {
	info := &TableInfo{
		Name:   tableName,
		Issues: []string{},
	}

	// Get row count
	err := db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)).Scan(&info.RowCount)
	if err != nil {
		info.Issues = append(info.Issues, fmt.Sprintf("Cannot count rows: %v", err))
	}

	// Get columns
	columns, err := getTableColumns(db, tableName)
	if err != nil {
		return nil, err
	}
	info.Columns = columns

	// Get sample data if table has rows
	if info.RowCount > 0 && info.RowCount < 10000 {
		info.SampleData = getSampleData(db, tableName, 3)
	}

	// Analyze issues
	analyzeTableIssues(info)

	return info, nil
}

func getTableColumns(db *sql.DB, tableName string) ([]ColumnInfo, error) {
	rows, err := db.Query(`
		SELECT column_name, data_type, is_nullable, column_default
		FROM information_schema.columns 
		WHERE table_name = $1 
		ORDER BY ordinal_position
	`, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []ColumnInfo
	for rows.Next() {
		var col ColumnInfo
		var isNullable string
		err := rows.Scan(&col.Name, &col.DataType, &isNullable, &col.DefaultValue)
		if err != nil {
			continue
		}
		col.IsNullable = (isNullable == "YES")
		columns = append(columns, col)
	}
	return columns, nil
}

func getSampleData(db *sql.DB, tableName string, limit int) []map[string]interface{} {
	// Get column names
	rows, err := db.Query(fmt.Sprintf("SELECT * FROM %s LIMIT %d", tableName, limit))
	if err != nil {
		return nil
	}
	defer rows.Close()

	columns, _ := rows.Columns()
	var results []map[string]interface{}

	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range columns {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			continue
		}

		entry := make(map[string]interface{})
		for i, col := range columns {
			entry[col] = values[i]
		}
		results = append(results, entry)
	}
	return results
}

func analyzeTableIssues(info *TableInfo) {
	// Check for unnamed columns
	for _, col := range info.Columns {
		if strings.HasPrefix(strings.ToLower(col.Name), "unnamed") {
			info.Issues = append(info.Issues, fmt.Sprintf("Generic column name: %s", col.Name))
		}
	}

	// Check for missing primary key (assuming 'id' column)
	hasID := false
	for _, col := range info.Columns {
		if strings.ToLower(col.Name) == "id" {
			hasID = true
			break
		}
	}
	if !hasID && len(info.Columns) > 0 {
		info.Issues = append(info.Issues, "No 'id' column found")
	}

	// Check if table is empty
	if info.RowCount == 0 {
		info.Issues = append(info.Issues, "Table is empty")
	}

	// Check for tables with too many nullable columns
	nullableCount := 0
	for _, col := range info.Columns {
		if col.IsNullable {
			nullableCount++
		}
	}
	if len(info.Columns) > 0 && float64(nullableCount)/float64(len(info.Columns)) > 0.8 {
		info.Issues = append(info.Issues, fmt.Sprintf("High nullable ratio: %d/%d columns", nullableCount, len(info.Columns)))
	}
}

func printTableSummary(tables map[string]*TableInfo) {
	fmt.Println("\n=== TABLE SUMMARY ===")
	fmt.Println("Total tables:", len(tables))
	
	// Sort tables by name
	var names []string
	for name := range tables {
		names = append(names, name)
	}
	sort.Strings(names)

	fmt.Printf("\n%-40s %-10s %-10s %s\n", "Table Name", "Rows", "Columns", "Issues")
	fmt.Println(strings.Repeat("-", 80))
	
	for _, name := range names {
		info := tables[name]
		issueCount := len(info.Issues)
		fmt.Printf("%-40s %-10d %-10d %d\n", info.Name, info.RowCount, len(info.Columns), issueCount)
	}
}

func printEmptyTables(tables map[string]*TableInfo) {
	fmt.Println("\n=== EMPTY TABLES ===")
	count := 0
	for name, info := range tables {
		if info.RowCount == 0 {
			fmt.Printf("- %s (%d columns)\n", name, len(info.Columns))
			count++
		}
	}
	if count == 0 {
		fmt.Println("No empty tables found")
	} else {
		fmt.Printf("\nTotal empty tables: %d\n", count)
	}
}

func printDuplicateTables(tables map[string]*TableInfo) {
	fmt.Println("\n=== POTENTIAL DUPLICATE TABLES ===")
	
	// Group tables by similar names
	groups := make(map[string][]string)
	
	for name := range tables {
		// Extract base name (remove plurals, underscores, etc.)
		base := strings.TrimSuffix(name, "s")
		base = strings.TrimSuffix(base, "_log")
		base = strings.TrimSuffix(base, "_logs")
		base = strings.TrimSuffix(base, "_record")
		base = strings.TrimSuffix(base, "_records")
		
		groups[base] = append(groups[base], name)
	}
	
	// Print groups with multiple tables
	for base, names := range groups {
		if len(names) > 1 {
			fmt.Printf("\nPotential duplicates for '%s':\n", base)
			for _, name := range names {
				info := tables[name]
				fmt.Printf("  - %s (%d rows, %d columns)\n", name, info.RowCount, len(info.Columns))
			}
		}
	}
}

func printColumnNamingIssues(tables map[string]*TableInfo) {
	fmt.Println("\n=== COLUMN NAMING ISSUES ===")
	
	for name, info := range tables {
		var issues []string
		
		// Check for unnamed columns
		unnamedCount := 0
		for _, col := range info.Columns {
			if strings.HasPrefix(strings.ToLower(col.Name), "unnamed") {
				unnamedCount++
			}
		}
		
		if unnamedCount > 0 {
			fmt.Printf("\n%s has %d unnamed columns:\n", name, unnamedCount)
			for _, col := range info.Columns {
				if strings.HasPrefix(strings.ToLower(col.Name), "unnamed") {
					fmt.Printf("  - %s (%s)\n", col.Name, col.DataType)
				}
			}
			
			// Show sample data to understand columns
			if len(info.SampleData) > 0 {
				fmt.Println("  Sample data:")
				for i, row := range info.SampleData {
					if i >= 2 {
						break
					}
					fmt.Printf("    Row %d:\n", i+1)
					for _, col := range info.Columns {
						if strings.HasPrefix(strings.ToLower(col.Name), "unnamed") {
							val := row[col.Name]
							fmt.Printf("      %s: %v\n", col.Name, val)
						}
					}
				}
			}
		}
		
		if len(issues) > 0 {
			fmt.Printf("%s:\n", name)
			for _, issue := range issues {
				fmt.Printf("  - %s\n", issue)
			}
		}
	}
}

func printDataQualityIssues(tables map[string]*TableInfo) {
	fmt.Println("\n=== DATA QUALITY ISSUES ===")
	
	// Check for inconsistent vehicle/bus references
	vehicleTables := []string{}
	busTables := []string{}
	
	for name := range tables {
		if strings.Contains(name, "vehicle") {
			vehicleTables = append(vehicleTables, name)
		}
		if strings.Contains(name, "bus") {
			busTables = append(busTables, name)
		}
	}
	
	if len(vehicleTables) > 0 {
		fmt.Println("\nVehicle-related tables:")
		for _, t := range vehicleTables {
			info := tables[t]
			fmt.Printf("  - %s (%d rows)\n", t, info.RowCount)
		}
	}
	
	if len(busTables) > 0 {
		fmt.Println("\nBus-related tables:")
		for _, t := range busTables {
			info := tables[t]
			fmt.Printf("  - %s (%d rows)\n", t, info.RowCount)
		}
	}
}

func printRelationshipAnalysis(tables map[string]*TableInfo) {
	fmt.Println("\n=== RELATIONSHIP ANALYSIS ===")
	
	// Look for foreign key patterns
	vehicleRefs := make(map[string][]string)
	userRefs := make(map[string][]string)
	
	for tableName, info := range tables {
		for _, col := range info.Columns {
			// Check for vehicle references
			if strings.Contains(col.Name, "vehicle_id") || strings.Contains(col.Name, "bus_id") {
				vehicleRefs[tableName] = append(vehicleRefs[tableName], col.Name)
			}
			// Check for user references
			if strings.Contains(col.Name, "driver") || strings.Contains(col.Name, "username") {
				userRefs[tableName] = append(userRefs[tableName], col.Name)
			}
		}
	}
	
	if len(vehicleRefs) > 0 {
		fmt.Println("\nTables with vehicle references:")
		for table, cols := range vehicleRefs {
			fmt.Printf("  %s: %s\n", table, strings.Join(cols, ", "))
		}
	}
	
	if len(userRefs) > 0 {
		fmt.Println("\nTables with user references:")
		for table, cols := range userRefs {
			fmt.Printf("  %s: %s\n", table, strings.Join(cols, ", "))
		}
	}
}

func printRecommendations(tables map[string]*TableInfo) {
	fmt.Println("\n=== RECOMMENDATIONS ===")
	fmt.Println("\n1. DATABASE STRUCTURE:")
	
	// Check for maintenance table confusion
	maintTables := []string{}
	for name, info := range tables {
		if strings.Contains(name, "maintenance") {
			maintTables = append(maintTables, fmt.Sprintf("%s (%d rows)", name, info.RowCount))
		}
	}
	if len(maintTables) > 1 {
		fmt.Printf("   - Consolidate maintenance tables: %s\n", strings.Join(maintTables, ", "))
	}
	
	// Check for service vs maintenance confusion
	if _, hasService := tables["service_records"]; hasService {
		if _, hasMaint := tables["maintenance_records"]; hasMaint {
			fmt.Println("   - Clarify difference between 'service_records' and 'maintenance_records'")
		}
	}
	
	fmt.Println("\n2. EMPTY TABLES TO POPULATE OR REMOVE:")
	for name, info := range tables {
		if info.RowCount == 0 {
			fmt.Printf("   - %s\n", name)
		}
	}
	
	fmt.Println("\n3. COLUMN NAMING FIXES:")
	for name, info := range tables {
		for _, col := range info.Columns {
			if strings.HasPrefix(strings.ToLower(col.Name), "unnamed") {
				fmt.Printf("   - Rename generic columns in %s\n", name)
				break
			}
		}
	}
	
	fmt.Println("\n4. DATA CONSOLIDATION:")
	fmt.Println("   - Standardize vehicle identification (vehicle_id vs bus_id)")
	fmt.Println("   - Create clear distinction between buses and other vehicles")
	fmt.Println("   - Establish consistent foreign key relationships")
}