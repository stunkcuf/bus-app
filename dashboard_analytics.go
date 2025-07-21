package main

import (
	"encoding/json"
	"net/http"
	"time"
)

// analyticsDashboardHandler serves the enhanced analytics dashboard
func analyticsDashboardHandler(w http.ResponseWriter, r *http.Request) {
	session, err := GetSession(r)
	if err != nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	if session.UserRole != "manager" {
		SendError(w, ErrForbidden("Manager access required"))
		return
	}

	data := struct {
		User      *User  `json:"user"`
		Role      string `json:"role"`
		CSRFToken string `json:"csrf_token"`
		CSPNonce  string `json:"csp_nonce"`
	}{
		User:      &User{Username: session.Username},
		Role:      session.UserRole,
		CSRFToken: session.CSRFToken,
		CSPNonce:  GenerateCSPNonce(),
	}

	if err := templates.ExecuteTemplate(w, "dashboard_enhanced.html", data); err != nil {
		LogError("Failed to execute analytics dashboard template", err)
		SendError(w, ErrInternal("Failed to render page", err))
		return
	}
}

// DashboardMetrics represents the analytics data for the dashboard
type DashboardMetrics struct {
	FleetOverview     FleetMetrics       `json:"fleet_overview"`
	RouteAnalytics    RouteMetrics       `json:"route_analytics"`
	MileageAnalytics  MileageMetrics     `json:"mileage_analytics"`
	MaintenanceCosts  MaintenanceMetrics `json:"maintenance_costs"`
	DriverPerformance []DriverMetric     `json:"driver_performance"`
	TrendData         TrendMetrics       `json:"trend_data"`
}

type FleetMetrics struct {
	TotalBuses       int            `json:"total_buses"`
	ActiveBuses      int            `json:"active_buses"`
	MaintenanceBuses int            `json:"maintenance_buses"`
	OutOfService     int            `json:"out_of_service"`
	UtilizationRate  float64        `json:"utilization_rate"`
	BusTypes         map[string]int `json:"bus_types"`
}

type RouteMetrics struct {
	TotalRoutes      int                `json:"total_routes"`
	ActiveRoutes     int                `json:"active_routes"`
	StudentsPerRoute map[string]int     `json:"students_per_route"`
	RouteEfficiency  map[string]float64 `json:"route_efficiency"`
	PeakHours        []HourlyActivity   `json:"peak_hours"`
}

type MileageMetrics struct {
	TotalMileage      int              `json:"total_mileage"`
	AverageDailyMiles float64          `json:"average_daily_miles"`
	MileageByVehicle  map[string]int   `json:"mileage_by_vehicle"`
	FuelEfficiency    float64          `json:"fuel_efficiency"`
	MileageTrend      []MonthlyMileage `json:"mileage_trend"`
}

type MaintenanceMetrics struct {
	TotalCost        float64            `json:"total_cost"`
	CostByType       map[string]float64 `json:"cost_by_type"`
	UpcomingServices []ServiceAlert     `json:"upcoming_services"`
	CostTrend        []MonthlyCost      `json:"cost_trend"`
	CostPerMile      float64            `json:"cost_per_mile"`
}

type DriverMetric struct {
	Username         string  `json:"username"`
	TotalTrips       int     `json:"total_trips"`
	OnTimePercentage float64 `json:"on_time_percentage"`
	SafetyScore      float64 `json:"safety_score"`
	StudentCount     int     `json:"student_count"`
	MilesDriven      int     `json:"miles_driven"`
}

type TrendMetrics struct {
	StudentGrowth []MonthlyCount `json:"student_growth"`
	FleetGrowth   []MonthlyCount `json:"fleet_growth"`
	CostAnalysis  []MonthlyCost  `json:"cost_analysis"`
	IncidentRate  []MonthlyCount `json:"incident_rate"`
}

type HourlyActivity struct {
	Hour  int `json:"hour"`
	Count int `json:"count"`
}

