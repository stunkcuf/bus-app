// data.go - Data loading and saving functions
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
)

// =============================================================================
// USER FUNCTIONS
// =============================================================================

func loadUsers() []User {
	if db != nil {
		return loadUsersFromDB()
	}
	return loadUsersFromJSON()
}

func saveUsers(users []User) error {
	if db != nil {
		return saveUsersToDB(users)
	}
	return saveUsersToJSON(users)
}

func loadUsersFromDB() []User {
	rows, err := db.Query("SELECT username, password, role FROM users")
	if err != nil {
		log.Printf("Error loading users from DB: %v", err)
		return []User{}
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		if err := rows.Scan(&user.Username, &user.Password, &user.Role); err != nil {
			log.Printf("Error scanning user: %v", err)
			continue
		}
		users = append(users, user)
	}
	return users
}

func saveUsersToDB(users []User) error {
	// This would typically be handled by individual user creation
	// For bulk operations, we'd need transaction handling
	return nil
}

func loadUsersFromJSON() []User {
	users, _ := loadJSON[User]("data/users.json")
	return users
}

func saveUsersToJSON(users []User) error {
	return saveJSONFile("data/users.json", users)
}

// =============================================================================
// BUS FUNCTIONS
// =============================================================================

func loadBuses() []*Bus {
	if db != nil {
		return loadBusesFromDB()
	}
	return loadBusesFromJSON()
}

func saveBuses(buses []*Bus) error {
	if db != nil {
		return saveBusesToDB(buses)
	}
	return saveBusesToJSON(buses)
}

