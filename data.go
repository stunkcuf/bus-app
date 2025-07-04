// data.go - PostgreSQL-only data loading and saving functions with proper error handling
package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
)

// =============================================================================
// USER FUNCTIONS WITH ERROR HANDLING
// =============================================================================

// DEPRECATED: Use loadUsersFromDB instead
func loadUsers() []User {
	users, err := loadUsersFromDB()
	if err != nil {
		log.Printf("Error loading users: %v", err)
		return []User{}
	}
	return users
}

func loadUsersFromDB() ([]User, error) {
	if db == nil {
		return nil, fmt.Errorf("database connection not available")
	}
	
	// Try to load with status and registration_date first
	rows, err := db.Query(`
		SELECT username, password, role, 
		       COALESCE(status, 'active') as status,
		       COALESCE(registration_date::text, created_at::text, '') as registration_date
		FROM users 
		ORDER BY username
	`)
	
	if err != nil {
		// Fallback to original query if new columns don't exist
		log.Printf("Loading users without new columns, trying basic query: %v", err)
		rows, err = db.Query("SELECT username, password, role FROM users ORDER BY username")
		if err != nil {
			return nil, fmt.Errorf("failed to load users from DB: %w", err)
		}
		defer rows.Close()

		var users []User
		for rows.Next() {
			var user User
			if err := rows.Scan(&user.Username, &user.Password, &user.Role); err != nil {
				log.Printf("Error scanning user: %v", err)
				continue
			}
			// Set defaults for missing fields
			user.Status = "active"
			user.RegistrationDate = ""
			users = append(users, user)
		}
		return users, nil
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		var status, regDate sql.NullString
		
		if err := rows.Scan(&user.Username, &user.Password, &user.Role, 
			&status, &regDate); err != nil {
			log.Printf("Error scanning user with full fields: %v", err)
			// Try simpler scan
			if err := rows.Scan(&user.Username, &user.Password, &user.Role); err != nil {
				log.Printf("Error scanning basic user fields: %v", err)
				continue
			}
			user.Status = "active"
			user.RegistrationDate = ""
		} else {
			// Use the scanned values
			if status.Valid {
				user.Status = status.String
			} else {
				user.Status = "active"
			}
			
			if regDate.Valid {
				user.RegistrationDate = regDate.String
			} else {
				user.RegistrationDate = ""
			}
		}
		
		users = append(users, user)
	}
	
	if err = rows.Err(); err != nil {
		return users, fmt.Errorf("error iterating users: %w", err)
	}
	
	return users, nil
}

