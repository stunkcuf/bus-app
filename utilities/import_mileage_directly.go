package main

import (
	"database/sql"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/xuri/excelize/v2"
)

type MonthlyMileageReport struct {
	ReportMonth    string `db:"report_month"`
	ReportYear     int    `db:"report_year"`
	BusYear        int    `db:"bus_year"`
	BusMake        string `db:"bus_make"`
	LicensePlate   string `db:"license_plate"`
	BusID          string `db:"bus_id"`
	LocatedAt      string `db:"located_at"`
	BeginningMiles int    `db:"beginning_miles"`
	EndingMiles    int    `db:"ending_miles"`
	TotalMiles     int    `db:"total_miles"`
	FuelCost       float64 `db:"fuel_cost"`
	MaintenanceCost float64 `db:"maintenance_cost"`
	Notes          string `db:"notes"`
}

var db *sqlx.DB

func main() {
	// Connect to database
	dbURL := "postgresql://postgres:jTKOWGEzlprRGbkPBCgGxsgnwyLeGoDL@shortline.proxy.rlwy.net:40148/railway"
	
	var err error
	db, err = sqlx.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("Failed to connect:", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal("Failed to ping:", err)
	}
	fmt.Println("Connected to database")

	// Clear existing data
	fmt.Println("\nClearing existing mileage data...")
	_, err = db.Exec("DELETE FROM monthly_mileage_reports")
	if err != nil {
		log.Printf("Warning: Failed to clear existing data: %v", err)
	}

	// Open Excel file
	filePath := `C:\Users\mycha\Downloads\MILEAGE REPORT-2024-2025 REPORT.xlsx`
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		log.Fatal("Failed to open Excel file:", err)
	}
	defer f.Close()

	// Process each sheet
	sheets := f.GetSheetList()
	totalImported := 0

	// Skip the first sheet (slots) and process monthly data
	for _, sheetName := range sheets {
		if sheetName == "slots" {
			continue
		}

		fmt.Printf("\n=== Processing sheet: '%s' ===\n", sheetName)
		
		rows, err := f.GetRows(sheetName)
		if err != nil {
			log.Printf("Error reading sheet %s: %v", sheetName, err)
			continue
		}

		// Extract month and year from sheet name
		month, year := parseSheetName(sheetName)
		if month == "" {
			fmt.Printf("Skipping sheet %s - could not parse month/year\n", sheetName)
			continue
		}

		// Process school buses section
		imported := processSchoolBuses(rows, month, year)
		totalImported += imported
		fmt.Printf("Imported %d bus records from %s %d\n", imported, month, year)
	}

	fmt.Printf("\n=== Import Complete ===\n")
	fmt.Printf("Total records imported: %d\n", totalImported)

	// Show summary
	showSummary()
}

func parseSheetName(sheetName string) (string, int) {
	// Handle various formats: "August", "August 24", "AUG 24", "sept 24", etc.
	sheetName = strings.TrimSpace(sheetName)
	
	// Try to extract month and year
	monthMap := map[string]string{
		"aug": "August", "august": "August",
		"sep": "September", "sept": "September", "september": "September",
		"oct": "October", "october": "October",
		"nov": "November", "november": "November",
		"dec": "December", "december": "December",
		"jan": "January", "january": "January",
		"feb": "February", "february": "February",
		"mar": "March", "march": "March",
		"apr": "April", "april": "April",
		"may": "May",
		"jun": "June", "june": "June",
	}

	// Convert to lowercase for matching
	lower := strings.ToLower(sheetName)
	
	// Extract month
	month := ""
	for key, value := range monthMap {
		if strings.Contains(lower, key) {
			month = value
			break
		}
	}

	if month == "" {
		return "", 0
	}

	// Extract year (default to 2024)
	year := 2024
	
	// Look for 2-digit year (24, 25)
	re := regexp.MustCompile(`\b(\d{2})\b`)
	if matches := re.FindStringSubmatch(sheetName); len(matches) > 1 {
		if y, err := strconv.Atoi(matches[1]); err == nil {
			if y >= 20 && y <= 30 {
				year = 2000 + y
			}
		}
	}
	
	// Look for 4-digit year
	re = regexp.MustCompile(`\b(20\d{2})\b`)
	if matches := re.FindStringSubmatch(sheetName); len(matches) > 1 {
		if y, err := strconv.Atoi(matches[1]); err == nil {
			year = y
		}
	}

	return month, year
}

func processSchoolBuses(rows [][]string, month string, year int) int {
	imported := 0
	inBusSection := false
	
	for i, row := range rows {
		if len(row) == 0 {
			continue
		}

		// Check for section markers
		if len(row) > 0 && strings.Contains(strings.ToUpper(row[0]), "SCHOOL BUS") {
			inBusSection = true
			continue
		}
		
		if len(row) > 0 && strings.Contains(strings.ToUpper(row[0]), "AGENCY VEHICLE") {
			// End of bus section
			break
		}

		// Skip header rows
		if inBusSection && i > 3 && len(row) >= 8 {
			// Check if this looks like a data row
			if isDataRow(row) {
				report := parseSchoolBusRow(row, month, year)
				if report != nil && report.BusID != "" {
					if err := insertMileageReport(report); err == nil {
						imported++
					}
				}
			}
		}
	}

	return imported
}

