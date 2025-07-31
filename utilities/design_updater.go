package utils

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

// UpdateAllPagesDesign updates specific pages with new design
func UpdateAllPagesDesign(templatesDir string) error {
	// List of key pages to update
	pagesToUpdate := []string{
		"login.html",
		"register.html",
		"students.html",
		"assign_routes.html",
		"fleet.html",
		"maintenance_records.html",
		"users.html",
		"approve_users.html",
		"ecse_dashboard.html",
		"analytics_dashboard.html",
		"messaging.html",
		"parent_dashboard.html",
		"parent_login.html",
		"manager_dashboard.html",
		"manager_reports.html",
		"manage_users.html",
		"fleet_vehicles.html",
		"fuel_records.html",
		"fuel_analytics.html",
		"service_records.html",
		"vehicle_maintenance.html",
		"monthly_mileage_reports.html",
		"budget_dashboard.html",
		"emergency_dashboard.html",
		"realtime_dashboard.html",
		"monitoring_dashboard.html",
		"progress_dashboard.html",
		"help_center.html",
		"settings.html",
		"profile.html",
	}

	// Old styles to replace
	oldBackground := `body {
      background: #0f0c29;
      background: linear-gradient(to right, #24243e, #302b63, #0f0c29);
      min-height: 100vh;`

	newBackground := `body {
      background: #1a1a2e;
      background: linear-gradient(135deg, #16213e 0%, #0f3460 50%, #533483 100%);
      min-height: 100vh;`

	oldAnimatedBg := `    /* Animated background */
    body::before {
      content: '';
      position: fixed;
      top: 0;
      left: 0;
      width: 100%;
      height: 100%;
      background-image: 
        radial-gradient(circle at 20% 80%, rgba(102, 126, 234, 0.3) 0%, transparent 50%),
        radial-gradient(circle at 80% 20%, rgba(240, 147, 251, 0.3) 0%, transparent 50%),
        radial-gradient(circle at 40% 40%, rgba(79, 172, 254, 0.2) 0%, transparent 50%);
      animation: backgroundShift 20s ease-in-out infinite;
      z-index: -1;
    }`

	newAnimatedBg := `    /* Enhanced animated background */
    body::before {
      content: '';
      position: fixed;
      top: -50%;
      right: -30%;
      width: 100%;
      height: 100%;
      background: radial-gradient(circle, rgba(147, 51, 234, 0.3) 0%, transparent 70%);
      filter: blur(100px);
      z-index: -1;
      animation: floatBackground 20s ease-in-out infinite;
    }
    
    body::after {
      content: '';
      position: fixed;
      bottom: -50%;
      left: -30%;
      width: 100%;
      height: 100%;
      background: radial-gradient(circle, rgba(59, 130, 246, 0.3) 0%, transparent 70%);
      filter: blur(100px);
      z-index: -1;
      animation: floatBackground 20s ease-in-out infinite reverse;
    }`

	oldKeyframes := `    @keyframes backgroundShift {
      0%, 100% { transform: translate(0, 0) rotate(0deg); }
      33% { transform: translate(-20px, -20px) rotate(120deg); }
      66% { transform: translate(20px, -10px) rotate(240deg); }
    }`

	newKeyframes := `    @keyframes floatBackground {
      0%, 100% { transform: translate(0, 0) scale(1); }
      33% { transform: translate(30px, -30px) scale(1.1); }
      66% { transform: translate(-20px, 20px) scale(0.9); }
    }`

	oldNavbar := `    /* Glassmorphism navigation */
    .navbar-glass {
      background: rgba(255, 255, 255, 0.1);
      backdrop-filter: blur(20px);
      border-bottom: 1px solid rgba(255, 255, 255, 0.2);
      padding: 1rem 0;
      position: sticky;
      top: 0;
      z-index: 1000;
      box-shadow: 0 8px 32px rgba(0, 0, 0, 0.3);
    }`

	newNavbar := `    /* Enhanced navigation */
    .navbar-glass {
      background: rgba(255, 255, 255, 0.1);
      backdrop-filter: blur(20px);
      -webkit-backdrop-filter: blur(20px);
      border-bottom: 1px solid rgba(255, 255, 255, 0.2);
      padding: 1.2rem 0;
      position: sticky;
      top: 0;
      z-index: 1000;
      box-shadow: 0 8px 32px rgba(0, 0, 0, 0.37);
      transition: all 0.3s ease;
    }
    
    .navbar-glass:hover {
      background: rgba(255, 255, 255, 0.12);
    }`

	// Remove floating orbs HTML
	oldOrbs := `  <!-- Floating orbs -->
  <div class="orb orb1"></div>
  <div class="orb orb2"></div>
  <div class="orb orb3"></div>`

	updatedCount := 0

	for _, page := range pagesToUpdate {
		filePath := filepath.Join(templatesDir, page)
		
		// Check if file exists
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			fmt.Printf("Skipping %s - file not found\n", page)
			continue
		}

		// Read file
		content, err := ioutil.ReadFile(filePath)
		if err != nil {
			fmt.Printf("Error reading %s: %v\n", page, err)
			continue
		}

		originalContent := string(content)
		updatedContent := originalContent

		// Apply replacements
		if strings.Contains(updatedContent, oldBackground) {
			updatedContent = strings.Replace(updatedContent, oldBackground, newBackground, 1)
		}

		if strings.Contains(updatedContent, oldAnimatedBg) {
			updatedContent = strings.Replace(updatedContent, oldAnimatedBg, newAnimatedBg, 1)
		}

		if strings.Contains(updatedContent, oldKeyframes) {
			updatedContent = strings.Replace(updatedContent, oldKeyframes, newKeyframes, 1)
		}

		if strings.Contains(updatedContent, oldNavbar) {
			updatedContent = strings.Replace(updatedContent, oldNavbar, newNavbar, 1)
		}

		// Remove floating orbs
		if strings.Contains(updatedContent, oldOrbs) {
			updatedContent = strings.Replace(updatedContent, oldOrbs, "", -1)
		}

		// Write back if changed
		if updatedContent != originalContent {
			err = ioutil.WriteFile(filePath, []byte(updatedContent), 0644)
			if err != nil {
				fmt.Printf("Error writing %s: %v\n", page, err)
			} else {
				fmt.Printf("Updated %s successfully\n", page)
				updatedCount++
			}
		} else {
			fmt.Printf("No changes needed for %s\n", page)
		}
	}

	fmt.Printf("\nTotal files updated: %d\n", updatedCount)
	return nil
}

