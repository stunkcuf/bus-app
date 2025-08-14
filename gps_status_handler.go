package main

import (
	"encoding/json"
	"log"
	"net/http"
)

// gpsStatusHandler returns the current GPS tracking status
func gpsStatusHandler(w http.ResponseWriter, r *http.Request) {
	// Check if GPS is enabled
	enabled, err := isGPSEnabled()
	if err != nil {
		// Log the error for debugging
		log.Printf("Error checking GPS status: %v", err)
		// If error or setting doesn't exist, default to disabled
		enabled = false
	}

	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"enabled": enabled,
		"status":  "operational",
	}
	
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding GPS status response: %v", err)
	}
}