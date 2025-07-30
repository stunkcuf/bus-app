package main

import (
	"fmt"
	"log"
	"os"

	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

var db *sqlx.DB

func main() {
	// Load .env file
	godotenv.Load()

	// Initialize database
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		fmt.Println("DATABASE_URL not set")
		os.Exit(1)
	}

	var err error
	db, err = sqlx.Connect("postgres", connStr)
	if err != nil {
		log.Printf("Database connection failed: %v", err)
		os.Exit(1)
	}
	defer db.Close()

	fmt.Println("=== Testing Fleet Handler Fix ===")

	// Test loadConsolidatedNonBusVehiclesFromDB
	fmt.Println("\n1. Testing loadConsolidatedNonBusVehiclesFromDB...")
	vehicles, err := loadConsolidatedNonBusVehiclesFromDB()
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
	} else {
		fmt.Printf("SUCCESS: Loaded %d consolidated vehicles\n", len(vehicles))
		for i, v := range vehicles[:min(5, len(vehicles))] {
			fmt.Printf("  Vehicle %d: ID=%s, Model=%s, Status=%s, VehicleType=%s\n", i+1, v.VehicleID, v.GetModel(), v.Status, v.VehicleType)
		}
	}

	fmt.Println("\n=== Test Complete ===")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}