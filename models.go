package main

import (
	"database/sql"
	"encoding/json"
)

// Move all your structs here
type User struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

type Attendance struct {
	Date    string `json:"date"`
	Driver  string `json:"driver"`
	Route   string `json:"route"`
	Present int    `json:"present"`
}

type Mileage struct {
	Date   string  `json:"date"`
	Driver string  `json:"driver"`
	Route  string  `json:"route"`
	Miles  float64 `json:"miles"`
}

type Activity struct {
	Date       string  `json:"date"`
	Driver     string  `json:"driver"`
	TripName   string  `json:"trip_name"`
	Attendance int     `json:"attendance"`
	Miles      float64 `json:"miles"`
	Notes      string  `json:"notes"`
}

type DriverSummary struct {
	Name              string
	TotalMorning      int
	TotalEvening      int
	TotalMiles        float64
	MonthlyAvgMiles   float64
	MonthlyAttendance int
}

type RouteStats struct {
	RouteName       string
	TotalMiles      float64
	AvgMiles        float64
	AttendanceDay   int
	AttendanceWeek  int
	AttendanceMonth int
}

type Route struct {
	RouteID     string `json:"route_id"`
	RouteName   string `json:"route_name"`
	Description string `json:"description"`
	Positions []struct {
		Position int    `json:"position"`
		Student  string `json:"student"`
	} `json:"positions"`
}

type Bus struct {
	BusID            string `json:"bus_id"`
	Status           string `json:"status"`
	Model            string `json:"model"`
	Capacity         int    `json:"capacity"`
	OilStatus        string `json:"oil_status"`
	TireStatus       string `json:"tire_status"`
	MaintenanceNotes string `json:"maintenance_notes"`
}

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

type Location struct {
	Type        string `json:"type"`
	Address     string `json:"address"`
	Description string `json:"description"`
}

type RouteAssignment struct {
	Driver       string `json:"driver"`
	BusID        string `json:"bus_id"`
	RouteID      string `json:"route_id"`
	RouteName    string `json:"route_name"`
	AssignedDate string `json:"assigned_date"`
}

type DriverLog struct {
	Driver     string `json:"driver"`
	BusID      string `json:"bus_id"`
	RouteID    string `json:"route_id"`
	Date       string `json:"date"`
	Period     string `json:"period"`
	Departure  string `json:"departure_time"`
	Arrival    string `json:"arrival_time"`
	Mileage    float64 `json:"mileage"`
	Attendance []struct {
		Position   int    `json:"position"`
		Present    bool   `json:"present"`
		PickupTime string `json:"pickup_time,omitempty"`
	} `json:"attendance"`
}

type MaintenanceLog struct {
	BusID    string `json:"bus_id"`
	Date     string `json:"date"`
	Category string `json:"category"`
	Notes    string `json:"notes"`
	Mileage  int    `json:"mileage"`
}

// Template data structures
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

type AssignRouteData struct {
	User            *User
	Assignments     []RouteAssignment
	Drivers         []User
	AvailableRoutes []Route
	AvailableBuses  []*Bus
}

type FleetData struct {
	User  *User
	Buses []*Bus
	Today string
}

type StudentData struct {
	User     *User
	Students []Student
	Routes   []Route
}

type CompanyFleetData struct {
	User     *User
	Vehicles []Vehicle
}
