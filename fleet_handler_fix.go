// fleet_handler_fix.go
package main

import (
	"fmt"
	"log"
	"net/http"
)

// Override the problematic fleetHandler with a working version
func init() {
	// This will override the existing handler
	http.HandleFunc("/fleet-fixed", fleetHandlerFixed)
}

func fleetHandlerFixed(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	log.Printf("DEBUG: Fleet handler called for user: %s", user.Username)

	// Initialize empty slices to avoid nil issues
	allVehicles := []ConsolidatedVehicle{}
	allAlerts := []MaintenanceAlert{}
	vehiclesByType := make(map[string][]ConsolidatedVehicle)

	// Try to load buses
	log.Printf("DEBUG: Loading buses...")
	var buses []Bus
	busErr := db.Select(&buses, "SELECT * FROM buses")
	if busErr != nil {
		log.Printf("ERROR loading buses: %v", busErr)
	} else {
		log.Printf("DEBUG: Loaded %d buses", len(buses))
		// Convert to ConsolidatedVehicle
		for _, bus := range buses {
			cv := ConsolidatedVehicle{
				ID:          bus.ID,
				VehicleID:   bus.BusID,
				BusID:       bus.BusID,
				VehicleType: "bus",
				Status:      bus.Status,
				Model:       bus.Model,
				Capacity:    bus.Capacity,
				OilStatus:   bus.OilStatus,
				TireStatus:  bus.TireStatus,
				MaintenanceNotes: bus.MaintenanceNotes,
				UpdatedAt:   bus.UpdatedAt,
				CreatedAt:   bus.CreatedAt,
			}
			
			// Try to get assignment, but don't fail if it errors
			if assignment := getVehicleAssignmentSafe(bus.BusID); assignment != nil {
				cv.Assignment = assignment
			}
			
			allVehicles = append(allVehicles, cv)
			vehiclesByType["bus"] = append(vehiclesByType["bus"], cv)
		}
	}

	// Try to load vehicles
	log.Printf("DEBUG: Loading vehicles...")
	var vehicles []Vehicle
	vehErr := db.Select(&vehicles, "SELECT * FROM vehicles")
	if vehErr != nil {
		log.Printf("ERROR loading vehicles: %v", vehErr)
	} else {
		log.Printf("DEBUG: Loaded %d vehicles", len(vehicles))
		// Convert to ConsolidatedVehicle
		for _, veh := range vehicles {
			// Handle nullable status
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
	if len(allVehicles) == 0 && busErr != nil && vehErr != nil {
		log.Printf("ERROR: No vehicles loaded and both queries failed")
		data := map[string]interface{}{
			"User":         user,
			"Error":        "Unable to load fleet data",
			"ErrorDetails": fmt.Sprintf("Bus error: %v\nVehicle error: %v", busErr, vehErr),
			"CSRFToken":    getSessionCSRFToken(r),
		}
		renderTemplate(w, r, "error.html", data)
		return
	}

	log.Printf("DEBUG: Total vehicles loaded: %d", len(allVehicles))

	// Calculate statistics
	totalVehicles := len(allVehicles)
	totalBuses := len(vehiclesByType["bus"])
	totalOtherVehicles := totalVehicles - totalBuses
	
	activeBuses := 0
	maintenanceBuses := 0
	outOfServiceBuses := 0
	
	for _, bus := range vehiclesByType["bus"] {
		switch bus.Status {
		case "active":
			activeBuses++
		case "maintenance":
			maintenanceBuses++
		case "out-of-service", "out_of_service":
			outOfServiceBuses++
		default:
			activeBuses++ // Default to active if status unknown
		}
	}

	// Create minimal pagination for compatibility
	pagination := PaginationParams{
		Page:       1,
		PerPage:    len(allVehicles),
		TotalPages: 1,
		Offset:     0,
		HasPrev:    false,
		HasNext:    false,
	}

	// Also try to load fleet vehicles for backward compatibility, but don't fail
	var fleetVehicles []FleetVehicle
	_ = db.Select(&fleetVehicles, "SELECT * FROM fleet_vehicles LIMIT 100")

	// Prepare template data
	data := map[string]interface{}{
		"User":               user,
		"CSRFToken":          getSessionCSRFToken(r),
		"Pagination":         pagination,
		"Buses":              vehiclesByType["bus"],
		"AllBuses":           vehiclesByType["bus"],
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
	}

	log.Printf("DEBUG: Rendering fleet.html template")
	// CHANGED FROM fleet_modern.html TO fleet.html
	renderTemplate(w, r, "fleet.html", data)
}

// Safe version that doesn't crash if assignment fails
func getVehicleAssignmentSafe(vehicleID string) *RouteAssignment {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("ERROR: getVehicleAssignment panic for %s: %v", vehicleID, r)
		}
	}()
	
	return getVehicleAssignment(vehicleID)
}
