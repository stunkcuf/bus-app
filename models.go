package main

import (
	"time"
)

// User represents a system user (driver or manager)
type User struct {
	Username         string `json:"username" db:"username"`
	Password         string `json:"password,omitempty" db:"password"`
	Role             string `json:"role" db:"role"`
	Status           string `json:"status" db:"status"`
	RegistrationDate string `json:"registration_date" db:"registration_date"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
}

// Bus represents a school bus
type Bus struct {
	BusID            string `json:"bus_id" db:"bus_id"`
	Status           string `json:"status" db:"status"`
	Model            string `json:"model" db:"model"`
	Capacity         int    `json:"capacity" db:"capacity"`
	OilStatus        string `json:"oil_status" db:"oil_status"`
	TireStatus       string `json:"tire_status" db:"tire_status"`
	MaintenanceNotes string `json:"maintenance_notes" db:"maintenance_notes"`
}

// Route represents a bus route
type Route struct {
	RouteID     string `json:"route_id" db:"route_id"`
	RouteName   string `json:"route_name" db:"route_name"`
	Description string `json:"description" db:"description"`
	Positions   []struct {
		Position int    `json:"position"`
		Student  string `json:"student"`
	} `json:"positions" db:"positions"`
}

// Student represents a student on a route
type Student struct {
	StudentID      string     `json:"student_id" db:"student_id"`
	Name           string     `json:"name" db:"name"`
	Locations      []Location `json:"locations" db:"locations"`
	PhoneNumber    string     `json:"phone_number" db:"phone_number"`
	AltPhoneNumber string     `json:"alt_phone_number" db:"alt_phone_number"`
	Guardian       string     `json:"guardian" db:"guardian"`
	PickupTime     string     `json:"pickup_time" db:"pickup_time"`
	DropoffTime    string     `json:"dropoff_time" db:"dropoff_time"`
	PositionNumber int        `json:"position_number" db:"position_number"`
	RouteID        string     `json:"route_id" db:"route_id"`
	Driver         string     `json:"driver" db:"driver"`
	Active         bool       `json:"active" db:"active"`
}

// Location represents a pickup/dropoff location
type Location struct {
	LocationID  string `json:"location_id" db:"location_id"`
	Type        string `json:"type"`        // "pickup" or "dropoff"
	Address     string `json:"address"`
	Description string `json:"description"`
}

// RouteAssignment represents driver-bus-route assignment
type RouteAssignment struct {
	Driver       string `json:"driver" db:"driver"`
	BusID        string `json:"bus_id" db:"bus_id"`
	RouteID      string `json:"route_id" db:"route_id"`
	RouteName    string `json:"route_name" db:"route_name"`
	AssignedDate string `json:"assigned_date" db:"assigned_date"`
}

// DriverAssignment represents driver-route-bus assignment details
type DriverAssignment struct {
	Driver    string `json:"driver" db:"driver"`
	RouteID   string `json:"route_id" db:"route_id"`
	BusID     string `json:"bus_id" db:"bus_id"`
	RouteName string `json:"route_name" db:"route_name"`
}

// DriverLog represents a driver's daily log
type DriverLog struct {
	ID         int         `json:"id" db:"id"`
	Driver     string      `json:"driver" db:"driver"`
	BusID      string      `json:"bus_id" db:"bus_id"`
	RouteID    string      `json:"route_id" db:"route_id"`
	Date       string      `json:"date" db:"date"`
	Period     string      `json:"period" db:"period"` // "morning" or "afternoon"
	Departure  string      `json:"departure" db:"departure_time"`
	Arrival    string      `json:"arrival" db:"arrival_time"`
	Mileage    float64     `json:"mileage" db:"mileage"`
	Attendance interface{} `json:"attendance" db:"attendance"`
}

// BusMaintenanceLog represents a maintenance record
type BusMaintenanceLog struct {
	ID              int       `json:"id" db:"id"`
	BusID           string    `json:"bus_id" db:"bus_id"`
	VehicleID       string    `json:"vehicle_id,omitempty" db:"vehicle_id"`
	Date            string    `json:"date" db:"date"`
	Type            string    `json:"type,omitempty" db:"type"`
	Category        string    `json:"category" db:"category"`
	Description     string    `json:"description,omitempty" db:"description"`
	Notes           string    `json:"notes" db:"notes"`
	Cost            float64   `json:"cost,omitempty" db:"cost"`
	Mileage         int       `json:"mileage" db:"mileage"`
	MaintenanceType string    `json:"maintenance_type,omitempty" db:"maintenance_type"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
}

