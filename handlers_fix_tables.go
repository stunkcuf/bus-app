package main

import (
	"log"
	"net/http"
	"time"
)

// fixTablesHandler recreates tables with correct structure
func fixTablesHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromSession(r)
	if user == nil || user.Role != "manager" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	if db == nil {
		http.Error(w, "Database not initialized", http.StatusInternalServerError)
		return
	}

	// Drop and recreate ECSE tables
	_, err := db.Exec(`
		DROP TABLE IF EXISTS ecse_attendance CASCADE;
		DROP TABLE IF EXISTS ecse_assessments CASCADE;
		DROP TABLE IF EXISTS ecse_services CASCADE;
		DROP TABLE IF EXISTS ecse_students CASCADE;
	`)
	if err != nil {
		log.Printf("Error dropping ECSE tables: %v", err)
	}

	// Recreate ECSE tables
	_, err = db.Exec(`
		CREATE TABLE ecse_students (
			student_id VARCHAR(50) PRIMARY KEY,
			first_name VARCHAR(100) NOT NULL,
			last_name VARCHAR(100) NOT NULL,
			date_of_birth DATE,
			grade VARCHAR(20),
			enrollment_status VARCHAR(50),
			iep_status VARCHAR(50),
			primary_disability VARCHAR(100),
			service_minutes INTEGER,
			transportation_required BOOLEAN DEFAULT false,
			bus_route VARCHAR(100),
			parent_name VARCHAR(100),
			parent_phone VARCHAR(20),
			parent_email VARCHAR(100),
			city VARCHAR(100),
			state VARCHAR(50),
			zip_code VARCHAR(20),
			notes TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		log.Printf("Error creating ecse_students table: %v", err)
		http.Error(w, "Failed to create ECSE tables", http.StatusInternalServerError)
		return
	}

	// Add ECSE sample data
	students := []struct {
		ID                     string
		FirstName              string
		LastName               string
		DOB                    string
		Grade                  string
		EnrollmentStatus       string
		IEPStatus              string
		PrimaryDisability      string
		ServiceMinutes         int
		TransportationRequired bool
		BusRoute               string
		ParentName             string
		ParentPhone            string
		ParentEmail            string
		City                   string
		State                  string
		ZipCode                string
		Notes                  string
	}{
		{
			ID: "ECSE001", FirstName: "Emma", LastName: "Johnson", DOB: "2019-03-15",
			Grade: "Pre-K", EnrollmentStatus: "active", IEPStatus: "active",
			PrimaryDisability: "Speech/Language Impairment", ServiceMinutes: 120,
			TransportationRequired: true, BusRoute: "ECSE-1", ParentName: "Sarah Johnson",
			ParentPhone: "555-0101", ParentEmail: "sarah.johnson@email.com",
			City: "Springfield", State: "IL", ZipCode: "62701",
			Notes: "Requires speech therapy 2x per week",
		},
		{
			ID: "ECSE002", FirstName: "Liam", LastName: "Smith", DOB: "2018-11-22",
			Grade: "Pre-K", EnrollmentStatus: "active", IEPStatus: "active",
			PrimaryDisability: "Autism Spectrum Disorder", ServiceMinutes: 180,
			TransportationRequired: true, BusRoute: "ECSE-1", ParentName: "Michael Smith",
			ParentPhone: "555-0102", ParentEmail: "m.smith@email.com",
			City: "Springfield", State: "IL", ZipCode: "62702",
			Notes: "Behavioral support plan in place",
		},
		{
			ID: "ECSE003", FirstName: "Sophia", LastName: "Williams", DOB: "2019-07-08",
			Grade: "Pre-K", EnrollmentStatus: "active", IEPStatus: "active",
			PrimaryDisability: "Developmental Delay", ServiceMinutes: 150,
			TransportationRequired: true, BusRoute: "ECSE-2", ParentName: "Jennifer Williams",
			ParentPhone: "555-0103", ParentEmail: "j.williams@email.com",
			City: "Springfield", State: "IL", ZipCode: "62703",
			Notes: "OT and PT services required",
		},
		{
			ID: "ECSE004", FirstName: "Noah", LastName: "Brown", DOB: "2019-01-30",
			Grade: "Pre-K", EnrollmentStatus: "active", IEPStatus: "evaluation",
			PrimaryDisability: "Other Health Impairment", ServiceMinutes: 90,
			TransportationRequired: false, BusRoute: "", ParentName: "David Brown",
			ParentPhone: "555-0104", ParentEmail: "d.brown@email.com",
			City: "Springfield", State: "IL", ZipCode: "62704",
			Notes: "Parent provides transportation",
		},
		{
			ID: "ECSE005", FirstName: "Olivia", LastName: "Davis", DOB: "2018-09-12",
			Grade: "Kindergarten", EnrollmentStatus: "active", IEPStatus: "active",
			PrimaryDisability: "Multiple Disabilities", ServiceMinutes: 240,
			TransportationRequired: true, BusRoute: "ECSE-1", ParentName: "Amanda Davis",
			ParentPhone: "555-0105", ParentEmail: "a.davis@email.com",
			City: "Springfield", State: "IL", ZipCode: "62705",
			Notes: "Requires full-time aide support",
		},
		{
			ID: "ECSE006", FirstName: "Ethan", LastName: "Miller", DOB: "2019-05-20",
			Grade: "Pre-K", EnrollmentStatus: "active", IEPStatus: "active",
			PrimaryDisability: "Intellectual Disability", ServiceMinutes: 180,
			TransportationRequired: true, BusRoute: "ECSE-2", ParentName: "Lisa Miller",
			ParentPhone: "555-0106", ParentEmail: "l.miller@email.com",
			City: "Springfield", State: "IL", ZipCode: "62706",
			Notes: "Adaptive PE services included",
		},
		{
			ID: "ECSE007", FirstName: "Ava", LastName: "Wilson", DOB: "2019-02-14",
			Grade: "Pre-K", EnrollmentStatus: "active", IEPStatus: "active",
			PrimaryDisability: "Hearing Impairment", ServiceMinutes: 120,
			TransportationRequired: true, BusRoute: "ECSE-1", ParentName: "Robert Wilson",
			ParentPhone: "555-0107", ParentEmail: "r.wilson@email.com",
			City: "Springfield", State: "IL", ZipCode: "62707",
			Notes: "Uses hearing aids, requires FM system",
		},
		{
			ID: "ECSE008", FirstName: "Mason", LastName: "Garcia", DOB: "2018-12-05",
			Grade: "Kindergarten", EnrollmentStatus: "active", IEPStatus: "active",
			PrimaryDisability: "Visual Impairment", ServiceMinutes: 150,
			TransportationRequired: true, BusRoute: "ECSE-2", ParentName: "Maria Garcia",
			ParentPhone: "555-0108", ParentEmail: "m.garcia@email.com",
			City: "Springfield", State: "IL", ZipCode: "62708",
			Notes: "Requires large print materials",
		},
	}

	for _, s := range students {
		_, err := db.Exec(`
			INSERT INTO ecse_students (
				student_id, first_name, last_name, date_of_birth, grade,
				enrollment_status, iep_status, primary_disability, service_minutes,
				transportation_required, bus_route, parent_name, parent_phone,
				parent_email, city, state, zip_code, notes, created_at
			) VALUES (
				$1, $2, $3, $4::DATE, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19
			)
		`, s.ID, s.FirstName, s.LastName, s.DOB, s.Grade, s.EnrollmentStatus,
			s.IEPStatus, s.PrimaryDisability, s.ServiceMinutes, s.TransportationRequired,
			s.BusRoute, s.ParentName, s.ParentPhone, s.ParentEmail, s.City, s.State,
			s.ZipCode, s.Notes, time.Now())

		if err != nil {
			log.Printf("Error inserting student %s: %v", s.ID, err)
		}
	}

	// Create ECSE services table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS ecse_services (
			id SERIAL PRIMARY KEY,
			student_id VARCHAR(50) NOT NULL REFERENCES ecse_students(student_id) ON DELETE CASCADE,
			service_type VARCHAR(50) NOT NULL,
			frequency VARCHAR(100),
			duration INTEGER,
			provider VARCHAR(100),
			start_date DATE,
			end_date DATE,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		log.Printf("Error creating ecse_services table: %v", err)
	}

	// Fix fuel records table
	_, err = db.Exec(`
		DROP TABLE IF EXISTS fuel_records CASCADE;
		CREATE TABLE fuel_records (
			id SERIAL PRIMARY KEY,
			vehicle_id VARCHAR(50) NOT NULL,
			date DATE NOT NULL,
			gallons DECIMAL(10,2) NOT NULL,
			price_per_gallon DECIMAL(10,2) NOT NULL,
			cost DECIMAL(10,2) NOT NULL,
			odometer INTEGER,
			location VARCHAR(255),
			driver VARCHAR(100),
			notes TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		log.Printf("Error recreating fuel_records table: %v", err)
	}

	// Add sample fuel records
	buses := []string{"B001", "B002", "B003", "B004", "B005"}
	now := time.Now()
	
	for _, busID := range buses {
		for w := 0; w < 4; w++ {
			date := now.AddDate(0, 0, -(w * 7)).Format("2006-01-02")
			gallons := 25.0 + float64(w*2)
			pricePerGallon := 3.50 + (float64(w) * 0.10)
			totalCost := gallons * pricePerGallon
			
			_, err := db.Exec(`
				INSERT INTO fuel_records (
					vehicle_id, date, gallons, price_per_gallon, cost,
					location, odometer, driver, created_at
				) VALUES (
					$1, $2, $3, $4, $5, 'Main Depot', $6, 'fleet_manager', $7
				)
			`, busID, date, gallons, pricePerGallon, totalCost, 50000 + (w * 500), time.Now())
			
			if err != nil {
				log.Printf("Error inserting fuel record: %v", err)
			}
		}
	}

	// Create both bus and vehicle maintenance logs tables
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS bus_maintenance_logs (
			id SERIAL PRIMARY KEY,
			bus_id VARCHAR(50) NOT NULL,
			date DATE NOT NULL,
			category VARCHAR(50),
			notes TEXT,
			mileage INTEGER,
			cost DECIMAL(10,2),
			performed_by VARCHAR(100),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		log.Printf("Error creating bus_maintenance_logs table: %v", err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS vehicle_maintenance_logs (
			id SERIAL PRIMARY KEY,
			vehicle_id VARCHAR(50) NOT NULL,
			date DATE NOT NULL,
			category VARCHAR(50),
			notes TEXT,
			mileage INTEGER,
			cost DECIMAL(10,2),
			performed_by VARCHAR(100),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		log.Printf("Error creating maintenance logs table: %v", err)
	}

	// Add sample maintenance logs for buses
	for _, busID := range buses {
		categories := []string{"oil_change", "tire_rotation", "inspection", "repair"}
		for i, category := range categories {
			date := now.AddDate(0, -(i + 1), 0).Format("2006-01-02")
			cost := 100.0 + float64(i*50)
			
			_, err := db.Exec(`
				INSERT INTO bus_maintenance_logs (
					bus_id, date, category, notes, mileage, cost, performed_by, created_at
				) VALUES (
					$1, $2, $3, $4, $5, $6, 'Maintenance Team', $7
				)
			`, busID, date, category, "Regular maintenance performed", 50000 + (i * 5000), cost, time.Now())
			
			if err != nil {
				log.Printf("Error inserting bus maintenance log: %v", err)
			}
		}
	}

	// Add sample maintenance logs for vehicles
	vehicles := []string{"FV001", "FV002", "FV003", "FV004", "FV005"}
	for _, vehicleID := range vehicles {
		categories := []string{"oil_change", "tire_rotation", "inspection", "repair"}
		for i, category := range categories {
			date := now.AddDate(0, -(i + 1), 0).Format("2006-01-02")
			cost := 150.0 + float64(i*75)
			
			_, err := db.Exec(`
				INSERT INTO vehicle_maintenance_logs (
					vehicle_id, date, category, notes, mileage, cost, performed_by, created_at
				) VALUES (
					$1, $2, $3, $4, $5, $6, 'Fleet Service Center', $7
				)
			`, vehicleID, date, category, "Scheduled maintenance completed", 45000 + (i * 5000), cost, time.Now())
			
			if err != nil {
				log.Printf("Error inserting vehicle maintenance log: %v", err)
			}
		}
	}

	log.Println("Tables fixed and sample data added!")
	
	// Redirect to ECSE dashboard
	http.Redirect(w, r, "/ecse-dashboard", http.StatusSeeOther)
}