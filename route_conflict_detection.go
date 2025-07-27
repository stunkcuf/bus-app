package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// RouteConflict represents a scheduling conflict
type RouteConflict struct {
	Type        string    `json:"type"`
	Description string    `json:"description"`
	Severity    string    `json:"severity"` // "error", "warning", "info"
	Details     map[string]interface{} `json:"details"`
}

// RouteAssignmentCheck represents the result of conflict checking
type RouteAssignmentCheck struct {
	CanAssign bool            `json:"can_assign"`
	Conflicts []RouteConflict `json:"conflicts"`
	Warnings  []RouteConflict `json:"warnings"`
}

// checkRouteConflictsHandler checks for conflicts before assigning a route
func checkRouteConflictsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request
	var req struct {
		DriverID string `json:"driver_id"`
		BusID    string `json:"bus_id"`
		RouteID  string `json:"route_id"`
		Period   string `json:"period"`
		Date     string `json:"date,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Check for conflicts
	conflicts := checkRouteConflicts(req.DriverID, req.BusID, req.RouteID, req.Period, req.Date)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(conflicts)
}

// checkRouteConflicts performs comprehensive conflict detection
func checkRouteConflicts(driverID, busID, routeID, period, dateStr string) RouteAssignmentCheck {
	check := RouteAssignmentCheck{
		CanAssign: true,
		Conflicts: []RouteConflict{},
		Warnings:  []RouteConflict{},
	}

	// Check driver conflicts
	driverConflicts := checkDriverConflicts(driverID, period, dateStr)
	check.Conflicts = append(check.Conflicts, driverConflicts...)

	// Check bus conflicts
	busConflicts := checkBusConflicts(busID, period, dateStr)
	check.Conflicts = append(check.Conflicts, busConflicts...)

	// Check route conflicts
	routeConflicts := checkRouteSpecificConflicts(routeID, period)
	check.Conflicts = append(check.Conflicts, routeConflicts...)

	// Check driver qualifications
	qualificationWarnings := checkDriverQualifications(driverID, busID, routeID)
	check.Warnings = append(check.Warnings, qualificationWarnings...)

	// Check bus capacity
	capacityWarnings := checkBusCapacity(busID, routeID)
	check.Warnings = append(check.Warnings, capacityWarnings...)

	// Check maintenance schedules
	maintenanceWarnings := checkMaintenanceSchedule(busID, dateStr)
	check.Warnings = append(check.Warnings, maintenanceWarnings...)

	// Determine if assignment can proceed
	for _, conflict := range check.Conflicts {
		if conflict.Severity == "error" {
			check.CanAssign = false
			break
		}
	}

	return check
}

// checkDriverConflicts checks for driver scheduling conflicts
func checkDriverConflicts(driverID, period, dateStr string) []RouteConflict {
	conflicts := []RouteConflict{}

	// Check if driver is already assigned to another route in the same period
	var existingRoute string
	query := `
		SELECT route_id 
		FROM route_assignments 
		WHERE driver_id = $1 AND period = $2 AND active = true
	`
	err := db.Get(&existingRoute, query, driverID, period)
	if err == nil {
		conflicts = append(conflicts, RouteConflict{
			Type:        "driver_already_assigned",
			Description: fmt.Sprintf("Driver is already assigned to route %s during %s period", existingRoute, period),
			Severity:    "error",
			Details: map[string]interface{}{
				"existing_route": existingRoute,
				"period":        period,
			},
		})
	}

	// Check driver availability
	if dateStr != "" {
		date, _ := time.Parse("2006-01-02", dateStr)
		var isAvailable bool
		availQuery := `
			SELECT NOT EXISTS(
				SELECT 1 FROM driver_unavailability 
				WHERE driver_id = $1 AND $2 BETWEEN start_date AND end_date
			)
		`
		db.Get(&isAvailable, availQuery, driverID, date)
		if !isAvailable {
			conflicts = append(conflicts, RouteConflict{
				Type:        "driver_unavailable",
				Description: "Driver is marked as unavailable on this date",
				Severity:    "error",
				Details: map[string]interface{}{
					"date": dateStr,
				},
			})
		}
	}

	// Check driver hours limit
	var weeklyHours float64
	hoursQuery := `
		SELECT COALESCE(SUM(hours_worked), 0) 
		FROM driver_logs 
		WHERE driver = $1 AND date >= CURRENT_DATE - INTERVAL '7 days'
	`
	db.Get(&weeklyHours, hoursQuery, driverID)
	if weeklyHours > 40 {
		conflicts = append(conflicts, RouteConflict{
			Type:        "excessive_hours",
			Description: fmt.Sprintf("Driver has already worked %.1f hours this week", weeklyHours),
			Severity:    "warning",
			Details: map[string]interface{}{
				"weekly_hours": weeklyHours,
				"limit":        40,
			},
		})
	}

	return conflicts
}

// checkBusConflicts checks for bus scheduling conflicts
func checkBusConflicts(busID, period, dateStr string) []RouteConflict {
	conflicts := []RouteConflict{}

	// Check if bus is already assigned to another route in the same period
	var existingRoute string
	query := `
		SELECT route_id 
		FROM route_assignments 
		WHERE bus_id = $1 AND period = $2 AND active = true
	`
	err := db.Get(&existingRoute, query, busID, period)
	if err == nil {
		conflicts = append(conflicts, RouteConflict{
			Type:        "bus_already_assigned",
			Description: fmt.Sprintf("Bus is already assigned to route %s during %s period", existingRoute, period),
			Severity:    "error",
			Details: map[string]interface{}{
				"existing_route": existingRoute,
				"period":        period,
			},
		})
	}

	// Check bus status
	var busStatus string
	statusQuery := `SELECT status FROM buses WHERE bus_id = $1`
	db.Get(&busStatus, statusQuery, busID)
	if busStatus != "active" {
		conflicts = append(conflicts, RouteConflict{
			Type:        "bus_not_available",
			Description: fmt.Sprintf("Bus is currently %s", busStatus),
			Severity:    "error",
			Details: map[string]interface{}{
				"status": busStatus,
			},
		})
	}

	return conflicts
}

// checkRouteSpecificConflicts checks for route-specific issues
func checkRouteSpecificConflicts(routeID, period string) []RouteConflict {
	conflicts := []RouteConflict{}

	// Check if route is already fully assigned
	var isAssigned bool
	query := `
		SELECT EXISTS(
			SELECT 1 FROM route_assignments 
			WHERE route_id = $1 AND period = $2 AND active = true
		)
	`
	db.Get(&isAssigned, query, routeID, period)
	if isAssigned {
		conflicts = append(conflicts, RouteConflict{
			Type:        "route_already_assigned",
			Description: "This route already has a driver and bus assigned",
			Severity:    "error",
			Details: map[string]interface{}{
				"route_id": routeID,
				"period":   period,
			},
		})
	}

	return conflicts
}

// checkDriverQualifications checks if driver is qualified for the route
func checkDriverQualifications(driverID, busID, routeID string) []RouteConflict {
	warnings := []RouteConflict{}

	// Check if driver has CDL for large buses
	var busCapacity int
	var hasCDL bool
	capacityQuery := `SELECT capacity FROM buses WHERE bus_id = $1`
	cdlQuery := `SELECT has_cdl FROM users WHERE username = $1`
	
	db.Get(&busCapacity, capacityQuery, busID)
	db.Get(&hasCDL, cdlQuery, driverID)

	if busCapacity > 15 && !hasCDL {
		warnings = append(warnings, RouteConflict{
			Type:        "cdl_required",
			Description: "This bus requires a CDL-certified driver",
			Severity:    "warning",
			Details: map[string]interface{}{
				"bus_capacity": busCapacity,
				"has_cdl":     hasCDL,
			},
		})
	}

	// Check if driver is trained for special needs routes
	var hasSpecialNeeds bool
	var driverSpecialTrained bool
	specialQuery := `
		SELECT EXISTS(
			SELECT 1 FROM students 
			WHERE route_id = $1 AND special_needs = true
		)
	`
	trainingQuery := `SELECT special_needs_trained FROM users WHERE username = $1`
	
	db.Get(&hasSpecialNeeds, specialQuery, routeID)
	db.Get(&driverSpecialTrained, trainingQuery, driverID)

	if hasSpecialNeeds && !driverSpecialTrained {
		warnings = append(warnings, RouteConflict{
			Type:        "special_training_recommended",
			Description: "This route has special needs students; driver should have special needs training",
			Severity:    "warning",
			Details: map[string]interface{}{
				"has_special_needs_students": hasSpecialNeeds,
				"driver_trained":            driverSpecialTrained,
			},
		})
	}

	return warnings
}

// checkBusCapacity checks if bus has sufficient capacity for the route
func checkBusCapacity(busID, routeID string) []RouteConflict {
	warnings := []RouteConflict{}

	var busCapacity int
	var studentCount int

	capacityQuery := `SELECT capacity FROM buses WHERE bus_id = $1`
	studentQuery := `SELECT COUNT(*) FROM students WHERE route_id = $1 AND active = true`

	db.Get(&busCapacity, capacityQuery, busID)
	db.Get(&studentCount, studentQuery, routeID)

	utilizationRate := float64(studentCount) / float64(busCapacity) * 100

	if studentCount > busCapacity {
		warnings = append(warnings, RouteConflict{
			Type:        "overcapacity",
			Description: fmt.Sprintf("Route has %d students but bus capacity is %d", studentCount, busCapacity),
			Severity:    "error",
			Details: map[string]interface{}{
				"student_count":    studentCount,
				"bus_capacity":     busCapacity,
				"utilization_rate": utilizationRate,
			},
		})
	} else if utilizationRate > 90 {
		warnings = append(warnings, RouteConflict{
			Type:        "high_utilization",
			Description: fmt.Sprintf("Bus will be at %.1f%% capacity", utilizationRate),
			Severity:    "warning",
			Details: map[string]interface{}{
				"student_count":    studentCount,
				"bus_capacity":     busCapacity,
				"utilization_rate": utilizationRate,
			},
		})
	}

	return warnings
}

// checkMaintenanceSchedule checks for upcoming maintenance
func checkMaintenanceSchedule(busID, dateStr string) []RouteConflict {
	warnings := []RouteConflict{}

	// Check for scheduled maintenance
	var nextMaintenance sql.NullTime
	maintenanceQuery := `
		SELECT MIN(scheduled_date) 
		FROM scheduled_maintenance 
		WHERE vehicle_id = $1 AND scheduled_date >= CURRENT_DATE AND completed = false
	`
	db.Get(&nextMaintenance, maintenanceQuery, busID)

	if nextMaintenance.Valid {
		daysUntil := int(nextMaintenance.Time.Sub(time.Now()).Hours() / 24)
		if daysUntil <= 7 {
			warnings = append(warnings, RouteConflict{
				Type:        "upcoming_maintenance",
				Description: fmt.Sprintf("Bus has scheduled maintenance in %d days", daysUntil),
				Severity:    "warning",
				Details: map[string]interface{}{
					"maintenance_date": nextMaintenance.Time.Format("2006-01-02"),
					"days_until":       daysUntil,
				},
			})
		}
	}

	// Check mileage-based maintenance
	var currentMileage, lastOilChange int
	mileageQuery := `SELECT current_mileage, last_oil_change FROM buses WHERE bus_id = $1`
	db.QueryRow(mileageQuery, busID).Scan(&currentMileage, &lastOilChange)

	milesSinceOil := currentMileage - lastOilChange
	if milesSinceOil > 4500 {
		warnings = append(warnings, RouteConflict{
			Type:        "maintenance_due",
			Description: fmt.Sprintf("Bus is due for oil change (%d miles since last change)", milesSinceOil),
			Severity:    "warning",
			Details: map[string]interface{}{
				"current_mileage":    currentMileage,
				"last_oil_change":    lastOilChange,
				"miles_since_change": milesSinceOil,
			},
		})
	}

	return warnings
}

// getRouteAssignmentSuggestionsHandler provides smart suggestions for route assignments
func getRouteAssignmentSuggestionsHandler(w http.ResponseWriter, r *http.Request) {
	routeID := r.URL.Query().Get("route_id")
	period := r.URL.Query().Get("period")

	suggestions := getRouteAssignmentSuggestions(routeID, period)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(suggestions)
}

// RouteAssignmentSuggestion represents a suggested driver-bus combination
type RouteAssignmentSuggestion struct {
	DriverID     string  `json:"driver_id"`
	DriverName   string  `json:"driver_name"`
	BusID        string  `json:"bus_id"`
	BusNumber    string  `json:"bus_number"`
	Score        float64 `json:"score"`
	Reasons      []string `json:"reasons"`
	HasConflicts bool    `json:"has_conflicts"`
}

// getRouteAssignmentSuggestions generates smart suggestions for route assignments
func getRouteAssignmentSuggestions(routeID, period string) []RouteAssignmentSuggestion {
	suggestions := []RouteAssignmentSuggestion{}

	// Get available drivers
	availableDrivers := getAvailableDrivers(period)
	
	// Get available buses
	availableBuses := getAvailableBuses(period)

	// Score each combination
	for _, driver := range availableDrivers {
		for _, bus := range availableBuses {
			score := calculateAssignmentScore(driver, bus, routeID, period)
			
			// Check for conflicts
			conflicts := checkRouteConflicts(driver.Username, bus.BusID, routeID, period, "")
			
			if score > 0.5 { // Only suggest combinations with decent scores
				suggestion := RouteAssignmentSuggestion{
					DriverID:     driver.Username,
					DriverName:   driver.Username,
					BusID:        bus.BusID,
					BusNumber:    bus.BusID,
					Score:        score,
					Reasons:      generateSuggestionReasons(driver, bus, routeID),
					HasConflicts: !conflicts.CanAssign,
				}
				suggestions = append(suggestions, suggestion)
			}
		}
	}

	// Sort by score (highest first)
	for i := 0; i < len(suggestions)-1; i++ {
		for j := i + 1; j < len(suggestions); j++ {
			if suggestions[j].Score > suggestions[i].Score {
				suggestions[i], suggestions[j] = suggestions[j], suggestions[i]
			}
		}
	}

	// Return top 5 suggestions
	if len(suggestions) > 5 {
		suggestions = suggestions[:5]
	}

	return suggestions
}

// Helper functions for suggestions
func getAvailableDrivers(period string) []User {
	var drivers []User
	query := `
		SELECT username, name, email, role 
		FROM users 
		WHERE role = 'driver' 
		AND username NOT IN (
			SELECT driver_id FROM route_assignments 
			WHERE period = $1 AND active = true
		)
	`
	db.Select(&drivers, query, period)
	return drivers
}

func getAvailableBuses(period string) []Bus {
	var buses []Bus
	query := `
		SELECT bus_id, status, capacity 
		FROM buses 
		WHERE status = 'active' 
		AND bus_id NOT IN (
			SELECT bus_id FROM route_assignments 
			WHERE period = $1 AND active = true
		)
	`
	db.Select(&buses, query, period)
	return buses
}

func calculateAssignmentScore(driver User, bus Bus, routeID, period string) float64 {
	score := 1.0

	// Factor in driver experience
	var tripCount int
	db.Get(&tripCount, "SELECT COUNT(*) FROM driver_logs WHERE driver = $1", driver.Username)
	if tripCount > 100 {
		score += 0.2
	}

	// Factor in previous route familiarity
	var familiarityCount int
	db.Get(&familiarityCount, 
		"SELECT COUNT(*) FROM driver_logs WHERE driver = $1 AND route_id = $2", 
		driver.Username, routeID)
	if familiarityCount > 0 {
		score += 0.3
	}

	// Factor in bus capacity match
	var studentCount int
	db.Get(&studentCount, "SELECT COUNT(*) FROM students WHERE route_id = $1 AND active = true", routeID)
	capacityRatio := float64(studentCount) / float64(bus.Capacity.Int32)
	if capacityRatio > 0.7 && capacityRatio < 0.9 {
		score += 0.2 // Good capacity match
	}

	return score
}

func generateSuggestionReasons(driver User, bus Bus, routeID string) []string {
	reasons := []string{}

	// Check experience
	var tripCount int
	db.Get(&tripCount, "SELECT COUNT(*) FROM driver_logs WHERE driver = $1", driver.Username)
	if tripCount > 100 {
		reasons = append(reasons, fmt.Sprintf("Experienced driver (%d trips)", tripCount))
	}

	// Check route familiarity
	var familiarityCount int
	db.Get(&familiarityCount, 
		"SELECT COUNT(*) FROM driver_logs WHERE driver = $1 AND route_id = $2", 
		driver.Username, routeID)
	if familiarityCount > 0 {
		reasons = append(reasons, fmt.Sprintf("Familiar with route (%d previous trips)", familiarityCount))
	}

	// Check capacity match
	var studentCount int
	db.Get(&studentCount, "SELECT COUNT(*) FROM students WHERE route_id = $1 AND active = true", routeID)
	reasons = append(reasons, fmt.Sprintf("Bus capacity: %d, Students: %d", bus.Capacity, studentCount))

	return reasons
}