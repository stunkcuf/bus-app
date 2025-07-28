package models

import (
	"database/sql"
	"time"
)

// Vehicle represents a general vehicle
type Vehicle struct {
	VehicleID        string         `json:"vehicle_id" db:"vehicle_id"`
	VehicleType      string         `json:"vehicle_type" db:"vehicle_type"`
	VehicleNumber    sql.NullInt32  `json:"vehicle_number" db:"vehicle_number"`
	Status           string         `json:"status" db:"status"`
	Model            sql.NullString `json:"model" db:"model"`
	Year             sql.NullString `json:"year" db:"year"`
	Make             sql.NullString `json:"make" db:"make"`
	TireSize         sql.NullString `json:"tire_size" db:"tire_size"`
	License          sql.NullString `json:"license" db:"license"`
	OilStatus        sql.NullString `json:"oil_status" db:"oil_status"`
	TireStatus       sql.NullString `json:"tire_status" db:"tire_status"`
	MaintenanceNotes sql.NullString `json:"maintenance_notes" db:"maintenance_notes"`
	Description      sql.NullString `json:"description" db:"description"`
	SerialNumber     sql.NullString `json:"serial_number" db:"serial_number"`
	Base             sql.NullString `json:"base" db:"base"`
	ServiceInterval  sql.NullInt32  `json:"service_interval" db:"service_interval"`
	CurrentMileage   sql.NullString `json:"current_mileage" db:"current_mileage"`
	LastOilChange    sql.NullTime   `json:"last_oil_change" db:"last_oil_change"`
	LastTireService  sql.NullTime   `json:"last_tire_service" db:"last_tire_service"`
	UpdatedAt        time.Time      `json:"updated_at" db:"updated_at"`
	CreatedAt        time.Time      `json:"created_at" db:"created_at"`
}

// FleetVehicle represents a fleet vehicle (legacy structure for compatibility)
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

// ConsolidatedVehicle represents a unified view of vehicles and buses
type ConsolidatedVehicle struct {
	ID               string         `json:"id" db:"id"`
	VehicleID        string         `json:"vehicle_id" db:"vehicle_id"`
	BusID            string         `json:"bus_id" db:"bus_id"`
	VehicleType      string         `json:"vehicle_type" db:"vehicle_type"`
	Status           string         `json:"status" db:"status"`
	Model            sql.NullString `json:"model" db:"model"`
	Capacity         sql.NullInt32  `json:"capacity" db:"capacity"`
	OilStatus        sql.NullString `json:"oil_status" db:"oil_status"`
	TireStatus       sql.NullString `json:"tire_status" db:"tire_status"`
	MaintenanceNotes sql.NullString `json:"maintenance_notes" db:"maintenance_notes"`
	Year             sql.NullString `json:"year" db:"year"`
	TireSize         sql.NullString `json:"tire_size" db:"tire_size"`
	License          sql.NullString `json:"license" db:"license"`
	Description      sql.NullString `json:"description" db:"description"`
	SerialNumber     sql.NullString `json:"serial_number" db:"serial_number"`
	Base             sql.NullString `json:"base" db:"base"`
	ServiceInterval  sql.NullInt32  `json:"service_interval" db:"service_interval"`
	CurrentMileage   sql.NullString `json:"current_mileage" db:"current_mileage"`
	LastOilChange    sql.NullTime   `json:"last_oil_change" db:"last_oil_change"`
	LastTireService  sql.NullTime   `json:"last_tire_service" db:"last_tire_service"`
	UpdatedAt        time.Time      `json:"updated_at" db:"updated_at"`
	CreatedAt        time.Time      `json:"created_at" db:"created_at"`
	Assignment       string         `json:"assignment"`
}

// VehicleMaintenanceLog represents a vehicle maintenance log entry
type VehicleMaintenanceLog struct {
	ID               int            `json:"id" db:"id"`
	VehicleID        string         `json:"vehicle_id" db:"vehicle_id"`
	MaintenanceDate  time.Time      `json:"maintenance_date" db:"maintenance_date"`
	MaintenanceType  string         `json:"maintenance_type" db:"maintenance_type"`
	Description      string         `json:"description" db:"description"`
	Cost             float64        `json:"cost" db:"cost"`
	PerformedBy      string         `json:"performed_by" db:"performed_by"`
	Mileage          sql.NullInt32  `json:"mileage" db:"mileage"`
	NextServiceDue   sql.NullTime   `json:"next_service_due" db:"next_service_due"`
	Notes            sql.NullString `json:"notes" db:"notes"`
	CreatedAt        time.Time      `json:"created_at" db:"created_at"`
}