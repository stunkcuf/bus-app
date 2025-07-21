@echo off
echo Quick PostgreSQL Setup for Fleet Management
echo ==========================================
echo.
echo This script will help you set up PostgreSQL locally.
echo.
echo STEP 1: Download PostgreSQL
echo ---------------------------
echo Please download PostgreSQL from:
echo https://www.postgresql.org/download/windows/
echo.
echo Choose the latest version (16.x) and install with:
echo - Password for postgres user: localpass123
echo - Port: 5432 (default)
echo - Other settings: defaults are fine
echo.
pause
echo.
echo STEP 2: Create Database
echo ----------------------
echo After installation, I'll create the database...
echo.

REM Try to create database
"C:\Program Files\PostgreSQL\16\bin\psql.exe" -U postgres -c "CREATE DATABASE fleet_management;" 2>nul
if %errorlevel% equ 0 (
    echo Database created successfully!
) else (
    "C:\Program Files\PostgreSQL\15\bin\psql.exe" -U postgres -c "CREATE DATABASE fleet_management;" 2>nul
    if %errorlevel% equ 0 (
        echo Database created successfully!
    ) else (
        echo Could not create database. Please ensure PostgreSQL is installed.
        echo Try running: psql -U postgres -c "CREATE DATABASE fleet_management;"
    )
)

echo.
echo STEP 3: Import Data
echo ------------------
echo Importing Railway backup...
"C:\Program Files\PostgreSQL\16\bin\psql.exe" -U postgres -d fleet_management -f utilities\railway_backup.sql 2>nul
if %errorlevel% neq 0 (
    "C:\Program Files\PostgreSQL\15\bin\psql.exe" -U postgres -d fleet_management -f utilities\railway_backup.sql 2>nul
)

echo.
echo STEP 4: Update Configuration
echo ---------------------------
echo Create a file named .env.local with:
echo DATABASE_URL=postgresql://postgres:localpass123@localhost:5432/fleet_management
echo.
echo Then update your Go application to use the local database.
echo.
pause