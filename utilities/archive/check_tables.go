package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	_ "github.com/lib/pq"
)

func main() {
	// Get database URL
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		// Use the connection string from the app
		dbURL = "postgres://