package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"
)

// FuelRecord represents a fuel purchase record
type FuelRecord struct {
	ID               int            `json:"id" db:"id"`
	VehicleID        string         `json:"vehicle_id" db:"vehicle_id"`
	Date             string         `json:"date" db:"date"`
	Gallons          float64        `json:"gallons" db:"gallons"`
	PricePerGallon   float64        `json:"price_per_gallon" db:"price_per_gallon"`
	TotalCost        float64        `json:"total_cost" db:"cost"`
	Odometer         sql.NullInt32  `json:"odometer" db:"odometer"`
	PreviousOdometer sql.NullInt32  `json:"previous_odometer" db:"previous_odometer"`
	MPG              sql.NullFloat64 `json:"mpg" db:"mpg"`
	Location         sql.NullString `json:"location" db:"location"`
	Notes            sql.NullString `json:"notes" db:"notes"`
	RecordedBy       sql.NullString `json:"recorded_by" db:"driver"`
	CreatedAt        time.Time      `json:"created_at" db:"created_at"`
}

// Helper methods for FuelRecord to handle null values in templates
func (f FuelRecord) GetGallons() float64 {
	return f.Gallons
}

func (f FuelRecord) GetCost() float64 {
	return f.TotalCost
}

func (f FuelRecord) GetPricePerGallon() float64 {
	return f.PricePerGallon
}

func (f FuelRecord) GetMPG() float64 {
	if f.MPG.Valid {
		return f.MPG.Float64
	}
	return 0
}

// FuelEfficiency represents fuel efficiency metrics
type FuelEfficiency struct {
	VehicleID      string  `json:"vehicle_id"`
	Model          string  `json:"model"`
	AverageMPG     float64 `json:"average_mpg"`
	LastMPG        float64 `json:"last_mpg"`
	TotalGallons   float64 `json:"total_gallons"`
	TotalMiles     int     `json:"total_miles"`
	TotalCost      float64 `json:"total_cost"`
	CostPerMile    float64 `json:"cost_per_mile"`
	BestMPG        float64 `json:"best_mpg"`
	WorstMPG       float64 `json:"worst_mpg"`
	FillupCount    int     `json:"fillup_count"`
	LastFillupDate string  `json:"last_fillup_date"`
	TrendDirection string  `json:"trend_direction"`
}

// FleetFuelSummary represents overall fleet fuel metrics
type FleetFuelSummary struct {
	TotalGallons    float64            `json:"total_gallons"`
	TotalCost       float64            `json:"total_cost"`
	AveragePrice    float64            `json:"average_price"`
	FleetAverageMPG float64            `json:"fleet_average_mpg"`
	PeriodLabel     string             `json:"period_label"`
	TopPerformers   []FuelEfficiency   `json:"top_performers"`
	WorstPerformers []FuelEfficiency   `json:"worst_performers"`
	CostByVehicle   map[string]float64 `json:"cost_by_vehicle"`
	TrendData       []FuelTrendPoint   `json:"trend_data"`
}

// FuelTrendPoint represents a point in fuel trend data
type FuelTrendPoint struct {
	Period       string  `json:"period"`
	TotalGallons float64 `json:"total_gallons"`
	TotalCost    float64 `json:"total_cost"`
	AverageMPG   float64 `json:"average_mpg"`
	AveragePrice float64 `json:"average_price"`
}

// calculateMPG calculates miles per gallon between two odometer readings
func calculateMPG(startOdometer, endOdometer int, gallons float64) float64 {
	if gallons <= 0 {
		return 0
	}
	miles := endOdometer - startOdometer
	return float64(miles) / gallons
}

