package main

import (
    "database/sql"
    "fmt"
    "log"
    "os"
    "strconv"
    "strings"
    
    "github.com/xuri/excelize/v2"
    _ "github.com/lib/pq"
)

// MileageRecord represents a row from the Excel file
type MileageRecord struct {
    ReportMonth    string
    ReportYear     int
    BusYear        int
    BusMake        string
    LicensePlate   string
    BusID          string
    LocatedAt      string
    BeginningMiles int
    EndingMiles    int
    TotalMiles     int
}

func main() {
    // Connect to database
    db, err := connectDB()
    if err != nil {
        log.Fatalf("Failed to connect to database: %v", err)
    }
    defer db.Close()
    
    // Read Excel file
    records, err := readExcelFile("MILEAGE REPORT20242025 REPORT.xlsx")
    if err != nil {
        log.Fatalf("Failed to read Excel file: %v", err)
    }
    
    // Insert records into database
    err = insertRecords(db, records)
    if err != nil {
        log.Fatalf("Failed to insert records: %v", err)
    }
    
    log.Printf("Successfully imported %d records", len(records))
}

func connectDB() (*sql.DB, error) {
    // Use environment variables for connection
    dbURL := os.Getenv("DATABASE_URL")
    if dbURL == "" {
        // Fallback to individual environment variables
        host := os.Getenv("PGHOST")
        port := os.Getenv("PGPORT")
        user := os.Getenv("PGUSER")
        password := os.Getenv("PGPASSWORD")
        dbname := os.Getenv("PGDATABASE")
        
        if host == "" || port == "" || user == "" || password == "" || dbname == "" {
            return nil, fmt.Errorf("database connection parameters not set")
        }
        
        dbURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=require",
            user, password, host, port, dbname)
    }
    
    return sql.Open("postgres", dbURL)
}

func readExcelFile(filename string) ([]MileageRecord, error) {
    f, err := excelize.OpenFile(filename)
    if err != nil {
        return nil, fmt.Errorf("failed to open file: %w", err)
    }
    defer f.Close()
    
    var records []MileageRecord
    
    // Get all sheet names
    sheets := f.GetSheetList()
    if len(sheets) == 0 {
        return nil, fmt.Errorf("no sheets found in Excel file")
    }
    
    // Read from the first sheet
    sheetName := sheets[0]
    rows, err := f.GetRows(sheetName)
    if err != nil {
        return nil, fmt.Errorf("failed to get rows: %w", err)
    }
    
    // Skip header row (assuming first row is headers)
    for i, row := range rows {
        if i == 0 {
            // Log headers to understand the structure
            log.Printf("Headers: %v", row)
            continue
        }
        
        // Skip empty rows
        if len(row) < 10 {
            continue
        }
        
        record := MileageRecord{}
        
        // Parse each column - adjust indices based on your Excel structure
        // Example mapping (adjust based on your actual Excel columns):
        // Column 0: Report Month
        // Column 1: Report Year
        // Column 2: Bus Year
        // Column 3: Bus Make
        // Column 4: License Plate
        // Column 5: Bus ID
        // Column 6: Located At
        // Column 7: Beginning Miles
        // Column 8: Ending Miles
        // Column 9: Total Miles
        
        if len(row) > 0 {
            record.ReportMonth = strings.TrimSpace(row[0])
        }
        if len(row) > 1 {
            record.ReportYear, _ = strconv.Atoi(strings.TrimSpace(row[1]))
        }
        if len(row) > 2 {
            record.BusYear, _ = strconv.Atoi(strings.TrimSpace(row[2]))
        }
        if len(row) > 3 {
            record.BusMake = strings.TrimSpace(row[3])
        }
        if len(row) > 4 {
            record.LicensePlate = strings.TrimSpace(row[4])
        }
        if len(row) > 5 {
            record.BusID = strings.TrimSpace(row[5])
        }
        if len(row) > 6 {
            record.LocatedAt = strings.TrimSpace(row[6])
        }
        if len(row) > 7 {
            record.BeginningMiles, _ = strconv.Atoi(strings.TrimSpace(row[7]))
        }
        if len(row) > 8 {
            record.EndingMiles, _ = strconv.Atoi(strings.TrimSpace(row[8]))
        }
        if len(row) > 9 {
            record.TotalMiles, _ = strconv.Atoi(strings.TrimSpace(row[9]))
        }
        
        // Validate required fields
        if record.ReportMonth != "" && record.ReportYear != 0 && record.BusID != "" {
            records = append(records, record)
        }
    }
    
    return records, nil
}

