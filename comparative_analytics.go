package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/jmoiron/sqlx"
)

// ComparativeAnalytics provides month-over-month and period comparison analytics
type ComparativeAnalytics struct {
	db *sqlx.DB
}

// NewComparativeAnalytics creates a new comparative analytics instance
func NewComparativeAnalytics() *ComparativeAnalytics {
	return &ComparativeAnalytics{db: db}
}

// ComparisonPeriod represents a time period for comparison
type ComparisonPeriod struct {
	Label     string    `json:"label"`
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
}

// ComparisonMetric represents a metric comparison between periods
type ComparisonMetric struct {
	MetricName     string  `json:"metric_name"`
	CurrentValue   float64 `json:"current_value"`
	PreviousValue  float64 `json:"previous_value"`
	Change         float64 `json:"change"`
	ChangePercent  float64 `json:"change_percent"`
	TrendDirection string  `json:"trend_direction"` // up, down, stable
}

// ComparisonReport represents a complete comparison report
type ComparisonReport struct {
	Title          string             `json:"title"`
	CurrentPeriod  ComparisonPeriod   `json:"current_period"`
	PreviousPeriod ComparisonPeriod   `json:"previous_period"`
	Metrics        []ComparisonMetric `json:"metrics"`
	Charts         []ChartData        `json:"charts"`
	GeneratedAt    time.Time          `json:"generated_at"`
}

// GetMonthOverMonthComparison returns month-over-month comparison data
func (ca *ComparativeAnalytics) GetMonthOverMonthComparison() (*ComparisonReport, error) {
	now := time.Now()
	currentMonthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)
	currentMonthEnd := currentMonthStart.AddDate(0, 1, -1)

	previousMonthStart := currentMonthStart.AddDate(0, -1, 0)
	previousMonthEnd := currentMonthStart.AddDate(0, 0, -1)

	report := &ComparisonReport{
		Title: "Month-over-Month Comparison",
		CurrentPeriod: ComparisonPeriod{
			Label:     currentMonthStart.Format("January 2006"),
			StartDate: currentMonthStart,
			EndDate:   currentMonthEnd,
		},
		PreviousPeriod: ComparisonPeriod{
			Label:     previousMonthStart.Format("January 2006"),
			StartDate: previousMonthStart,
			EndDate:   previousMonthEnd,
		},
		GeneratedAt: now,
	}

	// Collect various metrics
	metrics := []ComparisonMetric{}

	// Total Mileage
	currentMileage, prevMileage, err := ca.compareMileage(currentMonthStart, currentMonthEnd, previousMonthStart, previousMonthEnd)
	if err == nil {
		metrics = append(metrics, ca.calculateMetricChange("Total Mileage", currentMileage, prevMileage))
	}

	// Maintenance Costs
	currentCost, prevCost, err := ca.compareMaintenanceCosts(currentMonthStart, currentMonthEnd, previousMonthStart, previousMonthEnd)
	if err == nil {
		metrics = append(metrics, ca.calculateMetricChange("Maintenance Costs", currentCost, prevCost))
	}

	// Active Buses
	currentBuses, prevBuses, err := ca.compareActiveBuses(currentMonthEnd, previousMonthEnd)
	if err == nil {
		metrics = append(metrics, ca.calculateMetricChange("Active Buses", currentBuses, prevBuses))
	}

	// Trip Count
	currentTrips, prevTrips, err := ca.compareTripCount(currentMonthStart, currentMonthEnd, previousMonthStart, previousMonthEnd)
	if err == nil {
		metrics = append(metrics, ca.calculateMetricChange("Total Trips", currentTrips, prevTrips))
	}

	// Student Count
	currentStudents, prevStudents, err := ca.compareActiveStudents(currentMonthEnd, previousMonthEnd)
	if err == nil {
		metrics = append(metrics, ca.calculateMetricChange("Active Students", currentStudents, prevStudents))
	}

	report.Metrics = metrics

	// Generate comparison charts
	charts, err := ca.generateComparisonCharts(report)
	if err == nil {
		report.Charts = charts
	}

	return report, nil
}

