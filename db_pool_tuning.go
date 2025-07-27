package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
)

// DBPoolConfig holds configuration for database connection pooling
type DBPoolConfig struct {
	MaxOpenConns        int
	MaxIdleConns        int
	ConnMaxLifetime     time.Duration
	ConnMaxIdleTime     time.Duration
	HealthCheckInterval time.Duration
}

// DBPoolMetrics tracks database pool performance metrics
type DBPoolMetrics struct {
	mu                sync.RWMutex
	OpenConnections   int
	InUse             int
	Idle              int
	WaitCount         int64
	WaitDuration      time.Duration
	MaxIdleClosed     int64
	MaxLifetimeClosed int64
	QueryCount        int64
	ErrorCount        int64
	LastHealthCheck   time.Time
	HealthStatus      bool
}

var (
	poolMetrics = &DBPoolMetrics{}
	poolConfig  *DBPoolConfig
)

// DefaultPoolConfig returns optimized default pool configuration
func DefaultPoolConfig() *DBPoolConfig {
	// Base configuration on available CPU cores
	numCPU := runtime.NumCPU()
	
	// Calculate optimal connection numbers
	// Formula: max_connections = (num_cores * 2) + effective_spindle_count
	// For SSDs, effective_spindle_count = 1
	maxConns := (numCPU * 2) + 1
	
	// Ensure minimum viable connections
	if maxConns < 5 {
		maxConns = 5
	}
	
	// Cap at a reasonable maximum to prevent overwhelming the database
	if maxConns > 25 {
		maxConns = 25
	}
	
	return &DBPoolConfig{
		MaxOpenConns:        maxConns,
		MaxIdleConns:        maxConns / 2, // Keep half as idle
		ConnMaxLifetime:     time.Hour,     // Refresh connections hourly
		ConnMaxIdleTime:     10 * time.Minute,
		HealthCheckInterval: 30 * time.Second,
	}
}

// LoadPoolConfigFromEnv loads pool configuration from environment variables
func LoadPoolConfigFromEnv() *DBPoolConfig {
	config := DefaultPoolConfig()
	
	// Override with environment variables if set
	if val := os.Getenv("DB_MAX_OPEN_CONNS"); val != "" {
		if n, err := strconv.Atoi(val); err == nil && n > 0 {
			config.MaxOpenConns = n
		}
	}
	
	if val := os.Getenv("DB_MAX_IDLE_CONNS"); val != "" {
		if n, err := strconv.Atoi(val); err == nil && n >= 0 {
			config.MaxIdleConns = n
		}
	}
	
	if val := os.Getenv("DB_CONN_MAX_LIFETIME"); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			config.ConnMaxLifetime = d
		}
	}
	
	if val := os.Getenv("DB_CONN_MAX_IDLE_TIME"); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			config.ConnMaxIdleTime = d
		}
	}
	
	if val := os.Getenv("DB_HEALTH_CHECK_INTERVAL"); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			config.HealthCheckInterval = d
		}
	}
	
	return config
}

// ConfigureDBPool applies optimal connection pool settings to the database
func ConfigureDBPool(db *sqlx.DB, config *DBPoolConfig) error {
	if db == nil {
		return fmt.Errorf("database connection is nil")
	}
	
	if config == nil {
		config = DefaultPoolConfig()
	}
	
	poolConfig = config
	
	// Apply connection pool settings
	db.SetMaxOpenConns(config.MaxOpenConns)
	db.SetMaxIdleConns(config.MaxIdleConns)
	db.SetConnMaxLifetime(config.ConnMaxLifetime)
	db.SetConnMaxIdleTime(config.ConnMaxIdleTime)
	
	log.Printf("Database pool configured: MaxOpen=%d, MaxIdle=%d, MaxLifetime=%v, MaxIdleTime=%v",
		config.MaxOpenConns, config.MaxIdleConns, config.ConnMaxLifetime, config.ConnMaxIdleTime)
	
	// Start health check routine
	go startHealthCheckRoutine(db, config.HealthCheckInterval)
	
	// Start metrics collection
	go startMetricsCollection(db)
	
	return nil
}

// startHealthCheckRoutine performs periodic health checks on the database
func startHealthCheckRoutine(db *sqlx.DB, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	
	for range ticker.C {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err := db.PingContext(ctx)
		cancel()
		
		poolMetrics.mu.Lock()
		poolMetrics.LastHealthCheck = time.Now()
		poolMetrics.HealthStatus = err == nil
		poolMetrics.mu.Unlock()
		
		if err != nil {
			log.Printf("Database health check failed: %v", err)
			// Attempt to recover connections
			recoverDBConnections(db)
		}
	}
}

// startMetricsCollection collects database pool metrics
func startMetricsCollection(db *sqlx.DB) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	
	for range ticker.C {
		stats := db.Stats()
		
		poolMetrics.mu.Lock()
		poolMetrics.OpenConnections = stats.OpenConnections
		poolMetrics.InUse = stats.InUse
		poolMetrics.Idle = stats.Idle
		poolMetrics.WaitCount = stats.WaitCount
		poolMetrics.WaitDuration = stats.WaitDuration
		poolMetrics.MaxIdleClosed = stats.MaxIdleClosed
		poolMetrics.MaxLifetimeClosed = stats.MaxLifetimeClosed
		poolMetrics.mu.Unlock()
	}
}