func insertRecords(db *sql.DB, records []MileageRecord) error {
    // Begin transaction
    tx, err := db.Begin()
    if err != nil {
        return fmt.Errorf("failed to begin transaction: %w", err)
    }
    defer tx.Rollback()
    
    // Prepare insert statement
    stmt, err := tx.Prepare(`
        INSERT INTO monthly_mileage_reports 
        (report_month, report_year, bus_year, bus_make, license_plate, 
         bus_id, located_at, beginning_miles, ending_miles, total_miles)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
        ON CONFLICT (report_month, report_year, bus_id) 
        DO UPDATE SET
            bus_year = EXCLUDED.bus_year,
            bus_make = EXCLUDED.bus_make,
            license_plate = EXCLUDED.license_plate,
            located_at = EXCLUDED.located_at,
            beginning_miles = EXCLUDED.beginning_miles,
            ending_miles = EXCLUDED.ending_miles,
            total_miles = EXCLUDED.total_miles,
            updated_at = CURRENT_TIMESTAMP
    `)
    if err != nil {
        return fmt.Errorf("failed to prepare statement: %w", err)
    }
    defer stmt.Close()
    
    // Insert each record
    for i, record := range records {
        _, err := stmt.Exec(
            record.ReportMonth,
            record.ReportYear,
            record.BusYear,
            record.BusMake,
            record.LicensePlate,
            record.BusID,
            record.LocatedAt,
            record.BeginningMiles,
            record.EndingMiles,
            record.TotalMiles,
        )
        if err != nil {
            log.Printf("Failed to insert record %d: %v", i+1, err)
            return fmt.Errorf("failed to insert record %d: %w", i+1, err)
        }
    }
    
    // Commit transaction
    if err := tx.Commit(); err != nil {
        return fmt.Errorf("failed to commit transaction: %w", err)
    }
    
    return nil
}
package main

import (
    "database/sql"
    "fmt"
    "log"
    "regexp"
    "strconv"
    "strings"
    "mime/multipart"
    
    "github.com/xuri/excelize/v2"
)

// Data structures for different report types
type AgencyVehicleRecord struct {
    ReportMonth    string
    ReportYear     int
    VehicleYear    int
    MakeModel      string
    LicensePlate   string
    VehicleID      string
    Location       string
    BeginningMiles int
    EndingMiles    int
    TotalMiles     int
    Status         string // FOR SALE, SOLD, out of lease, etc.
    Notes          string
}

type SchoolBusRecord struct {
    ReportMonth    string
    ReportYear     int
    BusYear        int
    BusMake        string
    LicensePlate   string
    BusID          string
    Location       string
    BeginningMiles int
    EndingMiles    int
    TotalMiles     int
    Status         string // SPARE, SLATED FOR, etc.
    Notes          string
}

type ProgramStaffRecord struct {
    ReportMonth  string
    ReportYear   int
    ProgramType  string // HS, OPK, EHS
    StaffCount1  int
    StaffCount2  int
}

