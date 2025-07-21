# Setting Up Local PostgreSQL Database

## 1. Install PostgreSQL

1. Download PostgreSQL installer from: https://www.postgresql.org/download/windows/
2. Choose the latest version (16.x)
3. Run the installer with these settings:
   - Installation Directory: Default is fine
   - Data Directory: Default is fine
   - Password: Choose a password for the postgres user (remember this!)
   - Port: 5432 (default)
   - Locale: Default
   - Install Stack Builder: Not needed, uncheck

## 2. Add PostgreSQL to PATH (Optional but recommended)

1. Open System Properties > Environment Variables
2. Edit the System PATH variable
3. Add: `C:\Program Files\PostgreSQL\16\bin` (adjust version number if different)
4. Click OK and restart any command prompts

## 3. Create Local Database

Open Command Prompt or PowerShell and run:

```bash
# Connect to PostgreSQL as postgres user
psql -U postgres

# You'll be prompted for the password you set during installation
# Then create the database:
CREATE DATABASE fleet_management;

# Exit psql
\q
```

## 4. Import the Data

```bash
# Import the exported data
psql -U postgres -d fleet_management -f C:\Users\mycha\hs-bus\utilities\railway_backup.sql
```

## 5. Update Your .env File

Create a `.env.local` file with:

```
DATABASE_URL=postgresql://postgres:YOUR_PASSWORD@localhost:5432/fleet_management
```

## 6. Test the Local Database

You can verify the import worked:

```bash
psql -U postgres -d fleet_management -c "SELECT COUNT(*) FROM buses;"
psql -U postgres -d fleet_management -c "SELECT COUNT(*) FROM maintenance_records;"
```

## Alternative: Use pgAdmin

If you prefer a GUI:
1. pgAdmin is installed with PostgreSQL
2. Open pgAdmin from Start Menu
3. Connect to your local server
4. Right-click on Databases > Create > Database
5. Name it "fleet_management"
6. Use Query Tool to run the SQL file

## Quick Check Script

After setup, you can test with this Go script:

```go
package main

import (
    "fmt"
    "log"
    "github.com/jmoiron/sqlx"
    _ "github.com/lib/pq"
)

func main() {
    // Update with your password
    db, err := sqlx.Connect("postgres", "postgresql://postgres:YOUR_PASSWORD@localhost:5432/fleet_management")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()
    
    var count int
    db.Get(&count, "SELECT COUNT(*) FROM buses")
    fmt.Printf("Buses: %d\n", count)
    
    db.Get(&count, "SELECT COUNT(*) FROM maintenance_records")  
    fmt.Printf("Maintenance records: %d\n", count)
}
```