// GetYearOverYearComparison returns year-over-year comparison data
func (ca *ComparativeAnalytics) GetYearOverYearComparison() (*ComparisonReport, error) {
	now := time.Now()
	currentYearStart := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, time.Local)
	currentYearEnd := time.Date(now.Year(), 12, 31, 23, 59, 59, 0, time.Local)

	previousYearStart := currentYearStart.AddDate(-1, 0, 0)
	_ = currentYearEnd.AddDate(-1, 0, 0) // previousYearEnd

	report := &ComparisonReport{
		Title: "Year-over-Year Comparison",
		CurrentPeriod: ComparisonPeriod{
			Label:     fmt.Sprintf("%d", currentYearStart.Year()),
			StartDate: currentYearStart,
			EndDate:   now, // Use current date instead of year end for YTD
		},
		PreviousPeriod: ComparisonPeriod{
			Label:     fmt.Sprintf("%d", previousYearStart.Year()),
			StartDate: previousYearStart,
			EndDate:   now.AddDate(-1, 0, 0), // Same day last year
		},
		GeneratedAt: now,
	}

	// Similar metric collection as month-over-month but for yearly data
	metrics := []ComparisonMetric{}

	// Annual metrics...
	currentMileage, prevMileage, err := ca.compareMileage(currentYearStart, now, previousYearStart, now.AddDate(-1, 0, 0))
	if err == nil {
		metrics = append(metrics, ca.calculateMetricChange("YTD Mileage", currentMileage, prevMileage))
	}

	report.Metrics = metrics
	return report, nil
}

// GetCustomPeriodComparison returns comparison between two custom periods
func (ca *ComparativeAnalytics) GetCustomPeriodComparison(period1Start, period1End, period2Start, period2End time.Time) (*ComparisonReport, error) {
	report := &ComparisonReport{
		Title: "Custom Period Comparison",
		CurrentPeriod: ComparisonPeriod{
			Label:     fmt.Sprintf("%s - %s", period1Start.Format("Jan 2, 2006"), period1End.Format("Jan 2, 2006")),
			StartDate: period1Start,
			EndDate:   period1End,
		},
		PreviousPeriod: ComparisonPeriod{
			Label:     fmt.Sprintf("%s - %s", period2Start.Format("Jan 2, 2006"), period2End.Format("Jan 2, 2006")),
			StartDate: period2Start,
			EndDate:   period2End,
		},
		GeneratedAt: time.Now(),
	}

	// Collect metrics for custom periods
	metrics := []ComparisonMetric{}

	currentMileage, prevMileage, err := ca.compareMileage(period1Start, period1End, period2Start, period2End)
	if err == nil {
		metrics = append(metrics, ca.calculateMetricChange("Total Mileage", currentMileage, prevMileage))
	}

	report.Metrics = metrics
	return report, nil
}

// Metric comparison functions

func (ca *ComparativeAnalytics) compareMileage(currentStart, currentEnd, prevStart, prevEnd time.Time) (float64, float64, error) {
	// Current period mileage
	var currentMileage float64
	err := ca.db.Get(&currentMileage, `
		SELECT COALESCE(SUM(total_miles), 0)
		FROM mileage_reports
		WHERE (year * 12 + month) >= $1 AND (year * 12 + month) <= $2
	`, currentStart.Year()*12+int(currentStart.Month()), currentEnd.Year()*12+int(currentEnd.Month()))
	if err != nil {
		return 0, 0, err
	}

	// Previous period mileage
	var prevMileage float64
	err = ca.db.Get(&prevMileage, `
		SELECT COALESCE(SUM(total_miles), 0)
		FROM mileage_reports
		WHERE (year * 12 + month) >= $1 AND (year * 12 + month) <= $2
	`, prevStart.Year()*12+int(prevStart.Month()), prevEnd.Year()*12+int(prevEnd.Month()))
	if err != nil {
		return 0, 0, err
	}

	return currentMileage, prevMileage, nil
}

