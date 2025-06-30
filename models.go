package models

import (
	"database/sql/driver"
	"time"
)

// Vehicle represents a vehicle in the fleet (both buses and other vehicles)
type Vehicle struct {
	VehicleID        string `json:"vehicle_id" db:"vehicle_id"`           // Maps to vehicle_id column
	VehicleNumber    string `json:"vehicle_number" db:"vehicle_number"`   // Maps to vehicle_number column
	Unnamed1         string `json:"unnamed_1" db:"unnamed_1"`             // Maps to unnamed_1 column
	Model            string `json:"model" db:"model"`
	Description      string `json:"description" db:"description"`
	Year             string `json:"year" db:"year"`
	TireSize         string `json:"tire_size" db:"tire_size"`
	License          string `json:"license" db:"license"`
	OilStatus        string `json:"oil_status" db:"oil_status"`
	TireStatus       string `json:"tire_status" db:"tire_status"`
	Status           string `json:"status" db:"status"`
	MaintenanceNotes string `json:"maintenance_notes" db:"maintenance_notes"`
	SerialNumber     string `json:"serial_number" db:"serial_number"`
	Base             string `json:"base" db:"base"`
	ServiceInterval  int    `json:"service_interval" db:"service_interval"`
	
	// Additional fields for display/logic
	IsBus              bool                  `json:"is_bus" db:"is_bus"`
	MaintenanceHistory []MaintenanceRecord   `json:"maintenance_history"`
	ActiveIssues       []Issue               `json:"active_issues"`
}

// Bus represents a bus in the fleet
type Bus struct {
	BusID            string `json:"bus_id" db:"bus_id"`
	Model            string `json:"model" db:"model"`
	Capacity         int    `json:"capacity" db:"capacity"`
	Status           string `json:"status" db:"status"` // active, maintenance, out_of_service
	OilStatus        string `json:"oil_status" db:"oil_status"` // good, due, overdue
	TireStatus       string `json:"tire_status" db:"tire_status"` // good, worn, replace
	MaintenanceNotes string `json:"maintenance_notes" db:"maintenance_notes"`
}

// MaintenanceRecord represents a maintenance history entry
type MaintenanceRecord struct {
	ID          int       `json:"id" db:"id"`
	VehicleID   string    `json:"vehicle_id" db:"vehicle_id"`
	BusID       string    `json:"bus_id" db:"bus_id"`
	Date        string    `json:"date" db:"date"`
	Type        string    `json:"type" db:"type"` // Routine, Repair, Inspection, Emergency
	Category    string    `json:"category" db:"category"` // oil_change, tire_service, inspection, repair, other
	Description string    `json:"description" db:"description"`
	Cost        float64   `json:"cost" db:"cost"`
	Mileage     int       `json:"mileage" db:"mileage"`
	Notes       string    `json:"notes" db:"notes"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// Issue represents an active issue with a vehicle
type Issue struct {
	ID           string    `json:"id" db:"id"`
	VehicleID    string    `json:"vehicle_id" db:"vehicle_id"`
	Description  string    `json:"description" db:"description"`
	Severity     string    `json:"severity" db:"severity"` // High, Medium, Low
	ReportedDate string    `json:"reported_date" db:"reported_date"`
	Status       string    `json:"status" db:"status"` // active, resolved
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

// User represents a system user
type User struct {
	ID        int       `json:"id" db:"id"`
	Username  string    `json:"username" db:"username"`
	Password  string    `json:"-" db:"password"` // Never send password in JSON
	Role      string    `json:"role" db:"role"`  // manager, driver, driver_pending
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// Route represents a bus route
type Route struct {
	RouteID     string `json:"route_id" db:"route_id"`
	RouteName   string `json:"route_name" db:"route_name"`
	Description string `json:"description" db:"description"`
	IsAssigned  bool   `json:"is_assigned"` // Calculated field
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// RouteWithStatus includes assignment status
type RouteWithStatus struct {
	Route
	IsAssigned bool `json:"is_assigned"`
}

// Assignment represents a driver-bus-route assignment
type Assignment struct {
	ID        int       `json:"id" db:"id"`
	Driver    string    `json:"driver" db:"driver"`
	BusID     string    `json:"bus_id" db:"bus_id"`
	RouteID   string    `json:"route_id" db:"route_id"`
	RouteName string    `json:"route_name"` // Joined from routes table
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// Student represents a student in the system
type Student struct {
	StudentID      int        `json:"student_id" db:"student_id"`
	Name           string     `json:"name" db:"name"`
	Guardian       string     `json:"guardian" db:"guardian"`
	PhoneNumber    string     `json:"phone_number" db:"phone_number"`
	AltPhoneNumber string     `json:"alt_phone_number" db:"alt_phone_number"`
	PickupTime     string     `json:"pickup_time" db:"pickup_time"`
	DropoffTime    string     `json:"dropoff_time" db:"dropoff_time"`
	RouteID        string     `json:"route_id" db:"route_id"`
	PositionNumber int        `json:"position_number" db:"position_number"`
	Active         bool       `json:"active" db:"active"`
	Locations      []Location `json:"locations"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
}

// Location represents a pickup/dropoff location
type Location struct {
	ID          int    `json:"id" db:"id"`
	StudentID   int    `json:"student_id" db:"student_id"`
	Type        string `json:"type" db:"type"` // pickup, dropoff
	Address     string `json:"address" db:"address"`
	Description string `json:"description" db:"description"`
}

