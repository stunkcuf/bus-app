package main

import (
    "database/sql"
    "fmt"
    "log"
    "os"
    "path/filepath"
    "github.com/joho/godotenv"
    _ "github.com/lib/pq"
)

func main() {
    // Load .env file from parent directory
    envPath := filepath.Join("..", ".env")
    if err := godotenv.Load(envPath); err != nil {
        fmt.Println("Could not load .env file")
    }
    
    dbURL := os.Getenv("DATABASE_URL")
    db, err := sql.Open("postgres", dbURL)
    if err != nil {
        log.Fatal("Failed to connect:", err)
    }
    defer db.Close()
    
    fmt.Println("Sample data from fleet_vehicles:")
    rows, err := db.Query("SELECT * FROM fleet_vehicles LIMIT 3")
    if err != nil {
        log.Fatal(err)
    }
    defer rows.Close()
    
    cols, _ := rows.Columns()
    fmt.Println("Columns:", cols)
    
    for rows.Next() {
        var id, vehNum sql.NullInt32
        var sheet, make, model, desc, serial, license, location, tireSize, vehType sql.NullString
        var year sql.NullInt32
        var createdAt, updatedAt sql.NullTime
        
        err := rows.Scan(&id, &vehNum, &sheet, &year, &make, &model, &desc, &serial, &license, &location, &tireSize, &createdAt, &updatedAt, &vehType)
        if err != nil {
            fmt.Println("Scan error:", err)
            continue
        }
        fmt.Printf("Vehicle %d: %s %s (%d) - %s [Type: %s]\n", vehNum.Int32, make.String, model.String, year.Int32, desc.String, vehType.String)
    }
    
    // Check if fleet_vehicles has the data we need
    fmt.Println("\nTotal in fleet_vehicles:")
    var count int
    db.QueryRow("SELECT COUNT(*) FROM fleet_vehicles").Scan(&count)
    fmt.Printf("Fleet vehicles: %d\n", count)
    
    // Check buses + vehicles total
    var busCount, vehCount int
    db.QueryRow("SELECT COUNT(*) FROM buses").Scan(&busCount)
    db.QueryRow("SELECT COUNT(*) FROM vehicles").Scan(&vehCount)
    fmt.Printf("Buses: %d, Vehicles: %d, Total: %d\n", busCount, vehCount, busCount+vehCount)
}