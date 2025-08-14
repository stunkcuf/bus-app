package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"time"

	"github.com/jmoiron/sqlx"
)

// DriverScorecard represents a comprehensive driver performance scorecard
type DriverScorecard struct {
	Driver          string          `json:"driver"`
	Period          string          `json:"period"`
	OverallScore    float64         `json:"overall_score"`
	OverallGrade    string          `json:"overall_grade"`
	Categories      []ScoreCategory `json:"categories"`
	Achievements    []Achievement   `json:"achievements"`
	Recommendations []string        `json:"recommendations"`
	Ranking         int             `json:"ranking"`
	TotalDrivers    int             `json:"total_drivers"`
	Trend           string          `json:"trend"`
	GeneratedAt     time.Time       `json:"generated_at"`
}

// ScoreCategory represents a scoring category
type ScoreCategory struct {
	Name     string  `json:"name"`
	Score    float64 `json:"score"`
	MaxScore float64 `json:"max_score"`
	Weight   float64 `json:"weight"`
	Grade    string  `json:"grade"`
	Details  string  `json:"details"`
}

// Achievement represents a driver achievement
type Achievement struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
	Date        string `json:"date"`
}

// DriverStats holds statistical data for scoring
type DriverStats struct {
	TotalTrips         int
	OnTimeTrips        int
	TotalMileage       int
	AccidentsReported  int
	StudentComplaints  int
	AttendanceAccuracy float64
	FuelEfficiency     float64
	MaintenanceReports int
	SafetyViolations   int
	RouteCompletions   int
	StudentCount       int
}

// ScorecardService handles driver scorecard generation
type ScorecardService struct {
	db *sqlx.DB
}

// NewScorecardService creates a new scorecard service
func NewScorecardService() *ScorecardService {
	return &ScorecardService{db: db}
}

// GenerateDriverScorecard generates a scorecard for a specific driver
func (ss *ScorecardService) GenerateDriverScorecard(driverUsername string, startDate, endDate time.Time) (*DriverScorecard, error) {
	scorecard := &DriverScorecard{
		Driver:      driverUsername,
		Period:      fmt.Sprintf("%s to %s", startDate.Format("Jan 2, 2006"), endDate.Format("Jan 2, 2006")),
		GeneratedAt: time.Now(),
	}

	// Gather driver statistics
	stats, err := ss.gatherDriverStats(driverUsername, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to gather driver stats: %w", err)
	}

	// Calculate scores for each category
	categories := []ScoreCategory{
		ss.calculateSafetyScore(stats),
		ss.calculatePunctualityScore(stats),
		ss.calculateAttendanceScore(stats),
		ss.calculateEfficiencyScore(stats),
		ss.calculateReliabilityScore(stats),
	}

	scorecard.Categories = categories

	// Calculate overall score
	totalScore := 0.0
	totalWeight := 0.0
	for _, cat := range categories {
		totalScore += cat.Score * cat.Weight
		totalWeight += cat.Weight
	}

	if totalWeight > 0 {
		scorecard.OverallScore = totalScore / totalWeight
	}

	scorecard.OverallGrade = ss.getGrade(scorecard.OverallScore)

	// Generate achievements
	scorecard.Achievements = ss.generateAchievements(driverUsername, stats, categories)

	// Generate recommendations
	scorecard.Recommendations = ss.generateRecommendations(categories)

	// Calculate ranking
	ranking, total, err := ss.calculateDriverRanking(driverUsername, startDate, endDate)
	if err == nil {
		scorecard.Ranking = ranking
		scorecard.TotalDrivers = total
	}

	// Determine trend
	scorecard.Trend = ss.calculateTrend(driverUsername, startDate, endDate)

	return scorecard, nil
}

