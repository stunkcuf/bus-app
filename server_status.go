package main

import (
	"encoding/json"
	"net/http"
	"runtime"
	"time"
)

var (
	serverStartTime = time.Now()
)

// ServerStatus represents the current server state
type ServerStatus struct {
	Status       string    `json:"status"`
	Uptime       string    `json:"uptime"`
	StartTime    time.Time `json:"start_time"`
	DatabaseConn bool      `json:"database_connected"`
	Version      string    `json:"version"`
	GoVersion    string    `json:"go_version"`
	NumGoroutine int       `json:"goroutines"`
	MemoryStats  struct {
		Alloc      uint64 `json:"allocated_mb"`
		TotalAlloc uint64 `json:"total_allocated_mb"`
		Sys        uint64 `json:"system_mb"`
		NumGC      uint32 `json:"garbage_collections"`
	} `json:"memory_stats"`
	SessionCount int `json:"active_sessions"`
}

// serverStatusHandler returns server health and status information
func serverStatusHandler(w http.ResponseWriter, r *http.Request) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	status := ServerStatus{
		Status:       "healthy",
		Uptime:       time.Since(serverStartTime).Round(time.Second).String(),
		StartTime:    serverStartTime,
		DatabaseConn: db != nil && db.Ping() == nil,
		Version:      "1.0.0",
		GoVersion:    runtime.Version(),
		NumGoroutine: runtime.NumGoroutine(),
		SessionCount: GetActiveSessionCount(),
	}

	// Convert bytes to MB
	status.MemoryStats.Alloc = m.Alloc / 1024 / 1024
	status.MemoryStats.TotalAlloc = m.TotalAlloc / 1024 / 1024
	status.MemoryStats.Sys = m.Sys / 1024 / 1024
	status.MemoryStats.NumGC = m.NumGC

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}