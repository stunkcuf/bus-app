package main

import (
	"log"
	"sync"
	"time"
)

// DataCache provides thread-safe caching for frequently accessed data
type DataCache struct {
	mu                   sync.RWMutex
	buses                []Bus
	vehicles             []Vehicle
	consolidatedVehicles []ConsolidatedVehicle
	routes               []Route
	users                []User
	students             []Student
	lastFetch            map[string]time.Time
	ttl                  time.Duration
}

// NewDataCache creates a new data cache instance
func NewDataCache(ttl time.Duration) *DataCache {
	return &DataCache{
		lastFetch: make(map[string]time.Time),
		ttl:       ttl,
	}
}

// getBuses returns cached buses or fetches from database
func (c *DataCache) getBuses() ([]Bus, error) {
	c.mu.RLock()
	lastFetch, exists := c.lastFetch["buses"]
	cachedBuses := c.buses
	c.mu.RUnlock()

	// If cache is valid, return cached data
	if exists && time.Since(lastFetch) < c.ttl && len(cachedBuses) > 0 {
		log.Printf("DEBUG: Returning %d buses from cache", len(cachedBuses))
		return cachedBuses, nil
	}

	// Fetch from database
	log.Printf("DEBUG: Loading buses from database")
	buses, err := loadBusesFromDB()
	if err != nil {
		log.Printf("ERROR: Failed to load buses from DB: %v", err)
		return nil, err
	}
	
	log.Printf("DEBUG: Loaded %d buses from database", len(buses))

	// Update cache
	c.mu.Lock()
	c.buses = buses
	c.lastFetch["buses"] = time.Now()
	c.mu.Unlock()

	return buses, nil
}

// getVehicles returns cached vehicles or fetches from database
func (c *DataCache) getVehicles() ([]Vehicle, error) {
	c.mu.RLock()
	lastFetch, exists := c.lastFetch["vehicles"]
	cachedVehicles := c.vehicles
	c.mu.RUnlock()

	// If cache is valid, return cached data
	if exists && time.Since(lastFetch) < c.ttl && len(cachedVehicles) > 0 {
		return cachedVehicles, nil
	}

	// Fetch from database
	vehicles, err := loadVehiclesFromDB()
	if err != nil {
		return nil, err
	}

	// Update cache
	c.mu.Lock()
	c.vehicles = vehicles
	c.lastFetch["vehicles"] = time.Now()
	c.mu.Unlock()

	return vehicles, nil
}

// getRoutes returns cached routes or fetches from database
func (c *DataCache) getRoutes() ([]Route, error) {
	c.mu.RLock()
	lastFetch, exists := c.lastFetch["routes"]
	cachedRoutes := c.routes
	c.mu.RUnlock()

	// If cache is valid, return cached data
	if exists && time.Since(lastFetch) < c.ttl && len(cachedRoutes) > 0 {
		return cachedRoutes, nil
	}

	// Fetch from database
	routes, err := loadRoutesFromDB()
	if err != nil {
		return nil, err
	}

	// Update cache
	c.mu.Lock()
	c.routes = routes
	c.lastFetch["routes"] = time.Now()
	c.mu.Unlock()

	return routes, nil
}

// getUsers returns cached users or fetches from database
func (c *DataCache) getUsers() ([]User, error) {
	c.mu.RLock()
	lastFetch, exists := c.lastFetch["users"]
	cachedUsers := c.users
	c.mu.RUnlock()

	// If cache is valid, return cached data
	if exists && time.Since(lastFetch) < c.ttl && len(cachedUsers) > 0 {
		return cachedUsers, nil
	}

	// Fetch from database
	users, err := loadUsersFromDB()
	if err != nil {
		return nil, err
	}

	// Update cache
	c.mu.Lock()
	c.users = users
	c.lastFetch["users"] = time.Now()
	c.mu.Unlock()

	return users, nil
}

