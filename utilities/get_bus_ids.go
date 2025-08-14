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

	var buses []struct {
		BusID string `db:"bus_id"`
		Model string `db:"model"`
	}
	
	err = db.Select(&buses, "SELECT bus_id, model FROM buses WHERE status = 'active' LIMIT 5")
	if err != nil {
		log.Fatal("Failed to get buses:", err)
	}
	
	fmt.Println("Active buses:")
	for _, bus := range buses {
		fmt.Printf("  Bus ID: %s, Model: %s\n", bus.BusID, bus.Model)
	}
}