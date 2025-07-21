package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"
)

// ChartType represents the type of chart
type ChartType string

const (
	ChartTypeLine    ChartType = "line"
	ChartTypeBar     ChartType = "bar"
	ChartTypePie     ChartType = "pie"
	ChartTypeDonut   ChartType = "donut"
	ChartTypeArea    ChartType = "area"
	ChartTypeRadar   ChartType = "radar"
	ChartTypeScatter ChartType = "scatter"
)

// ChartData represents data for a chart
type ChartData struct {
	Type     ChartType              `json:"type"`
	Title    string                 `json:"title"`
	Labels   []string               `json:"labels"`
	Datasets []ChartDataset         `json:"datasets"`
	Options  map[string]interface{} `json:"options"`
}

// ChartDataset represents a dataset within a chart
type ChartDataset struct {
	Label           string      `json:"label"`
	Data            []float64   `json:"data"`
	BackgroundColor interface{} `json:"backgroundColor,omitempty"`
	BorderColor     string      `json:"borderColor,omitempty"`
	BorderWidth     int         `json:"borderWidth,omitempty"`
	Fill            bool        `json:"fill,omitempty"`
}

// Color palette for charts
var chartColors = []string{
	"#3B82F6", // Blue
	"#10B981", // Green
	"#F59E0B", // Amber
	"#EF4444", // Red
	"#8B5CF6", // Purple
	"#EC4899", // Pink
	"#14B8A6", // Teal
	"#F97316", // Orange
}

// GetFleetStatusChart returns fleet status pie chart data
func GetFleetStatusChart() (*ChartData, error) {
	buses, err := dataCache.getBuses()
	if err != nil {
		return nil, err
	}

	statusCounts := map[string]int{
		"active":         0,
		"maintenance":    0,
		"out_of_service": 0,
	}

	for _, bus := range buses {
		statusCounts[bus.Status]++
	}

	labels := []string{"Active", "Maintenance", "Out of Service"}
	data := []float64{
		float64(statusCounts["active"]),
		float64(statusCounts["maintenance"]),
		float64(statusCounts["out_of_service"]),
	}

	return &ChartData{
		Type:   ChartTypePie,
		Title:  "Fleet Status Distribution",
		Labels: labels,
		Datasets: []ChartDataset{
			{
				Data: data,
				BackgroundColor: []string{
					"#10B981", // Green for active
					"#F59E0B", // Amber for maintenance
					"#EF4444", // Red for out of service
				},
			},
		},
		Options: map[string]interface{}{
			"responsive": true,
			"plugins": map[string]interface{}{
				"legend": map[string]interface{}{
					"position": "bottom",
				},
			},
		},
	}, nil
}

// GetMileageTrendChart returns mileage trend line chart
func GetMileageTrendChart(months int) (*ChartData, error) {
	endDate := time.Now()
	startDate := endDate.AddDate(0, -months, 0)

	query := `
		SELECT 
			TO_CHAR(TO_DATE(year::text || '-' || month::text, 'YYYY-MM'), 'Mon YYYY') as month_label,
			SUM(total_miles) as total_miles
		FROM mileage_reports
		WHERE (year * 12 + month) >= $1 AND (year * 12 + month) <= $2
		GROUP BY year, month
		ORDER BY year, month
	`

	startMonthNum := startDate.Year()*12 + int(startDate.Month())
	endMonthNum := endDate.Year()*12 + int(endDate.Month())

	type monthlyMileage struct {
		MonthLabel string  `db:"month_label"`
		TotalMiles float64 `db:"total_miles"`
	}

	var results []monthlyMileage
	err := db.Select(&results, query, startMonthNum, endMonthNum)
	if err != nil {
		return nil, err
	}

	labels := make([]string, 0, len(results))
	data := make([]float64, 0, len(results))

	for _, result := range results {
		labels = append(labels, result.MonthLabel)
		data = append(data, result.TotalMiles)
	}

	return &ChartData{
		Type:   ChartTypeLine,
		Title:  fmt.Sprintf("Mileage Trend - Last %d Months", months),
		Labels: labels,
		Datasets: []ChartDataset{
			{
				Label:           "Total Miles",
				Data:            data,
				BorderColor:     "#3B82F6",
				BackgroundColor: "rgba(59, 130, 246, 0.1)",
				BorderWidth:     2,
				Fill:            true,
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
				},
			},
		},
	}, nil
}