// getStudents returns cached students or fetches from database
func (c *DataCache) getStudents() ([]Student, error) {
	c.mu.RLock()
	lastFetch, exists := c.lastFetch["students"]
	cachedStudents := c.students
	c.mu.RUnlock()

	// If cache is valid, return cached data
	if exists && time.Since(lastFetch) < c.ttl && len(cachedStudents) > 0 {
		return cachedStudents, nil
	}

	// Fetch from database
	students, err := loadStudentsFromDB()
	if err != nil {
		return nil, err
	}

	// Update cache
	c.mu.Lock()
	c.students = students
	c.lastFetch["students"] = time.Now()
	c.mu.Unlock()

	return students, nil
}

// Invalidation methods
func (c *DataCache) invalidateBuses() {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.lastFetch, "buses")
	c.buses = nil
}

func (c *DataCache) invalidateVehicles() {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.lastFetch, "vehicles")
	c.vehicles = nil
}

func (c *DataCache) invalidateRoutes() {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.lastFetch, "routes")
	c.routes = nil
}

func (c *DataCache) invalidateUsers() {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.lastFetch, "users")
	c.users = nil
}

func (c *DataCache) invalidateStudents() {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.lastFetch, "students")
	c.students = nil
}

// getConsolidatedVehicles returns cached consolidated vehicles or fetches from database
func (c *DataCache) getConsolidatedVehicles() ([]ConsolidatedVehicle, error) {
	c.mu.RLock()
	lastFetch, exists := c.lastFetch["consolidatedVehicles"]
	cachedVehicles := c.consolidatedVehicles
	c.mu.RUnlock()

	// If cache is valid, return cached data
	if exists && time.Since(lastFetch) < c.ttl && len(cachedVehicles) > 0 {
		log.Printf("Returning %d consolidated vehicles from cache", len(cachedVehicles))
		return cachedVehicles, nil
	}

	// Fetch from database
	vehicles, err := loadConsolidatedVehiclesFromDB()
	if err != nil {
		return nil, err
	}

	// Update cache
	c.mu.Lock()
	c.consolidatedVehicles = vehicles
	c.lastFetch["consolidatedVehicles"] = time.Now()
	c.mu.Unlock()

	return vehicles, nil
}

// invalidateConsolidatedVehicles clears the consolidated vehicles cache
func (c *DataCache) invalidateConsolidatedVehicles() {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.consolidatedVehicles = nil
	delete(c.lastFetch, "consolidatedVehicles")
}

// invalidateAll clears all cached data
func (c *DataCache) invalidateAll() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.buses = nil
	c.vehicles = nil
	c.consolidatedVehicles = nil
	c.routes = nil
	c.users = nil
	c.students = nil
	c.lastFetch = make(map[string]time.Time)
}

// getStats returns cache statistics
func (c *DataCache) getStats() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stats := map[string]interface{}{
		"ttl_minutes":   c.ttl.Minutes(),
		"cached_tables": make(map[string]interface{}),
	}

	tables := stats["cached_tables"].(map[string]interface{})

	if len(c.buses) > 0 {
		tables["buses"] = map[string]interface{}{
			"count":      len(c.buses),
			"last_fetch": c.lastFetch["buses"],
		}
	}

	if len(c.vehicles) > 0 {
		tables["vehicles"] = map[string]interface{}{
			"count":      len(c.vehicles),
			"last_fetch": c.lastFetch["vehicles"],
		}
	}

	if len(c.routes) > 0 {
		tables["routes"] = map[string]interface{}{
			"count":      len(c.routes),
			"last_fetch": c.lastFetch["routes"],
		}
	}

	if len(c.users) > 0 {
		tables["users"] = map[string]interface{}{
			"count":      len(c.users),
			"last_fetch": c.lastFetch["users"],
		}
	}

	if len(c.students) > 0 {
		tables["students"] = map[string]interface{}{
			"count":      len(c.students),
			"last_fetch": c.lastFetch["students"],
		}
	}

	return stats
}

// clear removes all cached data (alias for invalidateAll)
func (c *DataCache) clear() {
	c.invalidateAll()
}

// Global cache instance
var dataCache = &DataCache{
	lastFetch: make(map[string]time.Time),
	ttl:       5 * time.Minute,
}
