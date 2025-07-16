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
	UpdatedAt        time.Time `json:"updated_at" db:"updated_at"`
	CurrentMileage   int    `json:"current_mileage" db:"current_mileage"` // NEW FIELD
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
	CurrentMileage   int       `json:"current_mileage" db:"current_mileage"` // NEW FIELD
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time `json:"updated_at" db:"updated_at"`
}

// MaintenanceSchedule defines when maintenance is due
type MaintenanceSchedule struct {
	ID               int    `json:"id" db:"id"`
	VehicleID        string `json:"vehicle_id" db:"vehicle_id"`
	MaintenanceType  string `json:"maintenance_type" db:"maintenance_type"`
	IntervalMiles    int    `json:"interval_miles" db:"interval_miles"`
	LastServiceMiles int    `json:"last_service_miles" db:"last_service_miles"`
	LastServiceDate  string `json:"last_service_date" db:"last_service_date"`
	NextServiceMiles int    `json:"next_service_miles" db:"next_service_miles"`
	IsDue            bool   `json:"is_due" db:"is_due"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
}

// MaintenanceAlert represents a maintenance alert
type MaintenanceAlert struct {
	VehicleID       string `json:"vehicle_id"`
	MaintenanceType string `json:"maintenance_type"`
	CurrentMileage  int    `json:"current_mileage"`
	DueMileage      int    `json:"due_mileage"`
	MilesUntilDue   int    `json:"miles_until_due"`
	IsOverdue       bool   `json:"is_overdue"`
	Severity        string `json:"severity"` // "info", "warning", "danger"
	Message         string `json:"message"`
}

// MileageValidationResult represents the result of mileage validation
type MileageValidationResult struct {
	Valid        bool   `json:"valid"`
	Error        string `json:"error,omitempty"`
	Warning      string `json:"warning,omitempty"`
	ShouldUpdate bool   `json:"should_update"`
	NewOilStatus string `json:"new_oil_status,omitempty"`
	NewTireStatus string `json:"new_tire_status,omitempty"`
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

// ECSE (Early Childhood Special Education) Models
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
    Address                string    `json:"address" db:"address"`
    City                   string    `json:"city" db:"city"`
    State                  string    `json:"state" db:"state"`
    ZipCode                string    `json:"zip_code" db:"zip_code"`
    Notes                  string    `json:"notes" db:"notes"`
    CreatedAt              time.Time `json:"created_at" db:"created_at"`
    UpdatedAt              time.Time `json:"updated_at" db:"updated_at"`
}

type ECSEService struct {
    ID          int       `json:"id" db:"id"`
    StudentID   string    `json:"student_id" db:"student_id"`
    ServiceType string    `json:"service_type" db:"service_type"`
    Frequency   string    `json:"frequency" db:"frequency"`
    Duration    int       `json:"duration" db:"duration"`
    Provider    string    `json:"provider" db:"provider"`
    StartDate   string    `json:"start_date" db:"start_date"`
    EndDate     string    `json:"end_date" db:"end_date"`
    Goals       string    `json:"goals" db:"goals"`
    Progress    string    `json:"progress" db:"progress"`
    CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

type ECSEAssessment struct {
    ID             int       `json:"id" db:"id"`
    StudentID      string    `json:"student_id" db:"student_id"`
    AssessmentType string    `json:"assessment_type" db:"assessment_type"`
    AssessmentDate string    `json:"assessment_date" db:"assessment_date"`
    Score          string    `json:"score" db:"score"`
    Evaluator      string    `json:"evaluator" db:"evaluator"`
    Notes          string    `json:"notes" db:"notes"`
    NextReviewDate string    `json:"next_review_date" db:"next_review_date"`
    CreatedAt      time.Time `json:"created_at" db:"created_at"`
}

type ECSEAttendance struct {
    ID            int       `json:"id" db:"id"`
    StudentID     string    `json:"student_id" db:"student_id"`
    AttendanceDate string   `json:"attendance_date" db:"attendance_date"`
    Status        string    `json:"status" db:"status"`
    ArrivalTime   string    `json:"arrival_time" db:"arrival_time"`
    DepartureTime string    `json:"departure_time" db:"departure_time"`
    Notes         string    `json:"notes" db:"notes"`
    CreatedAt     time.Time `json:"created_at" db:"created_at"`
}

// View models for ECSE
type ECSEStudentView struct {
    StudentID              string `json:"student_id" db:"student_id"`
    FirstName              string `json:"first_name" db:"first_name"`
    LastName               string `json:"last_name" db:"last_name"`
    DateOfBirth            string `json:"date_of_birth" db:"date_of_birth"`
    Grade                  string `json:"grade" db:"grade"`
    EnrollmentStatus       string `json:"enrollment_status" db:"enrollment_status"`
    IEPStatus              string `json:"iep_status" db:"iep_status"`
    PrimaryDisability      string `json:"primary_disability" db:"primary_disability"`
    ServiceMinutes         int    `json:"service_minutes" db:"service_minutes"`
    TransportationRequired bool   `json:"transportation_required" db:"transportation_required"`
    BusRoute               string `json:"bus_route" db:"bus_route"`
    ParentName             string `json:"parent_name" db:"parent_name"`
    ParentPhone            string `json:"parent_phone" db:"parent_phone"`
    ParentEmail            string `json:"parent_email" db:"parent_email"`
    Address                string `json:"address" db:"address"`
    City                   string `json:"city" db:"city"`
    State                  string `json:"state" db:"state"`
    ZipCode                string `json:"zip_code" db:"zip_code"`
    ServiceCount           int    `json:"service_count" db:"service_count"`
    AssessmentCount        int    `json:"assessment_count" db:"assessment_count"`
}

type ECSEAttendanceRecord struct {
    Date          string `json:"date" db:"date"`
    Status        string `json:"status" db:"status"`
    ArrivalTime   string `json:"arrival_time" db:"arrival_time"`
    DepartureTime string `json:"departure_time" db:"departure_time"`
    Notes         string `json:"notes" db:"notes"`
}

// ECSE Statistics
type ECSEStats struct {
    TotalStudents          int              `json:"total_students"`
    ActiveStudents         int              `json:"active_students"`
    TransportationStudents int              `json:"transportation_students"`
    IEPStudents            int              `json:"iep_students"`
    TotalServices          int              `json:"total_services"`
    ServiceTypes           map[string]int   `json:"service_types"`
}

// ECSE Import Result
type ECSEImportResult struct {
    StudentsImported    int      `json:"students_imported"`
    ServicesImported    int
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
	ID         int                 `json:"id" db:"id"`
	Driver     string              `json:"driver" db:"driver"`
	BusID      string              `json:"bus_id" db:"bus_id"`
	RouteID    string              `json:"route_id" db:"route_id"`
	Date       string              `json:"date" db:"date"`
	Period     string              `json:"period" db:"period"` // "morning" or "afternoon"
	Departure  string              `json:"departure_time" db:"departure_time"`
	Arrival    string              `json:"arrival_time" db:"arrival_time"`
	Mileage    float64             `json:"mileage" db:"mileage"`
	Attendance []StudentAttendance `json:"attendance"`
	CreatedAt  time.Time           `json:"created_at" db:"created_at"`
}

// Activity represents a special activity or trip
type Activity struct {
	Date       string  `db:"date"`
	Driver     string  `db:"driver"`
	TripName   string  `db:"trip_name"`
	Attendance int     `db:"attendance"`
	Miles      float64 `db:"miles"`
	Notes      string  `db:"notes"`
}

// RouteLog represents a driver's daily route log
type RouteLog struct {
	ID         int                 `db:"id"`
	Driver     string              `db:"driver"`
	Date       string              `db:"date"`
	Period     string              `db:"period"`
	RouteID    string              `db:"route_id"`
	BusID      string              `db:"bus_id"`
	Mileage    float64             `db:"mileage"`
	Departure  string              `db:"departure"`
	Arrival    string              `db:"arrival"`
	Attendance []StudentAttendance `db:"attendance"`
}

// StudentAttendance represents student attendance on a route
type StudentAttendance struct {
	Position   int    `db:"position"`
	Present    bool   `db:"present"`
	PickupTime string `db:"pickup_time"`
}

// MileageReport represents a monthly mileage report for a vehicle
type MileageReport struct {
	ReportMonth    string `db:"report_month"`
	ReportYear     int    `db:"report_year"`
	VehicleYear    int    `db:"vehicle_year"`
	MakeModel      string `db:"make_model"`
	LicensePlate   string `db:"license_plate"`
	VehicleID      string `db:"vehicle_id"`
	Location       string `db:"location"`
	BeginningMiles int    `db:"beginning_miles"`
	EndingMiles    int    `db:"ending_miles"`
	TotalMiles     int    `db:"total_miles"`
	Status         string `db:"status"`
}

// Mileage reporting structures
type AgencyVehicleRecord struct {
	ReportMonth    string `db:"report_month"`
	ReportYear     int    `db:"report_year"`
	VehicleYear    int    `db:"vehicle_year"`
	MakeModel      string `db:"make_model"`
	LicensePlate   string `db:"license_plate"`
	VehicleID      string `db:"vehicle_id"`
	Location       string `db:"location"`
	BeginningMiles int    `db:"beginning_miles"`
	EndingMiles    int    `db:"ending_miles"`
	TotalMiles     int    `db:"total_miles"`
	Status         string `db:"status"`
	Notes          string `db:"notes"`
}

type SchoolBusRecord struct {
	ReportMonth    string `db:"report_month"`
	ReportYear     int    `db:"report_year"`
	BusYear        int    `db:"bus_year"`
	BusMake        string `db:"bus_make"`
	LicensePlate   string `db:"license_plate"`
	BusID          string `db:"bus_id"`
	Location       string `db:"location"`
	BeginningMiles int    `db:"beginning_miles"`
	EndingMiles    int    `db:"ending_miles"`
	TotalMiles     int    `db:"total_miles"`
	Status         string `db:"status"`
	Notes          string `db:"notes"`
}

type ProgramStaffRecord struct {
	ReportMonth  string `db:"report_month"`
	ReportYear   int    `db:"report_year"`
	ProgramType  string `db:"program_type"` // "HS", "OPK", or "EHS"
	StaffCount1  int    `db:"staff_count_1"`
	StaffCount2  int    `db:"staff_count_2"`
}

type MileageStatistics struct {
    TotalVehicles      int
    ActiveVehicles     int
    TotalMiles         int
    EstimatedCost      float64
    CostPerMile        float64
    AvgMilesPerVehicle int
    VehicleUtilization float64
    PercentChange      float64
    AverageMPG         float64
    FuelEfficiency     float64
    DriverStats        []DriverStatistic
    RouteStats         []RouteStatistic
}

type DriverStatistic struct {
    DriverName          string
    TotalTrips          int
    TotalMiles          int
    AvgMilesPerTrip     int
    StudentsTransported int
    AttendanceRate      float64
    EfficiencyScore     int
}

type RouteStatistic struct {
    RouteName      string
    TotalRuns      int
    TotalMiles     int
    AvgStudents    int
    Efficiency     float64
    CostPerStudent float64
}

// Template data structures

type DashboardData struct {
	User            *User
	Role            string
	Users           []User
	Buses           []*Bus
	Routes          []*Route
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
	Assignments           []*RouteAssignment
	Drivers               []User
	Routes                []*Route
	AvailableRoutes       []*Route
	AvailableBuses        []*Bus
	RoutesWithStatus      []*RouteWithStatus
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
