@echo off
echo ==========================================
echo   FLEET MANAGEMENT SYSTEM
echo   Stopping any running instances...
echo ==========================================
echo.

REM Kill any Go processes
taskkill /F /IM go.exe 2>nul
if %errorlevel%==0 (
    echo Stopped running Go processes
) else (
    echo No Go processes were running
)

REM Kill any process on port 8080
for /f "tokens=5" %%a in ('netstat -aon ^| findstr :8080') do (
    taskkill /F /PID %%a 2>nul
)

echo.
echo Waiting for cleanup...
timeout /t 3 /nobreak >nul

echo.
echo ==========================================
echo   Starting Fleet Management System
echo   Port: 8080
echo ==========================================
echo.

REM Set environment variables
set PORT=8080
set APP_ENV=development

REM Start the application
go run .

pause