type MonthlyMileage struct {
	Month   string `json:"month"`
	Mileage int    `json:"mileage"`
}

type ServiceAlert struct {
	VehicleID    string `json:"vehicle_id"`
	ServiceType  string `json:"service_type"`
	DueDate      string `json:"due_date"`
	DaysUntilDue int    `json:"days_until_due"`
}

type MonthlyCost struct {
	Month string  `json:"month"`
	Cost  float64 `json:"cost"`
}

type MonthlyCount struct {
	Month string `json:"month"`
	Count int    `json:"count"`
}

// dashboardAnalyticsHandler returns comprehensive analytics data
func dashboardAnalyticsHandler(w http.ResponseWriter, r *http.Request) {
	session, err := GetSession(r)
	if err != nil || session.UserRole != "manager" {
		SendError(w, ErrForbidden("Manager access required"))
		return
	}

	metrics, err := gatherDashboardMetrics()
	if err != nil {
		SendError(w, ErrInternal("Failed to gather metrics", err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

// gatherDashboardMetrics collects all dashboard analytics
func gatherDashboardMetrics() (*DashboardMetrics, error) {
	metrics := &DashboardMetrics{}

	// Gather fleet metrics
	fleetMetrics, err := getFleetMetrics()
	if err != nil {
		LogError("Failed to get fleet metrics", err)
	} else {
		metrics.FleetOverview = *fleetMetrics
	}

	// Gather route analytics
	routeMetrics, err := getRouteMetrics()
	if err != nil {
		LogError("Failed to get route metrics", err)
	} else {
		metrics.RouteAnalytics = *routeMetrics
	}

	// Gather mileage analytics
	mileageMetrics, err := getMileageMetrics()
	if err != nil {
		LogError("Failed to get mileage metrics", err)
	} else {
		metrics.MileageAnalytics = *mileageMetrics
	}

	// Gather maintenance costs
	maintenanceMetrics, err := getMaintenanceMetrics()
	if err != nil {
		LogError("Failed to get maintenance metrics", err)
	} else {
		metrics.MaintenanceCosts = *maintenanceMetrics
	}

	// Gather driver performance
	driverMetrics, err := getDriverPerformance()
	if err != nil {
		LogError("Failed to get driver performance", err)
	} else {
		metrics.DriverPerformance = driverMetrics
	}

	// Gather trend data
	trendMetrics, err := getTrendMetrics()
	if err != nil {
		LogError("Failed to get trend metrics", err)
	} else {
		metrics.TrendData = *trendMetrics
	}

	return metrics, nil
}

// Fleet metrics functions
func getFleetMetrics() (*FleetMetrics, error) {
	metrics := &FleetMetrics{
		BusTypes: make(map[string]int),
	}

	// Count buses by status
	err := db.QueryRow(`
		SELECT 
			COUNT(*) as total,
			COUNT(CASE WHEN status = 'active' THEN 1 END) as active,
			COUNT(CASE WHEN status = 'maintenance' THEN 1 END) as maintenance,
			COUNT(CASE WHEN status = 'out_of_service' THEN 1 END) as out_of_service
		FROM buses
	`).Scan(&metrics.TotalBuses, &metrics.ActiveBuses, &metrics.MaintenanceBuses, &metrics.OutOfService)

	if err != nil {
		return nil, err
	}

	// Calculate utilization rate
	if metrics.TotalBuses > 0 {
		metrics.UtilizationRate = float64(metrics.ActiveBuses) / float64(metrics.TotalBuses) * 100
	}

	// Count by bus model/type
	rows, err := db.Query(`
		SELECT model, COUNT(*) as count
		FROM buses
		GROUP BY model
	`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var model string
			var count int
			if err := rows.Scan(&model, &count); err == nil {
				metrics.BusTypes[model] = count
			}
		}
	}

	return metrics, nil
}

// Route metrics functions
func getRouteMetrics() (*RouteMetrics, error) {
	metrics := &RouteMetrics{
		StudentsPerRoute: make(map[string]int),
		RouteEfficiency:  make(map[string]float64),
		PeakHours:        []HourlyActivity{},
	}

	// Count routes
	err := db.QueryRow(`
		SELECT COUNT(*) as total,
		       COUNT(DISTINCT ra.route_id) as active
		FROM routes r
		LEFT JOIN route_assignments ra ON r.id = ra.route_id
	`).Scan(&metrics.TotalRoutes, &metrics.ActiveRoutes)

	if err != nil {
		return nil, err
	}

	// Students per route
	rows, err := db.Query(`
		SELECT r.name, COUNT(s.id) as student_count
		FROM routes r
		LEFT JOIN students s ON s.route_id = r.id AND s.active = true
		GROUP BY r.id, r.name
	`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var routeName string
			var count int
			if err := rows.Scan(&routeName, &count); err == nil {
				metrics.StudentsPerRoute[routeName] = count
			}
		}
	}

	// Peak hours from trip logs
	rows, err = db.Query(`
		SELECT EXTRACT(HOUR FROM TO_TIMESTAMP(departure_time, 'HH24:MI')::time) as hour,
		       COUNT(*) as count
		FROM trip_logs
		WHERE date >= CURRENT_DATE - INTERVAL '7 days'
		GROUP BY hour
		ORDER BY hour
	`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var hour, count int
			if err := rows.Scan(&hour, &count); err == nil {
				metrics.PeakHours = append(metrics.PeakHours, HourlyActivity{
					Hour:  hour,
					Count: count,
				})
			}
		}
	}

	return metrics, nil
}

// Mileage metrics functions
func getMileageMetrics() (*MileageMetrics, error) {
	metrics := &MileageMetrics{
		MileageByVehicle: make(map[string]int),
		MileageTrend:     []MonthlyMileage{},
	}

	// Get total mileage for current month
	now := time.Now()
	err := db.QueryRow(`
		SELECT COALESCE(SUM(total_miles), 0)
		FROM mileage_reports
		WHERE month = $1 AND year = $2
	`, now.Month(), now.Year()).Scan(&metrics.TotalMileage)

	if err != nil {
		return nil, err
	}

	// Calculate average daily miles (only count weekdays as operational days)
	operationalDays := 0
	for d := 1; d <= now.Day(); d++ {
		date := time.Date(now.Year(), now.Month(), d, 0, 0, 0, 0, now.Location())
		weekday := date.Weekday()
		// Count Monday through Friday as operational days
		if weekday >= time.Monday && weekday <= time.Friday {
			operationalDays++
		}
	}
	
	if operationalDays > 0 {
		metrics.AverageDailyMiles = float64(metrics.TotalMileage) / float64(operationalDays)
	} else {
		// Fallback to calendar days if no operational days yet
		if now.Day() > 0 {
			metrics.AverageDailyMiles = float64(metrics.TotalMileage) / float64(now.Day())
		}
	}

	// Mileage by vehicle
	rows, err := db.Query(`
		SELECT vehicle_id, SUM(total_miles) as miles
		FROM mileage_reports
		WHERE month = $1 AND year = $2
		GROUP BY vehicle_id
	`, now.Month(), now.Year())
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var vehicleID string
			var miles int
			if err := rows.Scan(&vehicleID, &miles); err == nil {
				metrics.MileageByVehicle[vehicleID] = miles
			}
		}
	}

	// Mileage trend (last 6 months)
	rows, err = db.Query(`
		SELECT TO_CHAR(TO_DATE(year::text || '-' || month::text, 'YYYY-MM'), 'Mon YYYY') as month,
		       SUM(total_miles) as miles
		FROM mileage_reports
		WHERE (year * 12 + month) >= ($1 * 12 + $2) - 5
		GROUP BY year, month
		ORDER BY year, month
	`, now.Year(), int(now.Month()))
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var month string
			var miles int
			if err := rows.Scan(&month, &miles); err == nil {
				metrics.MileageTrend = append(metrics.MileageTrend, MonthlyMileage{
					Month:   month,
					Mileage: miles,
				})
			}
		}
	}

	// Calculate fuel efficiency (assuming average mpg)
	metrics.FuelEfficiency = 7.5 // Default MPG for school buses

	return metrics, nil
}