func (ca *ComparativeAnalytics) compareMaintenanceCosts(currentStart, currentEnd, prevStart, prevEnd time.Time) (float64, float64, error) {
	// Current period costs
	var currentCost float64
	err := ca.db.Get(&currentCost, `
		SELECT COALESCE(SUM(cost), 0)
		FROM (
			SELECT cost FROM bus_maintenance_logs WHERE date BETWEEN $1 AND $2
			UNION ALL
			SELECT cost FROM vehicle_maintenance_logs WHERE date BETWEEN $1 AND $2
		) as maintenance
	`, currentStart, currentEnd)
	if err != nil {
		return 0, 0, err
	}

	// Previous period costs
	var prevCost float64
	err = ca.db.Get(&prevCost, `
		SELECT COALESCE(SUM(cost), 0)
		FROM (
			SELECT cost FROM bus_maintenance_logs WHERE date BETWEEN $1 AND $2
			UNION ALL
			SELECT cost FROM vehicle_maintenance_logs WHERE date BETWEEN $1 AND $2
		) as maintenance
	`, prevStart, prevEnd)
	if err != nil {
		return 0, 0, err
	}

	return currentCost, prevCost, nil
}

func (ca *ComparativeAnalytics) compareActiveBuses(currentDate, prevDate time.Time) (float64, float64, error) {
	// Current active buses
	var currentBuses int
	err := ca.db.Get(&currentBuses, `
		SELECT COUNT(*) FROM buses WHERE status = 'active'
	`)
	if err != nil {
		return 0, 0, err
	}

	// For historical comparison, we would need a history table
	// For now, we'll use the same count
	prevBuses := currentBuses

	return float64(currentBuses), float64(prevBuses), nil
}

func (ca *ComparativeAnalytics) compareTripCount(currentStart, currentEnd, prevStart, prevEnd time.Time) (float64, float64, error) {
	// Current period trips
	var currentTrips int
	err := ca.db.Get(&currentTrips, `
		SELECT COUNT(*) FROM trip_logs WHERE date BETWEEN $1 AND $2
	`, currentStart, currentEnd)
	if err != nil {
		return 0, 0, err
	}

	// Previous period trips
	var prevTrips int
	err = ca.db.Get(&prevTrips, `
		SELECT COUNT(*) FROM trip_logs WHERE date BETWEEN $1 AND $2
	`, prevStart, prevEnd)
	if err != nil {
		return 0, 0, err
	}

	return float64(currentTrips), float64(prevTrips), nil
}

func (ca *ComparativeAnalytics) compareActiveStudents(currentDate, prevDate time.Time) (float64, float64, error) {
	// Current active students
	var currentStudents int
	err := ca.db.Get(&currentStudents, `
		SELECT COUNT(*) FROM students WHERE active = true
	`)
	if err != nil {
		return 0, 0, err
	}

	// For historical comparison, we would need a history table
	prevStudents := currentStudents

	return float64(currentStudents), float64(prevStudents), nil
}

// calculateMetricChange calculates the change between two values
func (ca *ComparativeAnalytics) calculateMetricChange(metricName string, current, previous float64) ComparisonMetric {
	change := current - previous
	changePercent := 0.0
	if previous > 0 {
		changePercent = (change / previous) * 100
	}

	trendDirection := "stable"
	if change > 0 {
		trendDirection = "up"
	} else if change < 0 {
		trendDirection = "down"
	}

	return ComparisonMetric{
		MetricName:     metricName,
		CurrentValue:   current,
		PreviousValue:  previous,
		Change:         change,
		ChangePercent:  changePercent,
		TrendDirection: trendDirection,
	}
}