// GetVehicleFuelEfficiency calculates fuel efficiency for a specific vehicle
func GetVehicleFuelEfficiency(vehicleID string, startDate, endDate string) (*FuelEfficiency, error) {
	efficiency := &FuelEfficiency{
		VehicleID: vehicleID,
	}

	// Get vehicle details
	var model string
	err := db.Get(&model, "SELECT model FROM buses WHERE bus_id = $1 UNION SELECT model FROM vehicles WHERE vehicle_id = $1 LIMIT 1", vehicleID)
	if err != nil {
		log.Printf("Failed to get vehicle model: %v", err)
		model = "Unknown"
	}
	efficiency.Model = model

	// Get fuel records for the vehicle
	query := `
		SELECT id, vehicle_id, date, gallons, cost, price_per_gallon, odometer, driver
		FROM fuel_records
		WHERE vehicle_id = $1 AND date BETWEEN $2 AND $3
		ORDER BY date, odometer
	`

	var records []FuelRecord
	err = db.Select(&records, query, vehicleID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get fuel records: %w", err)
	}

	if len(records) == 0 {
		return efficiency, nil
	}

	// Calculate metrics
	var totalGallons, totalCost float64
	var mpgSum float64
	var mpgCount int
	var bestMPG, worstMPG float64

	for i := 1; i < len(records); i++ {
		prevRecord := records[i-1]
		currentRecord := records[i]

		if prevRecord.Odometer.Valid && currentRecord.Odometer.Valid {
			mpg := calculateMPG(int(prevRecord.Odometer.Int32), int(currentRecord.Odometer.Int32), currentRecord.Gallons)

			if mpg > 0 {
				mpgSum += mpg
				mpgCount++

				if bestMPG == 0 || mpg > bestMPG {
					bestMPG = mpg
				}
				if worstMPG == 0 || mpg < worstMPG {
					worstMPG = mpg
				}

				if i == len(records)-1 {
					efficiency.LastMPG = mpg
				}
			}
		}

		totalGallons += currentRecord.Gallons
		totalCost += currentRecord.TotalCost
	}

	// Include first record in totals
	if len(records) > 0 {
		totalGallons += records[0].Gallons
		totalCost += records[0].TotalCost
	}

	// Calculate total miles
	if len(records) > 1 && records[len(records)-1].Odometer.Valid && records[0].Odometer.Valid {
		efficiency.TotalMiles = int(records[len(records)-1].Odometer.Int32) - int(records[0].Odometer.Int32)
	}

	efficiency.TotalGallons = totalGallons
	efficiency.TotalCost = totalCost
	efficiency.FillupCount = len(records)

	if mpgCount > 0 {
		efficiency.AverageMPG = mpgSum / float64(mpgCount)
	}

	if efficiency.TotalMiles > 0 {
		efficiency.CostPerMile = totalCost / float64(efficiency.TotalMiles)
	}

	efficiency.BestMPG = bestMPG
	efficiency.WorstMPG = worstMPG

	if len(records) > 0 {
		efficiency.LastFillupDate = records[len(records)-1].Date
	}

	// Determine trend
	if mpgCount >= 3 {
		// Simple trend: compare last 3 MPG readings average to overall average
		recentSum := 0.0
		recentCount := 0
		for i := len(records) - 3; i < len(records) && i > 0; i++ {
			if i > 0 && records[i-1].Odometer.Valid && records[i].Odometer.Valid {
				mpg := calculateMPG(int(records[i-1].Odometer.Int32), int(records[i].Odometer.Int32), records[i].Gallons)
				if mpg > 0 {
					recentSum += mpg
					recentCount++
				}
			}
		}

		if recentCount > 0 {
			recentAvg := recentSum / float64(recentCount)
			if recentAvg > efficiency.AverageMPG*1.05 {
				efficiency.TrendDirection = "improving"
			} else if recentAvg < efficiency.AverageMPG*0.95 {
				efficiency.TrendDirection = "declining"
			} else {
				efficiency.TrendDirection = "stable"
			}
		}
	}

	return efficiency, nil
}