// MaintenanceRecord represents a vehicle maintenance log entry
type MaintenanceRecord struct {
	VehicleID string    `json:"vehicle_id" db:"vehicle_id"`
	Date      string    `json:"date" db:"date"`
	Category  string    `json:"category" db:"category"`
	Mileage   int       `json:"mileage" db:"mileage"`
	Cost      float64   `json:"cost" db:"cost"`
	Notes     string    `json:"notes" db:"notes"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// Vehicle represents a company vehicle (non-bus)
type Vehicle struct {
	VehicleID        string    `json:"vehicle_id" db:"vehicle_id"`
	BusID            string    `json:"bus_id,omitempty" db:"bus_id"`
	Model            string    `json:"model" db:"model"`
	Description      string    `json:"description" db:"description"`
	Year             int       `json:"year" db:"year"`
	TireSize         string    `json:"tire_size" db:"tire_size"`
	License          string    `json:"license" db:"license"`
	OilStatus        string    `json:"oil_status" db:"oil_status"`
	TireStatus       string    `json:"tire_status" db:"tire_status"`
	Status           string    `json:"status" db:"status"`
	MaintenanceNotes string    `json:"maintenance_notes" db:"maintenance_notes"`
	SerialNumber     string    `json:"serial_number" db:"serial_number"`
	Base             string    `json:"base" db:"base"`
	ServiceInterval  int       `json:"service_interval" db:"service_interval"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
}

// Activity represents a special activity or trip
type Activity struct {
	Date       string  `json:"date" db:"date"`
	Driver     string  `json:"driver" db:"driver"`
	TripName   string  `json:"trip_name" db:"trip_name"`
	Attendance int     `json:"attendance" db:"attendance"`
	Miles      float64 `json:"miles" db:"miles"`
	Notes      string  `json:"notes" db:"notes"`
}

// RouteLog represents a driver's daily route log
type RouteLog struct {
	ID         int                 `json:"id" db:"id"`
	Driver     string              `json:"driver" db:"driver"`
	Date       string              `json:"date" db:"date"`
	Period     string              `json:"period" db:"period"`
	RouteID    string              `json:"route_id" db:"route_id"`
	BusID      string              `json:"bus_id" db:"bus_id"`
	Mileage    float64             `json:"mileage" db:"mileage"`
	Departure  string              `json:"departure" db:"departure"`
	Arrival    string              `json:"arrival" db:"arrival"`
	Attendance []StudentAttendance `json:"attendance" db:"attendance"`
}

// StudentAttendance represents student attendance on a route
type StudentAttendance struct {
	Position   int    `json:"position" db:"position"`
	Present    bool   `json:"present" db:"present"`
	PickupTime string `json:"pickup_time,omitempty" db:"pickup_time"`
}

// MileageReport represents a monthly mileage report for a vehicle
type MileageReport struct {
	ReportMonth    string `json:"report_month" db:"report_month"`
	ReportYear     int    `json:"report_year" db:"report_year"`
	VehicleYear    int    `json:"vehicle_year" db:"vehicle_year"`
	MakeModel      string `json:"make_model" db:"make_model"`
	LicensePlate   string `json:"license_plate" db:"license_plate"`
	VehicleID      string `json:"vehicle_id" db:"vehicle_id"`
	Location       string `json:"location" db:"location"`
	BeginningMiles int    `json:"beginning_miles" db:"beginning_miles"`
	EndingMiles    int    `json:"ending_miles" db:"ending_miles"`
	TotalMiles     int    `json:"total_miles" db:"total_miles"`
	Status         string `json:"status" db:"status"`
}

