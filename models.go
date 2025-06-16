// models.go - All data structures for the bus transportation app
// This file can be created without breaking anything in main.go
package main

// User represents a system user (driver or manager)
type User struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

// Attendance tracks student attendance
type Attendance struct {
	Date    string `json:"date"`
	Driver  string `json:"driver"`
	Route   string `json:"route"`
	Present int    `json:"present"`
}

// Mileage tracks route mileage
type Mileage struct {
	Date   string  `json:"date"`
	Driver string  `json:"driver"`
	Route  string  `json:"route"`
	Miles  float64 `json:"miles"`
}

// Activity represents a special trip or activity
type Activity struct {
	Date       string  `json:"date"`
	Driver     string  `json:"driver"`
	TripName   string  `json:"trip_name"`
	Attendance int     `json:"attendance"`
	Miles      float64 `json:"miles"`
	Notes      string  `json:"notes"`
}

// DriverSummary contains aggregated driver statistics
type DriverSummary struct {
	Name              string
	TotalMorning      int
	TotalEvening      int
	TotalMiles        float64
	MonthlyAvgMiles   float64
	MonthlyAttendance int
}

// RouteStats contains route statistics
type RouteStats struct {
	RouteName       string
	TotalMiles      float64
	AvgMiles        float64
	AttendanceDay   int
	AttendanceWeek  int
	AttendanceMonth int
}

// Route represents a bus route
type Route struct {
	RouteID     string `json:"route_id"`
	RouteName   string `json:"route_name"`
	Description string `json:"description"`
	Positions   []struct {
		Position int    `json:"position"`
		Student  string `json:"student"`
	} `json:"positions"`
}

// Bus represents a school bus
type Bus struct {
	BusID            string `json:"bus_id"`
	Status           string `json:"status"` // active, maintenance, out_of_service
	Model            string `json:"model"`
	Capacity         int    `json:"capacity"`
	OilStatus        string `json:"oil_status"`        // good, due, overdue
	TireStatus       string `json:"tire_status"`       // good, worn, replace
	MaintenanceNotes string `json:"maintenance_notes"`
}

// Vehicle represents a company vehicle
type Vehicle struct {
	VehicleID        string `json:"vehicle_id"`
	Model            string `json:"model"`
	Description      string `json:"description"`
	Year             string `json:"year"`
	TireSize         string `json:"tire_size"`
	License          string `json:"license"`
	OilStatus        string `json:"oil_status"`
	TireStatus       string `json:"tire_status"`
	Status           string `json:"status"`
	MaintenanceNotes string `json:"maintenance_notes"`
	SerialNumber     string `json:"serial_number"`
	Base             string `json:"base"`
	ServiceInterval  int    `json:"service_interval"`
}

// Student represents a student rider
type Student struct {
	StudentID       string     `json:"student_id"`
	Name            string     `json:"name"`
	Locations       []Location `json:"locations"`
	PhoneNumber     string     `json:"phone_number"`
	AltPhoneNumber  string     `json:"alt_phone_number"`
	Guardian        string     `json:"guardian"`
	PickupTime      string     `json:"pickup_time"`
	DropoffTime     string     `json:"dropoff_time"`
	PositionNumber  int        `json:"position_number"`
	RouteID         string     `json:"route_id"`
	Driver          string     `json:"driver"`
	Active          bool       `json:"active"`
}

// Location represents a pickup or dropoff location
type Location struct {
	Type        string `json:"type"` // "pickup" or "dropoff"
	Address     string `json:"address"`
	Description string `json:"description"`
}

// RouteAssignment links drivers to routes and buses
type RouteAssignment struct {
	Driver       string `json:"driver"`
	BusID        string `json:"bus_id"`
	RouteID      string `json:"route_id"`
	RouteName    string `json:"route_name"`
	AssignedDate string `json:"assigned_date"`
}

// DriverLog represents a completed route log
type DriverLog struct {
	Driver     string  `json:"driver"`
	BusID      string  `json:"bus_id"`
	RouteID    string  `json:"route_id"`
	Date       string  `json:"date"`
	Period     string  `json:"period"` // morning or evening
	Departure  string  `json:"departure_time"`
	Arrival    string  `json:"arrival_time"`
	Mileage    float64 `json:"mileage"`
	Attendance []struct {
		Position   int    `json:"position"`
		Present    bool   `json:"present"`
		PickupTime string `json:"pickup_time,omitempty"`
	} `json:"attendance"`
}

// MaintenanceLog tracks vehicle maintenance
type MaintenanceLog struct {
	BusID    string `json:"bus_id"`
	Date     string `json:"date"`      // YYYY-MM-DD
	Category string `json:"category"`  // oil, tires, brakes, etc.
	Notes    string `json:"notes"`
	Mileage  int    `json:"mileage"`   // optional
}

// DashboardData is used for the manager dashboard template
type DashboardData struct {
	User            *User
	Role            string
	DriverSummaries []*DriverSummary
	RouteStats      []*RouteStats
	Activities      []Activity
	Routes          []Route
	Users           []User
	Buses           []*Bus
}

// AssignRouteData is used for the route assignment page
type AssignRouteData struct {
	User            *User
	Assignments     []RouteAssignment
	Drivers         []User
	AvailableRoutes []Route
	AvailableBuses  []*Bus
}

// FleetData is used for the fleet management page
type FleetData struct {
	User  *User
	Buses []*Bus
	Today string
}

// StudentData is used for the student management page
type StudentData struct {
	User     *User
	Students []Student
	Routes   []Route
}

// CompanyFleetData is used for the company fleet page
type CompanyFleetData struct {
	User     *User
	Vehicles []Vehicle
}