func updateUser(user User) error {
	if db == nil {
		return fmt.Errorf("database connection not available")
	}
	
	// Ensure status is set
	if user.Status == "" {
		user.Status = "active"
	}
	
	// First, check if updated_at column exists
	var columnExists bool
	err := db.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM information_schema.columns 
			WHERE table_name = 'users' 
			AND column_name = 'updated_at'
		)
	`).Scan(&columnExists)
	
	if err != nil {
		// If we can't check, just try without updated_at
		columnExists = false
	}
	
	// Update based on whether updated_at exists
	if columnExists {
		_, err = db.Exec(`
			UPDATE users 
			SET password = $2, 
			    role = $3, 
			    status = $4,
			    updated_at = CURRENT_TIMESTAMP
			WHERE username = $1
		`, user.Username, user.Password, user.Role, user.Status)
	} else {
		// Update without updated_at column
		_, err = db.Exec(`
			UPDATE users 
			SET password = $2, 
			    role = $3, 
			    status = $4
			WHERE username = $1
		`, user.Username, user.Password, user.Role, user.Status)
	}
	
	if err != nil {
		return fmt.Errorf("failed to update user %s: %w", user.Username, err)
	}
	
	return nil
}

func saveUser(user User) error {
	if db == nil {
		return fmt.Errorf("database connection not available")
	}
	
	// Set default status if not provided
	if user.Status == "" {
		user.Status = "active"
	}
	
	_, err := db.Exec(`
		INSERT INTO users (username, password, role, status) 
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (username) 
		DO UPDATE SET 
			password = $2, 
			role = $3, 
			status = $4
	`, user.Username, user.Password, user.Role, user.Status)
	
	return err
}

func saveUsers(users []User) error {
	if db == nil {
		return fmt.Errorf("database connection not available")
	}
	
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	
	success := false
	defer func() {
		if !success {
			tx.Rollback()
		}
	}()
	
	for _, user := range users {
		// Set default status if not provided
		if user.Status == "" {
			user.Status = "active"
		}
		
		_, err := tx.Exec(`
			INSERT INTO users (username, password, role, status) 
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (username) 
			DO UPDATE SET 
				password = $2, 
				role = $3, 
				status = $4
		`, user.Username, user.Password, user.Role, user.Status)
		
		if err != nil {
			return fmt.Errorf("failed to save user %s: %w", user.Username, err)
		}
	}
	
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	
	success = true
	return nil
}

func deleteUser(username string) error {
	if db == nil {
		return fmt.Errorf("database connection not available")
	}
	
	_, err := db.Exec("DELETE FROM users WHERE username = $1", username)
	return err
}

// =============================================================================
// BUS FUNCTIONS WITH ERROR HANDLING
// =============================================================================

// DEPRECATED: Use loadBusesFromDB instead
func loadBuses() []*Bus {
	buses, err := loadBusesFromDB()
	if err != nil {
		log.Printf("Error loading buses: %v", err)
		return []*Bus{}
	}
	return buses
}

func loadBusesFromDB() ([]*Bus, error) {
	if db == nil {
		return nil, fmt.Errorf("database connection not available")
	}
	
	rows, err := db.Query(`
		SELECT bus_id, status, model, capacity, oil_status, tire_status, maintenance_notes 
		FROM buses ORDER BY bus_id
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to load buses from DB: %w", err)
	}
	defer rows.Close()

	var buses []*Bus
	for rows.Next() {
		bus := &Bus{}
		if err := rows.Scan(&bus.BusID, &bus.Status, &bus.Model, &bus.Capacity, 
			&bus.OilStatus, &bus.TireStatus, &bus.MaintenanceNotes); err != nil {
			log.Printf("Error scanning bus: %v", err)
			continue
		}
		buses = append(buses, bus)
	}
	
	if err = rows.Err(); err != nil {
		return buses, fmt.Errorf("error iterating buses: %w", err)
	}
	
	return buses, nil
}

func saveBus(bus *Bus) error {
	if db == nil {
		return fmt.Errorf("database connection not available")
	}
	
	_, err := db.Exec(`
		INSERT INTO buses (bus_id, status, model, capacity, oil_status, tire_status, maintenance_notes) 
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (bus_id) 
		DO UPDATE SET 
			status = $2, model = $3, capacity = $4, 
			oil_status = $5, tire_status = $6, maintenance_notes = $7,
			updated_at = CURRENT_TIMESTAMP
	`, bus.BusID, bus.Status, bus.Model, bus.Capacity,
	   bus.OilStatus, bus.TireStatus, bus.MaintenanceNotes)
	
	return err
}

