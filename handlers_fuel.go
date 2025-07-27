package main

import (
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"time"
)

// fuelRecordsHandler shows fuel consumption tracking
func fuelRecordsHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	// Get filter parameters
	vehicleID := r.URL.Query().Get("vehicle")
	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")

	// Load fuel records
	records, err := loadFuelRecords(vehicleID, startDate, endDate)
	if err != nil {
		log.Printf("Error loading fuel records: %v", err)
		records = []FuelRecord{}
	}

	// Calculate statistics
	totalGallons := 0.0
	totalCost := 0.0
	totalMiles := 0
	avgMPG := 0.0

	vehicleStats := make(map[string]map[string]float64)

	for _, record := range records {
		gallons := record.GetGallons()
		cost := record.GetCost()
		
		totalGallons += gallons
		totalCost += cost

		// Calculate per-vehicle stats
		if _, exists := vehicleStats[record.VehicleID]; !exists {
			vehicleStats[record.VehicleID] = map[string]float64{
				"gallons": 0,
				"cost":    0,
				"miles":   0,
			}
		}
		
		vehicleStats[record.VehicleID]["gallons"] += gallons
		vehicleStats[record.VehicleID]["cost"] += cost
		
		if record.Odometer.Valid && record.PreviousOdometer.Valid {
			miles := record.Odometer.Int32 - record.PreviousOdometer.Int32
			vehicleStats[record.VehicleID]["miles"] += float64(miles)
			totalMiles += int(miles)
		}
	}

	// Calculate average MPG
	if totalGallons > 0 && totalMiles > 0 {
		avgMPG = float64(totalMiles) / totalGallons
	}

	// Get list of vehicles for filter
	vehicles, _ := dataCache.getBuses()

	data := map[string]interface{}{
		"User":         user,
		"CSRFToken":    getSessionCSRFToken(r),
		"Records":      records,
		"TotalGallons": totalGallons,
		"TotalCost":    totalCost,
		"TotalMiles":   totalMiles,
		"AvgMPG":       avgMPG,
		"VehicleStats": vehicleStats,
		"Vehicles":     vehicles,
		"SelectedVehicle": vehicleID,
		"StartDate":    startDate,
		"EndDate":      endDate,
		"CurrentDate":  time.Now().Format("2006-01-02"),
	}

	renderTemplate(w, r, "fuel_records.html", data)
}

// addFuelRecordHandler handles adding new fuel records
func addFuelRecordHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	if r.Method == "GET" {
		// Show add form
		vehicles, _ := dataCache.getBuses()
		data := map[string]interface{}{
			"User":      user,
			"CSRFToken": getSessionCSRFToken(r),
			"Vehicles":  vehicles,
			"Date":      time.Now().Format("2006-01-02"),
		}
		renderTemplate(w, r, "add_fuel_record.html", data)
		return
	}

	if r.Method == "POST" {
		// Parse form
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Failed to parse form", http.StatusBadRequest)
			return
		}

		// Validate CSRF
		if !validateCSRF(r) {
			http.Error(w, "Invalid CSRF token", http.StatusForbidden)
			return
		}

		// Extract form data
		vehicleID := r.FormValue("vehicle_id")
		date := r.FormValue("date")
		gallons, _ := strconv.ParseFloat(r.FormValue("gallons"), 64)
		pricePerGallon, _ := strconv.ParseFloat(r.FormValue("price_per_gallon"), 64)
		totalCost := gallons * pricePerGallon
		odometer, _ := strconv.Atoi(r.FormValue("odometer"))
		location := r.FormValue("location")
		notes := r.FormValue("notes")

		// Get previous odometer reading
		var previousOdometer sql.NullInt32
		err := db.Get(&previousOdometer, `
			SELECT odometer FROM fuel_records 
			WHERE vehicle_id = $1 AND date < $2 
			ORDER BY date DESC LIMIT 1
		`, vehicleID, date)
		
		if err != nil && err != sql.ErrNoRows {
			log.Printf("Error getting previous odometer: %v", err)
		}

		// Calculate MPG if we have previous reading
		var mpg sql.NullFloat64
		if previousOdometer.Valid && odometer > int(previousOdometer.Int32) && gallons > 0 {
			miles := odometer - int(previousOdometer.Int32)
			mpg = sql.NullFloat64{
				Float64: float64(miles) / gallons,
				Valid:   true,
			}
		}

		// Insert fuel record
		_, err = db.Exec(`
			INSERT INTO fuel_records 
			(vehicle_id, date, gallons, price_per_gallon, total_cost, odometer, 
			 previous_odometer, mpg, location, notes, recorded_by)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		`, vehicleID, date, gallons, pricePerGallon, totalCost, odometer,
			previousOdometer, mpg, location, notes, user.Username)

		if err != nil {
			log.Printf("Error adding fuel record: %v", err)
			http.Error(w, "Failed to add fuel record", http.StatusInternalServerError)
			return
		}

		// Redirect to fuel records
		http.Redirect(w, r, "/fuel-records?success=true", http.StatusSeeOther)
	}
}

