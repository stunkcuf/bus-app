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
            v.vin,
            v.description,
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
