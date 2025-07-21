package main

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// QueryCache represents a thread-safe query result cache
type QueryCache struct {
	mu      sync.RWMutex
	cache   map[string]*CacheEntry
	maxSize int
	ttl     time.Duration
}

// CacheEntry represents a cached query result
type CacheEntry struct {
	Data      interface{}
	CreatedAt time.Time
	ExpiresAt time.Time
	HitCount  int
}

// NewQueryCache creates a new query cache
func NewQueryCache(maxSize int, ttl time.Duration) *QueryCache {
	qc := &QueryCache{
		cache:   make(map[string]*CacheEntry),
		maxSize: maxSize,
		ttl:     ttl,
	}

	// Start cleanup goroutine
	go qc.cleanupExpired()

	return qc
}

// generateKey creates a cache key from query and parameters
func (qc *QueryCache) generateKey(query string, params ...interface{}) string {
	// Create a unique key from query and parameters
	keyData := struct {
		Query  string      `json:"query"`
		Params interface{} `json:"params"`
	}{
		Query:  query,
		Params: params,
	}

	data, _ := json.Marshal(keyData)
	hash := md5.Sum(data)
	return fmt.Sprintf("%x", hash)
}

// Get retrieves a value from the cache
func (qc *QueryCache) Get(query string, params ...interface{}) (interface{}, bool) {
	key := qc.generateKey(query, params...)

	qc.mu.RLock()
	defer qc.mu.RUnlock()

	entry, exists := qc.cache[key]
	if !exists {
		return nil, false
	}

	// Check if entry is expired
	if time.Now().After(entry.ExpiresAt) {
		// Mark for cleanup but don't remove here to avoid blocking
		return nil, false
	}

	// Increment hit count
	entry.HitCount++

	return entry.Data, true
}

// Set stores a value in the cache
func (qc *QueryCache) Set(query string, data interface{}, params ...interface{}) {
	key := qc.generateKey(query, params...)

	qc.mu.Lock()
	defer qc.mu.Unlock()

	// Check if we need to evict entries
	if len(qc.cache) >= qc.maxSize {
		qc.evictLRU()
	}

	now := time.Now()
	qc.cache[key] = &CacheEntry{
		Data:      data,
		CreatedAt: now,
		ExpiresAt: now.Add(qc.ttl),
		HitCount:  0,
	}
}

// evictLRU removes the least recently used entry (lowest hit count)
func (qc *QueryCache) evictLRU() {
	var oldestKey string
	var lowestHits int = -1

	for key, entry := range qc.cache {
		if lowestHits == -1 || entry.HitCount < lowestHits {
			lowestHits = entry.HitCount
			oldestKey = key
		}
	}

	if oldestKey != "" {
		delete(qc.cache, oldestKey)
	}
}

// cleanupExpired removes expired entries periodically
func (qc *QueryCache) cleanupExpired() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		qc.mu.Lock()
		now := time.Now()

		for key, entry := range qc.cache {
			if now.After(entry.ExpiresAt) {
				delete(qc.cache, key)
			}
		}
		qc.mu.Unlock()
	}
}

// Clear removes all entries from the cache
func (qc *QueryCache) Clear() {
	qc.mu.Lock()
	defer qc.mu.Unlock()
	qc.cache = make(map[string]*CacheEntry)
}

// Stats returns cache statistics
func (qc *QueryCache) Stats() map[string]interface{} {
	qc.mu.RLock()
	defer qc.mu.RUnlock()

	totalHits := 0
	for _, entry := range qc.cache {
		totalHits += entry.HitCount
	}

	return map[string]interface{}{
		"size":        len(qc.cache),
		"max_size":    qc.maxSize,
		"total_hits":  totalHits,
		"ttl_minutes": qc.ttl.Minutes(),
	}
}

// Global query cache instance
var queryCache *QueryCache

// initQueryCache initializes the global query cache
func initQueryCache() {
	// Cache up to 1000 queries for 10 minutes each
	queryCache = NewQueryCache(1000, 10*time.Minute)
	LogInfo("Query cache initialized with 1000 entries, 10 minute TTL")
}

// CachedQuery executes a query with caching
func CachedQuery(dest interface{}, query string, args ...interface{}) error {
	// Try to get from cache first
	if cached, found := queryCache.Get(query, args...); found {
		// Copy cached data to destination
		if cachedBytes, err := json.Marshal(cached); err == nil {
			if err := json.Unmarshal(cachedBytes, dest); err == nil {
				return nil // Cache hit
			}
		}
	}

	// Cache miss - execute query
	if err := db.Select(dest, query, args...); err != nil {
		return err
	}

	// Store result in cache
	queryCache.Set(query, dest, args...)

	return nil
}

// CachedGet executes a single-row query with caching
func CachedGet(dest interface{}, query string, args ...interface{}) error {
	// Try to get from cache first
	if cached, found := queryCache.Get(query, args...); found {
		// Copy cached data to destination
		if cachedBytes, err := json.Marshal(cached); err == nil {
			if err := json.Unmarshal(cachedBytes, dest); err == nil {
				return nil // Cache hit
			}
		}
	}

	// Cache miss - execute query
	if err := db.Get(dest, query, args...); err != nil {
		return err
	}

	// Store result in cache
	queryCache.Set(query, dest, args...)

	return nil
}