// Maintenance metrics functions
func getMaintenanceMetrics() (*MaintenanceMetrics, error) {
	metrics := &MaintenanceMetrics{
		CostByType:       make(map[string]float64),
		UpcomingServices: []ServiceAlert{},
		CostTrend:        []MonthlyCost{},
	}

	// Get total maintenance cost for current year
	now := time.Now()
	err := db.QueryRow(`
		SELECT COALESCE(SUM(cost), 0)
		FROM (
			SELECT cost FROM bus_maintenance_logs WHERE EXTRACT(YEAR FROM date) = $1
			UNION ALL
			SELECT cost FROM vehicle_maintenance_logs WHERE EXTRACT(YEAR FROM date) = $1
		) as all_maintenance
	`, now.Year()).Scan(&metrics.TotalCost)

	if err != nil {
		return nil, err
	}

	// Cost by maintenance type
	rows, err := db.Query(`
		SELECT maintenance_type, SUM(cost) as total_cost
		FROM (
			SELECT maintenance_type, cost FROM bus_maintenance_logs WHERE EXTRACT(YEAR FROM date) = $1
			UNION ALL
			SELECT maintenance_type, cost FROM vehicle_maintenance_logs WHERE EXTRACT(YEAR FROM date) = $1
		) as all_maintenance
		GROUP BY maintenance_type
	`, now.Year())
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var mainType string
			var cost float64
			if err := rows.Scan(&mainType, &cost); err == nil {
				metrics.CostByType[mainType] = cost
			}
		}
	}

	// Get upcoming services
	rows, err = db.Query(`
		SELECT bus_id, 'Oil Change' as service,
		       current_mileage + 3000 - last_oil_change as miles_until
		FROM buses
		WHERE status = 'active' AND (current_mileage - last_oil_change) > 2500
		ORDER BY miles_until
		LIMIT 5
	`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var vehicleID, serviceType string
			var milesUntil int
			if err := rows.Scan(&vehicleID, &serviceType, &milesUntil); err == nil {
				daysUntil := milesUntil / 100 // Rough estimate
				dueDate := now.AddDate(0, 0, daysUntil).Format("2006-01-02")

				metrics.UpcomingServices = append(metrics.UpcomingServices, ServiceAlert{
					VehicleID:    vehicleID,
					ServiceType:  serviceType,
					DueDate:      dueDate,
					DaysUntilDue: daysUntil,
				})
			}
		}
	}

	// Cost trend (last 6 months)
	rows, err = db.Query(`
		SELECT TO_CHAR(date, 'Mon YYYY') as month, SUM(cost) as total
		FROM (
			SELECT date, cost FROM bus_maintenance_logs
			UNION ALL
			SELECT date, cost FROM vehicle_maintenance_logs
		) as all_maintenance
		WHERE date >= CURRENT_DATE - INTERVAL '6 months'
		GROUP BY TO_CHAR(date, 'Mon YYYY'), EXTRACT(YEAR FROM date), EXTRACT(MONTH FROM date)
		ORDER BY EXTRACT(YEAR FROM date), EXTRACT(MONTH FROM date)
	`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var month string
			var cost float64
			if err := rows.Scan(&month, &cost); err == nil {
				metrics.CostTrend = append(metrics.CostTrend, MonthlyCost{
					Month: month,
					Cost:  cost,
				})
			}
		}
	}

	// Calculate cost per mile
	var totalMiles int
	db.QueryRow(`
		SELECT COALESCE(SUM(total_miles), 1)
		FROM mileage_reports
		WHERE year = $1
	`, now.Year()).Scan(&totalMiles)

	if totalMiles > 0 {
		metrics.CostPerMile = metrics.TotalCost / float64(totalMiles)
	}

	return metrics, nil
}

