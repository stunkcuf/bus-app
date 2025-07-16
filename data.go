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
		var departureTime, arrivalTime sql.NullString
		
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
			driverLog.Departure = departureTime.String
		}
		if arrivalTime.Valid {
			driverLog.Arrival = arrivalTime.String
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

// Load driver logs for a specific driver with limit
func loadDriverLogsForDriver(driver string, limit int) ([]DriverLog, error) {
	if db == nil {
		return nil, fmt.Errorf("database connection not available")
	}
	
	query := `
		SELECT driver, bus_id, route_id, date, period, departure_time, 
			arrival_time, mileage, attendance 
		FROM driver_logs 
		WHERE driver = $1
		ORDER BY date DESC, period DESC
		LIMIT $2
	`
	
	rows, err := db.Query(query, driver, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to load driver logs: %w", err)
	}
	defer rows.Close()

	var logs []DriverLog
	for rows.Next() {
		var driverLog DriverLog
		var attendanceJSON []byte
		var date sql.NullTime
		var departureTime, arrivalTime sql.NullString
		
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
			driverLog.Departure = departureTime.String
		}
		if arrivalTime.Valid {
			driverLog.Arrival = arrivalTime.String
		}
		
		if len(attendanceJSON) > 0 {
			if err := json.Unmarshal(attendanceJSON, &driverLog.Attendance); err != nil {
				log.Printf("Error unmarshaling attendance: %v", err)
			}
		}
		
		logs = append(logs, driverLog)
	}
	
	return logs, nil
}

// Get driver logs by date range
func getDriverLogsByDateRange(driver string, startDate, endDate string) ([]DriverLog, error) {
	if db == nil {
		return nil, fmt.Errorf("database connection not available")
	}
	
	query := `
		SELECT driver, bus_id, route_id, date, period, departure_time, 
			arrival_time, mileage, attendance 
		FROM driver_logs 
		WHERE driver = $1 AND date BETWEEN $2 AND $3
		ORDER BY date DESC, period DESC
	`
	
	rows, err := db.Query(query, driver, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to load driver logs by date range: %w", err)
	}
	defer rows.Close()

	var logs []DriverLog
	for rows.Next() {
		var driverLog DriverLog
		var attendanceJSON []byte
		var date sql.NullTime
		var departureTime, arrivalTime sql.NullString
		
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
			driverLog.Departure = departureTime.String
		}
		if arrivalTime.Valid {
			driverLog.Arrival = arrivalTime.String
		}
		
		if len(attendanceJSON) > 0 {
			if err := json.Unmarshal(attendanceJSON, &driverLog.Attendance); err != nil {
				log.Printf("Error unmarshaling attendance: %v", err)
			}
		}
		
		logs = append(logs, driverLog)
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
// MAINTENANCE LOG FUNCTIONS WITH ERROR HANDLING - FIXED VERSION
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
		SELECT id, bus_id, date, category, notes, mileage, cost, created_at
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
		var createdAt sql.NullTime
		
		if err := rows.Scan(&maintenanceLog.ID, &maintenanceLog.BusID, &date,
			&maintenanceLog.Category, &maintenanceLog.Notes, 
			&maintenanceLog.Mileage, &cost, &createdAt); err != nil {
			log.Printf("Error scanning maintenance log: %v", err)
			continue
		}
		
		if date.Valid {
			maintenanceLog.Date = date.Time.Format("2006-01-02")
		}
		if cost.Valid {
			maintenanceLog.Cost = cost.Float64
		}
		if createdAt.Valid {
			maintenanceLog.CreatedAt = createdAt.Time
		}
		
		logs = append(logs, maintenanceLog)
	}
	
	if err = rows.Err(); err != nil {
		return logs, fmt.Errorf("error iterating maintenance logs: %w", err)
	}
	
	return logs, nil
}

// FIXED VERSION - Get maintenance logs for specific vehicle
func getMaintenanceLogsForVehicle(vehicleID string) ([]BusMaintenanceLog, error) {
	if db == nil {
		return nil, fmt.Errorf("database connection not available")
	}
	
	var logs []BusMaintenanceLog
	
	// Query for both bus maintenance logs and general maintenance records
	query := `
		SELECT 
			id,
			vehicle_id as bus_id,
			date::text,
			category,
			notes,
			mileage,
			COALESCE(cost, 0) as cost,
			created_at
		FROM (
			-- Bus maintenance logs
			SELECT id, bus_id as vehicle_id, date, category, notes, mileage, cost, created_at
			FROM bus_maintenance_logs
			WHERE bus_id = $1
			
			UNION ALL
			
			-- Vehicle maintenance records
			SELECT id, vehicle_id, date, category, notes, mileage, cost, created_at
			FROM maintenance_records
			WHERE vehicle_id = $1
		) combined_logs
		ORDER BY date DESC, created_at DESC
	`
	
	err := db.Select(&logs, query, vehicleID)
	if err != nil {
		log.Printf("Error getting maintenance logs for vehicle %s: %v", vehicleID, err)
		return logs, err
	}
	
	log.Printf("Retrieved %d maintenance records for vehicle %s", len(logs), vehicleID)
	
	return logs, nil
}