// GetMaintenanceCostChart returns maintenance cost bar chart
func GetMaintenanceCostChart(monthsBack int) (*ChartData, error) {
	endDate := time.Now()
	startDate := endDate.AddDate(0, -monthsBack, 0)

	query := `
		SELECT 
			TO_CHAR(date, 'Mon YYYY') as month_label,
			SUM(cost) as total_cost,
			maintenance_type
		FROM (
			SELECT date, cost, maintenance_type FROM bus_maintenance_logs
			WHERE date >= $1::date
			UNION ALL
			SELECT date, cost, maintenance_type FROM vehicle_maintenance_logs
			WHERE date >= $1::date
		) as all_maintenance
		GROUP BY TO_CHAR(date, 'Mon YYYY'), EXTRACT(YEAR FROM date), EXTRACT(MONTH FROM date), maintenance_type
		ORDER BY EXTRACT(YEAR FROM date), EXTRACT(MONTH FROM date)
	`

	type maintenanceCost struct {
		MonthLabel      string  `db:"month_label"`
		TotalCost       float64 `db:"total_cost"`
		MaintenanceType string  `db:"maintenance_type"`
	}

	var results []maintenanceCost
	err := db.Select(&results, query, startDate)
	if err != nil {
		return nil, err
	}

	// Group by month and type
	monthMap := make(map[string]map[string]float64)
	typeSet := make(map[string]bool)

	for _, result := range results {
		if monthMap[result.MonthLabel] == nil {
			monthMap[result.MonthLabel] = make(map[string]float64)
		}
		monthMap[result.MonthLabel][result.MaintenanceType] = result.TotalCost
		typeSet[result.MaintenanceType] = true
	}

	// Extract unique months and types
	months := make([]string, 0, len(monthMap))
	for month := range monthMap {
		months = append(months, month)
	}

	types := make([]string, 0, len(typeSet))
	for t := range typeSet {
		types = append(types, t)
	}

	// Build datasets
	datasets := make([]ChartDataset, 0, len(types))
	for i, mainType := range types {
		data := make([]float64, len(months))
		for j, month := range months {
			if cost, exists := monthMap[month][mainType]; exists {
				data[j] = cost
			}
		}

		datasets = append(datasets, ChartDataset{
			Label:           mainType,
			Data:            data,
			BackgroundColor: chartColors[i%len(chartColors)],
			BorderWidth:     1,
		})
	}

	return &ChartData{
		Type:     ChartTypeBar,
		Title:    fmt.Sprintf("Maintenance Costs - Last %d Months", months),
		Labels:   months,
		Datasets: datasets,
		Options: map[string]interface{}{
			"responsive": true,
			"plugins": map[string]interface{}{
				"legend": map[string]interface{}{
					"position": "bottom",
				},
			},
			"scales": map[string]interface{}{
				"x": map[string]interface{}{
					"stacked": true,
				},
				"y": map[string]interface{}{
					"stacked":     true,
					"beginAtZero": true,
				},
			},
		},
	}, nil
}

// GetRouteUtilizationChart returns route utilization donut chart
func GetRouteUtilizationChart() (*ChartData, error) {
	query := `
		SELECT 
			r.name as route_name,
			COUNT(s.id) as student_count,
			50 as capacity -- Default bus capacity
		FROM routes r
		LEFT JOIN students s ON s.route_id = r.id AND s.active = true
		GROUP BY r.id, r.name
		ORDER BY student_count DESC
	`

	type routeUtil struct {
		RouteName    string `db:"route_name"`
		StudentCount int    `db:"student_count"`
		Capacity     int    `db:"capacity"`
	}

	var results []routeUtil
	err := db.Select(&results, query)
	if err != nil {
		return nil, err
	}

	labels := make([]string, 0, len(results))
	data := make([]float64, 0, len(results))
	colors := make([]string, 0, len(results))

	for i, result := range results {
		labels = append(labels, result.RouteName)
		utilization := float64(result.StudentCount) / float64(result.Capacity) * 100
		data = append(data, utilization)
		colors = append(colors, chartColors[i%len(chartColors)])
	}

	return &ChartData{
		Type:   ChartTypeDonut,
		Title:  "Route Utilization",
		Labels: labels,
		Datasets: []ChartDataset{
			{
				Data:            data,
				BackgroundColor: colors,
			},
		},
		Options: map[string]interface{}{
			"responsive": true,
			"plugins": map[string]interface{}{
				"legend": map[string]interface{}{
					"position": "right",
				},
			},
		},
	}, nil
}

