package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"
)

// NotificationTriggers handles automated notifications based on system events
type NotificationTriggers struct {
	system *NotificationSystem
}

// NewNotificationTriggers creates a new notification triggers manager
func NewNotificationTriggers(system *NotificationSystem) *NotificationTriggers {
	return &NotificationTriggers{
		system: system,
	}
}

// TriggerMaintenanceDueNotifications checks for vehicles due for maintenance
func (nt *NotificationTriggers) TriggerMaintenanceDueNotifications() {
	// Check for oil changes due
	oilDueVehicles, err := getVehiclesDueForOilChange(7) // 7 days notice
	if err != nil {
		log.Printf("Error checking oil due vehicles: %v", err)
		return
	}

	for _, vehicle := range oilDueVehicles {
		notification := BuildMaintenanceNotification(vehicle, time.Now().AddDate(0, 0, 7))
		notification.Message = fmt.Sprintf("Vehicle %s is due for an oil change. Current mileage: %d, Last oil change: %d miles",
			vehicle.VehicleID, vehicle.CurrentMileage, vehicle.LastOilChange)
		
		// Get managers to notify
		recipients, err := nt.getManagerRecipients()
		if err != nil {
			log.Printf("Error getting recipients: %v", err)
			continue
		}
		
		// Also notify assigned drivers for this vehicle
		drivers, err := nt.getDriversForVehicle(vehicle.VehicleID)
		if err == nil && len(drivers) > 0 {
			recipients = append(recipients, drivers...)
		}
		
		notification.Recipients = recipients
		
		if err := nt.system.Send(notification); err != nil {
			log.Printf("Failed to send maintenance notification: %v", err)
		}
	}

	// Check for tire service due
	tireDueVehicles, err := getVehiclesDueForTireService(14) // 14 days notice
	if err != nil {
		log.Printf("Error checking tire due vehicles: %v", err)
		return
	}

	for _, vehicle := range tireDueVehicles {
		notification := BuildMaintenanceNotification(vehicle, time.Now().AddDate(0, 0, 14))
		notification.Message = fmt.Sprintf("Vehicle %s is due for tire service. Current mileage: %d, Last tire service: %d miles",
			vehicle.VehicleID, vehicle.CurrentMileage, vehicle.LastTireService)
		
		recipients, err := nt.getManagerRecipients()
		if err != nil {
			log.Printf("Error getting recipients: %v", err)
			continue
		}
		
		notification.Recipients = recipients
		
		if err := nt.system.Send(notification); err != nil {
			log.Printf("Failed to send tire service notification: %v", err)
		}
	}
}

// TriggerAttendanceIssueNotifications checks for student attendance issues
func (nt *NotificationTriggers) TriggerAttendanceIssueNotifications() {
	// Check for students marked absent today
	absentStudents, err := getAbsentStudentsToday()
	if err != nil {
		log.Printf("Error checking absent students: %v", err)
		return
	}

	for _, student := range absentStudents {
		notification := Notification{
			Type:     NotifyAttendanceIssue,
			Priority: "medium",
			Subject:  fmt.Sprintf("Student Absence Alert: %s", student.Name),
			Message:  fmt.Sprintf("Student %s was marked absent from the %s route today",
				student.Name, student.RouteID),
			Data: map[string]interface{}{
				"student_id": student.StudentID,
				"route_id":   student.RouteID,
				"date":       time.Now().Format("2006-01-02"),
			},
			Channels: []string{"email", "in-app"},
		}

		// Notify managers
		recipients, err := nt.getManagerRecipients()
		if err != nil {
			log.Printf("Error getting recipients: %v", err)
			continue
		}
		
		notification.Recipients = recipients
		
		if err := nt.system.Send(notification); err != nil {
			log.Printf("Failed to send attendance notification: %v", err)
		}
	}
}

// TriggerVehicleStatusChangeNotification for vehicle status updates
func (nt *NotificationTriggers) TriggerVehicleStatusChangeNotification(vehicleID, oldStatus, newStatus, changedBy string) {
	// Don't notify for minor changes
	if oldStatus == newStatus {
		return
	}

	// Get vehicle details
	vehicle, err := getVehicleByID(vehicleID)
	if err != nil {
		log.Printf("Error getting vehicle: %v", err)
		return
	}

	priority := "low"
	if newStatus == "out_of_service" {
		priority = "high"
	} else if newStatus == "maintenance" {
		priority = "medium"
	}

	notification := Notification{
		Type:     NotifyVehicleIssue,
		Priority: priority,
		Subject:  fmt.Sprintf("Vehicle Status Changed: %s", vehicle.Model),
		Message: fmt.Sprintf("Vehicle %s (%s) status changed from %s to %s by %s",
			vehicleID, vehicle.Model, oldStatus, newStatus, changedBy),
		Data: map[string]interface{}{
			"vehicle_id": vehicleID,
			"old_status": oldStatus,
			"new_status": newStatus,
			"changed_by": changedBy,
			"timestamp":  time.Now(),
		},
		Channels: []string{"email", "push", "in-app"},
	}

	// Notify managers
	recipients, err := nt.getManagerRecipients()
	if err != nil {
		log.Printf("Error getting recipients: %v", err)
		return
	}

	// Also notify affected drivers if vehicle is out of service
	if newStatus == "out_of_service" {
		affectedDrivers, err := nt.getDriversForVehicle(vehicleID)
		if err == nil {
			recipients = append(recipients, affectedDrivers...)
		}
	}

	notification.Recipients = recipients

	if err := nt.system.Send(notification); err != nil {
		log.Printf("Failed to send vehicle status notification: %v", err)
	}
}

