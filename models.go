package main

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// User represents a user in the system
type User struct {
	// NO ID field - username is the primary key
	Username         string           `json:"username" db:"username"`
	Password         string           `json:"password,omitempty" db:"password"`
	Role             string           `json:"role" db:"role"`
	Status           string           `json:"status" db:"status"`
	RegistrationDate time.Time        `json:"registration_date" db:"registration_date"`
	CreatedAt        time.Time        `json:"created_at" db:"created_at"`
	Assignment       *RouteAssignment `json:"assignment,omitempty" db:"-"` // Current route assignment
}

// Bus represents a school bus
type Bus struct {
	// NO ID field in database - bus_id is the primary key
	BusID            string           `json:"bus_id" db:"bus_id"`
	Status           string           `json:"status" db:"status"`
	Model            sql.NullString   `json:"model" db:"model"`
	Capacity         sql.NullInt32    `json:"capacity" db:"capacity"`
	OilStatus        sql.NullString   `json:"oil_status" db:"oil_status"`
	TireStatus       sql.NullString   `json:"tire_status" db:"tire_status"`
	MaintenanceNotes sql.NullString   `json:"maintenance_notes" db:"maintenance_notes"`
	CurrentMileage   sql.NullInt32    `json:"current_mileage" db:"current_mileage"`
	LastOilChange    sql.NullInt32    `json:"last_oil_change" db:"last_oil_change"`
	LastTireService  sql.NullInt32    `json:"last_tire_service" db:"last_tire_service"`
	UpdatedAt        sql.NullTime     `json:"updated_at" db:"updated_at"`
	CreatedAt        sql.NullTime     `json:"created_at" db:"created_at"`
	Assignment       *RouteAssignment `json:"assignment,omitempty" db:"-"` // Current route assignment
}

// Helper methods for Bus to handle null values in templates
func (b Bus) GetModel() string {
	if b.Model.Valid {
		return b.Model.String
	}
	return ""
}

func (b Bus) GetCapacity() int {
	if b.Capacity.Valid {
		return int(b.Capacity.Int32)
	}
	return 0
}

func (b Bus) GetOilStatus() string {
	if b.OilStatus.Valid {
		return b.OilStatus.String
	}
	return "good"
}

func (b Bus) GetTireStatus() string {
	if b.TireStatus.Valid {
		return b.TireStatus.String
	}
	return "good"
}

func (b Bus) GetMaintenanceNotes() string {
	if b.MaintenanceNotes.Valid {
		return b.MaintenanceNotes.String
	}
	return ""
}

// Vehicle represents a company vehicle
type Vehicle struct {
	// NO ID field in database - vehicle_id is the primary key
	VehicleID        string         `json:"vehicle_id" db:"vehicle_id"`
	Model            sql.NullString `json:"model" db:"model"`
	Description      sql.NullString `json:"description" db:"description"`
	Year             sql.NullString `json:"year" db:"year"` // VARCHAR in database, not INTEGER
	TireSize         sql.NullString `json:"tire_size" db:"tire_size"`
	License          sql.NullString `json:"license" db:"license"`
	OilStatus        sql.NullString `json:"oil_status" db:"oil_status"`
	TireStatus       sql.NullString `json:"tire_status" db:"tire_status"`
	Status           sql.NullString `json:"status" db:"status"`
	MaintenanceNotes sql.NullString `json:"maintenance_notes" db:"maintenance_notes"`
	SerialNumber     sql.NullString `json:"serial_number" db:"serial_number"`
	Base             sql.NullString `json:"base" db:"base"`
	ServiceInterval  sql.NullInt32  `json:"service_interval" db:"service_interval"`
	CurrentMileage   sql.NullInt32  `json:"current_mileage" db:"current_mileage"`
	LastOilChange    sql.NullInt32  `json:"last_oil_change" db:"last_oil_change"`
	LastTireService  sql.NullInt32  `json:"last_tire_service" db:"last_tire_service"`
	UpdatedAt        sql.NullTime   `json:"updated_at" db:"updated_at"`
	CreatedAt        sql.NullTime   `json:"created_at" db:"created_at"`
	ImportID         sql.NullString `json:"import_id" db:"import_id"`
}

