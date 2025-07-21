package main

import (
	"log"
	"os"
)

// SetupLocalMode configures the application to use Railway directly but with better debugging
func SetupLocalMode() {
	// Use Railway database but with enhanced logging
	os.Setenv("DATABASE_URL", "postgresql://postgres:jTKOWGEzlprRGbkPBCgGxsgnwyLeGoDL@shortline.proxy.rlwy.net:40148/railway")
	os.Setenv("APP_ENV", "development")
	os.Setenv("DEBUG", "true")
	
	log.Println("Running in LOCAL DEBUG mode - using Railway database with enhanced logging")
}

// Add this to your main.go at the beginning of main():
// SetupLocalMode()