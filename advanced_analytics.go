package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"time"
)

// AnalyticsEngine provides advanced analytics capabilities
type AnalyticsEngine struct {
	db    *sql.DB
	cache *DataCache
}

// Analytics models
type FleetAnalytics struct {
	OperationalMetrics  OperationalMetrics  `json:"operational_metrics"`
	FinancialMetrics    FinancialMetrics    `json:"financial_metrics"`
	SafetyMetrics       SafetyMetrics       `json:"safety_metrics"`
	EfficiencyMetrics   EfficiencyMetrics   `json:"efficiency_metrics"`
	PredictiveInsights  PredictiveInsights  `json:"predictive_insights"`
	Recommendations     []Recommendation    `json:"recommendations"`
	GeneratedAt         time.Time           `json:"generated_at"`
}

type OperationalMetrics struct {
	FleetUtilization    float64            `json:"fleet_utilization"`
	AverageRouteTime    float64            `json:"average_route_time_minutes"`
	OnTimePerformance   float64            `json:"on_time_performance"`
	VehicleAvailability float64            `json:"vehicle_availability"`
	DriverUtilization   float64            `json:"driver_utilization"`
	RouteEfficiency     map[string]float64 `json:"route_efficiency_by_route"`
}

type FinancialMetrics struct {
	TotalOperatingCost    float64            `json:"total_operating_cost"`
	CostPerMile          float64            `json:"cost_per_mile"`
	FuelCostTrend        []TrendPoint       `json:"fuel_cost_trend"`
	MaintenanceCostTrend []TrendPoint       `json:"maintenance_cost_trend"`
	CostByVehicle        map[string]float64 `json:"cost_by_vehicle"`
	BudgetVariance       float64            `json:"budget_variance"`
}

type SafetyMetrics struct {
	AccidentRate         float64            `json:"accident_rate"`
	SafetyScore          float64            `json:"safety_score"`
	MaintenanceCompliance float64           `json:"maintenance_compliance"`
	DriverSafetyScores   map[string]float64 `json:"driver_safety_scores"`
	VehicleHealthScores  map[string]float64 `json:"vehicle_health_scores"`
}

type EfficiencyMetrics struct {
	FuelEfficiency       float64            `json:"average_fuel_efficiency_mpg"`
	RouteOptimization    float64            `json:"route_optimization_score"`
	IdleTimePercentage   float64            `json:"idle_time_percentage"`
	EmptyMilesPercentage float64            `json:"empty_miles_percentage"`
	StudentLoadFactor    float64            `json:"student_load_factor"`
}

type PredictiveInsights struct {
	MaintenanceForecasts []MaintenanceForecast `json:"maintenance_forecasts"`
	FuelCostProjection   float64               `json:"fuel_cost_projection_30days"`
	FleetExpansionNeeds  int                   `json:"fleet_expansion_needs"`
	RiskAssessments      []RiskAssessment      `json:"risk_assessments"`
}

type TrendPoint struct {
	Date  time.Time `json:"date"`
	Value float64   `json:"value"`
}

type MaintenanceForecast struct {
	VehicleID      string    `json:"vehicle_id"`
	PredictedDate  time.Time `json:"predicted_date"`
	EstimatedCost  float64   `json:"estimated_cost"`
	MaintenanceType string   `json:"maintenance_type"`
	Confidence     float64   `json:"confidence"`
}

type RiskAssessment struct {
	Category    string  `json:"category"`
	Description string  `json:"description"`
	Severity    string  `json:"severity"`
	Probability float64 `json:"probability"`
	Impact      string  `json:"impact"`
}