// Driver performance functions
func getDriverPerformance() ([]DriverMetric, error) {
	drivers := []DriverMetric{}

	rows, err := db.Query(`
		SELECT 
			u.username,
			COUNT(DISTINCT tl.id) as trip_count,
			COUNT(DISTINCT s.id) as student_count,
			COALESCE(SUM(tl.ending_mileage - tl.beginning_mileage), 0) as miles_driven
		FROM users u
		LEFT JOIN trip_logs tl ON tl.driver = u.username
		LEFT JOIN students s ON s.driver = u.username AND s.active = true
		WHERE u.role = 'driver' AND u.status = 'active'
		GROUP BY u.username
		ORDER BY trip_count DESC
		LIMIT 10
	`)

	if err != nil {
		return drivers, err
	}
	defer rows.Close()

	for rows.Next() {
		var driver DriverMetric
		err := rows.Scan(&driver.Username, &driver.TotalTrips, &driver.StudentCount, &driver.MilesDriven)
		if err == nil {
			// Calculate on-time percentage (simulated for now)
			driver.OnTimePercentage = 95.0 + float64(driver.TotalTrips%5)
			driver.SafetyScore = 98.0 - float64(driver.MilesDriven%1000)/1000

			drivers = append(drivers, driver)
		}
	}

	return drivers, nil
}