// generateComparisonCharts generates charts for the comparison report
func (ca *ComparativeAnalytics) generateComparisonCharts(report *ComparisonReport) ([]ChartData, error) {
	charts := []ChartData{}

	// Metric comparison bar chart
	labels := []string{}
	currentValues := []float64{}
	previousValues := []float64{}

	for _, metric := range report.Metrics {
		labels = append(labels, metric.MetricName)
		currentValues = append(currentValues, metric.CurrentValue)
		previousValues = append(previousValues, metric.PreviousValue)
	}

	comparisonChart := ChartData{
		Type:   ChartTypeBar,
		Title:  "Period Comparison",
		Labels: labels,
		Datasets: []ChartDataset{
			{
				Label:           report.CurrentPeriod.Label,
				Data:            currentValues,
				BackgroundColor: "#3B82F6",
				BorderWidth:     1,
			},
			{
				Label:           report.PreviousPeriod.Label,
				Data:            previousValues,
				BackgroundColor: "#9CA3AF",
				BorderWidth:     1,
			},
		},
		Options: map[string]interface{}{
			"responsive": true,
			"plugins": map[string]interface{}{
				"legend": map[string]interface{}{
					"position": "bottom",
				},
			},
			"scales": map[string]interface{}{
				"y": map[string]interface{}{
					"beginAtZero": true,
				},
			},
		},
	}

	charts = append(charts, comparisonChart)

	// Trend chart showing percentage changes
	changeLabels := []string{}
	changeValues := []float64{}
	changeColors := []string{}

	for _, metric := range report.Metrics {
		changeLabels = append(changeLabels, metric.MetricName)
		changeValues = append(changeValues, metric.ChangePercent)

		// Color based on positive/negative change
		if metric.ChangePercent > 0 {
			changeColors = append(changeColors, "#10B981") // Green
		} else if metric.ChangePercent < 0 {
			changeColors = append(changeColors, "#EF4444") // Red
		} else {
			changeColors = append(changeColors, "#9CA3AF") // Gray
		}
	}

	trendChart := ChartData{
		Type:   ChartTypeBar,
		Title:  "Percentage Change",
		Labels: changeLabels,
		Datasets: []ChartDataset{
			{
				Label:           "% Change",
				Data:            changeValues,
				BackgroundColor: changeColors,
				BorderWidth:     1,
			},
		},
		Options: map[string]interface{}{
			"responsive": true,
			"plugins": map[string]interface{}{
				"legend": map[string]interface{}{
					"display": false,
				},
			},
			"scales": map[string]interface{}{
				"y": map[string]interface{}{
					"beginAtZero": true,
					"ticks": map[string]interface{}{
						"callback": "function(value) { return value + '%'; }",
					},
				},
			},
		},
	}

	charts = append(charts, trendChart)

	return charts, nil
}

// GetTrendAnalysis returns trend analysis over multiple periods
func (ca *ComparativeAnalytics) GetTrendAnalysis(metricType string, periods int) (*TrendAnalysis, error) {
	analysis := &TrendAnalysis{
		MetricType: metricType,
		Periods:    []TrendPeriod{},
	}

	now := time.Now()

	for i := 0; i < periods; i++ {
		periodStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local).AddDate(0, -i, 0)
		_ = periodStart.AddDate(0, 1, -1) // periodEnd

		var value float64
		var err error

		switch metricType {
		case "mileage":
			err = ca.db.Get(&value, `
				SELECT COALESCE(SUM(total_miles), 0)
				FROM mileage_reports
				WHERE year = $1 AND month = $2
			`, periodStart.Year(), int(periodStart.Month()))

		case "maintenance_cost":
			err = ca.db.Get(&value, `
				SELECT COALESCE(SUM(cost), 0)
				FROM (
					SELECT cost FROM bus_maintenance_logs 
					WHERE EXTRACT(YEAR FROM date) = $1 AND EXTRACT(MONTH FROM date) = $2
					UNION ALL
					SELECT cost FROM vehicle_maintenance_logs 
					WHERE EXTRACT(YEAR FROM date) = $1 AND EXTRACT(MONTH FROM date) = $2
				) as maintenance
			`, periodStart.Year(), int(periodStart.Month()))

		case "trips":
			err = ca.db.Get(&value, `
				SELECT COUNT(*)
				FROM trip_logs
				WHERE EXTRACT(YEAR FROM date) = $1 AND EXTRACT(MONTH FROM date) = $2
			`, periodStart.Year(), int(periodStart.Month()))

		default:
			return nil, fmt.Errorf("unknown metric type: %s", metricType)
		}

		if err != nil {
			log.Printf("Error getting trend data for %s: %v", metricType, err)
			value = 0
		}

		analysis.Periods = append([]TrendPeriod{{
			Label: periodStart.Format("Jan 2006"),
			Value: value,
			Date:  periodStart,
		}}, analysis.Periods...)
	}

	// Calculate trend statistics
	if len(analysis.Periods) > 1 {
		analysis.CalculateTrend()
	}

	return analysis, nil
}

