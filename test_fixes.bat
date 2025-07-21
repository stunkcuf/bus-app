@echo off
echo Fleet Management System - Testing Data Display Fixes
echo ====================================================
echo.

echo Starting the application...
start /B go run .

echo Waiting for server to start (15 seconds)...
timeout /t 15 /nobreak > nul

echo.
echo Server should be running at http://localhost:5000
echo.
echo Please test the following:
echo.
echo 1. LOGIN and go to Manager Dashboard
echo    - Check if "Recent Activity" shows real data (not mock)
echo    - Verify driver counts are correct
echo.
echo 2. FLEET PAGE (/fleet)
echo    - Should show 91 vehicles total
echo    - Click maintenance links to verify logs display
echo.
echo 3. ECSE DASHBOARD (/ecse-dashboard)
echo    - Should be accessible from manager dashboard
echo    - Check "Upcoming Assessments" count
echo.
echo 4. ROUTE ASSIGNMENTS (/assign-routes)
echo    - Should show student counts per route
echo.
echo 5. IMPORT SYSTEM
echo    - Try importing a test Excel file
echo    - Should analyze and show real data
echo.
echo Press any key to open the application in your browser...
pause > nul

start http://localhost:5000

echo.
echo Testing checklist saved to: TESTING_CHECKLIST.md
echo.
pause