// GetDriverPerformanceChart returns driver performance radar chart
func GetDriverPerformanceChart() (*ChartData, error) {
	metrics, err := getDriverPerformance()
	if err != nil {
		return nil, err
	}

	// Take top 5 drivers
	limit := 5
	if len(metrics) < limit {
		limit = len(metrics)
	}

	labels := make([]string, limit)
	tripData := make([]float64, limit)
	mileageData := make([]float64, limit)
	studentData := make([]float64, limit)

	for i := 0; i < limit; i++ {
		labels[i] = metrics[i].Username
		tripData[i] = float64(metrics[i].TotalTrips)
		mileageData[i] = float64(metrics[i].MilesDriven) / 100 // Scale down for visibility
		studentData[i] = float64(metrics[i].StudentCount)
	}

	return &ChartData{
		Type:   ChartTypeRadar,
		Title:  "Driver Performance Comparison",
		Labels: labels,
		Datasets: []ChartDataset{
			{
				Label:           "Trips",
				Data:            tripData,
				BorderColor:     "#3B82F6",
				BackgroundColor: "rgba(59, 130, 246, 0.2)",
			},
			{
				Label:           "Miles (÷100)",
				Data:            mileageData,
				BorderColor:     "#10B981",
				BackgroundColor: "rgba(16, 185, 129, 0.2)",
			},
			{
				Label:           "Students",
				Data:            studentData,
				BorderColor:     "#F59E0B",
				BackgroundColor: "rgba(245, 158, 11, 0.2)",
			},
		},
		Options: map[string]interface{}{
			"responsive": true,
			"plugins": map[string]interface{}{
				"legend": map[string]interface{}{
					"position": "bottom",
				},
			},
		},
	}, nil
}

// Chart API Handlers

// chartDataHandler handles requests for chart data
func chartDataHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	chartType := r.URL.Query().Get("type")
	if chartType == "" {
		http.Error(w, "Chart type required", http.StatusBadRequest)
		return
	}

	var chartData *ChartData
	var err error

	switch chartType {
	case "fleet-status":
		chartData, err = GetFleetStatusChart()
	case "mileage-trend":
		months := 6
		if m := r.URL.Query().Get("months"); m != "" {
			if parsed, _ := strconv.Atoi(m); parsed > 0 {
				months = parsed
			}
		}
		chartData, err = GetMileageTrendChart(months)
	case "maintenance-cost":
		months := 6
		if m := r.URL.Query().Get("months"); m != "" {
			if parsed, _ := strconv.Atoi(m); parsed > 0 {
				months = parsed
			}
		}
		chartData, err = GetMaintenanceCostChart(months)
	case "route-utilization":
		chartData, err = GetRouteUtilizationChart()
	case "driver-performance":
		if user.Role != "manager" {
			http.Error(w, "Manager access required", http.StatusForbidden)
			return
		}
		chartData, err = GetDriverPerformanceChart()
	default:
		http.Error(w, "Invalid chart type", http.StatusBadRequest)
		return
	}

	if err != nil {
		log.Printf("Failed to generate chart data: %v", err)
		http.Error(w, "Failed to generate chart data", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(chartData)
}

// availableChartsHandler returns available chart types
func availableChartsHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	charts := []map[string]interface{}{
		{
			"id":          "fleet-status",
			"name":        "Fleet Status",
			"description": "Current status distribution of all buses",
			"type":        "pie",
		},
		{
			"id":          "mileage-trend",
			"name":        "Mileage Trend",
			"description": "Monthly mileage trends over time",
			"type":        "line",
			"params": map[string]interface{}{
				"months": "Number of months to display (default: 6)",
			},
		},
		{
			"id":          "maintenance-cost",
			"name":        "Maintenance Costs",
			"description": "Monthly maintenance costs by type",
			"type":        "bar",
			"params": map[string]interface{}{
				"months": "Number of months to display (default: 6)",
			},
		},
		{
			"id":          "route-utilization",
			"name":        "Route Utilization",
			"description": "Student capacity utilization by route",
			"type":        "donut",
		},
	}

	// Add manager-only charts
	if user.Role == "manager" {
		charts = append(charts, map[string]interface{}{
			"id":          "driver-performance",
			"name":        "Driver Performance",
			"description": "Comparative driver performance metrics",
			"type":        "radar",
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(charts)
}