// TrendAnalysis represents trend data over time
type TrendAnalysis struct {
	MetricType   string        `json:"metric_type"`
	Periods      []TrendPeriod `json:"periods"`
	AverageValue float64       `json:"average_value"`
	TrendSlope   float64       `json:"trend_slope"`
	TrendType    string        `json:"trend_type"` // increasing, decreasing, stable
}

// TrendPeriod represents a single period in trend analysis
type TrendPeriod struct {
	Label string    `json:"label"`
	Value float64   `json:"value"`
	Date  time.Time `json:"date"`
}

// CalculateTrend calculates trend statistics
func (ta *TrendAnalysis) CalculateTrend() {
	if len(ta.Periods) == 0 {
		return
	}

	// Calculate average
	sum := 0.0
	for _, period := range ta.Periods {
		sum += period.Value
	}
	ta.AverageValue = sum / float64(len(ta.Periods))

	// Calculate trend slope using simple linear regression
	n := float64(len(ta.Periods))
	sumX := 0.0
	sumY := 0.0
	sumXY := 0.0
	sumX2 := 0.0

	for i, period := range ta.Periods {
		x := float64(i)
		y := period.Value
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}

	ta.TrendSlope = (n*sumXY - sumX*sumY) / (n*sumX2 - sumX*sumX)

	// Determine trend type
	if ta.TrendSlope > 0.01 {
		ta.TrendType = "increasing"
	} else if ta.TrendSlope < -0.01 {
		ta.TrendType = "decreasing"
	} else {
		ta.TrendType = "stable"
	}
}

// API Handlers

// comparativeAnalyticsHandler handles requests for comparative analytics
func comparativeAnalyticsHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	comparisonType := r.URL.Query().Get("type")
	if comparisonType == "" {
		comparisonType = "month-over-month"
	}

	analytics := NewComparativeAnalytics()
	var report *ComparisonReport
	var err error

	switch comparisonType {
	case "month-over-month":
		report, err = analytics.GetMonthOverMonthComparison()

	case "year-over-year":
		report, err = analytics.GetYearOverYearComparison()

	case "custom":
		// Parse custom date parameters
		period1Start := r.URL.Query().Get("period1_start")
		period1End := r.URL.Query().Get("period1_end")
		period2Start := r.URL.Query().Get("period2_start")
		period2End := r.URL.Query().Get("period2_end")

		if period1Start == "" || period1End == "" || period2Start == "" || period2End == "" {
			http.Error(w, "Custom comparison requires all date parameters", http.StatusBadRequest)
			return
		}

		p1Start, _ := time.Parse("2006-01-02", period1Start)
		p1End, _ := time.Parse("2006-01-02", period1End)
		p2Start, _ := time.Parse("2006-01-02", period2Start)
		p2End, _ := time.Parse("2006-01-02", period2End)

		report, err = analytics.GetCustomPeriodComparison(p1Start, p1End, p2Start, p2End)

	default:
		http.Error(w, "Invalid comparison type", http.StatusBadRequest)
		return
	}

	if err != nil {
		log.Printf("Failed to generate comparative analytics: %v", err)
		http.Error(w, "Failed to generate analytics", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(report)
}

// trendAnalysisHandler handles requests for trend analysis
func trendAnalysisHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	metricType := r.URL.Query().Get("metric")
	if metricType == "" {
		http.Error(w, "Metric type required", http.StatusBadRequest)
		return
	}

	periods := 12 // Default to 12 months
	if p := r.URL.Query().Get("periods"); p != "" {
		if parsed, _ := strconv.Atoi(p); parsed > 0 && parsed <= 24 {
			periods = parsed
		}
	}

	analytics := NewComparativeAnalytics()
	trend, err := analytics.GetTrendAnalysis(metricType, periods)
	if err != nil {
		log.Printf("Failed to generate trend analysis: %v", err)
		http.Error(w, "Failed to generate trend analysis", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(trend)
}