func isDataRow(row []string) bool {
	// Check if row has bus ID (column 3)
	if len(row) > 3 && row[3] != "" {
		// Check if it's not a header
		id := strings.ToLower(row[3])
		if id != "id" && id != "" {
			return true
		}
	}
	return false
}

func parseSchoolBusRow(row []string, month string, year int) *MonthlyMileageReport {
	report := &MonthlyMileageReport{
		ReportMonth: month,
		ReportYear:  year,
	}

	// Parse bus year (column 0)
	if len(row) > 0 {
		if y := parseIntSafe(row[0]); y > 1900 && y < 2100 {
			report.BusYear = y
		}
	}

	// Parse make (column 1)
	if len(row) > 1 {
		report.BusMake = cleanText(row[1])
	}

	// Parse license plate (column 2)
	if len(row) > 2 {
		report.LicensePlate = cleanText(row[2])
	}

	// Parse bus ID (column 3)
	if len(row) > 3 {
		report.BusID = cleanText(row[3])
		// Ensure bus ID has proper format
		if report.BusID != "" && !strings.HasPrefix(report.BusID, "BUS") {
			report.BusID = fmt.Sprintf("BUS%s", report.BusID)
		}
	}

	// Parse location (column 4)
	if len(row) > 4 {
		report.LocatedAt = cleanText(row[4])
	}

	// Parse miles (columns 5, 6, 7)
	if len(row) > 5 {
		report.BeginningMiles = parseIntSafe(row[5])
	}
	if len(row) > 6 {
		report.EndingMiles = parseIntSafe(row[6])
	}
	if len(row) > 7 {
		report.TotalMiles = parseIntSafe(row[7])
		
		// Handle negative total miles
		if report.TotalMiles < 0 {
			report.TotalMiles = 0
		}
	}

	// Calculate total miles if not provided or invalid
	if report.EndingMiles > 0 && report.BeginningMiles >= 0 {
		calculated := report.EndingMiles - report.BeginningMiles
		if calculated >= 0 && (report.TotalMiles == 0 || report.TotalMiles < 0) {
			report.TotalMiles = calculated
		}
	}

	return report
}

func cleanText(s string) string {
	// Remove #REF! errors
	s = strings.ReplaceAll(s, "#REF!", "")
	// Remove strikethrough markers
	s = regexp.MustCompile(`~~(.+?)~~`).ReplaceAllString(s, "$1")
	// Clean whitespace
	s = strings.TrimSpace(s)
	s = regexp.MustCompile(`\s+`).ReplaceAllString(s, " ")
	return s
}

func parseIntSafe(s string) int {
	s = cleanText(s)
	// Remove commas
	s = strings.ReplaceAll(s, ",", "")
	// Remove decimal points
	if idx := strings.Index(s, "."); idx != -1 {
		s = s[:idx]
	}
	val, _ := strconv.Atoi(s)
	return val
}

func insertMileageReport(report *MonthlyMileageReport) error {
	_, err := db.Exec(`
		INSERT INTO monthly_mileage_reports 
		(report_month, report_year, bus_year, bus_make, license_plate, 
		 bus_id, located_at, beginning_miles, ending_miles, total_miles)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		report.ReportMonth, report.ReportYear, report.BusYear, report.BusMake,
		report.LicensePlate, report.BusID, report.LocatedAt, report.BeginningMiles,
		report.EndingMiles, report.TotalMiles)
	
	if err != nil {
		log.Printf("Error inserting %s %d - Bus %s: %v", 
			report.ReportMonth, report.ReportYear, report.BusID, err)
	}
	return err
}

func showSummary() {
	fmt.Println("\n=== Database Summary ===")
	
	var count int
	err := db.Get(&count, "SELECT COUNT(*) FROM monthly_mileage_reports")
	if err == nil {
		fmt.Printf("Total records: %d\n", count)
	}

	var totalMiles sql.NullInt64
	err = db.Get(&totalMiles, "SELECT SUM(total_miles) FROM monthly_mileage_reports WHERE total_miles > 0")
	if err == nil && totalMiles.Valid {
		fmt.Printf("Total miles across all records: %d\n", totalMiles.Int64)
	}

	// Show records by month
	fmt.Println("\nRecords by month:")
	rows, err := db.Query(`
		SELECT report_month, report_year, COUNT(*), SUM(total_miles)
		FROM monthly_mileage_reports
		GROUP BY report_month, report_year
		ORDER BY report_year, 
			CASE report_month
				WHEN 'January' THEN 1
				WHEN 'February' THEN 2
				WHEN 'March' THEN 3
				WHEN 'April' THEN 4
				WHEN 'May' THEN 5
				WHEN 'June' THEN 6
				WHEN 'July' THEN 7
				WHEN 'August' THEN 8
				WHEN 'September' THEN 9
				WHEN 'October' THEN 10
				WHEN 'November' THEN 11
				WHEN 'December' THEN 12
			END
	`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var month string
			var year, recordCount int
			var miles sql.NullInt64
			rows.Scan(&month, &year, &recordCount, &miles)
			if miles.Valid {
				fmt.Printf("  %s %d: %d records, %d total miles\n", month, year, recordCount, miles.Int64)
			} else {
				fmt.Printf("  %s %d: %d records, 0 total miles\n", month, year, recordCount)
			}
		}
	}
}