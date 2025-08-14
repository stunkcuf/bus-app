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

	// Count driver logs
	var count int
	err = db.Get(&count, "SELECT COUNT(*) FROM driver_logs")
	if err != nil {
		log.Printf("Error counting logs: %v", err)
	} else {
		fmt.Printf("Total driver logs in database: %d\n", count)
	}

	// Get recent logs
	var logs []struct {
		Driver string  `db:"driver"`
		BusID  string  `db:"bus_id"`
		Date   string  `db:"date"`
		Period string  `db:"period"`
	}
	
	err = db.Select(&logs, "SELECT driver, bus_id, date, period FROM driver_logs ORDER BY id DESC LIMIT 5")
	if err != nil {
		log.Printf("Error getting logs: %v", err)
	} else {
		fmt.Println("\nRecent driver logs:")
		for _, l := range logs {
			fmt.Printf("  - Driver: %s, Bus: %s, Date: %s, Period: %s\n", 
				l.Driver, l.BusID, l.Date, l.Period)
		}
	}

	// Check for today's logs
	var todayCount int
	err = db.Get(&todayCount, "SELECT COUNT(*) FROM driver_logs WHERE date = '2025-08-13'")
	if err != nil {
		log.Printf("Error checking today's logs: %v", err)
	} else {
		fmt.Printf("\nLogs for 2025-08-13: %d\n", todayCount)
	}
}