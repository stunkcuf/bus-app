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

// Vehicle represents a company vehicle (UPDATED WITH POSTGRESQL COLUMNS)
type Vehicle struct {
	VehicleID        string `json:"vehicle_id" db:"vehicle_id"`           // Maps to vehicle_id column
	VehicleNumber    string `json:"vehicle_number" db:"vehicle_number"`   // Maps to vehicle_number column
	Unnamed1         string `json:"unnamed_1" db:"unnamed_1"`             // Maps to unnamed_1 column
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
// MonthlyMileageReport represents monthly bus mileage data
type MonthlyMileageReport struct {
	ID             int     `json:"id" db:"id"`
	ReportMonth    string  `json:"report_month" db:"report_month"`
	ReportYear     int     `json:"report_year" db:"report_year"`
	BusYear        int     `json:"bus_year" db:"bus_year"`
	BusMake        string  `json:"bus_make" db:"bus_make"`
	LicensePlate   string  `json:"license_plate" db:"license_plate"`
	BusID          string  `json:"bus_id" db:"bus_id"`
	LocatedAt      string  `json:"located_at" db:"located_at"`
	BeginningMiles int     `json:"beginning_miles" db:"beginning_miles"`
	EndingMiles    int     `json:"ending_miles" db:"ending_miles"`
	TotalMiles     int     `json:"total_miles" db:"total_miles"`
	CreatedAt      string  `json:"created_at" db:"created_at"`
	UpdatedAt      string  `json:"updated_at" db:"updated_at"`
}

// ECSETransportationReport represents ECSE transportation cost data
type ECSETransportationReport struct {
	ID                           int     `json:"id" db:"id"`
	ReportMonth                  string  `json:"report_month" db:"report_month"`
	ReportYear                   int     `json:"report_year" db:"report_year"`
	SchoolDistrict               string  `json:"school_district" db:"school_district"`
	Center                       string  `json:"center" db:"center"`
	RouteType                    string  `json:"route_type" db:"route_type"`
	DriverName                   string  `json:"driver_name" db:"driver_name"`
	TotalStudents                int     `json:"total_students" db:"total_students"`
	ECSEStudents                 int     `json:"ecse_students" db:"ecse_students"`
	CostPerMile                  float64 `json:"cost_per_mile" db:"cost_per_mile"`
	MilesPerRoute                float64 `json:"miles_per_route" db:"miles_per_route"`
	CostPerRoute                 float64 `json:"cost_per_route" db:"cost_per_route"`
	DistrictResponsibilityPercent float64 `json:"district_responsibility_percent" db:"district_responsibility_percent"`
	DistrictCostPerRoute         float64 `json:"district_cost_per_route" db:"district_cost_per_route"`
	CreatedAt                    string  `json:"created_at" db:"created_at"`
	UpdatedAt                    string  `json:"updated_at" db:"updated_at"`
}

// ReportSummaryData for manager dashboard
type ReportSummaryData struct {
	User                      *User                       `json:"user"`
	MileageSummary            MileageSummary              `json:"mileage_summary"`
	ECSESummary               ECSESummary                 `json:"ecse_summary"`
	RecentMileageReports      []MonthlyMileageReport      `json:"recent_mileage_reports"`
	RecentECSEReports         []ECSETransportationReport  `json:"recent_ecse_reports"`
	CSRFToken                 string                      `json:"csrf_token"`
}

// MileageSummary contains aggregated mileage data
type MileageSummary struct {
	TotalBuses        int     `json:"total_buses"`
	TotalMiles        int     `json:"total_miles"`
	AverageMilesPerBus float64 `json:"average_miles_per_bus"`
	CurrentMonthMiles int     `json:"current_month_miles"`
}

// ECSESummary contains aggregated ECSE data
type ECSESummary struct {
	TotalDistricts    int     `json:"total_districts"`
	TotalRoutes       int     `json:"total_routes"`
	TotalECSEStudents int     `json:"total_ecse_students"`
	TotalCost         float64 `json:"total_cost"`
	AverageCostPerRoute float64 `json:"average_cost_per_route"`
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

// ============= ENHANCED ROUTE ASSIGNMENT DATA MODEL =============
// AssignRouteData is used for the route assignment page with enhanced status tracking
type AssignRouteData struct {
	User            *User             `json:"user"`
	Assignments     []RouteAssignment `json:"assignments"`
	Drivers         []User            `json:"drivers"`          // Available drivers only
	AvailableRoutes []Route           `json:"available_routes"`
	AvailableBuses  []*Bus            `json:"available_buses"`  // Available buses only
	CSRFToken       string            `json:"csrf_token"`
	
	// ===== ENHANCED FIELDS FOR STATUS TRACKING =====
	RoutesWithStatus []struct {
		Route
		IsAssigned bool `json:"is_assigned"`
	} `json:"routes_with_status"`
	
	// ===== PRE-CALCULATED STATISTICS =====
	TotalAssignments      int `json:"total_assignments"`       // Total driver assignments
	TotalRoutes           int `json:"total_routes"`            // Total route definitions
	AvailableDriversCount int `json:"available_drivers_count"` // Unassigned drivers only
	AvailableBusesCount   int `json:"available_buses_count"`   // Unassigned buses only
}

// FleetData is used for the fleet management page
type FleetData struct {
    User            *User                `json:"user"`
    Buses           []*Bus               `json:"buses"`
    Today           string               `json:"today"`
    CSRFToken       string               `json:"csrf_token"`
    MaintenanceLogs []BusMaintenanceLog  `json:"maintenance_logs"` // Add this field
}

// StudentData is used for the student management page
type StudentData struct {
	User      *User     `json:"user"`
	Students  []Student `json:"students"`
	Routes    []Route   `json:"routes"`
	CSRFToken string    `json:"csrf_token"` // Added for CSRF protection
}

// CompanyFleetData is used for the company fleet page (UPDATED FOR POSTGRESQL COLUMNS)
type CompanyFleetData struct {
	User                  *User     `json:"user"`
	Vehicles              []Vehicle `json:"vehicles"`
	CSRFToken             string    `json:"csrf_token"` // Added for CSRF protection
	AvailableDriversCount int       `json:"available_drivers_count"`
	AvailableBusesCount   int       `json:"available_buses_count"`
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
	Students   []Student   `json:"students"`  // Added for student list
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

// VehicleMaintenanceData is used for the vehicle maintenance page (NEW)
type VehicleMaintenanceData struct {
	User               *User               `json:"user"`
	Vehicle            Vehicle             `json:"vehicle"`
	VehicleID          string              `json:"vehicle_id"`       // For backward compatibility
	VehicleNumber      string              `json:"vehicle_number"`   // For backward compatibility
	Unnamed1           string              `json:"unnamed_1"`        // For backward compatibility
	IsBus              bool                `json:"is_bus"`
	MaintenanceRecords []BusMaintenanceLog `json:"maintenance_records"`
	TotalRecords       int                 `json:"total_records"`
	TotalCost          float64             `json:"total_cost"`
	AverageCost        float64             `json:"average_cost"`
	RecentCount        int                 `json:"recent_count"`
	Today              string              `json:"today"`
	CSRFToken          string              `json:"csrf_token"`
	
	// Vehicle info fields for display
	VehicleInfo *Vehicle `json:"vehicle_info,omitempty"`
}

// Helper method for Vehicle to get display ID
func (v *Vehicle) GetDisplayID() string {
	if v.VehicleNumber != "" {
		return v.VehicleNumber
	}
	if v.VehicleID != "" {
		return v.VehicleID
	}
	return v.Unnamed1
}

// Helper method for Vehicle to get any identifier
func (v *Vehicle) GetIdentifier() string {
	if v.VehicleID != "" {
		return v.VehicleID
	}
	if v.VehicleNumber != "" {
		return v.VehicleNumber
	}
	return v.Unnamed1
}
