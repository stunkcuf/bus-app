// handlers.go - HTTP handlers for route assignment fixes
package main

import (
    "encoding/json"
    "log"
    "net/http"
    "fmt"
    "strings"
)

// NEW HANDLER: Check if driver has existing bus
func handleCheckDriverBus(w http.ResponseWriter, r *http.Request) {
    driver := r.URL.Query().Get("driver")
    if driver == "" {
        http.Error(w, "Driver parameter required", http.StatusBadRequest)
        return
    }
    
    busID, err := getDriverAssignedBus(driver)
    if err != nil {
        log.Printf("Error checking driver bus: %v", err)
        http.Error(w, "Failed to check driver bus", http.StatusInternalServerError)
        return
    }
    
    response := map[string]string{
        "bus_id": busID,
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

// UPDATED HANDLER: Handle route assignment with optional bus
func handleSaveRouteAssignment(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }
    
    var assignment RouteAssignment
    if err := json.NewDecoder(r.Body).Decode(&assignment); err != nil {
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }
    
    // Validate required fields (bus_id is now optional)
    if assignment.Driver == "" || assignment.RouteID == "" {
        http.Error(w, "Driver and route_id are required", http.StatusBadRequest)
        return
    }
    
    if err := saveRouteAssignment(assignment); err != nil {
        log.Printf("Error saving route assignment: %v", err)
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]bool{"success": true})
}
// Enhanced view mileage reports handler with support for all data types
func viewEnhancedMileageReportsHandler(w http.ResponseWriter, r *http.Request) {
    user := getUserFromSession(r)
    if user == nil || user.Role != "manager" {
        http.Redirect(w, r, "/", http.StatusFound)
        return
    }
    
    // Get query parameters
    reportType := r.URL.Query().Get("type") // agency, school_bus, program, or all
    month := r.URL.Query().Get("month")
    year := r.URL.Query().Get("year")
    vehicleID := r.URL.Query().Get("vehicle_id")
    
    // Default to showing all if no type specified
    if reportType == "" {
        reportType = "all"
    }
    
    // Load data based on type
    var agencyVehicles []AgencyVehicleRecord
    var schoolBuses []SchoolBusRecord
    var programStaff []ProgramStaffRecord
    var err error
    
    if reportType == "all" || reportType == "agency" {
        agencyVehicles, err = getAgencyVehicles(month, year, vehicleID)
        if err != nil {
            log.Printf("Error loading agency vehicles: %v", err)
        }
    }
    
    if reportType == "all" || reportType == "school_bus" {
        schoolBuses, err = getSchoolBuses(month, year, vehicleID)
        if err != nil {
            log.Printf("Error loading school buses: %v", err)
        }
    }
    
    if reportType == "all" || reportType == "program" {
        programStaff, err = getProgramStaff(month, year)
        if err != nil {
            log.Printf("Error loading program staff: %v", err)
        }
    }
    
    // Calculate statistics
    stats := calculateEnhancedStats(agencyVehicles, schoolBuses, programStaff)
    
    data := struct {
        User           *User
        AgencyVehicles []AgencyVehicleRecord
        SchoolBuses    []SchoolBusRecord
        ProgramStaff   []ProgramStaffRecord
        Stats          EnhancedMileageStats
        CSRFToken      string
        // Filter values
        FilterType     string
        FilterMonth    string
        FilterYear     string
        FilterVehicleID string
    }{
        User:            user,
        AgencyVehicles:  agencyVehicles,
        SchoolBuses:     schoolBuses,
        ProgramStaff:    programStaff,
        Stats:           stats,
        CSRFToken:       getCSRFToken(r),
        FilterType:      reportType,
        FilterMonth:     month,
        FilterYear:      year,
        FilterVehicleID: vehicleID,
    }
    
    renderTemplate(w, "view_enhanced_mileage_reports.html", data)
}

type EnhancedMileageStats struct {
    // Vehicle stats
    TotalAgencyVehicles int
    TotalSchoolBuses    int
    TotalVehicles       int
    ActiveVehicles      int
    InactiveVehicles    int
    
    // Mileage stats
    TotalMiles          int
    AgencyMiles         int
    SchoolBusMiles      int
    AverageMilesPerVehicle float64
    
    // Status breakdown
    VehiclesForSale     int
    VehiclesSold        int
    VehiclesOutOfLease  int
    SpareVehicles       int
    
    // Program stats
    TotalProgramStaff   int
    HSStaff             int
    OPKStaff            int
    EHSStaff            int
}

