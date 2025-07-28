package models

import (
	"database/sql"
	"time"
)

// User represents a system user
type User struct {
	ID           int            `json:"id" db:"id"`
	Username     string         `json:"username" db:"username"`
	DisplayName  string         `json:"display_name" db:"display_name"`
	Email        string         `json:"email" db:"email"`
	PasswordHash string         `json:"-" db:"password_hash"`
	Role         string         `json:"role" db:"role"`
	Active       bool           `json:"active" db:"active"`
	CreatedAt    time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at" db:"updated_at"`
	Phone        sql.NullString `json:"phone,omitempty" db:"phone"`
	HasCDL       bool           `json:"has_cdl" db:"has_cdl"`
	CDLExpiry    sql.NullTime   `json:"cdl_expiry,omitempty" db:"cdl_expiry"`
	LastLogin    sql.NullTime   `json:"last_login,omitempty" db:"last_login"`
}

// UserStats represents user statistics
type UserStats struct {
	TotalUsers     int `json:"total_users"`
	ActiveUsers    int `json:"active_users"`
	TotalDrivers   int `json:"total_drivers"`
	TotalManagers  int `json:"total_managers"`
	UsersWithCDL   int `json:"users_with_cdl"`
	ExpiredCDL     int `json:"expired_cdl"`
	ExpiringCDL30  int `json:"expiring_cdl_30"`
}

// Session represents a user session
type Session struct {
	SessionToken string    `json:"session_token" db:"session_token"`
	Username     string    `json:"username" db:"username"`
	CSRFToken    string    `json:"csrf_token" db:"csrf_token"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	ExpiresAt    time.Time `json:"expires_at" db:"expires_at"`
	IPAddress    string    `json:"ip_address" db:"ip_address"`
	UserAgent    string    `json:"user_agent" db:"user_agent"`
}

// LoginAttempt tracks login attempts
type LoginAttempt struct {
	ID         int       `db:"id"`
	Username   string    `db:"username"`
	IPAddress  string    `db:"ip_address"`
	Success    bool      `db:"success"`
	AttemptAt  time.Time `db:"attempt_at"`
	UserAgent  string    `db:"user_agent"`
}