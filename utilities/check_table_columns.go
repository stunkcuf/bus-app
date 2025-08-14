package main

import (
	"fmt"
	"log"
	"os"
	_ "github.com/lib/pq"
	"github.com/jmoiron/sqlx"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgresql://postgres:jTKOWGEzlprRGbkPBCgGxsgnwyLeGoDL@shortline.proxy.rlwy.net:40148/railway"
	}

	db, err := sqlx.Connect("postgres", dbURL)
	if err != nil {
		log.Fatal("Failed to connect:", err)
	}
	defer db.Close()

	// Get students table columns
	fmt.Println("=== STUDENTS TABLE COLUMNS ===")
	var studentCols []struct {
		ColumnName string `db:"column_name"`
		DataType   string `db:"data_type"`
	}
	
	err = db.Select(&studentCols, `
		SELECT column_name, data_type 
		FROM information_schema.columns 
		WHERE table_name = 'students' 
		ORDER BY ordinal_position
	`)
	
	if err != nil {
		log.Fatal("Failed to get columns:", err)
	}
	
	for _, col := range studentCols {
		fmt.Printf("  - %s (%s)\n", col.ColumnName, col.DataType)
	}

	// Check what columns exist that might be grade-related
	fmt.Println("\nLooking for grade-like columns...")
	for _, col := range studentCols {
		if col.ColumnName == "class" || col.ColumnName == "grade_level" || col.ColumnName == "year" {
			fmt.Printf("  Found possible grade column: %s\n", col.ColumnName)
		}
	}
}