func calculateEnhancedStats(agency []AgencyVehicleRecord, buses []SchoolBusRecord, staff []ProgramStaffRecord) EnhancedMileageStats {
    stats := EnhancedMileageStats{
        TotalAgencyVehicles: len(agency),
        TotalSchoolBuses:    len(buses),
        TotalVehicles:       len(agency) + len(buses),
    }
    
    // Process agency vehicles
    for _, v := range agency {
        stats.AgencyMiles += v.TotalMiles
        
        switch strings.ToUpper(v.Status) {
        case "FOR SALE":
            stats.VehiclesForSale++
            stats.InactiveVehicles++
        case "SOLD":
            stats.VehiclesSold++
            stats.InactiveVehicles++
        case "OUT OF LEASE":
            stats.VehiclesOutOfLease++
            stats.InactiveVehicles++
        default:
            if v.TotalMiles > 0 {
                stats.ActiveVehicles++
            }
        }
    }
    
    // Process school buses
    for _, b := range buses {
        stats.SchoolBusMiles += b.TotalMiles
        
        if strings.ToUpper(b.Status) == "SPARE" {
            stats.SpareVehicles++
            stats.InactiveVehicles++
        } else if b.TotalMiles > 0 {
            stats.ActiveVehicles++
        }
    }
    
    // Calculate total miles and average
    stats.TotalMiles = stats.AgencyMiles + stats.SchoolBusMiles
    if stats.ActiveVehicles > 0 {
        stats.AverageMilesPerVehicle = float64(stats.TotalMiles) / float64(stats.ActiveVehicles)
    }
    
    // Process program staff
    for _, p := range staff {
        totalStaff := p.StaffCount1 + p.StaffCount2
        stats.TotalProgramStaff += totalStaff
        
        switch p.ProgramType {
        case "HS":
            stats.HSStaff += totalStaff
        case "OPK":
            stats.OPKStaff += totalStaff
        case "EHS":
            stats.EHSStaff += totalStaff
        }
    }
    
    return stats
}

// Database query functions
func getAgencyVehicles(month, year, vehicleID string) ([]AgencyVehicleRecord, error) {
    query := `
        SELECT report_month, report_year, vehicle_year, make_model, 
               license_plate, vehicle_id, location, beginning_miles, 
               ending_miles, total_miles, COALESCE(status, ''), COALESCE(notes, '')
        FROM agency_vehicles
        WHERE 1=1
    `
    args := []interface{}{}
    argCount := 0
    
    if month != "" {
        argCount++
        query += fmt.Sprintf(" AND report_month = $%d", argCount)
        args = append(args, month)
    }
    
    if year != "" {
        argCount++
        query += fmt.Sprintf(" AND report_year = $%d", argCount)
        args = append(args, year)
    }
    
    if vehicleID != "" {
        argCount++
        query += fmt.Sprintf(" AND vehicle_id = $%d", argCount)
        args = append(args, vehicleID)
    }
    
    query += " ORDER BY report_year DESC, report_month, vehicle_id"
    
    rows, err := db.Query(query, args...)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    var vehicles []AgencyVehicleRecord
    for rows.Next() {
        var v AgencyVehicleRecord
        err := rows.Scan(&v.ReportMonth, &v.ReportYear, &v.VehicleYear, 
                        &v.MakeModel, &v.LicensePlate, &v.VehicleID, 
                        &v.Location, &v.BeginningMiles, &v.EndingMiles, 
                        &v.TotalMiles, &v.Status, &v.Notes)
        if err != nil {
            log.Printf("Error scanning agency vehicle: %v", err)
            continue
        }
        vehicles = append(vehicles, v)
    }
    
    return vehicles, nil
}

func getSchoolBuses(month, year, busID string) ([]SchoolBusRecord, error) {
    query := `
        SELECT report_month, report_year, bus_year, bus_make, 
               license_plate, bus_id, location, beginning_miles, 
               ending_miles, total_miles, COALESCE(status, ''), COALESCE(notes, '')
        FROM school_buses
        WHERE 1=1
    `
    args := []interface{}{}
    argCount := 0
    
    if month != "" {
        argCount++
        query += fmt.Sprintf(" AND report_month = $%d", argCount)
        args = append(args, month)
    }
    
    if year != "" {
        argCount++
        query += fmt.Sprintf(" AND report_year = $%d", argCount)
        args = append(args, year)
    }
    
    if busID != "" {
        argCount++
        query += fmt.Sprintf(" AND bus_id = $%d", argCount)
        args = append(args, busID)
    }
    
    query += " ORDER BY report_year DESC, report_month, bus_id"
    
    rows, err := db.Query(query, args...)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    var buses []SchoolBusRecord
    for rows.Next() {
        var b SchoolBusRecord
        err := rows.Scan(&b.ReportMonth, &b.ReportYear, &b.BusYear, 
                        &b.BusMake, &b.LicensePlate, &b.BusID, 
                        &b.Location, &b.BeginningMiles, &b.EndingMiles, 
                        &b.TotalMiles, &b.Status, &b.Notes)
        if err != nil {
            log.Printf("Error scanning school bus: %v", err)
            continue
        }
        buses = append(buses, b)
    }
    
    return buses, nil
}

func getProgramStaff(month, year string) ([]ProgramStaffRecord, error) {
    query := `
        SELECT report_month, report_year, program_type, 
               staff_count_1, staff_count_2
        FROM program_staff
        WHERE 1=1
    `
    args := []interface{}{}
    argCount := 0
    
    if month != "" {
        argCount++
        query += fmt.Sprintf(" AND report_month = $%d", argCount)
        args = append(args, month)
    }
    
    if year != "" {
        argCount++
        query += fmt.Sprintf(" AND report_year = $%d", argCount)
        args = append(args, year)
    }
    
    query += " ORDER BY report_year DESC, report_month, program_type"
    
    rows, err := db.Query(query, args...)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    var staff []ProgramStaffRecord
    for rows.Next() {
        var s ProgramStaffRecord
        err := rows.Scan(&s.ReportMonth, &s.ReportYear, &s.ProgramType,
                        &s.StaffCount1, &s.StaffCount2)
        if err != nil {
            log.Printf("Error scanning program staff: %v", err)
            continue
        }
        staff = append(staff, s)
    }
    
    return staff, nil
}
