package main

import (
	"time"
)

// User represents a user in the system
type User struct {
	Username         string    `json:"username" db:"username"`
	Password         string    `json:"password,omitempty" db:"password"`
	Role             string    `json:"role" db:"role"`
	Status           string    `json:"status" db:"status"`
	RegistrationDate time.Time `json:"registration_date" db:"registration_date"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
}

// Bus represents a school bus
type Bus struct {
	ID               int       `json:"id" db:"id"`
	BusID            string    `json:"bus_id" db:"bus_id"`
	Status           string    `json:"status" db:"status"`
	Model            string    `json:"model" db:"model"`
	Capacity         int       `json:"capacity" db:"capacity"`
	OilStatus        string    `json:"oil_status" db:"oil_status"`
	TireStatus       string    `json:"tire_status" db:"tire_status"`
	MaintenanceNotes string    `json:"maintenance_notes" db:"maintenance_notes"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time `json:"updated_at" db:"updated_at"`
}

// Vehicle represents a company vehicle
type Vehicle struct {
	ID               int       `json:"id" db:"id"`
	VehicleID        string    `json:"vehicle_id" db:"vehicle_id"`
	Model            string    `json:"model" db:"model"`
	Description      string    `json:"description" db:"description"`
	Year             string    `json:"year" db:"year"` // VARCHAR in database
	TireSize         string    `json:"tire_size" db:"tire_size"`
	License          string    `json:"license" db:"license"`
	OilStatus        string    `json:"oil_status" db:"oil_status"`
	TireStatus       string    `json:"tire_status" db:"tire_status"`
	Status           string    `json:"status" db:"status"`
	MaintenanceNotes string    `json:"maintenance_notes" db:"maintenance_notes"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
	SerialNumber     string    `json:"serial_number" db:"serial_number"`
	Base             string    `json:"base" db:"base"`
	ServiceInterval  int       `json:"service_interval" db:"service_interval"`
	UpdatedAt        time.Time `json:"updated_at" db:"updated_at"`
	ImportID         string    `json:"import_id" db:"import_id"`
}

// BusMaintenanceLog represents a maintenance record for a bus
type BusMaintenanceLog struct {
	ID        int       `json:"id" db:"id"`
	BusID     string    `json:"bus_id" db:"bus_id"`
	Date      string    `json:"date" db:"date"`
	Category  string    `json:"category" db:"category"`
	Notes     string    `json:"notes" db:"notes"`
	Mileage   int       `json:"mileage" db:"mileage"`
	Cost      float64   `json:"cost" db:"cost"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// VehicleMaintenanceLog represents a maintenance record for a vehicle
type VehicleMaintenanceLog struct {
	ID        int       `json:"id" db:"id"`
	VehicleID string    `json:"vehicle_id" db:"vehicle_id"`
	Date      string    `json:"date" db:"date"`
	Category  string    `json:"category" db:"category"`
	Notes     string    `json:"notes" db:"notes"`
	Mileage   int       `json:"mileage" db:"mileage"`
	Cost      float64   `json:"cost" db:"cost"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// CombinedMaintenanceLog represents a unified maintenance record
type CombinedMaintenanceLog struct {
	ID         int       `json:"id" db:"id"`
	VehicleID  string    `json:"vehicle_id" db:"vehicle_id"`
	VehicleType string   `json:"vehicle_type" db:"vehicle_type"`
	Date       string    `json:"date" db:"date"`
	Category   string    `json:"category" db:"category"`
	Notes      string    `json:"notes" db:"notes"`
	Mileage    int       `json:"mileage" db:"mileage"`
	Cost       float64   `json:"cost" db:"cost"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

// MaintenanceSchedule defines when maintenance is due
type MaintenanceSchedule struct {
	ItemName        string
	Interval        int  // miles
	WarningMiles    int  // miles before due to show warning
	CriticalMiles   int  // miles overdue to show critical
}

// MaintenanceAlert represents a maintenance alert
type MaintenanceAlert struct {
	VehicleID    string `json:"vehicle_id"`
	VehicleType  string `json:"vehicle_type"`
	AlertType    string `json:"alert_type"`
	ItemName     string `json:"item_name"`
	Message      string `json:"message"`
	Severity     string `json:"severity"` // "warning", "due", "overdue"
	MilesOverdue int    `json:"miles_overdue"`
}

// MileageValidation represents validation result
type MileageValidation struct {
	Valid   bool   `json:"valid"`
	Error   string `json:"error,omitempty"`
	Warning string `json:"warning,omitempty"`
}

// Route represents a bus route
type Route struct {
	RouteID     string    `json:"route_id" db:"route_id"`
	RouteName   string    `json:"route_name" db:"route_name"`
	Description string    `json:"description" db:"description"`
	Positions   string    `json:"positions" db:"positions"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// RouteAssignment represents a driver-bus-route assignment
type RouteAssignment struct {
	ID           int       `json:"id" db:"id"`
	Driver       string    `json:"driver" db:"driver"`
	BusID        string    `json:"bus_id" db:"bus_id"`
	RouteID      string    `json:"route_id" db:"route_id"`
	RouteName    string    `json:"route_name" db:"route_name"`
	AssignedDate string    `json:"assigned_date" db:"assigned_date"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

// Student represents a student
type Student struct {
	StudentID      string    `json:"student_id" db:"student_id"`
	Name           string    `json:"name" db:"name"`
	Locations      string    `json:"locations" db:"locations"`
	PhoneNumber    string    `json:"phone_number" db:"phone_number"`
	AltPhoneNumber string    `json:"alt_phone_number" db:"alt_phone_number"`
	Guardian       string    `json:"guardian" db:"guardian"`
	PickupTime     string    `json:"pickup_time" db:"pickup_time"`
	DropoffTime    string    `json:"dropoff_time" db:"dropoff_time"`
	PositionNumber int       `json:"position_number" db:"position_number"`
	RouteID        string    `json:"route_id" db:"route_id"`
	Driver         string    `json:"driver" db:"driver"`
	Active         bool      `json:"active" db:"active"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
}

// DriverLog represents a driver's daily log
type DriverLog struct {
	ID           int       `json:"id" db:"id"`
	Driver       string    `json:"driver" db:"driver"`
	BusID        string    `json:"bus_id" db:"bus_id"`
	RouteID      string    `json:"route_id" db:"route_id"`
	Date         string    `json:"date" db:"date"`
	Period       string    `json:"period" db:"period"`
	Departure    string    `json:"departure_time" db:"departure_time"`
	Arrival      string    `json:"arrival_time" db:"arrival_time"`
	BeginMileage float64   `json:"begin_mileage" db:"begin_mileage"`
	EndMileage   float64   `json:"end_mileage" db:"end_mileage"`
	Attendance   string    `json:"attendance" db:"attendance"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

// Replace the ECSEStudent struct in models.go with this updated version

// ECSEStudent represents a special education student
type ECSEStudent struct {
	StudentID              string    `json:"student_id" db:"student_id"`
	FirstName              string    `json:"first_name" db:"first_name"`
	LastName               string    `json:"last_name" db:"last_name"`
	DateOfBirth            string    `json:"date_of_birth" db:"date_of_birth"`
	Grade                  string    `json:"grade" db:"grade"`
	EnrollmentStatus       string    `json:"enrollment_status" db:"enrollment_status"`
	IEPStatus              string    `json:"iep_status" db:"iep_status"`
	PrimaryDisability      string    `json:"primary_disability" db:"primary_disability"`
	ServiceMinutes         int       `json:"service_minutes" db:"service_minutes"`
	TransportationRequired bool      `json:"transportation_required" db:"transportation_required"`
	BusRoute               string    `json:"bus_route" db:"bus_route"`
	ParentName             string    `json:"parent_name" db:"parent_name"`
	ParentPhone            string    `json:"parent_phone" db:"parent_phone"`
	ParentEmail            string    `json:"parent_email" db:"parent_email"`
	City                   string    `json:"city" db:"city"`
	State                  string    `json:"state" db:"state"`
	ZipCode                string    `json:"zip_code" db:"zip_code"`
	Notes                  string    `json:"notes" db:"notes"`
	CreatedAt              time.Time `json:"created_at" db:"created_at"`
}


// MileageReport represents imported mileage data
type MileageReport struct {
	ID          int       `json:"id" db:"id"`
	Unit        string    `json:"unit" db:"unit"`
	VehicleNo   string    `json:"vehicle_no" db:"vehicle_no"`
	Driver      string    `json:"driver" db:"driver"`
	Month       string    `json:"month" db:"month"`
	Year        int       `json:"year" db:"year"`
	BeginMiles  int       `json:"begin_miles" db:"begin_miles"`
	EndMiles    int       `json:"end_miles" db:"end_miles"`
	TotalMiles  int       `json:"total_miles" db:"total_miles"`
	DailyMiles  string    `json:"daily_miles" db:"daily_miles"`
	Utilization float64   `json:"utilization" db:"utilization"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// Import-specific structs for mileage reports
// AgencyVehicleRecord represents a record from the Agency Vehicle Report
type AgencyVehicleRecord struct {
	ReportMonth    string
	ReportYear     int
	VehicleYear    int
	MakeModel      string
	LicensePlate   string
	VehicleID      string
	Location       string
	BeginningMiles int
	EndingMiles    int
	TotalMiles     int
	Status         string
	Notes          string
}

// SchoolBusRecord represents a record from the School Bus Report
type SchoolBusRecord struct {
	ReportMonth    string
	ReportYear     int
	BusYear        int
	BusMake        string
	LicensePlate   string
	BusID          string
	Location       string
	BeginningMiles int
	EndingMiles    int
	TotalMiles     int
	Status         string
	Notes          string
}

// ProgramStaffRecord represents a record from the Program Staff Report
type ProgramStaffRecord struct {
	ReportMonth string
	ReportYear  int
	ProgramType string
	StaffCount1 int
	StaffCount2 int
}

// ECSEService represents services provided to ECSE students
type ECSEService struct {
	ID           int       `json:"id" db:"id"`
	StudentID    string    `json:"student_id" db:"student_id"`
	ServiceType  string    `json:"service_type" db:"service_type"`
	Frequency    string    `json:"frequency" db:"frequency"`
	Duration     int       `json:"duration" db:"duration"`
	Provider     string    `json:"provider" db:"provider"`
	StartDate    time.Time `json:"start_date" db:"start_date"`
	EndDate      time.Time `json:"end_date" db:"end_date"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

// ECSEAssessment represents assessment data for ECSE students
type ECSEAssessment struct {
	ID                  int       `json:"id" db:"id"`
	StudentID           string    `json:"student_id" db:"student_id"`
	AssessmentDate      time.Time `json:"assessment_date" db:"assessment_date"`
	AssessmentType      string    `json:"assessment_type" db:"assessment_type"`
	Results             string    `json:"results" db:"results"`
	Evaluator           string    `json:"evaluator" db:"evaluator"`
	NextAssessmentDate  time.Time `json:"next_assessment_date" db:"next_assessment_date"`
	CreatedAt           time.Time `json:"created_at" db:"created_at"`
}

// ECSEAttendance represents attendance records for ECSE students
type ECSEAttendance struct {
	ID            int       `json:"id" db:"id"`
	StudentID     string    `json:"student_id" db:"student_id"`
	Date          time.Time `json:"date" db:"date"`
	Status        string    `json:"status" db:"status"`
	ArrivalTime   string    `json:"arrival_time" db:"arrival_time"`
	DepartureTime string    `json:"departure_time" db:"departure_time"`
	Notes         string    `json:"notes" db:"notes"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
}

// ECSE-specific view structs
type ECSEStudentView struct {
	StudentID              string    `json:"student_id" db:"student_id"`
	FirstName              string    `json:"first_name" db:"first_name"`
	LastName               string    `json:"last_name" db:"last_name"`
	DateOfBirth            string    `json:"date_of_birth" db:"date_of_birth"`
	Grade                  string    `json:"grade" db:"grade"`
	EnrollmentStatus       string    `json:"enrollment_status" db:"enrollment_status"`
	IEPStatus              string    `json:"iep_status" db:"iep_status"`
	PrimaryDisability      string    `json:"primary_disability" db:"primary_disability"`
	ServiceMinutes         int       `json:"service_minutes" db:"service_minutes"`
	TransportationRequired bool      `json:"transportation_required" db:"transportation_required"`
	BusRoute               string    `json:"bus_route" db:"bus_route"`
	ParentName             string    `json:"parent_name" db:"parent_name"`
	ParentPhone            string    `json:"parent_phone" db:"parent_phone"`
	ParentEmail            string    `json:"parent_email" db:"parent_email"`
	City                   string    `json:"city" db:"city"`
	State                  string    `json:"state" db:"state"`
	ZipCode                string    `json:"zip_code" db:"zip_code"`
	Notes                  string    `json:"notes" db:"notes"`
	CreatedAt              time.Time `json:"created_at" db:"created_at"`
}