// recoverDBConnections attempts to recover database connections
func recoverDBConnections(db *sqlx.DB) {
	log.Println("Attempting to recover database connections...")
	
	// Close idle connections to force reconnection
	db.SetMaxIdleConns(0)
	time.Sleep(100 * time.Millisecond)
	
	// Restore idle connection limit
	if poolConfig != nil {
		db.SetMaxIdleConns(poolConfig.MaxIdleConns)
	}
	
	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := db.PingContext(ctx); err == nil {
		log.Println("Database connection recovered successfully")
	} else {
		log.Printf("Failed to recover database connection: %v", err)
	}
}

// GetPoolMetrics returns current database pool metrics
func GetPoolMetrics() DBPoolMetrics {
	poolMetrics.mu.RLock()
	defer poolMetrics.mu.RUnlock()
	
	return DBPoolMetrics{
		OpenConnections:   poolMetrics.OpenConnections,
		InUse:             poolMetrics.InUse,
		Idle:              poolMetrics.Idle,
		WaitCount:         poolMetrics.WaitCount,
		WaitDuration:      poolMetrics.WaitDuration,
		MaxIdleClosed:     poolMetrics.MaxIdleClosed,
		MaxLifetimeClosed: poolMetrics.MaxLifetimeClosed,
		QueryCount:        poolMetrics.QueryCount,
		ErrorCount:        poolMetrics.ErrorCount,
		LastHealthCheck:   poolMetrics.LastHealthCheck,
		HealthStatus:      poolMetrics.HealthStatus,
	}
}

// IncrementQueryCount increments the query counter
func IncrementQueryCount() {
	poolMetrics.mu.Lock()
	poolMetrics.QueryCount++
	poolMetrics.mu.Unlock()
}

// IncrementErrorCount increments the error counter
func IncrementErrorCount() {
	poolMetrics.mu.Lock()
	poolMetrics.ErrorCount++
	poolMetrics.mu.Unlock()
}

// OptimizePoolForLoad dynamically adjusts pool size based on load
func OptimizePoolForLoad(db *sqlx.DB, currentLoad int) {
	if poolConfig == nil {
		return
	}
	
	stats := db.Stats()
	utilizationRate := float64(stats.InUse) / float64(poolConfig.MaxOpenConns)
	
	// If utilization is consistently high, consider increasing pool size
	if utilizationRate > 0.8 && stats.WaitCount > 10 {
		newMax := poolConfig.MaxOpenConns + 2
		if newMax <= 50 { // Safety cap
			db.SetMaxOpenConns(newMax)
			poolConfig.MaxOpenConns = newMax
			log.Printf("Increased max connections to %d due to high load", newMax)
		}
	}
	
	// If utilization is consistently low, consider decreasing pool size
	if utilizationRate < 0.2 && poolConfig.MaxOpenConns > 5 {
		newMax := poolConfig.MaxOpenConns - 1
		db.SetMaxOpenConns(newMax)
		poolConfig.MaxOpenConns = newMax
		log.Printf("Decreased max connections to %d due to low load", newMax)
	}
}

// WrapQueryWithMetrics wraps a query function with metrics collection
func WrapQueryWithMetrics(queryFunc func() error) error {
	start := time.Now()
	err := queryFunc()
	
	IncrementQueryCount()
	if err != nil {
		IncrementErrorCount()
	}
	
	// Log slow queries
	duration := time.Since(start)
	if duration > 100*time.Millisecond {
		log.Printf("Slow query detected: %v", duration)
	}
	
	return err
}

// GetOptimalPoolSize calculates optimal pool size based on system resources
func GetOptimalPoolSize() (maxOpen, maxIdle int) {
	numCPU := runtime.NumCPU()
	
	// For web applications: connections = ((core_count * 2) + effective_spindle_count)
	// Assuming SSD (spindle_count = 1)
	maxOpen = (numCPU * 2) + 1
	
	// Ensure reasonable bounds
	if maxOpen < 5 {
		maxOpen = 5
	}
	if maxOpen > 30 {
		maxOpen = 30
	}
	
	// Idle connections should be 25-50% of max
	maxIdle = maxOpen / 3
	if maxIdle < 2 {
		maxIdle = 2
	}
	
	return maxOpen, maxIdle
}

// MonitorPoolHealth returns a health status report for the connection pool
func MonitorPoolHealth(db *sqlx.DB) map[string]interface{} {
	if db == nil {
		return map[string]interface{}{
			"status": "error",
			"error":  "database connection is nil",
		}
	}
	
	stats := db.Stats()
	metrics := GetPoolMetrics()
	
	health := map[string]interface{}{
		"status":              "healthy",
		"open_connections":    stats.OpenConnections,
		"in_use":             stats.InUse,
		"idle":               stats.Idle,
		"wait_count":         stats.WaitCount,
		"wait_duration_ms":   stats.WaitDuration.Milliseconds(),
		"max_idle_closed":    stats.MaxIdleClosed,
		"max_lifetime_closed": stats.MaxLifetimeClosed,
		"total_queries":      metrics.QueryCount,
		"total_errors":       metrics.ErrorCount,
		"last_health_check":  metrics.LastHealthCheck,
		"database_healthy":   metrics.HealthStatus,
		"utilization_rate":   float64(stats.InUse) / float64(poolConfig.MaxOpenConns) * 100,
	}
	
	// Determine overall health status
	if !metrics.HealthStatus {
		health["status"] = "unhealthy"
	} else if stats.WaitCount > 100 {
		health["status"] = "degraded"
		health["warning"] = "high connection wait count"
	} else if float64(stats.InUse)/float64(poolConfig.MaxOpenConns) > 0.9 {
		health["status"] = "warning"
		health["warning"] = "connection pool near capacity"
	}
	
	return health
}