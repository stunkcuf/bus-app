// handlers.go - Specific changes needed

// 1. At the top of the file, ensure these imports are present:
package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
	"github.com/xuri/excelize/v2"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

// 2. REMOVE these lines (around line 28-40):
// DELETE THIS ENTIRE BLOCK:
/*
var (
	sessions = make(map[string]*Session)
	mu       sync.RWMutex
)

// Session represents a user session
type Session struct {
	Username string
	Role     string
	Expires  time.Time
}
*/

// 3. In the fleetHandler function (around line 456), replace:
func fleetHandler(w http.ResponseWriter, r *http.Request) {
	if !isLoggedIn(r) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	user := getUser(r)
	if user.Role != "manager" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	buses := loadBusesFromCache()
	
	// Get recent maintenance logs
	maintenanceLogs := []MaintenanceRecord{} // Changed from 'var maintenanceLogs []MaintenanceRecord'
	rows, err := db.Query(`
		SELECT vehicle_id, date, category, mileage, cost, notes, created_at
		FROM maintenance_logs
		WHERE vehicle_id LIKE 'BUS%'
		ORDER BY created_at DESC
		LIMIT 10
	`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var log MaintenanceRecord
			err := rows.Scan(&log.VehicleID, &log.Date, &log.Category, 
				&log.Mileage, &log.Cost, &log.Notes, &log.CreatedAt)
			if err == nil {
				maintenanceLogs = append(maintenanceLogs, log)
			}
		}
	} else {
		log.Printf("Error loading maintenance logs: %v", err)
	}

	data := map[string]interface{}{
		"Buses":           buses,
		"MaintenanceLogs": maintenanceLogs,
		"Today":           time.Now().Format("2006-01-02"),
		"CSRFToken":       generateCSRFToken(),
	}
	executeTemplate(w, "fleet.html", data)
}

// 4. In the vehicleMaintenanceHandler function (around line 741), add this struct:
	// CHANGED: Create a simple anonymous struct that matches what the template expects
	data := struct {
		CSPNonce           string
		VehicleID          string
		IsBus              bool
		MaintenanceRecords []MaintenanceRecord
		TotalRecords       int
		TotalCost          float64
		AverageCost        float64
		RecentCount        int
		Today              string
		CSRFToken          string
	}{
		CSPNonce:           generateCSPNonce(), // This function needs to be added
		VehicleID:          vehicleID,
		IsBus:              isBus,
		MaintenanceRecords: records,
		TotalRecords:       totalRecords,
		TotalCost:          totalCost,
		AverageCost:        averageCost,
		RecentCount:        recentCount,
		Today:              time.Now().Format("2006-01-02"),
		CSRFToken:          generateCSRFToken(),
	}

// 5. Add these functions at the end of the file (before the last closing brace):

// generateCSPNonce generates a CSP nonce for inline scripts
func generateCSPNonce() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return base64.StdEncoding.EncodeToString(b)
}

// viewEnhancedMileageReportsHandler is an alias for viewMileageReportsHandler
func viewEnhancedMileageReportsHandler(w http.ResponseWriter, r *http.Request) {
	viewMileageReportsHandler(w, r)
}

// getUser gets the current user from session
func getUser(r *http.Request) *User {
	return getUserFromSession(r)
}

// isLoggedIn checks if user has valid session
func isLoggedIn(r *http.Request) bool {
	cookie, err := r.Cookie("session")
	if err != nil {
		return false
	}

	session, err := GetSecureSession(cookie.Value)
	if err != nil {
		return false
	}

	return session != nil
}
