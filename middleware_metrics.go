package main

import (
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// MetricsCollector collects request metrics
type MetricsCollector struct {
	requestCount     uint64
	errorCount       uint64
	activeRequests   int32
	endpointMetrics  map[string]*EndpointMetrics
	userSessions     map[string]time.Time
	mu               sync.RWMutex
}

// EndpointMetrics tracks metrics for a specific endpoint
type EndpointMetrics struct {
	count          uint64
	totalDuration  time.Duration
	minDuration    time.Duration
	maxDuration    time.Duration
	lastAccessTime time.Time
	errorCount     uint64
	mu             sync.RWMutex
}

// Global metrics collector instance
var metricsCollector = &MetricsCollector{
	endpointMetrics: make(map[string]*EndpointMetrics),
	userSessions:    make(map[string]time.Time),
}

// MetricsMiddleware tracks request metrics
func MetricsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		// Track active requests
		atomic.AddInt32(&metricsCollector.activeRequests, 1)
		defer atomic.AddInt32(&metricsCollector.activeRequests, -1)
		
		// Increment total request count
		atomic.AddUint64(&metricsCollector.requestCount, 1)
		
		// Create a response writer wrapper to capture status code
		wrapped := &metricsResponseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}
		
		// Process the request
		next(wrapped, r)
		
		// Calculate duration
		duration := time.Since(start)
		
		// Update endpoint metrics
		endpoint := fmt.Sprintf("%s %s", r.Method, r.URL.Path)
		metricsCollector.updateEndpointMetrics(endpoint, duration, wrapped.statusCode)
		
		// Track errors (5xx status codes)
		if wrapped.statusCode >= 500 {
			atomic.AddUint64(&metricsCollector.errorCount, 1)
		}
		
		// Track user sessions
		if sessionID := getSessionID(r); sessionID != "" {
			metricsCollector.updateUserSession(sessionID)
		}
	}
}

// metricsResponseWriter wraps http.ResponseWriter to capture status code
type metricsResponseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

func (rw *metricsResponseWriter) WriteHeader(code int) {
	if !rw.written {
		rw.statusCode = code
		rw.written = true
	}
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *metricsResponseWriter) Write(b []byte) (int, error) {
	if !rw.written {
		rw.written = true
	}
	return rw.ResponseWriter.Write(b)
}

// updateEndpointMetrics updates metrics for a specific endpoint
func (mc *MetricsCollector) updateEndpointMetrics(endpoint string, duration time.Duration, statusCode int) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	metrics, exists := mc.endpointMetrics[endpoint]
	if !exists {
		metrics = &EndpointMetrics{
			minDuration: duration,
			maxDuration: duration,
		}
		mc.endpointMetrics[endpoint] = metrics
	}
	
	metrics.mu.Lock()
	defer metrics.mu.Unlock()
	
	metrics.count++
	metrics.totalDuration += duration
	metrics.lastAccessTime = time.Now()
	
	if duration < metrics.minDuration || metrics.minDuration == 0 {
		metrics.minDuration = duration
	}
	if duration > metrics.maxDuration {
		metrics.maxDuration = duration
	}
	
	if statusCode >= 500 {
		metrics.errorCount++
	}
}

// updateUserSession updates the last activity time for a user session
func (mc *MetricsCollector) updateUserSession(sessionID string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.userSessions[sessionID] = time.Now()
}

// GetMetrics returns current metrics
func (mc *MetricsCollector) GetMetrics() map[string]interface{} {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	
	// Calculate active sessions (sessions active in last 30 minutes)
	activeSessionCount := 0
	cutoffTime := time.Now().Add(-30 * time.Minute)
	for _, lastActivity := range mc.userSessions {
		if lastActivity.After(cutoffTime) {
			activeSessionCount++
		}
	}
	
	// Prepare endpoint statistics
	endpointStats := make(map[string]interface{})
	for endpoint, metrics := range mc.endpointMetrics {
		metrics.mu.RLock()
		avgDuration := time.Duration(0)
		if metrics.count > 0 {
			avgDuration = metrics.totalDuration / time.Duration(metrics.count)
		}
		
		endpointStats[endpoint] = map[string]interface{}{
			"count":          metrics.count,
			"avgDuration":    avgDuration.Milliseconds(),
			"minDuration":    metrics.minDuration.Milliseconds(),
			"maxDuration":    metrics.maxDuration.Milliseconds(),
			"errorCount":     metrics.errorCount,
			"errorRate":      float64(metrics.errorCount) / float64(metrics.count) * 100,
			"lastAccessTime": metrics.lastAccessTime,
		}
		metrics.mu.RUnlock()
	}
	
	return map[string]interface{}{
		"requestCount":    atomic.LoadUint64(&mc.requestCount),
		"errorCount":      atomic.LoadUint64(&mc.errorCount),
		"activeRequests":  atomic.LoadInt32(&mc.activeRequests),
		"activeSessions":  activeSessionCount,
		"endpointStats":   endpointStats,
	}
}

// GetRequestRate calculates requests per minute
func (mc *MetricsCollector) GetRequestRate() float64 {
	// This would need to track requests over time windows
	// For now, return a calculated value based on recent activity
	requestCount := atomic.LoadUint64(&mc.requestCount)
	if requestCount == 0 {
		return 0
	}
	// Simplified calculation - in production, track time windows
	return float64(requestCount) / 10.0 // Assume 10 minutes of operation
}

// GetErrorRate calculates error rate percentage
func (mc *MetricsCollector) GetErrorRate() float64 {
	requestCount := atomic.LoadUint64(&mc.requestCount)
	errorCount := atomic.LoadUint64(&mc.errorCount)
	
	if requestCount == 0 {
		return 0
	}
	
	return (float64(errorCount) / float64(requestCount)) * 100
}

// GetActiveUserCount returns the number of active users
func (mc *MetricsCollector) GetActiveUserCount() int {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	
	activeCount := 0
	cutoffTime := time.Now().Add(-30 * time.Minute)
	for _, lastActivity := range mc.userSessions {
		if lastActivity.After(cutoffTime) {
			activeCount++
		}
	}
	
	return activeCount
}

// GetAverageResponseTime calculates average response time across all endpoints
func (mc *MetricsCollector) GetAverageResponseTime() float64 {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	
	var totalDuration time.Duration
	var totalCount uint64
	
	for _, metrics := range mc.endpointMetrics {
		metrics.mu.RLock()
		totalDuration += metrics.totalDuration
		totalCount += metrics.count
		metrics.mu.RUnlock()
	}
	
	if totalCount == 0 {
		return 0
	}
	
	avgDuration := totalDuration / time.Duration(totalCount)
	return float64(avgDuration.Milliseconds())
}

// CleanupOldSessions removes inactive sessions from tracking
func (mc *MetricsCollector) CleanupOldSessions() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	cutoffTime := time.Now().Add(-24 * time.Hour)
	for sessionID, lastActivity := range mc.userSessions {
		if lastActivity.Before(cutoffTime) {
			delete(mc.userSessions, sessionID)
		}
	}
}

// StartMetricsCleanup starts a goroutine to periodically clean up old data
func StartMetricsCleanup() {
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		
		for range ticker.C {
			metricsCollector.CleanupOldSessions()
		}
	}()
}

// getSessionID extracts session ID from request
func getSessionID(r *http.Request) string {
	cookie, err := r.Cookie(SessionCookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}