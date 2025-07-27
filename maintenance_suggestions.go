package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// MaintenanceSuggestion represents a maintenance recommendation
type MaintenanceSuggestion struct {
	Type        string    `json:"type"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Priority    string    `json:"priority"` // "high", "medium", "low"
	DueDate     time.Time `json:"due_date,omitempty"`
	Cost        float64   `json:"estimated_cost,omitempty"`
	Mileage     int       `json:"mileage_threshold,omitempty"`
	Icon        string    `json:"icon"`
}

// MaintenanceHistory represents past maintenance for pattern analysis
type MaintenanceHistory struct {
	ServiceType string    `json:"service_type"`
	LastDate    time.Time `json:"last_date"`
	LastMileage int       `json:"last_mileage"`
	AverageCost float64   `json:"average_cost"`
	Frequency   int       `json:"frequency"`
}

// getMaintenanceSuggestionsHandler returns maintenance suggestions for a vehicle
func getMaintenanceSuggestionsHandler(w http.ResponseWriter, r *http.Request) {
	vehicleType := r.URL.Query().Get("vehicle_type")
	vehicleID := r.URL.Query().Get("vehicle_id")

	if vehicleID == "" {
		http.Error(w, "Vehicle ID required", http.StatusBadRequest)
		return
	}

	suggestions := getMaintenanceSuggestions(vehicleType, vehicleID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(suggestions)
}

// getMaintenanceSuggestions generates smart maintenance suggestions
func getMaintenanceSuggestions(vehicleType, vehicleID string) []MaintenanceSuggestion {
	suggestions := []MaintenanceSuggestion{}

	// Get vehicle details
	var currentMileage int
	var lastOilChange int
	var lastTireRotation int
	// var inspectionDue time.Time - not used yet

	if vehicleType == "bus" {
		query := `SELECT current_mileage, last_oil_change, last_tire_rotation 
		          FROM buses WHERE bus_id = $1`
		db.QueryRow(query, vehicleID).Scan(&currentMileage, &lastOilChange, &lastTireRotation)
	} else {
		query := `SELECT 
			CASE WHEN current_mileage ~ '^\d+$' THEN current_mileage::INTEGER ELSE 0 END 
		FROM vehicles 
		WHERE vehicle_id = CONCAT('FV', $1::text) AND vehicle_type = 'fleet'`
		db.QueryRow(query, vehicleID).Scan(&currentMileage)
	}

	// Oil Change Suggestion
	milesSinceOilChange := currentMileage - lastOilChange
	if milesSinceOilChange > 4500 {
		suggestions = append(suggestions, MaintenanceSuggestion{
			Type:        "oil_change",
			Title:       "Oil Change Overdue",
			Description: fmt.Sprintf("Vehicle has traveled %d miles since last oil change (recommended: 5000 miles)", milesSinceOilChange),
			Priority:    "high",
			Cost:        75.00,
			Mileage:     currentMileage,
			Icon:        "🛢️",
		})
	} else if milesSinceOilChange > 4000 {
		suggestions = append(suggestions, MaintenanceSuggestion{
			Type:        "oil_change",
			Title:       "Oil Change Due Soon",
			Description: fmt.Sprintf("Vehicle has traveled %d miles since last oil change", milesSinceOilChange),
			Priority:    "medium",
			Cost:        75.00,
			Mileage:     currentMileage,
			Icon:        "🛢️",
		})
	}

	// Tire Rotation Suggestion
	milesSinceTireRotation := currentMileage - lastTireRotation
	if milesSinceTireRotation > 7500 {
		suggestions = append(suggestions, MaintenanceSuggestion{
			Type:        "tire_service",
			Title:       "Tire Rotation Overdue",
			Description: fmt.Sprintf("Vehicle has traveled %d miles since last tire rotation", milesSinceTireRotation),
			Priority:    "high",
			Cost:        50.00,
			Mileage:     currentMileage,
			Icon:        "🚙",
		})
	}

	// Check maintenance history for patterns
	history := getMaintenanceHistory(vehicleType, vehicleID)
	
	// Predict next maintenance based on history
	for _, hist := range history {
		suggestion := predictNextMaintenance(hist, currentMileage)
		if suggestion != nil {
			suggestions = append(suggestions, *suggestion)
		}
	}

	// Check for seasonal maintenance
	seasonalSuggestions := getSeasonalMaintenanceSuggestions(time.Now())
	suggestions = append(suggestions, seasonalSuggestions...)

	// Check for recalls or service bulletins
	recallSuggestions := checkVehicleRecalls(vehicleType, vehicleID)
	suggestions = append(suggestions, recallSuggestions...)

	// Sort by priority
	sortSuggestionsByPriority(suggestions)

	return suggestions
}

// getMaintenanceHistory retrieves maintenance history for pattern analysis
func getMaintenanceHistory(vehicleType, vehicleID string) []MaintenanceHistory {
	history := []MaintenanceHistory{}

	query := `
		SELECT 
			service_type,
			MAX(service_date) as last_date,
			MAX(mileage) as last_mileage,
			AVG(cost) as average_cost,
			COUNT(*) as frequency
		FROM maintenance_records
		WHERE vehicle_id = $1 AND vehicle_type = $2
		GROUP BY service_type
		HAVING COUNT(*) > 1
	`

	rows, err := db.Query(query, vehicleID, vehicleType)
	if err != nil {
		return history
	}
	defer rows.Close()

	for rows.Next() {
		var h MaintenanceHistory
		err := rows.Scan(&h.ServiceType, &h.LastDate, &h.LastMileage, &h.AverageCost, &h.Frequency)
		if err == nil {
			history = append(history, h)
		}
	}

	return history
}

// predictNextMaintenance predicts when maintenance will be due based on history
func predictNextMaintenance(history MaintenanceHistory, currentMileage int) *MaintenanceSuggestion {
	// Calculate average interval
	var avgInterval int
	var avgDays int

	// Get all maintenance records for this type
	query := `
		SELECT mileage, service_date 
		FROM maintenance_records 
		WHERE service_type = $1 
		ORDER BY service_date DESC 
		LIMIT 5
	`
	
	rows, err := db.Query(query, history.ServiceType)
	if err == nil {
		defer rows.Close()
		
		var prevMileage int
		var prevDate time.Time
		first := true
		intervals := []int{}
		dayIntervals := []int{}
		
		for rows.Next() {
			var mileage int
			var date time.Time
			rows.Scan(&mileage, &date)
			
			if !first {
				if prevMileage > 0 && mileage > 0 {
					intervals = append(intervals, prevMileage-mileage)
				}
				dayIntervals = append(dayIntervals, int(prevDate.Sub(date).Hours()/24))
			}
			
			prevMileage = mileage
			prevDate = date
			first = false
		}
		
		// Calculate averages
		if len(intervals) > 0 {
			sum := 0
			for _, interval := range intervals {
				sum += interval
			}
			avgInterval = sum / len(intervals)
		}
		
		if len(dayIntervals) > 0 {
			sum := 0
			for _, days := range dayIntervals {
				sum += days
			}
			avgDays = sum / len(dayIntervals)
		}
	}

	// Predict next maintenance
	if avgInterval > 0 {
		mileageUntilNext := (history.LastMileage + avgInterval) - currentMileage
		daysUntilNext := int(time.Since(history.LastDate).Hours()/24) - avgDays
		
		if mileageUntilNext < 500 || daysUntilNext > -30 {
			priority := "low"
			if mileageUntilNext < 200 || daysUntilNext > -7 {
				priority = "medium"
			}
			if mileageUntilNext < 0 || daysUntilNext > 0 {
				priority = "high"
			}
			
			return &MaintenanceSuggestion{
				Type:        history.ServiceType,
				Title:       fmt.Sprintf("%s Due Soon", formatServiceType(history.ServiceType)),
				Description: fmt.Sprintf("Based on history, this service is due in approximately %d miles", mileageUntilNext),
				Priority:    priority,
				Cost:        history.AverageCost,
				Mileage:     currentMileage + mileageUntilNext,
				Icon:        getServiceIcon(history.ServiceType),
			}
		}
	}

	return nil
}

// getSeasonalMaintenanceSuggestions returns maintenance based on season
func getSeasonalMaintenanceSuggestions(date time.Time) []MaintenanceSuggestion {
	suggestions := []MaintenanceSuggestion{}
	month := date.Month()

	// Winter prep (October-November)
	if month == time.October || month == time.November {
		suggestions = append(suggestions, MaintenanceSuggestion{
			Type:        "seasonal",
			Title:       "Winter Preparation",
			Description: "Check antifreeze levels, battery condition, and tire tread depth",
			Priority:    "medium",
			Cost:        100.00,
			Icon:        "❄️",
		})
	}

	// Spring maintenance (March-April)
	if month == time.March || month == time.April {
		suggestions = append(suggestions, MaintenanceSuggestion{
			Type:        "seasonal",
			Title:       "Spring Maintenance",
			Description: "Check air conditioning, replace windshield wipers, and inspect for winter damage",
			Priority:    "medium",
			Cost:        150.00,
			Icon:        "🌸",
		})
	}

	// Summer prep (May-June)
	if month == time.May || month == time.June {
		suggestions = append(suggestions, MaintenanceSuggestion{
			Type:        "seasonal",
			Title:       "Summer Preparation",
			Description: "Check cooling system, air conditioning, and tire pressure",
			Priority:    "low",
			Cost:        75.00,
			Icon:        "☀️",
		})
	}

	return suggestions
}

// checkVehicleRecalls checks for any recalls or service bulletins
func checkVehicleRecalls(vehicleType, vehicleID string) []MaintenanceSuggestion {
	suggestions := []MaintenanceSuggestion{}

	// In a real system, this would check against manufacturer databases
	// For now, we'll check a local recalls table
	query := `
		SELECT title, description, priority 
		FROM vehicle_recalls 
		WHERE vehicle_make IN (
			SELECT make FROM vehicles WHERE vehicle_id = CONCAT('FV', $1::text) AND vehicle_type = 'fleet'
		) AND resolved = false
	`

	if vehicleType == "bus" {
		query = `
			SELECT title, description, priority 
			FROM vehicle_recalls 
			WHERE vehicle_type = 'bus' AND resolved = false
		`
	}

	rows, err := db.Query(query, vehicleID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var title, description, priority string
			if err := rows.Scan(&title, &description, &priority); err == nil {
				suggestions = append(suggestions, MaintenanceSuggestion{
					Type:        "recall",
					Title:       title,
					Description: description,
					Priority:    priority,
					Cost:        0, // Recalls are typically free
					Icon:        "⚠️",
				})
			}
		}
	}

	return suggestions
}

// getMaintenanceAutocompleteHandler provides autocomplete for maintenance descriptions
func getMaintenanceAutocompleteHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	category := r.URL.Query().Get("category")

	suggestions := getMaintenanceAutocomplete(query, category)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(suggestions)
}

// AutocompleteSuggestion represents an autocomplete suggestion
type AutocompleteSuggestion struct {
	Value       string  `json:"value"`
	Label       string  `json:"label"`
	Category    string  `json:"category"`
	AverageCost float64 `json:"average_cost,omitempty"`
}

// getMaintenanceAutocomplete returns autocomplete suggestions for maintenance descriptions
func getMaintenanceAutocomplete(searchTerm, category string) []AutocompleteSuggestion {
	suggestions := []AutocompleteSuggestion{}

	// Common maintenance tasks by category
	commonTasks := map[string][]AutocompleteSuggestion{
		"oil_change": {
			{Value: "Oil and filter change", Label: "Oil and filter change", Category: "oil_change", AverageCost: 75},
			{Value: "Synthetic oil change", Label: "Synthetic oil change", Category: "oil_change", AverageCost: 95},
			{Value: "Oil change and inspection", Label: "Oil change and inspection", Category: "oil_change", AverageCost: 85},
		},
		"tire_service": {
			{Value: "Tire rotation", Label: "Tire rotation", Category: "tire_service", AverageCost: 50},
			{Value: "Tire replacement (all 4)", Label: "Tire replacement (all 4)", Category: "tire_service", AverageCost: 800},
			{Value: "Tire balancing", Label: "Tire balancing", Category: "tire_service", AverageCost: 60},
			{Value: "Tire repair", Label: "Tire repair", Category: "tire_service", AverageCost: 25},
			{Value: "Tire pressure check", Label: "Tire pressure check", Category: "tire_service", AverageCost: 0},
		},
		"inspection": {
			{Value: "Annual state inspection", Label: "Annual state inspection", Category: "inspection", AverageCost: 50},
			{Value: "Pre-trip inspection", Label: "Pre-trip inspection", Category: "inspection", AverageCost: 0},
			{Value: "Safety inspection", Label: "Safety inspection", Category: "inspection", AverageCost: 75},
			{Value: "DOT inspection", Label: "DOT inspection", Category: "inspection", AverageCost: 150},
		},
		"repair": {
			{Value: "Brake pad replacement", Label: "Brake pad replacement", Category: "repair", AverageCost: 300},
			{Value: "Battery replacement", Label: "Battery replacement", Category: "repair", AverageCost: 150},
			{Value: "Windshield wiper replacement", Label: "Windshield wiper replacement", Category: "repair", AverageCost: 30},
			{Value: "Air filter replacement", Label: "Air filter replacement", Category: "repair", AverageCost: 40},
			{Value: "Transmission fluid change", Label: "Transmission fluid change", Category: "repair", AverageCost: 150},
		},
	}

	// Filter by category if specified
	if category != "" && category != "other" {
		if tasks, ok := commonTasks[category]; ok {
			for _, task := range tasks {
				if searchTerm == "" || maintSuggestionsContains(task.Label, searchTerm) {
					suggestions = append(suggestions, task)
				}
			}
		}
	} else {
		// Search all categories
		for _, tasks := range commonTasks {
			for _, task := range tasks {
				if searchTerm == "" || maintSuggestionsContains(task.Label, searchTerm) {
					suggestions = append(suggestions, task)
				}
			}
		}
	}

	// Also get suggestions from historical data
	historicalSuggestions := getHistoricalMaintenanceSuggestions(searchTerm, category)
	suggestions = append(suggestions, historicalSuggestions...)

	// Limit to 10 suggestions
	if len(suggestions) > 10 {
		suggestions = suggestions[:10]
	}

	return suggestions
}

// getHistoricalMaintenanceSuggestions gets suggestions from past maintenance records
func getHistoricalMaintenanceSuggestions(searchTerm, category string) []AutocompleteSuggestion {
	suggestions := []AutocompleteSuggestion{}

	query := `
		SELECT DISTINCT description, service_type, AVG(cost) as avg_cost
		FROM maintenance_records
		WHERE description ILIKE $1
	`
	args := []interface{}{"%" + searchTerm + "%"}

	if category != "" {
		query += " AND service_type = $2"
		args = append(args, category)
	}

	query += " GROUP BY description, service_type ORDER BY COUNT(*) DESC LIMIT 5"

	rows, err := db.Query(query, args...)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var desc, serviceType string
			var avgCost float64
			if err := rows.Scan(&desc, &serviceType, &avgCost); err == nil {
				suggestions = append(suggestions, AutocompleteSuggestion{
					Value:       desc,
					Label:       desc,
					Category:    serviceType,
					AverageCost: avgCost,
				})
			}
		}
	}

	return suggestions
}

// Helper functions
func formatServiceType(serviceType string) string {
	switch serviceType {
	case "oil_change":
		return "Oil Change"
	case "tire_service":
		return "Tire Service"
	case "inspection":
		return "Inspection"
	case "repair":
		return "Repair"
	default:
		return "Maintenance"
	}
}

func getServiceIcon(serviceType string) string {
	switch serviceType {
	case "oil_change":
		return "🛢️"
	case "tire_service":
		return "🚙"
	case "inspection":
		return "🔍"
	case "repair":
		return "🔧"
	default:
		return "🔨"
	}
}

func sortSuggestionsByPriority(suggestions []MaintenanceSuggestion) {
	// Simple bubble sort by priority
	for i := 0; i < len(suggestions)-1; i++ {
		for j := 0; j < len(suggestions)-i-1; j++ {
			if priorityValue(suggestions[j].Priority) < priorityValue(suggestions[j+1].Priority) {
				suggestions[j], suggestions[j+1] = suggestions[j+1], suggestions[j]
			}
		}
	}
}

func priorityValue(priority string) int {
	switch priority {
	case "high":
		return 3
	case "medium":
		return 2
	case "low":
		return 1
	default:
		return 0
	}
}

func maintSuggestionsContains(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}