package main

import (
    "fmt"
    "log"
    "mime/multipart"
    "regexp"
    "strconv"
    "strings"
    
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
        imported, err := processMileageSheet(f, sheetName)
        if err != nil {
            log.Printf("Error processing sheet '%s': %v", sheetName, err)
            continue
        }
        totalImported += imported
    }
    
    return totalImported, nil
}

func processMileageSheet(f *excelize.File, sheetName string) (int, error) {
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
    if isEmptyMileageRow(row) {
        return nil
    }
    
    record := &AgencyVehicleRecord{
        ReportMonth: reportMonth,
        ReportYear:  reportYear,
    }
    
    // Parse year (column 0)
    if year := parseMileageInt(row[0]); year > 1900 && year < 2100 {
        record.VehicleYear = year
    }
    
    // Parse make/model (column 1)
    if len(row) > 1 {
        record.MakeModel = cleanMileageText(row[1])
    }
    
    // Parse license plate (column 2)
    if len(row) > 2 {
        record.LicensePlate = cleanMileageText(row[2])
    }
    
    // Parse vehicle ID (column 3)
    if len(row) > 3 {
        record.VehicleID = cleanMileageText(row[3])
        if record.VehicleID == "" {
            return nil // Skip if no vehicle ID
        }
    }
    
    // Parse location (column 4)
    if len(row) > 4 {
        record.Location = cleanMileageText(row[4])
    }
    
    // Parse miles (columns 5, 6, 7)
    if len(row) > 5 {
        record.BeginningMiles = parseMileageInt(row[5])
    }
    if len(row) > 6 {
        record.EndingMiles = parseMileageInt(row[6])
    }
    if len(row) > 7 {
        record.TotalMiles = parseMileageInt(row[7])
    }
    
    // Parse status/notes from the end of the row
    if len(row) > 8 {
        statusText := strings.ToLower(cleanMileageText(row[8]))
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
            record.Notes = cleanMileageText(row[8])
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
    if isEmptyMileageRow(row) {
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
        record.BusID = cleanMileageText(row[0])
        if record.BusID == "" {
            return nil
        }
    }
    
    // Parse location/status (column 1)
    if len(row) > 1 {
        locationStatus := cleanMileageText(row[1])
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
        if year := parseMileageInt(row[i]); year > 2000 && year < 2100 {
            record.BusYear = year
        } else if strings.Contains(strings.ToUpper(row[i]), "CHEV") {
            record.BusMake = cleanMileageText(row[i])
        } else if strings.HasPrefix(strings.ToUpper(row[i]), "SC") {
            record.LicensePlate = cleanMileageText(row[i])
        }
    }
    
    // Parse miles from the last columns
    if len(row) >= 7 {
        // Try to find miles in the last 3 columns
        for i := len(row) - 3; i < len(row); i++ {
            if i >= 0 && i < len(row) {
                miles := parseMileageInt(row[i])
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
    firstCell := strings.ToUpper(cleanMileageText(row[0]))
    
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
        if count := parseMileageInt(row[i]); count > 0 {
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
func cleanMileageText(s string) string {
    // Remove strikethrough markers (~~text~~)
    s = regexp.MustCompile(`~~(.+?)~~`).ReplaceAllString(s, "$1")
    // Remove extra spaces and trim
    s = strings.TrimSpace(s)
    s = regexp.MustCompile(`\s+`).ReplaceAllString(s, " ")
    return s
}

func parseMileageInt(s string) int {
    s = cleanMileageText(s)
    // Remove commas from numbers
    s = strings.ReplaceAll(s, ",", "")
    val, _ := strconv.Atoi(s)
    return val
}

func isEmptyMileageRow(row []string) bool {
    for _, cell := range row {
        if cleanMileageText(cell) != "" && cleanMileageText(cell) != "-" {
            return false
        }
    }
    return true
}
