package main

import (
	"sync"
	"time"
)

// DataCache provides thread-safe caching for frequently accessed data
type DataCache struct {
	mu        sync.RWMutex
	buses     []Bus
	vehicles  []Vehicle
	routes    []Route
	users     []User
	students  []Student
	lastFetch map[string]time.Time
	ttl       time.Duration
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
		return cachedBuses, nil
	}

	// Fetch from database
	buses, err := loadBusesFromDB()
	if err != nil {
		return nil, err
	}

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

// invalidateAll clears all cached data
func (c *DataCache) invalidateAll() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.buses = nil
	c.vehicles = nil
	c.routes = nil
	c.users = nil
	c.students = nil
	c.lastFetch = make(map[string]time.Time)
}

// Global cache instance
var dataCache = &DataCache{
	lastFetch: make(map[string]time.Time),
	ttl:       5 * time.Minute,
}