// Trend metrics functions
func getTrendMetrics() (*TrendMetrics, error) {
	metrics := &TrendMetrics{
		StudentGrowth: []MonthlyCount{},
		FleetGrowth:   []MonthlyCount{},
		CostAnalysis:  []MonthlyCost{},
		IncidentRate:  []MonthlyCount{},
	}

	// Student growth (simulated for now)
	now := time.Now()
	for i := 5; i >= 0; i-- {
		month := now.AddDate(0, -i, 0)
		count := 150 + (5-i)*8 // Simulated growth
		metrics.StudentGrowth = append(metrics.StudentGrowth, MonthlyCount{
			Month: month.Format("Jan 2006"),
			Count: count,
		})
	}

	// Fleet growth
	rows, err := db.Query(`
		SELECT COUNT(*) as count
		FROM buses
	`)
	if err == nil {
		defer rows.Close()
		if rows.Next() {
			var currentCount int
			rows.Scan(&currentCount)

			// Simulate historical data
			for i := 5; i >= 0; i-- {
				month := now.AddDate(0, -i, 0)
				count := currentCount - (i * 2) // Simulated historical data
				if count < 1 {
					count = 1
				}
				metrics.FleetGrowth = append(metrics.FleetGrowth, MonthlyCount{
					Month: month.Format("Jan 2006"),
					Count: count,
				})
			}
		}
	}

	// Use the cost trend data from maintenance metrics
	maintenanceMetrics, _ := getMaintenanceMetrics()
	if maintenanceMetrics != nil {
		metrics.CostAnalysis = maintenanceMetrics.CostTrend
	}

	// Incident rate (simulated - would come from incident tracking system)
	for i := 5; i >= 0; i-- {
		month := now.AddDate(0, -i, 0)
		count := 2 - (i % 3) // Low incident rate
		if count < 0 {
			count = 0
		}
		metrics.IncidentRate = append(metrics.IncidentRate, MonthlyCount{
			Month: month.Format("Jan 2006"),
			Count: count,
		})
	}

	return metrics, nil
}

// Widget-specific endpoints

