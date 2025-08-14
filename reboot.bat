@echo off
REM Stop any running instances and restart in new CMD window

echo ==========================================
echo   FLEET MANAGEMENT SYSTEM
echo   Stopping any running instances...
echo ==========================================
echo.

REM Kill any fleet.exe processes
taskkill /F /IM fleet.exe 2>nul
if %errorlevel%==0 (
    echo Stopped running Fleet processes
) else (
    echo No Fleet processes were running
)

REM Kill any Go processes
taskkill /F /IM go.exe 2>nul
if %errorlevel%==0 (
    echo Stopped running Go processes
)

REM Kill any process on port 8080
for /f "tokens=5" %%a in ('netstat -aon ^| findstr :8080') do (
    taskkill /F /PID %%a 2>nul
)

echo.
echo Waiting for cleanup...
timeout /t 3 /nobreak >nul

REM Set environment variables
set DATABASE_URL=postgresql://postgres:jTKOWGEzlprRGbkPBCgGxsgnwyLeGoDL@shortline.proxy.rlwy.net:40148/railway
set PORT=8080
set APP_ENV=development

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
)

echo.
echo ==========================================
echo   Starting Fleet Management System
echo   in new window...
echo ==========================================
echo.

REM Open new CMD window and run fleet.exe
start "HS Bus Fleet Management System" cmd /k "cd /d C:\Users\mycha\hs-bus && set DATABASE_URL=postgresql://postgres:jTKOWGEzlprRGbkPBCgGxsgnwyLeGoDL@shortline.proxy.rlwy.net:40148/railway && set PORT=8080 && set APP_ENV=development && echo ========================================== && echo   FLEET MANAGEMENT SYSTEM RESTARTED && echo   Server running on http://localhost:8080 && echo ========================================== && echo. && fleet.exe"

echo Fleet Management System restarted in new window.
echo Access the application at http://localhost:8080
echo.
echo This window will close in 5 seconds...
timeout /t 5 /nobreak >nul