// TriggerRouteAssignmentNotification when routes are assigned or changed
func (nt *NotificationTriggers) TriggerRouteAssignmentNotification(driver, busID, routeID string, isNew bool) {
	action := "assigned to"
	if !isNew {
		action = "reassigned to"
	}

	notification := Notification{
		Type:     NotifyRouteChange,
		Priority: "medium",
		Subject:  "Route Assignment Update",
		Message:  fmt.Sprintf("Driver %s has been %s route %s with bus %s",
			driver, action, routeID, busID),
		Data: map[string]interface{}{
			"driver":   driver,
			"bus_id":   busID,
			"route_id": routeID,
			"is_new":   isNew,
		},
		Channels: []string{"email", "push", "in-app"},
	}

	// Notify the driver
	driverRecipient, err := nt.getUserRecipient(driver)
	if err != nil {
		log.Printf("Error getting driver recipient: %v", err)
		return
	}

	notification.Recipients = []Recipient{driverRecipient}

	if err := nt.system.Send(notification); err != nil {
		log.Printf("Failed to send route assignment notification: %v", err)
	}
}

// TriggerDailyReminderNotifications sends daily schedule reminders
func (nt *NotificationTriggers) TriggerDailyReminderNotifications() {
	// Get all active drivers with routes for tomorrow
	tomorrow := time.Now().AddDate(0, 0, 1)
	driversWithRoutes, err := getDriversWithRoutesForDate(tomorrow)
	if err != nil {
		log.Printf("Error getting drivers with routes: %v", err)
		return
	}

	for _, assignment := range driversWithRoutes {
		notification := Notification{
			Type:     NotifyScheduleReminder,
			Priority: "low",
			Subject:  "Tomorrow's Route Reminder",
			Message: fmt.Sprintf("Reminder: You are scheduled to drive route %s with bus %s tomorrow",
				assignment.RouteID, assignment.BusID),
			Data: map[string]interface{}{
				"date":     tomorrow.Format("2006-01-02"),
				"route_id": assignment.RouteID,
				"bus_id":   assignment.BusID,
			},
			Channels:    []string{"email", "push"},
			ScheduledAt: &[]time.Time{tomorrow.Add(-12 * time.Hour)}[0], // Send at 12 hours before
		}

		// Notify the driver
		driverRecipient, err := nt.getUserRecipient(assignment.Driver)
		if err != nil {
			log.Printf("Error getting driver recipient: %v", err)
			continue
		}

		notification.Recipients = []Recipient{driverRecipient}

		if err := nt.system.Send(notification); err != nil {
			log.Printf("Failed to send daily reminder notification: %v", err)
		}
	}
}

// TriggerReportGeneratedNotification when reports are ready
func (nt *NotificationTriggers) TriggerReportGeneratedNotification(reportType, reportName string, generatedBy string, downloadURL string) {
	notification := Notification{
		Type:     NotifyReportReady,
		Priority: "low",
		Subject:  fmt.Sprintf("Report Ready: %s", reportName),
		Message:  fmt.Sprintf("Your %s report '%s' has been generated and is ready for download",
			reportType, reportName),
		Data: map[string]interface{}{
			"report_type": reportType,
			"report_name": reportName,
			"download_url": downloadURL,
			"generated_at": time.Now(),
		},
		Channels: []string{"email", "in-app"},
	}

	// Notify the user who requested the report
	recipient, err := nt.getUserRecipient(generatedBy)
	if err != nil {
		log.Printf("Error getting recipient: %v", err)
		return
	}

	notification.Recipients = []Recipient{recipient}

	if err := nt.system.Send(notification); err != nil {
		log.Printf("Failed to send report ready notification: %v", err)
	}
}

// TriggerEmergencyNotification for emergency situations
func (nt *NotificationTriggers) TriggerEmergencyNotification(driver, message string, location *LocationUpdate) {
	notification := BuildEmergencyNotification(driver, message, location)

	// Get all managers and supervisors
	recipients, err := nt.getEmergencyRecipients()
	if err != nil {
		log.Printf("Error getting emergency recipients: %v", err)
		// Continue anyway - this is an emergency
	}

	notification.Recipients = recipients

	// Send immediately
	if err := nt.system.Send(notification); err != nil {
		log.Printf("CRITICAL: Failed to send emergency notification: %v", err)
	}
}

