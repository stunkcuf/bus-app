package main

import (
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
	"github.com/jmoiron/sqlx"
)

func main() {
	// Connect to database
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgresql://postgres:jTKOWGEzlprRGbkPBCgGxsgnwyLeGoDL@shortline.proxy.rlwy.net:40148/railway"
	}

	db, err := sqlx.Connect("postgres", dbURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Check driver_logs table columns
	fmt.Println("driver_logs table columns:")
	var columns []struct {
		ColumnName string `db:"column_name"`
		DataType   string `db:"data_type"`
	}
	
	err = db.Select(&columns, `
		SELECT column_name, data_type 
		FROM information_schema.columns 
		WHERE table_name = 'driver_logs' 
		ORDER BY ordinal_position
	`)
	
	if err != nil {
		log.Fatal("Failed to get columns:", err)
	}
	
	for _, col := range columns {
		fmt.Printf("  - %s (%s)\n", col.ColumnName, col.DataType)
	}

	// Check buses table columns
	fmt.Println("\nbuses table columns:")
	err = db.Select(&columns, `
		SELECT column_name, data_type 
		FROM information_schema.columns 
		WHERE table_name = 'buses' 
		ORDER BY ordinal_position
	`)
	
	if err != nil {
		log.Fatal("Failed to get columns:", err)
	}
	
	for _, col := range columns {
		fmt.Printf("  - %s (%s)\n", col.ColumnName, col.DataType)
	}

	// Get some existing buses
	fmt.Println("\nExisting buses:")
	rows, err := db.Query("SELECT * FROM buses LIMIT 3")
	if err != nil {
		log.Printf("Error querying buses: %v", err)
	} else {
		defer rows.Close()
		cols, _ := rows.Columns()
		fmt.Printf("Columns: %v\n", cols)
		
		count := 0
		for rows.Next() {
			count++
			// Just count rows for now
		}
		fmt.Printf("Found %d buses\n", count)
	}
}