// Enhanced Excel processing function
func processEnhancedMileageExcelFile(file multipart.File, filename string) (int, error) {
    f, err := excelize.OpenReader(file)
    if err != nil {
        return 0, fmt.Errorf("failed to open Excel file: %v", err)
    }
    defer f.Close()
    
    sheets := f.GetSheetList()
    log.Printf("Excel file has %d sheets: %v", len(sheets), sheets)
    
    if len(sheets) == 0 {
        return 0, fmt.Errorf("no sheets found in Excel file")
    }
    
    totalImported := 0
    
    // Process each sheet
    for _, sheetName := range sheets {
        imported, err := processSheet(f, sheetName)
        if err != nil {
            log.Printf("Error processing sheet '%s': %v", sheetName, err)
            continue
        }
        totalImported += imported
    }
    
    return totalImported, nil
}

func processSheet(f *excelize.File, sheetName string) (int, error) {
    log.Printf("\n=== Processing sheet: '%s' ===", sheetName)
    
    rows, err := f.GetRows(sheetName)
    if err != nil {
        return 0, fmt.Errorf("error reading sheet: %v", err)
    }
    
    if len(rows) == 0 {
        return 0, nil
    }
    
    // Extract month and year from sheet name if possible
    reportMonth := sheetName
    reportYear := 2024 // Default, can be overridden
    
    // Try to extract year from sheet name (e.g., "January 2024")
    parts := strings.Split(sheetName, " ")
    if len(parts) > 1 {
        if year, err := strconv.Atoi(parts[len(parts)-1]); err == nil && year > 2000 && year < 2100 {
            reportYear = year
            reportMonth = strings.Join(parts[:len(parts)-1], " ")
        }
    }
    
    var agencyVehicles []AgencyVehicleRecord
    var schoolBuses []SchoolBusRecord
    var programStaff []ProgramStaffRecord
    
    currentSection := ""
    headerRowIndex := -1
    
    // Process rows
    for i, row := range rows {
        if len(row) == 0 {
            continue
        }
        
        firstCell := strings.TrimSpace(row[0])
        firstCellLower := strings.ToLower(firstCell)
        
        // Detect section headers
        if strings.Contains(firstCellLower, "agency vehicle") {
            currentSection = "agency"
            headerRowIndex = -1
            log.Printf("Found Agency Vehicles section at row %d", i+1)
            continue
        } else if strings.Contains(firstCellLower, "school bus") {
            currentSection = "school_bus"
            headerRowIndex = -1
            log.Printf("Found School Buses section at row %d", i+1)
            continue
        } else if strings.Contains(firstCellLower, "program") {
            currentSection = "program"
            headerRowIndex = -1
            log.Printf("Found Programs section at row %d", i+1)
            continue
        }
        
        // Look for header row
        if headerRowIndex == -1 && isHeaderRow(row) {
            headerRowIndex = i
            log.Printf("Found header row at index %d", i)
            continue
        }
        
        // Skip if we haven't found a section or header yet
        if currentSection == "" || headerRowIndex == -1 {
            continue
        }
        
        // Process data rows based on section
        switch currentSection {
        case "agency":
            if vehicle := parseAgencyVehicleRow(row, reportMonth, reportYear); vehicle != nil {
                agencyVehicles = append(agencyVehicles, *vehicle)
            }
        case "school_bus":
            if bus := parseSchoolBusRow(row, reportMonth, reportYear); bus != nil {
                schoolBuses = append(schoolBuses, *bus)
            }
        case "program":
            if staff := parseProgramStaffRow(row, reportMonth, reportYear); staff != nil {
                programStaff = append(programStaff, *staff)
            }
        }
    }
    
    // Insert records into database
    imported := 0
    
    if len(agencyVehicles) > 0 {
        count, err := insertAgencyVehicles(agencyVehicles)
        if err != nil {
            log.Printf("Error inserting agency vehicles: %v", err)
        } else {
            imported += count
        }
    }
    
    if len(schoolBuses) > 0 {
        count, err := insertSchoolBuses(schoolBuses)
        if err != nil {
            log.Printf("Error inserting school buses: %v", err)
        } else {
            imported += count
        }
    }
    
    if len(programStaff) > 0 {
        count, err := insertProgramStaff(programStaff)
        if err != nil {
            log.Printf("Error inserting program staff: %v", err)
        } else {
            imported += count
        }
    }
    
    log.Printf("Sheet '%s' - Imported: %d records", sheetName, imported)
    return imported, nil
}

