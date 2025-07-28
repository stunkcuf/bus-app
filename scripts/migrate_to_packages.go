package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type FileMove struct {
	Source      string
	Destination string
	Package     string
}

var fileMoves = []FileMove{
	// Database package
	{Source: "database.go", Destination: "internal/database/connection.go", Package: "database"},
	{Source: "db_pool_handlers.go", Destination: "internal/database/pool_handlers.go", Package: "database"},
	{Source: "db_pool_tuning.go", Destination: "internal/database/pool_tuning.go", Package: "database"},
	{Source: "run_migrations.go", Destination: "internal/database/migrations.go", Package: "database"},
	{Source: "backup_recovery.go", Destination: "internal/database/backup.go", Package: "database"},
	{Source: "secure_query.go", Destination: "internal/database/secure_query.go", Package: "database"},
	
	// Auth package
	{Source: "sessions.go", Destination: "internal/auth/sessions.go", Package: "auth"},
	{Source: "middleware.go", Destination: "internal/auth/middleware.go", Package: "auth"},
	{Source: "middleware_auth.go", Destination: "internal/auth/middleware_auth.go", Package: "auth"},
	{Source: "middleware_csrf.go", Destination: "internal/auth/middleware_csrf.go", Package: "auth"},
	{Source: "middleware_security.go", Destination: "internal/auth/middleware_security.go", Package: "auth"},
	{Source: "csrf.go", Destination: "internal/auth/csrf.go", Package: "auth"},
	{Source: "secure_headers.go", Destination: "internal/auth/secure_headers.go", Package: "auth"},
	
	// Models package
	{Source: "models.go", Destination: "internal/models/models.go", Package: "models"},
	
	// Utils package
	{Source: "utils.go", Destination: "internal/utils/utils.go", Package: "utils"},
	{Source: "errors.go", Destination: "internal/utils/errors.go", Package: "utils"},
	{Source: "validation.go", Destination: "internal/utils/validation.go", Package: "utils"},
	{Source: "logger.go", Destination: "internal/utils/logger.go", Package: "utils"},
	
	// Templates package
	{Source: "template_cache.go", Destination: "internal/templates/cache.go", Package: "templates"},
	{Source: "template_functions.go", Destination: "internal/templates/functions.go", Package: "templates"},
}

func main() {
	fmt.Println("🚀 Starting Go package migration...")
	
	// Create directories
	dirs := []string{
		"cmd/hs-bus",
		"internal/auth",
		"internal/database", 
		"internal/models",
		"internal/handlers",
		"internal/services",
		"internal/api",
		"internal/utils",
		"internal/templates",
	}
	
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			fmt.Printf("❌ Failed to create directory %s: %v\n", dir, err)
			return
		}
	}
	fmt.Println("✅ Created directory structure")
	
	// Move files
	successCount := 0
	failCount := 0
	
	for _, move := range fileMoves {
		if err := moveAndUpdateFile(move); err != nil {
			fmt.Printf("❌ Failed to move %s: %v\n", move.Source, err)
			failCount++
		} else {
			fmt.Printf("✅ Moved %s → %s\n", move.Source, move.Destination)
			successCount++
		}
	}
	
	fmt.Printf("\n📊 Migration Summary: %d succeeded, %d failed\n", successCount, failCount)
	
	// Create go.mod if it doesn't exist
	if _, err := os.Stat("go.mod"); os.IsNotExist(err) {
		createGoMod()
	}
	
	fmt.Println("\n📝 Next steps:")
	fmt.Println("1. Move handler files to internal/handlers/")
	fmt.Println("2. Move service files to internal/services/")
	fmt.Println("3. Update import statements")
	fmt.Println("4. Move main.go to cmd/hs-bus/")
	fmt.Println("5. Run 'go mod tidy'")
}

func moveAndUpdateFile(move FileMove) error {
	// Check if source exists
	if _, err := os.Stat(move.Source); os.IsNotExist(err) {
		return fmt.Errorf("source file does not exist")
	}
	
	// Read source file
	content, err := ioutil.ReadFile(move.Source)
	if err != nil {
		return fmt.Errorf("failed to read source: %w", err)
	}
	
	// Update package declaration
	newContent := regexp.MustCompile(`package main\b`).ReplaceAllString(
		string(content), 
		fmt.Sprintf("package %s", move.Package),
	)
	
	// Write to destination
	if err := ioutil.WriteFile(move.Destination, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to write destination: %w", err)
	}
	
	// Remove source file
	if err := os.Remove(move.Source); err != nil {
		return fmt.Errorf("failed to remove source: %w", err)
	}
	
	return nil
}

func createGoMod() {
	content := `module github.com/yourusername/hs-bus

go 1.21

require (
	github.com/gorilla/mux v1.8.0
	github.com/jmoiron/sqlx v1.3.5
	github.com/lib/pq v1.10.9
	github.com/xuri/excelize/v2 v2.7.1
	golang.org/x/crypto v0.14.0
)
`
	if err := ioutil.WriteFile("go.mod", []byte(content), 0644); err != nil {
		fmt.Printf("❌ Failed to create go.mod: %v\n", err)
	} else {
		fmt.Println("✅ Created go.mod")
	}
}