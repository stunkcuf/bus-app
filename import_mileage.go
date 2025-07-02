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
