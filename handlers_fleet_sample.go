package main

import (
	"log"
	"net/http"
	"time"
)

// addSampleFleetDataHandler adds sample fleet vehicles for demonstration
func addSampleFleetDataHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	if db == nil {
		http.Error(w, "Database not initialized", http.StatusInternalServerError)
		return
	}

	// Sample fleet vehicles data (40+ vehicles as requested)
	vehicles := []struct {
		VehicleID       string
		Model           string
		Description     string
		Year            string
		TireSize        string
		License         string
		OilStatus       string
		TireStatus      string
		Status          string
		MaintenanceNotes string
		SerialNumber    string
		Base            string
		ServiceInterval int
		CurrentMileage  int
		LastOilChange   int
		LastTireService int
	}{
		// Service trucks
		{"FV001", "Ford F-250", "Service Truck #1", "2019", "265/70R17", "IL-SRV-001", "good", "good", "active", "Regular maintenance up to date", "1FTBF2B68KED12345", "Main Base", 5000, 45230, 42000, 40000},
		{"FV002", "Chevrolet Silverado 2500", "Service Truck #2", "2020", "265/70R17", "IL-SRV-002", "good", "good", "active", "", "1GC4KVCY5LF123456", "Main Base", 5000, 38450, 36000, 35000},
		{"FV003", "GMC Sierra 3500", "Heavy Duty Service", "2018", "275/70R18", "IL-SRV-003", "good", "maintenance", "active", "Tire rotation due", "1GT42VCY7JF234567", "North Base", 5000, 67890, 65000, 62000},
		{"FV004", "Ford F-350", "Service Truck #4", "2021", "275/65R18", "IL-SRV-004", "good", "good", "active", "", "1FT8W3BT5MED34567", "Main Base", 5000, 23100, 20000, 18000},
		{"FV005", "Ram 2500", "Service Truck #5", "2019", "265/70R17", "IL-SRV-005", "maintenance", "good", "active", "Oil change due soon", "3C6TR5DT2KG567890", "South Base", 5000, 52300, 47500, 45000},
		
		// Vans
		{"FV006", "Ford Transit 350", "Passenger Van #1", "2020", "235/65R16", "IL-VAN-001", "good", "good", "active", "", "1FBAX2CM5LKA12345", "Main Base", 5000, 41200, 39000, 37000},
		{"FV007", "Chevrolet Express 3500", "Cargo Van", "2019", "245/75R16", "IL-VAN-002", "good", "good", "active", "", "1GAZG3FG5K1234567", "Main Base", 7500, 58900, 55000, 52000},
		{"FV008", "Ford Transit 250", "Passenger Van #2", "2021", "235/65R16", "IL-VAN-003", "good", "good", "active", "", "1FBZX2YM7MKB23456", "East Base", 5000, 18700, 15000, 14000},
		{"FV009", "GMC Savana 2500", "Cargo Van #2", "2018", "245/75R16", "IL-VAN-004", "good", "maintenance", "active", "Needs tire alignment", "1GTW7AFG9J1234567", "Main Base", 7500, 73200, 70000, 67000},
		{"FV010", "Nissan NV200", "Compact Van", "2020", "185/60R15", "IL-VAN-005", "good", "good", "active", "", "3N6CM0KN7LK789012", "West Base", 5000, 34500, 32000, 30000},
		
		// SUVs
		{"FV011", "Chevrolet Tahoe", "Supervisor Vehicle #1", "2021", "275/60R20", "IL-SUV-001", "good", "good", "active", "", "1GNSKCKC5MR123456", "Admin", 5000, 15600, 13000, 12000},
		{"FV012", "Ford Explorer", "Supervisor Vehicle #2", "2020", "255/60R18", "IL-SUV-002", "good", "good", "active", "", "1FM5K8D85LGA12345", "Admin", 5000, 28900, 26000, 24000},
		{"FV013", "GMC Yukon", "Executive Transport", "2022", "275/60R20", "IL-SUV-003", "good", "good", "active", "Premium vehicle", "1GKS2CKJ5NR234567", "Admin", 5000, 8900, 7500, 7000},
		{"FV014", "Toyota Highlander", "Staff Vehicle", "2019", "245/60R18", "IL-SUV-004", "good", "good", "active", "", "5TDBZRFH7KS345678", "North Base", 5000, 45600, 43000, 41000},
		{"FV015", "Honda Pilot", "Staff Vehicle #2", "2020", "245/60R18", "IL-SUV-005", "maintenance", "good", "active", "Oil change needed", "5FNYF6H57LB456789", "South Base", 5000, 37800, 32500, 31000},
		
		// Maintenance vehicles
		{"FV016", "Ford F-550", "Tow Truck", "2018", "225/70R19.5", "IL-TOW-001", "good", "good", "active", "Heavy duty towing", "1FDUF5HT7JEA12345", "Main Base", 7500, 89200, 85000, 82000},
		{"FV017", "International 4300", "Box Truck", "2017", "11R22.5", "IL-BOX-001", "good", "critical", "maintenance", "Tire replacement urgent", "1HTMMAAL8HH123456", "Main Base", 10000, 125600, 120000, 115000},
		{"FV018", "Freightliner Sprinter", "Mobile Repair", "2019", "195/75R16", "IL-RPR-001", "good", "good", "active", "Fully equipped workshop", "WD4PF0CD5KP234567", "Main Base", 5000, 67300, 65000, 63000},
		{"FV019", "Chevrolet 3500HD", "Utility Truck", "2020", "235/80R17", "IL-UTL-001", "good", "good", "active", "", "1GB3CVCT6LF345678", "East Base", 5000, 43100, 40000, 38000},
		{"FV020", "Ford F-450", "Flatbed Truck", "2019", "225/70R19.5", "IL-FLT-001", "good", "maintenance", "active", "Brake inspection due", "1FD0W4HT5KED45678", "North Base", 7500, 71200, 68000, 65000},
		
		// Specialty vehicles
		{"FV021", "Kubota RTV-X1100C", "Utility Vehicle", "2020", "25x10-12", "IL-UTV-001", "good", "good", "active", "Off-road capable", "KBCPX110CLF567890", "Main Base", 100, 2340, 2200, 2100},
		{"FV022", "John Deere Gator", "Grounds Vehicle", "2019", "25x11-12", "IL-UTV-002", "good", "good", "active", "", "1M0865LABKM678901", "South Base", 100, 1850, 1700, 1600},
		{"FV023", "Club Car Carryall", "Campus Transport", "2021", "18x8.5-8", "IL-GC-001", "good", "good", "active", "Electric vehicle", "PH2137-123456", "Admin", 0, 0, 0, 0},
		{"FV024", "Cushman Hauler", "Maintenance Cart", "2018", "18x8.5-8", "IL-GC-002", "good", "good", "active", "", "2HC1928374", "Main Base", 0, 0, 0, 0},
		{"FV025", "Bobcat S650", "Skid Steer", "2019", "12-16.5", "IL-SSL-001", "good", "good", "seasonal", "Snow removal equipment", "AHGL12345", "North Base", 250, 1890, 1750, 1700},
		
		// Additional service vehicles
		{"FV026", "Ford E-450", "Shuttle Bus", "2020", "225/75R16", "IL-SHT-001", "good", "good", "active", "15 passenger", "1FDXE4FS5LDA12345", "Main Base", 5000, 56700, 54000, 52000},
		{"FV027", "Chevrolet Express 4500", "Activity Bus", "2019", "225/75R16", "IL-ACT-001", "good", "good", "active", "Wheelchair accessible", "1GB3GRFG2K1234567", "East Base", 5000, 48900, 46000, 44000},
		{"FV028", "Ford Transit 350HD", "Cargo Van #3", "2021", "235/65R16", "IL-VAN-006", "good", "good", "active", "", "1FBVU4XM7MKA23456", "West Base", 5000, 21300, 19000, 17000},
		{"FV029", "Ram ProMaster 3500", "Delivery Van", "2020", "225/75R16", "IL-DLV-001", "maintenance", "good", "active", "AC service needed", "3C6URVJG7LE234567", "Main Base", 5000, 63400, 58000, 56000},
		{"FV030", "Nissan Frontier", "Pickup Truck", "2019", "265/70R16", "IL-PU-001", "good", "good", "active", "", "1N6AD0EV5KN345678", "South Base", 5000, 41200, 39000, 37000},
		
		// Emergency/backup vehicles
		{"FV031", "Ford F-150", "Backup Truck #1", "2018", "275/65R18", "IL-BKP-001", "good", "maintenance", "standby", "Reserve vehicle", "1FTFW1ET7JKC45678", "Main Base", 5000, 35600, 33000, 30000},
		{"FV032", "Chevrolet Colorado", "Backup Truck #2", "2017", "265/60R18", "IL-BKP-002", "good", "good", "standby", "Reserve vehicle", "1GCGTCE38H1234567", "East Base", 5000, 42100, 40000, 38000},
		{"FV033", "GMC Canyon", "Backup Truck #3", "2019", "265/60R18", "IL-BKP-003", "good", "good", "standby", "Reserve vehicle", "1GTG6CEN0K1345678", "West Base", 5000, 28900, 26000, 24000},
		{"FV034", "Toyota Tacoma", "Emergency Response", "2020", "265/70R16", "IL-EMR-001", "good", "good", "active", "First aid equipped", "3TMCZ5AN7LM456789", "Admin", 5000, 31400, 29000, 27000},
		{"FV035", "Honda Ridgeline", "Backup Truck #4", "2021", "245/60R18", "IL-BKP-004", "good", "good", "standby", "", "5FPYK3F75MB567890", "North Base", 5000, 19800, 17000, 15000},
		
		// Older vehicles (still in service)
		{"FV036", "Ford F-250", "Old Reliable", "2015", "245/75R17", "IL-OLD-001", "maintenance", "critical", "active", "High mileage unit", "1FT7W2B65FEA67890", "South Base", 3000, 189500, 187000, 182000},
		{"FV037", "Chevrolet 2500HD", "Vintage Service", "2014", "245/75R17", "IL-OLD-002", "critical", "maintenance", "limited", "Parts availability issue", "1GC1KXE87EF789012", "Main Base", 3000, 205300, 202000, 198000},
		{"FV038", "GMC Sierra 1500", "Old Timer", "2016", "265/70R17", "IL-OLD-003", "good", "good", "active", "Well maintained", "3GTU2MEC2GG890123", "East Base", 5000, 156700, 154000, 150000},
		{"FV039", "Ford E-350", "Legacy Van", "2015", "225/75R16", "IL-OLD-004", "good", "maintenance", "active", "Transmission service due", "1FTSS3EL5FDA90123", "West Base", 5000, 178900, 175000, 170000},
		{"FV040", "Dodge Ram 1500", "Veteran Truck", "2013", "275/60R20", "IL-OLD-005", "maintenance", "good", "limited", "Semi-retired", "1C6RR7FT5DS012345", "North Base", 3000, 223400, 220000, 215000},
		
		// Recent additions
		{"FV041", "Ford Maverick", "Compact Truck", "2022", "225/65R17", "IL-NEW-001", "good", "good", "active", "Fuel efficient", "3FTTW8E39NRA12345", "Admin", 5000, 5600, 5000, 4500},
		{"FV042", "Rivian R1T", "Electric Truck", "2023", "275/65R20", "IL-EV-001", "good", "good", "active", "Zero emissions", "7FCMGEE42NA123456", "Admin", 0, 3200, 0, 3000},
	}

	added := 0
	for _, v := range vehicles {
		_, err := db.Exec(`
			INSERT INTO vehicles (
				vehicle_id, model, description, year, tire_size, license,
				oil_status, tire_status, status, maintenance_notes, serial_number,
				base, service_interval, current_mileage, last_oil_change, last_tire_service,
				created_at, updated_at
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $17
			) ON CONFLICT (vehicle_id) DO UPDATE SET
				model = EXCLUDED.model,
				status = EXCLUDED.status,
				oil_status = EXCLUDED.oil_status,
				tire_status = EXCLUDED.tire_status,
				current_mileage = EXCLUDED.current_mileage,
				updated_at = EXCLUDED.updated_at
		`, v.VehicleID, v.Model, v.Description, v.Year, v.TireSize, v.License,
			v.OilStatus, v.TireStatus, v.Status, v.MaintenanceNotes, v.SerialNumber,
			v.Base, v.ServiceInterval, v.CurrentMileage, v.LastOilChange, v.LastTireService,
			time.Now())

		if err != nil {
			log.Printf("Error inserting vehicle %s: %v", v.VehicleID, err)
		} else {
			added++
		}
	}

	// Also add them to fleet_vehicles table
	for _, v := range vehicles {
		_, err := db.Exec(`
			INSERT INTO fleet_vehicles (
				vehicle_id, type, model, year, status, location, mileage,
				last_service_date, next_service_due, license_plate, vin_number,
				fuel_type, fuel_capacity, mpg_city, mpg_highway, 
				purchase_date, purchase_price, current_value, insurance_policy,
				insurance_expiry, registration_expiry, created_at, updated_at
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $22
			) ON CONFLICT (vehicle_id) DO UPDATE SET
				status = EXCLUDED.status,
				mileage = EXCLUDED.mileage,
				updated_at = EXCLUDED.updated_at
		`, v.VehicleID, "service", v.Model, v.Year, v.Status, v.Base, v.CurrentMileage,
			time.Now().AddDate(0, -2, 0), time.Now().AddDate(0, 2, 0), v.License, v.SerialNumber,
			"gasoline", 26.0, 18, 24,
			time.Now().AddDate(-3, 0, 0), 35000.00, 28000.00, "POL-2024-" + v.VehicleID,
			time.Now().AddDate(0, 6, 0), time.Now().AddDate(1, 0, 0), time.Now())

		if err != nil {
			log.Printf("Error inserting fleet vehicle %s: %v", v.VehicleID, err)
		}
	}

	log.Printf("Added %d fleet vehicles", added)
	
	// Redirect to company fleet page
	http.Redirect(w, r, "/company-fleet", http.StatusSeeOther)
}