func isHeaderRow(row []string) bool {
    // Check for common header keywords
    headerKeywords := []string{"year", "make", "lic", "id", "located", "beginning", "ending", "total", "miles"}
    
    rowText := strings.ToLower(strings.Join(row, " "))
    matchCount := 0
    
    for _, keyword := range headerKeywords {
        if strings.Contains(rowText, keyword) {
            matchCount++
        }
    }
    
    return matchCount >= 3
}

func parseAgencyVehicleRow(row []string, reportMonth string, reportYear int) *AgencyVehicleRecord {
    if len(row) < 7 {
        return nil
    }
    
    // Skip empty or invalid rows
    if isEmptyRow(row) {
        return nil
    }
    
    record := &AgencyVehicleRecord{
        ReportMonth: reportMonth,
        ReportYear:  reportYear,
    }
    
    // Parse year (column 0)
    if year := parseInt(row[0]); year > 1900 && year < 2100 {
        record.VehicleYear = year
    }
    
    // Parse make/model (column 1)
    if len(row) > 1 {
        record.MakeModel = cleanText(row[1])
    }
    
    // Parse license plate (column 2)
    if len(row) > 2 {
        record.LicensePlate = cleanText(row[2])
    }
    
    // Parse vehicle ID (column 3)
    if len(row) > 3 {
        record.VehicleID = cleanText(row[3])
        if record.VehicleID == "" {
            return nil // Skip if no vehicle ID
        }
    }
    
    // Parse location (column 4)
    if len(row) > 4 {
        record.Location = cleanText(row[4])
    }
    
    // Parse miles (columns 5, 6, 7)
    if len(row) > 5 {
        record.BeginningMiles = parseInt(row[5])
    }
    if len(row) > 6 {
        record.EndingMiles = parseInt(row[6])
    }
    if len(row) > 7 {
        record.TotalMiles = parseInt(row[7])
    }
    
    // Parse status/notes from the end of the row
    if len(row) > 8 {
        statusText := strings.ToLower(cleanText(row[8]))
        if strings.Contains(statusText, "for sale") {
            record.Status = "FOR SALE"
        } else if strings.Contains(statusText, "sold") {
            record.Status = "SOLD"
        } else if strings.Contains(statusText, "out of lease") {
            record.Status = "OUT OF LEASE"
        } else if strings.Contains(statusText, "no report") {
            record.Status = "NO REPORT"
        } else if strings.Contains(statusText, "repair") {
            record.Status = "REPAIRS"
        } else {
            record.Notes = cleanText(row[8])
        }
    }
    
    log.Printf("Parsed agency vehicle: ID=%s, Status=%s, Miles=%d", 
        record.VehicleID, record.Status, record.TotalMiles)
    
    return record
}

