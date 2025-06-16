// Add these migration functions to the end of your database.go file

// =============================================================================
// MIGRATION FUNCTIONS - JSON to PostgreSQL
// =============================================================================

// migrateJSONToPostgreSQL orchestrates the migration from JSON files to PostgreSQL
func migrateJSONToPostgreSQL() error {
	log.Println("🔄 Starting migration from JSON files to PostgreSQL...")

	// Migrate Users
	if err := migrateUsers(); err != nil {
		log.Printf("❌ Failed to migrate users: %v", err)
		return err
	}

	// Migrate Buses
	if err := migrateBuses(); err != nil {
		log.Printf("❌ Failed to migrate buses: %v", err)
		return err
	}

	// Migrate Routes
	if err := migrateRoutes(); err != nil {
		log.Printf("❌ Failed to migrate routes: %v", err)
		return err
	}

	// Migrate Students
	if err := migrateStudents(); err != nil {
		log.Printf("❌ Failed to migrate students: %v", err)
		return err
	}

	// Migrate Route Assignments
	if err := migrateRouteAssignments(); err != nil {
		log.Printf("❌ Failed to migrate route assignments: %v", err)
		return err
	}

	// Migrate Driver Logs
	if err := migrateDriverLogs(); err != nil {
		log.Printf("❌ Failed to migrate driver logs: %v", err)
		return err
	}

	// Migrate Maintenance Logs
	if err := migrateMaintenanceLogs(); err != nil {
		log.Printf("❌ Failed to migrate maintenance logs: %v", err)
		return err
	}

	// Migrate Vehicles
	if err := migrateVehicles(); err != nil {
		log.Printf("❌ Failed to migrate vehicles: %v", err)
		return err
	}

	log.Println("✅ Migration completed successfully!")
	return nil
}

func migrateUsers() error {
	users := loadUsersFromJSON()
	if len(users) == 0 {
		log.Println("📝 No users to migrate")
		return nil
	}

	for _, user := range users {
		_, err := db.Exec(`
			INSERT INTO users (username, password, role) 
			VALUES ($1, $2, $3) 
			ON CONFLICT (username) DO UPDATE SET 
				password = EXCLUDED.password,
				role = EXCLUDED.role
		`, user.Username, user.Password, user.Role)
		
		if err != nil {
			return fmt.Errorf("failed to insert user %s: %w", user.Username, err)
		}
	}

	log.Printf("✅ Migrated %d users", len(users))
	return nil
}

func migrateBuses() error {
	buses := loadBusesFromJSON()
	if len(buses) == 0 {
		log.Println("📝 No buses to migrate")
		return nil
	}

	for _, bus := range buses {
		_, err := db.Exec(`
			INSERT INTO buses (bus_id, status, model, capacity, oil_status, tire_status, maintenance_notes) 
			VALUES ($1, $2, $3, $4, $5, $6, $7) 
			ON CONFLICT (bus_id) DO UPDATE SET 
				status = EXCLUDED.status,
				model = EXCLUDED.model,
				capacity = EXCLUDED.capacity,
				oil_status = EXCLUDED.oil_status,
				tire_status = EXCLUDED.tire_status,
				maintenance_notes = EXCLUDED.maintenance_notes,
				updated_at = CURRENT_TIMESTAMP
		`, bus.BusID, bus.Status, bus.Model, bus.Capacity, bus.OilStatus, bus.TireStatus, bus.MaintenanceNotes)
		
		if err != nil {
			return fmt.Errorf("failed to insert bus %s: %w", bus.BusID, err)
		}
	}

	log.Printf("✅ Migrated %d buses", len(buses))
	return nil
}

func migrateRoutes() error {
	routes, err := loadJSON[Route]("data/routes.json")
	if err != nil {
		log.Println("📝 No routes to migrate")
		return nil
	}

	for _, route := range routes {
		positionsJSON, _ := json.Marshal(route.Positions)
		
		_, err := db.Exec(`
			INSERT INTO routes (route_id, route_name, description, positions) 
			VALUES ($1, $2, $3, $4) 
			ON CONFLICT (route_id) DO UPDATE SET 
				route_name = EXCLUDED.route_name,
				description = EXCLUDED.description,
				positions = EXCLUDED.positions
		`, route.RouteID, route.RouteName, route.Description, positionsJSON)
		
		if err != nil {
			return fmt.Errorf("failed to insert route %s: %w", route.RouteID, err)
		}
	}

	log.Printf("✅ Migrated %d routes", len(routes))
	return nil
}

func migrateStudents() error {
	students := loadStudentsFromJSON()
	if len(students) == 0 {
		log.Println("📝 No students to migrate")
		return nil
	}

	for _, student := range students {
		locationsJSON, _ := json.Marshal(student.Locations)
		
		_, err := db.Exec(`
			INSERT INTO students (student_id, name, locations, phone_number, alt_phone_number, 
				guardian, pickup_time, dropoff_time, position_number, route_id, driver, active) 
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12) 
			ON CONFLICT (student_id) DO UPDATE SET 
				name = EXCLUDED.name,
				locations = EXCLUDED.locations,
				phone_number = EXCLUDED.phone_number,
				alt_phone_number = EXCLUDED.alt_phone_number,
				guardian = EXCLUDED.guardian,
				pickup_time = EXCLUDED.pickup_time,
				dropoff_time = EXCLUDED.dropoff_time,
				position_number = EXCLUDED.position_number,
				route_id = EXCLUDED.route_id,
				driver = EXCLUDED.driver,
				active = EXCLUDED.active
		`, student.StudentID, student.Name, locationsJSON, student.PhoneNumber, student.AltPhoneNumber,
		   student.Guardian, student.PickupTime, student.DropoffTime, student.PositionNumber, 
		   student.RouteID, student.Driver, student.Active)
		
		if err != nil {
			return fmt.Errorf("failed to insert student %s: %w", student.StudentID, err)
		}
	}

	log.Printf("✅ Migrated %d students", len(students))
	return nil
}

