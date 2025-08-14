@echo off
REM Open a new Command Prompt window with detailed output

echo ==========================================
echo   FLEET MANAGEMENT SYSTEM
echo   Preparing to start server...
echo ==========================================
echo.

set DATABASE_URL=postgresql://postgres:jTKOWGEzlprRGbkPBCgGxsgnwyLeGoDL@shortline.proxy.rlwy.net:40148/railway
set PORT=8080
set BACKUP_PATH=./backups

cd /d "C:\Users\mycha\hs-bus"

REM Check if fleet.exe exists, if not build it
if not exist fleet.exe (
    echo Building application...
    go build -o fleet.exe .
    if %ERRORLEVEL% NEQ 0 (
        echo Build failed!
        pause
        exit /b %ERRORLEVEL%
    )
    echo Build successful!
    echo.
)

echo Opening Fleet Management System in new window...
echo.

REM Open new CMD window with full configuration display
start "HS Bus Fleet Management System - Debug Mode" cmd /k "cd /d C:\Users\mycha\hs-bus && set DATABASE_URL=postgresql://postgres:jTKOWGEzlprRGbkPBCgGxsgnwyLeGoDL@shortline.proxy.rlwy.net:40148/railway && set PORT=8080 && set BACKUP_PATH=./backups && echo ========================================== && echo   FLEET MANAGEMENT SYSTEM && echo ========================================== && echo. && echo Configuration: && echo - Port: 8080 && echo - Backup Path: ./backups && echo - Database: Railway PostgreSQL && echo. && echo Server starting... && echo Access at: http://localhost:8080 && echo. && echo Press Ctrl+C to stop the server && echo ========================================== && echo. && fleet.exe"

echo Fleet Management System started in new window.
echo.
echo You can close this window now.
pause