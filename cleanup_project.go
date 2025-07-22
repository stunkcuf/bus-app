package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	fmt.Println("🧹 Cleaning up project folder...")
	fmt.Println("================================")
	
	// Define what to clean
	toDelete := []string{
		// Test and debug files
		"check_all_pages.go",
		"direct_test.go",
		"simple_debug.go",
		"simple_status.go",
		"fix_all_templates.go",
		"use_local_mode.go",
		
		// Temporary cookie files
		"cookie.txt",
		"cookies.txt",
		"cookies2.txt",
		"new_cookies.txt",
		"test_cookies.txt",
		
		// Bat files that are one-time use
		"check_port.bat",
		"diagnose_fleet.bat",
		"test_fixes.bat",
		"quick_postgres_setup.bat",
		
		// HTML test files
		"maintenance_test.html",
		"mileage_page.html",
		
		// Old migration files
		"migrate_passwords.go",
		
		// Duplicate or old MD files
		"CUsersmychahs-busUI_UX_AUDIT_REPORT.md",
		"DATABASE_MIGRATION_COMPLETE.md",
		"DATABASE_MIGRATION_EXECUTION.md",
		"DATA_DISPLAY_ISSUES.md",
		"DATA_DISPLAY_ISSUES_REPORT.md",
		"DATA_FIXES_SUMMARY.md",
		"DATA_INTEGRATION_FIX_PLAN.md",
		"E2E_TEST_CONTENT_FIX.md",
		"FIXED_ISSUES_REPORT.md",
		"FIXES_COMPLETE_SUMMARY.md",
		"FIXES_SUMMARY.md",
		"FIX_PLAN.md",
		"FLEET_FIXES_SUMMARY.md",
		"FULL_PROJECT_FIX.md",
		"IMPORT_ENHANCEMENT_SUMMARY.md",
		"MAINTENANCE_FIX_SUMMARY.md",
		"MIGRATION_FINAL_REPORT.md",
		"PHASE_3_5_UX_PLAN.md",
		"PROJECT_COMPLETION_SUMMARY.md",
		"PROJECT_FIX_SUMMARY.md",
		"REMAINING_DATA_DISPLAY_ISSUES.md",
		"SYSTEM_STATUS_REPORT.md",
		"TESTING_SUMMARY.md",
		"test_content_guide.md",
		"WEEK_1_ACCESSIBILITY_IMPROVEMENTS.md",
		"WEEK_2_ACCESSIBILITY_COMPLETION.md",
		
		// Old template
		"templates/manager_dashboard_old.html",
		
		// Temporary JS/CSS files with weird names
		"CUsersmychahs-busstaticautocomplete.js",
		"CUsersmychahs-busstaticmobile_responsive.css",
		
		// Windows null file
		"nul",
	}
	
	// Create folders to organize utilities
	folders := []string{
		"docs",
		"scripts",
		"utilities/archive",
	}
	
	// Create folders
	for _, folder := range folders {
		if err := os.MkdirAll(folder, 0755); err != nil {
			fmt.Printf("❌ Failed to create %s: %v\n", folder, err)
		} else {
			fmt.Printf("✅ Created folder: %s\n", folder)
		}
	}
	
	// Delete files
	deletedCount := 0
	for _, file := range toDelete {
		if err := os.Remove(file); err != nil {
			if !os.IsNotExist(err) {
				fmt.Printf("⚠️  Failed to delete %s: %v\n", file, err)
			}
		} else {
			fmt.Printf("🗑️  Deleted: %s\n", file)
			deletedCount++
		}
	}
	
	// Move documentation to docs folder
	docsToMove := map[string]string{
		"API_DOCUMENTATION.md": "docs/API_DOCUMENTATION.md",
		"DEPLOYMENT_CHECKLIST.md": "docs/DEPLOYMENT_CHECKLIST.md",
		"ENVIRONMENT_VARIABLES.md": "docs/ENVIRONMENT_VARIABLES.md",
		"ENV_QUICK_REFERENCE.md": "docs/ENV_QUICK_REFERENCE.md",
		"SECURITY_AUDIT.md": "docs/SECURITY_AUDIT.md",
		"SECURITY_CHECKLIST.md": "docs/SECURITY_CHECKLIST.md",
		"TESTING_CHECKLIST.md": "docs/TESTING_CHECKLIST.md",
		"TESTING_GUIDE.md": "docs/TESTING_GUIDE.md",
		"DEBUGGING_GUIDE.md": "docs/DEBUGGING_GUIDE.md",
		"PERFORMANCE_IMPROVEMENTS.md": "docs/PERFORMANCE_IMPROVEMENTS.md",
		"TABLE_CONNECTION_AUDIT.md": "docs/TABLE_CONNECTION_AUDIT.md",
		"TABLE_CONNECTION_TASKS.md": "docs/TABLE_CONNECTION_TASKS.md",
		"TEST_RESULTS.md": "docs/TEST_RESULTS.md",
		"UI_UX_AUDIT_REPORT.md": "docs/UI_UX_AUDIT_REPORT.md",
		"FRONTEND_BUILD.md": "docs/FRONTEND_BUILD.md",
		"DATABASE_FIX_PLAN.md": "docs/DATABASE_FIX_PLAN.md",
		"FINAL_PROJECT_STATUS.md": "docs/FINAL_PROJECT_STATUS.md",
	}
	
	movedCount := 0
	for src, dst := range docsToMove {
		if err := os.Rename(src, dst); err != nil {
			if !os.IsNotExist(err) {
				fmt.Printf("⚠️  Failed to move %s: %v\n", src, err)
			}
		} else {
			fmt.Printf("📁 Moved %s -> %s\n", src, dst)
			movedCount++
		}
	}
	
	// Archive old utilities
	utilFiles, _ := filepath.Glob("utilities/*.go")
	archivedCount := 0
	for _, file := range utilFiles {
		base := filepath.Base(file)
		// Keep only essential utilities
		essentialUtils := []string{
			"reset_password.go",
			"test_connection.go",
			"check_database.go",
			"ensure_admin.go",
			"claude_doctor.go",
		}
		
		isEssential := false
		for _, essential := range essentialUtils {
			if base == essential {
				isEssential = true
				break
			}
		}
		
		if !isEssential && strings.HasPrefix(base, "test_") || 
		   strings.HasPrefix(base, "check_") || 
		   strings.HasPrefix(base, "debug_") ||
		   strings.HasPrefix(base, "fix_") {
			newPath := filepath.Join("utilities/archive", base)
			if err := os.Rename(file, newPath); err != nil {
				fmt.Printf("⚠️  Failed to archive %s: %v\n", file, err)
			} else {
				fmt.Printf("📦 Archived: %s\n", base)
				archivedCount++
			}
		}
	}
	
	fmt.Println("\n================================")
	fmt.Printf("✅ Cleanup complete!\n")
	fmt.Printf("   🗑️  Deleted: %d files\n", deletedCount)
	fmt.Printf("   📁 Moved to docs: %d files\n", movedCount)
	fmt.Printf("   📦 Archived utilities: %d files\n", archivedCount)
	fmt.Println("\nProject structure is now cleaner!")
}