func loadBusesFromDB() []*Bus {
	rows, err := db.Query(`
		SELECT bus_id, status, model, capacity, oil_status, tire_status, maintenance_notes 
		FROM buses ORDER BY bus_id
	`)
	if err != nil {
		log.Printf("Error loading buses from DB: %v", err)
		return []*Bus{}
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
	return buses
}

func saveBusesToDB(buses []*Bus) error {
	// Start a transaction for bulk update
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// For each bus, update its fields
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

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func loadBusesFromJSON() []*Bus {
	buses, _ := loadJSON[*Bus]("data/buses.json")
	return buses
}

func saveBusesToJSON(buses []*Bus) error {
	return saveJSONFile("data/buses.json", buses)
}

// =============================================================================
// ROUTE FUNCTIONS
// =============================================================================

func loadRoutes() ([]Route, error) {
	if db != nil {
		return loadRoutesFromDB()
	}
	return loadRoutesFromJSON()
}

func saveRoutes(routes []Route) error {
	if db != nil {
		return saveRoutesToDB(routes)
	}
	return saveRoutesToJSON(routes)
}

func loadRoutesFromDB() ([]Route, error) {
	rows, err := db.Query(`
		SELECT route_id, route_name, description, positions 
		FROM routes ORDER BY route_id
	`)
	if err != nil {
		log.Printf("Error loading routes from DB: %v", err)
		return []Route{}, err
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
		
		// Parse positions JSON
		if len(positionsJSON) > 0 {
			if err := json.Unmarshal(positionsJSON, &route.Positions); err != nil {
				log.Printf("Error unmarshaling positions for route %s: %v", route.RouteID, err)
			}
		}
		
		routes = append(routes, route)
	}
	return routes, nil
}

func saveRoutesToDB(routes []Route) error {
	// Individual route operations are handled in handlers
	return nil
}

func loadRoutesFromJSON() ([]Route, error) {
	return loadJSON[Route]("data/routes.json")
}

func saveRoutesToJSON(routes []Route) error {
	return saveJSONFile("data/routes.json", routes)
}

// =============================================================================
// STUDENT FUNCTIONS
// =============================================================================

func loadStudents() []Student {
	if db != nil {
		return loadStudentsFromDB()
	}
	return loadStudentsFromJSON()
}

func saveStudents(students []Student) error {
	if db != nil {
		return saveStudentsToDB(students)
	}
	return saveStudentsToJSON(students)
}

func loadStudentsFromDB() []Student {
	rows, err := db.Query(`
		SELECT student_id, name, locations, phone_number, alt_phone_number, 
			guardian, pickup_time, dropoff_time, position_number, route_id, driver, active
		FROM students ORDER BY student_id
	`)
	if err != nil {
		log.Printf("Error loading students from DB: %v", err)
		return []Student{}
	}
	defer rows.Close()

	var students []Student
	for rows.Next() {
		var student Student
		var locationsJSON []byte
		if err := rows.Scan(&student.StudentID, &student.Name, &locationsJSON,
			&student.PhoneNumber, &student.AltPhoneNumber, &student.Guardian,
			&student.PickupTime, &student.DropoffTime, &student.PositionNumber,
			&student.RouteID, &student.Driver, &student.Active); err != nil {
			log.Printf("Error scanning student: %v", err)
			continue
		}
		
		// Parse locations JSON
		if len(locationsJSON) > 0 {
			if err := json.Unmarshal(locationsJSON, &student.Locations); err != nil {
				log.Printf("Error unmarshaling locations for student %s: %v", student.StudentID, err)
			}
		}
		
		students = append(students, student)
	}
	return students
}

func saveStudentsToDB(students []Student) error {
	// Individual student operations are handled in handlers
	return nil
}

func loadStudentsFromJSON() []Student {
	students, _ := loadJSON[Student]("data/students.json")
	return students
}

func saveStudentsToJSON(students []Student) error {
	return saveJSONFile("data/students.json", students)
}

// =============================================================================
// ROUTE ASSIGNMENT FUNCTIONS
// =============================================================================

func loadRouteAssignments() ([]RouteAssignment, error) {
	if db != nil {
		return loadRouteAssignmentsFromDB()
	}
	return loadRouteAssignmentsFromJSON()
}

func saveRouteAssignments(assignments []RouteAssignment) error {
	if db != nil {
		return saveRouteAssignmentsToDB(assignments)
	}
	return saveRouteAssignmentsToJSON(assignments)
}

func loadRouteAssignmentsFromDB() ([]RouteAssignment, error) {
	rows, err := db.Query(`
		SELECT driver, bus_id, route_id, route_name, assigned_date 
		FROM route_assignments ORDER BY driver
	`)
	if err != nil {
		log.Printf("Error loading route assignments from DB: %v", err)
		return []RouteAssignment{}, err
	}
	defer rows.Close()

	var assignments []RouteAssignment
	for rows.Next() {
		var assignment RouteAssignment
		if err := rows.Scan(&assignment.Driver, &assignment.BusID, &assignment.RouteID,
			&assignment.RouteName, &assignment.AssignedDate); err != nil {
			log.Printf("Error scanning route assignment: %v", err)
			continue
		}
		assignments = append(assignments, assignment)
	}
	return assignments, nil
}

func saveRouteAssignmentsToDB(assignments []RouteAssignment) error {
	// Individual assignment operations are handled in handlers
	return nil
}

func loadRouteAssignmentsFromJSON() ([]RouteAssignment, error) {
	return loadJSON[RouteAssignment]("data/route_assignments.json")
}

func saveRouteAssignmentsToJSON(assignments []RouteAssignment) error {
	return saveJSONFile("data/route_assignments.json", assignments)
}

// =============================================================================
// DRIVER LOG FUNCTIONS
// =============================================================================

func loadDriverLogs() ([]DriverLog, error) {
	if db != nil {
		return loadDriverLogsFromDB()
	}
	return loadDriverLogsFromJSON()
}

func saveDriverLogs(logs []DriverLog) error {
	if db != nil {
		return saveDriverLogsToDB(logs)
	}
	return saveDriverLogsToJSON(logs)
}

func loadDriverLogsFromDB() ([]DriverLog, error) {
	rows, err := db.Query(`
		SELECT driver, bus_id, route_id, date, period, departure_time, 
			arrival_time, mileage, attendance 
		FROM driver_logs ORDER BY date DESC, driver
	`)
	if err != nil {
		log.Printf("Error loading driver logs from DB: %v", err)
		return []DriverLog{}, err
	}
	defer rows.Close()

	var logs []DriverLog
	for rows.Next() {
		var driverLog DriverLog
		var attendanceJSON []byte
		if err := rows.Scan(&driverLog.Driver, &driverLog.BusID, &driverLog.RouteID,
			&driverLog.Date, &driverLog.Period, &driverLog.Departure,
			&driverLog.Arrival, &driverLog.Mileage, &attendanceJSON); err != nil {
			log.Printf("Error scanning driver log: %v", err)
			continue
		}
		
		// Parse attendance JSON
		if len(attendanceJSON) > 0 {
			if err := json.Unmarshal(attendanceJSON, &driverLog.Attendance); err != nil {
				log.Printf("Error unmarshaling attendance for log %s/%s: %v", driverLog.Driver, driverLog.Date, err)
			}
		}
		
		logs = append(logs, driverLog)
	}
	return logs, nil
}

func saveDriverLogsToDB(logs []DriverLog) error {
	// Individual log operations are handled in handlers
	return nil
}

func loadDriverLogsFromJSON() ([]DriverLog, error) {
	return loadJSON[DriverLog]("data/driver_logs.json")
}

func saveDriverLogsToJSON(logs []DriverLog) error {
	return saveJSONFile("data/driver_logs.json", logs)
}

// =============================================================================
// MAINTENANCE LOG FUNCTIONS
// =============================================================================

func loadMaintenanceLogs() []MaintenanceLog {
	if db != nil {
		return loadMaintenanceLogsFromDB()
	}
	return loadMaintenanceLogsFromJSON()
}

func saveMaintenanceLogs(logs []MaintenanceLog) error {
	if db != nil {
		return saveMaintenanceLogsToDB(logs)
	}
	return saveMaintenanceLogsToJSON(logs)
}

func loadMaintenanceLogsFromDB() []MaintenanceLog {
	rows, err := db.Query(`
		SELECT bus_id, date, category, notes, mileage 
		FROM maintenance_logs ORDER BY date DESC
	`)
	if err != nil {
		log.Printf("Error loading maintenance logs from DB: %v", err)
		return []MaintenanceLog{}
	}
	defer rows.Close()

	var logs []MaintenanceLog
	for rows.Next() {
		var maintenanceLog MaintenanceLog
		if err := rows.Scan(&maintenanceLog.BusID, &maintenanceLog.Date,
			&maintenanceLog.Category, &maintenanceLog.Notes, &maintenanceLog.Mileage); err != nil {
			log.Printf("Error scanning maintenance log: %v", err)
			continue
		}
		logs = append(logs, maintenanceLog)
	}
	return logs
}

func saveMaintenanceLogsToDB(logs []MaintenanceLog) error {
	// Individual log operations are handled in handlers
	return nil
}

func loadMaintenanceLogsFromJSON() []MaintenanceLog {
	logs, _ := loadJSON[MaintenanceLog]("data/maintenance.json")
	return logs
}

func saveMaintenanceLogsToJSON(logs []MaintenanceLog) error {
	return saveJSONFile("data/maintenance.json", logs)
}

// =============================================================================
// VEHICLE FUNCTIONS
// =============================================================================

func loadVehicles() []Vehicle {
	if db != nil {
		return loadVehiclesFromDB()
	}
	return loadVehiclesFromJSON()
}

func saveVehicles(vehicles []Vehicle) error {
	if db != nil {
		return saveVehiclesToDB(vehicles)
	}
	return saveVehiclesToJSON(vehicles)
}

func loadVehiclesFromDB() []Vehicle {
	rows, err := db.Query(`
		SELECT vehicle_id, model, description, year, tire_size, license,
			oil_status, tire_status, status, maintenance_notes, serial_number, base, service_interval
		FROM vehicles ORDER BY vehicle_id
	`)
	if err != nil {
		log.Printf("Error loading vehicles from DB: %v", err)
		return []Vehicle{}
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
	return vehicles
}

func saveVehiclesToDB(vehicles []Vehicle) error {
	// Start a transaction for bulk update
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// For each vehicle, update its status fields
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

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func loadVehiclesFromJSON() []Vehicle {
	vehicles, _ := loadJSON[Vehicle]("data/vehicle.json")
	return vehicles
}

func saveVehiclesToJSON(vehicles []Vehicle) error {
	return saveJSONFile("data/vehicle.json", vehicles)
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

func saveJSONFile(filename string, data interface{}) error {
	// Ensure data directory exists
	if err := os.MkdirAll("data", 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", filename, err)
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(data); err != nil {
		return fmt.Errorf("failed to encode data to %s: %w", filename, err)
	}

	return nil
}
