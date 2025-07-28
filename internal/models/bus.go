package models

import (
	"database/sql"
	"time"
)

// Bus represents a school bus
type Bus struct {
	BusID            string         `json:"bus_id" db:"bus_id"`
	Status           string         `json:"status" db:"status"`
	Model            sql.NullString `json:"model" db:"model"`
	Capacity         int            `json:"capacity" db:"capacity"`
	OilStatus        sql.NullString `json:"oil_status" db:"oil_status"`
	TireStatus       sql.NullString `json:"tire_status" db:"tire_status"`
	MaintenanceNotes sql.NullString `json:"maintenance_notes" db:"maintenance_notes"`
	CurrentMileage   int            `json:"current_mileage" db:"current_mileage"`
	LastOilChange    int            `json:"last_oil_change" db:"last_oil_change"`
	LastTireRotation int            `json:"last_tire_rotation" db:"last_tire_rotation"`
	UpdatedAt        time.Time      `json:"updated_at" db:"updated_at"`
	CreatedAt        time.Time      `json:"created_at" db:"created_at"`
}

// BusAssignment represents a bus assignment
type BusAssignment struct {
	BusID      string    `json:"bus_id" db:"bus_id"`
	DriverName string    `json:"driver_name" db:"driver_name"`
	RouteName  string    `json:"route_name" db:"route_name"`
	RouteID    string    `json:"route_id" db:"route_id"`
	Period     string    `json:"period" db:"period"`
	StartTime  string    `json:"start_time" db:"start_time"`
	EndTime    string    `json:"end_time" db:"end_time"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

// BusMaintenanceLog represents a maintenance log entry
type BusMaintenanceLog struct {
	ID               int            `json:"id" db:"id"`
	BusID            string         `json:"bus_id" db:"bus_id"`
	MaintenanceDate  time.Time      `json:"maintenance_date" db:"maintenance_date"`
	MaintenanceType  string         `json:"maintenance_type" db:"maintenance_type"`
	Description      string         `json:"description" db:"description"`
	Cost             float64        `json:"cost" db:"cost"`
	PerformedBy      string         `json:"performed_by" db:"performed_by"`
	Mileage          int            `json:"mileage" db:"mileage"`
	NextServiceDue   sql.NullTime   `json:"next_service_due" db:"next_service_due"`
	Notes            sql.NullString `json:"notes" db:"notes"`
	CreatedAt        time.Time      `json:"created_at" db:"created_at"`
}