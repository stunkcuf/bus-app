package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// SearchResult represents a search result for auto-complete
type SearchResult struct {
	Value       string                 `json:"value"`
	Label       string                 `json:"label"`
	Subtitle    string                 `json:"subtitle,omitempty"`
	Icon        string                 `json:"icon,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// searchBusesHandler handles bus search requests for auto-complete
func searchBusesHandler(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if query == "" {
		json.NewEncoder(w).Encode([]SearchResult{})
		return
	}
	
	// Load buses from cache or database
	buses, err := loadBusesFromDB()
	if err != nil {
		logError(&AppError{
			Type:       ErrorTypeDatabase,
			Message:    "Failed to load buses for search",
			Detail:     err.Error(),
			StatusCode: http.StatusInternalServerError,
			Internal:   err,
		})
		http.Error(w, "Failed to search buses", http.StatusInternalServerError)
		return
	}
	
	var results []SearchResult
	queryLower := strings.ToLower(query)
	
	for _, bus := range buses {
		// Search in bus ID and model
		if strings.Contains(strings.ToLower(bus.BusID), queryLower) ||
		   strings.Contains(strings.ToLower(bus.GetModel()), queryLower) {
			
			status := bus.Status
			if status == "" {
				status = "unknown"
			}
			
			results = append(results, SearchResult{
				Value:    bus.BusID,
				Label:    fmt.Sprintf("%s - %s", bus.BusID, bus.GetModel()),
				Subtitle: fmt.Sprintf("Status: %s, Capacity: %d", status, bus.GetCapacity()),
				Icon:     "bi-bus-front",
				Metadata: map[string]interface{}{
					"status":   status,
					"capacity": bus.GetCapacity(),
					"model":    bus.GetModel(),
				},
			})
		}
		
		// Limit results
		if len(results) >= 10 {
			break
		}
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

// searchDriversHandler handles driver search requests for auto-complete
func searchDriversHandler(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if query == "" {
		json.NewEncoder(w).Encode([]SearchResult{})
		return
	}
	
	// Load drivers
	drivers, err := loadUsersFromDB()
	if err != nil {
		logError(&AppError{
			Type:       ErrorTypeDatabase,
			Message:    "Failed to load drivers for search",
			Detail:     err.Error(),
			StatusCode: http.StatusInternalServerError,
			Internal:   err,
		})
		http.Error(w, "Failed to search drivers", http.StatusInternalServerError)
		return
	}
	
	// Load assignments to show current routes
	assignments, _ := loadRouteAssignments()
	assignmentMap := make(map[string][]string)
	for _, assignment := range assignments {
		assignmentMap[assignment.Driver] = append(assignmentMap[assignment.Driver], assignment.RouteName)
	}
	
	var results []SearchResult
	queryLower := strings.ToLower(query)
	
	for _, driver := range drivers {
		if driver.Role == "driver" && driver.Status == "active" &&
		   strings.Contains(strings.ToLower(driver.Username), queryLower) {
			
			routes := assignmentMap[driver.Username]
			routeStr := "No routes assigned"
			if len(routes) > 0 {
				routeStr = "Routes: " + strings.Join(routes, ", ")
			}
			
			results = append(results, SearchResult{
				Value:    driver.Username,
				Label:    driver.Username,
				Subtitle: routeStr,
				Icon:     "bi-person-circle",
				Metadata: map[string]interface{}{
					"routes": routes,
					"status": driver.Status,
				},
			})
		}
		
		// Limit results
		if len(results) >= 10 {
			break
		}
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

// searchStudentsHandler handles student search requests for auto-complete
func searchStudentsHandler(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if query == "" {
		json.NewEncoder(w).Encode([]SearchResult{})
		return
	}
	
	// Load students
	students, err := loadStudentsFromDB()
	if err != nil {
		logError(&AppError{
			Type:       ErrorTypeDatabase,
			Message:    "Failed to load students for search",
			Detail:     err.Error(),
			StatusCode: http.StatusInternalServerError,
			Internal:   err,
		})
		http.Error(w, "Failed to search students", http.StatusInternalServerError)
		return
	}
	
	var results []SearchResult
	queryLower := strings.ToLower(query)
	
	for _, student := range students {
		if strings.Contains(strings.ToLower(student.Name), queryLower) ||
		   strings.Contains(strings.ToLower(student.StudentID), queryLower) {
			
			status := "Active"
			if !student.Active {
				status = "Inactive"
			}
			
			results = append(results, SearchResult{
				Value:    student.StudentID,
				Label:    student.Name,
				Subtitle: fmt.Sprintf("ID: %s, Route: %s, %s", student.StudentID, student.RouteID, status),
				Icon:     "bi-person-badge",
				Metadata: map[string]interface{}{
					"routeId":  student.RouteID,
					"guardian": student.Guardian,
					"phone":    student.PhoneNumber,
					"active":   student.Active,
				},
			})
		}
		
		// Limit results
		if len(results) >= 10 {
			break
		}
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

// searchAddressesHandler handles address search requests for auto-complete
func searchAddressesHandler(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	addressType := r.URL.Query().Get("type") // "pickup" or "dropoff"
	
	if query == "" || len(query) < 3 {
		json.NewEncoder(w).Encode([]SearchResult{})
		return
	}
	
	// Common addresses - in production, this would query a database or geocoding API
	commonAddresses := []string{
		"123 Main Street",
		"456 Oak Avenue",
		"789 Elm Drive",
		"321 Pine Road",
		"654 Maple Lane",
		"987 Cedar Court",
		"111 School Street",
		"222 Park Avenue",
		"333 Church Road",
		"444 Market Street",
		"555 River Drive",
		"666 Lake Boulevard",
		"777 Hill Street",
		"888 Valley Road",
		"999 Forest Lane",
	}
	
	// School locations for dropoff
	schoolLocations := []string{
		"Washington Elementary School - Main Entrance",
		"Washington Elementary School - Bus Loop",
		"Lincoln Middle School - Front Gate",
		"Lincoln Middle School - Side Entrance",
		"Roosevelt High School - Main Entrance",
		"Roosevelt High School - Athletic Field Gate",
		"Community Center - Main Parking Lot",
		"Public Library - Front Steps",
		"City Park - North Entrance",
	}
	
	// Combine addresses based on type
	allAddresses := commonAddresses
	if addressType == "dropoff" {
		allAddresses = append(allAddresses, schoolLocations...)
	}
	
	var results []SearchResult
	queryLower := strings.ToLower(query)
	
	// Also get addresses from existing students for suggestions
	students, _ := loadStudentsFromDB()
	uniqueAddresses := make(map[string]bool)
	
	for range students {
		// Extract addresses from locations (JSONB field)
		// For now, we'll skip this as it requires parsing JSONB
	}
	
	// Add unique student addresses to search pool
	for addr := range uniqueAddresses {
		allAddresses = append(allAddresses, addr)
	}
	
	// Search and build results
	for _, addr := range allAddresses {
		if strings.Contains(strings.ToLower(addr), queryLower) {
			icon := "bi-geo-alt"
			if addressType == "dropoff" && strings.Contains(addr, "School") {
				icon = "bi-building"
			}
			
			results = append(results, SearchResult{
				Value:    addr,
				Label:    addr,
				Subtitle: strings.Title(addressType) + " Location",
				Icon:     icon,
				Metadata: map[string]interface{}{
					"type": addressType,
				},
			})
		}
		
		// Limit results
		if len(results) >= 10 {
			break
		}
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

// suggestModelsHandler handles vehicle model suggestions for auto-complete
func suggestModelsHandler(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	
	// Common bus models
	busModels := []struct {
		Make  string
		Model string
		Type  string
	}{
		{"Blue Bird", "Vision", "Full Size"},
		{"Blue Bird", "All American FE", "Full Size"},
		{"Blue Bird", "All American RE", "Full Size"},
		{"Blue Bird", "Micro Bird G5", "Mini Bus"},
		{"Thomas Built", "Saf-T-Liner C2", "Full Size"},
		{"Thomas Built", "Saf-T-Liner HDX", "Full Size"},
		{"Thomas Built", "Minotour", "Mini Bus"},
		{"IC Bus", "CE Series", "Full Size"},
		{"IC Bus", "RE Series", "Full Size"},
		{"IC Bus", "AE Series", "Activity Bus"},
		{"Ford", "Transit 350", "Van"},
		{"Ford", "E-450", "Cutaway"},
		{"Chevrolet", "Express 4500", "Cutaway"},
		{"Mercedes-Benz", "Sprinter 2500", "Van"},
		{"Mercedes-Benz", "Sprinter 3500", "Van"},
		{"Freightliner", "FS-65", "Full Size"},
		{"Collins", "NexBus", "Mini Bus"},
		{"Collins", "Low Floor", "Special Needs"},
	}
	
	var results []SearchResult
	queryLower := strings.ToLower(query)
	
	for _, model := range busModels {
		fullName := fmt.Sprintf("%s %s", model.Make, model.Model)
		if query == "" || strings.Contains(strings.ToLower(fullName), queryLower) ||
		   strings.Contains(strings.ToLower(model.Make), queryLower) ||
		   strings.Contains(strings.ToLower(model.Model), queryLower) {
			
			results = append(results, SearchResult{
				Value:    fullName,
				Label:    fullName,
				Subtitle: fmt.Sprintf("%s - %s", model.Make, model.Type),
				Icon:     "bi-bus-front",
				Metadata: map[string]interface{}{
					"make":  model.Make,
					"model": model.Model,
					"type":  model.Type,
				},
			})
		}
		
		// Limit results
		if len(results) >= 10 {
			break
		}
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}