func migrateRouteAssignments() error {
	assignments, err := loadRouteAssignmentsFromJSON()
	if err != nil {
		log.Println("📝 No route assignments to migrate")
		return nil
	}

	for _, assignment := range assignments {
		_, err := db.Exec(`
			INSERT INTO route_assignments (driver, bus_id, route_id, route_name, assigned_date) 
			VALUES ($1, $2, $3, $4, $5) 
			ON CONFLICT (driver) DO UPDATE SET 
				bus_id = EXCLUDED.bus_id,
				route_id = EXCLUDED.route_id,
				route_name = EXCLUDED.route_name,
				assigned_date = EXCLUDED.assigned_date
		`, assignment.Driver, assignment.BusID, assignment.RouteID, assignment.RouteName, assignment.AssignedDate)
		
		if err != nil {
			return fmt.Errorf("failed to insert assignment for driver %s: %w", assignment.Driver, err)
		}
	}

	log.Printf("✅ Migrated %d route assignments", len(assignments))
	return nil
}

func migrateDriverLogs() error {
	logs, err := loadJSON[DriverLog]("data/driver_logs.json")
	if err != nil {
		log.Println("📝 No driver logs to migrate")
		return nil
	}

	for _, driverLog := range logs {
		attendanceJSON, _ := json.Marshal(driverLog.Attendance)
		
		_, err := db.Exec(`
			INSERT INTO driver_logs (driver, bus_id, route_id, date, period, departure_time, 
				arrival_time, mileage, attendance) 
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) 
			ON CONFLICT (driver, date, period) DO UPDATE SET 
				bus_id = EXCLUDED.bus_id,
				route_id = EXCLUDED.route_id,
				departure_time = EXCLUDED.departure_time,
				arrival_time = EXCLUDED.arrival_time,
				mileage = EXCLUDED.mileage,
				attendance = EXCLUDED.attendance
		`, driverLog.Driver, driverLog.BusID, driverLog.RouteID, driverLog.Date, driverLog.Period, driverLog.Departure,
		   driverLog.Arrival, driverLog.Mileage, attendanceJSON)
		
		if err != nil {
			return fmt.Errorf("failed to insert driver log for %s: %w", driverLog.Driver, err)
		}
	}

	log.Printf("✅ Migrated %d driver logs", len(logs))
	return nil
}

func migrateMaintenanceLogs() error {
	logs, _ := loadJSON[MaintenanceLog]("data/maintenance.json")
	if len(logs) == 0 {
		log.Println("📝 No maintenance logs to migrate")
		return nil
	}

	for _, maintLog := range logs {
		_, err := db.Exec(`
			INSERT INTO maintenance_logs (bus_id, date, category, notes, mileage) 
			VALUES ($1, $2, $3, $4, $5)
		`, maintLog.BusID, maintLog.Date, maintLog.Category, maintLog.Notes, maintLog.Mileage)
		
		if err != nil {
			return fmt.Errorf("failed to insert maintenance log for bus %s: %w", maintLog.BusID, err)
		}
	}

	log.Printf("✅ Migrated %d maintenance logs", len(logs))
	return nil
}

func migrateVehicles() error {
	vehicles := loadVehiclesFromJSON()
	if len(vehicles) == 0 {
		log.Println("📝 No vehicles to migrate")
		return nil
	}

	for _, vehicle := range vehicles {
		// Set default service interval if not set
		if vehicle.ServiceInterval == 0 {
			vehicle.ServiceInterval = 5000
		}
		
		_, err := db.Exec(`
			INSERT INTO vehicles (vehicle_id, model, description, year, tire_size, license, 
				oil_status, tire_status, status, maintenance_notes, serial_number, base, service_interval) 
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13) 
			ON CONFLICT (vehicle_id) DO UPDATE SET 
				model = EXCLUDED.model,
				description = EXCLUDED.description,
				year = EXCLUDED.year,
				tire_size = EXCLUDED.tire_size,
				license = EXCLUDED.license,
				oil_status = EXCLUDED.oil_status,
				tire_status = EXCLUDED.tire_status,
				status = EXCLUDED.status,
				maintenance_notes = EXCLUDED.maintenance_notes,
				serial_number = EXCLUDED.serial_number,
				base = EXCLUDED.base,
				service_interval = EXCLUDED.service_interval
		`, vehicle.VehicleID, vehicle.Model, vehicle.Description, vehicle.Year, vehicle.TireSize,
		   vehicle.License, vehicle.OilStatus, vehicle.TireStatus, vehicle.Status, vehicle.MaintenanceNotes,
		   vehicle.SerialNumber, vehicle.Base, vehicle.ServiceInterval)
		
		if err != nil {
			return fmt.Errorf("failed to insert vehicle %s: %w", vehicle.VehicleID, err)
		}
	}

	log.Printf("✅ Migrated %d vehicles", len(vehicles))
	return nil
}
