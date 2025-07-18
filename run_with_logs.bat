@echo off
echo ========================================
echo Starting HS-Bus with Debug Logging
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
echo ========================================
echo Application starting...
echo Admin login: username=admin, password=Headstart1
echo Access at: http://localhost:5000
echo.
echo Watch the logs below for debugging:
echo ========================================
echo.

hs-bus.exe 2>&1