// fleetStatusWidgetHandler returns fleet status for dashboard widget
func fleetStatusWidgetHandler(w http.ResponseWriter, r *http.Request) {
	_, err := GetSession(r)
	if err != nil {
		SendError(w, ErrUnauthorized("Please log in"))
		return
	}

	data := struct {
		Active       int `json:"active"`
		Maintenance  int `json:"maintenance"`
		OutOfService int `json:"out_of_service"`
		Total        int `json:"total"`
	}{}

	err = db.QueryRow(`
		SELECT 
			COUNT(CASE WHEN status = 'active' THEN 1 END),
			COUNT(CASE WHEN status = 'maintenance' THEN 1 END),
			COUNT(CASE WHEN status = 'out_of_service' THEN 1 END),
			COUNT(*)
		FROM buses
	`).Scan(&data.Active, &data.Maintenance, &data.OutOfService, &data.Total)

	if err != nil {
		SendError(w, ErrInternal("Failed to get fleet status", err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

// maintenanceAlertsWidgetHandler returns maintenance alerts for dashboard
func maintenanceAlertsWidgetHandler(w http.ResponseWriter, r *http.Request) {
	_, err := GetSession(r)
	if err != nil {
		SendError(w, ErrUnauthorized("Please log in"))
		return
	}

	alerts := []ServiceAlert{}

	// Get buses needing oil changes
	rows, err := db.Query(`
		SELECT bus_id, 
		       current_mileage - last_oil_change as miles_since,
		       3000 - (current_mileage - last_oil_change) as miles_until
		FROM buses
		WHERE status = 'active' AND (current_mileage - last_oil_change) > 2500
		ORDER BY miles_until
		LIMIT 5
	`)

	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var busID string
			var milesSince, milesUntil int
			if err := rows.Scan(&busID, &milesSince, &milesUntil); err == nil {
				daysUntil := milesUntil / 100 // Rough estimate

				alerts = append(alerts, ServiceAlert{
					VehicleID:    busID,
					ServiceType:  "Oil Change",
					DueDate:      time.Now().AddDate(0, 0, daysUntil).Format("2006-01-02"),
					DaysUntilDue: daysUntil,
				})
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(alerts)
}

// routeEfficiencyWidgetHandler returns route efficiency metrics
func routeEfficiencyWidgetHandler(w http.ResponseWriter, r *http.Request) {
	_, err := GetSession(r)
	if err != nil {
		SendError(w, ErrUnauthorized("Please log in"))
		return
	}

	type RouteEfficiency struct {
		RouteName       string  `json:"route_name"`
		StudentCount    int     `json:"student_count"`
		Capacity        int     `json:"capacity"`
		UtilizationRate float64 `json:"utilization_rate"`
		AverageTime     int     `json:"average_time_minutes"`
	}

	routes := []RouteEfficiency{}

	rows, err := db.Query(`
		SELECT 
			r.name,
			COUNT(DISTINCT s.id) as student_count,
			50 as capacity, -- Default bus capacity
			AVG(EXTRACT(EPOCH FROM (TO_TIMESTAMP(tl.arrival_time, 'HH24:MI') - 
			    TO_TIMESTAMP(tl.departure_time, 'HH24:MI')))/60) as avg_time
		FROM routes r
		LEFT JOIN students s ON s.route_id = r.id AND s.active = true
		LEFT JOIN route_assignments ra ON ra.route_id = r.id
		LEFT JOIN trip_logs tl ON tl.driver = ra.driver AND tl.date >= CURRENT_DATE - INTERVAL '30 days'
		GROUP BY r.id, r.name
		ORDER BY r.name
	`)

	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var route RouteEfficiency
			var avgTime *float64

			err := rows.Scan(&route.RouteName, &route.StudentCount, &route.Capacity, &avgTime)
			if err == nil {
				route.UtilizationRate = float64(route.StudentCount) / float64(route.Capacity) * 100
				if avgTime != nil {
					route.AverageTime = int(*avgTime)
				}
				routes = append(routes, route)
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(routes)
}
