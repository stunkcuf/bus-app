@echo off
echo ========================================
echo Starting HS-Bus Fleet Management System
echo ========================================
echo.

REM Kill any existing instances
echo Stopping any running instances...
taskkill /F /IM hs-bus.exe 2>nul
timeout /t 2 /nobreak >nul

REM Clear terminal
cls

echo Building application...
go build -o hs-bus.exe .
if %errorlevel% neq 0 (
    echo Build failed!
    pause
    exit /b 1
)

echo.
echo Starting application...
echo Admin credentials: username=admin, password=Headstart1
echo Access the application at: http://localhost:5000
echo.
echo Press Ctrl+C to stop the server
echo ========================================
echo.

hs-bus.exe