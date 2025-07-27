package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// alertsAPIHandler handles API requests for alerts management
func alertsAPIHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		SendError(w, ErrUnauthorized("Access denied"))
		return
	}
	
	switch r.Method {
	case "GET":
		getAlertsHandler(w, r)
	case "POST":
		acknowledgeAlertHandler(w, r)
	case "PUT":
		resolveAlertHandler(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// getAlertsHandler retrieves active and recent alerts
func getAlertsHandler(w http.ResponseWriter, r *http.Request) {
	if metricsStorage == nil {
		SendError(w, ErrInternal("Metrics storage not initialized", fmt.Errorf("metricsStorage is nil")))
		return
	}
	
	// Get filter parameters
	filterType := r.URL.Query().Get("type")
	includeResolved := r.URL.Query().Get("include_resolved") == "true"
	
	var alerts []AlertRecord
	var err error
	
	if filterType == "active" || !includeResolved {
		// Get only active (unacknowledged and unresolved) alerts
		alerts, err = metricsStorage.GetActiveAlerts()
	} else {
		// Get all alerts from the last 24 hours
		query := `SELECT * FROM alerts 
		          WHERE timestamp > NOW() - INTERVAL '24 hours' 
		          ORDER BY timestamp DESC`
		err = metricsStorage.db.Select(&alerts, query)
	}
	
	if err != nil {
		SendError(w, ErrDatabase("query execution", err))
		return
	}
	
	// Also check for real-time system issues and generate alerts
	generateSystemAlerts()
	
	// Convert to response format
	response := make([]map[string]interface{}, len(alerts))
	for i, alert := range alerts {
		alertMap := map[string]interface{}{
			"id":           alert.ID,
			"level":        alert.Level,
			"component":    alert.Component,
			"message":      alert.Message,
			"timestamp":    alert.Timestamp,
			"acknowledged": alert.Acknowledged,
		}
		
		if alert.AcknowledgedBy != nil {
			alertMap["acknowledged_by"] = *alert.AcknowledgedBy
		}
		if alert.AcknowledgedAt != nil {
			alertMap["acknowledged_at"] = *alert.AcknowledgedAt
		}
		if alert.ResolvedAt != nil {
			alertMap["resolved_at"] = *alert.ResolvedAt
		}
		
		// Parse metadata if present
		if alert.Metadata != "" {
			var metadata map[string]interface{}
			if err := json.Unmarshal([]byte(alert.Metadata), &metadata); err == nil {
				alertMap["metadata"] = metadata
			}
		}
		
		response[i] = alertMap
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// acknowledgeAlertHandler marks an alert as acknowledged
func acknowledgeAlertHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil {
		SendError(w, ErrUnauthorized("Not authenticated"))
		return
	}
	
	var req struct {
		AlertID int64 `json:"alert_id"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, ErrBadRequest("Invalid request body"))
		return
	}
	
	if metricsStorage == nil {
		SendError(w, ErrInternal("Metrics storage not initialized", fmt.Errorf("metricsStorage is nil")))
		return
	}
	
	err := metricsStorage.AcknowledgeAlert(req.AlertID, user.Username)
	if err != nil {
		SendError(w, ErrDatabase("query execution", err))
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"message": "Alert acknowledged",
		"alert_id": req.AlertID,
		"acknowledged_by": user.Username,
		"acknowledged_at": time.Now(),
	})
}

// resolveAlertHandler marks an alert as resolved
func resolveAlertHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil {
		SendError(w, ErrUnauthorized("Not authenticated"))
		return
	}
	
	var req struct {
		AlertID int64 `json:"alert_id"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, ErrBadRequest("Invalid request body"))
		return
	}
	
	if metricsStorage == nil {
		SendError(w, ErrInternal("Metrics storage not initialized", fmt.Errorf("metricsStorage is nil")))
		return
	}
	
	err := metricsStorage.ResolveAlert(req.AlertID)
	if err != nil {
		SendError(w, ErrDatabase("query execution", err))
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"message": "Alert resolved",
		"alert_id": req.AlertID,
		"resolved_at": time.Now(),
	})
}

// generateSystemAlerts checks current system state and generates alerts if needed
func generateSystemAlerts() {
	if metricsStorage == nil {
		return
	}
	
	metrics := collectSystemMetrics()
	
	// Check database connection
	if !metrics.Database.Connected {
		metricsStorage.StoreAlert("critical", "database", "Database connection lost", map[string]interface{}{
			"status": metrics.Database.Status,
		})
	} else if metrics.Database.ResponseTime > 500 {
		metricsStorage.StoreAlert("warning", "database", 
			"Slow database response time", map[string]interface{}{
			"response_time_ms": metrics.Database.ResponseTime,
			"threshold_ms": 500,
		})
	}
	
	// Check memory usage
	memoryUsageMB := metrics.Memory.Allocated / 1024 / 1024
	if memoryUsageMB > 1000 { // Over 1GB
		metricsStorage.StoreAlert("critical", "memory", 
			"Critical memory usage", map[string]interface{}{
			"allocated_mb": memoryUsageMB,
			"threshold_mb": 1000,
			"goroutines": metrics.Memory.Goroutines,
		})
	} else if memoryUsageMB > 500 { // Over 500MB
		metricsStorage.StoreAlert("warning", "memory", 
			"High memory usage", map[string]interface{}{
			"allocated_mb": memoryUsageMB,
			"threshold_mb": 500,
			"goroutines": metrics.Memory.Goroutines,
		})
	}
	
	// Check error rate
	if metrics.ErrorRate > 0.1 { // Over 10%
		metricsStorage.StoreAlert("critical", "application", 
			"Critical error rate", map[string]interface{}{
			"error_rate_percent": metrics.ErrorRate * 100,
			"threshold_percent": 10,
		})
	} else if metrics.ErrorRate > 0.05 { // Over 5%
		metricsStorage.StoreAlert("warning", "application", 
			"High error rate", map[string]interface{}{
			"error_rate_percent": metrics.ErrorRate * 100,
			"threshold_percent": 5,
		})
	}
	
	// Check goroutine count
	if metrics.Memory.Goroutines > 5000 {
		metricsStorage.StoreAlert("critical", "runtime", 
			"Excessive goroutine count", map[string]interface{}{
			"goroutine_count": metrics.Memory.Goroutines,
			"threshold": 5000,
		})
	} else if metrics.Memory.Goroutines > 1000 {
		metricsStorage.StoreAlert("warning", "runtime", 
			"High goroutine count", map[string]interface{}{
			"goroutine_count": metrics.Memory.Goroutines,
			"threshold": 1000,
		})
	}
	
	// Check database connection pool
	if metrics.Database.ConnectionPool > 80 {
		metricsStorage.StoreAlert("warning", "database", 
			"High database connection pool usage", map[string]interface{}{
			"connections": metrics.Database.ConnectionPool,
			"threshold": 80,
		})
	}
}

// metricsHistoryHandler returns historical metrics data
func metricsHistoryHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		SendError(w, ErrUnauthorized("Access denied"))
		return
	}
	
	if metricsStorage == nil {
		SendError(w, ErrInternal("Metrics storage not initialized", fmt.Errorf("metricsStorage is nil")))
		return
	}
	
	// Get query parameters
	metricType := r.URL.Query().Get("metric")
	startStr := r.URL.Query().Get("start")
	endStr := r.URL.Query().Get("end")
	interval := r.URL.Query().Get("interval") // raw, hourly, daily
	
	if metricType == "" {
		SendError(w, ErrBadRequest("Metric type required"))
		return
	}
	
	// Parse time range
	var start, end time.Time
	var err error
	
	if startStr == "" {
		start = time.Now().Add(-24 * time.Hour) // Default to last 24 hours
	} else {
		start, err = time.Parse(time.RFC3339, startStr)
		if err != nil {
			SendError(w, ErrBadRequest("Invalid start time format"))
			return
		}
	}
	
	if endStr == "" {
		end = time.Now()
	} else {
		end, err = time.Parse(time.RFC3339, endStr)
		if err != nil {
			SendError(w, ErrBadRequest("Invalid end time format"))
			return
		}
	}
	
	// Get aggregated metrics
	data, err := metricsStorage.GetAggregatedMetrics(metricType, start, end, interval)
	if err != nil {
		SendError(w, ErrDatabase("query execution", err))
		return
	}
	
	response := map[string]interface{}{
		"metric_type": metricType,
		"start": start,
		"end": end,
		"interval": interval,
		"data": data,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// alertsSummaryHandler returns a summary of alerts by category
func alertsSummaryHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		SendError(w, ErrUnauthorized("Access denied"))
		return
	}
	
	if metricsStorage == nil {
		SendError(w, ErrInternal("Metrics storage not initialized", fmt.Errorf("metricsStorage is nil")))
		return
	}
	
	// Get alert counts by level and component
	query := `SELECT 
	            level, 
	            component, 
	            COUNT(*) as count,
	            COUNT(CASE WHEN acknowledged = FALSE THEN 1 END) as unacknowledged_count
	          FROM alerts 
	          WHERE timestamp > NOW() - INTERVAL '24 hours'
	          GROUP BY level, component`
	
	var results []struct {
		Level               string `db:"level"`
		Component          string `db:"component"`
		Count              int    `db:"count"`
		UnacknowledgedCount int    `db:"unacknowledged_count"`
	}
	
	err := metricsStorage.db.Select(&results, query)
	if err != nil {
		SendError(w, ErrDatabase("query execution", err))
		return
	}
	
	// Organize by level
	summary := map[string]map[string]interface{}{
		"critical": {
			"total": 0,
			"unacknowledged": 0,
			"components": make(map[string]int),
		},
		"warning": {
			"total": 0,
			"unacknowledged": 0,
			"components": make(map[string]int),
		},
		"info": {
			"total": 0,
			"unacknowledged": 0,
			"components": make(map[string]int),
		},
	}
	
	for _, result := range results {
		if levelData, ok := summary[result.Level]; ok {
			levelData["total"] = levelData["total"].(int) + result.Count
			levelData["unacknowledged"] = levelData["unacknowledged"].(int) + result.UnacknowledgedCount
			levelData["components"].(map[string]int)[result.Component] = result.Count
		}
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}