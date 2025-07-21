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
	
	// Load buses directly
	buses, busErr := loadBusesFromDB()
	if busErr != nil {
		log.Printf("ERROR loading buses: %v", busErr)
	} else {
		log.Printf("SUCCESS: Loaded %d buses", len(buses))
		// Convert buses to ConsolidatedVehicle
		for _, bus := range buses {
			cv := ConsolidatedVehicle{
				ID:               bus.ID,
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

	// Load vehicles directly
	vehicles, vehErr := loadVehiclesFromDB()
	if vehErr != nil {
		log.Printf("ERROR loading vehicles: %v", vehErr)
	} else {
		log.Printf("SUCCESS: Loaded %d vehicles", len(vehicles))
		// Convert vehicles to ConsolidatedVehicle
		for _, veh := range vehicles {
			// Safe status handling
			status := "active"
			if veh.Status.Valid {
				status = veh.Status.String
			}
			
			cv := ConsolidatedVehicle{
				ID:               veh.ID,
				VehicleID:        veh.VehicleID,
				BusID:            veh.VehicleID,
				VehicleType:      "vehicle",
				Status:           status,
				Model:            veh.Model,
				Year:             veh.Year,
				TireSize:         veh.TireSize,
				License:          veh.License,
				OilStatus:        veh.OilStatus,
				TireStatus:       veh.TireStatus,
				Description:      veh.Description,
				SerialNumber:     veh.SerialNumber,
				Base:             veh.Base,
				ServiceInterval:  veh.ServiceInterval,
				MaintenanceNotes: veh.MaintenanceNotes,
				UpdatedAt:        veh.UpdatedAt,
				CreatedAt:        veh.CreatedAt,
			}
			
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

	// Create dummy pagination
	pagination := PaginationParams{
		Page:       1,
		PerPage:    len(allVehicles),
		TotalPages: 1,
		Offset:     0,
		HasPrev:    false,
		HasNext:    false,
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
	// CHANGED FROM fleet_modern.html TO fleet.html
	renderTemplate(w, r, "fleet.html", data)
}