func parseSchoolBusRow(row []string, reportMonth string, reportYear int) *SchoolBusRecord {
    if len(row) < 7 {
        return nil
    }
    
    // Skip empty rows
    if isEmptyRow(row) {
        return nil
    }
    
    record := &SchoolBusRecord{
        ReportMonth: reportMonth,
        ReportYear:  reportYear,
    }
    
    // Column mapping for school buses:
    // 0: ID, 1: Location/Status, 2-3: Miles or Year/Make info
    
    // Parse bus ID (usually first column for school buses)
    if len(row) > 0 {
        record.BusID = cleanText(row[0])
        if record.BusID == "" {
            return nil
        }
    }
    
    // Parse location/status (column 1)
    if len(row) > 1 {
        locationStatus := cleanText(row[1])
        statusLower := strings.ToLower(locationStatus)
        
        if strings.Contains(statusLower, "spare") {
            record.Status = "SPARE"
            record.Location = "SPARE"
        } else if strings.Contains(statusLower, "slated for") {
            record.Status = "SLATED FOR"
            record.Location = locationStatus
        } else if strings.Contains(statusLower, "sub for") {
            record.Status = "SUBSTITUTE"
            record.Location = locationStatus
        } else {
            record.Location = locationStatus
        }
    }
    
    // Look for year and make in subsequent columns
    for i := 2; i < len(row) && i < 5; i++ {
        if year := parseInt(row[i]); year > 2000 && year < 2100 {
            record.BusYear = year
        } else if strings.Contains(strings.ToUpper(row[i]), "CHEV") {
            record.BusMake = cleanText(row[i])
        } else if strings.HasPrefix(strings.ToUpper(row[i]), "SC") {
            record.LicensePlate = cleanText(row[i])
        }
    }
    
    // Parse miles from the last columns
    if len(row) >= 7 {
        // Try to find miles in the last 3 columns
        for i := len(row) - 3; i < len(row); i++ {
            if i >= 0 && i < len(row) {
                miles := parseInt(row[i])
                if miles > 0 {
                    if record.BeginningMiles == 0 {
                        record.BeginningMiles = miles
                    } else if record.EndingMiles == 0 {
                        record.EndingMiles = miles
                    } else {
                        record.TotalMiles = miles
                    }
                }
            }
        }
    }
    
    log.Printf("Parsed school bus: ID=%s, Status=%s, Location=%s", 
        record.BusID, record.Status, record.Location)
    
    return record
}

func parseProgramStaffRow(row []string, reportMonth string, reportYear int) *ProgramStaffRecord {
    if len(row) < 2 {
        return nil
    }
    
    // Look for program type in first column
    programType := ""
    firstCell := strings.ToUpper(cleanText(row[0]))
    
    if strings.Contains(firstCell, "HS") {
        programType = "HS"
    } else if strings.Contains(firstCell, "OPK") {
        programType = "OPK"
    } else if strings.Contains(firstCell, "EHS") {
        programType = "EHS"
    }
    
    if programType == "" {
        return nil
    }
    
    record := &ProgramStaffRecord{
        ReportMonth: reportMonth,
        ReportYear:  reportYear,
        ProgramType: programType,
    }
    
    // Look for staff counts in the row
    counts := []int{}
    for i := 1; i < len(row); i++ {
        if count := parseInt(row[i]); count > 0 {
            counts = append(counts, count)
        }
    }
    
    if len(counts) >= 1 {
        record.StaffCount1 = counts[0]
    }
    if len(counts) >= 2 {
        record.StaffCount2 = counts[1]
    }
    
    log.Printf("Parsed program staff: Type=%s, Count1=%d, Count2=%d", 
        record.ProgramType, record.StaffCount1, record.StaffCount2)
    
    return record
}

// Database insert functions
func insertAgencyVehicles(records []AgencyVehicleRecord) (int, error) {
    if db == nil {
        return 0, fmt.Errorf("database not initialized")
    }
    
    count := 0
    for _, record := range records {
        _, err := db.Exec(`
            INSERT INTO agency_vehicles 
            (report_month, report_year, vehicle_year, make_model, license_plate, 
             vehicle_id, location, beginning_miles, ending_miles, total_miles, status, notes)
            VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
            ON CONFLICT (report_month, report_year, vehicle_id) 
            DO UPDATE SET
                vehicle_year = EXCLUDED.vehicle_year,
                make_model = EXCLUDED.make_model,
                license_plate = EXCLUDED.license_plate,
                location = EXCLUDED.location,
                beginning_miles = EXCLUDED.beginning_miles,
                ending_miles = EXCLUDED.ending_miles,
                total_miles = EXCLUDED.total_miles,
                status = EXCLUDED.status,
                notes = EXCLUDED.notes,
                updated_at = CURRENT_TIMESTAMP
        `, record.ReportMonth, record.ReportYear, record.VehicleYear, record.MakeModel,
           record.LicensePlate, record.VehicleID, record.Location, record.BeginningMiles,
           record.EndingMiles, record.TotalMiles, record.Status, record.Notes)
        
        if err != nil {
            log.Printf("Error inserting agency vehicle %s: %v", record.VehicleID, err)
        } else {
            count++
        }
    }
    
    log.Printf("Successfully inserted %d agency vehicles", count)
    return count, nil
}

