package main

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
	
	"github.com/jmoiron/sqlx"
)

// MetricsStorage handles persistent storage of monitoring metrics
type MetricsStorage struct {
	mu sync.RWMutex
	db *sqlx.DB
}

// MetricRecord represents a single metric data point
type MetricRecord struct {
	ID         int64                  `db:"id"`
	MetricType string                 `db:"metric_type"`
	Timestamp  time.Time              `db:"timestamp"`
	Value      float64                `db:"value"`
	Metadata   string                 `db:"metadata"`
	Tags       map[string]interface{} `json:"-"`
}

// AlertRecord represents a system alert
type AlertRecord struct {
	ID            int64     `db:"id"`
	Level         string    `db:"level"`         // critical, warning, info
	Component     string    `db:"component"`     // database, memory, application, runtime
	Message       string    `db:"message"`
	Timestamp     time.Time `db:"timestamp"`
	Acknowledged  bool      `db:"acknowledged"`
	AcknowledgedBy *string  `db:"acknowledged_by"`
	AcknowledgedAt *time.Time `db:"acknowledged_at"`
	ResolvedAt    *time.Time `db:"resolved_at"`
	Metadata      string    `db:"metadata"`
}

// Global metrics storage instance
var metricsStorage *MetricsStorage

// InitializeMetricsStorage sets up the metrics storage system
func InitializeMetricsStorage(db *sqlx.DB) error {
	metricsStorage = &MetricsStorage{
		db: db,
	}
	
	// Create tables if they don't exist
	if err := metricsStorage.createTables(); err != nil {
		return fmt.Errorf("failed to create metrics tables: %w", err)
	}
	
	// Start the metrics aggregation routine
	go metricsStorage.startAggregationRoutine()
	
	// Start the cleanup routine for old metrics
	go metricsStorage.startCleanupRoutine()
	
	return nil
}