// fuelAnalyticsHandler shows fuel consumption analytics
func fuelAnalyticsHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	// Get date range (default to last 30 days)
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -30)

	if start := r.URL.Query().Get("start_date"); start != "" {
		if parsed, err := time.Parse("2006-01-02", start); err == nil {
			startDate = parsed
		}
	}

	if end := r.URL.Query().Get("end_date"); end != "" {
		if parsed, err := time.Parse("2006-01-02", end); err == nil {
			endDate = parsed
		}
	}

	// Load analytics data
	monthlyData, err := getFuelMonthlyAnalytics(startDate, endDate)
	if err != nil {
		log.Printf("Error loading fuel analytics: %v", err)
		monthlyData = []map[string]interface{}{}
	}

	vehicleData, err := getFuelVehicleAnalytics(startDate, endDate)
	if err != nil {
		log.Printf("Error loading vehicle analytics: %v", err)
		vehicleData = []map[string]interface{}{}
	}

	data := map[string]interface{}{
		"User":        user,
		"CSRFToken":   getSessionCSRFToken(r),
		"MonthlyData": monthlyData,
		"VehicleData": vehicleData,
		"StartDate":   startDate.Format("2006-01-02"),
		"EndDate":     endDate.Format("2006-01-02"),
	}

	renderTemplate(w, r, "fuel_analytics.html", data)
}

// Helper functions for fuel data

func loadFuelRecords(vehicleID, startDate, endDate string) ([]FuelRecord, error) {
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	query := `SELECT * FROM fuel_records WHERE 1=1`
	args := []interface{}{}

	if vehicleID != "" {
		query += " AND vehicle_id = $" + strconv.Itoa(len(args)+1)
		args = append(args, vehicleID)
	}

	if startDate != "" {
		query += " AND date >= $" + strconv.Itoa(len(args)+1)
		args = append(args, startDate)
	}

	if endDate != "" {
		query += " AND date <= $" + strconv.Itoa(len(args)+1)
		args = append(args, endDate)
	}

	query += " ORDER BY date DESC, id DESC"

	var records []FuelRecord
	err := db.Select(&records, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to load fuel records: %w", err)
	}

	return records, nil
}

func getFuelMonthlyAnalytics(startDate, endDate time.Time) ([]map[string]interface{}, error) {
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	query := `
		SELECT 
			DATE_TRUNC('month', date) as month,
			COUNT(*) as fill_ups,
			SUM(gallons) as total_gallons,
			SUM(total_cost) as total_cost,
			AVG(price_per_gallon) as avg_price,
			AVG(mpg) as avg_mpg
		FROM fuel_records
		WHERE date BETWEEN $1 AND $2
		GROUP BY DATE_TRUNC('month', date)
		ORDER BY month DESC
	`

	rows, err := db.Query(query, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var month time.Time
		var fillUps int
		var totalGallons, totalCost, avgPrice, avgMPG sql.NullFloat64

		err := rows.Scan(&month, &fillUps, &totalGallons, &totalCost, &avgPrice, &avgMPG)
		if err != nil {
			continue
		}

		results = append(results, map[string]interface{}{
			"Month":        month.Format("January 2006"),
			"FillUps":      fillUps,
			"TotalGallons": totalGallons.Float64,
			"TotalCost":    totalCost.Float64,
			"AvgPrice":     avgPrice.Float64,
			"AvgMPG":       avgMPG.Float64,
		})
	}

	return results, nil
}

