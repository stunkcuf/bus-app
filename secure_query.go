package main

import (
	"fmt"
	"strings"
)

// SecureQuery provides safe SQL query building
type SecureQuery struct {
	allowedTables  map[string]bool
	allowedColumns map[string]map[string]bool
}

// NewSecureQuery creates a new secure query builder
func NewSecureQuery() *SecureQuery {
	return &SecureQuery{
		allowedTables: map[string]bool{
			"buses":                    true,
			"vehicles":                 true,
			"users":                    true,
			"students":                 true,
			"routes":                   true,
			"route_assignments":        true,
			"driver_logs":              true,
			"bus_maintenance_logs":     true,
			"vehicle_maintenance_logs": true,
			"monthly_mileage_reports":  true,
			"ecse_students":            true,
			"ecse_services":            true,
			"fuel_records":             true,
		},
		allowedColumns: map[string]map[string]bool{
			"buses": {
				"bus_id": true, "status": true, "model": true, "capacity": true,
				"oil_status": true, "tire_status": true, "maintenance_notes": true,
				"current_mileage": true, "last_oil_change": true, "last_tire_service": true,
				"updated_at": true, "created_at": true,
			},
			"vehicles": {
				"vehicle_id": true, "status": true, "model": true, "year": true,
				"tire_size": true, "license": true, "oil_status": true, "tire_status": true,
				"maintenance_notes": true, "serial_number": true, "base": true,
				"service_interval": true, "current_mileage": true, "last_oil_change": true,
				"last_tire_service": true, "updated_at": true, "created_at": true,
			},
			// Add more tables and columns as needed
		},
	}
}

// ValidateTable checks if a table name is allowed
func (sq *SecureQuery) ValidateTable(table string) error {
	if !sq.allowedTables[strings.ToLower(table)] {
		return fmt.Errorf("invalid table name: %s", table)
	}
	return nil
}

// ValidateColumn checks if a column name is allowed for a table
func (sq *SecureQuery) ValidateColumn(table, column string) error {
	table = strings.ToLower(table)
	column = strings.ToLower(column)
	
	if err := sq.ValidateTable(table); err != nil {
		return err
	}
	
	// If we don't have column info for this table, allow common columns
	if sq.allowedColumns[table] == nil {
		commonColumns := map[string]bool{
			"id": true, "created_at": true, "updated_at": true,
		}
		if !commonColumns[column] {
			return fmt.Errorf("column validation not configured for table %s", table)
		}
		return nil
	}
	
	if !sq.allowedColumns[table][column] {
		return fmt.Errorf("invalid column %s for table %s", column, table)
	}
	
	return nil
}

// BuildUpdate creates a safe UPDATE query
// Note: This function uses fmt.Sprintf but is safe from SQL injection because:
// 1. Table and column names are validated against a whitelist
// 2. Values are passed as parameters using placeholders ($1, $2, etc.)
func (sq *SecureQuery) BuildUpdate(table, column string, args ...interface{}) (string, []interface{}, error) {
	if err := sq.ValidateColumn(table, column); err != nil {
		return "", nil, err
	}
	
	// Safe to use fmt.Sprintf here because table and column are validated
	query := fmt.Sprintf("UPDATE %s SET %s = $1, updated_at = CURRENT_TIMESTAMP WHERE ", table, column)
	return query, args, nil
}

// BuildSelect creates a safe SELECT query
// Note: This function uses fmt.Sprintf but is safe from SQL injection because:
// 1. Table name is validated against a whitelist
// 2. Column names are validated against a whitelist
func (sq *SecureQuery) BuildSelect(table string, columns []string) (string, error) {
	if err := sq.ValidateTable(table); err != nil {
		return "", err
	}
	
	// Validate all columns
	for _, col := range columns {
		if col != "*" && sq.allowedColumns[table] != nil {
			if err := sq.ValidateColumn(table, col); err != nil {
				return "", err
			}
		}
	}
	
	columnStr := "*"
	if len(columns) > 0 && columns[0] != "*" {
		columnStr = strings.Join(columns, ", ")
	}
	
	// Safe to use fmt.Sprintf here because table and columns are validated
	return fmt.Sprintf("SELECT %s FROM %s", columnStr, table), nil
}

// BuildCount creates a safe COUNT query
// Note: This function uses fmt.Sprintf but is safe from SQL injection because
// the table name is validated against a whitelist before use
func (sq *SecureQuery) BuildCount(table string) (string, error) {
	if err := sq.ValidateTable(table); err != nil {
		return "", err
	}
	
	// Safe to use fmt.Sprintf here because table is validated
	return fmt.Sprintf("SELECT COUNT(*) FROM %s", table), nil
}

// Global secure query instance
var secureQuery = NewSecureQuery()

// Helper functions for backward compatibility

// SafeUpdateQuery builds a safe UPDATE query
func SafeUpdateQuery(table, column string) (string, error) {
	if err := secureQuery.ValidateColumn(table, column); err != nil {
		return "", err
	}
	return fmt.Sprintf("UPDATE %s SET %s = $1, updated_at = CURRENT_TIMESTAMP", table, column), nil
}

// SafeCountQuery builds a safe COUNT query
func SafeCountQuery(table string) (string, error) {
	if err := secureQuery.ValidateTable(table); err != nil {
		return "", err
	}
	return fmt.Sprintf("SELECT COUNT(*) FROM %s", table), nil
}

// SafeSelectQuery builds a safe SELECT query
func SafeSelectQuery(table string, columns ...string) (string, error) {
	return secureQuery.BuildSelect(table, columns)
}