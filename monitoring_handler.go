package main

import (
	"encoding/json"
	"net/http"
	"runtime"
	"time"
)

type SystemMetrics struct {
	Timestamp    time.Time              `json:"timestamp"`
	Database     DatabaseMetrics        `json:"database"`
	Memory       MemoryMetrics          `json:"memory"`
	Performance  PerformanceMetrics     `json:"performance"`
	ActiveUsers  int                    `json:"active_users"`
	RequestRate  float64                `json:"request_rate"`
	ErrorRate    float64                `json:"error_rate"`
	DataCounts   map[string]int         `json:"data_counts"`
}

type DatabaseMetrics struct {
	Connected       bool   `json:"connected"`
	ActiveQueries   int    `json:"active_queries"`
	ConnectionPool  int    `json:"connection_pool"`
	ResponseTime    int64  `json:"response_time_ms"`
	Status          string `json:"status"`
}

type MemoryMetrics struct {
	Allocated      uint64 `json:"allocated"`
	TotalAllocated uint64 `json:"total_allocated"`
	SystemMemory   uint64 `json:"system_memory"`
	NumGC          uint32 `json:"num_gc"`
	Goroutines     int    `json:"goroutines"`
}

type PerformanceMetrics struct {
	AverageResponseTime float64            `json:"avg_response_time_ms"`
	RequestsPerSecond   float64            `json:"requests_per_second"`
	SlowQueries         int                `json:"slow_queries"`
	CacheHitRate        float64            `json:"cache_hit_rate"`
	EndpointMetrics     map[string]float64 `json:"endpoint_metrics"`
}

// monitoringDashboardHandler serves the monitoring dashboard page
func monitoringDashboardHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	data := map[string]interface{}{
		"User":       user,
		"CSRFToken":  getSessionCSRFToken(r),
		"Title":      "System Monitoring",
		"PageTitle":  "Real-time System Monitoring",
	}

	renderTemplate(w, r, "monitoring_dashboard.html", data)
}

// monitoringAPIHandler provides real-time system metrics
func monitoringAPIHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		SendError(w, ErrUnauthorized("Access denied"))
		return
	}

	metrics := collectSystemMetrics()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

// monitoringWebSocketHandler provides real-time updates via WebSocket
func monitoringWebSocketHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// For now, return regular metrics updates
	// WebSocket implementation would go here
	metrics := collectSystemMetrics()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

func collectSystemMetrics() SystemMetrics {
	metrics := SystemMetrics{
		Timestamp: time.Now(),
	}

	// Database metrics
	metrics.Database = collectDatabaseMetrics()
	
	// Memory metrics
	metrics.Memory = collectMemoryMetrics()
	
	// Performance metrics
	metrics.Performance = collectPerformanceMetrics()
	
	// Active users
	metrics.ActiveUsers = getActiveUserCount()
	
	// Request rate
	metrics.RequestRate = calculateRequestRate()
	
	// Error rate
	metrics.ErrorRate = monitoringCalculateErrorRate()
	
	// Data counts
	metrics.DataCounts = collectDataCounts()
	
	return metrics
}

func collectDatabaseMetrics() DatabaseMetrics {
	start := time.Now()
	metrics := DatabaseMetrics{
		Status: "healthy",
	}
	
	if db == nil {
		metrics.Connected = false
		metrics.Status = "disconnected"
		return metrics
	}
	
	// Test database connection
	err := db.Ping()
	metrics.Connected = err == nil
	metrics.ResponseTime = time.Since(start).Milliseconds()
	
	if !metrics.Connected {
		metrics.Status = "error"
		return metrics
	}
	
	// Get connection pool stats
	stats := db.Stats()
	metrics.ConnectionPool = stats.OpenConnections
	metrics.ActiveQueries = stats.InUse
	
	// Check if response time is slow
	if metrics.ResponseTime > 100 {
		metrics.Status = "slow"
	}
	
	return metrics
}

func collectMemoryMetrics() MemoryMetrics {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	return MemoryMetrics{
		Allocated:      m.Alloc,
		TotalAllocated: m.TotalAlloc,
		SystemMemory:   m.Sys,
		NumGC:          m.NumGC,
		Goroutines:     runtime.NumGoroutine(),
	}
}

func collectPerformanceMetrics() PerformanceMetrics {
	metrics := PerformanceMetrics{
		EndpointMetrics: make(map[string]float64),
	}
	
	// Get cache statistics if available
	if queryCache != nil {
		stats := queryCache.Stats()
		if requests, ok := stats["requests"].(int); ok && requests > 0 {
			if hits, ok := stats["hits"].(int); ok {
				metrics.CacheHitRate = float64(hits) / float64(requests)
			}
		}
	}
	
	// Get real metrics from metrics collector
	collectorMetrics := metricsCollector.GetMetrics()
	
	// Collect endpoint-specific metrics and count slow queries
	slowCount := 0
	if endpointStats, ok := collectorMetrics["endpointStats"].(map[string]interface{}); ok {
		for endpoint, stats := range endpointStats {
			if statsMap, ok := stats.(map[string]interface{}); ok {
				if avgDuration, ok := statsMap["avgDuration"].(float64); ok {
					metrics.EndpointMetrics[endpoint] = avgDuration
				}
				// Count slow queries (queries over 100ms)
				if maxDuration, ok := statsMap["maxDuration"].(float64); ok && maxDuration > 100 {
					slowCount++
				}
			}
		}
	}
	metrics.SlowQueries = slowCount
	
	// Calculate average response time from real data
	metrics.AverageResponseTime = metricsCollector.GetAverageResponseTime()
	
	// Get actual request rate
	metrics.RequestsPerSecond = metricsCollector.GetRequestRate() / 60.0 // Convert from per minute to per second
	
	return metrics
}

func getActiveUserCount() int {
	// Get active user count from metrics collector
	return metricsCollector.GetActiveUserCount()
}

func calculateRequestRate() float64 {
	// Get request rate from metrics collector (returns requests per minute)
	return metricsCollector.GetRequestRate()
}

func monitoringCalculateErrorRate() float64 {
	// Get error rate from metrics collector (returns percentage)
	return metricsCollector.GetErrorRate()
}

func collectDataCounts() map[string]int {
	counts := make(map[string]int)
	
	if db == nil {
		return counts
	}
	
	// Collect record counts
	// Using a map to store table names and their corresponding count queries
	// This approach avoids SQL injection by using predefined queries
	tableQueries := map[string]string{
		"buses":               "SELECT COUNT(*) FROM buses",
		"vehicles":            "SELECT COUNT(*) FROM vehicles",
		"students":            "SELECT COUNT(*) FROM students",
		"routes":              "SELECT COUNT(*) FROM routes",
		"maintenance_records": "SELECT COUNT(*) FROM maintenance_records",
		"service_records":     "SELECT COUNT(*) FROM service_records",
		"fuel_records":        "SELECT COUNT(*) FROM fuel_records",
		"ecse_students":       "SELECT COUNT(*) FROM ecse_students",
		"users":               "SELECT COUNT(*) FROM users",
	}
	
	for table, query := range tableQueries {
		var count int
		err := db.Get(&count, query)
		if err == nil {
			counts[table] = count
		}
	}
	
	return counts
}

// alertsHandler returns system alerts
func alertsHandler(w http.ResponseWriter, r *http.Request) {
	// Delegate to the new alerts API handler which has persistent storage
	getAlertsHandler(w, r)
}