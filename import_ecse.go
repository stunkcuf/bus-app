package main

import (
    "database/sql"
    "fmt"
    "io"
    "log"
    "mime/multipart"
    "strconv"
    "strings"
    "time"
    
    "github.com/xuri/excelize/v2"
)

// Main import function
func processECSEExcelFile(file multipart.File, filename string) (int, error) {
    f, err := excelize.OpenReader(file)
    if err != nil {
        return 0, fmt.Errorf("failed to open Excel file: %v", err)
    }
    defer f.Close()
    
    sheets := f.GetSheetList()
    log.Printf("ECSE Excel file has %d sheets: %v", len(sheets), sheets)
    
    if len(sheets) == 0 {
        return 0, fmt.Errorf("no sheets found in Excel file")
    }
    
    totalImported := 0
    
    // Process each sheet
    for _, sheetName := range sheets {
        imported, err := processECSESheet(f, sheetName)
        if err != nil {
            log.Printf("Error processing ECSE sheet '%s': %v", sheetName, err)
            continue
        }
        totalImported += imported
    }
    
    return totalImported, nil
}

func processECSESheet(f *excelize.File, sheetName string) (int, error) {
    log.Printf("\n=== Processing ECSE sheet: '%s' ===", sheetName)
    
    rows, err := f.GetRows(sheetName)
    if err != nil {
        return 0, fmt.Errorf("error reading sheet: %v", err)
    }
    
    if len(rows) < 2 {
        return 0, fmt.Errorf("sheet has no data rows")
    }
    
    // Detect header row and column mapping
    headers := rows[0]
    columnMap := detectECSEColumns(headers)
    
    log.Printf("Detected columns: %+v", columnMap)
    
    var students []ECSEStudent
    var services []ECSEService
    var assessments []ECSEAssessment
    
    // Process data rows
    for i := 1; i < len(rows); i++ {
        row := rows[i]
        if isEmptyRow(row) {
            continue
        }
        
        // Parse student data
        student := parseECSEStudentRow(row, columnMap)
        if student != nil && student.StudentID != "" {
            students = append(students, *student)
            
            // Parse related services if present
            if service := parseECSEServiceRow(row, columnMap, student.StudentID); service != nil {
                services = append(services, *service)
            }
            
            // Parse assessment data if present
            if assessment := parseECSEAssessmentRow(row, columnMap, student.StudentID); assessment != nil {
                assessments = append(assessments, *assessment)
            }
        }
    }
    
    // Insert data into database
    imported := 0
    
    if len(students) > 0 {
        count, err := insertECSEStudents(students)
        if err != nil {
            log.Printf("Error inserting ECSE students: %v", err)
        } else {
            imported += count
        }
    }
    
    if len(services) > 0 {
        count, err := insertECSEServices(services)
        if err != nil {
            log.Printf("Error inserting ECSE services: %v", err)
        } else {
            log.Printf("Inserted %d ECSE services", count)
        }
    }
    
    if len(assessments) > 0 {
        count, err := insertECSEAssessments(assessments)
        if err != nil {
            log.Printf("Error inserting ECSE assessments: %v", err)
        } else {
            log.Printf("Inserted %d ECSE assessments", count)
        }
    }
    
    log.Printf("Sheet '%s' - Imported: %d student records", sheetName, imported)
    return imported, nil
}

// Column detection for ECSE data
func detectECSEColumns(headers []string) map[string]int {
    columnMap := make(map[string]int)
    
    for i, header := range headers {
        headerLower := strings.ToLower(strings.TrimSpace(header))
        
        // Student information columns
        if strings.Contains(headerLower, "student") && strings.Contains(headerLower, "id") {
            columnMap["student_id"] = i
        } else if strings.Contains(headerLower, "first") && strings.Contains(headerLower, "name") {
            columnMap["first_name"] = i
        } else if strings.Contains(headerLower, "last") && strings.Contains(headerLower, "name") {
            columnMap["last_name"] = i
        } else if strings.Contains(headerLower, "birth") || strings.Contains(headerLower, "dob") {
            columnMap["date_of_birth"] = i
        } else if strings.Contains(headerLower, "grade") {
            columnMap["grade"] = i
        } else if strings.Contains(headerLower, "enrollment") || strings.Contains(headerLower, "status") {
            columnMap["enrollment_status"] = i
        } else if strings.Contains(headerLower, "iep") {
            columnMap["iep_status"] = i
        } else if strings.Contains(headerLower, "disability") {
            columnMap["disability"] = i
        } else if strings.Contains(headerLower, "transport") {
            columnMap["transportation"] = i
        } else if strings.Contains(headerLower, "bus") && strings.Contains(headerLower, "route") {
            columnMap["bus_route"] = i
        } else if strings.Contains(headerLower, "parent") || strings.Contains(headerLower, "guardian") {
            columnMap["parent_name"] = i
        } else if strings.Contains(headerLower, "phone") {
            columnMap["phone"] = i
        } else if strings.Contains(headerLower, "email") {
            columnMap["email"] = i
        } else if strings.Contains(headerLower, "address") {
            columnMap["address"] = i
        } else if strings.Contains(headerLower, "city") {
            columnMap["city"] = i
        } else if strings.Contains(headerLower, "state") {
            columnMap["state"] = i
        } else if strings.Contains(headerLower, "zip") {
            columnMap["zip"] = i
        }
        
        // Service columns
        if strings.Contains(headerLower, "service") && strings.Contains(headerLower, "type") {
            columnMap["service_type"] = i
        } else if strings.Contains(headerLower, "frequency") {
            columnMap["frequency"] = i
        } else if strings.Contains(headerLower, "duration") || strings.Contains(headerLower, "minutes") {
            columnMap["duration"] = i
        } else if strings.Contains(headerLower, "provider") {
            columnMap["provider"] = i
        }
        
        // Assessment columns
        if strings.Contains(headerLower, "assessment") {
            columnMap["assessment"] = i
        } else if strings.Contains(headerLower, "score") {
            columnMap["score"] = i
        } else if strings.Contains(headerLower, "evaluator") {
            columnMap["evaluator"] = i
        }
    }
    
    return columnMap
}

