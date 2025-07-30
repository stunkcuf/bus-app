package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	_ "github.com/lib/pq"
)

func main() {
	dbURL := "postgresql://postgres:jTKOWGEzlprRGbkPBCgGxsgnwyLeGoDL@shortline.proxy.rlwy.net:40148/railway"
	db, err := sql.Open("postgres", dbURL)
	if err \!= nil {
		log.Fatal(err)
	}
	defer db.Close()

	fmt.Println("Checking maintenance_records table schema:")
	rows, err := db.Query("SELECT column_name, data_type, is_nullable FROM information_schema.columns WHERE table_name = 'maintenance_records' ORDER BY ordinal_position")
	if err \!= nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var colName, dataType, nullable string
		rows.Scan(&colName, &dataType, &nullable)
		fmt.Printf("  %s (%s) nullable: %s\n", colName, dataType, nullable)
	}
}
