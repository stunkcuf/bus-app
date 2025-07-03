package main

import (
	"log"
	"sync"
	"time"
)

// DataCache implements a thread-safe cache with TTL for frequently accessed data
type DataCache struct {
	mu    sync.RWMutex
	ttl   time.Duration
	
	// User cache
	users      []User
	usersTime  time.Time
	
	// Bus cache
	buses      []*Bus
	busesTime  time.Time
	
	// Route cache
	routes     []Route
	routesTime time.Time
	
	// Vehicle cache
	vehicles     []Vehicle
	vehiclesTime time.Time
	
	// Student cache
	students     []Student
	studentsTime time.Time
}

// GetUsers returns cached users or loads fresh data if expired
func (c *DataCache) GetUsers() ([]User, error) {
	c.mu.RLock()
	if time.Since(c.usersTime) < c.ttl && c.users != nil {
		users := make([]User, len(c.users))
		copy(users, c.users)
		c.mu.RUnlock()
		return users, nil
	}
	c.mu.RUnlock()
	
	// Load fresh data
	c.mu.Lock()
	defer c.mu.Unlock()
	
	// Double-check after acquiring write lock
	if time.Since(c.usersTime) < c.ttl && c.users != nil {
		users := make([]User, len(c.users))
		copy(users, c.users)
		return users, nil
	}
	
	log.Println("Cache miss: loading users from database")
	users, err := loadUsersFromDB()
	if err != nil {
		// If we have stale data and can't refresh, return stale data
		if c.users != nil {
			log.Printf("Error refreshing users cache, returning stale data: %v", err)
			users := make([]User, len(c.users))
			copy(users, c.users)
			return users, nil
		}
		return nil, err
	}
	
	c.users = users
	c.usersTime = time.Now()
	
	// Return a copy
	result := make([]User, len(users))
	copy(result, users)
	return result, nil
}

// GetBuses returns cached buses or loads fresh data if expired
func (c *DataCache) GetBuses() ([]*Bus, error) {
	c.mu.RLock()
	if time.Since(c.busesTime) < c.ttl && c.buses != nil {
		buses := make([]*Bus, len(c.buses))
		copy(buses, c.buses)
		c.mu.RUnlock()
		return buses, nil
	}
	c.mu.RUnlock()
	
	// Load fresh data
	c.mu.Lock()
	defer c.mu.Unlock()
	
	// Double-check after acquiring write lock
	if time.Since(c.busesTime) < c.ttl && c.buses != nil {
		buses := make([]*Bus, len(c.buses))
		copy(buses, c.buses)
		return buses, nil
	}
	
	log.Println("Cache miss: loading buses from database")
	buses, err := loadBusesFromDB()
	if err != nil {
		// If we have stale data and can't refresh, return stale data
		if c.buses != nil {
			log.Printf("Error refreshing buses cache, returning stale data: %v", err)
			buses := make([]*Bus, len(c.buses))
			copy(buses, c.buses)
			return buses, nil
		}
		return nil, err
	}
	
	c.buses = buses
	c.busesTime = time.Now()
	
	// Return a copy
	result := make([]*Bus, len(buses))
	copy(result, buses)
	return result, nil
}

// GetRoutes returns cached routes or loads fresh data if expired
func (c *DataCache) GetRoutes() ([]Route, error) {
	c.mu.RLock()
	if time.Since(c.routesTime) < c.ttl && c.routes != nil {
		routes := make([]Route, len(c.routes))
		copy(routes, c.routes)
		c.mu.RUnlock()
		return routes, nil
	}
	c.mu.RUnlock()
	
	// Load fresh data
	c.mu.Lock()
	defer c.mu.Unlock()
	
	// Double-check after acquiring write lock
	if time.Since(c.routesTime) < c.ttl && c.routes != nil {
		routes := make([]Route, len(c.routes))
		copy(routes, c.routes)
		return routes, nil
	}
	
	log.Println("Cache miss: loading routes from database")
	routes, err := loadRoutesFromDB()
	if err != nil {
		// If we have stale data and can't refresh, return stale data
		if c.routes != nil {
			log.Printf("Error refreshing routes cache, returning stale data: %v", err)
			routes := make([]Route, len(c.routes))
			copy(routes, c.routes)
			return routes, nil
		}
		return nil, err
	}
	
	c.routes = routes
	c.routesTime = time.Now()
	
	// Return a copy
	result := make([]Route, len(routes))
	copy(result, routes)
	return result, nil
}