// Helper methods to get recipients

func (nt *NotificationTriggers) getManagerRecipients() ([]Recipient, error) {
	rows, err := db.Query(`
		SELECT username, COALESCE(email, '') as email, COALESCE(phone, '') as phone
		FROM users
		WHERE role = 'manager' AND status = 'active'
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var recipients []Recipient
	for rows.Next() {
		var r Recipient
		err := rows.Scan(&r.Username, &r.Email, &r.Phone)
		if err != nil {
			continue
		}
		
		r.UserID = r.Username
		r.Preferences = NotificationPreferences{
			Email: true,
			SMS:   true,
			Push:  true,
		}
		
		recipients = append(recipients, r)
	}

	return recipients, nil
}

func (nt *NotificationTriggers) getUserRecipient(username string) (Recipient, error) {
	var r Recipient
	err := db.QueryRow(`
		SELECT username, COALESCE(email, '') as email, COALESCE(phone, '') as phone
		FROM users
		WHERE username = $1
	`, username).Scan(&r.Username, &r.Email, &r.Phone)
	
	if err != nil {
		return r, err
	}
	
	r.UserID = r.Username
	r.Preferences = NotificationPreferences{
		Email: true,
		SMS:   false, // Default to email only for regular users
		Push:  true,
	}
	
	return r, nil
}

func (nt *NotificationTriggers) getDriversForVehicle(vehicleID string) ([]Recipient, error) {
	// Get drivers assigned to routes with this vehicle
	rows, err := db.Query(`
		SELECT DISTINCT u.username, COALESCE(u.email, '') as email, COALESCE(u.phone, '') as phone
		FROM users u
		JOIN route_assignments ra ON u.username = ra.driver
		WHERE ra.bus_id = $1 AND u.status = 'active'
	`, vehicleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var recipients []Recipient
	for rows.Next() {
		var r Recipient
		err := rows.Scan(&r.Username, &r.Email, &r.Phone)
		if err != nil {
			continue
		}
		
		r.UserID = r.Username
		r.Preferences = NotificationPreferences{
			Email: true,
			SMS:   false,
			Push:  true,
		}
		
		recipients = append(recipients, r)
	}

	return recipients, nil
}

func (nt *NotificationTriggers) getEmergencyRecipients() ([]Recipient, error) {
	// For emergencies, notify all managers and supervisors
	rows, err := db.Query(`
		SELECT username, COALESCE(email, '') as email, COALESCE(phone, '') as phone
		FROM users
		WHERE role IN ('manager', 'supervisor') AND status = 'active'
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var recipients []Recipient
	for rows.Next() {
		var r Recipient
		err := rows.Scan(&r.Username, &r.Email, &r.Phone)
		if err != nil {
			continue
		}
		
		r.UserID = r.Username
		r.Preferences = NotificationPreferences{
			Email: true,
			SMS:   true, // Enable SMS for emergencies
			Push:  true,
		}
		
		recipients = append(recipients, r)
	}

	return recipients, nil
}

// Database helper functions

func getVehicleByID(vehicleID string) (*Vehicle, error) {
	var vehicle Vehicle
	
	// Try vehicles table first
	err := db.QueryRow(`
		SELECT vehicle_id, COALESCE(model, ''), COALESCE(description, ''),
			   COALESCE(current_mileage, 0), COALESCE(last_oil_change, 0),
			   COALESCE(last_tire_service, 0)
		FROM vehicles
		WHERE vehicle_id = $1
	`, vehicleID).Scan(&vehicle.VehicleID, &vehicle.Model, &vehicle.Description,
		&vehicle.CurrentMileage, &vehicle.LastOilChange, &vehicle.LastTireService)
	
	if err == sql.ErrNoRows {
		// Try buses table
		err = db.QueryRow(`
			SELECT bus_id as vehicle_id, COALESCE(model, ''), '' as description,
				   COALESCE(current_mileage, 0), COALESCE(last_oil_change, 0),
				   COALESCE(last_tire_service, 0)
			FROM buses
			WHERE bus_id = $1
		`, vehicleID).Scan(&vehicle.VehicleID, &vehicle.Model, &vehicle.Description,
			&vehicle.CurrentMileage, &vehicle.LastOilChange, &vehicle.LastTireService)
	}
	
	if err != nil {
		return nil, err
	}
	
	return &vehicle, nil
}

func getVehiclesDueForOilChange(daysNotice int) ([]Vehicle, error) {
	var vehicles []Vehicle
	
	// Check vehicles where current mileage exceeds last oil change by threshold
	rows, err := db.Query(`
		SELECT vehicle_id, COALESCE(model, ''), COALESCE(description, ''), 
			   COALESCE(current_mileage, 0), COALESCE(last_oil_change, 0)
		FROM vehicles
		WHERE status = 'active'
		AND current_mileage - last_oil_change >= 4500  -- 5000 mile interval with 500 mile buffer
		UNION
		SELECT bus_id as vehicle_id, COALESCE(model, ''), '' as description,
			   COALESCE(current_mileage, 0), COALESCE(last_oil_change, 0)
		FROM buses
		WHERE status = 'active'
		AND current_mileage - last_oil_change >= 4500
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var v Vehicle
		err := rows.Scan(&v.VehicleID, &v.Model, &v.Description, 
			&v.CurrentMileage, &v.LastOilChange)
		if err != nil {
			continue
		}
		vehicles = append(vehicles, v)
	}

	return vehicles, nil
}

func getVehiclesDueForTireService(daysNotice int) ([]Vehicle, error) {
	var vehicles []Vehicle
	
	// Check vehicles where current mileage exceeds last tire service by threshold
	rows, err := db.Query(`
		SELECT vehicle_id, COALESCE(model, ''), COALESCE(description, ''),
			   COALESCE(current_mileage, 0), COALESCE(last_tire_service, 0)
		FROM vehicles
		WHERE status = 'active'
		AND current_mileage - last_tire_service >= 19000  -- 20000 mile interval with 1000 mile buffer
		UNION
		SELECT bus_id as vehicle_id, COALESCE(model, ''), '' as description,
			   COALESCE(current_mileage, 0), COALESCE(last_tire_service, 0)
		FROM buses
		WHERE status = 'active'
		AND current_mileage - last_tire_service >= 19000
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var v Vehicle
		err := rows.Scan(&v.VehicleID, &v.Model, &v.Description,
			&v.CurrentMileage, &v.LastTireService)
		if err != nil {
			continue
		}
		vehicles = append(vehicles, v)
	}

	return vehicles, nil
}

func getAbsentStudentsToday() ([]Student, error) {
	var students []Student
	
	today := time.Now().Format("2006-01-02")
	rows, err := db.Query(`
		SELECT s.student_id, s.name, COALESCE(s.route_id, '')
		FROM students s
		JOIN student_attendance sa ON s.student_id = sa.student_id
		WHERE sa.date = $1
		AND sa.present = false
		AND s.active = true
	`, today)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var s Student
		err := rows.Scan(&s.StudentID, &s.Name, &s.RouteID)
		if err != nil {
			continue
		}
		students = append(students, s)
	}

	return students, nil
}