// Parse student data from row
func parseECSEStudentRow(row []string, columnMap map[string]int) *ECSEStudent {
    student := &ECSEStudent{
        EnrollmentStatus: "Active",
        CreatedAt: time.Now(),
        UpdatedAt: time.Now(),
    }
    
    // Parse student ID
    if idx, ok := columnMap["student_id"]; ok && idx < len(row) {
        student.StudentID = cleanText(row[idx])
    }
    
    // If no student ID, try to generate one
    if student.StudentID == "" {
        return nil
    }
    
    // Parse names
    if idx, ok := columnMap["first_name"]; ok && idx < len(row) {
        student.FirstName = cleanText(row[idx])
    }
    if idx, ok := columnMap["last_name"]; ok && idx < len(row) {
        student.LastName = cleanText(row[idx])
    }
    
    // Parse date of birth
    if idx, ok := columnMap["date_of_birth"]; ok && idx < len(row) {
        student.DateOfBirth = parseDate(row[idx])
    }
    
    // Parse grade
    if idx, ok := columnMap["grade"]; ok && idx < len(row) {
        student.Grade = cleanText(row[idx])
    }
    
    // Parse enrollment status
    if idx, ok := columnMap["enrollment_status"]; ok && idx < len(row) {
        status := cleanText(row[idx])
        if status != "" {
            student.EnrollmentStatus = status
        }
    }
    
    // Parse IEP status
    if idx, ok := columnMap["iep_status"]; ok && idx < len(row) {
        student.IEPStatus = cleanText(row[idx])
    }
    
    // Parse disability
    if idx, ok := columnMap["disability"]; ok && idx < len(row) {
        student.PrimaryDisability = cleanText(row[idx])
    }
    
    // Parse transportation
    if idx, ok := columnMap["transportation"]; ok && idx < len(row) {
        trans := strings.ToLower(cleanText(row[idx]))
        student.TransportationRequired = trans == "yes" || trans == "y" || trans == "true" || trans == "1"
    }
    
    // Parse bus route
    if idx, ok := columnMap["bus_route"]; ok && idx < len(row) {
        student.BusRoute = cleanText(row[idx])
    }
    
    // Parse contact information
    if idx, ok := columnMap["parent_name"]; ok && idx < len(row) {
        student.ParentName = cleanText(row[idx])
    }
    if idx, ok := columnMap["phone"]; ok && idx < len(row) {
        student.ParentPhone = cleanText(row[idx])
    }
    if idx, ok := columnMap["email"]; ok && idx < len(row) {
        student.ParentEmail = cleanText(row[idx])
    }
    
    // Parse address
    if idx, ok := columnMap["address"]; ok && idx < len(row) {
        student.Address = cleanText(row[idx])
    }
    if idx, ok := columnMap["city"]; ok && idx < len(row) {
        student.City = cleanText(row[idx])
    }
    if idx, ok := columnMap["state"]; ok && idx < len(row) {
        student.State = cleanText(row[idx])
    }
    if idx, ok := columnMap["zip"]; ok && idx < len(row) {
        student.ZipCode = cleanText(row[idx])
    }
    
    return student
}

// Parse service data from row
func parseECSEServiceRow(row []string, columnMap map[string]int, studentID string) *ECSEService {
    if idx, ok := columnMap["service_type"]; ok && idx < len(row) {
        serviceType := cleanText(row[idx])
        if serviceType == "" {
            return nil
        }
        
        service := &ECSEService{
            StudentID: studentID,
            ServiceType: serviceType,
            CreatedAt: time.Now(),
        }
        
        if idx, ok := columnMap["frequency"]; ok && idx < len(row) {
            service.Frequency = cleanText(row[idx])
        }
        
        if idx, ok := columnMap["duration"]; ok && idx < len(row) {
            service.Duration = parseInt(row[idx])
        }
        
        if idx, ok := columnMap["provider"]; ok && idx < len(row) {
            service.Provider = cleanText(row[idx])
        }
        
        return service
    }
    
    return nil
}

