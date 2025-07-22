// fleet_handler_clean.go
package main

import (
	"fmt"
	"log"
	"net/http"
)

// Clean fleet handler that replaces the broken one
func fleetHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	log.Printf("DEBUG: Fleet handler called by user: %s", user.Username)

	// Initialize all variables upfront
	allVehicles := []ConsolidatedVehicle{}
	vehiclesByType := make(map[string][]ConsolidatedVehicle)
	allAlerts := make(map[string][]MaintenanceAlert) 
	
	// Get total counts for pagination
	busCount, busCountErr := getBusCount()
	if busCountErr != nil {
		log.Printf("ERROR getting bus count: %v", busCountErr)
		busCount = 0
	}
	log.Printf("DEBUG: Total bus count in database: %d", busCount)
	totalCount := busCount // We'll add vehicle count later
	
	// Get pagination params (default 50 per page for fleet view)
	pagination := GetPaginationParams(r, totalCount, 50)
	log.Printf("DEBUG: Pagination - Page: %d, PerPage: %d, Offset: %d, TotalPages: %d", 
		pagination.Page, pagination.PerPage, pagination.Offset, pagination.TotalPages)
	
	// Load buses with pagination
	log.Printf("DEBUG: About to call loadBusesFromDBPaginated with pagination: page=%d, perPage=%d, offset=%d", 
		pagination.Page, pagination.PerPage, pagination.Offset)
	buses, busErr := loadBusesFromDBPaginated(pagination)
	if busErr != nil {
		log.Printf("ERROR loading buses: %v", busErr)
	} else {
		log.Printf("SUCCESS: Loaded %d buses (page %d of %d)", len(buses), pagination.Page, pagination.TotalPages)
		for i, bus := range buses {
			log.Printf("DEBUG: Bus[%d]: ID=%s, Model=%v, Status=%s", i, bus.BusID, bus.Model, bus.Status)
		}
		if len(buses) > 0 {
			log.Printf("DEBUG: First bus: ID=%s, Status=%s", buses[0].BusID, buses[0].Status)
		}
		// Convert buses to ConsolidatedVehicle
		for _, bus := range buses {
			cv := ConsolidatedVehicle{
				ID:               bus.BusID,
				VehicleID:        bus.BusID,
				BusID:            bus.BusID,
				VehicleType:      "bus",
				Status:           bus.Status,
				Model:            bus.Model,
				Capacity:         bus.Capacity,
				OilStatus:        bus.OilStatus,
				TireStatus:       bus.TireStatus,
				MaintenanceNotes: bus.MaintenanceNotes,
				UpdatedAt:        bus.UpdatedAt,
				CreatedAt:        bus.CreatedAt,
			}
			
			// Try to get assignment but don't fail
			func() {
				defer func() {
					if r := recover(); r != nil {
						log.Printf("WARNING: Failed to get assignment for bus %s: %v", bus.BusID, r)
					}
				}()
				cv.Assignment = getVehicleAssignment(bus.BusID)
			}()
			
			allVehicles = append(allVehicles, cv)
			vehiclesByType["bus"] = append(vehiclesByType["bus"], cv)
		}
	}

	// Get vehicle count and update total
	vehicleCount, _ := getVehicleCount()
	totalCount = busCount + vehicleCount
	
	// Update pagination with combined total
	pagination.TotalItems = totalCount
	pagination.TotalPages = (totalCount + pagination.PerPage - 1) / pagination.PerPage
	
	// Load vehicles using the consolidated approach that works with proper null handling
	var vehErr error
	consolidatedVehicles, vehErr := loadConsolidatedNonBusVehiclesFromDB()
	if vehErr != nil {
		log.Printf("ERROR loading vehicles: %v", vehErr)
	} else {
		log.Printf("SUCCESS: Loaded %d vehicles", len(consolidatedVehicles))
		
		// Apply pagination to vehicles if needed
		// For combined view, we need to calculate which vehicles to show
		// based on buses already shown
		remainingSlots := pagination.PerPage - len(buses)
		if remainingSlots > 0 && pagination.Page == 1 {
			// Show first N vehicles to fill the remaining slots on page 1
			end := remainingSlots
			if end > len(consolidatedVehicles) {
				end = len(consolidatedVehicles)
			}
			consolidatedVehicles = consolidatedVehicles[:end]
		} else if pagination.Page > 1 {
			// Calculate vehicle offset for subsequent pages
			vehicleOffset := 0
			if pagination.Page > 1 {
				// Account for vehicles shown on previous pages
				vehicleOffset = (pagination.Page-1)*pagination.PerPage - busCount
				if vehicleOffset < 0 {
					vehicleOffset = 0
				}
			}
			
			// Apply pagination to vehicles
			end := vehicleOffset + pagination.PerPage
			if vehicleOffset >= len(consolidatedVehicles) {
				consolidatedVehicles = []ConsolidatedVehicle{}
			} else {
				if end > len(consolidatedVehicles) {
					end = len(consolidatedVehicles)
				}
				consolidatedVehicles = consolidatedVehicles[vehicleOffset:end]
			}
		} else {
			// Page 1 and buses filled the entire page, no vehicles to show
			consolidatedVehicles = []ConsolidatedVehicle{}
		}
		
		// Add vehicles to the combined list
		for _, cv := range consolidatedVehicles {
			allVehicles = append(allVehicles, cv)
			vehiclesByType["vehicle"] = append(vehiclesByType["vehicle"], cv)
		}
	}

	// Check if we have any data
	if len(allVehicles) == 0 {
		log.Printf("ERROR: No vehicles loaded!")
		data := map[string]interface{}{
			"User":         user,
			"Error":        "No fleet data available",
			"ErrorDetails": fmt.Sprintf("Bus error: %v<br>Vehicle error: %v", busErr, vehErr),
			"CSRFToken":    getSessionCSRFToken(r),
		}
		renderTemplate(w, r, "error.html", data)
		return
	}

	log.Printf("SUCCESS: Total vehicles loaded: %d", len(allVehicles))

	// Calculate statistics for ALL vehicles (buses + vehicles)
	activeCount := 0
	maintenanceCount := 0
	outOfServiceCount := 0
	
	// Count oil status issues
	oilOverdueCount := 0
	oilDueCount := 0
	
	for _, vehicle := range allVehicles {
		// Count by status
		switch vehicle.Status {
		case "active":
			activeCount++
		case "maintenance":
			maintenanceCount++
		case "out-of-service", "out_of_service":
			outOfServiceCount++
		default:
			// Log unknown status but still count as active
			log.Printf("Unknown status '%s' for vehicle %s", vehicle.Status, vehicle.VehicleID)
			activeCount++
		}
		
		// Count oil status issues
		if vehicle.OilStatus.Valid {
			switch vehicle.OilStatus.String {
			case "overdue":
				oilOverdueCount++
			case "due":
				oilDueCount++
			}
		}
		
		// If vehicle has overdue oil or tires, it should be in maintenance
		if (vehicle.OilStatus.Valid && vehicle.OilStatus.String == "overdue") || 
		   (vehicle.TireStatus.Valid && vehicle.TireStatus.String == "overdue") {
			if vehicle.Status == "active" {
				oilStr := ""
				if vehicle.OilStatus.Valid {
					oilStr = vehicle.OilStatus.String
				}
				tireStr := ""
				if vehicle.TireStatus.Valid {
					tireStr = vehicle.TireStatus.String
				}
				log.Printf("WARNING: Vehicle %s is active but has overdue maintenance (oil: %s, tire: %s)",
					vehicle.VehicleID, oilStr, tireStr)
			}
		}
	}
	
	// Calculate bus-specific stats
	busesSlice := vehiclesByType["bus"]
	if busesSlice == nil {
		busesSlice = []ConsolidatedVehicle{}
	}
	log.Printf("DEBUG: busesSlice has %d items for template", len(busesSlice))
	
	activeBuses := 0
	maintenanceBuses := 0
	outOfServiceBuses := 0
	
	for _, bus := range busesSlice {
		switch bus.Status {
		case "active":
			activeBuses++
		case "maintenance":
			maintenanceBuses++
		case "out-of-service", "out_of_service":
			outOfServiceBuses++
		default:
			activeBuses++
		}
	}

	// Update pagination with actual page numbers
	pagination.PageNumbers = generatePageNumbers(pagination.Page, pagination.TotalPages)
	pagination.StartItem = pagination.Offset + 1
	pagination.EndItem = pagination.Offset + len(allVehicles)
	if pagination.EndItem > pagination.TotalItems {
		pagination.EndItem = pagination.TotalItems
	}
	if pagination.StartItem > pagination.TotalItems {
		pagination.StartItem = pagination.TotalItems
	}

	// Calculate totals
	totalVehicles := len(allVehicles)
	totalBuses := len(busesSlice)
	totalOtherVehicles := totalVehicles - totalBuses

	// Also try to load fleet vehicles for backward compatibility
	var fleetVehicles []FleetVehicle
	db.Select(&fleetVehicles, "SELECT * FROM fleet_vehicles LIMIT 100")

	// Prepare data for template
	data := map[string]interface{}{
		"User":               user,
		"CSRFToken":          getSessionCSRFToken(r),
		"Pagination":         pagination,
		"Buses":              busesSlice,
		"AllBuses":           busesSlice,
		"AllVehicles":        allVehicles,
		"VehiclesByType":     vehiclesByType,
		"MaintenanceAlerts":  allAlerts,
		"ActiveBuses":        activeBuses,
		"MaintenanceBuses":   maintenanceBuses,
		"OutOfServiceBuses":  outOfServiceBuses,
		"TotalVehicles":      totalVehicles,
		"TotalBuses":         totalBuses,
		"TotalOtherVehicles": totalOtherVehicles,
		"FleetVehicles":      fleetVehicles,
		// Overall fleet statistics
		"ActiveCount":        activeCount,
		"MaintenanceCount":   maintenanceCount,
		"OutOfServiceCount":  outOfServiceCount,
		"OilOverdueCount":    oilOverdueCount,
		"OilDueCount":        oilDueCount,
	}

	log.Printf("DEBUG: Rendering fleet.html with %d vehicles", totalVehicles)
	log.Printf("DEBUG: Template data - Buses: %d items, AllVehicles: %d items", len(busesSlice), len(allVehicles))
	log.Printf("DEBUG: Stats - Active: %d, Maintenance: %d, OutOfService: %d", activeBuses, maintenanceBuses, outOfServiceBuses)
	
	// Debug: Check if Buses key is correctly set
	if dataB, ok := data["Buses"].([]ConsolidatedVehicle); ok {
		log.Printf("DEBUG: data[\"Buses\"] is correctly set with %d items", len(dataB))
	} else {
		log.Printf("DEBUG: data[\"Buses\"] type issue or not set correctly")
	}
	
	log.Printf("DEBUG: Before rendering - Data.Buses count: %d", len(busesSlice))
	log.Printf("DEBUG: Before rendering - Data.AllVehicles count: %d", len(allVehicles))
	if len(busesSlice) > 0 {
		log.Printf("DEBUG: First bus in data: ID=%s", busesSlice[0].BusID)
	}
	// CHANGED FROM fleet_modern.html TO fleet.html
	renderTemplate(w, r, "fleet.html", data)
}