// Helper methods for Vehicle to handle null values in templates
func (v Vehicle) GetModel() string {
	if v.Model.Valid {
		return v.Model.String
	}
	return ""
}

func (v Vehicle) GetDescription() string {
	if v.Description.Valid {
		return v.Description.String
	}
	return ""
}

func (v Vehicle) GetYear() string {
	if v.Year.Valid {
		return v.Year.String
	}
	return ""
}

func (v Vehicle) GetTireSize() string {
	if v.TireSize.Valid {
		return v.TireSize.String
	}
	return ""
}

func (v Vehicle) GetLicense() string {
	if v.License.Valid {
		return v.License.String
	}
	return ""
}

func (v Vehicle) GetOilStatus() string {
	if v.OilStatus.Valid {
		return v.OilStatus.String
	}
	return "unknown"
}

func (v Vehicle) GetTireStatus() string {
	if v.TireStatus.Valid {
		return v.TireStatus.String
	}
	return "unknown"
}

func (v Vehicle) GetMaintenanceNotes() string {
	if v.MaintenanceNotes.Valid {
		return v.MaintenanceNotes.String
	}
	return ""
}

func (v Vehicle) GetSerialNumber() string {
	if v.SerialNumber.Valid {
		return v.SerialNumber.String
	}
	return ""
}

func (v Vehicle) GetBase() string {
	if v.Base.Valid {
		return v.Base.String
	}
	return ""
}

