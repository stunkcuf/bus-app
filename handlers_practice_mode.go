package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"time"
)

// PracticeData represents the temporary practice data for a user session
type PracticeData struct {
	Buses        []Bus
	Routes       []Route
	Students     []Student
	Logs         []DriverLog
	Maintenance  []VehicleMaintenanceLog
	Assignments  []RouteAssignment
	IsActive     bool
	CreatedAt    time.Time
}

// Store practice data in memory (session-based)
var practiceDataStore = make(map[string]*PracticeData)

func practiceModeHandler(w http.ResponseWriter, r *http.Request) {
	session := getUserFromSession(r)
	if session == nil {
		log.Printf("Practice mode access without login")
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// Check if practice mode is already active for this user
	sessionID := getSessionID(r)
	practiceData, exists := practiceDataStore[sessionID]

	switch r.Method {
	case "GET":
		// Show practice mode page
		data := struct {
			Title        string
			Username     string
			UserType     string
			CSPNonce     string
			IsActive     bool
			PracticeData *PracticeData
		}{
			Title:        "Practice Mode",
			Username:     session.Username,
			UserType:     session.Role,
			CSPNonce:     generateNonce(),
			IsActive:     exists && practiceData != nil && practiceData.IsActive,
			PracticeData: practiceData,
		}

		tmpl := template.Must(template.ParseFiles("templates/practice_mode.html"))
		if err := tmpl.Execute(w, data); err != nil {
			log.Printf("Error rendering practice mode: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

	case "POST":
		// Handle practice mode actions
		action := r.FormValue("action")

		switch action {
		case "start":
			// Generate practice data
			practiceData = generatePracticeData(session.Role)
			practiceDataStore[sessionID] = practiceData

			// Set a practice mode flag in the session
			http.SetCookie(w, &http.Cookie{
				Name:     "practice_mode",
				Value:    "active",
				Path:     "/",
				HttpOnly: true,
				Secure:   true,
				SameSite: http.SameSiteStrictMode,
				MaxAge:   3600, // 1 hour
			})

			// Redirect to appropriate dashboard
			if session.Role == "manager" {
				http.Redirect(w, r, "/manager-dashboard?practice=1", http.StatusSeeOther)
			} else {
				http.Redirect(w, r, "/driver-dashboard?practice=1", http.StatusSeeOther)
			}

		case "stop":
			// Clear practice data
			delete(practiceDataStore, sessionID)

			// Clear practice mode cookie
			http.SetCookie(w, &http.Cookie{
				Name:     "practice_mode",
				Value:    "",
				Path:     "/",
				HttpOnly: true,
				Secure:   true,
				SameSite: http.SameSiteStrictMode,
				MaxAge:   -1,
			})

			// Redirect to practice mode page
			http.Redirect(w, r, "/practice-mode", http.StatusSeeOther)
		}
	}
}

// generatePracticeData creates sample data based on user role
func generatePracticeData(userType string) *PracticeData {
	rand.Seed(time.Now().UnixNano())

	data := &PracticeData{
		IsActive:  true,
		CreatedAt: time.Now(),
	}

	// Generate sample buses
	busModels := []string{"Blue Bird Vision", "Thomas C2", "IC CE Series", "Blue Bird All American"}
	for i := 1; i <= 5; i++ {
		bus := Bus{
			BusID:          fmt.Sprintf("PRACTICE-%d", i),
			Model:          sql.NullString{String: busModels[rand.Intn(len(busModels))], Valid: true},
			Capacity:       sql.NullInt32{Int32: int32(rand.Intn(20) + 50), Valid: true},
			Status:         "active",
			CurrentMileage: sql.NullInt32{Int32: int32(rand.Intn(50000) + 10000), Valid: true},
			UpdatedAt:      sql.NullTime{Time: time.Now(), Valid: true},
			CreatedAt:      sql.NullTime{Time: time.Now(), Valid: true},
		}
		data.Buses = append(data.Buses, bus)
	}

	// Generate sample routes
	routeNames := []string{"North Elementary", "South Middle School", "East High School", "West Academy", "Central District"}
	for i, name := range routeNames {
		route := Route{
			RouteID:   fmt.Sprintf("R%03d-PRACTICE", i+1),
			RouteName: name + " Route",
		}
		data.Routes = append(data.Routes, route)
	}

	// Generate sample students
	firstNames := []string{"Emma", "Liam", "Olivia", "Noah", "Ava", "Ethan", "Sophia", "Mason", "Isabella", "William"}
	lastNames := []string{"Smith", "Johnson", "Williams", "Brown", "Jones", "Garcia", "Miller", "Davis", "Rodriguez", "Martinez"}
	
	for i := 0; i < 20; i++ {
		student := Student{
			StudentID:      fmt.Sprintf("S%04d-PRACTICE", i+1),
			Name:           fmt.Sprintf("%s %s", firstNames[rand.Intn(len(firstNames))], lastNames[rand.Intn(len(lastNames))]),
			Guardian:       fmt.Sprintf("Parent of Student %d", i+1),
			PhoneNumber:    fmt.Sprintf("555-%04d", rand.Intn(10000)),
			Locations:      fmt.Sprintf("%d Practice Lane", 100+i),
			PickupTime:     fmt.Sprintf("07:%02d AM", 15+rand.Intn(30)),
			DropoffTime:    fmt.Sprintf("03:%02d PM", 15+rand.Intn(30)),
			Driver:         "PRACTICE-DRIVER",
			Active:         true,
			PositionNumber: i + 1,
			CreatedAt:      time.Now(),
		}
		data.Students = append(data.Students, student)
	}

	// Generate sample maintenance records
	maintenanceTypes := []string{"Oil Change", "Tire Rotation", "Brake Inspection", "Engine Check", "Safety Inspection"}
	for i := 0; i < 10; i++ {
		maintenance := VehicleMaintenanceLog{
			VehicleID: data.Buses[rand.Intn(len(data.Buses))].BusID,
			Date:      time.Now().AddDate(0, -rand.Intn(6), -rand.Intn(30)).Format("2006-01-02"),
			Category:  maintenanceTypes[rand.Intn(len(maintenanceTypes))],
			Notes:     maintenanceTypes[rand.Intn(len(maintenanceTypes))],
			Mileage:   rand.Intn(50000) + 10000,
			Cost:      float64(100 + rand.Intn(900)), // $100-$1000
			CreatedAt: time.Now(),
		}
		data.Maintenance = append(data.Maintenance, maintenance)
	}

	// Generate sample driver logs
	for i := 0; i < 10; i++ {
		log := DriverLog{
			Date:         time.Now().AddDate(0, 0, -i).Format("2006-01-02"),
			Driver:       "PRACTICE-DRIVER",
			BusID:        data.Buses[rand.Intn(len(data.Buses))].BusID,
			RouteID:      data.Routes[rand.Intn(len(data.Routes))].RouteID,
			Period:       []string{"morning", "afternoon"}[rand.Intn(2)],
			Departure:    fmt.Sprintf("%02d:%02d", 7+rand.Intn(2), rand.Intn(60)),
			Arrival:      fmt.Sprintf("%02d:%02d", 8+rand.Intn(2), rand.Intn(60)),
			BeginMileage: float64(50000 + i*100),
			EndMileage:   float64(50000 + i*100 + 20 + rand.Intn(30)),
			CreatedAt:    time.Now(),
		}
		data.Logs = append(data.Logs, log)
	}

	// Create route assignments if manager
	if userType == "manager" {
		for i, route := range data.Routes {
			if i < len(data.Buses) {
				assignment := RouteAssignment{
					RouteID:  route.RouteID,
					BusID:    data.Buses[i].BusID,
					Driver:   fmt.Sprintf("PRACTICE-DRIVER-%d", i+1),
					RouteName: route.RouteName,
					AssignedDate: time.Now().Format("2006-01-02"),
					CreatedAt: time.Now(),
				}
				data.Assignments = append(data.Assignments, assignment)
			}
		}
	}

	return data
}

// practiceDataMiddleware checks if practice mode is active and provides practice data
func practiceDataMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check for practice mode cookie or query parameter
		cookie, err := r.Cookie("practice_mode")
		isPracticeMode := (err == nil && cookie.Value == "active") || r.URL.Query().Get("practice") == "1"

		if isPracticeMode {
			// Add practice mode indicator to request context
			r = r.WithContext(addPracticeModeToContext(r.Context(), true))
			
			// Add warning banner
			r.Header.Set("X-Practice-Mode", "active")
		}

		next(w, r)
	}
}

// Helper function to check if in practice mode
func isInPracticeMode(r *http.Request) bool {
	cookie, err := r.Cookie("practice_mode")
	return (err == nil && cookie.Value == "active") || r.URL.Query().Get("practice") == "1"
}

// API endpoint for practice mode data
func practiceModeDataHandler(w http.ResponseWriter, r *http.Request) {
	session := getUserFromSession(r)
	if session == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	sessionID := getSessionID(r)
	practiceData, exists := practiceDataStore[sessionID]

	if !exists || !practiceData.IsActive {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"active": false,
			"message": "Practice mode is not active",
		})
		return
	}

	// Return practice data based on request
	dataType := r.URL.Query().Get("type")
	
	var response interface{}
	switch dataType {
	case "buses":
		response = practiceData.Buses
	case "routes":
		response = practiceData.Routes
	case "students":
		response = practiceData.Students
	case "logs":
		response = practiceData.Logs
	case "maintenance":
		response = practiceData.Maintenance
	case "summary":
		response = map[string]interface{}{
			"active":      true,
			"created_at":  practiceData.CreatedAt,
			"bus_count":   len(practiceData.Buses),
			"route_count": len(practiceData.Routes),
			"student_count": len(practiceData.Students),
			"log_count":   len(practiceData.Logs),
		}
	default:
		response = practiceData
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Clean up old practice data periodically
func cleanupPracticeData() {
	ticker := time.NewTicker(1 * time.Hour)
	go func() {
		for range ticker.C {
			now := time.Now()
			for sessionID, data := range practiceDataStore {
				// Remove practice data older than 2 hours
				if now.Sub(data.CreatedAt) > 2*time.Hour {
					delete(practiceDataStore, sessionID)
					log.Printf("Cleaned up practice data for session: %s", sessionID)
				}
			}
		}
	}()
}

// getSessionID is already defined in middleware_metrics.go

// Context key for practice mode
// contextKey is already defined in middleware.go
type practiceModeContextKey string

const practiceModeKey practiceModeContextKey = "practice_mode"

// Add practice mode to context
func addPracticeModeToContext(ctx context.Context, isPractice bool) context.Context {
	return context.WithValue(ctx, practiceModeKey, isPractice)
}

// Get practice mode from context
func getPracticeModeFromContext(ctx context.Context) bool {
	if val, ok := ctx.Value(practiceModeKey).(bool); ok {
		return val
	}
	return false
}