func insertSchoolBuses(records []SchoolBusRecord) (int, error) {
    if db == nil {
        return 0, fmt.Errorf("database not initialized")
    }
    
    count := 0
    for _, record := range records {
        _, err := db.Exec(`
            INSERT INTO school_buses 
            (report_month, report_year, bus_year, bus_make, license_plate, 
             bus_id, location, beginning_miles, ending_miles, total_miles, status, notes)
            VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
            ON CONFLICT (report_month, report_year, bus_id) 
            DO UPDATE SET
                bus_year = EXCLUDED.bus_year,
                bus_make = EXCLUDED.bus_make,
                license_plate = EXCLUDED.license_plate,
                location = EXCLUDED.location,
                beginning_miles = EXCLUDED.beginning_miles,
                ending_miles = EXCLUDED.ending_miles,
                total_miles = EXCLUDED.total_miles,
                status = EXCLUDED.status,
                notes = EXCLUDED.notes,
                updated_at = CURRENT_TIMESTAMP
        `, record.ReportMonth, record.ReportYear, record.BusYear, record.BusMake,
           record.LicensePlate, record.BusID, record.Location, record.BeginningMiles,
           record.EndingMiles, record.TotalMiles, record.Status, record.Notes)
        
        if err != nil {
            log.Printf("Error inserting school bus %s: %v", record.BusID, err)
        } else {
            count++
        }
    }
    
    log.Printf("Successfully inserted %d school buses", count)
    return count, nil
}

func insertProgramStaff(records []ProgramStaffRecord) (int, error) {
    if db == nil {
        return 0, fmt.Errorf("database not initialized")
    }
    
    count := 0
    for _, record := range records {
        _, err := db.Exec(`
            INSERT INTO program_staff 
            (report_month, report_year, program_type, staff_count_1, staff_count_2)
            VALUES ($1, $2, $3, $4, $5)
            ON CONFLICT (report_month, report_year, program_type) 
            DO UPDATE SET
                staff_count_1 = EXCLUDED.staff_count_1,
                staff_count_2 = EXCLUDED.staff_count_2,
                updated_at = CURRENT_TIMESTAMP
        `, record.ReportMonth, record.ReportYear, record.ProgramType,
           record.StaffCount1, record.StaffCount2)
        
        if err != nil {
            log.Printf("Error inserting program staff %s: %v", record.ProgramType, err)
        } else {
            count++
        }
    }
    
    log.Printf("Successfully inserted %d program staff records", count)
    return count, nil
}

// Helper functions
func cleanText(s string) string {
    // Remove strikethrough markers (~~text~~)
    s = regexp.MustCompile(`~~(.+?)~~`).ReplaceAllString(s, "$1")
    // Remove extra spaces and trim
    s = strings.TrimSpace(s)
    s = regexp.MustCompile(`\s+`).ReplaceAllString(s, " ")
    return s
}

func parseInt(s string) int {
    s = cleanText(s)
    // Remove commas from numbers
    s = strings.ReplaceAll(s, ",", "")
    val, _ := strconv.Atoi(s)
    return val
}

func isEmptyRow(row []string) bool {
    for _, cell := range row {
        if cleanText(cell) != "" && cleanText(cell) != "-" {
            return false
        }
    }
    return true
}

// Update the main import handler to use this new function
func (handler *ImportHandler) processFile(file multipart.File, filename string) (int, error) {
    return processEnhancedMileageExcelFile(file, filename)
}
