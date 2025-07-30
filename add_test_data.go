package main

import (
	"log"
	"time"
)

func AddTestData() error {
	log.Println("📝 Adding test data...")

	// Add test drivers if none exist
	var driverCount int
	err := db.Get(&driverCount, `SELECT COUNT(*) FROM users WHERE role = 'driver'`)
	if err != nil {
		return err
	}

	if driverCount == 0 {
		log.Println("Adding test drivers...")
		drivers := []struct {
			username string
			password string
		}{
			{"driver1", "password123"},
			{"driver2", "password123"},
			{"driver3", "password123"},
			{"driver4", "password123"},
			{"driver5", "password123"},
		}

		for _, d := range drivers {
			hashedPassword, err := hashPassword(d.password)
			if err != nil {
				log.Printf("Error hashing password: %v", err)
				continue
			}

			_, err = db.Exec(`
				INSERT INTO users (username, password, role, status, created_at, registration_date)
				VALUES ($1, $2, 'driver', 'active', $3, $4)
				ON CONFLICT (username) DO NOTHING
			`, d.username, hashedPassword, time.Now(), time.Now())
			
			if err != nil {
				log.Printf("Error adding driver %s: %v", d.username, err)
			} else {
				log.Printf("Added driver: %s", d.username)
			}
		}
	}

	// Add test routes if none exist
	var routeCount int
	err = db.Get(&routeCount, `SELECT COUNT(*) FROM routes`)
	if err != nil {
		return err
	}

	if routeCount == 0 {
		log.Println("Adding test routes...")
		routes := []struct {
			routeID   string
			routeName string
		}{
			{"R001", "North Elementary Route"},
			{"R002", "South Elementary Route"},
			{"R003", "East Middle School Route"},
			{"R004", "West Middle School Route"},
			{"R005", "Central High School Route"},
			{"R006", "Downtown Route"},
			{"R007", "Suburban Route"},
			{"R008", "Rural Route North"},
			{"R009", "Rural Route South"},
			{"R010", "Express Route"},
		}

		for _, r := range routes {
			_, err = db.Exec(`
				INSERT INTO routes (route_id, route_name, created_at)
				VALUES ($1, $2, $3)
				ON CONFLICT (route_id) DO NOTHING
			`, r.routeID, r.routeName, time.Now())
			
			if err != nil {
				log.Printf("Error adding route %s: %v", r.routeID, err)
			} else {
				log.Printf("Added route: %s - %s", r.routeID, r.routeName)
			}
		}
	}

	// Clear the cache to ensure fresh data
	dataCache.invalidateUsers()
	dataCache.clearRoutes()

	log.Println("✅ Test data added successfully")
	return nil
}