// GetVehicles returns cached vehicles or loads fresh data if expired
func (c *DataCache) GetVehicles() ([]Vehicle, error) {
	c.mu.RLock()
	if time.Since(c.vehiclesTime) < c.ttl && c.vehicles != nil {
		vehicles := make([]Vehicle, len(c.vehicles))
		copy(vehicles, c.vehicles)
		c.mu.RUnlock()
		return vehicles, nil
	}
	c.mu.RUnlock()
	
	// Load fresh data
	c.mu.Lock()
	defer c.mu.Unlock()
	
	// Double-check after acquiring write lock
	if time.Since(c.vehiclesTime) < c.ttl && c.vehicles != nil {
		vehicles := make([]Vehicle, len(c.vehicles))
		copy(vehicles, c.vehicles)
		return vehicles, nil
	}
	
	log.Println("Cache miss: loading vehicles from database")
	vehicles, err := loadVehiclesFromDB()
	if err != nil {
		// If we have stale data and can't refresh, return stale data
		if c.vehicles != nil {
			log.Printf("Error refreshing vehicles cache, returning stale data: %v", err)
			vehicles := make([]Vehicle, len(c.vehicles))
			copy(vehicles, c.vehicles)
			return vehicles, nil
		}
		return nil, err
	}
	
	c.vehicles = vehicles
	c.vehiclesTime = time.Now()
	
	// Return a copy
	result := make([]Vehicle, len(vehicles))
	copy(result, vehicles)
	return result, nil
}

// GetStudents returns cached students or loads fresh data if expired
func (c *DataCache) GetStudents() ([]Student, error) {
	c.mu.RLock()
	if time.Since(c.studentsTime) < c.ttl && c.students != nil {
		students := make([]Student, len(c.students))
		copy(students, c.students)
		c.mu.RUnlock()
		return students, nil
	}
	c.mu.RUnlock()
	
	// Load fresh data
	c.mu.Lock()
	defer c.mu.Unlock()
	
	// Double-check after acquiring write lock
	if time.Since(c.studentsTime) < c.ttl && c.students != nil {
		students := make([]Student, len(c.students))
		copy(students, c.students)
		return students, nil
	}
	
	log.Println("Cache miss: loading students from database")
	students, err := loadStudentsFromDB()
	if err != nil {
		// If we have stale data and can't refresh, return stale data
		if c.students != nil {
			log.Printf("Error refreshing students cache, returning stale data: %v", err)
			students := make([]Student, len(c.students))
			copy(students, c.students)
			return students, nil
		}
		return nil, err
	}
	
	c.students = students
	c.studentsTime = time.Now()
	
	// Return a copy
	result := make([]Student, len(students))
	copy(result, students)
	return result, nil
}

// Invalidation methods to clear cache when data changes

// InvalidateUsers clears the user cache
func (c *DataCache) InvalidateUsers() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.users = nil
	c.usersTime = time.Time{}
	log.Println("User cache invalidated")
}

// InvalidateBuses clears the bus cache
func (c *DataCache) InvalidateBuses() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.buses = nil
	c.busesTime = time.Time{}
	log.Println("Bus cache invalidated")
}

// InvalidateRoutes clears the route cache
func (c *DataCache) InvalidateRoutes() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.routes = nil
	c.routesTime = time.Time{}
	log.Println("Route cache invalidated")
}

// InvalidateVehicles clears the vehicle cache
func (c *DataCache) InvalidateVehicles() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.vehicles = nil
	c.vehiclesTime = time.Time{}
	log.Println("Vehicle cache invalidated")
}

// InvalidateStudents clears the student cache
func (c *DataCache) InvalidateStudents() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.students = nil
	c.studentsTime = time.Time{}
	log.Println("Student cache invalidated")
}

// InvalidateAll clears all caches
func (c *DataCache) InvalidateAll() {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.users = nil
	c.usersTime = time.Time{}
	
	c.buses = nil
	c.busesTime = time.Time{}
	
	c.routes = nil
	c.routesTime = time.Time{}
	
	c.vehicles = nil
	c.vehiclesTime = time.Time{}
	
	c.students = nil
	c.studentsTime = time.Time{}
	
	log.Println("All caches invalidated")
}

// SetTTL updates the cache TTL
func (c *DataCache) SetTTL(ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ttl = ttl
	log.Printf("Cache TTL set to %v", ttl)
}

// GetStats returns cache statistics
func (c *DataCache) GetStats() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	stats := map[string]interface{}{
		"ttl": c.ttl.String(),
		"users": map[string]interface{}{
			"cached": c.users != nil,
			"age":    time.Since(c.usersTime).String(),
			"count":  len(c.users),
		},
		"buses": map[string]interface{}{
			"cached": c.buses != nil,
			"age":    time.Since(c.busesTime).String(),
			"count":  len(c.buses),
		},
		"routes": map[string]interface{}{
			"cached": c.routes != nil,
			"age":    time.Since(c.routesTime).String(),
			"count":  len(c.routes),
		},
		"vehicles": map[string]interface{}{
			"cached": c.vehicles != nil,
			"age":    time.Since(c.vehiclesTime).String(),
			"count":  len(c.vehicles),
		},
		"students": map[string]interface{}{
			"cached": c.students != nil,
			"age":    time.Since(c.studentsTime).String(),
			"count":  len(c.students),
		},
	}
	
	return stats
}
