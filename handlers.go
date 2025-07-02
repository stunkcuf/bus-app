// handlers.go - HTTP handlers for route assignment fixes
package main

import (
    "encoding/json"
    "log"
    "net/http"
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
func importMileageHandler(w http.ResponseWriter, r *http.Request) {
    user := getUserFromSession(r)
    if user == nil || user.Role != "manager" {
        http.Redirect(w, r, "/", http.StatusFound)
        return
    }

    if r.Method == "GET" {
        // Show import page
        html := `
        <!DOCTYPE html>
        <html>
        <head>
            <title>Import Mileage Data</title>
            <link href="https://cdn.jsdelivr.net/npm/bootstrap@5.3.0/dist/css/bootstrap.min.css" rel="stylesheet">
        </head>
        <body>
            <div class="container mt-5">
                <h2>Import Mileage Report</h2>
                <div class="alert alert-info">
                    This will import data from the hardcoded Excel file into the database.
                </div>
                <form method="POST">
                    <input type="hidden" name="csrf_token" value="` + getCSRFToken(r) + `">
                    <button type="submit" class="btn btn-primary">Start Import</button>
                    <a href="/manager-dashboard" class="btn btn-secondary">Cancel</a>
                </form>
            </div>
        </body>
        </html>`
        w.Write([]byte(html))
        return
    }

    // Handle POST - do the import
    if !validateCSRF(r) {
        http.Error(w, "Invalid CSRF token", http.StatusForbidden)
        return
    }

    // Run import
    message, err := doMileageImport()
    if err != nil {
        w.Write([]byte(fmt.Sprintf(`
        <html><body>
        <h2>Import Failed</h2>
        <p>Error: %v</p>
        <a href="/manager-dashboard">Back to Dashboard</a>
        </body></html>`, err)))
        return
    }

    w.Write([]byte(fmt.Sprintf(`
    <html><body>
    <h2>Import Successful</h2>
    <p>%s</p>
    <a href="/manager-dashboard">Back to Dashboard</a>
    </body></html>`, message)))
}

// Add this function to handle the actual import
func doMileageImport() (string, error) {
    // For Railway, you'll need to either:
    // 1. Include the Excel file in your deployment
    // 2. Store it in a cloud service like S3
    // 3. Hardcode the data
    
    // For now, let's assume you'll hardcode some sample data
    records := []MileageRecord{
        {
            ReportMonth: "January",
            ReportYear: 2024,
            BusYear: 2020,
            BusMake: "Blue Bird",
            LicensePlate: "ABC123",
            BusID: "101",
            LocatedAt: "Main Depot",
            BeginningMiles: 50000,
            EndingMiles: 52500,
            TotalMiles: 2500,
        },
        // Add more records here from your Excel file
    }

    // Insert into database
    successCount := 0
    for _, record := range records {
        err := insertMileageRecord(record)
        if err != nil {
            log.Printf("Failed to insert record: %v", err)
            continue
        }
        successCount++
    }

    return fmt.Sprintf("Successfully imported %d records out of %d", successCount, len(records)), nil
}

// Add to your data.go file
func insertMileageRecord(record MileageRecord) error {
    _, err := db.Exec(`
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
            updated_at = CURRENT_TIMESTAMP`,
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
    return err
}