// Mileage reporting structures
type AgencyVehicleRecord struct {
	ReportMonth    string `json:"report_month" db:"report_month"`
	ReportYear     int    `json:"report_year" db:"report_year"`
	VehicleYear    int    `json:"vehicle_year" db:"vehicle_year"`
	MakeModel      string `json:"make_model" db:"make_model"`
	LicensePlate   string `json:"license_plate" db:"license_plate"`
	VehicleID      string `json:"vehicle_id" db:"vehicle_id"`
	Location       string `json:"location" db:"location"`
	BeginningMiles int    `json:"beginning_miles" db:"beginning_miles"`
	EndingMiles    int    `json:"ending_miles" db:"ending_miles"`
	TotalMiles     int    `json:"total_miles" db:"total_miles"`
	Status         string `json:"status" db:"status"`
	Notes          string `json:"notes" db:"notes"`
}

type SchoolBusRecord struct {
	ReportMonth    string `json:"report_month" db:"report_month"`
	ReportYear     int    `json:"report_year" db:"report_year"`
	BusYear        int    `json:"bus_year" db:"bus_year"`
	BusMake        string `json:"bus_make" db:"bus_make"`
	LicensePlate   string `json:"license_plate" db:"license_plate"`
	BusID          string `json:"bus_id" db:"bus_id"`
	Location       string `json:"location" db:"location"`
	BeginningMiles int    `json:"beginning_miles" db:"beginning_miles"`
	EndingMiles    int    `json:"ending_miles" db:"ending_miles"`
	TotalMiles     int    `json:"total_miles" db:"total_miles"`
	Status         string `json:"status" db:"status"`
	Notes          string `json:"notes" db:"notes"`
}

type ProgramStaffRecord struct {
	ReportMonth  string `json:"report_month" db:"report_month"`
	ReportYear   int    `json:"report_year" db:"report_year"`
	ProgramType  string `json:"program_type" db:"program_type"` // "HS", "OPK", or "EHS"
	StaffCount1  int    `json:"staff_count_1" db:"staff_count_1"`
	StaffCount2  int    `json:"staff_count_2" db:"staff_count_2"`
}

// Template data structures

type DashboardData struct {
	User            *User
	Role            string
	Users           []User
	Buses           []*Bus
	Routes          []Route
	DriverSummaries []*DriverSummary
	RouteStats      []*RouteStats
	Activities      []Activity
	CSRFToken       string
	PendingUsers    int
}

type DriverSummary struct {
	Driver       string
	BusID        string
	RouteID      string
	RouteName    string
	LastActivity time.Time
	TotalMiles   float64
}

type RouteStats struct {
	RouteID      string
	RouteName    string
	ActiveBuses  int
	TotalStudents int
}

type FleetData struct {
	User             *User
	Buses            []*Bus
	Today            string
	CSRFToken        string
	MaintenanceLogs  []BusMaintenanceLog
}

type CompanyFleetData struct {
	User      *User
	Vehicles  []Vehicle
	CSRFToken string
}

type StudentData struct {
	User      *User
	Students  []Student
	Routes    []Route
	CSRFToken string
}

type AssignRouteData struct {
	User                  *User
	Assignments           []RouteAssignment
	Drivers               []User
	AvailableRoutes       []Route
	AvailableBuses        []*Bus
	RoutesWithStatus      []struct {
		Route
		IsAssigned bool `json:"is_assigned"`
	}
	TotalAssignments      int
	TotalRoutes           int
	AvailableDriversCount int
	AvailableBusesCount   int
	CSRFToken             string
}

type UserFormData struct {
	Username  string
	Role      string
	CSRFToken string
	Error     string
}

type LoginFormData struct {
	Error     string
	CSRFToken string
}

type RouteWithStatus struct {
	Route
	IsAssigned bool `json:"is_assigned"`
}
