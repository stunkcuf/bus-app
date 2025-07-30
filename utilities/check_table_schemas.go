package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"

	_ "github.com/lib/pq"
	"github.com/joho/godotenv"
)

func main() {
	fmt.Println("📊 CHECKING TABLE SCHEMAS")
	fmt.Println("=" + strings.Repeat("=", 60))

	// Load environment
	godotenv.Load("../.env")

	// Connect to database
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL not set")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Tables to check
	tables := []string{
		"users",
		"students", 
		"fuel_records",
		"route_assignments",
	}

	for _, table := range tables {
		fmt.Printf("\n📋 TABLE: %s\n", table)
		fmt.Println(strings.Repeat("-", 60))
		
		// Get column information
		rows, err := db.Query(`
			SELECT column_name, data_type, is_nullable, column_default
			FROM information_schema.columns 
			WHERE table_name = $1
			ORDER BY ordinal_position
		`, table)
		
		if err != nil {
			log.Printf("Error checking table %s: %v", table, err)
			continue
		}
		
		hasRows := false
		for rows.Next() {
			hasRows = true
			var colName, dataType, isNullable string
			var colDefault sql.NullString
			
			err = rows.Scan(&colName, &dataType, &isNullable, &colDefault)
			if err != nil {
				continue
			}
			
			nullable := ""
			if isNullable == "NO" {
				nullable = " NOT NULL"
			}
			
			defaultVal := ""
			if colDefault.Valid {
				defaultVal = fmt.Sprintf(" DEFAULT %s", colDefault.String)
			}
			
			fmt.Printf("  • %-25s %s%s%s\n", colName, dataType, nullable, defaultVal)
		}
		rows.Close()
		
		if !hasRows {
			fmt.Println("  ❌ Table not found or no columns")
		}
	}
}