// gatherDriverStats collects all statistics for a driver
func (ss *ScorecardService) gatherDriverStats(driver string, startDate, endDate time.Time) (*DriverStats, error) {
	stats := &DriverStats{}

	// Get trip statistics
	err := ss.db.QueryRow(`
		SELECT COUNT(*) as total_trips,
		       COUNT(CASE WHEN departure_time IS NOT NULL AND arrival_time IS NOT NULL THEN 1 END) as on_time_trips,
		       COALESCE(SUM(end_mileage - start_mileage), 0) as total_mileage
		FROM driver_logs
		WHERE driver = $1 AND date BETWEEN $2 AND $3
	`, driver, startDate, endDate).Scan(&stats.TotalTrips, &stats.OnTimeTrips, &stats.TotalMileage)
	if err != nil {
		log.Printf("Error getting trip stats: %v", err)
	}

	// Get student count
	err = ss.db.QueryRow(`
		SELECT COUNT(DISTINCT s.id)
		FROM students s
		WHERE s.driver = $1 AND s.active = true
	`, driver).Scan(&stats.StudentCount)
	if err != nil {
		log.Printf("Error getting student count: %v", err)
	}

	// Get attendance accuracy (simplified - counts students marked present)
	var totalAttendance, accurateAttendance int
	err = ss.db.QueryRow(`
		SELECT COUNT(*) as total,
		       COUNT(CASE WHEN attendance IS NOT NULL THEN 1 END) as recorded
		FROM driver_logs
		WHERE driver = $1 AND date BETWEEN $2 AND $3
	`, driver, startDate, endDate).Scan(&totalAttendance, &accurateAttendance)
	if err == nil && totalAttendance > 0 {
		stats.AttendanceAccuracy = float64(accurateAttendance) / float64(totalAttendance) * 100
	}

	// Get fuel efficiency if fuel records exist
	var avgMPG sql.NullFloat64
	err = ss.db.QueryRow(`
		SELECT AVG(mpg) as avg_mpg
		FROM (
			SELECT (f2.odometer - f1.odometer)::float / f2.gallons as mpg
			FROM fuel_records f1
			JOIN fuel_records f2 ON f1.vehicle_id = f2.vehicle_id 
			    AND f2.date > f1.date
			    AND f2.odometer > f1.odometer
			WHERE f2.driver = $1 AND f2.date BETWEEN $2 AND $3
			ORDER BY f2.date
		) as mpg_calc
	`, driver, startDate, endDate).Scan(&avgMPG)
	if err == nil && avgMPG.Valid {
		stats.FuelEfficiency = avgMPG.Float64
	} else {
		stats.FuelEfficiency = 7.5 // Default MPG for school buses
	}

	// Get maintenance reports submitted
	err = ss.db.QueryRow(`
		SELECT COUNT(*)
		FROM (
			SELECT 1 FROM bus_maintenance_logs WHERE performed_by = $1 AND date BETWEEN $2 AND $3
			UNION ALL
			SELECT 1 FROM vehicle_maintenance_logs WHERE performed_by = $1 AND date BETWEEN $2 AND $3
		) as reports
	`, driver, startDate, endDate).Scan(&stats.MaintenanceReports)
	if err != nil {
		log.Printf("Error getting maintenance reports: %v", err)
	}

	// For demo purposes, set some values to 0 (would come from incident tracking system)
	stats.AccidentsReported = 0
	stats.StudentComplaints = 0
	stats.SafetyViolations = 0

	// Calculate route completions
	stats.RouteCompletions = stats.OnTimeTrips

	return stats, nil
}

// Score calculation methods

func (ss *ScorecardService) calculateSafetyScore(stats *DriverStats) ScoreCategory {
	score := 100.0

	// Deduct for accidents (major penalty)
	score -= float64(stats.AccidentsReported) * 25

	// Deduct for safety violations
	score -= float64(stats.SafetyViolations) * 10

	// Deduct for complaints
	score -= float64(stats.StudentComplaints) * 5

	// Ensure score doesn't go below 0
	if score < 0 {
		score = 0
	}

	details := fmt.Sprintf("Accidents: %d, Violations: %d, Complaints: %d",
		stats.AccidentsReported, stats.SafetyViolations, stats.StudentComplaints)

	return ScoreCategory{
		Name:     "Safety",
		Score:    score,
		MaxScore: 100,
		Weight:   0.35, // 35% weight
		Grade:    ss.getGrade(score),
		Details:  details,
	}
}

func (ss *ScorecardService) calculatePunctualityScore(stats *DriverStats) ScoreCategory {
	score := 0.0

	if stats.TotalTrips > 0 {
		score = float64(stats.OnTimeTrips) / float64(stats.TotalTrips) * 100
	}

	details := fmt.Sprintf("On-time trips: %d/%d (%.1f%%)",
		stats.OnTimeTrips, stats.TotalTrips, score)

	return ScoreCategory{
		Name:     "Punctuality",
		Score:    score,
		MaxScore: 100,
		Weight:   0.25, // 25% weight
		Grade:    ss.getGrade(score),
		Details:  details,
	}
}

func (ss *ScorecardService) calculateAttendanceScore(stats *DriverStats) ScoreCategory {
	score := stats.AttendanceAccuracy

	details := fmt.Sprintf("Attendance recording accuracy: %.1f%%", score)

	return ScoreCategory{
		Name:     "Attendance Management",
		Score:    score,
		MaxScore: 100,
		Weight:   0.15, // 15% weight
		Grade:    ss.getGrade(score),
		Details:  details,
	}
}

