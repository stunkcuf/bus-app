package main

import (
	"fmt"
	"log"
)

func checkAndFixDrivers() error {
	// First, check how many drivers exist
	var driverCount int
	err := db.Get(&driverCount, "SELECT COUNT(*) FROM users WHERE role = 'driver'")
	if err != nil {
		return fmt.Errorf("error counting drivers: %v", err)
	}
	
	fmt.Printf("Total drivers in database: %d\n", driverCount)
	
	// Check how many are active
	var activeCount int
	err = db.Get(&activeCount, "SELECT COUNT(*) FROM users WHERE role = 'driver' AND status = 'active'")
	if err != nil {
		return fmt.Errorf("error counting active drivers: %v", err)
	}
	
	fmt.Printf("Active drivers: %d\n", activeCount)
	
	// If no active drivers, activate some
	if activeCount == 0 && driverCount > 0 {
		fmt.Println("No active drivers found. Activating existing drivers...")
		
		_, err = db.Exec(`
			UPDATE users 
			SET status = 'active' 
			WHERE role = 'driver' 
			AND username IN (
				SELECT username FROM users 
				WHERE role = 'driver' 
				LIMIT 10
			)
		`)
		
		if err != nil {
			return fmt.Errorf("error activating drivers: %v", err)
		}
		
		fmt.Println("Activated up to 10 drivers")
		
		// Clear cache to reload users
		dataCache.invalidateAll()
		
		fmt.Println("Cache cleared")
	}
	
	// List some drivers for verification
	type DriverInfo struct {
		Username string `db:"username"`
		Status   string `db:"status"`
	}
	
	var drivers []DriverInfo
	err = db.Select(&drivers, "SELECT username, status FROM users WHERE role = 'driver' LIMIT 5")
	if err != nil {
		log.Printf("Error listing drivers: %v", err)
	} else {
		fmt.Println("\nSample drivers:")
		for _, d := range drivers {
			fmt.Printf("  - %s (status: %s)\n", d.Username, d.Status)
		}
	}
	
	return nil
}