// Get bus-specific maintenance logs
func getBusMaintenanceLogs(busID string) ([]BusMaintenanceLog, error) {
	if db == nil {
		return nil, fmt.Errorf("database connection not available")
	}
	
	rows, err := db.Query(`
		SELECT id, bus_id, date, category, notes, mileage, cost, created_at
		FROM bus_maintenance_logs 
		WHERE bus_id = $1
		ORDER BY date DESC
	`, busID)
	if err != nil {
		return nil, fmt.Errorf("failed to load bus maintenance logs: %w", err)
	}
	defer rows.Close()

	var logs []BusMaintenanceLog
	for rows.Next() {
		var maintenanceLog BusMaintenanceLog
		var date sql.NullTime
		var cost sql.NullFloat64
		var createdAt sql.NullTime
		
		if err := rows.Scan(&maintenanceLog.ID, &maintenanceLog.BusID, &date,
			&maintenanceLog.Category, &maintenanceLog.Notes, 
			&maintenanceLog.Mileage, &cost, &createdAt); err != nil {
			log.Printf("Error scanning maintenance log: %v", err)
			continue
		}
		
		if date.Valid {
			maintenanceLog.Date = date.Time.Format("2006-01-02")
		}
		if cost.Valid {
			maintenanceLog.Cost = cost.Float64
		}
		if createdAt.Valid {
			maintenanceLog.CreatedAt = createdAt.Time
		}
		
		logs = append(logs, maintenanceLog)
	}
	
	return logs, nil
}

// Get vehicle maintenance logs from unified table
func getVehicleMaintenanceLogs(vehicleID string) ([]BusMaintenanceLog, error) {
	if db == nil {
		return nil, fmt.Errorf("database connection not available")
	}
	
	rows, err := db.Query(`
		SELECT id, vehicle_id, date, category, notes, mileage, cost, created_at
		FROM maintenance_records 
		WHERE vehicle_id = $1
		ORDER BY date DESC
	`, vehicleID)
	if err != nil {
		return nil, fmt.Errorf("failed to load vehicle maintenance logs: %w", err)
	}
	defer rows.Close()

	var logs []BusMaintenanceLog
	for rows.Next() {
		var maintenanceLog BusMaintenanceLog
		var date sql.NullTime
		var cost sql.NullFloat64
		var createdAt sql.NullTime
		
		if err := rows.Scan(&maintenanceLog.ID, &maintenanceLog.VehicleID, &date,
			&maintenanceLog.Category, &maintenanceLog.Notes, 
			&maintenanceLog.Mileage, &cost, &createdAt); err != nil {
			log.Printf("Error scanning vehicle maintenance log: %v", err)
			continue
		}
		
		// Use VehicleID as BusID for compatibility
		maintenanceLog.BusID = maintenanceLog.VehicleID
		
		if date.Valid {
			maintenanceLog.Date = date.Time.Format("2006-01-02")
		}
		if cost.Valid {
			maintenanceLog.Cost = cost.Float64
		}
		if createdAt.Valid {
			maintenanceLog.CreatedAt = createdAt.Time
		}
		
		logs = append(logs, maintenanceLog)
	}
	
	return logs, nil
}

// Get legacy service records
func getLegacyServiceRecords(vehicleID string) ([]BusMaintenanceLog, error) {
	if db == nil {
		return nil, fmt.Errorf("database connection not available")
	}
	
	rows, err := db.Query(`
		SELECT id, COALESCE(vehicle_id, vehicle_number, unnamed_1), 
		       maintenance_date, service_type, notes, created_at
		FROM service_records 
		WHERE vehicle_id = $1 OR vehicle_number = $1 OR unnamed_1 = $1
		ORDER BY maintenance_date DESC
	`, vehicleID)
	if err != nil {
		return nil, fmt.Errorf("failed to load legacy service records: %w", err)
	}
	defer rows.Close()

	var logs []BusMaintenanceLog
	for rows.Next() {
		var maintenanceLog BusMaintenanceLog
		var vehicleRef sql.NullString
		var date sql.NullTime
		var serviceType sql.NullString
		var notes sql.NullString
		var createdAt sql.NullTime
		
		if err := rows.Scan(&maintenanceLog.ID, &vehicleRef, &date,
			&serviceType, &notes, &createdAt); err != nil {
			log.Printf("Error scanning legacy service record: %v", err)
			continue
		}
		
		if vehicleRef.Valid {
			maintenanceLog.BusID = vehicleRef.String
			maintenanceLog.VehicleID = vehicleRef.String
		}
		
		if date.Valid {
			maintenanceLog.Date = date.Time.Format("2006-01-02")
		}
		
		if serviceType.Valid {
			maintenanceLog.Category = serviceType.String
		} else {
			maintenanceLog.Category = "service"
		}
		
		if notes.Valid {
			maintenanceLog.Notes = notes.String
		}
		
		if createdAt.Valid {
			maintenanceLog.CreatedAt = createdAt.Time
		}
		
		logs = append(logs, maintenanceLog)
	}
	
	return logs, nil
}