// UpdatePagesDesignSimple updates all HTML files with basic design changes
func UpdatePagesDesignSimple(templatesDir string) error {
	updatedCount := 0
	
	// Get all HTML files
	files, err := filepath.Glob(filepath.Join(templatesDir, "*.html"))
	if err != nil {
		return fmt.Errorf("error finding files: %v", err)
	}
	
	for _, filePath := range files {
		// Read file
		content, err := ioutil.ReadFile(filePath)
		if err != nil {
			fmt.Printf("Error reading %s: %v\n", filePath, err)
			continue
		}
		
		originalContent := string(content)
		updatedContent := originalContent
		
		// Replace old background colors with new ones
		if strings.Contains(updatedContent, "background: #0f0c29;") {
			updatedContent = strings.Replace(updatedContent, "background: #0f0c29;", "background: #1a1a2e;", -1)
		}
		
		if strings.Contains(updatedContent, "linear-gradient(to right, #24243e, #302b63, #0f0c29)") {
			updatedContent = strings.Replace(updatedContent, 
				"linear-gradient(to right, #24243e, #302b63, #0f0c29)", 
				"linear-gradient(135deg, #16213e 0%, #0f3460 50%, #533483 100%)", -1)
		}
		
		// Replace old navbar glass with enhanced version
		if strings.Contains(updatedContent, "backdrop-filter: blur(20px);") && 
		   !strings.Contains(updatedContent, "-webkit-backdrop-filter: blur(20px);") {
			updatedContent = strings.Replace(updatedContent,
				"backdrop-filter: blur(20px);",
				"backdrop-filter: blur(20px);\n      -webkit-backdrop-filter: blur(20px);", -1)
		}
		
		// Write back if changed
		if updatedContent != originalContent {
			err = ioutil.WriteFile(filePath, []byte(updatedContent), 0644)
			if err != nil {
				fmt.Printf("Error writing %s: %v\n", filePath, err)
			} else {
				fmt.Printf("Updated %s\n", filepath.Base(filePath))
				updatedCount++
			}
		}
	}
	
	fmt.Printf("\nTotal files updated: %d\n", updatedCount)
	return nil
}