func (v Vehicle) GetServiceInterval() int {
	if v.ServiceInterval.Valid {
		return int(v.ServiceInterval.Int32)
	}
	return 0
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
	ID          int       `json:"id" db:"id"`
	VehicleID   string    `json:"vehicle_id" db:"vehicle_id"`
	VehicleType string    `json:"vehicle_type" db:"vehicle_type"`
	Date        string    `json:"date" db:"date"`
	Category    string    `json:"category" db:"category"`
	Notes       string    `json:"notes" db:"notes"`
	Mileage     int       `json:"mileage" db:"mileage"`
	Cost        float64   `json:"cost" db:"cost"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// MaintenanceSchedule defines when maintenance is due
type MaintenanceSchedule struct {
	ItemName      string
	Interval      int // miles
	WarningMiles  int // miles before due to show warning
	CriticalMiles int // miles overdue to show critical
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
	// NO ID field - route_id is the primary key
	RouteID     string    `json:"route_id" db:"route_id"`
	RouteName   string    `json:"route_name" db:"route_name"`
	Description string    `json:"description" db:"description"`
	Positions   string    `json:"positions" db:"positions"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// RouteAssignment represents a driver-bus-route assignment
type RouteAssignment struct {
	ID           int       `json:"id" db:"id"` // This table DOES have an id field
	Driver       string    `json:"driver" db:"driver"`
	BusID        string    `json:"bus_id" db:"bus_id"`
	RouteID      string    `json:"route_id" db:"route_id"`
	RouteName    string    `json:"route_name" db:"route_name"`
	AssignedDate string    `json:"assigned_date" db:"assigned_date"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

// Student represents a student
type Student struct {
	// NO ID field - student_id is the primary key
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

// ECSEStudent represents a special education student
type ECSEStudent struct {
	// NO ID field - student_id is the primary key
	StudentID              string         `json:"student_id" db:"student_id"`
	FirstName              string         `json:"first_name" db:"first_name"`
	LastName               string         `json:"last_name" db:"last_name"`
	DateOfBirth            sql.NullString `json:"date_of_birth" db:"date_of_birth"`
	Grade                  sql.NullString `json:"grade" db:"grade"`
	EnrollmentStatus       sql.NullString `json:"enrollment_status" db:"enrollment_status"`
	IEPStatus              sql.NullString `json:"iep_status" db:"iep_status"`
	PrimaryDisability      sql.NullString `json:"primary_disability" db:"primary_disability"`
	ServiceMinutes         sql.NullInt32  `json:"service_minutes" db:"service_minutes"`
	TransportationRequired sql.NullBool   `json:"transportation_required" db:"transportation_required"`
	BusRoute               sql.NullString `json:"bus_route" db:"bus_route"`
	ParentName             sql.NullString `json:"parent_name" db:"parent_name"`
	ParentPhone            sql.NullString `json:"parent_phone" db:"parent_phone"`
	ParentEmail            sql.NullString `json:"parent_email" db:"parent_email"`
	Address                sql.NullString `json:"address" db:"address"`
	City                   sql.NullString `json:"city" db:"city"`
	State                  sql.NullString `json:"state" db:"state"`
	ZipCode                sql.NullString `json:"zip_code" db:"zip_code"`
	Notes                  sql.NullString `json:"notes" db:"notes"`
	CreatedAt              sql.NullTime   `json:"created_at" db:"created_at"`
	UpdatedAt              sql.NullTime   `json:"updated_at" db:"updated_at"`
	ImportID               sql.NullString `json:"import_id" db:"import_id"`
}

// MileageReport represents imported mileage data
type MileageReport struct {
	ID               int       `json:"id" db:"id"`
	VehicleID        string    `json:"vehicle_id" db:"vehicle_id"`
	Driver           string    `json:"driver" db:"driver"`
	Month            int       `json:"month" db:"month"`
	Year             int       `json:"year" db:"year"`
	BeginningMileage float64   `json:"beginning_mileage" db:"beginning_mileage"`
	EndingMileage    float64   `json:"ending_mileage" db:"ending_mileage"`
	TotalMiles       float64   `json:"total_miles" db:"total_miles"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
}

// ECSEService represents services provided to ECSE students
type ECSEService struct {
	ID          int       `json:"id" db:"id"`
	StudentID   string    `json:"student_id" db:"student_id"`
	ServiceType string    `json:"service_type" db:"service_type"`
	Frequency   string    `json:"frequency" db:"frequency"`
	Duration    int       `json:"duration" db:"duration"`
	Provider    string    `json:"provider" db:"provider"`
	StartDate   time.Time `json:"start_date" db:"start_date"`
	EndDate     time.Time `json:"end_date" db:"end_date"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// ECSEAssessment represents assessment data for ECSE students
type ECSEAssessment struct {
	ID                 int       `json:"id" db:"id"`
	StudentID          string    `json:"student_id" db:"student_id"`
	AssessmentDate     time.Time `json:"assessment_date" db:"assessment_date"`
	AssessmentType     string    `json:"assessment_type" db:"assessment_type"`
	Results            string    `json:"results" db:"results"`
	Evaluator          string    `json:"evaluator" db:"evaluator"`
	NextAssessmentDate time.Time `json:"next_assessment_date" db:"next_assessment_date"`
	CreatedAt          time.Time `json:"created_at" db:"created_at"`
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

// MonthlyMileageReport represents monthly mileage report data
type MonthlyMileageReport struct {
	ID             int       `json:"id" db:"id"`
	ReportMonth    string    `json:"report_month" db:"report_month"`
	ReportYear     int       `json:"report_year" db:"report_year"`
	BusYear        int       `json:"bus_year" db:"bus_year"`
	BusMake        string    `json:"bus_make" db:"bus_make"`
	LicensePlate   string    `json:"license_plate" db:"license_plate"`
	BusID          string    `json:"bus_id" db:"bus_id"`
	LocatedAt      string    `json:"located_at" db:"located_at"`
	BeginningMiles int       `json:"beginning_miles" db:"beginning_miles"`
	EndingMiles    int       `json:"ending_miles" db:"ending_miles"`
	TotalMiles     int       `json:"total_miles" db:"total_miles"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" db:"updated_at"`
}

// GetMonthYearString returns formatted month/year for display
func (m MonthlyMileageReport) GetMonthYearString() string {
	return fmt.Sprintf("%s %d", m.ReportMonth, m.ReportYear)
}

// GetMileageDifference calculates the difference between ending and beginning miles
func (m MonthlyMileageReport) GetMileageDifference() int {
	return m.EndingMiles - m.BeginningMiles
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

// FleetVehicle represents a vehicle in the fleet_vehicles table
type FleetVehicle struct {
	ID            int            `json:"id" db:"id"`
	VehicleNumber sql.NullInt32  `json:"vehicle_number" db:"vehicle_number"`
	SheetName     sql.NullString `json:"sheet_name" db:"sheet_name"`
	Year          sql.NullInt32  `json:"year" db:"year"`
	Make          sql.NullString `json:"make" db:"make"`
	Model         sql.NullString `json:"model" db:"model"`
	Description   sql.NullString `json:"description" db:"description"`
	SerialNumber  sql.NullString `json:"serial_number" db:"serial_number"`
	License       sql.NullString `json:"license" db:"license"`
	Location      sql.NullString `json:"location" db:"location"`
	TireSize      sql.NullString `json:"tire_size" db:"tire_size"`
	CreatedAt     time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at" db:"updated_at"`
}

// ConsolidatedVehicle represents a vehicle in the new consolidated fleet_vehicles table
type ConsolidatedVehicle struct {
	ID               string           `json:"id" db:"id"`
	VehicleID        string           `json:"vehicle_id" db:"vehicle_id"`
	VehicleType      string           `json:"vehicle_type" db:"vehicle_type"` // "bus" or "vehicle"
	Status           string           `json:"status" db:"status"`
	Model            sql.NullString   `json:"model" db:"model"`
	Capacity         sql.NullInt32    `json:"capacity" db:"capacity"`
	OilStatus        sql.NullString   `json:"oil_status" db:"oil_status"`
	TireStatus       sql.NullString   `json:"tire_status" db:"tire_status"`
	MaintenanceNotes sql.NullString   `json:"maintenance_notes" db:"maintenance_notes"`
	Year             sql.NullString   `json:"year" db:"year"`
	TireSize         sql.NullString   `json:"tire_size" db:"tire_size"`
	License          sql.NullString   `json:"license" db:"license"`
	Description      sql.NullString   `json:"description" db:"description"`
	SerialNumber     sql.NullString   `json:"serial_number" db:"serial_number"`
	Base             sql.NullString   `json:"base" db:"base"`
	ServiceInterval  sql.NullInt32    `json:"service_interval" db:"service_interval"`
	UpdatedAt        sql.NullTime     `json:"updated_at" db:"updated_at"`
	CreatedAt        sql.NullTime     `json:"created_at" db:"created_at"`
	Assignment       *RouteAssignment `json:"assignment,omitempty" db:"-"` // Current route assignment
	
	// Computed fields for backward compatibility
	BusID string `json:"bus_id" db:"-"` // Alias for VehicleID
}

// Helper methods for ConsolidatedVehicle to handle null values in templates
func (cv ConsolidatedVehicle) GetModel() string {
	if cv.Model.Valid {
		return cv.Model.String
	}
	return ""
}

func (cv ConsolidatedVehicle) GetCapacity() int {
	if cv.Capacity.Valid {
		return int(cv.Capacity.Int32)
	}
	return 0
}

func (cv ConsolidatedVehicle) GetOilStatus() string {
	if cv.OilStatus.Valid {
		return cv.OilStatus.String
	}
	return "good"
}

func (cv ConsolidatedVehicle) GetTireStatus() string {
	if cv.TireStatus.Valid {
		return cv.TireStatus.String
	}
	return "good"
}

func (cv ConsolidatedVehicle) GetMaintenanceNotes() string {
	if cv.MaintenanceNotes.Valid {
		return cv.MaintenanceNotes.String
	}
	return ""
}

func (cv ConsolidatedVehicle) GetYear() string {
	if cv.Year.Valid {
		return cv.Year.String
	}
	return ""
}

func (cv ConsolidatedVehicle) GetTireSize() string {
	if cv.TireSize.Valid {
		return cv.TireSize.String
	}
	return ""
}

func (cv ConsolidatedVehicle) GetLicense() string {
	if cv.License.Valid {
		return cv.License.String
	}
	return ""
}

func (cv ConsolidatedVehicle) GetDescription() string {
	if cv.Description.Valid {
		return cv.Description.String
	}
	return ""
}

func (cv ConsolidatedVehicle) GetSerialNumber() string {
	if cv.SerialNumber.Valid {
		return cv.SerialNumber.String
	}
	return ""
}

func (cv ConsolidatedVehicle) GetBase() string {
	if cv.Base.Valid {
		return cv.Base.String
	}
	return ""
}

func (cv ConsolidatedVehicle) GetServiceInterval() int {
	if cv.ServiceInterval.Valid {
		return int(cv.ServiceInterval.Int32)
	}
	return 0
}

// GetBusID returns the VehicleID for backward compatibility with templates
func (cv ConsolidatedVehicle) GetBusID() string {
	return cv.VehicleID
}

// Helper methods for FleetVehicle to handle null values in templates
func (fv FleetVehicle) GetVehicleNumber() int {
	if fv.VehicleNumber.Valid {
		return int(fv.VehicleNumber.Int32)
	}
	return 0
}

func (fv FleetVehicle) GetSheetName() string {
	if fv.SheetName.Valid {
		return fv.SheetName.String
	}
	return ""
}

func (fv FleetVehicle) GetYear() int {
	if fv.Year.Valid {
		return int(fv.Year.Int32)
	}
	return 0
}

func (fv FleetVehicle) GetMake() string {
	if fv.Make.Valid {
		return fv.Make.String
	}
	return ""
}

func (fv FleetVehicle) GetModel() string {
	if fv.Model.Valid {
		return fv.Model.String
	}
	return ""
}

func (fv FleetVehicle) GetDescription() string {
	if fv.Description.Valid {
		return fv.Description.String
	}
	return ""
}

func (fv FleetVehicle) GetSerialNumber() string {
	if fv.SerialNumber.Valid {
		return fv.SerialNumber.String
	}
	return ""
}

func (fv FleetVehicle) GetLicense() string {
	if fv.License.Valid {
		return fv.License.String
	}
	return ""
}

func (fv FleetVehicle) GetLocation() string {
	if fv.Location.Valid {
		return fv.Location.String
	}
	return ""
}

func (fv FleetVehicle) GetTireSize() string {
	if fv.TireSize.Valid {
		return fv.TireSize.String
	}
	return ""
}

// GetVehicleIdentifier returns the best available identifier for the vehicle
func (fv FleetVehicle) GetVehicleIdentifier() string {
	if fv.VehicleNumber.Valid && fv.VehicleNumber.Int32 > 0 {
		return fmt.Sprintf("%d", fv.VehicleNumber.Int32)
	}
	if fv.License.Valid && fv.License.String != "" {
		return fv.License.String
	}
	if fv.SerialNumber.Valid && fv.SerialNumber.String != "" {
		return fv.SerialNumber.String[:min(8, len(fv.SerialNumber.String))]
	}
	return fmt.Sprintf("FV-%d", fv.ID)
}

// GetVehicleIDForMaintenance returns a properly formatted vehicle ID for maintenance URLs
func (fv FleetVehicle) GetVehicleIDForMaintenance() string {
	if fv.VehicleNumber.Valid && fv.VehicleNumber.Int32 > 0 {
		return fmt.Sprintf("%d", fv.VehicleNumber.Int32)
	}
	return fmt.Sprintf("fleet-%d", fv.ID)
}

// GetMakeModel returns combined make and model
func (fv FleetVehicle) GetMakeModel() string {
	make := fv.GetMake()
	model := fv.GetModel()
	if make != "" && model != "" {
		return fmt.Sprintf("%s %s", make, model)
	}
	if make != "" {
		return make
	}
	if model != "" {
		return model
	}
	return "Unknown"
}

// MaintenanceRecord represents a maintenance record from the maintenance_records table
type MaintenanceRecord struct {
	ID              int            `json:"id" db:"id"`
	VehicleNumber   sql.NullInt32  `json:"vehicle_number" db:"vehicle_number"`
	ServiceDate     sql.NullTime   `json:"service_date" db:"service_date"`
	Mileage         sql.NullInt32  `json:"mileage" db:"mileage"`
	PONumber        sql.NullString `json:"po_number" db:"po_number"`
	Cost            sql.NullString `json:"cost" db:"cost"` // Stored as string in DB due to varying formats
	WorkDescription sql.NullString `json:"work_description" db:"work_description"`
	RawData         sql.NullString `json:"raw_data" db:"raw_data"`
	CreatedAt       time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at" db:"updated_at"`
	VehicleID       sql.NullString `json:"vehicle_id" db:"vehicle_id"`
	Date            sql.NullTime   `json:"date" db:"date"`
}

// Helper methods for MaintenanceRecord to handle null values in templates
func (mr MaintenanceRecord) GetVehicleNumber() int {
	if mr.VehicleNumber.Valid {
		return int(mr.VehicleNumber.Int32)
	}
	return 0
}

func (mr MaintenanceRecord) GetServiceDate() string {
	if mr.ServiceDate.Valid {
		return mr.ServiceDate.Time.Format("2006-01-02")
	}
	return ""
}

func (mr MaintenanceRecord) GetMileage() int {
	if mr.Mileage.Valid {
		return int(mr.Mileage.Int32)
	}
	return 0
}

func (mr MaintenanceRecord) GetPONumber() string {
	if mr.PONumber.Valid {
		return mr.PONumber.String
	}
	return ""
}

func (mr MaintenanceRecord) GetCost() string {
	if mr.Cost.Valid {
		return mr.Cost.String
	}
	return ""
}

func (mr MaintenanceRecord) GetWorkDescription() string {
	if mr.WorkDescription.Valid {
		return mr.WorkDescription.String
	}
	return ""
}

func (mr MaintenanceRecord) GetVehicleID() string {
	if mr.VehicleID.Valid {
		return mr.VehicleID.String
	}
	return ""
}

func (mr MaintenanceRecord) GetDate() string {
	if mr.Date.Valid {
		return mr.Date.Time.Format("2006-01-02")
	}
	return ""
}

// GetVehicleIdentifier returns the best available vehicle identifier
func (mr MaintenanceRecord) GetVehicleIdentifier() string {
	if mr.VehicleNumber.Valid && mr.VehicleNumber.Int32 > 0 {
		return fmt.Sprintf("Vehicle #%d", mr.VehicleNumber.Int32)
	}
	if mr.VehicleID.Valid && mr.VehicleID.String != "" {
		return mr.VehicleID.String
	}
	return fmt.Sprintf("Record #%d", mr.ID)
}

// GetFormattedServiceDate returns a user-friendly service date
func (mr MaintenanceRecord) GetFormattedServiceDate() string {
	if mr.ServiceDate.Valid {
		return mr.ServiceDate.Time.Format("Jan 2, 2006")
	}
	if mr.Date.Valid {
		return mr.Date.Time.Format("Jan 2, 2006")
	}
	return "Unknown"
}

// GetCostAsFloat attempts to parse cost as float for calculations
func (mr MaintenanceRecord) GetCostAsFloat() float64 {
	if !mr.Cost.Valid {
		return 0
	}

	// Try to parse various cost formats
	costStr := strings.ReplaceAll(mr.Cost.String, ",", "")
	costStr = strings.ReplaceAll(costStr, "$", "")
	costStr = strings.TrimSpace(costStr)

	if cost, err := strconv.ParseFloat(costStr, 64); err == nil {
		return cost
	}
	return 0
}

// ServiceRecord represents a service record from the service_records table
// This table has generic unnamed columns from imported data
type ServiceRecord struct {
	ID              int            `json:"id" db:"id"`
	Unnamed0        sql.NullString `json:"unnamed_0" db:"unnamed_0"`
	Unnamed1        sql.NullString `json:"unnamed_1" db:"unnamed_1"`
	Unnamed2        sql.NullString `json:"unnamed_2" db:"unnamed_2"`
	Unnamed3        sql.NullString `json:"unnamed_3" db:"unnamed_3"`
	Unnamed4        sql.NullString `json:"unnamed_4" db:"unnamed_4"`
	Unnamed5        sql.NullString `json:"unnamed_5" db:"unnamed_5"`
	Unnamed6        sql.NullString `json:"unnamed_6" db:"unnamed_6"`
	Unnamed7        sql.NullString `json:"unnamed_7" db:"unnamed_7"`
	Unnamed8        sql.NullString `json:"unnamed_8" db:"unnamed_8"`
	Unnamed9        sql.NullString `json:"unnamed_9" db:"unnamed_9"`
	Unnamed10       sql.NullString `json:"unnamed_10" db:"unnamed_10"`
	Unnamed11       sql.NullString `json:"unnamed_11" db:"unnamed_11"`
	Unnamed12       sql.NullString `json:"unnamed_12" db:"unnamed_12"`
	Unnamed13       sql.NullString `json:"unnamed_13" db:"unnamed_13"`
	CreatedAt       time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at" db:"updated_at"`
	MaintenanceDate sql.NullTime   `json:"maintenance_date" db:"maintenance_date"`
}

// Helper methods for ServiceRecord to handle null values and interpret data
func (sr ServiceRecord) GetField(index int) string {
	switch index {
	case 0:
		if sr.Unnamed0.Valid {
			return sr.Unnamed0.String
		}
	case 1:
		if sr.Unnamed1.Valid {
			return sr.Unnamed1.String
		}
	case 2:
		if sr.Unnamed2.Valid {
			return sr.Unnamed2.String
		}
	case 3:
		if sr.Unnamed3.Valid {
			return sr.Unnamed3.String
		}
	case 4:
		if sr.Unnamed4.Valid {
			return sr.Unnamed4.String
		}
	case 5:
		if sr.Unnamed5.Valid {
			return sr.Unnamed5.String
		}
	case 6:
		if sr.Unnamed6.Valid {
			return sr.Unnamed6.String
		}
	case 7:
		if sr.Unnamed7.Valid {
			return sr.Unnamed7.String
		}
	case 8:
		if sr.Unnamed8.Valid {
			return sr.Unnamed8.String
		}
	case 9:
		if sr.Unnamed9.Valid {
			return sr.Unnamed9.String
		}
	case 10:
		if sr.Unnamed10.Valid {
			return sr.Unnamed10.String
		}
	case 11:
		if sr.Unnamed11.Valid {
			return sr.Unnamed11.String
		}
	case 12:
		if sr.Unnamed12.Valid {
			return sr.Unnamed12.String
		}
	case 13:
		if sr.Unnamed13.Valid {
			return sr.Unnamed13.String
		}
	}
	return ""
}

// GetVehicleInfo attempts to extract vehicle information from the data
func (sr ServiceRecord) GetVehicleInfo() string {
	vehicleInfo := sr.GetField(0)
	if vehicleInfo == "" {
		vehicleInfo = sr.GetField(1)
	}
	if vehicleInfo == "" {
		return fmt.Sprintf("Record #%d", sr.ID)
	}
	return vehicleInfo
}

// GetVehicleNumber attempts to extract vehicle number from the data
func (sr ServiceRecord) GetVehicleNumber() string {
	return sr.GetField(2)
}

// GetServicedMiles attempts to extract serviced miles
func (sr ServiceRecord) GetServicedMiles() string {
	miles := sr.GetField(3)
	if miles == "" {
		miles = sr.GetField(4)
	}
	return miles
}

// GetLastMileage attempts to extract last mileage
func (sr ServiceRecord) GetLastMileage() string {
	return sr.GetField(5)
}

// GetNeedsService attempts to extract needs service info
func (sr ServiceRecord) GetNeedsService() string {
	return sr.GetField(6)
}

// GetMilesToService attempts to extract miles to service
func (sr ServiceRecord) GetMilesToService() string {
	return sr.GetField(8)
}

// GetServiceAt attempts to extract service at mileage
func (sr ServiceRecord) GetServiceAt() string {
	return sr.GetField(9)
}

// GetMaintenanceDate returns formatted maintenance date
func (sr ServiceRecord) GetMaintenanceDate() string {
	if sr.MaintenanceDate.Valid {
		return sr.MaintenanceDate.Time.Format("2006-01-02")
	}
	return ""
}

// GetFormattedMaintenanceDate returns user-friendly maintenance date
func (sr ServiceRecord) GetFormattedMaintenanceDate() string {
	if sr.MaintenanceDate.Valid {
		return sr.MaintenanceDate.Time.Format("Jan 2, 2006")
	}
	return "Unknown"
}

// GetAllFields returns all non-empty fields for display
func (sr ServiceRecord) GetAllFields() []string {
	var fields []string
	for i := 0; i <= 13; i++ {
		field := sr.GetField(i)
		if field != "" && field != "VEH. NUMBER" && field != "SERVICED" {
			fields = append(fields, field)
		}
	}
	return fields
}

// StudentAttendance represents attendance data for students
type StudentAttendance struct {
	Position   int    `json:"position"`
	Present    bool   `json:"present"`
	PickupTime string `json:"pickup_time"`
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

// FuelRecord is defined in fuel_efficiency.go
// Commented out to avoid redeclaration

// min function is defined in main.go
// Commented out to avoid redeclaration
