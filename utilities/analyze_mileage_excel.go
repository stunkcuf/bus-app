package main

import (
	"fmt"
	"log"
	"github.com/xuri/excelize/v2"
)

func main() {
	// Open the Excel file
	filePath := `C:\Users\mycha\Downloads\MILEAGE REPORT-2024-2025 REPORT.xlsx`
	
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		log.Fatal("Failed to open Excel file:", err)
	}
	defer f.Close()

	// Get all sheet names
	sheets := f.GetSheetList()
	fmt.Printf("Excel file has %d sheets:\n", len(sheets))
	for i, sheet := range sheets {
		fmt.Printf("  %d. %s\n", i+1, sheet)
	}
	fmt.Println()

	// Analyze each sheet
	for _, sheetName := range sheets {
		fmt.Printf("\n=== Analyzing sheet: '%s' ===\n", sheetName)
		
		rows, err := f.GetRows(sheetName)
		if err != nil {
			log.Printf("Error reading sheet %s: %v", sheetName, err)
			continue
		}

		if len(rows) == 0 {
			fmt.Println("  Sheet is empty")
			continue
		}

		fmt.Printf("  Total rows: %d\n", len(rows))
		
		// Show first 20 rows to understand structure
		fmt.Println("\n  First 20 rows:")
		for i, row := range rows {
			if i >= 20 {
				break
			}
			fmt.Printf("  Row %d: ", i+1)
			for j, cell := range row {
				if j > 10 { // Limit columns shown
					fmt.Print("...")
					break
				}
				if cell != "" {
					fmt.Printf("[%d]='%s' ", j, cell)
				}
			}
			fmt.Println()
		}

		// Look for data patterns
		fmt.Println("\n  Data sections found:")
		for i, row := range rows {
			if len(row) > 0 && row[0] != "" {
				firstCell := row[0]
				if contains(firstCell, "Agency Vehicle") || contains(firstCell, "AGENCY VEHICLE") {
					fmt.Printf("    - Agency Vehicles section at row %d\n", i+1)
				} else if contains(firstCell, "School Bus") || contains(firstCell, "SCHOOL BUS") {
					fmt.Printf("    - School Buses section at row %d\n", i+1)
				} else if contains(firstCell, "Program") || contains(firstCell, "PROGRAM") {
					fmt.Printf("    - Program section at row %d\n", i+1)
				}
			}
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (contains(s[1:], substr) || contains(s[:len(s)-1], substr)))
}