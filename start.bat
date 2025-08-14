@echo off
REM Start Fleet Management System in new window

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

REM Start in new window with proper environment
start "HS Bus Fleet Management System" cmd /c "set DATABASE_URL=postgresql://postgres:jTKOWGEzlprRGbkPBCgGxsgnwyLeGoDL@shortline.proxy.rlwy.net:40148/railway&& set PORT=8080&& echo ========================================== && echo   FLEET MANAGEMENT SYSTEM && echo   Server running on http://localhost:8080 && echo ========================================== && echo. && fleet.exe && pause"

echo Fleet Management System started in new window.
echo Access the application at http://localhost:8080
timeout /t 3 /nobreak >nul