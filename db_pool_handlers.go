package main

import (
	"encoding/json"
	"net/http"
	"runtime"
	"time"
)

// dbPoolMetricsHandler returns current database pool metrics
func dbPoolMetricsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	metrics := GetPoolMetrics()
	stats := db.Stats()

	response := map[string]interface{}{
		"timestamp": time.Now().UTC(),
		"pool": map[string]interface{}{
			"open_connections":    stats.OpenConnections,
			"in_use":             stats.InUse,
			"idle":               stats.Idle,
			"wait_count":         stats.WaitCount,
			"wait_duration_ms":   stats.WaitDuration.Milliseconds(),
			"max_idle_closed":    stats.MaxIdleClosed,
			"max_lifetime_closed": stats.MaxLifetimeClosed,
		},
		"performance": map[string]interface{}{
			"total_queries":    metrics.QueryCount,
			"total_errors":     metrics.ErrorCount,
			"error_rate":       dbPoolCalculateErrorRate(metrics.QueryCount, metrics.ErrorCount),
			"last_health_check": metrics.LastHealthCheck,
			"health_status":     metrics.HealthStatus,
		},
		"configuration": map[string]interface{}{
			"max_open_conns":        poolConfig.MaxOpenConns,
			"max_idle_conns":        poolConfig.MaxIdleConns,
			"conn_max_lifetime":     poolConfig.ConnMaxLifetime.String(),
			"conn_max_idle_time":    poolConfig.ConnMaxIdleTime.String(),
			"health_check_interval": poolConfig.HealthCheckInterval.String(),
		},
		"system": map[string]interface{}{
			"cpu_cores":     runtime.NumCPU(),
			"goroutines":    runtime.NumGoroutine(),
			"optimal_conns": getOptimalConnectionNumbers(),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// dbPoolHealthHandler returns database pool health status
func dbPoolHealthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	health := MonitorPoolHealth(db)
	
	// Set appropriate HTTP status based on health
	status := http.StatusOK
	if healthStatus, ok := health["status"].(string); ok {
		switch healthStatus {
		case "unhealthy":
			status = http.StatusServiceUnavailable
		case "degraded", "warning":
			status = http.StatusOK // Still operational but with warnings
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(health)
}

// dbPoolOptimizeHandler triggers pool optimization
func dbPoolOptimizeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get current metrics
	stats := db.Stats()
	currentLoad := stats.InUse

	// Optimize pool based on current load
	OptimizePoolForLoad(db, currentLoad)

	// Get optimal settings
	optimalMax, optimalIdle := GetOptimalPoolSize()

	response := map[string]interface{}{
		"timestamp": time.Now().UTC(),
		"action":    "pool_optimization",
		"before": map[string]interface{}{
			"max_open_conns": poolConfig.MaxOpenConns,
			"max_idle_conns": poolConfig.MaxIdleConns,
			"in_use":        stats.InUse,
			"idle":          stats.Idle,
		},
		"recommendations": map[string]interface{}{
			"optimal_max_open":  optimalMax,
			"optimal_max_idle":  optimalIdle,
			"current_load":      currentLoad,
			"utilization_rate":  calculateUtilizationRate(stats.InUse, poolConfig.MaxOpenConns),
		},
		"after": map[string]interface{}{
			"max_open_conns": poolConfig.MaxOpenConns,
			"max_idle_conns": poolConfig.MaxIdleConns,
		},
	}

	LogInfo("Database pool optimization triggered")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Helper functions

func dbPoolCalculateErrorRate(totalQueries, totalErrors int64) float64 {
	if totalQueries == 0 {
		return 0
	}
	return float64(totalErrors) / float64(totalQueries) * 100
}

func calculateUtilizationRate(inUse, maxOpen int) float64 {
	if maxOpen == 0 {
		return 0
	}
	return float64(inUse) / float64(maxOpen) * 100
}

func getOptimalConnectionNumbers() map[string]interface{} {
	maxOpen, maxIdle := GetOptimalPoolSize()
	return map[string]interface{}{
		"recommended_max_open": maxOpen,
		"recommended_max_idle": maxIdle,
		"based_on_cpu_cores":  runtime.NumCPU(),
	}
}

// dbStatsHandler returns general database statistics
func dbStatsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats := db.Stats()
	
	response := map[string]interface{}{
		"timestamp": time.Now().UTC(),
		"stats": map[string]interface{}{
			"open_connections": stats.OpenConnections,
			"in_use":          stats.InUse,
			"idle":            stats.Idle,
			"wait_count":      stats.WaitCount,
			"wait_duration":   stats.WaitDuration.String(),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// dbMetricsHandler returns database performance metrics
func dbMetricsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get table sizes
	var tableSizes []map[string]interface{}
	query := `
		SELECT 
			schemaname,
			tablename,
			pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS size,
			pg_total_relation_size(schemaname||'.'||tablename) AS size_bytes
		FROM pg_tables 
		WHERE schemaname NOT IN ('pg_catalog', 'information_schema')
		ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC
		LIMIT 20
	`
	
	rows, err := db.Query(query)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var schema, table, size string
			var sizeBytes int64
			if err := rows.Scan(&schema, &table, &size, &sizeBytes); err == nil {
				tableSizes = append(tableSizes, map[string]interface{}{
					"schema":     schema,
					"table":      table,
					"size":       size,
					"size_bytes": sizeBytes,
				})
			}
		}
	}

	// Get index usage stats
	var indexStats []map[string]interface{}
	indexQuery := `
		SELECT 
			schemaname,
			tablename,
			indexname,
			idx_scan,
			idx_tup_read,
			idx_tup_fetch
		FROM pg_stat_user_indexes
		ORDER BY idx_scan DESC
		LIMIT 20
	`
	
	rows, err = db.Query(indexQuery)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var schema, table, index string
			var scans, reads, fetches int64
			if err := rows.Scan(&schema, &table, &index, &scans, &reads, &fetches); err == nil {
				indexStats = append(indexStats, map[string]interface{}{
					"schema":    schema,
					"table":     table,
					"index":     index,
					"scans":     scans,
					"reads":     reads,
					"fetches":   fetches,
				})
			}
		}
	}

	response := map[string]interface{}{
		"timestamp":    time.Now().UTC(),
		"table_sizes":  tableSizes,
		"index_stats":  indexStats,
		"pool_metrics": GetPoolMetrics(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// dbHealthCheckHandler provides a simple health check endpoint
func dbHealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Perform a simple ping with timeout
	if err := db.Ping(); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "unhealthy",
			"error":  err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "healthy",
		"timestamp": time.Now().UTC(),
	})
}

// dbPoolMonitorHandler renders the database pool monitoring page
func dbPoolMonitorHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	data := map[string]interface{}{
		"Title":     "Database Pool Monitor",
		"User":      user,
		"CSRFToken": getSessionCSRFToken(r),
	}

	renderTemplate(w, r, "db_pool_monitor.html", data)
}