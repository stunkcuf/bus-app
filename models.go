// models.go - All data structures for the bus transportation app with proper JSON tags
package main

// User represents a system user (driver or manager)
type User struct {
	Username         string `json:"username"`
	Password         string `json:"password"` // This will store bcrypt hash
	Role             string `json:"role"`     // driver, manager, or driver_pending
	Status           string `json:"status"`           // active, pending, or suspended
	RegistrationDate string `json:"registration_date"` // When the user registered
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
	Name              string  `json:"name"`
	TotalMorning      int     `json:"total_morning"`
	TotalEvening      int     `json:"total_evening"`
	TotalMiles        float64 `json:"total_miles"`
	MonthlyAvgMiles   float64 `json:"monthly_avg_miles"`
	MonthlyAttendance int     `json:"monthly_attendance"`
}

// RouteStats contains route statistics
type RouteStats struct {
	RouteName       string  `json:"route_name"`
	TotalMiles      float64 `json:"total_miles"`
	AvgMiles        float64 `json:"avg_miles"`
	AttendanceDay   int     `json:"attendance_day"`
	AttendanceWeek  int     `json:"attendance_week"`
	AttendanceMonth int     `json:"attendance_month"`
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

// Vehicle represents a company vehicle (ONLY ONE DECLARATION)
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
	Period     string  `json:"period"` // morning, afternoon, or evening
	Departure  string  `json:"departure_time"`
	Arrival    string  `json:"arrival_time"`
	Mileage    float64 `json:"mileage"`
	Attendance []struct {
		Position   int    `json:"position"`
		Present    bool   `json:"present"`
		PickupTime string `json:"pickup_time,omitempty"`
	} `json:"attendance"`
}

// MaintenanceLog for database - matches your fleet database schema
type MaintenanceLog struct {
	ID            int      `json:"id" db:"id"`
	VehicleNumber int      `json:"vehicle_number" db:"vehicle_number"`
	ServiceDate   string   `json:"service_date" db:"maintenance_date"`
	Mileage       *int     `json:"mileage" db:"mileage"`
	PONumber      *string  `json:"po_number" db:"po_number"`
	Cost          *float64 `json:"cost" db:"cost"`
	WorkDone      string   `json:"work_done" db:"work_done"`
	CreatedAt     string   `json:"created_at" db:"created_at"`
}

// BusMaintenanceLog for bus-specific maintenance (JSON files)
type BusMaintenanceLog struct {
	BusID    string `json:"bus_id"`
	Date     string `json:"date"`
	Category string `json:"category"`
	Notes    string `json:"notes"`
	Mileage  int    `json:"mileage"`
}

// VehicleWithStats for displaying vehicle with maintenance count
type VehicleWithStats struct {
	Vehicle
	MaintenanceCount int     `json:"maintenance_count"`
	TotalCost        float64 `json:"total_cost"`
	LastService      *string `json:"last_service"`
}

// FleetStats contains overall fleet statistics
type FleetStats struct {
	TotalVehicles           int     `json:"total_vehicles"`
	TotalMaintenanceRecords int     `json:"total_maintenance_records"`
	TotalMaintenanceCost    float64 `json:"total_maintenance_cost"`
	AverageMaintenanceCost  float64 `json:"average_maintenance_cost"`
	YearRange               string  `json:"year_range"`
	UniqueMakes             int     `json:"unique_makes"`
}

// DashboardData is used for the manager dashboard template
type DashboardData struct {
	User            *User            `json:"user"`
	Role            string           `json:"role"`
	DriverSummaries []*DriverSummary `json:"driver_summaries"`
	RouteStats      []*RouteStats    `json:"route_stats"`
	Activities      []Activity       `json:"activities"`
	Routes          []Route          `json:"routes"`
	Users           []User           `json:"users"`
	Buses           []*Bus           `json:"buses"`
	CSRFToken       string           `json:"csrf_token"` // Added for CSRF protection
	PendingUsers    int              `json:"pending_users"` // Added for pending user count
}

// AssignRouteData is used for the route assignment page
type AssignRouteData struct {
	User            *User             `json:"user"`
	Assignments     []RouteAssignment `json:"assignments"`
	Drivers         []User            `json:"drivers"`
	AvailableRoutes []Route           `json:"available_routes"`
	AvailableBuses  []*Bus            `json:"available_buses"`
	CSRFToken       string            `json:"csrf_token"` // Added for CSRF protection
}

// FleetData is used for the fleet management page
type FleetData struct {
	User      *User  `json:"user"`
	Buses     []*Bus `json:"buses"`
	Today     string `json:"today"`
	CSRFToken string `json:"csrf_token"` // Added for CSRF protection
}

// StudentData is used for the student management page
type StudentData struct {
	User      *User     `json:"user"`
	Students  []Student `json:"students"`
	Routes    []Route   `json:"routes"`
	CSRFToken string    `json:"csrf_token"` // Added for CSRF protection
}

// CompanyFleetData is used for the company fleet page
type CompanyFleetData struct {
	User      *User     `json:"user"`
	Vehicles  []Vehicle `json:"vehicles"`
	CSRFToken string    `json:"csrf_token"` // Added for CSRF protection
}

// DriverDashboardData is used for the driver dashboard template
type DriverDashboardData struct {
	User       *User       `json:"user"`
	Date       string      `json:"date"`
	Period     string      `json:"period"`
	Route      *Route      `json:"route"`
	DriverLog  *DriverLog  `json:"driver_log"`
	Bus        *Bus        `json:"bus"`
	RecentLogs []DriverLog `json:"recent_logs"`
	CSRFToken  string      `json:"csrf_token"` // Added for CSRF protection
}

// LoginFormData is used for the login page template
type LoginFormData struct {
	Error     string `json:"error,omitempty"`
	CSRFToken string `json:"csrf_token"`
}

// UserFormData is used for user creation/edit forms
type UserFormData struct {
	User      *User  `json:"user,omitempty"`
	Error     string `json:"error,omitempty"`
	CSRFToken string `json:"csrf_token"`
}