func getDriversWithRoutesForDate(date time.Time) ([]RouteAssignment, error) {
	var assignments []RouteAssignment
	
	rows, err := db.Query(`
		SELECT driver, bus_id, route_id
		FROM route_assignments
		WHERE assigned_date <= $1
		ORDER BY driver
	`, date.Format("2006-01-02"))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var a RouteAssignment
		err := rows.Scan(&a.Driver, &a.BusID, &a.RouteID)
		if err != nil {
			continue
		}
		assignments = append(assignments, a)
	}

	return assignments, nil
}

// Initialize notification triggers
var notificationTriggers *NotificationTriggers

func InitializeNotificationTriggers() {
	if notificationSystem == nil {
		log.Println("Notification system not initialized, skipping triggers")
		return
	}

	notificationTriggers = NewNotificationTriggers(notificationSystem)
	
	// Start scheduled trigger jobs
	go runScheduledNotificationTriggers()
	
	log.Println("Notification triggers initialized")
}

// Run scheduled notification checks
func runScheduledNotificationTriggers() {
	// Daily maintenance check at 8 AM
	go scheduleDaily(8, 0, func() {
		log.Println("Running daily maintenance notifications")
		notificationTriggers.TriggerMaintenanceDueNotifications()
	})

	// Daily attendance check at 10 AM
	go scheduleDaily(10, 0, func() {
		log.Println("Running daily attendance notifications")
		notificationTriggers.TriggerAttendanceIssueNotifications()
	})

	// Daily reminders at 6 PM
	go scheduleDaily(18, 0, func() {
		log.Println("Running daily reminder notifications")
		notificationTriggers.TriggerDailyReminderNotifications()
	})
}

// Schedule a function to run daily at specific time
func scheduleDaily(hour, minute int, fn func()) {
	for {
		now := time.Now()
		next := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())
		
		if now.After(next) {
			next = next.Add(24 * time.Hour)
		}
		
		duration := next.Sub(now)
		log.Printf("Scheduled task will run in %v", duration)
		
		time.Sleep(duration)
		fn()
		
		// Sleep for a minute to avoid double execution
		time.Sleep(time.Minute)
	}
}