func getFuelVehicleAnalytics(startDate, endDate time.Time) ([]map[string]interface{}, error) {
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	query := `
		SELECT 
			vehicle_id,
			COUNT(*) as fill_ups,
			SUM(gallons) as total_gallons,
			SUM(total_cost) as total_cost,
			AVG(mpg) as avg_mpg,
			MIN(odometer) as start_odometer,
			MAX(odometer) as end_odometer
		FROM fuel_records
		WHERE date BETWEEN $1 AND $2
		GROUP BY vehicle_id
		ORDER BY total_cost DESC
	`

	rows, err := db.Query(query, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var vehicleID string
		var fillUps int
		var totalGallons, totalCost, avgMPG sql.NullFloat64
		var startOdo, endOdo sql.NullInt32

		err := rows.Scan(&vehicleID, &fillUps, &totalGallons, &totalCost, 
			&avgMPG, &startOdo, &endOdo)
		if err != nil {
			continue
		}

		totalMiles := 0
		if startOdo.Valid && endOdo.Valid {
			totalMiles = int(endOdo.Int32 - startOdo.Int32)
		}

		results = append(results, map[string]interface{}{
			"VehicleID":    vehicleID,
			"FillUps":      fillUps,
			"TotalGallons": totalGallons.Float64,
			"TotalCost":    totalCost.Float64,
			"AvgMPG":       avgMPG.Float64,
			"TotalMiles":   totalMiles,
		})
	}

	return results, nil
}

// addSampleFuelDataHandler creates sample fuel records for demonstration
func addSampleFuelDataHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check if fuel records already exist
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM fuel_records").Scan(&count)
	if err != nil {
		log.Printf("Error checking fuel records: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	if count > 0 {
		http.Error(w, "Fuel records already exist", http.StatusBadRequest)
		return
	}

	// Get some vehicles to create fuel records for
	var vehicles []struct {
		VehicleID string `db:"vehicle_id"`
	}
	
	// Try to get vehicle numbers from vehicles table
	err = db.Select(&vehicles, `
		SELECT vehicle_id 
		FROM vehicles 
		WHERE vehicle_type = 'fleet' AND vehicle_number IS NOT NULL 
		LIMIT 20
	`)
	if err != nil {
		// Try buses instead
		err = db.Select(&vehicles, "SELECT bus_id as vehicle_id FROM buses LIMIT 20")
		if err != nil {
			log.Printf("Error getting vehicles: %v", err)
			http.Error(w, "Failed to get vehicles", http.StatusInternalServerError)
			return
		}
	}

	if len(vehicles) == 0 {
		http.Error(w, "No vehicles found to create fuel records for", http.StatusBadRequest)
		return
	}

	// Create sample fuel records
	locations := []string{"Shell Station Main St", "BP Gas Downtown", "Chevron Highway 101", "Mobil Central Ave", "Texaco Fleet Center"}
	drivers := []string{"John Smith", "Jane Doe", "Mike Johnson", "Sarah Williams", "Tom Brown"}
	
	rand.Seed(time.Now().UnixNano())
	recordsCreated := 0

	// Create 3-6 months of fuel records for each vehicle
	for _, vehicle := range vehicles {
		baseOdometer := 50000 + rand.Intn(100000)
		currentOdometer := baseOdometer
		
		// Create records for the past 6 months
		for i := 6; i >= 0; i-- {
			date := time.Now().AddDate(0, -i, -rand.Intn(15))
			
			// Skip some months randomly
			if rand.Float32() > 0.8 {
				continue
			}

			gallons := 15.0 + rand.Float64()*25.0 // 15-40 gallons
			pricePerGallon := 3.00 + rand.Float64()*1.50 // $3.00-$4.50
			totalCost := gallons * pricePerGallon
			currentOdometer += 300 + rand.Intn(1200) // 300-1500 miles between fill-ups

			_, err := db.Exec(`
				INSERT INTO fuel_records (vehicle_id, date, gallons, price_per_gallon, cost, odometer, location, driver, created_at)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
				vehicle.VehicleID,
				date.Format("2006-01-02"),
				gallons,
				pricePerGallon,
				totalCost,
				currentOdometer,
				locations[rand.Intn(len(locations))],
				drivers[rand.Intn(len(drivers))],
				time.Now(),
			)
			if err != nil {
				log.Printf("Error inserting fuel record: %v", err)
				continue
			}
			recordsCreated++
		}
	}

	log.Printf("Created %d sample fuel records", recordsCreated)
	
	// Clear cache to ensure new data is loaded
	if dataCache != nil {
		dataCache.mu.Lock()
		dataCache.buses = nil
		dataCache.vehicles = nil
		dataCache.routes = nil
		dataCache.students = nil
		dataCache.users = nil
		dataCache.lastFetch = make(map[string]time.Time)
		dataCache.mu.Unlock()
	}

	// Redirect back to fuel records page
	http.Redirect(w, r, "/fuel-records?message="+fmt.Sprintf("Created %d sample fuel records", recordsCreated), http.StatusSeeOther)
}