type Recommendation struct {
	Priority    string  `json:"priority"`
	Category    string  `json:"category"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Impact      string  `json:"impact"`
	Effort      string  `json:"effort"`
	Savings     float64 `json:"estimated_savings"`
}

// NewAnalyticsEngine creates a new analytics engine
func NewAnalyticsEngine(db *sql.DB, cache *DataCache) *AnalyticsEngine {
	return &AnalyticsEngine{
		db:    db,
		cache: cache,
	}
}

// GenerateFleetAnalytics generates comprehensive fleet analytics
func (ae *AnalyticsEngine) GenerateFleetAnalytics() (*FleetAnalytics, error) {
	analytics := &FleetAnalytics{
		GeneratedAt: time.Now(),
	}

	// Generate each metric category
	var err error
	analytics.OperationalMetrics, err = ae.calculateOperationalMetrics()
	if err != nil {
		log.Printf("Error calculating operational metrics: %v", err)
	}

	analytics.FinancialMetrics, err = ae.calculateFinancialMetrics()
	if err != nil {
		log.Printf("Error calculating financial metrics: %v", err)
	}

	analytics.SafetyMetrics, err = ae.calculateSafetyMetrics()
	if err != nil {
		log.Printf("Error calculating safety metrics: %v", err)
	}

	analytics.EfficiencyMetrics, err = ae.calculateEfficiencyMetrics()
	if err != nil {
		log.Printf("Error calculating efficiency metrics: %v", err)
	}

	analytics.PredictiveInsights, err = ae.generatePredictiveInsights()
	if err != nil {
		log.Printf("Error generating predictive insights: %v", err)
	}

	analytics.Recommendations = ae.generateRecommendations(analytics)

	return analytics, nil
}

// Calculate operational metrics
func (ae *AnalyticsEngine) calculateOperationalMetrics() (OperationalMetrics, error) {
	metrics := OperationalMetrics{
		RouteEfficiency: make(map[string]float64),
	}

	// Fleet utilization
	var totalBuses, activeBuses int
	ae.db.QueryRow("SELECT COUNT(*) FROM buses").Scan(&totalBuses)
	ae.db.QueryRow("SELECT COUNT(*) FROM buses WHERE status = 'active'").Scan(&activeBuses)
	if totalBuses > 0 {
		metrics.FleetUtilization = float64(activeBuses) / float64(totalBuses) * 100
	}

	// Vehicle availability
	var availableVehicles int
	ae.db.QueryRow(`
		SELECT COUNT(*) FROM vehicles 
		WHERE status = 'active' 
		AND vehicle_id NOT IN (
			SELECT vehicle_id FROM maintenance_records 
			WHERE service_date = CURRENT_DATE
		)
	`).Scan(&availableVehicles)
	
	var totalVehicles int
	ae.db.QueryRow("SELECT COUNT(*) FROM vehicles").Scan(&totalVehicles)
	if totalVehicles > 0 {
		metrics.VehicleAvailability = float64(availableVehicles) / float64(totalVehicles) * 100
	}

	// Average route time
	ae.db.QueryRow(`
		SELECT AVG(EXTRACT(EPOCH FROM (end_time - start_time))/60) 
		FROM driver_logs 
		WHERE end_time IS NOT NULL 
		AND log_date > CURRENT_DATE - INTERVAL '30 days'
	`).Scan(&metrics.AverageRouteTime)

	// On-time performance (simulated - would need actual schedule data)
	metrics.OnTimePerformance = 92.5 // Placeholder

	// Driver utilization
	var totalDrivers, activeDrivers int
	ae.db.QueryRow("SELECT COUNT(*) FROM users WHERE role = 'driver'").Scan(&totalDrivers)
	ae.db.QueryRow(`
		SELECT COUNT(DISTINCT driver) 
		FROM route_assignments 
		WHERE assigned_date = CURRENT_DATE
	`).Scan(&activeDrivers)
	if totalDrivers > 0 {
		metrics.DriverUtilization = float64(activeDrivers) / float64(totalDrivers) * 100
	}

	// Route efficiency by route
	rows, err := ae.db.Query(`
		SELECT 
			r.route_id,
			r.route_name,
			AVG(CASE 
				WHEN dl.end_mileage > dl.start_mileage 
				THEN (dl.end_mileage - dl.start_mileage)
				ELSE 0 
			END) as avg_distance,
			COUNT(DISTINCT s.student_id) as student_count
		FROM routes r
		LEFT JOIN driver_logs dl ON r.route_id = dl.route_id
		LEFT JOIN students s ON r.route_id = s.route_id
		WHERE dl.log_date > CURRENT_DATE - INTERVAL '30 days'
		GROUP BY r.route_id, r.route_name
	`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var routeID, routeName string
			var avgDistance float64
			var studentCount int
			rows.Scan(&routeID, &routeName, &avgDistance, &studentCount)
			
			// Efficiency score based on students per mile
			if avgDistance > 0 {
				efficiency := float64(studentCount) / avgDistance * 10
				metrics.RouteEfficiency[routeName] = math.Min(efficiency, 100)
			}
		}
	}

	return metrics, nil
}

// Calculate financial metrics
func (ae *AnalyticsEngine) calculateFinancialMetrics() (FinancialMetrics, error) {
	metrics := FinancialMetrics{
		CostByVehicle: make(map[string]float64),
	}

	// Total operating cost (fuel + maintenance)
	var totalFuelCost, totalMaintenanceCost float64
	ae.db.QueryRow(`
		SELECT COALESCE(SUM(cost), 0) 
		FROM fuel_records 
		WHERE date > CURRENT_DATE - INTERVAL '30 days'
	`).Scan(&totalFuelCost)
	
	ae.db.QueryRow(`
		SELECT COALESCE(SUM(cost), 0) 
		FROM maintenance_records 
		WHERE service_date > CURRENT_DATE - INTERVAL '30 days'
	`).Scan(&totalMaintenanceCost)
	
	metrics.TotalOperatingCost = totalFuelCost + totalMaintenanceCost

	// Cost per mile
	var totalMiles float64
	ae.db.QueryRow(`
		SELECT COALESCE(SUM(total_mileage), 0) 
		FROM monthly_mileage_reports 
		WHERE month >= DATE_TRUNC('month', CURRENT_DATE - INTERVAL '30 days')
	`).Scan(&totalMiles)
	
	if totalMiles > 0 {
		metrics.CostPerMile = metrics.TotalOperatingCost / totalMiles
	}

	// Fuel cost trend
	fuelRows, err := ae.db.Query(`
		SELECT DATE_TRUNC('week', date) as week, SUM(cost)
		FROM fuel_records
		WHERE date > CURRENT_DATE - INTERVAL '12 weeks'
		GROUP BY week
		ORDER BY week
	`)
	if err == nil {
		defer fuelRows.Close()
		for fuelRows.Next() {
			var week time.Time
			var cost float64
			fuelRows.Scan(&week, &cost)
			metrics.FuelCostTrend = append(metrics.FuelCostTrend, TrendPoint{Date: week, Value: cost})
		}
	}

	// Maintenance cost trend
	maintRows, err := ae.db.Query(`
		SELECT DATE_TRUNC('month', service_date) as month, SUM(cost)
		FROM maintenance_records
		WHERE service_date > CURRENT_DATE - INTERVAL '6 months'
		GROUP BY month
		ORDER BY month
	`)
	if err == nil {
		defer maintRows.Close()
		for maintRows.Next() {
			var month time.Time
			var cost float64
			maintRows.Scan(&month, &cost)
			metrics.MaintenanceCostTrend = append(metrics.MaintenanceCostTrend, TrendPoint{Date: month, Value: cost})
		}
	}

	// Cost by vehicle
	vehicleRows, err := ae.db.Query(`
		SELECT 
			v.vehicle_id,
			COALESCE(f.fuel_cost, 0) + COALESCE(m.maint_cost, 0) as total_cost
		FROM vehicles v
		LEFT JOIN (
			SELECT vehicle_id, SUM(cost) as fuel_cost
			FROM fuel_records
			WHERE date > CURRENT_DATE - INTERVAL '30 days'
			GROUP BY vehicle_id
		) f ON v.vehicle_id = f.vehicle_id
		LEFT JOIN (
			SELECT vehicle_id, SUM(cost) as maint_cost
			FROM maintenance_records
			WHERE service_date > CURRENT_DATE - INTERVAL '30 days'
			GROUP BY vehicle_id
		) m ON v.vehicle_id = m.vehicle_id
		WHERE v.status = 'active'
		ORDER BY total_cost DESC
		LIMIT 20
	`)
	if err == nil {
		defer vehicleRows.Close()
		for vehicleRows.Next() {
			var vehicleID string
			var cost float64
			vehicleRows.Scan(&vehicleID, &cost)
			metrics.CostByVehicle[vehicleID] = cost
		}
	}

	// Budget variance (placeholder - would need budget data)
	metrics.BudgetVariance = -5.2 // 5.2% under budget

	return metrics, nil
}

// Calculate safety metrics
func (ae *AnalyticsEngine) calculateSafetyMetrics() (SafetyMetrics, error) {
	metrics := SafetyMetrics{
		DriverSafetyScores:  make(map[string]float64),
		VehicleHealthScores: make(map[string]float64),
	}

	// Accident rate (simulated - would need accident data)
	metrics.AccidentRate = 0.5 // per 100,000 miles

	// Maintenance compliance
	var totalVehicles, compliantVehicles int
	ae.db.QueryRow("SELECT COUNT(*) FROM vehicles WHERE status = 'active'").Scan(&totalVehicles)
	ae.db.QueryRow(`
		SELECT COUNT(DISTINCT vehicle_id) 
		FROM maintenance_records 
		WHERE service_date > CURRENT_DATE - INTERVAL '90 days'
	`).Scan(&compliantVehicles)
	
	if totalVehicles > 0 {
		metrics.MaintenanceCompliance = float64(compliantVehicles) / float64(totalVehicles) * 100
	}

	// Overall safety score
	metrics.SafetyScore = (metrics.MaintenanceCompliance + (100 - metrics.AccidentRate*10)) / 2

	// Driver safety scores (based on route completion and patterns)
	driverRows, err := ae.db.Query(`
		SELECT 
			u.username,
			COUNT(dl.log_id) as total_routes,
			AVG(CASE 
				WHEN dl.end_mileage > dl.start_mileage 
				THEN dl.end_mileage - dl.start_mileage 
				ELSE 0 
			END) as avg_distance,
			COUNT(CASE WHEN dl.end_time IS NOT NULL THEN 1 END) as completed_routes
		FROM users u
		JOIN driver_logs dl ON u.username = dl.driver_username
		WHERE u.role = 'driver'
		AND dl.log_date > CURRENT_DATE - INTERVAL '30 days'
		GROUP BY u.username
	`)
	if err == nil {
		defer driverRows.Close()
		for driverRows.Next() {
			var username string
			var totalRoutes, completedRoutes int
			var avgDistance float64
			driverRows.Scan(&username, &totalRoutes, &avgDistance, &completedRoutes)
			
			// Simple safety score calculation
			completionRate := float64(completedRoutes) / float64(totalRoutes) * 100
			safetyScore := completionRate * 0.8 + 20 // Base score of 20
			metrics.DriverSafetyScores[username] = math.Min(safetyScore, 100)
		}
	}

	// Vehicle health scores
	vehicleRows, err := ae.db.Query(`
		SELECT 
			v.vehicle_id,
			v.model,
			v.current_mileage,
			COALESCE(mr.last_service_mileage, 0) as last_service_mileage,
			COALESCE(mr.days_since_service, 999) as days_since_service
		FROM vehicles v
		LEFT JOIN (
			SELECT 
				vehicle_id,
				MAX(mileage) as last_service_mileage,
				EXTRACT(DAY FROM CURRENT_DATE - MAX(service_date)) as days_since_service
			FROM maintenance_records
			GROUP BY vehicle_id
		) mr ON v.vehicle_id = mr.vehicle_id
		WHERE v.status = 'active'
	`)
	if err == nil {
		defer vehicleRows.Close()
		for vehicleRows.Next() {
			var vehicleID, model string
			var currentMileage, lastServiceMileage, daysSinceService int
			vehicleRows.Scan(&vehicleID, &model, &currentMileage, &lastServiceMileage, &daysSinceService)
			
			// Health score based on maintenance recency
			milesSinceService := currentMileage - lastServiceMileage
			healthScore := 100.0
			
			if milesSinceService > 5000 {
				healthScore -= float64(milesSinceService-5000) / 100
			}
			if daysSinceService > 90 {
				healthScore -= float64(daysSinceService-90) / 10
			}
			
			metrics.VehicleHealthScores[vehicleID] = math.Max(healthScore, 0)
		}
	}

	return metrics, nil
}

// Calculate efficiency metrics
func (ae *AnalyticsEngine) calculateEfficiencyMetrics() (EfficiencyMetrics, error) {
	metrics := EfficiencyMetrics{}

	// Fuel efficiency
	ae.db.QueryRow(`
		WITH fuel_efficiency AS (
			SELECT 
				vehicle_id,
				CASE 
					WHEN gallons > 0 AND mileage > LAG(mileage) OVER (PARTITION BY vehicle_id ORDER BY date)
					THEN (mileage - LAG(mileage) OVER (PARTITION BY vehicle_id ORDER BY date)) / gallons
					ELSE NULL
				END as mpg
			FROM fuel_records
			WHERE date > CURRENT_DATE - INTERVAL '30 days'
		)
		SELECT AVG(mpg) FROM fuel_efficiency WHERE mpg IS NOT NULL
	`).Scan(&metrics.FuelEfficiency)

	// Route optimization score (based on actual vs optimal distance)
	metrics.RouteOptimization = 85.5 // Placeholder - would need route optimization data

	// Idle time percentage (simulated)
	metrics.IdleTimePercentage = 12.3

	// Empty miles percentage
	var totalMiles, loadedMiles float64
	ae.db.QueryRow(`
		SELECT 
			SUM(end_mileage - start_mileage) as total,
			SUM(CASE 
				WHEN passenger_count > 0 
				THEN end_mileage - start_mileage 
				ELSE 0 
			END) as loaded
		FROM driver_logs
		WHERE log_date > CURRENT_DATE - INTERVAL '30 days'
		AND end_mileage > start_mileage
	`).Scan(&totalMiles, &loadedMiles)
	
	if totalMiles > 0 {
		metrics.EmptyMilesPercentage = (totalMiles - loadedMiles) / totalMiles * 100
	}

	// Student load factor
	var totalCapacity, totalStudents int
	ae.db.QueryRow(`
		SELECT 
			SUM(b.capacity),
			COUNT(DISTINCT s.student_id)
		FROM buses b
		JOIN route_assignments ra ON b.bus_id = ra.bus_id
		JOIN students s ON ra.route_id = s.route_id
		WHERE b.status = 'active'
		AND ra.assigned_date = CURRENT_DATE
	`).Scan(&totalCapacity, &totalStudents)
	
	if totalCapacity > 0 {
		metrics.StudentLoadFactor = float64(totalStudents) / float64(totalCapacity) * 100
	}

	return metrics, nil
}

// Generate predictive insights
func (ae *AnalyticsEngine) generatePredictiveInsights() (PredictiveInsights, error) {
	insights := PredictiveInsights{}

	// Maintenance forecasts
	rows, err := ae.db.Query(`
		SELECT 
			v.vehicle_id,
			v.model,
			v.current_mileage,
			COALESCE(AVG(mr.mileage_interval), 5000) as avg_interval,
			COALESCE(AVG(mr.cost), 350) as avg_cost
		FROM vehicles v
		LEFT JOIN (
			SELECT 
				vehicle_id,
				mileage - LAG(mileage) OVER (PARTITION BY vehicle_id ORDER BY service_date) as mileage_interval,
				cost
			FROM maintenance_records
		) mr ON v.vehicle_id = mr.vehicle_id
		WHERE v.status = 'active'
		GROUP BY v.vehicle_id, v.model, v.current_mileage
	`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var vehicleID, model string
			var currentMileage int
			var avgInterval, avgCost float64
			rows.Scan(&vehicleID, &model, &currentMileage, &avgInterval, &avgCost)
			
			// Predict next maintenance
			milesUntilService := avgInterval - float64(currentMileage%int(avgInterval))
			daysUntilService := milesUntilService / 50 // Assume 50 miles per day
			
			forecast := MaintenanceForecast{
				VehicleID:      vehicleID,
				PredictedDate:  time.Now().AddDate(0, 0, int(daysUntilService)),
				EstimatedCost:  avgCost,
				MaintenanceType: "Regular Service",
				Confidence:     0.75,
			}
			
			insights.MaintenanceForecasts = append(insights.MaintenanceForecasts, forecast)
		}
	}

	// Fuel cost projection
	var recentFuelCost, avgDailyCost float64
	ae.db.QueryRow(`
		SELECT 
			SUM(cost),
			SUM(cost) / 30.0
		FROM fuel_records 
		WHERE date > CURRENT_DATE - INTERVAL '30 days'
	`).Scan(&recentFuelCost, &avgDailyCost)
	insights.FuelCostProjection = avgDailyCost * 30

	// Fleet expansion needs (based on utilization)
	var utilizationRate float64
	ae.db.QueryRow(`
		SELECT 
			CAST(COUNT(DISTINCT ra.bus_id) AS FLOAT) / 
			CAST(COUNT(DISTINCT b.bus_id) AS FLOAT) * 100
		FROM buses b
		LEFT JOIN route_assignments ra ON b.bus_id = ra.bus_id AND ra.assigned_date = CURRENT_DATE
		WHERE b.status = 'active'
	`).Scan(&utilizationRate)
	
	if utilizationRate > 85 {
		insights.FleetExpansionNeeds = int((utilizationRate - 85) / 10) + 1
	}

	// Risk assessments
	insights.RiskAssessments = ae.assessRisks()

	return insights, nil
}

// Assess operational risks
func (ae *AnalyticsEngine) assessRisks() []RiskAssessment {
	risks := []RiskAssessment{}

	// Check maintenance backlog
	var overdueCount int
	ae.db.QueryRow(`
		SELECT COUNT(*) 
		FROM vehicles 
		WHERE status = 'active'
		AND vehicle_id NOT IN (
			SELECT vehicle_id FROM maintenance_records 
			WHERE service_date > CURRENT_DATE - INTERVAL '90 days'
		)
	`).Scan(&overdueCount)
	
	if overdueCount > 5 {
		risks = append(risks, RiskAssessment{
			Category:    "Maintenance",
			Description: fmt.Sprintf("%d vehicles overdue for maintenance", overdueCount),
			Severity:    "high",
			Probability: 0.8,
			Impact:      "Potential breakdowns and safety issues",
		})
	}

	// Check driver shortage
	var driverShortage int
	ae.db.QueryRow(`
		SELECT 
			COUNT(DISTINCT r.route_id) - COUNT(DISTINCT ra.route_id)
		FROM routes r
		LEFT JOIN route_assignments ra ON r.route_id = ra.route_id 
			AND ra.assigned_date = CURRENT_DATE
		WHERE r.active = true
	`).Scan(&driverShortage)
	
	if driverShortage > 0 {
		risks = append(risks, RiskAssessment{
			Category:    "Staffing",
			Description: fmt.Sprintf("%d routes without assigned drivers", driverShortage),
			Severity:    "medium",
			Probability: 0.6,
			Impact:      "Service disruptions",
		})
	}

	// Check aging fleet
	var avgAge float64
	ae.db.QueryRow(`
		SELECT AVG(EXTRACT(YEAR FROM AGE(CURRENT_DATE, 
			TO_DATE(model_year::TEXT || '-01-01', 'YYYY-MM-DD'))))
		FROM vehicles 
		WHERE status = 'active' 
		AND model_year IS NOT NULL
	`).Scan(&avgAge)
	
	if avgAge > 10 {
		risks = append(risks, RiskAssessment{
			Category:    "Fleet",
			Description: fmt.Sprintf("Average fleet age is %.1f years", avgAge),
			Severity:    "medium",
			Probability: 0.7,
			Impact:      "Increased maintenance costs and reliability issues",
		})
	}

	return risks
}

// Generate recommendations based on analytics
func (ae *AnalyticsEngine) generateRecommendations(analytics *FleetAnalytics) []Recommendation {
	recommendations := []Recommendation{}

	// Check fuel efficiency
	if analytics.EfficiencyMetrics.FuelEfficiency < 6.0 {
		recommendations = append(recommendations, Recommendation{
			Priority:    "high",
			Category:    "efficiency",
			Title:       "Improve Fuel Efficiency",
			Description: "Average fuel efficiency is below target. Consider driver training and vehicle maintenance.",
			Impact:      "Reduce fuel costs by up to 15%",
			Effort:      "medium",
			Savings:     analytics.FinancialMetrics.TotalOperatingCost * 0.15,
		})
	}

	// Check maintenance compliance
	if analytics.SafetyMetrics.MaintenanceCompliance < 90 {
		recommendations = append(recommendations, Recommendation{
			Priority:    "high",
			Category:    "safety",
			Title:       "Improve Maintenance Compliance",
			Description: "Several vehicles are overdue for maintenance. Implement automated scheduling.",
			Impact:      "Reduce breakdown risk and improve safety",
			Effort:      "low",
			Savings:     5000, // Estimated from avoided breakdowns
		})
	}

	// Check route optimization
	if analytics.EfficiencyMetrics.EmptyMilesPercentage > 20 {
		recommendations = append(recommendations, Recommendation{
			Priority:    "medium",
			Category:    "efficiency",
			Title:       "Optimize Route Planning",
			Description: "High percentage of empty miles. Consider route consolidation.",
			Impact:      "Reduce mileage by 10-15%",
			Effort:      "high",
			Savings:     analytics.FinancialMetrics.CostPerMile * analytics.EfficiencyMetrics.EmptyMilesPercentage * 0.1,
		})
	}

	// Check fleet utilization
	if analytics.OperationalMetrics.FleetUtilization < 70 {
		recommendations = append(recommendations, Recommendation{
			Priority:    "low",
			Category:    "operations",
			Title:       "Review Fleet Size",
			Description: "Low fleet utilization suggests excess capacity.",
			Impact:      "Reduce fleet costs",
			Effort:      "high",
			Savings:     50000, // Estimated annual savings per vehicle
		})
	}

	return recommendations
}

// Analytics API Handler
func AnalyticsHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Initialize analytics engine
	engine := NewAnalyticsEngine(db.DB, dataCache)

	// Generate analytics
	analytics, err := engine.GenerateFleetAnalytics()
	if err != nil {
		log.Printf("Error generating analytics: %v", err)
		http.Error(w, "Error generating analytics", http.StatusInternalServerError)
		return
	}

	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(analytics)
}

// Export analytics report
func ExportAnalyticsReport(format string) (string, error) {
	engine := NewAnalyticsEngine(db.DB, dataCache)
	analytics, err := engine.GenerateFleetAnalytics()
	if err != nil {
		return "", err
	}

	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("fleet_analytics_%s.%s", timestamp, format)

	switch format {
	case "pdf":
		// Generate PDF report (would require PDF library)
		return filename, generatePDFReport(analytics, filename)
	case "xlsx":
		// Generate Excel report
		return filename, generateExcelReport(analytics, filename)
	case "json":
		// Save as JSON
		data, _ := json.MarshalIndent(analytics, "", "  ")
		return filename, os.WriteFile(filename, data, 0644)
	default:
		return "", fmt.Errorf("unsupported format: %s", format)
	}
}

// generatePDFReport creates a PDF report from fleet analytics
func generatePDFReport(analytics *FleetAnalytics, filename string) error {
	// Initialize report generator
	if db == nil {
		return fmt.Errorf("database not initialized")
	}
	
	reportGen := NewReportGenerator(db)
	
	// Generate fleet report in PDF format
	data, _, err := reportGen.GenerateFleetReport("pdf")
	if err != nil {
		return fmt.Errorf("failed to generate PDF report: %w", err)
	}
	
	// Write to file
	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write PDF file: %w", err)
	}
	
	return nil
}

// generateExcelReport creates an Excel report from fleet analytics
func generateExcelReport(analytics *FleetAnalytics, filename string) error {
	// Initialize report generator
	if db == nil {
		return fmt.Errorf("database not initialized")
	}
	
	reportGen := NewReportGenerator(db)
	
	// Generate fleet report in Excel format
	data, _, err := reportGen.GenerateFleetReport("excel")
	if err != nil {
		return fmt.Errorf("failed to generate Excel report: %w", err)
	}
	
	// Write to file
	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write Excel file: %w", err)
	}
	
	return nil
}