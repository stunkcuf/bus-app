package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"

	_ "github.com/lib/pq"
)

func main() {
	fmt.Println("=== HS Bus Fleet Management System Health Check ===")
	fmt.Println()

	var errors []string
	var warnings []string
	var success []string

	// 1. Check environment variables
	fmt.Println("1. Checking environment variables...")
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgresql://postgres:jTKOWGEzlprRGbkPBCgGxsgnwyLeGoDL@shortline.proxy.rlwy.net:40148/railway"
		warnings = append(warnings, "DATABASE_URL not set, using default")
		os.Setenv("DATABASE_URL", dbURL)
	}
	success = append(success, "✓ Database URL configured")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		os.Setenv("PORT", port)
	}
	success = append(success, fmt.Sprintf("✓ Port configured: %s", port))

	// 2. Test database connection
	fmt.Println("\n2. Testing database connection...")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		errors = append(errors, fmt.Sprintf("✗ Failed to open database: %v", err))
	} else {
		defer db.Close()
		
		// Test ping
		err = db.Ping()
		if err != nil {
			errors = append(errors, fmt.Sprintf("✗ Database ping failed: %v", err))
		} else {
			success = append(success, "✓ Database connection successful")
			
			// Check table counts
			tables := []string{
				"users", "buses", "students", "routes", 
				"maintenance_records", "service_records",
			}
			
			for _, table := range tables {
				var count int
				err := db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&count)
				if err != nil {
					warnings = append(warnings, fmt.Sprintf("⚠ Cannot query %s: %v", table, err))
				} else {
					success = append(success, fmt.Sprintf("✓ Table %s: %d records", table, count))
				}
			}
		}
	}

	// 3. Check file system
	fmt.Println("\n3. Checking file system...")
	
	// Check templates directory
	if _, err := os.Stat("templates"); os.IsNotExist(err) {
		errors = append(errors, "✗ templates directory not found")
	} else {
		success = append(success, "✓ templates directory exists")
	}
	
	// Check static directory
	if _, err := os.Stat("static"); os.IsNotExist(err) {
		errors = append(errors, "✗ static directory not found")
	} else {
		success = append(success, "✓ static directory exists")
	}
	
	// Check backups directory
	if _, err := os.Stat("backups"); os.IsNotExist(err) {
		warnings = append(warnings, "⚠ backups directory not found (will be created)")
		os.MkdirAll("backups", 0755)
	} else {
		success = append(success, "✓ backups directory exists")
	}

	// 4. Check if application is already running
	fmt.Println("\n4. Checking if application is running...")
	resp, err := http.Get(fmt.Sprintf("http://localhost:%s/health", port))
	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode == 200 {
			warnings = append(warnings, fmt.Sprintf("⚠ Application already running on port %s", port))
		}
	} else {
		success = append(success, fmt.Sprintf("✓ Port %s is available", port))
	}

	// 5. Check sessions file
	fmt.Println("\n5. Checking sessions file...")
	if _, err := os.Stat("sessions.json"); err == nil {
		success = append(success, "✓ sessions.json exists")
	} else {
		warnings = append(warnings, "⚠ sessions.json not found (will be created)")
	}

	// Print results
	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("HEALTH CHECK RESULTS")
	fmt.Println(strings.Repeat("=", 50))
	
	if len(success) > 0 {
		fmt.Println("\n✅ SUCCESSFUL CHECKS:")
		for _, s := range success {
			fmt.Println("  " + s)
		}
	}
	
	if len(warnings) > 0 {
		fmt.Println("\n⚠️  WARNINGS:")
		for _, w := range warnings {
			fmt.Println("  " + w)
		}
	}
	
	if len(errors) > 0 {
		fmt.Println("\n❌ ERRORS:")
		for _, e := range errors {
			fmt.Println("  " + e)
		}
		fmt.Println("\n🔧 RECOMMENDATION: Fix the errors above before starting the application")
	} else {
		fmt.Println("\n✅ System is healthy and ready to run!")
		fmt.Println("\n🚀 To start the application, run:")
		fmt.Println("   ./run.bat")
		fmt.Println("   or")
		fmt.Println("   go run .")
	}
}

// Helper to repeat strings (since strings package isn't imported)
var strings = struct {
	Repeat func(string, int) string
}{
	Repeat: func(s string, n int) string {
		result := ""
		for i := 0; i < n; i++ {
			result += s
		}
		return result
	},
}