// DriverLog represents a driver's trip log
type DriverLog struct {
	ID         int                `json:"id" db:"id"`
	Driver     string             `json:"driver" db:"driver"`
	Date       string             `json:"date" db:"date"`
	Period     string             `json:"period" db:"period"` // morning, afternoon
	RouteID    string             `json:"route_id" db:"route_id"`
	BusID      string             `json:"bus_id" db:"bus_id"`
	Departure  string             `json:"departure" db:"departure"`
	Arrival    string             `json:"arrival" db:"arrival"`
	Mileage    float64            `json:"mileage" db:"mileage"`
	Attendance []StudentAttendance `json:"attendance"`
	CreatedAt  time.Time          `json:"created_at" db:"created_at"`
}

// StudentAttendance represents attendance for a student on a trip
type StudentAttendance struct {
	Position   int    `json:"position" db:"position"`
	Present    bool   `json:"present" db:"present"`
	PickupTime string `json:"pickup_time" db:"pickup_time"`
}

// PageData represents common page data passed to templates
type PageData struct {
	User      *User       `json:"user"`
	CSRFToken string      `json:"csrf_token"`
	Error     string      `json:"error,omitempty"`
	Success   string      `json:"success,omitempty"`
	Data      interface{} `json:"data,omitempty"`
}

// DashboardData represents data for the manager dashboard
type DashboardData struct {
	PageData
	Users        []User       `json:"users"`
	PendingUsers int          `json:"pending_users"`
	Buses        []Bus        `json:"buses"`
	Routes       []Route      `json:"routes"`
	Assignments  []Assignment `json:"assignments"`
}

// FleetPageData represents data for the fleet management page
type FleetPageData struct {
	PageData
	Buses              []Bus               `json:"buses"`
	MaintenanceRecords []MaintenanceRecord `json:"maintenance_records"`
	Today              string              `json:"today"`
}

// VehicleMaintenanceData represents data for the vehicle maintenance page
type VehicleMaintenanceData struct {
	PageData
	Vehicle
	TotalRecords  int     `json:"total_records"`
	TotalCost     float64 `json:"total_cost"`
	AverageCost   float64 `json:"average_cost"`
	RecentCount   int     `json:"recent_count"`
	Today         string  `json:"today"`
}

// CompanyFleetData represents data for the company fleet page
type CompanyFleetData struct {
	PageData
	Vehicles              []Vehicle `json:"vehicles"`
	AvailableDriversCount int       `json:"available_drivers_count"`
	AvailableBusesCount   int       `json:"available_buses_count"`
}

// RouteAssignmentData represents data for the route assignment page
type RouteAssignmentData struct {
	PageData
	Routes                []Route             `json:"routes"`
	RoutesWithStatus      []RouteWithStatus   `json:"routes_with_status"`
	AvailableRoutes       []Route             `json:"available_routes"`
	Drivers               []User              `json:"drivers"`
	AvailableBuses        []Bus               `json:"available_buses"`
	Assignments           []Assignment        `json:"assignments"`
	TotalAssignments      int                 `json:"total_assignments"`
	TotalRoutes           int                 `json:"total_routes"`
	AvailableDriversCount int                 `json:"available_drivers_count"`
	AvailableBusesCount   int                 `json:"available_buses_count"`
}

// DriverDashboardData represents data for the driver dashboard
type DriverDashboardData struct {
	PageData
	Route       *Route      `json:"route"`
	Bus         *Bus        `json:"bus"`
	Students    []Student   `json:"students"`
	Date        string      `json:"date"`
	Period      string      `json:"period"`
	DriverLog   *DriverLog  `json:"driver_log"`
	RecentLogs  []DriverLog `json:"recent_logs"`
}

// StudentManagementData represents data for the student management page
type StudentManagementData struct {
	PageData
	Students []Student `json:"students"`
	Routes   []Route   `json:"routes"`
}

// NullString handles nullable string fields
type NullString struct {
	String string
	Valid  bool
}

// Scan implements the Scanner interface
func (ns *NullString) Scan(value interface{}) error {
	if value == nil {
		ns.String, ns.Valid = "", false
		return nil
	}
	ns.Valid = true
	ns.String = value.(string)
	return nil
}

// Value implements the driver Valuer interface
func (ns NullString) Value() (driver.Value, error) {
	if !ns.Valid {
		return nil, nil
	}
	return ns.String, nil
}

// Helper methods for Vehicle
func (v *Vehicle) GetDisplayID() string {
	if v.VehicleNumber != "" {
		return v.VehicleNumber
	}
	if v.VehicleID != "" {
		return v.VehicleID
	}
	return v.Unnamed1
}

func (v *Vehicle) GetIdentifier() string {
	// Returns the first non-empty identifier
	if v.VehicleID != "" {
		return v.VehicleID
	}
	if v.VehicleNumber != "" {
		return v.VehicleNumber
	}
	return v.Unnamed1
}

// Status constants
const (
	StatusActive       = "active"
	StatusMaintenance  = "maintenance"
	StatusOutOfService = "out_of_service"
	
	OilStatusGood        = "good"
	OilStatusNeedsService = "needs_service"
	OilStatusOverdue     = "overdue"
	
	TireStatusGood    = "good"
	TireStatusWorn    = "worn"
	TireStatusReplace = "replace"
	
	RoleManager       = "manager"
	RoleDriver        = "driver"
	RoleDriverPending = "driver_pending"
)
