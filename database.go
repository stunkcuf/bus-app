// database.go - Fleet Maintenance Database Operations
package main

import (
    "database/sql"
    "fmt"
    "log"
    "os"
    
    "github.com/jmoiron/sqlx"
    _ "github.com/lib/pq"
)

var db *sqlx.DB

// Initialize database connection to your Railway PostgreSQL
func initDatabase() error {
    // Use environment variables for Railway connection
    dbURL := os.Getenv("DATABASE_URL")
    if dbURL == "" {
        // Fallback to individual environment variables
        host := os.Getenv("PGHOST")
        port := os.Getenv("PGPORT")
        user := os.Getenv("PGUSER")
        password := os.Getenv("PGPASSWORD")
        dbname := os.Getenv("PGDATABASE")
        
        if host == "" || port == "" || user == "" || password == "" || dbname == "" {
            return fmt.Errorf("database connection parameters not set")
        }
        
        dbURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=require",
            user, password, host, port, dbname)
    }
    
    var err error
    db, err = sqlx.Connect("postgres", dbURL)
    if err != nil {
        return fmt.Errorf("failed to connect to database: %v", err)
    }
    
    // Test the connection
    if err = db.Ping(); err != nil {
        return fmt.Errorf("failed to ping database: %v", err)
    }
    
    log.Println("Successfully connected to Railway PostgreSQL database")
    return nil
}

// Get vehicle list for the fleet overview page
func getVehicleList() ([]VehicleWithStats, error) {
    query := `
        SELECT 
            v.vehicle_number,
            v.make,
            v.model,
            v.year,
            COALESCE(v.vin, '') as vin,
            COALESCE(v.description, '') as description,
            COUNT(m.id) as maintenance_count,
            COALESCE(SUM(m.cost), 0) as total_cost,
            MAX(m.maintenance_date) as last_service
        FROM fleet_vehicles v
        LEFT JOIN maintenance_records m ON v.vehicle_number = m.vehicle_number
        GROUP BY v.vehicle_number, v.make, v.model, v.year, v.vin, v.description
        ORDER BY v.vehicle_number ASC`
    
    var vehicles []VehicleWithStats
    err := db.Select(&vehicles, query)
    if err != nil {
        return nil, fmt.Errorf("failed to fetch vehicles: %v", err)
    }
    
    return vehicles, nil
}

// Get a specific vehicle by vehicle number
func getVehicle(vehicleNumber int) (*Vehicle, error) {
    var vehicle Vehicle
    query := `
        SELECT vehicle_number, make, model, year, vin, description 
        FROM fleet_vehicles 
        WHERE vehicle_number = $1`
    
    err := db.Get(&vehicle, query, vehicleNumber)
    if err != nil {
        if err == sql.ErrNoRows {
            return nil, fmt.Errorf("vehicle not found")
        }
        return nil, fmt.Errorf("failed to fetch vehicle: %v", err)
    }
    
    return &vehicle, nil
}

// Get maintenance records for a specific vehicle
func getVehicleMaintenanceRecords(vehicleNumber int) ([]MaintenanceLog, error) {
    query := `
        SELECT 
            id,
            vehicle_number,
            maintenance_date as service_date,
            mileage,
            po_number,
            cost,
            work_done,
            created_at
        FROM maintenance_records 
        WHERE vehicle_number = $1 
        ORDER BY maintenance_date DESC`
    
    var records []MaintenanceLog
    err := db.Select(&records, query, vehicleNumber)
    if err != nil {
        return nil, fmt.Errorf("failed to fetch maintenance records: %v", err)
    }
    
    return records, nil
}

// Add a new maintenance record
func addMaintenanceRecord(record MaintenanceLog) error {
    query := `
        INSERT INTO maintenance_records 
        (vehicle_number, maintenance_date, mileage, po_number, cost, work_done)
        VALUES ($1, $2, $3, $4, $5, $6)`
    
    _, err := db.Exec(query, 
        record.VehicleNumber,
        record.ServiceDate,
        record.Mileage,
        record.PONumber,
        record.Cost,
        record.WorkDone)
    
    if err != nil {
        return fmt.Errorf("failed to add maintenance record: %v", err)
    }
    
    return nil
}