func (ss *ScorecardService) calculateEfficiencyScore(stats *DriverStats) ScoreCategory {
	score := 75.0 // Base score

	// Bonus for good fuel efficiency
	if stats.FuelEfficiency > 8.0 {
		score += 15
	} else if stats.FuelEfficiency > 7.0 {
		score += 10
	}

	// Bonus for maintenance reports
	if stats.MaintenanceReports > 0 {
		score += math.Min(float64(stats.MaintenanceReports)*2, 10) // Max 10 points
	}

	if score > 100 {
		score = 100
	}

	details := fmt.Sprintf("Fuel efficiency: %.1f MPG, Maintenance reports: %d",
		stats.FuelEfficiency, stats.MaintenanceReports)

	return ScoreCategory{
		Name:     "Efficiency",
		Score:    score,
		MaxScore: 100,
		Weight:   0.15, // 15% weight
		Grade:    ss.getGrade(score),
		Details:  details,
	}
}

func (ss *ScorecardService) calculateReliabilityScore(stats *DriverStats) ScoreCategory {
	score := 100.0

	// Calculate expected trips (rough estimate)
	expectedTrips := 20 // Assume 20 trips per month minimum
	if stats.TotalTrips < expectedTrips {
		score = float64(stats.TotalTrips) / float64(expectedTrips) * 100
	}

	details := fmt.Sprintf("Trips completed: %d, Students managed: %d",
		stats.TotalTrips, stats.StudentCount)

	return ScoreCategory{
		Name:     "Reliability",
		Score:    score,
		MaxScore: 100,
		Weight:   0.10, // 10% weight
		Grade:    ss.getGrade(score),
		Details:  details,
	}
}

// getGrade converts a numeric score to letter grade
func (ss *ScorecardService) getGrade(score float64) string {
	switch {
	case score >= 95:
		return "A+"
	case score >= 90:
		return "A"
	case score >= 85:
		return "B+"
	case score >= 80:
		return "B"
	case score >= 75:
		return "C+"
	case score >= 70:
		return "C"
	case score >= 65:
		return "D+"
	case score >= 60:
		return "D"
	default:
		return "F"
	}
}

// generateAchievements creates achievement badges for the driver
func (ss *ScorecardService) generateAchievements(driver string, stats *DriverStats, categories []ScoreCategory) []Achievement {
	achievements := []Achievement{}

	// Perfect safety
	for _, cat := range categories {
		if cat.Name == "Safety" && cat.Score == 100 {
			achievements = append(achievements, Achievement{
				Title:       "Safety Star",
				Description: "Perfect safety record",
				Icon:        "🛡️",
				Date:        time.Now().Format("Jan 2006"),
			})
		}
	}

	// High mileage
	if stats.TotalMileage > 5000 {
		achievements = append(achievements, Achievement{
			Title:       "Road Warrior",
			Description: fmt.Sprintf("%d miles driven", stats.TotalMileage),
			Icon:        "🚌",
			Date:        time.Now().Format("Jan 2006"),
		})
	}

	// Perfect attendance recording
	if stats.AttendanceAccuracy >= 98 {
		achievements = append(achievements, Achievement{
			Title:       "Attendance Excellence",
			Description: "98%+ attendance accuracy",
			Icon:        "📋",
			Date:        time.Now().Format("Jan 2006"),
		})
	}

	// Fuel efficiency
	if stats.FuelEfficiency > 8.5 {
		achievements = append(achievements, Achievement{
			Title:       "Eco Driver",
			Description: fmt.Sprintf("%.1f MPG efficiency", stats.FuelEfficiency),
			Icon:        "🌱",
			Date:        time.Now().Format("Jan 2006"),
		})
	}

	return achievements
}

// generateRecommendations creates improvement recommendations
func (ss *ScorecardService) generateRecommendations(categories []ScoreCategory) []string {
	recommendations := []string{}

	for _, cat := range categories {
		if cat.Score < 70 {
			switch cat.Name {
			case "Safety":
				recommendations = append(recommendations, "Focus on defensive driving techniques and safety protocols")
			case "Punctuality":
				recommendations = append(recommendations, "Review route timing and allow buffer time for unexpected delays")
			case "Attendance Management":
				recommendations = append(recommendations, "Ensure consistent attendance recording for all students")
			case "Efficiency":
				recommendations = append(recommendations, "Monitor fuel consumption and practice eco-driving techniques")
			case "Reliability":
				recommendations = append(recommendations, "Maintain consistent schedule and communicate any issues promptly")
			}
		}
	}

	if len(recommendations) == 0 {
		recommendations = append(recommendations, "Keep up the excellent work! Continue maintaining high standards")
	}

	return recommendations
}