// Parse assessment data from row
func parseECSEAssessmentRow(row []string, columnMap map[string]int, studentID string) *ECSEAssessment {
    if idx, ok := columnMap["assessment"]; ok && idx < len(row) {
        assessmentType := cleanText(row[idx])
        if assessmentType == "" {
            return nil
        }
        
        assessment := &ECSEAssessment{
            StudentID: studentID,
            AssessmentType: assessmentType,
            AssessmentDate: time.Now().Format("2006-01-02"),
            CreatedAt: time.Now(),
        }
        
        if idx, ok := columnMap["score"]; ok && idx < len(row) {
            assessment.Score = cleanText(row[idx])
        }
        
        if idx, ok := columnMap["evaluator"]; ok && idx < len(row) {
            assessment.Evaluator = cleanText(row[idx])
        }
        
        return assessment
    }
    
    return nil
}

// Database insert functions
func insertECSEStudents(students []ECSEStudent) (int, error) {
    if db == nil {
        return 0, fmt.Errorf("database not initialized")
    }
    
    count := 0
    for _, student := range students {
        _, err := db.Exec(`
            INSERT INTO ecse_students 
            (student_id, first_name, last_name, date_of_birth, grade, 
             enrollment_status, iep_status, primary_disability, service_minutes,
             transportation_required, bus_route, parent_name, parent_phone,
             parent_email, address, city, state, zip_code, notes,
             created_at, updated_at)
            VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21)
            ON CONFLICT (student_id) 
            DO UPDATE SET
                first_name = EXCLUDED.first_name,
                last_name = EXCLUDED.last_name,
                date_of_birth = EXCLUDED.date_of_birth,
                grade = EXCLUDED.grade,
                enrollment_status = EXCLUDED.enrollment_status,
                iep_status = EXCLUDED.iep_status,
                primary_disability = EXCLUDED.primary_disability,
                service_minutes = EXCLUDED.service_minutes,
                transportation_required = EXCLUDED.transportation_required,
                bus_route = EXCLUDED.bus_route,
                parent_name = EXCLUDED.parent_name,
                parent_phone = EXCLUDED.parent_phone,
                parent_email = EXCLUDED.parent_email,
                address = EXCLUDED.address,
                city = EXCLUDED.city,
                state = EXCLUDED.state,
                zip_code = EXCLUDED.zip_code,
                updated_at = CURRENT_TIMESTAMP
        `, student.StudentID, student.FirstName, student.LastName, student.DateOfBirth,
           student.Grade, student.EnrollmentStatus, student.IEPStatus, student.PrimaryDisability,
           student.ServiceMinutes, student.TransportationRequired, student.BusRoute,
           student.ParentName, student.ParentPhone, student.ParentEmail,
           student.Address, student.City, student.State, student.ZipCode, student.Notes,
           student.CreatedAt, student.UpdatedAt)
        
        if err != nil {
            log.Printf("Error inserting ECSE student %s: %v", student.StudentID, err)
        } else {
            count++
        }
    }
    
    log.Printf("Successfully inserted %d ECSE students", count)
    return count, nil
}

func insertECSEServices(services []ECSEService) (int, error) {
    if db == nil {
        return 0, fmt.Errorf("database not initialized")
    }
    
    count := 0
    for _, service := range services {
        _, err := db.Exec(`
            INSERT INTO ecse_services 
            (student_id, service_type, frequency, duration, provider,
             start_date, end_date, goals, progress, created_at)
            VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
        `, service.StudentID, service.ServiceType, service.Frequency, service.Duration,
           service.Provider, service.StartDate, service.EndDate, service.Goals,
           service.Progress, service.CreatedAt)
        
        if err != nil {
            log.Printf("Error inserting ECSE service: %v", err)
        } else {
            count++
        }
    }
    
    return count, nil
}

func insertECSEAssessments(assessments []ECSEAssessment) (int, error) {
    if db == nil {
        return 0, fmt.Errorf("database not initialized")
    }
    
    count := 0
    for _, assessment := range assessments {
        _, err := db.Exec(`
            INSERT INTO ecse_assessments 
            (student_id, assessment_type, assessment_date, score,
             evaluator, notes, next_review_date, created_at)
            VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
        `, assessment.StudentID, assessment.AssessmentType, assessment.AssessmentDate,
           assessment.Score, assessment.Evaluator, assessment.Notes,
           assessment.NextReviewDate, assessment.CreatedAt)
        
        if err != nil {
            log.Printf("Error inserting ECSE assessment: %v", err)
        } else {
            count++
        }
    }
    
    return count, nil
}

// Helper function to parse dates
func parseDate(dateStr string) string {
    dateStr = cleanText(dateStr)
    if dateStr == "" {
        return ""
    }
    
    // Try to parse common date formats
    formats := []string{
        "01/02/2006",
        "1/2/2006",
        "01-02-2006",
        "2006-01-02",
        "Jan 2, 2006",
        "January 2, 2006",
    }
    
    for _, format := range formats {
        if t, err := time.Parse(format, dateStr); err == nil {
            return t.Format("2006-01-02")
        }
    }
    
    return dateStr // Return as-is if parsing fails
}
