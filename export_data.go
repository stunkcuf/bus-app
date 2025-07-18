package main

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"net/http"
	"time"

	"github.com/xuri/excelize/v2"
)

// exportMileageData exports mileage data in the specified format
func exportMileageData(w http.ResponseWriter, r *http.Request, startDate, endDate string, format string) {
	// Query mileage data
	query := `
		SELECT m.vehicle_id, m.month, m.year, m.beginning_mileage, 
		       m.ending_mileage, m.total_miles, m.driver
		FROM mileage_reports m
		WHERE (m.year || '-' || LPAD(m.month::text, 2, '0') || '-01')::date 
		      BETWEEN $1::date AND $2::date
		ORDER BY m.year, m.month, m.vehicle_id
	`
	
	rows, err := db.Query(query, startDate, endDate)
	if err != nil {
		SendError(w, ErrInternal("Failed to query mileage data", err))
		return
	}
	defer rows.Close()
	
	// Collect data
	var data [][]string
	headers := []string{"Vehicle ID", "Month", "Year", "Beginning Mileage", "Ending Mileage", "Total Miles", "Driver"}
	data = append(data, headers)
	
	for rows.Next() {
		var vehicleID, driver string
		var month, year, beginMileage, endMileage, totalMiles int
		
		err := rows.Scan(&vehicleID, &month, &year, &beginMileage, &endMileage, &totalMiles, &driver)
		if err != nil {
			continue
		}
		
		data = append(data, []string{
			vehicleID,
			fmt.Sprintf("%d", month),
			fmt.Sprintf("%d", year),
			fmt.Sprintf("%d", beginMileage),
			fmt.Sprintf("%d", endMileage),
			fmt.Sprintf("%d", totalMiles),
			driver,
		})
	}
	
	// Generate file based on format
	if format == "csv" {
		exportCSV(w, "mileage_report", data)
	} else {
		exportExcel(w, "mileage_report", "Mileage Report", data)
	}
}

// exportStudentData exports student roster
func exportStudentData(w http.ResponseWriter, r *http.Request, format string) {
	// Query student data
	query := `
		SELECT s.name, s.grade, s.address, s.phone, s.guardian_name,
		       s.pickup_time, s.dropoff_time, s.active, s.driver, r.name as route_name
		FROM students s
		LEFT JOIN routes r ON s.route_id = r.id
		WHERE s.active = true
		ORDER BY s.name
	`
	
	rows, err := db.Query(query)
	if err != nil {
		SendError(w, ErrInternal("Failed to query student data", err))
		return
	}
	defer rows.Close()
	
	// Collect data
	var data [][]string
	headers := []string{"Name", "Grade", "Address", "Phone", "Guardian", "Pickup Time", "Dropoff Time", "Driver", "Route"}
	data = append(data, headers)
	
	for rows.Next() {
		var name, grade, address, phone, guardian, pickupTime, dropoffTime, driver, route string
		var active bool
		
		err := rows.Scan(&name, &grade, &address, &phone, &guardian, 
			&pickupTime, &dropoffTime, &active, &driver, &route)
		if err != nil {
			continue
		}
		
		data = append(data, []string{
			name, grade, address, phone, guardian,
			pickupTime, dropoffTime, driver, route,
		})
	}
	
	// Generate file
	if format == "csv" {
		exportCSV(w, "student_roster", data)
	} else {
		exportExcel(w, "student_roster", "Student Roster", data)
	}
}

// exportVehicleData exports vehicle fleet information
func exportVehicleData(w http.ResponseWriter, r *http.Request, format string) {
	// Query all vehicles (buses and company vehicles)
	var data [][]string
	headers := []string{"Type", "ID", "Year", "Make/Model", "License", "Status", "Current Mileage", "Last Oil Change", "Last Tire Service"}
	data = append(data, headers)
	
	// Get buses
	busQuery := `
		SELECT bus_id, model, status, current_mileage, last_oil_change, last_tire_service
		FROM buses
		ORDER BY bus_id
	`
	
	rows, err := db.Query(busQuery)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var busID, model, status string
			var mileage, oilChange, tireService int
			
			err := rows.Scan(&busID, &model, &status, &mileage, &oilChange, &tireService)
			if err != nil {
				continue
			}
			
			data = append(data, []string{
				"Bus", busID, "", model, "", status,
				fmt.Sprintf("%d", mileage),
				fmt.Sprintf("%d", oilChange),
				fmt.Sprintf("%d", tireService),
			})
		}
	}
	
	// Get company vehicles
	vehicleQuery := `
		SELECT vehicle_id, year, model, license, status, current_mileage, 
		       last_oil_change, last_tire_service
		FROM vehicles
		ORDER BY vehicle_id
	`
	
	rows, err = db.Query(vehicleQuery)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var vehicleID, model, license, status string
			var year, mileage, oilChange, tireService int
			
			err := rows.Scan(&vehicleID, &year, &model, &license, &status, 
				&mileage, &oilChange, &tireService)
			if err != nil {
				continue
			}
			
			data = append(data, []string{
				"Company", vehicleID, fmt.Sprintf("%d", year), model, license, status,
				fmt.Sprintf("%d", mileage),
				fmt.Sprintf("%d", oilChange),
				fmt.Sprintf("%d", tireService),
			})
		}
	}
	
	// Generate file
	if format == "csv" {
		exportCSV(w, "vehicle_fleet", data)
	} else {
		exportExcel(w, "vehicle_fleet", "Vehicle Fleet", data)
	}
}