func saveBuses(buses []*Bus) error {
	if db == nil {
		return fmt.Errorf("database connection not available")
	}
	
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	
	success := false
	defer func() {
		if !success {
			tx.Rollback()
		}
	}()

	for _, bus := range buses {
		_, err := tx.Exec(`
			UPDATE buses 
			SET status = $2, model = $3, capacity = $4, 
				oil_status = $5, tire_status = $6, maintenance_notes = $7,
				updated_at = CURRENT_TIMESTAMP
			WHERE bus_id = $1
		`, bus.BusID, bus.Status, bus.Model, bus.Capacity,
		   bus.OilStatus, bus.TireStatus, bus.MaintenanceNotes)
		
		if err != nil {
			return fmt.Errorf("failed to update bus %s: %w", bus.BusID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	
	success = true
	return nil
}

func deleteBus(busID string) error {
	if db == nil {
		return fmt.Errorf("database connection not available")
	}
	
	_, err := db.Exec("DELETE FROM buses WHERE bus_id = $1", busID)
	return err
}

// =============================================================================
// ROUTE FUNCTIONS WITH ERROR HANDLING
// =============================================================================

// DEPRECATED: Use loadRoutesFromDB instead
func loadRoutes() ([]Route, error) {
	return loadRoutesFromDB()
}

func loadRoutesFromDB() ([]Route, error) {
	if db == nil {
		return nil, fmt.Errorf("database connection not available")
	}
	
	rows, err := db.Query(`
		SELECT route_id, route_name, description, positions 
		FROM routes ORDER BY route_id
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to load routes from DB: %w", err)
	}
	defer rows.Close()

	var routes []Route
	for rows.Next() {
		var route Route
		var positionsJSON []byte
		if err := rows.Scan(&route.RouteID, &route.RouteName, &route.Description, &positionsJSON); err != nil {
			log.Printf("Error scanning route: %v", err)
			continue
		}
		
		if len(positionsJSON) > 0 {
			if err := json.Unmarshal(positionsJSON, &route.Positions); err != nil {
				log.Printf("Error unmarshaling positions for route %s: %v", route.RouteID, err)
			}
		}
		
		routes = append(routes, route)
	}
	
	if err = rows.Err(); err != nil {
		return routes, fmt.Errorf("error iterating routes: %w", err)
	}
	
	return routes, nil
}

func saveRoute(route Route) error {
	if db == nil {
		return fmt.Errorf("database connection not available")
	}
	
	positionsJSON, err := json.Marshal(route.Positions)
	if err != nil {
		return fmt.Errorf("failed to marshal positions: %w", err)
	}
	
	_, err = db.Exec(`
		INSERT INTO routes (route_id, route_name, description, positions) 
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (route_id) 
		DO UPDATE SET 
			route_name = $2, description = $3, positions = $4,
			updated_at = CURRENT_TIMESTAMP
	`, route.RouteID, route.RouteName, route.Description, positionsJSON)
	
	return err
}

func saveRoutes(routes []Route) error {
	if db == nil {
		return fmt.Errorf("database connection not available")
	}
	
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	
	success := false
	defer func() {
		if !success {
			tx.Rollback()
		}
	}()
	
	for _, route := range routes {
		positionsJSON, err := json.Marshal(route.Positions)
		if err != nil {
			return fmt.Errorf("failed to marshal positions: %w", err)
		}
		
		_, err = tx.Exec(`
			UPDATE routes 
			SET route_name = $2, description = $3, positions = $4,
				updated_at = CURRENT_TIMESTAMP
			WHERE route_id = $1
		`, route.RouteID, route.RouteName, route.Description, positionsJSON)
		
		if err != nil {
			return fmt.Errorf("failed to update route %s: %w", route.RouteID, err)
		}
	}
	
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	
	success = true
	return nil
}

func deleteRoute(routeID string) error {
	if db == nil {
		return fmt.Errorf("database connection not available")
	}
	
	_, err := db.Exec("DELETE FROM routes WHERE route_id = $1", routeID)
	return err
}

// =============================================================================
// STUDENT FUNCTIONS WITH ERROR HANDLING
// =============================================================================

// DEPRECATED: Use loadStudentsFromDB instead
func loadStudents() []Student {
	students, err := loadStudentsFromDB()
	if err != nil {
		log.Printf("Error loading students: %v", err)
		return []Student{}
	}
	return students
}

func loadStudentsFromDB() ([]Student, error) {
	if db == nil {
		return nil, fmt.Errorf("database connection not available")
	}
	
	rows, err := db.Query(`
		SELECT student_id, name, locations, phone_number, alt_phone_number, 
			guardian, pickup_time, dropoff_time, position_number, route_id, driver, active
		FROM students ORDER BY student_id
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to load students from DB: %w", err)
	}
	defer rows.Close()

	var students []Student
	for rows.Next() {
		var student Student
		var locationsJSON []byte
		var pickupTime, dropoffTime sql.NullTime
		
		if err := rows.Scan(&student.StudentID, &student.Name, &locationsJSON,
			&student.PhoneNumber, &student.AltPhoneNumber, &student.Guardian,
			&pickupTime, &dropoffTime, &student.PositionNumber,
			&student.RouteID, &student.Driver, &student.Active); err != nil {
			log.Printf("Error scanning student: %v", err)
			continue
		}
		
		if pickupTime.Valid {
			student.PickupTime = pickupTime.Time.Format("15:04")
		}
		if dropoffTime.Valid {
			student.DropoffTime = dropoffTime.Time.Format("15:04")
		}
		
		if len(locationsJSON) > 0 {
			if err := json.Unmarshal(locationsJSON, &student.Locations); err != nil {
				log.Printf("Error unmarshaling locations for student %s: %v", student.StudentID, err)
			}
		}
		
		students = append(students, student)
	}
	
	if err = rows.Err(); err != nil {
		return students, fmt.Errorf("error iterating students: %w", err)
	}
	
	return students, nil
}

func saveStudent(student Student) error {
	if db == nil {
		return fmt.Errorf("database connection not available")
	}
	
	locationsJSON, err := json.Marshal(student.Locations)
	if err != nil {
		return fmt.Errorf("failed to marshal locations: %w", err)
	}
	
	log.Printf("DEBUG: Saving student %s with %d locations: %s", 
		student.StudentID, len(student.Locations), string(locationsJSON))
	
	// Handle NULL times
	var pickupTime, dropoffTime interface{}
	if student.PickupTime != "" {
		pickupTime = student.PickupTime
	} else {
		pickupTime = nil
	}
	if student.DropoffTime != "" {
		dropoffTime = student.DropoffTime
	} else {
		dropoffTime = nil
	}
	
	_, err = db.Exec(`
		INSERT INTO students (student_id, name, locations, phone_number, alt_phone_number,
			guardian, pickup_time, dropoff_time, position_number, route_id, driver, active) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (student_id) 
		DO UPDATE SET 
			name = $2, locations = $3, phone_number = $4, alt_phone_number = $5,
			guardian = $6, pickup_time = $7, dropoff_time = $8, position_number = $9,
			route_id = $10, driver = $11, active = $12,
			updated_at = CURRENT_TIMESTAMP
	`, student.StudentID, student.Name, locationsJSON, student.PhoneNumber,
		student.AltPhoneNumber, student.Guardian, pickupTime, dropoffTime,
		student.PositionNumber, student.RouteID, student.Driver, student.Active)
	
	if err != nil {
		log.Printf("ERROR: Failed to save student %s: %v", student.StudentID, err)
		return fmt.Errorf("failed to save student %s: %w", student.StudentID, err)
	}
	
	return nil
}

func deleteStudent(studentID string) error {
	if db == nil {
		return fmt.Errorf("database connection not available")
	}
	
	_, err := db.Exec("DELETE FROM students WHERE student_id = $1", studentID)
	return err
}

// =============================================================================
// ROUTE ASSIGNMENT FUNCTIONS WITH ERROR HANDLING
// =============================================================================

func loadRouteAssignments() ([]RouteAssignment, error) {
	if db == nil {
		return nil, fmt.Errorf("database connection not available")
	}
	
	rows, err := db.Query(`
		SELECT driver, bus_id, route_id, route_name, assigned_date 
		FROM route_assignments ORDER BY driver
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to load route assignments from DB: %w", err)
	}
	defer rows.Close()

	var assignments []RouteAssignment
	for rows.Next() {
		var assignment RouteAssignment
		var assignedDate sql.NullTime
		var routeName sql.NullString
		
		if err := rows.Scan(&assignment.Driver, &assignment.BusID, &assignment.RouteID,
			&routeName, &assignedDate); err != nil {
			log.Printf("Error scanning route assignment: %v", err)
			continue
		}
		
		// Handle nullable route name
		if routeName.Valid {
			assignment.RouteName = routeName.String
		} else {
			assignment.RouteName = ""
			log.Printf("Warning: Route name is NULL for assignment: driver=%s, route_id=%s", 
				assignment.Driver, assignment.RouteID)
		}
		
		if assignedDate.Valid {
			assignment.AssignedDate = assignedDate.Time.Format("2006-01-02")
		}
		
		log.Printf("Loaded assignment: driver=%s, bus=%s, route=%s, route_name=%s", 
			assignment.Driver, assignment.BusID, assignment.RouteID, assignment.RouteName)
		
		assignments = append(assignments, assignment)
	}
	
	if err = rows.Err(); err != nil {
		return assignments, fmt.Errorf("error iterating route assignments: %w", err)
	}
	
	log.Printf("Total route assignments loaded: %d", len(assignments))
	return assignments, nil
}

func deleteRouteAssignment(driver string) error {
	if db == nil {
		return fmt.Errorf("database connection not available")
	}
	
	_, err := db.Exec("DELETE FROM route_assignments WHERE driver = $1", driver)
	return err
}

// =============================================================================
// DRIVER LOG FUNCTIONS WITH ERROR HANDLING
// =============================================================================

// DEPRECATED: Use loadDriverLogsFromDB instead
func loadDriverLogs() ([]DriverLog, error) {
	return loadDriverLogsFromDB()
}

func loadDriverLogsFromDB() ([]DriverLog, error) {
	if db == nil {
		return nil, fmt.Errorf("database connection not available")
	}
	
	rows, err := db.Query(`
		SELECT driver, bus_id, route_id, date, period, departure_time, 
			arrival_time, mileage, attendance 
		FROM driver_logs ORDER BY date DESC, driver
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to load driver logs from DB: %w", err)
	}
	defer rows.Close()

	var logs []DriverLog
	for rows.Next() {
		var driverLog DriverLog
		var attendanceJSON []byte
		var date sql.NullTime
		var departureTime, arrivalTime sql.NullTime
		
		if err := rows.Scan(&driverLog.Driver, &driverLog.BusID, &driverLog.RouteID,
			&date, &driverLog.Period, &departureTime,
			&arrivalTime, &driverLog.Mileage, &attendanceJSON); err != nil {
			log.Printf("Error scanning driver log: %v", err)
			continue
		}
		
		if date.Valid {
			driverLog.Date = date.Time.Format("2006-01-02")
		}
		if departureTime.Valid {
			driverLog.Departure = departureTime.Time.Format("15:04")
		}
		if arrivalTime.Valid {
			driverLog.Arrival = arrivalTime.Time.Format("15:04")
		}
		
		if len(attendanceJSON) > 0 {
			if err := json.Unmarshal(attendanceJSON, &driverLog.Attendance); err != nil {
				log.Printf("Error unmarshaling attendance: %v", err)
			}
		}
		
		logs = append(logs, driverLog)
	}
	
	if err = rows.Err(); err != nil {
		return logs, fmt.Errorf("error iterating driver logs: %w", err)
	}
	
	return logs, nil
}

func saveDriverLog(driverLog DriverLog) error {
	if db == nil {
		return fmt.Errorf("database connection not available")
	}
	
	attendanceJSON, err := json.Marshal(driverLog.Attendance)
	if err != nil {
		return fmt.Errorf("failed to marshal attendance: %w", err)
	}
	
	_, err = db.Exec(`
		INSERT INTO driver_logs (driver, bus_id, route_id, date, period, 
			departure_time, arrival_time, mileage, attendance) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (driver, date, period) 
		DO UPDATE SET 
			bus_id = $2, route_id = $3, departure_time = $6, 
			arrival_time = $7, mileage = $8, attendance = $9
	`, driverLog.Driver, driverLog.BusID, driverLog.RouteID, driverLog.Date,
		driverLog.Period, driverLog.Departure, driverLog.Arrival, 
		driverLog.Mileage, attendanceJSON)
	
	return err
}

func saveDriverLogs(logs []DriverLog) error {
	if db == nil {
		return fmt.Errorf("database connection not available")
	}
	
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	
	success := false
	defer func() {
		if !success {
			tx.Rollback()
		}
	}()
	
	for _, log := range logs {
		if err := saveDriverLog(log); err != nil {
			return fmt.Errorf("failed to save log for driver %s: %w", log.Driver, err)
		}
	}
	
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	
	success = true
	return nil
}

// =============================================================================
// MAINTENANCE LOG FUNCTIONS WITH ERROR HANDLING
// =============================================================================

// DEPRECATED: Use loadMaintenanceLogsFromDB instead
func loadMaintenanceLogs() []BusMaintenanceLog {
	logs, err := loadMaintenanceLogsFromDB()
	if err != nil {
		log.Printf("Error loading maintenance logs: %v", err)
		return []BusMaintenanceLog{}
	}
	return logs
}

func loadMaintenanceLogsFromDB() ([]BusMaintenanceLog, error) {
	if db == nil {
		return nil, fmt.Errorf("database connection not available")
	}
	
	rows, err := db.Query(`
		SELECT bus_id, date, category, notes, mileage, cost 
		FROM bus_maintenance_logs ORDER BY date DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to load maintenance logs from DB: %w", err)
	}
	defer rows.Close()

	var logs []BusMaintenanceLog
	for rows.Next() {
		var maintenanceLog BusMaintenanceLog
		var date sql.NullTime
		var cost sql.NullFloat64
		
		if err := rows.Scan(&maintenanceLog.BusID, &date,
			&maintenanceLog.Category, &maintenanceLog.Notes, 
			&maintenanceLog.Mileage, &cost); err != nil {
			log.Printf("Error scanning maintenance log: %v", err)
			continue
		}
		
		if date.Valid {
			maintenanceLog.Date = date.Time.Format("2006-01-02")
		}
		// Note: BusMaintenanceLog doesn't have a Cost field in your model
		
		logs = append(logs, maintenanceLog)
	}
	
	if err = rows.Err(); err != nil {
		return logs, fmt.Errorf("error iterating maintenance logs: %w", err)
	}
	
	return logs, nil
}

func saveMaintenanceLog(log BusMaintenanceLog) error {
	if db == nil {
		return fmt.Errorf("database connection not available")
	}
	
	_, err := db.Exec(`
		INSERT INTO bus_maintenance_logs (bus_id, date, category, notes, mileage) 
		VALUES ($1, $2, $3, $4, $5)
	`, log.BusID, log.Date, log.Category, log.Notes, log.Mileage)
	
	return err
}

func saveMaintenanceLogs(logs []BusMaintenanceLog) error {
	if db == nil {
		return fmt.Errorf("database connection not available")
	}
	
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	
	success := false
	defer func() {
		if !success {
			tx.Rollback()
		}
	}()
	
	for _, log := range logs {
		if err := saveMaintenanceLog(log); err != nil {
			return fmt.Errorf("failed to save maintenance log: %w", err)
		}
	}
	
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	
	success = true
	return nil
}

// =============================================================================
// VEHICLE FUNCTIONS WITH ERROR HANDLING
// =============================================================================

// DEPRECATED: Use loadVehiclesFromDB instead
func loadVehicles() []Vehicle {
	vehicles, err := loadVehiclesFromDB()
	if err != nil {
		log.Printf("Error loading vehicles: %v", err)
		return []Vehicle{}
	}
	return vehicles
}

func loadVehiclesFromDB() ([]Vehicle, error) {
	if db == nil {
		return nil, fmt.Errorf("database connection not available")
	}
	
	rows, err := db.Query(`
		SELECT vehicle_id, model, description, year, tire_size, license,
			oil_status, tire_status, status, maintenance_notes, serial_number, base, service_interval
		FROM vehicles ORDER BY vehicle_id
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to load vehicles from DB: %w", err)
	}
	defer rows.Close()

	var vehicles []Vehicle
	for rows.Next() {
		var vehicle Vehicle
		if err := rows.Scan(&vehicle.VehicleID, &vehicle.Model, &vehicle.Description,
			&vehicle.Year, &vehicle.TireSize, &vehicle.License, &vehicle.OilStatus,
			&vehicle.TireStatus, &vehicle.Status, &vehicle.MaintenanceNotes,
			&vehicle.SerialNumber, &vehicle.Base, &vehicle.ServiceInterval); err != nil {
			log.Printf("Error scanning vehicle: %v", err)
			continue
		}
		vehicles = append(vehicles, vehicle)
	}
	
	if err = rows.Err(); err != nil {
		return vehicles, fmt.Errorf("error iterating vehicles: %w", err)
	}
	
	return vehicles, nil
}

func saveVehicle(vehicle Vehicle) error {
	if db == nil {
		return fmt.Errorf("database connection not available")
	}
	
	_, err := db.Exec(`
		INSERT INTO vehicles (vehicle_id, model, description, year, tire_size, license,
			oil_status, tire_status, status, maintenance_notes, serial_number, base, service_interval) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		ON CONFLICT (vehicle_id) 
		DO UPDATE SET 
			model = $2, description = $3, year = $4, tire_size = $5, 
			license = $6, oil_status = $7, tire_status = $8, status = $9, 
			maintenance_notes = $10, serial_number = $11, base = $12, 
			service_interval = $13, updated_at = CURRENT_TIMESTAMP
	`, vehicle.VehicleID, vehicle.Model, vehicle.Description, vehicle.Year,
		vehicle.TireSize, vehicle.License, vehicle.OilStatus, vehicle.TireStatus,
		vehicle.Status, vehicle.MaintenanceNotes, vehicle.SerialNumber, 
		vehicle.Base, vehicle.ServiceInterval)
	
	return err
}

func saveVehicles(vehicles []Vehicle) error {
	if db == nil {
		return fmt.Errorf("database connection not available")
	}
	
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	
	success := false
	defer func() {
		if !success {
			tx.Rollback()
		}
	}()

	for _, vehicle := range vehicles {
		_, err := tx.Exec(`
			UPDATE vehicles 
			SET model = $2, description = $3, year = $4, tire_size = $5, 
				license = $6, oil_status = $7, tire_status = $8, status = $9, 
				maintenance_notes = $10, serial_number = $11, base = $12, 
				service_interval = $13
			WHERE vehicle_id = $1
		`, vehicle.VehicleID, vehicle.Model, vehicle.Description, vehicle.Year,
		   vehicle.TireSize, vehicle.License, vehicle.OilStatus, vehicle.TireStatus,
		   vehicle.Status, vehicle.MaintenanceNotes, vehicle.SerialNumber, 
		   vehicle.Base, vehicle.ServiceInterval)
		
		if err != nil {
			return fmt.Errorf("failed to update vehicle %s: %w", vehicle.VehicleID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	
	success = true
	return nil
}

// =============================================================================
// ACTIVITY FUNCTIONS WITH ERROR HANDLING
// =============================================================================

func loadActivities() ([]Activity, error) {
	if db == nil {
		return nil, fmt.Errorf("database connection not available")
	}
	
	rows, err := db.Query(`
		SELECT date, driver, trip_name, attendance, miles, notes 
		FROM activities ORDER BY date DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to load activities from DB: %w", err)
	}
	defer rows.Close()

	var activities []Activity
	for rows.Next() {
		var activity Activity
		var date sql.NullTime
		
		if err := rows.Scan(&date, &activity.Driver, &activity.TripName,
			&activity.Attendance, &activity.Miles, &activity.Notes); err != nil {
			log.Printf("Error scanning activity: %v", err)
			continue
		}
		
		if date.Valid {
			activity.Date = date.Time.Format("2006-01-02")
		}
		
		activities = append(activities, activity)
	}
	
	if err = rows.Err(); err != nil {
		return activities, fmt.Errorf("error iterating activities: %w", err)
	}
	
	return activities, nil
}

func saveActivity(activity Activity) error {
	if db == nil {
		return fmt.Errorf("database connection not available")
	}
	
	_, err := db.Exec(`
		INSERT INTO activities (date, driver, trip_name, attendance, miles, notes) 
		VALUES ($1, $2, $3, $4, $5, $6)
	`, activity.Date, activity.Driver, activity.TripName, 
		activity.Attendance, activity.Miles, activity.Notes)
	
	return err
}