// GetFleetFuelSummary returns fuel summary for the entire fleet
func GetFleetFuelSummary(startDate, endDate string) (*FleetFuelSummary, error) {
	summary := &FleetFuelSummary{
		PeriodLabel:   fmt.Sprintf("%s to %s", startDate, endDate),
		CostByVehicle: make(map[string]float64),
	}

	// Get all fuel records for the period
	query := `
		SELECT vehicle_id, SUM(gallons) as total_gallons, SUM(cost) as total_cost,
		       AVG(price_per_gallon) as avg_price, COUNT(*) as fillup_count
		FROM fuel_records
		WHERE date BETWEEN $1 AND $2
		GROUP BY vehicle_id
	`

	type vehicleSummary struct {
		VehicleID    string  `db:"vehicle_id"`
		TotalGallons float64 `db:"total_gallons"`
		TotalCost    float64 `db:"total_cost"`
		AvgPrice     float64 `db:"avg_price"`
		FillupCount  int     `db:"fillup_count"`
	}

	var vehicleSummaries []vehicleSummary
	err := db.Select(&vehicleSummaries, query, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get vehicle summaries: %w", err)
	}

	// Calculate fleet totals and get individual vehicle efficiency
	var vehicleEfficiencies []FuelEfficiency

	for _, vs := range vehicleSummaries {
		summary.TotalGallons += vs.TotalGallons
		summary.TotalCost += vs.TotalCost
		summary.CostByVehicle[vs.VehicleID] = vs.TotalCost

		// Get efficiency for each vehicle
		efficiency, err := GetVehicleFuelEfficiency(vs.VehicleID, startDate, endDate)
		if err == nil && efficiency.AverageMPG > 0 {
			vehicleEfficiencies = append(vehicleEfficiencies, *efficiency)
		}
	}

	// Calculate fleet average price
	if summary.TotalGallons > 0 {
		summary.AveragePrice = summary.TotalCost / summary.TotalGallons
	}

	// Calculate fleet average MPG
	if len(vehicleEfficiencies) > 0 {
		totalMPG := 0.0
		for _, ve := range vehicleEfficiencies {
			totalMPG += ve.AverageMPG
		}
		summary.FleetAverageMPG = totalMPG / float64(len(vehicleEfficiencies))
	}

	// Sort vehicles by MPG to get top and worst performers
	// Simple bubble sort for small dataset
	for i := 0; i < len(vehicleEfficiencies); i++ {
		for j := i + 1; j < len(vehicleEfficiencies); j++ {
			if vehicleEfficiencies[j].AverageMPG > vehicleEfficiencies[i].AverageMPG {
				vehicleEfficiencies[i], vehicleEfficiencies[j] = vehicleEfficiencies[j], vehicleEfficiencies[i]
			}
		}
	}

	// Get top 5 performers
	topCount := 5
	if len(vehicleEfficiencies) < topCount {
		topCount = len(vehicleEfficiencies)
	}
	summary.TopPerformers = vehicleEfficiencies[:topCount]

	// Get worst 5 performers
	worstCount := 5
	if len(vehicleEfficiencies) < worstCount {
		worstCount = len(vehicleEfficiencies)
	}
	if len(vehicleEfficiencies) > 0 {
		summary.WorstPerformers = vehicleEfficiencies[len(vehicleEfficiencies)-worstCount:]
	}

	// Get trend data (monthly)
	trendQuery := `
		SELECT 
			TO_CHAR(date::date, 'Mon YYYY') as period,
			SUM(gallons) as total_gallons,
			SUM(cost) as total_cost,
			AVG(price_per_gallon) as avg_price
		FROM fuel_records
		WHERE date BETWEEN $1 AND $2
		GROUP BY TO_CHAR(date::date, 'Mon YYYY'), EXTRACT(YEAR FROM date::date), EXTRACT(MONTH FROM date::date)
		ORDER BY EXTRACT(YEAR FROM date::date), EXTRACT(MONTH FROM date::date)
	`

	var trendPoints []FuelTrendPoint
	err = db.Select(&trendPoints, trendQuery, startDate, endDate)
	if err == nil {
		summary.TrendData = trendPoints

		// Calculate MPG for each trend point
		// This is simplified - in reality you'd need more complex calculations
		for i := range summary.TrendData {
			if summary.TrendData[i].TotalGallons > 0 {
				// Estimate based on fleet average
				summary.TrendData[i].AverageMPG = summary.FleetAverageMPG
			}
		}
	}

	return summary, nil
}

// API Handlers

