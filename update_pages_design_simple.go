package main

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
)

func main() {
	templatesDir := "templates"
	updatedCount := 0
	
	// Get all HTML files
	files, err := filepath.Glob(filepath.Join(templatesDir, "*.html"))
	if err != nil {
		fmt.Printf("Error finding files: %v\n", err)
		return
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
}