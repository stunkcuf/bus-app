@echo off
echo ========================================
echo Fleet Management Server Restart Script
echo ========================================
echo.

echo Checking if server is running on port 5003...
netstat -an | findstr :5003 >nul
if %errorlevel% == 0 (
    echo Server is running on port 5003
    echo Please stop it manually with Ctrl+C in the server window
    echo Then press any key to continue...
    pause >nul
) else (
    echo No server detected on port 5003
)

echo.
echo Starting server with latest fixes...
echo ========================================
go run .

if %errorlevel% neq 0 (
    echo.
    echo ERROR: Failed to start server!
    echo Check the error messages above.
    pause
    exit /b 1
)