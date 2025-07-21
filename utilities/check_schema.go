package main

import (
	"bufio"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

func loadEnvFile() {
	file, err := os.Open(".env")
	if err != nil {
		return // .env file not found, skip
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			os.Setenv(parts[0], parts[1])
		}
	}
}

func main() {
	// Load .env file
	loadEnvFile()

	// Get database URL from environment
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable is not set")
	}

	// Connect to database
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Set connection parameters
	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Test connection
	err = db.Ping()
	if err != nil {
		log.Fatal("Failed to ping database:", err)
	}

	fmt.Println("Connected to database successfully!")
	fmt.Println()

	// Check buses table columns
	fmt.Println("=== BUSES TABLE COLUMNS ===")
	checkTableColumns(db, "buses")
	fmt.Println()

	// Check vehicles table columns
	fmt.Println("=== VEHICLES TABLE COLUMNS ===")
	checkTableColumns(db, "vehicles")
	fmt.Println()

	// Check if tables have any data
	fmt.Println("=== TABLE ROW COUNTS ===")
	checkRowCount(db, "buses")
	checkRowCount(db, "vehicles")
	
	// Show sample data
	fmt.Println("\n=== SAMPLE BUS DATA ===")
	showSampleBusData(db)
	
	fmt.Println("\n=== SAMPLE VEHICLE DATA ===")
	showSampleVehicleData(db)
}

func checkTableColumns(db *sql.DB, tableName string) {
	query := `
		SELECT 
			column_name, 
			data_type, 
			is_nullable,
			column_default
		FROM information_schema.columns 
		WHERE table_name = $1 
		ORDER BY ordinal_position;
	`

	rows, err := db.Query(query, tableName)
	if err != nil {
		log.Printf("Error querying columns for %s: %v", tableName, err)
		return
	}
	defer rows.Close()

	fmt.Printf("Columns in %s table:\n", tableName)
	fmt.Println("----------------------------------------")
	
	hasRows := false
	for rows.Next() {
		hasRows = true
		var columnName, dataType, isNullable string
		var columnDefault sql.NullString
		
		err := rows.Scan(&columnName, &dataType, &isNullable, &columnDefault)
		if err != nil {
			log.Printf("Error scanning row: %v", err)
			continue
		}
		
		defaultStr := "NULL"
		if columnDefault.Valid {
			defaultStr = columnDefault.String
		}
		
		fmt.Printf("%-20s | %-15s | Nullable: %-5s | Default: %s\n", 
			columnName, dataType, isNullable, defaultStr)
	}
	
	if !hasRows {
		fmt.Printf("Table '%s' not found or has no columns\n", tableName)
	}
}

func checkRowCount(db *sql.DB, tableName string) {
	var count int
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)
	
	err := db.QueryRow(query).Scan(&count)
	if err != nil {
		log.Printf("Error counting rows in %s: %v", tableName, err)
		return
	}
	
	fmt.Printf("%s table has %d rows\n", tableName, count)
}

func showSampleBusData(db *sql.DB) {
	query := "SELECT bus_id, status, model, capacity, oil_status, tire_status FROM buses LIMIT 3"
	rows, err := db.Query(query)
	if err != nil {
		log.Printf("Error querying buses: %v", err)
		return
	}
	defer rows.Close()
	
	for rows.Next() {
		var busID, status string
		var model, oilStatus, tireStatus sql.NullString
		var capacity sql.NullInt32
		
		err := rows.Scan(&busID, &status, &model, &capacity, &oilStatus, &tireStatus)
		if err != nil {
			log.Printf("Error scanning row: %v", err)
			continue
		}
		
		fmt.Printf("Bus: %s | Status: %s | Model: %s | Capacity: %d | Oil: %s | Tire: %s\n",
			busID, status, 
			getStringValue(model), getIntValue(capacity),
			getStringValue(oilStatus), getStringValue(tireStatus))
	}
}

func showSampleVehicleData(db *sql.DB) {
	query := "SELECT vehicle_id, model, year, status, oil_status, tire_status FROM vehicles LIMIT 3"
	rows, err := db.Query(query)
	if err != nil {
		log.Printf("Error querying vehicles: %v", err)
		return
	}
	defer rows.Close()
	
	for rows.Next() {
		var vehicleID string
		var model, year, status, oilStatus, tireStatus sql.NullString
		
		err := rows.Scan(&vehicleID, &model, &year, &status, &oilStatus, &tireStatus)
		if err != nil {
			log.Printf("Error scanning row: %v", err)
			continue
		}
		
		fmt.Printf("Vehicle: %s | Model: %s | Year: %s | Status: %s | Oil: %s | Tire: %s\n",
			vehicleID, getStringValue(model), getStringValue(year),
			getStringValue(status), getStringValue(oilStatus), getStringValue(tireStatus))
	}
}

func getStringValue(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return "NULL"
}

func getIntValue(ni sql.NullInt32) int {
	if ni.Valid {
		return int(ni.Int32)
	}
	return 0
}