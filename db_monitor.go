package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// DBStats represents database connection pool statistics
type DBStats struct {
	OpenConnections      int     `json:"open_connections"`
	InUse                int     `json:"in_use"`
	Idle                 int     `json:"idle"`
	WaitCount            int64   `json:"wait_count"`
	WaitDuration         string  `json:"wait_duration"`
	MaxIdleClosed        int64   `json:"max_idle_closed"`
	MaxIdleTimeClosed    int64   `json:"max_idle_time_closed"`
	MaxLifetimeClosed    int64   `json:"max_lifetime_closed"`
	ConnectionUtilization float64 `json:"connection_utilization"`
	HealthStatus         string  `json:"health_status"`
	Timestamp            string  `json:"timestamp"`
}

// getDBStats returns current database connection pool statistics
func getDBStats() DBStats {
	if db == nil {
		return DBStats{
			HealthStatus: "error",
			Timestamp:    time.Now().Format(time.RFC3339),
		}
	}
	
	stats := db.Stats()
	
	// Calculate connection utilization percentage
	maxConns := db.Stats().MaxOpenConnections
	utilization := float64(0)
	if maxConns > 0 {
		utilization = (float64(stats.InUse) / float64(maxConns)) * 100
	}
	
	// Determine health status based on utilization and wait count
	healthStatus := "healthy"
	if utilization > 80 {
		healthStatus = "warning"
	}
	if utilization > 95 || stats.WaitCount > 100 {
		healthStatus = "critical"
	}
	
	return DBStats{
		OpenConnections:      stats.OpenConnections,
		InUse:                stats.InUse,
		Idle:                 stats.Idle,
		WaitCount:            stats.WaitCount,
		WaitDuration:         stats.WaitDuration.String(),
		MaxIdleClosed:        stats.MaxIdleClosed,
		MaxIdleTimeClosed:    stats.MaxIdleTimeClosed,
		MaxLifetimeClosed:    stats.MaxLifetimeClosed,
		ConnectionUtilization: utilization,
		HealthStatus:         healthStatus,
		Timestamp:            time.Now().Format(time.RFC3339),
	}
}

// dbStatsHandler returns database connection pool statistics as JSON
func dbStatsHandler(w http.ResponseWriter, r *http.Request) {
	// Check if user is authenticated and is a manager
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	
	stats := getDBStats()
	
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(stats); err != nil {
		log.Printf("Error encoding DB stats: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// dbHealthCheckHandler provides a simple health check endpoint
func dbHealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	// This endpoint doesn't require authentication for monitoring tools
	stats := getDBStats()
	
	response := map[string]interface{}{
		"status":       stats.HealthStatus,
		"timestamp":    stats.Timestamp,
		"utilization":  fmt.Sprintf("%.2f%%", stats.ConnectionUtilization),
		"wait_count":   stats.WaitCount,
		"connections":  fmt.Sprintf("%d/%d", stats.InUse, stats.OpenConnections),
	}
	
	// Set appropriate HTTP status code based on health
	statusCode := http.StatusOK
	if stats.HealthStatus == "warning" {
		statusCode = http.StatusOK // Still OK but with warning
	} else if stats.HealthStatus == "critical" {
		statusCode = http.StatusServiceUnavailable
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

// startDBMonitoring starts a background goroutine to monitor database health
func startDBMonitoring() {
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		
		for range ticker.C {
			stats := getDBStats()
			
			// Log warnings if connection pool is under stress
			if stats.HealthStatus == "warning" {
				log.Printf("WARNING: Database connection pool utilization high: %.2f%% (%d/%d connections in use)",
					stats.ConnectionUtilization, stats.InUse, stats.OpenConnections)
			} else if stats.HealthStatus == "critical" {
				log.Printf("CRITICAL: Database connection pool critical: %.2f%% utilization, %d requests waiting",
					stats.ConnectionUtilization, stats.WaitCount)
			}
			
			// Log if connections are being closed frequently
			if stats.MaxIdleClosed > 10 || stats.MaxLifetimeClosed > 10 {
				log.Printf("INFO: Connection pool cleanup - MaxIdleClosed: %d, MaxLifetimeClosed: %d",
					stats.MaxIdleClosed, stats.MaxLifetimeClosed)
			}
		}
	}()
	
	log.Println("Database connection pool monitoring started")
}

// getDetailedDBMetrics returns more detailed database metrics for debugging
func getDetailedDBMetrics() map[string]interface{} {
	if db == nil {
		return map[string]interface{}{
			"error": "database not initialized",
		}
	}
	
	stats := db.Stats()
	
	// Get database size (PostgreSQL specific)
	var dbSize string
	err := db.QueryRow(`
		SELECT pg_size_pretty(pg_database_size(current_database()))
	`).Scan(&dbSize)
	if err != nil {
		dbSize = "unknown"
	}
	
	// Get active queries count
	var activeQueries int
	err = db.QueryRow(`
		SELECT COUNT(*) 
		FROM pg_stat_activity 
		WHERE datname = current_database() 
		AND state = 'active'
		AND pid != pg_backend_pid()
	`).Scan(&activeQueries)
	if err != nil {
		activeQueries = -1
	}
	
	// Get longest running query
	var longestQuery sql.NullString
	var longestDuration sql.NullString
	err = db.QueryRow(`
		SELECT 
			query,
			age(clock_timestamp(), query_start)::text
		FROM pg_stat_activity
		WHERE datname = current_database()
		AND state = 'active'
		AND pid != pg_backend_pid()
		ORDER BY query_start ASC
		LIMIT 1
	`).Scan(&longestQuery, &longestDuration)
	
	return map[string]interface{}{
		"connection_pool": map[string]interface{}{
			"open_connections":       stats.OpenConnections,
			"in_use":                 stats.InUse,
			"idle":                   stats.Idle,
			"wait_count":             stats.WaitCount,
			"wait_duration":          stats.WaitDuration.String(),
			"max_open_connections":   stats.MaxOpenConnections,
			"max_idle_closed":        stats.MaxIdleClosed,
			"max_idle_time_closed":   stats.MaxIdleTimeClosed,
			"max_lifetime_closed":    stats.MaxLifetimeClosed,
		},
		"database": map[string]interface{}{
			"size":                   dbSize,
			"active_queries":         activeQueries,
			"longest_running_query":  longestQuery.String,
			"longest_query_duration": longestDuration.String,
		},
		"health": getDBStats(),
	}
}

// dbMetricsHandler provides detailed database metrics
func dbMetricsHandler(w http.ResponseWriter, r *http.Request) {
	// Check if user is authenticated and is a manager
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	
	metrics := getDetailedDBMetrics()
	
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(metrics); err != nil {
		log.Printf("Error encoding DB metrics: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}