// exportMaintenanceData exports maintenance records
func exportMaintenanceData(w http.ResponseWriter, r *http.Request, startDate, endDate string, format string) {
	var data [][]string
	headers := []string{"Date", "Vehicle Type", "Vehicle ID", "Type", "Description", "Mileage", "Cost", "Performed By"}
	data = append(data, headers)
	
	// Get bus maintenance
	busQuery := `
		SELECT date, bus_id, maintenance_type, description, mileage, cost, performed_by
		FROM bus_maintenance_logs
		WHERE date BETWEEN $1::date AND $2::date
		ORDER BY date DESC
	`
	
	rows, err := db.Query(busQuery, startDate, endDate)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var date, busID, mainType, description, performedBy string
			var mileage int
			var cost float64
			
			err := rows.Scan(&date, &busID, &mainType, &description, &mileage, &cost, &performedBy)
			if err != nil {
				continue
			}
			
			data = append(data, []string{
				date, "Bus", busID, mainType, description,
				fmt.Sprintf("%d", mileage),
				fmt.Sprintf("%.2f", cost),
				performedBy,
			})
		}
	}
	
	// Get vehicle maintenance
	vehicleQuery := `
		SELECT date, vehicle_id, maintenance_type, description, mileage, cost, performed_by
		FROM vehicle_maintenance_logs
		WHERE date BETWEEN $1::date AND $2::date
		ORDER BY date DESC
	`
	
	rows, err = db.Query(vehicleQuery, startDate, endDate)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var date, vehicleID, mainType, description, performedBy string
			var mileage int
			var cost float64
			
			err := rows.Scan(&date, &vehicleID, &mainType, &description, &mileage, &cost, &performedBy)
			if err != nil {
				continue
			}
			
			data = append(data, []string{
				date, "Company", vehicleID, mainType, description,
				fmt.Sprintf("%d", mileage),
				fmt.Sprintf("%.2f", cost),
				performedBy,
			})
		}
	}
	
	// Generate file
	if format == "csv" {
		exportCSV(w, "maintenance_records", data)
	} else {
		exportExcel(w, "maintenance_records", "Maintenance Records", data)
	}
}

// Helper functions for generating exports

func exportCSV(w http.ResponseWriter, filename string, data [][]string) {
	// Set headers
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s_%s.csv\"", 
		filename, time.Now().Format("20060102")))
	
	// Write CSV
	csvWriter := csv.NewWriter(w)
	for _, row := range data {
		csvWriter.Write(row)
	}
	csvWriter.Flush()
}

func exportExcel(w http.ResponseWriter, filename, sheetName string, data [][]string) {
	f := excelize.NewFile()
	f.SetSheetName("Sheet1", sheetName)
	
	// Apply styles
	headerStyle, dataStyle := createExcelStyles(f)
	
	// Write data
	for i, row := range data {
		for j, value := range row {
			cell, _ := excelize.CoordinatesToCellName(j+1, i+1)
			f.SetCellValue(sheetName, cell, value)
			
			if i == 0 {
				f.SetCellStyle(sheetName, cell, cell, headerStyle)
			} else {
				f.SetCellStyle(sheetName, cell, cell, dataStyle)
			}
		}
	}
	
	// Auto-size columns
	for i := 0; i < len(data[0]); i++ {
		col, _ := excelize.ColumnNumberToName(i + 1)
		f.SetColWidth(sheetName, col, col, 15)
	}
	
	// Set headers
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s_%s.xlsx\"", 
		filename, time.Now().Format("20060102")))
	
	// Write file
	f.Write(w)
}

// Functions for scheduled exports that return data

func generateMileageExport(format string) ([]byte, string, error) {
	// Get current month data
	now := time.Now()
	startDate := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	endDate := startDate.AddDate(0, 1, -1)
	
	// Create a buffer to write to
	buf := new(bytes.Buffer)
	
	// Create a mock response writer
	mockWriter := &mockResponseWriter{Buffer: buf}
	
	// Generate export
	exportMileageData(mockWriter, nil, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"), format)
	
	filename := fmt.Sprintf("mileage_report_%s.%s", now.Format("200601"), format)
	return buf.Bytes(), filename, nil
}

func generateStudentExport(format string) ([]byte, string, error) {
	buf := new(bytes.Buffer)
	mockWriter := &mockResponseWriter{Buffer: buf}
	
	exportStudentData(mockWriter, nil, format)
	
	filename := fmt.Sprintf("student_roster_%s.%s", time.Now().Format("20060102"), format)
	return buf.Bytes(), filename, nil
}

func generateVehicleExport(format string) ([]byte, string, error) {
	buf := new(bytes.Buffer)
	mockWriter := &mockResponseWriter{Buffer: buf}
	
	exportVehicleData(mockWriter, nil, format)
	
	filename := fmt.Sprintf("vehicle_fleet_%s.%s", time.Now().Format("20060102"), format)
	return buf.Bytes(), filename, nil
}

func generateMaintenanceExport(format string) ([]byte, string, error) {
	// Get last 30 days of data
	now := time.Now()
	startDate := now.AddDate(0, 0, -30)
	
	buf := new(bytes.Buffer)
	mockWriter := &mockResponseWriter{Buffer: buf}
	
	exportMaintenanceData(mockWriter, nil, startDate.Format("2006-01-02"), now.Format("2006-01-02"), format)
	
	filename := fmt.Sprintf("maintenance_records_%s.%s", now.Format("20060102"), format)
	return buf.Bytes(), filename, nil
}

// mockResponseWriter implements http.ResponseWriter for generating exports to buffer
type mockResponseWriter struct {
	*bytes.Buffer
	headers http.Header
}

func (m *mockResponseWriter) Header() http.Header {
	if m.headers == nil {
		m.headers = make(http.Header)
	}
	return m.headers
}

func (m *mockResponseWriter) WriteHeader(statusCode int) {}

// Start the scheduled export job in main.go
func startScheduledExportsJob() {
	go runScheduledExportsJob()
}