// Get fleet statistics
func getFleetStats() (*FleetStats, error) {
    var stats FleetStats
    
    // Get total vehicles
    err := db.Get(&stats.TotalVehicles, "SELECT COUNT(*) FROM fleet_vehicles")
    if err != nil {
        return nil, fmt.Errorf("failed to get vehicle count: %v", err)
    }
    
    // Get total maintenance records
    err = db.Get(&stats.TotalMaintenanceRecords, "SELECT COUNT(*) FROM maintenance_records")
    if err != nil {
        return nil, fmt.Errorf("failed to get maintenance count: %v", err)
    }
    
    // Get total maintenance cost
    err = db.Get(&stats.TotalMaintenanceCost, "SELECT COALESCE(SUM(cost), 0) FROM maintenance_records WHERE cost IS NOT NULL")
    if err != nil {
        return nil, fmt.Errorf("failed to get total cost: %v", err)
    }
    
    // Calculate average cost
    if stats.TotalMaintenanceRecords > 0 {
        stats.AverageMaintenanceCost = stats.TotalMaintenanceCost / float64(stats.TotalMaintenanceRecords)
    }
    
    // Get vehicles by year range
    yearQuery := `
        SELECT 
            MIN(year) as min_year,
            MAX(year) as max_year
        FROM fleet_vehicles`
    
    var minYear, maxYear sql.NullInt32
    err = db.QueryRow(yearQuery).Scan(&minYear, &maxYear)
    if err != nil {
        log.Printf("Failed to get year range: %v", err)
    } else if minYear.Valid && maxYear.Valid {
        stats.YearRange = fmt.Sprintf("%d-%d", minYear.Int32, maxYear.Int32)
    }
    
    // Get unique makes
    err = db.Get(&stats.UniqueMakes, "SELECT COUNT(DISTINCT make) FROM fleet_vehicles")
    if err != nil {
        return nil, fmt.Errorf("failed to get unique makes: %v", err)
    }
    
    return &stats, nil
}

// Get recent maintenance activity
func getRecentMaintenanceActivity(limit int) ([]MaintenanceLog, error) {
    query := `
        SELECT 
            m.id,
            m.vehicle_number,
            m.maintenance_date as service_date,
            m.mileage,
            m.po_number,
            m.cost,
            m.work_done,
            m.created_at
        FROM maintenance_records m
        ORDER BY m.maintenance_date DESC, m.created_at DESC
        LIMIT $1`
    
    var records []MaintenanceLog
    err := db.Select(&records, query, limit)
    if err != nil {
        return nil, fmt.Errorf("failed to fetch recent maintenance: %v", err)
    }
    
    return records, nil
}

// Search vehicles by make, model, or vehicle number
func searchVehicles(searchTerm string) ([]VehicleWithStats, error) {
    searchPattern := "%" + searchTerm + "%"
    query := `
        SELECT 
            v.vehicle_number,
            v.make,
            v.model,
            v.year,
            COALESCE(v.vin, '') as vin,
            COALESCE(v.description, '') as description,
            COUNT(m.id) as maintenance_count,
            COALESCE(SUM(m.cost), 0) as total_cost,
            MAX(m.maintenance_date) as last_service
        FROM fleet_vehicles v
        LEFT JOIN maintenance_records m ON v.vehicle_number = m.vehicle_number
        WHERE 
            CAST(v.vehicle_number AS TEXT) ILIKE $1 OR
            v.make ILIKE $1 OR 
            v.model ILIKE $1 OR
            v.description ILIKE $1
        GROUP BY v.vehicle_number, v.make, v.model, v.year, v.vin, v.description
        ORDER BY v.vehicle_number ASC`
    
    var vehicles []VehicleWithStats
    err := db.Select(&vehicles, query, searchPattern)
    if err != nil {
        return nil, fmt.Errorf("failed to search vehicles: %v", err)
    }
    
    return vehicles, nil
}

// Check database health
func checkDatabaseHealth() error {
    if db == nil {
        return fmt.Errorf("database connection is nil")
    }
    
    if err := db.Ping(); err != nil {
        return fmt.Errorf("database ping failed: %v", err)
    }
    
    // Verify tables exist
    var vehicleCount int
    err := db.Get(&vehicleCount, "SELECT COUNT(*) FROM fleet_vehicles")
    if err != nil {
        return fmt.Errorf("cannot access fleet_vehicles table: %v", err)
    }
    
    var maintenanceCount int
    err = db.Get(&maintenanceCount, "SELECT COUNT(*) FROM maintenance_records")
    if err != nil {
        return fmt.Errorf("cannot access maintenance_records table: %v", err)
    }
    
    log.Printf("Database health check passed: %d vehicles, %d maintenance records", 
        vehicleCount, maintenanceCount)
    
    return nil
}