// saveFuelRecordHandler handles saving new fuel records
func saveFuelRecordHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user := getUserFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	if !validateCSRF(r) {
		http.Error(w, "Invalid CSRF token", http.StatusForbidden)
		return
	}

	var record FuelRecord
	err := json.NewDecoder(r.Body).Decode(&record)
	if err != nil {
		http.Error(w, "Invalid request data", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if record.VehicleID == "" || record.Date == "" || record.Gallons <= 0 || !record.Odometer.Valid || record.Odometer.Int32 <= 0 {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	// Calculate price per gallon if not provided
	if record.PricePerGallon == 0 && record.TotalCost > 0 && record.Gallons > 0 {
		record.PricePerGallon = record.TotalCost / record.Gallons
	}

	// Set driver if not provided
	if !record.RecordedBy.Valid || record.RecordedBy.String == "" {
		record.RecordedBy = sql.NullString{String: user.Username, Valid: true}
	}

	// Insert record
	query := `
		INSERT INTO fuel_records (vehicle_id, date, gallons, cost, price_per_gallon, 
		                         odometer, location, driver, notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id
	`

	err = db.QueryRow(query, record.VehicleID, record.Date, record.Gallons, record.TotalCost,
		record.PricePerGallon, record.Odometer, record.Location, record.RecordedBy, record.Notes).Scan(&record.ID)

	if err != nil {
		log.Printf("Failed to save fuel record: %v", err)
		http.Error(w, "Failed to save fuel record", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"id":      record.ID,
	})
}

// vehicleFuelEfficiencyHandler returns fuel efficiency for a specific vehicle
func vehicleFuelEfficiencyHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	vehicleID := r.URL.Query().Get("vehicle_id")
	if vehicleID == "" {
		http.Error(w, "Vehicle ID required", http.StatusBadRequest)
		return
	}

	// Default to last 12 months
	endDate := time.Now().Format("2006-01-02")
	startDate := time.Now().AddDate(0, -12, 0).Format("2006-01-02")

	if sd := r.URL.Query().Get("start_date"); sd != "" {
		startDate = sd
	}
	if ed := r.URL.Query().Get("end_date"); ed != "" {
		endDate = ed
	}

	efficiency, err := GetVehicleFuelEfficiency(vehicleID, startDate, endDate)
	if err != nil {
		log.Printf("Failed to get fuel efficiency: %v", err)
		http.Error(w, "Failed to get fuel efficiency", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(efficiency)
}

// fleetFuelSummaryHandler returns fuel summary for the fleet
func fleetFuelSummaryHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Default to current month
	now := time.Now()
	startDate := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local).Format("2006-01-02")
	endDate := now.Format("2006-01-02")

	if sd := r.URL.Query().Get("start_date"); sd != "" {
		startDate = sd
	}
	if ed := r.URL.Query().Get("end_date"); ed != "" {
		endDate = ed
	}

	summary, err := GetFleetFuelSummary(startDate, endDate)
	if err != nil {
		log.Printf("Failed to get fleet fuel summary: %v", err)
		http.Error(w, "Failed to get fuel summary", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}

// fuelTrendChartHandler returns fuel trend chart data
func fuelTrendChartHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get parameters
	vehicleID := r.URL.Query().Get("vehicle_id")
	months := 12
	if m := r.URL.Query().Get("months"); m != "" {
		if parsed, _ := strconv.Atoi(m); parsed > 0 {
			months = parsed
		}
	}

	endDate := time.Now()
	startDate := endDate.AddDate(0, -months, 0)

	var chartData *ChartData

	if vehicleID != "" {
		// Single vehicle trend
		_, err := GetVehicleFuelEfficiency(vehicleID, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
		if err != nil {
			http.Error(w, "Failed to get fuel data", http.StatusInternalServerError)
			return
		}

		// Create simple chart showing fillup history
		labels := []string{}
		mpgData := []float64{}
		_ = []float64{} // costData

		// This is simplified - in reality you'd query individual records
		chartData = &ChartData{
			Type:   ChartTypeLine,
			Title:  fmt.Sprintf("Fuel Efficiency - %s", vehicleID),
			Labels: labels,
			Datasets: []ChartDataset{
				{
					Label:           "MPG",
					Data:            mpgData,
					BorderColor:     "#3B82F6",
					BackgroundColor: "rgba(59, 130, 246, 0.1)",
					Fill:            true,
				},
			},
		}
	} else {
		// Fleet-wide trend
		summary, err := GetFleetFuelSummary(startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
		if err != nil {
			http.Error(w, "Failed to get fuel summary", http.StatusInternalServerError)
			return
		}

		labels := make([]string, len(summary.TrendData))
		gallonData := make([]float64, len(summary.TrendData))
		costData := make([]float64, len(summary.TrendData))

		for i, trend := range summary.TrendData {
			labels[i] = trend.Period
			gallonData[i] = trend.TotalGallons
			costData[i] = trend.TotalCost
		}

		chartData = &ChartData{
			Type:   ChartTypeBar,
			Title:  "Fleet Fuel Usage Trend",
			Labels: labels,
			Datasets: []ChartDataset{
				{
					Label:           "Gallons",
					Data:            gallonData,
					BackgroundColor: "#3B82F6",
				},
			},
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(chartData)
}