// createTables creates the necessary database tables for metrics storage
func (ms *MetricsStorage) createTables() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS metrics (
			id SERIAL PRIMARY KEY,
			metric_type VARCHAR(100) NOT NULL,
			timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			value NUMERIC NOT NULL,
			metadata JSONB,
			CONSTRAINT metrics_timestamp_idx_unique UNIQUE (metric_type, timestamp)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_metrics_timestamp ON metrics(timestamp)`,
		`CREATE INDEX IF NOT EXISTS idx_metrics_type_timestamp ON metrics(metric_type, timestamp)`,
		
		`CREATE TABLE IF NOT EXISTS alerts (
			id SERIAL PRIMARY KEY,
			level VARCHAR(20) NOT NULL,
			component VARCHAR(50) NOT NULL,
			message TEXT NOT NULL,
			timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			acknowledged BOOLEAN DEFAULT FALSE,
			acknowledged_by VARCHAR(100),
			acknowledged_at TIMESTAMP,
			resolved_at TIMESTAMP,
			metadata JSONB
		)`,
		`CREATE INDEX IF NOT EXISTS idx_alerts_timestamp ON alerts(timestamp)`,
		`CREATE INDEX IF NOT EXISTS idx_alerts_acknowledged ON alerts(acknowledged)`,
		`CREATE INDEX IF NOT EXISTS idx_alerts_level ON alerts(level)`,
		
		// Aggregated metrics tables for better performance
		`CREATE TABLE IF NOT EXISTS metrics_hourly (
			id SERIAL PRIMARY KEY,
			metric_type VARCHAR(100) NOT NULL,
			hour TIMESTAMP NOT NULL,
			avg_value NUMERIC,
			min_value NUMERIC,
			max_value NUMERIC,
			sum_value NUMERIC,
			count INTEGER,
			CONSTRAINT metrics_hourly_unique UNIQUE (metric_type, hour)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_metrics_hourly_hour ON metrics_hourly(hour)`,
		
		`CREATE TABLE IF NOT EXISTS metrics_daily (
			id SERIAL PRIMARY KEY,
			metric_type VARCHAR(100) NOT NULL,
			day DATE NOT NULL,
			avg_value NUMERIC,
			min_value NUMERIC,
			max_value NUMERIC,
			sum_value NUMERIC,
			count INTEGER,
			CONSTRAINT metrics_daily_unique UNIQUE (metric_type, day)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_metrics_daily_day ON metrics_daily(day)`,
	}
	
	for _, query := range queries {
		if _, err := ms.db.Exec(query); err != nil {
			return err
		}
	}
	
	return nil
}

// StoreMetric stores a single metric data point
func (ms *MetricsStorage) StoreMetric(metricType string, value float64, tags map[string]interface{}) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	
	metadata := ""
	if tags != nil {
		data, err := json.Marshal(tags)
		if err != nil {
			return err
		}
		metadata = string(data)
	}
	
	query := `INSERT INTO metrics (metric_type, timestamp, value, metadata) 
	          VALUES ($1, $2, $3, $4)
	          ON CONFLICT (metric_type, timestamp) DO UPDATE 
	          SET value = EXCLUDED.value, metadata = EXCLUDED.metadata`
	
	_, err := ms.db.Exec(query, metricType, time.Now(), value, metadata)
	return err
}

// StoreBatchMetrics stores multiple metrics in a single transaction
func (ms *MetricsStorage) StoreBatchMetrics(metrics []MetricRecord) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	
	tx, err := ms.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	
	stmt, err := tx.Prepare(`INSERT INTO metrics (metric_type, timestamp, value, metadata) 
	                         VALUES ($1, $2, $3, $4)
	                         ON CONFLICT (metric_type, timestamp) DO UPDATE 
	                         SET value = EXCLUDED.value, metadata = EXCLUDED.metadata`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	
	for _, metric := range metrics {
		metadata := ""
		if metric.Tags != nil {
			data, err := json.Marshal(metric.Tags)
			if err != nil {
				continue
			}
			metadata = string(data)
		}
		
		if _, err := stmt.Exec(metric.MetricType, metric.Timestamp, metric.Value, metadata); err != nil {
			return err
		}
	}
	
	return tx.Commit()
}

// GetMetrics retrieves metrics for a specific type and time range
func (ms *MetricsStorage) GetMetrics(metricType string, start, end time.Time) ([]MetricRecord, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	
	var metrics []MetricRecord
	query := `SELECT id, metric_type, timestamp, value, metadata 
	          FROM metrics 
	          WHERE metric_type = $1 AND timestamp BETWEEN $2 AND $3 
	          ORDER BY timestamp`
	
	err := ms.db.Select(&metrics, query, metricType, start, end)
	if err != nil {
		return nil, err
	}
	
	// Parse metadata for each metric
	for i := range metrics {
		if metrics[i].Metadata != "" {
			var tags map[string]interface{}
			if err := json.Unmarshal([]byte(metrics[i].Metadata), &tags); err == nil {
				metrics[i].Tags = tags
			}
		}
	}
	
	return metrics, nil
}

// GetAggregatedMetrics retrieves aggregated metrics for reporting
func (ms *MetricsStorage) GetAggregatedMetrics(metricType string, start, end time.Time, interval string) ([]map[string]interface{}, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	
	var results []map[string]interface{}
	var query string
	
	switch interval {
	case "hourly":
		query = `SELECT hour as time, avg_value, min_value, max_value, sum_value, count 
		         FROM metrics_hourly 
		         WHERE metric_type = $1 AND hour BETWEEN $2 AND $3 
		         ORDER BY hour`
	case "daily":
		query = `SELECT day as time, avg_value, min_value, max_value, sum_value, count 
		         FROM metrics_daily 
		         WHERE metric_type = $1 AND day BETWEEN $2::date AND $3::date 
		         ORDER BY day`
	default:
		// Raw metrics with time bucketing
		query = `SELECT 
		           date_trunc('minute', timestamp) as time,
		           AVG(value) as avg_value,
		           MIN(value) as min_value,
		           MAX(value) as max_value,
		           SUM(value) as sum_value,
		           COUNT(*) as count
		         FROM metrics 
		         WHERE metric_type = $1 AND timestamp BETWEEN $2 AND $3 
		         GROUP BY date_trunc('minute', timestamp)
		         ORDER BY time`
	}
	
	rows, err := ms.db.Query(query, metricType, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	for rows.Next() {
		var result map[string]interface{} = make(map[string]interface{})
		var time time.Time
		var avgValue, minValue, maxValue, sumValue float64
		var count int
		
		if err := rows.Scan(&time, &avgValue, &minValue, &maxValue, &sumValue, &count); err != nil {
			continue
		}
		
		result["time"] = time
		result["avg_value"] = avgValue
		result["min_value"] = minValue
		result["max_value"] = maxValue
		result["sum_value"] = sumValue
		result["count"] = count
		
		results = append(results, result)
	}
	
	return results, nil
}

// StoreAlert stores a new system alert
func (ms *MetricsStorage) StoreAlert(level, component, message string, metadata map[string]interface{}) (*AlertRecord, error) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	
	metadataJSON := ""
	if metadata != nil {
		data, err := json.Marshal(metadata)
		if err != nil {
			return nil, err
		}
		metadataJSON = string(data)
	}
	
	var alert AlertRecord
	query := `INSERT INTO alerts (level, component, message, metadata) 
	          VALUES ($1, $2, $3, $4) 
	          RETURNING id, level, component, message, timestamp, acknowledged, metadata`
	
	err := ms.db.Get(&alert, query, level, component, message, metadataJSON)
	return &alert, err
}

// GetActiveAlerts retrieves all unacknowledged alerts
func (ms *MetricsStorage) GetActiveAlerts() ([]AlertRecord, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	
	var alerts []AlertRecord
	query := `SELECT * FROM alerts 
	          WHERE acknowledged = FALSE AND resolved_at IS NULL 
	          ORDER BY timestamp DESC`
	
	err := ms.db.Select(&alerts, query)
	return alerts, err
}

// AcknowledgeAlert marks an alert as acknowledged
func (ms *MetricsStorage) AcknowledgeAlert(alertID int64, acknowledgedBy string) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	
	query := `UPDATE alerts 
	          SET acknowledged = TRUE, acknowledged_by = $1, acknowledged_at = $2 
	          WHERE id = $3`
	
	_, err := ms.db.Exec(query, acknowledgedBy, time.Now(), alertID)
	return err
}

// ResolveAlert marks an alert as resolved
func (ms *MetricsStorage) ResolveAlert(alertID int64) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	
	query := `UPDATE alerts SET resolved_at = $1 WHERE id = $2`
	_, err := ms.db.Exec(query, time.Now(), alertID)
	return err
}

// startAggregationRoutine aggregates metrics periodically
func (ms *MetricsStorage) startAggregationRoutine() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	
	for range ticker.C {
		// Aggregate hourly metrics
		ms.aggregateHourlyMetrics()
		
		// Aggregate daily metrics
		ms.aggregateDailyMetrics()
	}
}

// aggregateHourlyMetrics aggregates raw metrics into hourly buckets
func (ms *MetricsStorage) aggregateHourlyMetrics() {
	query := `INSERT INTO metrics_hourly (metric_type, hour, avg_value, min_value, max_value, sum_value, count)
	          SELECT 
	            metric_type,
	            date_trunc('hour', timestamp) as hour,
	            AVG(value),
	            MIN(value),
	            MAX(value),
	            SUM(value),
	            COUNT(*)
	          FROM metrics
	          WHERE timestamp >= NOW() - INTERVAL '2 hours'
	          GROUP BY metric_type, date_trunc('hour', timestamp)
	          ON CONFLICT (metric_type, hour) DO UPDATE SET
	            avg_value = EXCLUDED.avg_value,
	            min_value = EXCLUDED.min_value,
	            max_value = EXCLUDED.max_value,
	            sum_value = EXCLUDED.sum_value,
	            count = EXCLUDED.count`
	
	ms.db.Exec(query)
}

// aggregateDailyMetrics aggregates hourly metrics into daily buckets
func (ms *MetricsStorage) aggregateDailyMetrics() {
	query := `INSERT INTO metrics_daily (metric_type, day, avg_value, min_value, max_value, sum_value, count)
	          SELECT 
	            metric_type,
	            date_trunc('day', hour)::date as day,
	            AVG(avg_value),
	            MIN(min_value),
	            MAX(max_value),
	            SUM(sum_value),
	            SUM(count)
	          FROM metrics_hourly
	          WHERE hour >= NOW() - INTERVAL '2 days'
	          GROUP BY metric_type, date_trunc('day', hour)::date
	          ON CONFLICT (metric_type, day) DO UPDATE SET
	            avg_value = EXCLUDED.avg_value,
	            min_value = EXCLUDED.min_value,
	            max_value = EXCLUDED.max_value,
	            sum_value = EXCLUDED.sum_value,
	            count = EXCLUDED.count`
	
	ms.db.Exec(query)
}

// startCleanupRoutine removes old metrics based on retention policy
func (ms *MetricsStorage) startCleanupRoutine() {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()
	
	for range ticker.C {
		// Keep raw metrics for 7 days
		ms.db.Exec("DELETE FROM metrics WHERE timestamp < NOW() - INTERVAL '7 days'")
		
		// Keep hourly metrics for 30 days
		ms.db.Exec("DELETE FROM metrics_hourly WHERE hour < NOW() - INTERVAL '30 days'")
		
		// Keep daily metrics for 1 year
		ms.db.Exec("DELETE FROM metrics_daily WHERE day < NOW() - INTERVAL '365 days'")
		
		// Keep resolved alerts for 30 days
		ms.db.Exec("DELETE FROM alerts WHERE resolved_at IS NOT NULL AND resolved_at < NOW() - INTERVAL '30 days'")
	}
}

// CollectAndStoreCurrentMetrics collects current system metrics and stores them
func CollectAndStoreCurrentMetrics() error {
	if metricsStorage == nil {
		return fmt.Errorf("metrics storage not initialized")
	}
	
	// Collect current metrics
	metrics := metricsCollector.GetMetrics()
	
	// Store request count
	if requestCount, ok := metrics["requestCount"].(uint64); ok {
		metricsStorage.StoreMetric("request_count", float64(requestCount), nil)
	}
	
	// Store error count
	if errorCount, ok := metrics["errorCount"].(uint64); ok {
		metricsStorage.StoreMetric("error_count", float64(errorCount), nil)
	}
	
	// Store active requests
	if activeRequests, ok := metrics["activeRequests"].(int32); ok {
		metricsStorage.StoreMetric("active_requests", float64(activeRequests), nil)
	}
	
	// Store active sessions
	if activeSessions, ok := metrics["activeSessions"].(int); ok {
		metricsStorage.StoreMetric("active_sessions", float64(activeSessions), nil)
	}
	
	// Collect and store system metrics
	systemMetrics := collectSystemMetrics()
	
	// Store database metrics
	metricsStorage.StoreMetric("db_response_time", float64(systemMetrics.Database.ResponseTime), nil)
	metricsStorage.StoreMetric("db_connection_pool", float64(systemMetrics.Database.ConnectionPool), nil)
	metricsStorage.StoreMetric("db_active_queries", float64(systemMetrics.Database.ActiveQueries), nil)
	
	// Store memory metrics
	metricsStorage.StoreMetric("memory_allocated", float64(systemMetrics.Memory.Allocated), nil)
	metricsStorage.StoreMetric("memory_total_allocated", float64(systemMetrics.Memory.TotalAllocated), nil)
	metricsStorage.StoreMetric("memory_system", float64(systemMetrics.Memory.SystemMemory), nil)
	metricsStorage.StoreMetric("goroutines", float64(systemMetrics.Memory.Goroutines), nil)
	
	// Store performance metrics
	metricsStorage.StoreMetric("avg_response_time", systemMetrics.Performance.AverageResponseTime, nil)
	metricsStorage.StoreMetric("requests_per_second", systemMetrics.Performance.RequestsPerSecond, nil)
	metricsStorage.StoreMetric("cache_hit_rate", systemMetrics.Performance.CacheHitRate, nil)
	
	return nil
}

// StartMetricsCollection starts a routine to periodically collect and store metrics
func StartMetricsCollection() {
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		
		for range ticker.C {
			if err := CollectAndStoreCurrentMetrics(); err != nil {
				LogError("Failed to collect and store metrics", err)
			}
		}
	}()
}