func saveMaintenanceLog(log BusMaintenanceLog) error {
	if db == nil {
		return fmt.Errorf("database connection not available")
	}
	
	_, err := db.Exec(`
		INSERT INTO bus_maintenance_logs (bus_id, date, category, notes, mileage, cost) 
		VALUES ($1, $2, $3, $4, $5, $6)
	`, log.BusID, log.Date, log.Category, log.Notes, log.Mileage, log.Cost)
	
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

// Get activities for a specific driver
func getDriverActivities(driver string, limit int) ([]Activity, error) {
	if db == nil {
		return nil, fmt.Errorf("database connection not available")
	}
	
	query := `
		SELECT date, driver, trip_name, attendance, miles, notes 
		FROM activities 
		WHERE driver = $1
		ORDER BY date DESC
		LIMIT $2
	`
	
	rows, err := db.Query(query, driver, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to load driver activities: %w", err)
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
	
	return activities, nil
}

// Get activities by date range
func getActivitiesByDateRange(startDate, endDate string) ([]Activity, error) {
	if db == nil {
		return nil, fmt.Errorf("database connection not available")
	}
	
	rows, err := db.Query(`
		SELECT date, driver, trip_name, attendance, miles, notes 
		FROM activities 
		WHERE date BETWEEN $1 AND $2
		ORDER BY date DESC
	`, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to load activities by date range: %w", err)
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

// =============================================================================
// MILEAGE REPORT FUNCTIONS
// =============================================================================

// Get monthly mileage reports for all vehicles
func getMonthlyMileageReports(month string, year int) ([]MileageReport, error) {
	if db == nil {
		return nil, fmt.Errorf("database connection not available")
	}
	
	var reports []MileageReport
	
	// Query from agency_vehicles
	query1 := `
		SELECT report_month, report_year, vehicle_year, make_model, license_plate,
		       vehicle_id, location, beginning_miles, ending_miles, total_miles, status
		FROM agency_vehicles
		WHERE report_month = $1 AND report_year = $2
		ORDER BY vehicle_id
	`
	
	rows, err := db.Query(query1, month, year)
	if err != nil {
		log.Printf("Error querying agency vehicles: %v", err)
	} else {
		defer rows.Close()
		for rows.Next() {
			var report MileageReport
			err := rows.Scan(&report.ReportMonth, &report.ReportYear, &report.VehicleYear,
				&report.MakeModel, &report.LicensePlate, &report.VehicleID,
				&report.Location, &report.BeginningMiles, &report.EndingMiles,
				&report.TotalMiles, &report.Status)
			if err != nil {
				log.Printf("Error scanning agency vehicle: %v", err)
				continue
			}
			reports = append(reports, report)
		}
	}
	
	// Query from school_buses
	query2 := `
		SELECT report_month, report_year, bus_year, bus_make, license_plate,
		       bus_id, location, beginning_miles, ending_miles, total_miles, status
		FROM school_buses
		WHERE report_month = $1 AND report_year = $2
		ORDER BY bus_id
	`
	
	rows, err = db.Query(query2, month, year)
	if err != nil {
		log.Printf("Error querying school buses: %v", err)
	} else {
		defer rows.Close()
		for rows.Next() {
			var report MileageReport
			var busYear sql.NullInt64
			var busMake sql.NullString
			
			err := rows.Scan(&report.ReportMonth, &report.ReportYear, &busYear,
				&busMake, &report.LicensePlate, &report.VehicleID,
				&report.Location, &report.BeginningMiles, &report.EndingMiles,
				&report.TotalMiles, &report.Status)
			if err != nil {
				log.Printf("Error scanning school bus: %v", err)
				continue
			}
			
			if busYear.Valid {
				report.VehicleYear = int(busYear.Int64)
			}
			if busMake.Valid {
				report.MakeModel = busMake.String
			}
			
			reports = append(reports, report)
		}
	}
	
	return reports, nil
}

// Get available report periods
func getAvailableReportPeriods() ([]string, error) {
	if db == nil {
		return nil, fmt.Errorf("database connection not available")
	}
	
	query := `
		SELECT DISTINCT report_month || ' ' || report_year::text as period
		FROM (
			SELECT report_month, report_year FROM agency_vehicles
			UNION
			SELECT report_month, report_year FROM school_buses
		) combined
		ORDER BY period DESC
	`
	
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to get available report periods: %w", err)
	}
	defer rows.Close()
	
	var periods []string
	for rows.Next() {
		var period string
		if err := rows.Scan(&period); err != nil {
			log.Printf("Error scanning period: %v", err)
			continue
		}
		periods = append(periods, period)
	}
	
	return periods, nil
}
