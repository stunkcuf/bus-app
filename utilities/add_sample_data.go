package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgresql://postgres:jTKOWGEzlprRGbkPBCgGxsgnwyLeGoDL@shortline.proxy.rlwy.net:40148/railway"
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	fmt.Println("Adding sample data for driver dashboard testing...")

	// Ensure test driver has a route assignment
	_, err = db.Exec(`
		INSERT INTO route_assignments (driver, bus_id, route_id, assigned_date)
		SELECT 'test', '1', '5', CURRENT_DATE
		WHERE NOT EXISTS (
			SELECT 1 FROM route_assignments 
			WHERE driver = 'test' AND status = 'active'
		)`)
	if err != nil {
		log.Printf("Error adding route assignment: %v", err)
	}

	// Add sample students (without grade column)
	students := []struct {
		ID       string
		Name     string
		Phone    string
		AltPhone string
		Guardian string
		Position int
		Address  string
		Pickup   string
		Dropoff  string
	}{
		{"STU001", "Emma Johnson", "555-0101", "555-0102", "Sarah Johnson", 1, "123 Oak Street", "7:00", "3:30"},
		{"STU002", "Liam Smith", "555-0103", "555-0104", "Michael Smith", 2, "456 Pine Avenue", "7:05", "3:35"},
		{"STU003", "Olivia Davis", "555-0105", "555-0106", "Jennifer Davis", 3, "789 Maple Drive", "7:10", "3:40"},
		{"STU004", "Noah Brown", "555-0107", "555-0108", "David Brown", 4, "321 Elm Street", "7:15", "3:45"},
		{"STU005", "Ava Wilson", "555-0109", "555-0110", "Lisa Wilson", 5, "654 Cedar Lane", "7:20", "3:50"},
		{"STU006", "Ethan Martinez", "555-0111", "555-0112", "Maria Martinez", 6, "987 Birch Road", "7:25", "3:55"},
		{"STU007", "Sophia Anderson", "555-0113", "555-0114", "Robert Anderson", 7, "147 Willow Way", "7:30", "4:00"},
		{"STU008", "Mason Taylor", "555-0115", "555-0116", "Emily Taylor", 8, "258 Spruce Street", "7:35", "4:05"},
	}

	for _, s := range students {
		locations := fmt.Sprintf(`[{"position": %d, "address": "%s", "pickup_time": "%s", "dropoff_time": "%s"}]`,
			s.Position, s.Address, s.Pickup, s.Dropoff)
		
		_, err = db.Exec(`
			INSERT INTO students (student_id, name, phone_number, alt_phone_number, guardian, locations)
			VALUES ($1, $2, $3, $4, $5, $6::jsonb)
			ON CONFLICT (student_id) DO UPDATE SET
				name = EXCLUDED.name,
				phone_number = EXCLUDED.phone_number,
				locations = EXCLUDED.locations`,
			s.ID, s.Name, s.Phone, s.AltPhone, s.Guardian, locations)
		
		if err != nil {
			log.Printf("Error adding student %s: %v", s.ID, err)
		} else {
			fmt.Printf("Added student: %s\n", s.Name)
		}
	}

	// Update Route 5 positions
	_, err = db.Exec(`UPDATE routes SET positions = '[1,2,3,4,5,6,7,8]'::jsonb WHERE route_id = '5'`)
	if err != nil {
		log.Printf("Error updating route positions: %v", err)
	}

	// Add driver logs (using correct column names)
	type LogEntry struct {
		Date         time.Time
		Period       string
		Departure    string
		Arrival      string
		BeginMileage float64
		EndMileage   float64
		Attendance   string
	}

	logs := []LogEntry{
		// Today morning
		{
			Date:         time.Now(),
			Period:       "morning",
			Departure:    "06:45",
			Arrival:      "08:00",
			BeginMileage: 15000,
			EndMileage:   15025,
			Attendance: `[{"position": 1, "present": true, "pickup_time": "07:00"}, 
				{"position": 2, "present": true, "pickup_time": "07:05"},
				{"position": 3, "present": false, "pickup_time": null},
				{"position": 4, "present": true, "pickup_time": "07:15"},
				{"position": 5, "present": true, "pickup_time": "07:20"},
				{"position": 6, "present": true, "pickup_time": "07:25"},
				{"position": 7, "present": true, "pickup_time": "07:30"},
				{"position": 8, "present": true, "pickup_time": "07:35"}]`,
		},
		// Yesterday
		{
			Date:         time.Now().AddDate(0, 0, -1),
			Period:       "morning",
			Departure:    "06:45",
			Arrival:      "08:00",
			BeginMileage: 14950,
			EndMileage:   14975,
			Attendance: `[{"position": 1, "present": true, "pickup_time": "07:00"}, 
				{"position": 2, "present": true, "pickup_time": "07:05"},
				{"position": 3, "present": true, "pickup_time": "07:10"},
				{"position": 4, "present": true, "pickup_time": "07:15"},
				{"position": 5, "present": true, "pickup_time": "07:20"},
				{"position": 6, "present": false, "pickup_time": null},
				{"position": 7, "present": true, "pickup_time": "07:30"},
				{"position": 8, "present": true, "pickup_time": "07:35"}]`,
		},
		{
			Date:         time.Now().AddDate(0, 0, -1),
			Period:       "afternoon",
			Departure:    "14:30",
			Arrival:      "16:00",
			BeginMileage: 14975,
			EndMileage:   15000,
			Attendance: `[{"position": 1, "present": true, "pickup_time": "15:30"}, 
				{"position": 2, "present": true, "pickup_time": "15:35"},
				{"position": 3, "present": true, "pickup_time": "15:40"},
				{"position": 4, "present": true, "pickup_time": "15:45"},
				{"position": 5, "present": true, "pickup_time": "15:50"},
				{"position": 6, "present": false, "pickup_time": null},
				{"position": 7, "present": true, "pickup_time": "16:00"},
				{"position": 8, "present": true, "pickup_time": "16:05"}]`,
		},
	}

	for _, log := range logs {
		_, err = db.Exec(`
			INSERT INTO driver_logs (driver, bus_id, route_id, date, period, departure_time, arrival_time, start_mileage, end_mileage, attendance)
			VALUES ('test', '1', '5', $1, $2, $3, $4, $5, $6, $7::jsonb)`,
			log.Date.Format("2006-01-02"), log.Period, log.Departure, log.Arrival, log.BeginMileage, log.EndMileage, log.Attendance)
		
		if err != nil {
			fmt.Printf("Error adding driver log: %v\n", err)
		} else {
			fmt.Printf("Added driver log for %s %s\n", log.Date.Format("2006-01-02"), log.Period)
		}
	}

	// Skip notifications for now as schema is different
	fmt.Println("\nSkipping notifications - schema mismatch")

	fmt.Println("\nSample data added successfully!")
	fmt.Println("You can now test the driver dashboard with the 'test' account.")
}