// calculateDriverRanking determines driver's rank among peers
func (ss *ScorecardService) calculateDriverRanking(driver string, startDate, endDate time.Time) (int, int, error) {
	// Get all active drivers
	var drivers []string
	err := ss.db.Select(&drivers, "SELECT username FROM users WHERE role = 'driver' AND status = 'active'")
	if err != nil {
		return 0, 0, err
	}

	type driverScore struct {
		driver string
		score  float64
	}

	scores := []driverScore{}

	// Calculate scores for all drivers
	for _, d := range drivers {
		stats, err := ss.gatherDriverStats(d, startDate, endDate)
		if err != nil {
			continue
		}

		// Simple scoring based on key metrics
		score := 0.0
		if stats.TotalTrips > 0 {
			score = float64(stats.OnTimeTrips) / float64(stats.TotalTrips) * 100
		}

		scores = append(scores, driverScore{driver: d, score: score})
	}

	// Sort by score
	for i := 0; i < len(scores); i++ {
		for j := i + 1; j < len(scores); j++ {
			if scores[j].score > scores[i].score {
				scores[i], scores[j] = scores[j], scores[i]
			}
		}
	}

	// Find driver's rank
	rank := 1
	for i, s := range scores {
		if s.driver == driver {
			rank = i + 1
			break
		}
	}

	return rank, len(scores), nil
}

// calculateTrend determines performance trend
func (ss *ScorecardService) calculateTrend(driver string, startDate, endDate time.Time) string {
	// Compare with previous period
	duration := endDate.Sub(startDate)
	prevEndDate := startDate.AddDate(0, 0, -1)
	prevStartDate := prevEndDate.Add(-duration)

	currentStats, err := ss.gatherDriverStats(driver, startDate, endDate)
	if err != nil {
		return "stable"
	}

	prevStats, err := ss.gatherDriverStats(driver, prevStartDate, prevEndDate)
	if err != nil {
		return "stable"
	}

	// Compare key metrics
	currentScore := 0.0
	prevScore := 0.0

	if currentStats.TotalTrips > 0 {
		currentScore = float64(currentStats.OnTimeTrips) / float64(currentStats.TotalTrips)
	}

	if prevStats.TotalTrips > 0 {
		prevScore = float64(prevStats.OnTimeTrips) / float64(prevStats.TotalTrips)
	}

	if currentScore > prevScore*1.05 {
		return "improving"
	} else if currentScore < prevScore*0.95 {
		return "declining"
	}

	return "stable"
}

// API Handlers

// driverScorecardHandler generates a scorecard for a specific driver
func driverScorecardHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get driver parameter
	driver := r.URL.Query().Get("driver")
	if driver == "" {
		// Default to current user if driver
		if user.Role == "driver" {
			driver = user.Username
		} else {
			http.Error(w, "Driver parameter required", http.StatusBadRequest)
			return
		}
	}

	// Check permissions
	if user.Role == "driver" && driver != user.Username {
		http.Error(w, "Cannot view other driver scorecards", http.StatusForbidden)
		return
	}

	// Get date range (default to current month)
	now := time.Now()
	startDate := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)
	endDate := now

	if sd := r.URL.Query().Get("start_date"); sd != "" {
		if parsed, err := time.Parse("2006-01-02", sd); err == nil {
			startDate = parsed
		}
	}

	if ed := r.URL.Query().Get("end_date"); ed != "" {
		if parsed, err := time.Parse("2006-01-02", ed); err == nil {
			endDate = parsed
		}
	}

	service := NewScorecardService()
	scorecard, err := service.GenerateDriverScorecard(driver, startDate, endDate)
	if err != nil {
		log.Printf("Failed to generate scorecard: %v", err)
		http.Error(w, "Failed to generate scorecard", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(scorecard)
}

// allDriverScorecardsHandler returns scorecards for all drivers (manager only)
func allDriverScorecardsHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get date range
	now := time.Now()
	startDate := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)
	endDate := now

	if sd := r.URL.Query().Get("start_date"); sd != "" {
		if parsed, err := time.Parse("2006-01-02", sd); err == nil {
			startDate = parsed
		}
	}

	if ed := r.URL.Query().Get("end_date"); ed != "" {
		if parsed, err := time.Parse("2006-01-02", ed); err == nil {
			endDate = parsed
		}
	}

	// Get all active drivers
	var drivers []string
	err := db.Select(&drivers, "SELECT username FROM users WHERE role = 'driver' AND status = 'active' ORDER BY username")
	if err != nil {
		log.Printf("Failed to get drivers: %v", err)
		http.Error(w, "Failed to get drivers", http.StatusInternalServerError)
		return
	}

	service := NewScorecardService()
	scorecards := []DriverScorecard{}

	for _, driver := range drivers {
		scorecard, err := service.GenerateDriverScorecard(driver, startDate, endDate)
		if err != nil {
			log.Printf("Failed to generate scorecard for %s: %v", driver, err)
			continue
		}
		scorecards = append(scorecards, *scorecard)
	}

	// Sort by overall score
	for i := 0; i < len(scorecards); i++ {
		for j := i + 1; j < len(scorecards); j++ {
			if scorecards[j].OverallScore > scorecards[i].OverallScore {
				scorecards[i], scorecards[j] = scorecards[j], scorecards[i]
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(scorecards)
}
