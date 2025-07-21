@echo off
echo Fleet Management System - Fleet Data Diagnostics
echo ===============================================
echo.

echo This script will help diagnose why fleet data isn't loading.
echo.

echo 1. Restart the application with the fix:
echo    - Press Ctrl+C to stop the current instance
echo    - Run: go run .
echo.

echo 2. Check the application logs for:
echo    - "Error loading all vehicles from fleet_vehicles"
echo    - "Attempting to load from old bus/vehicle tables"
echo    - "Loaded X vehicles from old tables"
echo.

echo 3. Common issues:
echo    a) fleet_vehicles table doesn't exist
echo       Solution: System will use buses/vehicles tables instead
echo.
echo    b) Database connection issue
echo       Solution: Check DATABASE_URL and network connection
echo.
echo    c) Empty tables
echo       Solution: Verify data exists in buses/vehicles tables
echo.

echo 4. To check your database tables, run these SQL queries:
echo    - SELECT COUNT(*) FROM buses;
echo    - SELECT COUNT(*) FROM vehicles;
echo    - SELECT table_name FROM information_schema.tables WHERE table_name LIKE '%%vehicle%%';
echo.

echo The fix I've implemented will:
echo - Try to load from fleet_vehicles table first
echo - If that fails, automatically fall back to buses + vehicles tables
echo - Convert the data to work with the new display format
echo.

echo Try accessing http://localhost:5000/fleet again after restarting.
echo.
pause